# MySQL Schema Sync

A powerful CLI tool for synchronizing MySQL database schemas. Compare two MySQL databases, identify schema differences, and safely apply changes to keep your databases in sync.

## Features

- **Schema Comparison**: Compare tables, columns, indexes, and constraints between source and target databases
- **Safe Operations**: Dry-run mode to preview changes before applying them
- **Interactive Confirmation**: Review and approve changes before execution
- **Comprehensive Change Detection**: Detects additions, deletions, and modifications at all schema levels
- **Dependency-Aware**: Applies changes in the correct order to avoid conflicts
- **Flexible Configuration**: Support for CLI flags, configuration files, and environment variables
- **Detailed Logging**: Comprehensive logging with configurable verbosity levels
- **Cross-Platform**: Available for Linux, macOS, and Windows

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/your-org/mysql-schema-sync/releases).

### Build from Source

Requirements:
- Go 1.24.5 or later
- MySQL client libraries (for testing)

```bash
git clone https://github.com/your-org/mysql-schema-sync.git
cd mysql-schema-sync
make build
```

### Using Go Install

```bash
go install github.com/your-org/mysql-schema-sync@latest
```

## Quick Start

### Basic Usage

Compare two databases using command-line flags:

```bash
mysql-schema-sync \
  --source-host=localhost --source-user=root --source-db=source_db \
  --target-host=localhost --target-user=root --target-db=target_db
```

### Using Configuration File

Create a configuration file:

```bash
mysql-schema-sync config > config.yaml
```

Edit the configuration file with your database details, then run:

```bash
mysql-schema-sync --config=config.yaml
```

### Dry Run Mode

Preview changes without applying them:

```bash
mysql-schema-sync --config=config.yaml --dry-run
```

## Configuration

### Configuration File

The tool supports YAML configuration files. Generate a sample configuration:

```bash
mysql-schema-sync config > config.yaml
```

Example configuration:

```yaml
source:
  host: localhost
  port: 3306
  username: root
  password: secret
  database: source_db
  timeout: 30s

target:
  host: localhost
  port: 3306
  username: root
  password: secret
  database: target_db
  timeout: 30s

# Operation settings
dry_run: false      # Show changes without applying them
verbose: false      # Enable verbose output
quiet: false        # Suppress non-error output
auto_approve: false # Automatically approve changes without confirmation
timeout: 30s        # Global timeout for operations
log_file: ""        # Optional log file path
```

### Command Line Options

#### Database Connection Options

**Source Database:**
- `--source-host`: Source database host
- `--source-port`: Source database port (default: 3306)
- `--source-user`: Source database username
- `--source-password`: Source database password
- `--source-db`: Source database name

**Target Database:**
- `--target-host`: Target database host
- `--target-port`: Target database port (default: 3306)
- `--target-user`: Target database username
- `--target-password`: Target database password
- `--target-db`: Target database name

#### Operation Options

- `--config`: Configuration file path
- `--dry-run`: Show changes without applying them
- `--verbose, -v`: Enable verbose output
- `--quiet, -q`: Suppress non-error output
- `--auto-approve`: Automatically approve changes without confirmation
- `--timeout`: Database operation timeout (default: 30s)
- `--log-file`: Write logs to file instead of stdout

### Environment Variables

All configuration options can be set via environment variables with the prefix `MYSQL_SCHEMA_SYNC_`:

```bash
export MYSQL_SCHEMA_SYNC_SOURCE_HOST=localhost
export MYSQL_SCHEMA_SYNC_SOURCE_USER=root
export MYSQL_SCHEMA_SYNC_SOURCE_DB=source_db
export MYSQL_SCHEMA_SYNC_TARGET_HOST=localhost
export MYSQL_SCHEMA_SYNC_TARGET_USER=root
export MYSQL_SCHEMA_SYNC_TARGET_DB=target_db
```

## Usage Examples

### Development Workflow

Sync development database with production schema:

```bash
mysql-schema-sync \
  --source-host=prod.example.com --source-user=readonly --source-db=app_prod \
  --target-host=localhost --target-user=root --target-db=app_dev \
  --dry-run --verbose
```

