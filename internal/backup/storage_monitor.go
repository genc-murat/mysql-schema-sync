package backup

import (
	"context"
	"fmt"
	"sort"
	"time"

	"mysql-schema-sync/internal/logging"
)

// StorageMonitor handles backup storage monitoring and optimization
type StorageMonitor interface {
	// GetStorageUsage returns current storage usage statistics
	GetStorageUsage(ctx context.Context) (*StorageUsageReport, error)

	// GetStorageUsageByDatabase returns storage usage grouped by database
	GetStorageUsageByDatabase(ctx context.Context) (map[string]*DatabaseStorageUsage, error)

	// CheckStorageQuotas checks if storage usage exceeds configured quotas
	CheckStorageQuotas(ctx context.Context) (*StorageQuotaStatus, error)

	// GetStorageOptimizationRecommendations provides recommendations for storage optimization
	GetStorageOptimizationRecommendations(ctx context.Context) (*StorageOptimizationReport, error)

	// MonitorStorageHealth performs health checks on storage providers
	MonitorStorageHealth(ctx context.Context) (*StorageHealthReport, error)

	// GetStorageTrends analyzes storage usage trends over time
	GetStorageTrends(ctx context.Context, period time.Duration) (*StorageTrendReport, error)

	// GenerateStorageAlerts generates alerts based on storage conditions
	GenerateStorageAlerts(ctx context.Context) ([]*StorageAlert, error)

	// MonitorStorageHealthWithDetails performs comprehensive health checks
	MonitorStorageHealthWithDetails(ctx context.Context) (*StorageHealthReport, error)

	// GetStorageHealthSummary provides a quick health summary
	GetStorageHealthSummary(ctx context.Context) (*StorageHealthSummary, error)
}

// StorageUsageReport provides comprehensive storage usage information
type StorageUsageReport struct {
	TotalBackups        int                       `json:"total_backups"`
	TotalSize           int64                     `json:"total_size"`
	TotalCompressedSize int64                     `json:"total_compressed_size"`
	CompressionRatio    float64                   `json:"compression_ratio"`
	AverageBackupSize   int64                     `json:"average_backup_size"`
	LargestBackup       *BackupMetadata           `json:"largest_backup"`
	SmallestBackup      *BackupMetadata           `json:"smallest_backup"`
	StorageByProvider   map[string]*ProviderUsage `json:"storage_by_provider"`
	StorageByDatabase   map[string]*DatabaseUsage `json:"storage_by_database"`
	StorageByAge        map[string]*AgeGroupUsage `json:"storage_by_age"`
	GeneratedAt         time.Time                 `json:"generated_at"`
}

// ProviderUsage represents storage usage for a specific provider
type ProviderUsage struct {
	Provider         StorageProviderType `json:"provider"`
	BackupCount      int                 `json:"backup_count"`
	TotalSize        int64               `json:"total_size"`
	CompressedSize   int64               `json:"compressed_size"`
	CompressionRatio float64             `json:"compression_ratio"`
}

// DatabaseUsage represents storage usage for a specific database
type DatabaseUsage struct {
	DatabaseName     string    `json:"database_name"`
	BackupCount      int       `json:"backup_count"`
	TotalSize        int64     `json:"total_size"`
	CompressedSize   int64     `json:"compressed_size"`
	CompressionRatio float64   `json:"compression_ratio"`
	OldestBackup     time.Time `json:"oldest_backup"`
	NewestBackup     time.Time `json:"newest_backup"`
}

// AgeGroupUsage represents storage usage by backup age groups
type AgeGroupUsage struct {
	AgeGroup         string  `json:"age_group"` // "daily", "weekly", "monthly", "older"
	BackupCount      int     `json:"backup_count"`
	TotalSize        int64   `json:"total_size"`
	CompressedSize   int64   `json:"compressed_size"`
	CompressionRatio float64 `json:"compression_ratio"`
}

// DatabaseStorageUsage provides detailed storage usage for a database
type DatabaseStorageUsage struct {
	DatabaseName      string             `json:"database_name"`
	BackupCount       int                `json:"backup_count"`
	TotalSize         int64              `json:"total_size"`
	CompressedSize    int64              `json:"compressed_size"`
	CompressionRatio  float64            `json:"compression_ratio"`
	AverageBackupSize int64              `json:"average_backup_size"`
	LargestBackup     *BackupMetadata    `json:"largest_backup"`
	SmallestBackup    *BackupMetadata    `json:"smallest_backup"`
	BackupsByStatus   map[string]int     `json:"backups_by_status"`
	BackupsByAge      map[string]int     `json:"backups_by_age"`
	StorageGrowth     *StorageGrowthInfo `json:"storage_growth"`
}

// StorageGrowthInfo tracks storage growth patterns
type StorageGrowthInfo struct {
	DailyGrowth   int64     `json:"daily_growth"`   // Bytes per day
	WeeklyGrowth  int64     `json:"weekly_growth"`  // Bytes per week
	MonthlyGrowth int64     `json:"monthly_growth"` // Bytes per month
	GrowthTrend   string    `json:"growth_trend"`   // "increasing", "stable", "decreasing"
	LastUpdated   time.Time `json:"last_updated"`
}

// StorageQuotaStatus represents storage quota status
type StorageQuotaStatus struct {
	QuotaEnabled        bool                      `json:"quota_enabled"`
	TotalQuota          int64                     `json:"total_quota"`
	UsedStorage         int64                     `json:"used_storage"`
	AvailableStorage    int64                     `json:"available_storage"`
	UsagePercentage     float64                   `json:"usage_percentage"`
	QuotaExceeded       bool                      `json:"quota_exceeded"`
	DatabaseQuotas      map[string]*DatabaseQuota `json:"database_quotas"`
	ProviderQuotas      map[string]*ProviderQuota `json:"provider_quotas"`
	QuotaWarnings       []*QuotaWarning           `json:"quota_warnings"`
	EstimatedTimeToFull time.Duration             `json:"estimated_time_to_full"`
}

// DatabaseQuota represents quota information for a database
type DatabaseQuota struct {
	DatabaseName    string  `json:"database_name"`
	Quota           int64   `json:"quota"`
	Used            int64   `json:"used"`
	Available       int64   `json:"available"`
	UsagePercentage float64 `json:"usage_percentage"`
	QuotaExceeded   bool    `json:"quota_exceeded"`
}

// ProviderQuota represents quota information for a storage provider
type ProviderQuota struct {
	Provider        StorageProviderType `json:"provider"`
	Quota           int64               `json:"quota"`
	Used            int64               `json:"used"`
	Available       int64               `json:"available"`
	UsagePercentage float64             `json:"usage_percentage"`
	QuotaExceeded   bool                `json:"quota_exceeded"`
}

