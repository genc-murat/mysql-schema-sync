package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestChangeReviewDialog_Basic(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetTitle("Review Changes")
	dialog.SetMessage("Please review the following changes:")

	// Add some test changes
	change1 := &Change{
		ID:          "1",
		Type:        ReviewChangeTypeAdd,
		Title:       "Add table users",
		Description: "Create new users table with basic fields",
		Details:     []string{"id INT PRIMARY KEY", "name VARCHAR(255)", "email VARCHAR(255)"},
		Impact:      ChangeImpactMedium,
		Category:    "Schema",
		IsSelected:  true, // Default to selected for basic test
	}

	change2 := &Change{
		ID:            "2",
		Type:          ReviewChangeTypeRemove,
		Title:         "Drop column old_field",
		Description:   "Remove deprecated old_field column",
		Details:       []string{"ALTER TABLE products DROP COLUMN old_field"},
		Impact:        ChangeImpactHigh,
		IsDestructive: true,
		Category:      "Schema",
		IsSelected:    true, // Default to selected for basic test
	}

	dialog.AddChange(change1)
	dialog.AddChange(change2)

	// Test rendering
	dialog.render()

	output_str := output.String()

	// Check that title is rendered
	if !strings.Contains(output_str, "Review Changes") {
		t.Error("Title not found in output")
	}

	// Check that changes are rendered
	if !strings.Contains(output_str, "Add table users") {
		t.Error("Change 1 not found in output")
	}

	if !strings.Contains(output_str, "Drop column old_field") {
		t.Error("Change 2 not found in output")
	}

	// Check that summary is rendered
	if !strings.Contains(output_str, "Total changes: 2") {
		t.Error("Summary not found in output")
	}

	// Check that destructive warning is shown
	if !strings.Contains(output_str, "Destructive changes: 1") {
		t.Error("Destructive warning not found in output")
	}
}

func TestChangeReviewDialog_SelectionOperations(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})

	// Add test changes
	for i := 1; i <= 3; i++ {
		change := &Change{
			ID:    string(rune('0' + i)),
			Type:  ReviewChangeTypeAdd,
			Title: "Test change " + string(rune('0'+i)),
		}
		dialog.AddChange(change)
	}

	// Test select all
	dialog.selectAll()
	for i, change := range dialog.Changes {
		if !change.IsSelected {
			t.Errorf("Change %d should be selected after selectAll", i+1)
		}
	}

	// Test select none
	dialog.selectNone()
	for i, change := range dialog.Changes {
		if change.IsSelected {
			t.Errorf("Change %d should not be selected after selectNone", i+1)
		}
	}

	// Test invert selection
	dialog.Changes[0].IsSelected = true
	dialog.Changes[1].IsSelected = false
	dialog.Changes[2].IsSelected = true

	dialog.invertSelection()

	if dialog.Changes[0].IsSelected {
		t.Error("Change 1 should not be selected after invert")
	}
	if !dialog.Changes[1].IsSelected {
		t.Error("Change 2 should be selected after invert")
	}
	if dialog.Changes[2].IsSelected {
		t.Error("Change 3 should not be selected after invert")
	}
}

func TestChangeReviewDialog_IndividualSelection(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})

	// Add test changes
	for i := 1; i <= 5; i++ {
		change := &Change{
			ID:         string(rune('0' + i)),
			Type:       ReviewChangeTypeAdd,
			Title:      "Test change " + string(rune('0'+i)),
			IsSelected: false, // Start with none selected
		}
		dialog.AddChange(change)
	}

	// Test single number toggle
	if !dialog.handleIndividualSelection("1") {
		t.Error("Should handle single number selection")
	}
	if !dialog.Changes[0].IsSelected {
		t.Error("Change 1 should be selected")
	}

	// Test toggle again
	if !dialog.handleIndividualSelection("1") {
		t.Error("Should handle single number toggle")
	}
	if dialog.Changes[0].IsSelected {
		t.Error("Change 1 should be deselected after toggle")
	}

	// Test add multiple
	if !dialog.handleIndividualSelection("1,3,5+") {
		t.Error("Should handle multiple add selection")
	}
	if !dialog.Changes[0].IsSelected || !dialog.Changes[2].IsSelected || !dialog.Changes[4].IsSelected {
		t.Error("Changes 1, 3, 5 should be selected")
	}
	if dialog.Changes[1].IsSelected || dialog.Changes[3].IsSelected {
		t.Error("Changes 2, 4 should not be selected")
	}

	// Test remove multiple
	if !dialog.handleIndividualSelection("1,3-") {
		t.Error("Should handle multiple remove selection")
	}
	if dialog.Changes[0].IsSelected || dialog.Changes[2].IsSelected {
		t.Error("Changes 1, 3 should be deselected")
	}
	if !dialog.Changes[4].IsSelected {
		t.Error("Change 5 should still be selected")
	}
}

