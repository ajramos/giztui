package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// BedrockClient implements Provider for Amazon Bedrock
type BedrockClient struct {
	Region  string
	Model   string
	Timeout time.Duration

	svc *bedrockruntime.Client
}

// NewBedrock initializes a Bedrock client using default AWS config chain
func NewBedrock(region, model string, timeout time.Duration) (*BedrockClient, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("bedrock model is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cfg aws.Config
	var err error
	if strings.TrimSpace(region) != "" {
		cfg, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	} else {
		// Allow region to be resolved from AWS profile/env
		cfg, err = awsconfig.LoadDefaultConfig(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	if cfg.Region == "" && strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("AWS region not resolved. Set llm_region, AWS_REGION or define region in the selected AWS profile")
	}
	client := bedrockruntime.NewFromConfig(cfg)
	return &BedrockClient{Region: region, Model: model, Timeout: timeout, svc: client}, nil
}

// Name returns provider name
func (b *BedrockClient) Name() string { return "bedrock" }

// Generate sends a prompt to Bedrock and returns the generated text
func (b *BedrockClient) Generate(prompt string) (string, error) {
	family := detectBedrockFamily(b.Model)
	switch family {
	case "anthropic":
		return b.generateAnthropic(prompt)
	default:
		return "", fmt.Errorf("unsupported Bedrock model family for %q", b.Model)
	}
}

func (b *BedrockClient) generateAnthropic(prompt string) (string, error) {
	// See: Anthropic Claude 3 messages API via Bedrock (InvokeModel)
	// Normalize model ID conservadoramente: no tocar ARNs ni inference profiles.
	modelID := b.Model
	lower := strings.ToLower(modelID)
	if !strings.HasPrefix(lower, "arn:") && !strings.Contains(lower, "inference-profile/") {
		if !strings.Contains(modelID, ":") {
			// Some integrations require the revision suffix (:0)
			modelID = modelID + ":0"
		}
	}
	payload := map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        1024,
		"temperature":       0.2,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": prompt},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.Timeout)
	defer cancel()
	// Build InvokeModel input. SDK v2 uses only ModelId even for profile ARNs.
	out, err := b.svc.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return "", annotateBedrockError(fmt.Errorf("bedrock invoke error: %w", err), modelID)
	}
	defer func() { _, _ = io.Copy(io.Discard, bytes.NewReader(out.Body)) }()

	// Parse Claude 3 messages response
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		// Try to parse Anthropic text response format as fallback
		var alt struct {
			OutputText string `json:"outputText"`
		}
		if err2 := json.Unmarshal(out.Body, &alt); err2 == nil && strings.TrimSpace(alt.OutputText) != "" {
			return strings.TrimSpace(alt.OutputText), nil
		}
		return "", fmt.Errorf("failed to decode Anthropic response: %w", err)
	}
	for _, c := range resp.Content {
		if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
			return strings.TrimSpace(c.Text), nil
		}
	}
	return "", fmt.Errorf("empty response from Bedrock Anthropic model")
}

func detectBedrockFamily(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "anthropic.") || strings.Contains(m, "anthropic."):
		return "anthropic"
	case strings.HasPrefix(m, "meta.") || strings.Contains(m, "meta."):
		return "meta" // not yet implemented
	case strings.HasPrefix(m, "amazon.titan") || strings.Contains(m, "amazon.titan"):
		return "titan" // not yet implemented
	default:
		return ""
	}
}

// annotateBedrockError adds common hints for Bedrock model ID issues
func annotateBedrockError(err error, modelID string) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "validationexception") && strings.Contains(msg, "throughput isn't supported") {
		return fmt.Errorf("%v\nHint: This model may require an inference profile. Try setting llm_model to the profile ID/ARN for %q, or ensure the ID includes region/vendor and revision (e.g., us.anthropic...:0)", err, modelID)
	}
	if strings.Contains(msg, "provided model identifier is invalid") {
		return fmt.Errorf("%v\nHint: Verify the exact Bedrock ModelId or use the inference profile ID. Regional prefixes (e.g., us.) and revision suffix (:0) may be required", err)
	}
	return err
}
