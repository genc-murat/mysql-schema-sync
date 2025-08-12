package confirmation

import (
	"testing"

	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

func TestNewConfirmationService(t *testing.T) {
	service := NewConfirmationService(true)
	if service == nil {
		t.Fatal("NewConfirmationService returned nil")
	}

	// Test with colors disabled
	serviceNoColors := NewConfirmationService(false)
	if serviceNoColors == nil {
		t.Fatal("NewConfirmationService with colors disabled returned nil")
	}
}

func TestDisplayChangeSummary(t *testing.T) {
	service := NewConfirmationService(false) // Disable colors for testing
	cs := service.(*confirmationService)

	// Create test data
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "test_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "int",
						IsNullable: false,
						Position:   1,
					},
				},
			},
		},
	}

	plan := &migration.MigrationPlan{
		Statements: []migration.MigrationStatement{
			{
				SQL:           "CREATE TABLE test_table (id int NOT NULL)",
				Type:          migration.StatementTypeCreateTable,
				Description:   "Create table test_table",
				IsDestructive: false,
			},
		},
		Warnings: []string{"This is a test warning"},
	}

	// Test displaying change summary
	err := cs.DisplayChangeSummary(diff, plan)
	if err != nil {
		t.Errorf("DisplayChangeSummary failed: %v", err)
	}
}

func TestCountDestructiveOperations(t *testing.T) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)

	tests := []struct {
		name     string
		plan     *migration.MigrationPlan
		expected int
	}{
		{
			name: "no destructive operations",
			plan: &migration.MigrationPlan{
				Statements: []migration.MigrationStatement{
					{IsDestructive: false},
					{IsDestructive: false},
				},
			},
			expected: 0,
		},
		{
			name: "some destructive operations",
			plan: &migration.MigrationPlan{
				Statements: []migration.MigrationStatement{
					{IsDestructive: false},
					{IsDestructive: true},
					{IsDestructive: true},
				},
			},
			expected: 2,
		},
		{
			name: "all destructive operations",
			plan: &migration.MigrationPlan{
				Statements: []migration.MigrationStatement{
					{IsDestructive: true},
					{IsDestructive: true},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.countDestructiveOperations(tt.plan)
			if result != tt.expected {
				t.Errorf("countDestructiveOperations() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestEstimateExecutionTime(t *testing.T) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)

	tests := []struct {
		name           string
		statementCount int
		expectedPrefix string
	}{
		{
			name:           "no statements",
			statementCount: 0,
			expectedPrefix: "< 1 second",
		},
		{
			name:           "few statements",
			statementCount: 5,
			expectedPrefix: "< 5 seconds",
		},
		{
			name:           "moderate statements",
			statementCount: 25,
			expectedPrefix: "< 30 seconds",
		},
		{
			name:           "many statements",
			statementCount: 100,
			expectedPrefix: "> 1 minute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &migration.MigrationPlan{
				Statements: make([]migration.MigrationStatement, tt.statementCount),
			}
			result := cs.estimateExecutionTime(plan)
			if result != tt.expectedPrefix {
				t.Errorf("estimateExecutionTime() = %s, expected %s", result, tt.expectedPrefix)
			}
		})
	}
}

func TestParseConfirmationInput(t *testing.T) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "yes lowercase",
			input:    "y",
			expected: true,
		},
		{
			name:     "yes full",
			input:    "yes",
			expected: true,
		},
		{
			name:     "yes uppercase",
			input:    "Y",
			expected: true,
		},
		{
			name:     "no lowercase",
			input:    "n",
			expected: false,
		},
		{
			name:     "no full",
			input:    "no",
			expected: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace",
			input:    "  ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.parseConfirmationInput(tt.input)
			if result != tt.expected {
				t.Errorf("parseConfirmationInput(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfirmChangesWithEmptyDiff(t *testing.T) {
	service := NewConfirmationService(false)

	// Create empty diff
	diff := &schema.SchemaDiff{}
	plan := &migration.MigrationPlan{}

	// Test with empty diff
	approved, err := service.ConfirmChanges(diff, plan, false)
	if err != nil {
		t.Errorf("ConfirmChanges with empty diff failed: %v", err)
	}
	if approved {
		t.Error("ConfirmChanges with empty diff should return false")
	}
}

func TestConfirmChangesWithAutoApprove(t *testing.T) {
	service := NewConfirmationService(false)

	// Create test diff with changes
	diff := &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "test_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "int",
						IsNullable: false,
						Position:   1,
					},
				},
			},
		},
	}

	plan := &migration.MigrationPlan{
		Statements: []migration.MigrationStatement{
			{
				SQL:           "CREATE TABLE test_table (id int NOT NULL)",
				Type:          migration.StatementTypeCreateTable,
				Description:   "Create table test_table",
				IsDestructive: false,
			},
		},
	}

	// Test with auto-approve enabled
	approved, err := service.ConfirmChanges(diff, plan, true)
	if err != nil {
		t.Errorf("ConfirmChanges with auto-approve failed: %v", err)
	}
	if !approved {
		t.Error("ConfirmChanges with auto-approve should return true")
	}
}

