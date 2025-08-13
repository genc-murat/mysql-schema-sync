package display

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestTableFormatter_BasicTable(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Name", "Age", "City"})
	formatter.AddRow([]string{"John", "30", "New York"})
	formatter.AddRow([]string{"Jane", "25", "Los Angeles"})

	result := formatter.Render()

	// Check that the table contains expected content
	if !strings.Contains(result, "Name") {
		t.Error("Table should contain header 'Name'")
	}
	if !strings.Contains(result, "John") {
		t.Error("Table should contain row data 'John'")
	}
	if !strings.Contains(result, "Jane") {
		t.Error("Table should contain row data 'Jane'")
	}

	// Check that borders are present (default style)
	if !strings.Contains(result, "+") {
		t.Error("Table should contain border characters")
	}
	if !strings.Contains(result, "|") {
		t.Error("Table should contain vertical separators")
	}
}

func TestTableFormatter_EmptyTable(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	result := formatter.Render()

	if result != "" {
		t.Error("Empty table should render as empty string")
	}
}

func TestTableFormatter_HeadersOnly(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Column1", "Column2"})

	result := formatter.Render()

	if !strings.Contains(result, "Column1") {
		t.Error("Table should contain header 'Column1'")
	}
	if !strings.Contains(result, "Column2") {
		t.Error("Table should contain header 'Column2'")
	}
}

func TestTableFormatter_RowsOnly(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.AddRow([]string{"Data1", "Data2"})
	formatter.AddRow([]string{"Data3", "Data4"})

	result := formatter.Render()

	if !strings.Contains(result, "Data1") {
		t.Error("Table should contain row data 'Data1'")
	}
	if !strings.Contains(result, "Data4") {
		t.Error("Table should contain row data 'Data4'")
	}
}

func TestTableFormatter_Alignment(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Left", "Center", "Right"})
	formatter.AddRow([]string{"L", "C", "R"})

	formatter.SetColumnAlignment(0, AlignLeft)
	formatter.SetColumnAlignment(1, AlignCenter)
	formatter.SetColumnAlignment(2, AlignRight)

	result := formatter.Render()

	// Basic check that the table renders without error
	if result == "" {
		t.Error("Table with alignment should not be empty")
	}
}

func TestTableFormatter_ColumnWidth(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Short", "VeryLongHeaderName"})
	formatter.AddRow([]string{"A", "B"})

	formatter.SetColumnWidth(0, 10)
	formatter.SetColumnWidth(1, 5)

	result := formatter.Render()

	if result == "" {
		t.Error("Table with custom column widths should not be empty")
	}
}

func TestTableFormatter_Separators(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Col1", "Col2"})
	formatter.AddRow([]string{"Row1", "Data1"})
	formatter.AddSeparator()
	formatter.AddRow([]string{"Row2", "Data2"})

	result := formatter.Render()

	if result == "" {
		t.Error("Table with separators should not be empty")
	}
}

func TestTableFormatter_Styles(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())

	testCases := []struct {
		name  string
		style TableStyle
	}{
		{"Default", DefaultTableStyle},
		{"Rounded", RoundedTableStyle},
		{"Compact", CompactTableStyle},
		{"Grid", GridTableStyle},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatter := NewTableFormatter(colorSystem, DefaultColorTheme())
			formatter.SetStyle(tc.style)
			formatter.SetHeaders([]string{"Header1", "Header2"})
			formatter.AddRow([]string{"Data1", "Data2"})

			result := formatter.Render()

			if result == "" {
				t.Errorf("Table with %s style should not be empty", tc.name)
			}
		})
	}
}

func TestTableFormatter_LongContent(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Short", "Long"})
	formatter.AddRow([]string{"A", "This is a very long piece of content that should be truncated"})

	formatter.SetColumnWidth(1, 10) // Force truncation

	result := formatter.Render()

	if !strings.Contains(result, "...") {
		t.Error("Long content should be truncated with ellipsis")
	}
}

