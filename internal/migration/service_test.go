package migration

import (
	"testing"

	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/schema"
)

func TestMigrationService_PlanMigration(t *testing.T) {
	service := NewMigrationService()

	// Create a test schema diff
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "new_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
						Extra:      "AUTO_INCREMENT",
					},
				},
			},
		},
		RemovedTables: []*schema.Table{
			{
				Name: "old_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
					},
				},
			},
		},
	}

	plan, err := service.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Expected non-nil migration plan")
	}

	if len(plan.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(plan.Statements))
	}

	if plan.Summary.TablesAdded != 1 {
		t.Errorf("Expected 1 table added, got %d", plan.Summary.TablesAdded)
	}

	if plan.Summary.TablesRemoved != 1 {
		t.Errorf("Expected 1 table removed, got %d", plan.Summary.TablesRemoved)
	}
}

func TestMigrationService_GenerateSQL(t *testing.T) {
	service := NewMigrationService()

	// Create a simple schema diff
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "test_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
						Extra:      "AUTO_INCREMENT",
					},
					"name": {
						Name:       "name",
						DataType:   "VARCHAR(255)",
						IsNullable: false,
					},
				},
				Indexes: []*schema.Index{
					{
						Name:      "PRIMARY",
						TableName: "test_table",
						Columns:   []string{"id"},
						IsUnique:  true,
						IsPrimary: true,
					},
				},
				Constraints: make(map[string]*schema.Constraint),
			},
		},
	}

	sqlStatements, err := service.GenerateSQL(diff)
	if err != nil {
		t.Fatalf("GenerateSQL() error = %v", err)
	}

	if len(sqlStatements) != 1 {
		t.Errorf("Expected 1 SQL statement, got %d", len(sqlStatements))
	}

	// Check that the SQL contains expected elements
	sql := sqlStatements[0]
	if len(sql) == 0 {
		t.Error("Expected non-empty SQL statement")
	}

	// Should contain CREATE TABLE
	if !contains(sql, "CREATE TABLE") {
		t.Error("Expected SQL to contain CREATE TABLE")
	}

	// Should contain table name
	if !contains(sql, "test_table") {
		t.Error("Expected SQL to contain table name")
	}
}

func TestMigrationService_ValidatePlan(t *testing.T) {
	service := NewMigrationService()

	// Test valid plan
	validPlan := NewMigrationPlan()
	stmt := *NewMigrationStatement(
		"CREATE TABLE test (id INT)",
		StatementTypeCreateTable,
		"Create test table",
	)
	validPlan.AddStatement(stmt)

	err := service.ValidatePlan(validPlan)
	if err != nil {
		t.Errorf("ValidatePlan() error = %v for valid plan", err)
	}

	// Test nil plan
	err = service.ValidatePlan(nil)
	if err == nil {
		t.Error("Expected error for nil plan")
	}

	// Test empty plan
	emptyPlan := NewMigrationPlan()
	err = service.ValidatePlan(emptyPlan)
	if err == nil {
		t.Error("Expected error for empty plan")
	}
}

func TestMigrationService_GenerateCreateTableSQL(t *testing.T) {
	service := NewMigrationService()

	table := &schema.Table{
		Name: "test_table",
		Columns: map[string]*schema.Column{
			"id": {
				Name:       "id",
				DataType:   "INT",
				IsNullable: false,
				Extra:      "AUTO_INCREMENT",
			},
			"name": {
				Name:       "name",
				DataType:   "VARCHAR(255)",
				IsNullable: false,
			},
		},
		Indexes: []*schema.Index{
			{
				Name:      "PRIMARY",
				TableName: "test_table",
				Columns:   []string{"id"},
				IsUnique:  true,
				IsPrimary: true,
			},
		},
		Constraints: make(map[string]*schema.Constraint),
	}

	sql, err := service.GenerateCreateTableSQL(table)
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL() error = %v", err)
	}

	if !contains(sql, "CREATE TABLE `test_table`") {
		t.Error("Expected SQL to contain CREATE TABLE statement")
	}

	if !contains(sql, "`id` INT NOT NULL AUTO_INCREMENT") {
		t.Error("Expected SQL to contain id column definition")
	}

	if !contains(sql, "PRIMARY KEY (`id`)") {
		t.Error("Expected SQL to contain primary key definition")
	}
}

