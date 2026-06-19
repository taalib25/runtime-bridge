variable "hcloud_token" {
  description = "Hetzner Cloud API token."
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key content. In CI this is derived from SSH_PRIVATE_KEY at runtime."
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key content. Passed as TF_VAR_ssh_private_key in CI."
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for hermeshq.net."
  type        = string
  sensitive   = true
}

variable "acme_email" {
  description = "Email registered with Let's Encrypt for the Traefik 'letsencrypt' certificate resolver."
  type        = string
}
