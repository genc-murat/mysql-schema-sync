package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigLoader handles loading and parsing backup configuration
type ConfigLoader struct {
	configPath string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// LoadConfig loads the backup configuration from file and environment variables
func (cl *ConfigLoader) LoadConfig() (*BackupSystemConfig, error) {
	config := &BackupSystemConfig{}

	// Set defaults first
	config.SetDefaults()

	// Load from file if it exists
	if cl.configPath != "" {
		if err := cl.loadFromFile(config); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	config.LoadFromEnvironment()

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file
func (cl *ConfigLoader) loadFromFile(config *BackupSystemConfig) error {
	// Check if file exists
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		// File doesn't exist, use defaults
		return nil
	}

	// Read the file
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", cl.configPath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// SaveConfig saves the backup configuration to a YAML file
func (cl *ConfigLoader) SaveConfig(config *BackupSystemConfig) error {
	// Validate configuration before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cl.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cl.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfigFromBytes loads configuration from YAML bytes
func LoadConfigFromBytes(data []byte) (*BackupSystemConfig, error) {
	config := &BackupSystemConfig{}
	config.SetDefaults()

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Load environment variables
	config.LoadFromEnvironment()

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// GenerateDefaultConfig generates a default configuration with comments
func GenerateDefaultConfig() *BackupSystemConfig {
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "./backups",
				Permissions: 0755,
			},
		},
		Retention: RetentionConfig{
			MaxBackups:      10,
			MaxAge:          0, // Disabled
			CleanupInterval: 24 * time.Hour,
			KeepDaily:       7,
			KeepWeekly:      4,
			KeepMonthly:     12,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: CompressionTypeGzip,
			Level:     6,
			Threshold: 1024, // 1KB
		},
		Encryption: EncryptionConfig{
			Enabled:         false,
			KeySource:       "env",
			KeyEnvVar:       "BACKUP_ENCRYPTION_KEY",
			RotationEnabled: false,
			RotationDays:    90,
		},
		Validation: ValidationConfig{
			Enabled:           true,
			ChecksumAlgorithm: "sha256",
			ValidateOnCreate:  true,
			ValidateOnRestore: true,
			ValidationTimeout: 5 * time.Minute,
			DryRunValidation:  true,
		},
	}

	return config
}

// GenerateDefaultConfigYAML generates a default configuration as YAML with comments
func GenerateDefaultConfigYAML() ([]byte, error) {
	configYAML := `# Backup System Configuration
# This file configures the backup and rollback system for mysql-schema-sync

# Storage configuration
storage:
  # Storage provider: LOCAL, S3, AZURE, GCS
  provider: LOCAL
  
  # Local storage configuration (when provider is LOCAL)
  local:
    base_path: "./backups"
    permissions: 0755
  
  # S3 storage configuration (when provider is S3)
  # s3:
  #   bucket: "my-backup-bucket"
  #   region: "us-east-1"
  #   access_key: "your-access-key"
  #   secret_key: "your-secret-key"
  
  # Azure storage configuration (when provider is AZURE)
  # azure:
  #   account_name: "your-account-name"
  #   account_key: "your-account-key"
  #   container_name: "backups"
  
  # Google Cloud Storage configuration (when provider is GCS)
  # gcs:
  #   bucket: "my-backup-bucket"
  #   credentials_path: "/path/to/credentials.json"
  #   project_id: "your-project-id"

# Retention policies
retention:
  # Maximum number of backups to keep (0 = unlimited)
  max_backups: 10
  
  # Maximum age of backups (0 = unlimited)
  # Examples: "720h" (30 days), "168h" (7 days)
  max_age: 0s
  
  # How often to run cleanup (default: 24h)
  cleanup_interval: 24h
  
  # Advanced retention: keep specific numbers by time period
  keep_daily: 7     # Keep 7 daily backups
  keep_weekly: 4    # Keep 4 weekly backups
  keep_monthly: 12  # Keep 12 monthly backups

# Compression settings
compression:
  # Enable compression
  enabled: true
  
  # Compression algorithm: GZIP, LZ4, ZSTD
  algorithm: GZIP
  
  # Compression level (1-9 for GZIP, 1-12 for LZ4, 1-22 for ZSTD)
  level: 6
  
  # Minimum size in bytes to compress (smaller files won't be compressed)
  threshold: 1024

# Encryption settings
encryption:
  # Enable encryption
  enabled: false
  
  # Key source: env, file, external
  key_source: env
  
  # Environment variable name for encryption key (when key_source is env)
  key_env_var: BACKUP_ENCRYPTION_KEY
  
  # Path to key file (when key_source is file)
  # key_path: "/path/to/encryption.key"
  
  # Enable key rotation
  rotation_enabled: false
  
  # Days between key rotations
  rotation_days: 90

# Validation settings
validation:
  # Enable backup validation
  enabled: true
  
  # Checksum algorithm: sha256, md5
  checksum_algorithm: sha256
  
  # Validate backups when created
  validate_on_create: true
  
  # Validate backups before restore
  validate_on_restore: true
  
  # Timeout for validation operations
  validation_timeout: 5m
  
  # Enable dry-run validation (test restore without applying)
  dry_run_validation: true
`

	return []byte(configYAML), nil
}

// MergeConfigs merges two configurations, with the second one taking precedence
func MergeConfigs(base, override *BackupSystemConfig) *BackupSystemConfig {
	merged := *base

	// Merge storage config
	if override.Storage.Provider != "" {
		merged.Storage.Provider = override.Storage.Provider
	}
	if override.Storage.Local != nil {
		if merged.Storage.Local == nil {
			merged.Storage.Local = &LocalConfig{}
		}
		if override.Storage.Local.BasePath != "" {
			merged.Storage.Local.BasePath = override.Storage.Local.BasePath
		}
		if override.Storage.Local.Permissions != 0 {
			merged.Storage.Local.Permissions = override.Storage.Local.Permissions
		}
	}
	if override.Storage.S3 != nil {
		merged.Storage.S3 = override.Storage.S3
	}
	if override.Storage.Azure != nil {
		merged.Storage.Azure = override.Storage.Azure
	}
	if override.Storage.GCS != nil {
		merged.Storage.GCS = override.Storage.GCS
	}

	// Merge retention config
	if override.Retention.MaxBackups != 0 {
		merged.Retention.MaxBackups = override.Retention.MaxBackups
	}
	if override.Retention.MaxAge != 0 {
		merged.Retention.MaxAge = override.Retention.MaxAge
	}
	if override.Retention.CleanupInterval != 0 {
		merged.Retention.CleanupInterval = override.Retention.CleanupInterval
	}
	if override.Retention.KeepDaily != 0 {
		merged.Retention.KeepDaily = override.Retention.KeepDaily
	}
	if override.Retention.KeepWeekly != 0 {
		merged.Retention.KeepWeekly = override.Retention.KeepWeekly
	}
	if override.Retention.KeepMonthly != 0 {
		merged.Retention.KeepMonthly = override.Retention.KeepMonthly
	}

	// Merge compression config
	merged.Compression.Enabled = override.Compression.Enabled
	if override.Compression.Algorithm != "" {
		merged.Compression.Algorithm = override.Compression.Algorithm
	}
	if override.Compression.Level != 0 {
		merged.Compression.Level = override.Compression.Level
	}
	if override.Compression.Threshold != 0 {
		merged.Compression.Threshold = override.Compression.Threshold
	}

	// Merge encryption config
	merged.Encryption.Enabled = override.Encryption.Enabled
	if override.Encryption.KeySource != "" {
		merged.Encryption.KeySource = override.Encryption.KeySource
	}
	if override.Encryption.KeyPath != "" {
		merged.Encryption.KeyPath = override.Encryption.KeyPath
	}
	if override.Encryption.KeyEnvVar != "" {
		merged.Encryption.KeyEnvVar = override.Encryption.KeyEnvVar
	}
	merged.Encryption.RotationEnabled = override.Encryption.RotationEnabled
	if override.Encryption.RotationDays != 0 {
		merged.Encryption.RotationDays = override.Encryption.RotationDays
	}

	// Merge validation config
	merged.Validation.Enabled = override.Validation.Enabled
	if override.Validation.ChecksumAlgorithm != "" {
		merged.Validation.ChecksumAlgorithm = override.Validation.ChecksumAlgorithm
	}
	merged.Validation.ValidateOnCreate = override.Validation.ValidateOnCreate
	merged.Validation.ValidateOnRestore = override.Validation.ValidateOnRestore
	if override.Validation.ValidationTimeout != 0 {
		merged.Validation.ValidationTimeout = override.Validation.ValidationTimeout
	}
	merged.Validation.DryRunValidation = override.Validation.DryRunValidation

	return &merged
}
