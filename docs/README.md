# HermesCloud / Hermes Runtime Operator — Docs Index

Project context for agents lives in the **`hermescloud` skill** (`SKILL.md` +
`REFERENCE.md`); load it with "hermescloud context". These `docs/` files are the
durable specs the skill points into.

## Specs

| Doc | What it covers |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | What the bridge owns / doesn't own; control flow; runtime model |
| [RUNTIME_MODEL.md](RUNTIME_MODEL.md) | The runtime image + per-instance resource model |
| [instance-lifecycle.md](instance-lifecycle.md) | Operation model (async 202 + poll), create/update/delete, restart/redeploy/rollback/upgrade/repair, maintenance + drain |
| [label-contract.md](label-contract.md) | Backend↔bridge label/annotation contract (`commonLabels`/`commonAnnotations`, `hermescloud.dev/*`, bridge-derived identity) |
| [backend-integration-changes.md](backend-integration-changes.md) | **What the backend (hermes-client) must change** to stay compatible: labels, operation model, status fields, maintenance/drain, observability |
| [multi-cluster-architecture.md](multi-cluster-architecture.md) | Multi-cluster routing, registration, cluster lifecycle |
| [auth-strategy.md](auth-strategy.md) | Auth model (X-Bridge-Secret, ForwardAuth roadmap) |
| [security-hardening.md](security-hardening.md) | Pod security, isolation, hardening posture |
| [scaling-plan.md](scaling-plan.md) | Capacity / scaling approach |
| [infrastructure-overview.md](infrastructure-overview.md) | Hetzner / k3s / networking infra |

## Agent guides

| File | What it covers |
|---|---|
| [CODING_AGENT_RULES.md](CODING_AGENT_RULES.md) | Rules for agents editing this repo |
| [LEGACY_PATHS.md](LEGACY_PATHS.md) | Deprecated paths to avoid |
| [../bridge/CLAUDE.md](../bridge/CLAUDE.md) | Bridge package structure, endpoints, conventions |

## Source of truth

When code and docs disagree, code wins — but fix the doc. Key code anchors:

- API routes: `bridge/bridge.go` → `Router()`
- Request/response types: `bridge/types.go`
- Lifecycle ops: `bridge/lifecycle.go`, operation model `bridge/operations.go` + `bridge/bridge.go`
- Label contract: `bridge/labels.go` (+ chart `charts/runtime-node-core/templates/_helpers.tpl`)
