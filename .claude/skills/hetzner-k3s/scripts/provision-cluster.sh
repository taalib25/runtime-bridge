#!/usr/bin/env bash
# provision-cluster.sh — provision a new HermesCloud cluster end-to-end
#
# Wraps `hetzner-k3s create` and handles everything after:
#   - Creates the k3s cluster
#   - Discovers the node IP
#   - Creates the GitHub Environment + 5 secrets
#   - Configures /etc/rancher/k3s/registries.yaml (GHCR auth)
#   - Builds + pushes the bridge image to GHCR
#   - Applies bridge manifests + rollout
#   - Applies Traefik ingress (host: bridge-<CLUSTER_NAME>.hermeshq.net)
#   - Creates Cloudflare DNS A record (proxied) if CF_TOKEN/CF_ZONE_ID are set
#   - Waits for HTTPS to become reachable
#   - Smoke tests bridge
#   - Registers the cluster with the backend (using HTTPS URL)
#   - Runs validate-cluster.sh for final end-to-end verification
#
# Required env vars:
#   HCLOUD_TOKEN        Hetzner Cloud API token (read+write)
#   GHCR_PAT            GitHub PAT with packages:read/write
#   ADMIN_API_SECRET    HermesCloud backend admin API key
#   SSH_PRIVATE_KEY     Content of SSH private key (not path) — export "$(cat ~/.ssh/id_ed25519)"
#
# Optional:
#   BRIDGE_SECRET       Bridge auth secret — auto-generated if unset
#   BACKEND_URL         Default: https://api.hermeshq.net
#   CF_TOKEN            Cloudflare API token — enables automated DNS record creation
#   CF_ZONE_ID          Cloudflare zone ID for hermeshq.net
#   DRY_RUN=true        Print plan, make no changes
#
# Usage:
#   ./provision-cluster.sh <config-file> <cluster-name> <region> [gh-env-name]
#
# Example:
#   export HCLOUD_TOKEN=... GHCR_PAT=... ADMIN_API_SECRET=... SSH_PRIVATE_KEY="$(cat ~/.ssh/id_ed25519)"
#   export CF_TOKEN=... CF_ZONE_ID=...
#   ./provision-cluster.sh cluster-config-eu2.yaml hermes-eu-2 eu
#   DRY_RUN=true ./provision-cluster.sh cluster-config-eu2.yaml hermes-eu-2 eu
set -euo pipefail

# ── Args ──────────────────────────────────────────────────────────────────────
CONFIG="${1:?Usage: provision-cluster.sh <config-file> <cluster-name> <region> [gh-env-name]}"
CLUSTER_NAME="${2:?cluster-name required}"
REGION="${3:?region required (eu|us|ap)}"
GH_ENV="${4:-$CLUSTER_NAME}"
REPO="taalib25/hermes-runtime-operator"
BACKEND_URL="${BACKEND_URL:-https://api.hermeshq.net}"
DRY_RUN="${DRY_RUN:-false}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BRIDGE_DOMAIN="bridge-${CLUSTER_NAME}.hermeshq.net"

# ── Colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; YLW='\033[1;33m'; GRN='\033[0;32m'; BLU='\033[0;34m'; NC='\033[0m'
info() { echo -e "${BLU}[provision]${NC} $*"; }
ok()   { echo -e "${GRN}[ok]${NC} $*"; }
warn() { echo -e "${YLW}[warn]${NC} $*"; }
die()  { echo -e "${RED}[error]${NC} $*" >&2; exit 1; }
run()  {
  if [ "$DRY_RUN" = "true" ]; then echo -e "${YLW}[dry-run]${NC} $*"
  else "$@"; fi
}

# ── 1. Preflight ──────────────────────────────────────────────────────────────
info "Preflight checks..."
[ -f "$CONFIG" ] || die "Config file not found: $CONFIG"

for tool in hetzner-k3s gh curl jq docker; do
  command -v "$tool" &>/dev/null || die "Required tool not found: $tool"
done

