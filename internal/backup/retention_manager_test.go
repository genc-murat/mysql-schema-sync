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

func TestNewRetentionManager(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      10,
		MaxAge:          30 * 24 * time.Hour,
		CleanupInterval: 24 * time.Hour,
		KeepDaily:       7,
		KeepWeekly:      4,
		KeepMonthly:     3,
	}
	logger := logging.NewDefaultLogger()

	rm := NewRetentionManager(mockBackupManager, config, logger)

	assert.NotNil(t, rm)
	assert.IsType(t, &retentionManager{}, rm)
}

func TestRetentionManager_ValidateRetentionPolicy(t *testing.T) {
	tests := []struct {
		name        string
		config      *RetentionConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &RetentionConfig{
				MaxBackups:      10,
				MaxAge:          30 * 24 * time.Hour,
				CleanupInterval: 24 * time.Hour,
				KeepDaily:       7,
				KeepWeekly:      4,
				KeepMonthly:     3,
			},
			expectError: false,
		},
		{
			name: "negative max backups",
			config: &RetentionConfig{
				MaxBackups:      -1,
				MaxAge:          30 * 24 * time.Hour,
				CleanupInterval: 24 * time.Hour,
			},
			expectError: true,
		},
		{
			name: "negative max age",
			config: &RetentionConfig{
				MaxBackups:      10,
				MaxAge:          -1 * time.Hour,
				CleanupInterval: 24 * time.Hour,
			},
			expectError: true,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackupManager := &MockBackupManager{}
			rm := NewRetentionManager(mockBackupManager, tt.config, nil)

			err := rm.ValidateRetentionPolicy()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetentionManager_ApplyRetentionPolicy_MaxBackups(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      3,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups (newest first after sorting)
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-1 * time.Hour), // Newest
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-3",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-3 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-4",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-4 * time.Hour), // Should be deleted
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-5",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-5 * time.Hour), // Should be deleted
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-4").Return(nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-5").Return(nil)

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)

	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalBackupsProcessed)
	assert.Equal(t, 2, result.BackupsDeleted)
	assert.Equal(t, 3, result.BackupsKept)
	assert.False(t, result.DryRun)
	assert.True(t, result.PolicyApplied.MaxBackupsApplied)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_ApplyRetentionPolicy_MaxAge(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxAge:          48 * time.Hour, // Keep backups newer than 48 hours
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-1 * time.Hour), // Keep (within 48h)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-24 * time.Hour), // Keep (within 48h)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-3",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-72 * time.Hour), // Delete (older than 48h)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-4",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-96 * time.Hour), // Delete (older than 48h)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-3").Return(nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-4").Return(nil)

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)

	require.NoError(t, err)
	assert.Equal(t, 4, result.TotalBackupsProcessed)
	assert.Equal(t, 2, result.BackupsDeleted)
	assert.Equal(t, 2, result.BackupsKept)
	assert.True(t, result.PolicyApplied.MaxAgeApplied)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_ApplyRetentionPolicy_DryRun(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      2,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-1 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-3",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-3 * time.Hour), // Should be marked for deletion but not deleted
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)
	// No DeleteBackup calls should be made in dry run

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, true)

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalBackupsProcessed)
	assert.Equal(t, 1, result.BackupsDeleted)
	assert.Equal(t, 2, result.BackupsKept)
	assert.True(t, result.DryRun)

	// Verify that the backup marked for deletion is correct
	assert.Equal(t, "backup-3", result.DeletedBackups[0].ID)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_ApplyRetentionPolicy_PeriodicRetention(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		KeepDaily:       2, // Keep 2 daily backups
		KeepWeekly:      1, // Keep 1 weekly backup
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups spanning multiple days and weeks
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-today-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-2 * time.Hour), // Today
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-today-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-4 * time.Hour), // Today (older)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-yesterday-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-26 * time.Hour), // Yesterday
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-yesterday-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-30 * time.Hour), // Yesterday (older)
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-last-week",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-8 * 24 * time.Hour), // Last week
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-old",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-15 * 24 * time.Hour), // Old backup
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)
	// Mock delete calls for backups that should be removed
	mockBackupManager.On("DeleteBackup", ctx, mock.AnythingOfType("string")).Return(nil).Maybe()

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)

	require.NoError(t, err)
	assert.Equal(t, 6, result.TotalBackupsProcessed)
	assert.True(t, result.PolicyApplied.DailyApplied)
	assert.True(t, result.PolicyApplied.WeeklyApplied)

	// Should keep at least the newest backup from each day/week period
	assert.Greater(t, result.BackupsKept, 0)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_ApplyRetentionPolicy_ProtectedBackups(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      1, // Very aggressive policy
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups with protected backup
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-1 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2-protected",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
			Tags: map[string]string{
				"protected": "true",
			},
		},
		{
			ID:           "backup-3-premigration",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-3 * time.Hour), // Recent pre-migration backup
			Size:         1000,
			Status:       BackupStatusCompleted,
			Tags: map[string]string{
				"type": "pre-migration",
			},
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)
	// No backups should be deleted due to protection

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalBackupsProcessed)
	assert.Equal(t, 0, result.BackupsDeleted) // All backups protected
	assert.Equal(t, 3, result.BackupsKept)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_GetRetentionCandidates(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      2,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "test_db"

	// Create test backups
	now := time.Now()
	backups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-1 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-3",
			DatabaseName: databaseName,
			CreatedAt:    now.Add(-3 * time.Hour), // Should be candidate for deletion
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return(backups, nil)

	candidates, err := rm.GetRetentionCandidates(ctx, databaseName)

	require.NoError(t, err)
	assert.Len(t, candidates, 1)
	assert.Equal(t, "backup-3", candidates[0].ID)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_GetRetentionReport(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      5,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups across multiple databases and ages
	now := time.Now()
	allBackups := []*BackupMetadata{
		{
			ID:           "backup-1",
			DatabaseName: "db1",
			CreatedAt:    now.Add(-1 * time.Hour), // Daily
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-2",
			DatabaseName: "db1",
			CreatedAt:    now.Add(-3 * 24 * time.Hour), // Weekly
			Size:         2000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-3",
			DatabaseName: "db2",
			CreatedAt:    now.Add(-15 * 24 * time.Hour), // Monthly
			Size:         1500,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-4",
			DatabaseName: "db2",
			CreatedAt:    now.Add(-45 * 24 * time.Hour), // Older
			Size:         3000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(allBackups, nil)
	mockBackupManager.On("GetBackupsByDatabase", ctx, "db1").Return([]*BackupMetadata{allBackups[0], allBackups[1]}, nil)
	mockBackupManager.On("GetBackupsByDatabase", ctx, "db2").Return([]*BackupMetadata{allBackups[2], allBackups[3]}, nil)

	report, err := rm.GetRetentionReport(ctx)

	require.NoError(t, err)
	assert.Equal(t, 4, report.TotalBackups)
	assert.Equal(t, 2, report.BackupsByDatabase["db1"])
	assert.Equal(t, 2, report.BackupsByDatabase["db2"])
	assert.Equal(t, int64(7500), report.StorageUsage) // Sum of all backup sizes
	assert.NotNil(t, report.RecommendedPolicies)
	assert.NotEmpty(t, report.RecommendedPolicies.Reasoning)

	// Check age distribution
	assert.Equal(t, 1, report.BackupsByAge["daily"])
	assert.Equal(t, 1, report.BackupsByAge["weekly"])
	assert.Equal(t, 1, report.BackupsByAge["monthly"])
	assert.Equal(t, 1, report.BackupsByAge["older"])

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_ApplyRetentionPolicyToAll(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      2,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()

	// Create test backups for multiple databases
	now := time.Now()
	allBackups := []*BackupMetadata{
		{
			ID:           "backup-db1-1",
			DatabaseName: "db1",
			CreatedAt:    now.Add(-1 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-db1-2",
			DatabaseName: "db1",
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-db1-3",
			DatabaseName: "db1",
			CreatedAt:    now.Add(-3 * time.Hour), // Should be deleted
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-db2-1",
			DatabaseName: "db2",
			CreatedAt:    now.Add(-1 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-db2-2",
			DatabaseName: "db2",
			CreatedAt:    now.Add(-2 * time.Hour),
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
		{
			ID:           "backup-db2-3",
			DatabaseName: "db2",
			CreatedAt:    now.Add(-3 * time.Hour), // Should be deleted
			Size:         1000,
			Status:       BackupStatusCompleted,
		},
	}

	mockBackupManager.On("ListBackups", ctx, BackupFilter{}).Return(allBackups, nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-db1-3").Return(nil)
	mockBackupManager.On("DeleteBackup", ctx, "backup-db2-3").Return(nil)

	result, err := rm.ApplyRetentionPolicyToAll(ctx, false)

	require.NoError(t, err)
	assert.Equal(t, 6, result.TotalBackupsProcessed)
	assert.Equal(t, 2, result.BackupsDeleted)
	assert.Equal(t, 4, result.BackupsKept)
	assert.False(t, result.DryRun)

	mockBackupManager.AssertExpectations(t)
}

func TestRetentionManager_EmptyDatabase(t *testing.T) {
	mockBackupManager := &MockBackupManager{}
	config := &RetentionConfig{
		MaxBackups:      5,
		CleanupInterval: 24 * time.Hour,
	}
	rm := NewRetentionManager(mockBackupManager, config, nil)

	ctx := context.Background()
	databaseName := "empty_db"

	mockBackupManager.On("GetBackupsByDatabase", ctx, databaseName).Return([]*BackupMetadata{}, nil)

	result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalBackupsProcessed)
	assert.Equal(t, 0, result.BackupsDeleted)
	assert.Equal(t, 0, result.BackupsKept)

	mockBackupManager.AssertExpectations(t)
}
