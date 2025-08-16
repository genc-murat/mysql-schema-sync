package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mysql-schema-sync/internal/logging"
)

// MetricsCollector collects and reports backup system metrics
type MetricsCollector struct {
	logger     *logging.Logger
	metrics    *BackupMetrics
	alerts     *AlertManager
	mu         sync.RWMutex
	startTime  time.Time
	reportPath string
}

// BackupMetrics holds various backup system metrics
type BackupMetrics struct {
	// Operation counters
	BackupOperations     *OperationMetrics `json:"backup_operations"`
	RollbackOperations   *OperationMetrics `json:"rollback_operations"`
	ValidationOperations *OperationMetrics `json:"validation_operations"`

	// Performance metrics
	Performance *PerformanceMetrics `json:"performance"`

	// Storage metrics
	Storage *StorageMetrics `json:"storage"`

	// System health
	Health *HealthMetrics `json:"health"`

	// Time range
	StartTime  time.Time `json:"start_time"`
	LastUpdate time.Time `json:"last_update"`
}

// OperationMetrics tracks success/failure rates for operations
type OperationMetrics struct {
	Total       int64   `json:"total"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`

	// Timing metrics
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`

	// Recent operations (last 24 hours)
	RecentTotal       int64   `json:"recent_total"`
	RecentSuccess     int64   `json:"recent_success"`
	RecentFailed      int64   `json:"recent_failed"`
	RecentSuccessRate float64 `json:"recent_success_rate"`
}

// PerformanceMetrics tracks backup system performance
type PerformanceMetrics struct {
	// Backup performance
	AverageBackupSize       int64   `json:"average_backup_size"`
	AverageCompressionRatio float64 `json:"average_compression_ratio"`
	BackupThroughput        float64 `json:"backup_throughput_mb_per_sec"`

	// Storage performance
	AverageUploadSpeed   float64 `json:"average_upload_speed_mb_per_sec"`
	AverageDownloadSpeed float64 `json:"average_download_speed_mb_per_sec"`

	// Validation performance
	AverageValidationTime time.Duration `json:"average_validation_time"`

	// Resource usage
	PeakMemoryUsage    int64 `json:"peak_memory_usage_bytes"`
	AverageMemoryUsage int64 `json:"average_memory_usage_bytes"`
}

// StorageMetrics tracks storage usage and health
type StorageMetrics struct {
	TotalBackups      int64            `json:"total_backups"`
	TotalStorageUsed  int64            `json:"total_storage_used_bytes"`
	StorageByProvider map[string]int64 `json:"storage_by_provider"`

	// Growth metrics
	DailyGrowthRate   float64 `json:"daily_growth_rate_bytes"`
	WeeklyGrowthRate  float64 `json:"weekly_growth_rate_bytes"`
	MonthlyGrowthRate float64 `json:"monthly_growth_rate_bytes"`

	// Retention metrics
	BackupsDeletedToday int64 `json:"backups_deleted_today"`
	SpaceFreedToday     int64 `json:"space_freed_today_bytes"`
}

// HealthMetrics tracks overall system health
type HealthMetrics struct {
	OverallHealth     HealthStatus `json:"overall_health"`
	StorageHealth     HealthStatus `json:"storage_health"`
	ValidationHealth  HealthStatus `json:"validation_health"`
	PerformanceHealth HealthStatus `json:"performance_health"`

	// Health indicators
	RecentFailureRate      float64 `json:"recent_failure_rate"`
	StorageUtilization     float64 `json:"storage_utilization"`
	PerformanceDegradation float64 `json:"performance_degradation"`

	// Last health check
	LastHealthCheck     time.Time     `json:"last_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "HEALTHY"
	HealthStatusWarning  HealthStatus = "WARNING"
	HealthStatusCritical HealthStatus = "CRITICAL"
	HealthStatusUnknown  HealthStatus = "UNKNOWN"
)

