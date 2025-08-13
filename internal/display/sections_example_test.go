package display

import (
	"fmt"
	"os"
	"testing"
)

func TestSectionFormatterExample(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	fmt.Println("\n=== Section Formatter Examples ===")

	colorSystem := NewColorSystem(DefaultColorTheme())
	iconSystem := NewIconSystem()
	formatter := NewSectionFormatter(colorSystem, iconSystem, DefaultColorTheme(), os.Stdout)

	// Example 1: Simple section with content
	fmt.Println("\n1. Simple Section:")
	section1 := NewSection("Database Connection")
	section1.SetContent("Successfully connected to MySQL database 'production'")
	formatter.RenderSection(section1)

	// Example 2: Section with statistics
	fmt.Println("\n2. Section with Statistics:")
	section2 := NewSection("Schema Analysis")
	stats := NewSectionStatistics()
	stats.ItemCount = 15
	stats.SuccessCount = 12
	stats.WarningCount = 2
	stats.ErrorCount = 1
	stats.AddCustomStat("Tables", 8)
	stats.AddCustomStat("Indexes", 23)
	section2.SetStatistics(stats)
	section2.SetContent("Schema analysis completed with some warnings")
	formatter.RenderSection(section2)

	// Example 3: Nested sections
	fmt.Println("\n3. Nested Sections:")
	parent := NewSection("Migration Summary")

	// Add table changes subsection
	tableChanges := NewSection("Table Changes")
	tableChanges.SetContent([]string{
		"users: Added column 'last_login'",
		"orders: Modified column 'status'",
		"products: Added index 'idx_category'",
	})
	parent.AddSubsection(tableChanges)

	// Add index changes subsection
	indexChanges := NewSection("Index Changes")
	indexStats := NewSectionStatistics()
	indexStats.ItemCount = 5
	indexStats.SuccessCount = 5
	indexChanges.SetStatistics(indexStats)
	indexChanges.SetContent(map[string]interface{}{
		"Added":    3,
		"Modified": 1,
		"Dropped":  1,
	})
	parent.AddSubsection(indexChanges)

	formatter.RenderSection(parent)

	// Example 4: Collapsible section (expanded)
	fmt.Println("\n4. Collapsible Section (Expanded):")
	collapsible := NewSection("SQL Statements")
	collapsible.SetCollapsible(true)
	collapsible.SetCollapsed(false)
	collapsible.SetContent([]string{
		"ALTER TABLE users ADD COLUMN last_login TIMESTAMP",
		"ALTER TABLE orders MODIFY COLUMN status ENUM('pending','completed','cancelled')",
		"CREATE INDEX idx_category ON products (category_id)",
	})
	formatter.RenderSection(collapsible)

	// Example 5: Collapsible section (collapsed)
	fmt.Println("\n5. Collapsible Section (Collapsed):")
	collapsed := NewSection("Detailed Logs")
	collapsed.SetCollapsible(true)
	collapsed.SetCollapsed(true)
	collapsed.SetContent("This content is hidden because the section is collapsed")
	formatter.RenderSection(collapsed)

	// Example 6: Multiple sections
	fmt.Println("\n6. Multiple Sections:")
	sections := []*Section{
		{
			Title:   "Pre-Migration Checks",
			Content: "All checks passed successfully",
			Statistics: &SectionStatistics{
				ItemCount:    5,
				SuccessCount: 5,
			},
		},
		{
			Title:   "Schema Changes Applied",
			Content: "7 changes applied to target database",
			Statistics: &SectionStatistics{
				ItemCount:    7,
				SuccessCount: 7,
				TotalSize:    1024,
			},
		},
		{
			Title:   "Post-Migration Validation",
			Content: "Schema validation completed",
			Statistics: &SectionStatistics{
				ItemCount:    3,
				SuccessCount: 2,
				WarningCount: 1,
			},
		},
	}
	formatter.RenderSections(sections)

	fmt.Println("\n=== Section Formatter Examples Complete ===")
}

