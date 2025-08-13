package display

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmationDialog represents an enhanced confirmation dialog
type ConfirmationDialog struct {
	Title          string
	Message        string
	Options        []ConfirmationOption
	DefaultOption  int
	IsDestructive  bool
	ShowWarning    bool
	WarningMessage string
	Details        []string
	colorSystem    ColorSystem
	iconSystem     IconSystem
	theme          ColorTheme
	writer         io.Writer
	reader         *bufio.Reader
}

// ConfirmationOption represents a single option in a confirmation dialog
type ConfirmationOption struct {
	Key         string
	Label       string
	Description string
	IsDefault   bool
	IsCancel    bool
}

// ConfirmationResult represents the result of a confirmation dialog
type ConfirmationResult struct {
	Confirmed   bool
	SelectedKey string
	ShowDetails bool
	Cancelled   bool
}

// NewConfirmationDialog creates a new enhanced confirmation dialog
func NewConfirmationDialog(colorSystem ColorSystem, iconSystem IconSystem, theme ColorTheme, writer io.Writer) *ConfirmationDialog {
	return &ConfirmationDialog{
		colorSystem: colorSystem,
		iconSystem:  iconSystem,
		theme:       theme,
		writer:      writer,
		reader:      bufio.NewReader(os.Stdin),
	}
}

// SetTitle sets the dialog title
func (cd *ConfirmationDialog) SetTitle(title string) *ConfirmationDialog {
	cd.Title = title
	return cd
}

// SetMessage sets the main message
func (cd *ConfirmationDialog) SetMessage(message string) *ConfirmationDialog {
	cd.Message = message
	return cd
}

// AddOption adds a confirmation option
func (cd *ConfirmationDialog) AddOption(key, label, description string, isDefault bool) *ConfirmationDialog {
	option := ConfirmationOption{
		Key:         key,
		Label:       label,
		Description: description,
		IsDefault:   isDefault,
		IsCancel:    strings.ToLower(key) == "n" || strings.ToLower(key) == "no",
	}
	cd.Options = append(cd.Options, option)

	if isDefault {
		cd.DefaultOption = len(cd.Options) - 1
	}

	return cd
}

// AddCancelOption adds a confirmation option that represents a cancel/negative action
func (cd *ConfirmationDialog) AddCancelOption(key, label, description string, isDefault bool) *ConfirmationDialog {
	option := ConfirmationOption{
		Key:         key,
		Label:       label,
		Description: description,
		IsDefault:   isDefault,
		IsCancel:    true,
	}
	cd.Options = append(cd.Options, option)

	if isDefault {
		cd.DefaultOption = len(cd.Options) - 1
	}

	return cd
}

// SetDestructive marks the dialog as representing a destructive operation
func (cd *ConfirmationDialog) SetDestructive(isDestructive bool) *ConfirmationDialog {
	cd.IsDestructive = isDestructive
	return cd
}

// SetWarning adds a warning message to the dialog
func (cd *ConfirmationDialog) SetWarning(message string) *ConfirmationDialog {
	cd.ShowWarning = true
	cd.WarningMessage = message
	return cd
}

// AddDetails adds detail lines that can be shown on request
func (cd *ConfirmationDialog) AddDetails(details ...string) *ConfirmationDialog {
	cd.Details = append(cd.Details, details...)
	return cd
}

// Show displays the confirmation dialog and returns the result
func (cd *ConfirmationDialog) Show() (*ConfirmationResult, error) {
	for {
		cd.render()

		input, err := cd.readInput()
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		result := cd.parseInput(input)
		if result != nil {
			return result, nil
		}

		// Invalid input, show error and continue
		cd.showError("Invalid input. Please try again.")
		fmt.Fprintln(cd.writer)
	}
}

// render displays the complete confirmation dialog
func (cd *ConfirmationDialog) render() {
	fmt.Fprintln(cd.writer)

	// Render title if present
	if cd.Title != "" {
		cd.renderTitle()
	}

	// Render destructive operation warning
	if cd.IsDestructive {
		cd.renderDestructiveWarning()
	}

	// Render custom warning if present
	if cd.ShowWarning && cd.WarningMessage != "" {
		cd.renderWarning()
	}

	// Render main message
	if cd.Message != "" {
		cd.renderMessage()
	}

	// Render options
	cd.renderOptions()

	// Render prompt
	cd.renderPrompt()
}

// renderTitle renders the dialog title with styling
func (cd *ConfirmationDialog) renderTitle() {
	titleIcon := cd.iconSystem.RenderIcon("info")
	title := fmt.Sprintf("%s %s", titleIcon, cd.Title)

	if cd.colorSystem.IsColorSupported() {
		title = cd.colorSystem.Colorize(title, cd.theme.Primary)
	}

	fmt.Fprintln(cd.writer, title)
	fmt.Fprintln(cd.writer, strings.Repeat("─", len(cd.Title)+4))
	fmt.Fprintln(cd.writer)
}

