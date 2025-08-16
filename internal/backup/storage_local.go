package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorageProvider implements StorageProvider for local file system storage
type LocalStorageProvider struct {
	basePath    string
	permissions os.FileMode
}

// NewLocalStorageProvider creates a new LocalStorageProvider instance
func NewLocalStorageProvider(config *LocalConfig) (*LocalStorageProvider, error) {
	if config == nil {
		return nil, NewValidationError("local storage configuration is required", nil)
	}

	if err := config.Validate(); err != nil {
		return nil, NewValidationError("invalid local storage configuration", err)
	}

	provider := &LocalStorageProvider{
		basePath:    config.BasePath,
		permissions: config.Permissions,
	}

	// Ensure base directory exists
	if err := provider.ensureBaseDirectory(); err != nil {
		return nil, NewStorageError("failed to create base directory", err)
	}

	return provider, nil
}

// Store saves a backup to the local file system
func (lsp *LocalStorageProvider) Store(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return NewValidationError("backup cannot be nil", nil)
	}

	// Create backup directory structure
	backupDir := lsp.getBackupDirectory(backup.ID)
	if err := os.MkdirAll(backupDir, lsp.permissions); err != nil {
		return NewStorageError("failed to create backup directory", err)
	}

	// Update storage location in metadata before validation
	backup.Metadata.StorageLocation = backupDir

	// Calculate checksum after updating storage location but before validation
	if err := backup.CalculateChecksum(); err != nil {
		return NewStorageError("failed to calculate backup checksum", err)
	}
	backup.Metadata.Checksum = backup.Checksum

	if err := backup.Validate(); err != nil {
		return NewValidationError("invalid backup data", err)
	}

	// Save backup data
	backupPath := filepath.Join(backupDir, "backup.json")
	if err := lsp.saveBackupData(backupPath, backup); err != nil {
		return err
	}

	// Save metadata separately for quick access
	metadataPath := filepath.Join(backupDir, "metadata.json")
	if err := lsp.saveMetadata(metadataPath, backup.Metadata); err != nil {
		return err
	}

	return nil
}

// Retrieve loads a backup from the local file system
func (lsp *LocalStorageProvider) Retrieve(ctx context.Context, backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	backupPath := filepath.Join(lsp.getBackupDirectory(backupID), "backup.json")

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil, NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}

	// Load backup data
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, NewStorageError("failed to read backup file", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, NewStorageError("failed to unmarshal backup data", err)
	}

	// Validate backup integrity
	if !backup.VerifyChecksum() {
		return nil, NewCorruptionError("backup checksum verification failed", nil)
	}

	return &backup, nil
}

// Delete removes a backup from the local file system
func (lsp *LocalStorageProvider) Delete(ctx context.Context, backupID string) error {
	if backupID == "" {
		return NewValidationError("backup ID cannot be empty", nil)
	}

	backupDir := lsp.getBackupDirectory(backupID)

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}

	// Remove entire backup directory
	if err := os.RemoveAll(backupDir); err != nil {
		return NewStorageError("failed to delete backup directory", err)
	}

	return nil
}

// List returns a list of backup metadata matching the filter
func (lsp *LocalStorageProvider) List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error) {
	var backups []*BackupMetadata

	// Walk through backup directories
	err := filepath.WalkDir(lsp.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the base path
		if !d.IsDir() || path == lsp.basePath {
			return nil
		}

		// Check if this looks like a backup directory (contains metadata.json)
		metadataPath := filepath.Join(path, "metadata.json")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			return nil
		}

		// Load metadata
		metadata, err := lsp.loadMetadata(metadataPath)
		if err != nil {
			// Log error but continue processing other backups
			return nil
		}

		// Apply filter
		if lsp.matchesFilter(metadata, filter) {
			backups = append(backups, metadata)
		}

		// Check if we've reached the maximum number of items
		if filter.MaxItems > 0 && len(backups) >= filter.MaxItems {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil, NewStorageError("failed to list backups", err)
	}

	return backups, nil
}

