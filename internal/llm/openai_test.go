package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIGenerate(t *testing.T) {
	var gotAuth, gotPath string
	var gotStream bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var body struct {
			Stream   bool `json:"stream"`
			Messages []struct {
				Role, Content string
			} `json:"messages"`
		}
		_ = decodeJSON(r, &body)
		gotStream = body.Stream
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi there"}}]}`))
	}))
	defer srv.Close()

	c := NewOpenAI("sk-test", srv.URL, "gpt-4o-mini", 5*time.Second)
	out, err := c.Generate("hello")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out != "hi there" {
		t.Errorf("out = %q", out)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotStream {
		t.Error("Generate must not request streaming")
	}
}

func TestOpenAIGenerateStream(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: {"choices":[{"delta":{}}]}`,
		`data: [DONE]`,
	}, "\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	c := NewOpenAI("sk-test", srv.URL, "", 5*time.Second)
	var sb strings.Builder
	if err := c.GenerateStream(context.Background(), "hi", func(tok string) { sb.WriteString(tok) }); err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if sb.String() != "Hello" {
		t.Errorf("aggregated = %q, want Hello", sb.String())
	}
}

func TestOpenAINoAPIKey(t *testing.T) {
	c := NewOpenAI("", "", "", 0)
	if _, err := c.Generate("hi"); err == nil {
		t.Error("Generate must error without api_key")
	}
	if err := c.GenerateStream(context.Background(), "hi", func(string) {}); err == nil {
		t.Error("GenerateStream must error without api_key")
	}
}

func TestOpenAIUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewOpenAI("bad", srv.URL, "", 5*time.Second)
	_, err := c.Generate("hi")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got %v", err)
	}
}

func TestNewOpenAIDefaults(t *testing.T) {
	c := NewOpenAI("k", "", "", 0)
	if c.BaseURL != defaultOpenAIBaseURL {
		t.Errorf("baseURL = %q", c.BaseURL)
	}
	if c.Model != defaultOpenAIModel {
		t.Errorf("model = %q", c.Model)
	}
	if c.Timeout <= 0 {
		t.Error("timeout must be positive")
	}
	// Trailing slash on a custom endpoint is trimmed.
	if got := NewOpenAI("k", "https://x/v1/", "m", time.Second).BaseURL; got != "https://x/v1" {
		t.Errorf("trailing slash not trimmed: %q", got)
	}
	if c.Name() != "openai" {
		t.Errorf("name = %q", c.Name())
	}
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
