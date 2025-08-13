package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// displayService implements the DisplayService interface
type displayService struct {
	config            *DisplayConfig
	colorSystem       ColorSystem
	iconSystem        IconSystem
	writer            io.Writer
	spinnerManager    *spinnerManager
	formatterRegistry *FormatterRegistry
}

// NewDisplayService creates a new display service with the given configuration
func NewDisplayService(config *DisplayConfig) DisplayService {
	if config == nil {
		config = DefaultDisplayConfig()
	}

	if config.Writer == nil {
		config.Writer = os.Stdout
	}

	colorSystem := NewColorSystem(config.GetColorTheme())
	iconSystem := NewIconSystem()

	return &displayService{
		config:            config,
		colorSystem:       colorSystem,
		iconSystem:        iconSystem,
		writer:            config.Writer,
		spinnerManager:    newSpinnerManager(),
		formatterRegistry: NewFormatterRegistry(),
	}
}

// PrintHeader prints a formatted header
func (ds *displayService) PrintHeader(title string) {
	if ds.config.QuietMode {
		return
	}

	separator := strings.Repeat("=", len(title)+4)
	headerText := fmt.Sprintf("\n%s\n  %s  \n%s\n", separator, title, separator)

	if ds.config.IsColorEnabled() && ds.colorSystem.IsColorSupported() {
		theme := ds.config.GetColorTheme()
		headerText = ds.colorSystem.Colorize(headerText, theme.Primary)
	}

	fmt.Fprint(ds.writer, headerText)
}

// PrintSection prints a formatted section with title and content
func (ds *displayService) PrintSection(title string, content interface{}) {
	if ds.config.QuietMode {
		return
	}

	// Use formatter for structured formats
	if ds.config.OutputFormat == string(FormatJSON) || ds.config.OutputFormat == string(FormatYAML) || ds.config.OutputFormat == string(FormatCompact) {
		data := map[string]interface{}{
			"title":   title,
			"content": content,
		}

		output, err := ds.formatterRegistry.FormatOutput(OutputFormat(ds.config.OutputFormat), "section", data)
		if err != nil {
			fmt.Fprintf(ds.writer, "Error formatting section: %v\n", err)
			return
		}

		fmt.Fprint(ds.writer, output)
		fmt.Fprintln(ds.writer) // Add newline for readability
		return
	}

	// Default table format with visual enhancements
	sectionTitle := fmt.Sprintf("\n--- %s ---\n", title)

	if ds.config.IsColorEnabled() && ds.colorSystem.IsColorSupported() {
		theme := ds.config.GetColorTheme()
		sectionTitle = ds.colorSystem.Colorize(sectionTitle, theme.Highlight)
	}

	fmt.Fprint(ds.writer, sectionTitle)
	ds.printDefault(content)
}

// PrintTable prints a formatted table
func (ds *displayService) PrintTable(headers []string, rows [][]string) {
	if ds.config.QuietMode && ds.config.OutputFormat != string(FormatJSON) && ds.config.OutputFormat != string(FormatYAML) {
		return
	}

	// Use formatter for structured formats
	if ds.config.OutputFormat == string(FormatJSON) || ds.config.OutputFormat == string(FormatYAML) || ds.config.OutputFormat == string(FormatCompact) {
		data := map[string]interface{}{
			"headers": headers,
			"rows":    rows,
		}

		output, err := ds.formatterRegistry.FormatOutput(OutputFormat(ds.config.OutputFormat), "table", data)
		if err != nil {
			fmt.Fprintf(ds.writer, "Error formatting table: %v\n", err)
			return
		}

		fmt.Fprint(ds.writer, output)
		fmt.Fprintln(ds.writer) // Add newline for readability
		return
	}

	// Default table format with visual enhancements
	ds.printTableFormatted(headers, rows)
}

