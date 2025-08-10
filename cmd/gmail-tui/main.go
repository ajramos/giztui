package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/llm"
	"github.com/ajramos/gmail-tui/internal/tui"
	"github.com/ajramos/gmail-tui/pkg/auth"
)

func main() {
	// Command line flags
	credPathFlag := flag.String("credentials", "", "Ruta al archivo JSON de credenciales del cliente OAuth")
	tokenPathFlag := flag.String("token", "", "Ruta al archivo de token OAuth cacheado")
	ollamaEndpointFlag := flag.String("ollama-endpoint", "", "Endpoint de Ollama (incluye /api/generate)")
	ollamaModelFlag := flag.String("ollama-model", "", "Nombre del modelo de Ollama")
	ollamaTimeoutFlag := flag.Duration("ollama-timeout", 0, "Timeout de la petición al modelo LLM")
	llmProviderFlag := flag.String("llm-provider", "", "Proveedor LLM (ollama, bedrock)")
	llmModelFlag := flag.String("llm-model", "", "Modelo LLM a usar (por ejemplo, anthropic.claude-3-haiku-20240307)")
	llmRegionFlag := flag.String("llm-region", "", "Región para proveedores basados en región (por ejemplo, us-east-1 para Bedrock)")
	configPathFlag := flag.String("config", "", "Ruta al fichero de configuración JSON")
	flag.Parse()

	// Load configuration
	configPath := *configPathFlag
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Advertencia: no se pudo cargar la configuración: %v", err)
		cfg = config.DefaultConfig()
	}

	// Determine credential and token paths
	credPath, tokenPath := config.DefaultCredentialPaths()
	if cfg.Credentials != "" {
		credPath = cfg.Credentials
		// Expand ~ to home directory if present
		if strings.HasPrefix(credPath, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				credPath = filepath.Join(home, credPath[2:])
			}
		}
	}
	if cfg.Token != "" {
		tokenPath = cfg.Token
		// Expand ~ to home directory if present
		if strings.HasPrefix(tokenPath, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				tokenPath = filepath.Join(home, tokenPath[2:])
			}
		}
	}
	if *credPathFlag != "" {
		credPath = *credPathFlag
	}
	if *tokenPathFlag != "" {
		tokenPath = *tokenPathFlag
	}

	// Validate credentials path
	if credPath == "" {
		log.Fatal("Se requiere una ruta al archivo de credenciales de Gmail. Proporciónala mediante --credentials o en el fichero de configuración.")
	}

	if _, err := os.Stat(credPath); err != nil {
		log.Fatalf("No se encontró el archivo de credenciales en %s. Descarga las credenciales de la consola de Google Cloud y colócalas ahí.", credPath)
	}

	// Initialize Gmail service
	ctx := context.Background()
	service, err := auth.NewGmailService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.compose")
	if err != nil {
		log.Fatalf("No se pudo inicializar el servicio de Gmail: %v", err)
	}

	// Create Gmail client
	gmailClient := gmail.NewClient(service)

	// Initialize LLM provider if enabled
	var llmProvider llm.Provider
	endpoint := cfg.LLMEndpoint
	model := cfg.LLMModel
	timeout := cfg.GetLLMTimeout()
	providerName := cfg.LLMProvider
	region := cfg.LLMRegion
	if endpoint == "" && cfg.OllamaEndpoint != "" {
		endpoint = cfg.OllamaEndpoint
	}
	if model == "" && cfg.OllamaModel != "" {
		model = cfg.OllamaModel
	}
	if cfg.OllamaTimeout != "" {
		timeout = cfg.GetOllamaTimeout()
	}
	if *ollamaEndpointFlag != "" {
		endpoint = *ollamaEndpointFlag
	}
	if *ollamaModelFlag != "" {
		model = *ollamaModelFlag
	}
	if *ollamaTimeoutFlag != 0 {
		timeout = *ollamaTimeoutFlag
	}
	if *llmProviderFlag != "" {
		providerName = *llmProviderFlag
	}
	if *llmModelFlag != "" {
		model = *llmModelFlag
	}
	if *llmRegionFlag != "" {
		region = *llmRegionFlag
	}
	if providerName == "" {
		providerName = "ollama"
	}
	if cfg.LLMEnabled && model != "" {
		// For Bedrock, use region; for Ollama, use endpoint
		arg := endpoint
		if providerName == "bedrock" {
			if region == "" {
				if env := os.Getenv("AWS_REGION"); env != "" {
					region = env
				}
			}
			arg = region
		}
		var err error
		llmProvider, err = llm.NewProviderFromConfig(providerName, arg, model, timeout, cfg.LLMAPIKey)
		if err != nil {
			log.Printf("Warning: could not initialize LLM provider (%s): %v", providerName, err)
		}
	}

	// Create and run TUI
	app := tui.NewApp(gmailClient, llmProvider, cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error ejecutando la aplicación: %v\n", err)
		os.Exit(1)
	}
}
