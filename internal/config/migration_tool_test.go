package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationTool_DiscoverConfigurations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test configuration files
	configFiles := []string{
		"config.yaml",
		"mysql-schema-sync.yaml",
		".mysql-schema-sync.yaml",
	}

	for _, file := range configFiles {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test: config"), 0644)
		require.NoError(t, err)
	}

	// Change to temp directory for discovery
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test discovery
	mt := NewMigrationTool(false, false)
	discovered, err := mt.DiscoverConfigurations()
	require.NoError(t, err)

	// Should find at least the files we created
	assert.GreaterOrEqual(t, len(discovered), 3)

	// Verify specific files are found
	foundFiles := make(map[string]bool)
	for _, path := range discovered {
		foundFiles[filepath.Base(path)] = true
	}

	for _, expectedFile := range configFiles {
		assert.True(t, foundFiles[expectedFile], "Should find %s", expectedFile)
	}
}

func TestMigrationTool_MigrateConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		existingConfig string
		expectSuccess  bool
		expectExists   bool
	}{
		{
			name: "successful migration",
			existingConfig: `source:
  host: localhost
target:
  host: localhost`,
			expectSuccess: true,
			expectExists:  false,
		},
		{
			name: "already has backup config",
			existingConfig: `source:
  host: localhost
backup:
  enabled: true`,
			expectSuccess: true,
			expectExists:  true,
		},
		{
			name: "invalid yaml",
			existingConfig: `source:
  host: localhost
  invalid: [unclosed`,
			expectSuccess: false,
			expectExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Write test config
			err := os.WriteFile(configPath, []byte(tt.existingConfig), 0644)
			require.NoError(t, err)

			// Test migration
			mt := NewMigrationTool(false, false)
			result := mt.MigrateConfiguration(configPath)

			assert.Equal(t, tt.expectSuccess, result.Success)
			assert.Equal(t, tt.expectExists, result.AlreadyExists)

			if tt.expectSuccess && !tt.expectExists {
				// Verify backup was created
				assert.NotEmpty(t, result.BackupPath)
				assert.FileExists(t, result.BackupPath)

				// Verify config was updated
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)
				configContent := string(data)
				assert.Contains(t, configContent, "backup:")
			}
		})
	}
}

func TestMigrationTool_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create test config
	config := `source:
  host: localhost
target:
  host: localhost`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Test dry run
	mt := NewMigrationTool(true, false) // dry run enabled
	result := mt.MigrateConfiguration(configPath)

	assert.True(t, result.Success)
	assert.Empty(t, result.BackupPath) // No backup in dry run

	// Verify original file was not modified
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, config, string(data))
}

func TestMigrationTool_MigrateAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple config files with standard names that will be discovered
	configs := map[string]string{
		"config.yaml": `source:
  host: localhost1`,
		"mysql-schema-sync.yaml": `source:
  host: localhost2
backup:
  enabled: true`, // Already has backup
		".mysql-schema-sync.yaml": `source:
  host: localhost3`,
	}

	for filename, content := range configs {
		path := filepath.Join(tmpDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test migration of all configs
	mt := NewMigrationTool(false, false)
	results, err := mt.MigrateAll()
	require.NoError(t, err)

	// Should have results for discovered configs
	assert.GreaterOrEqual(t, len(results), 3)

	// Count results
	successful := 0
	alreadyExists := 0
	for _, result := range results {
		if result.Success {
			if result.AlreadyExists {
				alreadyExists++
			} else {
				successful++
			}
		}
	}

	assert.Greater(t, successful+alreadyExists, 0)
}

func TestMigrationTool_CreateDefaultConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new-config.yaml")

	mt := NewMigrationTool(false, false)
	err := mt.CreateDefaultConfiguration(configPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configPath)

	// Verify content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "source:")
	assert.Contains(t, content, "target:")
	assert.Contains(t, content, "backup:")
	assert.Contains(t, content, "display:")
	assert.Contains(t, content, "enabled: false")
	assert.Contains(t, content, "provider: local")
}