// PrintSQL prints formatted SQL statements
func (ds *displayService) PrintSQL(statements []string) {
	if ds.config.QuietMode && ds.config.OutputFormat != string(FormatJSON) && ds.config.OutputFormat != string(FormatYAML) {
		return
	}

	// Use formatter for structured formats
	if ds.config.OutputFormat == string(FormatJSON) || ds.config.OutputFormat == string(FormatYAML) || ds.config.OutputFormat == string(FormatCompact) {
		output, err := ds.formatterRegistry.FormatOutput(OutputFormat(ds.config.OutputFormat), "sql", statements)
		if err != nil {
			fmt.Fprintf(ds.writer, "Error formatting SQL: %v\n", err)
			return
		}

		fmt.Fprint(ds.writer, output)
		fmt.Fprintln(ds.writer) // Add newline for readability
		return
	}

	// Default format with syntax highlighting
	ds.printSQLFormatted(statements)
}

// Success prints a success message
func (ds *displayService) Success(message string) {
	theme := ds.config.GetColorTheme()
	ds.printStatusMessage("SUCCESS", message, theme.Success)
}

// Warning prints a warning message
func (ds *displayService) Warning(message string) {
	theme := ds.config.GetColorTheme()
	ds.printStatusMessage("WARNING", message, theme.Warning)
}

// Error prints an error message
func (ds *displayService) Error(message string) {
	theme := ds.config.GetColorTheme()
	ds.printStatusMessage("ERROR", message, theme.Error)
}

// Info prints an info message
func (ds *displayService) Info(message string) {
	if ds.config.QuietMode {
		return
	}
	theme := ds.config.GetColorTheme()
	ds.printStatusMessage("INFO", message, theme.Info)
}

// SetOutput sets the output writer
func (ds *displayService) SetOutput(writer io.Writer) {
	ds.writer = writer
	ds.config.Writer = writer
}

// GetConfig returns the current configuration
func (ds *displayService) GetConfig() *DisplayConfig {
	return ds.config
}

// SetConfig updates the configuration
func (ds *displayService) SetConfig(config *DisplayConfig) {
	ds.config = config
	if config.Writer != nil {
		ds.writer = config.Writer
	}
	ds.colorSystem.SetTheme(config.GetColorTheme())
}

// Helper methods

func (ds *displayService) printStatusMessage(level, message string, color Color) {
	// Use formatter for structured formats
	if ds.config.OutputFormat == string(FormatJSON) || ds.config.OutputFormat == string(FormatYAML) || ds.config.OutputFormat == string(FormatCompact) {
		data := map[string]string{
			"level":   level,
			"message": message,
		}

		output, err := ds.formatterRegistry.FormatOutput(OutputFormat(ds.config.OutputFormat), "status", data)
		if err != nil {
			fmt.Fprintf(ds.writer, "Error formatting status message: %v\n", err)
			return
		}

		fmt.Fprint(ds.writer, output)
		fmt.Fprintln(ds.writer) // Add newline for readability
		return
	}

	// Default format with colors
	prefix := fmt.Sprintf("[%s]", level)

	if ds.config.IsColorEnabled() && ds.colorSystem.IsColorSupported() {
		prefix = ds.colorSystem.Colorize(prefix, color)
	}

	fmt.Fprintf(ds.writer, "%s %s\n", prefix, message)
}

func (ds *displayService) printJSON(content interface{}) {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Fprintf(ds.writer, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Fprint(ds.writer, string(data)+"\n")
}

func (ds *displayService) printYAML(content interface{}) {
	data, err := yaml.Marshal(content)
	if err != nil {
		fmt.Fprintf(ds.writer, "Error formatting YAML: %v\n", err)
		return
	}
	fmt.Fprint(ds.writer, string(data))
}

func (ds *displayService) printCompact(content interface{}) {
	fmt.Fprintf(ds.writer, "%v\n", content)
}

func (ds *displayService) printDefault(content interface{}) {
	fmt.Fprintf(ds.writer, "%v\n", content)
}

func (ds *displayService) printTableFormatted(headers []string, rows [][]string) {
	theme := ds.config.GetColorTheme()
	formatter := NewTableFormatter(ds.colorSystem, theme)

	// Set headers
	if len(headers) > 0 {
		formatter.SetHeaders(headers)
	}

	// Add rows
	for _, row := range rows {
		formatter.AddRow(row)
	}

	// Render the table
	formatter.RenderTo(ds.writer)
}

func (ds *displayService) printTableCompact(headers []string, rows [][]string) {
	// Print headers
	fmt.Fprintln(ds.writer, strings.Join(headers, "\t"))

	// Print rows
	for _, row := range rows {
		fmt.Fprintln(ds.writer, strings.Join(row, "\t"))
	}
}

func (ds *displayService) printTableAsJSON(headers []string, rows [][]string) {
	var data []map[string]string

	for _, row := range rows {
		rowMap := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				rowMap[header] = row[i]
			}
		}
		data = append(data, rowMap)
	}

	ds.printJSON(data)
}

