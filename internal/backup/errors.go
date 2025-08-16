package backup

import (
	"fmt"
)

// BackupError represents errors that occur during backup operations
type BackupError struct {
	Type    BackupErrorType        `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"-"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *BackupError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause error
func (e *BackupError) Unwrap() error {
	return e.Cause
}

// BackupErrorType represents different types of backup errors
type BackupErrorType string

const (
	BackupErrorTypeStorage       BackupErrorType = "STORAGE_ERROR"
	BackupErrorTypeValidation    BackupErrorType = "VALIDATION_ERROR"
	BackupErrorTypeCompression   BackupErrorType = "COMPRESSION_ERROR"
	BackupErrorTypeEncryption    BackupErrorType = "ENCRYPTION_ERROR"
	BackupErrorTypeCorruption    BackupErrorType = "CORRUPTION_ERROR"
	BackupErrorTypePermission    BackupErrorType = "PERMISSION_ERROR"
	BackupErrorTypeNetwork       BackupErrorType = "NETWORK_ERROR"
	BackupErrorTypeDatabase      BackupErrorType = "DATABASE_ERROR"
	BackupErrorTypeConfiguration BackupErrorType = "CONFIGURATION_ERROR"
	BackupErrorTypeNotFound      BackupErrorType = "NOT_FOUND_ERROR"
	BackupErrorTypeConflict      BackupErrorType = "CONFLICT_ERROR"
	BackupErrorTypeRollback      BackupErrorType = "ROLLBACK_ERROR"
)

// NewBackupError creates a new BackupError
func NewBackupError(errorType BackupErrorType, message string, cause error) *BackupError {
	return &BackupError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *BackupError) WithContext(key string, value interface{}) *BackupError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Common error constructors
func NewStorageError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeStorage, message, cause)
}

func NewValidationError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeValidation, message, cause)
}

func NewCompressionError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeCompression, message, cause)
}

func NewEncryptionError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeEncryption, message, cause)
}

func NewCorruptionError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeCorruption, message, cause)
}

func NewPermissionError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypePermission, message, cause)
}

func NewNetworkError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeNetwork, message, cause)
}

func NewDatabaseError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeDatabase, message, cause)
}

func NewConfigurationError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeConfiguration, message, cause)
}

func NewNotFoundError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeNotFound, message, cause)
}

func NewConflictError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeConflict, message, cause)
}

func NewRollbackError(message string, cause error) *BackupError {
	return NewBackupError(BackupErrorTypeRollback, message, cause)
}

// ValidationError represents validation-specific errors
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s (and %d more)", len(e), e[0].Error(), len(e)-1)
}

// Add adds a validation error to the collection
func (e *ValidationErrors) Add(field, message string, value interface{}) {
	*e = append(*e, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	if backupErr, ok := err.(*BackupError); ok {
		switch backupErr.Type {
		case BackupErrorTypeNetwork, BackupErrorTypeStorage:
			return true
		default:
			return false
		}
	}
	return false
}

// IsPermanent determines if an error is permanent and should not be retried
func IsPermanent(err error) bool {
	if backupErr, ok := err.(*BackupError); ok {
		switch backupErr.Type {
		case BackupErrorTypeValidation, BackupErrorTypeCorruption,
			BackupErrorTypePermission, BackupErrorTypeConfiguration:
			return true
		default:
			return false
		}
	}
	return false
}
