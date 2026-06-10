variable "cluster_id" {
  description = "Short cluster identifier used in DNS and resource names (e.g. kh-test, eu-1, eu-2)."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9\\-]+$", var.cluster_id))
    error_message = "cluster_id must be lowercase alphanumeric and dashes only."
  }
}

variable "hcloud_token" {
  description = "Hetzner Cloud API token."
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key content for node access."
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key content used by kube-hetzner during node provisioning."
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for hermeshq.net."
  type        = string
  sensitive   = true
}

variable "cloudflare_proxied" {
  description = "Whether to proxy DNS records through Cloudflare (orange cloud). Required for edge TLS."
  type        = bool
  default     = true
}

variable "root_domain" {
  description = "Root domain for bridge and runtime agent DNS records."
  type        = string
  default     = "hermeshq.net"
}

variable "network_region" {
  description = "Hetzner network region. eu-central covers fsn1, nbg1, hel1."
  type        = string
  default     = "eu-central"
}

variable "location" {
  description = "Hetzner datacenter location for control plane nodes."
  type        = string
  default     = "fsn1"
}

variable "server_type" {
  description = "Hetzner server type for control plane nodes."
  type        = string
  default     = "cpx31"
}

variable "initial_k3s_channel" {
  description = "k3s release channel. Must be stable, latest, or a minor version like v1.32."
  type        = string
  default     = "v1.32"
}

variable "automatically_upgrade_k3s" {
  description = "Whether to automatically upgrade k3s. Disabled by default for controlled upgrades."
  type        = bool
  default     = false
}

variable "automatically_upgrade_os" {
  description = "Whether to automatically upgrade OS. Disabled for single-node clusters (no node to evict to)."
  type        = bool
  default     = false
}
