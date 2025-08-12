# Contributing to MySQL Schema Sync

Thank you for your interest in contributing to MySQL Schema Sync! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please treat all contributors and users with respect.

## Getting Started

### Prerequisites

- Go 1.24.5 or later
- MySQL 5.7+ or 8.0+ for testing
- Docker (for integration tests)
- Git

### Development Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/mysql-schema-sync.git
   cd mysql-schema-sync
   ```

3. Install dependencies:
   ```bash
   make deps
   ```

4. Install development tools:
   ```bash
   make install-tools
   ```

5. Run tests to ensure everything works:
   ```bash
   make test
   ```

## Development Workflow

### Making Changes

1. Create a new branch for your feature or bug fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards below

3. Add or update tests for your changes

4. Run the test suite:
   ```bash
   make test-all
   ```

5. Run the linter:
   ```bash
   make lint
   ```

6. Format your code:
   ```bash
   make fmt
   ```

### Commit Guidelines

- Use clear, descriptive commit messages
- Follow the conventional commit format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `test:` for test additions or modifications
  - `refactor:` for code refactoring
  - `chore:` for maintenance tasks

Example:
```
feat: add support for foreign key constraints

- Implement foreign key detection in schema extractor
- Add SQL generation for foreign key creation/deletion
- Update tests to cover foreign key scenarios
```

### Pull Request Process

1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a pull request on GitHub with:
   - Clear title and description
   - Reference to any related issues
   - List of changes made
   - Testing instructions

3. Ensure all CI checks pass

4. Address any review feedback

## Coding Standards

### Go Style Guide

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting (included in `make fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and reasonably sized

### Project Structure

```
mysql-schema-sync/
├── cmd/                    # CLI commands and entry points
├── internal/               # Internal application code
│   ├── application/        # Main application logic
│   ├── database/          # Database connection and configuration
│   ├── schema/            # Schema extraction and comparison
│   ├── migration/         # Migration planning and execution
│   ├── confirmation/      # User confirmation handling
│   ├── execution/         # SQL execution logic
│   ├── logging/           # Logging utilities
│   └── errors/            # Error handling utilities
├── docs/                  # Documentation
├── examples/              # Example configurations
└── tests/                 # Test utilities and fixtures
```

### Testing Guidelines

#### Unit Tests

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for high test coverage (>80%)

Example:
```go
func TestSchemaComparison(t *testing.T) {
    tests := []struct {
        name     string
        source   *Schema
        target   *Schema
        expected *SchemaDiff
    }{
        {
            name: "detect added table",
            source: &Schema{
                Tables: map[string]*Table{
                    "users": {Name: "users"},
                },
            },
            target: &Schema{
                Tables: map[string]*Table{},
            },
            expected: &SchemaDiff{
                AddedTables: []*Table{{Name: "users"}},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := CompareSchemas(tt.source, tt.target)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### Integration Tests

- Test real database interactions
- Use Docker containers for consistent test environments
- Clean up test data after each test
- Tag integration tests: `// +build integration`

### Error Handling

- Use structured error handling with context
- Wrap errors with meaningful messages
- Use custom error types for specific error categories

Example:
```go
func (s *SchemaService) ExtractSchema(db *sql.DB) (*Schema, error) {
    tables, err := s.extractTables(db)
    if err != nil {
        return nil, fmt.Errorf("failed to extract tables: %w", err)
    }
    
    return &Schema{Tables: tables}, nil
}
```

### Logging

- Use structured logging with appropriate levels
- Include relevant context in log messages
- Use the project's logging utilities

Example:
```go
logger.Info("extracting schema",
    "database", config.Database,
    "host", config.Host,
    "table_count", len(tables))
```

## Testing

### Running Tests

```bash
# Unit tests only
make test-unit

# Integration tests (requires MySQL)
make test-integration

# All tests
make test-all

# Tests with coverage
make coverage

# Tests with race detection
make test-race
```

### Docker Testing

Integration tests use Docker containers to provide consistent MySQL environments:

```bash
# Run all tests in Docker
make docker-test

# Clean up Docker resources
make docker-clean
```

### Test Database Setup

For local integration testing, you can set up MySQL using Docker:

```bash
docker run --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=testpass \
  -e MYSQL_DATABASE=testdb \
  -p 3306:3306 \
  -d mysql:8.0
```

## Documentation

### Code Documentation

- Add godoc comments for all exported functions and types
- Include usage examples in documentation
- Keep documentation up to date with code changes

### User Documentation

- Update README.md for user-facing changes
- Add examples for new features
- Update configuration documentation

## Issue Reporting

### Bug Reports

When reporting bugs, please include:

- Go version and operating system
- MySQL version
- Complete error messages
- Steps to reproduce
- Expected vs actual behavior
- Relevant configuration

### Feature Requests

For feature requests, please provide:

- Clear description of the feature
- Use case and motivation
- Proposed implementation approach (if any)
- Potential impact on existing functionality

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):
- MAJOR: Incompatible API changes
- MINOR: New functionality (backward compatible)
- PATCH: Bug fixes (backward compatible)

### Release Checklist

1. Update version in relevant files
2. Update CHANGELOG.md
3. Run full test suite
4. Create release tag
5. Build and publish binaries
6. Update documentation

## Getting Help

- Check existing [issues](https://github.com/your-org/mysql-schema-sync/issues)
- Join our [discussions](https://github.com/your-org/mysql-schema-sync/discussions)
- Read the [documentation](https://github.com/your-org/mysql-schema-sync/wiki)

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributor statistics

Thank you for contributing to MySQL Schema Sync!