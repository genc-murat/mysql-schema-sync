package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MigrationTool handles migration of existing configurations to include backup settings
type MigrationTool struct {
	configPaths []string
	dryRun      bool
	verbose     bool
}

// NewMigrationTool creates a new configuration migration tool
func NewMigrationTool(dryRun, verbose bool) *MigrationTool {
	return &MigrationTool{
		dryRun:  dryRun,
		verbose: verbose,
	}
}

// MigrationResult represents the result of a configuration migration
type MigrationResult struct {
	ConfigPath    string
	Success       bool
	AlreadyExists bool
	Error         error
	BackupPath    string
}

// DiscoverConfigurations discovers existing configuration files
func (mt *MigrationTool) DiscoverConfigurations() ([]string, error) {
	var configPaths []string

	// Common configuration file locations
	searchPaths := []string{
		"./config.yaml",
		"./mysql-schema-sync.yaml",
		"./.mysql-schema-sync.yaml",
	}

	// Add home directory paths
	homeDir, err := os.UserHomeDir()
	if err == nil {
		searchPaths = append(searchPaths,
			filepath.Join(homeDir, ".mysql-schema-sync.yaml"),
			filepath.Join(homeDir, ".config", "mysql-schema-sync", "config.yaml"),
			filepath.Join(homeDir, ".config", "mysql-schema-sync.yaml"),
		)
	}

	// Check each path
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			configPaths = append(configPaths, path)
			if mt.verbose {
				fmt.Printf("Found configuration file: %s\n", path)
			}
		}
	}

	return configPaths, nil
}

// MigrateAll migrates all discovered configuration files
func (mt *MigrationTool) MigrateAll() ([]MigrationResult, error) {
	configPaths, err := mt.DiscoverConfigurations()
	if err != nil {
		return nil, fmt.Errorf("failed to discover configurations: %w", err)
	}

	if len(configPaths) == 0 {
		return nil, fmt.Errorf("no configuration files found")
	}

	var results []MigrationResult

	for _, configPath := range configPaths {
		result := mt.MigrateConfiguration(configPath)
		results = append(results, result)
	}

	return results, nil
}

// MigrateConfiguration migrates a single configuration file
func (mt *MigrationTool) MigrateConfiguration(configPath string) MigrationResult {
	result := MigrationResult{
		ConfigPath: configPath,
	}

	if mt.verbose {
		fmt.Printf("Processing configuration: %s\n", configPath)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		result.Error = fmt.Errorf("configuration file does not exist: %s", configPath)
		return result
	}

	// Read existing configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read configuration file: %w", err)
		return result
	}

	// Parse existing configuration
	var existingConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &existingConfig); err != nil {
		result.Error = fmt.Errorf("failed to parse existing configuration: %w", err)
		return result
	}

	// Check if backup configuration already exists
	if _, exists := existingConfig["backup"]; exists {
		result.AlreadyExists = true
		result.Success = true
		if mt.verbose {
			fmt.Printf("Backup configuration already exists in %s\n", configPath)
		}
		return result
	}

	// Create backup of original file
	if !mt.dryRun {
		backupPath := mt.generateBackupPath(configPath)
		if err := mt.createBackup(configPath, backupPath); err != nil {
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result
		}
		result.BackupPath = backupPath
	}

	// Add backup configuration
	existingConfig["backup"] = mt.getDefaultBackupConfig()

	// Write updated configuration
	if !mt.dryRun {
		updatedData, err := yaml.Marshal(existingConfig)
		if err != nil {
			result.Error = fmt.Errorf("failed to marshal updated configuration: %w", err)
			return result
		}

		if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
			result.Error = fmt.Errorf("failed to write updated configuration: %w", err)
			return result
		}
	}

	result.Success = true
	if mt.verbose {
		if mt.dryRun {
			fmt.Printf("Would migrate configuration: %s\n", configPath)
		} else {
			fmt.Printf("Successfully migrated configuration: %s\n", configPath)
		}
	}

	return result
}

// generateBackupPath generates a unique backup path
func (mt *MigrationTool) generateBackupPath(configPath string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s.backup-%s", configPath, timestamp)
}

// createBackup creates a backup of the original configuration file
func (mt *MigrationTool) createBackup(configPath, backupPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// getDefaultBackupConfig returns the default backup configuration
func (mt *MigrationTool) getDefaultBackupConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled": false,
		"storage": map[string]interface{}{
			"provider": "local",
			"local": map[string]interface{}{
				"base_path":   "./backups",
				"permissions": "0755",
			},
		},
		"retention": map[string]interface{}{
			"max_backups":      10,
			"cleanup_interval": "24h",
		},
		"compression": map[string]interface{}{
			"enabled":   false,
			"algorithm": "gzip",
			"level":     6,
			"threshold": 1024,
		},
		"encryption": map[string]interface{}{
			"enabled":          false,
			"key_source":       "env",
			"key_env_var":      "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY",
			"rotation_enabled": false,
			"rotation_days":    90,
		},
		"validation": map[string]interface{}{
			"enabled":             true,
			"checksum_algorithm":  "sha256",
			"validate_on_create":  true,
			"validate_on_restore": true,
			"validation_timeout":  "5m",
			"dry_run_validation":  true,
		},
	}
}

