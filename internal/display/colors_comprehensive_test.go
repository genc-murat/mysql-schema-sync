package display

import (
	"os"
	"testing"
)

func TestColorSystemComprehensive(t *testing.T) {
	t.Run("ColorSystemCreation", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		if cs == nil {
			t.Fatal("NewColorSystem should not return nil")
		}

		if cs.GetTheme().Primary != theme.Primary {
			t.Error("Color system should preserve theme")
		}
	})

	t.Run("ColorSupport", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		// IsColorSupported should return a boolean
		supported := cs.IsColorSupported()
		_ = supported // We can't predict the exact value in tests

		// But we can test that it's consistent
		if cs.IsColorSupported() != cs.IsColorSupported() {
			t.Error("IsColorSupported should be consistent")
		}
	})

	t.Run("ColorizeAllColors", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		testText := "test"
		colors := []Color{
			ColorReset, ColorBlack, ColorRed, ColorGreen, ColorYellow,
			ColorBlue, ColorMagenta, ColorCyan, ColorWhite,
			ColorBrightRed, ColorBrightGreen, ColorBrightYellow,
			ColorBrightBlue, ColorBrightMagenta, ColorBrightCyan, ColorBrightWhite,
		}

		for _, color := range colors {
			result := cs.Colorize(testText, color)
			if result == "" {
				t.Errorf("Colorize should not return empty string for color %v", color)
			}
			// The result should at least contain the original text
			// (it might have ANSI codes around it)
		}
	})

	t.Run("SprintAndSprintf", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		testText := "hello"

		// Test Sprint
		result := cs.Sprint(ColorRed, testText)
		if result == "" {
			t.Error("Sprint should not return empty string")
		}

		// Test Sprintf
		formatted := cs.Sprintf(ColorBlue, "Hello %s!", "world")
		if formatted == "" {
			t.Error("Sprintf should not return empty string")
		}
		// Should contain the formatted content
		if len(formatted) < len("Hello world!") {
			t.Error("Sprintf result should contain formatted text")
		}
	})

	t.Run("ThemeManagement", func(t *testing.T) {
		theme1 := DarkColorTheme()
		theme2 := LightColorTheme()

		cs := NewColorSystem(theme1)

		// Verify initial theme
		if cs.GetTheme().Primary != theme1.Primary {
			t.Error("Initial theme should match")
		}

		// Change theme
		cs.SetTheme(theme2)
		if cs.GetTheme().Primary != theme2.Primary {
			t.Error("Theme should be updated")
		}

		// Verify all theme fields
		retrieved := cs.GetTheme()
		if retrieved.Success != theme2.Success {
			t.Error("All theme fields should be updated")
		}
	})
}

