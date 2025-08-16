package backup

import (
	"context"
	"fmt"
)

// StorageProviderFactory creates storage providers based on configuration
type StorageProviderFactory struct{}

// NewStorageProviderFactory creates a new storage provider factory
func NewStorageProviderFactory() *StorageProviderFactory {
	return &StorageProviderFactory{}
}

// CreateStorageProvider creates a storage provider based on the storage configuration
func (spf *StorageProviderFactory) CreateStorageProvider(ctx context.Context, config StorageConfig) (StorageProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, NewValidationError("invalid storage configuration", err)
	}

	switch config.Provider {
	case StorageProviderLocal:
		return NewLocalStorageProvider(config.Local)

	case StorageProviderS3:
		return NewS3StorageProvider(config.S3)

	case StorageProviderAzure:
		return NewAzureStorageProvider(config.Azure)

	case StorageProviderGCS:
		return NewGCSStorageProvider(ctx, config.GCS)

	default:
		return nil, NewValidationError(fmt.Sprintf("unsupported storage provider: %s", config.Provider), nil)
	}
}

// GetSupportedProviders returns a list of supported storage provider types
func (spf *StorageProviderFactory) GetSupportedProviders() []StorageProviderType {
	return []StorageProviderType{
		StorageProviderLocal,
		StorageProviderS3,
		StorageProviderAzure,
		StorageProviderGCS,
	}
}

// ValidateStorageConfig validates a storage configuration without creating the provider
func (spf *StorageProviderFactory) ValidateStorageConfig(config StorageConfig) error {
	return config.Validate()
}

// CreateMultipleStorageProviders creates multiple storage providers for redundancy
func (spf *StorageProviderFactory) CreateMultipleStorageProviders(ctx context.Context, configs []StorageConfig) ([]StorageProvider, error) {
	if len(configs) == 0 {
		return nil, NewValidationError("at least one storage configuration is required", nil)
	}

	var providers []StorageProvider
	var errors []error

	for i, config := range configs {
		provider, err := spf.CreateStorageProvider(ctx, config)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create storage provider %d (%s): %w", i, config.Provider, err))
			continue
		}
		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, NewStorageError("failed to create any storage providers", fmt.Errorf("errors: %v", errors))
	}

	return providers, nil
}

// MultiStorageProvider wraps multiple storage providers for redundancy
type MultiStorageProvider struct {
	providers []StorageProvider
	primary   StorageProvider
}

// NewMultiStorageProvider creates a new multi-storage provider
func NewMultiStorageProvider(providers []StorageProvider) (*MultiStorageProvider, error) {
	if len(providers) == 0 {
		return nil, NewValidationError("at least one storage provider is required", nil)
	}

	return &MultiStorageProvider{
		providers: providers,
		primary:   providers[0], // First provider is primary
	}, nil
}

// Store saves a backup to all storage providers
func (msp *MultiStorageProvider) Store(ctx context.Context, backup *Backup) error {
	var lastError error

	// Try to store in primary provider first
	if err := msp.primary.Store(ctx, backup); err != nil {
		lastError = err
	} else {
		// Primary succeeded, try to store in secondary providers
		for i := 1; i < len(msp.providers); i++ {
			if err := msp.providers[i].Store(ctx, backup); err != nil {
				// Log error but don't fail the operation
				// In a real implementation, you would log this error
				lastError = err
			}
		}
		return nil // Primary succeeded, so operation is successful
	}

	// Primary failed, try secondary providers
	for i := 1; i < len(msp.providers); i++ {
		if err := msp.providers[i].Store(ctx, backup); err != nil {
			lastError = err
			continue
		}
		return nil // At least one provider succeeded
	}

	return NewStorageError("failed to store backup in any provider", lastError)
}

// Retrieve loads a backup from the first available storage provider
func (msp *MultiStorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	var lastError error

	for _, provider := range msp.providers {
		backup, err := provider.Retrieve(ctx, backupID)
		if err != nil {
			lastError = err
			continue
		}
		return backup, nil
	}

	return nil, NewStorageError("failed to retrieve backup from any provider", lastError)
}

// Delete removes a backup from all storage providers
func (msp *MultiStorageProvider) Delete(ctx context.Context, backupID string) error {
	var errors []error

	for _, provider := range msp.providers {
		if err := provider.Delete(ctx, backupID); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == len(msp.providers) {
		return NewStorageError("failed to delete backup from any provider", fmt.Errorf("errors: %v", errors))
	}

	return nil // At least one provider succeeded
}

// List returns a list of backup metadata from the primary storage provider
func (msp *MultiStorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	// Use primary provider for listing
	return msp.primary.List(ctx, filter)
}

// GetMetadata retrieves metadata from the first available storage provider
func (msp *MultiStorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	var lastError error

	for _, provider := range msp.providers {
		metadata, err := provider.GetMetadata(ctx, backupID)
		if err != nil {
			lastError = err
			continue
		}
		return metadata, nil
	}

	return nil, NewStorageError("failed to get metadata from any provider", lastError)
}

// GetProviders returns the list of storage providers
func (msp *MultiStorageProvider) GetProviders() []StorageProvider {
	return msp.providers
}

// GetPrimaryProvider returns the primary storage provider
func (msp *MultiStorageProvider) GetPrimaryProvider() StorageProvider {
	return msp.primary
}

// SetPrimaryProvider sets the primary storage provider
func (msp *MultiStorageProvider) SetPrimaryProvider(index int) error {
	if index < 0 || index >= len(msp.providers) {
		return NewValidationError("invalid provider index", nil)
	}
	msp.primary = msp.providers[index]
	return nil
}

// HealthCheck performs health checks on all storage providers
func (msp *MultiStorageProvider) HealthCheck(ctx context.Context) map[int]error {
	results := make(map[int]error)

	for i, provider := range msp.providers {
		// Check if provider supports health check
		if healthChecker, ok := provider.(interface {
			HealthCheck(context.Context) error
		}); ok {
			results[i] = healthChecker.HealthCheck(ctx)
		} else {
			results[i] = nil // Provider doesn't support health check
		}
	}

	return results
}

// GetStorageInfo returns information about all storage providers
func (msp *MultiStorageProvider) GetStorageInfo() []map[string]interface{} {
	var infos []map[string]interface{}

	for _, provider := range msp.providers {
		// Check if provider supports storage info
		if infoProvider, ok := provider.(interface {
			GetStorageInfo() map[string]interface{}
		}); ok {
			infos = append(infos, infoProvider.GetStorageInfo())
		} else {
			infos = append(infos, map[string]interface{}{
				"provider": "unknown",
			})
		}
	}

	return infos
}
