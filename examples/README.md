# Configuration Examples

This directory contains example configuration files for different use cases of MySQL Schema Sync.

## Available Examples

### basic-config.yaml
A minimal configuration for local development with two local MySQL databases.

**Use case**: Local development, testing, learning the tool

**Features**:
- Simple localhost connections
- Basic settings only
- No advanced features

### development-config.yaml
Configuration optimized for development workflow with safety features.

**Use case**: Development environment, staging to local sync

**Features**:
- Dry-run enabled by default
- Verbose logging
- Manual approval required
- Local log file

### production-config.yaml
Production-ready configuration with comprehensive settings and security considerations.

**Use case**: Production deployments, critical environments

**Features**:
- Extended timeouts
- Centralized logging
- Security best practices
- Environment variable usage

### ci-cd-config.yaml
Configuration designed for automated CI/CD pipelines.

**Use case**: Automated deployments, continuous integration

**Features**:
- Auto-approval enabled
- Detailed logging for audit
- Environment variable integration
- Pipeline-friendly settings

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