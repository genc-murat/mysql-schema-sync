package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// BackupConfig holds backup system configuration
type BackupConfig struct {
	Enabled     bool              `mapstructure:"enabled" yaml:"enabled"`
	Storage     StorageConfig     `mapstructure:"storage" yaml:"storage"`
	Retention   RetentionConfig   `mapstructure:"retention" yaml:"retention"`
	Compression CompressionConfig `mapstructure:"compression" yaml:"compression"`
	Encryption  EncryptionConfig  `mapstructure:"encryption" yaml:"encryption"`
	Validation  ValidationConfig  `mapstructure:"validation" yaml:"validation"`
}

// StorageConfig defines storage provider configuration
type StorageConfig struct {
	Provider string       `mapstructure:"provider" yaml:"provider"`
	Local    *LocalConfig `mapstructure:"local,omitempty" yaml:"local,omitempty"`
	S3       *S3Config    `mapstructure:"s3,omitempty" yaml:"s3,omitempty"`
	Azure    *AzureConfig `mapstructure:"azure,omitempty" yaml:"azure,omitempty"`
	GCS      *GCSConfig   `mapstructure:"gcs,omitempty" yaml:"gcs,omitempty"`
}

// LocalConfig for local file system storage
type LocalConfig struct {
	BasePath    string `mapstructure:"base_path" yaml:"base_path"`
	Permissions string `mapstructure:"permissions" yaml:"permissions"`
}

// S3Config for Amazon S3 storage
type S3Config struct {
	Bucket    string `mapstructure:"bucket" yaml:"bucket"`
	Region    string `mapstructure:"region" yaml:"region"`
	AccessKey string `mapstructure:"access_key" yaml:"access_key"`
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
}

// AzureConfig for Azure Blob Storage
type AzureConfig struct {
	AccountName   string `mapstructure:"account_name" yaml:"account_name"`
	AccountKey    string `mapstructure:"account_key" yaml:"account_key"`
	ContainerName string `mapstructure:"container_name" yaml:"container_name"`
}

// GCSConfig for Google Cloud Storage
type GCSConfig struct {
	Bucket          string `mapstructure:"bucket" yaml:"bucket"`
	CredentialsPath string `mapstructure:"credentials_path" yaml:"credentials_path"`
	ProjectID       string `mapstructure:"project_id" yaml:"project_id"`
}

// RetentionConfig defines backup retention policies
type RetentionConfig struct {
	MaxBackups      int    `mapstructure:"max_backups" yaml:"max_backups"`
	MaxAge          string `mapstructure:"max_age" yaml:"max_age"`
	CleanupInterval string `mapstructure:"cleanup_interval" yaml:"cleanup_interval"`
	KeepDaily       int    `mapstructure:"keep_daily" yaml:"keep_daily"`
	KeepWeekly      int    `mapstructure:"keep_weekly" yaml:"keep_weekly"`
	KeepMonthly     int    `mapstructure:"keep_monthly" yaml:"keep_monthly"`
}

