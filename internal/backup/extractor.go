package backup

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SchemaExtractor handles complete schema backup extraction from MySQL databases
type SchemaExtractor struct {
	queryTimeout   time.Duration
	displayService ExtractorDisplayService
}

// ExtractorDisplayService interface for progress tracking and user feedback
type ExtractorDisplayService interface {
	ShowProgress(current, total int, message string)
	Info(message string)
	Error(message string)
	Debug(message string)
}

// NewSchemaExtractor creates a new schema extractor
func NewSchemaExtractor(displayService ExtractorDisplayService) *SchemaExtractor {
	return &SchemaExtractor{
		queryTimeout:   30 * time.Second,
		displayService: displayService,
	}
}

// SetQueryTimeout sets the query timeout for database operations
func (e *SchemaExtractor) SetQueryTimeout(timeout time.Duration) {
	e.queryTimeout = timeout
}

// SetDisplayService sets the display service for progress tracking
func (e *SchemaExtractor) SetDisplayService(displayService ExtractorDisplayService) {
	e.displayService = displayService
}

// ExtractCompleteSchema extracts a complete database schema for backup purposes
// TODO: Implement when schema dependency is resolved
func (e *SchemaExtractor) ExtractCompleteSchema(ctx context.Context, db *sql.DB, schemaName string) (*Backup, error) {
	return nil, fmt.Errorf("not implemented - schema dependency needs to be resolved")
}

// Helper methods for logging and progress tracking
func (e *SchemaExtractor) logInfo(message string) {
	if e.displayService != nil {
		e.displayService.Info(message)
	}
}

func (e *SchemaExtractor) logError(message string) {
	if e.displayService != nil {
		e.displayService.Error(message)
	}
}

func (e *SchemaExtractor) logDebug(message string) {
	if e.displayService != nil {
		e.displayService.Debug(message)
	}
}

func (e *SchemaExtractor) showProgress(current, total int, message string) {
	if e.displayService != nil {
		e.displayService.ShowProgress(current, total, message)
	}
}
