package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/gmail"
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

	client *gmail.Client
	logger *log.Logger
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

	repo := services.NewMessageRepository(client)
	labelService := services.NewLabelService(client)
	renderer := render.NewEmailRenderer(cfg)
	emailService := services.NewEmailService(repo, client, renderer)

	api := NewAPI(repo, emailService, labelService, client, logger)

	return &Session{
		API:    api,
		Config: cfg,
		client: client,
		logger: logger,
	}, nil
}

// AccountEmail returns the active account's email address.
func (s *Session) AccountEmail(ctx context.Context) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("gmail client not initialized")
	}
	return s.client.ActiveAccountEmail(ctx)
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
