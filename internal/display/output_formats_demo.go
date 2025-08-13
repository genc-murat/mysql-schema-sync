package display

import (
	"fmt"
	"os"
)

// DemoOutputFormats demonstrates the different output formats available
func DemoOutputFormats() {
	fmt.Println("=== MySQL Schema Sync Output Format Demo ===")
	fmt.Println()

	// Sample data for demonstration
	headers := []string{"Table", "Operation", "Status", "Changes"}
	rows := [][]string{
		{"users", "ALTER", "SUCCESS", "2"},
		{"orders", "CREATE", "SUCCESS", "0"},
		{"products", "DROP", "WARNING", "1"},
	}

	sqlStatements := []string{
		"ALTER TABLE users ADD COLUMN last_login TIMESTAMP;",
		"CREATE TABLE orders (id INT PRIMARY KEY, user_id INT);",
		"DROP TABLE products;",
	}

	sectionData := map[string]interface{}{
		"total_changes":   3,
		"tables_affected": 3,
		"estimated_time":  "30 seconds",
		"safe_mode":       true,
	}

	// Demonstrate JSON format
	fmt.Println("1. JSON Format (for APIs and structured data processing):")
	fmt.Println("=" + fmt.Sprintf("%*s", 60, "="))

	config := &DisplayConfig{
		ColorEnabled:    false,
		Theme:           "dark",
		OutputFormat:    string(FormatJSON),
		VerboseMode:     false,
		QuietMode:       false,
		InteractiveMode: false,
		Writer:          os.Stdout,
	}

	service := NewDisplayService(config)
	service.PrintSection("Migration Summary", sectionData)
	service.PrintTable(headers, rows)
	service.Success("Migration completed successfully")
	fmt.Println()

	// Demonstrate YAML format
	fmt.Println("2. YAML Format (for configuration and human-readable structured data):")
	fmt.Println("=" + fmt.Sprintf("%*s", 70, "="))

	config.OutputFormat = string(FormatYAML)
	service.SetConfig(config)
	service.PrintSection("Migration Summary", sectionData)
	service.PrintTable(headers, rows)
	service.PrintSQL(sqlStatements)
	fmt.Println()

	// Demonstrate Compact format
	fmt.Println("3. Compact Format (for scripting and automation):")
	fmt.Println("=" + fmt.Sprintf("%*s", 50, "="))

	config.OutputFormat = string(FormatCompact)
	service.SetConfig(config)
	service.PrintSection("Migration Summary", sectionData)
	service.PrintTable(headers, rows)
	service.PrintSQL(sqlStatements)
	service.Success("Migration completed successfully")
	service.Warning("Some tables had warnings")
	service.Error("One operation failed")
	fmt.Println()

	// Demonstrate Table format (default)
	fmt.Println("4. Table Format (default, for human-readable terminal output):")
	fmt.Println("=" + fmt.Sprintf("%*s", 65, "="))

	config.OutputFormat = string(FormatTable)
	config.ColorEnabled = false // Disable colors for consistent demo output
	service.SetConfig(config)
	service.PrintHeader("MySQL Schema Migration Report")
	service.PrintSection("Migration Summary", "3 tables affected, 3 operations completed")
	service.PrintTable(headers, rows)
	service.Success("Migration completed successfully")
	fmt.Println()

	// Demonstrate custom compact formatter options
	fmt.Println("5. Custom Compact Format (CSV-style for spreadsheet import):")
	fmt.Println("=" + fmt.Sprintf("%*s", 60, "="))

	csvFormatter := NewCompactFormatterWithOptions(",", true)
	csvOutput, _ := csvFormatter.FormatTable(headers, rows)
	fmt.Print(csvOutput)
	fmt.Println()

	// Demonstrate headerless compact format
	fmt.Println("6. Headerless Compact Format (for data-only processing):")
	fmt.Println("=" + fmt.Sprintf("%*s", 55, "="))

	headerlessFormatter := NewCompactFormatterWithOptions("\t", false)
	headerlessOutput, _ := headerlessFormatter.FormatTable(headers, rows)
	fmt.Print(headerlessOutput)
	fmt.Println()

	fmt.Println("=== Demo Complete ===")
	fmt.Println()
	fmt.Println("Usage Examples:")
	fmt.Println("  mysql-schema-sync --format=json > migration.json")
	fmt.Println("  mysql-schema-sync --format=yaml > migration.yaml")
	fmt.Println("  mysql-schema-sync --format=compact | grep '^STATUS:' | cut -d: -f2-")
	fmt.Println("  mysql-schema-sync --format=table  # Default human-readable output")
}

