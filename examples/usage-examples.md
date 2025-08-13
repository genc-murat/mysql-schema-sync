# MySQL Schema Sync Usage Examples

This document provides comprehensive examples of using mysql-schema-sync with various visual enhancement options and output formats.

## Basic Usage Examples

### 1. Standard Schema Comparison
```bash
# Basic comparison with default visual enhancements
mysql-schema-sync \
  --source-host=localhost --source-user=root --source-db=source_db \
  --target-host=localhost --target-user=root --target-db=target_db
```

### 2. Using Configuration File
```bash
# Generate and use a configuration file
mysql-schema-sync config > config.yaml
mysql-schema-sync --config=config.yaml
```

### 3. Dry Run with Enhanced Output
```bash
# Preview changes with colorized, formatted output
mysql-schema-sync --config=config.yaml --dry-run --verbose
```

## Visual Enhancement Examples

### 4. Different Color Themes
```bash
# Dark theme (default)
mysql-schema-sync --config=config.yaml --theme=dark

# Light theme for bright terminals
mysql-schema-sync --config=config.yaml --theme=light

# High contrast for accessibility
mysql-schema-sync --config=config.yaml --theme=high-contrast

# Auto-detect terminal theme
mysql-schema-sync --config=config.yaml --theme=auto
```

### 5. Table Styling Options
```bash
# Default ASCII table borders
mysql-schema-sync --config=config.yaml --table-style=default

# Rounded corners (requires Unicode support)
mysql-schema-sync --config=config.yaml --table-style=rounded

# Heavy borders for emphasis
mysql-schema-sync --config=config.yaml --table-style=border

# Minimal borders for clean look
mysql-schema-sync --config=config.yaml --table-style=minimal
```

### 6. Icon and Progress Options
```bash
# Disable Unicode icons (use ASCII alternatives)
mysql-schema-sync --config=config.yaml --no-icons

# Disable progress indicators
mysql-schema-sync --config=config.yaml --no-progress

# Disable all interactive prompts
mysql-schema-sync --config=config.yaml --no-interactive
```

## Output Format Examples

### 7. Table Format (Default)
```bash
mysql-schema-sync --config=config.yaml --format=table
```
**Output Example:**
```
┌─────────────────┬──────────────┬─────────────────────────────────┐
│ Change Type     │ Object       │ Description                     │
├─────────────────┼──────────────┼─────────────────────────────────┤
│ ➕ ADD TABLE    │ users        │ Create new table with 5 columns │
│ 🔄 MODIFY TABLE │ products     │ Add column 'created_at'         │
│ ➖ DROP COLUMN  │ orders.notes │ Remove unused notes column      │
└─────────────────┴──────────────┴─────────────────────────────────┘
```

### 8. JSON Format
```bash
mysql-schema-sync --config=config.yaml --format=json
```
**Output Example:**
```json
{
  "summary": {
    "total_changes": 3,
    "tables_added": 1,
    "tables_modified": 1,
    "columns_dropped": 1
  },
  "changes": [
    {
      "type": "add_table",
      "object": "users",
      "description": "Create new table with 5 columns",
      "sql": "CREATE TABLE users (...)"
    }
  ]
}
```

### 9. YAML Format
```bash
mysql-schema-sync --config=config.yaml --format=yaml
```
**Output Example:**
```yaml
summary:
  total_changes: 3
  tables_added: 1
  tables_modified: 1
  columns_dropped: 1
changes:
  - type: add_table
    object: users
    description: Create new table with 5 columns
    sql: CREATE TABLE users (...)
```

### 10. Compact Format (for Scripting)
```bash
mysql-schema-sync --config=config.yaml --format=compact
```
**Output Example:**
```
CHANGES: 3
ADD_TABLE: users
MODIFY_TABLE: products
DROP_COLUMN: orders.notes
STATUS: SUCCESS
```

## Accessibility Examples

### 11. High Contrast Mode
```bash
# Optimized for visual impairments
mysql-schema-sync --config=config.yaml \
  --theme=high-contrast \
  --no-icons \
  --table-style=minimal \
  --max-table-width=80
```

