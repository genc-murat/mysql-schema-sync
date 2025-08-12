package errors

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestAppError(t *testing.T) {
	cause := errors.New("underlying error")
	appErr := NewAppError(ErrorTypeConnection, "connection failed", cause)

	if appErr.Type != ErrorTypeConnection {
		t.Errorf("Expected type %v, got %v", ErrorTypeConnection, appErr.Type)
	}

	if appErr.Message != "connection failed" {
		t.Errorf("Expected message 'connection failed', got %v", appErr.Message)
	}

	if appErr.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, appErr.Cause)
	}

	if appErr.IsRecoverable() {
		t.Error("Expected non-recoverable error")
	}

	expectedError := "connection: connection failed (caused by: underlying error)"
	if appErr.Error() != expectedError {
		t.Errorf("Expected error string %v, got %v", expectedError, appErr.Error())
	}
}

func TestAppErrorWithContext(t *testing.T) {
	appErr := NewAppError(ErrorTypeSQL, "query failed", nil)
	appErr.WithContext("table", "users").WithContext("query_id", 123)

	if appErr.Context["table"] != "users" {
		t.Errorf("Expected context table=users, got %v", appErr.Context["table"])
	}

	if appErr.Context["query_id"] != 123 {
		t.Errorf("Expected context query_id=123, got %v", appErr.Context["query_id"])
	}
}

func TestNewRecoverableError(t *testing.T) {
	appErr := NewRecoverableError(ErrorTypeConnection, "temporary failure", nil)

	if !appErr.IsRecoverable() {
		t.Error("Expected recoverable error")
	}
}

func TestErrorClassifier_ClassifyMySQLError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		mysqlErr     *mysql.MySQLError
		expectedType ErrorType
		recoverable  bool
	}{
		{
			name:         "access denied",
			mysqlErr:     &mysql.MySQLError{Number: 1045, Message: "Access denied"},
			expectedType: ErrorTypePermission,
			recoverable:  false,
		},
		{
			name:         "unknown database",
			mysqlErr:     &mysql.MySQLError{Number: 1049, Message: "Unknown database"},
			expectedType: ErrorTypeValidation,
			recoverable:  false,
		},
		{
			name:         "table doesn't exist",
			mysqlErr:     &mysql.MySQLError{Number: 1146, Message: "Table doesn't exist"},
			expectedType: ErrorTypeSchema,
			recoverable:  false,
		},
		{
			name:         "can't connect to server",
			mysqlErr:     &mysql.MySQLError{Number: 2003, Message: "Can't connect to MySQL server"},
			expectedType: ErrorTypeConnection,
			recoverable:  true,
		},
		{
			name:         "server has gone away",
			mysqlErr:     &mysql.MySQLError{Number: 2006, Message: "MySQL server has gone away"},
			expectedType: ErrorTypeConnection,
			recoverable:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := classifier.ClassifyError(tt.mysqlErr)

			if appErr.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, appErr.Type)
			}

			if appErr.IsRecoverable() != tt.recoverable {
				t.Errorf("Expected recoverable=%v, got %v", tt.recoverable, appErr.IsRecoverable())
			}

			if appErr.Context["mysql_error_code"] != tt.mysqlErr.Number {
				t.Errorf("Expected mysql_error_code=%v, got %v", tt.mysqlErr.Number, appErr.Context["mysql_error_code"])
			}
		})
	}
}

func TestErrorClassifier_ClassifySQLError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
		recoverable  bool
	}{
		{
			name:         "no rows",
			err:          sql.ErrNoRows,
			expectedType: ErrorTypeValidation,
			recoverable:  false,
		},
		{
			name:         "transaction done",
			err:          sql.ErrTxDone,
			expectedType: ErrorTypeSQL,
			recoverable:  false,
		},
		{
			name:         "connection done",
			err:          sql.ErrConnDone,
			expectedType: ErrorTypeConnection,
			recoverable:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := classifier.ClassifyError(tt.err)

			if appErr.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, appErr.Type)
			}

			if appErr.IsRecoverable() != tt.recoverable {
				t.Errorf("Expected recoverable=%v, got %v", tt.recoverable, appErr.IsRecoverable())
			}
		})
	}
}

func TestErrorClassifier_ClassifyContextError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
		recoverable  bool
	}{
		{
			name:         "deadline exceeded",
			err:          context.DeadlineExceeded,
			expectedType: ErrorTypeTimeout,
			recoverable:  true,
		},
		{
			name:         "context canceled",
			err:          context.Canceled,
			expectedType: ErrorTypeInterruption,
			recoverable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := classifier.ClassifyError(tt.err)

			if appErr.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, appErr.Type)
			}

			if appErr.IsRecoverable() != tt.recoverable {
				t.Errorf("Expected recoverable=%v, got %v", tt.recoverable, appErr.IsRecoverable())
			}
		})
	}
}

func TestErrorClassifier_ClassifyFileSystemError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
	}{
		{
			name:         "file not found",
			err:          &os.PathError{Op: "open", Path: "/nonexistent", Err: syscall.ENOENT},
			expectedType: ErrorTypeValidation,
		},
		{
			name:         "permission denied",
			err:          &os.PathError{Op: "open", Path: "/restricted", Err: syscall.EACCES},
			expectedType: ErrorTypePermission,
		},
		{
			name:         "no space left",
			err:          &os.PathError{Op: "write", Path: "/full", Err: syscall.ENOSPC},
			expectedType: ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := classifier.ClassifyError(tt.err)

			if appErr.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, appErr.Type)
			}
		})
	}
}

