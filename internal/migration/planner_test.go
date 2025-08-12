package migration

import (
	"testing"

	"mysql-schema-sync/internal/schema"
)

func TestMigrationPlanner_PlanMigration(t *testing.T) {
	planner := NewMigrationPlanner()

	// Create a simple schema diff
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

	plan, err := planner.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Expected non-nil migration plan")
	}

	// Should have statements for both table creation and removal
	if len(plan.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(plan.Statements))
	}

	// Check that statements are ordered correctly (drops before creates)
	if len(plan.Statements) >= 2 {
		firstStmt := plan.Statements[0]
		lastStmt := plan.Statements[len(plan.Statements)-1]

		if firstStmt.Type != StatementTypeDropTable {
			t.Errorf("Expected first statement to be DROP TABLE, got %s", firstStmt.Type)
		}

		if lastStmt.Type != StatementTypeCreateTable {
			t.Errorf("Expected last statement to be CREATE TABLE, got %s", lastStmt.Type)
		}
	}

	// Should have warnings for destructive operations
	if len(plan.Warnings) == 0 {
		t.Error("Expected warnings for destructive operations")
	}

	// Check summary
	if plan.Summary.TablesAdded != 1 {
		t.Errorf("Expected 1 table added, got %d", plan.Summary.TablesAdded)
	}

	if plan.Summary.TablesRemoved != 1 {
		t.Errorf("Expected 1 table removed, got %d", plan.Summary.TablesRemoved)
	}

	if !plan.HasDestructiveOperations() {
		t.Error("Expected plan to have destructive operations")
	}
}

func TestMigrationPlanner_PlanTableModifications(t *testing.T) {
	planner := NewMigrationPlanner()

	// Create table diff with column changes
	tableDiff := &schema.TableDiff{
		TableName: "test_table",
		AddedColumns: []*schema.Column{
			{
				Name:       "new_column",
				DataType:   "VARCHAR(255)",
				IsNullable: true,
			},
		},
		RemovedColumns: []*schema.Column{
			{
				Name:       "old_column",
				DataType:   "INT",
				IsNullable: false,
			},
		},
		ModifiedColumns: []*schema.ColumnDiff{
			{
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
			},
		},
	}

	diff := &schema.SchemaDiff{
		ModifiedTables: []*schema.TableDiff{tableDiff},
	}

	plan, err := planner.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	// Should have statements for column operations
	expectedStatements := 3 // DROP, ADD, MODIFY
	if len(plan.Statements) != expectedStatements {
		t.Errorf("Expected %d statements, got %d", expectedStatements, len(plan.Statements))
	}

	// Check statement types and order
	stmtTypes := make([]StatementType, len(plan.Statements))
	for i, stmt := range plan.Statements {
		stmtTypes[i] = stmt.Type
	}

	// Should be ordered: DROP, ADD, MODIFY
	expectedOrder := []StatementType{
		StatementTypeDropColumn,
		StatementTypeAddColumn,
		StatementTypeModifyColumn,
	}

	for i, expectedType := range expectedOrder {
		if i < len(stmtTypes) && stmtTypes[i] != expectedType {
			t.Errorf("Expected statement %d to be %s, got %s", i, expectedType, stmtTypes[i])
		}
	}

	// Should have warnings for destructive operations and column modifications
	if len(plan.Warnings) == 0 {
		t.Error("Expected warnings for destructive operations and column modifications")
	}
}

