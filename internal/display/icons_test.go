package display

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestNewIconSystem(t *testing.T) {
	iconSystem := NewIconSystem()

	if iconSystem == nil {
		t.Fatal("Expected icon system to be created")
	}

	// Test that Unicode support is detected
	if !iconSystem.IsUnicodeSupported() && os.Getenv("NO_UNICODE") == "" {
		// This might be expected in some CI environments
		t.Log("Unicode support not detected, this might be expected in CI")
	}
}

func TestIconSystemGetIcon(t *testing.T) {
	iconSystem := NewIconSystem()

	// Test getting a known icon
	addIcon := iconSystem.GetIcon("add")
	if addIcon.Unicode != "➕" {
		t.Errorf("Expected Unicode '➕', got '%s'", addIcon.Unicode)
	}
	if addIcon.ASCII != "+" {
		t.Errorf("Expected ASCII '+', got '%s'", addIcon.ASCII)
	}
	if addIcon.Color != ColorGreen {
		t.Errorf("Expected ColorGreen, got %v", addIcon.Color)
	}

	// Test getting an unknown icon
	unknownIcon := iconSystem.GetIcon("nonexistent")
	if unknownIcon.Unicode != "?" {
		t.Errorf("Expected default Unicode '?', got '%s'", unknownIcon.Unicode)
	}
	if unknownIcon.ASCII != "?" {
		t.Errorf("Expected default ASCII '?', got '%s'", unknownIcon.ASCII)
	}
}

func TestIconSystemRenderIcon(t *testing.T) {
	iconSystem := NewIconSystem()

	// Test with Unicode support
	iconSystem.SetUnicodeSupport(true)
	result := iconSystem.RenderIcon("add")
	if result != "➕" {
		t.Errorf("Expected Unicode '➕', got '%s'", result)
	}

	// Test without Unicode support
	iconSystem.SetUnicodeSupport(false)
	result = iconSystem.RenderIcon("add")
	if result != "+" {
		t.Errorf("Expected ASCII '+', got '%s'", result)
	}
}

