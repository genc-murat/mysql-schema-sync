package backup

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"

	"mysql-schema-sync/internal/display"
	"mysql-schema-sync/internal/logging"
)

// backupManager implements the BackupManager interface
type backupManager struct {
	storageProvider StorageProvider
	validator       BackupValidator
	extractor       *SchemaExtractor
	compressionMgr  *CompressionManager
	encryptionMgr   *EncryptionManager
	logger          *logging.Logger
	displayService  display.DisplayService
	config          *BackupSystemConfig
	databaseService interface{} // TODO: Define proper interface when dependency is resolved
	retentionMgr    RetentionManager
	storageMon      StorageMonitor
}

// NewBackupManager creates a new backup manager
func NewBackupManager(config *BackupSystemConfig) (BackupManager, error) {
	if config == nil {
		return nil, NewValidationError("backup system configuration is required", nil)
	}

	// Set defaults if not provided
	config.SetDefaults()

	// Create storage provider
	factory := NewStorageProviderFactory()
	storageProvider, err := factory.CreateStorageProvider(context.Background(), config.Storage)
	if err != nil {
		return nil, NewStorageError("failed to create storage provider", err)
	}

	// Create validator
	validator := NewBackupValidator(nil)

	// Create extractor
	extractor := NewSchemaExtractor(nil)

	// Create compression manager
	compressionMgr := NewCompressionManager()

	// Create encryption manager
	encryptionMgr := NewEncryptionManager(&config.Encryption)

	// Create logger
	logger := logging.NewDefaultLogger()

	// Create database service
	// TODO: Implement when database dependency is resolved
	var databaseService interface{}

	manager := &backupManager{
		storageProvider: storageProvider,
		validator:       validator,
		extractor:       extractor,
		compressionMgr:  compressionMgr,
		encryptionMgr:   encryptionMgr,
		logger:          logger,
		config:          config,
		databaseService: databaseService,
	}

	// Initialize retention manager and storage monitor
	manager.retentionMgr = NewRetentionManager(manager, &config.Retention, logger)
	manager.storageMon = NewStorageMonitor(manager, config, logger)

	return manager, nil
}

// NewBackupManagerWithDependencies creates a backup manager with provided dependencies
func NewBackupManagerWithDependencies(
	storageProvider StorageProvider,
	validator BackupValidator,
	extractor *SchemaExtractor,
	compressionMgr *CompressionManager,
	encryptionMgr *EncryptionManager,
	logger *logging.Logger,
	displayService display.DisplayService,
	databaseService interface{}, // TODO: Define proper interface when dependency is resolved
	config *BackupSystemConfig,
) BackupManager {
	manager := &backupManager{
		storageProvider: storageProvider,
		validator:       validator,
		extractor:       extractor,
		compressionMgr:  compressionMgr,
		encryptionMgr:   encryptionMgr,
		logger:          logger,
		displayService:  displayService,
		databaseService: databaseService,
		config:          config,
	}

	// Initialize retention manager and storage monitor
	manager.retentionMgr = NewRetentionManager(manager, &config.Retention, logger)
	manager.storageMon = NewStorageMonitor(manager, config, logger)

	// Set display service on extractor if provided
	if displayService != nil && extractor != nil {
		extractor.SetDisplayService(&extractorDisplayAdapter{displayService})
	}

	return manager
}

// SetDisplayService sets the display service for progress tracking
func (bm *backupManager) SetDisplayService(displayService display.DisplayService) {
	bm.displayService = displayService
	if bm.extractor != nil {
		bm.extractor.SetDisplayService(&extractorDisplayAdapter{displayService})
	}
}

// CreateBackup creates a new backup
// TODO: Implement when dependencies are resolved
func (bm *backupManager) CreateBackup(ctx context.Context, config BackupConfig) (*Backup, error) {
	return nil, fmt.Errorf("not implemented - dependencies need to be resolved")

}

