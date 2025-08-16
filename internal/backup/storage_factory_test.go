package backup

import (
	"context"
	"testing"
)

func TestStorageProviderFactory(t *testing.T) {
	ctx := context.Background()
	factory := NewStorageProviderFactory()

	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
	}{
		{
			name: "local storage provider",
			config: StorageConfig{
				Provider: StorageProviderLocal,
				Local: &LocalConfig{
					BasePath:    "/tmp/backups",
					Permissions: 0755,
				},
			},
			wantErr: false,
		},
		{
			name: "S3 storage provider",
			config: StorageConfig{
				Provider: StorageProviderS3,
				S3: &S3Config{
					Bucket:    "test-bucket",
					Region:    "us-east-1",
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			},
			wantErr: false,
		},
		{
			name: "Azure storage provider",
			config: StorageConfig{
				Provider: StorageProviderAzure,
				Azure: &AzureConfig{
					AccountName:   "testaccount",
					AccountKey:    "dGVzdC1hY2NvdW50LWtleQ==",
					ContainerName: "test-container",
				},
			},
			wantErr: false,
		},
		{
			name: "GCS storage provider",
			config: StorageConfig{
				Provider: StorageProviderGCS,
				GCS: &GCSConfig{
					Bucket:          "test-bucket",
					CredentialsPath: "/path/to/credentials.json",
					ProjectID:       "test-project",
				},
			},
			wantErr: true, // Will fail because credentials file doesn't exist
		},
		{
			name: "invalid provider",
			config: StorageConfig{
				Provider: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateStorageProvider(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateStorageProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("CreateStorageProvider() returned nil provider without error")
			}
		})
	}
}

func TestStorageProviderFactoryGetSupportedProviders(t *testing.T) {
	factory := NewStorageProviderFactory()
	providers := factory.GetSupportedProviders()

	expectedProviders := []StorageProviderType{
		StorageProviderLocal,
		StorageProviderS3,
		StorageProviderAzure,
		StorageProviderGCS,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	for i, expected := range expectedProviders {
		if i >= len(providers) || providers[i] != expected {
			t.Errorf("Expected provider %s at index %d, got %s", expected, i, providers[i])
		}
	}
}

func TestStorageProviderFactoryValidateStorageConfig(t *testing.T) {
	factory := NewStorageProviderFactory()

	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
	}{
		{
			name: "valid local config",
			config: StorageConfig{
				Provider: StorageProviderLocal,
				Local: &LocalConfig{
					BasePath:    "/tmp/backups",
					Permissions: 0755,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing provider-specific config",
			config: StorageConfig{
				Provider: StorageProviderLocal,
				Local:    nil,
			},
			wantErr: true,
		},
		{
			name: "invalid config - empty provider",
			config: StorageConfig{
				Provider: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateStorageConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStorageConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiStorageProvider(t *testing.T) {
	// Create mock providers
	provider1 := NewMockStorageProvider()
	provider2 := NewMockStorageProvider()

	multiProvider, err := NewMultiStorageProvider([]StorageProvider{provider1, provider2})
	if err != nil {
		t.Fatalf("NewMultiStorageProvider() error = %v", err)
	}

	// Test that primary provider is set correctly
	if multiProvider.GetPrimaryProvider() != provider1 {
		t.Error("Expected first provider to be primary")
	}

	// Test SetPrimaryProvider
	err = multiProvider.SetPrimaryProvider(1)
	if err != nil {
		t.Errorf("SetPrimaryProvider() error = %v", err)
	}
	if multiProvider.GetPrimaryProvider() != provider2 {
		t.Error("Expected second provider to be primary after SetPrimaryProvider")
	}

	// Test invalid index
	err = multiProvider.SetPrimaryProvider(5)
	if err == nil {
		t.Error("Expected error for invalid provider index")
	}

	// Test GetProviders
	providers := multiProvider.GetProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestCreateMultipleStorageProviders(t *testing.T) {
	ctx := context.Background()
	factory := NewStorageProviderFactory()

	configs := []StorageConfig{
		{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "/tmp/backups1",
				Permissions: 0755,
			},
		},
		{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "/tmp/backups2",
				Permissions: 0755,
			},
		},
	}

	providers, err := factory.CreateMultipleStorageProviders(ctx, configs)
	if err != nil {
		t.Errorf("CreateMultipleStorageProviders() error = %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Test with empty configs
	_, err = factory.CreateMultipleStorageProviders(ctx, []StorageConfig{})
	if err == nil {
		t.Error("Expected error for empty configs")
	}
}

func TestMultiStorageProviderOperations(t *testing.T) {
	ctx := context.Background()

	// Create mock providers
	provider1 := NewMockStorageProvider()
	provider2 := NewMockStorageProvider()

	multiProvider, err := NewMultiStorageProvider([]StorageProvider{provider1, provider2})
	if err != nil {
		t.Fatalf("NewMultiStorageProvider() error = %v", err)
	}

	// Create test backup
	backup := createTestCloudBackup()

	// Test Store operation
	err = multiProvider.Store(ctx, backup)
	if err != nil {
		t.Errorf("Store() error = %v", err)
	}

	// Test Retrieve operation
	retrieved, err := multiProvider.Retrieve(ctx, backup.ID)
	if err != nil {
		t.Errorf("Retrieve() error = %v", err)
	}
	if retrieved.ID != backup.ID {
		t.Errorf("Expected backup ID %s, got %s", backup.ID, retrieved.ID)
	}

	// Test List operation
	backups, err := multiProvider.List(ctx, StorageFilter{})
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(backups) == 0 {
		t.Error("Expected at least one backup in list")
	}

	// Test GetMetadata operation
	metadata, err := multiProvider.GetMetadata(ctx, backup.ID)
	if err != nil {
		t.Errorf("GetMetadata() error = %v", err)
	}
	if metadata.ID != backup.ID {
		t.Errorf("Expected metadata ID %s, got %s", backup.ID, metadata.ID)
	}

	// Test Delete operation
	err = multiProvider.Delete(ctx, backup.ID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestMultiStorageProviderHealthCheck(t *testing.T) {
	ctx := context.Background()

	// Create mock providers
	provider1 := NewMockStorageProvider()
	provider2 := NewMockStorageProvider()

	multiProvider, err := NewMultiStorageProvider([]StorageProvider{provider1, provider2})
	if err != nil {
		t.Fatalf("NewMultiStorageProvider() error = %v", err)
	}

	// Test health check
	results := multiProvider.HealthCheck(ctx)
	if len(results) != 2 {
		t.Errorf("Expected 2 health check results, got %d", len(results))
	}

	// Mock providers don't implement health check, so results should be nil
	for i, result := range results {
		if result != nil {
			t.Errorf("Expected nil health check result for provider %d, got %v", i, result)
		}
	}
}

func TestMultiStorageProviderGetStorageInfo(t *testing.T) {
	// Create mock providers
	provider1 := NewMockStorageProvider()
	provider2 := NewMockStorageProvider()

	multiProvider, err := NewMultiStorageProvider([]StorageProvider{provider1, provider2})
	if err != nil {
		t.Fatalf("NewMultiStorageProvider() error = %v", err)
	}

	// Test get storage info
	infos := multiProvider.GetStorageInfo()
	if len(infos) != 2 {
		t.Errorf("Expected 2 storage info results, got %d", len(infos))
	}

	// Mock providers don't implement GetStorageInfo, so should return unknown
	for i, info := range infos {
		if info["provider"] != "unknown" {
			t.Errorf("Expected 'unknown' provider for info %d, got %v", i, info["provider"])
		}
	}
}
