# Scaling Plan — Hermes Runtime Operator

## How It Works Right Now

One Hetzner server (`178.104.185.60`) does everything — it is the master,
the worker, and runs the bridge. The CI deploys the bridge to that one server
by SSHing into its IP directly.

```
178.104.185.60  (cpx22 — 2 vCPU, 4GB RAM)
├── Master (brain)      — Kubernetes API, scheduler, etcd
├── Worker (muscle)     — workspace pods run here
└── hermes-bridge pod   — your backend calls this to create/manage workspaces
```

Cost: ~€4/month. Capacity: roughly 4–8 workspaces before it gets crowded.

---

## The Target Architecture

Each cluster is fully independent. Each has its own master, its own workers,
its own bridge, and its own domain. Your backend has a cluster registry that
tracks them all and picks the right one when creating a workspace.

```
Your Backend
     │
     ├── Cluster Registry
     │   ├── eu1 → bridge-eu1.hermeshq.net  (Germany)
     │   ├── us1 → bridge-us1.hermeshq.net  (US)
     │   └── ap1 → bridge-ap1.hermeshq.net  (Asia)
     │
     ▼               ▼               ▼
  EU Cluster      US Cluster      AP Cluster
  ├── bridge      ├── bridge      ├── bridge
  ├── master      ├── master      ├── master
  └── workers     └── workers     └── workers
     (pods)          (pods)          (pods)
```

The bridge lives inside the cluster it controls. It never talks to other clusters.
Your backend is the only thing that knows about all of them.

---

## Stage 1 — Add taalib-hermes as a Worker (Do This Now)

You already have a second server sitting idle (`taalib-hermes`, 65.109.232.178).
It is not part of the cluster yet. Adding it as a worker costs nothing extra and
immediately doubles your pod capacity.

```
BEFORE
178.104.185.60  [master + bridge + ws-aaa + ws-bbb + ws-ccc]

AFTER
178.104.185.60  [master + bridge]
65.109.232.178  [worker — workspace pods move here]
```

**How to do it:**

```bash
# SSH into the new server and join it to the existing cluster
# Get the join token from the master
JOIN_TOKEN=$(ssh root@178.104.185.60 \
  "cat /var/lib/rancher/k3s/server/node-token")

# On the new server — install k3s as an agent (worker only)
ssh root@65.109.232.178 "
  curl -sfL https://get.k3s.io | \
    K3S_URL=https://178.104.185.60:6443 \
    K3S_TOKEN=$JOIN_TOKEN \
    sh -
"

# Verify it joined
ssh root@178.104.185.60 \
  "KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl get nodes"
```

Done. Kubernetes automatically starts scheduling new workspace pods on the new
worker. Existing pods stay where they are until restarted.

**What this gives you:**
- 4 vCPU + 8GB RAM added instantly (cx23 specs)
- Master is no longer competing with workspace pods for resources
- No new cost — server is already paid for

---

## Stage 2 — First Real Cluster (When You Need a New Region or More Capacity)

When `taalib-hermes` fills up or you need a different region, spin up a full
new cluster using `hetzner-k3s`. Each new cluster gets its own bridge and domain.

**The four things that happen per new cluster:**

```
1. hetzner-k3s create     → Hetzner VMs provisioned, k3s installed (~10 min)
2. Cloudflare API call    → bridge-eu2.hermeshq.net → new cluster IP
3. CI deploys bridge      → same bridge binary, deployed to new cluster
4. Backend registry entry → { id: "eu2", bridge_url: "...", secret: "..." }
```

**Cluster config for a lean new cluster:**

```yaml
cluster_name: hermes-eu2
k3s_version: v1.32.0+k3s1

masters_pool:
  instance_type: cpx22     # €4/month — brain only
  instance_count: 1         # no HA yet, add 3 when traffic justifies it
  locations:
    - fsn1

worker_node_pools:
  - name: hermes-workers
    instance_type: cpx32   # €8/month — runs workspace pods
    instance_count: 1
    location: fsn1
    autoscaling:
      enabled: true
      min_instances: 1
      max_instances: 20    # scales automatically as workspaces grow
```

