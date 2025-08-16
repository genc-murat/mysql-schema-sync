# Backup and Rollback System Guide

## Overview

The MySQL Schema Sync backup and rollback system provides comprehensive data protection and recovery capabilities for your database schema migrations. This system automatically creates backups before applying schema changes and enables you to rollback to previous states if needed.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Backup Operations](#backup-operations)
- [Rollback Operations](#rollback-operations)
- [Storage Providers](#storage-providers)
- [Security Features](#security-features)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Quick Start

### Basic Setup

1. **Enable backup system in your configuration:**

```yaml
backup:
  enabled: true
  storage:
    provider: local
    local:
      base_path: "./backups"
      permissions: "0755"
  retention:
    max_backups: 10
    max_age: "168h" # 7 days
```

2. **Run a migration with automatic backup:**

```bash
mysql-schema-sync migrate --config config.yaml --auto-backup
```

3. **List available backups:**

```bash
mysql-schema-sync backup list
```

4. **Rollback to a previous state:**

```bash
mysql-schema-sync rollback --backup-id backup_20240816_143022_a1b2c3d4
```

## Configuration

### Complete Configuration Example

```yaml
backup:
  enabled: true
  
  # Storage configuration
  storage:
    provider: local  # Options: local, s3, azure, gcs
    local:
      base_path: "./backups"
      permissions: "0755"
    # s3:
    #   bucket: "my-backup-bucket"
    #   region: "us-west-2"
    #   access_key: "${AWS_ACCESS_KEY_ID}"
    #   secret_key: "${AWS_SECRET_ACCESS_KEY}"
  
  # Retention policy
  retention:
    max_backups: 10
    max_age: "168h"        # 7 days
    cleanup_interval: "1h"
    keep_daily: 7
    keep_weekly: 4
    keep_monthly: 12
  
  # Compression settings
  compression:
    enabled: true
    algorithm: "gzip"      # Options: gzip, lz4, zstd
    level: 6
    threshold: 1024        # Compress files larger than 1KB
  
  # Encryption settings
  encryption:
    enabled: true
    algorithm: "aes256"
    key_source: "env"      # Options: env, file, direct
    key_env_var: "BACKUP_ENCRYPTION_KEY"
    # key_file: "/path/to/key/file"
    rotation_enabled: true
    rotation_days: 90
  
  # Validation settings
  validation:
    enabled: true
    checksum_algorithm: "sha256"
    validate_on_create: true
    validate_on_restore: true
    dry_run_validation: true
    validation_timeout: "5m"
```

### Configuration Options

#### Storage Providers

**Local Storage:**
```yaml
storage:
  provider: local
  local:
    base_path: "./backups"
    permissions: "0755"
```

**Amazon S3:**
```yaml
storage:
  provider: s3
  s3:
    bucket: "my-backup-bucket"
    region: "us-west-2"
    access_key: "${AWS_ACCESS_KEY_ID}"
    secret_key: "${AWS_SECRET_ACCESS_KEY}"
    prefix: "mysql-backups/"
```

**Azure Blob Storage:**
```yaml
storage:
  provider: azure
  azure:
    account_name: "mystorageaccount"
    account_key: "${AZURE_STORAGE_KEY}"
    container_name: "backups"
```

**Google Cloud Storage:**
```yaml
storage:
  provider: gcs
  gcs:
    bucket: "my-backup-bucket"
    credentials_path: "/path/to/service-account.json"
    project_id: "my-project-id"
```

## Backup Operations

### Automatic Backups

Automatic backups are created before each migration:

```bash
# Enable automatic backup for this migration
mysql-schema-sync migrate --config config.yaml --auto-backup

# Disable automatic backup (not recommended)
mysql-schema-sync migrate --config config.yaml --no-backup
```

### Manual Backups

Create backups manually for specific scenarios:

```bash
# Create a manual backup with description
mysql-schema-sync backup create --description "Before major refactoring"

# Create backup with custom tags
mysql-schema-sync backup create --tag environment=production --tag type=manual

# Create backup for specific database
mysql-schema-sync backup create --database my_database
```

### Listing Backups

```bash
# List all backups
mysql-schema-sync backup list

# List backups for specific database
mysql-schema-sync backup list --database my_database

# List backups with filters
mysql-schema-sync backup list --status completed --created-after 2024-08-01

# List backups with specific tags
mysql-schema-sync backup list --tag environment=production
```

### Backup Information

```bash
# Show detailed backup information
mysql-schema-sync backup info --backup-id backup_20240816_143022_a1b2c3d4

# Validate backup integrity
mysql-schema-sync backup validate --backup-id backup_20240816_143022_a1b2c3d4

# Show backup size and compression stats
mysql-schema-sync backup stats --backup-id backup_20240816_143022_a1b2c3d4
```

### Backup Management

```bash
# Export backup to file
mysql-schema-sync backup export --backup-id backup_20240816_143022_a1b2c3d4 --output backup.tar.gz

# Import backup from file
mysql-schema-sync backup import --file backup.tar.gz

# Delete specific backup
mysql-schema-sync backup delete --backup-id backup_20240816_143022_a1b2c3d4

# Clean up old backups according to retention policy
mysql-schema-sync backup cleanup
```

## Rollback Operations

### Planning a Rollback

Before executing a rollback, plan and review the changes:

```bash
# List available rollback points
mysql-schema-sync rollback list --database my_database

# Plan rollback to specific backup
mysql-schema-sync rollback plan --backup-id backup_20240816_143022_a1b2c3d4

# Show detailed rollback plan with SQL statements
mysql-schema-sync rollback plan --backup-id backup_20240816_143022_a1b2c3d4 --show-sql
```

### Executing a Rollback

```bash
# Execute rollback with confirmation
mysql-schema-sync rollback execute --backup-id backup_20240816_143022_a1b2c3d4

# Execute rollback without confirmation (use with caution)
mysql-schema-sync rollback execute --backup-id backup_20240816_143022_a1b2c3d4 --force

# Dry run rollback (show what would be done)
mysql-schema-sync rollback execute --backup-id backup_20240816_143022_a1b2c3d4 --dry-run
```

### Rollback Validation

```bash
# Validate rollback was successful
mysql-schema-sync rollback validate --backup-id backup_20240816_143022_a1b2c3d4

# Compare current schema with backup
mysql-schema-sync rollback compare --backup-id backup_20240816_143022_a1b2c3d4
```

## Storage Providers

### Local Storage

Best for development and small deployments:

**Pros:**
- Simple setup
- No external dependencies
- Fast access

**Cons:**
- Limited scalability
- No built-in redundancy
- Single point of failure

**Configuration:**
```yaml
storage:
  provider: local
  local:
    base_path: "/var/backups/mysql-schema-sync"
    permissions: "0700"  # Restrictive permissions for security
```

### Amazon S3

Recommended for production environments:

**Pros:**
- Highly scalable
- Built-in redundancy
- Cost-effective for long-term storage
- Versioning support

**Cons:**
- Requires AWS credentials
- Network dependency
- Potential egress costs

**Setup:**
1. Create S3 bucket
2. Configure IAM user with appropriate permissions
3. Set environment variables:
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

**Required S3 Permissions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::your-backup-bucket",
        "arn:aws:s3:::your-backup-bucket/*"
      ]
    }
  ]
}
```

### Azure Blob Storage

Good for Azure-based infrastructure:

**Setup:**
1. Create storage account
2. Create container
3. Get access key from Azure portal
4. Set environment variable:
   ```bash
   export AZURE_STORAGE_KEY=your_storage_key
   ```

### Google Cloud Storage

Ideal for GCP environments:

**Setup:**
1. Create GCS bucket
2. Create service account with Storage Admin role
3. Download service account key file
4. Configure path in settings

## Security Features

### Encryption

#### Encryption at Rest

All backups can be encrypted using AES-256:

```yaml
encryption:
  enabled: true
  algorithm: "aes256"
  key_source: "env"
  key_env_var: "BACKUP_ENCRYPTION_KEY"
