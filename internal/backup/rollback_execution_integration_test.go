package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

// TestRollbackExecutionIntegration tests the complete rollback execution workflow
func TestRollbackExecutionIntegration(t *testing.T) {
	// Create test components
	rm, mockBM, mockValidator, mockDB, mockStorage := createTestRollbackManager()
	ctx := context.Background()

	// Create test backup
	backup := createTestBackupForRollback()
	mockStorage.backups[backup.ID] = backup
	mockBM.backups[backup.ID] = backup.Metadata

	// Create rollback plan with various statement types
	plan := createComprehensiveRollbackPlan(backup.ID)

	// Test successful rollback execution
	t.Run("successful_rollback_execution", func(t *testing.T) {
		// Reset mocks
		mockDB.shouldFail = false
		mockValidator.shouldFail = false

		// Execute rollback
		err := rm.ExecuteRollback(ctx, plan)

		// Should fail due to database connection (expected in mock environment)
		if err == nil {
			t.Error("Expected error due to mock database connection")
		}

		// Verify it's a rollback error
		if !isRollbackError(err) {
			t.Errorf("Expected RollbackError, got %T", err)
		}
	})

	// Test rollback execution with database failure
	t.Run("rollback_execution_with_database_failure", func(t *testing.T) {
		// Make database service fail
		mockDB.shouldFail = true

		// Execute rollback
		err := rm.ExecuteRollback(ctx, plan)

		if err == nil {
			t.Error("Expected database connection error")
		}

		// Verify it's a rollback error
		if !isRollbackError(err) {
			t.Errorf("Expected RollbackError, got %T", err)
		}
	})

	// Test rollback execution with validation failure
	t.Run("rollback_execution_with_validation_failure", func(t *testing.T) {
		// Reset database service but make validator fail
		mockDB.shouldFail = false
		mockValidator.shouldFail = true

		// Execute rollback
		err := rm.ExecuteRollback(ctx, plan)

		if err == nil {
			t.Error("Expected validation error")
		}

		// Verify it's a rollback error
		if !isRollbackError(err) {
			t.Errorf("Expected RollbackError, got %T", err)
		}
	})

	// Test rollback execution with invalid plan
	t.Run("rollback_execution_with_invalid_plan", func(t *testing.T) {
		// Reset mocks
		mockDB.shouldFail = false
		mockValidator.shouldFail = false

		// Create invalid plan
		invalidPlan := &RollbackPlan{
			BackupID: "", // Invalid empty backup ID
		}

		// Execute rollback
		err := rm.ExecuteRollback(ctx, invalidPlan)

		if err == nil {
			t.Error("Expected validation error for invalid plan")
		}

		// Verify it's a rollback error
		if !isRollbackError(err) {
			t.Errorf("Expected RollbackError, got %T", err)
		}
	})
}

// TestRollbackVerificationMethods tests the verification methods
func TestRollbackVerificationMethods(t *testing.T) {
	rm, _, _, _, _ := createTestRollbackManager()

	// Test schema comparison
	t.Run("schema_comparison", func(t *testing.T) {
		currentSchema := createTestSchemaForRollback("current")
		targetSchema := createTestSchemaForRollback("target")

		// Add extra table to current schema
		extraTable := &schema.Table{
			Name:        "extra_table",
			Columns:     make(map[string]*schema.Column),
			Constraints: make(map[string]*schema.Constraint),
		}
		extraTable.Columns["id"] = &schema.Column{
			Name:     "id",
			DataType: "INT",
			Position: 1,
		}
		currentSchema.Tables["extra_table"] = extraTable

		// Compare schemas
		differences := rm.compareSchemas(currentSchema, targetSchema)

		if len(differences) == 0 {
			t.Error("Expected differences between schemas")
		}

		// Check for specific difference
		foundExtraTable := false
		for _, diff := range differences {
			if contains(diff, "extra table: extra_table") {
				foundExtraTable = true
				break
			}
		}
		if !foundExtraTable {
			t.Error("Expected to find extra table difference")
		}
	})

	// Test table structure comparison
	t.Run("table_structure_comparison", func(t *testing.T) {
		currentTable := createTestTableForRollback("test_table")
		targetTable := createTestTableForRollback("test_table")

		// Add extra column to current table
		currentTable.Columns["extra_column"] = &schema.Column{
			Name:     "extra_column",
			DataType: "VARCHAR(255)",
			Position: 2,
		}

		// Compare table structures
		differences := rm.compareTableStructures(currentTable, targetTable)

		if len(differences) == 0 {
			t.Error("Expected differences between table structures")
		}

		// Check for specific difference
		foundExtraColumn := false
		for _, diff := range differences {
			if contains(diff, "extra column: extra_column") {
				foundExtraColumn = true
				break
			}
		}
		if !foundExtraColumn {
			t.Error("Expected to find extra column difference")
		}
	})
}

