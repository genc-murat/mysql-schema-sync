package database

import (
	"context"
	"database/sql"
	"fmt"
	"mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/logging"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// DatabaseService defines the interface for database operations
type DatabaseService interface {
	Connect(config DatabaseConfig) (*sql.DB, error)
	TestConnection(db *sql.DB) error
	Close(db *sql.DB) error
	GetVersion(db *sql.DB) (string, error)
	ExecuteSQL(db *sql.DB, statements []string) error
}

// Service implements the DatabaseService interface
type Service struct {
	connectionTimeout time.Duration
	maxRetries        int
	retryDelay        time.Duration
	logger            *logging.Logger
	retryHandler      *errors.RetryHandler
	displayService    DisplayService
}

// DisplayService interface for visual enhancements (to avoid circular imports)
type DisplayService interface {
	StartSpinner(message string) SpinnerHandle
	UpdateSpinner(handle SpinnerHandle, message string)
	StopSpinner(handle SpinnerHandle, finalMessage string)
	ShowProgress(current, total int, message string)
	Success(message string)
	Warning(message string)
	Error(message string)
	Info(message string)
	RenderIconWithColor(name string) string
}

// SpinnerHandle interface for spinner management
type SpinnerHandle interface {
	ID() string
	IsActive() bool
}

// NewService creates a new database service with default settings
func NewService() *Service {
	return &Service{
		connectionTimeout: 30 * time.Second,
		maxRetries:        3,
		retryDelay:        2 * time.Second,
		logger:            logging.NewDefaultLogger(),
		retryHandler:      errors.NewDefaultRetryHandler(),
		displayService:    nil, // Will be set via SetDisplayService
	}
}

// NewServiceWithOptions creates a new database service with custom options
func NewServiceWithOptions(timeout time.Duration, maxRetries int, retryDelay time.Duration) *Service {
	retryConfig := errors.RetryConfig{
		MaxAttempts: maxRetries,
		BaseDelay:   retryDelay,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}

	return &Service{
		connectionTimeout: timeout,
		maxRetries:        maxRetries,
		retryDelay:        retryDelay,
		logger:            logging.NewDefaultLogger(),
		retryHandler:      errors.NewRetryHandler(retryConfig),
		displayService:    nil, // Will be set via SetDisplayService
	}
}

// NewServiceWithLogger creates a new database service with a custom logger
func NewServiceWithLogger(logger *logging.Logger) *Service {
	return &Service{
		connectionTimeout: 30 * time.Second,
		maxRetries:        3,
		retryDelay:        2 * time.Second,
		logger:            logger,
		retryHandler:      errors.NewDefaultRetryHandler(),
		displayService:    nil, // Will be set via SetDisplayService
	}
}

// SetDisplayService sets the display service for visual enhancements
func (s *Service) SetDisplayService(displayService DisplayService) {
	s.displayService = displayService
}

// Connect establishes a connection to the MySQL database with retry logic
func (s *Service) Connect(config DatabaseConfig) (*sql.DB, error) {
	startTime := time.Now()

	s.logger.WithFields(map[string]interface{}{
		"host":     config.Host,
		"database": config.Database,
		"port":     config.Port,
	}).Info("Attempting database connection")

	// Start spinner if display service is available
	var spinner SpinnerHandle
	if s.displayService != nil {
		connectionMsg := fmt.Sprintf("Connecting to %s@%s:%d...", config.Database, config.Host, config.Port)
		spinner = s.displayService.StartSpinner(connectionMsg)
	}

	ctx, cancel := errors.CreateContextWithTimeout(s.connectionTimeout)
	defer cancel()

	var db *sql.DB
	attempt := 0
	err := s.retryHandler.Retry(ctx, func() error {
		attempt++
		var connectErr error

		// Update spinner with retry information
		if s.displayService != nil && spinner != nil {
			if attempt > 1 {
				retryMsg := fmt.Sprintf("Connecting to %s@%s:%d (attempt %d/%d)...",
					config.Database, config.Host, config.Port, attempt, s.maxRetries)
				s.displayService.UpdateSpinner(spinner, retryMsg)
			}
		}

		dsn := config.DSN()
		db, connectErr = sql.Open("mysql", dsn)
		if connectErr != nil {
			return errors.WrapError(connectErr, "failed to open database connection")
		}

		// Set connection pool settings
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Test the connection
		if testErr := s.TestConnection(db); testErr != nil {
			db.Close()
			return testErr
		}

		return nil
	})

	duration := time.Since(startTime)
	success := err == nil

	// Stop spinner and show result
	if s.displayService != nil && spinner != nil {
		if success {
			successMsg := fmt.Sprintf("%s Connected to %s@%s:%d successfully (%.2fs)",
				s.displayService.RenderIconWithColor("success"),
				config.Database, config.Host, config.Port, duration.Seconds())
			s.displayService.StopSpinner(spinner, successMsg)
		} else {
			errorMsg := fmt.Sprintf("%s Failed to connect to %s@%s:%d after %d attempts",
				s.displayService.RenderIconWithColor("error"),
				config.Database, config.Host, config.Port, attempt)
			s.displayService.StopSpinner(spinner, errorMsg)
		}
	}

	s.logger.LogDatabaseConnection(config.Host, config.Database, success, duration, err)

	if err != nil {
		// Show enhanced error message
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Database connection failed: %v", err))
		}
		return nil, err
	}

	return db, nil
}

