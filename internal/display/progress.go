package display

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// spinner implements SpinnerHandle interface
type spinner struct {
	id       string
	message  string
	style    SpinnerStyle
	active   bool
	writer   io.Writer
	colorSys ColorSystem
	theme    ColorTheme
	stopCh   chan struct{}
	doneCh   chan struct{}
	mu       sync.RWMutex
}

// ID returns the spinner's unique identifier
func (s *spinner) ID() string {
	return s.id
}

// IsActive returns whether the spinner is currently running
func (s *spinner) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// start begins the spinner animation
func (s *spinner) start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.animate()
}

// stop terminates the spinner animation
func (s *spinner) stop(finalMessage string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	close(s.stopCh)
	s.mu.Unlock()

	// Wait for animation to finish
	<-s.doneCh

	// Clear the spinner line and print final message
	s.clearLine()
	if finalMessage != "" {
		fmt.Fprintln(s.writer, finalMessage)
	}
}

// updateMessage changes the spinner's message
func (s *spinner) updateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// animate runs the spinner animation loop
func (s *spinner) animate() {
	defer close(s.doneCh)

	ticker := time.NewTicker(time.Duration(s.style.Delay) * time.Millisecond)
	defer ticker.Stop()

	frameIndex := 0

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.RLock()
			if !s.active {
				s.mu.RUnlock()
				return
			}

			frame := s.style.Frames[frameIndex%len(s.style.Frames)]
			message := s.message
			s.mu.RUnlock()

			// Format the spinner output
			output := s.formatSpinnerOutput(frame, message)

			// Clear line and print spinner
			s.clearLine()
			fmt.Fprint(s.writer, output)

			frameIndex++
		}
	}
}

// formatSpinnerOutput formats the spinner frame and message with colors
func (s *spinner) formatSpinnerOutput(frame, message string) string {
	if s.colorSys != nil && s.colorSys.IsColorSupported() {
		coloredFrame := s.colorSys.Colorize(frame, s.theme.Primary)
		return fmt.Sprintf("\r%s %s", coloredFrame, message)
	}
	return fmt.Sprintf("\r%s %s", frame, message)
}

// clearLine clears the current terminal line
func (s *spinner) clearLine() {
	fmt.Fprint(s.writer, "\r\033[K")
}

// ProgressBar represents a progress bar component
type ProgressBar struct {
	current     int
	total       int
	message     string
	width       int
	writer      io.Writer
	colorSys    ColorSystem
	theme       ColorTheme
	showPercent bool
	mu          sync.RWMutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string, writer io.Writer, colorSys ColorSystem, theme ColorTheme) *ProgressBar {
	return &ProgressBar{
		current:     0,
		total:       total,
		message:     message,
		width:       40, // Default width
		writer:      writer,
		colorSys:    colorSys,
		theme:       theme,
		showPercent: true,
	}
}

// Update updates the progress bar's current value
func (pb *ProgressBar) Update(current int, message string) {
	pb.mu.Lock()
	pb.current = current
	if message != "" {
		pb.message = message
	}
	pb.mu.Unlock()
	pb.render()
}

