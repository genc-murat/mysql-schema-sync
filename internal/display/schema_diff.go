package display

import (
	"fmt"
	"strings"

	"mysql-schema-sync/internal/schema"
)

// ChangeType represents the type of change in a schema diff
type ChangeType int

const (
	ChangeAdded ChangeType = iota
	ChangeRemoved
	ChangeModified
)

// String returns the string representation of a ChangeType
func (ct ChangeType) String() string {
	switch ct {
	case ChangeAdded:
		return "ADDED"
	case ChangeRemoved:
		return "REMOVED"
	case ChangeModified:
		return "MODIFIED"
	default:
		return "UNKNOWN"
	}
}

// SchemaDiffPresenter handles the presentation of schema differences
type SchemaDiffPresenter struct {
	colorSystem ColorSystem
	iconSystem  IconSystem
	theme       ColorTheme
}

// NewSchemaDiffPresenter creates a new schema diff presenter
func NewSchemaDiffPresenter(colorSystem ColorSystem, iconSystem IconSystem, theme ColorTheme) *SchemaDiffPresenter {
	return &SchemaDiffPresenter{
		colorSystem: colorSystem,
		iconSystem:  iconSystem,
		theme:       theme,
	}
}

// FormatSchemaDiff formats a complete schema diff using tables
func (sdp *SchemaDiffPresenter) FormatSchemaDiff(diff *schema.SchemaDiff) string {
	var result strings.Builder

	// Summary section
	result.WriteString(sdp.formatSummary(diff))
	result.WriteString("\n")

	// Tables section
	if sdp.hasTableChanges(diff) {
		result.WriteString(sdp.formatTableChanges(diff))
		result.WriteString("\n")
	}

	// Indexes section
	if sdp.hasIndexChanges(diff) {
		result.WriteString(sdp.formatIndexChanges(diff))
		result.WriteString("\n")
	}

	// Constraints section
	if sdp.hasConstraintChanges(diff) {
		result.WriteString(sdp.formatConstraintChanges(diff))
		result.WriteString("\n")
	}

	return result.String()
}

// formatSummary creates a summary table of all changes
func (sdp *SchemaDiffPresenter) formatSummary(diff *schema.SchemaDiff) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(RoundedTableStyle)
	formatter.SetHeaders([]string{"Change Type", "Count", "Details"})

	// Count changes
	addedTables := len(diff.AddedTables)
	removedTables := len(diff.RemovedTables)
	modifiedTables := len(diff.ModifiedTables)
	addedIndexes := len(diff.AddedIndexes)
	removedIndexes := len(diff.RemovedIndexes)
	addedConstraints := len(diff.AddedConstraints)
	removedConstraints := len(diff.RemovedConstraints)

	// Add summary rows
	if addedTables > 0 {
		icon := sdp.getChangeIcon(ChangeAdded)
		formatter.AddRow([]string{
			icon + " Tables Added",
			fmt.Sprintf("%d", addedTables),
			sdp.formatTableNames(diff.AddedTables),
		})
	}

	if removedTables > 0 {
		icon := sdp.getChangeIcon(ChangeRemoved)
		formatter.AddRow([]string{
			icon + " Tables Removed",
			fmt.Sprintf("%d", removedTables),
			sdp.formatTableNames(diff.RemovedTables),
		})
	}

	if modifiedTables > 0 {
		icon := sdp.getChangeIcon(ChangeModified)
		formatter.AddRow([]string{
			icon + " Tables Modified",
			fmt.Sprintf("%d", modifiedTables),
			sdp.formatModifiedTableNames(diff.ModifiedTables),
		})
	}

	if addedIndexes > 0 {
		icon := sdp.getChangeIcon(ChangeAdded)
		formatter.AddRow([]string{
			icon + " Indexes Added",
			fmt.Sprintf("%d", addedIndexes),
			sdp.formatIndexNames(diff.AddedIndexes),
		})
	}

	if removedIndexes > 0 {
		icon := sdp.getChangeIcon(ChangeRemoved)
		formatter.AddRow([]string{
			icon + " Indexes Removed",
			fmt.Sprintf("%d", removedIndexes),
			sdp.formatIndexNames(diff.RemovedIndexes),
		})
	}

	if addedConstraints > 0 {
		icon := sdp.getChangeIcon(ChangeAdded)
		formatter.AddRow([]string{
			icon + " Constraints Added",
			fmt.Sprintf("%d", addedConstraints),
			sdp.formatConstraintNames(diff.AddedConstraints),
		})
	}

	if removedConstraints > 0 {
		icon := sdp.getChangeIcon(ChangeRemoved)
		formatter.AddRow([]string{
			icon + " Constraints Removed",
			fmt.Sprintf("%d", removedConstraints),
			sdp.formatConstraintNames(diff.RemovedConstraints),
		})
	}

	if formatter.(*tableFormatter).rows == nil || len(formatter.(*tableFormatter).rows) == 0 {
		return sdp.colorizeText("No schema changes detected.", sdp.theme.Success)
	}

	return "Schema Changes Summary:\n" + formatter.Render()
}

