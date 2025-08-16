package backup

import (
	"os"
	"time"
)

// Backup represents a complete database schema backup
type Backup struct {
	ID              string                `json:"id"`
	Metadata        *BackupMetadata       `json:"metadata"`
	SchemaSnapshot  interface{}           `json:"schema_snapshot"` // TODO: Define proper schema interface
	DataDefinitions []string              `json:"data_definitions"`
	Triggers        []TriggerDefinition   `json:"triggers"`
	Views           []ViewDefinition      `json:"views"`
	Procedures      []ProcedureDefinition `json:"procedures"`
	Functions       []FunctionDefinition  `json:"functions"`
	Checksum        string                `json:"checksum"`
}

// BackupMetadata contains metadata about a backup
type BackupMetadata struct {
	ID                string            `json:"id"`
	DatabaseName      string            `json:"database_name"`
	CreatedAt         time.Time         `json:"created_at"`
	CreatedBy         string            `json:"created_by"`
	Description       string            `json:"description"`
	Tags              map[string]string `json:"tags"`
	Size              int64             `json:"size"`
	CompressedSize    int64             `json:"compressed_size"`
	CompressionType   CompressionType   `json:"compression_type"`
	EncryptionEnabled bool              `json:"encryption_enabled"`
	MigrationContext  *MigrationContext `json:"migration_context,omitempty"`
	StorageLocation   string            `json:"storage_location"`
	Checksum          string            `json:"checksum"`
	Status            BackupStatus      `json:"status"`
}

// MigrationContext provides context about the migration that triggered the backup
type MigrationContext struct {
	PlanHash       string    `json:"plan_hash"`
	SourceSchema   string    `json:"source_schema"`
	PreMigrationID string    `json:"pre_migration_id"`
	MigrationTime  time.Time `json:"migration_time"`
	ToolVersion    string    `json:"tool_version"`
}

// BackupConfig contains configuration for backup creation
type BackupConfig struct {
	DatabaseConfig  DatabaseConfig // TODO: Move to local type when dependency is resolved
	StorageConfig   StorageConfig
	CompressionType CompressionType
	EncryptionKey   []byte
	Description     string
	Tags            map[string]string
}

// StorageConfig defines storage provider configuration
type StorageConfig struct {
	Provider StorageProviderType `yaml:"provider"`
	Local    *LocalConfig        `yaml:"local,omitempty"`
	S3       *S3Config           `yaml:"s3,omitempty"`
	Azure    *AzureConfig        `yaml:"azure,omitempty"`
	GCS      *GCSConfig          `yaml:"gcs,omitempty"`
}

// LocalConfig for local file system storage
type LocalConfig struct {
	BasePath    string      `yaml:"base_path"`
	Permissions os.FileMode `yaml:"permissions"`
}

// S3Config for Amazon S3 storage
type S3Config struct {
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

// AzureConfig for Azure Blob Storage
type AzureConfig struct {
	AccountName   string `yaml:"account_name"`
	AccountKey    string `yaml:"account_key"`
	ContainerName string `yaml:"container_name"`
}

// GCSConfig for Google Cloud Storage
type GCSConfig struct {
	Bucket          string `yaml:"bucket"`
	CredentialsPath string `yaml:"credentials_path"`
	ProjectID       string `yaml:"project_id"`
}

// BackupFilter for filtering backup lists
type BackupFilter struct {
	DatabaseName  string
	StartDate     *time.Time
	EndDate       *time.Time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Tags          map[string]string
	Status        *BackupStatus
}

// StorageFilter for filtering storage operations
type StorageFilter struct {
	Prefix   string
	MaxItems int
}

// ValidationResult contains backup validation results
type ValidationResult struct {
	Valid         bool      `json:"valid"`
	Errors        []string  `json:"errors,omitempty"`
	Warnings      []string  `json:"warnings,omitempty"`
	CheckedAt     time.Time `json:"checked_at"`
	ChecksumValid bool      `json:"checksum_valid"`
}

// RollbackPoint represents a point in time that can be rolled back to
type RollbackPoint struct {
	BackupID     string            `json:"backup_id"`
	DatabaseName string            `json:"database_name"`
	CreatedAt    time.Time         `json:"created_at"`
	Description  string            `json:"description"`
	SchemaHash   string            `json:"schema_hash"`
	Tags         map[string]string `json:"tags"`
}

// RollbackPlan contains the plan for executing a rollback
type RollbackPlan struct {
	BackupID      string        `json:"backup_id"`
	TargetSchema  interface{}   `json:"target_schema"`  // TODO: Define proper schema interface
	CurrentSchema interface{}   `json:"current_schema"` // TODO: Define proper schema interface
	Statements    []interface{} `json:"statements"`     // TODO: Define proper migration statement interface
	Dependencies  []string      `json:"dependencies"`
	Warnings      []string      `json:"warnings"`
}

// RollbackRecoveryPlan provides guidance for manual intervention when rollback fails
type RollbackRecoveryPlan struct {
	OriginalPlan         *RollbackPlan `json:"original_plan"`
	FailedStatementIndex int           `json:"failed_statement_index"`
	FailureReason        string        `json:"failure_reason"`
	RecoverySteps        []string      `json:"recovery_steps"`
	ManualInterventions  []string      `json:"manual_interventions"`
}

// Database object definitions
type TriggerDefinition struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	Timing     string `json:"timing"`
	Event      string `json:"event"`
	Definition string `json:"definition"`
}

type ViewDefinition struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type ProcedureDefinition struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type FunctionDefinition struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Enums and constants
type BackupStatus string

const (
	BackupStatusCreating   BackupStatus = "CREATING"
	BackupStatusCompleted  BackupStatus = "COMPLETED"
	BackupStatusFailed     BackupStatus = "FAILED"
	BackupStatusValidating BackupStatus = "VALIDATING"
	BackupStatusCorrupted  BackupStatus = "CORRUPTED"
)

type CompressionType string

const (
	CompressionTypeNone CompressionType = "NONE"
	CompressionTypeGzip CompressionType = "GZIP"
	CompressionTypeLZ4  CompressionType = "LZ4"
	CompressionTypeZstd CompressionType = "ZSTD"
)

type StorageProviderType string

const (
	StorageProviderLocal StorageProviderType = "LOCAL"
	StorageProviderS3    StorageProviderType = "S3"
	StorageProviderAzure StorageProviderType = "AZURE"
	StorageProviderGCS   StorageProviderType = "GCS"
)

// DatabaseConfig represents database connection configuration
// TODO: This is a temporary local type to avoid import cycle
type DatabaseConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}
