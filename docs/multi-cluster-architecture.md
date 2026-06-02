# Multi-Cluster Architecture — Hermes Runtime Operator

## Overview

This document explains how Hermes scales across multiple Hetzner clusters, how each cluster gets its own bridge and domain, and how the entire provisioning process can be automated from a single GitHub Actions workflow.

---

## 1. What We Have Today (Single Cluster)

Right now everything runs on one Hetzner server:

```
Your Backend
     │
     │  HTTP (X-Bridge-Secret)
     ▼
bridge.hermeshq.net  →  178.104.185.60  (hermes-test-master1, fsn1, Germany)
                              │
                              └── k3s (single node)
                                    ├── hermes-bridge pod
                                    └── tenant-xxxx pods  (one per workspace)
```

**Problems with this:**
- Single point of failure — if the server goes down, every workspace goes down
- No geographic distribution — all workspaces land in Germany regardless of where the user is
- Limited capacity — one `cpx22` (2 cores, 4GB RAM) can only hold so many workspaces
- No isolation between tenants on different plans/regions

---

## 2. What We Are Building (Multi-Cluster)

Each cluster is completely independent. Each has its own servers, its own bridge, and its own domain. The backend has a **cluster registry** that tracks where each cluster lives. When the backend creates a workspace for a user, it picks the right cluster and calls that cluster's bridge directly.

```
Your Backend
     │
     ├──────────────────────────────────────────────────┐
     │                                                  │
     │  Cluster Registry (DB)                           │
     │  ┌──────────────────────────────────────────┐   │
     │  │ id    │ bridge_url                │ region│   │
     │  │ eu1   │ bridge-eu1.hermeshq.net   │ eu    │   │
     │  │ us1   │ bridge-us1.hermeshq.net   │ us    │   │
     │  │ ap1   │ bridge-ap1.hermeshq.net   │ ap    │   │
     │  └──────────────────────────────────────────┘   │
     │                                                  │
     ▼                           ▼                      ▼
bridge-eu1.hermeshq.net  bridge-us1.hermeshq.net  bridge-ap1.hermeshq.net
        │                        │                       │
   EU Cluster               US Cluster              AP Cluster
   (fsn1/hel1/nbg1)         (ash)                   (sin)
        │                        │                       │
   k3s + bridge             k3s + bridge            k3s + bridge
   workspace pods           workspace pods          workspace pods
```

**Each cluster is identical in structure.** The only things that differ are:
- The Hetzner server locations (EU / US / Asia-Pacific)
- The bridge subdomain (`bridge-eu1`, `bridge-us1`, etc.)
- The bridge auth secret (unique random secret per cluster)
- The cluster name stored in the registry

---

## 3. The hetzner-k3s CLI — What It Does

`hetzner-k3s` is a CLI tool that provisions a full production-ready Kubernetes cluster on Hetzner Cloud from a single YAML config file. It handles:

- Creating the Hetzner VMs (masters + workers)
- Installing k3s on all nodes
- Setting up private networking between nodes
- Installing the Hetzner CSI driver (for persistent storage / PVCs)
- Installing the Hetzner Cloud Controller Manager (for LoadBalancer services)
- Configuring the Cluster Autoscaler (nodes scale up/down automatically)
- Writing a kubeconfig file you can use immediately with `kubectl`

**The three commands you'll use:**

```bash
# Create a new cluster from a config file
hetzner-k3s create --config cluster-config.yaml

# Delete a cluster (destructive — removes all servers)
hetzner-k3s delete --config cluster-config.yaml

# Upgrade k3s version across all nodes (rolling, zero-downtime)
hetzner-k3s upgrade --config cluster-config.yaml --new-k3s-version v1.32.1+k3s1
```

A cluster takes roughly **5–10 minutes** to provision from scratch. After that you have a fully working Kubernetes cluster you can deploy to immediately.

---

## 4. Cluster Config File — What Each Field Does

