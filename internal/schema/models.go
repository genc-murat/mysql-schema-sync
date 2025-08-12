package schema

import (
	"fmt"
	"strings"
)

// Schema represents a complete database schema
type Schema struct {
	Name    string            `json:"name"`
	Tables  map[string]*Table `json:"tables"`
	Indexes map[string]*Index `json:"indexes"`
}

// Table represents a database table
type Table struct {
	Name        string                 `json:"name"`
	Columns     map[string]*Column     `json:"columns"`
	Indexes     []*Index               `json:"indexes"`
	Constraints map[string]*Constraint `json:"constraints"`
}

// Column represents a table column
type Column struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value"`
	Extra        string  `json:"extra"`
	Position     int     `json:"position"`
}

// Index represents a database index
type Index struct {
	Name      string   `json:"name"`
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
	IndexType string   `json:"index_type"`
}

// ConstraintType represents the type of database constraint
type ConstraintType string

const (
	ConstraintTypeForeignKey ConstraintType = "FOREIGN_KEY"
	ConstraintTypeUnique     ConstraintType = "UNIQUE"
	ConstraintTypeCheck      ConstraintType = "CHECK"
)

// Constraint represents a database constraint
type Constraint struct {
	Name              string         `json:"name"`
	TableName         string         `json:"table_name"`
	Type              ConstraintType `json:"type"`
	Columns           []string       `json:"columns"`
	ReferencedTable   string         `json:"referenced_table,omitempty"`
	ReferencedColumns []string       `json:"referenced_columns,omitempty"`
	OnUpdate          string         `json:"on_update,omitempty"`
	OnDelete          string         `json:"on_delete,omitempty"`
	CheckExpression   string         `json:"check_expression,omitempty"`
}

// SchemaDiff represents differences between two schemas
type SchemaDiff struct {
	AddedTables        []*Table      `json:"added_tables"`
	RemovedTables      []*Table      `json:"removed_tables"`
	ModifiedTables     []*TableDiff  `json:"modified_tables"`
	AddedIndexes       []*Index      `json:"added_indexes"`
	RemovedIndexes     []*Index      `json:"removed_indexes"`
	AddedConstraints   []*Constraint `json:"added_constraints"`
	RemovedConstraints []*Constraint `json:"removed_constraints"`
}

// TableDiff represents differences between two tables
type TableDiff struct {
	TableName          string        `json:"table_name"`
	AddedColumns       []*Column     `json:"added_columns"`
	RemovedColumns     []*Column     `json:"removed_columns"`
	ModifiedColumns    []*ColumnDiff `json:"modified_columns"`
	AddedConstraints   []*Constraint `json:"added_constraints"`
	RemovedConstraints []*Constraint `json:"removed_constraints"`
}

// ColumnDiff represents differences between two columns
type ColumnDiff struct {
	ColumnName string  `json:"column_name"`
	OldColumn  *Column `json:"old_column"`
	NewColumn  *Column `json:"new_column"`
}

// Validation methods

// Validate validates the Schema structure
func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	if s.Tables == nil {
		s.Tables = make(map[string]*Table)
	}

	if s.Indexes == nil {
		s.Indexes = make(map[string]*Index)
	}

	// Validate all tables
	for tableName, table := range s.Tables {
		if err := table.Validate(); err != nil {
			return fmt.Errorf("invalid table %s: %w", tableName, err)
		}
	}

	// Validate all indexes
	for indexName, index := range s.Indexes {
		if err := index.Validate(); err != nil {
			return fmt.Errorf("invalid index %s: %w", indexName, err)
		}
	}

	return nil
}

