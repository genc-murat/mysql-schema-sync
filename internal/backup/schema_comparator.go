package backup

import (
	"fmt"
	"strings"

	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

// SchemaComparator provides functionality to compare schemas and generate migration statements
type SchemaComparator struct{}

// NewSchemaComparator creates a new SchemaComparator instance
func NewSchemaComparator() *SchemaComparator {
	return &SchemaComparator{}
}

// CompareSchemas compares two schemas and returns the differences
func (sc *SchemaComparator) CompareSchemas(current, target *schema.Schema) (*schema.SchemaDiff, error) {
	if current == nil || target == nil {
		return nil, fmt.Errorf("schemas cannot be nil")
	}

	diff := &schema.SchemaDiff{
		AddedTables:        make([]*schema.Table, 0),
		RemovedTables:      make([]*schema.Table, 0),
		ModifiedTables:     make([]*schema.TableDiff, 0),
		AddedIndexes:       make([]*schema.Index, 0),
		RemovedIndexes:     make([]*schema.Index, 0),
		AddedConstraints:   make([]*schema.Constraint, 0),
		RemovedConstraints: make([]*schema.Constraint, 0),
	}

	// Find added and removed tables
	for tableName, currentTable := range current.Tables {
		if _, exists := target.Tables[tableName]; !exists {
			diff.RemovedTables = append(diff.RemovedTables, currentTable)
		}
	}

	for tableName, targetTable := range target.Tables {
		if _, exists := current.Tables[tableName]; !exists {
			diff.AddedTables = append(diff.AddedTables, targetTable)
		}
	}

	// Find modified tables
	for tableName, currentTable := range current.Tables {
		if targetTable, exists := target.Tables[tableName]; exists {
			tableDiff := sc.compareTables(currentTable, targetTable)
			if sc.hasTableChanges(tableDiff) {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
		}
	}

	// Find added and removed indexes at schema level
	for indexName, currentIndex := range current.Indexes {
		if _, exists := target.Indexes[indexName]; !exists {
			diff.RemovedIndexes = append(diff.RemovedIndexes, currentIndex)
		}
	}

	for indexName, targetIndex := range target.Indexes {
		if _, exists := current.Indexes[indexName]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, targetIndex)
		}
	}

	return diff, nil
}

// GenerateRollbackStatements generates rollback statements from current to target schema
func (sc *SchemaComparator) GenerateRollbackStatements(current, target *schema.Schema) ([]migration.MigrationStatement, []string, error) {
	diff, err := sc.CompareSchemas(current, target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compare schemas: %w", err)
	}

	var statements []migration.MigrationStatement
	var warnings []string

	// Generate statements for removed tables (DROP TABLE)
	for _, table := range diff.RemovedTables {
		stmt := migration.MigrationStatement{
			SQL:           fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table.Name),
			Type:          migration.StatementTypeDropTable,
			Description:   fmt.Sprintf("Drop table %s (not present in target schema)", table.Name),
			IsDestructive: true,
			TableName:     table.Name,
		}
		statements = append(statements, stmt)
		warnings = append(warnings, fmt.Sprintf("Table %s will be dropped - this is destructive and will result in data loss", table.Name))
	}

	// Generate statements for added tables (CREATE TABLE)
	for _, table := range diff.AddedTables {
		createSQL, err := sc.generateCreateTableSQL(table)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate CREATE TABLE for %s: %w", table.Name, err)
		}

		stmt := migration.MigrationStatement{
			SQL:           createSQL,
			Type:          migration.StatementTypeCreateTable,
			Description:   fmt.Sprintf("Create table %s (present in target schema)", table.Name),
			IsDestructive: false,
			TableName:     table.Name,
		}
		statements = append(statements, stmt)
	}

	// Generate statements for modified tables
	for _, tableDiff := range diff.ModifiedTables {
		tableStatements, tableWarnings := sc.generateTableModificationStatements(tableDiff)
		statements = append(statements, tableStatements...)
		warnings = append(warnings, tableWarnings...)
	}

	// Generate statements for removed indexes
	for _, index := range diff.RemovedIndexes {
		if !index.IsPrimary { // Don't drop primary keys separately
			stmt := migration.MigrationStatement{
				SQL:           fmt.Sprintf("DROP INDEX `%s` ON `%s`", index.Name, index.TableName),
				Type:          migration.StatementTypeDropIndex,
				Description:   fmt.Sprintf("Drop index %s on table %s", index.Name, index.TableName),
				IsDestructive: true,
				TableName:     index.TableName,
			}
			statements = append(statements, stmt)
		}
	}

	// Generate statements for added indexes
	for _, index := range diff.AddedIndexes {
		if !index.IsPrimary { // Don't create primary keys separately
			createIndexSQL := sc.generateCreateIndexSQL(index)
			stmt := migration.MigrationStatement{
				SQL:           createIndexSQL,
				Type:          migration.StatementTypeCreateIndex,
				Description:   fmt.Sprintf("Create index %s on table %s", index.Name, index.TableName),
				IsDestructive: false,
				TableName:     index.TableName,
			}
			statements = append(statements, stmt)
		}
	}

	// Generate statements for removed constraints
	for _, constraint := range diff.RemovedConstraints {
		stmt := migration.MigrationStatement{
			SQL:           fmt.Sprintf("ALTER TABLE `%s` DROP CONSTRAINT `%s`", constraint.TableName, constraint.Name),
			Type:          migration.StatementTypeDropConstraint,
			Description:   fmt.Sprintf("Drop constraint %s from table %s", constraint.Name, constraint.TableName),
			IsDestructive: true,
			TableName:     constraint.TableName,
		}
		statements = append(statements, stmt)
	}

	// Generate statements for added constraints
	for _, constraint := range diff.AddedConstraints {
		addConstraintSQL := sc.generateAddConstraintSQL(constraint)
		stmt := migration.MigrationStatement{
			SQL:           addConstraintSQL,
			Type:          migration.StatementTypeAddConstraint,
			Description:   fmt.Sprintf("Add constraint %s to table %s", constraint.Name, constraint.TableName),
			IsDestructive: false,
			TableName:     constraint.TableName,
		}
		statements = append(statements, stmt)
	}

	// Sort statements by execution order
	sc.sortStatementsByExecutionOrder(statements)

	return statements, warnings, nil
}

