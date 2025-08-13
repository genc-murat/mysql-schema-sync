package display

import (
	"fmt"
	"io"
	"strings"
)

// Section represents a structured output section
type Section struct {
	Title       string
	Content     interface{}
	Subsections []*Section
	Collapsible bool
	Collapsed   bool
	Level       int
	Statistics  *SectionStatistics
}

// SectionStatistics holds summary statistics for a section
type SectionStatistics struct {
	ItemCount    int
	SuccessCount int
	WarningCount int
	ErrorCount   int
	TotalSize    int64
	CustomStats  map[string]interface{}
}

// SectionFormatter handles structured section-based output
type SectionFormatter struct {
	colorSystem ColorSystem
	iconSystem  IconSystem
	theme       ColorTheme
	writer      io.Writer
	maxWidth    int
	indentSize  int
}

// NewSectionFormatter creates a new section formatter
func NewSectionFormatter(colorSystem ColorSystem, iconSystem IconSystem, theme ColorTheme, writer io.Writer) *SectionFormatter {
	return &SectionFormatter{
		colorSystem: colorSystem,
		iconSystem:  iconSystem,
		theme:       theme,
		writer:      writer,
		maxWidth:    120,
		indentSize:  2,
	}
}

// SetMaxWidth sets the maximum width for section formatting
func (sf *SectionFormatter) SetMaxWidth(width int) {
	sf.maxWidth = width
}

// SetIndentSize sets the indentation size for nested sections
func (sf *SectionFormatter) SetIndentSize(size int) {
	sf.indentSize = size
}

// RenderSection renders a section with all its subsections
func (sf *SectionFormatter) RenderSection(section *Section) {
	sf.renderSectionRecursive(section, 0)
}

// RenderSections renders multiple sections
func (sf *SectionFormatter) RenderSections(sections []*Section) {
	for i, section := range sections {
		sf.renderSectionRecursive(section, 0)

		// Add spacing between top-level sections
		if i < len(sections)-1 {
			fmt.Fprintln(sf.writer)
		}
	}
}

// renderSectionRecursive renders a section and its subsections recursively
func (sf *SectionFormatter) renderSectionRecursive(section *Section, depth int) {
	indent := strings.Repeat(" ", depth*sf.indentSize)

	// Render section header
	sf.renderSectionHeader(section, indent, depth)

	// Skip content if collapsed
	if section.Collapsible && section.Collapsed {
		return
	}

	// Render statistics if available
	if section.Statistics != nil {
		sf.renderSectionStatistics(section.Statistics, indent+"  ")
	}

	// Render content
	if section.Content != nil {
		sf.renderSectionContent(section.Content, indent+"  ")
	}

	// Render subsections
	for _, subsection := range section.Subsections {
		sf.renderSectionRecursive(subsection, depth+1)
	}
}

// renderSectionHeader renders the section header with appropriate styling
func (sf *SectionFormatter) renderSectionHeader(section *Section, indent string, depth int) {
	var headerStyle string
	var separatorChar string
	var titleColor Color

	switch depth {
	case 0:
		// Top-level section
		separatorChar = "="
		titleColor = sf.theme.Primary
		headerStyle = "double"
	case 1:
		// Second-level section
		separatorChar = "-"
		titleColor = sf.theme.Highlight
		headerStyle = "single"
	default:
		// Deeper sections
		separatorChar = "·"
		titleColor = sf.theme.Info
		headerStyle = "minimal"
	}

	// Add collapse indicator for collapsible sections
	collapseIndicator := ""
	if section.Collapsible {
		if section.Collapsed {
			collapseIndicator = sf.iconSystem.RenderIcon("expand") + " "
		} else {
			collapseIndicator = sf.iconSystem.RenderIcon("collapse") + " "
		}
	}

	title := collapseIndicator + section.Title

	// Apply color if supported
	if sf.colorSystem.IsColorSupported() {
		title = sf.colorSystem.Colorize(title, titleColor)
	}

	switch headerStyle {
	case "double":
		// Double-line separator for top-level sections
		separatorLength := len(section.Title) + len(collapseIndicator) + 4
		if separatorLength > sf.maxWidth {
			separatorLength = sf.maxWidth
		}
		separator := strings.Repeat(separatorChar, separatorLength)

		fmt.Fprintf(sf.writer, "%s%s\n", indent, separator)
		fmt.Fprintf(sf.writer, "%s  %s  \n", indent, title)
		fmt.Fprintf(sf.writer, "%s%s\n", indent, separator)

	case "single":
		// Single-line separator for second-level sections
		fmt.Fprintf(sf.writer, "%s%s %s\n", indent, separatorChar+separatorChar+separatorChar, title)

	case "minimal":
		// Minimal formatting for deeper sections
		fmt.Fprintf(sf.writer, "%s%s %s\n", indent, separatorChar, title)
	}
}