func (ds *displayService) printTableAsYAML(headers []string, rows [][]string) {
	var data []map[string]string

	for _, row := range rows {
		rowMap := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				rowMap[header] = row[i]
			}
		}
		data = append(data, rowMap)
	}

	ds.printYAML(data)
}

func (ds *displayService) printSQLFormatted(statements []string) {
	theme := ds.config.GetColorTheme()
	highlighter := NewSQLHighlighter(ds.colorSystem, theme)

	for i, stmt := range statements {
		if ds.config.IsColorEnabled() && ds.colorSystem.IsColorSupported() {
			highlighted := highlighter.HighlightStatement(stmt, i)
			fmt.Fprintln(ds.writer, highlighted)
		} else {
			fmt.Fprintf(ds.writer, "-- Statement %d\n", i+1)
			fmt.Fprintln(ds.writer, stmt)
		}

		fmt.Fprintln(ds.writer, "")
	}
}

func (ds *displayService) printSQLAsJSON(statements []string) {
	data := map[string][]string{
		"sql_statements": statements,
	}
	ds.printJSON(data)
}

func (ds *displayService) printSQLAsYAML(statements []string) {
	data := map[string][]string{
		"sql_statements": statements,
	}
	ds.printYAML(data)
}

// RenderIcon returns the appropriate icon representation (Unicode or ASCII)
func (ds *displayService) RenderIcon(name string) string {
	return ds.iconSystem.RenderIcon(name)
}

// RenderIconWithColor returns the icon with color applied
func (ds *displayService) RenderIconWithColor(name string) string {
	return ds.iconSystem.RenderIconWithColor(name, ds.colorSystem)
}

// GetIconSystem returns the icon system
func (ds *displayService) GetIconSystem() IconSystem {
	return ds.iconSystem
}

// StartSpinner starts a new spinner with the given message
func (ds *displayService) StartSpinner(message string) SpinnerHandle {
	if ds.config.QuietMode {
		return &noOpSpinner{}
	}

	// Choose spinner style based on terminal capabilities
	var style SpinnerStyle
	if ds.iconSystem.IsUnicodeSupported() {
		style = DefaultSpinnerStyles["dots"]
	} else {
		style = DefaultSpinnerStyles["line"]
	}

	theme := ds.config.GetColorTheme()
	spinner := ds.spinnerManager.createSpinner(message, style, ds.writer, ds.colorSystem, theme)
	spinner.start()
	return spinner
}

// UpdateSpinner updates the message of an existing spinner
func (ds *displayService) UpdateSpinner(handle SpinnerHandle, message string) {
	if ds.config.QuietMode {
		return
	}

	if spinner := ds.spinnerManager.getSpinner(handle); spinner != nil {
		spinner.updateMessage(message)
	}
}

// StopSpinner stops a spinner and optionally displays a final message
func (ds *displayService) StopSpinner(handle SpinnerHandle, finalMessage string) {
	if ds.config.QuietMode {
		if finalMessage != "" {
			fmt.Fprintln(ds.writer, finalMessage)
		}
		return
	}

	if spinner := ds.spinnerManager.getSpinner(handle); spinner != nil {
		spinner.stop(finalMessage)
		ds.spinnerManager.removeSpinner(handle)
	}
}

// ShowProgress displays a simple progress indicator
func (ds *displayService) ShowProgress(current, total int, message string) {
	if ds.config.QuietMode {
		return
	}

	if total <= 0 {
		return
	}

	percentage := float64(current) / float64(total) * 100
	if percentage > 100 {
		percentage = 100
	}

	// Simple progress display
	var output string
	if ds.config.IsColorEnabled() && ds.colorSystem.IsColorSupported() {
		theme := ds.config.GetColorTheme()
		progressText := ds.colorSystem.Sprintf(theme.Info, "Progress: %.1f%% (%d/%d)", percentage, current, total)
		output = fmt.Sprintf("\r%s %s", progressText, message)
	} else {
		output = fmt.Sprintf("\rProgress: %.1f%% (%d/%d) %s", percentage, current, total, message)
	}

	fmt.Fprint(ds.writer, output)

	// Add newline if completed
	if current >= total {
		fmt.Fprintln(ds.writer)
	}
}

