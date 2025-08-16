package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionStats contains statistics about encryption operations
type EncryptionStats struct {
	OriginalSize  int64         `json:"original_size"`
	EncryptedSize int64         `json:"encrypted_size"`
	Algorithm     string        `json:"algorithm"`
	KeyDerivation string        `json:"key_derivation"`
	Duration      time.Duration `json:"duration"`
}

// EncryptionManager manages encryption operations
type EncryptionManager struct {
	config *EncryptionConfig
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager(config *EncryptionConfig) *EncryptionManager {
	return &EncryptionManager{
		config: config,
	}
}

// Encrypt encrypts data using AES-256-GCM
func (em *EncryptionManager) Encrypt(data []byte) ([]byte, *EncryptionStats, error) {
	if !em.config.Enabled {
		return data, &EncryptionStats{
			OriginalSize:  int64(len(data)),
			EncryptedSize: int64(len(data)),
			Algorithm:     "NONE",
			Duration:      0,
		}, nil
	}

	start := time.Now()

	// Get encryption key
	key, err := em.config.GetEncryptionKey()
	if err != nil {
		return nil, nil, NewEncryptionError("failed to get encryption key", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, NewEncryptionError("failed to create AES cipher", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, NewEncryptionError("failed to create GCM cipher", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, NewEncryptionError("failed to generate nonce", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	duration := time.Since(start)

	stats := &EncryptionStats{
		OriginalSize:  int64(len(data)),
		EncryptedSize: int64(len(ciphertext)),
		Algorithm:     "AES-256-GCM",
		KeyDerivation: em.config.KeySource,
		Duration:      duration,
	}

	return ciphertext, stats, nil
}

// Decrypt decrypts data using AES-256-GCM
func (em *EncryptionManager) Decrypt(encryptedData []byte) ([]byte, error) {
	if !em.config.Enabled {
		return encryptedData, nil
	}

	// Get encryption key
	key, err := em.config.GetEncryptionKey()
	if err != nil {
		return nil, NewEncryptionError("failed to get encryption key", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, NewEncryptionError("failed to create AES cipher", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, NewEncryptionError("failed to create GCM cipher", err)
	}

	// Check minimum size
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, NewEncryptionError("encrypted data too short", nil)
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, NewEncryptionError("failed to decrypt data", err)
	}

	return plaintext, nil
}

// IsEnabled returns whether encryption is enabled
func (em *EncryptionManager) IsEnabled() bool {
	return em.config.Enabled
}

// GetAlgorithm returns the encryption algorithm being used
func (em *EncryptionManager) GetAlgorithm() string {
	if !em.config.Enabled {
		return "NONE"
	}
	return "AES-256-GCM"
}

// KeyManager handles encryption key operations
type KeyManager struct {
	config *EncryptionConfig
}

// NewKeyManager creates a new key manager
func NewKeyManager(config *EncryptionConfig) *KeyManager {
	return &KeyManager{
		config: config,
	}
}

// GenerateKey generates a new 256-bit encryption key
func (km *KeyManager) GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := rand.Read(key); err != nil {
		return nil, NewEncryptionError("failed to generate encryption key", err)
	}
	return key, nil
}

// GenerateKeyFromPassword derives a key from a password using PBKDF2
func (km *KeyManager) GenerateKeyFromPassword(password string, salt []byte) []byte {
	if len(salt) == 0 {
		// Generate random salt if not provided
		salt = make([]byte, 32)
		rand.Read(salt)
	}

	// Use PBKDF2 with SHA-256, 100,000 iterations
	return pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
}

// SaveKeyToFile saves an encryption key to a file
func (km *KeyManager) SaveKeyToFile(key []byte, filepath string) error {
	// Ensure the key is the correct size
	if len(key) != 32 {
		return NewEncryptionError("key must be 32 bytes for AES-256", nil)
	}

	// Write key to file with restricted permissions
	if err := os.WriteFile(filepath, key, 0600); err != nil {
		return NewEncryptionError("failed to save key to file", err)
	}

	return nil
}

// LoadKeyFromFile loads an encryption key from a file
func (km *KeyManager) LoadKeyFromFile(filepath string) ([]byte, error) {
	key, err := os.ReadFile(filepath)
	if err != nil {
		return nil, NewEncryptionError("failed to read key from file", err)
	}

	if len(key) != 32 {
		return nil, NewEncryptionError("key file must contain 32 bytes for AES-256", nil)
	}

	return key, nil
}

// SaveKeyToEnv saves an encryption key to an environment variable (hex-encoded)
func (km *KeyManager) SaveKeyToEnv(key []byte, envVar string) error {
	if len(key) != 32 {
		return NewEncryptionError("key must be 32 bytes for AES-256", nil)
	}

	hexKey := hex.EncodeToString(key)
	if err := os.Setenv(envVar, hexKey); err != nil {
		return NewEncryptionError("failed to set environment variable", err)
	}

	return nil
}

// LoadKeyFromEnv loads an encryption key from an environment variable (hex-encoded)
func (km *KeyManager) LoadKeyFromEnv(envVar string) ([]byte, error) {
	hexKey := os.Getenv(envVar)
	if hexKey == "" {
		return nil, NewEncryptionError(fmt.Sprintf("environment variable %s not set", envVar), nil)
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, NewEncryptionError("failed to decode hex key from environment variable", err)
	}

	if len(key) != 32 {
		return nil, NewEncryptionError("key from environment variable must be 32 bytes for AES-256", nil)
	}

	return key, nil
}

// RotateKey generates a new key and updates the configuration
func (km *KeyManager) RotateKey() ([]byte, error) {
	// Generate new key
	newKey, err := km.GenerateKey()
	if err != nil {
		return nil, err
	}

	// Save new key based on configuration
	switch km.config.KeySource {
	case "env":
		if err := km.SaveKeyToEnv(newKey, km.config.KeyEnvVar); err != nil {
			return nil, err
		}
	case "file":
		if err := km.SaveKeyToFile(newKey, km.config.KeyPath); err != nil {
			return nil, err
		}
	case "external":
		// External key management would be implementation-specific
		return nil, NewEncryptionError("external key rotation not implemented", nil)
	default:
		return nil, NewEncryptionError("invalid key source for rotation", nil)
	}

	return newKey, nil
}

// ValidateKey validates that a key is suitable for AES-256
func (km *KeyManager) ValidateKey(key []byte) error {
	if len(key) != 32 {
		return NewEncryptionError("key must be 32 bytes for AES-256", nil)
	}

	// Check for weak keys (all zeros, all ones, etc.)
	allZeros := true
	allOnes := true
	for _, b := range key {
		if b != 0 {
			allZeros = false
		}
		if b != 0xFF {
			allOnes = false
		}
	}

	if allZeros {
		return NewEncryptionError("key cannot be all zeros", nil)
	}
	if allOnes {
		return NewEncryptionError("key cannot be all ones", nil)
	}

	return nil
}

// BackupEncryption provides high-level encryption operations for backups
type BackupEncryption struct {
	encryptionManager *EncryptionManager
	keyManager        *KeyManager
}

// NewBackupEncryption creates a new backup encryption instance
func NewBackupEncryption(config *EncryptionConfig) *BackupEncryption {
	return &BackupEncryption{
		encryptionManager: NewEncryptionManager(config),
		keyManager:        NewKeyManager(config),
	}
}

// EncryptBackup encrypts backup data
func (be *BackupEncryption) EncryptBackup(data []byte) ([]byte, *EncryptionStats, error) {
	return be.encryptionManager.Encrypt(data)
}

// DecryptBackup decrypts backup data
func (be *BackupEncryption) DecryptBackup(encryptedData []byte) ([]byte, error) {
	return be.encryptionManager.Decrypt(encryptedData)
}

// IsEnabled returns whether encryption is enabled
func (be *BackupEncryption) IsEnabled() bool {
	return be.encryptionManager.IsEnabled()
}

// GetAlgorithm returns the encryption algorithm
func (be *BackupEncryption) GetAlgorithm() string {
	return be.encryptionManager.GetAlgorithm()
}

// GenerateNewKey generates a new encryption key
func (be *BackupEncryption) GenerateNewKey() ([]byte, error) {
	return be.keyManager.GenerateKey()
}

// RotateKey rotates the encryption key
func (be *BackupEncryption) RotateKey() ([]byte, error) {
	return be.keyManager.RotateKey()
}

// ValidateKey validates an encryption key
func (be *BackupEncryption) ValidateKey(key []byte) error {
	return be.keyManager.ValidateKey(key)
}

// ReEncryptWithNewKey re-encrypts data with a new key
func (be *BackupEncryption) ReEncryptWithNewKey(encryptedData []byte, newKey []byte) ([]byte, error) {
	// First decrypt with current key
	plaintext, err := be.DecryptBackup(encryptedData)
	if err != nil {
		return nil, NewEncryptionError("failed to decrypt data for re-encryption", err)
	}

	// Create temporary encryption manager with new key
	tempConfig := *be.encryptionManager.config
	tempConfig.KeySource = "memory" // Temporary override
	tempManager := NewEncryptionManager(&tempConfig)

	// Override the key retrieval for this operation
	originalGetKey := tempConfig.KeyRetriever
	tempConfig.KeyRetriever = func() ([]byte, error) {
		return newKey, nil
	}

	// Encrypt with new key
	encrypted, _, err := tempManager.Encrypt(plaintext)
	if err != nil {
		return nil, NewEncryptionError("failed to encrypt data with new key", err)
	}

	// Restore original key retrieval function
	tempConfig.KeyRetriever = originalGetKey

	return encrypted, nil
}
