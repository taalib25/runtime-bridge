# Hermes Agent Helm Chart - Integration Recommendations for SaaS

## Executive Summary

The Hermes Agent Helm chart is **production-ready for direct deployment mode** but requires careful architectural decisions for multi-tenant SaaS deployments. The chart enforces single-replica, single-writer semantics to protect mutable state in `HERMES_HOME`.

**Key Decision**: One Helm release per tenant (recommended) vs. one operator-managed tenant CR per tenant (experimental).

---

## 1. RECOMMENDED SAAS ARCHITECTURE

### Deployment Model: One Release Per Tenant

```
┌─────────────────────────────────────────────────────────────┐
│                    SaaS Platform                             │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────┐ │
│  │  Tenant A        │  │  Tenant B        │  │ Tenant N   │ │
│  │  Namespace       │  │  Namespace       │  │ Namespace  │ │
│  ├──────────────────┤  ├──────────────────┤  ├────────────┤ │
│  │ Helm Release     │  │ Helm Release     │  │ Helm       │ │
│  │ hermes-tenant-a  │  │ hermes-tenant-b  │  │ Release    │ │
│  ├──────────────────┤  ├──────────────────┤  ├────────────┤ │
│  │ Deployment (1)   │  │ Deployment (1)   │  │ Deployment │ │
│  │ PVC (10Gi)       │  │ PVC (10Gi)       │  │ PVC        │ │
│  │ Secret           │  │ Secret           │  │ Secret     │ │
│  │ Service          │  │ Service          │  │ Service    │ │
│  │ Ingress          │  │ Ingress          │  │ Ingress    │ │
│  │ NetworkPolicy    │  │ NetworkPolicy    │  │ NetPolicy  │ │
│  └──────────────────┘  └──────────────────┘  └────────────┘ │
│                                                               │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  Shared Infrastructure                                   ││
│  │  - Ingress Controller (nginx)                            ││
│  │  - Cert Manager (TLS)                                    ││
│  │  - External Secrets Operator (secret rotation)           ││
│  │  - Monitoring (Prometheus, Grafana)                      ││
│  │  - Logging (ELK, Loki)                                   ││
│  └──────────────────────────────────────────────────────────┘│
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

**Why this model**:
- ✅ Clean tenant isolation (namespace boundary)
- ✅ Independent scaling (add/remove tenants)
- ✅ Fault isolation (one tenant's issue doesn't affect others)
- ✅ Flexible resource allocation (per-tenant sizing)
- ✅ Simplified RBAC (per-tenant service accounts)
- ✅ Easy backup/restore (per-tenant PVCs)

---

## 2. TENANT PROVISIONING WORKFLOW

### Step 1: Pre-Deployment Setup

```bash
#!/bin/bash
TENANT_ID="tenant-a"
NAMESPACE="tenant-a"
STORAGE_SIZE="10Gi"
STORAGE_CLASS="fast-ssd"

# Create namespace
kubectl create namespace $NAMESPACE

# Label namespace for network policies
kubectl label namespace $NAMESPACE \
  tenant.hermes.ai/id=$TENANT_ID \
  environment=production

# Create PVC
kubectl apply -f - <<YAML
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hermes-$TENANT_ID-data
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: $STORAGE_CLASS
  resources:
    requests:
      storage: $STORAGE_SIZE
YAML

# Wait for PVC to bind
kubectl wait --for=condition=Bound pvc/hermes-$TENANT_ID-data -n $NAMESPACE --timeout=300s
```

### Step 2: Secret Management

**Option A: External Secrets Operator (Recommended)**

```bash
# Create ExternalSecret that pulls from Vault/AWS Secrets Manager
kubectl apply -f - <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: hermes-$TENANT_ID-secrets
  namespace: $NAMESPACE
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: platform-vault
    kind: ClusterSecretStore
  target:
    name: hermes-$TENANT_ID-secrets
    creationPolicy: Owner
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: tenants/$TENANT_ID/hermes
        property: OPENROUTER_API_KEY
    - secretKey: API_SERVER_KEY
      remoteRef:
        key: tenants/$TENANT_ID/hermes
        property: API_SERVER_KEY
    - secretKey: ANTHROPIC_API_KEY
      remoteRef:
        key: tenants/$TENANT_ID/hermes
        property: ANTHROPIC_API_KEY
