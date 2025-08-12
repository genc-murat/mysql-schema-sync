package schema

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Extractor handles schema extraction from MySQL databases
type Extractor struct {
	queryTimeout time.Duration
}

// NewExtractor creates a new schema extractor
func NewExtractor() *Extractor {
	return &Extractor{
		queryTimeout: 30 * time.Second,
	}
}

// NewExtractorWithTimeout creates a new schema extractor with custom timeout
func NewExtractorWithTimeout(timeout time.Duration) *Extractor {
	return &Extractor{
		queryTimeout: timeout,
	}
}

// ExtractSchema extracts the complete schema from a MySQL database
func (e *Extractor) ExtractSchema(db *sql.DB, schemaName string) (*Schema, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if schemaName == "" {
		return nil, fmt.Errorf("schema name cannot be empty")
	}

	schema := NewSchema(schemaName)

	// Extract tables
	tables, err := e.extractTables(db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tables: %w", err)
	}

	// For each table, extract columns and indexes
	for _, table := range tables {
		// Extract columns
		columns, err := e.extractColumns(db, schemaName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to extract columns for table %s: %w", table.Name, err)
		}
		table.Columns = columns

		// Extract indexes
		indexes, err := e.extractIndexes(db, schemaName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to extract indexes for table %s: %w", table.Name, err)
		}
		table.Indexes = indexes

		// Add table to schema
		schema.Tables[table.Name] = table
	}

	// Extract global indexes (if any)
	globalIndexes, err := e.extractGlobalIndexes(db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global indexes: %w", err)
	}
	schema.Indexes = globalIndexes

	// Validate the extracted schema
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("extracted schema is invalid: %w", err)
	}

	return schema, nil
}

// extractTables extracts all tables from the specified schema
func (e *Extractor) extractTables(db *sql.DB, schemaName string) (map[string]*Table, error) {
	query := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`

	ctx, cancel := context.WithTimeout(context.Background(), e.queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	tables := make(map[string]*Table)

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}

		table := NewTable(tableName)
		tables[tableName] = table
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// extractColumns extracts all columns for a specific table
func (e *Extractor) extractColumns(db *sql.DB, schemaName, tableName string) (map[string]*Column, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			EXTRA,
			ORDINAL_POSITION,
			COLUMN_TYPE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	ctx, cancel := context.WithTimeout(context.Background(), e.queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %s: %w", tableName, err)
	}
	defer rows.Close()

	columns := make(map[string]*Column)

	for rows.Next() {
		var columnName, dataType, isNullable, extra, columnType string
		var defaultValue sql.NullString
		var position int

		err := rows.Scan(
			&columnName,
			&dataType,
			&isNullable,
			&defaultValue,
			&extra,
			&position,
			&columnType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column data: %w", err)
		}

		column := &Column{
			Name:       columnName,
			DataType:   columnType, // Use COLUMN_TYPE for full type info (e.g., varchar(255))
			IsNullable: isNullable == "YES",
			Extra:      extra,
			Position:   position,
		}

		// Handle default value
		if defaultValue.Valid {
			column.DefaultValue = &defaultValue.String
		}

		columns[columnName] = column
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	return columns, nil
}

// extractIndexes extracts all indexes for a specific table
func (e *Extractor) extractIndexes(db *sql.DB, schemaName, tableName string) ([]*Index, error) {
	query := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE,
			SEQ_IN_INDEX
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	ctx, cancel := context.WithTimeout(context.Background(), e.queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes for table %s: %w", tableName, err)
	}
	defer rows.Close()

	// Group columns by index name
	indexMap := make(map[string]*indexBuilder)

	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int
		var seqInIndex int

		err := rows.Scan(
			&indexName,
			&columnName,
			&nonUnique,
			&indexType,
			&seqInIndex,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index data: %w", err)
		}

		// Get or create index builder
		builder, exists := indexMap[indexName]
		if !exists {
			builder = &indexBuilder{
				name:      indexName,
				tableName: tableName,
				isUnique:  nonUnique == 0,
				isPrimary: indexName == "PRIMARY",
				indexType: indexType,
				columns:   make([]string, 0),
			}
			indexMap[indexName] = builder
		}

		// Add column to index (maintain order)
		builder.columns = append(builder.columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating index rows: %w", err)
	}

	// Convert builders to indexes
	indexes := make([]*Index, 0, len(indexMap))
	for _, builder := range indexMap {
		index := &Index{
			Name:      builder.name,
			TableName: builder.tableName,
			Columns:   builder.columns,
			IsUnique:  builder.isUnique,
			IsPrimary: builder.isPrimary,
			IndexType: builder.indexType,
		}
		indexes = append(indexes, index)
	}

	return indexes, nil
}

// extractGlobalIndexes extracts schema-level indexes (if any)
func (e *Extractor) extractGlobalIndexes(db *sql.DB, schemaName string) (map[string]*Index, error) {
	// For MySQL, all indexes are table-specific, so we return an empty map
	// This method is here for completeness and future extensibility
	return make(map[string]*Index), nil
}

// indexBuilder is a helper struct for building indexes from multiple rows
type indexBuilder struct {
	name      string
	tableName string
	isUnique  bool
	isPrimary bool
	indexType string
	columns   []string
}

// GetCurrentSchema retrieves the current schema name from the database connection
func (e *Extractor) GetCurrentSchema(db *sql.DB) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database connection is nil")
	}

	var schemaName string
	query := "SELECT DATABASE()"

	ctx, cancel := context.WithTimeout(context.Background(), e.queryTimeout)
	defer cancel()

	err := db.QueryRowContext(ctx, query).Scan(&schemaName)
	if err != nil {
		return "", fmt.Errorf("failed to get current schema: %w", err)
	}

	if schemaName == "" {
		return "", fmt.Errorf("no schema selected")
	}

	return schemaName, nil
}

// ValidateSchemaExists checks if the specified schema exists
func (e *Extractor) ValidateSchemaExists(db *sql.DB, schemaName string) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	if schemaName == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	query := `
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE SCHEMA_NAME = ?
	`

	ctx, cancel := context.WithTimeout(context.Background(), e.queryTimeout)
	defer cancel()

	var count int
	err := db.QueryRowContext(ctx, query, schemaName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate schema existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("schema '%s' does not exist", schemaName)
	}

	return nil
}
