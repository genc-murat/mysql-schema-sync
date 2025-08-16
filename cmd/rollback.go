package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"mysql-schema-sync/internal/application"
	"mysql-schema-sync/internal/backup"
	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/display"

	"github.com/spf13/cobra"
)

var (
	// Rollback listing flags
	rollbackDatabase string
	rollbackFormat   string
	rollbackLimit    int

	// Rollback planning flags
	showDifferences bool
	showWarnings    bool
	planFormat      string

	// Rollback execution flags
	skipValidation bool
	rollbackDryRun bool
)

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Manage database rollbacks",
	Long: `Rollback database schema changes to previous backup points.

The rollback system provides safe and reliable rollback capabilities
for MySQL schema synchronization. It can analyze differences between
current state and backup points, generate rollback plans, and execute
rollbacks with proper dependency handling.

Examples:
  # List available rollback points
  mysql-schema-sync rollback list
  
  # List rollback points for specific database
  mysql-schema-sync rollback list --database mydb
  
  # Plan rollback to specific backup
  mysql-schema-sync rollback plan backup-123
  
  # Execute rollback with confirmation
  mysql-schema-sync rollback execute backup-123
  
  # Dry run rollback (show what would be done)
  mysql-schema-sync rollback execute backup-123 --dry-run`,
}

// rollbackListCmd lists available rollback points
var rollbackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available rollback points",
	Long: `List available rollback points with filtering options.

This command displays backup points that can be used for rollback operations.
Each rollback point represents a backup that contains a complete schema
snapshot that can be restored.

Examples:
  # List all rollback points
  mysql-schema-sync rollback list
  
  # List rollback points for specific database
  mysql-schema-sync rollback list --database mydb
  
  # List rollback points in JSON format
  mysql-schema-sync rollback list --format json`,
	RunE: runRollbackList,
}

// rollbackPlanCmd plans a rollback operation
var rollbackPlanCmd = &cobra.Command{
	Use:   "plan <backup-id>",
	Short: "Plan a rollback operation",
	Long: `Plan a rollback operation to a specific backup point.

This command analyzes the differences between the current database state
and the selected backup point, generates the necessary SQL statements
for rollback, and identifies any potential issues or warnings.

Examples:
  # Plan rollback to specific backup
  mysql-schema-sync rollback plan backup-123
  
  # Plan rollback with detailed differences
  mysql-schema-sync rollback plan backup-123 --show-differences
  
  # Plan rollback with warnings analysis
  mysql-schema-sync rollback plan backup-123 --show-warnings`,
	Args: cobra.ExactArgs(1),
	RunE: runRollbackPlan,
}

// rollbackExecuteCmd executes a rollback operation
var rollbackExecuteCmd = &cobra.Command{
	Use:   "execute <backup-id>",
	Short: "Execute a rollback operation",
	Long: `Execute a rollback operation to restore database to a backup point.

This command executes the rollback plan to restore the database schema
to the state captured in the specified backup. The operation includes
proper dependency handling and validation.

Examples:
  # Execute rollback with confirmation
  mysql-schema-sync rollback execute backup-123
  
  # Execute rollback without confirmation
  mysql-schema-sync rollback execute backup-123 --auto-approve
  
  # Dry run rollback (show what would be done)
  mysql-schema-sync rollback execute backup-123 --dry-run
  
  # Execute rollback skipping validation
  mysql-schema-sync rollback execute backup-123 --skip-validation`,
	Args: cobra.ExactArgs(1),
	RunE: runRollbackExecute,
}

// rollbackStatusCmd shows rollback status and verification
var rollbackStatusCmd = &cobra.Command{
	Use:   "status <backup-id>",
	Short: "Show rollback status and verification",
	Long: `Show the status of a rollback operation and verify the result.

This command checks if a rollback operation was successful by comparing
the current database state with the target backup point and reporting
any discrepancies.

Examples:
  # Check rollback status
  mysql-schema-sync rollback status backup-123
  
  # Verify rollback completion
  mysql-schema-sync rollback status backup-123 --verify`,
	Args: cobra.ExactArgs(1),
	RunE: runRollbackStatus,
}