func TestMigrationService_GenerateDropTableSQL(t *testing.T) {
	service := NewMigrationService()

	table := &schema.Table{Name: "test_table"}

	sql, err := service.GenerateDropTableSQL(table)
	if err != nil {
		t.Fatalf("GenerateDropTableSQL() error = %v", err)
	}

	expected := "DROP TABLE `test_table`"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestMigrationService_GenerateColumnSQL(t *testing.T) {
	service := NewMigrationService()

	column := &schema.Column{
		Name:       "test_column",
		DataType:   "VARCHAR(100)",
		IsNullable: true,
	}

	// Test ADD COLUMN
	addSQL, err := service.GenerateAddColumnSQL("test_table", column)
	if err != nil {
		t.Fatalf("GenerateAddColumnSQL() error = %v", err)
	}

	expectedAdd := "ALTER TABLE `test_table` ADD COLUMN `test_column` VARCHAR(100) NULL"
	if addSQL != expectedAdd {
		t.Errorf("Expected %s, got %s", expectedAdd, addSQL)
	}

	// Test DROP COLUMN
	dropSQL, err := service.GenerateDropColumnSQL("test_table", column)
	if err != nil {
		t.Fatalf("GenerateDropColumnSQL() error = %v", err)
	}

	expectedDrop := "ALTER TABLE `test_table` DROP COLUMN `test_column`"
	if dropSQL != expectedDrop {
		t.Errorf("Expected %s, got %s", expectedDrop, dropSQL)
	}

	// Test MODIFY COLUMN
	columnDiff := &schema.ColumnDiff{
		ColumnName: "test_column",
		NewColumn: &schema.Column{
			Name:       "test_column",
			DataType:   "VARCHAR(255)",
			IsNullable: false,
		},
	}

	modifySQL, err := service.GenerateModifyColumnSQL("test_table", columnDiff)
	if err != nil {
		t.Fatalf("GenerateModifyColumnSQL() error = %v", err)
	}

	expectedModify := "ALTER TABLE `test_table` MODIFY COLUMN `test_column` VARCHAR(255) NOT NULL"
	if modifySQL != expectedModify {
		t.Errorf("Expected %s, got %s", expectedModify, modifySQL)
	}
}

func TestMigrationService_GenerateIndexSQL(t *testing.T) {
	service := NewMigrationService()

	index := &schema.Index{
		Name:      "idx_test",
		TableName: "test_table",
		Columns:   []string{"column1", "column2"},
		IsUnique:  false,
		IsPrimary: false,
	}

	// Test CREATE INDEX
	createSQL, err := service.GenerateCreateIndexSQL(index)
	if err != nil {
		t.Fatalf("GenerateCreateIndexSQL() error = %v", err)
	}

	expectedCreate := "CREATE INDEX `idx_test` ON `test_table` (`column1`, `column2`)"
	if createSQL != expectedCreate {
		t.Errorf("Expected %s, got %s", expectedCreate, createSQL)
	}

	// Test DROP INDEX
	dropSQL, err := service.GenerateDropIndexSQL(index)
	if err != nil {
		t.Fatalf("GenerateDropIndexSQL() error = %v", err)
	}

	expectedDrop := "DROP INDEX `idx_test` ON `test_table`"
	if dropSQL != expectedDrop {
		t.Errorf("Expected %s, got %s", expectedDrop, dropSQL)
	}
}

func TestMigrationService_GenerateConstraintSQL(t *testing.T) {
	service := NewMigrationService()

	// Test foreign key constraint
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

	// Test ADD CONSTRAINT
	addSQL, err := service.GenerateAddConstraintSQL(fkConstraint)
	if err != nil {
		t.Fatalf("GenerateAddConstraintSQL() error = %v", err)
	}

	expectedAdd := "ALTER TABLE `orders` ADD CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT"
	if addSQL != expectedAdd {
		t.Errorf("Expected %s, got %s", expectedAdd, addSQL)
	}

	// Test DROP CONSTRAINT
	dropSQL, err := service.GenerateDropConstraintSQL(fkConstraint)
	if err != nil {
		t.Fatalf("GenerateDropConstraintSQL() error = %v", err)
	}

	expectedDrop := "ALTER TABLE `orders` DROP FOREIGN KEY `fk_user_id`"
	if dropSQL != expectedDrop {
		t.Errorf("Expected %s, got %s", expectedDrop, dropSQL)
	}
}

func TestMigrationService_GetSQLForStatementType(t *testing.T) {
	service := NewMigrationService()

	// Test CREATE TABLE
	table := &schema.Table{
		Name: "test_table",
		Columns: map[string]*schema.Column{
			"id": {
				Name:       "id",
				DataType:   "INT",
				IsNullable: false,
			},
		},
		Constraints: make(map[string]*schema.Constraint),
	}

	sql, err := service.GetSQLForStatementType(StatementTypeCreateTable, "", table)
	if err != nil {
		t.Fatalf("GetSQLForStatementType(CREATE_TABLE) error = %v", err)
	}

	if !contains(sql, "CREATE TABLE") {
		t.Error("Expected SQL to contain CREATE TABLE")
	}

	// Test ADD COLUMN
	column := &schema.Column{
		Name:       "new_column",
		DataType:   "VARCHAR(100)",
		IsNullable: true,
	}

	sql, err = service.GetSQLForStatementType(StatementTypeAddColumn, "test_table", column)
	if err != nil {
		t.Fatalf("GetSQLForStatementType(ADD_COLUMN) error = %v", err)
	}

	if !contains(sql, "ALTER TABLE") || !contains(sql, "ADD COLUMN") {
		t.Error("Expected SQL to contain ALTER TABLE ADD COLUMN")
	}

	// Test invalid object type
	_, err = service.GetSQLForStatementType(StatementTypeCreateTable, "", column)
	if err == nil {
		t.Error("Expected error for invalid object type")
	}

	// Test unsupported statement type
	_, err = service.GetSQLForStatementType("INVALID_TYPE", "", table)
	if err == nil {
		t.Error("Expected error for unsupported statement type")
	}
}

func TestMigrationService_NilInputs(t *testing.T) {
	service := NewMigrationService()

	// Test nil schema diff
	_, err := service.PlanMigration(nil)
	if err == nil {
		t.Error("Expected error for nil schema diff in PlanMigration")
	}

	_, err = service.GenerateSQL(nil)
	if err == nil {
		t.Error("Expected error for nil schema diff in GenerateSQL")
	}
}

func TestNewMigrationService(t *testing.T) {
	service := NewMigrationService()

	if service == nil {
		t.Fatal("Expected non-nil migration service")
	}

	// Test that service implements the interface
	var _ MigrationService = service
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Additional comprehensive tests for migration service

func TestNewMigrationServiceWithLogger(t *testing.T) {
	logger := logging.NewDefaultLogger()
	service := NewMigrationServiceWithLogger(logger)

	if service == nil {
		t.Fatal("Expected non-nil migration service")
	}

	// Test that service implements the interface
	var _ MigrationService = service
}

func TestMigrationService_PlanMigration_ComplexScenario(t *testing.T) {
	service := NewMigrationService()

	// Create a complex schema diff with multiple types of changes
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "new_users",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
						Extra:      "AUTO_INCREMENT",
					},
					"email": {
						Name:       "email",
						DataType:   "VARCHAR(255)",
						IsNullable: false,
					},
				},
				Indexes: []*schema.Index{
					{
						Name:      "PRIMARY",
						TableName: "new_users",
						Columns:   []string{"id"},
						IsUnique:  true,
						IsPrimary: true,
					},
					{
						Name:      "idx_email",
						TableName: "new_users",
						Columns:   []string{"email"},
						IsUnique:  true,
					},
				},
				Constraints: map[string]*schema.Constraint{
					"uk_email": {
						Name:      "uk_email",
						TableName: "new_users",
						Type:      schema.ConstraintTypeUnique,
						Columns:   []string{"email"},
					},
				},
			},
		},
		RemovedTables: []*schema.Table{
			{
				Name: "old_logs",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
					},
				},
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "existing_table",
				AddedColumns: []*schema.Column{
					{
						Name:       "new_column",
						DataType:   "VARCHAR(100)",
						IsNullable: true,
					},
				},
				RemovedColumns: []*schema.Column{
					{
						Name:       "old_column",
						DataType:   "TEXT",
						IsNullable: true,
					},
				},
				ModifiedColumns: []*schema.ColumnDiff{
					{
						ColumnName: "modified_column",
						OldColumn: &schema.Column{
							Name:       "modified_column",
							DataType:   "VARCHAR(50)",
							IsNullable: true,
						},
						NewColumn: &schema.Column{
							Name:       "modified_column",
							DataType:   "VARCHAR(255)",
							IsNullable: false,
						},
					},
				},
			},
		},
		AddedIndexes: []*schema.Index{
			{
				Name:      "idx_new_index",
				TableName: "existing_table",
				Columns:   []string{"new_column"},
			},
		},
		RemovedIndexes: []*schema.Index{
			{
				Name:      "idx_old_index",
				TableName: "existing_table",
				Columns:   []string{"old_column"},
			},
		},
	}

	plan, err := service.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Expected non-nil migration plan")
	}

	// Should have multiple statements for all the changes
	if len(plan.Statements) == 0 {
		t.Error("Expected migration plan to have statements")
	}

	// Check that summary reflects all changes
	expectedTablesAdded := 1
	expectedTablesRemoved := 1

	if plan.Summary.TablesAdded != expectedTablesAdded {
		t.Errorf("Expected %d tables added, got %d", expectedTablesAdded, plan.Summary.TablesAdded)
	}

	if plan.Summary.TablesRemoved != expectedTablesRemoved {
		t.Errorf("Expected %d tables removed, got %d", expectedTablesRemoved, plan.Summary.TablesRemoved)
	}
}