// CreatePreMigrationBackup creates a backup before migration with migration context
// TODO: Implement when migration dependency is resolved
func (bm *backupManager) CreatePreMigrationBackup(ctx context.Context, config BackupConfig, migrationPlan interface{}) (*Backup, error) {
	// Add migration context to the backup config
	if config.Description == "" {
		config.Description = "Automatic pre-migration backup"
	} else {
		config.Description = fmt.Sprintf("Pre-migration backup: %s", config.Description)
	}

	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
	config.Tags["type"] = "pre-migration"
	config.Tags["migration_plan_hash"] = "not-implemented" // TODO: Implement when migration dependency is resolved

	// Create the backup
	backup, err := bm.CreateBackup(ctx, config)
	if err != nil {
		return nil, err
	}

	// Add migration context to metadata
	if backup.Metadata != nil {
		backup.Metadata.MigrationContext = &MigrationContext{
			PlanHash:       "not-implemented", // TODO: Implement when migration dependency is resolved
			SourceSchema:   config.DatabaseConfig.Database,
			PreMigrationID: backup.ID,
			MigrationTime:  time.Now(),
			ToolVersion:    "1.0.0", // TODO: Get from build info
		}
	}

	return backup, nil
}

// CreateManualBackup creates a manual backup with custom description and tags
func (bm *backupManager) CreateManualBackup(ctx context.Context, config BackupConfig) (*Backup, error) {
	// Ensure manual backup is tagged appropriately
	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
	config.Tags["type"] = "manual"

	if config.Description == "" {
		config.Description = "Manual backup"
	}

	// Create the backup
	return bm.CreateBackup(ctx, config)
}

// CreateBackupWithProgress creates a backup with detailed progress reporting
// TODO: Implement when dependencies are resolved
func (bm *backupManager) CreateBackupWithProgress(ctx context.Context, config BackupConfig, progressCallback func(stage string, progress int, total int)) (*Backup, error) {
	return nil, fmt.Errorf("not implemented - dependencies need to be resolved")
}

// ListBackups lists backups with optional filtering
func (bm *backupManager) ListBackups(ctx context.Context, filter BackupFilter) ([]*BackupMetadata, error) {
	bm.logDebug("Listing backups with filter")

	// Convert to storage filter
	storageFilter := StorageFilter{
		MaxItems: 1000, // Default limit
	}

	// Get backups from storage
	backups, err := bm.storageProvider.List(ctx, storageFilter)
	if err != nil {
		return nil, NewStorageError("failed to list backups", err)
	}

	// Apply filters
	filteredBackups := bm.applyBackupFilter(backups, filter)

	// Sort by creation time (newest first)
	sort.Slice(filteredBackups, func(i, j int) bool {
		return filteredBackups[i].CreatedAt.After(filteredBackups[j].CreatedAt)
	})

	bm.logDebug(fmt.Sprintf("Found %d backups matching filter", len(filteredBackups)))

	return filteredBackups, nil
}

// ListBackupsWithSorting lists backups with custom sorting options
func (bm *backupManager) ListBackupsWithSorting(ctx context.Context, filter BackupFilter, sortBy string, ascending bool) ([]*BackupMetadata, error) {
	backups, err := bm.ListBackups(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply custom sorting
	switch sortBy {
	case "created_at", "date":
		sort.Slice(backups, func(i, j int) bool {
			if ascending {
				return backups[i].CreatedAt.Before(backups[j].CreatedAt)
			}
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})
	case "size":
		sort.Slice(backups, func(i, j int) bool {
			if ascending {
				return backups[i].Size < backups[j].Size
			}
			return backups[i].Size > backups[j].Size
		})
	case "database", "database_name":
		sort.Slice(backups, func(i, j int) bool {
			if ascending {
				return backups[i].DatabaseName < backups[j].DatabaseName
			}
			return backups[i].DatabaseName > backups[j].DatabaseName
		})
	case "description":
		sort.Slice(backups, func(i, j int) bool {
			if ascending {
				return backups[i].Description < backups[j].Description
			}
			return backups[i].Description > backups[j].Description
		})
	default:
		// Default to creation time sorting
		sort.Slice(backups, func(i, j int) bool {
			if ascending {
				return backups[i].CreatedAt.Before(backups[j].CreatedAt)
			}
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})
	}

	return backups, nil
}

// GetBackupDetails retrieves detailed information about a specific backup
func (bm *backupManager) GetBackupDetails(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID is required", nil)
	}

	bm.logDebug(fmt.Sprintf("Getting backup details: %s", backupID))

	metadata, err := bm.storageProvider.GetMetadata(ctx, backupID)
	if err != nil {
		return nil, NewStorageError("failed to get backup metadata", err)
	}

	return metadata, nil
}

