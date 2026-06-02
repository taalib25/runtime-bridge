---
name: hermescloud
description: Full project context for HermesCloud — the platform that runs AI coding instances for tenants. Covers bridge architecture, multi-cluster routing, CI/CD, networking, and backend contract. Use when starting any work on hermes-runtime-operator, the bridge service, instance lifecycle, multi-cluster, or when the user says "hermescloud context" or "load context".
---

# HermesCloud — Project Context

## What this is

HermesCloud runs isolated AI coding workspaces for tenants. Each workspace is a Kubernetes pod running `hermes-webui` + `hermes-agent` on a k3s cluster. The Go bridge manages pod lifecycle via Helm. The backend (hermes-client, separate repo) routes users to pods.

## Vocabulary rule — critical

**Always say `instance` / `instanceID`. Never say `workspace`.** Workspace is a backend concept. Bridge code, types, logs, comments: instances only.

## Repo layout

```
hermes-runtime-operator/
├── bridge/                  # Go bridge — the cluster agent
│   ├── bridge.go            # Router, auth middleware, Bridge struct, operation submit/supersede
│   ├── helm.go              # buildValues(), normalizeInstanceSpec(), instanceSpecFromRelease(), install/upgrade/delete
│   ├── operations.go        # OperationRunner — async op records (value-snapshot reads)
│   ├── labels.go            # Label/annotation contract: validate, derive identity, imperative labels
│   ├── admin_cluster.go     # GetClusterSummary, GetClusterResources, DrainCluster
│   ├── bridge_config.go     # Config struct, env var overlays, defaults
│   ├── types.go             # InstanceSpec, InstanceStatus, all request/response types
│   ├── handlers_*.go        # HTTP handlers (instance, admin, config, agents, integrations)
│   ├── lifecycle.go         # restart, redeploy, upgrade, rollback, repair, waitForDeploymentReady
│   ├── status.go            # GetInstanceStatus, health check, phase derivation, pod selection
│   ├── ingressroute.go      # Traefik IngressRoute + CORS/ForwardAuth middleware
│   ├── integrations.go      # Messaging platform enable/disable
│   ├── providers.go         # LLM provider config, soul write, instance Secret
│   ├── sync.go              # Background metrics/alert loop
│   └── exec.go              # WebSocket terminal
├── prod-hermes-docker-image/ # Runtime image (Dockerfile, extension JS/CSS)
│   ├── Dockerfile            # python:3.12-slim, hermes-webui cloned at build
│   └── extension/            # hermescloud.js + hermescloud.css (brand UX)
├── charts/runtime-node-core/ # Helm chart the bridge installs (active runtime)
├── docs/                     # Specs — see docs/README.md (label-contract, instance-lifecycle, …)
├── deploy/                   # Bridge k8s manifests
└── .github/workflows/
    ├── bridge.yml            # Tests + deploy hermes-test cluster
    ├── build-runtime.yml     # Builds runtime image → GHCR, restarts pods
    ├── _deploy-bridge.yml    # Reusable deploy workflow (called per cluster)
    └── deploy-cluster-template.yml  # Copy this to add a new cluster
```

## Bridge API (all routes require X-Bridge-Secret header)

```
POST   /v1/instances/:id          create instance
GET    /v1/instances              list all instances on this cluster
GET    /v1/instances/:id          get instance status
PUT    /v1/instances/:id          update instance spec
DELETE /v1/instances/:id          delete instance (+ namespace; ?purge=false soft-deletes)

# lifecycle ops — all async: return 202 + Operation, poll GET /v1/operations/:id
POST   /v1/instances/:id/restart  rolling restart
POST   /v1/instances/:id/redeploy re-run Helm upgrade from stored spec
POST   /v1/instances/:id/upgrade  upgrade image/tag (auto-rollback on health fail)
POST   /v1/instances/:id/rollback Helm rollback ({"version": N})
POST   /v1/instances/:id/repair   bounded auto-recovery from pod state
GET    /v1/instances/:id/operations  recent ops for this instance
GET    /v1/operations/:id         operation status (running|succeeded|failed|superseded)

# cluster
GET    /v1/cluster/summary        cluster load + health (used for routing)
GET    /v1/cluster/resources      per-node allocatable CPU/mem + pressure
GET    /v1/cluster/maintenance    maintenance-mode state
PUT    /v1/cluster/maintenance    set maintenance mode ({"enabled": bool}) — blocks new creates (503)
POST   /v1/cluster/drain          maintenance + delete all instances ({"purge": bool})

# diagnostics / proxied
GET    /v1/instances/:id/diagnostics  pod logs, events, recommendation
GET    /v1/instances/:id/profiles     hermes profile list (proxied)
GET    /api/providers                 configured LLM providers (proxied)
```

