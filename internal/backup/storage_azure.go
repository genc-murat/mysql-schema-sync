package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureStorageProvider implements StorageProvider for Azure Blob Storage
type AzureStorageProvider struct {
	serviceURL    azblob.ServiceURL
	containerName string
	prefix        string
}

// NewAzureStorageProvider creates a new AzureStorageProvider instance
func NewAzureStorageProvider(config *AzureConfig) (*AzureStorageProvider, error) {
	if config == nil {
		return nil, NewValidationError("Azure storage configuration is required", nil)
	}

	if err := config.Validate(); err != nil {
		return nil, NewValidationError("invalid Azure storage configuration", err)
	}

	// Create Azure credentials
	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, NewStorageError("failed to create Azure credentials", err)
	}

	// Create pipeline
	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// Create service URL
	serviceURL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", config.AccountName))
	if err != nil {
		return nil, NewStorageError("failed to parse Azure service URL", err)
	}

	provider := &AzureStorageProvider{
		serviceURL:    azblob.NewServiceURL(*serviceURL, pipeline),
		containerName: config.ContainerName,
		prefix:        "backups/", // Default prefix for backup blobs
	}

	return provider, nil
}

// Store saves a backup to Azure Blob Storage
func (azp *AzureStorageProvider) Store(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return NewValidationError("backup cannot be nil", nil)
	}

	// Set storage location
	blobName := azp.getBackupBlobName(backup.ID)
	backup.Metadata.StorageLocation = fmt.Sprintf("azure://%s/%s", azp.containerName, blobName)

	// Calculate checksum after updating storage location
	if err := backup.CalculateChecksum(); err != nil {
		return NewStorageError("failed to calculate backup checksum", err)
	}
	backup.Metadata.Checksum = backup.Checksum

	if err := backup.Validate(); err != nil {
		return NewValidationError("invalid backup data", err)
	}

	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)

	// Serialize backup data
	backupData, err := backup.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize backup data", err)
	}

	// Upload backup data to Azure
	backupBlobURL := containerURL.NewBlockBlobURL(blobName + "/backup.json")
	_, err = azblob.UploadBufferToBlockBlob(ctx, backupData, backupBlobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024, // 4MB blocks
		Parallelism: 16,
		Metadata: azblob.Metadata{
			"backup-id":       backup.ID,
			"database-name":   backup.Metadata.DatabaseName,
			"created-by":      backup.Metadata.CreatedBy,
			"compression":     string(backup.Metadata.CompressionType),
			"backup-checksum": backup.Checksum,
		},
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{
			ContentType: "application/json",
		},
	})
	if err != nil {
		return NewStorageError("failed to upload backup to Azure", err)
	}

	// Upload metadata separately for quick access
	metadataData, err := backup.Metadata.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize metadata", err)
	}

	metadataBlobURL := containerURL.NewBlockBlobURL(blobName + "/metadata.json")
	_, err = azblob.UploadBufferToBlockBlob(ctx, metadataData, metadataBlobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16,
		Metadata: azblob.Metadata{
			"backup-id":     backup.ID,
			"database-name": backup.Metadata.DatabaseName,
		},
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{
			ContentType: "application/json",
		},
	})
	if err != nil {
		return NewStorageError("failed to upload metadata to Azure", err)
	}

	return nil
}

// Retrieve loads a backup from Azure Blob Storage
func (azp *AzureStorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	blobName := azp.getBackupBlobName(backupID) + "/backup.json"

	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)
	blobURL := containerURL.NewBlockBlobURL(blobName)

	// Download backup data from Azure
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("failed to download backup %s from Azure", backupID), err)
	}

	// Read backup data
	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	defer bodyStream.Close()

	data, err := io.ReadAll(bodyStream)
	if err != nil {
		return nil, NewStorageError("failed to read backup data", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, NewStorageError("failed to unmarshal backup data", err)
	}

	// Validate backup integrity
	if !backup.VerifyChecksum() {
		return nil, NewCorruptionError("backup checksum verification failed", nil)
	}

	return &backup, nil
}

// Delete removes a backup from Azure Blob Storage
func (azp *AzureStorageProvider) Delete(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID cannot be empty", nil)
	}

	blobPrefix := azp.getBackupBlobName(backupID)

	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)

	// List all blobs with the backup prefix
	var blobsToDelete []string
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listResponse, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{
			Prefix: blobPrefix,
		})
		if err != nil {
			return NewStorageError("failed to list backup blobs", err)
		}

		for _, blob := range listResponse.Segment.BlobItems {
			blobsToDelete = append(blobsToDelete, blob.Name)
		}

		marker = listResponse.NextMarker
	}

	if len(blobsToDelete) == 0 {
		return NewStorageError(fmt.Sprintf("backup %s not found", backupID), nil)
	}

	// Delete all blobs for this backup
	for _, blobName := range blobsToDelete {
		blobURL := containerURL.NewBlockBlobURL(blobName)
		_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
		if err != nil {
			return NewStorageError(fmt.Sprintf("failed to delete blob %s", blobName), err)
		}
	}

	return nil
}

