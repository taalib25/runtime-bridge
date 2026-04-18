# Hermes Agent Environment Variables Reference

Complete reference for all 60+ environment variables organized by category.

## Location

```
~/.hermes/.env
```

## Quick Reference Table

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `OPENROUTER_API_KEY` | string | - | OpenRouter API key |
| `OPENAI_API_KEY` | string | - | OpenAI API key |
| `ANTHROPIC_API_KEY` | string | - | Anthropic API key |
| `MODEL` | string | `anthropic/claude-opus-4.6` | Default model |
| `BASE_URL` | string | `https://openrouter.ai/api/v1` | LLM API base URL |
| `TEMPERATURE` | float | `0.7` | Model temperature (0.0-2.0) |
| `MAX_TOKENS` | int | `4096` | Max tokens per response |
| `API_SERVER_KEY` | string | - | API server authentication key |
| `API_SERVER_PORT` | int | `8642` | API server port |
| `TELEGRAM_BOT_TOKEN` | string | - | Telegram bot token |
| `DISCORD_BOT_TOKEN` | string | - | Discord bot token |
| `SLACK_BOT_TOKEN` | string | - | Slack bot token |

## Detailed Reference

### LLM Provider Configuration

#### OpenRouter
```bash
OPENROUTER_API_KEY=sk-or-YOUR_KEY
MODEL=anthropic/claude-opus-4.6
BASE_URL=https://openrouter.ai/api/v1
```

#### OpenAI
```bash
OPENAI_API_KEY=sk-YOUR_KEY
MODEL=gpt-4
BASE_URL=https://api.openai.com/v1
```

#### Anthropic
```bash
ANTHROPIC_API_KEY=sk-ant-YOUR_KEY
MODEL=claude-opus-4.6
BASE_URL=https://api.anthropic.com
```

#### Local LLM (Ollama)
```bash
MODEL=llama2
BASE_URL=http://localhost:11434/v1
```

### Model Configuration

```bash
# Model selection
MODEL=anthropic/claude-opus-4.6

# Model parameters
TEMPERATURE=0.7              # 0.0 (deterministic) to 2.0 (creative)
TOP_P=0.9                    # Nucleus sampling
TOP_K=40                     # Top-k sampling
MAX_TOKENS=4096              # Max response length
FREQUENCY_PENALTY=0.0        # Reduce repetition
PRESENCE_PENALTY=0.0         # Encourage new topics
```

### API Server Configuration

```bash
# Enable API server
API_SERVER_ENABLED=true

# API server settings
API_SERVER_PORT=8642
API_SERVER_HOST=0.0.0.0
API_SERVER_KEY=test-key-change-in-production

# CORS settings
API_SERVER_CORS_ENABLED=true
API_SERVER_CORS_ORIGINS=http://localhost:3000,http://localhost:8000

# Rate limiting
API_SERVER_RATE_LIMIT_ENABLED=true
API_SERVER_RATE_LIMIT_RPM=60

# SSL/TLS
API_SERVER_SSL_ENABLED=false
API_SERVER_SSL_CERT_PATH=/path/to/cert.pem
API_SERVER_SSL_KEY_PATH=/path/to/key.pem
```

### Terminal Backend Configuration

#### Local Terminal
```bash
TERMINAL_BACKEND=local
```

#### Docker Terminal
```bash
TERMINAL_BACKEND=docker
DOCKER_IMAGE=ubuntu:22.04
DOCKER_CONTAINER_NAME=hermes-terminal
DOCKER_VOLUMES=/tmp:/tmp
```

#### SSH Terminal
```bash
TERMINAL_BACKEND=ssh
SSH_HOST=localhost
SSH_PORT=22
SSH_USERNAME=user
SSH_KEY_PATH=~/.ssh/id_rsa
```

#### Modal Terminal
```bash
TERMINAL_BACKEND=modal
MODAL_TOKEN_PATH=~/.modal/token
MODAL_WORKSPACE=default
```

#### Daytona Terminal
```bash
TERMINAL_BACKEND=daytona
DAYTONA_API_URL=http://localhost:3000
DAYTONA_API_KEY=your-api-key
```

