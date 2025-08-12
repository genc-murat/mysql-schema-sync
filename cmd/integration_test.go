//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// TestCLIIntegration tests the CLI application end-to-end
func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI integration tests in short mode")
	}

	config := getCLITestConfig(t)
	if config == nil {
		t.Skip("CLI integration test configuration not available")
	}

	// Build the CLI application
	buildCLI(t)
	defer cleanupCLI(t)

	// Setup test databases
	setupCLITestDatabases(t, config)
	defer cleanupCLITestDatabases(t, config)

	t.Run("CLI Help Command", func(t *testing.T) {
		testCLIHelp(t)
	})

	t.Run("CLI Version Command", func(t *testing.T) {
		testCLIVersion(t)
	})

	t.Run("CLI Schema Sync Dry Run", func(t *testing.T) {
		testCLISchemaSyncDryRun(t, config)
	})

	t.Run("CLI Schema Sync with Verbose Output", func(t *testing.T) {
		testCLISchemaSyncVerbose(t, config)
	})

	t.Run("CLI Error Handling", func(t *testing.T) {
		testCLIErrorHandling(t, config)
	})

	t.Run("CLI Configuration File", func(t *testing.T) {
		testCLIConfigFile(t, config)
	})
}

type CLITestConfig struct {
	SourceHost     string
	SourcePort     string
	SourceUser     string
	SourcePassword string
	SourceDatabase string
	TargetHost     string
	TargetPort     string
	TargetUser     string
	TargetPassword string
	TargetDatabase string
}

func getCLITestConfig(t *testing.T) *CLITestConfig {
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("MYSQL_TEST_PORT")
	if port == "" {
		port = "3306"
	}

	user := os.Getenv("MYSQL_TEST_USER")
	if user == "" {
		user = "root"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")
	if password == "" {
		password = "password"
	}

	// Test MySQL connectivity
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Logf("MySQL not available for CLI integration tests: %v", err)
		return nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Logf("MySQL not available for CLI integration tests: %v", err)
		return nil
	}

	return &CLITestConfig{
		SourceHost:     host,
		SourcePort:     port,
		SourceUser:     user,
		SourcePassword: password,
		SourceDatabase: "cli_test_source",
		TargetHost:     host,
		TargetPort:     port,
		TargetUser:     user,
		TargetPassword: password,
		TargetDatabase: "cli_test_target",
	}
}

func buildCLI(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "mysql-schema-sync-test", ".")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI application: %v", err)
	}
}

func cleanupCLI(t *testing.T) {
	os.Remove("../mysql-schema-sync-test")
}

func setupCLITestDatabases(t *testing.T, config *CLITestConfig) {
	// Connect to MySQL server
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql",
		config.SourceUser, config.SourcePassword, config.SourceHost, config.SourcePort)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// Create test databases
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.SourceDatabase))
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.TargetDatabase))
	if err != nil {
		t.Fatalf("Failed to create target database: %v", err)
	}

	// Setup source schema
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.SourceUser, config.SourcePassword, config.SourceHost, config.SourcePort, config.SourceDatabase)
	sourceDB, err := sql.Open("mysql", sourceDSN)
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	sourceStatements := []string{
		`DROP TABLE IF EXISTS orders`,
		`DROP TABLE IF EXISTS customers`,
		`CREATE TABLE customers (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			phone VARCHAR(20),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id INT PRIMARY KEY AUTO_INCREMENT,
			customer_id INT NOT NULL,
			order_date DATE NOT NULL,
			total_amount DECIMAL(10,2) NOT NULL,
			status ENUM('pending', 'processing', 'shipped', 'delivered') DEFAULT 'pending',
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_orders_date ON orders(order_date)`,
		`CREATE INDEX idx_orders_status ON orders(status)`,
	}

	for _, stmt := range sourceStatements {
		if _, err := sourceDB.Exec(stmt); err != nil {
			t.Fatalf("Failed to setup source schema: %v", err)
		}
	}

	// Setup target schema (different from source)
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.TargetUser, config.TargetPassword, config.TargetHost, config.TargetPort, config.TargetDatabase)
	targetDB, err := sql.Open("mysql", targetDSN)
	if err != nil {
		t.Fatalf("Failed to connect to target database: %v", err)
	}
	defer targetDB.Close()

	targetStatements := []string{
		`DROP TABLE IF EXISTS orders`,
		`DROP TABLE IF EXISTS customers`,
		`CREATE TABLE customers (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id INT PRIMARY KEY AUTO_INCREMENT,
			customer_id INT NOT NULL,
			order_date DATE NOT NULL,
			amount DECIMAL(8,2) NOT NULL,
			FOREIGN KEY (customer_id) REFERENCES customers(id)
		)`,
	}

	for _, stmt := range targetStatements {
		if _, err := targetDB.Exec(stmt); err != nil {
			t.Fatalf("Failed to setup target schema: %v", err)
		}
	}
}

