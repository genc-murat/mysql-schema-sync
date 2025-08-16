package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mysql-schema-sync/internal/application"
	"mysql-schema-sync/internal/backup"
	"mysql-schema-sync/internal/display"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Backup creation flags
	backupDescription string
	backupTags        []string
	compressionType   string
	encryptionKey     string
	storageProvider   string

	// Backup listing flags
	listDatabase  string
	listStartDate string
	listEndDate   string
	listStatus    string
	listFormat    string
	listLimit     int

	// Backup validation flags
	validateIntegrity     bool
	validateCompleteness  bool
	validateRestorability bool

	// Export/Import flags
	exportDestination string
	importSource      string
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage database backups",
	Long: `Create, list, validate, and manage database backups.

The backup system provides comprehensive backup and restore capabilities
for MySQL schema synchronization. It supports multiple storage providers,
compression options, and encryption for secure backup management.

Examples:
  # Create a backup with description
  mysql-schema-sync backup create --description "Pre-migration backup"
  
  # List all backups
  mysql-schema-sync backup list
  
  # List backups for specific database
  mysql-schema-sync backup list --database mydb
  
  # Validate backup integrity
  mysql-schema-sync backup validate backup-123 --integrity
  
  # Export backup to file
  mysql-schema-sync backup export backup-123 --destination /path/to/backup.tar.gz`,
}

// backupCreateCmd creates a new backup
var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new database backup",
	Long: `Create a new backup of the target database schema.

This command creates a complete backup including tables, indexes, constraints,
triggers, views, stored procedures, and functions. The backup is validated
for integrity and stored according to the configured storage provider.

Examples:
  # Create basic backup
  mysql-schema-sync backup create
  
  # Create backup with description and tags
  mysql-schema-sync backup create --description "Pre-migration backup" --tags env=prod,version=1.2.3
  
  # Create compressed and encrypted backup
  mysql-schema-sync backup create --compression gzip --encryption-key /path/to/key.txt`,
	RunE: runBackupCreate,
}

// backupListCmd lists existing backups
var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing backups",
	Long: `List existing backups with filtering and formatting options.

This command displays backup metadata including creation time, size,
status, and description. Results can be filtered by database, date range,
or status, and formatted as table, JSON, or YAML.

Examples:
  # List all backups
  mysql-schema-sync backup list
  
  # List backups for specific database
  mysql-schema-sync backup list --database mydb
  
  # List recent backups (last 7 days)
  mysql-schema-sync backup list --start-date 7d
  
  # List backups in JSON format
  mysql-schema-sync backup list --format json`,
	RunE: runBackupList,
}

// backupValidateCmd validates backup integrity
var backupValidateCmd = &cobra.Command{
	Use:   "validate <backup-id>",
	Short: "Validate backup integrity and completeness",
	Long: `Validate backup integrity, completeness, and restorability.

This command performs comprehensive validation of backup files including
checksum verification, completeness checking against original schema,
and dry-run restore validation to ensure the backup is restorable.

Examples:
  # Validate backup integrity
  mysql-schema-sync backup validate backup-123 --integrity
  
  # Validate backup completeness
  mysql-schema-sync backup validate backup-123 --completeness
  
  # Full validation (integrity, completeness, restorability)
  mysql-schema-sync backup validate backup-123 --integrity --completeness --restorability`,
	Args: cobra.ExactArgs(1),
	RunE: runBackupValidate,
}

// backupDeleteCmd deletes a backup
var backupDeleteCmd = &cobra.Command{
	Use:   "delete <backup-id>",
	Short: "Delete a backup",
	Long: `Delete a backup from storage.

This command permanently removes a backup from the configured storage
provider. The operation requires confirmation unless --auto-approve is used.

Examples:
  # Delete backup with confirmation
  mysql-schema-sync backup delete backup-123
  
  # Delete backup without confirmation
  mysql-schema-sync backup delete backup-123 --auto-approve`,
	Args: cobra.ExactArgs(1),
	RunE: runBackupDelete,
}

// backupExportCmd exports a backup
var backupExportCmd = &cobra.Command{
	Use:   "export <backup-id>",
	Short: "Export a backup to a file",
	Long: `Export a backup to a local file or different storage location.

This command exports a backup from the configured storage provider to
a specified destination. The exported backup maintains all metadata
and can be imported later.

Examples:
  # Export backup to local file
  mysql-schema-sync backup export backup-123 --destination /path/to/backup.tar.gz
  
  # Export backup to different directory
  mysql-schema-sync backup export backup-123 --destination /backups/`,
	Args: cobra.ExactArgs(1),
	RunE: runBackupExport,
}

