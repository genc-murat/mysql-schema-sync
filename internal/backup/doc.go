// Package backup provides comprehensive backup and rollback functionality for MySQL schema synchronization.
//
// This package implements a robust backup system that automatically creates backups before schema migrations
// and provides reliable rollback capabilities. The system is designed with the following key principles:
//
// 1. Safety First: All schema changes are preceded by automatic backups
// 2. Reliability: Comprehensive validation and integrity checking
// 3. Flexibility: Support for multiple storage providers and compression options
// 4. Security: Built-in encryption and access control
// 5. Observability: Detailed logging and monitoring capabilities
//
// Core Components:
//
// - BackupManager: Orchestrates backup creation, validation, and management
// - StorageProvider: Abstracts storage operations for different backends (local, S3, Azure, GCS)
// - RollbackManager: Handles rollback operations and schema restoration
// - BackupValidator: Ensures backup integrity and completeness
//
// The package integrates seamlessly with the existing mysql-schema-sync architecture while providing
// enterprise-grade backup and recovery capabilities.
//
// Example usage:
//
//	// Create a backup manager
//	manager := backup.NewBackupManager(storageProvider, validator)
//
//	// Create a backup before migration
//	backupConfig := backup.BackupConfig{
//		DatabaseConfig: dbConfig,
//		StorageConfig:  storageConfig,
//		Description:    "Pre-migration backup",
//	}
//	backup, err := manager.CreateBackup(ctx, backupConfig)
//	if err != nil {
//		return fmt.Errorf("backup creation failed: %w", err)
//	}
//
//	// List available backups
//	backups, err := manager.ListBackups(ctx, backup.BackupFilter{})
//	if err != nil {
//		return fmt.Errorf("failed to list backups: %w", err)
//	}
//
//	// Rollback to a previous state
//	rollbackManager := backup.NewRollbackManager(storageProvider)
//	plan, err := rollbackManager.PlanRollback(ctx, backupID)
//	if err != nil {
//		return fmt.Errorf("rollback planning failed: %w", err)
//	}
//
//	err = rollbackManager.ExecuteRollback(ctx, plan)
//	if err != nil {
//		return fmt.Errorf("rollback execution failed: %w", err)
//	}
package backup