// QuotaWarning represents a storage quota warning
type QuotaWarning struct {
	Type              string  `json:"type"`   // "database", "provider", "total"
	Target            string  `json:"target"` // Database name or provider name
	CurrentUsage      int64   `json:"current_usage"`
	Quota             int64   `json:"quota"`
	UsagePercentage   float64 `json:"usage_percentage"`
	Severity          string  `json:"severity"` // "warning", "critical"
	Message           string  `json:"message"`
	RecommendedAction string  `json:"recommended_action"`
}

// StorageOptimizationReport provides storage optimization recommendations
type StorageOptimizationReport struct {
	TotalPotentialSavings int64                         `json:"total_potential_savings"`
	Recommendations       []*OptimizationRecommendation `json:"recommendations"`
	CompressionAnalysis   *CompressionAnalysis          `json:"compression_analysis"`
	RetentionAnalysis     *RetentionAnalysis            `json:"retention_analysis"`
	DuplicationAnalysis   *DuplicationAnalysis          `json:"duplication_analysis"`
	GeneratedAt           time.Time                     `json:"generated_at"`
}

// OptimizationRecommendation represents a storage optimization recommendation
type OptimizationRecommendation struct {
	Type               string `json:"type"`     // "compression", "retention", "deduplication"
	Priority           string `json:"priority"` // "high", "medium", "low"
	EstimatedSavings   int64  `json:"estimated_savings"`
	Description        string `json:"description"`
	ActionRequired     string `json:"action_required"`
	Impact             string `json:"impact"`              // "low", "medium", "high"
	ImplementationTime string `json:"implementation_time"` // "immediate", "short", "long"
}

// CompressionAnalysis analyzes compression effectiveness
type CompressionAnalysis struct {
	OverallCompressionRatio     float64                              `json:"overall_compression_ratio"`
	CompressionByAlgorithm      map[string]*CompressionAnalysisStats `json:"compression_by_algorithm"`
	UncompressedBackups         []*BackupMetadata                    `json:"uncompressed_backups"`
	PoorlyCompressedBackups     []*BackupMetadata                    `json:"poorly_compressed_backups"`
	RecommendedAlgorithm        CompressionType                      `json:"recommended_algorithm"`
	PotentialCompressionSavings int64                                `json:"potential_compression_savings"`
}

// CompressionAnalysisStats represents compression statistics for analysis
type CompressionAnalysisStats struct {
	Algorithm        CompressionType `json:"algorithm"`
	BackupCount      int             `json:"backup_count"`
	OriginalSize     int64           `json:"original_size"`
	CompressedSize   int64           `json:"compressed_size"`
	CompressionRatio float64         `json:"compression_ratio"`
	AverageRatio     float64         `json:"average_ratio"`
}

// RetentionAnalysis analyzes retention policy effectiveness
type RetentionAnalysis struct {
	RetentionPolicyEffectiveness float64                    `json:"retention_policy_effectiveness"`
	BackupsEligibleForCleanup    []*BackupMetadata          `json:"backups_eligible_for_cleanup"`
	PotentialRetentionSavings    int64                      `json:"potential_retention_savings"`
	RecommendedRetentionChanges  []*RetentionRecommendation `json:"recommended_retention_changes"`
}

// RetentionRecommendation represents a retention policy recommendation
type RetentionRecommendation struct {
	DatabaseName      string `json:"database_name"`
	CurrentPolicy     string `json:"current_policy"`
	RecommendedPolicy string `json:"recommended_policy"`
	EstimatedSavings  int64  `json:"estimated_savings"`
	Reasoning         string `json:"reasoning"`
}

// DuplicationAnalysis analyzes backup duplication
type DuplicationAnalysis struct {
	DuplicateBackups              []*BackupDuplication `json:"duplicate_backups"`
	PotentialDeduplicationSavings int64                `json:"potential_deduplication_savings"`
	DeduplicationRecommendations  []string             `json:"deduplication_recommendations"`
}

// BackupDuplication represents information about duplicate backups
type BackupDuplication struct {
	Checksum         string            `json:"checksum"`
	BackupCount      int               `json:"backup_count"`
	Backups          []*BackupMetadata `json:"backups"`
	TotalSize        int64             `json:"total_size"`
	PotentialSavings int64             `json:"potential_savings"`
}

// StorageHealthReport provides storage provider health information
type StorageHealthReport struct {
	OverallHealth      string                     `json:"overall_health"` // "healthy", "warning", "critical"
	ProviderHealth     map[string]*ProviderHealth `json:"provider_health"`
	ConnectivityTests  []*ConnectivityTest        `json:"connectivity_tests"`
	PerformanceMetrics *StoragePerformanceMetrics `json:"performance_metrics"`
	HealthIssues       []*HealthIssue             `json:"health_issues"`
	GeneratedAt        time.Time                  `json:"generated_at"`
}

// ProviderHealth represents health status for a storage provider
type ProviderHealth struct {
	Provider         StorageProviderType `json:"provider"`
	Status           string              `json:"status"` // "healthy", "warning", "critical", "unavailable"
	LastSuccessfulOp time.Time           `json:"last_successful_op"`
	LastFailedOp     time.Time           `json:"last_failed_op"`
	ErrorRate        float64             `json:"error_rate"`
	ResponseTime     time.Duration       `json:"response_time"`
	Issues           []string            `json:"issues"`
}

// ConnectivityTest represents a storage connectivity test result
type ConnectivityTest struct {
	Provider     StorageProviderType `json:"provider"`
	TestType     string              `json:"test_type"` // "read", "write", "delete", "list"
	Success      bool                `json:"success"`
	ResponseTime time.Duration       `json:"response_time"`
	Error        string              `json:"error,omitempty"`
	TestedAt     time.Time           `json:"tested_at"`
}

// StoragePerformanceMetrics represents storage performance metrics
type StoragePerformanceMetrics struct {
	AverageWriteTime  time.Duration `json:"average_write_time"`
	AverageReadTime   time.Duration `json:"average_read_time"`
	AverageDeleteTime time.Duration `json:"average_delete_time"`
	ThroughputMBps    float64       `json:"throughput_mbps"`
	IOPS              float64       `json:"iops"`
	ErrorRate         float64       `json:"error_rate"`
}

