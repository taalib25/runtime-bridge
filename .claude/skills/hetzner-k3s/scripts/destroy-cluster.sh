#!/usr/bin/env bash
# destroy-cluster.sh — safely destroy a cluster and clean up
#
# Guardrails:
#   - Checks protect_against_deletion is false in config
#   - Requires typing the cluster name to confirm
#   - Offers to delete the GitHub Environment and drain bridge registration
#
# Resources NOT auto-deleted (clean manually in Hetzner Console):
#   Load Balancers, Persistent Volumes, Floating IPs, Snapshots
#
# Usage:
#   HCLOUD_TOKEN=xxx ./destroy-cluster.sh <config-file>
set -euo pipefail

CONFIG="${1:?Usage: destroy-cluster.sh <config-file>}"

[ -n "${HCLOUD_TOKEN:-}" ] || { echo "Error: HCLOUD_TOKEN not set"; exit 1; }
[ -f "$CONFIG" ]           || { echo "Error: config not found: $CONFIG"; exit 1; }

CLUSTER_NAME=$(grep 'cluster_name:' "$CONFIG" | awk '{print $2}' | tr -d '"')
PROTECTION=$(grep 'protect_against_deletion:' "$CONFIG" | awk '{print $2}')

echo "!!! DESTROY CLUSTER: $CLUSTER_NAME"
echo ""

if [ "${PROTECTION:-true}" = "true" ]; then
  echo "ERROR: protect_against_deletion is 'true' in $CONFIG"
  echo ""
  echo "  Edit $CONFIG and set:"
  echo "    protect_against_deletion: false"
  echo ""
  echo "  Then re-run this script."
  exit 1
fi

echo "This will permanently destroy all VMs, networks, and firewalls."
echo "NOT deleted automatically: load balancers, PVCs, floating IPs, snapshots."
echo ""
read -rp "Type cluster name to confirm DESTRUCTION ($CLUSTER_NAME): " confirm
[ "$confirm" = "$CLUSTER_NAME" ] || { echo "Aborted."; exit 0; }

# Optional: drain bridge from backend before destroying
echo ""
read -rp "Drain bridge from backend first (recommended)? [Y/n] " drain
if [[ ! "$drain" =~ ^[Nn]$ ]]; then
  read -rp "ADMIN_API_SECRET: " ADMIN_API_SECRET_INPUT
  BACKEND_URL="${BACKEND_URL:-https://api.hermeshq.net}"
  curl -sf -X PUT "$BACKEND_URL/api/admin/clusters" \
    -H "Authorization: Bearer $ADMIN_API_SECRET_INPUT" \
    -H "Content-Type: application/json" \
    -d "{\"cluster_id\":\"$CLUSTER_NAME\",\"status\":\"offline\"}" \
    && echo "Cluster marked offline in backend" || echo "Warning: could not update backend"
fi

echo ""
echo "==> Destroying cluster..."
hetzner-k3s delete --config "$CONFIG" --force

# Optional: remove GitHub Environment
echo ""
REPO="taalib25/hermes-runtime-operator"
read -rp "Delete GitHub Environment '$CLUSTER_NAME' from $REPO? [y/N] " del_env
if [[ "$del_env" =~ ^[Yy]$ ]]; then
  gh api --method DELETE "repos/$REPO/environments/$CLUSTER_NAME" 2>/dev/null \
    && echo "GitHub Environment '$CLUSTER_NAME' deleted" \
    || echo "Warning: could not delete GitHub Environment (may not exist)"
fi

echo ""
echo "==> Cluster $CLUSTER_NAME destroyed."
echo ""
echo "Manual cleanup needed in Hetzner Console (Project → $CLUSTER_NAME):"
echo "  - Load Balancers"
echo "  - Volumes (PVCs)"
echo "  - Floating IPs (if any)"
echo "  - Snapshots (if any)"
