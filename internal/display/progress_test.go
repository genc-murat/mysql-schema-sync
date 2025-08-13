package display

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSpinner(t *testing.T) {
	tests := []struct {
		name    string
		message string
		style   SpinnerStyle
	}{
		{
			name:    "dots spinner",
			message: "Loading...",
			style:   DefaultSpinnerStyles["dots"],
		},
		{
			name:    "line spinner",
			message: "Processing...",
			style:   DefaultSpinnerStyles["line"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			colorSys := NewColorSystem(DefaultColorTheme())
			theme := DefaultColorTheme()

			sm := newSpinnerManager()
			spinner := sm.createSpinner(tt.message, tt.style, &buf, colorSys, theme)

			// Test spinner creation
			if spinner.ID() == "" {
				t.Error("Spinner ID should not be empty")
			}

			if spinner.IsActive() {
				t.Error("Spinner should not be active initially")
			}

			// Test starting spinner
			spinner.start()
			if !spinner.IsActive() {
				t.Error("Spinner should be active after start")
			}

			// Let it run briefly
			time.Sleep(200 * time.Millisecond)

			// Test updating message
			newMessage := "Updated message"
			spinner.updateMessage(newMessage)

			// Let it run a bit more
			time.Sleep(200 * time.Millisecond)

			// Test stopping spinner
			finalMessage := "Completed!"
			spinner.stop(finalMessage)

			if spinner.IsActive() {
				t.Error("Spinner should not be active after stop")
			}

			// Check that output was written
			output := buf.String()
			if output == "" {
				t.Error("Expected spinner output, got empty string")
			}

			// Check that final message is present
			if !strings.Contains(output, finalMessage) {
				t.Errorf("Expected final message '%s' in output, got: %s", finalMessage, output)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		updates []struct {
			current int
			message string
		}
	}{
		{
			name:  "basic progress",
			total: 10,
			updates: []struct {
				current int
				message string
			}{
				{2, "Step 1"},
				{5, "Step 2"},
				{8, "Step 3"},
				{10, "Complete"},
			},
		},
		{
			name:  "single step",
			total: 1,
			updates: []struct {
				current int
				message string
			}{
				{1, "Done"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			colorSys := NewColorSystem(DefaultColorTheme())
			theme := DefaultColorTheme()

			pb := NewProgressBar(tt.total, "Starting...", &buf, colorSys, theme)

			// Test initial state
			if pb.current != 0 {
				t.Errorf("Expected initial current to be 0, got %d", pb.current)
			}

			if pb.total != tt.total {
				t.Errorf("Expected total to be %d, got %d", tt.total, pb.total)
			}

			// Test updates
			for _, update := range tt.updates {
				pb.Update(update.current, update.message)

				// Check that current was updated
				pb.mu.RLock()
				current := pb.current
				message := pb.message
				pb.mu.RUnlock()

				if current != update.current {
					t.Errorf("Expected current to be %d, got %d", update.current, current)
				}

				if message != update.message {
					t.Errorf("Expected message to be '%s', got '%s'", update.message, message)
				}
			}

			// Test finish
			pb.Finish("All done!")

			// Check that output was written
			output := buf.String()
			if output == "" {
				t.Error("Expected progress bar output, got empty string")
			}

			// Check that progress indicators are present
			if !strings.Contains(output, "%") {
				t.Error("Expected percentage in output")
			}

			if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
				t.Error("Expected progress bar brackets in output")
			}
		})
	}
}

func TestProgressBarIncrement(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	pb := NewProgressBar(5, "Testing increment", &buf, colorSys, theme)

	// Test increment
	pb.Increment("Step 1")
	pb.mu.RLock()
	current := pb.current
	pb.mu.RUnlock()

	if current != 1 {
		t.Errorf("Expected current to be 1 after increment, got %d", current)
	}

	// Test multiple increments
	pb.Increment("Step 2")
	pb.Increment("Step 3")

	pb.mu.RLock()
	current = pb.current
	pb.mu.RUnlock()

	if current != 3 {
		t.Errorf("Expected current to be 3 after multiple increments, got %d", current)
	}
}

func TestProgressBarWidth(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	pb := NewProgressBar(10, "Testing width", &buf, colorSys, theme)

	// Test default width
	if pb.width != 40 {
		t.Errorf("Expected default width to be 40, got %d", pb.width)
	}

	// Test setting width
	pb.SetWidth(20)
	if pb.width != 20 {
		t.Errorf("Expected width to be 20 after SetWidth, got %d", pb.width)
	}

	// Test show percent
	pb.SetShowPercent(false)
	if pb.showPercent {
		t.Error("Expected showPercent to be false after SetShowPercent(false)")
	}
}

func TestMultiProgress(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	mp := NewMultiProgress(&buf)

	// Test initial state
	if mp.active {
		t.Error("MultiProgress should not be active initially")
	}

	if len(mp.bars) != 0 {
		t.Errorf("Expected 0 bars initially, got %d", len(mp.bars))
	}

	// Add progress bars
	pb1 := NewProgressBar(10, "Task 1", &buf, colorSys, theme)
	pb2 := NewProgressBar(5, "Task 2", &buf, colorSys, theme)

	mp.AddBar(pb1)
	mp.AddBar(pb2)

	if len(mp.bars) != 2 {
		t.Errorf("Expected 2 bars after adding, got %d", len(mp.bars))
	}

	// Test start/stop
	mp.Start()
	if !mp.active {
		t.Error("MultiProgress should be active after Start()")
	}

	mp.Stop()
	if mp.active {
		t.Error("MultiProgress should not be active after Stop()")
	}
}

func TestSpinnerManager(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	sm := newSpinnerManager()

	// Test creating spinners
	s1 := sm.createSpinner("Task 1", DefaultSpinnerStyles["dots"], &buf, colorSys, theme)
	s2 := sm.createSpinner("Task 2", DefaultSpinnerStyles["line"], &buf, colorSys, theme)

	if s1.ID() == s2.ID() {
		t.Error("Spinner IDs should be unique")
	}

	// Test getting spinners
	retrieved := sm.getSpinner(s1)
	if retrieved != s1 {
		t.Error("Should retrieve the same spinner instance")
	}

	// Test removing spinners
	sm.removeSpinner(s1)
	retrieved = sm.getSpinner(s1)
	if retrieved != nil {
		t.Error("Spinner should be nil after removal")
	}
}

func TestDisplayServiceProgressMethods(t *testing.T) {
	config := DefaultDisplayConfig()
	config.QuietMode = false

	var buf bytes.Buffer
	config.Writer = &buf

	ds := NewDisplayService(config)

	// Test StartSpinner
	handle := ds.StartSpinner("Testing spinner")
	if handle == nil {
		t.Error("StartSpinner should return a non-nil handle")
	}

	if !handle.IsActive() {
		t.Error("Spinner should be active after start")
	}

	// Test UpdateSpinner
	ds.UpdateSpinner(handle, "Updated message")

	// Test StopSpinner
	ds.StopSpinner(handle, "Spinner completed")

	if handle.IsActive() {
		t.Error("Spinner should not be active after stop")
	}

	// Test ShowProgress
	ds.ShowProgress(3, 10, "Progress test")
	ds.ShowProgress(10, 10, "Progress complete")

	// Check that output was written
	output := buf.String()
	if output == "" {
		t.Error("Expected output from progress methods")
	}
}

func TestDisplayServiceProgressQuietMode(t *testing.T) {
	config := DefaultDisplayConfig()
	config.QuietMode = true

	var buf bytes.Buffer
	config.Writer = &buf

	ds := NewDisplayService(config)

	// Test that spinner methods work in quiet mode
	handle := ds.StartSpinner("Testing spinner")
	ds.UpdateSpinner(handle, "Updated message")
	ds.StopSpinner(handle, "Spinner completed")

	// Test ShowProgress in quiet mode
	ds.ShowProgress(5, 10, "Progress test")

	// In quiet mode, only final messages should be shown
	output := buf.String()
	if !strings.Contains(output, "Spinner completed") {
		t.Error("Expected final spinner message in quiet mode")
	}
}

func TestDefaultSpinnerStyles(t *testing.T) {
	// Test that default styles are defined
	expectedStyles := []string{"dots", "line", "arrow"}

	for _, styleName := range expectedStyles {
		style, exists := DefaultSpinnerStyles[styleName]
		if !exists {
			t.Errorf("Expected default style '%s' to exist", styleName)
		}

		if len(style.Frames) == 0 {
			t.Errorf("Style '%s' should have frames", styleName)
		}

		if style.Delay <= 0 {
			t.Errorf("Style '%s' should have positive delay", styleName)
		}
	}
}

func TestProgressTracker(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	phases := []string{"Connect", "Extract", "Compare", "Apply"}
	pt := NewProgressTracker(phases, &buf, colorSys, theme)

	// Test initial state
	if pt.GetPhaseCount() != 4 {
		t.Errorf("Expected 4 phases, got %d", pt.GetPhaseCount())
	}

	if pt.GetCurrentPhase() != 0 {
		t.Errorf("Expected current phase to be 0, got %d", pt.GetCurrentPhase())
	}

	if pt.IsCompleted() {
		t.Error("Progress tracker should not be completed initially")
	}

	// Test starting first phase
	pt.StartPhase(0, 10, "Connecting to database")
	if pt.GetCurrentPhase() != 0 {
		t.Errorf("Expected current phase to be 0 after StartPhase, got %d", pt.GetCurrentPhase())
	}

	// Test updating phase
	pt.UpdatePhase(5, "Connection established")
	pt.UpdatePhase(10, "Connection complete")

	// Test completing phase
	pt.CompletePhase("Connected successfully")

	// Test starting next phase
	pt.StartPhase(1, 20, "Extracting schema")
	pt.UpdatePhase(10, "Extracting tables")
	pt.CompletePhase("Schema extracted")

	// Complete remaining phases
	pt.StartPhase(2, 5, "Comparing schemas")
	pt.CompletePhase("Comparison complete")

	pt.StartPhase(3, 15, "Applying changes")
	pt.CompletePhase("Changes applied")

	// Test completion
	if !pt.IsCompleted() {
		t.Error("Progress tracker should be completed after all phases")
	}

	// Check that output was written
	output := buf.String()
	if output == "" {
		t.Error("Expected progress tracker output")
	}

	// Check for phase names in output
	for _, phase := range phases {
		if !strings.Contains(output, phase) {
			t.Errorf("Expected phase '%s' in output", phase)
		}
	}
}

func TestProgressTrackerEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	colorSys := NewColorSystem(DefaultColorTheme())
	theme := DefaultColorTheme()

	// Test with empty phases
	pt := NewProgressTracker([]string{}, &buf, colorSys, theme)
	if pt.GetPhaseCount() != 0 {
		t.Errorf("Expected 0 phases for empty input, got %d", pt.GetPhaseCount())
	}

	if !pt.IsCompleted() {
		t.Error("Empty progress tracker should be completed")
	}

	// Test with single phase
	pt = NewProgressTracker([]string{"Single"}, &buf, colorSys, theme)
	pt.StartPhase(0, 1, "Single task")
	pt.CompletePhase("Done")

	if !pt.IsCompleted() {
		t.Error("Single phase tracker should be completed")
	}

	// Test invalid phase index
	pt = NewProgressTracker([]string{"Phase1", "Phase2"}, &buf, colorSys, theme)
	pt.StartPhase(5, 10, "Invalid phase") // Should not crash
	pt.UpdatePhase(5, "Invalid update")   // Should not crash
	pt.CompletePhase("Invalid complete")  // Should not crash
}

func TestDisplayServiceProgressBars(t *testing.T) {
	config := DefaultDisplayConfig()
	config.QuietMode = false

	var buf bytes.Buffer
	config.Writer = &buf

	ds := NewDisplayService(config)

	// Test NewProgressBar
	pb := ds.NewProgressBar(10, "Testing progress bar")
	if pb == nil {
		t.Error("NewProgressBar should return a non-nil progress bar")
	}

	pb.Update(5, "Half way")
	pb.Finish("Complete")

	// Test NewMultiProgress
	mp := ds.NewMultiProgress()
	if mp == nil {
		t.Error("NewMultiProgress should return a non-nil multi-progress")
	}

	pb1 := ds.NewProgressBar(5, "Task 1")
	pb2 := ds.NewProgressBar(3, "Task 2")

	mp.AddBar(pb1)
	mp.AddBar(pb2)
	mp.Start()
	mp.Stop()

	// Test NewProgressTracker
	pt := ds.NewProgressTracker([]string{"Phase1", "Phase2"})
	if pt == nil {
		t.Error("NewProgressTracker should return a non-nil progress tracker")
	}

	pt.StartPhase(0, 5, "Starting phase 1")
	pt.UpdatePhase(3, "Progress update")
	pt.CompletePhase("Phase 1 complete")

	// Check that output was written
	output := buf.String()
	if output == "" {
		t.Error("Expected output from progress bar methods")
	}
}

// Comprehensive tests for progress components with different terminal capabilities
func TestProgressBarComprehensive(t *testing.T) {
	t.Run("ZeroTotal", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pb := NewProgressBar(0, "Zero total", &buf, colorSys, theme)
		pb.Update(5, "Should handle gracefully")
		pb.render() // Should not crash

		// Should not render anything for zero total
		output := buf.String()
		if output != "" {
			t.Error("Progress bar with zero total should not render")
		}
	})

	t.Run("NegativeTotal", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pb := NewProgressBar(-5, "Negative total", &buf, colorSys, theme)
		pb.Update(1, "Should handle gracefully")
		pb.render() // Should not crash
	})

	t.Run("ExcessiveProgress", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pb := NewProgressBar(10, "Test excessive", &buf, colorSys, theme)
		pb.Update(15, "Over 100%") // More than total

		// Should cap at 100%
		pb.mu.RLock()
		current := pb.current
		pb.mu.RUnlock()

		if current != 15 {
			t.Error("Current should be set to actual value")
		}

		pb.render()
		output := buf.String()
		if !strings.Contains(output, "100.0%") {
			t.Error("Should cap percentage at 100%")
		}
	})

	t.Run("WidthSettings", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pb := NewProgressBar(10, "Width test", &buf, colorSys, theme)

		// Test different widths
		widths := []int{10, 20, 50, 100}
		for _, width := range widths {
			pb.SetWidth(width)
			if pb.width != width {
				t.Errorf("Width should be set to %d, got %d", width, pb.width)
			}

			pb.Update(5, "Testing width")
			// Should not crash with different widths
		}
	})

	t.Run("PercentageDisplay", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pb := NewProgressBar(10, "Percentage test", &buf, colorSys, theme)

		// Test with percentage enabled
		pb.SetShowPercent(true)
		pb.Update(3, "30%")
		pb.render()

		output := buf.String()
		if !strings.Contains(output, "30.0%") {
			t.Error("Should show percentage when enabled")
		}

		buf.Reset()

		// Test with percentage disabled
		pb.SetShowPercent(false)
		pb.Update(5, "50%")
		pb.render()

		output = buf.String()
		if strings.Contains(output, "50.0%") {
			t.Error("Should not show percentage when disabled")
		}
	})

	t.Run("ColorSupport", func(t *testing.T) {
		var buf bytes.Buffer

		// Test with color support
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()
		pb := NewProgressBar(10, "Color test", &buf, colorSys, theme)
		pb.Update(5, "Half")
		pb.render()

		colorOutput := buf.String()
		buf.Reset()

		// Test without color support (plain theme)
		plainTheme := PlainTextTheme()
		plainColorSys := NewColorSystem(plainTheme)
		pb2 := NewProgressBar(10, "Plain test", &buf, plainColorSys, plainTheme)
		pb2.Update(5, "Half")
		pb2.render()

		plainOutput := buf.String()

		// Both should produce output
		if colorOutput == "" || plainOutput == "" {
			t.Error("Both color and plain progress bars should produce output")
		}
	})
}

