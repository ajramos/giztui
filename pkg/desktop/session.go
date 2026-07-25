package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/calendar"
	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/llm"
	"github.com/ajramos/giztui/internal/render"
	"github.com/ajramos/giztui/internal/services"
	"github.com/ajramos/giztui/pkg/auth"
)

// calendarScope is the extra OAuth scope needed to respond to invites. It is
// requested by the TUI too, so a token created there already carries it.
const calendarScope = "https://www.googleapis.com/auth/calendar.events"

// SetAuthURLHook registers a callback invoked with the OAuth consent URL when
// interactive sign-in begins, so a GUI can open the browser and show a modal
// instead of the user copy/pasting the URL. Passing nil restores the default
// (print-only) behaviour. Thin wrapper so the nested desktop module doesn't need
// to import pkg/auth directly.
func SetAuthURLHook(fn func(url string)) { auth.AuthURLHook = fn }

// calAdapter wraps internal/calendar.Client to satisfy the calClient interface
// without leaking the calendar package's types into the public API.
type calAdapter struct{ c *calendar.Client }

func (a calAdapter) FindEventID(ctx context.Context, iCalUID string) (string, error) {
	evt, err := a.c.FindByICalUID(ctx, iCalUID)
	if err != nil {
		return "", err
	}
	return evt.Id, nil
}

func (a calAdapter) RespondToInvite(ctx context.Context, eventID, attendeeEmail, status string) error {
	return a.c.RespondToInvite(ctx, eventID, attendeeEmail, status, true)
}

// gmailScopes mirrors the OAuth scopes requested by the TUI so the desktop
// client can reuse the same token without re-consent. calendar.events is
// included so a token the desktop mints itself can respond to invites (RSVP) —
// without it, any desktop-initiated auth produced a token that 403'd on the
// Calendar API.
var gmailScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/gmail.compose",
	"https://www.googleapis.com/auth/gmail.settings.basic",
	calendarScope,
}

