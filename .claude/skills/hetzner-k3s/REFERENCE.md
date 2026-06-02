# hetzner-k3s v2 — Full Reference

Source: `vitobotta/hetzner-k3s` fetched via opensrc. Reflects v2.5.0 source.

## CLI flags

### `create`
| Flag | Short | Required | Default | Description |
|---|---|---|---|---|
| `--config` | `-c` | ✓ | — | Path to YAML config file |
| `--skip-current-ip-validation` | — | — | false | Skip validation that current IP is in allowed SSH/API networks |
| `--quiet` | `-q` | — | false | Suppress sponsor message |

### `delete`
| Flag | Short | Required | Default | Description |
|---|---|---|---|---|
| `--config` | `-c` | ✓ | — | Path to YAML config file |
| `--force` | — | — | false | Skip confirmation prompt |
| `--quiet` | `-q` | — | false | Suppress sponsor message |

### `upgrade`
| Flag | Short | Required | Default | Description |
|---|---|---|---|---|
| `--config` | `-c` | ✓ | — | Path to YAML config file |
| `--new-k3s-version` | — | ✓ | — | Target version e.g. `v1.33.0+k3s1` |
| `--force` | — | — | false | Skip confirmation prompt |
| `--quiet` | `-q` | — | false | Suppress sponsor message |

### `run`
| Flag | Short | Required | Default | Description |
|---|---|---|---|---|
| `--config` | `-c` | ✓ | — | Path to YAML config file |
| `--command` | — | ✗ | — | Shell command (mutually exclusive with `--script`) |
| `--script` | — | ✗ | — | Path to script file (mutually exclusive with `--command`) |
| `--instance` | — | — | all nodes | Specific node name to target |

Exactly one of `--command` or `--script` is required.

---

## Complete config schema

```yaml
# ─── Required ────────────────────────────────────────────────────────────────
hetzner_token: "${HCLOUD_TOKEN}"   # env var HCLOUD_TOKEN takes precedence
cluster_name: my-cluster
kubeconfig_path: "./kubeconfig"
k3s_version: v1.32.0+k3s1         # see: hetzner-k3s releases

# ─── Networking ──────────────────────────────────────────────────────────────
networking:
  ssh:
    port: 22
    use_agent: false               # set true if SSH key has a passphrase
    use_private_ip: false          # connect via private IPs
    public_key_path: "~/.ssh/id_ed25519.pub"
    private_key_path: "~/.ssh/id_ed25519"
    # existing_ssh_key_name: "my-key"  # use a pre-existing Hetzner SSH key

  allowed_networks:
    ssh:
      - 0.0.0.0/0
    api:                           # firewalls port 6443
      - 0.0.0.0/0

    # custom_firewall_rules:       # max 50 total rules; built-ins use ~10
    #   - description: "Allow HTTP"
    #     direction: in            # in | out
    #     protocol: tcp            # tcp | udp | icmp | esp | gre
    #     port: "80"               # single, range "30000-32767", or "any"
    #     source_ips:
    #       - 0.0.0.0/0
    # WARNING: any outbound rule switches egress to implicit deny-all

  # node_port_firewall_enabled: true
  # node_port_range: "30000-32767"

  public_network:
    ipv4: true
    ipv6: true

  private_network:
    enabled: true
    subnet: 10.0.0.0/16
    existing_network_name: ""      # blank = auto-create

  cni:
    enabled: true
    encryption: false
    mode: flannel                  # flannel (default) | cilium
    # cilium:
    #   helm_values_path: "./cilium-values.yaml"
    #   chart_version: "v1.17.2"

  # cluster_cidr: 10.244.0.0/16   # pod IP range — cannot change post-create
  # service_cidr: 10.43.0.0/16    # service IP range — cannot change post-create
  # cluster_dns: 10.43.0.10       # CoreDNS IP; must be in service_cidr

# ─── Datastore ───────────────────────────────────────────────────────────────
datastore:
  mode: etcd                       # etcd (embedded, default) | external
  # external_datastore_endpoint: postgres://...

  # etcd:
  #   snapshot_retention: 24
  #   snapshot_schedule_cron: "0 * * * *"
  #   s3_enabled: false
  #   s3_endpoint: ""              # or env ETCD_S3_ENDPOINT
  #   s3_region: ""                # or env ETCD_S3_REGION
  #   s3_bucket: ""                # or env ETCD_S3_BUCKET
  #   s3_access_key: ""            # or env ETCD_S3_ACCESS_KEY
  #   s3_secret_key: ""            # or env ETCD_S3_SECRET_KEY
  #   s3_folder: ""
  #   s3_force_path_style: false

# ─── Scheduling ──────────────────────────────────────────────────────────────
schedule_workloads_on_masters: false   # set true for single-node dev clusters

# ─── Image ───────────────────────────────────────────────────────────────────
# image: ubuntu-24.04             # default; use snapshot ID for custom images
# autoscaling_image: 103908130    # overrides image for autoscaled pools only
# snapshot_os: microos            # required when using MicroOS snapshot

# ─── Masters ─────────────────────────────────────────────────────────────────
masters_pool:
  instance_type: cpx22
  instance_count: 3               # 1 = single master dev; 3 = HA production
  locations:                      # HA: one per location (EU-central only)
    - fsn1
    - hel1
    - nbg1
  # image: ubuntu-24.04
  # labels:
  #   - key: purpose
  #     value: control-plane
  # taints:
  #   - key: node-role
  #     value: master:NoSchedule

# ─── Workers ─────────────────────────────────────────────────────────────────
worker_node_pools:
  - name: static-workers
    instance_type: cpx32
    instance_count: 3
    location: fsn1
    # image: debian-12
    # labels:
    #   - key: pool
    #     value: primary
    # taints:
    #   - key: workload
    #     value: hermes:NoSchedule
    # additional_packages: []
    # additional_pre_k3s_commands: []
    # additional_post_k3s_commands: []

  - name: autoscaled-workers
    instance_type: cpx32
    location: fsn1
    autoscaling:
      enabled: true
      min_instances: 0
      max_instances: 10

# ─── Addons ──────────────────────────────────────────────────────────────────
addons:
  csi_driver:
    enabled: true                  # Hetzner block storage PVs
    manifest_url: "https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.20.2/deploy/kubernetes/hcloud-csi.yml"

  cloud_controller_manager:
    enabled: true                  # auto-provisions Hetzner LBs for Services
    manifest_url: "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.30.1/ccm-networks.yaml"

  system_upgrade_controller:
    enabled: true                  # zero-downtime rolling k3s upgrades
    deployment_manifest_url: "https://github.com/rancher/system-upgrade-controller/releases/download/v0.19.2/system-upgrade-controller.yaml"
    crd_manifest_url: "https://github.com/rancher/system-upgrade-controller/releases/download/v0.19.2/crd.yaml"

  cluster_autoscaler:
    enabled: true                  # required when autoscaling pools defined
    manifest_url: "https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/hetzner/examples/cluster-autoscaler-run-on-master.yaml"
    container_image_tag: "v1.35.0"
    scan_interval: "10s"
    scale_down_delay_after_add: "10m"
    scale_down_delay_after_delete: "10s"
    scale_down_delay_after_failure: "3m"
    max_node_provision_time: "15m"

  traefik:
    enabled: false                 # k3s built-in; HermesCloud deploys its own
  servicelb:
    enabled: false
  metrics_server:
    enabled: false

  # embedded_registry_mirror:
  #   enabled: false               # p2p image distribution; check k3s compat

# ─── Misc ────────────────────────────────────────────────────────────────────
protect_against_deletion: true     # must set false before hetzner-k3s delete

create_load_balancer_for_the_kubernetes_api: false  # set true for HA API LB
# api_server_hostname: k8s.example.com

k3s_upgrade_concurrency: 1         # nodes upgraded in parallel

# additional_packages: []
# additional_pre_k3s_commands: []  # before k3s install
# additional_post_k3s_commands: [] # after k3s install (pool-level overrides root)

# kube_api_server_args: []
# kube_scheduler_args: []
# kube_controller_manager_args: []
# kube_cloud_controller_manager_args: []
# kubelet_args: []
# kube_proxy_args: []
```

