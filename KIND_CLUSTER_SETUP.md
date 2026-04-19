# KIND CLUSTER SETUP FOR HERMES AGENT

## Prerequisites

### Install Required Tools
```bash
# Install Kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.24.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
```

## Step 1: Create Kind Cluster

```bash
cat <<EOF | kind create cluster --name hermes-test --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
EOF
```

## Step 2: Install NGINX Ingress Controller

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.0/deploy/static/provider/kind/deploy.yaml

# Wait for ingress controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

## Step 3: Install cert-manager (for TLS certificates)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cert-manager -n cert-manager --timeout=120s
```

## Step 4: Install External Secrets Operator

```bash
# Add Helm repository
helm repo add external-secrets https://charts.external-secrets.io
helm repo update

# Install External Secrets Operator
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets \
  --create-namespace \
  --set installCRDs=true

# Wait for ESO to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=external-secrets -n external-secrets --timeout=120s
```

## Step 5: Create Production ClusterIssuer

For production, replace the staging issuer with a production one:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@hermeshq.net
    privateKeySecretRef:
      name: letsencrypt-prod-account-key
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## Step 6: Deploy Hermes Agent

### Create Namespace
```bash
kubectl create namespace hermes
```

### Create Values File
```yaml
# hermes-values.yaml
nameOverride: "hermes-tenant123"
fullnameOverride: "hermes-tenant123"

image:
  repository: nousresearch/hermes-agent
  tag: "0.8.0"  # Pin to specific version
  pullPolicy: IfNotPresent

replicaCount: 1
strategy:
  type: Recreate

# Enable API server
apiServer:
  enabled: true
  host: "0.0.0.0"
  port: 8642

# Enable service
service:
  enabled: true
  type: ClusterIP
  ports:
    - name: api-server
      port: 8642
      targetPort: 8642
      protocol: TCP

# Minimal config
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
      backend: docker
      timeout: 180
    compression:
      enabled: true
      threshold: 0.5
      target_ratio: 0.2

# Use External Secrets Operator
secrets:
  existingSecret: "hermes-tenant123-secrets"

# Enable persistence
persistence:
  enabled: true
  size: 10Gi
  storageClass: "standard"
  accessMode: ReadWriteOnce

# Bootstrap config
bootstrap:
  enabled: true
  overwrite: false  # Only seed if not exists

# Custom SOUL
soul:
  text: |
    # Hermes - Your AI Assistant
    
    You are a helpful, professional AI assistant focused on providing accurate and useful responses.
    
    ## Guidelines
    - Be concise but thorough
    - Use tools when needed (web search, file operations, etc.)
    - Ask for clarification when uncertain
    - Maintain professional tone

# Enable browser automation via external services
env:
  WEB_TOOLS_DEBUG: "false"
  GATEWAY_ALLOW_ALL_USERS: "false"
```

### Create External Secret (example)
```yaml
# external-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: hermes-tenant123-secrets
  namespace: hermes
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: hermes-tenant123-secrets
    creationPolicy: Owner
  data:
  - secretKey: OPENROUTER_API_KEY
    remoteRef:
      key: hermes/tenant123/openrouter-api-key
  - secretKey: API_SERVER_KEY
    remoteRef:
      key: hermes/tenant123/api-server-key
```

### Deploy with Helm
```bash
# Add Helm repository (if using official chart)
helm repo add hermes-agent https://ultraworkers.github.io/hermes-agent-helm-chart
helm repo update

# Deploy
helm upgrade --install hermes-tenant123 hermes-agent/hermes-agent \
  -f hermes-values.yaml \
  --namespace hermes
```

## Step 7: Create Ingress with TLS

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hermes-tenant123
  namespace: hermes
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - tenant123.hermeshq.net
    secretName: hermes-tenant123-tls
  rules:
  - host: tenant123.hermeshq.net
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hermes-tenant123
            port:
              number: 8642
```

```bash
kubectl apply -f ingress.yaml
```

## Step 8: Verify Deployment

### Check Pods
```bash
kubectl get pods -n hermes
# Should show: hermes-tenant123-xxxxx Running
```

### Check PVC
```bash
kubectl get pvc -n hermes
# Should show: hermes-tenant123-data Bound
```

### Check Ingress
```bash
kubectl get ingress -n hermes
# Should show: hermes-tenant123 with ADDRESS and HOSTS
```

