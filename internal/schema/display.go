package schema

import (
	"fmt"
	"sort"
	"strings"
)

// DisplayFormatter handles formatting schema differences for user-friendly display
type DisplayFormatter struct {
	ShowDetails bool
	UseColors   bool
}

// NewDisplayFormatter creates a new DisplayFormatter instance
func NewDisplayFormatter(showDetails, useColors bool) *DisplayFormatter {
	return &DisplayFormatter{
		ShowDetails: showDetails,
		UseColors:   useColors,
	}
}

// FormatSchemaDiff formats a SchemaDiff for display
func (df *DisplayFormatter) FormatSchemaDiff(diff *SchemaDiff) string {
	if df.IsEmpty(diff) {
		return df.colorize("✓ No schema differences found - databases are synchronized", "green")
	}

	var output strings.Builder
	output.WriteString(df.colorize("Schema Differences Summary", "bold"))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("=", 50))
	output.WriteString("\n\n")

	// Format table changes
	if len(diff.AddedTables) > 0 || len(diff.RemovedTables) > 0 || len(diff.ModifiedTables) > 0 {
		output.WriteString(df.formatTableChanges(diff))
		output.WriteString("\n")
	}

	// Format index changes
	if len(diff.AddedIndexes) > 0 || len(diff.RemovedIndexes) > 0 {
		output.WriteString(df.formatIndexChanges(diff))
		output.WriteString("\n")
	}

	// Format constraint changes
	if len(diff.AddedConstraints) > 0 || len(diff.RemovedConstraints) > 0 {
		output.WriteString(df.formatConstraintChanges(diff))
		output.WriteString("\n")
	}

	return output.String()
}

// IsEmpty checks if the schema diff has any changes
func (df *DisplayFormatter) IsEmpty(diff *SchemaDiff) bool {
	return len(diff.AddedTables) == 0 &&
		len(diff.RemovedTables) == 0 &&
		len(diff.ModifiedTables) == 0 &&
		len(diff.AddedIndexes) == 0 &&
		len(diff.RemovedIndexes) == 0 &&
		len(diff.AddedConstraints) == 0 &&
		len(diff.RemovedConstraints) == 0
}

