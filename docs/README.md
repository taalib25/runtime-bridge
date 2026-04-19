# Hermes Agent Documentation

Complete documentation for Hermes Agent configuration, deployment, and security.

## 📚 Documentation Structure

```
docs/
├── README.md                          # This file
├── configuration/                     # Configuration guides
│   ├── CONFIG_REFERENCE.md           # Complete config reference
│   ├── GATEWAY_CONFIGURATION.md      # Gateway & security settings
│   ├── ENVIRONMENT_VARIABLES.md      # All environment variables
│   └── SOUL_CUSTOMIZATION.md         # Agent personality customization
├── deployment/                        # Deployment guides
│   ├── DEPLOYMENT_GUIDE.md           # Deployment procedures
│   └── QUICK_START.md                # Quick start guide
├── guides/                            # How-to guides
│   ├── API_SERVER_INTEGRATION.md     # API server setup
│   ├── MESSAGING_PLATFORMS.md        # Messaging platform setup
│   └── QUICK_START.md                # Quick start
├── reference/                         # Reference documentation
│   ├── SKILLS_COMPREHENSIVE_GUIDE.md # Skills reference
│   ├── SKILLS_QUICK_REFERENCE.md     # Skills quick ref
│   ├── TOOLS_INDEX.md                # Tools index
│   ├── TOOLS_REFERENCE.md            # Tools reference
│   └── TOOL_PARAMETERS_SCHEMA.md     # Tool parameters
└── security/                          # Security documentation
    ├── README.md                      # Security quick reference
    └── SECURITY_AND_BACKEND_COMPREHENSIVE.md  # Comprehensive security guide
```

## 🔐 Security Documentation

### Quick Links

- **[Security Quick Reference](./security/README.md)** — Fast lookup for security settings
- **[Comprehensive Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md)** — Exhaustive documentation of all 15 security features

### Security Features Documented

1. ✅ Dangerous Command Approval System
2. ✅ YOLO Mode
3. ✅ Command Allowlist
4. ✅ Container Isolation & Docker Security
5. ✅ Terminal Backends (6 types)
6. ✅ Environment Variable Passthrough
7. ✅ Credential File Passthrough
8. ✅ MCP Credential Handling
9. ✅ SSRF Protection
10. ✅ Tirith Pre-Exec Scanning
11. ✅ Context File Injection Protection
12. ✅ Website Blocklist
13. ✅ Privacy Settings & PII Redaction
14. ✅ Group Session Isolation
15. ✅ Unauthorized DM Behavior

## ⚙️ Configuration Documentation

### Quick Links

- **[Configuration Reference](./configuration/CONFIG_REFERENCE.md)** — Complete config.yaml reference
- **[Gateway Configuration](./configuration/GATEWAY_CONFIGURATION.md)** — Messaging platform setup
- **[Environment Variables](./configuration/ENVIRONMENT_VARIABLES.md)** — All environment variables
- **[SOUL Customization](./configuration/SOUL_CUSTOMIZATION.md)** — Agent personality

### Key Configuration Topics

- Model configuration (providers, routing, fallback)
- Terminal backend configuration (local, docker, ssh, modal, daytona, singularity)
- Memory configuration (short-term, long-term, hybrid)
- Gateway configuration (Telegram, Discord, Slack, etc.)
- Approval system configuration
- Security settings
- Logging configuration

## 🚀 Deployment Documentation

### Quick Links

- **[Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md)** — Full deployment procedures
- **[Quick Start](./guides/QUICK_START.md)** — Get started in 5 minutes

### Deployment Topics

- Installation
- Configuration setup
- Gateway setup
- Security configuration
- Monitoring and logging
- Troubleshooting

## 🛠️ Integration Guides

### Quick Links

- **[API Server Integration](./guides/API_SERVER_INTEGRATION.md)** — REST API setup
- **[Messaging Platforms](./guides/MESSAGING_PLATFORMS.md)** — Platform-specific setup

## 📖 Reference Documentation

### Quick Links

