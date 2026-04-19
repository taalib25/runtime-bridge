# Hermes Agent - Comprehensive LLM Provider Documentation

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)  
**Documentation Scope:** Complete provider and model configuration reference

---

## Table of Contents

1. [Provider Overview](#provider-overview)
2. [Provider Setup & Configuration](#provider-setup--configuration)
3. [Model Selection & Configuration](#model-selection--configuration)
4. [Fallback Model Configuration](#fallback-model-configuration)
5. [Auxiliary Model Configuration](#auxiliary-model-configuration)
6. [Credential Pool Strategies](#credential-pool-strategies)
7. [Reasoning Effort Settings](#reasoning-effort-settings)
8. [Tool-Use Enforcement](#tool-use-enforcement)
9. [Streaming Timeouts](#streaming-timeouts)
10. [API Timeout Settings](#api-timeout-settings)
11. [Custom Provider Configuration](#custom-provider-configuration)
12. [Provider Routing & Smart Selection](#provider-routing--smart-selection)
13. [Configuration Examples](#configuration-examples)

---

## Provider Overview

### Supported Providers Matrix

| Provider | Type | Auth Method | Base URL | Env Var | Vision | Reasoning | Notes |
|----------|------|-------------|----------|---------|--------|-----------|-------|
| **openrouter** | Aggregator | API Key | `https://openrouter.ai/v1` | `OPENROUTER_API_KEY` | ✓ | ✓ | 100+ models, recommended |
| **nous** | Portal | OAuth | Portal URL | — | ✓ | ✓ | Nous Portal OAuth (`hermes login`) |
| **nous-api** | Direct | API Key | Portal API | `NOUS_API_KEY` | ✓ | ✓ | Nous Portal API key |
| **anthropic** | Direct | API Key | `https://api.anthropic.com` | `ANTHROPIC_API_KEY` | ✓ | ✓ | Direct Anthropic API |
| **openai-codex** | OAuth | OAuth | ChatGPT | — | ✓ | ✗ | OpenAI Codex (`hermes auth`) |
| **copilot** | GitHub | Token | GitHub Models | `COPILOT_GITHUB_TOKEN` | ✓ | ✓ | GitHub Copilot / GitHub Models |
| **copilot-acp** | GitHub | ACP | GitHub Models | — | ✓ | ✓ | Copilot ACP mode |
| **gemini** | Direct | API Key | `https://generativelanguage.googleapis.com` | `GOOGLE_API_KEY` | ✓ | ✓ | Google AI Studio direct |
| **zai** | Direct | API Key | `https://open.bigmodel.cn` | `GLM_API_KEY` | ✓ | ✓ | z.ai / ZhipuAI GLM |
| **kimi-coding** | Direct | API Key | `https://api.moonshot.cn` | `KIMI_API_KEY` | ✓ | ✓ | Kimi / Moonshot AI |
| **kimi-coding-cn** | Direct | API Key | China endpoint | `KIMI_CN_API_KEY` | ✓ | ✓ | Kimi China endpoint |
| **minimax** | Direct | API Key | `https://api.minimax.chat` | `MINIMAX_API_KEY` | ✓ | ✓ | MiniMax global |
| **minimax-cn** | Direct | API Key | China endpoint | `MINIMAX_CN_API_KEY` | ✓ | ✓ | MiniMax China |
| **kilocode** | Gateway | API Key | Custom | `KILOCODE_API_KEY` | ✓ | ✓ | Kilo Code gateway |
| **xiaomi** | Direct | API Key | `https://api.xiaomi.com` | `XIAOMI_API_KEY` | ✓ | ✓ | Xiaomi MiMo |
| **arcee** | Direct | API Key | `https://api.arcee.ai` | `ARCEEAI_API_KEY` | ✓ | ✓ | Arcee AI Trinity |
| **huggingface** | Direct | Token | HF Inference | `HF_TOKEN` | ✓ | ✗ | Hugging Face Inference |
| **ollama-cloud** | Cloud | API Key | Cloud endpoint | `OLLAMA_API_KEY` | ✓ | ✗ | Ollama Cloud |
| **ai-gateway** | Aggregator | API Key | `https://api.vercel.ai` | `AI_GATEWAY_API_KEY` | ✓ | ✓ | Vercel AI Gateway |
| **custom** | OpenAI-compatible | API Key | Custom | `OPENAI_API_KEY` | ✓ | ✓ | Any OpenAI-compatible endpoint |
| **lmstudio** | Alias | — | `http://localhost:1234` | — | ✓ | ✗ | Alias for `custom` (LM Studio) |
| **ollama** | Alias | — | `http://localhost:11434` | — | ✓ | ✗ | Alias for `custom` (Ollama local) |
| **vllm** | Alias | — | `http://localhost:8000` | — | ✓ | ✗ | Alias for `custom` (vLLM) |
| **llamacpp** | Alias | — | `http://localhost:8080` | — | ✓ | ✗ | Alias for `custom` (llama.cpp) |
| **dashscope** | Direct | API Key | `https://dashscope.aliyuncs.com` | `DASHSCOPE_API_KEY` | ✓ | ✓ | Alibaba DashScope |
| **deepseek** | Direct | API Key | `https://api.deepseek.com` | `DEEPSEEK_API_KEY` | ✓ | ✓ | DeepSeek API |
| **opencode-zen** | Direct | API Key | Custom | `OPENCODE_ZEN_API_KEY` | ✓ | ✓ | OpenCode Zen |
| **opencode-go** | Direct | API Key | Custom | `OPENCODE_GO_API_KEY` | ✓ | ✓ | OpenCode Go |

### Provider Auto-Detection

When `provider: "auto"` is set, Hermes attempts detection in this order:

1. Check for `OPENROUTER_API_KEY` → use OpenRouter
2. Check for Nous Portal OAuth → use Nous
3. Check for `ANTHROPIC_API_KEY` → use Anthropic
4. Check for `COPILOT_GITHUB_TOKEN` → use Copilot
5. Check for `OPENAI_API_KEY` → use custom endpoint
6. Check for other provider keys in priority order
7. Fall back to built-in defaults

---

## Provider Setup & Configuration

### Basic Model Configuration

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "auto"                    # auto | openrouter | nous | anthropic | copilot | etc.
  base_url: "https://openrouter.ai/v1"  # Optional; auto-detected per provider
  # api_key: "sk-..."                 # Prefer .env file
  context_length: 131072              # Auto-detected; set manually if wrong
  max_tokens: 8192                    # Output cap (optional; use model default if unset)
```

### OpenRouter Setup

**Best for:** 100+ models, price optimization, provider routing

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"
  base_url: "https://openrouter.ai/v1"
  context_length: 131072
  max_tokens: 8192
```

**Environment:**
```bash
# ~/.hermes/.env
OPENROUTER_API_KEY=sk-or-v1-...
OPENROUTER_BASE_URL=https://openrouter.ai/v1  # Optional override
```

**Available Models (Sample):**
- `anthropic/claude-opus-4.6` — Best reasoning
- `anthropic/claude-sonnet-4` — Balanced
- `google/gemini-3-flash-preview` — Fast, cheap
- `openai/gpt-4-turbo` — Powerful
- `meta-llama/llama-3.1-405b` — Open source
- `mistralai/mistral-large` — Efficient
- `deepseek/deepseek-chat` — Cost-effective

### Nous Portal Setup

**Best for:** Nous-hosted models, OAuth integration

```yaml
model:
  default: "nous-hermes-3-405b"
  provider: "nous"
  # base_url auto-detected from OAuth
```

**Setup:**
```bash
# Interactive OAuth login
hermes login

# Or set manually in ~/.hermes/.env
NOUS_INFERENCE_BASE_URL=https://api.nous.ai/v1
```

**Available Models:**
- `nous-hermes-3-405b` — Flagship
- `nous-hermes-2-mixtral-8x7b` — Efficient
- `nous-hermes-2-vision` — Vision-capable

### Anthropic Direct Setup

**Best for:** Direct Anthropic API, no intermediaries

```yaml
model:
  default: "claude-opus-4.6"
  provider: "anthropic"
  base_url: "https://api.anthropic.com"
  context_length: 200000
  max_tokens: 4096
```

**Environment:**
```bash
# ~/.hermes/.env
ANTHROPIC_API_KEY=sk-ant-...
```

**Available Models:**
- `claude-opus-4.6` — Latest flagship
- `claude-sonnet-4` — Balanced
- `claude-haiku-4.5` — Fast, cheap
- `claude-3-5-sonnet-20241022` — Previous version

### GitHub Copilot Setup

**Best for:** GitHub Models, free tier available

```yaml
model:
  default: "gpt-4-turbo"
  provider: "copilot"
  base_url: "https://models.inference.ai.azure.com"
```

**Environment:**
```bash
# ~/.hermes/.env
COPILOT_GITHUB_TOKEN=ghp_...
# Or fallback:
GH_TOKEN=ghp_...
GITHUB_TOKEN=ghp_...
```

**Available Models:**
- `gpt-4-turbo`
- `gpt-4`
- `gpt-3.5-turbo`
- `claude-3.5-sonnet`
- `claude-3-haiku`
- `llama-2-7b`
- `llama-2-70b`
- `mistral-large`
- `phi-3.5-mini`

### Google Gemini Setup

**Best for:** Vision models, multimodal

```yaml
model:
  default: "gemini-3-flash-preview"
  provider: "gemini"
  base_url: "https://generativelanguage.googleapis.com/v1beta/openai/"
```

**Environment:**
```bash
# ~/.hermes/.env
GOOGLE_API_KEY=AIzaSy...
# Aliases:
GEMINI_API_KEY=AIzaSy...
```

**Available Models:**
- `gemini-3-flash-preview` — Latest, fast
- `gemini-2.0-flash` — Multimodal
- `gemini-pro` — Standard
- `gemini-pro-vision` — Vision

### z.ai / ZhipuAI GLM Setup

**Best for:** Chinese models, GLM series

```yaml
model:
  default: "glm-4-plus"
  provider: "zai"
  base_url: "https://open.bigmodel.cn/api/paas/v4"
```

**Environment:**
```bash
# ~/.hermes/.env
GLM_API_KEY=...
# Aliases:
ZAI_API_KEY=...
Z_AI_API_KEY=...
```

**Available Models:**
- `glm-4-plus` — Latest
- `glm-4` — Standard
- `glm-3-turbo` — Fast
- `glm-4-vision` — Vision

### Kimi / Moonshot AI Setup

**Best for:** Long context, Chinese models

```yaml
model:
  default: "moonshot-v1-128k"
  provider: "kimi-coding"
  base_url: "https://api.moonshot.cn/v1"
```

**Environment:**
```bash
# ~/.hermes/.env
KIMI_API_KEY=sk-...
KIMI_BASE_URL=https://api.moonshot.cn/v1  # Optional

# China endpoint:
KIMI_CN_API_KEY=sk-...
```

**Available Models:**
- `moonshot-v1-128k` — 128K context
- `moonshot-v1-32k` — 32K context
- `moonshot-v1-8k` — 8K context

### MiniMax Setup

**Best for:** Chinese market, multimodal

```yaml
model:
  default: "abab6.5-chat"
  provider: "minimax"
  base_url: "https://api.minimax.chat/v1"
```

**Environment:**
```bash
# ~/.hermes/.env
MINIMAX_API_KEY=...
MINIMAX_BASE_URL=https://api.minimax.chat/v1  # Optional

# China endpoint:
MINIMAX_CN_API_KEY=...
MINIMAX_CN_BASE_URL=...
```

**Available Models:**
- `abab6.5-chat` — Latest
- `abab6-chat` — Standard
- `abab5.5-chat` — Previous

### Arcee AI Trinity Setup

**Best for:** Enterprise, specialized models

```yaml
model:
  default: "arcee-trinity"
  provider: "arcee"
  base_url: "https://api.arcee.ai/v1"
```

**Environment:**
```bash
# ~/.hermes/.env
ARCEEAI_API_KEY=...
ARCEE_BASE_URL=https://api.arcee.ai/v1  # Optional
```

### DeepSeek Setup

**Best for:** Cost-effective, reasoning models

```yaml
model:
  default: "deepseek-chat"
  provider: "deepseek"
  base_url: "https://api.deepseek.com"
```

**Environment:**
```bash
# ~/.hermes/.env
DEEPSEEK_API_KEY=sk-...
DEEPSEEK_BASE_URL=https://api.deepseek.com  # Optional
```

**Available Models:**
- `deepseek-chat` — Standard
- `deepseek-coder` — Code-specialized
- `deepseek-reasoner` — Reasoning

### Custom OpenAI-Compatible Setup

**Best for:** Self-hosted, private endpoints

```yaml
model:
  default: "gpt-3.5-turbo"
  provider: "custom"
  base_url: "http://localhost:8000/v1"
  api_key: "${OPENAI_API_KEY}"
  context_length: 4096
```

**Environment:**
```bash
# ~/.hermes/.env
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=http://localhost:8000/v1  # Optional override
```

**Common Aliases:**
- `lmstudio` → `http://localhost:1234/v1`
- `ollama` → `http://localhost:11434/v1`
- `vllm` → `http://localhost:8000/v1`
- `llamacpp` → `http://localhost:8080/v1`

---

## Model Selection & Configuration

### Model Naming Conventions

**OpenRouter format:**
```
provider/model-name
anthropic/claude-opus-4.6
google/gemini-3-flash-preview
meta-llama/llama-3.1-405b
```

**Direct provider format:**
```
model-name
claude-opus-4.6          # Anthropic
gpt-4-turbo              # OpenAI
gemini-3-flash-preview   # Google
```

### Context Length Configuration

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  context_length: 200000              # Explicit override
  # If omitted, auto-detected from provider
```

**Auto-Detection Priority:**
1. Explicit `context_length` in config
2. Provider API metadata
3. Model registry lookup
4. Safe default (4096)

### Max Tokens Configuration

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  max_tokens: 8192                    # Output cap
  # If omitted, uses model's default
```

**Behavior:**
- If set: Hermes caps output at this value
- If omitted: Uses model's default max
- Useful for cost control or API limits

### Model Switching at Runtime

```bash
# Override model for single command
hermes chat --model anthropic/claude-sonnet-4

# Override provider
hermes chat --provider openrouter

# Override base URL
hermes chat --base-url http://localhost:8000/v1

# Combine
hermes chat --model gpt-4-turbo --provider custom --base-url http://localhost:8000/v1
```

### Model Capabilities Detection

Hermes automatically detects:
- Vision support (image analysis)
- Reasoning support (extended thinking)
- Tool use support (function calling)
- Context length
- Max output tokens

---

## Fallback Model Configuration

### Basic Fallback Setup

```yaml
fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"
  # Optional overrides:
  base_url: "https://openrouter.ai/v1"
  api_key: "${OPENROUTER_API_KEY}"
```

### Fallback Triggers

Automatic failover when:
1. Primary model API returns error
2. Rate limit exceeded
3. Model unavailable
4. Timeout exceeded
5. Authentication fails

### Multi-Level Fallback Chain

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

fallback_model_3:
  provider: "anthropic"
  model: "claude-haiku-4.5"
```

### Fallback Behavior

```yaml
fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"
  retry_count: 3                      # Retry N times before fallback
  retry_delay: 2                      # Seconds between retries
  exponential_backoff: true           # Double delay each retry
```

---

## Auxiliary Model Configuration

### Universal Auxiliary Configuration Pattern

Every auxiliary task uses the same three knobs:

```yaml
auxiliary:
  TASK_NAME:
    provider: "auto"                  # Provider for auth/routing
    model: ""                         # Model name (empty = provider default)
    base_url: ""                      # Custom OpenAI-compatible endpoint
    api_key: ""                       # API key for base_url
    timeout: 120                      # LLM call timeout (seconds)
```

### Vision Analysis

**Used for:** Image analysis, browser screenshots, vision_analyze tool

```yaml
auxiliary:
  vision:
    provider: "auto"                  # auto | openrouter | gemini | copilot | etc.
    model: "google/gemini-3-flash-preview"  # Empty = provider default
    base_url: ""                      # Custom endpoint (optional)
    api_key: ""                       # Custom API key (optional)
    timeout: 120                      # LLM call timeout (seconds)
    download_timeout: 30              # Image HTTP download timeout
```

**Provider Options:**
- `"auto"` — Best available (OpenRouter → Gemini → Copilot)
- `"openrouter"` — Route to any vision model
- `"gemini"` — Google Gemini (best for vision)
- `"copilot"` — GitHub Models
- `"anthropic"` — Claude vision
- `"main"` — Use primary model provider

**Recommended Models:**
- `google/gemini-3-flash-preview` — Best quality
- `anthropic/claude-opus-4.6` — Highest accuracy
- `openai/gpt-4-vision` — Reliable

### Web Extraction & Summarization

**Used for:** Web page summarization, browser text extraction

```yaml
auxiliary:
  web_extract:
    provider: "auto"
    model: "google/gemini-3-flash-preview"
    timeout: 360                      # 6 minutes for summarization
    download_timeout: 30              # Page download timeout
```

**Provider Options:**
- `"auto"` — Best available
- `"openrouter"` — Any model
- `"main"` — Use primary provider

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast, accurate
- `anthropic/claude-sonnet-4` — Detailed summaries

### Dangerous Command Approval

**Used for:** Classify dangerous commands (rm -rf, DROP TABLE, etc.)

```yaml
auxiliary:
  approval:
    provider: "auto"
    model: ""                         # Fast model recommended
    timeout: 30                       # Quick classification
```

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast
- `anthropic/claude-haiku-4.5` — Cheap, fast

### Context Compression

**Used for:** Summarize old messages when approaching context limit

```yaml
auxiliary:
  compression:
    provider: "auto"
    model: "google/gemini-3-flash-preview"
    timeout: 120
```

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast compression
- `anthropic/claude-sonnet-4` — Accurate compression

### Session Search

**Used for:** Summarize past session matches for context

```yaml
auxiliary:
  session_search:
    provider: "auto"
    model: ""
    timeout: 30
```

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast
- `anthropic/claude-haiku-4.5` — Cheap

### Skills Hub

**Used for:** Skill matching and search

```yaml
auxiliary:
  skills_hub:
    provider: "auto"
    model: ""
    timeout: 30
```

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast matching
- `anthropic/claude-haiku-4.5` — Cheap

### MCP Tool Dispatch

**Used for:** Route to appropriate MCP server

```yaml
auxiliary:
  mcp:
    provider: "auto"
    model: ""
    timeout: 30
```

### Memory Flush

**Used for:** Summarize conversation for persistent memory

```yaml
auxiliary:
  flush_memories:
    provider: "auto"
    model: ""
    timeout: 30
```

**Recommended Models:**
- `google/gemini-3-flash-preview` — Fast
- `anthropic/claude-sonnet-4` — Accurate

### Complete Auxiliary Configuration Example

```yaml
auxiliary:
  vision:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120
    download_timeout: 30

  web_extract:
    provider: "openrouter"
    model: "anthropic/claude-sonnet-4"
    timeout: 360

  approval:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30

  compression:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120

  session_search:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30

  skills_hub:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30

  mcp:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30

  flush_memories:
    provider: "openrouter"
    model: "anthropic/claude-sonnet-4"
    timeout: 30
```

---

## Credential Pool Strategies

### Overview

Hermes supports multiple API keys per provider for load balancing and failover.

### Fill-First Strategy

**Behavior:** Use first key until exhausted, then move to next

```yaml
credential_pool:
  strategy: "fill_first"              # Default
  keys:
    - "sk-or-v1-key1..."
    - "sk-or-v1-key2..."
    - "sk-or-v1-key3..."
```

**Use Case:** Different rate limits per key, sequential consumption

### Round-Robin Strategy

**Behavior:** Distribute requests evenly across all keys

```yaml
credential_pool:
  strategy: "round_robin"
  keys:
    - "sk-or-v1-key1..."
    - "sk-or-v1-key2..."
    - "sk-or-v1-key3..."
```

**Use Case:** Load balancing, equal rate limits

### Least-Used Strategy

**Behavior:** Use key with lowest usage count

```yaml
credential_pool:
  strategy: "least_used"
  keys:
    - "sk-or-v1-key1..."
    - "sk-or-v1-key2..."
    - "sk-or-v1-key3..."
```

**Use Case:** Maximize throughput, avoid single key exhaustion

### Random Strategy

**Behavior:** Randomly select key for each request

```yaml
credential_pool:
  strategy: "random"
  keys:
    - "sk-or-v1-key1..."
    - "sk-or-v1-key2..."
    - "sk-or-v1-key3..."
```

**Use Case:** Chaos testing, uniform distribution

### Credential Pool with Fallback

```yaml
credential_pool:
  strategy: "round_robin"
  keys:
    - "sk-or-v1-primary..."
    - "sk-or-v1-backup1..."
    - "sk-or-v1-backup2..."
  retry_on_failure: true              # Retry with next key on error
  retry_count: 3
```

### Environment Variable Credential Pool

```bash
# ~/.hermes/.env
OPENROUTER_API_KEY=sk-or-v1-key1:sk-or-v1-key2:sk-or-v1-key3
# Colon-separated keys
```

---

## Reasoning Effort Settings

### Reasoning Effort Levels

| Level | Description | Tokens | Latency | Use Case |
|-------|-------------|--------|---------|----------|
| `none` | Disable reasoning | Minimal | Fastest | Simple queries |
| `minimal` | Minimal thinking | Low | Fast | Quick answers |
| `low` | Low reasoning effort | Medium | Medium | Standard queries |
| `medium` | Balanced (default) | High | Medium | Most use cases |
| `high` | High reasoning effort | Very High | Slow | Complex problems |
| `xhigh` | Maximum reasoning | Extreme | Very Slow | Research, analysis |

### Configuration

```yaml
agent:
  reasoning_effort: "medium"          # Default
```

### Per-Request Override

```bash
hermes chat --reasoning-effort high "Complex problem requiring deep analysis"
```

### Provider Support

| Provider | none | minimal | low | medium | high | xhigh |
|----------|------|---------|-----|--------|------|-------|
| Anthropic | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| OpenAI | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |
| Google | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |
| OpenRouter | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| DeepSeek | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Others | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |

### Reasoning Effort with Fallback

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  reasoning_effort: "high"

fallback_model:
  provider: "openrouter"
  model: "google/gemini-3-flash-preview"
  reasoning_effort: "medium"          # Fallback uses lower effort
```

### Budget Pressure Warnings

Automatic warnings as agent approaches iteration limit:

```
[BUDGET: 63/90. 27 iterations left. Start consolidating.]  # 70%
[BUDGET WARNING: 81/90. Only 9 left. Respond NOW.]         # 90%
```

---

## Tool-Use Enforcement

### Tool-Use Configuration

```yaml
agent:
  tool_use:
    enabled: true                     # Enable/disable tool calling
    enforce: false                    # Require tool use for every turn
    max_parallel_calls: 5             # Max concurrent tool calls
    timeout: 300                      # Tool execution timeout (seconds)
```

### Enforce Tool Use

**Behavior:** Model must use at least one tool per turn

```yaml
agent:
  tool_use:
    enforce: true
```

**Use Case:** Ensure agent takes action, not just reasoning

### Disable Tool Use

**Behavior:** Model cannot call tools

```yaml
agent:
  tool_use:
    enabled: false
```

**Use Case:** Pure reasoning, no side effects

### Tool Allowlist

```yaml
agent:
  tool_use:
    enabled: true
    allowed_tools:
      - "terminal"
      - "file"
      - "web_search"
      # Deny all others
```

### Tool Denylist

```yaml
agent:
  tool_use:
    enabled: true
    denied_tools:
      - "terminal"
      - "process"
      # Allow all others
```

### Parallel Tool Calls

```yaml
agent:
  tool_use:
    max_parallel_calls: 10            # Allow up to 10 concurrent calls
```

---

## Streaming Timeouts

### Streaming Configuration

```yaml
streaming:
  enabled: false                      # Enable progressive response
  transport: "edit"                   # edit = progressive editMessageText
  edit_interval: 0.3                  # Seconds between edits
  buffer_threshold: 40                # Chars before forcing edit
  cursor: " ▉"                        # Cursor shown during streaming
```

### Streaming Timeouts

```yaml
streaming:
  enabled: true
  read_timeout: 30                    # Socket read timeout (seconds)
  stale_timeout: 60                   # Stale stream detection (seconds)
  chunk_timeout: 5                    # Max time between chunks
```

### Environment Variable Overrides

```bash
# ~/.hermes/.env
HERMES_STREAM_READ_TIMEOUT=30         # Socket read timeout
HERMES_STREAM_STALE_TIMEOUT=60        # Stale detection
```

### Streaming with Messaging Platforms

```yaml
streaming:
  enabled: true
  transport: "edit"                   # Progressive message edits
  edit_interval: 0.5                  # Update every 0.5 seconds
  buffer_threshold: 50                # Edit when 50+ chars buffered
```

---

## API Timeout Settings

### Global API Timeout

```yaml
agent:
  api_timeout: 120                    # LLM API call timeout (seconds)
```

### Per-Provider Timeout

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  timeout: 120                        # Anthropic timeout

fallback_model:
  provider: "openrouter"
  model: "google/gemini-3-flash-preview"
  timeout: 60                         # OpenRouter timeout
```

### Auxiliary Task Timeouts

```yaml
auxiliary:
  vision:
    timeout: 120                      # Vision API timeout
    download_timeout: 30              # Image download timeout

  web_extract:
    timeout: 360                      # Web extraction timeout (6 min)

  compression:
    timeout: 120                      # Compression timeout

  approval:
    timeout: 30                       # Quick classification

  session_search:
    timeout: 30

  skills_hub:
    timeout: 30

  mcp:
    timeout: 30

  flush_memories:
    timeout: 30
```

### Terminal Command Timeout

```yaml
terminal:
  timeout: 180                        # Per-command timeout (seconds)
  lifetime_seconds: 300               # Session lifetime
```

### Browser Timeout

```yaml
browser:
  inactivity_timeout: 120             # Browser session timeout (seconds)
```

### Code Execution Timeout

```yaml
code_execution:
  timeout: 300                        # Max seconds per script (5 min)
```

### Environment Variable Overrides

```bash
# ~/.hermes/.env
HERMES_API_TIMEOUT=120                # LLM API timeout
HERMES_STREAM_READ_TIMEOUT=30         # Streaming read timeout
HERMES_STREAM_STALE_TIMEOUT=60        # Stale stream timeout
TERMINAL_TIMEOUT=180                  # Terminal command timeout
```

---

## Custom Provider Configuration

### Custom OpenAI-Compatible Endpoint

```yaml
model:
  default: "gpt-3.5-turbo"
  provider: "custom"
  base_url: "http://localhost:8000/v1"
  api_key: "${OPENAI_API_KEY}"
  context_length: 4096
  max_tokens: 2048
```

### LM Studio (Local)

```yaml
model:
  default: "mistral-7b"
  provider: "lmstudio"
  # Auto-configured to http://localhost:1234/v1
```

### Ollama (Local)

```yaml
model:
  default: "llama2"
  provider: "ollama"
  # Auto-configured to http://localhost:11434/v1
```

### vLLM (Local)

```yaml
model:
  default: "meta-llama/Llama-2-7b-hf"
  provider: "vllm"
  # Auto-configured to http://localhost:8000/v1
```

### llama.cpp (Local)

```yaml
model:
  default: "llama-2-7b"
  provider: "llamacpp"
  # Auto-configured to http://localhost:8080/v1
```

### Custom Endpoint with Auth

```yaml
model:
  default: "custom-model"
  provider: "custom"
  base_url: "https://api.custom.com/v1"
  api_key: "${CUSTOM_API_KEY}"
  headers:
    Authorization: "Bearer ${CUSTOM_TOKEN}"
    X-Custom-Header: "value"
```

### Custom Endpoint with SSL Verification

```yaml
model:
  default: "custom-model"
  provider: "custom"
  base_url: "https://api.custom.com/v1"
  api_key: "${CUSTOM_API_KEY}"
  ssl_verify: true                    # Verify SSL certificates
  ssl_ca_bundle: "/path/to/ca-bundle.crt"  # Custom CA bundle
```

### Custom Endpoint with Proxy

```yaml
model:
  default: "custom-model"
  provider: "custom"
  base_url: "https://api.custom.com/v1"
  api_key: "${CUSTOM_API_KEY}"
  proxy: "http://proxy.example.com:8080"
  proxy_auth: "${PROXY_USER}:${PROXY_PASS}"
```

---

## Provider Routing & Smart Selection

### Provider Routing (OpenRouter Only)

```yaml
provider_routing:
  sort: "price"                       # price | throughput | latency
  only: ["anthropic", "google"]       # Whitelist providers
  ignore: ["deepinfra"]               # Blacklist providers
  order: ["anthropic", "google"]      # Explicit priority order
  require_parameters: true            # Only use providers supporting all params
  data_collection: "deny"             # "allow" | "deny" (exclude data-storing providers)
```

### Smart Model Routing

Use cheaper model for simple turns:

```yaml
smart_model_routing:
  enabled: true
  max_simple_chars: 160               # Threshold for "simple" query
  max_simple_words: 28
  cheap_model:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"
```

### Provider Routing Example

```yaml
provider_routing:
  sort: "price"                       # Prioritize cost
  only: ["anthropic", "google", "openai"]  # Only these providers
  ignore: ["deepinfra", "together"]   # Exclude these
  require_parameters: true            # Must support all params
  data_collection: "deny"             # No data collection
```

---

## Configuration Examples

### Example 1: Production Setup (OpenRouter)

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"
  base_url: "https://openrouter.ai/v1"
  context_length: 200000
  max_tokens: 8192

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"

auxiliary:
  vision:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120

  web_extract:
    provider: "openrouter"
    model: "anthropic/claude-sonnet-4"
    timeout: 360

  compression:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120

agent:
  reasoning_effort: "high"
  api_timeout: 120
  max_turns: 90

provider_routing:
  sort: "price"
  only: ["anthropic", "google"]
  data_collection: "deny"
```

**Environment:**
```bash
OPENROUTER_API_KEY=sk-or-v1-...
```

### Example 2: Cost-Optimized Setup

```yaml
model:
  default: "google/gemini-3-flash-preview"
  provider: "openrouter"

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-haiku-4.5"

smart_model_routing:
  enabled: true
  max_simple_chars: 200
  cheap_model:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"

auxiliary:
  vision:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"

  web_extract:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"

  compression:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"

agent:
  reasoning_effort: "low"
  api_timeout: 60
```

### Example 3: Local Development Setup

```yaml
model:
  default: "mistral-7b"
  provider: "ollama"
  context_length: 8192
  max_tokens: 2048

fallback_model:
  provider: "lmstudio"
  model: "neural-chat-7b"

auxiliary:
  vision:
    provider: "ollama"
    model: "llava"

  web_extract:
    provider: "ollama"
    model: "mistral-7b"

agent:
  reasoning_effort: "low"
  api_timeout: 300
```

### Example 4: Multi-Provider Failover

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "anthropic"

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-opus-4.6"

fallback_model_2:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"

fallback_model_3:
  provider: "openrouter"
  model: "google/gemini-3-flash-preview"

credential_pool:
  strategy: "round_robin"
  keys:
    - "${ANTHROPIC_API_KEY_1}"
    - "${ANTHROPIC_API_KEY_2}"
    - "${ANTHROPIC_API_KEY_3}"
```

### Example 5: Enterprise Setup (Multiple Providers)

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "openrouter"
  context_length: 200000

fallback_model:
  provider: "openrouter"
  model: "anthropic/claude-sonnet-4"

auxiliary:
  vision:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120

  web_extract:
    provider: "openrouter"
    model: "anthropic/claude-sonnet-4"
    timeout: 360

  compression:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 120

  approval:
    provider: "openrouter"
    model: "google/gemini-3-flash-preview"
    timeout: 30

agent:
  reasoning_effort: "high"
  api_timeout: 120
  max_turns: 90
  tool_use:
    enabled: true
    enforce: false
    max_parallel_calls: 5

provider_routing:
  sort: "price"
  only: ["anthropic", "google", "openai"]
  ignore: ["deepinfra"]
  require_parameters: true
  data_collection: "deny"

credential_pool:
  strategy: "round_robin"
  keys:
    - "${OPENROUTER_API_KEY_1}"
    - "${OPENROUTER_API_KEY_2}"
    - "${OPENROUTER_API_KEY_3}"
  retry_on_failure: true
  retry_count: 3
```

---

## Quick Reference

### Setting Model via CLI

```bash
# Override model
hermes chat --model anthropic/claude-sonnet-4

# Override provider
hermes chat --provider openrouter

# Override base URL
hermes chat --base-url http://localhost:8000/v1

# Override reasoning effort
hermes chat --reasoning-effort high

# Combine
hermes chat --model gpt-4-turbo --provider custom --base-url http://localhost:8000/v1
```

### Setting Model via Config

```bash
# Set default model
hermes config set model.default anthropic/claude-opus-4.6

# Set provider
hermes config set model.provider openrouter

# Set context length
hermes config set model.context_length 200000

# Set reasoning effort
hermes config set agent.reasoning_effort high
```

### Checking Configuration

```bash
# View current configuration
hermes config

# View specific setting
hermes config show model.default

# Check for missing options
hermes config check

# Interactively migrate config
hermes config migrate
```

---

## Troubleshooting

### Provider Not Found

**Error:** `Provider 'xyz' not found`

**Solution:**
1. Check provider name spelling
2. Verify API key is set in `.env`
3. Run `hermes config check`

### Authentication Failed

**Error:** `Authentication failed for provider`

**Solution:**
1. Verify API key in `~/.hermes/.env`
2. Check key hasn't expired
3. Verify key has correct permissions
4. Try `hermes login` for OAuth providers

### Model Not Available

**Error:** `Model 'xyz' not found`

**Solution:**
1. Check model name format (provider/model for OpenRouter)
2. Verify model is available in provider
3. Check provider routing settings
4. Try fallback model

### Timeout Issues

**Error:** `Request timeout`

**Solution:**
1. Increase `api_timeout` in config
2. Check network connectivity
3. Try different provider
4. Check provider status page

### Context Length Exceeded

**Error:** `Context length exceeded`

**Solution:**
1. Enable compression: `compression.enabled: true`
2. Reduce `memory_char_limit`
3. Use model with larger context
4. Clear session history

---

## References

- [OpenRouter API Docs](https://openrouter.ai/docs)
- [Anthropic API Docs](https://docs.anthropic.com)
- [Google Gemini API Docs](https://ai.google.dev)
- [GitHub Models](https://github.com/marketplace/models)
- [Nous Portal](https://nous.ai)
- [DeepSeek API](https://platform.deepseek.com)

---

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)  
**Maintained by:** Hermes Agent Team
