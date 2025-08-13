package display

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Change represents a single change that can be reviewed
type Change struct {
	ID            string
	Type          ReviewChangeType
	Title         string
	Description   string
	Details       []string
	IsDestructive bool
	IsSelected    bool
	Category      string
	Impact        ChangeImpact
}

// ReviewChangeType represents the type of change in a review
type ReviewChangeType int

const (
	ReviewChangeTypeAdd ReviewChangeType = iota
	ReviewChangeTypeRemove
	ReviewChangeTypeModify
	ReviewChangeTypeRename
)

// ChangeImpact represents the impact level of a change
type ChangeImpact int

const (
	ChangeImpactLow ChangeImpact = iota
	ChangeImpactMedium
	ChangeImpactHigh
	ChangeImpactCritical
)

// ChangeReviewDialog provides an interface for reviewing and selecting changes
type ChangeReviewDialog struct {
	Title           string
	Message         string
	Changes         []*Change
	AllowIndividual bool
	ShowSummary     bool
	colorSystem     ColorSystem
	iconSystem      IconSystem
	theme           ColorTheme
	writer          io.Writer
}

// ChangeReviewResult represents the result of a change review
type ChangeReviewResult struct {
	Approved        bool
	SelectedChanges []*Change
	RejectedChanges []*Change
	Summary         *ReviewSummary
}

// ReviewSummary provides statistics about the review
type ReviewSummary struct {
	TotalChanges       int
	SelectedChanges    int
	RejectedChanges    int
	DestructiveChanges int
	ByType             map[ReviewChangeType]int
	ByImpact           map[ChangeImpact]int
}

// NewChangeReviewDialog creates a new change review dialog
func NewChangeReviewDialog(colorSystem ColorSystem, iconSystem IconSystem, theme ColorTheme, writer io.Writer) *ChangeReviewDialog {
	return &ChangeReviewDialog{
		colorSystem:     colorSystem,
		iconSystem:      iconSystem,
		theme:           theme,
		writer:          writer,
		AllowIndividual: true,
		ShowSummary:     true,
	}
}

// SetTitle sets the dialog title
func (crd *ChangeReviewDialog) SetTitle(title string) *ChangeReviewDialog {
	crd.Title = title
	return crd
}

// SetMessage sets the main message
func (crd *ChangeReviewDialog) SetMessage(message string) *ChangeReviewDialog {
	crd.Message = message
	return crd
}

// AddChange adds a change to be reviewed
func (crd *ChangeReviewDialog) AddChange(change *Change) *ChangeReviewDialog {
	if change.ID == "" {
		change.ID = fmt.Sprintf("change_%d", len(crd.Changes)+1)
	}
	// Don't override IsSelected - respect the value that was set
	crd.Changes = append(crd.Changes, change)
	return crd
}

// SetAllowIndividual sets whether individual change selection is allowed
func (crd *ChangeReviewDialog) SetAllowIndividual(allow bool) *ChangeReviewDialog {
	crd.AllowIndividual = allow
	return crd
}

// Show displays the change review dialog and returns the result
func (crd *ChangeReviewDialog) Show() (*ChangeReviewResult, error) {
	for {
		crd.render()

		input, err := crd.readInput()
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		result := crd.parseInput(input)
		if result != nil {
			return result, nil
		}

		// Invalid input, show error and continue
		crd.showError("Invalid input. Please try again.")
		fmt.Fprintln(crd.writer)
	}
}

// render displays the complete change review dialog
func (crd *ChangeReviewDialog) render() {
	fmt.Fprintln(crd.writer)

	// Render title
	if crd.Title != "" {
		crd.renderTitle()
	}

	// Render message
	if crd.Message != "" {
		crd.renderMessage()
	}

	// Render summary statistics
	if crd.ShowSummary {
		crd.renderSummary()
	}

	// Render changes
	crd.renderChanges()

	// Render options
	crd.renderOptions()

	// Render prompt
	crd.renderPrompt()
}

// renderTitle renders the dialog title
func (crd *ChangeReviewDialog) renderTitle() {
	titleIcon := crd.iconSystem.RenderIcon("info")
	title := fmt.Sprintf("%s %s", titleIcon, crd.Title)

	if crd.colorSystem.IsColorSupported() {
		title = crd.colorSystem.Colorize(title, crd.theme.Primary)
	}

	fmt.Fprintln(crd.writer, title)
	fmt.Fprintln(crd.writer, strings.Repeat("═", len(crd.Title)+4))
	fmt.Fprintln(crd.writer)
}

