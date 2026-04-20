# Hermes Go Bridge - Rebuild Plan

## Executive Summary

Migrating from Operator-based architecture to Helm-driven Go Bridge.

### Target Architecture

```
Hono (Cloudflare Workers) → Go Bridge (per cluster) → Helm SDK → hetzner-k3s Cluster
```

**Key changes:**
- Remove: CRD, Controller, Operator SDK dependencies
- Keep: HTTP API patterns, workspace management concepts
- Add: Helm SDK integration, direct k8s status queries

---

## Phase 1: Prerequisites (COMPLETED)

- ✅ hetzner-k3s CLI installed (`/home/taalib/bin/hetzner-k3s`)
- ⚠️ Hetzner API token needed (user must provide)
- ✅ Helm v3.20.2 installed
- ✅ Git repo: hermes-runtime-operator
- ✅ New branch: `go-bridge`

---

## Phase 2: Project Restructure

### Files to Remove (Operator Components)

```
api/v1alpha1/runtime_types.go          # CRD schema
api/v1alpha1/runtimegroup_types.go     # (if exists)
config/crd/bases/*.yaml                 # CRD manifests
config/rbac/role.yaml                   # Generated RBAC
config/webhook/                         # Webhook configs
internal/controller/                    # Controller logic
test/e2e/                               # Operator e2e tests
PROJECT                                 # Kubebuilder metadata
```

### Files to Keep/Modify

```
internal/api/server.go                  # → repurpose for bridge HTTP API
go.mod                                  # → update dependencies (remove controller-runtime)
go.sum                                  # → update
Dockerfile                              # → modify for bridge
Makefile                                # → simplify
```

### New Files to Create

```
bridge/
├── main.go              # Entry point
├── bridge.go            # Bridge struct + core operations
├── helm.go              # Helm SDK integration
├── status.go            # Workspace status tracking
├── handlers.go          # HTTP handlers
├── config.go            # Configuration loading
├── types.go             # Workspace types
├── sync.go              # Periodic sync loop
├── metrics.go           # Prometheus metrics
└── health.go            # Health endpoints
charts/
└── hermes-agent/        # Vendored Helm chart (direct mode)
deploy/
├── deployment.yaml      # Bridge deployment
├── service.yaml         # Bridge service  
├── ingress.yaml         # Bridge ingress
├── secret.yaml          # Bridge auth secret
└── configmap.yaml       # Bridge config
```

---

## Phase 3: Go Bridge Implementation

### Core Dependencies

```go
// go.mod
require (
  helm.sh/helm/v3 v3.20.2
  k8s.io/client-go v0.32.0
  k8s.io/api v0.32.0
  k8s.io/apimachinery v0.32.0
  github.com/gorilla/mux v1.8.1
  github.com/prometheus/client_golang v1.20.0
)
```

### HTTP API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/workspaces` | GET | List all workspaces |
| `/v1/workspaces/{id}` | POST | Create workspace |
| `/v1/workspaces/{id}` | GET | Get workspace details |
| `/v1/workspaces/{id}` | PUT | Update workspace |
| `/v1/workspaces/{id}` | DELETE | Delete workspace |
| `/v1/workspaces/{id}/status` | GET | Get workspace status |
| `/v1/workspaces/{id}/health` | GET | Health check |
| `/v1/workspaces/{id}/secrets` | PUT | Update secrets |
| `/v1/workspaces/{id}/actions/{action}` | POST | Suspend/resume/restart |
| `/v1/clusters` | GET | List clusters |
| `/healthz` | GET | Bridge health |
| `/readyz` | GET | Bridge ready |

### Workspace Spec (replaces Runtime CR)

```go
type WorkspaceSpec struct {
  WorkspaceID   string            `json:"workspaceId"`
  TenantID      string            `json:"tenantId"`
  ClusterID     string            `json:"clusterId"`  // Multi-cluster routing
  Image         string            `json:"image"`
  Resources     ResourceSpec      `json:"resources"`
  Storage       StorageSpec       `json:"storage"`
  Network       NetworkSpec       `json:"network"`
  Env           []EnvVar          `json:"env"`
  Secrets       map[string]string `json:"secrets,omitempty"`
  Config        map[string]any    `json:"config,omitempty"`
}

type WorkspaceStatus struct {
  WorkspaceID   string    `json:"workspaceId"`
  Phase         string    `json:"phase"`      // creating, running, failed, deleting
  Ready         int32     `json:"ready"`
  Replicas      int32     `json:"replicas"`
  Healthy       bool      `json:"healthy"`
  URL           string    `json:"url"`
  CreatedAt     time.Time `json:"createdAt"`
  ClusterID     string    `json:"clusterId"`
  ReleaseName   string    `json:"releaseName"`
  Namespace     string    `json:"namespace"`
}
```

