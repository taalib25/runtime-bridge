# Instance Lifecycle & Operations

How the bridge manages an instance's life: create, update, the recovery/maintenance
operations, and the async operation model the backend polls. See also
[label-contract.md](label-contract.md) for the metadata applied to every resource.

## Operation model — async, poll by ID

Every mutating instance call returns **`202 Accepted`** with an `Operation`
record and runs the work in a background goroutine. The backend polls
`GET /v1/operations/{id}` for the result.

```json
{ "id": "ab12…", "type": "upgrade", "instanceId": "ws-…",
  "status": "running", "message": "upgrade scheduled", "startedAt": "…" }
```

`status` transitions: `running` → `succeeded` | `failed` | `superseded`.

- `failed` — the operation errored; `error` carries the message.
- `superseded` — a delete cancelled this operation (see below); not an error.

Operation records are snapshots: `GET /v1/operations/{id}` and
`GET /v1/instances/{id}/operations` return **value copies** taken under lock, so a
reader never races the worker goroutine mutating the live record. The `202` body is
likewise a snapshot taken before the goroutine launches.

### One-in-flight per instance

`restart`, `redeploy`, `rollback`, `repair`, and `upgrade` are guarded: only one may
run per instance at a time. A second request while one is in flight returns
**`409 Conflict`** naming the in-flight operation. `create`, `update`, and `delete`
use a separate path (`delete` is special — see below).

## Create

`POST /v1/instances/{id}` → Helm install into a per-instance namespace.

- Rejected with **`503`** when the cluster is in maintenance mode.
- Rejected with **`422`** when `commonLabels`/`commonAnnotations` violate the
  [label contract](label-contract.md).
- Namespace, bridge-owned Secret, and IngressRoute are created imperatively and
  carry the contract labels; Deployment/Service/PVC/Secret are Helm-rendered.

## Update

`PUT /v1/instances/{id}` → Helm upgrade. Same `422` label validation as create.
`commonLabels`/`commonAnnotations` are **full-replace** (omitted keys are removed).
`OverwriteConfig` is set only when the request carries a `config` block, so a
metadata-only PUT never wipes the agent's runtime `config.yaml`.

## Delete — always wins

`DELETE /v1/instances/{id}` (purge by default; `?purge=false` for soft delete
keeping the PVC + Helm history tombstone).

Delete **supersedes** any in-flight operation: it cancels that operation's context
so the operation short-circuits cleanly (marked `superseded`, not `failed`) instead
of, say, an upgrade rolling back into a release that's being uninstalled. A user can
always remove an instance, even mid-upgrade.

`DeleteInstance` is idempotent — a missing release is treated as success (safe for
retries / QStash redelivery).

## Recovery operations

| Op | Endpoint | What it does |
|---|---|---|
| `restart` | `POST …/restart` | Rolling restart (patches pod-template annotation), waits for ready + health |
| `redeploy` | `POST …/redeploy` | Re-runs Helm upgrade from the stored spec (PVC/namespace preserved, config untouched) |
| `rollback` | `POST …/rollback` | Helm rollback to a previous revision (`{"version": N}`, 0 = previous) |
| `upgrade` | `POST …/upgrade` | Helm upgrade to a new image/tag (`{"image","imageTag"}`); **auto-rolls-back on health failure** |
| `repair` | `POST …/repair` | Bounded auto-recovery based on pod state (see below) |

### Upgrade auto-rollback

On post-upgrade readiness/health failure, `upgrade` rolls back to the previous
revision under a **fresh context** (so the rollback's health-wait isn't starved by an
already-drained operation deadline). If the operation was superseded by a delete
(context canceled), it does **not** roll back — the release is going away.

### Repair decision table

`repair` inspects pod state and picks the minimal safe action; unrecoverable states
are returned as errors (retrying would loop forever):

| Condition | Action |
|---|---|
| `ImagePullBackOff` / `ErrImagePull` / `InvalidImageName` / `Unschedulable` | error (needs external fix) |
| Pod `Failed` (evicted / ContainerCannotRun) / `Unknown` | redeploy |
| Pod `Succeeded` (agent exited) | restart |
| `CrashLoopBackOff` / `RunContainerError` / `PostStartHookError` / `OOMKilled` | restart |
| `CreateContainerConfigError` / `CreateContainerError` / `ContainerCannotRun` | redeploy |
| `Init:*` stuck (non-image) | redeploy |
| otherwise unhealthy | restart |

## Status & phase

`GET /v1/instances/{id}` returns `InstanceStatus`. Derived `phase`:

`creating` → `starting` → `ready`, or `error` (container/image/schedule failure) or
`failed` (pod-level) or `deleted` (soft-deleted release). A `Running` pod that hasn't
passed its health probe yet is `starting`, never `failed`. Status echoes the effective
`selectorLabels`, `commonLabels`, and `commonAnnotations` for round-trip verification.

Pod selection prefers the **newest Running** pod, so during a rolling update status
reflects the incoming revision, not the terminating one.

## Cluster operations

| Op | Endpoint | Notes |
|---|---|---|
| Maintenance mode | `GET`/`PUT /v1/cluster/maintenance` (`{"enabled": bool}`) | When on, `create` returns `503`. Surfaced as `maintenance` in `/v1/cluster/summary`. |
| Drain | `POST /v1/cluster/drain` (`{"purge": bool}`) | Enables maintenance synchronously, then deletes every managed instance. `409` if a drain is already running. Skips already-tombstoned namespaces unless `purge=true`. Surfaced as `draining` in summary. |

## Control operations are label-independent

Discovery, delete, repair, drain, and diagnostics select **only** on bridge-owned
anchors — `app.kubernetes.io/instance=<releaseName>` for resources and
`hermeshq/managed-by=bridge` for namespaces — never on backend `commonLabels`. A
backend label-schema change can never strand or mis-target a control operation. See
[label-contract.md](label-contract.md).
