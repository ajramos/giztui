package config

import (
	"fmt"

	"github.com/derailed/tcell/v2"
)

// Color represents a color in the application
type Color string

const (
	// DefaultColor represents a default color
	DefaultColor Color = "default"

	// TransparentColor represents the terminal bg color
	TransparentColor Color = "-"
)

// Colors tracks multiple colors
type Colors []Color

// Colors converts series string colors to colors
func (c Colors) Colors() []tcell.Color {
	cc := make([]tcell.Color, 0, len(c))
	for _, color := range c {
		cc = append(cc, color.Color())
	}
	return cc
}

// NewColor returns a new color
func NewColor(c string) Color {
	return Color(c)
}

// String returns color as string
func (c Color) String() string {
	if c.isHex() {
		return string(c)
	}
	if c == DefaultColor {
		return "-"
	}
	col := c.Color().TrueColor().Hex()
	if col < 0 {
		return "-"
	}
	return fmt.Sprintf("#%06x", col)
}

func (c Color) isHex() bool {
	return len(c) == 7 && c[0] == '#'
}

// Color returns a view color
func (c Color) Color() tcell.Color {
	if c == DefaultColor {
		return tcell.ColorDefault
	}
	return tcell.GetColor(string(c)).TrueColor()
}

// EmailColors defines colors for email states
type EmailColors struct {
	UnreadColor    Color `yaml:"unreadColor"`
	ReadColor      Color `yaml:"readColor"`
	ImportantColor Color `yaml:"importantColor"`
	SentColor      Color `yaml:"sentColor"`
	DraftColor     Color `yaml:"draftColor"`
}

// FrameColors defines colors for UI frame elements
type FrameColors struct {
	Border struct {
		FgColor    Color `yaml:"fgColor"`
		FocusColor Color `yaml:"focusColor"`
	} `yaml:"border"`
	Title struct {
		FgColor        Color `yaml:"fgColor"`
		BgColor        Color `yaml:"bgColor"`
		HighlightColor Color `yaml:"highlightColor"`
		CounterColor   Color `yaml:"counterColor"`
		FilterColor    Color `yaml:"filterColor"`
	} `yaml:"title"`
}

// TableColors defines colors for table elements
type TableColors struct {
	FgColor       Color `yaml:"fgColor"`
	BgColor       Color `yaml:"bgColor"`
	HeaderFgColor Color `yaml:"headerFgColor"`
	HeaderBgColor Color `yaml:"headerBgColor"`
}

// BodyColors defines colors for body elements
type BodyColors struct {
	FgColor   Color `yaml:"fgColor"`
	BgColor   Color `yaml:"bgColor"`
	LogoColor Color `yaml:"logoColor"`
}

// UIColors defines colors for UI components (previously hardcoded)
type UIColors struct {
	// Panel and text colors
	TitleColor  Color `yaml:"titleColor"`  // Panel titles
	FooterColor Color `yaml:"footerColor"` // Footer/instruction text
	HintColor   Color `yaml:"hintColor"`   // Hint text color

	// Selection colors (cursor/navigation)
	SelectionBgColor Color `yaml:"selectionBgColor"` // Cursor selection background
	SelectionFgColor Color `yaml:"selectionFgColor"` // Cursor selection text

	// Bulk selection colors (checked items)
	BulkSelectionBgColor Color `yaml:"bulkSelectionBgColor"` // Bulk selection background
	BulkSelectionFgColor Color `yaml:"bulkSelectionFgColor"` // Bulk selection text

	// Status message colors
	ErrorColor   Color `yaml:"errorColor"`   // Error messages
	SuccessColor Color `yaml:"successColor"` // Success messages
	WarningColor Color `yaml:"warningColor"` // Warning messages
	InfoColor    Color `yaml:"infoColor"`    // Info messages

	// Input field colors
	InputBgColor Color `yaml:"inputBgColor"` // Input field background
	InputFgColor Color `yaml:"inputFgColor"` // Input field text
	LabelColor   Color `yaml:"labelColor"`   // Input field labels

	// Status bar colors
	StatusBarBgColor Color `yaml:"statusBarBgColor"` // Status bar background
	StatusBarFgColor Color `yaml:"statusBarFgColor"` // Status bar text color
}

