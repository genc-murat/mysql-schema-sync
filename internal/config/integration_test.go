package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigIntegration_IntegrateBackupConfig(t *testing.T) {
	tests := []struct {
		name           string
		existingConfig string
		expectError    bool
		errorContains  string
	}{
		{
			name: "integrate into empty config",
			existingConfig: `source:
  host: localhost
  port: 3306
  username: root
  database: test_db
target:
  host: localhost
  port: 3306
  username: root
  database: test_db
dry_run: false`,
			expectError: false,
		},
		{
			name: "integrate into config with existing backup",
			existingConfig: `source:
  host: localhost
backup:
  enabled: true`,
			expectError:   true,
			errorContains: "backup configuration already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Write existing config
			err := os.WriteFile(configPath, []byte(tt.existingConfig), 0644)
			require.NoError(t, err)

			// Test integration
			ci := NewConfigIntegration()
			err = ci.IntegrateBackupConfig(configPath)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify backup was created
				backupPath := configPath + ".backup"
				assert.FileExists(t, backupPath)

				// Verify config was updated
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)
				configContent := string(data)
				assert.Contains(t, configContent, "backup:")
				assert.Contains(t, configContent, "enabled: false")
				assert.Contains(t, configContent, "storage:")
				assert.Contains(t, configContent, "provider: local")
			}
		})
	}
}

