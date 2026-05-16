# Hermes Runtime Operator

A Go service that manages tenant Hermes instances on Kubernetes via Helm. It exposes a simple HTTP API that the backend calls to create, update, and delete isolated runtime environments — each backed by a Helm release of the `hermes-agent` chart.

## Architecture

```
Backend
  │
  │  POST /v1/instances/{id}   (tenantId, plan, config)
  ▼
Bridge (this service)
  │
  │  helm install/upgrade/uninstall
  ▼
hermes-agent Helm release
  └── Deployment (nousresearch/hermes-agent:latest)
  └── Service
  └── Ingress  ({workspace-id}.hermeshq.net)
  └── PersistentVolumeClaim
  └── ConfigMap (Hermes config.yaml + SOUL.md)
```

**Cluster:** Single Hetzner VM running k3s, managed via hetzner-k3s CLI.  
**Ingress:** Traefik (k3s default) with Cloudflare DNS (`*.hermeshq.net → 178.104.185.60`).  
**Bridge endpoint:** `http://bridge.hermeshq.net`

## API

All workspace endpoints require the `X-Bridge-Secret` header.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/instances` | List all instances |
| `POST` | `/v1/instances/{id}` | Create instance (Helm install) |
| `GET` | `/v1/instances/{id}` | Get instance + status |
| `PUT` | `/v1/instances/{id}` | Update instance (Helm upgrade) |
| `DELETE` | `/v1/instances/{id}` | Delete instance (Helm uninstall) |
| `GET` | `/v1/instances/{id}/status` | Detailed status |
| `GET` | `/v1/instances/{id}/health` | Health check |
| `GET` | `/healthz` | Bridge liveness |
| `GET` | `/readyz` | Bridge readiness |
| `GET` | `/metrics` | Prometheus metrics |

### Example: Create instance

```bash
curl -X POST http://bridge.hermeshq.net/v1/instances/tenant-001 \
  -H "X-Bridge-Secret: <secret>" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-001",
    "image": "nousresearch/hermes-agent",
    "imageTag": "latest",
    "namespace": "tenant-001",
    "ingressEnabled": true,
    "createNamespace": true,
    "resources": { "cpuRequest": "500m", "memoryRequest": "1Gi" },
    "storage": { "enabled": true, "size": "10Gi" },
    "network": { "subdomain": "tenant-001", "host": "hermeshq.net" }
  }'
```

See `BACKEND_API_GUIDE.md` for the full backend integration spec including plan tiers and config merging.

## Configuration

Bridge is configured via environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `BRIDGE_SECRET` | ✓ | Shared secret for API auth (`X-Bridge-Secret` header) |
| `BRIDGE_CLUSTER_NAME` | ✓ | Cluster identifier returned in health responses |
| `BRIDGE_CHART_PATH` | ✓ | Path to the hermes-agent Helm chart (mounted ConfigMap) |
| `BRIDGE_NAMESPACE` | ✓ | Default namespace for Helm operations |
| `BRIDGE_SYNC_INTERVAL` | | Workspace health sync interval (default: `5m`) |
| `BRIDGE_LISTEN_ADDRESS` | | HTTP listen address (default: `:8080`) |
| `BRIDGE_KUBECONFIG` | | Path to kubeconfig (defaults to in-cluster config) |
| `BRIDGE_RELEASE_PREFIX` | | Prefix added to all Helm release names |
| `BRIDGE_CREATE_NAMESPACE` | | Auto-create namespaces on install (default: `false`) |

## Local Development

```bash
# Build
go build -o /tmp/bridge ./bridge/

# Run unit tests
go test ./bridge/... -v

# Run locally against test cluster
BRIDGE_SECRET=dev-secret \
BRIDGE_CLUSTER_NAME=local \
BRIDGE_CHART_PATH=./charts/hermes-agent \
BRIDGE_NAMESPACE=default \
KUBECONFIG=./kubeconfig-test \
/tmp/bridge
```

## Deployment

The bridge runs in the `hermes-bridge` namespace on k3s. Manifests are in `deploy/`:

```
deploy/
  deployment-test.yaml   # Bridge Deployment
  service.yaml           # ClusterIP Service
  rbac.yaml              # ServiceAccount + ClusterRole
  secret.yaml            # bridge-auth Secret template
  ingress.yaml           # Traefik Ingress (bridge.hermeshq.net)
```

The `hermes-agent` chart is mounted into the bridge pod as a ConfigMap at `/charts/hermes-agent`.

### Manual deploy

```bash
# Build and load image directly into k3s (no registry needed)
docker build -t hermes-bridge:latest .
docker save hermes-bridge:latest | ssh root@178.104.185.60 "k3s ctr images import -"

# Sync chart ConfigMap
tar czf - charts/hermes-agent | ssh root@178.104.185.60 "cd /tmp && tar xzf -"
ssh root@178.104.185.60 "kubectl -n hermes-bridge create configmap hermes-agent-chart \
  --from-file=/tmp/charts/hermes-agent/ --dry-run=client -o yaml | kubectl apply -f -"

# Apply manifests and restart
kubectl apply -f deploy/ --kubeconfig kubeconfig-test
kubectl rollout restart deployment/hermes-bridge -n hermes-bridge --kubeconfig kubeconfig-test
```

## CI/CD

GitHub Actions workflow: `.github/workflows/bridge.yml`

| Phase | Trigger | Steps |
|-------|---------|-------|
| **Unit tests** | Every PR + push | `go test ./bridge/...`, `go vet` |
| **Deploy** | Push to `main` / `go-bridge` | Build image → SSH pipe to k3s → sync chart ConfigMap → apply manifests → rollout → smoke test |

No container registry is used — the image is piped directly from the CI runner into k3s via SSH.

**Required GitHub secret:** `HETZNER_SSH_PRIVATE_KEY` — the private key for `root@178.104.185.60`.

## Bridge code structure

```
bridge/
  main.go       Entry point — load config, build bridge, start HTTP server + sync loop
  bridge.go     HTTP router, auth middleware, Bridge struct, operation tracking
  handlers.go   HTTP request handlers (create/get/update/delete/status/health)
  helm.go       Helm client operations (install, upgrade, uninstall, list, buildValues)
  sync.go       Periodic sync loop — alerts on stuck/unhealthy instances
  status.go     Translates Helm release state → WorkspaceStatus
  config.go     Config struct, env var loading, validation
  types.go      InstanceSpec, WorkspaceStatus, Operation, ErrorResponse
  metrics.go    Prometheus metrics (workspace count, operation latency, results)
  bridge_test.go  Unit tests (21 tests — auth, handlers, config, buildValues, helpers)
```

## Workspace lifecycle

1. Backend POSTs `InstanceSpec` to `/v1/instances/{id}`
2. Bridge validates spec, spawns async operation
3. Helm installs `hermes-agent` chart into `{namespace}` with generated values
4. Pod starts, mounts PVC, bootstraps Hermes config from ConfigMap
5. Ingress routes `{workspace-id}.hermeshq.net` to the workspace service
6. Bridge sync loop monitors health every 5 minutes and logs alerts

Operations (create/update/delete) are async — bridge returns an operation object immediately. Poll `/v1/instances/{id}` to check when `phase: ready`.
