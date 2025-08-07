package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
)

func main() {
	fmt.Println("🎨 Gmail TUI Theme System Demo")
	fmt.Println("==============================\n")

	// Create theme loader
	skinsDir := "skins"
	loader := config.NewThemeLoader(skinsDir)

	// Create default theme if it doesn't exist
	if err := loader.CreateDefaultTheme(); err != nil {
		log.Printf("Warning: Could not create default theme: %v", err)
	}

	// List available themes
	themes, err := loader.ListAvailableThemes()
	if err != nil {
		log.Fatalf("Failed to list themes: %v", err)
	}

	fmt.Printf("📁 Available themes (%s):\n", skinsDir)
	for _, theme := range themes {
		fmt.Printf("  • %s\n", theme)
	}
	fmt.Println()

	// Load and preview each theme
	for _, themeFile := range themes {
		fmt.Printf("🎨 Loading theme: %s\n", themeFile)
		fmt.Println("─" + strings.Repeat("─", len(themeFile)+15))

		theme, err := loader.LoadThemeFromFile(themeFile)
		if err != nil {
			log.Printf("Failed to load theme %s: %v", themeFile, err)
			continue
		}

		// Validate theme
		if err := loader.ValidateTheme(theme); err != nil {
			log.Printf("Theme %s validation failed: %v", themeFile, err)
			continue
		}

		// Show preview
		preview := config.GetThemePreview(theme)
		fmt.Println(preview)

		// Show color examples
		fmt.Println("📧 Email State Examples:")
		fmt.Printf("  🔵 Unread email: %s\n", theme.Email.UnreadColor)
		fmt.Printf("  ⚪ Read email: %s\n", theme.Email.ReadColor)
		fmt.Printf("  🔴 Important email: %s\n", theme.Email.ImportantColor)
		fmt.Printf("  🟢 Sent email: %s\n", theme.Email.SentColor)
		fmt.Printf("  🟡 Draft email: %s\n", theme.Email.DraftColor)
		fmt.Println()

		// Show UI colors
		fmt.Println("🎨 UI Colors:")
		fmt.Printf("  📝 Text: %s\n", theme.Body.FgColor)
		fmt.Printf("  🖼️  Background: %s\n", theme.Body.BgColor)
		fmt.Printf("  🔲 Border: %s\n", theme.Frame.Border.FgColor)
		fmt.Printf("  ✨ Focus: %s\n", theme.Frame.Border.FocusColor)
		fmt.Println()
	}

	// Create a custom theme example
	fmt.Println("🚀 Creating Custom Theme Example")
	fmt.Println("────────────────────────────────")

	customTheme := &config.ColorsConfig{
		Body: config.BodyColors{
			FgColor:   config.NewColor("#2c3e50"),
			BgColor:   config.NewColor("#ecf0f1"),
			LogoColor: config.NewColor("#3498db"),
		},
		Frame: config.FrameColors{
			Border: struct {
				FgColor    config.Color `yaml:"fgColor"`
				FocusColor config.Color `yaml:"focusColor"`
			}{
				FgColor:    config.NewColor("#bdc3c7"),
				FocusColor: config.NewColor("#3498db"),
			},
			Title: struct {
				FgColor        config.Color `yaml:"fgColor"`
				BgColor        config.Color `yaml:"bgColor"`
				HighlightColor config.Color `yaml:"highlightColor"`
				CounterColor   config.Color `yaml:"counterColor"`
				FilterColor    config.Color `yaml:"filterColor"`
			}{
				FgColor:        config.NewColor("#2c3e50"),
				BgColor:        config.NewColor("#ecf0f1"),
				HighlightColor: config.NewColor("#f39c12"),
				CounterColor:   config.NewColor("#27ae60"),
				FilterColor:    config.NewColor("#3498db"),
			},
		},
		Table: config.TableColors{
			FgColor:       config.NewColor("#2c3e50"),
			BgColor:       config.NewColor("#ecf0f1"),
			HeaderFgColor: config.NewColor("#27ae60"),
			HeaderBgColor: config.NewColor("#ecf0f1"),
		},
		Email: config.EmailColors{
			UnreadColor:    config.NewColor("#e67e22"),
			ReadColor:      config.NewColor("#7f8c8d"),
			ImportantColor: config.NewColor("#e74c3c"),
			SentColor:      config.NewColor("#27ae60"),
			DraftColor:     config.NewColor("#f39c12"),
		},
	}

	// Save custom theme
	customThemeFile := "custom-example.yaml"
	if err := loader.SaveThemeToFile(customTheme, customThemeFile); err != nil {
		log.Printf("Failed to save custom theme: %v", err)
	} else {
		fmt.Printf("✅ Custom theme saved to: %s\n", filepath.Join(skinsDir, customThemeFile))
	}

	// Show custom theme preview
	fmt.Println("\n🎨 Custom Theme Preview:")
	preview := config.GetThemePreview(customTheme)
	fmt.Println(preview)

	fmt.Println("✨ Theme system demo completed!")
	fmt.Println("\n💡 Next steps:")
	fmt.Println("  1. Run the main application: make run")
	fmt.Println("  2. Modify theme files in the skins/ directory")
	fmt.Println("  3. Create your own custom themes")
	fmt.Println("  4. Check the documentation: docs/COLORS.md")
}
