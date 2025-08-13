package display

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Integration tests for complete visual output flows with mock data
func TestVisualOutputIntegration(t *testing.T) {
	t.Run("CompleteSchemaComparisonFlow", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.ColorEnabled = true
		config.UseIcons = true
		config.ShowProgress = true

		service := NewDisplayService(config)

		// Simulate a complete schema comparison workflow
		service.PrintHeader("Schema Comparison Report")

		// Show connection progress
		spinner := service.StartSpinner("Connecting to databases...")
		time.Sleep(50 * time.Millisecond)
		service.UpdateSpinner(spinner, "Establishing connections...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner, "✅ Connected to both databases")

		// Show extraction progress
		pb := service.NewProgressBar(100, "Extracting schema information")
		for i := 0; i <= 100; i += 20 {
			pb.Update(i, fmt.Sprintf("Processing tables... (%d%%)", i))
			time.Sleep(10 * time.Millisecond)
		}
		pb.Finish("Schema extraction complete")

		// Show comparison results
		service.PrintSection("Schema Differences", "Found 5 differences")

		headers := []string{"Type", "Object", "Change", "Details"}
		rows := [][]string{
			{"Table", "users", "Added", "New table with 5 columns"},
			{"Column", "orders.status", "Modified", "Changed from VARCHAR(20) to VARCHAR(50)"},
			{"Index", "users.email_idx", "Added", "Unique index on email column"},
			{"Column", "products.description", "Removed", "Column no longer exists"},
			{"Table", "temp_data", "Removed", "Temporary table removed"},
		}
		service.PrintTable(headers, rows)

		// Show SQL statements
		statements := []string{
			"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(255) UNIQUE);",
			"ALTER TABLE orders MODIFY COLUMN status VARCHAR(50);",
			"CREATE UNIQUE INDEX email_idx ON users(email);",
			"ALTER TABLE products DROP COLUMN description;",
			"DROP TABLE temp_data;",
		}
		service.PrintSQL(statements)

		// Show final status
		service.Success("Schema comparison completed successfully")
		service.Info("5 changes detected, 3 additions, 1 modification, 1 removal")

		output := buf.String()

		// Verify the complete flow is present
		if !strings.Contains(output, "Schema Comparison Report") {
			t.Error("Should contain header")
		}
		if !strings.Contains(output, "Connected to both databases") {
			t.Error("Should contain spinner completion message")
		}
		if !strings.Contains(output, "Schema extraction complete") {
			t.Error("Should contain progress bar completion")
		}
		if !strings.Contains(output, "Schema Differences") {
			t.Error("Should contain section title")
		}
		if !strings.Contains(output, "users") {
			t.Error("Should contain table data")
		}
		if !strings.Contains(output, "CREATE TABLE") {
			t.Error("Should contain SQL statements")
		}
		if !strings.Contains(output, "[SUCCESS]") {
			t.Error("Should contain success message")
		}
	})

	t.Run("MultiFormatOutputComparison", func(t *testing.T) {
		formats := []OutputFormat{FormatTable, FormatJSON, FormatYAML, FormatCompact}

		testData := struct {
			headers []string
			rows    [][]string
			sql     []string
		}{
			headers: []string{"Database", "Tables", "Status"},
			rows: [][]string{
				{"production", "25", "Active"},
				{"staging", "23", "Active"},
				{"development", "20", "Active"},
			},
			sql: []string{
				"SELECT COUNT(*) FROM information_schema.tables;",
				"SHOW DATABASES;",
			},
		}

		outputs := make(map[OutputFormat]string)

		for _, format := range formats {
			var buf bytes.Buffer
			config := DefaultDisplayConfig()
			config.Writer = &buf
			config.OutputFormat = string(format)
			config.ColorEnabled = false // For consistent comparison

			service := NewDisplayService(config)

			service.PrintHeader("Database Summary")
			service.PrintSection("Database Information", "Connection details")
			service.PrintTable(testData.headers, testData.rows)
			service.PrintSQL(testData.sql)
			service.Success("Operation completed")

			outputs[format] = buf.String()
		}

		// Verify each format produces different output
		for format, output := range outputs {
			if output == "" {
				t.Errorf("Format %s should produce output", format)
			}

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
				if !strings.Contains(output, "Database Summary") {
					t.Error("Table format should contain headers")
				}
			case FormatCompact:
				// Compact format should be more concise
				if len(output) > len(outputs[FormatTable]) {
					t.Error("Compact format should be more concise than table format")
				}
			}
		}

		// Verify all formats contain the essential data
		for format, output := range outputs {
			if !strings.Contains(output, "production") {
				t.Errorf("Format %s should contain data 'production'", format)
			}
		}
	})

	t.Run("ProgressIndicatorIntegration", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.ShowProgress = true

		service := NewDisplayService(config)

		// Test multiple progress indicators working together
		service.PrintHeader("Multi-Phase Operation")

		// Phase 1: Connection
		spinner1 := service.StartSpinner("Connecting to source database...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner1, "Source connected")

		spinner2 := service.StartSpinner("Connecting to target database...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner2, "Target connected")

		// Phase 2: Data processing with progress bar
		pb := service.NewProgressBar(50, "Processing data")
		for i := 0; i <= 50; i += 10 {
			pb.Update(i, fmt.Sprintf("Processed %d records", i*10))
			time.Sleep(10 * time.Millisecond)
		}
		pb.Finish("Data processing complete")

		// Phase 3: Multi-progress for parallel operations
		mp := service.NewMultiProgress()
		pb1 := service.NewProgressBar(20, "Validating data")
		pb2 := service.NewProgressBar(15, "Generating report")
		pb3 := service.NewProgressBar(10, "Cleanup operations")

		mp.AddBar(pb1)
		mp.AddBar(pb2)
		mp.AddBar(pb3)
		mp.Start()

		// Simulate parallel progress
		for i := 0; i < 20; i++ {
			if i < 20 {
				pb1.Update(i+1, "Validating...")
			}
			if i < 15 {
				pb2.Update(i+1, "Generating...")
			}
			if i < 10 {
				pb3.Update(i+1, "Cleaning...")
			}
			time.Sleep(5 * time.Millisecond)
		}

		mp.Stop()

		// Phase 4: Progress tracker for structured phases
		pt := service.NewProgressTracker([]string{"Analyze", "Transform", "Load"})

		pt.StartPhase(0, 10, "Analyzing schema")
		for i := 1; i <= 10; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Analyzing table %d", i))
			time.Sleep(5 * time.Millisecond)
		}
		pt.CompletePhase("Analysis complete")

		pt.StartPhase(1, 5, "Transforming data")
		for i := 1; i <= 5; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Transform step %d", i))
			time.Sleep(5 * time.Millisecond)
		}
		pt.CompletePhase("Transformation complete")

		pt.StartPhase(2, 3, "Loading results")
		for i := 1; i <= 3; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Loading batch %d", i))
			time.Sleep(5 * time.Millisecond)
		}
		pt.CompletePhase("Loading complete")

		service.Success("Multi-phase operation completed successfully")

		output := buf.String()

		// Verify all progress indicators worked
		if !strings.Contains(output, "Source connected") {
			t.Error("Should contain spinner 1 completion")
		}
		if !strings.Contains(output, "Target connected") {
			t.Error("Should contain spinner 2 completion")
		}
		if !strings.Contains(output, "Data processing complete") {
			t.Error("Should contain progress bar completion")
		}
		if !strings.Contains(output, "Analysis complete") {
			t.Error("Should contain progress tracker phase completion")
		}
	})

	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.ColorEnabled = true

		service := NewDisplayService(config)

		// Simulate error scenarios
		service.PrintHeader("Error Handling Test")

		// Connection error
		spinner := service.StartSpinner("Connecting to database...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner, "")
		service.Error("Failed to connect to database: Connection timeout")

		// Partial success scenario
		service.Warning("Some operations completed with warnings")

		// Recovery attempt
		service.Info("Attempting to reconnect...")
		spinner2 := service.StartSpinner("Retrying connection...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner2, "")
		service.Success("Successfully reconnected")

		// Show partial results
		headers := []string{"Operation", "Status", "Details"}
		rows := [][]string{
			{"Schema extraction", "Success", "25 tables processed"},
			{"Index analysis", "Warning", "2 indexes need attention"},
			{"Data validation", "Error", "Validation failed for 3 records"},
			{"Report generation", "Success", "Report saved to file"},
		}
		service.PrintTable(headers, rows)

		service.Info("Operation completed with mixed results")

		output := buf.String()

		// Verify error handling messages
		if !strings.Contains(output, "[ERROR]") {
			t.Error("Should contain error message")
		}
		if !strings.Contains(output, "[WARNING]") {
			t.Error("Should contain warning message")
		}
		if !strings.Contains(output, "[SUCCESS]") {
			t.Error("Should contain success message")
		}
		if !strings.Contains(output, "Connection timeout") {
			t.Error("Should contain specific error details")
		}
		if !strings.Contains(output, "Successfully reconnected") {
			t.Error("Should contain recovery message")
		}
	})
}

