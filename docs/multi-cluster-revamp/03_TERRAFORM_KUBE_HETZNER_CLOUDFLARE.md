# Terraform, kube-hetzner, Cloudflare DNS, and TLS

## Purpose

This document defines how Terraform/OpenTofu, kube-hetzner, Hetzner Cloud, and Cloudflare work together to create runtime clusters.

## Responsibility Boundary

```txt
Terraform/kube-hetzner:
  creates cluster infrastructure
  creates node pools
  creates networking/LB/ingress base
  creates Cloudflare cluster DNS

GitHub Actions:
  runs Terraform
  deploys bridge
  validates cluster
  registers backend

Bridge:
  creates customer Hermes agent runtime resources
```

Terraform must not create customer agent pods.

## Suggested Repo Structure

```txt
infra/
  kube-hetzner/
    modules/
      runtime-cluster/
        main.tf
        variables.tf
        outputs.tf
        providers.tf
        versions.tf
    clusters/
      kh-test/
        main.tf
        terraform.tfvars.example
      eu-1/
        main.tf
        terraform.tfvars.example
      eu-2/
        main.tf
        terraform.tfvars.example
```

## kube-hetzner Module Version

Pin to an exact version. Do not use `main`, `latest`, or `~> 2.0`.

```hcl
module "kube_hetzner" {
  source  = "kube-hetzner/kube-hetzner/hcloud"
  version = "2.20.0"
  # ...
}
```

Verified present on registry.terraform.io as of 2026-06-02.
Upgrade only intentionally after reading the kube-hetzner changelog.

## Infrastructure CLI

Use **OpenTofu** — not Terraform CLI / hashicorp/setup-terraform.

```txt
tofu init
tofu fmt
tofu validate
tofu plan
tofu apply
```

In GitHub Actions use `opentofu/setup-opentofu@v1` with a pinned OpenTofu version.

## Providers

```hcl
terraform {
  required_version = ">= 1.8"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.50"
    }

    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

provider "cloudflare" {
  # Read CLOUDFLARE_API_TOKEN from environment.
}
```

## Required Variables

```hcl
variable "cluster_id" {
  type = string
}

variable "root_domain" {
  type    = string
  default = "hermeshq.net"
}

variable "hcloud_token" {
  type      = string
  sensitive = true
}

variable "cloudflare_zone_id" {
  type      = string
  sensitive = true
}

variable "cloudflare_proxied" {
  type    = bool
  default = true
}

variable "region" {
  type    = string
  default = "fsn1"
}
```

## Cluster Shape

### kh-test

Single-node proof cluster. Do not use a 3-pool shape for kh-test.

```txt
1 x cpx31 (4 vCPU, 8 GB RAM)
schedule_workloads_on_masters = true
no system pool
no runtime worker pool
```

Rationale: proves the full Terraform → DNS → bridge → backend pipeline cheaply (~€10/month).
Delete and recreate freely during proof phase.
Upgrade to 3-pool shape only when graduating to eu-1/eu-2.

### eu-1/eu-2 V1

```txt
control-plane:
  cpx22 x 1

system pool:
  cpx11 or cpx21 x 1

runtime workers:
  cpx32 autoscaling min 2, max 20
```

### Later HA Upgrade

```txt
control-plane:
  3 nodes

system pool:
  2 nodes

runtime workers:
  autoscaling by demand
```

## Node Labels

The cluster factory must label nodes so bridge and agents schedule predictably.

```txt
system pool:
  platform.hermes/node-role=system

runtime workers:
  platform.hermes/node-role=runtime
  platform.hermes/tier=standard
```

Future pools:

```txt
platform.hermes/tier=free
platform.hermes/tier=pro
platform.hermes/tier=power
```

## Cloudflare DNS Records

For `cluster_id = eu-1`:

```txt
bridge-eu-1.hermeshq.net
*.runtime-eu-1.hermeshq.net
```

Terraform resources:

```hcl
locals {
  bridge_hostname     = "bridge-${var.cluster_id}.${var.root_domain}"
  runtime_base_domain = "runtime-${var.cluster_id}.${var.root_domain}"
  ingress_ip          = module.kube_hetzner.ingress_ip # confirm actual output name
}

resource "cloudflare_dns_record" "bridge" {
  zone_id = var.cloudflare_zone_id
  name    = "bridge-${var.cluster_id}"
  type    = "A"
  content = local.ingress_ip
  ttl     = 1
  proxied = var.cloudflare_proxied
  comment = "HermesCloud bridge for ${var.cluster_id}"
}

resource "cloudflare_dns_record" "runtime_wildcard" {
  zone_id = var.cloudflare_zone_id
  name    = "*.runtime-${var.cluster_id}"
  type    = "A"
  content = local.ingress_ip
  ttl     = 1
  proxied = var.cloudflare_proxied
  comment = "HermesCloud runtime wildcard for ${var.cluster_id}"
}
```

The coding agent must confirm the exact kube-hetzner output names with `terraform output` and the module docs.

## Outputs

```hcl
output "cluster_id" {
  value = var.cluster_id
}

output "bridge_url" {
  value = "https://${local.bridge_hostname}"
}

output "runtime_base_domain" {
  value = local.runtime_base_domain
}

output "ingress_ip" {
  value = local.ingress_ip
}

output "kubeconfig" {
  value     = module.kube_hetzner.kubeconfig
  sensitive = true
}
```

## TLS Strategy

### V1 Minimum

```txt
Cloudflare DNS proxied = true
Backend registers https://bridge-<cluster>.hermeshq.net
Cloudflare serves HTTPS to backend/admin/frontend
Raw IP bridge access blocked by firewall or not routable
```

Validation must confirm:

```txt
curl https://bridge-<cluster>.hermeshq.net/healthz works
curl http://<raw-ip>:8080 does not expose production bridge publicly
backend rejects http:// bridge_url in production
```

### V1.5 Stronger Origin TLS

Add one of:

```txt
- cert-manager with DNS01/HTTP01
- Cloudflare Origin Certificates
- Traefik TLS config managed by platform bootstrap
```

Do not leave certificate behavior implicit.

## Remote State

**Decision: OpenTofu + Cloudflare R2 (S3-compatible, free tier), one state key per cluster.**

Do not use HCP Terraform / Terraform Cloud.
Do not use Hetzner Object Storage.
Do not use a single shared state file for all clusters.

Backend block per cluster:

```hcl
terraform {
  backend "s3" {
    bucket   = "hermes-tofu-state"
    key      = "clusters/kh-test/tofu.tfstate"   # one key per cluster
    region   = "auto"
    endpoint = "https://<ACCOUNT_ID>.r2.cloudflarestorage.com"

    use_lockfile = true   # OpenTofu native S3 locking, no DynamoDB needed

    skip_credentials_validation = true
    skip_region_validation      = true
    skip_metadata_api_check     = true
  }
}
```

State key pattern:

```txt
clusters/kh-test/tofu.tfstate
clusters/eu-1/tofu.tfstate
clusters/eu-2/tofu.tfstate
```

Required GitHub secrets for state access (per environment):

```txt
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_ENDPOINT        (https://<ACCOUNT_ID>.r2.cloudflarestorage.com)
```

These are passed as environment variables `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` and `-backend-config=endpoint=...` to `tofu init`.

Required: validate that `use_lockfile = true` prevents concurrent applies before any production cluster provisioning.

## What Terraform Must Not Do

```txt
- must not create per-user agent pods
- must not run live customer runtime operations
- must not perform terminal/log operations
- must not replace bridge operation tracking
- must not run nightly cleanup as an apply step
```

## What Terraform May Bootstrap

```txt
- namespace for platform components
- Cloudflare DNS
- base ingress/LB config
- optional GitOps controller later
- optional bridge chart if team intentionally chooses Terraform Helm provider
```

Recommended V1:

```txt
Terraform creates infra and DNS.
GitHub Actions deploys bridge after Terraform output is available.
```