// formatTableChanges formats table-level changes
func (sdp *SchemaDiffPresenter) formatTableChanges(diff *schema.SchemaDiff) string {
	var result strings.Builder

	result.WriteString("Table Changes:\n")

	// Added tables
	if len(diff.AddedTables) > 0 {
		result.WriteString(sdp.formatAddedTables(diff.AddedTables))
		result.WriteString("\n")
	}

	// Removed tables
	if len(diff.RemovedTables) > 0 {
		result.WriteString(sdp.formatRemovedTables(diff.RemovedTables))
		result.WriteString("\n")
	}

	// Modified tables
	if len(diff.ModifiedTables) > 0 {
		result.WriteString(sdp.formatModifiedTables(diff.ModifiedTables))
		result.WriteString("\n")
	}

	return result.String()
}

// formatAddedTables formats added tables
func (sdp *SchemaDiffPresenter) formatAddedTables(tables []*schema.Table) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(DefaultTableStyle)
	formatter.SetHeaders([]string{"Table Name", "Columns", "Indexes", "Constraints"})

	for _, table := range tables {
		icon := sdp.getChangeIcon(ChangeAdded)
		formatter.AddRow([]string{
			icon + " " + table.Name,
			fmt.Sprintf("%d", len(table.Columns)),
			fmt.Sprintf("%d", len(table.Indexes)),
			fmt.Sprintf("%d", len(table.Constraints)),
		})
	}

	return "Added Tables:\n" + formatter.Render()
}

// formatRemovedTables formats removed tables
func (sdp *SchemaDiffPresenter) formatRemovedTables(tables []*schema.Table) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(DefaultTableStyle)
	formatter.SetHeaders([]string{"Table Name", "Columns", "Indexes", "Constraints"})

	for _, table := range tables {
		icon := sdp.getChangeIcon(ChangeRemoved)
		formatter.AddRow([]string{
			icon + " " + table.Name,
			fmt.Sprintf("%d", len(table.Columns)),
			fmt.Sprintf("%d", len(table.Indexes)),
			fmt.Sprintf("%d", len(table.Constraints)),
		})
	}

	return "Removed Tables:\n" + formatter.Render()
}

// formatModifiedTables formats modified tables with detailed column changes
func (sdp *SchemaDiffPresenter) formatModifiedTables(tableDiffs []*schema.TableDiff) string {
	var result strings.Builder

	result.WriteString("Modified Tables:\n")

	for _, tableDiff := range tableDiffs {
		result.WriteString(sdp.formatTableDiff(tableDiff))
		result.WriteString("\n")
	}

	return result.String()
}

