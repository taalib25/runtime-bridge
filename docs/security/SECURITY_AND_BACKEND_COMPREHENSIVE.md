# Hermes Agent - Comprehensive Security & Backend Configuration Guide

**Last Updated:** April 18, 2026  
**Hermes Version:** v0.10.0+  
**Document Version:** 1.0

---

## Table of Contents

1. [Dangerous Command Approval System](#dangerous-command-approval-system)
2. [YOLO Mode](#yolo-mode)
3. [Command Allowlist](#command-allowlist)
4. [Container Isolation & Docker Security](#container-isolation--docker-security)
5. [Terminal Backends](#terminal-backends)
6. [Environment Variable Passthrough](#environment-variable-passthrough)
7. [Credential File Passthrough](#credential-file-passthrough)
8. [MCP Credential Handling](#mcp-credential-handling)
9. [SSRF Protection](#ssrf-protection)
10. [Tirith Pre-Exec Scanning](#tirith-pre-exec-scanning)
11. [Context File Injection Protection](#context-file-injection-protection)
12. [Website Blocklist](#website-blocklist)
13. [Privacy Settings & PII Redaction](#privacy-settings--pii-redaction)
14. [Group Session Isolation](#group-session-isolation)
15. [Unauthorized DM Behavior](#unauthorized-dm-behavior)

---

## Dangerous Command Approval System

### Overview

The approval system prevents accidental or malicious execution of dangerous commands by requiring explicit user confirmation before execution.

### Approval Modes

#### 1. **Manual Mode** (Default)
```yaml
approvals:
  mode: "manual"
```

**Behavior:**
- Agent detects dangerous patterns and **pauses execution**
- Sends approval request to configured channels (Telegram, Email, etc.)
- Waits for user confirmation before proceeding
- Timeout: 300 seconds (5 minutes) by default
- If timeout expires: **command is NOT executed**

**Environment Variable:**
```bash
APPROVALS_MODE=manual
APPROVALS_TIMEOUT=300
```

**Example Flow:**
```
1. Agent: "I'll delete old logs with: rm -rf /var/log/old"
2. [APPROVAL REQUIRED] Dangerous pattern detected: recursive delete
3. Approval request sent to Telegram
4. User confirms in Telegram: "✅ Approve"
5. Command executes
```

#### 2. **Smart Mode**
```yaml
approvals:
  mode: "smart"
```

**Behavior:**
- Uses auxiliary LLM to classify command danger level
- **Low-risk commands** (e.g., `ls`, `cat`) → Execute immediately
- **Medium-risk commands** (e.g., `rm file.txt`) → Require approval
- **High-risk commands** (e.g., `rm -rf /`) → Always require approval
- Configurable risk thresholds

**Configuration:**
```yaml
auxiliary:
  approval:
    provider: "auto"
    model: "google/gemini-3-flash-preview"
    timeout: 30
```

#### 3. **Off Mode**
```yaml
approvals:
  mode: "off"
```

**Behavior:**
- **No approval required** for any command
- ⚠️ **DANGEROUS** — Use only in trusted environments
- All dangerous patterns execute immediately

**Environment Variable:**
```bash
APPROVALS_MODE=off
```

### Dangerous Command Patterns

Automatically detected and trigger approval:

| Pattern | Example | Risk Level |
|---------|---------|-----------|
| Recursive delete | `rm -rf /path` | **CRITICAL** |
| Destructive file ops | `dd if=/dev/zero of=/dev/sda` | **CRITICAL** |
| SQL destructive | `DROP TABLE users` | **CRITICAL** |
| SQL delete without WHERE | `DELETE FROM users` | **CRITICAL** |
| System reboot | `sudo reboot`, `shutdown -h now` | **CRITICAL** |
| Network config change | `ip route add`, `iptables -F` | **HIGH** |
| Sudo without password | `sudo -S` with empty password | **HIGH** |
| Credential exposure | `echo $PASSWORD`, `cat ~/.ssh/id_rsa` | **HIGH** |
| Package manager force | `apt-get remove -y`, `yum erase` | **MEDIUM** |
| Single file delete | `rm important_file.txt` | **MEDIUM** |
| Config overwrite | `> /etc/config.conf` | **MEDIUM** |

### Approval Channels

Configure where approval requests are sent:

```yaml
approvals:
  channels:
    - telegram
    - email
    - slack
```

**Supported Channels:**
- `telegram` — Telegram DM or channel
- `email` — Email notification
- `slack` — Slack DM or channel
- `discord` — Discord DM
- `whatsapp` — WhatsApp message
- `signal` — Signal message

**Environment Variable:**
```bash
APPROVALS_CHANNELS=telegram,email
```

### Approval Timeout

```yaml
approvals:
  timeout: 300  # 5 minutes
```

**Behavior:**
- If user doesn't respond within timeout: **command is NOT executed**
- Agent logs: `[APPROVAL TIMEOUT] Command not approved within 300s`
- Execution continues without the dangerous command

**Environment Variable:**
```bash
APPROVALS_TIMEOUT=300
```

### Dangerous Operations List

Explicit list of operations requiring approval:

```yaml
approvals:
  dangerous_operations:
    - file_delete
    - file_recursive_delete
    - system_reboot
    - network_change
    - database_drop
    - database_delete_all
    - credential_exposure
    - package_uninstall
    - config_overwrite
```

**Environment Variable:**
```bash
APPROVALS_DANGEROUS_OPS=file_delete,system_reboot,network_change,database_drop
```

---

## YOLO Mode

### Overview

YOLO (You Only Live Once) mode disables **all safety checks** and approval requirements. Use only in trusted, isolated environments.

### Enabling YOLO Mode

```yaml
approvals:
  mode: "off"

security:
  yolo_mode: true
```

**Environment Variables:**
```bash
APPROVALS_MODE=off
HERMES_YOLO_MODE=true
```

**CLI Flag:**
```bash
hermes chat --yolo
```

### What YOLO Mode Disables

| Feature | Normal | YOLO |
|---------|--------|------|
| Approval system | ✓ | ✗ |
| Dangerous command detection | ✓ | ✗ |
| SSRF protection | ✓ | ✗ |
| Tirith scanning | ✓ | ✗ |
| Context injection protection | ✓ | ✗ |
| Website blocklist | ✓ | ✗ |
| PII redaction | ✓ | ✗ |
| Rate limiting | ✓ | ✗ |

### YOLO Mode Warning

When YOLO mode is enabled, Hermes displays:

```
⚠️  YOLO MODE ENABLED
All safety checks disabled. Use only in trusted environments.
```

---

## Command Allowlist

### Overview

Allowlist specific commands that should **always execute** without approval, even if they match dangerous patterns.

### Configuration

```yaml
command_allowlist:
  - "^rm -rf /tmp"
  - "^sudo reboot"
  - "^docker rm"
  - "^git reset --hard"
```

**Environment Variable:**
```bash
COMMAND_ALLOWLIST=^rm -rf /tmp,^sudo reboot,^docker rm
```

### Allowlist Patterns

Patterns are **regex** (case-sensitive by default):

| Pattern | Matches | Doesn't Match |
|---------|---------|---------------|
| `^rm -rf /tmp` | `rm -rf /tmp/old` | `rm -rf /var/tmp` |
| `^sudo reboot` | `sudo reboot` | `sudo reboot -f` |
| `docker rm .*` | `docker rm container1` | `docker rmi image1` |
| `^git reset` | `git reset --hard` | `git reset-hard` |

---

## Container Isolation & Docker Security

### Overview

Docker backend provides full process isolation using Linux namespaces, cgroups, and seccomp.

### Docker Backend Configuration

```yaml
terminal:
  backend: "docker"
  cwd: "/workspace"
  timeout: 180
  lifetime_seconds: 300
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_mount_cwd_to_workspace: false
  docker_forward_env:
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
  docker_volumes:
    - "/home/user/projects:/workspace/projects"
    - "/home/user/data:/data:ro"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true
```

### Security Flags

#### 1. **Read-Only Root Filesystem**
```yaml
docker_security:
  read_only_root_fs: true
```

**Effect:**
- Container root filesystem is read-only
- Only `/tmp`, `/var/tmp`, `/workspace` are writable
- Prevents modification of system files

#### 2. **Drop Capabilities**
```yaml
docker_security:
  cap_drop:
    - ALL
  cap_add:
    - NET_BIND_SERVICE
    - CHOWN
```

**Default Dropped Capabilities:**
- `CAP_SYS_ADMIN` — Prevent privilege escalation
- `CAP_NET_ADMIN` — Prevent network manipulation
- `CAP_SYS_MODULE` — Prevent kernel module loading
- `CAP_SYS_BOOT` — Prevent system reboot

#### 3. **No New Privileges**
```yaml
docker_security:
  no_new_privileges: true
```

**Effect:**
- Prevents privilege escalation via setuid/setgid
- Processes cannot gain more privileges than parent

#### 4. **Seccomp Profile**
```yaml
docker_security:
  seccomp_profile: "default"  # default | unconfined | custom_path
```

**Profiles:**
- `default` — Docker default seccomp (blocks ~50 syscalls)
- `unconfined` — No seccomp filtering (⚠️ not recommended)
- `custom_path` — Path to custom seccomp JSON

### Resource Limits

#### CPU Limits
```yaml
container_cpu: 1  # 1 core
```

#### Memory Limits
```yaml
container_memory: 5120  # 5 GB
```

#### Disk Limits
```yaml
container_disk: 51200  # 50 GB
```

#### Process Limits
```yaml
docker_security:
  pids_limit: 256
```

---

## Terminal Backends

### Overview

Hermes supports 6 terminal backends with different isolation levels and use cases.

### Backend Comparison

| Backend | Isolation | Latency | Cost | Best For |
|---------|-----------|---------|------|----------|
| `local` | None | Minimal | Free | Development |
| `docker` | Full (namespaces) | Low | Free | Safe sandboxing |
| `ssh` | Network boundary | Medium | Low | Remote servers |
| `modal` | Full (cloud) | Medium | $$ | GPU access |
| `daytona` | Full (cloud) | Medium | $$ | Managed environments |
| `singularity` | Namespaces | Low | Free | HPC clusters |

### 1. Local Backend

```yaml
terminal:
  backend: "local"
  cwd: "."
  timeout: 180
  lifetime_seconds: 300
```

**Characteristics:**
- Runs commands on host machine
- No isolation
- Full access to host filesystem
- Fastest execution

### 2. Docker Backend

```yaml
terminal:
  backend: "docker"
  cwd: "/workspace"
  timeout: 180
  lifetime_seconds: 300
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_mount_cwd_to_workspace: false
  docker_forward_env:
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
  docker_volumes:
    - "/home/user/projects:/workspace/projects"
    - "/home/user/data:/data:ro"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: true
```

**Characteristics:**
- Full process isolation via Linux namespaces
- Resource limits via cgroups
- Separate filesystem
- Network isolation

### 3. SSH Backend

```yaml
terminal:
  backend: "ssh"
  cwd: "/home/myuser/project"
  timeout: 180
  lifetime_seconds: 300
  persistent_shell: true
```

**Required Environment Variables:**
```bash
TERMINAL_SSH_HOST=remote.example.com
TERMINAL_SSH_USER=myuser
TERMINAL_SSH_PORT=22
TERMINAL_SSH_KEY=~/.ssh/id_rsa
TERMINAL_SSH_PERSISTENT=true
```

### 4. Modal Backend

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
  container_persistent: true
```

**Required Environment Variables:**
```bash
MODAL_TOKEN_ID=token_id_here
MODAL_TOKEN_SECRET=token_secret_here
```

### 5. Daytona Backend

```yaml
terminal:
  backend: "daytona"
  cwd: "~"
  timeout: 180
  lifetime_seconds: 300
  daytona_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  container_cpu: 1
  container_memory: 5120
  container_disk: 10240
  container_persistent: true
```

**Required Environment Variables:**
```bash
DAYTONA_API_KEY=api_key_here
DAYTONA_API_URL=https://api.daytona.io
```

### 6. Singularity/Apptainer Backend

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
  container_persistent: true
```

---

## Environment Variable Passthrough

### Overview

Control which environment variables are forwarded from host to container.

### Docker Environment Forwarding

```yaml
terminal:
  backend: "docker"
  docker_forward_env:
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
    - "AWS_ACCESS_KEY_ID"
    - "AWS_SECRET_ACCESS_KEY"
```

**Environment Variable:**
```bash
TERMINAL_DOCKER_FORWARD_ENV=GITHUB_TOKEN,NPM_TOKEN,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY
```

### Selective Forwarding

Only forward specific variables:

```yaml
docker_forward_env:
  - "GITHUB_TOKEN"
  - "NPM_TOKEN"
```

**Effect:**
- Only listed variables forwarded
- All other host variables blocked
- Prevents credential leakage

### Wildcard Forwarding

```yaml
docker_forward_env:
  - "AWS_*"
  - "GITHUB_*"
```

**Effect:**
- All variables matching pattern forwarded
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` forwarded
- `GITHUB_TOKEN`, `GITHUB_USER` forwarded

---

## Credential File Passthrough

### Overview

Mount credential files into containers securely without exposing them to the agent.

### SSH Key Mounting

```yaml
terminal:
  backend: "docker"
  docker_volumes:
    - "/home/user/.ssh:/root/.ssh:ro"
```

**Effect:**
- SSH keys mounted read-only
- Container can use keys for git/ssh
- Keys cannot be modified

### AWS Credentials

```yaml
docker_volumes:
  - "/home/user/.aws:/root/.aws:ro"
```

### Docker Config

```yaml
docker_volumes:
  - "/home/user/.docker:/root/.docker:ro"
```

### Kubernetes Config

```yaml
docker_volumes:
  - "/home/user/.kube:/root/.kube:ro"
```

### Git Credentials

```yaml
docker_volumes:
  - "/home/user/.gitconfig:/root/.gitconfig:ro"
  - "/home/user/.git-credentials:/root/.git-credentials:ro"
```

---

## MCP Credential Handling

### Overview

Model Context Protocol (MCP) servers may require credentials. Hermes securely manages MCP credentials.

### MCP Server Configuration

```yaml
mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
  
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user"]
  
  notion:
    url: https://mcp.notion.com/mcp
    headers:
      Authorization: "Bearer ${NOTION_API_KEY}"
```

### Environment Variable Substitution

```yaml
mcp_servers:
  github:
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
```

**Effect:**
- `${GITHUB_TOKEN}` replaced with value from `.env`
- Credentials never hardcoded in config
- Supports all env vars

### Credential Scoping

```yaml
mcp_servers:
  github:
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN_READ_ONLY}"
```

**Effect:**
- Use read-only tokens for MCP servers
- Limits damage if MCP server compromised

---

## SSRF Protection

### Overview

Server-Side Request Forgery (SSRF) protection prevents the agent from making requests to internal/private networks.

### SSRF Configuration

```yaml
security:
  ssrf_protection:
    enabled: true
    blocked_ips:
      - "127.0.0.1"
      - "169.254.169.254"  # AWS metadata
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
    blocked_domains:
      - "localhost"
      - "*.internal"
      - "*.local"
    allowed_ips: []
    allowed_domains: []
```

**Environment Variables:**
```bash
SECURITY_SSRF_PROTECTION_ENABLED=true
SECURITY_SSRF_BLOCKED_IPS=127.0.0.1,169.254.169.254,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
SECURITY_SSRF_BLOCKED_DOMAINS=localhost,*.internal,*.local
```

### Blocked IP Ranges

| Range | Purpose | Blocked |
|-------|---------|---------|
| `127.0.0.1` | Localhost | ✓ |
| `169.254.169.254` | AWS metadata | ✓ |
| `10.0.0.0/8` | Private network | ✓ |
| `172.16.0.0/12` | Private network | ✓ |
| `192.168.0.0/16` | Private network | ✓ |

### Allowed Exceptions

```yaml
security:
  ssrf_protection:
    allowed_ips:
      - "10.0.1.100"  # Specific internal service
    allowed_domains:
      - "internal-api.company.com"
```

---

## Tirith Pre-Exec Scanning

### Overview

Tirith is a security scanner that analyzes commands before execution to detect malicious patterns.

### Tirith Configuration

```yaml
security:
  tirith_enabled: true
  tirith_path: "tirith"
  tirith_timeout: 5
  tirith_fail_open: true
```

**Environment Variables:**
```bash
SECURITY_TIRITH_ENABLED=true
SECURITY_TIRITH_PATH=tirith
SECURITY_TIRITH_TIMEOUT=5
SECURITY_TIRITH_FAIL_OPEN=true
```

### Tirith Installation

```bash
# Install tirith
cargo install tirith

# Or download binary
wget https://github.com/unode/tirith/releases/download/v1.0/tirith-linux-x64
chmod +x tirith-linux-x64
```

### Tirith Scanning

Tirith scans commands for:

| Pattern | Risk | Example |
|---------|------|---------|
| Credential exposure | HIGH | `echo $AWS_SECRET_ACCESS_KEY` |
| Privilege escalation | HIGH | `sudo -i`, `su -` |
| Malware signatures | CRITICAL | Known malware commands |
| Suspicious syscalls | MEDIUM | `ptrace`, `process_vm_readv` |
| Network exfiltration | HIGH | `curl http://attacker.com` |
| Reverse shell | CRITICAL | `bash -i >& /dev/tcp/...` |

### Fail-Open vs Fail-Closed

#### Fail-Open (Default)
```yaml
tirith_fail_open: true
```

**Behavior:**
- If Tirith unavailable/timeout: **allow command**
- Logs warning
- Continues execution

#### Fail-Closed
```yaml
tirith_fail_open: false
```

**Behavior:**
- If Tirith unavailable/timeout: **block command**
- Logs error
- Requires manual approval

---

## Context File Injection Protection

### Overview

Prevent injection of malicious content into context files that could manipulate the agent.

### Context File Types

| File | Purpose | Risk |
|------|---------|------|
| `SOUL.md` | Agent personality | HIGH |
| `MEMORY.md` | Agent memories | MEDIUM |
| `USER.md` | User profile | MEDIUM |
| `.env` | Secrets | CRITICAL |
| `config.yaml` | Configuration | HIGH |

### Injection Protection

#### 1. **File Integrity Verification**
```yaml
security:
  context_file_integrity:
    enabled: true
    algorithm: "sha256"
```

**Effect:**
- Computes hash of context files on startup
- Detects unauthorized modifications
- Blocks if hash mismatch

#### 2. **File Permissions Validation**
```yaml
security:
  context_file_permissions:
    enabled: true
    expected_permissions:
      SOUL.md: "0644"
      MEMORY.md: "0644"
      USER.md: "0644"
      .env: "0600"
      config.yaml: "0644"
```

**Effect:**
- Verifies file permissions on startup
- Blocks if permissions too permissive
- Logs permission violations

#### 3. **Content Validation**
```yaml
security:
  context_file_validation:
    enabled: true
    max_file_size:
      SOUL.md: 10000
      MEMORY.md: 50000
      USER.md: 10000
```

**Effect:**
- Validates file size
- Detects suspicious growth
- Blocks oversized files

---

## Website Blocklist

### Overview

Prevent the agent from accessing certain websites via web_search, web_extract, and browser tools.

### Blocklist Configuration

```yaml
security:
  website_blocklist:
    enabled: true
    blocked_domains:
      - "malware.com"
      - "phishing.com"
      - "*.scam"
      - "attacker.com"
    blocked_urls:
      - "http://internal-admin.local/*"
      - "https://*/admin/*"
    blocked_ip_ranges:
      - "10.0.0.0/8"
      - "192.168.0.0/16"
```

**Environment Variables:**
```bash
SECURITY_WEBSITE_BLOCKLIST_ENABLED=true
SECURITY_WEBSITE_BLOCKLIST_BLOCKED_DOMAINS=malware.com,phishing.com,*.scam,attacker.com
SECURITY_WEBSITE_BLOCKLIST_BLOCKED_URLS=http://internal-admin.local/*,https://*/admin/*
SECURITY_WEBSITE_BLOCKLIST_BLOCKED_IP_RANGES=10.0.0.0/8,192.168.0.0/16
```

### Blocklist Patterns

#### Domain Patterns
```yaml
blocked_domains:
  - "malware.com"           # Exact match
  - "*.malware.com"         # Subdomain wildcard
  - "malware.*"             # TLD wildcard
```

#### URL Patterns
```yaml
blocked_urls:
  - "http://internal/*"     # Path wildcard
  - "https://*/admin/*"     # Multiple wildcards
  - "*://localhost/*"       # Protocol wildcard
```

#### IP Range Patterns
```yaml
blocked_ip_ranges:
  - "10.0.0.0/8"            # CIDR notation
  - "192.168.1.0/24"        # Subnet
  - "127.0.0.1"             # Single IP
```

### Allowlist Exceptions

```yaml
security:
  website_blocklist:
    allowed_domains:
      - "trusted-internal.company.com"
    allowed_urls:
      - "https://internal-api.company.com/public/*"
```

---

## Privacy Settings & PII Redaction

### Overview

Automatically detect and redact Personally Identifiable Information (PII) from logs and outputs.

### PII Redaction Configuration

```yaml
security:
  pii_redaction:
    enabled: true
    redact_emails: true
    redact_phone_numbers: true
    redact_credit_cards: true
    redact_ssn: true
    redact_api_keys: true
    redact_passwords: true
    redact_ips: false
    redact_custom_patterns:
      - pattern: "\\b[A-Z]{2}\\d{6}\\b"
        name: "passport_number"
```

**Environment Variables:**
```bash
SECURITY_PII_REDACTION_ENABLED=true
SECURITY_PII_REDACTION_REDACT_EMAILS=true
SECURITY_PII_REDACTION_REDACT_PHONE_NUMBERS=true
SECURITY_PII_REDACTION_REDACT_CREDIT_CARDS=true
SECURITY_PII_REDACTION_REDACT_SSN=true
SECURITY_PII_REDACTION_REDACT_API_KEYS=true
SECURITY_PII_REDACTION_REDACT_PASSWORDS=true
```

### PII Types

#### Email Addresses
```yaml
redact_emails: true
```

**Example:**
```
Before: user@example.com
After:  [EMAIL_REDACTED]
```

#### Phone Numbers
```yaml
redact_phone_numbers: true
```

**Example:**
```
Before: +1 (555) 123-4567
After:  [PHONE_REDACTED]
```

#### Credit Cards
```yaml
redact_credit_cards: true
```

**Example:**
```
Before: 4532-1234-5678-9010
After:  [CREDIT_CARD_REDACTED]
```

#### Social Security Numbers
```yaml
redact_ssn: true
```

**Example:**
```
Before: 123-45-6789
After:  [SSN_REDACTED]
```

#### API Keys
```yaml
redact_api_keys: true
```

**Example:**
```
Before: OPENAI_API_KEY=sk-proj-abc123xyz...
After:  OPENAI_API_KEY=[API_KEY_REDACTED]
```

#### Passwords
```yaml
redact_passwords: true
```

**Example:**
```
Before: password=hunter2
After:  password=[PASSWORD_REDACTED]
```

### PII Redaction Scope

#### Logs
```yaml
security:
  pii_redaction:
    redact_logs: true
```

#### Session Transcripts
```yaml
security:
  pii_redaction:
    redact_sessions: true
```

#### Memory Files
```yaml
security:
  pii_redaction:
    redact_memory: true
```

#### API Responses
```yaml
security:
  pii_redaction:
    redact_api_responses: true
```

---

## Group Session Isolation

### Overview

Isolate sessions per user in group chats to prevent cross-user interference.

### Group Session Configuration

```yaml
group_sessions_per_user: true
```

**Environment Variable:**
```bash
GROUP_SESSIONS_PER_USER=true
```

### Behavior

#### With Isolation (Default)
```yaml
group_sessions_per_user: true
```

**Effect:**
- Each user in group has separate session
- User A's memory doesn't affect User B
- User A's context doesn't affect User B
- Separate tool call history per user

#### Without Isolation
```yaml
group_sessions_per_user: false
```

**Effect:**
- Single shared session for entire group
- All users share memory
- All users share context
- Potential for cross-user interference

### Session Storage

```
~/.hermes/sessions/
├── group_dev-team_alice_20260418_103045.json
├── group_dev-team_bob_20260418_103100.json
└── group_dev-team_charlie_20260418_103115.json
```

### Memory Isolation

```
~/.hermes/memories/
├── alice/
│   ├── MEMORY.md
│   └── USER.md
├── bob/
│   ├── MEMORY.md
│   └── USER.md
└── charlie/
    ├── MEMORY.md
    └── USER.md
```

---

## Unauthorized DM Behavior

### Overview

Control how Hermes handles Direct Messages (DMs) from unauthorized users.

### DM Policy Configuration

```yaml
gateways:
  telegram:
    dm_policy: "pairing"  # open | allowlist | pairing | disabled
    dm_pairing_timeout: 300
    dm_pairing_code_length: 6
```

**Environment Variables:**
```bash
TELEGRAM_DM_POLICY=pairing
TELEGRAM_DM_PAIRING_TIMEOUT=300
TELEGRAM_DM_PAIRING_CODE_LENGTH=6
```

### DM Policies

#### 1. Open Policy
```yaml
dm_policy: "open"
```

**Behavior:**
- Any user can DM the bot
- No authorization required
- ⚠️ **Not recommended** for production

#### 2. Allowlist Policy
```yaml
dm_policy: "allowlist"
allowed_users:
  - 123456789
  - 987654321
```

**Behavior:**
- Only allowlisted users can DM
- Unauthorized users get rejection message
- No pairing required

#### 3. Pairing Policy (Default)
```yaml
dm_policy: "pairing"
dm_pairing_timeout: 300
dm_pairing_code_length: 6
```

**Behavior:**
- New users must pair with bot
- User initiates pairing: `/pair`
- Bot generates pairing code
- User confirms in group chat
- After confirmation: user can DM

**Pairing Flow:**
```
1. User (DM): /pair
2. Bot (DM): "Pairing code: 123456. Confirm in group chat."
3. User (Group): /confirm 123456
4. Bot (Group): "Pairing confirmed for @user"
5. User (DM): Now can send messages
```

#### 4. Disabled Policy
```yaml
dm_policy: "disabled"
```

**Behavior:**
- DMs completely disabled
- All DM attempts rejected
- Users must use group chat

### Pairing Code Security

#### Code Generation
```yaml
dm_pairing:
  code_length: 6
  code_format: "numeric"  # numeric | alphanumeric | hex
```

**Formats:**
- `numeric` — 0-9 (1 million combinations)
- `alphanumeric` — 0-9, a-z, A-Z (2.2 billion combinations)
- `hex` — 0-9, a-f (1 million combinations)

#### Code Expiration
```yaml
dm_pairing:
  code_ttl: 300  # 5 minutes
  max_attempts: 3
```

**Effect:**
- Code expires after 5 minutes
- Max 3 confirmation attempts
- After 3 failures: code invalidated

### Per-Platform DM Policies

```yaml
gateways:
  telegram:
    dm_policy: "pairing"
  
  discord:
    dm_policy: "allowlist"
    allowed_users:
      - "user_id_1"
      - "user_id_2"
  
  slack:
    dm_policy: "open"
  
  whatsapp:
    dm_policy: "disabled"
```

---

## Summary Table

| Feature | Config Key | Default | Modes |
|---------|-----------|---------|-------|
| Approval System | `approvals.mode` | `manual` | manual, smart, off |
| YOLO Mode | `security.yolo_mode` | `false` | true, false |
| Command Allowlist | `command_allowlist` | `[]` | regex patterns |
| Docker Security | `docker_security.*` | Enabled | Various flags |
| Terminal Backend | `terminal.backend` | `local` | local, docker, ssh, modal, daytona, singularity |
| Env Forwarding | `docker_forward_env` | `[]` | Env var names |
| Credential Files | `docker_volumes` | `[]` | Volume mounts |
| MCP Credentials | `mcp_servers.*.env` | `{}` | Env var substitution |
| SSRF Protection | `security.ssrf_protection.enabled` | `true` | true, false |
| Tirith Scanning | `security.tirith_enabled` | `true` | true, false |
| Context Injection | `security.context_file_integrity.enabled` | `true` | true, false |
| Website Blocklist | `security.website_blocklist.enabled` | `true` | true, false |
| PII Redaction | `security.pii_redaction.enabled` | `true` | true, false |
| Group Sessions | `group_sessions_per_user` | `true` | true, false |
| DM Policy | `gateways.*.dm_policy` | `pairing` | open, allowlist, pairing, disabled |

---

## Best Practices

### 1. **Defense in Depth**
- Enable multiple security layers
- Don't rely on single mechanism
- Combine approval + allowlist + SSRF protection

### 2. **Principle of Least Privilege**
- Forward only needed env vars
- Mount only needed volumes
- Use read-only mounts when possible
- Allowlist specific commands

### 3. **Regular Audits**
- Review approval logs weekly
- Check blocklist effectiveness
- Monitor PII redaction
- Audit credential access

### 4. **Credential Rotation**
- Rotate API keys monthly
- Use short-lived tokens
- Revoke old credentials
- Monitor credential usage

### 5. **Isolation Strategy**
- Use Docker for untrusted code
- Use SSH for remote execution
- Use Modal for GPU workloads
- Use Daytona for team collaboration

### 6. **Logging & Monitoring**
- Enable all security logs
- Monitor for suspicious patterns
- Alert on policy violations
- Archive logs for compliance

---

**Last Updated:** April 18, 2026  
**Version:** 1.0  
**Maintained By:** Hermes Security Team
