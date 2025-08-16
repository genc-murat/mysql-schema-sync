package database

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigLoader handles loading configuration from various sources
type ConfigLoader struct {
	viper *viper.Viper
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		viper: viper.New(),
	}
}

// AddFlags adds CLI flags to the cobra command
func (cl *ConfigLoader) AddFlags(cmd *cobra.Command) {
	// Source database flags
	cmd.Flags().String("source-host", "", "Source database host")
	cmd.Flags().Int("source-port", 3306, "Source database port")
	cmd.Flags().String("source-username", "", "Source database username")
	cmd.Flags().String("source-password", "", "Source database password")
	cmd.Flags().String("source-database", "", "Source database name")

	// Target database flags
	cmd.Flags().String("target-host", "", "Target database host")
	cmd.Flags().Int("target-port", 3306, "Target database port")
	cmd.Flags().String("target-username", "", "Target database username")
	cmd.Flags().String("target-password", "", "Target database password")
	cmd.Flags().String("target-database", "", "Target database name")

	// General options
	cmd.Flags().Bool("dry-run", false, "Show changes without applying them")
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().Bool("auto-approve", false, "Automatically approve changes without confirmation")

	// Backup options
	cmd.Flags().Bool("backup-enabled", false, "Enable automatic backup before migrations")
	cmd.Flags().String("backup-storage-provider", "local", "Backup storage provider (local, s3, azure, gcs)")
	cmd.Flags().String("backup-local-path", "./backups", "Local backup storage path")
	cmd.Flags().Int("backup-max-backups", 10, "Maximum number of backups to retain")
	cmd.Flags().String("backup-max-age", "", "Maximum age of backups to retain (e.g., '30d', '720h')")
	cmd.Flags().Bool("backup-compression-enabled", false, "Enable backup compression")
	cmd.Flags().String("backup-compression-algorithm", "gzip", "Compression algorithm (gzip, lz4, zstd)")
	cmd.Flags().Bool("backup-encryption-enabled", false, "Enable backup encryption")
	cmd.Flags().String("backup-encryption-key-source", "env", "Encryption key source (env, file, external)")

	// Bind flags to viper
	cl.viper.BindPFlag("source.host", cmd.Flags().Lookup("source-host"))
	cl.viper.BindPFlag("source.port", cmd.Flags().Lookup("source-port"))
	cl.viper.BindPFlag("source.username", cmd.Flags().Lookup("source-username"))
	cl.viper.BindPFlag("source.password", cmd.Flags().Lookup("source-password"))
	cl.viper.BindPFlag("source.database", cmd.Flags().Lookup("source-database"))

	cl.viper.BindPFlag("target.host", cmd.Flags().Lookup("target-host"))
	cl.viper.BindPFlag("target.port", cmd.Flags().Lookup("target-port"))
	cl.viper.BindPFlag("target.username", cmd.Flags().Lookup("target-username"))
	cl.viper.BindPFlag("target.password", cmd.Flags().Lookup("target-password"))
	cl.viper.BindPFlag("target.database", cmd.Flags().Lookup("target-database"))

	cl.viper.BindPFlag("dry_run", cmd.Flags().Lookup("dry-run"))
	cl.viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	cl.viper.BindPFlag("auto_approve", cmd.Flags().Lookup("auto-approve"))

	// Bind backup flags
	cl.viper.BindPFlag("backup.enabled", cmd.Flags().Lookup("backup-enabled"))
	cl.viper.BindPFlag("backup.storage.provider", cmd.Flags().Lookup("backup-storage-provider"))
	cl.viper.BindPFlag("backup.storage.local.base_path", cmd.Flags().Lookup("backup-local-path"))
	cl.viper.BindPFlag("backup.retention.max_backups", cmd.Flags().Lookup("backup-max-backups"))
	cl.viper.BindPFlag("backup.retention.max_age", cmd.Flags().Lookup("backup-max-age"))
	cl.viper.BindPFlag("backup.compression.enabled", cmd.Flags().Lookup("backup-compression-enabled"))
	cl.viper.BindPFlag("backup.compression.algorithm", cmd.Flags().Lookup("backup-compression-algorithm"))
	cl.viper.BindPFlag("backup.encryption.enabled", cmd.Flags().Lookup("backup-encryption-enabled"))
	cl.viper.BindPFlag("backup.encryption.key_source", cmd.Flags().Lookup("backup-encryption-key-source"))
}

// LoadConfig loads configuration from file, environment variables, and CLI flags
func (cl *ConfigLoader) LoadConfig(configFile string) (*CLIConfig, error) {
	// Set config file if provided
	if configFile != "" {
		cl.viper.SetConfigFile(configFile)
	} else {
		// Set default config file locations
		cl.viper.SetConfigName("mysql-schema-sync")
		cl.viper.SetConfigType("yaml")
		cl.viper.AddConfigPath(".")
		cl.viper.AddConfigPath("$HOME/.config/mysql-schema-sync")
		cl.viper.AddConfigPath("$HOME")
	}

	// Enable environment variable support
	cl.viper.AutomaticEnv()
	cl.viper.SetEnvPrefix("MYSQL_SCHEMA_SYNC")
	cl.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set backup configuration defaults
	cl.setBackupDefaults()

	// Try to read config file (it's okay if it doesn't exist)
	if err := cl.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal configuration
	var config CLIConfig
	if err := cl.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	// Load environment variables for backup configuration
	config.Backup.LoadFromEnvironment()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// setBackupDefaults sets default values for backup configuration in viper
func (cl *ConfigLoader) setBackupDefaults() {
	// Main backup settings
	cl.viper.SetDefault("backup.enabled", false)

	// Storage configuration
	cl.viper.SetDefault("backup.storage.provider", "local")
	cl.viper.SetDefault("backup.storage.local.base_path", "./backups")
	cl.viper.SetDefault("backup.storage.local.permissions", "0755")

	// Retention configuration
	cl.viper.SetDefault("backup.retention.max_backups", 10)
	cl.viper.SetDefault("backup.retention.cleanup_interval", "24h")

	// Compression configuration
	cl.viper.SetDefault("backup.compression.enabled", false)
	cl.viper.SetDefault("backup.compression.algorithm", "gzip")
	cl.viper.SetDefault("backup.compression.level", 6)
	cl.viper.SetDefault("backup.compression.threshold", 1024)

	// Encryption configuration
	cl.viper.SetDefault("backup.encryption.enabled", false)
	cl.viper.SetDefault("backup.encryption.key_source", "env")
	cl.viper.SetDefault("backup.encryption.key_env_var", "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY")
	cl.viper.SetDefault("backup.encryption.rotation_enabled", false)
	cl.viper.SetDefault("backup.encryption.rotation_days", 90)

	// Validation configuration
	cl.viper.SetDefault("backup.validation.enabled", true)
	cl.viper.SetDefault("backup.validation.checksum_algorithm", "sha256")
	cl.viper.SetDefault("backup.validation.validate_on_create", true)
	cl.viper.SetDefault("backup.validation.validate_on_restore", true)
	cl.viper.SetDefault("backup.validation.validation_timeout", "5m")
	cl.viper.SetDefault("backup.validation.dry_run_validation", true)
}

// GetUsedConfigFile returns the path of the config file that was used
func (cl *ConfigLoader) GetUsedConfigFile() string {
	return cl.viper.ConfigFileUsed()
}