```

#### Key Management

**Environment Variable:**
```bash
export BACKUP_ENCRYPTION_KEY=$(openssl rand -hex 32)
```

**Key File:**
```bash
# Generate key file
openssl rand 32 > /secure/path/backup.key
chmod 600 /secure/path/backup.key
```

```yaml
encryption:
  key_source: "file"
  key_file: "/secure/path/backup.key"
```

**Key Rotation:**
```yaml
encryption:
  rotation_enabled: true
  rotation_days: 90
```

### Access Control

#### File Permissions

Set restrictive permissions for local storage:

```yaml
storage:
  local:
    permissions: "0600"  # Owner read/write only
```

#### Cloud Storage Security

**S3 Bucket Policy Example:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyInsecureConnections",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": [
        "arn:aws:s3:::your-backup-bucket",
        "arn:aws:s3:::your-backup-bucket/*"
      ],
      "Condition": {
        "Bool": {
          "aws:SecureTransport": "false"
        }
      }
    }
  ]
}
```

### Audit Logging

Enable audit logging for compliance:

```yaml
logging:
  enable_audit_log: true
  audit_log_path: "/var/log/mysql-schema-sync/audit.log"
  log_level: "INFO"
```

## Monitoring and Maintenance

### Storage Monitoring

```bash
# Check storage usage
mysql-schema-sync backup storage-usage

# Monitor storage health
mysql-schema-sync backup storage-health

# Get storage optimization recommendations
mysql-schema-sync backup storage-optimize
```

