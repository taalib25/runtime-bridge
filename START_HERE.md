# 🚀 Hermes Agent Documentation - START HERE

Welcome! This is your entry point to comprehensive Hermes Agent documentation.

## 📍 Where to Start?

### ⏱️ I have 5 minutes
→ **[Quick Start Guide](./docs/guides/QUICK_START.md)**
- Deploy Hermes in 5 minutes
- Helm Chart, Docker Compose, or local installation
- Verification steps

### ⏱️ I have 30 minutes
→ **[Deployment Guide](./docs/deployment/DEPLOYMENT_GUIDE.md)**
- Choose your deployment platform
- Kubernetes, Helm, Kustomize, or manual YAML
- Production patterns

### ⏱️ I have 1 hour
→ **[Configuration Reference](./docs/configuration/CONFIG_REFERENCE.md)**
- Complete config.yaml structure
- All configuration options
- Use case examples

### ⏱️ I have 2+ hours
→ **[Production Deployment](./docs/deployment/PRODUCTION_DEPLOYMENT.md)**
- Security hardening
- High availability setup
- Monitoring and observability
- Best practices

## 🎯 By Role

### 👤 First-Time Users
1. [Quick Start](./docs/guides/QUICK_START.md) — Get running in 5 minutes
2. [Configuration Reference](./docs/configuration/CONFIG_REFERENCE.md) — Understand settings
3. [Troubleshooting](./docs/guides/TROUBLESHOOTING.md) — Fix common issues

