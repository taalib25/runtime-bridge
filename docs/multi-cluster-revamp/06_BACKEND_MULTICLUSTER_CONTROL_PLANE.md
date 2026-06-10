# Backend Multi-Cluster Control Plane

## Purpose

The backend/control plane owns product state, routing, operation tracking, capacity reservations, support visibility, and bridge communication.

The backend does not create Kubernetes resources directly for customer runtime operations.

## Core Tables

### runtime_clusters

```sql
create table runtime_clusters (
  id text primary key,
  cluster_id text unique not null,
  provider text not null,
  provisioner text not null,
  region text,
  bridge_url text not null,
  runtime_base_domain text not null,
  ingress_ip text,
  bridge_version text,
  min_supported_bridge_version text,
  status text not null default 'maintenance',
  health_status text not null default 'unknown',
  last_seen_at timestamp,
  last_validated_at timestamp,
  last_summary_json jsonb,
  created_at timestamp not null default now(),
  updated_at timestamp not null default now()
);
```

Status values:

```txt
active
maintenance
draining
unhealthy
disabled
```

### agents

```sql
create table agents (
  id text primary key,
  user_id text not null,
  workspace_id text,
  cluster_id text references runtime_clusters(cluster_id),
  name text not null,
  plan text not null,
  status text not null,
  namespace text,
  helm_release text,
  runtime_url text,
  last_operation_id text,
  created_at timestamp not null default now(),
  updated_at timestamp not null default now()
);
```

### runtime_operations

```sql
create table runtime_operations (
  id text primary key,
  agent_id text,
  cluster_id text not null,
  bridge_operation_id text,
  type text not null,
  status text not null,
  requested_by_user_id text,
  error_code text,
  message text,
  started_at timestamp,
  finished_at timestamp,
  timeout_at timestamp,
  raw_json jsonb,
  created_at timestamp not null default now(),
  updated_at timestamp not null default now()
);
```

Status values:

```txt
queued
running
succeeded
failed
timeout
canceled
```

### capacity_reservations

```sql
create table capacity_reservations (
  id text primary key,
  cluster_id text not null,
  agent_id text,
  operation_id text,
  requested_cpu_m int not null,
  requested_memory_mib int not null,
  requested_storage_gib int,
  status text not null,
  expires_at timestamp not null,
  created_at timestamp not null default now(),
  released_at timestamp
);
```

Status values:

```txt
active
committed
released
expired
```

## Cluster Registration Endpoint

```txt
POST /api/admin/clusters
Authorization: Bearer $ADMIN_API_SECRET
```

This is the **real, existing** endpoint — do not use `/internal/runtime-clusters/register` (it does not exist).

Payload:

```json
{
  "cluster_id":    "kh-test",
  "bridge_url":    "https://bridge-kh-test.hermeshq.net",
  "bridge_secret": "<BRIDGE_SECRET>",
  "region":        "eu",
  "name":          "kh-test",
  "status":        "active"
}
```

Validation:

```txt
- bridge_url must start with https:// in production
- raw http://NODE_IP bridge_url not production-ready
- cluster_id is idempotent upsert (safe to re-run)
- bridge_secret is encrypted at rest (AES-256-GCM)
```

## Health Poller

Backend periodically polls each non-disabled cluster:

```txt
GET /healthz
GET /readyz
GET /version
GET /v1/cluster/summary
GET /v1/cluster/resources
```

Health behavior:

```txt
readyz failed -> health_status=unhealthy
bridge version too old -> health_status=incompatible
cluster status maintenance -> no new agents
cluster status draining -> no new agents, existing operations allowed by policy
```

## Routing Algorithm

New agent create:

```txt
1. Load active clusters.
2. Filter health_status=healthy.
3. Filter bridge_version >= min_supported_bridge_version.
4. Pull latest summary or poll bridge.
5. Compute effective headroom:
   bridge_headroom - active capacity reservations.
6. Filter clusters that can fit requested plan resources.
7. Pick highest score:
   headroom, lower pending pods, lower failure rate, plan/tier preference.
8. Create capacity reservation with short expiry.
9. Create agent row in creating status.
10. Call bridge /v1/instances.
11. Store bridge_operation_id.
12. Poll operation until terminal state.
13. Commit or release reservation.
```

## Capacity Reservation Rules

Reservation expiry:

```txt
create operation: 10 minutes
upgrade operation: 10 minutes
```

Release reservation when:

```txt
bridge create fails
operation times out
agent deleted before success
backend cancel occurs
```

Commit reservation when:

```txt
agent reaches Live and resources are now represented in cluster summary
```

Garbage collector:

```txt
run every minute
mark expired active reservations as expired
release expired capacity from routing calculation
```

## Operation Locking

Only one mutating operation per agent at a time.

Mutating operations:

```txt
create
update_config
update_secrets
restart
delete
upgrade_plan
backup
restore
```

Read operations can run concurrently:

```txt
diagnostics
logs
status
```

Implementation:

```txt
Use DB transaction or unique partial index for active mutating operation per agent.
```

## Bridge Auth

Backend signs short-lived JWTs per bridge request.

JWT must include:

```txt
iss
aud
sub
cluster_id
action
agent_id or operation scope
jti
exp
nbf
iat
```

See `08_SECURITY_SECRETS_AND_AUTH.md`.

## User-Facing Status Mapping

Do not expose Kubernetes terms to normal users.

```txt
Pending pod       -> Starting
PVC pending       -> Starting / Storage delayed
CrashLoopBackOff  -> Failed to start
Bridge unhealthy  -> Maintenance
Capacity full     -> Capacity delayed
Provider invalid  -> Needs setup
Messenger invalid -> Needs reconnection
```

## Backend Must Never

```txt
- hold kubeconfigs for every cluster for customer operations
- create customer pods directly
- patch Kubernetes Secrets directly for customer runtime operations
- route to unhealthy/draining/maintenance clusters for new creates
- leave operations running forever
```
