package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSStorageProvider implements StorageProvider for Google Cloud Storage
type GCSStorageProvider struct {
	client     *storage.Client
	bucketName string
	prefix     string
}

// NewGCSStorageProvider creates a new GCSStorageProvider instance
func NewGCSStorageProvider(ctx context.Context, config *GCSConfig) (*GCSStorageProvider, error) {
	if config == nil {
		return nil, NewValidationError("GCS storage configuration is required", nil)
	}

	if err := config.Validate(); err != nil {
		return nil, NewValidationError("invalid GCS storage configuration", err)
	}

	// Create GCS client
	var client *storage.Client
	var err error

	if config.CredentialsPath != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(config.CredentialsPath))
	} else {
		// Use default credentials (e.g., from environment or metadata server)
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, NewStorageError("failed to create GCS client", err)
	}

	provider := &GCSStorageProvider{
		client:     client,
		bucketName: config.Bucket,
		prefix:     "backups/", // Default prefix for backup objects
	}

	return provider, nil
}

// Store saves a backup to Google Cloud Storage
func (gcsp *GCSStorageProvider) Store(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return NewValidationError("backup cannot be nil", nil)
	}

	// Set storage location
	objectName := gcsp.getBackupObjectName(backup.ID)
	backup.Metadata.StorageLocation = fmt.Sprintf("gs://%s/%s", gcsp.bucketName, objectName)

	// Calculate checksum after updating storage location
	if err := backup.CalculateChecksum(); err != nil {
		return NewStorageError("failed to calculate backup checksum", err)
	}
	backup.Metadata.Checksum = backup.Checksum

	if err := backup.Validate(); err != nil {
		return NewValidationError("invalid backup data", err)
	}

	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)

	// Serialize backup data
	backupData, err := backup.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize backup data", err)
	}

	// Upload backup data to GCS
	backupObject := bucket.Object(objectName + "/backup.json")
	backupWriter := backupObject.NewWriter(ctx)
	backupWriter.ContentType = "application/json"
	backupWriter.Metadata = map[string]string{
		"backup-id":       backup.ID,
		"database-name":   backup.Metadata.DatabaseName,
		"created-by":      backup.Metadata.CreatedBy,
		"compression":     string(backup.Metadata.CompressionType),
		"backup-checksum": backup.Checksum,
	}

	if _, err := backupWriter.Write(backupData); err != nil {
		backupWriter.Close()
		return NewStorageError("failed to write backup data to GCS", err)
	}

	if err := backupWriter.Close(); err != nil {
		return NewStorageError("failed to upload backup to GCS", err)
	}

	// Upload metadata separately for quick access
	metadataData, err := backup.Metadata.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize metadata", err)
	}

	metadataObject := bucket.Object(objectName + "/metadata.json")
	metadataWriter := metadataObject.NewWriter(ctx)
	metadataWriter.ContentType = "application/json"
	metadataWriter.Metadata = map[string]string{
		"backup-id":     backup.ID,
		"database-name": backup.Metadata.DatabaseName,
	}

	if _, err := metadataWriter.Write(metadataData); err != nil {
		metadataWriter.Close()
		return NewStorageError("failed to write metadata to GCS", err)
	}

	if err := metadataWriter.Close(); err != nil {
		return NewStorageError("failed to upload metadata to GCS", err)
	}

	return nil
}

// Retrieve loads a backup from Google Cloud Storage
func (gcsp *GCSStorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	objectName := gcsp.getBackupObjectName(backupID) + "/backup.json"

	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)
	object := bucket.Object(objectName)

	// Download backup data from GCS
	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("failed to download backup %s from GCS", backupID), err)
	}
	defer reader.Close()

	// Read backup data
	data, err := io.ReadAll(reader)
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

// Delete removes a backup from Google Cloud Storage
func (gcsp *GCSStorageProvider) Delete(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID cannot be empty", nil)
	}

	objectPrefix := gcsp.getBackupObjectName(backupID)

	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)

	// List all objects with the backup prefix
	query := &storage.Query{Prefix: objectPrefix}
	it := bucket.Objects(ctx, query)

	var objectsToDelete []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return NewStorageError("failed to list backup objects", err)
		}
		objectsToDelete = append(objectsToDelete, attrs.Name)
	}

	if len(objectsToDelete) == 0 {
		return NewStorageError(fmt.Sprintf("backup %s not found", backupID), nil)
	}

	// Delete all objects for this backup
	for _, objectName := range objectsToDelete {
		object := bucket.Object(objectName)
		if err := object.Delete(ctx); err != nil {
			return NewStorageError(fmt.Sprintf("failed to delete object %s", objectName), err)
		}
	}

	return nil
}

