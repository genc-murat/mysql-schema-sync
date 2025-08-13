package database

import (
	"database/sql"
	"fmt"
)

// ConnectionManager manages database connections for source and target databases
type ConnectionManager struct {
	service        DatabaseService
	sourceDB       *sql.DB
	targetDB       *sql.DB
	displayService DisplayService
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		service:        NewService(),
		displayService: nil, // Will be set via SetDisplayService
	}
}

// NewConnectionManagerWithService creates a new connection manager with a custom service
func NewConnectionManagerWithService(service DatabaseService) *ConnectionManager {
	return &ConnectionManager{
		service:        service,
		displayService: nil, // Will be set via SetDisplayService
	}
}

// SetDisplayService sets the display service for visual enhancements
func (cm *ConnectionManager) SetDisplayService(displayService DisplayService) {
	cm.displayService = displayService

	// Also set it on the underlying service if it supports it
	if serviceWithDisplay, ok := cm.service.(*Service); ok {
		serviceWithDisplay.SetDisplayService(displayService)
	}
}

// ConnectToSource establishes connection to the source database
func (cm *ConnectionManager) ConnectToSource(config DatabaseConfig) error {
	if cm.sourceDB != nil {
		cm.service.Close(cm.sourceDB)
	}

	if cm.displayService != nil {
		cm.displayService.Info(fmt.Sprintf("Establishing source database connection..."))
	}

	db, err := cm.service.Connect(config)
	if err != nil {
		if cm.displayService != nil {
			cm.displayService.Error(fmt.Sprintf("Failed to connect to source database: %v", err))
		}
		return fmt.Errorf("failed to connect to source database: %w", err)
	}

	cm.sourceDB = db
	if cm.displayService != nil {
		cm.displayService.Success(fmt.Sprintf("%s Source database connection established",
			cm.displayService.RenderIconWithColor("success")))
	}
	return nil
}

// ConnectToTarget establishes connection to the target database
func (cm *ConnectionManager) ConnectToTarget(config DatabaseConfig) error {
	if cm.targetDB != nil {
		cm.service.Close(cm.targetDB)
	}

	if cm.displayService != nil {
		cm.displayService.Info(fmt.Sprintf("Establishing target database connection..."))
	}

	db, err := cm.service.Connect(config)
	if err != nil {
		if cm.displayService != nil {
			cm.displayService.Error(fmt.Sprintf("Failed to connect to target database: %v", err))
		}
		return fmt.Errorf("failed to connect to target database: %w", err)
	}

	cm.targetDB = db
	if cm.displayService != nil {
		cm.displayService.Success(fmt.Sprintf("%s Target database connection established",
			cm.displayService.RenderIconWithColor("success")))
	}
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
		err := fmt.Errorf("source database connection is not established")
		if cm.displayService != nil {
			cm.displayService.Error("Source database connection is not established")
		}
		return err
	}

	if cm.targetDB == nil {
		err := fmt.Errorf("target database connection is not established")
		if cm.displayService != nil {
			cm.displayService.Error("Target database connection is not established")
		}
		return err
	}

	if cm.displayService != nil {
		cm.displayService.Info("Testing database connections...")
	}

	// Test source connection
	if err := cm.service.TestConnection(cm.sourceDB); err != nil {
		if cm.displayService != nil {
			cm.displayService.Error(fmt.Sprintf("Source database connection test failed: %v", err))
		}
		return fmt.Errorf("source database connection test failed: %w", err)
	}

	// Test target connection
	if err := cm.service.TestConnection(cm.targetDB); err != nil {
		if cm.displayService != nil {
			cm.displayService.Error(fmt.Sprintf("Target database connection test failed: %v", err))
		}
		return fmt.Errorf("target database connection test failed: %w", err)
	}

	if cm.displayService != nil {
		cm.displayService.Success(fmt.Sprintf("%s Both database connections are healthy",
			cm.displayService.RenderIconWithColor("success")))
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