func TestTableFormatter_UnicodeContent(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Unicode", "Emoji"})
	formatter.AddRow([]string{"测试", "🚀"})
	formatter.AddRow([]string{"Тест", "✅"})

	result := formatter.Render()

	if !strings.Contains(result, "测试") {
		t.Error("Table should handle Unicode characters")
	}
	if !strings.Contains(result, "🚀") {
		t.Error("Table should handle emoji characters")
	}
}

func TestTableFormatter_IrregularRows(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Col1", "Col2", "Col3"})
	formatter.AddRow([]string{"A"})                // Short row
	formatter.AddRow([]string{"B", "C"})           // Medium row
	formatter.AddRow([]string{"D", "E", "F", "G"}) // Long row

	result := formatter.Render()

	if result == "" {
		t.Error("Table with irregular rows should not be empty")
	}
}

func TestTableFormatter_ResponsiveLayout(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	style := DefaultTableStyle
	style.MaxWidth = 50
	style.Responsive = true
	formatter.SetStyle(style)

	formatter.SetHeaders([]string{"VeryLongHeaderName1", "VeryLongHeaderName2", "VeryLongHeaderName3"})
	formatter.AddRow([]string{"VeryLongDataContent1", "VeryLongDataContent2", "VeryLongDataContent3"})

	result := formatter.Render()

	if result == "" {
		t.Error("Responsive table should not be empty")
	}

	// Check that the table doesn't exceed the maximum width significantly
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if len(line) > 60 { // Allow some tolerance
			t.Errorf("Table line exceeds expected width: %d characters", len(line))
		}
	}
}

func TestBorderStyles(t *testing.T) {
	testCases := []struct {
		name  string
		style BorderStyle
	}{
		{"ASCII", ASCIIBorderStyle},
		{"Rounded", RoundedBorderStyle},
		{"NoBorder", NoBorderStyle},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that border style has expected characters
			if tc.name == "ASCII" {
				if tc.style.TopLeft != "+" {
					t.Error("ASCII border should use '+' for corners")
				}
				if tc.style.Horizontal != "-" {
					t.Error("ASCII border should use '-' for horizontal lines")
				}
			}
			if tc.name == "Rounded" {
				if tc.style.TopLeft != "╭" {
					t.Error("Rounded border should use '╭' for top-left corner")
				}
			}
			if tc.name == "NoBorder" {
				if tc.style.TopLeft != "" {
					t.Error("No border style should use empty strings")
				}
			}
		})
	}
}

func TestAlignment(t *testing.T) {
	testCases := []struct {
		name      string
		alignment Alignment
	}{
		{"Left", AlignLeft},
		{"Center", AlignCenter},
		{"Right", AlignRight},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			colorSystem := NewColorSystem(DefaultColorTheme())
			formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

			formatter.SetHeaders([]string{"Test"})
			formatter.AddRow([]string{"X"})
			formatter.SetColumnAlignment(0, tc.alignment)

			result := formatter.Render()

			if result == "" {
				t.Errorf("Table with %s alignment should not be empty", tc.name)
			}
		})
	}
}

