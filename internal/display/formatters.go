package display

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// OutputFormatter defines the interface for different output format renderers
type OutputFormatter interface {
	FormatSection(title string, content interface{}) (string, error)
	FormatTable(headers []string, rows [][]string) (string, error)
	FormatSQL(statements []string) (string, error)
	FormatStatusMessage(level, message string) (string, error)
	FormatSchemaDiff(diff interface{}) (string, error)
}

// JSONFormatter implements OutputFormatter for JSON output
type JSONFormatter struct {
	indent string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		indent: "  ",
	}
}

// FormatSection formats a section as JSON
func (f *JSONFormatter) FormatSection(title string, content interface{}) (string, error) {
	data := map[string]interface{}{
		"section": title,
		"content": content,
	}

	jsonData, err := json.MarshalIndent(data, "", f.indent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal section to JSON: %w", err)
	}

	return string(jsonData), nil
}

// FormatTable formats a table as JSON
func (f *JSONFormatter) FormatTable(headers []string, rows [][]string) (string, error) {
	var data []map[string]string

	for _, row := range rows {
		rowMap := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				rowMap[header] = row[i]
			} else {
				rowMap[header] = ""
			}
		}
		data = append(data, rowMap)
	}

	jsonData, err := json.MarshalIndent(data, "", f.indent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal table to JSON: %w", err)
	}

	return string(jsonData), nil
}

// FormatSQL formats SQL statements as JSON
func (f *JSONFormatter) FormatSQL(statements []string) (string, error) {
	data := map[string]interface{}{
		"sql_statements": statements,
		"count":          len(statements),
	}

	jsonData, err := json.MarshalIndent(data, "", f.indent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SQL to JSON: %w", err)
	}

	return string(jsonData), nil
}

// FormatStatusMessage formats a status message as JSON
func (f *JSONFormatter) FormatStatusMessage(level, message string) (string, error) {
	data := map[string]string{
		"level":   level,
		"message": message,
	}

	jsonData, err := json.MarshalIndent(data, "", f.indent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal status message to JSON: %w", err)
	}

	return string(jsonData), nil
}

// FormatSchemaDiff formats schema differences as JSON
func (f *JSONFormatter) FormatSchemaDiff(diff interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(diff, "", f.indent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema diff to JSON: %w", err)
	}

	return string(jsonData), nil
}

// YAMLFormatter implements OutputFormatter for YAML output
type YAMLFormatter struct{}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

// FormatSection formats a section as YAML
func (f *YAMLFormatter) FormatSection(title string, content interface{}) (string, error) {
	data := map[string]interface{}{
		"section": title,
		"content": content,
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal section to YAML: %w", err)
	}

	return string(yamlData), nil
}

// FormatTable formats a table as YAML
func (f *YAMLFormatter) FormatTable(headers []string, rows [][]string) (string, error) {
	var data []map[string]string

	for _, row := range rows {
		rowMap := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				rowMap[header] = row[i]
			} else {
				rowMap[header] = ""
			}
		}
		data = append(data, rowMap)
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal table to YAML: %w", err)
	}

	return string(yamlData), nil
}

// FormatSQL formats SQL statements as YAML
func (f *YAMLFormatter) FormatSQL(statements []string) (string, error) {
	data := map[string]interface{}{
		"sql_statements": statements,
		"count":          len(statements),
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SQL to YAML: %w", err)
	}

	return string(yamlData), nil
}

// FormatStatusMessage formats a status message as YAML
func (f *YAMLFormatter) FormatStatusMessage(level, message string) (string, error) {
	data := map[string]string{
		"level":   level,
		"message": message,
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal status message to YAML: %w", err)
	}

	return string(yamlData), nil
}

// FormatSchemaDiff formats schema differences as YAML
func (f *YAMLFormatter) FormatSchemaDiff(diff interface{}) (string, error) {
	yamlData, err := yaml.Marshal(diff)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema diff to YAML: %w", err)
	}

	return string(yamlData), nil
}

// CompactFormatter implements OutputFormatter for compact/scripting output
// It provides minimal, machine-readable output suitable for automation and parsing
type CompactFormatter struct {
	separator      string
	includeHeaders bool
}

// NewCompactFormatter creates a new compact formatter with default settings
func NewCompactFormatter() *CompactFormatter {
	return &CompactFormatter{
		separator:      "\t",
		includeHeaders: true,
	}
}

