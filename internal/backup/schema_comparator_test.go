package backup

import (
	"testing"

	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

func TestNewSchemaComparator(t *testing.T) {
	sc := NewSchemaComparator()
	if sc == nil {
		t.Fatal("NewSchemaComparator returned nil")
	}
}

func TestCompareSchemas(t *testing.T) {
	sc := NewSchemaComparator()

	// Create test schemas
	current := createComparatorTestSchema("test_db")
	target := createComparatorTestSchema("test_db")

	// Add a table only in current
	currentOnlyTable := schema.NewTable("current_only")
	currentOnlyTable.AddColumn(&schema.Column{Name: "id", DataType: "INT", Position: 1})
	current.AddTable(currentOnlyTable)

	// Add a table only in target
	targetOnlyTable := schema.NewTable("target_only")
	targetOnlyTable.AddColumn(&schema.Column{Name: "name", DataType: "VARCHAR(255)", Position: 1})
	target.AddTable(targetOnlyTable)

	// Add a table in both with modifications
	sharedTable := schema.NewTable("shared_table")
	sharedTable.AddColumn(&schema.Column{Name: "id", DataType: "INT", Position: 1})
	sharedTable.AddColumn(&schema.Column{Name: "current_col", DataType: "VARCHAR(100)", Position: 2})
	current.AddTable(sharedTable)

	sharedTableTarget := schema.NewTable("shared_table")
	sharedTableTarget.AddColumn(&schema.Column{Name: "id", DataType: "INT", Position: 1})
	sharedTableTarget.AddColumn(&schema.Column{Name: "target_col", DataType: "VARCHAR(200)", Position: 2})
	target.AddTable(sharedTableTarget)

	// Compare schemas
	diff, err := sc.CompareSchemas(current, target)
	if err != nil {
		t.Fatalf("CompareSchemas failed: %v", err)
	}

	// Verify removed tables
	if len(diff.RemovedTables) != 1 {
		t.Errorf("Expected 1 removed table, got %d", len(diff.RemovedTables))
	} else if diff.RemovedTables[0].Name != "current_only" {
		t.Errorf("Expected removed table 'current_only', got %s", diff.RemovedTables[0].Name)
	}

	// Verify added tables
	if len(diff.AddedTables) != 1 {
		t.Errorf("Expected 1 added table, got %d", len(diff.AddedTables))
	} else if diff.AddedTables[0].Name != "target_only" {
		t.Errorf("Expected added table 'target_only', got %s", diff.AddedTables[0].Name)
	}

	// Verify modified tables
	if len(diff.ModifiedTables) != 1 {
		t.Errorf("Expected 1 modified table, got %d", len(diff.ModifiedTables))
	} else {
		tableDiff := diff.ModifiedTables[0]
		if tableDiff.TableName != "shared_table" {
			t.Errorf("Expected modified table 'shared_table', got %s", tableDiff.TableName)
		}
		if len(tableDiff.RemovedColumns) != 1 || tableDiff.RemovedColumns[0].Name != "current_col" {
			t.Errorf("Expected removed column 'current_col'")
		}
		if len(tableDiff.AddedColumns) != 1 || tableDiff.AddedColumns[0].Name != "target_col" {
			t.Errorf("Expected added column 'target_col'")
		}
	}
}

func TestCompareSchemasNilInput(t *testing.T) {
	sc := NewSchemaComparator()

	_, err := sc.CompareSchemas(nil, createComparatorTestSchema("test"))
	if err == nil {
		t.Error("Expected error for nil current schema")
	}

	_, err = sc.CompareSchemas(createComparatorTestSchema("test"), nil)
	if err == nil {
		t.Error("Expected error for nil target schema")
	}
}

func TestSchemaComparatorGenerateRollbackStatements(t *testing.T) {
	sc := NewSchemaComparator()

	// Create test schemas
	current := createComparatorTestSchema("test_db")
	target := createComparatorTestSchema("test_db")

	// Add a table only in current (should be dropped)
	currentOnlyTable := schema.NewTable("to_drop")
	currentOnlyTable.AddColumn(&schema.Column{Name: "id", DataType: "INT", Position: 1})
	current.AddTable(currentOnlyTable)

	// Add a table only in target (should be created)
	targetOnlyTable := schema.NewTable("to_create")
	targetOnlyTable.AddColumn(&schema.Column{Name: "name", DataType: "VARCHAR(255)", Position: 1})
	target.AddTable(targetOnlyTable)

	// Generate rollback statements
	statements, warnings, err := sc.GenerateRollbackStatements(current, target)
	if err != nil {
		t.Fatalf("GenerateRollbackStatements failed: %v", err)
	}

	// Should have statements for both operations
	if len(statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(statements))
	}

	// Should have warnings for destructive operations
	if len(warnings) == 0 {
		t.Error("Expected warnings for destructive operations")
	}

	// Verify statement types
	foundDrop := false
	foundCreate := false
	for _, stmt := range statements {
		if stmt.Type == migration.StatementTypeDropTable {
			foundDrop = true
			if stmt.TableName != "to_drop" {
				t.Errorf("Expected drop statement for 'to_drop', got %s", stmt.TableName)
			}
			if !stmt.IsDestructive {
				t.Error("Drop statement should be marked as destructive")
			}
		}
		if stmt.Type == migration.StatementTypeCreateTable {
			foundCreate = true
			if stmt.TableName != "to_create" {
				t.Errorf("Expected create statement for 'to_create', got %s", stmt.TableName)
			}
			if stmt.IsDestructive {
				t.Error("Create statement should not be marked as destructive")
			}
		}
	}

	if !foundDrop {
		t.Error("Expected DROP TABLE statement")
	}
	if !foundCreate {
		t.Error("Expected CREATE TABLE statement")
	}
}

