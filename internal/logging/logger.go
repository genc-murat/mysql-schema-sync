package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// LogLevelQuiet suppresses all output except critical errors
	LogLevelQuiet LogLevel = "quiet"
	// LogLevelNormal shows standard operational messages
	LogLevelNormal LogLevel = "normal"
	// LogLevelVerbose shows detailed operational information
	LogLevelVerbose LogLevel = "verbose"
	// LogLevelDebug shows all debug information
	LogLevelDebug LogLevel = "debug"
)

// Logger provides structured logging capabilities
type Logger struct {
	logger *logrus.Logger
	level  LogLevel
}

// Config holds logger configuration
type Config struct {
	Level      LogLevel
	Output     io.Writer
	Format     string // "text" or "json"
	ShowCaller bool
	LogFile    string
}

// NewLogger creates a new logger with the specified configuration
func NewLogger(config Config) (*Logger, error) {
	logger := logrus.New()

	// Set output
	if config.Output != nil {
		logger.SetOutput(config.Output)
	} else {
		logger.SetOutput(os.Stdout)
	}

	// Set format
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			DisableColors:   false,
		})
	}

	// Set log level based on our custom levels
	switch config.Level {
	case LogLevelQuiet:
		logger.SetLevel(logrus.ErrorLevel)
	case LogLevelNormal:
		logger.SetLevel(logrus.InfoLevel)
	case LogLevelVerbose:
		logger.SetLevel(logrus.DebugLevel)
	case LogLevelDebug:
		logger.SetLevel(logrus.TraceLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Enable caller reporting if requested
	if config.ShowCaller {
		logger.SetReportCaller(true)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	}

	// Set up file logging if specified
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", config.LogFile, err)
		}

		// Use multi-writer to write to both file and stdout
		if config.Output == nil {
			logger.SetOutput(io.MultiWriter(os.Stdout, file))
		} else {
			logger.SetOutput(io.MultiWriter(config.Output, file))
		}
	}

	return &Logger{
		logger: logger,
		level:  config.Level,
	}, nil
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *Logger {
	config := Config{
		Level:      LogLevelNormal,
		Output:     os.Stdout,
		Format:     "text",
		ShowCaller: false,
	}

	logger, _ := NewLogger(config)
	return logger
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.logger.WithContext(ctx)

	// Add request ID if available in context
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	return entry
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.logger.WithFields(fields)
}

// WithField returns a logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.logger.WithField(key, value)
}

// Database operation logging methods

// LogDatabaseConnection logs database connection attempts
func (l *Logger) LogDatabaseConnection(host string, database string, success bool, duration time.Duration, err error) {
	fields := logrus.Fields{
		"operation": "database_connection",
		"host":      host,
		"database":  database,
		"duration":  duration.String(),
		"success":   success,
	}

	if success {
		l.logger.WithFields(fields).Info("Database connection established")
	} else {
		if err != nil {
			fields["error"] = err.Error()
		}
		l.logger.WithFields(fields).Error("Database connection failed")
	}
}

// LogSQLExecution logs SQL statement execution
func (l *Logger) LogSQLExecution(sql string, duration time.Duration, rowsAffected int64, err error) {
	fields := logrus.Fields{
		"operation":     "sql_execution",
		"duration":      duration.String(),
		"rows_affected": rowsAffected,
	}

	// Truncate long SQL statements for readability
	if len(sql) > 200 {
		fields["sql"] = sql[:200] + "..."
		fields["sql_length"] = len(sql)
	} else {
		fields["sql"] = sql
	}

	if err != nil {
		fields["error"] = err.Error()
		l.logger.WithFields(fields).Error("SQL execution failed")
	} else {
		if l.level == LogLevelVerbose || l.level == LogLevelDebug {
			l.logger.WithFields(fields).Debug("SQL executed successfully")
		}
	}
}

// LogSchemaExtraction logs schema extraction operations
func (l *Logger) LogSchemaExtraction(database string, tableCount int, duration time.Duration, err error) {
	fields := logrus.Fields{
		"operation":   "schema_extraction",
		"database":    database,
		"table_count": tableCount,
		"duration":    duration.String(),
	}

	if err != nil {
		fields["error"] = err.Error()
		l.logger.WithFields(fields).Error("Schema extraction failed")
	} else {
		l.logger.WithFields(fields).Info("Schema extraction completed")
	}
}

// LogSchemaComparison logs schema comparison operations
func (l *Logger) LogSchemaComparison(sourceDB, targetDB string, changesFound int, duration time.Duration) {
	fields := logrus.Fields{
		"operation":     "schema_comparison",
		"source_db":     sourceDB,
		"target_db":     targetDB,
		"changes_found": changesFound,
		"duration":      duration.String(),
	}

	if changesFound > 0 {
		l.logger.WithFields(fields).Info("Schema differences detected")
	} else {
		l.logger.WithFields(fields).Info("No schema differences found")
	}
}

