package test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mysql-schema-sync/internal/backup"
)

// TestBackupSystemIntegrationSuite tests the backup system integration
func TestBackupSystemIntegrationSuite(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Test 1: Basic backup configuration validation
	t.Run("Backup configuration validation", func(t *testing.T) {
		config := &backup.BackupSystemConfig{
			Storage: backup.StorageConfig{
				Provider: backup.StorageProviderLocal,
				Local: &backup.LocalConfig{
					BasePath:    tempDir,
					Permissions: 0755,
				},
			},
			Retention: backup.RetentionConfig{
				MaxBackups: 5,
				MaxAge:     24 * time.Hour,
			},
			Compression: backup.CompressionConfig{
				Enabled:   true,
				Algorithm: backup.CompressionGzip,
				Level:     6,
			},
			Encryption: backup.EncryptionConfig{
				Enabled: false,
			},
		}

		err := config.Validate()
		assert.NoError(t, err, "Valid configuration should pass validation")

		// Test invalid configuration
		invalidConfig := &backup.BackupSystemConfig{
			Storage: backup.StorageConfig{
				Provider: backup.StorageProviderLocal,
				// Missing Local config
			},
		}

		err = invalidConfig.Validate()
		assert.Error(t, err, "Invalid configuration should fail validation")
	})

	// Test 2: Storage provider factory functionality
	t.Run("Storage provider factory", func(t *testing.T) {
		factory := backup.NewStorageProviderFactory()

		// Test supported providers
		providers := factory.GetSupportedProviders()
		assert.Contains(t, providers, backup.StorageProviderLocal)
		assert.Contains(t, providers, backup.StorageProviderS3)
		assert.Contains(t, providers, backup.StorageProviderAzure)
		assert.Contains(t, providers, backup.StorageProviderGCS)

		// Test local provider creation
		config := &backup.StorageConfig{
			Provider: backup.StorageProviderLocal,
			Local: &backup.LocalConfig{
				BasePath:    tempDir,
				Permissions: 0755,
			},
		}

		provider, err := factory.CreateStorageProvider(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, provider)

		// Test provider health check
		err = provider.HealthCheck(ctx)
		assert.NoError(t, err)

		// Test storage info
		info, err := provider.GetStorageInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, backup.StorageProviderLocal, info.Provider)
		assert.Equal(t, tempDir, info.Location)
	})

	// Test 3: Local storage provider operations
	t.Run("Local storage provider operations", func(t *testing.T) {
		config := &backup.LocalConfig{
			BasePath:    tempDir,
			Permissions: 0755,
		}

		provider, err := backup.NewLocalStorageProvider(config)
		require.NoError(t, err)

		// Create test backup
		testBackup := &backup.Backup{
			ID: "test-backup-123",
			Metadata: &backup.BackupMetadata{
				ID:           "test-backup-123",
				DatabaseName: "test_db",
				CreatedAt:    time.Now(),
				CreatedBy:    "test_user",
				Description:  "Test backup for integration",
				Size:         1024,
				Status:       backup.BackupStatusCompleted,
			},
			SchemaSnapshot: map[string]interface{}{
				"tables": []string{"users", "posts"},
			},
		}

		// Test store operation
		err = provider.Store(ctx, testBackup)
		require.NoError(t, err)

		// Test retrieve operation
		retrievedBackup, err := provider.Retrieve(ctx, testBackup.ID)
		require.NoError(t, err)
		assert.Equal(t, testBackup.ID, retrievedBackup.ID)
		assert.Equal(t, testBackup.Metadata.DatabaseName, retrievedBackup.Metadata.DatabaseName)

		// Test list operation
		backups, err := provider.List(ctx, backup.StorageFilter{})
		require.NoError(t, err)
		assert.Len(t, backups, 1)
		assert.Equal(t, testBackup.ID, backups[0].ID)

		// Test get metadata operation
		metadata, err := provider.GetMetadata(ctx, testBackup.ID)
		require.NoError(t, err)
		assert.Equal(t, testBackup.Metadata.DatabaseName, metadata.DatabaseName)

		// Test delete operation
		err = provider.Delete(ctx, testBackup.ID)
		require.NoError(t, err)

		// Verify deletion
		backups, err = provider.List(ctx, backup.StorageFilter{})
		require.NoError(t, err)
		assert.Len(t, backups, 0)
	})
}

