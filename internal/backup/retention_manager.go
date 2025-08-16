package backup

import (
	"context"
	"fmt"
	"sort"
	"time"

	"mysql-schema-sync/internal/logging"
)

// RetentionManager handles backup retention policies and cleanup operations
type RetentionManager interface {
	// ApplyRetentionPolicy applies retention policies to clean up old backups
	ApplyRetentionPolicy(ctx context.Context, databaseName string, dryRun bool) (*RetentionResult, error)

	// ApplyRetentionPolicyToAll applies retention policies to all databases
	ApplyRetentionPolicyToAll(ctx context.Context, dryRun bool) (*RetentionResult, error)

	// GetRetentionCandidates returns backups that would be deleted by retention policy
	GetRetentionCandidates(ctx context.Context, databaseName string) ([]*BackupMetadata, error)

	// ValidateRetentionPolicy validates the retention configuration
	ValidateRetentionPolicy() error

	// ScheduleCleanup schedules automatic cleanup based on cleanup interval
	ScheduleCleanup(ctx context.Context) error

	// GetRetentionReport generates a report of retention policy effectiveness
	GetRetentionReport(ctx context.Context) (*RetentionReport, error)

	// GetCleanupReport generates a detailed report of cleanup operations
	GetCleanupReport(ctx context.Context, databaseName string, dryRun bool) (*CleanupReport, error)

	// GetCleanupHistory returns a history of cleanup operations
	GetCleanupHistory(ctx context.Context, limit int) ([]*CleanupHistoryEntry, error)

	// ScheduleCleanupWithReporting schedules cleanup with detailed reporting
	ScheduleCleanupWithReporting(ctx context.Context, reportCallback func(*CleanupReport)) error
}

// RetentionResult represents the result of applying retention policies
type RetentionResult struct {
	TotalBackupsProcessed int                    `json:"total_backups_processed"`
	BackupsDeleted        int                    `json:"backups_deleted"`
	BackupsKept           int                    `json:"backups_kept"`
	DeletedBackups        []*BackupMetadata      `json:"deleted_backups"`
	KeptBackups           []*BackupMetadata      `json:"kept_backups"`
	Errors                []string               `json:"errors"`
	ProcessingTime        time.Duration          `json:"processing_time"`
	DryRun                bool                   `json:"dry_run"`
	PolicyApplied         *RetentionPolicyStatus `json:"policy_applied"`
}

// RetentionPolicyStatus shows which retention policies were applied
type RetentionPolicyStatus struct {
	MaxBackupsApplied bool `json:"max_backups_applied"`
	MaxAgeApplied     bool `json:"max_age_applied"`
	DailyApplied      bool `json:"daily_applied"`
	WeeklyApplied     bool `json:"weekly_applied"`
	MonthlyApplied    bool `json:"monthly_applied"`
}

// RetentionReport provides insights into retention policy effectiveness
type RetentionReport struct {
	TotalBackups        int                         `json:"total_backups"`
	BackupsByDatabase   map[string]int              `json:"backups_by_database"`
	BackupsByAge        map[string]int              `json:"backups_by_age"` // "daily", "weekly", "monthly", "older"
	StorageUsage        int64                       `json:"storage_usage"`
	EstimatedSavings    int64                       `json:"estimated_savings"`
	RecommendedPolicies *RecommendedRetentionPolicy `json:"recommended_policies"`
	GeneratedAt         time.Time                   `json:"generated_at"`
}

// RecommendedRetentionPolicy suggests optimal retention settings
type RecommendedRetentionPolicy struct {
	MaxBackups  int           `json:"max_backups"`
	MaxAge      time.Duration `json:"max_age"`
	KeepDaily   int           `json:"keep_daily"`
	KeepWeekly  int           `json:"keep_weekly"`
	KeepMonthly int           `json:"keep_monthly"`
	Reasoning   string        `json:"reasoning"`
}

