#!/usr/bin/env bash
# scale-pool.sh — add or remove nodes from a worker pool
#
# Scaling up:   edit instance_count in config, re-run hetzner-k3s create (idempotent)
# Scaling down: drain+delete the extra nodes first, then edit config, then create
#
# Usage:
#   HCLOUD_TOKEN=xxx ./scale-pool.sh <config-file> <pool-name> <new-count>
set -euo pipefail

CONFIG="${1:?Usage: scale-pool.sh <config-file> <pool-name> <new-count>}"
POOL="${2:?pool-name required}"
NEW_COUNT="${3:?new-count required}"
KUBECONFIG_PATH=$(grep 'kubeconfig_path:' "$CONFIG" | awk '{print $2}' | tr -d '"')

[ -n "${HCLOUD_TOKEN:-}" ] || { echo "Error: HCLOUD_TOKEN not set"; exit 1; }
[ -f "$CONFIG" ]           || { echo "Error: config not found: $CONFIG"; exit 1; }
[[ "$NEW_COUNT" =~ ^[0-9]+$ ]] || { echo "Error: new-count must be a number"; exit 1; }

CLUSTER_NAME=$(grep 'cluster_name:' "$CONFIG" | awk '{print $2}' | tr -d '"')
CURRENT_COUNT=$(grep -A5 "name: $POOL" "$CONFIG" | grep 'instance_count:' | awk '{print $2}')

echo "Cluster    : $CLUSTER_NAME"
echo "Pool       : $POOL"
echo "Current    : ${CURRENT_COUNT:-unknown}"
echo "Target     : $NEW_COUNT"

if [ "${CURRENT_COUNT:-0}" -gt "$NEW_COUNT" ]; then
  REMOVING=$(( CURRENT_COUNT - NEW_COUNT ))
  echo ""
  echo "WARNING: Scaling DOWN by $REMOVING node(s)."
  echo "         You must drain and delete the extra nodes from Kubernetes first,"
  echo "         then this script will reconcile the config."
  echo ""
  echo "  Nodes to remove (last $REMOVING alphabetically):"
  KUBECONFIG="$KUBECONFIG_PATH" kubectl get nodes --sort-by=.metadata.name \
    -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[-1].type 2>/dev/null || true
  echo ""
  echo "  For each node to remove:"
  echo "    KUBECONFIG=$KUBECONFIG_PATH kubectl drain <node> --ignore-daemonsets --delete-emptydir-data"
  echo "    KUBECONFIG=$KUBECONFIG_PATH kubectl delete node <node>"
  echo "    Then delete the VM in Hetzner Console"
  echo ""
  read -rp "Have you drained and deleted the extra nodes? [y/N] " done_drain
  [[ "$done_drain" =~ ^[Yy]$ ]] || { echo "Aborted. Drain the nodes first."; exit 0; }
fi

# Edit the config in-place
sed -i "/name: $POOL/,/instance_count:/ s/instance_count: .*/instance_count: $NEW_COUNT/" "$CONFIG"
echo ""
echo "==> Updated $CONFIG: pool '$POOL' instance_count → $NEW_COUNT"
echo "==> Reconciling cluster..."
hetzner-k3s create --config "$CONFIG" --quiet
echo ""
echo "==> Pool '$POOL' scaled to $NEW_COUNT nodes."