// formatTableDiff formats a single table diff with hierarchical display
func (sdp *SchemaDiffPresenter) formatTableDiff(tableDiff *schema.TableDiff) string {
	var result strings.Builder

	// Table header
	tableIcon := sdp.iconSystem.RenderIcon("table")
	result.WriteString(fmt.Sprintf("%s Table: %s\n", tableIcon, tableDiff.TableName))

	// Column changes
	if sdp.hasColumnChanges(tableDiff) {
		result.WriteString(sdp.formatColumnChanges(tableDiff))
		result.WriteString("\n")
	}

	// Constraint changes
	if len(tableDiff.AddedConstraints) > 0 || len(tableDiff.RemovedConstraints) > 0 {
		result.WriteString(sdp.formatTableConstraintChanges(tableDiff))
		result.WriteString("\n")
	}

	return result.String()
}

// formatColumnChanges formats column changes within a table
func (sdp *SchemaDiffPresenter) formatColumnChanges(tableDiff *schema.TableDiff) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(CompactTableStyle)
	formatter.SetHeaders([]string{"Change", "Column", "Type", "Nullable", "Default", "Extra"})

	// Added columns
	for _, column := range tableDiff.AddedColumns {
		icon := sdp.getChangeIcon(ChangeAdded)
		defaultVal := ""
		if column.DefaultValue != nil {
			defaultVal = *column.DefaultValue
		}
		formatter.AddRow([]string{
			icon + " ADD",
			"  " + column.Name, // Indent to show hierarchy
			column.DataType,
			sdp.formatNullable(column.IsNullable),
			defaultVal,
			column.Extra,
		})
	}

	// Removed columns
	for _, column := range tableDiff.RemovedColumns {
		icon := sdp.getChangeIcon(ChangeRemoved)
		defaultVal := ""
		if column.DefaultValue != nil {
			defaultVal = *column.DefaultValue
		}
		formatter.AddRow([]string{
			icon + " DROP",
			"  " + column.Name, // Indent to show hierarchy
			column.DataType,
			sdp.formatNullable(column.IsNullable),
			defaultVal,
			column.Extra,
		})
	}

	// Modified columns
	for _, columnDiff := range tableDiff.ModifiedColumns {
		icon := sdp.getChangeIcon(ChangeModified)

		// Show old values
		oldDefault := ""
		if columnDiff.OldColumn.DefaultValue != nil {
			oldDefault = *columnDiff.OldColumn.DefaultValue
		}
		formatter.AddRow([]string{
			icon + " OLD",
			"  " + columnDiff.ColumnName,
			columnDiff.OldColumn.DataType,
			sdp.formatNullable(columnDiff.OldColumn.IsNullable),
			oldDefault,
			columnDiff.OldColumn.Extra,
		})

		// Show new values
		newDefault := ""
		if columnDiff.NewColumn.DefaultValue != nil {
			newDefault = *columnDiff.NewColumn.DefaultValue
		}
		formatter.AddRow([]string{
			icon + " NEW",
			"  " + columnDiff.ColumnName,
			columnDiff.NewColumn.DataType,
			sdp.formatNullable(columnDiff.NewColumn.IsNullable),
			newDefault,
			columnDiff.NewColumn.Extra,
		})

		// Add separator between column modifications
		formatter.AddSeparator()
	}

	return "  Column Changes:\n" + sdp.indentText(formatter.Render(), "  ")
}

// formatIndexChanges formats index changes
func (sdp *SchemaDiffPresenter) formatIndexChanges(diff *schema.SchemaDiff) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(DefaultTableStyle)
	formatter.SetHeaders([]string{"Change", "Index Name", "Table", "Columns", "Type", "Unique"})

	// Added indexes
	for _, index := range diff.AddedIndexes {
		icon := sdp.getChangeIcon(ChangeAdded)
		formatter.AddRow([]string{
			icon + " ADD",
			index.Name,
			index.TableName,
			strings.Join(index.Columns, ", "),
			index.IndexType,
			sdp.formatBoolean(index.IsUnique),
		})
	}

	// Removed indexes
	for _, index := range diff.RemovedIndexes {
		icon := sdp.getChangeIcon(ChangeRemoved)
		formatter.AddRow([]string{
			icon + " DROP",
			index.Name,
			index.TableName,
			strings.Join(index.Columns, ", "),
			index.IndexType,
			sdp.formatBoolean(index.IsUnique),
		})
	}

	return "Index Changes:\n" + formatter.Render()
}

