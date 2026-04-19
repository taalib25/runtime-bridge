# Hermes Agent - Quick Start Tutorials

**Step-by-step tutorials for common Hermes Agent deployment scenarios.**

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## 🚀 Tutorial Index

1. [Local Development Setup](#local-development-setup)
2. [Telegram Bot Setup](#telegram-bot-setup)
3. [Discord Bot Setup](#discord-bot-setup)
4. [Docker Deployment](#docker-deployment)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [Open WebUI Integration](#open-webui-integration)
7. [Multi-Platform Setup](#multi-platform-setup)
8. [Production Hardening](#production-hardening)

---

## Local Development Setup

### Prerequisites
- Go 1.24+
- Docker (optional)
- Git

### Step 1: Clone Repository

```bash
git clone https://github.com/NousResearch/hermes-agent.git
cd hermes-agent
```

### Step 2: Install Dependencies

```bash
go mod download
go mod tidy
```

### Step 3: Create Configuration

```bash
mkdir -p ~/.hermes
cat > ~/.hermes/config.yaml << 'YAML'
model:
  name: gpt-3.5-turbo
  provider: openai

gateways:
  - name: telegram
    platform: telegram
    enabled: false
    token: ${TELEGRAM_BOT_TOKEN}

api_server:
  enabled: true
  port: 8080
  host: 0.0.0.0
YAML
```

### Step 4: Create Environment File

```bash
cat > ~/.hermes/.env << 'ENV'
HERMES_API_KEY=dev-key-12345
OPENAI_API_KEY=sk-your-key-here
TELEGRAM_BOT_TOKEN=your-token-here
HERMES_LOG_LEVEL=debug
ENV

chmod 600 ~/.hermes/.env
```

### Step 5: Run Agent

```bash
# Build
go build -o hermes ./cmd/main.go

# Run
./hermes run

# In another terminal, test API
curl http://localhost:8080/health
```

### Step 6: Test API

```bash
# Get models
curl -H "Authorization: Bearer dev-key-12345" \
  http://localhost:8080/v1/models

# Send message
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer dev-key-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## Telegram Bot Setup

### Prerequisites
- Telegram account
- BotFather access (@BotFather on Telegram)

### Step 1: Create Bot with BotFather

1. Open Telegram and search for @BotFather
2. Send `/newbot`
3. Follow prompts:
   - Bot name: "My Hermes Bot"
   - Bot username: "my_hermes_bot"
4. Copy the token (looks like: `123456789:ABCdefGHIjklmnoPQRstuvWXYZ`)

### Step 2: Configure Hermes

```bash
# Add to ~/.hermes/.env
echo "TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklmnoPQRstuvWXYZ" >> ~/.hermes/.env

# Update config.yaml
cat >> ~/.hermes/config.yaml << 'YAML'
gateways:
  - name: telegram
    platform: telegram
    enabled: true
    token: ${TELEGRAM_BOT_TOKEN}
    allowlist:
      users:
        - "YOUR_TELEGRAM_ID"  # Get this from @userinfobot
YAML
```

### Step 3: Get Your Telegram ID

1. Send `/start` to @userinfobot
2. It will reply with your user ID
3. Update the allowlist in config.yaml

### Step 4: Start Agent

```bash
./hermes run
```

### Step 5: Test Bot

1. Open Telegram
2. Search for your bot (my_hermes_bot)
3. Send `/start`
4. Send a message: "Hello"
5. Bot should respond

### Troubleshooting

```bash
# Check bot token
curl -s https://api.telegram.org/bot<YOUR_TOKEN>/getMe

# Check webhook status
curl -s https://api.telegram.org/bot<YOUR_TOKEN>/getWebhookInfo

# View logs
./hermes logs --tail 50
```

---

## Discord Bot Setup

### Prerequisites
- Discord account
- Discord server (or create one)
- Discord Developer Portal access

### Step 1: Create Discord Application

1. Go to https://discord.com/developers/applications
2. Click "New Application"
3. Name it "Hermes Agent"
4. Go to "Bot" section
5. Click "Add Bot"
6. Copy the token

### Step 2: Configure Bot Permissions

1. Go to "OAuth2" → "URL Generator"
2. Select scopes:
   - `bot`
   - `applications.commands`
3. Select permissions:
   - Send Messages
   - Read Messages/View Channels
   - Read Message History
   - Mention @everyone, @here, and @[Role]
4. Copy the generated URL

### Step 3: Add Bot to Server

1. Open the generated URL in browser
2. Select your server
3. Click "Authorize"

### Step 4: Configure Hermes

```bash
# Add to ~/.hermes/.env
echo "DISCORD_BOT_TOKEN=your-bot-token-here" >> ~/.hermes/.env

# Update config.yaml
cat >> ~/.hermes/config.yaml << 'YAML'
gateways:
  - name: discord
    platform: discord
    enabled: true
    token: ${DISCORD_BOT_TOKEN}
    intents:
      - message_content
      - guild_messages
      - direct_messages
    allowlist:
      channels:
        - "CHANNEL_ID"  # Get from Discord
YAML
```

### Step 5: Get Channel ID

1. Enable Developer Mode in Discord (User Settings → Advanced → Developer Mode)
2. Right-click channel → "Copy Channel ID"
3. Update config.yaml

### Step 6: Start Agent

```bash
./hermes run
```

### Step 7: Test Bot

1. Go to Discord server
2. Mention bot: `@Hermes Agent hello`
3. Bot should respond

---

## Docker Deployment

### Step 1: Create Dockerfile

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o hermes ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/hermes /usr/local/bin/
ENTRYPOINT ["hermes"]
```

### Step 2: Build Image

```bash
docker build -t hermes-agent:latest .
```

### Step 3: Create docker-compose.yml

```yaml
version: '3.8'

services:
  hermes:
    image: hermes-agent:latest
    container_name: hermes-agent
    restart: always
    ports:
      - "8080:8080"
    environment:
      - HERMES_LOG_LEVEL=info
    env_file:
      - .env
    volumes:
      - ./config.yaml:/etc/hermes/config.yaml:ro
      - hermes-cache:/var/cache/hermes
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Step 4: Deploy

```bash
# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f hermes

# Test API
curl http://localhost:8080/health
```

---

## Kubernetes Deployment

### Step 1: Create Namespace

```bash
kubectl create namespace hermes
```

### Step 2: Create Secrets

```bash
kubectl create secret generic hermes-secrets \
  --from-literal=HERMES_API_KEY=your-key \
  --from-literal=TELEGRAM_BOT_TOKEN=your-token \
  -n hermes
```

### Step 3: Create ConfigMap

```bash
kubectl create configmap hermes-config \
  --from-file=config.yaml=./config.yaml \
  -n hermes
```

### Step 4: Deploy

```bash
kubectl apply -f deployment.yaml -n hermes
```

### Step 5: Verify

```bash
# Check pods
kubectl get pods -n hermes

# Check logs
kubectl logs -n hermes -l app=hermes-agent -f

# Port forward
kubectl port-forward -n hermes svc/hermes-agent 8080:80

# Test
curl http://localhost:8080/health
```

---

## Open WebUI Integration

### Step 1: Start Hermes API Server

```bash
# Ensure API server is enabled in config.yaml
cat >> ~/.hermes/config.yaml << 'YAML'
api_server:
  enabled: true
  port: 8080
  host: 0.0.0.0
  cors:
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:8000"
YAML

# Start agent
./hermes run
```

### Step 2: Deploy Open WebUI

```bash
# Using Docker
docker run -d \
  -p 3000:8080 \
  -e OPENAI_API_BASE_URL=http://localhost:8080/v1 \
  -e OPENAI_API_KEY=your-api-key \
  ghcr.io/open-webui/open-webui:latest

# Or using Docker Compose
cat > docker-compose.yml << 'YAML'
version: '3.8'

services:
  open-webui:
    image: ghcr.io/open-webui/open-webui:latest
    container_name: open-webui
    ports:
      - "3000:8080"
    environment:
      - OPENAI_API_BASE_URL=http://hermes:8080/v1
      - OPENAI_API_KEY=your-api-key
    depends_on:
      - hermes

  hermes:
    image: hermes-agent:latest
    container_name: hermes-agent
    ports:
      - "8080:8080"
    env_file:
      - .env
    volumes:
      - ./config.yaml:/etc/hermes/config.yaml:ro
YAML

docker-compose up -d
```

### Step 3: Access Open WebUI

1. Open browser: http://localhost:3000
2. Sign up or login
3. Go to Settings → Models
4. Verify Hermes models are listed
5. Start chatting!

---

## Multi-Platform Setup

### Step 1: Configure Multiple Platforms

```yaml
# config.yaml
model:
  name: gpt-4
  provider: openai

gateways:
  - name: telegram
    platform: telegram
    enabled: true
    token: ${TELEGRAM_BOT_TOKEN}
    allowlist:
      users:
        - "123456789"

  - name: discord
    platform: discord
    enabled: true
    token: ${DISCORD_BOT_TOKEN}
    intents:
      - message_content
      - guild_messages

  - name: slack
    platform: slack
    enabled: true
    bot_token: ${SLACK_BOT_TOKEN}
    app_token: ${SLACK_APP_TOKEN}

api_server:
  enabled: true
  port: 8080
```

### Step 2: Set Environment Variables

```bash
cat >> ~/.hermes/.env << 'ENV'
TELEGRAM_BOT_TOKEN=your-telegram-token
DISCORD_BOT_TOKEN=your-discord-token
SLACK_BOT_TOKEN=xoxb-your-slack-token
SLACK_APP_TOKEN=xapp-your-slack-app-token
ENV
```

### Step 3: Start Agent

```bash
./hermes run
```

### Step 4: Test All Platforms

```bash
# Telegram
# Send message to bot

# Discord
# Mention bot in channel

# Slack
# Mention bot in channel

# API
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/v1/models
```

---

## Production Hardening

### Step 1: Security Configuration

```yaml
# config.yaml
security:
  dm_pairing:
    enabled: true
    require_verification: true
    timeout: 300

rate_limiting:
  global:
    requests_per_minute: 60
  per_user:
    requests_per_minute: 10

webhooks:
  - url: "https://example.com/webhook"
    verify_signature: true
    secret: ${WEBHOOK_SECRET}
```

### Step 2: Enable Monitoring

```yaml
# config.yaml
monitoring:
  enabled: true
  prometheus:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    jaeger_endpoint: "http://jaeger:6831"
```

### Step 3: Set Resource Limits

```yaml
# config.yaml
resources:
  memory_limit: 2048  # MB
  cpu_limit: 1000    # millicores
  cache_size: 500    # MB
```

### Step 4: Enable Logging

```bash
# Set log level
export HERMES_LOG_LEVEL=info

# Enable structured logging
export HERMES_LOG_FORMAT=json

# Start agent
./hermes run
```

### Step 5: Deploy with Monitoring

```bash
# Start with Prometheus
docker-compose -f docker-compose.prod.yml up -d

# Verify metrics
curl http://localhost:9090/metrics

# Access Grafana
# http://localhost:3000
```

---

## Next Steps

- **Configuration**: See [CONFIG_REFERENCE.md](../configuration/CONFIG_REFERENCE.md)
- **Troubleshooting**: See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- **Best Practices**: See [BEST_PRACTICES.md](./BEST_PRACTICES.md)
- **Production Deployment**: See [PRODUCTION_DEPLOYMENT.md](../deployment/PRODUCTION_DEPLOYMENT.md)