This is the YAML file you pass to `hetzner-k3s create`. We already have one at `cluster-config.yaml` in this repo. Here is what each section controls:

```yaml
hetzner_token: "${HCLOUD_TOKEN}"   # Your Hetzner API token (from env var)
cluster_name: hermes-eu1           # Must be unique across all your clusters
kubeconfig_path: "./kubeconfig-eu1" # Where to write the kubeconfig after creation
k3s_version: v1.32.0+k3s1         # Which k3s version to install on all nodes
```

### SSH — How hetzner-k3s accesses the nodes

```yaml
networking:
  ssh:
    port: 22
    public_key_path: "~/.ssh/id_ed25519.pub"
    private_key_path: "~/.ssh/id_ed25519"
```

hetzner-k3s SSHes into each node to install k3s and configure it. The SSH key is installed onto the VM at creation time. In CI this key is stored as a GitHub secret.

### Masters — The Kubernetes control plane nodes

```yaml
masters_pool:
  instance_type: cpx22    # 2 vCPU, 4GB RAM — sufficient for control plane
  instance_count: 3        # Must be odd for HA (1 or 3 in practice)
  locations:
    - fsn1   # Falkenstein, Germany
    - hel1   # Helsinki, Finland
    - nbg1   # Nuremberg, Germany
```

Masters run the Kubernetes API server, scheduler, and etcd database. They do NOT run workspace pods (we set `schedule_workloads_on_masters: false`). Three masters spread across three Hetzner datacenters means the cluster survives a full datacenter outage.

You need an odd number because etcd (the cluster database) uses a majority vote — 3 nodes means 2 must agree, so 1 can fail. With 2 nodes, if one goes down you lose quorum.

### Workers — Where workspace pods actually run

```yaml
worker_node_pools:
  - name: hermes-workers
    instance_type: cpx32        # 4 vCPU, 8GB RAM — sized for workspace pods
    instance_count: 2           # Minimum always-on nodes
    location: fsn1
    autoscaling:
      enabled: true
      min_instances: 2          # Never scale below 2 (always ready)
      max_instances: 20         # Maximum nodes this pool can grow to
```

Workers are where the actual workspace pods (`tenant-xxxx`) run. The Cluster Autoscaler watches pending pods — if a pod can't be scheduled because there's not enough capacity, it automatically provisions a new worker node. When nodes are underutilized for 10+ minutes, it removes them. You pay only for what you use.

`cpx32` gives 4 vCPU and 8GB RAM per node. A single Hermes workspace pod typically uses 0.5–2 vCPU and 512MB–2GB RAM, so a `cpx32` node can comfortably host 4–8 workspaces simultaneously.

---

## 5. DNS — How Each Cluster Gets Its Domain

DNS for `hermeshq.net` is managed by Cloudflare. Every cluster bridge needs a subdomain pointing to that cluster's IP address.

**Pattern:** `bridge-{cluster-id}.hermeshq.net → {cluster master/LB IP}`

Examples:
- `bridge-eu1.hermeshq.net → 178.104.185.60`
- `bridge-us1.hermeshq.net → 5.78.123.45`
- `bridge-ap1.hermeshq.net → 188.34.56.78`

Workspace URLs within a cluster follow the existing wildcard:
- `tenant-abc123.hermeshq.net` (resolved by the same Cloudflare wildcard `*.hermeshq.net`)

To create a DNS record via the Cloudflare API:

```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records" \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "A",
    "name": "bridge-eu1.hermeshq.net",
    "content": "178.104.185.60",
    "ttl": 1,
    "proxied": true
  }'
```

`proxied: true` means traffic goes through Cloudflare (DDoS protection, TLS termination at edge). This is the same setup as the current bridge.

---

## 6. What Gets Deployed to Each Cluster

After `hetzner-k3s create` finishes, the automation deploys these to the new cluster:

