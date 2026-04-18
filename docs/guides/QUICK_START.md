# Hermes Agent - Quick Start Guide

Deploy Hermes Agent in your Kind cluster in 5 minutes.

## Prerequisites

- Kubernetes cluster (Kind, minikube, or cloud)
- `kubectl` configured
- `helm` 3.0+
- API key for LLM provider (OpenRouter, OpenAI, Anthropic, etc.)

## Option 1: Helm Chart (Recommended for Kubernetes)

### Step 1: Add Helm Repository

```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
```

### Step 2: Create Values File

```bash
cat > values-kind.yaml << 'YAML'
# Hermes Agent Helm Values for Kind Cluster

# API Keys and Secrets
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_OPENROUTER_KEY"
  API_SERVER_KEY: "test-key-change-in-production"

# API Server Configuration
apiServer:
  enabled: true
  port: 8642

# Service Configuration
service:
  enabled: true
  type: ClusterIP
  port: 8642

# Model Configuration
config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      base_url: https://openrouter.ai/api/v1

# Persistence
persistence:
  enabled: true
  size: 5Gi
  storageClassName: standard

# Resource Limits
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
YAML
```

### Step 3: Deploy

```bash
# Create namespace
kubectl create namespace hermes

# Deploy Hermes
helm install hermes . \
  --namespace hermes \
  -f values-kind.yaml

# Wait for deployment
kubectl rollout status deployment/hermes -n hermes --timeout=5m
```

### Step 4: Verify Deployment

```bash
# Check pod status
kubectl get pods -n hermes

# View logs
kubectl logs -n hermes deployment/hermes -f

# Port forward to access API
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test API
curl http://localhost:8642/health
```

## Option 2: Docker Compose (Local Development)

### Step 1: Create docker-compose.yml

```yaml
version: '3.8'

services:
  hermes:
    image: nousresearch/hermes-agent:latest
    container_name: hermes-agent
    ports:
      - "8642:8642"
    environment:
      OPENROUTER_API_KEY: "sk-or-YOUR_KEY"
      API_SERVER_KEY: "test-key"
      MODEL: "anthropic/claude-opus-4.6"
      BASE_URL: "https://openrouter.ai/api/v1"
    volumes:
      - hermes-data:/root/.hermes
    restart: unless-stopped

volumes:
  hermes-data:
```

### Step 2: Start

```bash
docker-compose up -d
docker-compose logs -f
```

## Option 3: Direct Installation (Local)

### Step 1: Install

```bash
# Using pip
pip install hermes-agent

# Or using the install script
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
```

### Step 2: Initialize

```bash
hermes init
```

This creates `~/.hermes/` with:
- `config.yaml` — Main configuration
- `.env` — Secrets and API keys
- `SOUL.md` — Agent personality
- `memories/` — Persistent memory

### Step 3: Configure

Edit `~/.hermes/.env`:

```bash
OPENROUTER_API_KEY=sk-or-YOUR_KEY
MODEL=anthropic/claude-opus-4.6
BASE_URL=https://openrouter.ai/api/v1
API_SERVER_KEY=test-key
```

### Step 4: Start

```bash
hermes start
```

## First Steps After Deployment

### 1. Access the API

```bash
# If using Kubernetes
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test the API
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key-change-in-production" \
  -d '{
    "model": "hermes",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
  }'
```

### 2. Connect to Open WebUI

If using Open WebUI:

```bash
# Port forward Hermes API
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# In Open WebUI:
# 1. Go to Settings → Models
# 2. Click "Add Model"
# 3. Enter:
#    - Name: hermes
#    - API URL: http://localhost:8642/v1
#    - API Key: test-key-change-in-production
```

### 3. Configure a Messaging Platform

Example: Telegram

```bash
# Get your Telegram bot token from @BotFather

# Edit config.yaml
cat >> ~/.hermes/config.yaml << 'YAML'
gateways:
  telegram:
    enabled: true
    token: "YOUR_TELEGRAM_BOT_TOKEN"
    allowed_users:
      - YOUR_TELEGRAM_USER_ID
YAML

# Restart Hermes
hermes restart
```

### 4. Customize Agent Personality

Edit `~/.hermes/SOUL.md`:

```markdown
# Hermes Agent

You are Hermes, an AI assistant created by Nous Research.

## Personality
- Helpful and knowledgeable
- Direct and concise
- Proactive in offering solutions

## Capabilities
- Code analysis and generation
- System administration
- Data analysis
- Creative writing

## Constraints
- Always verify information before sharing
- Respect user privacy
- Decline harmful requests
```

## Troubleshooting

### Pod won't start

```bash
# Check logs
kubectl logs -n hermes deployment/hermes

# Check events
kubectl describe pod -n hermes <pod-name>

# Check resource availability
kubectl top nodes
kubectl top pods -n hermes
```

### API returns 401 Unauthorized

- Verify API key in `.env` or Helm values
- Check `API_SERVER_KEY` matches in requests
- Ensure `Authorization: Bearer <key>` header is set

### Hermes not responding to messages

- Check gateway configuration in `config.yaml`
- Verify bot token/credentials in `.env`
- Check logs: `kubectl logs -n hermes deployment/hermes -f`
- Verify user is in allowlist

### Out of memory

- Increase resource limits in Helm values
- Reduce model size or use quantized version
- Enable memory compression in `config.yaml`

## Next Steps

1. **[Configuration Reference](../configuration/CONFIG_REFERENCE.md)** — Customize all settings
2. **[Messaging Platforms](./MESSAGING_PLATFORMS.md)** — Connect to Telegram, Discord, Slack, etc.
3. **[API Server Integration](./API_SERVER_INTEGRATION.md)** — Set up with Open WebUI
4. **[Production Deployment](../deployment/PRODUCTION_DEPLOYMENT.md)** — Harden for production
5. **[Tools Reference](../reference/TOOLS_REFERENCE.md)** — Enable/disable tools

## Common Commands

```bash
# Kubernetes
kubectl get pods -n hermes
kubectl logs -n hermes deployment/hermes -f
kubectl exec -it -n hermes deployment/hermes -- bash
kubectl port-forward -n hermes svc/hermes 8642:8642

# Local
hermes start
hermes stop
hermes restart
hermes status
hermes logs
hermes config show
```

## Support

- **Docs**: https://hermes-agent.nousresearch.com/docs/
- **GitHub**: https://github.com/NousResearch/hermes-agent
- **Discord**: https://discord.gg/NousResearch

---

**Last Updated**: April 18, 2026