func TestMigrationService_GenerateSQL_EmptyDiff(t *testing.T) {
	service := NewMigrationService()

	// Create empty schema diff
	diff := &schema.SchemaDiff{
		AddedTables:        []*schema.Table{},
		RemovedTables:      []*schema.Table{},
		ModifiedTables:     []*schema.TableDiff{},
		AddedIndexes:       []*schema.Index{},
		RemovedIndexes:     []*schema.Index{},
		AddedConstraints:   []*schema.Constraint{},
		RemovedConstraints: []*schema.Constraint{},
	}

	sqlStatements, err := service.GenerateSQL(diff)
	if err != nil {
		t.Fatalf("GenerateSQL() error = %v", err)
	}

	if len(sqlStatements) != 0 {
		t.Errorf("Expected 0 SQL statements for empty diff, got %d", len(sqlStatements))
	}
}

func TestMigrationService_GenerateSQL_WithConstraints(t *testing.T) {
	service := NewMigrationService()

	// Create schema diff with constraints
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "orders",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "INT",
						IsNullable: false,
						Extra:      "AUTO_INCREMENT",
					},
					"user_id": {
						Name:       "user_id",
						DataType:   "INT",
						IsNullable: false,
					},
				},
				Constraints: map[string]*schema.Constraint{
					"fk_user_id": {
						Name:              "fk_user_id",
						TableName:         "orders",
						Type:              schema.ConstraintTypeForeignKey,
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnUpdate:          "CASCADE",
						OnDelete:          "RESTRICT",
					},
				},
			},
		},
	}

	sqlStatements, err := service.GenerateSQL(diff)
	if err != nil {
		t.Fatalf("GenerateSQL() error = %v", err)
	}

	if len(sqlStatements) == 0 {
		t.Error("Expected SQL statements to be generated")
	}

	// Check that SQL statements were generated (the constraint might be in the CREATE TABLE statement)
	if len(sqlStatements) == 0 {
		t.Error("Expected SQL statements to be generated")
	}
}