```
New Cluster
├── hermes-bridge namespace
│   ├── bridge-auth secret          (random 64-char hex, unique per cluster)
│   ├── hermes-agent-chart ConfigMap (the Helm chart for workspace pods)
│   ├── hermes-bridge Deployment    (the Go bridge binary)
│   └── Traefik IngressRoute        (routes bridge-eu1.hermeshq.net → bridge pod)
│
└── hermes-agent chart (pre-pulled image on worker nodes)
```

The bridge binary is identical across all clusters. The only per-cluster differences are:
- `CLUSTER_NAME` env var (so the bridge knows its own identity)
- The auth secret (unique per cluster, stored in the cluster registry)
- The Traefik IngressRoute hostname

---

## 7. Cluster Registry — How the Backend Tracks Everything

The backend needs a table that maps cluster IDs to bridge URLs and secrets. The bridge never knows about this table — it only knows about its own workspaces.

```sql
CREATE TABLE clusters (
  id             TEXT PRIMARY KEY,    -- "eu1", "us1", "ap1"
  bridge_url     TEXT NOT NULL,       -- "https://bridge-eu1.hermeshq.net"
  bridge_secret  TEXT NOT NULL,       -- the X-Bridge-Secret value (keep secret)
  region         TEXT NOT NULL,       -- "eu", "us", "ap"
  location       TEXT,                -- "fsn1", "ash", "sin" (Hetzner datacenter)
  active         BOOLEAN DEFAULT true, -- set false to drain before decommissioning
  created_at     TIMESTAMPTZ DEFAULT NOW()
);
```

**Workspace creation flow:**

```
Backend receives: POST /workspaces { userId, region: "eu" }
  │
  ├── Look up cluster registry WHERE region = "eu" AND active = true
  │   → picks "eu1", bridge_url = "https://bridge-eu1.hermeshq.net"
  │
  ├── Generate workspace ID: "tenant-" + random hex
  │
  ├── POST https://bridge-eu1.hermeshq.net/v1/instances/{id}
  │     header: X-Bridge-Secret: {bridge_secret for eu1}
  │     body: { image, imageTag, ... }
  │
  └── Store in DB: workspaces.cluster_id = "eu1", workspaces.instance_id = "tenant-xxx"
```

All future operations on that workspace go to the same cluster's bridge.

---

## 8. Full Automation — Provision a New Cluster in One Command

The entire process of spinning up a new cluster, creating its DNS record, deploying the bridge, and outputting the cluster registry entry can be a single GitHub Actions workflow you trigger manually.

### The workflow (`.github/workflows/provision-cluster.yml`)

