# Hermes Agent Deployment Guide

Complete guide for deploying Hermes Agent in Kubernetes environments.

## Deployment Options

### Option 1: Helm Chart (Recommended)

**Best for**: Production Kubernetes deployments, multi-environment setups

#### Prerequisites
- Kubernetes 1.20+
- Helm 3.0+
- kubectl configured

#### Step 1: Clone Chart Repository

```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
```

#### Step 2: Create Values File

```bash
cat > values-prod.yaml << 'YAML'
# Production Hermes Deployment

replicaCount: 2

image:
  repository: nousresearch/hermes-agent
  tag: latest
  pullPolicy: IfNotPresent

# Secrets (use sealed-secrets or external-secrets in production)
secrets:
  OPENROUTER_API_KEY: "sk-or-YOUR_KEY"
  API_SERVER_KEY: "prod-key-change-me"

# Configuration
config:
  values:
    model:
      default: anthropic/claude-opus-4.6
      base_url: https://openrouter.ai/api/v1
      temperature: 0.5
    
    terminal:
      backend: docker
    
    memory:
      enabled: true
      type: hybrid
      compression:
        enabled: true

# API Server
apiServer:
  enabled: true
  port: 8642
  cors:
    enabled: true
    allowedOrigins:
      - "https://app.example.com"

# Service
service:
  enabled: true
  type: ClusterIP
  port: 8642
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8642"

# Ingress
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
    - secretName: hermes-tls
      hosts:
        - hermes.example.com

# Persistence
persistence:
  enabled: true
  size: 10Gi
  storageClassName: standard

# Resources
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "4Gi"
    cpu: "2000m"

# Autoscaling
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 70

# Health checks
livenessProbe:
  httpGet:
    path: /health
    port: 8642
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8642
  initialDelaySeconds: 10
  periodSeconds: 5

# Security
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsReadOnlyRootFilesystem: true
  allowPrivilegeEscalation: false

# Pod Disruption Budget
podDisruptionBudget:
  enabled: true
  minAvailable: 1

# Network Policy
networkPolicy:
  enabled: true
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            name: ingress-nginx
YAML
```

#### Step 3: Deploy

```bash
# Create namespace
kubectl create namespace hermes

# Deploy
helm install hermes . \
  --namespace hermes \
  -f values-prod.yaml

# Verify
kubectl rollout status deployment/hermes -n hermes
kubectl get pods -n hermes
```

#### Step 4: Verify Deployment

```bash
# Check pod status
kubectl get pods -n hermes -o wide

# View logs
kubectl logs -n hermes deployment/hermes -f

# Check events
kubectl describe deployment hermes -n hermes

# Test API
kubectl port-forward -n hermes svc/hermes 8642:8642 &
curl http://localhost:8642/health
```

### Option 2: Kustomize

**Best for**: GitOps workflows, environment-specific configurations

#### Step 1: Create Kustomize Structure

```bash
mkdir -p hermes-deployment/{base,overlays/{dev,prod}}

# Base manifests
cat > hermes-deployment/base/kustomization.yaml << 'YAML'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: hermes

resources:
  - deployment.yaml
  - service.yaml
  - configmap.yaml

commonLabels:
  app: hermes
  managed-by: kustomize
YAML
```

#### Step 2: Create Base Manifests

```bash
# deployment.yaml, service.yaml, configmap.yaml
# See examples below
```

#### Step 3: Create Overlays

```bash
# overlays/prod/kustomization.yaml
cat > hermes-deployment/overlays/prod/kustomization.yaml << 'YAML'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../../base

replicas:
  - name: hermes
    count: 3

resources:
  - ingress.yaml
  - pdb.yaml

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: hermes-config
    files:
      - config.yaml
    behavior: merge
YAML
```

#### Step 4: Deploy

```bash
kubectl apply -k hermes-deployment/overlays/prod/
```

### Option 3: Manual YAML

**Best for**: Simple deployments, learning, custom requirements

#### Step 1: Create Namespace

```bash
kubectl create namespace hermes
```

