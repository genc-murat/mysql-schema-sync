package backup

import (
	"os"
	"testing"
	"time"
)

func TestBackupSystemConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *BackupSystemConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &BackupSystemConfig{
				Storage: StorageConfig{
					Provider: StorageProviderLocal,
					Local: &LocalConfig{
						BasePath:    "/tmp/backups",
						Permissions: 0755,
					},
				},
				Retention: RetentionConfig{
					MaxBackups:      10,
					CleanupInterval: 24 * time.Hour,
				},
				Compression: CompressionConfig{
					Enabled:   true,
					Algorithm: CompressionTypeGzip,
					Level:     6,
					Threshold: 1024,
				},
				Encryption: EncryptionConfig{
					Enabled: false,
				},
				Validation: ValidationConfig{
					Enabled:           true,
					ChecksumAlgorithm: "sha256",
					ValidationTimeout: 5 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid storage configuration",
			config: &BackupSystemConfig{
				Storage: StorageConfig{
					Provider: "INVALID",
				},
				Retention: RetentionConfig{
					MaxBackups: 10,
				},
				Compression: CompressionConfig{},
				Encryption:  EncryptionConfig{},
				Validation:  ValidationConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BackupSystemConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBackupSystemConfig_SetDefaults(t *testing.T) {
	config := &BackupSystemConfig{}
	config.SetDefaults()

	// Check that defaults are set
	if config.Storage.Provider != StorageProviderLocal {
		t.Errorf("Expected default storage provider to be LOCAL, got %s", config.Storage.Provider)
	}

	if config.Retention.MaxBackups != 10 {
		t.Errorf("Expected default max backups to be 10, got %d", config.Retention.MaxBackups)
	}

	if config.Validation.ChecksumAlgorithm != "sha256" {
		t.Errorf("Expected default checksum algorithm to be sha256, got %s", config.Validation.ChecksumAlgorithm)
	}
}

func TestRetentionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RetentionConfig
		wantErr bool
	}{
		{
			name: "valid retention config",
			config: &RetentionConfig{
				MaxBackups:      10,
				CleanupInterval: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "negative max backups",
			config: &RetentionConfig{
				MaxBackups: -1,
			},
			wantErr: true,
		},
		{
			name: "no retention policies",
			config: &RetentionConfig{
				MaxBackups:  0,
				MaxAge:      0,
				KeepDaily:   0,
				KeepWeekly:  0,
				KeepMonthly: 0,
			},
			wantErr: true,
		},
		{
			name: "valid with keep policies",
			config: &RetentionConfig{
				KeepDaily:       7,
				CleanupInterval: 24 * time.Hour,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RetentionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetentionConfig_LoadFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("BACKUP_MAX_BACKUPS", "20")
	os.Setenv("BACKUP_MAX_AGE", "720h")
	os.Setenv("BACKUP_KEEP_DAILY", "14")
	defer func() {
		os.Unsetenv("BACKUP_MAX_BACKUPS")
		os.Unsetenv("BACKUP_MAX_AGE")
		os.Unsetenv("BACKUP_KEEP_DAILY")
	}()

	config := &RetentionConfig{}
	config.LoadFromEnvironment()

	if config.MaxBackups != 20 {
		t.Errorf("Expected MaxBackups to be 20, got %d", config.MaxBackups)
	}

	if config.MaxAge != 720*time.Hour {
		t.Errorf("Expected MaxAge to be 720h, got %v", config.MaxAge)
	}

	if config.KeepDaily != 14 {
		t.Errorf("Expected KeepDaily to be 14, got %d", config.KeepDaily)
	}
}

func TestCompressionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CompressionConfig
		wantErr bool
	}{
		{
			name: "valid gzip compression",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeGzip,
				Level:     6,
				Threshold: 1024,
			},
			wantErr: false,
		},
		{
			name: "invalid gzip level",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeGzip,
				Level:     15, // Invalid for gzip
				Threshold: 1024,
			},
			wantErr: true,
		},
		{
			name: "valid lz4 compression",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeLZ4,
				Level:     1,
				Threshold: 1024,
			},
			wantErr: false,
		},
		{
			name: "invalid lz4 level",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeLZ4,
				Level:     20, // Invalid for lz4
				Threshold: 1024,
			},
			wantErr: true,
		},
		{
			name: "valid zstd compression",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeZstd,
				Level:     3,
				Threshold: 1024,
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: "INVALID",
				Level:     6,
				Threshold: 1024,
			},
			wantErr: true,
		},
		{
			name: "negative threshold",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeGzip,
				Level:     6,
				Threshold: -1,
			},
			wantErr: true,
		},
		{
			name: "disabled compression",
			config: &CompressionConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CompressionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressionConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *CompressionConfig
		expected *CompressionConfig
	}{
		{
			name: "enabled with no algorithm",
			config: &CompressionConfig{
				Enabled: true,
			},
			expected: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeGzip,
				Level:     6,
				Threshold: 1024,
			},
		},
		{
			name: "enabled with lz4",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeLZ4,
			},
			expected: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeLZ4,
				Level:     1,
				Threshold: 1024,
			},
		},
		{
			name: "enabled with zstd",
			config: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeZstd,
			},
			expected: &CompressionConfig{
				Enabled:   true,
				Algorithm: CompressionTypeZstd,
				Level:     3,
				Threshold: 1024,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			if tt.config.Algorithm != tt.expected.Algorithm {
				t.Errorf("Expected algorithm %s, got %s", tt.expected.Algorithm, tt.config.Algorithm)
			}
			if tt.config.Level != tt.expected.Level {
				t.Errorf("Expected level %d, got %d", tt.expected.Level, tt.config.Level)
			}
			if tt.config.Threshold != tt.expected.Threshold {
				t.Errorf("Expected threshold %d, got %d", tt.expected.Threshold, tt.config.Threshold)
			}
		})
	}
}

func TestEncryptionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EncryptionConfig
		wantErr bool
	}{
		{
			name: "disabled encryption",
			config: &EncryptionConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid env key source",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "env",
				KeyEnvVar: "BACKUP_KEY",
			},
			wantErr: false,
		},
		{
			name: "valid file key source",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "file",
				KeyPath:   "/path/to/key",
			},
			wantErr: false,
		},
		{
			name: "missing key source",
			config: &EncryptionConfig{
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "invalid key source",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "invalid",
			},
			wantErr: true,
		},
		{
			name: "env key source without env var",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "env",
			},
			wantErr: true,
		},
		{
			name: "file key source without path",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "file",
			},
			wantErr: true,
		},
		{
			name: "rotation enabled without days",
			config: &EncryptionConfig{
				Enabled:         true,
				KeySource:       "env",
				KeyEnvVar:       "BACKUP_KEY",
				RotationEnabled: true,
				RotationDays:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionConfig_GetEncryptionKey(t *testing.T) {
	tests := []struct {
		name    string
		config  *EncryptionConfig
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "disabled encryption",
			config: &EncryptionConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "env key source with valid key",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "env",
				KeyEnvVar: "TEST_BACKUP_KEY",
			},
			envVars: map[string]string{
				"TEST_BACKUP_KEY": "1234567890123456789012345678901234567890123456789012345678901234", // 32 bytes hex-encoded (64 chars)
			},
			wantErr: false,
		},
		{
			name: "env key source with missing key",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "env",
				KeyEnvVar: "MISSING_KEY",
			},
			wantErr: true,
		},
		{
			name: "env key source with invalid key length",
			config: &EncryptionConfig{
				Enabled:   true,
				KeySource: "env",
				KeyEnvVar: "SHORT_KEY",
			},
			envVars: map[string]string{
				"SHORT_KEY": "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			key, err := tt.config.GetEncryptionKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptionConfig.GetEncryptionKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.config.Enabled {
				if len(key) != 32 {
					t.Errorf("Expected key length 32, got %d", len(key))
				}
			}
		})
	}
}

func TestValidationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ValidationConfig
		wantErr bool
	}{
		{
			name: "valid validation config",
			config: &ValidationConfig{
				Enabled:           true,
				ChecksumAlgorithm: "sha256",
				ValidationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "invalid checksum algorithm",
			config: &ValidationConfig{
				Enabled:           true,
				ChecksumAlgorithm: "invalid",
				ValidationTimeout: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &ValidationConfig{
				Enabled:           true,
				ChecksumAlgorithm: "sha256",
				ValidationTimeout: -1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "disabled validation",
			config: &ValidationConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationConfig_SetDefaults(t *testing.T) {
	config := &ValidationConfig{}
	config.SetDefaults()

	if !config.Enabled {
		t.Error("Expected validation to be enabled by default")
	}

	if config.ChecksumAlgorithm != "sha256" {
		t.Errorf("Expected default checksum algorithm to be sha256, got %s", config.ChecksumAlgorithm)
	}

	if config.ValidationTimeout != 5*time.Minute {
		t.Errorf("Expected default validation timeout to be 5m, got %v", config.ValidationTimeout)
	}

	if !config.ValidateOnCreate {
		t.Error("Expected ValidateOnCreate to be true by default")
	}

	if !config.ValidateOnRestore {
		t.Error("Expected ValidateOnRestore to be true by default")
	}

	if !config.DryRunValidation {
		t.Error("Expected DryRunValidation to be true by default")
	}
}

func TestStorageConfig_SetDefaults(t *testing.T) {
	config := &StorageConfig{}
	config.SetDefaults()

	if config.Provider != StorageProviderLocal {
		t.Errorf("Expected default provider to be LOCAL, got %s", config.Provider)
	}

	if config.Local == nil {
		t.Error("Expected local config to be initialized")
	}

	if config.Local.BasePath != "./backups" {
		t.Errorf("Expected default base path to be ./backups, got %s", config.Local.BasePath)
	}

	if config.Local.Permissions != 0755 {
		t.Errorf("Expected default permissions to be 0755, got %o", config.Local.Permissions)
	}
}

func TestLocalConfig_LoadFromEnvironment(t *testing.T) {
	os.Setenv("BACKUP_LOCAL_BASE_PATH", "/custom/path")
	os.Setenv("BACKUP_LOCAL_PERMISSIONS", "0644")
	defer func() {
		os.Unsetenv("BACKUP_LOCAL_BASE_PATH")
		os.Unsetenv("BACKUP_LOCAL_PERMISSIONS")
	}()

	config := &LocalConfig{}
	config.LoadFromEnvironment()

	if config.BasePath != "/custom/path" {
		t.Errorf("Expected base path to be /custom/path, got %s", config.BasePath)
	}

	if config.Permissions != 0644 {
		t.Errorf("Expected permissions to be 0644, got %o", config.Permissions)
	}
}

func TestS3Config_LoadFromEnvironment(t *testing.T) {
	os.Setenv("BACKUP_S3_BUCKET", "test-bucket")
	os.Setenv("BACKUP_S3_REGION", "us-west-2")
	os.Setenv("BACKUP_S3_ACCESS_KEY", "test-access-key")
	os.Setenv("BACKUP_S3_SECRET_KEY", "test-secret-key")
	defer func() {
		os.Unsetenv("BACKUP_S3_BUCKET")
		os.Unsetenv("BACKUP_S3_REGION")
		os.Unsetenv("BACKUP_S3_ACCESS_KEY")
		os.Unsetenv("BACKUP_S3_SECRET_KEY")
	}()

	config := &S3Config{}
	config.LoadFromEnvironment()

	if config.Bucket != "test-bucket" {
		t.Errorf("Expected bucket to be test-bucket, got %s", config.Bucket)
	}

	if config.Region != "us-west-2" {
		t.Errorf("Expected region to be us-west-2, got %s", config.Region)
	}

	if config.AccessKey != "test-access-key" {
		t.Errorf("Expected access key to be test-access-key, got %s", config.AccessKey)
	}

	if config.SecretKey != "test-secret-key" {
		t.Errorf("Expected secret key to be test-secret-key, got %s", config.SecretKey)
	}
}
