# Coding Agent Handoff Prompt

Copy this into the coding agent.

---

We are revamping HermesCloud runtime infrastructure to support multi-cluster runtime operation using kube-hetzner/Terraform while keeping the bridge as the runtime operator.

## Mission

Implement a new kube-hetzner based runtime cluster factory and multi-cluster bridge/backend control layer.

## Critical Direction

```txt
Terraform/kube-hetzner = cluster factory
Cloudflare provider    = DNS factory
GitHub Actions         = bootstrap pipeline
Bridge                 = runtime operator
Backend                = product control plane
Admin dashboard        = support visibility layer
```

Do not replace the bridge.

Do not let Terraform create customer Hermes agent pods.

Do not let backend talk directly to Kubernetes for customer runtime operations.

Do not destroy or modify the existing hetzner-k3s cluster.

## Documents To Read First

Read these in order:

```txt
README.md
CONTEXT.md
00_FOR_TAALIB_MANUAL_SETUP.md
01_TARGET_ARCHITECTURE.md
02_IMPLEMENTATION_MASTER_PLAN.md
03_TERRAFORM_KUBE_HETZNER_CLOUDFLARE.md
04_GITHUB_ACTIONS_BOOTSTRAP_PIPELINE.md
05_BRIDGE_V2_RUNTIME_OPERATOR_SPEC.md
06_BACKEND_MULTICLUSTER_CONTROL_PLANE.md
07_ADMIN_SUPPORT_DASHBOARD.md
08_SECURITY_SECRETS_AND_AUTH.md
09_STORAGE_BACKUPS_AND_CLEANUP.md
10_TESTING_VALIDATION_AND_ROLLOUT.md
12_PATCHED_CONTRACTS_SUMMARY.md
```

Also read ADRs:

```txt
docs/adr/0001-kube-hetzner-as-cluster-factory.md
docs/adr/0002-bridge-per-runtime-cluster.md
docs/adr/0003-terraform-dns-bridge-agent-lifecycle.md
docs/adr/0004-jwt-bridge-auth.md
docs/adr/0005-capacity-reservations.md
docs/adr/0006-maintenance-window-backups.md
```

## Implement As Separate PRs

Do not implement everything in one PR.

### PR 0 — docs/source of record

Patch repo docs to match this pack.

### PR 1 — Terraform kube-hetzner skeleton

Create:

```txt
infra/kube-hetzner/modules/runtime-cluster
infra/kube-hetzner/clusters/kh-test
infra/kube-hetzner/clusters/eu-1
infra/kube-hetzner/clusters/eu-2
```

Add providers, variables, outputs, remote-state placeholders.

Do not apply production.

### PR 2 — Cloudflare DNS via Terraform

Create cluster DNS records:

```txt
bridge-${cluster_id}.hermeshq.net
*.runtime-${cluster_id}.hermeshq.net
```

Output:

```txt
bridge_url
runtime_base_domain
ingress_ip
```

### PR 3 — GitHub Actions provision workflow

Manual workflow:

```txt
provision-runtime-cluster.yml
```

Must use:

```txt
terraform plan -out=tfplan
terraform apply tfplan
GitHub Environments
remote state
no destroy path
```

### PR 4 — Bridge deployment/bootstrap

Deploy bridge to `hermes-system` namespace with:

```txt
cluster_id
runtime_base_domain
bridge auth public key
bridge ingress host
system node scheduling
```

Validate:

```txt
/healthz
/readyz
/version
/v1/cluster/summary
/v1/cluster/resources
```

### PR 5 — Bridge V2 runtime operator contracts

Implement/verify:

```txt
startup RBAC SSAR self-check
readyz degraded on missing permissions
operation IDs
diagnostics
config injection
secret injection
restart/delete
terminal session endpoints
machine-readable error codes
```

### PR 6 — Backend control plane

Add:

```txt
runtime_clusters
runtime_operations
capacity_reservations
health poller
cluster registration endpoint
routing algorithm
operation locking
JWT bridge auth
```

### PR 7 — Admin support dashboard

Add:

```txt
/admin/runtime
/admin/runtime/:clusterId
/admin/agents/:agentId/diagnostics
/admin/operations
/admin/support
```

### PR 8 — Backup/restore/cleanup

Implement V1:

```txt
maintenance-window backup
restore proof
backup metadata
cleanup visibility
```

### PR 9 — kh-test proof

Only after kh-test passes, create eu-1/eu-2.

## Important Safety Contracts

```txt
1. Production bridge URLs must be HTTPS.
2. Raw http://NODE_IP bridge URLs are rejected in production.
3. Backend operation state is durable.
4. Bridge in-memory operation state is not the source of truth.
5. Capacity reservations prevent concurrent over-routing.
6. Terminal access is audited and limited to the user's own agent container.
7. Admin logs are redacted.
8. Backup is not valid until restore test succeeds.
9. Existing hetzner-k3s cluster remains fallback until new path is proven.
```

## First Deliverable

Start with PR 0 and PR 1 only.

Do not attempt eu-1/eu-2 provisioning until kh-test passes all tests.
