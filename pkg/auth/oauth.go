package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// authResultPage renders the branded page the browser lands on after the Google
// OAuth redirect.
func authResultPage(ok bool) string {
	if ok {
		return ResultPage(true, "You're all set",
			"GizTUI has been authorized. You can close this tab and return to the app.")
	}
	return ResultPage(false, "Something went wrong",
		"No authorization code was received. Close this tab and try signing in again from GizTUI.")
}

// ResultPage renders the branded page a browser lands on after an OAuth redirect.
// It's a full, self-contained document (inline CSS, light/dark aware) so it looks
// like part of GizTUI instead of a bare "Authorization successful". ok picks the
// success (green ✓) or error (red !) accent; title/body are the shown copy.
// Exported so other OAuth flows (e.g. the ChatGPT subscription login) share the
// same look. title/body are plain text — the caller must not pass HTML.
func ResultPage(ok bool, title, body string) string {
	accent, mark := "#16a34a", "✓"
	if !ok {
		accent, mark = "#dc2626", "!"
	}
	return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GizTUI — ` + title + `</title>
<style>
:root{color-scheme:light dark}
*{box-sizing:border-box}
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
  font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
  background:#0f1115;color:#e6e8ec}
@media (prefers-color-scheme:light){body{background:#f4f6fb;color:#1b1e24}}
.card{max-width:440px;padding:40px 36px;border-radius:16px;text-align:center;
  background:#171a21;border:1px solid #262b36;box-shadow:0 20px 60px rgba(0,0,0,.35)}
@media (prefers-color-scheme:light){.card{background:#fff;border-color:#e3e8f0;box-shadow:0 20px 60px rgba(20,30,60,.12)}}
.logo{font-size:34px;color:#6d8bff;margin-bottom:8px}
.mark{width:60px;height:60px;margin:6px auto 16px;border-radius:50%;
  display:flex;align-items:center;justify-content:center;font-size:30px;font-weight:700;color:#fff;
  background:` + accent + `}
h1{margin:0 0 8px;font-size:20px}
p{margin:0;opacity:.75}
.hint{margin-top:18px;font-size:12px;opacity:.5}
</style></head>
<body><div class="card">
<div class="logo">✦</div>
<div class="mark">` + mark + `</div>
<h1>` + title + `</h1>
<p>` + body + `</p>
<p class="hint">GizTUI · Gmail terminal &amp; desktop client</p>
</div>
<script>setTimeout(function(){try{window.close()}catch(e){}},2500)</script>
</body></html>`
}

// AuthURLHook, when set, is called with the OAuth consent URL as soon as
// interactive authorization begins. GUI front-ends (e.g. the Wails desktop) set
// it to open the URL in the system browser and surface it in a modal, instead of
// relying on the URL that authenticate() also prints to stdout. It is a no-op by
// default, so the CLI/TUI behaviour is unchanged.
var AuthURLHook func(url string)

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	CredentialsPath string
	TokenPath       string
	Scopes          []string
	AccountName     string // Optional account name for better user messaging
}

// NewOAuth2Config creates a new OAuth2 configuration
func NewOAuth2Config(credentialsPath string, tokenPath string, scopes ...string) *OAuth2Config {
	return &OAuth2Config{
		CredentialsPath: credentialsPath,
		TokenPath:       tokenPath,
		Scopes:          scopes,
	}
}

// SetAccountName sets the account name for better user messaging during OAuth
func (c *OAuth2Config) SetAccountName(accountName string) {
	c.AccountName = accountName
}

// LoadCredentials loads OAuth2 credentials from file
func (c *OAuth2Config) LoadCredentials() (*oauth2.Config, error) {
	data, err := os.ReadFile(c.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read credentials file: %w", err)
	}

	config, err := google.ConfigFromJSON(data, c.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("could not parse credentials file: %w", err)
	}

	return config, nil
}

// LoadToken loads cached token from file
func (c *OAuth2Config) LoadToken(config *oauth2.Config) (*oauth2.Token, error) {
	f, err := os.Open(c.TokenPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// SaveToken saves token to file
func (c *OAuth2Config) SaveToken(token *oauth2.Token) error {
	// Ensure directory exists
	dir := filepath.Dir(c.TokenPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	f, err := os.OpenFile(c.TokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not save OAuth token: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	// #nosec G117 -- persisting the OAuth token to disk is the intended behaviour; the file is created with 0600 perms above to keep it private to the user.
	return json.NewEncoder(f).Encode(token)
}

// GetToken retrieves a token, refreshing if necessary
func (c *OAuth2Config) GetToken(ctx context.Context) (*oauth2.Token, error) {
	config, err := c.LoadCredentials()
	if err != nil {
		return nil, err
	}

	// Try to load cached token
	token, err := c.LoadToken(config)
	if err != nil {
		// Token not found, need to authenticate
		token, err = c.authenticate(ctx, config)
		if err != nil {
			return nil, err
		}
	}

	// Refresh token if needed
	if !token.Valid() {
		token, err = c.refreshToken(ctx, config, token)
		if err != nil {
			// Check if refresh token is invalid (expired or revoked)
			if strings.Contains(err.Error(), "invalid_grant") ||
				strings.Contains(err.Error(), "Token has been expired or revoked") {
				// Refresh token is invalid, need to re-authenticate
				fmt.Println("\n⚠️  Your Gmail access token has expired or been revoked.")
				fmt.Println("🔐 Re-authentication is required to continue using Gmail TUI.")
				token, err = c.authenticate(ctx, config)
				if err != nil {
					return nil, fmt.Errorf("re-authentication failed: %w", err)
				}
				fmt.Println("✅ Successfully re-authenticated! Gmail TUI is ready to use.")
			} else {
				return nil, fmt.Errorf("token refresh failed: %w", err)
			}
		}
	}

	// Save refreshed token
	if err := c.SaveToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

// authenticate performs OAuth2 authentication with local server
func (c *OAuth2Config) authenticate(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Create a local server to capture the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start local server
	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if code != "" {
				// Send success response
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(authResultPage(true)))
				codeChan <- code
			} else {
				// Send error response
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(authResultPage(false)))
				errorChan <- fmt.Errorf("authorization code not received")
			}
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	// Create OAuth2 config with local redirect URI
	localConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  "http://localhost:8080",
		Scopes:       config.Scopes,
		Endpoint:     config.Endpoint,
	}

	authURL := localConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if c.AccountName != "" {
		fmt.Printf("\n🔐 Authorization required for account: %s\n", c.AccountName)
	} else {
		fmt.Printf("\n🔐 Authorization required\n")
	}
	fmt.Printf("1. Open this link: %s\n", authURL)
	fmt.Printf("2. Grant access to the application\n")
	fmt.Printf("3. You will be redirected automatically\n")
	if c.AccountName != "" {
		fmt.Printf("\nWaiting for authorization for %s...\n", c.AccountName)
	} else {
		fmt.Printf("\nWaiting for authorization...\n")
	}
	// Let a GUI front-end open the URL in the browser and show it in a modal.
	if AuthURLHook != nil {
		AuthURLHook(authURL)
	}

	// Wait for authorization code
	var authCode string
	select {
	case authCode = <-codeChan:
		// Success
	case err := <-errorChan:
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("local server error: %w", err)
	case <-time.After(5 * time.Minute):
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("authorization timeout exceeded")
	}

	// Shutdown server
	_ = server.Shutdown(ctx)

	// Exchange code for token
	token, err := localConfig.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("could not exchange authorization code for token: %w", err)
	}

	fmt.Printf("✅ Authorization successful!\n")
	return token, nil
}

// refreshToken refreshes an expired token
func (c *OAuth2Config) refreshToken(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("could not refresh token: %w", err)
	}

	return newToken, nil
}

// NewGmailService creates a new Gmail service using OAuth2
func NewGmailService(ctx context.Context, credentialsPath, tokenPath string, scopes ...string) (*gmail.Service, error) {
	return NewGmailServiceWithAccount(ctx, credentialsPath, tokenPath, "", scopes...)
}

// NewGmailServiceWithAccount creates a new Gmail service using OAuth2 with account context for better user messaging
func NewGmailServiceWithAccount(ctx context.Context, credentialsPath, tokenPath string, accountName string, scopes ...string) (*gmail.Service, error) {
	oauthConfig := NewOAuth2Config(credentialsPath, tokenPath, scopes...)

	// Set account context for better user messaging during OAuth
	oauthConfig.SetAccountName(accountName)

	token, err := oauthConfig.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	config, err := oauthConfig.LoadCredentials()
	if err != nil {
		return nil, err
	}

	httpClient := config.Client(ctx, token)

	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("could not create Gmail service: %w", err)
	}

	return service, nil
}

// NewCalendarService creates a new Google Calendar service using OAuth2
func NewCalendarService(ctx context.Context, credentialsPath, tokenPath string, scopes ...string) (*calapi.Service, error) {
	oauthConfig := NewOAuth2Config(credentialsPath, tokenPath, scopes...)

	token, err := oauthConfig.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	config, err := oauthConfig.LoadCredentials()
	if err != nil {
		return nil, err
	}

	httpClient := config.Client(ctx, token)

	service, err := calapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("could not create Calendar service: %w", err)
	}

	return service, nil
}
