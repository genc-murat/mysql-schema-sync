package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ConfigIntegration handles integration of backup configuration with the existing config system
type ConfigIntegration struct {
	viper *viper.Viper
}

// NewConfigIntegration creates a new configuration integration instance
func NewConfigIntegration() *ConfigIntegration {
	return &ConfigIntegration{
		viper: viper.New(),
	}
}

// IntegrateBackupConfig integrates backup configuration into the existing configuration system
func (ci *ConfigIntegration) IntegrateBackupConfig(configPath string) error {
	// Set up viper configuration
	ci.setupViper(configPath)

	// Check if config file exists
	configExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configExists = false
	}

	// Load existing configuration if it exists
	if configExists {
		if err := ci.viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("error reading config file: %w", err)
			}
		}

		// Check if backup configuration already exists
		if ci.viper.IsSet("backup") {
			return fmt.Errorf("backup configuration already exists in %s", configPath)
		}

		// Create backup of original file
		backupPath := configPath + ".backup"
		if err := ci.createConfigBackup(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup of configuration file: %w", err)
		}
		fmt.Printf("Configuration backup created: %s\n", backupPath)
	}

	// Add backup configuration to existing config
	ci.setBackupDefaults()

	// Write updated configuration
	if err := ci.writeConfig(configPath); err != nil {
		return fmt.Errorf("failed to write updated configuration: %w", err)
	}

	fmt.Printf("Backup configuration integrated successfully into: %s\n", configPath)
	return nil
}