// TagColors defines colors for text markup tags
type TagColors struct {
	Title     Color `yaml:"title"`     // [title]text[/title] - replaces [yellow]
	Header    Color `yaml:"header"`    // [header]text[/header] - replaces [green]
	Emphasis  Color `yaml:"emphasis"`  // [emphasis]text[/emphasis] - replaces [orange]
	Secondary Color `yaml:"secondary"` // [secondary]text[/secondary] - replaces [dim]/[gray]
	Link      Color `yaml:"link"`      // [link]text[/link] - replaces [blue]
	Code      Color `yaml:"code"`      // [code]text[/code] - replaces [purple]
}

// StatusColors defines colors for status messages
type StatusColors struct {
	Error    Color `yaml:"error"`    // Error messages - replaces tcell.ColorRed
	Success  Color `yaml:"success"`  // Success messages - replaces tcell.ColorGreen
	Warning  Color `yaml:"warning"`  // Warning messages - replaces tcell.ColorYellow
	Info     Color `yaml:"info"`     // Info messages - replaces tcell.ColorBlue
	Progress Color `yaml:"progress"` // Progress indicators - replaces tcell.ColorOrange
}

// ComponentColors defines colors for specific UI components
type ComponentColors struct {
	AI           ComponentColorSet `yaml:"ai"`
	Slack        ComponentColorSet `yaml:"slack"`
	Obsidian     ComponentColorSet `yaml:"obsidian"`
	Links        ComponentColorSet `yaml:"links"`
	Stats        ComponentColorSet `yaml:"stats"`
	Prompts      ComponentColorSet `yaml:"prompts"`
	Labels       ComponentColorSet `yaml:"labels"`       // Label management UI colors
	Search       ComponentColorSet `yaml:"search"`       // Search interface colors
	Attachments  ComponentColorSet `yaml:"attachments"`  // Attachment picker colors
	SavedQueries ComponentColorSet `yaml:"saved_queries"` // Saved queries picker colors
	Compose      ComponentColorSet `yaml:"compose"`      // Email composition UI colors
}

// ComponentColorSet defines a complete color set for a UI component
type ComponentColorSet struct {
	Border     Color `yaml:"border"`     // Component border color
	Title      Color `yaml:"title"`      // Component title color
	Background Color `yaml:"background"` // Component background color
	Text       Color `yaml:"text"`       // Component text color
	Accent     Color `yaml:"accent"`     // Component accent/highlight color
}

// FoundationColors defines base colors for all components
type FoundationColors struct {
	Background Color `yaml:"background"` // Primary background color
	Foreground Color `yaml:"foreground"` // Primary text color
	Border     Color `yaml:"border"`     // Default border color
	Focus      Color `yaml:"focus"`      // Focus highlight color
}

// SemanticColors defines meaning-based colors
type SemanticColors struct {
	Primary   Color `yaml:"primary"`   // Main actions, titles
	Secondary Color `yaml:"secondary"` // Supporting elements
	Accent    Color `yaml:"accent"`    // Highlights, links
	Success   Color `yaml:"success"`   // Success states
	Warning   Color `yaml:"warning"`   // Warning states
	Error     Color `yaml:"error"`     // Error states
	Info      Color `yaml:"info"`      // Info states
}

// InteractionColors defines user interaction state colors
type InteractionColors struct {
	Selection struct {
		Cursor struct {
			Bg Color `yaml:"bg"` // Single item cursor background
			Fg Color `yaml:"fg"` // Single item cursor text
		} `yaml:"cursor"`
		Bulk struct {
			Bg Color `yaml:"bg"` // Multi-item selection background
			Fg Color `yaml:"fg"` // Multi-item selection text
		} `yaml:"bulk"`
	} `yaml:"selection"`
	Input struct {
		Bg    Color `yaml:"bg"`    // Input field background
		Fg    Color `yaml:"fg"`    // Input field text
		Label Color `yaml:"label"` // Input field labels
	} `yaml:"input"`
	StatusBar struct {
		Bg Color `yaml:"bg"` // Status bar background
		Fg Color `yaml:"fg"` // Status bar text
	} `yaml:"statusBar"`
}

