package display

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// ColorSystem handles color application and terminal detection
type ColorSystem interface {
	Colorize(text string, color Color) string
	Sprint(color Color, text string) string
	Sprintf(color Color, format string, args ...interface{}) string
	IsColorSupported() bool
	SetTheme(theme ColorTheme)
	GetTheme() ColorTheme
}

// colorSystem implements ColorSystem interface
type colorSystem struct {
	theme          ColorTheme
	colorSupported bool
	profile        termenv.Profile
	colorMap       map[Color]*color.Color
}

// NewColorSystem creates a new color system with terminal detection
func NewColorSystem(theme ColorTheme) ColorSystem {
	cs := &colorSystem{
		theme:          theme,
		colorSupported: detectColorSupport(),
		profile:        termenv.ColorProfile(),
		colorMap:       make(map[Color]*color.Color),
	}

	cs.initializeColorMap()
	return cs
}

// detectColorSupport checks if the terminal supports colors
func detectColorSupport() bool {
	// Check if output is a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}

	// Check environment variables that disable color
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	// Check if FORCE_COLOR is set
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	return true
}

// initializeColorMap sets up the mapping between Color constants and fatih/color colors
func (cs *colorSystem) initializeColorMap() {
	cs.colorMap = map[Color]*color.Color{
		ColorReset:         color.New(color.Reset),
		ColorBlack:         color.New(color.FgBlack),
		ColorRed:           color.New(color.FgRed),
		ColorGreen:         color.New(color.FgGreen),
		ColorYellow:        color.New(color.FgYellow),
		ColorBlue:          color.New(color.FgBlue),
		ColorMagenta:       color.New(color.FgMagenta),
		ColorCyan:          color.New(color.FgCyan),
		ColorWhite:         color.New(color.FgWhite),
		ColorBrightRed:     color.New(color.FgHiRed),
		ColorBrightGreen:   color.New(color.FgHiGreen),
		ColorBrightYellow:  color.New(color.FgHiYellow),
		ColorBrightBlue:    color.New(color.FgHiBlue),
		ColorBrightMagenta: color.New(color.FgHiMagenta),
		ColorBrightCyan:    color.New(color.FgHiCyan),
		ColorBrightWhite:   color.New(color.FgHiWhite),
	}

	// Disable colors if not supported
	if !cs.colorSupported {
		color.NoColor = true
	}
}

// Colorize applies color to text if color is supported
func (cs *colorSystem) Colorize(text string, clr Color) string {
	if !cs.colorSupported {
		return text
	}

	if colorFunc, exists := cs.colorMap[clr]; exists {
		return colorFunc.Sprint(text)
	}

	return text
}

// Sprint formats text with color
func (cs *colorSystem) Sprint(clr Color, text string) string {
	return cs.Colorize(text, clr)
}

// Sprintf formats text with color using format string
func (cs *colorSystem) Sprintf(clr Color, format string, args ...interface{}) string {
	text := fmt.Sprintf(format, args...)
	return cs.Colorize(text, clr)
}

// IsColorSupported returns whether colors are supported
func (cs *colorSystem) IsColorSupported() bool {
	return cs.colorSupported
}

// SetTheme updates the color theme
func (cs *colorSystem) SetTheme(theme ColorTheme) {
	cs.theme = theme
}

// GetTheme returns the current color theme
func (cs *colorSystem) GetTheme() ColorTheme {
	return cs.theme
}

// Predefined color themes

// DarkColorTheme returns a color theme optimized for dark terminals
func DarkColorTheme() ColorTheme {
	return ColorTheme{
		Primary:   ColorBrightBlue,
		Success:   ColorBrightGreen,
		Warning:   ColorBrightYellow,
		Error:     ColorBrightRed,
		Info:      ColorCyan,
		Muted:     ColorWhite,
		Highlight: ColorBrightBlue,
	}
}

// LightColorTheme returns a color theme optimized for light terminals
func LightColorTheme() ColorTheme {
	return ColorTheme{
		Primary:   ColorBlue,
		Success:   ColorGreen,
		Warning:   ColorYellow,
		Error:     ColorRed,
		Info:      ColorCyan,
		Muted:     ColorMagenta,
		Highlight: ColorBlue,
	}
}

// HighContrastColorTheme returns a high-contrast color theme for accessibility
func HighContrastColorTheme() ColorTheme {
	return ColorTheme{
		Primary:   ColorBrightBlue,
		Success:   ColorBrightGreen,
		Warning:   ColorBrightYellow,
		Error:     ColorBrightRed,
		Info:      ColorBrightCyan,
		Muted:     ColorWhite,
		Highlight: ColorBrightWhite,
	}
}

// PlainTextTheme returns a theme that uses no colors (fallback)
func PlainTextTheme() ColorTheme {
	return ColorTheme{
		Primary:   ColorReset,
		Success:   ColorReset,
		Warning:   ColorReset,
		Error:     ColorReset,
		Info:      ColorReset,
		Muted:     ColorReset,
		Highlight: ColorReset,
	}
}

// GetThemeByName returns a color theme by name
func GetThemeByName(name string) ColorTheme {
	switch name {
	case "dark":
		return DarkColorTheme()
	case "light":
		return LightColorTheme()
	case "high-contrast":
		return HighContrastColorTheme()
	case "plain", "none":
		return PlainTextTheme()
	default:
		return DarkColorTheme() // Default to dark theme
	}
}
