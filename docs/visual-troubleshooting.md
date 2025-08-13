# Visual Features Troubleshooting Guide

This guide helps resolve issues with the visual enhancement features in MySQL Schema Sync.

## Quick Diagnostics

### Test Your Terminal Capabilities

Run this command to see what your terminal supports:

```bash
mysql-schema-sync --config=config.yaml --verbose --dry-run
```

Look for output like:
```
Terminal capabilities detected:
  Color support: true
  Unicode support: true  
  Interactive: true
  Width: 120 columns
```

### Basic Compatibility Test

```bash
# Test colors
echo -e "\033[31mRed\033[0m \033[32mGreen\033[0m \033[33mYellow\033[0m"

# Test Unicode icons
echo "Icons: ➕ 🔄 ➖ ✅ ❌ ⚠️"

# Test table characters
echo "┌─────┬─────┐"
echo "│ A   │ B   │"
echo "└─────┴─────┘"
```

## Common Issues and Solutions

### 1. Colors Not Displaying

**Symptoms:**
- All output appears in plain text
- No color highlighting for changes
- Themes have no effect

**Causes and Solutions:**

#### Terminal doesn't support colors
```bash
# Check color support
echo $TERM
echo $COLORTERM

# Force color output (if terminal actually supports it)
mysql-schema-sync --config=config.yaml --theme=dark

# Disable colors if not supported
mysql-schema-sync --config=config.yaml --no-color
```

#### Environment variables disabling colors
```bash
# Check for color-disabling variables
env | grep -i color
env | grep -i no_color

# Unset if found
unset NO_COLOR
unset MYSQL_SCHEMA_SYNC_NO_COLOR
```

#### Piping output to files or other commands
```bash
# Colors are automatically disabled when piping
mysql-schema-sync --config=config.yaml | less

# Force colors when piping (if needed)
mysql-schema-sync --config=config.yaml --theme=dark | less -R
```

### 2. Unicode Icons Not Showing

**Symptoms:**
- Icons appear as question marks or boxes
- Garbled characters in output
- Missing symbols

**Solutions:**

#### Use ASCII alternatives
```bash
mysql-schema-sync --config=config.yaml --no-icons
```

#### Check terminal encoding
```bash
# Check current locale
locale

# Set UTF-8 if needed
export LC_ALL=en_US.UTF-8
export LANG=en_US.UTF-8
```

#### Terminal-specific fixes

**Windows Command Prompt:**
```cmd
chcp 65001
mysql-schema-sync --config=config.yaml --no-icons
```

**Windows PowerShell:**
```powershell
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
mysql-schema-sync --config=config.yaml
```

**SSH sessions:**
```bash
# Ensure UTF-8 forwarding
ssh -o SendEnv=LANG,LC_* user@host
```

### 3. Table Formatting Issues

**Symptoms:**
- Tables appear misaligned
- Borders are broken or missing
- Text wrapping incorrectly

**Solutions:**

#### Use minimal table style
```bash
mysql-schema-sync --config=config.yaml --table-style=minimal
```

#### Adjust table width
```bash
# For narrow terminals
mysql-schema-sync --config=config.yaml --max-table-width=80

# For wide terminals
mysql-schema-sync --config=config.yaml --max-table-width=160
```

#### Check terminal width
```bash
# Check current terminal width
tput cols

# Set appropriate width
mysql-schema-sync --config=config.yaml --max-table-width=$(tput cols)
```

### 4. Progress Indicators Not Working

**Symptoms:**
- No progress bars or spinners
- Static output during long operations
- Progress indicators appear garbled

**Solutions:**

#### Check if running in interactive terminal
```bash
# Test TTY detection
if [ -t 1 ]; then
    echo "Interactive terminal"
else
    echo "Non-interactive (disable progress)"
fi
```

#### Disable progress for non-interactive use
```bash
# For scripts and automation
mysql-schema-sync --config=config.yaml --no-progress

# For CI/CD
mysql-schema-sync --config=config.yaml --no-progress --no-interactive
```

#### Terminal-specific issues

**Screen/Tmux:**
```bash
# May need to disable progress
mysql-schema-sync --config=config.yaml --no-progress
```

**Remote SSH:**
```bash
# Check SSH configuration
ssh -o RequestTTY=yes user@host
```

### 5. Interactive Prompts Not Working

**Symptoms:**
- No confirmation prompts appear
- Cannot interact with the tool
- Prompts appear but don't accept input

**Solutions:**

#### Check TTY availability
```bash
# Verify stdin/stdout are connected to terminal
[ -t 0 ] && echo "stdin is TTY" || echo "stdin not TTY"
[ -t 1 ] && echo "stdout is TTY" || echo "stdout not TTY"
```

#### Force non-interactive mode
```bash
# For automation
mysql-schema-sync --config=config.yaml --no-interactive --auto-approve

# For scripts
mysql-schema-sync --config=config.yaml --no-interactive --dry-run
```

#### SSH and remote sessions
```bash
# Ensure proper TTY allocation
ssh -t user@host mysql-schema-sync --config=config.yaml
```

## Terminal-Specific Configurations

### Windows

#### Command Prompt (cmd.exe)
```yaml
display:
  color_enabled: false
  use_icons: false
  table_style: minimal
  show_progress: false
```

