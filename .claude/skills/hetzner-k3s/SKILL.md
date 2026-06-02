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

## Common workflows

### Add / scale a worker pool
1. Edit `worker_node_pools` in the config (add new pool or increase `instance_count`)
2. `hetzner-k3s create --config cluster-config.yaml` — idempotent, only provisions the delta

### Scale down a pool
1. Reduce `instance_count` in config
2. Drain + delete extra nodes: `kubectl drain <node> --ignore-daemonsets --delete-emptydir-data && kubectl delete node <node>`
3. Delete the instance in Hetzner Console (Cloud Controller Manager may do this automatically)
4. `hetzner-k3s create --config cluster-config.yaml` to reconcile

### Replace a broken node
1. `kubectl drain <node> && kubectl delete node <node>`
2. Delete the VM from Hetzner Console
3. `hetzner-k3s create --config cluster-config.yaml` — recreates the missing node

### Convert single-master to HA
1. Increase `masters_pool.instance_count` to 3 and add locations (fsn1/hel1/nbg1)
2. `hetzner-k3s create --config cluster-config.yaml`

### Deploy bridge to a new cluster
1. `kubectl create namespace hermes-bridge`
2. `kubectl create secret generic bridge-auth --namespace hermes-bridge --from-literal=secret=$(openssl rand -hex 32)`
3. `kubectl apply -f deploy/` (RBAC + service + deployment)
4. Add DNS A record in Cloudflare pointing `bridge-<name>.hermeshq.net` → node/LB IP
5. Register cluster in backend: `POST /api/admin/clusters`

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
