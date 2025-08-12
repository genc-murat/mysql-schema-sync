package schema

import (
	"database/sql"
	"fmt"
	"mysql-schema-sync/internal/errors"
	"mysql-schema-sync/internal/logging"
	"strings"
	"time"
)

// Service provides high-level schema operations
type Service struct {
	extractor *Extractor
	logger    *logging.Logger
}

// NewService creates a new schema service
func NewService() *Service {
	return &Service{
		extractor: NewExtractor(),
		logger:    logging.NewDefaultLogger(),
	}
}

// NewServiceWithTimeout creates a new schema service with custom timeout
func NewServiceWithTimeout(timeout time.Duration) *Service {
	return &Service{
		extractor: NewExtractorWithTimeout(timeout),
		logger:    logging.NewDefaultLogger(),
	}
}

// NewServiceWithLogger creates a new schema service with a custom logger
func NewServiceWithLogger(logger *logging.Logger) *Service {
	return &Service{
		extractor: NewExtractor(),
		logger:    logger,
	}
}

// ExtractSchemaFromDB extracts schema from a database connection
// If schemaName is empty, it will use the current database
func (s *Service) ExtractSchemaFromDB(db *sql.DB, schemaName string) (*Schema, error) {
	if db == nil {
		return nil, errors.NewAppError(errors.ErrorTypeValidation, "database connection is nil", nil)
	}

	startTime := time.Now()
	finishLog := s.logger.LogOperationStart("schema_extraction", map[string]interface{}{
		"schema": schemaName,
	})

	// If no schema name provided, get the current one
	if schemaName == "" {
		currentSchema, err := s.extractor.GetCurrentSchema(db)
		if err != nil {
			finishLog(err)
			return nil, errors.WrapError(err, "failed to get current schema")
		}
		schemaName = currentSchema
		s.logger.WithField("detected_schema", schemaName).Debug("Using current database schema")
	}

	// Validate that the schema exists
	if err := s.extractor.ValidateSchemaExists(db, schemaName); err != nil {
		finishLog(err)
		return nil, errors.WrapError(err, "schema validation failed")
	}

	// Extract the schema
	schema, err := s.extractor.ExtractSchema(db, schemaName)
	duration := time.Since(startTime)

	if err != nil {
		finishLog(err)
		s.logger.LogSchemaExtraction(schemaName, 0, duration, err)
		return nil, errors.WrapError(err, "failed to extract schema")
	}

	tableCount := len(schema.Tables)
	finishLog(nil)
	s.logger.LogSchemaExtraction(schemaName, tableCount, duration, nil)

	return schema, nil
}

// CompareSchemas compares two schemas and returns the differences
func (s *Service) CompareSchemas(source, target *Schema) (*SchemaDiff, error) {
	if source == nil {
		return nil, errors.NewAppError(errors.ErrorTypeValidation, "source schema is nil", nil)
	}
	if target == nil {
		return nil, errors.NewAppError(errors.ErrorTypeValidation, "target schema is nil", nil)
	}

	startTime := time.Now()
	finishLog := s.logger.LogOperationStart("schema_comparison", map[string]interface{}{
		"source_tables": len(source.Tables),
		"target_tables": len(target.Tables),
	})

	diff := &SchemaDiff{
		AddedTables:        make([]*Table, 0),
		RemovedTables:      make([]*Table, 0),
		ModifiedTables:     make([]*TableDiff, 0),
		AddedIndexes:       make([]*Index, 0),
		RemovedIndexes:     make([]*Index, 0),
		AddedConstraints:   make([]*Constraint, 0),
		RemovedConstraints: make([]*Constraint, 0),
	}

	// Find added and removed tables
	for tableName, sourceTable := range source.Tables {
		if _, exists := target.Tables[tableName]; !exists {
			diff.AddedTables = append(diff.AddedTables, sourceTable)
		}
	}

	for tableName, targetTable := range target.Tables {
		if _, exists := source.Tables[tableName]; !exists {
			diff.RemovedTables = append(diff.RemovedTables, targetTable)
		}
	}

	// Find modified tables
	for tableName, sourceTable := range source.Tables {
		if targetTable, exists := target.Tables[tableName]; exists {
			tableDiff := s.compareTable(sourceTable, targetTable)
			if !s.isTableDiffEmpty(tableDiff) {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
		}
	}

	// Compare global indexes (if any)
	for indexName, sourceIndex := range source.Indexes {
		if _, exists := target.Indexes[indexName]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, sourceIndex)
		}
	}

	for indexName, targetIndex := range target.Indexes {
		if _, exists := source.Indexes[indexName]; !exists {
			diff.RemovedIndexes = append(diff.RemovedIndexes, targetIndex)
		}
	}

	// Compare table-level indexes for all common tables
	s.compareTableIndexes(source, target, diff)

	duration := time.Since(startTime)
	changesFound := len(diff.AddedTables) + len(diff.RemovedTables) + len(diff.ModifiedTables) +
		len(diff.AddedIndexes) + len(diff.RemovedIndexes) + len(diff.AddedConstraints) + len(diff.RemovedConstraints)

	finishLog(nil)
	s.logger.LogSchemaComparison(source.Name, target.Name, changesFound, duration)

	return diff, nil
}

