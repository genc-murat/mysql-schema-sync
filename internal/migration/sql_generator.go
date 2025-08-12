package migration

import (
	"fmt"
	"strings"

	"mysql-schema-sync/internal/schema"
)

// SQLGenerator handles the generation of SQL statements for schema changes
type SQLGenerator struct{}

// NewSQLGenerator creates a new SQLGenerator instance
func NewSQLGenerator() *SQLGenerator {
	return &SQLGenerator{}
}

// GenerateCreateTableSQL generates SQL for creating a table
func (sg *SQLGenerator) GenerateCreateTableSQL(table *schema.Table) (string, error) {
	if table == nil {
		return "", fmt.Errorf("table cannot be nil")
	}

	if err := table.Validate(); err != nil {
		return "", fmt.Errorf("invalid table: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", table.Name))

	// Add columns
	columnDefs := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		colDef, err := sg.generateColumnDefinition(column)
		if err != nil {
			return "", fmt.Errorf("failed to generate column definition for %s: %w", column.Name, err)
		}
		columnDefs = append(columnDefs, colDef)
	}

	// Add primary key if exists
	primaryKey := table.GetPrimaryKey()
	if primaryKey != nil {
		pkDef := sg.generatePrimaryKeyDefinition(primaryKey)
		columnDefs = append(columnDefs, pkDef)
	}

	// Add unique constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == schema.ConstraintTypeUnique {
			uniqueDef := sg.generateUniqueConstraintDefinition(constraint)
			columnDefs = append(columnDefs, uniqueDef)
		}
	}

	builder.WriteString("  " + strings.Join(columnDefs, ",\n  "))
	builder.WriteString("\n)")

	return builder.String(), nil
}

// GenerateDropTableSQL generates SQL for dropping a table
func (sg *SQLGenerator) GenerateDropTableSQL(table *schema.Table) (string, error) {
	if table == nil {
		return "", fmt.Errorf("table cannot be nil")
	}

	return fmt.Sprintf("DROP TABLE `%s`", table.Name), nil
}

// GenerateAddColumnSQL generates SQL for adding a column
func (sg *SQLGenerator) GenerateAddColumnSQL(tableName string, column *schema.Column) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("table name cannot be empty")
	}

	if column == nil {
		return "", fmt.Errorf("column cannot be nil")
	}

	colDef, err := sg.generateColumnDefinition(column)
	if err != nil {
		return "", fmt.Errorf("failed to generate column definition: %w", err)
	}

	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", tableName, colDef), nil
}

// GenerateDropColumnSQL generates SQL for dropping a column
func (sg *SQLGenerator) GenerateDropColumnSQL(tableName string, column *schema.Column) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("table name cannot be empty")
	}

	if column == nil {
		return "", fmt.Errorf("column cannot be nil")
	}

	return fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", tableName, column.Name), nil
}

// GenerateModifyColumnSQL generates SQL for modifying a column
func (sg *SQLGenerator) GenerateModifyColumnSQL(tableName string, columnDiff *schema.ColumnDiff) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("table name cannot be empty")
	}

	if columnDiff == nil {
		return "", fmt.Errorf("column diff cannot be nil")
	}

	if columnDiff.NewColumn == nil {
		return "", fmt.Errorf("new column cannot be nil")
	}

	colDef, err := sg.generateColumnDefinition(columnDiff.NewColumn)
	if err != nil {
		return "", fmt.Errorf("failed to generate column definition: %w", err)
	}

	return fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", tableName, colDef), nil
}

// GenerateCreateIndexSQL generates SQL for creating an index
func (sg *SQLGenerator) GenerateCreateIndexSQL(index *schema.Index) (string, error) {
	if index == nil {
		return "", fmt.Errorf("index cannot be nil")
	}

	if err := index.Validate(); err != nil {
		return "", fmt.Errorf("invalid index: %w", err)
	}

	var builder strings.Builder

	// Handle different index types
	if index.IsUnique {
		builder.WriteString("CREATE UNIQUE INDEX ")
	} else {
		builder.WriteString("CREATE INDEX ")
	}

	builder.WriteString(fmt.Sprintf("`%s` ON `%s` (", index.Name, index.TableName))

	// Add column list
	quotedColumns := make([]string, len(index.Columns))
	for i, col := range index.Columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}
	builder.WriteString(strings.Join(quotedColumns, ", "))
	builder.WriteString(")")

	// Add index type if specified and not default
	if index.IndexType != "" && index.IndexType != "BTREE" {
		builder.WriteString(fmt.Sprintf(" USING %s", index.IndexType))
	}

	return builder.String(), nil
}

// GenerateDropIndexSQL generates SQL for dropping an index
func (sg *SQLGenerator) GenerateDropIndexSQL(index *schema.Index) (string, error) {
	if index == nil {
		return "", fmt.Errorf("index cannot be nil")
	}

	return fmt.Sprintf("DROP INDEX `%s` ON `%s`", index.Name, index.TableName), nil
}