// Increment increments the progress bar by 1
func (pb *ProgressBar) Increment(message string) {
	pb.mu.Lock()
	pb.current++
	if message != "" {
		pb.message = message
	}
	pb.mu.Unlock()
	pb.render()
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish(finalMessage string) {
	pb.mu.Lock()
	pb.current = pb.total
	if finalMessage != "" {
		pb.message = finalMessage
	}
	pb.mu.Unlock()
	pb.render()
	fmt.Fprintln(pb.writer) // Add newline after completion
}

// render draws the progress bar
func (pb *ProgressBar) render() {
	pb.mu.RLock()
	current := pb.current
	total := pb.total
	message := pb.message
	pb.mu.RUnlock()

	if total <= 0 {
		return
	}

	// Calculate percentage
	percentage := float64(current) / float64(total) * 100
	if percentage > 100 {
		percentage = 100
	}

	// Calculate filled width
	filledWidth := int(float64(pb.width) * float64(current) / float64(total))
	if filledWidth > pb.width {
		filledWidth = pb.width
	}

	// Build progress bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", pb.width-filledWidth)

	var progressBar string
	if pb.colorSys != nil && pb.colorSys.IsColorSupported() {
		coloredFilled := pb.colorSys.Colorize(filled, pb.theme.Success)
		coloredEmpty := pb.colorSys.Colorize(empty, pb.theme.Muted)
		progressBar = fmt.Sprintf("[%s%s]", coloredFilled, coloredEmpty)
	} else {
		progressBar = fmt.Sprintf("[%s%s]", filled, empty)
	}

	// Format output
	var output string
	if pb.showPercent {
		output = fmt.Sprintf("\r%s %6.1f%% (%d/%d) %s",
			progressBar, percentage, current, total, message)
	} else {
		output = fmt.Sprintf("\r%s (%d/%d) %s",
			progressBar, current, total, message)
	}

	fmt.Fprint(pb.writer, output)
}

// SetWidth sets the width of the progress bar
func (pb *ProgressBar) SetWidth(width int) {
	pb.mu.Lock()
	pb.width = width
	pb.mu.Unlock()
}

// SetShowPercent enables or disables percentage display
func (pb *ProgressBar) SetShowPercent(show bool) {
	pb.mu.Lock()
	pb.showPercent = show
	pb.mu.Unlock()
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	bars   []*ProgressBar
	writer io.Writer
	active bool
	mu     sync.RWMutex
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress(writer io.Writer) *MultiProgress {
	return &MultiProgress{
		bars:   make([]*ProgressBar, 0),
		writer: writer,
		active: false,
	}
}

// AddBar adds a progress bar to the multi-progress
func (mp *MultiProgress) AddBar(bar *ProgressBar) {
	mp.mu.Lock()
	mp.bars = append(mp.bars, bar)
	mp.mu.Unlock()
}

// Start begins displaying all progress bars
func (mp *MultiProgress) Start() {
	mp.mu.Lock()
	mp.active = true
	mp.mu.Unlock()
}

// Stop stops displaying all progress bars
func (mp *MultiProgress) Stop() {
	mp.mu.Lock()
	mp.active = false
	mp.mu.Unlock()

	// Move cursor down past all bars
	fmt.Fprintf(mp.writer, "\n")
}

// Render renders all progress bars
func (mp *MultiProgress) Render() {
	mp.mu.RLock()
	if !mp.active {
		mp.mu.RUnlock()
		return
	}
	bars := make([]*ProgressBar, len(mp.bars))
	copy(bars, mp.bars)
	mp.mu.RUnlock()

	// Move cursor to beginning and clear lines
	for i := 0; i < len(bars); i++ {
		if i > 0 {
			fmt.Fprint(mp.writer, "\033[1A") // Move up one line
		}
		fmt.Fprint(mp.writer, "\r\033[K") // Clear line
	}

	// Render each bar
	for i, bar := range bars {
		if i > 0 {
			fmt.Fprintln(mp.writer)
		}
		bar.render()
	}
}

// spinnerManager manages multiple spinners
type spinnerManager struct {
	spinners map[string]*spinner
	counter  int
	mu       sync.RWMutex
}

// newSpinnerManager creates a new spinner manager
func newSpinnerManager() *spinnerManager {
	return &spinnerManager{
		spinners: make(map[string]*spinner),
		counter:  0,
	}
}

// createSpinner creates a new spinner with the given configuration
func (sm *spinnerManager) createSpinner(message string, style SpinnerStyle, writer io.Writer, colorSys ColorSystem, theme ColorTheme) *spinner {
	sm.mu.Lock()
	sm.counter++
	id := fmt.Sprintf("spinner_%d", sm.counter)
	sm.mu.Unlock()

	s := &spinner{
		id:       id,
		message:  message,
		style:    style,
		active:   false,
		writer:   writer,
		colorSys: colorSys,
		theme:    theme,
	}

	sm.mu.Lock()
	sm.spinners[id] = s
	sm.mu.Unlock()

	return s
}

// getSpinner retrieves a spinner by its handle
func (sm *spinnerManager) getSpinner(handle SpinnerHandle) *spinner {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if s, exists := sm.spinners[handle.ID()]; exists {
		return s
	}
	return nil
}

// removeSpinner removes a spinner from the manager
func (sm *spinnerManager) removeSpinner(handle SpinnerHandle) {
	sm.mu.Lock()
	delete(sm.spinners, handle.ID())
	sm.mu.Unlock()
}

// ProgressTracker manages progress across multiple phases
type ProgressTracker struct {
	phases       []ProgressPhase
	currentPhase int
	writer       io.Writer
	colorSys     ColorSystem
	theme        ColorTheme
	mu           sync.RWMutex
}

// ProgressPhase represents a phase in a multi-step operation
type ProgressPhase struct {
	Name      string
	Total     int
	Current   int
	Message   string
	Completed bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(phases []string, writer io.Writer, colorSys ColorSystem, theme ColorTheme) *ProgressTracker {
	progressPhases := make([]ProgressPhase, len(phases))
	for i, name := range phases {
		progressPhases[i] = ProgressPhase{
			Name:      name,
			Total:     0, // Will be set when phase starts
			Current:   0,
			Message:   "",
			Completed: false,
		}
	}

	return &ProgressTracker{
		phases:       progressPhases,
		currentPhase: 0,
		writer:       writer,
		colorSys:     colorSys,
		theme:        theme,
	}
}

// StartPhase starts a new phase with the given total steps
func (pt *ProgressTracker) StartPhase(phaseIndex int, total int, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if phaseIndex >= 0 && phaseIndex < len(pt.phases) {
		pt.currentPhase = phaseIndex
		pt.phases[phaseIndex].Total = total
		pt.phases[phaseIndex].Current = 0
		pt.phases[phaseIndex].Message = message
		pt.phases[phaseIndex].Completed = false
	}

	pt.render()
}

// UpdatePhase updates the current phase progress
func (pt *ProgressTracker) UpdatePhase(current int, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.currentPhase >= 0 && pt.currentPhase < len(pt.phases) {
		pt.phases[pt.currentPhase].Current = current
		if message != "" {
			pt.phases[pt.currentPhase].Message = message
		}
	}

	pt.render()
}

// CompletePhase marks the current phase as completed
func (pt *ProgressTracker) CompletePhase(finalMessage string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.currentPhase >= 0 && pt.currentPhase < len(pt.phases) {
		phase := &pt.phases[pt.currentPhase]
		phase.Current = phase.Total
		phase.Completed = true
		if finalMessage != "" {
			phase.Message = finalMessage
		}
	}

	pt.render()
}

// render displays the current progress state
func (pt *ProgressTracker) render() {
	// Clear previous output
	fmt.Fprint(pt.writer, "\r\033[K")

	var output strings.Builder

	// Show overall progress
	completed := 0
	for _, phase := range pt.phases {
		if phase.Completed {
			completed++
		}
	}

	overallPercent := float64(completed) / float64(len(pt.phases)) * 100

	if pt.colorSys != nil && pt.colorSys.IsColorSupported() {
		overallText := pt.colorSys.Sprintf(pt.theme.Info, "Overall: %.0f%% (%d/%d phases)",
			overallPercent, completed, len(pt.phases))
		output.WriteString(overallText)
	} else {
		output.WriteString(fmt.Sprintf("Overall: %.0f%% (%d/%d phases)",
			overallPercent, completed, len(pt.phases)))
	}

	// Show current phase details
	if pt.currentPhase >= 0 && pt.currentPhase < len(pt.phases) {
		phase := pt.phases[pt.currentPhase]

		var phasePercent float64
		if phase.Total > 0 {
			phasePercent = float64(phase.Current) / float64(phase.Total) * 100
		}

		phaseInfo := fmt.Sprintf(" | %s: %.0f%% (%d/%d) %s",
			phase.Name, phasePercent, phase.Current, phase.Total, phase.Message)

		if pt.colorSys != nil && pt.colorSys.IsColorSupported() {
			if phase.Completed {
				phaseInfo = pt.colorSys.Colorize(phaseInfo, pt.theme.Success)
			} else {
				phaseInfo = pt.colorSys.Colorize(phaseInfo, pt.theme.Primary)
			}
		}

		output.WriteString(phaseInfo)
	}

	fmt.Fprint(pt.writer, output.String())

	// Add newline if all phases completed
	if completed == len(pt.phases) {
		fmt.Fprintln(pt.writer)
	}
}

// GetPhaseCount returns the total number of phases
func (pt *ProgressTracker) GetPhaseCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return len(pt.phases)
}

// GetCurrentPhase returns the current phase index
func (pt *ProgressTracker) GetCurrentPhase() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.currentPhase
}

// IsCompleted returns true if all phases are completed
func (pt *ProgressTracker) IsCompleted() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	for _, phase := range pt.phases {
		if !phase.Completed {
			return false
		}
	}
	return true
}
