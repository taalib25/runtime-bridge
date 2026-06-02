#!/usr/bin/env bash
# Usage: upgrade-cluster.sh <config-file> [k3s-version]
# Upgrades k3s on all nodes, then re-pins the version for future nodes.
# If no version given, lists available versions and exits.
#
# Example:
#   HCLOUD_TOKEN=xxx upgrade-cluster.sh cluster-config-test.yaml v1.33.0+k3s1
set -euo pipefail

CONFIG="${1:?Usage: upgrade-cluster.sh <config-file> [k3s-version]}"
NEW_VERSION="${2:-}"

if [ -z "${HCLOUD_TOKEN:-}" ]; then
  echo "Error: HCLOUD_TOKEN env var not set"
  exit 1
fi

if [ -z "$NEW_VERSION" ]; then
  echo "Available k3s versions:"
  hetzner-k3s releases | head -20
  echo ""
  echo "Run: upgrade-cluster.sh $CONFIG <version>"
  exit 0
fi

KUBECONFIG_PATH=$(grep 'kubeconfig_path' "$CONFIG" | awk '{print $2}' | tr -d '"')
CURRENT_VERSION=$(KUBECONFIG="$KUBECONFIG_PATH" kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.kubeletVersion}' 2>/dev/null || echo "unknown")

echo "==> Upgrading cluster: $CURRENT_VERSION → $NEW_VERSION"
echo "    Config: $CONFIG"
echo ""
read -rp "Continue? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || exit 0

echo "--- Running upgrade"
hetzner-k3s upgrade --config "$CONFIG" --new-k3s-version "$NEW_VERSION"

echo ""
echo "--- Monitoring nodes (ctrl+c when all show new version)"
KUBECONFIG="$KUBECONFIG_PATH" watch kubectl get nodes -o wide

echo ""
echo "--- Pinning new version for future nodes (re-running create)"
hetzner-k3s create --config "$CONFIG"

echo ""
echo "==> Upgrade complete. Verify:"
KUBECONFIG="$KUBECONFIG_PATH" kubectl get nodes -o custom-columns=NAME:.metadata.name,VERSION:.status.nodeInfo.kubeletVersion
