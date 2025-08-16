package backup

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// CompressionStats contains statistics about compression operations
type CompressionStats struct {
	OriginalSize     int64           `json:"original_size"`
	CompressedSize   int64           `json:"compressed_size"`
	CompressionRatio float64         `json:"compression_ratio"`
	Algorithm        CompressionType `json:"algorithm"`
	Level            int             `json:"level"`
	Duration         time.Duration   `json:"duration"`
}

// Compressor interface defines compression operations
type Compressor interface {
	Compress(data []byte, level int) ([]byte, *CompressionStats, error)
	Decompress(data []byte) ([]byte, error)
	GetAlgorithm() CompressionType
	GetDefaultLevel() int
	GetMaxLevel() int
	GetMinLevel() int
}

// CompressionManager manages compression operations
type CompressionManager struct {
	compressors map[CompressionType]Compressor
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager() *CompressionManager {
	cm := &CompressionManager{
		compressors: make(map[CompressionType]Compressor),
	}

	// Register available compressors
	cm.compressors[CompressionTypeGzip] = &GzipCompressor{}
	cm.compressors[CompressionTypeLZ4] = &LZ4Compressor{}
	cm.compressors[CompressionTypeZstd] = &ZstdCompressor{}

	return cm
}

// Compress compresses data using the specified algorithm and level
func (cm *CompressionManager) Compress(data []byte, algorithm CompressionType, level int) ([]byte, *CompressionStats, error) {
	if algorithm == CompressionTypeNone {
		return data, &CompressionStats{
			OriginalSize:     int64(len(data)),
			CompressedSize:   int64(len(data)),
			CompressionRatio: 1.0,
			Algorithm:        CompressionTypeNone,
			Level:            0,
			Duration:         0,
		}, nil
	}

	compressor, exists := cm.compressors[algorithm]
	if !exists {
		return nil, nil, NewCompressionError(fmt.Sprintf("unsupported compression algorithm: %s", algorithm), nil)
	}

	// Validate compression level
	if level < compressor.GetMinLevel() || level > compressor.GetMaxLevel() {
		level = compressor.GetDefaultLevel()
	}

	return compressor.Compress(data, level)
}

// Decompress decompresses data using the specified algorithm
func (cm *CompressionManager) Decompress(data []byte, algorithm CompressionType) ([]byte, error) {
	if algorithm == CompressionTypeNone {
		return data, nil
	}

	compressor, exists := cm.compressors[algorithm]
	if !exists {
		return nil, NewCompressionError(fmt.Sprintf("unsupported compression algorithm: %s", algorithm), nil)
	}

	return compressor.Decompress(data)
}

// GetCompressor returns a compressor for the specified algorithm
func (cm *CompressionManager) GetCompressor(algorithm CompressionType) (Compressor, error) {
	compressor, exists := cm.compressors[algorithm]
	if !exists {
		return nil, NewCompressionError(fmt.Sprintf("unsupported compression algorithm: %s", algorithm), nil)
	}
	return compressor, nil
}

// GetSupportedAlgorithms returns a list of supported compression algorithms
func (cm *CompressionManager) GetSupportedAlgorithms() []CompressionType {
	algorithms := make([]CompressionType, 0, len(cm.compressors))
	for algorithm := range cm.compressors {
		algorithms = append(algorithms, algorithm)
	}
	return algorithms
}

// ShouldCompress determines if data should be compressed based on size threshold
func (cm *CompressionManager) ShouldCompress(dataSize int64, threshold int64) bool {
	return dataSize >= threshold
}

// CalculateCompressionRatio calculates the compression ratio
func CalculateCompressionRatio(originalSize, compressedSize int64) float64 {
	if originalSize == 0 {
		return 1.0
	}
	return float64(compressedSize) / float64(originalSize)
}

// GzipCompressor implements gzip compression
type GzipCompressor struct{}

func (gc *GzipCompressor) Compress(data []byte, level int) ([]byte, *CompressionStats, error) {
	start := time.Now()

	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, nil, NewCompressionError("failed to create gzip writer", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, nil, NewCompressionError("failed to write data to gzip writer", err)
	}

	if err := writer.Close(); err != nil {
		return nil, nil, NewCompressionError("failed to close gzip writer", err)
	}

	compressed := buf.Bytes()
	duration := time.Since(start)

	stats := &CompressionStats{
		OriginalSize:     int64(len(data)),
		CompressedSize:   int64(len(compressed)),
		CompressionRatio: CalculateCompressionRatio(int64(len(data)), int64(len(compressed))),
		Algorithm:        CompressionTypeGzip,
		Level:            level,
		Duration:         duration,
	}

	return compressed, stats, nil
}

func (gc *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, NewCompressionError("failed to create gzip reader", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewCompressionError("failed to decompress gzip data", err)
	}

	return decompressed, nil
}

func (gc *GzipCompressor) GetAlgorithm() CompressionType {
	return CompressionTypeGzip
}

func (gc *GzipCompressor) GetDefaultLevel() int {
	return gzip.DefaultCompression
}

func (gc *GzipCompressor) GetMaxLevel() int {
	return gzip.BestCompression
}

func (gc *GzipCompressor) GetMinLevel() int {
	return gzip.BestSpeed
}

// LZ4Compressor implements LZ4 compression
type LZ4Compressor struct{}

func (lc *LZ4Compressor) Compress(data []byte, level int) ([]byte, *CompressionStats, error) {
	start := time.Now()

	var buf bytes.Buffer

	// LZ4 has limited level options - use fast or high compression
	var writer *lz4.Writer
	if level > 6 {
		// High compression mode
		writer = lz4.NewWriter(&buf)
		if err := writer.Apply(lz4.CompressionLevelOption(lz4.Level9)); err != nil {
			return nil, nil, NewCompressionError("failed to set LZ4 high compression", err)
		}
	} else {
		// Fast compression mode (default)
		writer = lz4.NewWriter(&buf)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, nil, NewCompressionError("failed to write data to LZ4 writer", err)
	}

	if err := writer.Close(); err != nil {
		return nil, nil, NewCompressionError("failed to close LZ4 writer", err)
	}

	compressed := buf.Bytes()
	duration := time.Since(start)

	stats := &CompressionStats{
		OriginalSize:     int64(len(data)),
		CompressedSize:   int64(len(compressed)),
		CompressionRatio: CalculateCompressionRatio(int64(len(data)), int64(len(compressed))),
		Algorithm:        CompressionTypeLZ4,
		Level:            level,
		Duration:         duration,
	}

	return compressed, stats, nil
}

func (lc *LZ4Compressor) Decompress(data []byte) ([]byte, error) {
	reader := lz4.NewReader(bytes.NewReader(data))

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewCompressionError("failed to decompress LZ4 data", err)
	}

	return decompressed, nil
}

