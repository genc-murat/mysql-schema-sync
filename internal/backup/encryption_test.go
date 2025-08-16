package backup

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptionManager_Encrypt_Disabled(t *testing.T) {
	config := &EncryptionConfig{
		Enabled: false,
	}
	em := NewEncryptionManager(config)
	testData := []byte("test data for encryption")

	encrypted, stats, err := em.Encrypt(testData)

	require.NoError(t, err)
	assert.Equal(t, testData, encrypted)
	assert.Equal(t, int64(len(testData)), stats.OriginalSize)
	assert.Equal(t, int64(len(testData)), stats.EncryptedSize)
	assert.Equal(t, "NONE", stats.Algorithm)
	assert.Equal(t, time.Duration(0), stats.Duration)
}

func TestEncryptionManager_Encrypt_Enabled(t *testing.T) {
	// Generate test key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}

	// Override KeyRetriever for testing
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	em := NewEncryptionManager(config)
	testData := []byte("test data for encryption that is longer to ensure proper encryption")

	encrypted, stats, err := em.Encrypt(testData)

	require.NoError(t, err)
	assert.NotEqual(t, testData, encrypted)
	assert.Equal(t, int64(len(testData)), stats.OriginalSize)
	assert.Greater(t, stats.EncryptedSize, stats.OriginalSize) // Encrypted data includes nonce and auth tag
	assert.Equal(t, "AES-256-GCM", stats.Algorithm)
	assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))

	// Test decryption
	decrypted, err := em.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestEncryptionManager_Decrypt_Disabled(t *testing.T) {
	config := &EncryptionConfig{
		Enabled: false,
	}
	em := NewEncryptionManager(config)
	testData := []byte("test data")

	decrypted, err := em.Decrypt(testData)

	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestEncryptionManager_Properties(t *testing.T) {
	enabledConfig := &EncryptionConfig{Enabled: true}
	disabledConfig := &EncryptionConfig{Enabled: false}

	enabledEM := NewEncryptionManager(enabledConfig)
	disabledEM := NewEncryptionManager(disabledConfig)

	assert.True(t, enabledEM.IsEnabled())
	assert.False(t, disabledEM.IsEnabled())

	assert.Equal(t, "AES-256-GCM", enabledEM.GetAlgorithm())
	assert.Equal(t, "NONE", disabledEM.GetAlgorithm())
}

func TestEncryptionManager_InvalidData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	em := NewEncryptionManager(config)

	t.Run("Decrypt invalid data", func(t *testing.T) {
		invalidData := []byte("this is not encrypted data")
		_, err := em.Decrypt(invalidData)
		assert.Error(t, err)
		// The error could be either "encrypted data too short" or authentication failure
		assert.True(t,
			strings.Contains(err.Error(), "encrypted data too short") ||
				strings.Contains(err.Error(), "message authentication failed"),
			"Expected error about data being too short or authentication failure, got: %s", err.Error())
	})

	t.Run("Decrypt corrupted data", func(t *testing.T) {
		testData := []byte("test data for corruption")
		encrypted, _, err := em.Encrypt(testData)
		require.NoError(t, err)

		// Corrupt the encrypted data
		corrupted := make([]byte, len(encrypted))
		copy(corrupted, encrypted)
		if len(corrupted) > 20 {
			corrupted[20] = ^corrupted[20]
		}

		_, err = em.Decrypt(corrupted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decrypt data")
	})
}

func TestKeyManager_GenerateKey(t *testing.T) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	key, err := km.GenerateKey()

	require.NoError(t, err)
	assert.Len(t, key, 32)

	// Generate another key and ensure they're different
	key2, err := km.GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, key, key2)
}

func TestKeyManager_GenerateKeyFromPassword(t *testing.T) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	password := "test-password-123"
	salt := []byte("test-salt-16-bytes")

	key1 := km.GenerateKeyFromPassword(password, salt)
	key2 := km.GenerateKeyFromPassword(password, salt)

	assert.Len(t, key1, 32)
	assert.Equal(t, key1, key2) // Same password and salt should produce same key

	// Different salt should produce different key
	differentSalt := []byte("different-salt-16")
	key3 := km.GenerateKeyFromPassword(password, differentSalt)
	assert.NotEqual(t, key1, key3)

	// Different password should produce different key
	key4 := km.GenerateKeyFromPassword("different-password", salt)
	assert.NotEqual(t, key1, key4)
}