// renderSectionStatistics renders section statistics with highlighting
func (sf *SectionFormatter) renderSectionStatistics(stats *SectionStatistics, indent string) {
	if stats.ItemCount == 0 && len(stats.CustomStats) == 0 {
		return
	}

	fmt.Fprintf(sf.writer, "%s", indent)

	// Render standard statistics
	statParts := []string{}

	if stats.ItemCount > 0 {
		itemText := fmt.Sprintf("Items: %d", stats.ItemCount)
		if sf.colorSystem.IsColorSupported() {
			itemText = sf.colorSystem.Colorize(itemText, sf.theme.Info)
		}
		statParts = append(statParts, itemText)
	}

	if stats.SuccessCount > 0 {
		successText := fmt.Sprintf("Success: %d", stats.SuccessCount)
		successIcon := sf.iconSystem.RenderIcon("success")
		if sf.colorSystem.IsColorSupported() {
			successText = sf.colorSystem.Colorize(fmt.Sprintf("%s %s", successIcon, successText), sf.theme.Success)
		} else {
			successText = fmt.Sprintf("%s %s", successIcon, successText)
		}
		statParts = append(statParts, successText)
	}

	if stats.WarningCount > 0 {
		warningText := fmt.Sprintf("Warnings: %d", stats.WarningCount)
		warningIcon := sf.iconSystem.RenderIcon("warning")
		if sf.colorSystem.IsColorSupported() {
			warningText = sf.colorSystem.Colorize(fmt.Sprintf("%s %s", warningIcon, warningText), sf.theme.Warning)
		} else {
			warningText = fmt.Sprintf("%s %s", warningIcon, warningText)
		}
		statParts = append(statParts, warningText)
	}

	if stats.ErrorCount > 0 {
		errorText := fmt.Sprintf("Errors: %d", stats.ErrorCount)
		errorIcon := sf.iconSystem.RenderIcon("error")
		if sf.colorSystem.IsColorSupported() {
			errorText = sf.colorSystem.Colorize(fmt.Sprintf("%s %s", errorIcon, errorText), sf.theme.Error)
		} else {
			errorText = fmt.Sprintf("%s %s", errorIcon, errorText)
		}
		statParts = append(statParts, errorText)
	}

	if stats.TotalSize > 0 {
		sizeText := fmt.Sprintf("Size: %s", formatBytes(stats.TotalSize))
		if sf.colorSystem.IsColorSupported() {
			sizeText = sf.colorSystem.Colorize(sizeText, sf.theme.Muted)
		}
		statParts = append(statParts, sizeText)
	}

	// Render custom statistics
	for key, value := range stats.CustomStats {
		customText := fmt.Sprintf("%s: %v", key, value)
		if sf.colorSystem.IsColorSupported() {
			customText = sf.colorSystem.Colorize(customText, sf.theme.Info)
		}
		statParts = append(statParts, customText)
	}

	if len(statParts) > 0 {
		fmt.Fprintf(sf.writer, "[%s]\n", strings.Join(statParts, " | "))
	}
}

// renderSectionContent renders the section content
func (sf *SectionFormatter) renderSectionContent(content interface{}, indent string) {
	switch v := content.(type) {
	case string:
		// Simple string content
		lines := strings.Split(v, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintf(sf.writer, "%s%s\n", indent, line)
			}
		}

	case []string:
		// List of strings
		for _, item := range v {
			bullet := sf.iconSystem.RenderIcon("bullet")
			fmt.Fprintf(sf.writer, "%s%s %s\n", indent, bullet, item)
		}

	case map[string]interface{}:
		// Key-value pairs
		for key, value := range v {
			keyText := key
			if sf.colorSystem.IsColorSupported() {
				keyText = sf.colorSystem.Colorize(key, sf.theme.Highlight)
			}
			fmt.Fprintf(sf.writer, "%s%s: %v\n", indent, keyText, value)
		}

	default:
		// Fallback to string representation
		fmt.Fprintf(sf.writer, "%s%v\n", indent, content)
	}
}

// NewSection creates a new section
func NewSection(title string) *Section {
	return &Section{
		Title:       title,
		Subsections: make([]*Section, 0),
		Level:       0,
	}
}

// AddSubsection adds a subsection to this section
func (s *Section) AddSubsection(subsection *Section) {
	subsection.Level = s.Level + 1
	s.Subsections = append(s.Subsections, subsection)
}

// SetContent sets the content for this section
func (s *Section) SetContent(content interface{}) {
	s.Content = content
}

// SetCollapsible makes this section collapsible
func (s *Section) SetCollapsible(collapsible bool) {
	s.Collapsible = collapsible
}

// SetCollapsed sets the collapsed state
func (s *Section) SetCollapsed(collapsed bool) {
	s.Collapsed = collapsed
}

// SetStatistics sets the statistics for this section
func (s *Section) SetStatistics(stats *SectionStatistics) {
	s.Statistics = stats
}

// NewSectionStatistics creates a new section statistics object
func NewSectionStatistics() *SectionStatistics {
	return &SectionStatistics{
		CustomStats: make(map[string]interface{}),
	}
}

// AddCustomStat adds a custom statistic
func (ss *SectionStatistics) AddCustomStat(key string, value interface{}) {
	ss.CustomStats[key] = value
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
