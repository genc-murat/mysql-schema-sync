package migration

import (
	"fmt"
	"strings"
)

// StatementType represents the type of migration statement
type StatementType string

const (
	StatementTypeCreateTable    StatementType = "CREATE_TABLE"
	StatementTypeDropTable      StatementType = "DROP_TABLE"
	StatementTypeAddColumn      StatementType = "ADD_COLUMN"
	StatementTypeDropColumn     StatementType = "DROP_COLUMN"
	StatementTypeModifyColumn   StatementType = "MODIFY_COLUMN"
	StatementTypeCreateIndex    StatementType = "CREATE_INDEX"
	StatementTypeDropIndex      StatementType = "DROP_INDEX"
	StatementTypeAddConstraint  StatementType = "ADD_CONSTRAINT"
	StatementTypeDropConstraint StatementType = "DROP_CONSTRAINT"
)

// MigrationStatement represents a single SQL statement in a migration
type MigrationStatement struct {
	SQL           string        `json:"sql"`
	Type          StatementType `json:"type"`
	Description   string        `json:"description"`
	IsDestructive bool          `json:"is_destructive"`
	TableName     string        `json:"table_name,omitempty"`
	Dependencies  []string      `json:"dependencies,omitempty"`
}

// MigrationPlan represents a complete migration plan with ordered statements
type MigrationPlan struct {
	Statements []MigrationStatement `json:"statements"`
	Warnings   []string             `json:"warnings"`
	Summary    MigrationSummary     `json:"summary"`
}

// MigrationSummary provides a high-level overview of the migration
type MigrationSummary struct {
	TotalStatements    int `json:"total_statements"`
	DestructiveCount   int `json:"destructive_count"`
	TablesAdded        int `json:"tables_added"`
	TablesRemoved      int `json:"tables_removed"`
	TablesModified     int `json:"tables_modified"`
	ColumnsAdded       int `json:"columns_added"`
	ColumnsRemoved     int `json:"columns_removed"`
	ColumnsModified    int `json:"columns_modified"`
	IndexesAdded       int `json:"indexes_added"`
	IndexesRemoved     int `json:"indexes_removed"`
	ConstraintsAdded   int `json:"constraints_added"`
	ConstraintsRemoved int `json:"constraints_removed"`
}

// Validate validates the MigrationStatement
func (ms *MigrationStatement) Validate() error {
	if ms.SQL == "" {
		return fmt.Errorf("migration statement SQL cannot be empty")
	}

	if ms.Type == "" {
		return fmt.Errorf("migration statement type cannot be empty")
	}

	if ms.Description == "" {
		return fmt.Errorf("migration statement description cannot be empty")
	}

	// Validate statement type
	validTypes := map[StatementType]bool{
		StatementTypeCreateTable:    true,
		StatementTypeDropTable:      true,
		StatementTypeAddColumn:      true,
		StatementTypeDropColumn:     true,
		StatementTypeModifyColumn:   true,
		StatementTypeCreateIndex:    true,
		StatementTypeDropIndex:      true,
		StatementTypeAddConstraint:  true,
		StatementTypeDropConstraint: true,
	}

	if !validTypes[ms.Type] {
		return fmt.Errorf("invalid statement type: %s", ms.Type)
	}

	return nil
}

// Validate validates the MigrationPlan
func (mp *MigrationPlan) Validate() error {
	if len(mp.Statements) == 0 {
		return fmt.Errorf("migration plan must have at least one statement")
	}

	// Validate all statements
	for i, stmt := range mp.Statements {
		if err := stmt.Validate(); err != nil {
			return fmt.Errorf("invalid statement at index %d: %w", i, err)
		}
	}

	return nil
}

// IsDestructive returns true if the statement type is considered destructive
func (st StatementType) IsDestructive() bool {
	destructiveTypes := map[StatementType]bool{
		StatementTypeDropTable:      true,
		StatementTypeDropColumn:     true,
		StatementTypeDropIndex:      true,
		StatementTypeDropConstraint: true,
	}

	return destructiveTypes[st]
}

// GetExecutionOrder returns the execution order priority for statement types
// Lower numbers execute first
func (st StatementType) GetExecutionOrder() int {
	orderMap := map[StatementType]int{
		// First: Drop foreign key constraints to avoid dependency issues
		StatementTypeDropConstraint: 1,
		// Second: Drop indexes (except primary keys handled with tables)
		StatementTypeDropIndex: 2,
		// Third: Drop columns
		StatementTypeDropColumn: 3,
		// Fourth: Drop tables
		StatementTypeDropTable: 4,
		// Fifth: Create tables
		StatementTypeCreateTable: 5,
		// Sixth: Add columns
		StatementTypeAddColumn: 6,
		// Seventh: Modify columns
		StatementTypeModifyColumn: 7,
		// Eighth: Create indexes
		StatementTypeCreateIndex: 8,
		// Ninth: Add constraints (foreign keys last)
		StatementTypeAddConstraint: 9,
	}

	if order, exists := orderMap[st]; exists {
		return order
	}
	return 999 // Unknown types go last
}

