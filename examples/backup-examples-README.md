# Backup System Configuration Examples

This directory contains example configurations for the MySQL Schema Sync backup and rollback system, tailored for different deployment scenarios and environments.

## Available Examples

### 1. Development Environment (`backup-config-development.yaml`)

**Use Case:** Local development, testing, and experimentation

**Key Features:**
- Local file storage for simplicity
- Fast LZ4 compression for quick backups
- Minimal retention (24 hours, 5 backups max)
- No encryption (for development speed)
- Basic validation
- Verbose logging for debugging

**Best For:**
- Individual developer workstations
- Local testing environments
- Rapid prototyping
- Learning and experimentation

**Setup:**
```bash
# Copy the configuration
cp examples/backup-config-development.yaml config-dev.yaml

# Create backup directory
mkdir -p ./dev-backups

# Test the configuration
mysql-schema-sync backup validate-config --config config-dev.yaml
```

### 2. Production Environment (`backup-config-production.yaml`)

**Use Case:** Production deployments with high security and compliance requirements

**Key Features:**
- AWS S3 storage with optimization settings
- Strong AES-256 encryption with key rotation
- Comprehensive retention policy (90 days, tiered retention)
- ZSTD compression for optimal storage efficiency
- Full validation and integrity checking
- Audit logging and compliance features
- Monitoring and alerting integration

**Best For:**
- Production databases
- Regulated industries (healthcare, finance)
- Enterprise environments
- High-security requirements

**Setup:**
```bash
# Set required environment variables
export BACKUP_S3_BUCKET="your-production-backup-bucket"
export AWS_REGION="us-west-2"
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export BACKUP_ENCRYPTION_KEY=$(openssl rand -hex 32)
export DB_HOST="your-db-host"
export DB_USERNAME="your-db-user"
export DB_PASSWORD="your-db-password"
export DB_NAME="your-database"

# Copy and customize the configuration
cp examples/backup-config-production.yaml config-prod.yaml

# Validate the configuration
mysql-schema-sync backup validate-config --config config-prod.yaml

# Test backup creation
mysql-schema-sync backup create --config config-prod.yaml --description "Production test backup"
```

### 3. CI/CD Pipeline (`backup-config-ci-cd.yaml`)

**Use Case:** Automated deployment pipelines and continuous integration

**Key Features:**
- Optimized for automation and speed
- Pipeline-specific tagging and metadata
- Automatic rollback on failure
- JSON output for machine parsing
- Integration with CI/CD platforms (GitLab, GitHub)
- Structured logging and metrics
- Notification integration (Slack, email)

**Best For:**
- GitLab CI/CD pipelines
- GitHub Actions workflows
- Jenkins deployments
- Automated testing environments
- DevOps workflows

**Setup:**
```bash
# Set CI/CD environment variables (example for GitLab CI)
export CI_BACKUP_BUCKET="ci-backup-bucket"
export CI_BACKUP_ENCRYPTION_KEY=$(openssl rand -hex 32)
export CI_DB_HOST="staging-db-host"
export CI_DB_USERNAME="ci-user"
export CI_DB_PASSWORD="ci-password"
export CI_DB_NAME="staging_db"

# Copy the configuration
cp examples/backup-config-ci-cd.yaml .gitlab-ci-backup.yaml

# Add to your .gitlab-ci.yml
cat >> .gitlab-ci.yml << 'EOF'
backup_and_migrate:
  stage: deploy
  script:
    - mysql-schema-sync backup create --config .gitlab-ci-backup.yaml
    - mysql-schema-sync migrate --config .gitlab-ci-backup.yaml --auto-backup
  artifacts:
    reports:
      junit: backup-report.xml
    paths:
      - backup-logs/
EOF
```

### 4. High Availability (`backup-config-high-availability.yaml`)

**Use Case:** Mission-critical systems requiring maximum uptime and redundancy

**Key Features:**
- Multi-provider storage redundancy (S3, Azure, GCS, Local)
- Tiered storage strategy (hot, warm, cold)
- Advanced replication and failover
- Circuit breaker patterns
- Comprehensive monitoring and alerting
- Disaster recovery capabilities
- Enterprise security features

**Best For:**
- Mission-critical production systems
- 24/7 operations
- Financial services
- Healthcare systems
- Large enterprise deployments

