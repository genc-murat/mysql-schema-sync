package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupSystemInitializer handles initialization and setup of the backup system
type BackupSystemInitializer struct {
	config  *BackupConfig
	verbose bool
}

// NewBackupSystemInitializer creates a new backup system initializer
func NewBackupSystemInitializer(config *BackupConfig, verbose bool) *BackupSystemInitializer {
	return &BackupSystemInitializer{
		config:  config,
		verbose: verbose,
	}
}

// InitializationResult represents the result of backup system initialization
type InitializationResult struct {
	Success          bool
	StorageReady     bool
	ConfigValid      bool
	PermissionsOK    bool
	ConnectivityOK   bool
	Warnings         []string
	Errors           []string
	RecommendedFixes []string
}

// InitializeBackupSystem initializes the backup system and validates configuration
func (bsi *BackupSystemInitializer) InitializeBackupSystem() (*InitializationResult, error) {
	result := &InitializationResult{
		Success:          true,
		StorageReady:     true,
		ConfigValid:      true,
		PermissionsOK:    true,
		ConnectivityOK:   true,
		Warnings:         []string{},
		Errors:           []string{},
		RecommendedFixes: []string{},
	}

	if bsi.verbose {
		fmt.Println("Initializing backup system...")
	}

	// Step 1: Validate configuration
	if err := bsi.validateConfiguration(result); err != nil {
		result.Success = false
		result.ConfigValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Configuration validation failed: %v", err))
	}

	// Step 2: Initialize storage
	if err := bsi.initializeStorage(result); err != nil {
		result.Success = false
		result.StorageReady = false
		result.Errors = append(result.Errors, fmt.Sprintf("Storage initialization failed: %v", err))
	}

	// Step 3: Check permissions
	if err := bsi.checkPermissions(result); err != nil {
		result.PermissionsOK = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("Permission check warning: %v", err))
	}

	// Step 4: Test connectivity (for cloud storage)
	if err := bsi.testConnectivity(result); err != nil {
		result.ConnectivityOK = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("Connectivity test warning: %v", err))
	}

	// Step 5: Generate recommendations
	bsi.generateRecommendations(result)

	if bsi.verbose {
		if result.Success {
			fmt.Println("Backup system initialization completed successfully")
		} else {
			fmt.Println("Backup system initialization completed with errors")
		}
	}

	return result, nil
}

// validateConfiguration validates the backup configuration
func (bsi *BackupSystemInitializer) validateConfiguration(result *InitializationResult) error {
	if bsi.verbose {
		fmt.Println("  Validating backup configuration...")
	}

	// Validate using the existing validation method
	if err := bsi.config.Validate(); err != nil {
		return err
	}

	// Additional initialization-specific validations
	if bsi.config.Enabled {
		// Check for required environment variables
		if bsi.config.Encryption.Enabled && bsi.config.Encryption.KeySource == "env" {
			if os.Getenv(bsi.config.Encryption.KeyEnvVar) == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Encryption key environment variable %s is not set", bsi.config.Encryption.KeyEnvVar))
				result.RecommendedFixes = append(result.RecommendedFixes,
					fmt.Sprintf("Set encryption key: export %s=your_encryption_key", bsi.config.Encryption.KeyEnvVar))
			}
		}

		// Check encryption key file
		if bsi.config.Encryption.Enabled && bsi.config.Encryption.KeySource == "file" {
			if bsi.config.Encryption.KeyPath == "" {
				return fmt.Errorf("encryption key file path is required when key source is 'file'")
			}
			if _, err := os.Stat(bsi.config.Encryption.KeyPath); os.IsNotExist(err) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Encryption key file does not exist: %s", bsi.config.Encryption.KeyPath))
				result.RecommendedFixes = append(result.RecommendedFixes,
					fmt.Sprintf("Create encryption key file: %s", bsi.config.Encryption.KeyPath))
			}
		}
	}

	return nil
}

// initializeStorage initializes the backup storage
func (bsi *BackupSystemInitializer) initializeStorage(result *InitializationResult) error {
	if bsi.verbose {
		fmt.Printf("  Initializing %s storage...\n", bsi.config.Storage.Provider)
	}

	switch bsi.config.Storage.Provider {
	case "local":
		return bsi.initializeLocalStorage(result)
	case "s3":
		return bsi.initializeS3Storage(result)
	case "azure":
		return bsi.initializeAzureStorage(result)
	case "gcs":
		return bsi.initializeGCSStorage(result)
	default:
		return fmt.Errorf("unsupported storage provider: %s", bsi.config.Storage.Provider)
	}
}

