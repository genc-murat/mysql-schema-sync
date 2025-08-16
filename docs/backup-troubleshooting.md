# Backup System Troubleshooting Guide

## Overview

This guide provides detailed troubleshooting steps for common issues with the MySQL Schema Sync backup and rollback system. Use this guide to diagnose and resolve problems quickly.

## Table of Contents

- [Diagnostic Tools](#diagnostic-tools)
- [Common Issues](#common-issues)
- [Error Messages](#error-messages)
- [Performance Issues](#performance-issues)
- [Storage Provider Issues](#storage-provider-issues)
- [Security and Permissions](#security-and-permissions)
- [Recovery Procedures](#recovery-procedures)
- [Preventive Measures](#preventive-measures)

## Diagnostic Tools

### Built-in Diagnostics

```bash
# System health check
mysql-schema-sync backup health-check

# Storage provider connectivity test
mysql-schema-sync backup test-connection

# Configuration validation
mysql-schema-sync backup validate-config

# Backup integrity check
mysql-schema-sync backup integrity-check --backup-id <backup-id>

# Storage usage analysis
mysql-schema-sync backup storage-usage --detailed
```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
# Enable debug for single command
mysql-schema-sync --debug backup create

# Enable debug globally
export MYSQL_SCHEMA_SYNC_DEBUG=true
export MYSQL_SCHEMA_SYNC_LOG_LEVEL=debug
```

### Log Analysis

```bash
# View recent backup operations
tail -f /var/log/mysql-schema-sync/backup.log

# Search for specific errors
grep -i "error\|fail\|exception" /var/log/mysql-schema-sync/backup.log

# Analyze backup performance
grep "duration\|time" /var/log/mysql-schema-sync/backup.log | tail -20

# Check storage operations
grep "storage" /var/log/mysql-schema-sync/backup.log
```

## Common Issues

### Issue 1: Backup Creation Fails

#### Symptoms
- Backup creation command returns error
- No backup files created
- Process terminates unexpectedly

#### Diagnostic Steps

1. **Check available disk space:**
   ```bash
   df -h /path/to/backup/directory
   ```

2. **Verify directory permissions:**
   ```bash
   ls -la /path/to/backup/directory
   stat /path/to/backup/directory
   ```

3. **Test database connectivity:**
   ```bash
   mysql-schema-sync test-connection --config config.yaml
   ```

4. **Check configuration:**
   ```bash
   mysql-schema-sync backup validate-config
   ```

#### Solutions

**Insufficient Disk Space:**
```bash
# Clean up old backups
mysql-schema-sync backup cleanup

# Move backups to different location
mysql-schema-sync backup move --destination /new/path

# Configure retention policy
# Edit config.yaml:
retention:
  max_backups: 5
  max_age: "72h"
```

**Permission Issues:**
```bash
# Fix directory permissions
sudo chown -R $(whoami):$(whoami) /path/to/backup/directory
chmod 755 /path/to/backup/directory

# For restrictive permissions:
chmod 700 /path/to/backup/directory
```

**Database Connection Issues:**
```bash
# Test connection manually
mysql -h hostname -u username -p database_name

# Check firewall rules
telnet hostname 3306

# Verify credentials in config
mysql-schema-sync test-connection --config config.yaml
```

### Issue 2: Backup Validation Fails

#### Symptoms
- Checksum mismatch errors
- Backup marked as invalid
- Corruption warnings

#### Diagnostic Steps

1. **Check backup file integrity:**
   ```bash
   mysql-schema-sync backup info --backup-id <backup-id>
   ```

2. **Verify storage provider health:**
   ```bash
   mysql-schema-sync backup storage-health
   ```

3. **Compare file sizes:**
   ```bash
   # For local storage
   ls -la /path/to/backup/directory/<backup-id>.*
   ```

#### Solutions

**Checksum Mismatch:**
```bash
# Recalculate checksums
mysql-schema-sync backup recalculate-checksum --backup-id <backup-id>

# Re-validate backup
mysql-schema-sync backup validate --backup-id <backup-id>

# If still failing, recreate backup
mysql-schema-sync backup create --description "Replacement for corrupted backup"
```

**File Corruption:**
```bash
# Check if backup can be partially recovered
mysql-schema-sync backup partial-recovery --backup-id <backup-id>

# Create new backup immediately
mysql-schema-sync backup create --description "Emergency backup"

# Remove corrupted backup
mysql-schema-sync backup delete --backup-id <backup-id>
```

### Issue 3: Rollback Fails

#### Symptoms
- Rollback process stops with errors
- Database left in inconsistent state
- Foreign key constraint violations

#### Diagnostic Steps

1. **Review rollback plan:**
   ```bash
   mysql-schema-sync rollback plan --backup-id <backup-id> --show-sql
   ```

2. **Check database state:**
   ```bash
   mysql-schema-sync schema compare --backup-id <backup-id>
   ```

3. **Verify backup integrity:**
   ```bash
   mysql-schema-sync backup validate --backup-id <backup-id>
   ```

#### Solutions

**Foreign Key Constraint Violations:**
```bash
# Disable foreign key checks during rollback
mysql-schema-sync rollback execute --backup-id <backup-id> --disable-fk-checks

# Or manually disable in MySQL:
mysql -e "SET FOREIGN_KEY_CHECKS = 0; SOURCE rollback.sql; SET FOREIGN_KEY_CHECKS = 1;"
```

**Partial Rollback Failure:**
```bash
# Get recovery information
mysql-schema-sync rollback recovery-info --backup-id <backup-id>

# Manual intervention may be required
mysql-schema-sync rollback manual-steps --backup-id <backup-id>

# Force rollback (use with extreme caution)
mysql-schema-sync rollback execute --backup-id <backup-id> --force
```

### Issue 4: Storage Provider Connection Issues

#### Symptoms
- Cannot connect to cloud storage
- Authentication failures
- Network timeouts

#### Diagnostic Steps

1. **Test connectivity:**
   ```bash
   mysql-schema-sync backup test-connection
   ```

2. **Check credentials:**
   ```bash
   # For AWS S3
   aws s3 ls s3://your-bucket-name

   # For Azure
   az storage blob list --container-name backups --account-name youraccount

   # For GCS
   gsutil ls gs://your-bucket-name
   ```

3. **Verify network connectivity:**
   ```bash
   # Test DNS resolution
   nslookup s3.amazonaws.com
   
   # Test connectivity
   telnet s3.amazonaws.com 443
   ```

#### Solutions

**AWS S3 Issues:**
```bash
# Check AWS credentials
aws configure list

# Test with different region
aws s3 ls s3://your-bucket-name --region us-west-2

# Verify IAM permissions
aws iam simulate-principal-policy --policy-source-arn arn:aws:iam::account:user/username --action-names s3:GetObject --resource-arns arn:aws:s3:::bucket/key
```

**Azure Blob Storage Issues:**
```bash
# Check storage account key
az storage account keys list --account-name youraccount

# Test connection
az storage blob list --container-name backups --account-name youraccount --account-key yourkey
```

**Google Cloud Storage Issues:**
```bash
# Check service account permissions
gcloud projects get-iam-policy your-project-id

# Test with service account
gcloud auth activate-service-account --key-file=/path/to/service-account.json
gsutil ls gs://your-bucket-name
```

## Error Messages

### "Permission denied" Errors

**Error Message:**
```
Error: failed to create backup: permission denied: /path/to/backup/directory
```

**Cause:** Insufficient file system permissions

**Solution:**
```bash
# Check current permissions
ls -la /path/to/backup/directory

# Fix permissions
sudo chown -R $(whoami):$(whoami) /path/to/backup/directory
chmod 755 /path/to/backup/directory

# For more restrictive permissions
chmod 700 /path/to/backup/directory
```

### "No space left on device" Errors

**Error Message:**
```
Error: failed to write backup file: no space left on device
```

**Cause:** Insufficient disk space

**Solution:**
```bash
# Check disk usage
df -h /path/to/backup/directory

# Clean up old backups
mysql-schema-sync backup cleanup

# Move to different location with more space
mysql-schema-sync backup move --destination /new/path

# Configure retention policy to use less space
# Edit config.yaml:
retention:
  max_backups: 3
  max_age: "24h"
```

### "Connection refused" Errors

**Error Message:**
```
Error: failed to connect to database: connection refused
```

**Cause:** Database server not accessible

**Solution:**
```bash
# Check if database is running
systemctl status mysql

# Test connection manually
mysql -h hostname -u username -p

# Check firewall rules
sudo ufw status
sudo iptables -L

# Verify database configuration
mysql-schema-sync test-connection --config config.yaml
```

### "Access denied" Errors

**Error Message:**
```
Error: access denied for user 'username'@'hostname'
```

**Cause:** Insufficient database privileges

**Solution:**
```sql
-- Grant necessary privileges
GRANT SELECT, SHOW DATABASES, SHOW VIEW, LOCK TABLES ON *.* TO 'username'@'hostname';
GRANT PROCESS ON *.* TO 'username'@'hostname';
FLUSH PRIVILEGES;
```

### "Checksum mismatch" Errors

**Error Message:**
```
Error: backup validation failed: checksum mismatch (expected: abc123, got: def456)
```

**Cause:** Backup file corruption

**Solution:**
```bash
# Try to recalculate checksum
mysql-schema-sync backup recalculate-checksum --backup-id <backup-id>

# If still failing, backup may be corrupted
mysql-schema-sync backup info --backup-id <backup-id>

# Create new backup
mysql-schema-sync backup create --description "Replacement backup"

# Remove corrupted backup
mysql-schema-sync backup delete --backup-id <backup-id>
```

## Performance Issues

### Slow Backup Creation

#### Symptoms
- Backup creation takes excessive time
- High CPU or memory usage during backup
- System becomes unresponsive

#### Diagnostic Steps

1. **Monitor system resources:**
   ```bash
   top -p $(pgrep mysql-schema-sync)
   iostat -x 1
   ```

2. **Check database performance:**
   ```sql
   SHOW PROCESSLIST;
   SHOW ENGINE INNODB STATUS;
   ```

3. **Analyze backup size:**
   ```bash
   mysql-schema-sync backup stats --backup-id <backup-id>
   ```

#### Solutions

**Large Database Optimization:**
```yaml
# Optimize compression settings
compression:
  enabled: true
  algorithm: "lz4"  # Faster than gzip
  level: 1          # Lower compression, faster speed
  threshold: 10240  # Only compress larger files
```

**Parallel Processing:**
```yaml
# Enable parallel operations (if supported)
backup:
  parallel_operations: true
  max_workers: 4
```

**Memory Optimization:**
```yaml
# Reduce memory usage
backup:
  buffer_size: "64MB"  # Smaller buffer for limited memory
  streaming: true      # Stream data instead of loading in memory
```

### Slow Rollback Operations

#### Symptoms
- Rollback takes very long time
- Database locks during rollback
- Timeout errors

#### Solutions

**Optimize Rollback Process:**
```bash
# Use parallel rollback (if available)
mysql-schema-sync rollback execute --backup-id <backup-id> --parallel

# Disable foreign key checks for faster rollback
mysql-schema-sync rollback execute --backup-id <backup-id> --disable-fk-checks

# Use larger transaction batches
mysql-schema-sync rollback execute --backup-id <backup-id> --batch-size 1000
```

## Storage Provider Issues

### AWS S3 Specific Issues

**Issue: Slow upload/download speeds**

```yaml
# Optimize S3 configuration
storage:
  s3:
    multipart_threshold: "64MB"
    multipart_chunksize: "16MB"
    max_concurrency: 10
    use_accelerate_endpoint: true
```

**Issue: Access denied errors**

```bash
# Check bucket policy
aws s3api get-bucket-policy --bucket your-bucket-name

# Verify IAM user permissions
aws iam get-user-policy --user-name your-user --policy-name your-policy
```

### Azure Blob Storage Issues

**Issue: Connection timeouts**

```yaml
# Increase timeout settings
storage:
  azure:
    connection_timeout: 300
    read_timeout: 300
    retry_attempts: 5
```

### Google Cloud Storage Issues

**Issue: Service account authentication**

```bash
# Verify service account key
gcloud auth activate-service-account --key-file=/path/to/key.json

# Check project permissions
gcloud projects get-iam-policy your-project-id
```

## Security and Permissions

### Encryption Issues

**Issue: Encryption key not found**

```bash
# Check environment variable
echo $BACKUP_ENCRYPTION_KEY

# Verify key file exists and is readable
ls -la /path/to/encryption/key
cat /path/to/encryption/key | wc -c  # Should be 32 bytes for AES-256
```

**Issue: Decryption fails**

```bash
# Verify key is correct
mysql-schema-sync backup test-encryption --backup-id <backup-id>

# Check if backup was encrypted with different key
mysql-schema-sync backup info --backup-id <backup-id>
```

### File Permission Issues

**Issue: Cannot read backup files**

```bash
# Check file ownership and permissions
ls -la /path/to/backup/files

# Fix ownership
sudo chown -R backup-user:backup-group /path/to/backup/files

# Set appropriate permissions
chmod 600 /path/to/backup/files/*  # Restrictive
# or
chmod 644 /path/to/backup/files/*  # More permissive
```

## Recovery Procedures

### Emergency Backup Recovery

When backups are corrupted or inaccessible:

1. **Assess the situation:**
   ```bash
   mysql-schema-sync backup list --status all
   mysql-schema-sync backup validate-all
   ```

2. **Identify recoverable backups:**
   ```bash
   mysql-schema-sync backup recovery-scan
   ```

3. **Attempt partial recovery:**
   ```bash
   mysql-schema-sync backup partial-recovery --backup-id <backup-id>
   ```

4. **Create emergency backup:**
   ```bash
   mysql-schema-sync backup create --description "Emergency backup" --force
   ```

### Database Recovery from Backup

When database is corrupted and needs full recovery:

1. **Stop application services:**
   ```bash
   sudo systemctl stop your-application
   ```

2. **Create database dump for safety:**
   ```bash
   mysqldump --all-databases > emergency-dump.sql
   ```

3. **Identify best backup for recovery:**
   ```bash
   mysql-schema-sync backup list --database your-db --status completed
   ```

4. **Execute recovery:**
   ```bash
   mysql-schema-sync rollback execute --backup-id <backup-id> --force
   ```

5. **Validate recovery:**
   ```bash
   mysql-schema-sync rollback validate --backup-id <backup-id>
   ```

6. **Restart services:**
   ```bash
   sudo systemctl start your-application
   ```

## Preventive Measures

### Regular Health Checks

Set up automated health monitoring:

```bash
#!/bin/bash
# backup-health-check.sh

# Daily health check
mysql-schema-sync backup health-check

# Weekly integrity check
if [ $(date +%u) -eq 1 ]; then  # Monday
    mysql-schema-sync backup validate-all
fi

# Monthly storage optimization
if [ $(date +%d) -eq 1 ]; then  # First day of month
    mysql-schema-sync backup storage-optimize
fi
```

### Monitoring Setup

```bash
# Add to crontab
0 6 * * * /path/to/backup-health-check.sh >> /var/log/backup-health.log 2>&1

# Set up alerts for failures
0 7 * * * grep -q "ERROR\|FAIL" /var/log/backup-health.log && echo "Backup health check failed" | mail -s "Backup Alert" admin@company.com
```

### Configuration Validation

Regularly validate your backup configuration:

```bash
# Weekly configuration check
mysql-schema-sync backup validate-config

# Test backup and rollback process in staging
mysql-schema-sync backup create --description "Test backup"
BACKUP_ID=$(mysql-schema-sync backup list --limit 1 --format json | jq -r '.[0].id')
mysql-schema-sync rollback plan --backup-id $BACKUP_ID
```

### Documentation Maintenance

Keep troubleshooting documentation updated:

1. Document any new issues and solutions
2. Update configuration examples
3. Review and test recovery procedures
4. Train team members on troubleshooting steps

## Getting Additional Help

### Log Collection for Support

When reporting issues, collect relevant logs:

```bash
# Create support bundle
mkdir support-bundle
cp /var/log/mysql-schema-sync/*.log support-bundle/
mysql-schema-sync backup health-check > support-bundle/health-check.txt
mysql-schema-sync backup validate-config > support-bundle/config-validation.txt
tar -czf support-bundle.tar.gz support-bundle/
```

### Useful Information to Include

When seeking help, provide:

1. **Error messages** (exact text)
2. **Configuration files** (sanitized)
3. **System information** (OS, version, hardware)
4. **Log files** (relevant sections)
5. **Steps to reproduce** the issue
6. **Expected vs actual behavior**

### Community Resources

- Project documentation and wiki
- Community forums and discussions
- Issue tracker for bug reports
- Stack Overflow for general questions

---

*This troubleshooting guide is regularly updated based on common issues and user feedback. For the latest information, check the project documentation.*