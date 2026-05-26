# Hermes Bridge

A lightweight Go HTTP service that manages Hermes runtime instance lifecycles via the Helm SDK on a Kubernetes (k3s) cluster.

## Overview

- **No CRDs** — instances are Helm releases
- **No controller** — lifecycle logic runs via HTTP API + background sync loop
- **Async operations** — all mutating ops return an operation ID you can poll
- **Multi-bridge ready** — each bridge manages one cluster; the backend routes to the right bridge per instance

## Architecture

```
Backend API  ──▶  Bridge (HTTPS)  ──▶  Helm SDK  ──▶  k8s Deployments
                       │
                       └──▶  k8s Secrets (instance spec + provider keys)
                       └──▶  Traefik IngressRoutes (per instance)
```

Each instance gets its own namespace, Helm release, PVC, and Traefik IngressRoute. The bridge's `clusterId` (set via `BRIDGE_CLUSTER_NAME`) is returned on every create so the backend knows which bridge owns the instance.

---

## API Endpoints

All `/v1/*` endpoints require the `X-Bridge-Secret` header.

### System

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness — returns `{"status":"ok","cluster":"...","version":"..."}` |
| GET | `/readyz` | Readiness — checks kube + Helm connectivity |
| GET | `/metrics` | Prometheus metrics |

### Instances

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/instances` | List all instances |
| POST | `/v1/instances/{id}` | Create instance (async, 202) |
| GET | `/v1/instances/{id}` | Get instance details |
| PUT | `/v1/instances/{id}` | Update instance spec (async, 202) |
| DELETE | `/v1/instances/{id}` | Delete instance (async, 202) |
| DELETE | `/v1/instances/{id}?purge=true` | Permanently destroy instance + PVC (requires `X-Confirm-Data-Deletion: {id}` header) |
| GET | `/v1/instances/{id}/status` | Pod phase, health, replicas |
| GET | `/v1/instances/{id}/health` | 200 if healthy, 503 if not |
| GET | `/v1/instances/{id}/events` | Recent Kubernetes events |

### Lifecycle Operations

All ops are async — they return 202 with an `Operation` object. Poll `/v1/operations/{opId}` for result.
If an op is already in-flight for the instance, returns **409 Conflict** with the existing op ID.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/instances/{id}/restart` | Rolling restart (patch annotation) |
| POST | `/v1/instances/{id}/redeploy` | Re-run Helm upgrade from stored spec |
| POST | `/v1/instances/{id}/rollback` | Rollback to previous Helm revision (body: `{"version": N}`, 0 = previous) |
| POST | `/v1/instances/{id}/repair` | Auto-detect and fix pod issues |
| POST | `/v1/instances/{id}/upgrade` | Upgrade to a new image (see below) |
| GET | `/v1/instances/{id}/operations` | List ops for this instance (most recent first, default limit 20) |
| GET | `/v1/operations/{opId}` | Poll a specific operation |

#### Upgrade endpoint

Upgrades the running image via Helm rolling update. On failure, automatically rolls back to the previous Helm revision.

```bash
POST /v1/instances/{id}/upgrade
{
  "image": "ghcr.io/taalib25/runtime-node-core",  # optional — defaults to stored spec
  "imageTag": "v0.2.0"                              # required
}
```

Response on success: `"message": "upgrade completed: ghcr.io/taalib25/runtime-node-core:v0.2.0"`

Timeouts: 6 min for deployment ready + 2 min for health check. Auto-rollback fires if either fails.

### Terminal

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/instances/{id}/exec` | WebSocket exec into pod (auth via `?token=` short-lived token) |
| POST | `/v1/instances/{id}/terminal/recreate` | Issue a new terminal session token |

### Config

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/instances/{id}/config/providers` | List configured AI providers |
| POST | `/v1/instances/{id}/config/providers` | Add a provider |
| PUT | `/v1/instances/{id}/config/providers/{name}` | Update provider API key |
| DELETE | `/v1/instances/{id}/config/providers/{name}` | Remove a provider |
| PUT | `/v1/instances/{id}/config/model` | Set active provider + model |