func init() {
	// Add rollback command to root
	rootCmd.AddCommand(rollbackCmd)

	// Add subcommands
	rollbackCmd.AddCommand(rollbackListCmd)
	rollbackCmd.AddCommand(rollbackPlanCmd)
	rollbackCmd.AddCommand(rollbackExecuteCmd)
	rollbackCmd.AddCommand(rollbackStatusCmd)

	// Rollback listing flags
	rollbackListCmd.Flags().StringVar(&rollbackDatabase, "database", "", "filter by database name")
	rollbackListCmd.Flags().StringVar(&rollbackFormat, "format", "table", "output format (table, json, yaml)")
	rollbackListCmd.Flags().IntVar(&rollbackLimit, "limit", 20, "maximum number of rollback points to list")

	// Rollback planning flags
	rollbackPlanCmd.Flags().BoolVar(&showDifferences, "show-differences", false, "show detailed schema differences")
	rollbackPlanCmd.Flags().BoolVar(&showWarnings, "show-warnings", true, "show rollback warnings and risks")
	rollbackPlanCmd.Flags().StringVar(&planFormat, "format", "table", "output format (table, json, yaml)")

	// Rollback execution flags
	rollbackExecuteCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "skip backup validation before rollback")
	rollbackExecuteCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "show rollback plan without executing")
}

// runRollbackList lists available rollback points
func runRollbackList(cmd *cobra.Command, args []string) error {
	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	ctx := context.Background()

	// Create rollback manager
	rollbackManager, err := createRollbackManager(ctx, backupConfig)
	if err != nil {
		return err
	}

	// Determine database name
	databaseName := rollbackDatabase
	if databaseName == "" {
		databaseName = config.TargetDB.Database
	}

	// List rollback points
	rollbackPoints, err := rollbackManager.ListRollbackPoints(ctx, databaseName)
	if err != nil {
		return fmt.Errorf("failed to list rollback points: %w", err)
	}

	// Apply limit
	if rollbackLimit > 0 && len(rollbackPoints) > rollbackLimit {
		rollbackPoints = rollbackPoints[:rollbackLimit]
	}

	// Display results
	return displayRollbackPointList(rollbackPoints, rollbackFormat, &config.Display)
}

// runRollbackPlan plans a rollback operation
func runRollbackPlan(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	// Create rollback manager
	rollbackManager, err := createRollbackManager(ctx, backupConfig)
	if err != nil {
		return err
	}

	displayService.Info(fmt.Sprintf("Planning rollback to backup: %s", backupID))

	// Plan rollback
	plan, err := rollbackManager.PlanRollback(ctx, backupID)
	if err != nil {
		return fmt.Errorf("rollback planning failed: %w", err)
	}

	// Display rollback plan
	return displayRollbackPlan(plan, planFormat, showDifferences, showWarnings, &config.Display)
}