---

## Hetzner locations

| Code | City | Country |
|---|---|---|
| fsn1 | Falkenstein | DE |
| hel1 | Helsinki | FI |
| nbg1 | Nuremberg | DE |
| ash | Ashburn, VA | US |
| hil | Hillsboro, OR | US |
| sin | Singapore | SG |

HA masters (3 different locations) only available in EU-central: **fsn1 / hel1 / nbg1**.

---

## Upgrade procedure (two-step — both steps mandatory)

```bash
# Step 1: rolling upgrade via System Upgrade Controller
hetzner-k3s upgrade --config cluster-config.yaml --new-k3s-version v1.33.0+k3s1

# Step 2: monitor until all nodes show new version
watch kubectl get nodes -o wide
watch kubectl get jobs,pods -n system-upgrade

# Step 3 — MANDATORY: re-run create to pin new version for future nodes
# Without this, new nodes provision with the OLD version first, then re-upgrade
hetzner-k3s create --config cluster-config.yaml
```

**If the upgrade stalls:**
```bash
# Clean up plans/jobs and restart controller
kubectl -n system-upgrade delete job --all
kubectl -n system-upgrade delete plan --all
kubectl label node --all plan.upgrade.cattle.io/k3s-server- plan.upgrade.cattle.io/k3s-agent-
kubectl -n system-upgrade rollout restart deployment system-upgrade-controller

# If workers stuck after masters finished, mark masters as done:
kubectl label node <master1> <master2> <master3> plan.upgrade.cattle.io/k3s-server=upgraded
```

---

## HermesCloud bridge deployment (new cluster)

```bash
# 1. Namespace
kubectl create namespace hermes-bridge

# 2. Auth secret (store value as BRIDGE_SECRET GitHub env secret)
kubectl create secret generic bridge-auth \
  --namespace hermes-bridge \
  --from-literal=secret=$(openssl rand -hex 32)

# 3. Deploy bridge manifests
kubectl apply -f deploy/

# 4. Verify
kubectl get pods -n hermes-bridge
kubectl get svc -n hermes-bridge

# 5. DNS — add A record in Cloudflare
# bridge-<name>.hermeshq.net → node/LB IP

# 6. Register with backend
# POST https://api.hermeshq.net/api/admin/clusters
# { cluster_id, bridge_url, bridge_secret, region, status: "active" }
```

---

## HermesCloud cluster config quick-reference

| Setting | hermes-test | hermes-production |
|---|---|---|
| masters | 1 × cpx22 (fsn1) | 3 × cpx22 (fsn1/hel1/nbg1) |
| workers | none (schedule on master) | cpx32 autoscale 0→20 (fsn1 + ash) |
| HA | no | yes |
| kubeconfig | `./kubeconfig-test` | `./kubeconfig` |
| protect_against_deletion | true | true |
| API LB | no | yes |
| Traefik addon | enabled | enabled |
| schedule_workloads_on_masters | true | false |
