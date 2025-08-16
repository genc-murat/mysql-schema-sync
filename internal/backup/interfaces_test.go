package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/schema"
)

// TestBackupConfigValidation tests the validation of backup configuration
func TestBackupConfigValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  BackupConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BackupConfig{
				DatabaseConfig: database.DatabaseConfig{
					Host:     "localhost",
					Port:     3306,
					Username: "user",
					Password: "pass",
					Database: "testdb",
					Timeout:  30 * time.Second,
				},
				StorageConfig: StorageConfig{
					Provider: StorageProviderLocal,
					Local: &LocalConfig{
						BasePath: "/tmp/backups",
					},
				},
				CompressionType: CompressionTypeGzip,
				Description:     "Test backup",
				Tags:            map[string]string{"env": "test"},
			},
			wantErr: false,
		},
		{
			name: "missing database host",
			config: BackupConfig{
				DatabaseConfig: database.DatabaseConfig{
					Username: "user",
					Database: "testdb",
				},
				StorageConfig: StorageConfig{
					Provider: StorageProviderLocal,
					Local: &LocalConfig{
						BasePath: "/tmp/backups",
					},
				},
				CompressionType: CompressionTypeGzip,
			},
			wantErr: true,
		},
		{
			name: "invalid compression type",
			config: BackupConfig{
				DatabaseConfig: database.DatabaseConfig{
					Host:     "localhost",
					Username: "user",
					Database: "testdb",
				},
				StorageConfig: StorageConfig{
					Provider: StorageProviderLocal,
					Local: &LocalConfig{
						BasePath: "/tmp/backups",
					},
				},
				CompressionType: CompressionType("INVALID"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBackupConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBackupConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBackupMetadataValidation tests the validation of backup metadata
func TestBackupMetadataValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		metadata *BackupMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: &BackupMetadata{
				ID:              "backup-123",
				DatabaseName:    "testdb",
				CreatedAt:       time.Now().Add(-1 * time.Hour),
				CreatedBy:       "user",
				Description:     "Test backup",
				Size:            1024,
				CompressedSize:  512,
				CompressionType: CompressionTypeGzip,
				Status:          BackupStatusCompleted,
				Tags:            map[string]string{"env": "test"},
			},
			wantErr: false,
		},
		{
			name: "missing backup ID",
			metadata: &BackupMetadata{
				DatabaseName: "testdb",
				CreatedAt:    time.Now(),
				Status:       BackupStatusCompleted,
			},
			wantErr: true,
		},
		{
			name: "future creation time",
			metadata: &BackupMetadata{
				ID:           "backup-123",
				DatabaseName: "testdb",
				CreatedAt:    time.Now().Add(1 * time.Hour),
				Status:       BackupStatusCompleted,
			},
			wantErr: true,
		},
		{
			name: "negative size",
			metadata: &BackupMetadata{
				ID:           "backup-123",
				DatabaseName: "testdb",
				CreatedAt:    time.Now().Add(-1 * time.Hour),
				Size:         -100,
				Status:       BackupStatusCompleted,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBackupMetadata(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBackupMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBackupErrorTypes tests the backup error types and constructors
func TestBackupErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() error
		errorType   BackupErrorType
	}{
		{
			name:        "storage error",
			constructor: func() error { return NewStorageError("storage failed", nil) },
			errorType:   BackupErrorTypeStorage,
		},
		{
			name:        "validation error",
			constructor: func() error { return NewValidationError("validation failed", nil) },
			errorType:   BackupErrorTypeValidation,
		},
		{
			name:        "compression error",
			constructor: func() error { return NewCompressionError("compression failed", nil) },
			errorType:   BackupErrorTypeCompression,
		},
		{
			name:        "encryption error",
			constructor: func() error { return NewEncryptionError("encryption failed", nil) },
			errorType:   BackupErrorTypeEncryption,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			backupErr, ok := err.(*BackupError)
			if !ok {
				t.Errorf("Expected BackupError, got %T", err)
				return
			}
			if backupErr.Type != tt.errorType {
				t.Errorf("Expected error type %s, got %s", tt.errorType, backupErr.Type)
			}
		})
	}
}

// TestIsRetryable tests the retry logic for different error types
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		retry bool
	}{
		{
			name:  "network error is retryable",
			err:   NewNetworkError("network timeout", nil),
			retry: true,
		},
		{
			name:  "storage error is retryable",
			err:   NewStorageError("storage unavailable", nil),
			retry: true,
		},
		{
			name:  "validation error is not retryable",
			err:   NewValidationError("invalid data", nil),
			retry: false,
		},
		{
			name:  "corruption error is not retryable",
			err:   NewCorruptionError("data corrupted", nil),
			retry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.retry {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retry)
			}
		})
	}
}

// MockBackupManager implements BackupManager for testing
type MockBackupManager struct{}

func (m *MockBackupManager) CreateBackup(ctx context.Context, config BackupConfig) (*Backup, error) {
	return &Backup{
		ID: "test-backup-123",
		Metadata: &BackupMetadata{
			ID:           "test-backup-123",
			DatabaseName: config.DatabaseConfig.Database,
			CreatedAt:    time.Now(),
			Status:       BackupStatusCompleted,
		},
		SchemaSnapshot: &schema.Schema{},
	}, nil
}

func (m *MockBackupManager) ListBackups(ctx context.Context, filter BackupFilter) ([]*BackupMetadata, error) {
	return []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: "testdb",
			CreatedAt:    time.Now().Add(-1 * time.Hour),
			Status:       BackupStatusCompleted,
		},
	}, nil
}

func (m *MockBackupManager) DeleteBackup(ctx context.Context, backupID string) error {
	return nil
}

func (m *MockBackupManager) ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	return &ValidationResult{
		Valid:         true,
		CheckedAt:     time.Now(),
		ChecksumValid: true,
	}, nil
}

func (m *MockBackupManager) ExportBackup(ctx context.Context, backupID string, destination string) error {
	return nil
}

func (m *MockBackupManager) GetBackupsByDatabase(ctx context.Context, databaseName string) ([]*BackupMetadata, error) {
	return []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    time.Now().Add(-1 * time.Hour),
			Status:       BackupStatusCompleted,
		},
	}, nil
}

func (m *MockBackupManager) GetRetentionManager() RetentionManager {
	return nil // Return nil for testing
}

func (m *MockBackupManager) GetStorageMonitor() StorageMonitor {
	return nil // Return nil for testing
}

// TestMockBackupManager tests that our mock implements the interface correctly
func TestMockBackupManager(t *testing.T) {
	var _ BackupManager = &MockBackupManager{}

	manager := &MockBackupManager{}
	ctx := context.Background()

	// Test CreateBackup
	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Database: "testdb",
		},
	}
	backup, err := manager.CreateBackup(ctx, config)
	if err != nil {
		t.Errorf("CreateBackup() error = %v", err)
	}
	if backup.ID != "test-backup-123" {
		t.Errorf("Expected backup ID 'test-backup-123', got %s", backup.ID)
	}

	// Test ListBackups
	backups, err := manager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		t.Errorf("ListBackups() error = %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	// Test ValidateBackup
	result, err := manager.ValidateBackup(ctx, "test-backup")
	if err != nil {
		t.Errorf("ValidateBackup() error = %v", err)
	}
	if !result.Valid {
		t.Errorf("Expected backup to be valid")
	}
}
