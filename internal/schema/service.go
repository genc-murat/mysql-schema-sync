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
	extractor      *Extractor
	logger         *logging.Logger
	displayService DisplayService
}

// DisplayService interface for visual enhancements (to avoid circular imports)
type DisplayService interface {
	StartSpinner(message string) SpinnerHandle
	UpdateSpinner(handle SpinnerHandle, message string)
	StopSpinner(handle SpinnerHandle, finalMessage string)
	ShowProgress(current, total int, message string)
	NewProgressTracker(phases []string) ProgressTracker
	Success(message string)
	Warning(message string)
	Error(message string)
	Info(message string)
	RenderIconWithColor(name string) string
	NewSchemaDiffPresenter() SchemaDiffPresenter
	PrintTable(headers []string, rows [][]string)
}

// SpinnerHandle interface for spinner management
type SpinnerHandle interface {
	ID() string
	IsActive() bool
}

// ProgressTracker interface for multi-phase progress tracking
type ProgressTracker interface {
	StartPhase(phaseIndex int, total int, message string)
	UpdatePhase(current int, message string)
	CompletePhase(finalMessage string)
	GetPhaseCount() int
	GetCurrentPhase() int
	IsCompleted() bool
}

// SchemaDiffPresenter interface for enhanced diff presentation
type SchemaDiffPresenter interface {
	FormatDiff(diff *SchemaDiff) string
	FormatTable(table *Table, changeType ChangeType) string
	FormatColumn(column *Column, changeType ChangeType) string
	FormatIndex(index *Index, changeType ChangeType) string
}

// ChangeType represents the type of change in schema comparison
type ChangeType int

const (
	ChangeAdded ChangeType = iota
	ChangeRemoved
	ChangeModified
)

// NewService creates a new schema service
func NewService() *Service {
	return &Service{
		extractor:      NewExtractor(),
		logger:         logging.NewDefaultLogger(),
		displayService: nil, // Will be set via SetDisplayService
	}
}

// NewServiceWithTimeout creates a new schema service with custom timeout
func NewServiceWithTimeout(timeout time.Duration) *Service {
	return &Service{
		extractor:      NewExtractorWithTimeout(timeout),
		logger:         logging.NewDefaultLogger(),
		displayService: nil, // Will be set via SetDisplayService
	}
}

// NewServiceWithLogger creates a new schema service with a custom logger
func NewServiceWithLogger(logger *logging.Logger) *Service {
	return &Service{
		extractor:      NewExtractor(),
		logger:         logger,
		displayService: nil, // Will be set via SetDisplayService
	}
}

// SetDisplayService sets the display service for visual enhancements
func (s *Service) SetDisplayService(displayService DisplayService) {
	s.displayService = displayService

	// Also set it on the extractor (cast to the simpler interface)
	s.extractor.SetDisplayService(displayService)
}

