# Hermes Agent Helm Chart - Quick Reference

## TL;DR - Deploy in 3 Steps

```bash
# 1. Clone chart
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart

# 2. Create values file
cat > values-kind.yaml << 'YAML'
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_KEY"
  API_SERVER_KEY: "test-key"

apiServer:
  enabled: true
  port: 8642

service:
  enabled: true
  type: ClusterIP

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      base_url: https://openrouter.ai/api/v1
YAML

# 3. Deploy
helm install hermes . \
  --namespace hermes \
  --create-namespace \
  -f values-kind.yaml
```

## Verify Deployment

```bash
# Check status
kubectl rollout status deployment/hermes -n hermes

# View logs
kubectl logs -n hermes deployment/hermes -f

# Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test API
curl http://localhost:8642/health
```

## Key Facts

| Item | Value |
|------|-------|
| **Chart Version** | 0.1.0 |
| **App Version** | 0.8.0 |
| **Min K8s** | 1.25.0+ |
| **Default Image** | nousresearch/hermes-agent:0.8.0 |
| **Default Replicas** | 1 (must stay 1 with persistence) |
| **Default Storage** | 5Gi PVC at /opt/data |
| **Default Strategy** | Recreate (for state safety) |
| **API Server Port** | 8642 (disabled by default) |
| **Webhook Port** | 8644 (disabled by default) |
| **Telegram Webhook Port** | 8443 (disabled by default) |

## Minimal Secrets Required

```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-..."  # Required for LLM
  API_SERVER_KEY: "..."             # Required if apiServer.enabled=true
```

## Common Configurations

### Gateway Only (No API)
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-..."

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      base_url: https://openrouter.ai/api/v1
```

### With API Server
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-..."
  API_SERVER_KEY: "your-key"

apiServer:
  enabled: true
  host: 0.0.0.0
  port: 8642

service:
  enabled: true
  type: ClusterIP
```

### With Webhooks
```yaml
webhook:
  enabled: true
  port: 8644

telegramWebhook:
  enabled: true
  url: https://hermes.example.com/telegram
  port: 8443

secrets:
  TELEGRAM_BOT_TOKEN: "..."
  WEBHOOK_SECRET: "..."
```

## Troubleshooting

| Issue | Check |
|-------|-------|
| Pod Pending | `kubectl describe pod -n hermes <pod>` |
| ImagePullBackOff | `docker pull nousresearch/hermes-agent:0.8.0` |
| CrashLoopBackOff | `kubectl logs -n hermes <pod> --previous` |
| API Not Responding | `kubectl exec -n hermes <pod> -- netstat -tlnp` |
| Config Not Found | `kubectl exec -n hermes <pod> -- ls -la /opt/data/` |

## Useful Commands

```bash
# Get all resources
kubectl get all -n hermes

# Describe deployment
kubectl describe deployment hermes -n hermes

# Check PVC
kubectl get pvc -n hermes

# Check secrets
kubectl get secrets -n hermes

# Exec into pod
kubectl exec -n hermes -it deployment/hermes -- /bin/bash

# View config
kubectl exec -n hermes deployment/hermes -- cat /opt/data/config.yaml

# Check environment
kubectl exec -n hermes deployment/hermes -- env | grep OPENROUTER

# Uninstall
helm uninstall hermes -n hermes
```

## Important Notes

1. **State Safety**: Hermes stores mutable state in `/opt/data`. Always use:
   - `replicaCount: 1`
   - `strategy.type: Recreate`
   - `persistence.enabled: true`

2. **Security**: Pod runs as non-root (UID 1000) with dropped capabilities

3. **Bootstrap**: Config is bootstrapped on every deploy (can be disabled with `bootstrap.overwrite: false`)

4. **Multi-tenancy**: Deploy one release per tenant, not multiple replicas

## Links

- **Chart**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Hermes**: https://github.com/nousresearch/hermes-agent
- **OpenRouter**: https://openrouter.ai
