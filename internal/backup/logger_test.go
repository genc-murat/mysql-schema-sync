package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/migration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackupLogger(t *testing.T) {
	tests := []struct {
		name           string
		config         BackupLoggerConfig
		expectError    bool
		expectAuditLog bool
	}{
		{
			name: "basic logger without audit",
			config: BackupLoggerConfig{
				Logger:         logging.NewDefaultLogger(),
				EnableAuditLog: false,
			},
			expectError:    false,
			expectAuditLog: false,
		},
		{
			name: "logger with audit log",
			config: BackupLoggerConfig{
				Logger:         logging.NewDefaultLogger(),
				AuditLogFile:   filepath.Join(t.TempDir(), "audit.log"),
				EnableAuditLog: true,
			},
			expectError:    false,
			expectAuditLog: true,
		},
		{
			name: "logger with custom correlation ID",
			config: BackupLoggerConfig{
				Logger:        logging.NewDefaultLogger(),
				CorrelationID: "test-correlation-123",
			},
			expectError:    false,
			expectAuditLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bl, err := NewBackupLogger(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, bl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bl)
				assert.NotEmpty(t, bl.GetCorrelationID())

				if tt.config.CorrelationID != "" {
					assert.Equal(t, tt.config.CorrelationID, bl.GetCorrelationID())
				}

				if tt.expectAuditLog {
					assert.NotNil(t, bl.auditLogger)
				} else {
					assert.Nil(t, bl.auditLogger)
				}
			}
		})
	}
}

func TestBackupLogger_LogBackupStart(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	backupID := "backup-123"
	databaseName := "test_db"
	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Host:     "localhost",
			Database: databaseName,
		},
		StorageConfig: StorageConfig{
			Provider: StorageProviderLocal,
		},
		CompressionType: CompressionTypeGzip,
		Description:     "Test backup",
		Tags: map[string]string{
			"env": "test",
		},
	}

	// Test successful backup
	t.Run("successful backup", func(t *testing.T) {
		logCompletion := bl.LogBackupStart(ctx, backupID, databaseName, config)

		metadata := &BackupMetadata{
			ID:             backupID,
			Size:           1024,
			CompressedSize: 512,
			Checksum:       "abc123",
		}

		logCompletion(nil, metadata)
	})

	// Test failed backup
	t.Run("failed backup", func(t *testing.T) {
		logCompletion := bl.LogBackupStart(ctx, backupID, databaseName, config)
		logCompletion(fmt.Errorf("backup failed"), nil)
	})
}

func TestBackupLogger_LogBackupValidation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	backupID := "backup-123"
	validationType := "integrity"

	// Test successful validation
	t.Run("successful validation", func(t *testing.T) {
		logCompletion := bl.LogBackupValidation(ctx, backupID, validationType)

		result := &ValidationResult{
			Valid:         true,
			ChecksumValid: true,
			CheckedAt:     time.Now(),
		}

		logCompletion(nil, result)
	})

	// Test failed validation
	t.Run("failed validation", func(t *testing.T) {
		logCompletion := bl.LogBackupValidation(ctx, backupID, validationType)

		result := &ValidationResult{
			Valid:         false,
			ChecksumValid: false,
			Errors:        []string{"checksum mismatch"},
			Warnings:      []string{"minor issue"},
			CheckedAt:     time.Now(),
		}

		logCompletion(nil, result)
	})

	// Test validation error
	t.Run("validation error", func(t *testing.T) {
		logCompletion := bl.LogBackupValidation(ctx, backupID, validationType)
		logCompletion(fmt.Errorf("validation failed"), nil)
	})
}

func TestBackupLogger_LogBackupDeletion(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	backupID := "backup-123"
	reason := "retention policy"

	// Test successful deletion
	t.Run("successful deletion", func(t *testing.T) {
		logCompletion := bl.LogBackupDeletion(ctx, backupID, reason)
		logCompletion(nil)
	})

	// Test failed deletion
	t.Run("failed deletion", func(t *testing.T) {
		logCompletion := bl.LogBackupDeletion(ctx, backupID, reason)
		logCompletion(fmt.Errorf("deletion failed"))
	})
}

func TestBackupLogger_LogRollbackStart(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	backupID := "backup-123"
	databaseName := "test_db"
	plan := &RollbackPlan{
		BackupID:     backupID,
		Statements:   make([]migration.MigrationStatement, 5), // Mock 5 statements
		Dependencies: []string{"table1", "table2"},
		Warnings:     []string{"warning1"},
	}

	// Test successful rollback
	t.Run("successful rollback", func(t *testing.T) {
		logCompletion := bl.LogRollbackStart(ctx, backupID, databaseName, plan)
		logCompletion(nil)
	})

	// Test failed rollback
	t.Run("failed rollback", func(t *testing.T) {
		logCompletion := bl.LogRollbackStart(ctx, backupID, databaseName, plan)
		logCompletion(fmt.Errorf("rollback failed"))
	})
}

func TestBackupLogger_LogRetentionCleanup(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	databaseName := "test_db"
	policy := map[string]interface{}{
		"max_backups": 10,
		"max_age":     "30d",
	}

	// Test successful cleanup
	t.Run("successful cleanup", func(t *testing.T) {
		logCompletion := bl.LogRetentionCleanup(ctx, databaseName, policy)
		deletedBackups := []string{"backup-1", "backup-2"}
		logCompletion(nil, deletedBackups)
	})

	// Test failed cleanup
	t.Run("failed cleanup", func(t *testing.T) {
		logCompletion := bl.LogRetentionCleanup(ctx, databaseName, policy)
		logCompletion(fmt.Errorf("cleanup failed"), nil)
	})
}

