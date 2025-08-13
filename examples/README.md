# Configuration Examples

This directory contains example configuration files and usage documentation for different use cases of MySQL Schema Sync, including the new visual enhancement features.

## Available Examples

### basic-config.yaml
A minimal configuration for local development with two local MySQL databases.

**Use case**: Local development, testing, learning the tool

**Features**:
- Simple localhost connections
- Basic settings with default visual enhancements
- Colorized output and progress indicators

### development-config.yaml
Configuration optimized for development workflow with enhanced visual features.

**Use case**: Development environment, staging to local sync

**Features**:
- Dry-run enabled by default
- Verbose logging with enhanced formatting
- Rounded table styling for modern terminals
- Interactive confirmations with visual indicators

### production-config.yaml
Production-ready configuration with comprehensive settings and visual options.

**Use case**: Production deployments, critical environments

**Features**:
- Auto-theme detection for different environments
- Structured table output for operations review
- Alternative configurations for CI/CD and accessibility
- Security best practices

### ci-cd-config.yaml
Configuration designed for automated CI/CD pipelines with automation-friendly output.

**Use case**: Automated deployments, continuous integration

**Features**:
- Compact output format for log parsing
- Disabled colors and interactive elements
- ASCII-only compatibility
- JSON/YAML output options for CI tools

### accessibility-config.yaml
Configuration optimized for users with accessibility needs.

**Use case**: Visual impairments, screen readers, accessibility requirements

**Features**:
- High contrast color theme
- ASCII icons for screen reader compatibility
- Minimal table styling
- Narrow table width for better navigation

## Documentation Files

### usage-examples.md
Comprehensive examples showing different ways to use mysql-schema-sync with visual enhancements.

**Contents**:
- Basic usage patterns
- Visual enhancement options
- Output format examples
- Accessibility configurations
- Automation and scripting examples
- Terminal compatibility guide

## Usage

1. Copy the appropriate example to your project:
   ```bash
   cp examples/development-config.yaml config.yaml
   ```

2. Edit the configuration with your database details:
   ```bash
   nano config.yaml
   ```

3. Run mysql-schema-sync with your configuration:
   ```bash
   mysql-schema-sync --config=config.yaml
   ```

## Security Best Practices

### Password Management
- Never store passwords in configuration files for production
- Use environment variables for sensitive data
- Consider using secret management systems

### File Permissions
Restrict access to configuration files:
```bash
chmod 600 config.yaml
```

### Database Users
- Create dedicated users for schema synchronization
- Grant minimal required privileges
- Use different users for source (read-only) and target databases

### Network Security
- Use SSL connections for remote databases
- Restrict network access with firewalls
- Consider VPN or private networks for production

## Environment Variables

All configuration options can be overridden with environment variables using the prefix `MYSQL_SCHEMA_SYNC_`:

```bash
# Database connections
export MYSQL_SCHEMA_SYNC_SOURCE_HOST=localhost
export MYSQL_SCHEMA_SYNC_SOURCE_PASSWORD=secret
export MYSQL_SCHEMA_SYNC_TARGET_HOST=localhost
export MYSQL_SCHEMA_SYNC_TARGET_PASSWORD=secret

# Operation settings
export MYSQL_SCHEMA_SYNC_DRY_RUN=true
export MYSQL_SCHEMA_SYNC_VERBOSE=true
export MYSQL_SCHEMA_SYNC_AUTO_APPROVE=false

# Visual enhancement settings
export MYSQL_SCHEMA_SYNC_THEME=light
export MYSQL_SCHEMA_SYNC_FORMAT=json
export MYSQL_SCHEMA_SYNC_NO_COLOR=1
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_NO_PROGRESS=1
export MYSQL_SCHEMA_SYNC_NO_INTERACTIVE=1
export MYSQL_SCHEMA_SYNC_TABLE_STYLE=rounded
export MYSQL_SCHEMA_SYNC_MAX_TABLE_WIDTH=140
```

## Customization

Feel free to modify these examples to fit your specific needs:

1. **Database Settings**: Adjust host, port, and connection parameters
2. **Operation Modes**: Enable/disable dry-run, verbose output, auto-approval
3. **Timeouts**: Adjust based on your database size and network conditions
4. **Logging**: Configure log files and verbosity levels

## Testing Configurations

Before using in production, test your configuration:

1. **Dry-run first**:
   ```bash
   mysql-schema-sync --config=config.yaml --dry-run
   ```

2. **Verbose output**:
   ```bash
   mysql-schema-sync --config=config.yaml --verbose
   ```

3. **Test connectivity**:
   ```bash
   mysql-schema-sync --config=config.yaml --dry-run --verbose
   ```

## Support

If you need help with configuration:
- Check the main [README.md](../README.md) for detailed documentation
- Review the [troubleshooting guide](../README.md#troubleshooting)
- Open an issue on GitHub for specific problems