locals {
  bridge_hostname     = "bridge-${var.cluster_id}.${var.root_domain}"
  runtime_base_domain = "runtime-${var.cluster_id}.${var.root_domain}"
}

module "kube_hetzner" {
  source  = "kube-hetzner/kube-hetzner/hcloud"
  version = "2.20.0"

  providers = {
    hcloud = hcloud
  }

  hcloud_token    = var.hcloud_token
  ssh_public_key  = var.ssh_public_key
  ssh_private_key = var.ssh_private_key

  cluster_name   = var.cluster_id
  network_region = var.network_region

  control_plane_nodepools = [
    {
      name        = "control-plane-${var.location}"
      server_type = var.server_type
      location    = var.location
      count       = 1
      labels      = ["platform.hermes/node-role=system"]
      taints      = []
    }
  ]

  # No agent pools for single-node kh-test.
  # For eu-1/eu-2, override with runtime worker pools.
  agent_nodepools = []

  # Single-node: module auto-detects is_single_node_cluster = true,
  # enables klipper LB, and allows scheduling on control plane automatically.
  ingress_controller = "traefik"

  # Without this, every IngressRoute's `tls.certResolver: letsencrypt` (set by
  # charts/hermes-agent/values.yaml) references a resolver Traefik never defined —
  # Traefik logs "Router uses a nonexistent certificate resolver" and falls back to
  # its self-signed default cert on every hostname. TLS-ALPN-01 (not HTTP-01) so it
  # only needs port 443, already open — no port-80/web-entrypoint config required.
  # Resolver name "letsencrypt" must exactly match the chart's certResolver value.
  traefik_additional_options = [
    "--certificatesresolvers.letsencrypt.acme.tlschallenge=true",
    "--certificatesresolvers.letsencrypt.acme.email=${var.acme_email}",
    "--certificatesresolvers.letsencrypt.acme.storage=/data/acme.json",
  ]

  initial_k3s_channel       = var.initial_k3s_channel
  automatically_upgrade_k3s = var.automatically_upgrade_k3s
  automatically_upgrade_os  = var.automatically_upgrade_os
}

resource "cloudflare_dns_record" "bridge" {
  zone_id = var.cloudflare_zone_id
  name    = "bridge-${var.cluster_id}"
  type    = "A"
  content = module.kube_hetzner.ingress_public_ipv4
  ttl     = 1
  proxied = var.cloudflare_proxied
  comment = "HermesCloud bridge — ${var.cluster_id}"
}

resource "cloudflare_dns_record" "runtime_wildcard" {
  zone_id = var.cloudflare_zone_id
  name    = "*.runtime-${var.cluster_id}"
  type    = "A"
  content = module.kube_hetzner.ingress_public_ipv4
  ttl     = 1
  proxied = var.cloudflare_runtime_proxied
  comment = "HermesCloud runtime wildcard — ${var.cluster_id}"
}
