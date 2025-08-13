package cmd

import (
	"fmt"
	"mysql-schema-sync/internal/application"
	"mysql-schema-sync/internal/database"
	"os"
	"strings"
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

	// Display flags
	noColor       bool
	theme         string
	outputFormat  string
	noIcons       bool
	noProgress    bool
	noInteractive bool
	tableStyle    string
	maxTableWidth int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mysql-schema-sync",
	Short: "A CLI tool to synchronize MySQL database schemas with enhanced visual output",
	Long: `MySQL Schema Sync is a CLI application that compares two MySQL databases
(source and target) to detect schema differences, presents a summary of changes
with enhanced visual formatting, and applies approved changes to the target database.

Features include colorized output, progress indicators, interactive confirmations,
multiple output formats, and accessibility support with graceful fallbacks.

Examples:
  # Basic schema comparison with enhanced visual output
  mysql-schema-sync --source-host=localhost --source-user=root --source-db=source_db \
                    --target-host=localhost --target-user=root --target-db=target_db

  # Use configuration file with custom theme
  mysql-schema-sync --config=config.yaml --theme=light

  # Dry run with JSON output format
  mysql-schema-sync --config=config.yaml --dry-run --format=json

  # Compact output for scripting (no colors, minimal formatting)
  mysql-schema-sync --config=config.yaml --format=compact --no-color --no-progress

  # High contrast mode for accessibility
  mysql-schema-sync --config=config.yaml --theme=high-contrast --no-icons

  # Verbose output with custom table styling
  mysql-schema-sync --config=config.yaml --verbose --table-style=rounded

  # Non-interactive mode for automation
  mysql-schema-sync --config=config.yaml --auto-approve --no-interactive`,
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

	// Display flags
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "disable color output")
	rootCmd.Flags().StringVar(&theme, "theme", "dark", "color theme (dark, light, high-contrast, auto)")
	rootCmd.Flags().StringVar(&outputFormat, "format", "table", "output format (table, json, yaml, compact)")
	rootCmd.Flags().BoolVar(&noIcons, "no-icons", false, "disable Unicode icons")
	rootCmd.Flags().BoolVar(&noProgress, "no-progress", false, "disable progress indicators")
	rootCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "disable interactive prompts")
	rootCmd.Flags().StringVar(&tableStyle, "table-style", "default", "table style (default, rounded, border, minimal)")
	rootCmd.Flags().IntVar(&maxTableWidth, "max-table-width", 120, "maximum table width (40-300)")

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

	// Bind display flags (only non-inverted ones)
	viper.BindPFlag("display.theme", rootCmd.Flags().Lookup("theme"))
	viper.BindPFlag("display.output_format", rootCmd.Flags().Lookup("format"))
	viper.BindPFlag("display.table_style", rootCmd.Flags().Lookup("table-style"))
	viper.BindPFlag("display.max_table_width", rootCmd.Flags().Lookup("max-table-width"))

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