```yaml
name: Provision New Cluster

on:
  workflow_dispatch:
    inputs:
      cluster_id:
        description: "Cluster ID (e.g. eu2, us1, ap1)"
        required: true
      location:
        description: "Primary Hetzner location (fsn1, hel1, nbg1, ash, sin)"
        required: true
      region:
        description: "Region label for cluster registry (eu, us, ap)"
        required: true
      master_type:
        description: "Master instance type"
        default: "cpx22"
      worker_type:
        description: "Worker instance type"
        default: "cpx32"

jobs:
  provision:
    runs-on: ubuntu-latest
    steps:
      # ── Step 1: Generate cluster config from inputs ──────────────────────
      - name: Generate cluster config
        run: |
          cat > cluster-config-${{ inputs.cluster_id }}.yaml << EOF
          hetzner_token: "${HCLOUD_TOKEN}"
          cluster_name: hermes-${{ inputs.cluster_id }}
          kubeconfig_path: "./kubeconfig-${{ inputs.cluster_id }}"
          k3s_version: v1.32.0+k3s1
          
          networking:
            ssh:
              port: 22
              public_key_path: "~/.ssh/id_ed25519.pub"
              private_key_path: "~/.ssh/id_ed25519"
            private_network:
              enabled: true
              subnet: 10.0.0.0/16
            cni:
              enabled: true
              mode: flannel
          
          schedule_workloads_on_masters: false
          
          masters_pool:
            instance_type: ${{ inputs.master_type }}
            instance_count: 1      # Start with 1 for dev, 3 for production HA
            locations:
              - ${{ inputs.location }}
          
          worker_node_pools:
            - name: hermes-workers
              instance_type: ${{ inputs.worker_type }}
              instance_count: 1
              location: ${{ inputs.location }}
              autoscaling:
                enabled: true
                min_instances: 1
                max_instances: 20
          
          addons:
            csi_driver:
              enabled: true
            cloud_controller_manager:
              enabled: true
            traefik:
              enabled: true
            cluster_autoscaler:
              enabled: true
          EOF

      # ── Step 2: Create the cluster (5–10 min) ────────────────────────────
      - name: Create k3s cluster
        env:
          HCLOUD_TOKEN: ${{ secrets.HCLOUD_TOKEN }}
        run: |
          hetzner-k3s create --config cluster-config-${{ inputs.cluster_id }}.yaml

      # ── Step 3: Get the master IP from Hetzner API ───────────────────────
      - name: Get cluster IP
        id: get_ip
        env:
          HCLOUD_TOKEN: ${{ secrets.HCLOUD_TOKEN }}
        run: |
          IP=$(curl -sf -H "Authorization: Bearer $HCLOUD_TOKEN" \
            "https://api.hetzner.cloud/v1/servers" | \
            jq -r '.servers[] | select(.name == "hermes-${{ inputs.cluster_id }}-master1") | .public_net.ipv4.ip')
          echo "ip=$IP" >> $GITHUB_OUTPUT
          echo "Master IP: $IP"

      # ── Step 4: Create DNS record in Cloudflare ──────────────────────────
      - name: Create DNS record
        run: |
          curl -sf -X POST \
            "https://api.cloudflare.com/client/v4/zones/${{ secrets.CF_ZONE_ID }}/dns_records" \
            -H "Authorization: Bearer ${{ secrets.CF_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{
              "type": "A",
              "name": "bridge-${{ inputs.cluster_id }}.hermeshq.net",
              "content": "${{ steps.get_ip.outputs.ip }}",
              "ttl": 1,
              "proxied": true
            }'
          echo "DNS: bridge-${{ inputs.cluster_id }}.hermeshq.net → ${{ steps.get_ip.outputs.ip }}"

      # ── Step 5: Deploy bridge to the new cluster ─────────────────────────
      - name: Deploy bridge
        env:
          KUBECONFIG: ./kubeconfig-${{ inputs.cluster_id }}
        run: |
          # Create namespace + auth secret
          kubectl create namespace hermes-bridge
          BRIDGE_SECRET=$(openssl rand -hex 32)
          kubectl create secret generic bridge-auth \
            -n hermes-bridge \
            --from-literal=secret=$BRIDGE_SECRET
          
          # Build and load bridge image
          docker build -t hermes-bridge:latest .
          docker save hermes-bridge:latest | \
            ssh root@${{ steps.get_ip.outputs.ip }} "k3s ctr images import -"
          
          # Sync chart ConfigMap
          tar czf - charts/hermes-agent | \
            ssh root@${{ steps.get_ip.outputs.ip }} "cd /tmp && tar xzf -"
          kubectl create configmap hermes-agent-chart \
            --from-file=/tmp/charts/hermes-agent/ \
            -n hermes-bridge
          
          # Deploy bridge with correct hostname
          CLUSTER_ID="${{ inputs.cluster_id }}" \
          BRIDGE_HOST="bridge-${{ inputs.cluster_id }}.hermeshq.net" \
            envsubst < deploy/deployment.yaml | kubectl apply -f -
          
          kubectl rollout status deployment/hermes-bridge -n hermes-bridge --timeout=120s
          
          # Output cluster registry entry
          echo ""
          echo "════════════════════════════════════════════"
          echo " CLUSTER REGISTRY ENTRY — add to your DB"
          echo "════════════════════════════════════════════"
          echo " id:            ${{ inputs.cluster_id }}"
          echo " bridge_url:    https://bridge-${{ inputs.cluster_id }}.hermeshq.net"
          echo " bridge_secret: $BRIDGE_SECRET"
          echo " region:        ${{ inputs.region }}"
          echo " location:      ${{ inputs.location }}"
          echo "════════════════════════════════════════════"

      # ── Step 6: Smoke test ───────────────────────────────────────────────
      - name: Smoke test
        run: |
          sleep 10  # wait for DNS propagation
          SECRET=$(KUBECONFIG=./kubeconfig-${{ inputs.cluster_id }} \
            kubectl -n hermes-bridge get secret bridge-auth \
            -o jsonpath='{.data.secret}' | base64 -d)
          curl -sf \
            -H "X-Bridge-Secret: $SECRET" \
            "https://bridge-${{ inputs.cluster_id }}.hermeshq.net/healthz"
          echo "Smoke test passed."
```