func cleanupCLITestDatabases(t *testing.T, config *CLITestConfig) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql",
		config.SourceUser, config.SourcePassword, config.SourceHost, config.SourcePort)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return // Best effort cleanup
	}
	defer db.Close()

	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", config.SourceDatabase))
	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", config.TargetDatabase))
}

func testCLIHelp(t *testing.T) {
	cmd := exec.Command("../mysql-schema-sync-test", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI help command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "mysql-schema-sync") {
		t.Error("Help output should contain application name")
	}

	if !strings.Contains(outputStr, "Usage:") {
		t.Error("Help output should contain usage information")
	}
}

func testCLIVersion(t *testing.T) {
	cmd := exec.Command("../mysql-schema-sync-test", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI version command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if len(strings.TrimSpace(outputStr)) == 0 {
		t.Error("Version output should not be empty")
	}
}

func testCLISchemaSyncDryRun(t *testing.T, config *CLITestConfig) {
	args := []string{
		"--source-host", config.SourceHost,
		"--source-port", config.SourcePort,
		"--source-user", config.SourceUser,
		"--source-password", config.SourcePassword,
		"--source-database", config.SourceDatabase,
		"--target-host", config.TargetHost,
		"--target-port", config.TargetPort,
		"--target-user", config.TargetUser,
		"--target-password", config.TargetPassword,
		"--target-database", config.TargetDatabase,
		"--dry-run",
	}

	cmd := exec.Command("../mysql-schema-sync-test", args...)
	output, err := cmd.CombinedOutput()

	// The command might exit with non-zero status if differences are found
	// but that's expected behavior, not an error
	outputStr := string(output)

	if strings.Contains(outputStr, "panic") || strings.Contains(outputStr, "fatal error") {
		t.Fatalf("CLI dry run failed with panic/fatal error: %s", outputStr)
	}

	// Should contain some indication of dry run mode
	if !strings.Contains(outputStr, "dry") && !strings.Contains(outputStr, "DRY") {
		t.Logf("Warning: Dry run output doesn't clearly indicate dry run mode: %s", outputStr)
	}
}

func testCLISchemaSyncVerbose(t *testing.T, config *CLITestConfig) {
	args := []string{
		"--source-host", config.SourceHost,
		"--source-port", config.SourcePort,
		"--source-user", config.SourceUser,
		"--source-password", config.SourcePassword,
		"--source-database", config.SourceDatabase,
		"--target-host", config.TargetHost,
		"--target-port", config.TargetPort,
		"--target-user", config.TargetUser,
		"--target-password", config.TargetPassword,
		"--target-database", config.TargetDatabase,
		"--dry-run",
		"--verbose",
	}

	cmd := exec.Command("../mysql-schema-sync-test", args...)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if strings.Contains(outputStr, "panic") || strings.Contains(outputStr, "fatal error") {
		t.Fatalf("CLI verbose run failed with panic/fatal error: %s", outputStr)
	}

	// Verbose mode should produce more detailed output
	if len(outputStr) < 100 {
		t.Errorf("Verbose output seems too short, expected more detailed information: %s", outputStr)
	}
}

func testCLIErrorHandling(t *testing.T, config *CLITestConfig) {
	// Test with invalid source database
	args := []string{
		"--source-host", config.SourceHost,
		"--source-port", config.SourcePort,
		"--source-user", config.SourceUser,
		"--source-password", config.SourcePassword,
		"--source-database", "non_existent_database_12345",
		"--target-host", config.TargetHost,
		"--target-port", config.TargetPort,
		"--target-user", config.TargetUser,
		"--target-password", config.TargetPassword,
		"--target-database", config.TargetDatabase,
		"--dry-run",
	}

	cmd := exec.Command("../mysql-schema-sync-test", args...)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected error when using non-existent source database")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Error output should contain error message: %s", outputStr)
	}
}