**Setup:**
```bash
# Set up multiple storage providers
export PRIMARY_BACKUP_BUCKET="ha-primary-backups"
export PRIMARY_AWS_REGION="us-west-2"
export PRIMARY_AWS_ACCESS_KEY_ID="primary-access-key"
export PRIMARY_AWS_SECRET_ACCESS_KEY="primary-secret-key"

export AZURE_STORAGE_ACCOUNT="hasecondarybackups"
export AZURE_STORAGE_KEY="azure-storage-key"

export GCS_BACKUP_BUCKET="ha-tertiary-backups"
export GCS_CREDENTIALS_PATH="/path/to/gcs-credentials.json"
export GCS_PROJECT_ID="your-gcp-project"

# Database configuration
export PRIMARY_DB_HOST="primary-db-host"
export REPLICA_DB_HOST="replica-db-host"
export DB_NAME="production_db"

# Copy and customize
cp examples/backup-config-high-availability.yaml config-ha.yaml

# Validate all storage providers
mysql-schema-sync backup test-connection --config config-ha.yaml --all-providers

# Set up monitoring
mysql-schema-sync backup setup-monitoring --config config-ha.yaml
```

## Configuration Customization

### Environment Variables

All configurations support environment variable substitution using the `${VARIABLE_NAME}` syntax. Common variables include:

**Database Connection:**
- `DB_HOST` - Database hostname
- `DB_PORT` - Database port (default: 3306)
- `DB_USERNAME` - Database username
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name

**Storage Configuration:**
- `BACKUP_S3_BUCKET` - S3 bucket name
- `AWS_REGION` - AWS region
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AZURE_STORAGE_ACCOUNT` - Azure storage account
- `AZURE_STORAGE_KEY` - Azure storage key
- `GCS_BACKUP_BUCKET` - Google Cloud Storage bucket

**Security:**
- `BACKUP_ENCRYPTION_KEY` - Encryption key (32 bytes hex)
- `BACKUP_ENCRYPTION_KEY_FILE` - Path to encryption key file

**Notifications:**
- `SLACK_WEBHOOK_URL` - Slack webhook URL
- `ALERT_WEBHOOK_URL` - Generic alert webhook URL

### Customization Guidelines

1. **Start with the closest example** to your use case
2. **Modify storage settings** for your infrastructure
3. **Adjust retention policies** based on your requirements
4. **Configure security settings** appropriate for your environment
5. **Set up monitoring and alerting** for your notification systems
6. **Test thoroughly** in a non-production environment first

### Common Customizations

**Adjust Retention Policy:**
```yaml
retention:
  max_backups: 20        # Increase/decrease based on storage capacity
  max_age: "336h"        # 14 days instead of default
  keep_daily: 14         # Keep daily backups for 2 weeks
  keep_weekly: 8         # Keep weekly backups for 2 months
  keep_monthly: 6        # Keep monthly backups for 6 months
```

**Change Compression Settings:**
```yaml
compression:
  enabled: true
  algorithm: "lz4"       # Fast: lz4, Balanced: gzip, Best: zstd
  level: 3               # 1 (fast) to 9 (best compression)
  threshold: 10240       # Only compress files larger than 10KB
```

**Modify Storage Location:**
```yaml
storage:
  provider: local
  local:
    base_path: "/backup/mysql-schema-sync"  # Custom path
    permissions: "0600"                     # Restrictive permissions
```

## Testing Configurations

### Validation

Before using any configuration in production:

```bash
# Validate configuration syntax
mysql-schema-sync backup validate-config --config your-config.yaml

# Test storage connectivity
mysql-schema-sync backup test-connection --config your-config.yaml

# Test database connectivity
mysql-schema-sync test-connection --config your-config.yaml

# Perform dry-run backup
mysql-schema-sync backup create --config your-config.yaml --dry-run
```

### Integration Testing

```bash
# Create test backup
mysql-schema-sync backup create --config your-config.yaml --description "Integration test"

# List backups
mysql-schema-sync backup list --config your-config.yaml

# Validate backup
BACKUP_ID=$(mysql-schema-sync backup list --config your-config.yaml --limit 1 --format json | jq -r '.[0].id')
mysql-schema-sync backup validate --config your-config.yaml --backup-id $BACKUP_ID