YAML
```

**Option B: Pre-Created Secret**

```bash
kubectl create secret generic hermes-$TENANT_ID-secrets \
  --from-literal=OPENROUTER_API_KEY=$OPENROUTER_KEY \
  --from-literal=API_SERVER_KEY=$API_KEY \
  --from-literal=ANTHROPIC_API_KEY=$ANTHROPIC_KEY \
  -n $NAMESPACE
```

### Step 3: Helm Installation

```bash
# Create values file
cat > values-$TENANT_ID.yaml <<YAML
fullnameOverride: hermes-$TENANT_ID

tenant:
  id: $TENANT_ID
  labels:
    environment: production
    owner: platform

apiServer:
  enabled: true
  port: 8642
  corsOrigins: "https://app.example.com"
  modelName: "hermes-agent-$TENANT_ID"

secrets:
  existingSecret: hermes-$TENANT_ID-secrets

persistence:
  enabled: true
  existingClaim: hermes-$TENANT_ID-data
  size: 10Gi

service:
  enabled: true
  type: ClusterIP
  annotations:
    external-dns.alpha.kubernetes.io/hostname: hermes-$TENANT_ID.example.com

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
  hosts:
    - host: hermes-$TENANT_ID.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - hosts:
        - hermes-$TENANT_ID.example.com
      secretName: hermes-$TENANT_ID-tls

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
        - protocol: TCP
          port: 53

tenantIsolation:
  enabled: true
  allowSameNamespace: true
  authorizedNamespaces:
    - ingress-nginx
    - istio-system
    - monitoring
  allowDns: true
  podAntiAffinity:
    enabled: true
    type: preferredDuringSchedulingIgnoredDuringExecution
    topologyKey: kubernetes.io/hostname

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

pdb:
  enabled: true
  minAvailable: 1
YAML

# Install release
helm install hermes-$TENANT_ID ./hermes-agent-helm-chart \
  --namespace $NAMESPACE \
  -f values-$TENANT_ID.yaml
```

### Step 4: Post-Deployment Verification

```bash
# Check pod status
kubectl get pods -n $NAMESPACE

# Verify config was bootstrapped
kubectl exec -n $NAMESPACE deployment/hermes-$TENANT_ID -- \
  cat /opt/data/config.yaml

# Test API server
kubectl port-forward -n $NAMESPACE svc/hermes-$TENANT_ID 8642:8642 &
sleep 2
curl http://localhost:8642/health
kill %1
```

---

## 3. CONFIGURATION MANAGEMENT STRATEGY

### Recommended: External ConfigMap + bootstrap.overwrite=false

**Why**: Allows users to customize config without Helm overwriting changes.

```yaml
# Step 1: Create shared ConfigMap
kubectl apply -f - <<YAML
apiVersion: v1
kind: ConfigMap
metadata:
  name: hermes-shared-config
  namespace: $NAMESPACE
data:
  config.yaml: |
    model:
      default: anthropic/claude-opus-4.6
      provider: auto
      base_url: https://openrouter.ai/api/v1
    agent:
      max_turns: 90
      gateway_timeout: 1800
      tool_use_enforcement: auto
    security:
      redact_secrets: true
      tirith_enabled: true
YAML

# Step 2: Reference in Helm values
bootstrap:
  enabled: true
  overwrite: false  # Don't overwrite user changes
  existingConfigMap: hermes-shared-config
```

**Workflow**:
1. Helm bootstraps config from ConfigMap on first deployment
2. User can modify `/opt/data/config.yaml` inside pod
3. Changes persist across pod restarts (stored in PVC)
4. Helm upgrades don't overwrite user changes

---

## 4. SECRETS ROTATION STRATEGY

### Using External Secrets Operator

```yaml
externalSecret:
  enabled: true
  refreshInterval: 15m  # Rotate every 15 minutes
  secretStoreRef:
    kind: ClusterSecretStore
    name: platform-vault
  target:
    name: hermes-tenant-a-secrets
    creationPolicy: Owner
    deletionPolicy: Retain
    template:
      type: Opaque
      metadata:
        annotations:
          reloader.stakater.com/match: "true"  # Trigger pod restart on secret change
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: tenants/tenant-a/hermes
        property: OPENROUTER_API_KEY
    - secretKey: API_SERVER_KEY
      remoteRef:
        key: tenants/tenant-a/hermes
        property: API_SERVER_KEY