### Test API
```bash
# From within the cluster
kubectl run -it --rm --restart=Never debug --image=curlimages/curl --namespace hermes -- curl http://hermes-tenant123:8642/health

# From outside (with proper DNS)
curl -H "Host: tenant123.hermeshq.net" https://your-cluster-ip/health
```

## Step 9: Configure Cloudflare DNS

### Wildcard DNS Record
- **Type**: A
- **Name**: `*.hermeshq.net`
- **Content**: Your k3s server IP (46.62.238.47)
- **Proxy status**: Proxied (orange cloud)

### SSL/TLS Settings
- **SSL/TLS**: Full (strict)
- **Minimum TLS Version**: 1.2
- **Opportunistic Encryption**: On
- **TLS 1.3**: On

## Step 10: Production Considerations

### Backup Strategy
```bash
# Install Velero for backup
velero install --provider aws --plugins velero/velero-plugin-for-aws:v1.8.0 --bucket velero-backups --secret-file ./credentials-velero --use-volume-snapshots=false --backup-location-config region=minio,s3ForcePathStyle="true",s3Url=http://minio:9000

# Create backup
velero backup create hermes-tenant123-backup --include-namespaces hermes --snapshot-volumes
```

### Monitoring
```yaml
# prometheus-service-monitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hermes-tenant123
  namespace: hermes
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: hermes-tenant123
  endpoints:
  - port: api-server
    path: /metrics
    interval: 30s
```

### Resource Limits
```yaml
resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 2000m
    memory: 4Gi
```

## Troubleshooting

### Common Issues

**Issue: Pod stuck in Pending state**
- **Cause**: Insufficient resources or PVC issues
- **Solution**: Check `kubectl describe pod` and ensure sufficient CPU/memory

**Issue: Certificate not ready**
- **Cause**: Let's Encrypt rate limits or DNS issues
- **Solution**: Check `kubectl describe certificate` and verify DNS propagation

**Issue: API server not responding**
- **Cause**: Hermes not fully initialized or configuration issues
- **Solution**: Check `kubectl logs` and verify secrets are properly mounted

**Issue: Ingress returns 404**
- **Cause**: Host header mismatch or service not found
- **Solution**: Verify ingress host matches request Host header and service exists

### Debug Commands
```bash
# Check pod logs
kubectl logs -n hermes deploy/hermes-tenant123

# Check pod events
kubectl describe pod -n hermes -l app.kubernetes.io/name=hermes-tenant123

# Check ingress status
kubectl describe ingress -n hermes hermes-tenant123

# Check certificate status
kubectl describe certificate -n hermes hermes-tenant123-tls

# Exec into pod
kubectl exec -it -n hermes deploy/hermes-tenant123 -- sh

# Check persistent data
kubectl exec -n hermes deploy/hermes-tenant123 -- ls -la /opt/data/
```

## Scaling Considerations

### Single Tenant per Instance
- Each tenant gets their own Helm release
- Isolated persistent storage per tenant
- Independent scaling and lifecycle management

### Multiple Tenants on Same Cluster
- Use separate namespaces per tenant (optional but recommended)
- Shared ingress controller and cert-manager
- Independent resource quotas per tenant

### Multi-Cluster Setup
- Deploy identical clusters in different regions
- Use DNS-based routing for geo-distribution
- Implement cross-cluster backup and failover

## Security Best Practices

### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hermes-tenant123-deny-all
  namespace: hermes
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hermes-tenant123
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
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - 10.0.0.0/8
        - 172.16.0.0/12
        - 192.168.0.0/16
```

### Pod Security
- Run as non-root user (UID 1000)
- Drop all capabilities except necessary ones
- Use read-only root filesystem
- Set proper fsGroup for persistent volumes

### Secret Management
- Never store secrets in Helm values directly
- Use External Secrets Operator with secure backends (Vault, AWS Secrets Manager)
- Rotate secrets regularly
- Limit secret access to only necessary pods

## Upgrade Strategy

### Blue/Green Deployment
1. Deploy new version with different name (`hermes-tenant123-v2`)
2. Update ingress to point to new version
3. Verify functionality
4. Delete old version

### Rolling Updates (not recommended for Hermes)
- Hermes is stateful and single-writer
- Rolling updates can cause data corruption
- Use recreate strategy instead

### Configuration Updates
- Update Helm values and run `helm upgrade`
- Changes to config.yaml require pod restart
- Secret updates are automatically picked up by ESO

---

This setup provides a production-ready foundation for running Hermes Agent in a Kubernetes cluster with proper security, monitoring, and operational practices.