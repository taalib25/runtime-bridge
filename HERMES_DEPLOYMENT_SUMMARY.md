# Hermes Agent Helm Chart - Complete Analysis Summary

## Executive Summary

The **ultraworkers/hermes-agent-helm-chart** is an unofficial community Helm chart for deploying Nous Research's Hermes Agent on Kubernetes. It's production-ready for Kind clusters with sensible defaults for state safety, security, and composability.

**Key Takeaway**: Deploy Hermes in Kind with 3 commands:
```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
helm install hermes . --namespace hermes --create-namespace \
  --set secrets.OPENROUTER_API_KEY=sk-or-YOUR_KEY \
  --set apiServer.enabled=true \
  --set service.enabled=true
```

---

## 1. How to Get the Chart

### Clone Repository
```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
```

**Repository Details**:
- **URL**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Latest Commit**: e3b685d4d0288668a37216435742cd0a659ebc6c
- **Chart Version**: 0.1.0
- **App Version**: 0.8.0
- **Min Kubernetes**: 1.25.0+

### Chart Structure
```
hermes-agent-helm-chart/
├── Chart.yaml                 # Metadata
├── values.yaml               # 354 lines of defaults
├── values.schema.json        # Validation schema
├── templates/                # 16 template files
│   ├── deployment.yaml       # Main workload (338 lines)
│   ├── service.yaml          # Network exposure
│   ├── configmap.yaml        # Config bootstrap
│   ├── secret.yaml           # Credentials
│   ├── pvc.yaml              # Persistent storage
│   ├── ingress.yaml          # HTTP routing
│   ├── virtualservice.yaml   # Istio support
│   ├── networkpolicy.yaml    # Network isolation
│   ├── rbac.yaml             # Kubernetes API access
│   ├── external-secret.yaml  # ESO integration
│   ├── pdb.yaml              # Pod disruption budget
│   └── ...
├── crds/                     # Custom Resource Definitions
├── ci/                       # Test configurations
└── docs/                     # Documentation
```

---

## 2. Minimal Values.yaml for Kind Deployment

### Absolute Minimum (Gateway Only)
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_API_KEY"

config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      base_url: https://openrouter.ai/api/v1
```

### Recommended for Testing (With API Server)
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_API_KEY"
  API_SERVER_KEY: "test-api-key"

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
      base_url: https://openrouter.ai/api/v1
```

### Deploy Command
```bash
helm install hermes . \
  --namespace hermes \
  --create-namespace \
  -f values-kind.yaml
```

---

## 3. Required Secrets

### Mandatory
| Secret | Purpose | Format | Source |
|--------|---------|--------|--------|
| **OPENROUTER_API_KEY** | LLM inference | `sk-or-...` | https://openrouter.ai/keys |

### Conditional (Based on Features)
| Secret | Required When | Format |
|--------|---------------|--------|
| API_SERVER_KEY | `apiServer.enabled=true` | Any string |
| TELEGRAM_BOT_TOKEN | `telegramWebhook.enabled=true` | Bot token |
| WEBHOOK_SECRET | Webhook validation | Any string |
| ANTHROPIC_API_KEY | Using Anthropic directly | API key |
| OPENAI_API_KEY | Using OpenAI directly | API key |
| GOOGLE_API_KEY | Google services | API key |
| EXA_API_KEY | Web search | API key |
| FIRECRAWL_API_KEY | Web scraping | API key |
| GITHUB_TOKEN | GitHub integration | Personal access token |

### How to Provide Secrets

**Option 1: Inline (Testing Only)**
```yaml
secrets:
  OPENROUTER_API_KEY: "sk-or-..."
  API_SERVER_KEY: "test-key"
```

**Option 2: Pre-created Secret**
```bash
kubectl create secret generic hermes-secrets \
  --from-literal=OPENROUTER_API_KEY=sk-or-... \
  --from-literal=API_SERVER_KEY=test-key \
  -n hermes

# In values.yaml:
secrets:
  existingSecret: hermes-secrets
```

