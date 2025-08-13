package display

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestSQLHighlighter_Highlight(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	tests := []struct {
		name     string
		sql      string
		expected []string // Keywords/patterns that should be highlighted
	}{
		{
			name:     "basic SELECT statement",
			sql:      "SELECT id, name FROM users WHERE active = 1",
			expected: []string{"SELECT", "FROM", "WHERE"},
		},
		{
			name:     "CREATE TABLE statement",
			sql:      "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255) NOT NULL)",
			expected: []string{"CREATE", "TABLE", "INT", "PRIMARY", "KEY", "VARCHAR", "NOT", "NULL"},
		},
		{
			name:     "complex query with joins",
			sql:      "SELECT u.name, p.title FROM users u LEFT JOIN posts p ON u.id = p.user_id",
			expected: []string{"SELECT", "FROM", "LEFT", "JOIN", "ON"},
		},
		{
			name:     "statement with string literals",
			sql:      "INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com')",
			expected: []string{"INSERT", "INTO", "VALUES"},
		},
		{
			name:     "statement with comments",
			sql:      "-- This is a comment\nSELECT * FROM users /* inline comment */",
			expected: []string{"SELECT", "FROM"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlighter.Highlight(tt.sql)

			// Check that expected keywords are present (case-insensitive)
			for _, expected := range tt.expected {
				upperResult := strings.ToUpper(result)
				if !strings.Contains(upperResult, strings.ToUpper(expected)) {
					t.Errorf("Expected highlighted SQL to contain %q, but it didn't. Result:\n%s", expected, result)
				}
			}

			// Verify the result is not empty
			if strings.TrimSpace(result) == "" {
				t.Error("Highlighted SQL should not be empty")
			}
		})
	}
}

func TestSQLHighlighter_HighlightStatement(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	sql := "SELECT * FROM users"
	result := highlighter.HighlightStatement(sql, 0)

	// Should contain statement number comment
	if !strings.Contains(result, "Statement 1") {
		t.Errorf("Expected result to contain statement number, got: %s", result)
	}

	// Should contain the SQL
	if !strings.Contains(strings.ToUpper(result), "SELECT") {
		t.Errorf("Expected result to contain SQL keywords, got: %s", result)
	}
}

func TestSQLHighlighter_NoColorMode(t *testing.T) {
	// Create a color system with colors disabled
	theme := DefaultColorTheme()
	colorSystem := &colorSystem{
		theme:          theme,
		colorSupported: false,
		colorMap:       make(map[Color]*color.Color),
	}

	highlighter := NewSQLHighlighter(colorSystem, theme)
	sql := "SELECT * FROM users"
	result := highlighter.Highlight(sql)

	// In no-color mode, should still format but not add color codes
	// The result should be formatted but without ANSI color codes
	if strings.Contains(result, "\033[") {
		t.Error("Expected no color codes in no-color mode")
	}

	// Should still contain the SQL keywords
	if !strings.Contains(strings.ToUpper(result), "SELECT") {
		t.Errorf("Expected result to contain SQL keywords, got: %s", result)
	}
}

func TestSQLHighlighter_KeywordCategories(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	tests := []struct {
		name     string
		sql      string
		keywords []string
	}{
		{
			name:     "DDL keywords",
			sql:      "CREATE TABLE test (id INT)",
			keywords: []string{"CREATE", "TABLE"},
		},
		{
			name:     "DML keywords",
			sql:      "SELECT * FROM test WHERE id = 1",
			keywords: []string{"SELECT", "FROM", "WHERE"},
		},
		{
			name:     "data types",
			sql:      "CREATE TABLE test (id INT, name VARCHAR(255), created DATETIME)",
			keywords: []string{"INT", "VARCHAR", "DATETIME"},
		},
		{
			name:     "constraints",
			sql:      "CREATE TABLE test (id INT PRIMARY KEY AUTO_INCREMENT)",
			keywords: []string{"PRIMARY", "KEY", "AUTO_INCREMENT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlighter.Highlight(tt.sql)

			for _, keyword := range tt.keywords {
				upperResult := strings.ToUpper(result)
				if !strings.Contains(upperResult, strings.ToUpper(keyword)) {
					t.Errorf("Expected highlighted SQL to contain %q keyword, but it didn't. Result:\n%s", keyword, result)
				}
			}
		})
	}
}

