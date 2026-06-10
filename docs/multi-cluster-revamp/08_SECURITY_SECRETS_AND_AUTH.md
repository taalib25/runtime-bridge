# Security, Secrets, Auth, TLS, RBAC, Terminal, and Redaction

## Purpose

This document defines security contracts for backend-to-bridge communication, cluster access, terminal access, TLS, secrets, and logs.

## Bridge Auth Model

Recommended V1/V1.5:

```txt
Backend signs short-lived JWT.
Bridge verifies using public key.
```

This avoids one shared secret per cluster.

## JWT Required Claims

```txt
iss = hermescloud-backend
aud = hermes-bridge
sub = service account or backend actor
cluster_id = target runtime cluster id
action = requested bridge action
agent_id = target agent id if applicable
operation_id = backend operation id if applicable
jti = unique token id
iat = issued at
nbf = not before
exp = short expiry, recommended 60 seconds
```

Example:

```json
{
  "iss": "hermescloud-backend",
  "aud": "hermes-bridge",
  "sub": "backend-runtime-control-plane",
  "cluster_id": "eu-1",
  "action": "instance.restart",
  "agent_id": "agent_abc123",
  "operation_id": "op_123",
  "jti": "jwt_456",
  "iat": 1780000000,
  "nbf": 1780000000,
  "exp": 1780000060
}
```

Bridge validation:

```txt
- signature valid
- exp not expired
- nbf valid
- aud == hermes-bridge
- iss == hermescloud-backend
- cluster_id matches bridge CLUSTER_ID
- action is allowed for endpoint
- agent_id matches path where applicable
- jti replay protection if feasible for high-risk actions
```

## TLS Rules

Production bridge URLs must be HTTPS.

Backend must reject:

```txt
http://NODE_IP:8080
http://bridge-...
empty bridge URL
unvalidated bridge URL
```

### V1 Minimum

```txt
Cloudflare proxied DNS for bridge domains
Backend talks to https://bridge-<cluster>.hermeshq.net
Firewall/security rules prevent raw bridge exposure where possible
```

### Stronger Origin TLS

Later add:

```txt
cert-manager
Cloudflare Origin Certificates
Traefik TLS configuration
mTLS if needed
```

## Secrets Placement

```txt
GitHub Actions:
  HCLOUD_TOKEN
  CLOUDFLARE_API_TOKEN
  CLOUDFLARE_ZONE_ID
  ADMIN_API_SECRET

Backend:
  BRIDGE_AUTH_PRIVATE_KEY
  database secrets
  billing secrets

Bridge:
  BRIDGE_AUTH_PUBLIC_KEY
  cluster_id
  runtime_base_domain

Agent runtime Secret:
  provider keys
  Telegram/Discord/Slack tokens
  per-agent credentials
```

Do not put provider keys or messenger tokens in ConfigMaps.

## Kubernetes RBAC

Bridge ServiceAccount should have only the permissions needed for customer runtime management.

Must not have:

```txt
cluster-admin
node shell access
secrets access outside managed namespaces except required platform config
access to kube-system unrelated resources
```

Bridge startup self-check must detect missing permissions without expanding permissions.

## Terminal Security

Terminal access is high-risk.

Requirements:

```txt
- backend authenticates user
- backend checks workspace/agent permission
- backend checks plan allows terminal
- terminal session record created before connect
- bridge only execs into allowed agent container
- command path cannot target arbitrary namespace/pod
- session start/end audited
- concurrent session limit per agent
- optional idle timeout
- no terminal into system namespaces
- no terminal into bridge pod
```

Terminal audit table:

```sql
create table terminal_sessions (
  id text primary key,
  agent_id text not null,
  cluster_id text not null,
  user_id text not null,
  status text not null,
  started_at timestamp,
  ended_at timestamp,
  ip_address text,
  user_agent text,
  reason text
);
```

Do not store full terminal keystrokes by default unless legal/product policy explicitly accepts it. Store session metadata first.

## Log Redaction

All admin-displayed logs must be redacted.

Redact patterns:

```txt
Authorization: Bearer ...
api_key=...
OPENAI_API_KEY=...
ANTHROPIC_API_KEY=...
GEMINI_API_KEY=...
TELEGRAM_BOT_TOKEN=...
xoxb- Slack tokens
Discord bot tokens
cookies
basic auth URLs
```

Admin raw log download:

```txt
requires elevated admin permission
is audited
warns about possible secrets
```

## Backend Registration Security

`POST /api/admin/clusters`:

```txt
- protected by ADMIN_API_SECRET (Authorization: Bearer)
- only GitHub Actions should call it
- idempotent upsert by cluster_id
- bridge_secret encrypted at rest (AES-256-GCM, SECRETS_ENCRYPTION_KEY)
- production bridge_url should be https:// (http:// is not production-ready)
```

## Network Policy

Customer agents should not freely access system namespaces or metadata endpoints.

Minimum goal:

```txt
- block access to instance metadata endpoint where possible
- restrict access to bridge/system services
- allow required outbound internet/API access according to plan
```

## Secret Rotation

Required rotation strategy:

```txt
BRIDGE_AUTH_PRIVATE_KEY:
  support key id/kid
  allow overlapping old/new public keys during rotation

ADMIN_API_SECRET:
  rotate manually through GitHub secret and backend config (ADMIN_API_SECRET env var)

Agent provider/messenger secrets:
  rotate through backend -> bridge -> Secret patch -> restart/reload
```
