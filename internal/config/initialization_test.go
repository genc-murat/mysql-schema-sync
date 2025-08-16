package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupSystemInitializer_InitializeBackupSystem(t *testing.T) {
	tests := []struct {
		name           string
		config         *BackupConfig
		setupEnv       func()
		cleanupEnv     func()
		expectSuccess  bool
		expectWarnings bool
	}{
		{
			name: "successful local storage initialization",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath:    "./test_backups",
						Permissions: "0755",
					},
				},
				Retention: RetentionConfig{
					MaxBackups: 10,
				},
				Validation: ValidationConfig{
					Enabled: true,
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() { os.RemoveAll("./test_backups") },
			expectSuccess:  true,
			expectWarnings: false,
		},
		{
			name: "S3 storage with missing credentials",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "s3",
					S3: &S3Config{
						Bucket: "test-bucket",
						Region: "us-east-1",
					},
				},
				Retention: RetentionConfig{
					MaxBackups: 10,
				},
				Validation: ValidationConfig{
					Enabled: true,
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectSuccess:  true,
			expectWarnings: true,
		},
		{
			name: "encryption enabled with missing key",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: "./test_backups",
					},
				},
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "env",
					KeyEnvVar: "TEST_ENCRYPTION_KEY",
				},
				Retention: RetentionConfig{
					MaxBackups: 10,
				},
				Validation: ValidationConfig{
					Enabled: true,
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() { os.RemoveAll("./test_backups") },
			expectSuccess:  true,
			expectWarnings: true,
		},
		{
			name: "invalid storage provider",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "invalid",
				},
			},
			setupEnv:      func() {},
			cleanupEnv:    func() {},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			bsi := NewBackupSystemInitializer(tt.config, false)
			result, err := bsi.InitializeBackupSystem()

			require.NoError(t, err)
			assert.Equal(t, tt.expectSuccess, result.Success)

			if tt.expectWarnings {
				assert.Greater(t, len(result.Warnings), 0)
			}

			if tt.expectSuccess {
				assert.True(t, result.ConfigValid)
			}
		})
	}
}

func TestBackupSystemInitializer_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      *BackupConfig
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "valid configuration",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: "./test_backups",
					},
				},
				Retention: RetentionConfig{
					MaxBackups: 10,
				},
				Validation: ValidationConfig{
					Enabled: true,
				},
			},
			setupEnv:    func() {},
			cleanupEnv:  func() {},
			expectError: false,
		},
		{
			name: "encryption with file key source but no path",
			config: &BackupConfig{
				Enabled: true,
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: "./test_backups",
					},
				},
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "file",
					KeyPath:   "",
				},
				Validation: ValidationConfig{
					Enabled: true,
				},
			},
			setupEnv:    func() {},
			cleanupEnv:  func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			bsi := NewBackupSystemInitializer(tt.config, false)
			result := &InitializationResult{}
			err := bsi.validateConfiguration(result)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackupSystemInitializer_InitializeLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath:    tmpDir,
				Permissions: "0755",
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.initializeLocalStorage(result)
	assert.NoError(t, err)

	// Verify directory was created and is writable
	assert.DirExists(t, tmpDir)
}

func TestBackupSystemInitializer_InitializeLocalStorage_InvalidPath(t *testing.T) {
	// On Windows, we need a path that will actually fail
	// Use a path with invalid characters
	invalidPath := "C:\\invalid<>path\\that\\cannot\\be\\created"

	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: invalidPath,
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.initializeLocalStorage(result)
	assert.Error(t, err)
}

