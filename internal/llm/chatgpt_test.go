package llm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/llm/authstore"
)

// newTestChatGPT builds a client whose token store points at a temp file and
// whose endpoints point at the given test servers.
func newTestChatGPT(t *testing.T, tokenURL, responsesURL string) *ChatGPTClient {
	t.Helper()
	return &ChatGPTClient{
		Model:        "gpt-5",
		Timeout:      5 * time.Second,
		store:        authstore.New(filepath.Join(t.TempDir(), "llm-auth.json")),
		http:         &http.Client{Timeout: 5 * time.Second},
		tokenURL:     tokenURL,
		responsesURL: responsesURL,
	}
}

func TestPKCE(t *testing.T) {
	verifier, challenge, err := pkce()
	if err != nil {
		t.Fatalf("pkce: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatal("empty verifier/challenge")
	}
	// challenge must be the base64url-unpadded SHA-256 of the verifier.
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge = %q, want %q", challenge, want)
	}
	if strings.ContainsAny(challenge, "=+/") {
		t.Errorf("challenge is not url-safe/unpadded: %q", challenge)
	}
	// Two calls must not collide.
	if v2, _, _ := pkce(); v2 == verifier {
		t.Error("verifier not random across calls")
	}
}

func TestAccountIDFromIDToken(t *testing.T) {
	// Craft a JWT-shaped token: header.payload.signature (only payload is read).
	claims := map[string]interface{}{
		"https://api.openai.com/auth": map[string]interface{}{
			"chatgpt_account_id": "acct-xyz",
		},
	}
	payload, _ := json.Marshal(claims)
	jwt := "hdr." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	if got := accountIDFromIDToken(jwt); got != "acct-xyz" {
		t.Errorf("accountID = %q, want acct-xyz", got)
	}

	// Malformed inputs return "" without panicking.
	for _, bad := range []string{"", "onlyone", "a.!!!.c", "a.b"} {
		if got := accountIDFromIDToken(bad); got != "" {
			t.Errorf("accountIDFromIDToken(%q) = %q, want empty", bad, got)
		}
	}
}

func TestExchangeCodeAndStore(t *testing.T) {
	// A token server that returns a canned token for an authorization_code grant.
	var gotGrant, gotVerifier string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		gotVerifier = r.Form.Get("code_verifier")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at1","refresh_token":"rt1","id_token":"","expires_in":3600}`))
	}))
	defer srv.Close()

	c := newTestChatGPT(t, srv.URL, "")
	tok, err := c.exchangeCode(context.Background(), "the-code", "the-verifier", "http://cb")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if gotGrant != "authorization_code" {
		t.Errorf("grant_type = %q", gotGrant)
	}
	if gotVerifier != "the-verifier" {
		t.Errorf("code_verifier = %q", gotVerifier)
	}
	if tok.AccessToken != "at1" || tok.RefreshToken != "rt1" {
		t.Errorf("token = %+v", tok)
	}
	if tok.Expiry.IsZero() {
		t.Error("expiry not set from expires_in")
	}
}

func TestRefreshKeepsRefreshTokenWhenOmitted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		// No refresh_token in the response — the client must keep the old one.
		_, _ = w.Write([]byte(`{"access_token":"at2","expires_in":3600}`))
	}))
	defer srv.Close()

	c := newTestChatGPT(t, srv.URL, "")
	tok, err := c.refresh(context.Background(), "old-rt")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tok.AccessToken != "at2" {
		t.Errorf("access = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "old-rt" {
		t.Errorf("refresh token not preserved: %q", tok.RefreshToken)
	}
}

func TestPostTokenErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestChatGPT(t, srv.URL, "")
	if _, err := c.exchangeCode(context.Background(), "c", "v", "http://cb"); err == nil {
		t.Fatal("expected error on non-200 token response")
	}
}

