package database

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"mysql-schema-sync/internal/backup"
	"mysql-schema-sync/internal/config"
)

// ConfigConverter converts between database config types and backup config types
type ConfigConverter struct{}

// NewConfigConverter creates a new configuration converter
func NewConfigConverter() *ConfigConverter {
	return &ConfigConverter{}
}

// ToBackupSystemConfig converts config.BackupConfig to backup.BackupSystemConfig
func (cc *ConfigConverter) ToBackupSystemConfig(dbConfig *config.BackupConfig) (*backup.BackupSystemConfig, error) {
	if dbConfig == nil {
		return nil, fmt.Errorf("database backup config is nil")
	}

	backupConfig := &backup.BackupSystemConfig{}

	// Convert storage configuration
	storageConfig, err := cc.convertStorageConfig(&dbConfig.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert storage config: %w", err)
	}
	backupConfig.Storage = *storageConfig

	// Convert retention configuration
	retentionConfig, err := cc.convertRetentionConfig(&dbConfig.Retention)
	if err != nil {
		return nil, fmt.Errorf("failed to convert retention config: %w", err)
	}
	backupConfig.Retention = *retentionConfig

	// Convert compression configuration
	compressionConfig, err := cc.convertCompressionConfig(&dbConfig.Compression)
	if err != nil {
		return nil, fmt.Errorf("failed to convert compression config: %w", err)
	}
	backupConfig.Compression = *compressionConfig

	// Convert encryption configuration
	encryptionConfig, err := cc.convertEncryptionConfig(&dbConfig.Encryption)
	if err != nil {
		return nil, fmt.Errorf("failed to convert encryption config: %w", err)
	}
	backupConfig.Encryption = *encryptionConfig

	// Convert validation configuration
	validationConfig, err := cc.convertValidationConfig(&dbConfig.Validation)
	if err != nil {
		return nil, fmt.Errorf("failed to convert validation config: %w", err)
	}
	backupConfig.Validation = *validationConfig

	return backupConfig, nil
}

// convertStorageConfig converts config storage config to backup storage config
func (cc *ConfigConverter) convertStorageConfig(dbStorage *config.StorageConfig) (*backup.StorageConfig, error) {
	backupStorage := &backup.StorageConfig{}

	// Convert provider type
	switch strings.ToLower(dbStorage.Provider) {
	case "local":
		backupStorage.Provider = backup.StorageProviderLocal
		if dbStorage.Local != nil {
			permissions, err := cc.parseFileMode(dbStorage.Local.Permissions)
			if err != nil {
				return nil, fmt.Errorf("invalid permissions: %w", err)
			}
			backupStorage.Local = &backup.LocalConfig{
				BasePath:    dbStorage.Local.BasePath,
				Permissions: permissions,
			}
		}
	case "s3":
		backupStorage.Provider = backup.StorageProviderS3
		if dbStorage.S3 != nil {
			backupStorage.S3 = &backup.S3Config{
				Bucket:    dbStorage.S3.Bucket,
				Region:    dbStorage.S3.Region,
				AccessKey: dbStorage.S3.AccessKey,
				SecretKey: dbStorage.S3.SecretKey,
			}
		}
	case "azure":
		backupStorage.Provider = backup.StorageProviderAzure
		if dbStorage.Azure != nil {
			backupStorage.Azure = &backup.AzureConfig{
				AccountName:   dbStorage.Azure.AccountName,
				AccountKey:    dbStorage.Azure.AccountKey,
				ContainerName: dbStorage.Azure.ContainerName,
			}
		}
	case "gcs":
		backupStorage.Provider = backup.StorageProviderGCS
		if dbStorage.GCS != nil {
			backupStorage.GCS = &backup.GCSConfig{
				Bucket:          dbStorage.GCS.Bucket,
				CredentialsPath: dbStorage.GCS.CredentialsPath,
				ProjectID:       dbStorage.GCS.ProjectID,
			}
		}
	default:
		return nil, fmt.Errorf("invalid storage provider: %s", dbStorage.Provider)
	}

	return backupStorage, nil
}

// convertRetentionConfig converts config retention config to backup retention config
func (cc *ConfigConverter) convertRetentionConfig(dbRetention *config.RetentionConfig) (*backup.RetentionConfig, error) {
	backupRetention := &backup.RetentionConfig{
		MaxBackups:  dbRetention.MaxBackups,
		KeepDaily:   dbRetention.KeepDaily,
		KeepWeekly:  dbRetention.KeepWeekly,
		KeepMonthly: dbRetention.KeepMonthly,
	}

	// Parse duration strings
	if dbRetention.MaxAge != "" {
		maxAge, err := time.ParseDuration(dbRetention.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("invalid max age duration: %w", err)
		}
		backupRetention.MaxAge = maxAge
	}

	if dbRetention.CleanupInterval != "" {
		cleanupInterval, err := time.ParseDuration(dbRetention.CleanupInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid cleanup interval duration: %w", err)
		}
		backupRetention.CleanupInterval = cleanupInterval
	}

	return backupRetention, nil
}

