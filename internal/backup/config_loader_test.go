package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigLoader_LoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "backup.yaml")

	configYAML := `
storage:
  provider: LOCAL
  local:
    base_path: "/tmp/test-backups"
    permissions: 0755

retention:
  max_backups: 5
  cleanup_interval: 12h

compression:
  enabled: true
  algorithm: GZIP
  level: 9
  threshold: 2048

encryption:
  enabled: false

validation:
  enabled: true
  checksum_algorithm: sha256
  validation_timeout: 10m
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loader := NewConfigLoader(configPath)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify loaded values
	if config.Storage.Provider != StorageProviderLocal {
		t.Errorf("Expected provider LOCAL, got %s", config.Storage.Provider)
	}

	if config.Storage.Local.BasePath != "/tmp/test-backups" {
		t.Errorf("Expected base path /tmp/test-backups, got %s", config.Storage.Local.BasePath)
	}

	if config.Retention.MaxBackups != 5 {
		t.Errorf("Expected max backups 5, got %d", config.Retention.MaxBackups)
	}

	if config.Retention.CleanupInterval != 12*time.Hour {
		t.Errorf("Expected cleanup interval 12h, got %v", config.Retention.CleanupInterval)
	}

	if !config.Compression.Enabled {
		t.Error("Expected compression to be enabled")
	}

	if config.Compression.Level != 9 {
		t.Errorf("Expected compression level 9, got %d", config.Compression.Level)
	}

	if config.Validation.ValidationTimeout != 10*time.Minute {
		t.Errorf("Expected validation timeout 10m, got %v", config.Validation.ValidationTimeout)
	}
}

func TestConfigLoader_LoadConfig_NonExistentFile(t *testing.T) {
	loader := NewConfigLoader("/non/existent/path.yaml")
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() should not fail for non-existent file: %v", err)
	}

	// Should have defaults
	if config.Storage.Provider != StorageProviderLocal {
		t.Errorf("Expected default provider LOCAL, got %s", config.Storage.Provider)
	}

	if config.Retention.MaxBackups != 10 {
		t.Errorf("Expected default max backups 10, got %d", config.Retention.MaxBackups)
	}
}

func TestConfigLoader_LoadConfig_WithEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("BACKUP_MAX_BACKUPS", "15")
	os.Setenv("BACKUP_COMPRESSION_ENABLED", "true")
	os.Setenv("BACKUP_COMPRESSION_ALGORITHM", "LZ4")
	os.Setenv("BACKUP_COMPRESSION_LEVEL", "1")
	defer func() {
		os.Unsetenv("BACKUP_MAX_BACKUPS")
		os.Unsetenv("BACKUP_COMPRESSION_ENABLED")
		os.Unsetenv("BACKUP_COMPRESSION_ALGORITHM")
		os.Unsetenv("BACKUP_COMPRESSION_LEVEL")
	}()

	// Create a temporary config file with different values
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "backup.yaml")

	configYAML := `
retention:
  max_backups: 5

compression:
  enabled: false
  algorithm: GZIP
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loader := NewConfigLoader(configPath)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Environment variables should override file values
	if config.Retention.MaxBackups != 15 {
		t.Errorf("Expected max backups 15 (from env), got %d", config.Retention.MaxBackups)
	}

	if !config.Compression.Enabled {
		t.Error("Expected compression to be enabled (from env)")
	}

	if config.Compression.Algorithm != CompressionTypeLZ4 {
		t.Errorf("Expected compression algorithm LZ4 (from env), got %s", config.Compression.Algorithm)
	}
}

func TestConfigLoader_SaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "backup.yaml")

	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderS3,
			S3: &S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-key",
				SecretKey: "test-secret",
			},
		},
		Retention: RetentionConfig{
			MaxBackups:      20,
			CleanupInterval: 6 * time.Hour,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: CompressionTypeZstd,
			Level:     5,
			Threshold: 4096,
		},
		Encryption: EncryptionConfig{
			Enabled:   true,
			KeySource: "env",
			KeyEnvVar: "MY_BACKUP_KEY",
		},
		Validation: ValidationConfig{
			Enabled:           true,
			ChecksumAlgorithm: "sha256",
			ValidationTimeout: 15 * time.Minute,
		},
	}

	loader := NewConfigLoader(configPath)
	err := loader.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the saved config and verify
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Storage.Provider != StorageProviderS3 {
		t.Errorf("Expected provider S3, got %s", loadedConfig.Storage.Provider)
	}

	if loadedConfig.Storage.S3.Bucket != "test-bucket" {
		t.Errorf("Expected bucket test-bucket, got %s", loadedConfig.Storage.S3.Bucket)
	}

	if loadedConfig.Retention.MaxBackups != 20 {
		t.Errorf("Expected max backups 20, got %d", loadedConfig.Retention.MaxBackups)
	}

	if loadedConfig.Compression.Algorithm != CompressionTypeZstd {
		t.Errorf("Expected algorithm ZSTD, got %s", loadedConfig.Compression.Algorithm)
	}
}

func TestConfigLoader_SaveConfig_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "backup.yaml")

	// Create invalid config
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: "INVALID_PROVIDER",
		},
	}

	loader := NewConfigLoader(configPath)
	err := loader.SaveConfig(config)
	if err == nil {
		t.Error("SaveConfig() should fail for invalid configuration")
	}
}

