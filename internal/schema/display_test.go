package schema

import (
	"strings"
	"testing"
)

func TestDisplayFormatter_IsEmpty(t *testing.T) {
	formatter := NewDisplayFormatter(false, false)

	tests := []struct {
		name     string
		diff     *SchemaDiff
		expected bool
	}{
		{
			name:     "empty diff",
			diff:     &SchemaDiff{},
			expected: true,
		},
		{
			name: "diff with added table",
			diff: &SchemaDiff{
				AddedTables: []*Table{NewTable("test_table")},
			},
			expected: false,
		},
		{
			name: "diff with removed table",
			diff: &SchemaDiff{
				RemovedTables: []*Table{NewTable("test_table")},
			},
			expected: false,
		},
		{
			name: "diff with modified table",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName:    "test_table",
						AddedColumns: []*Column{NewColumn("new_col", "VARCHAR(255)", true)},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.IsEmpty(tt.diff)
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDisplayFormatter_GetChangeSummary(t *testing.T) {
	formatter := NewDisplayFormatter(false, false)

	tests := []struct {
		name     string
		diff     *SchemaDiff
		expected string
	}{
		{
			name:     "empty diff",
			diff:     &SchemaDiff{},
			expected: "No changes detected",
		},
		{
			name: "table changes only",
			diff: &SchemaDiff{
				AddedTables:   []*Table{NewTable("table1")},
				RemovedTables: []*Table{NewTable("table2")},
			},
			expected: "2 table changes",
		},
		{
			name: "column changes only",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "test_table",
						AddedColumns: []*Column{
							NewColumn("col1", "VARCHAR(255)", true),
							NewColumn("col2", "INT", false),
						},
						RemovedColumns: []*Column{
							NewColumn("old_col", "TEXT", true),
						},
					},
				},
			},
			expected: "3 column changes",
		},
		{
			name: "mixed changes",
			diff: &SchemaDiff{
				AddedTables: []*Table{NewTable("new_table")},
				ModifiedTables: []*TableDiff{
					{
						TableName:    "existing_table",
						AddedColumns: []*Column{NewColumn("new_col", "INT", false)},
					},
				},
				AddedIndexes: []*Index{
					NewIndex("idx_test", "test_table", []string{"col1"}),
				},
			},
			expected: "1 table changes, 1 column changes, 1 index changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.GetChangeSummary(tt.diff)
			if result != tt.expected {
				t.Errorf("GetChangeSummary() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
func TestDisplayFormatter_FormatSchemaDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     *SchemaDiff
		details  bool
		colors   bool
		contains []string
	}{
		{
			name:     "empty diff",
			diff:     &SchemaDiff{},
			details:  false,
			colors:   false,
			contains: []string{"No schema differences found", "synchronized"},
		},
		{
			name: "added table",
			diff: &SchemaDiff{
				AddedTables: []*Table{
					func() *Table {
						table := NewTable("users")
						table.AddColumn(NewColumn("id", "INT", false))
						table.AddColumn(NewColumn("name", "VARCHAR(255)", false))
						return table
					}(),
				},
			},
			details:  true,
			colors:   false,
			contains: []string{"Added Tables", "users", "id INT", "name VARCHAR(255)"},
		},
		{
			name: "removed table",
			diff: &SchemaDiff{
				RemovedTables: []*Table{
					func() *Table {
						table := NewTable("old_table")
						table.AddColumn(NewColumn("id", "INT", false))
						return table
					}(),
				},
			},
			details:  false,
			colors:   false,
			contains: []string{"Removed Tables", "old_table"},
		},
		{
			name: "modified table with column changes",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "products",
						AddedColumns: []*Column{
							NewColumn("description", "TEXT", true),
						},
						RemovedColumns: []*Column{
							NewColumn("old_field", "VARCHAR(100)", true),
						},
						ModifiedColumns: []*ColumnDiff{
							{
								ColumnName: "price",
								OldColumn:  NewColumn("price", "DECIMAL(10,2)", false),
								NewColumn:  NewColumn("price", "DECIMAL(12,2)", true),
							},
						},
					},
				},
			},
			details: true,
			colors:  false,
			contains: []string{
				"Modified Tables", "products",
				"Added Columns", "description TEXT",
				"Removed Columns", "old_field VARCHAR(100)",
				"Modified Columns", "price",
				"Data Type:", "DECIMAL(10,2)", "DECIMAL(12,2)",
				"Nullability:", "NOT NULL", "NULL",
			},
		},
		{
			name: "index changes",
			diff: &SchemaDiff{
				AddedIndexes: []*Index{
					func() *Index {
						idx := NewIndex("idx_name", "users", []string{"name"})
						idx.IsUnique = true
						return idx
					}(),
				},
				RemovedIndexes: []*Index{
					NewIndex("idx_old", "users", []string{"old_col"}),
				},
			},
			details: false,
			colors:  false,
			contains: []string{
				"Indexes",
				"Added Indexes", "idx_name", "UNIQUE",
				"Removed Indexes", "idx_old",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewDisplayFormatter(tt.details, tt.colors)
			result := formatter.FormatSchemaDiff(tt.diff)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatSchemaDiff() result does not contain %q\nResult:\n%s", expected, result)
				}
			}
		})
	}
}
func TestDisplayFormatter_FormatCompactSummary(t *testing.T) {
	tests := []struct {
		name     string
		diff     *SchemaDiff
		colors   bool
		contains []string
	}{
		{
			name:     "empty diff without colors",
			diff:     &SchemaDiff{},
			colors:   false,
			contains: []string{"✓", "synchronized"},
		},
		{
			name: "changes without colors",
			diff: &SchemaDiff{
				AddedTables: []*Table{NewTable("test")},
			},
			colors:   false,
			contains: []string{"⚠", "Found:", "1 table changes"},
		},
		{
			name:     "empty diff with colors",
			diff:     &SchemaDiff{},
			colors:   true,
			contains: []string{"✓", "synchronized", "\033[32m", "\033[0m"}, // green color codes
		},
		{
			name: "changes with colors",
			diff: &SchemaDiff{
				AddedTables: []*Table{NewTable("test")},
			},
			colors:   true,
			contains: []string{"⚠", "Found:", "1 table changes", "\033[33m", "\033[0m"}, // yellow color codes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewDisplayFormatter(false, tt.colors)
			result := formatter.FormatCompactSummary(tt.diff)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatCompactSummary() result does not contain %q\nResult: %q", expected, result)
				}
			}
		})
	}
}

func TestDisplayFormatter_Colorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		color    string
		useColor bool
		expected string
	}{
		{
			name:     "no color",
			text:     "test",
			color:    "red",
			useColor: false,
			expected: "test",
		},
		{
			name:     "red color",
			text:     "error",
			color:    "red",
			useColor: true,
			expected: "\033[31merror\033[0m",
		},
		{
			name:     "green color",
			text:     "success",
			color:    "green",
			useColor: true,
			expected: "\033[32msuccess\033[0m",
		},
		{
			name:     "bold text",
			text:     "header",
			color:    "bold",
			useColor: true,
			expected: "\033[1mheader\033[0m",
		},
		{
			name:     "invalid color",
			text:     "test",
			color:    "invalid",
			useColor: true,
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewDisplayFormatter(false, tt.useColor)
			result := formatter.colorize(tt.text, tt.color)
			if result != tt.expected {
				t.Errorf("colorize() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
