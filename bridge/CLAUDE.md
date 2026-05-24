# Bridge — AI Agent Guide

## Package structure

All bridge code lives in a single `package main` under `bridge/`. Files follow a naming convention:

| Pattern | Contents |
|---|---|
| `handlers_*.go` | HTTP handler functions (`func (b *Bridge) handleXxx(...)`) |
| `admin_*.go` | Business logic for admin/diagnostic endpoints |
| `bridge.go` | `Bridge` struct, router (`Router()`), auth middleware, helpers |
| `types.go` | All request/response types |
| `lifecycle.go` | Instance lifecycle ops (repair, events, waitForDeploymentReady) |
| `providers.go` | LLM provider config, soul write, model update, SetInstanceConfig |
| `integrations.go` | Messaging platform enable/disable |
| `helm.go` | Helm install/upgrade/delete wrappers |
| `status.go` | `GetInstanceStatus`, health check, phase derivation |
| `exec.go` | WebSocket exec (terminal), `findExecPod`, `podRunCommand` |

## Vocabulary rule

The bridge knows only **instances** — never "workspaces". Workspaces are a backend concept. All bridge code, types, logs, and comments must use `instance` / `instanceID`.

## Admin endpoints

Business logic lives in `bridge/admin_*.go`. HTTP handlers are in `bridge/handlers_admin.go`. Routes are grouped under the `// Admin / diagnostics` comment block in `bridge.go:Router()`.

**Rule: never call these from the frontend. Flow is: Admin UI → Backend Admin API → Bridge.**

### Cluster-level (no instance ID)

| Endpoint | File | Description |
|---|---|---|
| `GET /v1/cluster/summary` | `admin_cluster.go` | Pod counts (running/pending/failed/crashLooping), instance count, node count, k8s reachability |
| `GET /v1/cluster/resources` | `admin_cluster.go` | Node info: ready status, allocatable CPU/memory, pressure conditions |

Cluster summary scans all namespaces labelled `hermeshq/managed-by=bridge`. `ensureNamespace` sets this label at instance creation — do not remove it.

CrashLooping definition: `WaitingReason == "CrashLoopBackOff"` OR `restartCount > 5`.

### Instance-level

| Endpoint | File | Description |
|---|---|---|
| `GET /v1/instances/{id}/diagnostics` | `admin_diagnostics.go` | Aggregated view: status, logs, PVC, service, events, recommendation |
| `GET /v1/instances/{id}/logs` | `admin_diagnostics.go` | Pod logs (`?tail=200&previous=false&container=`) |
| `GET /v1/instances/{id}/resources` | `admin_resources.go` | k8s inventory: pods, deployments, services, PVCs, secrets metadata (key names only) |

**Adding a new admin endpoint:**
1. Business logic → `admin_*.go`
2. Handler → `handlers_admin.go`
3. Route → `bridge.go` under `// Admin / diagnostics` block
4. Types → `types.go`

## Security rules

- **Never return Secret data values.** `SecretMeta` lists key names only.
- All `/v1/` routes require `X-Bridge-Secret` header (enforced by `authMiddleware`).
- Workspace lookup always goes through `lookupRelease` — never assume `workspaceID == namespace`.

## Key patterns

```go
// Workspace namespace + release name — always via lookupRelease
rel, err := b.lookupRelease(ctx, workspaceID)
ns := rel.Namespace
releaseName := rel.Name

// Label selector for workspace resources
selector := labels.Set{"app.kubernetes.io/instance": releaseName}.AsSelector().String()

// Find running pod
pod, err := b.findExecPod(ctx, ns)  // takes namespace string

// Admin read timeout
ctx, cancel := adminContextTimeout(r.Context())
defer cancel()
```