// LogMigrationExecution logs migration execution
func (l *Logger) LogMigrationExecution(statementsCount int, duration time.Duration, success bool, err error) {
	fields := logrus.Fields{
		"operation":        "migration_execution",
		"statements_count": statementsCount,
		"duration":         duration.String(),
		"success":          success,
	}

	if err != nil {
		fields["error"] = err.Error()
		l.logger.WithFields(fields).Error("Migration execution failed")
	} else {
		l.logger.WithFields(fields).Info("Migration executed successfully")
	}
}

// Standard logging methods

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
	switch level {
	case LogLevelQuiet:
		l.logger.SetLevel(logrus.ErrorLevel)
	case LogLevelNormal:
		l.logger.SetLevel(logrus.InfoLevel)
	case LogLevelVerbose:
		l.logger.SetLevel(logrus.DebugLevel)
	case LogLevelDebug:
		l.logger.SetLevel(logrus.TraceLevel)
	}
}

// IsLevelEnabled checks if a log level is enabled
func (l *Logger) IsLevelEnabled(level LogLevel) bool {
	switch level {
	case LogLevelQuiet:
		return l.logger.IsLevelEnabled(logrus.ErrorLevel)
	case LogLevelNormal:
		return l.logger.IsLevelEnabled(logrus.InfoLevel)
	case LogLevelVerbose:
		return l.logger.IsLevelEnabled(logrus.DebugLevel)
	case LogLevelDebug:
		return l.logger.IsLevelEnabled(logrus.TraceLevel)
	default:
		return false
	}
}

// LogOperationStart logs the start of an operation and returns a function to log completion
func (l *Logger) LogOperationStart(operation string, fields map[string]interface{}) func(error) {
	startTime := time.Now()

	logFields := logrus.Fields{
		"operation": operation,
		"status":    "started",
	}

	// Add additional fields
	for k, v := range fields {
		logFields[k] = v
	}

	l.logger.WithFields(logFields).Debug("Operation started")

	return func(err error) {
		duration := time.Since(startTime)
		logFields["status"] = "completed"
		logFields["duration"] = duration.String()

		if err != nil {
			logFields["error"] = err.Error()
			logFields["success"] = false
			l.logger.WithFields(logFields).Error("Operation failed")
		} else {
			logFields["success"] = true
			l.logger.WithFields(logFields).Info("Operation completed")
		}
	}
}

// CreateContextWithRequestID creates a context with a request ID for tracing
func CreateContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, "request_id", requestID)
}

// GetRequestIDFromContext extracts request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// SanitizeSQL sanitizes SQL for logging by removing sensitive information
func SanitizeSQL(sql string) string {
	// Remove potential passwords or sensitive data from connection strings
	// Use regex-like replacement to handle password values properly
	if strings.Contains(sql, "password=") {
		parts := strings.Split(sql, "password=")
		if len(parts) > 1 {
			// Find the end of the password value (space, quote, or end of string)
			passwordPart := parts[1]
			var endIndex int
			if len(passwordPart) > 0 && (passwordPart[0] == '\'' || passwordPart[0] == '"') {
				// Quoted password - find closing quote
				quote := passwordPart[0]
				endIndex = strings.Index(passwordPart[1:], string(quote))
				if endIndex != -1 {
					endIndex += 2 // Include both quotes
				} else {
					endIndex = len(passwordPart)
				}
			} else {
				// Unquoted password - find space or end
				endIndex = strings.Index(passwordPart, " ")
				if endIndex == -1 {
					endIndex = len(passwordPart)
				}
			}
			sql = parts[0] + "password=***" + passwordPart[endIndex:]
		}
	}

	if strings.Contains(sql, "PASSWORD=") {
		parts := strings.Split(sql, "PASSWORD=")
		if len(parts) > 1 {
			// Find the end of the password value (space, quote, or end of string)
			passwordPart := parts[1]
			var endIndex int
			if len(passwordPart) > 0 && (passwordPart[0] == '\'' || passwordPart[0] == '"') {
				// Quoted password - find closing quote
				quote := passwordPart[0]
				endIndex = strings.Index(passwordPart[1:], string(quote))
				if endIndex != -1 {
					endIndex += 2 // Include both quotes
				} else {
					endIndex = len(passwordPart)
				}
			} else {
				// Unquoted password - find space or end
				endIndex = strings.Index(passwordPart, " ")
				if endIndex == -1 {
					endIndex = len(passwordPart)
				}
			}
			sql = parts[0] + "PASSWORD=***" + passwordPart[endIndex:]
		}
	}

	// Truncate very long SQL statements
	if len(sql) > 500 {
		return sql[:500] + "... [truncated]"
	}

	return sql
}
