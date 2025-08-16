package backup

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Validate validates the Backup struct
func (b *Backup) Validate() error {
	var errors ValidationErrors

	if b.ID == "" {
		errors.Add("id", "backup ID is required", b.ID)
	}

	if b.Metadata == nil {
		errors.Add("metadata", "backup metadata is required", nil)
	} else {
		if err := b.Metadata.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("metadata", err.Error(), nil)
			}
		}
	}

	if b.SchemaSnapshot == nil {
		errors.Add("schema_snapshot", "schema snapshot is required", nil)
	}

	if b.Checksum == "" {
		errors.Add("checksum", "backup checksum is required", b.Checksum)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ToJSON serializes the Backup to JSON
func (b *Backup) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// FromJSON deserializes JSON data into a Backup
func (b *Backup) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, b); err != nil {
		return NewValidationError("failed to unmarshal backup JSON", err)
	}
	return b.Validate()
}

// CalculateChecksum calculates and sets the checksum for the backup
func (b *Backup) CalculateChecksum() error {
	// Create a copy without the checksum field to avoid circular dependency
	temp := *b
	temp.Checksum = ""

	// Also create a copy of metadata to avoid circular dependency
	if temp.Metadata != nil {
		metadataCopy := *temp.Metadata
		metadataCopy.Checksum = ""
		temp.Metadata = &metadataCopy
	}

	data, err := json.Marshal(temp)
	if err != nil {
		return NewValidationError("failed to marshal backup for checksum calculation", err)
	}

	hash := sha256.Sum256(data)
	b.Checksum = hex.EncodeToString(hash[:])
	return nil
}

// VerifyChecksum verifies the backup's checksum
func (b *Backup) VerifyChecksum() bool {
	originalChecksum := b.Checksum
	if err := b.CalculateChecksum(); err != nil {
		return false
	}

	calculatedChecksum := b.Checksum
	b.Checksum = originalChecksum

	return originalChecksum == calculatedChecksum
}

// Validate validates the BackupMetadata struct
func (bm *BackupMetadata) Validate() error {
	var errors ValidationErrors

	if bm.ID == "" {
		errors.Add("id", "backup metadata ID is required", bm.ID)
	}

	if bm.DatabaseName == "" {
		errors.Add("database_name", "database name is required", bm.DatabaseName)
	}

	if bm.CreatedAt.IsZero() {
		errors.Add("created_at", "creation timestamp is required", bm.CreatedAt)
	}

	if bm.CreatedBy == "" {
		errors.Add("created_by", "creator information is required", bm.CreatedBy)
	}

	if bm.Size < 0 {
		errors.Add("size", "backup size cannot be negative", bm.Size)
	}

	if bm.CompressedSize < 0 {
		errors.Add("compressed_size", "compressed size cannot be negative", bm.CompressedSize)
	}

	if bm.CompressionType != "" {
		if !isValidCompressionType(bm.CompressionType) {
			errors.Add("compression_type", "invalid compression type", bm.CompressionType)
		}
	}

	if bm.Status == "" {
		errors.Add("status", "backup status is required", bm.Status)
	} else if !isValidBackupStatus(bm.Status) {
		errors.Add("status", "invalid backup status", bm.Status)
	}

	if bm.StorageLocation == "" {
		errors.Add("storage_location", "storage location is required", bm.StorageLocation)
	}

	if bm.Checksum == "" {
		errors.Add("checksum", "backup checksum is required", bm.Checksum)
	}

	// Validate migration context if present
	if bm.MigrationContext != nil {
		if err := bm.MigrationContext.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("migration_context", err.Error(), nil)
			}
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ToJSON serializes the BackupMetadata to JSON
func (bm *BackupMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(bm, "", "  ")
}

// FromJSON deserializes JSON data into BackupMetadata
func (bm *BackupMetadata) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, bm); err != nil {
		return NewValidationError("failed to unmarshal backup metadata JSON", err)
	}
	return bm.Validate()
}

// Validate validates the MigrationContext struct
func (mc *MigrationContext) Validate() error {
	var errors ValidationErrors

	if mc.PlanHash == "" {
		errors.Add("plan_hash", "migration plan hash is required", mc.PlanHash)
	}

	if mc.SourceSchema == "" {
		errors.Add("source_schema", "source schema is required", mc.SourceSchema)
	}

	if mc.PreMigrationID == "" {
		errors.Add("pre_migration_id", "pre-migration ID is required", mc.PreMigrationID)
	}

	if mc.MigrationTime.IsZero() {
		errors.Add("migration_time", "migration timestamp is required", mc.MigrationTime)
	}

	if mc.ToolVersion == "" {
		errors.Add("tool_version", "tool version is required", mc.ToolVersion)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ToJSON serializes the MigrationContext to JSON
func (mc *MigrationContext) ToJSON() ([]byte, error) {
	return json.MarshalIndent(mc, "", "  ")
}

// FromJSON deserializes JSON data into MigrationContext
func (mc *MigrationContext) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, mc); err != nil {
		return NewValidationError("failed to unmarshal migration context JSON", err)
	}
	return mc.Validate()
}

// GenerateBackupID generates a unique backup ID
func GenerateBackupID() string {
	// Use UUID v4 for uniqueness with timestamp prefix for sorting
	timestamp := time.Now().UTC().Format("20060102-150405")
	uuid := uuid.New().String()

	// Remove hyphens from UUID and take first 8 characters for brevity
	shortUUID := strings.ReplaceAll(uuid, "-", "")[:8]

	return fmt.Sprintf("backup-%s-%s", timestamp, shortUUID)
}