// renderDestructiveWarning renders a warning for destructive operations
func (cd *ConfirmationDialog) renderDestructiveWarning() {
	warningIcon := cd.iconSystem.RenderIcon("warning")
	warningText := fmt.Sprintf("%s DESTRUCTIVE OPERATION", warningIcon)

	if cd.colorSystem.IsColorSupported() {
		warningText = cd.colorSystem.Colorize(warningText, cd.theme.Error)
	}

	fmt.Fprintln(cd.writer, warningText)
	fmt.Fprintln(cd.writer, "This operation may result in data loss. Please review carefully.")
	fmt.Fprintln(cd.writer)
}

// renderWarning renders a custom warning message
func (cd *ConfirmationDialog) renderWarning() {
	warningIcon := cd.iconSystem.RenderIcon("warning")
	warningText := fmt.Sprintf("%s %s", warningIcon, cd.WarningMessage)

	if cd.colorSystem.IsColorSupported() {
		warningText = cd.colorSystem.Colorize(warningText, cd.theme.Warning)
	}

	fmt.Fprintln(cd.writer, warningText)
	fmt.Fprintln(cd.writer)
}

// renderMessage renders the main dialog message
func (cd *ConfirmationDialog) renderMessage() {
	fmt.Fprintln(cd.writer, cd.Message)
	fmt.Fprintln(cd.writer)
}

// renderOptions renders the available options with descriptions
func (cd *ConfirmationDialog) renderOptions() {
	if len(cd.Options) == 0 {
		return
	}

	fmt.Fprintln(cd.writer, "Options:")

	for _, option := range cd.Options {
		cd.renderOption(option)
	}

	// Add details option if details are available
	if len(cd.Details) > 0 {
		detailsOption := ConfirmationOption{
			Key:         "d",
			Label:       "details",
			Description: "Show detailed information",
		}
		cd.renderOption(detailsOption)
	}

	fmt.Fprintln(cd.writer)
}

// renderOption renders a single option with styling
func (cd *ConfirmationDialog) renderOption(option ConfirmationOption) {
	keyDisplay := fmt.Sprintf("[%s]", option.Key)

	// Highlight default option
	if option.IsDefault {
		if cd.colorSystem.IsColorSupported() {
			keyDisplay = cd.colorSystem.Colorize(keyDisplay, cd.theme.Highlight)
		} else {
			keyDisplay = fmt.Sprintf("%s (default)", keyDisplay)
		}
	}

	// Color code based on option type
	var color Color
	if option.IsCancel {
		color = cd.theme.Muted
	} else if option.IsDefault {
		color = cd.theme.Success
	} else {
		color = cd.theme.Info
	}

	optionText := fmt.Sprintf("  %s %s", keyDisplay, option.Label)
	if cd.colorSystem.IsColorSupported() {
		optionText = cd.colorSystem.Colorize(optionText, color)
	}

	fmt.Fprintln(cd.writer, optionText)

	// Show description if available
	if option.Description != "" {
		description := fmt.Sprintf("      %s", option.Description)
		if cd.colorSystem.IsColorSupported() {
			description = cd.colorSystem.Colorize(description, cd.theme.Muted)
		}
		fmt.Fprintln(cd.writer, description)
	}
}

// renderPrompt renders the input prompt
func (cd *ConfirmationDialog) renderPrompt() {
	// Build prompt text with available keys
	var keys []string
	for _, option := range cd.Options {
		if option.IsDefault {
			keys = append(keys, strings.ToUpper(option.Key))
		} else {
			keys = append(keys, option.Key)
		}
	}

	// Add details key if available
	if len(cd.Details) > 0 {
		keys = append(keys, "d")
	}

	promptText := fmt.Sprintf("Choose [%s]: ", strings.Join(keys, "/"))

	if cd.colorSystem.IsColorSupported() {
		promptText = cd.colorSystem.Colorize(promptText, cd.theme.Primary)
	}

	fmt.Fprint(cd.writer, promptText)
}

