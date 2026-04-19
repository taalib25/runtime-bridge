# HERMES AGENT - COMPLETE CONFIGURATION & CAPABILITIES REFERENCE

## TABLE OF CONTENTS
1. [Configuration Files Structure](#configuration-files-structure)
2. [Config.yaml Complete Reference](#configyaml-complete-reference)
3. [Environment Variables Complete Reference](#environment-variables-complete-reference)
4. [Built-in Tools (47 Tools)](#built-in-tools-47-tools)
5. [Toolsets System](#toolsets-system)
6. [Skills System (96 Bundled + 22 Optional)](#skills-system-96-bundled--22-optional)
7. [Messaging Platforms (16+ Platforms)](#messaging-platforms-16-platforms)
8. [API Server Configuration](#api-server-configuration)
9. [Terminal Backends (6 Options)](#terminal-backends-6-options)
10. [Browser Automation (5 Backends)](#browser-automation-5-backends)
11. [LLM Providers (23+ Providers)](#llm-providers-23-providers)
12. [Security Features](#security-features)
13. [Voice & Audio Capabilities](#voice--audio-capabilities)
14. [Memory & Persistence](#memory--persistence)
15. [Profiles & Multi-User Setup](#profiles--multi-user-setup)

---

## CONFIGURATION FILES STRUCTURE

```
~/.hermes/ (HERMES_HOME in K8s = /opt/data)
├── config.yaml              # Main configuration settings
├── .env                     # API keys and secrets
├── auth.json                # OAuth credentials
├── SOUL.md                  # Agent personality (system prompt slot #1)
├── memories/                # Persistent memory files
│   ├── MEMORY.md            # Cross-session memory
│   └── USER.md              # User profile memory
├── skills/                  # Custom and bundled skills
│   ├── category/skill/SKILL.md
│   └── .hub/                # Skills Hub state
├── cron/                    # Scheduled jobs
├── sessions/                # Chat history and session data
├── logs/                    # Error and gateway logs
└── plugins/                 # Custom plugins
```

### Configuration Precedence Order
1. **CLI arguments** - Highest priority (per-invocation override)
2. **`~/.hermes/config.yaml`** - Primary config file for non-secret settings
3. **`~/.hermes/.env`** - Fallback for env vars; **required** for secrets
4. **Built-in defaults** - Hardcoded safe defaults when nothing else is set

### Environment Variable Substitution
Reference env vars in `config.yaml` using `${VAR_NAME}` syntax:
```yaml
auxiliary:
  vision:
    api_key: ${GOOGLE_API_KEY}
    base_url: ${CUSTOM_VISION_URL}
```

---

## CONFIG.YAML COMPLETE REFERENCE

### Model Configuration
```yaml
model:
  provider: auto                    # openrouter, anthropic, openai, nous, codex, copilot, custom, zai, kimi-coding, minimax, arcee, huggingface, etc.
  model: anthropic/claude-opus-4.6  # Model identifier
  base_url: https://openrouter.ai/api/v1  # Custom endpoint
```

### Agent Behavior
```yaml
agent:
  max_turns: 90                     # Max iterations per conversation
  gateway_timeout: 1800             # Gateway timeout in seconds (30 min)
  restart_drain_timeout: 60         # Restart drain timeout
  tool_use_enforcement: auto        # auto | true | false | ["model-substring", ...]
  reasoning_effort: ""              # none, minimal, low, medium, high, xhigh
```

### Terminal Backend
```yaml
terminal:
  backend: local                    # local | docker | ssh | modal | daytona | singularity
  cwd: "."                          # Working directory
  timeout: 180                      # Command timeout in seconds
  persistent_shell: true            # Keep shell state between commands
  env_passthrough: []               # Env vars to forward to sandboxed execution
  
  # Docker-specific
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_mount_cwd_to_workspace: false
  docker_forward_env: []
  docker_volumes: []
  
  # Resource limits (all container backends)
  container_cpu: 1
  container_memory: 5120            # MB
  container_disk: 51200             # MB
  container_persistent: true        # Persist filesystem across sessions
```

### Browser Configuration
```yaml
browser:
  inactivity_timeout: 120           # Seconds before auto-closing idle sessions
  command_timeout: 30               # Timeout for browser commands
  record_sessions: false            # Auto-record browser sessions as WebM videos
  allow_private_urls: false         # Block private network access
```

### Context Compression
```yaml
compression:
  enabled: true                     # Enable context compression
  threshold: 0.50                   # Compress at this % of context limit
  target_ratio: 0.20                # Fraction of threshold to preserve as recent tail
  protect_last_n: 20                # Min recent messages to keep uncompressed

# Auxiliary compression model
auxiliary:
  compression:
    model: "google/gemini-3-flash-preview"
    provider: "auto"
    base_url: null
```

### Memory Configuration
```yaml
memory:
  memory_enabled: true
  user_profile_enabled: true
  memory_char_limit: 2200           # ~800 tokens
  user_char_limit: 1375             # ~500 tokens
```

### Display Settings
```yaml
display:
  tool_progress: all                # off | new | all | verbose
  tool_progress_command: false       # Enable /verbose slash command
  interim_assistant_messages: true   # Send mid-turn updates as separate messages
  skin: default                     # CLI skin
  compact: false                    # Compact output mode
  bell_on_complete: false           # Play terminal bell when agent finishes
  show_reasoning: false             # Show model reasoning above responses
  streaming: false                  # Stream tokens to terminal
  show_cost: false                  # Show estimated $ cost
  tool_preview_length: 0            # Max chars for tool call previews
```

### File Read Safety
```yaml
file_read_max_chars: 100000         # Max chars per read_file call
```

### Web Configuration
```yaml
web:
  backend: firecrawl               # firecrawl | parallel | tavily | exa
```

### TTS Configuration
```yaml
tts:
  provider: "edge"                  # edge | elevenlabs | openai | minimax | mistral | neutts
  speed: 1.0                        # Global speed multiplier
  edge:
    voice: "en-US-AriaNeural"
    speed: 1.0
  elevenlabs:
    voice_id: "pNInz6obpgDQGcFmaJgB"
    model_id: "eleven_multilingual_v2"
  openai:
    model: "gpt-4o-mini-tts"
    voice: "alloy"
    speed: 1.0
```

### STT Configuration
```yaml
stt:
  provider: "local"                 # local | groq | openai | mistral
  local:
    model: "base"                   # tiny, base, small, medium, large-v3
  openai:
    model: "whisper-1"
```

### Voice Mode (CLI)
```yaml
voice:
  record_key: "ctrl+b"              # Push-to-talk key
  max_recording_seconds: 120        # Hard stop for long recordings
  auto_tts: false                   # Enable spoken replies automatically
  silence_threshold: 200            # RMS threshold for speech detection
  silence_duration: 3.0             # Seconds of silence before auto-stop
```

### Privacy
```yaml
privacy:
  redact_pii: false                 # Strip PII from LLM context (gateway only)
```

### Quick Commands
```yaml
quick_commands:
  status:
    type: exec
    command: systemctl status hermes-agent
  gpu:
    type: exec
    command: nvidia-smi --query-gpu=name,utilization.gpu,memory.used,memory.total --format=csv,noheader
```

### Human Delay
```yaml
human_delay:
  mode: "off"                       # off | natural | custom
  min_ms: 800                       # Minimum delay (custom mode)
  max_ms: 2500                      # Maximum delay (custom mode)
```

### Code Execution
```yaml
code_execution:
  timeout: 300                      # Max execution time in seconds
  max_tool_calls: 50                # Max tool calls within code execution
```

### Group Chat Session Isolation
```yaml
group_sessions_per_user: true        # true = per-user isolation in groups/channels
```

### Unauthorized DM Behavior
```yaml
unauthorized_dm_behavior: pair       # pair | ignore

# Platform overrides
whatsapp:
  unauthorized_dm_behavior: ignore
```

### Credential Pool Strategies
```yaml
credential_pool_strategies:
  openrouter: round_robin           # fill_first | round_robin | least_used | random
  anthropic: least_used
```

### Auxiliary Models
```yaml
auxiliary:
  # Image analysis
  vision:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 120
    download_timeout: 30
  
  # Web page summarization
  web_extract:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 360
  
  # Dangerous command approval
  approval:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 30
  
  # Context compression
  compression:
    timeout: 120
  
  # Session search
  session_search:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 30
  
  # Skills hub
  skills_hub:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 30
  
  # MCP tool dispatch
  mcp:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 30
  
  # Memory flush
  flush_memories:
    provider: "auto"
    model: ""
    base_url: ""
    api_key: ""
    timeout: 30
```

### Skills Configuration
```yaml
skills:
  config: {}                        # Skill-specific configuration
  preload: []                       # Skills to load at startup
  external_dirs: []                 # External skill directories
```

### Streaming Configuration
```yaml
streaming:
  enabled: true                     # Enable progressive message editing
  transport: edit                   # edit | off
  edit_interval: 0.3                # Seconds between message edits
  buffer_threshold: 40              # Characters before forcing an edit flush
  cursor: " ▉"                      # Cursor shown during streaming
```

### Tool Progress Overrides
```yaml
display:
  tool_progress_overrides:
    signal: 'off'
    telegram: verbose
    slack: 'off'
```

### Website Blocklist
```yaml
security:
  website_blocklist:
    enabled: true
    domains:
      - "*.internal.company.com"
      - "admin.example.com"
    shared_files:
      - "/etc/hermes/blocked-sites.txt"
```

### Tirith Security Scanning
```yaml
security:
  tirith_enabled: true
  tirith_path: "tirith"
  tirith_timeout: 5
  tirith_fail_open: true
```

### Cron Job Configuration
```yaml
cron:
  script_timeout_seconds: 120       # Timeout for pre-run scripts
```

### Context Engine
```yaml
context:
  engine: "compressor"              # compressor | lcm | custom-plugin
```

### Fallback Model
```yaml
fallback_model:
  provider: openrouter
  model: anthropic/claude-sonnet-4
```

### Provider Routing
```yaml
provider_routing:
  sort: "price"                     # price | throughput | latency
  only: []                          # Only use these providers
  ignore: []                        # Skip these providers
  order: []                         # Try providers in this order
  require_parameters: true          # Only use providers supporting all request params
  data_collection: "allow"          # allow | deny
```

---

## ENVIRONMENT VARIABLES COMPLETE REFERENCE

### LLM Providers
| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `ANTHROPIC_API_KEY` | Anthropic Console API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `COPILOT_GITHUB_TOKEN` | GitHub token for Copilot API |
| `GLM_API_KEY` | ZhipuAI GLM API key |
| `KIMI_API_KEY` | Kimi/Moonshot AI API key |
| `ARCEEAI_API_KEY` | Arcee AI API key |
| `MINIMAX_API_KEY` | MiniMax API key |
| `HF_TOKEN` | Hugging Face token |
| `GOOGLE_API_KEY` | Google AI Studio API key |
| `DASHSCOPE_API_KEY` | Alibaba Cloud DashScope API key |
| `DEEPSEEK_API_KEY` | DeepSeek API key |

### Tool APIs
| Variable | Description |
|----------|-------------|
| `PARALLEL_API_KEY` | Parallel AI web search |
| `FIRECRAWL_API_KEY` | Firecrawl web scraping |
| `TAVILY_API_KEY` | Tavily web search |
| `EXA_API_KEY` | Exa web search |
| `BROWSERBASE_API_KEY` | Browserbase automation |
| `FAL_KEY` | FAL.ai image generation |
| `GROQ_API_KEY` | Groq Whisper STT |
| `ELEVENLABS_API_KEY` | ElevenLabs TTS |
| `HONCHO_API_KEY` | Honcho cross-session memory |
| `WANDB_API_KEY` | Weights & Biases |

### Messaging Platforms
| Variable | Description |
|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `SLACK_BOT_TOKEN` | Slack bot token |
| `WHATSAPP_ENABLED` | Enable WhatsApp bridge |
| `SIGNAL_HTTP_URL` | Signal CLI daemon URL |
| `TWILIO_ACCOUNT_SID` | Twilio Account SID |
| `EMAIL_ADDRESS` | Email address for email gateway |
| `HASS_TOKEN` | Home Assistant token |

### API Server & Gateway
| Variable | Description |
|----------|-------------|
| `API_SERVER_ENABLED` | Enable API server (`true`/`false`) |
| `API_SERVER_KEY` | API server bearer token |
| `API_SERVER_PORT` | API server port (default: 8642) |
| `API_SERVER_HOST` | API server host (default: 127.0.0.1) |
| `API_SERVER_CORS_ORIGINS` | CORS origins for browser access |
| `WEBHOOK_ENABLED` | Enable webhook platform |
| `WEBHOOK_PORT` | Webhook server port (default: 8644) |
| `WEBHOOK_SECRET` | Global webhook HMAC secret |

### Terminal Backend
| Variable | Description |
|----------|-------------|
| `TERMINAL_ENV` | Backend: local, docker, ssh, singularity, modal, daytona |
| `TERMINAL_DOCKER_IMAGE` | Docker image |
| `TERMINAL_SSH_HOST` | SSH hostname |
| `TERMINAL_SSH_USER` | SSH username |
| `DAYTONA_API_KEY` | Daytona cloud sandboxes |

### Agent Behavior
| Variable | Description |
|----------|-------------|
| `HERMES_MAX_ITERATIONS` | Max tool-calling iterations (default: 90) |
| `HERMES_HUMAN_DELAY_MODE` | Response pacing: off/natural/custom |
| `HERMES_API_TIMEOUT` | LLM API call timeout (default: 1800s) |
| `HERMES_STREAM_READ_TIMEOUT` | Streaming socket read timeout (default: 120s) |

---

## BUILT-IN TOOLS (47 TOOLS)

### Browser Tools (10)
1. `browser_navigate` - Navigate to URL
2. `browser_snapshot` - Get accessibility tree snapshot
3. `browser_click` - Click element by ref ID
4. `browser_type` - Type into input field
5. `browser_scroll` - Scroll page
6. `browser_press` - Press keyboard key
7. `browser_back` - Navigate back
8. `browser_get_images` - Get image URLs from page
9. `browser_vision` - Screenshot + vision analysis
10. `browser_console` - Get browser console output

### Web Tools (2)
1. `web_search` - Search web (5 results)
2. `web_extract` - Extract & summarize page content

### Terminal Tools (2)
1. `terminal` - Execute shell commands
2. `process` - Manage background processes

### File Tools (4)
1. `read_file` - Read text files with pagination
2. `write_file` - Write/overwrite files
3. `patch` - Targeted find-and-replace edits
4. `search_files` - Search file contents or find by name

### Vision & Media Tools (3)
1. `vision_analyze` - Analyze images with AI vision
2. `image_generate` - Generate images from text (FAL.ai)
3. `text_to_speech` - Convert text to audio

### Code Execution Tool (1)
1. `execute_code` - Run Python scripts that call Hermes tools programmatically

### Agent Orchestration Tools (4)
1. `todo` - Task list management
2. `clarify` - Ask user for clarification
3. `delegate_task` - Spawn isolated subagents
4. `mixture_of_agents` - Route to 5 frontier LLMs for consensus

### Memory & Recall Tools (3)
1. `memory` - Save persistent cross-session memory
2. `session_search` - Search past conversations
3. `skills_list`, `skill_view`, `skill_manage` - Skill CRUD operations

### Scheduling & Messaging Tools (2)
1. `cronjob` - Create/manage scheduled tasks
2. `send_message` - Send to messaging platforms

### Home Assistant Tools (4)
1. `ha_list_entities` - List smart home devices
2. `ha_list_services` - List available actions
3. `ha_get_state` - Get device state
4. `ha_call_service` - Control devices

### RL Training Tools (10)
1. `rl_list_environments` - List available training environments
2. `rl_select_environment` - Choose environment
3. `rl_get_current_config` - View settings
4. `rl_edit_config` - Modify settings
5. `rl_start_training` - Begin training run
6. `rl_stop_training` - Stop training
7. `rl_test_inference` - Quick inference test
8. `rl_check_status` - Get metrics
9. `rl_get_results` - Final results
10. `rl_list_runs` - View all runs

### Utility Tools (2)
1. `clarify` - Ask user questions (multiple choice or open-ended)

---

## TOOLSETS SYSTEM

### Core Toolsets (19)
| Toolset | Tools | Purpose |
|---------|-------|---------|
| `browser` | 10 browser tools + web_search | Full browser automation |
| `web` | web_search, web_extract | Web research |
| `terminal` | terminal, process | Shell execution |
| `file` | read_file, write_file, patch, search_files | File operations |
| `vision` | vision_analyze | Image analysis |
| `image_gen` | image_generate | Image generation |
| `tts` | text_to_speech | Text-to-speech |
| `code_execution` | execute_code | Sandboxed Python |
| `delegation` | delegate_task | Subagent spawning |
| `todo` | todo | Task management |
| `memory` | memory | Persistent memory |
| `session_search` | session_search | Past conversation search |
| `skills` | skill_manage, skill_view, skills_list | Skill management |
| `cronjob` | cronjob | Scheduled tasks |
| `messaging` | send_message | Platform messaging |
| `moa` | mixture_of_agents | Multi-model consensus |
| `homeassistant` | ha_* (4 tools) | Smart home control |
| `rl` | rl_* (10 tools) | RL training |
| `clarify` | clarify | User clarification |

### Composite Toolsets
| Toolset | Expands to | Use case |
|---------|-----------|----------|
| `debugging` | file + terminal + web | Debug sessions (no browser) |
| `safe` | web + vision + image_gen + tts | Read-only research (no terminal) |

### Platform Toolsets
- `hermes-cli` - Full toolset (default for CLI)
- `hermes-telegram`, `hermes-discord`, `hermes-slack`, etc. - Full toolset for each platform
- `hermes-acp` - Drops clarify, cronjob, image_gen, send_message, TTS, HA tools (IDE-focused)
- `hermes-api-server` - Drops clarify, send_message, TTS

### Dynamic Toolsets
- **MCP toolsets** - `mcp-<server>` for each configured MCP server
- **Plugin toolsets** - Custom toolsets from plugins
- **Custom toolsets** - User-defined in config.yaml

### Using Toolsets
```bash
hermes chat --toolsets web,terminal          # Specific toolsets
hermes chat --toolsets debugging             # Composite
hermes chat --toolsets all                   # Everything
hermes tools                                 # Interactive UI
/tools list                                  # In-session
/tools disable browser                       # Disable specific tool
```

---

## SKILLS SYSTEM (96 BUNDLED + 22 OPTIONAL)

### What Are Skills?
Skills are **on-demand knowledge documents** (SKILL.md files) that:
- Follow the **agentskills.io open standard**
- Live in `~/.hermes/skills/` (primary) + external directories
- Are **automatically available as slash commands** (`/skill-name`)
- Use **progressive disclosure** to minimize token usage (~3K tokens for full catalog)
- Can be **created, updated, and deleted by the agent** (procedural memory)

### Skill Directory Structure
```
~/.hermes/skills/
├── mlops/
│   ├── axolotl/
│   │   ├── SKILL.md              # Main instructions (required)
│   │   ├── references/           # Additional docs
│   │   ├── templates/            # Output formats
│   │   ├── scripts/              # Helper scripts
│   │   └── assets/               # Supplementary files
├── devops/
│   └── deploy-k8s/
├── .hub/                         # Skills Hub state
│   ├── lock.json
│   ├── quarantine/
│   └── audit.log
└── .bundled_manifest             # Tracks bundled skills
```

### Bundled Skills Categories (96 total)
- **MLOps** - axolotl, vllm, fine-tuning, evaluation, inference, models
- **DevOps** - Kubernetes, Docker, deployment, webhook subscriptions
- **GitHub** - PR workflows, issue management, code review, repo management
- **Research** - arXiv search, web research, domain intel, paper writing
- **Browser** - Web automation, scraping, dogfood testing
- **Productivity** - Google Workspace, Linear, Notion, Obsidian, PowerPoint
- **Security** - 1Password, OSS forensics, Sherlock OSINT
- **Data** - CSV processing, data analysis, Jupyter notebooks
- **Creative** - ASCII art, Excalidraw, p5.js, meme generation
- **Gaming** - Minecraft modpack servers, Pokemon player
- **Social Media** - Xitter (Twitter), GIF search
- **Email** - Himalaya CLI, AgentMail
- **Smart Home** - OpenHue (Philips Hue)
- **Health** - NeuroSkill BCI integration
- **Blockchain** - Base, Solana blockchain queries
- **And 20+ more categories**

### Optional Skills (22 total)
Installed via `hermes skills install official/...`:
- **Autonomous AI Agents** - Blackbox AI CLI
- **Migration** - OpenClaw migration
- **Specialized Integrations** - Blender MCP, Telephony, Bioinformatics

### SKILL.md Format
```markdown
---
name: my-skill
description: Brief description
version: 1.0.0
platforms: [macos, linux]        # Optional — OS restriction
metadata:
  hermes:
    tags: [category, keywords]
    category: devops
    fallback_for_toolsets: [web]  # Show when web unavailable
    requires_toolsets: [terminal] # Show when terminal available
    config:
      - key: my.setting
        description: "What this controls"
        default: "value"
        prompt: "Setup prompt"
required_environment_variables:
  - name: API_KEY
    prompt: "Your API key"
    help: "Get one at https://..."
---

# Skill Title

## When to Use
Trigger conditions.

## Procedure
1. Step one
2. Step two

## Pitfalls
Known failure modes.

## Verification
How to confirm it worked.
```

### Skills Hub Integration
**Sources:**
1. **Official** - `official/security/1password` (Hermes repo)
2. **skills.sh** - Vercel's public directory
3. **Well-known endpoints** - `/.well-known/skills/index.json`
4. **GitHub** - Direct repo installs
5. **ClawHub** - Community marketplace
6. **LobeHub** - Agent catalog

**Commands:**
```bash
hermes skills browse                                    # Browse all
hermes skills search kubernetes                         # Search
hermes skills inspect openai/skills/k8s                # Preview
hermes skills install openai/skills/k8s                # Install
hermes skills install official/security/1password      # Official
hermes skills list                                     # Installed
hermes skills check                                    # Check updates
hermes skills update                                   # Update
hermes skills audit                                    # Security scan
hermes skills uninstall k8s                            # Remove
hermes skills reset google-workspace --restore         # Restore bundled
hermes skills publish skills/my-skill --to github      # Publish
```

### Progressive Disclosure Pattern
```
Level 0: skills_list()           → [{name, description}, ...]   (~3K tokens)
Level 1: skill_view(name)        → Full content + metadata      (varies)
Level 2: skill_view(name, path)  → Specific reference file      (varies)
```

### External Skill Directories
```yaml
# ~/.hermes/config.yaml
skills:
  external_dirs:
    - ~/.agents/skills
    - /home/shared/team-skills
    - ${SKILLS_REPO}/skills
```

### Agent-Managed Skills
The agent **automatically creates skills** after:
- Completing complex tasks (5+ tool calls)
- Finding working solutions after errors
- User corrections
- Non-trivial workflows

**Actions:**
- `create` - New skill from scratch
- `patch` - Targeted fixes (preferred)
- `edit` - Major rewrites
- `delete` - Remove entirely
- `write_file` - Add supporting files
- `remove_file` - Remove supporting files

---

## MESSAGING PLATFORMS (16+ PLATFORMS)

### Supported Platforms
| Platform | Voice | Images | Files | Threads | Typing | Streaming |
|----------|-------|--------|-------|---------|--------|-----------|
| Telegram | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Discord | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Slack | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| WhatsApp | — | ✅ | ✅ | — | ✅ | ✅ |
| Signal | — | ✅ | ✅ | — | ✅ | ✅ |
| Email | — | ✅ | ✅ | ✅ | — | — |
| SMS (Twilio) | — | — | — | — | — | — |
| Home Assistant | — | — | — | — | — | — |
| Mattermost | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Matrix | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| DingTalk | — | ✅ | ✅ | — | — | ✅ |
| Feishu/Lark | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| WeCom | ✅ | ✅ | ✅ | — | ✅ | ✅ |
| Weixin (WeChat) | ✅ | ✅ | ✅ | — | ✅ | ✅ |
| BlueBubbles (iMessage) | — | ✅ | ✅ | — | ✅ | — |
| QQ Bot | ✅ | ✅ | ✅ | — | ✅ | — |

### Platform Setup Requirements
**Telegram:**
```bash
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_ALLOWED_USERS=123456789,987654321
TELEGRAM_HOME_CHANNEL=-1001234567890
```

**Discord:**
```bash
DISCORD_BOT_TOKEN=your_bot_token
DISCORD_ALLOWED_USERS=111222333444
DISCORD_HOME_CHANNEL=123456789
```

**Slack:**
```bash
SLACK_BOT_TOKEN=xoxb-...
SLACK_APP_TOKEN=xapp-...
SLACK_ALLOWED_USERS=U01ABC123
```

**WhatsApp:**
```bash
WHATSAPP_ENABLED=true
WHATSAPP_ALLOWED_USERS=15551234567
```

**Signal:**
```bash
SIGNAL_HTTP_URL=http://signal-cli:8080
SIGNAL_ACCOUNT=+15551234567
```

### Security & Authorization
**Authorization Check Order:**
1. **Per-platform allow-all flag** (e.g., `DISCORD_ALLOW_ALL_USERS=true`)
2. **DM pairing approved list** (users approved via pairing codes)
3. **Platform-specific allowlists** (e.g., `TELEGRAM_ALLOWED_USERS=123456789`)
4. **Global allowlist** (`GATEWAY_ALLOWED_USERS=123456789`)
5. **Global allow-all** (`GATEWAY_ALLOW_ALL_USERS=true`)
6. **Default: deny**

**DM Pairing System:**
- Unknown users receive 8-character pairing code
- Bot owner approves via `hermes pairing approve <platform> <code>`
- Codes expire after 1 hour, rate-limited to 1 per 10 minutes per user

**Pairing CLI Commands:**
```bash
hermes pairing list                # List pending and approved users
hermes pairing approve telegram ABC12DEF  # Approve a pairing code
hermes pairing revoke telegram 123456789  # Revoke a user's access
hermes pairing clear-pending       # Clear all pending codes
```

### Gateway Configuration
```yaml
# Platform-specific configuration
telegram:
  reactions: false                 # Enable emoji reactions during processing
  ignored_threads: []              # Thread IDs where bot never responds

discord:
  require_mention: true            # Require @mention in server channels
  free_response_channels: []        # Channels where mention not required
  auto_thread: true                # Auto-thread long replies
  reactions: true                  # Enable emoji reactions
  ignored_channels: []             # Channel IDs where bot never responds
  no_thread_channels: []           # Channels where bot responds without threading
  reply_to_mode: first             # off | first | all
  allow_mention_everyone: false     # Allow @everyone/@here pings
  allow_mention_roles: false        # Allow @role mentions
  allow_mention_users: true         # Allow @user mentions
  allow_mention_replied_user: true  # Ping author when replying

slack:
  # Similar platform-specific options
```

### Webhook Configuration
**Quick Setup:**
```bash
WEBHOOK_ENABLED=true
WEBHOOK_PORT=8644
WEBHOOK_SECRET=your-global-secret
```

**Route Configuration (config.yaml):**
```yaml
platforms:
  webhook:
    enabled: true
    extra:
      port: 8644
      secret: "global-fallback-secret"
      routes:
        github-pr:
          events: ["pull_request"]
          secret: "github-webhook-secret"
          prompt: |
            Review this pull request:
            Repository: {repository.full_name}
            PR #{number}: {pull_request.title}
            Author: {pull_request.user.login}
            URL: {pull_request.html_url}
          skills: ["github-code-review"]
          deliver: "github_comment"
          deliver_extra:
            repo: "{repository.full_name}"
            pr_number: "{number}"
```

**Webhook Endpoint Format:**
```
http://your-server:8644/webhooks/<route-name>
```

**Delivery Options:**
- `log` - Logs response to gateway output (default)
- `github_comment` - Posts as PR/issue comment via `gh` CLI
- `telegram` - Routes to Telegram
- `discord` - Routes to Discord
- `slack` - Routes to Slack
- `signal` - Routes to Signal
- `sms` - Routes to SMS via Twilio
- `whatsapp` - Routes to WhatsApp
- `matrix` - Routes to Matrix
- `email` - Routes to Email

**Dynamic Subscriptions (CLI):**
```bash
hermes webhook subscribe github-issues \
  --events "issues" \
  --prompt "New issue #{issue.number}: {issue.title}" \
  --deliver telegram \
  --deliver-chat-id "-100123456789"

hermes webhook list                # List subscriptions
hermes webhook remove github-issues  # Remove subscription
hermes webhook test github-issues  # Test webhook
```

**Security:**
- **HMAC signature validation**: GitHub (`X-Hub-Signature-256`), GitLab (`X-Gitlab-Token`), Generic (`X-Webhook-Signature`)
- **Rate limiting**: 30 requests/minute per route (configurable)
- **Idempotency**: Delivery IDs cached for 1 hour to prevent duplicate runs
- **Body size limits**: Max 1 MB (configurable)
- **Secret required**: Every route must have a secret (or inherit global)

---

## API SERVER CONFIGURATION

### Overview
Hermes Agent exposes a fully-featured **OpenAI-compatible HTTP API server** running on **port 8642** (configurable). This allows any frontend that speaks the OpenAI format to connect and use Hermes as a backend with its full toolset.

### Quick Start
```bash
# 1. Enable in ~/.hermes/.env
API_SERVER_ENABLED=true
API_SERVER_KEY=change-me-local-dev

# 2. Start the gateway
hermes gateway

# Output: [API Server] API server listening on http://127.0.0.1:8642
```

### Core Endpoints

#### POST /v1/chat/completions (Standard OpenAI Format)
**Request:**
```bash
curl http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer change-me-local-dev" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hermes-agent",
    "messages": [
      {"role": "system", "content": "You are a Python expert."},
      {"role": "user", "content": "Write a fibonacci function"}
    ],
    "stream": false
  }'
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1710000000,
  "model": "hermes-agent",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "Here's a fibonacci function..."},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 50, "completion_tokens": 200, "total_tokens": 250}
}
```

#### POST /v1/responses (OpenAI Responses API Format)
**Request:**
```json
{
  "model": "hermes-agent",
  "input": "What files are in my project?",
  "instructions": "You are a helpful coding assistant.",
  "store": true
}
```

**Response:**
```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "completed",
  "model": "hermes-agent",
  "output": [
    {"type": "function_call", "name": "terminal", "arguments": "{\"command\": \"ls\"}", "call_id": "call_1"},
    {"type": "function_call_output", "call_id": "call_1", "output": "README.md src/ tests/"},
    {"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "Your project has..."}]}
  ],
  "usage": {"input_tokens": 50, "output_tokens": 200, "total_tokens": 250}
}
```

**Multi-turn with `previous_response_id`:**
```json
{
  "input": "Now show me the README",
  "previous_response_id": "resp_abc123"
}
```

**Named conversations (automatic chaining):**
```json
{"input": "Hello", "conversation": "my-project"}
{"input": "What's in src/?", "conversation": "my-project"}
{"input": "Run the tests", "conversation": "my-project"}
```

#### GET /v1/responses/{id}
Retrieve a previously stored response by ID.

#### DELETE /v1/responses/{id}
Delete a stored response.

#### GET /v1/models
Lists the agent as an available model.

**Response:**
```json
[
  {
    "id": "hermes-agent",
    "object": "model",
    "owned_by": "hermes"
  }
]
```

#### GET /health or GET /v1/health
Health check endpoint.

**Response:**
```json
{"status": "ok"}
```

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `API_SERVER_ENABLED` | `false` | Enable the API server |
| `API_SERVER_PORT` | `8642` | HTTP server port |
| `API_SERVER_HOST` | `127.0.0.1` | Bind address (localhost only by default) |
| `API_SERVER_KEY` | *(none)* | Bearer token for auth |
| `API_SERVER_CORS_ORIGINS` | *(none)* | Comma-separated allowed browser origins |
| `API_SERVER_MODEL_NAME` | *(profile name)* | Model name on `/v1/models` |

### Security Requirements
⚠️ **Critical**: The API server gives full access to Hermes' toolset, **including terminal commands**.

- **Default bind address** (`127.0.0.1`) is for local-only use
- When binding to non-loopback address like `0.0.0.0`, `API_SERVER_KEY` is **required**
- Keep `API_SERVER_CORS_ORIGINS` narrow to control browser access
- Browser access is disabled by default; enable only for explicit trusted origins

### Security Headers
All responses include:
- `X-Content-Type-Options: nosniff` — prevents MIME type sniffing
- `Referrer-Policy: no-referrer` — prevents referrer leakage

### CORS Configuration
For direct browser access, set an explicit allowlist:
```bash
API_SERVER_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

When CORS is enabled:
- **Preflight responses** include `Access-Control-Max-Age: 600` (10 minute cache)
- **SSE streaming responses** include CORS headers for browser EventSource clients
- **`Idempotency-Key`** is an allowed request header — clients can send it for deduplication (responses cached for 5 minutes)

### Streaming Responses
**Chat Completions Streaming** (`"stream": true`):
- Returns Server-Sent Events (SSE) with token-by-token response chunks
- Uses standard `chat.completion.chunk` events
- Custom `hermes.tool.progress` event for tool-start UX (shows what tool is running)

**Responses API Streaming**:
- Uses spec-native `response.created`, `response.output_text.delta`, `response.output_item.added`, `response.output_item.done`, `response.completed` events
- Emits `function_call` and `function_call_output` items during SSE stream for structured tool UI

### Compatible Frontends
| Frontend | Stars | Connection |
|----------|-------|------------|
| Open WebUI | 126k | Full guide available |
| LobeChat | 73k | Custom provider endpoint |
| LibreChat | 34k | Custom endpoint in librechat.yaml |
| AnythingLLM | 56k | Generic OpenAI provider |
| NextChat | 87k | BASE_URL env var |
| ChatBox | 39k | API Host setting |
| Jan | 26k | Remote model config |
| HF Chat-UI | 8k | OPENAI_BASE_URL |
| big-AGI | 7k | Custom endpoint |

### Python SDK Integration
```python
from openai import OpenAI

client = OpenAI(
    api_key="your-secret-key",
    base_url="http://localhost:8642/v1"
)

response = client.chat.completions.create(
    model="hermes-agent",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Write a Python function"}
    ]
)

print(response.choices[0].message.content)
```

### JavaScript/TypeScript SDK Integration
```typescript
import OpenAI from "openai";

const openai = new OpenAI({
  apiKey: "your-secret-key",
  baseURL: "http://localhost:8642/v1"
});

const response = await openai.chat.completions.create({
  model: "hermes-agent",
  messages: [
    { role: "system", content: "You are a helpful assistant." },
    { role: "user", content: "Write a TypeScript function" }
  ]
});

console.log(response.choices[0].message.content);
```

### Multi-User Setup with Profiles
```bash
# Create profiles
hermes profile create alice
hermes profile create bob

# Configure API servers on different ports
hermes -p alice config set API_SERVER_ENABLED true
hermes -p alice config set API_SERVER_PORT 8643
hermes -p alice config set API_SERVER_KEY alice-secret

hermes -p bob config set API_SERVER_ENABLED true
hermes -p bob config set API_SERVER_PORT 8644
hermes -p bob config set API_SERVER_KEY bob-secret

# Start each gateway
hermes -p alice gateway &
hermes -p bob gateway &
```

In Open WebUI, add each as a separate connection:
- Alice: `http://host.docker.internal:8643/v1` (key: `alice-secret`)
- Bob: `http://host.docker.internal:8644/v1` (key: `bob-secret`)

### Limitations & Considerations
- **Response storage**: Stored responses (for `previous_response_id`) are persisted in SQLite and survive gateway restarts. Max 100 stored responses (LRU eviction).
- **No file upload**: Vision/document analysis via uploaded files is not yet supported through the API.
- **Model field is cosmetic**: The `model` field in requests is accepted but the actual LLM model used is configured server-side in `config.yaml`.

---

## TERMINAL BACKENDS (6 OPTIONS)

### Backend Comparison
| Backend | Where | Isolation | Best for |
|---------|-------|-----------|----------|
| **local** | Your machine | None | Development |
| **docker** | Container | Full | Safe sandboxing |
| **ssh** | Remote server | Network | Remote dev |
| **singularity** | Container | Namespaces | HPC clusters |
| **modal** | Cloud VM | Full | Serverless compute |
| **daytona** | Cloud container | Full | Managed dev envs |

### Local Backend
```yaml
terminal:
  backend: local
```
- **Default**: Runs directly on your machine with no isolation
- **Requirements**: No special setup required
- **Warning**: Agent has same filesystem access as your user account

### Docker Backend
```yaml
terminal:
  backend: docker
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_mount_cwd_to_workspace: false
  docker_forward_env:
    - "GITHUB_TOKEN"
  docker_volumes:
    - "/home/user/projects:/workspace/projects"
    - "/home/user/data:/data:ro"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true
```
- **Requirements**: Docker Desktop or Docker Engine installed and running
- **Security hardening**:
  - `--cap-drop ALL` with only `DAC_OVERRIDE`, `CHOWN`, `FOWNER` added back
  - `--security-opt no-new-privileges`
  - `--pids-limit 256`
  - Size-limited tmpfs for `/tmp` (512MB), `/var/tmp` (256MB), `/run` (64MB)
- **Container lifecycle**: Each session starts a long-lived container (`docker run -d ... sleep 2h`). Commands run via `docker exec` with a login shell.

### SSH Backend
```yaml
terminal:
  backend: ssh
  persistent_shell: true
```
**Required environment variables:**
```bash
TERMINAL_SSH_HOST=my-server.example.com
TERMINAL_SSH_USER=ubuntu
```
**Optional:**
| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINAL_SSH_PORT` | `22` | SSH port |
| `TERMINAL_SSH_KEY` | (system default) | Path to SSH private key |
| `TERMINAL_SSH_PERSISTENT` | `true` | Enable persistent shell |

**How it works**: Connects at init time with `BatchMode=yes` and `StrictHostKeyChecking=accept-new`. Persistent shell keeps a single `bash -l` process alive on the remote host, communicating via temporary files.

### Modal Backend
```yaml
terminal:
  backend: modal
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true
```
**Required**: Either `MODAL_TOKEN_ID` + `MODAL_TOKEN_SECRET` environment variables, or a `~/.modal.toml` config file.

**Persistence**: When enabled, the sandbox filesystem is snapshotted on cleanup and restored on next session. Snapshots are tracked in `~/.hermes/modal_snapshots.json`.

### Daytona Backend
```yaml
terminal:
  backend: daytona
  container_cpu: 1
  container_memory: 5120
  container_disk: 10240  # Max 10 GiB
  container_persistent: true
```
**Required**: `DAYTONA_API_KEY` environment variable.

**Persistence**: When enabled, sandboxes are stopped (not deleted) on cleanup and resumed on next session. Sandbox names follow the pattern `hermes-{task_id}`.

### Singularity/Apptainer Backend
```yaml
terminal:
  backend: singularity
  singularity_image: "docker://nikolaik/python-nodejs:python3.11-nodejs20"
  container_cpu: 1
  container_memory: 5120
  container_persistent: true
```
**Requirements**: `apptainer` or `singularity` binary in `$PATH`.

**Image handling**: Docker URLs (`docker://...`) are automatically converted to SIF files and cached. Existing `.sif` files are used directly.

**Isolation**: Uses `--containall --no-home` for full namespace isolation without mounting the host home directory.

### Common Terminal Backend Issues
If terminal commands fail immediately or the terminal tool is reported as disabled:

- **Local** — No special requirements. The safest default when getting started.
- **Docker** — Run `docker version` to verify Docker is working. If it fails, fix Docker or `hermes config set terminal.backend local`.
- **SSH** — Both `TERMINAL_SSH_HOST` and `TERMINAL_SSH_USER` must be set. Hermes logs a clear error if either is missing.
- **Modal** — Needs `MODAL_TOKEN_ID` env var or `~/.modal.toml`. Run `hermes doctor` to check.
- **Daytona** — Needs `DAYTONA_API_KEY`. The Daytona SDK handles server URL configuration.
- **Singularity** — Needs `apptainer` or `singularity` in `$PATH`. Common on HPC clusters.

### Docker Volume Mounts
When using the Docker backend, `docker_volumes` lets you share host directories with the container:
```yaml
terminal:
  backend: docker
  docker_volumes:
    - "/home/user/projects:/workspace/projects"   # Read-write (default)
    - "/home/user/datasets:/data:ro"              # Read-only
    - "/home/user/outputs:/outputs"               # Agent writes, you read
```

### Docker Credential Forwarding
By default, Docker terminal sessions do not inherit arbitrary host credentials. If you need a specific token inside the container, add it to `terminal.docker_forward_env`:
```yaml
terminal:
  backend: docker
  docker_forward_env:
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
```

### Persistent Shell
By default, each terminal command runs in its own subprocess — working directory, environment variables, and shell variables reset between commands. When **persistent shell** is enabled, a single long-lived bash process is kept alive across `execute()` calls so that state survives between commands.

This is most useful for the **SSH backend**, where it also eliminates per-command connection overhead. Persistent shell is **enabled by default for SSH** and disabled for the local backend.

---

## BROWSER AUTOMATION (5 BACKENDS)

### Browser Backends
1. **Browserbase** (cloud) — Managed browsers, CAPTCHA solving, residential proxies
2. **Browser Use** (cloud) — Alternative cloud provider
3. **Firecrawl** (cloud) — Cloud browsers + built-in scraping
4. **Camofox** (local) — Firefox-based anti-detection
5. **Local Chrome via CDP** — Your own Chrome instance

### Setup
**Cloud (Browserbase, Browser Use, Firecrawl):**
```bash
export BROWSERBASE_API_KEY=...
export BROWSER_USE_API_KEY=...
export FIRECRAWL_API_KEY=...
```

**Local Chrome:**
```bash
/browser connect              # Attach to running Chrome
/browser disconnect           # Detach
/browser status               # Show connection
```

### Workflow Example
```
1. browser_navigate("https://example.com/signup")
2. browser_snapshot()  → sees form fields with refs
3. browser_type(ref="@e3", text="john@example.com")
4. browser_type(ref="@e5", text="password")
5. browser_click(ref="@e7")  # Submit button
6. browser_snapshot()  → confirm success
```

### Web Tools Configuration
```yaml
web:
  backend: firecrawl    # firecrawl | parallel | tavily | exa
```

### External Browser Services (No Playwright Needed!)
| Service | Free Tier | Setup |
|---------|-----------|-------|
| **Jina.ai Reader** | **Unlimited, no key!** | `curl https://r.jina.ai/{url}` |
| **Firecrawl** | 500 credits/month | `FIRECRAWL_API_KEY=...` |
| **Exa** | 1000 searches | `EXA_API_KEY=...` |
| **Tavily** | 1000 searches | `TAVILY_API_KEY=...` |

---

## LLM PROVIDERS (23+ PROVIDERS)

### Provider Support Table
| Provider | API Key Var | OAuth | Custom Endpoint | Notes |
|----------|-------------|-------|-----------------|-------|
| **OpenRouter** | `OPENROUTER_API_KEY` | ❌ | ✅ | Recommended for flexibility |
| **Anthropic** | `ANTHROPIC_API_KEY` | ✅ | ✅ | Claude Pro/Max via OAuth |
| **OpenAI** | `OPENAI_API_KEY` | ❌ | ✅ | GPT-4, GPT-4o, etc. |
| **Nous** | Built-in | ✅ | ✅ | Paid Nous Portal subscribers |
| **Codex** | Built-in | ✅ | ❌ | ChatGPT Pro/Plus account |
| **Copilot** | `COPILOT_GITHUB_TOKEN` | ✅ | ✅ | GitHub Copilot API |
| **Gemini** | `GOOGLE_API_KEY` | ✅ | ✅ | Google AI Studio |
| **ZhipuAI (GLM)** | `GLM_API_KEY` | ❌ | ✅ | Chinese LLM provider |
| **Kimi/Moonshot** | `KIMI_API_KEY` | ❌ | ✅ | Chinese LLM provider |
| **MiniMax** | `MINIMAX_API_KEY` | ❌ | ✅ | Chinese LLM provider |
| **Arcee** | `ARCEEAI_API_KEY` | ❌ | ✅ | Specialized AI models |
| **Hugging Face** | `HF_TOKEN` | ❌ | ✅ | Inference Providers |
| **Alibaba (Qwen)** | `DASHSCOPE_API_KEY` | ❌ | ✅ | Chinese LLM provider |
| **DeepSeek** | `DEEPSEEK_API_KEY` | ❌ | ✅ | Chinese LLM provider |
| **OpenCode Zen** | `OPENCODE_ZEN_API_KEY` | ❌ | ✅ | Pay-as-you-go curated models |
| **OpenCode Go** | `OPENCODE_GO_API_KEY` | ❌ | ✅ | $10/month subscription |
| **Vercel AI Gateway** | `AI_GATEWAY_API_KEY` | ❌ | ✅ | Vercel's AI routing |
| **KiLo Code** | `KILOCODE_API_KEY` | ❌ | ✅ | Code-focused models |
| **Xiaomi MiMo** | `XIAOMI_API_KEY` | ❌ | ✅ | Xiaomi's LLM |
| **Groq** | `GROQ_API_KEY` | ❌ | ✅ | Fast inference |
| **Fal.ai** | `FAL_KEY` | ❌ | ✅ | Image generation |
| **ElevenLabs** | `ELEVENLABS_API_KEY` | ❌ | ✅ | Premium TTS voices |
| **Weights & Biases** | `WANDB_API_KEY` | ❌ | ✅ | ML experiment tracking |

### Provider Configuration
```yaml
model:
  provider: openrouter        # or: anthropic, openai, nous, custom, etc.
  model: openrouter/openai/gpt-4o-mini
  base_url: https://api.openai.com/v1  # Custom endpoint
```

### Fallback Model Configuration
```yaml
fallback_model:
  provider: openrouter
  model: anthropic/claude-sonnet-4
```

### Auxiliary Model Configuration
```yaml
auxiliary:
  vision:
    provider: "auto"
    model: "openai/gpt-4o"
  compression:
    provider: "nous"
    model: "gemini-3-flash"
  web_extract:
    provider: "openrouter"
    model: "google/gemini-2.5-flash"
```

### Credential Pool Strategies
```yaml
credential_pool_strategies:
  openrouter: round_robin    # cycle through keys evenly
  anthropic: least_used      # always pick the least-used key
```
Options: `fill_first` (default), `round_robin`, `least_used`, `random`

### Reasoning Effort Settings
```yaml
agent:
  reasoning_effort: ""   # empty = medium (default). Options: none, minimal, low, medium, high, xhigh (max)
```

### Tool-Use Enforcement
```yaml
agent:
  tool_use_enforcement: "auto"   # "auto" | true | false | ["model-substring", ...]
```
- `"auto"` (default): Enabled for models matching: `gpt`, `codex`, `gemini`, `gemma`, `grok`. Disabled for Claude, DeepSeek, Qwen, etc.
- `true`: Always enabled, regardless of model
- `false`: Always disabled, regardless of model
- `["gpt", "codex", "qwen", "llama"]`: Enabled only when model name contains listed substrings

---

## SECURITY FEATURES

### Dangerous Command Approval
Before executing any command, Hermes checks it against a curated list of dangerous patterns. If a match is found, the user must explicitly approve it.

**Approval Modes:**
```yaml
approvals:
  mode: manual    # manual | smart | off
  timeout: 60     # seconds to wait for user response (default: 60)
```

| Mode | Behavior |
|------|----------|
| **manual** (default) | Always prompt the user for approval on dangerous commands |
| **smart** | Use an auxiliary LLM to assess risk. Low-risk commands auto-approved, genuinely dangerous commands auto-denied, uncertain cases escalate to manual prompt |
| **off** | Disable all approval checks — equivalent to running with `--yolo` |

**YOLO Mode** (bypasses all safety prompts):
```bash
hermes --yolo                    # CLI flag
hermes chat --yolo              # Single query
/yolo                           # Slash command (toggle)
HERMES_YOLO_MODE=1              # Environment variable
```

**What Triggers Approval:**
- `rm -r` / `rm --recursive` - Recursive delete
- `chmod 777/666` / `o+w` / `a+w` - World-writable permissions
- `chown -R root` - Recursive chown to root
- `mkfs` - Format filesystem
- `dd if=` - Disk copy
- `> /dev/sd` - Write to block device
- SQL DROP/DELETE/TRUNCATE commands
- `> /etc/` - Overwrite system config
- `systemctl stop/disable/mask` - Stop/disable system services
- `kill -9 -1` - Kill all processes
- `pkill -9` - Force kill processes
- Fork bomb patterns
- `bash -c` / `sh -c` / `zsh -c` / `ksh -c` - Shell command execution
- `python -e` / `perl -e` / `ruby -e` / `node -c` - Script execution
- `curl ... \| sh` / `wget ... \| sh` - Pipe remote content to shell
- `tee` to `/etc/`, `~/.ssh/`, `~/.hermes/.env` - Overwrite sensitive file
- `xargs rm` - xargs with rm
- `find -exec rm` / `find -delete` - Find with destructive actions

**Container bypass**: When running in `docker`, `singularity`, `modal`, or `daytona` backends, dangerous command checks are **skipped** because the container itself is the security boundary.

**Approval Flow (CLI):**
```
  ⚠️  DANGEROUS COMMAND: recursive delete
      rm -rf /tmp/old-project

      [o]nce  |  [s]ession  |  [a]lways  |  [d]eny

      Choice [o/s/a/D]:
```

**Approval Flow (Gateway/Messaging):**
- Reply **yes**, **y**, **approve**, **ok**, or **go** to approve
- Reply **no**, **n**, **deny**, or **cancel** to deny

**Permanent Allowlist:**
Commands approved with "always" are saved to `~/.hermes/config.yaml`:
```yaml
command_allowlist:
  - rm
  - systemctl
```

### Container Isolation
When using the `docker` terminal backend, Hermes applies strict security hardening:

**Docker Security Flags:**
```python
_SECURITY_ARGS = [
    "--cap-drop", "ALL",                          # Drop ALL Linux capabilities
    "--cap-add", "DAC_OVERRIDE",                  # Root can write to bind-mounted dirs
    "--cap-add", "CHOWN",                         # Package managers need file ownership
    "--cap-add", "FOWNER",                        # Package managers need file ownership
    "--security-opt", "no-new-privileges",         # Block privilege escalation
    "--pids-limit", "256",                         # Limit process count
    "--tmpfs", "/tmp:rw,nosuid,size=512m",         # Size-limited /tmp
    "--tmpfs", "/var/tmp:rw,noexec,nosuid,size=256m",  # No-exec /var/tmp
    "--tmpfs", "/run:rw,noexec,nosuid,size=64m",   # No-exec /run
]
```

**Resource Limits:**
```yaml
terminal:
  backend: docker
  container_cpu: 1        # CPU cores
  container_memory: 5120  # MB (default 5GB)
  container_disk: 51200   # MB (default 50GB, requires overlay2 on XFS)
  container_persistent: true  # Persist filesystem across sessions
```

### Environment Variable Passthrough
Both `execute_code` and `terminal` strip sensitive environment variables from child processes to prevent credential exfiltration by LLM-generated code. However, skills that declare `required_environment_variables` legitimately need access to those vars.

**Skill-scoped passthrough (automatic):**
When a skill is loaded and declares `required_environment_variables`, any of those vars that are actually set in the environment are automatically registered as passthrough.

**Config-based passthrough (manual):**
```yaml
terminal:
  env_passthrough:
    - MY_CUSTOM_KEY
    - ANOTHER_TOKEN
```

### MCP Credential Handling
MCP (Model Context Protocol) server subprocesses receive a **filtered environment** to prevent accidental credential leakage.

**Safe Environment Variables:**
Only these variables are passed through from the host to MCP stdio subprocesses:
```
PATH, HOME, USER, LANG, LC_ALL, TERM, SHELL, TMPDIR
```
Plus any `XDG_*` variables. All other environment variables (API keys, tokens, secrets) are **stripped**.

**Credential Redaction:**
Error messages from MCP tools are sanitized before being returned to the LLM. The following patterns are replaced with `[REDACTED]`:
- GitHub PATs (`ghp_...`)
- OpenAI-style keys (`sk-...`)
- Bearer tokens
- `token=`, `key=`, `API_KEY=`, `password=`, `secret=` parameters

### SSRF Protection
All URL-capable tools (web search, web extract, vision, browser) validate URLs before fetching them to prevent Server-Side Request Forgery (SSRF) attacks. Blocked addresses include:

- **Private networks** (RFC 1918): `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- **Loopback**: `127.0.0.0/8`, `::1`
- **Link-local**: `169.254.0.0/16` (includes cloud metadata at `169.254.169.254`)
- **CGNAT / shared address space** (RFC 6598): `100.64.0.0/10` (Tailscale, WireGuard VPNs)
- **Cloud metadata hostnames**: `metadata.google.internal`, `metadata.goog`
- **Reserved, multicast, and unspecified addresses**

SSRF protection is always active and cannot be disabled. DNS failures are treated as blocked (fail-closed). Redirect chains are re-validated at each hop to prevent redirect-based bypasses.

### Tirith Pre-Exec Security Scanning
Hermes integrates [tirith](https://github.com/sheeki03/tirith) for content-level command scanning before execution. Tirith detects threats that pattern matching alone misses:

- Homograph URL spoofing (internationalized domain attacks)
- Pipe-to-interpreter patterns (`curl | bash`, `wget | sh`)
- Terminal injection attacks

Tirith auto-installs from GitHub releases on first use with SHA-256 checksum verification (and cosign provenance verification if cosign is available).

```yaml
security:
  tirith_enabled: true       # Enable/disable tirith scanning (default: true)
  tirith_path: "tirith"      # Path to tirith binary (default: PATH lookup)
  tirith_timeout: 5          # Subprocess timeout in seconds
  tirith_fail_open: true     # Allow execution when tirith is unavailable (default: true)
```

### Context File Injection Protection
Context files (AGENTS.md, .cursorrules, SOUL.md) are scanned for prompt injection before being included in the system prompt. The scanner checks for:

- Instructions to ignore/disregard prior instructions
- Hidden HTML comments with suspicious keywords
- Attempts to read secrets (`.env`, `credentials`, `.netrc`)
- Credential exfiltration via `curl`
- Invisible Unicode characters (zero-width spaces, bidirectional overrides)

Blocked files show a warning:
```
[BLOCKED: AGENTS.md contained potential prompt injection (prompt_injection). Content not loaded.]
```

### Website Blocklist
```yaml
security:
  website_blocklist:
    enabled: true
    domains:
      - "*.internal.company.com"
      - "admin.example.com"
    shared_files:
      - "/etc/hermes/blocked-sites.txt"
```

When a blocked URL is requested, the tool returns an error explaining the domain is blocked by policy.

### Privacy Settings
```yaml
privacy:
  redact_pii: false  # Strip PII from LLM context (gateway only)
```

When `redact_pii` is `true`, the gateway redacts personally identifiable information from the system prompt before sending it to the LLM on supported platforms:

| Field | Treatment |
|-------|-----------|
| Phone numbers (user ID on WhatsApp/Signal) | Hashed to `user_<12-char-sha256>` |
| User IDs | Hashed to `user_<12-char-sha256>` |
| Chat IDs | Numeric portion hashed, platform prefix preserved (`telegram:<hash>`) |
| Home channel IDs | Numeric portion hashed |
| User names / usernames | **Not affected** (user-chosen, publicly visible) |

**Platform support**: Redaction applies to WhatsApp, Signal, and Telegram. Discord and Slack are excluded because their mention systems (`<@user_id>`) require the real ID in the LLM context.

---

## VOICE & AUDIO CAPABILITIES

### Voice Modes
| Feature | Platform | Description |
|---------|----------|-------------|
| **Interactive Voice** | CLI | Press Ctrl+B to record, auto-detects silence |
| **Auto Voice Reply** | Telegram, Discord | Agent sends spoken audio alongside text |
| **Voice Channel** | Discord | Bot joins VC, listens, speaks replies |

### Speech-to-Text (STT) Providers
| Provider | Speed | Quality | Cost | API Key |
|----------|-------|---------|------|---------|
| **Local (faster-whisper)** | Fast | Good | Free | No |
| **Groq** | Very fast (~0.5s) | Good | Free tier | Yes |
| **OpenAI Whisper** | Fast (~1s) | Good | Paid | Yes |

### Text-to-Speech (TTS) Providers
| Provider | Quality | Cost | Latency | Key |
|----------|---------|------|---------|-----|
| **Edge TTS** | Good | Free | ~1s | No |
| **ElevenLabs** | Excellent | Paid | ~2s | Yes |
| **OpenAI TTS** | Good | Paid | ~1.5s | Yes |
| **NeuTTS** | Good | Free | CPU-dependent | No |

### CLI Voice Mode
```bash
hermes                  # Start CLI
/voice on              # Enable voice mode
# Press Ctrl+B to record
# Auto-stops after 3s silence
# TTS replies automatically
```

### Discord Voice Channels
```bash
/voice join            # Bot joins your VC
/voice leave           # Bot leaves
/voice status          # Show status
```

**Setup requirements:**
- Discord bot permissions: Connect, Speak, Use Voice Activity
- Privileged Gateway Intents: Presence, Server Members, Message Content
- Opus codec installed (`brew install opus`)

### Configuration
```yaml
# STT Configuration
stt:
  provider: "local"           # or: groq, openai
  local:
    model: "base"             # tiny, base, small, medium, large-v3

# TTS Configuration
tts:
  provider: "edge"            # or: elevenlabs, openai, neutts
  speed: 1.0                  # Global speed multiplier
  edge:
    voice: "en-US-AriaNeural"
    speed: 1.0
  elevenlabs:
    voice_id: "pNInz6obpgDQGcFmaJgB"
    model_id: "eleven_multilingual_v2"
```

---

## MEMORY & PERSISTENCE

### Memory Configuration
```yaml
memory:
  memory_enabled: true
  user_profile_enabled: true
  memory_char_limit: 2200   # ~800 tokens
  user_char_limit: 1375     # ~500 tokens
```

### Memory Providers
Hermes supports multiple memory backends:

| Provider | Description | Environment Variable |
|----------|-------------|---------------------|
| **Honcho** | Cross-session user modeling | `HONCHO_API_KEY` |
| **Supermemory** | Semantic long-term memory | `SUPERMEMORY_API_KEY` |
| **Local** | File-based memory (default) | None |

### Memory Files
- `~/.hermes/memories/MEMORY.md` - Cross-session memory
- `~/.hermes/memories/USER.md` - User profile memory

### Session Search
The `session_search` tool allows searching past conversations with LLM summarization:
```bash
/session_search "kubernetes deployment"
```

### Skills Memory
Skills created by the agent are stored in `~/.hermes/skills/` and become part of procedural memory.

### Cron Job Persistence
Scheduled tasks are stored in `~/.hermes/cron/` and persist across sessions.

### Session Persistence
Chat history is stored in `~/.hermes/sessions/` with automatic compression and cleanup.

---

## PROFILES & MULTI-USER SETUP

### Creating Profiles
```bash
hermes profile create alice
hermes profile create bob
hermes profile list
hermes profile delete charlie
```

### Using Profiles
```bash
hermes -p alice                # Use alice profile
hermes -p bob chat -q "Hello"  # Single query with bob profile
hermes profile switch alice    # Switch default profile
```

### Profile Directory Structure
Each profile gets its own directory:
```
~/.hermes-profiles/
├── alice/
│   ├── config.yaml
│   ├── .env
│   ├── SOUL.md
│   └── ...
└── bob/
    ├── config.yaml
    ├── .env
    ├── SOUL.md
    └── ...
```

### Multi-User API Server Setup
```bash
# Configure each profile's API server on different ports
hermes -p alice config set API_SERVER_ENABLED true
hermes -p alice config set API_SERVER_PORT 8643
hermes -p alice config set API_SERVER_KEY alice-secret

hermes -p bob config set API_SERVER_ENABLED true
hermes -p bob config set API_SERVER_PORT 8644
hermes -p bob config set API_SERVER_KEY bob-secret

# Start each profile's gateway
hermes -p alice gateway &
hermes -p bob gateway &
```

Each profile's API server automatically advertises the profile name as the model ID:
- `http://localhost:8643/v1/models` → model `alice`
- `http://localhost:8644/v1/models` → model `bob`

### Profile-Specific Configurations
Each profile can have completely different configurations:
- Different LLM providers and models
- Different terminal backends
- Different messaging platform setups
- Different skills and personalities
- Different memory and persistence settings

### Switching Between Profiles
```bash
hermes profile switch alice    # Make alice the default
hermes profile current         # Show current profile
hermes profile reset          # Reset current profile to defaults
```

---

## QUICK REFERENCE COMMANDS

### Configuration Management
```bash
hermes config                   # View current configuration
hermes config edit              # Open config.yaml in editor
hermes config set KEY VAL       # Set specific value
hermes config check             # Check for missing options
hermes config migrate           # Add missing options
hermes config show              # Show all settings including skill settings
```

### Skills Management
```bash
hermes skills browse            # Browse skills hub
hermes skills search kubernetes # Search skills
hermes skills install official/security/1password  # Install skill
hermes skills list              # List installed skills
hermes skills check             # Check for updates
hermes skills update            # Update skills
hermes skills audit             # Security scan
hermes skills uninstall k8s     # Remove skill
```

### Messaging & Gateway
```bash
hermes gateway setup            # Interactive setup wizard
hermes gateway                  # Run gateway in foreground
hermes gateway install          # Install as service
hermes gateway start/stop       # Manage service
hermes pairing list             # List DM pairing requests
hermes pairing approve telegram ABC12DEF  # Approve pairing
hermes webhook list             # List webhook subscriptions
hermes webhook subscribe github-pr --events "pull_request"  # Create webhook
```

### Profiles
```bash
hermes profile create alice     # Create profile
hermes profile list             # List profiles
hermes profile switch alice     # Switch default profile
hermes -p bob chat -q "Hello"   # Use specific profile for single query
```

### Tools & Toolsets
```bash
hermes tools                    # Interactive tool management UI
/tools list                     # List available tools in session
/tools disable browser          # Disable specific tool
/tools enable rl                # Enable specific tool
```

### Session Management
```bash
hermes --continue               # Resume most recent session (-c)
hermes --resume <session_id>    # Resume specific session (-r)
/session_search "topic"         # Search past conversations
/todo list                      # Show current task list
```

### Voice & Audio
```bash
/voice on                       # Enable CLI voice mode
/voice join                     # Join Discord voice channel
/tts on                         # Enable spoken replies
```

---

## BEST PRACTICES FOR PRODUCTION DEPLOYMENT

### Gateway Deployment Checklist
1. **Set explicit allowlists** — never use `GATEWAY_ALLOW_ALL_USERS=true` in production
2. **Use container backend** — set `terminal.backend: docker` in config.yaml
3. **Restrict resource limits** — set appropriate CPU, memory, and disk limits
4. **Store secrets securely** — keep API keys in `~/.hermes/.env` with proper file permissions
5. **Enable DM pairing** — use pairing codes instead of hardcoding user IDs when possible
6. **Review command allowlist** — periodically audit `command_allowlist` in config.yaml
7. **Set `MESSAGING_CWD`** — don't let the agent operate from sensitive directories
8. **Run as non-root** — never run the gateway as root
9. **Monitor logs** — check `~/.hermes/logs/` for unauthorized access attempts
10. **Keep updated** — run `hermes update` regularly for security patches

### Securing API Keys
```bash
# Set proper permissions on the .env file
chmod 600 ~/.hermes/.env

# Keep separate keys for different services
# Never commit .env files to version control
```

### Network Isolation
For maximum security, run the gateway on a separate machine or VM:
```yaml
terminal:
  backend: ssh
  ssh_host: "agent-worker.local"
  ssh_user: "hermes"
  ssh_key: "~/.ssh/hermes_agent_key"
```

This keeps the gateway's messaging connections separate from the agent's command execution.

### Container Security
When using Docker backend, ensure proper security:
- Use minimal base images
- Run as non-root user inside container
- Limit resource usage
- Use read-only root filesystem where possible
- Regularly update container images

### Monitoring & Logging
- Monitor `~/.hermes/logs/gateway.log` for unusual activity
- Set up alerts for failed authentication attempts
- Log all command executions for audit purposes
- Monitor resource usage to detect abuse

### Backup & Recovery
- Regularly backup `~/.hermes/` directory
- Export important skills and configurations
- Document custom setups for recovery
- Test restore procedures periodically

---

## TROUBLESHOOTING COMMON ISSUES

### Terminal Commands Fail
**Symptoms**: Terminal tool disabled, commands fail immediately
**Solutions**:
- **Local**: No special requirements. Safest default.
- **Docker**: Run `docker version` to verify Docker is working. Fix Docker or switch to local.
- **SSH**: Ensure both `TERMINAL_SSH_HOST` and `TERMINAL_SSH_USER` are set.
- **Modal**: Verify `MODAL_TOKEN_ID` or `~/.modal.toml` exists. Run `hermes doctor`.
- **Daytona**: Ensure `DAYTONA_API_KEY` is set.
- **Singularity**: Ensure `apptainer` or `singularity` is in `$PATH`.

### Messaging Platform Not Responding
**Symptoms**: Bot doesn't respond to messages
**Solutions**:
- Check allowlists are configured (`TELEGRAM_ALLOWED_USERS`, etc.)
- Verify bot tokens are correct
- Check gateway logs for errors
- Ensure platform-specific requirements are met (Discord intents, etc.)
- Test with DM pairing if allowlists are empty

### API Server Connection Issues
**Symptoms**: Cannot connect to API server
**Solutions**:
- Verify `API_SERVER_ENABLED=true`
- Check `API_SERVER_HOST` binding (127.0.0.1 vs 0.0.0.0)
- Ensure `API_SERVER_KEY` is set for non-localhost binding
- Test with `curl http://localhost:8642/health`
- Check firewall rules for port 8642

### Skills Not Loading
**Symptoms**: Skills not available as slash commands
**Solutions**:
- Verify skill directory structure (`skills/category/name/SKILL.md`)
- Check skill file format (proper YAML frontmatter)
- Ensure skill name matches directory name
- Run `hermes skills list` to see installed skills
- Check for syntax errors in SKILL.md

### Browser Automation Failures
**Symptoms**: Browser tools fail or timeout
**Solutions**:
- Verify browser backend is configured (`BROWSERBASE_API_KEY`, etc.)
- Check internet connectivity
- Increase `browser.command_timeout` if needed
- Test with simple navigation first
- Check for CAPTCHA or bot protection on target sites

### Memory Not Persisting
**Symptoms**: Agent doesn't remember previous conversations
**Solutions**:
- Verify `memory.memory_enabled: true`
- Check file permissions on `~/.hermes/memories/`
- Ensure sufficient disk space
- Test with simple memory save: `/memory save "test"`

### Voice Recognition Issues
**Symptoms**: STT not working or poor quality
**Solutions**:
- Verify STT provider is configured (`stt.provider: local`)
- For local STT, ensure `faster-whisper` is installed
- Check microphone permissions (CLI)
- Test with different STT providers (Groq, OpenAI)
- Adjust `voice.silence_threshold` and `voice.silence_duration`

---

## RESOURCES & DOCUMENTATION

### Official Documentation
- **Main Docs**: https://hermes-agent.nousresearch.com/docs
- **Configuration**: https://hermes-agent.nousresearch.com/docs/user-guide/configuration
- **Tools Reference**: https://hermes-agent.nousresearch.com/docs/reference/tools-reference
- **Toolsets Reference**: https://hermes-agent.nousresearch.com/docs/reference/toolsets-reference
- **Skills Catalog**: https://hermes-agent.nousresearch.com/docs/reference/skills-catalog
- **Optional Skills**: https://hermes-agent.nousresearch.com/docs/reference/optional-skills-catalog
- **API Server**: https://hermes-agent.nousresearch.com/docs/user-guide/features/api-server
- **Webhooks**: https://hermes-agent.nousresearch.com/docs/user-guide/messaging/webhooks
- **Open WebUI Integration**: https://hermes-agent.nousresearch.com/docs/user-guide/messaging/open-webui

### GitHub Repositories
- **Hermes Agent**: https://github.com/NousResearch/hermes-agent
- **Hermes Agent Helm Chart**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Skills Hub**: https://skills.sh

### Community Resources
- **Discord Community**: https://discord.gg/NousResearch
- **Agentskills Standard**: https://agentskills.io

---

## VERSION INFORMATION

**Current Version**: Hermes Agent v0.8.0 (April 2026)
**Config Schema Version**: 10
**Documentation Last Updated**: April 18, 2026

This comprehensive reference document covers all configuration options, capabilities, and features of Hermes Agent as of April 2026. For the most up-to-date information, always refer to the official documentation and GitHub repositories.