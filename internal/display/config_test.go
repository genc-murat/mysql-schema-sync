package display

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestDefaultDisplayConfig(t *testing.T) {
	config := DefaultDisplayConfig()

	if !config.ColorEnabled {
		t.Error("Expected ColorEnabled to be true by default")
	}

	if config.Theme != "dark" {
		t.Errorf("Expected default theme to be 'dark', got '%s'", config.Theme)
	}

	if config.OutputFormat != string(FormatTable) {
		t.Errorf("Expected default output format to be 'table', got '%s'", config.OutputFormat)
	}

	if !config.UseIcons {
		t.Error("Expected UseIcons to be true by default")
	}

	if !config.ShowProgress {
		t.Error("Expected ShowProgress to be true by default")
	}

	if !config.InteractiveMode {
		t.Error("Expected InteractiveMode to be true by default")
	}

	if config.TableStyle != "default" {
		t.Errorf("Expected default table style to be 'default', got '%s'", config.TableStyle)
	}

	if config.MaxTableWidth != 120 {
		t.Errorf("Expected default max table width to be 120, got %d", config.MaxTableWidth)
	}

	if config.Writer != os.Stdout {
		t.Error("Expected default writer to be os.Stdout")
	}
}

func TestDisplayConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *DisplayConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			config:      DefaultDisplayConfig(),
			expectError: false,
		},
		{
			name: "invalid theme",
			config: &DisplayConfig{
				Theme:         "invalid",
				OutputFormat:  string(FormatTable),
				TableStyle:    "default",
				MaxTableWidth: 120,
			},
			expectError: true,
			errorMsg:    "invalid theme",
		},
		{
			name: "invalid output format",
			config: &DisplayConfig{
				Theme:         "dark",
				OutputFormat:  "invalid",
				TableStyle:    "default",
				MaxTableWidth: 120,
			},
			expectError: true,
			errorMsg:    "invalid output format",
		},
		{
			name: "invalid table style",
			config: &DisplayConfig{
				Theme:         "dark",
				OutputFormat:  string(FormatTable),
				TableStyle:    "invalid",
				MaxTableWidth: 120,
			},
			expectError: true,
			errorMsg:    "invalid table style",
		},
		{
			name: "invalid max table width - too small",
			config: &DisplayConfig{
				Theme:         "dark",
				OutputFormat:  string(FormatTable),
				TableStyle:    "default",
				MaxTableWidth: 30,
			},
			expectError: true,
			errorMsg:    "max table width must be between 40 and 300",
		},
		{
			name: "invalid max table width - too large",
			config: &DisplayConfig{
				Theme:         "dark",
				OutputFormat:  string(FormatTable),
				TableStyle:    "default",
				MaxTableWidth: 400,
			},
			expectError: true,
			errorMsg:    "max table width must be between 40 and 300",
		},
		{
			name: "conflicting verbose and quiet modes",
			config: &DisplayConfig{
				Theme:         "dark",
				OutputFormat:  string(FormatTable),
				TableStyle:    "default",
				MaxTableWidth: 120,
				VerboseMode:   true,
				QuietMode:     true,
			},
			expectError: true,
			errorMsg:    "verbose and quiet modes are mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %s", err.Error())
				}
			}
		})
	}
}

func TestDisplayConfigSetDefaults(t *testing.T) {
	config := &DisplayConfig{}
	config.SetDefaults()

	if config.Theme != "dark" {
		t.Errorf("Expected theme to be set to 'dark', got '%s'", config.Theme)
	}

	if config.OutputFormat != string(FormatTable) {
		t.Errorf("Expected output format to be set to 'table', got '%s'", config.OutputFormat)
	}

	if config.TableStyle != "default" {
		t.Errorf("Expected table style to be set to 'default', got '%s'", config.TableStyle)
	}

	if config.MaxTableWidth != 120 {
		t.Errorf("Expected max table width to be set to 120, got %d", config.MaxTableWidth)
	}

	if config.Writer != os.Stdout {
		t.Error("Expected writer to be set to os.Stdout")
	}
}

func TestGetColorTheme(t *testing.T) {
	tests := []struct {
		name     string
		theme    string
		expected string
	}{
		{"dark theme", "dark", "dark"},
		{"light theme", "light", "light"},
		{"high-contrast theme", "high-contrast", "high-contrast"},
		{"auto theme", "auto", "dark"},       // auto defaults to dark in simplified implementation
		{"invalid theme", "invalid", "dark"}, // invalid defaults to dark
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DisplayConfig{Theme: tt.theme}
			theme := config.GetColorTheme()

			// We can't easily test the exact theme values, but we can test that it returns a valid theme
			if theme.Primary == ColorReset && theme.Success == ColorReset && theme.Error == ColorReset {
				// This would indicate a plain text theme, which should only happen for "plain" or "none"
				if tt.theme != "plain" && tt.theme != "none" {
					t.Errorf("Expected a colored theme for '%s', but got plain text theme", tt.theme)
				}
			}
		})
	}
}

