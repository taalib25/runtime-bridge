# Hermes Agent - Production Deployment Guide

**Complete guide for deploying Hermes Agent to production environments.**

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## 📋 Pre-Deployment Checklist

### Infrastructure Requirements

- [ ] Kubernetes cluster (v1.24+) or Docker host
- [ ] Persistent storage (for configuration and cache)
- [ ] Load balancer (for API server)
- [ ] Monitoring stack (Prometheus + Grafana)
- [ ] Logging infrastructure (ELK, Loki, etc.)
- [ ] Backup storage
- [ ] SSL/TLS certificates

### Configuration Requirements

- [ ] All API keys and tokens in `.env`
- [ ] User allowlists configured
- [ ] Rate limiting configured
- [ ] Webhook endpoints configured
- [ ] Model selection finalized
- [ ] Resource limits defined
- [ ] Backup strategy documented

### Security Requirements

- [ ] Security audit completed
- [ ] Penetration testing completed
- [ ] API keys rotated
- [ ] SSL/TLS certificates valid
- [ ] Firewall rules configured
- [ ] Network policies configured
- [ ] RBAC configured

---

## 🐳 Docker Deployment

### Build Production Image

```bash
# Build image
docker build -t hermes-agent:v0.10.0 .

# Tag for registry
docker tag hermes-agent:v0.10.0 registry.example.com/hermes-agent:v0.10.0

# Push to registry
docker push registry.example.com/hermes-agent:v0.10.0

# Verify image
docker inspect registry.example.com/hermes-agent:v0.10.0
```

### Docker Compose Deployment

```yaml
# docker-compose.yml
version: '3.8'

services:
  hermes:
    image: registry.example.com/hermes-agent:v0.10.0
    container_name: hermes-agent
    restart: always
    ports:
      - "8080:8080"
    environment:
      - HERMES_LOG_LEVEL=info
      - HERMES_ENABLE_METRICS=true
      - HERMES_ENABLE_TRACING=true
    env_file:
      - .env
    volumes:
      - ./config.yaml:/etc/hermes/config.yaml:ro
      - hermes-cache:/var/cache/hermes
      - hermes-logs:/var/log/hermes
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - hermes-network
    depends_on:
      - prometheus
      - grafana

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: always
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    networks:
      - hermes-network

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    restart: always
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - hermes-network

volumes:
  hermes-cache:
  hermes-logs:
  prometheus-data:
  grafana-data:

networks:
  hermes-network:
    driver: bridge
```

### Docker Deployment Commands

```bash
# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f hermes

# Stop services
docker-compose down

# Backup volumes
docker run --rm -v hermes-cache:/data -v $(pwd):/backup \
  alpine tar czf /backup/hermes-cache.tar.gz -C /data .
```

---

## ☸️ Kubernetes Deployment

### Namespace Setup

```bash
# Create namespace
kubectl create namespace hermes

# Set default namespace
kubectl config set-context --current --namespace=hermes
```

### ConfigMap and Secrets

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hermes-config
  namespace: hermes
data:
  config.yaml: |
    model:
      name: gpt-4
      provider: openai
    gateways:
      - name: telegram
        platform: telegram
        enabled: true
        token: ${TELEGRAM_BOT_TOKEN}
    api_server:
      enabled: true
      port: 8080
      host: 0.0.0.0
```

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: hermes-secrets
  namespace: hermes
type: Opaque
stringData:
  HERMES_API_KEY: "your-api-key"
  TELEGRAM_BOT_TOKEN: "your-telegram-token"
  DISCORD_BOT_TOKEN: "your-discord-token"
  OPENAI_API_KEY: "your-openai-key"
```

### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hermes-agent
  namespace: hermes
  labels:
    app: hermes-agent
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: hermes-agent
  template:
    metadata:
      labels:
        app: hermes-agent
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: hermes-agent
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: hermes
        image: registry.example.com/hermes-agent:v0.10.0
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        env:
        - name: HERMES_LOG_LEVEL
          value: "info"
        - name: HERMES_ENABLE_METRICS
          value: "true"
        - name: HERMES_ENABLE_TRACING
          value: "true"
        envFrom:
        - configMapRef:
            name: hermes-config
        - secretRef:
            name: hermes-secrets
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
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
        volumeMounts:
        - name: config
          mountPath: /etc/hermes
          readOnly: true
        - name: cache
          mountPath: /var/cache/hermes
        - name: logs
          mountPath: /var/log/hermes
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
      volumes:
      - name: config
        configMap:
          name: hermes-config
      - name: cache
        emptyDir: {}
      - name: logs
        emptyDir: {}
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
                  - hermes-agent
              topologyKey: kubernetes.io/hostname
```

### Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: hermes-agent
  namespace: hermes
  labels:
    app: hermes-agent
spec:
  type: LoadBalancer
  selector:
    app: hermes-agent
  ports:
  - name: http
    port: 80
    targetPort: http
    protocol: TCP
  - name: https
    port: 443
    targetPort: http
    protocol: TCP
```

### Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hermes-agent
  namespace: hermes
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - hermes.example.com
    secretName: hermes-tls
  rules:
  - host: hermes.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hermes-agent
            port:
              name: http
```

### Deploy to Kubernetes

```bash
# Create namespace and secrets
kubectl create namespace hermes
kubectl apply -f secret.yaml

# Deploy
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml

# Verify deployment
kubectl get pods -n hermes
kubectl get svc -n hermes
kubectl get ingress -n hermes

# Check logs
kubectl logs -n hermes -l app=hermes-agent -f

# Port forward for testing
kubectl port-forward -n hermes svc/hermes-agent 8080:80
```

---

## 🔒 Security Hardening

### Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hermes-agent
  namespace: hermes
spec:
  podSelector:
    matchLabels:
      app: hermes-agent
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
  - to:
    - podSelector:
        matchLabels:
          app: prometheus
    ports:
    - protocol: TCP
      port: 9090
```

### Pod Security Policy

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: hermes-agent
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - 'configMap'
  - 'emptyDir'
  - 'projected'
  - 'secret'
  - 'downwardAPI'
  - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'MustRunAs'
    seLinuxOptions:
      level: "s0:c123,c456"
  fsGroup:
    rule: 'MustRunAs'
    ranges:
    - min: 1000
      max: 65535
  readOnlyRootFilesystem: true
```

### RBAC

```yaml
# rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hermes-agent
  namespace: hermes

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: hermes-agent
  namespace: hermes
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hermes-agent
  namespace: hermes
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: hermes-agent
subjects:
- kind: ServiceAccount
  name: hermes-agent
  namespace: hermes
```

---

## 📊 Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'hermes'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Hermes Agent",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(hermes_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(hermes_errors_total[5m])"
          }
        ]
      },
      {
        "title": "Latency (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, hermes_request_duration_seconds)"
          }
        ]
      },
      {
        "title": "Gateway Status",
        "targets": [
          {
            "expr": "hermes_gateway_status"
          }
        ]
      }
    ]
  }
}
```

---

## 🔄 Backup & Disaster Recovery

### Backup Strategy

```bash
# Daily backup script
#!/bin/bash
BACKUP_DIR="/backups/hermes"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup configuration
tar -czf $BACKUP_DIR/config_$DATE.tar.gz ~/.hermes/

# Backup database (if applicable)
pg_dump hermes_db | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# Upload to S3
aws s3 cp $BACKUP_DIR/config_$DATE.tar.gz s3://hermes-backups/
aws s3 cp $BACKUP_DIR/db_$DATE.sql.gz s3://hermes-backups/

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "*.tar.gz" -mtime +30 -delete
```

### Disaster Recovery

```bash
# Restore from backup
#!/bin/bash
BACKUP_FILE=$1

# Stop agent
hermes stop

# Restore configuration
tar -xzf $BACKUP_FILE -C ~/

# Restore database (if applicable)
gunzip < $BACKUP_FILE | psql hermes_db

# Restart agent
hermes start

# Verify
hermes health
```

---

## 🚀 Deployment Automation

### CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build image
        run: docker build -t hermes-agent:${{ github.ref_name }} .
      
      - name: Push to registry
        run: |
          docker tag hermes-agent:${{ github.ref_name }} \
            registry.example.com/hermes-agent:${{ github.ref_name }}
          docker push registry.example.com/hermes-agent:${{ github.ref_name }}
      
      - name: Deploy to Kubernetes
        run: |
          kubectl set image deployment/hermes-agent \
            hermes=registry.example.com/hermes-agent:${{ github.ref_name }} \
            -n hermes
      
      - name: Verify deployment
        run: kubectl rollout status deployment/hermes-agent -n hermes
```

---

## ✅ Post-Deployment Verification

```bash
# Check service health
curl https://hermes.example.com/health

# Check metrics
curl https://hermes.example.com/metrics

# Test API
curl -H "Authorization: Bearer $API_KEY" \
  https://hermes.example.com/v1/models

# Check logs
kubectl logs -n hermes -l app=hermes-agent

# Monitor metrics
kubectl port-forward -n hermes svc/prometheus 9090:9090
# Visit http://localhost:9090

# Check Grafana
kubectl port-forward -n hermes svc/grafana 3000:3000
# Visit http://localhost:3000
```

---

## 📞 Support & Troubleshooting

- **Deployment Issues**: See [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)
- **Troubleshooting**: See [TROUBLESHOOTING.md](../guides/TROUBLESHOOTING.md)
- **Best Practices**: See [BEST_PRACTICES.md](../guides/BEST_PRACTICES.md)
- **GitHub Issues**: https://github.com/NousResearch/hermes-agent/issues
- **Discord Community**: https://discord.gg/NousResearch

