# Hermes Agent Documentation

Complete reference guide for configuring, deploying, and operating Hermes Agent in Kubernetes environments.

## Quick Navigation

### 🚀 Getting Started
- **[Quick Start](./guides/QUICK_START.md)** — Deploy Hermes in 5 minutes
- **[Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md)** — Kubernetes & Helm deployment patterns

### ⚙️ Configuration
- **[Configuration Reference](./configuration/CONFIG_REFERENCE.md)** — Complete `config.yaml` and `.env` reference
- **[Environment Variables](./configuration/ENVIRONMENT_VARIABLES.md)** — 60+ variables across 9 categories
- **[SOUL.md Customization](./configuration/SOUL_CUSTOMIZATION.md)** — Agent personality and behavior

### 📱 Messaging & Integration
- **[Messaging Platforms](./guides/MESSAGING_PLATFORMS.md)** — Setup for 15+ platforms (Telegram, Discord, Slack, WhatsApp, etc.)
- **[Gateway Configuration](./configuration/GATEWAY_CONFIGURATION.md)** — Security, allowlists, and user management
- **[API Server & Open WebUI](./guides/API_SERVER_INTEGRATION.md)** — OpenAI-compatible API setup

### 🛠️ Tools & Skills
- **[Tools Reference](./reference/TOOLS_REFERENCE.md)** — 47+ built-in tools with enable/disable patterns
- **[Toolsets Guide](./guides/TOOLSETS_GUIDE.md)** — Organizing and managing tool collections

### 📊 Production & Operations
- **[Production Deployment](./deployment/PRODUCTION_DEPLOYMENT.md)** — Security hardening, multi-user, backup/recovery
- **[Troubleshooting](./guides/TROUBLESHOOTING.md)** — Common issues and solutions
- **[Best Practices](./guides/BEST_PRACTICES.md)** — Operational guidelines

### 📚 Reference
- **[Helm Chart Reference](./reference/HELM_CHART_REFERENCE.md)** — Chart values and customization
- **[Terminal Backends](./reference/TERMINAL_BACKENDS.md)** — Local, Docker, SSH, Modal, Daytona, Singularity
- **[Chat Commands](./reference/CHAT_COMMANDS.md)** — Built-in commands and shortcuts

## Documentation Structure

```
docs/
├── README.md (this file)
├── configuration/
│   ├── CONFIG_REFERENCE.md          # config.yaml structure
│   ├── ENVIRONMENT_VARIABLES.md     # .env and env vars
│   ├── SOUL_CUSTOMIZATION.md        # Personality/behavior
│   └── GATEWAY_CONFIGURATION.md     # Security & allowlists
├── deployment/
│   ├── DEPLOYMENT_GUIDE.md          # Kubernetes & Helm
│   ├── PRODUCTION_DEPLOYMENT.md     # Security & operations
│   └── KIND_CLUSTER_SETUP.md        # Kind-specific setup
├── guides/
│   ├── QUICK_START.md               # 5-minute setup
│   ├── MESSAGING_PLATFORMS.md       # Platform setup (15+)
│   ├── API_SERVER_INTEGRATION.md    # Open WebUI integration
│   ├── TOOLSETS_GUIDE.md            # Tool management
│   ├── TROUBLESHOOTING.md           # Common issues
│   └── BEST_PRACTICES.md            # Operational guidelines
└── reference/
    ├── TOOLS_REFERENCE.md           # 47+ tools catalog
    ├── HELM_CHART_REFERENCE.md      # Chart values
    ├── TERMINAL_BACKENDS.md         # Backend options
    └── CHAT_COMMANDS.md             # Command reference
```

## Key Information

### Current Version
- **Release**: v0.10.0 (April 16, 2026)
- **Latest Feature**: Tool Gateway for Nous Portal subscribers
- **Python**: 3.11+ required

### Critical Configuration Files
| File | Location | Purpose |
|------|----------|---------|
| `config.yaml` | `~/.hermes/config.yaml` | Main settings (model, terminal, memory, tools) |
| `.env` | `~/.hermes/.env` | Secrets (API keys, bot tokens) |
| `SOUL.md` | `~/.hermes/SOUL.md` | Agent personality and behavior |
| `gateway.json` | `~/.hermes/gateway.json` | Gateway policies and platform settings |
| `MEMORY.md` | `~/.hermes/memories/MEMORY.md` | Persistent memory |
| `USER.md` | `~/.hermes/memories/USER.md` | User information |

### Supported Messaging Platforms (15+)
Telegram, Discord, Slack, WhatsApp, Signal, SMS, Email, Matrix, Mattermost, Feishu/Lark, WeCom, Weixin, DingTalk, BlueBubbles, QQ

### Terminal Backends
Local, Docker, SSH, Modal, Daytona, Singularity

### Built-in Tools (47+)
Across 8 categories: Web, Code, System, Data, Communication, Productivity, AI, Custom

## Installation

### One-Liner
```bash
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
```

### Manual Installation
```bash
pip install hermes-agent
hermes init
hermes start
```

### Kubernetes/Helm
```bash
git clone https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
helm install hermes . --namespace hermes --create-namespace -f values.yaml
```

## Common Tasks

### Deploy Hermes in Kind Cluster
See: [Quick Start](./guides/QUICK_START.md) → [Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md)

### Configure Telegram Bot
See: [Messaging Platforms](./guides/MESSAGING_PLATFORMS.md) → Telegram section

### Set Up API Server with Open WebUI
See: [API Server Integration](./guides/API_SERVER_INTEGRATION.md)

### Customize Agent Personality
See: [SOUL.md Customization](./configuration/SOUL_CUSTOMIZATION.md)

### Enable/Disable Tools
See: [Tools Reference](./reference/TOOLS_REFERENCE.md) → Configuration section

### Production Hardening
See: [Production Deployment](./deployment/PRODUCTION_DEPLOYMENT.md)

## External Resources

- **Official Website**: https://hermes-agent.nousresearch.com/
- **Official Docs**: https://hermes-agent.nousresearch.com/docs/
- **GitHub**: https://github.com/NousResearch/hermes-agent
- **Helm Chart**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Discord Community**: https://discord.gg/NousResearch
- **Skills Hub**: https://agentskills.io

## Support

- **Issues**: https://github.com/NousResearch/hermes-agent/issues
- **Discussions**: https://github.com/NousResearch/hermes-agent/discussions
- **Discord**: https://discord.gg/NousResearch

## License

Hermes Agent is open source. See the main repository for license details.

---

**Last Updated**: April 18, 2026  
**Documentation Version**: 1.0  
**Hermes Version**: v0.10.0