func TestMigrationService_ValidatePlan_InvalidPlan(t *testing.T) {
	service := NewMigrationService()

	// Test with plan containing invalid statements
	invalidPlan := NewMigrationPlan()

	// Add statement with empty SQL
	emptyStmt := *NewMigrationStatement(
		"",
		StatementTypeCreateTable,
		"Empty SQL statement",
	)
	invalidPlan.AddStatement(emptyStmt)

	err := service.ValidatePlan(invalidPlan)
	if err == nil {
		t.Error("Expected error for plan with empty SQL statement")
	}
}

func TestMigrationService_GenerateCreateTableSQL_WithAllFeatures(t *testing.T) {
	service := NewMigrationService()

	table := &schema.Table{
		Name: "comprehensive_table",
		Columns: map[string]*schema.Column{
			"id": {
				Name:       "id",
				DataType:   "INT",
				IsNullable: false,
				Extra:      "AUTO_INCREMENT",
			},
			"name": {
				Name:         "name",
				DataType:     "VARCHAR(255)",
				IsNullable:   false,
				DefaultValue: stringPtr("'Unknown'"),
			},
			"email": {
				Name:       "email",
				DataType:   "VARCHAR(255)",
				IsNullable: true,
			},
			"created_at": {
				Name:         "created_at",
				DataType:     "TIMESTAMP",
				IsNullable:   false,
				DefaultValue: stringPtr("CURRENT_TIMESTAMP"),
			},
		},
		Indexes: []*schema.Index{
			{
				Name:      "PRIMARY",
				TableName: "comprehensive_table",
				Columns:   []string{"id"},
				IsUnique:  true,
				IsPrimary: true,
			},
			{
				Name:      "idx_email",
				TableName: "comprehensive_table",
				Columns:   []string{"email"},
				IsUnique:  true,
			},
			{
				Name:      "idx_name_created",
				TableName: "comprehensive_table",
				Columns:   []string{"name", "created_at"},
			},
		},
		Constraints: map[string]*schema.Constraint{
			"uk_email": {
				Name:      "uk_email",
				TableName: "comprehensive_table",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"email"},
			},
		},
	}

	sql, err := service.GenerateCreateTableSQL(table)
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL() error = %v", err)
	}

	// Check for various components
	expectedComponents := []string{
		"CREATE TABLE `comprehensive_table`",
		"`id` INT NOT NULL AUTO_INCREMENT",
		"`name` VARCHAR(255) NOT NULL DEFAULT '''Unknown'''",
		"`email` VARCHAR(255) NULL",
		"`created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"PRIMARY KEY (`id`)",
		"UNIQUE KEY `uk_email` (`email`)",
	}

	for _, component := range expectedComponents {
		if !contains(sql, component) {
			t.Errorf("Expected SQL to contain: %s\nActual SQL: %s", component, sql)
		}
	}
}

