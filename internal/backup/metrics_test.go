package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mysql-schema-sync/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	config := MetricsConfig{
		Logger:             logger,
		ReportPath:         filepath.Join(tempDir, "reports"),
		CollectionInterval: 1 * time.Minute,
	}

	mc := NewMetricsCollector(config)

	assert.NotNil(t, mc)
	assert.NotNil(t, mc.metrics)
	assert.NotNil(t, mc.alerts)
	assert.Equal(t, logger, mc.logger)
	assert.NotZero(t, mc.startTime)
}

func TestMetricsCollector_RecordBackupOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	// Test successful backup
	t.Run("successful backup", func(t *testing.T) {
		duration := 30 * time.Second
		size := int64(1024 * 1024) // 1MB
		compressionRatio := 0.7

		mc.RecordBackupOperation(true, duration, size, compressionRatio)

		metrics := mc.GetMetrics()
		assert.Equal(t, int64(1), metrics.BackupOperations.Total)
		assert.Equal(t, int64(1), metrics.BackupOperations.Success)
		assert.Equal(t, int64(0), metrics.BackupOperations.Failed)
		assert.Equal(t, 1.0, metrics.BackupOperations.SuccessRate)
		assert.Equal(t, duration, metrics.BackupOperations.MinDuration)
		assert.Equal(t, duration, metrics.BackupOperations.MaxDuration)
		assert.Equal(t, duration, metrics.BackupOperations.AverageDuration)
		assert.Equal(t, size, metrics.Performance.AverageBackupSize)
		assert.Equal(t, compressionRatio, metrics.Performance.AverageCompressionRatio)
		assert.Greater(t, metrics.Performance.BackupThroughput, 0.0)
	})

	// Test failed backup
	t.Run("failed backup", func(t *testing.T) {
		duration := 10 * time.Second

		mc.RecordBackupOperation(false, duration, 0, 0)

		metrics := mc.GetMetrics()
		assert.Equal(t, int64(2), metrics.BackupOperations.Total)
		assert.Equal(t, int64(1), metrics.BackupOperations.Success)
		assert.Equal(t, int64(1), metrics.BackupOperations.Failed)
		assert.Equal(t, 0.5, metrics.BackupOperations.SuccessRate)
		assert.Equal(t, duration, metrics.BackupOperations.MinDuration)
	})

	// Test multiple operations for average calculation
	t.Run("multiple operations", func(t *testing.T) {
		// Reset metrics
		mc = NewMetricsCollector(MetricsConfig{
			Logger: logger,
		})

		durations := []time.Duration{
			10 * time.Second,
			20 * time.Second,
			30 * time.Second,
		}

		for _, duration := range durations {
			mc.RecordBackupOperation(true, duration, 1024, 0.8)
		}

		metrics := mc.GetMetrics()
		assert.Equal(t, int64(3), metrics.BackupOperations.Total)
		assert.Equal(t, int64(3), metrics.BackupOperations.Success)
		assert.Equal(t, 1.0, metrics.BackupOperations.SuccessRate)
		assert.Equal(t, 10*time.Second, metrics.BackupOperations.MinDuration)
		assert.Equal(t, 30*time.Second, metrics.BackupOperations.MaxDuration)
		assert.Equal(t, 20*time.Second, metrics.BackupOperations.AverageDuration)
	})
}

func TestMetricsCollector_RecordRollbackOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	duration := 45 * time.Second

	// Test successful rollback
	mc.RecordRollbackOperation(true, duration)

	metrics := mc.GetMetrics()
	assert.Equal(t, int64(1), metrics.RollbackOperations.Total)
	assert.Equal(t, int64(1), metrics.RollbackOperations.Success)
	assert.Equal(t, int64(0), metrics.RollbackOperations.Failed)
	assert.Equal(t, 1.0, metrics.RollbackOperations.SuccessRate)
	assert.Equal(t, duration, metrics.RollbackOperations.AverageDuration)

	// Test failed rollback
	mc.RecordRollbackOperation(false, 20*time.Second)

	metrics = mc.GetMetrics()
	assert.Equal(t, int64(2), metrics.RollbackOperations.Total)
	assert.Equal(t, int64(1), metrics.RollbackOperations.Success)
	assert.Equal(t, int64(1), metrics.RollbackOperations.Failed)
	assert.Equal(t, 0.5, metrics.RollbackOperations.SuccessRate)
}

func TestMetricsCollector_RecordValidationOperation(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	duration := 5 * time.Second

	// Test successful validation
	mc.RecordValidationOperation(true, duration)

	metrics := mc.GetMetrics()
	assert.Equal(t, int64(1), metrics.ValidationOperations.Total)
	assert.Equal(t, int64(1), metrics.ValidationOperations.Success)
	assert.Equal(t, int64(0), metrics.ValidationOperations.Failed)
	assert.Equal(t, 1.0, metrics.ValidationOperations.SuccessRate)
	assert.Equal(t, duration, metrics.Performance.AverageValidationTime)

	// Test failed validation
	mc.RecordValidationOperation(false, 3*time.Second)

	metrics = mc.GetMetrics()
	assert.Equal(t, int64(2), metrics.ValidationOperations.Total)
	assert.Equal(t, int64(1), metrics.ValidationOperations.Success)
	assert.Equal(t, int64(1), metrics.ValidationOperations.Failed)
	assert.Equal(t, 0.5, metrics.ValidationOperations.SuccessRate)
}

