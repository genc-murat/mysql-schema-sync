package migration

import (
	"strings"
	"testing"

	"mysql-schema-sync/internal/schema"
)

// TestMigrationIntegration tests the complete migration workflow
func TestMigrationIntegration(t *testing.T) {
	service := NewMigrationService()

	// Create a comprehensive schema diff that covers all change types
	diff := createComprehensiveSchemaDiff()

	// Test planning the migration
	plan, err := service.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	// Verify plan structure
	if plan == nil {
		t.Fatal("Expected non-nil migration plan")
	}

	// Should have statements for all types of changes
	expectedMinStatements := 10 // At least one of each type
	if len(plan.Statements) < expectedMinStatements {
		t.Errorf("Expected at least %d statements, got %d", expectedMinStatements, len(plan.Statements))
	}

	// Verify statement ordering (destructive operations first)
	verifyStatementOrdering(t, plan)

	// Test SQL generation
	sqlStatements, err := service.GenerateSQL(diff)
	if err != nil {
		t.Fatalf("GenerateSQL() error = %v", err)
	}

	if len(sqlStatements) != len(plan.Statements) {
		t.Errorf("Expected %d SQL statements, got %d", len(plan.Statements), len(sqlStatements))
	}

	// Verify SQL content
	verifySQLContent(t, sqlStatements)

	// Test plan validation
	err = service.ValidatePlan(plan)
	if err != nil {
		t.Errorf("ValidatePlan() error = %v", err)
	}

	// Verify summary
	verifySummary(t, plan)

	// Verify warnings for destructive operations
	if !plan.HasDestructiveOperations() {
		t.Error("Expected plan to have destructive operations")
	}

	if len(plan.Warnings) == 0 {
		t.Error("Expected warnings for destructive operations")
	}
}

// TestAllSQLGenerationMethods tests all individual SQL generation methods
func TestAllSQLGenerationMethods(t *testing.T) {
	service := NewMigrationService()

	// Test CREATE TABLE
	table := createTestTable()
	sql, err := service.GenerateCreateTableSQL(table)
	if err != nil {
		t.Errorf("GenerateCreateTableSQL() error = %v", err)
	}
	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("Expected CREATE TABLE SQL")
	}

	// Test DROP TABLE
	sql, err = service.GenerateDropTableSQL(table)
	if err != nil {
		t.Errorf("GenerateDropTableSQL() error = %v", err)
	}
	if !strings.Contains(sql, "DROP TABLE") {
		t.Error("Expected DROP TABLE SQL")
	}

	// Test ADD COLUMN
	column := createTestColumn()
	sql, err = service.GenerateAddColumnSQL("test_table", column)
	if err != nil {
		t.Errorf("GenerateAddColumnSQL() error = %v", err)
	}
	if !strings.Contains(sql, "ADD COLUMN") {
		t.Error("Expected ADD COLUMN SQL")
	}

	// Test DROP COLUMN
	sql, err = service.GenerateDropColumnSQL("test_table", column)
	if err != nil {
		t.Errorf("GenerateDropColumnSQL() error = %v", err)
	}
	if !strings.Contains(sql, "DROP COLUMN") {
		t.Error("Expected DROP COLUMN SQL")
	}

	// Test MODIFY COLUMN
	columnDiff := createTestColumnDiff()
	sql, err = service.GenerateModifyColumnSQL("test_table", columnDiff)
	if err != nil {
		t.Errorf("GenerateModifyColumnSQL() error = %v", err)
	}
	if !strings.Contains(sql, "MODIFY COLUMN") {
		t.Error("Expected MODIFY COLUMN SQL")
	}

	// Test CREATE INDEX
	index := createTestIndex()
	sql, err = service.GenerateCreateIndexSQL(index)
	if err != nil {
		t.Errorf("GenerateCreateIndexSQL() error = %v", err)
	}
	if !strings.Contains(sql, "CREATE INDEX") {
		t.Error("Expected CREATE INDEX SQL")
	}

	// Test DROP INDEX
	sql, err = service.GenerateDropIndexSQL(index)
	if err != nil {
		t.Errorf("GenerateDropIndexSQL() error = %v", err)
	}
	if !strings.Contains(sql, "DROP INDEX") {
		t.Error("Expected DROP INDEX SQL")
	}

	// Test ADD CONSTRAINT
	constraint := createTestConstraint()
	sql, err = service.GenerateAddConstraintSQL(constraint)
	if err != nil {
		t.Errorf("GenerateAddConstraintSQL() error = %v", err)
	}
	if !strings.Contains(sql, "ADD CONSTRAINT") {
		t.Error("Expected ADD CONSTRAINT SQL")
	}

	// Test DROP CONSTRAINT
	sql, err = service.GenerateDropConstraintSQL(constraint)
	if err != nil {
		t.Errorf("GenerateDropConstraintSQL() error = %v", err)
	}
	if !strings.Contains(sql, "DROP FOREIGN KEY") {
		t.Error("Expected DROP FOREIGN KEY SQL")
	}
}

