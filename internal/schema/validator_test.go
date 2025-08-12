package schema

import (
	"strings"
	"testing"
)

func TestSchemaValidator_ValidateChanges(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name             string
		diff             *SchemaDiff
		expectedValid    bool
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name:             "empty diff",
			diff:             &SchemaDiff{},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "drop table warning",
			diff: &SchemaDiff{
				RemovedTables: []*Table{NewTable("users")},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "drop column warning",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						RemovedColumns: []*Column{
							NewColumn("old_field", "VARCHAR(255)", true),
						},
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "data type change warning",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						ModifiedColumns: []*ColumnDiff{
							{
								ColumnName: "age",
								OldColumn:  NewColumn("age", "INT", false),
								NewColumn:  NewColumn("age", "TINYINT", false),
							},
						},
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "nullability change warning",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						ModifiedColumns: []*ColumnDiff{
							{
								ColumnName: "email",
								OldColumn:  NewColumn("email", "VARCHAR(255)", true),
								NewColumn:  NewColumn("email", "VARCHAR(255)", false),
							},
						},
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "add NOT NULL column without default warning",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						AddedColumns: []*Column{
							NewColumn("required_field", "VARCHAR(255)", false),
						},
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateChanges(tt.diff)

			if result.IsValid != tt.expectedValid {
				t.Errorf("ValidateChanges() IsValid = %v, expected %v", result.IsValid, tt.expectedValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("ValidateChanges() Errors count = %d, expected %d", len(result.Errors), tt.expectedErrors)
			}

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("ValidateChanges() Warnings count = %d, expected %d", len(result.Warnings), tt.expectedWarnings)
			}
		})
	}
}
func TestSchemaValidator_ValidateDependencies(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name           string
		diff           *SchemaDiff
		expectedValid  bool
		expectedErrors int
	}{
		{
			name: "drop table with foreign key dependency error",
			diff: &SchemaDiff{
				RemovedTables: []*Table{NewTable("users")},
				AddedConstraints: []*Constraint{
					NewForeignKeyConstraint("fk_user", "orders", []string{"user_id"}, "users", []string{"id"}),
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "drop column with constraint dependency error",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						RemovedColumns: []*Column{
							NewColumn("email", "VARCHAR(255)", true),
						},
					},
				},
				AddedConstraints: []*Constraint{
					NewConstraint("uk_email", "users", ConstraintTypeUnique, []string{"email"}),
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "drop column with index dependency error",
			diff: &SchemaDiff{
				ModifiedTables: []*TableDiff{
					{
						TableName: "users",
						RemovedColumns: []*Column{
							NewColumn("name", "VARCHAR(255)", true),
						},
					},
				},
				AddedIndexes: []*Index{
					NewIndex("idx_name", "users", []string{"name"}),
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "no dependency conflicts",
			diff: &SchemaDiff{
				AddedTables: []*Table{NewTable("new_table")},
				AddedConstraints: []*Constraint{
					NewConstraint("uk_test", "new_table", ConstraintTypeUnique, []string{"test_col"}),
				},
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateChanges(tt.diff)

			if result.IsValid != tt.expectedValid {
				t.Errorf("ValidateChanges() IsValid = %v, expected %v", result.IsValid, tt.expectedValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("ValidateChanges() Errors count = %d, expected %d", len(result.Errors), tt.expectedErrors)
			}
		})
	}
}

func TestSchemaValidator_IsDataLossyTypeChange(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name     string
		oldType  string
		newType  string
		expected bool
	}{
		{
			name:     "same type",
			oldType:  "VARCHAR(255)",
			newType:  "VARCHAR(255)",
			expected: false,
		},
		{
			name:     "varchar to text (compatible)",
			oldType:  "VARCHAR(255)",
			newType:  "TEXT",
			expected: false,
		},
		{
			name:     "text to varchar (potentially lossy)",
			oldType:  "TEXT",
			newType:  "VARCHAR(255)",
			expected: true,
		},
		{
			name:     "int to bigint (safe)",
			oldType:  "INT",
			newType:  "BIGINT",
			expected: false, // Safe conversion
		},
		{
			name:     "bigint to int (lossy)",
			oldType:  "BIGINT",
			newType:  "INT",
			expected: true, // Potentially lossy conversion
		},
		{
			name:     "datetime to date (lossy)",
			oldType:  "DATETIME",
			newType:  "DATE",
			expected: true, // Potentially lossy conversion
		},
		{
			name:     "completely different types",
			oldType:  "INT",
			newType:  "VARCHAR(255)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isDataLossyTypeChange(tt.oldType, tt.newType)
			if result != tt.expected {
				t.Errorf("isDataLossyTypeChange(%s, %s) = %v, expected %v", tt.oldType, tt.newType, result, tt.expected)
			}
		})
	}
}
func TestSchemaValidator_IsSizeReduction(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name     string
		oldType  string
		newType  string
		expected bool
	}{
		{
			name:     "varchar size reduction",
			oldType:  "VARCHAR(255)",
			newType:  "VARCHAR(100)",
			expected: true,
		},
		{
			name:     "varchar size increase",
			oldType:  "VARCHAR(100)",
			newType:  "VARCHAR(255)",
			expected: false,
		},
		{
			name:     "same size",
			oldType:  "VARCHAR(255)",
			newType:  "VARCHAR(255)",
			expected: false,
		},
		{
			name:     "no size specified",
			oldType:  "TEXT",
			newType:  "LONGTEXT",
			expected: false,
		},
		{
			name:     "decimal precision reduction",
			oldType:  "DECIMAL(10,2)",
			newType:  "DECIMAL(8,2)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isSizeReduction(tt.oldType, tt.newType)
			if result != tt.expected {
				t.Errorf("isSizeReduction(%s, %s) = %v, expected %v", tt.oldType, tt.newType, result, tt.expected)
			}
		})
	}
}