#### Singularity Terminal
```bash
TERMINAL_BACKEND=singularity
SINGULARITY_IMAGE_PATH=/path/to/image.sif
SINGULARITY_BIND_PATHS=/tmp:/tmp
```

### Memory Configuration

```bash
# Enable memory system
MEMORY_ENABLED=true

# Memory type: short_term, long_term, hybrid
MEMORY_TYPE=hybrid

# Memory limits
MEMORY_MAX_ENTRIES=1000
MEMORY_MAX_SIZE_MB=500

# Memory compression
MEMORY_COMPRESSION_ENABLED=true
MEMORY_COMPRESSION_THRESHOLD=500
MEMORY_COMPRESSION_RATIO=0.5

# Memory storage
MEMORY_STORAGE_TYPE=file
MEMORY_STORAGE_PATH=~/.hermes/memories
```

### Messaging Gateway Configuration

#### Telegram
```bash
TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN
TELEGRAM_ALLOWED_USERS=123456789,987654321
TELEGRAM_DM_PAIRING=true
TELEGRAM_DM_PAIRING_TIMEOUT=300
TELEGRAM_WEBHOOK_URL=https://your-domain.com/telegram
```

#### Discord
```bash
DISCORD_BOT_TOKEN=YOUR_BOT_TOKEN
DISCORD_ALLOWED_SERVERS=123456789
DISCORD_ALLOWED_CHANNELS=987654321
DISCORD_WEBHOOK_URL=https://your-domain.com/discord
```

#### Slack
```bash
SLACK_BOT_TOKEN=xoxb-YOUR_TOKEN
SLACK_APP_TOKEN=xapp-YOUR_TOKEN
SLACK_ALLOWED_WORKSPACES=workspace-id
SLACK_WEBHOOK_URL=https://your-domain.com/slack
```

#### WhatsApp
```bash
WHATSAPP_PHONE_NUMBER=+1234567890
WHATSAPP_API_KEY=YOUR_API_KEY
WHATSAPP_WEBHOOK_URL=https://your-domain.com/whatsapp
```

#### Signal
```bash
SIGNAL_PHONE_NUMBER=+1234567890
SIGNAL_API_URL=http://localhost:8080
```

#### Email
```bash
EMAIL_SMTP_SERVER=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_ADDRESS=your-email@gmail.com
EMAIL_PASSWORD=YOUR_APP_PASSWORD
EMAIL_ALLOWED_SENDERS=trusted@example.com
```

#### Matrix
```bash
MATRIX_HOMESERVER=https://matrix.org
MATRIX_USER_ID=@hermes:matrix.org
MATRIX_ACCESS_TOKEN=YOUR_ACCESS_TOKEN
MATRIX_ALLOWED_ROOMS=!roomid:matrix.org
```

#### Mattermost
```bash
MATTERMOST_SERVER_URL=https://mattermost.example.com
MATTERMOST_BOT_TOKEN=YOUR_BOT_TOKEN
MATTERMOST_ALLOWED_TEAMS=team-id
```

#### Feishu/Lark
```bash
FEISHU_APP_ID=YOUR_APP_ID
FEISHU_APP_SECRET=YOUR_APP_SECRET
FEISHU_WEBHOOK_URL=https://your-domain.com/feishu
```

#### WeCom
```bash
WECOM_CORP_ID=YOUR_CORP_ID
WECOM_AGENT_ID=YOUR_AGENT_ID
WECOM_SECRET=YOUR_SECRET
```

#### Weixin (WeChat)
```bash
WEIXIN_APP_ID=YOUR_APP_ID
WEIXIN_APP_SECRET=YOUR_APP_SECRET
WEIXIN_TOKEN=YOUR_TOKEN
```

#### DingTalk
```bash
DINGTALK_APP_ID=YOUR_APP_ID
DINGTALK_APP_SECRET=YOUR_APP_SECRET
DINGTALK_WEBHOOK_URL=https://your-domain.com/dingtalk
```

#### BlueBubbles
```bash
BLUEBUBBLES_SERVER_URL=http://localhost:1234
BLUEBUBBLES_PASSWORD=YOUR_PASSWORD
```

#### QQ
```bash
QQ_NUMBER=123456789
QQ_API_URL=http://localhost:5700
```