### 🔧 DevOps / Platform Engineers
1. [Deployment Guide](./docs/deployment/DEPLOYMENT_GUIDE.md) — Choose platform
2. [Production Deployment](./docs/deployment/PRODUCTION_DEPLOYMENT.md) — Harden for production
3. [Best Practices](./docs/guides/BEST_PRACTICES.md) — Operational guidelines
4. [Monitoring](./docs/guides/BEST_PRACTICES.md#monitoring) — Set up observability

### 🔌 Integration / Backend Teams
1. [API Server Integration](./docs/guides/API_SERVER_INTEGRATION.md) — Set up API
2. [Messaging Platforms](./docs/guides/MESSAGING_PLATFORMS.md) — Connect to Telegram, Discord, etc.
3. [Tools Reference](./docs/reference/TOOLS_REFERENCE.md) — Manage tools and skills
4. [Configuration Reference](./docs/configuration/CONFIG_REFERENCE.md) — Fine-tune settings

### 🎨 Customization / Product Teams
1. [SOUL.md Customization](./docs/configuration/SOUL_CUSTOMIZATION.md) — Define agent personality
2. [Tools Reference](./docs/reference/TOOLS_REFERENCE.md) — Enable/disable tools
3. [Toolsets Guide](./docs/guides/TOOLSETS_GUIDE.md) — Organize tool collections
4. [Configuration Reference](./docs/configuration/CONFIG_REFERENCE.md) — Customize behavior

## 📚 Documentation Map

```
docs/
├── README.md                          ← Main index & navigation
├── configuration/
│   ├── CONFIG_REFERENCE.md            ← Complete config.yaml
│   ├── ENVIRONMENT_VARIABLES.md       ← All env vars (60+)
│   ├── SOUL_CUSTOMIZATION.md          ← Agent personality
│   └── GATEWAY_CONFIGURATION.md       ← Security & allowlists
├── deployment/
│   ├── DEPLOYMENT_GUIDE.md            ← Kubernetes/Helm/Kustomize
│   ├── PRODUCTION_DEPLOYMENT.md       ← Security hardening
│   └── KIND_CLUSTER_SETUP.md          ← Kind-specific setup
├── guides/
│   ├── QUICK_START.md                 ← 5-minute deployment
│   ├── MESSAGING_PLATFORMS.md         ← 15+ platform setup
│   ├── API_SERVER_INTEGRATION.md      ← Open WebUI integration
│   ├── TOOLSETS_GUIDE.md              ← Tool management
│   ├── TROUBLESHOOTING.md             ← Common issues
│   └── BEST_PRACTICES.md              ← Operational guidelines
└── reference/
    ├── TOOLS_REFERENCE.md             ← 47+ tools catalog
    ├── HELM_CHART_REFERENCE.md        ← Chart values
    ├── TERMINAL_BACKENDS.md           ← 6 backends
    └── CHAT_COMMANDS.md               ← Built-in commands
```

## 🔍 Quick Lookup

### Common Tasks

**Deploy Hermes**
→ [Quick Start](./docs/guides/QUICK_START.md)

**Configure Telegram Bot**
→ [Messaging Platforms](./docs/guides/MESSAGING_PLATFORMS.md) → Telegram

**Set Up API Server**
→ [API Server Integration](./docs/guides/API_SERVER_INTEGRATION.md)

**Customize Agent Personality**
→ [SOUL.md Customization](./docs/configuration/SOUL_CUSTOMIZATION.md)

**Enable/Disable Tools**
→ [Tools Reference](./docs/reference/TOOLS_REFERENCE.md)

**Production Hardening**
→ [Production Deployment](./docs/deployment/PRODUCTION_DEPLOYMENT.md)

**Troubleshoot Issues**
→ [Troubleshooting](./docs/guides/TROUBLESHOOTING.md)

**Configure Environment**
→ [Environment Variables](./docs/configuration/ENVIRONMENT_VARIABLES.md)

**Understand config.yaml**
→ [Configuration Reference](./docs/configuration/CONFIG_REFERENCE.md)

## 📊 Documentation Statistics

- **Total Files**: 18 markdown files
- **Total Lines**: 13,698 lines
- **Configuration Examples**: 50+
- **Code Snippets**: 100+
- **Deployment Patterns**: 10+
- **Platform Integrations**: 15+
- **Tools Documented**: 47+
- **Environment Variables**: 60+
- **Common Issues**: 20+

## 🌐 Supported Platforms

### Messaging (15+)
Telegram, Discord, Slack, WhatsApp, Signal, Email, Matrix, Mattermost, Feishu/Lark, WeCom, Weixin, DingTalk, BlueBubbles, QQ, SMS

### Terminal Backends (6)
Local, Docker, SSH, Modal, Daytona, Singularity

### LLM Providers
OpenRouter, OpenAI, Anthropic, Local (Ollama), OpenAI-compatible APIs

### Deployment
Kubernetes (Kind, minikube, cloud), Docker, Docker Compose, Local

## 🔗 External Resources

- **Official Docs**: https://hermes-agent.nousresearch.com/docs/
- **GitHub**: https://github.com/NousResearch/hermes-agent
- **Helm Chart**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Discord**: https://discord.gg/NousResearch
- **Skills Hub**: https://agentskills.io

## 📌 Version Information

- **Hermes Version**: v0.10.0 (April 16, 2026)
- **Documentation Version**: 1.0 (April 18, 2026)
- **Python**: 3.11+ required
- **Kubernetes**: 1.20+ recommended
- **Helm**: 3.0+ required

## ✨ Key Features

✅ **Comprehensive** — All 9 areas fully documented  
✅ **Production-Ready** — Security hardening and best practices  
✅ **Well-Organized** — Clear structure with cross-references  
✅ **Code Examples** — 100+ configuration and deployment examples  
✅ **Step-by-Step** — Easy-to-follow setup instructions  
✅ **Troubleshooting** — Solutions for 20+ common issues  
✅ **Best Practices** — Operational guidelines for teams  
✅ **Version-Specific** — Aligned with Hermes v0.10.0  

## 🎯 Next Steps

1. **Choose your path** based on your role above
2. **Read the relevant guide** for your use case
3. **Follow the step-by-step instructions**
4. **Reference the configuration guides** as needed
5. **Check troubleshooting** if you hit any issues

---

**Ready to get started?** Pick your path above and dive in! 🚀

**Questions?** Check [Troubleshooting](./docs/guides/TROUBLESHOOTING.md) or visit the [Discord community](https://discord.gg/NousResearch).

**Last Updated**: April 18, 2026
