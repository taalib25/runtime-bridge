# Hermes Agent - Messaging Platforms Integration Guide

Complete reference for integrating Hermes Agent with 15+ messaging platforms.

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## Table of Contents

1. [Platform Overview](#platform-overview)
2. [Telegram](#telegram)
3. [Discord](#discord)
4. [Slack](#slack)
5. [WhatsApp](#whatsapp)
6. [Signal](#signal)
7. [Email](#email)
8. [SMS](#sms)
9. [Matrix](#matrix)
10. [Mattermost](#mattermost)
11. [Feishu/Lark](#feishulark)
12. [WeCom](#wecom)
13. [Weixin (WeChat)](#weixin-wechat)
14. [DingTalk](#dingtalk)
15. [BlueBubbles](#bluebubbles)
16. [QQ Bot](#qq-bot)
17. [Home Assistant](#home-assistant)

---

## Platform Overview

| Platform | Type | Voice | Images | Files | Threads | Typing | Streaming | Setup |
|----------|------|-------|--------|-------|---------|--------|-----------|-------|
| Telegram | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Easy |
| Discord | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| Slack | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| WhatsApp | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Hard |
| Signal | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Hard |
| Email | Async | ✗ | ✓ | ✓ | ✗ | ✗ | ✗ | Easy |
| SMS | Chat | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | Medium |
| Matrix | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| Mattermost | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| Feishu/Lark | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| WeCom | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| Weixin | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Hard |
| DingTalk | Chat | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Medium |
| BlueBubbles | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Hard |
| QQ Bot | Chat | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | Hard |
| Home Assistant | Automation | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | Easy |

---

## Telegram

### Setup

1. Create bot with @BotFather
2. Get bot token
3. Get your user ID
4. Configure in config.yaml and .env

### Configuration

**config.yaml**:
```yaml
gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
    allowed_users:
      - 123456789
    dm_pairing: true
    dm_pairing_timeout: 300
```

**.env**:
```bash
TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN
TELEGRAM_ALLOWED_USERS=123456789,987654321
TELEGRAM_DM_PAIRING=true
```

---

## Discord

### Setup

1. Create application in Developer Portal
2. Add bot to application
3. Copy bot token
4. Get server and channel IDs

### Configuration

**config.yaml**:
```yaml
gateways:
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
    allowed_servers:
      - 123456789012345678
    allowed_channels:
      - 987654321098765432
    dm_pairing: true
```

**.env**:
```bash
DISCORD_BOT_TOKEN=YOUR_BOT_TOKEN
DISCORD_ALLOWED_SERVERS=123456789012345678
DISCORD_ALLOWED_CHANNELS=987654321098765432
```

---

## Slack

### Setup

1. Create app at api.slack.com
2. Add OAuth scopes
3. Copy bot token (xoxb-...)
4. Create app token (xapp-...)

### Configuration

**config.yaml**:
```yaml
gateways:
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"
    app_token: "${SLACK_APP_TOKEN}"
    allowed_channels:
      - general
      - hermes-bot
    dm_pairing: true
```

**.env**:
```bash
SLACK_BOT_TOKEN=xoxb-YOUR_TOKEN
SLACK_APP_TOKEN=xapp-YOUR_TOKEN
SLACK_ALLOWED_CHANNELS=general,hermes-bot
```

---

## WhatsApp

### Setup

1. Create Meta Business Account
2. Create WhatsApp Business App
3. Get API key
4. Configure webhook

### Configuration

**config.yaml**:
```yaml
gateways:
  whatsapp:
    enabled: true
    api_key: "${WHATSAPP_API_KEY}"
    phone_number: "+1234567890"
    webhook_url: "https://your-domain.com/whatsapp"
    allowed_numbers:
      - "+1111111111"
```

**.env**:
```bash
WHATSAPP_API_KEY=YOUR_API_KEY
WHATSAPP_PHONE_NUMBER=+1234567890
WHATSAPP_ALLOWED_NUMBERS=+1111111111,+2222222222
```

---

## Signal

### Setup

1. Install Signal CLI
2. Register phone number
3. Configure API

### Configuration

**config.yaml**:
```yaml
gateways:
  signal:
    enabled: true
    phone_number: "+1234567890"
    api_url: "http://localhost:8080"
    allowed_numbers:
      - "+1111111111"
```

**.env**:
```bash
SIGNAL_PHONE_NUMBER=+1234567890
SIGNAL_API_URL=http://localhost:8080
SIGNAL_ALLOWED_NUMBERS=+1111111111,+2222222222
```

---

## Email

### Setup

1. Configure SMTP server
2. Get app password (Gmail/Outlook)
3. Add to config

### Configuration

**config.yaml**:
```yaml
gateways:
  email:
    enabled: true
    smtp_server: "smtp.gmail.com"
    smtp_port: 587
    email: "your-email@gmail.com"
    password: "${EMAIL_PASSWORD}"
    allowed_senders:
      - "trusted@example.com"
```

**.env**:
```bash
EMAIL_SMTP_SERVER=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_ADDRESS=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
EMAIL_ALLOWED_SENDERS=trusted@example.com
```

---

## SMS

### Setup

1. Create Twilio account
2. Get phone number
3. Copy Account SID and Auth Token

### Configuration

**config.yaml**:
```yaml
gateways:
  sms:
    enabled: true
    provider: "twilio"
    account_sid: "${SMS_ACCOUNT_SID}"
    auth_token: "${SMS_AUTH_TOKEN}"
    phone_number: "+1234567890"
    allowed_numbers:
      - "+1111111111"
```

**.env**:
```bash
SMS_ACCOUNT_SID=ACxxxxxx
SMS_AUTH_TOKEN=your-auth-token
SMS_PHONE_NUMBER=+1234567890
SMS_ALLOWED_NUMBERS=+1111111111,+2222222222
```

---

## Matrix

### Setup

1. Create Matrix account
2. Generate access token
3. Configure homeserver

### Configuration

**config.yaml**:
```yaml
gateways:
  matrix:
    enabled: true
    homeserver: "https://matrix.org"
    user_id: "@hermes:matrix.org"
    access_token: "${MATRIX_ACCESS_TOKEN}"
    allowed_rooms:
      - "!roomid:matrix.org"
```

**.env**:
```bash
MATRIX_HOMESERVER=https://matrix.org
MATRIX_USER_ID=@hermes:matrix.org
MATRIX_ACCESS_TOKEN=syt_your_token
MATRIX_ALLOWED_ROOMS=!roomid:matrix.org
```

---

## Mattermost

### Setup

1. Create bot account
2. Copy token
3. Configure server URL

### Configuration

**config.yaml**:
```yaml
gateways:
  mattermost:
    enabled: true
    server_url: "https://mattermost.example.com"
    bot_token: "${MATTERMOST_BOT_TOKEN}"
    allowed_channels:
      - general
      - hermes-bot
```

**.env**:
```bash
MATTERMOST_SERVER_URL=https://mattermost.example.com
MATTERMOST_BOT_TOKEN=your-bot-token
MATTERMOST_ALLOWED_CHANNELS=general,hermes-bot
```

---

## Feishu/Lark

### Setup

1. Create Feishu app
2. Get App ID and Secret
3. Configure webhook

### Configuration

**config.yaml**:
```yaml
gateways:
  feishu:
    enabled: true
    app_id: "${FEISHU_APP_ID}"
    app_secret: "${FEISHU_APP_SECRET}"
    webhook_url: "https://your-domain.com/feishu"
```

**.env**:
```bash
FEISHU_APP_ID=cli_xxxxx
FEISHU_APP_SECRET=your-app-secret
FEISHU_WEBHOOK_URL=https://your-domain.com/feishu
```

---

## WeCom

### Setup

1. Create WeCom app
2. Get Corp ID, Agent ID, Secret
3. Configure webhook

### Configuration

**config.yaml**:
```yaml
gateways:
  wecom:
    enabled: true
    corp_id: "${WECOM_CORP_ID}"
    agent_id: "${WECOM_AGENT_ID}"
    secret: "${WECOM_SECRET}"
    webhook_url: "https://your-domain.com/wecom"
```

**.env**:
```bash
WECOM_CORP_ID=ww123456789
WECOM_AGENT_ID=1000001
WECOM_SECRET=your-secret
WECOM_WEBHOOK_URL=https://your-domain.com/wecom
```

---

## Weixin (WeChat)

### Setup

1. Create WeChat Official Account
2. Get App ID and Secret
3. Configure webhook

### Configuration

**config.yaml**:
```yaml
gateways:
  weixin:
    enabled: true
    app_id: "${WEIXIN_APP_ID}"
    app_secret: "${WEIXIN_APP_SECRET}"
    token: "${WEIXIN_TOKEN}"
    webhook_url: "https://your-domain.com/weixin"
```

**.env**:
```bash
WEIXIN_APP_ID=wx123456789
WEIXIN_APP_SECRET=your-app-secret
WEIXIN_TOKEN=your-token
WEIXIN_WEBHOOK_URL=https://your-domain.com/weixin
```

---

## DingTalk

### Setup

1. Create DingTalk app
2. Get App ID and Secret
3. Configure webhook

### Configuration

**config.yaml**:
```yaml
gateways:
  dingtalk:
    enabled: true
    app_id: "${DINGTALK_APP_ID}"
    app_secret: "${DINGTALK_APP_SECRET}"
    webhook_url: "https://your-domain.com/dingtalk"
```

**.env**:
```bash
DINGTALK_APP_ID=ding123456789
DINGTALK_APP_SECRET=your-app-secret
DINGTALK_WEBHOOK_URL=https://your-domain.com/dingtalk
```

---

## BlueBubbles

### Setup

1. Install BlueBubbles Server (macOS only)
2. Configure server
3. Get server URL and password

### Configuration

**config.yaml**:
```yaml
gateways:
  bluebubbles:
    enabled: true
    server_url: "http://localhost:1234"
    password: "${BLUEBUBBLES_PASSWORD}"
    allowed_numbers:
      - "+1111111111"
```

**.env**:
```bash
BLUEBUBBLES_SERVER_URL=http://localhost:1234
BLUEBUBBLES_PASSWORD=your-server-password
BLUEBUBBLES_ALLOWED_NUMBERS=+1111111111,+2222222222
```

---

## QQ Bot

### Setup

1. Create QQ Bot
2. Get Bot ID and Token
3. Configure API URL

### Configuration

**config.yaml**:
```yaml
gateways:
  qq:
    enabled: true
    qq_number: "${QQ_NUMBER}"
    api_url: "http://localhost:5700"
    allowed_users:
      - qq-id-1
```

**.env**:
```bash
QQ_NUMBER=123456789
QQ_API_URL=http://localhost:5700
QQ_ALLOWED_USERS=qq-id-1,qq-id-2
```

---

## Home Assistant

### Setup

1. Install Home Assistant
2. Get API token
3. Configure entities

### Configuration

**config.yaml**:
```yaml
gateways:
  home_assistant:
    enabled: true
    api_url: "http://localhost:8123"
    api_token: "${HOME_ASSISTANT_API_TOKEN}"
    allowed_entities:
      - light.living_room
      - climate.bedroom
```

**.env**:
```bash
HOME_ASSISTANT_API_URL=http://localhost:8123
HOME_ASSISTANT_API_TOKEN=your-api-token
```

---

## Best Practices

1. Use environment variables for secrets
2. Enable allowlists for security
3. Use webhooks in production
4. Enable streaming for better UX
5. Monitor logs regularly
6. Test with DM pairing
7. Use separate bots per environment
8. Rotate tokens regularly
9. Enable rate limiting
10. Document your setup

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+
