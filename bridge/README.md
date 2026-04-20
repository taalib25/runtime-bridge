# Hermes Go Bridge

A lightweight HTTP service that manages Hermes workspace lifecycles via Helm SDK, replacing the Kubernetes operator approach.

## Overview

The Go Bridge provides a simpler alternative to the CRD/controller pattern:

- **No CRDs**: Workspaces are managed as Helm releases
- **No Controller**: Lifecycle logic runs in a sync loop instead of reconcile
- **Thin HTTP API**: RESTful endpoints for workspace CRUD operations
- **Helm SDK**: Direct Helm operations for install/upgrade/uninstall

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