// formatConstraintChanges formats constraint changes
func (sdp *SchemaDiffPresenter) formatConstraintChanges(diff *schema.SchemaDiff) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(DefaultTableStyle)
	formatter.SetHeaders([]string{"Change", "Constraint Name", "Table", "Type", "Columns", "References"})

	// Added constraints
	for _, constraint := range diff.AddedConstraints {
		icon := sdp.getChangeIcon(ChangeAdded)
		references := ""
		if constraint.Type == schema.ConstraintTypeForeignKey {
			references = fmt.Sprintf("%s(%s)", constraint.ReferencedTable, strings.Join(constraint.ReferencedColumns, ", "))
		}
		formatter.AddRow([]string{
			icon + " ADD",
			constraint.Name,
			constraint.TableName,
			string(constraint.Type),
			strings.Join(constraint.Columns, ", "),
			references,
		})
	}

	// Removed constraints
	for _, constraint := range diff.RemovedConstraints {
		icon := sdp.getChangeIcon(ChangeRemoved)
		references := ""
		if constraint.Type == schema.ConstraintTypeForeignKey {
			references = fmt.Sprintf("%s(%s)", constraint.ReferencedTable, strings.Join(constraint.ReferencedColumns, ", "))
		}
		formatter.AddRow([]string{
			icon + " DROP",
			constraint.Name,
			constraint.TableName,
			string(constraint.Type),
			strings.Join(constraint.Columns, ", "),
			references,
		})
	}

	return "Constraint Changes:\n" + formatter.Render()
}

// formatTableConstraintChanges formats constraint changes within a specific table
func (sdp *SchemaDiffPresenter) formatTableConstraintChanges(tableDiff *schema.TableDiff) string {
	formatter := NewTableFormatter(sdp.colorSystem, sdp.theme)
	formatter.SetStyle(CompactTableStyle)
	formatter.SetHeaders([]string{"Change", "Constraint", "Type", "Columns", "References"})

	// Added constraints
	for _, constraint := range tableDiff.AddedConstraints {
		icon := sdp.getChangeIcon(ChangeAdded)
		references := ""
		if constraint.Type == schema.ConstraintTypeForeignKey {
			references = fmt.Sprintf("%s(%s)", constraint.ReferencedTable, strings.Join(constraint.ReferencedColumns, ", "))
		}
		formatter.AddRow([]string{
			icon + " ADD",
			"  " + constraint.Name, // Indent to show hierarchy
			string(constraint.Type),
			strings.Join(constraint.Columns, ", "),
			references,
		})
	}

	// Removed constraints
	for _, constraint := range tableDiff.RemovedConstraints {
		icon := sdp.getChangeIcon(ChangeRemoved)
		references := ""
		if constraint.Type == schema.ConstraintTypeForeignKey {
			references = fmt.Sprintf("%s(%s)", constraint.ReferencedTable, strings.Join(constraint.ReferencedColumns, ", "))
		}
		formatter.AddRow([]string{
			icon + " DROP",
			"  " + constraint.Name, // Indent to show hierarchy
			string(constraint.Type),
			strings.Join(constraint.Columns, ", "),
			references,
		})
	}

	return "  Constraint Changes:\n" + sdp.indentText(formatter.Render(), "  ")
}

// Helper methods