// renderMessage renders the main message
func (crd *ChangeReviewDialog) renderMessage() {
	fmt.Fprintln(crd.writer, crd.Message)
	fmt.Fprintln(crd.writer)
}

// renderSummary renders summary statistics
func (crd *ChangeReviewDialog) renderSummary() {
	summary := crd.calculateSummary()

	summaryIcon := crd.iconSystem.RenderIcon("info")
	summaryTitle := fmt.Sprintf("%s Summary", summaryIcon)

	if crd.colorSystem.IsColorSupported() {
		summaryTitle = crd.colorSystem.Colorize(summaryTitle, crd.theme.Info)
	}

	fmt.Fprintln(crd.writer, summaryTitle)
	fmt.Fprintln(crd.writer, strings.Repeat("─", 20))

	// Total changes
	fmt.Fprintf(crd.writer, "Total changes: %d\n", summary.TotalChanges)
	fmt.Fprintf(crd.writer, "Selected: %d\n", summary.SelectedChanges)

	// Destructive changes warning
	if summary.DestructiveChanges > 0 {
		warningIcon := crd.iconSystem.RenderIcon("warning")
		warningText := fmt.Sprintf("%s Destructive changes: %d", warningIcon, summary.DestructiveChanges)

		if crd.colorSystem.IsColorSupported() {
			warningText = crd.colorSystem.Colorize(warningText, crd.theme.Warning)
		}

		fmt.Fprintln(crd.writer, warningText)
	}

	// By type
	fmt.Fprintln(crd.writer, "\nBy type:")
	for changeType, count := range summary.ByType {
		if count > 0 {
			typeIcon := crd.getChangeTypeIcon(changeType)
			typeName := crd.getChangeTypeName(changeType)
			fmt.Fprintf(crd.writer, "  %s %s: %d\n", typeIcon, typeName, count)
		}
	}

	// By impact
	fmt.Fprintln(crd.writer, "\nBy impact:")
	for impact, count := range summary.ByImpact {
		if count > 0 {
			impactIcon := crd.getImpactIcon(impact)
			impactName := crd.getImpactName(impact)
			impactText := fmt.Sprintf("  %s %s: %d", impactIcon, impactName, count)

			if crd.colorSystem.IsColorSupported() {
				impactText = crd.colorSystem.Colorize(impactText, crd.getImpactColor(impact))
			}

			fmt.Fprintln(crd.writer, impactText)
		}
	}

	fmt.Fprintln(crd.writer)
}

// renderChanges renders the list of changes
func (crd *ChangeReviewDialog) renderChanges() {
	changesIcon := crd.iconSystem.RenderIcon("list")
	changesTitle := fmt.Sprintf("%s Changes", changesIcon)

	if crd.colorSystem.IsColorSupported() {
		changesTitle = crd.colorSystem.Colorize(changesTitle, crd.theme.Primary)
	}

	fmt.Fprintln(crd.writer, changesTitle)
	fmt.Fprintln(crd.writer, strings.Repeat("─", 20))

	for i, change := range crd.Changes {
		crd.renderChange(i+1, change)
	}

	fmt.Fprintln(crd.writer)
}

// renderChange renders a single change
func (crd *ChangeReviewDialog) renderChange(index int, change *Change) {
	// Selection indicator
	var selectionIcon string
	if change.IsSelected {
		selectionIcon = crd.iconSystem.RenderIcon("success")
		if crd.colorSystem.IsColorSupported() {
			selectionIcon = crd.colorSystem.Colorize(selectionIcon, crd.theme.Success)
		}
	} else {
		selectionIcon = crd.iconSystem.RenderIcon("error")
		if crd.colorSystem.IsColorSupported() {
			selectionIcon = crd.colorSystem.Colorize(selectionIcon, crd.theme.Muted)
		}
	}

	// Change type icon
	typeIcon := crd.getChangeTypeIcon(change.Type)

	// Impact indicator
	impactIcon := crd.getImpactIcon(change.Impact)
	if crd.colorSystem.IsColorSupported() {
		impactIcon = crd.colorSystem.Colorize(impactIcon, crd.getImpactColor(change.Impact))
	}

	// Destructive warning
	destructiveWarning := ""
	if change.IsDestructive {
		destructiveWarning = crd.iconSystem.RenderIcon("warning")
		if crd.colorSystem.IsColorSupported() {
			destructiveWarning = crd.colorSystem.Colorize(destructiveWarning, crd.theme.Error)
		}
		destructiveWarning = " " + destructiveWarning
	}

	// Main change line
	changeText := fmt.Sprintf("%2d. %s %s %s %s%s",
		index, selectionIcon, typeIcon, impactIcon, change.Title, destructiveWarning)

	fmt.Fprintln(crd.writer, changeText)

	// Description
	if change.Description != "" {
		description := fmt.Sprintf("     %s", change.Description)
		if crd.colorSystem.IsColorSupported() {
			description = crd.colorSystem.Colorize(description, crd.theme.Muted)
		}
		fmt.Fprintln(crd.writer, description)
	}

	// Category
	if change.Category != "" {
		category := fmt.Sprintf("     Category: %s", change.Category)
		if crd.colorSystem.IsColorSupported() {
			category = crd.colorSystem.Colorize(category, crd.theme.Muted)
		}
		fmt.Fprintln(crd.writer, category)
	}
}