// convertCompressionConfig converts config compression config to backup compression config
func (cc *ConfigConverter) convertCompressionConfig(dbCompression *config.CompressionConfig) (*backup.CompressionConfig, error) {
	backupCompression := &backup.CompressionConfig{
		Enabled:   dbCompression.Enabled,
		Level:     dbCompression.Level,
		Threshold: dbCompression.Threshold,
	}

	// Convert algorithm string to backup compression type
	switch strings.ToLower(dbCompression.Algorithm) {
	case "gzip":
		backupCompression.Algorithm = backup.CompressionTypeGzip
	case "lz4":
		backupCompression.Algorithm = backup.CompressionTypeLZ4
	case "zstd":
		backupCompression.Algorithm = backup.CompressionTypeZstd
	case "none", "":
		backupCompression.Algorithm = backup.CompressionTypeNone
	default:
		return nil, fmt.Errorf("invalid compression algorithm: %s", dbCompression.Algorithm)
	}

	return backupCompression, nil
}

// convertEncryptionConfig converts config encryption config to backup encryption config
func (cc *ConfigConverter) convertEncryptionConfig(dbEncryption *config.EncryptionConfig) (*backup.EncryptionConfig, error) {
	backupEncryption := &backup.EncryptionConfig{
		Enabled:         dbEncryption.Enabled,
		KeySource:       dbEncryption.KeySource,
		KeyPath:         dbEncryption.KeyPath,
		KeyEnvVar:       dbEncryption.KeyEnvVar,
		RotationEnabled: dbEncryption.RotationEnabled,
		RotationDays:    dbEncryption.RotationDays,
	}

	return backupEncryption, nil
}

// convertValidationConfig converts config validation config to backup validation config
func (cc *ConfigConverter) convertValidationConfig(dbValidation *config.ValidationConfig) (*backup.ValidationConfig, error) {
	backupValidation := &backup.ValidationConfig{
		Enabled:           dbValidation.Enabled,
		ChecksumAlgorithm: dbValidation.ChecksumAlgorithm,
		ValidateOnCreate:  dbValidation.ValidateOnCreate,
		ValidateOnRestore: dbValidation.ValidateOnRestore,
		DryRunValidation:  dbValidation.DryRunValidation,
	}

	// Parse validation timeout
	if dbValidation.ValidationTimeout != "" {
		timeout, err := time.ParseDuration(dbValidation.ValidationTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid validation timeout duration: %w", err)
		}
		backupValidation.ValidationTimeout = timeout
	}

	return backupValidation, nil
}

// parseFileMode parses a string file mode to os.FileMode
func (cc *ConfigConverter) parseFileMode(modeStr string) (os.FileMode, error) {
	if modeStr == "" {
		return 0755, nil // Default permissions
	}

	// Remove quotes if present
	modeStr = strings.Trim(modeStr, "\"'")

	// Parse as octal
	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode: %s", modeStr)
	}

	return os.FileMode(mode), nil
}

// FromBackupSystemConfig converts backup.BackupSystemConfig to config.BackupConfig
func (cc *ConfigConverter) FromBackupSystemConfig(backupConfig *backup.BackupSystemConfig) (*config.BackupConfig, error) {
	if backupConfig == nil {
		return nil, fmt.Errorf("backup system config is nil")
	}

	dbConfig := &config.BackupConfig{
		Enabled: true, // If we have a backup config, assume it's enabled
	}

	// Convert storage configuration
	storageConfig, err := cc.convertBackupStorageConfig(&backupConfig.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert storage config: %w", err)
	}
	dbConfig.Storage = *storageConfig

	// Convert retention configuration
	retentionConfig, err := cc.convertBackupRetentionConfig(&backupConfig.Retention)
	if err != nil {
		return nil, fmt.Errorf("failed to convert retention config: %w", err)
	}
	dbConfig.Retention = *retentionConfig

	// Convert compression configuration
	compressionConfig, err := cc.convertBackupCompressionConfig(&backupConfig.Compression)
	if err != nil {
		return nil, fmt.Errorf("failed to convert compression config: %w", err)
	}
	dbConfig.Compression = *compressionConfig

	// Convert encryption configuration
	encryptionConfig, err := cc.convertBackupEncryptionConfig(&backupConfig.Encryption)
	if err != nil {
		return nil, fmt.Errorf("failed to convert encryption config: %w", err)
	}
	dbConfig.Encryption = *encryptionConfig

	// Convert validation configuration
	validationConfig, err := cc.convertBackupValidationConfig(&backupConfig.Validation)
	if err != nil {
		return nil, fmt.Errorf("failed to convert validation config: %w", err)
	}
	dbConfig.Validation = *validationConfig

	return dbConfig, nil
}

