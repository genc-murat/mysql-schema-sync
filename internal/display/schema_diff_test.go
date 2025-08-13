package display

import (
	"strings"
	"testing"

	"mysql-schema-sync/internal/schema"
)

func TestSchemaDiffPresenter_FormatSchemaDiff(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	// Create a sample schema diff
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name:        "users",
				Columns:     map[string]*schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes:     []*schema.Index{},
				Constraints: map[string]*schema.Constraint{},
			},
		},
		RemovedTables: []*schema.Table{
			{
				Name:        "old_table",
				Columns:     map[string]*schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes:     []*schema.Index{},
				Constraints: map[string]*schema.Constraint{},
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "products",
				AddedColumns: []*schema.Column{
					{Name: "description", DataType: "text", IsNullable: true},
				},
				RemovedColumns: []*schema.Column{
					{Name: "old_field", DataType: "varchar(50)", IsNullable: false},
				},
			},
		},
	}

	result := presenter.FormatSchemaDiff(diff)

	// Check that the result contains expected content
	if !strings.Contains(result, "Schema Changes Summary") {
		t.Error("Result should contain summary section")
	}
	if !strings.Contains(result, "users") {
		t.Error("Result should contain added table 'users'")
	}
	if !strings.Contains(result, "old_table") {
		t.Error("Result should contain removed table 'old_table'")
	}
	if !strings.Contains(result, "products") {
		t.Error("Result should contain modified table 'products'")
	}
	if !strings.Contains(result, "description") {
		t.Error("Result should contain added column 'description'")
	}
	if !strings.Contains(result, "old_field") {
		t.Error("Result should contain removed column 'old_field'")
	}
}

func TestSchemaDiffPresenter_EmptyDiff(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	// Create an empty schema diff
	diff := &schema.SchemaDiff{}

	result := presenter.FormatSchemaDiff(diff)

	if !strings.Contains(result, "No schema changes detected") {
		t.Error("Empty diff should indicate no changes detected")
	}
}

func TestSchemaDiffPresenter_AddedTables(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	table := &schema.Table{
		Name: "test_table",
		Columns: map[string]*schema.Column{
			"id":   {Name: "id", DataType: "int"},
			"name": {Name: "name", DataType: "varchar(100)"},
		},
		Indexes: []*schema.Index{
			{Name: "idx_name", TableName: "test_table", Columns: []string{"name"}},
		},
		Constraints: map[string]*schema.Constraint{
			"pk_id": {Name: "pk_id", TableName: "test_table", Type: schema.ConstraintTypeUnique, Columns: []string{"id"}},
		},
	}

	result := presenter.formatAddedTables([]*schema.Table{table})

	if !strings.Contains(result, "Added Tables") {
		t.Error("Result should contain 'Added Tables' header")
	}
	if !strings.Contains(result, "test_table") {
		t.Error("Result should contain table name")
	}
	if !strings.Contains(result, "2") { // 2 columns
		t.Error("Result should show column count")
	}
	if !strings.Contains(result, "1") { // 1 index and 1 constraint
		t.Error("Result should show index and constraint counts")
	}
}

func TestSchemaDiffPresenter_ModifiedTables(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	tableDiff := &schema.TableDiff{
		TableName: "modified_table",
		AddedColumns: []*schema.Column{
			{Name: "new_col", DataType: "varchar(255)", IsNullable: true},
		},
		RemovedColumns: []*schema.Column{
			{Name: "old_col", DataType: "int", IsNullable: false},
		},
		ModifiedColumns: []*schema.ColumnDiff{
			{
				ColumnName: "changed_col",
				OldColumn:  &schema.Column{Name: "changed_col", DataType: "varchar(50)", IsNullable: false},
				NewColumn:  &schema.Column{Name: "changed_col", DataType: "varchar(100)", IsNullable: true},
			},
		},
	}

	result := presenter.formatTableDiff(tableDiff)

	if !strings.Contains(result, "modified_table") {
		t.Error("Result should contain table name")
	}
	if !strings.Contains(result, "new_col") {
		t.Error("Result should contain added column")
	}
	if !strings.Contains(result, "old_col") {
		t.Error("Result should contain removed column")
	}
	if !strings.Contains(result, "changed_col") {
		t.Error("Result should contain modified column")
	}
	if !strings.Contains(result, "varchar(50)") {
		t.Error("Result should contain old column type")
	}
	if !strings.Contains(result, "varchar(100)") {
		t.Error("Result should contain new column type")
	}
}