// ExtractSchemaFromDB extracts schema from a database connection
// If schemaName is empty, it will use the current database
func (s *Service) ExtractSchemaFromDB(db *sql.DB, schemaName string) (*Schema, error) {
	if db == nil {
		err := errors.NewAppError(errors.ErrorTypeValidation, "database connection is nil", nil)
		if s.displayService != nil {
			s.displayService.Error("Database connection is nil")
		}
		return nil, err
	}

	startTime := time.Now()
	finishLog := s.logger.LogOperationStart("schema_extraction", map[string]interface{}{
		"schema": schemaName,
	})

	// Start progress tracking
	var progressTracker ProgressTracker
	if s.displayService != nil {
		phases := []string{"Validation", "Schema Discovery", "Table Extraction", "Index Extraction"}
		progressTracker = s.displayService.NewProgressTracker(phases)
		s.displayService.Info(fmt.Sprintf("Starting schema extraction for '%s'...", schemaName))
	}

	// Phase 1: Schema validation and discovery
	if progressTracker != nil {
		progressTracker.StartPhase(0, 2, "Validating schema...")
	}

	// If no schema name provided, get the current one
	if schemaName == "" {
		if progressTracker != nil {
			progressTracker.UpdatePhase(1, "Detecting current schema...")
		}

		currentSchema, err := s.extractor.GetCurrentSchema(db)
		if err != nil {
			finishLog(err)
			if s.displayService != nil {
				s.displayService.Error(fmt.Sprintf("Failed to get current schema: %v", err))
			}
			return nil, errors.WrapError(err, "failed to get current schema")
		}
		schemaName = currentSchema
		s.logger.WithField("detected_schema", schemaName).Debug("Using current database schema")

		if s.displayService != nil {
			s.displayService.Info(fmt.Sprintf("Detected schema: %s", schemaName))
		}
	}

	// Validate that the schema exists
	if progressTracker != nil {
		progressTracker.UpdatePhase(2, "Validating schema existence...")
	}

	if err := s.extractor.ValidateSchemaExists(db, schemaName); err != nil {
		finishLog(err)
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Schema validation failed: %v", err))
		}
		return nil, errors.WrapError(err, "schema validation failed")
	}

	if progressTracker != nil {
		progressTracker.CompletePhase("Schema validation completed")
	}

	// Phase 2: Extract the schema
	if progressTracker != nil {
		progressTracker.StartPhase(1, 1, "Extracting schema structure...")
	}

	schema, err := s.extractor.ExtractSchema(db, schemaName)
	duration := time.Since(startTime)

	if err != nil {
		finishLog(err)
		s.logger.LogSchemaExtraction(schemaName, 0, duration, err)
		if s.displayService != nil {
			s.displayService.Error(fmt.Sprintf("Schema extraction failed: %v", err))
		}
		return nil, errors.WrapError(err, "failed to extract schema")
	}

	if progressTracker != nil {
		progressTracker.CompletePhase("Schema extraction completed")
	}

	tableCount := len(schema.Tables)
	finishLog(nil)
	s.logger.LogSchemaExtraction(schemaName, tableCount, duration, nil)

	// Show extraction results
	if s.displayService != nil {
		s.displayService.Success(fmt.Sprintf("%s Schema '%s' extracted successfully",
			s.displayService.RenderIconWithColor("success"), schemaName))

		// Show schema statistics
		stats := s.GetSchemaStats(schema)
		s.displayService.Info(fmt.Sprintf("Found %d tables, %d columns, %d indexes (%.2fs)",
			stats["tables"], stats["columns"], stats["indexes"], duration.Seconds()))
	}

	return schema, nil
}

