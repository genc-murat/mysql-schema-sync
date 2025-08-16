package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionManager_Integration(t *testing.T) {
	// Create a mock backup manager
	mockBackupManager := &MockBackupManager{}

	// Create retention configuration
	config := &RetentionConfig{
		MaxBackups:      3,
		MaxAge:          7 * 24 * time.Hour, // 7 days
		CleanupInterval: 24 * time.Hour,     // Daily cleanup
		KeepDaily:       7,
		KeepWeekly:      4,
		KeepMonthly:     3,
	}

	// Create retention manager
	logger := logging.NewDefaultLogger()
	rm := NewRetentionManager(mockBackupManager, config, logger)

	// Test validation
	err := rm.ValidateRetentionPolicy()
	require.NoError(t, err)

	// Test getting retention report
	ctx := context.Background()
	report, err := rm.GetRetentionReport(ctx)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, report.TotalBackups, 0)
	assert.NotNil(t, report.RecommendedPolicies)
}

func TestStorageMonitor_Integration(t *testing.T) {
	// Create a mock backup manager
	mockBackupManager := &MockBackupManager{}

	// Create backup system configuration
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "./test-backups",
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
		},
		Validation: ValidationConfig{
			Enabled:           true,
			ChecksumAlgorithm: "sha256",
		},
	}

	// Create storage monitor
	logger := logging.NewDefaultLogger()
	sm := NewStorageMonitor(mockBackupManager, config, logger)

	ctx := context.Background()

	// Test storage usage report
	usage, err := sm.GetStorageUsage(ctx)
	require.NoError(t, err)
	assert.NotNil(t, usage)
	assert.GreaterOrEqual(t, usage.TotalBackups, 0)

	// Test storage health summary
	health, err := sm.GetStorageHealthSummary(ctx)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.NotEmpty(t, health.OverallStatus)

	// Test storage quota check
	quotas, err := sm.CheckStorageQuotas(ctx)
	require.NoError(t, err)
	assert.NotNil(t, quotas)

	// Test storage optimization recommendations
	optimization, err := sm.GetStorageOptimizationRecommendations(ctx)
	require.NoError(t, err)
	assert.NotNil(t, optimization)
	assert.NotNil(t, optimization.CompressionAnalysis)
	assert.NotNil(t, optimization.RetentionAnalysis)
	assert.NotNil(t, optimization.DuplicationAnalysis)

	// Test storage alerts
	alerts, err := sm.GenerateStorageAlerts(ctx)
	require.NoError(t, err)
	assert.NotNil(t, alerts)
}

func TestRetentionManager_CleanupReporting(t *testing.T) {
	// Create a mock backup manager
	mockBackupManager := &MockBackupManager{}

	// Create retention configuration
	config := &RetentionConfig{
		MaxBackups:      2, // Very aggressive for testing
		CleanupInterval: time.Hour,
	}

	// Create retention manager
	logger := logging.NewDefaultLogger()
	rm := NewRetentionManager(mockBackupManager, config, logger)

	ctx := context.Background()

	// Test cleanup report generation
	report, err := rm.GetCleanupReport(ctx, "testdb", true) // Dry run
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "testdb", report.DatabaseName)
	assert.True(t, report.DryRun)
	assert.NotNil(t, report.PolicyApplied)
	assert.NotNil(t, report.CleanupImpact)

	// Test cleanup history (should be empty for now)
	history, err := rm.GetCleanupHistory(ctx, 10)
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Len(t, history, 0) // Empty since we don't have persistent storage yet
}

func TestBackupManager_RetentionAndMonitoringIntegration(t *testing.T) {
	// Create backup system configuration
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath:    "./test-backups",
				Permissions: 0755,
			},
		},
		Retention: RetentionConfig{
			MaxBackups:      5,
			MaxAge:          30 * 24 * time.Hour,
			CleanupInterval: 24 * time.Hour,
			KeepDaily:       7,
			KeepWeekly:      4,
			KeepMonthly:     3,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: CompressionTypeGzip,
			Level:     6,
		},
		Encryption: EncryptionConfig{
			Enabled: false, // Disabled for testing
		},
		Validation: ValidationConfig{
			Enabled:           true,
			ChecksumAlgorithm: "sha256",
		},
	}

	// Test that we can create a backup manager with retention and monitoring
	// Note: This would normally require actual storage providers and database connections
	// For this integration test, we're just testing that the components can be created
	// and basic operations work

	// Validate the configuration
	err := config.Validate()
	require.NoError(t, err)

	// Test that retention config is valid
	err = config.Retention.Validate()
	require.NoError(t, err)

	// Test that storage config is valid
	err = config.Storage.Validate()
	require.NoError(t, err)

	// Test that compression config is valid
	err = config.Compression.Validate()
	require.NoError(t, err)

	// Test that validation config is valid
	err = config.Validation.Validate()
	require.NoError(t, err)
}