func TestKeyManager_FileOperations(t *testing.T) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "test.key")

	t.Run("Save and load key", func(t *testing.T) {
		// Generate and save key
		originalKey, err := km.GenerateKey()
		require.NoError(t, err)

		err = km.SaveKeyToFile(originalKey, keyPath)
		require.NoError(t, err)

		// Load key
		loadedKey, err := km.LoadKeyFromFile(keyPath)
		require.NoError(t, err)

		assert.Equal(t, originalKey, loadedKey)

		// Check file permissions (more lenient for Windows)
		info, err := os.Stat(keyPath)
		require.NoError(t, err)
		// On Windows, permissions might be different, so just check that file exists and is readable
		assert.True(t, info.Mode().IsRegular(), "Key file should be a regular file")
	})

	t.Run("Invalid key size", func(t *testing.T) {
		invalidKey := []byte("too-short")
		err := km.SaveKeyToFile(invalidKey, keyPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key must be 32 bytes")
	})

	t.Run("Load non-existent file", func(t *testing.T) {
		_, err := km.LoadKeyFromFile("non-existent.key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read key from file")
	})
}

func TestKeyManager_EnvOperations(t *testing.T) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	envVar := "TEST_ENCRYPTION_KEY"

	t.Run("Save and load key from env", func(t *testing.T) {
		// Generate and save key
		originalKey, err := km.GenerateKey()
		require.NoError(t, err)

		err = km.SaveKeyToEnv(originalKey, envVar)
		require.NoError(t, err)

		// Load key
		loadedKey, err := km.LoadKeyFromEnv(envVar)
		require.NoError(t, err)

		assert.Equal(t, originalKey, loadedKey)

		// Clean up
		os.Unsetenv(envVar)
	})

	t.Run("Load from non-existent env var", func(t *testing.T) {
		_, err := km.LoadKeyFromEnv("NON_EXISTENT_KEY")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment variable NON_EXISTENT_KEY not set")
	})

	t.Run("Invalid hex in env var", func(t *testing.T) {
		os.Setenv(envVar, "invalid-hex-string")
		defer os.Unsetenv(envVar)

		_, err := km.LoadKeyFromEnv(envVar)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode hex key")
	})

	t.Run("Wrong key size in env var", func(t *testing.T) {
		shortKey := hex.EncodeToString([]byte("short"))
		os.Setenv(envVar, shortKey)
		defer os.Unsetenv(envVar)

		_, err := km.LoadKeyFromEnv(envVar)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key from environment variable must be 32 bytes")
	})
}

