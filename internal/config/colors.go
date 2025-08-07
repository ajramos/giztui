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

// ColorsConfig defines the complete color configuration
type ColorsConfig struct {
	Body  BodyColors  `yaml:"body"`
	Frame FrameColors `yaml:"frame"`
	Table TableColors `yaml:"table"`
	Email EmailColors `yaml:"email"`
}

// DefaultColors returns the default color configuration
func DefaultColors() *ColorsConfig {
	return &ColorsConfig{
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
	}
}