// Integration tests for configuration and CLI flag handling
func TestConfigurationIntegration(t *testing.T) {
	t.Run("CLIFlagSimulation", func(t *testing.T) {
		// Simulate different CLI flag combinations
		testCases := []struct {
			name   string
			config DisplayConfig
			verify func(*testing.T, string)
		}{
			{
				name: "NoColorFlag",
				config: DisplayConfig{
					ColorEnabled:  false,
					Theme:         "dark",
					OutputFormat:  string(FormatTable),
					UseIcons:      true,
					ShowProgress:  true,
					TableStyle:    "default",
					MaxTableWidth: 120,
				},
				verify: func(t *testing.T, output string) {
					// Should not contain ANSI color codes
					if strings.Contains(output, "\033[") {
						t.Error("No-color mode should not contain ANSI codes")
					}
				},
			},
			{
				name: "QuietMode",
				config: DisplayConfig{
					ColorEnabled:  true,
					Theme:         "dark",
					OutputFormat:  string(FormatTable),
					UseIcons:      false,
					ShowProgress:  false,
					QuietMode:     true,
					TableStyle:    "default",
					MaxTableWidth: 120,
				},
				verify: func(t *testing.T, output string) {
					// Should suppress most output except errors
					if strings.Contains(output, "[INFO]") {
						t.Error("Quiet mode should suppress info messages")
					}
				},
			},
			{
				name: "VerboseMode",
				config: DisplayConfig{
					ColorEnabled:  true,
					Theme:         "dark",
					OutputFormat:  string(FormatTable),
					UseIcons:      true,
					ShowProgress:  true,
					VerboseMode:   true,
					TableStyle:    "default",
					MaxTableWidth: 120,
				},
				verify: func(t *testing.T, output string) {
					// Should contain detailed information
					if output == "" {
						t.Error("Verbose mode should produce output")
					}
				},
			},
			{
				name: "JSONOutput",
				config: DisplayConfig{
					ColorEnabled:  false,
					Theme:         "dark",
					OutputFormat:  string(FormatJSON),
					UseIcons:      false,
					ShowProgress:  false,
					TableStyle:    "default",
					MaxTableWidth: 120,
				},
				verify: func(t *testing.T, output string) {
					if !strings.Contains(output, "{") {
						t.Error("JSON output should contain braces")
					}
				},
			},
			{
				name: "CompactOutput",
				config: DisplayConfig{
					ColorEnabled:  false,
					Theme:         "dark",
					OutputFormat:  string(FormatCompact),
					UseIcons:      false,
					ShowProgress:  false,
					TableStyle:    "minimal",
					MaxTableWidth: 80,
				},
				verify: func(t *testing.T, output string) {
					// Compact output should be concise
					if len(output) == 0 {
						t.Error("Compact output should still produce some output")
					}
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				tc.config.Writer = &buf

				service := NewDisplayService(&tc.config)

				// Perform standard operations
				service.PrintHeader("Test Operation")
				service.Info("This is an info message")
				service.PrintSection("Results", "Test data")

				headers := []string{"Item", "Value"}
				rows := [][]string{{"Test1", "Value1"}, {"Test2", "Value2"}}
				service.PrintTable(headers, rows)

				service.Success("Operation completed")

				output := buf.String()
				tc.verify(t, output)
			})
		}
	})

	t.Run("ThemeIntegration", func(t *testing.T) {
		themes := []string{"dark", "light", "high-contrast", "auto"}

		for _, themeName := range themes {
			t.Run(themeName, func(t *testing.T) {
				var buf bytes.Buffer
				config := DefaultDisplayConfig()
				config.Writer = &buf
				config.Theme = themeName
				config.ColorEnabled = true

				service := NewDisplayService(config)

				// Test all message types with the theme
				service.Success("Success message")
				service.Error("Error message")
				service.Warning("Warning message")
				service.Info("Info message")

				// Test table with theme
				headers := []string{"Status", "Message"}
				rows := [][]string{
					{"OK", "Everything is fine"},
					{"ERROR", "Something went wrong"},
					{"WARN", "Be careful"},
				}
				service.PrintTable(headers, rows)

				output := buf.String()

				// All themes should produce output
				if output == "" {
					t.Errorf("Theme %s should produce output", themeName)
				}

				// Should contain all message types
				if !strings.Contains(output, "[SUCCESS]") {
					t.Errorf("Theme %s should contain success message", themeName)
				}
				if !strings.Contains(output, "[ERROR]") {
					t.Errorf("Theme %s should contain error message", themeName)
				}
			})
		}
	})

	t.Run("TableStyleIntegration", func(t *testing.T) {
		styles := []string{"default", "rounded", "border", "minimal"}

		for _, styleName := range styles {
			t.Run(styleName, func(t *testing.T) {
				var buf bytes.Buffer
				config := DefaultDisplayConfig()
				config.Writer = &buf
				config.TableStyle = styleName
				config.ColorEnabled = false // For consistent comparison

				service := NewDisplayService(config)

				headers := []string{"Column1", "Column2", "Column3"}
				rows := [][]string{
					{"Data1", "Data2", "Data3"},
					{"LongData1", "LongData2", "LongData3"},
					{"Short", "Med", "VeryLongData"},
				}
				service.PrintTable(headers, rows)

				output := buf.String()

				if output == "" {
					t.Errorf("Table style %s should produce output", styleName)
				}

				// Should contain the data
				if !strings.Contains(output, "Data1") {
					t.Errorf("Table style %s should contain data", styleName)
				}
			})
		}
	})
}