// ComponentColorOverrides defines component-specific color overrides
// Only specify colors that should override semantic/foundation defaults
type ComponentColorOverrides struct {
	AI           ComponentOverrideSet `yaml:"ai,omitempty"`
	Slack        ComponentOverrideSet `yaml:"slack,omitempty"`
	Obsidian     ComponentOverrideSet `yaml:"obsidian,omitempty"`
	Links        ComponentOverrideSet `yaml:"links,omitempty"`
	Stats        ComponentOverrideSet `yaml:"stats,omitempty"`
	Prompts      ComponentOverrideSet `yaml:"prompts,omitempty"`
	Labels       ComponentOverrideSet `yaml:"labels,omitempty"`       // Label management overrides
	Search       ComponentOverrideSet `yaml:"search,omitempty"`       // Search interface overrides
	Attachments  ComponentOverrideSet `yaml:"attachments,omitempty"`  // Attachment picker overrides
	SavedQueries ComponentOverrideSet `yaml:"saved_queries,omitempty"` // Saved queries picker overrides
	Themes       ComponentOverrideSet `yaml:"themes,omitempty"`       // Theme picker overrides
	Compose      ComponentOverrideSet `yaml:"compose,omitempty"`      // Email composition overrides
}

// ComponentOverrideSet defines optional color overrides for a specific component
type ComponentOverrideSet struct {
	Primary    Color `yaml:"primary,omitempty"`    // Override semantic.primary
	Secondary  Color `yaml:"secondary,omitempty"`  // Override semantic.secondary
	Accent     Color `yaml:"accent,omitempty"`     // Override semantic.accent
	Background Color `yaml:"background,omitempty"` // Override foundation.background
	Foreground Color `yaml:"foreground,omitempty"` // Override foundation.foreground
	Border     Color `yaml:"border,omitempty"`     // Override foundation.border
}

// ColorsConfig defines the complete color configuration
type ColorsConfig struct {
	Name        string `yaml:"name"`        // Theme name (e.g., "Gmail Dark")
	Description string `yaml:"description"` // Theme description
	Version     string `yaml:"version"`     // Theme version

	// New hierarchical structure
	Foundation  FoundationColors         `yaml:"foundation"`  // Base colors for all components
	Semantic    SemanticColors           `yaml:"semantic"`    // Meaning-based colors
	Interaction InteractionColors        `yaml:"interaction"` // User interaction colors
	Overrides   ComponentColorOverrides  `yaml:"overrides"`   // Component-specific overrides

	// Legacy structure (for backward compatibility)
	Body       BodyColors      `yaml:"body"`
	Frame      FrameColors     `yaml:"frame"`
	Table      TableColors     `yaml:"table"`
	Email      EmailColors     `yaml:"email"`
	UI         UIColors        `yaml:"ui"`         // UI component colors (previously hardcoded)
	Tags       TagColors       `yaml:"tags"`       // Color tags for text markup
	Status     StatusColors    `yaml:"status"`     // Status message colors
	Components ComponentColors `yaml:"components"` // Component-specific colors
}

// ColorType represents different types of colors a component might need
type ColorType string

const (
	ColorTypePrimary    ColorType = "primary"
	ColorTypeSecondary  ColorType = "secondary"
	ColorTypeAccent     ColorType = "accent"
	ColorTypeBackground ColorType = "background"
	ColorTypeForeground ColorType = "foreground"
	ColorTypeBorder     ColorType = "border"
	ColorTypeFocus      ColorType = "focus"
	ColorTypeSuccess    ColorType = "success"
	ColorTypeWarning    ColorType = "warning"
	ColorTypeError      ColorType = "error"
	ColorTypeInfo       ColorType = "info"
)

// ComponentType represents different UI components
type ComponentType string

const (
	ComponentTypeGeneral      ComponentType = "general"
	ComponentTypeAI           ComponentType = "ai"
	ComponentTypeSlack        ComponentType = "slack"
	ComponentTypeObsidian     ComponentType = "obsidian"
	ComponentTypeLinks        ComponentType = "links"
	ComponentTypeStats        ComponentType = "stats"
	ComponentTypePrompts      ComponentType = "prompts"
	ComponentTypeSearch       ComponentType = "search"
	ComponentTypeAttachments  ComponentType = "attachments"
	ComponentTypeSavedQueries ComponentType = "saved_queries"
	ComponentTypeLabels       ComponentType = "labels"
	ComponentTypeThemes       ComponentType = "themes"
	ComponentTypeCompose      ComponentType = "compose"
)