// NewCompactFormatterWithOptions creates a compact formatter with custom options
func NewCompactFormatterWithOptions(separator string, includeHeaders bool) *CompactFormatter {
	return &CompactFormatter{
		separator:      separator,
		includeHeaders: includeHeaders,
	}
}

// FormatSection formats a section in compact format
// Output format: SECTION:title:key=value,key=value
func (f *CompactFormatter) FormatSection(title string, content interface{}) (string, error) {
	var result strings.Builder
	result.WriteString("SECTION:")
	result.WriteString(title)
	result.WriteString(":")

	switch v := content.(type) {
	case map[string]interface{}:
		// Sort keys for consistent output
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		// Sort keys alphabetically
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}

		var pairs []string
		for _, key := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%v", key, v[key]))
		}
		result.WriteString(strings.Join(pairs, ","))
	case string:
		result.WriteString(v)
	default:
		result.WriteString(fmt.Sprintf("%v", content))
	}

	return result.String(), nil
}

// FormatTable formats a table in compact format (separator-separated values)
// Output format: TSV (Tab-Separated Values) by default, customizable separator
func (f *CompactFormatter) FormatTable(headers []string, rows [][]string) (string, error) {
	var result strings.Builder

	// Write headers if enabled
	if f.includeHeaders && len(headers) > 0 {
		result.WriteString(strings.Join(headers, f.separator))
		result.WriteString("\n")
	}

	// Write rows
	for _, row := range rows {
		// Ensure row has same length as headers, pad with empty strings if needed
		paddedRow := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				paddedRow[i] = row[i]
			} else {
				paddedRow[i] = ""
			}
		}
		result.WriteString(strings.Join(paddedRow, f.separator))
		result.WriteString("\n")
	}

	return result.String(), nil
}

// FormatSQL formats SQL statements in compact format
// Output format: SQL:count:statement1|statement2|...
func (f *CompactFormatter) FormatSQL(statements []string) (string, error) {
	if len(statements) == 0 {
		return "SQL:0:", nil
	}

	// Use pipe separator to avoid conflicts with semicolons in SQL
	// Escape pipe characters in statements to avoid conflicts
	escapedStatements := make([]string, len(statements))
	for i, stmt := range statements {
		escapedStatements[i] = strings.ReplaceAll(stmt, "|", "\\|")
	}

	return fmt.Sprintf("SQL:%d:%s", len(statements), strings.Join(escapedStatements, "|")), nil
}

// FormatStatusMessage formats a status message in compact format
// Output format: STATUS:level:message
func (f *CompactFormatter) FormatStatusMessage(level, message string) (string, error) {
	return fmt.Sprintf("STATUS:%s:%s", level, message), nil
}

// FormatSchemaDiff formats schema differences in compact format
// Output format: DIFF:type:count:details
func (f *CompactFormatter) FormatSchemaDiff(diff interface{}) (string, error) {
	switch v := diff.(type) {
	case map[string]interface{}:
		// Sort keys for consistent output
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		// Sort keys alphabetically
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}

		var parts []string
		for _, key := range keys {
			value := v[key]
			switch val := value.(type) {
			case []string:
				if len(val) > 0 {
					parts = append(parts, fmt.Sprintf("%s=%d", key, len(val)))
				}
			case []interface{}:
				if len(val) > 0 {
					parts = append(parts, fmt.Sprintf("%s=%d", key, len(val)))
				}
			default:
				parts = append(parts, fmt.Sprintf("%s=%v", key, value))
			}
		}
		return fmt.Sprintf("DIFF:schema:%s", strings.Join(parts, ",")), nil
	default:
		return fmt.Sprintf("DIFF:unknown:%v", diff), nil
	}
}

// SetSeparator changes the field separator for table output
func (f *CompactFormatter) SetSeparator(separator string) {
	f.separator = separator
}

// SetIncludeHeaders controls whether table headers are included
func (f *CompactFormatter) SetIncludeHeaders(include bool) {
	f.includeHeaders = include
}

// GetSeparator returns the current field separator
func (f *CompactFormatter) GetSeparator() string {
	return f.separator
}

// GetIncludeHeaders returns whether headers are included
func (f *CompactFormatter) GetIncludeHeaders() bool {
	return f.includeHeaders
}

// FormatterRegistry manages different output formatters
type FormatterRegistry struct {
	formatters map[OutputFormat]OutputFormatter
}