#### Step 2: Create ConfigMap

```bash
kubectl create configmap hermes-config \
  --from-file=config.yaml \
  -n hermes
```

#### Step 3: Create Secret

```bash
kubectl create secret generic hermes-secrets \
  --from-literal=OPENROUTER_API_KEY=sk-or-YOUR_KEY \
  --from-literal=API_SERVER_KEY=test-key \
  -n hermes
```

#### Step 4: Create Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hermes
  namespace: hermes
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hermes
  template:
    metadata:
      labels:
        app: hermes
    spec:
      containers:
      - name: hermes
        image: nousresearch/hermes-agent:latest
        ports:
        - containerPort: 8642
        env:
        - name: OPENROUTER_API_KEY
          valueFrom:
            secretKeyRef:
              name: hermes-secrets
              key: OPENROUTER_API_KEY
        - name: API_SERVER_KEY
          valueFrom:
            secretKeyRef:
              name: hermes-secrets
              key: API_SERVER_KEY
        volumeMounts:
        - name: config
          mountPath: /root/.hermes
        - name: data
          mountPath: /data
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8642
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8642
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: hermes-config
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: hermes
  namespace: hermes
spec:
  selector:
    app: hermes
  ports:
  - protocol: TCP
    port: 8642
    targetPort: 8642
  type: ClusterIP
```

#### Step 5: Deploy

```bash
kubectl apply -f deployment.yaml
```

## Deployment Patterns

### High Availability

```yaml
# Multiple replicas with pod disruption budget
replicaCount: 3

podDisruptionBudget:
  minAvailable: 1

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - hermes
        topologyKey: kubernetes.io/hostname
```

### Multi-Environment

```bash
# Development
helm install hermes . -f values-dev.yaml -n hermes-dev

# Staging
helm install hermes . -f values-staging.yaml -n hermes-staging

# Production
helm install hermes . -f values-prod.yaml -n hermes-prod
```

### GitOps (ArgoCD)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hermes
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/ultraworkers/hermes-agent-helm-chart
    targetRevision: main
    path: .
    helm:
      values: |
        replicaCount: 2
  destination:
    server: https://kubernetes.default.svc
    namespace: hermes
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Monitoring & Observability

### Prometheus Metrics

```yaml
# ServiceMonitor for Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hermes
  namespace: hermes
spec:
  selector:
    matchLabels:
      app: hermes
  endpoints:
  - port: metrics
    interval: 30s
```

### Logging

```bash
# View logs
kubectl logs -n hermes deployment/hermes -f

# Stream logs from all pods
kubectl logs -n hermes -l app=hermes -f

# Export logs
kubectl logs -n hermes deployment/hermes > hermes.log
```

## Troubleshooting

### Pod Won't Start

```bash
# Check pod status
kubectl describe pod -n hermes <pod-name>

# Check logs
kubectl logs -n hermes <pod-name>

# Check events
kubectl get events -n hermes --sort-by='.lastTimestamp'
```

### API Not Responding

```bash
# Check service
kubectl get svc -n hermes

# Port forward and test
kubectl port-forward -n hermes svc/hermes 8642:8642
curl http://localhost:8642/health

# Check pod logs
kubectl logs -n hermes deployment/hermes -f
```

### Out of Memory

```bash
# Check resource usage
kubectl top pods -n hermes

# Increase limits in values.yaml
resources:
  limits:
    memory: "4Gi"

# Redeploy
helm upgrade hermes . -f values.yaml -n hermes
```

## Cleanup

```bash
# Delete deployment
helm uninstall hermes -n hermes

# Delete namespace
kubectl delete namespace hermes

# Delete persistent volumes
kubectl delete pvc -n hermes --all
```

## Next Steps

- **[Production Deployment](./PRODUCTION_DEPLOYMENT.md)** — Security hardening
- **[Quick Start](../guides/QUICK_START.md)** — Get started quickly
- **[Troubleshooting](../guides/TROUBLESHOOTING.md)** — Common issues

---

**Last Updated**: April 18, 2026