// GetBackupsByDatabase lists all backups for a specific database
func (bm *backupManager) GetBackupsByDatabase(ctx context.Context, databaseName string) ([]*BackupMetadata, error) {
	filter := BackupFilter{
		DatabaseName: databaseName,
	}
	return bm.ListBackups(ctx, filter)
}

// GetBackupsByType lists backups by type (manual, pre-migration, etc.)
func (bm *backupManager) GetBackupsByType(ctx context.Context, backupType string) ([]*BackupMetadata, error) {
	filter := BackupFilter{
		Tags: map[string]string{
			"type": backupType,
		},
	}
	return bm.ListBackups(ctx, filter)
}

// GetRecentBackups gets the most recent backups up to a specified limit
func (bm *backupManager) GetRecentBackups(ctx context.Context, limit int) ([]*BackupMetadata, error) {
	backups, err := bm.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, err
	}

	// Limit results
	if limit > 0 && len(backups) > limit {
		backups = backups[:limit]
	}

	return backups, nil
}

// DeleteBackup deletes a backup
func (bm *backupManager) DeleteBackup(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Deleting backup: %s", backupID))

	// Show progress if display service is available
	var spinner display.SpinnerHandle
	if bm.displayService != nil {
		spinner = bm.displayService.StartSpinner(fmt.Sprintf("Deleting backup %s...", backupID))
	}

	// Delete from storage
	if err := bm.storageProvider.Delete(ctx, backupID); err != nil {
		bm.stopSpinner(spinner, "")
		return NewStorageError("failed to delete backup", err)
	}

	bm.stopSpinner(spinner, fmt.Sprintf("Backup deleted: %s", backupID))
	bm.logInfo(fmt.Sprintf("Backup deletion completed: %s", backupID))

	return nil
}

// DeleteBackupWithConfirmation deletes a backup with safety checks and confirmation
func (bm *backupManager) DeleteBackupWithConfirmation(ctx context.Context, backupID string, force bool) error {
	if backupID == "" {
		return NewValidationError("backup ID is required", nil)
	}

	// Get backup metadata first for safety checks
	metadata, err := bm.GetBackupDetails(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup details for safety check: %w", err)
	}

	// Safety checks
	if !force {
		// Check if this is a recent backup (less than 24 hours old)
		if time.Since(metadata.CreatedAt) < 24*time.Hour {
			return NewValidationError("cannot delete recent backup without force flag (backup is less than 24 hours old)", nil)
		}

		// Check if this is the only backup for the database
		databaseBackups, err := bm.GetBackupsByDatabase(ctx, metadata.DatabaseName)
		if err != nil {
			return fmt.Errorf("failed to check database backups for safety: %w", err)
		}

		if len(databaseBackups) == 1 {
			return NewValidationError("cannot delete the only backup for database without force flag", nil)
		}

		// Check if this is a pre-migration backup
		if backupType, exists := metadata.Tags["type"]; exists && backupType == "pre-migration" {
			return NewValidationError("cannot delete pre-migration backup without force flag", nil)
		}
	}

	// Proceed with deletion
	return bm.DeleteBackup(ctx, backupID)
}

// DeleteMultipleBackups deletes multiple backups with progress reporting
func (bm *backupManager) DeleteMultipleBackups(ctx context.Context, backupIDs []string, force bool) error {
	if len(backupIDs) == 0 {
		return NewValidationError("at least one backup ID is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Deleting %d backups", len(backupIDs)))

	var errors []error
	successCount := 0

	for i, backupID := range backupIDs {
		// Show progress
		if bm.displayService != nil {
			bm.displayService.ShowProgress(i+1, len(backupIDs), fmt.Sprintf("Deleting backup %s", backupID))
		}

		if err := bm.DeleteBackupWithConfirmation(ctx, backupID, force); err != nil {
			bm.logError(fmt.Sprintf("Failed to delete backup %s: %v", backupID, err))
			errors = append(errors, fmt.Errorf("backup %s: %w", backupID, err))
		} else {
			successCount++
		}
	}

	bm.logInfo(fmt.Sprintf("Deleted %d out of %d backups", successCount, len(backupIDs)))

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete %d backups: %v", len(errors), errors)
	}

	return nil
}

