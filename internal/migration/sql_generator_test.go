package migration

import (
	"strings"
	"testing"

	"mysql-schema-sync/internal/schema"
)

func TestSQLGenerator_GenerateCreateTableSQL(t *testing.T) {
	generator := NewSQLGenerator()

	// Create a test table
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
				Name:         "name",
				DataType:     "VARCHAR(255)",
				IsNullable:   false,
				DefaultValue: nil,
			},
			"email": {
				Name:       "email",
				DataType:   "VARCHAR(255)",
				IsNullable: true,
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

	sql, err := generator.GenerateCreateTableSQL(table)
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL() error = %v", err)
	}

	// Check basic structure
	if !strings.Contains(sql, "CREATE TABLE `test_table`") {
		t.Error("Expected CREATE TABLE statement")
	}

	if !strings.Contains(sql, "`id` INT NOT NULL AUTO_INCREMENT") {
		t.Error("Expected id column definition")
	}

	if !strings.Contains(sql, "`name` VARCHAR(255) NOT NULL") {
		t.Error("Expected name column definition")
	}

	if !strings.Contains(sql, "`email` VARCHAR(255) NULL") {
		t.Error("Expected email column definition")
	}

	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Error("Expected primary key definition")
	}
}

func TestSQLGenerator_GenerateDropTableSQL(t *testing.T) {
	generator := NewSQLGenerator()

	table := &schema.Table{Name: "test_table"}

	sql, err := generator.GenerateDropTableSQL(table)
	if err != nil {
		t.Fatalf("GenerateDropTableSQL() error = %v", err)
	}

	expected := "DROP TABLE `test_table`"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestSQLGenerator_GenerateAddColumnSQL(t *testing.T) {
	generator := NewSQLGenerator()

	column := &schema.Column{
		Name:       "new_column",
		DataType:   "VARCHAR(100)",
		IsNullable: true,
	}

	sql, err := generator.GenerateAddColumnSQL("test_table", column)
	if err != nil {
		t.Fatalf("GenerateAddColumnSQL() error = %v", err)
	}

	expected := "ALTER TABLE `test_table` ADD COLUMN `new_column` VARCHAR(100) NULL"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestSQLGenerator_GenerateDropColumnSQL(t *testing.T) {
	generator := NewSQLGenerator()

	column := &schema.Column{Name: "old_column"}

	sql, err := generator.GenerateDropColumnSQL("test_table", column)
	if err != nil {
		t.Fatalf("GenerateDropColumnSQL() error = %v", err)
	}

	expected := "ALTER TABLE `test_table` DROP COLUMN `old_column`"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestSQLGenerator_GenerateModifyColumnSQL(t *testing.T) {
	generator := NewSQLGenerator()

	columnDiff := &schema.ColumnDiff{
		ColumnName: "modified_column",
		NewColumn: &schema.Column{
			Name:       "modified_column",
			DataType:   "VARCHAR(255)",
			IsNullable: false,
		},
	}

	sql, err := generator.GenerateModifyColumnSQL("test_table", columnDiff)
	if err != nil {
		t.Fatalf("GenerateModifyColumnSQL() error = %v", err)
	}

	expected := "ALTER TABLE `test_table` MODIFY COLUMN `modified_column` VARCHAR(255) NOT NULL"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestSQLGenerator_GenerateCreateIndexSQL(t *testing.T) {
	generator := NewSQLGenerator()

	tests := []struct {
		name     string
		index    *schema.Index
		expected string
	}{
		{
			name: "regular index",
			index: &schema.Index{
				Name:      "idx_name",
				TableName: "test_table",
				Columns:   []string{"name"},
				IsUnique:  false,
				IsPrimary: false,
			},
			expected: "CREATE INDEX `idx_name` ON `test_table` (`name`)",
		},
		{
			name: "unique index",
			index: &schema.Index{
				Name:      "uk_email",
				TableName: "test_table",
				Columns:   []string{"email"},
				IsUnique:  true,
				IsPrimary: false,
			},
			expected: "CREATE UNIQUE INDEX `uk_email` ON `test_table` (`email`)",
		},
		{
			name: "composite index",
			index: &schema.Index{
				Name:      "idx_name_email",
				TableName: "test_table",
				Columns:   []string{"name", "email"},
				IsUnique:  false,
				IsPrimary: false,
			},
			expected: "CREATE INDEX `idx_name_email` ON `test_table` (`name`, `email`)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := generator.GenerateCreateIndexSQL(tt.index)
			if err != nil {
				t.Fatalf("GenerateCreateIndexSQL() error = %v", err)
			}

			if sql != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, sql)
			}
		})
	}
}

func TestSQLGenerator_GenerateDropIndexSQL(t *testing.T) {
	generator := NewSQLGenerator()

	index := &schema.Index{
		Name:      "idx_name",
		TableName: "test_table",
	}

	sql, err := generator.GenerateDropIndexSQL(index)
	if err != nil {
		t.Fatalf("GenerateDropIndexSQL() error = %v", err)
	}

	expected := "DROP INDEX `idx_name` ON `test_table`"
	if sql != expected {
		t.Errorf("Expected %s, got %s", expected, sql)
	}
}

func TestSQLGenerator_GenerateAddConstraintSQL(t *testing.T) {
	generator := NewSQLGenerator()

	tests := []struct {
		name       string
		constraint *schema.Constraint
		expected   string
	}{
		{
			name: "foreign key constraint",
			constraint: &schema.Constraint{
				Name:              "fk_user_id",
				TableName:         "orders",
				Type:              schema.ConstraintTypeForeignKey,
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedColumns: []string{"id"},
				OnUpdate:          "CASCADE",
				OnDelete:          "RESTRICT",
			},
			expected: "ALTER TABLE `orders` ADD CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT",
		},
		{
			name: "unique constraint",
			constraint: &schema.Constraint{
				Name:      "uk_email",
				TableName: "users",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"email"},
			},
			expected: "ALTER TABLE `users` ADD CONSTRAINT `uk_email` UNIQUE (`email`)",
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
			expected: "ALTER TABLE `users` ADD CONSTRAINT `chk_age` CHECK (age >= 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := generator.GenerateAddConstraintSQL(tt.constraint)
			if err != nil {
				t.Fatalf("GenerateAddConstraintSQL() error = %v", err)
			}

			if sql != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, sql)
			}
		})
	}
}

