package display

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestProgressSystemDemo demonstrates the progress indication system
func TestProgressSystemDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo test in short mode")
	}
	// Create display service
	config := DefaultDisplayConfig()
	config.ColorEnabled = true
	ds := NewDisplayService(config)

	fmt.Println("=== Spinner Example ===")

	// Example 1: Simple spinner for database connection
	spinner := ds.StartSpinner("Connecting to database...")
	time.Sleep(1 * time.Second)

	ds.UpdateSpinner(spinner, "Establishing connection...")
	time.Sleep(500 * time.Millisecond)

	ds.StopSpinner(spinner, "✓ Connected to database")

	fmt.Println("\n=== Progress Bar Example ===")

	// Example 2: Progress bar for schema extraction
	pb := ds.NewProgressBar(10, "Extracting schema...")

	for i := 1; i <= 10; i++ {
		time.Sleep(100 * time.Millisecond)
		pb.Update(i, fmt.Sprintf("Processing table %d/10", i))
	}
	pb.Finish("✓ Schema extraction complete")

	fmt.Println("\n=== Multi-Progress Example ===")

	// Example 3: Multiple progress bars
	mp := ds.NewMultiProgress()

	pb1 := ds.NewProgressBar(5, "Extracting tables...")
	pb2 := ds.NewProgressBar(3, "Extracting indexes...")

	mp.AddBar(pb1)
	mp.AddBar(pb2)
	mp.Start()

	// Simulate concurrent operations
	for i := 1; i <= 5; i++ {
		time.Sleep(200 * time.Millisecond)
		pb1.Update(i, fmt.Sprintf("Table %d", i))

		if i <= 3 {
			pb2.Update(i, fmt.Sprintf("Index %d", i))
		}

		mp.Render()
	}

	mp.Stop()

	fmt.Println("\n=== Progress Tracker Example ===")

	// Example 4: Multi-phase progress tracker
	phases := []string{"Connect", "Extract", "Compare", "Apply"}
	pt := ds.NewProgressTracker(phases)

	// Phase 1: Connect
	pt.StartPhase(0, 3, "Connecting to source database...")
	time.Sleep(300 * time.Millisecond)
	pt.UpdatePhase(1, "Connecting to target database...")
	time.Sleep(300 * time.Millisecond)
	pt.UpdatePhase(2, "Validating connections...")
	time.Sleep(300 * time.Millisecond)
	pt.CompletePhase("✓ Database connections established")

	// Phase 2: Extract
	pt.StartPhase(1, 15, "Extracting source schema...")
	for i := 1; i <= 15; i++ {
		time.Sleep(50 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Processing object %d/15", i))
	}
	pt.CompletePhase("✓ Schema extraction complete")

	// Phase 3: Compare
	pt.StartPhase(2, 8, "Comparing schemas...")
	for i := 1; i <= 8; i++ {
		time.Sleep(100 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Comparing section %d/8", i))
	}
	pt.CompletePhase("✓ Schema comparison complete")

	// Phase 4: Apply
	pt.StartPhase(3, 5, "Applying changes...")
	for i := 1; i <= 5; i++ {
		time.Sleep(200 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Executing statement %d/5", i))
	}
	pt.CompletePhase("✓ All changes applied successfully")

	fmt.Println("\n=== Simple Progress Example ===")

	// Example 5: Simple progress display
	for i := 1; i <= 10; i++ {
		time.Sleep(100 * time.Millisecond)
		ds.ShowProgress(i, 10, fmt.Sprintf("Processing item %d", i))
	}

	fmt.Println("\n=== All Examples Complete ===")
}

// TestDatabaseOperationProgressDemo demonstrates progress tracking for typical database operations
func TestDatabaseOperationProgressDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo test in short mode")
	}
	// This example shows how progress indicators would be used in actual database operations

	config := DefaultDisplayConfig()
	config.ColorEnabled = true
	ds := NewDisplayService(config)

	// Simulate a complete schema sync operation
	fmt.Println("Starting MySQL Schema Sync...")

	// Phase 1: Database Connection
	spinner := ds.StartSpinner("Connecting to databases...")
	time.Sleep(800 * time.Millisecond)
	ds.UpdateSpinner(spinner, "Validating credentials...")
	time.Sleep(400 * time.Millisecond)
	ds.StopSpinner(spinner, ds.RenderIconWithColor("success")+" Connected to both databases")

	// Phase 2: Schema Extraction with Progress Bar
	fmt.Println()
	pb := ds.NewProgressBar(25, "Extracting schema information...")

	schemaItems := []string{
		"users", "orders", "products", "categories", "reviews",
		"payments", "shipping", "inventory", "suppliers", "customers",
		"user_roles", "permissions", "sessions", "logs", "settings",
		"notifications", "messages", "files", "tags", "comments",
		"analytics", "reports", "backups", "migrations", "indexes",
	}

	for i, item := range schemaItems {
		time.Sleep(80 * time.Millisecond)
		pb.Update(i+1, fmt.Sprintf("Processing %s table", item))
	}
	pb.Finish(ds.RenderIconWithColor("success") + " Schema extraction complete")

	// Phase 3: Schema Comparison with Multi-Phase Tracker
	fmt.Println()
	phases := []string{"Tables", "Columns", "Indexes", "Constraints"}
	pt := ds.NewProgressTracker(phases)

	// Compare tables
	pt.StartPhase(0, 12, "Comparing table structures...")
	for i := 1; i <= 12; i++ {
		time.Sleep(150 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Analyzing table %d/12", i))
	}
	pt.CompletePhase(ds.RenderIconWithColor("success") + " Table comparison complete")

	// Compare columns
	pt.StartPhase(1, 45, "Comparing column definitions...")
	for i := 1; i <= 45; i++ {
		time.Sleep(40 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Checking column %d/45", i))
	}
	pt.CompletePhase(ds.RenderIconWithColor("success") + " Column comparison complete")

	// Compare indexes
	pt.StartPhase(2, 18, "Comparing index definitions...")
	for i := 1; i <= 18; i++ {
		time.Sleep(80 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Validating index %d/18", i))
	}
	pt.CompletePhase(ds.RenderIconWithColor("success") + " Index comparison complete")

	// Compare constraints
	pt.StartPhase(3, 8, "Comparing constraints...")
	for i := 1; i <= 8; i++ {
		time.Sleep(120 * time.Millisecond)
		pt.UpdatePhase(i, fmt.Sprintf("Checking constraint %d/8", i))
	}
	pt.CompletePhase(ds.RenderIconWithColor("success") + " Constraint comparison complete")

	// Phase 4: SQL Generation and Execution
	fmt.Println()
	spinner = ds.StartSpinner("Generating SQL statements...")
	time.Sleep(600 * time.Millisecond)
	ds.UpdateSpinner(spinner, "Optimizing execution order...")
	time.Sleep(400 * time.Millisecond)
	ds.StopSpinner(spinner, ds.RenderIconWithColor("success")+" SQL generation complete")

	// Execute SQL with progress
	fmt.Println()
	pb = ds.NewProgressBar(7, "Executing SQL statements...")

	statements := []string{
		"ALTER TABLE users ADD COLUMN email_verified BOOLEAN",
		"CREATE INDEX idx_users_email ON users(email)",
		"ALTER TABLE orders MODIFY COLUMN status VARCHAR(50)",
		"DROP INDEX old_idx_products ON products",
		"CREATE TABLE audit_log (id INT PRIMARY KEY AUTO_INCREMENT)",
		"ALTER TABLE categories ADD CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES categories(id)",
		"UPDATE schema_version SET version = '2.1.0'",
	}

	for i, stmt := range statements {
		time.Sleep(300 * time.Millisecond)
		pb.Update(i+1, fmt.Sprintf("Executing: %s...", stmt[:min(40, len(stmt))]))
	}
	pb.Finish(ds.RenderIconWithColor("success") + " All SQL statements executed successfully")

	// Final summary
	fmt.Println()
	ds.Success("Schema synchronization completed successfully!")
	ds.Info("7 changes applied to target database")
	ds.Info("Operation completed in simulated time")

}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// This demo can be run with: go test -run TestProgressSystemDemo
func init() {
	// Only run examples if explicitly requested
	if os.Getenv("RUN_EXAMPLES") == "1" {
		// Examples would run here in a real scenario
	}
}