// HealthIssue represents a storage health issue
type HealthIssue struct {
	Severity    string    `json:"severity"` // "info", "warning", "critical"
	Type        string    `json:"type"`     // "connectivity", "performance", "quota", "corruption"
	Provider    string    `json:"provider"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Resolution  string    `json:"resolution"`
	DetectedAt  time.Time `json:"detected_at"`
}

// StorageTrendReport provides storage usage trends over time
type StorageTrendReport struct {
	Period            time.Duration             `json:"period"`
	TotalGrowth       int64                     `json:"total_growth"`
	GrowthRate        float64                   `json:"growth_rate"` // Bytes per day
	DatabaseTrends    map[string]*DatabaseTrend `json:"database_trends"`
	BackupFrequency   *BackupFrequencyTrend     `json:"backup_frequency"`
	CompressionTrends *CompressionTrend         `json:"compression_trends"`
	PredictedUsage    *UsagePrediction          `json:"predicted_usage"`
	GeneratedAt       time.Time                 `json:"generated_at"`
}

// DatabaseTrend represents storage trend for a database
type DatabaseTrend struct {
	DatabaseName string  `json:"database_name"`
	Growth       int64   `json:"growth"`
	GrowthRate   float64 `json:"growth_rate"`
	BackupCount  int     `json:"backup_count"`
	AverageSize  int64   `json:"average_size"`
	Trend        string  `json:"trend"` // "increasing", "stable", "decreasing"
}

// BackupFrequencyTrend represents backup frequency trends
type BackupFrequencyTrend struct {
	DailyAverage   float64 `json:"daily_average"`
	WeeklyAverage  float64 `json:"weekly_average"`
	MonthlyAverage float64 `json:"monthly_average"`
	Trend          string  `json:"trend"` // "increasing", "stable", "decreasing"
}

// CompressionTrend represents compression effectiveness trends
type CompressionTrend struct {
	AverageRatio   float64        `json:"average_ratio"`
	RatioTrend     string         `json:"ratio_trend"` // "improving", "stable", "degrading"
	AlgorithmUsage map[string]int `json:"algorithm_usage"`
}

// UsagePrediction provides storage usage predictions
type UsagePrediction struct {
	PredictedUsageIn30Days int64         `json:"predicted_usage_in_30_days"`
	PredictedUsageIn90Days int64         `json:"predicted_usage_in_90_days"`
	PredictedUsageIn1Year  int64         `json:"predicted_usage_in_1_year"`
	EstimatedTimeToQuota   time.Duration `json:"estimated_time_to_quota"`
	Confidence             float64       `json:"confidence"` // 0.0 to 1.0
}

// StorageAlert represents a storage-related alert
type StorageAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`     // "quota", "health", "performance", "growth"
	Severity    string                 `json:"severity"` // "info", "warning", "critical"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	Actions     []string               `json:"actions"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// storageMonitor implements the StorageMonitor interface
type storageMonitor struct {
	backupManager BackupManager
	config        *BackupSystemConfig
	logger        *logging.Logger
}

// NewStorageMonitor creates a new storage monitor
func NewStorageMonitor(backupManager BackupManager, config *BackupSystemConfig, logger *logging.Logger) StorageMonitor {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &storageMonitor{
		backupManager: backupManager,
		config:        config,
		logger:        logger,
	}
}

// GetStorageUsage returns current storage usage statistics
func (sm *storageMonitor) GetStorageUsage(ctx context.Context) (*StorageUsageReport, error) {
	sm.logger.Debug("Generating storage usage report")

	// Get all backups
	backups, err := sm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups for storage usage: %w", err)
	}

	if len(backups) == 0 {
		return &StorageUsageReport{
			TotalBackups:        0,
			TotalSize:           0,
			TotalCompressedSize: 0,
			CompressionRatio:    0,
			AverageBackupSize:   0,
			StorageByProvider:   make(map[string]*ProviderUsage),
			StorageByDatabase:   make(map[string]*DatabaseUsage),
			StorageByAge:        make(map[string]*AgeGroupUsage),
			GeneratedAt:         time.Now(),
		}, nil
	}

	report := &StorageUsageReport{
		TotalBackups:      len(backups),
		StorageByProvider: make(map[string]*ProviderUsage),
		StorageByDatabase: make(map[string]*DatabaseUsage),
		StorageByAge:      make(map[string]*AgeGroupUsage),
		GeneratedAt:       time.Now(),
	}

	var totalSize, totalCompressedSize int64
	var largestBackup, smallestBackup *BackupMetadata

	now := time.Now()

	// Process each backup
	for _, backup := range backups {
		totalSize += backup.Size
		totalCompressedSize += backup.CompressedSize

		// Track largest and smallest backups
		if largestBackup == nil || backup.Size > largestBackup.Size {
			largestBackup = backup
		}
		if smallestBackup == nil || backup.Size < smallestBackup.Size {
			smallestBackup = backup
		}

		// Update provider usage
		providerKey := string(sm.config.Storage.Provider)
		if report.StorageByProvider[providerKey] == nil {
			report.StorageByProvider[providerKey] = &ProviderUsage{
				Provider: sm.config.Storage.Provider,
			}
		}
		providerUsage := report.StorageByProvider[providerKey]
		providerUsage.BackupCount++
		providerUsage.TotalSize += backup.Size
		providerUsage.CompressedSize += backup.CompressedSize

		// Update database usage
		if report.StorageByDatabase[backup.DatabaseName] == nil {
			report.StorageByDatabase[backup.DatabaseName] = &DatabaseUsage{
				DatabaseName: backup.DatabaseName,
			}
		}
		dbUsage := report.StorageByDatabase[backup.DatabaseName]
		dbUsage.BackupCount++
		dbUsage.TotalSize += backup.Size
		dbUsage.CompressedSize += backup.CompressedSize

		// Update age group usage
		age := now.Sub(backup.CreatedAt)
		var ageGroup string
		switch {
		case age <= 24*time.Hour:
			ageGroup = "daily"
		case age <= 7*24*time.Hour:
			ageGroup = "weekly"
		case age <= 30*24*time.Hour:
			ageGroup = "monthly"
		default:
			ageGroup = "older"
		}

		if report.StorageByAge[ageGroup] == nil {
			report.StorageByAge[ageGroup] = &AgeGroupUsage{
				AgeGroup: ageGroup,
			}
		}
		ageUsage := report.StorageByAge[ageGroup]
		ageUsage.BackupCount++
		ageUsage.TotalSize += backup.Size
		ageUsage.CompressedSize += backup.CompressedSize
	}

	// Calculate final statistics
	report.TotalSize = totalSize
	report.TotalCompressedSize = totalCompressedSize
	report.LargestBackup = largestBackup
	report.SmallestBackup = smallestBackup

	if len(backups) > 0 {
		report.AverageBackupSize = totalSize / int64(len(backups))
	}

	if totalSize > 0 {
		report.CompressionRatio = float64(totalCompressedSize) / float64(totalSize)
	}

	// Calculate compression ratios for sub-categories
	for _, providerUsage := range report.StorageByProvider {
		if providerUsage.TotalSize > 0 {
			providerUsage.CompressionRatio = float64(providerUsage.CompressedSize) / float64(providerUsage.TotalSize)
		}
	}

	for _, dbUsage := range report.StorageByDatabase {
		if dbUsage.TotalSize > 0 {
			dbUsage.CompressionRatio = float64(dbUsage.CompressedSize) / float64(dbUsage.TotalSize)
		}
	}

	for _, ageUsage := range report.StorageByAge {
		if ageUsage.TotalSize > 0 {
			ageUsage.CompressionRatio = float64(ageUsage.CompressedSize) / float64(ageUsage.TotalSize)
		}
	}

	return report, nil
}