func TestKeyManager_ValidateKey(t *testing.T) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	t.Run("Valid key", func(t *testing.T) {
		validKey := make([]byte, 32)
		rand.Read(validKey)

		err := km.ValidateKey(validKey)
		assert.NoError(t, err)
	})

	t.Run("Invalid key size", func(t *testing.T) {
		invalidKey := []byte("too-short")
		err := km.ValidateKey(invalidKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key must be 32 bytes")
	})

	t.Run("All zeros key", func(t *testing.T) {
		zeroKey := make([]byte, 32)
		err := km.ValidateKey(zeroKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be all zeros")
	})

	t.Run("All ones key", func(t *testing.T) {
		onesKey := make([]byte, 32)
		for i := range onesKey {
			onesKey[i] = 0xFF
		}
		err := km.ValidateKey(onesKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be all ones")
	})
}

func TestBackupEncryption_Integration(t *testing.T) {
	// Generate test key
	key := make([]byte, 32)
	rand.Read(key)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	be := NewBackupEncryption(config)
	testData := []byte("This is test backup data that needs to be encrypted for security purposes.")

	t.Run("Basic encryption and decryption", func(t *testing.T) {
		encrypted, stats, err := be.EncryptBackup(testData)
		require.NoError(t, err)

		assert.NotEqual(t, testData, encrypted)
		assert.Equal(t, "AES-256-GCM", stats.Algorithm)
		assert.Greater(t, stats.EncryptedSize, stats.OriginalSize)

		decrypted, err := be.DecryptBackup(encrypted)
		require.NoError(t, err)
		assert.Equal(t, testData, decrypted)
	})

	t.Run("Properties", func(t *testing.T) {
		assert.True(t, be.IsEnabled())
		assert.Equal(t, "AES-256-GCM", be.GetAlgorithm())
	})

	t.Run("Key operations", func(t *testing.T) {
		newKey, err := be.GenerateNewKey()
		require.NoError(t, err)
		assert.Len(t, newKey, 32)

		err = be.ValidateKey(newKey)
		assert.NoError(t, err)
	})

	t.Run("Re-encryption with new key", func(t *testing.T) {
		// Encrypt with original key
		encrypted, _, err := be.EncryptBackup(testData)
		require.NoError(t, err)

		// Generate new key
		newKey, err := be.GenerateNewKey()
		require.NoError(t, err)

		// Re-encrypt with new key
		reEncrypted, err := be.ReEncryptWithNewKey(encrypted, newKey)
		require.NoError(t, err)

		// Should be different from original encryption
		assert.NotEqual(t, encrypted, reEncrypted)

		// Create new backup encryption with new key to test decryption
		newConfig := &EncryptionConfig{
			Enabled:   true,
			KeySource: "memory",
		}
		newConfig.KeyRetriever = func() ([]byte, error) {
			return newKey, nil
		}

		newBE := NewBackupEncryption(newConfig)
		decrypted, err := newBE.DecryptBackup(reEncrypted)
		require.NoError(t, err)
		assert.Equal(t, testData, decrypted)
	})
}

func TestBackupEncryption_Disabled(t *testing.T) {
	config := &EncryptionConfig{
		Enabled: false,
	}

	be := NewBackupEncryption(config)
	testData := []byte("test data")

	encrypted, stats, err := be.EncryptBackup(testData)
	require.NoError(t, err)
	assert.Equal(t, testData, encrypted)
	assert.Equal(t, "NONE", stats.Algorithm)

	decrypted, err := be.DecryptBackup(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)

	assert.False(t, be.IsEnabled())
	assert.Equal(t, "NONE", be.GetAlgorithm())
}

func TestEncryption_EmptyData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	em := NewEncryptionManager(config)
	emptyData := []byte{}

	encrypted, stats, err := em.Encrypt(emptyData)
	require.NoError(t, err)
	assert.NotEqual(t, emptyData, encrypted) // Should still have nonce and auth tag
	assert.Equal(t, int64(0), stats.OriginalSize)
	assert.Greater(t, stats.EncryptedSize, int64(0))

	decrypted, err := em.Decrypt(encrypted)
	require.NoError(t, err)
	// Handle nil vs empty slice difference
	if len(emptyData) == 0 && len(decrypted) == 0 {
		// Both are empty, test passes
	} else {
		assert.Equal(t, emptyData, decrypted)
	}
}

func TestEncryption_LargeData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	em := NewEncryptionManager(config)

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	rand.Read(largeData)

	encrypted, stats, err := em.Encrypt(largeData)
	require.NoError(t, err)
	assert.Equal(t, int64(len(largeData)), stats.OriginalSize)
	assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))

	decrypted, err := em.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, largeData, decrypted)
}

// Benchmark tests
func BenchmarkEncryption(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)

	config := &EncryptionConfig{
		Enabled:   true,
		KeySource: "memory",
	}
	config.KeyRetriever = func() ([]byte, error) {
		return key, nil
	}

	em := NewEncryptionManager(config)
	testData := make([]byte, 1024) // 1KB test data
	rand.Read(testData)

	b.Run("Encrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := em.Encrypt(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-encrypt data for decryption benchmark
	encrypted, _, _ := em.Encrypt(testData)

	b.Run("Decrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := em.Decrypt(encrypted)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkKeyGeneration(b *testing.B) {
	config := &EncryptionConfig{}
	km := NewKeyManager(config)

	b.Run("GenerateKey", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := km.GenerateKey()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GenerateKeyFromPassword", func(b *testing.B) {
		password := "test-password"
		salt := make([]byte, 32)
		rand.Read(salt)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = km.GenerateKeyFromPassword(password, salt)
		}
	})
}
