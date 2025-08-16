package backup

import (
	"context"
)

// BackupManager orchestrates backup creation, validation, and management operations
type BackupManager interface {
	CreateBackup(ctx context.Context, config BackupConfig) (*Backup, error)
	ListBackups(ctx context.Context, filter BackupFilter) ([]*BackupMetadata, error)
	DeleteBackup(ctx context.Context, backupID string) error
	ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error)
	ExportBackup(ctx context.Context, backupID string, destination string) error

	// GetBackupsByDatabase returns backups for a specific database
	GetBackupsByDatabase(ctx context.Context, databaseName string) ([]*BackupMetadata, error)

	// GetRetentionManager returns the retention manager
	GetRetentionManager() RetentionManager

	// GetStorageMonitor returns the storage monitor
	GetStorageMonitor() StorageMonitor
}

// StorageProvider abstracts storage operations for different backend types
type StorageProvider interface {
	Store(ctx context.Context, backup *Backup) error
	Retrieve(ctx context.Context, backupID string) (*Backup, error)
	Delete(ctx context.Context, backupID string) error
	List(ctx context.Context, filter StorageFilter) ([]*BackupMetadata, error)
	GetMetadata(ctx context.Context, backupID string) (*BackupMetadata, error)
}

// RollbackManager handles rollback operations and schema restoration
type RollbackManager interface {
	ListRollbackPoints(ctx context.Context, database string) ([]*RollbackPoint, error)
	PlanRollback(ctx context.Context, backupID string) (*RollbackPlan, error)
	ExecuteRollback(ctx context.Context, plan *RollbackPlan) error
	ValidateRollback(ctx context.Context, backupID string) error
}

// BackupValidator ensures backup integrity and completeness
type BackupValidator interface {
	ValidateIntegrity(ctx context.Context, backup *Backup) error
	ValidateCompleteness(ctx context.Context, backup *Backup, originalSchema interface{}) error
	ValidateRestorability(ctx context.Context, backup *Backup) error
	CalculateChecksum(data []byte) string
	VerifyChecksum(data []byte, expectedChecksum string) bool
}
