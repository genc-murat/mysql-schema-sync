package logging

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   LogLevel
	}{
		{
			name: "default config",
			config: Config{
				Level:  LogLevelNormal,
				Format: "text",
			},
			want: LogLevelNormal,
		},
		{
			name: "verbose config",
			config: Config{
				Level:  LogLevelVerbose,
				Format: "json",
			},
			want: LogLevelVerbose,
		},
		{
			name: "quiet config",
			config: Config{
				Level:  LogLevelQuiet,
				Format: "text",
			},
			want: LogLevelQuiet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.config.Output = &buf

			logger, err := NewLogger(tt.config)
			if err != nil {
				t.Errorf("NewLogger() error = %v", err)
				return
			}

			if logger.GetLevel() != tt.want {
				t.Errorf("NewLogger() level = %v, want %v", logger.GetLevel(), tt.want)
			}
		})
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Error("NewDefaultLogger() returned nil")
	}

	if logger.GetLevel() != LogLevelNormal {
		t.Errorf("NewDefaultLogger() level = %v, want %v", logger.GetLevel(), LogLevelNormal)
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	fields := map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
	}

	logger.WithFields(fields).Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test_field=test_value") {
		t.Errorf("Expected output to contain test_field=test_value, got: %s", output)
	}
	if !strings.Contains(output, "number=42") {
		t.Errorf("Expected output to contain number=42, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
}

func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	ctx := CreateContextWithRequestID(context.Background(), "test-request-123")
	logger.WithContext(ctx).Info("test message with context")

	output := buf.String()
	if !strings.Contains(output, "request_id=test-request-123") {
		t.Errorf("Expected output to contain request_id=test-request-123, got: %s", output)
	}
}

func TestLogDatabaseConnection(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Test successful connection
	logger.LogDatabaseConnection("localhost", "testdb", true, 100*time.Millisecond, nil)
	output := buf.String()
	if !strings.Contains(output, "Database connection established") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, "host=localhost") {
		t.Errorf("Expected host=localhost, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test failed connection
	testErr := errors.New("connection timeout")
	logger.LogDatabaseConnection("localhost", "testdb", false, 5*time.Second, testErr)
	output = buf.String()
	if !strings.Contains(output, "Database connection failed") {
		t.Errorf("Expected failure message, got: %s", output)
	}
	if !strings.Contains(output, "connection timeout") {
		t.Errorf("Expected error message, got: %s", output)
	}
}

func TestLogSQLExecution(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Test successful SQL execution
	sql := "SELECT * FROM users WHERE id = 1"
	logger.LogSQLExecution(sql, 50*time.Millisecond, 1, nil)
	output := buf.String()
	if !strings.Contains(output, "SQL executed successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, sql) {
		t.Errorf("Expected SQL statement, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test failed SQL execution
	testErr := errors.New("syntax error")
	logger.LogSQLExecution(sql, 10*time.Millisecond, 0, testErr)
	output = buf.String()
	if !strings.Contains(output, "SQL execution failed") {
		t.Errorf("Expected failure message, got: %s", output)
	}
	if !strings.Contains(output, "syntax error") {
		t.Errorf("Expected error message, got: %s", output)
	}
}

func TestLogSQLExecutionTruncation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Create a long SQL statement
	longSQL := strings.Repeat("SELECT * FROM very_long_table_name ", 10) + "WHERE condition = 'value'"
	logger.LogSQLExecution(longSQL, 50*time.Millisecond, 1, nil)

	output := buf.String()
	if !strings.Contains(output, "...") {
		t.Errorf("Expected truncated SQL with '...', got: %s", output)
	}
	if !strings.Contains(output, "sql_length=") {
		t.Errorf("Expected sql_length field, got: %s", output)
	}
}

func TestLogSchemaExtraction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelNormal,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.LogSchemaExtraction("testdb", 5, 200*time.Millisecond, nil)
	output := buf.String()
	if !strings.Contains(output, "Schema extraction completed") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, "table_count=5") {
		t.Errorf("Expected table_count=5, got: %s", output)
	}
}

