# Hermes Agent - Complete Configuration Reference

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)  
**Config Schema Version:** 10

---

## Table of Contents

1. [Configuration Files & Structure](#configuration-files--structure)
2. [Configuration Precedence](#configuration-precedence)
3. [Model Configuration](#model-configuration)
4. [Terminal Backend Configuration](#terminal-backend-configuration)
5. [Context Compression](#context-compression)
6. [Auxiliary Models](#auxiliary-models)
7. [Memory Configuration](#memory-configuration)
8. [Agent Behavior](#agent-behavior)
9. [Display & Interface](#display--interface)
10. [Toolsets & Tools](#toolsets--tools)
11. [MCP Servers](#mcp-servers)
12. [Messaging Platforms](#messaging-platforms)
13. [Session Management](#session-management)
14. [Security & Approval](#security--approval)
15. [Cron & Scheduling](#cron--scheduling)
16. [Skills Configuration](#skills-configuration)
17. [Profiles (Multi-Agent)](#profiles-multi-agent)
18. [Environment Variables Reference](#environment-variables-reference)

---

## Configuration Files & Structure

### Directory Layout

```
~/.hermes/                          # Default Hermes home (override with HERMES_HOME)
├── config.yaml                     # Main configuration file
├── .env                            # API keys and secrets (mode 0600)
├── auth.json                       # OAuth credentials (Nous Portal, etc.)
├── SOUL.md                         # Primary agent identity/personality
├── memories/
│   ├── MEMORY.md                   # Agent's personal notes
│   └── USER.md                     # User profile
├── skills/                         # Active skills (bundled + hub + agent-created)
├── cron/                           # Scheduled job data
├── sessions/                       # JSON session logs
├── state.db                        # SQLite session database
├── logs/
│   ├── errors.log                  # Error logs (secrets auto-redacted)
│   └── gateway.log                 # Gateway logs
├── sandboxes/                      # Container overlays (Docker, Singularity, Modal)
├── modal_snapshots.json            # Modal filesystem snapshots
└── profiles/                       # Multi-agent profiles
    ├── coder/
    │   ├── config.yaml
    │   ├── .env
    │   └── SOUL.md
    └── assistant/
        ├── config.yaml
        ├── .env
        └── SOUL.md
```

### File Purposes

| File | Purpose | Permissions |
|------|---------|-------------|
| `config.yaml` | All behavioral settings (model, terminal, display, memory, tools) | 0644 |
| `.env` | API keys, tokens, passwords | 0600 (owner-only) |
| `auth.json` | OAuth provider credentials | 0600 |
| `SOUL.md` | Agent identity, personality, system instructions | 0644 |
| `MEMORY.md` | Agent's persistent notes (auto-managed) | 0644 |
| `USER.md` | User profile (auto-managed) | 0644 |

---

## Configuration Precedence

Settings are resolved in this order (highest priority first):

1. **CLI arguments** — e.g., `hermes chat --model anthropic/claude-sonnet-4`
2. **`~/.hermes/config.yaml`** — primary config for non-secret settings
3. **`~/.hermes/.env`** — fallback for env vars; required for secrets
4. **Built-in defaults** — hardcoded safe defaults

### Rule of Thumb

- **Secrets** (API keys, bot tokens, passwords) → `.env`
- **Everything else** (model, terminal, compression, memory) → `config.yaml`
- When both are set, `config.yaml` wins for non-secret settings

### Environment Variable Substitution

Reference env vars in `config.yaml` using `${VAR_NAME}` syntax:

```yaml
auxiliary:
  vision:
    api_key: ${GOOGLE_API_KEY}
    base_url: ${CUSTOM_VISION_URL}
delegation:
  api_key: ${DELEGATION_KEY}
```

Multiple references work: `url: "${HOST}:${PORT}"`. If undefined, placeholder is kept verbatim.

---

## Model Configuration

### Basic Setup

```yaml
model:
  default: "anthropic/claude-opus-4.6"
  provider: "auto"                    # auto | openrouter | nous | anthropic | copilot | etc.
  base_url: "https://openrouter.ai/v1"
  # api_key: "sk-..."                 # Optional; prefer .env
  context_length: 131072              # Auto-detected; set manually if wrong
  max_tokens: 8192                    # Output cap (optional; use model default if unset)
```

### Supported Providers

| Provider | Env Var | Notes |
|----------|---------|-------|
| `auto` | — | Auto-detect from credentials |
| `openrouter` | `OPENROUTER_API_KEY` | 100+ models, recommended |
| `nous` | — | Nous Portal OAuth (`hermes login`) |
| `nous-api` | `NOUS_API_KEY` | Nous Portal API key |
| `anthropic` | `ANTHROPIC_API_KEY` | Direct Anthropic API |
| `openai-codex` | — | OpenAI Codex (`hermes auth`) |
| `copilot` | `COPILOT_GITHUB_TOKEN` | GitHub Copilot / GitHub Models |
| `copilot-acp` | — | Copilot ACP mode |
| `gemini` | `GOOGLE_API_KEY` | Google AI Studio direct |
| `zai` | `GLM_API_KEY` | z.ai / ZhipuAI GLM |
| `kimi-coding` | `KIMI_API_KEY` | Kimi / Moonshot AI |
| `kimi-coding-cn` | `KIMI_CN_API_KEY` | Kimi China endpoint |
| `minimax` | `MINIMAX_API_KEY` | MiniMax global |
| `minimax-cn` | `MINIMAX_CN_API_KEY` | MiniMax China |
| `kilocode` | `KILOCODE_API_KEY` | Kilo Code gateway |
| `xiaomi` | `XIAOMI_API_KEY` | Xiaomi MiMo |
| `arcee` | `ARCEEAI_API_KEY` | Arcee AI Trinity |
| `huggingface` | `HF_TOKEN` | Hugging Face Inference |
| `ollama-cloud` | `OLLAMA_API_KEY` | Ollama Cloud |
| `ai-gateway` | `AI_GATEWAY_API_KEY` | Vercel AI Gateway |
| `custom` | `OPENAI_API_KEY` | Any OpenAI-compatible endpoint |
| `lmstudio` | — | Alias for `custom` (LM Studio) |
| `ollama` | — | Alias for `custom` (Ollama local) |
| `vllm` | — | Alias for `custom` (vLLM) |
| `llamacpp` | — | Alias for `custom` (llama.cpp) |

### Fallback Model

Automatic failover when primary model fails:

```yaml
fallback_model:
  provider: openrouter
  model: anthropic/claude-sonnet-4
```

### Provider Routing (OpenRouter only)

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
  max_simple_chars: 160
  max_simple_words: 28
  cheap_model:
    provider: openrouter
    model: google/gemini-2.5-flash
```

---

## Terminal Backend Configuration

### Overview

| Backend | Where | Isolation | Best For |
|---------|-------|-----------|----------|
| `local` | Your machine | None | Development, personal use |
| `docker` | Docker container | Full (namespaces) | Safe sandboxing, CI/CD |
| `ssh` | Remote server | Network boundary | Remote dev, powerful hardware |
| `modal` | Modal cloud VM | Full (cloud) | GPU access, ephemeral compute |
| `daytona` | Daytona workspace | Full (cloud) | Managed cloud dev environments |
| `singularity` | Singularity container | Namespaces | HPC clusters, shared machines |

### Local Backend (Default)

```yaml
terminal:
  backend: "local"
  cwd: "."                            # Current directory
  timeout: 180                        # Per-command timeout (seconds)
  lifetime_seconds: 300               # Session lifetime
```

### Docker Backend

```yaml
terminal:
  backend: "docker"
  cwd: "/workspace"
  timeout: 180
  lifetime_seconds: 300
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_mount_cwd_to_workspace: false  # SECURITY: opt-in to mount launch dir
  docker_forward_env:                 # Env vars to forward into container
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
  docker_volumes:                     # Host directory mounts
    - "/home/user/projects:/workspace/projects"
    - "/home/user/data:/data:ro"      # :ro for read-only
  container_cpu: 1                    # CPU cores
  container_memory: 5120              # MB (5GB)
  container_disk: 51200               # MB (50GB)
  container_persistent: true          # Persist /workspace across sessions
```

### SSH Backend

```yaml
terminal:
  backend: "ssh"
  cwd: "/home/myuser/project"
  timeout: 180
  lifetime_seconds: 300
  persistent_shell: true              # Keep long-lived bash session
```

**Required environment variables:**
- `TERMINAL_SSH_HOST` — Remote server hostname
- `TERMINAL_SSH_USER` — SSH username

**Optional:**
- `TERMINAL_SSH_PORT` — SSH port (default: 22)
- `TERMINAL_SSH_KEY` — Path to private key
- `TERMINAL_SSH_PERSISTENT` — Override persistent shell

### Modal Backend

```yaml
terminal:
  backend: "modal"
  cwd: "/workspace"
  timeout: 180
  lifetime_seconds: 300
  modal_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true          # Snapshot/restore filesystem
```

**Required:** `MODAL_TOKEN_ID` + `MODAL_TOKEN_SECRET` or `~/.modal.toml`

### Daytona Backend

```yaml
terminal:
  backend: "daytona"
  cwd: "~"
  timeout: 180
  lifetime_seconds: 300
  daytona_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  container_cpu: 1
  container_memory: 5120              # Converted to GiB
  container_disk: 10240               # Max 10 GiB
  container_persistent: true          # Stop/resume instead of delete
```

**Required:** `DAYTONA_API_KEY`

### Singularity/Apptainer Backend

```yaml
terminal:
  backend: "singularity"
  cwd: "/workspace"
  timeout: 180
  lifetime_seconds: 300
  singularity_image: "docker://nikolaik/python-nodejs:python3.11-nodejs20"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true          # Writable overlay persists
```

### Persistent Shell

Keep state across commands (enabled by default for SSH):

```yaml
terminal:
  persistent_shell: true              # Global default
  # Or per-backend:
  # TERMINAL_LOCAL_PERSISTENT=true
  # TERMINAL_SSH_PERSISTENT=true
```

### Sudo Support

```yaml
terminal:
  sudo_password: "hunter2"            # Pipe via sudo -S
  # sudo_password: ""                 # Empty = try empty password (passwordless sudo)
```

### Security Scanning (tirith)

```yaml
security:
  tirith_enabled: true
  tirith_path: "tirith"               # Path to tirith binary
  tirith_timeout: 5                   # Scan timeout (seconds)
  tirith_fail_open: true              # Allow if tirith unavailable
```

---

## Context Compression

Automatically shrinks long conversations to stay within context limit.

```yaml
compression:
  enabled: true                       # Toggle on/off
  threshold: 0.50                     # Compress at 50% of context limit
  target_ratio: 0.20                  # Preserve 20% of threshold as recent tail
  protect_last_n: 20                  # Min recent messages to keep uncompressed
```

### Compression Model Configuration

```yaml
auxiliary:
  compression:
    provider: "auto"                  # auto | openrouter | nous | main | etc.
    model: "google/gemini-3-flash-preview"
    base_url: null                    # Custom OpenAI-compatible endpoint
    timeout: 120                      # LLM call timeout (seconds)
```

### How It Works

1. Tracks actual token usage from API responses
2. When `prompt_tokens >= threshold% of context_length`, triggers compression
3. Protects first 3 turns (system prompt, initial request, first response)
4. Protects last N turns (default 20 messages ≈ 10 full turns)
5. Summarizes middle turns using auxiliary model
6. Inserts summary as user message, continues seamlessly

---

## Auxiliary Models

Lightweight models for side tasks: image analysis, web summarization, browser screenshots, compression.

### Universal Config Pattern

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

### Available Tasks

```yaml
auxiliary:
  # Image analysis (vision_analyze tool + browser screenshots)
  vision:
    provider: "auto"
    model: ""
    timeout: 120
    download_timeout: 30              # Image HTTP download timeout

  # Web page summarization + browser text extraction
  web_extract:
    provider: "auto"
    model: ""
    timeout: 360                      # 6 minutes for summarization

  # Dangerous command approval classifier
  approval:
    provider: "auto"
    model: ""
    timeout: 30

  # Context compression (separate from compression.* config)
  compression:
    timeout: 120

  # Session search — summarizes past session matches
  session_search:
    provider: "auto"
    model: ""
    timeout: 30

  # Skills hub — skill matching and search
  skills_hub:
    provider: "auto"
    model: ""
    timeout: 30

  # MCP tool dispatch
  mcp:
    provider: "auto"
    model: ""
    timeout: 30

  # Memory flush — summarizes conversation for persistent memory
  flush_memories:
    provider: "auto"
    model: ""
    timeout: 30
```

### Provider Options for Auxiliary Tasks

| Provider | Requirements | Notes |
|----------|--------------|-------|
| `"auto"` | — | Best available (OpenRouter → Nous → Codex) |
| `"openrouter"` | `OPENROUTER_API_KEY` | Route to any model |
| `"nous"` | `hermes auth` | Nous Portal |
| `"codex"` | `hermes model` → Codex | ChatGPT OAuth; supports vision |
| `"main"` | Custom endpoint | Use your active main provider |
| `"gemini"` | `GOOGLE_API_KEY` | Google AI Studio direct |
| `"zai"` | `GLM_API_KEY` | z.ai / ZhipuAI |
| `"kimi-coding"` | `KIMI_API_KEY` | Kimi / Moonshot |
| `"minimax"` | `MINIMAX_API_KEY` | MiniMax |
| `"arcee"` | `ARCEEAI_API_KEY` | Arcee AI |

---

## Memory Configuration

```yaml
memory:
  memory_enabled: true                # Agent's personal notes
  user_profile_enabled: true          # User profile
  memory_char_limit: 2200             # ~800 tokens
  user_char_limit: 1375               # ~500 tokens
  nudge_interval: 10                  # Remind agent every N user turns (0 = disabled)
  flush_min_turns: 6                  # Min turns to trigger flush on exit/reset
  provider: "holographic"             # Memory provider: honcho | openviking | mem0 | hindsight | holographic | retaindb | byterover | supermemory
```

### Memory Providers

| Provider | Config | Notes |
|----------|--------|-------|
| Built-in | — | MEMORY.md + USER.md (always active) |
| `honcho` | `~/.honcho/config.json` | Cross-session user modeling |
| `openviking` | `.env` vars | Semantic memory |
| `mem0` | `~/.hermes/mem0.json` | Persistent memory |
| `hindsight` | `~/.hermes/hindsight.json` | Memory with trust scoring |
| `holographic` | `~/.hermes/memory_store.db` | Local SQLite + FTS5 |
| `retaindb` | `~/.hermes/retaindb.json` | Retention-based memory |
| `byterover` | `~/.hermes/byterover.json` | Byte-level memory |
| `supermemory` | `.env` vars | Semantic long-term memory |

---

## Agent Behavior

```yaml
agent:
  max_turns: 90                       # Max tool-calling iterations per turn
  reasoning_effort: "medium"          # none | minimal | low | medium | high | xhigh
  verbose: false                      # Enable verbose logging
  gateway_timeout: 1800               # Inactivity timeout for gateway runs (seconds, 0 = unlimited)
  gateway_timeout_warning: 900        # Warning threshold before timeout
  restart_drain_timeout: 60           # Graceful drain timeout for gateway stop/restart
```

### Reasoning Effort Levels

| Level | Description |
|-------|-------------|
| `none` | Disable reasoning |
| `minimal` | Minimal thinking |
| `low` | Low reasoning effort |
| `medium` | Balanced (default) |
| `high` | High reasoning effort |
| `xhigh` | Maximum reasoning (most tokens/latency) |

### Iteration Budget Pressure

Automatic warnings as agent approaches iteration limit:

- **70%** (Caution): `[BUDGET: 63/90. 27 iterations left. Start consolidating.]`
- **90%** (Warning): `[BUDGET WARNING: 81/90. Only 9 left. Respond NOW.]`

---

## Display & Interface

```yaml
display:
  skin: "default"                     # default | ares | mono | slate | poseidon | sisyphus | charizard
  tool_progress: "all"                # off | new | all | verbose
  compact: false                      # Compact output
  show_reasoning: false               # Show model thinking
```

### Browser Configuration

```yaml
browser:
  inactivity_timeout: 120             # Browser session timeout (seconds)
```

### Response Pacing (Messaging)

```yaml
human_delay:
  mode: "off"                         # off | natural | custom
  min_ms: 800                         # Custom mode minimum delay
  max_ms: 2500                        # Custom mode maximum delay
```

### TUI (Terminal UI)

```yaml
# Launch TUI instead of classic CLI
export HERMES_TUI=1
# Or: hermes --tui
```

---

## Toolsets & Tools

### Platform Toolsets

```yaml
platform_toolsets:
  cli: [hermes-cli]
  telegram: [hermes-telegram]
  discord: [hermes-discord]
  whatsapp: [hermes-whatsapp]
  slack: [hermes-slack]
  signal: [hermes-signal]
  homeassistant: [hermes-homeassistant]
  qqbot: [hermes-qqbot]
```

### Individual Toolsets

| Toolset | Tools | Notes |
|---------|-------|-------|
| `web` | web_search, web_extract | Web search and scraping |
| `search` | web_search | Web search only |
| `terminal` | terminal, process | Command execution |
| `file` | read_file, write_file, patch, search | File operations |
| `browser` | browser_navigate, browser_snapshot, browser_click, etc. | Browser automation |
| `vision` | vision_analyze | Image analysis |
| `image_gen` | image_generate | Image generation (FLUX) |
| `skills` | skills_list, skill_view | Load skill documents |
| `skills_hub` | skill_hub | Search/install from registries |
| `moa` | mixture_of_agents | Mixture of Agents reasoning |
| `todo` | todo | Task planning |
| `memory` | — | Persistent memory |
| `session_search` | — | Search past conversations |
| `tts` | text_to_speech | Text-to-speech |
| `cronjob` | cronjob | Schedule tasks (CLI-only) |
| `rl` | rl_* | RL training tools |

### Presets

| Preset | Contents |
|--------|----------|
| `hermes-cli` | All tools except RL + send_message |
| `hermes-telegram` | terminal, file, web, vision, image_gen, tts, browser, skills, todo, cronjob, send_message |
| `hermes-discord` | Same as hermes-telegram |
| `hermes-whatsapp` | Same as hermes-telegram |
| `hermes-slack` | Same as hermes-telegram |

### Code Execution Sandbox

```yaml
code_execution:
  timeout: 300                        # Max seconds per script (5 min)
  max_tool_calls: 50                  # Max RPC tool calls per execution
```

### Subagent Delegation

```yaml
delegation:
  max_iterations: 50                  # Max tool-calling turns per child
  default_toolsets: ["terminal", "file", "web"]
  # model: "google/gemini-3-flash-preview"  # Override model (empty = inherit)
  # provider: "openrouter"             # Override provider (empty = inherit)
```

---

## MCP Servers

```yaml
mcp_servers:
  time:
    command: uvx
    args: ["mcp-server-time"]
  
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user"]
  
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "ghp_..."
  
  notion:
    url: https://mcp.notion.com/mcp
    headers:
      Authorization: "Bearer token"
  
  # Per-server settings
  analysis:
    command: npx
    args: ["-y", "analysis-server"]
    timeout: 120                      # Tool call timeout (seconds)
    connect_timeout: 60               # Connection timeout
    sampling:
      enabled: true
      model: "gemini-3-flash"
      max_tokens_cap: 4096
      timeout: 30
      max_rpm: 10
      allowed_models: []
      max_tool_rounds: 5
      log_level: "info"
```

---

## Messaging Platforms

### Telegram

```yaml
# Environment variables in ~/.hermes/.env
TELEGRAM_BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
TELEGRAM_ALLOWED_USERS=123456789,987654321
TELEGRAM_HOME_CHANNEL=123456789
TELEGRAM_HOME_CHANNEL_NAME="My Bot"
TELEGRAM_WEBHOOK_URL=https://example.com/webhook
TELEGRAM_WEBHOOK_PORT=8443
TELEGRAM_WEBHOOK_SECRET=secret123
TELEGRAM_REACTIONS=false
TELEGRAM_IGNORED_THREADS=111,222,333
TELEGRAM_PROXY=http://proxy.example.com:8080
```

### Discord

```yaml
DISCORD_BOT_TOKEN=MzA3NzA1NzA1NzA1NzA1NzA1.C...
DISCORD_ALLOWED_USERS=123456789,987654321
DISCORD_HOME_CHANNEL=123456789
DISCORD_HOME_CHANNEL_NAME="My Bot"
DISCORD_REQUIRE_MENTION=true
DISCORD_FREE_RESPONSE_CHANNELS=111,222
DISCORD_AUTO_THREAD=true
DISCORD_REACTIONS=true
DISCORD_IGNORED_CHANNELS=333,444
DISCORD_NO_THREAD_CHANNELS=555
DISCORD_REPLY_TO_MODE=first              # off | first | all
DISCORD_ALLOW_MENTION_EVERYONE=false
DISCORD_ALLOW_MENTION_ROLES=false
DISCORD_ALLOW_MENTION_USERS=true
DISCORD_ALLOW_MENTION_REPLIED_USER=true
```

### Slack

```yaml
SLACK_BOT_TOKEN=xoxb-...
SLACK_APP_TOKEN=xapp-...
SLACK_ALLOWED_USERS=U123,U456
SLACK_HOME_CHANNEL=C123456
SLACK_HOME_CHANNEL_NAME="My Bot"
```

### WhatsApp

```yaml
WHATSAPP_ENABLED=true
WHATSAPP_MODE=bot                       # bot | self-chat
WHATSAPP_ALLOWED_USERS=1234567890,9876543210
WHATSAPP_ALLOW_ALL_USERS=false
WHATSAPP_DEBUG=false
```

### Signal

```yaml
SIGNAL_HTTP_URL=http://127.0.0.1:8080
SIGNAL_ACCOUNT=+1234567890
SIGNAL_ALLOWED_USERS=+1234567890,+9876543210
SIGNAL_GROUP_ALLOWED_USERS=group1,group2
SIGNAL_HOME_CHANNEL_NAME="My Bot"
SIGNAL_IGNORE_STORIES=true
SIGNAL_ALLOW_ALL_USERS=false
```

### SMS (Twilio)

```yaml
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_PHONE_NUMBER=+1234567890
SMS_WEBHOOK_URL=https://example.com/sms
SMS_WEBHOOK_PORT=8080
SMS_WEBHOOK_HOST=0.0.0.0
SMS_INSECURE_NO_SIGNATURE=false         # Local dev only
SMS_ALLOWED_USERS=+1234567890,+9876543210
SMS_ALLOW_ALL_USERS=false
SMS_HOME_CHANNEL=+1234567890
SMS_HOME_CHANNEL_NAME="My Bot"
```

### Email

```yaml
EMAIL_ADDRESS=bot@example.com
EMAIL_PASSWORD=app_password_here
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_ALLOWED_USERS=user1@example.com,user2@example.com
EMAIL_HOME_ADDRESS=recipient@example.com
EMAIL_HOME_ADDRESS_NAME="My Bot"
EMAIL_POLL_INTERVAL=60
EMAIL_ALLOW_ALL_USERS=false
```

### DingTalk

```yaml
DINGTALK_CLIENT_ID=ding...
DINGTALK_CLIENT_SECRET=...
DINGTALK_ALLOWED_USERS=user1,user2
```

### Feishu/Lark

```yaml
FEISHU_APP_ID=cli_...
FEISHU_APP_SECRET=...
FEISHU_DOMAIN=feishu                    # feishu | lark
FEISHU_CONNECTION_MODE=websocket        # websocket | webhook
FEISHU_ENCRYPT_KEY=...
FEISHU_VERIFICATION_TOKEN=...
FEISHU_ALLOWED_USERS=user1,user2
FEISHU_HOME_CHANNEL=chat_id
```

### WeCom

```yaml
WECOM_BOT_ID=...
WECOM_SECRET=...
WECOM_WEBSOCKET_URL=wss://openws.work.weixin.qq.com
WECOM_ALLOWED_USERS=user1,user2
WECOM_HOME_CHANNEL=chat_id
WECOM_CALLBACK_CORP_ID=...
WECOM_CALLBACK_CORP_SECRET=...
WECOM_CALLBACK_AGENT_ID=...
WECOM_CALLBACK_TOKEN=...
WECOM_CALLBACK_ENCODING_AES_KEY=...
WECOM_CALLBACK_HOST=0.0.0.0
WECOM_CALLBACK_PORT=8645
WECOM_CALLBACK_ALLOWED_USERS=user1,user2
WECOM_CALLBACK_ALLOW_ALL_USERS=false
```

### Weixin

```yaml
WEIXIN_ACCOUNT_ID=...
WEIXIN_TOKEN=...
WEIXIN_BASE_URL=https://ilinkai.weixin.qq.com
WEIXIN_CDN_BASE_URL=https://novac2c.cdn.weixin.qq.com/c2c
WEIXIN_DM_POLICY=open                   # open | allowlist | pairing | disabled
WEIXIN_GROUP_POLICY=open                # open | allowlist | disabled
WEIXIN_ALLOWED_USERS=user1,user2
WEIXIN_GROUP_ALLOWED_USERS=group1,group2
WEIXIN_HOME_CHANNEL=chat_id
WEIXIN_HOME_CHANNEL_NAME="My Bot"
WEIXIN_ALLOW_ALL_USERS=false
```

### BlueBubbles (iMessage)

```yaml
BLUEBUBBLES_SERVER_URL=http://192.168.1.10:1234
BLUEBUBBLES_PASSWORD=server_password
BLUEBUBBLES_WEBHOOK_HOST=127.0.0.1
BLUEBUBBLES_WEBHOOK_PORT=8645
BLUEBUBBLES_HOME_CHANNEL=+1234567890
BLUEBUBBLES_ALLOWED_USERS=user1,user2
BLUEBUBBLES_ALLOW_ALL_USERS=false
```

### QQ Bot

```yaml
QQ_APP_ID=...
QQ_CLIENT_SECRET=...
QQ_STT_API_KEY=...                      # Optional external STT
QQ_STT_BASE_URL=...
QQ_STT_MODEL=...
QQ_ALLOWED_USERS=openid1,openid2
QQ_GROUP_ALLOWED_USERS=group1,group2
QQ_ALLOW_ALL_USERS=false
QQBOT_HOME_CHANNEL=openid
```

### Mattermost

```yaml
MATTERMOST_URL=https://mm.example.com
MATTERMOST_TOKEN=token_here
MATTERMOST_ALLOWED_USERS=user1,user2
MATTERMOST_HOME_CHANNEL=channel_id
MATTERMOST_REQUIRE_MENTION=true
MATTERMOST_FREE_RESPONSE_CHANNELS=channel1,channel2
MATTERMOST_REPLY_MODE=thread            # thread | off
```

### Matrix

```yaml
MATRIX_HOMESERVER=https://matrix.org
MATRIX_ACCESS_TOKEN=syt_...
MATRIX_USER_ID=@hermes:matrix.org
MATRIX_PASSWORD=password               # Alternative to access token
MATRIX_ALLOWED_USERS=@alice:matrix.org,@bob:matrix.org
MATRIX_HOME_ROOM=!abc123:matrix.org
MATRIX_ENCRYPTION=false
MATRIX_REQUIRE_MENTION=true
MATRIX_FREE_RESPONSE_ROOMS=!room1:matrix.org
MATRIX_AUTO_THREAD=true
MATRIX_DM_MENTION_THREADS=false
MATRIX_RECOVERY_KEY=...
```

### Home Assistant

```yaml
HASS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
HASS_URL=http://homeassistant.local:8123
```

### Webhooks

```yaml
WEBHOOK_ENABLED=true
WEBHOOK_PORT=8644
WEBHOOK_SECRET=global_secret_here
```

### API Server

```yaml
API_SERVER_ENABLED=true
API_SERVER_KEY=bearer_token_here
API_SERVER_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
API_SERVER_PORT=8642
API_SERVER_HOST=127.0.0.1              # 0.0.0.0 for network access
API_SERVER_MODEL_NAME=hermes-agent
```

### Gateway Proxy Mode

```yaml
GATEWAY_PROXY_URL=http://remote-hermes:8642
GATEWAY_PROXY_KEY=bearer_token
```

---

## Session Management

```yaml
session_reset:
  mode: "both"                        # both | idle | daily | none
  idle_minutes: 1440                  # 24 hours
  at_hour: 4                          # 4 AM local time

group_sessions_per_user: true         # One session per user in groups

streaming:
  enabled: false
  transport: "edit"                   # edit = progressive editMessageText
  edit_interval: 0.3                  # Seconds between edits
  buffer_threshold: 40                # Chars before forcing edit
  cursor: " ▉"                        # Cursor shown during streaming
```

### Session Logging

Sessions automatically saved to `logs/session_YYYYMMDD_HHMMSS_UUID.json` with full trajectory.

---

## Security & Approval

```yaml
approvals:
  mode: "manual"                      # manual | smart | off
```

### Dangerous Command Patterns

Automatically detected and require approval:

- Recursive delete (`rm -rf`)
- Destructive file operations (`> /etc/`, `dd if=/dev/zero`)
- SQL destructive operations (`DROP TABLE`, `DELETE FROM` without `WHERE`)
- System config overwrites

### Command Allowlist

```yaml
command_allowlist:
  - "^rm -rf /tmp"
  - "^sudo reboot"
```

---

## Cron & Scheduling

```yaml
cron:
  script_timeout_seconds: 120         # Pre-run script timeout
```

### Cron Job Configuration

```yaml
# In ~/.hermes/cron/
# Jobs are stored as JSON with schedule, prompt, delivery platform
```

---

## Skills Configuration

```yaml
skills:
  creation_nudge_interval: 15         # Remind agent every N iterations
  external_dirs:                      # Read-only skill directories
    - ~/.agents/skills
    - /home/shared/team-skills
  config:                             # Per-skill configuration
    myplugin:
      path: ~/myplugin-data
```

---

## Profiles (Multi-Agent)

### Creating Profiles

```bash
hermes profile create coder           # Blank profile
hermes profile create work --clone    # Clone config only
hermes profile create backup --clone-all  # Clone everything
hermes profile create work --clone --clone-from coder  # Clone from specific
```

### Using Profiles

```bash
coder chat                            # Use profile command alias
hermes -p coder chat                  # Explicit -p flag
hermes --profile=coder chat           # Long form
hermes profile use coder              # Set sticky default
```

### Profile Structure

```
~/.hermes/profiles/
├── coder/
│   ├── config.yaml
│   ├── .env
│   ├── SOUL.md
│   ├── memories/
│   ├── sessions/
│   ├── skills/
│   ├── cron/
│   └── state.db
└── assistant/
    ├── config.yaml
    ├── .env
    └── SOUL.md
```

### Managing Profiles

```bash
hermes profile list                   # Show all profiles
hermes profile show coder             # Detailed info
hermes profile rename coder dev-bot   # Rename
hermes profile export coder           # Export to coder.tar.gz
hermes profile import coder.tar.gz    # Import from archive
hermes profile delete coder           # Delete (confirm required)
```

---

## Environment Variables Reference

### LLM Providers

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `OPENROUTER_BASE_URL` | Override OpenRouter base URL |
| `AI_GATEWAY_API_KEY` | Vercel AI Gateway API key |
| `AI_GATEWAY_BASE_URL` | Override AI Gateway base URL |
| `OPENAI_API_KEY` | OpenAI API key (custom endpoints) |
| `OPENAI_BASE_URL` | Custom OpenAI-compatible endpoint |
| `COPILOT_GITHUB_TOKEN` | GitHub token for Copilot |
| `GH_TOKEN` | GitHub token (Copilot fallback) |
| `GITHUB_TOKEN` | GitHub token (Copilot fallback) |
| `HERMES_COPILOT_ACP_COMMAND` | Override Copilot ACP binary path |
| `COPILOT_CLI_PATH` | Alias for HERMES_COPILOT_ACP_COMMAND |
| `HERMES_COPILOT_ACP_ARGS` | Override Copilot ACP arguments |
| `COPILOT_ACP_BASE_URL` | Override Copilot ACP base URL |
| `GLM_API_KEY` | z.ai / ZhipuAI GLM API key |
| `ZAI_API_KEY` | Alias for GLM_API_KEY |
| `Z_AI_API_KEY` | Alias for GLM_API_KEY |
| `GLM_BASE_URL` | Override z.ai base URL |
| `KIMI_API_KEY` | Kimi / Moonshot AI API key |
| `KIMI_BASE_URL` | Override Kimi base URL |
| `KIMI_CN_API_KEY` | Kimi China API key |
| `ARCEEAI_API_KEY` | Arcee AI API key |
| `ARCEE_BASE_URL` | Override Arcee base URL |
| `MINIMAX_API_KEY` | MiniMax global API key |
| `MINIMAX_BASE_URL` | Override MiniMax base URL |
| `MINIMAX_CN_API_KEY` | MiniMax China API key |
| `MINIMAX_CN_BASE_URL` | Override MiniMax China base URL |
| `KILOCODE_API_KEY` | Kilo Code API key |
| `KILOCODE_BASE_URL` | Override Kilo Code base URL |
| `XIAOMI_API_KEY` | Xiaomi MiMo API key |
| `XIAOMI_BASE_URL` | Override Xiaomi MiMo base URL |
| `HF_TOKEN` | Hugging Face token |
| `HF_BASE_URL` | Override Hugging Face base URL |
| `GOOGLE_API_KEY` | Google AI Studio API key |
| `GEMINI_API_KEY` | Alias for GOOGLE_API_KEY |
| `GEMINI_BASE_URL` | Override Google AI Studio base URL |
| `HERMES_GEMINI_CLIENT_ID` | OAuth client ID for google-gemini-cli |
| `HERMES_GEMINI_CLIENT_SECRET` | OAuth client secret |
| `HERMES_GEMINI_PROJECT_ID` | GCP project ID for paid Gemini |
| `ANTHROPIC_API_KEY` | Anthropic Console API key |
| `ANTHROPIC_TOKEN` | Manual/legacy Anthropic OAuth token |
| `DASHSCOPE_API_KEY` | Alibaba DashScope API key |
| `DASHSCOPE_BASE_URL` | Custom DashScope base URL |
| `DEEPSEEK_API_KEY` | DeepSeek API key |
| `DEEPSEEK_BASE_URL` | Custom DeepSeek API base URL |
| `OPENCODE_ZEN_API_KEY` | OpenCode Zen API key |
| `OPENCODE_ZEN_BASE_URL` | Override OpenCode Zen base URL |
| `OPENCODE_GO_API_KEY` | OpenCode Go API key |
| `OPENCODE_GO_BASE_URL` | Override OpenCode Go base URL |
| `CLAUDE_CODE_OAUTH_TOKEN` | Explicit Claude Code token override |
| `HERMES_MODEL` | Override model name at process level |
| `VOICE_TOOLS_OPENAI_KEY` | Preferred OpenAI key for STT/TTS |
| `HERMES_LOCAL_STT_COMMAND` | Local speech-to-text command template |
| `HERMES_LOCAL_STT_LANGUAGE` | Default language for local STT |
| `HERMES_HOME` | Override Hermes config directory |

### Provider Auth (OAuth)

| Variable | Description |
|----------|-------------|
| `HERMES_INFERENCE_PROVIDER` | Override provider selection |
| `HERMES_PORTAL_BASE_URL` | Override Nous Portal URL |
| `NOUS_INFERENCE_BASE_URL` | Override Nous inference API URL |
| `HERMES_NOUS_MIN_KEY_TTL_SECONDS` | Min agent key TTL before re-mint |
| `HERMES_NOUS_TIMEOUT_SECONDS` | HTTP timeout for Nous flows |
| `HERMES_DUMP_REQUESTS` | Dump API request payloads to logs |
| `HERMES_PREFILL_MESSAGES_FILE` | Path to JSON prefill messages |
| `HERMES_TIMEZONE` | IANA timezone override |

### Tool APIs

| Variable | Description |
|----------|-------------|
| `PARALLEL_API_KEY` | Parallel.ai web search |
| `FIRECRAWL_API_KEY` | Firecrawl web scraping |
| `FIRECRAWL_API_URL` | Custom Firecrawl endpoint |
| `TAVILY_API_KEY` | Tavily web search |
| `EXA_API_KEY` | Exa web search |
| `BROWSERBASE_API_KEY` | Browserbase browser automation |
| `BROWSERBASE_PROJECT_ID` | Browserbase project ID |
| `BROWSER_USE_API_KEY` | Browser Use cloud API key |
| `FIRECRAWL_BROWSER_TTL` | Firecrawl browser session TTL |
| `BROWSER_CDP_URL` | Chrome DevTools Protocol URL |
| `CAMOFOX_URL` | Camofox anti-detection browser URL |
| `BROWSER_INACTIVITY_TIMEOUT` | Browser session inactivity timeout |
| `FAL_KEY` | FAL.ai image generation |
| `GROQ_API_KEY` | Groq Whisper STT API key |
| `ELEVENLABS_API_KEY` | ElevenLabs TTS API key |
| `STT_GROQ_MODEL` | Override Groq STT model |
| `GROQ_BASE_URL` | Override Groq STT endpoint |
| `STT_OPENAI_MODEL` | Override OpenAI STT model |
| `STT_OPENAI_BASE_URL` | Override OpenAI STT endpoint |
| `GITHUB_TOKEN` | GitHub token (Skills Hub) |
| `HONCHO_API_KEY` | Honcho cross-session modeling |
| `HONCHO_BASE_URL` | Self-hosted Honcho base URL |
| `SUPERMEMORY_API_KEY` | Supermemory semantic memory |
| `TINKER_API_KEY` | Tinker RL training |
| `WANDB_API_KEY` | Weights & Biases RL metrics |
| `DAYTONA_API_KEY` | Daytona cloud sandboxes |

### Terminal Backend

| Variable | Description |
|----------|-------------|
| `TERMINAL_ENV` | Backend: local, docker, ssh, singularity, modal, daytona |
| `TERMINAL_DOCKER_IMAGE` | Docker image |
| `TERMINAL_DOCKER_FORWARD_ENV` | JSON array of env vars to forward |
| `TERMINAL_DOCKER_VOLUMES` | Comma-separated volume mounts |
| `TERMINAL_DOCKER_MOUNT_CWD_TO_WORKSPACE` | Mount launch cwd to /workspace |
| `TERMINAL_SINGULARITY_IMAGE` | Singularity image or .sif path |
| `TERMINAL_MODAL_IMAGE` | Modal container image |
| `TERMINAL_DAYTONA_IMAGE` | Daytona sandbox image |
| `TERMINAL_TIMEOUT` | Command timeout in seconds |
| `TERMINAL_LIFETIME_SECONDS` | Max lifetime for terminal sessions |
| `TERMINAL_CWD` | Working directory for all sessions |
| `SUDO_PASSWORD` | Sudo password (plaintext) |
| `TERMINAL_SSH_HOST` | Remote server hostname |
| `TERMINAL_SSH_USER` | SSH username |
| `TERMINAL_SSH_PORT` | SSH port |
| `TERMINAL_SSH_KEY` | Path to SSH private key |
| `TERMINAL_SSH_PERSISTENT` | Override persistent shell for SSH |
| `TERMINAL_CONTAINER_CPU` | CPU cores |
| `TERMINAL_CONTAINER_MEMORY` | Memory in MB |
| `TERMINAL_CONTAINER_DISK` | Disk in MB |
| `TERMINAL_CONTAINER_PERSISTENT` | Persist container filesystem |
| `TERMINAL_SANDBOX_DIR` | Host directory for workspaces |
| `TERMINAL_PERSISTENT_SHELL` | Enable persistent shell |
| `TERMINAL_LOCAL_PERSISTENT` | Enable persistent shell for local |
| `MESSAGING_CWD` | Working directory for messaging mode |

### Agent Behavior

| Variable | Description |
|----------|-------------|
| `HERMES_MAX_ITERATIONS` | Max tool-calling iterations |
| `HERMES_TOOL_PROGRESS` | Deprecated; use display.tool_progress |
| `HERMES_TOOL_PROGRESS_MODE` | Deprecated; use display.tool_progress |
| `HERMES_HUMAN_DELAY_MODE` | Response pacing: off, natural, custom |
| `HERMES_HUMAN_DELAY_MIN_MS` | Custom delay minimum |
| `HERMES_HUMAN_DELAY_MAX_MS` | Custom delay maximum |
| `HERMES_QUIET` | Suppress non-essential output |
| `HERMES_API_TIMEOUT` | LLM API call timeout (seconds) |
| `HERMES_STREAM_READ_TIMEOUT` | Streaming socket read timeout |
| `HERMES_STREAM_STALE_TIMEOUT` | Stale stream detection timeout |
| `HERMES_EXEC_ASK` | Enable execution approval prompts |
| `HERMES_ENABLE_PROJECT_PLUGINS` | Enable repo-local plugins |
| `HERMES_BACKGROUND_NOTIFICATIONS` | Background process notification mode |
| `HERMES_EPHEMERAL_SYSTEM_PROMPT` | Ephemeral system prompt |

### Interface

| Variable | Description |
|----------|-------------|
| `HERMES_TUI` | Launch TUI instead of classic CLI |
| `HERMES_TUI_DIR` | Path to prebuilt ui-tui directory |

### Cron Scheduler

| Variable | Description |
|----------|-------------|
| `HERMES_CRON_TIMEOUT` | Inactivity timeout for cron runs |
| `HERMES_CRON_SCRIPT_TIMEOUT` | Timeout for pre-run scripts |

### Session Settings

| Variable | Description |
|----------|-------------|
| `SESSION_IDLE_MINUTES` | Reset sessions after N minutes |
| `SESSION_RESET_HOUR` | Daily reset hour (24h format) |

### Auxiliary Task Overrides

| Variable | Description |
|----------|-------------|
| `AUXILIARY_VISION_PROVIDER` | Override vision provider |
| `AUXILIARY_VISION_MODEL` | Override vision model |
| `AUXILIARY_VISION_BASE_URL` | Custom vision endpoint |
| `AUXILIARY_VISION_API_KEY` | Vision API key |
| `AUXILIARY_WEB_EXTRACT_PROVIDER` | Override web extract provider |
| `AUXILIARY_WEB_EXTRACT_MODEL` | Override web extract model |
| `AUXILIARY_WEB_EXTRACT_BASE_URL` | Custom web extract endpoint |
| `AUXILIARY_WEB_EXTRACT_API_KEY` | Web extract API key |

---

## Configuration Management Commands

```bash
# View current configuration
hermes config

# Edit config.yaml in $EDITOR
hermes config edit

# Set a specific value (dot-notation for nested)
hermes config set model.default anthropic/claude-opus-4.6
hermes config set terminal.backend docker
hermes config set OPENROUTER_API_KEY sk-or-...

# Check for missing options after updates
hermes config check

# Interactively add missing options
hermes config migrate

# View .env file path
hermes config env-path

# Show current auxiliary model settings
hermes config show
```

---

## Configuration Precedence Summary

```
CLI args (highest priority)
    ↓
config.yaml
    ↓
.env
    ↓
Built-in defaults (lowest priority)
```

**Key Rules:**
- Secrets always go in `.env`
- Behavioral settings go in `config.yaml`
- `config.yaml` wins for non-secret settings when both are set
- Environment variables can override `.env` values
- Use `${VAR_NAME}` syntax in `config.yaml` to reference env vars

---

## Default Values Summary

| Setting | Default |
|---------|---------|
| `model.provider` | `auto` |
| `model.context_length` | Auto-detected |
| `terminal.backend` | `local` |
| `terminal.timeout` | 180 seconds |
| `terminal.lifetime_seconds` | 300 seconds |
| `compression.enabled` | `true` |
| `compression.threshold` | 0.50 (50%) |
| `compression.target_ratio` | 0.20 (20%) |
| `compression.protect_last_n` | 20 messages |
| `memory.memory_enabled` | `true` |
| `memory.user_profile_enabled` | `true` |
| `memory.memory_char_limit` | 2200 (~800 tokens) |
| `memory.user_char_limit` | 1375 (~500 tokens) |
| `memory.nudge_interval` | 10 turns |
| `memory.flush_min_turns` | 6 turns |
| `agent.max_turns` | 90 |
| `agent.reasoning_effort` | `medium` |
| `display.skin` | `default` |
| `display.tool_progress` | `all` |
| `display.compact` | `false` |
| `display.show_reasoning` | `false` |
| `session_reset.mode` | `both` |
| `session_reset.idle_minutes` | 1440 (24 hours) |
| `session_reset.at_hour` | 4 (4 AM) |
| `group_sessions_per_user` | `true` |
| `browser.inactivity_timeout` | 120 seconds |
| `code_execution.timeout` | 300 seconds (5 min) |
| `code_execution.max_tool_calls` | 50 |
| `delegation.max_iterations` | 50 |
| `skills.creation_nudge_interval` | 15 iterations |
| `file_read_max_chars` | 100000 (~25-35K tokens) |

---

## Tips & Best Practices

1. **Use `hermes config set` for changes** — automatically routes to correct file
2. **Keep secrets in `.env`** — never commit API keys to config.yaml
3. **Use environment variable substitution** — reference secrets in config.yaml via `${VAR_NAME}`
4. **Profile isolation** — each profile has its own config, API keys, memory
5. **Compression tuning** — lower threshold = more aggressive compression
6. **Terminal backend choice** — local for dev, docker for safety, ssh for remote
7. **Memory limits** — keep memory char limits reasonable to avoid token waste
8. **Reasoning effort** — higher effort = better results but more tokens/latency
9. **Toolset customization** — disable tools you don't need via platform_toolsets
10. **Backup profiles** — use `hermes profile export` for snapshots

---

## Configuration Version History

| Version | Changes |
|---------|---------|
| 10 | Current (April 2026) |
| 9 | Previous version |
| ... | ... |

Run `hermes config migrate` to upgrade to the latest schema version.

---

**Last Updated:** April 18, 2026  
**Source:** Official Hermes Agent Documentation  
**Repository:** https://github.com/NousResearch/hermes-agent