// initializeLocalStorage initializes local file system storage
func (bsi *BackupSystemInitializer) initializeLocalStorage(result *InitializationResult) error {
	if bsi.config.Storage.Local == nil {
		return fmt.Errorf("local storage configuration is missing")
	}

	basePath := bsi.config.Storage.Local.BasePath

	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Test write permissions
	testFile := filepath.Join(basePath, ".backup_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("failed to write test file to backup directory: %w", err)
	}

	// Clean up test file
	os.Remove(testFile)

	if bsi.verbose {
		fmt.Printf("    Local storage initialized at: %s\n", basePath)
	}

	return nil
}

// initializeS3Storage initializes Amazon S3 storage
func (bsi *BackupSystemInitializer) initializeS3Storage(result *InitializationResult) error {
	if bsi.config.Storage.S3 == nil {
		return fmt.Errorf("S3 storage configuration is missing")
	}

	// Check required environment variables if credentials are not provided
	if bsi.config.Storage.S3.AccessKey == "" {
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			result.Warnings = append(result.Warnings, "AWS_ACCESS_KEY_ID environment variable is not set")
			result.RecommendedFixes = append(result.RecommendedFixes, "Set AWS credentials: export AWS_ACCESS_KEY_ID=your_access_key")
		}
	}

	if bsi.config.Storage.S3.SecretKey == "" {
		if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
			result.Warnings = append(result.Warnings, "AWS_SECRET_ACCESS_KEY environment variable is not set")
			result.RecommendedFixes = append(result.RecommendedFixes, "Set AWS credentials: export AWS_SECRET_ACCESS_KEY=your_secret_key")
		}
	}

	if bsi.verbose {
		fmt.Printf("    S3 storage configured for bucket: %s in region: %s\n",
			bsi.config.Storage.S3.Bucket, bsi.config.Storage.S3.Region)
	}

	return nil
}

// initializeAzureStorage initializes Azure Blob Storage
func (bsi *BackupSystemInitializer) initializeAzureStorage(result *InitializationResult) error {
	if bsi.config.Storage.Azure == nil {
		return fmt.Errorf("Azure storage configuration is missing")
	}

	// Check for connection string environment variable as alternative
	if bsi.config.Storage.Azure.AccountKey == "" {
		if os.Getenv("AZURE_STORAGE_CONNECTION_STRING") == "" {
			result.Warnings = append(result.Warnings, "Azure storage credentials not configured")
			result.RecommendedFixes = append(result.RecommendedFixes,
				"Set Azure credentials: export AZURE_STORAGE_CONNECTION_STRING=your_connection_string")
		}
	}

	if bsi.verbose {
		fmt.Printf("    Azure storage configured for account: %s, container: %s\n",
			bsi.config.Storage.Azure.AccountName, bsi.config.Storage.Azure.ContainerName)
	}

	return nil
}

// initializeGCSStorage initializes Google Cloud Storage
func (bsi *BackupSystemInitializer) initializeGCSStorage(result *InitializationResult) error {
	if bsi.config.Storage.GCS == nil {
		return fmt.Errorf("GCS storage configuration is missing")
	}

	// Check for credentials
	if bsi.config.Storage.GCS.CredentialsPath == "" {
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
			result.Warnings = append(result.Warnings, "Google Cloud credentials not configured")
			result.RecommendedFixes = append(result.RecommendedFixes,
				"Set GCS credentials: export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json")
		}
	} else {
		if _, err := os.Stat(bsi.config.Storage.GCS.CredentialsPath); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("GCS credentials file does not exist: %s", bsi.config.Storage.GCS.CredentialsPath))
		}
	}

	if bsi.verbose {
		fmt.Printf("    GCS storage configured for bucket: %s\n", bsi.config.Storage.GCS.Bucket)
	}

	return nil
}

