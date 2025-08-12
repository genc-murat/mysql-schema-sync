package migration

import (
	"fmt"
	"sort"
	"strings"

	"mysql-schema-sync/internal/schema"
)

// MigrationPlanner handles the creation of migration plans from schema differences
type MigrationPlanner struct {
	sqlGenerator *SQLGenerator
}

// NewMigrationPlanner creates a new MigrationPlanner instance
func NewMigrationPlanner() *MigrationPlanner {
	return &MigrationPlanner{
		sqlGenerator: NewSQLGenerator(),
	}
}

// PlanMigration creates a migration plan from schema differences
func (mp *MigrationPlanner) PlanMigration(diff *schema.SchemaDiff) (*MigrationPlan, error) {
	if diff == nil {
		return nil, fmt.Errorf("schema diff cannot be nil")
	}

	plan := NewMigrationPlan()

	// Generate statements for each type of change
	if err := mp.planTableRemovals(plan, diff.RemovedTables); err != nil {
		return nil, fmt.Errorf("failed to plan table removals: %w", err)
	}

	if err := mp.planConstraintRemovals(plan, diff.RemovedConstraints); err != nil {
		return nil, fmt.Errorf("failed to plan constraint removals: %w", err)
	}

	if err := mp.planIndexRemovals(plan, diff.RemovedIndexes); err != nil {
		return nil, fmt.Errorf("failed to plan index removals: %w", err)
	}

	if err := mp.planTableModifications(plan, diff.ModifiedTables); err != nil {
		return nil, fmt.Errorf("failed to plan table modifications: %w", err)
	}

	if err := mp.planTableAdditions(plan, diff.AddedTables); err != nil {
		return nil, fmt.Errorf("failed to plan table additions: %w", err)
	}

	if err := mp.planIndexAdditions(plan, diff.AddedIndexes); err != nil {
		return nil, fmt.Errorf("failed to plan index additions: %w", err)
	}

	if err := mp.planConstraintAdditions(plan, diff.AddedConstraints); err != nil {
		return nil, fmt.Errorf("failed to plan constraint additions: %w", err)
	}

	// Sort statements by execution order
	mp.sortStatements(plan)

	// Add warnings for destructive operations
	mp.addDestructiveWarnings(plan)

	return plan, nil
}

// planTableRemovals plans the removal of tables
func (mp *MigrationPlanner) planTableRemovals(plan *MigrationPlan, tables []*schema.Table) error {
	for _, table := range tables {
		sql, err := mp.sqlGenerator.GenerateDropTableSQL(table)
		if err != nil {
			return fmt.Errorf("failed to generate drop table SQL for %s: %w", table.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeDropTable,
			fmt.Sprintf("Drop table %s", table.Name),
		)
		stmt.TableName = table.Name

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add drop table statement: %w", err)
		}

		plan.AddWarning(fmt.Sprintf("Dropping table '%s' will permanently delete all data in the table", table.Name))
	}

	return nil
}

// planTableAdditions plans the addition of new tables
func (mp *MigrationPlanner) planTableAdditions(plan *MigrationPlan, tables []*schema.Table) error {
	for _, table := range tables {
		sql, err := mp.sqlGenerator.GenerateCreateTableSQL(table)
		if err != nil {
			return fmt.Errorf("failed to generate create table SQL for %s: %w", table.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeCreateTable,
			fmt.Sprintf("Create table %s", table.Name),
		)
		stmt.TableName = table.Name

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add create table statement: %w", err)
		}
	}

	return nil
}

// planTableModifications plans modifications to existing tables
func (mp *MigrationPlanner) planTableModifications(plan *MigrationPlan, tableDiffs []*schema.TableDiff) error {
	for _, tableDiff := range tableDiffs {
		// Plan constraint removals first
		if err := mp.planTableConstraintRemovals(plan, tableDiff); err != nil {
			return fmt.Errorf("failed to plan constraint removals for table %s: %w", tableDiff.TableName, err)
		}

		// Plan column removals
		if err := mp.planColumnRemovals(plan, tableDiff); err != nil {
			return fmt.Errorf("failed to plan column removals for table %s: %w", tableDiff.TableName, err)
		}

		// Plan column additions
		if err := mp.planColumnAdditions(plan, tableDiff); err != nil {
			return fmt.Errorf("failed to plan column additions for table %s: %w", tableDiff.TableName, err)
		}

		// Plan column modifications
		if err := mp.planColumnModifications(plan, tableDiff); err != nil {
			return fmt.Errorf("failed to plan column modifications for table %s: %w", tableDiff.TableName, err)
		}

		// Plan constraint additions
		if err := mp.planTableConstraintAdditions(plan, tableDiff); err != nil {
			return fmt.Errorf("failed to plan constraint additions for table %s: %w", tableDiff.TableName, err)
		}
	}

	return nil
}