### 12. Screen Reader Friendly
```bash
# ASCII-only output for screen readers
mysql-schema-sync --config=config.yaml \
  --no-color \
  --no-icons \
  --table-style=minimal \
  --verbose
```

## Automation Examples

### 13. CI/CD Pipeline
```bash
# Non-interactive mode for automation
mysql-schema-sync --config=ci-config.yaml \
  --format=compact \
  --no-color \
  --no-progress \
  --no-interactive \
  --auto-approve
```

### 14. Scripting with JSON Output
```bash
# Generate JSON report for processing
mysql-schema-sync --config=config.yaml \
  --format=json \
  --dry-run > schema-changes.json

# Parse results in script
if jq -e '.summary.total_changes > 0' schema-changes.json; then
  echo "Schema changes detected"
fi
```

### 15. Log File Generation
```bash
# Generate detailed logs with timestamps
mysql-schema-sync --config=config.yaml \
  --verbose \
  --log-file=schema-sync-$(date +%Y%m%d-%H%M%S).log
```

## Environment Variable Examples

### 16. Using Environment Variables
```bash
# Set display preferences via environment
export MYSQL_SCHEMA_SYNC_THEME=light
export MYSQL_SCHEMA_SYNC_FORMAT=yaml
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_TABLE_STYLE=rounded

mysql-schema-sync --config=config.yaml
```

### 17. Database Credentials via Environment
```bash
# Secure credential handling
export MYSQL_SCHEMA_SYNC_SOURCE_PASSWORD=source_password
export MYSQL_SCHEMA_SYNC_TARGET_PASSWORD=target_password

mysql-schema-sync --config=config.yaml
```

## Advanced Usage Examples

### 18. Custom Table Width
```bash
# Adjust table width for different terminal sizes
mysql-schema-sync --config=config.yaml --max-table-width=160  # Wide terminals
mysql-schema-sync --config=config.yaml --max-table-width=60   # Narrow terminals
```

### 19. Combining Multiple Options
```bash
# Comprehensive example with multiple visual options
mysql-schema-sync \
  --config=config.yaml \
  --theme=dark \
  --format=table \
  --table-style=rounded \
  --max-table-width=140 \
  --verbose \
  --dry-run
```

### 20. Troubleshooting with Enhanced Output
```bash
# Maximum verbosity for debugging
mysql-schema-sync --config=config.yaml \
  --verbose \
  --theme=high-contrast \
  --table-style=border \
  --log-file=debug.log
```

## Output Format Comparison

| Format  | Use Case                    | Machine Readable | Human Readable | Colors | Icons |
|---------|----------------------------|------------------|----------------|--------|-------|
| table   | Interactive use            | No               | Yes            | Yes    | Yes   |
| json    | API integration, scripting | Yes              | No             | No     | No    |
| yaml    | Configuration, reports     | Yes              | Yes            | No     | No    |
| compact | Shell scripting, parsing   | Partial          | No             | No     | No    |

## Terminal Compatibility

### Modern Terminals (Full Support)
- iTerm2, Terminal.app (macOS)
- Windows Terminal, PowerShell
- GNOME Terminal, Konsole (Linux)
- VS Code integrated terminal

**Features:** Full color support, Unicode icons, progress bars, interactive prompts

### Legacy Terminals (Graceful Fallback)
- Command Prompt (Windows)
- Basic SSH terminals
- Screen/tmux sessions

**Features:** ASCII icons, basic colors, simplified tables

### Accessibility Tools
- Screen readers (NVDA, JAWS, VoiceOver)
- High contrast themes
- Terminal magnifiers

**Recommendations:** Use `--no-icons`, `--theme=high-contrast`, `--table-style=minimal`

## Best Practices

1. **Development**: Use `--theme=dark --table-style=rounded --verbose`
2. **Production**: Use `--theme=auto --format=table --log-file=sync.log`
3. **CI/CD**: Use `--format=compact --no-color --no-interactive`
4. **Accessibility**: Use `--theme=high-contrast --no-icons --table-style=minimal`
5. **Scripting**: Use `--format=json` or `--format=compact`
6. **Debugging**: Use `--verbose --theme=high-contrast --log-file=debug.log`