// retentionManager implements the RetentionManager interface
type retentionManager struct {
	backupManager BackupManager
	config        *RetentionConfig
	logger        *logging.Logger
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager(backupManager BackupManager, config *RetentionConfig, logger *logging.Logger) RetentionManager {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &retentionManager{
		backupManager: backupManager,
		config:        config,
		logger:        logger,
	}
}

// ApplyRetentionPolicy applies retention policies to clean up old backups for a specific database
func (rm *retentionManager) ApplyRetentionPolicy(ctx context.Context, databaseName string, dryRun bool) (*RetentionResult, error) {
	startTime := time.Now()

	rm.logger.Info(fmt.Sprintf("Applying retention policy for database: %s (dry run: %v)", databaseName, dryRun))

	// Get all backups for the database
	backups, err := rm.backupManager.GetBackupsByDatabase(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backups for database %s: %w", databaseName, err)
	}

	if len(backups) == 0 {
		rm.logger.Info(fmt.Sprintf("No backups found for database: %s", databaseName))
		return &RetentionResult{
			TotalBackupsProcessed: 0,
			BackupsDeleted:        0,
			BackupsKept:           0,
			ProcessingTime:        time.Since(startTime),
			DryRun:                dryRun,
		}, nil
	}

	// Sort backups by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Apply retention policies
	toDelete, toKeep, policyStatus := rm.applyRetentionRules(backups)

	result := &RetentionResult{
		TotalBackupsProcessed: len(backups),
		BackupsDeleted:        len(toDelete),
		BackupsKept:           len(toKeep),
		DeletedBackups:        toDelete,
		KeptBackups:           toKeep,
		ProcessingTime:        time.Since(startTime),
		DryRun:                dryRun,
		PolicyApplied:         policyStatus,
	}

	// Delete backups if not a dry run
	if !dryRun && len(toDelete) > 0 {
		var deleteErrors []string
		for _, backup := range toDelete {
			if err := rm.backupManager.DeleteBackup(ctx, backup.ID); err != nil {
				errorMsg := fmt.Sprintf("failed to delete backup %s: %v", backup.ID, err)
				deleteErrors = append(deleteErrors, errorMsg)
				rm.logger.Error(errorMsg)
			} else {
				rm.logBackupCleanup(backup, "retention_policy")
				rm.logger.Info(fmt.Sprintf("Deleted backup: %s (created: %s)", backup.ID, backup.CreatedAt.Format(time.RFC3339)))
			}
		}
		result.Errors = deleteErrors
	}

	rm.logger.Info(fmt.Sprintf("Retention policy applied for %s: %d processed, %d to delete, %d to keep",
		databaseName, result.TotalBackupsProcessed, result.BackupsDeleted, result.BackupsKept))

	return result, nil
}

// ApplyRetentionPolicyToAll applies retention policies to all databases
func (rm *retentionManager) ApplyRetentionPolicyToAll(ctx context.Context, dryRun bool) (*RetentionResult, error) {
	startTime := time.Now()

	rm.logger.Info(fmt.Sprintf("Applying retention policy to all databases (dry run: %v)", dryRun))

	// Get all backups
	allBackups, err := rm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all backups: %w", err)
	}

	if len(allBackups) == 0 {
		rm.logger.Info("No backups found")
		return &RetentionResult{
			TotalBackupsProcessed: 0,
			BackupsDeleted:        0,
			BackupsKept:           0,
			ProcessingTime:        time.Since(startTime),
			DryRun:                dryRun,
		}, nil
	}

	// Group backups by database
	backupsByDatabase := make(map[string][]*BackupMetadata)
	for _, backup := range allBackups {
		backupsByDatabase[backup.DatabaseName] = append(backupsByDatabase[backup.DatabaseName], backup)
	}

	// Apply retention policies to each database
	var allToDelete []*BackupMetadata
	var allToKeep []*BackupMetadata
	var allErrors []string
	policyStatus := &RetentionPolicyStatus{}

	for databaseName, backups := range backupsByDatabase {
		// Sort backups by creation time (newest first)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})

		toDelete, toKeep, dbPolicyStatus := rm.applyRetentionRules(backups)
		allToDelete = append(allToDelete, toDelete...)
		allToKeep = append(allToKeep, toKeep...)

		// Merge policy status
		policyStatus.MaxBackupsApplied = policyStatus.MaxBackupsApplied || dbPolicyStatus.MaxBackupsApplied
		policyStatus.MaxAgeApplied = policyStatus.MaxAgeApplied || dbPolicyStatus.MaxAgeApplied
		policyStatus.DailyApplied = policyStatus.DailyApplied || dbPolicyStatus.DailyApplied
		policyStatus.WeeklyApplied = policyStatus.WeeklyApplied || dbPolicyStatus.WeeklyApplied
		policyStatus.MonthlyApplied = policyStatus.MonthlyApplied || dbPolicyStatus.MonthlyApplied

		rm.logger.Debug(fmt.Sprintf("Database %s: %d to delete, %d to keep", databaseName, len(toDelete), len(toKeep)))
	}

	result := &RetentionResult{
		TotalBackupsProcessed: len(allBackups),
		BackupsDeleted:        len(allToDelete),
		BackupsKept:           len(allToKeep),
		DeletedBackups:        allToDelete,
		KeptBackups:           allToKeep,
		ProcessingTime:        time.Since(startTime),
		DryRun:                dryRun,
		PolicyApplied:         policyStatus,
	}

	// Delete backups if not a dry run
	if !dryRun && len(allToDelete) > 0 {
		for _, backup := range allToDelete {
			if err := rm.backupManager.DeleteBackup(ctx, backup.ID); err != nil {
				errorMsg := fmt.Sprintf("failed to delete backup %s: %v", backup.ID, err)
				allErrors = append(allErrors, errorMsg)
				rm.logger.Error(errorMsg)
			} else {
				rm.logBackupCleanup(backup, "retention_policy")
				rm.logger.Info(fmt.Sprintf("Deleted backup: %s (database: %s, created: %s)",
					backup.ID, backup.DatabaseName, backup.CreatedAt.Format(time.RFC3339)))
			}
		}
		result.Errors = allErrors
	}

	rm.logger.Info(fmt.Sprintf("Retention policy applied to all databases: %d processed, %d to delete, %d to keep",
		result.TotalBackupsProcessed, result.BackupsDeleted, result.BackupsKept))

	return result, nil
}