Cost: ~€12/month per cluster to start. Workers auto-scale so you only pay
for capacity actually in use.

---

## Stage 3 — Automate Everything (GitHub Actions)

Right now CI hardcodes `178.104.185.60`. Change it to a matrix so every push
to `main` deploys the bridge to all clusters simultaneously.

```yaml
# .github/workflows/bridge.yml — change the deploy job to:
strategy:
  matrix:
    cluster:
      - { ip: "178.104.185.60", name: "eu1", host: "bridge-eu1.hermeshq.net" }
      - { ip: "NEW_IP",         name: "eu2", host: "bridge-eu2.hermeshq.net" }
```

Adding a new cluster = adding one line to this matrix + pushing.
All clusters stay on the same bridge version automatically.

**To spin up a brand new cluster end-to-end, a separate workflow handles it:**

```
Trigger: workflow_dispatch
Inputs:  cluster_id, location, region

Steps:
  1. Generate cluster-config.yaml from inputs
  2. hetzner-k3s create  (VMs + k3s)
  3. Get master IP from Hetzner API
  4. Create DNS record via Cloudflare API
  5. Deploy bridge (same steps as normal CI)
  6. Print cluster registry entry → add to backend DB
```

One button click in GitHub → fully working cluster with domain in ~15 minutes.

---

## Stage 4 — High Availability Masters (When You Have Real Users)

A single master means if that server reboots, the bridge stops working for 2–3 minutes.
Workspace pods keep running but nothing new can happen. For most early-stage products
this is acceptable.

When uptime matters, upgrade to 3 masters:

```yaml
masters_pool:
  instance_count: 3        # was 1
  locations:
    - fsn1
    - hel1
    - nbg1
```

Run `hetzner-k3s create` again — it adds the two new master VMs and joins them
to the existing etcd cluster. Workspaces keep running during the upgrade.

With 3 masters spread across 3 Hetzner datacenters, one full datacenter can go
offline and the cluster keeps working. Cost goes from €4/month to €12/month for masters.

---

## Cluster Registry — What the Backend Needs

A simple DB table. The backend reads this to pick a cluster when creating a workspace.

```sql
CREATE TABLE clusters (
  id            TEXT PRIMARY KEY,      -- "eu1", "us1", "ap1"
  bridge_url    TEXT NOT NULL,         -- "https://bridge-eu1.hermeshq.net"
  bridge_secret TEXT NOT NULL,         -- X-Bridge-Secret header value
  region        TEXT NOT NULL,         -- "eu", "us", "ap"
  active        BOOLEAN DEFAULT true   -- false = no new workspaces, draining
);
```

Workspace creation flow:
```
User requests workspace in "eu"
  → SELECT * FROM clusters WHERE region = "eu" AND active = true LIMIT 1
  → POST bridge-eu1.hermeshq.net/v1/instances/tenant-xxx
  → Store: workspaces.cluster_id = "eu1"
  → All future calls for this workspace go to eu1's bridge
```

---

## Cost at Each Stage

| Stage | Setup | Monthly cost |
|---|---|---|
| Now | 1 server, everything on it | ~€4 |
| Stage 1 | Add taalib-hermes as worker | €0 extra (already paid) |
| Stage 2 | New cluster (1 master + 1 worker) | +€12/cluster |
| Stage 3 | Automation only — no new servers | €0 extra |
| Stage 4 | HA masters (3 instead of 1) | +€8/cluster |
| At scale | 3 masters + autoscaled workers | ~€35 base + ~€0.011/hr per extra worker |

---

## The Order of Operations

```
Today
  └── Add taalib-hermes as a worker node  (Stage 1)

When taalib-hermes fills up OR you need a new region
  └── hetzner-k3s create new cluster  (Stage 2)
  └── Add to CI matrix  (Stage 3)
  └── Add to backend cluster registry

When a cluster gets real user traffic
  └── Upgrade from 1 master to 3  (Stage 4)

Repeat Stage 2–3 per new region as you expand
```