// GetStorageUsageByDatabase returns storage usage grouped by database
func (sm *storageMonitor) GetStorageUsageByDatabase(ctx context.Context) (map[string]*DatabaseStorageUsage, error) {
	sm.logger.Debug("Getting storage usage by database")

	// Get all backups
	backups, err := sm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	result := make(map[string]*DatabaseStorageUsage)

	// Group backups by database
	backupsByDatabase := make(map[string][]*BackupMetadata)
	for _, backup := range backups {
		backupsByDatabase[backup.DatabaseName] = append(backupsByDatabase[backup.DatabaseName], backup)
	}

	// Process each database
	for databaseName, dbBackups := range backupsByDatabase {
		usage := &DatabaseStorageUsage{
			DatabaseName:    databaseName,
			BackupCount:     len(dbBackups),
			BackupsByStatus: make(map[string]int),
			BackupsByAge:    make(map[string]int),
		}

		var totalSize, totalCompressedSize int64
		var largestBackup, smallestBackup *BackupMetadata
		var oldestTime, newestTime time.Time

		now := time.Now()

		for i, backup := range dbBackups {
			totalSize += backup.Size
			totalCompressedSize += backup.CompressedSize

			// Track largest and smallest backups
			if largestBackup == nil || backup.Size > largestBackup.Size {
				largestBackup = backup
			}
			if smallestBackup == nil || backup.Size < smallestBackup.Size {
				smallestBackup = backup
			}

			// Track oldest and newest backups
			if i == 0 || backup.CreatedAt.Before(oldestTime) {
				oldestTime = backup.CreatedAt
			}
			if i == 0 || backup.CreatedAt.After(newestTime) {
				newestTime = backup.CreatedAt
			}

			// Count by status
			usage.BackupsByStatus[string(backup.Status)]++

			// Count by age
			age := now.Sub(backup.CreatedAt)
			var ageGroup string
			switch {
			case age <= 24*time.Hour:
				ageGroup = "daily"
			case age <= 7*24*time.Hour:
				ageGroup = "weekly"
			case age <= 30*24*time.Hour:
				ageGroup = "monthly"
			default:
				ageGroup = "older"
			}
			usage.BackupsByAge[ageGroup]++
		}

		usage.TotalSize = totalSize
		usage.CompressedSize = totalCompressedSize
		usage.LargestBackup = largestBackup
		usage.SmallestBackup = smallestBackup

		if len(dbBackups) > 0 {
			usage.AverageBackupSize = totalSize / int64(len(dbBackups))
		}

		if totalSize > 0 {
			usage.CompressionRatio = float64(totalCompressedSize) / float64(totalSize)
		}

		// Calculate storage growth (simplified)
		if !oldestTime.IsZero() && !newestTime.IsZero() && newestTime.After(oldestTime) {
			duration := newestTime.Sub(oldestTime)
			if duration > 0 {
				dailyGrowth := totalSize / int64(duration.Hours()/24)
				usage.StorageGrowth = &StorageGrowthInfo{
					DailyGrowth:   dailyGrowth,
					WeeklyGrowth:  dailyGrowth * 7,
					MonthlyGrowth: dailyGrowth * 30,
					GrowthTrend:   "stable", // Simplified
					LastUpdated:   time.Now(),
				}
			}
		}

		result[databaseName] = usage
	}

	return result, nil
}

// CheckStorageQuotas checks if storage usage exceeds configured quotas
func (sm *storageMonitor) CheckStorageQuotas(ctx context.Context) (*StorageQuotaStatus, error) {
	sm.logger.Debug("Checking storage quotas")

	// Get storage usage
	usage, err := sm.GetStorageUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage usage for quota check: %w", err)
	}

	status := &StorageQuotaStatus{
		QuotaEnabled:   sm.isQuotaEnabled(),
		TotalQuota:     sm.getTotalQuota(),
		UsedStorage:    usage.TotalSize,
		DatabaseQuotas: make(map[string]*DatabaseQuota),
		ProviderQuotas: make(map[string]*ProviderQuota),
		QuotaWarnings:  []*QuotaWarning{},
	}

	// Calculate available storage and usage percentage
	if status.TotalQuota > 0 {
		status.AvailableStorage = status.TotalQuota - status.UsedStorage
		if status.AvailableStorage < 0 {
			status.AvailableStorage = 0
		}
		status.UsagePercentage = float64(status.UsedStorage) / float64(status.TotalQuota) * 100
		status.QuotaExceeded = status.UsedStorage > status.TotalQuota
	}

	// Check database quotas
	for databaseName, dbUsage := range usage.StorageByDatabase {
		quota := sm.getDatabaseQuota(databaseName)
		if quota > 0 {
			dbQuota := &DatabaseQuota{
				DatabaseName:    databaseName,
				Quota:           quota,
				Used:            dbUsage.TotalSize,
				Available:       quota - dbUsage.TotalSize,
				UsagePercentage: float64(dbUsage.TotalSize) / float64(quota) * 100,
				QuotaExceeded:   dbUsage.TotalSize > quota,
			}
			if dbQuota.Available < 0 {
				dbQuota.Available = 0
			}
			status.DatabaseQuotas[databaseName] = dbQuota

			// Generate warnings for database quotas
			if dbQuota.UsagePercentage >= 90 {
				status.QuotaWarnings = append(status.QuotaWarnings, &QuotaWarning{
					Type:              "database",
					Target:            databaseName,
					CurrentUsage:      dbQuota.Used,
					Quota:             dbQuota.Quota,
					UsagePercentage:   dbQuota.UsagePercentage,
					Severity:          sm.getWarningSeverity(dbQuota.UsagePercentage),
					Message:           fmt.Sprintf("Database %s is using %.1f%% of its quota", databaseName, dbQuota.UsagePercentage),
					RecommendedAction: sm.getQuotaRecommendation(dbQuota.UsagePercentage),
				})
			}
		}
	}

	// Check provider quotas
	for providerName, providerUsage := range usage.StorageByProvider {
		quota := sm.getProviderQuota(providerName)
		if quota > 0 {
			providerQuota := &ProviderQuota{
				Provider:        providerUsage.Provider,
				Quota:           quota,
				Used:            providerUsage.TotalSize,
				Available:       quota - providerUsage.TotalSize,
				UsagePercentage: float64(providerUsage.TotalSize) / float64(quota) * 100,
				QuotaExceeded:   providerUsage.TotalSize > quota,
			}
			if providerQuota.Available < 0 {
				providerQuota.Available = 0
			}
			status.ProviderQuotas[providerName] = providerQuota

			// Generate warnings for provider quotas
			if providerQuota.UsagePercentage >= 85 {
				status.QuotaWarnings = append(status.QuotaWarnings, &QuotaWarning{
					Type:              "provider",
					Target:            providerName,
					CurrentUsage:      providerQuota.Used,
					Quota:             providerQuota.Quota,
					UsagePercentage:   providerQuota.UsagePercentage,
					Severity:          sm.getWarningSeverity(providerQuota.UsagePercentage),
					Message:           fmt.Sprintf("Storage provider %s is using %.1f%% of its quota", providerName, providerQuota.UsagePercentage),
					RecommendedAction: sm.getQuotaRecommendation(providerQuota.UsagePercentage),
				})
			}
		}
	}

	// Generate total quota warnings
	if status.QuotaEnabled && status.UsagePercentage >= 80 {
		status.QuotaWarnings = append(status.QuotaWarnings, &QuotaWarning{
			Type:              "total",
			Target:            "system",
			CurrentUsage:      status.UsedStorage,
			Quota:             status.TotalQuota,
			UsagePercentage:   status.UsagePercentage,
			Severity:          sm.getWarningSeverity(status.UsagePercentage),
			Message:           fmt.Sprintf("Total storage usage is %.1f%% of quota", status.UsagePercentage),
			RecommendedAction: sm.getQuotaRecommendation(status.UsagePercentage),
		})
	}

	// Estimate time to full based on growth trends
	if status.QuotaEnabled && status.AvailableStorage > 0 {
		trends, err := sm.GetStorageTrends(ctx, 30*24*time.Hour) // 30 days
		if err == nil && trends.GrowthRate > 0 {
			daysToFull := float64(status.AvailableStorage) / trends.GrowthRate
			status.EstimatedTimeToFull = time.Duration(daysToFull * 24 * float64(time.Hour))
		}
	}

	return status, nil
}