func TestBackupSystemInitializer_InitializeS3Storage(t *testing.T) {
	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "s3",
			S3: &S3Config{
				Bucket: "test-bucket",
				Region: "us-east-1",
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.initializeS3Storage(result)
	assert.NoError(t, err)

	// Should have warnings about missing credentials
	assert.Greater(t, len(result.Warnings), 0)
}

func TestBackupSystemInitializer_InitializeAzureStorage(t *testing.T) {
	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "azure",
			Azure: &AzureConfig{
				AccountName:   "testaccount",
				ContainerName: "backups",
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.initializeAzureStorage(result)
	assert.NoError(t, err)
}

func TestBackupSystemInitializer_InitializeGCSStorage(t *testing.T) {
	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "gcs",
			GCS: &GCSConfig{
				Bucket:    "test-bucket",
				ProjectID: "test-project",
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.initializeGCSStorage(result)
	assert.NoError(t, err)
}

func TestBackupSystemInitializer_CheckPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: tmpDir,
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.checkPermissions(result)
	assert.NoError(t, err)
}

func TestBackupSystemInitializer_CheckPermissions_NonExistentPath(t *testing.T) {
	config := &BackupConfig{
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: "/nonexistent/path",
			},
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	err := bsi.checkPermissions(result)
	assert.Error(t, err)
}

func TestBackupSystemInitializer_TestConnectivity(t *testing.T) {
	tests := []struct {
		name        string
		config      *BackupConfig
		expectError bool
	}{
		{
			name: "valid S3 config",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "s3",
					S3: &S3Config{
						Bucket: "test-bucket",
						Region: "us-east-1",
					},
				},
			},
			expectError: false,
		},
		{
			name: "S3 config missing bucket",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "s3",
					S3: &S3Config{
						Region: "us-east-1",
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid Azure config",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "azure",
					Azure: &AzureConfig{
						AccountName:   "testaccount",
						ContainerName: "backups",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid GCS config",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "gcs",
					GCS: &GCSConfig{
						Bucket: "test-bucket",
					},
				},
			},
			expectError: false,
		},
		{
			name: "local storage (no connectivity test needed)",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: "./test",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsi := NewBackupSystemInitializer(tt.config, false)
			result := &InitializationResult{}

			err := bsi.testConnectivity(result)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackupSystemInitializer_RunHealthCheck(t *testing.T) {
	tmpDir := t.TempDir()

	config := &BackupConfig{
		Enabled: true,
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: tmpDir,
			},
		},
		Retention: RetentionConfig{
			MaxBackups: 10,
		},
		Validation: ValidationConfig{
			Enabled: true,
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result, err := bsi.RunHealthCheck()

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "healthy", result.OverallHealth)
	assert.Equal(t, "healthy", result.ComponentStatus["configuration"])
	assert.Equal(t, "healthy", result.ComponentStatus["storage"])
	assert.True(t, result.Timestamp.Before(time.Now().Add(time.Second)))
}

func TestBackupSystemInitializer_RunHealthCheck_UnhealthyConfig(t *testing.T) {
	config := &BackupConfig{
		Enabled: true,
		Storage: StorageConfig{
			Provider: "invalid",
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result, err := bsi.RunHealthCheck()

	require.NoError(t, err)
	// The overall health should be degraded or unhealthy
	assert.Contains(t, []string{"unhealthy", "degraded"}, result.OverallHealth)
	assert.Equal(t, "unhealthy", result.ComponentStatus["configuration"])
	assert.Greater(t, len(result.Issues), 0)
}

func TestBackupSystemInitializer_CheckStorageHealth(t *testing.T) {
	tests := []struct {
		name           string
		config         *BackupConfig
		expectedHealth string
	}{
		{
			name: "healthy local storage",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: t.TempDir(),
					},
				},
			},
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy local storage - missing config",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "local",
					Local:    nil,
				},
			},
			expectedHealth: "unhealthy",
		},
		{
			name: "unhealthy local storage - nonexistent path",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "local",
					Local: &LocalConfig{
						BasePath: "/nonexistent/path",
					},
				},
			},
			expectedHealth: "unhealthy",
		},
		{
			name: "healthy S3 storage",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "s3",
					S3: &S3Config{
						Bucket: "test-bucket",
						Region: "us-east-1",
					},
				},
			},
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy S3 storage - missing config",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "s3",
					S3:       nil,
				},
			},
			expectedHealth: "unhealthy",
		},
		{
			name: "invalid storage provider",
			config: &BackupConfig{
				Storage: StorageConfig{
					Provider: "invalid",
				},
			},
			expectedHealth: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bsi := NewBackupSystemInitializer(tt.config, false)
			health := bsi.checkStorageHealth()
			assert.Equal(t, tt.expectedHealth, health)
		})
	}
}