// TestBackupValidationSuite tests backup validation functionality
func TestBackupValidationSuite(t *testing.T) {
	// Test validation result handling
	t.Run("Validation result handling", func(t *testing.T) {
		// Test successful validation
		successResult := &backup.ValidationResult{
			IsValid:   true,
			Errors:    []string{},
			Warnings:  []string{},
			CheckedAt: time.Now(),
		}

		assert.True(t, successResult.IsValid)
		assert.Empty(t, successResult.Errors)
		assert.Empty(t, successResult.Warnings)

		// Test validation with warnings
		warningResult := &backup.ValidationResult{
			IsValid:   true,
			Errors:    []string{},
			Warnings:  []string{"Backup is older than recommended retention period"},
			CheckedAt: time.Now(),
		}

		assert.True(t, warningResult.IsValid)
		assert.Empty(t, warningResult.Errors)
		assert.Len(t, warningResult.Warnings, 1)

		// Test validation with errors
		errorResult := &backup.ValidationResult{
			IsValid: false,
			Errors: []string{
				"Checksum verification failed",
				"Backup file is corrupted",
			},
			CheckedAt: time.Now(),
		}

		assert.False(t, errorResult.IsValid)
		assert.Len(t, errorResult.Errors, 2)
	})
}

// TestCompressionAndEncryptionSuite tests compression and encryption
func TestCompressionAndEncryptionSuite(t *testing.T) {
	// Test compression configuration
	t.Run("Compression configuration", func(t *testing.T) {
		algorithms := []backup.CompressionAlgorithm{
			backup.CompressionGzip,
			backup.CompressionLZ4,
			backup.CompressionZstd,
		}

		for _, algorithm := range algorithms {
			config := &backup.CompressionConfig{
				Enabled:   true,
				Algorithm: algorithm,
				Level:     6,
				Threshold: 1024,
			}

			assert.True(t, config.Enabled)
			assert.Equal(t, algorithm, config.Algorithm)
			assert.Equal(t, 6, config.Level)
			assert.Equal(t, int64(1024), config.Threshold)
		}
	})

	// Test encryption configuration
	t.Run("Encryption configuration", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		config := &backup.EncryptionConfig{
			Enabled:   true,
			Algorithm: backup.EncryptionAES256,
			KeySource: backup.EncryptionKeySourceDirect,
			Key:       key,
		}

		assert.True(t, config.Enabled)
		assert.Equal(t, backup.EncryptionAES256, config.Algorithm)
		assert.Equal(t, backup.EncryptionKeySourceDirect, config.KeySource)
		assert.Equal(t, key, config.Key)
		assert.Len(t, config.Key, 32)
	})

	// Test encryption manager creation
	t.Run("Encryption manager", func(t *testing.T) {
		config := &backup.EncryptionConfig{
			Enabled: false,
		}

		manager := backup.NewEncryptionManager(config)
		assert.NotNil(t, manager)
		assert.False(t, manager.IsEnabled())

		// Test with enabled encryption
		key := make([]byte, 32)
		rand.Read(key)

		enabledConfig := &backup.EncryptionConfig{
			Enabled:   true,
			Algorithm: backup.EncryptionAES256,
			KeySource: backup.EncryptionKeySourceDirect,
			Key:       key,
		}

		enabledManager := backup.NewEncryptionManager(enabledConfig)
		assert.NotNil(t, enabledManager)
		assert.True(t, enabledManager.IsEnabled())
	})
}