[ -n "${HCLOUD_TOKEN:-}"     ] || die "HCLOUD_TOKEN not set"
[ -n "${GHCR_PAT:-}"         ] || die "GHCR_PAT not set"
[ -n "${ADMIN_API_SECRET:-}" ] || die "ADMIN_API_SECRET not set"
[ -n "${SSH_PRIVATE_KEY:-}"  ] || die "SSH_PRIVATE_KEY not set"
gh auth status &>/dev/null    || die "gh not authenticated — run: gh auth login"

CONFIG_CLUSTER=$(grep 'cluster_name:' "$CONFIG" | awk '{print $2}' | tr -d '"')
[ "$CONFIG_CLUSTER" = "$CLUSTER_NAME" ] || \
  die "cluster_name in config ('$CONFIG_CLUSTER') ≠ argument ('$CLUSTER_NAME')"

CF_DNS=false
if [ -n "${CF_TOKEN:-}" ] && [ -n "${CF_ZONE_ID:-}" ]; then
  CF_DNS=true
fi

ok "Preflight passed"

# ── 2. Conflict checks ────────────────────────────────────────────────────────
info "Checking for existing cluster..."

EXISTING=$(curl -sf -H "Authorization: Bearer $HCLOUD_TOKEN" \
  "https://api.hetzner.cloud/v1/servers?label_selector=hetzner-k3s-cluster=$CLUSTER_NAME" \
  | jq -r '.servers | length')
[ "${EXISTING:-0}" -eq 0 ] || \
  die "Cluster '$CLUSTER_NAME' already has $EXISTING servers in Hetzner. Use create-cluster.sh to reconcile."

gh api "repos/$REPO/environments/$GH_ENV" &>/dev/null 2>&1 && \
  die "GitHub Environment '$GH_ENV' already exists. Delete it first or use a different name."

ok "No conflicts found"

# ── 3. Plan + confirmation ────────────────────────────────────────────────────
DNS_NOTE="(skipped — CF_TOKEN/CF_ZONE_ID not set)"
[ "$CF_DNS" = "true" ] && DNS_NOTE="bridge-${CLUSTER_NAME}.hermeshq.net → $CF_DNS"
echo ""
echo -e "${YLW}╔═══════════════════════════════════════════╗${NC}"
echo -e "${YLW}║      NEW CLUSTER PROVISIONING PLAN        ║${NC}"
echo -e "${YLW}╚═══════════════════════════════════════════╝${NC}"
printf "  %-20s %s\n" "Cluster:"      "$CLUSTER_NAME"
printf "  %-20s %s\n" "Config:"       "$CONFIG"
printf "  %-20s %s\n" "Region:"       "$REGION"
printf "  %-20s %s\n" "GitHub env:"   "$GH_ENV ($REPO)"
printf "  %-20s %s\n" "Bridge FQDN:"  "https://$BRIDGE_DOMAIN"
printf "  %-20s %s\n" "CF DNS:"       "$([ "$CF_DNS" = "true" ] && echo "auto (proxied)" || echo "manual — set CF_TOKEN + CF_ZONE_ID")"
printf "  %-20s %s\n" "Backend:"      "$BACKEND_URL"
printf "  %-20s %s\n" "Dry run:"      "$DRY_RUN"
echo ""
echo "  Steps:"
echo "    1   hetzner-k3s create --config $CONFIG"
echo "    2   Discover node IP from Hetzner API"
echo "    3   Create GitHub Environment '$GH_ENV' with 5 secrets"
echo "    4   Configure /etc/rancher/k3s/registries.yaml on node"
echo "    5   Build + push bridge image to GHCR (private)"
echo "    6   Apply bridge manifests + rollout"
echo "    7   Apply Traefik ingress (host: $BRIDGE_DOMAIN)"
if [ "$CF_DNS" = "true" ]; then
  echo "    8   Create Cloudflare DNS A record: $BRIDGE_DOMAIN → <node-ip> (proxied)"
  echo "    9   Wait for HTTPS on https://$BRIDGE_DOMAIN"
else
  echo "    8   [manual] Create CF DNS A record for $BRIDGE_DOMAIN"
  echo "    9   [skip]   HTTPS wait (no CF automation)"
fi
echo "   10   Smoke test bridge"
echo "   11   Register cluster with backend (HTTPS URL)"
echo "   12   validate-cluster.sh end-to-end verification"
echo ""

