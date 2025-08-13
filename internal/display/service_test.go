package display

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewDisplayService(t *testing.T) {
	// Test with default config
	service := NewDisplayService(nil)
	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	config := service.GetConfig()
	if config == nil {
		t.Fatal("Expected config to be set, got nil")
	}

	if !config.ColorEnabled {
		t.Error("Expected color to be enabled by default")
	}

	if config.OutputFormat != string(FormatTable) {
		t.Errorf("Expected default format to be table, got %s", config.OutputFormat)
	}
}

func TestDisplayServiceOutput(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.ColorEnabled = false // Disable colors for predictable testing

	service := NewDisplayService(config)

	// Test header output
	service.PrintHeader("Test Header")
	output := buf.String()
	if !strings.Contains(output, "Test Header") {
		t.Error("Expected header to contain 'Test Header'")
	}

	// Reset buffer
	buf.Reset()

	// Test section output
	service.PrintSection("Test Section", "Test Content")
	output = buf.String()
	if !strings.Contains(output, "Test Section") {
		t.Error("Expected section to contain 'Test Section'")
	}
	if !strings.Contains(output, "Test Content") {
		t.Error("Expected section to contain 'Test Content'")
	}
}

func TestDisplayServiceStatusMessages(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.ColorEnabled = false // Disable colors for predictable testing

	service := NewDisplayService(config)

	// Test success message
	service.Success("Operation completed")
	output := buf.String()
	if !strings.Contains(output, "[SUCCESS]") {
		t.Error("Expected success message to contain '[SUCCESS]'")
	}
	if !strings.Contains(output, "Operation completed") {
		t.Error("Expected success message to contain 'Operation completed'")
	}

	// Reset buffer
	buf.Reset()

	// Test error message
	service.Error("Something went wrong")
	output = buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Expected error message to contain '[ERROR]'")
	}
	if !strings.Contains(output, "Something went wrong") {
		t.Error("Expected error message to contain 'Something went wrong'")
	}
}

func TestDisplayServiceTable(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.ColorEnabled = false

	service := NewDisplayService(config)

	headers := []string{"Column1", "Column2"}
	rows := [][]string{
		{"Value1", "Value2"},
		{"Value3", "Value4"},
	}

	service.PrintTable(headers, rows)
	output := buf.String()

	// Debug output to see what's actually generated
	t.Logf("Table output: %q", output)

	if !strings.Contains(output, "Column1") {
		t.Error("Expected table to contain 'Column1'")
	}
	if !strings.Contains(output, "Value1") {
		t.Error("Expected table to contain 'Value1'")
	}
}

func TestDisplayServiceQuietMode(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.QuietMode = true

	service := NewDisplayService(config)

	// Test that quiet mode suppresses output
	service.PrintHeader("Test Header")
	service.Info("Test info message")

	output := buf.String()
	if strings.Contains(output, "Test Header") {
		t.Error("Expected quiet mode to suppress header output")
	}
	if strings.Contains(output, "Test info message") {
		t.Error("Expected quiet mode to suppress info messages")
	}

	// But error messages should still show
	service.Error("Test error")
	output = buf.String()
	if !strings.Contains(output, "Test error") {
		t.Error("Expected error messages to show even in quiet mode")
	}
}

func TestColorSystemDetection(t *testing.T) {
	theme := DefaultColorTheme()
	colorSystem := NewColorSystem(theme)

	if colorSystem == nil {
		t.Fatal("Expected color system to be created, got nil")
	}

	// Test colorization (should work regardless of terminal support)
	result := colorSystem.Colorize("test", ColorRed)
	if result == "" {
		t.Error("Expected colorized text to not be empty")
	}

	// Test theme operations
	newTheme := ColorTheme{
		Primary: ColorBlue,
		Success: ColorGreen,
		Warning: ColorYellow,
		Error:   ColorRed,
	}

	colorSystem.SetTheme(newTheme)
	retrievedTheme := colorSystem.GetTheme()

	if retrievedTheme.Primary != ColorBlue {
		t.Error("Expected theme to be updated")
	}
}

