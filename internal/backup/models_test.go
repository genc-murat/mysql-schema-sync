package backup

import (
	"testing"
	"time"

	"mysql-schema-sync/internal/schema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backup  *Backup
		wantErr bool
		errType string
	}{
		{
			name: "valid backup",
			backup: &Backup{
				ID: "backup-20240101-123456-abc12345",
				Metadata: &BackupMetadata{
					ID:              "backup-20240101-123456-abc12345",
					DatabaseName:    "test_db",
					CreatedAt:       time.Now(),
					CreatedBy:       "test_user",
					Status:          BackupStatusCompleted,
					StorageLocation: "/tmp/backups",
					Checksum:        "abc123",
				},
				SchemaSnapshot: &schema.Schema{
					Name:   "test_db",
					Tables: make(map[string]*schema.Table),
				},
				Checksum: "def456",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			backup: &Backup{
				Metadata: &BackupMetadata{
					ID:              "backup-20240101-123456-abc12345",
					DatabaseName:    "test_db",
					CreatedAt:       time.Now(),
					CreatedBy:       "test_user",
					Status:          BackupStatusCompleted,
					StorageLocation: "/tmp/backups",
					Checksum:        "abc123",
				},
				SchemaSnapshot: &schema.Schema{
					Name:   "test_db",
					Tables: make(map[string]*schema.Table),
				},
				Checksum: "def456",
			},
			wantErr: true,
			errType: "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backup.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != "" {
					validationErrs, ok := err.(ValidationErrors)
					require.True(t, ok, "expected ValidationErrors")
					found := false
					for _, validationErr := range validationErrs {
						if validationErr.Field == tt.errType {
							found = true
							break
						}
					}
					assert.True(t, found, "expected validation error for field: %s", tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGenerateBackupID(t *testing.T) {
	id1 := GenerateBackupID()
	id2 := GenerateBackupID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "backup-")
	assert.Contains(t, id2, "backup-")

	// Test with custom prefix
	customID := GenerateBackupIDWithPrefix("manual")
	assert.Contains(t, customID, "manual-")
	assert.NotContains(t, customID, "backup-")
}

func TestCalculateDataChecksum(t *testing.T) {
	data1 := []byte("test data 1")
	data2 := []byte("test data 2")

	checksum1 := CalculateDataChecksum(data1)
	checksum2 := CalculateDataChecksum(data2)

	assert.NotEmpty(t, checksum1)
	assert.NotEmpty(t, checksum2)
	assert.NotEqual(t, checksum1, checksum2)

	// Same data should produce same checksum
	checksum1Again := CalculateDataChecksum(data1)
	assert.Equal(t, checksum1, checksum1Again)
}
