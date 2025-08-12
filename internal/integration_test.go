package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"mysql-schema-sync/internal/database"
	"mysql-schema-sync/internal/logging"
	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"

	_ "github.com/go-sql-driver/mysql"
)

// Integration test configuration
type IntegrationTestConfig struct {
	SourceDB database.DatabaseConfig
	TargetDB database.DatabaseConfig
}

// TestIntegrationEndToEnd tests the complete workflow from schema extraction to migration
func TestIntegrationEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	config := getTestConfig(t)
	if config == nil {
		t.Skip("Integration test configuration not available")
	}

	// Create test databases
	sourceDB, targetDB := setupTestDatabases(t, config)
	defer cleanupTestDatabases(t, sourceDB, targetDB)

	// Create different schemas in source and target
	setupSourceSchema(t, sourceDB)
	setupTargetSchema(t, targetDB)

	// Test the complete workflow
	t.Run("Complete Schema Sync Workflow", func(t *testing.T) {
		testCompleteWorkflow(t, config)
	})

	// t.Run("CLI Application Integration", func(t *testing.T) {
	// 	testCLIIntegration(t, config)
	// })

	t.Run("Error Handling Integration", func(t *testing.T) {
		testErrorHandlingIntegration(t, config)
	})
}

func testCompleteWorkflow(t *testing.T, config *IntegrationTestConfig) {
	// Initialize services
	dbService := database.NewService()
	schemaService := schema.NewService()
	migrationService := migration.NewMigrationService()

	// Connect to databases
	sourceConn, err := dbService.Connect(config.SourceDB)
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer dbService.Close(sourceConn)

	targetConn, err := dbService.Connect(config.TargetDB)
	if err != nil {
		t.Fatalf("Failed to connect to target database: %v", err)
	}
	defer dbService.Close(targetConn)

	// Extract schemas
	sourceSchema, err := schemaService.ExtractSchemaFromDB(sourceConn, config.SourceDB.Database)
	if err != nil {
		t.Fatalf("Failed to extract source schema: %v", err)
	}

	targetSchema, err := schemaService.ExtractSchemaFromDB(targetConn, config.TargetDB.Database)
	if err != nil {
		t.Fatalf("Failed to extract target schema: %v", err)
	}

	// Compare schemas
	diff, err := schemaService.CompareSchemas(sourceSchema, targetSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Verify differences were detected
	if schemaService.IsSchemaDiffEmpty(diff) {
		t.Error("Expected schema differences but found none")
	}

	// Generate migration plan
	plan, err := migrationService.PlanMigration(diff)
	if err != nil {
		t.Fatalf("Failed to plan migration: %v", err)
	}

	// Validate plan
	if err := migrationService.ValidatePlan(plan); err != nil {
		t.Fatalf("Migration plan validation failed: %v", err)
	}

	// Generate SQL
	sqlStatements, err := migrationService.GenerateSQL(diff)
	if err != nil {
		t.Fatalf("Failed to generate SQL: %v", err)
	}

	if len(sqlStatements) == 0 {
		t.Error("Expected SQL statements to be generated")
	}

	// Apply changes (in a transaction for safety)
	tx, err := targetConn.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Always rollback in tests

	for _, stmt := range sqlStatements {
		if _, err := tx.Exec(stmt); err != nil {
			t.Logf("Failed to execute SQL: %s", stmt)
			t.Fatalf("SQL execution failed: %v", err)
		}
	}

	// Verify changes were applied correctly by re-extracting and comparing
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Re-extract target schema to verify changes
	updatedTargetSchema, err := schemaService.ExtractSchemaFromDB(targetConn, config.TargetDB.Database)
	if err != nil {
		t.Fatalf("Failed to extract updated target schema: %v", err)
	}

	// Compare again - should have fewer differences now
	newDiff, err := schemaService.CompareSchemas(sourceSchema, updatedTargetSchema)
	if err != nil {
		t.Fatalf("Failed to compare updated schemas: %v", err)
	}

	// The differences should be reduced (though may not be completely empty due to test setup)
	if len(newDiff.AddedTables) >= len(diff.AddedTables) &&
		len(newDiff.RemovedTables) >= len(diff.RemovedTables) &&
		len(newDiff.ModifiedTables) >= len(diff.ModifiedTables) {
		t.Error("Expected migration to reduce schema differences")
	}
}

// func testCLIIntegration(t *testing.T, config *IntegrationTestConfig) {
// 	// Test the CLI application integration
// 	app, _ := application.NewApplication(config)

// 	cliConfig := &database.CLIConfig{
// 		SourceDB: config.SourceDB,
// 		TargetDB: config.TargetDB,
// 		DryRun:   true, // Always use dry run in tests
// 		Verbose:  true,
// 	}

// 	// Test dry run mode
// 	result, err := app.ExecuteSchemaSync(cliConfig)
// 	if err != nil {
// 		t.Fatalf("CLI integration failed: %v", err)
// 	}

// 	if result == nil {
// 		t.Error("Expected result from CLI execution")
// 	}

// 	// Verify dry run didn't make actual changes
// 	if !cliConfig.DryRun {
// 		t.Error("Expected dry run mode to be preserved")
// 	}
// }

func testErrorHandlingIntegration(t *testing.T, config *IntegrationTestConfig) {
	dbService := database.NewService()

	// Test connection to non-existent database
	invalidConfig := config.SourceDB
	invalidConfig.Database = "non_existent_database_12345"

	_, err := dbService.Connect(invalidConfig)
	if err == nil {
		t.Error("Expected error when connecting to non-existent database")
	}

	// Test invalid host
	invalidConfig.Host = "invalid-host-that-does-not-exist"
	_, err = dbService.Connect(invalidConfig)
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}

	// Test schema extraction with invalid database
	validConn, err := dbService.Connect(config.SourceDB)
	if err != nil {
		t.Fatalf("Failed to connect to valid database: %v", err)
	}
	defer dbService.Close(validConn)

	schemaService := schema.NewService()
	_, err = schemaService.ExtractSchemaFromDB(validConn, "non_existent_schema")
	if err == nil {
		t.Error("Expected error when extracting non-existent schema")
	}
}