// Validate validates the Table structure
func (t *Table) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	if t.Columns == nil {
		t.Columns = make(map[string]*Column)
	}

	if t.Constraints == nil {
		t.Constraints = make(map[string]*Constraint)
	}

	if len(t.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}

	// Validate all columns
	for columnName, column := range t.Columns {
		if err := column.Validate(); err != nil {
			return fmt.Errorf("invalid column %s: %w", columnName, err)
		}
	}

	// Validate indexes
	for _, index := range t.Indexes {
		if err := index.Validate(); err != nil {
			return fmt.Errorf("invalid index %s: %w", index.Name, err)
		}

		// Ensure index belongs to this table
		if index.TableName != t.Name {
			return fmt.Errorf("index %s table name mismatch: expected %s, got %s",
				index.Name, t.Name, index.TableName)
		}
	}

	// Validate constraints
	for constraintName, constraint := range t.Constraints {
		if err := constraint.Validate(); err != nil {
			return fmt.Errorf("invalid constraint %s: %w", constraintName, err)
		}

		// Ensure constraint belongs to this table
		if constraint.TableName != t.Name {
			return fmt.Errorf("constraint %s table name mismatch: expected %s, got %s",
				constraint.Name, t.Name, constraint.TableName)
		}
	}

	return nil
}

// Validate validates the Column structure
func (c *Column) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	if c.DataType == "" {
		return fmt.Errorf("column data type cannot be empty")
	}

	// Validate MySQL data types
	if !isValidMySQLDataType(c.DataType) {
		return fmt.Errorf("invalid MySQL data type: %s", c.DataType)
	}

	if c.Position < 0 {
		return fmt.Errorf("column position must be non-negative")
	}

	return nil
}

// Validate validates the Index structure
func (i *Index) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	if i.TableName == "" {
		return fmt.Errorf("index table name cannot be empty")
	}

	if len(i.Columns) == 0 {
		return fmt.Errorf("index must have at least one column")
	}

	// Validate index type
	if i.IndexType != "" && !isValidIndexType(i.IndexType) {
		return fmt.Errorf("invalid index type: %s", i.IndexType)
	}

	return nil
}

// Validate validates the Constraint structure
func (c *Constraint) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("constraint name cannot be empty")
	}

	if c.TableName == "" {
		return fmt.Errorf("constraint table name cannot be empty")
	}

	if c.Type == "" {
		return fmt.Errorf("constraint type cannot be empty")
	}

	// Validate constraint type
	switch c.Type {
	case ConstraintTypeForeignKey:
		if c.ReferencedTable == "" {
			return fmt.Errorf("foreign key constraint must have referenced table")
		}
		if len(c.ReferencedColumns) == 0 {
			return fmt.Errorf("foreign key constraint must have referenced columns")
		}
		if len(c.Columns) != len(c.ReferencedColumns) {
			return fmt.Errorf("foreign key constraint must have same number of columns and referenced columns")
		}
	case ConstraintTypeUnique:
		if len(c.Columns) == 0 {
			return fmt.Errorf("unique constraint must have at least one column")
		}
	case ConstraintTypeCheck:
		if c.CheckExpression == "" {
			return fmt.Errorf("check constraint must have check expression")
		}
	default:
		return fmt.Errorf("invalid constraint type: %s", c.Type)
	}

	if len(c.Columns) == 0 {
		return fmt.Errorf("constraint must have at least one column")
	}

	return nil
}

// Helper functions for validation

// isValidMySQLDataType checks if the data type is a valid MySQL data type
func isValidMySQLDataType(dataType string) bool {
	// Convert to lowercase for comparison
	dt := strings.ToLower(strings.TrimSpace(dataType))

	// Extract base type (remove size/precision info)
	if idx := strings.Index(dt, "("); idx != -1 {
		dt = dt[:idx]
	}

	validTypes := map[string]bool{
		// Numeric types
		"tinyint": true, "smallint": true, "mediumint": true, "int": true, "integer": true,
		"bigint": true, "decimal": true, "numeric": true, "float": true, "double": true,
		"bit": true,

		// String types
		"char": true, "varchar": true, "binary": true, "varbinary": true,
		"tinyblob": true, "blob": true, "mediumblob": true, "longblob": true,
		"tinytext": true, "text": true, "mediumtext": true, "longtext": true,

		// Date and time types
		"date": true, "time": true, "datetime": true, "timestamp": true, "year": true,

		// JSON type
		"json": true,

		// Enum and Set
		"enum": true, "set": true,

		// Geometry types
		"geometry": true, "point": true, "linestring": true, "polygon": true,
		"multipoint": true, "multilinestring": true, "multipolygon": true, "geometrycollection": true,
	}

	return validTypes[dt]
}