// setupViper configures viper for configuration loading
func (ci *ConfigIntegration) setupViper(configPath string) {
	if configPath != "" {
		ci.viper.SetConfigFile(configPath)
	} else {
		ci.viper.SetConfigName("mysql-schema-sync")
		ci.viper.SetConfigType("yaml")
		ci.viper.AddConfigPath(".")
		ci.viper.AddConfigPath("$HOME/.config/mysql-schema-sync")
		ci.viper.AddConfigPath("$HOME")
	}

	// Enable environment variable support
	ci.viper.AutomaticEnv()
	ci.viper.SetEnvPrefix("MYSQL_SCHEMA_SYNC")
	ci.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// setBackupDefaults sets default backup configuration values
func (ci *ConfigIntegration) setBackupDefaults() {
	// Main backup settings
	ci.viper.SetDefault("backup.enabled", false)

	// Storage configuration
	ci.viper.SetDefault("backup.storage.provider", "local")
	ci.viper.SetDefault("backup.storage.local.base_path", "./backups")
	ci.viper.SetDefault("backup.storage.local.permissions", "0755")

	// Retention configuration
	ci.viper.SetDefault("backup.retention.max_backups", 10)
	ci.viper.SetDefault("backup.retention.cleanup_interval", "24h")

	// Compression configuration
	ci.viper.SetDefault("backup.compression.enabled", false)
	ci.viper.SetDefault("backup.compression.algorithm", "gzip")
	ci.viper.SetDefault("backup.compression.level", 6)
	ci.viper.SetDefault("backup.compression.threshold", 1024)

	// Encryption configuration
	ci.viper.SetDefault("backup.encryption.enabled", false)
	ci.viper.SetDefault("backup.encryption.key_source", "env")
	ci.viper.SetDefault("backup.encryption.key_env_var", "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY")
	ci.viper.SetDefault("backup.encryption.rotation_enabled", false)
	ci.viper.SetDefault("backup.encryption.rotation_days", 90)

	// Validation configuration
	ci.viper.SetDefault("backup.validation.enabled", true)
	ci.viper.SetDefault("backup.validation.checksum_algorithm", "sha256")
	ci.viper.SetDefault("backup.validation.validate_on_create", true)
	ci.viper.SetDefault("backup.validation.validate_on_restore", true)
	ci.viper.SetDefault("backup.validation.validation_timeout", "5m")
	ci.viper.SetDefault("backup.validation.dry_run_validation", true)
}

// createConfigBackup creates a backup of the original configuration file
func (ci *ConfigIntegration) createConfigBackup(configPath, backupPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// writeConfig writes the updated configuration to file
func (ci *ConfigIntegration) writeConfig(configPath string) error {
	// Get all settings from viper
	allSettings := ci.viper.AllSettings()

	// Marshal to YAML
	data, err := yaml.Marshal(allSettings)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// ValidateIntegratedConfig validates the integrated configuration
func (ci *ConfigIntegration) ValidateIntegratedConfig(configPath string) error {
	// Set up viper and load config
	ci.setupViper(configPath)
	if err := ci.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration: %w", err)
	}

	// Unmarshal backup configuration
	var backupConfig BackupConfig
	if err := ci.viper.UnmarshalKey("backup", &backupConfig); err != nil {
		return fmt.Errorf("failed to unmarshal backup configuration: %w", err)
	}

	// Load environment variables
	backupConfig.LoadFromEnvironment()

	// Set defaults
	backupConfig.SetDefaults()

	// Validate
	if err := backupConfig.Validate(); err != nil {
		return fmt.Errorf("backup configuration validation failed: %w", err)
	}

	fmt.Printf("Integrated backup configuration is valid. Backup enabled: %t\n", backupConfig.Enabled)
	return nil
}

// GenerateConfigTemplate generates a complete configuration template with backup settings
func (ci *ConfigIntegration) GenerateConfigTemplate() string {
	return `# MySQL Schema Sync Configuration File
# Complete configuration template with backup system integration

# Source database connection
source:
  host: localhost          # Source database hostname or IP
  port: 3306              # Source database port
  username: root          # Source database username
  password: ""            # Source database password (use env var for security)
  database: source_db     # Source database name
  timeout: 30s            # Connection timeout for source database

# Target database connection
target:
  host: localhost          # Target database hostname or IP
  port: 3306              # Target database port
  username: root          # Target database username
  password: ""            # Target database password (use env var for security)
  database: target_db     # Target database name
  timeout: 30s            # Connection timeout for target database

# Operation settings
dry_run: false            # Show changes without applying them
verbose: false            # Enable verbose output with detailed information
auto_approve: false       # Automatically approve changes without confirmation

# Backup system configuration
backup:
  enabled: false          # Enable automatic backup before migrations
  
  # Storage configuration
  storage:
    provider: local       # Storage provider (local, s3, azure, gcs)
    
    # Local storage settings (used when provider is 'local')
    local:
      base_path: ./backups    # Local backup storage path
      permissions: "0755"     # Directory permissions
    
    # Amazon S3 storage settings (used when provider is 's3')
    # s3:
    #   bucket: my-backups      # S3 bucket name
    #   region: us-east-1       # AWS region
    #   access_key: ""          # AWS access key (or use env var)
    #   secret_key: ""          # AWS secret key (or use env var)
    
    # Azure Blob Storage settings (used when provider is 'azure')
    # azure:
    #   account_name: ""        # Azure storage account name
    #   account_key: ""         # Azure storage account key
    #   container_name: backups # Azure container name
    
    # Google Cloud Storage settings (used when provider is 'gcs')
    # gcs:
    #   bucket: my-backups      # GCS bucket name
    #   credentials_path: ""    # Path to GCS credentials JSON file
    #   project_id: ""          # GCP project ID
  
  # Retention policy configuration
  retention:
    max_backups: 10         # Maximum number of backups to retain (0 = unlimited)
    max_age: ""             # Maximum age of backups (e.g., "30d", "720h")
    cleanup_interval: 24h   # How often to run cleanup
    keep_daily: 0           # Number of daily backups to keep
    keep_weekly: 0          # Number of weekly backups to keep
    keep_monthly: 0         # Number of monthly backups to keep
  
  # Compression configuration
  compression:
    enabled: false          # Enable backup compression
    algorithm: gzip         # Compression algorithm (gzip, lz4, zstd)
    level: 6                # Compression level (algorithm-specific)
    threshold: 1024         # Minimum size to compress (bytes)
  
  # Encryption configuration
  encryption:
    enabled: false          # Enable backup encryption
    key_source: env         # Key source (env, file, external)
    key_env_var: MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY  # Environment variable name
    key_path: ""            # Path to key file (if key_source is 'file')
    rotation_enabled: false # Enable automatic key rotation
    rotation_days: 90       # Days between key rotations
  
  # Validation configuration
  validation:
    enabled: true           # Enable backup validation
    checksum_algorithm: sha256  # Checksum algorithm (sha256, md5)
    validate_on_create: true    # Validate backup after creation
    validate_on_restore: true   # Validate backup before restore
    validation_timeout: 5m      # Validation operation timeout
    dry_run_validation: true    # Enable dry-run validation tests

# Display configuration
display:
  color_enabled: true       # Enable colorized output
  theme: dark              # Color theme (dark, light, high-contrast, auto)
  output_format: table     # Output format (table, json, yaml, compact)
  use_icons: true          # Enable Unicode icons with ASCII fallbacks
  show_progress: true      # Show progress indicators and spinners
  interactive: true        # Enable interactive confirmations
  table_style: default     # Table styling (default, rounded, border, minimal)
  max_table_width: 120     # Maximum table width (40-300)

# Environment variable examples for backup configuration:
# MYSQL_SCHEMA_SYNC_BACKUP_ENABLED=true
# MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER=s3
# MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET=my-backups
# MYSQL_SCHEMA_SYNC_BACKUP_S3_REGION=us-east-1
# MYSQL_SCHEMA_SYNC_BACKUP_S3_ACCESS_KEY=your_access_key
# MYSQL_SCHEMA_SYNC_BACKUP_S3_SECRET_KEY=your_secret_key
# MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS=20
# MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED=true
# MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED=true
# MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY=your_encryption_key

# Security recommendations:
# 1. Store sensitive values in environment variables
# 2. Set restrictive file permissions: chmod 600 config.yaml
# 3. Use dedicated database users with minimal privileges
# 4. Enable encryption for sensitive backup data
# 5. Regularly rotate encryption keys
# 6. Monitor backup storage usage and access
`
}

// ListEnvironmentVariables lists all backup-related environment variables
func (ci *ConfigIntegration) ListEnvironmentVariables() []string {
	return []string{
		"MYSQL_SCHEMA_SYNC_BACKUP_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER",
		"MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_BASE_PATH",
		"MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_PERMISSIONS",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_REGION",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_ACCESS_KEY",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_SECRET_KEY",
		"MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_NAME",
		"MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_KEY",
		"MYSQL_SCHEMA_SYNC_BACKUP_AZURE_CONTAINER_NAME",
		"MYSQL_SCHEMA_SYNC_BACKUP_GCS_BUCKET",
		"MYSQL_SCHEMA_SYNC_BACKUP_GCS_CREDENTIALS_PATH",
		"MYSQL_SCHEMA_SYNC_BACKUP_GCS_PROJECT_ID",
		"MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS",
		"MYSQL_SCHEMA_SYNC_BACKUP_MAX_AGE",
		"MYSQL_SCHEMA_SYNC_BACKUP_CLEANUP_INTERVAL",
		"MYSQL_SCHEMA_SYNC_BACKUP_KEEP_DAILY",
		"MYSQL_SCHEMA_SYNC_BACKUP_KEEP_WEEKLY",
		"MYSQL_SCHEMA_SYNC_BACKUP_KEEP_MONTHLY",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ALGORITHM",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_LEVEL",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_THRESHOLD",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_SOURCE",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_PATH",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_ENV_VAR",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_DAYS",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_CHECKSUM_ALGORITHM",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_CREATE",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_RESTORE",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_TIMEOUT",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_DRY_RUN",
	}
}

// GetConfigurationHelp returns help text for backup configuration
func (ci *ConfigIntegration) GetConfigurationHelp() string {
	return `Backup Configuration Help

The backup system can be configured through:
1. Configuration file (YAML)
2. Environment variables
3. Command-line flags

Configuration Hierarchy (highest to lowest priority):
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

Storage Providers:
- local: Store backups on local filesystem
- s3: Store backups in Amazon S3
- azure: Store backups in Azure Blob Storage
- gcs: Store backups in Google Cloud Storage

Retention Policies:
- max_backups: Maximum number of backups to keep
- max_age: Maximum age of backups (e.g., "30d", "720h")
- keep_daily/weekly/monthly: Advanced retention rules

Compression Algorithms:
- gzip: Good compression ratio, moderate speed
- lz4: Fast compression, lower ratio
- zstd: Best balance of speed and compression

Security Features:
- AES-256 encryption for backup files
- Key rotation support
- Environment variable key storage
- External key management system support

For detailed configuration examples, use:
  mysql-schema-sync config > config.yaml
`
}
