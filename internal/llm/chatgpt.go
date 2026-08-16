package llm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/llm/authstore"
)

// ChatGPT subscription provider — reuses a ChatGPT Plus/Pro subscription via the
// same OAuth method OpenAI's Codex CLI uses, instead of the metered API. It logs
// in with OAuth 2.0 + PKCE against auth.openai.com, stores the tokens in the
// machine-wide authstore, and calls the Codex backend (Responses API) with the
// access token.
//
// EXPERIMENTAL / UNOFFICIAL: the backend endpoint, headers and request/response
// shape below mirror the Codex CLI and community plugins (opencode-openai-codex-
// auth) and are NOT an officially supported API. They can change or break, and
// this is intended for personal use only. The OAuth constants (client_id, token
// endpoint) are the Codex CLI's public values; the request/response shape is the
// OpenAI Responses API streaming format. If OpenAI changes the backend, adjust
// the constants + parseCodexSSE below. This path cannot be exercised without a
// real subscription, so it is opt-in (`:llm login chatgpt`) and best-effort.
const (
	chatgptProviderKey = "chatgpt"

	openaiAuthorizeURL  = "https://auth.openai.com/oauth/authorize"
	openaiTokenURL      = "https://auth.openai.com/oauth/token" //nolint:gosec // OAuth endpoint, not a credential
	openaiOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"        // Codex CLI public client id
	// Scope must match the Codex CLI exactly — the extra api.connectors.* scopes
	// are part of the registered client and OpenAI rejects the request otherwise.
	openaiOAuthScope = "openid profile email offline_access api.connectors.read api.connectors.invoke"
	// The registered redirect_uri is on "localhost" (not 127.0.0.1); we bind the
	// loopback listener on 127.0.0.1 and rely on localhost→127.0.0.1 resolution.
	openaiCallbackAddr = "127.0.0.1:1455"
	openaiCallbackHost = "localhost:1455"
	openaiCallbackPath = "/auth/callback"
	// originator identifies the client to the Codex backend and the authorize flow.
	openaiOriginator = "codex_cli_rs"

	codexResponsesURL = "https://chatgpt.com/backend-api/codex/responses"

	defaultChatGPTModel = "gpt-5"
	tokenRefreshSkew    = 2 * time.Minute
)

// ChatGPTClient implements Provider + StreamProvider, backed by OAuth tokens in
// the authstore.
type ChatGPTClient struct {
	Model   string
	Timeout time.Duration
	store   *authstore.Store
	http    *http.Client
	// Endpoints default to the OpenAI constants; overridable for tests (and to
	// adjust if OpenAI changes the backend).
	tokenURL     string
	responsesURL string
}

// NewChatGPT builds a ChatGPT subscription client. model "" → a sane default.
func NewChatGPT(model string, timeout time.Duration) *ChatGPTClient {
	if model == "" {
		model = defaultChatGPTModel
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &ChatGPTClient{
		Model:        model,
		Timeout:      timeout,
		store:        authstore.New(authstore.DefaultPath()),
		http:         &http.Client{Timeout: timeout},
		tokenURL:     openaiTokenURL,
		responsesURL: codexResponsesURL,
	}
}

func (c *ChatGPTClient) Name() string { return "chatgpt" }

// LoggedIn reports whether a stored access token exists.
func (c *ChatGPTClient) LoggedIn() bool {
	t, ok, err := c.store.Get(chatgptProviderKey)
	return err == nil && ok && t.AccessToken != ""
}

// Logout drops the stored tokens.
func (c *ChatGPTClient) Logout() error { return c.store.Delete(chatgptProviderKey) }

// ---- OAuth login (PKCE) -----------------------------------------------------

// StartLogin begins the OAuth PKCE flow: it starts a loopback callback server
// and returns the authorization URL to open plus a wait func that blocks until
// the browser redirect completes, exchanges the code, and persists the token.
// The caller (e.g. the :llm command) shows authURL and runs wait() in the
// background. ctx cancels the wait.
func (c *ChatGPTClient) StartLogin(ctx context.Context) (authURL string, wait func() error, err error) {
	verifier, challenge, err := pkce()
	if err != nil {
		return "", nil, err
	}
	state, err := randomURLSafe(24)
	if err != nil {
		return "", nil, err
	}
	redirectURI := "http://" + openaiCallbackHost + openaiCallbackPath

	ln, err := net.Listen("tcp", openaiCallbackAddr)
	if err != nil {
		return "", nil, fmt.Errorf("cannot bind %s for the login callback: %w", openaiCallbackAddr, err)
	}

	q := url.Values{
		"response_type":              {"code"},
		"client_id":                  {openaiOAuthClientID},
		"redirect_uri":               {redirectURI},
		"scope":                      {openaiOAuthScope},
		"code_challenge":             {challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"originator":                 {openaiOriginator},
		"state":                      {state},
	}
	authURL = openaiAuthorizeURL + "?" + q.Encode()

	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(openaiCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		if e := r.URL.Query().Get("error"); e != "" {
			http.Error(w, "login failed: "+e, http.StatusBadRequest)
			done <- result{err: fmt.Errorf("authorization error: %s", e)}
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			done <- result{err: fmt.Errorf("state mismatch (possible CSRF)")}
			return
		}
		code := r.URL.Query().Get("code")
		_, _ = w.Write([]byte("<html><body>GizTUI: ChatGPT login complete. You can close this tab.</body></html>"))
		done <- result{code: code}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()

	// NOTE: we do NOT auto-open a browser here — the caller decides (the TUI
	// copies the URL to the clipboard so the user can paste it into whatever
	// browser they want; the desktop opens it explicitly). This avoids forcing
	// the system-default browser on users who log in elsewhere.

	wait = func() error {
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutCtx)
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-done:
			if res.err != nil {
				return res.err
			}
			if res.code == "" {
				return fmt.Errorf("no authorization code received")
			}
			tok, err := c.exchangeCode(ctx, res.code, verifier, redirectURI)
			if err != nil {
				return err
			}
			return c.store.Put(chatgptProviderKey, tok)
		}
	}
	return authURL, wait, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (c *ChatGPTClient) exchangeCode(ctx context.Context, code, verifier, redirectURI string) (authstore.Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {openaiOAuthClientID},
		"code_verifier": {verifier},
	}
	return c.postToken(ctx, form)
}

func (c *ChatGPTClient) refresh(ctx context.Context, refreshToken string) (authstore.Token, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {openaiOAuthClientID},
		"scope":         {openaiOAuthScope},
	}
	tok, err := c.postToken(ctx, form)
	if err != nil {
		return tok, err
	}
	if tok.RefreshToken == "" { // some servers omit a new refresh token on refresh
		tok.RefreshToken = refreshToken
	}
	return tok, nil
}