func init() {
	// Add backup command to root
	rootCmd.AddCommand(backupCmd)

	// Add subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupValidateCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupExportCmd)

	// Backup creation flags
	backupCreateCmd.Flags().StringVar(&backupDescription, "description", "", "backup description")
	backupCreateCmd.Flags().StringSliceVar(&backupTags, "tags", []string{}, "backup tags in key=value format")
	backupCreateCmd.Flags().StringVar(&compressionType, "compression", "gzip", "compression type (none, gzip, lz4, zstd)")
	backupCreateCmd.Flags().StringVar(&encryptionKey, "encryption-key", "", "path to encryption key file")
	backupCreateCmd.Flags().StringVar(&storageProvider, "storage", "", "storage provider override (local, s3, azure, gcs)")

	// Backup listing flags
	backupListCmd.Flags().StringVar(&listDatabase, "database", "", "filter by database name")
	backupListCmd.Flags().StringVar(&listStartDate, "start-date", "", "filter by start date (YYYY-MM-DD or relative like 7d, 1w, 1m)")
	backupListCmd.Flags().StringVar(&listEndDate, "end-date", "", "filter by end date (YYYY-MM-DD)")
	backupListCmd.Flags().StringVar(&listStatus, "status", "", "filter by status (creating, completed, failed, validating, corrupted)")
	backupListCmd.Flags().StringVar(&listFormat, "format", "table", "output format (table, json, yaml)")
	backupListCmd.Flags().IntVar(&listLimit, "limit", 50, "maximum number of backups to list")

	// Backup validation flags
	backupValidateCmd.Flags().BoolVar(&validateIntegrity, "integrity", false, "validate backup integrity")
	backupValidateCmd.Flags().BoolVar(&validateCompleteness, "completeness", false, "validate backup completeness")
	backupValidateCmd.Flags().BoolVar(&validateRestorability, "restorability", false, "validate backup restorability")

	// Export flags
	backupExportCmd.Flags().StringVar(&exportDestination, "destination", "", "export destination path")
	backupExportCmd.MarkFlagRequired("destination")
}