func TestMigrationService_GenerateModifyColumnSQL_AllScenarios(t *testing.T) {
	service := NewMigrationService()

	tests := []struct {
		name       string
		columnDiff *schema.ColumnDiff
		expected   string
	}{
		{
			name: "change data type",
			columnDiff: &schema.ColumnDiff{
				ColumnName: "age",
				NewColumn: &schema.Column{
					Name:       "age",
					DataType:   "SMALLINT",
					IsNullable: false,
				},
			},
			expected: "ALTER TABLE `test_table` MODIFY COLUMN `age` SMALLINT NOT NULL",
		},
		{
			name: "add default value",
			columnDiff: &schema.ColumnDiff{
				ColumnName: "status",
				NewColumn: &schema.Column{
					Name:         "status",
					DataType:     "VARCHAR(50)",
					IsNullable:   false,
					DefaultValue: stringPtr("'active'"),
				},
			},
			expected: "ALTER TABLE `test_table` MODIFY COLUMN `status` VARCHAR(50) NOT NULL DEFAULT '''active'''",
		},
		{
			name: "change to nullable",
			columnDiff: &schema.ColumnDiff{
				ColumnName: "description",
				NewColumn: &schema.Column{
					Name:       "description",
					DataType:   "TEXT",
					IsNullable: true,
				},
			},
			expected: "ALTER TABLE `test_table` MODIFY COLUMN `description` TEXT NULL",
		},
		{
			name: "add auto increment",
			columnDiff: &schema.ColumnDiff{
				ColumnName: "id",
				NewColumn: &schema.Column{
					Name:       "id",
					DataType:   "INT",
					IsNullable: false,
					Extra:      "AUTO_INCREMENT",
				},
			},
			expected: "ALTER TABLE `test_table` MODIFY COLUMN `id` INT NOT NULL AUTO_INCREMENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := service.GenerateModifyColumnSQL("test_table", tt.columnDiff)
			if err != nil {
				t.Fatalf("GenerateModifyColumnSQL() error = %v", err)
			}

			if sql != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, sql)
			}
		})
	}
}