// getChangeIcon returns the appropriate icon for a change type
func (sdp *SchemaDiffPresenter) getChangeIcon(changeType ChangeType) string {
	switch changeType {
	case ChangeAdded:
		if sdp.colorSystem != nil && sdp.colorSystem.IsColorSupported() {
			return sdp.colorSystem.Colorize(sdp.iconSystem.RenderIcon("add"), sdp.theme.Success)
		}
		return sdp.iconSystem.RenderIcon("add")
	case ChangeRemoved:
		if sdp.colorSystem != nil && sdp.colorSystem.IsColorSupported() {
			return sdp.colorSystem.Colorize(sdp.iconSystem.RenderIcon("remove"), sdp.theme.Error)
		}
		return sdp.iconSystem.RenderIcon("remove")
	case ChangeModified:
		if sdp.colorSystem != nil && sdp.colorSystem.IsColorSupported() {
			return sdp.colorSystem.Colorize(sdp.iconSystem.RenderIcon("modify"), sdp.theme.Warning)
		}
		return sdp.iconSystem.RenderIcon("modify")
	default:
		return sdp.iconSystem.RenderIcon("info")
	}
}

// colorizeText applies color to text if color is supported
func (sdp *SchemaDiffPresenter) colorizeText(text string, color Color) string {
	if sdp.colorSystem != nil && sdp.colorSystem.IsColorSupported() {
		return sdp.colorSystem.Colorize(text, color)
	}
	return text
}

// formatNullable formats nullable boolean as YES/NO
func (sdp *SchemaDiffPresenter) formatNullable(nullable bool) string {
	if nullable {
		return "YES"
	}
	return "NO"
}

// formatBoolean formats boolean as YES/NO
func (sdp *SchemaDiffPresenter) formatBoolean(value bool) string {
	if value {
		return "YES"
	}
	return "NO"
}

// indentText indents all lines of text with the given prefix
func (sdp *SchemaDiffPresenter) indentText(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// formatTableNames formats a list of table names
func (sdp *SchemaDiffPresenter) formatTableNames(tables []*schema.Table) string {
	names := make([]string, len(tables))
	for i, table := range tables {
		names[i] = table.Name
	}
	return strings.Join(names, ", ")
}

// formatModifiedTableNames formats a list of modified table names
func (sdp *SchemaDiffPresenter) formatModifiedTableNames(tableDiffs []*schema.TableDiff) string {
	names := make([]string, len(tableDiffs))
	for i, tableDiff := range tableDiffs {
		names[i] = tableDiff.TableName
	}
	return strings.Join(names, ", ")
}

// formatIndexNames formats a list of index names
func (sdp *SchemaDiffPresenter) formatIndexNames(indexes []*schema.Index) string {
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = fmt.Sprintf("%s.%s", index.TableName, index.Name)
	}
	return strings.Join(names, ", ")
}

// formatConstraintNames formats a list of constraint names
func (sdp *SchemaDiffPresenter) formatConstraintNames(constraints []*schema.Constraint) string {
	names := make([]string, len(constraints))
	for i, constraint := range constraints {
		names[i] = fmt.Sprintf("%s.%s", constraint.TableName, constraint.Name)
	}
	return strings.Join(names, ", ")
}

// Check methods for determining if changes exist

// hasTableChanges checks if there are any table-level changes
func (sdp *SchemaDiffPresenter) hasTableChanges(diff *schema.SchemaDiff) bool {
	return len(diff.AddedTables) > 0 || len(diff.RemovedTables) > 0 || len(diff.ModifiedTables) > 0
}

// hasIndexChanges checks if there are any index changes
func (sdp *SchemaDiffPresenter) hasIndexChanges(diff *schema.SchemaDiff) bool {
	return len(diff.AddedIndexes) > 0 || len(diff.RemovedIndexes) > 0
}

// hasConstraintChanges checks if there are any constraint changes
func (sdp *SchemaDiffPresenter) hasConstraintChanges(diff *schema.SchemaDiff) bool {
	return len(diff.AddedConstraints) > 0 || len(diff.RemovedConstraints) > 0
}

// hasColumnChanges checks if there are any column changes in a table diff
func (sdp *SchemaDiffPresenter) hasColumnChanges(tableDiff *schema.TableDiff) bool {
	return len(tableDiff.AddedColumns) > 0 || len(tableDiff.RemovedColumns) > 0 || len(tableDiff.ModifiedColumns) > 0
}