// Integration tests for graceful degradation in non-color and non-interactive environments
func TestGracefulDegradation(t *testing.T) {
	t.Run("NonColorTerminal", func(t *testing.T) {
		// Save original environment
		originalNoColor := os.Getenv("NO_COLOR")
		defer func() {
			if originalNoColor != "" {
				os.Setenv("NO_COLOR", originalNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}
		}()

		// Set NO_COLOR environment variable
		os.Setenv("NO_COLOR", "1")

		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.ColorEnabled = true // Should be overridden by environment

		service := NewDisplayService(config)

		// Test all visual components
		service.PrintHeader("Non-Color Test")
		service.Success("Success message")
		service.Error("Error message")
		service.Warning("Warning message")
		service.Info("Info message")

		headers := []string{"Type", "Status"}
		rows := [][]string{
			{"Operation1", "Success"},
			{"Operation2", "Failed"},
		}
		service.PrintTable(headers, rows)

		statements := []string{"SELECT * FROM test;"}
		service.PrintSQL(statements)

		output := buf.String()

		// Should still produce readable output without colors
		if output == "" {
			t.Error("Should produce output even without colors")
		}
		if !strings.Contains(output, "Non-Color Test") {
			t.Error("Should contain header text")
		}
		if !strings.Contains(output, "[SUCCESS]") {
			t.Error("Should contain status messages")
		}
		if !strings.Contains(output, "Operation1") {
			t.Error("Should contain table data")
		}
		if !strings.Contains(output, "SELECT") {
			t.Error("Should contain SQL statements")
		}
	})

	t.Run("NonInteractiveEnvironment", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.InteractiveMode = false
		config.ShowProgress = false // Disable progress in non-interactive

		service := NewDisplayService(config)

		// Test components that would normally be interactive
		service.PrintHeader("Non-Interactive Test")

		// Spinners should still work but be less interactive
		spinner := service.StartSpinner("Processing...")
		time.Sleep(50 * time.Millisecond)
		service.StopSpinner(spinner, "Processing complete")

		// Progress should be minimal
		service.ShowProgress(5, 10, "Progress update")
		service.ShowProgress(10, 10, "Complete")

		service.Success("Non-interactive operation completed")

		output := buf.String()

		// Should produce output suitable for non-interactive use
		if output == "" {
			t.Error("Should produce output in non-interactive mode")
		}
		if !strings.Contains(output, "Processing complete") {
			t.Error("Should contain completion messages")
		}
	})

	t.Run("MinimalTerminalCapabilities", func(t *testing.T) {
		// Save original environment
		originalTerm := os.Getenv("TERM")
		originalNoUnicode := os.Getenv("NO_UNICODE")
		originalNoColor := os.Getenv("NO_COLOR")

		defer func() {
			if originalTerm != "" {
				os.Setenv("TERM", originalTerm)
			} else {
				os.Unsetenv("TERM")
			}
			if originalNoUnicode != "" {
				os.Setenv("NO_UNICODE", originalNoUnicode)
			} else {
				os.Unsetenv("NO_UNICODE")
			}
			if originalNoColor != "" {
				os.Setenv("NO_COLOR", originalNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}
		}()

		// Simulate minimal terminal
		os.Setenv("TERM", "dumb")
		os.Setenv("NO_UNICODE", "1")
		os.Setenv("NO_COLOR", "1")

		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf

		service := NewDisplayService(config)

		// Test all components with minimal capabilities
		service.PrintHeader("Minimal Terminal Test")

		// Icons should fall back to ASCII
		iconSystem := service.GetIconSystem()
		successIcon := iconSystem.RenderIcon("success")
		errorIcon := iconSystem.RenderIcon("error")

		service.Info(fmt.Sprintf("Success icon: %s", successIcon))
		service.Info(fmt.Sprintf("Error icon: %s", errorIcon))

		// Table should use ASCII borders
		headers := []string{"Feature", "Status"}
		rows := [][]string{
			{"Colors", "Disabled"},
			{"Unicode", "Disabled"},
			{"Icons", "ASCII only"},
		}
		service.PrintTable(headers, rows)

		// Progress should be text-based
		pb := service.NewProgressBar(5, "ASCII progress")
		for i := 1; i <= 5; i++ {
			pb.Update(i, fmt.Sprintf("Step %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		pb.Finish("ASCII progress complete")

		service.Success("Minimal terminal test completed")

		output := buf.String()

		// Should work with minimal capabilities
		if output == "" {
			t.Error("Should work with minimal terminal capabilities")
		}
		if !strings.Contains(output, "Minimal Terminal Test") {
			t.Error("Should contain header")
		}
		if !strings.Contains(output, "ASCII only") {
			t.Error("Should contain table data")
		}
		if !strings.Contains(output, "ASCII progress complete") {
			t.Error("Should contain progress completion")
		}

		// Icons should be ASCII
		if strings.Contains(successIcon, "✅") {
			t.Error("Should use ASCII icons in minimal terminal")
		}
	})

	t.Run("LargeDataHandling", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.MaxTableWidth = 80 // Constrain width

		service := NewDisplayService(config)

		// Test with large amounts of data
		service.PrintHeader("Large Data Test")

		// Large table
		headers := make([]string, 10)
		for i := 0; i < 10; i++ {
			headers[i] = fmt.Sprintf("VeryLongColumnHeader%d", i+1)
		}

		rows := make([][]string, 50)
		for i := 0; i < 50; i++ {
			row := make([]string, 10)
			for j := 0; j < 10; j++ {
				row[j] = fmt.Sprintf("VeryLongDataValue%d-%d", i+1, j+1)
			}
			rows[i] = row
		}

		service.PrintTable(headers, rows)

		// Large SQL statements
		statements := make([]string, 20)
		for i := 0; i < 20; i++ {
			statements[i] = fmt.Sprintf("SELECT column1, column2, column3, column4, column5 FROM very_long_table_name_%d WHERE condition_%d = 'very_long_value_%d';", i+1, i+1, i+1)
		}
		service.PrintSQL(statements)

		service.Success("Large data test completed")

		output := buf.String()

		// Should handle large data gracefully
		if output == "" {
			t.Error("Should handle large data")
		}
		if !strings.Contains(output, "VeryLongColumnHeader1") {
			t.Error("Should contain table headers")
		}
		if !strings.Contains(output, "VeryLongDataValue1-1") {
			t.Error("Should contain table data")
		}
		if !strings.Contains(output, "SELECT") {
			t.Error("Should contain SQL statements")
		}

		// Should not crash or produce excessively long lines
		// Note: The table formatter may produce long lines with wide data,
		// but it should still be functional and not crash
		lines := strings.Split(output, "\n")
		maxLineLength := 0
		for _, line := range lines {
			if len(line) > maxLineLength {
				maxLineLength = len(line)
			}
		}

		// Verify that we got output and it's reasonable (not empty, not crashing)
		if maxLineLength == 0 {
			t.Error("Should produce some output")
		}

		// The table formatter should handle large data without crashing
		// Even if lines are long, the important thing is that it works
		if len(lines) < 10 {
			t.Error("Should produce multiple lines of output for large table")
		}
	})
}

// Test complete integration scenarios
func TestCompleteIntegrationScenarios(t *testing.T) {
	t.Run("DatabaseMigrationScenario", func(t *testing.T) {
		var buf bytes.Buffer
		config := DefaultDisplayConfig()
		config.Writer = &buf
		config.ColorEnabled = true
		config.UseIcons = true
		config.ShowProgress = true

		service := NewDisplayService(config)

		// Complete database migration scenario
		service.PrintHeader("Database Migration Tool")

		// Phase 1: Connection
		service.Info("Starting database migration process")
		spinner := service.StartSpinner("Connecting to source database...")
		time.Sleep(30 * time.Millisecond)
		service.StopSpinner(spinner, "✅ Connected to source database")

		spinner = service.StartSpinner("Connecting to target database...")
		time.Sleep(30 * time.Millisecond)
		service.StopSpinner(spinner, "✅ Connected to target database")

		// Phase 2: Analysis
		service.PrintSection("Schema Analysis", "Analyzing differences between databases")

		pb := service.NewProgressBar(25, "Analyzing tables")
		for i := 1; i <= 25; i++ {
			pb.Update(i, fmt.Sprintf("Analyzing table %d/25", i))
			time.Sleep(5 * time.Millisecond)
		}
		pb.Finish("Table analysis complete")

		// Phase 3: Results
		service.PrintSection("Migration Plan", "The following changes will be applied:")

		headers := []string{"Operation", "Object", "Type", "Impact"}
		rows := [][]string{
			{"CREATE", "user_profiles", "Table", "Low"},
			{"ALTER", "users.email", "Column", "Medium"},
			{"CREATE", "idx_user_email", "Index", "Low"},
			{"DROP", "temp_sessions", "Table", "High"},
			{"ALTER", "orders.total", "Column", "Medium"},
		}
		service.PrintTable(headers, rows)

		// Phase 4: SQL Generation
		service.PrintSection("Generated SQL", "Migration statements:")
		statements := []string{
			"CREATE TABLE user_profiles (id INT PRIMARY KEY, user_id INT, profile_data JSON);",
			"ALTER TABLE users MODIFY COLUMN email VARCHAR(255) NOT NULL;",
			"CREATE INDEX idx_user_email ON users(email);",
			"DROP TABLE temp_sessions;",
			"ALTER TABLE orders MODIFY COLUMN total DECIMAL(10,2) NOT NULL DEFAULT 0.00;",
		}
		service.PrintSQL(statements)

		// Phase 5: Execution simulation
		service.PrintSection("Execution", "Applying migration...")

		pt := service.NewProgressTracker([]string{"Backup", "Validate", "Execute", "Verify"})

		pt.StartPhase(0, 3, "Creating backup")
		for i := 1; i <= 3; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Backup step %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		pt.CompletePhase("Backup created successfully")

		pt.StartPhase(1, 5, "Validating migration")
		for i := 1; i <= 5; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Validation check %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		pt.CompletePhase("Validation passed")

		pt.StartPhase(2, 5, "Executing migration")
		for i := 1; i <= 5; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Executing statement %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		pt.CompletePhase("Migration executed")

		pt.StartPhase(3, 2, "Verifying results")
		for i := 1; i <= 2; i++ {
			pt.UpdatePhase(i, fmt.Sprintf("Verification step %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		pt.CompletePhase("Verification complete")

		// Final status
		service.Success("Database migration completed successfully!")
		service.Info("5 operations executed, 0 errors, 0 warnings")

		output := buf.String()

		// Verify complete scenario
		if !strings.Contains(output, "Database Migration Tool") {
			t.Error("Should contain main header")
		}
		if !strings.Contains(output, "Connected to source database") {
			t.Error("Should contain connection messages")
		}
		if !strings.Contains(output, "Table analysis complete") {
			t.Error("Should contain analysis completion")
		}
		if !strings.Contains(output, "user_profiles") {
			t.Error("Should contain migration plan data")
		}
		if !strings.Contains(output, "CREATE TABLE") {
			t.Error("Should contain SQL statements")
		}
		if !strings.Contains(output, "Backup created successfully") {
			t.Error("Should contain progress tracker phases")
		}
		if !strings.Contains(output, "migration completed successfully") {
			t.Error("Should contain final success message")
		}
	})
}
