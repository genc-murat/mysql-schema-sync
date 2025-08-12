package database

import (
	"errors"
	"fmt"
	"time"
)

// DatabaseConfig holds the configuration parameters for database connection
type DatabaseConfig struct {
	Host     string        `mapstructure:"host" yaml:"host"`
	Port     int           `mapstructure:"port" yaml:"port"`
	Username string        `mapstructure:"username" yaml:"username"`
	Password string        `mapstructure:"password" yaml:"password"`
	Database string        `mapstructure:"database" yaml:"database"`
	Timeout  time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

// CLIConfig holds the complete CLI configuration including source and target databases
type CLIConfig struct {
	SourceDB    DatabaseConfig `mapstructure:"source" yaml:"source"`
	TargetDB    DatabaseConfig `mapstructure:"target" yaml:"target"`
	DryRun      bool           `mapstructure:"dry_run" yaml:"dry_run"`
	Verbose     bool           `mapstructure:"verbose" yaml:"verbose"`
	AutoApprove bool           `mapstructure:"auto_approve" yaml:"auto_approve"`
}

// Validate checks if the database configuration has all required parameters
func (dc *DatabaseConfig) Validate() error {
	var errs []error

	if dc.Host == "" {
		errs = append(errs, errors.New("host is required"))
	}

	if dc.Port <= 0 || dc.Port > 65535 {
		errs = append(errs, errors.New("port must be between 1 and 65535"))
	}

	if dc.Username == "" {
		errs = append(errs, errors.New("username is required"))
	}

	if dc.Database == "" {
		errs = append(errs, errors.New("database name is required"))
	}

	if dc.Timeout <= 0 {
		dc.Timeout = 30 * time.Second // Set default timeout
	}

	if len(errs) > 0 {
		return fmt.Errorf("database configuration validation failed: %v", errs)
	}

	return nil
}

// DSN returns the Data Source Name for MySQL connection
func (dc *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true",
		dc.Username, dc.Password, dc.Host, dc.Port, dc.Database, dc.Timeout)
}

// Validate checks if the CLI configuration is valid
func (cc *CLIConfig) Validate() error {
	if err := cc.SourceDB.Validate(); err != nil {
		return fmt.Errorf("source database: %w", err)
	}

	if err := cc.TargetDB.Validate(); err != nil {
		return fmt.Errorf("target database: %w", err)
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (cc *CLIConfig) SetDefaults() {
	// Set default port if not specified
	if cc.SourceDB.Port == 0 {
		cc.SourceDB.Port = 3306
	}
	if cc.TargetDB.Port == 0 {
		cc.TargetDB.Port = 3306
	}

	// Set default timeout if not specified
	if cc.SourceDB.Timeout == 0 {
		cc.SourceDB.Timeout = 30 * time.Second
	}
	if cc.TargetDB.Timeout == 0 {
		cc.TargetDB.Timeout = 30 * time.Second
	}
}