// compareTableIndexes compares indexes between tables in source and target schemas
func (s *Service) compareTableIndexes(source, target *Schema, diff *SchemaDiff) {
	// For each table that exists in both schemas, compare their indexes
	for tableName, sourceTable := range source.Tables {
		if targetTable, exists := target.Tables[tableName]; exists {
			s.compareIndexesForTable(sourceTable, targetTable, diff)
		}
	}
}

// compareIndexesForTable compares indexes between two tables
func (s *Service) compareIndexesForTable(sourceTable, targetTable *Table, diff *SchemaDiff) {
	sourceIndexMap := make(map[string]*Index)
	targetIndexMap := make(map[string]*Index)

	// Build maps for easier comparison
	for _, index := range sourceTable.Indexes {
		sourceIndexMap[index.Name] = index
	}

	for _, index := range targetTable.Indexes {
		targetIndexMap[index.Name] = index
	}

	// Find added indexes (exist in source but not in target)
	for indexName, sourceIndex := range sourceIndexMap {
		if _, exists := targetIndexMap[indexName]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, sourceIndex)
		}
	}

	// Find removed indexes (exist in target but not in source)
	for indexName, targetIndex := range targetIndexMap {
		if _, exists := sourceIndexMap[indexName]; !exists {
			diff.RemovedIndexes = append(diff.RemovedIndexes, targetIndex)
		}
	}

	// Find modified indexes (same name but different properties)
	for indexName, sourceIndex := range sourceIndexMap {
		if targetIndex, exists := targetIndexMap[indexName]; exists {
			if !s.areIndexesEqual(sourceIndex, targetIndex) {
				// Treat as remove old and add new
				diff.RemovedIndexes = append(diff.RemovedIndexes, targetIndex)
				diff.AddedIndexes = append(diff.AddedIndexes, sourceIndex)
			}
		}
	}
}

// areIndexesEqual compares two indexes for equality
func (s *Service) areIndexesEqual(idx1, idx2 *Index) bool {
	if idx1.Name != idx2.Name {
		return false
	}
	if idx1.TableName != idx2.TableName {
		return false
	}
	if idx1.IsUnique != idx2.IsUnique {
		return false
	}
	if idx1.IsPrimary != idx2.IsPrimary {
		return false
	}
	if idx1.IndexType != idx2.IndexType {
		return false
	}

	// Compare columns
	if len(idx1.Columns) != len(idx2.Columns) {
		return false
	}

	for i, col := range idx1.Columns {
		if col != idx2.Columns[i] {
			return false
		}
	}

	return true
}

// compareTable compares two tables and returns the differences
func (s *Service) compareTable(source, target *Table) *TableDiff {
	diff := &TableDiff{
		TableName:          source.Name,
		AddedColumns:       make([]*Column, 0),
		RemovedColumns:     make([]*Column, 0),
		ModifiedColumns:    make([]*ColumnDiff, 0),
		AddedConstraints:   make([]*Constraint, 0),
		RemovedConstraints: make([]*Constraint, 0),
	}

	// Find added and removed columns
	for columnName, sourceColumn := range source.Columns {
		if _, exists := target.Columns[columnName]; !exists {
			diff.AddedColumns = append(diff.AddedColumns, sourceColumn)
		}
	}

	for columnName, targetColumn := range target.Columns {
		if _, exists := source.Columns[columnName]; !exists {
			diff.RemovedColumns = append(diff.RemovedColumns, targetColumn)
		}
	}

	// Find modified columns
	for columnName, sourceColumn := range source.Columns {
		if targetColumn, exists := target.Columns[columnName]; exists {
			if !s.areColumnsEqual(sourceColumn, targetColumn) {
				columnDiff := &ColumnDiff{
					ColumnName: columnName,
					OldColumn:  targetColumn,
					NewColumn:  sourceColumn,
				}
				diff.ModifiedColumns = append(diff.ModifiedColumns, columnDiff)
			}
		}
	}

	// Compare constraints
	s.compareConstraintsForTable(source, target, diff)

	return diff
}

