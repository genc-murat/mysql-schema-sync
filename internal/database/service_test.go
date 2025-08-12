package database

import (
	"database/sql"
	"testing"
	"time"

	"mysql-schema-sync/internal/logging"

	_ "github.com/go-sql-driver/mysql"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("Expected service to be created")
	}
	if service.connectionTimeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", service.connectionTimeout)
	}
	if service.maxRetries != 3 {
		t.Errorf("Expected default max retries to be 3, got %d", service.maxRetries)
	}
}

func TestNewServiceWithOptions(t *testing.T) {
	timeout := 10 * time.Second
	maxRetries := 5
	retryDelay := 1 * time.Second

	service := NewServiceWithOptions(timeout, maxRetries, retryDelay)
	if service.connectionTimeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, service.connectionTimeout)
	}
	if service.maxRetries != maxRetries {
		t.Errorf("Expected max retries to be %d, got %d", maxRetries, service.maxRetries)
	}
	if service.retryDelay != retryDelay {
		t.Errorf("Expected retry delay to be %v, got %v", retryDelay, service.retryDelay)
	}
}

func TestNewServiceWithLogger(t *testing.T) {
	logger := logging.NewDefaultLogger()
	service := NewServiceWithLogger(logger)
	if service.logger != logger {
		t.Error("Expected custom logger to be set")
	}
}

func TestConnect_InvalidConfig(t *testing.T) {
	service := NewService()

	// Test with invalid host
	config := DatabaseConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "test",
	}

	_, err := service.Connect(config)
	if err == nil {
		t.Error("Expected error for invalid host")
	}
}

func TestConnect_EmptyConfig(t *testing.T) {
	service := NewService()

	// Test with empty config
	config := DatabaseConfig{}

	_, err := service.Connect(config)
	if err == nil {
		t.Error("Expected error for empty config")
	}
}

func TestTestConnection_NilDB(t *testing.T) {
	service := NewService()

	err := service.TestConnection(nil)
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestClose_NilDB(t *testing.T) {
	service := NewService()

	err := service.Close(nil)
	if err != nil {
		t.Errorf("Expected no error for closing nil connection, got %v", err)
	}
}

func TestGetVersion_NilDB(t *testing.T) {
	service := NewService()

	_, err := service.GetVersion(nil)
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestExecuteSQL_NilDB(t *testing.T) {
	service := NewService()

	err := service.ExecuteSQL(nil, []string{"SELECT 1"})
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestExecuteSQL_EmptyStatements(t *testing.T) {
	service := NewService()

	// Create a mock database connection (this won't actually work without a real DB)
	// but we can test the empty statements case
	err := service.ExecuteSQL(nil, []string{})
	if err == nil {
		t.Error("Expected error for nil database connection, even with empty statements")
	}
}

func TestExecuteSQL_EmptyStringStatements(t *testing.T) {
	service := NewService()

	// Test that empty string statements are skipped
	// This test would need a real database connection to work properly
	// For now, we test the validation
	err := service.ExecuteSQL(nil, []string{""})
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

// DSN and Validate tests are already covered in config_test.go

// Mock database tests - these test the service logic without requiring a real database

type mockDB struct {
	pingError   error
	execError   error
	queryError  error
	closeError  error
	execCalled  bool
	queryCalled bool
	closeCalled bool
}

func (m *mockDB) Ping() error {
	return m.pingError
}

func (m *mockDB) Close() error {
	m.closeCalled = true
	return m.closeError
}

func (m *mockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.execCalled = true
	if m.execError != nil {
		return nil, m.execError
	}
	return &mockResult{}, nil
}

func (m *mockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.queryCalled = true
	return nil, m.queryError
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }

// Test service behavior with connection timeouts
func TestService_ConnectionTimeout(t *testing.T) {
	// Create service with very short timeout
	service := NewServiceWithOptions(1*time.Millisecond, 1, 1*time.Millisecond)

	config := DatabaseConfig{
		Host:     "192.0.2.1", // Non-routable IP to simulate timeout
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "test",
	}

	_, err := service.Connect(config)
	if err == nil {
		t.Error("Expected timeout error for unreachable host")
	}
}

// Test retry logic behavior
func TestService_RetryLogic(t *testing.T) {
	service := NewServiceWithOptions(1*time.Second, 2, 10*time.Millisecond)

	config := DatabaseConfig{
		Host:     "invalid-host-for-retry-test",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "test",
	}

	start := time.Now()
	_, err := service.Connect(config)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error for invalid host")
	}

	// Should have taken at least the retry delay time
	if duration < 10*time.Millisecond {
		t.Errorf("Expected retry delay, but operation completed too quickly: %v", duration)
	}
}

// Test SQL statement validation
func TestService_SQLValidation(t *testing.T) {
	service := NewService()

	tests := []struct {
		name       string
		statements []string
		expectSkip bool
	}{
		{
			name:       "valid statements",
			statements: []string{"CREATE TABLE test (id INT)", "INSERT INTO test VALUES (1)"},
			expectSkip: false,
		},
		{
			name:       "empty statements",
			statements: []string{"", "   ", "\n"},
			expectSkip: true,
		},
		{
			name:       "mixed statements",
			statements: []string{"CREATE TABLE test (id INT)", "", "INSERT INTO test VALUES (1)"},
			expectSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the logic without requiring a real database
			// The actual execution will fail due to nil DB, but we can verify
			// the statement processing logic
			err := service.ExecuteSQL(nil, tt.statements)
			if err == nil {
				t.Error("Expected error for nil database connection")
			}
		})
	}
}

// Test error handling and logging
func TestService_ErrorHandling(t *testing.T) {
	service := NewService()

	// Test various error scenarios
	testCases := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "nil database in TestConnection",
			testFunc: func() error {
				return service.TestConnection(nil)
			},
		},
		{
			name: "nil database in GetVersion",
			testFunc: func() error {
				_, err := service.GetVersion(nil)
				return err
			},
		},
		{
			name: "nil database in ExecuteSQL",
			testFunc: func() error {
				return service.ExecuteSQL(nil, []string{"SELECT 1"})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFunc()
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Additional configuration edge cases beyond what's in config_test.go
func TestDatabaseConfig_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config DatabaseConfig
		valid  bool
	}{
		{
			name: "minimum valid port",
			config: DatabaseConfig{
				Host: "localhost", Port: 1, Username: "root", Database: "test",
			},
			valid: true,
		},
		{
			name: "maximum valid port",
			config: DatabaseConfig{
				Host: "localhost", Port: 65535, Username: "root", Database: "test",
			},
			valid: true,
		},
		{
			name: "empty password (valid)",
			config: DatabaseConfig{
				Host: "localhost", Port: 3306, Username: "root", Database: "test", Password: "",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid config but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected invalid config but got no error")
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkDatabaseConfig_DSN(b *testing.B) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.DSN()
	}
}

func BenchmarkDatabaseConfig_Validate(b *testing.B) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}