// formatTableChanges formats table-level changes
func (df *DisplayFormatter) formatTableChanges(diff *SchemaDiff) string {
	var output strings.Builder
	output.WriteString(df.colorize("Tables", "bold"))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("-", 20))
	output.WriteString("\n")

	// Added tables
	if len(diff.AddedTables) > 0 {
		output.WriteString(df.colorize("+ Added Tables:", "green"))
		output.WriteString("\n")
		for _, table := range diff.AddedTables {
			output.WriteString(fmt.Sprintf("  + %s", df.colorize(table.Name, "green")))
			if df.ShowDetails {
				output.WriteString(fmt.Sprintf(" (%d columns)", len(table.Columns)))
			}
			output.WriteString("\n")
			if df.ShowDetails {
				output.WriteString(df.formatTableDetails(table, "    "))
			}
		}
		output.WriteString("\n")
	}

	// Removed tables
	if len(diff.RemovedTables) > 0 {
		output.WriteString(df.colorize("- Removed Tables:", "red"))
		output.WriteString("\n")
		for _, table := range diff.RemovedTables {
			output.WriteString(fmt.Sprintf("  - %s", df.colorize(table.Name, "red")))
			if df.ShowDetails {
				output.WriteString(fmt.Sprintf(" (%d columns)", len(table.Columns)))
			}
			output.WriteString("\n")
			if df.ShowDetails {
				output.WriteString(df.formatTableDetails(table, "    "))
			}
		}
		output.WriteString("\n")
	}

	// Modified tables
	if len(diff.ModifiedTables) > 0 {
		output.WriteString(df.colorize("~ Modified Tables:", "yellow"))
		output.WriteString("\n")
		for _, tableDiff := range diff.ModifiedTables {
			output.WriteString(fmt.Sprintf("  ~ %s", df.colorize(tableDiff.TableName, "yellow")))
			output.WriteString("\n")
			output.WriteString(df.formatTableDiff(tableDiff, "    "))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// formatTableDetails formats detailed table information
func (df *DisplayFormatter) formatTableDetails(table *Table, indent string) string {
	var output strings.Builder

	// Sort columns by position for consistent display
	columns := make([]*Column, 0, len(table.Columns))
	for _, col := range table.Columns {
		columns = append(columns, col)
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position < columns[j].Position
	})

	for _, col := range columns {
		output.WriteString(fmt.Sprintf("%s%s %s", indent, col.Name, col.DataType))
		if !col.IsNullable {
			output.WriteString(" NOT NULL")
		}
		if col.DefaultValue != nil {
			output.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
		}
		if col.Extra != "" {
			output.WriteString(fmt.Sprintf(" %s", col.Extra))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// formatTableDiff formats table-level differences
func (df *DisplayFormatter) formatTableDiff(tableDiff *TableDiff, indent string) string {
	var output strings.Builder

	// Added columns
	if len(tableDiff.AddedColumns) > 0 {
		output.WriteString(fmt.Sprintf("%s%s\n", indent, df.colorize("+ Added Columns:", "green")))
		for _, col := range tableDiff.AddedColumns {
			output.WriteString(fmt.Sprintf("%s  + %s %s", indent, df.colorize(col.Name, "green"), col.DataType))
			if !col.IsNullable {
				output.WriteString(" NOT NULL")
			}
			if col.DefaultValue != nil {
				output.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
			}
			if col.Extra != "" {
				output.WriteString(fmt.Sprintf(" %s", col.Extra))
			}
			output.WriteString("\n")
		}
	}

	// Removed columns
	if len(tableDiff.RemovedColumns) > 0 {
		output.WriteString(fmt.Sprintf("%s%s\n", indent, df.colorize("- Removed Columns:", "red")))
		for _, col := range tableDiff.RemovedColumns {
			output.WriteString(fmt.Sprintf("%s  - %s %s", indent, df.colorize(col.Name, "red"), col.DataType))
			if !col.IsNullable {
				output.WriteString(" NOT NULL")
			}
			if col.DefaultValue != nil {
				output.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
			}
			if col.Extra != "" {
				output.WriteString(fmt.Sprintf(" %s", col.Extra))
			}
			output.WriteString("\n")
		}
	}

	// Modified columns
	if len(tableDiff.ModifiedColumns) > 0 {
		output.WriteString(fmt.Sprintf("%s%s\n", indent, df.colorize("~ Modified Columns:", "yellow")))
		for _, colDiff := range tableDiff.ModifiedColumns {
			output.WriteString(fmt.Sprintf("%s  ~ %s\n", indent, df.colorize(colDiff.ColumnName, "yellow")))
			output.WriteString(df.formatColumnDiff(colDiff, indent+"    "))
		}
	}

	// Added constraints
	if len(tableDiff.AddedConstraints) > 0 {
		output.WriteString(fmt.Sprintf("%s%s\n", indent, df.colorize("+ Added Constraints:", "green")))
		for _, constraint := range tableDiff.AddedConstraints {
			output.WriteString(fmt.Sprintf("%s  + %s\n", indent, df.formatConstraint(constraint, "green")))
		}
	}

	// Removed constraints
	if len(tableDiff.RemovedConstraints) > 0 {
		output.WriteString(fmt.Sprintf("%s%s\n", indent, df.colorize("- Removed Constraints:", "red")))
		for _, constraint := range tableDiff.RemovedConstraints {
			output.WriteString(fmt.Sprintf("%s  - %s\n", indent, df.formatConstraint(constraint, "red")))
		}
	}

	return output.String()
}

// formatColumnDiff formats column-level differences
func (df *DisplayFormatter) formatColumnDiff(colDiff *ColumnDiff, indent string) string {
	var output strings.Builder
	old := colDiff.OldColumn
	new := colDiff.NewColumn

	// Data type change
	if old.DataType != new.DataType {
		output.WriteString(fmt.Sprintf("%sData Type: %s → %s\n",
			indent,
			df.colorize(old.DataType, "red"),
			df.colorize(new.DataType, "green")))
	}

	// Nullability change
	if old.IsNullable != new.IsNullable {
		oldNull := "NULL"
		newNull := "NULL"
		if !old.IsNullable {
			oldNull = "NOT NULL"
		}
		if !new.IsNullable {
			newNull = "NOT NULL"
		}
		output.WriteString(fmt.Sprintf("%sNullability: %s → %s\n",
			indent,
			df.colorize(oldNull, "red"),
			df.colorize(newNull, "green")))
	}

	// Default value change
	oldDefault := "NULL"
	newDefault := "NULL"
	if old.DefaultValue != nil {
		oldDefault = *old.DefaultValue
	}
	if new.DefaultValue != nil {
		newDefault = *new.DefaultValue
	}
	if oldDefault != newDefault {
		output.WriteString(fmt.Sprintf("%sDefault: %s → %s\n",
			indent,
			df.colorize(oldDefault, "red"),
			df.colorize(newDefault, "green")))
	}

	// Extra attributes change
	if old.Extra != new.Extra {
		output.WriteString(fmt.Sprintf("%sExtra: %s → %s\n",
			indent,
			df.colorize(old.Extra, "red"),
			df.colorize(new.Extra, "green")))
	}

	return output.String()
}

// formatIndexChanges formats index-level changes
func (df *DisplayFormatter) formatIndexChanges(diff *SchemaDiff) string {
	var output strings.Builder
	output.WriteString(df.colorize("Indexes", "bold"))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("-", 20))
	output.WriteString("\n")

	// Added indexes
	if len(diff.AddedIndexes) > 0 {
		output.WriteString(df.colorize("+ Added Indexes:", "green"))
		output.WriteString("\n")
		for _, index := range diff.AddedIndexes {
			output.WriteString(fmt.Sprintf("  + %s\n", df.formatIndex(index, "green")))
		}
		output.WriteString("\n")
	}

	// Removed indexes
	if len(diff.RemovedIndexes) > 0 {
		output.WriteString(df.colorize("- Removed Indexes:", "red"))
		output.WriteString("\n")
		for _, index := range diff.RemovedIndexes {
			output.WriteString(fmt.Sprintf("  - %s\n", df.formatIndex(index, "red")))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// formatConstraintChanges formats constraint-level changes
func (df *DisplayFormatter) formatConstraintChanges(diff *SchemaDiff) string {
	var output strings.Builder
	output.WriteString(df.colorize("Constraints", "bold"))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("-", 20))
	output.WriteString("\n")

	// Added constraints
	if len(diff.AddedConstraints) > 0 {
		output.WriteString(df.colorize("+ Added Constraints:", "green"))
		output.WriteString("\n")
		for _, constraint := range diff.AddedConstraints {
			output.WriteString(fmt.Sprintf("  + %s\n", df.formatConstraint(constraint, "green")))
		}
		output.WriteString("\n")
	}

	// Removed constraints
	if len(diff.RemovedConstraints) > 0 {
		output.WriteString(df.colorize("- Removed Constraints:", "red"))
		output.WriteString("\n")
		for _, constraint := range diff.RemovedConstraints {
			output.WriteString(fmt.Sprintf("  - %s\n", df.formatConstraint(constraint, "red")))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// formatIndex formats an index for display
func (df *DisplayFormatter) formatIndex(index *Index, color string) string {
	var parts []string

	// Index name and table
	parts = append(parts, df.colorize(index.Name, color))
	parts = append(parts, fmt.Sprintf("on %s", index.TableName))

	// Index type and properties
	var properties []string
	if index.IsPrimary {
		properties = append(properties, "PRIMARY")
	} else if index.IsUnique {
		properties = append(properties, "UNIQUE")
	}
	if index.IndexType != "" && index.IndexType != "BTREE" {
		properties = append(properties, index.IndexType)
	}

	if len(properties) > 0 {
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(properties, ", ")))
	}

	// Columns
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(index.Columns, ", ")))

	return strings.Join(parts, " ")
}

// formatConstraint formats a constraint for display
func (df *DisplayFormatter) formatConstraint(constraint *Constraint, color string) string {
	var parts []string

	// Constraint name and table
	parts = append(parts, df.colorize(constraint.Name, color))
	parts = append(parts, fmt.Sprintf("on %s", constraint.TableName))

	// Constraint type and details
	switch constraint.Type {
	case ConstraintTypeForeignKey:
		parts = append(parts, "FOREIGN KEY")
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(constraint.Columns, ", ")))
		parts = append(parts, fmt.Sprintf("REFERENCES %s(%s)",
			constraint.ReferencedTable,
			strings.Join(constraint.ReferencedColumns, ", ")))
		if constraint.OnUpdate != "" && constraint.OnUpdate != "RESTRICT" {
			parts = append(parts, fmt.Sprintf("ON UPDATE %s", constraint.OnUpdate))
		}
		if constraint.OnDelete != "" && constraint.OnDelete != "RESTRICT" {
			parts = append(parts, fmt.Sprintf("ON DELETE %s", constraint.OnDelete))
		}
	case ConstraintTypeUnique:
		parts = append(parts, "UNIQUE")
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(constraint.Columns, ", ")))
	case ConstraintTypeCheck:
		parts = append(parts, "CHECK")
		parts = append(parts, fmt.Sprintf("(%s)", constraint.CheckExpression))
	}

	return strings.Join(parts, " ")
}

