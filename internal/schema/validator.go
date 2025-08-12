package schema

import (
	"fmt"
	"strings"
)

// ValidationResult represents the result of schema change validation
type ValidationResult struct {
	IsValid  bool
	Warnings []Warning
	Errors   []ValidationError
}

// Warning represents a warning about a potentially risky operation
type Warning struct {
	Type       WarningType
	Severity   WarningSeverity
	Message    string
	TableName  string
	ColumnName string
	Suggestion string
}

// ValidationError represents a validation error that prevents execution
type ValidationError struct {
	Type      ErrorType
	Message   string
	TableName string
	Details   string
}

// WarningType represents the type of warning
type WarningType string

const (
	WarningTypeDataLoss      WarningType = "DATA_LOSS"
	WarningTypePerformance   WarningType = "PERFORMANCE"
	WarningTypeCompatibility WarningType = "COMPATIBILITY"
	WarningTypeDestructive   WarningType = "DESTRUCTIVE"
	WarningTypeDependency    WarningType = "DEPENDENCY"
)

// WarningSeverity represents the severity level of a warning
type WarningSeverity string

const (
	SeverityLow      WarningSeverity = "LOW"
	SeverityMedium   WarningSeverity = "MEDIUM"
	SeverityHigh     WarningSeverity = "HIGH"
	SeverityCritical WarningSeverity = "CRITICAL"
)

// ErrorType represents the type of validation error
type ErrorType string

const (
	ErrorTypeDependency   ErrorType = "DEPENDENCY"
	ErrorTypeIncompatible ErrorType = "INCOMPATIBLE"
	ErrorTypeConstraint   ErrorType = "CONSTRAINT"
)

// SchemaValidator validates schema changes and generates warnings
type SchemaValidator struct {
	StrictMode bool
}

// NewSchemaValidator creates a new SchemaValidator instance
func NewSchemaValidator(strictMode bool) *SchemaValidator {
	return &SchemaValidator{
		StrictMode: strictMode,
	}
}

// ValidateChanges validates a schema diff and returns warnings and errors
func (sv *SchemaValidator) ValidateChanges(diff *SchemaDiff) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Warnings: make([]Warning, 0),
		Errors:   make([]ValidationError, 0),
	}

	// Validate table changes
	sv.validateTableChanges(diff, result)

	// Validate index changes
	sv.validateIndexChanges(diff, result)

	// Validate constraint changes
	sv.validateConstraintChanges(diff, result)

	// Validate dependencies
	sv.validateDependencies(diff, result)

	// Set overall validity
	result.IsValid = len(result.Errors) == 0

	return result
}

// validateTableChanges validates table-level changes
func (sv *SchemaValidator) validateTableChanges(diff *SchemaDiff, result *ValidationResult) {
	// Validate removed tables
	for _, table := range diff.RemovedTables {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeDestructive,
			Severity:   SeverityCritical,
			Message:    fmt.Sprintf("Dropping table '%s' will permanently delete all data", table.Name),
			TableName:  table.Name,
			Suggestion: "Consider backing up data before proceeding",
		})
	}

	// Validate modified tables
	for _, tableDiff := range diff.ModifiedTables {
		sv.validateColumnChanges(tableDiff, result)
	}
}

// validateColumnChanges validates column-level changes
func (sv *SchemaValidator) validateColumnChanges(tableDiff *TableDiff, result *ValidationResult) {
	// Validate removed columns
	for _, column := range tableDiff.RemovedColumns {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeDataLoss,
			Severity:   SeverityHigh,
			Message:    fmt.Sprintf("Dropping column '%s.%s' will permanently delete all data in this column", tableDiff.TableName, column.Name),
			TableName:  tableDiff.TableName,
			ColumnName: column.Name,
			Suggestion: "Consider backing up column data before proceeding",
		})
	}

	// Validate modified columns
	for _, colDiff := range tableDiff.ModifiedColumns {
		sv.validateColumnModification(tableDiff.TableName, colDiff, result)
	}

	// Validate added columns
	for _, column := range tableDiff.AddedColumns {
		sv.validateAddedColumn(tableDiff.TableName, column, result)
	}
}

