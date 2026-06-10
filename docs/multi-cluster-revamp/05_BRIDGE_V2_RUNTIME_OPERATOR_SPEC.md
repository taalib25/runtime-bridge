# Bridge V2 Runtime Operator Specification

## Purpose

The bridge is the runtime operator for one runtime cluster.

It receives product-level commands from HermesCloud backend and translates them into Kubernetes/Helm operations inside its own cluster.

## Non-Negotiable Boundary

```txt
Bridge controls exactly one runtime cluster.
Bridge does not hold kubeconfigs for every cluster.
Backend does not talk directly to Kubernetes for customer runtime operations.
Terraform does not create customer agent pods.
```

## Required Endpoints

### Health and Version

```txt
GET /healthz
GET /readyz
GET /version
```

`/version` response:

```json
{
  "bridgeVersion": "0.4.0",
  "gitSha": "abc123",
  "chartVersion": "0.2.1",
  "apiVersion": "v1",
  "clusterId": "eu-1",
  "builtAt": "2026-06-09T00:00:00Z"
}
```

### Cluster

```txt
GET /v1/cluster/summary
GET /v1/cluster/resources
```

`summary` should include:

```txt
cluster_id
headroomMiB
podsRunning
podsPending
podsFailed
podsCrashLooping
instanceCount
reservedCpuM
reservedMemoryMiB
maintenance
draining
bridgeVersion
```

### Agent Runtime Lifecycle

```txt
POST   /v1/instances
PATCH  /v1/instances/:id/config
PATCH  /v1/instances/:id/secrets
POST   /v1/instances/:id/restart
DELETE /v1/instances/:id
GET    /v1/instances/:id
GET    /v1/instances/:id/diagnostics
```

### Operations

```txt
GET /v1/operations/:id
```

### Terminal

```txt
POST /v1/instances/:id/terminal/session
WS   /v1/instances/:id/terminal/connect
```

## Startup RBAC Self-Check

On startup, the bridge must run SelfSubjectAccessReview checks for all Kubernetes resources it needs.

Minimum resource verbs/kinds to check:

```txt
namespaces: create/get/list/delete
secrets: create/get/list/patch/update/delete
configmaps: create/get/list/patch/update/delete
services: create/get/list/patch/update/delete
deployments.apps: create/get/list/patch/update/delete
persistentvolumeclaims: create/get/list/patch/update/delete
ingresses.networking.k8s.io: create/get/list/patch/update/delete
networkpolicies.networking.k8s.io: create/get/list/patch/update/delete
resourcequotas: create/get/list/patch/update/delete
limitranges: create/get/list/patch/update/delete
pods: get/list/watch
pods/log: get
pods/exec: create
```

If any required permission is missing:

```txt
/readyz returns degraded/non-200 according to existing readiness convention
bridge logs missing permission clearly
backend must not route new operations to this bridge
admin UI shows missing permission
```

## Operation Durability Contract

The backend is the durable source of truth for operations.

Bridge may keep in-memory progress, but bridge memory must not be the only place operation state exists.

### Required behavior

When bridge accepts a mutating request:

```txt
1. Bridge returns operation_id immediately after input validation.
2. Backend stores operation_id in runtime_operations.
3. Backend polls bridge /v1/operations/:id.
4. If bridge restarts and loses in-memory operation status, backend must recover by calling diagnostics/status endpoints.
5. Operations must eventually become succeeded, failed, canceled, or timeout.
```

### Operation response

```json
{
  "operationId": "op_123",
  "instanceId": "agent_abc",
  "type": "create",
  "status": "running",
  "startedAt": "2026-06-09T00:00:00Z",
  "updatedAt": "2026-06-09T00:01:00Z",
  "steps": [
    { "name": "namespace", "status": "succeeded" },
    { "name": "secret", "status": "succeeded" },
    { "name": "helm-install", "status": "running" }
  ],
  "errorCode": null,
  "message": null
}
```

### Operation timeout

Backend sets timeout per operation type:

```txt
create: 10 minutes
update config: 3 minutes
restart: 5 minutes
delete: 5 minutes
backup: plan-specific
restore: plan-specific
```

If timeout occurs:

```txt
backend marks operation timeout
backend calls diagnostics
admin UI shows recovery recommendation
capacity reservation is released
```

## Helm Ownership

The bridge uses the Hermes Agent Helm chart for customer agent runtimes.

One customer agent runtime equals:

```txt
1 namespace
1 Helm release
1 Deployment/Pod
1 PVC
1 Secret set
1 Service
1 Ingress host
```

Never scale one Hermes agent Deployment to multiple replicas unless the runtime is proven stateless and safe. Default must be replica count 1.

## Config and Secret Injection

Use:

```txt
Secret:
  provider keys
  Telegram/Discord/Slack tokens
  private credentials

ConfigMap:
  non-sensitive config only

PVC:
  active runtime data
```

Bridge endpoints must support:

```txt
PATCH /v1/instances/:id/config
PATCH /v1/instances/:id/secrets
```

If Hermes Agent only reads config at boot, bridge must trigger restart or supported reload after config/secret changes.

## Terminal Security

Terminal access means shell inside the customer agent container only.

Must never allow:

```txt
node shell
host shell
control plane shell
bridge pod shell
kube-system pod shell
cluster-admin shell
```

Terminal requirements:

```txt
- backend authenticates user
- backend authorizes agent access
- terminal session ID is created and audited
- bridge validates target instance belongs to requested cluster
- bridge executes only into allowed container name
- session start/stop recorded
- terminal disabled for suspended/billing-failed agents
- terminal disabled for free tier if product requires
```

## Diagnostics Contract

`GET /v1/instances/:id/diagnostics` returns:

```txt
instance status
namespace
pod name
pod phase
container statuses
PVC status
latest Kubernetes events
latest logs, redacted
runtime URL
recommendation
machine-readable issue category
```

Issue categories:

```txt
capacity
storage
runtime_boot
provider_key
messenger
network_dns
bridge_rbac
unknown
```

## Error Codes

Bridge errors must be machine-readable:

```txt
ERR_CLUSTER_AT_CAPACITY
ERR_RBAC_MISSING_PERMISSION
ERR_STORAGE_PVC_PENDING
ERR_HELM_INSTALL_FAILED
ERR_INSTANCE_NOT_FOUND
ERR_TERMINAL_NOT_ALLOWED
ERR_PROVIDER_SECRET_INVALID
ERR_RUNTIME_HEALTH_FAILED
```

## Auth

Bridge accepts only valid backend-signed JWTs.

See `08_SECURITY_SECRETS_AND_AUTH.md` for exact claims.
