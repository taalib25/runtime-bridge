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
│   ├── bridge.go            # Router, auth middleware, Bridge struct
│   ├── helm.go              # buildValues(), normalizeInstanceSpec(), Helm install/upgrade/delete
│   ├── admin_cluster.go     # GetClusterSummary (load metrics), GetClusterResources
│   ├── bridge_config.go     # Config struct, env var overlays, defaults
│   ├── types.go             # InstanceSpec, all request/response types
│   ├── handlers_*.go        # HTTP handlers
│   ├── lifecycle.go         # repair, waitForDeploymentReady
│   ├── providers.go         # LLM provider config, soul write
│   └── exec.go              # WebSocket terminal
├── prod-hermes-docker-image/ # Runtime image (Dockerfile, extension JS/CSS)
│   ├── Dockerfile            # python:3.12-slim, hermes-webui cloned at build
│   └── extension/            # hermescloud.js + hermescloud.css (brand UX)
├── charts/hermes-instance/   # Helm chart for pod deployment
├── deploy/                   # Bridge k8s manifests
└── .github/workflows/
    ├── bridge.yml            # Tests + deploy hermes-test cluster
    ├── build-runtime.yml     # Builds runtime image → GHCR, restarts pods
    ├── _deploy-bridge.yml    # Reusable deploy workflow (called per cluster)
    └── deploy-cluster-template.yml  # Copy this to add a new cluster
```

## Bridge API (all routes require X-Bridge-Secret header)

```
POST   /v1/instances              create instance
GET    /v1/instances              list all instances on this cluster
GET    /v1/instances/:id          get instance status
PUT    /v1/instances/:id          update instance spec
DELETE /v1/instances/:id          delete instance + namespace
GET    /v1/cluster/summary        cluster load + health (used for routing)
GET    /v1/cluster/resources      per-node allocatable CPU/mem + pressure
GET    /v1/instances/:id/diagnostics  pod logs, events, recommendation
GET    /v1/instances/:id/profiles     hermes profile list (proxied)
GET    /api/providers                 configured LLM providers (proxied)
```

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
  }
}
```

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

- Namespaces labelled `hermeshq/managed-by=bridge` — set by `ensureNamespace()` on every Helm op
- Release name = instance ID (no prefix)
- One namespace per instance, auto-created
- Pod security: `allowPrivilegeEscalation: true`, `CAP_SETUID`, `CAP_SETGID`
- Default tag `latest` → `imagePullPolicy: Always` (set automatically in `buildValues()`)

## Open TODOs (do not implement without explicit instruction)

- S3 backup sidecar (rclone, `BackupS3Bucket` in InstanceSpec)
- Cross-cluster instance migration (blocked on PVC replication)
- ForwardAuth re-enable once `/api/client/auth/verify` is live on backend
