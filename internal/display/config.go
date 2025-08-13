package display

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// DisplayConfig holds configuration for visual display options
type DisplayConfig struct {
	// Visual options
	ColorEnabled bool   `mapstructure:"color_enabled" yaml:"color_enabled"`
	Theme        string `mapstructure:"theme" yaml:"theme"`
	OutputFormat string `mapstructure:"output_format" yaml:"output_format"`
	UseIcons     bool   `mapstructure:"use_icons" yaml:"use_icons"`
	ShowProgress bool   `mapstructure:"show_progress" yaml:"show_progress"`

	// Interaction options
	InteractiveMode bool `mapstructure:"interactive" yaml:"interactive"`

	// Output control
	VerboseMode bool `mapstructure:"verbose" yaml:"verbose"`
	QuietMode   bool `mapstructure:"quiet" yaml:"quiet"`

	// Table formatting options
	TableStyle    string `mapstructure:"table_style" yaml:"table_style"`
	MaxTableWidth int    `mapstructure:"max_table_width" yaml:"max_table_width"`

	// Internal fields (not serialized)
	Writer io.Writer `mapstructure:"-" yaml:"-"`
}

// ThemeName represents available color themes
type ThemeName string

const (
	ThemeDark         ThemeName = "dark"
	ThemeLight        ThemeName = "light"
	ThemeHighContrast ThemeName = "high-contrast"
	ThemeAuto         ThemeName = "auto"
)

// TableStyleName represents available table styles
type TableStyleName string

const (
	TableStyleDefault TableStyleName = "default"
	TableStyleRounded TableStyleName = "rounded"
	TableStyleBorder  TableStyleName = "border"
	TableStyleMinimal TableStyleName = "minimal"
)

// DefaultDisplayConfig returns a default display configuration
func DefaultDisplayConfig() *DisplayConfig {
	return &DisplayConfig{
		ColorEnabled:    true,
		Theme:           string(ThemeDark),
		OutputFormat:    string(FormatTable),
		UseIcons:        true,
		ShowProgress:    true,
		InteractiveMode: true,
		VerboseMode:     false,
		QuietMode:       false,
		TableStyle:      string(TableStyleDefault),
		MaxTableWidth:   120,
		Writer:          os.Stdout,
	}
}

// Validate validates the display configuration
func (dc *DisplayConfig) Validate() error {
	var errs []error

	// Validate theme
	validThemes := []string{string(ThemeDark), string(ThemeLight), string(ThemeHighContrast), string(ThemeAuto)}
	if !contains(validThemes, dc.Theme) {
		errs = append(errs, fmt.Errorf("invalid theme '%s', must be one of: %s", dc.Theme, strings.Join(validThemes, ", ")))
	}

	// Validate output format
	validFormats := []string{string(FormatTable), string(FormatJSON), string(FormatYAML), string(FormatCompact)}
	if !contains(validFormats, string(dc.OutputFormat)) {
		errs = append(errs, fmt.Errorf("invalid output format '%s', must be one of: %s", dc.OutputFormat, strings.Join(validFormats, ", ")))
	}

	// Validate table style
	validTableStyles := []string{string(TableStyleDefault), string(TableStyleRounded), string(TableStyleBorder), string(TableStyleMinimal)}
	if !contains(validTableStyles, dc.TableStyle) {
		errs = append(errs, fmt.Errorf("invalid table style '%s', must be one of: %s", dc.TableStyle, strings.Join(validTableStyles, ", ")))
	}

	// Validate max table width
	if dc.MaxTableWidth < 40 || dc.MaxTableWidth > 300 {
		errs = append(errs, fmt.Errorf("max table width must be between 40 and 300, got %d", dc.MaxTableWidth))
	}

	// Check for conflicting options
	if dc.VerboseMode && dc.QuietMode {
		errs = append(errs, fmt.Errorf("verbose and quiet modes are mutually exclusive"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("display configuration validation failed: %v", errs)
	}

	return nil
}

// SetDefaults sets default values for unspecified configuration options
func (dc *DisplayConfig) SetDefaults() {
	if dc.Theme == "" {
		dc.Theme = string(ThemeDark)
	}

	if dc.OutputFormat == "" {
		dc.OutputFormat = string(FormatTable)
	}

	if dc.TableStyle == "" {
		dc.TableStyle = string(TableStyleDefault)
	}

	if dc.MaxTableWidth == 0 {
		dc.MaxTableWidth = 120
	}

	if dc.Writer == nil {
		dc.Writer = os.Stdout
	}
}

// GetColorTheme returns the ColorTheme based on the theme name
func (dc *DisplayConfig) GetColorTheme() ColorTheme {
	return GetThemeByName(dc.Theme)
}

// IsColorEnabled returns true if colors should be used
func (dc *DisplayConfig) IsColorEnabled() bool {
	return dc.ColorEnabled && !dc.QuietMode
}

// IsProgressEnabled returns true if progress indicators should be shown
func (dc *DisplayConfig) IsProgressEnabled() bool {
	return dc.ShowProgress && !dc.QuietMode
}

// IsIconsEnabled returns true if icons should be used
func (dc *DisplayConfig) IsIconsEnabled() bool {
	return dc.UseIcons && !dc.QuietMode
}

// IsInteractiveEnabled returns true if interactive features should be used
func (dc *DisplayConfig) IsInteractiveEnabled() bool {
	return dc.InteractiveMode && !dc.QuietMode
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