Valid providers: `openai`, `anthropic`, `gemini`, `groq`, `nous`, `opencode-go`, `opencode-zen`, `mistral`, `openrouter`, `minimax`, `glm`, `kimi`, `huggingface`, `ai-gateway`

### Integrations & Agents

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/instances/{id}/integrations` | List messaging integrations |
| POST | `/v1/instances/{id}/integrations/{platform}` | Enable integration |
| DELETE | `/v1/instances/{id}/integrations/{platform}` | Disable integration |
| GET | `/v1/agents` | List agent templates |
| POST | `/v1/agents` | Create agent template |
| GET | `/v1/agents/{agentId}` | Get agent template |
| PUT | `/v1/agents/{agentId}` | Update agent template |
| DELETE | `/v1/agents/{agentId}` | Delete agent template |
| POST | `/v1/instances/{id}/agent` | Apply agent template to instance |
| GET | `/v1/instances/{id}/agent` | Get instance's agent config |

---

## Create Response

```json
{
  "instanceId": "ws-2d434ac4914de483",
  "clusterId": "hermes-test",
  "status": "provisioning",
  "url": "https://ws-2d434ac4914de483.hermeshq.net",
  "dashboardUrl": "https://dash-ws-2d434ac4914de483.hermeshq.net",
  "secrets": {
    "API_SERVER_KEY": "f75d0a..."
  }
}
```

The backend should store `clusterId` alongside `instanceId` to route future calls to the correct bridge.

---

## Operation Object

```json
{
  "id": "abc123",
  "type": "upgrade",
  "instanceId": "ws-2d434ac4914de483",
  "status": "running | succeeded | failed",
  "message": "upgrade completed: ghcr.io/taalib25/runtime-node-core:v0.2.0",
  "error": "",
  "startedAt": "2026-05-18T12:00:00Z",
  "completedAt": "2026-05-18T12:03:00Z"
}
```

---

## Instance Spec (POST/PUT body)

```json
{
  "instanceId": "ws-2d434ac4914de483",
  "tenantId": "tenant-abc",
  "image": "ghcr.io/taalib25/runtime-node-core",
  "imageTag": "0.1.0",
  "runtimeMode": "runtime-node-core",
  "plan": "backend-determined tier (e.g. free, pro)",
  "namespace": "ws-2d434ac4914de483",
  "createNamespace": true,
  "ingressEnabled": true,
  "network": {
    "host": "hermeshq.net",
    "subdomain": "ws-2d434ac4914de483"
  },
  "resources": {
    "cpuRequest": "100m",
    "cpuLimit": "500m",
    "memoryRequest": "256Mi",
    "memoryLimit": "1Gi"
  },
  "storage": {
    "enabled": true,
    "size": "10Gi",
    "storageClass": "hcloud-volumes"
  },
  "secrets": {
    "API_SERVER_KEY": "reuse-from-create-response"
  }
}
```

`API_SERVER_KEY` — generated by the bridge on first create and returned in the response. The backend must pass it back on every subsequent PUT so it isn't rotated.

---

## Purge Delete

Permanently destroys the instance namespace, PVC, and Helm history. Requires two signals:

```bash
DELETE /v1/instances/{id}?purge=true
X-Confirm-Data-Deletion: {id}
```

Without `purge=true`, delete keeps the namespace tombstoned and Helm history intact (rollback is possible).

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `BRIDGE_CLUSTER_NAME` | *(required)* | Cluster identifier returned on every response |
| `BRIDGE_CHART_PATH` | *(required)* | Path to the Helm chart directory |
| `BRIDGE_SECRET` | *(required)* | Shared secret for `X-Bridge-Secret` auth |
| `BRIDGE_LISTEN_ADDRESS` | `:8080` | HTTP listen address |
| `BRIDGE_NAMESPACE` | `default` | Default namespace for the bridge itself |
| `BRIDGE_KUBECONFIG` | *(in-cluster)* | Path to kubeconfig (omit when running in-cluster) |
| `BRIDGE_SYNC_INTERVAL` | `5m` | Background sync loop interval |
| `BRIDGE_OPERATION_TIMEOUT` | `10m` | Max duration for async operations |
| `BRIDGE_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown window |
| `BRIDGE_HTTP_CLIENT_TIMEOUT` | `5s` | Instance health check HTTP timeout |
| `BRIDGE_HEALTH_PATH` | `/health` | Path to poll on instances for health |
| `BRIDGE_RELEASE_PREFIX` | *(empty)* | Prefix prepended to Helm release names |
| `BRIDGE_CREATE_NAMESPACE` | `false` | Auto-create namespace if missing |
| `BRIDGE_FORWARD_AUTH_URL` | *(empty)* | Backend auth verify URL (Traefik ForwardAuth) |
| `BRIDGE_CORS_ORIGINS` | *(empty)* | Comma-separated allowed CORS origins |
| `BRIDGE_DEFAULT_DOMAIN` | *(empty)* | Domain for auto-generated ingress hosts |
| `BRIDGE_RUNTIME_NODE_CORE_IMAGE` | `ghcr.io/taalib25/runtime-node-core` | Default image repo |
| `BRIDGE_RUNTIME_NODE_CORE_TAG` | `0.1.0` | Default image tag |

