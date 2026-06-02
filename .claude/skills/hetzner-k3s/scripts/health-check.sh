#!/usr/bin/env bash
# Usage: health-check.sh <vm-ip> [kubeconfig-path]
# Checks cluster health: nodes, pods, disk, bridge API
set -euo pipefail

VM_IP="${1:?Usage: health-check.sh <vm-ip> [kubeconfig-path]}"
KUBECONFIG_PATH="${2:-/etc/rancher/k3s/k3s.yaml}"
K="KUBECONFIG=$KUBECONFIG_PATH kubectl"

echo "=== Nodes ==="
ssh root@"$VM_IP" "$K get nodes -o wide"

echo ""
echo "=== Pods (non-Running) ==="
ssh root@"$VM_IP" "$K get pods -A --no-headers" | grep -v -E 'Running|Completed' || echo "All pods healthy"

echo ""
echo "=== Disk ==="
ssh root@"$VM_IP" "df -h / | tail -1"

echo ""
echo "=== Images (count) ==="
ssh root@"$VM_IP" "k3s ctr images ls | grep -v REF | wc -l"

echo ""
echo "=== Bridge health ==="
BRIDGE_NS="hermes-bridge"
SECRET=$(ssh root@"$VM_IP" "$K -n $BRIDGE_NS get secret bridge-auth -o jsonpath='{.data.secret}' 2>/dev/null | base64 -d" || echo "")
if [ -z "$SECRET" ]; then
  echo "bridge-auth secret not found — bridge may not be deployed"
else
  curl -sf --max-time 5 -H "X-Bridge-Secret: $SECRET" -H "Host: bridge.hermeshq.net" \
    "http://$VM_IP/healthz" | python3 -m json.tool 2>/dev/null || \
    echo "Bridge not reachable at http://$VM_IP/healthz"
fi

echo ""
echo "=== Instances ==="
if [ -n "$SECRET" ]; then
  curl -sf --max-time 5 -H "X-Bridge-Secret: $SECRET" "https://bridge.hermeshq.net/v1/instances" 2>/dev/null | \
    python3 -c "
import json,sys
try:
  items = json.load(sys.stdin)['items']
  for i in items:
    print(f\"  {i['instanceId']}  {i['phase']}  healthy={i['healthy']}  {i['spec']['image']}:{i['spec']['imageTag']}\")
  print(f'Total: {len(items)}')
except: print('Could not parse response')
" || echo "Could not reach bridge API"
fi