// validateColumnModification validates modifications to existing columns
func (sv *SchemaValidator) validateColumnModification(tableName string, colDiff *ColumnDiff, result *ValidationResult) {
	old := colDiff.OldColumn
	new := colDiff.NewColumn

	// Check for data type changes that might cause data loss
	if sv.isDataLossyTypeChange(old.DataType, new.DataType) {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeDataLoss,
			Severity:   SeverityHigh,
			Message:    fmt.Sprintf("Changing column '%s.%s' from %s to %s may cause data loss", tableName, colDiff.ColumnName, old.DataType, new.DataType),
			TableName:  tableName,
			ColumnName: colDiff.ColumnName,
			Suggestion: "Verify that existing data is compatible with the new type",
		})
	}

	// Check for nullability changes
	if old.IsNullable && !new.IsNullable {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeDataLoss,
			Severity:   SeverityMedium,
			Message:    fmt.Sprintf("Making column '%s.%s' NOT NULL may fail if existing data contains NULL values", tableName, colDiff.ColumnName),
			TableName:  tableName,
			ColumnName: colDiff.ColumnName,
			Suggestion: "Ensure no NULL values exist in this column before applying changes",
		})
	}

	// Check for default value removal
	if old.DefaultValue != nil && new.DefaultValue == nil {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeCompatibility,
			Severity:   SeverityLow,
			Message:    fmt.Sprintf("Removing default value from column '%s.%s' may affect application behavior", tableName, colDiff.ColumnName),
			TableName:  tableName,
			ColumnName: colDiff.ColumnName,
			Suggestion: "Verify that applications handle missing default values correctly",
		})
	}

	// Check for size reduction
	if sv.isSizeReduction(old.DataType, new.DataType) {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeDataLoss,
			Severity:   SeverityMedium,
			Message:    fmt.Sprintf("Reducing size of column '%s.%s' from %s to %s may truncate existing data", tableName, colDiff.ColumnName, old.DataType, new.DataType),
			TableName:  tableName,
			ColumnName: colDiff.ColumnName,
			Suggestion: "Check that all existing data fits within the new size constraints",
		})
	}
}

// validateAddedColumn validates newly added columns
func (sv *SchemaValidator) validateAddedColumn(tableName string, column *Column, result *ValidationResult) {
	// Check for NOT NULL columns without default values
	if !column.IsNullable && column.DefaultValue == nil && column.Extra != "AUTO_INCREMENT" {
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypeCompatibility,
			Severity:   SeverityMedium,
			Message:    fmt.Sprintf("Adding NOT NULL column '%s.%s' without a default value may fail if table contains existing data", tableName, column.Name),
			TableName:  tableName,
			ColumnName: column.Name,
			Suggestion: "Consider adding a default value or making the column nullable initially",
		})
	}
}

// validateIndexChanges validates index-level changes
func (sv *SchemaValidator) validateIndexChanges(diff *SchemaDiff, result *ValidationResult) {
	// Validate removed indexes
	for _, index := range diff.RemovedIndexes {
		if index.IsPrimary {
			result.Errors = append(result.Errors, ValidationError{
				Type:      ErrorTypeConstraint,
				Message:   fmt.Sprintf("Cannot drop primary key index '%s' without adding a new one", index.Name),
				TableName: index.TableName,
				Details:   "Tables must have a primary key",
			})
		} else if index.IsUnique {
			result.Warnings = append(result.Warnings, Warning{
				Type:       WarningTypeCompatibility,
				Severity:   SeverityMedium,
				Message:    fmt.Sprintf("Dropping unique index '%s' will remove uniqueness constraint", index.Name),
				TableName:  index.TableName,
				Suggestion: "Ensure application logic handles potential duplicate values",
			})
		}

		// Performance warning for large tables
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypePerformance,
			Severity:   SeverityLow,
			Message:    fmt.Sprintf("Dropping index '%s' may impact query performance", index.Name),
			TableName:  index.TableName,
			Suggestion: "Monitor query performance after removing this index",
		})
	}

	// Validate added indexes
	for _, index := range diff.AddedIndexes {
		// Performance warning for large tables
		result.Warnings = append(result.Warnings, Warning{
			Type:       WarningTypePerformance,
			Severity:   SeverityLow,
			Message:    fmt.Sprintf("Creating index '%s' may take significant time on large tables", index.Name),
			TableName:  index.TableName,
			Suggestion: "Consider creating indexes during maintenance windows",
		})
	}
}

