# Hermes Agent Configuration Reference

Complete reference for `config.yaml` structure and all available settings.

## Configuration File Location

```
~/.hermes/config.yaml
```

## Complete config.yaml Structure

```yaml
# ============================================================================
# HERMES AGENT CONFIGURATION
# ============================================================================

# Model Configuration
model:
  # Default model to use
  default: anthropic/claude-opus-4.6
  
  # Base URL for API (OpenRouter, OpenAI, Anthropic, etc.)
  base_url: https://openrouter.ai/api/v1
  
  # Temperature (0.0 - 2.0)
  # Lower = more deterministic, Higher = more creative
  temperature: 0.7
  
  # Max tokens per response
  max_tokens: 4096
  
  # Top-p sampling (0.0 - 1.0)
  top_p: 0.9
  
  # Top-k sampling
  top_k: 40

# Terminal Configuration
terminal:
  # Backend: local, docker, ssh, modal, daytona, singularity
  backend: local
  
  # Docker configuration (if backend: docker)
  docker:
    image: ubuntu:22.04
    container_name: hermes-terminal
    volumes:
      - /tmp:/tmp
    environment:
      - PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
  
  # SSH configuration (if backend: ssh)
  ssh:
    host: localhost
    port: 22
    username: user
    key_path: ~/.ssh/id_rsa
  
  # Modal configuration (if backend: modal)
  modal:
    token_path: ~/.modal/token
    workspace: default
  
  # Daytona configuration (if backend: daytona)
  daytona:
    api_url: http://localhost:3000
    api_key: your-api-key
  
  # Singularity configuration (if backend: singularity)
  singularity:
    image_path: /path/to/image.sif
    bind_paths:
      - /tmp:/tmp

# Memory Configuration
memory:
  # Enable memory system
  enabled: true
  
  # Memory type: short_term, long_term, hybrid
  type: hybrid
  
  # Max memory entries
  max_entries: 1000
  
  # Memory compression (reduces token usage)
  compression:
    enabled: true
    threshold: 500  # Compress when > 500 entries
    ratio: 0.5      # Keep 50% of entries

# Gateway Configuration (Messaging Platforms)
gateways:
  # Telegram
  telegram:
    enabled: false
    token: "YOUR_TELEGRAM_BOT_TOKEN"
    allowed_users:
      - 123456789
    dm_pairing: true
    dm_pairing_timeout: 300
  
  # Discord
  discord:
    enabled: false
    token: "YOUR_DISCORD_BOT_TOKEN"
    allowed_servers:
      - 123456789
    allowed_channels:
      - 987654321
  
  # Slack
  slack:
    enabled: false
    bot_token: "xoxb-YOUR_TOKEN"
    app_token: "xapp-YOUR_TOKEN"
    allowed_workspaces:
      - workspace-id
  
  # WhatsApp
  whatsapp:
    enabled: false
    phone_number: "+1234567890"
    api_key: "YOUR_API_KEY"
    webhook_url: "https://your-domain.com/webhook"
  
  # Signal
  signal:
    enabled: false
    phone_number: "+1234567890"
    api_url: "http://localhost:8080"
  
  # Email
  email:
    enabled: false
    smtp_server: "smtp.gmail.com"
    smtp_port: 587
    email: "your-email@gmail.com"
    password: "YOUR_APP_PASSWORD"
    allowed_senders:
      - "trusted@example.com"
  
  # Matrix
  matrix:
    enabled: false
    homeserver: "https://matrix.org"
    user_id: "@hermes:matrix.org"
    access_token: "YOUR_ACCESS_TOKEN"
    allowed_rooms:
      - "!roomid:matrix.org"
  
  # Mattermost
  mattermost:
    enabled: false
    server_url: "https://mattermost.example.com"
    bot_token: "YOUR_BOT_TOKEN"
    allowed_teams:
      - team-id
  
  # Feishu/Lark
  feishu:
    enabled: false
    app_id: "YOUR_APP_ID"
    app_secret: "YOUR_APP_SECRET"
    webhook_url: "https://your-domain.com/webhook"
  
  # WeCom
  wecom:
    enabled: false
    corp_id: "YOUR_CORP_ID"
    agent_id: "YOUR_AGENT_ID"
    secret: "YOUR_SECRET"
  
  # Weixin (WeChat)
  weixin:
    enabled: false
    app_id: "YOUR_APP_ID"
    app_secret: "YOUR_APP_SECRET"
    token: "YOUR_TOKEN"
  
  # DingTalk
  dingtalk:
    enabled: false
    app_id: "YOUR_APP_ID"
    app_secret: "YOUR_APP_SECRET"
    webhook_url: "https://your-domain.com/webhook"
  
  # BlueBubbles
  bluebubbles:
    enabled: false
    server_url: "http://localhost:1234"
    password: "YOUR_PASSWORD"
  
  # QQ
  qq:
    enabled: false
    qq_number: "123456789"
    api_url: "http://localhost:5700"

# Approval Configuration
approvals:
  # Require approval for dangerous operations
  enabled: true
  
  # Approval mode: manual, auto, hybrid
  mode: manual
  
  # Dangerous operations requiring approval
  dangerous_operations:
    - file_delete
    - system_reboot
    - network_change
    - database_drop
  
  # Approval timeout (seconds)
  timeout: 300
  
  # Approval channels (where to send approval requests)
  channels:
    - telegram
    - email

# Tools Configuration
tools:
  # Enable/disable tool categories
  enabled_categories:
    - web
    - code
    - system
    - data
    - communication
    - productivity
    - ai
    - custom
  
  # Specific tools to disable
  disabled_tools:
    - system_reboot
    - network_change
  
  # Tool timeout (seconds)
  timeout: 30
  
  # Max concurrent tools
  max_concurrent: 5

# Toolsets Configuration
toolsets:
  # Define custom toolsets
  web_scraper:
    tools:
      - web_fetch
      - web_parse
      - web_screenshot
    description: "Web scraping and analysis"
  
  code_analyzer:
    tools:
      - code_search
      - code_analyze
      - code_format
    description: "Code analysis and formatting"

# API Server Configuration
api_server:
  # Enable API server
  enabled: true
  
  # Port
  port: 8642
  
  # Host
  host: 0.0.0.0
  
  # API key (set in .env as API_SERVER_KEY)
  # key: set-in-env
  
  # CORS settings
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:8000"
  
  # Rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60
  
  # OpenAI-compatible endpoints
  openai_compatible: true

# Logging Configuration
logging:
  # Log level: DEBUG, INFO, WARNING, ERROR, CRITICAL
  level: INFO
  
  # Log format: json, text
  format: json
  
  # Log file
  file: ~/.hermes/logs/hermes.log
  
  # Max log file size (MB)
  max_size: 100
  
  # Number of backup log files
  backup_count: 5
  
  # Log to console
  console: true

# Session Configuration
session:
  # Session timeout (seconds)
  timeout: 3600
  
  # Session storage: memory, file, redis
  storage: file
  
  # Session directory
  directory: ~/.hermes/sessions
  
  # Max sessions
  max_sessions: 100

# Security Configuration
security:
  # Enable security features
  enabled: true
  
  # Require authentication
  require_auth: true
  
  # Allowed IPs (empty = all)
  allowed_ips: []
  
  # Blocked IPs
  blocked_ips: []
  
  # SSL/TLS
  ssl:
    enabled: false
    cert_path: /path/to/cert.pem
    key_path: /path/to/key.pem
  
  # Rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60

# Advanced Configuration
advanced:
  # Enable debug mode
  debug: false
  
  # Enable experimental features
  experimental: false
  
  # Custom plugins directory
  plugins_dir: ~/.hermes/plugins
  
  # Custom skills directory
  skills_dir: ~/.hermes/skills
  
  # Enable telemetry
  telemetry: true
  
  # Telemetry endpoint
  telemetry_endpoint: "https://telemetry.hermes-agent.nousresearch.com"
```