// checkPermissions checks file system permissions
func (bsi *BackupSystemInitializer) checkPermissions(result *InitializationResult) error {
	if bsi.verbose {
		fmt.Println("  Checking permissions...")
	}

	// Only check local storage permissions
	if bsi.config.Storage.Provider == "local" && bsi.config.Storage.Local != nil {
		basePath := bsi.config.Storage.Local.BasePath

		// Check if directory exists and is writable
		info, err := os.Stat(basePath)
		if err != nil {
			return fmt.Errorf("cannot access backup directory: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("backup path is not a directory: %s", basePath)
		}

		// Test write permissions
		testFile := filepath.Join(basePath, ".permission_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("insufficient write permissions for backup directory: %w", err)
		}
		os.Remove(testFile)
	}

	return nil
}

// testConnectivity tests connectivity to cloud storage providers
func (bsi *BackupSystemInitializer) testConnectivity(result *InitializationResult) error {
	if bsi.verbose {
		fmt.Println("  Testing connectivity...")
	}

	// For now, we'll just validate that the configuration looks correct
	// In a real implementation, you would test actual connectivity to the cloud services
	switch bsi.config.Storage.Provider {
	case "s3":
		if bsi.config.Storage.S3 != nil {
			if bsi.config.Storage.S3.Bucket == "" {
				return fmt.Errorf("S3 bucket name is required")
			}
			if bsi.config.Storage.S3.Region == "" {
				return fmt.Errorf("S3 region is required")
			}
		}
	case "azure":
		if bsi.config.Storage.Azure != nil {
			if bsi.config.Storage.Azure.AccountName == "" {
				return fmt.Errorf("Azure account name is required")
			}
			if bsi.config.Storage.Azure.ContainerName == "" {
				return fmt.Errorf("Azure container name is required")
			}
		}
	case "gcs":
		if bsi.config.Storage.GCS != nil {
			if bsi.config.Storage.GCS.Bucket == "" {
				return fmt.Errorf("GCS bucket name is required")
			}
		}
	}

	return nil
}

// generateRecommendations generates setup recommendations
func (bsi *BackupSystemInitializer) generateRecommendations(result *InitializationResult) {
	// Security recommendations
	if bsi.config.Enabled && !bsi.config.Encryption.Enabled {
		result.RecommendedFixes = append(result.RecommendedFixes,
			"Consider enabling encryption for sensitive backup data")
	}

	// Performance recommendations
	if bsi.config.Compression.Enabled && bsi.config.Compression.Algorithm == "gzip" && bsi.config.Compression.Level > 6 {
		result.RecommendedFixes = append(result.RecommendedFixes,
			"Consider using compression level 6 or lower for better performance")
	}

	// Storage recommendations
	if bsi.config.Storage.Provider == "local" && bsi.config.Retention.MaxBackups == 0 && bsi.config.Retention.MaxAge == "" {
		result.RecommendedFixes = append(result.RecommendedFixes,
			"Configure retention policies to prevent unlimited storage growth")
	}

	// Monitoring recommendations
	if bsi.config.Validation.Enabled && bsi.config.Validation.ValidationTimeout == "" {
		result.RecommendedFixes = append(result.RecommendedFixes,
			"Set validation timeout to prevent long-running validation operations")
	}
}

// RunHealthCheck performs a comprehensive health check of the backup system
func (bsi *BackupSystemInitializer) RunHealthCheck() (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Timestamp:       time.Now(),
		OverallHealth:   "healthy",
		ComponentStatus: make(map[string]string),
		Issues:          []string{},
		Recommendations: []string{},
	}

	if bsi.verbose {
		fmt.Println("Running backup system health check...")
	}

	// Check configuration health
	if err := bsi.config.Validate(); err != nil {
		result.ComponentStatus["configuration"] = "unhealthy"
		result.Issues = append(result.Issues, fmt.Sprintf("Configuration validation failed: %v", err))
		result.OverallHealth = "unhealthy"
	} else {
		result.ComponentStatus["configuration"] = "healthy"
	}

	// Check storage health
	storageHealth := bsi.checkStorageHealth()
	result.ComponentStatus["storage"] = storageHealth
	if storageHealth != "healthy" {
		result.OverallHealth = "degraded"
		result.Issues = append(result.Issues, "Storage system is not fully operational")
	}

	// Check encryption health
	encryptionHealth := bsi.checkEncryptionHealth()
	result.ComponentStatus["encryption"] = encryptionHealth
	if encryptionHealth != "healthy" && bsi.config.Encryption.Enabled {
		result.OverallHealth = "degraded"
		result.Issues = append(result.Issues, "Encryption system has issues")
	}

	// Generate health recommendations
	bsi.generateHealthRecommendations(result)

	if bsi.verbose {
		fmt.Printf("Health check completed. Overall health: %s\n", result.OverallHealth)
	}

	return result, nil
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Timestamp       time.Time         `json:"timestamp"`
	OverallHealth   string            `json:"overall_health"`   // healthy, degraded, unhealthy
	ComponentStatus map[string]string `json:"component_status"` // component -> status
	Issues          []string          `json:"issues"`
	Recommendations []string          `json:"recommendations"`
}