func TestSQLGenerator_GenerateDropConstraintSQL(t *testing.T) {
	generator := NewSQLGenerator()

	tests := []struct {
		name       string
		constraint *schema.Constraint
		expected   string
	}{
		{
			name: "foreign key constraint",
			constraint: &schema.Constraint{
				Name:      "fk_user_id",
				TableName: "orders",
				Type:      schema.ConstraintTypeForeignKey,
			},
			expected: "ALTER TABLE `orders` DROP FOREIGN KEY `fk_user_id`",
		},
		{
			name: "unique constraint",
			constraint: &schema.Constraint{
				Name:      "uk_email",
				TableName: "users",
				Type:      schema.ConstraintTypeUnique,
			},
			expected: "ALTER TABLE `users` DROP INDEX `uk_email`",
		},
		{
			name: "check constraint",
			constraint: &schema.Constraint{
				Name:      "chk_age",
				TableName: "users",
				Type:      schema.ConstraintTypeCheck,
			},
			expected: "ALTER TABLE `users` DROP CHECK `chk_age`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := generator.GenerateDropConstraintSQL(tt.constraint)
			if err != nil {
				t.Fatalf("GenerateDropConstraintSQL() error = %v", err)
			}

			if sql != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, sql)
			}
		})
	}
}

func TestSQLGenerator_generateColumnDefinition(t *testing.T) {
	generator := NewSQLGenerator()

	tests := []struct {
		name     string
		column   *schema.Column
		expected string
	}{
		{
			name: "basic column",
			column: &schema.Column{
				Name:       "id",
				DataType:   "INT",
				IsNullable: false,
			},
			expected: "`id` INT NOT NULL",
		},
		{
			name: "nullable column",
			column: &schema.Column{
				Name:       "name",
				DataType:   "VARCHAR(255)",
				IsNullable: true,
			},
			expected: "`name` VARCHAR(255) NULL",
		},
		{
			name: "column with default",
			column: &schema.Column{
				Name:         "status",
				DataType:     "VARCHAR(50)",
				IsNullable:   false,
				DefaultValue: stringPtr("active"),
			},
			expected: "`status` VARCHAR(50) NOT NULL DEFAULT 'active'",
		},
		{
			name: "column with auto increment",
			column: &schema.Column{
				Name:       "id",
				DataType:   "INT",
				IsNullable: false,
				Extra:      "AUTO_INCREMENT",
			},
			expected: "`id` INT NOT NULL AUTO_INCREMENT",
		},
		{
			name: "column with current timestamp default",
			column: &schema.Column{
				Name:         "created_at",
				DataType:     "TIMESTAMP",
				IsNullable:   false,
				DefaultValue: stringPtr("CURRENT_TIMESTAMP"),
			},
			expected: "`created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.generateColumnDefinition(tt.column)
			if err != nil {
				t.Fatalf("generateColumnDefinition() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSQLGenerator_NilInputs(t *testing.T) {
	generator := NewSQLGenerator()

	// Test nil table
	_, err := generator.GenerateCreateTableSQL(nil)
	if err == nil {
		t.Error("Expected error for nil table")
	}

	_, err = generator.GenerateDropTableSQL(nil)
	if err == nil {
		t.Error("Expected error for nil table")
	}

	// Test nil column
	_, err = generator.GenerateAddColumnSQL("test", nil)
	if err == nil {
		t.Error("Expected error for nil column")
	}

	_, err = generator.GenerateDropColumnSQL("test", nil)
	if err == nil {
		t.Error("Expected error for nil column")
	}

	// Test nil column diff
	_, err = generator.GenerateModifyColumnSQL("test", nil)
	if err == nil {
		t.Error("Expected error for nil column diff")
	}

	// Test nil index
	_, err = generator.GenerateCreateIndexSQL(nil)
	if err == nil {
		t.Error("Expected error for nil index")
	}

	_, err = generator.GenerateDropIndexSQL(nil)
	if err == nil {
		t.Error("Expected error for nil index")
	}

	// Test nil constraint
	_, err = generator.GenerateAddConstraintSQL(nil)
	if err == nil {
		t.Error("Expected error for nil constraint")
	}

	_, err = generator.GenerateDropConstraintSQL(nil)
	if err == nil {
		t.Error("Expected error for nil constraint")
	}
}

func TestSQLGenerator_EmptyInputs(t *testing.T) {
	generator := NewSQLGenerator()

	// Test empty table name
	column := &schema.Column{Name: "test", DataType: "INT", IsNullable: false}
	_, err := generator.GenerateAddColumnSQL("", column)
	if err == nil {
		t.Error("Expected error for empty table name")
	}

	_, err = generator.GenerateDropColumnSQL("", column)
	if err == nil {
		t.Error("Expected error for empty table name")
	}

	columnDiff := &schema.ColumnDiff{
		ColumnName: "test",
		NewColumn:  column,
	}
	_, err = generator.GenerateModifyColumnSQL("", columnDiff)
	if err == nil {
		t.Error("Expected error for empty table name")
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