// GetRetentionCandidates returns backups that would be deleted by retention policy
func (rm *retentionManager) GetRetentionCandidates(ctx context.Context, databaseName string) ([]*BackupMetadata, error) {
	rm.logger.Debug(fmt.Sprintf("Getting retention candidates for database: %s", databaseName))

	// Get all backups for the database
	backups, err := rm.backupManager.GetBackupsByDatabase(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backups for database %s: %w", databaseName, err)
	}

	if len(backups) == 0 {
		return []*BackupMetadata{}, nil
	}

	// Sort backups by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Apply retention rules to get candidates for deletion
	toDelete, _, _ := rm.applyRetentionRules(backups)

	return toDelete, nil
}

// ValidateRetentionPolicy validates the retention configuration
func (rm *retentionManager) ValidateRetentionPolicy() error {
	if rm.config == nil {
		return fmt.Errorf("retention configuration is nil")
	}

	return rm.config.Validate()
}

// ScheduleCleanup schedules automatic cleanup based on cleanup interval
func (rm *retentionManager) ScheduleCleanup(ctx context.Context) error {
	if rm.config.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive")
	}

	rm.logger.Info(fmt.Sprintf("Scheduling automatic cleanup every %v", rm.config.CleanupInterval))

	ticker := time.NewTicker(rm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rm.logger.Info("Cleanup scheduler stopped")
			return ctx.Err()
		case <-ticker.C:
			rm.logger.Info("Running scheduled cleanup")
			result, err := rm.ApplyRetentionPolicyToAll(ctx, false)
			if err != nil {
				rm.logger.Error(fmt.Sprintf("Scheduled cleanup failed: %v", err))
			} else {
				rm.logger.Info(fmt.Sprintf("Scheduled cleanup completed: %d deleted, %d kept",
					result.BackupsDeleted, result.BackupsKept))
			}
		}
	}
}