**Option 3: External Secrets Operator**
```yaml
externalSecret:
  enabled: true
  secretStoreRef:
    kind: ClusterSecretStore
    name: my-secret-store
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes/openrouter
```

---

## 4. Default Ports and Services

### Container Ports
| Service | Port | Enabled By | Default |
|---------|------|-----------|---------|
| API Server | 8642 | `apiServer.enabled` | ❌ Disabled |
| Webhook | 8644 | `webhook.enabled` | ❌ Disabled |
| Telegram Webhook | 8443 | `telegramWebhook.enabled` | ❌ Disabled |

### Service Configuration
- **Default Service Type**: `ClusterIP` (suitable for Kind)
- **Default Service Enabled**: `false`
- **Default Ingress Enabled**: `false`

### Enable Service for Kind
```yaml
service:
  enabled: true
  type: ClusterIP
  # Ports auto-derived from enabled listeners
```

### Port Forwarding for Local Testing
```bash
kubectl port-forward -n hermes svc/hermes 8642:8642
# Access at: http://localhost:8642
```

---

## 5. How to Verify the Deployment Works

### Step 1: Check Deployment Status
```bash
# Watch rollout
kubectl rollout status deployment/hermes -n hermes

# Check pod
kubectl get pods -n hermes
kubectl describe pod -n hermes -l app.kubernetes.io/name=hermes-agent

# View logs
kubectl logs -n hermes deployment/hermes -f
```

### Step 2: Verify Configuration
```bash
# Check bootstrap
kubectl exec -n hermes deployment/hermes -- ls -la /opt/data/

# View config
kubectl exec -n hermes deployment/hermes -- cat /opt/data/config.yaml

# Check secrets
kubectl exec -n hermes deployment/hermes -- env | grep OPENROUTER
```

### Step 3: Test API Server (if enabled)
```bash
# Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# Test health
curl http://localhost:8642/health

# Test chat endpoint
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_SERVER_KEY" \
  -d '{
    "model": "hermes-agent",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'
```

### Step 4: Check Storage
```bash
# Verify PVC
kubectl get pvc -n hermes

# Check usage
kubectl exec -n hermes deployment/hermes -- df -h /opt/data
```

### Common Issues & Fixes

| Issue | Check | Fix |
|-------|-------|-----|
| Pod Pending | `kubectl describe pod` | Check PVC binding, resources |
| ImagePullBackOff | `docker pull nousresearch/hermes-agent:0.8.0` | Verify image exists |
| CrashLoopBackOff | `kubectl logs --previous` | Check OPENROUTER_API_KEY, config.yaml |
| API Not Responding | `kubectl exec -- netstat -tlnp \| grep 8642` | Verify service created |

---

## 6. Complete Kind Deployment Example

### Full Step-by-Step

```bash
# 1. Create Kind cluster
kind create cluster --name hermes-test

# 2. Clone chart
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart

# 3. Create values file
cat > values-kind.yaml << 'YAML'
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

# 4. Install chart
helm install hermes . \
  --namespace hermes \
  --create-namespace \
  -f values-kind.yaml

# 5. Wait for deployment
kubectl rollout status deployment/hermes -n hermes --timeout=5m

# 6. Verify
kubectl get pods -n hermes
kubectl logs -n hermes deployment/hermes -f

# 7. Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642 &

# 8. Test
curl http://localhost:8642/health

# 9. Cleanup
helm uninstall hermes -n hermes
kubectl delete namespace hermes
kind delete cluster --name hermes-test
```

---

## 7. Key Configuration Options

### State Safety (CRITICAL!)
```yaml
replicaCount: 1                    # Must be 1 with persistence
strategy:
  type: Recreate                   # Must be Recreate with persistence
persistence:
  enabled: true                    # Recommended for Kind
```

