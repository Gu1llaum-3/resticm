# Hooks Guide

Hooks allow you to run custom scripts at different stages of the backup workflow. This is essential for preparing your system before backup (database dumps, stopping services, etc.) and cleaning up afterward.

## Table of Contents

- [Overview](#overview)
- [Hook Types](#hook-types)
- [Execution Order](#execution-order)
- [Exit Codes and Error Handling](#exit-codes-and-error-handling)
- [Environment Variables](#environment-variables)
- [Basic Setup](#basic-setup)
- [Architecture Patterns](#architecture-patterns)
  - [Pattern 1: Orchestrator (Recommended)](#pattern-1-orchestrator-recommended)
  - [Pattern 2: Monolithic Script](#pattern-2-monolithic-script)
  - [Pattern 3: Conditional Detection](#pattern-3-conditional-detection)
- [Complete Examples](#complete-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Hooks are executable scripts that resticm runs at specific points during the backup workflow. They enable you to:

- **Prepare data** before backup (database dumps, snapshots)
- **Stop services** temporarily for consistent backups
- **Clean up** temporary files after backup
- **Handle errors** with custom actions
- **Send custom notifications** on success

## Hook Types

resticm supports four types of hooks:

| Hook | When | Use Case |
|------|------|----------|
| `pre_backup` | Before backup starts | Database dumps, service preparation |
| `post_backup` | After backup (success or failure) | Cleanup, restart services |
| `on_success` | After successful backup | Custom notifications, post-processing |
| `on_error` | After failed backup | Alert systems, rollback operations |

## Execution Order

```
┌─────────────────────────────────────────────────────────┐
│                    resticm backup                       │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   1. pre_backup       │  ← Prepare system
              └───────────────────────┘
                          │
                    [if succeeds]
                          ▼
              ┌───────────────────────┐
              │   2. Backup operation │  ← Run restic
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   3. post_backup      │  ← Cleanup (only if backup ran)
              └───────────────────────┘
                          │
                ┌─────────┴─────────┐
                ▼                   ▼
        ┌───────────────┐   ┌───────────────┐
        │  4a. on_error │   │ 4b. on_success│  ← Result handlers
        └───────────────┘   └───────────────┘
```

## Bypassing Hooks

Sometimes you need to run a backup without executing hooks. For example:
- Testing backups without stopping services
- Emergency backups while services are already stopped
- Manual operations where hooks would interfere

Use the `--no-hooks` flag to skip all hooks:

```bash
# Run backup without any hooks
resticm backup --no-hooks

# Run full maintenance without hooks
resticm full --no-hooks
```

When `--no-hooks` is set:
- ❌ `pre_backup` is skipped
- ❌ `post_backup` is skipped
- ❌ `on_success` is skipped
- ❌ `on_error` is skipped
- ✅ Backup operation still runs normally
- ✅ Notifications still work (if configured)

## Exit Codes and Error Handling

Hooks use standard Unix exit codes to communicate success or failure:

```bash
exit 0   # Success - workflow continues
exit 1   # Failure - triggers error handling
```

### Critical Behavior: pre_backup Failures

**⚠️ Important**: When `pre_backup` fails (exit code != 0):

1. ❌ **Backup is SKIPPED** - restic is never executed
2. ❌ **post_backup does NOT run** - hook execution stops
3. 🪝 **on_error is triggered** - for notifications
4. 📢 **Error notifications are sent**
5. 💥 **resticm returns error code** at the end

### Other Hook Behaviors

**When `backup` operation fails:**
- 🪝 **post_backup IS executed** with `BACKUP_STATUS=failure` and `BACKUP_ERROR`
- 🪝 **on_error is triggered**
- 📢 **Error notifications are sent**

**When `post_backup` hook fails:**
- ⚠️ Failure is **logged but does NOT fail the backup**
- ✅ Backup is still considered successful
- 🪝 **on_success is still executed** (if backup succeeded)

### Command Behavior Differences

| Command | pre_backup failure | Behavior |
|---------|-------------------|----------|
| `resticm backup` | Exit != 0 | ❌ **Aborts immediately** |
| `resticm` (default) | Exit != 0 | ⚠️ **Continues to forget/copy** |

**Recommendation**: Use `resticm backup` explicitly for critical operations (like database backups) to ensure the entire workflow stops if preparation fails.

**⚠️ Important**: If `pre_backup` fails, `post_backup` will **NOT run**. This means cleanup operations in `post_backup` won't execute. Design your `pre_backup` scripts to clean up after themselves on failure, or handle cleanup in `on_error`.

### Example: Failing Hook

```bash
#!/bin/bash
# pre-backup.sh

echo "Starting database dump..."
if ! mariadb-dump mydb > /backup/mydb.sql; then
    echo "ERROR: Database dump failed"
    exit 1  # ← Backup will be SKIPPED
fi

echo "Database dump successful"
exit 0  # ← Backup will proceed
```

## Environment Variables

Hooks receive the following environment variables:

### Common Variables

| Variable | Description | Available in |
|----------|-------------|--------------|
| `BACKUP_STATUS` | "success" or "failure" | post_backup |
| `BACKUP_ERROR` | Error message if failed | post_backup |
| `ERROR` | Error details | on_error |

### Using Environment Variables

```bash
#!/bin/bash
# post-backup.sh

if [ "$BACKUP_STATUS" = "success" ]; then
    echo "✅ Backup completed successfully"
    # Start services, update status, etc.
else
    echo "❌ Backup failed: $BACKUP_ERROR"
    # Keep services stopped, send alerts, etc.
fi
```

## Basic Setup

### 1. Create Hook Directory

```bash
sudo mkdir -p /etc/resticm/hooks
sudo chmod 700 /etc/resticm/hooks
```

### 2. Create Hook Scripts

```bash
# Create pre-backup hook
sudo nano /etc/resticm/hooks/pre-backup.sh

# Make it executable
sudo chmod +x /etc/resticm/hooks/pre-backup.sh
sudo chown root:root /etc/resticm/hooks/pre-backup.sh
```

### 3. Configure resticm

```yaml
# /etc/resticm/config.yaml
hooks:
  pre_backup: "/etc/resticm/hooks/pre-backup.sh"
  post_backup: "/etc/resticm/hooks/post-backup.sh"
  on_error: "/etc/resticm/hooks/on-error.sh"
  on_success: "/etc/resticm/hooks/on-success.sh"
```

### 4. Test Hooks

```bash
# Test hook execution
sudo /etc/resticm/hooks/pre-backup.sh
echo $?  # Should output 0 for success

# Test with dry-run
sudo resticm --dry-run
```

## Architecture Patterns

### Pattern 1: Orchestrator (Recommended)

**Best for**: Multiple servers, complex workflows, team maintenance

Create a main script that calls modular sub-scripts:

#### Directory Structure

```
/etc/resticm/hooks/
├── pre-backup.sh              # Main orchestrator
├── post-backup.sh             # Main orchestrator
├── pre-backup.d/              # Pre-backup modules
│   ├── 10-dump-mariadb.sh
│   ├── 20-dump-postgres.sh
│   ├── 30-stop-containers.sh
│   └── 40-snapshot-volumes.sh
└── post-backup.d/             # Post-backup modules
    ├── 10-cleanup-dumps.sh
    ├── 20-start-containers.sh
    └── 30-verify-backup.sh
```

#### Main Orchestrator

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e  # Exit on any error

SCRIPT_DIR="/etc/resticm/hooks/pre-backup.d"
LOG_FILE="/var/log/resticm/hooks.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log "========================================="
log "Starting pre-backup hooks"
log "========================================="

# Execute all scripts in numerical order
for script in "$SCRIPT_DIR"/*.sh; do
    if [ ! -x "$script" ]; then
        log "⚠️  Skipping non-executable: $(basename $script)"
        continue
    fi
    
    script_name=$(basename "$script")
    log "🔄 Executing: $script_name"
    
    if ! "$script" 2>&1 | tee -a "$LOG_FILE"; then
        log "❌ Failed: $script_name"
        exit 1  # Stop on first failure
    fi
    
    log "✅ Completed: $script_name"
done

log "========================================="
log "All pre-backup hooks completed"
log "========================================="
exit 0
```

#### Module Example: MariaDB Dump

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.d/10-dump-mariadb.sh

set -e

BACKUP_DIR="/var/backups/databases"
RETENTION_DAYS=3

echo "📦 Dumping MariaDB databases..."

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Dump all databases
mariadb-dump \
    --all-databases \
    --single-transaction \
    --quick \
    --lock-tables=false \
    --events \
    --routines \
    --triggers \
    | gzip > "$BACKUP_DIR/mariadb-$(date +%Y%m%d-%H%M%S).sql.gz"

# Verify dump is not empty
if [ ! -s "$BACKUP_DIR/mariadb-"*".sql.gz" ]; then
    echo "ERROR: Dump file is empty"
    exit 1
fi

# Cleanup old dumps (keep last N days)
find "$BACKUP_DIR" -name "mariadb-*.sql.gz" -mtime +${RETENTION_DAYS} -delete

echo "✅ MariaDB dump completed: $(ls -lh "$BACKUP_DIR"/mariadb-*.sql.gz | tail -1 | awk '{print $5}')"
exit 0
```

#### Module Example: PostgreSQL Dump

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.d/20-dump-postgres.sh

set -e

BACKUP_DIR="/var/backups/databases"
RETENTION_DAYS=3

echo "📦 Dumping PostgreSQL databases..."

mkdir -p "$BACKUP_DIR"

# Dump all databases
sudo -u postgres pg_dumpall \
    | gzip > "$BACKUP_DIR/postgres-$(date +%Y%m%d-%H%M%S).sql.gz"

# Verify dump
if [ ! -s "$BACKUP_DIR/postgres-"*".sql.gz" ]; then
    echo "ERROR: Dump file is empty"
    exit 1
fi

# Cleanup old dumps
find "$BACKUP_DIR" -name "postgres-*.sql.gz" -mtime +${RETENTION_DAYS} -delete

echo "✅ PostgreSQL dump completed: $(ls -lh "$BACKUP_DIR"/postgres-*.sql.gz | tail -1 | awk '{print $5}')"
exit 0
```

#### Module Example: Stop Docker Containers

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.d/30-stop-containers.sh

set -e

echo "⏸️  Stopping containers for backup..."

# List of containers to stop
CONTAINERS=(
    "webapp-prod"
    "redis-cache"
    "worker-queue"
)

# Save list of stopped containers for post-backup
STOPPED_FILE="/tmp/resticm-stopped-containers.txt"
: > "$STOPPED_FILE"  # Clear file

for container in "${CONTAINERS[@]}"; do
    if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        echo "  Stopping $container..."
        docker stop "$container" --time 30
        echo "$container" >> "$STOPPED_FILE"
    else
        echo "  $container is not running, skipping"
    fi
done

echo "✅ Containers stopped: $(wc -l < "$STOPPED_FILE")"
exit 0
```

#### Post-Backup: Start Containers

```bash
#!/bin/bash
# /etc/resticm/hooks/post-backup.d/20-start-containers.sh

STOPPED_FILE="/tmp/resticm-stopped-containers.txt"

if [ ! -f "$STOPPED_FILE" ]; then
    echo "No containers to restart"
    exit 0
fi

echo "▶️  Starting containers after backup..."

while IFS= read -r container; do
    if [ -n "$container" ]; then
        echo "  Starting $container..."
        docker start "$container" || echo "  Warning: Failed to start $container"
    fi
done < "$STOPPED_FILE"

rm -f "$STOPPED_FILE"
echo "✅ Containers restarted"
exit 0
```

#### Post-Backup: Cleanup

```bash
#!/bin/bash
# /etc/resticm/hooks/post-backup.d/10-cleanup-dumps.sh

BACKUP_DIR="/var/backups/databases"
MAX_AGE_HOURS=2

echo "🧹 Cleaning up temporary dumps..."

# Only cleanup if backup was successful
if [ "$BACKUP_STATUS" = "success" ]; then
    # Remove dumps older than N hours
    find "$BACKUP_DIR" -name "*.sql.gz" -mmin +$((MAX_AGE_HOURS * 60)) -delete
    echo "✅ Cleanup completed"
else
    echo "⚠️  Backup failed, keeping dumps for investigation"
fi

exit 0
```

### Pattern 2: Monolithic Script

**Best for**: Simple setups, single server, few operations

All operations in one script:

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e

BACKUP_DIR="/var/backups"
LOG="/var/log/resticm/hooks.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

log "Starting pre-backup operations"

# 1. Dump MariaDB
log "Dumping MariaDB..."
mariadb-dump --all-databases | gzip > "$BACKUP_DIR/mariadb.sql.gz" || {
    log "ERROR: MariaDB dump failed"
    exit 1
}

# 2. Dump PostgreSQL
log "Dumping PostgreSQL..."
sudo -u postgres pg_dumpall | gzip > "$BACKUP_DIR/postgres.sql.gz" || {
    log "ERROR: PostgreSQL dump failed"
    exit 1
}

# 3. Stop containers
log "Stopping containers..."
docker stop webapp-prod redis-cache || {
    log "ERROR: Failed to stop containers"
    exit 1
}

# 4. Create application snapshot
log "Creating application snapshot..."
tar czf "$BACKUP_DIR/app-snapshot.tar.gz" /opt/myapp || {
    log "ERROR: Application snapshot failed"
    exit 1
}

log "✅ Pre-backup completed successfully"
exit 0
```

```bash
#!/bin/bash
# /etc/resticm/hooks/post-backup.sh

BACKUP_DIR="/var/backups"
LOG="/var/log/resticm/hooks.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

log "Starting post-backup cleanup"

# 1. Start containers
log "Starting containers..."
docker start redis-cache webapp-prod

# 2. Cleanup old dumps (keep only if backup succeeded)
if [ "$BACKUP_STATUS" = "success" ]; then
    log "Cleaning up temporary files..."
    rm -f "$BACKUP_DIR"/*.sql.gz
    rm -f "$BACKUP_DIR"/app-snapshot.tar.gz
else
    log "⚠️  Backup failed, keeping dumps for investigation"
fi

log "✅ Post-backup completed"
exit 0
```

### Pattern 3: Conditional Detection

**Best for**: Same script across multiple servers with different configurations

The script automatically detects what services are available:

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e

BACKUP_DIR="/var/backups/databases"
mkdir -p "$BACKUP_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "Starting intelligent pre-backup"

# Dump MariaDB if installed and running
if systemctl is-active --quiet mariadb 2>/dev/null; then
    log "📦 Dumping MariaDB..."
    mariadb-dump --all-databases | gzip > "$BACKUP_DIR/mariadb-$(date +%Y%m%d).sql.gz"
    log "✅ MariaDB dump completed"
fi

# Dump PostgreSQL if installed and running
if systemctl is-active --quiet postgresql 2>/dev/null; then
    log "📦 Dumping PostgreSQL..."
    sudo -u postgres pg_dumpall | gzip > "$BACKUP_DIR/postgres-$(date +%Y%m%d).sql.gz"
    log "✅ PostgreSQL dump completed"
fi

# Stop Docker containers labeled for backup
if command -v docker &> /dev/null; then
    log "⏸️  Stopping Docker containers..."
    
    # Stop containers with label backup-stop=true
    docker ps --filter "label=backup-stop=true" --format "{{.Names}}" | while read container; do
        log "  Stopping $container..."
        docker stop "$container" --time 30
        echo "$container" >> /tmp/resticm-stopped-containers.txt
    done
    
    stopped_count=$(wc -l < /tmp/resticm-stopped-containers.txt 2>/dev/null || echo 0)
    log "✅ Stopped $stopped_count containers"
fi

# Dump MongoDB if installed
if systemctl is-active --quiet mongod 2>/dev/null; then
    log "📦 Dumping MongoDB..."
    mongodump --archive="$BACKUP_DIR/mongodb-$(date +%Y%m%d).archive" --gzip
    log "✅ MongoDB dump completed"
fi

log "========================================="
log "✅ Pre-backup completed successfully"
log "========================================="
exit 0
```

**Docker labels for selective stopping**:

```yaml
# docker-compose.yml
services:
  webapp:
    image: myapp:latest
    labels:
      - "backup-stop=true"  # Will be stopped during backup
  
  redis:
    image: redis:latest
    labels:
      - "backup-stop=true"
  
  monitoring:
    image: prometheus:latest
    # No label - keeps running during backup
```

## Complete Examples

### Example 1: WordPress Site

Complete backup workflow for a WordPress site with MariaDB:

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e

BACKUP_DIR="/var/backups/wordpress"
WP_PATH="/var/www/wordpress"
DB_NAME="wordpress"

mkdir -p "$BACKUP_DIR"

echo "🌐 Preparing WordPress backup..."

# 1. Enable maintenance mode
sudo -u www-data wp --path="$WP_PATH" maintenance-mode activate || true

# 2. Dump database
echo "📦 Dumping WordPress database..."
mariadb-dump "$DB_NAME" | gzip > "$BACKUP_DIR/wordpress-db.sql.gz"

# 3. Export WP options
echo "⚙️  Exporting WordPress settings..."
sudo -u www-data wp --path="$WP_PATH" option list --format=json > "$BACKUP_DIR/wp-options.json"

# 4. List installed plugins
sudo -u www-data wp --path="$WP_PATH" plugin list --format=json > "$BACKUP_DIR/wp-plugins.json"

echo "✅ WordPress pre-backup completed"
exit 0
```

```bash
#!/bin/bash
# /etc/resticm/hooks/post-backup.sh

WP_PATH="/var/www/wordpress"
BACKUP_DIR="/var/backups/wordpress"

echo "🌐 WordPress post-backup cleanup..."

# Disable maintenance mode
sudo -u www-data wp --path="$WP_PATH" maintenance-mode deactivate || true

# Cleanup if backup succeeded
if [ "$BACKUP_STATUS" = "success" ]; then
    rm -f "$BACKUP_DIR"/*.sql.gz
    rm -f "$BACKUP_DIR"/*.json
    echo "✅ WordPress backup completed successfully"
else
    echo "⚠️  WordPress backup failed, files kept for debugging"
fi

exit 0
```

### Example 2: Docker Stack

Complete backup for a Docker stack with multiple services:

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e

BACKUP_DIR="/var/backups/docker"
COMPOSE_FILE="/opt/docker/docker-compose.yml"

mkdir -p "$BACKUP_DIR"

echo "🐳 Preparing Docker stack backup..."

# 1. Stop services gracefully
echo "⏸️  Stopping Docker stack..."
docker-compose -f "$COMPOSE_FILE" stop

# 2. Dump PostgreSQL from stopped container
echo "📦 Dumping PostgreSQL..."
docker-compose -f "$COMPOSE_FILE" run --rm --no-deps postgres \
    pg_dumpall -U postgres | gzip > "$BACKUP_DIR/postgres.sql.gz"

# 3. Export configs
echo "⚙️  Exporting Docker configs..."
docker-compose -f "$COMPOSE_FILE" config > "$BACKUP_DIR/docker-compose-resolved.yml"

# 4. List volumes
docker volume ls --format "{{.Name}}" | grep "^$(basename $(dirname $COMPOSE_FILE))_" \
    > "$BACKUP_DIR/volumes-list.txt"

echo "✅ Docker stack prepared for backup"
exit 0
```

```bash
#!/bin/bash
# /etc/resticm/hooks/post-backup.sh

COMPOSE_FILE="/opt/docker/docker-compose.yml"
BACKUP_DIR="/var/backups/docker"

echo "🐳 Restarting Docker stack..."

# Start services
docker-compose -f "$COMPOSE_FILE" start

# Cleanup if successful
if [ "$BACKUP_STATUS" = "success" ]; then
    rm -rf "$BACKUP_DIR"
    echo "✅ Docker stack backup completed"
else
    echo "⚠️  Backup failed, keeping dumps"
fi

exit 0
```

### Example 3: Multi-Database Server

Server running multiple database engines:

```bash
#!/bin/bash
# /etc/resticm/hooks/pre-backup.sh

set -e

BACKUP_DIR="/var/backups/databases"
DATE=$(date +%Y%m%d-%H%M%S)
LOG="/var/log/resticm/db-backup.log"

mkdir -p "$BACKUP_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

log "========================================="
log "Starting multi-database backup"
log "========================================="

# Function to backup and verify
backup_and_verify() {
    local name=$1
    local file=$2
    local min_size=$3  # Minimum expected size in MB
    
    if [ -f "$file" ]; then
        size=$(du -m "$file" | cut -f1)
        if [ "$size" -lt "$min_size" ]; then
            log "ERROR: $name dump too small (${size}MB < ${min_size}MB)"
            return 1
        fi
        log "✅ $name: ${size}MB"
        return 0
    else
        log "ERROR: $name dump not created"
        return 1
    fi
}

# MariaDB
if systemctl is-active --quiet mariadb; then
    log "📦 Backing up MariaDB..."
    mariadb-dump --all-databases --single-transaction --events --routines --triggers \
        | gzip > "$BACKUP_DIR/mariadb-$DATE.sql.gz"
    backup_and_verify "MariaDB" "$BACKUP_DIR/mariadb-$DATE.sql.gz" 1 || exit 1
fi

# PostgreSQL
if systemctl is-active --quiet postgresql; then
    log "📦 Backing up PostgreSQL..."
    sudo -u postgres pg_dumpall \
        | gzip > "$BACKUP_DIR/postgres-$DATE.sql.gz"
    backup_and_verify "PostgreSQL" "$BACKUP_DIR/postgres-$DATE.sql.gz" 1 || exit 1
fi

# MongoDB
if systemctl is-active --quiet mongod; then
    log "📦 Backing up MongoDB..."
    mongodump --archive="$BACKUP_DIR/mongodb-$DATE.archive" --gzip
    backup_and_verify "MongoDB" "$BACKUP_DIR/mongodb-$DATE.archive" 1 || exit 1
fi

# Redis
if systemctl is-active --quiet redis; then
    log "📦 Backing up Redis..."
    cp /var/lib/redis/dump.rdb "$BACKUP_DIR/redis-$DATE.rdb"
    gzip "$BACKUP_DIR/redis-$DATE.rdb"
    backup_and_verify "Redis" "$BACKUP_DIR/redis-$DATE.rdb.gz" 0 || exit 1
fi

log "========================================="
log "✅ All database backups completed"
log "========================================="

exit 0
```

## Best Practices

### 1. Always Use `set -e`

Fail fast on any error:

```bash
#!/bin/bash
set -e  # Exit immediately on error

# If any command fails, script stops
mariadb-dump mydb > backup.sql
gzip backup.sql
```

### 2. Verify Backups

Never trust dumps without verification:

```bash
# Verify file exists and is not empty
if [ ! -s /backup/db.sql.gz ]; then
    echo "ERROR: Dump file is empty or missing"
    exit 1
fi

# Verify minimum size
min_size=1048576  # 1MB in bytes
actual_size=$(stat -f%z /backup/db.sql.gz 2>/dev/null || stat -c%s /backup/db.sql.gz)
if [ "$actual_size" -lt "$min_size" ]; then
    echo "ERROR: Dump too small ($actual_size bytes)"
    exit 1
fi
```

### 3. Use Timeouts

Prevent hanging operations:

```bash
# Timeout after 5 minutes
timeout 300 mariadb-dump --all-databases > backup.sql || {
    echo "ERROR: Database dump timed out"
    exit 1
}
```

### 4. Log Everything

Maintain detailed logs for debugging:

```bash
LOG_FILE="/var/log/resticm/hooks.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Rotate logs
if [ -f "$LOG_FILE" ] && [ $(stat -f%z "$LOG_FILE" 2>/dev/null || stat -c%s "$LOG_FILE") -gt 10485760 ]; then
    mv "$LOG_FILE" "$LOG_FILE.old"
    gzip "$LOG_FILE.old"
fi
```

### 5. Handle Partial Failures

```bash
# Track failures but continue
FAILED=0

dump_mariadb || {
    echo "Warning: MariaDB dump failed"
    FAILED=$((FAILED + 1))
}

dump_postgres || {
    echo "Warning: PostgreSQL dump failed"
    FAILED=$((FAILED + 1))
}

if [ $FAILED -gt 0 ]; then
    echo "ERROR: $FAILED dumps failed"
    exit 1
fi
```

### 6. Use Descriptive Exit Messages

```bash
# Good: Clear error message
if ! docker stop webapp; then
    echo "ERROR: Failed to stop webapp container - backup aborted"
    exit 1
fi

# Bad: Silent failure
docker stop webapp || exit 1
```

### 7. Set Permissions Correctly

```bash
# Make scripts executable
chmod +x /etc/resticm/hooks/*.sh

# Restrict access
chmod 700 /etc/resticm/hooks/
chown root:root /etc/resticm/hooks/*.sh

# Verify before running
if [ ! -x /etc/resticm/hooks/pre-backup.sh ]; then
    echo "ERROR: Hook is not executable"
    exit 1
fi
```

### 8. Test Hooks Independently

```bash
# Test hook directly
sudo /etc/resticm/hooks/pre-backup.sh
echo "Exit code: $?"

# Test with resticm dry-run
sudo resticm backup --dry-run

# Test error handling
sudo resticm backup -v  # Verbose to see hook output
```

### 9. Handle Cleanup in pre_backup Failures

**⚠️ Critical**: Since `post_backup` doesn't run when `pre_backup` fails, you must handle cleanup in your `pre_backup` script or use `on_error` hook:

**Option 1: Cleanup in pre_backup on failure**
```bash
#!/bin/bash
# pre-backup.sh

set -e

STOPPED_CONTAINERS=()

# Function to restart containers on error
cleanup_on_error() {
    echo "ERROR: Cleaning up after failure..."
    for container in "${STOPPED_CONTAINERS[@]}"; do
        echo "Restarting $container..."
        docker start "$container" || true
    done
}

# Set trap to call cleanup on error
trap cleanup_on_error ERR EXIT

# Stop containers
for container in webapp redis; do
    if docker stop "$container"; then
        STOPPED_CONTAINERS+=("$container")
    fi
done

# Do your backup preparation
mariadb-dump mydb > /backup/dump.sql

# If we get here, disable the trap (success)
trap - ERR EXIT
exit 0
```

**Option 2: Cleanup in on_error hook**
```bash
#!/bin/bash
# on-error.sh

# Restart any stopped containers
STOPPED_FILE="/tmp/resticm-stopped-containers.txt"

if [ -f "$STOPPED_FILE" ]; then
    echo "ERROR detected - restarting stopped containers..."
    while IFS= read -r container; do
        docker start "$container" || true
    done < "$STOPPED_FILE"
    rm -f "$STOPPED_FILE"
fi

# Send alert
echo "Backup failed: $ERROR"
```

## Troubleshooting

### Hook Not Executing

**Symptom**: Hook is configured but doesn't run

**Solutions**:

```bash
# 1. Check if file exists
ls -la /etc/resticm/hooks/pre-backup.sh

# 2. Check permissions
# Should show: -rwx------ (700)
stat /etc/resticm/hooks/pre-backup.sh

# 3. Make executable
chmod +x /etc/resticm/hooks/pre-backup.sh

# 4. Check shebang
head -1 /etc/resticm/hooks/pre-backup.sh
# Should be: #!/bin/bash

# 5. Test directly
sudo /etc/resticm/hooks/pre-backup.sh
```

### Hook Fails Silently

**Symptom**: Hook exits with error but you don't see why

**Solution**: Add verbose logging

```bash
#!/bin/bash
set -e
set -x  # Print each command before executing

# Or use verbose logging
exec 1> >(logger -s -t $(basename $0)) 2>&1

# Rest of script...
```

### Backup Skipped After Hook Failure

**Symptom**: pre_backup fails, backup doesn't run

**This is expected behavior**. To fix:

1. Check hook logs: `/var/log/resticm/hooks.log`
2. Test hook manually: `sudo /etc/resticm/hooks/pre-backup.sh`
3. Fix the failing command
4. Consider making non-critical operations optional:

```bash
# Critical: will stop backup if fails
mariadb-dump mydb > backup.sql || exit 1

# Non-critical: will continue if fails
cleanup_temp_files || true
```

### Container Won't Stop

**Symptom**: `docker stop` times out

**Solution**: Increase grace period

```bash
# Default timeout is 10 seconds
docker stop webapp --time 30  # Wait 30 seconds

# Or force kill if necessary
docker stop webapp --time 30 || docker kill webapp
```

### Database Dump Too Large

**Symptom**: Dump takes forever or fills disk

**Solutions**:

```bash
# 1. Use compression
mariadb-dump mydb | gzip > backup.sql.gz

# 2. Exclude large tables
mariadb-dump mydb --ignore-table=mydb.logs > backup.sql

# 3. Check available space before dumping
available=$(df -BG /var/backups | tail -1 | awk '{print $4}' | sed 's/G//')
if [ "$available" -lt 10 ]; then
    echo "ERROR: Not enough disk space (${available}GB available)"
    exit 1
fi
```

### Hook Hangs Indefinitely

**Symptom**: Hook never completes

**Solution**: Use timeouts

```bash
# Timeout entire script after 10 minutes
timeout 600 /etc/resticm/hooks/pre-backup.sh

# Or timeout individual commands
timeout 300 mariadb-dump --all-databases > backup.sql || {
    echo "ERROR: Dump timed out after 5 minutes"
    exit 1
}
```

### Permission Denied Errors

**Symptom**: Hook can't access files or databases

**Solutions**:

```bash
# 1. Run as correct user
sudo -u postgres pg_dumpall > backup.sql

# 2. Check file permissions
ls -la /var/backups/

# 3. Ensure resticm runs as root
sudo resticm backup

# 4. Check SELinux/AppArmor
getenforce  # If Enforcing, might block operations
sudo ausearch -m avc -ts recent  # Check denials
```

---

## Summary

- **Use orchestrator pattern** for complex setups with multiple operations
- **Always handle errors** with proper exit codes and logging
- **Verify backups** before considering them successful
- **Test hooks independently** before running full backups
- **Monitor hook execution** through logs and notifications
- **Keep hooks modular** for easier maintenance and debugging

For more information, see:
- [Configuration Guide](../README.md#configuration)
- [Notification Setup](../config.example.yaml)
- [Systemd Integration](../README.md#systemd-timer)
