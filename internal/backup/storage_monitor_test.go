package backup

import (
	"context"
	"testing"
	"time"

	"mysql-schema-sync/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewStorageMonitor(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	logger := logging.NewDefaultLogger()

	sm := NewStorageMonitor(mockBackupManager, config, logger)

	assert.NotNil(t, sm)
	assert.IsType(t, &storageMonitor{}, sm)
}

func TestStorageMonitor_GetStorageUsage(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups with different characteristics
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:              "backup-1",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-1 * time.Hour), // Daily
			Size:            1000,
			CompressedSize:  600,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
		},
		{
			ID:              "backup-2",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-3 * 24 * time.Hour), // Weekly
			Size:            2000,
			CompressedSize:  1200,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
		},
		{
			ID:              "backup-3",
			DatabaseName:    "db2",
			CreatedAt:       now.Add(-15 * 24 * time.Hour), // Monthly
			Size:            1500,
			CompressedSize:  900,
			CompressionType: CompressionTypeLZ4,
			Status:          BackupStatusCompleted,
		},
		{
			ID:              "backup-4",
			DatabaseName:    "db2",
			CreatedAt:       now.Add(-45 * 24 * time.Hour), // Older
			Size:            3000,
			CompressedSize:  1800,
			CompressionType: CompressionTypeNone,
			Status:          BackupStatusCompleted,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(backups, nil)

	report, err := sm.GetStorageUsage(ctx)

	require.NoError(t, err)
	assert.Equal(t, 4, report.TotalBackups)
	assert.Equal(t, int64(7500), report.TotalSize)           // 1000 + 2000 + 1500 + 3000
	assert.Equal(t, int64(4500), report.TotalCompressedSize) // 600 + 1200 + 900 + 1800
	assert.Equal(t, int64(1875), report.AverageBackupSize)   // 7500 / 4
	assert.Equal(t, float64(4500)/float64(7500), report.CompressionRatio)

	// Check largest and smallest backups
	assert.Equal(t, "backup-4", report.LargestBackup.ID)
	assert.Equal(t, "backup-1", report.SmallestBackup.ID)

	// Check storage by database
	assert.Len(t, report.StorageByDatabase, 2)
	assert.Equal(t, 2, report.StorageByDatabase["db1"].BackupCount)
	assert.Equal(t, int64(3000), report.StorageByDatabase["db1"].TotalSize)
	assert.Equal(t, 2, report.StorageByDatabase["db2"].BackupCount)
	assert.Equal(t, int64(4500), report.StorageByDatabase["db2"].TotalSize)

	// Check storage by age
	assert.Equal(t, 1, report.StorageByAge["daily"].BackupCount)
	assert.Equal(t, 1, report.StorageByAge["weekly"].BackupCount)
	assert.Equal(t, 1, report.StorageByAge["monthly"].BackupCount)
	assert.Equal(t, 1, report.StorageByAge["older"].BackupCount)

	// Check storage by provider
	providerKey := string(StorageProviderLocal)
	assert.Equal(t, 4, report.StorageByProvider[providerKey].BackupCount)
	assert.Equal(t, int64(7500), report.StorageByProvider[providerKey].TotalSize)

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_GetStorageUsage_EmptyBackups(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return([]*BackupMetadata{}, nil)

	report, err := sm.GetStorageUsage(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, report.TotalBackups)
	assert.Equal(t, int64(0), report.TotalSize)
	assert.Equal(t, int64(0), report.TotalCompressedSize)
	assert.Equal(t, int64(0), report.AverageBackupSize)
	assert.Equal(t, float64(0), report.CompressionRatio)
	assert.Nil(t, report.LargestBackup)
	assert.Nil(t, report.SmallestBackup)

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_GetStorageUsageByDatabase(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups for multiple databases
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:              "backup-db1-1",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-1 * time.Hour),
			Size:            1000,
			CompressedSize:  600,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
		},
		{
			ID:              "backup-db1-2",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-2 * time.Hour),
			Size:            2000,
			CompressedSize:  1200,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
		},
		{
			ID:              "backup-db2-1",
			DatabaseName:    "db2",
			CreatedAt:       now.Add(-1 * time.Hour),
			Size:            1500,
			CompressedSize:  900,
			CompressionType: CompressionTypeLZ4,
			Status:          BackupStatusFailed,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(backups, nil)

	usage, err := sm.GetStorageUsageByDatabase(ctx)

	require.NoError(t, err)
	assert.Len(t, usage, 2)

	// Check db1 usage
	db1Usage := usage["db1"]
	assert.NotNil(t, db1Usage)
	assert.Equal(t, "db1", db1Usage.DatabaseName)
	assert.Equal(t, 2, db1Usage.BackupCount)
	assert.Equal(t, int64(3000), db1Usage.TotalSize)
	assert.Equal(t, int64(1800), db1Usage.CompressedSize)
	assert.Equal(t, int64(1500), db1Usage.AverageBackupSize)
	assert.Equal(t, float64(1800)/float64(3000), db1Usage.CompressionRatio)
	assert.Equal(t, "backup-db1-2", db1Usage.LargestBackup.ID)
	assert.Equal(t, "backup-db1-1", db1Usage.SmallestBackup.ID)
	assert.Equal(t, 2, db1Usage.BackupsByStatus[string(BackupStatusCompleted)])

	// Check db2 usage
	db2Usage := usage["db2"]
	assert.NotNil(t, db2Usage)
	assert.Equal(t, "db2", db2Usage.DatabaseName)
	assert.Equal(t, 1, db2Usage.BackupCount)
	assert.Equal(t, int64(1500), db2Usage.TotalSize)
	assert.Equal(t, int64(900), db2Usage.CompressedSize)
	assert.Equal(t, int64(1500), db2Usage.AverageBackupSize)
	assert.Equal(t, 1, db2Usage.BackupsByStatus[string(BackupStatusFailed)])

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_CheckStorageQuotas(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: "db1",
			CreatedAt:    time.Now(),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(backups, nil)

	status, err := sm.CheckStorageQuotas(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, int64(1000), status.UsedStorage)
	assert.False(t, status.QuotaEnabled) // Default implementation doesn't enable quotas

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_GetStorageOptimizationRecommendations(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups with optimization opportunities
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:              "backup-1",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-1 * time.Hour),
			Size:            1000,
			CompressedSize:  1000, // No compression
			CompressionType: CompressionTypeNone,
			Status:          BackupStatusCompleted,
			Checksum:        "checksum1",
		},
		{
			ID:              "backup-2",
			DatabaseName:    "db1",
			CreatedAt:       now.Add(-100 * 24 * time.Hour), // Very old
			Size:            2000,
			CompressedSize:  1200,
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
			Checksum:        "checksum2",
		},
		{
			ID:              "backup-3",
			DatabaseName:    "db2",
			CreatedAt:       now.Add(-1 * time.Hour),
			Size:            1500,
			CompressedSize:  1500, // No compression
			CompressionType: CompressionTypeNone,
			Status:          BackupStatusCompleted,
			Checksum:        "checksum1", // Duplicate checksum
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(backups, nil)

	report, err := sm.GetStorageOptimizationRecommendations(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.CompressionAnalysis)
	assert.NotNil(t, report.RetentionAnalysis)
	assert.NotNil(t, report.DuplicationAnalysis)

	// Check compression analysis
	assert.Len(t, report.CompressionAnalysis.UncompressedBackups, 2) // backup-1 and backup-3
	assert.Greater(t, report.CompressionAnalysis.PotentialCompressionSavings, int64(0))

	// Check retention analysis
	assert.Len(t, report.RetentionAnalysis.BackupsEligibleForCleanup, 1) // backup-2 (old)
	assert.Greater(t, report.RetentionAnalysis.PotentialRetentionSavings, int64(0))

	// Check duplication analysis
	assert.Len(t, report.DuplicationAnalysis.DuplicateBackups, 1) // backup-1 and backup-3 have same checksum
	assert.Greater(t, report.DuplicationAnalysis.PotentialDeduplicationSavings, int64(0))

	// Check recommendations
	assert.Greater(t, len(report.Recommendations), 0)
	assert.Greater(t, report.TotalPotentialSavings, int64(0))

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_MonitorStorageHealth(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	report, err := sm.MonitorStorageHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "healthy", report.OverallHealth) // Default implementation returns healthy
	assert.NotNil(t, report.ProviderHealth)
	assert.NotNil(t, report.ConnectivityTests)
	assert.NotNil(t, report.PerformanceMetrics)
	assert.NotNil(t, report.HealthIssues)
}

func TestStorageMonitor_GetStorageTrends(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()
	period := 30 * 24 * time.Hour // 30 days

	mockBackupManager.On("ListBackups", ctx, BackupFilter{
		CreatedAfter: &time.Time{}, // Will be set by the method
	}).Return([]*BackupMetadata{}, nil).Run(func(args mock.Arguments) {
		// Verify that CreatedAfter is set to approximately 30 days ago
		filter := args.Get(1).(BackupFilter)
		assert.NotNil(t, filter.CreatedAfter)
		assert.True(t, time.Since(*filter.CreatedAfter) >= period-time.Minute) // Allow 1 minute tolerance
	})

	report, err := sm.GetStorageTrends(ctx, period)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, period, report.Period)
	assert.NotNil(t, report.DatabaseTrends)

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_GenerateStorageAlerts(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups that will trigger alerts
	backups := []*BackupMetadata{
		{
			ID:              "backup-1",
			DatabaseName:    "db1",
			CreatedAt:       time.Now(),
			Size:            15 * 1024 * 1024 * 1024, // 15GB - should trigger high usage alert
			CompressedSize:  14 * 1024 * 1024 * 1024, // Poor compression ratio
			CompressionType: CompressionTypeGzip,
			Status:          BackupStatusCompleted,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(backups, nil)

	alerts, err := sm.GenerateStorageAlerts(ctx)

	require.NoError(t, err)
	assert.Greater(t, len(alerts), 0)

	// Should have high storage usage alert
	var hasStorageAlert, hasCompressionAlert bool
	for _, alert := range alerts {
		if alert.Type == "quota" && alert.Title == "High Storage Usage" {
			hasStorageAlert = true
		}
		if alert.Type == "performance" && alert.Title == "Poor Compression Efficiency" {
			hasCompressionAlert = true
		}
	}

	assert.True(t, hasStorageAlert, "Should have storage usage alert")
	assert.True(t, hasCompressionAlert, "Should have compression efficiency alert")

	mockBackupManager.AssertExpectations(t)
}

func TestStorageMonitor_AnalyzeCompression(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil).(*storageMonitor)

	// Create test backups with different compression characteristics
	backups := []*BackupMetadata{
		{
			ID:              "backup-1",
			Size:            1000,
			CompressedSize:  1000, // No compression
			CompressionType: CompressionTypeNone,
		},
		{
			ID:              "backup-2",
			Size:            2000,
			CompressedSize:  1000, // Good compression (50%)
			CompressionType: CompressionTypeGzip,
		},
		{
			ID:              "backup-3",
			Size:            1500,
			CompressedSize:  1400, // Poor compression (93%)
			CompressionType: CompressionTypeGzip,
		},
	}

	analysis := sm.analyzeCompression(backups)

	assert.NotNil(t, analysis)
	assert.Equal(t, float64(3400)/float64(4500), analysis.OverallCompressionRatio)
	assert.Len(t, analysis.UncompressedBackups, 1)
	assert.Equal(t, "backup-1", analysis.UncompressedBackups[0].ID)
	assert.Len(t, analysis.PoorlyCompressedBackups, 1)
	assert.Equal(t, "backup-3", analysis.PoorlyCompressedBackups[0].ID)
	assert.Greater(t, analysis.PotentialCompressionSavings, int64(0))

	// Check algorithm stats
	assert.Len(t, analysis.CompressionByAlgorithm, 2) // NONE and GZIP
	gzipStats := analysis.CompressionByAlgorithm[string(CompressionTypeGzip)]
	assert.NotNil(t, gzipStats)
	assert.Equal(t, 2, gzipStats.BackupCount)
	assert.Equal(t, int64(3500), gzipStats.OriginalSize)
	assert.Equal(t, int64(2400), gzipStats.CompressedSize)
}

func TestStorageMonitor_AnalyzeRetention(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil).(*storageMonitor)

	ctx := context.Background()

	// Create test backups with different ages
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:        "backup-recent",
			CreatedAt: now.Add(-1 * time.Hour),
			Size:      1000,
		},
		{
			ID:        "backup-old",
			CreatedAt: now.Add(-100 * 24 * time.Hour), // 100 days old
			Size:      2000,
		},
		{
			ID:        "backup-very-old",
			CreatedAt: now.Add(-200 * 24 * time.Hour), // 200 days old
			Size:      1500,
		},
	}

	analysis := sm.analyzeRetention(ctx, backups)

	assert.NotNil(t, analysis)
	assert.Len(t, analysis.BackupsEligibleForCleanup, 2)             // Both old backups
	assert.Equal(t, int64(3500), analysis.PotentialRetentionSavings) // 2000 + 1500
	assert.Equal(t, 1.0/3.0, analysis.RetentionPolicyEffectiveness)  // 1 kept out of 3
}

