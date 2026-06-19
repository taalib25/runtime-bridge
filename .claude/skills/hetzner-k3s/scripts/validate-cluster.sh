#!/usr/bin/env bash
# validate-cluster.sh — fail-fast cluster validator; called by provision-cluster.sh
#                       and runnable standalone for ongoing health verification.
#
# Usage:
#   validate-cluster.sh <cluster-name> <node-ip>
#
# Env (all optional):
#   SSH_PRIVATE_KEY   key CONTENTS (not path) — falls back to default ssh agent
#   BRIDGE_SECRET     bridge auth secret — fetched from k8s secret on node if unset
#   BRIDGE_DOMAIN     FQDN for HTTPS check; skip if unset
#   SCRIPT_DIR        path to sibling scripts — auto-detected from $0
set -uo pipefail

CLUSTER_NAME="${1:?Usage: validate-cluster.sh <cluster-name> <node-ip>}"
NODE_IP="${2:?Usage: validate-cluster.sh <cluster-name> <node-ip>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

RED='\033[0;31m'; GRN='\033[0;32m'; YLW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "  ${GRN}[ok]${NC}   $*"; }
fail() { echo -e "  ${RED}[FAIL]${NC} $*" >&2; }

if [ -n "${SSH_PRIVATE_KEY:-}" ]; then
  remote() { ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
                 -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" "$@"; }
else
  remote() { ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
                 root@"$NODE_IP" "$@"; }
fi

FAILED=0

echo ""
echo -e "${YLW}=== validate-cluster: $CLUSTER_NAME @ $NODE_IP ===${NC}"

# ── 1. SSH ──────────────────────────────────────────────────────────────────────
if remote exit 2>/dev/null; then
  ok "SSH reachable"
else
  fail "SSH not reachable on $NODE_IP"
  exit 1
fi

# ── 2. k3s / kubectl ────────────────────────────────────────────────────────────
READY_NODES=$(remote "KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl get nodes --no-headers 2>/dev/null \
  | grep -c ' Ready '" 2>/dev/null || echo 0)
if [ "${READY_NODES:-0}" -gt 0 ]; then
  ok "k3s — $READY_NODES Ready node(s)"
else
  fail "k3s — no Ready nodes (check systemctl status k3s on $NODE_IP)"
  FAILED=1
fi

# ── 3. Bridge namespace + pod ───────────────────────────────────────────────────
K="KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl"

NS_EXISTS=$(remote "$K get namespace hermes-bridge --no-headers 2>/dev/null | wc -l" 2>/dev/null || echo 0)
if [ "${NS_EXISTS:-0}" -gt 0 ]; then
  ok "namespace hermes-bridge exists"
else
  fail "namespace hermes-bridge missing"
  FAILED=1
fi

POD_RUNNING=$(remote "$K -n hermes-bridge get pods -l app.kubernetes.io/name=hermes-bridge \
  --no-headers 2>/dev/null | grep -c 'Running'" 2>/dev/null || echo 0)
if [ "${POD_RUNNING:-0}" -gt 0 ]; then
  ok "bridge pod Running"
else
  fail "bridge pod not Running (check kubectl -n hermes-bridge get pods on $NODE_IP)"
  FAILED=1
fi

# abort early — remaining checks require the pod to be up
[ "$FAILED" -eq 0 ] || { echo ""; echo -e "${RED}=== validate-cluster FAILED on $CLUSTER_NAME ===${NC}" >&2; exit 1; }

# ── 4. Fetch bridge secret if not provided ──────────────────────────────────────
if [ -z "${BRIDGE_SECRET:-}" ]; then
  BRIDGE_SECRET=$(remote "$K -n hermes-bridge get secret bridge-auth \
    -o jsonpath='{.data.secret}' 2>/dev/null | base64 -d" 2>/dev/null || echo "")
fi
[ -n "${BRIDGE_SECRET:-}" ] || { fail "Could not obtain BRIDGE_SECRET (k8s secret missing?)"; exit 1; }

# ── 5. /healthz ─────────────────────────────────────────────────────────────────
HEALTHZ=$(remote "curl -sf --max-time 10 \
  -H 'X-Bridge-Secret: $BRIDGE_SECRET' http://localhost:8080/healthz 2>/dev/null" || echo "")
if echo "$HEALTHZ" | grep -q '"status":"ok"'; then
  ok "/healthz → ok"
else
  fail "/healthz → unexpected: $HEALTHZ"
  FAILED=1
fi

# ── 6. /readyz — must be "ready", not "degraded" ────────────────────────────────
READYZ=$(remote "curl -sf --max-time 10 \
  -H 'X-Bridge-Secret: $BRIDGE_SECRET' http://localhost:8080/readyz 2>/dev/null" || echo "")
if echo "$READYZ" | grep -q '"status":"ready"'; then
  ok "/readyz → ready"
elif echo "$READYZ" | grep -q '"status":"degraded"'; then
  MISSING=$(echo "$READYZ" | \
    python3 -c "import json,sys; d=json.load(sys.stdin); print(' '.join(d.get('missingPermissions',[])))" \
    2>/dev/null || echo "(parse error)")
  fail "/readyz → degraded — missingPermissions: $MISSING"
  FAILED=1
else
  fail "/readyz → unexpected response: $READYZ"
  FAILED=1
fi

# ── 7. /v1/cluster/summary ──────────────────────────────────────────────────────
SUMMARY=$(remote "curl -sf --max-time 10 \
  -H 'X-Bridge-Secret: $BRIDGE_SECRET' http://localhost:8080/v1/cluster/summary 2>/dev/null" || echo "")
if echo "$SUMMARY" | grep -q '"kubernetesReachable":true'; then
  ok "/v1/cluster/summary → kubernetesReachable=true"
else
  fail "/v1/cluster/summary → unexpected: $SUMMARY"
  FAILED=1
fi

# ── 7b. Warm-node status (informational — never fails the deploy) ───────────────
# A fresh deploy's pre-puller hasn't had time to actually pull the image yet, so this
# is reported, not enforced here. The backend is what actually gates "don't place the
# first user on a cold node" — see cluster-scheduler.service.ts's runtimeImageWarmed
# preference. This check exists so a human watching the deploy can see warm-up progress
# instead of having to go dig through kubectl.
WARM=$(echo "$SUMMARY" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    print(f\"warmed={d.get('runtimeImageWarmed')} prepuller={d.get('prepullerReady')}/{d.get('prepullerDesired')} unavailable={d.get('prepullerUnavailable')} image={d.get('runtimeImageReference','?')}\")
except Exception as e:
    print(f'parse error: {e}')
" 2>/dev/null || echo "parse error")
if echo "$WARM" | grep -q "warmed=True"; then
  ok "warm-node: $WARM"
else
  echo -e "  ${YLW}[warn]${NC} warm-node: $WARM (not warmed yet — image still pulling, or pre-puller not deployed)"
fi

# ── 8. HTTPS via Cloudflare (if BRIDGE_DOMAIN set) ──────────────────────────────
if [ -n "${BRIDGE_DOMAIN:-}" ]; then
  HTTPS_URL="https://$BRIDGE_DOMAIN"
  HTTPS_BODY=$(curl -sf --max-time 15 "$HTTPS_URL/healthz" 2>/dev/null || echo "")
  if echo "$HTTPS_BODY" | grep -q '"status":"ok"'; then
    ok "HTTPS $HTTPS_URL → ok"
  else
    fail "HTTPS $HTTPS_URL → not reachable (check CF DNS + SSL mode)"
    FAILED=1
  fi
fi

# ── 9. validate-cleanup (if P1 scripts present) ──────────────────────────────────
VALIDATE_CLEANUP="$SCRIPT_DIR/validate-cleanup.sh"
if [ -f "$VALIDATE_CLEANUP" ]; then
  echo ""
  echo -e "${YLW}--- validate-cleanup @ $NODE_IP ---${NC}"
  if SSH_PRIVATE_KEY="${SSH_PRIVATE_KEY:-}" "$VALIDATE_CLEANUP" "$NODE_IP"; then
    ok "cleanup system healthy"
  else
    fail "cleanup validation failed on $NODE_IP"
    FAILED=1
  fi
fi

# ── Result ───────────────────────────────────────────────────────────────────────
echo ""
if [ "$FAILED" -eq 0 ]; then
  echo -e "${GRN}=== validate-cluster PASSED: $CLUSTER_NAME ===${NC}"
else
  echo -e "${RED}=== validate-cluster FAILED: $CLUSTER_NAME ===${NC}" >&2
  exit 1
fi
