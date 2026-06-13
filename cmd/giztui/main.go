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
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/llm"
	"github.com/ajramos/giztui/internal/services"
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
	migrateConfigFlag := flag.Bool("migrate-config", false, "Add missing default options to the config file and exit")

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
		fmt.Fprintf(os.Stderr, "  --version\n        %s\n", "Show version information and exit")
		fmt.Fprintf(os.Stderr, "  --migrate-config\n        %s\n\n", "Add missing default options to the config file and exit")
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

	// Handle config migration (add missing default keys to the config file, then exit)
	if *migrateConfigFlag {
		added, removed, backup, mErr := config.MigrateConfigFile(configPath)
		if mErr != nil {
			fmt.Fprintf(os.Stderr, "Config migrate failed: %v\n", mErr)
			os.Exit(1)
		}
		if len(added) == 0 && len(removed) == 0 {
			fmt.Println("Config is already up to date.")
			return
		}
		fmt.Printf("Updated %s (backup: %s): +%d added, -%d removed\n", configPath, backup, len(added), len(removed))
		for _, k := range added {
			fmt.Printf("  + %s\n", k)
		}
		for _, k := range removed {
			fmt.Printf("  - %s\n", k)
		}
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Initialize Gmail service using multi-account logic
	ctx := context.Background()

	var credPath, tokenPath string

	// Try multi-account validation with file logging if available
	logger := createFileLogger()

	// Create AccountService (will use logger if available, or default logger if not)
	accountServiceLogger := logger
	if accountServiceLogger == nil {
		accountServiceLogger = log.New(os.Stderr, "", log.LstdFlags)
	}
	accountService := services.NewAccountService(cfg, accountServiceLogger)

	if logger != nil {
		logger.Printf("🔍 Starting account validation and selection...")
		accounts, err := accountService.ListAccounts(ctx)
		if err != nil {
			logger.Printf("⚠️  Failed to list accounts: %v", err)
		} else {
			logger.Printf("📋 Found %d configured accounts", len(accounts))

			// Check for multiple active accounts (warn if found)
			activeCount := 0
			var activeAccounts []string
			for _, account := range accounts {
				if account.IsActive {
					activeCount++
					activeAccounts = append(activeAccounts, fmt.Sprintf("%s (%s)", account.ID, account.DisplayName))
				}
			}

			if activeCount > 1 {
				logger.Printf("⚠️  Multiple active accounts detected (%d): %v", activeCount, activeAccounts)
				logger.Printf("⚠️  Will use first valid active account found")
			} else if activeCount == 1 {
				logger.Printf("🎯 Single active account found: %s", activeAccounts[0])
			} else {
				logger.Printf("⚠️  No active accounts found, will fall back to legacy configuration")
			}

			// First, validate ALL accounts for UI status (don't break early)
			logger.Printf("🔍 Validating all accounts for UI status...")
			for _, account := range accounts {
				logger.Printf("🔍 Validating account: %s (%s)", account.ID, account.DisplayName)
				result, err := accountService.ValidateAccount(ctx, account.ID)
				if err != nil {
					logger.Printf("❌ Account validation failed for %s: %v", account.ID, err)
					printUserFriendlyError(err, account.ID)
				} else if result.IsValid {
					logger.Printf("✅ Account validation successful for %s (%s) - Email: %s", account.ID, account.DisplayName, result.Email)
				} else {
					logger.Printf("❌ Account validation failed for %s: %s", account.ID, result.ErrorMsg)
					printUserFriendlyValidationError(result.ErrorMsg, account.ID)
				}
			}

			// Then, find first active and valid account for startup
			logger.Printf("🔍 Finding first valid active account for startup...")
			var selectedAccount *services.Account
			for _, account := range accounts {
				if !account.IsActive {
					logger.Printf("⏭️  Skipping inactive account: %s (%s)", account.ID, account.DisplayName)
					continue
				}

				// Get fresh validation result (already validated above)
				result, err := accountService.ValidateAccount(ctx, account.ID)
				if err != nil {
					logger.Printf("❌ Account validation failed for %s: %v", account.ID, err)
					printUserFriendlyError(err, account.ID)
					continue
				}

				if result.IsValid {
					logger.Printf("✅ Using account for startup: %s (%s) - Email: %s", account.ID, account.DisplayName, result.Email)
					selectedAccount = account
					selectedAccount.Email = result.Email // Update account with validated email
					credPath = expandPath(account.CredPath)
					tokenPath = expandPath(account.TokenPath)
					break
				} else {
					logger.Printf("❌ Account validation failed for %s: %s", account.ID, result.ErrorMsg)
					printUserFriendlyValidationError(result.ErrorMsg, account.ID)
				}
			}

			// Log final selection result
			if selectedAccount != nil {
				logger.Printf("🎉 Selected account: %s (%s) - Email: %s", selectedAccount.ID, selectedAccount.DisplayName, selectedAccount.Email)
			} else {
				logger.Printf("❌ No valid active account found, falling back to legacy configuration")
			}
		}

		if credPath != "" {
			logger.Printf("🚀 Initializing Gmail service with validated account (creds: %s, token: %s)", credPath, tokenPath)
		}
	}

	// Graceful multi-level credential fallback if multi-account validation failed
	if credPath == "" {
		if logger != nil {
			logger.Printf("🔄 Starting graceful credential fallback sequence...")
		}

		var fallbackMethod string
		var attemptNumber = 1

		// Level 1: Try CLI flag credentials (highest priority)
		if *credPathFlag != "" {
			if logger != nil {
				logger.Printf("🎯 Attempt %d: Trying CLI flag credentials: %s", attemptNumber, *credPathFlag)
			}
			attemptNumber++

			testCredPath := *credPathFlag
			testTokenPath := getTokenPath("", cfg.Token)

			if logger != nil {
				logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)
			}

			if testCredPath != "" {
				if _, err := os.Stat(testCredPath); err == nil {
					credPath = testCredPath
					tokenPath = testTokenPath
					fallbackMethod = "CLI flag"
					if logger != nil {
						logger.Printf("✅ CLI flag credentials found and validated")
					}
				} else {
					if logger != nil {
						logger.Printf("❌ CLI flag credentials not found at %s", testCredPath)
					}
				}
			}
		}

		// Level 2: Try config file credentials (if CLI didn't work and config has credentials)
		if credPath == "" && cfg.Credentials != "" {
			if logger != nil {
				logger.Printf("🎯 Attempt %d: Trying config file credentials: %s", attemptNumber, cfg.Credentials)
			}
			attemptNumber++

			testCredPath := expandPath(cfg.Credentials)
			testTokenPath := getTokenPath("", cfg.Token)

			if logger != nil {
				logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)
			}

			if _, err := os.Stat(testCredPath); err == nil {
				credPath = testCredPath
				tokenPath = testTokenPath
				fallbackMethod = "config file"
				if logger != nil {
					logger.Printf("✅ Config file credentials found and validated")
				}
			} else {
				if logger != nil {
					logger.Printf("❌ Config file credentials not found at %s", testCredPath)
				}
			}
		}

		// Level 3: Try hardcoded default credentials (final fallback)
		if credPath == "" {
			if logger != nil {
				if cfg.Credentials != "" {
					logger.Printf("🎯 Attempt %d: Config credentials failed, trying hardcoded defaults as final fallback", attemptNumber)
				} else {
					logger.Printf("🎯 Attempt %d: No config credentials (disabled with prefix), trying hardcoded defaults", attemptNumber)
				}
			}

			defaultCredPath, _ := config.DefaultCredentialPaths()
			testCredPath := defaultCredPath
			testTokenPath := getTokenPath("", "")

			if logger != nil {
				logger.Printf("📍 Resolved default paths - creds: %s, token: %s", testCredPath, testTokenPath)
			}

			if testCredPath != "" {
				if _, err := os.Stat(testCredPath); err == nil {
					credPath = testCredPath
					tokenPath = testTokenPath
					fallbackMethod = "hardcoded defaults"
					if logger != nil {
						logger.Printf("✅ Hardcoded default credentials found and validated")
					}
				} else {
					if logger != nil {
						logger.Printf("❌ Hardcoded default credentials not found at %s", testCredPath)
					}
				}
			}
		}

		// Final validation - if still no valid credentials found, exit fatally
		if credPath == "" {
			if logger != nil {
				logger.Printf("❌ All credential fallback methods exhausted")
				logger.Printf("💡 Tried CLI flag, config file, and hardcoded defaults")
				logger.Printf("💡 Please ensure at least one credential file exists and is accessible")
			}
			log.Fatal("Gmail credentials file is required. No valid credentials found in CLI flag, config file, or default location.")
		}

		// Success - log which method worked
		if logger != nil {
			logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
		}
	}

	service, err := auth.NewGmailService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.compose",
		"https://www.googleapis.com/auth/calendar.events",
	)
	if err != nil {
		if logger != nil {
			logger.Printf("❌ Could not initialize Gmail service: %v", err)
			logger.Printf("🔄 Will start in limited mode - account picker will show validation status")
		}

		// Continue in limited mode - create a nil client
		// The account service will still work and show validation status
		fmt.Fprintf(os.Stderr, "⚠️  Gmail service initialization failed - starting in limited mode\n")
		fmt.Fprintf(os.Stderr, "💡 Use Ctrl+A to open account picker and check account status\n")

		// Create a dummy client that will be replaced when user fixes accounts
		service = nil
	}

	// Create Gmail client (might be nil in limited mode)
	var gmailClient *gmail.Client
	if service != nil {
		gmailClient = gmail.NewClient(service)
	} else {
		// Limited mode - no Gmail client available
		gmailClient = nil
		if logger != nil {
			logger.Printf("⚠️  Running in limited mode - Gmail client is not available")
		}
	}

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

	// Create and run TUI (database management is now handled internally)
	// Pass the logger and accountService to avoid duplicate initialization
	app := tui.NewApp(gmailClient, calClient, llmProvider, cfg, logger, accountService)
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

