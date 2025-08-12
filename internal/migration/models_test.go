package migration

import (
	"testing"
)

func TestMigrationStatement_Validate(t *testing.T) {
	tests := []struct {
		name    string
		stmt    MigrationStatement
		wantErr bool
	}{
		{
			name: "valid statement",
			stmt: MigrationStatement{
				SQL:         "CREATE TABLE test (id INT)",
				Type:        StatementTypeCreateTable,
				Description: "Create test table",
			},
			wantErr: false,
		},
		{
			name: "empty SQL",
			stmt: MigrationStatement{
				SQL:         "",
				Type:        StatementTypeCreateTable,
				Description: "Create test table",
			},
			wantErr: true,
		},
		{
			name: "empty type",
			stmt: MigrationStatement{
				SQL:         "CREATE TABLE test (id INT)",
				Type:        "",
				Description: "Create test table",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			stmt: MigrationStatement{
				SQL:         "CREATE TABLE test (id INT)",
				Type:        StatementTypeCreateTable,
				Description: "",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			stmt: MigrationStatement{
				SQL:         "CREATE TABLE test (id INT)",
				Type:        "INVALID_TYPE",
				Description: "Create test table",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.stmt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MigrationStatement.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStatementType_IsDestructive(t *testing.T) {
	tests := []struct {
		name string
		st   StatementType
		want bool
	}{
		{"CREATE_TABLE", StatementTypeCreateTable, false},
		{"DROP_TABLE", StatementTypeDropTable, true},
		{"ADD_COLUMN", StatementTypeAddColumn, false},
		{"DROP_COLUMN", StatementTypeDropColumn, true},
		{"MODIFY_COLUMN", StatementTypeModifyColumn, false},
		{"CREATE_INDEX", StatementTypeCreateIndex, false},
		{"DROP_INDEX", StatementTypeDropIndex, true},
		{"ADD_CONSTRAINT", StatementTypeAddConstraint, false},
		{"DROP_CONSTRAINT", StatementTypeDropConstraint, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.IsDestructive(); got != tt.want {
				t.Errorf("StatementType.IsDestructive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatementType_GetExecutionOrder(t *testing.T) {
	tests := []struct {
		name string
		st   StatementType
		want int
	}{
		{"DROP_CONSTRAINT", StatementTypeDropConstraint, 1},
		{"DROP_INDEX", StatementTypeDropIndex, 2},
		{"DROP_COLUMN", StatementTypeDropColumn, 3},
		{"DROP_TABLE", StatementTypeDropTable, 4},
		{"CREATE_TABLE", StatementTypeCreateTable, 5},
		{"ADD_COLUMN", StatementTypeAddColumn, 6},
		{"MODIFY_COLUMN", StatementTypeModifyColumn, 7},
		{"CREATE_INDEX", StatementTypeCreateIndex, 8},
		{"ADD_CONSTRAINT", StatementTypeAddConstraint, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.GetExecutionOrder(); got != tt.want {
				t.Errorf("StatementType.GetExecutionOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrationPlan_AddStatement(t *testing.T) {
	plan := NewMigrationPlan()

	stmt := *NewMigrationStatement(
		"CREATE TABLE test (id INT)",
		StatementTypeCreateTable,
		"Create test table",
	)

	err := plan.AddStatement(stmt)
	if err != nil {
		t.Errorf("MigrationPlan.AddStatement() error = %v", err)
	}

	if len(plan.Statements) != 1 {
		t.Errorf("Expected 1 statement, got %d", len(plan.Statements))
	}

	if plan.Summary.TotalStatements != 1 {
		t.Errorf("Expected summary total statements = 1, got %d", plan.Summary.TotalStatements)
	}

	if plan.Summary.TablesAdded != 1 {
		t.Errorf("Expected summary tables added = 1, got %d", plan.Summary.TablesAdded)
	}
}

func TestMigrationPlan_HasDestructiveOperations(t *testing.T) {
	plan := NewMigrationPlan()

	// Initially no destructive operations
	if plan.HasDestructiveOperations() {
		t.Error("Expected no destructive operations initially")
	}

	// Add non-destructive statement
	stmt1 := *NewMigrationStatement(
		"CREATE TABLE test (id INT)",
		StatementTypeCreateTable,
		"Create test table",
	)
	plan.AddStatement(stmt1)

	if plan.HasDestructiveOperations() {
		t.Error("Expected no destructive operations after adding CREATE TABLE")
	}

	// Add destructive statement
	stmt2 := *NewMigrationStatement(
		"DROP TABLE test",
		StatementTypeDropTable,
		"Drop test table",
	)
	plan.AddStatement(stmt2)

	if !plan.HasDestructiveOperations() {
		t.Error("Expected destructive operations after adding DROP TABLE")
	}

	if plan.Summary.DestructiveCount != 1 {
		t.Errorf("Expected 1 destructive operation, got %d", plan.Summary.DestructiveCount)
	}
}

func TestMigrationPlan_GetStatementsByType(t *testing.T) {
	plan := NewMigrationPlan()

	// Add statements of different types
	stmt1 := *NewMigrationStatement(
		"CREATE TABLE test1 (id INT)",
		StatementTypeCreateTable,
		"Create test1 table",
	)
	stmt2 := *NewMigrationStatement(
		"CREATE TABLE test2 (id INT)",
		StatementTypeCreateTable,
		"Create test2 table",
	)
	stmt3 := *NewMigrationStatement(
		"DROP TABLE old_table",
		StatementTypeDropTable,
		"Drop old table",
	)

	plan.AddStatement(stmt1)
	plan.AddStatement(stmt2)
	plan.AddStatement(stmt3)

	createStatements := plan.GetStatementsByType(StatementTypeCreateTable)
	if len(createStatements) != 2 {
		t.Errorf("Expected 2 CREATE TABLE statements, got %d", len(createStatements))
	}

	dropStatements := plan.GetStatementsByType(StatementTypeDropTable)
	if len(dropStatements) != 1 {
		t.Errorf("Expected 1 DROP TABLE statement, got %d", len(dropStatements))
	}
}

func TestMigrationPlan_GetStatementsByTable(t *testing.T) {
	plan := NewMigrationPlan()

	// Add statements for different tables
	stmt1 := *NewMigrationStatement(
		"CREATE TABLE test (id INT)",
		StatementTypeCreateTable,
		"Create test table",
	)
	stmt1.TableName = "test"

	stmt2 := *NewMigrationStatement(
		"ALTER TABLE test ADD COLUMN name VARCHAR(255)",
		StatementTypeAddColumn,
		"Add name column to test",
	)
	stmt2.TableName = "test"

	stmt3 := *NewMigrationStatement(
		"CREATE TABLE other (id INT)",
		StatementTypeCreateTable,
		"Create other table",
	)
	stmt3.TableName = "other"

	plan.AddStatement(stmt1)
	plan.AddStatement(stmt2)
	plan.AddStatement(stmt3)

	testStatements := plan.GetStatementsByTable("test")
	if len(testStatements) != 2 {
		t.Errorf("Expected 2 statements for 'test' table, got %d", len(testStatements))
	}

	otherStatements := plan.GetStatementsByTable("other")
	if len(otherStatements) != 1 {
		t.Errorf("Expected 1 statement for 'other' table, got %d", len(otherStatements))
	}
}

func TestNewMigrationStatement(t *testing.T) {
	sql := "CREATE TABLE test (id INT)"
	stmtType := StatementTypeCreateTable
	description := "Create test table"

	stmt := NewMigrationStatement(sql, stmtType, description)

	if stmt.SQL != sql {
		t.Errorf("Expected SQL = %s, got %s", sql, stmt.SQL)
	}

	if stmt.Type != stmtType {
		t.Errorf("Expected Type = %s, got %s", stmtType, stmt.Type)
	}

	if stmt.Description != description {
		t.Errorf("Expected Description = %s, got %s", description, stmt.Description)
	}

	if stmt.IsDestructive != stmtType.IsDestructive() {
		t.Errorf("Expected IsDestructive = %v, got %v", stmtType.IsDestructive(), stmt.IsDestructive)
	}

	if stmt.Dependencies == nil {
		t.Error("Expected Dependencies to be initialized")
	}
}

func TestNewMigrationPlan(t *testing.T) {
	plan := NewMigrationPlan()

	if plan.Statements == nil {
		t.Error("Expected Statements to be initialized")
	}

	if plan.Warnings == nil {
		t.Error("Expected Warnings to be initialized")
	}

	if len(plan.Statements) != 0 {
		t.Errorf("Expected 0 statements initially, got %d", len(plan.Statements))
	}

	if len(plan.Warnings) != 0 {
		t.Errorf("Expected 0 warnings initially, got %d", len(plan.Warnings))
	}
}
