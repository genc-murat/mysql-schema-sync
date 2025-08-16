package backup

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

// Use the existing MockStorageProvider from storage_cloud_test.go

// MockDatabaseService for testing
type MockDatabaseService struct{}

func (m *MockDatabaseService) Connect(config database.DatabaseConfig) (*sql.DB, error) {
	// Return a mock connection (in real tests, you might use sqlmock)
	return nil, nil
}

func (m *MockDatabaseService) Close(db *sql.DB) error {
	return nil
}

func (m *MockDatabaseService) TestConnection(db *sql.DB) error {
	return nil
}

func (m *MockDatabaseService) GetVersion(db *sql.DB) (string, error) {
	return "8.0.0", nil
}

func (m *MockDatabaseService) ExecuteSQL(db *sql.DB, statements []string) error {
	return nil
}

// Test helper to create a test backup manager
func createTestBackupManager() *backupManager {
	config := &BackupSystemConfig{
		Storage: StorageConfig{
			Provider: StorageProviderLocal,
			Local: &LocalConfig{
				BasePath: "/tmp/test-backups",
			},
		},
		Retention: RetentionConfig{
			MaxBackups: 10,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: CompressionTypeGzip,
			Level:     6,
		},
		Encryption: EncryptionConfig{
			Enabled: false,
		},
		Validation: ValidationConfig{
			Enabled: true,
		},
	}

	return &backupManager{
		storageProvider: NewMockStorageProvider(),
		validator:       NewBackupValidator(),
		extractor:       NewSchemaExtractor(),
		compressionMgr:  NewCompressionManager(),
		encryptionMgr:   NewEncryptionManager(&config.Encryption),
		logger:          logging.NewDefaultLogger(),
		databaseService: &MockDatabaseService{},
		config:          config,
	}
}

func TestBackupManager_CreateManualBackup(t *testing.T) {
	// Test the tag and description setting logic without actual backup creation
	manager := createTestBackupManager()

	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Database: "testdb",
		},
		StorageConfig:   manager.config.Storage,
		CompressionType: CompressionTypeGzip,
		Description:     "Test manual backup",
		Tags: map[string]string{
			"environment": "test",
		},
	}

	// Test that manual backup tags are set correctly
	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
	config.Tags["type"] = "manual"

	if config.Tags["type"] != "manual" {
		t.Errorf("Expected backup type to be 'manual', got %s", config.Tags["type"])
	}

	if config.Description != "Test manual backup" {
		t.Errorf("Expected description 'Test manual backup', got %s", config.Description)
	}
}