- **[Skills Comprehensive Guide](./reference/SKILLS_COMPREHENSIVE_GUIDE.md)** — All available skills
- **[Skills Quick Reference](./reference/SKILLS_QUICK_REFERENCE.md)** — Quick skill lookup
- **[Tools Index](./reference/TOOLS_INDEX.md)** — All available tools
- **[Tools Reference](./reference/TOOLS_REFERENCE.md)** — Tool documentation
- **[Tool Parameters Schema](./reference/TOOL_PARAMETERS_SCHEMA.md)** — Tool parameter schemas

## 🎯 Getting Started

### For New Users

1. Start with [Quick Start Guide](./guides/QUICK_START.md)
2. Review [Configuration Reference](./configuration/CONFIG_REFERENCE.md)
3. Set up your messaging platform from [Messaging Platforms](./guides/MESSAGING_PLATFORMS.md)
4. Configure security from [Security Quick Reference](./security/README.md)

### For Operators

1. Review [Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md)
2. Configure security from [Comprehensive Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md)
3. Set up monitoring and logging
4. Review security checklist regularly

### For Developers

1. Review [API Server Integration](./guides/API_SERVER_INTEGRATION.md)
2. Reference [Tools Reference](./reference/TOOLS_REFERENCE.md)
3. Review [Skills Comprehensive Guide](./reference/SKILLS_COMPREHENSIVE_GUIDE.md)
4. Check [Tool Parameters Schema](./reference/TOOL_PARAMETERS_SCHEMA.md)

## 📋 Configuration Checklist

Before deploying to production:

- [ ] Review [Security Quick Reference](./security/README.md)
- [ ] Configure approval system
- [ ] Set up terminal backend
- [ ] Configure messaging platforms
- [ ] Enable security features
- [ ] Set up logging
- [ ] Configure monitoring
- [ ] Test all features
- [ ] Review security checklist

## 🔍 Finding What You Need

### By Topic

| Topic | Document |
|-------|----------|
| Approval system | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md#dangerous-command-approval-system) |
| Terminal backends | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md#terminal-backends) |
| Docker security | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md#container-isolation--docker-security) |
| SSRF protection | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md#ssrf-protection) |
| PII redaction | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md#privacy-settings--pii-redaction) |
| Messaging setup | [Messaging Platforms](./guides/MESSAGING_PLATFORMS.md) |
| API integration | [API Server Integration](./guides/API_SERVER_INTEGRATION.md) |
| Skills | [Skills Guide](./reference/SKILLS_COMPREHENSIVE_GUIDE.md) |
| Tools | [Tools Reference](./reference/TOOLS_REFERENCE.md) |

### By Use Case

| Use Case | Start Here |
|----------|-----------|
| Quick setup | [Quick Start](./guides/QUICK_START.md) |
| Production deployment | [Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md) |
| Security hardening | [Security Guide](./security/SECURITY_AND_BACKEND_COMPREHENSIVE.md) |
| Messaging integration | [Messaging Platforms](./guides/MESSAGING_PLATFORMS.md) |
| API integration | [API Server Integration](./guides/API_SERVER_INTEGRATION.md) |
| Custom configuration | [Configuration Reference](./configuration/CONFIG_REFERENCE.md) |
| Skill development | [Skills Guide](./reference/SKILLS_COMPREHENSIVE_GUIDE.md) |

## 📞 Support

For issues or questions:

1. Check the relevant documentation section
2. Review troubleshooting guides
3. Check logs in `~/.hermes/logs/`
4. Review configuration in `~/.hermes/config.yaml`

## 📝 Documentation Standards

All documentation includes:

- ✅ Overview and purpose
- ✅ Configuration examples
- ✅ Environment variables
- ✅ Best practices
- ✅ Troubleshooting
- ✅ Related documentation links

## 🔄 Documentation Updates

Documentation is updated regularly. Check the "Last Updated" date in each document.

Current version: **April 18, 2026**  
Hermes version: **v0.10.0+**

---

**Last Updated:** April 18, 2026  
**Version:** 1.0
