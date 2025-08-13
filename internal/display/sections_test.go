package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestSectionFormatter_RenderSection(t *testing.T) {
	tests := []struct {
		name        string
		section     *Section
		expected    []string // Expected strings to be present in output
		notExpected []string // Strings that should NOT be present
	}{
		{
			name: "simple section",
			section: &Section{
				Title:   "Test Section",
				Content: "This is test content",
			},
			expected: []string{"Test Section", "This is test content"},
		},
		{
			name: "section with statistics",
			section: &Section{
				Title:   "Section with Stats",
				Content: "Content here",
				Statistics: &SectionStatistics{
					ItemCount:    5,
					SuccessCount: 3,
					WarningCount: 1,
					ErrorCount:   1,
				},
			},
			expected: []string{"Section with Stats", "Items: 5", "Success: 3", "Warnings: 1", "Errors: 1"},
		},
		{
			name: "collapsible section expanded",
			section: &Section{
				Title:       "Collapsible Section",
				Content:     "This content should be visible",
				Collapsible: true,
				Collapsed:   false,
			},
			expected: []string{"Collapsible Section", "This content should be visible"},
		},
		{
			name: "collapsible section collapsed",
			section: &Section{
				Title:       "Collapsed Section",
				Content:     "This content should be hidden",
				Collapsible: true,
				Collapsed:   true,
			},
			expected:    []string{"Collapsed Section"},
			notExpected: []string{"This content should be hidden"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			colorSystem := NewColorSystem(DefaultColorTheme())
			iconSystem := NewIconSystem()

			formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), &buf)
			formatter.RenderSection(tt.section)

			output := buf.String()

			// Check expected strings are present
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", expected, output)
				}
			}

			// Check that strings that shouldn't be present aren't there
			if tt.notExpected != nil {
				for _, notExpected := range tt.notExpected {
					if strings.Contains(output, notExpected) {
						t.Errorf("Expected output to NOT contain %q, but it did. Output:\n%s", notExpected, output)
					}
				}
			}
		})
	}
}

func TestSectionFormatter_RenderSections(t *testing.T) {
	sections := []*Section{
		{
			Title:   "First Section",
			Content: "First content",
		},
		{
			Title:   "Second Section",
			Content: "Second content",
		},
	}

	var buf bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), &buf)
	formatter.RenderSections(sections)

	output := buf.String()

	expected := []string{"First Section", "First content", "Second Section", "Second content"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", exp, output)
		}
	}
}

func TestSection_AddSubsection(t *testing.T) {
	parent := NewSection("Parent Section")
	child := NewSection("Child Section")

	parent.AddSubsection(child)

	if len(parent.Subsections) != 1 {
		t.Errorf("Expected 1 subsection, got %d", len(parent.Subsections))
	}

	if parent.Subsections[0] != child {
		t.Error("Subsection was not added correctly")
	}

	if child.Level != 1 {
		t.Errorf("Expected child level to be 1, got %d", child.Level)
	}
}

func TestSectionFormatter_NestedSections(t *testing.T) {
	parent := NewSection("Parent Section")
	child1 := NewSection("Child 1")
	child2 := NewSection("Child 2")
	grandchild := NewSection("Grandchild")

	child1.SetContent("Child 1 content")
	child2.SetContent("Child 2 content")
	grandchild.SetContent("Grandchild content")

	child1.AddSubsection(grandchild)
	parent.AddSubsection(child1)
	parent.AddSubsection(child2)

	var buf bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), &buf)
	formatter.RenderSection(parent)

	output := buf.String()

	expected := []string{
		"Parent Section",
		"Child 1",
		"Child 1 content",
		"Child 2",
		"Child 2 content",
		"Grandchild",
		"Grandchild content",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", exp, output)
		}
	}
}

func TestSectionStatistics_CustomStats(t *testing.T) {
	stats := NewSectionStatistics()
	stats.ItemCount = 10
	stats.AddCustomStat("Custom Metric", 42)
	stats.AddCustomStat("Another Metric", "test value")

	section := &Section{
		Title:      "Section with Custom Stats",
		Statistics: stats,
	}

	var buf bytes.Buffer
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), &buf)
	formatter.RenderSection(section)

	output := buf.String()

	expected := []string{
		"Items: 10",
		"Custom Metric: 42",
		"Another Metric: test value",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", exp, output)
		}
	}
}

func TestSectionFormatter_ContentTypes(t *testing.T) {
	tests := []struct {
		name     string
		content  interface{}
		expected []string
	}{
		{
			name:     "string content",
			content:  "Simple string content",
			expected: []string{"Simple string content"},
		},
		{
			name:     "string slice content",
			content:  []string{"Item 1", "Item 2", "Item 3"},
			expected: []string{"Item 1", "Item 2", "Item 3"},
		},
		{
			name: "map content",
			content: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			expected: []string{"key1: value1", "key2: 42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := &Section{
				Title:   "Test Section",
				Content: tt.content,
			}

			var buf bytes.Buffer
			colorSystem := NewColorSystem(DefaultColorTheme())
			iconSystem := NewIconSystem()

			formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), &buf)
			formatter.RenderSection(section)

			output := buf.String()

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, expected %q", tt.bytes, result, tt.expected)
		}
	}
}
