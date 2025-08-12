package cmd

import (
	"fmt"
	"mysql-schema-sync/internal/application"
	"mysql-schema-sync/internal/database"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// CLI flag variables
var (
	// Source database flags
	sourceHost     string
	sourcePort     int
	sourceUsername string
	sourcePassword string
	sourceDatabase string

	// Target database flags
	targetHost     string
	targetPort     int
	targetUsername string
	targetPassword string
	targetDatabase string

	// Operation flags
	dryRun      bool
	verbose     bool
	quiet       bool
	autoApprove bool
	timeout     time.Duration
	logFile     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mysql-schema-sync",
	Short: "A CLI tool to synchronize MySQL database schemas",
	Long: `MySQL Schema Sync is a CLI application that compares two MySQL databases
(source and target) to detect schema differences, presents a summary of changes,
and applies approved changes to the target database.

Examples:
  # Compare schemas using command line flags
  mysql-schema-sync --source-host=localhost --source-user=root --source-db=source_db \
                    --target-host=localhost --target-user=root --target-db=target_db

  # Use configuration file
  mysql-schema-sync --config=config.yaml

  # Dry run mode (show changes without applying)
  mysql-schema-sync --config=config.yaml --dry-run

  # Auto-approve changes (no confirmation prompt)
  mysql-schema-sync --config=config.yaml --auto-approve

  # Verbose output
  mysql-schema-sync --config=config.yaml --verbose`,
	RunE: runSchemaSync,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configuration file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mysql-schema-sync.yaml)")

	// Source database flags
	rootCmd.Flags().StringVar(&sourceHost, "source-host", "", "source database host")
	rootCmd.Flags().IntVar(&sourcePort, "source-port", 3306, "source database port")
	rootCmd.Flags().StringVar(&sourceUsername, "source-user", "", "source database username")
	rootCmd.Flags().StringVar(&sourcePassword, "source-password", "", "source database password")
	rootCmd.Flags().StringVar(&sourceDatabase, "source-db", "", "source database name")

	// Target database flags
	rootCmd.Flags().StringVar(&targetHost, "target-host", "", "target database host")
	rootCmd.Flags().IntVar(&targetPort, "target-port", 3306, "target database port")
	rootCmd.Flags().StringVar(&targetUsername, "target-user", "", "target database username")
	rootCmd.Flags().StringVar(&targetPassword, "target-password", "", "target database password")
	rootCmd.Flags().StringVar(&targetDatabase, "target-db", "", "target database name")

	// Operation flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show changes without applying them")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	rootCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "automatically approve changes without confirmation")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "database operation timeout")
	rootCmd.Flags().StringVar(&logFile, "log-file", "", "write logs to file instead of stdout")

	// Bind flags to viper
	viper.BindPFlag("source.host", rootCmd.Flags().Lookup("source-host"))
	viper.BindPFlag("source.port", rootCmd.Flags().Lookup("source-port"))
	viper.BindPFlag("source.username", rootCmd.Flags().Lookup("source-user"))
	viper.BindPFlag("source.password", rootCmd.Flags().Lookup("source-password"))
	viper.BindPFlag("source.database", rootCmd.Flags().Lookup("source-db"))

	viper.BindPFlag("target.host", rootCmd.Flags().Lookup("target-host"))
	viper.BindPFlag("target.port", rootCmd.Flags().Lookup("target-port"))
	viper.BindPFlag("target.username", rootCmd.Flags().Lookup("target-user"))
	viper.BindPFlag("target.password", rootCmd.Flags().Lookup("target-password"))
	viper.BindPFlag("target.database", rootCmd.Flags().Lookup("target-db"))

	viper.BindPFlag("dry_run", rootCmd.Flags().Lookup("dry-run"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("quiet", rootCmd.Flags().Lookup("quiet"))
	viper.BindPFlag("auto_approve", rootCmd.Flags().Lookup("auto-approve"))
	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("log_file", rootCmd.Flags().Lookup("log-file"))

	// Mark required flags when not using config file
	rootCmd.MarkFlagsRequiredTogether("source-host", "source-user", "source-db", "target-host", "target-user", "target-db")

	// Add usage examples
	rootCmd.SetUsageTemplate(getUsageTemplate())
}

// runSchemaSync is the main execution function for the CLI
func runSchemaSync(cmd *cobra.Command, args []string) error {
	// Validate mutually exclusive flags
	if err := validateFlags(cmd); err != nil {
		return err
	}

	// Build configuration from flags and config file
	config, err := buildConfig(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create and run the application
	app, err := application.NewApplication(*config)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	return app.Run()
}

// validateFlags validates CLI flags and their combinations
func validateFlags(cmd *cobra.Command) error {
	// Check for mutually exclusive flags
	if verbose && quiet {
		return fmt.Errorf("--verbose and --quiet flags are mutually exclusive")
	}

	// If no config file is provided, ensure required connection flags are present
	if cfgFile == "" {
		missingFlags := []string{}

		if sourceHost == "" {
			missingFlags = append(missingFlags, "--source-host")
		}
		if sourceUsername == "" {
			missingFlags = append(missingFlags, "--source-user")
		}
		if sourceDatabase == "" {
			missingFlags = append(missingFlags, "--source-db")
		}
		if targetHost == "" {
			missingFlags = append(missingFlags, "--target-host")
		}
		if targetUsername == "" {
			missingFlags = append(missingFlags, "--target-user")
		}
		if targetDatabase == "" {
			missingFlags = append(missingFlags, "--target-db")
		}

		if len(missingFlags) > 0 {
			return fmt.Errorf("required flags missing: %v\nUse --config flag to specify a configuration file, or provide all required connection parameters", missingFlags)
		}
	}

	// Validate timeout
	if timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

// buildConfig builds the application configuration from CLI flags and config file
func buildConfig(cmd *cobra.Command) (*application.Config, error) {
	// Create configuration structure
	config := &application.Config{}

	// Load from viper (combines config file and CLI flags)
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Override with CLI flags if provided (viper binding should handle this, but explicit override for clarity)
	if sourceHost != "" {
		config.SourceDB.Host = sourceHost
	}
	if sourcePort != 3306 || cmd.Flags().Changed("source-port") {
		config.SourceDB.Port = sourcePort
	}
	if sourceUsername != "" {
		config.SourceDB.Username = sourceUsername
	}
	if sourcePassword != "" {
		config.SourceDB.Password = sourcePassword
	}
	if sourceDatabase != "" {
		config.SourceDB.Database = sourceDatabase
	}

	if targetHost != "" {
		config.TargetDB.Host = targetHost
	}
	if targetPort != 3306 || cmd.Flags().Changed("target-port") {
		config.TargetDB.Port = targetPort
	}
	if targetUsername != "" {
		config.TargetDB.Username = targetUsername
	}
	if targetPassword != "" {
		config.TargetDB.Password = targetPassword
	}
	if targetDatabase != "" {
		config.TargetDB.Database = targetDatabase
	}

	if cmd.Flags().Changed("dry-run") {
		config.DryRun = dryRun
	}
	if cmd.Flags().Changed("verbose") {
		config.Verbose = verbose
	}
	if cmd.Flags().Changed("quiet") {
		config.Quiet = quiet
	}
	if cmd.Flags().Changed("auto-approve") {
		config.AutoApprove = autoApprove
	}
	if cmd.Flags().Changed("timeout") {
		config.Timeout = timeout
	}
	if logFile != "" {
		config.LogFile = logFile
	}

	// Set default timeout if not specified
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Set default database timeouts
	if config.SourceDB.Timeout == 0 {
		config.SourceDB.Timeout = config.Timeout
	}
	if config.TargetDB.Timeout == 0 {
		config.TargetDB.Timeout = config.Timeout
	}

	// Validate configuration
	cliConfig := &database.CLIConfig{
		SourceDB:    config.SourceDB,
		TargetDB:    config.TargetDB,
		DryRun:      config.DryRun,
		Verbose:     config.Verbose,
		AutoApprove: config.AutoApprove,
	}

	if err := cliConfig.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".mysql-schema-sync" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".mysql-schema-sync")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("MYSQL_SCHEMA_SYNC")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// getUsageTemplate returns a custom usage template with examples
func getUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}

Configuration File:
  You can use a YAML configuration file instead of command-line flags:
  
  source:
    host: localhost
    port: 3306
    username: root
    password: secret
    database: source_db
  target:
    host: localhost
    port: 3306
    username: root
    password: secret
    database: target_db
  dry_run: false
  verbose: false
  auto_approve: false

Environment Variables:
  All configuration options can be set via environment variables with the prefix MYSQL_SCHEMA_SYNC_
  Example: MYSQL_SCHEMA_SYNC_SOURCE_HOST=localhost
`
}

// Version information (set by main package)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	goVersion = "unknown"
)

// SetVersionInfo sets the version information from build flags
func SetVersionInfo(v, bt, gc, gv string) {
	version = v
	buildTime = bt
	gitCommit = gc
	goVersion = gv
}

// createVersionCommand creates the version subcommand
func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  "Print the version information for mysql-schema-sync",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mysql-schema-sync version %s\n", version)
			fmt.Printf("Built: %s\n", buildTime)
			fmt.Printf("Commit: %s\n", gitCommit)
			fmt.Printf("Go version: %s\n", goVersion)
		},
	}
}

// createConfigCommand creates the config subcommand for generating sample config
func createConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Generate a sample configuration file",
		Long:  "Generate a sample configuration file that can be used with the --config flag",
		Run: func(cmd *cobra.Command, args []string) {
			sampleConfig := `# MySQL Schema Sync Configuration File
source:
  host: localhost
  port: 3306
  username: root
  password: ""
  database: source_db
  timeout: 30s

target:
  host: localhost
  port: 3306
  username: root
  password: ""
  database: target_db
  timeout: 30s

# Operation settings
dry_run: false      # Show changes without applying them
verbose: false      # Enable verbose output
quiet: false        # Suppress non-error output
auto_approve: false # Automatically approve changes without confirmation
timeout: 30s        # Global timeout for operations
log_file: ""        # Optional log file path
`
			fmt.Print(sampleConfig)
		},
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(createVersionCommand())
	rootCmd.AddCommand(createConfigCommand())
}
