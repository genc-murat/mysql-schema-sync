package backup

import (
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressionManager_NewCompressionManager(t *testing.T) {
	cm := NewCompressionManager()

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.compressors)

	// Verify all expected compressors are registered
	expectedAlgorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	supportedAlgorithms := cm.GetSupportedAlgorithms()
	assert.Len(t, supportedAlgorithms, len(expectedAlgorithms))

	for _, expected := range expectedAlgorithms {
		found := false
		for _, supported := range supportedAlgorithms {
			if supported == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Algorithm %s should be supported", expected)
	}
}

func TestCompressionManager_Compress_None(t *testing.T) {
	cm := NewCompressionManager()
	testData := []byte("test data for compression")

	compressed, stats, err := cm.Compress(testData, CompressionTypeNone, 0)

	require.NoError(t, err)
	assert.Equal(t, testData, compressed)
	assert.Equal(t, int64(len(testData)), stats.OriginalSize)
	assert.Equal(t, int64(len(testData)), stats.CompressedSize)
	assert.Equal(t, 1.0, stats.CompressionRatio)
	assert.Equal(t, CompressionTypeNone, stats.Algorithm)
	assert.Equal(t, 0, stats.Level)
	assert.Equal(t, time.Duration(0), stats.Duration)
}

func TestCompressionManager_Compress_UnsupportedAlgorithm(t *testing.T) {
	cm := NewCompressionManager()
	testData := []byte("test data")

	_, _, err := cm.Compress(testData, CompressionType("INVALID"), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported compression algorithm")
}

func TestCompressionManager_Decompress_UnsupportedAlgorithm(t *testing.T) {
	cm := NewCompressionManager()
	testData := []byte("test data")

	_, err := cm.Decompress(testData, CompressionType("INVALID"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported compression algorithm")
}

func TestCompressionManager_ShouldCompress(t *testing.T) {
	cm := NewCompressionManager()

	tests := []struct {
		name      string
		dataSize  int64
		threshold int64
		expected  bool
	}{
		{"Below threshold", 500, 1024, false},
		{"At threshold", 1024, 1024, true},
		{"Above threshold", 2048, 1024, true},
		{"Zero threshold", 100, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.ShouldCompress(tt.dataSize, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGzipCompressor(t *testing.T) {
	compressor := &GzipCompressor{}
	testData := []byte(strings.Repeat("This is test data for compression. ", 100))

	t.Run("Basic compression and decompression", func(t *testing.T) {
		compressed, stats, err := compressor.Compress(testData, compressor.GetDefaultLevel())
		require.NoError(t, err)

		assert.Equal(t, CompressionTypeGzip, stats.Algorithm)
		assert.Equal(t, int64(len(testData)), stats.OriginalSize)
		assert.Equal(t, int64(len(compressed)), stats.CompressedSize)
		assert.Less(t, stats.CompressedSize, stats.OriginalSize)
		assert.Less(t, stats.CompressionRatio, 1.0)
		assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))

		decompressed, err := compressor.Decompress(compressed)
		require.NoError(t, err)
		assert.Equal(t, testData, decompressed)
	})

	t.Run("Different compression levels", func(t *testing.T) {
		levels := []int{compressor.GetMinLevel(), compressor.GetDefaultLevel(), compressor.GetMaxLevel()}

		for _, level := range levels {
			compressed, stats, err := compressor.Compress(testData, level)
			require.NoError(t, err)

			assert.Equal(t, level, stats.Level)
			assert.Less(t, stats.CompressedSize, stats.OriginalSize)

			decompressed, err := compressor.Decompress(compressed)
			require.NoError(t, err)
			assert.Equal(t, testData, decompressed)
		}
	})

	t.Run("Properties", func(t *testing.T) {
		assert.Equal(t, CompressionTypeGzip, compressor.GetAlgorithm())
		assert.Equal(t, 1, compressor.GetMinLevel())
		assert.Equal(t, 9, compressor.GetMaxLevel())
		assert.Equal(t, -1, compressor.GetDefaultLevel()) // gzip.DefaultCompression
	})
}

func TestLZ4Compressor(t *testing.T) {
	compressor := &LZ4Compressor{}
	testData := []byte(strings.Repeat("This is test data for LZ4 compression. ", 100))

	t.Run("Basic compression and decompression", func(t *testing.T) {
		compressed, stats, err := compressor.Compress(testData, compressor.GetDefaultLevel())
		require.NoError(t, err)

		assert.Equal(t, CompressionTypeLZ4, stats.Algorithm)
		assert.Equal(t, int64(len(testData)), stats.OriginalSize)
		assert.Equal(t, int64(len(compressed)), stats.CompressedSize)
		assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))

		decompressed, err := compressor.Decompress(compressed)
		require.NoError(t, err)
		assert.Equal(t, testData, decompressed)
	})

	t.Run("Different compression levels", func(t *testing.T) {
		levels := []int{1, 6, 12}

		for _, level := range levels {
			compressed, stats, err := compressor.Compress(testData, level)
			require.NoError(t, err)

			assert.Equal(t, level, stats.Level)

			decompressed, err := compressor.Decompress(compressed)
			require.NoError(t, err)
			assert.Equal(t, testData, decompressed)
		}
	})

	t.Run("Properties", func(t *testing.T) {
		assert.Equal(t, CompressionTypeLZ4, compressor.GetAlgorithm())
		assert.Equal(t, 1, compressor.GetMinLevel())
		assert.Equal(t, 12, compressor.GetMaxLevel())
		assert.Equal(t, 1, compressor.GetDefaultLevel())
	})
}

func TestZstdCompressor(t *testing.T) {
	compressor := &ZstdCompressor{}
	testData := []byte(strings.Repeat("This is test data for Zstandard compression. ", 100))

	t.Run("Basic compression and decompression", func(t *testing.T) {
		compressed, stats, err := compressor.Compress(testData, compressor.GetDefaultLevel())
		require.NoError(t, err)

		assert.Equal(t, CompressionTypeZstd, stats.Algorithm)
		assert.Equal(t, int64(len(testData)), stats.OriginalSize)
		assert.Equal(t, int64(len(compressed)), stats.CompressedSize)
		assert.Less(t, stats.CompressedSize, stats.OriginalSize)
		assert.Less(t, stats.CompressionRatio, 1.0)
		assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))

		decompressed, err := compressor.Decompress(compressed)
		require.NoError(t, err)
		assert.Equal(t, testData, decompressed)
	})

	t.Run("Different compression levels", func(t *testing.T) {
		levels := []int{1, 3, 10, 22}

		for _, level := range levels {
			compressed, stats, err := compressor.Compress(testData, level)
			require.NoError(t, err)

			assert.Equal(t, level, stats.Level)
			assert.Less(t, stats.CompressedSize, stats.OriginalSize)

			decompressed, err := compressor.Decompress(compressed)
			require.NoError(t, err)
			assert.Equal(t, testData, decompressed)
		}
	})

	t.Run("Properties", func(t *testing.T) {
		assert.Equal(t, CompressionTypeZstd, compressor.GetAlgorithm())
		assert.Equal(t, 1, compressor.GetMinLevel())
		assert.Equal(t, 22, compressor.GetMaxLevel())
		assert.Equal(t, 3, compressor.GetDefaultLevel())
	})
}

func TestCompressionAlgorithmComparison(t *testing.T) {
	// Create test data with patterns that compress well
	testData := []byte(strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 1000))

	cm := NewCompressionManager()
	algorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	results := make(map[CompressionType]*CompressionStats)

	for _, algorithm := range algorithms {
		compressor, err := cm.GetCompressor(algorithm)
		require.NoError(t, err)

		compressed, stats, err := compressor.Compress(testData, compressor.GetDefaultLevel())
		require.NoError(t, err)

		// Verify decompression works
		decompressed, err := compressor.Decompress(compressed)
		require.NoError(t, err)
		assert.Equal(t, testData, decompressed)

		results[algorithm] = stats

		t.Logf("Algorithm: %s, Original: %d bytes, Compressed: %d bytes, Ratio: %.3f, Duration: %v",
			algorithm, stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio, stats.Duration)
	}

	// All algorithms should achieve some compression
	for algorithm, stats := range results {
		assert.Less(t, stats.CompressionRatio, 1.0, "Algorithm %s should achieve compression", algorithm)
		assert.GreaterOrEqual(t, stats.Duration, time.Duration(0), "Algorithm %s should have non-negative duration", algorithm)
	}
}

func TestCompressionWithRandomData(t *testing.T) {
	// Random data typically doesn't compress well
	randomData := make([]byte, 10000)
	_, err := rand.Read(randomData)
	require.NoError(t, err)

	cm := NewCompressionManager()
	algorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	for _, algorithm := range algorithms {
		t.Run(string(algorithm), func(t *testing.T) {
			compressor, err := cm.GetCompressor(algorithm)
			require.NoError(t, err)

			compressed, stats, err := compressor.Compress(randomData, compressor.GetDefaultLevel())
			require.NoError(t, err)

			// Random data might not compress well, but should still work
			assert.Equal(t, int64(len(randomData)), stats.OriginalSize)
			assert.Equal(t, int64(len(compressed)), stats.CompressedSize)

			decompressed, err := compressor.Decompress(compressed)
			require.NoError(t, err)
			assert.Equal(t, randomData, decompressed)
		})
	}
}

func TestCompressionWithEmptyData(t *testing.T) {
	emptyData := []byte{}

	cm := NewCompressionManager()
	algorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	for _, algorithm := range algorithms {
		t.Run(string(algorithm), func(t *testing.T) {
			compressor, err := cm.GetCompressor(algorithm)
			require.NoError(t, err)

			compressed, stats, err := compressor.Compress(emptyData, compressor.GetDefaultLevel())
			require.NoError(t, err)

			assert.Equal(t, int64(0), stats.OriginalSize)

			decompressed, err := compressor.Decompress(compressed)
			require.NoError(t, err)
			// Handle nil vs empty slice difference
			if len(emptyData) == 0 && len(decompressed) == 0 {
				// Both are empty, test passes
			} else {
				assert.Equal(t, emptyData, decompressed)
			}
		})
	}
}

func TestCompressionWithInvalidLevel(t *testing.T) {
	testData := []byte("test data")
	cm := NewCompressionManager()

	// Test with invalid levels - should use default level
	algorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	for _, algorithm := range algorithms {
		t.Run(string(algorithm), func(t *testing.T) {
			compressor, err := cm.GetCompressor(algorithm)
			require.NoError(t, err)

			// Test with level too high
			compressed, stats, err := cm.Compress(testData, algorithm, 999)
			require.NoError(t, err)
			assert.Equal(t, compressor.GetDefaultLevel(), stats.Level)

			// Test with level too low
			compressed, stats, err = cm.Compress(testData, algorithm, -1)
			require.NoError(t, err)
			assert.Equal(t, compressor.GetDefaultLevel(), stats.Level)

			// Verify decompression still works
			decompressed, err := cm.Decompress(compressed, algorithm)
			require.NoError(t, err)
			assert.Equal(t, testData, decompressed)
		})
	}
}

func TestCalculateCompressionRatio(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   int64
		compressedSize int64
		expectedRatio  float64
	}{
		{"50% compression", 1000, 500, 0.5},
		{"No compression", 1000, 1000, 1.0},
		{"Expansion", 1000, 1200, 1.2},
		{"Zero original", 0, 100, 1.0},
		{"Zero compressed", 1000, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := CalculateCompressionRatio(tt.originalSize, tt.compressedSize)
			assert.Equal(t, tt.expectedRatio, ratio)
		})
	}
}

