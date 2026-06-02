# HermesCloud Label & Annotation Contract (v1)

Status: accepted
Owners: backend ↔ bridge

## Principle

**Backend owns business metadata. Bridge owns Kubernetes identity. Bridge protects
Kubernetes and never mutates backend data.**

The backend is the source of truth for all business facts about an instance. The
bridge translates those facts onto Kubernetes resources without inventing,
rewriting, or silently fixing them. If backend sends something Kubernetes can't
accept, the bridge rejects the whole request with `422` — it does not lowercase,
truncate, or relocate values.

This decoupling is the point: the backend can add, rename, or drop business
metadata forever without a coordinated bridge release, because nothing the bridge
*does* (discovery, repair, delete, drain, diagnostics) depends on backend labels.

## What the backend sends

On `POST /v1/instances/{id}` (create) and `PUT /v1/instances/{id}` (update), the
backend may include two **optional** maps:

| Field | Purpose | Value rules |
|---|---|---|
| `commonLabels` | short, searchable business metadata | k8s label value: ≤63 chars, `[a-z0-9A-Z._-]`, alphanumeric start/end |
| `commonAnnotations` | long / non-searchable metadata (display names, billing refs, request IDs, URLs, JSON) | relaxed; long strings allowed, 256 KB total budget |

Both maps use **only** the `hermescloud.dev/*` key prefix.

Example create payload (excerpt):

```json
{
  "instanceId": "ws-1a2b3c4d5e6f7a8b",
  "tenantId": "t_123",
  "commonLabels": {
    "hermescloud.dev/instance-id": "ws-1a2b3c4d5e6f7a8b",
    "hermescloud.dev/workspace-id": "wsp_987",
    "hermescloud.dev/cluster-id": "hermes-test",
    "hermescloud.dev/plan": "pro",
    "hermescloud.dev/provisioner": "hermes-bridge",
    "hermescloud.dev/isolation-mode": "namespace",
    "hermescloud.dev/runtime-class": "standard"
  },
  "commonAnnotations": {
    "hermescloud.dev/display-name": "Ada's Workspace",
    "hermescloud.dev/billing-ref": "cus_1234567890/sub_abcdef"
  }
}
```

### Backend must NOT send

- `selectorLabels` — removed from the contract entirely (see "Identity" below).
- Any key under a reserved prefix:
  `app.kubernetes.io/*`, `helm.sh/*`, `meta.helm.sh/*`, `kubernetes.io/*`,
  `k8s.io/*`, `hermeshq/*`.

Because the selector keys (`app.kubernetes.io/name`, `app.kubernetes.io/instance`)
live under a reserved prefix, the `hermescloud.dev/*` allowlist also structurally
prevents any backend label from colliding with a selector key.

## Bridge validation (reject, never mutate)

The bridge validates `commonLabels` and `commonAnnotations` on create and update:

1. Every key must start with `hermescloud.dev/` → else `422`.
2. Every key must be a syntactically valid Kubernetes qualified name
   (`validation.IsQualifiedName`) → else `422`.
3. `commonLabels` **values** must be valid Kubernetes label values
   (`validation.IsValidLabelValue`, ≤63 chars) → else `422`, with a hint to move
   long values to `commonAnnotations`.
4. `commonAnnotations` values are unconstrained (relaxed) apart from the k8s total
   annotation size budget.

Rules the bridge does **not** enforce (backend obligations, documented here):

- lowercase preference — backend produces clean values.
- no emails / no URLs / no JSON in **labels** — mostly fall out of rule 3
  (illegal label chars), and otherwise belong in `commonAnnotations`.
- **no secrets in either channel** — undetectable by the bridge; backend's
  responsibility.

The bridge never lowercases, truncates, auto-fixes, or moves data between
channels. Absent/empty maps are valid and never `422`.

## Kubernetes identity (bridge-derived)

The bridge is the sole author of the selector identity. It is a pure function of
the stable release name, so it is identical on every create / upgrade / repair —
mismatch is structurally impossible and there is no immutability guard to trip:

```
selectorLabels = {
  app.kubernetes.io/name:     runtime-node-core
  app.kubernetes.io/instance: <releaseName>
}
```

`selectorLabels` never contain `plan`, `workspaceId`, `clusterId`, `provisioner`,
`isolationMode`, `runtimeClass`, or any `hermescloud.dev/*` key.

## Standard label ownership (bridge/chart-owned)

`app.kubernetes.io/{name,instance,part-of,managed-by}`, `helm.sh/chart`, Helm
release annotations, and `hermeshq/managed-by` are owned by the bridge/chart.

`app.kubernetes.io/managed-by` is set **truthfully per creator**:

- Helm-rendered resources keep `app.kubernetes.io/managed-by=Helm`.
- Bridge-created (imperative) resources use `app.kubernetes.io/managed-by=hermes-bridge`.
- Platform/business ownership is expressed with `hermescloud.dev/managed-by` or
  `hermescloud.dev/provisioner` — never by overriding the standard key.

## Chart responsibilities

- Accept `.Values.commonLabels` and `.Values.commonAnnotations`.
- Compute `selectorLabels` internally and use them **only** for selectors.
- Merge `commonLabels` into `metadata.labels` and `commonAnnotations` into
  `metadata.annotations` on every resource (and `commonLabels` onto the pod
  template labels).
- Guarantee `commonLabels` can never override `selectorLabels` or reach a
  selector (chart-owned labels are emitted last; last-wins in YAML).
- Preserve `app.kubernetes.io/instance=<Release.Name>`.

## Control-operation independence

Discovery, delete, repair, drain, and diagnostics key **only** on bridge-owned
operational anchors:

- resources: `app.kubernetes.io/instance=<releaseName>`
- namespaces: `hermeshq/managed-by=bridge`

Backend `commonLabels` / `commonAnnotations` are for observability, billing, and
admin filtering only. **They are never load-bearing for a control operation**, so a
backend label-schema change can never strand, mis-delete, or hide a running
instance.

## Persistence & round-trip

- Both maps are persisted into the Helm release values (`bridge.instance` block)
  and parsed back by `instanceSpecFromRelease`, so internal operations
  (`repair`, `redeploy`, `upgrade`) preserve them instead of silently stripping
  them.
- `PUT` is a **full replace**: the backend sends the complete desired set; keys it
  omits are removed (you cannot otherwise delete a label).
- The bridge echoes the effective `commonLabels`, `commonAnnotations`, and derived
  `selectorLabels` in `InstanceStatus` for round-trip verification and drift
  detection.

## Scope

Mechanism-complete over the resources that exist today:

- **Helm-rendered:** Deployment, Service, 2× PVC, Secret.
- **Imperative (bridge-created):** Namespace, Traefik IngressRoute, bridge-owned
  instance Secret.

Future resources (ResourceQuota, LimitRange, NetworkPolicy, ServiceAccount,
Ingress/HTTPRoute) are **not created** by this change. The chart label block and a
single bridge `instanceLabels()` / `instanceAnnotations()` helper are written so
those resources inherit the contract automatically when they are added.

## Rollout

Both maps are optional and the **bridge ships first** — it validates-if-present,
derives identity always, and persists if present. The backend starts sending
`hermescloud.dev/*` whenever ready. No flag day, no version lock; instances created
before this contract keep working and acquire labels organically on their next
`PUT`.