func TestGenerateCreateTableSQL(t *testing.T) {
	sc := NewSchemaComparator()

	// Create a test table
	table := schema.NewTable("test_table")
	table.AddColumn(&schema.Column{
		Name:       "id",
		DataType:   "INT",
		IsNullable: false,
		Extra:      "AUTO_INCREMENT",
		Position:   1,
	})

	defaultValue := "'default'"
	table.AddColumn(&schema.Column{
		Name:         "name",
		DataType:     "VARCHAR(255)",
		IsNullable:   true,
		DefaultValue: &defaultValue,
		Position:     2,
	})

	// Add primary key
	primaryKey := schema.NewIndex("PRIMARY", "test_table", []string{"id"})
	primaryKey.IsPrimary = true
	table.AddIndex(primaryKey)

	// Generate CREATE TABLE SQL
	sql, err := sc.generateCreateTableSQL(table)
	if err != nil {
		t.Fatalf("generateCreateTableSQL failed: %v", err)
	}

	// Verify SQL contains expected elements
	expectedElements := []string{
		"CREATE TABLE `test_table`",
		"`id` INT NOT NULL AUTO_INCREMENT",
		"`name` VARCHAR(255) DEFAULT 'default'",
		"PRIMARY KEY (`id`)",
	}

	for _, element := range expectedElements {
		if !contains(sql, element) {
			t.Errorf("Expected SQL to contain '%s', got: %s", element, sql)
		}
	}
}

func TestGenerateColumnDefinition(t *testing.T) {
	sc := NewSchemaComparator()

	// Test basic column
	col := &schema.Column{
		Name:       "test_col",
		DataType:   "VARCHAR(100)",
		IsNullable: true,
	}

	def := sc.generateColumnDefinition(col)
	expected := "`test_col` VARCHAR(100)"
	if def != expected {
		t.Errorf("Expected '%s', got '%s'", expected, def)
	}

	// Test NOT NULL column with default
	defaultValue := "'test'"
	col = &schema.Column{
		Name:         "test_col",
		DataType:     "VARCHAR(100)",
		IsNullable:   false,
		DefaultValue: &defaultValue,
		Extra:        "AUTO_INCREMENT",
	}

	def = sc.generateColumnDefinition(col)
	expected = "`test_col` VARCHAR(100) NOT NULL DEFAULT 'test' AUTO_INCREMENT"
	if def != expected {
		t.Errorf("Expected '%s', got '%s'", expected, def)
	}
}

