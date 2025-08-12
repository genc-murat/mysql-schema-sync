package execution

import (
	"errors"
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	appErrors "mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/logging"
)

func TestNewExecutor(t *testing.T) {
	config := ExecutionConfig{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		DryRun:      true,
		AutoApprove: false,
		Timeout:     30 * time.Second,
		LogLevel:    logging.LogLevelNormal,
	}

	executor, err := NewExecutor(config)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	if executor == nil {
		t.Error("NewExecutor() returned nil executor")
	}

	if executor.config.DryRun != config.DryRun {
		t.Errorf("Expected DryRun=%v, got %v", config.DryRun, executor.config.DryRun)
	}

	if executor.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if executor.dbService == nil {
		t.Error("Expected dbService to be initialized")
	}

	if executor.schemaService == nil {
		t.Error("Expected schemaService to be initialized")
	}

	if executor.migrationService == nil {
		t.Error("Expected migrationService to be initialized")
	}

	if executor.retryHandler == nil {
		t.Error("Expected retryHandler to be initialized")
	}

	if executor.shutdownHandler == nil {
		t.Error("Expected shutdownHandler to be initialized")
	}
}

func TestExecutor_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ExecutionConfig
		wantErr bool
		errType appErrors.ErrorType
	}{
		{
			name: "valid config",
			config: ExecutionConfig{
				SourceDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "source_db",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "target_db",
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: false,
		},
		{
			name: "missing source host",
			config: ExecutionConfig{
				SourceDB: database.DatabaseConfig{
					Database: "source_db",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "target_db",
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: true,
			errType: appErrors.ErrorTypeValidation,
		},
		{
			name: "missing source database",
			config: ExecutionConfig{
				SourceDB: database.DatabaseConfig{
					Host:     "localhost",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "target_db",
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: true,
			errType: appErrors.ErrorTypeValidation,
		},
		{
			name: "missing target host",
			config: ExecutionConfig{
				SourceDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "source_db",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Database: "target_db",
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: true,
			errType: appErrors.ErrorTypeValidation,
		},
		{
			name: "missing target database",
			config: ExecutionConfig{
				SourceDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "source_db",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Host:     "localhost",
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: true,
			errType: appErrors.ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(tt.config)
			if err != nil {
				t.Fatalf("NewExecutor() error = %v", err)
			}

			err = executor.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				var appErr *appErrors.AppError
				if !errors.As(err, &appErr) {
					t.Errorf("Expected AppError, got %T", err)
					return
				}

				if appErr.Type != tt.errType {
					t.Errorf("Expected error type %v, got %v", tt.errType, appErr.Type)
				}
			}
		})
	}
}

func TestExecutor_HandleError(t *testing.T) {
	config := ExecutionConfig{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		LogLevel: logging.LogLevelVerbose,
	}

	executor, err := NewExecutor(config)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "app error",
			err:     appErrors.NewAppError(appErrors.ErrorTypeConnection, "connection failed", nil),
			wantErr: true,
		},
		{
			name:    "recoverable error",
			err:     appErrors.NewRecoverableError(appErrors.ErrorTypeTimeout, "timeout occurred", nil),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.HandleError(tt.err)
			if (result != nil) != tt.wantErr {
				t.Errorf("HandleError() error = %v, wantErr %v", result, tt.wantErr)
			}

			if tt.wantErr && result != nil {
				var appErr *appErrors.AppError
				if !errors.As(result, &appErr) {
					t.Errorf("Expected AppError, got %T", result)
				}
			}
		})
	}
}

func TestExecutor_GetLogger(t *testing.T) {
	config := ExecutionConfig{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		LogLevel: logging.LogLevelNormal,
	}

	executor, err := NewExecutor(config)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	logger := executor.GetLogger()
	if logger == nil {
		t.Error("GetLogger() returned nil")
	}

	if logger.GetLevel() != logging.LogLevelNormal {
		t.Errorf("Expected log level %v, got %v", logging.LogLevelNormal, logger.GetLevel())
	}
}

func TestExecutor_GetShutdownHandler(t *testing.T) {
	config := ExecutionConfig{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		LogLevel: logging.LogLevelNormal,
	}

	executor, err := NewExecutor(config)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	shutdownHandler := executor.GetShutdownHandler()
	if shutdownHandler == nil {
		t.Error("GetShutdownHandler() returned nil")
	}
}

func TestExecutionResult(t *testing.T) {
	result := &ExecutionResult{
		Success:            true,
		ExecutedStatements: []string{"CREATE TABLE test (id INT)"},
		Warnings:           []string{"Warning: potential data loss"},
		Duration:           100 * time.Millisecond,
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if len(result.ExecutedStatements) != 1 {
		t.Errorf("Expected 1 executed statement, got %d", len(result.ExecutedStatements))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", result.Duration)
	}
}

// Integration test that would require actual database connections
// This is commented out as it requires real MySQL instances
/*
func TestExecutor_Execute_Integration(t *testing.T) {
	// This test would require actual MySQL databases
	// Skip in unit tests
	t.Skip("Integration test requires actual MySQL databases")

	config := ExecutionConfig{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "test_source",
			Username: "root",
			Password: "password",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "test_target",
			Username: "root",
			Password: "password",
		},
		DryRun:      true,
		AutoApprove: true,
		Timeout:     30 * time.Second,
		LogLevel:    logging.LogLevelVerbose,
	}

	executor, err := NewExecutor(config)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Error("Execute() returned nil result")
	}

	if !result.Success {
		t.Errorf("Expected successful execution, got error: %v", result.Error)
	}
}
*/
