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
	
	// Selection colors
	SelectionBgColor Color `yaml:"selectionBgColor"` // List selection background
	SelectionFgColor Color `yaml:"selectionFgColor"` // List selection text
	
	// Status message colors
	ErrorColor   Color `yaml:"errorColor"`   // Error messages
	SuccessColor Color `yaml:"successColor"` // Success messages
	WarningColor Color `yaml:"warningColor"` // Warning messages
	InfoColor    Color `yaml:"infoColor"`    // Info messages
	
	// Input field colors
	InputBgColor Color `yaml:"inputBgColor"` // Input field background
	InputFgColor Color `yaml:"inputFgColor"` // Input field text
	LabelColor   Color `yaml:"labelColor"`   // Input field labels
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
	AI       ComponentColorSet `yaml:"ai"`
	Slack    ComponentColorSet `yaml:"slack"`
	Obsidian ComponentColorSet `yaml:"obsidian"`
	Links    ComponentColorSet `yaml:"links"`
	Stats    ComponentColorSet `yaml:"stats"`
	Prompts  ComponentColorSet `yaml:"prompts"`
}

// ComponentColorSet defines a complete color set for a UI component
type ComponentColorSet struct {
	Border     Color `yaml:"border"`     // Component border color
	Title      Color `yaml:"title"`      // Component title color
	Background Color `yaml:"background"` // Component background color
	Text       Color `yaml:"text"`       // Component text color
	Accent     Color `yaml:"accent"`     // Component accent/highlight color
}

// ColorsConfig defines the complete color configuration
type ColorsConfig struct {
	Name        string      `yaml:"name"`        // Theme name (e.g., "Gmail Dark")
	Description string      `yaml:"description"` // Theme description
	Version     string      `yaml:"version"`     // Theme version
	
	Body       BodyColors       `yaml:"body"`
	Frame      FrameColors      `yaml:"frame"`
	Table      TableColors      `yaml:"table"`
	Email      EmailColors      `yaml:"email"`
	UI         UIColors         `yaml:"ui"`         // UI component colors (previously hardcoded)
	Tags       TagColors        `yaml:"tags"`       // Color tags for text markup
	Status     StatusColors     `yaml:"status"`     // Status message colors
	Components ComponentColors  `yaml:"components"` // Component-specific colors
}

// DefaultColors returns the default color configuration
func DefaultColors() *ColorsConfig {
	return &ColorsConfig{
		Name:        "Gmail Dark",
		Description: "Dark theme based on Dracula color scheme",
		Version:     "1.0",
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
			
			// Selection colors
			SelectionBgColor: NewColor("#44475a"), // Dark selection background
			SelectionFgColor: NewColor("#f8f8f2"), // Light selection text
			
			// Status message colors
			ErrorColor:   NewColor("#ff5555"), // Red for errors
			SuccessColor: NewColor("#50fa7b"), // Green for success
			WarningColor: NewColor("#f1fa8c"), // Yellow for warnings
			InfoColor:    NewColor("#8be9fd"), // Cyan for info
			
			// Input field colors
			InputBgColor: NewColor("#44475a"), // Dark input background
			InputFgColor: NewColor("#f8f8f2"), // Light input text
			LabelColor:   NewColor("#f1fa8c"), // Yellow for labels
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
		},
	}
}