func TestValidTokenRefreshesNearExpiry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"fresh","refresh_token":"rt2","expires_in":3600}`))
	}))
	defer srv.Close()

	c := newTestChatGPT(t, srv.URL, "")
	// Seed a near-expired token with a refresh token.
	seed := authstore.Token{
		AccessToken:  "stale",
		RefreshToken: "rt1",
		AccountID:    "acct-1",
		Expiry:       time.Now().Add(30 * time.Second), // within tokenRefreshSkew
	}
	if err := c.store.Put(chatgptProviderKey, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tok, err := c.validToken(context.Background())
	if err != nil {
		t.Fatalf("validToken: %v", err)
	}
	if tok.AccessToken != "fresh" {
		t.Errorf("expected refreshed token, got %q", tok.AccessToken)
	}
	// AccountID carries over when the refresh response omits an id_token.
	if tok.AccountID != "acct-1" {
		t.Errorf("accountID not preserved: %q", tok.AccountID)
	}
	// The refreshed token must be persisted.
	if stored, ok, _ := c.store.Get(chatgptProviderKey); !ok || stored.AccessToken != "fresh" {
		t.Errorf("refreshed token not persisted: ok=%v %+v", ok, stored)
	}
}

func TestValidTokenNotLoggedIn(t *testing.T) {
	c := newTestChatGPT(t, "", "")
	if _, err := c.validToken(context.Background()); err == nil {
		t.Fatal("expected error when not logged in")
	}
}

func TestGenerateStreamParsesSSE(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"Hello"}`,
		"",
		`data: {"type":"response.output_text.delta","delta":", world"}`,
		"",
		`: keep-alive`,
		"",
		`data: {"type":"response.completed"}`,
		"",
	}, "\n")

	var gotAuth, gotAccount string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("chatgpt-account-id")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	c := newTestChatGPT(t, "", srv.URL)
	if err := c.store.Put(chatgptProviderKey, authstore.Token{
		AccessToken: "at-live", AccountID: "acct-7",
		Expiry: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var sb strings.Builder
	if err := c.GenerateStream(context.Background(), "hi", func(tok string) { sb.WriteString(tok) }); err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if sb.String() != "Hello, world" {
		t.Errorf("aggregated = %q, want %q", sb.String(), "Hello, world")
	}
	if gotAuth != "Bearer at-live" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotAccount != "acct-7" {
		t.Errorf("chatgpt-account-id = %q", gotAccount)
	}
}

func TestGenerateStreamUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestChatGPT(t, "", srv.URL)
	_ = c.store.Put(chatgptProviderKey, authstore.Token{AccessToken: "x", Expiry: time.Now().Add(time.Hour)})
	err := c.GenerateStream(context.Background(), "hi", func(string) {})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got %v", err)
	}
}

func TestParseCodexSSEError(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"partial"}`,
		`data: {"type":"response.failed","error":{"message":"boom"}}`,
	}, "\n")
	var got strings.Builder
	err := parseCodexSSE(strings.NewReader(sse), func(tok string) { got.WriteString(tok) })
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected stream error, got %v", err)
	}
	if got.String() != "partial" {
		t.Errorf("delta before failure = %q", got.String())
	}
}

func TestCodexResetHint(t *testing.T) {
	// plan + multi-day reset.
	got := codexResetHint(`{"error":{"type":"usage_limit_reached","plan_type":"plus","resets_in_seconds":309070}}`)
	if !strings.Contains(got, "plan plus") || !strings.Contains(got, "3d") {
		t.Errorf("hint = %q, want plan + ~3d", got)
	}
	// sub-day reset, no plan.
	got = codexResetHint(`{"error":{"resets_in_seconds":7200}}`)
	if !strings.Contains(got, "2h") {
		t.Errorf("hint = %q, want ~2h", got)
	}
	// absent fields → empty.
	if h := codexResetHint(`{"error":{"type":"x"}}`); h != "" {
		t.Errorf("hint = %q, want empty", h)
	}
	// malformed → empty, no panic.
	if h := codexResetHint("not json"); h != "" {
		t.Errorf("hint = %q, want empty", h)
	}
}

func TestNewChatGPTDefaults(t *testing.T) {
	c := NewChatGPT("", 0)
	if c.Model != defaultChatGPTModel {
		t.Errorf("model = %q, want default", c.Model)
	}
	if c.Timeout <= 0 {
		t.Error("timeout must be positive")
	}
	if c.tokenURL != openaiTokenURL || c.responsesURL != codexResponsesURL {
		t.Error("endpoints not defaulted to constants")
	}
	if c.Name() != "chatgpt" {
		t.Errorf("name = %q", c.Name())
	}
}
