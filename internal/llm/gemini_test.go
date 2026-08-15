package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeminiAPIKeyGenerate(t *testing.T) {
	var gotPath, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"hi "},{"text":"there"}]}}]}`))
	}))
	defer srv.Close()

	c := NewGemini("", "", "gemini-1.5-flash", "AIza-test", srv.URL, 5*time.Second)
	out, err := c.Generate("hello")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out != "hi there" {
		t.Errorf("out = %q", out)
	}
	if !strings.HasSuffix(gotPath, "gemini-1.5-flash:generateContent") {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "AIza-test" {
		t.Errorf("key = %q", gotKey)
	}
}

func TestGeminiAPIKeyStream(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"candidates":[{"content":{"parts":[{"text":"Hel"}]}}]}`,
		`data: {"candidates":[{"content":{"parts":[{"text":"lo"}]}}]}`,
		`data: [DONE]`,
	}, "\n")
	var gotStream bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotStream = strings.HasSuffix(r.URL.Path, ":streamGenerateContent") && r.URL.Query().Get("alt") == "sse"
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	c := NewGemini("", "", "", "AIza-test", srv.URL, 5*time.Second)
	var sb strings.Builder
	if err := c.GenerateStream(context.Background(), "hi", func(tok string) { sb.WriteString(tok) }); err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if sb.String() != "Hello" {
		t.Errorf("aggregated = %q, want Hello", sb.String())
	}
	if !gotStream {
		t.Error("stream request must hit :streamGenerateContent?alt=sse")
	}
}

func TestGeminiVertexURLShape(t *testing.T) {
	// Vertex mode (no api_key) builds a projects/locations URL. We can't exercise
	// ADC here, so just assert the endpoint construction.
	c := NewGemini("my-proj", "us-central1", "gemini-1.5-pro", "", "https://host", 0)
	got := c.endpointURL("generateContent")
	want := "https://host/v1/projects/my-proj/locations/us-central1/publishers/google/models/gemini-1.5-pro:generateContent"
	if got != want {
		t.Errorf("vertex URL = %q, want %q", got, want)
	}
	if c.usesAPIKey() {
		t.Error("no api_key → must not use API-key mode")
	}
}

func TestGeminiVertexRequiresProjectRegion(t *testing.T) {
	// No api_key and no project/region → token() errors before any request.
	c := NewGemini("", "", "gemini-1.5-flash", "", "", 5*time.Second)
	if _, err := c.token(context.Background()); err == nil {
		t.Error("expected error when project/region missing and no api_key")
	}
}

func TestGeminiUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewGemini("", "", "", "bad-key", srv.URL, 5*time.Second)
	_, err := c.Generate("hi")
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 error, got %v", err)
	}
}

func TestNewGeminiDefaults(t *testing.T) {
	c := NewGemini("", "", "", "k", "", 0)
	if c.Model != defaultGeminiModel {
		t.Errorf("model = %q", c.Model)
	}
	if c.Timeout <= 0 {
		t.Error("timeout must be positive")
	}
	if c.Name() != "vertex" {
		t.Errorf("name = %q", c.Name())
	}
	// API-key mode default host.
	if got := c.baseHost(); got != generativeLanguageHost {
		t.Errorf("api-key host = %q", got)
	}
}