func TestSchemaDiffPresenter_IndexChanges(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	diff := &schema.SchemaDiff{
		AddedIndexes: []*schema.Index{
			{
				Name:      "idx_new",
				TableName: "test_table",
				Columns:   []string{"col1", "col2"},
				IsUnique:  true,
				IndexType: "BTREE",
			},
		},
		RemovedIndexes: []*schema.Index{
			{
				Name:      "idx_old",
				TableName: "test_table",
				Columns:   []string{"old_col"},
				IsUnique:  false,
				IndexType: "BTREE",
			},
		},
	}

	result := presenter.formatIndexChanges(diff)

	if !strings.Contains(result, "Index Changes") {
		t.Error("Result should contain 'Index Changes' header")
	}
	if !strings.Contains(result, "idx_new") {
		t.Error("Result should contain added index name")
	}
	if !strings.Contains(result, "idx_old") {
		t.Error("Result should contain removed index name")
	}
	if !strings.Contains(result, "col1, col2") {
		t.Error("Result should contain index columns")
	}
	if !strings.Contains(result, "YES") {
		t.Error("Result should show unique status")
	}
	if !strings.Contains(result, "NO") {
		t.Error("Result should show non-unique status")
	}
}

func TestSchemaDiffPresenter_ConstraintChanges(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	diff := &schema.SchemaDiff{
		AddedConstraints: []*schema.Constraint{
			{
				Name:              "fk_user_id",
				TableName:         "orders",
				Type:              schema.ConstraintTypeForeignKey,
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedColumns: []string{"id"},
			},
		},
		RemovedConstraints: []*schema.Constraint{
			{
				Name:      "uk_email",
				TableName: "users",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"email"},
			},
		},
	}

	result := presenter.formatConstraintChanges(diff)

	if !strings.Contains(result, "Constraint Changes") {
		t.Error("Result should contain 'Constraint Changes' header")
	}
	if !strings.Contains(result, "fk_user_id") {
		t.Error("Result should contain added constraint name")
	}
	if !strings.Contains(result, "uk_email") {
		t.Error("Result should contain removed constraint name")
	}
	if !strings.Contains(result, "FOREIGN_KEY") {
		t.Error("Result should contain constraint type")
	}
	if !strings.Contains(result, "users(id)") {
		t.Error("Result should contain foreign key reference")
	}
}

func TestSchemaDiffPresenter_ChangeIcons(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	addIcon := presenter.getChangeIcon(ChangeAdded)
	removeIcon := presenter.getChangeIcon(ChangeRemoved)
	modifyIcon := presenter.getChangeIcon(ChangeModified)

	if addIcon == "" {
		t.Error("Add icon should not be empty")
	}
	if removeIcon == "" {
		t.Error("Remove icon should not be empty")
	}
	if modifyIcon == "" {
		t.Error("Modify icon should not be empty")
	}

	// Icons should be different
	if addIcon == removeIcon || addIcon == modifyIcon || removeIcon == modifyIcon {
		t.Error("Change icons should be different from each other")
	}
}

func TestSchemaDiffPresenter_FormatHelpers(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	// Test nullable formatting
	if presenter.formatNullable(true) != "YES" {
		t.Error("Nullable true should format as 'YES'")
	}
	if presenter.formatNullable(false) != "NO" {
		t.Error("Nullable false should format as 'NO'")
	}

	// Test boolean formatting
	if presenter.formatBoolean(true) != "YES" {
		t.Error("Boolean true should format as 'YES'")
	}
	if presenter.formatBoolean(false) != "NO" {
		t.Error("Boolean false should format as 'NO'")
	}

	// Test text indentation
	text := "line1\nline2\nline3"
	indented := presenter.indentText(text, "  ")
	if !strings.Contains(indented, "  line1") {
		t.Error("Text should be properly indented")
	}
}

func TestSchemaDiffPresenter_NameFormatting(t *testing.T) {
	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	presenter := NewSchemaDiffPresenter(colorSystem, iconSystem, DefaultColorTheme())

	// Test table name formatting
	tables := []*schema.Table{
		{Name: "table1"},
		{Name: "table2"},
		{Name: "table3"},
	}
	result := presenter.formatTableNames(tables)
	expected := "table1, table2, table3"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test index name formatting
	indexes := []*schema.Index{
		{Name: "idx1", TableName: "table1"},
		{Name: "idx2", TableName: "table2"},
	}
	result = presenter.formatIndexNames(indexes)
	expected = "table1.idx1, table2.idx2"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestChangeType_String(t *testing.T) {
	testCases := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeAdded, "ADDED"},
		{ChangeRemoved, "REMOVED"},
		{ChangeModified, "MODIFIED"},
		{ChangeType(999), "UNKNOWN"}, // Invalid change type
	}

	for _, tc := range testCases {
		result := tc.changeType.String()
		if result != tc.expected {
			t.Errorf("ChangeType %d: expected '%s', got '%s'", tc.changeType, tc.expected, result)
		}
	}
}
