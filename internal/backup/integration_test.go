package backup

import (
	"context"
	"testing"
)

// TestCloudStorageProvidersIntegration tests that cloud storage providers can be created
// through the factory and have the expected interfaces
func TestCloudStorageProvidersIntegration(t *testing.T) {
	ctx := context.Background()
	factory := NewStorageProviderFactory()

	tests := []struct {
		name     string
		config   StorageConfig
		wantType string
	}{
		{
			name: "S3 provider creation",
			config: StorageConfig{
				Provider: StorageProviderS3,
				S3: &S3Config{
					Bucket:    "test-bucket",
					Region:    "us-east-1",
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			},
			wantType: "*backup.S3StorageProvider",
		},
		{
			name: "Azure provider creation",
			config: StorageConfig{
				Provider: StorageProviderAzure,
				Azure: &AzureConfig{
					AccountName:   "testaccount",
					AccountKey:    "dGVzdC1hY2NvdW50LWtleQ==",
					ContainerName: "test-container",
				},
			},
			wantType: "*backup.AzureStorageProvider",
		},
		{
			name: "Local provider creation",
			config: StorageConfig{
				Provider: StorageProviderLocal,
				Local: &LocalConfig{
					BasePath:    "/tmp/backups",
					Permissions: 0755,
				},
			},
			wantType: "*backup.LocalStorageProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateStorageProvider(ctx, tt.config)
			if err != nil {
				t.Errorf("CreateStorageProvider() error = %v", err)
				return
			}

			if provider == nil {
				t.Error("CreateStorageProvider() returned nil provider")
				return
			}

			// Verify the provider implements the StorageProvider interface
			var _ StorageProvider = provider

			// Test that the provider has the expected methods by calling them with safe parameters
			_, err = provider.List(ctx, StorageFilter{MaxItems: 1})
			if err != nil {
				// This is expected to fail for cloud providers without real credentials
				// but it should not panic and should return a proper error
				t.Logf("List() returned expected error: %v", err)
			}

			// Test GetMetadata with a non-existent backup ID
			_, err = provider.GetMetadata(ctx, "non-existent-backup")
			if err == nil {
				t.Error("GetMetadata() should return error for non-existent backup")
			}
		})
	}
}

// TestMultiStorageProviderIntegration tests the multi-storage provider functionality
func TestMultiStorageProviderIntegration(t *testing.T) {
	ctx := context.Background()
	factory := NewStorageProviderFactory()

	// Create multiple local storage providers for testing
	configs := []StorageConfig{
		{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "/tmp/backups-primary",
				Permissions: 0755,
			},
		},
		{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "/tmp/backups-secondary",
				Permissions: 0755,
			},
		},
	}

	providers, err := factory.CreateMultipleStorageProviders(ctx, configs)
	if err != nil {
		t.Errorf("CreateMultipleStorageProviders() error = %v", err)
		return
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
		return
	}

	// Create multi-storage provider
	multiProvider, err := NewMultiStorageProvider(providers)
	if err != nil {
		t.Errorf("NewMultiStorageProvider() error = %v", err)
		return
	}

	// Test that it implements the StorageProvider interface
	var _ StorageProvider = multiProvider

	// Test basic operations
	_, err = multiProvider.List(ctx, StorageFilter{MaxItems: 1})
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	// Test health check
	healthResults := multiProvider.HealthCheck(ctx)
	if len(healthResults) != 2 {
		t.Errorf("Expected 2 health check results, got %d", len(healthResults))
	}

	// Test storage info
	storageInfos := multiProvider.GetStorageInfo()
	if len(storageInfos) != 2 {
		t.Errorf("Expected 2 storage info results, got %d", len(storageInfos))
	}
}

// TestStorageProviderFactorySupport tests that all expected providers are supported
func TestStorageProviderFactorySupport(t *testing.T) {
	factory := NewStorageProviderFactory()
	supportedProviders := factory.GetSupportedProviders()

	expectedProviders := map[StorageProviderType]bool{
		StorageProviderLocal: false,
		StorageProviderS3:    false,
		StorageProviderAzure: false,
		StorageProviderGCS:   false,
	}

	for _, provider := range supportedProviders {
		if _, exists := expectedProviders[provider]; exists {
			expectedProviders[provider] = true
		} else {
			t.Errorf("Unexpected provider type: %s", provider)
		}
	}

	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s not found in supported providers", provider)
		}
	}
}

// TestCloudStorageProviderValidation tests validation of cloud storage configurations
func TestCloudStorageProviderValidation(t *testing.T) {
	factory := NewStorageProviderFactory()

	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
	}{
		{
			name: "valid S3 config",
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
			name: "invalid S3 config - missing bucket",
			config: StorageConfig{
				Provider: StorageProviderS3,
				S3: &S3Config{
					Region:    "us-east-1",
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			},
			wantErr: true,
		},
		{
			name: "valid Azure config",
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
			name: "invalid Azure config - missing account name",
			config: StorageConfig{
				Provider: StorageProviderAzure,
				Azure: &AzureConfig{
					AccountKey:    "dGVzdC1hY2NvdW50LWtleQ==",
					ContainerName: "test-container",
				},
			},
			wantErr: true,
		},
		{
			name: "valid GCS config",
			config: StorageConfig{
				Provider: StorageProviderGCS,
				GCS: &GCSConfig{
					Bucket:          "test-bucket",
					CredentialsPath: "/path/to/credentials.json",
					ProjectID:       "test-project",
				},
			},
			wantErr: false, // Validation passes, but creation will fail due to missing credentials file
		},
		{
			name: "invalid GCS config - missing bucket",
			config: StorageConfig{
				Provider: StorageProviderGCS,
				GCS: &GCSConfig{
					CredentialsPath: "/path/to/credentials.json",
					ProjectID:       "test-project",
				},
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
