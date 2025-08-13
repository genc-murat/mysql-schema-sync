package display

import (
	"fmt"
	"os"
	"testing"

	"mysql-schema-sync/internal/schema"
)

// TestTableFormatterExample demonstrates the table formatter capabilities
func TestTableFormatterExample(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()

	fmt.Println("\n=== Table Formatter Examples ===")

	// Example 1: Basic table
	fmt.Println("\n1. Basic Table:")
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())
	formatter.SetHeaders([]string{"Name", "Type", "Nullable", "Default"})
	formatter.AddRow([]string{"id", "int(11)", "NO", "NULL"})
	formatter.AddRow([]string{"name", "varchar(255)", "YES", "''"})
	formatter.AddRow([]string{"email", "varchar(100)", "NO", "NULL"})
	formatter.RenderTo(os.Stdout)

	// Example 2: Different styles
	fmt.Println("\n2. Rounded Style:")
	formatter2 := NewTableFormatter(colorSystem, DefaultColorTheme())
	formatter2.SetStyle(RoundedTableStyle)
	formatter2.SetHeaders([]string{"Table", "Operation", "Status"})
	formatter2.AddRow([]string{"users", "CREATE", "✓ Success"})
	formatter2.AddRow([]string{"orders", "ALTER", "⚠ Warning"})
	formatter2.AddRow([]string{"products", "DROP", "✗ Error"})
	formatter2.RenderTo(os.Stdout)

	// Example 3: Grid style with separators
	fmt.Println("\n3. Grid Style with Separators:")
	formatter3 := NewTableFormatter(colorSystem, DefaultColorTheme())
	formatter3.SetStyle(GridTableStyle)
	formatter3.SetHeaders([]string{"Database", "Tables", "Size"})
	formatter3.AddRow([]string{"production", "25", "2.5GB"})
	formatter3.AddSeparator()
	formatter3.AddRow([]string{"staging", "25", "1.2GB"})
	formatter3.AddSeparator()
	formatter3.AddRow([]string{"development", "20", "500MB"})
	formatter3.RenderTo(os.Stdout)

	// Example 4: Compact style
	fmt.Println("\n4. Compact Style:")
	formatter4 := NewTableFormatter(colorSystem, DefaultColorTheme())
	formatter4.SetStyle(CompactTableStyle)
	formatter4.SetHeaders([]string{"Index", "Columns", "Unique"})
	formatter4.AddRow([]string{"idx_email", "email", "YES"})
	formatter4.AddRow([]string{"idx_name", "name", "NO"})
	formatter4.AddRow([]string{"idx_created", "created_at", "NO"})
	formatter4.RenderTo(os.Stdout)

	fmt.Println("\n=== Schema Diff Presenter Examples ===")

	// Example 5: Schema diff presentation
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	// Create sample schema diff
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "audit_log",
				Columns: map[string]*schema.Column{
					"id":         {Name: "id", DataType: "bigint", IsNullable: false},
					"table_name": {Name: "table_name", DataType: "varchar(100)", IsNullable: false},
					"action":     {Name: "action", DataType: "varchar(50)", IsNullable: false},
					"created_at": {Name: "created_at", DataType: "timestamp", IsNullable: false},
				},
				Indexes:     []*schema.Index{},
				Constraints: map[string]*schema.Constraint{},
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "users",
				AddedColumns: []*schema.Column{
					{Name: "last_login", DataType: "timestamp", IsNullable: true},
					{Name: "status", DataType: "enum('active','inactive')", IsNullable: false},
				},
				RemovedColumns: []*schema.Column{
					{Name: "old_field", DataType: "varchar(50)", IsNullable: true},
				},
				ModifiedColumns: []*schema.ColumnDiff{
					{
						ColumnName: "email",
						OldColumn:  &schema.Column{Name: "email", DataType: "varchar(100)", IsNullable: false},
						NewColumn:  &schema.Column{Name: "email", DataType: "varchar(255)", IsNullable: false},
					},
				},
			},
		},
		AddedIndexes: []*schema.Index{
			{Name: "idx_status", TableName: "users", Columns: []string{"status"}, IsUnique: false, IndexType: "BTREE"},
			{Name: "idx_last_login", TableName: "users", Columns: []string{"last_login"}, IsUnique: false, IndexType: "BTREE"},
		},
		AddedConstraints: []*schema.Constraint{
			{
				Name:              "fk_audit_user",
				TableName:         "audit_log",
				Type:              schema.ConstraintTypeForeignKey,
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedColumns: []string{"id"},
			},
		},
	}

	fmt.Println("\n5. Schema Diff Summary:")
	result := presenter.FormatSchemaDiff(diff)
	fmt.Print(result)

	fmt.Println("\n=== Examples Complete ===")
}

// TestTableFormatterAlignment demonstrates different alignment options
func TestTableFormatterAlignment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping alignment example in short mode")
	}

	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	fmt.Println("\n=== Alignment Examples ===")

	formatter.SetHeaders([]string{"Left", "Center", "Right", "Default"})
	formatter.SetColumnAlignment(0, AlignLeft)
	formatter.SetColumnAlignment(1, AlignCenter)
	formatter.SetColumnAlignment(2, AlignRight)
	// Column 3 uses default (left) alignment

	formatter.AddRow([]string{"Short", "Medium Text", "Very Long Content", "Normal"})
	formatter.AddRow([]string{"Very Long Content Here", "Short", "Med", "Test"})
	formatter.AddRow([]string{"X", "Y", "Z", "ABC"})

	formatter.RenderTo(os.Stdout)
	fmt.Println("\n=== Alignment Examples Complete ===")
}

// TestTableFormatterResponsive demonstrates responsive layout
func TestTableFormatterResponsive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping responsive example in short mode")
	}

	colorSystem := NewColorSystem(DefaultColorTheme())
	formatter := NewTableFormatter(colorSystem, DefaultColorTheme())

	fmt.Println("\n=== Responsive Layout Example ===")

	style := DefaultTableStyle
	style.MaxWidth = 60 // Force narrow width
	style.Responsive = true
	formatter.SetStyle(style)

	formatter.SetHeaders([]string{"Very Long Header Name 1", "Very Long Header Name 2", "Very Long Header Name 3"})
	formatter.AddRow([]string{
		"This is very long content that should be truncated",
		"Another very long piece of content here",
		"And yet another long content piece",
	})
	formatter.AddRow([]string{"Short", "Medium length content", "X"})

	formatter.RenderTo(os.Stdout)
	fmt.Println("\n=== Responsive Layout Example Complete ===")
}
