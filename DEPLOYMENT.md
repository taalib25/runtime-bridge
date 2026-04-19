# Hermes Runtime Operator Deployment Guide

## Current Contract

- Primary CRD: `Runtime`
- API group: `hermes.hermeshq.net/v1alpha1`
- Controller: `RuntimeReconciler`
- Backend entrypoint: built-in HTTP compatibility API in `cmd/main.go`

The repo previously had a second CRD path (`HermesRuntime`) and legacy `runtime.hermescloud.io` references. The active path is now `Runtime` under `hermes.hermeshq.net`.

## What the Operator Manages

For each `Runtime` object, the controller reconciles resources in the **same namespace as the CR**:

- `ConfigMap` for `config.yaml` and `SOUL.md`
- `PersistentVolumeClaim`
- single-replica `Deployment`
- `Service`
- `Ingress`

The operator compatibility API manages one reserved Secret per runtime for backend-driven provider keys.

- managed Secret name: `<runtimeId>-secrets`
- backend should not create this Secret directly
- backend should not place this reserved name in `spec.envFromSecrets`

The backend may still reference additional external Secrets by name through `spec.envFromSecrets` when needed.

## Backend Integration Model

The backend should talk to the operator’s HTTP API, not patch Deployments or pods directly.

### Auth

Every runtime API request must include:

- header: `X-Controller-Secret`

The operator reads the shared secret from:

- env: `CONTROLLER_SHARED_SECRET`

### Namespace Routing

The operator API creates and reads `Runtime` CRs in a configured namespace.

- env: `RUNTIME_NAMESPACE`
- flag: `--runtime-namespace`

For local-against-prod testing, set `RUNTIME_NAMESPACE=hermes-prod`.

## Compatibility API

### Health

```http
GET /healthz
```

### Create runtime

```http
POST /runtimes
```

Body:

```json
{
  "tenantId": "tenant-001",
  "runtimeId": "tenant-001",
  "image": "nousresearch/hermes-agent:latest",
  "plan": {
    "cpuMillis": 1000,
    "memoryMi": 2048
  },
  "storage": {
    "sizeGi": 10,
    "storageClassName": "local-path"
  },
  "network": {
    "mode": "subdomain",
    "host": "hermeshq.net",
    "subdomain": "tenant-001",
    "ingressClassName": "traefik"
  },
  "config": {
    "model": {
      "provider": "auto",
      "default": "anthropic/claude-opus-4.6",
      "base_url": "https://openrouter.ai/api/v1"
    },
    "agent": {
      "max_turns": 90,
      "gateway_timeout": 1800
    }
  },
  "soul": "# Hermes Runtime\n\nYou are a helpful assistant.",
  "env": [
    {"name": "GATEWAY_ALLOW_ALL_USERS", "value": "false"}
  ]
}
```

### Update runtime

```http
PUT /runtimes/{id}
```

`runtimeId` and `tenantId` are treated as immutable. Send a full desired spec payload for mutable fields.

### Get managed secret metadata

```http
GET /runtimes/{id}/secrets
```

### Upsert managed secret keys

```http
POST /runtimes/{id}/secrets
```

Body:

```json
{
  "data": {
    "OPENROUTER_API_KEY": "sk-...",
    "API_SERVER_KEY": "..."
  },
  "actorId": "user-123"
}
```

### Delete one managed secret key

```http
DELETE /runtimes/{id}/secrets/{key}
```

### List runtimes

```http
GET /runtimes
GET /runtimes?tenantId=tenant-001
```

### Get runtime

```http
GET /runtimes/{id}
```

### Runtime health

```http
GET /runtimes/{id}/health
```

### Delete runtime

```http
DELETE /runtimes/{id}
```

## Running Locally Against the Production Cluster

You can run the operator locally and have it reconcile against the production cluster without deploying the manager image yet.

### 1. Build the binary

```bash
go build -o /tmp/runtime-operator ./cmd
```

### 2. Get a kubeconfig that points at prod

If the server is a remote k3s node, copy its kubeconfig and replace `127.0.0.1:6443` with the reachable server IP.

### 3. Run locally

```bash
CONTROLLER_SHARED_SECRET=local-test-secret \
RUNTIME_NAMESPACE=hermes-prod \
/tmp/runtime-operator \
  --leader-elect=false \
  --metrics-bind-address=0 \
  --api-bind-address=127.0.0.1:18080 \
  --kubeconfig /path/to/prod-kubeconfig
```

### 4. Call the API from the backend

Example create request:

```bash
curl -H 'X-Controller-Secret: local-test-secret' \
  -H 'Content-Type: application/json' \
  -d @runtime.json \
  http://127.0.0.1:18080/runtimes
```

## Resource Naming

For runtime ID `tenant-001`, the controller creates:

- `Deployment`: `tenant-001`
- `Service`: `tenant-001`
- `ConfigMap`: `tenant-001-config`
- `PVC`: `tenant-001-data`
- `Ingress`: `tenant-001-ingress`

## Ownership Boundaries

### Backend owns

- `Runtime` CR creation/update/delete
- optional external secret references other than the reserved managed Secret
- tenant metadata and API key lifecycle

### Operator owns

- child workload resources
- reserved managed Secret `<runtimeId>-secrets`
- status computation
- rollout/reconciliation

### Backend should not do

- patch the child `Deployment`
- patch the child `Service` or `Ingress`
- edit pod files directly

## Verification

### Repo verification

```bash
make generate manifests
go test ./...
```

### Cluster verification

```bash
kubectl get crd runtimes.hermes.hermeshq.net
kubectl get runtimes.hermes.hermeshq.net -n hermes-prod
kubectl get deploy,svc,ing,pvc -n hermes-prod
```