// Helper functions for creating test data

func createComprehensiveSchemaDiff() *schema.SchemaDiff {
	return &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			createTestTable(),
		},
		RemovedTables: []*schema.Table{
			{
				Name: "old_table",
				Columns: map[string]*schema.Column{
					"id": createTestColumn(),
				},
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "modified_table",
				AddedColumns: []*schema.Column{
					createTestColumn(),
				},
				RemovedColumns: []*schema.Column{
					{
						Name:       "old_column",
						DataType:   "INT",
						IsNullable: false,
					},
				},
				ModifiedColumns: []*schema.ColumnDiff{
					createTestColumnDiff(),
				},
				AddedConstraints: []*schema.Constraint{
					createTestConstraint(),
				},
				RemovedConstraints: []*schema.Constraint{
					{
						Name:      "old_constraint",
						TableName: "modified_table",
						Type:      schema.ConstraintTypeUnique,
						Columns:   []string{"old_column"},
					},
				},
			},
		},
		AddedIndexes: []*schema.Index{
			createTestIndex(),
		},
		RemovedIndexes: []*schema.Index{
			{
				Name:      "old_index",
				TableName: "test_table",
				Columns:   []string{"old_column"},
				IsUnique:  false,
				IsPrimary: false,
			},
		},
		AddedConstraints: []*schema.Constraint{
			createTestConstraint(),
		},
		RemovedConstraints: []*schema.Constraint{
			{
				Name:      "old_global_constraint",
				TableName: "test_table",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"old_column"},
			},
		},
	}
}

func createTestTable() *schema.Table {
	return &schema.Table{
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
		Constraints: map[string]*schema.Constraint{
			"uk_email": {
				Name:      "uk_email",
				TableName: "test_table",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"email"},
			},
		},
	}
}

func createTestColumn() *schema.Column {
	return &schema.Column{
		Name:       "test_column",
		DataType:   "VARCHAR(100)",
		IsNullable: true,
	}
}

func createTestColumnDiff() *schema.ColumnDiff {
	return &schema.ColumnDiff{
		ColumnName: "modified_column",
		OldColumn: &schema.Column{
			Name:       "modified_column",
			DataType:   "VARCHAR(100)",
			IsNullable: true,
		},
		NewColumn: &schema.Column{
			Name:       "modified_column",
			DataType:   "VARCHAR(255)",
			IsNullable: false,
		},
	}
}

func createTestIndex() *schema.Index {
	return &schema.Index{
		Name:      "idx_test",
		TableName: "test_table",
		Columns:   []string{"name", "email"},
		IsUnique:  false,
		IsPrimary: false,
	}
}

func createTestConstraint() *schema.Constraint {
	return &schema.Constraint{
		Name:              "fk_user_id",
		TableName:         "orders",
		Type:              schema.ConstraintTypeForeignKey,
		Columns:           []string{"user_id"},
		ReferencedTable:   "users",
		ReferencedColumns: []string{"id"},
		OnUpdate:          "CASCADE",
		OnDelete:          "RESTRICT",
	}
}

// Helper functions for verification