### Automated Deployment

Use in CI/CD pipelines with auto-approve:

```bash
mysql-schema-sync \
  --config=production.yaml \
  --auto-approve \
  --log-file=schema-sync.log
```

### Safe Production Updates

Review changes before applying to production:

```bash
# First, preview changes
mysql-schema-sync --config=prod.yaml --dry-run

# If changes look good, apply them
mysql-schema-sync --config=prod.yaml --verbose
```

## Supported MySQL Features

### Fully Supported

- **Tables**: Creation, deletion, and modification
- **Columns**: All MySQL data types, constraints, and properties
- **Indexes**: Primary keys, unique indexes, regular indexes, composite indexes
- **Auto Increment**: Column auto increment settings
- **Default Values**: Column default values and expressions
- **Null Constraints**: NOT NULL and nullable columns
- **Character Sets**: Table and column character set specifications
- **Collations**: Table and column collation settings

### Partially Supported

- **Foreign Keys**: Basic foreign key constraints (complex cascading rules may need manual review)
- **Triggers**: Detection only (manual migration required)
- **Stored Procedures**: Detection only (manual migration required)
- **Views**: Detection only (manual migration required)

### Not Supported

- **Partitioning**: Table partitioning schemes
- **Full-Text Indexes**: Full-text search indexes
- **Spatial Indexes**: Spatial data indexes
- **User-Defined Functions**: Custom MySQL functions
- **Events**: MySQL event scheduler events

## Limitations and Best Practices

### Data Safety

- **Always use `--dry-run` first** to preview changes before applying them
- **Backup your databases** before running schema synchronization
- **Test in development** environments before applying to production
- **Review destructive operations** carefully (column/table deletions)

### Performance Considerations

- **Large databases**: Schema extraction may take time for databases with many tables
- **Network latency**: Use appropriate timeout values for remote databases
- **Concurrent access**: Avoid running during heavy database usage periods

### Security Best Practices

- **Use read-only users** for source database connections when possible
- **Store passwords securely**: Use configuration files with restricted permissions
- **Environment variables**: Prefer environment variables over command-line passwords
- **Network security**: Use SSL connections for remote databases

### Operational Guidelines

1. **Development Workflow**:
   - Use dry-run mode for initial assessment
   - Review all changes before approval
   - Test schema changes in development first

2. **Production Deployment**:
   - Schedule during maintenance windows
   - Have rollback procedures ready
   - Monitor application compatibility after changes

3. **Automation**:
   - Use auto-approve only in trusted environments
   - Implement proper logging and monitoring
   - Set up alerts for failed synchronizations

## Troubleshooting

### Common Issues

**Connection Errors**:
```
Error: failed to connect to source database: dial tcp: connection refused
```
- Verify database host and port
- Check network connectivity
- Ensure MySQL server is running
- Verify firewall settings

**Permission Errors**:
```
Error: Access denied for user 'username'@'host'
```
- Verify username and password
- Check user permissions on both databases
- Ensure user has SELECT privileges on source and appropriate privileges on target

**Schema Extraction Errors**:
```
Error: failed to extract schema: table doesn't exist
```
- Verify database names are correct
- Check user permissions on INFORMATION_SCHEMA tables
- Ensure databases are accessible

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
mysql-schema-sync --config=config.yaml --verbose --log-file=debug.log
```

### Getting Help

- Check the [documentation](https://github.com/your-org/mysql-schema-sync/wiki)
- Review [common issues](https://github.com/your-org/mysql-schema-sync/issues)
- Open a [new issue](https://github.com/your-org/mysql-schema-sync/issues/new) for bugs or feature requests

## Development

### Building from Source

```bash
git clone https://github.com/your-org/mysql-schema-sync.git
cd mysql-schema-sync
make deps
make build
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires MySQL)
make test-integration

# All tests
make test-all

# With coverage
make coverage
```

### Docker Testing

```bash
# Run tests in Docker containers
make docker-test

# Clean up Docker resources
make docker-clean
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.