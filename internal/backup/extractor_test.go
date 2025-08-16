package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/schema"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockExtractorDisplayService for testing
type MockExtractorDisplayService struct {
	progressCalls []ProgressCall
	infoCalls     []string
	errorCalls    []string
	debugCalls    []string
}

type ProgressCall struct {
	Current int
	Total   int
	Message string
}

func (m *MockExtractorDisplayService) ShowProgress(current, total int, message string) {
	m.progressCalls = append(m.progressCalls, ProgressCall{
		Current: current,
		Total:   total,
		Message: message,
	})
}

func (m *MockExtractorDisplayService) Info(message string) {
	m.infoCalls = append(m.infoCalls, message)
}

func (m *MockExtractorDisplayService) Error(message string) {
	m.errorCalls = append(m.errorCalls, message)
}

func (m *MockExtractorDisplayService) Debug(message string) {
	m.debugCalls = append(m.debugCalls, message)
}

func TestNewSchemaExtractor(t *testing.T) {
	extractor := NewSchemaExtractor()

	if extractor == nil {
		t.Fatal("NewSchemaExtractor() returned nil")
	}

	if extractor.queryTimeout != 60*time.Second {
		t.Errorf("Expected default timeout of 60s, got %v", extractor.queryTimeout)
	}

	if extractor.displayService != nil {
		t.Error("Expected displayService to be nil initially")
	}
}

func TestNewSchemaExtractorWithTimeout(t *testing.T) {
	timeout := 30 * time.Second
	extractor := NewSchemaExtractorWithTimeout(timeout)

	if extractor == nil {
		t.Fatal("NewSchemaExtractorWithTimeout() returned nil")
	}

	if extractor.queryTimeout != timeout {
		t.Errorf("Expected timeout of %v, got %v", timeout, extractor.queryTimeout)
	}
}

func TestSetDisplayService(t *testing.T) {
	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}

	extractor.SetDisplayService(mockDisplay)

	if extractor.displayService != mockDisplay {
		t.Error("SetDisplayService() did not set the display service correctly")
	}
}

func TestExtractCompleteSchema_NilDB(t *testing.T) {
	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	_, err := extractor.ExtractCompleteSchema(ctx, nil, "testdb")

	if err == nil {
		t.Error("Expected error for nil database connection")
	}

	if len(mockDisplay.errorCalls) == 0 {
		t.Error("Expected error to be logged")
	}
}

func TestExtractCompleteSchema_EmptySchemaName(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	_, err = extractor.ExtractCompleteSchema(ctx, db, "")

	if err == nil {
		t.Error("Expected error for empty schema name")
	}

	if len(mockDisplay.errorCalls) == 0 {
		t.Error("Expected error to be logged")
	}
}