// compareTables compares two tables and returns the differences
func (sc *SchemaComparator) compareTables(current, target *schema.Table) *schema.TableDiff {
	diff := &schema.TableDiff{
		TableName:          current.Name,
		AddedColumns:       make([]*schema.Column, 0),
		RemovedColumns:     make([]*schema.Column, 0),
		ModifiedColumns:    make([]*schema.ColumnDiff, 0),
		AddedConstraints:   make([]*schema.Constraint, 0),
		RemovedConstraints: make([]*schema.Constraint, 0),
	}

	// Find added and removed columns
	for columnName, currentColumn := range current.Columns {
		if _, exists := target.Columns[columnName]; !exists {
			diff.RemovedColumns = append(diff.RemovedColumns, currentColumn)
		}
	}

	for columnName, targetColumn := range target.Columns {
		if _, exists := current.Columns[columnName]; !exists {
			diff.AddedColumns = append(diff.AddedColumns, targetColumn)
		}
	}

	// Find modified columns
	for columnName, currentColumn := range current.Columns {
		if targetColumn, exists := target.Columns[columnName]; exists {
			if sc.columnsAreDifferent(currentColumn, targetColumn) {
				diff.ModifiedColumns = append(diff.ModifiedColumns, &schema.ColumnDiff{
					ColumnName: columnName,
					OldColumn:  currentColumn,
					NewColumn:  targetColumn,
				})
			}
		}
	}

	// Find added and removed constraints
	for constraintName, currentConstraint := range current.Constraints {
		if _, exists := target.Constraints[constraintName]; !exists {
			diff.RemovedConstraints = append(diff.RemovedConstraints, currentConstraint)
		}
	}

	for constraintName, targetConstraint := range target.Constraints {
		if _, exists := current.Constraints[constraintName]; !exists {
			diff.AddedConstraints = append(diff.AddedConstraints, targetConstraint)
		}
	}

	return diff
}

// hasTableChanges checks if a table diff has any changes
func (sc *SchemaComparator) hasTableChanges(diff *schema.TableDiff) bool {
	return len(diff.AddedColumns) > 0 ||
		len(diff.RemovedColumns) > 0 ||
		len(diff.ModifiedColumns) > 0 ||
		len(diff.AddedConstraints) > 0 ||
		len(diff.RemovedConstraints) > 0
}

// columnsAreDifferent checks if two columns are different
func (sc *SchemaComparator) columnsAreDifferent(col1, col2 *schema.Column) bool {
	if col1.DataType != col2.DataType {
		return true
	}
	if col1.IsNullable != col2.IsNullable {
		return true
	}
	if (col1.DefaultValue == nil) != (col2.DefaultValue == nil) {
		return true
	}
	if col1.DefaultValue != nil && col2.DefaultValue != nil && *col1.DefaultValue != *col2.DefaultValue {
		return true
	}
	if col1.Extra != col2.Extra {
		return true
	}
	return false
}