// convertBackupStorageConfig converts backup storage config to config storage config
func (cc *ConfigConverter) convertBackupStorageConfig(backupStorage *backup.StorageConfig) (*config.StorageConfig, error) {
	dbStorage := &config.StorageConfig{}

	// Convert provider type
	switch backupStorage.Provider {
	case backup.StorageProviderLocal:
		dbStorage.Provider = "local"
		if backupStorage.Local != nil {
			dbStorage.Local = &config.LocalConfig{
				BasePath:    backupStorage.Local.BasePath,
				Permissions: fmt.Sprintf("%o", backupStorage.Local.Permissions),
			}
		}
	case backup.StorageProviderS3:
		dbStorage.Provider = "s3"
		if backupStorage.S3 != nil {
			dbStorage.S3 = &config.S3Config{
				Bucket:    backupStorage.S3.Bucket,
				Region:    backupStorage.S3.Region,
				AccessKey: backupStorage.S3.AccessKey,
				SecretKey: backupStorage.S3.SecretKey,
			}
		}
	case backup.StorageProviderAzure:
		dbStorage.Provider = "azure"
		if backupStorage.Azure != nil {
			dbStorage.Azure = &config.AzureConfig{
				AccountName:   backupStorage.Azure.AccountName,
				AccountKey:    backupStorage.Azure.AccountKey,
				ContainerName: backupStorage.Azure.ContainerName,
			}
		}
	case backup.StorageProviderGCS:
		dbStorage.Provider = "gcs"
		if backupStorage.GCS != nil {
			dbStorage.GCS = &config.GCSConfig{
				Bucket:          backupStorage.GCS.Bucket,
				CredentialsPath: backupStorage.GCS.CredentialsPath,
				ProjectID:       backupStorage.GCS.ProjectID,
			}
		}
	default:
		return nil, fmt.Errorf("invalid backup storage provider: %s", backupStorage.Provider)
	}

	return dbStorage, nil
}

// convertBackupRetentionConfig converts backup retention config to config retention config
func (cc *ConfigConverter) convertBackupRetentionConfig(backupRetention *backup.RetentionConfig) (*config.RetentionConfig, error) {
	dbRetention := &config.RetentionConfig{
		MaxBackups:  backupRetention.MaxBackups,
		KeepDaily:   backupRetention.KeepDaily,
		KeepWeekly:  backupRetention.KeepWeekly,
		KeepMonthly: backupRetention.KeepMonthly,
	}

	// Convert durations to strings
	if backupRetention.MaxAge > 0 {
		dbRetention.MaxAge = backupRetention.MaxAge.String()
	}

	if backupRetention.CleanupInterval > 0 {
		dbRetention.CleanupInterval = backupRetention.CleanupInterval.String()
	}

	return dbRetention, nil
}

// convertBackupCompressionConfig converts backup compression config to config compression config
func (cc *ConfigConverter) convertBackupCompressionConfig(backupCompression *backup.CompressionConfig) (*config.CompressionConfig, error) {
	dbCompression := &config.CompressionConfig{
		Enabled:   backupCompression.Enabled,
		Level:     backupCompression.Level,
		Threshold: backupCompression.Threshold,
	}

	// Convert algorithm type to string
	switch backupCompression.Algorithm {
	case backup.CompressionTypeGzip:
		dbCompression.Algorithm = "gzip"
	case backup.CompressionTypeLZ4:
		dbCompression.Algorithm = "lz4"
	case backup.CompressionTypeZstd:
		dbCompression.Algorithm = "zstd"
	case backup.CompressionTypeNone:
		dbCompression.Algorithm = "none"
	default:
		return nil, fmt.Errorf("invalid backup compression algorithm: %s", backupCompression.Algorithm)
	}

	return dbCompression, nil
}

// convertBackupEncryptionConfig converts backup encryption config to config encryption config
func (cc *ConfigConverter) convertBackupEncryptionConfig(backupEncryption *backup.EncryptionConfig) (*config.EncryptionConfig, error) {
	dbEncryption := &config.EncryptionConfig{
		Enabled:         backupEncryption.Enabled,
		KeySource:       backupEncryption.KeySource,
		KeyPath:         backupEncryption.KeyPath,
		KeyEnvVar:       backupEncryption.KeyEnvVar,
		RotationEnabled: backupEncryption.RotationEnabled,
		RotationDays:    backupEncryption.RotationDays,
	}

	return dbEncryption, nil
}

// convertBackupValidationConfig converts backup validation config to config validation config
func (cc *ConfigConverter) convertBackupValidationConfig(backupValidation *backup.ValidationConfig) (*config.ValidationConfig, error) {
	dbValidation := &config.ValidationConfig{
		Enabled:           backupValidation.Enabled,
		ChecksumAlgorithm: backupValidation.ChecksumAlgorithm,
		ValidateOnCreate:  backupValidation.ValidateOnCreate,
		ValidateOnRestore: backupValidation.ValidateOnRestore,
		DryRunValidation:  backupValidation.DryRunValidation,
	}

	// Convert validation timeout to string
	if backupValidation.ValidationTimeout > 0 {
		dbValidation.ValidationTimeout = backupValidation.ValidationTimeout.String()
	}

	return dbValidation, nil
}