func TestExtractCompleteSchema_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock schema extraction queries
	setupMockSchemaQueries(mock)

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	backup, err := extractor.ExtractCompleteSchema(ctx, db, "testdb")

	if err != nil {
		t.Errorf("ExtractCompleteSchema() error = %v", err)
	}

	if backup == nil {
		t.Fatal("Expected backup to be returned")
	}

	if backup.ID == "" {
		t.Error("Expected backup ID to be generated")
	}

	if backup.SchemaSnapshot == nil {
		t.Error("Expected schema snapshot to be present")
	}

	if backup.Checksum == "" {
		t.Error("Expected checksum to be calculated")
	}

	// Verify progress tracking
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls to be made")
	}

	// Verify info logging
	if len(mockDisplay.infoCalls) == 0 {
		t.Error("Expected info calls to be made")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestExtractTableDDL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock table list query
	mock.ExpectQuery("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME"}).
			AddRow("users").
			AddRow("orders"))

	// Mock SHOW CREATE TABLE queries
	mock.ExpectQuery("SHOW CREATE TABLE users").
		WillReturnRows(sqlmock.NewRows([]string{"Table", "Create Table"}).
			AddRow("users", "CREATE TABLE `users` (`id` int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`))"))

	mock.ExpectQuery("SHOW CREATE TABLE orders").
		WillReturnRows(sqlmock.NewRows([]string{"Table", "Create Table"}).
			AddRow("orders", "CREATE TABLE `orders` (`id` int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`))"))

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	ddlStatements, err := extractor.extractTableDDL(ctx, db, "testdb")

	if err != nil {
		t.Errorf("extractTableDDL() error = %v", err)
	}

	if len(ddlStatements) != 2 {
		t.Errorf("Expected 2 DDL statements, got %d", len(ddlStatements))
	}

	// Verify progress tracking for individual tables
	if len(mockDisplay.progressCalls) == 0 {
		t.Error("Expected progress calls for table extraction")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestExtractTriggers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock triggers query
	mock.ExpectQuery("SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, ACTION_TIMING, EVENT_MANIPULATION, ACTION_STATEMENT FROM INFORMATION_SCHEMA.TRIGGERS").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TRIGGER_NAME", "EVENT_OBJECT_TABLE", "ACTION_TIMING", "EVENT_MANIPULATION", "ACTION_STATEMENT"}).
			AddRow("user_audit", "users", "AFTER", "INSERT", "INSERT INTO audit_log VALUES (NEW.id, 'INSERT', NOW())").
			AddRow("order_update", "orders", "BEFORE", "UPDATE", "SET NEW.updated_at = NOW()"))

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	triggers, err := extractor.extractTriggers(ctx, db, "testdb")

	if err != nil {
		t.Errorf("extractTriggers() error = %v", err)
	}

	if len(triggers) != 2 {
		t.Errorf("Expected 2 triggers, got %d", len(triggers))
	}

	// Verify trigger details
	if triggers[0].Name != "user_audit" {
		t.Errorf("Expected trigger name 'user_audit', got %s", triggers[0].Name)
	}

	if triggers[0].Table != "users" {
		t.Errorf("Expected trigger table 'users', got %s", triggers[0].Table)
	}

	if triggers[0].Timing != "AFTER" {
		t.Errorf("Expected trigger timing 'AFTER', got %s", triggers[0].Timing)
	}

	if triggers[0].Event != "INSERT" {
		t.Errorf("Expected trigger event 'INSERT', got %s", triggers[0].Event)
	}

	// Verify debug logging
	if len(mockDisplay.debugCalls) == 0 {
		t.Error("Expected debug calls for trigger extraction")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestExtractViews(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock views query
	mock.ExpectQuery("SELECT TABLE_NAME, VIEW_DEFINITION FROM INFORMATION_SCHEMA.VIEWS").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "VIEW_DEFINITION"}).
			AddRow("user_summary", "SELECT id, name, email FROM users WHERE active = 1").
			AddRow("order_totals", "SELECT user_id, COUNT(*) as order_count, SUM(total) as total_amount FROM orders GROUP BY user_id"))

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	views, err := extractor.extractViews(ctx, db, "testdb")

	if err != nil {
		t.Errorf("extractViews() error = %v", err)
	}

	if len(views) != 2 {
		t.Errorf("Expected 2 views, got %d", len(views))
	}

	// Verify view details
	if views[0].Name != "user_summary" {
		t.Errorf("Expected view name 'user_summary', got %s", views[0].Name)
	}

	if views[0].Definition == "" {
		t.Error("Expected view definition to be present")
	}

	// Verify debug logging
	if len(mockDisplay.debugCalls) == 0 {
		t.Error("Expected debug calls for view extraction")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestExtractProcedures(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock procedure names query
	mock.ExpectQuery("SELECT ROUTINE_NAME FROM INFORMATION_SCHEMA.ROUTINES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"ROUTINE_NAME"}).
			AddRow("GetUserById").
			AddRow("UpdateUserStatus"))

	// Mock SHOW CREATE PROCEDURE queries
	mock.ExpectQuery("SHOW CREATE PROCEDURE GetUserById").
		WillReturnRows(sqlmock.NewRows([]string{"Procedure", "sql_mode", "Create Procedure", "character_set_client", "collation_connection", "Database Collation"}).
			AddRow("GetUserById", "STRICT_TRANS_TABLES", "CREATE PROCEDURE GetUserById(IN user_id INT) BEGIN SELECT * FROM users WHERE id = user_id; END", "utf8mb4", "utf8mb4_0900_ai_ci", "utf8mb4_0900_ai_ci"))

	mock.ExpectQuery("SHOW CREATE PROCEDURE UpdateUserStatus").
		WillReturnRows(sqlmock.NewRows([]string{"Procedure", "sql_mode", "Create Procedure", "character_set_client", "collation_connection", "Database Collation"}).
			AddRow("UpdateUserStatus", "STRICT_TRANS_TABLES", "CREATE PROCEDURE UpdateUserStatus(IN user_id INT, IN status VARCHAR(20)) BEGIN UPDATE users SET status = status WHERE id = user_id; END", "utf8mb4", "utf8mb4_0900_ai_ci", "utf8mb4_0900_ai_ci"))

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	procedures, err := extractor.extractProcedures(ctx, db, "testdb")

	if err != nil {
		t.Errorf("extractProcedures() error = %v", err)
	}

	if len(procedures) != 2 {
		t.Errorf("Expected 2 procedures, got %d", len(procedures))
	}

	// Verify procedure details
	if procedures[0].Name != "GetUserById" {
		t.Errorf("Expected procedure name 'GetUserById', got %s", procedures[0].Name)
	}

	if procedures[0].Definition == "" {
		t.Error("Expected procedure definition to be present")
	}

	// Verify debug logging
	if len(mockDisplay.debugCalls) == 0 {
		t.Error("Expected debug calls for procedure extraction")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestExtractFunctions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock function names query
	mock.ExpectQuery("SELECT ROUTINE_NAME FROM INFORMATION_SCHEMA.ROUTINES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"ROUTINE_NAME"}).
			AddRow("CalculateDiscount").
			AddRow("FormatCurrency"))

	// Mock SHOW CREATE FUNCTION queries
	mock.ExpectQuery("SHOW CREATE FUNCTION CalculateDiscount").
		WillReturnRows(sqlmock.NewRows([]string{"Function", "sql_mode", "Create Function", "character_set_client", "collation_connection", "Database Collation"}).
			AddRow("CalculateDiscount", "STRICT_TRANS_TABLES", "CREATE FUNCTION CalculateDiscount(amount DECIMAL(10,2), rate DECIMAL(5,2)) RETURNS DECIMAL(10,2) READS SQL DATA DETERMINISTIC RETURN amount * (rate / 100)", "utf8mb4", "utf8mb4_0900_ai_ci", "utf8mb4_0900_ai_ci"))

	mock.ExpectQuery("SHOW CREATE FUNCTION FormatCurrency").
		WillReturnRows(sqlmock.NewRows([]string{"Function", "sql_mode", "Create Function", "character_set_client", "collation_connection", "Database Collation"}).
			AddRow("FormatCurrency", "STRICT_TRANS_TABLES", "CREATE FUNCTION FormatCurrency(amount DECIMAL(10,2)) RETURNS VARCHAR(20) READS SQL DATA DETERMINISTIC RETURN CONCAT('$', FORMAT(amount, 2))", "utf8mb4", "utf8mb4_0900_ai_ci", "utf8mb4_0900_ai_ci"))

	extractor := NewSchemaExtractor()
	mockDisplay := &MockExtractorDisplayService{}
	extractor.SetDisplayService(mockDisplay)

	ctx := context.Background()
	functions, err := extractor.extractFunctions(ctx, db, "testdb")

	if err != nil {
		t.Errorf("extractFunctions() error = %v", err)
	}

	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	// Verify function details
	if functions[0].Name != "CalculateDiscount" {
		t.Errorf("Expected function name 'CalculateDiscount', got %s", functions[0].Name)
	}

	if functions[0].Definition == "" {
		t.Error("Expected function definition to be present")
	}

	// Verify debug logging
	if len(mockDisplay.debugCalls) == 0 {
		t.Error("Expected debug calls for function extraction")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled mock expectations: %v", err)
	}
}

func TestValidateExtractedSchema(t *testing.T) {
	extractor := NewSchemaExtractor()

	tests := []struct {
		name    string
		backup  *Backup
		wantErr bool
	}{
		{
			name:    "nil backup",
			backup:  nil,
			wantErr: true,
		},
		{
			name: "missing schema snapshot",
			backup: &Backup{
				ID: "test-backup",
			},
			wantErr: true,
		},
		{
			name: "valid backup",
			backup: &Backup{
				ID: "test-backup",
				SchemaSnapshot: &schema.Schema{
					Name: "testdb",
					Tables: map[string]*schema.Table{
						"users": {
							Name: "users",
							Columns: map[string]*schema.Column{
								"id": {
									Name:     "id",
									DataType: "int(11)",
									Position: 1,
								},
							},
						},
					},
					Indexes: make(map[string]*schema.Index),
				},
				DataDefinitions: []string{
					"CREATE TABLE `users` (`id` int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`))",
				},
			},
			wantErr: false,
		},
		{
			name: "DDL count mismatch",
			backup: &Backup{
				ID: "test-backup",
				SchemaSnapshot: &schema.Schema{
					Name: "testdb",
					Tables: map[string]*schema.Table{
						"users": {
							Name: "users",
							Columns: map[string]*schema.Column{
								"id": {
									Name:     "id",
									DataType: "int(11)",
									Position: 1,
								},
							},
						},
						"orders": {
							Name: "orders",
							Columns: map[string]*schema.Column{
								"id": {
									Name:     "id",
									DataType: "int(11)",
									Position: 1,
								},
							},
						},
					},
					Indexes: make(map[string]*schema.Index),
				},
				DataDefinitions: []string{
					"CREATE TABLE `users` (`id` int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`))",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid DDL statement",
			backup: &Backup{
				ID: "test-backup",
				SchemaSnapshot: &schema.Schema{
					Name: "testdb",
					Tables: map[string]*schema.Table{
						"users": {
							Name: "users",
							Columns: map[string]*schema.Column{
								"id": {
									Name:     "id",
									DataType: "int(11)",
									Position: 1,
								},
							},
						},
					},
					Indexes: make(map[string]*schema.Index),
				},
				DataDefinitions: []string{
					"DROP TABLE users",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extractor.ValidateExtractedSchema(tt.backup)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExtractedSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetExtractionStats(t *testing.T) {
	extractor := NewSchemaExtractor()

	backup := &Backup{
		SchemaSnapshot: &schema.Schema{
			Name: "testdb",
			Tables: map[string]*schema.Table{
				"users": {
					Name: "users",
					Columns: map[string]*schema.Column{
						"id":   {Name: "id", DataType: "int(11)", Position: 1},
						"name": {Name: "name", DataType: "varchar(255)", Position: 2},
					},
					Indexes: []*schema.Index{
						{Name: "PRIMARY", IsPrimary: true},
						{Name: "idx_name", IsUnique: false},
					},
					Constraints: map[string]*schema.Constraint{
						"fk_user_role": {Name: "fk_user_role", Type: schema.ConstraintTypeForeignKey},
					},
				},
				"orders": {
					Name: "orders",
					Columns: map[string]*schema.Column{
						"id":      {Name: "id", DataType: "int(11)", Position: 1},
						"user_id": {Name: "user_id", DataType: "int(11)", Position: 2},
					},
					Indexes:     []*schema.Index{},
					Constraints: map[string]*schema.Constraint{},
				},
			},
		},
		DataDefinitions: []string{"CREATE TABLE users", "CREATE TABLE orders"},
		Triggers:        []TriggerDefinition{{Name: "user_audit"}},
		Views:           []ViewDefinition{{Name: "user_summary"}},
		Procedures:      []ProcedureDefinition{{Name: "GetUserById"}},
		Functions:       []FunctionDefinition{{Name: "CalculateDiscount"}},
	}

	stats := extractor.GetExtractionStats(backup)

	expectedStats := map[string]int{
		"tables":         2,
		"columns":        4,
		"indexes":        2,
		"constraints":    1,
		"ddl_statements": 2,
		"triggers":       1,
		"views":          1,
		"procedures":     1,
		"functions":      1,
	}

	for key, expected := range expectedStats {
		if actual, exists := stats[key]; !exists || actual != expected {
			t.Errorf("Expected stats[%s] = %d, got %d", key, expected, actual)
		}
	}
}

// Helper function to setup mock queries for complete schema extraction
func setupMockSchemaQueries(mock sqlmock.Sqlmock) {
	// Mock schema extraction (from schema.Extractor)
	mock.ExpectQuery("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME"}).
			AddRow("users"))

	mock.ExpectQuery("SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA, ORDINAL_POSITION, COLUMN_TYPE FROM INFORMATION_SCHEMA.COLUMNS").
		WithArgs("testdb", "users").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "DATA_TYPE", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA", "ORDINAL_POSITION", "COLUMN_TYPE"}).
			AddRow("id", "int", "NO", nil, "auto_increment", 1, "int(11)"))

	mock.ExpectQuery("SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE, INDEX_TYPE, SEQ_IN_INDEX FROM INFORMATION_SCHEMA.STATISTICS").
		WithArgs("testdb", "users").
		WillReturnRows(sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE", "INDEX_TYPE", "SEQ_IN_INDEX"}).
			AddRow("PRIMARY", "id", 0, "BTREE", 1))

	// Mock DDL extraction
	mock.ExpectQuery("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME"}).
			AddRow("users"))

	mock.ExpectQuery("SHOW CREATE TABLE users").
		WillReturnRows(sqlmock.NewRows([]string{"Table", "Create Table"}).
			AddRow("users", "CREATE TABLE `users` (`id` int(11) NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`))"))

	// Mock triggers extraction
	mock.ExpectQuery("SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, ACTION_TIMING, EVENT_MANIPULATION, ACTION_STATEMENT FROM INFORMATION_SCHEMA.TRIGGERS").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TRIGGER_NAME", "EVENT_OBJECT_TABLE", "ACTION_TIMING", "EVENT_MANIPULATION", "ACTION_STATEMENT"}))

	// Mock views extraction
	mock.ExpectQuery("SELECT TABLE_NAME, VIEW_DEFINITION FROM INFORMATION_SCHEMA.VIEWS").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "VIEW_DEFINITION"}))

	// Mock procedures extraction
	mock.ExpectQuery("SELECT ROUTINE_NAME FROM INFORMATION_SCHEMA.ROUTINES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"ROUTINE_NAME"}))

	// Mock functions extraction
	mock.ExpectQuery("SELECT ROUTINE_NAME FROM INFORMATION_SCHEMA.ROUTINES").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"ROUTINE_NAME"}))
}