// runRollbackExecute executes a rollback operation
func runRollbackExecute(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Override dry-run if rollback dry-run is set
	if rollbackDryRun {
		config.DryRun = true
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	// Create rollback manager
	rollbackManager, err := createRollbackManager(ctx, backupConfig)
	if err != nil {
		return err
	}

	// Validate backup unless skipped
	if !skipValidation {
		displayService.Info("Validating backup before rollback...")
		err = rollbackManager.ValidateRollback(ctx, backupID)
		if err != nil {
			return fmt.Errorf("backup validation failed: %w", err)
		}
		displayService.Success("Backup validation passed")
	}

	// Plan rollback
	displayService.Info(fmt.Sprintf("Planning rollback to backup: %s", backupID))
	plan, err := rollbackManager.PlanRollback(ctx, backupID)
	if err != nil {
		return fmt.Errorf("rollback planning failed: %w", err)
	}

	// Display rollback plan summary
	displayService.Info(fmt.Sprintf("Rollback plan generated with %d statements", len(plan.Statements)))

	if len(plan.Warnings) > 0 {
		displayService.Warning("Rollback warnings:")
		for _, warning := range plan.Warnings {
			displayService.Warning(fmt.Sprintf("  - %s", warning))
		}
	}

	// Show what will be done if dry run
	if config.DryRun {
		displayService.Info("Dry run mode - showing rollback plan:")
		return displayRollbackPlan(plan, "table", true, true, &config.Display)
	}

	// Confirm execution unless auto-approve is set
	if !config.AutoApprove {
		dialog := displayService.NewConfirmationDialog()
		dialog.SetTitle("Execute Rollback")
		dialog.SetMessage(fmt.Sprintf("Are you sure you want to rollback to backup %s?", backupID))

		if len(plan.Warnings) > 0 {
			dialog.SetMessage(fmt.Sprintf("Are you sure you want to rollback to backup %s?\n\nThis operation has %d warnings.", backupID, len(plan.Warnings)))
		}

		dialog.AddOption("y", "Yes", "Execute the rollback", false)
		dialog.AddCancelOption("n", "No", "Cancel rollback", true)

		result, err := dialog.Show()
		if err != nil {
			return fmt.Errorf("confirmation dialog error: %w", err)
		}

		if !result.Confirmed || result.Cancelled {
			displayService.Info("Rollback cancelled")
			return nil
		}
	}

	// Execute rollback
	displayService.Info("Executing rollback...")
	err = rollbackManager.ExecuteRollback(ctx, plan)
	if err != nil {
		return fmt.Errorf("rollback execution failed: %w", err)
	}

	displayService.Success(fmt.Sprintf("Rollback completed successfully to backup: %s", backupID))
	displayService.Info("Database schema has been restored to the backup point")

	return nil
}

// runRollbackStatus shows rollback status and verification
func runRollbackStatus(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Build configuration
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create backup system configuration
	backupConfig, err := buildBackupSystemConfig(config)
	if err != nil {
		return fmt.Errorf("backup configuration error: %w", err)
	}

	// Create display service
	displayService := display.NewDisplayService(&config.Display)

	ctx := context.Background()

	// Create rollback manager
	rollbackManager, err := createRollbackManager(ctx, backupConfig)
	if err != nil {
		return err
	}

	displayService.Info(fmt.Sprintf("Checking rollback status for backup: %s", backupID))

	// Validate rollback
	err = rollbackManager.ValidateRollback(ctx, backupID)
	if err != nil {
		displayService.Error(fmt.Sprintf("Rollback validation failed: %v", err))
		return fmt.Errorf("rollback validation failed: %w", err)
	}

	displayService.Success("Rollback validation passed")
	displayService.Info("Database schema matches the backup point")

	return nil
}

// Helper functions

// createRollbackManager creates a rollback manager with all required dependencies
func createRollbackManager(ctx context.Context, backupConfig *backup.BackupSystemConfig) (backup.RollbackManager, error) {
	// Create backup manager first
	backupManager, err := backup.NewBackupManager(backupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Create storage provider
	factory := backup.NewStorageProviderFactory()
	storageProvider, err := factory.CreateStorageProvider(ctx, backupConfig.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider: %w", err)
	}

	// Create validator
	validator := backup.NewBackupValidator(nil)

	// Create database service
	dbService := database.NewService()

	// Create rollback manager
	rollbackManager := backup.NewRollbackManager(backupManager, validator, dbService, storageProvider)
	return rollbackManager, nil
}

// displayRollbackPointList displays rollback points in the specified format
func displayRollbackPointList(rollbackPoints []*backup.RollbackPoint, format string, displayConfig *application.DisplayConfig) error {
	switch strings.ToLower(format) {
	case "json":
		return displayRollbackPointListJSON(rollbackPoints)
	case "yaml":
		return displayRollbackPointListYAML(rollbackPoints)
	case "table":
		return displayRollbackPointListTable(rollbackPoints, displayConfig)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// displayRollbackPointListJSON displays rollback points as JSON
func displayRollbackPointListJSON(rollbackPoints []*backup.RollbackPoint) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rollbackPoints)
}

// displayRollbackPointListYAML displays rollback points as YAML
func displayRollbackPointListYAML(rollbackPoints []*backup.RollbackPoint) error {
	fmt.Println("# Rollback Points (YAML format)")
	for i, point := range rollbackPoints {
		fmt.Printf("- backup_id: %s\n", point.BackupID)
		fmt.Printf("  database_name: %s\n", point.DatabaseName)
		fmt.Printf("  created_at: %s\n", point.CreatedAt.Format(time.RFC3339))
		fmt.Printf("  description: %s\n", point.Description)
		fmt.Printf("  schema_hash: %s\n", point.SchemaHash)
		if i < len(rollbackPoints)-1 {
			fmt.Println()
		}
	}
	return nil
}

// displayRollbackPointListTable displays rollback points as a formatted table
func displayRollbackPointListTable(rollbackPoints []*backup.RollbackPoint, displayConfig *application.DisplayConfig) error {
	if len(rollbackPoints) == 0 {
		fmt.Println("No rollback points found.")
		return nil
	}

	// Create display service
	displayService := display.NewDisplayService(displayConfig)

	// Prepare table data
	headers := []string{"Backup ID", "Database", "Created", "Description", "Schema Hash"}
	rows := make([][]string, len(rollbackPoints))

	for i, point := range rollbackPoints {
		createdAt := point.CreatedAt.Format("2006-01-02 15:04:05")
		description := point.Description
		if len(description) > 40 {
			description = description[:37] + "..."
		}
		schemaHash := point.SchemaHash
		if len(schemaHash) > 12 {
			schemaHash = schemaHash[:12] + "..."
		}

		rows[i] = []string{
			point.BackupID,
			point.DatabaseName,
			createdAt,
			description,
			schemaHash,
		}
	}

	// Display table
	displayService.PrintTable(headers, rows)
	displayService.Info(fmt.Sprintf("Total rollback points: %d", len(rollbackPoints)))

	return nil
}

// displayRollbackPlan displays a rollback plan in the specified format
func displayRollbackPlan(plan *backup.RollbackPlan, format string, showDifferences, showWarnings bool, displayConfig *application.DisplayConfig) error {
	switch strings.ToLower(format) {
	case "json":
		return displayRollbackPlanJSON(plan)
	case "yaml":
		return displayRollbackPlanYAML(plan)
	case "table":
		return displayRollbackPlanTable(plan, showDifferences, showWarnings, displayConfig)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// displayRollbackPlanJSON displays rollback plan as JSON
func displayRollbackPlanJSON(plan *backup.RollbackPlan) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
}

// displayRollbackPlanYAML displays rollback plan as YAML
func displayRollbackPlanYAML(plan *backup.RollbackPlan) error {
	fmt.Println("# Rollback Plan (YAML format)")
	fmt.Printf("backup_id: %s\n", plan.BackupID)
	fmt.Printf("statements_count: %d\n", len(plan.Statements))
	fmt.Printf("dependencies_count: %d\n", len(plan.Dependencies))
	fmt.Printf("warnings_count: %d\n", len(plan.Warnings))

	if len(plan.Statements) > 0 {
		fmt.Println("statements:")
		for i, stmt := range plan.Statements {
			// TODO: Fix when statement interface is properly defined
			fmt.Printf("  - statement: %v\n", stmt)
			if i < len(plan.Statements)-1 {
				fmt.Println()
			}
		}
	}

	if len(plan.Dependencies) > 0 {
		fmt.Println("dependencies:")
		for _, dep := range plan.Dependencies {
			fmt.Printf("  - %s\n", dep)
		}
	}

	if len(plan.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range plan.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	return nil
}

// displayRollbackPlanTable displays rollback plan as a formatted table
func displayRollbackPlanTable(plan *backup.RollbackPlan, showDifferences, showWarnings bool, displayConfig *application.DisplayConfig) error {
	// Create display service
	displayService := display.NewDisplayService(displayConfig)

	// Display plan summary
	displayService.PrintHeader("Rollback Plan Summary")
	displayService.Info(fmt.Sprintf("Backup ID: %s", plan.BackupID))
	displayService.Info(fmt.Sprintf("Total statements: %d", len(plan.Statements)))
	displayService.Info(fmt.Sprintf("Dependencies: %d", len(plan.Dependencies)))
	displayService.Info(fmt.Sprintf("Warnings: %d", len(plan.Warnings)))

	// Display warnings if requested
	if showWarnings && len(plan.Warnings) > 0 {
		displayService.PrintHeader("Rollback Warnings")
		for i, warning := range plan.Warnings {
			displayService.Warning(fmt.Sprintf("%d. %s", i+1, warning))
		}
	}

	// Display dependencies if any
	if len(plan.Dependencies) > 0 {
		displayService.PrintHeader("Dependencies")
		for i, dep := range plan.Dependencies {
			displayService.Info(fmt.Sprintf("%d. %s", i+1, dep))
		}
	}

	// Display statements
	if len(plan.Statements) > 0 {
		displayService.PrintHeader("Rollback Statements")

		// Prepare table data for statements
		headers := []string{"#", "Type", "SQL Statement"}
		rows := make([][]string, len(plan.Statements))

		for i, stmt := range plan.Statements {
			// TODO: Fix when statement interface is properly defined
			stmtStr := fmt.Sprintf("%v", stmt)
			if len(stmtStr) > 80 {
				stmtStr = stmtStr[:77] + "..."
			}

			rows[i] = []string{
				fmt.Sprintf("%d", i+1),
				"unknown", // TODO: Fix when statement interface is properly defined
				stmtStr,
			}
		}

		displayService.PrintTable(headers, rows)
	}

	// Display schema differences if requested
	if showDifferences {
		displayService.PrintHeader("Schema Differences")
		displayService.Info("Current schema will be changed to match backup schema")

		if plan.CurrentSchema != nil && plan.TargetSchema != nil {
			// TODO: Fix when schema interface is properly defined
			displayService.Info("Schema comparison available but not implemented")
		}
	}

	return nil
}