// CompressionConfig defines compression settings
type CompressionConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Algorithm string `mapstructure:"algorithm" yaml:"algorithm"`
	Level     int    `mapstructure:"level" yaml:"level"`
	Threshold int64  `mapstructure:"threshold" yaml:"threshold"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	Enabled         bool   `mapstructure:"enabled" yaml:"enabled"`
	KeySource       string `mapstructure:"key_source" yaml:"key_source"`
	KeyPath         string `mapstructure:"key_path" yaml:"key_path"`
	KeyEnvVar       string `mapstructure:"key_env_var" yaml:"key_env_var"`
	RotationEnabled bool   `mapstructure:"rotation_enabled" yaml:"rotation_enabled"`
	RotationDays    int    `mapstructure:"rotation_days" yaml:"rotation_days"`
}

// ValidationConfig defines backup validation settings
type ValidationConfig struct {
	Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
	ChecksumAlgorithm string `mapstructure:"checksum_algorithm" yaml:"checksum_algorithm"`
	ValidateOnCreate  bool   `mapstructure:"validate_on_create" yaml:"validate_on_create"`
	ValidateOnRestore bool   `mapstructure:"validate_on_restore" yaml:"validate_on_restore"`
	ValidationTimeout string `mapstructure:"validation_timeout" yaml:"validation_timeout"`
	DryRunValidation  bool   `mapstructure:"dry_run_validation" yaml:"dry_run_validation"`
}

// Validate validates the backup configuration
func (bc *BackupConfig) Validate() error {
	if !bc.Enabled {
		return nil // Skip validation if backup is disabled
	}

	var errs []error

	// Validate storage configuration
	if err := bc.Storage.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("storage: %w", err))
	}

	// Validate retention configuration
	if err := bc.Retention.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("retention: %w", err))
	}

	// Validate compression configuration
	if err := bc.Compression.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("compression: %w", err))
	}

	// Validate encryption configuration
	if err := bc.Encryption.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("encryption: %w", err))
	}

	// Validate validation configuration
	if err := bc.Validation.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("validation: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("backup configuration validation failed: %v", errs)
	}

	return nil
}

// SetDefaults sets default values for backup configuration
func (bc *BackupConfig) SetDefaults() {
	// Backup is disabled by default
	if !bc.Enabled {
		return
	}

	bc.Storage.SetDefaults()
	bc.Retention.SetDefaults()
	bc.Compression.SetDefaults()
	bc.Encryption.SetDefaults()
	bc.Validation.SetDefaults()
}

// LoadFromEnvironment loads backup configuration from environment variables
func (bc *BackupConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENABLED"); val != "" {
		bc.Enabled = strings.ToLower(val) == "true"
	}

	if bc.Enabled {
		bc.Storage.LoadFromEnvironment()
		bc.Retention.LoadFromEnvironment()
		bc.Compression.LoadFromEnvironment()
		bc.Encryption.LoadFromEnvironment()
		bc.Validation.LoadFromEnvironment()
	}
}

// Validate validates the storage configuration
func (sc *StorageConfig) Validate() error {
	if sc.Provider == "" {
		return errors.New("storage provider is required")
	}

	switch sc.Provider {
	case "local":
		if sc.Local == nil {
			return errors.New("local storage configuration is required when provider is 'local'")
		}
		return sc.Local.Validate()
	case "s3":
		if sc.S3 == nil {
			return errors.New("S3 storage configuration is required when provider is 's3'")
		}
		return sc.S3.Validate()
	case "azure":
		if sc.Azure == nil {
			return errors.New("Azure storage configuration is required when provider is 'azure'")
		}
		return sc.Azure.Validate()
	case "gcs":
		if sc.GCS == nil {
			return errors.New("GCS storage configuration is required when provider is 'gcs'")
		}
		return sc.GCS.Validate()
	default:
		return fmt.Errorf("invalid storage provider: %s", sc.Provider)
	}
}

// SetDefaults sets default values for storage configuration
func (sc *StorageConfig) SetDefaults() {
	if sc.Provider == "" {
		sc.Provider = "local"
	}

	switch sc.Provider {
	case "local":
		if sc.Local == nil {
			sc.Local = &LocalConfig{}
		}
		sc.Local.SetDefaults()
	case "s3":
		if sc.S3 == nil {
			sc.S3 = &S3Config{}
		}
		sc.S3.SetDefaults()
	case "azure":
		if sc.Azure == nil {
			sc.Azure = &AzureConfig{}
		}
		sc.Azure.SetDefaults()
	case "gcs":
		if sc.GCS == nil {
			sc.GCS = &GCSConfig{}
		}
		sc.GCS.SetDefaults()
	}
}

// LoadFromEnvironment loads storage configuration from environment variables
func (sc *StorageConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER"); val != "" {
		sc.Provider = strings.ToLower(val)
	}

	switch sc.Provider {
	case "local":
		if sc.Local == nil {
			sc.Local = &LocalConfig{}
		}
		sc.Local.LoadFromEnvironment()
	case "s3":
		if sc.S3 == nil {
			sc.S3 = &S3Config{}
		}
		sc.S3.LoadFromEnvironment()
	case "azure":
		if sc.Azure == nil {
			sc.Azure = &AzureConfig{}
		}
		sc.Azure.LoadFromEnvironment()
	case "gcs":
		if sc.GCS == nil {
			sc.GCS = &GCSConfig{}
		}
		sc.GCS.LoadFromEnvironment()
	}
}

// Validate validates the local storage configuration
func (lc *LocalConfig) Validate() error {
	if lc.BasePath == "" {
		return errors.New("base path is required for local storage")
	}
	return nil
}

// SetDefaults sets default values for local storage configuration
func (lc *LocalConfig) SetDefaults() {
	if lc.BasePath == "" {
		lc.BasePath = "./backups"
	}
	if lc.Permissions == "" {
		lc.Permissions = "0755"
	}
}

// LoadFromEnvironment loads local storage configuration from environment variables
func (lc *LocalConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_BASE_PATH"); val != "" {
		lc.BasePath = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_PERMISSIONS"); val != "" {
		lc.Permissions = val
	}
}

// Validate validates the S3 storage configuration
func (s3c *S3Config) Validate() error {
	if s3c.Bucket == "" {
		return errors.New("bucket is required for S3 storage")
	}
	if s3c.Region == "" {
		return errors.New("region is required for S3 storage")
	}
	return nil
}

// SetDefaults sets default values for S3 storage configuration
func (s3c *S3Config) SetDefaults() {
	if s3c.Region == "" {
		s3c.Region = "us-east-1"
	}
}

// LoadFromEnvironment loads S3 storage configuration from environment variables
func (s3c *S3Config) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET"); val != "" {
		s3c.Bucket = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_S3_REGION"); val != "" {
		s3c.Region = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_S3_ACCESS_KEY"); val != "" {
		s3c.AccessKey = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_S3_SECRET_KEY"); val != "" {
		s3c.SecretKey = val
	}
}

// Validate validates the Azure storage configuration
func (ac *AzureConfig) Validate() error {
	if ac.AccountName == "" {
		return errors.New("account name is required for Azure storage")
	}
	if ac.AccountKey == "" {
		return errors.New("account key is required for Azure storage")
	}
	if ac.ContainerName == "" {
		return errors.New("container name is required for Azure storage")
	}
	return nil
}

// SetDefaults sets default values for Azure storage configuration
func (ac *AzureConfig) SetDefaults() {
	// Azure doesn't have meaningful defaults beyond what's required
}

// LoadFromEnvironment loads Azure storage configuration from environment variables
func (ac *AzureConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_NAME"); val != "" {
		ac.AccountName = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_KEY"); val != "" {
		ac.AccountKey = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_AZURE_CONTAINER_NAME"); val != "" {
		ac.ContainerName = val
	}
}

// Validate validates the GCS storage configuration
func (gc *GCSConfig) Validate() error {
	if gc.Bucket == "" {
		return errors.New("bucket is required for GCS storage")
	}
	return nil
}

// SetDefaults sets default values for GCS storage configuration
func (gc *GCSConfig) SetDefaults() {
	if gc.CredentialsPath == "" {
		gc.CredentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
}

// LoadFromEnvironment loads GCS storage configuration from environment variables
func (gc *GCSConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_GCS_BUCKET"); val != "" {
		gc.Bucket = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_GCS_CREDENTIALS_PATH"); val != "" {
		gc.CredentialsPath = val
	}
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_GCS_PROJECT_ID"); val != "" {
		gc.ProjectID = val
	}
}

// Validate validates the retention configuration
func (rc *RetentionConfig) Validate() error {
	var errs []error

	if rc.MaxBackups < 0 {
		errs = append(errs, errors.New("max backups cannot be negative"))
	}

	if rc.MaxAge != "" {
		if _, err := time.ParseDuration(rc.MaxAge); err != nil {
			errs = append(errs, fmt.Errorf("invalid max age duration: %w", err))
		}
	}

	if rc.CleanupInterval != "" {
		if _, err := time.ParseDuration(rc.CleanupInterval); err != nil {
			errs = append(errs, fmt.Errorf("invalid cleanup interval duration: %w", err))
		}
	}

	if rc.KeepDaily < 0 {
		errs = append(errs, errors.New("keep daily cannot be negative"))
	}

	if rc.KeepWeekly < 0 {
		errs = append(errs, errors.New("keep weekly cannot be negative"))
	}

	if rc.KeepMonthly < 0 {
		errs = append(errs, errors.New("keep monthly cannot be negative"))
	}

	// Ensure at least one retention policy is set
	if rc.MaxBackups == 0 && rc.MaxAge == "" && rc.KeepDaily == 0 && rc.KeepWeekly == 0 && rc.KeepMonthly == 0 {
		errs = append(errs, errors.New("at least one retention policy must be configured"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("retention configuration validation failed: %v", errs)
	}

	return nil
}

// SetDefaults sets default values for retention configuration
func (rc *RetentionConfig) SetDefaults() {
	if rc.MaxBackups == 0 && rc.MaxAge == "" && rc.KeepDaily == 0 && rc.KeepWeekly == 0 && rc.KeepMonthly == 0 {
		rc.MaxBackups = 10 // Default to keeping 10 backups
	}

	if rc.CleanupInterval == "" {
		rc.CleanupInterval = "24h" // Default cleanup every 24 hours
	}
}

// LoadFromEnvironment loads retention configuration from environment variables
func (rc *RetentionConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.MaxBackups = parsed
		}
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_MAX_AGE"); val != "" {
		rc.MaxAge = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_CLEANUP_INTERVAL"); val != "" {
		rc.CleanupInterval = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_KEEP_DAILY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepDaily = parsed
		}
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_KEEP_WEEKLY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepWeekly = parsed
		}
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_KEEP_MONTHLY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			rc.KeepMonthly = parsed
		}
	}
}

// Validate validates the compression configuration
func (cc *CompressionConfig) Validate() error {
	if cc.Enabled {
		if cc.Algorithm == "" {
			return errors.New("compression algorithm is required when compression is enabled")
		}

		switch strings.ToLower(cc.Algorithm) {
		case "gzip":
			if cc.Level < 1 || cc.Level > 9 {
				return errors.New("gzip compression level must be between 1 and 9")
			}
		case "lz4":
			if cc.Level < 1 || cc.Level > 12 {
				return errors.New("lz4 compression level must be between 1 and 12")
			}
		case "zstd":
			if cc.Level < 1 || cc.Level > 22 {
				return errors.New("zstd compression level must be between 1 and 22")
			}
		default:
			return fmt.Errorf("invalid compression algorithm: %s", cc.Algorithm)
		}

		if cc.Threshold < 0 {
			return errors.New("compression threshold cannot be negative")
		}
	}

	return nil
}

// SetDefaults sets default values for compression configuration
func (cc *CompressionConfig) SetDefaults() {
	if cc.Enabled && cc.Algorithm == "" {
		cc.Algorithm = "gzip"
	}

	if cc.Enabled && cc.Level == 0 {
		switch strings.ToLower(cc.Algorithm) {
		case "gzip":
			cc.Level = 6 // Default gzip level
		case "lz4":
			cc.Level = 1 // Default lz4 level (fast)
		case "zstd":
			cc.Level = 3 // Default zstd level
		}
	}

	if cc.Threshold == 0 {
		cc.Threshold = 1024 // Default threshold: 1KB
	}
}

// LoadFromEnvironment loads compression configuration from environment variables
func (cc *CompressionConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED"); val != "" {
		cc.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ALGORITHM"); val != "" {
		cc.Algorithm = strings.ToLower(val)
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_LEVEL"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			cc.Level = parsed
		}
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_THRESHOLD"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			cc.Threshold = parsed
		}
	}
}

// Validate validates the encryption configuration
func (ec *EncryptionConfig) Validate() error {
	if ec.Enabled {
		if ec.KeySource == "" {
			return errors.New("key source is required when encryption is enabled")
		}

		switch ec.KeySource {
		case "env":
			if ec.KeyEnvVar == "" {
				return errors.New("key environment variable name is required for env key source")
			}
		case "file":
			if ec.KeyPath == "" {
				return errors.New("key file path is required for file key source")
			}
		case "external":
			// External key management validation would be implementation-specific
		default:
			return fmt.Errorf("invalid key source: %s", ec.KeySource)
		}

		if ec.RotationEnabled && ec.RotationDays <= 0 {
			return errors.New("rotation days must be positive when rotation is enabled")
		}
	}

	return nil
}

// SetDefaults sets default values for encryption configuration
func (ec *EncryptionConfig) SetDefaults() {
	if ec.Enabled && ec.KeySource == "" {
		ec.KeySource = "env"
		ec.KeyEnvVar = "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY"
	}

	if ec.RotationEnabled && ec.RotationDays == 0 {
		ec.RotationDays = 90 // Default rotation every 90 days
	}
}

// LoadFromEnvironment loads encryption configuration from environment variables
func (ec *EncryptionConfig) LoadFromEnvironment() {
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED"); val != "" {
		ec.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_SOURCE"); val != "" {
		ec.KeySource = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_PATH"); val != "" {
		ec.KeyPath = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_ENV_VAR"); val != "" {
		ec.KeyEnvVar = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_ENABLED"); val != "" {
		ec.RotationEnabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_DAYS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			ec.RotationDays = parsed
		}
	}
}

// Validate validates the validation configuration
func (vc *ValidationConfig) Validate() error {
	if vc.Enabled {
		if vc.ChecksumAlgorithm == "" {
			vc.ChecksumAlgorithm = "sha256" // Set default
		} else {
			switch vc.ChecksumAlgorithm {
			case "sha256", "md5":
				// Valid algorithms
			default:
				return fmt.Errorf("invalid checksum algorithm: %s", vc.ChecksumAlgorithm)
			}
		}

		if vc.ValidationTimeout != "" {
			if _, err := time.ParseDuration(vc.ValidationTimeout); err != nil {
				return fmt.Errorf("invalid validation timeout duration: %w", err)
			}
		}
	}

	return nil
}

// SetDefaults sets default values for validation configuration
func (vc *ValidationConfig) SetDefaults() {
	if vc.ChecksumAlgorithm == "" {
		vc.ChecksumAlgorithm = "sha256"
	}

	if vc.ValidationTimeout == "" {
		vc.ValidationTimeout = "5m" // Default validation timeout
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
	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ENABLED"); val != "" {
		vc.Enabled = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_CHECKSUM_ALGORITHM"); val != "" {
		vc.ChecksumAlgorithm = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_CREATE"); val != "" {
		vc.ValidateOnCreate = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_RESTORE"); val != "" {
		vc.ValidateOnRestore = strings.ToLower(val) == "true"
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_TIMEOUT"); val != "" {
		vc.ValidationTimeout = val
	}

	if val := os.Getenv("MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_DRY_RUN"); val != "" {
		vc.DryRunValidation = strings.ToLower(val) == "true"
	}
}
