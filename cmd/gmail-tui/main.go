package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/gmail-tui/internal/cache"
	"github.com/ajramos/gmail-tui/internal/calendar"
	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/llm"
	"github.com/ajramos/gmail-tui/internal/tui"
	"github.com/ajramos/gmail-tui/pkg/auth"
)

func main() {
	// Command line flags
	credPathFlag := flag.String("credentials", "", "Path to OAuth client credentials JSON")
	tokenPathFlag := flag.String("token", "", "Path to cached OAuth token JSON")
	ollamaEndpointFlag := flag.String("ollama-endpoint", "", "Ollama endpoint (include /api/generate)")
	ollamaModelFlag := flag.String("ollama-model", "", "Ollama model name")
	ollamaTimeoutFlag := flag.Duration("ollama-timeout", 0, "LLM request timeout")
	llmProviderFlag := flag.String("llm-provider", "", "LLM provider (ollama, bedrock)")
	llmModelFlag := flag.String("llm-model", "", "LLM model to use (e.g., anthropic.claude-3-haiku-20240307)")
	llmRegionFlag := flag.String("llm-region", "", "Region for region-based providers (e.g., us-east-1 for Bedrock)")
	configPathFlag := flag.String("config", "", "Path to JSON configuration file")
	flag.Parse()

	// Load configuration
	configPath := *configPathFlag
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load configuration: %v", err)
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
		log.Fatal("Gmail credentials file is required. Provide it via --credentials or config file.")
	}

	if _, err := os.Stat(credPath); err != nil {
		log.Fatalf("Credentials file not found at %s. Download client credentials from Google Cloud Console and place it there.", credPath)
	}

	// Initialize Gmail service
	ctx := context.Background()
	service, err := auth.NewGmailService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.compose",
		"https://www.googleapis.com/auth/calendar.events",
	)
	if err != nil {
		log.Fatalf("Could not initialize Gmail service: %v", err)
	}

	// Create Gmail client
	gmailClient := gmail.NewClient(service)

	// Initialize Calendar service (Calendar-only RSVP)
	calSvc, err := auth.NewCalendarService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/calendar.events",
	)
	if err != nil {
		log.Printf("Warning: could not initialize Calendar (RSVP via API disabled): %v", err)
	}
	var calClient *calendar.Client
	if calSvc != nil {
		calClient = calendar.NewClient(calSvc)
	}

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

	// Optional: open cache store for AI summaries (and future features)
	var store *cache.Store
	if cfg.AISummaryCacheEnabled {
		email, _ := gmailClient.ActiveAccountEmail(ctx)
		baseDir := filepath.Join(os.Getenv("HOME"), ".config", "gmail-tui", "cache")
		if cfg.AISummaryCachePath != "" {
			baseDir = cfg.AISummaryCachePath
		}
		dbPath := baseDir
		if ext := filepath.Ext(baseDir); ext == "" || ext == "." {
			safe := strings.ToLower(strings.TrimSpace(email))
			safe = strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_").Replace(safe)
			if safe == "" {
				safe = "default"
			}
			dbPath = filepath.Join(baseDir, safe+".sqlite3")
		}
		if st, err := cache.Open(ctx, dbPath); err == nil {
			store = st
		} else {
			log.Printf("Warning: could not open cache store: %v", err)
		}
	}

	// Create and run TUI
	app := tui.NewApp(gmailClient, calClient, llmProvider, cfg)
	// Inject store if available (setter preferred; using unexported field would break encapsulation)
	if store != nil {
		// Provide a small helper in TUI to register the store without breaking module boundaries
		app.RegisterCacheStore(store)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