#### PowerShell
```yaml
display:
  color_enabled: true
  theme: auto
  use_icons: false  # Limited Unicode support
  table_style: default
  show_progress: true
```

#### Windows Terminal
```yaml
display:
  color_enabled: true
  theme: dark
  use_icons: true
  table_style: rounded
  show_progress: true
```

### macOS

#### Terminal.app
```yaml
display:
  color_enabled: true
  theme: auto
  use_icons: true
  table_style: default
  show_progress: true
```

#### iTerm2
```yaml
display:
  color_enabled: true
  theme: dark
  use_icons: true
  table_style: rounded
  show_progress: true
```

### Linux

#### GNOME Terminal
```yaml
display:
  color_enabled: true
  theme: auto
  use_icons: true
  table_style: rounded
  show_progress: true
```

#### Konsole (KDE)
```yaml
display:
  color_enabled: true
  theme: auto
  use_icons: true
  table_style: border
  show_progress: true
```

#### xterm
```yaml
display:
  color_enabled: true
  theme: auto
  use_icons: false  # May have Unicode issues
  table_style: default
  show_progress: true
```

## Accessibility Configurations

### Screen Readers

**NVDA (Windows):**
```yaml
display:
  color_enabled: false
  theme: high-contrast
  use_icons: false
  table_style: minimal
  max_table_width: 80
  output_format: table  # Structured for screen readers
```

**JAWS (Windows):**
```yaml
display:
  color_enabled: true
  theme: high-contrast
  use_icons: false
  table_style: minimal
  max_table_width: 60
```

**VoiceOver (macOS):**
```yaml
display:
  color_enabled: true
  theme: high-contrast
  use_icons: false
  table_style: minimal
  max_table_width: 80
```

### Visual Impairments

**High Contrast:**
```yaml
display:
  color_enabled: true
  theme: high-contrast
  use_icons: true  # High contrast icons can help
  table_style: border  # Clear borders
  max_table_width: 100
```

**Low Vision:**
```yaml
display:
  color_enabled: true
  theme: high-contrast
  use_icons: false
  table_style: border
  max_table_width: 60  # Larger text fits better
```

## Environment Variables for Quick Fixes

### Disable All Visual Features
```bash
export MYSQL_SCHEMA_SYNC_NO_COLOR=1
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_NO_PROGRESS=1
export MYSQL_SCHEMA_SYNC_TABLE_STYLE=minimal
export MYSQL_SCHEMA_SYNC_FORMAT=compact
```

### Maximum Compatibility Mode
```bash
export MYSQL_SCHEMA_SYNC_NO_COLOR=1
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_NO_PROGRESS=1
export MYSQL_SCHEMA_SYNC_NO_INTERACTIVE=1
export MYSQL_SCHEMA_SYNC_TABLE_STYLE=minimal
export MYSQL_SCHEMA_SYNC_MAX_TABLE_WIDTH=80
```

### Accessibility Mode
```bash
export MYSQL_SCHEMA_SYNC_THEME=high-contrast
export MYSQL_SCHEMA_SYNC_NO_ICONS=1
export MYSQL_SCHEMA_SYNC_TABLE_STYLE=minimal
export MYSQL_SCHEMA_SYNC_MAX_TABLE_WIDTH=80
```

## Testing Your Configuration

### Visual Feature Test
```bash
# Test all visual features
mysql-schema-sync --config=config.yaml --dry-run --verbose

# Test specific theme
mysql-schema-sync --config=config.yaml --theme=high-contrast --dry-run

# Test table formatting
mysql-schema-sync --config=config.yaml --table-style=rounded --dry-run

# Test output format
mysql-schema-sync --config=config.yaml --format=json --dry-run
```

### Compatibility Test Script
```bash
#!/bin/bash
echo "Testing MySQL Schema Sync visual compatibility..."

echo "1. Testing colors..."
mysql-schema-sync --version --theme=dark 2>/dev/null && echo "✓ Colors work" || echo "✗ Colors failed"

echo "2. Testing Unicode..."
mysql-schema-sync --version --no-color 2>/dev/null && echo "✓ Unicode works" || echo "✗ Unicode failed"

echo "3. Testing tables..."
mysql-schema-sync --config=config.yaml --dry-run --table-style=minimal 2>/dev/null && echo "✓ Tables work" || echo "✗ Tables failed"

echo "4. Testing progress..."
timeout 5 mysql-schema-sync --config=config.yaml --dry-run 2>/dev/null && echo "✓ Progress works" || echo "✗ Progress failed"
```

## Getting Help

If you're still experiencing issues:

1. **Check the main troubleshooting guide** in the README
2. **Run with verbose output** to see detailed error messages
3. **Test with minimal configuration** to isolate the issue
4. **Check terminal documentation** for specific compatibility notes
5. **Open an issue** on GitHub with your terminal type and error details

### Information to Include in Bug Reports

When reporting visual feature issues, please include:

```bash
# System information
echo "OS: $(uname -a)"
echo "Terminal: $TERM"
echo "Color term: $COLORTERM"
echo "Locale: $(locale)"
echo "Terminal size: $(tput cols)x$(tput lines)"

# Test output
mysql-schema-sync --version --verbose
mysql-schema-sync --config=config.yaml --dry-run --verbose 2>&1 | head -20
```