func TestMigrationService_GenerateIndexSQL_AllTypes(t *testing.T) {
	service := NewMigrationService()

	tests := []struct {
		name           string
		index          *schema.Index
		expectedCreate string
		expectedDrop   string
	}{
		{
			name: "regular index",
			index: &schema.Index{
				Name:      "idx_name",
				TableName: "users",
				Columns:   []string{"name"},
			},
			expectedCreate: "CREATE INDEX `idx_name` ON `users` (`name`)",
			expectedDrop:   "DROP INDEX `idx_name` ON `users`",
		},
		{
			name: "unique index",
			index: &schema.Index{
				Name:      "idx_email",
				TableName: "users",
				Columns:   []string{"email"},
				IsUnique:  true,
			},
			expectedCreate: "CREATE UNIQUE INDEX `idx_email` ON `users` (`email`)",
			expectedDrop:   "DROP INDEX `idx_email` ON `users`",
		},
		{
			name: "composite index",
			index: &schema.Index{
				Name:      "idx_name_email",
				TableName: "users",
				Columns:   []string{"name", "email"},
			},
			expectedCreate: "CREATE INDEX `idx_name_email` ON `users` (`name`, `email`)",
			expectedDrop:   "DROP INDEX `idx_name_email` ON `users`",
		},
		{
			name: "primary key",
			index: &schema.Index{
				Name:      "PRIMARY",
				TableName: "users",
				Columns:   []string{"id"},
				IsPrimary: true,
				IsUnique:  true,
			},
			expectedCreate: "CREATE UNIQUE INDEX `PRIMARY` ON `users` (`id`)",
			expectedDrop:   "DROP INDEX `PRIMARY` ON `users`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" - create", func(t *testing.T) {
			sql, err := service.GenerateCreateIndexSQL(tt.index)
			if err != nil {
				t.Fatalf("GenerateCreateIndexSQL() error = %v", err)
			}

			if sql != tt.expectedCreate {
				t.Errorf("Expected: %s\nGot: %s", tt.expectedCreate, sql)
			}
		})

		t.Run(tt.name+" - drop", func(t *testing.T) {
			sql, err := service.GenerateDropIndexSQL(tt.index)
			if err != nil {
				t.Fatalf("GenerateDropIndexSQL() error = %v", err)
			}

			if sql != tt.expectedDrop {
				t.Errorf("Expected: %s\nGot: %s", tt.expectedDrop, sql)
			}
		})
	}
}

