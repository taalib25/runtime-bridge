# 🚀 Hermes Agent Documentation - START HERE

**Welcome to the Hermes Agent documentation!**

This guide will help you find exactly what you need.

---

## 🎯 What Are You Trying to Do?

### 👤 I'm New to Hermes Agent
**Start here for a quick introduction:**

1. **[5-Minute Quick Start](docs/guides/QUICK_START.md)** - Get up and running in 5 minutes
2. **[Quick Start Tutorials](docs/guides/QUICK_START_TUTORIALS.md)** - 8 step-by-step tutorials for common scenarios
3. **[Messaging Platforms](docs/guides/MESSAGING_PLATFORMS.md)** - Setup guides for all 16 platforms

### 🔧 I Want to Set Up a Specific Platform
**Choose your platform:**

- **[Telegram](docs/guides/MESSAGING_PLATFORMS.md#telegram)** - Easy setup
- **[Discord](docs/guides/MESSAGING_PLATFORMS.md#discord)** - Medium setup
- **[Slack](docs/guides/MESSAGING_PLATFORMS.md#slack)** - Medium setup
- **[WhatsApp](docs/guides/MESSAGING_PLATFORMS.md#whatsapp)** - Hard setup
- **[Signal](docs/guides/MESSAGING_PLATFORMS.md#signal)** - Hard setup
- **[Email](docs/guides/MESSAGING_PLATFORMS.md#email)** - Easy setup
- **[SMS](docs/guides/MESSAGING_PLATFORMS.md#sms)** - Medium setup
- **[Matrix](docs/guides/MESSAGING_PLATFORMS.md#matrix)** - Medium setup
- **[Mattermost](docs/guides/MESSAGING_PLATFORMS.md#mattermost)** - Medium setup
- **[Feishu/Lark](docs/guides/MESSAGING_PLATFORMS.md#feishulark)** - Medium setup
- **[WeCom](docs/guides/MESSAGING_PLATFORMS.md#wecom)** - Medium setup
- **[Weixin (WeChat)](docs/guides/MESSAGING_PLATFORMS.md#weixin-wechat)** - Hard setup
- **[DingTalk](docs/guides/MESSAGING_PLATFORMS.md#dingtalk)** - Medium setup
- **[BlueBubbles](docs/guides/MESSAGING_PLATFORMS.md#bluebubbles)** - Hard setup
- **[QQ Bot](docs/guides/MESSAGING_PLATFORMS.md#qq-bot)** - Hard setup
- **[Home Assistant](docs/guides/MESSAGING_PLATFORMS.md#home-assistant)** - Easy setup

### 🐳 I Want to Deploy with Docker
**Follow these guides:**

1. **[Docker Deployment Tutorial](docs/guides/QUICK_START_TUTORIALS.md#docker-deployment)** - Step-by-step Docker setup
2. **[Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md#docker-deployment)** - Production-grade Docker setup

### ☸️ I Want to Deploy with Kubernetes
**Follow these guides:**

1. **[Kubernetes Deployment Tutorial](docs/guides/QUICK_START_TUTORIALS.md#kubernetes-deployment)** - Step-by-step Kubernetes setup
2. **[Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md#kubernetes-deployment)** - Production-grade Kubernetes setup

### 🌐 I Want to Use Open WebUI
**Follow this guide:**

1. **[Open WebUI Integration](docs/guides/QUICK_START_TUTORIALS.md#open-webui-integration)** - Setup Open WebUI with Hermes

### ⚙️ I Need Configuration Help
**Reference these guides:**

1. **[Configuration Reference](docs/configuration/CONFIG_REFERENCE.md)** - Complete configuration options
2. **[Environment Variables](docs/configuration/ENVIRONMENT_VARIABLES.md)** - All 60+ environment variables
3. **[Gateway Configuration](docs/configuration/GATEWAY_CONFIGURATION.md)** - Gateway setup and security
4. **[SOUL Customization](docs/configuration/SOUL_CUSTOMIZATION.md)** - Customize agent personality

### 🚨 Something's Not Working
**Troubleshooting guides:**

1. **[Troubleshooting Guide](docs/guides/TROUBLESHOOTING.md)** - Common issues and solutions
2. **[Best Practices](docs/guides/BEST_PRACTICES.md)** - Avoid common mistakes

### 📊 I'm Deploying to Production
**Production guides:**

1. **[Best Practices](docs/guides/BEST_PRACTICES.md)** - Production best practices
2. **[Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md)** - Complete production setup
3. **[Security Hardening](docs/guides/BEST_PRACTICES.md#security-best-practices)** - Security guidelines

### 🔍 I Need API Documentation
**API reference:**

1. **[API Server Integration](docs/guides/API_SERVER_INTEGRATION.md)** - Complete API documentation
2. **[Tools Reference](docs/reference/TOOLS_REFERENCE.md)** - All available tools
3. **[Skills Reference](docs/reference/SKILLS_COMPREHENSIVE_GUIDE.md)** - All available skills

### 📚 I Want to Learn Everything
**Complete documentation:**

1. **[Platform Integration Index](PLATFORM_INTEGRATION_INDEX.md)** - Master index of all documentation
2. **[Documentation Index](docs/reference/DOCUMENTATION_INDEX.md)** - Complete documentation structure

---

## 📖 Documentation Structure

```
docs/
├── guides/                          # User guides and tutorials
│   ├── QUICK_START.md              # 5-minute quick start
│   ├── QUICK_START_TUTORIALS.md    # 8 step-by-step tutorials
│   ├── MESSAGING_PLATFORMS.md      # 16 platform setup guides
│   ├── API_SERVER_INTEGRATION.md   # API server & Open WebUI
│   ├── BEST_PRACTICES.md           # Production best practices
│   └── TROUBLESHOOTING.md          # Troubleshooting guide
│
├── configuration/                   # Configuration reference
│   ├── CONFIG_REFERENCE.md         # Configuration options
│   ├── ENVIRONMENT_VARIABLES.md    # Environment variables
│   ├── GATEWAY_CONFIGURATION.md    # Gateway setup
│   └── SOUL_CUSTOMIZATION.md       # Agent personality
│
├── deployment/                      # Deployment guides
│   ├── DEPLOYMENT_GUIDE.md         # Deployment overview
│   └── PRODUCTION_DEPLOYMENT.md    # Production setup
│
└── reference/                       # Reference documentation
    ├── TOOLS_REFERENCE.md          # Tools reference
    ├── SKILLS_COMPREHENSIVE_GUIDE.md # Skills reference
    └── ... (more reference docs)
```

---

## 🎓 Learning Paths

### Path 1: Quick Start (30 minutes)
1. [5-Minute Quick Start](docs/guides/QUICK_START.md)
2. [Platform Setup](docs/guides/MESSAGING_PLATFORMS.md) (choose your platform)
3. Test your setup

### Path 2: Docker Deployment (1 hour)
1. [Docker Deployment Tutorial](docs/guides/QUICK_START_TUTORIALS.md#docker-deployment)
2. [Configuration Reference](docs/configuration/CONFIG_REFERENCE.md)
3. Deploy and test

### Path 3: Kubernetes Deployment (2 hours)
1. [Kubernetes Deployment Tutorial](docs/guides/QUICK_START_TUTORIALS.md#kubernetes-deployment)
2. [Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md)
3. [Best Practices](docs/guides/BEST_PRACTICES.md)
4. Deploy and test

### Path 4: Production Setup (4 hours)
1. [Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md)
2. [Best Practices](docs/guides/BEST_PRACTICES.md)
3. [Security Hardening](docs/guides/BEST_PRACTICES.md#security-best-practices)
4. [Monitoring Setup](docs/guides/BEST_PRACTICES.md#monitoring--observability)
5. Deploy and verify

### Path 5: API Integration (2 hours)
1. [API Server Integration](docs/guides/API_SERVER_INTEGRATION.md)
2. [Open WebUI Integration](docs/guides/QUICK_START_TUTORIALS.md#open-webui-integration)
3. [Tools Reference](docs/reference/TOOLS_REFERENCE.md)
4. Test API endpoints

---

## 🔗 Quick Links

### Getting Started
- [Quick Start](docs/guides/QUICK_START.md)
- [Quick Start Tutorials](docs/guides/QUICK_START_TUTORIALS.md)
- [Messaging Platforms](docs/guides/MESSAGING_PLATFORMS.md)

### Configuration
- [Configuration Reference](docs/configuration/CONFIG_REFERENCE.md)
- [Environment Variables](docs/configuration/ENVIRONMENT_VARIABLES.md)
- [Gateway Configuration](docs/configuration/GATEWAY_CONFIGURATION.md)

### Deployment
- [Docker Deployment](docs/guides/QUICK_START_TUTORIALS.md#docker-deployment)
- [Kubernetes Deployment](docs/guides/QUICK_START_TUTORIALS.md#kubernetes-deployment)
- [Production Deployment](docs/deployment/PRODUCTION_DEPLOYMENT.md)

### Operations
- [Best Practices](docs/guides/BEST_PRACTICES.md)
- [Troubleshooting](docs/guides/TROUBLESHOOTING.md)
- [API Server Integration](docs/guides/API_SERVER_INTEGRATION.md)

### Reference
- [Tools Reference](docs/reference/TOOLS_REFERENCE.md)
- [Skills Reference](docs/reference/SKILLS_COMPREHENSIVE_GUIDE.md)
- [Platform Integration Index](PLATFORM_INTEGRATION_INDEX.md)

---

## 📊 Documentation Statistics

- **28 markdown files**
- **19,745 lines of content**
- **508 KB total size**
- **16 messaging platforms documented**
- **150+ code examples**
- **60+ configuration examples**
- **8 step-by-step tutorials**
- **20+ troubleshooting scenarios**
- **50+ best practices**

---

## 🌐 Building Documentation Website

### Local Development
```bash
pip install mkdocs mkdocs-material
mkdocs serve
# Visit http://localhost:8000
```

### Build Static Site
```bash
mkdocs build
# Output in site/ directory
```

### Deploy to GitHub Pages
```bash
mkdocs gh-deploy
# Site available at: https://NousResearch.github.io/hermes-agent/
```

---

## 📞 Need Help?

### Documentation
- **Main Index**: [PLATFORM_INTEGRATION_INDEX.md](PLATFORM_INTEGRATION_INDEX.md)
- **Troubleshooting**: [docs/guides/TROUBLESHOOTING.md](docs/guides/TROUBLESHOOTING.md)
- **Best Practices**: [docs/guides/BEST_PRACTICES.md](docs/guides/BEST_PRACTICES.md)

### Community
- **GitHub**: https://github.com/NousResearch/hermes-agent
- **Discord**: https://discord.gg/NousResearch
- **Issues**: https://github.com/NousResearch/hermes-agent/issues

### External Resources
- **Kubernetes**: https://kubernetes.io/docs/
- **Docker**: https://docs.docker.com/
- **Prometheus**: https://prometheus.io/docs/
- **OpenAI**: https://platform.openai.com/docs/

---

## ✨ What's New?

**Latest Documentation Updates (April 18, 2026):**

- ✨ **Quick Start Tutorials** - 8 step-by-step tutorials for common scenarios
- ✨ **Troubleshooting Guide** - Comprehensive troubleshooting procedures
- ✨ **Best Practices** - Production best practices and guidelines
- ✨ **Production Deployment** - Complete production deployment guide
- ✨ **MkDocs Website** - Documentation website configuration
- ✨ **Validation Report** - Comprehensive documentation validation

---

## 🎯 Next Steps

1. **Choose your path** - Pick a learning path above
2. **Follow the guide** - Step-by-step instructions
3. **Test your setup** - Verify everything works
4. **Explore more** - Read additional documentation as needed
5. **Get help** - Use troubleshooting guide if needed

---

**Happy learning! 🚀**

For the most up-to-date documentation, visit:
https://github.com/NousResearch/hermes-agent/tree/main/docs

