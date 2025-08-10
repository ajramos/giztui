package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go"
)

// quick Anthropic Claude messages payload
func anthropicMessagesPayload(prompt string) ([]byte, error) {
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
	return json.Marshal(payload)
}

func formatAPIError(err error) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(err.Error())
	if ae, ok := err.(smithy.APIError); ok {
		b.WriteString("\nAPIError code=")
		b.WriteString(ae.ErrorCode())
		b.WriteString(" msg=")
		b.WriteString(ae.ErrorMessage())
	}
	return b.String()
}

func main() {
	var (
		model   string
		region  string
		prompt  string
		timeout time.Duration
	)
	flag.StringVar(&model, "model", "", "Model ID on-demand (with revision :0) o ARN de inference profile")
	flag.StringVar(&region, "region", "", "Región AWS (p. ej., us-east-1)")
	flag.StringVar(&prompt, "prompt", "Hello from gmail-tui bedrock test", "Texto a enviar al modelo")
	flag.DurationVar(&timeout, "timeout", 20*time.Second, "Timeout de la petición")
	flag.Parse()

	if model == "" {
		fmt.Fprintln(os.Stderr, "--model es obligatorio (p. ej., anthropic.claude-3-5-haiku-20241022-v1:0)")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cfg aws.Config
	var err error
	if strings.TrimSpace(region) != "" {
		cfg, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error cargando AWS config: %v\n", err)
		os.Exit(1)
	}
	if cfg.Region == "" && strings.TrimSpace(region) == "" {
		fmt.Fprintln(os.Stderr, "Región no resuelta. Use --region, AWS_REGION o defínala en el perfil.")
		os.Exit(2)
	}

	svc := bedrockruntime.NewFromConfig(cfg)

	body, err := anthropicMessagesPayload(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serializando payload: %v\n", err)
		os.Exit(1)
	}

	callCtx, cancel2 := context.WithTimeout(context.Background(), timeout)
	defer cancel2()

	// Añade revisión :0 si parece un ID on-demand sin revisión (no ARNs)
	modelID := model
	lower := strings.ToLower(modelID)
	if !strings.HasPrefix(lower, "arn:") && !strings.Contains(modelID, ":") {
		modelID = modelID + ":0"
	}

	out, err := svc.InvokeModel(callCtx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "bedrock invoke error:\n%s\n", formatAPIError(err))
		os.Exit(1)
	}

	// Respuesta Anthropic Messages: buscar content[].text o caer a outputText
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		OutputText string `json:"outputText"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		// Mostrar cuerpo bruto si no parsea
		fmt.Println(string(out.Body))
		return
	}
	var text string
	for _, c := range resp.Content {
		if strings.TrimSpace(c.Text) != "" {
			text = c.Text
			break
		}
	}
	if text == "" {
		text = resp.OutputText
	}
	if strings.TrimSpace(text) == "" {
		// Imprimir crudo como último recurso
		io.Copy(os.Stdout, bytes.NewReader(out.Body))
		return
	}
	fmt.Println(strings.TrimSpace(text))
}
