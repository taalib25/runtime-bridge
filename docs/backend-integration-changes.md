# Backend Integration — Required Changes

Audience: **hermes-client (backend)** team. This lists what the backend must change to
stay compatible with the current bridge, covering the label contract, the async
operation model, status fields, maintenance/drain, and observability.

Cross-refs: [label-contract.md](label-contract.md), [instance-lifecycle.md](instance-lifecycle.md),
bridge API in `bridge/bridge.go` `Router()`.

## Rollout note

The bridge ships **first** and is backward-compatible: `commonLabels`/`commonAnnotations`
are optional and identity is bridge-derived, so existing create/update calls keep
working. Backend changes can land incrementally — **except** the status-phase mapping
(#2) and 422 handling (#4), which you should verify regardless of rollout order because
they reflect how the bridge already behaves.

Legend: **[ACTION]** new work · **[VERIFY]** confirm/correct existing code · **[NOTE]** awareness.

---

## 1. [ACTION] Send `commonLabels` / `commonAnnotations` (label contract)

The bridge stamps backend-owned business metadata onto every Kubernetes resource, but
**only** what the backend sends. Add two optional maps to the create/update payload,
keyed under `hermescloud.dev/*` **only**.

```jsonc
// POST /v1/instances/:id  and  PUT /v1/instances/:id
{
  "tenantId": "...", "...": "...",
  "commonLabels": {
    "hermescloud.dev/instance-id":   "ws-1a2b...",
    "hermescloud.dev/workspace-id":  "wsp_987",
    "hermescloud.dev/cluster-id":    "hermes-test",
    "hermescloud.dev/plan":          "pro",
    "hermescloud.dev/provisioner":   "hermes-bridge",
    "hermescloud.dev/isolation-mode":"namespace",
    "hermescloud.dev/runtime-class": "standard"
  },
  "commonAnnotations": {
    "hermescloud.dev/display-name": "Ada's Workspace",
    "hermescloud.dev/billing-ref":  "cus_123/sub_abc"
  }
}
```

Rules the backend must follow:
- **Only** `hermescloud.dev/*` keys. Any other prefix (`app.kubernetes.io/*`, `helm.sh/*`,
  `hermeshq/*`, `kubernetes.io/*`, …) is **rejected with 422**.
- **Labels** = short, searchable. Values must be valid k8s label values (≤63 chars,
  `[a-z0-9A-Z._-]`, alphanumeric start/end). Anything longer or with `@ : / { }` →
  put it in **annotations** instead, or you get a 422.
- **Annotations** = long / non-searchable (display names, billing refs, URLs, JSON).
- **No secrets** in either — undetectable by the bridge, backend's responsibility.
- Do **not** send `selectorLabels` — the bridge derives Kubernetes identity itself.
- **PUT is full-replace**: send the complete desired set every time; a key you omit is
  removed. (There is no partial-merge / delete-one-key.)
- **Backfill:** instances created before you start sending labels acquire them on their
  next `PUT`. No migration job needed.

Verify round-trip: `GET /v1/instances/:id` echoes `selectorLabels`, `commonLabels`,
`commonAnnotations` — diff against what you sent to detect drift.

> `plan` stays a top-level InstanceSpec field too (drives the `PLAN` env var). Send it
> **both** as `plan` and as `hermescloud.dev/plan` in `commonLabels` for labelling.

---

## 2. [VERIFY] Status `phase` values are lowercase + `healthy` bool

`GET /v1/instances/:id` returns a derived `phase` — **not** the raw pod phase. If any
backend code switches on `Pending`/`Running`/`Failed`/`Unknown`, it is wrong.

| `phase` | Meaning | Suggested frontend state |
|---|---|---|
| `creating` | not scheduled / init containers running | "provisioning" |
| `starting` | pod Running, health probe not yet passing | "starting" |
| `ready` | healthy | "healthy" |
| `error` | container/image/schedule failure | "error — needs attention" |
| `failed` | pod-level failure | "failed" |
| `deleted` | release soft-deleted | stop polling, mark gone |

Also: readiness is the boolean **`healthy`** (not `ready`), and there is no top-level
`podName` (see `podPhase`, `waitingReason`, `restartCount`, `replicas`,
`readyReplicas`). Update any status-polling / loading-state mapping accordingly.

---

## 3. [ACTION] Use the async operation model for mutations

All mutating calls are async. Two response shapes:

**Create** — `POST /v1/instances/:id` → `202` with a provisioning payload (not an
Operation). Poll `GET /v1/instances/:id` until `phase == "ready"` (or `error`/`failed`).
```json
{ "instanceId": "...", "clusterId": "...", "status": "provisioning",
  "url": "https://...", "dashboardUrl": "https://...",
  "secrets": { "API_SERVER_KEY": "<generated>" } }
```

**Update / Delete / restart / redeploy / upgrade / rollback / repair** → `202` with an
**Operation**; poll `GET /v1/operations/:id` for the terminal status.
```json
{ "id": "ab12…", "type": "upgrade", "instanceId": "...",
  "status": "running", "message": "...", "startedAt": "..." }
```
- `status`: `running → succeeded | failed | superseded`.
- **`superseded`** = a delete cancelled this op. Treat as "instance is being deleted",
  not as an error.
- One op in flight per instance → a second lifecycle call returns **409**; back off and
  retry or surface "operation in progress".

Lifecycle endpoints the backend admin API should expose (Admin UI → Backend → Bridge):
`POST …/restart`, `…/redeploy`, `…/upgrade` (`{image,imageTag}`), `…/rollback`
(`{version}`), `…/repair`, and `GET …/operations`, `GET /v1/operations/:id`.

---

## 4. [ACTION] Handle `422` (label contract violations)

`POST`/`PUT` now return **422** when `commonLabels`/`commonAnnotations` are invalid
(wrong prefix, bad key, oversize label value). The body's `error` names the offending
key. **Do not blind-retry a 422** — it will fail identically. Surface it to the caller /
admin and fix the payload. (Distinct from `400` malformed-body and `503` capacity/maintenance.)

---

## 5. [ACTION] Capacity-aware routing + machine-readable 503 codes

`GET /v1/cluster/summary` now includes:
- `"maintenance": bool` — exclude from routing when true
- `"draining": bool` — exclude from routing when true
- `"headroomMiB": int` — estimated free memory for new instances (`allocatable − overhead − reserved`). **Exclude clusters where `headroomMiB < 1024`** (one instance's memory request) from `pickCluster()`.

All 503 responses from `POST /v1/instances/:id` now carry a machine-readable `code` field. **Switch on `code`, not the `error` string** — the error string is human-readable and may change:

```ts
const REROUTE_CODES = new Set([
  "CLUSTER_MAINTENANCE",
  "CLUSTER_AT_CAPACITY",
  "NODE_PRESSURE",
  "KUBERNETES_UNREACHABLE",
])

if (res.status === 503) {
  const { code } = await res.json()
  if (REROUTE_CODES.has(code)) {
    await markClusterDegraded(cluster.cluster_id, 60) // 60s cooldown in KV
    continue // try next cluster
  }
  throw new Error(`bridge 503: ${code}`)
}
```

**Optimistic cache update on successful create:** after a 202, immediately increment
`reservedMemoryMiB += 1024` and `instanceCount += 1` in the CF KV cluster-stats entry
without waiting for the next poll. Prevents overcommit under burst traffic where the 60s
cache TTL would otherwise route 5 concurrent creates to a cluster that fits 2.

`pickCluster()` exclusion filter (complete list):
```ts
.filter(({ stats }) =>
  stats?.kubernetesReachable &&
  stats?.pods.crashLooping === 0 &&
  !stats?.maintenance &&
  !stats?.draining &&
  (stats?.headroomMiB ?? 0) >= 1024  // room for one more instance
)
```

Mirror bridge state into `clusters.status` column (`active` | `draining` | `offline`)
so the admin dashboard and routing agree.

Expose admin controls that proxy to the bridge: `GET`/`PUT /v1/cluster/maintenance`
(`{enabled}`) and `POST /v1/cluster/drain` (`{purge}`).

---

## 6. [VERIFY] Delete semantics

- `DELETE /v1/instances/:id` **purges by default** (deletes namespace + PVC + data).
  Pass `?purge=false` for a soft delete that keeps the PVC and Helm history (tombstone).
- Delete is async (`202` + Operation) and **always wins**: it supersedes any in-flight
  op for that instance. Don't expect a concurrent upgrade/restart to complete — its op
  will report `superseded`.
- Delete is idempotent — deleting an already-gone instance succeeds.

---

## 7. [NOTE] API_SERVER_KEY lifecycle (unchanged, but easy to break)

The create response returns `secrets.API_SERVER_KEY`. **Store it and resend it unchanged
on every `PUT`** — if omitted, it is regenerated and the running instance's key rotates.
This is unrelated to labels but worth re-checking now that you'll be issuing PUTs to set
`commonLabels`.

---

## 8. [OBSERVABILITY] `hermes.ai/plan` pod label removed

The bridge no longer stamps the legacy `hermes.ai/plan` pod label (it was the bridge
inventing business metadata). Plan now lives at `hermescloud.dev/plan` (sent by backend,
applied to every resource incl. the pod). **Migrate any PromQL / Grafana / billing query
that grouped by `hermes.ai/plan` → `hermescloud.dev/plan`** before/with this rollout, or
those panels go blank.

---

## 9. [OBSERVABILITY] `GET /v1/cluster/summary` gained `headroomUnknown` / `runtimeImageStatusUnknown`

Both default to `false`/omitted and existing fields (`headroomMiB`, `runtimeImageWarmed`,
`prepullerDesired`/`prepullerReady`) are unchanged in shape — this is additive, no action
required to keep working. But if you want routing to actually benefit from it: when
`headroomUnknown` or `runtimeImageStatusUnknown` is `true`, the bridge could not check
(RBAC, timeout) rather than having checked and found zero/cold. Treat that the same as
`KUBERNETES_UNREACHABLE` for routing purposes (exclude/deprioritize, don't read the
zero-valued field as a real "no headroom" / "not warmed" signal) rather than ignoring it.

---

## Checklist

- [ ] Send `commonLabels` + `commonAnnotations` (`hermescloud.dev/*`) on create + PUT
- [ ] PUT sends the **complete** label set (full-replace semantics)
- [ ] Handle `422` without blind-retry; surface the offending key
- [ ] Fix status mapping to lowercase `phase` + `healthy` bool
- [ ] Poll `GET /v1/operations/:id` for update/delete/lifecycle ops; handle `superseded` + `409`
- [ ] Exclude `maintenance`/`draining`/`headroomMiB < 1024` clusters from `pickCluster()`
- [ ] Switch 503 handling on `code` field (`CLUSTER_MAINTENANCE` / `CLUSTER_AT_CAPACITY` / `NODE_PRESSURE` / `KUBERNETES_UNREACHABLE`) — not error string
- [ ] Optimistic KV cache update on successful create (`reservedMemoryMiB += 1024`, `instanceCount += 1`)
- [ ] Expose admin proxies: restart/redeploy/upgrade/rollback/repair + maintenance + drain
- [ ] Confirm delete `?purge` usage; expect async + supersede
- [ ] Resend `API_SERVER_KEY` on every PUT
- [ ] Migrate dashboards/billing from `hermes.ai/plan` → `hermescloud.dev/plan`
