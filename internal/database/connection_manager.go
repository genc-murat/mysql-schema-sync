package database

import (
	"database/sql"
	"fmt"
)

// ConnectionManager manages database connections for source and target databases
type ConnectionManager struct {
	service  DatabaseService
	sourceDB *sql.DB
	targetDB *sql.DB
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		service: NewService(),
	}
}

// NewConnectionManagerWithService creates a new connection manager with a custom service
func NewConnectionManagerWithService(service DatabaseService) *ConnectionManager {
	return &ConnectionManager{
		service: service,
	}
}

// ConnectToSource establishes connection to the source database
func (cm *ConnectionManager) ConnectToSource(config DatabaseConfig) error {
	if cm.sourceDB != nil {
		cm.service.Close(cm.sourceDB)
	}

	db, err := cm.service.Connect(config)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}

	cm.sourceDB = db
	return nil
}

// ConnectToTarget establishes connection to the target database
func (cm *ConnectionManager) ConnectToTarget(config DatabaseConfig) error {
	if cm.targetDB != nil {
		cm.service.Close(cm.targetDB)
	}

	db, err := cm.service.Connect(config)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}

	cm.targetDB = db
	return nil
}

// GetSourceDB returns the source database connection
func (cm *ConnectionManager) GetSourceDB() *sql.DB {
	return cm.sourceDB
}

// GetTargetDB returns the target database connection
func (cm *ConnectionManager) GetTargetDB() *sql.DB {
	return cm.targetDB
}

// TestConnections tests both source and target database connections
func (cm *ConnectionManager) TestConnections() error {
	if cm.sourceDB == nil {
		return fmt.Errorf("source database connection is not established")
	}

	if cm.targetDB == nil {
		return fmt.Errorf("target database connection is not established")
	}

	if err := cm.service.TestConnection(cm.sourceDB); err != nil {
		return fmt.Errorf("source database connection test failed: %w", err)
	}

	if err := cm.service.TestConnection(cm.targetDB); err != nil {
		return fmt.Errorf("target database connection test failed: %w", err)
	}

	return nil
}

// Close gracefully closes all database connections
func (cm *ConnectionManager) Close() error {
	var errs []error

	if cm.sourceDB != nil {
		if err := cm.service.Close(cm.sourceDB); err != nil {
			errs = append(errs, fmt.Errorf("failed to close source database: %w", err))
		}
		cm.sourceDB = nil
	}

	if cm.targetDB != nil {
		if err := cm.service.Close(cm.targetDB); err != nil {
			errs = append(errs, fmt.Errorf("failed to close target database: %w", err))
		}
		cm.targetDB = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// GetVersions returns the versions of both source and target databases
func (cm *ConnectionManager) GetVersions() (sourceVersion, targetVersion string, err error) {
	if cm.sourceDB == nil || cm.targetDB == nil {
		return "", "", fmt.Errorf("both database connections must be established")
	}

	sourceVersion, err = cm.service.GetVersion(cm.sourceDB)
	if err != nil {
		return "", "", fmt.Errorf("failed to get source database version: %w", err)
	}

	targetVersion, err = cm.service.GetVersion(cm.targetDB)
	if err != nil {
		return "", "", fmt.Errorf("failed to get target database version: %w", err)
	}

	return sourceVersion, targetVersion, nil
}