// renderOptions renders the available options
func (crd *ChangeReviewDialog) renderOptions() {
	fmt.Fprintln(crd.writer, "Options:")

	// Batch options
	fmt.Fprintln(crd.writer, "  [a] all      - Select all changes")
	fmt.Fprintln(crd.writer, "  [n] none     - Deselect all changes")
	fmt.Fprintln(crd.writer, "  [i] invert   - Invert selection")

	// Individual options (if allowed)
	if crd.AllowIndividual {
		fmt.Fprintln(crd.writer, "  [1-9] toggle - Toggle individual change")
		fmt.Fprintln(crd.writer, "  [1-9]+ add   - Add changes to selection (e.g., '1,3,5')")
		fmt.Fprintln(crd.writer, "  [1-9]- remove- Remove changes from selection (e.g., '2,4-')")
	}

	// Detail options
	fmt.Fprintln(crd.writer, "  [d] details  - Show detailed information for selected changes")
	fmt.Fprintln(crd.writer, "  [d1-9] detail- Show details for specific change")

	// Confirmation options
	fmt.Fprintln(crd.writer, "  [y] yes      - Apply selected changes")
	fmt.Fprintln(crd.writer, "  [q] quit     - Cancel without applying changes")

	fmt.Fprintln(crd.writer)
}

// renderPrompt renders the input prompt
func (crd *ChangeReviewDialog) renderPrompt() {
	selectedCount := 0
	for _, change := range crd.Changes {
		if change.IsSelected {
			selectedCount++
		}
	}

	promptText := fmt.Sprintf("Review changes (%d/%d selected) [a/n/i/1-9/d/y/q]: ",
		selectedCount, len(crd.Changes))

	if crd.colorSystem.IsColorSupported() {
		promptText = crd.colorSystem.Colorize(promptText, crd.theme.Primary)
	}

	fmt.Fprint(crd.writer, promptText)
}