// NewProgressBar creates a new progress bar with the service's configuration
func (ds *displayService) NewProgressBar(total int, message string) *ProgressBar {
	theme := ds.config.GetColorTheme()
	return NewProgressBar(total, message, ds.writer, ds.colorSystem, theme)
}

// NewMultiProgress creates a new multi-progress manager with the service's configuration
func (ds *displayService) NewMultiProgress() *MultiProgress {
	return NewMultiProgress(ds.writer)
}

// NewProgressTracker creates a new progress tracker with the service's configuration
func (ds *displayService) NewProgressTracker(phases []string) *ProgressTracker {
	theme := ds.config.GetColorTheme()
	return NewProgressTracker(phases, ds.writer, ds.colorSystem, theme)
}

// NewTableFormatter creates a new table formatter with the service's configuration
func (ds *displayService) NewTableFormatter() TableFormatter {
	theme := ds.config.GetColorTheme()
	return NewTableFormatter(ds.colorSystem, theme)
}

// NewSchemaDiffPresenter creates a new schema diff presenter with the service's configuration
func (ds *displayService) NewSchemaDiffPresenter() *SchemaDiffPresenter {
	theme := ds.config.GetColorTheme()
	return NewSchemaDiffPresenter(ds.colorSystem, ds.iconSystem, theme)
}

// NewSectionFormatter creates a new section formatter with the service's configuration
func (ds *displayService) NewSectionFormatter() *SectionFormatter {
	theme := ds.config.GetColorTheme()
	return NewSectionFormatter(ds.colorSystem, ds.iconSystem, theme, ds.writer)
}

// RenderSection renders a single section using the service's configuration
func (ds *displayService) RenderSection(section *Section) {
	if ds.config.QuietMode {
		return
	}

	formatter := ds.NewSectionFormatter()
	formatter.RenderSection(section)
}

// RenderSections renders multiple sections using the service's configuration
func (ds *displayService) RenderSections(sections []*Section) {
	if ds.config.QuietMode {
		return
	}

	formatter := ds.NewSectionFormatter()
	formatter.RenderSections(sections)
}

// NewSQLHighlighter creates a new SQL highlighter with the service's configuration
func (ds *displayService) NewSQLHighlighter() *SQLHighlighter {
	theme := ds.config.GetColorTheme()
	return NewSQLHighlighter(ds.colorSystem, theme)
}

// NewOutputWriter creates a new output writer with the specified format
func (ds *displayService) NewOutputWriter(format OutputFormat) *OutputWriter {
	return NewOutputWriter(format, ds.writer)
}

// GetFormatterRegistry returns the formatter registry
func (ds *displayService) GetFormatterRegistry() *FormatterRegistry {
	return ds.formatterRegistry
}

// NewConfirmationDialog creates a new confirmation dialog with the service's configuration
func (ds *displayService) NewConfirmationDialog() *ConfirmationDialog {
	theme := ds.config.GetColorTheme()
	return NewConfirmationDialog(ds.colorSystem, ds.iconSystem, theme, ds.writer)
}

// NewConfirmationBuilder creates a new confirmation dialog builder with the service's configuration
func (ds *displayService) NewConfirmationBuilder() *ConfirmationBuilder {
	theme := ds.config.GetColorTheme()
	return NewConfirmationBuilder(ds.colorSystem, ds.iconSystem, theme, ds.writer)
}

// NewChangeReviewDialog creates a new change review dialog with the service's configuration
func (ds *displayService) NewChangeReviewDialog() *ChangeReviewDialog {
	theme := ds.config.GetColorTheme()
	return NewChangeReviewDialog(ds.colorSystem, ds.iconSystem, theme, ds.writer)
}

// noOpSpinner is a no-operation spinner for quiet mode
type noOpSpinner struct{}

func (n *noOpSpinner) ID() string {
	return "noop"
}

func (n *noOpSpinner) IsActive() bool {
	return false
}