// Test database setup and cleanup functions

func getTestConfig(t *testing.T) *IntegrationTestConfig {
	// Check for environment variables or skip if not available
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 3306
	username := os.Getenv("MYSQL_TEST_USER")
	if username == "" {
		username = "root"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")
	if password == "" {
		password = "password"
	}

	// Try to connect to test if MySQL is available
	testConfig := database.DatabaseConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: "mysql", // Use system database for initial connection
		Timeout:  5 * time.Second,
	}

	db, err := sql.Open("mysql", testConfig.DSN())
	if err != nil {
		t.Logf("MySQL not available for integration tests: %v", err)
		return nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Logf("MySQL not available for integration tests: %v", err)
		return nil
	}

	return &IntegrationTestConfig{
		SourceDB: database.DatabaseConfig{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: "test_source_schema_sync",
			Timeout:  30 * time.Second,
		},
		TargetDB: database.DatabaseConfig{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: "test_target_schema_sync",
			Timeout:  30 * time.Second,
		},
	}
}

func setupTestDatabases(t *testing.T, config *IntegrationTestConfig) (*sql.DB, *sql.DB) {
	// Connect to MySQL server (not specific database)
	systemConfig := config.SourceDB
	systemConfig.Database = "mysql"

	systemDB, err := sql.Open("mysql", systemConfig.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to MySQL system database: %v", err)
	}
	defer systemDB.Close()

	// Create test databases
	_, err = systemDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.SourceDB.Database))
	if err != nil {
		t.Fatalf("Failed to create source test database: %v", err)
	}

	_, err = systemDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.TargetDB.Database))
	if err != nil {
		t.Fatalf("Failed to create target test database: %v", err)
	}

	// Connect to the test databases
	sourceDB, err := sql.Open("mysql", config.SourceDB.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to source test database: %v", err)
	}

	targetDB, err := sql.Open("mysql", config.TargetDB.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to target test database: %v", err)
	}

	return sourceDB, targetDB
}

func cleanupTestDatabases(t *testing.T, sourceDB, targetDB *sql.DB) {
	if sourceDB != nil {
		sourceDB.Close()
	}
	if targetDB != nil {
		targetDB.Close()
	}

	// Note: We don't drop the test databases here to avoid issues with concurrent tests
	// In a real CI environment, you might want to clean up or use unique database names
}

func setupSourceSchema(t *testing.T, db *sql.DB) {
	statements := []string{
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS users`,
		`CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			full_name VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE posts (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			status ENUM('draft', 'published', 'archived') DEFAULT 'draft',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_posts_status ON posts(status)`,
		`CREATE INDEX idx_posts_created ON posts(created_at)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("Failed to execute source schema statement: %s\nError: %v", stmt, err)
		}
	}
}

