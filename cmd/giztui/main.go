package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/calendar"
	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/llm"
	"github.com/ajramos/giztui/internal/tui"
	"github.com/ajramos/giztui/internal/version"
	"github.com/ajramos/giztui/pkg/auth"
)

func main() {
	// Essential command line flags only (GNU-style double dashes)
	configPathFlag := flag.String("config", "", "Path to JSON configuration file (default: ~/.config/giztui/config.json)")
	credPathFlag := flag.String("credentials", "", "Path to OAuth client credentials JSON (default: ~/.config/giztui/credentials.json)")
	setupFlag := flag.Bool("setup", false, "Run interactive setup wizard")
	versionFlag := flag.Bool("version", false, "Show version information and exit")

	// Override flag usage text to show clean, simple usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", version.GetVersionString())
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s                        # Run with default configuration\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --setup                # Run interactive setup wizard\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --version              # Show version information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --config custom.json   # Use custom configuration\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --config string\n        %s\n", "Path to JSON configuration file (default: ~/.config/giztui/config.json)")
		fmt.Fprintf(os.Stderr, "  --credentials string\n        %s\n", "Path to OAuth client credentials JSON (default: ~/.config/giztui/credentials.json)")
		fmt.Fprintf(os.Stderr, "  --setup\n        %s\n", "Run interactive setup wizard")
		fmt.Fprintf(os.Stderr, "  --version\n        %s\n\n", "Show version information and exit")
		fmt.Fprintf(os.Stderr, "Environment Variables:\n")
		fmt.Fprintf(os.Stderr, "  GMAIL_TUI_CONFIG      Override default config file path\n")
		fmt.Fprintf(os.Stderr, "  GMAIL_TUI_CREDENTIALS Override default credentials file path\n")
		fmt.Fprintf(os.Stderr, "  GMAIL_TUI_TOKEN       Override default token file path\n\n")
		fmt.Fprintf(os.Stderr, "For all other settings (LLM, timeouts, etc.), edit the config file.\n")
	}

	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Println(version.GetDetailedVersionString())
		return
	}

	// Handle setup mode
	if *setupFlag {
		runSetupWizard()
		return
	}

	// Load configuration with smart defaults and environment variable support
	configPath := getConfigPath(*configPathFlag)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Determine credential and token paths with smart defaults
	credPath := getCredentialsPath(*credPathFlag, cfg.Credentials)
	tokenPath := getTokenPath("", cfg.Token)

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
	var calClient *calendar.Client
	if calSvc, err := auth.NewCalendarService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/calendar.events",
	); err == nil && calSvc != nil {
		calClient = calendar.NewClient(calSvc)
	} else if err != nil {
		log.Printf("Warning: could not initialize Calendar service: %v", err)
	}

	// All LLM configuration is now handled via config file only

	// Initialize LLM provider
	var llmProvider llm.Provider
	if cfg.LLM.Enabled {
		model := cfg.LLM.Model
		timeout := cfg.GetLLMTimeout()

		if model != "" {
			providerName := cfg.LLM.Provider
			if providerName == "" {
				providerName = "ollama"
			}

			arg := cfg.LLM.Endpoint

			if providerName == "bedrock" {
				region := cfg.LLM.Region
				if region == "" {
					if env := os.Getenv("AWS_REGION"); env != "" {
						region = env
					}
				}
				arg = region
			}
			var err error
			llmProvider, err = llm.NewProviderFromConfig(providerName, arg, model, timeout, cfg.LLM.APIKey)
			if err != nil {
				log.Printf("Warning: could not initialize LLM provider (%s): %v", providerName, err)
			}
		}
	}

	// Optional: open database store for AI summaries and prompts
	var store *db.Store
	if cfg.LLM.CacheEnabled {
		email, _ := gmailClient.ActiveAccountEmail(ctx)
		baseDir := config.DefaultCacheDir()
		if cfg.LLM.CachePath != "" {
			baseDir = cfg.LLM.CachePath
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
		if st, err := db.Open(ctx, dbPath); err == nil {
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
		app.RegisterDBStore(store)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// getConfigPath returns the configuration file path using the following priority:
// 1. CLI flag
// 2. Environment variable GMAIL_TUI_CONFIG
// 3. Default path ~/.config/giztui/config.json
func getConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envPath := os.Getenv("GMAIL_TUI_CONFIG"); envPath != "" {
		return expandPath(envPath)
	}

	return config.DefaultConfigPath()
}

// getCredentialsPath returns the credentials file path using the following priority:
// 1. CLI flag
// 2. Environment variable GMAIL_TUI_CREDENTIALS
// 3. Config file setting
// 4. Default path ~/.config/giztui/credentials.json
func getCredentialsPath(flagValue, configValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envPath := os.Getenv("GMAIL_TUI_CREDENTIALS"); envPath != "" {
		return expandPath(envPath)
	}

	if configValue != "" {
		return expandPath(configValue)
	}

	credPath, _ := config.DefaultCredentialPaths()
	return credPath
}

// getTokenPath returns the token file path using the following priority:
// 1. CLI flag
// 2. Environment variable GMAIL_TUI_TOKEN
// 3. Config file setting
// 4. Default path ~/.config/giztui/token.json
func getTokenPath(flagValue, configValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envPath := os.Getenv("GMAIL_TUI_TOKEN"); envPath != "" {
		return expandPath(envPath)
	}

	if configValue != "" {
		return expandPath(configValue)
	}

	_, tokenPath := config.DefaultCredentialPaths()
	return tokenPath
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	return filepath.Join(home, path[2:])
}

// runSetupWizard runs an interactive setup wizard to help users configure Gmail TUI
func runSetupWizard() {
	fmt.Println("📧 Gmail TUI Setup Wizard")
	fmt.Println("=======================")
	fmt.Println()

	// Check if default config already exists
	defaultConfigPath := config.DefaultConfigPath()
	credPath, tokenPath := config.DefaultCredentialPaths()

	if _, err := os.Stat(defaultConfigPath); err == nil {
		fmt.Printf("✅ Configuration file already exists: %s\n", defaultConfigPath)
	} else {
		fmt.Printf("📝 Will create configuration file: %s\n", defaultConfigPath)
	}

	if _, err := os.Stat(credPath); err == nil {
		fmt.Printf("✅ Credentials file found: %s\n", credPath)
	} else {
		fmt.Printf("⚠️  Credentials file missing: %s\n", credPath)
		fmt.Println()
		fmt.Println("📋 To set up Gmail API credentials:")
		fmt.Println("1. Go to https://console.cloud.google.com/")
		fmt.Println("2. Create a new project or select existing one")
		fmt.Println("3. Enable Gmail API")
		fmt.Println("4. Create OAuth 2.0 credentials (Desktop application)")
		fmt.Println("5. Download the JSON file and save it as:")
		fmt.Printf("   %s\n", credPath)
		fmt.Println()
	}

	if _, err := os.Stat(tokenPath); err == nil {
		fmt.Printf("✅ Token file exists: %s\n", tokenPath)
	} else {
		fmt.Printf("🔐 Token will be created on first login: %s\n", tokenPath)
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		fmt.Println()
		fmt.Print("📄 Create default configuration file? [Y/n]: ")

		var response string
		_, _ = fmt.Scanln(&response) // User input - error not actionable

		if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			cfg := config.DefaultConfig()
			if err := cfg.SaveConfig(defaultConfigPath); err != nil {
				fmt.Printf("❌ Failed to create config file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Created configuration file: %s\n", defaultConfigPath)
		}
	}

	fmt.Println()
	fmt.Println("🚀 Setup complete! You can now run:")
	fmt.Printf("   %s\n", os.Args[0])
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("• Edit the config file to customize LLM settings")
	fmt.Println("• Use environment variables for different profiles")
	fmt.Println("• Run with -h to see all options")
}