// runBackupCreate creates a new backup
func runBackupCreate(cmd *cobra.Command, args []string) error {
	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create backup manager
	manager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Parse tags
	tags, err := parseTags(backupTags)
	if err != nil {
		return fmt.Errorf("invalid tags format: %w", err)
	}

	// Read encryption key if provided
	var encKey []byte
	if encryptionKey != "" {
		encKey, err = os.ReadFile(encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to read encryption key: %w", err)
		}
	}

	// Create backup configuration
	backupCfg := backup.BackupConfig{
		DatabaseConfig: backup.DatabaseConfig{
			Host:     config.TargetDB.Host,
			Port:     config.TargetDB.Port,
			Username: config.TargetDB.Username,
			Password: config.TargetDB.Password,
			Database: config.TargetDB.Database,
		},
		StorageConfig:   backupConfig.Storage,
		CompressionType: parseCompressionType(compressionType),
		EncryptionKey:   encKey,
		Description:     backupDescription,
		Tags:            tags,
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()
	displayService.Info("Creating backup...")

	// Create backup
	createdBackup, err := manager.CreateBackup(ctx, backupCfg)
	if err != nil {
		return fmt.Errorf("backup creation failed: %w", err)
	}

	// Display success message
	displayService.Success(fmt.Sprintf("Backup created successfully: %s", createdBackup.ID))
	displayService.Info(fmt.Sprintf("Database: %s", createdBackup.Metadata.DatabaseName))
	displayService.Info(fmt.Sprintf("Size: %s", formatBytes(createdBackup.Metadata.Size)))
	if createdBackup.Metadata.CompressedSize > 0 {
		ratio := float64(createdBackup.Metadata.CompressedSize) / float64(createdBackup.Metadata.Size) * 100
		displayService.Info(fmt.Sprintf("Compressed size: %s (%.1f%%)", formatBytes(createdBackup.Metadata.CompressedSize), ratio))
	}
	displayService.Info(fmt.Sprintf("Created at: %s", createdBackup.Metadata.CreatedAt.Format(time.RFC3339)))

	return nil
}

// runBackupList lists existing backups
func runBackupList(cmd *cobra.Command, args []string) error {
	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create backup manager
	manager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Build filter
	filter, err := buildBackupFilter()
	if err != nil {
		return fmt.Errorf("invalid filter parameters: %w", err)
	}

	ctx := context.Background()

	// List backups
	backups, err := manager.ListBackups(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Apply limit
	if listLimit > 0 && len(backups) > listLimit {
		backups = backups[:listLimit]
	}

	// Display results
	return displayBackupList(backups, listFormat, &config.Display)
}

// runBackupValidate validates a backup
func runBackupValidate(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create backup manager
	manager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	// If no specific validation flags are set, validate everything
	if !validateIntegrity && !validateCompleteness && !validateRestorability {
		validateIntegrity = true
		validateCompleteness = true
		validateRestorability = true
	}

	displayService.Info(fmt.Sprintf("Validating backup: %s", backupID))

	// Validate backup
	result, err := manager.ValidateBackup(ctx, backupID)
	if err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	// Display results
	if result.Valid {
		displayService.Success("Backup validation passed")
	} else {
		displayService.Error("Backup validation failed")
	}

	if len(result.Errors) > 0 {
		displayService.Error("Validation errors:")
		for _, errMsg := range result.Errors {
			displayService.Error(fmt.Sprintf("  - %s", errMsg))
		}
	}

	if len(result.Warnings) > 0 {
		displayService.Warning("Validation warnings:")
		for _, warning := range result.Warnings {
			displayService.Warning(fmt.Sprintf("  - %s", warning))
		}
	}

	displayService.Info(fmt.Sprintf("Checksum valid: %t", result.ChecksumValid))
	displayService.Info(fmt.Sprintf("Validated at: %s", result.CheckedAt.Format(time.RFC3339)))

	if !result.Valid {
		return fmt.Errorf("backup validation failed")
	}

	return nil
}

// runBackupDelete deletes a backup
func runBackupDelete(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create backup manager
	manager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	// Get backup metadata for confirmation
	backups, err := manager.ListBackups(ctx, backup.BackupFilter{})
	if err != nil {
		return fmt.Errorf("failed to get backup information: %w", err)
	}

	var targetBackup *backup.BackupMetadata
	for _, b := range backups {
		if b.ID == backupID {
			targetBackup = b
			break
		}
	}

	if targetBackup == nil {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Show backup information
	displayService.Info(fmt.Sprintf("Backup to delete: %s", backupID))
	displayService.Info(fmt.Sprintf("Database: %s", targetBackup.DatabaseName))
	displayService.Info(fmt.Sprintf("Created: %s", targetBackup.CreatedAt.Format(time.RFC3339)))
	displayService.Info(fmt.Sprintf("Size: %s", formatBytes(targetBackup.Size)))

	// Confirm deletion unless auto-approve is set
	if !config.AutoApprove {
		dialog := displayService.NewConfirmationDialog()
		dialog.SetTitle("Delete Backup")
		dialog.SetMessage(fmt.Sprintf("Are you sure you want to delete backup %s?", backupID))
		dialog.AddOption("y", "Yes", "Delete the backup", false)
		dialog.AddCancelOption("n", "No", "Cancel deletion", true)

		result, err := dialog.Show()
		if err != nil {
			return fmt.Errorf("confirmation dialog error: %w", err)
		}

		if !result.Confirmed || result.Cancelled {
			displayService.Info("Backup deletion cancelled")
			return nil
		}
	}

	// Delete backup
	err = manager.DeleteBackup(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	displayService.Success(fmt.Sprintf("Backup deleted successfully: %s", backupID))
	return nil
}

// runBackupExport exports a backup
func runBackupExport(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create backup manager
	manager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	displayService.Info(fmt.Sprintf("Exporting backup: %s", backupID))
	displayService.Info(fmt.Sprintf("Destination: %s", exportDestination))

	// Export backup
	err = manager.ExportBackup(ctx, backupID, exportDestination)
	if err != nil {
		return fmt.Errorf("backup export failed: %w", err)
	}

	displayService.Success(fmt.Sprintf("Backup exported successfully to: %s", exportDestination))
	return nil
}

// Helper functions

// buildBackupSystemConfig creates backup system configuration from application config
func buildBackupSystemConfig(config *application.Config) (*backup.BackupSystemConfig, error) {
	// Load backup configuration from viper or use defaults
	backupConfig := &backup.BackupSystemConfig{}

	// Set storage configuration
	backupConfig.Storage = backup.StorageConfig{
		Provider: backup.StorageProviderLocal, // Default to local
		Local: &backup.LocalConfig{
			BasePath:    filepath.Join(os.TempDir(), "mysql-schema-sync-backups"),
			Permissions: 0755,
		},
	}

	// Override with viper configuration if available
	if viper.IsSet("backup.storage.provider") {
		provider := viper.GetString("backup.storage.provider")
		switch strings.ToUpper(provider) {
		case "LOCAL":
			backupConfig.Storage.Provider = backup.StorageProviderLocal
			if viper.IsSet("backup.storage.local") {
				backupConfig.Storage.Local = &backup.LocalConfig{
					BasePath:    viper.GetString("backup.storage.local.base_path"),
					Permissions: os.FileMode(viper.GetInt("backup.storage.local.permissions")),
				}
			}
		case "S3":
			backupConfig.Storage.Provider = backup.StorageProviderS3
			backupConfig.Storage.S3 = &backup.S3Config{
				Bucket:    viper.GetString("backup.storage.s3.bucket"),
				Region:    viper.GetString("backup.storage.s3.region"),
				AccessKey: viper.GetString("backup.storage.s3.access_key"),
				SecretKey: viper.GetString("backup.storage.s3.secret_key"),
			}
		case "AZURE":
			backupConfig.Storage.Provider = backup.StorageProviderAzure
			backupConfig.Storage.Azure = &backup.AzureConfig{
				AccountName:   viper.GetString("backup.storage.azure.account_name"),
				AccountKey:    viper.GetString("backup.storage.azure.account_key"),
				ContainerName: viper.GetString("backup.storage.azure.container_name"),
			}
		case "GCS":
			backupConfig.Storage.Provider = backup.StorageProviderGCS
			backupConfig.Storage.GCS = &backup.GCSConfig{
				Bucket:          viper.GetString("backup.storage.gcs.bucket"),
				CredentialsPath: viper.GetString("backup.storage.gcs.credentials_path"),
				ProjectID:       viper.GetString("backup.storage.gcs.project_id"),
			}
		}
	}

	// Override storage provider if specified via flag
	if storageProvider != "" {
		switch strings.ToUpper(storageProvider) {
		case "LOCAL":
			backupConfig.Storage.Provider = backup.StorageProviderLocal
		case "S3":
			backupConfig.Storage.Provider = backup.StorageProviderS3
		case "AZURE":
			backupConfig.Storage.Provider = backup.StorageProviderAzure
		case "GCS":
			backupConfig.Storage.Provider = backup.StorageProviderGCS
		default:
			return nil, fmt.Errorf("invalid storage provider: %s", storageProvider)
		}
	}

	// Set retention configuration
	backupConfig.Retention = backup.RetentionConfig{
		MaxBackups:      viper.GetInt("backup.retention.max_backups"),
		MaxAge:          viper.GetDuration("backup.retention.max_age"),
		CleanupInterval: viper.GetDuration("backup.retention.cleanup_interval"),
		KeepDaily:       viper.GetInt("backup.retention.keep_daily"),
		KeepWeekly:      viper.GetInt("backup.retention.keep_weekly"),
		KeepMonthly:     viper.GetInt("backup.retention.keep_monthly"),
	}

	// Set compression configuration
	backupConfig.Compression = backup.CompressionConfig{
		Algorithm: backup.CompressionTypeGzip,
		Level:     6,
		Enabled:   true,
		Threshold: 1024, // Compress files larger than 1KB
	}

	// Set encryption configuration
	backupConfig.Encryption = backup.EncryptionConfig{
		Enabled:         viper.GetBool("backup.encryption.enabled"),
		KeySource:       viper.GetString("backup.encryption.key_source"),
		KeyPath:         viper.GetString("backup.encryption.key_path"),
		KeyEnvVar:       viper.GetString("backup.encryption.key_env_var"),
		RotationEnabled: viper.GetBool("backup.encryption.rotation_enabled"),
		RotationDays:    viper.GetInt("backup.encryption.rotation_days"),
	}

	// Set validation configuration
	backupConfig.Validation = backup.ValidationConfig{
		Enabled:           true,
		ValidateOnCreate:  true,
		ValidateOnRestore: false, // Expensive, disabled by default
		ChecksumAlgorithm: "sha256",
		ValidationTimeout: 30 * time.Second,
		DryRunValidation:  false,
	}

	return backupConfig, nil
}

// parseTags parses tag strings in key=value format
func parseTags(tagStrings []string) (map[string]string, error) {
	tags := make(map[string]string)
	for _, tagStr := range tagStrings {
		parts := strings.SplitN(tagStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format '%s', expected key=value", tagStr)
		}
		tags[parts[0]] = parts[1]
	}
	return tags, nil
}

// parseCompressionType converts string to compression type
func parseCompressionType(compressionStr string) backup.CompressionType {
	switch strings.ToUpper(compressionStr) {
	case "NONE":
		return backup.CompressionTypeNone
	case "GZIP":
		return backup.CompressionTypeGzip
	case "LZ4":
		return backup.CompressionTypeLZ4
	case "ZSTD":
		return backup.CompressionTypeZstd
	default:
		return backup.CompressionTypeGzip // Default
	}
}

// buildBackupFilter creates a backup filter from CLI flags
func buildBackupFilter() (backup.BackupFilter, error) {
	filter := backup.BackupFilter{}

	// Set database filter
	if listDatabase != "" {
		filter.DatabaseName = listDatabase
	}

	// Parse date filters
	if listStartDate != "" {
		startDate, err := parseDate(listStartDate)
		if err != nil {
			return filter, fmt.Errorf("invalid start date: %w", err)
		}
		filter.CreatedAfter = &startDate
	}

	if listEndDate != "" {
		endDate, err := parseDate(listEndDate)
		if err != nil {
			return filter, fmt.Errorf("invalid end date: %w", err)
		}
		filter.CreatedBefore = &endDate
	}

	// Parse status filter
	if listStatus != "" {
		status := backup.BackupStatus(strings.ToUpper(listStatus))
		filter.Status = &status
	}

	return filter, nil
}

// parseDate parses date strings including relative dates
func parseDate(dateStr string) (time.Time, error) {
	// Try parsing as RFC3339 first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}

	// Try parsing as date only
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}

	// Try parsing relative dates
	if strings.HasSuffix(dateStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(dateStr, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative date format: %s", dateStr)
		}
		return time.Now().AddDate(0, 0, -days), nil
	}

	if strings.HasSuffix(dateStr, "w") {
		weeks, err := strconv.Atoi(strings.TrimSuffix(dateStr, "w"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative date format: %s", dateStr)
		}
		return time.Now().AddDate(0, 0, -weeks*7), nil
	}

	if strings.HasSuffix(dateStr, "m") {
		months, err := strconv.Atoi(strings.TrimSuffix(dateStr, "m"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative date format: %s", dateStr)
		}
		return time.Now().AddDate(0, -months, 0), nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
}

// displayBackupList displays backup list in the specified format
func displayBackupList(backups []*backup.BackupMetadata, format string, displayConfig *application.DisplayConfig) error {
	switch strings.ToLower(format) {
	case "json":
		return displayBackupListJSON(backups)
	case "yaml":
		return displayBackupListYAML(backups)
	case "table":
		return displayBackupListTable(backups, displayConfig)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// displayBackupListJSON displays backups as JSON
func displayBackupListJSON(backups []*backup.BackupMetadata) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backups)
}

// displayBackupListYAML displays backups as YAML
func displayBackupListYAML(backups []*backup.BackupMetadata) error {
	// For now, use JSON format as YAML implementation would require additional dependency
	// In a real implementation, you would use gopkg.in/yaml.v3
	fmt.Println("# Backup List (YAML format)")
	for i, backup := range backups {
		fmt.Printf("- id: %s\n", backup.ID)
		fmt.Printf("  database_name: %s\n", backup.DatabaseName)
		fmt.Printf("  created_at: %s\n", backup.CreatedAt.Format(time.RFC3339))
		fmt.Printf("  status: %s\n", backup.Status)
		fmt.Printf("  size: %d\n", backup.Size)
		if backup.CompressedSize > 0 {
			fmt.Printf("  compressed_size: %d\n", backup.CompressedSize)
		}
		if backup.Description != "" {
			fmt.Printf("  description: %s\n", backup.Description)
		}
		if len(backup.Tags) > 0 {
			fmt.Printf("  tags:\n")
			for k, v := range backup.Tags {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		if i < len(backups)-1 {
			fmt.Println()
		}
	}
	return nil
}

// displayBackupListTable displays backups as a formatted table
func displayBackupListTable(backups []*backup.BackupMetadata, displayConfig *application.DisplayConfig) error {
	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Create display service
	displayService := display.NewDisplayService(displayConfig)

	// Prepare table data
	headers := []string{"ID", "Database", "Created", "Status", "Size", "Description"}
	rows := make([][]string, len(backups))

	for i, backup := range backups {
		createdAt := backup.CreatedAt.Format("2006-01-02 15:04:05")
		size := formatBytes(backup.Size)
		description := backup.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		rows[i] = []string{
			backup.ID,
			backup.DatabaseName,
			createdAt,
			string(backup.Status),
			size,
			description,
		}
	}

	// Display table
	displayService.PrintTable(headers, rows)
	displayService.Info(fmt.Sprintf("Total backups: %d", len(backups)))

	return nil
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
