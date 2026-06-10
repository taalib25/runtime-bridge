# Patched Contracts Summary

This file summarizes the gaps found during the grill review and how the docs have been patched.

## 1. TLS and Certificate Strategy

Patched in:

```txt
00_FOR_TAALIB_MANUAL_SETUP.md
01_TARGET_ARCHITECTURE.md
03_TERRAFORM_KUBE_HETZNER_CLOUDFLARE.md
04_GITHUB_ACTIONS_BOOTSTRAP_PIPELINE.md
08_SECURITY_SECRETS_AND_AUTH.md
```

Decision:

```txt
Production bridge URLs must be HTTPS.
Raw http://NODE_IP bridge URLs are forbidden in production.
V1 can use Cloudflare Edge TLS if raw origin access is blocked.
Origin TLS/cert-manager is a V1.5 hardening path.
```

## 2. Operation Durability

Patched in:

```txt
05_BRIDGE_V2_RUNTIME_OPERATOR_SPEC.md
06_BACKEND_MULTICLUSTER_CONTROL_PLANE.md
10_TESTING_VALIDATION_AND_ROLLOUT.md
```

Decision:

```txt
Backend runtime_operations table is durable source of truth.
Bridge returns operation_id and exposes pollable operation status.
Backend recovers via diagnostics if bridge restarts.
No operation stays running forever.
```

## 3. Capacity Reservations

Patched in:

```txt
01_TARGET_ARCHITECTURE.md
06_BACKEND_MULTICLUSTER_CONTROL_PLANE.md
10_TESTING_VALIDATION_AND_ROLLOUT.md
```

Decision:

```txt
Backend creates short-lived capacity_reservations before agent create.
Routing subtracts active reservations from bridge headroom.
Reservations commit, release, or expire.
```

## 4. Terraform Plan/Apply Safety

Patched in:

```txt
02_IMPLEMENTATION_MASTER_PLAN.md
04_GITHUB_ACTIONS_BOOTSTRAP_PIPELINE.md
```

Decision:

```txt
Use terraform plan -out=tfplan.
Apply exactly the saved plan artifact from same workflow run.
No destroy path in provision workflow.
Destroy requires separate break-glass workflow.
```

## 5. JWT Bridge Auth Claims

Patched in:

```txt
06_BACKEND_MULTICLUSTER_CONTROL_PLANE.md
08_SECURITY_SECRETS_AND_AUTH.md
```

Decision:

```txt
Backend signs short-lived JWT.
Bridge verifies signature, cluster_id, action, aud, iss, exp, and path consistency.
Private key stays in backend/control plane only.
Public key is deployed to bridges.
```

## 6. Terminal and Log Redaction

Patched in:

```txt
05_BRIDGE_V2_RUNTIME_OPERATOR_SPEC.md
07_ADMIN_SUPPORT_DASHBOARD.md
08_SECURITY_SECRETS_AND_AUTH.md
```

Decision:

```txt
Terminal only enters user's own agent container.
No system namespace, bridge pod, node shell, host shell, or cluster-admin shell.
Terminal sessions are audited.
Admin logs are redacted before display.
```

## 7. Backup/Restore Contract

Patched in:

```txt
09_STORAGE_BACKUPS_AND_CLEANUP.md
10_TESTING_VALIDATION_AND_ROLLOUT.md
```

Decision:

```txt
V1 backup mode is maintenance-window backup.
Backup is valid only after restore test succeeds.
Live zero-downtime backup is not promised yet.
```

## 8. Infrastructure CLI and Remote State

Decision:

```txt
Use OpenTofu, not Terraform CLI.
Commands: tofu init / fmt / validate / plan / apply.
GitHub Actions: opentofu/setup-opentofu@v1.
Do not use hashicorp/setup-terraform or HCP Terraform workspaces.
```

Remote state:

```txt
Cloudflare R2 (S3-compatible, free tier), bucket: hermes-tofu-state.
One state key per cluster: clusters/<id>/tofu.tfstate.
Locking via OpenTofu native use_lockfile = true.
No DynamoDB. No Terraform Cloud. No Hetzner Object Storage.
```

Required secrets added:

```txt
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_ENDPOINT
```

Implementation rule:

```txt
OpenTofu creates clusters and DNS.
GitHub Actions bootstraps/deploys bridge.
Bridge creates customer agent runtimes.
OpenTofu must not create dynamic customer agent pods.
```

## 9. Bridge Namespace

Decision:

```txt
All new clusters (kh-test, eu-1, eu-2) use hermes-system as the bridge namespace.
hermes-test legacy cluster continues to use hermes-bridge — do not change it.
```

Impact:

```txt
runtime-node-core chart: bridgeNamespace default changed to hermes-system.
deploy/ raw manifests: sed-substituted at bootstrap time — namespace: hermes-bridge → namespace: hermes-system.
Resource names (name: hermes-bridge) are unchanged.
validate-cluster.sh targets hermes-bridge — correct for hermes-test, not used for kube-hetzner clusters.
```

## Final Verdict

The architecture is now implementation-ready as a phased plan, provided the coding agent follows the PR boundaries and proves everything on `kh-test` before production clusters.
