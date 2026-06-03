#!/usr/bin/env bash
# upgrade-cluster.sh — two-step k3s upgrade with monitoring
#
# hetzner-k3s upgrade alone is only step 1. This script enforces step 2
# (re-run create to pin the version for future nodes) and monitors
# System Upgrade Controller jobs between the two steps.
#
# Usage:
#   HCLOUD_TOKEN=xxx ./upgrade-cluster.sh <config-file> [k3s-version]
#
# If no version given, lists available releases.
set -euo pipefail

CONFIG="${1:?Usage: upgrade-cluster.sh <config-file> [k3s-version]}"
NEW_VERSION="${2:-}"

[ -n "${HCLOUD_TOKEN:-}" ] || { echo "Error: HCLOUD_TOKEN not set"; exit 1; }
[ -f "$CONFIG" ]           || { echo "Error: config file not found: $CONFIG"; exit 1; }

CLUSTER_NAME=$(grep 'cluster_name:' "$CONFIG"   | awk '{print $2}' | tr -d '"')
KUBECONFIG_PATH=$(grep 'kubeconfig_path:' "$CONFIG" | awk '{print $2}' | tr -d '"')

if [ -z "$NEW_VERSION" ]; then
  echo "Available k3s releases:"
  hetzner-k3s releases | head -20
  echo ""
  echo "Usage: $0 $CONFIG <version>"
  exit 0
fi

CURRENT=$(KUBECONFIG="$KUBECONFIG_PATH" kubectl get nodes \
  -o jsonpath='{.items[0].status.nodeInfo.kubeletVersion}' 2>/dev/null || echo "unknown")

echo "Cluster  : $CLUSTER_NAME"
echo "Current  : $CURRENT"
echo "Upgrading: $NEW_VERSION"
echo ""
echo "WARNING: Single-master clusters will have a brief API outage during control-plane upgrade."
echo "         After all nodes upgrade you MUST re-run create to pin the version for future nodes."
echo ""
read -rp "Type cluster name to confirm upgrade ($CLUSTER_NAME): " confirm
[ "$confirm" = "$CLUSTER_NAME" ] || { echo "Aborted."; exit 0; }

# Step 1: trigger rolling upgrade via System Upgrade Controller
echo ""
echo "==> Step 1/2: Triggering upgrade..."
hetzner-k3s upgrade --config "$CONFIG" --new-k3s-version "$NEW_VERSION" --quiet

echo ""
echo "==> Monitoring — wait for all nodes to show the new version."
echo "    Press Ctrl+C when done, then confirm to proceed to step 2."
KUBECONFIG="$KUBECONFIG_PATH" watch -n 5 \
  "kubectl get nodes -o wide; echo '---'; kubectl get jobs -n system-upgrade 2>/dev/null" \
  || true   # Ctrl+C is the expected exit

echo ""
echo "==> Current node versions:"
KUBECONFIG="$KUBECONFIG_PATH" kubectl get nodes \
  -o custom-columns=NAME:.metadata.name,VERSION:.status.nodeInfo.kubeletVersion,STATUS:.status.conditions[-1].type

echo ""
read -rp "All nodes upgraded? Run create to pin version? [y/N] " pin
[[ "$pin" =~ ^[Yy]$ ]] || { echo "Run manually: hetzner-k3s create --config $CONFIG"; exit 0; }

# Step 2: pin version for future nodes — MANDATORY
echo ""
echo "==> Step 2/2: Pinning $NEW_VERSION for future nodes..."
hetzner-k3s create --config "$CONFIG" --quiet

echo ""
echo "==> Upgrade complete. New nodes will be provisioned on $NEW_VERSION."
echo ""
echo "    Stall recovery (if any node didn't upgrade):"
echo "      KUBECONFIG=$KUBECONFIG_PATH kubectl -n system-upgrade delete job --all"
echo "      KUBECONFIG=$KUBECONFIG_PATH kubectl -n system-upgrade delete plan --all"
echo "      KUBECONFIG=$KUBECONFIG_PATH kubectl label node --all plan.upgrade.cattle.io/k3s-server- plan.upgrade.cattle.io/k3s-agent-"
echo "      KUBECONFIG=$KUBECONFIG_PATH kubectl -n system-upgrade rollout restart deployment system-upgrade-controller"