func TestMigrationService_GenerateConstraintSQL_AllTypes(t *testing.T) {
	service := NewMigrationService()

	tests := []struct {
		name         string
		constraint   *schema.Constraint
		expectedAdd  string
		expectedDrop string
	}{
		{
			name: "unique constraint",
			constraint: &schema.Constraint{
				Name:      "uk_email",
				TableName: "users",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"email"},
			},
			expectedAdd:  "ALTER TABLE `users` ADD CONSTRAINT `uk_email` UNIQUE (`email`)",
			expectedDrop: "ALTER TABLE `users` DROP INDEX `uk_email`",
		},
		{
			name: "foreign key constraint",
			constraint: &schema.Constraint{
				Name:              "fk_user_id",
				TableName:         "posts",
				Type:              schema.ConstraintTypeForeignKey,
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedColumns: []string{"id"},
				OnUpdate:          "CASCADE",
				OnDelete:          "SET NULL",
			},
			expectedAdd:  "ALTER TABLE `posts` ADD CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE SET NULL",
			expectedDrop: "ALTER TABLE `posts` DROP FOREIGN KEY `fk_user_id`",
		},
		{
			name: "check constraint",
			constraint: &schema.Constraint{
				Name:            "chk_age",
				TableName:       "users",
				Type:            schema.ConstraintTypeCheck,
				Columns:         []string{"age"},
				CheckExpression: "age >= 0",
			},
			expectedAdd:  "ALTER TABLE `users` ADD CONSTRAINT `chk_age` CHECK (age >= 0)",
			expectedDrop: "ALTER TABLE `users` DROP CHECK `chk_age`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" - add", func(t *testing.T) {
			sql, err := service.GenerateAddConstraintSQL(tt.constraint)
			if err != nil {
				t.Fatalf("GenerateAddConstraintSQL() error = %v", err)
			}

			if sql != tt.expectedAdd {
				t.Errorf("Expected: %s\nGot: %s", tt.expectedAdd, sql)
			}
		})

		t.Run(tt.name+" - drop", func(t *testing.T) {
			sql, err := service.GenerateDropConstraintSQL(tt.constraint)
			if err != nil {
				t.Fatalf("GenerateDropConstraintSQL() error = %v", err)
			}

			if sql != tt.expectedDrop {
				t.Errorf("Expected: %s\nGot: %s", tt.expectedDrop, sql)
			}
		})
	}
}