func setupTargetSchema(t *testing.T, db *sql.DB) {
	statements := []string{
		`DROP TABLE IF EXISTS comments`,
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS users`,
		`CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE posts (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			title VARCHAR(200) NOT NULL,
			body TEXT,
			published BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE comments (
			id INT PRIMARY KEY AUTO_INCREMENT,
			post_id INT NOT NULL,
			author_name VARCHAR(100) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_comments_post ON comments(post_id)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("Failed to execute target schema statement: %s\nError: %v", stmt, err)
		}
	}
}

// Benchmark integration tests
func BenchmarkIntegrationSchemaExtraction(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	config := getTestConfigForBenchmark(b)
	if config == nil {
		b.Skip("Integration test configuration not available")
	}

	dbService := database.NewService()
	schemaService := schema.NewService()

	conn, err := dbService.Connect(config.SourceDB)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbService.Close(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := schemaService.ExtractSchemaFromDB(conn, config.SourceDB.Database)
		if err != nil {
			b.Fatalf("Schema extraction failed: %v", err)
		}
	}
}

func BenchmarkIntegrationSchemaComparison(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	config := getTestConfigForBenchmark(b)
	if config == nil {
		b.Skip("Integration test configuration not available")
	}

	// Pre-extract schemas
	dbService := database.NewService()
	schemaService := schema.NewService()

	sourceConn, err := dbService.Connect(config.SourceDB)
	if err != nil {
		b.Fatalf("Failed to connect to source database: %v", err)
	}
	defer dbService.Close(sourceConn)

	targetConn, err := dbService.Connect(config.TargetDB)
	if err != nil {
		b.Fatalf("Failed to connect to target database: %v", err)
	}
	defer dbService.Close(targetConn)

	sourceSchema, err := schemaService.ExtractSchemaFromDB(sourceConn, config.SourceDB.Database)
	if err != nil {
		b.Fatalf("Failed to extract source schema: %v", err)
	}

	targetSchema, err := schemaService.ExtractSchemaFromDB(targetConn, config.TargetDB.Database)
	if err != nil {
		b.Fatalf("Failed to extract target schema: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := schemaService.CompareSchemas(sourceSchema, targetSchema)
		if err != nil {
			b.Fatalf("Schema comparison failed: %v", err)
		}
	}
}

func getTestConfigForBenchmark(b *testing.B) *IntegrationTestConfig {
	// Similar to getTestConfig but for benchmarks
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	username := os.Getenv("MYSQL_TEST_USER")
	if username == "" {
		username = "root"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")
	if password == "" {
		password = "password"
	}

	testConfig := database.DatabaseConfig{
		Host:     host,
		Port:     3306,
		Username: username,
		Password: password,
		Database: "mysql",
		Timeout:  5 * time.Second,
	}

	db, err := sql.Open("mysql", testConfig.DSN())
	if err != nil {
		return nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil
	}

	return &IntegrationTestConfig{
		SourceDB: database.DatabaseConfig{
			Host:     host,
			Port:     3306,
			Username: username,
			Password: password,
			Database: "test_source_schema_sync",
			Timeout:  30 * time.Second,
		},
		TargetDB: database.DatabaseConfig{
			Host:     host,
			Port:     3306,
			Username: username,
			Password: password,
			Database: "test_target_schema_sync",
			Timeout:  30 * time.Second,
		},
	}
}

// Test helper functions for integration testing
func TestIntegrationHelpers(t *testing.T) {
	// Test configuration validation
	t.Run("Config Validation", func(t *testing.T) {
		config := &IntegrationTestConfig{
			SourceDB: database.DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Database: "test_source",
			},
			TargetDB: database.DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Database: "test_target",
			},
		}

		if err := config.SourceDB.Validate(); err != nil {
			t.Errorf("Source config validation failed: %v", err)
		}

		if err := config.TargetDB.Validate(); err != nil {
			t.Errorf("Target config validation failed: %v", err)
		}
	})

	// Test logging integration
	t.Run("Logging Integration", func(t *testing.T) {
		logger := logging.NewDefaultLogger()
		if logger == nil {
			t.Error("Failed to create logger")
		}

		// Test that services can be created with logger
		schemaService := schema.NewServiceWithLogger(logger)
		if schemaService == nil {
			t.Error("Failed to create schema service with logger")
		}

		migrationService := migration.NewMigrationServiceWithLogger(logger)
		if migrationService == nil {
			t.Error("Failed to create migration service with logger")
		}
	})
}
