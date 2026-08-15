package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GeminiClient talks to Google's Gemini models through one of two backends,
// chosen by how it's configured:
//
//   - **API key** (Generative Language API, `generativelanguage.googleapis.com`):
//     set `api_key`. Simplest path — no Google Cloud project needed.
//   - **Vertex AI** (`{region}-aiplatform.googleapis.com`): set `project` and
//     `region`, no api_key. Auth uses Application Default Credentials (ADC), i.e.
//     `gcloud auth application-default login` or a service account in
//     GOOGLE_APPLICATION_CREDENTIALS.
//
// Both speak the same generateContent request/response shape, so only the URL
// and Authorization differ. Selected with provider "vertex" (or "gemini").
type GeminiClient struct {
	Model   string
	APIKey  string // set → Generative Language API (key auth)
	Project string // set (with Region) → Vertex AI (ADC auth)
	Region  string
	Timeout time.Duration
	// BaseURL overrides the API host (no trailing slash); empty → the default for
	// the selected backend. Set by tests to point at an httptest server.
	BaseURL string

	http *http.Client
	// tokenSource yields ADC bearer tokens for Vertex; lazily initialized.
	tokenSource oauth2.TokenSource
}

const (
	defaultGeminiModel       = "gemini-1.5-flash"
	generativeLanguageHost   = "https://generativelanguage.googleapis.com"
	vertexCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
)

// NewGemini builds a Gemini client. When apiKey is set it uses the API-key
// backend; otherwise it uses Vertex AI with project+region and ADC. model "" → a
// sane default; timeout ≤ 0 → 120s.
func NewGemini(project, region, model, apiKey, endpoint string, timeout time.Duration) *GeminiClient {
	if model == "" {
		model = defaultGeminiModel
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &GeminiClient{
		Model:   model,
		APIKey:  apiKey,
		Project: project,
		Region:  region,
		Timeout: timeout,
		BaseURL: strings.TrimRight(endpoint, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *GeminiClient) Name() string { return "vertex" }

func (c *GeminiClient) usesAPIKey() bool { return c.APIKey != "" }

// baseHost returns the API host for the selected backend.
func (c *GeminiClient) baseHost() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	if c.usesAPIKey() {
		return generativeLanguageHost
	}
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com", c.Region)
}

// endpointURL builds the generateContent (or streamGenerateContent) URL.
func (c *GeminiClient) endpointURL(method string) string {
	host := c.baseHost()
	if c.usesAPIKey() {
		u := fmt.Sprintf("%s/v1beta/models/%s:%s", host, c.Model, method)
		if method == "streamGenerateContent" {
			return u + "?alt=sse&key=" + c.APIKey
		}
		return u + "?key=" + c.APIKey
	}
	u := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
		host, c.Project, c.Region, c.Model, method)
	if method == "streamGenerateContent" {
		return u + "?alt=sse"
	}
	return u
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (c *GeminiClient) newRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if !c.usesAPIKey() {
		tok, err := c.token(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return req, nil
}

// token returns a Vertex ADC access token (cached/refreshed by the TokenSource).
func (c *GeminiClient) token(ctx context.Context) (string, error) {
	if c.Project == "" || c.Region == "" {
		return "", fmt.Errorf("vertex: project and region are required (or set api_key for the Gemini API)")
	}
	if c.tokenSource == nil {
		creds, err := google.FindDefaultCredentials(ctx, vertexCloudPlatformScope)
		if err != nil {
			return "", fmt.Errorf("vertex: no Google credentials (run 'gcloud auth application-default login'): %w", err)
		}
		c.tokenSource = creds.TokenSource
	}
	t, err := c.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("vertex: token: %w", err)
	}
	return t.AccessToken, nil
}

// Generate sends a single-shot (non-streaming) request and returns the text.
func (c *GeminiClient) Generate(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	body, err := json.Marshal(geminiRequest{Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}}})
	if err != nil {
		return "", err
	}
	req, err := c.newRequest(ctx, c.endpointURL("generateContent"), body)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("vertex request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", geminiStatusError(resp)
	}
	var out geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("vertex: parse response: %w", err)
	}
	return geminiText(out), nil
}

// GenerateStream streams the response, forwarding each text delta.
func (c *GeminiClient) GenerateStream(ctx context.Context, prompt string, onToken func(string)) error {
	body, err := json.Marshal(geminiRequest{Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}}})
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, c.endpointURL("streamGenerateContent"), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("vertex request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return geminiStatusError(resp)
	}
	return parseGeminiSSE(resp.Body, onToken)
}

func geminiStatusError(resp *http.Response) error {
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("vertex auth rejected (%d) — check api_key or ADC credentials", resp.StatusCode)
	}
	return fmt.Errorf("vertex returned %d: %s", resp.StatusCode, strings.TrimSpace(b.String()))
}

// geminiText concatenates all parts of the first candidate.
func geminiText(r geminiResponse) string {
	if len(r.Candidates) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, p := range r.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String()
}

// parseGeminiSSE reads the streamGenerateContent SSE stream (alt=sse). Each data
// frame is a partial GenerateContentResponse; forward its candidate text.
func parseGeminiSSE(r interface{ Read([]byte) (int, error) }, onToken func(string)) error {
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
		var ev geminiResponse
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // ignore keep-alives / non-JSON frames
		}
		if txt := geminiText(ev); txt != "" {
			onToken(txt)
		}
	}
	return sc.Err()
}
