#!/usr/bin/env bash
# Usage: deploy-bridge.sh <vm-ip> <bridge-host> <git-sha> [cluster-name]
# Deploys the bridge to a fresh cluster node.
# Prerequisites: SSH access to vm-ip, GHCR_PAT env var set, docker available locally.
#
# Example:
#   GHCR_PAT=ghp_xxx deploy-bridge.sh 1.2.3.4 bridge-sg.hermeshq.net abc1234 hermes-sg-1
set -euo pipefail

VM_IP="${1:?Usage: deploy-bridge.sh <vm-ip> <bridge-host> <git-sha> [cluster-name]}"
BRIDGE_HOST="${2:?bridge host required, e.g. bridge-sg.hermeshq.net}"
GIT_SHA="${3:?git sha required}"
CLUSTER_NAME="${4:-hermes-$(echo "$VM_IP" | tr '.' '-')}"
GHCR_PAT="${GHCR_PAT:?GHCR_PAT env var required}"
K="KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl"
REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
IMAGE="ghcr.io/taalib25/hermes-bridge"

echo "==> Deploying bridge to $VM_IP ($CLUSTER_NAME), host: $BRIDGE_HOST"

# 1. Configure k3s private registry — write once, all pulls authenticated automatically
echo "--- Configuring k3s registries.yaml for ghcr.io"
ssh root@"$VM_IP" bash << ENDSSH
  mkdir -p /etc/rancher/k3s
  cat > /etc/rancher/k3s/registries.yaml << EOF
configs:
  "ghcr.io":
    auth:
      username: taalib25
      password: ${GHCR_PAT}
EOF
  if ! cmp -s /etc/rancher/k3s/registries.yaml.prev /etc/rancher/k3s/registries.yaml 2>/dev/null; then
    cp /etc/rancher/k3s/registries.yaml /etc/rancher/k3s/registries.yaml.prev
    systemctl restart k3s
    sleep 10
    echo "k3s restarted with updated registry auth"
  fi
ENDSSH

# 2. Build and push bridge image to private GHCR
echo "--- Building and pushing bridge image: $IMAGE:$GIT_SHA"
echo "$GHCR_PAT" | docker login ghcr.io -u taalib25 --password-stdin
docker build \
  --build-arg GIT_SHA="$GIT_SHA" \
  -t "$IMAGE:$GIT_SHA" \
  -t "$IMAGE:latest" \
  "$REPO_ROOT"
docker push "$IMAGE:$GIT_SHA"
docker push "$IMAGE:latest"

# 3. Namespace
echo "--- Creating namespace hermes-bridge"
ssh root@"$VM_IP" "$K create namespace hermes-bridge --dry-run=client -o yaml | $K apply -f -"

# 4. Bridge auth secret (skip if already exists)
if ! ssh root@"$VM_IP" "$K -n hermes-bridge get secret bridge-auth &>/dev/null"; then
  BRIDGE_SECRET=$(openssl rand -hex 32)
  ssh root@"$VM_IP" "$K create secret generic bridge-auth -n hermes-bridge --from-literal=secret=$BRIDGE_SECRET"
  echo "--- Created bridge-auth secret: $BRIDGE_SECRET"
  echo "    !! Save this — register it in the backend clusters table !!"
else
  echo "--- bridge-auth already exists, skipping"
  BRIDGE_SECRET=$(ssh root@"$VM_IP" "$K -n hermes-bridge get secret bridge-auth -o jsonpath='{.data.secret}' | base64 -d")
fi

# 5. RBAC + service
echo "--- Applying RBAC and service"
scp "$REPO_ROOT/deploy/rbac.yaml" "$REPO_ROOT/deploy/service.yaml" root@"$VM_IP":/tmp/
ssh root@"$VM_IP" "$K apply -f /tmp/rbac.yaml -f /tmp/service.yaml"

# 6. Deployment manifest — image is pulled from GHCR automatically via registries.yaml
echo "--- Applying deployment"
DEPLOY_TMP=$(mktemp)
sed \
  -e "s|ghcr.io/taalib25/hermes-bridge:__GIT_SHA__|$IMAGE:$GIT_SHA|g" \
  -e "s|value: \"hermes-test\"|value: \"$CLUSTER_NAME\"|g" \
  "$REPO_ROOT/deploy/deployment-test.yaml" > "$DEPLOY_TMP"
scp "$DEPLOY_TMP" root@"$VM_IP":/tmp/deployment.yaml
ssh root@"$VM_IP" "$K apply -f /tmp/deployment.yaml"
rm "$DEPLOY_TMP"

# 7. Rollout — containerd pulls from GHCR using registries.yaml auth
echo "--- Waiting for rollout"
ssh root@"$VM_IP" "$K rollout restart deployment/hermes-bridge -n hermes-bridge"
ssh root@"$VM_IP" "$K rollout status deployment/hermes-bridge -n hermes-bridge --timeout=120s"

# 8. Smoke test
echo "--- Smoke test"
ssh root@"$VM_IP" bash << ENDSSH
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  SECRET=\$(kubectl -n hermes-bridge get secret bridge-auth -o jsonpath='{.data.secret}' | base64 -d)
  curl -sf -H "X-Bridge-Secret: \$SECRET" http://localhost:8080/healthz | grep '"status":"ok"' && echo "healthz OK"
  curl -sf -H "X-Bridge-Secret: \$SECRET" http://localhost:8080/readyz | grep '"status":"ready"' && echo "readyz OK"
  curl -sf -H "X-Bridge-Secret: \$SECRET" http://localhost:8080/v1/cluster/summary | grep '"kubernetesReachable":true' && echo "k8s reachable"
ENDSSH

echo ""
echo "==> Done. Next steps:"
echo "    1. Add DNS A record: $BRIDGE_HOST -> $VM_IP (Cloudflare, orange cloud ON)"
echo "    2. Register in backend clusters table:"
echo "       bridge_url: https://$BRIDGE_HOST"
echo "       cluster_name: $CLUSTER_NAME"
echo "       bridge_secret: $BRIDGE_SECRET"