// planColumnRemovals plans the removal of columns
func (mp *MigrationPlanner) planColumnRemovals(plan *MigrationPlan, tableDiff *schema.TableDiff) error {
	for _, column := range tableDiff.RemovedColumns {
		sql, err := mp.sqlGenerator.GenerateDropColumnSQL(tableDiff.TableName, column)
		if err != nil {
			return fmt.Errorf("failed to generate drop column SQL: %w", err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeDropColumn,
			fmt.Sprintf("Drop column %s from table %s", column.Name, tableDiff.TableName),
		)
		stmt.TableName = tableDiff.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add drop column statement: %w", err)
		}

		plan.AddWarning(fmt.Sprintf("Dropping column '%s.%s' will permanently delete all data in the column",
			tableDiff.TableName, column.Name))
	}

	return nil
}

// planColumnAdditions plans the addition of new columns
func (mp *MigrationPlanner) planColumnAdditions(plan *MigrationPlan, tableDiff *schema.TableDiff) error {
	for _, column := range tableDiff.AddedColumns {
		sql, err := mp.sqlGenerator.GenerateAddColumnSQL(tableDiff.TableName, column)
		if err != nil {
			return fmt.Errorf("failed to generate add column SQL: %w", err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeAddColumn,
			fmt.Sprintf("Add column %s to table %s", column.Name, tableDiff.TableName),
		)
		stmt.TableName = tableDiff.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add column statement: %w", err)
		}

		// Add warning if column is NOT NULL without default
		if !column.IsNullable && column.DefaultValue == nil {
			plan.AddWarning(fmt.Sprintf("Adding NOT NULL column '%s.%s' without default value may fail if table contains data",
				tableDiff.TableName, column.Name))
		}
	}

	return nil
}

// planColumnModifications plans modifications to existing columns
func (mp *MigrationPlanner) planColumnModifications(plan *MigrationPlan, tableDiff *schema.TableDiff) error {
	for _, columnDiff := range tableDiff.ModifiedColumns {
		sql, err := mp.sqlGenerator.GenerateModifyColumnSQL(tableDiff.TableName, columnDiff)
		if err != nil {
			return fmt.Errorf("failed to generate modify column SQL: %w", err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeModifyColumn,
			fmt.Sprintf("Modify column %s in table %s", columnDiff.ColumnName, tableDiff.TableName),
		)
		stmt.TableName = tableDiff.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add modify column statement: %w", err)
		}

		// Add warnings for potentially destructive column modifications
		mp.addColumnModificationWarnings(plan, tableDiff.TableName, columnDiff)
	}

	return nil
}

// planIndexRemovals plans the removal of indexes
func (mp *MigrationPlanner) planIndexRemovals(plan *MigrationPlan, indexes []*schema.Index) error {
	for _, index := range indexes {
		// Skip primary key indexes as they're handled with table operations
		if index.IsPrimary {
			continue
		}

		sql, err := mp.sqlGenerator.GenerateDropIndexSQL(index)
		if err != nil {
			return fmt.Errorf("failed to generate drop index SQL for %s: %w", index.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeDropIndex,
			fmt.Sprintf("Drop index %s on table %s", index.Name, index.TableName),
		)
		stmt.TableName = index.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add drop index statement: %w", err)
		}
	}

	return nil
}

// planIndexAdditions plans the addition of new indexes
func (mp *MigrationPlanner) planIndexAdditions(plan *MigrationPlan, indexes []*schema.Index) error {
	for _, index := range indexes {
		// Skip primary key indexes as they're handled with table operations
		if index.IsPrimary {
			continue
		}

		sql, err := mp.sqlGenerator.GenerateCreateIndexSQL(index)
		if err != nil {
			return fmt.Errorf("failed to generate create index SQL for %s: %w", index.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeCreateIndex,
			fmt.Sprintf("Create index %s on table %s", index.Name, index.TableName),
		)
		stmt.TableName = index.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add create index statement: %w", err)
		}
	}

	return nil
}

