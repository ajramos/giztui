package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Provider defines a generic LLM interface
type Provider interface {
	Name() string
	Generate(prompt string) (string, error)
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
	reqBody := Request{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("no se pudo serializar la petición: %w", err)
	}

	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Post(c.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("falló la petición a Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama devolvió estado %s", resp.Status)
	}

	var response Response
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&response); err != nil {
		return "", fmt.Errorf("no se pudo decodificar la respuesta de Ollama: %w", err)
	}

	return strings.TrimSpace(response.Response), nil
}

// Name returns provider name
func (c *Client) Name() string { return "ollama" }

// SummarizeEmail generates a summary of an email
func (c *Client) SummarizeEmail(body string) (string, error) {
	template := c.SummarizeTemplate
	if template == "" {
		template = "Resume brevemente el siguiente correo electrónico:\n\n{{body}}\n\nDevuelve el resumen en español en un párrafo."
	}

	prompt := strings.ReplaceAll(template, "{{body}}", body)
	return c.Generate(prompt)
}

// DraftReply generates a reply to an email
func (c *Client) DraftReply(body string) (string, error) {
	template := c.ReplyTemplate
	if template == "" {
		template = "Redacta una respuesta profesional y amable al siguiente correo:\n\n{{body}}"
	}

	prompt := strings.ReplaceAll(template, "{{body}}", body)
	return c.Generate(prompt)
}

// RecommendLabel suggests a label for an email
func (c *Client) RecommendLabel(body string, existing []string) (string, error) {
	template := c.LabelTemplate
	if template == "" {
		template = "Sugiere una etiqueta adecuada para el siguiente correo considerando las ya existentes: {{labels}}.\n\nCorreo:\n{{body}}"
	}

	labelsStr := strings.Join(existing, ", ")
	prompt := strings.ReplaceAll(template, "{{body}}", body)
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