func TestSpinnerComprehensive(t *testing.T) {
	t.Run("SpinnerStyles", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		sm := newSpinnerManager()

		// Test all default spinner styles
		for styleName, style := range DefaultSpinnerStyles {
			t.Run(styleName, func(t *testing.T) {
				spinner := sm.createSpinner("Testing "+styleName, style, &buf, colorSys, theme)

				if spinner.ID() == "" {
					t.Error("Spinner should have non-empty ID")
				}

				spinner.start()
				if !spinner.IsActive() {
					t.Error("Spinner should be active after start")
				}

				time.Sleep(50 * time.Millisecond)

				spinner.updateMessage("Updated message")
				time.Sleep(50 * time.Millisecond)

				spinner.stop("Done")
				if spinner.IsActive() {
					t.Error("Spinner should not be active after stop")
				}
			})
		}
	})

	t.Run("ConcurrentSpinners", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		sm := newSpinnerManager()

		// Create multiple spinners
		spinners := make([]*spinner, 3)
		for i := 0; i < 3; i++ {
			spinners[i] = sm.createSpinner(
				fmt.Sprintf("Spinner %d", i+1),
				DefaultSpinnerStyles["dots"],
				&buf, colorSys, theme,
			)
		}

		// Start all spinners
		for _, s := range spinners {
			s.start()
		}

		// Let them run
		time.Sleep(100 * time.Millisecond)

		// Stop all spinners
		for i, s := range spinners {
			s.stop(fmt.Sprintf("Spinner %d done", i+1))
		}

		// Verify all are stopped
		for i, s := range spinners {
			if s.IsActive() {
				t.Errorf("Spinner %d should be stopped", i+1)
			}
		}
	})

	t.Run("SpinnerManagerOperations", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		sm := newSpinnerManager()

		// Create spinner
		s1 := sm.createSpinner("Test 1", DefaultSpinnerStyles["dots"], &buf, colorSys, theme)
		s2 := sm.createSpinner("Test 2", DefaultSpinnerStyles["line"], &buf, colorSys, theme)

		// Test unique IDs
		if s1.ID() == s2.ID() {
			t.Error("Spinners should have unique IDs")
		}

		// Test retrieval
		retrieved := sm.getSpinner(s1)
		if retrieved != s1 {
			t.Error("Should retrieve the same spinner instance")
		}

		// Test removal
		sm.removeSpinner(s1)
		retrieved = sm.getSpinner(s1)
		if retrieved != nil {
			t.Error("Spinner should be nil after removal")
		}

		// Test getting non-existent spinner
		nonExistent := &noOpSpinner{}
		retrieved = sm.getSpinner(nonExistent)
		if retrieved != nil {
			t.Error("Non-existent spinner should return nil")
		}
	})
}