// List returns a list of backup metadata matching the filter
func (gcsp *GCSStorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	var backups []*BackupMetadata

	// Build prefix for listing
	prefix := gcsp.prefix
	if filter.Prefix != "" {
		prefix = gcsp.prefix + filter.Prefix
	}

	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)

	// List objects in GCS
	query := &storage.Query{Prefix: prefix}
	it := bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, NewStorageError("failed to list backups from GCS", err)
		}

		// Only process metadata.json files
		if !strings.HasSuffix(attrs.Name, "/metadata.json") {
			continue
		}

		// Extract backup ID from object name
		backupID := gcsp.extractBackupIDFromObjectName(attrs.Name)
		if backupID == "" {
			continue
		}

		// Load metadata
		metadata, err := gcsp.GetMetadata(ctx, backupID)
		if err != nil {
			// Log error but continue processing other backups
			continue
		}

		backups = append(backups, metadata)

		// Check if we've reached the maximum number of items
		if filter.MaxItems > 0 && len(backups) >= filter.MaxItems {
			break
		}
	}

	return backups, nil
}

// GetMetadata retrieves metadata for a specific backup
func (gcsp *GCSStorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	objectName := gcsp.getBackupObjectName(backupID) + "/metadata.json"

	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)
	object := bucket.Object(objectName)

	// Download metadata from GCS
	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}
	defer reader.Close()

	// Read metadata
	data, err := io.ReadAll(reader)
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

// getBackupObjectName returns the GCS object name for a backup
func (gcsp *GCSStorageProvider) getBackupObjectName(backupID string) string {
	// Sanitize backup ID to ensure it's safe for GCS object names
	sanitizedID := gcsp.sanitizeBackupID(backupID)
	return gcsp.prefix + sanitizedID
}

// sanitizeBackupID removes potentially dangerous characters from backup ID
func (gcsp *GCSStorageProvider) sanitizeBackupID(backupID string) string {
	// GCS object names have specific requirements
	sanitized := strings.ReplaceAll(backupID, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	return sanitized
}

// extractBackupIDFromObjectName extracts the backup ID from a GCS object name
func (gcsp *GCSStorageProvider) extractBackupIDFromObjectName(objectName string) string {
	// Remove prefix and suffix to get backup ID
	if !strings.HasPrefix(objectName, gcsp.prefix) {
		return ""
	}

	// Remove prefix
	withoutPrefix := strings.TrimPrefix(objectName, gcsp.prefix)

	// Remove "/metadata.json" suffix
	if !strings.HasSuffix(withoutPrefix, "/metadata.json") {
		return ""
	}

	backupID := strings.TrimSuffix(withoutPrefix, "/metadata.json")
	return backupID
}

// GetBucketName returns the GCS bucket name
func (gcsp *GCSStorageProvider) GetBucketName() string {
	return gcsp.bucketName
}

// GetPrefix returns the object prefix used for backups
func (gcsp *GCSStorageProvider) GetPrefix() string {
	return gcsp.prefix
}

// SetPrefix sets the object prefix for backups
func (gcsp *GCSStorageProvider) SetPrefix(prefix string) {
	gcsp.prefix = prefix
}

// GetStorageInfo returns information about the storage provider
func (gcsp *GCSStorageProvider) GetStorageInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider": "gcs",
		"bucket":   gcsp.bucketName,
		"prefix":   gcsp.prefix,
	}
}

// HealthCheck verifies that the storage provider is accessible and functional
func (gcsp *GCSStorageProvider) HealthCheck(ctx context.Context) error {
	// Get bucket handle
	bucket := gcsp.client.Bucket(gcsp.bucketName)

	// Check if bucket exists and is accessible
	_, err := bucket.Attrs(ctx)
	if err != nil {
		return NewStorageError("GCS storage provider health check failed: bucket not accessible", err)
	}

	// Try to list objects to verify permissions
	query := &storage.Query{
		Prefix: gcsp.prefix,
	}
	it := bucket.Objects(ctx, query)
	_, err = it.Next()
	if err != nil && err != iterator.Done {
		return NewStorageError("GCS storage provider health check failed: cannot list objects", err)
	}

	return nil
}

// Close closes the GCS client
func (gcsp *GCSStorageProvider) Close() error {
	return gcsp.client.Close()
}