// GetComponentColor resolves a color for a specific component and color type
// using the hierarchical resolution: component override → semantic → foundation → fallback
func (c *ColorsConfig) GetComponentColor(component ComponentType, colorType ColorType) Color {
	// Step 1: Check component-specific overrides (new structure)
	if c.hasNewStructure() {
		if override := c.getComponentOverride(component, colorType); override != "" {
			return override
		}
		
		// Step 2: Check semantic colors
		if semantic := c.getSemanticColor(colorType); semantic != "" {
			return semantic
		}
		
		// Step 3: Check foundation colors  
		if foundation := c.getFoundationColor(colorType); foundation != "" {
			return foundation
		}
	}
	
	// Step 4: Fallback to legacy structure for backward compatibility
	if legacy := c.getLegacyColor(component, colorType); legacy != "" {
		return legacy
	}
	
	// Step 5: Final fallback colors
	return c.getFallbackColor(colorType)
}

// hasNewStructure checks if the theme uses the new hierarchical structure
func (c *ColorsConfig) hasNewStructure() bool {
	return c.Foundation.Background != "" || c.Semantic.Primary != ""
}

// getComponentOverride checks for component-specific color overrides
func (c *ColorsConfig) getComponentOverride(component ComponentType, colorType ColorType) Color {
	var override ComponentOverrideSet
	
	switch component {
	case ComponentTypeAI:
		override = c.Overrides.AI
	case ComponentTypeSlack:
		override = c.Overrides.Slack
	case ComponentTypeObsidian:
		override = c.Overrides.Obsidian
	case ComponentTypeLinks:
		override = c.Overrides.Links
	case ComponentTypeStats:
		override = c.Overrides.Stats
	case ComponentTypePrompts:
		override = c.Overrides.Prompts
	case ComponentTypeLabels:
		override = c.Overrides.Labels
	case ComponentTypeSearch:
		override = c.Overrides.Search
	case ComponentTypeAttachments:
		override = c.Overrides.Attachments
	case ComponentTypeSavedQueries:
		override = c.Overrides.SavedQueries
	case ComponentTypeThemes:
		override = c.Overrides.Themes
	case ComponentTypeCompose:
		override = c.Overrides.Compose
	default:
		return ""
	}
	
	switch colorType {
	case ColorTypePrimary:
		return override.Primary
	case ColorTypeSecondary:
		return override.Secondary
	case ColorTypeAccent:
		return override.Accent
	case ColorTypeBackground:
		return override.Background
	case ColorTypeForeground:
		return override.Foreground
	case ColorTypeBorder:
		return override.Border
	}
	
	return ""
}

// getSemanticColor gets semantic colors
func (c *ColorsConfig) getSemanticColor(colorType ColorType) Color {
	switch colorType {
	case ColorTypePrimary:
		return c.Semantic.Primary
	case ColorTypeSecondary:
		return c.Semantic.Secondary
	case ColorTypeAccent:
		return c.Semantic.Accent
	case ColorTypeSuccess:
		return c.Semantic.Success
	case ColorTypeWarning:
		return c.Semantic.Warning
	case ColorTypeError:
		return c.Semantic.Error
	case ColorTypeInfo:
		return c.Semantic.Info
	}
	return ""
}

// getFoundationColor gets foundation colors
func (c *ColorsConfig) getFoundationColor(colorType ColorType) Color {
	switch colorType {
	case ColorTypeBackground:
		return c.Foundation.Background
	case ColorTypeForeground:
		return c.Foundation.Foreground
	case ColorTypeBorder:
		return c.Foundation.Border
	case ColorTypeFocus:
		return c.Foundation.Focus
	}
	return ""
}