// Colorize applies color formatting to text if colors are enabled (public method)
func (df *DisplayFormatter) Colorize(text, color string) string {
	return df.colorize(text, color)
}

// colorize applies color formatting to text if colors are enabled
func (df *DisplayFormatter) colorize(text, color string) string {
	if !df.UseColors {
		return text
	}

	colorCodes := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"bold":   "\033[1m",
		"reset":  "\033[0m",
	}

	if code, exists := colorCodes[color]; exists {
		return fmt.Sprintf("%s%s%s", code, text, colorCodes["reset"])
	}

	return text
}

// GetChangeSummary returns a brief summary of changes
func (df *DisplayFormatter) GetChangeSummary(diff *SchemaDiff) string {
	if df.IsEmpty(diff) {
		return "No changes detected"
	}

	var parts []string

	// Count table changes (only added/removed tables, not modified)
	tableChanges := len(diff.AddedTables) + len(diff.RemovedTables)
	if tableChanges > 0 {
		parts = append(parts, fmt.Sprintf("%d table changes", tableChanges))
	}

	// Count column changes
	columnChanges := 0
	for _, tableDiff := range diff.ModifiedTables {
		columnChanges += len(tableDiff.AddedColumns) + len(tableDiff.RemovedColumns) + len(tableDiff.ModifiedColumns)
	}
	if columnChanges > 0 {
		parts = append(parts, fmt.Sprintf("%d column changes", columnChanges))
	}

	// Count index changes
	indexChanges := len(diff.AddedIndexes) + len(diff.RemovedIndexes)
	if indexChanges > 0 {
		parts = append(parts, fmt.Sprintf("%d index changes", indexChanges))
	}

	// Count constraint changes
	constraintChanges := len(diff.AddedConstraints) + len(diff.RemovedConstraints)
	for _, tableDiff := range diff.ModifiedTables {
		constraintChanges += len(tableDiff.AddedConstraints) + len(tableDiff.RemovedConstraints)
	}
	if constraintChanges > 0 {
		parts = append(parts, fmt.Sprintf("%d constraint changes", constraintChanges))
	}

	if len(parts) == 0 {
		return "No changes detected"
	}

	return strings.Join(parts, ", ")
}

// FormatCompactSummary returns a compact one-line summary
func (df *DisplayFormatter) FormatCompactSummary(diff *SchemaDiff) string {
	summary := df.GetChangeSummary(diff)
	if summary == "No changes detected" {
		return df.colorize("✓ Schemas are synchronized", "green")
	}
	return df.colorize(fmt.Sprintf("⚠ Found: %s", summary), "yellow")
}
