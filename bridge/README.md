# Bridge

The bridge is a Go control service that runs inside the K3s cluster in the
`hermes-bridge` namespace. It manages Hermes runtime instances using the
Kubernetes API and Helm SDK.

See `docs/ARCHITECTURE.md` for the full architecture and `docs/CODING_AGENT_RULES.md`
before making changes.

## Responsibilities

| Area | Files |
|------|-------|
| Runtime lifecycle (create/update/delete) | `helm.go` |
| Runtime operations (restart/redeploy/rollback/repair) | `lifecycle.go` |
| Health + status collection | `status.go` |
| Background sync loop | `sync.go` |
| Async operations queue | `bridge.go` (`submitOperation`) |
| Provider API key management | `providers.go`, `handlers_config.go` |
| Messaging integration secrets/env | `integrations.go`, `handlers_integrations.go` |
| Agent template storage + apply | `agent_templates.go`, `handlers_agent_templates.go` |
| Workspace CRUD HTTP handlers | `handlers_workspace.go` |
| Traefik IngressRoute + middleware | `ingressroute.go` |
| WebSocket terminal exec (PTY) | `terminal_exec.go` |
| Auth middleware | `middleware.go` |
| Prometheus metrics | `metrics.go` |
| Config loading | `bridge_config.go` |
| All request/response types | `types.go` |

## Non-Responsibilities

The bridge does **not** own:
- Billing and subscriptions
- User authentication (it validates `X-Bridge-Secret` only)
- Product chat session storage
- Global cluster provisioning
- Hetzner server creation (`hetzner-k3s` CLI is used manually)

## Runtime Path

The bridge installs `charts/runtime-node-core/` via Helm SDK for every workspace.
It never shells out to `kubectl` or `helm` CLI.

