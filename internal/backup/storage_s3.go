package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3StorageProvider implements StorageProvider for Amazon S3 storage
type S3StorageProvider struct {
	client *s3.S3
	bucket string
	prefix string
}

// NewS3StorageProvider creates a new S3StorageProvider instance
func NewS3StorageProvider(config *S3Config) (*S3StorageProvider, error) {
	if config == nil {
		return nil, NewValidationError("S3 storage configuration is required", nil)
	}

	if err := config.Validate(); err != nil {
		return nil, NewValidationError("invalid S3 storage configuration", err)
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			"", // token
		),
	})
	if err != nil {
		return nil, NewStorageError("failed to create AWS session", err)
	}

	provider := &S3StorageProvider{
		client: s3.New(sess),
		bucket: config.Bucket,
		prefix: "backups/", // Default prefix for backup objects
	}

	return provider, nil
}

// Store saves a backup to S3
func (s3p *S3StorageProvider) Store(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return NewValidationError("backup cannot be nil", nil)
	}

	// Set storage location
	objectKey := s3p.getBackupObjectKey(backup.ID)
	backup.Metadata.StorageLocation = fmt.Sprintf("s3://%s/%s", s3p.bucket, objectKey)

	// Calculate checksum after updating storage location
	if err := backup.CalculateChecksum(); err != nil {
		return NewStorageError("failed to calculate backup checksum", err)
	}
	backup.Metadata.Checksum = backup.Checksum

	if err := backup.Validate(); err != nil {
		return NewValidationError("invalid backup data", err)
	}

	// Serialize backup data
	backupData, err := backup.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize backup data", err)
	}

	// Upload backup data to S3
	_, err = s3p.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3p.bucket),
		Key:         aws.String(objectKey + "/backup.json"),
		Body:        bytes.NewReader(backupData),
		ContentType: aws.String("application/json"),
		Metadata: map[string]*string{
			"backup-id":       aws.String(backup.ID),
			"database-name":   aws.String(backup.Metadata.DatabaseName),
			"created-by":      aws.String(backup.Metadata.CreatedBy),
			"compression":     aws.String(string(backup.Metadata.CompressionType)),
			"backup-checksum": aws.String(backup.Checksum),
		},
	})
	if err != nil {
		return NewStorageError("failed to upload backup to S3", err)
	}

	// Upload metadata separately for quick access
	metadataData, err := backup.Metadata.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize metadata", err)
	}

	_, err = s3p.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3p.bucket),
		Key:         aws.String(objectKey + "/metadata.json"),
		Body:        bytes.NewReader(metadataData),
		ContentType: aws.String("application/json"),
		Metadata: map[string]*string{
			"backup-id":     aws.String(backup.ID),
			"database-name": aws.String(backup.Metadata.DatabaseName),
		},
	})
	if err != nil {
		return NewStorageError("failed to upload metadata to S3", err)
	}

	return nil
}

// Retrieve loads a backup from S3
func (s3p *S3StorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	objectKey := s3p.getBackupObjectKey(backupID) + "/backup.json"

	// Download backup data from S3
	result, err := s3p.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("failed to download backup %s from S3", backupID), err)
	}
	defer result.Body.Close()

	// Read backup data
	data, err := io.ReadAll(result.Body)
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

// Delete removes a backup from S3
func (s3p *S3StorageProvider) Delete(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID cannot be empty", nil)
	}

	objectPrefix := s3p.getBackupObjectKey(backupID)

	// List all objects with the backup prefix
	listResult, err := s3p.client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3p.bucket),
		Prefix: aws.String(objectPrefix),
	})
	if err != nil {
		return NewStorageError("failed to list backup objects", err)
	}

	if len(listResult.Contents) == 0 {
		return NewStorageError(fmt.Sprintf("backup %s not found", backupID), nil)
	}

	// Delete all objects for this backup
	var objectsToDelete []*s3.ObjectIdentifier
	for _, obj := range listResult.Contents {
		objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
			Key: obj.Key,
		})
	}

	_, err = s3p.client.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s3p.bucket),
		Delete: &s3.Delete{
			Objects: objectsToDelete,
		},
	})
	if err != nil {
		return NewStorageError("failed to delete backup objects from S3", err)
	}

	return nil
}

