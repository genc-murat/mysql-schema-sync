package display

import (
	"io"
)

// DisplayService provides centralized formatting and output management
type DisplayService interface {
	// Output formatting
	PrintHeader(title string)
	PrintSection(title string, content interface{})
	PrintTable(headers []string, rows [][]string)
	PrintSQL(statements []string)

	// Progress indicators
	StartSpinner(message string) SpinnerHandle
	UpdateSpinner(handle SpinnerHandle, message string)
	StopSpinner(handle SpinnerHandle, finalMessage string)
	ShowProgress(current, total int, message string)

	// Progress bars
	NewProgressBar(total int, message string) *ProgressBar
	NewMultiProgress() *MultiProgress
	NewProgressTracker(phases []string) *ProgressTracker

	// Status messages
	Success(message string)
	Warning(message string)
	Error(message string)
	Info(message string)

	// Icon rendering
	RenderIcon(name string) string
	RenderIconWithColor(name string) string
	GetIconSystem() IconSystem

	// Table formatting
	NewTableFormatter() TableFormatter
	NewSchemaDiffPresenter() *SchemaDiffPresenter

	// Section-based output
	NewSectionFormatter() *SectionFormatter
	RenderSection(section *Section)
	RenderSections(sections []*Section)

	// SQL syntax highlighting
	NewSQLHighlighter() *SQLHighlighter

	// Configuration
	SetOutput(writer io.Writer)
	GetConfig() *DisplayConfig
	SetConfig(config *DisplayConfig)

	// Output formatting
	NewOutputWriter(format OutputFormat) *OutputWriter
	GetFormatterRegistry() *FormatterRegistry

	// Interactive confirmation dialogs
	NewConfirmationDialog() *ConfirmationDialog
	NewConfirmationBuilder() *ConfirmationBuilder
	NewChangeReviewDialog() *ChangeReviewDialog
}

// OutputFormat represents different output format options
type OutputFormat string

const (
	FormatTable   OutputFormat = "table"
	FormatJSON    OutputFormat = "json"
	FormatYAML    OutputFormat = "yaml"
	FormatCompact OutputFormat = "compact"
)

// Color represents terminal color options
type Color int

const (
	ColorReset Color = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
	ColorBrightRed
	ColorBrightGreen
	ColorBrightYellow
	ColorBrightBlue
	ColorBrightMagenta
	ColorBrightCyan
	ColorBrightWhite
)

// ColorTheme defines color scheme for different message types
type ColorTheme struct {
	Primary   Color
	Success   Color
	Warning   Color
	Error     Color
	Info      Color
	Muted     Color
	Highlight Color
}

// DefaultColorTheme returns a default color theme
func DefaultColorTheme() ColorTheme {
	return ColorTheme{
		Primary:   ColorBlue,
		Success:   ColorGreen,
		Warning:   ColorYellow,
		Error:     ColorRed,
		Info:      ColorCyan,
		Muted:     ColorWhite,
		Highlight: ColorBrightBlue,
	}
}

// SpinnerHandle represents a handle to a running spinner
type SpinnerHandle interface {
	ID() string
	IsActive() bool
}

// SpinnerStyle defines the visual style of a spinner
type SpinnerStyle struct {
	Frames []string
	Delay  int // milliseconds between frames
}

// DefaultSpinnerStyles provides common spinner styles
var DefaultSpinnerStyles = map[string]SpinnerStyle{
	"dots": {
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Delay:  80,
	},
	"line": {
		Frames: []string{"-", "\\", "|", "/"},
		Delay:  100,
	},
	"arrow": {
		Frames: []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		Delay:  120,
	},
}