// NewFormatterRegistry creates a new formatter registry with default formatters
func NewFormatterRegistry() *FormatterRegistry {
	registry := &FormatterRegistry{
		formatters: make(map[OutputFormat]OutputFormatter),
	}

	// Register default formatters
	registry.Register(FormatJSON, NewJSONFormatter())
	registry.Register(FormatYAML, NewYAMLFormatter())
	registry.Register(FormatCompact, NewCompactFormatter())

	return registry
}

// Register registers a formatter for a specific output format
func (r *FormatterRegistry) Register(format OutputFormat, formatter OutputFormatter) {
	r.formatters[format] = formatter
}

// GetFormatter returns the formatter for the specified format
func (r *FormatterRegistry) GetFormatter(format OutputFormat) (OutputFormatter, bool) {
	formatter, exists := r.formatters[format]
	return formatter, exists
}

// FormatOutput formats content using the specified format
func (r *FormatterRegistry) FormatOutput(format OutputFormat, outputType string, data interface{}) (string, error) {
	formatter, exists := r.GetFormatter(format)
	if !exists {
		return "", fmt.Errorf("unsupported output format: %s", format)
	}

	switch outputType {
	case "section":
		if sectionData, ok := data.(map[string]interface{}); ok {
			if title, titleOk := sectionData["title"].(string); titleOk {
				if content, contentOk := sectionData["content"]; contentOk {
					return formatter.FormatSection(title, content)
				}
			}
		}
		return "", fmt.Errorf("invalid section data format")

	case "table":
		if tableData, ok := data.(map[string]interface{}); ok {
			if headers, headersOk := tableData["headers"].([]string); headersOk {
				if rows, rowsOk := tableData["rows"].([][]string); rowsOk {
					return formatter.FormatTable(headers, rows)
				}
			}
		}
		return "", fmt.Errorf("invalid table data format")

	case "sql":
		if statements, ok := data.([]string); ok {
			return formatter.FormatSQL(statements)
		}
		return "", fmt.Errorf("invalid SQL data format")

	case "status":
		if statusData, ok := data.(map[string]string); ok {
			if level, levelOk := statusData["level"]; levelOk {
				if message, messageOk := statusData["message"]; messageOk {
					return formatter.FormatStatusMessage(level, message)
				}
			}
		}
		return "", fmt.Errorf("invalid status message data format")

	case "schema_diff":
		return formatter.FormatSchemaDiff(data)

	default:
		return "", fmt.Errorf("unsupported output type: %s", outputType)
	}
}

// OutputWriter provides a unified interface for writing formatted output
type OutputWriter struct {
	registry *FormatterRegistry
	format   OutputFormat
	writer   io.Writer
}

// NewOutputWriter creates a new output writer with the specified format and writer
func NewOutputWriter(format OutputFormat, writer io.Writer) *OutputWriter {
	return &OutputWriter{
		registry: NewFormatterRegistry(),
		format:   format,
		writer:   writer,
	}
}

// WriteSection writes a formatted section
func (w *OutputWriter) WriteSection(title string, content interface{}) error {
	data := map[string]interface{}{
		"title":   title,
		"content": content,
	}

	output, err := w.registry.FormatOutput(w.format, "section", data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w.writer, output)
	return err
}

// WriteTable writes a formatted table
func (w *OutputWriter) WriteTable(headers []string, rows [][]string) error {
	data := map[string]interface{}{
		"headers": headers,
		"rows":    rows,
	}

	output, err := w.registry.FormatOutput(w.format, "table", data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w.writer, output)
	return err
}

// WriteSQL writes formatted SQL statements
func (w *OutputWriter) WriteSQL(statements []string) error {
	output, err := w.registry.FormatOutput(w.format, "sql", statements)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w.writer, output)
	return err
}

// WriteStatusMessage writes a formatted status message
func (w *OutputWriter) WriteStatusMessage(level, message string) error {
	data := map[string]string{
		"level":   level,
		"message": message,
	}

	output, err := w.registry.FormatOutput(w.format, "status", data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w.writer, output)
	return err
}

// WriteSchemaDiff writes a formatted schema diff
func (w *OutputWriter) WriteSchemaDiff(diff interface{}) error {
	output, err := w.registry.FormatOutput(w.format, "schema_diff", diff)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w.writer, output)
	return err
}

// SetFormat changes the output format
func (w *OutputWriter) SetFormat(format OutputFormat) {
	w.format = format
}

// GetFormat returns the current output format
func (w *OutputWriter) GetFormat() OutputFormat {
	return w.format
}