### Backup Health Checks

```bash
# Validate all backups
mysql-schema-sync backup validate-all

# Check backup integrity
mysql-schema-sync backup integrity-check

# Generate backup report
mysql-schema-sync backup report --format json --output backup-report.json
```

### Automated Maintenance

Set up automated maintenance tasks:

```bash
# Add to crontab for daily cleanup
0 2 * * * /usr/local/bin/mysql-schema-sync backup cleanup

# Weekly integrity check
0 3 * * 0 /usr/local/bin/mysql-schema-sync backup validate-all

# Monthly storage optimization
0 4 1 * * /usr/local/bin/mysql-schema-sync backup storage-optimize
```

## Troubleshooting

### Common Issues

#### Backup Creation Fails

**Problem:** Backup creation fails with permission error
```
Error: failed to create backup: permission denied
```

**Solution:**
1. Check directory permissions:
   ```bash
   ls -la /path/to/backup/directory
   ```
2. Ensure the user has write permissions:
   ```bash
   chmod 755 /path/to/backup/directory
   ```
3. Check disk space:
   ```bash
   df -h /path/to/backup/directory
   ```

#### Backup Validation Fails

**Problem:** Backup validation fails with checksum mismatch
```
Error: backup validation failed: checksum mismatch
```

**Solution:**
1. Check if backup file is corrupted:
   ```bash
   mysql-schema-sync backup info --backup-id <backup-id>
   ```
2. Try re-creating the backup:
   ```bash
   mysql-schema-sync backup create --description "Replacement backup"
   ```
3. Check storage provider health:
   ```bash
   mysql-schema-sync backup storage-health
   ```

#### Rollback Fails

**Problem:** Rollback fails with foreign key constraint error
```
Error: rollback failed: foreign key constraint violation
```

**Solution:**
1. Review rollback plan:
   ```bash
   mysql-schema-sync rollback plan --backup-id <backup-id> --show-sql
   ```
2. Disable foreign key checks temporarily:
   ```sql
   SET FOREIGN_KEY_CHECKS = 0;
   -- Rollback operations
   SET FOREIGN_KEY_CHECKS = 1;
   ```
3. Use force rollback (with caution):
   ```bash
   mysql-schema-sync rollback execute --backup-id <backup-id> --force
   ```

#### Storage Provider Issues

**Problem:** S3 connection fails
```
Error: failed to connect to S3: access denied
```

**Solution:**
1. Verify AWS credentials:
   ```bash
   aws s3 ls s3://your-backup-bucket
   ```