func TestMigrationTool_CreateDefaultConfiguration_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "existing-config.yaml")

	// Create existing file
	err := os.WriteFile(configPath, []byte("existing"), 0644)
	require.NoError(t, err)

	mt := NewMigrationTool(false, false)
	err = mt.CreateDefaultConfiguration(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMigrationTool_RollbackMigration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	backupPath := filepath.Join(tmpDir, "config.yaml.backup")

	// Create original config
	originalConfig := `source:
  host: original`

	err := os.WriteFile(configPath, []byte(originalConfig), 0644)
	require.NoError(t, err)

	// Create backup
	err = os.WriteFile(backupPath, []byte(originalConfig), 0644)
	require.NoError(t, err)

	// Modify config (simulate migration)
	modifiedConfig := `source:
  host: modified
backup:
  enabled: true`

	err = os.WriteFile(configPath, []byte(modifiedConfig), 0644)
	require.NoError(t, err)

	// Test rollback
	mt := NewMigrationTool(false, false)
	err = mt.RollbackMigration(configPath, backupPath)
	require.NoError(t, err)

	// Verify rollback
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, originalConfig, string(data))
}

func TestMigrationTool_RollbackMigration_NoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	backupPath := filepath.Join(tmpDir, "nonexistent.backup")

	mt := NewMigrationTool(false, false)
	err := mt.RollbackMigration(configPath, backupPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestMigrationTool_ListBackupFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file
	err := os.WriteFile(configPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Create backup files
	backupFiles := []string{
		"config.yaml.backup-20240101-120000",
		"config.yaml.backup-20240102-120000",
		"config.yaml.backup-20240103-120000",
		"other-file.backup", // Should not be included
	}

	for _, file := range backupFiles {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("backup"), 0644)
		require.NoError(t, err)
	}

	// Test listing
	mt := NewMigrationTool(false, false)
	found, err := mt.ListBackupFiles(configPath)
	require.NoError(t, err)

	// Should find 3 backup files (excluding other-file.backup)
	assert.Len(t, found, 3)

	// Verify correct files are found
	foundNames := make(map[string]bool)
	for _, path := range found {
		foundNames[filepath.Base(path)] = true
	}

	assert.True(t, foundNames["config.yaml.backup-20240101-120000"])
	assert.True(t, foundNames["config.yaml.backup-20240102-120000"])
	assert.True(t, foundNames["config.yaml.backup-20240103-120000"])
	assert.False(t, foundNames["other-file.backup"])
}

func TestMigrationTool_CleanupBackupFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file
	err := os.WriteFile(configPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Create backup files with different timestamps
	backupFiles := []string{
		"config.yaml.backup-20240101-120000",
		"config.yaml.backup-20240102-120000",
		"config.yaml.backup-20240103-120000",
		"config.yaml.backup-20240104-120000",
		"config.yaml.backup-20240105-120000",
	}

	for i, file := range backupFiles {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("backup"), 0644)
		require.NoError(t, err)

		// Set different modification times
		modTime := time.Now().Add(time.Duration(i) * time.Hour)
		err = os.Chtimes(path, modTime, modTime)
		require.NoError(t, err)
	}

	// Test cleanup (keep 3 files)
	mt := NewMigrationTool(false, false)
	err = mt.CleanupBackupFiles(configPath, 3)
	require.NoError(t, err)

	// Verify only 3 files remain
	found, err := mt.ListBackupFiles(configPath)
	require.NoError(t, err)
	assert.Len(t, found, 3)

	// Verify the newest files are kept
	foundNames := make(map[string]bool)
	for _, path := range found {
		foundNames[filepath.Base(path)] = true
	}

	// Should keep the 3 newest files
	assert.True(t, foundNames["config.yaml.backup-20240103-120000"])
	assert.True(t, foundNames["config.yaml.backup-20240104-120000"])
	assert.True(t, foundNames["config.yaml.backup-20240105-120000"])
}

