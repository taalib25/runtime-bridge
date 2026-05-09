# runtime-node-core Helm Chart

Deploy, upgrade, and roll back the runtime node image in Kubernetes/K3s.

## Namespace and Secrets

```bash
kubectl create namespace zamo-workspace
```

```bash
kubectl -n zamo-workspace create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN" \
  --docker-email="$GHCR_EMAIL" \
  --dry-run=client -o yaml | kubectl apply -f -
```

```bash
kubectl -n zamo-workspace create secret generic runtime-node-core-secrets \
  --from-literal=OPENCODE_GO_API_KEY="$OPENCODE_GO_API_KEY" \
  --from-literal=OPENAI_API_KEY="${OPENAI_API_KEY:-}" \
  --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}" \
  --from-literal=OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}" \
  --dry-run=client -o yaml | kubectl apply -f -
```

## Install / Upgrade

```bash
helm upgrade --install runtime-test ./charts/runtime-node-core \
  --namespace zamo-workspace \
  --set image.repository=ghcr.io/taalib25/runtime-node-core \
  --set image.tag=0.1.0 \
  --set imagePullSecrets.enabled=true \
  --set imagePullSecrets.name=ghcr-pull-secret \
  --set secrets.existingSecret=runtime-node-core-secrets
```

## Test

```bash
kubectl -n zamo-workspace rollout status deployment/runtime-test-runtime-node-core
kubectl -n zamo-workspace port-forward svc/runtime-test-runtime-node-core 8787:8787
curl -i http://localhost:8787/health
```

## Upgrade and Rollback

```bash
helm upgrade runtime-test ./charts/runtime-node-core \
  --namespace zamo-workspace \
  --set image.repository=ghcr.io/taalib25/runtime-node-core \
  --set image.tag=0.1.1
```

```bash
helm rollback runtime-test
```

## Digest Pinning

```bash
helm upgrade --install runtime-test ./charts/runtime-node-core \
  --namespace zamo-workspace \
  --set image.repository=ghcr.io/taalib25/runtime-node-core \
  --set image.digest=sha256:<digest> \
  --set image.tag=""
```
