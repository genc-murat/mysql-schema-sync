# MySQL Schema Sync

A powerful CLI tool for synchronizing MySQL database schemas. Compare two MySQL databases, identify schema differences, and safely apply changes to keep your databases in sync.

## Features

- **Schema Comparison**: Compare tables, columns, indexes, and constraints between source and target databases
- **Enhanced Visual Output**: Colorized output, progress indicators, and formatted tables for better readability
- **Multiple Output Formats**: Table, JSON, YAML, and compact formats for different use cases
- **Accessibility Support**: High contrast themes, screen reader compatibility, and graceful fallbacks
- **Safe Operations**: Dry-run mode to preview changes before applying them
- **Interactive Confirmation**: Enhanced confirmation dialogs with visual indicators and detailed change review
- **Comprehensive Change Detection**: Detects additions, deletions, and modifications at all schema levels
- **Dependency-Aware**: Applies changes in the correct order to avoid conflicts
- **Flexible Configuration**: Support for CLI flags, configuration files, and environment variables
- **Detailed Logging**: Comprehensive logging with configurable verbosity levels
- **Cross-Platform**: Available for Linux, macOS, and Windows with terminal compatibility

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

## Visual Enhancements

MySQL Schema Sync includes comprehensive visual enhancements to improve readability and user experience:

### Enhanced Output Examples

#### Colorized Schema Differences
```
┌─────────────────┬──────────────┬─────────────────────────────────┐
│ Change Type     │ Object       │ Description                     │
├─────────────────┼──────────────┼─────────────────────────────────┤
│ ➕ ADD TABLE    │ users        │ Create new table with 5 columns │
│ 🔄 MODIFY TABLE │ products     │ Add column 'created_at'         │
│ ➖ DROP COLUMN  │ orders.notes │ Remove unused notes column      │
└─────────────────┴──────────────┴─────────────────────────────────┘

Summary: 3 changes detected (1 addition, 1 modification, 1 deletion)
```

#### Progress Indicators
```
🔄 Connecting to source database...     ✅ Connected
🔄 Extracting source schema...          ✅ 15 tables processed
🔄 Connecting to target database...     ✅ Connected  
🔄 Extracting target schema...          ✅ 12 tables processed
🔄 Comparing schemas...                 ✅ Analysis complete

Progress: [████████████████████] 100% | 3/3 changes processed
```

#### Interactive Confirmations
```
⚠️  Schema Synchronization Confirmation

The following changes will be applied to the target database:

  ➕ CREATE TABLE users (5 columns)
  🔄 ALTER TABLE products ADD COLUMN created_at
  ➖ ALTER TABLE orders DROP COLUMN notes

❓ Do you want to proceed with these changes? [y/N]: 
```

### Visual Options

#### Color Themes
- **Dark Theme** (default): Bright colors on dark backgrounds
- **Light Theme**: Darker colors for light terminals  
- **High Contrast**: Enhanced contrast for accessibility
- **Auto**: Automatically detects terminal theme

#### Output Formats
- **Table**: Formatted tables with colors and styling (default)
- **JSON**: Machine-readable structured output
- **YAML**: Human-readable structured format
- **Compact**: Minimal output for scripting

#### Table Styles
- **Default**: Standard ASCII borders
- **Rounded**: Modern rounded corners (Unicode)
- **Border**: Heavy borders for emphasis
- **Minimal**: Clean, simple formatting

### Accessibility Features

- **Screen Reader Support**: Compatible with NVDA, JAWS, and VoiceOver
- **High Contrast Mode**: Enhanced visibility for visual impairments
- **ASCII Fallbacks**: Unicode icons automatically fall back to ASCII
- **Configurable Width**: Adjustable table width for different screen sizes
- **Keyboard Navigation**: Full keyboard support for interactive elements

### Usage Examples

