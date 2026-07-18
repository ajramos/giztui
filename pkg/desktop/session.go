package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/llm"
	"github.com/ajramos/giztui/internal/render"
	"github.com/ajramos/giztui/internal/services"
	"github.com/ajramos/giztui/pkg/auth"
)

// gmailScopes mirrors the OAuth scopes requested by the TUI so the desktop
// client can reuse the same token without re-consent.
var gmailScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/gmail.compose",
	"https://www.googleapis.com/auth/gmail.settings.basic",
}

// Options configures how a Session resolves its config and credentials. All
// fields are optional; empty values fall back to the same environment
// variables and default paths the TUI uses (GMAIL_TUI_CONFIG, etc.).
type Options struct {
	ConfigPath      string
	CredentialsPath string
	TokenPath       string
	Logger          *log.Logger
}

// Session owns the constructed service stack and exposes the front-end API.
type Session struct {
	API    *API
	Config *config.Config

	client           *gmail.Client
	dbManager        services.DatabaseManager
	accountService   services.AccountService
	currentAccountID string
	logger           *log.Logger
}

// Close releases the session's resources (e.g. the local database).
func (s *Session) Close() error {
	if s.dbManager != nil {
		return s.dbManager.Close()
	}
	return nil
}

// NewSession builds the full Gmail/LLM/service stack from config and OAuth
// credentials on disk, then returns a ready-to-use API. It reuses the exact
// same construction path as the TUI so behavior stays consistent.
func NewSession(ctx context.Context, opts Options) (*Session, error) {
	logger := opts.Logger

	configPath := resolvePath(opts.ConfigPath, "GMAIL_TUI_CONFIG", config.DefaultConfigPath())

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config %q: %w", configPath, err)
	}

	defaultCred, defaultToken := config.DefaultCredentialPaths()
	credPath := resolvePath(firstNonEmpty(opts.CredentialsPath, expandPath(cfg.Credentials)), "GMAIL_TUI_CREDENTIALS", defaultCred)
	tokenPath := resolvePath(firstNonEmpty(opts.TokenPath, expandPath(cfg.Token)), "GMAIL_TUI_TOKEN", defaultToken)

	service, err := auth.NewGmailService(ctx, credPath, tokenPath, gmailScopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gmail service (creds: %s, token: %s): %w", credPath, tokenPath, err)
	}
	client := gmail.NewClient(service)

	dbManager := services.NewDatabaseManager(cfg, logger)
	accountService := services.NewAccountService(cfg, logger)

	api := buildAPI(ctx, cfg, client, dbManager, logger)

	sess := &Session{
		API:            api,
		Config:         cfg,
		client:         client,
		dbManager:      dbManager,
		accountService: accountService,
		logger:         logger,
	}
	if active, err := accountService.GetActiveAccount(ctx); err == nil && active != nil {
		sess.currentAccountID = active.ID
	}
	return sess, nil
}

// buildAPI constructs the full service stack for a given Gmail client and wraps
// it in an API. It opens the per-account local database (best-effort; powers
// summary caching and prompts) and is reused verbatim when switching accounts.
func buildAPI(ctx context.Context, cfg *config.Config, client *gmail.Client, dbManager services.DatabaseManager, logger *log.Logger) *API {
	repo := services.NewMessageRepository(client)
	labelService := services.NewLabelService(client)
	renderer := render.NewEmailRenderer(cfg)
	emailService := services.NewEmailService(repo, client, renderer)
	attachmentService := services.NewAttachmentService(client, cfg)
	linkService := services.NewLinkService(client, renderer)
	webService := services.NewGmailWebService(linkService)
	compositionService := services.NewCompositionService(emailService, client, repo)

	// Capture the active account address so composed messages have a "from".
	accountEmail, _ := client.ActiveAccountEmail(ctx)

	var dbStore *db.Store
	if accountEmail != "" {
		if err := dbManager.SwitchToAccountDatabase(ctx, accountEmail); err != nil {
			if logger != nil {
				logger.Printf("desktop: could not open local database: %v", err)
			}
		} else {
			dbStore = dbManager.GetCurrentStore()
		}
	}

	var cacheService services.CacheService
	if dbStore != nil {
		cacheService = services.NewCacheService(db.NewCacheStore(dbStore))
	}

	// AIService is optional: only wired when an LLM provider is configured.
	aiService := buildAIService(cfg, cacheService, logger)

	// PromptService needs both an LLM (aiService) and the local database.
	var promptService services.PromptService
	if aiService != nil && dbStore != nil {
		promptService = services.NewPromptService(db.NewPromptStore(dbStore), aiService, nil)
	}

	// Obsidian needs the local database and an enabled config.
	var obsidianService services.ObsidianService
	if dbStore != nil && cfg.Obsidian != nil && cfg.Obsidian.Enabled {
		obsidianService = services.NewObsidianService(db.NewObsidianStore(dbStore), cfg.Obsidian, logger)
	}

	// Slack forwarding (optional; gated on config).
	var slackService services.SlackService
	if cfg.Slack.Enabled {
		slackService = services.NewSlackService(client, cfg, aiService)
	}

	// Threading needs the local database.
	var threadService services.ThreadService
	if dbStore != nil {
		threadService = services.NewThreadService(client, dbStore, aiService)
	}

	// Saved queries need the local database.
	var queryService services.QueryService
	if dbStore != nil {
		queryService = services.NewQueryService(db.NewQueryStore(dbStore), cfg)
	}

	// Inbox action plan needs an LLM.
	var analyzerService services.InboxAnalyzerService
	if aiService != nil {
		analyzerService = services.NewInboxAnalyzerService(aiService)
	}

	// Analyzer preference rules need the local database.
	var rulesService services.AnalyzerRulesService
	if dbStore != nil {
		rulesService = services.NewAnalyzerRulesService(db.NewAnalyzerRulesStore(dbStore))
	}

	// Theming: read the user's theme from config (best-effort).
	themeService := buildThemeService(cfg)

	return NewAPI(Deps{
		Repo:         repo,
		Email:        emailService,
		Labels:       labelService,
		Mail:         client,
		AI:           aiService,
		Attach:       attachmentService,
		Prompts:      promptService,
		Web:          webService,
		Composition:  compositionService,
		Draft:        client,
		Link:         linkService,
		Obsidian:     obsidianService,
		Slack:        slackService,
		Thread:       threadService,
		Query:        queryService,
		Analyzer:     analyzerService,
		Rules:        rulesService,
		Theme:        themeService,
		AccountEmail: accountEmail,
		Logger:       logger,
	})
}

