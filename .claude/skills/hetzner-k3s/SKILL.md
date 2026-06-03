---
name: hetzner-k3s
description: Manage HermesCloud k3s clusters on Hetzner Cloud using the hetzner-k3s CLI v2.x. Covers full cluster lifecycle: create, delete, upgrade, scale, run commands, and bridge deployment. Use when asked about cluster operations, spinning up new clusters, adding/scaling nodes, upgrading k3s, deploying the bridge to a new cluster, or checking cluster health. Configs live in repo root (cluster-config.yaml, cluster-config-test.yaml).
---

# HermesCloud — Hetzner k3s Cluster Management

Source used to build this skill: `vitobotta/hetzner-k3s` (via opensrc). CLI version in repo: **2.4.8** (latest: 2.5.0).

## Cluster inventory

| Name | Type | IP | Kubeconfig | Config file |
|---|---|---|---|---|
| hermes-test | Single VM cpx22 | 178.104.185.60 | `./kubeconfig-test` | `cluster-config-test.yaml` |
| hermes-production | HA 3×cpx22 masters + cpx32 autoscale workers | LB IP (TBD) | `./kubeconfig` | `cluster-config.yaml` |

## CLI subcommands

| Command | What it does |
|---|---|
| `hetzner-k3s create --config <file>` | Create **or reconcile** cluster — idempotent, safe to re-run |
| `hetzner-k3s delete --config <file>` | Destroy cluster and all its Hetzner resources |
| `hetzner-k3s upgrade --config <file> --new-k3s-version <ver>` | Rolling k3s upgrade via System Upgrade Controller |
| `hetzner-k3s run --config <file> --command <cmd>` | Run shell command on all nodes (or `--instance <name>` for one) |
| `hetzner-k3s run --config <file> --script <path>` | Run a script file on all nodes (or `--instance <name>`) |
| `hetzner-k3s releases` | List available k3s versions |

All subcommands accept `--quiet` / `-q` to suppress the sponsor message. `create` and `upgrade` accept `--force` to skip confirmation prompts.

## Quick commands

```bash
# Token — required for all Hetzner API calls
export HCLOUD_TOKEN="..."

# Create / reconcile
hetzner-k3s create --config cluster-config-test.yaml

# Delete (must set protect_against_deletion: false in config first)
hetzner-k3s delete --config cluster-config-test.yaml --force

# Upgrade k3s — two-step: upgrade then re-pin with create
hetzner-k3s upgrade --config cluster-config-test.yaml --new-k3s-version v1.33.0+k3s1
# Then re-run create so future nodes use the new version:
hetzner-k3s create --config cluster-config-test.yaml

# Run on all nodes
hetzner-k3s run --config cluster-config-test.yaml --command "df -h /"

# Run on one node
hetzner-k3s run --config cluster-config-test.yaml --command "hostname" --instance "hermes-test-master-fsn1-1"

# Run a script on all nodes
hetzner-k3s run --config cluster-config-test.yaml --script ./scripts/fix-ssh.sh

# List k3s versions
hetzner-k3s releases
```

## Helper scripts (`scripts/`)

These wrap the `hetzner-k3s` CLI with HermesCloud-specific glue and guardrails
(typed confirmation, conflict checks, `DRY_RUN`). **Prefer these over raw CLI calls**
for anything that touches infrastructure.

| Script | Wraps | What it adds |
|---|---|---|
| `provision-cluster.sh <config> <name> <region>` | `create` | End-to-end: create cluster → discover IP → GitHub Environment + 5 secrets → registries.yaml → build/push bridge → deploy → smoke test → register with backend. Refuses if cluster/env already exists. `DRY_RUN=true` to preview. |
| `create-cluster.sh <config>` | `create` | Idempotent create/reconcile of an **existing** cluster only |
| `scale-pool.sh <config> <pool> <count>` | `create` | Edits `instance_count`, reconciles; for scale-down, walks you through drain+delete first |
| `upgrade-cluster.sh <config> <version>` | `upgrade`+`create` | Enforces the mandatory two-step, monitors System Upgrade Controller, prints stall recovery |
| `run-on-cluster.sh <config> --command/--script` | `run` | Confirmation prompt when the command looks destructive |
| `destroy-cluster.sh <config>` | `delete` | Checks `protect_against_deletion`, typed confirm, drains bridge from backend, offers to delete the GitHub Environment |
| `health-check.sh <vm-ip>` | — | Read-only node + pod health |
| `cleanup-images.sh` | — | Prune old images on a node |

**New cluster, one command** (see `provision-cluster.sh` header for required env vars):
```bash
export HCLOUD_TOKEN=... GHCR_PAT=... ADMIN_API_SECRET=... SSH_PRIVATE_KEY="$(cat ~/.ssh/id_ed25519)"
DRY_RUN=true ./scripts/provision-cluster.sh cluster-config-eu2.yaml hermes-eu-2 eu  # preview
./scripts/provision-cluster.sh cluster-config-eu2.yaml hermes-eu-2 eu              # do it
```

## Raw CLI workflows (when not using scripts)

### Add / scale a worker pool
Edit `instance_count` in the pool → `hetzner-k3s create --config <file>` (idempotent).
Or use `scripts/scale-pool.sh`.

### Replace a broken node
`kubectl drain <node> && kubectl delete node <node>` → delete the VM in Hetzner Console
→ `hetzner-k3s create --config <file>` recreates it.

### Convert single-master to HA
Increase `masters_pool.instance_count` to 3, add locations (fsn1/hel1/nbg1),
`hetzner-k3s create`.

## Key constraints

- `protect_against_deletion: true` is the default — **must set to `false` in config before `delete`**
- **Upgrade is a two-step**: `upgrade` then `create` — skip the create and new nodes use the old version
- Autoscaling requires a standard OS image (not snapshots)
- HA (3 masters in different locations) only works in EU-central: fsn1 / hel1 / nbg1
- Private networks: max 100 nodes — disable for >100 nodes, use Cilium CNI
- Max 50 firewall rules per firewall — the built-in rules use ~10 slots
- Adding outbound firewall rules switches to implicit deny-all outbound — define egress carefully
- Do **not** enable cluster autoscaler on the test cluster

See [REFERENCE.md](REFERENCE.md) for the complete config schema, all flags, and upgrade troubleshooting.
