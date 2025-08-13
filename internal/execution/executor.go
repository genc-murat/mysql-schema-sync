package execution

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/display"
	"mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

// ExecutionConfig holds configuration for the execution service
type ExecutionConfig struct {
	SourceDB    database.DatabaseConfig
	TargetDB    database.DatabaseConfig
	DryRun      bool
	AutoApprove bool
	Timeout     time.Duration
	LogLevel    logging.LogLevel
}

// ExecutionResult holds the result of an execution
type ExecutionResult struct {
	Success            bool
	SchemaDiff         *schema.SchemaDiff
	MigrationPlan      *migration.MigrationPlan
	ExecutedStatements []string
	Warnings           []string
	Duration           time.Duration
	Error              error
}

// Executor handles the main application execution flow with error recovery
type Executor struct {
	config           ExecutionConfig
	logger           *logging.Logger
	dbService        database.DatabaseService
	schemaService    *schema.Service
	migrationService migration.MigrationService
	retryHandler     *errors.RetryHandler
	shutdownHandler  *errors.GracefulShutdownHandler
	displayService   display.DisplayService
}

// NewExecutor creates a new executor with the given configuration
func NewExecutor(config ExecutionConfig) (*Executor, error) {
	// Create logger with specified level
	loggerConfig := logging.Config{
		Level:      config.LogLevel,
		Format:     "text",
		ShowCaller: config.LogLevel == logging.LogLevelDebug,
	}

	logger, err := logging.NewLogger(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create services with the logger
	dbService := database.NewServiceWithLogger(logger)
	schemaService := schema.NewServiceWithLogger(logger)
	migrationService := migration.NewMigrationServiceWithLogger(logger)

	// Create retry handler with custom configuration
	retryConfig := errors.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   2 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
	retryHandler := errors.NewRetryHandler(retryConfig)

	// Create graceful shutdown handler
	shutdownHandler := errors.NewGracefulShutdownHandler()

	executor := &Executor{
		config:           config,
		logger:           logger,
		dbService:        dbService,
		schemaService:    schemaService,
		migrationService: migrationService,
		retryHandler:     retryHandler,
		shutdownHandler:  shutdownHandler,
	}

	return executor, nil
}

// Execute runs the complete schema synchronization process
func (e *Executor) Execute(ctx context.Context) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{
		Success:            false,
		ExecutedStatements: make([]string, 0),
		Warnings:           make([]string, 0),
	}

	// Set up graceful shutdown
	e.shutdownHandler.Start()
	defer e.shutdownHandler.Stop()

	// Create a context with timeout
	if e.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	e.logger.Info("Starting schema synchronization process")

	// Step 1: Connect to databases
	sourceDB, targetDB, err := e.connectToDatabases(ctx)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Register database cleanup
	e.shutdownHandler.RegisterShutdownFunc(func() error {
		if sourceDB != nil {
			e.dbService.Close(sourceDB)
		}
		if targetDB != nil {
			e.dbService.Close(targetDB)
		}
		return nil
	})

	// Step 2: Extract schemas
	sourceSchemaDef, targetSchemaDef, err := e.extractSchemas(ctx, sourceDB, targetDB)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Step 3: Compare schemas
	schemaDiff, err := e.compareSchemas(sourceSchemaDef, targetSchemaDef)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}
	result.SchemaDiff = schemaDiff

	// Check if there are any differences
	if e.schemaService.IsSchemaDiffEmpty(schemaDiff) {
		e.logger.Info("No schema differences found - databases are already synchronized")
		result.Success = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Step 4: Create migration plan
	migrationPlan, err := e.createMigrationPlan(schemaDiff)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}
	result.MigrationPlan = migrationPlan
	result.Warnings = migrationPlan.Warnings

	// Step 5: Execute migration (if not dry run and approved)
	if !e.config.DryRun {
		if err := e.executeMigration(ctx, targetDB, migrationPlan); err != nil {
			result.Error = err
			result.Duration = time.Since(startTime)
			return result, err
		}

		// Extract executed statements for result
		for _, stmt := range migrationPlan.Statements {
			result.ExecutedStatements = append(result.ExecutedStatements, stmt.SQL)
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	e.logger.WithFields(map[string]interface{}{
		"duration":         result.Duration.String(),
		"statements_count": len(result.ExecutedStatements),
		"warnings_count":   len(result.Warnings),
		"dry_run":          e.config.DryRun,
	}).Info("Schema synchronization completed successfully")

	return result, nil
}

// connectToDatabases establishes connections to both source and target databases
func (e *Executor) connectToDatabases(ctx context.Context) (*sql.DB, *sql.DB, error) {
	e.logger.Info("Connecting to databases")

	var sourceDB, targetDB *sql.DB
	var err error

	// Start spinner for database connections if display service is available
	var spinner display.SpinnerHandle
	if e.displayService != nil {
		spinner = e.displayService.StartSpinner("Connecting to source database...")
	}

	// Connect to source database with retry
	err = e.retryHandler.Retry(ctx, func() error {
		sourceDB, err = e.dbService.Connect(e.config.SourceDB)
		return err
	})
	if err != nil {
		if e.displayService != nil {
			e.displayService.StopSpinner(spinner, "")
			e.displayService.Error(fmt.Sprintf("Failed to connect to source database: %s", e.config.SourceDB.Host))
		}
		return nil, nil, errors.WrapError(err, "failed to connect to source database")
	}

	// Update spinner for target database connection
	if e.displayService != nil {
		e.displayService.UpdateSpinner(spinner, "Connecting to target database...")
	}

	// Connect to target database with retry
	err = e.retryHandler.Retry(ctx, func() error {
		targetDB, err = e.dbService.Connect(e.config.TargetDB)
		return err
	})
	if err != nil {
		// Close source DB if target connection fails
		e.dbService.Close(sourceDB)
		if e.displayService != nil {
			e.displayService.StopSpinner(spinner, "")
			e.displayService.Error(fmt.Sprintf("Failed to connect to target database: %s", e.config.TargetDB.Host))
		}
		return nil, nil, errors.WrapError(err, "failed to connect to target database")
	}

	if e.displayService != nil {
		e.displayService.StopSpinner(spinner, "Successfully connected to both databases")
	}

	e.logger.Info("Successfully connected to both databases")
	return sourceDB, targetDB, nil
}

// extractSchemas extracts schema information from both databases
func (e *Executor) extractSchemas(ctx context.Context, sourceDB, targetDB *sql.DB) (*schema.Schema, *schema.Schema, error) {
	e.logger.Info("Extracting schema information")

	var sourceSchema, targetSchema *schema.Schema
	var err error

	// Start spinner for schema extraction if display service is available
	var spinner display.SpinnerHandle
	if e.displayService != nil {
		spinner = e.displayService.StartSpinner(fmt.Sprintf("Extracting schema from source database (%s)...", e.config.SourceDB.Database))
	}

	// Extract source schema
	err = e.retryHandler.Retry(ctx, func() error {
		sourceSchema, err = e.schemaService.ExtractSchemaFromDB(sourceDB, e.config.SourceDB.Database)
		return err
	})
	if err != nil {
		if e.displayService != nil {
			e.displayService.StopSpinner(spinner, "")
			e.displayService.Error("Failed to extract source schema")
		}
		return nil, nil, errors.WrapError(err, "failed to extract source schema")
	}

	// Update spinner for target schema extraction
	if e.displayService != nil {
		e.displayService.UpdateSpinner(spinner, fmt.Sprintf("Extracting schema from target database (%s)...", e.config.TargetDB.Database))
	}

	// Extract target schema
	err = e.retryHandler.Retry(ctx, func() error {
		targetSchema, err = e.schemaService.ExtractSchemaFromDB(targetDB, e.config.TargetDB.Database)
		return err
	})
	if err != nil {
		if e.displayService != nil {
			e.displayService.StopSpinner(spinner, "")
			e.displayService.Error("Failed to extract target schema")
		}
		return nil, nil, errors.WrapError(err, "failed to extract target schema")
	}

	if e.displayService != nil {
		e.displayService.StopSpinner(spinner, fmt.Sprintf("Schema extraction completed (%d source tables, %d target tables)", len(sourceSchema.Tables), len(targetSchema.Tables)))
	}

	e.logger.WithFields(map[string]interface{}{
		"source_tables": len(sourceSchema.Tables),
		"target_tables": len(targetSchema.Tables),
	}).Info("Schema extraction completed")

	return sourceSchema, targetSchema, nil
}

// compareSchemas compares the extracted schemas
func (e *Executor) compareSchemas(sourceSchema, targetSchema *schema.Schema) (*schema.SchemaDiff, error) {
	e.logger.Info("Comparing schemas")

	// Start spinner for schema comparison if display service is available
	var spinner display.SpinnerHandle
	if e.displayService != nil {
		spinner = e.displayService.StartSpinner("Comparing schemas...")
	}

	schemaDiff, err := e.schemaService.CompareSchemas(sourceSchema, targetSchema)
	if err != nil {
		if e.displayService != nil {
			e.displayService.StopSpinner(spinner, "")
			e.displayService.Error("Failed to compare schemas")
		}
		return nil, errors.WrapError(err, "failed to compare schemas")
	}

	changesCount := len(schemaDiff.AddedTables) + len(schemaDiff.RemovedTables) +
		len(schemaDiff.ModifiedTables) + len(schemaDiff.AddedIndexes) + len(schemaDiff.RemovedIndexes)

	if e.displayService != nil {
		e.displayService.StopSpinner(spinner, fmt.Sprintf("Schema comparison completed (%d changes found)", changesCount))
	}

	e.logger.WithField("changes_count", changesCount).Info("Schema comparison completed")
	return schemaDiff, nil
}

// createMigrationPlan creates a migration plan from the schema differences
func (e *Executor) createMigrationPlan(schemaDiff *schema.SchemaDiff) (*migration.MigrationPlan, error) {
	e.logger.Info("Creating migration plan")

	migrationPlan, err := e.migrationService.PlanMigration(schemaDiff)
	if err != nil {
		return nil, errors.WrapError(err, "failed to create migration plan")
	}

	// Validate the migration plan
	if err := e.migrationService.ValidatePlan(migrationPlan); err != nil {
		return nil, errors.WrapError(err, "migration plan validation failed")
	}

	e.logger.WithFields(map[string]interface{}{
		"statements_count": len(migrationPlan.Statements),
		"warnings_count":   len(migrationPlan.Warnings),
	}).Info("Migration plan created and validated")

	return migrationPlan, nil
}

// executeMigration executes the migration plan on the target database
func (e *Executor) executeMigration(ctx context.Context, targetDB *sql.DB, migrationPlan *migration.MigrationPlan) error {
	if len(migrationPlan.Statements) == 0 {
		e.logger.Info("No migration statements to execute")
		if e.displayService != nil {
			e.displayService.Info("No migration statements to execute")
		}
		return nil
	}

	e.logger.WithField("statements_count", len(migrationPlan.Statements)).Info("Executing migration")

	// Extract SQL statements
	sqlStatements := make([]string, len(migrationPlan.Statements))
	for i, stmt := range migrationPlan.Statements {
		sqlStatements[i] = stmt.SQL
	}

	// Create progress bar if display service is available
	var progressBar *display.ProgressBar
	if e.displayService != nil {
		progressBar = e.displayService.NewProgressBar(len(sqlStatements), "Executing migration statements")
	}

	// Execute statements one by one to show progress
	for i, stmt := range sqlStatements {
		if e.displayService != nil {
			progressBar.Update(i, fmt.Sprintf("Executing statement %d/%d", i+1, len(sqlStatements)))
		}

		// Execute single statement with retry logic
		err := e.retryHandler.Retry(ctx, func() error {
			return e.dbService.ExecuteSQL(targetDB, []string{stmt})
		})

		if err != nil {
			if e.displayService != nil {
				progressBar.Finish("Migration failed")
				e.displayService.Error(fmt.Sprintf("Failed to execute statement %d: %s", i+1, stmt))
			}
			return errors.WrapError(err, fmt.Sprintf("failed to execute migration statement %d", i+1))
		}
	}

	if e.displayService != nil {
		progressBar.Finish("Migration executed successfully")
		e.displayService.Success(fmt.Sprintf("Successfully executed %d migration statements", len(sqlStatements)))
	}

	e.logger.Info("Migration executed successfully")
	return nil
}

// GetLogger returns the logger instance
func (e *Executor) GetLogger() *logging.Logger {
	return e.logger
}

// GetShutdownHandler returns the shutdown handler
func (e *Executor) GetShutdownHandler() *errors.GracefulShutdownHandler {
	return e.shutdownHandler
}

// SetDisplayService sets the display service for enhanced output
func (e *Executor) SetDisplayService(displayService display.DisplayService) {
	e.displayService = displayService
}

// HandleError processes and logs errors appropriately
func (e *Executor) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// Classify the error
	classifier := errors.NewErrorClassifier()
	appErr := classifier.ClassifyError(err)

	// Log the error with appropriate level
	fields := map[string]interface{}{
		"error_type":  string(appErr.Type),
		"recoverable": appErr.IsRecoverable(),
	}

	// Add context if available
	for k, v := range appErr.Context {
		fields[k] = v
	}

	if appErr.IsRecoverable() {
		e.logger.WithFields(fields).Warn("Recoverable error occurred")
	} else {
		e.logger.WithFields(fields).Error("Non-recoverable error occurred")
	}

	return appErr
}

// ValidateConfig validates the execution configuration
func (e *Executor) ValidateConfig() error {
	if e.config.SourceDB.Host == "" {
		return errors.NewAppError(errors.ErrorTypeValidation, "source database host is required", nil)
	}
	if e.config.SourceDB.Database == "" {
		return errors.NewAppError(errors.ErrorTypeValidation, "source database name is required", nil)
	}
	if e.config.TargetDB.Host == "" {
		return errors.NewAppError(errors.ErrorTypeValidation, "target database host is required", nil)
	}
	if e.config.TargetDB.Database == "" {
		return errors.NewAppError(errors.ErrorTypeValidation, "target database name is required", nil)
	}

	e.logger.Debug("Configuration validation passed")
	return nil
}
