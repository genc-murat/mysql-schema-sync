package application

import (
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/execution"
	"mysql-schema-sync/internal/logging"
)

func TestNewApplication(t *testing.T) {
	config := Config{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		DryRun:      true,
		AutoApprove: false,
		Verbose:     false,
		Quiet:       false,
		Timeout:     30 * time.Second,
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	if app == nil {
		t.Error("NewApplication() returned nil")
	}

	if app.executor == nil {
		t.Error("Expected executor to be initialized")
	}

	if app.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if app.shutdownHandler == nil {
		t.Error("Expected shutdownHandler to be initialized")
	}
}

func TestNewApplication_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		quiet    bool
		expected logging.LogLevel
	}{
		{
			name:     "normal level",
			verbose:  false,
			quiet:    false,
			expected: logging.LogLevelNormal,
		},
		{
			name:     "verbose level",
			verbose:  true,
			quiet:    false,
			expected: logging.LogLevelVerbose,
		},
		{
			name:     "quiet level",
			verbose:  false,
			quiet:    true,
			expected: logging.LogLevelQuiet,
		},
		{
			name:     "quiet takes precedence",
			verbose:  true,
			quiet:    true,
			expected: logging.LogLevelQuiet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				SourceDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "source_db",
					Username: "user",
					Password: "pass",
				},
				TargetDB: database.DatabaseConfig{
					Host:     "localhost",
					Database: "target_db",
					Username: "user",
					Password: "pass",
				},
				Verbose: tt.verbose,
				Quiet:   tt.quiet,
			}

			app, err := NewApplication(config)
			if err != nil {
				t.Fatalf("NewApplication() error = %v", err)
			}

			if app.logger.GetLevel() != tt.expected {
				t.Errorf("Expected log level %v, got %v", tt.expected, app.logger.GetLevel())
			}
		})
	}
}

func TestNewApplication_InvalidConfig(t *testing.T) {
	config := Config{
		SourceDB: database.DatabaseConfig{
			// Missing host
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
	}

	app, err := NewApplication(config)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}

	if app != nil {
		t.Error("Expected nil application for invalid config")
	}
}

func TestApplication_GetLogger(t *testing.T) {
	config := Config{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	logger := app.GetLogger()
	if logger == nil {
		t.Error("GetLogger() returned nil")
	}

	if logger != app.logger {
		t.Error("GetLogger() returned different logger instance")
	}
}

func TestApplication_GetStatusString(t *testing.T) {
	config := Config{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	tests := []struct {
		name    string
		success bool
		want    string
	}{
		{
			name:    "success",
			success: true,
			want:    "SUCCESS",
		},
		{
			name:    "failure",
			success: false,
			want:    "FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.getStatusString(tt.success)
			if result != tt.want {
				t.Errorf("getStatusString() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestApplication_DisplayResults(t *testing.T) {
	config := Config{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "source_db",
			Username: "user",
			Password: "pass",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Database: "target_db",
			Username: "user",
			Password: "pass",
		},
		Verbose: true, // Enable verbose for more detailed output
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	// Test with nil result
	app.displayResults(nil) // Should not panic

	// Test with empty result
	result := &execution.ExecutionResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
	}
	app.displayResults(result) // Should not panic

	// Test with result containing data
	result = &execution.ExecutionResult{
		Success:            true,
		ExecutedStatements: []string{"CREATE TABLE test (id INT)"},
		Warnings:           []string{"Warning: potential data loss"},
		Duration:           200 * time.Millisecond,
	}
	app.displayResults(result) // Should not panic
}

// Integration test that would require actual execution
// This is commented out as it requires real database setup
/*
func TestApplication_Run_Integration(t *testing.T) {
	// This test would require actual MySQL databases
	// Skip in unit tests
	t.Skip("Integration test requires actual MySQL databases")

	config := Config{
		SourceDB: database.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "test_source",
			Username: "root",
			Password: "password",
		},
		TargetDB: database.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "test_target",
			Username: "root",
			Password: "password",
		},
		DryRun:      true,
		AutoApprove: true,
		Verbose:     true,
		Timeout:     30 * time.Second,
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	err = app.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
*/