// List returns a list of backup metadata matching the filter
func (s3p *S3StorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	var backups []*BackupMetadata

	// Build prefix for listing
	prefix := s3p.prefix
	if filter.Prefix != "" {
		prefix = s3p.prefix + filter.Prefix
	}

	// List objects in S3
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s3p.bucket),
		Prefix: aws.String(prefix),
	}

	if filter.MaxItems > 0 {
		input.MaxKeys = aws.Int64(int64(filter.MaxItems * 2)) // Account for backup.json and metadata.json
	}

	err := s3p.client.ListObjectsV2PagesWithContext(ctx, input,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, obj := range page.Contents {
				// Only process metadata.json files
				if !strings.HasSuffix(*obj.Key, "/metadata.json") {
					continue
				}

				// Extract backup ID from object key
				backupID := s3p.extractBackupIDFromKey(*obj.Key)
				if backupID == "" {
					continue
				}

				// Load metadata
				metadata, err := s3p.GetMetadata(ctx, backupID)
				if err != nil {
					// Log error but continue processing other backups
					continue
				}

				backups = append(backups, metadata)

				// Check if we've reached the maximum number of items
				if filter.MaxItems > 0 && len(backups) >= filter.MaxItems {
					return false // Stop pagination
				}
			}
			return true // Continue pagination
		})

	if err != nil {
		return nil, NewStorageError("failed to list backups from S3", err)
	}

	return backups, nil
}

// GetMetadata retrieves metadata for a specific backup
func (s3p *S3StorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	objectKey := s3p.getBackupObjectKey(backupID) + "/metadata.json"

	// Download metadata from S3
	result, err := s3p.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}
	defer result.Body.Close()

	// Read metadata
	data, err := io.ReadAll(result.Body)
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

// getBackupObjectKey returns the S3 object key for a backup
func (s3p *S3StorageProvider) getBackupObjectKey(backupID string) string {
	// Sanitize backup ID to ensure it's safe for S3 object keys
	sanitizedID := s3p.sanitizeBackupID(backupID)
	return s3p.prefix + sanitizedID
}

// sanitizeBackupID removes potentially dangerous characters from backup ID
func (s3p *S3StorageProvider) sanitizeBackupID(backupID string) string {
	// S3 object keys should not contain certain characters
	sanitized := strings.ReplaceAll(backupID, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	return sanitized
}

// extractBackupIDFromKey extracts the backup ID from an S3 object key
func (s3p *S3StorageProvider) extractBackupIDFromKey(objectKey string) string {
	// Remove prefix and suffix to get backup ID
	if !strings.HasPrefix(objectKey, s3p.prefix) {
		return ""
	}

	// Remove prefix
	withoutPrefix := strings.TrimPrefix(objectKey, s3p.prefix)

	// Remove "/metadata.json" suffix
	if !strings.HasSuffix(withoutPrefix, "/metadata.json") {
		return ""
	}

	backupID := strings.TrimSuffix(withoutPrefix, "/metadata.json")
	return backupID
}

// GetBucket returns the S3 bucket name
func (s3p *S3StorageProvider) GetBucket() string {
	return s3p.bucket
}

// GetPrefix returns the object prefix used for backups
func (s3p *S3StorageProvider) GetPrefix() string {
	return s3p.prefix
}

// SetPrefix sets the object prefix for backups
func (s3p *S3StorageProvider) SetPrefix(prefix string) {
	s3p.prefix = prefix
}

// GetStorageInfo returns information about the storage provider
func (s3p *S3StorageProvider) GetStorageInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider": "s3",
		"bucket":   s3p.bucket,
		"prefix":   s3p.prefix,
	}
}

// HealthCheck verifies that the storage provider is accessible and functional
func (s3p *S3StorageProvider) HealthCheck(ctx context.Context) error {
	// Check if bucket exists and is accessible
	_, err := s3p.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s3p.bucket),
	})
	if err != nil {
		return NewStorageError("S3 storage provider health check failed: bucket not accessible", err)
	}

	// Try to list objects to verify permissions
	_, err = s3p.client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s3p.bucket),
		Prefix:  aws.String(s3p.prefix),
		MaxKeys: aws.Int64(1),
	})
	if err != nil {
		return NewStorageError("S3 storage provider health check failed: cannot list objects", err)
	}

	return nil
}
