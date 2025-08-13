package display

import (
	"fmt"
	"os"
)

// ExampleDisplayService demonstrates basic usage of the display service
func ExampleDisplayService() {
	// Create a display service with default configuration
	service := NewDisplayService(nil)

	// Print a header
	service.PrintHeader("Database Schema Sync")

	// Print status messages
	service.Info("Connecting to database...")
	service.Success("Connected successfully")

	// Print a section with content
	service.PrintSection("Schema Changes", "Found 3 differences")

	// Print a table
	headers := []string{"Table", "Change Type", "Description"}
	rows := [][]string{
		{"users", "ADD", "New column: email_verified"},
		{"products", "MODIFY", "Changed column: price (decimal precision)"},
		{"orders", "DROP", "Removed index: idx_temp"},
	}
	service.PrintTable(headers, rows)

	// Print SQL statements
	sqlStatements := []string{
		"ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;",
		"ALTER TABLE products MODIFY COLUMN price DECIMAL(10,2);",
		"DROP INDEX idx_temp ON orders;",
	}
	service.PrintSQL(sqlStatements)

	// Print warning and error messages
	service.Warning("This operation will modify existing data")
	service.Error("Failed to connect to target database")
}

// ExampleColorSystem demonstrates color system usage
func ExampleColorSystem() {
	theme := DefaultColorTheme()
	colorSystem := NewColorSystem(theme)

	// Check if colors are supported
	if colorSystem.IsColorSupported() {
		fmt.Println("Terminal supports colors")
	} else {
		fmt.Println("Terminal does not support colors")
	}

	// Colorize text
	redText := colorSystem.Colorize("Error message", ColorRed)
	greenText := colorSystem.Colorize("Success message", ColorGreen)

	fmt.Println(redText)
	fmt.Println(greenText)
}

// ExampleDisplayConfig demonstrates configuration options
func ExampleDisplayConfig() {
	// Create custom configuration
	config := &DisplayConfig{
		ColorEnabled:    true,
		Theme:           "dark",
		OutputFormat:    string(FormatJSON),
		VerboseMode:     true,
		QuietMode:       false,
		InteractiveMode: true,
		Writer:          os.Stdout,
	}

	service := NewDisplayService(config)

	// This will output in JSON format
	service.PrintSection("Configuration", map[string]interface{}{
		"colors_enabled": config.ColorEnabled,
		"output_format":  config.OutputFormat,
		"verbose_mode":   config.VerboseMode,
	})
}