// GetStorageOptimizationRecommendations provides recommendations for storage optimization
func (sm *storageMonitor) GetStorageOptimizationRecommendations(ctx context.Context) (*StorageOptimizationReport, error) {
	sm.logger.Debug("Generating storage optimization recommendations")

	// Get all backups for analysis
	backups, err := sm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups for optimization analysis: %w", err)
	}

	report := &StorageOptimizationReport{
		Recommendations:     []*OptimizationRecommendation{},
		CompressionAnalysis: sm.analyzeCompression(backups),
		RetentionAnalysis:   sm.analyzeRetention(ctx, backups),
		DuplicationAnalysis: sm.analyzeDuplication(backups),
		GeneratedAt:         time.Now(),
	}

	// Generate recommendations based on analysis
	report.Recommendations = sm.generateOptimizationRecommendations(report)

	// Calculate total potential savings
	for _, rec := range report.Recommendations {
		report.TotalPotentialSavings += rec.EstimatedSavings
	}

	return report, nil
}

// MonitorStorageHealth performs health checks on storage providers
func (sm *storageMonitor) MonitorStorageHealth(ctx context.Context) (*StorageHealthReport, error) {
	sm.logger.Debug("Monitoring storage health")

	report := &StorageHealthReport{
		OverallHealth:      "healthy",
		ProviderHealth:     make(map[string]*ProviderHealth),
		ConnectivityTests:  []*ConnectivityTest{},
		PerformanceMetrics: &StoragePerformanceMetrics{},
		HealthIssues:       []*HealthIssue{},
		GeneratedAt:        time.Now(),
	}

	// TODO: Implement actual health monitoring
	// This would involve testing connectivity, performance, and error rates

	return report, nil
}

// GetStorageTrends analyzes storage usage trends over time
func (sm *storageMonitor) GetStorageTrends(ctx context.Context, period time.Duration) (*StorageTrendReport, error) {
	sm.logger.Debug(fmt.Sprintf("Analyzing storage trends over %v", period))

	// Get backups within the specified period
	cutoffTime := time.Now().Add(-period)
	filter := BackupFilter{
		CreatedAfter: &cutoffTime,
	}

	backups, err := sm.backupManager.ListBackups(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups for trend analysis: %w", err)
	}

	// Also get all backups for comparison
	allBackups, err := sm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all backups for trend analysis: %w", err)
	}

	report := &StorageTrendReport{
		Period:         period,
		DatabaseTrends: make(map[string]*DatabaseTrend),
		GeneratedAt:    time.Now(),
	}

	// Calculate overall growth
	if len(backups) > 0 {
		var totalSize int64
		for _, backup := range backups {
			totalSize += backup.Size
		}
		report.TotalGrowth = totalSize

		// Calculate growth rate (bytes per day)
		days := period.Hours() / 24
		if days > 0 {
			report.GrowthRate = float64(totalSize) / days
		}
	}

	// Analyze database trends
	databaseBackups := make(map[string][]*BackupMetadata)
	for _, backup := range backups {
		databaseBackups[backup.DatabaseName] = append(databaseBackups[backup.DatabaseName], backup)
	}

	for databaseName, dbBackups := range databaseBackups {
		trend := sm.analyzeDatabaseTrend(dbBackups, period)
		report.DatabaseTrends[databaseName] = trend
	}

	// Analyze backup frequency trends
	report.BackupFrequency = sm.analyzeBackupFrequency(backups, period)

	// Analyze compression trends
	report.CompressionTrends = sm.analyzeCompressionTrends(backups)

	// Generate usage predictions
	report.PredictedUsage = sm.generateUsagePredictions(allBackups, report.GrowthRate)

	return report, nil
}

// GenerateStorageAlerts generates alerts based on storage conditions
func (sm *storageMonitor) GenerateStorageAlerts(ctx context.Context) ([]*StorageAlert, error) {
	sm.logger.Debug("Generating storage alerts")

	var alerts []*StorageAlert

	// Check storage usage
	usage, err := sm.GetStorageUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage usage for alerts: %w", err)
	}

	// Generate alerts based on usage patterns
	if usage.TotalSize > 10*1024*1024*1024 { // 10GB threshold
		alerts = append(alerts, &StorageAlert{
			ID:          fmt.Sprintf("storage-usage-%d", time.Now().Unix()),
			Type:        "quota",
			Severity:    "warning",
			Title:       "High Storage Usage",
			Description: fmt.Sprintf("Total backup storage usage is %d bytes", usage.TotalSize),
			Details: map[string]interface{}{
				"total_size":   usage.TotalSize,
				"backup_count": usage.TotalBackups,
			},
			Actions:   []string{"Review retention policies", "Enable compression", "Clean up old backups"},
			CreatedAt: time.Now(),
		})
	}

	// Check for poor compression ratios
	if usage.CompressionRatio > 0.8 { // Poor compression
		alerts = append(alerts, &StorageAlert{
			ID:          fmt.Sprintf("compression-%d", time.Now().Unix()),
			Type:        "performance",
			Severity:    "info",
			Title:       "Poor Compression Efficiency",
			Description: fmt.Sprintf("Compression ratio is %.2f, consider reviewing compression settings", usage.CompressionRatio),
			Details: map[string]interface{}{
				"compression_ratio": usage.CompressionRatio,
				"total_size":        usage.TotalSize,
				"compressed_size":   usage.TotalCompressedSize,
			},
			Actions:   []string{"Review compression algorithm", "Check backup content types"},
			CreatedAt: time.Now(),
		})
	}

	return alerts, nil
}