func TestErrorClassifier_ClassifyNetworkError(t *testing.T) {
	classifier := NewErrorClassifier()

	// Create a mock network error
	mockNetErr := &mockNetError{timeout: true, temporary: false}
	appErr := classifier.ClassifyError(mockNetErr)

	if appErr.Type != ErrorTypeTimeout {
		t.Errorf("Expected type %v, got %v", ErrorTypeTimeout, appErr.Type)
	}

	if !appErr.IsRecoverable() {
		t.Error("Expected recoverable error for timeout")
	}

	// Test temporary error
	mockNetErr = &mockNetError{timeout: false, temporary: true}
	appErr = classifier.ClassifyError(mockNetErr)

	if appErr.Type != ErrorTypeConnection {
		t.Errorf("Expected type %v, got %v", ErrorTypeConnection, appErr.Type)
	}
}

// Mock network error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock network error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestRetryHandler_Retry(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
	}
	handler := NewRetryHandler(config)

	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		ctx := context.Background()
		err := handler.Retry(ctx, operation)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 3 {
				return NewRecoverableError(ErrorTypeConnection, "temporary failure", nil)
			}
			return nil
		}

		ctx := context.Background()
		err := handler.Retry(ctx, operation)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("non-recoverable error", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return NewAppError(ErrorTypeValidation, "validation failed", nil)
		}

		ctx := context.Background()
		err := handler.Retry(ctx, operation)

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}

		var appErr *AppError
		if !errors.As(err, &appErr) || appErr.Type != ErrorTypeValidation {
			t.Errorf("Expected validation error, got %v", err)
		}
	})

	t.Run("max attempts exceeded", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return NewRecoverableError(ErrorTypeConnection, "always fails", nil)
		}

		ctx := context.Background()
		err := handler.Retry(ctx, operation)

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if attempts != config.MaxAttempts {
			t.Errorf("Expected %d attempts, got %d", config.MaxAttempts, attempts)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return NewRecoverableError(ErrorTypeConnection, "temporary failure", nil)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := handler.Retry(ctx, operation)

		if err == nil {
			t.Error("Expected error, got nil")
		}

		var appErr *AppError
		if !errors.As(err, &appErr) || appErr.Type != ErrorTypeInterruption {
			t.Errorf("Expected interruption error, got %v", err)
		}
	})
}

func TestRetryHandler_CalculateDelay(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
	}
	handler := NewRetryHandler(config)

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{1, 100 * time.Millisecond, 200 * time.Millisecond},
		{2, 200 * time.Millisecond, 400 * time.Millisecond},
		{3, 400 * time.Millisecond, 800 * time.Millisecond},
		{4, 800 * time.Millisecond, 1 * time.Second}, // Should be capped at MaxDelay
		{5, 1 * time.Second, 1 * time.Second},        // Should be capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := handler.calculateDelay(tt.attempt)

			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("Attempt %d: expected delay between %v and %v, got %v",
					tt.attempt, tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestGracefulShutdownHandler(t *testing.T) {
	handler := NewGracefulShutdownHandler()

	shutdownCalled := false
	handler.RegisterShutdownFunc(func() error {
		shutdownCalled = true
		return nil
	})

	// Test that shutdown function is called
	handler.shutdown()

	if !shutdownCalled {
		t.Error("Expected shutdown function to be called")
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "recoverable app error",
			err:  NewRecoverableError(ErrorTypeConnection, "temp failure", nil),
			want: true,
		},
		{
			name: "non-recoverable app error",
			err:  NewAppError(ErrorTypeValidation, "validation failed", nil),
			want: false,
		},
		{
			name: "regular error",
			err:  errors.New("regular error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRecoverableError(tt.err); got != tt.want {
				t.Errorf("IsRecoverableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{
			name: "app error",
			err:  NewAppError(ErrorTypeConnection, "connection failed", nil),
			want: ErrorTypeConnection,
		},
		{
			name: "regular error",
			err:  errors.New("regular error"),
			want: ErrorTypeUnknown,
		},
		{
			name: "nil error",
			err:  nil,
			want: ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorType(tt.err); got != tt.want {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatUserError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "app error with user message",
			err: &AppError{
				Type:        ErrorTypeConnection,
				Message:     "technical message",
				UserMessage: "user-friendly message",
			},
			want: "user-friendly message",
		},
		{
			name: "app error without user message",
			err: &AppError{
				Type:    ErrorTypeConnection,
				Message: "technical message",
			},
			want: "technical message",
		},
		{
			name: "regular error",
			err:  errors.New("regular error"),
			want: "An unexpected error occurred. Please check the logs for more details.",
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatUserError(tt.err); got != tt.want {
				t.Errorf("FormatUserError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, "wrapped message")

	var appErr *AppError
	if !errors.As(wrappedErr, &appErr) {
		t.Error("Expected wrapped error to be AppError")
	}

	if appErr.Message != "wrapped message" {
		t.Errorf("Expected message 'wrapped message', got %v", appErr.Message)
	}

	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected wrapped error to contain original error")
	}
}

func TestCreateContextWithTimeout(t *testing.T) {
	timeout := 100 * time.Millisecond
	ctx, cancel := CreateContextWithTimeout(timeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected context to have deadline")
	}

	if time.Until(deadline) > timeout {
		t.Error("Expected deadline to be within timeout duration")
	}
}

func TestCreateContextWithCancel(t *testing.T) {
	ctx, cancel := CreateContextWithCancel()

	select {
	case <-ctx.Done():
		t.Error("Expected context to not be canceled initially")
	default:
		// Context is not canceled, which is expected
	}

	cancel()

	select {
	case <-ctx.Done():
		// Context is canceled, which is expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be canceled after calling cancel()")
	}
}
