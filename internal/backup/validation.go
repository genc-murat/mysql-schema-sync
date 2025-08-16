package backup

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Validator provides validation utilities for backup system components
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateBackupConfig validates a backup configuration
func (v *Validator) ValidateBackupConfig(config BackupConfig) error {
	var errors ValidationErrors

	// Validate database configuration
	if config.DatabaseConfig.Host == "" {
		errors.Add("database.host", "database host is required", config.DatabaseConfig.Host)
	}
	if config.DatabaseConfig.Database == "" {
		errors.Add("database.database", "database name is required", config.DatabaseConfig.Database)
	}
	if config.DatabaseConfig.Username == "" {
		errors.Add("database.username", "database username is required", config.DatabaseConfig.Username)
	}

	// Validate storage configuration
	if err := v.ValidateStorageConfig(config.StorageConfig); err != nil {
		errors.Add("storage", "invalid storage configuration", err.Error())
	}

	// Validate compression type
	if !v.IsValidCompressionType(config.CompressionType) {
		errors.Add("compression_type", "invalid compression type", config.CompressionType)
	}

	// Validate description length
	if len(config.Description) > 500 {
		errors.Add("description", "description too long (max 500 characters)", len(config.Description))
	}

	// Validate tags
	if err := v.ValidateTags(config.Tags); err != nil {
		errors.Add("tags", "invalid tags", err.Error())
	}

	if errors.HasErrors() {
		return &errors
	}
	return nil
}

// ValidateStorageConfig validates storage configuration
func (v *Validator) ValidateStorageConfig(config StorageConfig) error {
	var errors ValidationErrors

	switch config.Provider {
	case StorageProviderLocal:
		if config.Local == nil {
			errors.Add("local", "local storage configuration is required", nil)
		} else if config.Local.BasePath == "" {
			errors.Add("local.base_path", "base path is required for local storage", config.Local.BasePath)
		}
	case StorageProviderS3:
		if config.S3 == nil {
			errors.Add("s3", "S3 storage configuration is required", nil)
		} else {
			if config.S3.Bucket == "" {
				errors.Add("s3.bucket", "S3 bucket name is required", config.S3.Bucket)
			}
			if config.S3.Region == "" {
				errors.Add("s3.region", "S3 region is required", config.S3.Region)
			}
		}
	case StorageProviderAzure:
		if config.Azure == nil {
			errors.Add("azure", "Azure storage configuration is required", nil)
		} else {
			if config.Azure.AccountName == "" {
				errors.Add("azure.account_name", "Azure account name is required", config.Azure.AccountName)
			}
			if config.Azure.ContainerName == "" {
				errors.Add("azure.container_name", "Azure container name is required", config.Azure.ContainerName)
			}
		}
	case StorageProviderGCS:
		if config.GCS == nil {
			errors.Add("gcs", "GCS storage configuration is required", nil)
		} else {
			if config.GCS.Bucket == "" {
				errors.Add("gcs.bucket", "GCS bucket name is required", config.GCS.Bucket)
			}
			if config.GCS.ProjectID == "" {
				errors.Add("gcs.project_id", "GCS project ID is required", config.GCS.ProjectID)
			}
		}
	default:
		errors.Add("provider", "invalid storage provider", config.Provider)
	}

	if errors.HasErrors() {
		return &errors
	}
	return nil
}

// ValidateBackupMetadata validates backup metadata
func (v *Validator) ValidateBackupMetadata(metadata *BackupMetadata) error {
	var errors ValidationErrors

	if metadata.ID == "" {
		errors.Add("id", "backup ID is required", metadata.ID)
	} else if !v.IsValidBackupID(metadata.ID) {
		errors.Add("id", "invalid backup ID format", metadata.ID)
	}

	if metadata.DatabaseName == "" {
		errors.Add("database_name", "database name is required", metadata.DatabaseName)
	}

	if metadata.CreatedAt.IsZero() {
		errors.Add("created_at", "creation timestamp is required", metadata.CreatedAt)
	} else if metadata.CreatedAt.After(time.Now()) {
		errors.Add("created_at", "creation timestamp cannot be in the future", metadata.CreatedAt)
	}

	if metadata.Size < 0 {
		errors.Add("size", "backup size cannot be negative", metadata.Size)
	}

	if metadata.CompressedSize < 0 {
		errors.Add("compressed_size", "compressed size cannot be negative", metadata.CompressedSize)
	}

	if !v.IsValidCompressionType(metadata.CompressionType) {
		errors.Add("compression_type", "invalid compression type", metadata.CompressionType)
	}

	if !v.IsValidBackupStatus(metadata.Status) {
		errors.Add("status", "invalid backup status", metadata.Status)
	}

	if err := v.ValidateTags(metadata.Tags); err != nil {
		errors.Add("tags", "invalid tags", err.Error())
	}

	if errors.HasErrors() {
		return &errors
	}
	return nil
}