func (lc *LZ4Compressor) GetAlgorithm() CompressionType {
	return CompressionTypeLZ4
}

func (lc *LZ4Compressor) GetDefaultLevel() int {
	return 1 // Fast compression
}

func (lc *LZ4Compressor) GetMaxLevel() int {
	return 12
}

func (lc *LZ4Compressor) GetMinLevel() int {
	return 1
}

// ZstdCompressor implements Zstandard compression
type ZstdCompressor struct{}

func (zc *ZstdCompressor) Compress(data []byte, level int) ([]byte, *CompressionStats, error) {
	start := time.Now()

	// Create encoder with specified level
	encoderLevel := zstd.SpeedFastest
	switch {
	case level <= 1:
		encoderLevel = zstd.SpeedFastest
	case level <= 3:
		encoderLevel = zstd.SpeedDefault
	case level <= 6:
		encoderLevel = zstd.SpeedBetterCompression
	default:
		encoderLevel = zstd.SpeedBestCompression
	}

	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(encoderLevel))
	if err != nil {
		return nil, nil, NewCompressionError("failed to create zstd encoder", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	duration := time.Since(start)

	stats := &CompressionStats{
		OriginalSize:     int64(len(data)),
		CompressedSize:   int64(len(compressed)),
		CompressionRatio: CalculateCompressionRatio(int64(len(data)), int64(len(compressed))),
		Algorithm:        CompressionTypeZstd,
		Level:            level,
		Duration:         duration,
	}

	return compressed, stats, nil
}

func (zc *ZstdCompressor) Decompress(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, NewCompressionError("failed to create zstd decoder", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, NewCompressionError("failed to decompress zstd data", err)
	}

	return decompressed, nil
}

func (zc *ZstdCompressor) GetAlgorithm() CompressionType {
	return CompressionTypeZstd
}

func (zc *ZstdCompressor) GetDefaultLevel() int {
	return 3 // Balanced compression
}

func (zc *ZstdCompressor) GetMaxLevel() int {
	return 22
}

func (zc *ZstdCompressor) GetMinLevel() int {
	return 1
}
