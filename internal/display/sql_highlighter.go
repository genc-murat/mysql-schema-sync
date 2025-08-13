package display

import (
	"regexp"
	"strings"
)

// SQLHighlighter provides SQL syntax highlighting with color support
type SQLHighlighter struct {
	colorSystem ColorSystem
	theme       ColorTheme
	keywords    map[string]Color
	patterns    map[string]*regexp.Regexp
}

// NewSQLHighlighter creates a new SQL syntax highlighter
func NewSQLHighlighter(colorSystem ColorSystem, theme ColorTheme) *SQLHighlighter {
	highlighter := &SQLHighlighter{
		colorSystem: colorSystem,
		theme:       theme,
		keywords:    make(map[string]Color),
		patterns:    make(map[string]*regexp.Regexp),
	}

	highlighter.initializeKeywords()
	highlighter.initializePatterns()
	return highlighter
}

// initializeKeywords sets up SQL keyword color mappings
func (sh *SQLHighlighter) initializeKeywords() {
	// DDL Keywords (Data Definition Language) - Primary color
	ddlKeywords := []string{
		"CREATE", "DROP", "ALTER", "TRUNCATE", "RENAME",
		"TABLE", "DATABASE", "SCHEMA", "INDEX", "VIEW",
		"PROCEDURE", "FUNCTION", "TRIGGER", "CONSTRAINT",
	}
	for _, keyword := range ddlKeywords {
		sh.keywords[keyword] = sh.theme.Primary
	}

	// DML Keywords (Data Manipulation Language) - Success color
	dmlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "REPLACE",
		"FROM", "WHERE", "JOIN", "INNER", "LEFT", "RIGHT",
		"FULL", "OUTER", "ON", "USING", "GROUP", "ORDER",
		"HAVING", "LIMIT", "OFFSET", "UNION", "INTERSECT",
		"EXCEPT", "WITH", "AS", "DISTINCT", "ALL",
	}
	for _, keyword := range dmlKeywords {
		sh.keywords[keyword] = sh.theme.Success
	}

	// Data Types - Info color
	dataTypes := []string{
		"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT",
		"DECIMAL", "NUMERIC", "FLOAT", "DOUBLE", "REAL",
		"VARCHAR", "CHAR", "TEXT", "LONGTEXT", "MEDIUMTEXT",
		"TINYTEXT", "BLOB", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB",
		"DATE", "TIME", "DATETIME", "TIMESTAMP", "YEAR",
		"BOOLEAN", "BOOL", "JSON", "ENUM", "SET",
	}
	for _, dataType := range dataTypes {
		sh.keywords[dataType] = sh.theme.Info
	}

	// Operators and Functions - Highlight color
	operators := []string{
		"AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE",
		"IS", "NULL", "TRUE", "FALSE", "CASE", "WHEN", "THEN",
		"ELSE", "END", "IF", "IFNULL", "COALESCE", "CONCAT",
		"COUNT", "SUM", "AVG", "MIN", "MAX", "LENGTH", "SUBSTRING",
	}
	for _, operator := range operators {
		sh.keywords[operator] = sh.theme.Highlight
	}

	// Constraints and Modifiers - Warning color
	constraints := []string{
		"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE",
		"CHECK", "DEFAULT", "AUTO_INCREMENT", "NOT", "NULL",
		"UNSIGNED", "ZEROFILL", "BINARY", "COLLATE", "CHARACTER",
		"SET", "ENGINE", "CHARSET", "COMMENT",
	}
	for _, constraint := range constraints {
		sh.keywords[constraint] = sh.theme.Warning
	}
}

// initializePatterns sets up regex patterns for different SQL elements
func (sh *SQLHighlighter) initializePatterns() {
	sh.patterns = map[string]*regexp.Regexp{
		// String literals (single and double quotes)
		"string": regexp.MustCompile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"`),

		// Numeric literals
		"number": regexp.MustCompile(`\b\d+(\.\d+)?\b`),

		// Comments (-- and /* */)
		"comment_line":  regexp.MustCompile(`--.*$`),
		"comment_block": regexp.MustCompile(`/\*[\s\S]*?\*/`),

		// Identifiers with backticks
		"identifier": regexp.MustCompile("`[^`]+`"),

		// SQL operators
		"operator": regexp.MustCompile(`[=<>!]+|[+\-*/]`),

		// Parentheses and brackets
		"bracket": regexp.MustCompile(`[()[\]{}]`),

		// Semicolons and commas
		"delimiter": regexp.MustCompile(`[;,]`),
	}
}

// Highlight applies syntax highlighting to SQL text
func (sh *SQLHighlighter) Highlight(sql string) string {
	if !sh.colorSystem.IsColorSupported() {
		return sh.formatSQL(sql)
	}

	result := sql

	// First, protect strings and comments from keyword highlighting
	protectedRanges := sh.findProtectedRanges(result)

	// Apply keyword highlighting
	result = sh.highlightKeywords(result, protectedRanges)

	// Apply pattern-based highlighting
	result = sh.highlightPatterns(result, protectedRanges)

	// Format the SQL with proper indentation
	return sh.formatSQL(result)
}

// HighlightStatement highlights a single SQL statement with proper formatting
func (sh *SQLHighlighter) HighlightStatement(statement string, index int) string {
	// Add statement number comment
	header := ""
	if index >= 0 {
		header = sh.colorSystem.Sprintf(sh.theme.Muted, "-- Statement %d", index+1) + "\n"
	}

	highlighted := sh.Highlight(statement)
	return header + highlighted
}

