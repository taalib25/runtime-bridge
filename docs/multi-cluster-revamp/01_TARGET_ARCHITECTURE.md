# Target Architecture: kube-hetzner Multi-Cluster HermesCloud Runtime

## Purpose

This document defines the target architecture for replacing future `hetzner-k3s` CLI based cluster creation with a `kube-hetzner` Terraform/OpenTofu based runtime cluster factory while keeping the bridge as the runtime operator.

## Final Mental Model

```txt
Terraform/kube-hetzner = cluster factory
Cloudflare provider    = DNS factory
GitHub Actions         = bootstrap pipeline
Bridge                 = runtime operator
Backend                = product control plane
Admin dashboard        = support and visibility layer
```

## Main Rule

Terraform creates clusters. The bridge creates agents.

Do not let Terraform create customer Hermes agent pods.

## High-Level Architecture

```txt
[GitHub Actions]
   | terraform plan/apply
   v
[Hetzner Cloud + kube-hetzner]
   | creates K3s cluster, MicroOS nodes, LB, CSI, node pools
   v
[Cloudflare DNS via Terraform]
   | creates bridge and wildcard runtime records
   v
[Bridge Deployment]
   | deployed into hermes-system namespace
   v
[HermesCloud Backend]
   | registers cluster and routes agent operations
   v
[Bridge per Cluster]
   | uses Kubernetes API + Helm chart to manage agent runtimes
   v
[Hermes Agent Pods]
```

## Control Plane vs Data Plane

### HermesCloud Control Plane

```txt
- users
- workspaces/orgs
- billing/plans
- agents table
- runtime_clusters table
- capacity_reservations table
- runtime_operations table
- routing logic
- support/admin dashboard
```

### Runtime Data Plane

Each runtime cluster has:

```txt
- Kubernetes control plane
- system node pool
- worker node pools
- bridge
- ingress controller
- CSI storage
- customer Hermes agent pods
- PVCs
- Secrets
- ConfigMaps
```

## Communication Flow

### Cluster Provisioning Flow

```txt
Admin triggers GitHub Actions workflow
  -> Terraform plan/apply
  -> kube-hetzner creates cluster
  -> Cloudflare DNS records created
  -> Bridge deployed
  -> Bridge validated
  -> Backend cluster registration endpoint called
  -> Admin dashboard shows cluster
```

### Agent Creation Flow

```txt
User clicks Create Agent
  -> Backend validates plan and quota
  -> Backend checks active runtime clusters
  -> Backend checks bridge health/version
  -> Backend computes effective headroom = bridge headroom - active capacity reservations
  -> Backend creates capacity reservation
  -> Backend creates agent row with cluster_id
  -> Backend calls bridge /v1/instances
  -> Bridge creates namespace, Secret, ConfigMap, PVC, Service, Ingress, Helm release
  -> Backend stores bridge operation_id
  -> Backend polls /v1/operations/:id
  -> Frontend shows Creating -> Live or Failed
  -> Reservation is committed or released
```

### Agent Config Update Flow

```txt
User updates model/provider/Telegram/config
  -> Backend stores desired product state
  -> Backend creates runtime operation with per-agent mutation lock
  -> Backend calls correct bridge based on agent.cluster_id
  -> Bridge patches Secret/ConfigMap or Helm values
  -> Bridge restarts or reloads agent if required
  -> Operation status returned to backend
```

### Terminal Flow

```txt
Frontend terminal UI
  -> Backend authenticates user and checks permission
  -> Backend creates audited terminal session
  -> Backend opens/proxies WebSocket to correct bridge
  -> Bridge uses Kubernetes exec API into agent pod
  -> User gets shell inside their own agent container only
```

Terminal access must never mean node SSH, host shell, bridge shell, or cluster-admin shell.

## Cluster Shape

Recommended first kube-hetzner test shape:

```txt
Cluster: kh-test
Control plane pool:
  1 x cpx22
  no customer workloads

System pool:
  1 x cpx21/cpx11
  bridge, ingress, monitoring later

Runtime worker pool:
  cpx32 autoscaling
  min 1 or 2 for test
  max 3 or 5 for test
  customer Hermes agent pods
```

Recommended first production shape:

```txt
Cluster: eu-1 / eu-2
Control plane pool:
  1 x cpx22 initially
  no customer workloads

System pool:
  1 x cpx21/cpx11 initially
  bridge, ingress, platform components

Runtime worker pool:
  cpx32 autoscaling
  min 2
  max 20
  customer Hermes agent pods
```

Later production upgrade:

```txt
Control plane:
  3 nodes for HA

System pool:
  2 nodes

Runtime workers:
  autoscaling by load
```

## DNS and TLS Model

For cluster `eu-1`:

```txt
Bridge:
  bridge-eu-1.hermeshq.net

Agent wildcard:
  *.runtime-eu-1.hermeshq.net
```

Terraform creates DNS:

```txt
bridge-eu-1.hermeshq.net -> cluster ingress/load balancer IP
*.runtime-eu-1.hermeshq.net -> cluster ingress/load balancer IP
```

Bridge creates per-agent Ingress:

```txt
agent-abc123.runtime-eu-1.hermeshq.net -> agent-abc123 Service -> agent pod
```

V1 TLS rule:

```txt
Backend must only register https:// bridge URLs in production.
Raw http://NODE_IP URLs are forbidden in production.
```

## Bridge Placement

Recommended:

```txt
Bridge runs in hermes-system namespace.
Bridge is scheduled onto system node pool.
Customer agents are scheduled onto runtime worker pool.
```

Avoid:

```txt
- central super-bridge with kubeconfigs for every cluster
- bridge mixed randomly with heavy customer workloads
- customer agents scheduled onto control-plane nodes
```

## Operational Truth

The backend DB is the product source of truth.

Kubernetes is actual runtime state.

The bridge is the runtime executor.

The admin dashboard reconciles these into support-visible truth.