**GitHub Secrets required:**

| Secret | What it is |
|---|---|
| `HCLOUD_TOKEN` | Hetzner Cloud API token |
| `CF_TOKEN` | Cloudflare API token with DNS:Edit permission |
| `CF_ZONE_ID` | Cloudflare Zone ID for hermeshq.net |
| `HETZNER_SSH_PRIVATE_KEY` | SSH private key for accessing new VMs |

---

## 9. Decommissioning a Cluster

To take a cluster offline gracefully:

```bash
# 1. Mark as inactive in the cluster registry (backend stops sending new workspaces)
UPDATE clusters SET active = false WHERE id = 'eu2';

# 2. Wait for existing workspaces to be migrated/terminated
# (or force-delete them via the bridge)

# 3. Delete the cluster (removes all Hetzner VMs)
hetzner-k3s delete --config cluster-config-eu2.yaml

# 4. Remove the DNS record via Cloudflare API
curl -X DELETE "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records/$RECORD_ID" \
  -H "Authorization: Bearer $CF_TOKEN"

# 5. Remove from cluster registry
DELETE FROM clusters WHERE id = 'eu2';
```

---

## 10. Cost Breakdown Per Cluster

Based on Hetzner pricing (approximate, EUR/month):

| Component | Type | Count | Cost/month |
|---|---|---|---|
| Master nodes | cpx22 (2 vCPU, 4GB) | 1 (dev) or 3 (prod HA) | €4.15 or €12.45 |
| Worker nodes (min) | cpx32 (4 vCPU, 8GB) | 1–2 always-on | €8.31–€16.62 |
| Load balancer | LB11 | 1 | €5.39 |
| Hetzner volumes | per workspace PVC | varies | ~€0.05/GB/month |

**Single dev cluster (1 master + 1 worker):** ~€18/month
**Production HA cluster (3 masters + 2 workers):** ~€35/month

Auto-scaled worker nodes are billed per-hour (~€0.011/hour for cpx32), so you only pay for capacity actually used.

---

## 11. Summary — What Changes and What Stays the Same

| Thing | Today | Multi-cluster |
|---|---|---|
| Bridge binary | Same binary for everything | Same binary, one per cluster |
| Bridge config | Single hardcoded kubeconfig | Each bridge uses its own local k3s |
| DNS | `bridge.hermeshq.net` | `bridge-{id}.hermeshq.net` per cluster |
| Workspace URLs | `tenant-xxx.hermeshq.net` | Same (wildcard covers all clusters) |
| Backend API calls | Always to same bridge URL | Looks up cluster registry first |
| Secrets management | One bridge-auth secret | One per cluster, stored in registry |
| Provisioning | Manual | Single GitHub Actions workflow trigger |
| Cost | ~€4/month (test) | ~€18–35/month per active cluster |
