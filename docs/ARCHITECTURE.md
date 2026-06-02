# Hermes Runtime Operator — Architecture

## Summary

The bridge is a Go control service that runs inside a K3s/Kubernetes cluster.
It manages Hermes runtime instances using the Kubernetes API and Helm SDK.

Kubernetes controllers/operators are designed as control loops that watch cluster
state and make changes to move actual state toward desired state. The bridge is a
lightweight operator that does exactly this for Hermes runtime instances.

## Control Flow

```
Backend API
  └─► Bridge (Go service, hermes-bridge namespace)
        ├─► Kubernetes API  (namespaces, secrets, deployments, events)
        ├─► Helm SDK        (install / upgrade / uninstall runtime chart)
        └─► Traefik CRDs   (IngressRoute, Middleware for per-instance routing)
```

## Core Responsibilities

The bridge owns:

| Area | Detail |
|------|--------|
| Runtime lifecycle | Create, update, delete Hermes runtime instances via Helm (async ops, poll by ID; delete supersedes in-flight ops) |
| Runtime operations | Restart, redeploy, rollback, repair, upgrade (auto-rollback on health fail) |
| Cluster operations | Maintenance mode (blocks new creates) and drain (delete all instances) |
| Resource metadata | Validate + apply backend `hermescloud.dev/*` labels/annotations; own k8s identity labels |
| Secret management | Provider API keys patched directly into workspace k8s Secret |
| Messaging integrations | Platform token secrets + env var flags (Telegram, Discord, Slack, WhatsApp, Signal, DingTalk, Feishu, WeCom, BlueBubbles) |
| Provider config | LLM provider key registration + model/base-URL config via Helm upgrade |
| Agent templates | Named config + SOUL.md snapshots stored as k8s ConfigMaps |
| Status + health | Pod readiness, deployment rollout state, workspace event collection |
| Ingress routing | Traefik IngressRoute + CORS/ForwardAuth middleware per workspace |
| Terminal exec | WebSocket → kubectl SPDY PTY proxy for in-browser terminal |

The bridge does **not** own:

- Billing and subscription enforcement
- User authentication (it validates the backend's `X-Bridge-Secret` only)
- Product chat session storage
- Long-term product database
- Frontend state
- Global multi-cluster placement decisions
- Hetzner server provisioning (`hetzner-k3s` CLI is used manually)

## Runtime Model

A Hermes runtime instance is a Helm release of the canonical runtime chart installed
into an isolated Kubernetes namespace. Each namespace contains:

```
Deployment   — one pod running the Hermes runtime
PVC          — persistent storage for HERMES_HOME (config, sessions, memories)
Secret       — bridge-owned API keys and integration tokens
Service      — ClusterIP for health checks and internal routing
IngressRoute — Traefik CRD for public HTTPS access (workspace.hermeshq.net)
```

The bridge writes Secrets directly via the Kubernetes API (never through Helm), then
does a Helm upgrade so the Deployment's `extraSecretKeys` references are updated.
This ensures the pod gets the correct secretKeyRef env vars on next rollout.

## Canonical Runtime Chart

The chart the bridge installs is:

```
charts/runtime-node-core/
```

See `charts/README.md` for chart selection rationale.

## Plan Tiers

| Plan | Isolation |
|------|-----------|
| Standard | Shared K3s node, isolated namespace |
| Team | Shared cluster, dedicated namespace, shared PVC with Hermes profiles |
| Enterprise | Dedicated namespace, node pool, or full K3s cluster |

The default internal unit is an isolated Hermes instance, not a raw Kubernetes pod
exposed directly to customers.

## Related specs

- [instance-lifecycle.md](instance-lifecycle.md) — operation model, recovery ops, maintenance/drain
- [label-contract.md](label-contract.md) — backend↔bridge label/annotation contract
- [README.md](README.md) — full docs index
