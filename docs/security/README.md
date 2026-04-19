# Hermes Agent Security Documentation

Complete security and backend configuration reference for Hermes Agent.

## Documents

### [SECURITY_AND_BACKEND_COMPREHENSIVE.md](./SECURITY_AND_BACKEND_COMPREHENSIVE.md)

**Exhaustive reference covering all 15 security features:**

1. **Dangerous Command Approval System** — Manual, smart, and off modes with approval channels and timeouts
2. **YOLO Mode** — Disable all safety checks for trusted environments
3. **Command Allowlist** — Regex patterns for commands that bypass approval
4. **Container Isolation & Docker Security** — Security flags, capabilities, seccomp, resource limits
5. **Terminal Backends** — Local, Docker, SSH, Modal, Daytona, Singularity with isolation levels
6. **Environment Variable Passthrough** — Selective forwarding, wildcards, blocking
7. **Credential File Passthrough** — SSH keys, AWS credentials, Docker config, Kubernetes config, Git credentials
8. **MCP Credential Handling** — Environment variable substitution, credential scoping, token rotation
9. **SSRF Protection** — Blocked IP ranges, domains, DNS rebinding, URL encoding, octal/hex IP formats
10. **Tirith Pre-Exec Scanning** — Malware detection, privilege escalation, credential exposure, fail-open/closed
11. **Context File Injection Protection** — File integrity verification, permissions validation, content validation
12. **Website Blocklist** — Domain patterns, URL patterns, IP ranges, allowlist exceptions
13. **Privacy Settings & PII Redaction** — Email, phone, credit cards, SSN, API keys, passwords, custom patterns
14. **Group Session Isolation** — Per-user sessions in group chats, memory isolation, context isolation
15. **Unauthorized DM Behavior** — Open, allowlist, pairing, and disabled policies with code generation and expiration

## Quick Reference

### Configuration Files

```
~/.hermes/
├── config.yaml              # All behavioral settings
├── .env                     # Secrets (mode 0600)
├── SOUL.md                  # Agent personality
├── MEMORY.md                # Agent memories
├── USER.md                  # User profile
└── logs/
    ├── approvals.log        # Approval requests
    ├── ssrf.log             # SSRF blocks
    ├── tirith.log           # Tirith scans
    ├── blocklist.log        # Website blocks
    ├── pii_redaction.log    # PII detections
    ├── pairing.log          # DM pairing
    └── unauthorized_dm.log  # Unauthorized DMs
```

### Environment Variables

**Approval System:**
```bash
APPROVALS_MODE=manual|smart|off
APPROVALS_TIMEOUT=300
APPROVALS_CHANNELS=telegram,email,slack
APPROVALS_DANGEROUS_OPS=file_delete,system_reboot,network_change,database_drop
```

**Terminal Backend:**
```bash
TERMINAL_ENV=local|docker|ssh|modal|daytona|singularity
TERMINAL_DOCKER_IMAGE=nikolaik/python-nodejs:python3.11-nodejs20
TERMINAL_DOCKER_FORWARD_ENV=GITHUB_TOKEN,NPM_TOKEN
TERMINAL_DOCKER_VOLUMES=/home/user/projects:/workspace/projects,/home/user/data:/data:ro
TERMINAL_CONTAINER_CPU=1
TERMINAL_CONTAINER_MEMORY=5120
TERMINAL_CONTAINER_DISK=51200
TERMINAL_CONTAINER_PERSISTENT=true
```

**Security:**
```bash
SECURITY_SSRF_PROTECTION_ENABLED=true
SECURITY_SSRF_BLOCKED_IPS=127.0.0.1,169.254.169.254,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
SECURITY_SSRF_BLOCKED_DOMAINS=localhost,*.internal,*.local
SECURITY_TIRITH_ENABLED=true
SECURITY_TIRITH_TIMEOUT=5
SECURITY_TIRITH_FAIL_OPEN=true
SECURITY_PII_REDACTION_ENABLED=true
SECURITY_PII_REDACTION_REDACT_EMAILS=true
SECURITY_PII_REDACTION_REDACT_PHONE_NUMBERS=true
SECURITY_PII_REDACTION_REDACT_CREDIT_CARDS=true
SECURITY_PII_REDACTION_REDACT_SSN=true
SECURITY_PII_REDACTION_REDACT_API_KEYS=true
SECURITY_PII_REDACTION_REDACT_PASSWORDS=true
SECURITY_WEBSITE_BLOCKLIST_ENABLED=true
SECURITY_WEBSITE_BLOCKLIST_BLOCKED_DOMAINS=malware.com,phishing.com,*.scam
SECURITY_CONTEXT_FILE_INTEGRITY_ENABLED=true
SECURITY_CONTEXT_FILE_PERMISSIONS_ENABLED=true
```