func (c *ChatGPTClient) postToken(ctx context.Context, form url.Values) (authstore.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return authstore.Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return authstore.Token{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return authstore.Token{}, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, strings.TrimSpace(buf.String()))
	}
	var tr tokenResponse
	if err := json.Unmarshal(buf.Bytes(), &tr); err != nil {
		return authstore.Token{}, fmt.Errorf("parse token response: %w", err)
	}
	tok := authstore.Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		IDToken:      tr.IDToken,
		AccountID:    accountIDFromIDToken(tr.IDToken),
	}
	if tr.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tok, nil
}

// validToken returns a usable access token, refreshing it when near expiry.
func (c *ChatGPTClient) validToken(ctx context.Context) (authstore.Token, error) {
	tok, ok, err := c.store.Get(chatgptProviderKey)
	if err != nil {
		return authstore.Token{}, err
	}
	if !ok || tok.AccessToken == "" {
		return authstore.Token{}, fmt.Errorf("not logged in — run :llm login chatgpt")
	}
	if tok.Expired(tokenRefreshSkew) && tok.RefreshToken != "" {
		refreshed, rerr := c.refresh(ctx, tok.RefreshToken)
		if rerr != nil {
			return authstore.Token{}, fmt.Errorf("token refresh failed (re-run :llm login chatgpt): %w", rerr)
		}
		if refreshed.AccountID == "" {
			refreshed.AccountID = tok.AccountID
		}
		if perr := c.store.Put(chatgptProviderKey, refreshed); perr != nil {
			return authstore.Token{}, perr
		}
		return refreshed, nil
	}
	return tok, nil
}

// ---- model calls (Codex backend, Responses API shape) -----------------------

// Generate aggregates the streamed response into a single string.
func (c *ChatGPTClient) Generate(prompt string) (string, error) {
	var sb strings.Builder
	err := c.GenerateStream(context.Background(), prompt, func(tok string) { sb.WriteString(tok) })
	return sb.String(), err
}

// GenerateStream sends the prompt to the Codex backend and streams tokens.
func (c *ChatGPTClient) GenerateStream(ctx context.Context, prompt string, onToken func(string)) error {
	tok, err := c.validToken(ctx)
	if err != nil {
		return err
	}
	body := map[string]interface{}{
		"model":  c.Model,
		"stream": true,
		"store":  false,
		"input": []map[string]interface{}{{
			"role":    "user",
			"content": []map[string]interface{}{{"type": "input_text", "text": prompt}},
		}},
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.responsesURL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("originator", openaiOriginator)
	if tok.AccountID != "" {
		req.Header.Set("chatgpt-account-id", tok.AccountID)
	}
	if sid, e := randomURLSafe(16); e == nil {
		req.Header.Set("session_id", sid)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("ChatGPT auth rejected (401) — re-run :llm login chatgpt")
		}
		return fmt.Errorf("codex backend %d: %s", resp.StatusCode, strings.TrimSpace(eb.String()))
	}
	return parseCodexSSE(resp.Body, onToken)
}

// parseCodexSSE reads the Responses-API SSE stream and forwards text deltas.
// Event shape (Responses API): lines "data: {json}"; text arrives as
// {"type":"response.output_text.delta","delta":"..."}; the stream ends on
// {"type":"response.completed"} or a "[DONE]" sentinel.
func parseCodexSSE(r interface{ Read([]byte) (int, error) }, onToken func(string)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var ev struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // ignore keep-alives / non-JSON frames
		}
		switch {
		case ev.Type == "response.output_text.delta" && ev.Delta != "":
			onToken(ev.Delta)
		case ev.Type == "response.completed":
			return nil
		case ev.Type == "response.failed" || ev.Error.Message != "":
			return fmt.Errorf("codex stream error: %s", ev.Error.Message)
		}
	}
	return sc.Err()
}

// ---- helpers ----------------------------------------------------------------

func pkce() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLSafe(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// accountIDFromIDToken pulls the ChatGPT account id from the id_token JWT claims
// (claim "https://api.openai.com/auth" → "chatgpt_account_id"). Best-effort: an
// empty result just omits the chatgpt-account-id header.
func accountIDFromIDToken(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Auth struct {
			ChatGPTAccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Auth.ChatGPTAccountID
}

// OpenBrowser opens u in the system default browser (best-effort). Exported so
// GUI callers (the desktop) can open the login URL; the TUI copies it to the
// clipboard instead so the user can choose their browser.
func OpenBrowser(u string) error { return openBrowser(u) }

func openBrowser(u string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	// #nosec G204 -- u is our own generated OAuth authorize URL (constants +
	// PKCE), not user-supplied input.
	return exec.Command(cmd, append(args, u)...).Start()
}
