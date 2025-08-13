package display

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// TableFormatter interface for creating formatted tables
type TableFormatter interface {
	SetHeaders(headers []string)
	AddRow(row []string)
	AddSeparator()
	SetColumnAlignment(column int, alignment Alignment)
	SetColumnWidth(column int, width int)
	SetStyle(style TableStyle)
	Render() string
	RenderTo(writer io.Writer)
}

// Alignment represents column alignment options
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// TableStyle defines the visual style of a table
type TableStyle struct {
	Name            string
	BorderStyle     BorderStyle
	HeaderSeparator bool
	RowSeparator    bool
	Padding         int
	MaxWidth        int
	Responsive      bool
}

// BorderStyle defines table border characters
type BorderStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	Cross       string
	TopTee      string
	BottomTee   string
	LeftTee     string
	RightTee    string
}

// Predefined table styles
var (
	// DefaultTableStyle is a simple ASCII table style
	DefaultTableStyle = TableStyle{
		Name:            "default",
		BorderStyle:     ASCIIBorderStyle,
		HeaderSeparator: true,
		RowSeparator:    false,
		Padding:         1,
		MaxWidth:        0, // Auto-detect
		Responsive:      true,
	}

	// RoundedTableStyle uses Unicode box drawing characters
	RoundedTableStyle = TableStyle{
		Name:            "rounded",
		BorderStyle:     RoundedBorderStyle,
		HeaderSeparator: true,
		RowSeparator:    false,
		Padding:         1,
		MaxWidth:        0, // Auto-detect
		Responsive:      true,
	}

	// CompactTableStyle is minimal with no borders
	CompactTableStyle = TableStyle{
		Name:            "compact",
		BorderStyle:     NoBorderStyle,
		HeaderSeparator: false,
		RowSeparator:    false,
		Padding:         1,
		MaxWidth:        0, // Auto-detect
		Responsive:      true,
	}

	// GridTableStyle has borders around all cells
	GridTableStyle = TableStyle{
		Name:            "grid",
		BorderStyle:     ASCIIBorderStyle,
		HeaderSeparator: true,
		RowSeparator:    true,
		Padding:         1,
		MaxWidth:        0, // Auto-detect
		Responsive:      true,
	}
)

// Border styles
var (
	ASCIIBorderStyle = BorderStyle{
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
		Horizontal:  "-",
		Vertical:    "|",
		Cross:       "+",
		TopTee:      "+",
		BottomTee:   "+",
		LeftTee:     "+",
		RightTee:    "+",
	}

	RoundedBorderStyle = BorderStyle{
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
		Horizontal:  "─",
		Vertical:    "│",
		Cross:       "┼",
		TopTee:      "┬",
		BottomTee:   "┴",
		LeftTee:     "├",
		RightTee:    "┤",
	}

	NoBorderStyle = BorderStyle{
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "",
		BottomRight: "",
		Horizontal:  "",
		Vertical:    "",
		Cross:       "",
		TopTee:      "",
		BottomTee:   "",
		LeftTee:     "",
		RightTee:    "",
	}
)