func TestMetricsCollector_RecordStorageUsage(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	provider := "s3"
	totalBackups := int64(10)
	totalSize := int64(1024 * 1024 * 100) // 100MB

	mc.RecordStorageUsage(provider, totalBackups, totalSize)

	metrics := mc.GetMetrics()
	assert.Equal(t, totalBackups, metrics.Storage.TotalBackups)
	assert.Equal(t, totalSize, metrics.Storage.TotalStorageUsed)
	assert.Equal(t, totalSize, metrics.Storage.StorageByProvider[provider])
}

func TestMetricsCollector_UpdateHealthStatus(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	// Record some operations to have data for health calculation
	mc.RecordBackupOperation(true, 10*time.Second, 1024, 0.8)
	mc.RecordValidationOperation(true, 2*time.Second)

	mc.UpdateHealthStatus()

	metrics := mc.GetMetrics()
	assert.NotZero(t, metrics.Health.LastHealthCheck)
	assert.NotEqual(t, HealthStatusUnknown, metrics.Health.OverallHealth)
}

func TestMetricsCollector_AlertGeneration(t *testing.T) {
	logger := logging.NewDefaultLogger()

	// Set low thresholds to trigger alerts
	config := MetricsConfig{
		Logger: logger,
		AlertThresholds: AlertThresholds{
			FailureRateWarning:         0.01, // 1%
			FailureRateCritical:        0.05, // 5%
			ValidationFailureThreshold: 1,
		},
	}

	mc := NewMetricsCollector(config)

	// Record operations that should trigger alerts
	mc.RecordBackupOperation(false, 10*time.Second, 0, 0) // Failed backup
	mc.RecordValidationOperation(false, 5*time.Second)    // Failed validation

	alerts := mc.GetActiveAlerts()
	assert.Greater(t, len(alerts), 0, "Should have generated alerts")

	// Check alert types
	alertTypes := make(map[AlertType]bool)
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
		assert.False(t, alert.Resolved)
		assert.NotEmpty(t, alert.ID)
		assert.NotEmpty(t, alert.Title)
		assert.NotEmpty(t, alert.Message)
		assert.NotZero(t, alert.Timestamp)
	}

	// Should have backup failure alert
	assert.True(t, alertTypes[AlertTypeBackupFailure], "Should have backup failure alert")
}

func TestMetricsCollector_ResolveAlert(t *testing.T) {
	logger := logging.NewDefaultLogger()

	config := MetricsConfig{
		Logger: logger,
		AlertThresholds: AlertThresholds{
			FailureRateWarning: 0.01,
		},
	}

	mc := NewMetricsCollector(config)

	// Generate an alert
	mc.RecordBackupOperation(false, 10*time.Second, 0, 0)

	alerts := mc.GetActiveAlerts()
	require.Greater(t, len(alerts), 0, "Should have at least one alert")

	alertID := alerts[0].ID

	// Resolve the alert
	err := mc.ResolveAlert(alertID)
	assert.NoError(t, err)

	// Check that alert is resolved
	activeAlerts := mc.GetActiveAlerts()
	for _, alert := range activeAlerts {
		assert.NotEqual(t, alertID, alert.ID, "Alert should be resolved")
	}

	// Test resolving non-existent alert
	err = mc.ResolveAlert("non-existent-alert")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alert not found")
}