// createFileLogger creates a logger that writes to the same log file as the TUI
func createFileLogger() *log.Logger {
	logDir := config.DefaultLogDir()
	if logDir == "" {
		return nil
	}

	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil
	}

	logFile := filepath.Join(logDir, "giztui.log")
	// Validate path to prevent directory traversal
	cleanPath := filepath.Clean(logFile)
	if strings.Contains(cleanPath, "..") {
		return nil
	}

	f, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil
	}

	// Note: We don't close the file here since main() will exit anyway
	return log.New(f, "[giztui] ", log.LstdFlags|log.Lmicroseconds)
}

// printUserFriendlyError provides helpful error messages for common Gmail API issues
func printUserFriendlyError(err error, accountID string) {
	errMsg := strings.ToLower(err.Error())

	// Check for common Gmail API issues
	if strings.Contains(errMsg, "access_not_configured") ||
		strings.Contains(errMsg, "api not enabled") ||
		strings.Contains(errMsg, "gmail api has not been used") {

		fmt.Printf("\n🚨 Gmail API Issue for account '%s':\n", accountID)
		fmt.Println("   The Gmail API is not enabled in your Google Cloud Console.")
		fmt.Println("")
		fmt.Println("📋 To fix this:")
		fmt.Println("   1. Go to: https://console.cloud.google.com/apis/library")
		fmt.Println("   2. Search for 'Gmail API'")
		fmt.Println("   3. Click 'ENABLE'")
		fmt.Println("   4. Restart GizTUI")
		fmt.Println("")

	} else if strings.Contains(errMsg, "credentials") || strings.Contains(errMsg, "token") {

		fmt.Printf("\n🚨 Credentials Issue for account '%s':\n", accountID)
		fmt.Println("   There's an issue with your OAuth2 credentials or token.")
		fmt.Println("")
		fmt.Println("📋 To fix this:")
		fmt.Println("   1. Check that credentials.json exists and is valid")
		fmt.Println("   2. Delete the token file to force re-authentication")
		fmt.Println("   3. Restart GizTUI and follow the OAuth flow")
		fmt.Println("")

	} else if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden") {

		fmt.Printf("\n🚨 Permission Issue for account '%s':\n", accountID)
		fmt.Println("   Access is forbidden. This usually means:")
		fmt.Println("   - Gmail API is not enabled, or")
		fmt.Println("   - Your credentials don't have the right permissions")
		fmt.Println("")
		fmt.Println("📋 To fix this:")
		fmt.Println("   1. Enable Gmail API: https://console.cloud.google.com/apis/library/gmail.googleapis.com")
		fmt.Println("   2. Ensure your OAuth2 app has the correct scopes")
		fmt.Println("   3. Try re-creating your credentials")
		fmt.Println("")

	} else {

		fmt.Printf("\n🚨 Account Issue for '%s': %v\n", accountID, err)
		fmt.Println("📋 General troubleshooting:")
		fmt.Println("   1. Check the detailed logs above")
		fmt.Println("   2. Verify Gmail API is enabled: https://console.cloud.google.com/apis/library/gmail.googleapis.com")
		fmt.Println("   3. Ensure credentials.json is valid")
		fmt.Println("")
	}
}

// printUserFriendlyValidationError provides helpful messages for validation result errors
func printUserFriendlyValidationError(errorMsg, accountID string) {
	errMsg := strings.ToLower(errorMsg)

	if strings.Contains(errMsg, "failed to connect") {

		fmt.Printf("\n🚨 Connection Issue for account '%s':\n", accountID)
		fmt.Println("   Unable to connect to Gmail API.")
		fmt.Println("")
		fmt.Println("📋 Check these items:")
		fmt.Println("   1. Gmail API is enabled: https://console.cloud.google.com/apis/library/gmail.googleapis.com")
		fmt.Println("   2. Internet connection is working")
		fmt.Println("   3. Credentials are valid and not expired")
		fmt.Println("")

	} else {

		fmt.Printf("\n🚨 Validation Issue for '%s': %s\n", accountID, errorMsg)
		fmt.Println("📋 See the detailed logs above for more information.")
		fmt.Println("")
	}
}