// validateConstraintChanges validates constraint-level changes
func (sv *SchemaValidator) validateConstraintChanges(diff *SchemaDiff, result *ValidationResult) {
	// Validate removed constraints
	for _, constraint := range diff.RemovedConstraints {
		switch constraint.Type {
		case ConstraintTypeForeignKey:
			result.Warnings = append(result.Warnings, Warning{
				Type:       WarningTypeCompatibility,
				Severity:   SeverityMedium,
				Message:    fmt.Sprintf("Dropping foreign key constraint '%s' removes referential integrity", constraint.Name),
				TableName:  constraint.TableName,
				Suggestion: "Ensure application logic maintains data integrity",
			})
		case ConstraintTypeUnique:
			result.Warnings = append(result.Warnings, Warning{
				Type:       WarningTypeCompatibility,
				Severity:   SeverityMedium,
				Message:    fmt.Sprintf("Dropping unique constraint '%s' allows duplicate values", constraint.Name),
				TableName:  constraint.TableName,
				Suggestion: "Ensure application logic handles potential duplicates",
			})
		}
	}

	// Validate added constraints
	for _, constraint := range diff.AddedConstraints {
		switch constraint.Type {
		case ConstraintTypeForeignKey:
			result.Warnings = append(result.Warnings, Warning{
				Type:       WarningTypeCompatibility,
				Severity:   SeverityMedium,
				Message:    fmt.Sprintf("Adding foreign key constraint '%s' may fail if existing data violates referential integrity", constraint.Name),
				TableName:  constraint.TableName,
				Suggestion: "Verify that all existing data satisfies the foreign key constraint",
			})
		case ConstraintTypeUnique:
			result.Warnings = append(result.Warnings, Warning{
				Type:       WarningTypeCompatibility,
				Severity:   SeverityMedium,
				Message:    fmt.Sprintf("Adding unique constraint '%s' may fail if existing data contains duplicates", constraint.Name),
				TableName:  constraint.TableName,
				Suggestion: "Ensure no duplicate values exist in the constrained columns",
			})
		}
	}
}

// validateDependencies validates dependencies between changes
func (sv *SchemaValidator) validateDependencies(diff *SchemaDiff, result *ValidationResult) {
	// Check for foreign key dependencies when dropping tables
	for _, removedTable := range diff.RemovedTables {
		// Check if any added or remaining constraints reference this table
		for _, constraint := range diff.AddedConstraints {
			if constraint.Type == ConstraintTypeForeignKey && constraint.ReferencedTable == removedTable.Name {
				result.Errors = append(result.Errors, ValidationError{
					Type:      ErrorTypeDependency,
					Message:   fmt.Sprintf("Cannot drop table '%s' because constraint '%s' references it", removedTable.Name, constraint.Name),
					TableName: removedTable.Name,
					Details:   fmt.Sprintf("Foreign key constraint '%s' on table '%s' references the table being dropped", constraint.Name, constraint.TableName),
				})
			}
		}
	}

	// Check for column dependencies when dropping columns
	for _, tableDiff := range diff.ModifiedTables {
		for _, removedColumn := range tableDiff.RemovedColumns {
			// Check if any constraints reference this column
			for _, constraint := range diff.AddedConstraints {
				if constraint.TableName == tableDiff.TableName {
					for _, col := range constraint.Columns {
						if col == removedColumn.Name {
							result.Errors = append(result.Errors, ValidationError{
								Type:      ErrorTypeDependency,
								Message:   fmt.Sprintf("Cannot drop column '%s.%s' because constraint '%s' references it", tableDiff.TableName, removedColumn.Name, constraint.Name),
								TableName: tableDiff.TableName,
								Details:   fmt.Sprintf("Constraint '%s' depends on column '%s'", constraint.Name, removedColumn.Name),
							})
						}
					}
				}
			}

			// Check if any indexes reference this column
			for _, index := range diff.AddedIndexes {
				if index.TableName == tableDiff.TableName {
					for _, col := range index.Columns {
						if col == removedColumn.Name {
							result.Errors = append(result.Errors, ValidationError{
								Type:      ErrorTypeDependency,
								Message:   fmt.Sprintf("Cannot drop column '%s.%s' because index '%s' references it", tableDiff.TableName, removedColumn.Name, index.Name),
								TableName: tableDiff.TableName,
								Details:   fmt.Sprintf("Index '%s' depends on column '%s'", index.Name, removedColumn.Name),
							})
						}
					}
				}
			}
		}
	}
}

// Helper functions for data type analysis

// isDataLossyTypeChange checks if changing from oldType to newType might cause data loss
func (sv *SchemaValidator) isDataLossyTypeChange(oldType, newType string) bool {
	oldBase := sv.extractBaseType(oldType)
	newBase := sv.extractBaseType(newType)

	// Same base type is not lossy (unless size reduction, handled separately)
	if oldBase == newBase {
		return false
	}

	// Define safe type conversions (from -> to)
	safeConversions := map[string][]string{
		"varchar":    {"text", "mediumtext", "longtext"},
		"char":       {"varchar", "text", "mediumtext", "longtext"},
		"tinytext":   {"text", "mediumtext", "longtext"},
		"text":       {"mediumtext", "longtext"},
		"mediumtext": {"longtext"},
		"tinyint":    {"smallint", "mediumint", "int", "bigint"},
		"smallint":   {"mediumint", "int", "bigint"},
		"mediumint":  {"int", "bigint"},
		"int":        {"bigint"},
		"float":      {"double"},
		"date":       {"datetime", "timestamp"},
		"time":       {"datetime", "timestamp"},
	}

	if safeTypes, exists := safeConversions[oldBase]; exists {
		for _, safeType := range safeTypes {
			if newBase == safeType {
				return false
			}
		}
	}

	// All other conversions are potentially lossy
	return true
}