# Test rollback planning
mysql-schema-sync rollback plan --config your-config.yaml --backup-id $BACKUP_ID

# Clean up test backup
mysql-schema-sync backup delete --config your-config.yaml --backup-id $BACKUP_ID
```

## Security Considerations

### Encryption Keys

**Generate Strong Keys:**
```bash
# Generate 256-bit encryption key
openssl rand -hex 32

# Generate key file
openssl rand 32 > /secure/path/backup.key
chmod 600 /secure/path/backup.key
```

**Key Storage Options:**
1. **Environment Variables** (development/CI/CD)
2. **Key Files** (production with proper permissions)
3. **Key Management Services** (enterprise: AWS KMS, Azure Key Vault, HashiCorp Vault)

### Access Control

**File Permissions:**
```bash
# Restrictive permissions for configuration files
chmod 600 config-prod.yaml

# Secure backup directory
chmod 700 /path/to/backups
chown backup-user:backup-group /path/to/backups
```

**Cloud Storage Security:**
- Use IAM roles instead of access keys when possible
- Enable bucket encryption and versioning
- Configure bucket policies to restrict access
- Enable access logging and monitoring

## Monitoring and Alerting

### Health Checks

Set up regular health checks:

```bash
#!/bin/bash
# backup-health-check.sh

CONFIG_FILE="config-prod.yaml"

# Check backup system health
if ! mysql-schema-sync backup health-check --config $CONFIG_FILE; then
    echo "Backup system health check failed" | mail -s "Backup Alert" ops@company.com
fi

# Check storage connectivity
if ! mysql-schema-sync backup test-connection --config $CONFIG_FILE; then
    echo "Backup storage connectivity failed" | mail -s "Storage Alert" ops@company.com
fi

# Validate recent backups
mysql-schema-sync backup validate-recent --config $CONFIG_FILE --hours 24
```

### Automated Monitoring

```bash
# Add to crontab for regular monitoring
0 */6 * * * /path/to/backup-health-check.sh

# Weekly comprehensive check
0 2 * * 0 mysql-schema-sync backup validate-all --config config-prod.yaml

# Monthly storage optimization
0 3 1 * * mysql-schema-sync backup storage-optimize --config config-prod.yaml
```

## Troubleshooting

### Common Issues

1. **Permission Denied:**
   - Check file/directory permissions
   - Verify user has access to backup location
   - Check database user privileges

2. **Storage Connection Failed:**
   - Verify credentials and network connectivity
   - Check firewall rules
   - Test with cloud provider CLI tools

3. **Backup Validation Failed:**
   - Check for file corruption
   - Verify encryption keys
   - Test storage provider health

4. **Out of Space:**
   - Implement retention policies
   - Monitor storage usage
   - Consider compression settings

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export MYSQL_SCHEMA_SYNC_DEBUG=true
export MYSQL_SCHEMA_SYNC_LOG_LEVEL=debug
mysql-schema-sync backup create --config your-config.yaml
```

## Migration from Other Tools

### From mysqldump

```bash
# Import existing mysqldump backup
mysql-schema-sync backup import --file backup.sql --format mysqldump

# Convert to native format
mysql-schema-sync backup convert --backup-id imported-backup-id --format native
```

### From Other Backup Tools

Most backup tools can be integrated by:
1. Creating a backup using the existing tool
2. Importing the backup into mysql-schema-sync
3. Gradually transitioning to native backups

## Best Practices

1. **Start Simple:** Begin with development configuration and gradually add features
2. **Test Thoroughly:** Always test configurations in non-production environments
3. **Monitor Actively:** Set up comprehensive monitoring and alerting
4. **Secure by Default:** Use encryption and restrictive permissions
5. **Document Changes:** Keep configuration changes in version control
6. **Regular Validation:** Periodically validate backup integrity
7. **Disaster Recovery Testing:** Regularly test restore procedures
8. **Capacity Planning:** Monitor storage usage and plan for growth

## Support

For additional help with configuration:

1. **Documentation:** Check the main backup system guide
2. **Validation:** Use built-in configuration validation tools
3. **Community:** Join project discussions and forums
4. **Issues:** Report configuration problems on the project repository

---

*These examples are regularly updated to reflect best practices and new features. Always refer to the latest documentation for the most current information.*