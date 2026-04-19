# Hermes Agent Helm Chart - Quick Reference

## Installation

### Minimal (Gateway Only)
```bash
helm install hermes ./hermes-agent-helm-chart \
  --namespace hermes \
  --create-namespace \
  --set secrets.OPENROUTER_API_KEY=sk-or-...
```

### With API Server
```bash
helm install hermes ./hermes-agent-helm-chart \
  --namespace hermes \
  --create-namespace \
  --set apiServer.enabled=true \
  --set secrets.API_SERVER_KEY=your-key \
  --set secrets.OPENROUTER_API_KEY=sk-or-...
```

### Multi-Tenant (Per-Tenant Release)
```bash
# Create namespace
kubectl create namespace tenant-a

# Create PVC
kubectl apply -f - <<YAML
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hermes-tenant-a-data
  namespace: tenant-a
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
YAML

# Create Secret
kubectl create secret generic hermes-tenant-a-secrets \
  --from-literal=OPENROUTER_API_KEY=sk-or-... \
  --from-literal=API_SERVER_KEY=... \
  -n tenant-a

# Install release
helm install hermes-tenant-a ./hermes-agent-helm-chart \
  --namespace tenant-a \
  -f values-tenant-a.yaml
```

## Configuration Patterns

### Pattern 1: Chart-Managed Secrets (Simple)
```yaml
secrets:
  OPENROUTER_API_KEY: sk-or-...
  API_SERVER_KEY: your-key
```

### Pattern 2: External Secret (Vault/AWS Secrets Manager)
```yaml
externalSecret:
  enabled: true
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: platform-secrets
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes/prod
        property: OPENROUTER_API_KEY
```

### Pattern 3: Pre-Existing Secret
```yaml
secrets:
  existingSecret: hermes-secrets
```

### Pattern 4: External ConfigMap (Config Management)
```yaml
bootstrap:
  enabled: true
  overwrite: false  # Don't overwrite user changes
  existingConfigMap: hermes-shared-config
```

## Common Values

### Enable API Server
```yaml
apiServer:
  enabled: true
  host: 0.0.0.0
  port: 8642
  corsOrigins: https://app.example.com
  modelName: hermes-agent
```

### Enable Webhooks
```yaml
webhook:
  enabled: true
  port: 8644

telegramWebhook:
  enabled: true
  url: https://hermes.example.com/telegram
  port: 8443
```

### Configure Persistence
```yaml
persistence:
  enabled: true
  existingClaim: hermes-data  # Use pre-provisioned PVC
  mountPath: /opt/data
  size: 10Gi
  storageClass: fast-ssd
```

### Configure Service & Ingress
```yaml
service:
  enabled: true
  type: ClusterIP

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
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

### Configure Istio
```yaml
service:
  enabled: true

virtualService:
  enabled: true
  gateways:
    - istio-system/public-gateway
  hosts:
    - hermes.example.com
  timeout: 3600s
```

### Configure Network Isolation
```yaml
networkPolicy:
  enabled: true
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8642
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - protocol: UDP
          port: 53
```

### Configure Tenant Isolation
```yaml
tenant:
  id: tenant-a
  labels:
    environment: prod

tenantIsolation:
  enabled: true
  allowSameNamespace: true
  authorizedNamespaces:
    - ingress-nginx
    - istio-system
  allowDns: true
  podAntiAffinity:
    enabled: true
    type: preferredDuringSchedulingIgnoredDuringExecution
    topologyKey: kubernetes.io/hostname
```

### Configure Resources
```yaml
resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 4
    memory: 8Gi

browser:
  shm:
    enabled: true
    sizeLimit: 2Gi
```

### Install NPM Packages
```yaml
npmPackages:
  - lodash@4.17.21
  - axios@1.6.0
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -n hermes
kubectl describe pod -n hermes deployment/hermes-agent
```

### View Logs
```bash
kubectl logs -n hermes deployment/hermes-agent -f
```

### Verify Config
```bash
kubectl exec -n hermes deployment/hermes-agent -- cat /opt/data/config.yaml
```

### Check PVC
```bash
kubectl get pvc -n hermes
kubectl describe pvc -n hermes hermes-agent-data
```

### Test API Server
```bash
kubectl port-forward -n hermes svc/hermes-agent 8642:8642
curl http://localhost:8642/health
```

### Verify Secrets
```bash
kubectl get secrets -n hermes
kubectl describe secret -n hermes hermes-agent-secrets
```

## Upgrade

### Update Values
```bash
helm upgrade hermes ./hermes-agent-helm-chart \
  --namespace hermes \
  -f values.yaml
```

### Update Chart Version
```bash
helm repo add hermes https://...
helm repo update
helm upgrade hermes hermes/hermes-agent \
  --namespace hermes \
  -f values.yaml
```

## Uninstall

```bash
helm uninstall hermes --namespace hermes
```

## Key Constraints

| Constraint | Reason |
|-----------|--------|
| `replicaCount: 1` | HERMES_HOME contains mutable state |
| `strategy: Recreate` | Prevents concurrent PVC access |
| `accessMode: ReadWriteOnce` | Single-writer guarantee |
| One release per tenant | Clean isolation |

## Important Notes

- ⚠️ **Unofficial chart** - maintained independently from Nous Research
- ✅ **Production-ready** for direct deployment mode
- ⚠️ **Operator mode is experimental** - requires external controller
- 🔴 **No horizontal scaling** within a single release
- 🔴 **No built-in backup** - use Velero or cloud-native solutions
- 🔴 **No monitoring** - add ServiceMonitor separately

## Resources

- **Chart Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Hermes Agent**: https://github.com/nousresearch/hermes-agent
- **Helm Docs**: https://helm.sh/docs/
- **Kubernetes Docs**: https://kubernetes.io/docs/

