package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mysql-schema-sync/internal/logging"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BackupLogger provides structured logging for backup operations with correlation IDs and audit trails
type BackupLogger struct {
	logger        *logging.Logger
	auditLogger   *logrus.Logger
	correlationID string
}

// BackupLoggerConfig holds configuration for backup logging
type BackupLoggerConfig struct {
	Logger         *logging.Logger
	AuditLogFile   string
	CorrelationID  string
	EnableAuditLog bool
}

// LogEntry represents a structured log entry for backup operations
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	Operation     string                 `json:"operation"`
	BackupID      string                 `json:"backup_id,omitempty"`
	DatabaseName  string                 `json:"database_name,omitempty"`
	Status        string                 `json:"status"`
	Duration      string                 `json:"duration,omitempty"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogEntry represents an audit trail entry for compliance
type AuditLogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	UserID        string                 `json:"user_id,omitempty"`
	Operation     string                 `json:"operation"`
	Resource      string                 `json:"resource"`
	Action        string                 `json:"action"`
	Result        string                 `json:"result"`
	Details       map[string]interface{} `json:"details,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
}

// NewBackupLogger creates a new backup logger with correlation ID support
func NewBackupLogger(config BackupLoggerConfig) (*BackupLogger, error) {
	correlationID := config.CorrelationID
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	bl := &BackupLogger{
		logger:        config.Logger,
		correlationID: correlationID,
	}

	// Set up audit logging if enabled
	if config.EnableAuditLog && config.AuditLogFile != "" {
		auditLogger := logrus.New()

		// Create audit log directory if it doesn't exist
		auditDir := filepath.Dir(config.AuditLogFile)
		if err := os.MkdirAll(auditDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create audit log directory: %w", err)
		}

		// Open audit log file
		auditFile, err := os.OpenFile(config.AuditLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}

		auditLogger.SetOutput(auditFile)
		auditLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		auditLogger.SetLevel(logrus.InfoLevel)

		bl.auditLogger = auditLogger
	}

	return bl, nil
}

// GetCorrelationID returns the current correlation ID
func (bl *BackupLogger) GetCorrelationID() string {
	return bl.correlationID
}

// WithCorrelationID creates a new logger with a different correlation ID
func (bl *BackupLogger) WithCorrelationID(correlationID string) *BackupLogger {
	return &BackupLogger{
		logger:        bl.logger,
		auditLogger:   bl.auditLogger,
		correlationID: correlationID,
	}
}

// LogBackupStart logs the start of a backup operation
func (bl *BackupLogger) LogBackupStart(ctx context.Context, backupID, databaseName string, config BackupConfig) func(error, *BackupMetadata) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     "backup_create",
		BackupID:      backupID,
		DatabaseName:  databaseName,
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"compression_type":   string(config.CompressionType),
			"encryption_enabled": len(config.EncryptionKey) > 0,
			"storage_provider":   string(config.StorageConfig.Provider),
			"description":        config.Description,
			"tags":               config.Tags,
		},
	}

	bl.logStructured(entry)
	bl.logAudit(ctx, "backup", "create", "started", map[string]interface{}{
		"backup_id":     backupID,
		"database_name": databaseName,
	})

	return func(err error, metadata *BackupMetadata) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		if metadata != nil {
			entry.Metadata["size"] = metadata.Size
			entry.Metadata["compressed_size"] = metadata.CompressedSize
			entry.Metadata["checksum"] = metadata.Checksum
		}

		bl.logStructured(entry)

		result := "success"
		if err != nil {
			result = "failure"
		}

		bl.logAudit(ctx, "backup", "create", result, map[string]interface{}{
			"backup_id":     backupID,
			"database_name": databaseName,
			"duration":      duration.String(),
			"error":         entry.Error,
		})
	}
}

// LogBackupValidation logs backup validation operations
func (bl *BackupLogger) LogBackupValidation(ctx context.Context, backupID string, validationType string) func(error, *ValidationResult) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     "backup_validation",
		BackupID:      backupID,
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"validation_type": validationType,
		},
	}

	bl.logStructured(entry)

	return func(err error, result *ValidationResult) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		if result != nil {
			entry.Metadata["valid"] = result.Valid
			entry.Metadata["checksum_valid"] = result.ChecksumValid
			entry.Metadata["error_count"] = len(result.Errors)
			entry.Metadata["warning_count"] = len(result.Warnings)
		}

		bl.logStructured(entry)

		auditResult := "success"
		if err != nil || (result != nil && !result.Valid) {
			auditResult = "failure"
		}

		bl.logAudit(ctx, "backup", "validate", auditResult, map[string]interface{}{
			"backup_id":       backupID,
			"validation_type": validationType,
			"duration":        duration.String(),
		})
	}
}