// GetRetentionReport generates a report of retention policy effectiveness
func (rm *retentionManager) GetRetentionReport(ctx context.Context) (*RetentionReport, error) {
	rm.logger.Debug("Generating retention report")

	// Get all backups
	allBackups, err := rm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups for report: %w", err)
	}

	report := &RetentionReport{
		TotalBackups:      len(allBackups),
		BackupsByDatabase: make(map[string]int),
		BackupsByAge:      make(map[string]int),
		GeneratedAt:       time.Now(),
	}

	now := time.Now()
	var totalSize int64
	var estimatedSavings int64

	// Analyze backups
	for _, backup := range allBackups {
		// Count by database
		report.BackupsByDatabase[backup.DatabaseName]++

		// Count by age
		age := now.Sub(backup.CreatedAt)
		switch {
		case age <= 24*time.Hour:
			report.BackupsByAge["daily"]++
		case age <= 7*24*time.Hour:
			report.BackupsByAge["weekly"]++
		case age <= 30*24*time.Hour:
			report.BackupsByAge["monthly"]++
		default:
			report.BackupsByAge["older"]++
		}

		// Calculate storage usage
		totalSize += backup.Size
	}

	report.StorageUsage = totalSize

	// Calculate estimated savings from retention policy
	for databaseName := range report.BackupsByDatabase {
		candidates, err := rm.GetRetentionCandidates(ctx, databaseName)
		if err != nil {
			rm.logger.Error(fmt.Sprintf("Failed to get retention candidates for %s: %v", databaseName, err))
			continue
		}

		for _, candidate := range candidates {
			estimatedSavings += candidate.Size
		}
	}

	report.EstimatedSavings = estimatedSavings

	// Generate recommendations
	report.RecommendedPolicies = rm.generateRecommendations(report)

	return report, nil
}

// applyRetentionRules applies all retention rules and returns backups to delete and keep
func (rm *retentionManager) applyRetentionRules(backups []*BackupMetadata) ([]*BackupMetadata, []*BackupMetadata, *RetentionPolicyStatus) {
	if len(backups) == 0 {
		return []*BackupMetadata{}, []*BackupMetadata{}, &RetentionPolicyStatus{}
	}

	// Create a map to track which backups to keep
	keepMap := make(map[string]bool)
	policyStatus := &RetentionPolicyStatus{}

	// Always keep at least one backup (the newest)
	if len(backups) > 0 {
		keepMap[backups[0].ID] = true
	}

	// Apply max backups policy
	if rm.config.MaxBackups > 0 {
		policyStatus.MaxBackupsApplied = true
		for i := 0; i < len(backups) && i < rm.config.MaxBackups; i++ {
			keepMap[backups[i].ID] = true
		}
	}

	// Apply max age policy
	if rm.config.MaxAge > 0 {
		policyStatus.MaxAgeApplied = true
		cutoffTime := time.Now().Add(-rm.config.MaxAge)
		for _, backup := range backups {
			if backup.CreatedAt.After(cutoffTime) {
				keepMap[backup.ID] = true
			}
		}
	}

	// Apply daily retention policy
	if rm.config.KeepDaily > 0 {
		policyStatus.DailyApplied = true
		rm.applyPeriodicRetention(backups, keepMap, rm.config.KeepDaily, 24*time.Hour)
	}

	// Apply weekly retention policy
	if rm.config.KeepWeekly > 0 {
		policyStatus.WeeklyApplied = true
		rm.applyPeriodicRetention(backups, keepMap, rm.config.KeepWeekly, 7*24*time.Hour)
	}

	// Apply monthly retention policy
	if rm.config.KeepMonthly > 0 {
		policyStatus.MonthlyApplied = true
		rm.applyPeriodicRetention(backups, keepMap, rm.config.KeepMonthly, 30*24*time.Hour)
	}

	// Separate backups into keep and delete lists
	var toKeep []*BackupMetadata
	var toDelete []*BackupMetadata

	for _, backup := range backups {
		if keepMap[backup.ID] {
			toKeep = append(toKeep, backup)
		} else {
			// Additional safety checks before marking for deletion
			if rm.shouldProtectBackup(backup) {
				toKeep = append(toKeep, backup)
			} else {
				toDelete = append(toDelete, backup)
			}
		}
	}

	return toDelete, toKeep, policyStatus
}