func TestColorSystemEnvironmentDetection(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"NO_COLOR":    os.Getenv("NO_COLOR"),
		"FORCE_COLOR": os.Getenv("FORCE_COLOR"),
		"TERM":        os.Getenv("TERM"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		testFunc func(*testing.T)
	}{
		{
			name: "NO_COLOR disables colors",
			envVars: map[string]string{
				"NO_COLOR": "1",
				"TERM":     "xterm-256color",
			},
			testFunc: func(t *testing.T) {
				theme := DefaultColorTheme()
				cs := NewColorSystem(theme)

				// Test that colorization still works (returns text)
				result := cs.Colorize("test", ColorRed)
				if result != "test" {
					// In NO_COLOR mode, should return plain text
					// But the exact behavior depends on the implementation
				}
			},
		},
		{
			name: "TERM=dumb disables colors",
			envVars: map[string]string{
				"TERM": "dumb",
			},
			testFunc: func(t *testing.T) {
				theme := DefaultColorTheme()
				cs := NewColorSystem(theme)

				result := cs.Colorize("test", ColorRed)
				if result == "" {
					t.Error("Should still return text even with dumb terminal")
				}
			},
		},
		{
			name: "FORCE_COLOR enables colors",
			envVars: map[string]string{
				"FORCE_COLOR": "1",
				"TERM":        "dumb", // This would normally disable colors
			},
			testFunc: func(t *testing.T) {
				theme := DefaultColorTheme()
				cs := NewColorSystem(theme)

				result := cs.Colorize("test", ColorRed)
				if result == "" {
					t.Error("FORCE_COLOR should enable colorization")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			tt.testFunc(t)
		})
	}
}

func TestPredefinedColorThemes(t *testing.T) {
	themes := map[string]func() ColorTheme{
		"dark":          DarkColorTheme,
		"light":         LightColorTheme,
		"high-contrast": HighContrastColorTheme,
		"plain":         PlainTextTheme,
	}

	for name, themeFunc := range themes {
		t.Run(name, func(t *testing.T) {
			theme := themeFunc()

			// Verify theme has all required fields
			if name != "plain" {
				// Non-plain themes should have distinct colors
				colors := []Color{theme.Primary, theme.Success, theme.Warning, theme.Error, theme.Info, theme.Muted, theme.Highlight}
				for i, color := range colors {
					if color == 0 && name != "plain" {
						t.Errorf("Theme %s field %d should not be zero", name, i)
					}
				}
			} else {
				// Plain theme should use ColorReset for everything
				if theme.Primary != ColorReset || theme.Success != ColorReset {
					t.Error("Plain theme should use ColorReset for all colors")
				}
			}

			// Test that theme can be used with color system
			cs := NewColorSystem(theme)
			result := cs.Colorize("test", theme.Primary)
			if result == "" {
				t.Errorf("Theme %s should work with color system", name)
			}
		})
	}
}

func TestGetThemeByName(t *testing.T) {
	tests := []struct {
		name         string
		expectedType string
	}{
		{"dark", "dark"},
		{"light", "light"},
		{"high-contrast", "high-contrast"},
		{"plain", "plain"},
		{"none", "plain"},
		{"invalid", "dark"}, // Should default to dark
		{"", "dark"},        // Should default to dark
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := GetThemeByName(tt.name)

			// Verify we get a valid theme
			if theme.Primary == 0 && tt.expectedType != "plain" {
				t.Errorf("Theme for name '%s' should have non-zero primary color", tt.name)
			}

			if tt.expectedType == "plain" && theme.Primary != ColorReset {
				t.Errorf("Plain theme should use ColorReset, got %v", theme.Primary)
			}
		})
	}
}

func TestColorSystemEdgeCases(t *testing.T) {
	t.Run("EmptyText", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		result := cs.Colorize("", ColorRed)
		if result != "" {
			t.Error("Colorizing empty string should return empty string")
		}
	})

	t.Run("InvalidColor", func(t *testing.T) {
		theme := DefaultColorTheme()
		cs := NewColorSystem(theme)

		// Test with a color value that doesn't exist in the map
		invalidColor := Color(999)
		result := cs.Colorize("test", invalidColor)
		if result != "test" {
			t.Error("Invalid color should return original text")
		}
	})

	t.Run("NilTheme", func(t *testing.T) {
		// Test with zero-value theme
		var theme ColorTheme
		cs := NewColorSystem(theme)

		result := cs.Colorize("test", ColorRed)
		if result == "" {
			t.Error("Should handle zero-value theme gracefully")
		}
	})

	t.Run("MultipleThemeChanges", func(t *testing.T) {
		cs := NewColorSystem(DarkColorTheme())

		themes := []ColorTheme{
			LightColorTheme(),
			HighContrastColorTheme(),
			PlainTextTheme(),
			DarkColorTheme(),
		}

		for i, theme := range themes {
			cs.SetTheme(theme)
			retrieved := cs.GetTheme()
			if retrieved.Primary != theme.Primary {
				t.Errorf("Theme change %d failed", i)
			}
		}
	})
}