// LogBackupDeletion logs backup deletion operations
func (bl *BackupLogger) LogBackupDeletion(ctx context.Context, backupID string, reason string) func(error) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     "backup_delete",
		BackupID:      backupID,
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"reason": reason,
		},
	}

	bl.logStructured(entry)

	return func(err error) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		bl.logStructured(entry)

		result := "success"
		if err != nil {
			result = "failure"
		}

		bl.logAudit(ctx, "backup", "delete", result, map[string]interface{}{
			"backup_id": backupID,
			"reason":    reason,
			"duration":  duration.String(),
		})
	}
}

// LogRollbackStart logs the start of a rollback operation
func (bl *BackupLogger) LogRollbackStart(ctx context.Context, backupID, databaseName string, plan *RollbackPlan) func(error) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     "rollback_execute",
		BackupID:      backupID,
		DatabaseName:  databaseName,
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"statement_count": len(plan.Statements),
			"warning_count":   len(plan.Warnings),
		},
	}

	bl.logStructured(entry)
	bl.logAudit(ctx, "rollback", "execute", "started", map[string]interface{}{
		"backup_id":     backupID,
		"database_name": databaseName,
	})

	return func(err error) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		bl.logStructured(entry)

		result := "success"
		if err != nil {
			result = "failure"
		}

		bl.logAudit(ctx, "rollback", "execute", result, map[string]interface{}{
			"backup_id":     backupID,
			"database_name": databaseName,
			"duration":      duration.String(),
			"error":         entry.Error,
		})
	}
}

// LogRetentionCleanup logs retention policy cleanup operations
func (bl *BackupLogger) LogRetentionCleanup(ctx context.Context, databaseName string, policy map[string]interface{}) func(error, []string) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     "retention_cleanup",
		DatabaseName:  databaseName,
		Status:        "started",
		Success:       true,
		Metadata:      policy,
	}

	bl.logStructured(entry)

	return func(err error, deletedBackups []string) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		entry.Metadata["deleted_count"] = len(deletedBackups)
		entry.Metadata["deleted_backups"] = deletedBackups

		bl.logStructured(entry)

		result := "success"
		if err != nil {
			result = "failure"
		}

		bl.logAudit(ctx, "retention", "cleanup", result, map[string]interface{}{
			"database_name":   databaseName,
			"deleted_count":   len(deletedBackups),
			"deleted_backups": deletedBackups,
			"duration":        duration.String(),
		})
	}
}

// LogStorageOperation logs storage provider operations
func (bl *BackupLogger) LogStorageOperation(ctx context.Context, operation, provider, backupID string) func(error, map[string]interface{}) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     fmt.Sprintf("storage_%s", operation),
		BackupID:      backupID,
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"storage_provider": provider,
		},
	}

	bl.logStructured(entry)

	return func(err error, metadata map[string]interface{}) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		// Merge additional metadata
		for k, v := range metadata {
			entry.Metadata[k] = v
		}

		bl.logStructured(entry)
	}
}

// LogCompressionOperation logs compression/decompression operations
func (bl *BackupLogger) LogCompressionOperation(ctx context.Context, operation string, compressionType CompressionType, originalSize int64) func(error, int64) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     fmt.Sprintf("compression_%s", operation),
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"compression_type": string(compressionType),
			"original_size":    originalSize,
		},
	}

	bl.logStructured(entry)

	return func(err error, finalSize int64) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		entry.Metadata["final_size"] = finalSize
		if originalSize > 0 && finalSize > 0 {
			ratio := float64(finalSize) / float64(originalSize)
			entry.Metadata["compression_ratio"] = fmt.Sprintf("%.2f", ratio)
		}

		bl.logStructured(entry)
	}
}

// LogEncryptionOperation logs encryption/decryption operations
func (bl *BackupLogger) LogEncryptionOperation(ctx context.Context, operation string, dataSize int64) func(error) {
	startTime := time.Now()

	entry := LogEntry{
		Timestamp:     startTime,
		CorrelationID: bl.correlationID,
		Operation:     fmt.Sprintf("encryption_%s", operation),
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"data_size": dataSize,
		},
	}

	bl.logStructured(entry)

	return func(err error) {
		duration := time.Since(startTime)
		entry.Timestamp = time.Now()
		entry.Status = "completed"
		entry.Duration = duration.String()
		entry.Success = err == nil

		if err != nil {
			entry.Error = err.Error()
			entry.Status = "failed"
		}

		bl.logStructured(entry)
	}
}