// getLegacyColor gets colors from legacy structure for backward compatibility
// Now maps to v2.0 hierarchical structure (foundation → semantic → interaction → overrides)
func (c *ColorsConfig) getLegacyColor(component ComponentType, colorType ColorType) Color {
	// Map to hierarchical v2.0 colors
	switch colorType {
	case ColorTypePrimary:
		// Primary colors from semantic layer
		return c.Semantic.Primary
	case ColorTypeBackground:
		// Background from foundation layer
		return c.Foundation.Background
	case ColorTypeForeground:
		// Foreground from foundation layer
		return c.Foundation.Foreground
	case ColorTypeBorder:
		// Border from foundation layer
		return c.Foundation.Border
	case ColorTypeFocus:
		// Focus from foundation layer
		return c.Foundation.Focus
	case ColorTypeSuccess:
		// Success from semantic layer
		return c.Semantic.Success
	case ColorTypeWarning:
		// Warning from semantic layer
		return c.Semantic.Warning
	case ColorTypeError:
		// Error from semantic layer
		return c.Semantic.Error
	case ColorTypeInfo:
		// Info from semantic layer
		return c.Semantic.Info
	}
	
	// Check legacy component colors
	var legacyComponent ComponentColorSet
	switch component {
	case ComponentTypeAI:
		legacyComponent = c.Components.AI
	case ComponentTypeSlack:
		legacyComponent = c.Components.Slack
	case ComponentTypeObsidian:
		legacyComponent = c.Components.Obsidian
	case ComponentTypeLinks:
		legacyComponent = c.Components.Links
	case ComponentTypeStats:
		legacyComponent = c.Components.Stats
	case ComponentTypePrompts:
		legacyComponent = c.Components.Prompts
	case ComponentTypeLabels:
		legacyComponent = c.Components.Labels
	case ComponentTypeSearch:
		legacyComponent = c.Components.Search
	case ComponentTypeAttachments:
		legacyComponent = c.Components.Attachments
	case ComponentTypeSavedQueries:
		legacyComponent = c.Components.SavedQueries
	default:
		return ""
	}
	
	switch colorType {
	case ColorTypePrimary:
		return legacyComponent.Title
	case ColorTypeAccent:
		return legacyComponent.Accent
	case ColorTypeBackground:
		return legacyComponent.Background
	case ColorTypeForeground:
		return legacyComponent.Text
	case ColorTypeBorder:
		return legacyComponent.Border
	}
	
	return ""
}

// getFallbackColor provides final fallback colors
func (c *ColorsConfig) getFallbackColor(colorType ColorType) Color {
	switch colorType {
	case ColorTypePrimary:
		return NewColor("#f1fa8c") // Yellow
	case ColorTypeSecondary:
		return NewColor("#6272a4") // Gray
	case ColorTypeAccent:
		return NewColor("#8be9fd") // Cyan
	case ColorTypeBackground:
		return NewColor("#282a36") // Dark gray
	case ColorTypeForeground:
		return NewColor("#f8f8f2") // Light gray
	case ColorTypeBorder:
		return NewColor("#44475a") // Medium gray
	case ColorTypeFocus:
		return NewColor("#6272a4") // Gray
	case ColorTypeSuccess:
		return NewColor("#50fa7b") // Green
	case ColorTypeWarning:
		return NewColor("#f1fa8c") // Yellow
	case ColorTypeError:
		return NewColor("#ff5555") // Red
	case ColorTypeInfo:
		return NewColor("#8be9fd") // Cyan
	}
	return DefaultColor
}

// GetCursorSelectionColors returns cursor selection colors
func (c *ColorsConfig) GetCursorSelectionColors() (bg, fg Color) {
	if c.hasNewStructure() {
		return c.Interaction.Selection.Cursor.Bg, c.Interaction.Selection.Cursor.Fg
	}
	// Legacy fallback
	return c.UI.SelectionBgColor, c.UI.SelectionFgColor
}

// GetBulkSelectionColors returns bulk selection colors  
func (c *ColorsConfig) GetBulkSelectionColors() (bg, fg Color) {
	if c.hasNewStructure() {
		return c.Interaction.Selection.Bulk.Bg, c.Interaction.Selection.Bulk.Fg
	}
	// Legacy fallback
	return c.UI.BulkSelectionBgColor, c.UI.BulkSelectionFgColor
}

// GetInputColors returns input field colors
func (c *ColorsConfig) GetInputColors() (bg, fg, label Color) {
	if c.hasNewStructure() {
		return c.Interaction.Input.Bg, c.Interaction.Input.Fg, c.Interaction.Input.Label
	}
	// Legacy fallback
	return c.UI.InputBgColor, c.UI.InputFgColor, c.UI.LabelColor
}

// GetStatusBarColors returns status bar colors
func (c *ColorsConfig) GetStatusBarColors() (bg, fg Color) {
	if c.hasNewStructure() {
		return c.Interaction.StatusBar.Bg, c.Interaction.StatusBar.Fg
	}
	// Legacy fallback  
	return c.UI.StatusBarBgColor, c.UI.StatusBarFgColor
}

