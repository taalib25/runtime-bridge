# GitHub Actions Bootstrap Pipeline

## Purpose

This workflow replaces the old cluster provisioning script for the kube-hetzner path.

The pipeline does four jobs:

```txt
1. Terraform plan/apply cluster and DNS.
2. Deploy bridge.
3. Validate bridge and cluster.
4. Register cluster with backend.
```

## Workflow Name

```txt
.github/workflows/provision-runtime-cluster.yml
```

## Inputs

```yaml
workflow_dispatch:
  inputs:
    cluster_id:
      description: "Runtime cluster id"
      required: true
      type: choice
      options:
        - kh-test
        - eu-1
        - eu-2
    action:
      description: "Plan or apply"
      required: true
      type: choice
      options:
        - plan
        - apply
```

## Required Secrets

Per GitHub Environment:

```txt
HCLOUD_TOKEN
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_ENDPOINT         (var, not secret — set as environment variable)
CLOUDFLARE_API_TOKEN
CLOUDFLARE_ZONE_ID
BRIDGE_SECRET
GHCR_PAT
```

Repo-level (not per-environment):

```txt
ADMIN_API_SECRET    — used by register-backend job to call POST /api/admin/clusters
HETZNER_SSH_PRIVATE_KEY
```

Optional:

```txt
BACKUP_BUCKET credentials
```

Do not add TERRAFORM_CLOUD_TOKEN. OpenTofu uses Cloudflare R2 for state.

## Backend Registration — Exact Contract

The `register-backend` job calls the **existing** admin endpoint:

```
POST https://api.hermeshq.net/api/admin/clusters
Authorization: Bearer <ADMIN_API_SECRET>
Content-Type: application/json

{
  "cluster_id":    "kh-test",
  "bridge_url":    "https://bridge-kh-test.hermeshq.net",
  "bridge_secret": "<BRIDGE_SECRET>",
  "region":        "eu",
  "status":        "maintenance"
}
```

- `ADMIN_API_SECRET` is the **repo-level** secret (not per-environment).
- `bridge_secret` is required in the body — the backend encrypts and stores it so it can authenticate future bridge API calls.
- The call is **idempotent** (upsert on `cluster_id`) — safe to re-run.
- New clusters register as `maintenance`. Admin promotes to `active` via the admin dashboard.
- Do NOT use `/internal/runtime-clusters/register` — that endpoint does not exist.

## Pipeline Jobs

### Job 1 — tofu-plan

```txt
checkout
opentofu/setup-opentofu@v1  (pin version)
tofu init -backend-config from environment secrets
tofu fmt -check
tofu validate
tofu plan -out=tfplan
upload tfplan artifact for this workflow run only
```

### Job 2 — terraform-apply

Only runs when input `action == apply`.

```txt
download tfplan artifact from same workflow run
tofu apply tfplan
tofu output -json > tofu-outputs.json
```

Safety rules:

```txt
- Do not run `terraform apply` without saved plan.
- Do not re-plan after approval unless explicitly accepted in docs.
- Do not include destroy logic in this workflow.
- eu-1/eu-2 require GitHub Environment approval.
```

### Job 3 — bridge-deploy

Input:

```txt
bridge_url
runtime_base_domain
cluster_id
kubeconfig
```

Steps:

```txt
write kubeconfig from Terraform output
kubectl cluster-info

# Deploy bridge using raw manifests (no Helm chart yet — charts/hermes-bridge/ is a follow-on)
# BRIDGE_NAMESPACE=hermes-system for all new clusters
# hermes-test legacy continues to use hermes-bridge (do not change)

# Apply order:
#   1. namespace
#   2. RBAC (rbac.yaml)
#   3. service (service.yaml)
#   4. deployment (deployment.yaml) — sed-substitute BRIDGE_NAMESPACE + GIT_SHA
#   5. ingress (ingress.yaml) — sed-substitute BRIDGE_NAMESPACE + CLUSTER_NAME

# Namespace substitution: sed "s/namespace: hermes-bridge/namespace: $BRIDGE_NAMESPACE/g"
# Resource names (name: hermes-bridge) stay unchanged
# CLUSTER_NAME substitution for ingress: sed "s/__CLUSTER_NAME__/$CLUSTER_ID/g"

kubectl create namespace "$BRIDGE_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f rbac.yaml (substituted)
kubectl apply -f service.yaml (substituted)
kubectl apply -f deployment.yaml (substituted — image tag = built SHA)
kubectl apply -f ingress.yaml (substituted — cluster_id + namespace)
kubectl rollout status deployment/hermes-bridge -n "$BRIDGE_NAMESPACE"
```

Bridge config:

```txt
CLUSTER_ID=<cluster_id>
RUNTIME_BASE_DOMAIN=runtime-<cluster_id>.hermeshq.net
BRIDGE_AUTH_PUBLIC_KEY=<public key>
```

Bridge Ingress host:

```txt
bridge-<cluster_id>.hermeshq.net
```

### Job 4 — validate-cluster

Validation must run before backend registration.

Checks:

```txt
DNS:
  bridge-<cluster_id>.hermeshq.net resolves
  *.runtime-<cluster_id>.hermeshq.net resolves or wildcard behavior is testable

HTTPS:
  GET https://bridge-<cluster_id>.hermeshq.net/healthz
  GET https://bridge-<cluster_id>.hermeshq.net/readyz
  GET https://bridge-<cluster_id>.hermeshq.net/version
  GET https://bridge-<cluster_id>.hermeshq.net/v1/cluster/summary
  GET https://bridge-<cluster_id>.hermeshq.net/v1/cluster/resources

Kubernetes:
  nodes Ready
  bridge Deployment Ready
  StorageClass exists
  test PVC binds
  test Ingress route works
  bridge RBAC self-check healthy

Security:
  backend bridge URL is https://
  raw http://NODE_IP bridge URL is not used
```

### Job 5 — register-backend

Call (see §Backend Registration — Exact Contract above for the full example):

```txt
POST /api/admin/clusters
Authorization: Bearer $ADMIN_API_SECRET
```

Payload:

```json
{
  "cluster_id":    "kh-test",
  "bridge_url":    "https://bridge-kh-test.hermeshq.net",
  "bridge_secret": "<BRIDGE_SECRET>",
  "region":        "eu",
  "name":          "kh-test",
  "status": "maintenance"
}
```

Recommended: new clusters register as `maintenance` first. Admin explicitly marks them `active` after smoke tests.

## Workflow Summary

At the end of the run, write a GitHub summary:

```txt
Cluster: kh-test
Terraform: applied
DNS: ok
Bridge URL: https://bridge-kh-test.hermeshq.net
Bridge version: abc123
Readyz: ok
Cluster summary: ok
Storage test: ok
Backend registration: ok
Initial status: maintenance
```

## Destroy Workflow

Do not add destroy to the provision workflow.

If needed later, create a separate break-glass workflow:

```txt
.github/workflows/destroy-runtime-cluster.yml
```

Requirements:

```txt
manual-only
production approval
requires typing cluster_id twice
requires checking no active agents
requires final confirmation
```

## Idempotency Expectations

Re-running the pipeline should:

```txt
- not duplicate DNS records
- not duplicate backend cluster rows
- update bridge version if changed
- update cluster outputs safely
- leave cluster registered if already registered
```