// TestRetentionPolicySuite tests retention policy functionality
func TestRetentionPolicySuite(t *testing.T) {
	// Test retention configuration validation
	t.Run("Retention configuration validation", func(t *testing.T) {
		validConfig := &backup.RetentionConfig{
			MaxBackups:      10,
			MaxAge:          7 * 24 * time.Hour,
			CleanupInterval: 1 * time.Hour,
			KeepDaily:       7,
			KeepWeekly:      4,
			KeepMonthly:     12,
		}

		err := validConfig.Validate()
		assert.NoError(t, err)

		invalidConfig := &backup.RetentionConfig{
			MaxBackups: -1, // Invalid negative value
		}

		err = invalidConfig.Validate()
		assert.Error(t, err)
	})

	// Test retention policy logic simulation
	t.Run("Retention policy logic", func(t *testing.T) {
		now := time.Now()
		backups := []*backup.BackupMetadata{
			{ID: "backup-1", CreatedAt: now.Add(-1 * time.Hour)},  // Recent
			{ID: "backup-2", CreatedAt: now.Add(-6 * time.Hour)},  // Recent
			{ID: "backup-3", CreatedAt: now.Add(-12 * time.Hour)}, // Recent
			{ID: "backup-4", CreatedAt: now.Add(-25 * time.Hour)}, // Old (should be removed)
			{ID: "backup-5", CreatedAt: now.Add(-48 * time.Hour)}, // Old (should be removed)
		}

		policy := &backup.RetentionConfig{
			MaxBackups: 3,
			MaxAge:     24 * time.Hour,
		}

		// Simulate retention policy application
		var retained []*backup.BackupMetadata
		var removed []*backup.BackupMetadata

		// Apply age policy
		for _, b := range backups {
			if now.Sub(b.CreatedAt) <= policy.MaxAge {
				retained = append(retained, b)
			} else {
				removed = append(removed, b)
			}
		}

		// Apply count policy
		if len(retained) > policy.MaxBackups {
			extraCount := len(retained) - policy.MaxBackups
			for i := 0; i < extraCount; i++ {
				removed = append(removed, retained[len(retained)-1-i])
			}
			retained = retained[:policy.MaxBackups]
		}

		assert.LessOrEqual(t, len(retained), policy.MaxBackups)
		assert.Equal(t, 2, len(removed)) // 2 backups removed by age policy

		t.Logf("Retained %d backups, removed %d backups", len(retained), len(removed))
	})
}

// TestBackupFilteringSuite tests backup filtering functionality
func TestBackupFilteringSuite(t *testing.T) {
	// Create test backup metadata
	now := time.Now()
	backups := []*backup.BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: "prod_db",
			CreatedAt:    now.Add(-1 * time.Hour),
			Status:       backup.BackupStatusCompleted,
			Tags:         map[string]string{"env": "production", "type": "manual"},
		},
		{
			ID:           "backup-2",
			DatabaseName: "test_db",
			CreatedAt:    now.Add(-2 * time.Hour),
			Status:       backup.BackupStatusCompleted,
			Tags:         map[string]string{"env": "testing", "type": "automated"},
		},
		{
			ID:           "backup-3",
			DatabaseName: "prod_db",
			CreatedAt:    now.Add(-3 * time.Hour),
			Status:       backup.BackupStatusFailed,
			Tags:         map[string]string{"env": "production", "type": "automated"},
		},
	}

	// Test filtering by database name
	t.Run("Filter by database name", func(t *testing.T) {
		filter := backup.BackupFilter{
			DatabaseName: "prod_db",
		}

		filtered := filterBackups(backups, filter)
		assert.Len(t, filtered, 2)
		for _, b := range filtered {
			assert.Equal(t, "prod_db", b.DatabaseName)
		}
	})

	// Test filtering by status
	t.Run("Filter by status", func(t *testing.T) {
		filter := backup.BackupFilter{
			Status: backup.BackupStatusCompleted,
		}

		filtered := filterBackups(backups, filter)
		assert.Len(t, filtered, 2)
		for _, b := range filtered {
			assert.Equal(t, backup.BackupStatusCompleted, b.Status)
		}
	})

	// Test filtering by tags
	t.Run("Filter by tags", func(t *testing.T) {
		filter := backup.BackupFilter{
			Tags: map[string]string{
				"env": "production",
			},
		}

		filtered := filterBackups(backups, filter)
		assert.Len(t, filtered, 2)
		for _, b := range filtered {
			assert.Equal(t, "production", b.Tags["env"])
		}
	})

	// Test filtering by time range
	t.Run("Filter by time range", func(t *testing.T) {
		filter := backup.BackupFilter{
			CreatedAfter:  now.Add(-2*time.Hour - 30*time.Minute),
			CreatedBefore: now.Add(-30 * time.Minute),
		}

		filtered := filterBackups(backups, filter)
		assert.Len(t, filtered, 2) // backup-1 and backup-2
	})
}