func TestDisplaySQLDetails(t *testing.T) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)

	plan := &migration.MigrationPlan{
		Statements: []migration.MigrationStatement{
			{
				SQL:           "CREATE TABLE test_table (id int NOT NULL)",
				Type:          migration.StatementTypeCreateTable,
				Description:   "Create table test_table",
				IsDestructive: false,
			},
			{
				SQL:           "DROP TABLE old_table",
				Type:          migration.StatementTypeDropTable,
				Description:   "Drop table old_table",
				IsDestructive: true,
			},
		},
	}

	// This test mainly ensures the method doesn't panic
	// In a real test environment, we might capture stdout to verify output
	cs.displaySQLDetails(plan)
}

// Helper function to create a test schema diff
func createTestSchemaDiff() *schema.SchemaDiff {
	return &schema.SchemaDiff{
		AddedTables: []*schema.Table{
			{
				Name: "new_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "int",
						IsNullable: false,
						Position:   1,
					},
					"name": {
						Name:       "name",
						DataType:   "varchar(255)",
						IsNullable: true,
						Position:   2,
					},
				},
			},
		},
		RemovedTables: []*schema.Table{
			{
				Name: "old_table",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "int",
						IsNullable: false,
						Position:   1,
					},
				},
			},
		},
		ModifiedTables: []*schema.TableDiff{
			{
				TableName: "modified_table",
				AddedColumns: []*schema.Column{
					{
						Name:       "new_column",
						DataType:   "varchar(100)",
						IsNullable: true,
						Position:   3,
					},
				},
				RemovedColumns: []*schema.Column{
					{
						Name:       "old_column",
						DataType:   "text",
						IsNullable: true,
						Position:   2,
					},
				},
			},
		},
	}
}

// Helper function to create a test migration plan
func createTestMigrationPlan() *migration.MigrationPlan {
	return &migration.MigrationPlan{
		Statements: []migration.MigrationStatement{
			{
				SQL:           "CREATE TABLE new_table (id int NOT NULL, name varchar(255))",
				Type:          migration.StatementTypeCreateTable,
				Description:   "Create table new_table",
				IsDestructive: false,
			},
			{
				SQL:           "ALTER TABLE modified_table ADD COLUMN new_column varchar(100)",
				Type:          migration.StatementTypeAddColumn,
				Description:   "Add column new_column to modified_table",
				IsDestructive: false,
			},
			{
				SQL:           "ALTER TABLE modified_table DROP COLUMN old_column",
				Type:          migration.StatementTypeDropColumn,
				Description:   "Drop column old_column from modified_table",
				IsDestructive: true,
			},
			{
				SQL:           "DROP TABLE old_table",
				Type:          migration.StatementTypeDropTable,
				Description:   "Drop table old_table",
				IsDestructive: true,
			},
		},
		Warnings: []string{
			"Dropping table 'old_table' will permanently delete all data",
			"Dropping column 'old_column' may result in data loss",
		},
	}
}

// Benchmark tests
func BenchmarkDisplayChangeSummary(b *testing.B) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)
	diff := createTestSchemaDiff()
	plan := createTestMigrationPlan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cs.DisplayChangeSummary(diff, plan)
	}
}

func BenchmarkCountDestructiveOperations(b *testing.B) {
	service := NewConfirmationService(false)
	cs := service.(*confirmationService)
	plan := createTestMigrationPlan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cs.countDestructiveOperations(plan)
	}
}