func TestMigrationTool_ValidateAllMigrations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid migrated config
	validConfigPath := filepath.Join(tmpDir, "valid.yaml")
	validConfig := `source:
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

	err := os.WriteFile(validConfigPath, []byte(validConfig), 0644)
	require.NoError(t, err)

	// Create invalid migrated config
	invalidConfigPath := filepath.Join(tmpDir, "invalid.yaml")
	invalidConfig := `backup:
  enabled: true
  storage:
    provider: invalid_provider`

	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	require.NoError(t, err)

	// Create migration results
	results := []MigrationResult{
		{
			ConfigPath: validConfigPath,
			Success:    true,
		},
		{
			ConfigPath: invalidConfigPath,
			Success:    true,
		},
		{
			ConfigPath: "/nonexistent/config.yaml",
			Success:    false, // Should be skipped
		},
	}

	// Test validation
	mt := NewMigrationTool(false, false)
	err = mt.ValidateAllMigrations(results)
	assert.Error(t, err) // Should fail due to invalid config
	assert.Contains(t, err.Error(), "invalid_provider")
}

func TestMigrationTool_PrintMigrationSummary(t *testing.T) {
	results := []MigrationResult{
		{
			ConfigPath:    "/path/to/config1.yaml",
			Success:       true,
			AlreadyExists: false,
			BackupPath:    "/path/to/config1.yaml.backup",
		},
		{
			ConfigPath:    "/path/to/config2.yaml",
			Success:       true,
			AlreadyExists: true,
		},
		{
			ConfigPath: "/path/to/config3.yaml",
			Success:    false,
			Error:      assert.AnError,
		},
	}

	mt := NewMigrationTool(false, true) // verbose enabled

	// This test mainly ensures the function doesn't panic
	// In a real scenario, you might want to capture stdout to verify output
	mt.PrintMigrationSummary(results)
}

func TestMigrationTool_GenerateBackupPath(t *testing.T) {
	mt := NewMigrationTool(false, false)

	configPath := "/path/to/config.yaml"
	backupPath := mt.generateBackupPath(configPath)

	assert.Contains(t, backupPath, configPath)
	assert.Contains(t, backupPath, ".backup-")
	assert.Regexp(t, `\.backup-\d{8}-\d{6}$`, backupPath)
}

func TestMigrationTool_GetDefaultBackupConfig(t *testing.T) {
	mt := NewMigrationTool(false, false)
	config := mt.getDefaultBackupConfig()

	// Verify structure
	assert.Equal(t, false, config["enabled"])

	storage, ok := config["storage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "local", storage["provider"])

	local, ok := storage["local"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "./backups", local["base_path"])
	assert.Equal(t, "0755", local["permissions"])

	retention, ok := config["retention"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 10, retention["max_backups"])
	assert.Equal(t, "24h", retention["cleanup_interval"])

	compression, ok := config["compression"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, compression["enabled"])
	assert.Equal(t, "gzip", compression["algorithm"])

	encryption, ok := config["encryption"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, encryption["enabled"])
	assert.Equal(t, "env", encryption["key_source"])

	validation, ok := config["validation"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, validation["enabled"])
	assert.Equal(t, "sha256", validation["checksum_algorithm"])
}

// Benchmark tests
func BenchmarkMigrationTool_MigrateConfiguration(b *testing.B) {
	tmpDir := b.TempDir()

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
dry_run: false
verbose: false`

	mt := NewMigrationTool(false, false)

	for i := 0; i < b.N; i++ {
		configPath := filepath.Join(tmpDir, "config_"+string(rune(i))+".yaml")

		err := os.WriteFile(configPath, []byte(config), 0644)
		require.NoError(b, err)

		result := mt.MigrateConfiguration(configPath)
		require.True(b, result.Success)
	}
}

func BenchmarkMigrationTool_DiscoverConfigurations(b *testing.B) {
	mt := NewMigrationTool(false, false)

	for i := 0; i < b.N; i++ {
		_, err := mt.DiscoverConfigurations()
		require.NoError(b, err)
	}
}