---

## Metrics

Prometheus metrics at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `hermes_bridge_instance_count` | Gauge | Active instance count per cluster |
| `hermes_bridge_instance_health` | Gauge | Per-instance health (1=healthy, 0=unhealthy) |
| `hermes_bridge_operation_latency_seconds` | Histogram | Op latency by type and result |
| `hermes_bridge_operation_total` | Counter | Total ops by type and result |
| `hermes_bridge_http_requests_total` | Counter | HTTP requests by method, path, status |
| `hermes_bridge_http_request_duration_seconds` | Histogram | HTTP request duration |

---

## Multi-Bridge Setup

Each bridge manages one cluster. The backend stores `bridgeUrl` (the URL it called) and `clusterId` (from the create response) per instance, then routes all subsequent calls to the correct bridge:

```
bridge-eu.hermeshq.net  →  k3s cluster EU
bridge-us.hermeshq.net  →  k3s cluster US
```

No changes to the bridge itself are needed — deploy the same binary with a different `BRIDGE_CLUSTER_NAME` and kubeconfig.

---

## Image Upgrade Pipeline

Recommended flow for upgrading the runtime image across instances:

```
1. Build + push new image to GHCR (GitHub Action, manually triggered)
   → ghcr.io/taalib25/runtime-node-core:v0.2.0

2. For each instance to upgrade:
   POST /v1/instances/{id}/upgrade
   {"image": "ghcr.io/taalib25/runtime-node-core", "imageTag": "v0.2.0"}

3. Poll GET /v1/operations/{opId} until succeeded or failed
   → succeeded: "upgrade completed: ghcr.io/taalib25/runtime-node-core:v0.2.0"
   → failed:    "upgrade failed: ... — rolled back to revision N"
```

---

## Development

```bash
# Build
go build -o /tmp/hermes-bridge ./bridge/

# Test
go test ./bridge/... -v -count=1 -timeout 60s

# Vet
go vet ./bridge/...

# Run locally (requires kubeconfig)
export BRIDGE_CLUSTER_NAME=local
export BRIDGE_CHART_PATH=./charts/runtime-node-core
export BRIDGE_SECRET=dev-secret
/tmp/hermes-bridge

# Smoke test
curl -H "X-Bridge-Secret: dev-secret" http://localhost:8080/healthz
curl -H "X-Bridge-Secret: dev-secret" http://localhost:8080/v1/instances
```