func TestBackupSystemInitializer_CheckEncryptionHealth(t *testing.T) {
	tests := []struct {
		name           string
		config         *BackupConfig
		setupEnv       func()
		cleanupEnv     func()
		expectedHealth string
	}{
		{
			name: "encryption disabled",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled: false,
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "healthy",
		},
		{
			name: "healthy env key source",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "env",
					KeyEnvVar: "TEST_KEY",
				},
			},
			setupEnv:       func() { os.Setenv("TEST_KEY", "test_value") },
			cleanupEnv:     func() { os.Unsetenv("TEST_KEY") },
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy env key source - missing key",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "env",
					KeyEnvVar: "MISSING_KEY",
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "unhealthy",
		},
		{
			name: "healthy file key source",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "file",
					KeyPath:   createTempKeyFile(t),
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy file key source - missing file",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "file",
					KeyPath:   "/nonexistent/key/file",
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "unhealthy",
		},
		{
			name: "unhealthy file key source - no path",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "file",
					KeyPath:   "",
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "unhealthy",
		},
		{
			name: "external key source",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "external",
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "healthy",
		},
		{
			name: "invalid key source",
			config: &BackupConfig{
				Encryption: EncryptionConfig{
					Enabled:   true,
					KeySource: "invalid",
				},
			},
			setupEnv:       func() {},
			cleanupEnv:     func() {},
			expectedHealth: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			bsi := NewBackupSystemInitializer(tt.config, false)
			health := bsi.checkEncryptionHealth()
			assert.Equal(t, tt.expectedHealth, health)
		})
	}
}

func createTempKeyFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp("", "test_key_*")
	require.NoError(t, err)

	_, err = tmpFile.WriteString("test_encryption_key")
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// Clean up the file after test
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestBackupSystemInitializer_GenerateRecommendations(t *testing.T) {
	config := &BackupConfig{
		Enabled: true,
		Storage: StorageConfig{
			Provider: "local",
		},
		Encryption: EncryptionConfig{
			Enabled: false,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: "gzip",
			Level:     9, // High compression level
		},
		Retention: RetentionConfig{
			MaxBackups: 0,
			MaxAge:     "",
		},
		Validation: ValidationConfig{
			Enabled:           true,
			ValidationTimeout: "",
		},
	}

	bsi := NewBackupSystemInitializer(config, false)
	result := &InitializationResult{}

	bsi.generateRecommendations(result)

	// Should have multiple recommendations
	assert.Greater(t, len(result.RecommendedFixes), 0)

	// Check for specific recommendations
	recommendations := strings.Join(result.RecommendedFixes, " ")
	assert.Contains(t, recommendations, "encryption")
	assert.Contains(t, recommendations, "compression level")
	assert.Contains(t, recommendations, "retention policies")
	assert.Contains(t, recommendations, "validation timeout")
}

func TestBackupSystemInitializer_CreateSetupWizard(t *testing.T) {
	config := &BackupConfig{}
	bsi := NewBackupSystemInitializer(config, false)

	wizard := bsi.CreateSetupWizard()
	assert.NotNil(t, wizard)
	assert.Equal(t, bsi, wizard.initializer)
	assert.False(t, wizard.verbose)
}

// Note: Testing the interactive setup wizard is complex as it requires user input
// In a real-world scenario, you might want to create a mock input interface
// or test individual configuration methods separately

func TestSetupWizard_ConfigureLocalStorage(t *testing.T) {
	wizard := &SetupWizard{verbose: false}

	// This test would require mocking user input
	// For now, we'll test the structure
	assert.NotNil(t, wizard)
}

// Benchmark tests
func BenchmarkBackupSystemInitializer_InitializeBackupSystem(b *testing.B) {
	tmpDir := b.TempDir()

	config := &BackupConfig{
		Enabled: true,
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: tmpDir,
			},
		},
		Retention: RetentionConfig{
			MaxBackups: 10,
		},
		Validation: ValidationConfig{
			Enabled: true,
		},
	}

	bsi := NewBackupSystemInitializer(config, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := bsi.InitializeBackupSystem()
		require.NoError(b, err)
		require.True(b, result.Success)
	}
}

func BenchmarkBackupSystemInitializer_RunHealthCheck(b *testing.B) {
	tmpDir := b.TempDir()

	config := &BackupConfig{
		Enabled: true,
		Storage: StorageConfig{
			Provider: "local",
			Local: &LocalConfig{
				BasePath: tmpDir,
			},
		},
		Retention: RetentionConfig{
			MaxBackups: 10,
		},
		Validation: ValidationConfig{
			Enabled: true,
		},
	}

	bsi := NewBackupSystemInitializer(config, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := bsi.RunHealthCheck()
		require.NoError(b, err)
		require.NotNil(b, result)
	}
}