// isValidIndexType checks if the index type is valid
func isValidIndexType(indexType string) bool {
	validTypes := map[string]bool{
		"BTREE":    true,
		"HASH":     true,
		"RTREE":    true,
		"FULLTEXT": true,
	}

	return validTypes[strings.ToUpper(indexType)]
}

// Utility methods

// NewSchema creates a new Schema instance
func NewSchema(name string) *Schema {
	return &Schema{
		Name:    name,
		Tables:  make(map[string]*Table),
		Indexes: make(map[string]*Index),
	}
}

// NewTable creates a new Table instance
func NewTable(name string) *Table {
	return &Table{
		Name:        name,
		Columns:     make(map[string]*Column),
		Indexes:     make([]*Index, 0),
		Constraints: make(map[string]*Constraint),
	}
}

// NewColumn creates a new Column instance
func NewColumn(name, dataType string, isNullable bool) *Column {
	return &Column{
		Name:       name,
		DataType:   dataType,
		IsNullable: isNullable,
	}
}

// NewIndex creates a new Index instance
func NewIndex(name, tableName string, columns []string) *Index {
	return &Index{
		Name:      name,
		TableName: tableName,
		Columns:   columns,
		IsUnique:  false,
		IsPrimary: false,
		IndexType: "BTREE",
	}
}

// NewConstraint creates a new Constraint instance
func NewConstraint(name, tableName string, constraintType ConstraintType, columns []string) *Constraint {
	return &Constraint{
		Name:      name,
		TableName: tableName,
		Type:      constraintType,
		Columns:   columns,
	}
}

// NewForeignKeyConstraint creates a new foreign key constraint
func NewForeignKeyConstraint(name, tableName string, columns []string, referencedTable string, referencedColumns []string) *Constraint {
	return &Constraint{
		Name:              name,
		TableName:         tableName,
		Type:              ConstraintTypeForeignKey,
		Columns:           columns,
		ReferencedTable:   referencedTable,
		ReferencedColumns: referencedColumns,
		OnUpdate:          "RESTRICT",
		OnDelete:          "RESTRICT",
	}
}

// AddTable adds a table to the schema
func (s *Schema) AddTable(table *Table) error {
	if err := table.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid table: %w", err)
	}

	s.Tables[table.Name] = table
	return nil
}

// AddColumn adds a column to the table
func (t *Table) AddColumn(column *Column) error {
	if err := column.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid column: %w", err)
	}

	t.Columns[column.Name] = column
	return nil
}

// AddIndex adds an index to the table
func (t *Table) AddIndex(index *Index) error {
	if err := index.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid index: %w", err)
	}

	// Ensure index belongs to this table
	if index.TableName != t.Name {
		return fmt.Errorf("index table name mismatch: expected %s, got %s",
			t.Name, index.TableName)
	}

	t.Indexes = append(t.Indexes, index)
	return nil
}

// AddConstraint adds a constraint to the table
func (t *Table) AddConstraint(constraint *Constraint) error {
	if err := constraint.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid constraint: %w", err)
	}

	// Ensure constraint belongs to this table
	if constraint.TableName != t.Name {
		return fmt.Errorf("constraint table name mismatch: expected %s, got %s",
			t.Name, constraint.TableName)
	}

	t.Constraints[constraint.Name] = constraint
	return nil
}

// GetTable retrieves a table by name
func (s *Schema) GetTable(name string) (*Table, bool) {
	table, exists := s.Tables[name]
	return table, exists
}

// GetColumn retrieves a column by name
func (t *Table) GetColumn(name string) (*Column, bool) {
	column, exists := t.Columns[name]
	return column, exists
}

// HasPrimaryKey checks if the table has a primary key
func (t *Table) HasPrimaryKey() bool {
	for _, index := range t.Indexes {
		if index.IsPrimary {
			return true
		}
	}
	return false
}

// GetPrimaryKey returns the primary key index if it exists
func (t *Table) GetPrimaryKey() *Index {
	for _, index := range t.Indexes {
		if index.IsPrimary {
			return index
		}
	}
	return nil
}
