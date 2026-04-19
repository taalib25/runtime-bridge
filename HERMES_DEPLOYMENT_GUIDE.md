# Hermes Agent Helm Chart - Kind Deployment Guide

**Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
**Commit SHA**: e3b685d4d0288668a37216435742cd0a659ebc6c
**Chart Version**: 0.1.0
**App Version**: 0.8.0
**Kubernetes Requirement**: >=1.25.0

---

## 1. How to Get the Chart

### Clone the Repository
```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
```

### Or Use Helm Directly (if published to a registry)
```bash
# Check if chart is available in a Helm repository
helm repo add ultraworkers https://charts.ultraworkers.ai  # (if available)
helm repo update
helm search repo hermes-agent
```

---

## 2. Minimal Values.yaml for Kind Deployment

### Simplest Configuration (Gateway Mode Only)
```yaml
# minimal-kind-values.yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_API_KEY_HERE"

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      provider: auto
      base_url: https://openrouter.ai/api/v1
```

**Deploy with:**
```bash
helm install hermes . \
  --namespace hermes \
  --create-namespace \
  -f minimal-kind-values.yaml
```

### With API Server Enabled (for testing)
```yaml
# kind-with-api-values.yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_API_KEY_HERE"
  API_SERVER_KEY: "your-api-server-key"

apiServer:
  enabled: true
  host: 0.0.0.0
  port: 8642

service:
  enabled: true
  type: ClusterIP

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      provider: auto
      base_url: https://openrouter.ai/api/v1
```

**Deploy with:**
```bash
helm install hermes . \
  --namespace hermes \
  --create-namespace \
  -f kind-with-api-values.yaml
```

---

## 3. Required Secrets

### Minimum Required
- **OPENROUTER_API_KEY**: OpenRouter API key (format: `sk-or-...`)
  - Get from: https://openrouter.ai/keys
  - Required for LLM inference

### Optional (Based on Features)
- **API_SERVER_KEY**: Required if `apiServer.enabled=true`
- **TELEGRAM_BOT_TOKEN**: Required if `telegramWebhook.enabled=true`
- **WEBHOOK_SECRET**: For webhook validation
- **ANTHROPIC_API_KEY**: If using Anthropic directly
- **OPENAI_API_KEY**: If using OpenAI directly
- **GOOGLE_API_KEY**: For Google services
- **EXA_API_KEY**: For web search
- **FIRECRAWL_API_KEY**: For web scraping
- **GITHUB_TOKEN**: For GitHub integration

### How to Provide Secrets

**Option 1: Inline in values.yaml (for testing only)**
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-..."
  API_SERVER_KEY: "your-key"
```

**Option 2: Create Secret First, Reference It**
```bash
kubectl create secret generic hermes-secrets \
  --from-literal=OPENROUTER_API_KEY=sk-or-... \
  --from-literal=API_SERVER_KEY=your-key \
  -n hermes

# Then in values.yaml:
secrets:
  existingSecret: hermes-secrets
```

**Option 3: Use External Secrets Operator (ESO)**
```yaml
externalSecret:
  enabled: true
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: my-secret-store
  target:
    name: hermes-secrets
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes/openrouter
        property: api_key
```

---

## 4. Default Ports and Services

### Container Ports (Inside Pod)
| Service | Port | Enabled By | Default |
|---------|------|-----------|---------|
| API Server | 8642 | `apiServer.enabled` | Disabled |
| Webhook | 8644 | `webhook.enabled` | Disabled |
| Telegram Webhook | 8443 | `telegramWebhook.enabled` | Disabled |

### Service Exposure
- **Service Type**: `ClusterIP` (default, suitable for Kind)
- **Service Enabled**: `service.enabled: false` (default)
- **Ingress Enabled**: `ingress.enabled: false` (default)

### Enable Service for Kind Testing
```yaml
service:
  enabled: true
  type: ClusterIP
  # Ports auto-derived from enabled listeners
```

### Port Forwarding for Local Testing
```bash
# Forward API server to localhost
kubectl port-forward -n hermes svc/hermes 8642:8642

# Then access at: http://localhost:8642
```

---

## 5. How to Verify the Deployment Works

### Check Deployment Status
```bash
# Watch deployment rollout
kubectl rollout status deployment/hermes -n hermes

# Check pod status
kubectl get pods -n hermes
kubectl describe pod -n hermes -l app.kubernetes.io/name=hermes-agent

# View logs
kubectl logs -n hermes -l app.kubernetes.io/name=hermes-agent -f
```

### Verify Configuration
```bash
# Check if config was bootstrapped
kubectl exec -n hermes -it deployment/hermes -- ls -la /opt/data/

# View the rendered config
kubectl exec -n hermes -it deployment/hermes -- cat /opt/data/config.yaml
```

### Test API Server (if enabled)
```bash
# Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test health endpoint
curl -X GET http://localhost:8642/health

# Test OpenAI-compatible endpoint
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_SERVER_KEY" \
  -d '{
    "model": "hermes-agent",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'