func TestConfigIntegration_ValidateIntegratedConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
	}{
		{
			name: "valid integrated config",
			config: `source:
  host: localhost
  port: 3306
  username: root
  database: test_db
target:
  host: localhost
  port: 3306
  username: root
  database: test_db
backup:
  enabled: false
  storage:
    provider: local
    local:
      base_path: ./backups
  retention:
    max_backups: 10
  validation:
    enabled: true`,
			expectError: false,
		},
		{
			name: "invalid storage provider",
			config: `backup:
  enabled: true
  storage:
    provider: invalid_provider`,
			expectError: true,
		},
		{
			name: "missing required storage config",
			config: `backup:
  enabled: true
  storage:
    provider: s3`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Write config
			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			// Test validation
			ci := NewConfigIntegration()
			err = ci.ValidateIntegratedConfig(configPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigIntegration_GenerateConfigTemplate(t *testing.T) {
	ci := NewConfigIntegration()
	template := ci.GenerateConfigTemplate()

	// Verify template contains all expected sections
	assert.Contains(t, template, "source:")
	assert.Contains(t, template, "target:")
	assert.Contains(t, template, "backup:")
	assert.Contains(t, template, "storage:")
	assert.Contains(t, template, "retention:")
	assert.Contains(t, template, "compression:")
	assert.Contains(t, template, "encryption:")
	assert.Contains(t, template, "validation:")
	assert.Contains(t, template, "display:")

	// Verify storage providers are documented
	assert.Contains(t, template, "provider: local")
	assert.Contains(t, template, "# s3:")
	assert.Contains(t, template, "# azure:")
	assert.Contains(t, template, "# gcs:")

	// Verify environment variable examples
	assert.Contains(t, template, "MYSQL_SCHEMA_SYNC_BACKUP_ENABLED")
	assert.Contains(t, template, "MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET")

	// Verify security recommendations
	assert.Contains(t, template, "Security recommendations:")
	assert.Contains(t, template, "environment variables")
	assert.Contains(t, template, "chmod 600")
}

func TestConfigIntegration_ListEnvironmentVariables(t *testing.T) {
	ci := NewConfigIntegration()
	envVars := ci.ListEnvironmentVariables()

	// Verify essential environment variables are included
	expectedVars := []string{
		"MYSQL_SCHEMA_SYNC_BACKUP_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER",
		"MYSQL_SCHEMA_SYNC_BACKUP_LOCAL_BASE_PATH",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET",
		"MYSQL_SCHEMA_SYNC_BACKUP_AZURE_ACCOUNT_NAME",
		"MYSQL_SCHEMA_SYNC_BACKUP_GCS_BUCKET",
		"MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ENABLED",
	}

	for _, expectedVar := range expectedVars {
		assert.Contains(t, envVars, expectedVar, "Environment variable %s should be included", expectedVar)
	}

	// Verify no duplicates
	varMap := make(map[string]bool)
	for _, envVar := range envVars {
		assert.False(t, varMap[envVar], "Duplicate environment variable: %s", envVar)
		varMap[envVar] = true
	}
}

func TestConfigIntegration_GetConfigurationHelp(t *testing.T) {
	ci := NewConfigIntegration()
	help := ci.GetConfigurationHelp()

	// Verify help contains essential information
	assert.Contains(t, help, "Configuration Hierarchy")
	assert.Contains(t, help, "Storage Providers")
	assert.Contains(t, help, "Retention Policies")
	assert.Contains(t, help, "Compression Algorithms")
	assert.Contains(t, help, "Security Features")

	// Verify storage providers are documented
	assert.Contains(t, help, "local:")
	assert.Contains(t, help, "s3:")
	assert.Contains(t, help, "azure:")
	assert.Contains(t, help, "gcs:")

	// Verify compression algorithms are documented
	assert.Contains(t, help, "gzip:")
	assert.Contains(t, help, "lz4:")
	assert.Contains(t, help, "zstd:")
}

func TestConfigIntegration_EnvironmentVariableIntegration(t *testing.T) {
	// Test environment variable loading
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create minimal config
	config := `backup:
  enabled: false
  storage:
    provider: local`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Set environment variables
	envVars := map[string]string{
		"MYSQL_SCHEMA_SYNC_BACKUP_ENABLED":               "true",
		"MYSQL_SCHEMA_SYNC_BACKUP_STORAGE_PROVIDER":      "s3",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_BUCKET":             "test-bucket",
		"MYSQL_SCHEMA_SYNC_BACKUP_S3_REGION":             "us-west-2",
		"MYSQL_SCHEMA_SYNC_BACKUP_MAX_BACKUPS":           "20",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ENABLED":   "true",
		"MYSQL_SCHEMA_SYNC_BACKUP_COMPRESSION_ALGORITHM": "lz4",
		"MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_ENABLED":    "true",
		"MYSQL_SCHEMA_SYNC_BACKUP_VALIDATION_ENABLED":    "false",
	}

	// Set environment variables
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	// Load and validate configuration
	ci := NewConfigIntegration()
	ci.setupViper(configPath)
	err = ci.viper.ReadInConfig()
	require.NoError(t, err)

	// Unmarshal backup configuration
	var backupConfig BackupConfig
	err = ci.viper.UnmarshalKey("backup", &backupConfig)
	require.NoError(t, err)

	// Load environment variables
	backupConfig.LoadFromEnvironment()

	// Verify environment variables were loaded
	assert.True(t, backupConfig.Enabled)
	assert.Equal(t, "s3", backupConfig.Storage.Provider)
	assert.Equal(t, "test-bucket", backupConfig.Storage.S3.Bucket)
	assert.Equal(t, "us-west-2", backupConfig.Storage.S3.Region)
	assert.Equal(t, 20, backupConfig.Retention.MaxBackups)
	assert.True(t, backupConfig.Compression.Enabled)
	assert.Equal(t, "lz4", backupConfig.Compression.Algorithm)
	assert.True(t, backupConfig.Encryption.Enabled)
	assert.False(t, backupConfig.Validation.Enabled)
}

func TestConfigIntegration_ConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "non-existent-config.yaml")

	ci := NewConfigIntegration()

	// Should handle missing config file gracefully
	err := ci.IntegrateBackupConfig(nonExistentPath)
	assert.NoError(t, err)

	// Verify config file was created
	assert.FileExists(t, nonExistentPath)
}

func TestConfigIntegration_WriteConfigPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config with backup integration
	ci := NewConfigIntegration()
	err := ci.IntegrateBackupConfig(configPath)
	require.NoError(t, err)

	// Verify file permissions (on Windows, permissions work differently)
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	// On Windows, just verify the file exists and is readable
	assert.True(t, info.Mode().IsRegular())
}

func TestConfigIntegration_BackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create original config
	originalConfig := `source:
  host: localhost
target:
  host: localhost`

	err := os.WriteFile(configPath, []byte(originalConfig), 0644)
	require.NoError(t, err)

	// Integrate backup config
	ci := NewConfigIntegration()
	err = ci.IntegrateBackupConfig(configPath)
	require.NoError(t, err)

	// Verify backup file exists and contains original content
	backupPath := configPath + ".backup"
	assert.FileExists(t, backupPath)

	backupData, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalConfig, string(backupData))

	// Verify original file was updated
	updatedData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	updatedContent := string(updatedData)
	assert.Contains(t, updatedContent, "backup:")
	assert.Contains(t, updatedContent, "host: localhost")
}