// NewMigrationStatement creates a new MigrationStatement
func NewMigrationStatement(sql string, stmtType StatementType, description string) *MigrationStatement {
	return &MigrationStatement{
		SQL:           sql,
		Type:          stmtType,
		Description:   description,
		IsDestructive: stmtType.IsDestructive(),
		Dependencies:  make([]string, 0),
	}
}

// NewMigrationPlan creates a new MigrationPlan
func NewMigrationPlan() *MigrationPlan {
	return &MigrationPlan{
		Statements: make([]MigrationStatement, 0),
		Warnings:   make([]string, 0),
		Summary:    MigrationSummary{},
	}
}

// AddStatement adds a statement to the migration plan
func (mp *MigrationPlan) AddStatement(stmt MigrationStatement) error {
	if err := stmt.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid statement: %w", err)
	}

	mp.Statements = append(mp.Statements, stmt)
	mp.updateSummary()
	return nil
}

// AddWarning adds a warning to the migration plan
func (mp *MigrationPlan) AddWarning(warning string) {
	mp.Warnings = append(mp.Warnings, warning)
}

// updateSummary updates the migration summary based on current statements
func (mp *MigrationPlan) updateSummary() {
	summary := MigrationSummary{}
	summary.TotalStatements = len(mp.Statements)

	for _, stmt := range mp.Statements {
		if stmt.IsDestructive {
			summary.DestructiveCount++
		}

		switch stmt.Type {
		case StatementTypeCreateTable:
			summary.TablesAdded++
		case StatementTypeDropTable:
			summary.TablesRemoved++
		case StatementTypeAddColumn:
			summary.ColumnsAdded++
		case StatementTypeDropColumn:
			summary.ColumnsRemoved++
		case StatementTypeModifyColumn:
			summary.ColumnsModified++
		case StatementTypeCreateIndex:
			summary.IndexesAdded++
		case StatementTypeDropIndex:
			summary.IndexesRemoved++
		case StatementTypeAddConstraint:
			summary.ConstraintsAdded++
		case StatementTypeDropConstraint:
			summary.ConstraintsRemoved++
		}
	}

	// Count modified tables by checking which tables have column modifications
	modifiedTables := make(map[string]bool)
	for _, stmt := range mp.Statements {
		if stmt.Type == StatementTypeAddColumn || stmt.Type == StatementTypeDropColumn || stmt.Type == StatementTypeModifyColumn {
			if stmt.TableName != "" {
				modifiedTables[stmt.TableName] = true
			}
		}
	}
	summary.TablesModified = len(modifiedTables)

	mp.Summary = summary
}

// HasDestructiveOperations returns true if the plan contains destructive operations
func (mp *MigrationPlan) HasDestructiveOperations() bool {
	return mp.Summary.DestructiveCount > 0
}

// GetStatementsByType returns all statements of a specific type
func (mp *MigrationPlan) GetStatementsByType(stmtType StatementType) []MigrationStatement {
	var statements []MigrationStatement
	for _, stmt := range mp.Statements {
		if stmt.Type == stmtType {
			statements = append(statements, stmt)
		}
	}
	return statements
}

// GetStatementsByTable returns all statements affecting a specific table
func (mp *MigrationPlan) GetStatementsByTable(tableName string) []MigrationStatement {
	var statements []MigrationStatement
	for _, stmt := range mp.Statements {
		if stmt.TableName == tableName {
			statements = append(statements, stmt)
		}
	}
	return statements
}

// String returns a string representation of the migration plan
func (mp *MigrationPlan) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Migration Plan Summary:\n"))
	builder.WriteString(fmt.Sprintf("  Total Statements: %d\n", mp.Summary.TotalStatements))
	builder.WriteString(fmt.Sprintf("  Destructive Operations: %d\n", mp.Summary.DestructiveCount))
	builder.WriteString(fmt.Sprintf("  Tables: +%d -%d ~%d\n",
		mp.Summary.TablesAdded, mp.Summary.TablesRemoved, mp.Summary.TablesModified))
	builder.WriteString(fmt.Sprintf("  Columns: +%d -%d ~%d\n",
		mp.Summary.ColumnsAdded, mp.Summary.ColumnsRemoved, mp.Summary.ColumnsModified))
	builder.WriteString(fmt.Sprintf("  Indexes: +%d -%d\n",
		mp.Summary.IndexesAdded, mp.Summary.IndexesRemoved))
	builder.WriteString(fmt.Sprintf("  Constraints: +%d -%d\n",
		mp.Summary.ConstraintsAdded, mp.Summary.ConstraintsRemoved))

	if len(mp.Warnings) > 0 {
		builder.WriteString(fmt.Sprintf("\nWarnings:\n"))
		for _, warning := range mp.Warnings {
			builder.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
	}

	return builder.String()
}