func TestChangeReviewDialog_ParseInput(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})

	// Add test changes
	change := &Change{
		ID:         "1",
		Type:       ReviewChangeTypeAdd,
		Title:      "Test change",
		IsSelected: true,
	}
	dialog.AddChange(change)

	// Test yes input
	result := dialog.parseInput("y")
	if result == nil || !result.Approved {
		t.Error("Should approve with 'y' input")
	}
	if len(result.SelectedChanges) != 1 {
		t.Error("Should have 1 selected change")
	}

	// Reset for next test
	dialog.Changes[0].IsSelected = false

	// Test quit input
	result = dialog.parseInput("q")
	if result == nil || result.Approved {
		t.Error("Should not approve with 'q' input")
	}

	// Test selection commands (should return nil to continue dialog)
	result = dialog.parseInput("a")
	if result != nil {
		t.Error("Selection commands should return nil to continue dialog")
	}
	if !dialog.Changes[0].IsSelected {
		t.Error("'a' command should select all changes")
	}

	result = dialog.parseInput("n")
	if result != nil {
		t.Error("Selection commands should return nil to continue dialog")
	}
	if dialog.Changes[0].IsSelected {
		t.Error("'n' command should deselect all changes")
	}
}

func TestChangeReviewDialog_Summary(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})

	// Add test changes with different types and impacts
	changes := []*Change{
		{Type: ReviewChangeTypeAdd, Impact: ChangeImpactLow, IsSelected: true},
		{Type: ReviewChangeTypeAdd, Impact: ChangeImpactMedium, IsSelected: true},
		{Type: ReviewChangeTypeRemove, Impact: ChangeImpactHigh, IsSelected: false, IsDestructive: true},
		{Type: ReviewChangeTypeModify, Impact: ChangeImpactCritical, IsSelected: true, IsDestructive: true},
	}

	for _, change := range changes {
		dialog.AddChange(change)
	}

	summary := dialog.calculateSummary()

	// Check totals
	if summary.TotalChanges != 4 {
		t.Errorf("Expected 4 total changes, got %d", summary.TotalChanges)
	}
	if summary.SelectedChanges != 3 {
		t.Errorf("Expected 3 selected changes, got %d", summary.SelectedChanges)
	}
	if summary.RejectedChanges != 1 {
		t.Errorf("Expected 1 rejected change, got %d", summary.RejectedChanges)
	}
	if summary.DestructiveChanges != 2 {
		t.Errorf("Expected 2 destructive changes, got %d", summary.DestructiveChanges)
	}

	// Check by type
	if summary.ByType[ReviewChangeTypeAdd] != 2 {
		t.Errorf("Expected 2 add changes, got %d", summary.ByType[ReviewChangeTypeAdd])
	}
	if summary.ByType[ReviewChangeTypeRemove] != 1 {
		t.Errorf("Expected 1 remove change, got %d", summary.ByType[ReviewChangeTypeRemove])
	}
	if summary.ByType[ReviewChangeTypeModify] != 1 {
		t.Errorf("Expected 1 modify change, got %d", summary.ByType[ReviewChangeTypeModify])
	}

	// Check by impact
	if summary.ByImpact[ChangeImpactLow] != 1 {
		t.Errorf("Expected 1 low impact change, got %d", summary.ByImpact[ChangeImpactLow])
	}
	if summary.ByImpact[ChangeImpactMedium] != 1 {
		t.Errorf("Expected 1 medium impact change, got %d", summary.ByImpact[ChangeImpactMedium])
	}
	if summary.ByImpact[ChangeImpactHigh] != 1 {
		t.Errorf("Expected 1 high impact change, got %d", summary.ByImpact[ChangeImpactHigh])
	}
	if summary.ByImpact[ChangeImpactCritical] != 1 {
		t.Errorf("Expected 1 critical impact change, got %d", summary.ByImpact[ChangeImpactCritical])
	}
}

func TestChangeReviewDialog_HelperMethods(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewChangeReviewDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})

	// Test change type names
	if dialog.getChangeTypeName(ReviewChangeTypeAdd) != "Add" {
		t.Error("ReviewChangeTypeAdd should return 'Add'")
	}
	if dialog.getChangeTypeName(ReviewChangeTypeRemove) != "Remove" {
		t.Error("ReviewChangeTypeRemove should return 'Remove'")
	}
	if dialog.getChangeTypeName(ReviewChangeTypeModify) != "Modify" {
		t.Error("ReviewChangeTypeModify should return 'Modify'")
	}

	// Test impact names
	if dialog.getImpactName(ChangeImpactLow) != "Low" {
		t.Error("ChangeImpactLow should return 'Low'")
	}
	if dialog.getImpactName(ChangeImpactMedium) != "Medium" {
		t.Error("ChangeImpactMedium should return 'Medium'")
	}
	if dialog.getImpactName(ChangeImpactHigh) != "High" {
		t.Error("ChangeImpactHigh should return 'High'")
	}
	if dialog.getImpactName(ChangeImpactCritical) != "Critical" {
		t.Error("ChangeImpactCritical should return 'Critical'")
	}

	// Test that icons are returned (not empty)
	if dialog.getChangeTypeIcon(ReviewChangeTypeAdd) == "" {
		t.Error("Should return non-empty icon for ReviewChangeTypeAdd")
	}
	if dialog.getImpactIcon(ChangeImpactCritical) == "" {
		t.Error("Should return non-empty icon for ChangeImpactCritical")
	}
}
