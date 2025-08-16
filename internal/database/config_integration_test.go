package database

import (
	"os"
	"path/filepath"
	"testing"

	"mysql-schema-sync/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      config.BackupConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid local storage config",
			config: config.BackupConfig{
				Enabled: true,
				Storage: config.StorageConfig{
					Provider: "local",
					Local: &config.LocalConfig{
						BasePath:    "./backups",
						Permissions: "0755",
					},
				},
				Retention: config.RetentionConfig{
					MaxBackups: 10,
				},
				Validation: config.ValidationConfig{
					Enabled:           true,
					ChecksumAlgorithm: "sha256",
				},
			},
			expectError: false,
		},
		{
			name: "invalid storage provider",
			config: config.BackupConfig{
				Enabled: true,
				Storage: config.StorageConfig{
					Provider: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "invalid storage provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackupConfigDefaults(t *testing.T) {
	cfg := config.BackupConfig{
		Enabled: true,
	}

	cfg.SetDefaults()

	assert.Equal(t, "local", cfg.Storage.Provider)
	assert.NotNil(t, cfg.Storage.Local)
	assert.Equal(t, "./backups", cfg.Storage.Local.BasePath)
	assert.Equal(t, "0755", cfg.Storage.Local.Permissions)
	assert.Equal(t, 10, cfg.Retention.MaxBackups)
	assert.Equal(t, "24h", cfg.Retention.CleanupInterval)
	assert.True(t, cfg.Validation.Enabled)
	assert.Equal(t, "sha256", cfg.Validation.ChecksumAlgorithm)
}

func TestCLIConfigWithBackup(t *testing.T) {
	cfg := CLIConfig{
		SourceDB: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Username: "test",
			Password: "test",
			Database: "testdb",
		},
		TargetDB: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Username: "test",
			Password: "test",
			Database: "testdb",
		},
		Backup: config.BackupConfig{
			Enabled: true,
			Storage: config.StorageConfig{
				Provider: "local",
				Local: &config.LocalConfig{
					BasePath: "./backups",
				},
			},
			Retention: config.RetentionConfig{
				MaxBackups: 5,
			},
		},
	}

	// Set defaults
	cfg.SetDefaults()

	// Validate
	err := cfg.Validate()
	assert.NoError(t, err)

	// Verify backup defaults were set
	assert.Equal(t, "0755", cfg.Backup.Storage.Local.Permissions)
	assert.Equal(t, "24h", cfg.Backup.Retention.CleanupInterval)
	assert.True(t, cfg.Backup.Validation.Enabled)
}

func TestConfigMigration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_migration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test config file without backup configuration
	configPath := filepath.Join(tempDir, "test-config.yaml")
	originalConfig := `source:
  host: localhost
  port: 3306
  username: test
  password: test
  database: testdb
target:
  host: localhost
  port: 3306
  username: test
  password: test
  database: testdb
dry_run: false
verbose: false
auto_approve: false`

	err = os.WriteFile(configPath, []byte(originalConfig), 0644)
	require.NoError(t, err)

	// Test migration
	migration := NewConfigMigration(configPath)
	err = migration.MigrateConfig()
	require.NoError(t, err)

	// Verify backup file was created
	backupPath := configPath + ".backup"
	assert.FileExists(t, backupPath)

	// Verify migrated config can be loaded
	err = migration.ValidateMigratedConfig()
	assert.NoError(t, err)

	// Test that migration fails if backup config already exists
	err = migration.MigrateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup configuration already exists")
}

func TestCreateDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "default_config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "default-config.yaml")

	// Create default config
	err = CreateDefaultConfig(configPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configPath)

	// Load and validate the created config
	loader := NewConfigLoader()
	cfg, err := loader.LoadConfig(configPath)
	require.NoError(t, err)

	// Verify backup configuration is present and valid
	assert.False(t, cfg.Backup.Enabled) // Should be disabled by default
	assert.Equal(t, "local", cfg.Backup.Storage.Provider)
	assert.NotNil(t, cfg.Backup.Storage.Local)
	assert.Equal(t, "./backups", cfg.Backup.Storage.Local.BasePath)

	// Test that creation fails if file already exists
	err = CreateDefaultConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file already exists")
}
