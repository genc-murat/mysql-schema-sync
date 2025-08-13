package display

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()

	t.Run("FormatSection", func(t *testing.T) {
		result, err := formatter.FormatSection("Test Section", "test content")
		if err != nil {
			t.Fatalf("FormatSection failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["section"] != "Test Section" {
			t.Errorf("Expected section 'Test Section', got %v", data["section"])
		}
		if data["content"] != "test content" {
			t.Errorf("Expected content 'test content', got %v", data["content"])
		}
	})

	t.Run("FormatTable", func(t *testing.T) {
		headers := []string{"Name", "Age", "City"}
		rows := [][]string{
			{"John", "30", "New York"},
			{"Jane", "25", "Los Angeles"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable failed: %v", err)
		}

		var data []map[string]string
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if len(data) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(data))
		}

		if data[0]["Name"] != "John" || data[0]["Age"] != "30" || data[0]["City"] != "New York" {
			t.Errorf("First row data incorrect: %v", data[0])
		}
	})

	t.Run("FormatSQL", func(t *testing.T) {
		statements := []string{
			"CREATE TABLE users (id INT PRIMARY KEY);",
			"INSERT INTO users (id) VALUES (1);",
		}

		result, err := formatter.FormatSQL(statements)
		if err != nil {
			t.Fatalf("FormatSQL failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		sqlStatements, ok := data["sql_statements"].([]interface{})
		if !ok {
			t.Fatalf("Expected sql_statements array, got %T", data["sql_statements"])
		}

		if len(sqlStatements) != 2 {
			t.Errorf("Expected 2 statements, got %d", len(sqlStatements))
		}

		count, ok := data["count"].(float64)
		if !ok || int(count) != 2 {
			t.Errorf("Expected count 2, got %v", data["count"])
		}
	})

	t.Run("FormatStatusMessage", func(t *testing.T) {
		result, err := formatter.FormatStatusMessage("ERROR", "Something went wrong")
		if err != nil {
			t.Fatalf("FormatStatusMessage failed: %v", err)
		}

		var data map[string]string
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["level"] != "ERROR" {
			t.Errorf("Expected level 'ERROR', got %v", data["level"])
		}
		if data["message"] != "Something went wrong" {
			t.Errorf("Expected message 'Something went wrong', got %v", data["message"])
		}
	})

	t.Run("FormatSchemaDiff", func(t *testing.T) {
		diff := map[string]interface{}{
			"added":    []string{"table1", "table2"},
			"removed":  []string{"table3"},
			"modified": []string{"table4"},
		}

		result, err := formatter.FormatSchemaDiff(diff)
		if err != nil {
			t.Fatalf("FormatSchemaDiff failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		added, ok := data["added"].([]interface{})
		if !ok || len(added) != 2 {
			t.Errorf("Expected 2 added items, got %v", data["added"])
		}
	})
}

func TestYAMLFormatter(t *testing.T) {
	formatter := NewYAMLFormatter()

	t.Run("FormatSection", func(t *testing.T) {
		result, err := formatter.FormatSection("Test Section", "test content")
		if err != nil {
			t.Fatalf("FormatSection failed: %v", err)
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid YAML output: %v", err)
		}

		if data["section"] != "Test Section" {
			t.Errorf("Expected section 'Test Section', got %v", data["section"])
		}
		if data["content"] != "test content" {
			t.Errorf("Expected content 'test content', got %v", data["content"])
		}
	})

	t.Run("FormatTable", func(t *testing.T) {
		headers := []string{"Name", "Age"}
		rows := [][]string{
			{"John", "30"},
			{"Jane", "25"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable failed: %v", err)
		}

		var data []map[string]string
		if err := yaml.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid YAML output: %v", err)
		}

		if len(data) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(data))
		}

		if data[0]["Name"] != "John" || data[0]["Age"] != "30" {
			t.Errorf("First row data incorrect: %v", data[0])
		}
	})

	t.Run("FormatSQL", func(t *testing.T) {
		statements := []string{"CREATE TABLE test;", "DROP TABLE old;"}

		result, err := formatter.FormatSQL(statements)
		if err != nil {
			t.Fatalf("FormatSQL failed: %v", err)
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid YAML output: %v", err)
		}

		sqlStatements, ok := data["sql_statements"].([]interface{})
		if !ok {
			t.Fatalf("Expected sql_statements array, got %T", data["sql_statements"])
		}

		if len(sqlStatements) != 2 {
			t.Errorf("Expected 2 statements, got %d", len(sqlStatements))
		}
	})
}

func TestCompactFormatter(t *testing.T) {
	formatter := NewCompactFormatter()

	t.Run("FormatSection", func(t *testing.T) {
		// Test with string content
		result, err := formatter.FormatSection("Test Section", "test content")
		if err != nil {
			t.Fatalf("FormatSection failed: %v", err)
		}

		expected := "SECTION:Test Section:test content"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		// Test with map content
		mapContent := map[string]interface{}{
			"tables": 5,
			"status": "ok",
		}
		result, err = formatter.FormatSection("Database", mapContent)
		if err != nil {
			t.Fatalf("FormatSection with map failed: %v", err)
		}

		if !strings.HasPrefix(result, "SECTION:Database:") {
			t.Errorf("Expected result to start with 'SECTION:Database:', got '%s'", result)
		}
		if !strings.Contains(result, "tables=5") {
			t.Errorf("Expected result to contain 'tables=5', got '%s'", result)
		}
		if !strings.Contains(result, "status=ok") {
			t.Errorf("Expected result to contain 'status=ok', got '%s'", result)
		}
	})

	t.Run("FormatTable", func(t *testing.T) {
		headers := []string{"Name", "Age"}
		rows := [][]string{
			{"John", "30"},
			{"Jane", "25"},
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 3 { // header + 2 rows
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}

		if lines[0] != "Name\tAge" {
			t.Errorf("Expected header 'Name\\tAge', got '%s'", lines[0])
		}

		if lines[1] != "John\t30" {
			t.Errorf("Expected first row 'John\\t30', got '%s'", lines[1])
		}
	})

	t.Run("FormatTableWithMismatchedRows", func(t *testing.T) {
		headers := []string{"Name", "Age", "City"}
		rows := [][]string{
			{"John", "30"},                // Missing city
			{"Jane", "25", "LA", "Extra"}, // Extra field
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 3 { // header + 2 rows
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}

		// First row should have empty city
		if lines[1] != "John\t30\t" {
			t.Errorf("Expected first row 'John\\t30\\t', got '%s'", lines[1])
		}

		// Second row should ignore extra field
		if lines[2] != "Jane\t25\tLA" {
			t.Errorf("Expected second row 'Jane\\t25\\tLA', got '%s'", lines[2])
		}
	})

	t.Run("FormatSQL", func(t *testing.T) {
		statements := []string{"CREATE TABLE test;", "DROP TABLE old;"}

		result, err := formatter.FormatSQL(statements)
		if err != nil {
			t.Fatalf("FormatSQL failed: %v", err)
		}

		expected := "SQL:2:CREATE TABLE test;|DROP TABLE old;"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		// Test empty statements
		result, err = formatter.FormatSQL([]string{})
		if err != nil {
			t.Fatalf("FormatSQL with empty statements failed: %v", err)
		}

		expected = "SQL:0:"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("FormatStatusMessage", func(t *testing.T) {
		result, err := formatter.FormatStatusMessage("INFO", "Operation completed")
		if err != nil {
			t.Fatalf("FormatStatusMessage failed: %v", err)
		}

		expected := "STATUS:INFO:Operation completed"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("FormatSchemaDiff", func(t *testing.T) {
		diff := map[string]interface{}{
			"added":    []string{"table1", "table2"},
			"removed":  []string{"table3"},
			"modified": []interface{}{"table4", "table5"},
		}

		result, err := formatter.FormatSchemaDiff(diff)
		if err != nil {
			t.Fatalf("FormatSchemaDiff failed: %v", err)
		}

		if !strings.HasPrefix(result, "DIFF:schema:") {
			t.Errorf("Expected result to start with 'DIFF:schema:', got '%s'", result)
		}
		if !strings.Contains(result, "added=2") {
			t.Errorf("Expected result to contain 'added=2', got '%s'", result)
		}
		if !strings.Contains(result, "removed=1") {
			t.Errorf("Expected result to contain 'removed=1', got '%s'", result)
		}
		if !strings.Contains(result, "modified=2") {
			t.Errorf("Expected result to contain 'modified=2', got '%s'", result)
		}
	})

	t.Run("CustomOptions", func(t *testing.T) {
		customFormatter := NewCompactFormatterWithOptions(",", false)

		headers := []string{"Name", "Age"}
		rows := [][]string{{"John", "30"}}

		result, err := customFormatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable with custom options failed: %v", err)
		}

		// Should only have one line (no headers) with comma separator
		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 1 {
			t.Errorf("Expected 1 line, got %d", len(lines))
		}

		if lines[0] != "John,30" {
			t.Errorf("Expected 'John,30', got '%s'", lines[0])
		}

		// Test getter methods
		if customFormatter.GetSeparator() != "," {
			t.Errorf("Expected separator ',', got '%s'", customFormatter.GetSeparator())
		}

		if customFormatter.GetIncludeHeaders() != false {
			t.Errorf("Expected includeHeaders false, got %v", customFormatter.GetIncludeHeaders())
		}

		// Test setter methods
		customFormatter.SetSeparator("|")
		customFormatter.SetIncludeHeaders(true)

		result, err = customFormatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable after setters failed: %v", err)
		}

		lines = strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 2 { // header + 1 row
			t.Errorf("Expected 2 lines, got %d", len(lines))
		}

		if lines[0] != "Name|Age" {
			t.Errorf("Expected header 'Name|Age', got '%s'", lines[0])
		}

		if lines[1] != "John|30" {
			t.Errorf("Expected row 'John|30', got '%s'", lines[1])
		}
	})
}

func TestFormatterRegistry(t *testing.T) {
	registry := NewFormatterRegistry()

	t.Run("GetFormatter", func(t *testing.T) {
		// Test getting existing formatters
		jsonFormatter, exists := registry.GetFormatter(FormatJSON)
		if !exists {
			t.Error("JSON formatter should exist")
		}
		if jsonFormatter == nil {
			t.Error("JSON formatter should not be nil")
		}

		yamlFormatter, exists := registry.GetFormatter(FormatYAML)
		if !exists {
			t.Error("YAML formatter should exist")
		}
		if yamlFormatter == nil {
			t.Error("YAML formatter should not be nil")
		}

		compactFormatter, exists := registry.GetFormatter(FormatCompact)
		if !exists {
			t.Error("Compact formatter should exist")
		}
		if compactFormatter == nil {
			t.Error("Compact formatter should not be nil")
		}

		// Test getting non-existent formatter
		_, exists = registry.GetFormatter(FormatTable)
		if exists {
			t.Error("Table formatter should not exist in registry")
		}
	})

	t.Run("Register", func(t *testing.T) {
		customFormatter := NewJSONFormatter()
		registry.Register("custom", customFormatter)

		formatter, exists := registry.GetFormatter("custom")
		if !exists {
			t.Error("Custom formatter should exist after registration")
		}
		if formatter != customFormatter {
			t.Error("Retrieved formatter should be the same as registered")
		}
	})

	t.Run("FormatOutput", func(t *testing.T) {
		// Test section formatting
		sectionData := map[string]interface{}{
			"title":   "Test Section",
			"content": "test content",
		}

		result, err := registry.FormatOutput(FormatJSON, "section", sectionData)
		if err != nil {
			t.Fatalf("FormatOutput failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["section"] != "Test Section" {
			t.Errorf("Expected section 'Test Section', got %v", data["section"])
		}

		// Test table formatting
		tableData := map[string]interface{}{
			"headers": []string{"Name", "Age"},
			"rows":    [][]string{{"John", "30"}},
		}

		result, err = registry.FormatOutput(FormatCompact, "table", tableData)
		if err != nil {
			t.Fatalf("FormatOutput failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(lines))
		}

		// Test SQL formatting
		sqlData := []string{"CREATE TABLE test;"}
		result, err = registry.FormatOutput(FormatYAML, "sql", sqlData)
		if err != nil {
			t.Fatalf("FormatOutput failed: %v", err)
		}

		var yamlData map[string]interface{}
		if err := yaml.Unmarshal([]byte(result), &yamlData); err != nil {
			t.Fatalf("Invalid YAML output: %v", err)
		}

		// Test unsupported format
		_, err = registry.FormatOutput("unsupported", "section", sectionData)
		if err == nil {
			t.Error("Expected error for unsupported format")
		}

		// Test unsupported output type
		_, err = registry.FormatOutput(FormatJSON, "unsupported", sectionData)
		if err == nil {
			t.Error("Expected error for unsupported output type")
		}
	})
}

func TestOutputWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewOutputWriter(FormatJSON, &buf)

	t.Run("WriteSection", func(t *testing.T) {
		buf.Reset()
		err := writer.WriteSection("Test Section", "test content")
		if err != nil {
			t.Fatalf("WriteSection failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["section"] != "Test Section" {
			t.Errorf("Expected section 'Test Section', got %v", data["section"])
		}
	})

	t.Run("WriteTable", func(t *testing.T) {
		buf.Reset()
		headers := []string{"Name", "Age"}
		rows := [][]string{{"John", "30"}}

		err := writer.WriteTable(headers, rows)
		if err != nil {
			t.Fatalf("WriteTable failed: %v", err)
		}

		var data []map[string]string
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if len(data) != 1 {
			t.Errorf("Expected 1 row, got %d", len(data))
		}
	})

	t.Run("WriteSQL", func(t *testing.T) {
		buf.Reset()
		statements := []string{"CREATE TABLE test;"}

		err := writer.WriteSQL(statements)
		if err != nil {
			t.Fatalf("WriteSQL failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		sqlStatements, ok := data["sql_statements"].([]interface{})
		if !ok || len(sqlStatements) != 1 {
			t.Errorf("Expected 1 SQL statement, got %v", data["sql_statements"])
		}
	})

	t.Run("WriteStatusMessage", func(t *testing.T) {
		buf.Reset()
		err := writer.WriteStatusMessage("INFO", "Test message")
		if err != nil {
			t.Fatalf("WriteStatusMessage failed: %v", err)
		}

		var data map[string]string
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["level"] != "INFO" {
			t.Errorf("Expected level 'INFO', got %v", data["level"])
		}
	})

	t.Run("SetFormat", func(t *testing.T) {
		writer.SetFormat(FormatCompact)
		if writer.GetFormat() != FormatCompact {
			t.Errorf("Expected format %s, got %s", FormatCompact, writer.GetFormat())
		}

		buf.Reset()
		err := writer.WriteStatusMessage("INFO", "Test message")
		if err != nil {
			t.Fatalf("WriteStatusMessage failed: %v", err)
		}

		result := buf.String()
		expected := "STATUS:INFO:Test message"
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
		}
	})
}

func TestFormatterEdgeCases(t *testing.T) {
	t.Run("JSONFormatterWithComplexData", func(t *testing.T) {
		formatter := NewJSONFormatter()

		complexData := map[string]interface{}{
			"nested": map[string]interface{}{
				"array":  []int{1, 2, 3},
				"string": "test",
				"bool":   true,
			},
		}

		result, err := formatter.FormatSection("Complex", complexData)
		if err != nil {
			t.Fatalf("FormatSection with complex data failed: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if data["section"] != "Complex" {
			t.Errorf("Expected section 'Complex', got %v", data["section"])
		}
	})

	t.Run("TableWithMismatchedRowLength", func(t *testing.T) {
		formatter := NewJSONFormatter()

		headers := []string{"Name", "Age", "City"}
		rows := [][]string{
			{"John", "30"},                // Missing city
			{"Jane", "25", "LA", "Extra"}, // Extra field
		}

		result, err := formatter.FormatTable(headers, rows)
		if err != nil {
			t.Fatalf("FormatTable with mismatched rows failed: %v", err)
		}

		var data []map[string]string
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// First row should have empty city
		if data[0]["City"] != "" {
			t.Errorf("Expected empty city for first row, got '%s'", data[0]["City"])
		}

		// Second row should ignore extra field
		if len(data[1]) != 3 {
			t.Errorf("Expected 3 fields in second row, got %d", len(data[1]))
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		formatter := NewCompactFormatter()

		// Empty table
		result, err := formatter.FormatTable([]string{}, [][]string{})
		if err != nil {
			t.Fatalf("FormatTable with empty data failed: %v", err)
		}

		if strings.TrimSpace(result) != "" {
			t.Errorf("Expected empty result for empty table, got '%s'", result)
		}

		// Empty SQL
		result, err = formatter.FormatSQL([]string{})
		if err != nil {
			t.Fatalf("FormatSQL with empty data failed: %v", err)
		}

		expected := "SQL:0:"
		if result != expected {
			t.Errorf("Expected '%s' for empty SQL, got '%s'", expected, result)
		}
	})
}