func TestStorageMonitor_AnalyzeDuplication(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
		},
	}
	sm := NewStorageMonitor(mockBackupManager, config, nil).(*storageMonitor)

	// Create test backups with duplicate checksums
	backups := []*BackupMetadata{
		{
			ID:       "backup-1",
			Size:     1000,
			Checksum: "checksum-a",
		},
		{
			ID:       "backup-2",
			Size:     1000,
			Checksum: "checksum-a", // Duplicate
		},
		{
			ID:       "backup-3",
			Size:     2000,
			Checksum: "checksum-b",
		},
		{
			ID:       "backup-4",
			Size:     2000,
			Checksum: "checksum-b", // Duplicate
		},
		{
			ID:       "backup-5",
			Size:     1500,
			Checksum: "checksum-c", // Unique
		},
	}

	analysis := sm.analyzeDuplication(backups)

	assert.NotNil(t, analysis)
	assert.Len(t, analysis.DuplicateBackups, 2) // Two groups of duplicates

	// Check duplicate groups
	var group1, group2 *BackupDuplication
	for _, dup := range analysis.DuplicateBackups {
		if dup.Checksum == "checksum-a" {
			group1 = dup
		} else if dup.Checksum == "checksum-b" {
			group2 = dup
		}
	}

	assert.NotNil(t, group1)
	assert.Equal(t, 2, group1.BackupCount)
	assert.Equal(t, int64(2000), group1.TotalSize)
	assert.Equal(t, int64(1000), group1.PotentialSavings) // Save one backup worth

	assert.NotNil(t, group2)
	assert.Equal(t, 2, group2.BackupCount)
	assert.Equal(t, int64(4000), group2.TotalSize)
	assert.Equal(t, int64(2000), group2.PotentialSavings) // Save one backup worth

	assert.Equal(t, int64(3000), analysis.PotentialDeduplicationSavings) // 1000 + 2000
	assert.Greater(t, len(analysis.DeduplicationRecommendations), 0)
}
