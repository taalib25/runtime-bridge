# Infrastructure Overview — Hermes Runtime Operator

Everything about how the cluster works, how it scales, and where things are going.

---

## 1. Current Setup (Today)

You have **one Hetzner server** doing everything:

```
hermes-test-master1  (178.104.185.60)
Type: cpx22  —  2 vCPU, 4GB RAM  —  Falkenstein, Germany (fsn1)
│
├── Kubernetes roles: control-plane + etcd + master   ← the brain
│   ├── API server      (receives kubectl / bridge commands)
│   ├── Scheduler       (decides where pods go)
│   └── etcd            (database storing all cluster state)
│
└── Workspace pods running on the same machine:       ← the muscle
    ├── ws-2d434ac4914de483
    ├── ws-920dd34ce0655f6d
    └── ws-c02776d5e18d139b
```

The brain and the workspace pods share the same 2 vCPU and 4GB RAM.
This is fine for testing but will get crowded as workspaces grow.

You also have a second server, `taalib-hermes` (65.109.232.178, cx23, Helsinki),
which is **not part of the Kubernetes cluster** — it's a standalone VM.

---

## 2. What Masters and Workers Actually Are

### Master Node — The Brain

The master runs three things:

| Process | What it does |
|---|---|
| API server | Receives all commands — from kubectl, the bridge, the scheduler |
| Scheduler | Decides which worker node each pod goes on |
| etcd | The database — stores everything (pod state, configs, secrets) |

Masters do **not** run workspace pods (when configured correctly).
They are pure control plane — they coordinate but don't do the actual work.

### Worker Node — The Muscle

Workers are simple. They run one process (the kubelet) whose only job is:

```
Receive instruction from master: "run this pod"
     ↓
Tell containerd: "start this container"
     ↓
Report back: "pod is running / crashed / restarting"
```

Workers have no intelligence. They just run what they're told.
All decision-making happens on the master.

A single `cpx32` worker (4 vCPU, 8GB RAM) can host roughly **4–8 workspace pods**
depending on how much CPU and RAM each workspace uses.

### Why Masters Don't Run Pods

```yaml
schedule_workloads_on_masters: false   # masters stay clean
```

If workspace pods compete with etcd for RAM, etcd gets slow.
A slow etcd means the entire cluster becomes unresponsive.
Keeping masters clean prevents this.

Exception: on a single-node setup (like you have now) the master runs everything
because there are no separate worker nodes.

---

## 3. Masters — Why You Need 3 for High Availability

### The Quorum Problem

etcd uses a voting system. To accept any write (creating a pod, updating a secret),
**more than half the etcd nodes must agree**. This is called quorum.

| Masters | Quorum needed | Servers you can lose |
|---|---|---|
| 1 | 1 of 1 | 0 — if it goes down, cluster is down |
| 2 | 2 of 2 | 0 — still no fault tolerance |
| **3** | **2 of 3** | **1 — one server can die, cluster keeps running** |
| 5 | 3 of 5 | 2 |

**2 masters is actually worse than 1** — you doubled the cost but gained no fault
tolerance because losing either one still breaks quorum.
The minimum meaningful HA number is always **3**.

### What Happens When a Master Goes Down

With 1 master:
```
Master dies
  → etcd gone
  → Kubernetes API stops responding
  → Bridge can't create / check / delete workspaces
  → Existing workspace pods keep running (already scheduled)
  → Nothing new can happen until master recovers (~2-3 min reboot)
```

With 3 masters across 3 datacenters:
```
fsn1 datacenter goes down entirely
  → hel1 + nbg1 still have 2 of 3 masters → quorum maintained
  → Kubernetes API keeps working
  → Bridge keeps working
  → Users notice nothing
```

### Master Locations Don't Affect Pod Locations

Masters spread across `fsn1 / hel1 / nbg1` does NOT mean workspace pods
spread across those cities. Masters are spread for **control plane fault tolerance only**.

Where pods land depends entirely on where your **worker nodes** are.

---

## 4. The hetzner-k3s CLI

A single tool that provisions a full Kubernetes cluster on Hetzner from one YAML file.

### The Three Commands

```bash
# Create a cluster (5–10 minutes)
hetzner-k3s create --config cluster-config.yaml

# Delete a cluster — removes all Hetzner VMs (destructive)
hetzner-k3s delete --config cluster-config.yaml

# Rolling upgrade of k3s version across all nodes (zero-downtime)
hetzner-k3s upgrade --config cluster-config.yaml --new-k3s-version v1.32.1+k3s1
```

### What It Does When You Run `create`

