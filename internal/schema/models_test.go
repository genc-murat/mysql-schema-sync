package schema

import (
	"testing"
)

func TestSchemaValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  *Schema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: &Schema{
				Name:    "test_db",
				Tables:  make(map[string]*Table),
				Indexes: make(map[string]*Index),
			},
			wantErr: false,
		},
		{
			name: "empty schema name",
			schema: &Schema{
				Name:    "",
				Tables:  make(map[string]*Table),
				Indexes: make(map[string]*Index),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Schema.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTableValidation(t *testing.T) {
	tests := []struct {
		name    string
		table   *Table
		wantErr bool
	}{
		{
			name: "valid table",
			table: &Table{
				Name: "users",
				Columns: map[string]*Column{
					"id": {
						Name:     "id",
						DataType: "int",
						Position: 1,
					},
				},
				Indexes: []*Index{},
			},
			wantErr: false,
		},
		{
			name: "empty table name",
			table: &Table{
				Name:    "",
				Columns: make(map[string]*Column),
				Indexes: []*Index{},
			},
			wantErr: true,
		},
		{
			name: "table without columns",
			table: &Table{
				Name:    "empty_table",
				Columns: make(map[string]*Column),
				Indexes: []*Index{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.table.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Table.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestColumnValidation(t *testing.T) {
	tests := []struct {
		name    string
		column  *Column
		wantErr bool
	}{
		{
			name: "valid column",
			column: &Column{
				Name:     "id",
				DataType: "int",
				Position: 1,
			},
			wantErr: false,
		},
		{
			name: "empty column name",
			column: &Column{
				Name:     "",
				DataType: "int",
				Position: 1,
			},
			wantErr: true,
		},
		{
			name: "empty data type",
			column: &Column{
				Name:     "id",
				DataType: "",
				Position: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid data type",
			column: &Column{
				Name:     "id",
				DataType: "invalid_type",
				Position: 1,
			},
			wantErr: true,
		},
		{
			name: "negative position",
			column: &Column{
				Name:     "id",
				DataType: "int",
				Position: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.column.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Column.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexValidation(t *testing.T) {
	tests := []struct {
		name    string
		index   *Index
		wantErr bool
	}{
		{
			name: "valid index",
			index: &Index{
				Name:      "idx_user_id",
				TableName: "users",
				Columns:   []string{"id"},
				IndexType: "BTREE",
			},
			wantErr: false,
		},
		{
			name: "empty index name",
			index: &Index{
				Name:      "",
				TableName: "users",
				Columns:   []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "empty table name",
			index: &Index{
				Name:      "idx_user_id",
				TableName: "",
				Columns:   []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "no columns",
			index: &Index{
				Name:      "idx_user_id",
				TableName: "users",
				Columns:   []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid index type",
			index: &Index{
				Name:      "idx_user_id",
				TableName: "users",
				Columns:   []string{"id"},
				IndexType: "INVALID",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.index.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Index.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMySQLDataTypeValidation(t *testing.T) {
	validTypes := []string{
		"int", "INT", "varchar", "VARCHAR(255)", "text", "datetime", "decimal(10,2)",
		"tinyint", "bigint", "float", "double", "char(10)", "json", "timestamp",
	}

	invalidTypes := []string{
		"invalid_type", "string", "number", "boolean", "",
	}

	for _, dataType := range validTypes {
		if !isValidMySQLDataType(dataType) {
			t.Errorf("Expected %s to be valid MySQL data type", dataType)
		}
	}

	for _, dataType := range invalidTypes {
		if isValidMySQLDataType(dataType) {
			t.Errorf("Expected %s to be invalid MySQL data type", dataType)
		}
	}
}

func TestUtilityMethods(t *testing.T) {
	// Test NewSchema
	schema := NewSchema("test_db")
	if schema.Name != "test_db" {
		t.Errorf("Expected schema name 'test_db', got %s", schema.Name)
	}
	if schema.Tables == nil {
		t.Error("Expected Tables map to be initialized")
	}

	// Test NewTable
	table := NewTable("users")
	if table.Name != "users" {
		t.Errorf("Expected table name 'users', got %s", table.Name)
	}
	if table.Columns == nil {
		t.Error("Expected Columns map to be initialized")
	}

	// Test AddTable
	err := schema.AddTable(table)
	if err == nil {
		t.Error("Expected error when adding table without columns")
	}

	// Add a column to make table valid
	column := NewColumn("id", "int", false)
	err = table.AddColumn(column)
	if err != nil {
		t.Errorf("Unexpected error adding column: %v", err)
	}

	// Now adding table should work
	err = schema.AddTable(table)
	if err != nil {
		t.Errorf("Unexpected error adding table: %v", err)
	}

	// Test GetTable
	retrievedTable, exists := schema.GetTable("users")
	if !exists {
		t.Error("Expected table to exist")
	}
	if retrievedTable.Name != "users" {
		t.Errorf("Expected table name 'users', got %s", retrievedTable.Name)
	}
}
