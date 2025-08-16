package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/schema"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockValidatorDisplayService for testing
type MockValidatorDisplayService struct {
	progressCalls []ProgressCall
	infoCalls     []string
	errorCalls    []string
	debugCalls    []string
	warningCalls  []string
}

func (m *MockValidatorDisplayService) ShowProgress(current, total int, message string) {
	m.progressCalls = append(m.progressCalls, ProgressCall{
		Current: current,
		Total:   total,
		Message: message,
	})
}

func (m *MockValidatorDisplayService) Info(message string) {
	m.infoCalls = append(m.infoCalls, message)
}

func (m *MockValidatorDisplayService) Error(message string) {
	m.errorCalls = append(m.errorCalls, message)
}

func (m *MockValidatorDisplayService) Debug(message string) {
	m.debugCalls = append(m.debugCalls, message)
}

func (m *MockValidatorDisplayService) Warning(message string) {
	m.warningCalls = append(m.warningCalls, message)
}

func TestNewBackupValidator(t *testing.T) {
	validator := NewBackupValidator()

	if validator == nil {
		t.Fatal("NewBackupValidator() returned nil")
	}

	// Test that it implements the interface
	_, ok := validator.(BackupValidator)
	if !ok {
		t.Error("NewBackupValidator() does not implement BackupValidator interface")
	}
}

func TestBackupValidatorImpl_SetDisplayService(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)
	mockDisplay := &MockValidatorDisplayService{}

	validator.SetDisplayService(mockDisplay)

	if validator.displayService != mockDisplay {
		t.Error("SetDisplayService() did not set the display service correctly")
	}
}

func TestBackupValidatorImpl_CalculateChecksum(t *testing.T) {
	validator := NewBackupValidator()

	testData := []byte("test data for checksum")
	checksum := validator.CalculateChecksum(testData)

	if checksum == "" {
		t.Error("CalculateChecksum() returned empty string")
	}

	if len(checksum) != 64 { // SHA-256 produces 64 character hex string
		t.Errorf("Expected checksum length of 64, got %d", len(checksum))
	}

	// Test consistency
	checksum2 := validator.CalculateChecksum(testData)
	if checksum != checksum2 {
		t.Error("CalculateChecksum() is not consistent")
	}
}

func TestBackupValidatorImpl_VerifyChecksum(t *testing.T) {
	validator := NewBackupValidator()

	testData := []byte("test data for checksum verification")
	expectedChecksum := validator.CalculateChecksum(testData)

	// Test valid checksum
	if !validator.VerifyChecksum(testData, expectedChecksum) {
		t.Error("VerifyChecksum() failed for valid checksum")
	}

	// Test invalid checksum
	if validator.VerifyChecksum(testData, "invalid_checksum") {
		t.Error("VerifyChecksum() passed for invalid checksum")
	}
}

func TestBackupValidatorImpl_ValidateIntegrity_NilBackup(t *testing.T) {
	validator := NewBackupValidator()
	mockDisplay := &MockValidatorDisplayService{}
	validator.(*BackupValidatorImpl).SetDisplayService(mockDisplay)

	ctx := context.Background()
	err := validator.ValidateIntegrity(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil backup")
	}

	// The error is returned immediately for nil backup, so no logging occurs
	// This is correct behavior - we don't need to log when input validation fails
}

func TestBackupValidatorImpl_ValidateIntegrity_Success(t *testing.T) {
	validator := NewBackupValidator()
	mockDisplay := &MockValidatorDisplayService{}
	validator.(*BackupValidatorImpl).SetDisplayService(mockDisplay)

	// Create a valid backup
	backup := createValidTestBackup()

	ctx := context.Background()
	err := validator.ValidateIntegrity(ctx, backup)

	if err != nil {
		t.Errorf("ValidateIntegrity() error = %v", err)
	}

	// Verify progress tracking
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls to be made")
	}

	// Verify info logging
	if len(mockDisplay.infoCalls) == 0 {
		t.Error("Expected info calls to be made")
	}
}

func TestBackupValidatorImpl_ValidateIntegrity_InvalidChecksum(t *testing.T) {
	validator := NewBackupValidator()

	backup := createValidTestBackup()
	backup.Checksum = "invalid_checksum"

	ctx := context.Background()
	err := validator.ValidateIntegrity(ctx, backup)

	if err == nil {
		t.Error("Expected error for invalid checksum")
	}

	// Should be a corruption error
	if _, ok := err.(*BackupError); !ok {
		t.Error("Expected BackupError type")
	}
}

