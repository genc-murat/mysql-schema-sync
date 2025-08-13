package display

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompactFormatterForScripting tests the compact formatter's suitability for automation and parsing
func TestCompactFormatterForScripting(t *testing.T) {
	t.Run("MachineReadableOutput", func(t *testing.T) {
		formatter := NewCompactFormatter()

		// Test consistent output format for sections
		result, err := formatter.FormatSection("Database Status", map[string]interface{}{
			"connected": true,
			"tables":    25,
			"version":   "8.0.33",
		})
		if err != nil {
			t.Fatalf("FormatSection failed: %v", err)
		}

		// Should start with SECTION: prefix for easy parsing
		if !strings.HasPrefix(result, "SECTION:Database Status:") {
			t.Errorf("Section output should start with 'SECTION:Database Status:', got: %s", result)
		}

		// Should contain all key-value pairs
		if !strings.Contains(result, "connected=true") {
			t.Errorf("Section output should contain 'connected=true', got: %s", result)
		}
		if !strings.Contains(result, "tables=25") {
			t.Errorf("Section output should contain 'tables=25', got: %s", result)
		}
		if !strings.Contains(result, "version=8.0.33") {
			t.Errorf("Section output should contain 'version=8.0.33', got: %s", result)
		}
	})

	t.Run("TabSeparatedValues", func(t *testing.T) {
		formatter := NewCompactFormatter()

		headers := []string{"Table", "Rows", "Size", "Status"}
		rows := [][]string{
			{"users", "1000", "2MB", "OK"},
			{"orders", "5000", "10MB", "OK"},
			{"logs", "50000", "100MB", "WARNING"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 4 { // header + 3 rows
			t.Errorf("Expected 4 lines, got %d", len(lines))
		}

		// Verify header line
		headerLine := lines[0]
		expectedHeader := "Table\tRows\tSize\tStatus"
		if headerLine != expectedHeader {
			t.Errorf("Expected header '%s', got '%s'", expectedHeader, headerLine)
		}

		// Verify data lines are tab-separated
		for i, expectedRow := range rows {
			actualRow := lines[i+1]
			expectedLine := strings.Join(expectedRow, "\t")
			if actualRow != expectedLine {
				t.Errorf("Expected row %d '%s', got '%s'", i, expectedLine, actualRow)
			}
		}
	})

	t.Run("SQLStatementParsing", func(t *testing.T) {
		formatter := NewCompactFormatter()

		statements := []string{
			"CREATE TABLE test (id INT PRIMARY KEY);",
			"INSERT INTO test VALUES (1);",
			"ALTER TABLE test ADD COLUMN name VARCHAR(50);",
		}

		result, err := formatter.FormatSQL(statements)
		if err != nil {
			t.Fatalf("FormatSQL failed: %v", err)
		}

		// Should start with SQL:count: prefix
		expectedPrefix := "SQL:3:"
		if !strings.HasPrefix(result, expectedPrefix) {
			t.Errorf("SQL output should start with '%s', got: %s", expectedPrefix, result)
		}

		// Should contain escaped statements separated by pipes
		sqlPart := strings.TrimPrefix(result, expectedPrefix)
		escapedStatements := strings.Split(sqlPart, "|")
		if len(escapedStatements) != 3 {
			t.Errorf("Expected 3 escaped statements, got %d", len(escapedStatements))
		}

		// Verify statements are properly escaped
		for i, stmt := range statements {
			expectedEscaped := strings.ReplaceAll(stmt, "|", "\\|")
			if escapedStatements[i] != expectedEscaped {
				t.Errorf("Expected escaped statement '%s', got '%s'", expectedEscaped, escapedStatements[i])
			}
		}
	})

	t.Run("StatusMessageParsing", func(t *testing.T) {
		formatter := NewCompactFormatter()

		testCases := []struct {
			level   string
			message string
		}{
			{"SUCCESS", "Migration completed successfully"},
			{"ERROR", "Connection failed: timeout"},
			{"WARNING", "Table 'old_table' will be dropped"},
			{"INFO", "Processing 1000 records"},
		}

		for _, tc := range testCases {
			result, err := formatter.FormatStatusMessage(tc.level, tc.message)
			if err != nil {
				t.Fatalf("FormatStatusMessage failed: %v", err)
			}

			expected := "STATUS:" + tc.level + ":" + tc.message
			if result != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result)
			}

			// Verify parsing
			parts := strings.SplitN(result, ":", 3)
			if len(parts) != 3 {
				t.Errorf("Expected 3 parts when splitting status message, got %d", len(parts))
			}
			if parts[0] != "STATUS" {
				t.Errorf("Expected first part to be 'STATUS', got '%s'", parts[0])
			}
			if parts[1] != tc.level {
				t.Errorf("Expected level '%s', got '%s'", tc.level, parts[1])
			}
			if parts[2] != tc.message {
				t.Errorf("Expected message '%s', got '%s'", tc.message, parts[2])
			}
		}
	})

	t.Run("SchemaDiffParsing", func(t *testing.T) {
		formatter := NewCompactFormatter()

		diff := map[string]interface{}{
			"added":    []string{"new_table1", "new_table2"},
			"removed":  []string{"old_table"},
			"modified": []interface{}{"users", "orders", "products"},
		}

		result, err := formatter.FormatSchemaDiff(diff)
		if err != nil {
			t.Fatalf("FormatSchemaDiff failed: %v", err)
		}

		// Should start with DIFF:schema: prefix
		if !strings.HasPrefix(result, "DIFF:schema:") {
			t.Errorf("Schema diff should start with 'DIFF:schema:', got: %s", result)
		}

		// Should contain counts for each change type
		if !strings.Contains(result, "added=2") {
			t.Errorf("Schema diff should contain 'added=2', got: %s", result)
		}
		if !strings.Contains(result, "removed=1") {
			t.Errorf("Schema diff should contain 'removed=1', got: %s", result)
		}
		if !strings.Contains(result, "modified=3") {
			t.Errorf("Schema diff should contain 'modified=3', got: %s", result)
		}
	})

	t.Run("CustomSeparatorForCSV", func(t *testing.T) {
		formatter := NewCompactFormatterWithOptions(",", true)

		headers := []string{"Name", "Email", "Status"}
		rows := [][]string{
			{"John Doe", "john@example.com", "Active"},
			{"Jane Smith", "jane@example.com", "Inactive"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable with CSV format failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 3 { // header + 2 rows
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}

		// Verify CSV format
		expectedHeader := "Name,Email,Status"
		if lines[0] != expectedHeader {
			t.Errorf("Expected CSV header '%s', got '%s'", expectedHeader, lines[0])
		}

		expectedRow1 := "John Doe,john@example.com,Active"
		if lines[1] != expectedRow1 {
			t.Errorf("Expected CSV row '%s', got '%s'", expectedRow1, lines[1])
		}
	})

	t.Run("HeaderlessOutputForParsing", func(t *testing.T) {
		formatter := NewCompactFormatterWithOptions("\t", false)

		headers := []string{"ID", "Name", "Count"}
		rows := [][]string{
			{"1", "Test1", "100"},
			{"2", "Test2", "200"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable without headers failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 2 { // only data rows, no header
			t.Errorf("Expected 2 lines (no header), got %d", len(lines))
		}

		// Verify no header line
		if lines[0] == "ID\tName\tCount" {
			t.Error("Should not include header line when includeHeaders is false")
		}

		// Verify data lines
		expectedRow1 := "1\tTest1\t100"
		if lines[0] != expectedRow1 {
			t.Errorf("Expected first row '%s', got '%s'", expectedRow1, lines[0])
		}
	})
}

// TestCompactFormatterIntegrationWithDisplayService tests the compact formatter integration
func TestCompactFormatterIntegrationWithDisplayService(t *testing.T) {
	var buf bytes.Buffer

	config := &DisplayConfig{
		ColorEnabled:    false,
		Theme:           "dark",
		OutputFormat:    string(FormatCompact),
		VerboseMode:     false,
		QuietMode:       false,
		InteractiveMode: false,
		Writer:          &buf,
	}

	service := NewDisplayService(config)

	t.Run("DisplayServiceCompactOutput", func(t *testing.T) {
		buf.Reset()

		// Test section output
		service.PrintSection("Migration Plan", map[string]interface{}{
			"total_steps":    5,
			"estimated_time": "2 minutes",
			"safe_mode":      true,
		})

		output := buf.String()
		if !strings.Contains(output, "SECTION:Migration Plan:") {
			t.Errorf("Expected section output to contain 'SECTION:Migration Plan:', got: %s", output)
		}
	})

	t.Run("DisplayServiceCompactTable", func(t *testing.T) {
		buf.Reset()

		headers := []string{"Step", "Action", "Status"}
		rows := [][]string{
			{"1", "Backup", "Complete"},
			{"2", "Migrate", "In Progress"},
			{"3", "Verify", "Pending"},
		}

		service.PrintTable(headers, rows)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 3 rows + extra newline from service
		if len(lines) < 4 {
			t.Errorf("Expected at least 4 lines, got %d", len(lines))
		}

		// Verify tab-separated format
		if !strings.Contains(lines[0], "Step\tAction\tStatus") {
			t.Errorf("Expected tab-separated header, got: %s", lines[0])
		}
	})

	t.Run("DisplayServiceCompactStatus", func(t *testing.T) {
		buf.Reset()

		service.Success("Operation completed successfully")

		output := strings.TrimSpace(buf.String())
		expected := "STATUS:SUCCESS:Operation completed successfully"
		if !strings.Contains(output, expected) {
			t.Errorf("Expected status output to contain '%s', got: %s", expected, output)
		}
	})
}

// BenchmarkCompactFormatter benchmarks the compact formatter performance
func BenchmarkCompactFormatter(b *testing.B) {
	formatter := NewCompactFormatter()

	b.Run("FormatTable", func(b *testing.B) {
		headers := []string{"ID", "Name", "Email", "Status", "Created", "Modified"}
		rows := make([][]string, 1000)
		for i := 0; i < 1000; i++ {
			rows[i] = []string{
				string(rune(i)),
				"User" + string(rune(i)),
				"user" + string(rune(i)) + "@example.com",
				"Active",
				"2023-01-01",
				"2023-12-31",
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := formatter.FormatTable(headers, rows)
			if err != nil {
				b.Fatalf("FormatTable failed: %v", err)
			}
		}
	})

	b.Run("FormatSQL", func(b *testing.B) {
		statements := make([]string, 100)
		for i := 0; i < 100; i++ {
			statements[i] = "CREATE TABLE table" + string(rune(i)) + " (id INT PRIMARY KEY, name VARCHAR(255));"
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := formatter.FormatSQL(statements)
			if err != nil {
				b.Fatalf("FormatSQL failed: %v", err)
			}
		}
	})
}
