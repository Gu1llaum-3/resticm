# resticm

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/Gu1llaum-3/resticm?style=for-the-badge)](https://github.com/Gu1llaum-3/resticm/releases)
[![License](https://img.shields.io/github/license/Gu1llaum-3/resticm?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux-lightgrey?style=for-the-badge&logo=linux)](https://github.com/Gu1llaum-3/resticm/releases)

**resticm** (Restic Manager) is a modern CLI wrapper for [restic](https://restic.net/) that provides automated backup workflows, multi-backend replication, webhook notifications, and configuration management.

## ✨ Features

- **🔄 Automated Workflows** - Run backup + forget + prune + check + copy in a single command
- **🗄️ Multi-Backend Sync** - Keep all repositories perfectly synchronized (same snapshots, same retention)
- **🔔 Webhook Notifications** - Slack, Discord, ntfy, Google Chat, Uptime Kuma, and generic webhooks
- **📋 Configuration Contexts** - Easily switch between different configurations (production, staging, etc.)
- **🪝 Hook Scripts** - Pre/post backup hooks for database dumps, custom scripts, etc.
- **📊 Structured Logging** - File-based logging with rotation and optional JSON output
- **🔒 Security** - File locking to prevent concurrent runs, secure permission validation
- **⏰ Smart Deep Checks** - Automatic deep verification at configurable intervals

## 📦 Installation

### Quick Install (Recommended)

Install the latest version with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/Gu1llaum-3/resticm/main/install/unix.sh | bash
```

**Install a specific version:**

```bash
RESTICM_VERSION=v0.4.0 curl -sSL https://raw.githubusercontent.com/Gu1llaum-3/resticm/main/install/unix.sh | bash
```

**Custom installation directory:**

```bash
INSTALL_DIR=/opt/bin curl -sSL https://raw.githubusercontent.com/Gu1llaum-3/resticm/main/install/unix.sh | bash
```

### Manual Installation

Download the appropriate binary for your platform from the [releases page](https://github.com/Gu1llaum-3/resticm/releases):

```bash
# Example for Linux amd64
tar -xzf resticm_Linux_x86_64.tar.gz
cd resticm_Linux_x86_64
sudo mv resticm /usr/local/bin/
sudo chmod +x /usr/local/bin/resticm
```

### From Source

```bash
# Clone the repository
git clone https://github.com/Gu1llaum-3/resticm.git
cd resticm

# Build
make build

# Install to /usr/local/bin (requires sudo)
make install
```

### Prerequisites

- [restic](https://restic.net/) must be installed and available in your PATH
- Go 1.24+ (for building from source only)

## 🚀 Quick Start

### 1. Create Configuration

```bash
# Copy the example configuration
mkdir -p ~/.config/resticm
cp config.example.yaml ~/.config/resticm/config.yaml

# Set secure permissions (required)
chmod 600 ~/.config/resticm/config.yaml
```

### 2. Edit Configuration

Edit `~/.config/resticm/config.yaml` with your settings:

```yaml
# Primary repository
repository: "s3:s3.amazonaws.com/my-bucket/restic"
password: "your-secure-password"

# AWS credentials (for S3 backends)
aws_access_key_id: "AKIAIOSFODNN7EXAMPLE"
aws_secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# Directories to backup
directories:
  - /etc
  - /home
  - /var/www

# Exclusion patterns
exclude_patterns:
  - "*.tmp"
  - "**/node_modules/**"
  - "**/.git/**"

# Retention policy
retention:
  keep_within: "7d"
  keep_hourly: 24
  keep_daily: 7
  keep_weekly: 4
  keep_monthly: 12
  keep_yearly: 5
```

### 3. Initialize Repository

```bash
resticm init
```

### 4. Run Your First Backup

```bash
# Default workflow: backup + forget + copy
resticm

# Or explicitly
resticm backup
```

## 📖 Usage

### Multi-Backend Philosophy

resticm is designed with a **"mirror sync"** philosophy: all configured copy backends are kept **identical** to the primary repository. This means:

| Operation | Primary | Copy Backends |
|-----------|---------|---------------|
| `backup` | ✅ | (via copy) |
| `forget` | ✅ | ✅ automatic |
| `prune` | ✅ | ✅ automatic |
| `check` | ✅ | ✅ automatic |

If you need to operate on the primary only, use the `--primary-only` flag.

### Default Workflow

Running `resticm` without arguments executes the default workflow:

1. **Backup** - Creates a new snapshot on primary
2. **Forget** - Applies retention policy on primary
3. **Copy & Sync** - For each secondary backend:
   - Copy new snapshots
   - Apply same retention policy (forget)
   - Prune if `--prune` is set
   - Check if `--check` is set

This ensures all backends stay **perfectly synchronized**.

```bash
# Run default workflow
resticm

# With additional options
resticm --prune          # Also run prune
resticm --check          # Also run repository check
resticm --deep           # Run deep check (verify all data)
resticm -t production    # Add extra tag
resticm --no-copy        # Skip copy to secondary backends
```

### Individual Commands

```bash
# Backup
resticm backup
resticm backup -t mytag          # Add custom tag

# Forget (apply retention policy) - applies to ALL backends by default
resticm forget
resticm forget --prune           # Also prune
resticm forget --all-hosts       # Process all hosts
resticm forget --primary-only    # Only primary, skip copy backends

# Prune (remove unused data) - applies to ALL backends by default
resticm prune
resticm prune --primary-only     # Only primary, skip copy backends

# Check repository integrity - applies to ALL backends by default
resticm check
resticm check --deep             # Full data verification
resticm check --auto             # Auto deep-check if interval elapsed
resticm check --subset 1/5       # Check 20% of data
resticm check --primary-only     # Only primary, skip copy backends

# Copy to secondary backends
resticm copy
resticm copy --all               # Copy all hosts
resticm copy --to secondary      # Specific backend

# Full maintenance - synchronized across ALL backends:
# Primary: backup → forget → prune → check
# Each copy backend: copy → forget → prune → check
resticm full
resticm full --deep              # Force deep check on all backends
```

### Repository Management

```bash
# Initialize repositories
resticm init                     # Initialize primary
resticm init --all               # Initialize all backends
resticm init --backend secondary # Initialize specific backend

# View snapshots
resticm snapshots
resticm snapshots --all          # All hosts
resticm snapshots --all-backends # All backends
resticm snapshots --latest       # Only latest
resticm snapshots --json         # JSON output

# Repository statistics
resticm stats
resticm stats --all-backends     # All backends

# Configuration info
resticm info

# Remove stale locks
resticm unlock
resticm unlock -f                # Force without confirmation
resticm unlock --restic          # Also unlock restic repository locks
resticm unlock --all-backends    # Unlock all backends (with --restic)
```

### Context Management

Contexts allow switching between different configurations:

```bash
# Show current context
resticm context

# Switch context
resticm context use ~/.config/resticm/production.yaml
resticm context use /etc/resticm/server1.yaml

# List available configs
resticm context list

# Reset to default
resticm context reset
```

### Backend Management

```bash
# Show current backend
resticm backend

# Switch active backend
resticm backend use secondary
resticm backend use primary

# List available backends
resticm backend list
```

### Run Any Restic Command

```bash
# Pass-through to restic with current context credentials
resticm run snapshots
resticm run list locks
resticm run restore latest --target /tmp/restore
resticm run mount /mnt/restic
```

### Export Environment Variables

Use `resticm env` to export restic environment variables for direct `restic` CLI usage:

```bash
# Export variables using active backend
eval $(resticm env)
restic snapshots

# Use specific backend temporarily
eval $(resticm env --backend s3-backup)
restic check

# Export to file for later use
resticm env > ~/.resticm.env
source ~/.resticm.env
restic snapshots

# Fish shell
resticm env --format fish | source

# PowerShell
resticm env --format powershell | Invoke-Expression
```

Exported variables:
- `RESTIC_REPOSITORY` - Repository URL
- `RESTIC_PASSWORD` - Repository password
- `AWS_ACCESS_KEY_ID` - AWS access key (if configured)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key (if configured)

### Global Flags

```bash
-c, --config string    Config file path
-v, --verbose          Verbose output
-n, --dry-run          Perform trial run without changes
    --json             Output in JSON format
```

## ⚙️ Configuration

### Configuration File Locations

resticm searches for configuration in this order:

1. `--config` flag (if specified)
2. Active context (if set via `resticm context use`)
3. `./config.yaml` (current directory)
4. `~/.config/resticm/config.yaml`
5. `/etc/resticm/config.yaml`

### Full Configuration Example

See [`config.example.yaml`](config.example.yaml) for a complete example with all options documented.

### Key Configuration Sections

#### Primary Repository

```yaml
repository: "s3:s3.amazonaws.com/bucket/path"
password: "your-password"
# OR use password file
password_file: "/root/.restic-password"

# AWS credentials (for S3)
aws_access_key_id: "AKIA..."
aws_secret_access_key: "..."
```

#### Directories & Exclusions

```yaml
directories:
  - /etc
  - /home
  - /var/www

exclude_patterns:
  - "*.tmp"
  - "**/.git/**"
  - "**/node_modules/**"

# Or use exclude file
exclude_file: "/etc/resticm/excludes.txt"
```

#### Retention Policy

```yaml
retention:
  keep_within: "7d"      # Keep all within 7 days
  keep_hourly: 24        # Keep 24 hourly snapshots
  keep_daily: 7          # Keep 7 daily snapshots
  keep_weekly: 4         # Keep 4 weekly snapshots
  keep_monthly: 12       # Keep 12 monthly snapshots
  keep_yearly: 5         # Keep 5 yearly snapshots
```

#### Secondary Backends

> **⚠️ Important - S3 Limitations**: Due to restic's use of global environment variables 
> (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`), you **cannot use multiple native S3 
> backends with different credentials** in the same configuration. 
>
> **Solution**: Use **rclone** for additional S3 backends with different credentials. 
> See the rclone example below.

```yaml
backends:
  # Example 1: Native S3 backend (shares credentials with primary)
  secondary:
    repository: "s3:s3.eu-west-1.amazonaws.com/dr-bucket/restic"
    password: "secondary-password"
    # NOTE: These credentials are IGNORED if primary is also S3
    # Restic will use the primary's AWS credentials for ALL S3 backends
    aws_access_key_id: "AKIA..."
    aws_secret_access_key: "..."

  # Example 2: Local backend (no credential conflicts)
  local:
    repository: "/mnt/backup/restic"
    password: "local-password"

  # Example 3: Rclone backend (RECOMMENDED for multiple S3 with different credentials)
  # Requires: rclone installed and configured (https://rclone.org/)
  aws-account-2:
    repository: "rclone:myremote:my-bucket/restic"
    password: "rclone-repo-password"
    # Configure the remote in ~/.config/rclone/rclone.conf:
    # [myremote]
    # type = s3
    # provider = AWS
    # access_key_id = AKIA...DIFFERENT...CREDENTIALS
    # secret_access_key = ...DIFFERENT...SECRET...
    # region = us-west-2

# Backends to copy to after each backup
# These backends will be kept PERFECTLY SYNCHRONIZED:
# - Same snapshots (via copy)
# - Same retention policy (via forget)
# - Same pruning (via prune)
# - Same integrity checks (via check)
copy_to_backends:
  - secondary
  - local
  - aws-account-2
```

> **Note**: When initializing copy backends with `resticm init --backend <name>`,
> chunker parameters are automatically copied from the primary repository to ensure
> optimal deduplication.

#### Notifications

Send real-time alerts about backup operations via multiple providers (Slack, Discord, ntfy, Google Chat, Uptime Kuma, webhooks).

```yaml
notifications:
  enabled: true
  notify_on_success: false
  notify_on_error: true

  providers:
    # Slack
    - type: slack
      url: "https://hooks.slack.com/services/T.../B.../..."

    # Discord
    - type: discord
      url: "https://discord.com/api/webhooks/..."

    # ntfy.sh
    - type: ntfy
      url: "https://ntfy.sh"
      options:
        topic: "my-backup-alerts"

    # Google Chat
    - type: google_chat
      url: "https://chat.googleapis.com/v1/spaces/..."

    # Uptime Kuma
    - type: uptime_kuma
      url: "https://uptime.example.com/api/push/..."

    # Generic webhook
    - type: webhook
      url: "https://example.com/api/webhook"
```

📖 **[Complete Notifications Documentation →](docs/notifications.md)**

#### Hooks

Hooks allow you to run custom scripts at different stages of the backup workflow. This is particularly useful for:
- Database dumps before backup
- Application-specific preparation (stopping services, creating snapshots, etc.)
- Cleanup operations after backup
- Custom notifications or logging

```yaml
hooks:
  pre_backup: "/etc/resticm/hooks/pre-backup.sh"
  post_backup: "/etc/resticm/hooks/post-backup.sh"
  on_error: "/etc/resticm/hooks/on-error.sh"
  on_success: "/etc/resticm/hooks/on-success.sh"
```

##### Hook Execution Order

1. **pre_backup** - Runs before backup starts
2. **backup** - Main backup operation
3. **post_backup** - Runs after backup (success or failure)
4. **on_error** OR **on_success** - Runs based on overall result

##### Exit Code Handling

**⚠️ Critical**: Hooks use standard Unix exit codes to determine success or failure:

- **Exit 0** - Success, workflow continues
- **Exit != 0** - Failure, behavior depends on the command

**Behavior on hook failure (exit code != 0):**

| Command | Behavior |
|---------|----------|
| `resticm backup` | ❌ **Aborts immediately** - backup is NOT executed |
| `resticm` (default workflow) | ⚠️ **Continues** - skips backup, but runs forget/copy |

**When `pre_backup` hook fails:**
- ❌ The backup operation is **skipped** (never executed)
- 🪝 The `on_error` hook is triggered
- 📢 Error notifications are sent
- ⚠️ resticm returns an error code at the end
- 🔄 Other operations (forget, copy) may continue (in default workflow)

**Recommendation**: Use `resticm backup` explicitly in critical scenarios (like database dumps) to ensure the entire workflow stops if preparation fails. The default workflow (`resticm`) is designed to be resilient and continue with maintenance tasks even if backup fails.

##### Hook Environment Variables

Hook scripts receive environment variables:
- `BACKUP_STATUS` - "success" or "failure" (post_backup)
- `BACKUP_ERROR` - Error message if failed (post_backup)
- `ERROR` - Error message (on_error)

##### Example: PostgreSQL Backup

**pre-backup.sh** - Create database dump:
```bash
#!/bin/bash
set -e  # Exit on any error

BACKUP_DIR="/mnt/backup_temp"

echo "Creating temporary directory..."
mkdir -p $BACKUP_DIR || exit 1

echo "Dumping PostgreSQL database..."
if ! docker exec -t postgres_container pg_dumpall -c -U postgres > $BACKUP_DIR/dump.sql; then
    echo "ERROR: Database dump failed"
    exit 1
fi

# Verify dump is not empty
if [ ! -s $BACKUP_DIR/dump.sql ]; then
    echo "ERROR: Dump file is empty"
    exit 1
fi

echo "Database dump successful: $(du -h $BACKUP_DIR/dump.sql | cut -f1)"
exit 0
```

**post-backup.sh** - Cleanup temporary files:
```bash
#!/bin/bash

BACKUP_DIR="/mnt/backup_temp"

echo "Cleaning up temporary dump..."
rm -f $BACKUP_DIR/dump.sql

echo "Cleanup completed"
exit 0
```

**Configuration** - Include dump directory in backup:
```yaml
directories:
  - /mnt/backup_temp              # Temporary dump location
  - /var/lib/docker/volumes       # Other data
  - /etc

hooks:
  pre_backup: "/etc/resticm/hooks/pre-backup.sh"
  post_backup: "/etc/resticm/hooks/post-backup.sh"
```

📖 **[Complete Hooks Documentation →](docs/hooks.md)**

This ensures:
1. Database is dumped **before** backup starts
2. If dump fails, backup is **aborted** (no incomplete backup)
3. Temporary dump is **cleaned up** after backup
4. All operations are **logged** and **monitored**

##### Hook Permissions

Hooks must be:
- **Executable**: `chmod +x /etc/resticm/hooks/*.sh`
- **Owned by root** (for system-wide configs): `chown root:root /etc/resticm/hooks/*.sh`

If a hook exists but is not executable, resticm will fail with an error message.

#### Logging

```yaml
logging:
  file: "/var/log/resticm/resticm.log"
  max_size_mb: 10
  max_files: 5
  level: "info"     # debug, info, warn, error
  console: true
  json: false       # Set true for log aggregation
```

#### Deep Check Interval

```yaml
# Automatically run deep check every N days
deep_check_interval_days: 30
```

## 🔧 Automation

### Cron Example

```bash
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/resticm -c /etc/resticm/config.yaml 2>&1 | logger -t resticm

# Weekly full maintenance on Sunday at 3 AM
0 3 * * 0 /usr/local/bin/resticm full -c /etc/resticm/config.yaml 2>&1 | logger -t resticm
```

### Systemd Timer

Create `/etc/systemd/system/resticm.service`:

```ini
[Unit]
Description=Resticm Backup
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/resticm -c /etc/resticm/config.yaml
Nice=10
IOSchedulingClass=idle
```

Create `/etc/systemd/system/resticm.timer`:

```ini
[Unit]
Description=Run resticm backup daily

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true
RandomizedDelaySec=1800

[Install]
WantedBy=timers.target
```

Enable the timer:

```bash
systemctl daemon-reload
systemctl enable --now resticm.timer
```

## 🛠️ Development

### Build

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
```

### Test

```bash
make test           # Run tests
make test-cover     # Run tests with coverage
make test-race      # Run tests with race detector
```

### Lint

```bash
make lint           # Run golangci-lint
```

### Shell Completions

```bash
make completions

# Install completions
# Bash
sudo cp completions/resticm.bash /etc/bash_completion.d/resticm

# Zsh
sudo cp completions/resticm.zsh /usr/local/share/zsh/site-functions/_resticm

# Fish
cp completions/resticm.fish ~/.config/fish/completions/
```

## 📁 Project Structure

```
resticm/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command & default workflow
│   ├── backup.go          # Backup command
│   ├── forget.go          # Forget command
│   ├── prune.go           # Prune command
│   ├── check.go           # Check command
│   ├── copy.go            # Copy command
│   ├── full.go            # Full maintenance command
│   ├── init.go            # Repository initialization
│   ├── snapshots.go       # List snapshots
│   ├── stats.go           # Repository statistics
│   ├── info.go            # Configuration info
│   ├── context.go         # Context management
│   ├── backend.go         # Backend management
│   ├── run.go             # Pass-through restic commands
│   ├── unlock.go          # Remove stale locks
│   └── version.go         # Version information
├── internal/
│   ├── config/            # Configuration handling
│   ├── restic/            # Restic wrapper
│   ├── hooks/             # Hook execution
│   ├── notify/            # Notification providers
│   ├── logging/           # Structured logging
│   └── security/          # File locking
├── config.example.yaml    # Example configuration
├── Makefile               # Build automation
└── main.go                # Entry point
```

## 🔐 Security Considerations

1. **Configuration File Permissions**: The config file must have `600` permissions (owner read/write only). resticm will refuse to run otherwise.

2. **Password Management**: Prefer using `password_file` or environment variables (`RESTIC_PASSWORD`) over storing passwords in the config file.

3. **Root Privileges**: For system-wide backups, run resticm as root. A warning is displayed when running without root privileges.

4. **Lock Files**: resticm uses file locking to prevent concurrent runs:
   - Root user: `/var/lock/resticm.lock`
   - Regular user: `~/.local/share/resticm/resticm.lock`

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [restic](https://restic.net/) - The amazing backup program this tool wraps
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [fatih/color](https://github.com/fatih/color) - Terminal colors
