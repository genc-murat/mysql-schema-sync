package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigMigration handles migration of existing configurations to include backup settings
type ConfigMigration struct {
	configPath string
}

// NewConfigMigration creates a new configuration migration instance
func NewConfigMigration(configPath string) *ConfigMigration {
	return &ConfigMigration{
		configPath: configPath,
	}
}

// MigrateConfig migrates an existing configuration file to include backup settings
func (cm *ConfigMigration) MigrateConfig() error {
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", cm.configPath)
	}

	// Read existing configuration
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse existing configuration
	var existingConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &existingConfig); err != nil {
		return fmt.Errorf("failed to parse existing configuration: %w", err)
	}

	// Check if backup configuration already exists
	if _, exists := existingConfig["backup"]; exists {
		return fmt.Errorf("backup configuration already exists in %s", cm.configPath)
	}

	// Create backup of original file
	backupPath := cm.configPath + ".backup"
	if err := cm.createBackup(backupPath); err != nil {
		return fmt.Errorf("failed to create backup of configuration file: %w", err)
	}

	// Add default backup configuration
	existingConfig["backup"] = cm.getDefaultBackupConfig()

	// Write updated configuration
	updatedData, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal updated configuration: %w", err)
	}

	if err := os.WriteFile(cm.configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated configuration: %w", err)
	}

	fmt.Printf("Configuration migrated successfully. Original backed up to: %s\n", backupPath)
	return nil
}

// createBackup creates a backup of the original configuration file
func (cm *ConfigMigration) createBackup(backupPath string) error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// getDefaultBackupConfig returns the default backup configuration
func (cm *ConfigMigration) getDefaultBackupConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled": false,
		"storage": map[string]interface{}{
			"provider": "local",
			"local": map[string]interface{}{
				"base_path":   "./backups",
				"permissions": "0755",
			},
		},
		"retention": map[string]interface{}{
			"max_backups":      10,
			"cleanup_interval": "24h",
		},
		"compression": map[string]interface{}{
			"enabled":   false,
			"algorithm": "gzip",
			"level":     6,
			"threshold": 1024,
		},
		"encryption": map[string]interface{}{
			"enabled":          false,
			"key_source":       "env",
			"key_env_var":      "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY",
			"rotation_enabled": false,
			"rotation_days":    90,
		},
		"validation": map[string]interface{}{
			"enabled":             true,
			"checksum_algorithm":  "sha256",
			"validate_on_create":  true,
			"validate_on_restore": true,
			"validation_timeout":  "5m",
			"dry_run_validation":  true,
		},
	}
}

// ValidateMigratedConfig validates the migrated configuration
func (cm *ConfigMigration) ValidateMigratedConfig() error {
	// Load the migrated configuration using the standard config loader
	loader := NewConfigLoader()
	config, err := loader.LoadConfig(cm.configPath)
	if err != nil {
		return fmt.Errorf("migrated configuration validation failed: %w", err)
	}

	// Additional validation can be added here if needed
	fmt.Printf("Migrated configuration is valid. Backup enabled: %t\n", config.Backup.Enabled)
	return nil
}

// CreateDefaultConfig creates a new configuration file with backup settings
func CreateDefaultConfig(configPath string) error {
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default configuration
	defaultConfig := map[string]interface{}{
		"source": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "",
			"password": "",
			"database": "",
		},
		"target": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "",
			"password": "",
			"database": "",
		},
		"dry_run":      false,
		"verbose":      false,
		"auto_approve": false,
		"backup": map[string]interface{}{
			"enabled": false,
			"storage": map[string]interface{}{
				"provider": "local",
				"local": map[string]interface{}{
					"base_path":   "./backups",
					"permissions": "0755",
				},
			},
			"retention": map[string]interface{}{
				"max_backups":      10,
				"cleanup_interval": "24h",
			},
			"compression": map[string]interface{}{
				"enabled":   false,
				"algorithm": "gzip",
				"level":     6,
				"threshold": 1024,
			},
			"encryption": map[string]interface{}{
				"enabled":          false,
				"key_source":       "env",
				"key_env_var":      "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY",
				"rotation_enabled": false,
				"rotation_days":    90,
			},
			"validation": map[string]interface{}{
				"enabled":             true,
				"checksum_algorithm":  "sha256",
				"validate_on_create":  true,
				"validate_on_restore": true,
				"validation_timeout":  "5m",
				"dry_run_validation":  true,
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("Default configuration created: %s\n", configPath)
	return nil
}

// ListConfigurationOptions lists all available configuration options for backup
func ListConfigurationOptions() {
	fmt.Println("Backup Configuration Options:")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENABLED                    - Enable/disable backup system")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER           - Storage provider (local, s3, azure, gcs)")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_BASE_PATH             - Local storage base path")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_PERMISSIONS           - Local storage permissions")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET                  - S3 bucket name")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_S3_REGION                  - S3 region")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_S3_ACCESS_KEY               - S3 access key")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_S3_SECRET_KEY               - S3 secret key")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_NAME          - Azure account name")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_KEY           - Azure account key")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_AZURE_CONTAINER_NAME        - Azure container name")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_GCS_BUCKET                  - GCS bucket name")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_GCS_CREDENTIALS_PATH        - GCS credentials file path")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_GCS_PROJECT_ID              - GCS project ID")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS                 - Maximum number of backups")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_MAX_AGE                     - Maximum backup age")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_CLEANUP_INTERVAL            - Cleanup interval")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_KEEP_DAILY                  - Daily backups to keep")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_KEEP_WEEKLY                 - Weekly backups to keep")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_KEEP_MONTHLY                - Monthly backups to keep")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED         - Enable compression")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ALGORITHM       - Compression algorithm")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_LEVEL           - Compression level")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_THRESHOLD       - Compression threshold")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED          - Enable encryption")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_SOURCE       - Encryption key source")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_PATH         - Encryption key file path")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY_ENV_VAR      - Encryption key env var")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_ENABLED - Enable key rotation")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ROTATION_DAYS    - Key rotation days")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ENABLED          - Enable validation")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_CHECKSUM_ALGORITHM - Checksum algorithm")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_CREATE        - Validate on create")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ON_RESTORE       - Validate on restore")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_TIMEOUT          - Validation timeout")
	fmt.Println("  MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_DRY_RUN          - Dry run validation")
	fmt.Println("")
	fmt.Println("CLI Flags:")
	fmt.Println("  --backup-enabled                    - Enable automatic backup")
	fmt.Println("  --backup-storage-provider           - Storage provider")
	fmt.Println("  --backup-local-path                 - Local storage path")
	fmt.Println("  --backup-max-backups                - Maximum backups")
	fmt.Println("  --backup-max-age                    - Maximum backup age")
	fmt.Println("  --backup-compression-enabled        - Enable compression")
	fmt.Println("  --backup-compression-algorithm      - Compression algorithm")
	fmt.Println("  --backup-encryption-enabled         - Enable encryption")
	fmt.Println("  --backup-encryption-key-source      - Encryption key source")
}
