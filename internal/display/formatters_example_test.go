package display

import (
	"bytes"
	"fmt"
	"os"
)

func ExampleJSONFormatter() {
	formatter := NewJSONFormatter()

	// Format a section
	result, err := formatter.FormatSection("Database Schema", map[string]interface{}{
		"tables": 5,
		"views":  2,
		"status": "synchronized",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("JSON Section Output:")
	fmt.Println(result)

	// Format a table
	headers := []string{"Table", "Status", "Changes"}
	rows := [][]string{
		{"users", "modified", "2"},
		{"orders", "added", "0"},
		{"products", "unchanged", "0"},
	}

	tableResult, err := formatter.FormatTable(headers, rows)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nJSON Table Output:")
	fmt.Println(tableResult)

	// Output:
	// JSON Section Output:
	// {
	//   "content": {
	//     "status": "synchronized",
	//     "tables": 5,
	//     "views": 2
	//   },
	//   "section": "Database Schema"
	// }
	//
	// JSON Table Output:
	// [
	//   {
	//     "Changes": "2",
	//     "Status": "modified",
	//     "Table": "users"
	//   },
	//   {
	//     "Changes": "0",
	//     "Status": "added",
	//     "Table": "orders"
	//   },
	//   {
	//     "Changes": "0",
	//     "Status": "unchanged",
	//     "Table": "products"
	//   }
	// ]
}

func ExampleYAMLFormatter() {
	formatter := NewYAMLFormatter()

	// Format SQL statements
	statements := []string{
		"ALTER TABLE users ADD COLUMN email VARCHAR(255);",
		"CREATE INDEX idx_users_email ON users(email);",
		"UPDATE users SET email = 'unknown@example.com' WHERE email IS NULL;",
	}

	result, err := formatter.FormatSQL(statements)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("YAML SQL Output:")
	fmt.Println(result)

	// Output:
	// YAML SQL Output:
	// count: 3
	// sql_statements:
	//     - ALTER TABLE users ADD COLUMN email VARCHAR(255);
	//     - CREATE INDEX idx_users_email ON users(email);
	//     - UPDATE users SET email = 'unknown@example.com' WHERE email IS NULL;
}

func ExampleCompactFormatter() {
	formatter := NewCompactFormatter()

	// Format a section with structured data
	sectionResult, err := formatter.FormatSection("Migration Summary", map[string]interface{}{
		"tables_affected": 3,
		"changes":         5,
		"status":          "ready",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Compact Section Output:")
	fmt.Println(sectionResult)

	// Format a table in compact format
	headers := []string{"Table", "Action", "Priority"}
	rows := [][]string{
		{"users", "ALTER", "HIGH"},
		{"orders", "CREATE", "MEDIUM"},
		{"logs", "DROP", "LOW"},
	}

	result, err := formatter.FormatTable(headers, rows)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Compact Table Output:")
	fmt.Print(result)

	// Format SQL statements
	sqlResult, err := formatter.FormatSQL([]string{
		"ALTER TABLE users ADD COLUMN email VARCHAR(255);",
		"CREATE INDEX idx_users_email ON users(email);",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Compact SQL Output:")
	fmt.Println(sqlResult)

	// Format status messages
	statusResult, err := formatter.FormatStatusMessage("SUCCESS", "Schema synchronization completed")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Compact Status Output:")
	fmt.Println(statusResult)

	// Output:
	// Compact Section Output:
	// SECTION:Migration Summary:changes=5,status=ready,tables_affected=3
	// Compact Table Output:
	// Table	Action	Priority
	// users	ALTER	HIGH
	// orders	CREATE	MEDIUM
	// logs	DROP	LOW
	// Compact SQL Output:
	// SQL:2:ALTER TABLE users ADD COLUMN email VARCHAR(255);|CREATE INDEX idx_users_email ON users(email);
	// Compact Status Output:
	// STATUS:SUCCESS:Schema synchronization completed
}

func ExampleOutputWriter() {
	var buf bytes.Buffer
	writer := NewOutputWriter(FormatJSON, &buf)

	// Write a section
	err := writer.WriteSection("Migration Summary", map[string]interface{}{
		"total_changes":   5,
		"tables_affected": []string{"users", "orders", "products"},
		"estimated_time":  "2 minutes",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("JSON Output Writer Result:")
	fmt.Print(buf.String())

	// Switch to YAML format
	buf.Reset()
	writer.SetFormat(FormatYAML)

	err = writer.WriteStatusMessage("INFO", "Starting database migration")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("YAML Output Writer Result:")
	fmt.Print(buf.String())

	// Output:
	// JSON Output Writer Result:
	// {
	//   "content": {
	//     "estimated_time": "2 minutes",
	//     "tables_affected": [
	//       "users",
	//       "orders",
	//       "products"
	//     ],
	//     "total_changes": 5
	//   },
	//   "section": "Migration Summary"
	// }
	// YAML Output Writer Result:
	// level: INFO
	// message: Starting database migration
}

func ExampleDisplayService_outputFormats() {
	// Create display service with JSON output format
	config := &DisplayConfig{
		ColorEnabled:    false, // Disable colors for consistent output
		Theme:           "dark",
		OutputFormat:    string(FormatJSON),
		VerboseMode:     false,
		QuietMode:       false,
		InteractiveMode: false,
		Writer:          os.Stdout,
	}

	service := NewDisplayService(config)

	// Print a section in JSON format
	fmt.Println("=== JSON Format ===")
	service.PrintSection("Schema Analysis", map[string]interface{}{
		"tables_analyzed":   10,
		"differences_found": 3,
		"sync_required":     true,
	})

	// Switch to YAML format
	config.OutputFormat = string(FormatYAML)
	service.SetConfig(config)

	fmt.Println("=== YAML Format ===")
	service.PrintTable([]string{"Table", "Status"}, [][]string{
		{"users", "modified"},
		{"orders", "new"},
	})

	// Switch to compact format
	config.OutputFormat = string(FormatCompact)
	service.SetConfig(config)

	fmt.Println("=== Compact Format ===")
	service.Success("Migration completed successfully")

	// Output:
	// === JSON Format ===
	// {
	//   "content": {
	//     "differences_found": 3,
	//     "sync_required": true,
	//     "tables_analyzed": 10
	//   },
	//   "section": "Schema Analysis"
	// }
	//
	// === YAML Format ===
	// - Status: modified
	//   Table: users
	// - Status: new
	//   Table: orders
	//
	// === Compact Format ===
	// STATUS:SUCCESS:Migration completed successfully
}

func ExampleFormatterRegistry() {
	registry := NewFormatterRegistry()

	// Format table data using different formats
	tableData := map[string]interface{}{
		"headers": []string{"Database", "Tables", "Size"},
		"rows": [][]string{
			{"production", "25", "2.5GB"},
			{"staging", "25", "1.2GB"},
			{"development", "20", "500MB"},
		},
	}

	// JSON format
	jsonOutput, err := registry.FormatOutput(FormatJSON, "table", tableData)
	if err != nil {
		fmt.Printf("JSON Error: %v\n", err)
		return
	}

	fmt.Println("Registry JSON Output:")
	fmt.Println(jsonOutput)

	// Compact format
	compactOutput, err := registry.FormatOutput(FormatCompact, "table", tableData)
	if err != nil {
		fmt.Printf("Compact Error: %v\n", err)
		return
	}

	fmt.Println("Registry Compact Output:")
	fmt.Print(compactOutput)

	// Output:
	// Registry JSON Output:
	// [
	//   {
	//     "Database": "production",
	//     "Size": "2.5GB",
	//     "Tables": "25"
	//   },
	//   {
	//     "Database": "staging",
	//     "Size": "1.2GB",
	//     "Tables": "25"
	//   },
	//   {
	//     "Database": "development",
	//     "Size": "500MB",
	//     "Tables": "20"
	//   }
	// ]
	//
	// Registry Compact Output:
	// Database	Tables	Size
	// production	25	2.5GB
	// staging	25	1.2GB
	// development	20	500MB
}