// MonitorStorageHealthWithDetails performs comprehensive health checks
func (sm *storageMonitor) MonitorStorageHealthWithDetails(ctx context.Context) (*StorageHealthReport, error) {
	sm.logger.Debug("Performing comprehensive storage health monitoring")

	report := &StorageHealthReport{
		OverallHealth:      "healthy",
		ProviderHealth:     make(map[string]*ProviderHealth),
		ConnectivityTests:  []*ConnectivityTest{},
		PerformanceMetrics: &StoragePerformanceMetrics{},
		HealthIssues:       []*HealthIssue{},
		GeneratedAt:        time.Now(),
	}

	// Test storage provider connectivity and performance
	providerHealth, err := sm.testStorageProviderHealth(ctx)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to test storage provider health: %v", err))
		report.HealthIssues = append(report.HealthIssues, &HealthIssue{
			Severity:    "critical",
			Type:        "connectivity",
			Provider:    string(sm.config.Storage.Provider),
			Description: fmt.Sprintf("Failed to test storage provider health: %v", err),
			Impact:      "Backup operations may fail",
			Resolution:  "Check storage provider configuration and connectivity",
			DetectedAt:  time.Now(),
		})
		report.OverallHealth = "critical"
	} else {
		report.ProviderHealth[string(sm.config.Storage.Provider)] = providerHealth
		if providerHealth.Status != "healthy" {
			report.OverallHealth = providerHealth.Status
		}
	}

	// Check storage quotas for health issues
	quotaStatus, err := sm.CheckStorageQuotas(ctx)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to check storage quotas: %v", err))
	} else {
		// Add quota-related health issues
		for _, warning := range quotaStatus.QuotaWarnings {
			severity := "warning"
			if warning.Severity == "critical" {
				severity = "critical"
				if report.OverallHealth == "healthy" {
					report.OverallHealth = "critical"
				}
			} else if report.OverallHealth == "healthy" {
				report.OverallHealth = "warning"
			}

			report.HealthIssues = append(report.HealthIssues, &HealthIssue{
				Severity:    severity,
				Type:        "quota",
				Provider:    string(sm.config.Storage.Provider),
				Description: warning.Message,
				Impact:      "Storage operations may be limited or fail",
				Resolution:  warning.RecommendedAction,
				DetectedAt:  time.Now(),
			})
		}
	}

	// Check for storage optimization issues
	optimization, err := sm.GetStorageOptimizationRecommendations(ctx)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to get optimization recommendations: %v", err))
	} else {
		// Add high-priority optimization issues as health concerns
		for _, rec := range optimization.Recommendations {
			if rec.Priority == "high" {
				report.HealthIssues = append(report.HealthIssues, &HealthIssue{
					Severity:    "info",
					Type:        "performance",
					Provider:    string(sm.config.Storage.Provider),
					Description: rec.Description,
					Impact:      rec.Impact,
					Resolution:  rec.ActionRequired,
					DetectedAt:  time.Now(),
				})
			}
		}
	}

	return report, nil
}

// GetStorageHealthSummary provides a quick health summary
func (sm *storageMonitor) GetStorageHealthSummary(ctx context.Context) (*StorageHealthSummary, error) {
	sm.logger.Debug("Generating storage health summary")

	// Get basic health report
	health, err := sm.MonitorStorageHealthWithDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get health details: %w", err)
	}

	// Get storage usage
	usage, err := sm.GetStorageUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage usage: %w", err)
	}

	// Get quota status
	quotas, err := sm.CheckStorageQuotas(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check quotas: %w", err)
	}

	summary := &StorageHealthSummary{
		OverallStatus:      health.OverallHealth,
		TotalBackups:       usage.TotalBackups,
		TotalStorageUsed:   usage.TotalSize,
		CompressionRatio:   usage.CompressionRatio,
		QuotaUsagePercent:  quotas.UsagePercentage,
		CriticalIssues:     sm.countIssuesBySeverity(health.HealthIssues, "critical"),
		WarningIssues:      sm.countIssuesBySeverity(health.HealthIssues, "warning"),
		LastHealthCheck:    time.Now(),
		RecommendedActions: sm.getTopRecommendations(health.HealthIssues),
	}

	return summary, nil
}

// Helper methods for quota and health management

func (sm *storageMonitor) isQuotaEnabled() bool {
	// This would be configured in the backup system config
	// For now, return false as quotas are not implemented in config
	return false
}

func (sm *storageMonitor) getTotalQuota() int64 {
	// This would be configured in the backup system config
	// For now, return 0 as quotas are not implemented in config
	return 0
}

func (sm *storageMonitor) getDatabaseQuota(databaseName string) int64 {
	// This would be configured per database
	// For now, return 0 as quotas are not implemented in config
	return 0
}

func (sm *storageMonitor) getProviderQuota(providerName string) int64 {
	// This would be configured per provider
	// For now, return 0 as quotas are not implemented in config
	return 0
}

func (sm *storageMonitor) getWarningSeverity(usagePercentage float64) string {
	if usagePercentage >= 95 {
		return "critical"
	} else if usagePercentage >= 85 {
		return "warning"
	}
	return "info"
}

func (sm *storageMonitor) getQuotaRecommendation(usagePercentage float64) string {
	if usagePercentage >= 95 {
		return "Immediate action required: Clean up old backups or increase quota"
	} else if usagePercentage >= 90 {
		return "Review retention policies and clean up old backups"
	} else if usagePercentage >= 85 {
		return "Monitor usage closely and consider cleanup"
	}
	return "Monitor usage trends"
}

func (sm *storageMonitor) testStorageProviderHealth(ctx context.Context) (*ProviderHealth, error) {
	providerHealth := &ProviderHealth{
		Provider:         sm.config.Storage.Provider,
		Status:           "healthy",
		LastSuccessfulOp: time.Now(),
		ErrorRate:        0.0,
		Issues:           []string{},
	}

	startTime := time.Now()

	// Test basic connectivity by listing backups
	_, err := sm.backupManager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		providerHealth.Status = "critical"
		providerHealth.LastFailedOp = time.Now()
		providerHealth.ErrorRate = 1.0
		providerHealth.Issues = append(providerHealth.Issues, fmt.Sprintf("Failed to list backups: %v", err))
		return providerHealth, nil
	}

	providerHealth.ResponseTime = time.Since(startTime)

	// Additional health checks could be added here
	// For example, testing write/read operations with test data

	return providerHealth, nil
}

func (sm *storageMonitor) countIssuesBySeverity(issues []*HealthIssue, severity string) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == severity {
			count++
		}
	}
	return count
}

func (sm *storageMonitor) getTopRecommendations(issues []*HealthIssue) []string {
	recommendations := make(map[string]bool)

	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "warning" {
			recommendations[issue.Resolution] = true
		}
	}

	result := make([]string, 0, len(recommendations))
	for rec := range recommendations {
		result = append(result, rec)
	}

	// Limit to top 5 recommendations
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// Helper methods for trend analysis