// applyPeriodicRetention applies periodic retention (daily, weekly, monthly)
func (rm *retentionManager) applyPeriodicRetention(backups []*BackupMetadata, keepMap map[string]bool, keepCount int, period time.Duration) {
	if keepCount <= 0 {
		return
	}

	now := time.Now()
	periodBuckets := make(map[int][]*BackupMetadata)

	// Group backups by period
	for _, backup := range backups {
		periodIndex := int(now.Sub(backup.CreatedAt) / period)
		periodBuckets[periodIndex] = append(periodBuckets[periodIndex], backup)
	}

	// Keep the newest backup from each of the most recent periods
	periods := make([]int, 0, len(periodBuckets))
	for period := range periodBuckets {
		periods = append(periods, period)
	}
	sort.Ints(periods)

	kept := 0
	for _, period := range periods {
		if kept >= keepCount {
			break
		}

		bucketBackups := periodBuckets[period]
		if len(bucketBackups) > 0 {
			// Sort by creation time (newest first) and keep the newest
			sort.Slice(bucketBackups, func(i, j int) bool {
				return bucketBackups[i].CreatedAt.After(bucketBackups[j].CreatedAt)
			})
			keepMap[bucketBackups[0].ID] = true
			kept++
		}
	}
}

// shouldProtectBackup determines if a backup should be protected from deletion
func (rm *retentionManager) shouldProtectBackup(backup *BackupMetadata) bool {
	// Protect pre-migration backups that are less than 7 days old
	if backupType, exists := backup.Tags["type"]; exists && backupType == "pre-migration" {
		if time.Since(backup.CreatedAt) < 7*24*time.Hour {
			return true
		}
	}

	// Protect backups with specific protection tags
	if protected, exists := backup.Tags["protected"]; exists && protected == "true" {
		return true
	}

	// Protect failed backups for debugging (keep for 24 hours)
	if backup.Status == BackupStatusFailed && time.Since(backup.CreatedAt) < 24*time.Hour {
		return true
	}

	return false
}

// generateRecommendations generates retention policy recommendations based on current usage
func (rm *retentionManager) generateRecommendations(report *RetentionReport) *RecommendedRetentionPolicy {
	recommendations := &RecommendedRetentionPolicy{}

	// Base recommendations on current usage patterns
	totalBackups := report.TotalBackups

	if totalBackups == 0 {
		recommendations.MaxBackups = 10
		recommendations.MaxAge = 30 * 24 * time.Hour
		recommendations.KeepDaily = 7
		recommendations.KeepWeekly = 4
		recommendations.KeepMonthly = 3
		recommendations.Reasoning = "Default recommendations for new installations"
		return recommendations
	}

	// Calculate recommendations based on usage
	dailyBackups := report.BackupsByAge["daily"]
	weeklyBackups := report.BackupsByAge["weekly"]
	monthlyBackups := report.BackupsByAge["monthly"]
	olderBackups := report.BackupsByAge["older"]

	// Recommend max backups based on current distribution
	if totalBackups > 50 {
		recommendations.MaxBackups = 30
	} else if totalBackups > 20 {
		recommendations.MaxBackups = 20
	} else {
		recommendations.MaxBackups = 15
	}

	// Recommend max age based on oldest backups
	if olderBackups > totalBackups/2 {
		recommendations.MaxAge = 60 * 24 * time.Hour // 60 days
	} else {
		recommendations.MaxAge = 90 * 24 * time.Hour // 90 days
	}

	// Recommend periodic retention based on current patterns
	if dailyBackups > 10 {
		recommendations.KeepDaily = 14
	} else {
		recommendations.KeepDaily = 7
	}

	if weeklyBackups > 8 {
		recommendations.KeepWeekly = 8
	} else {
		recommendations.KeepWeekly = 4
	}

	if monthlyBackups > 6 {
		recommendations.KeepMonthly = 6
	} else {
		recommendations.KeepMonthly = 3
	}

	// Generate reasoning
	savingsPercent := float64(report.EstimatedSavings) / float64(report.StorageUsage) * 100
	recommendations.Reasoning = fmt.Sprintf(
		"Based on %d total backups across %d databases. Current policy could save %.1f%% storage (%d bytes). Recommendations balance retention needs with storage efficiency.",
		totalBackups, len(report.BackupsByDatabase), savingsPercent, report.EstimatedSavings)

	return recommendations
}