func TestSchemaValidator_FormatValidationResult(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name      string
		result    *ValidationResult
		useColors bool
		contains  []string
	}{
		{
			name: "no issues",
			result: &ValidationResult{
				IsValid:  true,
				Warnings: []Warning{},
				Errors:   []ValidationError{},
			},
			useColors: false,
			contains:  []string{"✓", "No validation issues found"},
		},
		{
			name: "with errors",
			result: &ValidationResult{
				IsValid: false,
				Errors: []ValidationError{
					{
						Type:      ErrorTypeDependency,
						Message:   "Cannot drop table due to dependency",
						TableName: "users",
						Details:   "Foreign key constraint exists",
					},
				},
				Warnings: []Warning{},
			},
			useColors: false,
			contains:  []string{"❌", "Validation Errors", "Cannot drop table due to dependency", "Foreign key constraint exists"},
		},
		{
			name: "with warnings",
			result: &ValidationResult{
				IsValid: true,
				Errors:  []ValidationError{},
				Warnings: []Warning{
					{
						Type:       WarningTypeDataLoss,
						Severity:   SeverityHigh,
						Message:    "Dropping column will cause data loss",
						TableName:  "users",
						ColumnName: "old_field",
						Suggestion: "Back up data first",
					},
				},
			},
			useColors: false,
			contains:  []string{"⚠", "Validation Warnings", "🟠 High", "Dropping column will cause data loss", "💡", "Back up data first"},
		},
		{
			name: "with colors",
			result: &ValidationResult{
				IsValid:  true,
				Warnings: []Warning{},
				Errors:   []ValidationError{},
			},
			useColors: true,
			contains:  []string{"✓", "No validation issues found", "\033[32m", "\033[0m"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.FormatValidationResult(tt.result, tt.useColors)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatValidationResult() result does not contain %q\nResult:\n%s", expected, result)
				}
			}
		})
	}
}

func TestSchemaValidator_GetValidationSummary(t *testing.T) {
	validator := NewSchemaValidator(false)

	tests := []struct {
		name     string
		result   *ValidationResult
		expected string
	}{
		{
			name: "no issues",
			result: &ValidationResult{
				IsValid:  true,
				Warnings: []Warning{},
				Errors:   []ValidationError{},
			},
			expected: "No issues",
		},
		{
			name: "errors only",
			result: &ValidationResult{
				IsValid: false,
				Errors: []ValidationError{
					{Message: "Error 1"},
					{Message: "Error 2"},
				},
				Warnings: []Warning{},
			},
			expected: "2 errors",
		},
		{
			name: "warnings only",
			result: &ValidationResult{
				IsValid: true,
				Errors:  []ValidationError{},
				Warnings: []Warning{
					{Message: "Warning 1"},
				},
			},
			expected: "1 warnings",
		},
		{
			name: "errors and warnings",
			result: &ValidationResult{
				IsValid: false,
				Errors: []ValidationError{
					{Message: "Error 1"},
				},
				Warnings: []Warning{
					{Message: "Warning 1"},
					{Message: "Warning 2"},
				},
			},
			expected: "1 errors, 2 warnings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.GetValidationSummary(tt.result)
			if result != tt.expected {
				t.Errorf("GetValidationSummary() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