func TestColumnsAreDifferent(t *testing.T) {
	sc := NewSchemaComparator()

	// Same columns
	col1 := &schema.Column{Name: "test", DataType: "INT", IsNullable: false}
	col2 := &schema.Column{Name: "test", DataType: "INT", IsNullable: false}

	if sc.columnsAreDifferent(col1, col2) {
		t.Error("Identical columns should not be different")
	}

	// Different data types
	col2.DataType = "VARCHAR(255)"
	if !sc.columnsAreDifferent(col1, col2) {
		t.Error("Columns with different data types should be different")
	}

	// Different nullability
	col2.DataType = "INT"
	col2.IsNullable = true
	if !sc.columnsAreDifferent(col1, col2) {
		t.Error("Columns with different nullability should be different")
	}

	// Different default values
	col2.IsNullable = false
	defaultValue1 := "'default1'"
	defaultValue2 := "'default2'"
	col1.DefaultValue = &defaultValue1
	col2.DefaultValue = &defaultValue2
	if !sc.columnsAreDifferent(col1, col2) {
		t.Error("Columns with different default values should be different")
	}

	// One has default, other doesn't
	col1.DefaultValue = &defaultValue1
	col2.DefaultValue = nil
	if !sc.columnsAreDifferent(col1, col2) {
		t.Error("Columns with different default value presence should be different")
	}
}

func TestGenerateCreateIndexSQL(t *testing.T) {
	sc := NewSchemaComparator()

	// Regular index
	index := &schema.Index{
		Name:      "idx_name",
		TableName: "test_table",
		Columns:   []string{"name", "email"},
		IsUnique:  false,
	}

	sql := sc.generateCreateIndexSQL(index)
	expected := "CREATE INDEX `idx_name` ON `test_table` (`name`, `email`)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}

	// Unique index
	index.IsUnique = true
	sql = sc.generateCreateIndexSQL(index)
	expected = "CREATE UNIQUE INDEX `idx_name` ON `test_table` (`name`, `email`)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
}

func TestGenerateAddConstraintSQL(t *testing.T) {
	sc := NewSchemaComparator()

	// Foreign key constraint
	fkConstraint := &schema.Constraint{
		Name:              "fk_user_id",
		TableName:         "orders",
		Type:              schema.ConstraintTypeForeignKey,
		Columns:           []string{"user_id"},
		ReferencedTable:   "users",
		ReferencedColumns: []string{"id"},
		OnUpdate:          "CASCADE",
		OnDelete:          "RESTRICT",
	}

	sql := sc.generateAddConstraintSQL(fkConstraint)
	expected := "ALTER TABLE `orders` ADD CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}

	// Unique constraint
	uniqueConstraint := &schema.Constraint{
		Name:      "uk_email",
		TableName: "users",
		Type:      schema.ConstraintTypeUnique,
		Columns:   []string{"email"},
	}

	sql = sc.generateAddConstraintSQL(uniqueConstraint)
	expected = "ALTER TABLE `users` ADD CONSTRAINT `uk_email` UNIQUE (`email`)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}

	// Check constraint
	checkConstraint := &schema.Constraint{
		Name:            "chk_age",
		TableName:       "users",
		Type:            schema.ConstraintTypeCheck,
		Columns:         []string{"age"},
		CheckExpression: "age >= 0",
	}

	sql = sc.generateAddConstraintSQL(checkConstraint)
	expected = "ALTER TABLE `users` ADD CONSTRAINT `chk_age` CHECK (age >= 0)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
}

func TestHasTableChanges(t *testing.T) {
	sc := NewSchemaComparator()

	// Empty diff
	diff := &schema.TableDiff{
		TableName:          "test",
		AddedColumns:       []*schema.Column{},
		RemovedColumns:     []*schema.Column{},
		ModifiedColumns:    []*schema.ColumnDiff{},
		AddedConstraints:   []*schema.Constraint{},
		RemovedConstraints: []*schema.Constraint{},
	}

	if sc.hasTableChanges(diff) {
		t.Error("Empty diff should not have changes")
	}

	// Add a column
	diff.AddedColumns = append(diff.AddedColumns, &schema.Column{Name: "new_col"})
	if !sc.hasTableChanges(diff) {
		t.Error("Diff with added column should have changes")
	}
}

// Helper function to create a test schema
func createComparatorTestSchema(name string) *schema.Schema {
	return &schema.Schema{
		Name:    name,
		Tables:  make(map[string]*schema.Table),
		Indexes: make(map[string]*schema.Index),
	}
}
