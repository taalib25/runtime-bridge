# Hermes Agent - Gateway Configuration Guide

Complete reference for gateway security, allowlists, user management, and platform-specific settings.

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## Table of Contents

1. [Gateway Overview](#gateway-overview)
2. [Security Configuration](#security-configuration)
3. [Allowlist Management](#allowlist-management)
4. [User Management](#user-management)
5. [DM Pairing System](#dm-pairing-system)
6. [Rate Limiting](#rate-limiting)
7. [Webhook Configuration](#webhook-configuration)
8. [Platform-Specific Settings](#platform-specific-settings)
9. [Streaming Configuration](#streaming-configuration)
10. [Error Handling](#error-handling)
11. [Monitoring & Logging](#monitoring--logging)

---

## Gateway Overview

### What is a Gateway?

A gateway is a connection to a messaging platform (Telegram, Discord, Slack, etc.) that allows Hermes Agent to:
- Receive messages from users
- Send responses
- Handle media (images, files, voice)
- Manage user sessions
- Enforce security policies

### Gateway Architecture

```
User → Platform → Gateway → Hermes Agent → Response → Platform → User
                    ↓
              Allowlist Check
              Rate Limiting
              DM Pairing
              Streaming
```

### Supported Gateways

| Gateway | Type | Status | Polling | Webhook | DM Pairing |
|---------|------|--------|---------|---------|-----------|
| Telegram | Chat | ✓ | ✓ | ✓ | ✓ |
| Discord | Chat | ✓ | ✗ | ✓ | ✓ |
| Slack | Chat | ✓ | ✗ | ✓ | ✓ |
| WhatsApp | Chat | ✓ | ✗ | ✓ | ✓ |
| Signal | Chat | ✓ | ✗ | ✓ | ✓ |
| Email | Async | ✓ | ✓ | ✓ | ✗ |
| SMS | Chat | ✓ | ✗ | ✓ | ✗ |
| Matrix | Chat | ✓ | ✓ | ✓ | ✓ |
| Mattermost | Chat | ✓ | ✗ | ✓ | ✓ |
| Feishu/Lark | Chat | ✓ | ✗ | ✓ | ✓ |
| WeCom | Chat | ✓ | ✗ | ✓ | ✓ |
| Weixin | Chat | ✓ | ✗ | ✓ | ✓ |
| DingTalk | Chat | ✓ | ✗ | ✓ | ✓ |
| BlueBubbles | Chat | ✓ | ✗ | ✓ | ✓ |
| QQ Bot | Chat | ✓ | ✗ | ✓ | ✓ |
| Home Assistant | Automation | ✓ | ✗ | ✓ | ✗ |

---

## Security Configuration

### Global Security Settings

**config.yaml**:
```yaml
security:
  enabled: true
  require_auth: true
  
  # IP-based access control
  allowed_ips: []           # Empty = all IPs allowed
  blocked_ips:
    - 192.168.1.100
    - 10.0.0.0/8
  
  # SSL/TLS
  ssl:
    enabled: false
    cert_path: /path/to/cert.pem
    key_path: /path/to/key.pem
  
  # Rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60
  
  # Token rotation
  token_rotation:
    enabled: true
    rotation_interval: 86400  # 24 hours
```

**Environment Variables**:
```bash
SECURITY_ENABLED=true
SECURITY_REQUIRE_AUTH=true
SECURITY_ALLOWED_IPS=
SECURITY_BLOCKED_IPS=192.168.1.100,10.0.0.0/8
SECURITY_SSL_ENABLED=false
SECURITY_RATE_LIMIT_ENABLED=true
SECURITY_RATE_LIMIT_RPM=60
```

### Per-Gateway Security

**config.yaml**:
```yaml
gateways:
  telegram:
    security:
      enabled: true
      require_auth: true
      token_validation: true
      webhook_secret: "${TELEGRAM_WEBHOOK_SECRET}"
      ip_whitelist:
        - 149.154.160.0/20  # Telegram IP range
      rate_limit:
        enabled: true
        requests_per_minute: 30
```

---

## Allowlist Management

### User Allowlists

Allow only specific users to interact with Hermes Agent.

**config.yaml**:
```yaml
gateways:
  telegram:
    allowed_users:
      - 123456789
      - 987654321
      - 555555555
    blocked_users:
      - 111111111
    allow_all_users: false
```

**Environment Variables**:
```bash
TELEGRAM_ALLOWED_USERS=123456789,987654321,555555555
TELEGRAM_BLOCKED_USERS=111111111
TELEGRAM_ALLOW_ALL_USERS=false
```

### Channel/Room Allowlists

Allow only specific channels or rooms.

**config.yaml**:
```yaml
gateways:
  discord:
    allowed_channels:
      - 987654321098765432
      - 123456789012345678
    blocked_channels:
      - 555555555555555555
    allow_all_channels: false
```

**Environment Variables**:
```bash
DISCORD_ALLOWED_CHANNELS=987654321098765432,123456789012345678
DISCORD_BLOCKED_CHANNELS=555555555555555555
DISCORD_ALLOW_ALL_CHANNELS=false
```

### Server/Workspace Allowlists

Allow only specific servers or workspaces.

**config.yaml**:
```yaml
gateways:
  slack:
    allowed_workspaces:
      - workspace-id-1
      - workspace-id-2
    blocked_workspaces:
      - workspace-id-3
    allow_all_workspaces: false
```

**Environment Variables**:
```bash
SLACK_ALLOWED_WORKSPACES=workspace-id-1,workspace-id-2
SLACK_BLOCKED_WORKSPACES=workspace-id-3
SLACK_ALLOW_ALL_WORKSPACES=false
```

### Allowlist Priority

The allowlist system uses the following priority:

1. **Blocked list** (highest priority) - Always denied
2. **Allowed list** - Only allowed if in list
3. **Allow all** (lowest priority) - Allow if not blocked

**Logic**:
```
if user in blocked_list:
    DENY
elif allow_all:
    ALLOW
elif user in allowed_list:
    ALLOW
else:
    DENY
```

### Dynamic Allowlist Updates

Update allowlists without restarting:

```bash
# Reload configuration
hermes config reload

# Or via API
curl -X POST http://localhost:8642/admin/reload-config \
  -H "Authorization: Bearer your-api-key"
```

---

## User Management

### Admin Users

Designate users with elevated privileges.

**config.yaml**:
```yaml
gateways:
  telegram:
    admin_users:
      - 123456789
      - 987654321
```

**Environment Variables**:
```bash
TELEGRAM_ADMIN_USERS=123456789,987654321
```

### Admin Capabilities

Admin users can:
- Execute privileged commands
- Manage other users
- View system status
- Access logs
- Reload configuration

### Admin Commands

```
/admin status          - Show system status
/admin users           - List all users
/admin block <id>      - Block user
/admin unblock <id>    - Unblock user
/admin reload          - Reload configuration
/admin logs            - View recent logs
```

### User Roles

**config.yaml**:
```yaml
user_roles:
  admin:
    - 123456789
  moderator:
    - 987654321
    - 555555555
  user:
    - all_others
```

---

## DM Pairing System

### Overview

The DM pairing system allows users to securely pair their accounts with Hermes Agent, enabling automatic message routing.

### How It Works

1. **User initiates pairing**: Sends `/pair` command
2. **Bot generates code**: Creates unique pairing code
3. **User confirms**: User confirms pairing in DM
4. **Bot remembers**: Stores pairing for future messages
5. **Automatic routing**: Future messages automatically routed

### Configuration

**config.yaml**:
```yaml
dm_pairing:
  enabled: true
  timeout: 300              # Seconds before pairing expires
  code_length: 6            # Pairing code length
  code_format: "alphanumeric"  # alphanumeric, numeric, or hex
  max_attempts: 3           # Max confirmation attempts
  storage: "file"           # file or redis
  storage_path: "~/.hermes/pairings"
  
  # Per-gateway settings
  gateways:
    telegram:
      enabled: true
      timeout: 300
    discord:
      enabled: true
      timeout: 300
```

**Environment Variables**:
```bash
DM_PAIRING_ENABLED=true
DM_PAIRING_TIMEOUT=300
DM_PAIRING_CODE_LENGTH=6
DM_PAIRING_CODE_FORMAT=alphanumeric
DM_PAIRING_MAX_ATTEMPTS=3
DM_PAIRING_STORAGE=file
DM_PAIRING_STORAGE_PATH=~/.hermes/pairings
```

### Pairing Flow

```
User: /pair
Bot: Your pairing code is: ABC123
     Confirm pairing? (Y/N)
     Code expires in 5 minutes

User: Y
Bot: ✓ Pairing confirmed!
     You can now chat with me in any channel.

User: (in any channel) @Hermes hello
Bot: Hello! I recognized you from pairing.
```

### Pairing Storage

**File-based storage** (~/.hermes/pairings/):
```json
{
  "telegram:123456789": {
    "user_id": "123456789",
    "platform": "telegram",
    "paired_at": "2026-04-18T10:30:00Z",
    "expires_at": "2026-04-18T10:35:00Z",
    "verified": true
  }
}
```

**Redis-based storage**:
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=
```

### Pairing Management

**List pairings**:
```bash
hermes admin pairings list
```

**Remove pairing**:
```bash
hermes admin pairings remove <user_id>
```

**Clear all pairings**:
```bash
hermes admin pairings clear
```

---

## Rate Limiting

### Global Rate Limiting

**config.yaml**:
```yaml
rate_limit:
  enabled: true
  requests_per_minute: 60
  burst_size: 10
  
  # Per-user limits
  per_user:
    enabled: true
    requests_per_minute: 30
  
  # Per-gateway limits
  per_gateway:
    enabled: true
    requests_per_minute: 100
```

**Environment Variables**:
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPM=60
RATE_LIMIT_BURST_SIZE=10
RATE_LIMIT_PER_USER_ENABLED=true
RATE_LIMIT_PER_USER_RPM=30
RATE_LIMIT_PER_GATEWAY_ENABLED=true
RATE_LIMIT_PER_GATEWAY_RPM=100
```

### Per-Gateway Rate Limiting

**config.yaml**:
```yaml
gateways:
  telegram:
    rate_limit:
      enabled: true
      requests_per_minute: 30
      burst_size: 5
  
  discord:
    rate_limit:
      enabled: true
      requests_per_minute: 60
      burst_size: 10
  
  slack:
    rate_limit:
      enabled: true
      requests_per_minute: 60
      burst_size: 10
```

### Rate Limit Headers

API responses include rate limit information:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1713436200
```

### Rate Limit Errors

When rate limit is exceeded:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Try again in 30 seconds.",
  "retry_after": 30
}
```

---

## Webhook Configuration

### Webhook Setup

**config.yaml**:
```yaml
webhook:
  enabled: true
  port: 8644
  host: 0.0.0.0
  secret: "${WEBHOOK_SECRET}"
  
  # SSL/TLS
  ssl:
    enabled: false
    cert_path: /path/to/cert.pem
    key_path: /path/to/key.pem
  
  # Routes
  routes:
    - path: /telegram
      method: POST
      events:
        - message
        - callback_query
      template: telegram
      delivery:
        retry: 3
        timeout: 30
    
    - path: /discord
      method: POST
      events:
        - message
        - reaction
      template: discord
      delivery:
        retry: 3
        timeout: 30
```

**Environment Variables**:
```bash
WEBHOOK_ENABLED=true
WEBHOOK_PORT=8644
WEBHOOK_HOST=0.0.0.0
WEBHOOK_SECRET=your-webhook-secret
WEBHOOK_SSL_ENABLED=false
```

### Webhook Events

Supported events:
- `message` - New message
- `callback_query` - Button click
- `reaction` - Emoji reaction
- `status` - Delivery status
- `error` - Error notification

### Webhook Delivery

**Retry policy**:
```yaml
delivery:
  retry: 3              # Max retries
  backoff: exponential  # exponential or linear
  timeout: 30           # Seconds
```

**Retry delays**:
- Attempt 1: Immediate
- Attempt 2: 5 seconds
- Attempt 3: 25 seconds

### Webhook Signature Verification

All webhooks are signed with HMAC-SHA256:

```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

---

## Platform-Specific Settings

### Telegram

**config.yaml**:
```yaml
gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
    
    # Polling vs Webhook
    webhook_enabled: false
    webhook_url: "https://your-domain.com/telegram"
    webhook_port: 8443
    polling_interval: 1
    
    # Allowlists
    allowed_users:
      - 123456789
    blocked_users: []
    allow_all_users: false
    
    # DM Pairing
    dm_pairing: true
    dm_pairing_timeout: 300
    
    # Streaming
    streaming_enabled: true
    streaming_chunk_size: 1000
    streaming_chunk_delay: 100
    
    # Rate limiting
    rate_limit:
      enabled: true
      requests_per_minute: 30
```

### Discord

**config.yaml**:
```yaml
gateways:
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
    
    # Allowlists
    allowed_servers:
      - 123456789012345678
    allowed_channels:
      - 987654321098765432
    blocked_channels: []
    allow_all_channels: false
    
    # Admin users
    admin_users:
      - 111111111111111111
    
    # DM Pairing
    dm_pairing: true
    dm_pairing_timeout: 300
    
    # Streaming
    streaming_enabled: true
    streaming_chunk_size: 1000
    streaming_edit_interval: 2000
    
    # Thread replies
    thread_replies: true
    
    # Rate limiting
    rate_limit:
      enabled: true
      requests_per_minute: 60
```

### Slack

**config.yaml**:
```yaml
gateways:
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"
    app_token: "${SLACK_APP_TOKEN}"
    
    # Allowlists
    allowed_channels:
      - general
      - hermes-bot
    blocked_channels: []
    allow_all_channels: false
    
    # Admin users
    admin_users:
      - U123456789
    
    # DM Pairing
    dm_pairing: true
    dm_pairing_timeout: 300
    
    # Streaming
    streaming_enabled: true
    streaming_chunk_size: 1000
    
    # Thread replies
    thread_replies: true
    
    # Rate limiting
    rate_limit:
      enabled: true
      requests_per_minute: 60
```

---

## Streaming Configuration

### Global Streaming Settings

**config.yaml**:
```yaml
streaming:
  enabled: true
  default_chunk_size: 1000
  default_chunk_delay: 100
  
  # Per-platform settings
  platforms:
    telegram:
      enabled: true
      chunk_size: 1000
      chunk_delay: 100
      edit_interval: 2000
    
    discord:
      enabled: true
      chunk_size: 1000
      chunk_delay: 100
      edit_interval: 2000
    
    slack:
      enabled: true
      chunk_size: 1000
      chunk_delay: 100
      edit_interval: 2000
```

**Environment Variables**:
```bash
STREAMING_ENABLED=true
STREAMING_DEFAULT_CHUNK_SIZE=1000
STREAMING_DEFAULT_CHUNK_DELAY=100
STREAMING_TELEGRAM_ENABLED=true
STREAMING_DISCORD_ENABLED=true
STREAMING_SLACK_ENABLED=true
```

### Streaming Behavior

- **Chunk size**: Characters per message chunk
- **Chunk delay**: Milliseconds between chunks
- **Edit interval**: Milliseconds between message edits

### Streaming Example

```
User: Tell me a long story
Bot: [Streaming response...]
     Once upon a time...
     [Edit 1] Once upon a time, there was a...
     [Edit 2] Once upon a time, there was a kingdom...
     [Final] Once upon a time, there was a kingdom far away...
```

---

## Error Handling

### Error Configuration

**config.yaml**:
```yaml
error_handling:
  enabled: true
  
  # Error responses
  send_error_messages: true
  error_message_template: "Sorry, I encountered an error: {error}"
  
  # Error logging
  log_errors: true
  log_level: ERROR
  
  # Error recovery
  auto_retry: true
  max_retries: 3
  retry_delay: 5
```

### Error Types

| Error | Cause | Recovery |
|-------|-------|----------|
| `auth_failed` | Invalid token | Restart with new token |
| `rate_limit_exceeded` | Too many requests | Wait and retry |
| `message_too_long` | Message exceeds limit | Split message |
| `invalid_format` | Unsupported format | Convert format |
| `connection_lost` | Network error | Auto-reconnect |
| `timeout` | Request timeout | Retry |

### Error Responses

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "retry_after": 30,
  "timestamp": "2026-04-18T10:30:00Z"
}
```

---

## Monitoring & Logging

### Logging Configuration

**config.yaml**:
```yaml
logging:
  level: INFO
  format: json
  file: ~/.hermes/logs/hermes.log
  max_size: 100
  backup_count: 5
  console: true
  
  # Per-gateway logging
  gateways:
    telegram:
      level: DEBUG
    discord:
      level: INFO
```

**Environment Variables**:
```bash
LOG_LEVEL=INFO
LOG_FORMAT=json
LOG_FILE=~/.hermes/logs/hermes.log
LOG_MAX_SIZE_MB=100
LOG_BACKUP_COUNT=5
LOG_CONSOLE=true
```

### Log Levels

- `DEBUG` - Detailed information
- `INFO` - General information
- `WARNING` - Warning messages
- `ERROR` - Error messages
- `CRITICAL` - Critical errors

### Monitoring Metrics

**config.yaml**:
```yaml
monitoring:
  enabled: true
  
  # Prometheus metrics
  prometheus:
    enabled: true
    port: 9090
  
  # Metrics to track
  metrics:
    - messages_received
    - messages_sent
    - errors_total
    - response_time
    - rate_limit_hits
```

### Health Checks

```bash
# API health check
curl http://localhost:8642/health

# Gateway health check
curl http://localhost:8642/health/telegram
curl http://localhost:8642/health/discord
```

---

## Best Practices

1. **Use allowlists** - Restrict access to trusted users
2. **Enable rate limiting** - Prevent abuse
3. **Use webhooks** - More efficient than polling
4. **Enable streaming** - Better UX for long responses
5. **Monitor logs** - Catch issues early
6. **Rotate tokens** - Security best practice
7. **Use separate bots** - Dev, staging, production
8. **Enable DM pairing** - Secure user verification
9. **Test configuration** - Before production
10. **Document setup** - For team collaboration

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+