// Comprehensive tests for table formatting with various data sizes and terminal widths
func TestTableFormatterComprehensive(t *testing.T) {
	t.Run("VeryLargeTable", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		// Create large table
		headers := make([]string, 20)
		for i := 0; i < 20; i++ {
			headers[i] = fmt.Sprintf("Column%d", i+1)
		}
		formatter.SetHeaders(headers)

		// Add many rows
		for i := 0; i < 100; i++ {
			row := make([]string, 20)
			for j := 0; j < 20; j++ {
				row[j] = fmt.Sprintf("Data%d-%d", i+1, j+1)
			}
			formatter.AddRow(row)
		}

		result := formatter.Render()
		if result == "" {
			t.Error("Large table should render successfully")
		}

		// Should contain first and last row data
		if !strings.Contains(result, "Data1-1") {
			t.Error("Should contain first row data")
		}
		if !strings.Contains(result, "Data100-20") {
			t.Error("Should contain last row data")
		}
	})

	t.Run("VeryWideContent", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		// Create content that's very wide
		veryLongText := strings.Repeat("This is a very long piece of text that should be handled properly. ", 10)

		formatter.SetHeaders([]string{"Short", "VeryLongContent"})
		formatter.AddRow([]string{"A", veryLongText})

		result := formatter.Render()
		if result == "" {
			t.Error("Table with very wide content should render")
		}
	})

	t.Run("EmptyAndNilContent", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Empty", "Nil", "Normal"})
		formatter.AddRow([]string{"", "", "Content"})
		formatter.AddRow([]string{})    // Empty row
		formatter.AddRow([]string{"A"}) // Partial row

		result := formatter.Render()
		if result == "" {
			t.Error("Table with empty/nil content should render")
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Special", "Unicode", "Symbols"})
		formatter.AddRow([]string{"!@#$%^&*()", "测试🚀", "←→↑↓"})
		formatter.AddRow([]string{"<>&\"'", "Тест✅", "▲▼◄►"})

		result := formatter.Render()
		if result == "" {
			t.Error("Table with special characters should render")
		}

		// Should contain the special characters
		if !strings.Contains(result, "测试🚀") {
			t.Error("Should preserve Unicode characters")
		}
	})

	t.Run("DifferentTerminalWidths", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())

		widths := []int{40, 80, 120, 200}
		for _, width := range widths {
			t.Run(fmt.Sprintf("Width%d", width), func(t *testing.T) {
				formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

				style := DefaultTableStyle
				style.MaxWidth = width
				style.Responsive = true
				formatter.SetStyle(style)

				formatter.SetHeaders([]string{"Col1", "Col2", "Col3", "Col4"})
				formatter.AddRow([]string{"Data1", "Data2", "Data3", "Data4"})

				result := formatter.Render()
				if result == "" {
					t.Errorf("Table should render at width %d", width)
				}

				// Check that lines don't significantly exceed the width
				lines := strings.Split(result, "\n")
				for _, line := range lines {
					if len(line) > width+10 { // Allow some tolerance
						t.Errorf("Line exceeds width %d: %d characters", width, len(line))
					}
				}
			})
		}
	})

	t.Run("AllTableStyles", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())

		styles := []TableStyle{
			DefaultTableStyle,
			RoundedTableStyle,
			CompactTableStyle,
			GridTableStyle,
		}

		for _, style := range styles {
			t.Run(style.Name, func(t *testing.T) {
				formatter := NewTableFormatter(colorSystem, DefaultColorTheme())
				formatter.SetStyle(style)

				formatter.SetHeaders([]string{"Header1", "Header2"})
				formatter.AddRow([]string{"Data1", "Data2"})
				formatter.AddRow([]string{"Data3", "Data4"})

				result := formatter.Render()
				if result == "" {
					t.Errorf("Table with style %s should render", style.Name)
				}

				// Style-specific checks
				switch style.Name {
				case "compact":
					// Compact style should have minimal borders
					if strings.Count(result, "+") > 0 && style.BorderStyle.TopLeft == "" {
						t.Error("Compact style should not have border characters")
					}
				case "grid":
					// Grid style should have row separators
					if style.RowSeparator && !strings.Contains(result, "+") {
						t.Error("Grid style should have border characters")
					}
				}
			})
		}
	})

	t.Run("ComplexAlignment", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Left", "Center", "Right", "Default"})
		formatter.AddRow([]string{"L", "C", "R", "D"})
		formatter.AddRow([]string{"Long Left", "Long Center", "Long Right", "Long Default"})

		formatter.SetColumnAlignment(0, AlignLeft)
		formatter.SetColumnAlignment(1, AlignCenter)
		formatter.SetColumnAlignment(2, AlignRight)
		// Column 3 uses default alignment

		result := formatter.Render()
		if result == "" {
			t.Error("Table with complex alignment should render")
		}
	})

	t.Run("CustomColumnWidths", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Narrow", "Wide", "Auto"})
		formatter.AddRow([]string{"N", "This is wide content", "Auto-sized"})

		formatter.SetColumnWidth(0, 5)  // Very narrow
		formatter.SetColumnWidth(1, 30) // Very wide
		// Column 2 uses auto width

		result := formatter.Render()
		if result == "" {
			t.Error("Table with custom column widths should render")
		}
	})

	t.Run("ManySeparators", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Col1", "Col2"})
		formatter.AddRow([]string{"Row1", "Data1"})
		formatter.AddSeparator()
		formatter.AddRow([]string{"Row2", "Data2"})
		formatter.AddSeparator()
		formatter.AddRow([]string{"Row3", "Data3"})
		formatter.AddSeparator()

		result := formatter.Render()
		if result == "" {
			t.Error("Table with many separators should render")
		}
	})

	t.Run("ColorThemes", func(t *testing.T) {
		themes := []ColorTheme{
			DarkColorTheme(),
			LightColorTheme(),
			HighContrastColorTheme(),
			PlainTextTheme(),
		}

		for i, theme := range themes {
			t.Run(fmt.Sprintf("Theme%d", i), func(t *testing.T) {
				colorSystem := NewColorSystem(theme)
				formatter := NewTableFormatter(colorSystem, theme)

				formatter.SetHeaders([]string{"Header1", "Header2"})
				formatter.AddRow([]string{"Data1", "Data2"})

				result := formatter.Render()
				if result == "" {
					t.Errorf("Table with theme %d should render", i)
				}
			})
		}
	})

	t.Run("EdgeCaseWidths", func(t *testing.T) {
		colorSystem := NewColorSystem(DefaultColorTheme())
		formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

		formatter.SetHeaders([]string{"Test"})
		formatter.AddRow([]string{"Data"})

		// Test very small widths
		formatter.SetColumnWidth(0, 1)
		result := formatter.Render()
		if result == "" {
			t.Error("Table should handle very small column width")
		}

		// Test zero width
		formatter.SetColumnWidth(0, 0)
		result = formatter.Render()
		if result == "" {
			t.Error("Table should handle zero column width")
		}
	})
}

