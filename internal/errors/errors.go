package errors

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// ErrorTypeConnection represents database connection errors
	ErrorTypeConnection ErrorType = "connection"
	// ErrorTypeSQL represents SQL execution errors
	ErrorTypeSQL ErrorType = "sql"
	// ErrorTypeSchema represents schema-related errors
	ErrorTypeSchema ErrorType = "schema"
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypePermission represents permission/access errors
	ErrorTypePermission ErrorType = "permission"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeInterruption represents user interruption
	ErrorTypeInterruption ErrorType = "interruption"
	// ErrorTypeUnknown represents unknown errors
	ErrorTypeUnknown ErrorType = "unknown"
)

// AppError represents an application-specific error with context
type AppError struct {
	Type        ErrorType
	Message     string
	Cause       error
	Context     map[string]interface{}
	Recoverable bool
	UserMessage string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// GetUserMessage returns a user-friendly error message
func (e *AppError) GetUserMessage() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}
	return e.Message
}

// IsRecoverable returns whether the error is recoverable
func (e *AppError) IsRecoverable() bool {
	return e.Recoverable
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:        errorType,
		Message:     message,
		Cause:       cause,
		Context:     make(map[string]interface{}),
		Recoverable: false,
	}
}

// NewRecoverableError creates a new recoverable error
func NewRecoverableError(errorType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:        errorType,
		Message:     message,
		Cause:       cause,
		Context:     make(map[string]interface{}),
		Recoverable: true,
	}
}

// ErrorClassifier provides methods to classify and handle different types of errors
type ErrorClassifier struct{}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// ClassifyError analyzes an error and returns an AppError with appropriate classification
func (ec *ErrorClassifier) ClassifyError(err error) *AppError {
	if err == nil {
		return nil
	}

	// Check if it's already an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Classify MySQL errors
	if mysqlErr := ec.classifyMySQLError(err); mysqlErr != nil {
		return mysqlErr
	}

	// Classify network errors
	if netErr := ec.classifyNetworkError(err); netErr != nil {
		return netErr
	}

	// Classify context errors
	if ctxErr := ec.classifyContextError(err); ctxErr != nil {
		return ctxErr
	}

	// Classify file system errors
	if fsErr := ec.classifyFileSystemError(err); fsErr != nil {
		return fsErr
	}

	// Default to unknown error
	return NewAppError(ErrorTypeUnknown, "An unexpected error occurred", err)
}