// readInput reads user input from stdin
func (cd *ConfirmationDialog) readInput() (string, error) {
	input, err := cd.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

// parseInput parses user input and returns the appropriate result
func (cd *ConfirmationDialog) parseInput(input string) *ConfirmationResult {
	input = strings.ToLower(strings.TrimSpace(input))

	// Handle empty input (use default)
	if input == "" && cd.DefaultOption >= 0 && cd.DefaultOption < len(cd.Options) {
		defaultOption := cd.Options[cd.DefaultOption]
		return &ConfirmationResult{
			Confirmed:   !defaultOption.IsCancel,
			SelectedKey: defaultOption.Key,
			Cancelled:   defaultOption.IsCancel,
		}
	}

	// Handle details request
	if input == "d" || input == "details" {
		if len(cd.Details) > 0 {
			cd.showDetails()
			return nil // Continue dialog
		}
	}

	// Check against available options
	for _, option := range cd.Options {
		if strings.ToLower(option.Key) == input || strings.ToLower(option.Label) == input {
			return &ConfirmationResult{
				Confirmed:   !option.IsCancel,
				SelectedKey: option.Key,
				Cancelled:   option.IsCancel,
			}
		}
	}

	return nil // Invalid input
}

// showDetails displays the detailed information
func (cd *ConfirmationDialog) showDetails() {
	fmt.Fprintln(cd.writer)

	detailsIcon := cd.iconSystem.RenderIcon("info")
	title := fmt.Sprintf("%s Detailed Information", detailsIcon)

	if cd.colorSystem.IsColorSupported() {
		title = cd.colorSystem.Colorize(title, cd.theme.Info)
	}

	fmt.Fprintln(cd.writer, title)
	fmt.Fprintln(cd.writer, strings.Repeat("─", 30))

	for i, detail := range cd.Details {
		fmt.Fprintf(cd.writer, "%d. %s\n", i+1, detail)
	}

	fmt.Fprintln(cd.writer, strings.Repeat("─", 30))
	fmt.Fprintln(cd.writer)
}

// showError displays an error message
func (cd *ConfirmationDialog) showError(message string) {
	errorIcon := cd.iconSystem.RenderIcon("error")
	errorText := fmt.Sprintf("%s %s", errorIcon, message)

	if cd.colorSystem.IsColorSupported() {
		errorText = cd.colorSystem.Colorize(errorText, cd.theme.Error)
	}

	fmt.Fprintln(cd.writer, errorText)
}

// ConfirmationBuilder provides a fluent interface for building confirmation dialogs
type ConfirmationBuilder struct {
	dialog *ConfirmationDialog
}

// NewConfirmationBuilder creates a new confirmation dialog builder
func NewConfirmationBuilder(colorSystem ColorSystem, iconSystem IconSystem, theme ColorTheme, writer io.Writer) *ConfirmationBuilder {
	return &ConfirmationBuilder{
		dialog: NewConfirmationDialog(colorSystem, iconSystem, theme, writer),
	}
}

// Title sets the dialog title
func (cb *ConfirmationBuilder) Title(title string) *ConfirmationBuilder {
	cb.dialog.SetTitle(title)
	return cb
}

// Message sets the main message
func (cb *ConfirmationBuilder) Message(message string) *ConfirmationBuilder {
	cb.dialog.SetMessage(message)
	return cb
}

// YesNo adds standard yes/no options with no as default
func (cb *ConfirmationBuilder) YesNo() *ConfirmationBuilder {
	cb.dialog.AddOption("y", "yes", "Proceed with the operation", false)
	cb.dialog.AddOption("n", "no", "Cancel the operation", true)
	return cb
}

// YesNoDefault adds standard yes/no options with yes as default
func (cb *ConfirmationBuilder) YesNoDefault() *ConfirmationBuilder {
	cb.dialog.AddOption("y", "yes", "Proceed with the operation", true)
	cb.dialog.AddOption("n", "no", "Cancel the operation", false)
	return cb
}

// Option adds a custom option
func (cb *ConfirmationBuilder) Option(key, label, description string, isDefault bool) *ConfirmationBuilder {
	cb.dialog.AddOption(key, label, description, isDefault)
	return cb
}

// CancelOption adds a custom cancel option
func (cb *ConfirmationBuilder) CancelOption(key, label, description string, isDefault bool) *ConfirmationBuilder {
	cb.dialog.AddCancelOption(key, label, description, isDefault)
	return cb
}

// Destructive marks the dialog as representing a destructive operation
func (cb *ConfirmationBuilder) Destructive() *ConfirmationBuilder {
	cb.dialog.SetDestructive(true)
	return cb
}

// Warning adds a warning message
func (cb *ConfirmationBuilder) Warning(message string) *ConfirmationBuilder {
	cb.dialog.SetWarning(message)
	return cb
}

// Details adds detail lines
func (cb *ConfirmationBuilder) Details(details ...string) *ConfirmationBuilder {
	cb.dialog.AddDetails(details...)
	return cb
}

// Build returns the configured dialog
func (cb *ConfirmationBuilder) Build() *ConfirmationDialog {
	return cb.dialog
}

// Show builds and shows the dialog
func (cb *ConfirmationBuilder) Show() (*ConfirmationResult, error) {
	return cb.dialog.Show()
}