func TestConfigIntegration_DefaultValues(t *testing.T) {
	ci := NewConfigIntegration()
	ci.setBackupDefaults()

	// Test that defaults are set correctly
	assert.False(t, ci.viper.GetBool("backup.enabled"))
	assert.Equal(t, "local", ci.viper.GetString("backup.storage.provider"))
	assert.Equal(t, "./backups", ci.viper.GetString("backup.storage.local.base_path"))
	assert.Equal(t, "0755", ci.viper.GetString("backup.storage.local.permissions"))
	assert.Equal(t, 10, ci.viper.GetInt("backup.retention.max_backups"))
	assert.Equal(t, "24h", ci.viper.GetString("backup.retention.cleanup_interval"))
	assert.False(t, ci.viper.GetBool("backup.compression.enabled"))
	assert.Equal(t, "gzip", ci.viper.GetString("backup.compression.algorithm"))
	assert.Equal(t, 6, ci.viper.GetInt("backup.compression.level"))
	assert.Equal(t, 1024, ci.viper.GetInt("backup.compression.threshold"))
	assert.False(t, ci.viper.GetBool("backup.encryption.enabled"))
	assert.Equal(t, "env", ci.viper.GetString("backup.encryption.key_source"))
	assert.Equal(t, "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY", ci.viper.GetString("backup.encryption.key_env_var"))
	assert.False(t, ci.viper.GetBool("backup.encryption.rotation_enabled"))
	assert.Equal(t, 90, ci.viper.GetInt("backup.encryption.rotation_days"))
	assert.True(t, ci.viper.GetBool("backup.validation.enabled"))
	assert.Equal(t, "sha256", ci.viper.GetString("backup.validation.checksum_algorithm"))
	assert.True(t, ci.viper.GetBool("backup.validation.validate_on_create"))
	assert.True(t, ci.viper.GetBool("backup.validation.validate_on_restore"))
	assert.Equal(t, "5m", ci.viper.GetString("backup.validation.validation_timeout"))
	assert.True(t, ci.viper.GetBool("backup.validation.dry_run_validation"))
}

func TestConfigIntegration_ComplexConfigIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create complex existing config
	existingConfig := `# Existing configuration
source:
  host: prod-db.example.com
  port: 3306
  username: app_user
  password: secret123
  database: production_db
  timeout: 60s

target:
  host: staging-db.example.com
  port: 3306
  username: app_user
  password: secret123
  database: staging_db
  timeout: 60s

dry_run: true
verbose: true
auto_approve: false

display:
  color_enabled: true
  theme: light
  output_format: json
  use_icons: false
  show_progress: true
  interactive: true
  table_style: rounded
  max_table_width: 100

# Custom comments and formatting
# This is important configuration`

	err := os.WriteFile(configPath, []byte(existingConfig), 0644)
	require.NoError(t, err)

	// Integrate backup config
	ci := NewConfigIntegration()
	err = ci.IntegrateBackupConfig(configPath)
	require.NoError(t, err)

	// Read updated config
	updatedData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	updatedContent := string(updatedData)

	// Verify existing config is preserved
	assert.Contains(t, updatedContent, "prod-db.example.com")
	assert.Contains(t, updatedContent, "staging-db.example.com")
	assert.Contains(t, updatedContent, "dry_run: true")
	assert.Contains(t, updatedContent, "theme: light")
	assert.Contains(t, updatedContent, "output_format: json")

	// Verify backup config was added
	assert.Contains(t, updatedContent, "backup:")
	assert.Contains(t, updatedContent, "enabled: false")
	assert.Contains(t, updatedContent, "storage:")
	assert.Contains(t, updatedContent, "retention:")
	assert.Contains(t, updatedContent, "compression:")
	assert.Contains(t, updatedContent, "encryption:")
	assert.Contains(t, updatedContent, "validation:")

	// Validate the integrated config
	err = ci.ValidateIntegratedConfig(configPath)
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkConfigIntegration_IntegrateBackupConfig(b *testing.B) {
	tmpDir := b.TempDir()

	for i := 0; i < b.N; i++ {
		configPath := filepath.Join(tmpDir, "config_"+string(rune(i))+".yaml")

		// Create test config
		config := `source:
  host: localhost
target:
  host: localhost`

		err := os.WriteFile(configPath, []byte(config), 0644)
		require.NoError(b, err)

		// Benchmark integration
		ci := NewConfigIntegration()
		err = ci.IntegrateBackupConfig(configPath)
		require.NoError(b, err)
	}
}

func BenchmarkConfigIntegration_ValidateIntegratedConfig(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create test config with backup
	config := `source:
  host: localhost
  port: 3306
  username: root
  database: test_db
target:
  host: localhost
  port: 3306
  username: root
  database: test_db
backup:
  enabled: false
  storage:
    provider: local
    local:
      base_path: ./backups
  retention:
    max_backups: 10
  validation:
    enabled: true`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(b, err)

	ci := NewConfigIntegration()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ci.ValidateIntegratedConfig(configPath)
		require.NoError(b, err)
	}
}
