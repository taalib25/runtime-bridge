# HermesCloud Multi-Cluster Runtime Revamp Documentation Pack

This documentation pack defines the migration from the current `hetzner-k3s` CLI provisioning path to a `kube-hetzner` + Terraform/OpenTofu runtime cluster factory.

The goal is not to replace the bridge. The goal is to make infrastructure repeatable while keeping the bridge as the stable runtime operator for Hermes agents.

## Final Architecture in One Line

```txt
Terraform/kube-hetzner creates clusters and cluster DNS.
GitHub Actions bootstraps the cluster, deploys the bridge, validates it, and registers it.
Bridge manages Hermes agent runtimes inside each cluster.
HermesCloud backend manages product state, routing, billing, operations, and support.
```

## Non-Negotiable Rules

```txt
1. Do not destroy the existing hetzner-k3s cluster until kube-hetzner is proven.
2. Do not let Terraform create customer Hermes agent pods.
3. Do not let the backend talk directly to Kubernetes for customer runtime operations.
4. Run one bridge per runtime cluster.
5. Backend talks to bridges over HTTPS only.
6. Bridge manages agent Helm releases, PVCs, Secrets, ConfigMaps, Services, Ingresses, logs, terminal, and diagnostics.
7. Terraform manages cluster infrastructure and Cloudflare DNS.
8. GitHub Actions glues Terraform, bridge deploy, validation, and backend registration together.
9. Backend operation state is durable. Bridge in-memory state is never the source of truth.
10. Backups are not considered complete until restore is tested.
```

## Read Order

1. `00_FOR_TAALIB_MANUAL_SETUP.md` — What Taalib must manually configure.
2. `01_TARGET_ARCHITECTURE.md` — Final architecture and mental model.
3. `02_IMPLEMENTATION_MASTER_PLAN.md` — Multi-PR plan for the coding agent.
4. `03_TERRAFORM_KUBE_HETZNER_CLOUDFLARE.md` — Terraform, kube-hetzner, node pools, DNS, TLS.
5. `04_GITHUB_ACTIONS_BOOTSTRAP_PIPELINE.md` — Provision, bridge deploy, validation, registration.
6. `05_BRIDGE_V2_RUNTIME_OPERATOR_SPEC.md` — Bridge revamp requirements.
7. `06_BACKEND_MULTICLUSTER_CONTROL_PLANE.md` — DB schema, routing, operations, capacity reservations.
8. `07_ADMIN_SUPPORT_DASHBOARD.md` — Admin pages for support/debugging.
9. `08_SECURITY_SECRETS_AND_AUTH.md` — JWT bridge auth, RBAC, TLS, terminal audit, log redaction.
10. `09_STORAGE_BACKUPS_AND_CLEANUP.md` — PVCs, backups, restore testing, cleanup.
11. `10_TESTING_VALIDATION_AND_ROLLOUT.md` — kh-test proof, rollout, rollback, cutover.
12. `11_CODING_AGENT_HANDOFF_PROMPT.md` — Copy-paste prompt for the coding agent.
13. `12_PATCHED_CONTRACTS_SUMMARY.md` — Summary of contracts added after grill review.
14. `CONTEXT.md` — Glossary and canonical language.
15. `docs/adr/*.md` — Accepted architecture decisions.

## Migration Strategy

```txt
Existing hetzner-k3s cluster = fallback.
New kube-hetzner kh-test cluster = proof.
New eu-1/eu-2 kube-hetzner clusters = future standard.
Retire old cluster only after bridge, backend, support dashboard, backups, and agent lifecycle are proven.
```

## Implementation Safety

This must be implemented as multiple PRs. Do not bundle Terraform, backend, bridge, admin UI, backups, and rollout into one PR.

The first real proof is `kh-test`, not production.
