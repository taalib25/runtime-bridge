# Hermes Agent Helm Chart - Technical Analysis

**Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
**Commit**: e3b685d4d0288668a37216435742cd0a659ebc6c
**Chart Version**: 0.1.0 | **App Version**: 0.8.0

---

## Chart Metadata

```yaml
# Chart.yaml
apiVersion: v2
name: hermes-agent
description: Cloud-native Helm chart for Nous Research's Hermes Agent
type: application
version: 0.1.0
appVersion: "0.8.0"
kubeVersion: ">=1.25.0-0"
```

---

## Template Structure

### 1. **deployment.yaml** (Main Workload)
**Source**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/deployment.yaml

**Key Features**:
- Single replica enforcement when persistence enabled
- Recreate strategy for state safety
- Bootstrap init container for config/SOUL seeding
- NPM package installation support
- Security context: non-root (UID 1000), dropped capabilities
- Probes: liveness, readiness, startup (customizable)
- Volume mounts: persistence, bootstrap, browser shm

**Validation Rules** (lines 19-39):
```
- replicaCount > 1 + persistence.enabled = FAIL
- externalSecret.enabled + secrets.existingSecret = FAIL
- externalSecret.enabled + inline secrets = FAIL
- apiServer.enabled + no API_SERVER_KEY = FAIL
- telegramWebhook.enabled + no url = FAIL
- telegramWebhook.enabled + no TELEGRAM_BOT_TOKEN = FAIL
- tenantIsolation.enabled + no tenant.id = FAIL
```

### 2. **service.yaml** (Network Exposure)
**Conditional**: Only rendered if `service.enabled: true`

**Auto-Port Derivation**:
- If `service.ports` is empty, chart auto-derives from:
  - `apiServer.enabled` → port 8642
  - `webhook.enabled` → port 8644
  - `telegramWebhook.enabled` → port 8443

**Service Types Supported**:
- ClusterIP (default for Kind)
- LoadBalancer
- NodePort

### 3. **configmap.yaml** (Bootstrap Config)
**Conditional**: Only if `bootstrap.enabled: true` AND `bootstrap.existingConfigMap` is empty

**Contents**:
- `config.yaml`: Rendered from `config.values` or `config.raw`
- `SOUL.md`: Optional personality file from `soul.text`

**Templating**:
- `config.raw` takes precedence over `config.values`
- Both support Helm templating (tpl function)

### 4. **secret.yaml** (Credentials)
**Conditional**: Only if `secrets.existingSecret` is empty AND `externalSecret.enabled: false`

**Secrets Included**:
- OPENROUTER_API_KEY
- OPENAI_API_KEY
- ANTHROPIC_API_KEY
- GOOGLE_API_KEY
- GEMINI_API_KEY
- EXA_API_KEY
- FIRECRAWL_API_KEY
- TELEGRAM_BOT_TOKEN
- DISCORD_BOT_TOKEN
- SLACK_BOT_TOKEN
- GITHUB_TOKEN
- ... (20+ total)

### 5. **pvc.yaml** (Persistent Storage)
**Conditional**: Only if `persistence.enabled: true`

**Configuration**:
```yaml
accessMode: ReadWriteOnce
size: 5Gi (default)
storageClass: "" (uses default)
mountPath: /opt/data
```

**Binding**:
- Can use `persistence.existingClaim` to bind pre-provisioned PVC
- Annotations supported for cloud provider hints

### 6. **ingress.yaml** (HTTP Routing)
**Conditional**: Only if `ingress.enabled: true`

**Requirements**:
- `service.enabled: true` (prerequisite)
- `ingress.className` (e.g., "nginx")
- At least one host/path

**Features**:
- TLS support
- Multiple hosts
- Path-based routing
- Annotations for cert-manager, etc.

### 7. **virtualservice.yaml** (Istio)
**Conditional**: Only if `virtualService.enabled: true`

**Requirements**:
- At least one gateway
- At least one host
- `service.enabled: true` (prerequisite)

**Configuration**:
```yaml
gateways: ["istio-system/public-gateway"]
hosts: ["hermes.example.com"]
timeout: "3600s"
servicePortNumber: 8642
```

### 8. **networkpolicy.yaml** (Network Isolation)
**Conditional**: Only if `networkPolicy.enabled: true`

**Supports**:
- Ingress rules (from namespaces, pods)
- Egress rules (to namespaces, pods)
- Port-level granularity
- Policy types: Ingress, Egress

### 9. **rbac.yaml** (Kubernetes API Access)
**Conditional**: Only if `rbac.create: true`

**Components**:
- ServiceAccount
- ClusterRole or Role (based on `rbac.clusterWide`)
- ClusterRoleBinding or RoleBinding

**Default Rules**: Empty (user-provided via `rbac.rules`)

### 10. **external-secret.yaml** (ESO Integration)
**Conditional**: Only if `externalSecret.enabled: true`

**Supports**:
- ClusterSecretStore or SecretStore
- Multiple data sources
- Template rendering
- Refresh intervals

### 11. **pdb.yaml** (Pod Disruption Budget)
**Conditional**: Only if `pdb.enabled: true`

**Configuration**:
```yaml
minAvailable: 1 (default)
```

### 12. **hermes-tenants.yaml** (Operator Mode)
**Conditional**: Only if `operator.enabled: true`