[ "$DRY_RUN" = "true" ] && { echo -e "${YLW}DRY_RUN=true — stopping here.${NC}"; exit 0; }

read -rp "Type cluster name to confirm ($CLUSTER_NAME): " confirm
[ "$confirm" = "$CLUSTER_NAME" ] || { echo "Aborted."; exit 0; }
echo ""

# ── 4. Create cluster ─────────────────────────────────────────────────────────
info "Step 1/12 — Creating k3s cluster (this takes ~5 min)..."
run hetzner-k3s create --config "$CONFIG" --quiet
ok "Cluster created"

# ── 5. Discover IP ────────────────────────────────────────────────────────────
info "Step 2/12 — Discovering node IP..."
NODE_IP=""
for i in $(seq 1 18); do
  NODE_IP=$(curl -sf -H "Authorization: Bearer $HCLOUD_TOKEN" \
    "https://api.hetzner.cloud/v1/servers?label_selector=hetzner-k3s-cluster=$CLUSTER_NAME" \
    | jq -r '.servers[] | select(.status=="running") | .public_net.ipv4.ip' | head -1)
  [ -n "$NODE_IP" ] && break
  warn "Waiting for running node... ($i/18)"
  sleep 10
done
[ -n "$NODE_IP" ] || die "Could not find a running node for cluster '$CLUSTER_NAME'"
ok "Node IP: $NODE_IP"

# ── 6. Wait for SSH ───────────────────────────────────────────────────────────
info "Waiting for SSH on $NODE_IP..."
for i in $(seq 1 30); do
  ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no \
      -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" exit 2>/dev/null && break
  [ "$i" -eq 30 ] && die "SSH never became available on $NODE_IP"
  sleep 10
done
ok "SSH ready"

# ── 7. GitHub Environment + secrets ──────────────────────────────────────────
info "Step 3/12 — Creating GitHub Environment '$GH_ENV'..."
BRIDGE_SECRET="${BRIDGE_SECRET:-$(openssl rand -hex 32)}"
run gh api --method PUT "repos/$REPO/environments/$GH_ENV" --silent
run gh secret set NODE_IP          --repo "$REPO" --env "$GH_ENV" --body "$NODE_IP"
run gh secret set SSH_PRIVATE_KEY  --repo "$REPO" --env "$GH_ENV" --body "$SSH_PRIVATE_KEY"
run gh secret set BRIDGE_SECRET    --repo "$REPO" --env "$GH_ENV" --body "$BRIDGE_SECRET"
run gh secret set GHCR_PAT         --repo "$REPO" --env "$GH_ENV" --body "$GHCR_PAT"
run gh secret set ADMIN_API_SECRET --repo "$REPO" --env "$GH_ENV" --body "$ADMIN_API_SECRET"
ok "GitHub Environment created with 5 secrets"

# ── 8. registries.yaml ────────────────────────────────────────────────────────
info "Step 4/12 — Configuring GHCR auth on node..."
run ssh -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" bash << ENDSSH
  mkdir -p /etc/rancher/k3s
  cat > /etc/rancher/k3s/registries.yaml << EOF
configs:
  "ghcr.io":
    auth:
      username: taalib25
      password: ${GHCR_PAT}
EOF
  systemctl restart k3s && sleep 10
ENDSSH
ok "registries.yaml configured — GHCR auth active"

# ── 9. Build + push bridge image ─────────────────────────────────────────────
info "Step 5/12 — Building + pushing bridge image..."
GIT_SHA=$(git -C "$REPO_ROOT" rev-parse --short HEAD)
IMAGE="ghcr.io/taalib25/hermes-bridge"
run echo "$GHCR_PAT" | docker login ghcr.io -u taalib25 --password-stdin --quiet
run docker build --build-arg GIT_SHA="$GIT_SHA" \
    -t "$IMAGE:$GIT_SHA" -t "$IMAGE:latest" "$REPO_ROOT" --quiet
run docker push "$IMAGE:$GIT_SHA" --quiet
run docker push "$IMAGE:latest"   --quiet
ok "Bridge image pushed: $IMAGE:$GIT_SHA"