// DeleteOldBackups deletes backups older than the specified duration
func (bm *backupManager) DeleteOldBackups(ctx context.Context, olderThan time.Duration, dryRun bool) ([]string, error) {
	bm.logInfo(fmt.Sprintf("Finding backups older than %v (dry run: %v)", olderThan, dryRun))

	// Get all backups
	allBackups, err := bm.ListBackups(ctx, BackupFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Find old backups
	cutoffTime := time.Now().Add(-olderThan)
	var oldBackupIDs []string

	for _, backup := range allBackups {
		if backup.CreatedAt.Before(cutoffTime) {
			oldBackupIDs = append(oldBackupIDs, backup.ID)
		}
	}

	bm.logInfo(fmt.Sprintf("Found %d backups older than %v", len(oldBackupIDs), olderThan))

	if dryRun {
		return oldBackupIDs, nil
	}

	// Delete old backups
	if len(oldBackupIDs) > 0 {
		if err := bm.DeleteMultipleBackups(ctx, oldBackupIDs, true); err != nil {
			return oldBackupIDs, fmt.Errorf("failed to delete old backups: %w", err)
		}
	}

	return oldBackupIDs, nil
}

// ValidateBackup validates a backup
func (bm *backupManager) ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Validating backup: %s", backupID))

	// Show progress if display service is available
	var spinner display.SpinnerHandle
	if bm.displayService != nil {
		spinner = bm.displayService.StartSpinner(fmt.Sprintf("Validating backup %s...", backupID))
	}

	// Retrieve backup
	backup, err := bm.storageProvider.Retrieve(ctx, backupID)
	if err != nil {
		bm.stopSpinner(spinner, "")
		return nil, NewStorageError("failed to retrieve backup for validation", err)
	}

	// Validate integrity
	if err := bm.validator.ValidateIntegrity(ctx, backup); err != nil {
		bm.stopSpinner(spinner, "")
		return &ValidationResult{
			Valid:         false,
			Errors:        []string{err.Error()},
			CheckedAt:     time.Now(),
			ChecksumValid: false,
		}, nil
	}

	result := &ValidationResult{
		Valid:         true,
		CheckedAt:     time.Now(),
		ChecksumValid: true,
	}

	bm.stopSpinner(spinner, fmt.Sprintf("Backup validation completed: %s", backupID))
	bm.logInfo(fmt.Sprintf("Backup validation completed: %s", backupID))

	return result, nil
}

// ExportBackup exports a backup to a destination
func (bm *backupManager) ExportBackup(ctx context.Context, backupID string, destination string) error {
	if backupID == "" {
		return NewValidationError("backup ID is required", nil)
	}
	if destination == "" {
		return NewValidationError("destination is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Exporting backup %s to %s", backupID, destination))

	// Show progress if display service is available
	var spinner display.SpinnerHandle
	if bm.displayService != nil {
		spinner = bm.displayService.StartSpinner(fmt.Sprintf("Exporting backup %s...", backupID))
	}

	// Retrieve backup
	backup, err := bm.storageProvider.Retrieve(ctx, backupID)
	if err != nil {
		bm.stopSpinner(spinner, "")
		return NewStorageError("failed to retrieve backup for export", err)
	}

	// Serialize backup
	backupData, err := backup.ToJSON()
	if err != nil {
		bm.stopSpinner(spinner, "")
		return NewValidationError("failed to serialize backup for export", err)
	}

	// Write to destination file
	if err := os.WriteFile(destination, backupData, 0644); err != nil {
		bm.stopSpinner(spinner, "")
		return NewStorageError("failed to write backup to destination", err)
	}

	bm.stopSpinner(spinner, fmt.Sprintf("Backup exported: %s -> %s", backupID, destination))
	bm.logInfo(fmt.Sprintf("Backup export completed: %s -> %s", backupID, destination))

	return nil
}