func TestDisplayConfigBooleanMethods(t *testing.T) {
	config := DefaultDisplayConfig()

	// Test color enabled
	if !config.IsColorEnabled() {
		t.Error("Expected IsColorEnabled to return true for default config")
	}

	config.ColorEnabled = false
	if config.IsColorEnabled() {
		t.Error("Expected IsColorEnabled to return false when ColorEnabled is false")
	}

	config.ColorEnabled = true
	config.QuietMode = true
	if config.IsColorEnabled() {
		t.Error("Expected IsColorEnabled to return false when QuietMode is true")
	}

	// Reset for other tests
	config = DefaultDisplayConfig()

	// Test progress enabled
	if !config.IsProgressEnabled() {
		t.Error("Expected IsProgressEnabled to return true for default config")
	}

	config.ShowProgress = false
	if config.IsProgressEnabled() {
		t.Error("Expected IsProgressEnabled to return false when ShowProgress is false")
	}

	config.ShowProgress = true
	config.QuietMode = true
	if config.IsProgressEnabled() {
		t.Error("Expected IsProgressEnabled to return false when QuietMode is true")
	}

	// Reset for other tests
	config = DefaultDisplayConfig()

	// Test icons enabled
	if !config.IsIconsEnabled() {
		t.Error("Expected IsIconsEnabled to return true for default config")
	}

	config.UseIcons = false
	if config.IsIconsEnabled() {
		t.Error("Expected IsIconsEnabled to return false when UseIcons is false")
	}

	config.UseIcons = true
	config.QuietMode = true
	if config.IsIconsEnabled() {
		t.Error("Expected IsIconsEnabled to return false when QuietMode is true")
	}

	// Reset for other tests
	config = DefaultDisplayConfig()

	// Test interactive enabled
	if !config.IsInteractiveEnabled() {
		t.Error("Expected IsInteractiveEnabled to return true for default config")
	}

	config.InteractiveMode = false
	if config.IsInteractiveEnabled() {
		t.Error("Expected IsInteractiveEnabled to return false when InteractiveMode is false")
	}

	config.InteractiveMode = true
	config.QuietMode = true
	if config.IsInteractiveEnabled() {
		t.Error("Expected IsInteractiveEnabled to return false when QuietMode is true")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

// Comprehensive tests for edge cases and boundary conditions
func TestDisplayConfigComprehensive(t *testing.T) {
	t.Run("SetDefaultsPreservesExistingValues", func(t *testing.T) {
		config := &DisplayConfig{
			ColorEnabled:    false,
			Theme:           "light",
			OutputFormat:    string(FormatJSON),
			UseIcons:        false,
			ShowProgress:    false,
			InteractiveMode: false,
			VerboseMode:     true,
			QuietMode:       false,
			TableStyle:      "rounded",
			MaxTableWidth:   80,
		}

		config.SetDefaults()

		// Should preserve existing non-zero values
		if config.ColorEnabled != false {
			t.Error("SetDefaults should preserve existing ColorEnabled value")
		}
		if config.Theme != "light" {
			t.Error("SetDefaults should preserve existing Theme value")
		}
		if config.OutputFormat != string(FormatJSON) {
			t.Error("SetDefaults should preserve existing OutputFormat value")
		}
		if config.TableStyle != "rounded" {
			t.Error("SetDefaults should preserve existing TableStyle value")
		}
		if config.MaxTableWidth != 80 {
			t.Error("SetDefaults should preserve existing MaxTableWidth value")
		}
	})

	t.Run("BoundaryValueTesting", func(t *testing.T) {
		config := DefaultDisplayConfig()

		// Test exact boundary values
		config.MaxTableWidth = 40 // Minimum valid
		if err := config.Validate(); err != nil {
			t.Error("MaxTableWidth of 40 should be valid")
		}

		config.MaxTableWidth = 300 // Maximum valid
		if err := config.Validate(); err != nil {
			t.Error("MaxTableWidth of 300 should be valid")
		}

		config.MaxTableWidth = 39 // Just below minimum
		if err := config.Validate(); err == nil {
			t.Error("MaxTableWidth of 39 should be invalid")
		}

		config.MaxTableWidth = 301 // Just above maximum
		if err := config.Validate(); err == nil {
			t.Error("MaxTableWidth of 301 should be invalid")
		}
	})

	t.Run("AllValidEnumValues", func(t *testing.T) {
		config := DefaultDisplayConfig()

		// Test all valid themes
		validThemes := []string{"dark", "light", "high-contrast", "auto"}
		for _, theme := range validThemes {
			config.Theme = theme
			if err := config.Validate(); err != nil {
				t.Errorf("Theme '%s' should be valid", theme)
			}
		}

		// Test all valid output formats
		validFormats := []string{string(FormatTable), string(FormatJSON), string(FormatYAML), string(FormatCompact)}
		for _, format := range validFormats {
			config.OutputFormat = format
			if err := config.Validate(); err != nil {
				t.Errorf("OutputFormat '%s' should be valid", format)
			}
		}

		// Test all valid table styles
		validStyles := []string{"default", "rounded", "border", "minimal"}
		for _, style := range validStyles {
			config.TableStyle = style
			if err := config.Validate(); err != nil {
				t.Errorf("TableStyle '%s' should be valid", style)
			}
		}
	})

	t.Run("MultipleValidationErrors", func(t *testing.T) {
		config := &DisplayConfig{
			Theme:         "invalid-theme",
			OutputFormat:  "invalid-format",
			TableStyle:    "invalid-style",
			MaxTableWidth: 10, // Too small
			VerboseMode:   true,
			QuietMode:     true, // Conflicts with verbose
		}

		err := config.Validate()
		if err == nil {
			t.Error("Should have multiple validation errors")
		}

		errStr := err.Error()
		expectedErrors := []string{
			"invalid theme",
			"invalid output format",
			"invalid table style",
			"max table width",
			"mutually exclusive",
		}

		for _, expectedError := range expectedErrors {
			if !containsString(errStr, expectedError) {
				t.Errorf("Error message should contain '%s', got: %s", expectedError, errStr)
			}
		}
	})

	t.Run("QuietModeOverridesAllFeatures", func(t *testing.T) {
		config := &DisplayConfig{
			ColorEnabled:    true,
			UseIcons:        true,
			ShowProgress:    true,
			InteractiveMode: true,
			QuietMode:       true, // Should override all above
		}

		if config.IsColorEnabled() {
			t.Error("QuietMode should disable colors")
		}
		if config.IsIconsEnabled() {
			t.Error("QuietMode should disable icons")
		}
		if config.IsProgressEnabled() {
			t.Error("QuietMode should disable progress")
		}
		if config.IsInteractiveEnabled() {
			t.Error("QuietMode should disable interactive mode")
		}
	})

	t.Run("ThemeResolution", func(t *testing.T) {
		tests := []struct {
			themeName        string
			shouldHaveColors bool
		}{
			{"dark", true},
			{"light", true},
			{"high-contrast", true},
			{"auto", true},    // Should resolve to a colored theme
			{"invalid", true}, // Should default to dark (colored)
			{"", true},        // Empty should default to dark (colored)
		}

		for _, tt := range tests {
			t.Run(tt.themeName, func(t *testing.T) {
				config := &DisplayConfig{Theme: tt.themeName}
				theme := config.GetColorTheme()

				hasColors := theme.Primary != ColorReset || theme.Success != ColorReset || theme.Error != ColorReset

				if hasColors != tt.shouldHaveColors {
					t.Errorf("Theme '%s' should have colors: %v, but got: %v", tt.themeName, tt.shouldHaveColors, hasColors)
				}
			})
		}
	})

	t.Run("ConfigurationCombinations", func(t *testing.T) {
		// Test various valid combinations
		validCombinations := []DisplayConfig{
			{
				ColorEnabled: true, Theme: "dark", OutputFormat: string(FormatTable),
				UseIcons: true, ShowProgress: true, InteractiveMode: true,
				TableStyle: "default", MaxTableWidth: 120,
			},
			{
				ColorEnabled: false, Theme: "light", OutputFormat: string(FormatJSON),
				UseIcons: false, ShowProgress: false, InteractiveMode: false,
				TableStyle: "rounded", MaxTableWidth: 80,
			},
			{
				ColorEnabled: true, Theme: "high-contrast", OutputFormat: string(FormatYAML),
				UseIcons: true, ShowProgress: true, InteractiveMode: false,
				VerboseMode: true, TableStyle: "border", MaxTableWidth: 200,
			},
			{
				ColorEnabled: false, Theme: "auto", OutputFormat: string(FormatCompact),
				UseIcons: false, ShowProgress: false, InteractiveMode: false,
				QuietMode: true, TableStyle: "minimal", MaxTableWidth: 60,
			},
		}

		for i, config := range validCombinations {
			t.Run(fmt.Sprintf("Combination%d", i+1), func(t *testing.T) {
				if err := config.Validate(); err != nil {
					t.Errorf("Valid combination %d should pass validation: %v", i+1, err)
				}
			})
		}
	})

	t.Run("ZeroValueConfig", func(t *testing.T) {
		var config DisplayConfig

		// Zero value config should fail validation
		err := config.Validate()
		if err == nil {
			t.Error("Zero value config should fail validation")
		}

		// But SetDefaults should make it valid
		config.SetDefaults()
		if err := config.Validate(); err != nil {
			t.Errorf("Config with defaults should be valid: %v", err)
		}
	})

	t.Run("WriterHandling", func(t *testing.T) {
		config := &DisplayConfig{}

		// Initially nil writer
		if config.Writer != nil {
			t.Error("Zero value config should have nil writer")
		}

		config.SetDefaults()

		// Should set default writer
		if config.Writer == nil {
			t.Error("SetDefaults should set a writer")
		}
		if config.Writer != os.Stdout {
			t.Error("Default writer should be os.Stdout")
		}

		// Should preserve existing writer
		customWriter := &bytes.Buffer{}
		config.Writer = customWriter
		config.SetDefaults()

		if config.Writer != customWriter {
			t.Error("SetDefaults should preserve existing writer")
		}
	})
}
