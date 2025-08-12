# Design Document

## Overview

The MySQL Schema Sync CLI is a Go application that compares database schemas between source and target MySQL databases, identifies differences, and applies approved changes. The application follows a modular architecture with clear separation of concerns for database operations, schema comparison, and change application.

## Architecture

The application uses a layered architecture:

```
CLI Layer (Cobra) → Service Layer → Repository Layer → Database Layer
```

### Core Components:
- **CLI Interface**: Command-line interface using Cobra framework
- **Database Service**: Handles database connections and operations
- **Schema Service**: Manages schema extraction and comparison
- **Migration Service**: Handles change application and SQL generation
- **Configuration**: Database connection and application settings

## Components and Interfaces

### 1. CLI Interface (`cmd/`)
```go
type CLIConfig struct {
    SourceDB     DatabaseConfig
    TargetDB     DatabaseConfig
    DryRun       bool
    Verbose      bool
    AutoApprove  bool
}
```

### 2. Database Service (`internal/database/`)
```go
type DatabaseService interface {
    Connect(config DatabaseConfig) (*sql.DB, error)
    GetSchema(db *sql.DB) (*Schema, error)
    ExecuteSQL(db *sql.DB, statements []string) error
    Close(db *sql.DB) error
}

type DatabaseConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    Database string
}
```

### 3. Schema Service (`internal/schema/`)
```go
type SchemaService interface {
    CompareSchemas(source, target *Schema) (*SchemaDiff, error)
    GenerateSQL(diff *SchemaDiff) ([]string, error)
}

type Schema struct {
    Tables  map[string]*Table
    Indexes map[string]*Index
}

type Table struct {
    Name    string
    Columns map[string]*Column
    Indexes []*Index
}

type Column struct {
    Name         string
    DataType     string
    IsNullable   bool
    DefaultValue *string
    Extra        string
}

type SchemaDiff struct {
    AddedTables    []*Table
    RemovedTables  []*Table
    ModifiedTables []*TableDiff
    AddedIndexes   []*Index
    RemovedIndexes []*Index
}

type TableDiff struct {
    TableName       string
    AddedColumns    []*Column
    RemovedColumns  []*Column
    ModifiedColumns []*ColumnDiff
}
```

### 4. Migration Service (`internal/migration/`)
```go
type MigrationService interface {
    PlanMigration(diff *SchemaDiff) (*MigrationPlan, error)
    ExecuteMigration(db *sql.DB, plan *MigrationPlan) error
}

type MigrationPlan struct {
    Statements []MigrationStatement
    Warnings   []string
}

type MigrationStatement struct {
    SQL         string
    Type        StatementType
    Description string
    IsDestructive bool
}
```

## Data Models

### Schema Information Extraction
The application extracts schema information using MySQL's `INFORMATION_SCHEMA` tables:
- `INFORMATION_SCHEMA.TABLES`
- `INFORMATION_SCHEMA.COLUMNS`
- `INFORMATION_SCHEMA.STATISTICS`
- `INFORMATION_SCHEMA.KEY_COLUMN_USAGE`

### Change Detection Algorithm
1. **Table-level changes**: Compare table lists between source and target
2. **Column-level changes**: For each common table, compare column definitions
3. **Index changes**: Compare index definitions and compositions
4. **Constraint changes**: Compare foreign keys and unique constraints

### SQL Generation Strategy
Changes are ordered to prevent dependency conflicts:
1. Drop foreign key constraints
2. Drop indexes
3. Drop columns/tables
4. Add/modify tables
5. Add/modify columns
6. Add indexes
7. Add foreign key constraints

## Error Handling

### Connection Errors
- Retry logic with exponential backoff
- Clear error messages for common connection issues
- Graceful degradation when one database is unavailable

### SQL Execution Errors
- Transaction-based execution for atomic operations
- Rollback capability for failed migrations
- Detailed error logging with SQL context

### Schema Parsing Errors
- Validation of extracted schema information
- Handling of unsupported MySQL features
- Graceful handling of permission issues

## Testing Strategy

### Unit Tests
- Database service mocking for connection testing
- Schema comparison logic testing with various scenarios
- SQL generation testing for different change types
- Migration planning and execution testing

### Integration Tests
- Real MySQL database testing with Docker containers
- End-to-end CLI testing with sample databases
- Error scenario testing (connection failures, permission issues)

### Test Data
- Sample databases with various schema configurations
- Migration scenarios covering all supported change types
- Edge cases (empty databases, identical schemas, complex relationships)

## Configuration and Deployment

### Configuration Options
```yaml
# config.yaml
source:
  host: localhost
  port: 3306
  username: root
  password: password
  database: source_db

target:
  host: localhost
  port: 3306
  username: root
  password: password
  database: target_db

options:
  dry_run: false
  verbose: false
  auto_approve: false
  timeout: 30s
```

### CLI Usage
```bash
# Basic usage
mysql-schema-sync --source-host=src.db --target-host=tgt.db

# With configuration file
mysql-schema-sync --config=config.yaml

# Dry run mode
mysql-schema-sync --config=config.yaml --dry-run

# Auto-approve changes
mysql-schema-sync --config=config.yaml --auto-approve
```

### Build and Distribution
- Cross-platform binary compilation
- Docker container for isolated execution
- GitHub Actions for automated builds and releases