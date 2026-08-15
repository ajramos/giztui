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
)

// OpenAIClient calls the metered OpenAI Chat Completions API (api.openai.com)
// with an API key. This is the standard paid API — distinct from the ChatGPT
// subscription provider (chatgpt.go), which reuses a Plus/Pro subscription over
// OAuth and needs no key. Select it with provider "openai" and set api_key.
type OpenAIClient struct {
	APIKey  string
	Model   string
	Timeout time.Duration
	// BaseURL defaults to https://api.openai.com/v1; override for compatible
	// gateways (Azure OpenAI, local proxies) or tests. No trailing slash.
	BaseURL string
	http    *http.Client
}

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

// NewOpenAI builds a metered OpenAI client. endpoint "" → the public API; model
// "" → a sane default. apiKey is required at call time (empty → a clear error).
func NewOpenAI(apiKey, endpoint, model string, timeout time.Duration) *OpenAIClient {
	if endpoint == "" {
		endpoint = defaultOpenAIBaseURL
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if model == "" {
		model = defaultOpenAIModel
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &OpenAIClient{
		APIKey:  apiKey,
		Model:   model,
		Timeout: timeout,
		BaseURL: endpoint,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *OpenAIClient) Name() string { return "openai" }

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// Generate sends a single-shot (non-streaming) completion and returns the text.
func (c *OpenAIClient) Generate(prompt string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("openai: no api_key configured")
	}
	reqBody := openAIRequest{
		Model:    c.Model,
		Messages: []openAIMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", c.statusError(resp)
	}
	var out struct {
		Choices []struct {
			Message openAIMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("openai: parse response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}
	return out.Choices[0].Message.Content, nil
}

// GenerateStream streams the completion, forwarding each content delta.
func (c *OpenAIClient) GenerateStream(ctx context.Context, prompt string, onToken func(string)) error {
	if c.APIKey == "" {
		return fmt.Errorf("openai: no api_key configured")
	}
	reqBody := openAIRequest{
		Model:    c.Model,
		Messages: []openAIMessage{{Role: "user", Content: prompt}},
		Stream:   true,
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openai request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return c.statusError(resp)
	}
	return parseOpenAISSE(resp.Body, onToken)
}

func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
}

func (c *OpenAIClient) statusError(resp *http.Response) error {
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("openai auth rejected (401) — check the api_key")
	}
	return fmt.Errorf("openai returned %d: %s", resp.StatusCode, strings.TrimSpace(b.String()))
}

// parseOpenAISSE reads the Chat Completions SSE stream. Each data frame is
// {"choices":[{"delta":{"content":"..."}}]}; the stream ends with "[DONE]".
func parseOpenAISSE(r interface{ Read([]byte) (int, error) }, onToken func(string)) error {
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
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // ignore keep-alives / non-JSON frames
		}
		for _, ch := range ev.Choices {
			if ch.Delta.Content != "" {
				onToken(ch.Delta.Content)
			}
		}
	}
	return sc.Err()
}
