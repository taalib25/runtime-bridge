# Hermes Agent - Complete Platform Integration Documentation Index

**Comprehensive guide to all 15+ messaging platforms, API server, and integration options.**

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+  
**Documentation Version**: 1.0

---

## 📋 Quick Navigation

### 🚀 Getting Started
- **[Messaging Platforms Guide](./docs/guides/MESSAGING_PLATFORMS.md)** — Setup for 15+ platforms
- **[Gateway Configuration](./docs/configuration/GATEWAY_CONFIGURATION.md)** — Security, allowlists, user management
- **[API Server Integration](./docs/guides/API_SERVER_INTEGRATION.md)** — OpenAI-compatible API & Open WebUI

### 📱 Supported Messaging Platforms

| Platform | Type | Setup | Capabilities | Docs |
|----------|------|-------|--------------|------|
| **Telegram** | Chat | Easy | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#telegram) |
| **Discord** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#discord) |
| **Slack** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#slack) |
| **WhatsApp** | Chat | Hard | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#whatsapp) |
| **Signal** | Chat | Hard | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#signal) |
| **Email** | Async | Easy | Images, Files | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#email) |
| **SMS** | Chat | Medium | Text only | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#sms) |
| **Matrix** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#matrix) |
| **Mattermost** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#mattermost) |
| **Feishu/Lark** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#feishulark) |
| **WeCom** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#wecom) |
| **Weixin (WeChat)** | Chat | Hard | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#weixin-wechat) |
| **DingTalk** | Chat | Medium | Voice, Images, Files, Threads, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#dingtalk) |
| **BlueBubbles** | Chat | Hard | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#bluebubbles) |
| **QQ Bot** | Chat | Hard | Voice, Images, Files, Streaming | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#qq-bot) |
| **Home Assistant** | Automation | Easy | Entity control | [Setup](./docs/guides/MESSAGING_PLATFORMS.md#home-assistant) |

---

## 🔧 Configuration Files

### Main Configuration Files

| File | Location | Purpose | Reference |
|------|----------|---------|-----------|
| `config.yaml` | `~/.hermes/config.yaml` | Main settings (model, terminal, memory, gateways) | [CONFIG_REFERENCE.md](./docs/configuration/CONFIG_REFERENCE.md) |
| `.env` | `~/.hermes/.env` | Secrets (API keys, bot tokens) | [ENVIRONMENT_VARIABLES.md](./docs/configuration/ENVIRONMENT_VARIABLES.md) |
| `gateway.json` | `~/.hermes/gateway.json` | Gateway policies and platform settings | [GATEWAY_CONFIGURATION.md](./docs/configuration/GATEWAY_CONFIGURATION.md) |
| `SOUL.md` | `~/.hermes/SOUL.md` | Agent personality and behavior | [SOUL_CUSTOMIZATION.md](./docs/configuration/SOUL_CUSTOMIZATION.md) |

---

## 📱 Platform Integration Checklist

### Telegram Setup
- [ ] Create bot with @BotFather
- [ ] Get bot token
- [ ] Get your user ID
- [ ] Add to `config.yaml`
- [ ] Set `TELEGRAM_BOT_TOKEN` in `.env`
- [ ] Test with `/start` command

**Quick Setup**:
```yaml
gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
    allowed_users:
      - YOUR_USER_ID
    dm_pairing: true
```

### Discord Setup
- [ ] Create application in Developer Portal
- [ ] Add bot to application
- [ ] Copy bot token
- [ ] Get server and channel IDs
- [ ] Add to `config.yaml`
- [ ] Set `DISCORD_BOT_TOKEN` in `.env`
- [ ] Test with @mention

**Quick Setup**:
```yaml
gateways:
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
    allowed_servers:
      - SERVER_ID
    allowed_channels:
      - CHANNEL_ID
```

### Slack Setup
- [ ] Create app at api.slack.com
- [ ] Add OAuth scopes
- [ ] Copy bot token (xoxb-...)
- [ ] Create app token (xapp-...)
- [ ] Add to `config.yaml`
- [ ] Set tokens in `.env`
- [ ] Test with @mention

**Quick Setup**:
```yaml
gateways:
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"
    app_token: "${SLACK_APP_TOKEN}"
    allowed_channels:
      - general
```

### WhatsApp Setup
- [ ] Create Meta Business Account
- [ ] Create WhatsApp Business App
- [ ] Get API key
- [ ] Configure webhook
- [ ] Add to `config.yaml`
- [ ] Set credentials in `.env`
- [ ] Test with message

**Quick Setup**:
```yaml
gateways:
  whatsapp:
    enabled: true
    api_key: "${WHATSAPP_API_KEY}"
    phone_number: "+1234567890"
    webhook_url: "https://your-domain.com/whatsapp"
```

### Email Setup
- [ ] Configure SMTP server
- [ ] Get app password (Gmail/Outlook)
- [ ] Add to `config.yaml`
- [ ] Set credentials in `.env`
- [ ] Test with email

**Quick Setup**:
```yaml
gateways:
  email:
    enabled: true
    smtp_server: "smtp.gmail.com"
    email: "your-email@gmail.com"
    password: "${EMAIL_PASSWORD}"
    allowed_senders:
      - "trusted@example.com"
```

---

## 🔐 Security & Access Control

### Allowlist Configuration

**User Allowlists**:
```yaml
gateways:
  telegram:
    allowed_users:
      - 123456789
      - 987654321
    blocked_users:
      - 111111111
    allow_all_users: false
```

**Channel Allowlists**:
```yaml
gateways:
  discord:
    allowed_channels:
      - 987654321098765432
    blocked_channels:
      - 555555555555555555
    allow_all_channels: false
```

**Server/Workspace Allowlists**:
```yaml
gateways:
  slack:
    allowed_workspaces:
      - workspace-id-1
    blocked_workspaces:
      - workspace-id-3
    allow_all_workspaces: false
```

### Admin Users

```yaml
gateways:
  telegram:
    admin_users:
      - 123456789
      - 987654321
```

### Rate Limiting

```yaml
api_server:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
```

---

## 🔗 DM Pairing System

### How It Works

1. User sends `/pair` command
2. Bot generates pairing code
3. User confirms in DM
4. Bot remembers user for future messages

### Configuration

```yaml
dm_pairing:
  enabled: true
  timeout: 300
  code_length: 6
  code_format: "alphanumeric"
  max_attempts: 3
  storage: "file"
```

### Environment Variables

```bash
DM_PAIRING_ENABLED=true
DM_PAIRING_TIMEOUT=300
DM_PAIRING_CODE_LENGTH=6
DM_PAIRING_MAX_ATTEMPTS=3
```

---

## 🌐 API Server & Open WebUI

### API Server Configuration

```yaml
api_server:
  enabled: true
  host: 0.0.0.0
  port: 8642
  key: "${API_SERVER_KEY}"
  model_name: "hermes-agent"
  openai_compatible: true
```

### OpenAI-Compatible Endpoints

```bash
# Chat completion
POST /v1/chat/completions

# Models list
GET /v1/models

# Health check
GET /health
```

### Open WebUI Integration

```bash
docker run -d -p 3000:8080 \
  -e OPENAI_API_BASE_URL=http://hermes:8642/v1 \
  -e OPENAI_API_KEY=your-api-key \
  ghcr.io/open-webui/open-webui:latest
```

---

## 🚀 Streaming Configuration

### Global Streaming

```yaml
streaming:
  enabled: true
  default_chunk_size: 1000
  default_chunk_delay: 100
```

### Per-Platform Streaming

```yaml
streaming:
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
```

---

## 🪝 Webhook Configuration

### Webhook Setup

```yaml
webhook:
  enabled: true
  port: 8644
  host: 0.0.0.0
  secret: "${WEBHOOK_SECRET}"
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
```

### Webhook Events

- `message` - New message
- `callback_query` - Button click
- `reaction` - Emoji reaction
- `status` - Delivery status
- `error` - Error notification

---

## 📊 Environment Variables Reference

### LLM Configuration

```bash
OPENROUTER_API_KEY=sk-or-YOUR_KEY
MODEL=anthropic/claude-opus-4.6
BASE_URL=https://openrouter.ai/api/v1
TEMPERATURE=0.7
MAX_TOKENS=4096
```

### API Server

```bash
API_SERVER_ENABLED=true
API_SERVER_PORT=8642
API_SERVER_HOST=0.0.0.0
API_SERVER_KEY=your-api-key
API_SERVER_CORS_ENABLED=true
API_SERVER_CORS_ORIGINS=http://localhost:3000
API_SERVER_RATE_LIMIT_ENABLED=true
API_SERVER_RATE_LIMIT_RPM=60
```

### Telegram

```bash
TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN
TELEGRAM_ALLOWED_USERS=123456789,987654321
TELEGRAM_ALLOW_ALL_USERS=false
TELEGRAM_DM_PAIRING=true
TELEGRAM_DM_PAIRING_TIMEOUT=300
```

### Discord

```bash
DISCORD_BOT_TOKEN=YOUR_BOT_TOKEN
DISCORD_ALLOWED_SERVERS=123456789012345678
DISCORD_ALLOWED_CHANNELS=987654321098765432
DISCORD_ALLOW_ALL_CHANNELS=false
DISCORD_DM_PAIRING=true
```

### Slack

```bash
SLACK_BOT_TOKEN=xoxb-YOUR_TOKEN
SLACK_APP_TOKEN=xapp-YOUR_TOKEN
SLACK_ALLOWED_CHANNELS=general,hermes-bot
SLACK_ALLOW_ALL_CHANNELS=false
```

### WhatsApp

```bash
WHATSAPP_API_KEY=YOUR_API_KEY
WHATSAPP_BUSINESS_ACCOUNT_ID=123456789
WHATSAPP_PHONE_NUMBER_ID=987654321
WHATSAPP_WEBHOOK_SECRET=your-webhook-secret
WHATSAPP_ALLOWED_NUMBERS=+1111111111,+2222222222
```

### Email

```bash
EMAIL_SMTP_SERVER=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_ADDRESS=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
EMAIL_ALLOWED_SENDERS=trusted@example.com
```

### Matrix

```bash
MATRIX_HOMESERVER=https://matrix.org
MATRIX_USER_ID=@hermes:matrix.org
MATRIX_ACCESS_TOKEN=syt_your_token
MATRIX_ALLOWED_ROOMS=!roomid:matrix.org
```

### Mattermost

```bash
MATTERMOST_SERVER_URL=https://mattermost.example.com
MATTERMOST_BOT_TOKEN=your-bot-token
MATTERMOST_ALLOWED_TEAMS=team-id
MATTERMOST_ALLOWED_CHANNELS=general
```

### Feishu/Lark

```bash
FEISHU_APP_ID=cli_xxxxx
FEISHU_APP_SECRET=your-app-secret
FEISHU_WEBHOOK_SECRET=your-webhook-secret
FEISHU_ALLOWED_USERS=user-id-1
```

### WeCom

```bash
WECOM_CORP_ID=ww123456789
WECOM_AGENT_ID=1000001
WECOM_SECRET=your-secret
WECOM_WEBHOOK_SECRET=your-webhook-secret
```

### Weixin (WeChat)

```bash
WEIXIN_APP_ID=wx123456789
WEIXIN_APP_SECRET=your-app-secret
WEIXIN_TOKEN=your-token
WEIXIN_WEBHOOK_SECRET=your-webhook-secret
```

### DingTalk

```bash
DINGTALK_APP_ID=ding123456789
DINGTALK_APP_SECRET=your-app-secret
DINGTALK_WEBHOOK_SECRET=your-webhook-secret
```

### BlueBubbles

```bash
BLUEBUBBLES_SERVER_URL=http://localhost:1234
BLUEBUBBLES_PASSWORD=your-server-password
BLUEBUBBLES_ALLOWED_NUMBERS=+1111111111
```

### QQ Bot

```bash
QQ_NUMBER=123456789
QQ_API_URL=http://localhost:5700
QQ_ALLOWED_USERS=qq-id-1
```

### Home Assistant

```bash
HOME_ASSISTANT_API_URL=http://localhost:8123
HOME_ASSISTANT_API_TOKEN=your-api-token
HOME_ASSISTANT_ALLOWED_ENTITIES=light.living_room
```

---

## 📚 Complete Documentation Structure

```
docs/
├── README.md                                    # Main documentation index
├── configuration/
│   ├── CONFIG_REFERENCE.md                     # config.yaml structure
│   ├── ENVIRONMENT_VARIABLES.md                # .env and env vars
│   ├── GATEWAY_CONFIGURATION.md                # Gateway security & allowlists
│   └── SOUL_CUSTOMIZATION.md                   # Agent personality
├── guides/
│   ├── QUICK_START.md                          # 5-minute setup
│   ├── MESSAGING_PLATFORMS.md                  # 15+ platform setup (THIS FILE)
│   ├── API_SERVER_INTEGRATION.md               # OpenAI-compatible API
│   ├── TOOLSETS_GUIDE.md                       # Tool management
│   ├── TROUBLESHOOTING.md                      # Common issues
│   └── BEST_PRACTICES.md                       # Operational guidelines
├── deployment/
│   ├── DEPLOYMENT_GUIDE.md                     # Kubernetes & Helm
│   ├── PRODUCTION_DEPLOYMENT.md                # Security & operations
│   └── KIND_CLUSTER_SETUP.md                   # Kind-specific setup
└── reference/
    ├── TOOLS_REFERENCE.md                      # 47+ tools catalog
    ├── HELM_CHART_REFERENCE.md                 # Chart values
    ├── TERMINAL_BACKENDS.md                    # Backend options
    └── CHAT_COMMANDS.md                        # Command reference
```

---

## 🎯 Common Use Cases

### Use Case 1: Personal Telegram Bot

```yaml
gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
    allowed_users:
      - YOUR_USER_ID
    dm_pairing: true

api_server:
  enabled: false
```

### Use Case 2: Team Discord Bot

```yaml
gateways:
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
    allowed_servers:
      - YOUR_SERVER_ID
    allowed_channels:
      - general
      - hermes-bot
    admin_users:
      - ADMIN_USER_ID

api_server:
  enabled: false
```

### Use Case 3: Enterprise Slack Bot

```yaml
gateways:
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"
    app_token: "${SLACK_APP_TOKEN}"
    allowed_channels:
      - general
      - engineering
      - support

api_server:
  enabled: true
  port: 8642
  key: "${API_SERVER_KEY}"
  cors:
    enabled: true
    allowed_origins:
      - "https://app.example.com"
```

### Use Case 4: Open WebUI Integration

```yaml
api_server:
  enabled: true
  host: 0.0.0.0
  port: 8642
  key: "${API_SERVER_KEY}"
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"

gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
```

### Use Case 5: Multi-Platform Setup

```yaml
gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
  
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
  
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"
    app_token: "${SLACK_APP_TOKEN}"
  
  email:
    enabled: true
    smtp_server: "smtp.gmail.com"
    email: "${EMAIL_ADDRESS}"
    password: "${EMAIL_PASSWORD}"

api_server:
  enabled: true
  port: 8642
  key: "${API_SERVER_KEY}"
```

---

## 🔍 Troubleshooting

### Bot Not Responding

1. Check if gateway is enabled: `grep "enabled: true" ~/.hermes/config.yaml`
2. Verify credentials: `grep "TOKEN\|KEY" ~/.hermes/.env`
3. Check logs: `tail -f ~/.hermes/logs/hermes.log`
4. Verify allowlist: `grep "allowed_users" ~/.hermes/config.yaml`

### Webhook Not Working

1. Check webhook URL: `curl -X POST https://your-domain.com/telegram`
2. Verify firewall: `netstat -tlnp | grep 8644`
3. Check logs: `tail -f ~/.hermes/logs/hermes.log | grep webhook`

### Rate Limiting Issues

1. Increase rate limit: `API_SERVER_RATE_LIMIT_RPM=120`
2. Check current usage: `curl http://localhost:8642/metrics`

### API Server Not Responding

1. Check if enabled: `grep "api_server:" ~/.hermes/config.yaml`
2. Verify port: `netstat -tlnp | grep 8642`
3. Check logs: `tail -f ~/.hermes/logs/api-server.log`

---

## 📖 Additional Resources

- **Official Website**: https://hermes-agent.nousresearch.com/
- **GitHub Repository**: https://github.com/NousResearch/hermes-agent
- **Helm Chart**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Discord Community**: https://discord.gg/NousResearch
- **Skills Hub**: https://agentskills.io

---

## 📝 Document Versions

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | April 18, 2026 | Initial comprehensive documentation |

---

## ✅ Checklist for Complete Setup

- [ ] Read [Messaging Platforms Guide](./docs/guides/MESSAGING_PLATFORMS.md)
- [ ] Read [Gateway Configuration](./docs/configuration/GATEWAY_CONFIGURATION.md)
- [ ] Read [API Server Integration](./docs/guides/API_SERVER_INTEGRATION.md)
- [ ] Choose platforms to enable
- [ ] Create bot accounts for each platform
- [ ] Get API keys and tokens
- [ ] Configure `config.yaml`
- [ ] Set environment variables in `.env`
- [ ] Test each platform
- [ ] Enable API server (optional)
- [ ] Set up Open WebUI (optional)
- [ ] Configure allowlists
- [ ] Enable DM pairing
- [ ] Test streaming responses
- [ ] Monitor logs
- [ ] Deploy to production

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+  
**Documentation Version**: 1.0

For questions or issues, please visit:
- **GitHub Issues**: https://github.com/NousResearch/hermes-agent/issues
- **Discord**: https://discord.gg/NousResearch
- **Discussions**: https://github.com/NousResearch/hermes-agent/discussions