// logBackupCleanup logs detailed information about backup cleanup
func (rm *retentionManager) logBackupCleanup(backup *BackupMetadata, reason string) {
	cleanupInfo := map[string]interface{}{
		"backup_id":       backup.ID,
		"database_name":   backup.DatabaseName,
		"created_at":      backup.CreatedAt.Format(time.RFC3339),
		"size":            backup.Size,
		"compressed_size": backup.CompressedSize,
		"age_days":        int(time.Since(backup.CreatedAt).Hours() / 24),
		"cleanup_reason":  reason,
		"cleanup_time":    time.Now().Format(time.RFC3339),
	}

	// Add migration context if available
	if backup.MigrationContext != nil {
		cleanupInfo["migration_context"] = map[string]interface{}{
			"plan_hash":        backup.MigrationContext.PlanHash,
			"source_schema":    backup.MigrationContext.SourceSchema,
			"pre_migration_id": backup.MigrationContext.PreMigrationID,
			"migration_time":   backup.MigrationContext.MigrationTime.Format(time.RFC3339),
			"tool_version":     backup.MigrationContext.ToolVersion,
		}
	}

	// Add tags if available
	if len(backup.Tags) > 0 {
		cleanupInfo["tags"] = backup.Tags
	}

	rm.logger.Info(fmt.Sprintf("Backup cleanup executed: %+v", cleanupInfo))
}

// GetCleanupReport generates a detailed report of cleanup operations
func (rm *retentionManager) GetCleanupReport(ctx context.Context, databaseName string, dryRun bool) (*CleanupReport, error) {
	startTime := time.Now()

	rm.logger.Info(fmt.Sprintf("Generating cleanup report for database: %s (dry run: %v)", databaseName, dryRun))

	// Get retention candidates
	candidates, err := rm.GetRetentionCandidates(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get retention candidates: %w", err)
	}

	// Get all backups for the database for context
	allBackups, err := rm.backupManager.GetBackupsByDatabase(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get all backups: %w", err)
	}

	report := &CleanupReport{
		DatabaseName:          databaseName,
		TotalBackups:          len(allBackups),
		BackupsToCleanup:      len(candidates),
		BackupsToKeep:         len(allBackups) - len(candidates),
		CleanupCandidates:     candidates,
		EstimatedSpaceSavings: rm.calculateSpaceSavings(candidates),
		PolicyApplied:         rm.config,
		GeneratedAt:           time.Now(),
		ProcessingTime:        time.Since(startTime),
		DryRun:                dryRun,
	}

	// Categorize cleanup candidates by reason
	report.CleanupByReason = rm.categorizeCleanupReasons(candidates, allBackups)

	// Calculate cleanup impact
	report.CleanupImpact = rm.calculateCleanupImpact(candidates, allBackups)

	return report, nil
}

// GetCleanupHistory returns a history of cleanup operations
func (rm *retentionManager) GetCleanupHistory(ctx context.Context, limit int) ([]*CleanupHistoryEntry, error) {
	// This would typically read from a cleanup log or audit trail
	// For now, return empty history as this would require persistent storage
	rm.logger.Debug(fmt.Sprintf("Getting cleanup history (limit: %d)", limit))

	// TODO: Implement cleanup history storage and retrieval
	// This would involve storing cleanup operations in a database or log file
	// and retrieving them based on the limit parameter

	return []*CleanupHistoryEntry{}, nil
}