## Overview (original)

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Backend API   │────▶│    Go Bridge    │────▶│   Helm SDK      │
│   (Hermes HQ)   │     │   (HTTP REST)   │     │   (Releases)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                      │
                                                      ▼
                                              ┌─────────────────┐
                                              │  Kubernetes     │
                                              │  (Deployments)  │
                                              └─────────────────┘
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/healthz` | GET | Service health check |
| `/readyz` | GET | Readiness check (kube + helm) |
| `/metrics` | GET | Prometheus metrics |
| `/v1/workspaces` | GET | List all workspaces |
| `/v1/workspaces/{id}` | POST | Create workspace (async) |
| `/v1/workspaces/{id}` | GET | Get workspace details |
| `/v1/workspaces/{id}` | PUT | Update workspace (async) |
| `/v1/workspaces/{id}` | DELETE | Delete workspace (async) |
| `/v1/workspaces/{id}/status` | GET | Get workspace status |
| `/v1/workspaces/{id}/health` | GET | Health check endpoint |

All `/v1/*` endpoints require authentication via `X-Bridge-Secret` header.

## Configuration

Environment variables (with defaults):

| Variable | Default | Description |
|----------|---------|-------------|
| `BRIDGE_LISTEN_ADDRESS` | `:8080` | HTTP server address |
| `BRIDGE_NAMESPACE` | `default` | Default namespace for releases |
| `BRIDGE_CLUSTER_NAME` | *(required)* | Cluster identifier |
| `BRIDGE_KUBECONFIG` | *(in-cluster)* | Path to kubeconfig |
| `BRIDGE_CHART_PATH` | *(required)* | Path to Helm chart |
| `BRIDGE_SECRET` | *(required)* | Authentication secret |
| `BRIDGE_SYNC_INTERVAL` | `5m` | Status sync interval |
| `BRIDGE_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `BRIDGE_HTTP_CLIENT_TIMEOUT` | `5s` | Health check timeout |
| `BRIDGE_OPERATION_TIMEOUT` | `10m` | Async operation timeout |
| `BRIDGE_HEALTH_PATH` | `/healthz` | Workspace health path |
| `BRIDGE_RELEASE_PREFIX` | *(empty)* | Prefix for release names |
| `BRIDGE_CREATE_NAMESPACE` | `false` | Create namespace if missing |

Alternatively, use a YAML config file via `BRIDGE_CONFIG_FILE`:

```yaml
listenAddress: ":8080"
namespace: "hermes-workspaces"
clusterName: "hermes-prod"
chartPath: "/app/charts/hermes-agent"
bridgeSecret: "${BRIDGE_SECRET}"
syncInterval: 5m
shutdownTimeout: 10s
httpClientTimeout: 5s
operationTimeout: 10m
healthPath: "/healthz"
releasePrefix: "ws-"
createNamespace: true
```

## Workspace Spec

```json
{
  "workspaceId": "ws-12345",
  "tenantId": "tenant-abc",
  "clusterId": "hermes-prod",
  "namespace": "hermes-workspaces",
  "image": "hermes-agent:latest",
  "imageTag": "v1.2.3",
  "imagePullPolicy": "IfNotPresent",
  "resources": {
    "cpuRequest": "100m",
    "cpuLimit": "500m",
    "memoryRequest": "128Mi",
    "memoryLimit": "512Mi"
  },
  "storage": {
    "enabled": true,
    "size": "10Gi",
    "storageClass": "hcloud-volumes"
  },
  "network": {
    "host": "workspace.hermeshq.net",
    "path": "/ws-12345",
    "ingressClassName": "traefik",
    "scheme": "https"
  },
  "env": [
    {"name": "LOG_LEVEL", "value": "info"}
  ],
  "secrets": {
    "API_KEY": "secret-value"
  },
  "healthCheckPath": "/healthz"
}
```

## Deployment

### Docker

```bash
docker build -t hermes-bridge:latest ./bridge/
docker run -e BRIDGE_CLUSTER_NAME=local \
           -e BRIDGE_CHART_PATH=/app/charts/hermes-agent \
           -e BRIDGE_SECRET=your-secret \
           hermes-bridge:latest
```

### Kubernetes

Apply manifests from `deploy/`:

```bash
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/secret.yaml  # Update secret value first
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/ingress.yaml  # If using ingress
```

### Helm (from this repo)

```bash
helm install hermes-bridge ./charts/hermes-agent \
  --set bridge.enabled=true \
  --set bridge.clusterName=hermes-prod \
  --set bridge.secret=your-secret
```

## Metrics

Prometheus metrics exposed at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `hermes_bridge_workspace_count` | Gauge | Active workspace count |
| `hermes_bridge_operation_latency_seconds` | Histogram | Operation latency |
| `hermes_bridge_operation_total` | Counter | Total operations by result |

## Sync Loop

The bridge runs a periodic sync loop that:

1. Lists all Helm releases with `bridge.workspace` metadata
2. Collects status for each workspace (Deployment, Pod, health check)
3. Logs alerts for unhealthy or stuck workspaces
4. Updates Prometheus metrics

## Development

```bash
# Build
go build -o /tmp/hermes-bridge ./bridge

# Run locally (requires kubeconfig)
export BRIDGE_CLUSTER_NAME=local
export BRIDGE_CHART_PATH=./charts/hermes-agent
export BRIDGE_SECRET=dev-secret
/tmp/hermes-bridge

# Test
curl -H "X-Bridge-Secret: dev-secret" http://localhost:8080/v1/workspaces
```

## Comparison: Operator vs Bridge

| Aspect | Operator | Go Bridge |
|--------|----------|-----------|
| Deployment | CRD + Controller | Helm release |
| Lifecycle | Reconcile loop | Sync loop + HTTP |
| State storage | CRD status | Release config + in-memory |
| Drift detection | Automatic | Periodic sync |
| Complexity | High (kubebuilder) | Low (Go + Helm SDK) |
| Dependencies | controller-runtime | helm.sh/helm/v3 |

## License

See main project LICENSE file.