2. Check IAM permissions
3. Verify bucket exists and is accessible
4. Test with different region if needed

### Debug Mode

Enable debug logging for troubleshooting:

```bash
mysql-schema-sync --debug backup create
```

Or set environment variable:
```bash
export MYSQL_SCHEMA_SYNC_DEBUG=true
mysql-schema-sync backup create
```

### Log Analysis

Check logs for detailed error information:

```bash
# View recent backup operations
tail -f /var/log/mysql-schema-sync/backup.log

# Search for specific errors
grep "ERROR" /var/log/mysql-schema-sync/backup.log

# Analyze backup performance
grep "duration" /var/log/mysql-schema-sync/backup.log
```

## Best Practices

### Backup Strategy

1. **Enable automatic backups** for all production migrations
2. **Create manual backups** before major schema changes
3. **Test rollback procedures** regularly in non-production environments
4. **Monitor backup health** and storage usage
5. **Implement retention policies** to manage storage costs

### Security Best Practices

1. **Enable encryption** for all backups containing sensitive data
2. **Use strong encryption keys** and rotate them regularly
3. **Restrict file permissions** for local storage
4. **Use IAM roles** instead of access keys for cloud storage
5. **Enable audit logging** for compliance requirements
6. **Regularly validate** backup integrity

### Performance Optimization

1. **Use appropriate compression** algorithms for your data
2. **Configure compression thresholds** to avoid compressing small files
3. **Monitor backup creation time** and optimize as needed
4. **Use cloud storage** for better scalability
5. **Implement parallel operations** for large schemas

### Operational Best Practices

1. **Document your backup procedures** and rollback plans
2. **Train team members** on backup and rollback operations
3. **Set up monitoring alerts** for backup failures
4. **Regularly test disaster recovery** procedures
5. **Keep backup configurations** in version control

## Examples

### Example 1: Development Environment Setup

```yaml
# config-dev.yaml
backup:
  enabled: true
  storage:
    provider: local
    local:
      base_path: "./dev-backups"
      permissions: "0755"
  retention:
    max_backups: 5
    max_age: "24h"
  compression:
    enabled: true
    algorithm: "lz4"  # Fast compression for development
    level: 1
  encryption:
    enabled: false  # Disabled for development
```

### Example 2: Production Environment Setup

```yaml
# config-prod.yaml
backup:
  enabled: true
  storage:
    provider: s3
    s3:
      bucket: "prod-mysql-backups"
      region: "us-west-2"
      prefix: "schema-sync/"
  retention:
    max_backups: 50
    max_age: "2160h"  # 90 days
    keep_daily: 30
    keep_weekly: 12
    keep_monthly: 24
  compression:
    enabled: true
    algorithm: "zstd"  # Best compression for production
    level: 6
  encryption:
    enabled: true
    algorithm: "aes256"
    key_source: "env"
    key_env_var: "BACKUP_ENCRYPTION_KEY"
    rotation_enabled: true
    rotation_days: 90
  validation:
    enabled: true
    validate_on_create: true
    validate_on_restore: true
```

### Example 3: CI/CD Pipeline Integration

```bash
#!/bin/bash
# deploy-schema.sh

set -e

# Create pre-deployment backup
echo "Creating pre-deployment backup..."
BACKUP_ID=$(mysql-schema-sync backup create --description "Pre-deployment backup for release $RELEASE_VERSION" --tag release=$RELEASE_VERSION --output-format json | jq -r '.backup_id')

echo "Backup created: $BACKUP_ID"

# Run migration
echo "Running schema migration..."
if mysql-schema-sync migrate --config config-prod.yaml; then
    echo "Migration successful"
    
    # Validate post-migration state
    mysql-schema-sync backup validate --backup-id $BACKUP_ID
    
    echo "Deployment completed successfully"
else
    echo "Migration failed, rolling back..."
    
    # Rollback to pre-deployment state
    mysql-schema-sync rollback execute --backup-id $BACKUP_ID --force
    
    echo "Rollback completed"
    exit 1
fi
```

### Example 4: Disaster Recovery Script