Operation model + delete-wins/supersede, repair table, maintenance/drain:
see `docs/instance-lifecycle.md` in the repo (indexed by `docs/README.md`).

## InstanceSpec — key fields backend must send

```json
{
  "instanceId":      "ws-<hex>",
  "runtimeMode":     "runtime-node-core",
  "runtimePort":     8787,
  "plan":            "free | pro | enterprise",
  "clusterId":       "hermes-test",
  "image":           "ghcr.io/taalib25/runtime-node-core",
  "imageTag":        "latest",
  "imagePullPolicy": "Always",
  "hermesConfig": {
    "model": { "provider": "openrouter", "name": "..." }
  },
  "secrets": { "OPENROUTER_API_KEY": "sk-..." },
  "network": {
    "subdomain": "ws-<hex>",
    "host":      "hermeshq.net",
    "scheme":    "https"
  },
  "commonLabels": {
    "hermescloud.dev/plan": "pro",
    "hermescloud.dev/workspace-id": "wsp_..."
  },
  "commonAnnotations": {
    "hermescloud.dev/display-name": "Ada's Workspace"
  }
}
```

`commonLabels`/`commonAnnotations` are **optional**, backend-owned business metadata
under `hermescloud.dev/*` only — the bridge validates (422 on violation), applies them
to every resource, and never invents or mutates them. See `docs/label-contract.md`.

## Cluster routing — how backend picks a cluster

See [REFERENCE.md](REFERENCE.md) for full algorithm. Short version:

1. Exclude: `kubernetesReachable=false`, `pods.crashLooping > 0`, node pressure
2. Filter by plan tier (enterprise → dedicated, free → shared)
3. Filter by region preference if user set one
4. Pick lowest `resources.reservedMemoryMiB` (most headroom)

Cache cluster stats in CF KV with 60s lazy TTL — never block routing on a live bridge call.

## Runtime image

- `ghcr.io/taalib25/runtime-node-core:latest`
- Built by `build-runtime.yml` on any change to `prod-hermes-docker-image/`
- Clones `nesquena/hermes-webui` (branch: **master**) at build time
- Installs `nousresearch/hermes-agent:latest` source into `/opt/hermes-agent`
- Extension loaded via `HERMES_WEBUI_EXTENSION_DIR=/opt/hermescloud-extension`
- User: hermeswebui (UID 1024), full sudo via `NOPASSWD: ALL` + CAP_SETUID/SETGID

## CI/CD

- Push to `main` → `bridge.yml` tests + deploys bridge to `hermes-test` cluster
- Change in `prod-hermes-docker-image/**` → `build-runtime.yml` builds image + rolls all pods
- New cluster → copy `deploy-cluster-template.yml`, create GitHub Environment with 5 secrets

## GitHub Environments (one per cluster)

Each environment holds: `NODE_IP`, `SSH_PRIVATE_KEY`, `BRIDGE_SECRET`, `GHCR_PAT`, `ADMIN_API_SECRET`

## Networking

- Cloudflare orange-cloud proxying `*.hermeshq.net`
- Traefik inside k3s handles TLS (ACME) + ingress per instance
- Bridge URL is **internal only** — backend stores it in Neon, never public-facing
- Instance URL: `https://ws-<hex>.hermeshq.net` (each gets its own subdomain)

## Current clusters

| Name | Node IP | Region | Status |
|---|---|---|---|
| hermes-test | 178.104.185.60 | eu | active |

## Key conventions

- Namespaces labelled `hermeshq/managed-by=bridge` — set by `ensureNamespace()`; the operational anchor for drain + cluster summary
- Release name = instance ID (no prefix); discovery selector = `app.kubernetes.io/instance=<releaseName>`
- One namespace per instance, auto-created
- Pod security: `allowPrivilegeEscalation: true`, `CAP_SETUID`, `CAP_SETGID`
- Default tag `latest` → `imagePullPolicy: Always` (set automatically in `buildValues()`)
- **Labels:** bridge owns identity (`app.kubernetes.io/*`) + operational anchors; backend owns `hermescloud.dev/*` business metadata. Control ops never key on backend labels. `app.kubernetes.io/managed-by` = `Helm` on chart resources, `hermes-bridge` on imperative ones. See `docs/label-contract.md`.
- **Operations:** mutating ops are async (202 + `Operation`, poll `/v1/operations/:id`); one-in-flight per instance (409 on conflict). **Delete always wins** — it supersedes (cancels) any in-flight op, which then reports `superseded`. See `docs/instance-lifecycle.md`.

## Open TODOs (do not implement without explicit instruction)

- S3 backup sidecar (rclone, `BackupS3Bucket` in InstanceSpec)
- Cross-cluster instance migration (blocked on PVC replication)
- ForwardAuth re-enable once `/api/client/auth/verify` is live on backend