func TestLoadConfigFromBytes(t *testing.T) {
	configYAML := `
storage:
  provider: AZURE
  azure:
    account_name: "testaccount"
    account_key: "testkey"
    container_name: "backups"

retention:
  max_backups: 25
  keep_daily: 14

compression:
  enabled: true
  algorithm: LZ4
  level: 3

validation:
  checksum_algorithm: md5
`

	config, err := LoadConfigFromBytes([]byte(configYAML))
	if err != nil {
		t.Fatalf("LoadConfigFromBytes() failed: %v", err)
	}

	if config.Storage.Provider != StorageProviderAzure {
		t.Errorf("Expected provider AZURE, got %s", config.Storage.Provider)
	}

	if config.Storage.Azure.AccountName != "testaccount" {
		t.Errorf("Expected account name testaccount, got %s", config.Storage.Azure.AccountName)
	}

	if config.Retention.MaxBackups != 25 {
		t.Errorf("Expected max backups 25, got %d", config.Retention.MaxBackups)
	}

	if config.Compression.Algorithm != CompressionTypeLZ4 {
		t.Errorf("Expected algorithm LZ4, got %s", config.Compression.Algorithm)
	}

	if config.Validation.ChecksumAlgorithm != "md5" {
		t.Errorf("Expected checksum algorithm md5, got %s", config.Validation.ChecksumAlgorithm)
	}
}

func TestLoadConfigFromBytes_InvalidYAML(t *testing.T) {
	invalidYAML := `
storage:
  provider: LOCAL
  invalid_yaml: [
`

	_, err := LoadConfigFromBytes([]byte(invalidYAML))
	if err == nil {
		t.Error("LoadConfigFromBytes() should fail for invalid YAML")
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	config := GenerateDefaultConfig()

	if config.Storage.Provider != StorageProviderLocal {
		t.Errorf("Expected default provider LOCAL, got %s", config.Storage.Provider)
	}

	if config.Storage.Local.BasePath != "./backups" {
		t.Errorf("Expected default base path ./backups, got %s", config.Storage.Local.BasePath)
	}

	if config.Retention.MaxBackups != 10 {
		t.Errorf("Expected default max backups 10, got %d", config.Retention.MaxBackups)
	}

	if !config.Compression.Enabled {
		t.Error("Expected compression to be enabled by default")
	}

	if config.Compression.Algorithm != CompressionTypeGzip {
		t.Errorf("Expected default algorithm GZIP, got %s", config.Compression.Algorithm)
	}

	if config.Encryption.Enabled {
		t.Error("Expected encryption to be disabled by default")
	}

	if !config.Validation.Enabled {
		t.Error("Expected validation to be enabled by default")
	}

	// Validate the generated config
	if err := config.Validate(); err != nil {
		t.Errorf("Generated default config should be valid: %v", err)
	}
}

func TestGenerateDefaultConfigYAML(t *testing.T) {
	yamlData, err := GenerateDefaultConfigYAML()
	if err != nil {
		t.Fatalf("GenerateDefaultConfigYAML() failed: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Generated YAML should not be empty")
	}

	// Try to parse the generated YAML
	config, err := LoadConfigFromBytes(yamlData)
	if err != nil {
		t.Fatalf("Generated YAML should be parseable: %v", err)
	}

	// Verify it's valid
	if err := config.Validate(); err != nil {
		t.Errorf("Generated config should be valid: %v", err)
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "./base-backups",
				Permissions: 0755,
			},
		},
		Retention: RetentionConfig{
			MaxBackups:      10,
			CleanupInterval: 24 * time.Hour,
		},
		Compression: CompressionConfig{
			Enabled:   false,
			Algorithm: CompressionTypeGzip,
			Level:     6,
		},
	}

	override := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderS3,
			S3: &S3Config{
				Bucket: "override-bucket",
				Region: "us-west-2",
			},
		},
		Retention: RetentionConfig{
			MaxBackups: 20,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: CompressionTypeLZ4,
		},
	}

	merged := MergeConfigs(base, override)

	// Storage should be overridden
	if merged.Storage.Provider != StorageProviderS3 {
		t.Errorf("Expected merged provider S3, got %s", merged.Storage.Provider)
	}

	if merged.Storage.S3.Bucket != "override-bucket" {
		t.Errorf("Expected merged bucket override-bucket, got %s", merged.Storage.S3.Bucket)
	}

	// Retention should be partially merged
	if merged.Retention.MaxBackups != 20 {
		t.Errorf("Expected merged max backups 20, got %d", merged.Retention.MaxBackups)
	}

	if merged.Retention.CleanupInterval != 24*time.Hour {
		t.Errorf("Expected merged cleanup interval to remain 24h, got %v", merged.Retention.CleanupInterval)
	}

	// Compression should be overridden
	if !merged.Compression.Enabled {
		t.Error("Expected merged compression to be enabled")
	}

	if merged.Compression.Algorithm != CompressionTypeLZ4 {
		t.Errorf("Expected merged algorithm LZ4, got %s", merged.Compression.Algorithm)
	}

	// Level should remain from base since not overridden
	if merged.Compression.Level != 6 {
		t.Errorf("Expected merged level to remain 6, got %d", merged.Compression.Level)
	}
}