func TestCompressionErrorHandling(t *testing.T) {
	t.Run("Invalid compressed data", func(t *testing.T) {
		compressor := &GzipCompressor{}
		invalidData := []byte("this is not compressed data")

		_, err := compressor.Decompress(invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to")
	})

	t.Run("Corrupted compressed data", func(t *testing.T) {
		compressor := &GzipCompressor{}
		testData := []byte("test data for corruption test that is longer to ensure proper compression")

		compressed, _, err := compressor.Compress(testData, compressor.GetDefaultLevel())
		require.NoError(t, err)

		// Create completely invalid compressed data
		invalidCompressed := make([]byte, len(compressed))
		for i := range invalidCompressed {
			invalidCompressed[i] = byte(i % 256)
		}

		_, err = compressor.Decompress(invalidCompressed)
		// Should definitely fail with completely invalid data
		assert.Error(t, err)
	})
}

// Benchmark tests for performance comparison
func BenchmarkCompressionAlgorithms(b *testing.B) {
	// Create test data with good compression potential
	testData := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 1000))

	algorithms := []CompressionType{
		CompressionTypeGzip,
		CompressionTypeLZ4,
		CompressionTypeZstd,
	}

	cm := NewCompressionManager()

	for _, algorithm := range algorithms {
		compressor, _ := cm.GetCompressor(algorithm)

		b.Run(fmt.Sprintf("Compress_%s", algorithm), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := compressor.Compress(testData, compressor.GetDefaultLevel())
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Pre-compress data for decompression benchmark
		compressed, _, _ := compressor.Compress(testData, compressor.GetDefaultLevel())

		b.Run(fmt.Sprintf("Decompress_%s", algorithm), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := compressor.Decompress(compressed)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCompressionLevels(b *testing.B) {
	testData := []byte(strings.Repeat("Benchmark data for compression level testing. ", 500))

	algorithms := []struct {
		name   CompressionType
		levels []int
	}{
		{CompressionTypeGzip, []int{1, 6, 9}},
		{CompressionTypeLZ4, []int{1, 6, 12}},
		{CompressionTypeZstd, []int{1, 3, 10}},
	}

	cm := NewCompressionManager()

	for _, algo := range algorithms {
		compressor, _ := cm.GetCompressor(algo.name)

		for _, level := range algo.levels {
			b.Run(fmt.Sprintf("%s_Level_%d", algo.name, level), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, err := compressor.Compress(testData, level)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