func TestLogSchemaComparison(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelNormal,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.LogSchemaComparison("sourcedb", "targetdb", 3, 100*time.Millisecond)
	output := buf.String()
	if !strings.Contains(output, "Schema differences detected") {
		t.Errorf("Expected differences message, got: %s", output)
	}
	if !strings.Contains(output, "changes_found=3") {
		t.Errorf("Expected changes_found=3, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test no changes
	logger.LogSchemaComparison("sourcedb", "targetdb", 0, 100*time.Millisecond)
	output = buf.String()
	if !strings.Contains(output, "No schema differences found") {
		t.Errorf("Expected no differences message, got: %s", output)
	}
}

func TestLogMigrationExecution(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelNormal,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.LogMigrationExecution(5, 500*time.Millisecond, true, nil)
	output := buf.String()
	if !strings.Contains(output, "Migration executed successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, "statements_count=5") {
		t.Errorf("Expected statements_count=5, got: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	logger := NewDefaultLogger()

	logger.SetLevel(LogLevelVerbose)
	if logger.GetLevel() != LogLevelVerbose {
		t.Errorf("SetLevel() failed, got %v, want %v", logger.GetLevel(), LogLevelVerbose)
	}

	logger.SetLevel(LogLevelQuiet)
	if logger.GetLevel() != LogLevelQuiet {
		t.Errorf("SetLevel() failed, got %v, want %v", logger.GetLevel(), LogLevelQuiet)
	}
}

func TestIsLevelEnabled(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel LogLevel
		testLevel   LogLevel
		want        bool
	}{
		{"quiet logger, error level", LogLevelQuiet, LogLevelQuiet, true},
		{"quiet logger, normal level", LogLevelQuiet, LogLevelNormal, false},
		{"normal logger, normal level", LogLevelNormal, LogLevelNormal, true},
		{"normal logger, verbose level", LogLevelNormal, LogLevelVerbose, false},
		{"verbose logger, verbose level", LogLevelVerbose, LogLevelVerbose, true},
		{"verbose logger, debug level", LogLevelVerbose, LogLevelDebug, false},
		{"debug logger, debug level", LogLevelDebug, LogLevelDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  tt.loggerLevel,
				Output: &buf,
				Format: "text",
			}

			logger, err := NewLogger(config)
			if err != nil {
				t.Fatalf("NewLogger() error = %v", err)
			}

			if got := logger.IsLevelEnabled(tt.testLevel); got != tt.want {
				t.Errorf("IsLevelEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogOperationStart(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LogLevelVerbose,
		Output: &buf,
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	fields := map[string]interface{}{
		"table": "users",
		"count": 100,
	}

	finishFunc := logger.LogOperationStart("test_operation", fields)

	// Check start message
	output := buf.String()
	if !strings.Contains(output, "Operation started") {
		t.Errorf("Expected start message, got: %s", output)
	}
	if !strings.Contains(output, "table=users") {
		t.Errorf("Expected table=users, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test successful completion
	finishFunc(nil)
	output = buf.String()
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("Expected completion message, got: %s", output)
	}
	if !strings.Contains(output, "success=true") {
		t.Errorf("Expected success=true, got: %s", output)
	}

	// Reset buffer
	buf.Reset()

	// Test failed completion
	finishFunc2 := logger.LogOperationStart("test_operation_2", fields)
	buf.Reset() // Clear start message

	testErr := errors.New("operation failed")
	finishFunc2(testErr)
	output = buf.String()
	if !strings.Contains(output, "Operation failed") {
		t.Errorf("Expected failure message, got: %s", output)
	}
	if !strings.Contains(output, "success=false") {
		t.Errorf("Expected success=false, got: %s", output)
	}
	if !strings.Contains(output, "operation failed") {
		t.Errorf("Expected error message, got: %s", output)
	}
}

func TestCreateContextWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-123"

	newCtx := CreateContextWithRequestID(ctx, requestID)

	retrievedID := GetRequestIDFromContext(newCtx)
	if retrievedID != requestID {
		t.Errorf("GetRequestIDFromContext() = %v, want %v", retrievedID, requestID)
	}
}

func TestGetRequestIDFromContext(t *testing.T) {
	// Test with no request ID
	ctx := context.Background()
	id := GetRequestIDFromContext(ctx)
	if id != "" {
		t.Errorf("GetRequestIDFromContext() = %v, want empty string", id)
	}

	// Test with request ID
	requestID := "test-456"
	ctx = CreateContextWithRequestID(ctx, requestID)
	id = GetRequestIDFromContext(ctx)
	if id != requestID {
		t.Errorf("GetRequestIDFromContext() = %v, want %v", id, requestID)
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal SQL",
			input: "SELECT * FROM users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "SQL with password",
			input: "CREATE USER 'test'@'localhost' IDENTIFIED BY password='secret123'",
			want:  "CREATE USER 'test'@'localhost' IDENTIFIED BY password=***",
		},
		{
			name:  "SQL with uppercase PASSWORD",
			input: "ALTER USER 'test'@'localhost' IDENTIFIED BY PASSWORD='secret123'",
			want:  "ALTER USER 'test'@'localhost' IDENTIFIED BY PASSWORD=***",
		},
		{
			name:  "very long SQL",
			input: strings.Repeat("SELECT * FROM very_long_table_name ", 20),
			want:  strings.Repeat("SELECT * FROM very_long_table_name ", 20)[:500] + "... [truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeSQL(tt.input); got != tt.want {
				t.Errorf("SanitizeSQL() = %v, want %v", got, tt.want)
			}
		})
	}
}
