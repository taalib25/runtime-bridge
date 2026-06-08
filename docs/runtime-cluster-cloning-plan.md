# Runtime Cluster Cloning Plan — Source of Record

Status: **approved design**, phased implementation in progress.
Owner: runtime-operator. Last updated: 2026-06-09.

## Goal

Make the runtime cluster stack **cloneable and repeatable** on Hetzner k3s: stand up a
second (third, Nth) runtime cluster with the *same* bridge, Helm-chart behavior, cleanup
system, validation checks, and backend registration — **without manually SSH-ing in or
copying scripts**. First target: two runtime clusters, `runtime-eu-1` and `runtime-eu-2`.

Out of scope for this effort: cross-cluster migration, teams/canvas/agent-mesh.

## What already exists (do not rebuild)

- `.claude/skills/hetzner-k3s/scripts/provision-cluster.sh` — one-command cloner:
  `hetzner-k3s create` → discover node IP → **auto-create GitHub Environment + 5 secrets**
  → GHCR auth on node → build+push bridge → apply `rbac/service/deployment` + rollout →
  smoke test → register with backend.
- `.github/workflows/_deploy-bridge.yml` — reusable, parameterized deploy workflow.
- `.github/workflows/deploy-cluster-template.yml` — per-cluster caller using GitHub Environments.
- `cluster-config.yaml` (+ `-test`) — source-controlled hetzner-k3s config.
- Bridge endpoints already expose the data we need: `GET /v1/cluster/summary`
  (`headroomMiB`, pod states, instanceCount, reserved cpu/mem, `maintenance`, `draining`),
  `GET /v1/cluster/resources` (per-node allocatable), `/v1/instances/{id}/diagnostics`,
  `/v1/operations/{id}`.
- The Hermes Helm chart is **baked into the bridge image** (`Dockerfile` COPYs
  `charts/hermes-agent`; `BRIDGE_CHART_PATH=/charts/hermes-agent`).

The work is **closing 6 gaps**, not building a stack.

## The 6 gaps

1. Cleanup is not source-controlled and not installed by provisioning (node-only), and
   misses dangling `<none>` image records (runs `ctr content gc` but never `crictl rmi`).
2. No per-cluster configs for `runtime-eu-1` / `runtime-eu-2`.
3. Validation is shallow (`health-check.sh`): no RBAC self-check, helm dry-run,
   StorageClass/PVC bind, cleanup-present, or NetworkPolicy-egress checks; no bridge self-check.
4. Deploys to N clusters still need per-cluster workflow-file copies (no matrix).
5. DNS is manual; `provision-cluster.sh` prints "add the Cloudflare record yourself."
6. Backend is registered with `http://NODE_IP:8080` — bridge secret in **cleartext**, no TLS.

## Decisions (Q1–Q7)

| # | Decision | Rationale |
|---|---|---|
| **Q1** | **Extend** the existing layout; **no `infra/` reorg** | `provision-cluster.sh` + `deploy/` + root configs + skill scripts already form a working cloneable system; a reorg is path churn with rollback risk for cosmetic gain. |
| **Q2** | Cleanup = **host systemd timer**, not a privileged DaemonSet | Cleanup needs host/containerd access; keep it entirely out of Kubernetes — the bridge stays a constrained control-plane pod. Simplest safe V1. |
| **Q3** | Install via **`additional_post_k3s_commands`**, with the cleanup script/units **rendered into each generated cluster config** from one source-of-truth `.sh` | `additional_post_k3s_commands` runs on **every** node incl. autoscaled ones (pool config applies to all). No boot-time GitHub fetch, no tokens on nodes, no SSH-installed second copy, no drift. |
| **Q4** | **Both** checks: external `validate-cluster.sh` **and** in-bridge startup **SSAR** → `/readyz: degraded` | They cover different layers. The in-bridge check makes the bridge fail-fast and self-evident (gates rollout, kills the "stuck creating" class at the deepest point); the script covers cluster prerequisites the bridge can't self-check (storage, egress, cleanup-present) + exercises live endpoints. |
| **Q5** | CI **build-once → matrix deploy-many**, `fail-fast: false`, immutable `:${sha}`; hermes-test split from the prod matrix | One image built/pushed once, deployed to every cluster; adding a cluster = one matrix line (no workflow-file copy); one cluster's failure doesn't cancel the others. |
| **Q6** | **HTTPS bridge addressing**: parameterized ingress host + Cloudflare DNS automation + register `https://bridge-<cluster>.hermeshq.net` | Closes the cleartext-secret hole and makes the bridge uniformly addressable with edge TLS. |
| **Q7** | **1 master (cpx22) + autoscaling cpx32 workers** per runtime cluster; workloads off masters; `protect_against_deletion` | Cheapest shape that still has a real autoscaling worker pool (everything in capacity/cleanup/scaling depends on it). Redundancy lives at the **fleet** layer (eu-1 + eu-2), not inside each cluster. Graduate to 3-master HA under load. |

