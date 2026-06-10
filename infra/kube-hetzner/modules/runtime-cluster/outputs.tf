output "cluster_id" {
  description = "Cluster identifier."
  value       = var.cluster_id
}

output "ingress_ip" {
  description = "Public IPv4 of the cluster ingress. For single-node this is the control plane IP (klipper LB)."
  value       = module.kube_hetzner.ingress_public_ipv4
}

output "bridge_url" {
  description = "HTTPS bridge URL used for backend registration. Always https:// — never http://NODE_IP."
  value       = "https://${local.bridge_hostname}"
}

output "runtime_base_domain" {
  description = "Base domain for per-agent runtime Ingress records (e.g. runtime-kh-test.hermeshq.net)."
  value       = local.runtime_base_domain
}

output "kubeconfig" {
  description = "Cluster kubeconfig. Sensitive — never log or print."
  value       = module.kube_hetzner.kubeconfig_file
  sensitive   = true
}