```
Step 1 — Hetzner API: create VMs
  ├── Create master VM(s)
  ├── Create worker VM(s)
  ├── Create private network (10.0.0.0/16)
  ├── Attach all VMs to private network
  └── Set up Hetzner firewall rules

Step 2 — SSH into master-1
  ├── Install k3s in "server" mode
  ├── API server, scheduler, etcd all start
  └── Generate a join token

Step 3 — SSH into master-2 and master-3 (if instance_count: 3)
  ├── Install k3s with the join token
  └── Join the etcd cluster — now 3 masters share the database

Step 4 — SSH into each worker
  ├── Install k3s in "agent" mode (kubelet only, no control plane)
  └── Join using the same token — master registers this worker

Step 5 — Install addons via kubectl
  ├── Hetzner CSI driver       (persistent storage / PVCs)
  ├── Cloud Controller Manager  (LoadBalancer services get real IPs)
  └── Cluster Autoscaler        (auto-add/remove worker nodes)

Step 6 — Write kubeconfig to your machine
  └── Ready to use kubectl / helm immediately
```

### Autoscaler — Workers Scale Automatically

```
2 workspaces → worker-1 is half full, no action

New workspace → worker-1 is full → pod stays Pending

Autoscaler sees Pending pod within ~30 seconds
  → Calls Hetzner API: create new worker VM
  → VM boots, joins cluster (~2 min)
  → Pod scheduled on new worker

Later — workspaces deleted, workers idle for 10 minutes
  → Autoscaler drains pods off idle worker
  → Calls Hetzner API: delete the VM
  → You stop paying for it
```

You set the bounds:
```yaml
autoscaling:
  min_instances: 1    # always keep at least 1 worker warm
  max_instances: 20   # never exceed 20 workers
```

Cost at low traffic: pay for 1 worker (~€8/month).
Cost at peak: might temporarily have 15 workers (~€120/month), scales back down automatically.

---

## 5. Adding More Servers — Two Options

### Option A — Add Workers to the Existing Cluster (Simplest)

New Hetzner VMs join your existing cluster as worker nodes.
The master stays where it is. Pods spread across all workers.

```
BEFORE (today)
hermes-test-master1  [master + ws-aaa + ws-bbb + ws-ccc]

AFTER adding worker nodes
hermes-test-master1  [master only]
worker-node-1        [ws-aaa + ws-bbb + ws-ddd]
worker-node-2        [ws-ccc + ws-eee + ws-fff]
```

Same cluster. Same bridge. Same DNS. Just more capacity.
`taalib-hermes` (65.109.232.178) could be added as a worker right now
without creating anything new — it's already sitting idle.

### Option B — Create a Whole New Cluster (Multi-region)

Separate Hetzner servers, separate Kubernetes, separate bridge URL.
Used when you need different geographic regions or full isolation between tenants.

---

## 6. The Bridge — One Per Cluster or One Central?

### What the Bridge Does

The bridge is a Go HTTP service. It receives REST calls from your backend
and translates them into Helm/Kubernetes operations against the cluster.

```
Backend: POST /v1/instances/tenant-xxx
     ↓
Bridge: helm install hermes-agent (against this cluster's API)
     ↓
Kubernetes: schedules pod on a worker node
```

### Technically One Bridge Can Control Multiple Clusters

A Kubernetes client just needs a kubeconfig pointing to the right API server.
One bridge could hold multiple kubeconfigs and route based on cluster ID.

```
Central Bridge (one server)
  ├── kubeconfig-eu1  →  api.eu1-cluster:6443
  ├── kubeconfig-us1  →  api.us1-cluster:6443
  └── kubeconfig-ap1  →  api.ap1-cluster:6443
```

### Why One Bridge Per Cluster Makes Sense at Scale

| Concern | One central bridge | One bridge per cluster |
|---|---|---|
| Latency (exec/terminal) | High — bridge in Germany, cluster in Singapore = 180ms per call | Low — bridge next to its cluster |
| Blast radius | Bridge goes down → all clusters affected | Bridge goes down → only that cluster |
| Auth complexity | One bridge secret routes to many clusters | One secret per cluster, simple |
| Cost | Free (reuse existing server) | One small server per cluster (~€4/month) |

### Decision: What to Do at Each Stage

| Stage | Approach |
|---|---|
| Now — test, few users | **Central bridge** — no extra cost, simpler |
| Multiple regions, real users | **One bridge per cluster** — latency matters, isolation matters |

---

## 7. Multi-Cluster Architecture (Future)

When you do go multi-cluster, each cluster is fully independent:

```
Your Backend
     │
     ├── Cluster Registry (DB table)
     │   ┌─────────────────────────────────────────────────┐
     │   │ id   │ bridge_url                  │ region     │
     │   │ eu1  │ bridge-eu1.hermeshq.net     │ eu         │
     │   │ us1  │ bridge-us1.hermeshq.net     │ us         │
     │   │ ap1  │ bridge-ap1.hermeshq.net     │ ap         │
     │   └─────────────────────────────────────────────────┘
     │
     ▼                        ▼                        ▼
bridge-eu1.hermeshq.net  bridge-us1.hermeshq.net  bridge-ap1.hermeshq.net
EU Cluster               US Cluster               AP Cluster
(masters: fsn1/hel1/nbg1) (masters: ash)          (masters: sin)
workers: fsn1            workers: ash             workers: sin
```