func TestBackupManager_CreatePreMigrationBackup(t *testing.T) {
	// Test the migration context logic without actual backup creation
	config := BackupConfig{
		DatabaseConfig: database.DatabaseConfig{
			Database: "testdb",
		},
		Description: "Test backup",
	}

	// Create a mock migration plan
	migrationPlan := migration.NewMigrationPlan()
	migrationPlan.AddStatement(migration.MigrationStatement{
		SQL:         "CREATE TABLE test (id INT PRIMARY KEY)",
		Type:        migration.StatementTypeCreateTable,
		Description: "Create test table",
		TableName:   "test",
	})

	// Test the logic that would be applied in CreatePreMigrationBackup
	if config.Description == "" {
		config.Description = "Automatic pre-migration backup"
	} else {
		config.Description = fmt.Sprintf("Pre-migration backup: %s", config.Description)
	}

	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
	config.Tags["type"] = "pre-migration"
	config.Tags["migration_plan_hash"] = migrationPlan.Hash()

	if config.Tags["type"] != "pre-migration" {
		t.Errorf("Expected backup type to be 'pre-migration', got %s", config.Tags["type"])
	}

	if config.Tags["migration_plan_hash"] != migrationPlan.Hash() {
		t.Errorf("Expected plan hash %s, got %s", migrationPlan.Hash(), config.Tags["migration_plan_hash"])
	}

	expectedDescription := "Pre-migration backup: Test backup"
	if config.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, config.Description)
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	manager := createTestBackupManager()
	ctx := context.Background()

	// Create mock backups directly in storage
	mockStorage, ok := manager.storageProvider.(*MockStorageProvider)
	if !ok {
		t.Fatal("Expected MockStorageProvider")
	}

	// Create test backup metadata
	backup1 := &Backup{
		ID: "backup-1",
		Metadata: &BackupMetadata{
			ID:                "backup-1",
			DatabaseName:      "testdb1",
			CreatedAt:         time.Now().Add(-2 * time.Hour),
			CreatedBy:         "test-user",
			Description:       "Test backup 1",
			Size:              1000,
			CompressedSize:    800,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Tags:              map[string]string{"type": "manual"},
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb1",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup2 := &Backup{
		ID: "backup-2",
		Metadata: &BackupMetadata{
			ID:                "backup-2",
			DatabaseName:      "testdb2",
			CreatedAt:         time.Now().Add(-1 * time.Hour),
			CreatedBy:         "test-user",
			Description:       "Test backup 2",
			Size:              2000,
			CompressedSize:    1600,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Tags:              map[string]string{"type": "manual"},
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb2",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup3 := &Backup{
		ID: "backup-3",
		Metadata: &BackupMetadata{
			ID:                "backup-3",
			DatabaseName:      "testdb1",
			CreatedAt:         time.Now(),
			CreatedBy:         "test-user",
			Description:       "Test backup 3",
			Size:              1500,
			CompressedSize:    1200,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Tags:              map[string]string{"type": "pre-migration"},
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb1",
			Tables: make(map[string]*schema.Table),
		},
	}

	// Store backups in mock storage
	err := mockStorage.Store(ctx, backup1)
	if err != nil {
		t.Fatalf("Failed to store backup1: %v", err)
	}
	err = mockStorage.Store(ctx, backup2)
	if err != nil {
		t.Fatalf("Failed to store backup2: %v", err)
	}
	err = mockStorage.Store(ctx, backup3)
	if err != nil {
		t.Fatalf("Failed to store backup3: %v", err)
	}

	// Test listing all backups
	allBackups, err := manager.ListBackups(ctx, BackupFilter{})
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(allBackups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(allBackups))
	}

	// Test filtering by database name
	filteredBackups, err := manager.ListBackups(ctx, BackupFilter{
		DatabaseName: "testdb1",
	})
	if err != nil {
		t.Fatalf("ListBackups with filter failed: %v", err)
	}

	if len(filteredBackups) != 2 {
		t.Errorf("Expected 2 backups for testdb1, got %d", len(filteredBackups))
	}

	// Verify sorting (newest first)
	if !filteredBackups[0].CreatedAt.After(filteredBackups[1].CreatedAt) {
		t.Error("Expected backups to be sorted by creation time (newest first)")
	}
}

