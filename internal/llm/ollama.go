package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
    "context"
)

// Provider defines a generic LLM interface
type Provider interface {
	Name() string
	Generate(prompt string) (string, error)
}

// ParamProvider optionally supports passing provider-specific options (e.g., temperature, max_tokens)
// Callers should detect support via type assertion and fallback to Generate when unavailable.
type ParamProvider interface {
	Provider
	GenerateWithParams(prompt string, params map[string]interface{}) (string, error)
}

// StreamProvider optionally supports streaming token callbacks
type StreamProvider interface {
    Provider
    GenerateStream(ctx context.Context, prompt string, onToken func(string)) error
}

// Client represents an Ollama client for local LLM interactions
type Client struct {
	Endpoint string
	Model    string
	Timeout  time.Duration

	// Prompt templates
	SummarizeTemplate string
	ReplyTemplate     string
	LabelTemplate     string
}

// NewClient creates a new Ollama client
func NewClient(endpoint, model string, timeout time.Duration) *Client {
	return &Client{
		Endpoint: endpoint,
		Model:    model,
		Timeout:  timeout,
	}
}

// Request represents the JSON structure expected by Ollama
type Request struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// Response represents the response from Ollama
type Response struct {
	Response string `json:"response"`
}

// Generate sends a prompt to Ollama and returns the generated text
func (c *Client) Generate(prompt string) (string, error) {
	reqBody := Request{Model: c.Model, Prompt: prompt, Stream: false}

	data, err := json.Marshal(reqBody)
	if err != nil {
    return "", fmt.Errorf("could not serialize request: %w", err)
	}

	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Post(c.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
    return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("ollama returned status %s", resp.Status)
	}

	var response Response
	dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&response); err != nil {
        return "", fmt.Errorf("could not decode Ollama response: %w", err)
	}

	return strings.TrimSpace(response.Response), nil
}

// GenerateWithParams sends a prompt to Ollama with options (temperature, num_ctx, etc.)
func (c *Client) GenerateWithParams(prompt string, params map[string]interface{}) (string, error) {
	reqBody := Request{Model: c.Model, Prompt: prompt, Stream: false, Options: params}
	data, err := json.Marshal(reqBody)
	if err != nil {
    return "", fmt.Errorf("could not serialize request: %w", err)
	}
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Post(c.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
    return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("ollama returned status %s", resp.Status)
	}
	var response Response
	dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&response); err != nil {
        return "", fmt.Errorf("could not decode Ollama response: %w", err)
	}
	return strings.TrimSpace(response.Response), nil
}

// GenerateStream streams tokens from Ollama and invokes onToken for each response chunk
func (c *Client) GenerateStream(ctx context.Context, prompt string, onToken func(string)) error {
    reqBody := Request{Model: c.Model, Prompt: prompt, Stream: true}
    data, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("could not serialize request: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(data))
    if err != nil {
        return fmt.Errorf("ollama request build failed: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    client := &http.Client{Timeout: 0} // streaming; rely on ctx for cancellation
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("ollama request failed: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("ollama returned status %s", resp.Status)
    }
    dec := json.NewDecoder(resp.Body)
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        var chunk struct {
            Response string `json:"response"`
            Done     bool   `json:"done"`
        }
        if err := dec.Decode(&chunk); err != nil {
            if err == io.EOF {
                break
            }
            return fmt.Errorf("ollama stream decode error: %w", err)
        }
        if chunk.Response != "" && onToken != nil {
            onToken(chunk.Response)
        }
        if chunk.Done {
            break
        }
    }
    return nil
}

// Name returns provider name
func (c *Client) Name() string { return "ollama" }

// SummarizeEmail generates a summary of an email
func (c *Client) SummarizeEmail(body string) (string, error) {
	if strings.TrimSpace(c.SummarizeTemplate) == "" {
		return "", fmt.Errorf("missing summarize template")
	}
	prompt := strings.ReplaceAll(c.SummarizeTemplate, "{{body}}", body)
	return c.Generate(prompt)
}

// DraftReply generates a reply to an email
func (c *Client) DraftReply(body string) (string, error) {
	if strings.TrimSpace(c.ReplyTemplate) == "" {
		return "", fmt.Errorf("missing reply template")
	}
	prompt := strings.ReplaceAll(c.ReplyTemplate, "{{body}}", body)
	return c.Generate(prompt)
}

// RecommendLabel suggests a label for an email
func (c *Client) RecommendLabel(body string, existing []string) (string, error) {
	if strings.TrimSpace(c.LabelTemplate) == "" {
		return "", fmt.Errorf("missing label template")
	}
	labelsStr := strings.Join(existing, ", ")
	prompt := strings.ReplaceAll(c.LabelTemplate, "{{body}}", body)
	prompt = strings.ReplaceAll(prompt, "{{labels}}", labelsStr)
	return c.Generate(prompt)
}

// IsAvailable checks if the Ollama service is available
func (c *Client) IsAvailable() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.Replace(c.Endpoint, "/api/generate", "/api/tags", 1))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