# ── 10. Apply bridge manifests ────────────────────────────────────────────────
info "Step 6/12 — Deploying bridge..."
DEPLOY_TMP=$(mktemp)
sed \
  -e "s|ghcr.io/taalib25/hermes-bridge:__GIT_SHA__|$IMAGE:$GIT_SHA|g" \
  -e "s|value: \"hermes-test\"|value: \"$CLUSTER_NAME\"|g" \
  "$REPO_ROOT/deploy/deployment-test.yaml" > "$DEPLOY_TMP"

# Copy static manifests, then the rendered deployment with an explicit remote name.
run scp -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") -q \
  "$REPO_ROOT/deploy/rbac.yaml" \
  "$REPO_ROOT/deploy/service.yaml" \
  root@"$NODE_IP":/tmp/
run scp -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") -q \
  "$DEPLOY_TMP" root@"$NODE_IP":/tmp/deployment.yaml

run ssh -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" bash << ENDSSH
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  kubectl create namespace hermes-bridge --dry-run=client -o yaml | kubectl apply -f - -q
  kubectl -n hermes-bridge create secret generic bridge-auth \
    --from-literal=secret=${BRIDGE_SECRET} --dry-run=client -o yaml | kubectl apply -f - -q
  kubectl apply -f /tmp/rbac.yaml -q
  kubectl apply -f /tmp/service.yaml -q
  kubectl apply -f /tmp/deployment.yaml -q
  kubectl rollout restart deployment/hermes-bridge -n hermes-bridge
  kubectl rollout status deployment/hermes-bridge -n hermes-bridge --timeout=120s
ENDSSH
rm -f "$DEPLOY_TMP"
ok "Bridge deployed"

# ── 11. Apply ingress ─────────────────────────────────────────────────────────
info "Step 7/12 — Applying Traefik ingress (host: $BRIDGE_DOMAIN)..."
INGRESS_TMP=$(mktemp)
sed "s|__CLUSTER_NAME__|$CLUSTER_NAME|g" \
  "$REPO_ROOT/deploy/ingress.yaml" > "$INGRESS_TMP"

run scp -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") -q \
  "$INGRESS_TMP" root@"$NODE_IP":/tmp/ingress.yaml
run ssh -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" \
  "KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl apply -f /tmp/ingress.yaml -q"
rm -f "$INGRESS_TMP"
ok "Ingress applied"

# ── 12. Cloudflare DNS ────────────────────────────────────────────────────────
info "Step 8/12 — DNS..."
if [ "$CF_DNS" = "true" ]; then
  EXISTING_RECORD=$(curl -sf \
    "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records?type=A&name=bridge-${CLUSTER_NAME}.hermeshq.net" \
    -H "Authorization: Bearer $CF_TOKEN" | jq -r '.result | length')
  if [ "${EXISTING_RECORD:-0}" -gt 0 ]; then
    warn "DNS record for $BRIDGE_DOMAIN already exists — skipping"
  else
    CF_RESULT=$(run curl -sf -X POST \
      "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records" \
      -H "Authorization: Bearer $CF_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"type\":\"A\",\"name\":\"bridge-${CLUSTER_NAME}.hermeshq.net\",\"content\":\"$NODE_IP\",\"ttl\":1,\"proxied\":true}" \
      || echo '{"success":false}')
    echo "$CF_RESULT" | grep -q '"success":true' || \
      die "Cloudflare DNS creation failed: $CF_RESULT"
    ok "DNS A record created: $BRIDGE_DOMAIN → $NODE_IP (proxied)"
  fi
else
  warn "CF_TOKEN/CF_ZONE_ID not set — create DNS record manually:"
  warn "  $BRIDGE_DOMAIN → $NODE_IP  (Cloudflare proxied=true)"
fi

# ── 13. Wait for HTTPS ────────────────────────────────────────────────────────
info "Step 9/12 — Waiting for HTTPS..."
if [ "$CF_DNS" = "true" ]; then
  for i in $(seq 1 24); do
    if curl -sf --max-time 10 "https://$BRIDGE_DOMAIN/healthz" >/dev/null 2>&1; then
      ok "HTTPS reachable: https://$BRIDGE_DOMAIN"
      break
    fi
    [ "$i" -eq 24 ] && die "HTTPS did not come up at https://$BRIDGE_DOMAIN — check CF DNS / SSL mode"
    warn "Waiting for Cloudflare to activate... ($i/24)"
    sleep 5
  done