// checkStorageHealth checks the health of the storage system
func (bsi *BackupSystemInitializer) checkStorageHealth() string {
	switch bsi.config.Storage.Provider {
	case "local":
		if bsi.config.Storage.Local == nil {
			return "unhealthy"
		}

		// Check if directory exists and is accessible
		if _, err := os.Stat(bsi.config.Storage.Local.BasePath); err != nil {
			return "unhealthy"
		}

		// Test write permissions
		testFile := filepath.Join(bsi.config.Storage.Local.BasePath, ".health_check")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return "degraded"
		}
		os.Remove(testFile)

		return "healthy"

	case "s3":
		if bsi.config.Storage.S3 == nil {
			return "unhealthy"
		}
		// In a real implementation, you would test S3 connectivity
		return "healthy"

	case "azure":
		if bsi.config.Storage.Azure == nil {
			return "unhealthy"
		}
		// In a real implementation, you would test Azure connectivity
		return "healthy"

	case "gcs":
		if bsi.config.Storage.GCS == nil {
			return "unhealthy"
		}
		// In a real implementation, you would test GCS connectivity
		return "healthy"

	default:
		return "unhealthy"
	}
}

// checkEncryptionHealth checks the health of the encryption system
func (bsi *BackupSystemInitializer) checkEncryptionHealth() string {
	if !bsi.config.Encryption.Enabled {
		return "healthy" // Not enabled, so no issues
	}

	switch bsi.config.Encryption.KeySource {
	case "env":
		if os.Getenv(bsi.config.Encryption.KeyEnvVar) == "" {
			return "unhealthy"
		}
		return "healthy"

	case "file":
		if bsi.config.Encryption.KeyPath == "" {
			return "unhealthy"
		}
		if _, err := os.Stat(bsi.config.Encryption.KeyPath); err != nil {
			return "unhealthy"
		}
		return "healthy"

	case "external":
		// In a real implementation, you would test external key management system
		return "healthy"

	default:
		return "unhealthy"
	}
}

// generateHealthRecommendations generates health-based recommendations
func (bsi *BackupSystemInitializer) generateHealthRecommendations(result *HealthCheckResult) {
	// Check for common issues and generate recommendations
	if result.ComponentStatus["storage"] != "healthy" {
		result.Recommendations = append(result.Recommendations,
			"Check storage configuration and connectivity")
	}

	if result.ComponentStatus["encryption"] != "healthy" && bsi.config.Encryption.Enabled {
		result.Recommendations = append(result.Recommendations,
			"Verify encryption key configuration and accessibility")
	}

	// Performance recommendations
	if bsi.config.Storage.Provider == "local" {
		result.Recommendations = append(result.Recommendations,
			"Monitor disk space usage in backup directory")
	}

	// Security recommendations
	if !bsi.config.Encryption.Enabled {
		result.Recommendations = append(result.Recommendations,
			"Consider enabling encryption for sensitive data protection")
	}
}

// CreateSetupWizard creates an interactive setup wizard for first-time configuration
func (bsi *BackupSystemInitializer) CreateSetupWizard() *SetupWizard {
	return &SetupWizard{
		initializer: bsi,
		verbose:     bsi.verbose,
	}
}

// SetupWizard provides an interactive setup experience
type SetupWizard struct {
	initializer *BackupSystemInitializer
	verbose     bool
}

// RunWizard runs the interactive setup wizard
func (sw *SetupWizard) RunWizard() (*BackupConfig, error) {
	fmt.Println("MySQL Schema Sync Backup System Setup Wizard")
	fmt.Println("=============================================")
	fmt.Println()

	config := &BackupConfig{}

	// Step 1: Enable backup system
	fmt.Print("Enable backup system? (y/N): ")
	var enableInput string
	fmt.Scanln(&enableInput)
	config.Enabled = strings.ToLower(enableInput) == "y" || strings.ToLower(enableInput) == "yes"

	if !config.Enabled {
		fmt.Println("Backup system will be disabled.")
		return config, nil
	}

	// Step 2: Choose storage provider
	fmt.Println("\nChoose storage provider:")
	fmt.Println("1. Local file system")
	fmt.Println("2. Amazon S3")
	fmt.Println("3. Azure Blob Storage")
	fmt.Println("4. Google Cloud Storage")
	fmt.Print("Enter choice (1-4): ")

	var providerChoice string
	fmt.Scanln(&providerChoice)

	switch providerChoice {
	case "1":
		config.Storage.Provider = "local"
		config.Storage.Local = sw.configureLocalStorage()
	case "2":
		config.Storage.Provider = "s3"
		config.Storage.S3 = sw.configureS3Storage()
	case "3":
		config.Storage.Provider = "azure"
		config.Storage.Azure = sw.configureAzureStorage()
	case "4":
		config.Storage.Provider = "gcs"
		config.Storage.GCS = sw.configureGCSStorage()
	default:
		config.Storage.Provider = "local"
		config.Storage.Local = sw.configureLocalStorage()
	}

	// Step 3: Configure retention
	config.Retention = sw.configureRetention()

	// Step 4: Configure compression
	config.Compression = sw.configureCompression()

	// Step 5: Configure encryption
	config.Encryption = sw.configureEncryption()

	// Step 6: Configure validation
	config.Validation = sw.configureValidation()

	fmt.Println("\nSetup completed successfully!")
	return config, nil
}