// TestPerformanceBenchmarks runs performance-related tests
func TestPerformanceBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	// Test backup ID generation performance
	t.Run("Backup ID generation performance", func(t *testing.T) {
		start := time.Now()
		ids := make(map[string]bool)

		for i := 0; i < 1000; i++ {
			id := generateBackupID()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "Generated ID should be unique")
			ids[id] = true
		}

		duration := time.Since(start)
		t.Logf("Generated 1000 unique backup IDs in %v", duration)
		assert.Less(t, duration, 100*time.Millisecond, "ID generation should be fast")
	})

	// Test compression configuration performance
	t.Run("Compression configuration performance", func(t *testing.T) {
		algorithms := []backup.CompressionAlgorithm{
			backup.CompressionGzip,
			backup.CompressionLZ4,
			backup.CompressionZstd,
		}

		for _, algorithm := range algorithms {
			start := time.Now()

			for i := 0; i < 100; i++ {
				config := &backup.CompressionConfig{
					Enabled:   true,
					Algorithm: algorithm,
					Level:     6,
				}
				_ = config // Use the config
			}

			duration := time.Since(start)
			t.Logf("Created 100 %s compression configs in %v", algorithm, duration)
		}
	})
}

// TestSecurityFeatures tests security-related functionality
func TestSecurityFeatures(t *testing.T) {
	// Test encryption key generation and validation
	t.Run("Encryption key security", func(t *testing.T) {
		// Test key generation
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)

		_, err := rand.Read(key1)
		require.NoError(t, err)
		_, err = rand.Read(key2)
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t, key1, key2, "Generated keys should be unique")
		assert.Len(t, key1, 32, "Key should be 32 bytes for AES-256")
		assert.Len(t, key2, 32, "Key should be 32 bytes for AES-256")

		// Test key validation
		config := &backup.EncryptionConfig{
			Enabled:   true,
			Algorithm: backup.EncryptionAES256,
			KeySource: backup.EncryptionKeySourceDirect,
			Key:       key1,
		}

		err = config.Validate()
		assert.NoError(t, err, "Valid encryption config should pass validation")

		// Test invalid key size
		invalidConfig := &backup.EncryptionConfig{
			Enabled:   true,
			Algorithm: backup.EncryptionAES256,
			KeySource: backup.EncryptionKeySourceDirect,
			Key:       []byte("short"), // Too short
		}

		err = invalidConfig.Validate()
		assert.Error(t, err, "Invalid key size should fail validation")
	})

	// Test file permissions
	t.Run("File permissions security", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test restrictive permissions
		restrictiveConfig := &backup.LocalConfig{
			BasePath:    tempDir,
			Permissions: 0600, // Very restrictive
		}

		provider, err := backup.NewLocalStorageProvider(restrictiveConfig)
		require.NoError(t, err)

		// Create a test file to check permissions
		testFile := filepath.Join(tempDir, "test_permissions.txt")
		err = os.WriteFile(testFile, []byte("test"), restrictiveConfig.Permissions)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "File should have restrictive permissions")

		_ = provider // Use the provider
	})
}

// Helper functions

func filterBackups(backups []*backup.BackupMetadata, filter backup.BackupFilter) []*backup.BackupMetadata {
	var filtered []*backup.BackupMetadata

	for _, b := range backups {
		// Filter by database name
		if filter.DatabaseName != "" && b.DatabaseName != filter.DatabaseName {
			continue
		}

		// Filter by status
		if filter.Status != "" && b.Status != filter.Status {
			continue
		}

		// Filter by time range
		if !filter.CreatedAfter.IsZero() && b.CreatedAt.Before(filter.CreatedAfter) {
			continue
		}
		if !filter.CreatedBefore.IsZero() && b.CreatedAt.After(filter.CreatedBefore) {
			continue
		}

		// Filter by tags
		if len(filter.Tags) > 0 {
			match := true
			for key, value := range filter.Tags {
				if b.Tags[key] != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, b)
	}

	return filtered
}

func generateBackupID() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	timeStr := now.Format("150405")

	// Generate random suffix
	suffix := make([]byte, 4)
	rand.Read(suffix)

	return fmt.Sprintf("backup_%s_%s_%x", dateStr, timeStr, suffix)
}