// Comprehensive tests for color system with different terminal capabilities
func TestColorSystemTerminalCapabilities(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectColor bool
	}{
		{
			name:        "normal terminal",
			envVars:     map[string]string{},
			expectColor: true, // Depends on actual terminal
		},
		{
			name:        "NO_COLOR set",
			envVars:     map[string]string{"NO_COLOR": "1"},
			expectColor: false,
		},
		{
			name:        "TERM=dumb",
			envVars:     map[string]string{"TERM": "dumb"},
			expectColor: false,
		},
		{
			name:        "FORCE_COLOR set",
			envVars:     map[string]string{"FORCE_COLOR": "1"},
			expectColor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test env vars
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore env vars after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			// Create new color system to pick up env changes
			theme := DefaultColorTheme()
			colorSystem := NewColorSystem(theme)

			// Test all color functions
			testText := "test text"

			// Test Colorize
			colorized := colorSystem.Colorize(testText, ColorRed)
			if colorized == "" {
				t.Error("Colorize should never return empty string")
			}

			// Test Sprint
			sprinted := colorSystem.Sprint(ColorGreen, testText)
			if sprinted == "" {
				t.Error("Sprint should never return empty string")
			}

			// Test Sprintf
			formatted := colorSystem.Sprintf(ColorBlue, "formatted %s", testText)
			if !strings.Contains(formatted, testText) {
				t.Error("Sprintf should contain the formatted text")
			}
		})
	}
}

func TestColorSystemAllColors(t *testing.T) {
	theme := DefaultColorTheme()
	colorSystem := NewColorSystem(theme)

	colors := []Color{
		ColorReset, ColorBlack, ColorRed, ColorGreen, ColorYellow,
		ColorBlue, ColorMagenta, ColorCyan, ColorWhite,
		ColorBrightRed, ColorBrightGreen, ColorBrightYellow,
		ColorBrightBlue, ColorBrightMagenta, ColorBrightCyan, ColorBrightWhite,
	}

	testText := "test"
	for _, color := range colors {
		result := colorSystem.Colorize(testText, color)
		if result == "" {
			t.Errorf("Colorize should not return empty string for color %d", color)
		}
	}
}

func TestColorThemes(t *testing.T) {
	themes := map[string]ColorTheme{
		"dark":          DarkColorTheme(),
		"light":         LightColorTheme(),
		"high-contrast": HighContrastColorTheme(),
		"plain":         PlainTextTheme(),
		"none":          PlainTextTheme(),
		"invalid":       DarkColorTheme(), // Should default to dark
	}

	for name := range themes {
		t.Run(name, func(t *testing.T) {
			theme := GetThemeByName(name)

			// For plain/none themes, all colors should be ColorReset
			if name == "plain" || name == "none" {
				if theme.Primary != ColorReset || theme.Success != ColorReset {
					t.Error("Plain theme should use ColorReset for all colors")
				}
			}

			// Test that theme has all required fields
			if theme.Primary == 0 && name != "plain" && name != "none" {
				t.Error("Theme should have non-zero primary color")
			}
		})
	}
}

func TestDisplayServiceOutputFormats(t *testing.T) {
	formats := []OutputFormat{FormatTable, FormatJSON, FormatYAML, FormatCompact}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			var buf bytes.Buffer
			config := DefaultDisplayConfig()
			config.Writer = &buf
			config.ColorEnabled = false
			config.OutputFormat = string(format)

			service := NewDisplayService(config)

			// Test section output
			service.PrintSection("Test Section", map[string]string{"key": "value"})

			// Test table output
			headers := []string{"Col1", "Col2"}
			rows := [][]string{{"Val1", "Val2"}, {"Val3", "Val4"}}
			service.PrintTable(headers, rows)

			// Test SQL output
			statements := []string{"SELECT * FROM test;", "UPDATE test SET col = 'value';"}
			service.PrintSQL(statements)

			output := buf.String()
			if output == "" {
				t.Errorf("Expected output for format %s, got empty string", format)
			}

			// Format-specific checks
			switch format {
			case FormatJSON:
				if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
					t.Error("JSON format should contain braces")
				}
			case FormatYAML:
				if !strings.Contains(output, ":") {
					t.Error("YAML format should contain colons")
				}
			case FormatTable:
				// Table format should contain the original data
				if !strings.Contains(output, "Test Section") {
					t.Error("Table format should contain section title")
				}
			}
		})
	}
}