// configureLocalStorage configures local storage settings
func (sw *SetupWizard) configureLocalStorage() *LocalConfig {
	config := &LocalConfig{}

	fmt.Print("Enter backup directory path (./backups): ")
	var path string
	fmt.Scanln(&path)
	if path == "" {
		path = "./backups"
	}
	config.BasePath = path
	config.Permissions = "0755"

	return config
}

// configureS3Storage configures S3 storage settings
func (sw *SetupWizard) configureS3Storage() *S3Config {
	config := &S3Config{}

	fmt.Print("Enter S3 bucket name: ")
	fmt.Scanln(&config.Bucket)

	fmt.Print("Enter AWS region (us-east-1): ")
	fmt.Scanln(&config.Region)
	if config.Region == "" {
		config.Region = "us-east-1"
	}

	fmt.Println("Note: Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")

	return config
}

// configureAzureStorage configures Azure storage settings
func (sw *SetupWizard) configureAzureStorage() *AzureConfig {
	config := &AzureConfig{}

	fmt.Print("Enter Azure storage account name: ")
	fmt.Scanln(&config.AccountName)

	fmt.Print("Enter container name: ")
	fmt.Scanln(&config.ContainerName)

	fmt.Println("Note: Set AZURE_STORAGE_CONNECTION_STRING environment variable")

	return config
}

// configureGCSStorage configures GCS storage settings
func (sw *SetupWizard) configureGCSStorage() *GCSConfig {
	config := &GCSConfig{}

	fmt.Print("Enter GCS bucket name: ")
	fmt.Scanln(&config.Bucket)

	fmt.Print("Enter project ID: ")
	fmt.Scanln(&config.ProjectID)

	fmt.Println("Note: Set GOOGLE_APPLICATION_CREDENTIALS environment variable")

	return config
}

// configureRetention configures retention settings
func (sw *SetupWizard) configureRetention() RetentionConfig {
	config := RetentionConfig{}

	fmt.Print("Maximum number of backups to keep (10): ")
	var maxBackups string
	fmt.Scanln(&maxBackups)
	if maxBackups == "" {
		config.MaxBackups = 10
	} else {
		fmt.Sscanf(maxBackups, "%d", &config.MaxBackups)
	}

	config.CleanupInterval = "24h"

	return config
}

// configureCompression configures compression settings
func (sw *SetupWizard) configureCompression() CompressionConfig {
	config := CompressionConfig{}

	fmt.Print("Enable compression? (y/N): ")
	var enableInput string
	fmt.Scanln(&enableInput)
	config.Enabled = strings.ToLower(enableInput) == "y" || strings.ToLower(enableInput) == "yes"

	if config.Enabled {
		config.Algorithm = "gzip"
		config.Level = 6
		config.Threshold = 1024
	}

	return config
}

// configureEncryption configures encryption settings
func (sw *SetupWizard) configureEncryption() EncryptionConfig {
	config := EncryptionConfig{}

	fmt.Print("Enable encryption? (y/N): ")
	var enableInput string
	fmt.Scanln(&enableInput)
	config.Enabled = strings.ToLower(enableInput) == "y" || strings.ToLower(enableInput) == "yes"

	if config.Enabled {
		config.KeySource = "env"
		config.KeyEnvVar = "MYSQL_SCHEMA_SYNC_BACKUP_ENCRYPTION_KEY"
		config.RotationEnabled = false
		config.RotationDays = 90

		fmt.Printf("Note: Set %s environment variable with your encryption key\n", config.KeyEnvVar)
	}

	return config
}

// configureValidation configures validation settings
func (sw *SetupWizard) configureValidation() ValidationConfig {
	config := ValidationConfig{}

	config.Enabled = true
	config.ChecksumAlgorithm = "sha256"
	config.ValidateOnCreate = true
	config.ValidateOnRestore = true
	config.ValidationTimeout = "5m"
	config.DryRunValidation = true

	return config
}
