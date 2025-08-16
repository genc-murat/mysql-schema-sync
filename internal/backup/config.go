package backup

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// BackupSystemConfig represents the complete backup system configuration
type BackupSystemConfig struct {
	Storage     StorageConfig     `yaml:"storage"`
	Retention   RetentionConfig   `yaml:"retention"`
	Compression CompressionConfig `yaml:"compression"`
	Encryption  EncryptionConfig  `yaml:"encryption"`
	Validation  ValidationConfig  `yaml:"validation"`
}

// RetentionConfig defines backup retention policies
type RetentionConfig struct {
	MaxBackups      int           `yaml:"max_backups"`
	MaxAge          time.Duration `yaml:"max_age"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
	KeepDaily       int           `yaml:"keep_daily"`
	KeepWeekly      int           `yaml:"keep_weekly"`
	KeepMonthly     int           `yaml:"keep_monthly"`
}

// CompressionConfig defines compression settings
type CompressionConfig struct {
	Enabled   bool            `yaml:"enabled"`
	Algorithm CompressionType `yaml:"algorithm"`
	Level     int             `yaml:"level"`
	Threshold int64           `yaml:"threshold"` // Minimum size in bytes to compress
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	Enabled         bool   `yaml:"enabled"`
	KeySource       string `yaml:"key_source"`  // "env", "file", "external"
	KeyPath         string `yaml:"key_path"`    // Path to key file
	KeyEnvVar       string `yaml:"key_env_var"` // Environment variable name
	RotationEnabled bool   `yaml:"rotation_enabled"`
	RotationDays    int    `yaml:"rotation_days"`

	// KeyRetriever is a function that retrieves the encryption key
	// This can be overridden for testing or custom key management
	KeyRetriever func() ([]byte, error) `yaml:"-"`
}

// ValidationConfig defines backup validation settings
type ValidationConfig struct {
	Enabled           bool          `yaml:"enabled"`
	ChecksumAlgorithm string        `yaml:"checksum_algorithm"` // "sha256", "md5"
	ValidateOnCreate  bool          `yaml:"validate_on_create"`
	ValidateOnRestore bool          `yaml:"validate_on_restore"`
	ValidationTimeout time.Duration `yaml:"validation_timeout"`
	DryRunValidation  bool          `yaml:"dry_run_validation"`
}

// Validate validates the BackupSystemConfig
func (bsc *BackupSystemConfig) Validate() error {
	var errors ValidationErrors

	// Validate storage configuration
	if err := bsc.Storage.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("storage", err.Error(), nil)
		}
	}

	// Validate retention configuration
	if err := bsc.Retention.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("retention", err.Error(), nil)
		}
	}

	// Validate compression configuration
	if err := bsc.Compression.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("compression", err.Error(), nil)
		}
	}

	// Validate encryption configuration
	if err := bsc.Encryption.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("encryption", err.Error(), nil)
		}
	}

	// Validate validation configuration
	if err := bsc.Validation.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors.Add("validation", err.Error(), nil)
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// SetDefaults sets default values for the backup system configuration
func (bsc *BackupSystemConfig) SetDefaults() {
	bsc.Storage.SetDefaults()
	bsc.Retention.SetDefaults()
	bsc.Compression.SetDefaults()
	bsc.Encryption.SetDefaults()
	bsc.Validation.SetDefaults()
}

// LoadFromEnvironment loads configuration values from environment variables
func (bsc *BackupSystemConfig) LoadFromEnvironment() {
	bsc.Storage.LoadFromEnvironment()
	bsc.Retention.LoadFromEnvironment()
	bsc.Compression.LoadFromEnvironment()
	bsc.Encryption.LoadFromEnvironment()
	bsc.Validation.LoadFromEnvironment()
}

// Validate validates the RetentionConfig
func (rc *RetentionConfig) Validate() error {
	var errors ValidationErrors

	if rc.MaxBackups < 0 {
		errors.Add("max_backups", "max backups cannot be negative", rc.MaxBackups)
	}

	if rc.MaxAge < 0 {
		errors.Add("max_age", "max age cannot be negative", rc.MaxAge)
	}

	if rc.CleanupInterval < 0 {
		errors.Add("cleanup_interval", "cleanup interval cannot be negative", rc.CleanupInterval)
	}

	if rc.KeepDaily < 0 {
		errors.Add("keep_daily", "keep daily cannot be negative", rc.KeepDaily)
	}

	if rc.KeepWeekly < 0 {
		errors.Add("keep_weekly", "keep weekly cannot be negative", rc.KeepWeekly)
	}

	if rc.KeepMonthly < 0 {
		errors.Add("keep_monthly", "keep monthly cannot be negative", rc.KeepMonthly)
	}

	// Ensure at least one retention policy is set
	if rc.MaxBackups == 0 && rc.MaxAge == 0 && rc.KeepDaily == 0 && rc.KeepWeekly == 0 && rc.KeepMonthly == 0 {
		errors.Add("retention", "at least one retention policy must be configured", nil)
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// SetDefaults sets default values for retention configuration
func (rc *RetentionConfig) SetDefaults() {
	if rc.MaxBackups == 0 && rc.MaxAge == 0 && rc.KeepDaily == 0 && rc.KeepWeekly == 0 && rc.KeepMonthly == 0 {
		rc.MaxBackups = 10 // Default to keeping 10 backups
	}

	if rc.CleanupInterval == 0 {
		rc.CleanupInterval = 24 * time.Hour // Default cleanup every 24 hours
	}
}

// LoadFromEnvironment loads retention configuration from environment variables
func (rc *RetentionConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_MAX_BACKUPS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.MaxBackups = parsed
		}
	}

	if val := os.Getenv("BACKUP_MAX_AGE"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			rc.MaxAge = parsed
		}
	}

	if val := os.Getenv("BACKUP_CLEANUP_INTERVAL"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			rc.CleanupInterval = parsed
		}
	}

	if val := os.Getenv("BACKUP_KEEP_DAILY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepDaily = parsed
		}
	}

	if val := os.Getenv("BACKUP_KEEP_WEEKLY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepWeekly = parsed
		}
	}

	if val := os.Getenv("BACKUP_KEEP_MONTHLY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepMonthly = parsed
		}
	}
}

// Validate validates the CompressionConfig
func (cc *CompressionConfig) Validate() error {
	var errors ValidationErrors

	if cc.Enabled {
		if !isValidCompressionType(cc.Algorithm) {
			errors.Add("algorithm", "invalid compression algorithm", cc.Algorithm)
		}

		// Validate compression level based on algorithm
		switch cc.Algorithm {
		case CompressionTypeGzip:
			if cc.Level < 1 || cc.Level > 9 {
				errors.Add("level", "gzip compression level must be between 1 and 9", cc.Level)
			}
		case CompressionTypeLZ4:
			if cc.Level < 1 || cc.Level > 12 {
				errors.Add("level", "lz4 compression level must be between 1 and 12", cc.Level)
			}
		case CompressionTypeZstd:
			if cc.Level < 1 || cc.Level > 22 {
				errors.Add("level", "zstd compression level must be between 1 and 22", cc.Level)
			}
		}

		if cc.Threshold < 0 {
			errors.Add("threshold", "compression threshold cannot be negative", cc.Threshold)
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// SetDefaults sets default values for compression configuration
func (cc *CompressionConfig) SetDefaults() {
	if cc.Enabled && cc.Algorithm == "" {
		cc.Algorithm = CompressionTypeGzip
	}

	if cc.Enabled && cc.Level == 0 {
		switch cc.Algorithm {
		case CompressionTypeGzip:
			cc.Level = 6 // Default gzip level
		case CompressionTypeLZ4:
			cc.Level = 1 // Default lz4 level (fast)
		case CompressionTypeZstd:
			cc.Level = 3 // Default zstd level
		}
	}

	if cc.Threshold == 0 {
		cc.Threshold = 1024 // Default threshold: 1KB
	}
}

// LoadFromEnvironment loads compression configuration from environment variables
func (cc *CompressionConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_COMPRESSION_ENABLED"); val != "" {
		cc.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_COMPRESSION_ALGORITHM"); val != "" {
		cc.Algorithm = CompressionType(strings.ToUpper(val))
	}

	if val := os.Getenv("BACKUP_COMPRESSION_LEVEL"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cc.Level = parsed
		}
	}

	if val := os.Getenv("BACKUP_COMPRESSION_THRESHOLD"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			cc.Threshold = parsed
		}
	}
}

// Validate validates the EncryptionConfig
func (ec *EncryptionConfig) Validate() error {
	var errors ValidationErrors

	if ec.Enabled {
		if ec.KeySource == "" {
			errors.Add("key_source", "key source is required when encryption is enabled", ec.KeySource)
		} else {
			switch ec.KeySource {
			case "env":
				if ec.KeyEnvVar == "" {
					errors.Add("key_env_var", "key environment variable name is required for env key source", ec.KeyEnvVar)
				}
			case "file":
				if ec.KeyPath == "" {
					errors.Add("key_path", "key file path is required for file key source", ec.KeyPath)
				}
			case "external":
				// External key management validation would be implementation-specific
			default:
				errors.Add("key_source", "invalid key source, must be 'env', 'file', or 'external'", ec.KeySource)
			}
		}

		if ec.RotationEnabled && ec.RotationDays <= 0 {
			errors.Add("rotation_days", "rotation days must be positive when rotation is enabled", ec.RotationDays)
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// SetDefaults sets default values for encryption configuration
func (ec *EncryptionConfig) SetDefaults() {
	if ec.Enabled && ec.KeySource == "" {
		ec.KeySource = "env"
		ec.KeyEnvVar = "BACKUP_ENCRYPTION_KEY"
	}

	if ec.RotationEnabled && ec.RotationDays == 0 {
		ec.RotationDays = 90 // Default rotation every 90 days
	}
}

// LoadFromEnvironment loads encryption configuration from environment variables
func (ec *EncryptionConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_ENCRYPTION_ENABLED"); val != "" {
		ec.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_ENCRYPTION_KEY_SOURCE"); val != "" {
		ec.KeySource = val
	}

	if val := os.Getenv("BACKUP_ENCRYPTION_KEY_PATH"); val != "" {
		ec.KeyPath = val
	}

	if val := os.Getenv("BACKUP_ENCRYPTION_KEY_ENV_VAR"); val != "" {
		ec.KeyEnvVar = val
	}

	if val := os.Getenv("BACKUP_ENCRYPTION_ROTATION_ENABLED"); val != "" {
		ec.RotationEnabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_ENCRYPTION_ROTATION_DAYS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			ec.RotationDays = parsed
		}
	}
}

// Validate validates the ValidationConfig
func (vc *ValidationConfig) Validate() error {
	var errors ValidationErrors

	if vc.Enabled {
		if vc.ChecksumAlgorithm == "" {
			vc.ChecksumAlgorithm = "sha256" // Set default
		} else {
			switch vc.ChecksumAlgorithm {
			case "sha256", "md5":
				// Valid algorithms
			default:
				errors.Add("checksum_algorithm", "invalid checksum algorithm, must be 'sha256' or 'md5'", vc.ChecksumAlgorithm)
			}
		}

		if vc.ValidationTimeout < 0 {
			errors.Add("validation_timeout", "validation timeout cannot be negative", vc.ValidationTimeout)
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// SetDefaults sets default values for validation configuration
func (vc *ValidationConfig) SetDefaults() {
	if vc.ChecksumAlgorithm == "" {
		vc.ChecksumAlgorithm = "sha256"
	}

	if vc.ValidationTimeout == 0 {
		vc.ValidationTimeout = 5 * time.Minute // Default validation timeout
	}

	// Enable validation by default
	if !vc.Enabled {
		vc.Enabled = true
		vc.ValidateOnCreate = true
		vc.ValidateOnRestore = true
		vc.DryRunValidation = true
	}
}

// LoadFromEnvironment loads validation configuration from environment variables
func (vc *ValidationConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_VALIDATION_ENABLED"); val != "" {
		vc.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_VALIDATION_CHECKSUM_ALGORITHM"); val != "" {
		vc.ChecksumAlgorithm = val
	}

	if val := os.Getenv("BACKUP_VALIDATION_ON_CREATE"); val != "" {
		vc.ValidateOnCreate = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_VALIDATION_ON_RESTORE"); val != "" {
		vc.ValidateOnRestore = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("BACKUP_VALIDATION_TIMEOUT"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			vc.ValidationTimeout = parsed
		}
	}

	if val := os.Getenv("BACKUP_VALIDATION_DRY_RUN"); val != "" {
		vc.DryRunValidation = strings.ToLower(val) == "true"
	}
}

// SetDefaults sets default values for storage configuration
func (sc *StorageConfig) SetDefaults() {
	if sc.Provider == "" {
		sc.Provider = StorageProviderLocal
	}

	switch sc.Provider {
	case StorageProviderLocal:
		if sc.Local == nil {
			sc.Local = &LocalConfig{}
		}
		sc.Local.SetDefaults()
	case StorageProviderS3:
		if sc.S3 == nil {
			sc.S3 = &S3Config{}
		}
		sc.S3.SetDefaults()
	case StorageProviderAzure:
		if sc.Azure == nil {
			sc.Azure = &AzureConfig{}
		}
		sc.Azure.SetDefaults()
	case StorageProviderGCS:
		if sc.GCS == nil {
			sc.GCS = &GCSConfig{}
		}
		sc.GCS.SetDefaults()
	}
}

// LoadFromEnvironment loads storage configuration from environment variables
func (sc *StorageConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_STORAGE_PROVIDER"); val != "" {
		sc.Provider = StorageProviderType(strings.ToUpper(val))
	}

	switch sc.Provider {
	case StorageProviderLocal:
		if sc.Local == nil {
			sc.Local = &LocalConfig{}
		}
		sc.Local.LoadFromEnvironment()
	case StorageProviderS3:
		if sc.S3 == nil {
			sc.S3 = &S3Config{}
		}
		sc.S3.LoadFromEnvironment()
	case StorageProviderAzure:
		if sc.Azure == nil {
			sc.Azure = &AzureConfig{}
		}
		sc.Azure.LoadFromEnvironment()
	case StorageProviderGCS:
		if sc.GCS == nil {
			sc.GCS = &GCSConfig{}
		}
		sc.GCS.LoadFromEnvironment()
	}
}

// SetDefaults sets default values for local storage configuration
func (lc *LocalConfig) SetDefaults() {
	if lc.BasePath == "" {
		lc.BasePath = "./backups"
	}

	if lc.Permissions == 0 {
		lc.Permissions = 0755
	}
}

// LoadFromEnvironment loads local storage configuration from environment variables
func (lc *LocalConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_LOCAL_BASE_PATH"); val != "" {
		lc.BasePath = val
	}

	if val := os.Getenv("BACKUP_LOCAL_PERMISSIONS"); val != "" {
		if parsed, err := strconv.ParseUint(val, 8, 32); err == nil {
			lc.Permissions = os.FileMode(parsed)
		}
	}
}

// SetDefaults sets default values for S3 storage configuration
func (s3c *S3Config) SetDefaults() {
	if s3c.Region == "" {
		s3c.Region = "us-east-1"
	}
}

// LoadFromEnvironment loads S3 storage configuration from environment variables
func (s3c *S3Config) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_S3_BUCKET"); val != "" {
		s3c.Bucket = val
	}

	if val := os.Getenv("BACKUP_S3_REGION"); val != "" {
		s3c.Region = val
	}

	if val := os.Getenv("BACKUP_S3_ACCESS_KEY"); val != "" {
		s3c.AccessKey = val
	}

	if val := os.Getenv("BACKUP_S3_SECRET_KEY"); val != "" {
		s3c.SecretKey = val
	}
}

// SetDefaults sets default values for Azure storage configuration
func (ac *AzureConfig) SetDefaults() {
	// Azure doesn't have meaningful defaults beyond what's required
}

// LoadFromEnvironment loads Azure storage configuration from environment variables
func (ac *AzureConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_AZURE_ACCOUNT_NAME"); val != "" {
		ac.AccountName = val
	}

	if val := os.Getenv("BACKUP_AZURE_ACCOUNT_KEY"); val != "" {
		ac.AccountKey = val
	}

	if val := os.Getenv("BACKUP_AZURE_CONTAINER_NAME"); val != "" {
		ac.ContainerName = val
	}
}

// SetDefaults sets default values for GCS storage configuration
func (gc *GCSConfig) SetDefaults() {
	if gc.CredentialsPath == "" {
		gc.CredentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
}

// LoadFromEnvironment loads GCS storage configuration from environment variables
func (gc *GCSConfig) LoadFromEnvironment() {
	if val := os.Getenv("BACKUP_GCS_BUCKET"); val != "" {
		gc.Bucket = val
	}

	if val := os.Getenv("BACKUP_GCS_CREDENTIALS_PATH"); val != "" {
		gc.CredentialsPath = val
	}

	if val := os.Getenv("BACKUP_GCS_PROJECT_ID"); val != "" {
		gc.ProjectID = val
	}
}

// GetEncryptionKey retrieves the encryption key based on the configuration
func (ec *EncryptionConfig) GetEncryptionKey() ([]byte, error) {
	if !ec.Enabled {
		return nil, nil
	}

	// Use custom function if provided (for testing or custom key management)
	if ec.KeyRetriever != nil {
		return ec.KeyRetriever()
	}

	switch ec.KeySource {
	case "env":
		keyStr := os.Getenv(ec.KeyEnvVar)
		if keyStr == "" {
			return nil, fmt.Errorf("encryption key not found in environment variable %s", ec.KeyEnvVar)
		}
		// Expect key to be hex-encoded
		key, err := hex.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex key from environment variable: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d bytes", len(key))
		}
		return key, nil

	case "file":
		keyData, err := os.ReadFile(ec.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption key from file %s: %w", ec.KeyPath, err)
		}
		if len(keyData) != 32 {
			return nil, fmt.Errorf("encryption key file must contain 32 bytes for AES-256, got %d bytes", len(keyData))
		}
		return keyData, nil

	case "external":
		// External key management would be implemented based on specific requirements
		return nil, fmt.Errorf("external key management not implemented")

	default:
		return nil, fmt.Errorf("invalid key source: %s", ec.KeySource)
	}
}