Backend flow when creating a workspace:
```
1. User requests workspace in "eu" region
2. Backend looks up cluster registry: eu → eu1, bridge-eu1.hermeshq.net
3. Backend generates workspace ID: tenant-abc123
4. Backend calls: POST bridge-eu1.hermeshq.net/v1/instances/tenant-abc123
5. Bridge runs helm install on the EU cluster
6. Backend stores: workspaces.cluster_id = "eu1"
7. All future calls for this workspace go to bridge-eu1
```

### Cluster Registry DB Schema

```sql
CREATE TABLE clusters (
  id             TEXT PRIMARY KEY,     -- "eu1", "us1", "ap1"
  bridge_url     TEXT NOT NULL,        -- "https://bridge-eu1.hermeshq.net"
  bridge_secret  TEXT NOT NULL,        -- X-Bridge-Secret value (keep private)
  region         TEXT NOT NULL,        -- "eu", "us", "ap"
  location       TEXT,                 -- "fsn1", "ash", "sin"
  active         BOOLEAN DEFAULT true, -- false = draining, no new workspaces
  created_at     TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 8. DNS — How Each Cluster Gets Its Domain

All DNS for `hermeshq.net` is on Cloudflare. Adding a new cluster domain is one API call:

```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records" \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "A",
    "name": "bridge-eu1.hermeshq.net",
    "content": "178.104.185.60",
    "proxied": true
  }'
```

Workspace URLs use the existing wildcard (`*.hermeshq.net`) which already covers
`tenant-xxx.hermeshq.net` on any cluster. No DNS changes needed for workspaces.

---

## 9. Provisioning a New Cluster — Fully Automated

The entire process (servers → DNS → bridge → smoke test) runs as a single
GitHub Actions workflow triggered manually:

```
You trigger: workflow_dispatch
  inputs: cluster_id=eu2, location=fsn1, region=eu

Step 1: Generate cluster-config.yaml from inputs
Step 2: hetzner-k3s create  (5–10 min — creates VMs, installs k3s)
Step 3: Get master IP from Hetzner API
Step 4: Create DNS record in Cloudflare (bridge-eu2.hermeshq.net → IP)
Step 5: Deploy bridge to new cluster
  ├── Create hermes-bridge namespace
  ├── Generate random bridge auth secret
  ├── Build + load bridge image via SSH
  ├── Sync hermes-agent chart ConfigMap
  └── Apply deployment manifests
Step 6: Smoke test /healthz
Step 7: Print cluster registry entry
  └── id: eu2
      bridge_url: https://bridge-eu2.hermeshq.net
      bridge_secret: <generated>
      region: eu
```

Required GitHub secrets: `HCLOUD_TOKEN`, `CF_TOKEN`, `CF_ZONE_ID`, `HETZNER_SSH_PRIVATE_KEY`

Full workflow file: `.github/workflows/provision-cluster.yml` (to be created)

---

## 10. Cost Reference

### Per Cluster — Development (1 master, 1 worker)

| Component | Type | Cost/month |
|---|---|---|
| Master | cpx22 (2 vCPU, 4GB) | €4.15 |
| Worker (min 1) | cpx32 (4 vCPU, 8GB) | €8.31 |
| **Total** | | **~€12/month** |

### Per Cluster — Production HA (3 masters, 2 workers minimum)

| Component | Type | Count | Cost/month |
|---|---|---|---|
| Masters | cpx22 | 3 | €12.45 |
| Workers (min) | cpx32 | 2 | €16.62 |
| Load balancer | LB11 | 1 | €5.39 |
| **Total** | | | **~€35/month** |

Auto-scaled workers are billed per hour (~€0.011/hr for cpx32).
At peak you pay more, at low traffic the autoscaler removes idle nodes.

---

## 11. Scaling Path — Where Things Are Going

```
Stage 1 — Now (test)
  1 server (hermes-test-master1)
  Master + workers on same node
  Central bridge
  €4/month

Stage 2 — Growing
  Add taalib-hermes as a worker node to existing cluster
  Master stays on hermes-test-master1
  Workspace pods spread across both servers
  No new cost (server already exists)

Stage 3 — Production (single region)
  Upgrade to 3 masters for HA (hetzner-k3s handles the join)
  Dedicated worker pool with autoscaling
  ~€35/month

Stage 4 — Multi-region
  New cluster per region (EU / US / AP)
  One bridge per cluster OR central bridge routing by kubeconfig
  Cluster registry in backend DB
  ~€35/month × number of active clusters
```

---

## 12. Summary — Everything in One View

| Thing | Current | With Workers Added | Full Multi-cluster |
|---|---|---|---|
| Servers | 1 | 2+ | 3+ per cluster |
| Masters | 1 (does everything) | 1 (brain only) | 3 per cluster (HA) |
| Workers | 0 (master runs pods) | 1+ dedicated | Pool per cluster |
| Bridge | 1 on master | 1 on master | 1 per cluster or central |
| Clusters | 1 | 1 | Many |
| Regions | EU only | EU only | EU + US + AP |
| Monthly cost | ~€4 | ~€12 | ~€35 × clusters |
