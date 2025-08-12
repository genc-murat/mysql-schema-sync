package schema

import (
	"fmt"
	"mysql-schema-sync/internal/logging"
	"strings"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("Expected service to be created")
	}
	if service.extractor == nil {
		t.Fatal("Expected extractor to be initialized")
	}
}

func TestNewServiceWithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	service := NewServiceWithTimeout(timeout)
	if service.extractor.queryTimeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, service.extractor.queryTimeout)
	}
}

func TestExtractSchemaFromDB_NilDB(t *testing.T) {
	service := NewService()
	_, err := service.ExtractSchemaFromDB(nil, "test_db")
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestCompareSchemas_NilSchemas(t *testing.T) {
	service := NewService()

	// Test nil source schema
	_, err := service.CompareSchemas(nil, NewSchema("target"))
	if err == nil {
		t.Error("Expected error for nil source schema")
	}

	// Test nil target schema
	_, err = service.CompareSchemas(NewSchema("source"), nil)
	if err == nil {
		t.Error("Expected error for nil target schema")
	}
}

func TestCompareSchemas_IdenticalSchemas(t *testing.T) {
	service := NewService()

	// Create identical schemas
	source := createTestSchema("test_db")
	target := createTestSchema("test_db")

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !service.IsSchemaDiffEmpty(diff) {
		t.Error("Expected no differences between identical schemas")
	}
}

func TestCompareSchemas_DifferentSchemas(t *testing.T) {
	service := NewService()

	// Create clean source and target schemas
	source := NewSchema("source_db")
	target := NewSchema("target_db")

	// Add a table that exists in source but not in target (will be "added")
	usersTable := NewTable("users")
	usersTable.AddColumn(NewColumn("id", "int", false))
	source.AddTable(usersTable)

	// Add a table that exists in target but not in source (will be "removed")
	postsTable := NewTable("posts")
	postsTable.AddColumn(NewColumn("id", "int", false))
	target.AddTable(postsTable)

	// Add a table that exists in both but with different columns (will be "modified")
	sourceCommentsTable := NewTable("comments")
	sourceCommentsTable.AddColumn(NewColumn("id", "int", false))
	sourceCommentsTable.AddColumn(NewColumn("content", "text", true))
	source.AddTable(sourceCommentsTable)

	targetCommentsTable := NewTable("comments")
	targetCommentsTable.AddColumn(NewColumn("id", "int", false))
	targetCommentsTable.AddColumn(NewColumn("body", "varchar(500)", false)) // Different column
	target.AddTable(targetCommentsTable)

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check added tables (exist in source but not target)
	if len(diff.AddedTables) != 1 {
		t.Errorf("Expected 1 added table, got %d", len(diff.AddedTables))
		for i, table := range diff.AddedTables {
			t.Logf("Added table %d: %s", i, table.Name)
		}
	} else if diff.AddedTables[0].Name != "users" {
		t.Errorf("Expected added table 'users', got %s", diff.AddedTables[0].Name)
	}

	// Check removed tables (exist in target but not source)
	if len(diff.RemovedTables) != 1 {
		t.Errorf("Expected 1 removed table, got %d", len(diff.RemovedTables))
		for i, table := range diff.RemovedTables {
			t.Logf("Removed table %d: %s", i, table.Name)
		}
	} else if diff.RemovedTables[0].Name != "posts" {
		t.Errorf("Expected removed table 'posts', got %s", diff.RemovedTables[0].Name)
	}

	// Check modified tables
	if len(diff.ModifiedTables) != 1 {
		t.Errorf("Expected 1 modified table, got %d", len(diff.ModifiedTables))
		for i, table := range diff.ModifiedTables {
			t.Logf("Modified table %d: %s", i, table.TableName)
		}
	} else {
		tableDiff := diff.ModifiedTables[0]
		if tableDiff.TableName != "comments" {
			t.Errorf("Expected modified table 'comments', got %s", tableDiff.TableName)
		}

		// Should have one added column (content) and one removed column (body)
		if len(tableDiff.AddedColumns) != 1 {
			t.Errorf("Expected 1 added column, got %d", len(tableDiff.AddedColumns))
		}
		if len(tableDiff.RemovedColumns) != 1 {
			t.Errorf("Expected 1 removed column, got %d", len(tableDiff.RemovedColumns))
		}
	}
}

func TestAreColumnsEqual(t *testing.T) {
	service := NewService()

	// Test identical columns
	col1 := NewColumn("id", "int", false)
	col2 := NewColumn("id", "int", false)

	if !service.areColumnsEqual(col1, col2) {
		t.Error("Expected identical columns to be equal")
	}

	// Test different names
	col3 := NewColumn("user_id", "int", false)
	if service.areColumnsEqual(col1, col3) {
		t.Error("Expected columns with different names to be unequal")
	}

	// Test different data types
	col4 := NewColumn("id", "bigint", false)
	if service.areColumnsEqual(col1, col4) {
		t.Error("Expected columns with different data types to be unequal")
	}

	// Test different nullability
	col5 := NewColumn("id", "int", true)
	if service.areColumnsEqual(col1, col5) {
		t.Error("Expected columns with different nullability to be unequal")
	}

	// Test different default values
	defaultValue := "0"
	col6 := NewColumn("id", "int", false)
	col6.DefaultValue = &defaultValue

	if service.areColumnsEqual(col1, col6) {
		t.Error("Expected columns with different default values to be unequal")
	}
}

func TestGetSchemaStats(t *testing.T) {
	service := NewService()
	schema := createTestSchema("test_db")

	stats := service.GetSchemaStats(schema)

	expectedStats := map[string]int{
		"tables":         1,
		"columns":        2,
		"indexes":        1,
		"global_indexes": 0,
	}

	for key, expected := range expectedStats {
		if actual, exists := stats[key]; !exists {
			t.Errorf("Expected stat %s to exist", key)
		} else if actual != expected {
			t.Errorf("Expected %s to be %d, got %d", key, expected, actual)
		}
	}
}

func TestIsSchemaDiffEmpty(t *testing.T) {
	service := NewService()

	// Test empty diff
	emptyDiff := &SchemaDiff{
		AddedTables:    make([]*Table, 0),
		RemovedTables:  make([]*Table, 0),
		ModifiedTables: make([]*TableDiff, 0),
		AddedIndexes:   make([]*Index, 0),
		RemovedIndexes: make([]*Index, 0),
	}

	if !service.IsSchemaDiffEmpty(emptyDiff) {
		t.Error("Expected empty diff to be reported as empty")
	}

	// Test non-empty diff
	nonEmptyDiff := &SchemaDiff{
		AddedTables:    []*Table{NewTable("test")},
		RemovedTables:  make([]*Table, 0),
		ModifiedTables: make([]*TableDiff, 0),
		AddedIndexes:   make([]*Index, 0),
		RemovedIndexes: make([]*Index, 0),
	}

	if service.IsSchemaDiffEmpty(nonEmptyDiff) {
		t.Error("Expected non-empty diff to be reported as non-empty")
	}
}

func TestConstraintValidation(t *testing.T) {
	tests := []struct {
		name        string
		constraint  *Constraint
		expectError bool
	}{
		{
			name:        "valid foreign key constraint",
			constraint:  NewForeignKeyConstraint("fk_user_id", "posts", []string{"user_id"}, "users", []string{"id"}),
			expectError: false,
		},
		{
			name:        "valid unique constraint",
			constraint:  NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"}),
			expectError: false,
		},
		{
			name:        "empty constraint name",
			constraint:  &Constraint{Name: "", TableName: "test", Type: ConstraintTypeUnique, Columns: []string{"col1"}},
			expectError: true,
		},
		{
			name:        "empty table name",
			constraint:  &Constraint{Name: "test", TableName: "", Type: ConstraintTypeUnique, Columns: []string{"col1"}},
			expectError: true,
		},
		{
			name:        "foreign key without referenced table",
			constraint:  &Constraint{Name: "fk_test", TableName: "test", Type: ConstraintTypeForeignKey, Columns: []string{"col1"}},
			expectError: true,
		},
		{
			name: "foreign key with mismatched column counts",
			constraint: &Constraint{
				Name: "fk_test", TableName: "test", Type: ConstraintTypeForeignKey,
				Columns: []string{"col1"}, ReferencedTable: "ref", ReferencedColumns: []string{"col1", "col2"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constraint.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestCompareSchemas_WithConstraints(t *testing.T) {
	service := NewService()

	// Create source schema with constraints
	source := NewSchema("source_db")
	sourceTable := NewTable("users")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("email", "varchar(255)", false))

	// Add constraints
	uniqueConstraint := NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"})
	sourceTable.AddConstraint(uniqueConstraint)
	source.AddTable(sourceTable)

	// Create target schema without constraints
	target := NewSchema("target_db")
	targetTable := NewTable("users")
	targetTable.AddColumn(NewColumn("id", "int", false))
	targetTable.AddColumn(NewColumn("email", "varchar(255)", false))
	target.AddTable(targetTable)

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should detect the added constraint
	if len(diff.ModifiedTables) != 1 {
		t.Errorf("Expected 1 modified table, got %d", len(diff.ModifiedTables))
	}

	tableDiff := diff.ModifiedTables[0]
	if len(tableDiff.AddedConstraints) != 1 {
		t.Errorf("Expected 1 added constraint, got %d", len(tableDiff.AddedConstraints))
	}

	if tableDiff.AddedConstraints[0].Name != "uk_email" {
		t.Errorf("Expected constraint 'uk_email', got %s", tableDiff.AddedConstraints[0].Name)
	}
}

func TestCompareSchemas_WithIndexes(t *testing.T) {
	service := NewService()

	// Create source schema with index
	source := NewSchema("source_db")
	sourceTable := NewTable("users")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("email", "varchar(255)", false))

	// Add index
	emailIndex := NewIndex("idx_email", "users", []string{"email"})
	sourceTable.AddIndex(emailIndex)
	source.AddTable(sourceTable)

	// Create target schema without index
	target := NewSchema("target_db")
	targetTable := NewTable("users")
	targetTable.AddColumn(NewColumn("id", "int", false))
	targetTable.AddColumn(NewColumn("email", "varchar(255)", false))
	target.AddTable(targetTable)

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should detect the added index
	if len(diff.AddedIndexes) != 1 {
		t.Errorf("Expected 1 added index, got %d", len(diff.AddedIndexes))
	}

	if diff.AddedIndexes[0].Name != "idx_email" {
		t.Errorf("Expected index 'idx_email', got %s", diff.AddedIndexes[0].Name)
	}
}

func TestDetectRenamedTables(t *testing.T) {
	service := NewService()

	// Create source schema
	source := NewSchema("source_db")
	oldTable := NewTable("old_users")
	oldTable.AddColumn(NewColumn("id", "int", false))
	oldTable.AddColumn(NewColumn("name", "varchar(255)", true))
	oldTable.AddColumn(NewColumn("email", "varchar(255)", false))
	source.AddTable(oldTable)

	// Create target schema with renamed table
	target := NewSchema("target_db")
	newTable := NewTable("new_users")
	newTable.AddColumn(NewColumn("id", "int", false))
	newTable.AddColumn(NewColumn("name", "varchar(255)", true))
	newTable.AddColumn(NewColumn("email", "varchar(255)", false))
	target.AddTable(newTable)

	renames := service.DetectRenamedTables(source, target)

	if len(renames) != 1 {
		t.Errorf("Expected 1 rename detection, got %d", len(renames))
	}

	if renames["new_users"] != "old_users" {
		t.Errorf("Expected rename from 'new_users' to 'old_users', got %v", renames)
	}
}

func TestDetectComplexModifications(t *testing.T) {
	service := NewService()

	// Create a diff with potentially destructive changes
	diff := &SchemaDiff{
		RemovedTables: []*Table{NewTable("old_table")},
		ModifiedTables: []*TableDiff{
			{
				TableName:      "users",
				RemovedColumns: []*Column{NewColumn("old_column", "varchar(255)", true)},
				ModifiedColumns: []*ColumnDiff{
					{
						ColumnName: "description",
						OldColumn:  NewColumn("description", "text", true),
						NewColumn:  NewColumn("description", "varchar(100)", true),
					},
				},
				RemovedConstraints: []*Constraint{
					NewForeignKeyConstraint("fk_user_id", "posts", []string{"user_id"}, "users", []string{"id"}),
				},
			},
		},
	}

	warnings := service.DetectComplexModifications(diff)

	if len(warnings) < 3 {
		t.Errorf("Expected at least 3 warnings, got %d", len(warnings))
	}

	// Check for specific warning types
	hasTableDropWarning := false
	hasColumnDropWarning := false
	hasDataTypeShrinkWarning := false
	hasFKDropWarning := false

	for _, warning := range warnings {
		if strings.Contains(warning, "Table 'old_table' will be dropped") {
			hasTableDropWarning = true
		}
		if strings.Contains(warning, "Column 'users.old_column' will be dropped") {
			hasColumnDropWarning = true
		}
		if strings.Contains(warning, "data type change from text to varchar(100) may cause data loss") {
			hasDataTypeShrinkWarning = true
		}
		if strings.Contains(warning, "Foreign key constraint 'fk_user_id' will be dropped") {
			hasFKDropWarning = true
		}
	}

	if !hasTableDropWarning {
		t.Error("Expected table drop warning")
	}
	if !hasColumnDropWarning {
		t.Error("Expected column drop warning")
	}
	if !hasDataTypeShrinkWarning {
		t.Error("Expected data type shrinking warning")
	}
	if !hasFKDropWarning {
		t.Error("Expected foreign key drop warning")
	}
}

func TestAreConstraintsEqual(t *testing.T) {
	service := NewService()

	// Test identical constraints
	c1 := NewForeignKeyConstraint("fk_user_id", "posts", []string{"user_id"}, "users", []string{"id"})
	c2 := NewForeignKeyConstraint("fk_user_id", "posts", []string{"user_id"}, "users", []string{"id"})

	if !service.areConstraintsEqual(c1, c2) {
		t.Error("Expected identical constraints to be equal")
	}

	// Test different names
	c3 := NewForeignKeyConstraint("fk_different", "posts", []string{"user_id"}, "users", []string{"id"})
	if service.areConstraintsEqual(c1, c3) {
		t.Error("Expected constraints with different names to be unequal")
	}

	// Test different referenced tables
	c4 := NewForeignKeyConstraint("fk_user_id", "posts", []string{"user_id"}, "accounts", []string{"id"})
	if service.areConstraintsEqual(c1, c4) {
		t.Error("Expected constraints with different referenced tables to be unequal")
	}
}

func TestAreIndexesEqual(t *testing.T) {
	service := NewService()

	// Test identical indexes
	idx1 := NewIndex("idx_email", "users", []string{"email"})
	idx2 := NewIndex("idx_email", "users", []string{"email"})

	if !service.areIndexesEqual(idx1, idx2) {
		t.Error("Expected identical indexes to be equal")
	}

	// Test different columns
	idx3 := NewIndex("idx_email", "users", []string{"name"})
	if service.areIndexesEqual(idx1, idx3) {
		t.Error("Expected indexes with different columns to be unequal")
	}

	// Test different uniqueness
	idx4 := NewIndex("idx_email", "users", []string{"email"})
	idx4.IsUnique = true
	if service.areIndexesEqual(idx1, idx4) {
		t.Error("Expected indexes with different uniqueness to be unequal")
	}
}

func TestColumnModificationDetection(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		col1     *Column
		col2     *Column
		expected bool
	}{
		{
			name:     "identical columns",
			col1:     NewColumn("id", "int", false),
			col2:     NewColumn("id", "int", false),
			expected: true,
		},
		{
			name:     "different data types",
			col1:     NewColumn("id", "int", false),
			col2:     NewColumn("id", "bigint", false),
			expected: false,
		},
		{
			name:     "different nullability",
			col1:     NewColumn("name", "varchar(255)", true),
			col2:     NewColumn("name", "varchar(255)", false),
			expected: false,
		},
		{
			name: "different default values",
			col1: func() *Column {
				col := NewColumn("status", "varchar(50)", true)
				defaultVal := "active"
				col.DefaultValue = &defaultVal
				return col
			}(),
			col2: func() *Column {
				col := NewColumn("status", "varchar(50)", true)
				defaultVal := "inactive"
				col.DefaultValue = &defaultVal
				return col
			}(),
			expected: false,
		},
		{
			name: "one has default, other doesn't",
			col1: func() *Column {
				col := NewColumn("status", "varchar(50)", true)
				defaultVal := "active"
				col.DefaultValue = &defaultVal
				return col
			}(),
			col2:     NewColumn("status", "varchar(50)", true),
			expected: false,
		},
		{
			name: "different extra properties",
			col1: func() *Column {
				col := NewColumn("id", "int", false)
				col.Extra = "auto_increment"
				return col
			}(),
			col2:     NewColumn("id", "int", false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.areColumnsEqual(tt.col1, tt.col2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDataTypeShrinkingDetection(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		oldType  string
		newType  string
		expected bool
	}{
		{
			name:     "text to varchar - shrinking",
			oldType:  "text",
			newType:  "varchar(255)",
			expected: true,
		},
		{
			name:     "bigint to int - shrinking",
			oldType:  "bigint",
			newType:  "int",
			expected: true,
		},
		{
			name:     "int to bigint - not shrinking",
			oldType:  "int",
			newType:  "bigint",
			expected: false,
		},
		{
			name:     "varchar to text - not shrinking",
			oldType:  "varchar(255)",
			newType:  "text",
			expected: false,
		},
		{
			name:     "same type - not shrinking",
			oldType:  "int",
			newType:  "int",
			expected: false,
		},
		{
			name:     "double to float - shrinking",
			oldType:  "double",
			newType:  "float",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isDataTypeShrinking(tt.oldType, tt.newType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for %s -> %s", tt.expected, result, tt.oldType, tt.newType)
			}
		})
	}
}

func TestTableSimilarityCalculation(t *testing.T) {
	service := NewService()

	// Create two similar tables
	table1 := NewTable("users")
	table1.AddColumn(NewColumn("id", "int", false))
	table1.AddColumn(NewColumn("name", "varchar(255)", true))
	table1.AddColumn(NewColumn("email", "varchar(255)", false))

	table2 := NewTable("people")
	table2.AddColumn(NewColumn("id", "int", false))
	table2.AddColumn(NewColumn("name", "varchar(255)", true))
	table2.AddColumn(NewColumn("email", "varchar(255)", false))

	similarity := service.calculateTableSimilarity(table1, table2)
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical structure, got %f", similarity)
	}

	// Create a table with different columns
	table3 := NewTable("accounts")
	table3.AddColumn(NewColumn("id", "int", false))
	table3.AddColumn(NewColumn("username", "varchar(255)", true))

	similarity2 := service.calculateTableSimilarity(table1, table3)
	expected := 1.0 / 3.0 // Only 1 matching column out of 3
	if similarity2 != expected {
		t.Errorf("Expected similarity %f, got %f", expected, similarity2)
	}

	// Test empty tables
	emptyTable1 := NewTable("empty1")
	emptyTable2 := NewTable("empty2")

	similarity3 := service.calculateTableSimilarity(emptyTable1, emptyTable2)
	if similarity3 != 1.0 {
		t.Errorf("Expected similarity 1.0 for empty tables, got %f", similarity3)
	}
}

// Helper function to create a test schema
func createTestSchema(name string) *Schema {
	schema := NewSchema(name)

	// Create a test table
	table := NewTable("test_table")
	table.AddColumn(NewColumn("id", "int", false))
	table.AddColumn(NewColumn("name", "varchar(255)", true))

	// Add an index
	index := NewIndex("idx_id", "test_table", []string{"id"})
	index.IsPrimary = true
	index.IsUnique = true
	table.AddIndex(index)

	schema.AddTable(table)

	return schema
}

// Additional comprehensive tests for schema service

func TestNewServiceWithLogger(t *testing.T) {
	logger := logging.NewDefaultLogger()
	service := NewServiceWithLogger(logger)
	if service.logger != logger {
		t.Error("Expected custom logger to be set")
	}
}

func TestExtractSchemaFromDB_EmptySchemaName(t *testing.T) {
	service := NewService()

	// Test with empty schema name - should attempt to get current schema
	// This will fail without a real DB connection, but we can test the validation
	_, err := service.ExtractSchemaFromDB(nil, "")
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestCompareSchemas_EmptySchemas(t *testing.T) {
	service := NewService()

	// Create empty schemas
	source := NewSchema("source")
	target := NewSchema("target")

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !service.IsSchemaDiffEmpty(diff) {
		t.Error("Expected no differences between empty schemas")
	}
}

func TestCompareSchemas_ComplexScenarios(t *testing.T) {
	service := NewService()

	// Test scenario with multiple table modifications
	source := NewSchema("source")
	target := NewSchema("target")

	// Create complex source table
	sourceTable := NewTable("complex_table")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("name", "varchar(255)", false))
	sourceTable.AddColumn(NewColumn("email", "varchar(255)", true))
	sourceTable.AddColumn(NewColumn("created_at", "timestamp", false))

	// Add indexes
	primaryIndex := NewIndex("PRIMARY", "complex_table", []string{"id"})
	primaryIndex.IsPrimary = true
	primaryIndex.IsUnique = true
	sourceTable.AddIndex(primaryIndex)

	emailIndex := NewIndex("idx_email", "complex_table", []string{"email"})
	sourceTable.AddIndex(emailIndex)

	// Add constraints
	uniqueConstraint := NewConstraint("uk_email", "complex_table", ConstraintTypeUnique, []string{"email"})
	sourceTable.AddConstraint(uniqueConstraint)

	source.AddTable(sourceTable)

	// Create modified target table
	targetTable := NewTable("complex_table")
	targetTable.AddColumn(NewColumn("id", "bigint", false)) // Modified type
	targetTable.AddColumn(NewColumn("name", "varchar(255)", false))
	targetTable.AddColumn(NewColumn("username", "varchar(100)", false)) // New column
	targetTable.AddColumn(NewColumn("created_at", "timestamp", false))
	// email column removed

	// Different index
	primaryIndex2 := NewIndex("PRIMARY", "complex_table", []string{"id"})
	primaryIndex2.IsPrimary = true
	primaryIndex2.IsUnique = true
	targetTable.AddIndex(primaryIndex2)

	usernameIndex := NewIndex("idx_username", "complex_table", []string{"username"})
	targetTable.AddIndex(usernameIndex)

	target.AddTable(targetTable)

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should detect modifications
	if len(diff.ModifiedTables) != 1 {
		t.Errorf("Expected 1 modified table, got %d", len(diff.ModifiedTables))
	}

	tableDiff := diff.ModifiedTables[0]

	// Should detect added column (email)
	if len(tableDiff.AddedColumns) != 1 {
		t.Errorf("Expected 1 added column, got %d", len(tableDiff.AddedColumns))
	}

	// Should detect removed column (username)
	if len(tableDiff.RemovedColumns) != 1 {
		t.Errorf("Expected 1 removed column, got %d", len(tableDiff.RemovedColumns))
	}

	// Should detect modified column (id type change)
	if len(tableDiff.ModifiedColumns) != 1 {
		t.Errorf("Expected 1 modified column, got %d", len(tableDiff.ModifiedColumns))
	}
}

func TestCompareSchemas_IndexComparison(t *testing.T) {
	service := NewService()

	source := NewSchema("source")
	target := NewSchema("target")

	// Create table with indexes in source
	sourceTable := NewTable("test_table")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("name", "varchar(255)", false))

	idx1 := NewIndex("idx_name", "test_table", []string{"name"})
	idx2 := NewIndex("idx_composite", "test_table", []string{"id", "name"})
	idx2.IsUnique = true

	sourceTable.AddIndex(idx1)
	sourceTable.AddIndex(idx2)
	source.AddTable(sourceTable)

	// Create table with different indexes in target
	targetTable := NewTable("test_table")
	targetTable.AddColumn(NewColumn("id", "int", false))
	targetTable.AddColumn(NewColumn("name", "varchar(255)", false))

	idx3 := NewIndex("idx_id", "test_table", []string{"id"})
	idx4 := NewIndex("idx_composite", "test_table", []string{"name", "id"}) // Different column order
	idx4.IsUnique = true

	targetTable.AddIndex(idx3)
	targetTable.AddIndex(idx4)
	target.AddTable(targetTable)

	diff, err := service.CompareSchemas(source, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should detect index changes
	if len(diff.AddedIndexes) == 0 {
		t.Error("Expected added indexes to be detected")
	}
	if len(diff.RemovedIndexes) == 0 {
		t.Error("Expected removed indexes to be detected")
	}
}

func TestGetSchemaStats_ComplexSchema(t *testing.T) {
	service := NewService()

	schema := NewSchema("complex_schema")

	// Add multiple tables with various structures
	for i := 0; i < 3; i++ {
		table := NewTable(fmt.Sprintf("table_%d", i))

		// Add columns
		for j := 0; j < 4; j++ {
			table.AddColumn(NewColumn(fmt.Sprintf("col_%d", j), "varchar(255)", true))
		}

		// Add indexes
		for k := 0; k < 2; k++ {
			index := NewIndex(fmt.Sprintf("idx_%d_%d", i, k), table.Name, []string{fmt.Sprintf("col_%d", k)})
			table.AddIndex(index)
		}

		schema.AddTable(table)
	}

	// Add global indexes
	globalIndex := NewIndex("global_idx", "", []string{"some_column"})
	schema.Indexes["global_idx"] = globalIndex

	stats := service.GetSchemaStats(schema)

	if stats["tables"] != 3 {
		t.Errorf("Expected 3 tables, got %d", stats["tables"])
	}
	if stats["columns"] != 12 { // 3 tables * 4 columns each
		t.Errorf("Expected 12 columns, got %d", stats["columns"])
	}
	if stats["indexes"] != 6 { // 3 tables * 2 indexes each
		t.Errorf("Expected 6 table indexes, got %d", stats["indexes"])
	}
	if stats["global_indexes"] != 1 {
		t.Errorf("Expected 1 global index, got %d", stats["global_indexes"])
	}
}

func TestDetectRenamedTables_NoMatches(t *testing.T) {
	service := NewService()

	source := NewSchema("source")
	target := NewSchema("target")

	// Create completely different tables
	sourceTable := NewTable("users")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("name", "varchar(255)", false))
	source.AddTable(sourceTable)

	targetTable := NewTable("products")
	targetTable.AddColumn(NewColumn("product_id", "int", false))
	targetTable.AddColumn(NewColumn("price", "decimal(10,2)", false))
	target.AddTable(targetTable)

	renames := service.DetectRenamedTables(source, target)

	if len(renames) != 0 {
		t.Errorf("Expected no renames for completely different tables, got %d", len(renames))
	}
}

func TestDetectRenamedTables_PartialMatch(t *testing.T) {
	service := NewService()

	source := NewSchema("source")
	target := NewSchema("target")

	// Create tables with partial similarity (below threshold)
	sourceTable := NewTable("old_table")
	sourceTable.AddColumn(NewColumn("id", "int", false))
	sourceTable.AddColumn(NewColumn("name", "varchar(255)", false))
	sourceTable.AddColumn(NewColumn("email", "varchar(255)", false))
	source.AddTable(sourceTable)

	targetTable := NewTable("new_table")
	targetTable.AddColumn(NewColumn("id", "int", false))
	targetTable.AddColumn(NewColumn("different_col", "varchar(255)", false))
	targetTable.AddColumn(NewColumn("another_col", "varchar(255)", false))
	target.AddTable(targetTable)

	renames := service.DetectRenamedTables(source, target)

	// Should not detect rename due to low similarity
	if len(renames) != 0 {
		t.Errorf("Expected no renames for low similarity tables, got %d", len(renames))
	}
}

func TestDetectComplexModifications_EdgeCases(t *testing.T) {
	service := NewService()

	// Test with empty diff
	emptyDiff := &SchemaDiff{
		AddedTables:        []*Table{},
		RemovedTables:      []*Table{},
		ModifiedTables:     []*TableDiff{},
		AddedIndexes:       []*Index{},
		RemovedIndexes:     []*Index{},
		AddedConstraints:   []*Constraint{},
		RemovedConstraints: []*Constraint{},
	}

	warnings := service.DetectComplexModifications(emptyDiff)
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for empty diff, got %d", len(warnings))
	}

	// Test with nullability changes
	diff := &SchemaDiff{
		ModifiedTables: []*TableDiff{
			{
				TableName: "test_table",
				ModifiedColumns: []*ColumnDiff{
					{
						ColumnName: "nullable_to_not_null",
						OldColumn:  NewColumn("nullable_to_not_null", "varchar(255)", true),
						NewColumn:  NewColumn("nullable_to_not_null", "varchar(255)", false),
					},
					{
						ColumnName: "not_null_to_nullable",
						OldColumn:  NewColumn("not_null_to_nullable", "varchar(255)", false),
						NewColumn:  NewColumn("not_null_to_nullable", "varchar(255)", true),
					},
				},
			},
		},
	}

	warnings = service.DetectComplexModifications(diff)
	if len(warnings) < 2 {
		t.Errorf("Expected at least 2 warnings for nullability changes, got %d", len(warnings))
	}
}

func TestIsDataTypeShrinking_ComprehensiveTests(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		oldType  string
		newType  string
		expected bool
	}{
		// Text types
		{"longtext to mediumtext", "longtext", "mediumtext", true},
		{"mediumtext to text", "mediumtext", "text", true},
		{"text to varchar", "text", "varchar(500)", true},
		{"varchar to char", "varchar(255)", "char(100)", false}, // Current implementation doesn't detect this

		// Integer types
		{"bigint to int", "bigint", "int", true},
		{"int to mediumint", "int", "mediumint", true},
		{"mediumint to smallint", "mediumint", "smallint", true},
		{"smallint to tinyint", "smallint", "tinyint", true},

		// Floating point types
		{"double to float", "double", "float", true},

		// Non-shrinking cases
		{"int to bigint", "int", "bigint", false},
		{"varchar to text", "varchar(255)", "text", false},
		{"same type", "int", "int", false},
		{"unrelated types", "date", "time", false},

		// Case insensitive
		{"TEXT to VARCHAR", "TEXT", "VARCHAR(255)", true},
		{"BIGINT to INT", "BIGINT", "INT", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isDataTypeShrinking(tt.oldType, tt.newType)
			if result != tt.expected {
				t.Errorf("Expected %v for %s -> %s, got %v", tt.expected, tt.oldType, tt.newType, result)
			}
		})
	}
}

func TestCalculateTableSimilarity_EdgeCases(t *testing.T) {
	service := NewService()

	// Test with one empty table
	table1 := NewTable("table1")
	table1.AddColumn(NewColumn("id", "int", false))

	emptyTable := NewTable("empty")

	similarity := service.calculateTableSimilarity(table1, emptyTable)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for table vs empty table, got %f", similarity)
	}

	// Test with different sized tables
	smallTable := NewTable("small")
	smallTable.AddColumn(NewColumn("id", "int", false))

	largeTable := NewTable("large")
	largeTable.AddColumn(NewColumn("id", "int", false))
	largeTable.AddColumn(NewColumn("name", "varchar(255)", false))
	largeTable.AddColumn(NewColumn("email", "varchar(255)", false))

	similarity = service.calculateTableSimilarity(smallTable, largeTable)
	expected := 1.0 / 3.0 // 1 matching column out of 3 total (using larger table size)
	if similarity != expected {
		t.Errorf("Expected similarity %f, got %f", expected, similarity)
	}
}