```

### Check Persistent Storage
```bash
# Verify PVC is bound
kubectl get pvc -n hermes

# Check storage usage
kubectl exec -n hermes -it deployment/hermes -- df -h /opt/data
```

### Verify Secrets Mounted
```bash
# Check if secrets are available in pod
kubectl exec -n hermes -it deployment/hermes -- env | grep OPENROUTER
```

### Common Issues & Troubleshooting

**Pod stuck in Pending**
```bash
kubectl describe pod -n hermes <pod-name>
# Check: PVC binding, resource requests, node capacity
```

**ImagePullBackOff**
```bash
# Default image: nousresearch/hermes-agent:0.8.0
# Verify image exists and is accessible
docker pull nousresearch/hermes-agent:0.8.0
```

**CrashLoopBackOff**
```bash
# Check logs for errors
kubectl logs -n hermes <pod-name> --previous

# Common causes:
# - Missing OPENROUTER_API_KEY
# - Invalid config.yaml syntax
# - Insufficient disk space in PVC
```

**API Server Not Responding**
```bash
# Verify service is created
kubectl get svc -n hermes

# Check if port is exposed
kubectl get endpoints -n hermes

# Verify pod is listening
kubectl exec -n hermes -it deployment/hermes -- netstat -tlnp | grep 8642
```

---

## 6. Complete Kind Deployment Example

### Step 1: Create Kind Cluster
```bash
kind create cluster --name hermes-test
```

### Step 2: Create Namespace
```bash
kubectl create namespace hermes
```

### Step 3: Create values.yaml
```bash
cat > /tmp/hermes-values.yaml << 'YAML'
fullnameOverride: hermes

image:
  repository: nousresearch/hermes-agent
  tag: "0.8.0"
  pullPolicy: IfNotPresent

replicaCount: 1

secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_KEY_HERE"
  API_SERVER_KEY: "test-api-key"

apiServer:
  enabled: true
  host: 0.0.0.0
  port: 8642

service:
  enabled: true
  type: ClusterIP

persistence:
  enabled: true
  size: 5Gi
  storageClass: ""

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      provider: auto
      base_url: https://openrouter.ai/api/v1
    agent:
      max_turns: 90
      gateway_timeout: 1800
    terminal:
      backend: local
      timeout: 180

resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: "2"
    memory: 4Gi
YAML
```

### Step 4: Install Chart
```bash
cd hermes-agent-helm-chart
helm install hermes . \
  --namespace hermes \
  -f /tmp/hermes-values.yaml
```

### Step 5: Wait for Deployment
```bash
kubectl rollout status deployment/hermes -n hermes --timeout=5m
```

### Step 6: Verify
```bash
# Check pod
kubectl get pods -n hermes

# Check logs
kubectl logs -n hermes deployment/hermes -f

# Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test
curl http://localhost:8642/health
```

### Step 7: Cleanup
```bash
helm uninstall hermes -n hermes
kubectl delete namespace hermes
kind delete cluster --name hermes-test
```

---

## 7. Key Configuration Options

### State Safety (Important!)
- **replicaCount**: Must be `1` when `persistence.enabled=true`
- **strategy.type**: Must be `Recreate` when persistence is enabled
- **Reason**: Hermes stores mutable state in `HERMES_HOME` (/opt/data)

### Bootstrap Configuration
```yaml
bootstrap:
  enabled: true              # Enable config bootstrapping
  overwrite: true            # Overwrite persisted config on each deploy
  existingConfigMap: ""      # Use external ConfigMap instead
  configKey: config.yaml     # Key in ConfigMap
  soulKey: SOUL.md          # Optional SOUL file
```

### Model Configuration
```yaml
config:
  values:
    model:
      default: anthropic/claude-opus-4.6  # Default model
      provider: auto                       # Auto-detect provider
      base_url: https://openrouter.ai/api/v1
```

### Security Context (Non-root by default)
```yaml
podSecurityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  runAsNonRoot: true

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
```

---

## 8. Reference Links

- **Chart Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Hermes Agent**: https://github.com/nousresearch/hermes-agent
- **OpenRouter API**: https://openrouter.ai
- **Helm Documentation**: https://helm.sh/docs/
- **Kubernetes Documentation**: https://kubernetes.io/docs/

---

## 9. Chart Structure

```
hermes-agent-helm-chart/
├── Chart.yaml                 # Chart metadata
├── values.yaml               # Default values
├── values.schema.json        # Schema validation
├── templates/
│   ├── deployment.yaml       # Main deployment
│   ├── service.yaml          # Kubernetes Service
│   ├── configmap.yaml        # Config bootstrap
│   ├── secret.yaml           # Secrets
│   ├── pvc.yaml              # Persistent Volume Claim
│   ├── ingress.yaml          # Ingress (optional)
│   ├── networkpolicy.yaml    # Network policies (optional)
│   ├── rbac.yaml             # RBAC rules (optional)
│   └── ...
├── crds/                     # Custom Resource Definitions
├── ci/                       # Test values
└── docs/                     # Documentation
```