// validateDisplayConfig validates display configuration options
func validateDisplayConfig(config *application.DisplayConfig) error {
	var errs []error

	// Validate theme
	validThemes := []string{"dark", "light", "high-contrast", "auto"}
	if !contains(validThemes, config.Theme) {
		errs = append(errs, fmt.Errorf("invalid theme '%s', must be one of: %s", config.Theme, strings.Join(validThemes, ", ")))
	}

	// Validate output format
	validFormats := []string{"table", "json", "yaml", "compact"}
	if !contains(validFormats, config.OutputFormat) {
		errs = append(errs, fmt.Errorf("invalid output format '%s', must be one of: %s", config.OutputFormat, strings.Join(validFormats, ", ")))
	}

	// Validate table style
	validTableStyles := []string{"default", "rounded", "border", "minimal"}
	if !contains(validTableStyles, config.TableStyle) {
		errs = append(errs, fmt.Errorf("invalid table style '%s', must be one of: %s", config.TableStyle, strings.Join(validTableStyles, ", ")))
	}

	// Validate max table width
	if config.MaxTableWidth < 40 || config.MaxTableWidth > 300 {
		errs = append(errs, fmt.Errorf("max table width must be between 40 and 300, got %d", config.MaxTableWidth))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// setDisplayDefaults sets default values for display configuration
func setDisplayDefaults(config *application.DisplayConfig) {
	if config.Theme == "" {
		config.Theme = "dark"
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "table"
	}
	if config.TableStyle == "" {
		config.TableStyle = "default"
	}
	if config.MaxTableWidth == 0 {
		config.MaxTableWidth = 120
	}
	// Set default boolean values if not explicitly set
	// Note: viper will handle these from config file or environment variables
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

	// Set display defaults if not loaded from config
	setDisplayDefaults(&config.Display)

	// Override with CLI flags if provided (handle inverted flags)
	if cmd.Flags().Changed("no-color") {
		config.Display.ColorEnabled = !noColor
	} else if !viper.IsSet("display.color_enabled") {
		config.Display.ColorEnabled = true // Default to enabled
	}

	if cmd.Flags().Changed("no-icons") {
		config.Display.UseIcons = !noIcons
	} else if !viper.IsSet("display.use_icons") {
		config.Display.UseIcons = true // Default to enabled
	}

	if cmd.Flags().Changed("no-progress") {
		config.Display.ShowProgress = !noProgress
	} else if !viper.IsSet("display.show_progress") {
		config.Display.ShowProgress = true // Default to enabled
	}

	if cmd.Flags().Changed("no-interactive") {
		config.Display.InteractiveMode = !noInteractive
	} else if !viper.IsSet("display.interactive") {
		config.Display.InteractiveMode = true // Default to enabled
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

	// Validate display configuration
	if err := validateDisplayConfig(&config.Display); err != nil {
		return nil, fmt.Errorf("display configuration validation failed: %w", err)
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

Database Connection Flags:
  --source-host string      Source database host
  --source-port int         Source database port (default 3306)
  --source-user string      Source database username
  --source-password string  Source database password
  --source-db string        Source database name
  --target-host string      Target database host
  --target-port int         Target database port (default 3306)
  --target-user string      Target database username
  --target-password string  Target database password
  --target-db string        Target database name

Operation Flags:
  --config string           Configuration file path
  --dry-run                 Show changes without applying them
  -v, --verbose             Enable verbose output
  -q, --quiet               Suppress non-error output
  --auto-approve            Automatically approve changes without confirmation
  --timeout duration        Database operation timeout (default 30s)
  --log-file string         Write logs to file instead of stdout

Visual Enhancement Flags:
  --no-color                Disable color output
  --theme string            Color theme: dark, light, high-contrast, auto (default "dark")
  --format string           Output format: table, json, yaml, compact (default "table")
  --no-icons                Disable Unicode icons (use ASCII alternatives)
  --no-progress             Disable progress indicators and spinners
  --no-interactive          Disable interactive prompts and confirmations
  --table-style string      Table style: default, rounded, border, minimal (default "default")
  --max-table-width int     Maximum table width in characters (40-300) (default 120)

{{.LocalFlags.FlagUsages}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}

Configuration File:
  Generate a sample configuration file with: mysql-schema-sync config
  
  Complete configuration example:
  
  source:
    host: localhost
    port: 3306
    username: root
    password: secret
    database: source_db
    timeout: 30s
  target:
    host: localhost
    port: 3306
    username: root
    password: secret
    database: target_db
    timeout: 30s
  dry_run: false
  verbose: false
  auto_approve: false
  timeout: 30s
  log_file: ""
  display:
    color_enabled: true        # Enable colorized output
    theme: dark               # Color theme (dark, light, high-contrast, auto)
    output_format: table      # Output format (table, json, yaml, compact)
    use_icons: true           # Enable Unicode icons with ASCII fallbacks
    show_progress: true       # Show progress bars and spinners
    interactive: true         # Enable interactive confirmations
    table_style: default      # Table styling (default, rounded, border, minimal)
    max_table_width: 120      # Maximum table width (40-300)

Visual Themes:
  dark           - Dark theme with bright colors (default)
  light          - Light theme with darker colors
  high-contrast  - High contrast theme for accessibility
  auto           - Automatically detect terminal theme

Output Formats:
  table          - Formatted tables with colors and styling (default)
  json           - Machine-readable JSON output
  yaml           - Human-readable YAML output
  compact        - Minimal output for scripting and automation

Environment Variables:
  All configuration options can be set via environment variables with the prefix MYSQL_SCHEMA_SYNC_
  Examples:
    MYSQL_SCHEMA_SYNC_SOURCE_HOST=localhost
    MYSQL_SCHEMA_SYNC_THEME=light
    MYSQL_SCHEMA_SYNC_FORMAT=json
    MYSQL_SCHEMA_SYNC_NO_COLOR=1
    MYSQL_SCHEMA_SYNC_NO_ICONS=1

Accessibility Features:
  - Automatic color detection and graceful fallback to plain text
  - Unicode icon fallback to ASCII characters
  - High contrast theme for visual impairments
  - Screen reader friendly output in compact format
  - Configurable table width for narrow terminals
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
		Long: `Generate a sample configuration file that can be used with the --config flag.

This command outputs a complete configuration template with all available options
including the new visual enhancement settings. You can redirect the output to a
file and customize it for your environment.

Examples:
  # Generate basic config file
  mysql-schema-sync config > config.yaml
  
  # Generate config with comments for all options
  mysql-schema-sync config > my-config.yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			sampleConfig := `# MySQL Schema Sync Configuration File
# Complete configuration template with all available options

# Source database connection
source:
  host: localhost          # Source database hostname or IP
  port: 3306              # Source database port
  username: root          # Source database username
  password: ""            # Source database password (use env var for security)
  database: source_db     # Source database name
  timeout: 30s            # Connection timeout for source database

# Target database connection
target:
  host: localhost          # Target database hostname or IP
  port: 3306              # Target database port
  username: root          # Target database username
  password: ""            # Target database password (use env var for security)
  database: target_db     # Target database name
  timeout: 30s            # Connection timeout for target database

# Operation settings
dry_run: false            # Show changes without applying them
verbose: false            # Enable verbose output with detailed information
quiet: false              # Suppress non-error output (mutually exclusive with verbose)
auto_approve: false       # Automatically approve changes without confirmation
timeout: 30s              # Global timeout for operations
log_file: ""              # Optional log file path (empty = stdout)

# Visual enhancement settings
display:
  # Color and theming
  color_enabled: true     # Enable colorized output
  theme: dark             # Color theme options:
                         #   - dark: Dark theme with bright colors (default)
                         #   - light: Light theme with darker colors
                         #   - high-contrast: High contrast for accessibility
                         #   - auto: Automatically detect terminal theme
  
  # Output formatting
  output_format: table    # Output format options:
                         #   - table: Formatted tables with colors (default)
                         #   - json: Machine-readable JSON output
                         #   - yaml: Human-readable YAML output
                         #   - compact: Minimal output for scripting
  
  # Visual elements
  use_icons: true         # Enable Unicode icons with ASCII fallbacks
  show_progress: true     # Show progress bars and loading spinners
  interactive: true       # Enable interactive confirmations and prompts
  
  # Table formatting
  table_style: default    # Table style options:
                         #   - default: Standard ASCII table borders
                         #   - rounded: Rounded corners (Unicode required)
                         #   - border: Heavy borders for emphasis
                         #   - minimal: Minimal borders for clean look
  max_table_width: 120    # Maximum table width in characters (40-300)

# Security recommendations:
# 1. Store passwords in environment variables:
#    export MYSQL_SCHEMA_SYNC_SOURCE_PASSWORD=your_password
#    export MYSQL_SCHEMA_SYNC_TARGET_PASSWORD=your_password
# 2. Set restrictive file permissions: chmod 600 config.yaml
# 3. Use dedicated database users with minimal required privileges
# 4. Consider SSL connections for remote databases

# Environment variable examples:
# MYSQL_SCHEMA_SYNC_SOURCE_HOST=prod-db.example.com
# MYSQL_SCHEMA_SYNC_TARGET_HOST=staging-db.example.com
# MYSQL_SCHEMA_SYNC_THEME=light
# MYSQL_SCHEMA_SYNC_FORMAT=json
# MYSQL_SCHEMA_SYNC_NO_COLOR=1
# MYSQL_SCHEMA_SYNC_NO_ICONS=1
# MYSQL_SCHEMA_SYNC_NO_PROGRESS=1
# MYSQL_SCHEMA_SYNC_NO_INTERACTIVE=1
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