func TestIconSystemRenderIconWithColor(t *testing.T) {
	iconSystem := NewIconSystem()
	colorSystem := NewColorSystem(DarkColorTheme())

	// Test with color support
	result := iconSystem.RenderIconWithColor("add", colorSystem)
	// The result should contain the icon (either Unicode or ASCII)
	// We can't test the exact color codes as they depend on the terminal
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestPredefinedIcons(t *testing.T) {
	iconSystem := NewIconSystem()

	// Test all predefined icons exist
	expectedIcons := []string{
		"add", "remove", "modify",
		"table", "column", "index",
		"success", "error", "warning", "info",
		"loading", "done", "failed",
		"arrow-right", "arrow-down", "bullet",
		"critical", "high", "medium", "low",
	}

	for _, iconName := range expectedIcons {
		icon := iconSystem.GetIcon(iconName)
		if icon.Unicode == "?" && icon.ASCII == "?" {
			t.Errorf("Icon '%s' not found or not properly defined", iconName)
		}
	}
}

func TestUnicodeDetection(t *testing.T) {
	// Save original environment
	originalNoUnicode := os.Getenv("NO_UNICODE")
	originalForceUnicode := os.Getenv("FORCE_UNICODE")

	// Clean up after test
	defer func() {
		if originalNoUnicode != "" {
			os.Setenv("NO_UNICODE", originalNoUnicode)
		} else {
			os.Unsetenv("NO_UNICODE")
		}
		if originalForceUnicode != "" {
			os.Setenv("FORCE_UNICODE", originalForceUnicode)
		} else {
			os.Unsetenv("FORCE_UNICODE")
		}
	}()

	// Test with NO_UNICODE environment variable
	os.Unsetenv("FORCE_UNICODE")
	os.Setenv("NO_UNICODE", "1")
	iconSystem := NewIconSystem()
	if iconSystem.IsUnicodeSupported() {
		t.Error("Expected Unicode support to be disabled with NO_UNICODE=1")
	}

	// Test with FORCE_UNICODE environment variable
	os.Unsetenv("NO_UNICODE")
	os.Setenv("FORCE_UNICODE", "1")
	iconSystem = NewIconSystem()
	if !iconSystem.IsUnicodeSupported() {
		t.Error("Expected Unicode support to be enabled with FORCE_UNICODE=1")
	}
}

func TestIconSystemSetUnicodeSupport(t *testing.T) {
	iconSystem := NewIconSystem()

	// Test enabling Unicode support
	iconSystem.SetUnicodeSupport(true)
	if !iconSystem.IsUnicodeSupported() {
		t.Error("Expected Unicode support to be enabled")
	}

	// Test disabling Unicode support
	iconSystem.SetUnicodeSupport(false)
	if iconSystem.IsUnicodeSupported() {
		t.Error("Expected Unicode support to be disabled")
	}
}
func TestDisplayServiceIconIntegrationComprehensive(t *testing.T) {
	config := DefaultDisplayConfig()
	displayService := NewDisplayService(config)

	// Test RenderIcon method
	icon := displayService.RenderIcon("add")
	if icon == "" {
		t.Error("Expected non-empty icon")
	}

	// Test RenderIconWithColor method
	coloredIcon := displayService.RenderIconWithColor("success")
	if coloredIcon == "" {
		t.Error("Expected non-empty colored icon")
	}

	// Test GetIconSystem method
	iconSystem := displayService.GetIconSystem()
	if iconSystem == nil {
		t.Error("Expected icon system to be available")
	}

	// Verify icon system works through display service
	testIcon := iconSystem.GetIcon("error")
	if testIcon.Unicode == "?" && testIcon.ASCII == "?" {
		t.Error("Expected error icon to be properly defined")
	}
}

// Comprehensive tests for icon system edge cases and different terminal capabilities
func TestIconSystemComprehensive(t *testing.T) {
	t.Run("AllIconsHaveValidRepresentations", func(t *testing.T) {
		iconSystem := NewIconSystem()

		iconNames := []string{
			"add", "remove", "modify",
			"table", "column", "index",
			"success", "error", "warning", "info",
			"loading", "done", "failed",
			"arrow-right", "arrow-down", "bullet",
			"critical", "high", "medium", "low",
			"expand", "collapse",
		}

		for _, name := range iconNames {
			icon := iconSystem.GetIcon(name)
			if icon.Unicode == "" {
				t.Errorf("Icon %s should have Unicode representation", name)
			}
			if icon.ASCII == "" {
				t.Errorf("Icon %s should have ASCII representation", name)
			}
			if icon.Color == 0 {
				t.Errorf("Icon %s should have a color assigned", name)
			}
		}
	})

	t.Run("EnvironmentVariableHandling", func(t *testing.T) {
		// Save original environment
		originalEnv := map[string]string{
			"FORCE_UNICODE": os.Getenv("FORCE_UNICODE"),
			"NO_UNICODE":    os.Getenv("NO_UNICODE"),
			"LANG":          os.Getenv("LANG"),
			"TERM":          os.Getenv("TERM"),
		}

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
			expected bool
		}{
			{
				name:     "FORCE_UNICODE overrides other settings",
				envVars:  map[string]string{"FORCE_UNICODE": "1", "TERM": "dumb"},
				expected: true,
			},
			{
				name:     "NO_UNICODE disables Unicode",
				envVars:  map[string]string{"NO_UNICODE": "1"},
				expected: false,
			},
			{
				name:     "LANG=C disables Unicode",
				envVars:  map[string]string{"LANG": "C"},
				expected: false,
			},
			{
				name:     "TERM=dumb disables Unicode",
				envVars:  map[string]string{"TERM": "dumb"},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Clear all relevant env vars first
				for key := range originalEnv {
					os.Unsetenv(key)
				}

				// Set test env vars
				for key, value := range tt.envVars {
					os.Setenv(key, value)
				}

				iconSystem := NewIconSystem()
				if iconSystem.IsUnicodeSupported() != tt.expected {
					t.Errorf("Expected Unicode support to be %v", tt.expected)
				}
			})
		}
	})

	t.Run("UnicodeToggling", func(t *testing.T) {
		iconSystem := NewIconSystem()

		// Test multiple toggles
		for i := 0; i < 5; i++ {
			iconSystem.SetUnicodeSupport(true)
			if !iconSystem.IsUnicodeSupported() {
				t.Error("Unicode support should be enabled")
			}

			unicodeIcon := iconSystem.RenderIcon("success")
			if unicodeIcon == "" {
				t.Error("Unicode icon should render")
			}

			iconSystem.SetUnicodeSupport(false)
			if iconSystem.IsUnicodeSupported() {
				t.Error("Unicode support should be disabled")
			}

			asciiIcon := iconSystem.RenderIcon("success")
			if asciiIcon == "" {
				t.Error("ASCII icon should render")
			}
		}
	})

	t.Run("ColorApplicationWithDifferentThemes", func(t *testing.T) {
		iconSystem := NewIconSystem()

		themes := []ColorTheme{
			DarkColorTheme(),
			LightColorTheme(),
			HighContrastColorTheme(),
			PlainTextTheme(),
		}

		for i, theme := range themes {
			t.Run(fmt.Sprintf("Theme%d", i), func(t *testing.T) {
				colorSystem := NewColorSystem(theme)

				iconNames := []string{"success", "error", "warning", "info"}
				for _, name := range iconNames {
					colored := iconSystem.RenderIconWithColor(name, colorSystem)
					if colored == "" {
						t.Errorf("Colored icon %s should render with theme %d", name, i)
					}
				}
			})
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		iconSystem := NewIconSystem()
		colorSystem := NewColorSystem(DefaultColorTheme())

		done := make(chan bool, 20)

		// Test concurrent access to icon system
		for i := 0; i < 20; i++ {
			go func(id int) {
				defer func() { done <- true }()

				iconNames := []string{"success", "error", "warning", "info", "add", "remove"}
				name := iconNames[id%len(iconNames)]

				// These should be safe for concurrent access
				icon := iconSystem.GetIcon(name)
				if icon.Unicode == "" && icon.ASCII == "" {
					t.Errorf("Concurrent GetIcon failed for %s", name)
					return
				}

				rendered := iconSystem.RenderIcon(name)
				if rendered == "" {
					t.Errorf("Concurrent RenderIcon failed for %s", name)
					return
				}

				colored := iconSystem.RenderIconWithColor(name, colorSystem)
				if colored == "" {
					t.Errorf("Concurrent RenderIconWithColor failed for %s", name)
					return
				}

				// Test Unicode toggling concurrently
				iconSystem.SetUnicodeSupport(id%2 == 0)
				_ = iconSystem.IsUnicodeSupported()
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	t.Run("InvalidUnicodeHandling", func(t *testing.T) {
		iconSystem := NewIconSystem()

		// Force Unicode support
		iconSystem.SetUnicodeSupport(true)

		// Test with all icons to ensure no invalid Unicode causes crashes
		iconNames := []string{
			"add", "remove", "modify", "table", "column", "index",
			"success", "error", "warning", "info", "loading", "done", "failed",
			"arrow-right", "arrow-down", "bullet", "critical", "high", "medium", "low",
			"expand", "collapse",
		}

		for _, name := range iconNames {
			rendered := iconSystem.RenderIcon(name)
			if rendered == "" {
				t.Errorf("Icon %s should render even with Unicode issues", name)
			}
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		iconSystem := NewIconSystem()

		// Test empty icon name
		emptyIcon := iconSystem.GetIcon("")
		if emptyIcon.Unicode != "?" || emptyIcon.ASCII != "?" {
			t.Error("Empty icon name should return default icon")
		}

		// Test very long icon name
		longName := strings.Repeat("a", 1000)
		longIcon := iconSystem.GetIcon(longName)
		if longIcon.Unicode != "?" || longIcon.ASCII != "?" {
			t.Error("Long icon name should return default icon")
		}

		// Test special characters in icon name
		specialIcon := iconSystem.GetIcon("!@#$%^&*()")
		if specialIcon.Unicode != "?" || specialIcon.ASCII != "?" {
			t.Error("Special character icon name should return default icon")
		}
	})
}