**Messaging:**
```bash
GROUP_SESSIONS_PER_USER=true
TELEGRAM_DM_POLICY=pairing|allowlist|open|disabled
TELEGRAM_DM_PAIRING_TIMEOUT=300
TELEGRAM_DM_PAIRING_CODE_LENGTH=6
```

## Configuration Examples

### Secure Production Setup

```yaml
# config.yaml
approvals:
  mode: "manual"
  timeout: 300
  channels:
    - telegram
    - email
  dangerous_operations:
    - file_delete
    - system_reboot
    - network_change
    - database_drop

terminal:
  backend: "docker"
  docker_image: "nikolaik/python-nodejs:python3.11-nodejs20"
  docker_forward_env:
    - "GITHUB_TOKEN"
    - "NPM_TOKEN"
  docker_volumes:
    - "/home/user/projects:/workspace/projects:ro"
  container_cpu: 1
  container_memory: 5120
  container_disk: 51200
  container_persistent: false

security:
  ssrf_protection:
    enabled: true
    blocked_ips:
      - "127.0.0.1"
      - "169.254.169.254"
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
  
  tirith_enabled: true
  tirith_timeout: 5
  tirith_fail_open: false
  
  pii_redaction:
    enabled: true
    redact_emails: true
    redact_phone_numbers: true
    redact_credit_cards: true
    redact_ssn: true
    redact_api_keys: true
    redact_passwords: true
  
  website_blocklist:
    enabled: true
    blocked_domains:
      - "malware.com"
      - "phishing.com"
      - "*.scam"
  
  context_file_integrity:
    enabled: true
    algorithm: "sha256"
  
  context_file_permissions:
    enabled: true

group_sessions_per_user: true

gateways:
  telegram:
    dm_policy: "pairing"
    dm_pairing_timeout: 300
    dm_pairing_code_length: 6
```

### Development Setup

```yaml
# config.yaml
approvals:
  mode: "off"

terminal:
  backend: "local"
  timeout: 180

security:
  ssrf_protection:
    enabled: false
  
  tirith_enabled: false
  
  pii_redaction:
    enabled: false
  
  website_blocklist:
    enabled: false

group_sessions_per_user: false

gateways:
  telegram:
    dm_policy: "open"
```

## Security Checklist

- [ ] Approval system enabled (`approvals.mode: manual` or `smart`)
- [ ] Terminal backend isolated (Docker, SSH, or cloud)
- [ ] Environment variables whitelisted (not forwarding all)
- [ ] Credential files mounted read-only
- [ ] SSRF protection enabled
- [ ] Tirith scanning enabled
- [ ] PII redaction enabled
- [ ] Website blocklist configured
- [ ] Context file integrity enabled
- [ ] Group session isolation enabled
- [ ] DM policy configured (pairing or allowlist)
- [ ] Logs monitored regularly
- [ ] Credentials rotated monthly
- [ ] Allowlist reviewed quarterly

## Troubleshooting

### Approval Not Working

1. Check `APPROVALS_MODE` is not `off`
2. Verify approval channels configured
3. Check logs: `~/.hermes/logs/approvals.log`
4. Ensure user is in allowlist for approval channel

### SSRF Blocking Legitimate Requests

1. Add domain to `allowed_domains`
2. Add IP to `allowed_ips`
3. Check logs: `~/.hermes/logs/ssrf.log`

### PII Redaction Too Aggressive

1. Disable specific redaction types
2. Add custom patterns for false positives
3. Check logs: `~/.hermes/logs/pii_redaction.log`

### Docker Container Issues

1. Verify image exists: `docker images`
2. Check resource limits: `docker stats`
3. Verify volume mounts: `docker inspect`
4. Check logs: `docker logs <container>`

## Related Documentation

- [Configuration Reference](../configuration/CONFIG_REFERENCE.md)
- [Gateway Configuration](../configuration/GATEWAY_CONFIGURATION.md)
- [Environment Variables](../configuration/ENVIRONMENT_VARIABLES.md)
- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md)

---

**Last Updated:** April 18, 2026  
**Version:** 1.0