// ExportBackupWithFormat exports a backup in a specific format
func (bm *backupManager) ExportBackupWithFormat(ctx context.Context, backupID string, destination string, format string) error {
	if backupID == "" {
		return NewValidationError("backup ID is required", nil)
	}
	if destination == "" {
		return NewValidationError("destination is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Exporting backup %s to %s in format %s", backupID, destination, format))

	// Show progress if display service is available
	var spinner display.SpinnerHandle
	if bm.displayService != nil {
		spinner = bm.displayService.StartSpinner(fmt.Sprintf("Exporting backup %s...", backupID))
	}

	// Retrieve backup
	backup, err := bm.storageProvider.Retrieve(ctx, backupID)
	if err != nil {
		bm.stopSpinner(spinner, "")
		return NewStorageError("failed to retrieve backup for export", err)
	}

	var exportData []byte

	switch format {
	case "json":
		exportData, err = backup.ToJSON()
		if err != nil {
			bm.stopSpinner(spinner, "")
			return NewValidationError("failed to serialize backup to JSON", err)
		}
	case "sql":
		// Export as SQL statements
		exportData, err = bm.exportBackupAsSQL(backup)
		if err != nil {
			bm.stopSpinner(spinner, "")
			return NewValidationError("failed to export backup as SQL", err)
		}
	case "yaml":
		// Export as YAML (if needed)
		exportData, err = bm.exportBackupAsYAML(backup)
		if err != nil {
			bm.stopSpinner(spinner, "")
			return NewValidationError("failed to export backup as YAML", err)
		}
	default:
		bm.stopSpinner(spinner, "")
		return NewValidationError(fmt.Sprintf("unsupported export format: %s", format), nil)
	}

	// Write to destination file
	if err := os.WriteFile(destination, exportData, 0644); err != nil {
		bm.stopSpinner(spinner, "")
		return NewStorageError("failed to write backup to destination", err)
	}

	bm.stopSpinner(spinner, fmt.Sprintf("Backup exported: %s -> %s (%s)", backupID, destination, format))
	bm.logInfo(fmt.Sprintf("Backup export completed: %s -> %s (%s)", backupID, destination, format))

	return nil
}

// ImportBackup imports a backup from a file
func (bm *backupManager) ImportBackup(ctx context.Context, sourcePath string, validateIntegrity bool) (*Backup, error) {
	if sourcePath == "" {
		return nil, NewValidationError("source path is required", nil)
	}

	bm.logInfo(fmt.Sprintf("Importing backup from %s", sourcePath))

	// Show progress if display service is available
	var spinner display.SpinnerHandle
	if bm.displayService != nil {
		spinner = bm.displayService.StartSpinner(fmt.Sprintf("Importing backup from %s...", sourcePath))
	}

	// Read backup file
	backupData, err := os.ReadFile(sourcePath)
	if err != nil {
		bm.stopSpinner(spinner, "")
		return nil, NewStorageError("failed to read backup file", err)
	}

	// Parse backup
	backup := &Backup{}
	if err := backup.FromJSON(backupData); err != nil {
		bm.stopSpinner(spinner, "")
		return nil, NewValidationError("failed to parse backup file", err)
	}

	// Validate backup integrity if requested
	if validateIntegrity {
		bm.updateSpinner(spinner, "Validating backup integrity...")
		if err := bm.validator.ValidateIntegrity(ctx, backup); err != nil {
			bm.stopSpinner(spinner, "")
			return nil, NewValidationError("backup integrity validation failed", err)
		}
	}

	// Generate new backup ID to avoid conflicts
	originalID := backup.ID
	backup.ID = GenerateBackupID()
	if backup.Metadata != nil {
		backup.Metadata.ID = backup.ID
	}

	// Store imported backup
	bm.updateSpinner(spinner, "Storing imported backup...")
	if err := bm.storageProvider.Store(ctx, backup); err != nil {
		bm.stopSpinner(spinner, "")
		return nil, NewStorageError("failed to store imported backup", err)
	}

	bm.stopSpinner(spinner, fmt.Sprintf("Backup imported: %s (original ID: %s)", backup.ID, originalID))
	bm.logInfo(fmt.Sprintf("Backup import completed: %s from %s", backup.ID, sourcePath))

	return backup, nil
}

