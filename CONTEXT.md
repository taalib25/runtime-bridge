# HermesCloud — Domain Glossary

This file is a glossary only. No implementation details, no specs.
Updated inline during design sessions.

---

## Core concepts

**Instance**
A single running Hermes AI coding environment for one tenant. Implemented as a Helm
release in an isolated Kubernetes namespace. The only unit the bridge creates,
updates, or deletes. The backend and frontend may call this a "workspace" — that is
a product-layer synonym. The bridge and all infrastructure code use "instance" only.

**Cluster**
A single k3s installation on Hetzner Cloud with one bridge process. The physical
unit of deployment. A cluster has a `plan_tier` property (`shared` | `enterprise`)
but is not itself a different kind of thing based on tier — plan tier is a routing
and billing label, not a type distinction.

**Bridge**
The Go control service running inside a cluster. Owns instance lifecycle (create /
update / delete / repair / upgrade) and cluster operations (maintenance / drain).
Does not own placement decisions — that is the backend's job.

**Backend**
The hermes-client service (separate repo). Owns: user auth, cluster routing
(`pickCluster`), the instances DB table, and the CF KV cluster-stats cache. The
backend is the sole decision-maker for which cluster an instance lands on.

**Plan tier**
A property of a cluster: `shared` (free + pro tenants) or `enterprise` (dedicated).
Determines routing eligibility, not instance behaviour.

**Headroom**
The estimated memory available on a cluster for new instances:
`allocatableMiB − systemOverheadMiB − reservedMiB`.
Exposed as `headroomMiB` in `/v1/cluster/summary`. Backend excludes clusters where
`headroomMiB < 1024` (one instance's memory request) from routing.

**Capacity check**
The synchronous pre-flight guard on `POST /v1/instances/:id`. Returns a 503 with a
machine-readable `code` when the cluster cannot safely accept one more instance.
The check covers: node liveness, pressure conditions, and memory headroom.

**Operation**
An async unit of work returned by all mutating instance calls (202 Accepted). Has a
stable `id`, a `type`, and a `status` (`running | succeeded | failed | superseded`).
`superseded` means a delete cancelled the operation — not an error from the
backend's perspective.

**Maintenance mode**
A cluster-level flag set via `PUT /v1/cluster/maintenance`. When on, creates return
503 with code `CLUSTER_MAINTENANCE`. Backend must exclude these clusters from
routing and re-route on 503.

---

## Error codes (503 responses)

Machine-readable `code` field in the 503 body. Backend switches on this, not the
human-readable `error` string.

| Code | Meaning | Backend action |
|---|---|---|
| `CLUSTER_MAINTENANCE` | cluster in maintenance mode | re-route to another cluster |
| `CLUSTER_AT_CAPACITY` | no headroom for another instance | re-route to another cluster |
| `NODE_PRESSURE` | all ready nodes have memory/disk pressure | re-route to another cluster |
| `KUBERNETES_UNREACHABLE` | bridge cannot reach the k8s API | mark cluster offline, re-route |