func TestMultiProgressComprehensive(t *testing.T) {
	t.Run("EmptyMultiProgress", func(t *testing.T) {
		var buf bytes.Buffer
		mp := NewMultiProgress(&buf)

		if mp.active {
			t.Error("MultiProgress should not be active initially")
		}

		mp.Start()
		if !mp.active {
			t.Error("MultiProgress should be active after Start")
		}

		mp.Render() // Should not crash with no bars

		mp.Stop()
		if mp.active {
			t.Error("MultiProgress should not be active after Stop")
		}
	})

	t.Run("MultipleProgressBars", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		mp := NewMultiProgress(&buf)

		// Add multiple progress bars
		bars := make([]*ProgressBar, 3)
		for i := 0; i < 3; i++ {
			bars[i] = NewProgressBar(10, fmt.Sprintf("Task %d", i+1), &buf, colorSys, theme)
			mp.AddBar(bars[i])
		}

		mp.Start()

		// Update progress bars
		for i, bar := range bars {
			bar.Update(i+1, fmt.Sprintf("Progress %d", i+1))
		}

		mp.Render()
		mp.Stop()

		// Check that bars were added
		mp.mu.RLock()
		barCount := len(mp.bars)
		mp.mu.RUnlock()

		if barCount != 3 {
			t.Errorf("Expected 3 bars, got %d", barCount)
		}
	})
}