func TestTableFormatterRenderTo(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Col1", "Col2"})
	formatter.AddRow([]string{"Data1", "Data2"})

	var buf bytes.Buffer
	formatter.RenderTo(&buf)

	output := buf.String()
	if output == "" {
		t.Error("RenderTo should write output to buffer")
	}

	// Should be the same as Render()
	directRender := formatter.Render()
	if output != directRender {
		t.Error("RenderTo output should match Render output")
	}
}

func TestBorderStylesComprehensive(t *testing.T) {
	styles := map[string]BorderStyle{
		"ASCII":    ASCIIBorderStyle,
		"Rounded":  RoundedBorderStyle,
		"NoBorder": NoBorderStyle,
	}

	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			colorSystem := NewColorSystem(DefaultColorTheme())
			formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

			tableStyle := DefaultTableStyle
			tableStyle.BorderStyle = style
			formatter.SetStyle(tableStyle)

			formatter.SetHeaders([]string{"Test1", "Test2"})
			formatter.AddRow([]string{"Data1", "Data2"})

			result := formatter.Render()
			if result == "" {
				t.Errorf("Table with %s border style should render", name)
			}

			// Check for expected border characters
			switch name {
			case "ASCII":
				if !strings.Contains(result, "+") || !strings.Contains(result, "-") {
					t.Error("ASCII style should contain + and - characters")
				}
			case "Rounded":
				if !strings.Contains(result, "─") {
					t.Error("Rounded style should contain Unicode box drawing characters")
				}
			case "NoBorder":
				if strings.Contains(result, "+") || strings.Contains(result, "─") {
					t.Error("NoBorder style should not contain border characters")
				}
			}
		})
	}
}

func TestTableFormatterConcurrency(t *testing.T) {
	// Test that table formatter is safe for concurrent read operations
	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	formatter.SetHeaders([]string{"Col1", "Col2", "Col3"})
	for i := 0; i < 10; i++ {
		formatter.AddRow([]string{fmt.Sprintf("Data%d-1", i), fmt.Sprintf("Data%d-2", i), fmt.Sprintf("Data%d-3", i)})
	}

	// Render concurrently
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			result := formatter.Render()
			if result == "" {
				t.Error("Concurrent render should produce output")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