// tableFormatter implements the TableFormatter interface
type tableFormatter struct {
	headers       []string
	rows          [][]string
	separators    []int // Row indices where separators should be added
	alignments    map[int]Alignment
	columnWidths  map[int]int
	style         TableStyle
	colorSystem   ColorSystem
	theme         ColorTheme
	terminalWidth int
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(colorSystem ColorSystem, theme ColorTheme) TableFormatter {
	termWidth := getTerminalWidth()

	return &tableFormatter{
		headers:       make([]string, 0),
		rows:          make([][]string, 0),
		separators:    make([]int, 0),
		alignments:    make(map[int]Alignment),
		columnWidths:  make(map[int]int),
		style:         DefaultTableStyle,
		colorSystem:   colorSystem,
		theme:         theme,
		terminalWidth: termWidth,
	}
}

// SetHeaders sets the table headers
func (tf *tableFormatter) SetHeaders(headers []string) {
	tf.headers = headers
}

// AddRow adds a row to the table
func (tf *tableFormatter) AddRow(row []string) {
	tf.rows = append(tf.rows, row)
}

// AddSeparator adds a separator after the current row
func (tf *tableFormatter) AddSeparator() {
	tf.separators = append(tf.separators, len(tf.rows))
}

// SetColumnAlignment sets the alignment for a specific column
func (tf *tableFormatter) SetColumnAlignment(column int, alignment Alignment) {
	tf.alignments[column] = alignment
}

// SetColumnWidth sets the width for a specific column
func (tf *tableFormatter) SetColumnWidth(column int, width int) {
	tf.columnWidths[column] = width
}

// SetStyle sets the table style
func (tf *tableFormatter) SetStyle(style TableStyle) {
	tf.style = style
}

// Render returns the formatted table as a string
func (tf *tableFormatter) Render() string {
	if len(tf.headers) == 0 && len(tf.rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := tf.calculateColumnWidths()

	// Apply responsive layout if needed
	if tf.style.Responsive && tf.style.MaxWidth > 0 {
		colWidths = tf.adjustForMaxWidth(colWidths)
	}

	var result strings.Builder

	// Render top border
	if tf.style.BorderStyle.Horizontal != "" {
		result.WriteString(tf.renderTopBorder(colWidths))
		result.WriteString("\n")
	}

	// Render headers
	if len(tf.headers) > 0 {
		result.WriteString(tf.renderRow(tf.headers, colWidths, true))
		result.WriteString("\n")

		// Render header separator
		if tf.style.HeaderSeparator && tf.style.BorderStyle.Horizontal != "" {
			result.WriteString(tf.renderMiddleBorder(colWidths))
			result.WriteString("\n")
		}
	}

	// Render rows
	for i, row := range tf.rows {
		result.WriteString(tf.renderRow(row, colWidths, false))
		result.WriteString("\n")

		// Check if separator should be added after this row
		if tf.style.RowSeparator && i < len(tf.rows)-1 {
			result.WriteString(tf.renderMiddleBorder(colWidths))
			result.WriteString("\n")
		}

		// Check for manual separators
		for _, sepIndex := range tf.separators {
			if sepIndex == i+1 && i < len(tf.rows)-1 {
				result.WriteString(tf.renderMiddleBorder(colWidths))
				result.WriteString("\n")
				break
			}
		}
	}

	// Render bottom border
	if tf.style.BorderStyle.Horizontal != "" {
		result.WriteString(tf.renderBottomBorder(colWidths))
		result.WriteString("\n")
	}

	return result.String()
}

// RenderTo renders the table to the specified writer
func (tf *tableFormatter) RenderTo(writer io.Writer) {
	fmt.Fprint(writer, tf.Render())
}

// calculateColumnWidths calculates the optimal width for each column
func (tf *tableFormatter) calculateColumnWidths() []int {
	numCols := tf.getColumnCount()
	if numCols == 0 {
		return []int{}
	}

	widths := make([]int, numCols)

	// Initialize with header widths
	for i, header := range tf.headers {
		if i < numCols {
			widths[i] = utf8.RuneCountInString(header)
		}
	}

	// Check row widths
	for _, row := range tf.rows {
		for i, cell := range row {
			if i < numCols {
				cellWidth := utf8.RuneCountInString(cell)
				if cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			}
		}
	}

	// Apply manual column widths
	for col, width := range tf.columnWidths {
		if col < numCols && width > 0 {
			widths[col] = width
		}
	}

	// Add padding
	for i := range widths {
		widths[i] += tf.style.Padding * 2
	}

	return widths
}

// adjustForMaxWidth adjusts column widths to fit within maximum width
func (tf *tableFormatter) adjustForMaxWidth(widths []int) []int {
	maxWidth := tf.style.MaxWidth
	if maxWidth == 0 {
		maxWidth = tf.terminalWidth
	}

	if maxWidth <= 0 {
		return widths
	}

	// Calculate total width including borders
	totalWidth := tf.calculateTotalWidth(widths)

	if totalWidth <= maxWidth {
		return widths
	}

	// Need to reduce column widths
	// Simple strategy: reduce all columns proportionally
	reduction := float64(totalWidth-maxWidth) / float64(len(widths))

	for i := range widths {
		newWidth := int(float64(widths[i]) - reduction)
		if newWidth < tf.style.Padding*2+3 { // Minimum width
			newWidth = tf.style.Padding*2 + 3
		}
		widths[i] = newWidth
	}

	return widths
}

// calculateTotalWidth calculates the total width including borders
func (tf *tableFormatter) calculateTotalWidth(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}

	// Add border characters
	if tf.style.BorderStyle.Vertical != "" {
		total += len(widths) + 1 // One border per column plus one at the end
	}

	return total
}

// getColumnCount returns the maximum number of columns
func (tf *tableFormatter) getColumnCount() int {
	maxCols := len(tf.headers)

	for _, row := range tf.rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	return maxCols
}

// renderTopBorder renders the top border of the table
func (tf *tableFormatter) renderTopBorder(widths []int) string {
	if tf.style.BorderStyle.Horizontal == "" {
		return ""
	}

	var result strings.Builder

	result.WriteString(tf.style.BorderStyle.TopLeft)

	for i, width := range widths {
		result.WriteString(strings.Repeat(tf.style.BorderStyle.Horizontal, width))
		if i < len(widths)-1 {
			result.WriteString(tf.style.BorderStyle.TopTee)
		}
	}

	result.WriteString(tf.style.BorderStyle.TopRight)

	return result.String()
}

// renderBottomBorder renders the bottom border of the table
func (tf *tableFormatter) renderBottomBorder(widths []int) string {
	if tf.style.BorderStyle.Horizontal == "" {
		return ""
	}

	var result strings.Builder

	result.WriteString(tf.style.BorderStyle.BottomLeft)

	for i, width := range widths {
		result.WriteString(strings.Repeat(tf.style.BorderStyle.Horizontal, width))
		if i < len(widths)-1 {
			result.WriteString(tf.style.BorderStyle.BottomTee)
		}
	}

	result.WriteString(tf.style.BorderStyle.BottomRight)

	return result.String()
}

// renderMiddleBorder renders a middle border (separator) of the table
func (tf *tableFormatter) renderMiddleBorder(widths []int) string {
	if tf.style.BorderStyle.Horizontal == "" {
		return ""
	}

	var result strings.Builder

	result.WriteString(tf.style.BorderStyle.LeftTee)

	for i, width := range widths {
		result.WriteString(strings.Repeat(tf.style.BorderStyle.Horizontal, width))
		if i < len(widths)-1 {
			result.WriteString(tf.style.BorderStyle.Cross)
		}
	}

	result.WriteString(tf.style.BorderStyle.RightTee)

	return result.String()
}

// renderRow renders a single row of the table
func (tf *tableFormatter) renderRow(row []string, widths []int, isHeader bool) string {
	var result strings.Builder

	// Left border
	if tf.style.BorderStyle.Vertical != "" {
		result.WriteString(tf.style.BorderStyle.Vertical)
	}

	for i, width := range widths {
		var cell string
		if i < len(row) {
			cell = row[i]
		}

		// Apply alignment
		alignment := AlignLeft
		if align, exists := tf.alignments[i]; exists {
			alignment = align
		}

		formattedCell := tf.formatCell(cell, width, alignment, isHeader)
		result.WriteString(formattedCell)

		// Column separator
		if tf.style.BorderStyle.Vertical != "" {
			result.WriteString(tf.style.BorderStyle.Vertical)
		}
	}

	return result.String()
}

// formatCell formats a single cell with padding and alignment
func (tf *tableFormatter) formatCell(content string, width int, alignment Alignment, isHeader bool) string {
	// Truncate content if it's too long
	contentWidth := width - tf.style.Padding*2
	if contentWidth < 0 {
		contentWidth = 0
	}

	if utf8.RuneCountInString(content) > contentWidth {
		runes := []rune(content)
		if contentWidth > 3 {
			content = string(runes[:contentWidth-3]) + "..."
		} else {
			content = string(runes[:contentWidth])
		}
	}

	// Apply color for headers
	if isHeader && tf.colorSystem != nil && tf.colorSystem.IsColorSupported() {
		content = tf.colorSystem.Colorize(content, tf.theme.Primary)
	}

	// Apply alignment
	actualContentWidth := utf8.RuneCountInString(content)
	totalPadding := contentWidth - actualContentWidth

	var leftPad, rightPad int
	switch alignment {
	case AlignCenter:
		leftPad = totalPadding / 2
		rightPad = totalPadding - leftPad
	case AlignRight:
		leftPad = totalPadding
		rightPad = 0
	default: // AlignLeft
		leftPad = 0
		rightPad = totalPadding
	}

	// Add style padding
	leftPad += tf.style.Padding
	rightPad += tf.style.Padding

	return strings.Repeat(" ", leftPad) + content + strings.Repeat(" ", rightPad)
}

// getTerminalWidth returns the current terminal width
func getTerminalWidth() int {
	width, _, err := term.GetSize(0) // stdin
	if err != nil {
		return 80 // Default width
	}
	return width
}