func TestSQLHighlighter_StringLiterals(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	sql := `INSERT INTO users (name, email) VALUES ('John O''Brien', "jane@example.com")`
	result := highlighter.Highlight(sql)

	// Should contain the original string literals
	if !strings.Contains(result, "'John O''Brien'") {
		t.Errorf("Expected result to preserve string literals, got: %s", result)
	}

	if !strings.Contains(result, `"jane@example.com"`) {
		t.Errorf("Expected result to preserve double-quoted strings, got: %s", result)
	}
}

func TestSQLHighlighter_Comments(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "line comment",
			sql:  "SELECT * FROM users -- This is a comment",
		},
		{
			name: "block comment",
			sql:  "SELECT * FROM users /* This is a block comment */",
		},
		{
			name: "multiline block comment",
			sql:  "SELECT * FROM users /* This is a\nmultiline comment */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlighter.Highlight(tt.sql)

			// Should contain SELECT keyword
			if !strings.Contains(strings.ToUpper(result), "SELECT") {
				t.Errorf("Expected result to contain SELECT keyword, got: %s", result)
			}

			// Should preserve comments
			if strings.Contains(tt.sql, "This is a comment") && !strings.Contains(result, "This is a comment") {
				t.Errorf("Expected result to preserve comments, got: %s", result)
			}
		})
	}
}

func TestSQLHighlighter_NumericLiterals(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	sql := "SELECT * FROM products WHERE price > 19.99 AND quantity <= 100"
	result := highlighter.Highlight(sql)

	// Should preserve numeric values
	if !strings.Contains(result, "19.99") {
		t.Errorf("Expected result to preserve decimal numbers, got: %s", result)
	}

	if !strings.Contains(result, "100") {
		t.Errorf("Expected result to preserve integers, got: %s", result)
	}
}

func TestSQLHighlighter_SetTheme(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	// Change theme
	newTheme := ColorTheme{
		Primary:   ColorRed,
		Success:   ColorGreen,
		Warning:   ColorYellow,
		Error:     ColorBrightRed,
		Info:      ColorBlue,
		Muted:     ColorWhite,
		Highlight: ColorMagenta,
	}

	highlighter.SetTheme(newTheme)

	// Verify theme was updated
	if highlighter.theme.Primary != ColorRed {
		t.Error("Expected theme to be updated")
	}

	// Test that highlighting still works with new theme
	sql := "SELECT * FROM users"
	result := highlighter.Highlight(sql)

	if !strings.Contains(strings.ToUpper(result), "SELECT") {
		t.Errorf("Expected highlighting to work with new theme, got: %s", result)
	}
}

func TestSQLHighlighter_FormatSQL(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	// Test basic formatting
	sql := "SELECT id,name FROM users WHERE active=1"
	result := highlighter.formatSQL(sql)

	// Should preserve the content
	upperResult := strings.ToUpper(result)
	expectedKeywords := []string{"SELECT", "FROM", "WHERE"}
	for _, keyword := range expectedKeywords {
		if !strings.Contains(upperResult, keyword) {
			t.Errorf("Expected formatted SQL to contain %q, got: %s", keyword, result)
		}
	}
}

func TestSQLHighlighter_ProtectedRanges(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()
	highlighter := NewSQLHighlighter(colorSystem, theme)

	// SQL with keywords inside strings that shouldn't be highlighted
	sql := `SELECT 'This contains SELECT keyword' FROM users`
	result := highlighter.Highlight(sql)

	// The SELECT inside the string should not be highlighted separately
	// This is a bit tricky to test directly, but we can verify the string is preserved
	if !strings.Contains(result, "'This contains SELECT keyword'") {
		t.Errorf("Expected string literal to be preserved, got: %s", result)
	}
}
