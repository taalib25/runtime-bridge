# Implementation Master Plan

This plan must be executed as separate PRs. Do not combine everything into one giant PR.

## Migration Principle

```txt
Current hetzner-k3s cluster stays alive as fallback.
New kube-hetzner path is implemented separately.
kh-test proves the new flow.
eu-1/eu-2 are created only after kh-test passes.
```

## PR 0 — Documentation Source of Record

Create/update documentation:

```txt
docs/kube-hetzner-runtime-migration.md
docs/runtime-cluster-architecture.md
docs/manual-setup-checklist.md
```

Capture:

```txt
- final architecture
- responsibilities by layer
- migration strategy
- secrets checklist
- TLS strategy
- operation durability
- capacity reservation
- backup/restore contract
- rollout/rollback
- kh-test proof checklist
```

No runtime code changes in PR 0.

## PR 1 — Terraform kube-hetzner Skeleton

Create:

```txt
infra/kube-hetzner/
  modules/runtime-cluster/
  clusters/kh-test/
  clusters/eu-1/
  clusters/eu-2/
```

Requirements:

```txt
- Use kube-hetzner module.
- Configure remote Terraform state.
- Add cluster_id variable.
- Add root_domain variable.
- Add provider configuration for hcloud and cloudflare.
- Add outputs: kubeconfig, ingress_ip, bridge_url, runtime_base_domain, cluster_id.
- Do not deploy production cluster yet.
```

Acceptance:

```txt
terraform fmt
terraform validate
terraform plan for kh-test succeeds
no real apply to eu-1/eu-2 yet
```

## PR 2 — Cloudflare DNS via Terraform

Add Cloudflare provider resources for:

```txt
bridge-${cluster_id}.hermeshq.net
*.runtime-${cluster_id}.hermeshq.net
```

Requirements:

```txt
- idempotent DNS records
- records point to ingress/load balancer IP
- proxied setting configurable
- outputs available for pipeline
- bridge_url always uses https:// in production outputs
```

Acceptance:

```txt
terraform plan shows DNS records for kh-test
outputs include bridge_url and runtime_base_domain
```

## PR 3 — GitHub Actions Terraform Provision Workflow

Create:

```txt
.github/workflows/provision-runtime-cluster.yml
```

Inputs:

```txt
cluster_id: kh-test | eu-1 | eu-2
action: plan | apply
```

Requirements:

```txt
- Use GitHub Environment matching cluster_id.
- Use remote Terraform state.
- Plan using terraform plan -out=tfplan.
- Apply exactly the saved tfplan artifact from the same workflow run.
- Disable destroy from this workflow.
- eu-1/eu-2 require manual approval.
```

Acceptance:

```txt
kh-test plan works
apply requires explicit workflow action
no accidental destroy path exists
```

## PR 4 — Bridge Helm Chart / Deployment Manifests

Create or formalize bridge deployment:

```txt
charts/hermes-bridge/
# or deploy/bridge/ manifests if chart is not ready
```

Bridge deploy must include:

```txt
namespace hermes-system
ServiceAccount
RBAC
Deployment
Service
Ingress
ConfigMap/Secret for cluster_id, bridge auth public key, runtime_base_domain
nodeSelector/toleration for system pool if configured
```

Bridge Ingress host:

```txt
bridge-${cluster_id}.hermeshq.net
```

Acceptance:

```txt
bridge deploys into kh-test
rollout succeeds
/healthz passes
/readyz passes
/version passes
```

## PR 5 — Bridge V2 Runtime Operator Contracts

Implement/confirm bridge endpoints:

```txt
GET  /healthz
GET  /readyz
GET  /version
GET  /v1/cluster/summary
GET  /v1/cluster/resources
POST /v1/instances
PATCH /v1/instances/:id/config
PATCH /v1/instances/:id/secrets
POST /v1/instances/:id/restart
DELETE /v1/instances/:id
GET  /v1/instances/:id/diagnostics
GET  /v1/operations/:id
POST /v1/instances/:id/terminal/session
WS   /v1/instances/:id/terminal/connect
```

Must include:

```txt
- startup RBAC SelfSubjectAccessReview self-check
- degraded readyz on missing permissions
- durable operation IDs returned to backend
- clear errors with machine-readable codes
- no cluster-wide access beyond necessary resources
```

Acceptance:

```txt
missing RBAC causes /readyz degraded
create returns operation_id
operation can be polled
terminal only enters target agent container
```

## PR 6 — Backend Runtime Cluster Registry and Routing

Add/update DB:

```txt
runtime_clusters
runtime_cluster_health_snapshots
agents
runtime_operations
capacity_reservations
```

Backend behavior:

```txt
- register clusters from GitHub Actions
- poll bridge health/version/summary
- route creates only to active healthy compatible clusters
- create capacity reservations before bridge create
- release reservations on failure/timeout
- store operation_id from bridge
- enforce one mutating operation per agent
```

Acceptance:

```txt
backend can register kh-test
backend can create test agent through kh-test bridge
concurrent creates do not over-assign same headroom
operation timeout updates status to failed
```

## PR 7 — GitHub Actions Bootstrap Pipeline

Extend workflow after Terraform apply:

```txt
1. export Terraform outputs
2. wait for DNS
3. deploy bridge
4. validate bridge and cluster
5. register backend
6. show workflow summary
```

Validation:

```txt
- DNS resolves
- HTTPS /healthz works
- HTTPS /readyz works
- /version expected SHA
- /v1/cluster/summary sane
- /v1/cluster/resources sane
- StorageClass exists
- test PVC binds
- test Ingress route works
```

Acceptance:

```txt
kh-test appears in backend/admin only after all validation passes
```

## PR 8 — Admin Runtime and Support Dashboard

Create:

```txt
/admin/runtime
/admin/runtime/:clusterId
/admin/agents/:agentId/diagnostics
/admin/operations
```

Acceptance:

```txt
admin can see kh-test bridge health/version/headroom
admin can run diagnostics for test agent
admin can see operation timeline and failure reason
```

## PR 9 — Storage, Backup, Restore, Cleanup

Implement V1 backup/restore and cleanup plan:

```txt
- PVC per agent
- backup job to S3/R2/Storage Box
- maintenance-window backup mode initially
- restore test command/job
- cleanup for unused images/logs/status
```

Acceptance:

```txt
backup created for test agent
restore succeeds into same or new test agent
cleanup status visible in validation/admin
```

## PR 10 — kh-test Full Proof

Do not create eu-1/eu-2 before kh-test passes:

```txt
[ ] Terraform creates kh-test
[ ] DNS created
[ ] bridge deployed
[ ] backend registered
[ ] admin visible
[ ] test agent created
[ ] config injection works
[ ] secret injection works
[ ] terminal works
[ ] diagnostics works
[ ] PVC persists
[ ] backup/restore works
[ ] delete/purge works
```

## PR 11 — eu-1 and eu-2 Production Runtime Clusters

After kh-test proof:

```txt
- provision eu-1
- validate
- register
- keep in maintenance until smoke test passes
- set active
- provision eu-2
- validate
- register
- keep in maintenance until smoke test passes
- set active
```

## PR 12 — Drain/Retire Old hetzner-k3s Cluster

Only after eu-1/eu-2 stable:

```txt
- stop routing new agents to old cluster
- back up old agents
- migrate/recreate/delete old agents according to product decision
- validate no active customers remain
- retire old cluster manually
```