---

## Phase 4: Helm Chart Integration

### Use Existing Chart

- Source: `ultraworkers/hermes-agent-helm-chart`
- Mode: Direct deployment only (ignore operator-ready features)
- Location: Vendor into `charts/hermes-agent/`

### Values Mapping

| Workspace Spec Field | Helm Values Path |
|---------------------|------------------|
| `workspaceId` | `fullnameOverride` |
| `tenantId` | `tenant.id` |
| `image` | `image.repository` + `image.tag` |
| `resources.cpu` | `resources.requests.cpu` |
| `resources.memory` | `resources.requests.memory` |
| `storage.size` | `persistence.size` |
| `storage.storageClass` | `persistence.storageClass` |
| `network.host` | `ingress.hosts[0].host` |
| `network.ingressClassName` | `ingress.className` |
| `secrets.*` | `secrets.*` or `secrets.existingSecret` |
| `env` | `extraEnv` |

---

## Phase 5: Cluster Configuration

### hetzner-k3s Config Template

```yaml
# cluster-config.yaml
hetzner_token: <user-provides>
cluster_name: hermes-production
kubeconfig_path: "./kubeconfig"
k3s_version: v1.32.0+k3s1

networking:
  ssh:
    public_key_path: "~/.ssh/id_ed25519.pub"
    private_key_path: "~/.ssh/id_ed25519"
  private_network:
    enabled: true
    subnet: 10.0.0.0/16

masters_pool:
  instance_type: cpx22
  instance_count: 3
  locations:
    - fsn1
    - hel1
    - nbg1

worker_node_pools:
  - name: hermes-workers
    instance_type: cpx32
    instance_count: 3
    location: fsn1
    autoscaling:
      enabled: true
      min_instances: 2
      max_instances: 20

addons:
  csi_driver:
    enabled: true   # Hetzner block storage
  cloud_controller_manager:
    enabled: true   # Auto LoadBalancer
  cluster_autoscaler:
    enabled: true
  system_upgrade_controller:
    enabled: true

create_load_balancer_for_the_kubernetes_api: true
```

---

## Phase 6: Production Checklist

- [ ] HA cluster running with Traefik ingress, TLS (cert-manager)
- [ ] Hetzner CSI configured for PVCs (`storageClass: hcloud`)
- [ ] Chart and Hermes image pinned to exact versions
- [ ] Single replica + `Recreate` strategy enforced
- [ ] Bridge has auth (`X-Bridge-Secret`), `/healthz` + `/readyz`
- [ ] Async operation tracking for long-running Helm actions
- [ ] Prometheus metrics: latency, success/failure, workspace count
- [ ] Alerts: failed installs, stuck operations, PVC pressure
- [ ] Backup/restore tested (Velero or manual)
- [ ] One pilot workspace created, verified, rollback tested

---

## Migration Path

1. Build Go Bridge on new branch
2. Set up hetzner-k3s cluster (fresh, not migration)
3. Deploy Go Bridge to cluster
4. Create test workspace via Bridge API
5. Verify health, status, deletion
6. Configure Hono to call Bridge
7. Production rollout

---

## Effort Estimate

| Phase | Time |
|-------|------|
| Phase 2 (Restructure) | 1 hour |
| Phase 3 (Bridge code) | 3-4 hours |
| Phase 4 (Chart vendor) | 30 min |
| Phase 5 (Cluster config) | 30 min |
| Phase 6 (Docs/checklist) | 1 hour |
| **Total** | **6-7 hours** |

---

## Dependencies

### User Must Provide

- Hetzner Cloud API token (read & write)
- SSH key pair for cluster access
- Domain DNS configuration (hermeshq.net)

### Pre-installed

- ✅ Helm v3.20.2
- ✅ kubectl
- ✅ hetzner-k3s CLI
- ✅ Go 1.21+

---

## Files to Generate

This plan will be executed by creating:

1. `bridge/main.go` - Entry point
2. `bridge/bridge.go` - Core struct
3. `bridge/helm.go` - Helm SDK operations
4. `bridge/status.go` - Status tracking
5. `bridge/handlers.go` - HTTP handlers
6. `bridge/types.go` - Types definitions
7. `charts/hermes-agent/` - Vendored chart
8. `deploy/*.yaml` - Deployment manifests
9. `cluster-config.yaml` - hetzner-k3s template
10. Updated `go.mod`, `Dockerfile`, `Makefile`

---

## Status

- Phase 1: ✅ COMPLETED
- Phase 2: 🔄 IN PROGRESS
- Phase 3-6: ⏳ PENDING