else
  warn "Skipping HTTPS wait — no CF automation. HTTPS will work once DNS is manually created."
fi

# ── 14. Smoke test ────────────────────────────────────────────────────────────
info "Step 10/12 — Smoke testing..."
run ssh -o StrictHostKeyChecking=no -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" bash << ENDSSH
  S="${BRIDGE_SECRET}"
  curl -sf -H "X-Bridge-Secret: \$S" http://localhost:8080/healthz             | grep '"status":"ok"'
  curl -sf -H "X-Bridge-Secret: \$S" http://localhost:8080/readyz              | grep '"status":"ready"'
  curl -sf -H "X-Bridge-Secret: \$S" http://localhost:8080/v1/cluster/summary  | grep '"kubernetesReachable":true'
ENDSSH
ok "All smoke tests passed"

# ── 15. Register with backend ─────────────────────────────────────────────────
info "Step 11/12 — Registering with backend..."
if [ "$CF_DNS" = "true" ]; then
  BRIDGE_REGISTER_URL="https://$BRIDGE_DOMAIN"
else
  BRIDGE_REGISTER_URL="http://$NODE_IP:8080"
  warn "Registering with HTTP URL — update to https://$BRIDGE_DOMAIN after DNS is created"
fi

run curl -sf -X POST "$BACKEND_URL/api/admin/clusters" \
  -H "Authorization: Bearer $ADMIN_API_SECRET" \
  -H "Content-Type: application/json" \
  -d "{\"cluster_id\":\"$CLUSTER_NAME\",\"bridge_url\":\"$BRIDGE_REGISTER_URL\",\"bridge_secret\":\"$BRIDGE_SECRET\",\"region\":\"$REGION\",\"status\":\"active\"}"
ok "Cluster registered at $BRIDGE_REGISTER_URL"

# ── 16. End-to-end validation ─────────────────────────────────────────────────
info "Step 12/12 — End-to-end validation..."
VALIDATE="$SCRIPT_DIR/validate-cluster.sh"
if [ -f "$VALIDATE" ]; then
  BRIDGE_DOMAIN="$BRIDGE_DOMAIN" \
  BRIDGE_SECRET="$BRIDGE_SECRET" \
  SSH_PRIVATE_KEY="$SSH_PRIVATE_KEY" \
    run "$VALIDATE" "$CLUSTER_NAME" "$NODE_IP"
else
  warn "validate-cluster.sh not found in $SCRIPT_DIR — skipping"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GRN}╔═══════════════════════════════════════════╗${NC}"
echo -e "${GRN}║      CLUSTER READY                        ║${NC}"
echo -e "${GRN}╚═══════════════════════════════════════════╝${NC}"
printf "  %-20s %s\n" "Cluster:"      "$CLUSTER_NAME"
printf "  %-20s %s\n" "Node IP:"      "$NODE_IP"
printf "  %-20s %s\n" "Bridge URL:"   "${BRIDGE_REGISTER_URL:-http://$NODE_IP:8080}"
printf "  %-20s %s\n" "GitHub env:"   "$GH_ENV"
echo ""
if [ "$CF_DNS" != "true" ]; then
  echo "  Manual steps remaining:"
  echo "  1. Create Cloudflare DNS A record:"
  echo "       bridge-${CLUSTER_NAME}.hermeshq.net → $NODE_IP (proxied=true)"
  echo "  2. Update cluster bridge_url in backend to:"
  echo "       https://bridge-${CLUSTER_NAME}.hermeshq.net"
  echo ""
fi
echo "  Next: copy deploy-cluster-template.yml → deploy-${CLUSTER_NAME}.yml"
echo "        set cluster_name=$CLUSTER_NAME + region=$REGION"
echo "        Future bridge deploys are then fully automated on push to main."
echo ""
echo -e "  ${YLW}BRIDGE_SECRET (already in GitHub secrets — save a local copy):${NC}"
echo "  $BRIDGE_SECRET"