// AccountEmail returns the active account's email address.
func (s *Session) AccountEmail(ctx context.Context) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("gmail client not initialized")
	}
	return s.client.ActiveAccountEmail(ctx)
}

// ListAccounts returns all configured accounts for the account switcher.
func (s *Session) ListAccounts(ctx context.Context) ([]AccountInfo, error) {
	if s.accountService == nil {
		return []AccountInfo{}, nil
	}
	accounts, err := s.accountService.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AccountInfo, 0, len(accounts))
	for _, a := range accounts {
		if a == nil {
			continue
		}
		out = append(out, AccountInfo{
			ID:          a.ID,
			Email:       a.Email,
			DisplayName: a.DisplayName,
			Active:      a.ID == s.currentAccountID,
		})
	}
	return out, nil
}

// SwitchAccount rebuilds the service stack for a different account and makes it
// current. Subsequent API calls operate on the newly-selected account.
func (s *Session) SwitchAccount(ctx context.Context, accountID string) error {
	if s.accountService == nil {
		return fmt.Errorf("account service not available")
	}
	client, err := s.accountService.GetAccountClient(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get client for account %q: %w", accountID, err)
	}
	if err := s.accountService.SwitchAccount(ctx, accountID); err != nil {
		return err
	}
	s.client = client
	s.API = buildAPI(ctx, s.Config, client, s.dbManager, s.logger)
	s.currentAccountID = accountID
	return nil
}

// buildAIService constructs an AIService from config, mirroring the TUI's LLM
// wiring. Returns nil (not an error) when AI is disabled or misconfigured, so
// the rest of the app still works without AI.
func buildAIService(cfg *config.Config, cacheService services.CacheService, logger *log.Logger) services.AIService {
	if !cfg.LLM.Enabled || cfg.LLM.Model == "" {
		return nil
	}
	providerName := cfg.LLM.Provider
	if providerName == "" {
		providerName = "ollama"
	}
	arg := cfg.LLM.Endpoint
	if providerName == "bedrock" {
		region := cfg.LLM.Region
		if region == "" {
			region = os.Getenv("AWS_REGION")
		}
		arg = region
	}
	provider, err := llm.NewProviderFromConfig(providerName, arg, cfg.LLM.Model, cfg.GetLLMTimeout(), cfg.LLM.APIKey)
	if err != nil {
		if logger != nil {
			logger.Printf("desktop: LLM provider (%s) init failed: %v", providerName, err)
		}
		return nil
	}
	return services.NewAIService(provider, cacheService, cfg)
}

// resolvePath applies the standard priority: explicit value, then environment
// variable, then default.
func resolvePath(explicit, envVar, fallback string) string {
	if explicit != "" {
		return expandPath(explicit)
	}
	if env := os.Getenv(envVar); env != "" {
		return expandPath(env)
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// expandPath expands a leading ~ to the user's home directory.
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// buildThemeService resolves the built-in + custom theme directories the same
// way the TUI does and returns a ThemeService (apply is a no-op; the desktop
// applies colors in the frontend).
func buildThemeService(cfg *config.Config) services.ThemeService {
	customThemeDir := expandPath(cfg.Theme.CustomDir)
	builtin := "themes"
	if _, err := os.Stat(builtin); os.IsNotExist(err) {
		builtin = "../themes"
		if _, err := os.Stat(builtin); os.IsNotExist(err) {
			if exe, err := os.Executable(); err == nil {
				exeDir := filepath.Dir(exe)
				builtin = filepath.Join(exeDir, "..", "themes")
				if _, err := os.Stat(builtin); os.IsNotExist(err) {
					builtin = filepath.Join(exeDir, "themes")
				}
			}
		}
	}
	return services.NewThemeService(builtin, customThemeDir, func(*config.ColorsConfig) error { return nil })
}
