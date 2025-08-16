package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// BackupValidatorImpl implements the BackupValidator interface
type BackupValidatorImpl struct {
	displayService ValidatorDisplayService
}

// ValidatorDisplayService interface for progress tracking and user feedback
type ValidatorDisplayService interface {
	ShowProgress(current, total int, message string)
	Info(message string)
	Error(message string)
}

// NewBackupValidator creates a new backup validator
func NewBackupValidator(displayService ValidatorDisplayService) BackupValidator {
	return &BackupValidatorImpl{
		displayService: displayService,
	}
}

// ValidateIntegrity validates the integrity of a backup
func (v *BackupValidatorImpl) ValidateIntegrity(ctx context.Context, backup *Backup) error {
	if backup == nil {
		return fmt.Errorf("backup is nil")
	}

	if backup.Metadata == nil {
		return fmt.Errorf("backup metadata is missing")
	}

	// Basic integrity checks
	if backup.ID == "" {
		return fmt.Errorf("backup ID is empty")
	}

	if backup.Metadata.DatabaseName == "" {
		return fmt.Errorf("database name is empty")
	}

	// TODO: Add more comprehensive integrity checks when dependencies are resolved
	return nil
}

// ValidateCompleteness validates that the backup contains all expected schema elements
func (v *BackupValidatorImpl) ValidateCompleteness(ctx context.Context, backup *Backup, originalSchema interface{}) error {
	// TODO: Implement when schema dependency is resolved
	return fmt.Errorf("not implemented - schema dependency needs to be resolved")
}

// ValidateRestorability performs a dry-run validation to ensure the backup can be restored
func (v *BackupValidatorImpl) ValidateRestorability(ctx context.Context, backup *Backup) error {
	// TODO: Implement when dependencies are resolved
	return fmt.Errorf("not implemented - dependencies need to be resolved")
}

// CalculateChecksum calculates a checksum for the given data
func (v *BackupValidatorImpl) CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// VerifyChecksum verifies that the data matches the expected checksum
func (v *BackupValidatorImpl) VerifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := v.CalculateChecksum(data)
	return actualChecksum == expectedChecksum
}