```

**With Reloader**:
```bash
# Install Reloader to auto-restart pods when secrets change
helm repo add stakater https://stakater.github.io/stakater-helm-charts
helm install reloader stakater/reloader \
  --namespace kube-system \
  --set reloader.watchGlobally=true
```

---

## 5. MONITORING & OBSERVABILITY

### Add ServiceMonitor for Prometheus

```yaml
# extraObjects in Helm values
extraObjects:
  - apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: hermes-agent
      namespace: {{ .Release.Namespace }}
    spec:
      selector:
        matchLabels:
          app.kubernetes.io/name: hermes-agent
      endpoints:
        - port: api-server
          interval: 30s
          path: /metrics
```

### Add Logging Integration

```yaml
# extraContainers in Helm values
extraContainers:
  - name: log-forwarder
    image: fluent/fluent-bit:latest
    volumeMounts:
      - name: data
        mountPath: /opt/data
    env:
      - name: FLUENT_UID
        value: "0"
```

---

## 6. BACKUP & DISASTER RECOVERY

### Using Velero

```bash
# Install Velero
helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts
helm install velero vmware-tanzu/velero \
  --namespace velero \
  --create-namespace \
  --set configuration.backupStorageLocation.bucket=hermes-backups \
  --set configuration.backupStorageLocation.provider=aws

# Create backup schedule
velero schedule create hermes-daily \
  --schedule="0 2 * * *" \
  --include-namespaces "tenant-*" \
  --ttl 720h
```

### Manual Backup

```bash
# Backup PVC
kubectl get pvc -n tenant-a hermes-tenant-a-data -o yaml > pvc-backup.yaml

# Backup Secret
kubectl get secret -n tenant-a hermes-tenant-a-secrets -o yaml > secret-backup.yaml

# Backup Helm release
helm get values hermes-tenant-a -n tenant-a > values-backup.yaml
```

---

## 7. SCALING STRATEGY

### Horizontal Scaling: Multiple Releases

**Problem**: Cannot scale single release horizontally (single-writer constraint).

**Solution**: Create multiple releases for load distribution.

```bash
# Deploy 3 independent Hermes instances for tenant-a
for i in 1 2 3; do
  helm install hermes-tenant-a-shard-$i ./hermes-agent-helm-chart \
    --namespace tenant-a \
    --set fullnameOverride=hermes-tenant-a-shard-$i \
    --set persistence.existingClaim=hermes-tenant-a-shard-$i-data \
    -f values-tenant-a.yaml
done

# Create Service that load-balances across shards
kubectl apply -f - <<YAML
apiVersion: v1
kind: Service
metadata:
  name: hermes-tenant-a-lb
  namespace: tenant-a
spec:
  type: ClusterIP
  selector:
    tenant.hermes.ai/id: tenant-a
  ports:
    - name: api-server
      port: 8642
      targetPort: 8642
YAML
```

### Vertical Scaling: Increase Resources

```yaml
resources:
  requests:
    cpu: 2000m
    memory: 4Gi
  limits:
    cpu: 8
    memory: 16Gi

browser:
  shm:
    enabled: true
    sizeLimit: 4Gi
```

---

## 8. MULTI-REGION DEPLOYMENT

### Separate Clusters Per Region

```
┌─────────────────────────────────────────────────────────────┐
│                    Global SaaS Platform                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────┐ │
│  │  US-East         │  │  EU-West         │  │ APAC       │ │
│  │  Cluster         │  │  Cluster         │  │ Cluster    │ │
│  ├──────────────────┤  ├──────────────────┤  ├────────────┤ │
│  │ Hermes Releases  │  │ Hermes Releases  │  │ Hermes     │ │
│  │ (per tenant)     │  │ (per tenant)     │  │ Releases   │ │
│  │                  │  │                  │  │            │ │
│  │ PVCs (regional)  │  │ PVCs (regional)  │  │ PVCs       │ │
│  │ Secrets (ESO)    │  │ Secrets (ESO)    │  │ Secrets    │ │
│  └──────────────────┘  └──────────────────┘  └────────────┘ │
│                                                               │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  Global Control Plane                                    ││
│  │  - Tenant Management API                                 ││
│  │  - DNS (Route53, CloudFlare)                             ││
│  │  - Global Load Balancer                                  ││
│  │  - Secrets Management (Vault)                            ││
│  │  - Monitoring (Prometheus Federation)                    ││
│  └──────────────────────────────────────────────────────────┘│
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

