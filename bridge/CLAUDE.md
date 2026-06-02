# Bridge — AI Agent Guide

## Package structure

All bridge code lives in a single `package main` under `bridge/`. Files follow a naming convention:

| Pattern | Contents |
|---|---|
| `handlers_*.go` | HTTP handler functions (`func (b *Bridge) handleXxx(...)`) |
| `admin_*.go` | Business logic for admin/diagnostic endpoints (incl. `DrainCluster`) |
| `bridge.go` | `Bridge` struct, router (`Router()`), auth middleware, operation submit/supersede, `ensureNamespace` |
| `operations.go` | `OperationRunner` — async op records; returns value snapshots so reads never race the worker |
| `labels.go` | Label/annotation contract: `validateBackendMetadata`, `deriveSelectorLabels`, `instanceLabels`/`instanceAnnotations` |
| `types.go` | All request/response types (`InstanceSpec`, `InstanceStatus`) |
| `lifecycle.go` | Instance lifecycle ops (restart, redeploy, upgrade, rollback, repair, waitForDeploymentReady) |
| `providers.go` | LLM provider config, soul write, model update, instance Secret |
| `integrations.go` | Messaging platform enable/disable |
| `ingressroute.go` | Traefik IngressRoute + CORS/ForwardAuth middleware |
| `helm.go` | Helm install/upgrade/delete wrappers, `buildValues`, `instanceSpecFromRelease` |
| `status.go` | `GetInstanceStatus`, health check, phase derivation, pod selection |
| `sync.go` | Background metrics/alert loop |
| `exec.go` | WebSocket exec (terminal), `findExecPod`, `podRunCommand` |

## Vocabulary rule

The bridge knows only **instances** — never "workspaces". Workspaces are a backend concept. All bridge code, types, logs, and comments must use `instance` / `instanceID`.

## Admin endpoints

Business logic lives in `bridge/admin_*.go`. HTTP handlers are in `bridge/handlers_admin.go`. Routes are grouped under the `// Admin / diagnostics` comment block in `bridge.go:Router()`.

**Rule: never call these from the frontend. Flow is: Admin UI → Backend Admin API → Bridge.**

### Cluster-level (no instance ID)

| Endpoint | File | Description |
|---|---|---|
| `GET /v1/cluster/summary` | `admin_cluster.go` | Pod counts (running/pending/failed/crashLooping), instance count, node count, k8s reachability, `maintenance`/`draining` flags |
| `GET /v1/cluster/resources` | `admin_cluster.go` | Node info: ready status, allocatable CPU/memory, pressure conditions |
| `GET`/`PUT /v1/cluster/maintenance` | `handlers_admin.go` | Get/set maintenance mode; when on, `create` returns 503 |
| `POST /v1/cluster/drain` | `admin_cluster.go` | Enable maintenance, then delete all managed instances (`{"purge": bool}`); 409 if already draining |

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

## Label & annotation contract (`labels.go`)

Full spec: `docs/label-contract.md`. Rules that bind bridge code:

- **Backend owns** `commonLabels` / `commonAnnotations` under `hermescloud.dev/*` only.
  `validateBackendMetadata` rejects any other prefix or oversize label value → **422**.
  Never mutate backend values — reject instead.
- **Bridge owns** identity: `deriveSelectorLabels(releaseName)` = `app.kubernetes.io/name`
  + `app.kubernetes.io/instance`. Never put business metadata in selector labels.
- `app.kubernetes.io/managed-by` = `Helm` on chart-rendered resources, `hermes-bridge`
  on imperative ones (namespace, IngressRoute, instance Secret) via `instanceLabels`.
- Persist `commonLabels`/`commonAnnotations` in the spec so `instanceSpecFromRelease`
  round-trips them — repair/redeploy/upgrade must not strip them.
- **Control ops select on bridge-owned anchors only** (`app.kubernetes.io/instance`,
  namespace `hermeshq/managed-by=bridge`) — never on backend labels.

## Operations & lifecycle (`operations.go`, `bridge.go`, `lifecycle.go`)

- Mutating calls are async: handler submits, returns `202` + an `Operation` snapshot,
  worker goroutine runs it. Backend polls `GET /v1/operations/:id`.
- `OperationRunner.Get`/`ListForInstance`/`Submit` return **value snapshots** under
  lock — never hand a live `*Operation` to a reader (it races the worker).
- `submitInstanceOperation` enforces one-in-flight per instance (409). Each op holds a
  cancellable context in `pendingOps`.
- **Delete wins:** `handleDeleteInstance` calls `supersedeInFlight` to cancel any
  running op before deleting; the cancelled op reports `superseded`, not `failed`.
- `upgrade` auto-rolls-back on health failure under a *fresh* context; skips rollback
  if superseded. Full model: `docs/instance-lifecycle.md`.

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