// GetMetadata retrieves metadata for a specific backup
func (lsp *LocalStorageProvider) GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	if backupID == "" {
		return nil, NewValidationError("backup ID cannot be empty", nil)
	}

	metadataPath := filepath.Join(lsp.getBackupDirectory(backupID), "metadata.json")

	// Check if metadata file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, NewStorageError(fmt.Sprintf("backup %s not found", backupID), err)
	}

	return lsp.loadMetadata(metadataPath)
}

// Helper methods

// ensureBaseDirectory creates the base directory if it doesn't exist
func (lsp *LocalStorageProvider) ensureBaseDirectory() error {
	if err := os.MkdirAll(lsp.basePath, lsp.permissions); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", lsp.basePath, err)
	}
	return nil
}

// getBackupDirectory returns the directory path for a specific backup
func (lsp *LocalStorageProvider) getBackupDirectory(backupID string) string {
	// Sanitize backup ID to prevent directory traversal
	sanitizedID := lsp.sanitizeBackupID(backupID)
	return filepath.Join(lsp.basePath, sanitizedID)
}

// sanitizeBackupID removes potentially dangerous characters from backup ID
func (lsp *LocalStorageProvider) sanitizeBackupID(backupID string) string {
	// Replace any path separators and other dangerous characters
	sanitized := strings.ReplaceAll(backupID, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, "..", "_")
	return sanitized
}

// saveBackupData saves the complete backup data to a file
func (lsp *LocalStorageProvider) saveBackupData(path string, backup *Backup) error {
	data, err := backup.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize backup data", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return NewStorageError("failed to write backup file", err)
	}

	return nil
}

// saveMetadata saves backup metadata to a separate file for quick access
func (lsp *LocalStorageProvider) saveMetadata(path string, metadata *BackupMetadata) error {
	data, err := metadata.ToJSON()
	if err != nil {
		return NewStorageError("failed to serialize metadata", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return NewStorageError("failed to write metadata file", err)
	}

	return nil
}

// loadMetadata loads backup metadata from a file
func (lsp *LocalStorageProvider) loadMetadata(path string) (*BackupMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewStorageError("failed to read metadata file", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, NewStorageError("failed to unmarshal metadata", err)
	}

	if err := metadata.Validate(); err != nil {
		return nil, NewValidationError("invalid metadata", err)
	}

	return &metadata, nil
}

// matchesFilter checks if metadata matches the given filter
func (lsp *LocalStorageProvider) matchesFilter(metadata *BackupMetadata, filter StorageFilter) bool {
	// Apply prefix filter if specified
	if filter.Prefix != "" && !strings.HasPrefix(metadata.ID, filter.Prefix) {
		return false
	}

	return true
}

// GetBasePath returns the base path for the storage provider
func (lsp *LocalStorageProvider) GetBasePath() string {
	return lsp.basePath
}

// GetPermissions returns the file permissions used by the storage provider
func (lsp *LocalStorageProvider) GetPermissions() os.FileMode {
	return lsp.permissions
}

// GetStorageInfo returns information about the storage provider
func (lsp *LocalStorageProvider) GetStorageInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":    "local",
		"base_path":   lsp.basePath,
		"permissions": lsp.permissions.String(),
	}
}

// HealthCheck verifies that the storage provider is accessible and functional
func (lsp *LocalStorageProvider) HealthCheck(ctx context.Context) error {
	// Check if base directory exists and is writable
	testFile := filepath.Join(lsp.basePath, ".health_check")

	// Try to create a test file
	if err := os.WriteFile(testFile, []byte("health_check"), 0644); err != nil {
		return NewStorageError("storage provider health check failed: cannot write to base directory", err)
	}

	// Try to read the test file
	if _, err := os.ReadFile(testFile); err != nil {
		return NewStorageError("storage provider health check failed: cannot read from base directory", err)
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log warning but don't fail health check
		return nil
	}

	return nil
}