func (sm *storageMonitor) analyzeDatabaseTrend(backups []*BackupMetadata, period time.Duration) *DatabaseTrend {
	if len(backups) == 0 {
		return &DatabaseTrend{
			DatabaseName: "",
			Growth:       0,
			GrowthRate:   0,
			BackupCount:  0,
			AverageSize:  0,
			Trend:        "stable",
		}
	}

	var totalSize int64
	for _, backup := range backups {
		totalSize += backup.Size
	}

	trend := &DatabaseTrend{
		DatabaseName: backups[0].DatabaseName,
		Growth:       totalSize,
		BackupCount:  len(backups),
		AverageSize:  totalSize / int64(len(backups)),
	}

	// Calculate growth rate
	days := period.Hours() / 24
	if days > 0 {
		trend.GrowthRate = float64(totalSize) / days
	}

	// Determine trend direction
	if len(backups) >= 2 {
		// Sort by creation time
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.Before(backups[j].CreatedAt)
		})

		// Compare first half with second half
		midpoint := len(backups) / 2
		var firstHalfSize, secondHalfSize int64

		for i := 0; i < midpoint; i++ {
			firstHalfSize += backups[i].Size
		}
		for i := midpoint; i < len(backups); i++ {
			secondHalfSize += backups[i].Size
		}

		if float64(secondHalfSize) > float64(firstHalfSize)*1.1 { // 10% increase threshold
			trend.Trend = "increasing"
		} else if float64(secondHalfSize) < float64(firstHalfSize)*0.9 { // 10% decrease threshold
			trend.Trend = "decreasing"
		} else {
			trend.Trend = "stable"
		}
	}

	return trend
}

func (sm *storageMonitor) analyzeBackupFrequency(backups []*BackupMetadata, period time.Duration) *BackupFrequencyTrend {
	if len(backups) == 0 {
		return &BackupFrequencyTrend{
			DailyAverage:   0,
			WeeklyAverage:  0,
			MonthlyAverage: 0,
			Trend:          "stable",
		}
	}

	days := period.Hours() / 24
	weeks := days / 7
	months := days / 30

	frequency := &BackupFrequencyTrend{
		DailyAverage:   float64(len(backups)) / days,
		WeeklyAverage:  float64(len(backups)) / weeks,
		MonthlyAverage: float64(len(backups)) / months,
		Trend:          "stable",
	}

	// Analyze frequency trend by comparing periods
	if days >= 14 { // Need at least 2 weeks of data
		midpoint := time.Now().Add(-period / 2)

		var firstHalfCount, secondHalfCount int
		for _, backup := range backups {
			if backup.CreatedAt.Before(midpoint) {
				firstHalfCount++
			} else {
				secondHalfCount++
			}
		}

		if float64(secondHalfCount) > float64(firstHalfCount)*1.2 { // 20% increase threshold
			frequency.Trend = "increasing"
		} else if float64(secondHalfCount) < float64(firstHalfCount)*0.8 { // 20% decrease threshold
			frequency.Trend = "decreasing"
		}
	}

	return frequency
}

func (sm *storageMonitor) analyzeCompressionTrends(backups []*BackupMetadata) *CompressionTrend {
	if len(backups) == 0 {
		return &CompressionTrend{
			AverageRatio:   0,
			RatioTrend:     "stable",
			AlgorithmUsage: make(map[string]int),
		}
	}

	var totalOriginal, totalCompressed int64
	algorithmUsage := make(map[string]int)

	for _, backup := range backups {
		totalOriginal += backup.Size
		totalCompressed += backup.CompressedSize
		algorithmUsage[string(backup.CompressionType)]++
	}

	trend := &CompressionTrend{
		AlgorithmUsage: algorithmUsage,
		RatioTrend:     "stable",
	}

	if totalOriginal > 0 {
		trend.AverageRatio = float64(totalCompressed) / float64(totalOriginal)
	}

	// Analyze compression ratio trend
	if len(backups) >= 4 {
		// Sort by creation time
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.Before(backups[j].CreatedAt)
		})

		// Compare first quarter with last quarter
		quarterSize := len(backups) / 4

		var firstQuarterOriginal, firstQuarterCompressed int64
		var lastQuarterOriginal, lastQuarterCompressed int64

		for i := 0; i < quarterSize; i++ {
			firstQuarterOriginal += backups[i].Size
			firstQuarterCompressed += backups[i].CompressedSize
		}

		for i := len(backups) - quarterSize; i < len(backups); i++ {
			lastQuarterOriginal += backups[i].Size
			lastQuarterCompressed += backups[i].CompressedSize
		}

		var firstRatio, lastRatio float64
		if firstQuarterOriginal > 0 {
			firstRatio = float64(firstQuarterCompressed) / float64(firstQuarterOriginal)
		}
		if lastQuarterOriginal > 0 {
			lastRatio = float64(lastQuarterCompressed) / float64(lastQuarterOriginal)
		}

		if lastRatio < firstRatio*0.95 { // 5% improvement threshold
			trend.RatioTrend = "improving"
		} else if lastRatio > firstRatio*1.05 { // 5% degradation threshold
			trend.RatioTrend = "degrading"
		}
	}

	return trend
}

func (sm *storageMonitor) generateUsagePredictions(allBackups []*BackupMetadata, growthRate float64) *UsagePrediction {
	prediction := &UsagePrediction{
		Confidence: 0.5, // Default confidence
	}

	if len(allBackups) == 0 || growthRate <= 0 {
		return prediction
	}

	// Calculate current total usage
	var currentUsage int64
	for _, backup := range allBackups {
		currentUsage += backup.Size
	}

	// Predict future usage based on growth rate
	prediction.PredictedUsageIn30Days = currentUsage + int64(growthRate*30)
	prediction.PredictedUsageIn90Days = currentUsage + int64(growthRate*90)
	prediction.PredictedUsageIn1Year = currentUsage + int64(growthRate*365)

	// Estimate time to quota (if quota is enabled)
	if sm.isQuotaEnabled() {
		totalQuota := sm.getTotalQuota()
		if totalQuota > currentUsage && growthRate > 0 {
			daysToQuota := float64(totalQuota-currentUsage) / growthRate
			prediction.EstimatedTimeToQuota = time.Duration(daysToQuota * 24 * float64(time.Hour))
		}
	}

	// Adjust confidence based on data quality
	if len(allBackups) >= 10 {
		prediction.Confidence = 0.8
	} else if len(allBackups) >= 5 {
		prediction.Confidence = 0.6
	}

	return prediction
}

// Additional types for enhanced monitoring

// StorageHealthSummary provides a quick overview of storage health
type StorageHealthSummary struct {
	OverallStatus      string    `json:"overall_status"`
	TotalBackups       int       `json:"total_backups"`
	TotalStorageUsed   int64     `json:"total_storage_used"`
	CompressionRatio   float64   `json:"compression_ratio"`
	QuotaUsagePercent  float64   `json:"quota_usage_percent"`
	CriticalIssues     int       `json:"critical_issues"`
	WarningIssues      int       `json:"warning_issues"`
	LastHealthCheck    time.Time `json:"last_health_check"`
	RecommendedActions []string  `json:"recommended_actions"`
}