**Deployment per region**:
```bash
for region in us-east eu-west apac; do
  kubectl config use-context $region
  helm install hermes-tenant-a ./hermes-agent-helm-chart \
    --namespace tenant-a \
    --set region=$region \
    -f values-tenant-a.yaml
done
```

---

## 9. COST OPTIMIZATION

### Resource Requests/Limits

```yaml
# Development
resources:
  requests:
    cpu: 250m
    memory: 512Mi
  limits:
    cpu: 1
    memory: 2Gi

# Production
resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 4
    memory: 8Gi

# High-Performance
resources:
  requests:
    cpu: 2000m
    memory: 4Gi
  limits:
    cpu: 8
    memory: 16Gi
```

### Storage Optimization

```yaml
persistence:
  size: 5Gi      # Start small
  # Monitor usage and increase as needed
  # Alert at 80% capacity
```

### Node Affinity for Cost

```yaml
affinity:
  nodeAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
            - key: node.kubernetes.io/instance-type
              operator: In
              values:
                - t3.large    # Cheaper instance type
                - t3a.large
```

---

## 10. SECURITY HARDENING

### Network Policies

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8642
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow external API calls
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

### Pod Security Policy

```yaml
podSecurityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  fsGroupChangePolicy: OnRootMismatch
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

### RBAC

```yaml
rbac:
  create: true
  rules:
    # Only if Hermes needs K8s API access
    - apiGroups: [""]
      resources: ["configmaps", "secrets"]
      verbs: ["get", "list", "watch"]
```

---

## 11. TROUBLESHOOTING GUIDE

### Pod Won't Start

```bash
# Check events
kubectl describe pod -n tenant-a deployment/hermes-tenant-a

# Check logs
kubectl logs -n tenant-a deployment/hermes-tenant-a

# Common issues:
# - PVC not bound: kubectl get pvc -n tenant-a
# - Secret missing: kubectl get secrets -n tenant-a
# - Image pull error: kubectl describe pod -n tenant-a
```

### Config Not Applied

```bash
# Verify bootstrap ConfigMap
kubectl get configmap -n tenant-a

# Check init container logs
kubectl logs -n tenant-a deployment/hermes-tenant-a -c bootstrap-config

# Verify config in pod
kubectl exec -n tenant-a deployment/hermes-tenant-a -- cat /opt/data/config.yaml
```

### API Server Not Responding

```bash
# Port forward
kubectl port-forward -n tenant-a svc/hermes-tenant-a 8642:8642

# Test
curl http://localhost:8642/health

# Check pod logs
kubectl logs -n tenant-a deployment/hermes-tenant-a | grep -i "api"
```

### PVC Full

```bash
# Check usage
kubectl exec -n tenant-a deployment/hermes-tenant-a -- df -h /opt/data

# Increase size
kubectl patch pvc hermes-tenant-a-data -n tenant-a -p '{"spec":{"resources":{"requests":{"storage":"20Gi"}}}}'
```

---

## 12. MIGRATION PATH: Direct → Operator-Ready

**When to migrate**:
- Managing 50+ tenants
- Need dynamic tenant provisioning
- Want controller-based lifecycle management

**Migration steps**:
1. Implement custom controller that reconciles `HermesTenant` CRDs
2. Deploy controller alongside existing Helm releases
3. Gradually migrate tenants from Helm releases to CRs
4. Controller creates/manages Helm releases or native resources

---

## SUMMARY

| Aspect | Recommendation |
|--------|-----------------|
| **Deployment Model** | One Helm release per tenant |
| **Secrets** | External Secrets Operator + Vault |
| **Config** | External ConfigMap + bootstrap.overwrite=false |
| **Scaling** | Multiple releases per tenant (sharding) |
| **Backup** | Velero for automated backups |
| **Monitoring** | ServiceMonitor + Prometheus |
| **Logging** | Fluent Bit / Loki |
| **Security** | NetworkPolicy + Pod Security Context |
| **Multi-Region** | Separate clusters per region |
| **Cost** | Right-size resources per tenant tier |