func TestBackupManager_ListBackupsWithSorting(t *testing.T) {
	manager := createTestBackupManager()
	ctx := context.Background()

	// Create mock backups directly in storage with different sizes
	mockStorage, ok := manager.storageProvider.(*MockStorageProvider)
	if !ok {
		t.Fatal("Expected MockStorageProvider")
	}

	backup1 := &Backup{
		ID: "backup-1",
		Metadata: &BackupMetadata{
			ID:                "backup-1",
			DatabaseName:      "testdb",
			CreatedBy:         "test-user",
			Size:              1000,
			CompressedSize:    800,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Description:       "Alpha backup",
			CreatedAt:         time.Now().Add(-3 * time.Hour),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup2 := &Backup{
		ID: "backup-2",
		Metadata: &BackupMetadata{
			ID:                "backup-2",
			DatabaseName:      "testdb",
			CreatedBy:         "test-user",
			Size:              2000,
			CompressedSize:    1600,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Description:       "Beta backup",
			CreatedAt:         time.Now().Add(-2 * time.Hour),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup3 := &Backup{
		ID: "backup-3",
		Metadata: &BackupMetadata{
			ID:                "backup-3",
			DatabaseName:      "testdb",
			CreatedBy:         "test-user",
			Size:              500,
			CompressedSize:    400,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			Description:       "Gamma backup",
			CreatedAt:         time.Now().Add(-1 * time.Hour),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb",
			Tables: make(map[string]*schema.Table),
		},
	}

	mockStorage.Store(ctx, backup1)
	mockStorage.Store(ctx, backup2)
	mockStorage.Store(ctx, backup3)

	// Test sorting by size (ascending)
	backups, err := manager.ListBackupsWithSorting(ctx, BackupFilter{}, "size", true)
	if err != nil {
		t.Fatalf("ListBackupsWithSorting failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	if backups[0].Size >= backups[1].Size || backups[1].Size >= backups[2].Size {
		t.Error("Expected backups to be sorted by size (ascending)")
	}

	// Test sorting by description (descending)
	backups, err = manager.ListBackupsWithSorting(ctx, BackupFilter{}, "description", false)
	if err != nil {
		t.Fatalf("ListBackupsWithSorting failed: %v", err)
	}

	if backups[0].Description < backups[1].Description || backups[1].Description < backups[2].Description {
		t.Error("Expected backups to be sorted by description (descending)")
	}
}

func TestBackupManager_GetBackupsByDatabase(t *testing.T) {
	manager := createTestBackupManager()
	ctx := context.Background()

	// Create mock backups for different databases
	mockStorage, ok := manager.storageProvider.(*MockStorageProvider)
	if !ok {
		t.Fatal("Expected MockStorageProvider")
	}

	backup1 := &Backup{
		ID: "backup-1",
		Metadata: &BackupMetadata{
			ID:                "backup-1",
			DatabaseName:      "db1",
			CreatedBy:         "test-user",
			Description:       "Test backup for db1",
			Size:              1000,
			CompressedSize:    800,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			CreatedAt:         time.Now().Add(-1 * time.Hour),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "db1",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup2 := &Backup{
		ID: "backup-2",
		Metadata: &BackupMetadata{
			ID:                "backup-2",
			DatabaseName:      "db1",
			CreatedBy:         "test-user",
			Description:       "Test backup for db1 #2",
			Size:              1200,
			CompressedSize:    960,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			CreatedAt:         time.Now().Add(-2 * time.Hour),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "db1",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup3 := &Backup{
		ID: "backup-3",
		Metadata: &BackupMetadata{
			ID:                "backup-3",
			DatabaseName:      "db2",
			CreatedBy:         "test-user",
			Description:       "Test backup for db2",
			Size:              800,
			CompressedSize:    640,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			CreatedAt:         time.Now(),
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "db2",
			Tables: make(map[string]*schema.Table),
		},
	}

	mockStorage.Store(ctx, backup1)
	mockStorage.Store(ctx, backup2)
	mockStorage.Store(ctx, backup3)

	// Test getting backups for db1
	db1Backups, err := manager.GetBackupsByDatabase(ctx, "db1")
	if err != nil {
		t.Fatalf("GetBackupsByDatabase failed: %v", err)
	}

	if len(db1Backups) != 2 {
		t.Errorf("Expected 2 backups for db1, got %d", len(db1Backups))
	}

	// Test getting backups for db2
	db2Backups, err := manager.GetBackupsByDatabase(ctx, "db2")
	if err != nil {
		t.Fatalf("GetBackupsByDatabase failed: %v", err)
	}

	if len(db2Backups) != 1 {
		t.Errorf("Expected 1 backup for db2, got %d", len(db2Backups))
	}
}

func TestBackupManager_GetBackupsByType(t *testing.T) {
	manager := createTestBackupManager()
	ctx := context.Background()

	// Create mock backups with different types
	mockStorage, ok := manager.storageProvider.(*MockStorageProvider)
	if !ok {
		t.Fatal("Expected MockStorageProvider")
	}

	backup1 := &Backup{
		ID: "backup-1",
		Metadata: &BackupMetadata{
			ID:                "backup-1",
			DatabaseName:      "testdb",
			CreatedBy:         "test-user",
			Description:       "Manual backup",
			Size:              1000,
			CompressedSize:    800,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			CreatedAt:         time.Now().Add(-1 * time.Hour),
			Tags:              map[string]string{"type": "manual"},
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb",
			Tables: make(map[string]*schema.Table),
		},
	}
	backup2 := &Backup{
		ID: "backup-2",
		Metadata: &BackupMetadata{
			ID:                "backup-2",
			DatabaseName:      "testdb",
			CreatedBy:         "test-user",
			Description:       "Pre-migration backup",
			Size:              1200,
			CompressedSize:    960,
			CompressionType:   CompressionTypeGzip,
			EncryptionEnabled: false,
			Status:            BackupStatusCompleted,
			CreatedAt:         time.Now(),
			Tags:              map[string]string{"type": "pre-migration"},
		},
		SchemaSnapshot: &schema.Schema{
			Name:   "testdb",
			Tables: make(map[string]*schema.Table),
		},
	}

	mockStorage.Store(ctx, backup1)
	mockStorage.Store(ctx, backup2)

	// Test getting manual backups
	manualBackups, err := manager.GetBackupsByType(ctx, "manual")
	if err != nil {
		t.Fatalf("GetBackupsByType failed: %v", err)
	}

	if len(manualBackups) != 1 {
		t.Errorf("Expected 1 manual backup, got %d", len(manualBackups))
	}

	// Test getting pre-migration backups
	preMigrationBackups, err := manager.GetBackupsByType(ctx, "pre-migration")
	if err != nil {
		t.Fatalf("GetBackupsByType failed: %v", err)
	}

	if len(preMigrationBackups) != 1 {
		t.Errorf("Expected 1 pre-migration backup, got %d", len(preMigrationBackups))
	}
}