// DefaultColors returns the default color configuration
func DefaultColors() *ColorsConfig {
	return &ColorsConfig{
		Name:        "Gmail Dark",
		Description: "Dark theme based on Dracula color scheme",
		Version:     "2.0",

		// New hierarchical structure
		Foundation: FoundationColors{
			Background: NewColor("#282a36"), // Dark gray
			Foreground: NewColor("#f8f8f2"), // Light gray
			Border:     NewColor("#44475a"), // Medium gray
			Focus:      NewColor("#6272a4"), // Blue
		},
		Semantic: SemanticColors{
			Primary:   NewColor("#f1fa8c"), // Yellow for titles
			Secondary: NewColor("#6272a4"), // Gray for supporting elements
			Accent:    NewColor("#8be9fd"), // Cyan for highlights
			Success:   NewColor("#50fa7b"), // Green
			Warning:   NewColor("#f1fa8c"), // Yellow
			Error:     NewColor("#ff5555"), // Red
			Info:      NewColor("#8be9fd"), // Cyan
		},
		Interaction: InteractionColors{
			Selection: struct {
				Cursor struct {
					Bg Color `yaml:"bg"`
					Fg Color `yaml:"fg"`
				} `yaml:"cursor"`
				Bulk struct {
					Bg Color `yaml:"bg"`
					Fg Color `yaml:"fg"`
				} `yaml:"bulk"`
			}{
				Cursor: struct {
					Bg Color `yaml:"bg"`
					Fg Color `yaml:"fg"`
				}{
					Bg: NewColor("#44475a"), // Dark selection background
					Fg: NewColor("#f8f8f2"), // Light selection text
				},
				Bulk: struct {
					Bg Color `yaml:"bg"`
					Fg Color `yaml:"fg"`
				}{
					Bg: NewColor("#282a36"), // Subtle darker background
					Fg: NewColor("#f8f8f2"), // Same text color
				},
			},
			Input: struct {
				Bg    Color `yaml:"bg"`
				Fg    Color `yaml:"fg"`
				Label Color `yaml:"label"`
			}{
				Bg:    NewColor("#44475a"), // Dark input background
				Fg:    NewColor("#f8f8f2"), // Light input text
				Label: NewColor("#f1fa8c"), // Yellow for labels
			},
			StatusBar: struct {
				Bg Color `yaml:"bg"`
				Fg Color `yaml:"fg"`
			}{
				Bg: NewColor("#6272a4"), // Blue-gray status bar background
				Fg: NewColor("#f8f8f2"), // Light text for status bar
			},
		},
		Overrides: ComponentColorOverrides{
			AI: ComponentOverrideSet{
				Primary: NewColor("#bd93f9"), // Purple for AI titles
				Accent:  NewColor("#ff79c6"), // Pink accent
			},
			Prompts: ComponentOverrideSet{
				Primary: NewColor("#ff79c6"), // Pink for prompt titles
				Accent:  NewColor("#bd93f9"), // Purple accent
			},
		},
		Body: BodyColors{
			FgColor:   NewColor("#f8f8f2"),
			BgColor:   NewColor("#282a36"),
			LogoColor: NewColor("#bd93f9"),
		},
		Frame: FrameColors{
			Border: struct {
				FgColor    Color `yaml:"fgColor"`
				FocusColor Color `yaml:"focusColor"`
			}{
				FgColor:    NewColor("#44475a"),
				FocusColor: NewColor("#6272a4"),
			},
			Title: struct {
				FgColor        Color `yaml:"fgColor"`
				BgColor        Color `yaml:"bgColor"`
				HighlightColor Color `yaml:"highlightColor"`
				CounterColor   Color `yaml:"counterColor"`
				FilterColor    Color `yaml:"filterColor"`
			}{
				FgColor:        NewColor("#f8f8f2"),
				BgColor:        NewColor("#282a36"),
				HighlightColor: NewColor("#f1fa8c"),
				CounterColor:   NewColor("#50fa7b"),
				FilterColor:    NewColor("#8be9fd"),
			},
		},
		Table: TableColors{
			FgColor:       NewColor("#f8f8f2"),
			BgColor:       NewColor("#282a36"),
			HeaderFgColor: NewColor("#50fa7b"),
			HeaderBgColor: NewColor("#282a36"),
		},
		Email: EmailColors{
			UnreadColor:    NewColor("#ffb86c"),
			ReadColor:      NewColor("#6272a4"),
			ImportantColor: NewColor("#ff5555"),
			SentColor:      NewColor("#50fa7b"),
			DraftColor:     NewColor("#f1fa8c"),
		},
		UI: UIColors{
			// Panel and text colors
			TitleColor:  NewColor("#f1fa8c"), // Yellow for titles
			FooterColor: NewColor("#6272a4"), // Gray for footer text
			HintColor:   NewColor("#6272a4"), // Gray for hints

			// Selection colors (cursor/navigation)
			SelectionBgColor: NewColor("#44475a"), // Dark selection background
			SelectionFgColor: NewColor("#f8f8f2"), // Light selection text

			// Bulk selection colors (checked items)
			BulkSelectionBgColor: NewColor("#282a36"), // Subtle darker background
			BulkSelectionFgColor: NewColor("#f8f8f2"), // Same text color

			// Status message colors
			ErrorColor:   NewColor("#ff5555"), // Red for errors
			SuccessColor: NewColor("#50fa7b"), // Green for success
			WarningColor: NewColor("#f1fa8c"), // Yellow for warnings
			InfoColor:    NewColor("#8be9fd"), // Cyan for info

			// Input field colors
			InputBgColor: NewColor("#44475a"), // Dark input background
			InputFgColor: NewColor("#f8f8f2"), // Light input text
			LabelColor:   NewColor("#f1fa8c"), // Yellow for labels

			// Status bar colors
			StatusBarBgColor: NewColor("#6272a4"), // Blue-gray status bar background
			StatusBarFgColor: NewColor("#f8f8f2"), // Light text for status bar
		},

		// Color tags for text markup (replaces hardcoded [color] tags)
		Tags: TagColors{
			Title:     NewColor("#f1fa8c"), // Yellow for titles - replaces [yellow]
			Header:    NewColor("#50fa7b"), // Green for headers - replaces [green]
			Emphasis:  NewColor("#ffb86c"), // Orange for emphasis - replaces [orange]
			Secondary: NewColor("#6272a4"), // Gray for secondary text - replaces [dim]/[gray]
			Link:      NewColor("#8be9fd"), // Cyan for links - replaces [blue]
			Code:      NewColor("#bd93f9"), // Purple for code - replaces [purple]
		},

		// Status message colors (replaces hardcoded tcell.Color* constants)
		Status: StatusColors{
			Error:    NewColor("#ff5555"), // Red for errors - replaces tcell.ColorRed
			Success:  NewColor("#50fa7b"), // Green for success - replaces tcell.ColorGreen
			Warning:  NewColor("#f1fa8c"), // Yellow for warnings - replaces tcell.ColorYellow
			Info:     NewColor("#8be9fd"), // Cyan for info - replaces tcell.ColorBlue
			Progress: NewColor("#ffb86c"), // Orange for progress - replaces tcell.ColorOrange
		},

		// Component-specific colors (replaces hardcoded component colors)
		Components: ComponentColors{
			AI: ComponentColorSet{
				Border:     NewColor("#bd93f9"), // Purple border for AI
				Title:      NewColor("#bd93f9"), // Purple title for AI
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#ff79c6"), // Pink accent
			},
			Slack: ComponentColorSet{
				Border:     NewColor("#50fa7b"), // Green border for Slack
				Title:      NewColor("#50fa7b"), // Green title for Slack
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#8be9fd"), // Cyan accent
			},
			Obsidian: ComponentColorSet{
				Border:     NewColor("#ffb86c"), // Orange border for Obsidian
				Title:      NewColor("#ffb86c"), // Orange title for Obsidian
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#f1fa8c"), // Yellow accent
			},
			Links: ComponentColorSet{
				Border:     NewColor("#8be9fd"), // Cyan border for links
				Title:      NewColor("#8be9fd"), // Cyan title for links
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#50fa7b"), // Green accent
			},
			Stats: ComponentColorSet{
				Border:     NewColor("#f1fa8c"), // Yellow border for stats
				Title:      NewColor("#f1fa8c"), // Yellow title for stats
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#bd93f9"), // Purple accent
			},
			Prompts: ComponentColorSet{
				Border:     NewColor("#ff79c6"), // Pink border for prompts
				Title:      NewColor("#ff79c6"), // Pink title for prompts
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#ffb86c"), // Orange accent
			},
			Compose: ComponentColorSet{
				Border:     NewColor("#6272a4"), // Blue border for email composition
				Title:      NewColor("#6272a4"), // Blue title for email composition
				Background: NewColor("#282a36"), // Dark background
				Text:       NewColor("#f8f8f2"), // Light text
				Accent:     NewColor("#50fa7b"), // Green accent for send/success actions
			},
		},
	}
}
