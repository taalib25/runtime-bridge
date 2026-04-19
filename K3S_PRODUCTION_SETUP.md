# K3S PRODUCTION SETUP FOR HERMES AGENT

## Key Difference: Native Traefik

**k3s comes with Traefik ingress controller built-in** - no need to install NGINX or any other ingress controller. This is much simpler and more efficient.

## Prerequisites

### Install Required Tools on Your Server
```bash
# Install Helm (k3s already has kubectl)
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify k3s is running with Traefik
kubectl get pods -A | grep traefik
# Should show: traefik-xxxxx in kube-system namespace
```

## Step 1: Verify Traefik is Running

```bash
# Check Traefik deployment
kubectl get deploy -n kube-system traefik

# Check Traefik service
kubectl get svc -n kube-system traefik

# Check Traefik config
kubectl get cm -n kube-system traefik
```

**Expected output:**
```
NAME      READY   UP-TO-DATE   AVAILABLE   AGE
traefik   1/1     1            1           24h

NAME      TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
traefik   LoadBalancer   10.43.123.45    <pending>     80:32123/TCP,443:32456/TCP   24h
```

## Step 2: Install cert-manager (for TLS certificates)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cert-manager -n cert-manager --timeout=120s
```

## Step 3: Install External Secrets Operator

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

## Step 4: Create Production ClusterIssuer

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
          class: traefik  # Note: traefik instead of nginx
EOF
```

## Step 5: Deploy Hermes Agent

### Create Namespace
```bash
kubectl create namespace hermes
```

### Create Values File (same as before)
```yaml
# hermes-values.yaml
nameOverride: "hermes-tenant123"
fullnameOverride: "hermes-tenant123"

image:
  repository: nousresearch/hermes-agent
  tag: "0.8.0"
  pullPolicy: IfNotPresent

replicaCount: 1
strategy:
  type: Recreate

apiServer:
  enabled: true
  host: "0.0.0.0"
  port: 8642

service:
  enabled: true
  type: ClusterIP
  ports:
    - name: api-server
      port: 8642
      targetPort: 8642
      protocol: TCP

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

secrets:
  existingSecret: "hermes-tenant123-secrets"

persistence:
  enabled: true
  size: 10Gi
  storageClass: "local-path"  # k3s default storage class
  accessMode: ReadWriteOnce

bootstrap:
  enabled: true
  overwrite: false

soul:
  text: |
    # Hermes - Your AI Assistant
    You are a helpful, professional AI assistant focused on providing accurate and useful responses.
```

### Deploy with Helm
```bash
helm repo add hermes-agent https://ultraworkers.github.io/hermes-agent-helm-chart
helm repo update

helm upgrade --install hermes-tenant123 hermes-agent/hermes-agent \
  -f hermes-values.yaml \
  --namespace hermes
```

## Step 6: Create Ingress with Traefik and TLS

**Key difference: Use `traefik` ingress class instead of `nginx`**

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hermes-tenant123
  namespace: hermes
  annotations:
    # Traefik-specific annotations (different from NGINX)
    traefik.ingress.kubernetes.io/router.entrypoints: web,websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: traefik  # Important: traefik instead of nginx
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

## Step 7: Configure Cloudflare DNS

Same as before:
- **Type**: A
- **Name**: `*.hermeshq.net`
- **Content**: Your k3s server IP (46.62.238.47)
- **Proxy status**: Proxied

## Step 8: Verify Deployment

```bash
# Check pods
kubectl get pods -n hermes

# Check ingress
kubectl get ingress -n hermes

# Test from within cluster
kubectl run -it --rm --restart=Never debug --image=curlimages/curl --namespace hermes -- curl http://hermes-tenant123:8642/health
```

## Traefik vs NGINX Differences

| Feature | Traefik (k3s native) | NGINX (Kind manual) |
|---------|---------------------|-------------------|
| **Installation** | Built-in, zero setup | Manual installation required |
| **Resource Usage** | Lighter weight | Heavier |
| **Configuration** | Kubernetes CRDs + Ingress | Ingress + annotations |
| **TLS** | Automatic Let's Encrypt | Requires cert-manager |
| **Performance** | Good for most workloads | Excellent for high traffic |
| **Complexity** | Simpler for basic needs | More features but complex |

## Why Traefik is Better for Your Use Case

1. **Zero Installation**: Already running in your k3s cluster
2. **Lower Overhead**: Less resource consumption
3. **Simpler Configuration**: Fewer moving parts
4. **Perfect for Hermes**: Your workload doesn't need NGINX's advanced features
5. **Faster Deployment**: No need to wait for NGINX installation

## Migration from Kind to k3s

When moving from your Kind development environment to k3s production:

1. **Keep the same Helm values** (only change storageClass to `local-path`)
2. **Change ingressClassName** from `nginx` to `traefik`
3. **Update annotations** from NGINX format to Traefik format
4. **Use k3s default storage class** (`local-path` instead of `standard`)

## Troubleshooting Traefik

**Issue: Ingress not working**
- **Check**: `kubectl describe ingress -n hermes hermes-tenant123`
- **Verify**: IngressClassName is `traefik`
- **Check**: Traefik logs: `kubectl logs -n kube-system -l app.kubernetes.io/name=traefik`

**Issue: Certificate not issued**
- **Check**: `kubectl describe certificate -n hermes hermes-tenant123-tls`
- **Verify**: ClusterIssuer name matches
- **Check**: DNS propagation for your domain

**Issue: Service not reachable**
- **Check**: `kubectl get svc -n hermes`
- **Verify**: Service name matches ingress backend
- **Test**: `kubectl port-forward svc/hermes-tenant123 8642:8642`

---

**Summary**: Use native Traefik in k3s - it's simpler, lighter, and already configured. Only use NGINX if you specifically need its advanced features (which Hermes doesn't require).