// CompareSchemas compares two schemas and returns the differences
func (s *Service) CompareSchemas(source, target *Schema) (*SchemaDiff, error) {
	if source == nil {
		err := errors.NewAppError(errors.ErrorTypeValidation, "source schema is nil", nil)
		if s.displayService != nil {
			s.displayService.Error("Source schema is nil")
		}
		return nil, err
	}
	if target == nil {
		err := errors.NewAppError(errors.ErrorTypeValidation, "target schema is nil", nil)
		if s.displayService != nil {
			s.displayService.Error("Target schema is nil")
		}
		return nil, err
	}

	startTime := time.Now()
	finishLog := s.logger.LogOperationStart("schema_comparison", map[string]interface{}{
		"source_tables": len(source.Tables),
		"target_tables": len(target.Tables),
	})

	// Start progress tracking
	var progressTracker ProgressTracker
	if s.displayService != nil {
		phases := []string{"Table Analysis", "Column Comparison", "Index Comparison", "Constraint Analysis"}
		progressTracker = s.displayService.NewProgressTracker(phases)
		s.displayService.Info(fmt.Sprintf("Comparing schemas '%s' vs '%s'...", source.Name, target.Name))
	}

	diff := &SchemaDiff{
		AddedTables:        make([]*Table, 0),
		RemovedTables:      make([]*Table, 0),
		ModifiedTables:     make([]*TableDiff, 0),
		AddedIndexes:       make([]*Index, 0),
		RemovedIndexes:     make([]*Index, 0),
		AddedConstraints:   make([]*Constraint, 0),
		RemovedConstraints: make([]*Constraint, 0),
	}

	// Phase 1: Table Analysis
	if progressTracker != nil {
		progressTracker.StartPhase(0, len(source.Tables)+len(target.Tables), "Analyzing table differences...")
	}

	processed := 0

	// Find added and removed tables
	for tableName, sourceTable := range source.Tables {
		if _, exists := target.Tables[tableName]; !exists {
			diff.AddedTables = append(diff.AddedTables, sourceTable)
		}
		processed++
		if progressTracker != nil {
			progressTracker.UpdatePhase(processed, fmt.Sprintf("Analyzing table: %s", tableName))
		}
	}

	for tableName, targetTable := range target.Tables {
		if _, exists := source.Tables[tableName]; !exists {
			diff.RemovedTables = append(diff.RemovedTables, targetTable)
		}
		processed++
		if progressTracker != nil {
			progressTracker.UpdatePhase(processed, fmt.Sprintf("Analyzing table: %s", tableName))
		}
	}

	if progressTracker != nil {
		progressTracker.CompletePhase("Table analysis completed")
	}

	// Phase 2: Column Comparison
	commonTables := 0
	for tableName := range source.Tables {
		if _, exists := target.Tables[tableName]; exists {
			commonTables++
		}
	}

	if progressTracker != nil {
		progressTracker.StartPhase(1, commonTables, "Comparing table structures...")
	}

	processed = 0
	// Find modified tables
	for tableName, sourceTable := range source.Tables {
		if targetTable, exists := target.Tables[tableName]; exists {
			if progressTracker != nil {
				progressTracker.UpdatePhase(processed+1, fmt.Sprintf("Comparing table: %s", tableName))
			}

			tableDiff := s.compareTable(sourceTable, targetTable)
			if !s.isTableDiffEmpty(tableDiff) {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
			processed++
		}
	}

	if progressTracker != nil {
		progressTracker.CompletePhase("Column comparison completed")
	}

	// Phase 3: Index Comparison
	if progressTracker != nil {
		progressTracker.StartPhase(2, len(source.Indexes)+len(target.Indexes)+commonTables, "Comparing indexes...")
	}

	processed = 0

	// Compare global indexes (if any)
	for indexName, sourceIndex := range source.Indexes {
		if _, exists := target.Indexes[indexName]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, sourceIndex)
		}
		processed++
		if progressTracker != nil {
			progressTracker.UpdatePhase(processed, fmt.Sprintf("Analyzing global index: %s", indexName))
		}
	}

	for indexName, targetIndex := range target.Indexes {
		if _, exists := source.Indexes[indexName]; !exists {
			diff.RemovedIndexes = append(diff.RemovedIndexes, targetIndex)
		}
		processed++
		if progressTracker != nil {
			progressTracker.UpdatePhase(processed, fmt.Sprintf("Analyzing global index: %s", indexName))
		}
	}

	// Compare table-level indexes for all common tables
	for tableName := range source.Tables {
		if _, exists := target.Tables[tableName]; exists {
			processed++
			if progressTracker != nil {
				progressTracker.UpdatePhase(processed, fmt.Sprintf("Comparing indexes for table: %s", tableName))
			}
		}
	}

	s.compareTableIndexes(source, target, diff)

	if progressTracker != nil {
		progressTracker.CompletePhase("Index comparison completed")
	}

	// Phase 4: Final analysis
	if progressTracker != nil {
		progressTracker.StartPhase(3, 1, "Finalizing comparison...")
		progressTracker.UpdatePhase(1, "Calculating statistics...")
		progressTracker.CompletePhase("Schema comparison completed")
	}

	duration := time.Since(startTime)
	changesFound := len(diff.AddedTables) + len(diff.RemovedTables) + len(diff.ModifiedTables) +
		len(diff.AddedIndexes) + len(diff.RemovedIndexes) + len(diff.AddedConstraints) + len(diff.RemovedConstraints)

	finishLog(nil)
	s.logger.LogSchemaComparison(source.Name, target.Name, changesFound, duration)

	// Display comparison results
	if s.displayService != nil {
		if changesFound == 0 {
			s.displayService.Success(fmt.Sprintf("%s Schemas are identical - no changes detected (%.2fs)",
				s.displayService.RenderIconWithColor("success"), duration.Seconds()))
		} else {
			s.displayService.Info(fmt.Sprintf("Schema comparison completed: %d changes found (%.2fs)",
				changesFound, duration.Seconds()))

			// Show detailed comparison results
			s.displayComparisonSummary(diff)
		}
	}

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