```bash
#!/bin/bash
# disaster-recovery.sh

DATABASE_NAME="$1"
BACKUP_DATE="$2"

if [ -z "$DATABASE_NAME" ] || [ -z "$BACKUP_DATE" ]; then
    echo "Usage: $0 <database_name> <backup_date>"
    echo "Example: $0 production_db 2024-08-15"
    exit 1
fi

echo "Starting disaster recovery for $DATABASE_NAME..."

# Find backup from specific date
BACKUP_ID=$(mysql-schema-sync backup list --database $DATABASE_NAME --created-after $BACKUP_DATE --created-before $(date -d "$BACKUP_DATE + 1 day" +%Y-%m-%d) --format json | jq -r '.[0].id')

if [ "$BACKUP_ID" = "null" ] || [ -z "$BACKUP_ID" ]; then
    echo "No backup found for $DATABASE_NAME on $BACKUP_DATE"
    exit 1
fi

echo "Found backup: $BACKUP_ID"

# Validate backup before rollback
echo "Validating backup integrity..."
if ! mysql-schema-sync backup validate --backup-id $BACKUP_ID; then
    echo "Backup validation failed"
    exit 1
fi

# Show rollback plan
echo "Rollback plan:"
mysql-schema-sync rollback plan --backup-id $BACKUP_ID

# Confirm rollback
read -p "Proceed with rollback? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "Rollback cancelled"
    exit 0
fi

# Execute rollback
echo "Executing rollback..."
mysql-schema-sync rollback execute --backup-id $BACKUP_ID

# Validate rollback
echo "Validating rollback..."
mysql-schema-sync rollback validate --backup-id $BACKUP_ID

echo "Disaster recovery completed successfully"
```

### Example 5: Backup Monitoring Script

```bash
#!/bin/bash
# backup-monitor.sh

# Check backup health
echo "Checking backup system health..."

# Get storage usage
STORAGE_USAGE=$(mysql-schema-sync backup storage-usage --format json)
TOTAL_SIZE=$(echo $STORAGE_USAGE | jq -r '.total_size_gb')
BACKUP_COUNT=$(echo $STORAGE_USAGE | jq -r '.backup_count')

echo "Total backups: $BACKUP_COUNT"
echo "Total storage used: ${TOTAL_SIZE}GB"

# Check for failed backups in last 24 hours
FAILED_BACKUPS=$(mysql-schema-sync backup list --status failed --created-after $(date -d "1 day ago" +%Y-%m-%d) --format json | jq length)

if [ "$FAILED_BACKUPS" -gt 0 ]; then
    echo "WARNING: $FAILED_BACKUPS failed backups in the last 24 hours"
    # Send alert (example with curl to webhook)
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"⚠️ $FAILED_BACKUPS failed backups detected\"}" \
        $SLACK_WEBHOOK_URL
fi

# Check storage quota
if (( $(echo "$TOTAL_SIZE > 100" | bc -l) )); then
    echo "WARNING: Storage usage exceeds 100GB"
fi

# Validate random backup
RANDOM_BACKUP=$(mysql-schema-sync backup list --limit 1 --format json | jq -r '.[0].id')
if [ "$RANDOM_BACKUP" != "null" ]; then
    echo "Validating random backup: $RANDOM_BACKUP"
    mysql-schema-sync backup validate --backup-id $RANDOM_BACKUP
fi

echo "Backup monitoring completed"
```

## Support and Resources

### Getting Help

- **Documentation**: Check this guide and the main project documentation
- **Logs**: Enable debug logging for detailed troubleshooting information
- **Community**: Join the project community for support and discussions
- **Issues**: Report bugs and feature requests on the project repository

### Additional Resources

- [Configuration Reference](config-reference.md)
- [API Documentation](api-docs.md)
- [Security Guide](security-guide.md)
- [Performance Tuning](performance-guide.md)
- [Migration Best Practices](migration-best-practices.md)

---

*This guide covers the essential aspects of the MySQL Schema Sync backup and rollback system. For the most up-to-date information, please refer to the official documentation and release notes.*