// GenerateAddConstraintSQL generates SQL for adding a constraint
func (sg *SQLGenerator) GenerateAddConstraintSQL(constraint *schema.Constraint) (string, error) {
	if constraint == nil {
		return "", fmt.Errorf("constraint cannot be nil")
	}

	if err := constraint.Validate(); err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` ",
		constraint.TableName, constraint.Name))

	switch constraint.Type {
	case schema.ConstraintTypeForeignKey:
		return sg.generateAddForeignKeySQL(constraint)
	case schema.ConstraintTypeUnique:
		return sg.generateAddUniqueConstraintSQL(constraint)
	case schema.ConstraintTypeCheck:
		return sg.generateAddCheckConstraintSQL(constraint)
	default:
		return "", fmt.Errorf("unsupported constraint type: %s", constraint.Type)
	}
}

// GenerateDropConstraintSQL generates SQL for dropping a constraint
func (sg *SQLGenerator) GenerateDropConstraintSQL(constraint *schema.Constraint) (string, error) {
	if constraint == nil {
		return "", fmt.Errorf("constraint cannot be nil")
	}

	switch constraint.Type {
	case schema.ConstraintTypeForeignKey:
		return fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`",
			constraint.TableName, constraint.Name), nil
	case schema.ConstraintTypeUnique:
		return fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`",
			constraint.TableName, constraint.Name), nil
	case schema.ConstraintTypeCheck:
		return fmt.Sprintf("ALTER TABLE `%s` DROP CHECK `%s`",
			constraint.TableName, constraint.Name), nil
	default:
		return "", fmt.Errorf("unsupported constraint type: %s", constraint.Type)
	}
}

// Helper methods for generating SQL components

// generateColumnDefinition generates the SQL definition for a column
func (sg *SQLGenerator) generateColumnDefinition(column *schema.Column) (string, error) {
	if column == nil {
		return "", fmt.Errorf("column cannot be nil")
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("`%s` %s", column.Name, column.DataType))

	// Add nullability
	if !column.IsNullable {
		builder.WriteString(" NOT NULL")
	} else {
		builder.WriteString(" NULL")
	}

	// Add default value
	if column.DefaultValue != nil {
		defaultVal := *column.DefaultValue
		// Handle special MySQL default values
		if strings.ToUpper(defaultVal) == "CURRENT_TIMESTAMP" ||
			strings.ToUpper(defaultVal) == "NOW()" ||
			strings.ToUpper(defaultVal) == "NULL" {
			builder.WriteString(fmt.Sprintf(" DEFAULT %s", defaultVal))
		} else {
			// Quote string defaults
			builder.WriteString(fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(defaultVal, "'", "''")))
		}
	}

	// Add extra attributes (AUTO_INCREMENT, etc.)
	if column.Extra != "" {
		builder.WriteString(fmt.Sprintf(" %s", column.Extra))
	}

	return builder.String(), nil
}

// generatePrimaryKeyDefinition generates the SQL definition for a primary key
func (sg *SQLGenerator) generatePrimaryKeyDefinition(primaryKey *schema.Index) string {
	quotedColumns := make([]string, len(primaryKey.Columns))
	for i, col := range primaryKey.Columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}
	return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(quotedColumns, ", "))
}

// generateUniqueConstraintDefinition generates the SQL definition for a unique constraint
func (sg *SQLGenerator) generateUniqueConstraintDefinition(constraint *schema.Constraint) string {
	quotedColumns := make([]string, len(constraint.Columns))
	for i, col := range constraint.Columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}
	return fmt.Sprintf("UNIQUE KEY `%s` (%s)", constraint.Name, strings.Join(quotedColumns, ", "))
}

// generateAddForeignKeySQL generates SQL for adding a foreign key constraint
func (sg *SQLGenerator) generateAddForeignKeySQL(constraint *schema.Constraint) (string, error) {
	quotedColumns := make([]string, len(constraint.Columns))
	for i, col := range constraint.Columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}

	quotedRefColumns := make([]string, len(constraint.ReferencedColumns))
	for i, col := range constraint.ReferencedColumns {
		quotedRefColumns[i] = fmt.Sprintf("`%s`", col)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (%s) REFERENCES `%s` (%s)",
		constraint.TableName,
		constraint.Name,
		strings.Join(quotedColumns, ", "),
		constraint.ReferencedTable,
		strings.Join(quotedRefColumns, ", ")))

	// Add ON UPDATE and ON DELETE clauses
	if constraint.OnUpdate != "" {
		builder.WriteString(fmt.Sprintf(" ON UPDATE %s", constraint.OnUpdate))
	}

	if constraint.OnDelete != "" {
		builder.WriteString(fmt.Sprintf(" ON DELETE %s", constraint.OnDelete))
	}

	return builder.String(), nil
}

// generateAddUniqueConstraintSQL generates SQL for adding a unique constraint
func (sg *SQLGenerator) generateAddUniqueConstraintSQL(constraint *schema.Constraint) (string, error) {
	quotedColumns := make([]string, len(constraint.Columns))
	for i, col := range constraint.Columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}

	return fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` UNIQUE (%s)",
		constraint.TableName,
		constraint.Name,
		strings.Join(quotedColumns, ", ")), nil
}

// generateAddCheckConstraintSQL generates SQL for adding a check constraint
func (sg *SQLGenerator) generateAddCheckConstraintSQL(constraint *schema.Constraint) (string, error) {
	return fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` CHECK (%s)",
		constraint.TableName,
		constraint.Name,
		constraint.CheckExpression), nil
}