// logStructured logs a structured log entry
func (bl *BackupLogger) logStructured(entry LogEntry) {
	fields := logrus.Fields{
		"correlation_id": entry.CorrelationID,
		"operation":      entry.Operation,
		"status":         entry.Status,
		"success":        entry.Success,
	}

	if entry.BackupID != "" {
		fields["backup_id"] = entry.BackupID
	}
	if entry.DatabaseName != "" {
		fields["database_name"] = entry.DatabaseName
	}
	if entry.Duration != "" {
		fields["duration"] = entry.Duration
	}
	if entry.Error != "" {
		fields["error"] = entry.Error
	}

	// Add metadata fields
	for k, v := range entry.Metadata {
		fields[k] = v
	}

	logEntry := bl.logger.WithFields(fields)

	if entry.Success {
		if entry.Status == "started" {
			logEntry.Debug("Backup operation started")
		} else {
			logEntry.Info("Backup operation completed successfully")
		}
	} else {
		logEntry.Error("Backup operation failed")
	}
}

// logAudit logs an audit trail entry
func (bl *BackupLogger) logAudit(ctx context.Context, resource, action, result string, details map[string]interface{}) {
	if bl.auditLogger == nil {
		return
	}

	entry := AuditLogEntry{
		Timestamp:     time.Now(),
		CorrelationID: bl.correlationID,
		Operation:     fmt.Sprintf("%s_%s", resource, action),
		Resource:      resource,
		Action:        action,
		Result:        result,
		Details:       details,
	}

	// Extract user information from context if available
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			entry.UserID = id
		}
	}

	if ipAddr := ctx.Value("ip_address"); ipAddr != nil {
		if addr, ok := ipAddr.(string); ok {
			entry.IPAddress = addr
		}
	}

	bl.auditLogger.WithFields(logrus.Fields{
		"correlation_id": entry.CorrelationID,
		"user_id":        entry.UserID,
		"operation":      entry.Operation,
		"resource":       entry.Resource,
		"action":         entry.Action,
		"result":         entry.Result,
		"details":        entry.Details,
		"ip_address":     entry.IPAddress,
	}).Info("Audit log entry")
}

// ExportLogs exports backup logs in the specified format
func (bl *BackupLogger) ExportLogs(ctx context.Context, startTime, endTime time.Time, format string, outputPath string) error {
	// This is a placeholder for log export functionality
	// In a real implementation, this would query log storage and export in the requested format

	entry := LogEntry{
		Timestamp:     time.Now(),
		CorrelationID: bl.correlationID,
		Operation:     "log_export",
		Status:        "started",
		Success:       true,
		Metadata: map[string]interface{}{
			"start_time":  startTime.Format(time.RFC3339),
			"end_time":    endTime.Format(time.RFC3339),
			"format":      format,
			"output_path": outputPath,
		},
	}

	bl.logStructured(entry)

	// Simulate export operation
	switch format {
	case "json":
		return bl.exportLogsJSON(startTime, endTime, outputPath)
	case "csv":
		return bl.exportLogsCSV(startTime, endTime, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportLogsJSON exports logs in JSON format
func (bl *BackupLogger) exportLogsJSON(startTime, endTime time.Time, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write JSON header
	file.WriteString("[\n")

	// In a real implementation, this would query actual log data
	// For now, we'll write a sample entry to demonstrate the format
	sampleEntry := LogEntry{
		Timestamp:     time.Now(),
		CorrelationID: bl.correlationID,
		Operation:     "sample_export",
		Status:        "completed",
		Success:       true,
		Metadata: map[string]interface{}{
			"export_format": "json",
			"time_range":    fmt.Sprintf("%s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)),
		},
	}

	entryJSON, err := json.MarshalIndent(sampleEntry, "  ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	file.Write(entryJSON)
	file.WriteString("\n]")

	return nil
}

// exportLogsCSV exports logs in CSV format
func (bl *BackupLogger) exportLogsCSV(startTime, endTime time.Time, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write CSV header
	file.WriteString("timestamp,correlation_id,operation,backup_id,database_name,status,duration,success,error\n")

	// In a real implementation, this would query actual log data
	// For now, we'll write a sample entry to demonstrate the format
	file.WriteString(fmt.Sprintf("%s,%s,sample_export,,,completed,,true,\n",
		time.Now().Format(time.RFC3339), bl.correlationID))

	return nil
}