// areColumnsEqual compares two columns for equality
func (s *Service) areColumnsEqual(col1, col2 *Column) bool {
	if col1.Name != col2.Name {
		return false
	}
	if col1.DataType != col2.DataType {
		return false
	}
	if col1.IsNullable != col2.IsNullable {
		return false
	}
	if col1.Extra != col2.Extra {
		return false
	}

	// Compare default values
	if col1.DefaultValue == nil && col2.DefaultValue != nil {
		return false
	}
	if col1.DefaultValue != nil && col2.DefaultValue == nil {
		return false
	}
	if col1.DefaultValue != nil && col2.DefaultValue != nil {
		if *col1.DefaultValue != *col2.DefaultValue {
			return false
		}
	}

	return true
}

// compareConstraintsForTable compares constraints between two tables
func (s *Service) compareConstraintsForTable(sourceTable, targetTable *Table, diff *TableDiff) {
	// Find added constraints (exist in source but not in target)
	for constraintName, sourceConstraint := range sourceTable.Constraints {
		if _, exists := targetTable.Constraints[constraintName]; !exists {
			diff.AddedConstraints = append(diff.AddedConstraints, sourceConstraint)
		}
	}

	// Find removed constraints (exist in target but not in source)
	for constraintName, targetConstraint := range targetTable.Constraints {
		if _, exists := sourceTable.Constraints[constraintName]; !exists {
			diff.RemovedConstraints = append(diff.RemovedConstraints, targetConstraint)
		}
	}

	// Find modified constraints (same name but different properties)
	for constraintName, sourceConstraint := range sourceTable.Constraints {
		if targetConstraint, exists := targetTable.Constraints[constraintName]; exists {
			if !s.areConstraintsEqual(sourceConstraint, targetConstraint) {
				// Treat as remove old and add new
				diff.RemovedConstraints = append(diff.RemovedConstraints, targetConstraint)
				diff.AddedConstraints = append(diff.AddedConstraints, sourceConstraint)
			}
		}
	}
}

// areConstraintsEqual compares two constraints for equality
func (s *Service) areConstraintsEqual(c1, c2 *Constraint) bool {
	if c1.Name != c2.Name {
		return false
	}
	if c1.TableName != c2.TableName {
		return false
	}
	if c1.Type != c2.Type {
		return false
	}
	if c1.ReferencedTable != c2.ReferencedTable {
		return false
	}
	if c1.OnUpdate != c2.OnUpdate {
		return false
	}
	if c1.OnDelete != c2.OnDelete {
		return false
	}
	if c1.CheckExpression != c2.CheckExpression {
		return false
	}

	// Compare columns
	if len(c1.Columns) != len(c2.Columns) {
		return false
	}
	for i, col := range c1.Columns {
		if col != c2.Columns[i] {
			return false
		}
	}

	// Compare referenced columns
	if len(c1.ReferencedColumns) != len(c2.ReferencedColumns) {
		return false
	}
	for i, col := range c1.ReferencedColumns {
		if col != c2.ReferencedColumns[i] {
			return false
		}
	}

	return true
}

// isTableDiffEmpty checks if a table diff contains any changes
func (s *Service) isTableDiffEmpty(diff *TableDiff) bool {
	return len(diff.AddedColumns) == 0 &&
		len(diff.RemovedColumns) == 0 &&
		len(diff.ModifiedColumns) == 0 &&
		len(diff.AddedConstraints) == 0 &&
		len(diff.RemovedConstraints) == 0
}

// IsSchemaDiffEmpty checks if a schema diff contains any changes
func (s *Service) IsSchemaDiffEmpty(diff *SchemaDiff) bool {
	return len(diff.AddedTables) == 0 &&
		len(diff.RemovedTables) == 0 &&
		len(diff.ModifiedTables) == 0 &&
		len(diff.AddedIndexes) == 0 &&
		len(diff.RemovedIndexes) == 0 &&
		len(diff.AddedConstraints) == 0 &&
		len(diff.RemovedConstraints) == 0
}

// GetSchemaStats returns statistics about a schema
func (s *Service) GetSchemaStats(schema *Schema) map[string]int {
	stats := make(map[string]int)

	stats["tables"] = len(schema.Tables)
	stats["global_indexes"] = len(schema.Indexes)

	totalColumns := 0
	totalIndexes := 0

	for _, table := range schema.Tables {
		totalColumns += len(table.Columns)
		totalIndexes += len(table.Indexes)
	}

	stats["columns"] = totalColumns
	stats["indexes"] = totalIndexes

	return stats
}