func TestMetricsCollector_GenerateReport(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	config := MetricsConfig{
		Logger:     logger,
		ReportPath: filepath.Join(tempDir, "reports", "backup-report.json"),
	}

	mc := NewMetricsCollector(config)

	// Record some operations
	mc.RecordBackupOperation(true, 30*time.Second, 1024*1024, 0.7)
	mc.RecordBackupOperation(true, 25*time.Second, 2048*1024, 0.8)
	mc.RecordValidationOperation(true, 5*time.Second)
	mc.RecordStorageUsage("local", 2, 3072*1024)
	mc.UpdateHealthStatus()

	ctx := context.Background()
	report, err := mc.GenerateReport(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotZero(t, report.GeneratedAt)
	assert.Equal(t, int64(2), report.Metrics.BackupOperations.Total)
	assert.Equal(t, int64(2), report.Metrics.Storage.TotalBackups)
	assert.NotEmpty(t, report.Summary.RecentPerformance)
	assert.NotEmpty(t, report.Summary.Recommendations)

	// Check that report file was created
	reportFiles, err := filepath.Glob(filepath.Join(tempDir, "reports", "backup-report_*.json"))
	assert.NoError(t, err)
	assert.Len(t, reportFiles, 1, "Should have created one report file")

	// Verify report file content
	reportData, err := os.ReadFile(reportFiles[0])
	assert.NoError(t, err)

	var savedReport BackupSystemReport
	err = json.Unmarshal(reportData, &savedReport)
	assert.NoError(t, err)
	assert.Equal(t, report.GeneratedAt.Unix(), savedReport.GeneratedAt.Unix())
}

func TestHealthStatusDetermination(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	tests := []struct {
		name                  string
		setupOperations       func()
		expectedOverallHealth HealthStatus
	}{
		{
			name: "healthy system",
			setupOperations: func() {
				// All operations successful
				for i := 0; i < 10; i++ {
					mc.RecordBackupOperation(true, 10*time.Second, 1024, 0.8)
					mc.RecordValidationOperation(true, 2*time.Second)
				}
			},
			expectedOverallHealth: HealthStatusHealthy,
		},
		{
			name: "system with warnings",
			setupOperations: func() {
				// Some failed operations
				for i := 0; i < 8; i++ {
					mc.RecordBackupOperation(true, 10*time.Second, 1024, 0.8)
				}
				for i := 0; i < 2; i++ {
					mc.RecordBackupOperation(false, 10*time.Second, 0, 0)
				}
				for i := 0; i < 10; i++ {
					mc.RecordValidationOperation(true, 2*time.Second)
				}
			},
			expectedOverallHealth: HealthStatusHealthy, // Still healthy with 80% success rate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics
			mc = NewMetricsCollector(MetricsConfig{
				Logger: logger,
			})

			tt.setupOperations()
			mc.UpdateHealthStatus()

			metrics := mc.GetMetrics()
			assert.Equal(t, tt.expectedOverallHealth, metrics.Health.OverallHealth)
		})
	}
}

func TestReportSummaryGeneration(t *testing.T) {
	logger := logging.NewDefaultLogger()
	mc := NewMetricsCollector(MetricsConfig{
		Logger: logger,
	})

	// Record operations with good performance
	for i := 0; i < 10; i++ {
		mc.RecordBackupOperation(true, 5*time.Second, 10*1024*1024, 0.8) // 10MB backups, fast
	}
	mc.RecordStorageUsage("local", 10, 100*1024*1024) // 100MB total
	mc.UpdateHealthStatus()

	ctx := context.Background()
	report, err := mc.GenerateReport(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), report.Summary.TotalBackups)
	assert.Equal(t, 1.0, report.Summary.BackupSuccessRate)
	assert.Equal(t, "Good", report.Summary.RecentPerformance) // High throughput
	assert.Contains(t, report.Summary.Recommendations, "normally")
}

func TestAlertThresholds(t *testing.T) {
	logger := logging.NewDefaultLogger()

	tests := []struct {
		name              string
		thresholds        AlertThresholds
		operations        func(*MetricsCollector)
		expectedAlertType AlertType
		expectedSeverity  AlertSeverity
	}{
		{
			name: "backup failure warning",
			thresholds: AlertThresholds{
				FailureRateWarning:  0.10, // 10%
				FailureRateCritical: 0.20, // 20%
			},
			operations: func(mc *MetricsCollector) {
				// 15% failure rate (between warning and critical)
				for i := 0; i < 85; i++ {
					mc.RecordBackupOperation(true, 10*time.Second, 1024, 0.8)
				}
				for i := 0; i < 15; i++ {
					mc.RecordBackupOperation(false, 10*time.Second, 0, 0)
				}
			},
			expectedAlertType: AlertTypeBackupFailure,
			expectedSeverity:  AlertSeverityWarning,
		},
		{
			name: "backup failure critical",
			thresholds: AlertThresholds{
				FailureRateWarning:  0.10, // 10%
				FailureRateCritical: 0.20, // 20%
			},
			operations: func(mc *MetricsCollector) {
				// 25% failure rate (above critical)
				for i := 0; i < 75; i++ {
					mc.RecordBackupOperation(true, 10*time.Second, 1024, 0.8)
				}
				for i := 0; i < 25; i++ {
					mc.RecordBackupOperation(false, 10*time.Second, 0, 0)
				}
			},
			expectedAlertType: AlertTypeBackupFailure,
			expectedSeverity:  AlertSeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := MetricsConfig{
				Logger:          logger,
				AlertThresholds: tt.thresholds,
			}

			mc := NewMetricsCollector(config)
			tt.operations(mc)

			alerts := mc.GetActiveAlerts()

			// Find the expected alert
			var foundAlert *Alert
			for _, alert := range alerts {
				if alert.Type == tt.expectedAlertType {
					foundAlert = &alert
					break
				}
			}

			require.NotNil(t, foundAlert, "Expected alert not found")
			assert.Equal(t, tt.expectedSeverity, foundAlert.Severity)
		})
	}
}
