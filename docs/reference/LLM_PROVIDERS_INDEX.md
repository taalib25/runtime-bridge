# Hermes Agent - LLM Providers Documentation Index

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)

---

## Documentation Overview

This comprehensive documentation covers every aspect of LLM provider configuration in Hermes Agent.

### Quick Navigation

- **New to Hermes?** → Start with [Quick Reference](#quick-reference)
- **Need specific provider setup?** → See [Provider Setup Guides](#provider-setup-guides)
- **Comparing providers?** → Check [Provider Matrix](#provider-matrix)
- **Advanced configuration?** → Read [Comprehensive Guide](#comprehensive-guide)

---

## Documentation Files

### 1. **LLM_PROVIDERS_QUICK_REFERENCE.md** (Quick Reference)
**Best for:** Getting started quickly, common configurations

**Contents:**
- Provider quick setup (8 most popular)
- All providers at a glance
- Model selection (best overall, best value, best vision, etc.)
- Fallback configuration
- Auxiliary models
- Reasoning effort levels
- Timeouts
- Credential pool strategies
- Provider routing
- Smart model routing
- CLI overrides
- Config commands
- Environment variables
- Common configurations (production, cost-optimized, local, multi-provider)
- Troubleshooting

**When to use:**
- First-time setup
- Quick lookups
- Common configurations
- CLI reference

### 2. **LLM_PROVIDERS_COMPREHENSIVE.md** (Complete Reference)
**Best for:** Exhaustive documentation, all options explained

**Contents:**
- Provider overview (27 providers)
- Provider setup & configuration (detailed for each provider)
- Model selection & configuration
- Fallback model configuration
- Auxiliary model configuration (8 auxiliary tasks)
- Credential pool strategies (4 strategies)
- Reasoning effort settings (6 levels)
- Tool-use enforcement
- Streaming timeouts
- API timeout settings
- Custom provider configuration
- Provider routing & smart selection
- Configuration examples (5 real-world examples)
- Troubleshooting

**When to use:**
- Deep understanding needed
- Custom configurations
- Advanced features
- Troubleshooting complex issues

### 3. **LLM_PROVIDERS_MATRIX.md** (Comparison Matrix)
**Best for:** Comparing providers, decision making

**Contents:**
- Authentication & setup comparison
- Capabilities matrix (vision, reasoning, tool use, streaming, function calling)
- Reasoning effort support
- Performance characteristics
- Model availability
- Context length support
- Pricing comparison
- Regional availability
- Provider selection guide
- Quick decision tree

**When to use:**
- Choosing a provider
- Comparing capabilities
- Cost analysis
- Regional requirements

---

## Provider Setup Guides

### Aggregators (100+ Models)
- **OpenRouter** — Recommended, 100+ models, price optimization
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#openrouter-setup)
  - [Quick Setup](LLM_PROVIDERS_QUICK_REFERENCE.md#openrouter-recommended)

### Direct Providers
- **Anthropic** — Direct Claude access
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#anthropic-direct-setup)
  - [Quick Setup](LLM_PROVIDERS_QUICK_REFERENCE.md#anthropic-direct)

- **Google Gemini** — Vision, multimodal
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#google-gemini-setup)
  - [Quick Setup](LLM_PROVIDERS_QUICK_REFERENCE.md#google-gemini)

- **GitHub Copilot** — GitHub Models, free tier
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#github-copilot-setup)
  - [Quick Setup](LLM_PROVIDERS_QUICK_REFERENCE.md#github-copilot)

### Portal-Based
- **Nous Portal** — OAuth integration
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#nous-portal-setup)
  - [Quick Setup](LLM_PROVIDERS_QUICK_REFERENCE.md#nous-portal)

### Chinese Providers
- **ZhipuAI (z.ai)** — GLM models
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#zai--zhipuai-glm-setup)

- **Kimi (Moonshot)** — Long context
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#kimi--moonshot-ai-setup)

- **MiniMax** — Multimodal
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#minimax-setup)

### Specialized
- **Arcee AI** — Enterprise, specialized models
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#arcee-ai-trinity-setup)

- **DeepSeek** — Cost-effective, reasoning
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#deepseek-setup)

### Local Inference
- **Ollama** — Local inference
  - [Setup Guide](LLM_PROVIDERS_QUICK_REFERENCE.md#local-ollama)

- **LM Studio** — Local inference with GUI
  - [Setup Guide](LLM_PROVIDERS_QUICK_REFERENCE.md#local-lm-studio)

- **vLLM** — High-throughput local
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#custom-openai-compatible-setup)

- **llama.cpp** — Lightweight local
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#custom-openai-compatible-setup)

### Custom Endpoints
- **Any OpenAI-Compatible** — Self-hosted, private
  - [Setup Guide](LLM_PROVIDERS_COMPREHENSIVE.md#custom-openai-compatible-setup)

---

## Configuration Topics

### Model Configuration
- [Basic Setup](LLM_PROVIDERS_COMPREHENSIVE.md#basic-model-configuration)
- [Model Naming Conventions](LLM_PROVIDERS_COMPREHENSIVE.md#model-naming-conventions)
- [Context Length](LLM_PROVIDERS_COMPREHENSIVE.md#context-length-configuration)
- [Max Tokens](LLM_PROVIDERS_COMPREHENSIVE.md#max-tokens-configuration)
- [Runtime Switching](LLM_PROVIDERS_COMPREHENSIVE.md#model-switching-at-runtime)
- [Capabilities Detection](LLM_PROVIDERS_COMPREHENSIVE.md#model-capabilities-detection)

### Fallback Configuration
- [Basic Fallback](LLM_PROVIDERS_COMPREHENSIVE.md#basic-fallback-setup)
- [Fallback Triggers](LLM_PROVIDERS_COMPREHENSIVE.md#fallback-triggers)
- [Multi-Level Chains](LLM_PROVIDERS_COMPREHENSIVE.md#multi-level-fallback-chain)
- [Fallback Behavior](LLM_PROVIDERS_COMPREHENSIVE.md#fallback-behavior)

### Auxiliary Models
- [Vision Analysis](LLM_PROVIDERS_COMPREHENSIVE.md#vision-analysis)
- [Web Extraction](LLM_PROVIDERS_COMPREHENSIVE.md#web-extraction--summarization)
- [Command Approval](LLM_PROVIDERS_COMPREHENSIVE.md#dangerous-command-approval)
- [Context Compression](LLM_PROVIDERS_COMPREHENSIVE.md#context-compression)
- [Session Search](LLM_PROVIDERS_COMPREHENSIVE.md#session-search)
- [Skills Hub](LLM_PROVIDERS_COMPREHENSIVE.md#skills-hub)
- [MCP Dispatch](LLM_PROVIDERS_COMPREHENSIVE.md#mcp-tool-dispatch)
- [Memory Flush](LLM_PROVIDERS_COMPREHENSIVE.md#memory-flush)

### Credential Management
- [Credential Pool Strategies](LLM_PROVIDERS_COMPREHENSIVE.md#credential-pool-strategies)
- [Fill-First Strategy](LLM_PROVIDERS_COMPREHENSIVE.md#fill-first-strategy)
- [Round-Robin Strategy](LLM_PROVIDERS_COMPREHENSIVE.md#round-robin-strategy)
- [Least-Used Strategy](LLM_PROVIDERS_COMPREHENSIVE.md#least-used-strategy)
- [Random Strategy](LLM_PROVIDERS_COMPREHENSIVE.md#random-strategy)

### Advanced Features
- [Reasoning Effort](LLM_PROVIDERS_COMPREHENSIVE.md#reasoning-effort-settings)
- [Tool-Use Enforcement](LLM_PROVIDERS_COMPREHENSIVE.md#tool-use-enforcement)
- [Streaming Timeouts](LLM_PROVIDERS_COMPREHENSIVE.md#streaming-timeouts)
- [API Timeouts](LLM_PROVIDERS_COMPREHENSIVE.md#api-timeout-settings)
- [Provider Routing](LLM_PROVIDERS_COMPREHENSIVE.md#provider-routing--smart-selection)
- [Smart Model Routing](LLM_PROVIDERS_COMPREHENSIVE.md#smart-model-routing)

---

## Real-World Examples

### Production Setup
```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"

agent:
  reasoning_effort: "high"
  api_timeout: 120
```
[Full Example](LLM_PROVIDERS_COMPREHENSIVE.md#example-1-production-setup-openrouter)

### Cost-Optimized Setup
```yaml
model:
  default: "google/gemini-3-flash-preview"
  provider: "openrouter"

smart_model_routing:
  enabled: true
  cheap_model:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"
```
[Full Example](LLM_PROVIDERS_COMPREHENSIVE.md#example-2-cost-optimized-setup)

### Local Development
```yaml
model:
  default: "mistral-7b"
  provider: "ollama"

fallback_model:
  provider: "lmstudio"
  model: "neural-chat-7b"
```
[Full Example](LLM_PROVIDERS_COMPREHENSIVE.md#example-3-local-development-setup)

### Multi-Provider Failover
```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "anthropic"

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-opus-4.6"

fallback_model_2:
  provider: "openrouter"
  model: "google/gemini-3-flash-preview"
```
[Full Example](LLM_PROVIDERS_COMPREHENSIVE.md#example-4-multi-provider-failover)

### Enterprise Setup
[Full Example](LLM_PROVIDERS_COMPREHENSIVE.md#example-5-enterprise-setup-multiple-providers)

---

## Environment Variables Reference

### LLM Provider Keys
```bash
OPENROUTER_API_KEY=sk-or-v1-...
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_API_KEY=AIzaSy...
COPILOT_GITHUB_TOKEN=ghp_...
NOUS_API_KEY=...
GLM_API_KEY=...
KIMI_API_KEY=...
MINIMAX_API_KEY=...
ARCEEAI_API_KEY=...
DEEPSEEK_API_KEY=...
```

### Custom Endpoints
```bash
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=http://localhost:8000/v1
```

### Timeouts
```bash
HERMES_API_TIMEOUT=120
HERMES_STREAM_READ_TIMEOUT=30
HERMES_STREAM_STALE_TIMEOUT=60
TERMINAL_TIMEOUT=180
```

[Full Reference](LLM_PROVIDERS_COMPREHENSIVE.md#environment-variables-reference)

---

## Troubleshooting

### Common Issues

| Issue | Solution | Reference |
|-------|----------|-----------|
| Provider not found | Check spelling, verify API key | [Troubleshooting](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting) |
| Auth failed | Verify API key, check expiration | [Troubleshooting](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting) |
| Model not available | Check model name format | [Troubleshooting](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting) |
| Timeout | Increase api_timeout | [Troubleshooting](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting) |
| Context exceeded | Enable compression | [Troubleshooting](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting) |

[Full Troubleshooting Guide](LLM_PROVIDERS_COMPREHENSIVE.md#troubleshooting)

---

## CLI Commands

### Model Configuration
```bash
hermes chat --model anthropic/claude-sonnet-4
hermes chat --provider openrouter
hermes chat --base-url http://localhost:8000/v1
hermes chat --reasoning-effort high
```

### Config Management
```bash
hermes config                           # View config
hermes config set model.default ...     # Set value
hermes config check                     # Check config
hermes config migrate                   # Migrate config
hermes config env-path                  # Show .env path
```

[Full CLI Reference](LLM_PROVIDERS_QUICK_REFERENCE.md#cli-overrides)

---

## Provider Comparison

### By Use Case

**Best Overall Quality**
1. Anthropic Claude (OpenRouter or Direct)
2. Google Gemini
3. OpenRouter (access to all)

**Best Value**
1. DeepSeek
2. Google Gemini Flash
3. Anthropic Claude Haiku

**Best Vision**
1. Google Gemini
2. Anthropic Claude
3. OpenRouter

**Best Reasoning**
1. Anthropic Claude
2. DeepSeek
3. OpenRouter

**Best for Local**
1. Ollama
2. LM Studio
3. vLLM

**Best for China**
1. ZhipuAI (z.ai)
2. Kimi (Moonshot)
3. MiniMax

[Full Comparison Matrix](LLM_PROVIDERS_MATRIX.md)

---

## Getting Help

### Documentation
- [Quick Reference](LLM_PROVIDERS_QUICK_REFERENCE.md) — Fast lookups
- [Comprehensive Guide](LLM_PROVIDERS_COMPREHENSIVE.md) — Detailed explanations
- [Provider Matrix](LLM_PROVIDERS_MATRIX.md) — Comparisons

### External Resources
- [OpenRouter API Docs](https://openrouter.ai/docs)
- [Anthropic API Docs](https://docs.anthropic.com)
- [Google Gemini API Docs](https://ai.google.dev)
- [GitHub Models](https://github.com/marketplace/models)
- [Nous Portal](https://nous.ai)
- [DeepSeek API](https://platform.deepseek.com)

---

## Document Statistics

| Document | Lines | Sections | Examples |
|----------|-------|----------|----------|
| Quick Reference | ~400 | 20+ | 10+ |
| Comprehensive Guide | 1511 | 50+ | 5+ |
| Provider Matrix | 248 | 15+ | 1 |
| **Total** | **~2160** | **85+** | **15+** |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | April 2026 | Initial comprehensive documentation |

---

## Contributing

To update this documentation:

1. Edit the relevant markdown file
2. Update this index if adding new sections
3. Keep examples current with latest Hermes version
4. Test all configurations before committing

---

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)  
**Maintained by:** Hermes Agent Team

---

## Quick Links

- [Quick Reference](LLM_PROVIDERS_QUICK_REFERENCE.md)
- [Comprehensive Guide](LLM_PROVIDERS_COMPREHENSIVE.md)
- [Provider Matrix](LLM_PROVIDERS_MATRIX.md)
- [Main Documentation](README.md)