// ImportBackupWithID imports a backup and assigns a specific ID
func (bm *backupManager) ImportBackupWithID(ctx context.Context, sourcePath string, backupID string, validateIntegrity bool) (*Backup, error) {
	if sourcePath == "" {
		return nil, NewValidationError("source path is required", nil)
	}
	if backupID == "" {
		return nil, NewValidationError("backup ID is required", nil)
	}

	// Check if backup ID already exists
	if _, err := bm.GetBackupDetails(ctx, backupID); err == nil {
		return nil, NewValidationError(fmt.Sprintf("backup with ID %s already exists", backupID), nil)
	}

	backup, err := bm.ImportBackup(ctx, sourcePath, validateIntegrity)
	if err != nil {
		return nil, err
	}

	// Update backup ID
	oldID := backup.ID
	backup.ID = backupID
	if backup.Metadata != nil {
		backup.Metadata.ID = backupID
	}

	// Delete the backup with the temporary ID and store with the new ID
	if err := bm.storageProvider.Delete(ctx, oldID); err != nil {
		bm.logError(fmt.Sprintf("Failed to delete temporary backup %s: %v", oldID, err))
	}

	if err := bm.storageProvider.Store(ctx, backup); err != nil {
		return nil, NewStorageError("failed to store backup with new ID", err)
	}

	bm.logInfo(fmt.Sprintf("Backup imported with ID: %s from %s", backupID, sourcePath))

	return backup, nil
}

// GetRetentionManager returns the retention manager
func (bm *backupManager) GetRetentionManager() RetentionManager {
	return bm.retentionMgr
}

// GetStorageMonitor returns the storage monitor
func (bm *backupManager) GetStorageMonitor() StorageMonitor {
	return bm.storageMon
}

// Helper methods

func (bm *backupManager) applyBackupFilter(backups []*BackupMetadata, filter BackupFilter) []*BackupMetadata {
	var filtered []*BackupMetadata

	for _, backup := range backups {
		// Filter by database name
		if filter.DatabaseName != "" && backup.DatabaseName != filter.DatabaseName {
			continue
		}

		// Filter by date range
		if filter.StartDate != nil && backup.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && backup.CreatedAt.After(*filter.EndDate) {
			continue
		}

		// Filter by status
		if filter.Status != nil && backup.Status != *filter.Status {
			continue
		}

		// Filter by tags
		if len(filter.Tags) > 0 {
			matchesTags := true
			for key, value := range filter.Tags {
				if backupValue, exists := backup.Tags[key]; !exists || backupValue != value {
					matchesTags = false
					break
				}
			}
			if !matchesTags {
				continue
			}
		}

		filtered = append(filtered, backup)
	}

	return filtered
}

func (bm *backupManager) getCurrentUser() string {
	if currentUser, err := user.Current(); err == nil {
		return currentUser.Username
	}
	return "unknown"
}

func (bm *backupManager) logInfo(message string) {
	if bm.logger != nil {
		bm.logger.Info(message)
	}
}

func (bm *backupManager) logDebug(message string) {
	if bm.logger != nil {
		bm.logger.Debug(message)
	}
}

func (bm *backupManager) logError(message string) {
	if bm.logger != nil {
		bm.logger.Error(message)
	}
}

func (bm *backupManager) updateSpinner(spinner display.SpinnerHandle, message string) {
	if bm.displayService != nil && spinner != nil {
		bm.displayService.UpdateSpinner(spinner, message)
	}
}

func (bm *backupManager) stopSpinner(spinner display.SpinnerHandle, finalMessage string) {
	if bm.displayService != nil && spinner != nil {
		bm.displayService.StopSpinner(spinner, finalMessage)
	}
}

// Helper methods for export functionality

