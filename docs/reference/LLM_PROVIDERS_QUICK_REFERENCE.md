# Hermes Agent - LLM Providers Quick Reference

**Last Updated:** April 2026

---

## Provider Quick Setup

### OpenRouter (Recommended)
```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"
```
**Env:** `OPENROUTER_API_KEY=sk-or-v1-...`

### Anthropic Direct
```yaml
model:
  default: "claude-opus-4.6"
  provider: "anthropic"
```
**Env:** `ANTHROPIC_API_KEY=sk-ant-...`

### Google Gemini
```yaml
model:
  default: "gemini-3-flash-preview"
  provider: "gemini"
```
**Env:** `GOOGLE_API_KEY=AIzaSy...`

### GitHub Copilot
```yaml
model:
  default: "gpt-4-turbo"
  provider: "copilot"
```
**Env:** `COPILOT_GITHUB_TOKEN=ghp_...`

### Nous Portal
```yaml
model:
  default: "nous-hermes-3-405b"
  provider: "nous"
```
**Setup:** `hermes login`

### Local (Ollama)
```yaml
model:
  default: "llama2"
  provider: "ollama"
```
**Runs on:** `http://localhost:11434`

### Local (LM Studio)
```yaml
model:
  default: "mistral-7b"
  provider: "lmstudio"
```
**Runs on:** `http://localhost:1234`

### Custom Endpoint
```yaml
model:
  default: "gpt-3.5-turbo"
  provider: "custom"
  base_url: "http://localhost:8000/v1"
```
**Env:** `OPENAI_API_KEY=sk-...`

---

## All Providers at a Glance

| Provider | Setup | Env Var | Best For |
|----------|-------|---------|----------|
| openrouter | Easy | `OPENROUTER_API_KEY` | 100+ models, price optimization |
| anthropic | Easy | `ANTHROPIC_API_KEY` | Direct Claude access |
| gemini | Easy | `GOOGLE_API_KEY` | Vision, multimodal |
| copilot | Easy | `COPILOT_GITHUB_TOKEN` | GitHub Models, free tier |
| nous | OAuth | `hermes login` | Nous-hosted models |
| nous-api | Easy | `NOUS_API_KEY` | Nous API key |
| zai | Easy | `GLM_API_KEY` | Chinese models (GLM) |
| kimi-coding | Easy | `KIMI_API_KEY` | Long context, Chinese |
| minimax | Easy | `MINIMAX_API_KEY` | Chinese market |
| arcee | Easy | `ARCEEAI_API_KEY` | Enterprise, specialized |
| deepseek | Easy | `DEEPSEEK_API_KEY` | Cost-effective, reasoning |
| ollama | Local | — | Local inference |
| lmstudio | Local | — | Local inference |
| vllm | Local | — | Local inference |
| llamacpp | Local | — | Local inference |
| custom | Manual | `OPENAI_API_KEY` | Any OpenAI-compatible |

---

## Model Selection

### Best Overall
- `anthropic/claude-opus-4.6` (OpenRouter)
- `claude-opus-4.6` (Anthropic direct)

### Best Value
- `google/gemini-3-flash-preview` (OpenRouter)
- `anthropic/claude-haiku-4.5` (OpenRouter)

### Best Vision
- `google/gemini-3-flash-preview`
- `anthropic/claude-opus-4.6`

### Best Reasoning
- `anthropic/claude-opus-4.6`
- `deepseek-reasoner`

### Fastest
- `google/gemini-2.5-flash`
- `anthropic/claude-haiku-4.5`

### Cheapest
- `google/gemini-2.5-flash`
- `anthropic/claude-haiku-4.5`

---

## Fallback Configuration

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"

fallback_model_2:
  provider: "openrouter"
  model: "google/gemini-3-flash-preview"
```

---

## Auxiliary Models

### Vision (Image Analysis)
```yaml
auxiliary:
  vision:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120