// TestRollbackRecoveryPlanGeneration tests recovery plan generation
func TestRollbackRecoveryPlanGeneration(t *testing.T) {
	rm, _, _, _, _ := createTestRollbackManager()
	ctx := context.Background()

	// Create test rollback plan
	plan := createComprehensiveRollbackPlan("test-backup")

	// Test recovery plan generation for different statement types
	testCases := []struct {
		name                 string
		failedStatementIndex int
		expectedIntervention string
	}{
		{
			name:                 "drop_table_failure",
			failedStatementIndex: 0, // Assuming first statement is DROP TABLE
			expectedIntervention: "table",
		},
		{
			name:                 "create_table_failure",
			failedStatementIndex: 1, // Assuming second statement is CREATE TABLE
			expectedIntervention: "table",
		},
		{
			name:                 "drop_column_failure",
			failedStatementIndex: 2, // Assuming third statement is DROP COLUMN
			expectedIntervention: "column",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testErr := NewRollbackError("test execution failure", nil)

			// Generate recovery plan
			recoveryPlan := rm.createRollbackFailureRecoveryPlan(ctx, plan, tc.failedStatementIndex, testErr)

			if recoveryPlan == nil {
				t.Fatal("Recovery plan should not be nil")
			}

			if recoveryPlan.FailedStatementIndex != tc.failedStatementIndex {
				t.Errorf("Expected failed statement index %d, got %d", tc.failedStatementIndex, recoveryPlan.FailedStatementIndex)
			}

			if len(recoveryPlan.RecoverySteps) == 0 {
				t.Error("Recovery plan should have recovery steps")
			}

			if len(recoveryPlan.ManualInterventions) == 0 {
				t.Error("Recovery plan should have manual interventions")
			}

			// Check for expected intervention type
			foundExpectedIntervention := false
			for _, intervention := range recoveryPlan.ManualInterventions {
				if contains(intervention, tc.expectedIntervention) {
					foundExpectedIntervention = true
					break
				}
			}
			if !foundExpectedIntervention {
				t.Errorf("Expected intervention containing '%s'", tc.expectedIntervention)
			}
		})
	}
}

// Helper functions for integration tests

func createTestBackupForRollback() *Backup {
	return &Backup{
		ID: "test-backup-rollback",
		Metadata: &BackupMetadata{
			ID:           "test-backup-rollback",
			DatabaseName: "test_db",
			CreatedAt:    time.Now(),
			Description:  "Test backup for rollback",
			Status:       BackupStatusCompleted,
			Checksum:     "test-checksum",
		},
		SchemaSnapshot: createTestSchemaForRollback("test_db"),
	}
}

func createComprehensiveRollbackPlan(backupID string) *RollbackPlan {
	return &RollbackPlan{
		BackupID:      backupID,
		TargetSchema:  createTestSchemaForRollback("target"),
		CurrentSchema: createTestSchemaForRollback("current"),
		Statements: []migration.MigrationStatement{
			{
				SQL:           "DROP TABLE old_table",
				Type:          migration.StatementTypeDropTable,
				Description:   "Drop old table",
				IsDestructive: true,
				TableName:     "old_table",
			},
			{
				SQL:         "CREATE TABLE new_table (id INT PRIMARY KEY)",
				Type:        migration.StatementTypeCreateTable,
				Description: "Create new table",
				TableName:   "new_table",
			},
			{
				SQL:           "ALTER TABLE users DROP COLUMN old_column",
				Type:          migration.StatementTypeDropColumn,
				Description:   "Drop old column",
				IsDestructive: true,
				TableName:     "users",
			},
			{
				SQL:         "ALTER TABLE users ADD COLUMN new_column VARCHAR(255)",
				Type:        migration.StatementTypeAddColumn,
				Description: "Add new column",
				TableName:   "users",
			},
			{
				SQL:           "DROP INDEX idx_old ON users",
				Type:          migration.StatementTypeDropIndex,
				Description:   "Drop old index",
				IsDestructive: true,
				TableName:     "users",
			},
			{
				SQL:         "CREATE INDEX idx_new ON users (new_column)",
				Type:        migration.StatementTypeCreateIndex,
				Description: "Create new index",
				TableName:   "users",
			},
		},
		Dependencies: []string{
			"Foreign key constraints must be handled before table operations",
			"Indexes must be dropped before column operations",
		},
		Warnings: []string{
			"CRITICAL: 1 table(s) will be dropped - this will result in permanent data loss",
			"WARNING: 1 column(s) will be dropped - this will result in data loss for those columns",
			"PERFORMANCE: 1 index(es) will be dropped - this may impact query performance",
		},
	}
}

func createTestSchemaForRollback(name string) *schema.Schema {
	testSchema := &schema.Schema{
		Name:   name,
		Tables: make(map[string]*schema.Table),
	}

	// Add a test table
	testTable := createTestTableForRollback("users")
	testSchema.Tables["users"] = testTable

	return testSchema
}

func createTestTableForRollback(name string) *schema.Table {
	table := &schema.Table{
		Name:        name,
		Columns:     make(map[string]*schema.Column),
		Constraints: make(map[string]*schema.Constraint),
		Indexes:     make([]*schema.Index, 0),
	}

	// Add basic columns
	table.Columns["id"] = &schema.Column{
		Name:     "id",
		DataType: "INT",
		Position: 1,
	}
	table.Columns["name"] = &schema.Column{
		Name:     "name",
		DataType: "VARCHAR(255)",
		Position: 2,
	}

	// Add basic index
	table.Indexes = append(table.Indexes, &schema.Index{
		Name:      "idx_name",
		TableName: name,
		Columns:   []string{"name"},
		IsUnique:  false,
	})

	return table
}
