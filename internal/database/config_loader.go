package database

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigLoader handles loading configuration from various sources
type ConfigLoader struct {
	viper *viper.Viper
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		viper: viper.New(),
	}
}

// AddFlags adds CLI flags to the cobra command
func (cl *ConfigLoader) AddFlags(cmd *cobra.Command) {
	// Source database flags
	cmd.Flags().String("source-host", "", "Source database host")
	cmd.Flags().Int("source-port", 3306, "Source database port")
	cmd.Flags().String("source-username", "", "Source database username")
	cmd.Flags().String("source-password", "", "Source database password")
	cmd.Flags().String("source-database", "", "Source database name")

	// Target database flags
	cmd.Flags().String("target-host", "", "Target database host")
	cmd.Flags().Int("target-port", 3306, "Target database port")
	cmd.Flags().String("target-username", "", "Target database username")
	cmd.Flags().String("target-password", "", "Target database password")
	cmd.Flags().String("target-database", "", "Target database name")

	// General options
	cmd.Flags().Bool("dry-run", false, "Show changes without applying them")
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().Bool("auto-approve", false, "Automatically approve changes without confirmation")

	// Bind flags to viper
	cl.viper.BindPFlag("source.host", cmd.Flags().Lookup("source-host"))
	cl.viper.BindPFlag("source.port", cmd.Flags().Lookup("source-port"))
	cl.viper.BindPFlag("source.username", cmd.Flags().Lookup("source-username"))
	cl.viper.BindPFlag("source.password", cmd.Flags().Lookup("source-password"))
	cl.viper.BindPFlag("source.database", cmd.Flags().Lookup("source-database"))

	cl.viper.BindPFlag("target.host", cmd.Flags().Lookup("target-host"))
	cl.viper.BindPFlag("target.port", cmd.Flags().Lookup("target-port"))
	cl.viper.BindPFlag("target.username", cmd.Flags().Lookup("target-username"))
	cl.viper.BindPFlag("target.password", cmd.Flags().Lookup("target-password"))
	cl.viper.BindPFlag("target.database", cmd.Flags().Lookup("target-database"))

	cl.viper.BindPFlag("dry_run", cmd.Flags().Lookup("dry-run"))
	cl.viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	cl.viper.BindPFlag("auto_approve", cmd.Flags().Lookup("auto-approve"))
}

// LoadConfig loads configuration from file, environment variables, and CLI flags
func (cl *ConfigLoader) LoadConfig(configFile string) (*CLIConfig, error) {
	// Set config file if provided
	if configFile != "" {
		cl.viper.SetConfigFile(configFile)
	} else {
		// Set default config file locations
		cl.viper.SetConfigName("mysql-schema-sync")
		cl.viper.SetConfigType("yaml")
		cl.viper.AddConfigPath(".")
		cl.viper.AddConfigPath("$HOME/.config/mysql-schema-sync")
		cl.viper.AddConfigPath("$HOME")
	}

	// Enable environment variable support
	cl.viper.AutomaticEnv()
	cl.viper.SetEnvPrefix("MYSQL_SCHEMA_SYNC")

	// Try to read config file (it's okay if it doesn't exist)
	if err := cl.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal configuration
	var config CLIConfig
	if err := cl.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// GetUsedConfigFile returns the path of the config file that was used
func (cl *ConfigLoader) GetUsedConfigFile() string {
	return cl.viper.ConfigFileUsed()
}
