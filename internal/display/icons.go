package display

import (
	"os"
	"unicode/utf8"

	"github.com/mattn/go-isatty"
)

// Icon represents a visual icon with Unicode and ASCII fallbacks
type Icon struct {
	Unicode string
	ASCII   string
	Color   Color
}

// IconSystem handles icon rendering with fallbacks
type IconSystem interface {
	GetIcon(name string) Icon
	RenderIcon(name string) string
	RenderIconWithColor(name string, colorSystem ColorSystem) string
	IsUnicodeSupported() bool
	SetUnicodeSupport(enabled bool)
}

// iconSystem implements IconSystem interface
type iconSystem struct {
	unicodeSupported bool
	icons            map[string]Icon
}

// NewIconSystem creates a new icon system with Unicode detection
func NewIconSystem() IconSystem {
	is := &iconSystem{
		unicodeSupported: detectUnicodeSupport(),
		icons:            make(map[string]Icon),
	}

	is.initializeIcons()
	return is
}

// detectUnicodeSupport checks if the terminal supports Unicode characters
func detectUnicodeSupport() bool {
	// Check if FORCE_UNICODE is set first (highest priority)
	if os.Getenv("FORCE_UNICODE") != "" {
		return true
	}

	// Check if NO_UNICODE is set
	if os.Getenv("NO_UNICODE") != "" {
		return false
	}

	// Check environment variables
	if os.Getenv("LANG") == "C" || os.Getenv("LC_ALL") == "C" {
		return false
	}

	// Check if TERM supports Unicode
	term := os.Getenv("TERM")
	if term == "dumb" || term == "vt100" {
		return false
	}

	// Check if output is a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}

	return true
}

// initializeIcons sets up the predefined icon mappings
func (is *iconSystem) initializeIcons() {
	is.icons = map[string]Icon{
		// Change type icons
		"add": {
			Unicode: "➕",
			ASCII:   "+",
			Color:   ColorGreen,
		},
		"remove": {
			Unicode: "➖",
			ASCII:   "-",
			Color:   ColorRed,
		},
		"modify": {
			Unicode: "🔄",
			ASCII:   "*",
			Color:   ColorYellow,
		},

		// Database object icons
		"table": {
			Unicode: "📋",
			ASCII:   "[T]",
			Color:   ColorBlue,
		},
		"column": {
			Unicode: "📄",
			ASCII:   "[C]",
			Color:   ColorCyan,
		},
		"index": {
			Unicode: "🔍",
			ASCII:   "[I]",
			Color:   ColorMagenta,
		},

		// Status icons
		"success": {
			Unicode: "✅",
			ASCII:   "[OK]",
			Color:   ColorGreen,
		},
		"error": {
			Unicode: "❌",
			ASCII:   "[ERR]",
			Color:   ColorRed,
		},
		"warning": {
			Unicode: "⚠️",
			ASCII:   "[WARN]",
			Color:   ColorYellow,
		},
		"info": {
			Unicode: "ℹ️",
			ASCII:   "[INFO]",
			Color:   ColorBlue,
		},

		// Progress and loading icons
		"loading": {
			Unicode: "⏳",
			ASCII:   "...",
			Color:   ColorBlue,
		},
		"done": {
			Unicode: "✓",
			ASCII:   "OK",
			Color:   ColorGreen,
		},
		"failed": {
			Unicode: "✗",
			ASCII:   "FAIL",
			Color:   ColorRed,
		},

		// Arrow and navigation icons
		"arrow-right": {
			Unicode: "→",
			ASCII:   "->",
			Color:   ColorBlue,
		},
		"arrow-down": {
			Unicode: "↓",
			ASCII:   "v",
			Color:   ColorBlue,
		},
		"bullet": {
			Unicode: "•",
			ASCII:   "*",
			Color:   ColorWhite,
		},

		// Severity level icons
		"critical": {
			Unicode: "🔴",
			ASCII:   "[CRIT]",
			Color:   ColorBrightRed,
		},
		"high": {
			Unicode: "🟡",
			ASCII:   "[HIGH]",
			Color:   ColorBrightYellow,
		},
		"medium": {
			Unicode: "🔵",
			ASCII:   "[MED]",
			Color:   ColorBrightBlue,
		},
		"low": {
			Unicode: "⚪",
			ASCII:   "[LOW]",
			Color:   ColorWhite,
		},

		// Section control icons
		"expand": {
			Unicode: "▶",
			ASCII:   ">",
			Color:   ColorBlue,
		},
		"collapse": {
			Unicode: "▼",
			ASCII:   "v",
			Color:   ColorBlue,
		},
	}
}

// GetIcon returns the icon for the given name
func (is *iconSystem) GetIcon(name string) Icon {
	if icon, exists := is.icons[name]; exists {
		return icon
	}
	// Return a default icon if not found
	return Icon{
		Unicode: "?",
		ASCII:   "?",
		Color:   ColorWhite,
	}
}

// RenderIcon returns the appropriate icon representation (Unicode or ASCII)
func (is *iconSystem) RenderIcon(name string) string {
	icon := is.GetIcon(name)

	if is.unicodeSupported && utf8.ValidString(icon.Unicode) {
		return icon.Unicode
	}

	return icon.ASCII
}

// RenderIconWithColor returns the icon with color applied
func (is *iconSystem) RenderIconWithColor(name string, colorSystem ColorSystem) string {
	icon := is.GetIcon(name)
	iconText := is.RenderIcon(name)

	if colorSystem.IsColorSupported() {
		return colorSystem.Colorize(iconText, icon.Color)
	}

	return iconText
}

// IsUnicodeSupported returns whether Unicode is supported
func (is *iconSystem) IsUnicodeSupported() bool {
	return is.unicodeSupported
}

// SetUnicodeSupport manually sets Unicode support (for testing or configuration)
func (is *iconSystem) SetUnicodeSupport(enabled bool) {
	is.unicodeSupported = enabled
}