// List returns a list of backup metadata matching the filter
func (azp *AzureStorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	var backups []*BackupMetadata

	// Build prefix for listing
	prefix := azp.prefix
	if filter.Prefix != "" {
		prefix = azp.prefix + filter.Prefix
	}

	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)

	// List blobs in Azure
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listResponse, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{
			Prefix: prefix,
		})
		if err != nil {
			return nil, NewStorageError("failed to list backups from Azure", err)
		}

		for _, blob := range listResponse.Segment.BlobItems {
			// Only process metadata.json files
			if !strings.HasSuffix(blob.Name, "/metadata.json") {
				continue
			}

			// Extract backup ID from blob name
			backupID := azp.extractBackupIDFromBlobName(blob.Name)
			if backupID == "" {
				continue
			}

			// Load metadata
			metadata, err := azp.GetMetadata(ctx, backupID)
			if err != nil {
				// Log error but continue processing other backups
				continue
			}

			backups = append(backups, metadata)

			// Check if we've reached the maximum number of items
			if filter.MaxItems > 0 && len(backups) >= filter.MaxItems {
				return backups, nil
			}
		}

		marker = listResponse.NextMarker
	}

	return backups, nil
}

// GetMetadata retrieves metadata for a specific backup
func (azp *AzureStorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	blobName := azp.getBackupBlobName(backupID) + "/metadata.json"

	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)
	blobURL := containerURL.NewBlockBlobURL(blobName)

	// Download metadata from Azure
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}

	// Read metadata
	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	defer bodyStream.Close()

	data, err := io.ReadAll(bodyStream)
	if err != nil {
		return nil, NewStorageError("failed to read metadata", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, NewStorageError("failed to unmarshal metadata", err)
	}

	if err := metadata.Validate(); err != nil {
		return nil, NewValidationError("invalid metadata", err)
	}

	return &metadata, nil
}

// Helper methods

// getBackupBlobName returns the Azure blob name for a backup
func (azp *AzureStorageProvider) getBackupBlobName(backupID string) string {
	// Sanitize backup ID to ensure it's safe for Azure blob names
	sanitizedID := azp.sanitizeBackupID(backupID)
	return azp.prefix + sanitizedID
}

// sanitizeBackupID removes potentially dangerous characters from backup ID
func (azp *AzureStorageProvider) sanitizeBackupID(backupID string) string {
	// Azure blob names have specific requirements
	sanitized := strings.ReplaceAll(backupID, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	return sanitized
}

// extractBackupIDFromBlobName extracts the backup ID from an Azure blob name
func (azp *AzureStorageProvider) extractBackupIDFromBlobName(blobName string) string {
	// Remove prefix and suffix to get backup ID
	if !strings.HasPrefix(blobName, azp.prefix) {
		return ""
	}

	// Remove prefix
	withoutPrefix := strings.TrimPrefix(blobName, azp.prefix)

	// Remove "/metadata.json" suffix
	if !strings.HasSuffix(withoutPrefix, "/metadata.json") {
		return ""
	}

	backupID := strings.TrimSuffix(withoutPrefix, "/metadata.json")
	return backupID
}

// GetContainerName returns the Azure container name
func (azp *AzureStorageProvider) GetContainerName() string {
	return azp.containerName
}

// GetPrefix returns the blob prefix used for backups
func (azp *AzureStorageProvider) GetPrefix() string {
	return azp.prefix
}

// SetPrefix sets the blob prefix for backups
func (azp *AzureStorageProvider) SetPrefix(prefix string) {
	azp.prefix = prefix
}

// GetStorageInfo returns information about the storage provider
func (azp *AzureStorageProvider) GetStorageInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":  "azure",
		"container": azp.containerName,
		"prefix":    azp.prefix,
	}
}

// HealthCheck verifies that the storage provider is accessible and functional
func (azp *AzureStorageProvider) HealthCheck(ctx context.Context) error {
	// Get container URL
	containerURL := azp.serviceURL.NewContainerURL(azp.containerName)

	// Check if container exists and is accessible
	_, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if err != nil {
		return NewStorageError("Azure storage provider health check failed: container not accessible", err)
	}

	// Try to list blobs to verify permissions
	_, err = containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{
		Prefix:     azp.prefix,
		MaxResults: 1,
	})
	if err != nil {
		return NewStorageError("Azure storage provider health check failed: cannot list blobs", err)
	}

	return nil
}
