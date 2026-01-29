# Notifications Documentation

Complete guide to the notification system in resticm.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Supported Providers](#supported-providers)
  - [Slack](#slack)
  - [Discord](#discord)
  - [ntfy.sh](#ntfysh)
  - [Google Chat](#google-chat)
  - [Uptime Kuma](#uptime-kuma)
  - [Generic Webhook](#generic-webhook)
- [Notification Events](#notification-events)
- [Message Format](#message-format)
- [Advanced Usage](#advanced-usage)
  - [Multiple Providers](#multiple-providers)
  - [Command-Line Override](#command-line-override)
  - [Testing Notifications](#testing-notifications)
- [Provider-Specific Examples](#provider-specific-examples)
- [Troubleshooting](#troubleshooting)

---

## Overview

The notification system in resticm allows you to receive real-time alerts about backup operations, errors, and other events. This is particularly useful for:

- **Monitoring**: Know immediately when backups fail or succeed
- **Automation**: Integrate with incident management systems
- **Alerting**: Get notified on multiple channels (Slack, Discord, etc.)
- **Compliance**: Maintain audit trails of backup operations

### Key Features

- **Multiple providers**: Slack, Discord, ntfy.sh, Google Chat, Uptime Kuma, and generic webhooks
- **Flexible triggers**: Notify on success, error, or both
- **Rich messages**: Include hostname, repository, error details, and timestamps
- **Multiple destinations**: Send to multiple providers simultaneously
- **Command-line control**: Override configuration per command

---

## Quick Start

### 1. Enable Notifications

Add to your `config.yaml`:

```yaml
notifications:
  enabled: true
  notify_on_success: false  # Only notify on errors
  notify_on_error: true
  providers:
    - type: slack
      url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### 2. Test Notifications

Run a backup to trigger a notification:

```bash
resticm backup
```

### 3. Force Success Notification

Override the configuration to receive success notifications:

```bash
resticm backup --notify-success
```

---

## Configuration

### Global Settings

```yaml
notifications:
  # Enable/disable the entire notification system
  enabled: true

  # Send notifications on successful operations
  notify_on_success: false

  # Send notifications on errors (highly recommended)
  notify_on_error: true

  # List of notification providers (see below)
  providers: []
```

### Configuration Options

| Option               | Type    | Default | Description                                      |
|----------------------|---------|---------|--------------------------------------------------|
| `enabled`            | boolean | `false` | Enable/disable notifications globally            |
| `notify_on_success`  | boolean | `false` | Send notifications for successful operations     |
| `notify_on_error`    | boolean | `true`  | Send notifications for errors                    |
| `providers`          | array   | `[]`    | List of notification provider configurations     |

---

## Supported Providers

### Slack

Send notifications to Slack channels using incoming webhooks.

**Configuration:**

```yaml
providers:
  - type: slack
    url: "https://hooks.slack.com/services/YOUR_WORKSPACE_ID/YOUR_CHANNEL_ID/YOUR_WEBHOOK_TOKEN"
```

**Setup Steps:**

1. Go to your Slack workspace
2. Navigate to: https://api.slack.com/apps
3. Create a new app or select an existing one
4. Enable "Incoming Webhooks"
5. Click "Add New Webhook to Workspace"
6. Select the channel and authorize
7. Copy the webhook URL

**Message Format:**

- Color-coded attachments (green for success, red for error)
- Title and message body
- Structured fields for details (hostname, repository, etc.)
- Timestamp

**Example Notification:**

```
✅ Backup Completed
resticm backup completed successfully on prod-server-01

host: prod-server-01
repository: s3:s3.amazonaws.com/my-backup
```

---

### Discord

Send notifications to Discord channels using webhooks.

**Configuration:**

```yaml
providers:
  - type: discord
    url: "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
```

**Setup Steps:**

1. Open your Discord server
2. Go to Server Settings → Integrations → Webhooks
3. Click "New Webhook"
4. Name the webhook (e.g., "resticm Backup")
5. Select the channel
6. Copy the webhook URL

**Message Format:**

- Rich embeds with color coding
- Title and description
- Footer with "resticm" branding
- ISO 8601 timestamp

**Example:**

```json
{
  "embeds": [{
    "title": "✅ Backup Completed",
    "description": "resticm backup completed successfully on prod-server-01",
    "color": 3580527,
    "footer": { "text": "resticm" },
    "timestamp": "2026-01-28T10:30:00Z"
  }]
}
```

---

### ntfy.sh

Send push notifications using ntfy.sh (self-hosted or public service).

**Configuration:**

```yaml
providers:
  - type: ntfy
    url: "https://ntfy.sh"  # Or your self-hosted instance
    options:
      topic: "my-backup-alerts"
```

**Setup Steps:**

1. Choose ntfy.sh service:
   - Public: https://ntfy.sh
   - Self-hosted: Install ntfy server
2. Choose a unique topic name
3. Subscribe to the topic on your devices:
   - Mobile app: iOS/Android
   - Web: https://ntfy.sh/my-backup-alerts
   - CLI: `ntfy subscribe my-backup-alerts`

**Message Format:**

- Title in the notification header
- Body in the message content
- Priority: "default" for success, "high" for errors
- Tags: ✅ (white_check_mark) for success, ❌ (x) for errors

**Example Notification:**

```
Title: ✅ Backup Completed
Body: resticm backup completed successfully on prod-server-01
Priority: default
Tags: white_check_mark
```

**Benefits:**

- No account required for public service
- Works on iOS, Android, Desktop, Web
- Self-hosted option for privacy
- Simple HTTP API

---

### Google Chat

Send notifications to Google Chat spaces using webhooks.

**Configuration:**

```yaml
providers:
  - type: google
    url: "https://chat.googleapis.com/v1/spaces/SPACE_ID/messages?key=KEY&token=TOKEN"
```

**Aliases:** `googlechat`, `google_chat`

**Setup Steps:**

1. Open Google Chat
2. Go to the space where you want notifications
3. Click the space name → Apps & integrations
4. Click "Add webhooks"
5. Name the webhook (e.g., "resticm Backup")
6. Copy the webhook URL

**Message Format:**

- Card-based messages with header and sections
- Color-coded (via icon: ✅ or 🚨)
- Key-value pairs for details
- Structured widgets for body and details

**Example:**

```json
{
  "cards": [{
    "header": {
      "title": "✅ Backup Completed",
      "subtitle": "resticm backup"
    },
    "sections": [{
      "widgets": [
        { "textParagraph": { "text": "resticm backup completed successfully on prod-server-01" }},
        { "keyValue": { "topLabel": "host", "content": "prod-server-01" }},
        { "keyValue": { "topLabel": "repository", "content": "s3:..." }}
      ]
    }]
  }]
}
```

---

### Uptime Kuma

Send heartbeats to Uptime Kuma push monitors.

**Configuration:**

```yaml
providers:
  - type: uptimekuma
    url: "https://uptime.example.com/api/push/YOUR_PUSH_TOKEN"
```

**Aliases:** `uptime_kuma`, `uptime-kuma`

**Setup Steps:**

1. Access your Uptime Kuma instance
2. Create a new monitor
3. Select type: "Push"
4. Copy the push URL
5. Configure in resticm

**Message Format:**

- Simple GET request with query parameters
- `status=up` for success, `status=down` for errors
- `msg` parameter contains the message body

**Example Request:**

```
GET https://uptime.example.com/api/push/TOKEN?status=up&msg=OK
```

**Benefits:**

- Simple heartbeat monitoring
- Integrates with your existing monitoring infrastructure
- Status tracking over time
- Downtime alerts

---

### Generic Webhook

Send notifications to any HTTP endpoint that accepts JSON POST requests.

**Configuration:**

```yaml
providers:
  - type: webhook
    url: "https://example.com/api/webhook"
    options:
      Authorization: "Bearer YOUR_TOKEN"
      X-Custom-Header: "custom-value"
```

**Request Format:**

- Method: `POST`
- Content-Type: `application/json`
- Custom headers from `options`

**Payload Structure:**

```json
{
  "title": "✅ Backup Completed",
  "body": "resticm backup completed successfully on prod-server-01",
  "status": "success",
  "timestamp": "2026-01-28T10:30:00Z",
  "details": {
    "host": "prod-server-01",
    "repository": "s3:s3.amazonaws.com/my-backup"
  }
}
```

**Status Values:**

- `success`: Operation completed successfully
- `error`: Operation failed

**Example Configuration:**

```yaml
providers:
  - type: webhook
    url: "https://api.example.com/hooks/backup"
    options:
      Authorization: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
      X-Service-Name: "resticm"
      X-Environment: "production"
```

**Use Cases:**

- Custom monitoring systems
- Internal APIs
- Log aggregation services
- Incident management platforms (PagerDuty, Opsgenie)

---

## Notification Events

Notifications are triggered automatically by various resticm operations.

### When Notifications Are Sent

| Event                  | Success Notification | Error Notification | Details                                    |
|------------------------|----------------------|--------------------|---------------------------------------------|
| Backup completed       | ✅                   | ❌                 | Includes host and repository               |
| Backup failed          | ❌                   | ✅                 | Includes error message                     |
| Pre-backup hook failed | ❌                   | ✅                 | Hook failure prevents backup               |
| Full operation success | ✅                   | ❌                 | Backup + forget + prune                    |
| Full operation error   | ❌                   | ✅                 | Error at any stage                         |
| Prune failed           | ❌                   | ✅                 | Repository cleanup error                   |

### Notification Conditions

Notifications are sent based on:

1. **Global enabled flag**: `notifications.enabled: true`
2. **Event type settings**:
   - `notify_on_success: true/false`
   - `notify_on_error: true/false`
3. **Command-line override**: `--notify-success` flag

**Example Scenarios:**

```yaml
# Only notify on errors (recommended for production)
notifications:
  enabled: true
  notify_on_success: false
  notify_on_error: true

# Notify on both success and errors (verbose)
notifications:
  enabled: true
  notify_on_success: true
  notify_on_error: true

# Disabled (no notifications)
notifications:
  enabled: false
```

---

## Message Format

### Success Messages

**Title:**
```
✅ Backup Completed
✅ Full Backup Completed
```

**Body:**
```
resticm backup completed successfully on [hostname]
resticm full operation completed successfully on [hostname]
```

**Details:**
```yaml
host: prod-server-01
repository: s3:s3.amazonaws.com/my-backup
```

### Error Messages

**Title:**
```
❌ Backup Failed
❌ Pre-Backup Hook Failed
❌ Full Operation Failed
```

**Body:**
```
resticm backup failed on [hostname]: [error message]
resticm pre-backup hook failed on [hostname]: [error message]
```

**Details:**
```yaml
host: prod-server-01
repository: s3:s3.amazonaws.com/my-backup
error: exit status 1: connection timeout
```

### Message Components

| Component   | Description                                  | Example                                    |
|-------------|----------------------------------------------|--------------------------------------------|
| `title`     | Brief description with emoji                | "✅ Backup Completed"                      |
| `body`      | Detailed message                            | "resticm backup completed successfully..." |
| `status`    | Operation result                            | "success" or "error"                       |
| `timestamp` | ISO 8601 timestamp                          | "2026-01-28T10:30:00Z"                     |
| `details`   | Additional context (key-value pairs)        | {"host": "server01", "repository": "..."} |

---

## Advanced Usage

### Multiple Providers

Send notifications to multiple destinations simultaneously:

```yaml
notifications:
  enabled: true
  notify_on_success: false
  notify_on_error: true
  providers:
    # Alert team on Slack
    - type: slack
      url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
    
    # Log to Discord for audit trail
    - type: discord
      url: "https://discord.com/api/webhooks/YOUR/DISCORD/WEBHOOK"
    
    # Push notification to mobile devices
    - type: ntfy
      url: "https://ntfy.sh"
      options:
        topic: "prod-backup-alerts"
    
    # Update monitoring dashboard
    - type: uptimekuma
      url: "https://uptime.example.com/api/push/TOKEN"
```

**Behavior:**

- Notifications are sent to **all** configured providers
- Failures in one provider don't affect others
- Last error is returned if multiple providers fail
- Providers are processed sequentially

---

### Command-Line Override

Override configuration for specific commands:

```bash
# Force success notification even if notify_on_success is false
resticm backup --notify-success

# Regular backup (uses config settings)
resticm backup

# Full operation with success notification
resticm full --notify-success
```

**Use Cases:**

- Testing notifications
- One-off backups that require confirmation
- Critical backups where success confirmation is needed
- Scheduled jobs with different notification requirements

---

### Testing Notifications

#### 1. Test with Dry Run

```bash
# Dry run won't send notifications
resticm backup --dry-run
```

#### 2. Test with Success Override

```bash
# Force success notification
resticm backup --notify-success
```

#### 3. Test Individual Providers

Create a minimal test configuration:

```yaml
notifications:
  enabled: true
  notify_on_success: true
  notify_on_error: true
  providers:
    - type: slack  # Test only Slack
      url: "https://hooks.slack.com/services/TEST/WEBHOOK"
```

#### 4. Test Error Notifications

Trigger an error intentionally:

```bash
# Use invalid backend to trigger error
resticm backup --backend nonexistent-backend
```

#### 5. Verify Webhook URLs

Use `curl` to test webhook endpoints directly:

```bash
# Test Slack webhook
curl -X POST -H 'Content-Type: application/json' \
  -d '{"text":"Test from resticm"}' \
  https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Test ntfy
curl -H "Title: Test" -d "Test from resticm" https://ntfy.sh/my-topic
```

---

## Provider-Specific Examples

### Complete Slack Setup

```yaml
notifications:
  enabled: true
  notify_on_success: false
  notify_on_error: true
  providers:
    - type: slack
      url: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX"
```

**Expected Result:**

When a backup fails, Slack receives:

```
[RED ATTACHMENT]
❌ Backup Failed

resticm backup failed on prod-web-01: connection timeout

host: prod-web-01
repository: s3:s3.amazonaws.com/prod-backups
error: exit status 1

resticm
1738062000 (timestamp)
```

---

### ntfy.sh with Custom Server

```yaml
notifications:
  enabled: true
  notify_on_success: true
  notify_on_error: true
  providers:
    - type: ntfy
      url: "https://ntfy.example.com"  # Self-hosted
      options:
        topic: "backup-prod-alerts"
```

**Subscribe to notifications:**

```bash
# Mobile app
Open ntfy app → Add subscription → "ntfy.example.com/backup-prod-alerts"

# Command line
ntfy subscribe https://ntfy.example.com/backup-prod-alerts

# Web browser
https://ntfy.example.com/backup-prod-alerts
```

---

### Multiple Environments

Different providers per environment:

```yaml
# Production: production.yaml
notifications:
  enabled: true
  notify_on_error: true
  providers:
    - type: slack
      url: "https://hooks.slack.com/services/PROD/SLACK/WEBHOOK"
    - type: uptimekuma
      url: "https://uptime.example.com/api/push/PROD_TOKEN"

# Staging: staging.yaml
notifications:
  enabled: true
  notify_on_success: true  # Verbose for testing
  notify_on_error: true
  providers:
    - type: discord
      url: "https://discord.com/api/webhooks/STAGING/WEBHOOK"
```

---

### Integration with Monitoring Stack

```yaml
notifications:
  enabled: true
  notify_on_success: true
  notify_on_error: true
  providers:
    # Incident management
    - type: webhook
      url: "https://api.pagerduty.com/events/v2/enqueue"
      options:
        Authorization: "Token token=YOUR_API_KEY"
    
    # Metrics and logs
    - type: webhook
      url: "https://logs.example.com/api/v1/ingest"
      options:
        X-API-Key: "YOUR_LOG_API_KEY"
    
    # Team communication
    - type: slack
      url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
    
    # Health checks
    - type: uptimekuma
      url: "https://uptime.example.com/api/push/TOKEN"
```

---

## Troubleshooting

### Notifications Not Sending

**Check 1: Notifications Enabled**

```yaml
notifications:
  enabled: true  # Must be true
```

**Check 2: Event Type Configuration**

```yaml
notifications:
  notify_on_success: true  # For success notifications
  notify_on_error: true    # For error notifications
```

**Check 3: Providers Configured**

```yaml
notifications:
  providers:
    - type: slack  # At least one provider required
      url: "..."
```

**Check 4: Test Manually**

```bash
# Force success notification
resticm backup --notify-success
```

---

### Webhook Errors

**Error: "webhook returned status 400"**

**Cause:** Invalid payload format or missing required fields

**Solution:**

1. Check provider documentation for required format
2. Test webhook URL with `curl`:

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"text":"test"}' \
  YOUR_WEBHOOK_URL
```

3. Verify URL is correct (no typos, valid token)

---

**Error: "webhook returned status 401" or "403"**

**Cause:** Authentication failure

**Solution:**

1. Verify webhook token/URL is correct
2. Regenerate webhook in provider settings
3. Check webhook hasn't expired (Discord webhooks can expire)
4. Verify custom headers are correct (for generic webhooks)

---

**Error: "connection timeout" or "dial tcp: i/o timeout"**

**Cause:** Network connectivity issues

**Solution:**

1. Check firewall rules
2. Verify DNS resolution: `nslookup hooks.slack.com`
3. Test connectivity: `curl -I https://hooks.slack.com`
4. Check proxy settings if behind corporate firewall

---

### Provider-Specific Issues

#### Slack

**Problem:** Messages not appearing in channel

- Verify webhook is for correct workspace/channel
- Check channel permissions
- Regenerate webhook if channel was recreated

#### Discord

**Problem:** Webhook expired

- Discord webhooks can expire if not used
- Regenerate webhook in Discord server settings
- Update URL in resticm configuration

#### ntfy.sh

**Problem:** Not receiving push notifications

- Verify topic name matches exactly
- Check subscription is active in ntfy app
- Test topic: `curl -d "test" https://ntfy.sh/YOUR_TOPIC`
- For self-hosted: verify server is accessible

#### Uptime Kuma

**Problem:** Monitor shows "down" incorrectly

- Verify push token is correct
- Check if resticm is sending notifications (enable success notifications)
- Verify heartbeat interval in Uptime Kuma matches backup frequency
- Test push URL: `curl "https://uptime.example.com/api/push/TOKEN?status=up&msg=test"`

---

### Debugging

Enable verbose logging to see notification details:

```bash
resticm backup --verbose
```

Check logs for notification errors:

```bash
# If logging to file
tail -f /var/log/resticm/resticm.log | grep -i notif

# System journal
journalctl -u resticm -f | grep -i notif
```

**Common log messages:**

```
# Success
INFO  Notification sent successfully via slack

# Error
ERROR Failed to send notification via slack: webhook returned status 400

# Disabled
DEBUG Notifications disabled, skipping
DEBUG Notify on success disabled, skipping
```

---

### Best Practices

1. **Start Simple**
   - Begin with one provider (e.g., Slack or ntfy.sh)
   - Test with `--notify-success` flag
   - Gradually add more providers

2. **Error Notifications Only (Production)**
   ```yaml
   notifications:
     enabled: true
     notify_on_success: false  # Reduce noise
     notify_on_error: true     # Critical alerts
   ```

3. **Success Notifications (Testing/Development)**
   ```yaml
   notifications:
     enabled: true
     notify_on_success: true   # Confirm backups work
     notify_on_error: true
   ```

4. **Multiple Channels**
   - Errors → Slack (team alerts)
   - All events → Discord (audit log)
   - Heartbeat → Uptime Kuma (monitoring)
   - Metrics → Webhook (analytics)

5. **Secure Webhooks**
   - Never commit webhook URLs to version control
   - Use environment variables or secret management
   - Regenerate webhooks periodically
   - Restrict webhook permissions in provider settings

6. **Test Regularly**
   ```bash
   # Quarterly test
   resticm backup --notify-success
   
   # Verify error notifications
   resticm backup --backend invalid-backend
   ```

7. **Monitor Notification Delivery**
   - Check provider status pages
   - Set up secondary alerting
   - Review logs regularly
   - Test backup alert procedures

---

## Architecture Notes

### How Notifications Work

1. **Initialization**: `GetNotifier()` creates notifier from configuration
2. **Event Trigger**: Command operations call `NotifySuccess()` or `NotifyError()`
3. **Provider Loop**: Message sent to all configured providers
4. **Error Handling**: Provider failures logged, don't stop other providers

### Code Structure

```
internal/notify/
├── notify.go           # Core notification logic
├── notify_test.go      # Unit tests
└── providers:
    ├── SlackProvider
    ├── DiscordProvider
    ├── NtfyProvider
    ├── GoogleChatProvider
    ├── UptimeKumaProvider
    └── WebhookProvider
```

### Provider Interface

All providers implement:

```go
type Provider interface {
    Send(message *Message) error
    Name() string
}
```

### Message Structure

```go
type Message struct {
    Title     string            // Brief description
    Body      string            // Detailed message
    Status    string            // "success" or "error"
    Timestamp time.Time         // When event occurred
    Details   map[string]string // Additional context
}
```

### Configuration Flow

```
config.yaml
    ↓
Config.Notifications (internal/config)
    ↓
GetNotifier() (cmd/root.go)
    ↓
notify.NewNotifier() (internal/notify)
    ↓
createProvider() for each provider
    ↓
Notifier with []Provider
```

### Notification Flow

```
Command execution
    ↓
Success or Error
    ↓
notifier.NotifySuccess() or NotifyError()
    ↓
Check if enabled + event type enabled
    ↓
Build Message
    ↓
For each provider:
    provider.Send(message)
        ↓
    HTTP POST / GET to provider endpoint
        ↓
    Log success or error
```

---

## FAQ

**Q: Can I send notifications to multiple Slack channels?**

A: Yes, add multiple Slack providers with different webhook URLs:

```yaml
providers:
  - type: slack
    url: "https://hooks.slack.com/services/.../channel1"
  - type: slack
    url: "https://hooks.slack.com/services/.../channel2"
```

---

**Q: How do I temporarily disable notifications?**

A: Set `enabled: false` in configuration, or remove the `--notify-success` flag.

---

**Q: Can I use environment variables for webhook URLs?**

A: Yes, resticm supports environment variable expansion in configuration:

```yaml
providers:
  - type: slack
    url: "${SLACK_WEBHOOK_URL}"
```

Then:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
resticm backup
```

---

**Q: What happens if a provider fails?**

A: resticm logs the error and continues with other providers. The backup operation is not affected by notification failures.

---

**Q: Can I customize the message format?**

A: Currently, message format is fixed per provider. For custom formats, use the `webhook` provider and process the JSON payload on your server.

---

**Q: How do I know if notifications are working?**

A: Use the `--notify-success` flag to force a success notification:

```bash
resticm backup --notify-success
```

---

**Q: Can I get notifications for specific commands only?**

A: Currently, notification configuration is global. Use the `--notify-success` flag on specific commands, or disable/enable notifications in the configuration per environment.

---

**Q: Is there a rate limit?**

A: resticm has no built-in rate limiting, but providers may have limits:
- Slack: ~1 message per second
- Discord: ~5 messages per 5 seconds
- ntfy.sh: Depends on server configuration

---

**Q: Can I send notifications via email?**

A: Not directly, but you can:
1. Use ntfy.sh with email forwarding: https://docs.ntfy.sh/config/#e-mail-notifications
2. Use a webhook provider to call an email API
3. Use Slack/Discord email integration features

---

## Additional Resources

### Official Documentation

- **Slack Webhooks**: https://api.slack.com/messaging/webhooks
- **Discord Webhooks**: https://discord.com/developers/docs/resources/webhook
- **ntfy.sh**: https://docs.ntfy.sh/
- **Google Chat Webhooks**: https://developers.google.com/chat/how-tos/webhooks
- **Uptime Kuma**: https://github.com/louislam/uptime-kuma

### Related resticm Documentation

- [Hooks Documentation](./hooks.md) - Pre/post backup scripts
- [Configuration Guide](../README.md#configuration) - Complete config reference
- [Logging Documentation](./logging.md) - Log file configuration

### Community

- Report issues: https://github.com/Gu1llaum-3/resticm/issues
- Discussions: https://github.com/Gu1llaum-3/resticm/discussions

---

## Changelog

### v0.4.0
- Initial notification system implementation
- Support for Slack, Discord, ntfy.sh, Google Chat, Uptime Kuma, and generic webhooks
- Per-command `--notify-success` flag
- Multiple provider support
- Comprehensive error handling

---

*Last updated: January 28, 2026*