// isSizeReduction checks if the new type has a smaller size than the old type
func (sv *SchemaValidator) isSizeReduction(oldType, newType string) bool {
	oldSize := sv.extractSize(oldType)
	newSize := sv.extractSize(newType)

	return oldSize > 0 && newSize > 0 && newSize < oldSize
}

// extractBaseType extracts the base type from a MySQL type definition
func (sv *SchemaValidator) extractBaseType(dataType string) string {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	if idx := strings.Index(dt, "("); idx != -1 {
		return dt[:idx]
	}
	return dt
}

// extractSize extracts the size from a MySQL type definition
func (sv *SchemaValidator) extractSize(dataType string) int {
	dt := strings.TrimSpace(dataType)
	start := strings.Index(dt, "(")
	end := strings.Index(dt, ")")

	if start == -1 || end == -1 || end <= start {
		return 0
	}

	sizeStr := dt[start+1 : end]
	// Handle precision types like DECIMAL(10,2)
	if commaIdx := strings.Index(sizeStr, ","); commaIdx != -1 {
		sizeStr = sizeStr[:commaIdx]
	}

	var size int
	fmt.Sscanf(sizeStr, "%d", &size)
	return size
}

// FormatValidationResult formats a validation result for display
func (sv *SchemaValidator) FormatValidationResult(result *ValidationResult, useColors bool) string {
	if len(result.Warnings) == 0 && len(result.Errors) == 0 {
		return sv.colorize("✓ No validation issues found", "green", useColors)
	}

	var output strings.Builder

	// Format errors
	if len(result.Errors) > 0 {
		output.WriteString(sv.colorize("❌ Validation Errors", "red", useColors))
		output.WriteString("\n")
		output.WriteString(strings.Repeat("-", 30))
		output.WriteString("\n")

		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("• %s\n", sv.colorize(err.Message, "red", useColors)))
			if err.Details != "" {
				output.WriteString(fmt.Sprintf("  Details: %s\n", err.Details))
			}
		}
		output.WriteString("\n")
	}

	// Format warnings by severity
	if len(result.Warnings) > 0 {
		output.WriteString(sv.colorize("⚠ Validation Warnings", "yellow", useColors))
		output.WriteString("\n")
		output.WriteString(strings.Repeat("-", 30))
		output.WriteString("\n")

		// Group warnings by severity
		warningsBySeverity := make(map[WarningSeverity][]Warning)
		for _, warning := range result.Warnings {
			warningsBySeverity[warning.Severity] = append(warningsBySeverity[warning.Severity], warning)
		}

		// Display in order of severity
		severities := []WarningSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severities {
			if warnings, exists := warningsBySeverity[severity]; exists {
				output.WriteString(fmt.Sprintf("\n%s (%s):\n",
					sv.getSeverityLabel(severity),
					sv.colorize(string(severity), sv.getSeverityColor(severity), useColors)))

				for _, warning := range warnings {
					output.WriteString(fmt.Sprintf("• %s\n", warning.Message))
					if warning.Suggestion != "" {
						output.WriteString(fmt.Sprintf("  💡 %s\n", sv.colorize(warning.Suggestion, "blue", useColors)))
					}
				}
			}
		}
	}

	return output.String()
}

// GetValidationSummary returns a brief summary of validation results
func (sv *SchemaValidator) GetValidationSummary(result *ValidationResult) string {
	if len(result.Warnings) == 0 && len(result.Errors) == 0 {
		return "No issues"
	}

	var parts []string
	if len(result.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", len(result.Errors)))
	}
	if len(result.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", len(result.Warnings)))
	}

	return strings.Join(parts, ", ")
}

// Helper methods for formatting

func (sv *SchemaValidator) getSeverityLabel(severity WarningSeverity) string {
	switch severity {
	case SeverityCritical:
		return "🔴 Critical"
	case SeverityHigh:
		return "🟠 High"
	case SeverityMedium:
		return "🟡 Medium"
	case SeverityLow:
		return "🟢 Low"
	default:
		return "❓ Unknown"
	}
}

func (sv *SchemaValidator) getSeverityColor(severity WarningSeverity) string {
	switch severity {
	case SeverityCritical:
		return "red"
	case SeverityHigh:
		return "red"
	case SeverityMedium:
		return "yellow"
	case SeverityLow:
		return "green"
	default:
		return "white"
	}
}

func (sv *SchemaValidator) colorize(text, color string, useColors bool) string {
	if !useColors {
		return text
	}

	colorCodes := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"white":  "\033[37m",
		"bold":   "\033[1m",
		"reset":  "\033[0m",
	}

	if code, exists := colorCodes[color]; exists {
		return fmt.Sprintf("%s%s%s", code, text, colorCodes["reset"])
	}

	return text
}