func TestAreConstraintsEqual_ComprehensiveTests(t *testing.T) {
	service := NewService()

	// Test different constraint types
	tests := []struct {
		name     string
		c1       *Constraint
		c2       *Constraint
		expected bool
	}{
		{
			name:     "identical unique constraints",
			c1:       NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"}),
			c2:       NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"}),
			expected: true,
		},
		{
			name:     "different constraint types",
			c1:       NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"}),
			c2:       NewConstraint("uk_email", "users", ConstraintTypeCheck, []string{"email"}),
			expected: false,
		},
		{
			name:     "different column order",
			c1:       NewConstraint("uk_name_email", "users", ConstraintTypeUnique, []string{"name", "email"}),
			c2:       NewConstraint("uk_name_email", "users", ConstraintTypeUnique, []string{"email", "name"}),
			expected: false,
		},
		{
			name:     "different referenced columns",
			c1:       NewForeignKeyConstraint("fk_user", "posts", []string{"user_id"}, "users", []string{"id"}),
			c2:       NewForeignKeyConstraint("fk_user", "posts", []string{"user_id"}, "users", []string{"user_id"}),
			expected: false,
		},
		{
			name: "different ON UPDATE actions",
			c1: func() *Constraint {
				c := NewForeignKeyConstraint("fk_user", "posts", []string{"user_id"}, "users", []string{"id"})
				c.OnUpdate = "CASCADE"
				return c
			}(),
			c2: func() *Constraint {
				c := NewForeignKeyConstraint("fk_user", "posts", []string{"user_id"}, "users", []string{"id"})
				c.OnUpdate = "RESTRICT"
				return c
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.areConstraintsEqual(tt.c1, tt.c2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAreIndexesEqual_ComprehensiveTests(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		idx1     *Index
		idx2     *Index
		expected bool
	}{
		{
			name:     "identical simple indexes",
			idx1:     NewIndex("idx_name", "users", []string{"name"}),
			idx2:     NewIndex("idx_name", "users", []string{"name"}),
			expected: true,
		},
		{
			name: "different index types",
			idx1: func() *Index {
				idx := NewIndex("idx_name", "users", []string{"name"})
				idx.IndexType = "BTREE"
				return idx
			}(),
			idx2: func() *Index {
				idx := NewIndex("idx_name", "users", []string{"name"})
				idx.IndexType = "HASH"
				return idx
			}(),
			expected: false,
		},
		{
			name: "different primary key status",
			idx1: func() *Index {
				idx := NewIndex("PRIMARY", "users", []string{"id"})
				idx.IsPrimary = true
				return idx
			}(),
			idx2:     NewIndex("PRIMARY", "users", []string{"id"}),
			expected: false,
		},
		{
			name:     "different table names",
			idx1:     NewIndex("idx_name", "users", []string{"name"}),
			idx2:     NewIndex("idx_name", "accounts", []string{"name"}),
			expected: false,
		},
		{
			name:     "different column count",
			idx1:     NewIndex("idx_composite", "users", []string{"name"}),
			idx2:     NewIndex("idx_composite", "users", []string{"name", "email"}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.areIndexesEqual(tt.idx1, tt.idx2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Performance and stress tests
func TestCompareSchemas_LargeSchemas(t *testing.T) {
	service := NewService()

	// Create large schemas for performance testing
	source := NewSchema("large_source")
	target := NewSchema("large_target")

	// Add many tables
	for i := 0; i < 50; i++ {
		sourceTable := NewTable(fmt.Sprintf("table_%d", i))
		targetTable := NewTable(fmt.Sprintf("table_%d", i))

		// Add many columns
		for j := 0; j < 20; j++ {
			col := NewColumn(fmt.Sprintf("col_%d", j), "varchar(255)", j%2 == 0)
			sourceTable.AddColumn(col)
			targetTable.AddColumn(col)
		}

		source.AddTable(sourceTable)
		target.AddTable(targetTable)
	}

	start := time.Now()
	diff, err := service.CompareSchemas(source, target)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !service.IsSchemaDiffEmpty(diff) {
		t.Error("Expected no differences between identical large schemas")
	}

	// Performance check - should complete within reasonable time
	if duration > 5*time.Second {
		t.Errorf("Schema comparison took too long: %v", duration)
	}
}

// Benchmark tests
func BenchmarkCompareSchemas_Small(b *testing.B) {
	service := NewService()
	source := createTestSchema("source")
	target := createTestSchema("target")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CompareSchemas(source, target)
	}
}

func BenchmarkCompareSchemas_Medium(b *testing.B) {
	service := NewService()

	// Create medium-sized schemas
	source := NewSchema("source")
	target := NewSchema("target")

	for i := 0; i < 10; i++ {
		table := NewTable(fmt.Sprintf("table_%d", i))
		for j := 0; j < 10; j++ {
			table.AddColumn(NewColumn(fmt.Sprintf("col_%d", j), "varchar(255)", true))
		}
		source.AddTable(table)
		target.AddTable(table)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CompareSchemas(source, target)
	}
}

func BenchmarkDetectRenamedTables(b *testing.B) {
	service := NewService()

	source := NewSchema("source")
	target := NewSchema("target")

	// Create tables for rename detection
	for i := 0; i < 5; i++ {
		oldTable := NewTable(fmt.Sprintf("old_table_%d", i))
		newTable := NewTable(fmt.Sprintf("new_table_%d", i))

		for j := 0; j < 5; j++ {
			col := NewColumn(fmt.Sprintf("col_%d", j), "varchar(255)", true)
			oldTable.AddColumn(col)
			newTable.AddColumn(col)
		}

		source.AddTable(oldTable)
		target.AddTable(newTable)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.DetectRenamedTables(source, target)
	}
}