## Configuration by Use Case

### Minimal Configuration (Local)

```yaml
model:
  default: anthropic/claude-opus-4.6
  base_url: https://openrouter.ai/api/v1

terminal:
  backend: local

memory:
  enabled: true
  type: short_term

api_server:
  enabled: false

logging:
  level: INFO
```

### Kubernetes Deployment

```yaml
model:
  default: anthropic/claude-opus-4.6
  base_url: https://openrouter.ai/api/v1
  temperature: 0.7

terminal:
  backend: docker
  docker:
    image: ubuntu:22.04

memory:
  enabled: true
  type: hybrid
  compression:
    enabled: true

api_server:
  enabled: true
  port: 8642
  host: 0.0.0.0
  cors:
    enabled: true

gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"

logging:
  level: INFO
  format: json
  file: /var/log/hermes/hermes.log
```

### Production Deployment

```yaml
model:
  default: anthropic/claude-opus-4.6
  base_url: https://openrouter.ai/api/v1
  temperature: 0.5

terminal:
  backend: docker
  docker:
    image: ubuntu:22.04

memory:
  enabled: true
  type: hybrid
  compression:
    enabled: true
    threshold: 500

api_server:
  enabled: true
  port: 8642
  host: 0.0.0.0
  cors:
    enabled: true
    allowed_origins:
      - "https://app.example.com"
  rate_limit:
    enabled: true
    requests_per_minute: 100

gateways:
  telegram:
    enabled: true
    token: "${TELEGRAM_BOT_TOKEN}"
  slack:
    enabled: true
    bot_token: "${SLACK_BOT_TOKEN}"

approvals:
  enabled: true
  mode: manual

security:
  enabled: true
  require_auth: true
  ssl:
    enabled: true
    cert_path: /etc/hermes/cert.pem
    key_path: /etc/hermes/key.pem

logging:
  level: WARNING
  format: json
  file: /var/log/hermes/hermes.log
  max_size: 500
  backup_count: 10
```

## Environment Variable Overrides

Any configuration value can be overridden via environment variables using the format:

```
HERMES_<SECTION>_<KEY>=value
```

Examples:

```bash
HERMES_MODEL_DEFAULT=gpt-4
HERMES_TERMINAL_BACKEND=docker
HERMES_API_SERVER_PORT=9000
HERMES_GATEWAYS_TELEGRAM_ENABLED=true
```

## Configuration Validation

Validate your configuration:

```bash
hermes config validate
hermes config show
```

## Reloading Configuration

Reload configuration without restart:

```bash
hermes config reload
```

## Next Steps

- **[Environment Variables](./ENVIRONMENT_VARIABLES.md)** — Secrets and API keys
- **[SOUL.md Customization](./SOUL_CUSTOMIZATION.md)** — Agent personality
- **[Gateway Configuration](./GATEWAY_CONFIGURATION.md)** — Security and allowlists

---

**Last Updated**: April 18, 2026