// ScheduleCleanupWithReporting schedules cleanup with detailed reporting
func (rm *retentionManager) ScheduleCleanupWithReporting(ctx context.Context, reportCallback func(*CleanupReport)) error {
	if rm.config.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive")
	}

	rm.logger.Info(fmt.Sprintf("Scheduling automatic cleanup with reporting every %v", rm.config.CleanupInterval))

	ticker := time.NewTicker(rm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rm.logger.Info("Cleanup scheduler with reporting stopped")
			return ctx.Err()
		case <-ticker.C:
			rm.logger.Info("Running scheduled cleanup with reporting")

			// Get all databases
			allBackups, err := rm.backupManager.ListBackups(ctx, BackupFilter{})
			if err != nil {
				rm.logger.Error(fmt.Sprintf("Failed to list backups for scheduled cleanup: %v", err))
				continue
			}

			// Group by database
			databases := make(map[string]bool)
			for _, backup := range allBackups {
				databases[backup.DatabaseName] = true
			}

			// Run cleanup for each database and generate reports
			for databaseName := range databases {
				// Generate pre-cleanup report
				preReport, err := rm.GetCleanupReport(ctx, databaseName, true)
				if err != nil {
					rm.logger.Error(fmt.Sprintf("Failed to generate pre-cleanup report for %s: %v", databaseName, err))
					continue
				}

				// Execute cleanup
				result, err := rm.ApplyRetentionPolicy(ctx, databaseName, false)
				if err != nil {
					rm.logger.Error(fmt.Sprintf("Scheduled cleanup failed for %s: %v", databaseName, err))
					continue
				}

				// Generate post-cleanup report
				_, err = rm.GetCleanupReport(ctx, databaseName, false)
				if err != nil {
					rm.logger.Error(fmt.Sprintf("Failed to generate post-cleanup report for %s: %v", databaseName, err))
					continue
				}

				// Create combined report
				combinedReport := &CleanupReport{
					DatabaseName:          databaseName,
					TotalBackups:          preReport.TotalBackups,
					BackupsToCleanup:      result.BackupsDeleted,
					BackupsToKeep:         result.BackupsKept,
					CleanupCandidates:     result.DeletedBackups,
					EstimatedSpaceSavings: rm.calculateSpaceSavings(result.DeletedBackups),
					ActualSpaceSavings:    rm.calculateSpaceSavings(result.DeletedBackups),
					PolicyApplied:         rm.config,
					GeneratedAt:           time.Now(),
					ProcessingTime:        result.ProcessingTime,
					DryRun:                false,
					Errors:                result.Errors,
				}

				// Call the report callback if provided
				if reportCallback != nil {
					reportCallback(combinedReport)
				}

				rm.logger.Info(fmt.Sprintf("Scheduled cleanup completed for %s: %d deleted, %d kept",
					databaseName, result.BackupsDeleted, result.BackupsKept))
			}
		}
	}
}

// Helper methods for reporting

func (rm *retentionManager) calculateSpaceSavings(backups []*BackupMetadata) int64 {
	var totalSavings int64
	for _, backup := range backups {
		totalSavings += backup.Size
	}
	return totalSavings
}

func (rm *retentionManager) categorizeCleanupReasons(candidates []*BackupMetadata, allBackups []*BackupMetadata) map[string][]*BackupMetadata {
	reasons := make(map[string][]*BackupMetadata)

	now := time.Now()

	for _, candidate := range candidates {
		var reason string

		// Determine cleanup reason based on retention policy
		if rm.config.MaxAge > 0 && now.Sub(candidate.CreatedAt) > rm.config.MaxAge {
			reason = "max_age_exceeded"
		} else if rm.config.MaxBackups > 0 {
			// Sort all backups by creation time to determine position
			sortedBackups := make([]*BackupMetadata, len(allBackups))
			copy(sortedBackups, allBackups)
			sort.Slice(sortedBackups, func(i, j int) bool {
				return sortedBackups[i].CreatedAt.After(sortedBackups[j].CreatedAt)
			})

			// Find position of this backup
			for i, backup := range sortedBackups {
				if backup.ID == candidate.ID && i >= rm.config.MaxBackups {
					reason = "max_backups_exceeded"
					break
				}
			}
		}

		if reason == "" {
			reason = "retention_policy"
		}

		reasons[reason] = append(reasons[reason], candidate)
	}

	return reasons
}