// generateTableModificationStatements generates statements for table modifications
func (sc *SchemaComparator) generateTableModificationStatements(diff *schema.TableDiff) ([]migration.MigrationStatement, []string) {
	var statements []migration.MigrationStatement
	var warnings []string

	// Generate statements for removed columns
	for _, column := range diff.RemovedColumns {
		stmt := migration.MigrationStatement{
			SQL:           fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", diff.TableName, column.Name),
			Type:          migration.StatementTypeDropColumn,
			Description:   fmt.Sprintf("Drop column %s from table %s", column.Name, diff.TableName),
			IsDestructive: true,
			TableName:     diff.TableName,
		}
		statements = append(statements, stmt)
		warnings = append(warnings, fmt.Sprintf("Column %s.%s will be dropped - this will result in data loss", diff.TableName, column.Name))
	}

	// Generate statements for added columns
	for _, column := range diff.AddedColumns {
		columnDef := sc.generateColumnDefinition(column)
		stmt := migration.MigrationStatement{
			SQL:           fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", diff.TableName, columnDef),
			Type:          migration.StatementTypeAddColumn,
			Description:   fmt.Sprintf("Add column %s to table %s", column.Name, diff.TableName),
			IsDestructive: false,
			TableName:     diff.TableName,
		}
		statements = append(statements, stmt)
	}

	// Generate statements for modified columns
	for _, columnDiff := range diff.ModifiedColumns {
		columnDef := sc.generateColumnDefinition(columnDiff.NewColumn)
		stmt := migration.MigrationStatement{
			SQL:           fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", diff.TableName, columnDef),
			Type:          migration.StatementTypeModifyColumn,
			Description:   fmt.Sprintf("Modify column %s in table %s", columnDiff.ColumnName, diff.TableName),
			IsDestructive: false,
			TableName:     diff.TableName,
		}
		statements = append(statements, stmt)
		warnings = append(warnings, fmt.Sprintf("Column %s.%s will be modified - this may affect existing data", diff.TableName, columnDiff.ColumnName))
	}

	return statements, warnings
}

// generateCreateTableSQL generates a CREATE TABLE SQL statement
func (sc *SchemaComparator) generateCreateTableSQL(table *schema.Table) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", table.Name))

	// Add columns
	columnDefs := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		columnDef := "  " + sc.generateColumnDefinition(column)
		columnDefs = append(columnDefs, columnDef)
	}

	builder.WriteString(strings.Join(columnDefs, ",\n"))

	// Add primary key if exists
	if primaryKey := table.GetPrimaryKey(); primaryKey != nil {
		builder.WriteString(",\n")
		builder.WriteString(fmt.Sprintf("  PRIMARY KEY (`%s`)", strings.Join(primaryKey.Columns, "`, `")))
	}

	// Add unique constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == schema.ConstraintTypeUnique {
			builder.WriteString(",\n")
			builder.WriteString(fmt.Sprintf("  UNIQUE KEY `%s` (`%s`)", constraint.Name, strings.Join(constraint.Columns, "`, `")))
		}
	}

	builder.WriteString("\n)")

	return builder.String(), nil
}

// generateColumnDefinition generates a column definition string
func (sc *SchemaComparator) generateColumnDefinition(column *schema.Column) string {
	def := fmt.Sprintf("`%s` %s", column.Name, column.DataType)

	if !column.IsNullable {
		def += " NOT NULL"
	}

	if column.DefaultValue != nil {
		def += fmt.Sprintf(" DEFAULT %s", *column.DefaultValue)
	}

	if column.Extra != "" {
		def += " " + column.Extra
	}

	return def
}

// generateCreateIndexSQL generates a CREATE INDEX SQL statement
func (sc *SchemaComparator) generateCreateIndexSQL(index *schema.Index) string {
	indexType := ""
	if index.IsUnique {
		indexType = "UNIQUE "
	}

	return fmt.Sprintf("CREATE %sINDEX `%s` ON `%s` (`%s`)",
		indexType, index.Name, index.TableName, strings.Join(index.Columns, "`, `"))
}

// generateAddConstraintSQL generates an ADD CONSTRAINT SQL statement
func (sc *SchemaComparator) generateAddConstraintSQL(constraint *schema.Constraint) string {
	switch constraint.Type {
	case schema.ConstraintTypeForeignKey:
		sql := fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
			constraint.TableName, constraint.Name,
			strings.Join(constraint.Columns, "`, `"),
			constraint.ReferencedTable,
			strings.Join(constraint.ReferencedColumns, "`, `"))

		if constraint.OnUpdate != "" {
			sql += " ON UPDATE " + constraint.OnUpdate
		}
		if constraint.OnDelete != "" {
			sql += " ON DELETE " + constraint.OnDelete
		}

		return sql

	case schema.ConstraintTypeUnique:
		return fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` UNIQUE (`%s`)",
			constraint.TableName, constraint.Name, strings.Join(constraint.Columns, "`, `"))

	case schema.ConstraintTypeCheck:
		return fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` CHECK (%s)",
			constraint.TableName, constraint.Name, constraint.CheckExpression)

	default:
		return fmt.Sprintf("-- Unknown constraint type: %s", constraint.Type)
	}
}

// sortStatementsByExecutionOrder sorts statements by their execution order
func (sc *SchemaComparator) sortStatementsByExecutionOrder(statements []migration.MigrationStatement) {
	// Use the same sorting logic as in the migration package
	for i := 0; i < len(statements); i++ {
		for j := i + 1; j < len(statements); j++ {
			if statements[i].Type.GetExecutionOrder() > statements[j].Type.GetExecutionOrder() {
				statements[i], statements[j] = statements[j], statements[i]
			}
		}
	}
}
