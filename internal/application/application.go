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
}

// NewApplication creates a new application instance
func NewApplication(config Config) (*Application, error) {
	// Determine log level
	logLevel := logging.LogLevelNormal
	if config.Quiet {
		logLevel = logging.LogLevelQuiet
	} else if config.Verbose {
		logLevel = logging.LogLevelVerbose
	}

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

	app := &Application{
		executor:        executor,
		logger:          executor.GetLogger(),
		shutdownHandler: executor.GetShutdownHandler(),
	}

	return app, nil
}

// Run executes the application
func (app *Application) Run() error {
	app.logger.Info("MySQL Schema Sync starting")

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
	fmt.Fprintf(os.Stderr, "Error: %s\n", userMessage)

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
	switch appErr.Type {
	case appErrors.ErrorTypeConnection:
		fmt.Fprintf(os.Stderr, "\nTroubleshooting hints:\n")
		fmt.Fprintf(os.Stderr, "- Check that the database server is running\n")
		fmt.Fprintf(os.Stderr, "- Verify the host and port are correct\n")
		fmt.Fprintf(os.Stderr, "- Ensure network connectivity to the database server\n")
		fmt.Fprintf(os.Stderr, "- Check firewall settings\n")

	case appErrors.ErrorTypePermission:
		fmt.Fprintf(os.Stderr, "\nTroubleshooting hints:\n")
		fmt.Fprintf(os.Stderr, "- Verify the username and password are correct\n")
		fmt.Fprintf(os.Stderr, "- Check that the user has the required permissions\n")
		fmt.Fprintf(os.Stderr, "- Ensure the user can connect from your host\n")

	case appErrors.ErrorTypeValidation:
		fmt.Fprintf(os.Stderr, "\nTroubleshooting hints:\n")
		fmt.Fprintf(os.Stderr, "- Check that the database names are correct\n")
		fmt.Fprintf(os.Stderr, "- Verify that the databases exist\n")
		fmt.Fprintf(os.Stderr, "- Review the command line arguments\n")

	case appErrors.ErrorTypeTimeout:
		fmt.Fprintf(os.Stderr, "\nTroubleshooting hints:\n")
		fmt.Fprintf(os.Stderr, "- The operation may be taking longer than expected\n")
		fmt.Fprintf(os.Stderr, "- Try increasing the timeout value\n")
		fmt.Fprintf(os.Stderr, "- Check database server performance\n")

	case appErrors.ErrorTypeSQL:
		fmt.Fprintf(os.Stderr, "\nTroubleshooting hints:\n")
		fmt.Fprintf(os.Stderr, "- Review the SQL statements being executed\n")
		fmt.Fprintf(os.Stderr, "- Check for syntax errors or unsupported features\n")
		fmt.Fprintf(os.Stderr, "- Verify database permissions for schema modifications\n")
	}
}

// displayResults displays the execution results to the user
func (app *Application) displayResults(result *execution.ExecutionResult) {
	if result == nil {
		return
	}

	fmt.Printf("\n=== Schema Synchronization Results ===\n")
	fmt.Printf("Status: %s\n", app.getStatusString(result.Success))
	fmt.Printf("Duration: %s\n", result.Duration.String())

	if result.SchemaDiff != nil {
		app.displaySchemaDiff(result.SchemaDiff)
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(result.ExecutedStatements) > 0 {
		fmt.Printf("\nExecuted Statements (%d):\n", len(result.ExecutedStatements))
		for i, stmt := range result.ExecutedStatements {
			fmt.Printf("  %d. %s\n", i+1, stmt)
		}
	} else if result.Success && result.SchemaDiff != nil {
		if app.executor.GetLogger().GetLevel() == logging.LogLevelVerbose {
			fmt.Printf("\nNo statements executed (dry run mode or no changes needed)\n")
		}
	}

	fmt.Printf("\n")
}

// displaySchemaDiff displays schema differences
func (app *Application) displaySchemaDiff(diff *schema.SchemaDiff) {
	if diff == nil {
		return
	}

	totalChanges := len(diff.AddedTables) + len(diff.RemovedTables) + len(diff.ModifiedTables) +
		len(diff.AddedIndexes) + len(diff.RemovedIndexes)

	fmt.Printf("Changes Found: %d\n", totalChanges)

	if len(diff.AddedTables) > 0 {
		fmt.Printf("  Tables to Add: %d\n", len(diff.AddedTables))
		if app.logger.GetLevel() == logging.LogLevelVerbose {
			for _, table := range diff.AddedTables {
				fmt.Printf("    + %s\n", table.Name)
			}
		}
	}

	if len(diff.RemovedTables) > 0 {
		fmt.Printf("  Tables to Remove: %d\n", len(diff.RemovedTables))
		if app.logger.GetLevel() == logging.LogLevelVerbose {
			for _, table := range diff.RemovedTables {
				fmt.Printf("    - %s\n", table.Name)
			}
		}
	}

	if len(diff.ModifiedTables) > 0 {
		fmt.Printf("  Tables to Modify: %d\n", len(diff.ModifiedTables))
		if app.logger.GetLevel() == logging.LogLevelVerbose {
			for _, tableDiff := range diff.ModifiedTables {
				fmt.Printf("    ~ %s\n", tableDiff.TableName)
			}
		}
	}

	if len(diff.AddedIndexes) > 0 {
		fmt.Printf("  Indexes to Add: %d\n", len(diff.AddedIndexes))
	}

	if len(diff.RemovedIndexes) > 0 {
		fmt.Printf("  Indexes to Remove: %d\n", len(diff.RemovedIndexes))
	}
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

// Shutdown performs graceful shutdown
func (app *Application) Shutdown() error {
	app.logger.Info("Shutting down application")

	// Wait for any ongoing operations to complete
	app.shutdownHandler.WaitForShutdown()

	app.logger.Info("Application shutdown complete")
	return nil
}