```bash
# Use light theme with rounded tables
mysql-schema-sync --config=config.yaml --theme=light --table-style=rounded

# High contrast mode for accessibility
mysql-schema-sync --config=config.yaml --theme=high-contrast --no-icons

# JSON output for automation
mysql-schema-sync --config=config.yaml --format=json --no-color

# Compact output for scripting
mysql-schema-sync --config=config.yaml --format=compact --no-progress
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

# Visual enhancement settings
display:
  color_enabled: true      # Enable colorized output
  theme: dark              # Color theme (dark, light, high-contrast, auto)
  output_format: table     # Output format (table, json, yaml, compact)
  use_icons: true          # Enable Unicode icons with ASCII fallbacks
  show_progress: true      # Show progress indicators and spinners
  interactive: true        # Enable interactive confirmations
  table_style: default     # Table styling (default, rounded, border, minimal)
  max_table_width: 120     # Maximum table width (40-300)
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

#### Visual Enhancement Options

- `--no-color`: Disable color output
- `--theme`: Color theme (dark, light, high-contrast, auto) (default: dark)
- `--format`: Output format (table, json, yaml, compact) (default: table)
- `--no-icons`: Disable Unicode icons (use ASCII alternatives)
- `--no-progress`: Disable progress indicators and spinners
- `--no-interactive`: Disable interactive prompts and confirmations
- `--table-style`: Table style (default, rounded, border, minimal) (default: default)
- `--max-table-width`: Maximum table width in characters (40-300) (default: 120)

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

### Visual Display Issues

**Colors Not Displaying**:
```bash
# Force color output
mysql-schema-sync --config=config.yaml --theme=dark

# Check if terminal supports colors
mysql-schema-sync --config=config.yaml --verbose
```

**Unicode Icons Not Showing**:
```bash
# Use ASCII alternatives
mysql-schema-sync --config=config.yaml --no-icons

# Check terminal Unicode support
echo "Test: ➕ 🔄 ➖"
```

**Table Formatting Issues**:
```bash
# Use minimal table style for compatibility
mysql-schema-sync --config=config.yaml --table-style=minimal

# Adjust table width for narrow terminals
mysql-schema-sync --config=config.yaml --max-table-width=80
```

**Progress Indicators Not Working**:
```bash
# Disable progress bars for non-interactive environments
mysql-schema-sync --config=config.yaml --no-progress

# Check if running in TTY
tty && echo "Interactive terminal" || echo "Non-interactive"
```

### Terminal Compatibility

**Windows Command Prompt**:
- Limited color support
- No Unicode icon support
- Use: `--no-icons --table-style=minimal`

**Windows PowerShell**:
- Full color support
- Limited Unicode support
- Use: `--theme=auto --table-style=default`

**Windows Terminal**:
- Full feature support
- All visual enhancements work

**macOS Terminal**:
- Full feature support
- All themes and styles supported

**Linux Terminals**:
- Most modern terminals support all features
- Older terminals may need `--no-icons --no-color`

**SSH/Remote Sessions**:
- May have limited color/Unicode support
- Use: `--theme=auto --format=compact`

**Screen/Tmux**:
- Color support depends on configuration
- Test with: `--theme=auto --verbose`

### Accessibility Troubleshooting

**Screen Reader Issues**:
```bash
# Optimize for screen readers
mysql-schema-sync --config=config.yaml \
  --no-icons \
  --table-style=minimal \
  --theme=high-contrast \
  --verbose
```

**High Contrast Needs**:
```bash
# Use high contrast theme
mysql-schema-sync --config=config.yaml --theme=high-contrast
```

**Narrow Screen Issues**:
```bash
# Reduce table width
mysql-schema-sync --config=config.yaml --max-table-width=60
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Enhanced debug output with visual information
mysql-schema-sync --config=config.yaml \
  --verbose \
  --theme=high-contrast \
  --log-file=debug.log

# Test terminal capabilities
mysql-schema-sync --config=config.yaml --verbose --dry-run
```

### Environment Detection

The tool automatically detects terminal capabilities:

```bash
# Check what the tool detects about your terminal
mysql-schema-sync --config=config.yaml --verbose --dry-run 2>&1 | grep -i "terminal\|color\|unicode"
```

**Manual Override**:
```bash
# Force specific settings if auto-detection fails
export MYSQL_SCHEMA_SYNC_NO_COLOR=1
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_THEME=high-contrast
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