// planConstraintRemovals plans the removal of constraints
func (mp *MigrationPlanner) planConstraintRemovals(plan *MigrationPlan, constraints []*schema.Constraint) error {
	for _, constraint := range constraints {
		sql, err := mp.sqlGenerator.GenerateDropConstraintSQL(constraint)
		if err != nil {
			return fmt.Errorf("failed to generate drop constraint SQL for %s: %w", constraint.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeDropConstraint,
			fmt.Sprintf("Drop %s constraint %s on table %s",
				strings.ToLower(string(constraint.Type)), constraint.Name, constraint.TableName),
		)
		stmt.TableName = constraint.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add drop constraint statement: %w", err)
		}
	}

	return nil
}

// planConstraintAdditions plans the addition of new constraints
func (mp *MigrationPlanner) planConstraintAdditions(plan *MigrationPlan, constraints []*schema.Constraint) error {
	for _, constraint := range constraints {
		sql, err := mp.sqlGenerator.GenerateAddConstraintSQL(constraint)
		if err != nil {
			return fmt.Errorf("failed to generate add constraint SQL for %s: %w", constraint.Name, err)
		}

		stmt := NewMigrationStatement(
			sql,
			StatementTypeAddConstraint,
			fmt.Sprintf("Add %s constraint %s to table %s",
				strings.ToLower(string(constraint.Type)), constraint.Name, constraint.TableName),
		)
		stmt.TableName = constraint.TableName

		if err := plan.AddStatement(*stmt); err != nil {
			return fmt.Errorf("failed to add constraint statement: %w", err)
		}
	}

	return nil
}

// planTableConstraintRemovals plans constraint removals for a specific table
func (mp *MigrationPlanner) planTableConstraintRemovals(plan *MigrationPlan, tableDiff *schema.TableDiff) error {
	return mp.planConstraintRemovals(plan, tableDiff.RemovedConstraints)
}

// planTableConstraintAdditions plans constraint additions for a specific table
func (mp *MigrationPlanner) planTableConstraintAdditions(plan *MigrationPlan, tableDiff *schema.TableDiff) error {
	return mp.planConstraintAdditions(plan, tableDiff.AddedConstraints)
}

// sortStatements sorts migration statements by execution order
func (mp *MigrationPlanner) sortStatements(plan *MigrationPlan) {
	sort.Slice(plan.Statements, func(i, j int) bool {
		orderI := plan.Statements[i].Type.GetExecutionOrder()
		orderJ := plan.Statements[j].Type.GetExecutionOrder()

		if orderI != orderJ {
			return orderI < orderJ
		}

		// For statements of the same type, sort by table name for consistency
		return plan.Statements[i].TableName < plan.Statements[j].TableName
	})
}

// addDestructiveWarnings adds warnings for destructive operations
func (mp *MigrationPlanner) addDestructiveWarnings(plan *MigrationPlan) {
	if plan.HasDestructiveOperations() {
		plan.AddWarning("This migration contains destructive operations that may result in data loss")
		plan.AddWarning("Please ensure you have a backup before proceeding")
	}
}

// addColumnModificationWarnings adds warnings for potentially problematic column modifications
func (mp *MigrationPlanner) addColumnModificationWarnings(plan *MigrationPlan, tableName string, columnDiff *schema.ColumnDiff) {
	oldCol := columnDiff.OldColumn
	newCol := columnDiff.NewColumn

	// Check for data type changes that might cause data loss
	if oldCol.DataType != newCol.DataType {
		plan.AddWarning(fmt.Sprintf("Changing data type of column '%s.%s' from %s to %s may cause data loss or conversion errors",
			tableName, columnDiff.ColumnName, oldCol.DataType, newCol.DataType))
	}

	// Check for nullability changes
	if oldCol.IsNullable && !newCol.IsNullable {
		plan.AddWarning(fmt.Sprintf("Changing column '%s.%s' from nullable to NOT NULL may fail if existing data contains NULL values",
			tableName, columnDiff.ColumnName))
	}

	// Check for default value changes
	if (oldCol.DefaultValue == nil) != (newCol.DefaultValue == nil) ||
		(oldCol.DefaultValue != nil && newCol.DefaultValue != nil && *oldCol.DefaultValue != *newCol.DefaultValue) {
		plan.AddWarning(fmt.Sprintf("Default value change for column '%s.%s' will only affect new rows",
			tableName, columnDiff.ColumnName))
	}
}