func TestSQLHighlighterExample(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	fmt.Println("\n=== SQL Syntax Highlighting Examples ===")

	colorSystem := NewColorSystem(DefaultColorTheme())
	highlighter := NewSQLHighlighter(colorSystem, DefaultColorTheme())

	sqlStatements := []string{
		"SELECT id, name, email FROM users WHERE active = 1 AND created_at > '2023-01-01'",
		"CREATE TABLE audit_log (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, action VARCHAR(255), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
		"ALTER TABLE users ADD COLUMN last_login TIMESTAMP NULL, ADD INDEX idx_last_login (last_login)",
		"INSERT INTO users (name, email, status) VALUES ('John Doe', 'john@example.com', 'active')",
		"UPDATE orders SET status = 'completed' WHERE id IN (SELECT order_id FROM payments WHERE status = 'confirmed')",
	}

	for i, stmt := range sqlStatements {
		fmt.Printf("\n%d. SQL Statement with Highlighting:\n", i+1)
		highlighted := highlighter.HighlightStatement(stmt, i)
		fmt.Println(highlighted)
	}

	fmt.Println("\n=== SQL Syntax Highlighting Examples Complete ===")
}

func TestStructuredOutputIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration example test in short mode")
	}

	fmt.Println("\n=== Structured Output Integration Example ===")

	// Create display service
	config := DefaultDisplayConfig()
	displayService := NewDisplayService(config)

	// Example: Database migration report using structured sections
	fmt.Println("\nDatabase Migration Report:")

	// Create main section
	migrationReport := NewSection("MySQL Schema Migration Report")

	// Connection section
	connectionSection := NewSection("Database Connection")
	connectionStats := NewSectionStatistics()
	connectionStats.ItemCount = 2
	connectionStats.SuccessCount = 2
	connectionStats.AddCustomStat("Source", "production_db")
	connectionStats.AddCustomStat("Target", "staging_db")
	connectionSection.SetStatistics(connectionStats)
	connectionSection.SetContent("Successfully connected to both databases")
	migrationReport.AddSubsection(connectionSection)

	// Schema analysis section
	analysisSection := NewSection("Schema Analysis")
	analysisStats := NewSectionStatistics()
	analysisStats.ItemCount = 25
	analysisStats.SuccessCount = 23
	analysisStats.WarningCount = 2
	analysisStats.TotalSize = 2048576 // 2MB
	analysisSection.SetStatistics(analysisStats)

	// Add nested sections for different types of changes
	tableChanges := NewSection("Table Changes")
	tableChanges.SetContent(map[string]interface{}{
		"Added":    2,
		"Modified": 3,
		"Dropped":  0,
	})
	analysisSection.AddSubsection(tableChanges)

	columnChanges := NewSection("Column Changes")
	columnChanges.SetContent([]string{
		"users.last_login: Added TIMESTAMP column",
		"orders.status: Modified ENUM values",
		"products.description: Increased VARCHAR length",
	})
	analysisSection.AddSubsection(columnChanges)

	migrationReport.AddSubsection(analysisSection)

	// SQL generation section with collapsible SQL statements
	sqlSection := NewSection("Generated SQL Statements")
	sqlSection.SetCollapsible(true)
	sqlSection.SetCollapsed(false)
	sqlStats := NewSectionStatistics()
	sqlStats.ItemCount = 5
	sqlStats.SuccessCount = 5
	sqlSection.SetStatistics(sqlStats)

	// Use SQL highlighter for the content
	sqlHighlighter := displayService.NewSQLHighlighter()
	sqlStatements := []string{
		"ALTER TABLE users ADD COLUMN last_login TIMESTAMP NULL",
		"ALTER TABLE orders MODIFY COLUMN status ENUM('pending','processing','completed','cancelled')",
		"ALTER TABLE products MODIFY COLUMN description VARCHAR(1000)",
	}

	var highlightedSQL []string
	for _, stmt := range sqlStatements {
		highlightedSQL = append(highlightedSQL, sqlHighlighter.Highlight(stmt))
	}
	sqlSection.SetContent(highlightedSQL)
	migrationReport.AddSubsection(sqlSection)

	// Render the complete report
	displayService.RenderSection(migrationReport)

	// Show success message
	displayService.Success("Migration report generated successfully")
	displayService.Info("Use --format=json to export this report in JSON format")

	fmt.Println("\n=== Structured Output Integration Example Complete ===")
}
