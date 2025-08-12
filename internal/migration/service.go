package migration

import (
	"fmt"
	"time"

	"mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/schema"
)

// MigrationService provides high-level migration operations
type MigrationService interface {
	PlanMigration(diff *schema.SchemaDiff) (*MigrationPlan, error)
	GenerateSQL(diff *schema.SchemaDiff) ([]string, error)
	ValidatePlan(plan *MigrationPlan) error

	// SQL generation methods for specific operations
	GenerateCreateTableSQL(table *schema.Table) (string, error)
	GenerateDropTableSQL(table *schema.Table) (string, error)
	GenerateAddColumnSQL(tableName string, column *schema.Column) (string, error)
	GenerateDropColumnSQL(tableName string, column *schema.Column) (string, error)
	GenerateModifyColumnSQL(tableName string, columnDiff *schema.ColumnDiff) (string, error)
	GenerateCreateIndexSQL(index *schema.Index) (string, error)
	GenerateDropIndexSQL(index *schema.Index) (string, error)
	GenerateAddConstraintSQL(constraint *schema.Constraint) (string, error)
	GenerateDropConstraintSQL(constraint *schema.Constraint) (string, error)
	GetSQLForStatementType(stmtType StatementType, tableName string, object interface{}) (string, error)
}

// migrationService implements the MigrationService interface
type migrationService struct {
	planner   *MigrationPlanner
	generator *SQLGenerator
	logger    *logging.Logger
}

// NewMigrationService creates a new MigrationService instance
func NewMigrationService() MigrationService {
	return &migrationService{
		planner:   NewMigrationPlanner(),
		generator: NewSQLGenerator(),
		logger:    logging.NewDefaultLogger(),
	}
}

// NewMigrationServiceWithLogger creates a new MigrationService instance with a custom logger
func NewMigrationServiceWithLogger(logger *logging.Logger) MigrationService {
	return &migrationService{
		planner:   NewMigrationPlanner(),
		generator: NewSQLGenerator(),
		logger:    logger,
	}
}

// PlanMigration creates a migration plan from schema differences
func (ms *migrationService) PlanMigration(diff *schema.SchemaDiff) (*MigrationPlan, error) {
	if diff == nil {
		return nil, errors.NewAppError(errors.ErrorTypeValidation, "schema diff cannot be nil", nil)
	}

	startTime := time.Now()
	finishLog := ms.logger.LogOperationStart("migration_planning", map[string]interface{}{
		"added_tables":    len(diff.AddedTables),
		"removed_tables":  len(diff.RemovedTables),
		"modified_tables": len(diff.ModifiedTables),
	})

	plan, err := ms.planner.PlanMigration(diff)
	duration := time.Since(startTime)

	if err != nil {
		finishLog(err)
		return nil, errors.WrapError(err, "failed to create migration plan")
	}

	finishLog(nil)
	ms.logger.WithFields(map[string]interface{}{
		"statement_count": len(plan.Statements),
		"warning_count":   len(plan.Warnings),
		"duration":        duration.String(),
	}).Info("Migration plan created successfully")

	return plan, nil
}

// GenerateSQL generates SQL statements from schema differences
func (ms *migrationService) GenerateSQL(diff *schema.SchemaDiff) ([]string, error) {
	if diff == nil {
		return nil, errors.NewAppError(errors.ErrorTypeValidation, "schema diff cannot be nil", nil)
	}

	// Create migration plan
	plan, err := ms.planner.PlanMigration(diff)
	if err != nil {
		return nil, errors.WrapError(err, "failed to create migration plan")
	}

	// Extract SQL statements from the plan
	sqlStatements := make([]string, len(plan.Statements))
	for i, stmt := range plan.Statements {
		sqlStatements[i] = stmt.SQL
	}

	ms.logger.WithField("statement_count", len(sqlStatements)).Debug("Generated SQL statements from migration plan")
	return sqlStatements, nil
}

// ValidatePlan validates a migration plan
func (ms *migrationService) ValidatePlan(plan *MigrationPlan) error {
	if plan == nil {
		return errors.NewAppError(errors.ErrorTypeValidation, "migration plan cannot be nil", nil)
	}

	ms.logger.WithField("statement_count", len(plan.Statements)).Debug("Validating migration plan")

	if err := plan.Validate(); err != nil {
		ms.logger.WithField("error", err.Error()).Error("Migration plan validation failed")
		return errors.WrapError(err, "migration plan validation failed")
	}

	ms.logger.Debug("Migration plan validation successful")
	return nil
}

// Additional utility methods for specific SQL generation

// GenerateCreateTableSQL generates SQL for creating a table
func (ms *migrationService) GenerateCreateTableSQL(table *schema.Table) (string, error) {
	return ms.generator.GenerateCreateTableSQL(table)
}