func TestBackupValidatorImpl_ValidateCompleteness_NilInputs(t *testing.T) {
	validator := NewBackupValidator()
	ctx := context.Background()

	tests := []struct {
		name           string
		backup         *Backup
		originalSchema *schema.Schema
		expectError    bool
	}{
		{
			name:           "nil backup",
			backup:         nil,
			originalSchema: &schema.Schema{},
			expectError:    true,
		},
		{
			name:           "nil original schema",
			backup:         createValidTestBackup(),
			originalSchema: nil,
			expectError:    true,
		},
		{
			name:           "both nil",
			backup:         nil,
			originalSchema: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCompleteness(ctx, tt.backup, tt.originalSchema)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateCompleteness() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestBackupValidatorImpl_ValidateCompleteness_Success(t *testing.T) {
	validator := NewBackupValidator()
	mockDisplay := &MockValidatorDisplayService{}
	validator.(*BackupValidatorImpl).SetDisplayService(mockDisplay)

	backup := createValidTestBackup()
	originalSchema := createTestSchema()

	ctx := context.Background()
	err := validator.ValidateCompleteness(ctx, backup, originalSchema)

	if err != nil {
		t.Errorf("ValidateCompleteness() error = %v", err)
	}

	// Verify progress tracking
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls to be made")
	}
}

func TestBackupValidatorImpl_ValidateCompleteness_MissingTable(t *testing.T) {
	validator := NewBackupValidator()

	backup := createValidTestBackup()
	originalSchema := createTestSchema()

	// Add an extra table to original schema
	originalSchema.Tables["missing_table"] = &schema.Table{
		Name: "missing_table",
		Columns: map[string]*schema.Column{
			"id": {Name: "id", DataType: "int(11)", Position: 1},
		},
	}

	ctx := context.Background()
	err := validator.ValidateCompleteness(ctx, backup, originalSchema)

	if err == nil {
		t.Error("Expected error for missing table")
	}
}

func TestBackupValidatorImpl_ValidateRestorability_NilBackup(t *testing.T) {
	validator := NewBackupValidator()

	ctx := context.Background()
	err := validator.ValidateRestorability(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil backup")
	}
}

func TestBackupValidatorImpl_ValidateRestorability_Success(t *testing.T) {
	validator := NewBackupValidator()
	mockDisplay := &MockValidatorDisplayService{}
	validator.(*BackupValidatorImpl).SetDisplayService(mockDisplay)

	backup := createValidTestBackup()

	ctx := context.Background()
	err := validator.ValidateRestorability(ctx, backup)

	if err != nil {
		t.Errorf("ValidateRestorability() error = %v", err)
	}

	// Verify progress tracking
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls to be made")
	}
}

func TestBackupValidatorImpl_ValidateRestorability_InvalidDDL(t *testing.T) {
	validator := NewBackupValidator()

	backup := createValidTestBackup()
	backup.DataDefinitions = []string{
		"CREATE TABLE users (id int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)", // Missing closing parenthesis
	}

	ctx := context.Background()
	err := validator.ValidateRestorability(ctx, backup)

	if err == nil {
		t.Error("Expected error for invalid DDL syntax")
	}
}

func TestBackupValidatorImpl_ValidateSQLSyntax(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	tests := []struct {
		name          string
		sql           string
		statementType string
		expectError   bool
	}{
		{
			name:          "valid SQL",
			sql:           "CREATE TABLE users (id INT PRIMARY KEY)",
			statementType: "DDL",
			expectError:   false,
		},
		{
			name:          "empty SQL",
			sql:           "",
			statementType: "DDL",
			expectError:   true,
		},
		{
			name:          "unbalanced parentheses",
			sql:           "CREATE TABLE users (id INT PRIMARY KEY",
			statementType: "DDL",
			expectError:   true,
		},
		{
			name:          "unbalanced quotes",
			sql:           "CREATE TABLE users (name VARCHAR(255) DEFAULT 'test)",
			statementType: "DDL",
			expectError:   true,
		},
		{
			name:          "unbalanced backticks",
			sql:           "CREATE TABLE `users (id INT)",
			statementType: "DDL",
			expectError:   true,
		},
		{
			name:          "balanced nested parentheses",
			sql:           "CREATE TABLE users (id INT, CHECK (id > 0 AND (status = 'active' OR status = 'pending')))",
			statementType: "DDL",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSQLSyntax(tt.sql, tt.statementType)
			if (err != nil) != tt.expectError {
				t.Errorf("validateSQLSyntax() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestBackupValidatorImpl_HasBalancedParentheses(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "balanced parentheses",
			sql:      "CREATE TABLE users (id INT, name VARCHAR(255))",
			expected: true,
		},
		{
			name:     "unbalanced opening",
			sql:      "CREATE TABLE users (id INT, name VARCHAR(255)",
			expected: false,
		},
		{
			name:     "unbalanced closing",
			sql:      "CREATE TABLE users id INT, name VARCHAR(255))",
			expected: false,
		},
		{
			name:     "nested balanced",
			sql:      "CREATE TABLE users (id INT, CHECK (id > 0))",
			expected: true,
		},
		{
			name:     "parentheses in string",
			sql:      "CREATE TABLE users (name VARCHAR(255) DEFAULT '(test)')",
			expected: true,
		},
		{
			name:     "no parentheses",
			sql:      "CREATE TABLE users",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.hasBalancedParentheses(tt.sql)
			if result != tt.expected {
				t.Errorf("hasBalancedParentheses() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBackupValidatorImpl_HasBalancedQuotes(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "balanced single quotes",
			sql:      "SELECT * FROM users WHERE name = 'test'",
			expected: true,
		},
		{
			name:     "balanced double quotes",
			sql:      "SELECT * FROM users WHERE name = \"test\"",
			expected: true,
		},
		{
			name:     "unbalanced single quotes",
			sql:      "SELECT * FROM users WHERE name = 'test",
			expected: false,
		},
		{
			name:     "unbalanced double quotes",
			sql:      "SELECT * FROM users WHERE name = \"test",
			expected: false,
		},
		{
			name:     "escaped quotes",
			sql:      "SELECT * FROM users WHERE name = 'don\\'t'",
			expected: true,
		},
		{
			name:     "no quotes",
			sql:      "SELECT * FROM users WHERE id = 1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.hasBalancedQuotes(tt.sql)
			if result != tt.expected {
				t.Errorf("hasBalancedQuotes() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBackupValidatorImpl_HasBalancedBackticks(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "balanced backticks",
			sql:      "CREATE TABLE `users` (`id` INT)",
			expected: true,
		},
		{
			name:     "unbalanced backticks",
			sql:      "CREATE TABLE `users (`id` INT)",
			expected: false,
		},
		{
			name:     "no backticks",
			sql:      "CREATE TABLE users (id INT)",
			expected: true,
		},
		{
			name:     "multiple balanced pairs",
			sql:      "CREATE TABLE `users` (`id` INT, `name` VARCHAR(255))",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.hasBalancedBackticks(tt.sql)
			if result != tt.expected {
				t.Errorf("hasBalancedBackticks() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBackupValidatorImpl_PerformDryRunRestore(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)
	mockDisplay := &MockValidatorDisplayService{}
	validator.SetDisplayService(mockDisplay)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	backup := createValidTestBackup()

	// Mock the dry-run restore operations
	mock.ExpectExec("CREATE DATABASE IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("USE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock schema extraction for validation
	mock.ExpectQuery("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME"}).AddRow("users"))
	mock.ExpectQuery("SELECT COLUMN_NAME").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "DATA_TYPE", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA", "ORDINAL_POSITION", "COLUMN_TYPE"}).
			AddRow("id", "int", "NO", nil, "auto_increment", 1, "int(11)"))
	mock.ExpectQuery("SELECT INDEX_NAME").
		WillReturnRows(sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE", "INDEX_TYPE", "SEQ_IN_INDEX"}).
			AddRow("PRIMARY", "id", 0, "BTREE", 1))

	mock.ExpectExec("DROP DATABASE IF EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	err = validator.PerformDryRunRestore(ctx, backup, db)

	if err != nil {
		t.Errorf("PerformDryRunRestore() error = %v", err)
	}

	// Verify progress tracking
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls to be made")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestBackupValidatorImpl_CreateValidationReport(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	backup := createValidTestBackup()
	originalSchema := createTestSchema()

	ctx := context.Background()
	report, err := validator.CreateValidationReport(ctx, backup, originalSchema)

	if err != nil {
		t.Errorf("CreateValidationReport() error = %v", err)
	}

	if report == nil {
		t.Fatal("Expected validation report to be returned")
	}

	if !report.Valid {
		t.Errorf("Expected backup to be valid, got errors: %v", report.Errors)
	}

	if !report.ChecksumValid {
		t.Error("Expected checksum to be valid")
	}

	if report.CheckedAt.IsZero() {
		t.Error("Expected CheckedAt to be set")
	}
}

func TestBackupValidatorImpl_CreateValidationReport_InvalidBackup(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	backup := createValidTestBackup()
	backup.Checksum = "invalid_checksum" // Make backup invalid

	ctx := context.Background()
	report, err := validator.CreateValidationReport(ctx, backup, nil)

	if err != nil {
		t.Errorf("CreateValidationReport() error = %v", err)
	}

	if report.Valid {
		t.Error("Expected backup to be invalid")
	}

	if len(report.Errors) == 0 {
		t.Error("Expected validation errors to be reported")
	}

	if report.ChecksumValid {
		t.Error("Expected checksum to be invalid")
	}
}

// Helper functions for creating test data

func createValidTestBackup() *Backup {
	backup := &Backup{
		ID: "test-backup-123",
		Metadata: &BackupMetadata{
			ID:              "test-backup-123",
			DatabaseName:    "testdb",
			CreatedAt:       time.Now().Add(-1 * time.Hour),
			CreatedBy:       "test-user",
			Description:     "Test backup",
			Size:            1024,
			CompressedSize:  512,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
			StorageLocation: "/tmp/backups/test-backup-123",
			Checksum:        "placeholder", // Will be calculated
		},
		SchemaSnapshot: createTestSchema(),
		DataDefinitions: []string{
			"CREATE TABLE `users` (`id` int(11) NOT NULL AUTO_INCREMENT, `name` varchar(255) DEFAULT NULL, PRIMARY KEY (`id`))",
		},
		Triggers:   []TriggerDefinition{},
		Views:      []ViewDefinition{},
		Procedures: []ProcedureDefinition{},
		Functions:  []FunctionDefinition{},
	}

	// Calculate proper checksum
	_ = backup.CalculateChecksum()

	return backup
}

func createTestSchema() *schema.Schema {
	return &schema.Schema{
		Name: "testdb",
		Tables: map[string]*schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]*schema.Column{
					"id": {
						Name:       "id",
						DataType:   "int(11)",
						IsNullable: false,
						Position:   1,
					},
					"name": {
						Name:         "name",
						DataType:     "varchar(255)",
						IsNullable:   true,
						DefaultValue: nil,
						Position:     2,
					},
				},
				Indexes: []*schema.Index{
					{
						Name:      "PRIMARY",
						TableName: "users",
						Columns:   []string{"id"},
						IsUnique:  true,
						IsPrimary: true,
						IndexType: "BTREE",
					},
				},
				Constraints: make(map[string]*schema.Constraint),
			},
		},
		Indexes: make(map[string]*schema.Index),
	}
}

func TestBackupValidatorImpl_ValidateBackupStructure(t *testing.T) {
	validator := NewBackupValidator().(*BackupValidatorImpl)

	tests := []struct {
		name        string
		backup      *Backup
		expectError bool
	}{
		{
			name:        "valid backup",
			backup:      createValidTestBackup(),
			expectError: false,
		},
		{
			name: "missing ID",
			backup: &Backup{
				Metadata:        &BackupMetadata{},
				SchemaSnapshot:  &schema.Schema{},
				DataDefinitions: []string{},
				Triggers:        []TriggerDefinition{},
				Views:           []ViewDefinition{},
				Procedures:      []ProcedureDefinition{},
				Functions:       []FunctionDefinition{},
				Checksum:        "test",
			},
			expectError: true,
		},
		{
			name: "missing metadata",
			backup: &Backup{
				ID:              "test",
				SchemaSnapshot:  &schema.Schema{},
				DataDefinitions: []string{},
				Triggers:        []TriggerDefinition{},
				Views:           []ViewDefinition{},
				Procedures:      []ProcedureDefinition{},
				Functions:       []FunctionDefinition{},
				Checksum:        "test",
			},
			expectError: true,
		},
		{
			name: "missing schema snapshot",
			backup: &Backup{
				ID:              "test",
				Metadata:        &BackupMetadata{},
				DataDefinitions: []string{},
				Triggers:        []TriggerDefinition{},
				Views:           []ViewDefinition{},
				Procedures:      []ProcedureDefinition{},
				Functions:       []FunctionDefinition{},
				Checksum:        "test",
			},
			expectError: true,
		},
		{
			name: "nil data definitions",
			backup: &Backup{
				ID:             "test",
				Metadata:       &BackupMetadata{},
				SchemaSnapshot: &schema.Schema{},
				Triggers:       []TriggerDefinition{},
				Views:          []ViewDefinition{},
				Procedures:     []ProcedureDefinition{},
				Functions:      []FunctionDefinition{},
				Checksum:       "test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateBackupStructure(tt.backup)
			if (err != nil) != tt.expectError {
				t.Errorf("validateBackupStructure() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