**Why**: Hermes stores mutable state in `/opt/data` (HERMES_HOME). Multiple replicas would corrupt data.

### Bootstrap Configuration
```yaml
bootstrap:
  enabled: true                    # Enable config seeding
  overwrite: true                  # Helm is source of truth
  existingConfigMap: ""            # Use external ConfigMap if set
  configKey: config.yaml
  soulKey: SOUL.md
```

### Model Configuration
```yaml
config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      provider: auto
      base_url: https://openrouter.ai/api/v1
    agent:
      max_turns: 90
      gateway_timeout: 1800
      restart_drain_timeout: 60
    terminal:
      backend: local
      timeout: 180
    browser:
      inactivity_timeout: 120
      command_timeout: 30
    compression:
      enabled: true
      threshold: 0.5
```

### Security Context (Non-root by Default)
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

## 8. Deployment Modes

### Mode 1: Direct Deployment (Default)
- Helm manages full workload lifecycle
- Renders Deployment, Service, Ingress, etc.
- **Best for**: Single-tenant, simple deployments

### Mode 2: Operator-Ready
- Helm renders CRDs and tenant custom resources
- Separate controller reconciles HermesTenant CRs
- **Best for**: Multi-tenant platforms with a controller

**Enable operator mode**:
```yaml
operator:
  enabled: true
  controllerClass: hermes.ai/default
  tenants:
    - name: hermes-tenant-a
      namespace: tenant-a
      tenantId: tenant-a
```

---

## 9. Integration Points

### External Secrets Operator
```yaml
externalSecret:
  enabled: true
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: my-secret-store
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes/openrouter
```

### Istio Service Mesh
```yaml
virtualService:
  enabled: true
  gateways: ["istio-system/public-gateway"]
  hosts: ["hermes.example.com"]
  timeout: "3600s"
```

### Ingress Controller
```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: hermes.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - hosts:
        - hermes.example.com
      secretName: hermes-tls
```

### Network Policies
```yaml
networkPolicy:
  enabled: true
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
```

---

## 10. Useful Commands

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

# View logs
kubectl logs -n hermes deployment/hermes -f

# Port forward
kubectl port-forward -n hermes svc/hermes 8642:8642

# Uninstall
helm uninstall hermes -n hermes
```

---

## 11. Important Notes

1. **State Safety**: Hermes stores mutable state in `/opt/data`
   - Always use `replicaCount: 1`
   - Always use `strategy.type: Recreate`
   - Always use `persistence.enabled: true`

2. **Security**: Pod runs as non-root (UID 1000) with dropped capabilities

3. **Bootstrap**: Config is bootstrapped on every deploy (configurable)

4. **Multi-tenancy**: Deploy one release per tenant, not horizontal scaling

5. **Unofficial**: Community-maintained, not official Nous Research project

---

## 12. Reference Links

- **Chart Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Hermes Agent**: https://github.com/nousresearch/hermes-agent
- **OpenRouter API**: https://openrouter.ai
- **Helm Documentation**: https://helm.sh/docs/
- **Kubernetes Documentation**: https://kubernetes.io/docs/

---

## 13. Quick Troubleshooting Checklist

- [ ] Kind cluster created and running
- [ ] Namespace created: `kubectl create namespace hermes`
- [ ] OPENROUTER_API_KEY obtained from https://openrouter.ai/keys
- [ ] values.yaml created with secrets
- [ ] Helm chart cloned: `git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git`
- [ ] Helm install executed: `helm install hermes . --namespace hermes --create-namespace -f values.yaml`
- [ ] Pod is running: `kubectl get pods -n hermes`
- [ ] Logs show no errors: `kubectl logs -n hermes deployment/hermes`
- [ ] Config bootstrapped: `kubectl exec -n hermes deployment/hermes -- ls /opt/data/config.yaml`
- [ ] API responding (if enabled): `curl http://localhost:8642/health`