### Approval Configuration

```bash
# Enable approvals
APPROVALS_ENABLED=true

# Approval mode: manual, auto, hybrid
APPROVALS_MODE=manual

# Approval timeout (seconds)
APPROVALS_TIMEOUT=300

# Dangerous operations requiring approval
APPROVALS_DANGEROUS_OPS=file_delete,system_reboot,network_change

# Approval channels
APPROVALS_CHANNELS=telegram,email
```

### Tools Configuration

```bash
# Enable/disable tool categories
TOOLS_WEB_ENABLED=true
TOOLS_CODE_ENABLED=true
TOOLS_SYSTEM_ENABLED=true
TOOLS_DATA_ENABLED=true
TOOLS_COMMUNICATION_ENABLED=true
TOOLS_PRODUCTIVITY_ENABLED=true
TOOLS_AI_ENABLED=true
TOOLS_CUSTOM_ENABLED=true

# Disable specific tools
TOOLS_DISABLED=system_reboot,network_change

# Tool timeout (seconds)
TOOLS_TIMEOUT=30

# Max concurrent tools
TOOLS_MAX_CONCURRENT=5
```

### Logging Configuration

```bash
# Log level: DEBUG, INFO, WARNING, ERROR, CRITICAL
LOG_LEVEL=INFO

# Log format: json, text
LOG_FORMAT=json

# Log file
LOG_FILE=~/.hermes/logs/hermes.log

# Log file rotation
LOG_MAX_SIZE_MB=100
LOG_BACKUP_COUNT=5

# Log to console
LOG_CONSOLE=true

# Log to file
LOG_FILE_ENABLED=true
```

### Session Configuration

```bash
# Session timeout (seconds)
SESSION_TIMEOUT=3600

# Session storage: memory, file, redis
SESSION_STORAGE=file

# Session directory
SESSION_DIRECTORY=~/.hermes/sessions

# Max sessions
SESSION_MAX_SESSIONS=100

# Redis configuration (if SESSION_STORAGE=redis)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=
```

### Security Configuration

```bash
# Enable security features
SECURITY_ENABLED=true

# Require authentication
SECURITY_REQUIRE_AUTH=true

# Allowed IPs (comma-separated, empty = all)
SECURITY_ALLOWED_IPS=

# Blocked IPs (comma-separated)
SECURITY_BLOCKED_IPS=

# SSL/TLS
SECURITY_SSL_ENABLED=false
SECURITY_SSL_CERT_PATH=/path/to/cert.pem
SECURITY_SSL_KEY_PATH=/path/to/key.pem

# Rate limiting
SECURITY_RATE_LIMIT_ENABLED=true
SECURITY_RATE_LIMIT_RPM=60
```

### Advanced Configuration

```bash
# Debug mode
DEBUG=false

# Experimental features
EXPERIMENTAL=false

# Plugins directory
PLUGINS_DIR=~/.hermes/plugins

# Skills directory
SKILLS_DIR=~/.hermes/skills

# Enable telemetry
TELEMETRY_ENABLED=true

# Telemetry endpoint
TELEMETRY_ENDPOINT=https://telemetry.hermes-agent.nousresearch.com

# Custom configuration file
CONFIG_FILE=~/.hermes/config.yaml

# Custom SOUL file
SOUL_FILE=~/.hermes/SOUL.md
```

### Kubernetes-Specific Variables

```bash
# Pod name (auto-set by Kubernetes)
HOSTNAME=hermes-pod-name

# Namespace (auto-set by Kubernetes)
NAMESPACE=hermes

# Pod IP (auto-set by Kubernetes)
POD_IP=10.0.0.1

# Service name
SERVICE_NAME=hermes

# Service port
SERVICE_PORT=8642
```

## Setting Environment Variables

### Local Installation

Create `~/.hermes/.env`:

```bash
cat > ~/.hermes/.env << 'EOF'
OPENROUTER_API_KEY=sk-or-YOUR_KEY
MODEL=anthropic/claude-opus-4.6
BASE_URL=https://openrouter.ai/api/v1
API_SERVER_KEY=test-key
TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN
TELEGRAM_ALLOWED_USERS=123456789
