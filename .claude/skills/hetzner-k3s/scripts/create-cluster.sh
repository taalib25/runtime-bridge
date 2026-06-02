#!/usr/bin/env bash
# Usage: create-cluster.sh <config-file>
# Creates or reconciles a hetzner-k3s cluster. Idempotent.
# Requires: HCLOUD_TOKEN env var, hetzner-k3s CLI installed.
#
# Examples:
#   HCLOUD_TOKEN=xxx create-cluster.sh cluster-config-test.yaml
#   HCLOUD_TOKEN=xxx create-cluster.sh cluster-config.yaml
set -euo pipefail

CONFIG="${1:?Usage: create-cluster.sh <config-file>}"

if [ -z "${HCLOUD_TOKEN:-}" ]; then
  echo "Error: HCLOUD_TOKEN env var not set"
  exit 1
fi

if ! command -v hetzner-k3s &>/dev/null; then
  echo "Error: hetzner-k3s CLI not found. Install:"
  echo "  Linux:  wget https://github.com/vitobotta/hetzner-k3s/releases/latest/download/hetzner-k3s-linux-amd64 && chmod +x hetzner-k3s-linux-amd64 && sudo mv hetzner-k3s-linux-amd64 /usr/local/bin/hetzner-k3s"
  echo "  macOS:  brew install vitobotta/tap/hetzner_k3s"
  exit 1
fi

echo "==> Creating/reconciling cluster from $CONFIG"
hetzner-k3s create --config "$CONFIG"

# Extract kubeconfig path from config
KUBECONFIG_PATH=$(grep 'kubeconfig_path' "$CONFIG" | awk '{print $2}' | tr -d '"' | sed 's|^\./||')
echo ""
echo "==> Cluster ready. To connect:"
echo "    export KUBECONFIG=./$KUBECONFIG_PATH"
echo "    kubectl get nodes"
echo ""
echo "==> Get VM IP:"
echo "    curl -sf -H \"Authorization: Bearer \$HCLOUD_TOKEN\" https://api.hetzner.cloud/v1/servers | python3 -m json.tool | grep -A2 'public_net'"