## Constraints (binding on implementation)

- Do **not** integrate host cleanup execution into the bridge pod.
- Keep bridge RBAC minimal; the self-check is read-only SSAR and adds **no** new grants.
- One Hermes agent = one Helm release = one pod = one PVC.
- No boot-time GitHub fetch; no GitHub tokens on runtime nodes.
- No privileged DaemonSet for V1.
- No broad repo reorganization in this phase.

## Files to add / change

**Cleanup (P1)** — `.claude/skills/hetzner-k3s/scripts/`:
- `hermescloud-cleanup.sh` — dry-run support; `crictl rmi --prune` to clear dangling
  `<none>` records; skip images used by running containers; keep last N `:${sha}` bridge
  images for rollback; cap journald; optional apt clean; write `/var/log/hermescloud-cleanup.status`.
- `hermescloud-cleanup.service`, `hermescloud-cleanup.timer` (nightly).
- `render-cluster-config.sh` — embeds the three into `additional_post_k3s_commands`;
  deterministic output with a cleanup-script hash comment.
- `validate-cleanup.sh` — binary present, timer enabled, checksum matches rendered expected,
  status file writable, dry-run works.

**Cluster configs (P5 inputs)** — repo root (or `clusters/`):
- `cluster-config-eu1.yaml`, `cluster-config-eu2.yaml` — generated (1 master cpx22,
  autoscaling cpx32 workers, `protect_against_deletion`, `schedule_workloads_on_masters: false`).

**Validation (P3)** — `.claude/skills/hetzner-k3s/scripts/validate-cluster.sh` (extends
`health-check.sh`): nodes ready; bridge `/healthz`+`/readyz`; `/v1/cluster/summary` +
`/v1/cluster/resources` sane; Hetzner CSI StorageClass exists; **test PVC binds**; cleanup
timer on every node; NetworkPolicy egress blocks `169.254.169.254` and allows LLM APIs; DNS resolves.

**Bridge (P2)** — Go: startup `SelfSubjectAccessReview` over every kind the chart creates
(`namespaces, secrets, configmaps, services, deployments, persistentvolumeclaims, ingresses,
ingressroutes, networkpolicies, resourcequotas, limitranges`); on any miss → internal
readiness `degraded`, `/readyz` fails/degrades per existing convention and lists the missing
permissions (machine-readable). No new RBAC.

**CI (P4)** — split `_deploy-bridge.yml` into `build` (push `:${sha}`, output digest) +
`deploy` (matrix, `fail-fast: false`, `environment: ${{ matrix.cluster }}`, applies
rbac/service/**ingress**/deployment, rollout, `validate-cluster.sh`). `bridge.yml` matrix =
`[hermes-eu-1, hermes-eu-2]`; hermes-test → separate staging workflow.

