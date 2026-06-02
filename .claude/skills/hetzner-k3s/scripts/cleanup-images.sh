#!/usr/bin/env bash
# Usage: cleanup-images.sh <vm-ip> [keep-tag]
# Removes all stale hermes-bridge images from k3s containerd, then prunes layers.
# Keeps the currently-running image and optionally a pinned tag.
#
# Example:
#   cleanup-images.sh 178.104.185.60
#   cleanup-images.sh 178.104.185.60 a9e0029
set -euo pipefail

VM_IP="${1:?Usage: cleanup-images.sh <vm-ip> [keep-tag]}"
KEEP_TAG="${2:-}"
K="KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl"

echo "==> Disk before cleanup:"
ssh root@"$VM_IP" "df -h / | tail -1"

# Determine currently running image tag
RUNNING_TAG=$(ssh root@"$VM_IP" "$K get pods -A -o jsonpath='{.items[*].spec.containers[*].image}'" | \
  tr ' ' '\n' | grep 'hermes-bridge:' | sed 's/.*://' | sort -u | head -1)
echo "--- Currently running: hermes-bridge:$RUNNING_TAG"

# Build exclusion list
EXCLUDE="latest|$RUNNING_TAG"
[ -n "$KEEP_TAG" ] && EXCLUDE="$EXCLUDE|$KEEP_TAG"
echo "--- Keeping tags matching: $EXCLUDE"

# Remove stale tags
REMOVED=$(ssh root@"$VM_IP" "
  k3s ctr images ls | awk '{print \$1}' | grep 'hermes-bridge:' | grep -vE ':($EXCLUDE)' | \
  xargs -r k3s ctr images rm 2>&1 | wc -l
")
echo "--- Removed $REMOVED stale image tags"

# Prune unused layers
echo "--- Pruning unused layers"
ssh root@"$VM_IP" "k3s ctr images prune --all 2>&1 | grep -c 'deleted image' || true" | \
  xargs -I{} echo "--- Pruned {} layer(s)"

# Also clean journal and apt cache
ssh root@"$VM_IP" "
  apt-get clean -y 2>/dev/null || true
  journalctl --vacuum-time=3d 2>/dev/null || true
  find /tmp -type f -mtime +1 -delete 2>/dev/null || true
"

echo ""
echo "==> Disk after cleanup:"
ssh root@"$VM_IP" "df -h / | tail -1"