// exportBackupAsSQL exports a backup as SQL CREATE statements
func (bm *backupManager) exportBackupAsSQL(backup *Backup) ([]byte, error) {
	var sqlBuilder strings.Builder

	// Add header comment
	sqlBuilder.WriteString(fmt.Sprintf("-- Backup Export: %s\n", backup.ID))
	sqlBuilder.WriteString(fmt.Sprintf("-- Database: %s\n", backup.Metadata.DatabaseName))
	sqlBuilder.WriteString(fmt.Sprintf("-- Created: %s\n", backup.Metadata.CreatedAt.Format(time.RFC3339)))
	sqlBuilder.WriteString("-- Generated by mysql-schema-sync backup system\n\n")

	// TODO: Export schema as CREATE TABLE statements when schema dependency is resolved
	if backup.SchemaSnapshot != nil {
		// For now, just add a placeholder
		createSQL := "-- Schema export not implemented - dependency needs to be resolved"
		sqlBuilder.WriteString(createSQL)
		sqlBuilder.WriteString("\n\n")
	}

	// Export views
	for _, view := range backup.Views {
		sqlBuilder.WriteString(fmt.Sprintf("-- View: %s\n", view.Name))
		sqlBuilder.WriteString(view.Definition)
		sqlBuilder.WriteString(";\n\n")
	}

	// Export procedures
	for _, proc := range backup.Procedures {
		sqlBuilder.WriteString(fmt.Sprintf("-- Procedure: %s\n", proc.Name))
		sqlBuilder.WriteString(proc.Definition)
		sqlBuilder.WriteString(";\n\n")
	}

	// Export functions
	for _, fn := range backup.Functions {
		sqlBuilder.WriteString(fmt.Sprintf("-- Function: %s\n", fn.Name))
		sqlBuilder.WriteString(fn.Definition)
		sqlBuilder.WriteString(";\n\n")
	}

	// Export triggers
	for _, trigger := range backup.Triggers {
		sqlBuilder.WriteString(fmt.Sprintf("-- Trigger: %s on %s\n", trigger.Name, trigger.Table))
		sqlBuilder.WriteString(trigger.Definition)
		sqlBuilder.WriteString(";\n\n")
	}

	return []byte(sqlBuilder.String()), nil
}

// exportBackupAsYAML exports a backup as YAML format
func (bm *backupManager) exportBackupAsYAML(backup *Backup) ([]byte, error) {
	// For now, just convert to JSON and then to YAML-like format
	// In a real implementation, you might want to use a YAML library
	jsonData, err := backup.ToJSON()
	if err != nil {
		return nil, err
	}

	// Simple YAML-like conversion (this is a basic implementation)
	yamlData := strings.ReplaceAll(string(jsonData), "{", "")
	yamlData = strings.ReplaceAll(yamlData, "}", "")
	yamlData = strings.ReplaceAll(yamlData, "[", "")
	yamlData = strings.ReplaceAll(yamlData, "]", "")
	yamlData = strings.ReplaceAll(yamlData, "\"", "")
	yamlData = strings.ReplaceAll(yamlData, ",", "")

	return []byte(yamlData), nil
}

// generateCreateTableSQL generates a CREATE TABLE statement from a table schema
// TODO: Implement when schema dependency is resolved
func (bm *backupManager) generateCreateTableSQL(table interface{}) (string, error) {
	return "", fmt.Errorf("not implemented - schema dependency needs to be resolved")
}

// extractorDisplayAdapter adapts display.DisplayService to ExtractorDisplayService
type extractorDisplayAdapter struct {
	displayService display.DisplayService
}

func (eda *extractorDisplayAdapter) ShowProgress(current, total int, message string) {
	eda.displayService.ShowProgress(current, total, message)
}

func (eda *extractorDisplayAdapter) Info(message string) {
	eda.displayService.Info(message)
}

func (eda *extractorDisplayAdapter) Error(message string) {
	eda.displayService.Error(message)
}

func (eda *extractorDisplayAdapter) Debug(message string) {
	// Display service doesn't have Debug method, use Info instead
	eda.displayService.Info(message)
}