func TestDisplayServiceSpinnerIntegration(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.QuietMode = false

	service := NewDisplayService(config)

	// Test spinner lifecycle
	handle := service.StartSpinner("Testing spinner")
	if handle == nil {
		t.Fatal("StartSpinner should return non-nil handle")
	}

	if !handle.IsActive() {
		t.Error("Spinner should be active after start")
	}

	// Let spinner run briefly
	time.Sleep(100 * time.Millisecond)

	// Update spinner message
	service.UpdateSpinner(handle, "Updated message")
	time.Sleep(100 * time.Millisecond)

	// Stop spinner
	service.StopSpinner(handle, "Completed successfully")

	if handle.IsActive() {
		t.Error("Spinner should not be active after stop")
	}

	// Check output
	output := buf.String()
	if !strings.Contains(output, "Completed successfully") {
		t.Error("Output should contain final message")
	}
}

func TestDisplayServiceProgressIntegration(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf
	config.QuietMode = false

	service := NewDisplayService(config)

	// Test ShowProgress
	service.ShowProgress(0, 10, "Starting")
	service.ShowProgress(5, 10, "Half way")
	service.ShowProgress(10, 10, "Complete")

	output := buf.String()
	if !strings.Contains(output, "50.0%") {
		t.Error("Progress output should contain percentage")
	}
	if !strings.Contains(output, "100.0%") {
		t.Error("Progress output should show completion")
	}

	// Test progress bar creation
	pb := service.NewProgressBar(20, "Test progress bar")
	if pb == nil {
		t.Error("NewProgressBar should return non-nil progress bar")
	}

	pb.Update(10, "Half done")
	pb.Finish("All done")

	// Test multi-progress creation
	mp := service.NewMultiProgress()
	if mp == nil {
		t.Error("NewMultiProgress should return non-nil multi-progress")
	}

	// Test progress tracker creation
	pt := service.NewProgressTracker([]string{"Phase1", "Phase2"})
	if pt == nil {
		t.Error("NewProgressTracker should return non-nil progress tracker")
	}
}

func TestDisplayServiceConfigurationChanges(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultDisplayConfig()
	config.Writer = &buf

	service := NewDisplayService(config)

	// Test initial config
	initialConfig := service.GetConfig()
	if initialConfig.ColorEnabled != config.ColorEnabled {
		t.Error("Initial config should match provided config")
	}

	// Test config update
	newConfig := DefaultDisplayConfig()
	newConfig.Writer = &buf
	newConfig.ColorEnabled = false
	newConfig.QuietMode = true
	newConfig.OutputFormat = string(FormatJSON)

	service.SetConfig(newConfig)

	updatedConfig := service.GetConfig()
	if updatedConfig.ColorEnabled != false {
		t.Error("Config should be updated")
	}
	if updatedConfig.QuietMode != true {
		t.Error("Quiet mode should be updated")
	}
}

func TestDisplayServiceFactoryMethods(t *testing.T) {
	config := DefaultDisplayConfig()
	service := NewDisplayService(config)

	// Test all factory methods return non-nil objects
	if service.NewTableFormatter() == nil {
		t.Error("NewTableFormatter should return non-nil formatter")
	}

	if service.NewSchemaDiffPresenter() == nil {
		t.Error("NewSchemaDiffPresenter should return non-nil presenter")
	}

	if service.NewSectionFormatter() == nil {
		t.Error("NewSectionFormatter should return non-nil formatter")
	}

	if service.NewSQLHighlighter() == nil {
		t.Error("NewSQLHighlighter should return non-nil highlighter")
	}

	if service.NewOutputWriter(FormatJSON) == nil {
		t.Error("NewOutputWriter should return non-nil writer")
	}

	if service.GetFormatterRegistry() == nil {
		t.Error("GetFormatterRegistry should return non-nil registry")
	}

	if service.NewConfirmationDialog() == nil {
		t.Error("NewConfirmationDialog should return non-nil dialog")
	}

	if service.NewConfirmationBuilder() == nil {
		t.Error("NewConfirmationBuilder should return non-nil builder")
	}

	if service.NewChangeReviewDialog() == nil {
		t.Error("NewChangeReviewDialog should return non-nil dialog")
	}
}

func TestDisplayServiceIconIntegration(t *testing.T) {
	config := DefaultDisplayConfig()
	service := NewDisplayService(config)

	// Test icon rendering
	icon := service.RenderIcon("success")
	if icon == "" {
		t.Error("RenderIcon should return non-empty string")
	}

	coloredIcon := service.RenderIconWithColor("error")
	if coloredIcon == "" {
		t.Error("RenderIconWithColor should return non-empty string")
	}

	iconSystem := service.GetIconSystem()
	if iconSystem == nil {
		t.Error("GetIconSystem should return non-nil icon system")
	}
}