```

### Web Extraction
```yaml
auxiliary:
  web_extract:
    provider: "openrouter"
    model: "anthropic/claude-sonnet-4"
    timeout: 360
```

### Compression
```yaml
auxiliary:
  compression:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120
```

### Approval
```yaml
auxiliary:
  approval:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30
```

---

## Reasoning Effort

```yaml
agent:
  reasoning_effort: "medium"  # none | minimal | low | medium | high | xhigh
```

| Level | Speed | Quality | Cost |
|-------|-------|---------|------|
| none | ⚡⚡⚡ | ⭐ | $ |
| minimal | ⚡⚡ | ⭐⭐ | $$ |
| low | ⚡ | ⭐⭐⭐ | $$$ |
| medium | ⚡ | ⭐⭐⭐⭐ | $$$$ |
| high | 🐢 | ⭐⭐⭐⭐⭐ | $$$$$ |
| xhigh | 🐢🐢 | ⭐⭐⭐⭐⭐ | $$$$$$ |

---

## Timeouts

```yaml
agent:
  api_timeout: 120                    # LLM API timeout (seconds)

terminal:
  timeout: 180                        # Command timeout (seconds)

browser:
  inactivity_timeout: 120             # Browser timeout (seconds)

code_execution:
  timeout: 300                        # Script timeout (seconds)

auxiliary:
  vision:
    timeout: 120
  web_extract:
    timeout: 360
  compression:
    timeout: 120
  approval:
    timeout: 30
```

---

## Credential Pool

```yaml
credential_pool:
  strategy: "round_robin"             # fill_first | round_robin | least_used | random
  keys:
    - "sk-or-v1-key1..."
    - "sk-or-v1-key2..."
    - "sk-or-v1-key3..."
  retry_on_failure: true
  retry_count: 3
```

---

## Provider Routing (OpenRouter)

```yaml
provider_routing:
  sort: "price"                       # price | throughput | latency
  only: ["anthropic", "google"]       # Whitelist
  ignore: ["deepinfra"]               # Blacklist
  require_parameters: true            # Support all params
  data_collection: "deny"             # No data collection
```

---

## Smart Model Routing

```yaml
smart_model_routing:
  enabled: true
  max_simple_chars: 160               # Threshold for "simple"
  max_simple_words: 28
  cheap_model:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"
```

---

## CLI Overrides

```bash
# Override model
hermes chat --model anthropic/claude-sonnet-4

# Override provider
hermes chat --provider openrouter

# Override base URL
hermes chat --base-url http://localhost:8000/v1

# Override reasoning
hermes chat --reasoning-effort high

# Combine
hermes chat --model gpt-4-turbo --provider custom --base-url http://localhost:8000/v1
```

---

## Config Commands

```bash
# View config
hermes config

# Set value
hermes config set model.default anthropic/claude-opus-4.6

# Check config
hermes config check

# Migrate config
hermes config migrate

# View .env path
hermes config env-path
```

---

## Environment Variables

### LLM Providers
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

---

## Common Configurations

### Production (OpenRouter)
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

### Cost-Optimized
```yaml
model:
  default: "google/gemini-3-flash-preview"
  provider: "openrouter"

smart_model_routing:
  enabled: true
  cheap_model:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"

agent:
  reasoning_effort: "low"
```

### Local Development
```yaml
model:
  default: "mistral-7b"
  provider: "ollama"

fallback_model:
  provider: "lmstudio"
  model: "neural-chat-7b"

agent:
  reasoning_effort: "low"
```

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

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Provider not found | Check spelling, verify API key in `.env` |
| Auth failed | Verify API key, check expiration, check permissions |
| Model not available | Check model name format, verify provider has model |
| Timeout | Increase `api_timeout`, check network, try different provider |
| Context exceeded | Enable compression, reduce memory limits, use larger model |
| Rate limited | Use credential pool, add fallback provider, reduce requests |

---

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)