// ValidateAllMigrations validates all migrated configurations
func (mt *MigrationTool) ValidateAllMigrations(results []MigrationResult) error {
	var validationErrors []string

	for _, result := range results {
		if !result.Success {
			continue
		}

		if err := mt.validateMigratedConfig(result.ConfigPath); err != nil {
			validationErrors = append(validationErrors,
				fmt.Sprintf("%s: %v", result.ConfigPath, err))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors:\n%s", strings.Join(validationErrors, "\n"))
	}

	return nil
}

// validateMigratedConfig validates a migrated configuration
func (mt *MigrationTool) validateMigratedConfig(configPath string) error {
	ci := NewConfigIntegration()
	return ci.ValidateIntegratedConfig(configPath)
}

// PrintMigrationSummary prints a summary of migration results
func (mt *MigrationTool) PrintMigrationSummary(results []MigrationResult) {
	fmt.Println("\nMigration Summary:")
	fmt.Println("==================")

	successful := 0
	alreadyExists := 0
	failed := 0

	for _, result := range results {
		if result.Success {
			if result.AlreadyExists {
				alreadyExists++
				fmt.Printf("✓ %s (already has backup config)\n", result.ConfigPath)
			} else {
				successful++
				if mt.dryRun {
					fmt.Printf("✓ %s (would be migrated)\n", result.ConfigPath)
				} else {
					fmt.Printf("✓ %s (migrated successfully)\n", result.ConfigPath)
					if result.BackupPath != "" {
						fmt.Printf("  Backup created: %s\n", result.BackupPath)
					}
				}
			}
		} else {
			failed++
			fmt.Printf("✗ %s (failed: %v)\n", result.ConfigPath, result.Error)
		}
	}

	fmt.Printf("\nResults: %d successful, %d already exist, %d failed\n",
		successful, alreadyExists, failed)

	if mt.dryRun {
		fmt.Println("\nThis was a dry run. No files were modified.")
		fmt.Println("Run without --dry-run to perform the actual migration.")
	}
}

// CreateDefaultConfiguration creates a new configuration file with backup settings
func (mt *MigrationTool) CreateDefaultConfiguration(configPath string) error {
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default configuration with backup settings
	defaultConfig := map[string]interface{}{
		"source": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "",
			"password": "",
			"database": "",
			"timeout":  "30s",
		},
		"target": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "",
			"password": "",
			"database": "",
			"timeout":  "30s",
		},
		"dry_run":      false,
		"verbose":      false,
		"auto_approve": false,
		"backup":       mt.getDefaultBackupConfig(),
		"display": map[string]interface{}{
			"color_enabled":   true,
			"theme":           "dark",
			"output_format":   "table",
			"use_icons":       true,
			"show_progress":   true,
			"interactive":     true,
			"table_style":     "default",
			"max_table_width": 120,
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	if mt.verbose {
		fmt.Printf("Default configuration with backup settings created: %s\n", configPath)
	}

	return nil
}

// RollbackMigration rolls back a migration by restoring from backup
func (mt *MigrationTool) RollbackMigration(configPath, backupPath string) error {
	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Read backup data
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Restore original configuration
	if err := os.WriteFile(configPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to restore configuration: %w", err)
	}

	if mt.verbose {
		fmt.Printf("Configuration rolled back from backup: %s -> %s\n", backupPath, configPath)
	}

	return nil
}

// ListBackupFiles lists all backup files for a configuration
func (mt *MigrationTool) ListBackupFiles(configPath string) ([]string, error) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var backupFiles []string
	prefix := base + ".backup"

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			backupFiles = append(backupFiles, filepath.Join(dir, entry.Name()))
		}
	}

	return backupFiles, nil
}

// CleanupBackupFiles removes old backup files, keeping only the most recent ones
func (mt *MigrationTool) CleanupBackupFiles(configPath string, keepCount int) error {
	backupFiles, err := mt.ListBackupFiles(configPath)
	if err != nil {
		return err
	}

	if len(backupFiles) <= keepCount {
		return nil // Nothing to clean up
	}

	// Sort by modification time (newest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, path := range backupFiles {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: path, modTime: info.ModTime()})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.Before(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove old backup files
	for i := keepCount; i < len(files); i++ {
		if err := os.Remove(files[i].path); err != nil {
			if mt.verbose {
				fmt.Printf("Warning: failed to remove backup file %s: %v\n", files[i].path, err)
			}
		} else if mt.verbose {
			fmt.Printf("Removed old backup file: %s\n", files[i].path)
		}
	}

	return nil
}