// classifyMySQLError classifies MySQL-specific errors
func (ec *ErrorClassifier) classifyMySQLError(err error) *AppError {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1045: // Access denied
			return NewAppError(ErrorTypePermission,
				"Database access denied - check username and password", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 1049: // Unknown database
			return NewAppError(ErrorTypeValidation,
				"Database does not exist", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 1146: // Table doesn't exist
			return NewAppError(ErrorTypeSchema,
				"Table does not exist", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 1054: // Unknown column
			return NewAppError(ErrorTypeSchema,
				"Column does not exist", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 1062: // Duplicate entry
			return NewAppError(ErrorTypeValidation,
				"Duplicate entry - record already exists", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 1064: // SQL syntax error
			return NewAppError(ErrorTypeSQL,
				"SQL syntax error", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 2003: // Can't connect to MySQL server
			return NewRecoverableError(ErrorTypeConnection,
				"Cannot connect to MySQL server - server may be down or unreachable", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		case 2006: // MySQL server has gone away
			return NewRecoverableError(ErrorTypeConnection,
				"MySQL server connection lost - attempting to reconnect", err).
				WithContext("mysql_error_code", mysqlErr.Number)
		default:
			return NewAppError(ErrorTypeSQL,
				fmt.Sprintf("MySQL error: %s", mysqlErr.Message), err).
				WithContext("mysql_error_code", mysqlErr.Number)
		}
	}

	// Check for SQL driver errors
	if errors.Is(err, sql.ErrNoRows) {
		return NewAppError(ErrorTypeValidation, "No rows found", err)
	}
	if errors.Is(err, sql.ErrTxDone) {
		return NewAppError(ErrorTypeSQL, "Transaction has already been committed or rolled back", err)
	}
	if errors.Is(err, sql.ErrConnDone) {
		return NewRecoverableError(ErrorTypeConnection, "Database connection is closed", err)
	}

	return nil
}

// classifyNetworkError classifies network-related errors
func (ec *ErrorClassifier) classifyNetworkError(err error) *AppError {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NewRecoverableError(ErrorTypeTimeout,
				"Network operation timed out", err)
		}
		if netErr.Temporary() {
			return NewRecoverableError(ErrorTypeConnection,
				"Temporary network error", err)
		}
	}

	// Check for specific network error types
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		switch opErr.Op {
		case "dial":
			return NewRecoverableError(ErrorTypeConnection,
				"Failed to establish network connection", err)
		case "read", "write":
			return NewRecoverableError(ErrorTypeConnection,
				"Network I/O error", err)
		}
	}

	return nil
}

// classifyContextError classifies context-related errors
func (ec *ErrorClassifier) classifyContextError(err error) *AppError {
	if errors.Is(err, context.DeadlineExceeded) {
		return NewRecoverableError(ErrorTypeTimeout,
			"Operation timed out", err)
	}
	if errors.Is(err, context.Canceled) {
		return NewAppError(ErrorTypeInterruption,
			"Operation was canceled", err)
	}

	return nil
}

// classifyFileSystemError classifies file system errors
func (ec *ErrorClassifier) classifyFileSystemError(err error) *AppError {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		switch pathErr.Err {
		case syscall.ENOENT:
			return NewAppError(ErrorTypeValidation,
				fmt.Sprintf("File or directory not found: %s", pathErr.Path), err)
		case syscall.EACCES:
			return NewAppError(ErrorTypePermission,
				fmt.Sprintf("Permission denied: %s", pathErr.Path), err)
		case syscall.ENOSPC:
			return NewAppError(ErrorTypeValidation,
				"No space left on device", err)
		}
	}

	return nil
}

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// RetryHandler provides retry functionality for operations
type RetryHandler struct {
	config     RetryConfig
	classifier *ErrorClassifier
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(config RetryConfig) *RetryHandler {
	return &RetryHandler{
		config:     config,
		classifier: NewErrorClassifier(),
	}
}

// NewDefaultRetryHandler creates a retry handler with default configuration
func NewDefaultRetryHandler() *RetryHandler {
	return NewRetryHandler(DefaultRetryConfig())
}

// Retry executes a function with retry logic for recoverable errors
func (rh *RetryHandler) Retry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= rh.config.MaxAttempts; attempt++ {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			return NewAppError(ErrorTypeInterruption, "Operation canceled", ctx.Err())
		default:
		}

		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err
		appErr := rh.classifier.ClassifyError(err)

		// If error is not recoverable, don't retry
		if !appErr.IsRecoverable() {
			return appErr
		}

		// Don't retry on the last attempt
		if attempt == rh.config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := rh.calculateDelay(attempt)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return NewAppError(ErrorTypeInterruption, "Operation canceled during retry", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	return rh.classifier.ClassifyError(lastErr).
		WithContext("attempts", rh.config.MaxAttempts)
}

// calculateDelay calculates the delay for a given attempt using exponential backoff
func (rh *RetryHandler) calculateDelay(attempt int) time.Duration {
	// For attempt 1, use base delay
	// For attempt 2, use base delay * multiplier
	// For attempt 3, use base delay * multiplier^2, etc.
	multiplier := 1.0
	for i := 1; i < attempt; i++ {
		multiplier *= rh.config.Multiplier
	}

	delay := time.Duration(float64(rh.config.BaseDelay) * multiplier)

	if delay > rh.config.MaxDelay {
		delay = rh.config.MaxDelay
	}

	return delay
}

// GracefulShutdownHandler handles graceful shutdown on interruption signals
type GracefulShutdownHandler struct {
	shutdownFuncs []func() error
	signalChan    chan os.Signal
	done          chan bool
}

// NewGracefulShutdownHandler creates a new graceful shutdown handler
func NewGracefulShutdownHandler() *GracefulShutdownHandler {
	return &GracefulShutdownHandler{
		shutdownFuncs: make([]func() error, 0),
		signalChan:    make(chan os.Signal, 1),
		done:          make(chan bool, 1),
	}
}

// RegisterShutdownFunc registers a function to be called during shutdown
func (gsh *GracefulShutdownHandler) RegisterShutdownFunc(fn func() error) {
	gsh.shutdownFuncs = append(gsh.shutdownFuncs, fn)
}

// Start starts listening for shutdown signals
func (gsh *GracefulShutdownHandler) Start() {
	signal.Notify(gsh.signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-gsh.signalChan
		gsh.shutdown()
	}()
}

// Stop stops the graceful shutdown handler
func (gsh *GracefulShutdownHandler) Stop() {
	signal.Stop(gsh.signalChan)
	close(gsh.signalChan)
}

// WaitForShutdown waits for shutdown to complete
func (gsh *GracefulShutdownHandler) WaitForShutdown() {
	<-gsh.done
}

// shutdown executes all registered shutdown functions
func (gsh *GracefulShutdownHandler) shutdown() {
	defer func() {
		gsh.done <- true
	}()

	for i := len(gsh.shutdownFuncs) - 1; i >= 0; i-- {
		if err := gsh.shutdownFuncs[i](); err != nil {
			// Log error but continue with shutdown
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		}
	}
}

// CreateContextWithTimeout creates a context with timeout and cancellation support
func CreateContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateContextWithCancel creates a cancelable context
func CreateContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// IsRecoverableError checks if an error is recoverable
func IsRecoverableError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.IsRecoverable()
	}
	return false
}

// GetErrorType returns the error type of an error
func GetErrorType(err error) ErrorType {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type
	}
	return ErrorTypeUnknown
}

// FormatUserError formats an error for display to users
func FormatUserError(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.GetUserMessage()
	}

	// For non-AppError types, provide generic message
	return "An unexpected error occurred. Please check the logs for more details."
}

// WrapError wraps an existing error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return NewAppError(appErr.Type, message, err)
	}

	classifier := NewErrorClassifier()
	classifiedErr := classifier.ClassifyError(err)
	classifiedErr.Message = message
	return classifiedErr
}