// GenerateDropTableSQL generates SQL for dropping a table
func (ms *migrationService) GenerateDropTableSQL(table *schema.Table) (string, error) {
	return ms.generator.GenerateDropTableSQL(table)
}

// GenerateAddColumnSQL generates SQL for adding a column
func (ms *migrationService) GenerateAddColumnSQL(tableName string, column *schema.Column) (string, error) {
	return ms.generator.GenerateAddColumnSQL(tableName, column)
}

// GenerateDropColumnSQL generates SQL for dropping a column
func (ms *migrationService) GenerateDropColumnSQL(tableName string, column *schema.Column) (string, error) {
	return ms.generator.GenerateDropColumnSQL(tableName, column)
}

// GenerateModifyColumnSQL generates SQL for modifying a column
func (ms *migrationService) GenerateModifyColumnSQL(tableName string, columnDiff *schema.ColumnDiff) (string, error) {
	return ms.generator.GenerateModifyColumnSQL(tableName, columnDiff)
}

// GenerateCreateIndexSQL generates SQL for creating an index
func (ms *migrationService) GenerateCreateIndexSQL(index *schema.Index) (string, error) {
	return ms.generator.GenerateCreateIndexSQL(index)
}

// GenerateDropIndexSQL generates SQL for dropping an index
func (ms *migrationService) GenerateDropIndexSQL(index *schema.Index) (string, error) {
	return ms.generator.GenerateDropIndexSQL(index)
}

// GenerateAddConstraintSQL generates SQL for adding a constraint
func (ms *migrationService) GenerateAddConstraintSQL(constraint *schema.Constraint) (string, error) {
	return ms.generator.GenerateAddConstraintSQL(constraint)
}

// GenerateDropConstraintSQL generates SQL for dropping a constraint
func (ms *migrationService) GenerateDropConstraintSQL(constraint *schema.Constraint) (string, error) {
	return ms.generator.GenerateDropConstraintSQL(constraint)
}

// GetSQLForStatementType generates SQL for a specific statement type and object
func (ms *migrationService) GetSQLForStatementType(stmtType StatementType, tableName string, object interface{}) (string, error) {
	switch stmtType {
	case StatementTypeCreateTable:
		if table, ok := object.(*schema.Table); ok {
			return ms.generator.GenerateCreateTableSQL(table)
		}
		return "", fmt.Errorf("invalid object type for CREATE TABLE: expected *schema.Table")

	case StatementTypeDropTable:
		if table, ok := object.(*schema.Table); ok {
			return ms.generator.GenerateDropTableSQL(table)
		}
		return "", fmt.Errorf("invalid object type for DROP TABLE: expected *schema.Table")

	case StatementTypeAddColumn:
		if column, ok := object.(*schema.Column); ok {
			return ms.generator.GenerateAddColumnSQL(tableName, column)
		}
		return "", fmt.Errorf("invalid object type for ADD COLUMN: expected *schema.Column")

	case StatementTypeDropColumn:
		if column, ok := object.(*schema.Column); ok {
			return ms.generator.GenerateDropColumnSQL(tableName, column)
		}
		return "", fmt.Errorf("invalid object type for DROP COLUMN: expected *schema.Column")

	case StatementTypeModifyColumn:
		if columnDiff, ok := object.(*schema.ColumnDiff); ok {
			return ms.generator.GenerateModifyColumnSQL(tableName, columnDiff)
		}
		return "", fmt.Errorf("invalid object type for MODIFY COLUMN: expected *schema.ColumnDiff")

	case StatementTypeCreateIndex:
		if index, ok := object.(*schema.Index); ok {
			return ms.generator.GenerateCreateIndexSQL(index)
		}
		return "", fmt.Errorf("invalid object type for CREATE INDEX: expected *schema.Index")

	case StatementTypeDropIndex:
		if index, ok := object.(*schema.Index); ok {
			return ms.generator.GenerateDropIndexSQL(index)
		}
		return "", fmt.Errorf("invalid object type for DROP INDEX: expected *schema.Index")

	case StatementTypeAddConstraint:
		if constraint, ok := object.(*schema.Constraint); ok {
			return ms.generator.GenerateAddConstraintSQL(constraint)
		}
		return "", fmt.Errorf("invalid object type for ADD CONSTRAINT: expected *schema.Constraint")

	case StatementTypeDropConstraint:
		if constraint, ok := object.(*schema.Constraint); ok {
			return ms.generator.GenerateDropConstraintSQL(constraint)
		}
		return "", fmt.Errorf("invalid object type for DROP CONSTRAINT: expected *schema.Constraint")

	default:
		return "", fmt.Errorf("unsupported statement type: %s", stmtType)
	}
}