// DetectRenamedTables attempts to detect renamed tables by comparing structure
func (s *Service) DetectRenamedTables(source, target *Schema) map[string]string {
	renames := make(map[string]string)

	// Get tables that appear to be added and removed
	addedTables := make(map[string]*Table)
	removedTables := make(map[string]*Table)

	for tableName, sourceTable := range source.Tables {
		if _, exists := target.Tables[tableName]; !exists {
			addedTables[tableName] = sourceTable
		}
	}

	for tableName, targetTable := range target.Tables {
		if _, exists := source.Tables[tableName]; !exists {
			removedTables[tableName] = targetTable
		}
	}

	// Try to match added and removed tables by structure similarity
	for addedName, addedTable := range addedTables {
		bestMatch := ""
		bestScore := 0.0

		for removedName, removedTable := range removedTables {
			score := s.calculateTableSimilarity(addedTable, removedTable)
			if score > bestScore && score > 0.8 { // 80% similarity threshold
				bestScore = score
				bestMatch = removedName
			}
		}

		if bestMatch != "" {
			renames[bestMatch] = addedName
			delete(removedTables, bestMatch)
		}
	}

	return renames
}

// calculateTableSimilarity calculates similarity score between two tables (0.0 to 1.0)
func (s *Service) calculateTableSimilarity(table1, table2 *Table) float64 {
	if len(table1.Columns) == 0 && len(table2.Columns) == 0 {
		return 1.0
	}

	if len(table1.Columns) == 0 || len(table2.Columns) == 0 {
		return 0.0
	}

	matchingColumns := 0
	totalColumns := len(table1.Columns)

	for columnName, column1 := range table1.Columns {
		if column2, exists := table2.Columns[columnName]; exists {
			if s.areColumnsEqual(column1, column2) {
				matchingColumns++
			}
		}
	}

	// Consider the larger table size for denominator to penalize size differences
	if len(table2.Columns) > totalColumns {
		totalColumns = len(table2.Columns)
	}

	return float64(matchingColumns) / float64(totalColumns)
}

// DetectComplexModifications identifies complex changes that might need special handling
func (s *Service) DetectComplexModifications(diff *SchemaDiff) []string {
	warnings := make([]string, 0)

	// Check for potentially destructive operations
	for _, table := range diff.RemovedTables {
		warnings = append(warnings, fmt.Sprintf("Table '%s' will be dropped - this will result in data loss", table.Name))
	}

	for _, tableDiff := range diff.ModifiedTables {
		for _, column := range tableDiff.RemovedColumns {
			warnings = append(warnings, fmt.Sprintf("Column '%s.%s' will be dropped - this will result in data loss", tableDiff.TableName, column.Name))
		}

		for _, columnDiff := range tableDiff.ModifiedColumns {
			// Check for data type changes that might cause data loss
			if s.isDataTypeShrinking(columnDiff.OldColumn.DataType, columnDiff.NewColumn.DataType) {
				warnings = append(warnings, fmt.Sprintf("Column '%s.%s' data type change from %s to %s may cause data loss",
					tableDiff.TableName, columnDiff.ColumnName, columnDiff.OldColumn.DataType, columnDiff.NewColumn.DataType))
			}

			// Check for nullability changes
			if !columnDiff.OldColumn.IsNullable && columnDiff.NewColumn.IsNullable {
				warnings = append(warnings, fmt.Sprintf("Column '%s.%s' is changing from NOT NULL to NULL",
					tableDiff.TableName, columnDiff.ColumnName))
			} else if columnDiff.OldColumn.IsNullable && !columnDiff.NewColumn.IsNullable {
				warnings = append(warnings, fmt.Sprintf("Column '%s.%s' is changing from NULL to NOT NULL - ensure no NULL values exist",
					tableDiff.TableName, columnDiff.ColumnName))
			}
		}

		// Check for foreign key constraint removals
		for _, constraint := range tableDiff.RemovedConstraints {
			if constraint.Type == ConstraintTypeForeignKey {
				warnings = append(warnings, fmt.Sprintf("Foreign key constraint '%s' will be dropped from table '%s'",
					constraint.Name, tableDiff.TableName))
			}
		}
	}

	return warnings
}

// isDataTypeShrinking checks if a data type change might cause data loss
func (s *Service) isDataTypeShrinking(oldType, newType string) bool {
	// This is a simplified check - in reality, you'd want more sophisticated logic
	// to parse data type sizes and compare them

	// Examples of potentially shrinking changes
	shrinkingPatterns := map[string][]string{
		"text":       {"varchar", "char"},
		"longtext":   {"text", "mediumtext", "varchar", "char"},
		"mediumtext": {"text", "varchar", "char"},
		"bigint":     {"int", "mediumint", "smallint", "tinyint"},
		"int":        {"mediumint", "smallint", "tinyint"},
		"mediumint":  {"smallint", "tinyint"},
		"smallint":   {"tinyint"},
		"double":     {"float"},
	}

	oldTypeLower := strings.ToLower(strings.Split(oldType, "(")[0])
	newTypeLower := strings.ToLower(strings.Split(newType, "(")[0])

	if shrinkingTypes, exists := shrinkingPatterns[oldTypeLower]; exists {
		for _, shrinkingType := range shrinkingTypes {
			if newTypeLower == shrinkingType {
				return true
			}
		}
	}

	return false
}