func verifyStatementOrdering(t *testing.T, plan *MigrationPlan) {
	for i := 1; i < len(plan.Statements); i++ {
		prevOrder := plan.Statements[i-1].Type.GetExecutionOrder()
		currOrder := plan.Statements[i].Type.GetExecutionOrder()

		if prevOrder > currOrder {
			t.Errorf("Statement ordering violation: %s (order %d) should come after %s (order %d)",
				plan.Statements[i-1].Type, prevOrder,
				plan.Statements[i].Type, currOrder)
		}
	}
}

func verifySQLContent(t *testing.T, sqlStatements []string) {
	hasCreateTable := false
	hasDropTable := false
	hasAddColumn := false
	hasDropColumn := false
	hasModifyColumn := false
	hasCreateIndex := false
	hasDropIndex := false
	hasAddConstraint := false
	hasDropConstraint := false

	for _, sql := range sqlStatements {
		if strings.Contains(sql, "CREATE TABLE") {
			hasCreateTable = true
		}
		if strings.Contains(sql, "DROP TABLE") {
			hasDropTable = true
		}
		if strings.Contains(sql, "ADD COLUMN") {
			hasAddColumn = true
		}
		if strings.Contains(sql, "DROP COLUMN") {
			hasDropColumn = true
		}
		if strings.Contains(sql, "MODIFY COLUMN") {
			hasModifyColumn = true
		}
		if strings.Contains(sql, "CREATE INDEX") {
			hasCreateIndex = true
		}
		if strings.Contains(sql, "DROP INDEX") {
			hasDropIndex = true
		}
		if strings.Contains(sql, "ADD CONSTRAINT") {
			hasAddConstraint = true
		}
		// For constraints, MySQL uses different DROP syntax depending on constraint type
		if strings.Contains(sql, "DROP FOREIGN KEY") || strings.Contains(sql, "DROP CHECK") ||
			(strings.Contains(sql, "DROP INDEX") && strings.Contains(sql, "ALTER TABLE")) {
			hasDropConstraint = true
		}
	}

	if !hasCreateTable {
		t.Error("Expected CREATE TABLE statement")
	}
	if !hasDropTable {
		t.Error("Expected DROP TABLE statement")
	}
	if !hasAddColumn {
		t.Error("Expected ADD COLUMN statement")
	}
	if !hasDropColumn {
		t.Error("Expected DROP COLUMN statement")
	}
	if !hasModifyColumn {
		t.Error("Expected MODIFY COLUMN statement")
	}
	if !hasCreateIndex {
		t.Error("Expected CREATE INDEX statement")
	}
	if !hasDropIndex {
		t.Error("Expected DROP INDEX statement")
	}
	if !hasAddConstraint {
		t.Error("Expected ADD CONSTRAINT statement")
	}
	if !hasDropConstraint {
		t.Error("Expected DROP CONSTRAINT statement")
	}
}

func verifySummary(t *testing.T, plan *MigrationPlan) {
	summary := plan.Summary

	if summary.TotalStatements != len(plan.Statements) {
		t.Errorf("Summary total statements mismatch: expected %d, got %d",
			len(plan.Statements), summary.TotalStatements)
	}

	if summary.TablesAdded == 0 {
		t.Error("Expected tables added count > 0")
	}

	if summary.TablesRemoved == 0 {
		t.Error("Expected tables removed count > 0")
	}

	if summary.ColumnsAdded == 0 {
		t.Error("Expected columns added count > 0")
	}

	if summary.ColumnsRemoved == 0 {
		t.Error("Expected columns removed count > 0")
	}

	if summary.ColumnsModified == 0 {
		t.Error("Expected columns modified count > 0")
	}

	if summary.IndexesAdded == 0 {
		t.Error("Expected indexes added count > 0")
	}

	if summary.IndexesRemoved == 0 {
		t.Error("Expected indexes removed count > 0")
	}

	if summary.ConstraintsAdded == 0 {
		t.Error("Expected constraints added count > 0")
	}

	if summary.ConstraintsRemoved == 0 {
		t.Error("Expected constraints removed count > 0")
	}

	if summary.DestructiveCount == 0 {
		t.Error("Expected destructive operations count > 0")
	}
}