// Helper methods for analysis

func (sm *storageMonitor) analyzeCompression(backups []*BackupMetadata) *CompressionAnalysis {
	analysis := &CompressionAnalysis{
		CompressionByAlgorithm:  make(map[string]*CompressionAnalysisStats),
		UncompressedBackups:     []*BackupMetadata{},
		PoorlyCompressedBackups: []*BackupMetadata{},
	}

	var totalOriginal, totalCompressed int64

	for _, backup := range backups {
		totalOriginal += backup.Size
		totalCompressed += backup.CompressedSize

		// Check for uncompressed backups
		if backup.CompressionType == CompressionTypeNone {
			analysis.UncompressedBackups = append(analysis.UncompressedBackups, backup)
		}

		// Check for poorly compressed backups (ratio > 0.9)
		if backup.Size > 0 {
			ratio := float64(backup.CompressedSize) / float64(backup.Size)
			if ratio > 0.9 {
				analysis.PoorlyCompressedBackups = append(analysis.PoorlyCompressedBackups, backup)
			}
		}

		// Update algorithm stats
		algoKey := string(backup.CompressionType)
		if analysis.CompressionByAlgorithm[algoKey] == nil {
			analysis.CompressionByAlgorithm[algoKey] = &CompressionAnalysisStats{
				Algorithm: backup.CompressionType,
			}
		}
		stats := analysis.CompressionByAlgorithm[algoKey]
		stats.BackupCount++
		stats.OriginalSize += backup.Size
		stats.CompressedSize += backup.CompressedSize
	}

	// Calculate overall compression ratio
	if totalOriginal > 0 {
		analysis.OverallCompressionRatio = float64(totalCompressed) / float64(totalOriginal)
	}

	// Calculate algorithm-specific ratios
	for _, stats := range analysis.CompressionByAlgorithm {
		if stats.OriginalSize > 0 {
			stats.CompressionRatio = float64(stats.CompressedSize) / float64(stats.OriginalSize)
			stats.AverageRatio = stats.CompressionRatio
		}
	}

	// Calculate potential savings from uncompressed backups
	for _, backup := range analysis.UncompressedBackups {
		// Estimate 30% compression savings
		analysis.PotentialCompressionSavings += int64(float64(backup.Size) * 0.3)
	}

	// Recommend best compression algorithm
	analysis.RecommendedAlgorithm = CompressionTypeGzip // Default recommendation

	return analysis
}

func (sm *storageMonitor) analyzeRetention(ctx context.Context, backups []*BackupMetadata) *RetentionAnalysis {
	analysis := &RetentionAnalysis{
		BackupsEligibleForCleanup:   []*BackupMetadata{},
		RecommendedRetentionChanges: []*RetentionRecommendation{},
	}

	// Simple analysis - mark backups older than 90 days as eligible for cleanup
	cutoffTime := time.Now().Add(-90 * 24 * time.Hour)

	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoffTime) {
			analysis.BackupsEligibleForCleanup = append(analysis.BackupsEligibleForCleanup, backup)
			analysis.PotentialRetentionSavings += backup.Size
		}
	}

	// Calculate effectiveness (simplified)
	if len(backups) > 0 {
		analysis.RetentionPolicyEffectiveness = 1.0 - float64(len(analysis.BackupsEligibleForCleanup))/float64(len(backups))
	}

	return analysis
}

func (sm *storageMonitor) analyzeDuplication(backups []*BackupMetadata) *DuplicationAnalysis {
	analysis := &DuplicationAnalysis{
		DuplicateBackups:             []*BackupDuplication{},
		DeduplicationRecommendations: []string{},
	}

	// Group backups by checksum to find duplicates
	checksumGroups := make(map[string][]*BackupMetadata)
	for _, backup := range backups {
		if backup.Checksum != "" {
			checksumGroups[backup.Checksum] = append(checksumGroups[backup.Checksum], backup)
		}
	}

	// Find duplicate groups
	for checksum, group := range checksumGroups {
		if len(group) > 1 {
			var totalSize int64
			for _, backup := range group {
				totalSize += backup.Size
			}

			// Potential savings = total size - size of one backup
			potentialSavings := totalSize - group[0].Size

			analysis.DuplicateBackups = append(analysis.DuplicateBackups, &BackupDuplication{
				Checksum:         checksum,
				BackupCount:      len(group),
				Backups:          group,
				TotalSize:        totalSize,
				PotentialSavings: potentialSavings,
			})

			analysis.PotentialDeduplicationSavings += potentialSavings
		}
	}

	if len(analysis.DuplicateBackups) > 0 {
		analysis.DeduplicationRecommendations = append(analysis.DeduplicationRecommendations,
			"Consider implementing backup deduplication",
			"Review backup creation processes to avoid duplicates",
			"Implement content-based deduplication at storage level")
	}

	return analysis
}

func (sm *storageMonitor) generateOptimizationRecommendations(report *StorageOptimizationReport) []*OptimizationRecommendation {
	var recommendations []*OptimizationRecommendation

	// Compression recommendations
	if len(report.CompressionAnalysis.UncompressedBackups) > 0 {
		recommendations = append(recommendations, &OptimizationRecommendation{
			Type:               "compression",
			Priority:           "high",
			EstimatedSavings:   report.CompressionAnalysis.PotentialCompressionSavings,
			Description:        fmt.Sprintf("Enable compression for %d uncompressed backups", len(report.CompressionAnalysis.UncompressedBackups)),
			ActionRequired:     "Enable compression in backup configuration",
			Impact:             "low",
			ImplementationTime: "immediate",
		})
	}

	// Retention recommendations
	if len(report.RetentionAnalysis.BackupsEligibleForCleanup) > 0 {
		recommendations = append(recommendations, &OptimizationRecommendation{
			Type:               "retention",
			Priority:           "medium",
			EstimatedSavings:   report.RetentionAnalysis.PotentialRetentionSavings,
			Description:        fmt.Sprintf("Clean up %d old backups eligible for removal", len(report.RetentionAnalysis.BackupsEligibleForCleanup)),
			ActionRequired:     "Apply retention policies or manually delete old backups",
			Impact:             "low",
			ImplementationTime: "immediate",
		})
	}

	// Deduplication recommendations
	if report.DuplicationAnalysis.PotentialDeduplicationSavings > 0 {
		recommendations = append(recommendations, &OptimizationRecommendation{
			Type:               "deduplication",
			Priority:           "low",
			EstimatedSavings:   report.DuplicationAnalysis.PotentialDeduplicationSavings,
			Description:        fmt.Sprintf("Implement deduplication to save %d bytes from duplicate backups", report.DuplicationAnalysis.PotentialDeduplicationSavings),
			ActionRequired:     "Implement backup deduplication system",
			Impact:             "medium",
			ImplementationTime: "long",
		})
	}

	return recommendations
}