// displayComparisonSummary shows a formatted summary of schema comparison results
func (s *Service) displayComparisonSummary(diff *SchemaDiff) {
	if s.displayService == nil {
		return
	}

	// Create summary table
	headers := []string{"Change Type", "Count", "Details"}
	rows := [][]string{}

	if len(diff.AddedTables) > 0 {
		details := fmt.Sprintf("Tables: %s", s.formatTableNames(diff.AddedTables))
		rows = append(rows, []string{
			fmt.Sprintf("%s Added Tables", s.displayService.RenderIconWithColor("add")),
			fmt.Sprintf("%d", len(diff.AddedTables)),
			details,
		})
	}

	if len(diff.RemovedTables) > 0 {
		details := fmt.Sprintf("Tables: %s", s.formatTableNames(diff.RemovedTables))
		rows = append(rows, []string{
			fmt.Sprintf("%s Removed Tables", s.displayService.RenderIconWithColor("remove")),
			fmt.Sprintf("%d", len(diff.RemovedTables)),
			details,
		})
	}

	if len(diff.ModifiedTables) > 0 {
		details := fmt.Sprintf("Tables: %s", s.formatModifiedTableNames(diff.ModifiedTables))
		rows = append(rows, []string{
			fmt.Sprintf("%s Modified Tables", s.displayService.RenderIconWithColor("modify")),
			fmt.Sprintf("%d", len(diff.ModifiedTables)),
			details,
		})
	}

	if len(diff.AddedIndexes) > 0 {
		details := fmt.Sprintf("Indexes: %s", s.formatIndexNames(diff.AddedIndexes))
		rows = append(rows, []string{
			fmt.Sprintf("%s Added Indexes", s.displayService.RenderIconWithColor("add")),
			fmt.Sprintf("%d", len(diff.AddedIndexes)),
			details,
		})
	}

	if len(diff.RemovedIndexes) > 0 {
		details := fmt.Sprintf("Indexes: %s", s.formatIndexNames(diff.RemovedIndexes))
		rows = append(rows, []string{
			fmt.Sprintf("%s Removed Indexes", s.displayService.RenderIconWithColor("remove")),
			fmt.Sprintf("%d", len(diff.RemovedIndexes)),
			details,
		})
	}

	if len(rows) > 0 {
		s.displayService.PrintTable(headers, rows)
	}

	// Show warnings for potentially destructive changes
	warnings := s.DetectComplexModifications(diff)
	if len(warnings) > 0 {
		s.displayService.Warning("Potentially destructive changes detected:")
		for _, warning := range warnings {
			s.displayService.Warning(fmt.Sprintf("  %s %s",
				s.displayService.RenderIconWithColor("warning"), warning))
		}
	}
}

// formatTableNames formats a list of tables for display
func (s *Service) formatTableNames(tables []*Table) string {
	if len(tables) == 0 {
		return ""
	}

	names := make([]string, len(tables))
	for i, table := range tables {
		names[i] = table.Name
	}

	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}

	return fmt.Sprintf("%s, ... (%d more)", strings.Join(names[:3], ", "), len(names)-3)
}

// formatModifiedTableNames formats a list of modified tables for display
func (s *Service) formatModifiedTableNames(tableDiffs []*TableDiff) string {
	if len(tableDiffs) == 0 {
		return ""
	}

	names := make([]string, len(tableDiffs))
	for i, tableDiff := range tableDiffs {
		names[i] = tableDiff.TableName
	}

	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}

	return fmt.Sprintf("%s, ... (%d more)", strings.Join(names[:3], ", "), len(names)-3)
}

// formatIndexNames formats a list of indexes for display
func (s *Service) formatIndexNames(indexes []*Index) string {
	if len(indexes) == 0 {
		return ""
	}

	names := make([]string, len(indexes))
	for i, index := range indexes {
		if index.TableName != "" {
			names[i] = fmt.Sprintf("%s.%s", index.TableName, index.Name)
		} else {
			names[i] = index.Name
		}
	}

	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}

	return fmt.Sprintf("%s, ... (%d more)", strings.Join(names[:3], ", "), len(names)-3)
}