func TestMigrationPlanner_PlanIndexOperations(t *testing.T) {
	planner := NewMigrationPlanner()

	// Create schema diff with index changes
	diff := &schema.SchemaDiff{
		AddedIndexes: []*schema.Index{
			{
				Name:      "idx_new",
				TableName: "test_table",
				Columns:   []string{"column1", "column2"},
				IsUnique:  false,
				IsPrimary: false,
			},
		},
		RemovedIndexes: []*schema.Index{
			{
				Name:      "idx_old",
				TableName: "test_table",
				Columns:   []string{"old_column"},
				IsUnique:  true,
				IsPrimary: false,
			},
		},
	}

	plan, err := planner.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	// Should have statements for both index operations
	if len(plan.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(plan.Statements))
	}

	// Check that DROP comes before CREATE
	if len(plan.Statements) >= 2 {
		firstStmt := plan.Statements[0]
		lastStmt := plan.Statements[len(plan.Statements)-1]

		if firstStmt.Type != StatementTypeDropIndex {
			t.Errorf("Expected first statement to be DROP INDEX, got %s", firstStmt.Type)
		}

		if lastStmt.Type != StatementTypeCreateIndex {
			t.Errorf("Expected last statement to be CREATE INDEX, got %s", lastStmt.Type)
		}
	}

	// Check summary
	if plan.Summary.IndexesAdded != 1 {
		t.Errorf("Expected 1 index added, got %d", plan.Summary.IndexesAdded)
	}

	if plan.Summary.IndexesRemoved != 1 {
		t.Errorf("Expected 1 index removed, got %d", plan.Summary.IndexesRemoved)
	}
}

func TestMigrationPlanner_PlanConstraintOperations(t *testing.T) {
	planner := NewMigrationPlanner()

	// Create schema diff with constraint changes
	diff := &schema.SchemaDiff{
		AddedConstraints: []*schema.Constraint{
			{
				Name:              "fk_new",
				TableName:         "test_table",
				Type:              schema.ConstraintTypeForeignKey,
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedColumns: []string{"id"},
			},
		},
		RemovedConstraints: []*schema.Constraint{
			{
				Name:      "uk_old",
				TableName: "test_table",
				Type:      schema.ConstraintTypeUnique,
				Columns:   []string{"old_unique_column"},
			},
		},
	}

	plan, err := planner.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	// Should have statements for both constraint operations
	if len(plan.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(plan.Statements))
	}

	// Check that DROP comes before ADD
	if len(plan.Statements) >= 2 {
		firstStmt := plan.Statements[0]
		lastStmt := plan.Statements[len(plan.Statements)-1]

		if firstStmt.Type != StatementTypeDropConstraint {
			t.Errorf("Expected first statement to be DROP CONSTRAINT, got %s", firstStmt.Type)
		}

		if lastStmt.Type != StatementTypeAddConstraint {
			t.Errorf("Expected last statement to be ADD CONSTRAINT, got %s", lastStmt.Type)
		}
	}

	// Check summary
	if plan.Summary.ConstraintsAdded != 1 {
		t.Errorf("Expected 1 constraint added, got %d", plan.Summary.ConstraintsAdded)
	}

	if plan.Summary.ConstraintsRemoved != 1 {
		t.Errorf("Expected 1 constraint removed, got %d", plan.Summary.ConstraintsRemoved)
	}
}

func TestMigrationPlanner_NilDiff(t *testing.T) {
	planner := NewMigrationPlanner()

	_, err := planner.PlanMigration(nil)
	if err == nil {
		t.Error("Expected error for nil schema diff")
	}
}

func TestMigrationPlanner_EmptyDiff(t *testing.T) {
	planner := NewMigrationPlanner()

	diff := &schema.SchemaDiff{}

	plan, err := planner.PlanMigration(diff)
	if err != nil {
		t.Fatalf("PlanMigration() error = %v", err)
	}

	if len(plan.Statements) != 0 {
		t.Errorf("Expected 0 statements for empty diff, got %d", len(plan.Statements))
	}

	if plan.HasDestructiveOperations() {
		t.Error("Expected no destructive operations for empty diff")
	}
}

func TestNewMigrationPlanner(t *testing.T) {
	planner := NewMigrationPlanner()

	if planner == nil {
		t.Fatal("Expected non-nil migration planner")
	}

	if planner.sqlGenerator == nil {
		t.Error("Expected SQL generator to be initialized")
	}
}