// findProtectedRanges finds ranges that should not be processed for keywords (strings, comments)
func (sh *SQLHighlighter) findProtectedRanges(sql string) [][]int {
	var ranges [][]int

	// Find string literals
	for _, match := range sh.patterns["string"].FindAllStringIndex(sql, -1) {
		ranges = append(ranges, match)
	}

	// Find comments
	for _, match := range sh.patterns["comment_line"].FindAllStringIndex(sql, -1) {
		ranges = append(ranges, match)
	}
	for _, match := range sh.patterns["comment_block"].FindAllStringIndex(sql, -1) {
		ranges = append(ranges, match)
	}

	return ranges
}

// isInProtectedRange checks if a position is within a protected range
func (sh *SQLHighlighter) isInProtectedRange(pos int, ranges [][]int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// highlightKeywords applies color highlighting to SQL keywords
func (sh *SQLHighlighter) highlightKeywords(sql string, protectedRanges [][]int) string {
	result := sql

	// Sort keywords by length (longest first) to avoid partial matches
	var sortedKeywords []string
	for keyword := range sh.keywords {
		sortedKeywords = append(sortedKeywords, keyword)
	}

	// Simple sort by length (descending)
	for i := 0; i < len(sortedKeywords); i++ {
		for j := i + 1; j < len(sortedKeywords); j++ {
			if len(sortedKeywords[i]) < len(sortedKeywords[j]) {
				sortedKeywords[i], sortedKeywords[j] = sortedKeywords[j], sortedKeywords[i]
			}
		}
	}

	for _, keyword := range sortedKeywords {
		color := sh.keywords[keyword]

		// Create word boundary regex for the keyword
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)

		// Find all matches
		matches := pattern.FindAllStringIndex(result, -1)

		// Apply highlighting from right to left to preserve indices
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]

			// Skip if in protected range
			if sh.isInProtectedRange(match[0], protectedRanges) {
				continue
			}

			originalText := result[match[0]:match[1]]
			highlightedText := sh.colorSystem.Colorize(strings.ToUpper(originalText), color)

			result = result[:match[0]] + highlightedText + result[match[1]:]
		}
	}

	return result
}

// highlightPatterns applies highlighting to various SQL patterns
func (sh *SQLHighlighter) highlightPatterns(sql string, protectedRanges [][]int) string {
	result := sql

	// Highlight string literals
	result = sh.highlightPattern(result, "string", sh.theme.Success, protectedRanges, false)

	// Highlight numeric literals
	result = sh.highlightPattern(result, "number", sh.theme.Info, protectedRanges, true)

	// Highlight comments
	result = sh.highlightPattern(result, "comment_line", sh.theme.Muted, protectedRanges, false)
	result = sh.highlightPattern(result, "comment_block", sh.theme.Muted, protectedRanges, false)

	// Highlight identifiers with backticks
	result = sh.highlightPattern(result, "identifier", sh.theme.Highlight, protectedRanges, true)

	// Highlight operators
	result = sh.highlightPattern(result, "operator", sh.theme.Warning, protectedRanges, true)

	// Highlight brackets and delimiters
	result = sh.highlightPattern(result, "bracket", sh.theme.Primary, protectedRanges, true)
	result = sh.highlightPattern(result, "delimiter", sh.theme.Primary, protectedRanges, true)

	return result
}

// highlightPattern applies highlighting to a specific pattern
func (sh *SQLHighlighter) highlightPattern(sql string, patternName string, color Color, protectedRanges [][]int, checkProtected bool) string {
	pattern, exists := sh.patterns[patternName]
	if !exists {
		return sql
	}

	result := sql
	matches := pattern.FindAllStringIndex(result, -1)

	// Apply highlighting from right to left to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]

		// Skip if in protected range and we should check
		if checkProtected && sh.isInProtectedRange(match[0], protectedRanges) {
			continue
		}

		originalText := result[match[0]:match[1]]
		highlightedText := sh.colorSystem.Colorize(originalText, color)

		result = result[:match[0]] + highlightedText + result[match[1]:]
	}

	return result
}

// formatSQL applies proper indentation and formatting to SQL
func (sh *SQLHighlighter) formatSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	var formatted []string
	indentLevel := 0
	indentSize := 2

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}

		// Decrease indent for closing keywords
		if sh.isClosingKeyword(trimmed) {
			indentLevel--
			if indentLevel < 0 {
				indentLevel = 0
			}
		}

		// Apply indentation
		indent := strings.Repeat(" ", indentLevel*indentSize)
		formatted = append(formatted, indent+trimmed)

		// Increase indent for opening keywords
		if sh.isOpeningKeyword(trimmed) {
			indentLevel++
		}
	}

	return strings.Join(formatted, "\n")
}

// isOpeningKeyword checks if a line contains keywords that should increase indentation
func (sh *SQLHighlighter) isOpeningKeyword(line string) bool {
	upperLine := strings.ToUpper(line)
	openingKeywords := []string{
		"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "HAVING",
		"JOIN", "LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "OUTER JOIN",
		"UNION", "CASE", "BEGIN", "IF", "WHILE", "FOR",
	}

	for _, keyword := range openingKeywords {
		if strings.Contains(upperLine, keyword) {
			return true
		}
	}
	return false
}

// isClosingKeyword checks if a line contains keywords that should decrease indentation
func (sh *SQLHighlighter) isClosingKeyword(line string) bool {
	upperLine := strings.ToUpper(line)
	closingKeywords := []string{
		"END", "ELSE", "ELSIF", "WHEN",
	}

	for _, keyword := range closingKeywords {
		if strings.HasPrefix(upperLine, keyword) {
			return true
		}
	}
	return false
}

// SetTheme updates the color theme for the highlighter
func (sh *SQLHighlighter) SetTheme(theme ColorTheme) {
	sh.theme = theme
	sh.initializeKeywords() // Reinitialize with new theme colors
}