// GenerateBackupIDWithPrefix generates a backup ID with a custom prefix
func GenerateBackupIDWithPrefix(prefix string) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	uuid := uuid.New().String()
	shortUUID := strings.ReplaceAll(uuid, "-", "")[:8]

	return fmt.Sprintf("%s-%s-%s", prefix, timestamp, shortUUID)
}

// CalculateDataChecksum calculates a checksum for arbitrary data
func CalculateDataChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CalculateMD5Checksum calculates an MD5 checksum for data (for compatibility)
func CalculateMD5Checksum(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// GenerateSecureRandomBytes generates cryptographically secure random bytes
func GenerateSecureRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, NewEncryptionError("failed to generate secure random bytes", err)
	}
	return bytes, nil
}

// Helper functions for validation

func isValidCompressionType(ct CompressionType) bool {
	switch ct {
	case CompressionTypeNone, CompressionTypeGzip, CompressionTypeLZ4, CompressionTypeZstd:
		return true
	default:
		return false
	}
}

func isValidBackupStatus(status BackupStatus) bool {
	switch status {
	case BackupStatusCreating, BackupStatusCompleted, BackupStatusFailed,
		BackupStatusValidating, BackupStatusCorrupted:
		return true
	default:
		return false
	}
}

func isValidStorageProviderType(provider StorageProviderType) bool {
	switch provider {
	case StorageProviderLocal, StorageProviderS3, StorageProviderAzure, StorageProviderGCS:
		return true
	default:
		return false
	}
}

// Validate validates the BackupConfig struct
func (bc *BackupConfig) Validate() error {
	var errors ValidationErrors

	// Validate storage config
	if err := bc.StorageConfig.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("storage_config", err.Error(), nil)
		}
	}

	// Validate compression type
	if bc.CompressionType != "" && !isValidCompressionType(bc.CompressionType) {
		errors.Add("compression_type", "invalid compression type", bc.CompressionType)
	}

	// Validate encryption key if provided
	if len(bc.EncryptionKey) > 0 && len(bc.EncryptionKey) != 32 {
		errors.Add("encryption_key", "encryption key must be 32 bytes for AES-256", len(bc.EncryptionKey))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Validate validates the StorageConfig struct
func (sc *StorageConfig) Validate() error {
	var errors ValidationErrors

	if !isValidStorageProviderType(sc.Provider) {
		errors.Add("provider", "invalid storage provider type", sc.Provider)
		return errors
	}

	switch sc.Provider {
	case StorageProviderLocal:
		if sc.Local == nil {
			errors.Add("local", "local storage configuration is required", nil)
		} else if err := sc.Local.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("local", err.Error(), nil)
			}
		}
	case StorageProviderS3:
		if sc.S3 == nil {
			errors.Add("s3", "S3 storage configuration is required", nil)
		} else if err := sc.S3.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("s3", err.Error(), nil)
			}
		}
	case StorageProviderAzure:
		if sc.Azure == nil {
			errors.Add("azure", "Azure storage configuration is required", nil)
		} else if err := sc.Azure.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("azure", err.Error(), nil)
			}
		}
	case StorageProviderGCS:
		if sc.GCS == nil {
			errors.Add("gcs", "GCS storage configuration is required", nil)
		} else if err := sc.GCS.Validate(); err != nil {
			if validationErrs, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErrs...)
			} else {
				errors.Add("gcs", err.Error(), nil)
			}
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Validate validates the LocalConfig struct
func (lc *LocalConfig) Validate() error {
	var errors ValidationErrors

	if lc.BasePath == "" {
		errors.Add("base_path", "base path is required for local storage", lc.BasePath)
	}

	if lc.Permissions == 0 {
		lc.Permissions = 0755 // Set default permissions
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Validate validates the S3Config struct
func (s3c *S3Config) Validate() error {
	var errors ValidationErrors

	if s3c.Bucket == "" {
		errors.Add("bucket", "S3 bucket name is required", s3c.Bucket)
	}

	if s3c.Region == "" {
		errors.Add("region", "S3 region is required", s3c.Region)
	}

	if s3c.AccessKey == "" {
		errors.Add("access_key", "S3 access key is required", s3c.AccessKey)
	}

	if s3c.SecretKey == "" {
		errors.Add("secret_key", "S3 secret key is required", s3c.SecretKey)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Validate validates the AzureConfig struct
func (ac *AzureConfig) Validate() error {
	var errors ValidationErrors

	if ac.AccountName == "" {
		errors.Add("account_name", "Azure account name is required", ac.AccountName)
	}

	if ac.AccountKey == "" {
		errors.Add("account_key", "Azure account key is required", ac.AccountKey)
	}

	if ac.ContainerName == "" {
		errors.Add("container_name", "Azure container name is required", ac.ContainerName)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Validate validates the GCSConfig struct
func (gc *GCSConfig) Validate() error {
	var errors ValidationErrors

	if gc.Bucket == "" {
		errors.Add("bucket", "GCS bucket name is required", gc.Bucket)
	}

	if gc.CredentialsPath == "" {
		errors.Add("credentials_path", "GCS credentials path is required", gc.CredentialsPath)
	}

	if gc.ProjectID == "" {
		errors.Add("project_id", "GCS project ID is required", gc.ProjectID)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