func TestProgressTrackerComprehensive(t *testing.T) {
	t.Run("EmptyPhases", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pt := NewProgressTracker([]string{}, &buf, colorSys, theme)

		if pt.GetPhaseCount() != 0 {
			t.Error("Empty progress tracker should have 0 phases")
		}

		if !pt.IsCompleted() {
			t.Error("Empty progress tracker should be completed")
		}

		// These should not crash
		pt.StartPhase(0, 10, "Invalid")
		pt.UpdatePhase(5, "Invalid")
		pt.CompletePhase("Invalid")
	})

	t.Run("SinglePhase", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pt := NewProgressTracker([]string{"OnlyPhase"}, &buf, colorSys, theme)

		if pt.GetPhaseCount() != 1 {
			t.Error("Should have 1 phase")
		}

		if pt.IsCompleted() {
			t.Error("Should not be completed initially")
		}

		pt.StartPhase(0, 5, "Starting")
		if pt.GetCurrentPhase() != 0 {
			t.Error("Current phase should be 0")
		}

		pt.UpdatePhase(3, "Progress")
		pt.CompletePhase("Done")

		if !pt.IsCompleted() {
			t.Error("Should be completed after completing single phase")
		}
	})

	t.Run("MultiplePhases", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		phases := []string{"Connect", "Extract", "Compare", "Apply"}
		pt := NewProgressTracker(phases, &buf, colorSys, theme)

		if pt.GetPhaseCount() != 4 {
			t.Error("Should have 4 phases")
		}

		// Complete all phases
		for i, phase := range phases {
			pt.StartPhase(i, 10, fmt.Sprintf("Starting %s", phase))

			if pt.GetCurrentPhase() != i {
				t.Errorf("Current phase should be %d, got %d", i, pt.GetCurrentPhase())
			}

			// Update progress
			for j := 1; j <= 10; j++ {
				pt.UpdatePhase(j, fmt.Sprintf("%s progress %d", phase, j))
			}

			pt.CompletePhase(fmt.Sprintf("%s completed", phase))
		}

		if !pt.IsCompleted() {
			t.Error("Should be completed after all phases")
		}
	})

	t.Run("InvalidPhaseOperations", func(t *testing.T) {
		var buf bytes.Buffer
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()

		pt := NewProgressTracker([]string{"Phase1", "Phase2"}, &buf, colorSys, theme)

		// Test invalid phase indices - should not crash
		pt.StartPhase(-1, 10, "Invalid negative")
		pt.StartPhase(10, 10, "Invalid high")

		pt.UpdatePhase(5, "Invalid update")
		pt.CompletePhase("Invalid complete")

		// Should still work with valid operations
		pt.StartPhase(0, 5, "Valid")
		pt.UpdatePhase(3, "Valid update")
		pt.CompletePhase("Valid complete")
	})

	t.Run("ColorSupport", func(t *testing.T) {
		var buf bytes.Buffer

		// Test with colors
		colorSys := NewColorSystem(DefaultColorTheme())
		theme := DefaultColorTheme()
		pt := NewProgressTracker([]string{"Phase1"}, &buf, colorSys, theme)

		pt.StartPhase(0, 5, "With colors")
		pt.UpdatePhase(3, "Progress")
		pt.CompletePhase("Done")

		colorOutput := buf.String()
		buf.Reset()

		// Test without colors
		plainTheme := PlainTextTheme()
		plainColorSys := NewColorSystem(plainTheme)
		pt2 := NewProgressTracker([]string{"Phase1"}, &buf, plainColorSys, plainTheme)

		pt2.StartPhase(0, 5, "Without colors")
		pt2.UpdatePhase(3, "Progress")
		pt2.CompletePhase("Done")

		plainOutput := buf.String()

		// Both should produce output
		if colorOutput == "" || plainOutput == "" {
			t.Error("Both color and plain progress trackers should produce output")
		}
	})
}