**Renders**:
- HermesTenant CRDs
- Tenant custom resources from `operator.tenants`

---

## Default Values Analysis

### Image Configuration
```yaml
image:
  repository: nousresearch/hermes-agent
  tag: "0.8.0"
  pullPolicy: IfNotPresent
```

### Replica & Strategy
```yaml
replicaCount: 1                    # Must stay 1 with persistence
strategy:
  type: Recreate                   # For state safety
```

### Bootstrap
```yaml
bootstrap:
  enabled: true
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
      tool_use_enforcement: auto
    terminal:
      backend: local
      cwd: .
      timeout: 180
    display:
      tool_progress: all
      interim_assistant_messages: true
    browser:
      inactivity_timeout: 120
      command_timeout: 30
      allow_private_urls: false
    compression:
      enabled: true
      threshold: 0.5
      target_ratio: 0.2
      protect_last_n: 20
    security:
      redact_secrets: true
      tirith_enabled: true
      tirith_fail_open: true
```

### Secrets
```yaml
secrets:
  existingSecret: ""               # Use pre-created Secret
  OPENROUTER_API_KEY: ""
  OPENAI_API_KEY: ""
  ANTHROPIC_API_KEY: ""
  # ... 20+ more
```

### API Server
```yaml
apiServer:
  enabled: false
  host: "0.0.0.0"
  port: 8642
  corsOrigins: ""
  modelName: hermes-agent
```

### Webhooks
```yaml
webhook:
  enabled: false
  port: 8644

telegramWebhook:
  enabled: false
  url: ""
  port: 8443
```

### Service Account
```yaml
serviceAccount:
  create: true
  name: ""
  annotations: {}
  automountServiceAccountToken: false  # Secure default
```

### Persistence
```yaml
persistence:
  enabled: true
  existingClaim: ""
  mountPath: /opt/data
  accessMode: ReadWriteOnce
  size: 5Gi
  storageClass: ""
  annotations: {}
```

### Browser Support
```yaml
browser:
  shm:
    enabled: true
    mountPath: /dev/shm
    sizeLimit: 1Gi
```

### Security Context
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

### Resources
```yaml
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: "2"
    memory: 4Gi
```

---

## Deployment Modes

### Mode 1: Direct Deployment (Default)
- Helm manages the full workload lifecycle
- Renders Deployment, Service, Ingress, etc.
- Best for: Single-tenant, simple deployments

### Mode 2: Operator-Ready
- Helm renders CRDs and tenant custom resources
- Separate controller reconciles HermesTenant CRs
- Best for: Multi-tenant platforms with a controller

**Enable with**:
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

## Validation Schema

**File**: `values.schema.json`

**Validates**:
1. Persistence safety (replicaCount=1 when enabled)
2. Service prerequisites (service.enabled required for ingress)
3. Telegram webhook requirements (url required)
4. Istio VirtualService requirements (gateways + hosts)
5. Secret management conflicts (existingSecret vs inline)

---

## Test Configurations

**Location**: `ci/` directory

| File | Purpose |
|------|---------|
| test-values.yaml | Full feature test (API, webhooks, RBAC, etc.) |
| existing-claim-values.yaml | Pre-provisioned PVC binding |
| external-bootstrap-values.yaml | External ConfigMap bootstrap |
| default-service-ports-values.yaml | Auto port derivation |
| external-secret-values.yaml | ESO integration |
| tenant-isolation-values.yaml | Multi-tenant isolation |
| operator-values.yaml | Operator mode |

---

## Verification Script

**Location**: `ci/verify.sh`

**Checks**:
- Helm lint (syntax validation)
- Helm template (manifest generation)
- Schema validation
- Regression tests

**Run with**:
```bash
bash ci/verify.sh
```

---

## Key Design Decisions

1. **State Safety First**: Enforces single replica + Recreate strategy
2. **Composability**: Supports external ConfigMaps, Secrets, ESO
3. **Security by Default**: Non-root, dropped capabilities, read-only filesystem
4. **Flexibility**: Supports gateway, API server, webhooks, Istio, NetworkPolicy
5. **Tenant-Scoped**: One release per tenant, not horizontal scaling
6. **Bootstrap Automation**: Config seeded on every deploy (configurable)

---

## Integration Points

### External Secrets Operator
```yaml
externalSecret:
  enabled: true
  secretStoreRef:
    kind: ClusterSecretStore
    name: my-store
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes/openrouter
```

### Service Mesh (Istio)
```yaml
virtualService:
  enabled: true
  gateways: ["istio-system/public-gateway"]
  hosts: ["hermes.example.com"]
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
```

---

## Limitations & Constraints

1. **Single Replica**: Cannot scale horizontally with persistence enabled
2. **Recreate Strategy**: Pod restarts on every deployment (no rolling updates)
3. **No Multi-Tenancy**: One pod per tenant boundary
4. **Unofficial**: Community-maintained, not official Nous Research project
5. **No Built-in Controller**: Operator mode requires external controller

---

## References

- **Chart Repository**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Hermes Agent**: https://github.com/nousresearch/hermes-agent
- **Kubebuilder Book**: https://book.kubebuilder.io
- **Helm Documentation**: https://helm.sh/docs/
