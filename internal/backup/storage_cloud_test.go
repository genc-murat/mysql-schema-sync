package backup

import (
	"context"
	"strings"
	"testing"
	"time"

	"mysql-schema-sync/internal/schema"
)

// TestS3StorageProviderValidation tests S3 storage provider validation
func TestS3StorageProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing bucket",
			config: &S3Config{
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr: true,
		},
		{
			name: "missing region",
			config: &S3Config{
				Bucket:    "test-bucket",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr: true,
		},
		{
			name: "missing access key",
			config: &S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				SecretKey: "test-secret-key",
			},
			wantErr: true,
		},
		{
			name: "missing secret key",
			config: &S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-access-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewS3StorageProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewS3StorageProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAzureStorageProviderValidation tests Azure storage provider validation
func TestAzureStorageProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *AzureConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AzureConfig{
				AccountName:   "testaccount",
				AccountKey:    "dGVzdC1hY2NvdW50LWtleQ==", // base64 encoded test key
				ContainerName: "test-container",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing account name",
			config: &AzureConfig{
				AccountKey:    "dGVzdC1hY2NvdW50LWtleQ==",
				ContainerName: "test-container",
			},
			wantErr: true,
		},
		{
			name: "missing account key",
			config: &AzureConfig{
				AccountName:   "testaccount",
				ContainerName: "test-container",
			},
			wantErr: true,
		},
		{
			name: "missing container name",
			config: &AzureConfig{
				AccountName: "testaccount",
				AccountKey:  "dGVzdC1hY2NvdW50LWtleQ==",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureStorageProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureStorageProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGCSStorageProviderValidation tests GCS storage provider validation
func TestGCSStorageProviderValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *GCSConfig
		wantErr bool
	}{
		{
			name: "valid config with credentials file",
			config: &GCSConfig{
				Bucket:          "test-bucket",
				CredentialsPath: "/path/to/credentials.json",
				ProjectID:       "test-project",
			},
			wantErr: true, // Will fail because credentials file doesn't exist, but validation passes
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing bucket",
			config: &GCSConfig{
				CredentialsPath: "/path/to/credentials.json",
				ProjectID:       "test-project",
			},
			wantErr: true,
		},
		{
			name: "missing credentials path",
			config: &GCSConfig{
				Bucket:    "test-bucket",
				ProjectID: "test-project",
			},
			wantErr: true,
		},
		{
			name: "missing project ID",
			config: &GCSConfig{
				Bucket:          "test-bucket",
				CredentialsPath: "/path/to/credentials.json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGCSStorageProvider(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGCSStorageProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStorageProviderSanitization tests backup ID sanitization for cloud providers
func TestStorageProviderSanitization(t *testing.T) {
	tests := []struct {
		name     string
		backupID string
		expected string
	}{
		{
			name:     "normal backup ID",
			backupID: "backup-123",
			expected: "backup-123",
		},
		{
			name:     "backup ID with spaces",
			backupID: "backup 123",
			expected: "backup_123",
		},
		{
			name:     "backup ID with backslash",
			backupID: "backup\\123",
			expected: "backup_123",
		},
	}

	// Test S3 provider sanitization
	t.Run("S3 sanitization", func(t *testing.T) {
		provider := &S3StorageProvider{}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.sanitizeBackupID(tt.backupID)
				if result != tt.expected {
					t.Errorf("S3 sanitizeBackupID() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	// Test Azure provider sanitization
	t.Run("Azure sanitization", func(t *testing.T) {
		provider := &AzureStorageProvider{}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.sanitizeBackupID(tt.backupID)
				if result != tt.expected {
					t.Errorf("Azure sanitizeBackupID() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	// Test GCS provider sanitization
	t.Run("GCS sanitization", func(t *testing.T) {
		provider := &GCSStorageProvider{}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.sanitizeBackupID(tt.backupID)
				if result != tt.expected {
					t.Errorf("GCS sanitizeBackupID() = %v, want %v", result, tt.expected)
				}
			})
		}
	})
}

// TestStorageProviderObjectKeyExtraction tests backup ID extraction from object keys/names
func TestStorageProviderObjectKeyExtraction(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "valid metadata key",
			key:      "backups/backup-123/metadata.json",
			expected: "backup-123",
		},
		{
			name:     "backup data key",
			key:      "backups/backup-123/backup.json",
			expected: "",
		},
		{
			name:     "invalid key format",
			key:      "backups/backup-123",
			expected: "",
		},
		{
			name:     "wrong prefix",
			key:      "other/backup-123/metadata.json",
			expected: "",
		},
	}

	// Test S3 provider extraction
	t.Run("S3 extraction", func(t *testing.T) {
		provider := &S3StorageProvider{prefix: "backups/"}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.extractBackupIDFromKey(tt.key)
				if result != tt.expected {
					t.Errorf("S3 extractBackupIDFromKey() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	// Test Azure provider extraction
	t.Run("Azure extraction", func(t *testing.T) {
		provider := &AzureStorageProvider{prefix: "backups/"}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.extractBackupIDFromBlobName(tt.key)
				if result != tt.expected {
					t.Errorf("Azure extractBackupIDFromBlobName() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	// Test GCS provider extraction
	t.Run("GCS extraction", func(t *testing.T) {
		provider := &GCSStorageProvider{prefix: "backups/"}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := provider.extractBackupIDFromObjectName(tt.key)
				if result != tt.expected {
					t.Errorf("GCS extractBackupIDFromObjectName() = %v, want %v", result, tt.expected)
				}
			})
		}
	})
}

// TestStorageProviderGetStorageInfo tests storage info retrieval
func TestStorageProviderGetStorageInfo(t *testing.T) {
	// Test S3 provider
	t.Run("S3 storage info", func(t *testing.T) {
		provider := &S3StorageProvider{
			bucket: "test-bucket",
			prefix: "backups/",
		}
		info := provider.GetStorageInfo()

		if info["provider"] != "s3" {
			t.Errorf("Expected provider to be 's3', got %v", info["provider"])
		}
		if info["bucket"] != "test-bucket" {
			t.Errorf("Expected bucket to be 'test-bucket', got %v", info["bucket"])
		}
		if info["prefix"] != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", info["prefix"])
		}
	})

	// Test Azure provider
	t.Run("Azure storage info", func(t *testing.T) {
		provider := &AzureStorageProvider{
			containerName: "test-container",
			prefix:        "backups/",
		}
		info := provider.GetStorageInfo()

		if info["provider"] != "azure" {
			t.Errorf("Expected provider to be 'azure', got %v", info["provider"])
		}
		if info["container"] != "test-container" {
			t.Errorf("Expected container to be 'test-container', got %v", info["container"])
		}
		if info["prefix"] != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", info["prefix"])
		}
	})

	// Test GCS provider
	t.Run("GCS storage info", func(t *testing.T) {
		provider := &GCSStorageProvider{
			bucketName: "test-bucket",
			prefix:     "backups/",
		}
		info := provider.GetStorageInfo()

		if info["provider"] != "gcs" {
			t.Errorf("Expected provider to be 'gcs', got %v", info["provider"])
		}
		if info["bucket"] != "test-bucket" {
			t.Errorf("Expected bucket to be 'test-bucket', got %v", info["bucket"])
		}
		if info["prefix"] != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", info["prefix"])
		}
	})
}

// TestStorageProviderPrefixManagement tests prefix getter and setter methods
func TestStorageProviderPrefixManagement(t *testing.T) {
	// Test S3 provider
	t.Run("S3 prefix management", func(t *testing.T) {
		provider := &S3StorageProvider{prefix: "backups/"}

		if provider.GetPrefix() != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", provider.GetPrefix())
		}

		provider.SetPrefix("new-prefix/")
		if provider.GetPrefix() != "new-prefix/" {
			t.Errorf("Expected prefix to be 'new-prefix/', got %v", provider.GetPrefix())
		}
	})

	// Test Azure provider
	t.Run("Azure prefix management", func(t *testing.T) {
		provider := &AzureStorageProvider{prefix: "backups/"}

		if provider.GetPrefix() != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", provider.GetPrefix())
		}

		provider.SetPrefix("new-prefix/")
		if provider.GetPrefix() != "new-prefix/" {
			t.Errorf("Expected prefix to be 'new-prefix/', got %v", provider.GetPrefix())
		}
	})

	// Test GCS provider
	t.Run("GCS prefix management", func(t *testing.T) {
		provider := &GCSStorageProvider{prefix: "backups/"}

		if provider.GetPrefix() != "backups/" {
			t.Errorf("Expected prefix to be 'backups/', got %v", provider.GetPrefix())
		}

		provider.SetPrefix("new-prefix/")
		if provider.GetPrefix() != "new-prefix/" {
			t.Errorf("Expected prefix to be 'new-prefix/', got %v", provider.GetPrefix())
		}
	})
}

// MockStorageProvider implements StorageProvider for testing
type MockStorageProvider struct {
	backups map[string]*Backup
	prefix  string
}

// NewMockStorageProvider creates a new mock storage provider
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		backups: make(map[string]*Backup),
		prefix:  "backups/",
	}
}

func (msp *MockStorageProvider) Store(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return NewValidationError("backup cannot be nil", nil)
	}

	backup.Metadata.StorageLocation = "mock://" + backup.ID
	if err := backup.CalculateChecksum(); err != nil {
		return err
	}
	backup.Metadata.Checksum = backup.Checksum

	if err := backup.Validate(); err != nil {
		return NewValidationError("invalid backup data", err)
	}

	msp.backups[backup.ID] = backup
	return nil
}

func (msp *MockStorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	backup, exists := msp.backups[backupID]
	if !exists {
		return nil, NewStorageError("backup not found", nil)
	}

	return backup, nil
}

func (msp *MockStorageProvider) Delete(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID cannot be empty", nil)
	}

	if _, exists := msp.backups[backupID]; !exists {
		return NewStorageError("backup not found", nil)
	}

	delete(msp.backups, backupID)
	return nil
}

func (msp *MockStorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	var backups []*BackupMetadata

	for _, backup := range msp.backups {
		if filter.Prefix != "" && !strings.HasPrefix(backup.ID, filter.Prefix) {
			continue
		}

		backups = append(backups, backup.Metadata)

		if filter.MaxItems > 0 && len(backups) >= filter.MaxItems {
			break
		}
	}

	return backups, nil
}

func (msp *MockStorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	backup, exists := msp.backups[backupID]
	if !exists {
		return nil, NewStorageError("backup not found", nil)
	}

	return backup.Metadata, nil
}

// TestMockStorageProvider tests the mock storage provider implementation
func TestMockStorageProvider(t *testing.T) {
	var _ StorageProvider = &MockStorageProvider{}

	provider := NewMockStorageProvider()
	ctx := context.Background()

	// Create test backup
	backup := createTestCloudBackup()

	// Test Store
	err := provider.Store(ctx, backup)
	if err != nil {
		t.Errorf("Store() error = %v", err)
	}

	// Test Retrieve
	retrieved, err := provider.Retrieve(ctx, backup.ID)
	if err != nil {
		t.Errorf("Retrieve() error = %v", err)
	}
	if retrieved.ID != backup.ID {
		t.Errorf("Expected backup ID %s, got %s", backup.ID, retrieved.ID)
	}

	// Test List
	backups, err := provider.List(ctx, StorageFilter{})
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	// Test GetMetadata
	metadata, err := provider.GetMetadata(ctx, backup.ID)
	if err != nil {
		t.Errorf("GetMetadata() error = %v", err)
	}
	if metadata.ID != backup.ID {
		t.Errorf("Expected metadata ID %s, got %s", backup.ID, metadata.ID)
	}

	// Test Delete
	err = provider.Delete(ctx, backup.ID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = provider.Retrieve(ctx, backup.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted backup")
	}
}

// Helper function to create a test backup for cloud storage tests
func createTestCloudBackup() *Backup {
	now := time.Now()
	backupID := GenerateBackupID()
	backup := &Backup{
		ID: backupID,
		Metadata: &BackupMetadata{
			ID:              backupID,
			DatabaseName:    "testdb",
			CreatedAt:       now,
			CreatedBy:       "test-user",
			Description:     "Test cloud backup",
			Size:            1024,
			CompressedSize:  512,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
			StorageLocation: "", // Will be set by storage provider
			Checksum:        "",
			Tags:            map[string]string{"env": "test", "type": "cloud"},
		},
		SchemaSnapshot: &schema.Schema{
			Name: "testdb",
			Tables: map[string]*schema.Table{
				"test_table": {
					Name: "test_table",
					Columns: map[string]*schema.Column{
						"id": {
							Name:       "id",
							DataType:   "int",
							IsNullable: false,
							Position:   1,
						},
					},
					Indexes:     []*schema.Index{},
					Constraints: map[string]*schema.Constraint{},
				},
			},
			Indexes: map[string]*schema.Index{},
		},
		DataDefinitions: []string{"CREATE TABLE test_table (id INT NOT NULL)"},
		Triggers:        []TriggerDefinition{},
		Views:           []ViewDefinition{},
		Procedures:      []ProcedureDefinition{},
		Functions:       []FunctionDefinition{},
	}

	// Don't calculate checksum here - let the storage provider do it
	backup.Checksum = ""
	backup.Metadata.Checksum = ""

	return backup
}