func TestBackupLogger_LogStorageOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	operation := "upload"
	provider := "s3"
	backupID := "backup-123"

	// Test successful storage operation
	t.Run("successful storage operation", func(t *testing.T) {
		logCompletion := bl.LogStorageOperation(ctx, operation, provider, backupID)
		metadata := map[string]interface{}{
			"bytes_transferred": 1024,
			"transfer_rate":     "10MB/s",
		}
		logCompletion(nil, metadata)
	})

	// Test failed storage operation
	t.Run("failed storage operation", func(t *testing.T) {
		logCompletion := bl.LogStorageOperation(ctx, operation, provider, backupID)
		logCompletion(fmt.Errorf("storage operation failed"), nil)
	})
}

func TestBackupLogger_LogCompressionOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	operation := "compress"
	compressionType := CompressionTypeGzip
	originalSize := int64(1024)

	// Test successful compression
	t.Run("successful compression", func(t *testing.T) {
		logCompletion := bl.LogCompressionOperation(ctx, operation, compressionType, originalSize)
		finalSize := int64(512)
		logCompletion(nil, finalSize)
	})

	// Test failed compression
	t.Run("failed compression", func(t *testing.T) {
		logCompletion := bl.LogCompressionOperation(ctx, operation, compressionType, originalSize)
		logCompletion(fmt.Errorf("compression failed"), 0)
	})
}

func TestBackupLogger_LogEncryptionOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	operation := "encrypt"
	dataSize := int64(1024)

	// Test successful encryption
	t.Run("successful encryption", func(t *testing.T) {
		logCompletion := bl.LogEncryptionOperation(ctx, operation, dataSize)
		logCompletion(nil)
	})

	// Test failed encryption
	t.Run("failed encryption", func(t *testing.T) {
		logCompletion := bl.LogEncryptionOperation(ctx, operation, dataSize)
		logCompletion(fmt.Errorf("encryption failed"))
	})
}

func TestBackupLogger_WithCorrelationID(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger:        logger,
		CorrelationID: "original-id",
	})
	require.NoError(t, err)

	newCorrelationID := "new-correlation-id"
	newBL := bl.WithCorrelationID(newCorrelationID)

	assert.Equal(t, "original-id", bl.GetCorrelationID())
	assert.Equal(t, newCorrelationID, newBL.GetCorrelationID())
	assert.NotEqual(t, bl.GetCorrelationID(), newBL.GetCorrelationID())
}

func TestBackupLogger_ExportLogs(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()
	tempDir := t.TempDir()

	// Test JSON export
	t.Run("JSON export", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "logs.json")
		err := bl.ExportLogs(ctx, startTime, endTime, "json", outputPath)
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)

		// Verify file content is valid JSON
		content, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		var logs []LogEntry
		err = json.Unmarshal(content, &logs)
		assert.NoError(t, err)
		assert.Len(t, logs, 1) // Sample entry
	})

	// Test CSV export
	t.Run("CSV export", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "logs.csv")
		err := bl.ExportLogs(ctx, startTime, endTime, "csv", outputPath)
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)

		// Verify file content has CSV header
		content, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		lines := strings.Split(string(content), "\n")
		assert.GreaterOrEqual(t, len(lines), 2) // Header + at least one data line
		assert.Contains(t, lines[0], "timestamp,correlation_id,operation")
	})

	// Test unsupported format
	t.Run("unsupported format", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "logs.xml")
		err := bl.ExportLogs(ctx, startTime, endTime, "xml", outputPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
	})
}

func TestBackupLogger_AuditLogging(t *testing.T) {
	tempDir := t.TempDir()
	auditLogFile := filepath.Join(tempDir, "audit.log")

	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger:         logger,
		AuditLogFile:   auditLogFile,
		EnableAuditLog: true,
	})
	require.NoError(t, err)
	require.NotNil(t, bl.auditLogger)

	// Create context with user information
	ctx := context.WithValue(context.Background(), "user_id", "test-user")
	ctx = context.WithValue(ctx, "ip_address", "192.168.1.1")

	// Perform an operation that should generate audit logs
	backupID := "backup-123"
	databaseName := "test_db"
	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Database: databaseName,
		},
		StorageConfig: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}

	logCompletion := bl.LogBackupStart(ctx, backupID, databaseName, config)
	logCompletion(nil, &BackupMetadata{
		ID:       backupID,
		Size:     1024,
		Checksum: "abc123",
	})

	// Verify audit log file was created and contains entries
	_, err = os.Stat(auditLogFile)
	assert.NoError(t, err)

	// Read audit log content
	content, err := os.ReadFile(auditLogFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify audit log contains expected fields
	logContent := string(content)
	assert.Contains(t, logContent, "test-user")
	assert.Contains(t, logContent, "192.168.1.1")
	assert.Contains(t, logContent, "backup_create")
	assert.Contains(t, logContent, backupID)
}

func TestBackupLogger_ContextIntegration(t *testing.T) {
	logger := logging.NewDefaultLogger()
	bl, err := NewBackupLogger(BackupLoggerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	// Test with context containing request ID
	requestID := "req-123"
	ctx := logging.CreateContextWithRequestID(context.Background(), requestID)

	backupID := "backup-123"
	databaseName := "test_db"
	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Database: databaseName,
		},
		StorageConfig: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}

	// This should work without errors and include the request ID in logs
	logCompletion := bl.LogBackupStart(ctx, backupID, databaseName, config)
	logCompletion(nil, &BackupMetadata{
		ID:       backupID,
		Size:     1024,
		Checksum: "abc123",
	})

	// Verify the correlation ID is maintained
	assert.NotEmpty(t, bl.GetCorrelationID())
}