// DemoScriptingCapabilities demonstrates how the compact format can be used for automation
func DemoScriptingCapabilities() {
	fmt.Println("=== Scripting and Automation Capabilities ===")
	fmt.Println()

	formatter := NewCompactFormatter()

	// Demonstrate parsing different output types
	fmt.Println("1. Parsing Section Information:")
	sectionOutput, _ := formatter.FormatSection("Database Status", map[string]interface{}{
		"connected": true,
		"tables":    25,
		"version":   "8.0.33",
	})
	fmt.Printf("Output: %s\n", sectionOutput)
	fmt.Println("Parsing: Extract key-value pairs after 'SECTION:Database Status:'")
	fmt.Println("Example: connected=true, tables=25, version=8.0.33")
	fmt.Println()

	// Demonstrate parsing status messages
	fmt.Println("2. Parsing Status Messages:")
	statusOutput, _ := formatter.FormatStatusMessage("ERROR", "Connection timeout after 30 seconds")
	fmt.Printf("Output: %s\n", statusOutput)
	fmt.Println("Parsing: Split by ':' to get [STATUS, ERROR, Connection timeout after 30 seconds]")
	fmt.Println("Shell: echo 'STATUS:ERROR:Connection timeout' | cut -d: -f2  # Gets 'ERROR'")
	fmt.Println()

	// Demonstrate parsing SQL statements
	fmt.Println("3. Parsing SQL Statements:")
	sqlOutput, _ := formatter.FormatSQL([]string{
		"CREATE TABLE test (id INT);",
		"INSERT INTO test VALUES (1);",
	})
	fmt.Printf("Output: %s\n", sqlOutput)
	fmt.Println("Parsing: Extract count and statements separated by '|'")
	fmt.Println("Shell: echo 'SQL:2:stmt1|stmt2' | cut -d: -f3 | tr '|' '\\n'  # Gets individual statements")
	fmt.Println()

	// Demonstrate parsing table data
	fmt.Println("4. Parsing Table Data (TSV):")
	tableOutput, _ := formatter.FormatTable(
		[]string{"Table", "Status", "Changes"},
		[][]string{
			{"users", "modified", "2"},
			{"orders", "added", "0"},
		},
	)
	fmt.Printf("Output:\n%s", tableOutput)
	fmt.Println("Parsing: Standard TSV format, easily processed by awk, cut, or imported into spreadsheets")
	fmt.Println("Shell: mysql-schema-sync --format=compact | awk -F'\\t' '{print $1, $3}'  # Gets table and changes")
	fmt.Println()

	// Demonstrate schema diff parsing
	fmt.Println("5. Parsing Schema Differences:")
	diffOutput, _ := formatter.FormatSchemaDiff(map[string]interface{}{
		"added":    []string{"table1", "table2"},
		"removed":  []string{"table3"},
		"modified": []interface{}{"table4"},
	})
	fmt.Printf("Output: %s\n", diffOutput)
	fmt.Println("Parsing: Extract change counts after 'DIFF:schema:'")
	fmt.Println("Example: added=2, removed=1, modified=1")
	fmt.Println()

	fmt.Println("=== Automation Script Examples ===")
	fmt.Println()

	fmt.Println("Bash script to check migration status:")
	fmt.Println("```bash")
	fmt.Println("#!/bin/bash")
	fmt.Println("OUTPUT=$(mysql-schema-sync --format=compact)")
	fmt.Println("ERRORS=$(echo \"$OUTPUT\" | grep '^STATUS:ERROR:' | wc -l)")
	fmt.Println("if [ $ERRORS -gt 0 ]; then")
	fmt.Println("  echo \"Migration failed with $ERRORS errors\"")
	fmt.Println("  echo \"$OUTPUT\" | grep '^STATUS:ERROR:' | cut -d: -f3-")
	fmt.Println("  exit 1")
	fmt.Println("fi")
	fmt.Println("echo \"Migration completed successfully\"")
	fmt.Println("```")
	fmt.Println()

	fmt.Println("Python script to parse migration results:")
	fmt.Println("```python")
	fmt.Println("import subprocess")
	fmt.Println("import sys")
	fmt.Println("")
	fmt.Println("result = subprocess.run(['mysql-schema-sync', '--format=compact'], ")
	fmt.Println("                       capture_output=True, text=True)")
	fmt.Println("")
	fmt.Println("for line in result.stdout.strip().split('\\\\n'):")
	fmt.Println("    if line.startswith('STATUS:'):")
	fmt.Println("        parts = line.split(':', 2)")
	fmt.Println("        level, message = parts[1], parts[2]")
	fmt.Println("        print(f'{level}: {message}')")
	fmt.Println("    elif line.startswith('DIFF:schema:'):")
	fmt.Println("        changes = line.split(':', 2)[2]")
	fmt.Println("        print(f'Schema changes: {changes}')")
	fmt.Println("```")

	fmt.Println()
	fmt.Println("=== Scripting Demo Complete ===")
}