**Addressing (P3)** — parameterize `deploy/ingress.yaml` host →
`bridge-${CLUSTER_NAME}.hermeshq.net`; `provision-cluster.sh` applies the ingress, creates the
Cloudflare A record (`CF_TOKEN`/`CF_ZONE_ID`), waits for the live HTTPS endpoint, registers the
HTTPS URL, runs `validate-cluster.sh`, and **fails if cleanup is missing on any node**.

## Rollout — phased, one PR per phase

| Phase / PR | Scope | Depends on |
|---|---|---|
| **PR 0** | This document | — |
| **PR 1 (P1)** | Cleanup script + units + renderer + `validate-cleanup.sh` + checksum. No bridge/CI changes, no provisioning. | — |
| **PR 2 (P2)** | Bridge startup SSAR self-check → `/readyz: degraded` with missing-perm output. | — |
| **PR 3 (P3)** | `validate-cluster.sh`; parameterized bridge ingress; `provision-cluster.sh` applies ingress + Cloudflare DNS + waits for HTTPS + registers HTTPS URL. | P2 (readyz gating) |
| **PR 4 (P4)** | CI build-once / matrix deploy-many; immutable `:${sha}`; hermes-test separated. | P3 (validate step) |
| **PR 5 (P5)** | Generate + provision `runtime-eu-1`, validate, register; then `runtime-eu-2`. | P1–P4 |

Adding a future cluster after P1–P4: `provision-cluster.sh <config> hermes-eu-3 eu` +
one matrix line. No workflow-file copy, no manual SSH.

## Backend coordination (contract; backend lives in a separate repo)

- Registry per cluster: `cluster_id`, `bridge_url` (HTTPS), `runtime_base_domain`,
  `status` (active/draining/maintenance/disabled/unhealthy), last heartbeat/summary.
- Backend calls `GET /v1/cluster/summary` before creating an agent; if `headroomMiB` is
  insufficient or `/readyz` is `degraded`, it must **not** route there — try another cluster
  or return a clean `capacity_unavailable`. No agent should sit "creating" forever.
- A bridge reporting `/readyz: degraded` (failed SSAR) → mark that cluster
  unhealthy/maintenance-blocked; surface the specific missing permission in the admin UI
  (e.g. "Bridge missing permission: create networkpolicies"), not a generic failure.

## Risks → mitigations

- **DNS/cert not ready before backend register** → P3 waits on the live HTTPS endpoint
  (`/healthz`, `/readyz`, `/v1/cluster/summary`) before the register call.
- **Matrix rebuilds / `:latest` drift** → build-once job; deploy immutable `:${sha}`.
- **Single-master SPOF strands a cluster's tenants** (migration deferred) → documented;
  fleet-level redundancy across eu-1/eu-2; graduate to 3-master HA under load.
- **Renderer drift** between repo script and embedded copy → cleanup-hash comment in
  generated configs + `validate-cleanup` checksum check.
- **`crictl rmi --prune` aggressiveness** → keeps images used by running containers; the
  only collateral (rollback bridge image, idle agent image) re-pulls from the registry.

## Rollback notes (per phase)

- **PR 1**: cleanup is additive and node-local. Roll back by reverting the PR; existing
  clusters are unaffected until their configs are re-rendered and re-applied. The live
  manual `/usr/local/bin/hermescloud-cleanup` remains until replaced.
- **PR 2**: if the SSAR self-check misfires (false `degraded`), revert the PR — readiness
  returns to the prior convention. Self-check is read-only; it cannot break instance ops.
- **PR 3**: parameterized ingress + Cloudflare steps are guarded; on failure the cluster is
  still reachable via the existing path. Keep raw-IP registration as a documented fallback
  if Cloudflare automation fails.
- **PR 4**: the previous single-cluster `bridge.yml` deploy path is replaced; keep the prior
  workflow on a branch for one release as rollback. `:${sha}` images are immutable, so
  rollback = redeploy a prior SHA (must still exist in GHCR + on-node within the keep-N window).
- **PR 5**: `protect_against_deletion: true` guards accidental teardown; decommission via the
  documented drain → `hetzner-k3s delete` → DNS removal → registry removal sequence.