func (rm *retentionManager) calculateCleanupImpact(candidates []*BackupMetadata, allBackups []*BackupMetadata) *CleanupImpact {
	impact := &CleanupImpact{
		SpaceSavings:     rm.calculateSpaceSavings(candidates),
		BackupsRemoved:   len(candidates),
		BackupsRemaining: len(allBackups) - len(candidates),
	}

	if len(allBackups) > 0 {
		impact.PercentageReduction = float64(len(candidates)) / float64(len(allBackups)) * 100
	}

	// Calculate oldest backup that will be removed
	for _, candidate := range candidates {
		if impact.OldestBackupRemoved.IsZero() || candidate.CreatedAt.Before(impact.OldestBackupRemoved) {
			impact.OldestBackupRemoved = candidate.CreatedAt
		}
	}

	// Calculate average age of removed backups
	if len(candidates) > 0 {
		var totalAge time.Duration
		now := time.Now()
		for _, candidate := range candidates {
			totalAge += now.Sub(candidate.CreatedAt)
		}
		impact.AverageAgeRemoved = totalAge / time.Duration(len(candidates))
	}

	return impact
}

// Additional types for reporting

// CleanupReport provides detailed information about cleanup operations
type CleanupReport struct {
	DatabaseName          string                       `json:"database_name"`
	TotalBackups          int                          `json:"total_backups"`
	BackupsToCleanup      int                          `json:"backups_to_cleanup"`
	BackupsToKeep         int                          `json:"backups_to_keep"`
	CleanupCandidates     []*BackupMetadata            `json:"cleanup_candidates"`
	EstimatedSpaceSavings int64                        `json:"estimated_space_savings"`
	ActualSpaceSavings    int64                        `json:"actual_space_savings,omitempty"`
	CleanupByReason       map[string][]*BackupMetadata `json:"cleanup_by_reason"`
	CleanupImpact         *CleanupImpact               `json:"cleanup_impact"`
	PolicyApplied         *RetentionConfig             `json:"policy_applied"`
	GeneratedAt           time.Time                    `json:"generated_at"`
	ProcessingTime        time.Duration                `json:"processing_time"`
	DryRun                bool                         `json:"dry_run"`
	Errors                []string                     `json:"errors,omitempty"`
}

// CleanupImpact represents the impact of cleanup operations
type CleanupImpact struct {
	SpaceSavings        int64         `json:"space_savings"`
	BackupsRemoved      int           `json:"backups_removed"`
	BackupsRemaining    int           `json:"backups_remaining"`
	PercentageReduction float64       `json:"percentage_reduction"`
	OldestBackupRemoved time.Time     `json:"oldest_backup_removed"`
	AverageAgeRemoved   time.Duration `json:"average_age_removed"`
}

// CleanupHistoryEntry represents a historical cleanup operation
type CleanupHistoryEntry struct {
	ID             string           `json:"id"`
	DatabaseName   string           `json:"database_name"`
	ExecutedAt     time.Time        `json:"executed_at"`
	BackupsDeleted int              `json:"backups_deleted"`
	SpaceSaved     int64            `json:"space_saved"`
	PolicyApplied  *RetentionConfig `json:"policy_applied"`
	ExecutionTime  time.Duration    `json:"execution_time"`
	Errors         []string         `json:"errors,omitempty"`
	TriggerType    string           `json:"trigger_type"` // "manual", "scheduled", "api"
	TriggerBy      string           `json:"trigger_by"`   // User or system identifier
}
