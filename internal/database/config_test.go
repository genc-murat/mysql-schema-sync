package database

import (
	"testing"
	"time"
)

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				Database: "testdb",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: DatabaseConfig{
				Port:     3306,
				Username: "root",
				Password: "password",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     0,
				Username: "root",
				Password: "password",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Password: "password",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing database",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
		Timeout:  30 * time.Second,
	}

	expected := "root:password@tcp(localhost:3306)/testdb?timeout=30s&parseTime=true"
	actual := config.DSN()

	if actual != expected {
		t.Errorf("DatabaseConfig.DSN() = %v, want %v", actual, expected)
	}
}

func TestCLIConfig_SetDefaults(t *testing.T) {
	config := &CLIConfig{}
	config.SetDefaults()

	if config.SourceDB.Port != 3306 {
		t.Errorf("Expected source port to be 3306, got %d", config.SourceDB.Port)
	}

	if config.TargetDB.Port != 3306 {
		t.Errorf("Expected target port to be 3306, got %d", config.TargetDB.Port)
	}

	if config.SourceDB.Timeout != 30*time.Second {
		t.Errorf("Expected source timeout to be 30s, got %v", config.SourceDB.Timeout)
	}

	if config.TargetDB.Timeout != 30*time.Second {
		t.Errorf("Expected target timeout to be 30s, got %v", config.TargetDB.Timeout)
	}
}