// ErrNoCredentials indicates the OAuth client credentials file (credentials.json)
// does not exist. This is the desktop first-run case for a user who installed
// via Homebrew/DMG without ever using the TUI: the GUI shows an onboarding
// screen that explains what's needed and offers to import the file, rather than
// dumping a raw filesystem error. Missing *token* is NOT this error — that
// triggers the normal interactive OAuth consent flow.
var ErrNoCredentials = errors.New("credentials.json not found")

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
	cal              calClient
	configPath       string
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

	_, defaultToken := config.DefaultCredentialPaths()
	credPath := credentialsPath(opts, cfg)
	tokenPath := resolvePath(firstNonEmpty(opts.TokenPath, expandPath(cfg.Token)), "GMAIL_TUI_TOKEN", defaultToken)

	// Detect the missing-credentials case up front so the GUI can show a friendly
	// onboarding/import screen. (A missing token is fine — it drives OAuth.)
	if _, statErr := os.Stat(credPath); errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNoCredentials, credPath)
	}

	service, err := auth.NewGmailService(ctx, credPath, tokenPath, gmailScopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gmail service (creds: %s, token: %s): %w", credPath, tokenPath, err)
	}
	client := gmail.NewClient(service)

	dbManager := services.NewDatabaseManager(cfg, logger)
	accountService := services.NewAccountService(cfg, logger)

	// Calendar RSVP is best-effort: the token (created by the TUI) usually carries
	// the calendar.events scope. If it doesn't, RSVP stays disabled.
	var cal calClient
	if calSvc, err := auth.NewCalendarService(ctx, credPath, tokenPath, calendarScope); err == nil {
		cal = calAdapter{c: calendar.NewClient(calSvc)}
	} else if logger != nil {
		logger.Printf("desktop: calendar RSVP disabled: %v", err)
	}

	api := buildAPI(ctx, cfg, client, dbManager, cal, logger)

	sess := &Session{
		API:            api,
		Config:         cfg,
		client:         client,
		dbManager:      dbManager,
		accountService: accountService,
		cal:            cal,
		configPath:     configPath,
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
func buildAPI(ctx context.Context, cfg *config.Config, client *gmail.Client, dbManager services.DatabaseManager, cal calClient, logger *log.Logger) *API {
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

	// PromptService needs both an LLM (aiService) and the local database. Wire the
	// bulk-prompt service too (mirroring the TUI) so "apply a prompt to many
	// messages" works — without it ApplyBulkPrompt returns "bulk prompt service
	// not available". Cache needs the DB, so bulk is gated on cacheService.
	var promptService services.PromptService
	if aiService != nil && dbStore != nil {
		promptImpl := services.NewPromptService(db.NewPromptStore(dbStore), aiService, nil)
		if cacheService != nil {
			bulkPromptService := services.NewBulkPromptService(
				emailService, aiService, cacheService, repo, promptImpl,
			)
			promptImpl.SetBulkService(bulkPromptService)
		}
		promptService = promptImpl
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

	// Deterministic rules (:rules): structured Archive/Trash/Label/Prompt rules
	// with optional mirroring as real Gmail filters. Needs the DB; a nil filter
	// API just disables sync while CRUD/sweeps keep working.
	var detRulesService services.DeterministicRulesService
	if dbStore != nil {
		var filters services.GmailFilterAPI
		if client != nil {
			filters = client
		}
		detRulesService = services.NewDeterministicRulesService(
			db.NewDeterministicRulesStore(dbStore), repo, labelService, filters)
	}

	// Saved queries, analyzer rules and deterministic rules are account-scoped,
	// so they need the active email set or every call fails with "account email
	// not set". The setter lives on the concrete types / this interface.
	type accountScoped interface{ SetAccountEmail(string) }
	if accountEmail != "" {
		if s, ok := queryService.(accountScoped); ok {
			s.SetAccountEmail(accountEmail)
		}
		if s, ok := rulesService.(accountScoped); ok {
			s.SetAccountEmail(accountEmail)
		}
		if detRulesService != nil {
			detRulesService.SetAccountEmail(accountEmail)
		}
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
		DetRules:     detRulesService,
		Theme:        themeService,
		Invite:       client,
		Cal:          cal,
		Cache:        cacheService,
		AccountEmail: accountEmail,
		// Honor the same inbox-analyzer settings the TUI reads from config, so
		// tuning inbox_analyzer.batch_size in config.json affects the desktop too.
		AnalyzerBatchSize:    cfg.InboxAnalyzer.BatchSize,
		AnalyzerMaxBatches:   cfg.InboxAnalyzer.MaxBatches,
		AnalyzerStrictLabels: cfg.InboxAnalyzer.StrictLabels,
		Logger:               logger,
	})
}

// ConfigPath returns the path of the config file this session loaded.
func (s *Session) ConfigPath() string { return s.configPath }

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
	s.API = buildAPI(ctx, s.Config, client, s.dbManager, s.cal, s.logger)
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

// credentialsPath resolves the credentials.json path using the same priority as
// NewSession: explicit option, config value, GMAIL_TUI_CREDENTIALS, then default.
func credentialsPath(opts Options, cfg *config.Config) string {
	defaultCred, _ := config.DefaultCredentialPaths()
	return resolvePath(firstNonEmpty(opts.CredentialsPath, expandPath(cfg.Credentials)), "GMAIL_TUI_CREDENTIALS", defaultCred)
}

// CredentialsPathFor returns where NewSession would look for credentials.json,
// loading config (best-effort) the same way. A GUI uses it to show the expected
// path and to know where InstallCredentials should write.
func CredentialsPathFor(opts Options) string {
	configPath := resolvePath(opts.ConfigPath, "GMAIL_TUI_CONFIG", config.DefaultConfigPath())
	cfg, err := config.LoadConfig(configPath)
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	return credentialsPath(opts, cfg)
}

// InstallCredentials validates a user-selected OAuth credentials.json and copies
// it into the location NewSession reads (creating the parent dir with tight
// perms). It returns the destination path. Validation fails fast on the wrong
// file (e.g. a token, or unrelated JSON) so the user gets a clear message here
// instead of an opaque API error later.
func InstallCredentials(opts Options, srcPath string) (string, error) {
	if srcPath == "" {
		return "", fmt.Errorf("no file selected")
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", srcPath, err)
	}
	if err := validateCredentialsJSON(data); err != nil {
		return "", err
	}
	dest := CredentialsPathFor(opts)
	if dest == "" {
		return "", fmt.Errorf("could not determine where to store credentials.json")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", fmt.Errorf("could not create %s: %w", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", fmt.Errorf("could not write %s: %w", dest, err)
	}
	return dest, nil
}

// validateCredentialsJSON checks the bytes look like a Google OAuth client
// credentials file: valid JSON with an "installed" (desktop) or "web" block that
// carries a client_id. This is a cheap sanity check, not full schema validation.
func validateCredentialsJSON(data []byte) error {
	type oauthClient struct {
		ClientID string `json:"client_id"`
	}
	var probe struct {
		Installed *oauthClient `json:"installed"`
		Web       *oauthClient `json:"web"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("that file is not valid JSON — download the OAuth client credentials from Google Cloud Console")
	}
	block := probe.Installed
	if block == nil {
		block = probe.Web
	}
	if block == nil || block.ClientID == "" {
		return fmt.Errorf(`that JSON doesn't look like an OAuth client credentials file (expected an "installed" or "web" client_id)`)
	}
	return nil
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
