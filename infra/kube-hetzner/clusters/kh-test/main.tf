terraform {
  required_version = ">= 1.10.1"

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
  # Reads CLOUDFLARE_API_TOKEN from environment — do not hardcode.
}

module "kh_test" {
  source = "../../modules/runtime-cluster"

  providers = {
    hcloud     = hcloud
    cloudflare = cloudflare
  }

  cluster_id         = "kh-test"
  hcloud_token       = var.hcloud_token
  ssh_public_key     = var.ssh_public_key
  ssh_private_key    = var.ssh_private_key
  cloudflare_zone_id = var.cloudflare_zone_id

  # Single cpx22 in fsn1 — kube-hetzner auto-enables klipper LB + scheduling on CP.
  server_type = "cpx22"
  location    = "fsn1"

  initial_k3s_channel       = "v1.32"
  automatically_upgrade_k3s = false
  automatically_upgrade_os  = false
}

output "cluster_id" {
  value = module.kh_test.cluster_id
}

output "ingress_ip" {
  value = module.kh_test.ingress_ip
}

output "bridge_url" {
  value = module.kh_test.bridge_url
}

output "runtime_base_domain" {
  value = module.kh_test.runtime_base_domain
}

output "kubeconfig" {
  value     = module.kh_test.kubeconfig
  sensitive = true
}
