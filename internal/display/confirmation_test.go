package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmationDialog_Basic(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetTitle("Test Confirmation")
	dialog.SetMessage("Do you want to proceed?")
	dialog.AddOption("y", "yes", "Proceed with operation", false)
	dialog.AddOption("n", "no", "Cancel operation", true)

	// Test rendering without actually showing (which would wait for input)
	dialog.render()

	output_str := output.String()

	// Check that title is rendered
	if !strings.Contains(output_str, "Test Confirmation") {
		t.Error("Title not found in output")
	}

	// Check that message is rendered
	if !strings.Contains(output_str, "Do you want to proceed?") {
		t.Error("Message not found in output")
	}

	// Check that options are rendered
	if !strings.Contains(output_str, "yes") {
		t.Error("Yes option not found in output")
	}

	if !strings.Contains(output_str, "no") {
		t.Error("No option not found in output")
	}
}

func TestConfirmationDialog_Destructive(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetTitle("Destructive Operation")
	dialog.SetMessage("This will delete data.")
	dialog.SetDestructive(true)
	dialog.AddOption("y", "yes", "Proceed anyway", false)
	dialog.AddOption("n", "no", "Cancel", true)

	dialog.render()

	output_str := output.String()

	// Check that destructive warning is rendered
	if !strings.Contains(output_str, "DESTRUCTIVE OPERATION") {
		t.Error("Destructive warning not found in output")
	}

	if !strings.Contains(output_str, "data loss") {
		t.Error("Data loss warning not found in output")
	}
}

func TestConfirmationDialog_WithWarning(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetMessage("Proceed with operation?")
	dialog.SetWarning("This operation cannot be undone")
	dialog.AddOption("y", "yes", "Proceed", false)
	dialog.AddOption("n", "no", "Cancel", true)

	dialog.render()

	output_str := output.String()

	// Check that warning is rendered
	if !strings.Contains(output_str, "This operation cannot be undone") {
		t.Error("Warning message not found in output")
	}
}

func TestConfirmationDialog_WithDetails(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetMessage("Apply changes?")
	dialog.AddDetails("Change 1: Add table users", "Change 2: Drop column old_field")
	dialog.AddOption("y", "yes", "Apply", false)
	dialog.AddOption("n", "no", "Cancel", true)

	dialog.render()

	output_str := output.String()

	// Check that details option is available
	if !strings.Contains(output_str, "details") {
		t.Error("Details option not found in output")
	}

	// Test showing details
	output.Reset()
	dialog.showDetails()

	details_output := output.String()
	if !strings.Contains(details_output, "Change 1: Add table users") {
		t.Error("Detail 1 not found in details output")
	}

	if !strings.Contains(details_output, "Change 2: Drop column old_field") {
		t.Error("Detail 2 not found in details output")
	}
}

func TestConfirmationDialog_ParseInput(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &bytes.Buffer{})
	dialog.AddOption("y", "yes", "Proceed", false)
	dialog.AddOption("n", "no", "Cancel", true)

	// Test yes input
	result := dialog.parseInput("y")
	if result == nil || !result.Confirmed || result.SelectedKey != "y" {
		t.Error("Failed to parse 'y' input correctly")
	}

	// Test no input
	result = dialog.parseInput("n")
	if result == nil || result.Confirmed || result.SelectedKey != "n" || !result.Cancelled {
		t.Error("Failed to parse 'n' input correctly")
	}

	// Test empty input (should use default)
	result = dialog.parseInput("")
	if result == nil || result.Confirmed || result.SelectedKey != "n" {
		t.Error("Failed to use default option for empty input")
	}

	// Test invalid input
	result = dialog.parseInput("invalid")
	if result != nil {
		t.Error("Should return nil for invalid input")
	}
}

func TestConfirmationBuilder(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	builder := NewConfirmationBuilder(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog := builder.
		Title("Test Builder").
		Message("Builder test message").
		YesNo().
		Destructive().
		Warning("Test warning").
		Details("Detail 1", "Detail 2").
		Build()

	dialog.render()

	output_str := output.String()

	// Check all components are present
	if !strings.Contains(output_str, "Test Builder") {
		t.Error("Builder title not found")
	}

	if !strings.Contains(output_str, "Builder test message") {
		t.Error("Builder message not found")
	}

	if !strings.Contains(output_str, "DESTRUCTIVE OPERATION") {
		t.Error("Builder destructive warning not found")
	}

	if !strings.Contains(output_str, "Test warning") {
		t.Error("Builder warning not found")
	}

	if !strings.Contains(output_str, "details") {
		t.Error("Builder details option not found")
	}
}

func TestConfirmationBuilder_YesNoDefault(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	builder := NewConfirmationBuilder(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog := builder.
		Message("Test with yes default").
		YesNoDefault().
		Build()

	// Test that yes is default
	result := dialog.parseInput("")
	if result == nil || !result.Confirmed || result.SelectedKey != "y" {
		t.Error("YesNoDefault should make 'y' the default option")
	}
}

func TestConfirmationBuilder_CustomOptions(t *testing.T) {
	var output bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	dialog := NewConfirmationDialog(colorSystem, iconSystem, DefaultColorTheme(), &output)
	dialog.SetMessage("Custom options test")
	dialog.AddOption("a", "apply", "Apply changes", false)
	dialog.AddCancelOption("s", "skip", "Skip changes", true)
	dialog.AddOption("q", "quit", "Quit application", false)

	dialog.render()

	output_str := output.String()

	// Check custom options are present
	if !strings.Contains(output_str, "apply") {
		t.Error("Custom option 'apply' not found")
	}

	if !strings.Contains(output_str, "skip") {
		t.Error("Custom option 'skip' not found")
	}

	if !strings.Contains(output_str, "quit") {
		t.Error("Custom option 'quit' not found")
	}

	// Test parsing custom options
	result := dialog.parseInput("a")
	if result == nil || !result.Confirmed || result.SelectedKey != "a" {
		t.Error("Failed to parse custom option 'a'")
	}

	result = dialog.parseInput("s")
	if result == nil || result.Confirmed || result.SelectedKey != "s" {
		t.Error("Failed to parse custom option 's'")
	}
}