// ValidateBackupFilter validates backup filter parameters
func (v *Validator) ValidateBackupFilter(filter BackupFilter) error {
	var errors ValidationErrors

	if filter.StartDate != nil && filter.EndDate != nil {
		if filter.StartDate.After(*filter.EndDate) {
			errors.Add("date_range", "start date cannot be after end date",
				fmt.Sprintf("start: %v, end: %v", filter.StartDate, filter.EndDate))
		}
	}

	if filter.Status != nil && !v.IsValidBackupStatus(*filter.Status) {
		errors.Add("status", "invalid backup status", *filter.Status)
	}

	if err := v.ValidateTags(filter.Tags); err != nil {
		errors.Add("tags", "invalid tags", err.Error())
	}

	if errors.HasErrors() {
		return &errors
	}
	return nil
}

// ValidateTags validates tag key-value pairs
func (v *Validator) ValidateTags(tags map[string]string) error {
	if len(tags) > 50 {
		return fmt.Errorf("too many tags (max 50, got %d)", len(tags))
	}

	tagKeyRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	for key, value := range tags {
		if len(key) == 0 {
			return fmt.Errorf("tag key cannot be empty")
		}
		if len(key) > 128 {
			return fmt.Errorf("tag key too long (max 128 characters): %s", key)
		}
		if !tagKeyRegex.MatchString(key) {
			return fmt.Errorf("invalid tag key format (only alphanumeric, underscore, and hyphen allowed): %s", key)
		}
		if len(value) > 256 {
			return fmt.Errorf("tag value too long (max 256 characters) for key %s", key)
		}
	}

	return nil
}

// IsValidBackupID checks if a backup ID has valid format
func (v *Validator) IsValidBackupID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	// Backup ID should be alphanumeric with hyphens and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
	return matched
}

// IsValidCompressionType checks if compression type is valid
func (v *Validator) IsValidCompressionType(compressionType CompressionType) bool {
	switch compressionType {
	case CompressionTypeNone, CompressionTypeGzip, CompressionTypeLZ4, CompressionTypeZstd:
		return true
	default:
		return false
	}
}

// IsValidBackupStatus checks if backup status is valid
func (v *Validator) IsValidBackupStatus(status BackupStatus) bool {
	switch status {
	case BackupStatusCreating, BackupStatusCompleted, BackupStatusFailed,
		BackupStatusValidating, BackupStatusCorrupted:
		return true
	default:
		return false
	}
}

// IsValidStorageProvider checks if storage provider is valid
func (v *Validator) IsValidStorageProvider(provider StorageProviderType) bool {
	switch provider {
	case StorageProviderLocal, StorageProviderS3, StorageProviderAzure, StorageProviderGCS:
		return true
	default:
		return false
	}
}

// SanitizeDescription sanitizes backup description
func (v *Validator) SanitizeDescription(description string) string {
	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)

	// Replace multiple consecutive whitespace with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	description = spaceRegex.ReplaceAllString(description, " ")

	// Truncate if too long
	if len(description) > 500 {
		description = description[:497] + "..."
	}

	return description
}

// SanitizeTagValue sanitizes tag values
func (v *Validator) SanitizeTagValue(value string) string {
	// Remove leading/trailing whitespace
	value = strings.TrimSpace(value)

	// Truncate if too long
	if len(value) > 256 {
		value = value[:253] + "..."
	}

	return value
}
