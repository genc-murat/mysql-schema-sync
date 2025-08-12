package schema

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewExtractor(t *testing.T) {
	extractor := NewExtractor()
	if extractor == nil {
		t.Fatal("Expected extractor to be created")
	}
	if extractor.queryTimeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", extractor.queryTimeout)
	}
}

func TestNewExtractorWithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	extractor := NewExtractorWithTimeout(timeout)
	if extractor.queryTimeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, extractor.queryTimeout)
	}
}

func TestExtractSchema_NilDB(t *testing.T) {
	extractor := NewExtractor()
	_, err := extractor.ExtractSchema(nil, "test_db")
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestExtractSchema_EmptySchemaName(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	extractor := NewExtractor()
	_, err = extractor.ExtractSchema(db, "")
	if err == nil {
		t.Error("Expected error for empty schema name")
	}
}

func TestExtractTables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock the tables query
	rows := sqlmock.NewRows([]string{"TABLE_NAME"}).
		AddRow("users").
		AddRow("posts")

	mock.ExpectQuery("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES").
		WithArgs("test_db").
		WillReturnRows(rows)

	extractor := NewExtractor()
	tables, err := extractor.extractTables(db, "test_db")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables))
	}

	if _, exists := tables["users"]; !exists {
		t.Error("Expected 'users' table to exist")
	}

	if _, exists := tables["posts"]; !exists {
		t.Error("Expected 'posts' table to exist")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestExtractColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock the columns query
	rows := sqlmock.NewRows([]string{
		"COLUMN_NAME", "DATA_TYPE", "IS_NULLABLE", "COLUMN_DEFAULT",
		"EXTRA", "ORDINAL_POSITION", "COLUMN_TYPE",
	}).
		AddRow("id", "int", "NO", nil, "auto_increment", 1, "int(11)").
		AddRow("name", "varchar", "YES", nil, "", 2, "varchar(255)").
		AddRow("created_at", "timestamp", "NO", "CURRENT_TIMESTAMP", "", 3, "timestamp")

	mock.ExpectQuery("SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE").
		WithArgs("test_db", "users").
		WillReturnRows(rows)

	extractor := NewExtractor()
	columns, err := extractor.extractColumns(db, "test_db", "users")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	// Test id column
	idCol, exists := columns["id"]
	if !exists {
		t.Error("Expected 'id' column to exist")
	} else {
		if idCol.DataType != "int(11)" {
			t.Errorf("Expected id column type 'int(11)', got %s", idCol.DataType)
		}
		if idCol.IsNullable {
			t.Error("Expected id column to be NOT NULL")
		}
		if idCol.Extra != "auto_increment" {
			t.Errorf("Expected id column extra 'auto_increment', got %s", idCol.Extra)
		}
		if idCol.Position != 1 {
			t.Errorf("Expected id column position 1, got %d", idCol.Position)
		}
	}

	// Test name column
	nameCol, exists := columns["name"]
	if !exists {
		t.Error("Expected 'name' column to exist")
	} else {
		if nameCol.DataType != "varchar(255)" {
			t.Errorf("Expected name column type 'varchar(255)', got %s", nameCol.DataType)
		}
		if !nameCol.IsNullable {
			t.Error("Expected name column to be nullable")
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestExtractIndexes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock the indexes query
	rows := sqlmock.NewRows([]string{
		"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE", "INDEX_TYPE", "SEQ_IN_INDEX",
	}).
		AddRow("PRIMARY", "id", 0, "BTREE", 1).
		AddRow("idx_name", "name", 1, "BTREE", 1).
		AddRow("idx_composite", "name", 1, "BTREE", 1).
		AddRow("idx_composite", "created_at", 1, "BTREE", 2)

	mock.ExpectQuery("SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE").
		WithArgs("test_db", "users").
		WillReturnRows(rows)

	extractor := NewExtractor()
	indexes, err := extractor.extractIndexes(db, "test_db", "users")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(indexes) != 3 {
		t.Errorf("Expected 3 indexes, got %d", len(indexes))
	}

	// Find and test PRIMARY index
	var primaryIndex *Index
	for _, idx := range indexes {
		if idx.Name == "PRIMARY" {
			primaryIndex = idx
			break
		}
	}
	if primaryIndex == nil {
		t.Error("Expected PRIMARY index to exist")
	} else {
		if !primaryIndex.IsPrimary {
			t.Error("Expected PRIMARY index to be marked as primary")
		}
		if !primaryIndex.IsUnique {
			t.Error("Expected PRIMARY index to be unique")
		}
		if len(primaryIndex.Columns) != 1 || primaryIndex.Columns[0] != "id" {
			t.Errorf("Expected PRIMARY index to have column 'id', got %v", primaryIndex.Columns)
		}
	}

	// Find and test composite index
	var compositeIndex *Index
	for _, idx := range indexes {
		if idx.Name == "idx_composite" {
			compositeIndex = idx
			break
		}
	}
	if compositeIndex == nil {
		t.Error("Expected composite index to exist")
	} else {
		if len(compositeIndex.Columns) != 2 {
			t.Errorf("Expected composite index to have 2 columns, got %d", len(compositeIndex.Columns))
		}
		if compositeIndex.Columns[0] != "name" || compositeIndex.Columns[1] != "created_at" {
			t.Errorf("Expected composite index columns [name, created_at], got %v", compositeIndex.Columns)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetCurrentSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock the current schema query
	rows := sqlmock.NewRows([]string{"DATABASE()"}).AddRow("test_db")
	mock.ExpectQuery("SELECT DATABASE\\(\\)").WillReturnRows(rows)

	extractor := NewExtractor()
	schema, err := extractor.GetCurrentSchema(db)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if schema != "test_db" {
		t.Errorf("Expected schema 'test_db', got %s", schema)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetCurrentSchema_NilDB(t *testing.T) {
	extractor := NewExtractor()
	_, err := extractor.GetCurrentSchema(nil)
	if err == nil {
		t.Error("Expected error for nil database connection")
	}
}

func TestValidateSchemaExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock schema existence check
	rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM INFORMATION_SCHEMA.SCHEMATA").
		WithArgs("test_db").
		WillReturnRows(rows)

	extractor := NewExtractor()
	err = extractor.ValidateSchemaExists(db, "test_db")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestValidateSchemaExists_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock schema not found
	rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM INFORMATION_SCHEMA.SCHEMATA").
		WithArgs("nonexistent_db").
		WillReturnRows(rows)

	extractor := NewExtractor()
	err = extractor.ValidateSchemaExists(db, "nonexistent_db")
	if err == nil {
		t.Error("Expected error for nonexistent schema")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
