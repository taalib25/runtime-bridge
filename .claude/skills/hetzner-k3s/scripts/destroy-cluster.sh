#!/usr/bin/env bash
# Usage: destroy-cluster.sh <config-file>
# Safely destroys a cluster. Checks protect_against_deletion first.
# WARNING: Does NOT delete load balancers, PVCs, floating IPs — clean those manually.
#
# Example:
#   HCLOUD_TOKEN=xxx destroy-cluster.sh cluster-config-test.yaml
set -euo pipefail

CONFIG="${1:?Usage: destroy-cluster.sh <config-file>}"

if [ -z "${HCLOUD_TOKEN:-}" ]; then
  echo "Error: HCLOUD_TOKEN env var not set"
  exit 1
fi

CLUSTER_NAME=$(grep 'cluster_name:' "$CONFIG" | awk '{print $2}')
PROTECTION=$(grep 'protect_against_deletion:' "$CONFIG" | awk '{print $2}')

echo "!!! DESTROY CLUSTER: $CLUSTER_NAME"
echo ""

if [ "$PROTECTION" = "true" ]; then
  echo "ERROR: protect_against_deletion is true in $CONFIG"
  echo "Edit the config and set:  protect_against_deletion: false"
  exit 1
fi

echo "WARNING: This will permanently destroy all VMs, networks, and firewalls."
echo "It will NOT delete: load balancers, persistent volumes, floating IPs, snapshots."
echo "Clean those manually via Hetzner Console afterward."
echo ""
read -rp "Type cluster name to confirm ($CLUSTER_NAME): " confirm
[ "$confirm" = "$CLUSTER_NAME" ] || { echo "Aborted."; exit 0; }

hetzner-k3s delete --config "$CONFIG"

echo ""
echo "==> Cluster $CLUSTER_NAME destroyed."
echo "    Manual cleanup needed in Hetzner Console:"
echo "    - Load Balancers"
echo "    - Volumes (PVCs)"
echo "    - Floating IPs (if any)"
echo "    - Snapshots (if any)"