// TestConnection verifies that the database connection is working
func (s *Service) TestConnection(db *sql.DB) error {
	if db == nil {
		err := errors.NewAppError(errors.ErrorTypeValidation, "database connection is nil", nil)
		if s.displayService != nil {
			s.displayService.Error("Database connection is nil")
		}
		return err
	}

	// Create a context with timeout for the ping
	ctx, cancel := context.WithTimeout(context.Background(), s.connectionTimeout)
	defer cancel()

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		wrappedErr := errors.WrapError(err, "failed to ping database")
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Database ping failed: %v", err))
		}
		return wrappedErr
	}

	s.logger.Debug("Database connection test successful")
	return nil
}

// Close gracefully closes the database connection
func (s *Service) Close(db *sql.DB) error {
	if db == nil {
		s.logger.Debug("Database connection is nil, nothing to close")
		return nil
	}

	s.logger.Debug("Closing database connection")
	if err := db.Close(); err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to close database connection")
		return errors.WrapError(err, "failed to close database connection")
	}

	s.logger.Debug("Database connection closed successfully")
	return nil
}

// GetVersion retrieves the MySQL server version
func (s *Service) GetVersion(db *sql.DB) (string, error) {
	if db == nil {
		return "", errors.NewAppError(errors.ErrorTypeValidation, "database connection is nil", nil)
	}

	var version string
	query := "SELECT VERSION()"
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), s.connectionTimeout)
	defer cancel()

	err := db.QueryRowContext(ctx, query).Scan(&version)
	duration := time.Since(startTime)

	s.logger.LogSQLExecution(query, duration, 1, err)

	if err != nil {
		return "", errors.WrapError(err, "failed to get database version")
	}

	s.logger.WithField("version", version).Debug("Retrieved database version")
	return version, nil
}

// ExecuteSQL executes SQL statements with proper logging and error handling
func (s *Service) ExecuteSQL(db *sql.DB, statements []string) error {
	if db == nil {
		err := errors.NewAppError(errors.ErrorTypeValidation, "database connection is nil", nil)
		if s.displayService != nil {
			s.displayService.Error("Database connection is nil")
		}
		return err
	}

	if len(statements) == 0 {
		s.logger.Debug("No SQL statements to execute")
		if s.displayService != nil {
			s.displayService.Info("No SQL statements to execute")
		}
		return nil
	}

	s.logger.WithField("statement_count", len(statements)).Info("Executing SQL statements")

	// Show progress indicator for SQL execution
	if s.displayService != nil {
		s.displayService.Info(fmt.Sprintf("Executing %d SQL statements...", len(statements)))
	}

	// Start a transaction for atomic execution
	tx, err := db.Begin()
	if err != nil {
		wrappedErr := errors.WrapError(err, "failed to begin transaction")
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Failed to begin transaction: %v", err))
		}
		return wrappedErr
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.WithField("error", rollbackErr.Error()).Error("Failed to rollback transaction")
				if s.displayService != nil {
					s.displayService.Error(fmt.Sprintf("Failed to rollback transaction: %v", rollbackErr))
				}
			}
		}
	}()

	executed := 0
	for i, stmt := range statements {
		if stmt == "" {
			continue
		}

		// Show progress
		if s.displayService != nil {
			s.displayService.ShowProgress(i+1, len(statements), fmt.Sprintf("Executing statement %d/%d", i+1, len(statements)))
		}

		startTime := time.Now()
		result, execErr := tx.Exec(stmt)
		duration := time.Since(startTime)

		var rowsAffected int64
		if result != nil {
			rowsAffected, _ = result.RowsAffected()
		}

		s.logger.LogSQLExecution(logging.SanitizeSQL(stmt), duration, rowsAffected, execErr)

		if execErr != nil {
			err = errors.WrapError(execErr, fmt.Sprintf("failed to execute statement %d", i+1)).(*errors.AppError).WithContext("statement", stmt).WithContext("statement_index", i)
			if s.displayService != nil {
				s.displayService.Error(fmt.Sprintf("SQL execution failed at statement %d: %v", i+1, execErr))
			}
			return err
		}
		executed++
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		wrappedErr := errors.WrapError(err, "failed to commit transaction")
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Failed to commit transaction: %v", err))
		}
		return wrappedErr
	}

	s.logger.WithField("statement_count", len(statements)).Info("All SQL statements executed successfully")
	if s.displayService != nil {
		s.displayService.Success(fmt.Sprintf("%s Successfully executed %d SQL statements",
			s.displayService.RenderIconWithColor("success"), executed))
	}
	return nil
}