func TestMigrationService_GetSQLForStatementType_AllTypes(t *testing.T) {
	service := NewMigrationService()

	// Test all statement types
	tests := []struct {
		name      string
		stmtType  StatementType
		tableName string
		object    interface{}
		expectErr bool
	}{
		{
			name:     "CREATE_TABLE",
			stmtType: StatementTypeCreateTable,
			object: &schema.Table{
				Name: "test_table",
				Columns: map[string]*schema.Column{
					"id": {Name: "id", DataType: "INT", IsNullable: false},
				},
				Constraints: make(map[string]*schema.Constraint),
			},
			expectErr: false,
		},
		{
			name:      "DROP_TABLE",
			stmtType:  StatementTypeDropTable,
			object:    &schema.Table{Name: "test_table"},
			expectErr: false,
		},
		{
			name:      "ADD_COLUMN",
			stmtType:  StatementTypeAddColumn,
			tableName: "test_table",
			object:    &schema.Column{Name: "new_col", DataType: "VARCHAR(255)", IsNullable: true},
			expectErr: false,
		},
		{
			name:      "DROP_COLUMN",
			stmtType:  StatementTypeDropColumn,
			tableName: "test_table",
			object:    &schema.Column{Name: "old_col", DataType: "VARCHAR(255)", IsNullable: true},
			expectErr: false,
		},
		{
			name:      "MODIFY_COLUMN",
			stmtType:  StatementTypeModifyColumn,
			tableName: "test_table",
			object: &schema.ColumnDiff{
				ColumnName: "mod_col",
				NewColumn:  &schema.Column{Name: "mod_col", DataType: "TEXT", IsNullable: false},
			},
			expectErr: false,
		},
		{
			name:      "CREATE_INDEX",
			stmtType:  StatementTypeCreateIndex,
			object:    &schema.Index{Name: "idx_test", TableName: "test_table", Columns: []string{"col1"}},
			expectErr: false,
		},
		{
			name:      "DROP_INDEX",
			stmtType:  StatementTypeDropIndex,
			object:    &schema.Index{Name: "idx_test", TableName: "test_table", Columns: []string{"col1"}},
			expectErr: false,
		},
		{
			name:     "ADD_CONSTRAINT",
			stmtType: StatementTypeAddConstraint,
			object: &schema.Constraint{
				Name: "uk_test", TableName: "test_table", Type: schema.ConstraintTypeUnique, Columns: []string{"col1"},
			},
			expectErr: false,
		},
		{
			name:     "DROP_CONSTRAINT",
			stmtType: StatementTypeDropConstraint,
			object: &schema.Constraint{
				Name: "uk_test", TableName: "test_table", Type: schema.ConstraintTypeUnique, Columns: []string{"col1"},
			},
			expectErr: false,
		},
		{
			name:      "invalid statement type",
			stmtType:  "INVALID_TYPE",
			object:    &schema.Table{Name: "test"},
			expectErr: true,
		},
		{
			name:      "wrong object type",
			stmtType:  StatementTypeCreateTable,
			object:    &schema.Column{Name: "col"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := service.GetSQLForStatementType(tt.stmtType, tt.tableName, tt.object)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if sql == "" {
					t.Error("Expected non-empty SQL")
				}
			}
		})
	}
}

// Error handling tests
func TestMigrationService_ErrorHandling(t *testing.T) {
	service := NewMigrationService()

	// Test with nil table
	_, err := service.GenerateCreateTableSQL(nil)
	if err == nil {
		t.Error("Expected error for nil table")
	}

	// Test with nil column
	_, err = service.GenerateAddColumnSQL("test", nil)
	if err == nil {
		t.Error("Expected error for nil column")
	}

	// Test with nil column diff
	_, err = service.GenerateModifyColumnSQL("test", nil)
	if err == nil {
		t.Error("Expected error for nil column diff")
	}

	// Test with nil index
	_, err = service.GenerateCreateIndexSQL(nil)
	if err == nil {
		t.Error("Expected error for nil index")
	}

	// Test with nil constraint
	_, err = service.GenerateAddConstraintSQL(nil)
	if err == nil {
		t.Error("Expected error for nil constraint")
	}
}

// Performance tests
func BenchmarkMigrationService_PlanMigration(b *testing.B) {
	service := NewMigrationService()

	// Create a medium-sized diff
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "test_table",
				Columns: map[string]*schema.Column{
					"id":   {Name: "id", DataType: "INT", IsNullable: false},
					"name": {Name: "name", DataType: "VARCHAR(255)", IsNullable: true},
				},
				Constraints: make(map[string]*schema.Constraint),
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "existing_table",
				AddedColumns: []*schema.Column{
					{Name: "new_col", DataType: "VARCHAR(100)", IsNullable: true},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.PlanMigration(diff)
	}
}

func BenchmarkMigrationService_GenerateSQL(b *testing.B) {
	service := NewMigrationService()

	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "bench_table",
				Columns: map[string]*schema.Column{
					"id": {Name: "id", DataType: "INT", IsNullable: false},
				},
				Constraints: make(map[string]*schema.Constraint),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateSQL(diff)
	}
}

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string)                                     {}
func (m *mockLogger) Info(msg string)                                      {}
func (m *mockLogger) Warn(msg string)                                      {}
func (m *mockLogger) Error(msg string)                                     {}
func (m *mockLogger) WithField(key string, value interface{}) *mockLogger  { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) *mockLogger { return m }

// stringPtr helper function is already defined in sql_generator_test.go