// readInput reads user input
func (crd *ChangeReviewDialog) readInput() (string, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// parseInput parses user input and returns the appropriate result
func (crd *ChangeReviewDialog) parseInput(input string) *ChangeReviewResult {
	input = strings.ToLower(strings.TrimSpace(input))

	switch {
	case input == "a" || input == "all":
		crd.selectAll()
		return nil // Continue dialog

	case input == "n" || input == "none":
		crd.selectNone()
		return nil // Continue dialog

	case input == "i" || input == "invert":
		crd.invertSelection()
		return nil // Continue dialog

	case input == "d" || input == "details":
		crd.showSelectedDetails()
		return nil // Continue dialog

	case strings.HasPrefix(input, "d") && len(input) > 1:
		// Show details for specific change
		if indexStr := input[1:]; indexStr != "" {
			if index, err := strconv.Atoi(indexStr); err == nil && index > 0 && index <= len(crd.Changes) {
				crd.showChangeDetails(crd.Changes[index-1])
			}
		}
		return nil // Continue dialog

	case input == "y" || input == "yes":
		return crd.buildResult(true)

	case input == "q" || input == "quit":
		return crd.buildResult(false)

	default:
		// Handle individual change selection
		if crd.AllowIndividual {
			if crd.handleIndividualSelection(input) {
				return nil // Continue dialog
			}
		}
	}

	return nil // Invalid input
}

// handleIndividualSelection handles individual change selection commands
func (crd *ChangeReviewDialog) handleIndividualSelection(input string) bool {
	// Handle single number toggle
	if index, err := strconv.Atoi(input); err == nil && index > 0 && index <= len(crd.Changes) {
		crd.Changes[index-1].IsSelected = !crd.Changes[index-1].IsSelected
		return true
	}

	// Handle comma-separated list with + or - suffix
	if strings.HasSuffix(input, "+") || strings.HasSuffix(input, "-") {
		isAdd := strings.HasSuffix(input, "+")
		indexStr := strings.TrimSuffix(strings.TrimSuffix(input, "+"), "-")

		indices := strings.Split(indexStr, ",")
		for _, idxStr := range indices {
			if index, err := strconv.Atoi(strings.TrimSpace(idxStr)); err == nil && index > 0 && index <= len(crd.Changes) {
				crd.Changes[index-1].IsSelected = isAdd
			}
		}
		return true
	}

	return false
}

// selectAll selects all changes
func (crd *ChangeReviewDialog) selectAll() {
	for _, change := range crd.Changes {
		change.IsSelected = true
	}
}

// selectNone deselects all changes
func (crd *ChangeReviewDialog) selectNone() {
	for _, change := range crd.Changes {
		change.IsSelected = false
	}
}

// invertSelection inverts the current selection
func (crd *ChangeReviewDialog) invertSelection() {
	for _, change := range crd.Changes {
		change.IsSelected = !change.IsSelected
	}
}

// showSelectedDetails shows details for all selected changes
func (crd *ChangeReviewDialog) showSelectedDetails() {
	fmt.Fprintln(crd.writer)

	detailsIcon := crd.iconSystem.RenderIcon("info")
	title := fmt.Sprintf("%s Selected Change Details", detailsIcon)

	if crd.colorSystem.IsColorSupported() {
		title = crd.colorSystem.Colorize(title, crd.theme.Info)
	}

	fmt.Fprintln(crd.writer, title)
	fmt.Fprintln(crd.writer, strings.Repeat("═", 30))

	selectedCount := 0
	for i, change := range crd.Changes {
		if change.IsSelected {
			selectedCount++
			fmt.Fprintf(crd.writer, "\n%d. %s\n", i+1, change.Title)
			fmt.Fprintln(crd.writer, strings.Repeat("─", len(change.Title)+4))

			if change.Description != "" {
				fmt.Fprintf(crd.writer, "Description: %s\n", change.Description)
			}

			if len(change.Details) > 0 {
				fmt.Fprintln(crd.writer, "Details:")
				for _, detail := range change.Details {
					fmt.Fprintf(crd.writer, "  • %s\n", detail)
				}
			}

			fmt.Fprintf(crd.writer, "Type: %s\n", crd.getChangeTypeName(change.Type))
			fmt.Fprintf(crd.writer, "Impact: %s\n", crd.getImpactName(change.Impact))

			if change.IsDestructive {
				warningText := "⚠ DESTRUCTIVE OPERATION - May result in data loss"
				if crd.colorSystem.IsColorSupported() {
					warningText = crd.colorSystem.Colorize(warningText, crd.theme.Error)
				}
				fmt.Fprintln(crd.writer, warningText)
			}
		}
	}

	if selectedCount == 0 {
		fmt.Fprintln(crd.writer, "No changes selected.")
	}

	fmt.Fprintln(crd.writer, strings.Repeat("═", 30))
	fmt.Fprintln(crd.writer)
}

// showChangeDetails shows details for a specific change
func (crd *ChangeReviewDialog) showChangeDetails(change *Change) {
	fmt.Fprintln(crd.writer)

	detailsIcon := crd.iconSystem.RenderIcon("info")
	title := fmt.Sprintf("%s Change Details: %s", detailsIcon, change.Title)

	if crd.colorSystem.IsColorSupported() {
		title = crd.colorSystem.Colorize(title, crd.theme.Info)
	}

	fmt.Fprintln(crd.writer, title)
	fmt.Fprintln(crd.writer, strings.Repeat("═", len(change.Title)+20))

	if change.Description != "" {
		fmt.Fprintf(crd.writer, "Description: %s\n", change.Description)
	}

	if len(change.Details) > 0 {
		fmt.Fprintln(crd.writer, "\nDetailed Information:")
		for i, detail := range change.Details {
			fmt.Fprintf(crd.writer, "%d. %s\n", i+1, detail)
		}
	}

	fmt.Fprintf(crd.writer, "\nType: %s\n", crd.getChangeTypeName(change.Type))
	fmt.Fprintf(crd.writer, "Impact: %s\n", crd.getImpactName(change.Impact))

	if change.Category != "" {
		fmt.Fprintf(crd.writer, "Category: %s\n", change.Category)
	}

	if change.IsDestructive {
		warningText := "\n⚠ DESTRUCTIVE OPERATION - This change may result in data loss"
		if crd.colorSystem.IsColorSupported() {
			warningText = crd.colorSystem.Colorize(warningText, crd.theme.Error)
		}
		fmt.Fprintln(crd.writer, warningText)
	}

	fmt.Fprintln(crd.writer, strings.Repeat("═", len(change.Title)+20))
	fmt.Fprintln(crd.writer)
}

// buildResult builds the final result
func (crd *ChangeReviewDialog) buildResult(approved bool) *ChangeReviewResult {
	var selectedChanges, rejectedChanges []*Change

	for _, change := range crd.Changes {
		if change.IsSelected {
			selectedChanges = append(selectedChanges, change)
		} else {
			rejectedChanges = append(rejectedChanges, change)
		}
	}

	return &ChangeReviewResult{
		Approved:        approved && len(selectedChanges) > 0,
		SelectedChanges: selectedChanges,
		RejectedChanges: rejectedChanges,
		Summary:         crd.calculateSummary(),
	}
}

// calculateSummary calculates summary statistics
func (crd *ChangeReviewDialog) calculateSummary() *ReviewSummary {
	summary := &ReviewSummary{
		TotalChanges: len(crd.Changes),
		ByType:       make(map[ReviewChangeType]int),
		ByImpact:     make(map[ChangeImpact]int),
	}

	for _, change := range crd.Changes {
		if change.IsSelected {
			summary.SelectedChanges++
		} else {
			summary.RejectedChanges++
		}

		if change.IsDestructive {
			summary.DestructiveChanges++
		}

		summary.ByType[change.Type]++
		summary.ByImpact[change.Impact]++
	}

	return summary
}

// Helper methods for rendering

func (crd *ChangeReviewDialog) getChangeTypeIcon(changeType ReviewChangeType) string {
	switch changeType {
	case ReviewChangeTypeAdd:
		return crd.iconSystem.RenderIcon("add")
	case ReviewChangeTypeRemove:
		return crd.iconSystem.RenderIcon("remove")
	case ReviewChangeTypeModify:
		return crd.iconSystem.RenderIcon("modify")
	case ReviewChangeTypeRename:
		return crd.iconSystem.RenderIcon("modify")
	default:
		return crd.iconSystem.RenderIcon("info")
	}
}

func (crd *ChangeReviewDialog) getChangeTypeName(changeType ReviewChangeType) string {
	switch changeType {
	case ReviewChangeTypeAdd:
		return "Add"
	case ReviewChangeTypeRemove:
		return "Remove"
	case ReviewChangeTypeModify:
		return "Modify"
	case ReviewChangeTypeRename:
		return "Rename"
	default:
		return "Unknown"
	}
}

func (crd *ChangeReviewDialog) getImpactIcon(impact ChangeImpact) string {
	switch impact {
	case ChangeImpactLow:
		return crd.iconSystem.RenderIcon("info")
	case ChangeImpactMedium:
		return crd.iconSystem.RenderIcon("warning")
	case ChangeImpactHigh:
		return crd.iconSystem.RenderIcon("warning")
	case ChangeImpactCritical:
		return crd.iconSystem.RenderIcon("error")
	default:
		return crd.iconSystem.RenderIcon("info")
	}
}

func (crd *ChangeReviewDialog) getImpactName(impact ChangeImpact) string {
	switch impact {
	case ChangeImpactLow:
		return "Low"
	case ChangeImpactMedium:
		return "Medium"
	case ChangeImpactHigh:
		return "High"
	case ChangeImpactCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

func (crd *ChangeReviewDialog) getImpactColor(impact ChangeImpact) Color {
	switch impact {
	case ChangeImpactLow:
		return crd.theme.Info
	case ChangeImpactMedium:
		return crd.theme.Warning
	case ChangeImpactHigh:
		return crd.theme.Warning
	case ChangeImpactCritical:
		return crd.theme.Error
	default:
		return crd.theme.Info
	}
}

func (crd *ChangeReviewDialog) showError(message string) {
	errorIcon := crd.iconSystem.RenderIcon("error")
	errorText := fmt.Sprintf("%s %s", errorIcon, message)

	if crd.colorSystem.IsColorSupported() {
		errorText = crd.colorSystem.Colorize(errorText, crd.theme.Error)
	}

	fmt.Fprintln(crd.writer, errorText)
}