// AlertManager manages backup system alerts and notifications
type AlertManager struct {
	logger     *logging.Logger
	alerts     []Alert
	thresholds AlertThresholds
	mu         sync.RWMutex
}

// Alert represents a system alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       AlertType              `json:"type"`
	Severity   AlertSeverity          `json:"severity"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeBackupFailure     AlertType = "BACKUP_FAILURE"
	AlertTypeValidationFailure AlertType = "VALIDATION_FAILURE"
	AlertTypeStorageQuota      AlertType = "STORAGE_QUOTA"
	AlertTypePerformance       AlertType = "PERFORMANCE"
	AlertTypeSystemHealth      AlertType = "SYSTEM_HEALTH"
	AlertTypeRetentionPolicy   AlertType = "RETENTION_POLICY"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "INFO"
	AlertSeverityWarning  AlertSeverity = "WARNING"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

// AlertThresholds defines thresholds for triggering alerts
type AlertThresholds struct {
	FailureRateWarning             float64 `json:"failure_rate_warning"`
	FailureRateCritical            float64 `json:"failure_rate_critical"`
	StorageWarningPercent          float64 `json:"storage_warning_percent"`
	StorageCriticalPercent         float64 `json:"storage_critical_percent"`
	PerformanceDegradationWarning  float64 `json:"performance_degradation_warning"`
	PerformanceDegradationCritical float64 `json:"performance_degradation_critical"`
	ValidationFailureThreshold     int64   `json:"validation_failure_threshold"`
}

// MetricsConfig holds configuration for metrics collection
type MetricsConfig struct {
	Logger             *logging.Logger
	ReportPath         string
	CollectionInterval time.Duration
	AlertThresholds    AlertThresholds
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	defaultThresholds := AlertThresholds{
		FailureRateWarning:             0.05, // 5%
		FailureRateCritical:            0.10, // 10%
		StorageWarningPercent:          80.0,
		StorageCriticalPercent:         95.0,
		PerformanceDegradationWarning:  0.20, // 20%
		PerformanceDegradationCritical: 0.50, // 50%
		ValidationFailureThreshold:     3,
	}

	if config.AlertThresholds.FailureRateWarning == 0 {
		config.AlertThresholds = defaultThresholds
	}

	mc := &MetricsCollector{
		logger:     config.Logger,
		startTime:  time.Now(),
		reportPath: config.ReportPath,
		metrics: &BackupMetrics{
			BackupOperations:     &OperationMetrics{},
			RollbackOperations:   &OperationMetrics{},
			ValidationOperations: &OperationMetrics{},
			Performance:          &PerformanceMetrics{},
			Storage:              &StorageMetrics{StorageByProvider: make(map[string]int64)},
			Health:               &HealthMetrics{HealthCheckInterval: 5 * time.Minute},
			StartTime:            time.Now(),
		},
		alerts: &AlertManager{
			logger:     config.Logger,
			alerts:     make([]Alert, 0),
			thresholds: config.AlertThresholds,
		},
	}

	return mc
}

// RecordBackupOperation records metrics for a backup operation
func (mc *MetricsCollector) RecordBackupOperation(success bool, duration time.Duration, size int64, compressionRatio float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := mc.metrics.BackupOperations
	metrics.Total++

	if success {
		metrics.Success++
	} else {
		metrics.Failed++
	}

	metrics.SuccessRate = float64(metrics.Success) / float64(metrics.Total)

	// Update duration metrics
	if metrics.MinDuration == 0 || duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	// Calculate average duration
	if metrics.Total > 0 {
		totalDuration := time.Duration(int64(metrics.AverageDuration)*(metrics.Total-1)) + duration
		metrics.AverageDuration = totalDuration / time.Duration(metrics.Total)
	}

	// Update performance metrics
	if success && size > 0 {
		perf := mc.metrics.Performance
		if perf.AverageBackupSize == 0 {
			perf.AverageBackupSize = size
		} else {
			perf.AverageBackupSize = (perf.AverageBackupSize + size) / 2
		}

		if compressionRatio > 0 {
			if perf.AverageCompressionRatio == 0 {
				perf.AverageCompressionRatio = compressionRatio
			} else {
				perf.AverageCompressionRatio = (perf.AverageCompressionRatio + compressionRatio) / 2
			}
		}

		// Calculate throughput (MB/s)
		if duration.Seconds() > 0 {
			throughput := float64(size) / (1024 * 1024) / duration.Seconds()
			if perf.BackupThroughput == 0 {
				perf.BackupThroughput = throughput
			} else {
				perf.BackupThroughput = (perf.BackupThroughput + throughput) / 2
			}
		}
	}

	mc.metrics.LastUpdate = time.Now()

	// Check for alerts
	mc.checkBackupAlerts()
}

// RecordRollbackOperation records metrics for a rollback operation
func (mc *MetricsCollector) RecordRollbackOperation(success bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := mc.metrics.RollbackOperations
	metrics.Total++

	if success {
		metrics.Success++
	} else {
		metrics.Failed++
	}

	metrics.SuccessRate = float64(metrics.Success) / float64(metrics.Total)

	// Update duration metrics
	if metrics.MinDuration == 0 || duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	if metrics.Total > 0 {
		totalDuration := time.Duration(int64(metrics.AverageDuration)*(metrics.Total-1)) + duration
		metrics.AverageDuration = totalDuration / time.Duration(metrics.Total)
	}

	mc.metrics.LastUpdate = time.Now()
}

// RecordValidationOperation records metrics for a validation operation
func (mc *MetricsCollector) RecordValidationOperation(success bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := mc.metrics.ValidationOperations
	metrics.Total++

	if success {
		metrics.Success++
	} else {
		metrics.Failed++
	}

	metrics.SuccessRate = float64(metrics.Success) / float64(metrics.Total)

	// Update duration metrics
	if metrics.MinDuration == 0 || duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	if metrics.Total > 0 {
		totalDuration := time.Duration(int64(metrics.AverageDuration)*(metrics.Total-1)) + duration
		metrics.AverageDuration = totalDuration / time.Duration(metrics.Total)
	}

	// Update performance metrics
	perf := mc.metrics.Performance
	if perf.AverageValidationTime == 0 {
		perf.AverageValidationTime = duration
	} else {
		perf.AverageValidationTime = (perf.AverageValidationTime + duration) / 2
	}

	mc.metrics.LastUpdate = time.Now()

	// Check for validation alerts
	mc.checkValidationAlerts()
}

// RecordStorageUsage records storage usage metrics
func (mc *MetricsCollector) RecordStorageUsage(provider string, totalBackups int64, totalSize int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	storage := mc.metrics.Storage
	storage.TotalBackups = totalBackups
	storage.TotalStorageUsed = totalSize
	storage.StorageByProvider[provider] = totalSize

	mc.metrics.LastUpdate = time.Now()

	// Check for storage alerts
	mc.checkStorageAlerts()
}

// UpdateHealthStatus updates the overall health status
func (mc *MetricsCollector) UpdateHealthStatus() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	health := mc.metrics.Health
	health.LastHealthCheck = time.Now()

	// Calculate recent failure rate (last 24 hours)
	backupOps := mc.metrics.BackupOperations
	if backupOps.RecentTotal > 0 {
		health.RecentFailureRate = float64(backupOps.RecentFailed) / float64(backupOps.RecentTotal)
	}

	// Determine component health statuses
	health.ValidationHealth = mc.getValidationHealth()
	health.StorageHealth = mc.getStorageHealth()
	health.PerformanceHealth = mc.getPerformanceHealth()

	// Determine overall health
	health.OverallHealth = mc.getOverallHealth()

	mc.metrics.LastUpdate = time.Now()
}

// getValidationHealth determines validation component health
func (mc *MetricsCollector) getValidationHealth() HealthStatus {
	validationOps := mc.metrics.ValidationOperations

	if validationOps.Total == 0 {
		return HealthStatusUnknown
	}

	if validationOps.RecentFailed >= mc.alerts.thresholds.ValidationFailureThreshold {
		return HealthStatusCritical
	}

	if validationOps.SuccessRate < 0.95 { // Less than 95% success rate
		return HealthStatusWarning
	}

	return HealthStatusHealthy
}

// getStorageHealth determines storage component health
func (mc *MetricsCollector) getStorageHealth() HealthStatus {
	// This would typically check actual storage quotas
	// For now, we'll use placeholder logic
	utilization := mc.metrics.Health.StorageUtilization

	if utilization >= mc.alerts.thresholds.StorageCriticalPercent {
		return HealthStatusCritical
	}

	if utilization >= mc.alerts.thresholds.StorageWarningPercent {
		return HealthStatusWarning
	}

	return HealthStatusHealthy
}

// getPerformanceHealth determines performance component health
func (mc *MetricsCollector) getPerformanceHealth() HealthStatus {
	degradation := mc.metrics.Health.PerformanceDegradation

	if degradation >= mc.alerts.thresholds.PerformanceDegradationCritical {
		return HealthStatusCritical
	}

	if degradation >= mc.alerts.thresholds.PerformanceDegradationWarning {
		return HealthStatusWarning
	}

	return HealthStatusHealthy
}

// getOverallHealth determines overall system health
func (mc *MetricsCollector) getOverallHealth() HealthStatus {
	health := mc.metrics.Health

	// If any component is critical, overall is critical
	if health.ValidationHealth == HealthStatusCritical ||
		health.StorageHealth == HealthStatusCritical ||
		health.PerformanceHealth == HealthStatusCritical {
		return HealthStatusCritical
	}

	// If any component has warnings, overall has warnings
	if health.ValidationHealth == HealthStatusWarning ||
		health.StorageHealth == HealthStatusWarning ||
		health.PerformanceHealth == HealthStatusWarning {
		return HealthStatusWarning
	}

	return HealthStatusHealthy
}

// checkBackupAlerts checks for backup-related alerts
func (mc *MetricsCollector) checkBackupAlerts() {
	backupOps := mc.metrics.BackupOperations

	// Check failure rate
	if backupOps.SuccessRate < (1.0 - mc.alerts.thresholds.FailureRateCritical) {
		mc.alerts.createAlert(AlertTypeBackupFailure, AlertSeverityCritical,
			"High Backup Failure Rate",
			fmt.Sprintf("Backup failure rate is %.2f%%, exceeding critical threshold",
				(1.0-backupOps.SuccessRate)*100),
			map[string]interface{}{
				"failure_rate":      (1.0 - backupOps.SuccessRate) * 100,
				"total_operations":  backupOps.Total,
				"failed_operations": backupOps.Failed,
			})
	} else if backupOps.SuccessRate < (1.0 - mc.alerts.thresholds.FailureRateWarning) {
		mc.alerts.createAlert(AlertTypeBackupFailure, AlertSeverityWarning,
			"Elevated Backup Failure Rate",
			fmt.Sprintf("Backup failure rate is %.2f%%, exceeding warning threshold",
				(1.0-backupOps.SuccessRate)*100),
			map[string]interface{}{
				"failure_rate":      (1.0 - backupOps.SuccessRate) * 100,
				"total_operations":  backupOps.Total,
				"failed_operations": backupOps.Failed,
			})
	}
}

// checkValidationAlerts checks for validation-related alerts
func (mc *MetricsCollector) checkValidationAlerts() {
	validationOps := mc.metrics.ValidationOperations

	if validationOps.RecentFailed >= mc.alerts.thresholds.ValidationFailureThreshold {
		mc.alerts.createAlert(AlertTypeValidationFailure, AlertSeverityCritical,
			"Multiple Validation Failures",
			fmt.Sprintf("Recent validation failures: %d, exceeding threshold", validationOps.RecentFailed),
			map[string]interface{}{
				"recent_failures": validationOps.RecentFailed,
				"threshold":       mc.alerts.thresholds.ValidationFailureThreshold,
			})
	}
}

// checkStorageAlerts checks for storage-related alerts
func (mc *MetricsCollector) checkStorageAlerts() {
	utilization := mc.metrics.Health.StorageUtilization

	if utilization >= mc.alerts.thresholds.StorageCriticalPercent {
		mc.alerts.createAlert(AlertTypeStorageQuota, AlertSeverityCritical,
			"Storage Quota Critical",
			fmt.Sprintf("Storage utilization is %.1f%%, exceeding critical threshold", utilization),
			map[string]interface{}{
				"utilization": utilization,
				"threshold":   mc.alerts.thresholds.StorageCriticalPercent,
			})
	} else if utilization >= mc.alerts.thresholds.StorageWarningPercent {
		mc.alerts.createAlert(AlertTypeStorageQuota, AlertSeverityWarning,
			"Storage Quota Warning",
			fmt.Sprintf("Storage utilization is %.1f%%, exceeding warning threshold", utilization),
			map[string]interface{}{
				"utilization": utilization,
				"threshold":   mc.alerts.thresholds.StorageWarningPercent,
			})
	}
}

// createAlert creates a new alert
func (am *AlertManager) createAlert(alertType AlertType, severity AlertSeverity, title, message string, metadata map[string]interface{}) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Resolved:  false,
		Metadata:  metadata,
	}

	am.alerts = append(am.alerts, alert)

	// Log the alert
	am.logger.WithFields(map[string]interface{}{
		"alert_id":   alert.ID,
		"alert_type": string(alert.Type),
		"severity":   string(alert.Severity),
		"title":      alert.Title,
		"metadata":   alert.Metadata,
	}).Warn("Backup system alert generated")
}

// GetActiveAlerts returns all active (unresolved) alerts
func (mc *MetricsCollector) GetActiveAlerts() []Alert {
	mc.alerts.mu.RLock()
	defer mc.alerts.mu.RUnlock()

	var activeAlerts []Alert
	for _, alert := range mc.alerts.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// ResolveAlert marks an alert as resolved
func (mc *MetricsCollector) ResolveAlert(alertID string) error {
	mc.alerts.mu.Lock()
	defer mc.alerts.mu.Unlock()

	for i, alert := range mc.alerts.alerts {
		if alert.ID == alertID {
			now := time.Now()
			mc.alerts.alerts[i].Resolved = true
			mc.alerts.alerts[i].ResolvedAt = &now

			mc.logger.WithFields(map[string]interface{}{
				"alert_id":    alertID,
				"resolved_at": now,
			}).Info("Alert resolved")

			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetMetrics returns a copy of current metrics
func (mc *MetricsCollector) GetMetrics() BackupMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to avoid race conditions
	return *mc.metrics
}

// GenerateReport generates a comprehensive backup system report
func (mc *MetricsCollector) GenerateReport(ctx context.Context) (*BackupSystemReport, error) {
	mc.mu.RLock()
	metrics := *mc.metrics
	mc.mu.RUnlock()

	mc.alerts.mu.RLock()
	alerts := make([]Alert, len(mc.alerts.alerts))
	copy(alerts, mc.alerts.alerts)
	mc.alerts.mu.RUnlock()

	report := &BackupSystemReport{
		GeneratedAt: time.Now(),
		Metrics:     metrics,
		Alerts:      alerts,
		Summary:     mc.generateSummary(metrics, alerts),
	}

	// Save report to file if path is configured
	if mc.reportPath != "" {
		if err := mc.saveReport(report); err != nil {
			mc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
				"path":  mc.reportPath,
			}).Error("Failed to save backup system report")
			return report, fmt.Errorf("failed to save report: %w", err)
		}
	}

	return report, nil
}

// BackupSystemReport represents a comprehensive system report
type BackupSystemReport struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Metrics     BackupMetrics `json:"metrics"`
	Alerts      []Alert       `json:"alerts"`
	Summary     ReportSummary `json:"summary"`
}

// ReportSummary provides a high-level summary of the backup system
type ReportSummary struct {
	OverallHealth      HealthStatus `json:"overall_health"`
	TotalBackups       int64        `json:"total_backups"`
	ActiveAlerts       int          `json:"active_alerts"`
	CriticalAlerts     int          `json:"critical_alerts"`
	BackupSuccessRate  float64      `json:"backup_success_rate"`
	StorageUtilization float64      `json:"storage_utilization"`
	RecentPerformance  string       `json:"recent_performance"`
	Recommendations    []string     `json:"recommendations"`
}

// generateSummary generates a summary of the backup system status
func (mc *MetricsCollector) generateSummary(metrics BackupMetrics, alerts []Alert) ReportSummary {
	summary := ReportSummary{
		OverallHealth:      metrics.Health.OverallHealth,
		TotalBackups:       metrics.Storage.TotalBackups,
		BackupSuccessRate:  metrics.BackupOperations.SuccessRate,
		StorageUtilization: metrics.Health.StorageUtilization,
	}

	// Count active and critical alerts
	for _, alert := range alerts {
		if !alert.Resolved {
			summary.ActiveAlerts++
			if alert.Severity == AlertSeverityCritical {
				summary.CriticalAlerts++
			}
		}
	}

	// Generate performance assessment
	if metrics.Performance.BackupThroughput > 10.0 {
		summary.RecentPerformance = "Good"
	} else if metrics.Performance.BackupThroughput > 5.0 {
		summary.RecentPerformance = "Fair"
	} else {
		summary.RecentPerformance = "Poor"
	}

	// Generate recommendations
	summary.Recommendations = mc.generateRecommendations(metrics, alerts)

	return summary
}

// generateRecommendations generates system recommendations based on metrics and alerts
func (mc *MetricsCollector) generateRecommendations(metrics BackupMetrics, alerts []Alert) []string {
	var recommendations []string

	// Check backup success rate
	if metrics.BackupOperations.SuccessRate < 0.95 {
		recommendations = append(recommendations,
			"Consider investigating backup failures and improving backup reliability")
	}

	// Check storage utilization
	if metrics.Health.StorageUtilization > 80 {
		recommendations = append(recommendations,
			"Review retention policies to manage storage usage")
	}

	// Check performance
	if metrics.Performance.BackupThroughput < 5.0 {
		recommendations = append(recommendations,
			"Consider optimizing backup performance or upgrading storage infrastructure")
	}

	// Check validation failures
	if metrics.ValidationOperations.SuccessRate < 0.98 {
		recommendations = append(recommendations,
			"Investigate validation failures to ensure backup integrity")
	}

	// Check for critical alerts
	criticalAlerts := 0
	for _, alert := range alerts {
		if !alert.Resolved && alert.Severity == AlertSeverityCritical {
			criticalAlerts++
		}
	}

	if criticalAlerts > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Address %d critical alerts immediately", criticalAlerts))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System is operating normally")
	}

	return recommendations
}

// saveReport saves the report to a file
func (mc *MetricsCollector) saveReport(report *BackupSystemReport) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(mc.reportPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate timestamped filename
	timestamp := report.GeneratedAt.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup-report_%s.json", timestamp)
	fullPath := filepath.Join(dir, filename)

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	mc.logger.WithFields(map[string]interface{}{
		"report_path": fullPath,
		"report_size": len(data),
	}).Info("Backup system report saved")

	return nil
}