func testCLIConfigFile(t *testing.T, config *CLITestConfig) {
	// Create a temporary config file
	configContent := fmt.Sprintf(`
source:
  host: %s
  port: %s
  username: %s
  password: %s
  database: %s

target:
  host: %s
  port: %s
  username: %s
  password: %s
  database: %s

options:
  dry_run: true
  verbose: false
`, config.SourceHost, config.SourcePort, config.SourceUser, config.SourcePassword, config.SourceDatabase,
		config.TargetHost, config.TargetPort, config.TargetUser, config.TargetPassword, config.TargetDatabase)

	configFile := "test-config.yaml"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	args := []string{
		"--config", configFile,
	}

	cmd := exec.Command("../mysql-schema-sync-test", args...)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if strings.Contains(outputStr, "panic") || strings.Contains(outputStr, "fatal error") {
		t.Fatalf("CLI config file test failed with panic/fatal error: %s", outputStr)
	}

	// Should be able to read config file without errors
	if strings.Contains(outputStr, "config") && strings.Contains(outputStr, "error") {
		t.Errorf("Config file reading failed: %s", outputStr)
	}
}

// Test CLI with different output formats
func TestCLIOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI output format tests in short mode")
	}

	config := getCLITestConfig(t)
	if config == nil {
		t.Skip("CLI integration test configuration not available")
	}

	buildCLI(t)
	defer cleanupCLI(t)

	setupCLITestDatabases(t, config)
	defer cleanupCLITestDatabases(t, config)

	baseArgs := []string{
		"--source-host", config.SourceHost,
		"--source-port", config.SourcePort,
		"--source-user", config.SourceUser,
		"--source-password", config.SourcePassword,
		"--source-database", config.SourceDatabase,
		"--target-host", config.TargetHost,
		"--target-port", config.TargetPort,
		"--target-user", config.TargetUser,
		"--target-password", config.TargetPassword,
		"--target-database", config.TargetDatabase,
		"--dry-run",
	}

	t.Run("Default Output Format", func(t *testing.T) {
		cmd := exec.Command("../mysql-schema-sync-test", baseArgs...)
		output, err := cmd.CombinedOutput()

		outputStr := string(output)
		if strings.Contains(outputStr, "panic") {
			t.Fatalf("Default output format failed: %s", outputStr)
		}
	})

	t.Run("Quiet Mode", func(t *testing.T) {
		args := append(baseArgs, "--quiet")
		cmd := exec.Command("../mysql-schema-sync-test", args...)
		output, err := cmd.CombinedOutput()

		outputStr := string(output)
		if strings.Contains(outputStr, "panic") {
			t.Fatalf("Quiet mode failed: %s", outputStr)
		}

		// Quiet mode should produce less output
		if len(outputStr) > 500 {
			t.Logf("Warning: Quiet mode output seems verbose: %s", outputStr)
		}
	})
}

// Benchmark CLI performance
func BenchmarkCLISchemaSync(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping CLI benchmarks in short mode")
	}

	config := getCLITestConfig(nil)
	if config == nil {
		b.Skip("CLI benchmark configuration not available")
	}

	// Build CLI once
	cmd := exec.Command("go", "build", "-o", "mysql-schema-sync-bench", ".")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build CLI for benchmark: %v", err)
	}
	defer os.Remove("../mysql-schema-sync-bench")

	// Setup test databases once
	setupCLITestDatabases(nil, config)
	defer cleanupCLITestDatabases(nil, config)

	args := []string{
		"--source-host", config.SourceHost,
		"--source-port", config.SourcePort,
		"--source-user", config.SourceUser,
		"--source-password", config.SourcePassword,
		"--source-database", config.SourceDatabase,
		"--target-host", config.TargetHost,
		"--target-port", config.TargetPort,
		"--target-user", config.TargetUser,
		"--target-password", config.TargetPassword,
		"--target-database", config.TargetDatabase,
		"--dry-run",
		"--quiet",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../mysql-schema-sync-bench", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			b.Fatalf("CLI benchmark run failed: %v\nStdout: %s\nStderr: %s",
				err, stdout.String(), stderr.String())
		}
	}
}
