package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/display"
	appErrors "mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/execution"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/schema"
)

// Application represents the main application
type Application struct {
	executor        *execution.Executor
	logger          *logging.Logger
	shutdownHandler *appErrors.GracefulShutdownHandler
	displayService  display.DisplayService
}

// Config holds the application configuration
type Config struct {
	SourceDB    database.DatabaseConfig `mapstructure:"source" yaml:"source"`
	TargetDB    database.DatabaseConfig `mapstructure:"target" yaml:"target"`
	DryRun      bool                    `mapstructure:"dry_run" yaml:"dry_run"`
	AutoApprove bool                    `mapstructure:"auto_approve" yaml:"auto_approve"`
	Verbose     bool                    `mapstructure:"verbose" yaml:"verbose"`
	Quiet       bool                    `mapstructure:"quiet" yaml:"quiet"`
	LogFile     string                  `mapstructure:"log_file" yaml:"log_file"`
	Timeout     time.Duration           `mapstructure:"timeout" yaml:"timeout"`
	Display     DisplayConfig           `mapstructure:"display" yaml:"display"`
}

// DisplayConfig is an alias to the display package's DisplayConfig
type DisplayConfig = display.DisplayConfig

// NewApplication creates a new application instance
func NewApplication(config Config) (*Application, error) {
	// Determine log level
	logLevel := logging.LogLevelNormal
	if config.Quiet {
		logLevel = logging.LogLevelQuiet
	} else if config.Verbose {
		logLevel = logging.LogLevelVerbose
	}

	// Create and configure display service
	displayConfig := &config.Display

	// Handle conflicting verbose/quiet modes - quiet takes precedence
	if config.Quiet && config.Verbose {
		displayConfig.VerboseMode = false
		displayConfig.QuietMode = true
	} else {
		displayConfig.VerboseMode = config.Verbose
		displayConfig.QuietMode = config.Quiet
	}

	displayConfig.SetDefaults()

	// Validate display configuration
	if err := displayConfig.Validate(); err != nil {
		return nil, fmt.Errorf("display configuration validation failed: %w", err)
	}

	displayService := display.NewDisplayService(displayConfig)

	// Create execution config
	execConfig := execution.ExecutionConfig{
		SourceDB:    config.SourceDB,
		TargetDB:    config.TargetDB,
		DryRun:      config.DryRun,
		AutoApprove: config.AutoApprove,
		Timeout:     config.Timeout,
		LogLevel:    logLevel,
	}

	// Create executor
	executor, err := execution.NewExecutor(execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Validate configuration
	if err := executor.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set the display service on the executor
	executor.SetDisplayService(displayService)

	app := &Application{
		executor:        executor,
		logger:          executor.GetLogger(),
		shutdownHandler: executor.GetShutdownHandler(),
		displayService:  displayService,
	}

	return app, nil
}

// Run executes the application
func (app *Application) Run() error {
	app.logger.Info("MySQL Schema Sync starting")
	app.displayService.Info("MySQL Schema Sync starting")

	// Set up signal handling for graceful shutdown
	app.setupSignalHandling()

	// Create context for the operation
	ctx := context.Background()

	// Execute the schema synchronization
	result, err := app.executor.Execute(ctx)
	if err != nil {
		app.handleExecutionError(err)
		return err
	}

	// Display results
	app.displayResults(result)

	app.logger.Info("MySQL Schema Sync completed")
	app.displayService.Success("MySQL Schema Sync completed")
	return nil
}

// setupSignalHandling sets up graceful shutdown on interrupt signals
func (app *Application) setupSignalHandling() {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to handle signals
	go func() {
		sig := <-sigChan
		app.logger.WithField("signal", sig.String()).Info("Received shutdown signal")

		// Trigger graceful shutdown
		app.shutdownHandler.RegisterShutdownFunc(func() error {
			app.logger.Info("Performing graceful shutdown...")
			return nil
		})

		// Exit the application
		os.Exit(0)
	}()
}

// handleExecutionError handles and logs execution errors
func (app *Application) handleExecutionError(err error) {
	// Use the executor's error handling
	processedErr := app.executor.HandleError(err)

	// Display user-friendly error message
	userMessage := appErrors.FormatUserError(processedErr)
	app.displayService.Error(userMessage)

	// Log detailed error information
	var appErr *appErrors.AppError
	if errors.As(processedErr, &appErr) {
		app.logger.WithFields(map[string]interface{}{
			"error_type":  string(appErr.Type),
			"recoverable": appErr.IsRecoverable(),
			"context":     appErr.Context,
		}).Error("Execution failed")

		// Provide troubleshooting hints
		app.provideTroubleshootingHints(appErr)
	}
}

// provideTroubleshootingHints provides helpful troubleshooting information
func (app *Application) provideTroubleshootingHints(appErr *appErrors.AppError) {
	var hints []string

	switch appErr.Type {
	case appErrors.ErrorTypeConnection:
		hints = []string{
			"Check that the database server is running",
			"Verify the host and port are correct",
			"Ensure network connectivity to the database server",
			"Check firewall settings",
		}

	case appErrors.ErrorTypePermission:
		hints = []string{
			"Verify the username and password are correct",
			"Check that the user has the required permissions",
			"Ensure the user can connect from your host",
		}

	case appErrors.ErrorTypeValidation:
		hints = []string{
			"Check that the database names are correct",
			"Verify that the databases exist",
			"Review the command line arguments",
		}

	case appErrors.ErrorTypeTimeout:
		hints = []string{
			"The operation may be taking longer than expected",
			"Try increasing the timeout value",
			"Check database server performance",
		}

	case appErrors.ErrorTypeSQL:
		hints = []string{
			"Review the SQL statements being executed",
			"Check for syntax errors or unsupported features",
			"Verify database permissions for schema modifications",
		}
	}

	if len(hints) > 0 {
		app.displayService.PrintSection("Troubleshooting hints", hints)
	}
}

// displayResults displays the execution results to the user
func (app *Application) displayResults(result *execution.ExecutionResult) {
	if result == nil {
		return
	}

	app.displayService.PrintHeader("Schema Synchronization Results")

	// Display status
	statusMessage := fmt.Sprintf("Status: %s", app.getStatusString(result.Success))
	if result.Success {
		app.displayService.Success(statusMessage)
	} else {
		app.displayService.Error(statusMessage)
	}

	app.displayService.Info(fmt.Sprintf("Duration: %s", result.Duration.String()))

	if result.SchemaDiff != nil {
		app.displaySchemaDiff(result.SchemaDiff)
	}

	if len(result.Warnings) > 0 {
		app.displayService.PrintSection("Warnings", result.Warnings)
	}

	if len(result.ExecutedStatements) > 0 {
		app.displayService.PrintSection(fmt.Sprintf("Executed Statements (%d)", len(result.ExecutedStatements)), nil)
		app.displayService.PrintSQL(result.ExecutedStatements)
	} else if result.Success && result.SchemaDiff != nil {
		if app.executor.GetLogger().GetLevel() == logging.LogLevelVerbose {
			app.displayService.Info("No statements executed (dry run mode or no changes needed)")
		}
	}
}

// displaySchemaDiff displays schema differences
func (app *Application) displaySchemaDiff(diff *schema.SchemaDiff) {
	if diff == nil {
		return
	}

	totalChanges := len(diff.AddedTables) + len(diff.RemovedTables) + len(diff.ModifiedTables) +
		len(diff.AddedIndexes) + len(diff.RemovedIndexes)

	app.displayService.Info(fmt.Sprintf("Changes Found: %d", totalChanges))

	// Use the enhanced schema diff presenter for detailed formatting
	presenter := app.displayService.NewSchemaDiffPresenter()
	formattedDiff := presenter.FormatSchemaDiff(diff)

	// Display the formatted diff as a section
	app.displayService.PrintSection("Schema Differences", formattedDiff)
}

// getStatusString returns a formatted status string
func (app *Application) getStatusString(success bool) string {
	if success {
		return "SUCCESS"
	}
	return "FAILED"
}

// GetLogger returns the application logger
func (app *Application) GetLogger() *logging.Logger {
	return app.logger
}

// GetDisplayService returns the application display service
func (app *Application) GetDisplayService() display.DisplayService {
	return app.displayService
}

// Shutdown performs graceful shutdown
func (app *Application) Shutdown() error {
	app.logger.Info("Shutting down application")

	// Wait for any ongoing operations to complete
	app.shutdownHandler.WaitForShutdown()

	app.logger.Info("Application shutdown complete")
	return nil
}
