# Hermes Agent Helm Chart - Comprehensive Analysis

**Chart Version**: 0.1.0 | **App Version**: 0.8.0 | **Min K8s**: 1.25.0+  
**Repository**: [ultraworkers/hermes-agent-helm-chart](https://github.com/ultraworkers/hermes-agent-helm-chart)  
**Status**: ⚠️ **Unofficial community chart** (maintained independently from upstream Nous Research)

---

## 1. CHART STRUCTURE & ORGANIZATION

### Directory Layout
```
hermes-agent-helm-chart/
├── Chart.yaml                 # Chart metadata (v0.1.0, app v0.8.0)
├── values.yaml               # 354 lines of default configuration
├── values.schema.json        # JSON Schema validation (875 lines)
├── templates/
│   ├── deployment.yaml       # Main workload (338 lines, complex)
│   ├── configmap.yaml        # config.yaml + SOUL.md bootstrap
│   ├── secret.yaml           # Inline secrets (optional)
│   ├── external-secret.yaml  # External Secrets Operator integration
│   ├── pvc.yaml              # PersistentVolumeClaim
│   ├── service.yaml          # Service (auto-derives ports)
│   ├── ingress.yaml          # Kubernetes Ingress
│   ├── virtualservice.yaml   # Istio VirtualService
│   ├── networkpolicy.yaml    # Network isolation
│   ├── rbac.yaml             # ServiceAccount + RBAC
│   ├── serviceaccount.yaml   # ServiceAccount
│   ├── pdb.yaml              # PodDisruptionBudget
│   ├── hermes-tenants.yaml   # Operator-mode CRs
│   ├── extra-objects.yaml    # Arbitrary extra resources
│   └── _helpers.tpl          # 111 lines of template helpers
├── crds/                     # CRD definitions (operator mode)
├── ci/                       # Test values for regression testing
│   ├── test-values.yaml
│   ├── tenant-isolation-values.yaml
│   ├── external-secret-values.yaml
│   └── ... (7 more test scenarios)
└── docs/                     # Documentation + screenshots
```

### Key Design Principle
**One release per tenant** in direct mode, or **one tenant CR per tenant** in operator-ready mode.

---

## 2. CORE ASSUMPTIONS & STATE SAFETY

### Single-Writer Guarantee
The chart enforces **strict single-replica deployment** when persistence is enabled:

```yaml
# From deployment.yaml (lines 19-20)
{{- if and (gt $replicas 1) .Values.persistence.enabled }}
{{- fail "Hermes Agent supports only one replica when persistence.enabled=true because HERMES_HOME contains mutable shared state" }}
{{- end }}
```

**Why**: Hermes stores mutable state under `HERMES_HOME` (default: `/opt/data`). Multiple replicas would corrupt this state.

**Implications for SaaS**:
- ✅ **One release per tenant** = safe multi-tenancy
- ❌ **Cannot scale horizontally** within a single release
- ✅ **Can scale by creating multiple releases** (one per tenant)

### Deployment Strategy
```yaml
strategy:
  type: Recreate  # Enforced when persistence.enabled=true
```

**Why**: Ensures old pod is fully terminated before new one starts, preventing concurrent access to PVC.

---

## 3. CONFIGURATION LAYERS

### Layer 1: `config.values` (Structured YAML)
Merged into `config.yaml` via Helm templating:

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

**Rendered into**: ConfigMap `hermes-agent-config` → mounted at `/bootstrap/config.yaml`

### Layer 2: `config.raw` (Raw Templated YAML)
Takes precedence over `config.values`:

```yaml
config:
  raw: |
    model:
      default: {{ .Values.customModel }}
    # ... raw YAML with Helm templating
```

**Use case**: When you need full control over config structure.

### Layer 3: Environment Variables
Direct pod env vars + `extraEnv`:

```yaml
env:
  WEB_TOOLS_DEBUG: "false"
  VISION_TOOLS_DEBUG: "false"
  GATEWAY_ALLOW_ALL_USERS: "false"
  WHATSAPP_ENABLED: "false"
  # ... 15+ more

extraEnv: []  # User-provided additional env vars
extraEnvFrom: []  # ConfigMap/Secret references
```

### Layer 4: Secrets
Three patterns supported:

**Pattern A: Chart-managed Secret** (inline values)
```yaml
secrets:
  OPENROUTER_API_KEY: sk-or-...
  ANTHROPIC_API_KEY: sk-ant-...
  # ... 40+ API keys
```

**Pattern B: External Secret** (External Secrets Operator)
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
        key: tenants/tenant-a/hermes
        property: OPENROUTER_API_KEY
```

**Pattern C: Pre-existing Secret**
```yaml
secrets:
  existingSecret: hermes-tenant-a-secrets
```

---

## 4. BOOTSTRAP & PERSISTENCE

### Bootstrap Process
Init container runs **before** main container:

```yaml
# From deployment.yaml (lines 102-136)
initContainers:
  - name: bootstrap-config
    command:
      - /bin/sh
      - -ec
    args:
      - |
        mkdir -p /opt/data/home
        if [ -f /bootstrap/config.yaml ]; then
          if [ "true" = "true" ] || [ ! -f /opt/data/config.yaml ]; then
            cp /bootstrap/config.yaml /opt/data/config.yaml
          fi
        fi
        if [ -f /bootstrap/SOUL.md ]; then
          if [ "true" = "true" ] || [ ! -f /opt/data/SOUL.md ]; then
            cp /bootstrap/SOUL.md /opt/data/SOUL.md
          fi
        fi
```

**Key behavior**:
- `bootstrap.overwrite=true` (default): Always overwrite persisted config with Helm-rendered version
- `bootstrap.overwrite=false`: Only seed if file doesn't exist (preserve user changes)
- `bootstrap.existingConfigMap`: Use external ConfigMap instead of chart-managed one

### Persistence Configuration
```yaml
persistence:
  enabled: true
  existingClaim: ""           # Bind to pre-provisioned PVC
  mountPath: /opt/data        # HERMES_HOME location
  accessMode: ReadWriteOnce   # Single-writer
  size: 5Gi                   # Default size
  storageClass: ""            # Use cluster default
  annotations: {}
```

**PVC Template** ([permalink](https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/pvc.yaml)):
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "hermes-agent.fullname" . }}-data
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: {{ .Values.persistence.storageClass | quote }}
  resources:
    requests:
      storage: {{ .Values.persistence.size | quote }}
```

### NPM Packages (Optional)
Init container installs global npm packages:

```yaml
npmPackages:
  - lodash@4.17.21
  - axios@1.6.0
```

**Behavior**:
- Installed to `$HERMES_HOME/npm-global`
- Cached via hash file (only reinstalls if list changes)
- Exposed via `PATH` and `NODE_PATH`

---

## 5. SECRETS HANDLING

### 40+ Supported API Keys
```yaml
secrets:
  # LLM Providers
  OPENROUTER_API_KEY: ""
  OPENAI_API_KEY: ""
  ANTHROPIC_API_KEY: ""
  GOOGLE_API_KEY: ""
  GEMINI_API_KEY: ""
  GROQ_API_KEY: ""
  MISTRAL_API_KEY: ""
  
  # Search & Crawling
  EXA_API_KEY: ""
  FIRECRAWL_API_KEY: ""
  BROWSERBASE_API_KEY: ""
  BROWSERBASE_PROJECT_ID: ""
  
  # Chat Platforms
  TELEGRAM_BOT_TOKEN: ""
  DISCORD_BOT_TOKEN: ""
  SLACK_BOT_TOKEN: ""
  SLACK_APP_TOKEN: ""
  SIGNAL_HTTP_URL: ""
  SIGNAL_ACCOUNT: ""
  
  # Enterprise Platforms
  DINGTALK_CLIENT_ID: ""
  DINGTALK_CLIENT_SECRET: ""
  FEISHU_APP_ID: ""
  FEISHU_APP_SECRET: ""
  WECOM_BOT_ID: ""
  WECOM_SECRET: ""
  
  # Email
  EMAIL_ADDRESS: ""
  EMAIL_PASSWORD: ""
  EMAIL_IMAP_HOST: ""
  EMAIL_IMAP_PORT: ""
  EMAIL_SMTP_HOST: ""
  EMAIL_SMTP_PORT: ""
  
  # System
  API_SERVER_KEY: ""
  WEBHOOK_SECRET: ""
  SUDO_PASSWORD: ""
  GITHUB_TOKEN: ""
```

### Validation Rules
From deployment.yaml (lines 28-36):

```yaml
# API Server requires API_SERVER_KEY
{{- if and $apiServerEnabled (not $secretManagedExternally) (eq (printf "%v" .Values.secrets.API_SERVER_KEY) "") }}
{{- fail "apiServer.enabled requires secrets.API_SERVER_KEY when neither secrets.existingSecret nor externalSecret.enabled is set" }}
{{- end }}

# Telegram Webhook requires TELEGRAM_BOT_TOKEN
{{- if and .Values.telegramWebhook.enabled (not $secretManagedExternally) (eq (printf "%v" .Values.secrets.TELEGRAM_BOT_TOKEN) "") }}
{{- fail "telegramWebhook.enabled requires secrets.TELEGRAM_BOT_TOKEN when neither secrets.existingSecret nor externalSecret.enabled is set" }}
{{- end }}
```

---

## 6. SERVICE & INGRESS CONFIGURATION

### Service Auto-Port Derivation
Service ports are **automatically derived** from enabled listeners:

```yaml
# From service.yaml (lines 4-14)
{{- if eq (len $servicePorts) 0 }}
  {{- if .Values.apiServer.enabled }}
    {{- $servicePorts = append $servicePorts (dict "name" "api-server" "port" (.Values.apiServer.port | int) ...) }}
  {{- end }}
  {{- if .Values.webhook.enabled }}
    {{- $servicePorts = append $servicePorts (dict "name" "webhook" "port" (.Values.webhook.port | int) ...) }}
  {{- end }}
  {{- if .Values.telegramWebhook.enabled }}
    {{- $servicePorts = append $servicePorts (dict "name" "telegram-webhook" "port" (.Values.telegramWebhook.port | int) ...) }}
  {{- end }}
{{- end }}
```

**Example**: Enable API server → Service automatically exposes port 8642

### Ingress Prerequisites
```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
  hosts:
    - host: hermes.tenant-a.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - hosts:
        - hermes.tenant-a.example.com
      secretName: hermes-tenant-a-tls
```

**Validation** (ingress.yaml, line 3-4):
```yaml
{{- if not .Values.service.enabled }}
{{- fail "ingress.enabled=true requires service.enabled=true" }}
{{- end }}
```

### Istio VirtualService
```yaml
virtualService:
  enabled: true
  gateways:
    - istio-system/public-gateway
  hosts:
    - hermes.tenant-a.example.com
  timeout: 3600s
  servicePortNumber: 8642
```

**Validation**: Requires at least one gateway and one host.

---

## 7. NETWORK ISOLATION & SECURITY

### Network Policy
```yaml
networkPolicy:
  enabled: false
  policyTypes:
    - Ingress
    - Egress
  ingress: []
  egress: []
```

**Example** (from test values):
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
        - protocol: TCP
          port: 53
```

### Tenant Isolation
```yaml
tenantIsolation:
  enabled: false
  allowSameNamespace: true
  authorizedNamespaces: []
  allowDns: true
  additionalEgress: []
  podAntiAffinity:
    enabled: false
    type: preferredDuringSchedulingIgnoredDuringExecution
    topologyKey: kubernetes.io/hostname
```

**When enabled** (deployment.yaml, lines 316-324):
- Adds pod anti-affinity rules based on `tenant.id`
- Restricts network egress to authorized namespaces
- Allows DNS by default

### Pod Security Context
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

**Implications**:
- ✅ Non-root execution (UID 1000)
- ✅ Read-only root filesystem
- ✅ No Linux capabilities
- ✅ RuntimeDefault seccomp
- ⚠️ Requires writable `/tmp` and `$HERMES_HOME`

---

## 8. OPERATOR-READY MODE

### Two Deployment Modes

**Mode 1: Direct Deployment** (default)
- Helm renders Deployment, Service, Ingress, etc. directly
- One release per tenant
- Full lifecycle control via Helm

**Mode 2: Operator-Ready** (experimental)
- Helm renders `HermesTenant` CRDs and CR objects
- Separate controller reconciles CRs into running workloads
- Helm acts as CRD/CR producer, not workload owner

### Operator Configuration
```yaml
operator:
  enabled: false
  controllerClass: hermes.ai/default
  installCustomResources: true
  managementNamespace: ""
  defaultChartValues: {}
  tenants: []
```

### HermesTenant CRD
```yaml
apiVersion: hermes.ai/v1alpha1
kind: HermesTenant
metadata:
  name: hermes-tenant-a
  namespace: tenant-a
spec:
  controllerClass: hermes.ai/default
  release:
    name: hermes-tenant-a
    namespace: tenant-a
  tenant:
    id: tenant-a
    labels:
      environment: prod
  chartValues:
    apiServer:
      enabled: true
      port: 8642
    secrets:
      API_SERVER_KEY: ...
```

**Note**: Chart does **not** bundle a controller. You must provide your own.

---

## 9. RESOURCE REQUIREMENTS & DEFAULTS

### Compute Resources
```yaml
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: "2"
    memory: 4Gi
```

### Browser Shared Memory
```yaml
browser:
  shm:
    enabled: true
    mountPath: /dev/shm
    sizeLimit: 1Gi
```

**Why**: Chromium/Playwright needs shared memory for browser automation.

### Termination Grace Period
```yaml
terminationGracePeriodSeconds: 120
```

**Why**: Allows Hermes to gracefully shut down long-running operations.

### Pod Disruption Budget
```yaml
pdb:
  enabled: false
  minAvailable: 1
```

---

## 10. MULTI-TENANT SAAS DEPLOYMENT PATTERN

### Recommended Architecture
```
SaaS Platform
├── Namespace: tenant-a
│   ├── Helm Release: hermes-tenant-a
│   ├── PVC: hermes-tenant-a-data (5Gi)
│   ├── Secret: hermes-tenant-a-secrets
│   ├── Service: hermes-tenant-a
│   ├── Ingress: hermes.tenant-a.example.com
│   └── NetworkPolicy: tenant-a-isolation
├── Namespace: tenant-b
│   ├── Helm Release: hermes-tenant-b
│   ├── PVC: hermes-tenant-b-data (5Gi)
│   ├── Secret: hermes-tenant-b-secrets
│   ├── Service: hermes-tenant-b
│   ├── Ingress: hermes.tenant-b.example.com
│   └── NetworkPolicy: tenant-b-isolation
└── Namespace: ingress-nginx
    └── Ingress Controller
```

### Per-Tenant Values Template
```yaml
# values-tenant-a.yaml
fullnameOverride: hermes-tenant-a

tenant:
  id: tenant-a
  labels:
    environment: prod
    owner: platform

apiServer:
  enabled: true
  port: 8642

secrets:
  existingSecret: hermes-tenant-a-secrets

persistence:
  enabled: true
  existingClaim: hermes-tenant-a-data
  size: 10Gi

service:
  enabled: true
  annotations:
    external-dns.alpha.kubernetes.io/hostname: hermes.tenant-a.example.com

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: hermes.tenant-a.example.com
      paths:
        - path: /
          pathType: Prefix

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

### Deployment Command
```bash
helm install hermes-tenant-a ./hermes-agent-helm-chart \
  --namespace tenant-a \
  --create-namespace \
  -f values-tenant-a.yaml
```

---

## 11. OPINIONATED DEFAULTS & POTENTIAL CONFLICTS

### ✅ Good Defaults
| Setting | Default | Rationale |
|---------|---------|-----------|
| `replicaCount` | 1 | Single-writer safety |
| `strategy.type` | Recreate | Prevents concurrent PVC access |
| `persistence.enabled` | true | Preserves state across restarts |
| `persistence.accessMode` | ReadWriteOnce | Enforces single-writer |
| `bootstrap.overwrite` | true | Helm is source of truth |
| `runAsNonRoot` | true | Security best practice |
| `readOnlyRootFilesystem` | true | Immutable container |
| `serviceAccount.automountServiceAccountToken` | false | Least privilege |

### ⚠️ Potential Conflicts for SaaS

| Setting | Default | Conflict | Solution |
|---------|---------|----------|----------|
| `service.enabled` | false | No network exposure by default | Explicitly enable + configure |
| `ingress.enabled` | false | No ingress by default | Explicitly enable + configure |
| `networkPolicy.enabled` | false | No network isolation by default | Explicitly enable for multi-tenant |
| `rbac.create` | false | No RBAC by default | Enable only if Hermes needs K8s API access |
| `bootstrap.overwrite` | true | Overwrites user config on every restart | Set to `false` if users customize config |
| `replicaCount: 1` | Hard limit | Cannot scale horizontally | Create multiple releases per tenant |
| `persistence.size` | 5Gi | May be too small for production | Adjust per tenant needs |
| `resources.limits.memory` | 4Gi | May be insufficient for large operations | Increase for heavy workloads |

### 🔴 Breaking Changes Risk

**If you change these, pods will restart**:
- `config.values` or `config.raw` (triggers checksum change)
- `secrets.*` values (triggers checksum change)
- `bootstrap.overwrite` (changes init container behavior)
- `npmPackages` (triggers npm reinstall)

**Mitigation**: Use `secrets.existingSecret` and `bootstrap.existingConfigMap` to decouple config from Helm releases.

---

## 12. VALIDATION & SCHEMA

### JSON Schema Validation
The chart includes `values.schema.json` (875 lines) that validates:

- ✅ `replicaCount >= 1`
- ✅ `strategy.type` in [Recreate, RollingUpdate]
- ✅ `persistence.accessMode` in [ReadWriteOnce, ReadWriteMany, ReadOnlyMany]
- ✅ `service.type` in [ClusterIP, NodePort, LoadBalancer, ExternalName]
- ✅ `ingress.enabled=true` requires `service.enabled=true`
- ✅ `telegramWebhook.enabled=true` requires `telegramWebhook.url`
- ✅ `apiServer.enabled=true` requires `secrets.API_SERVER_KEY`
- ✅ Port numbers in valid range (1-65535)

### Helm Lint Checks
```bash
helm lint .
helm lint . -f ci/test-values.yaml
helm lint . -f ci/tenant-isolation-values.yaml
helm lint . -f ci/external-secret-values.yaml
```

---

## 13. TESTING & CI/CD

### Regression Test Scenarios
```
ci/
├── test-values.yaml                    # Full feature test
├── existing-claim-values.yaml          # Pre-provisioned PVC
├── external-bootstrap-values.yaml      # External ConfigMap
├── default-service-ports-values.yaml   # Auto-derived ports
├── external-secret-values.yaml         # External Secrets Operator
├── tenant-isolation-values.yaml        # Multi-tenant isolation
├── operator-values.yaml                # Operator-ready mode
├── invalid-telegram-values.yaml        # Validation failure
├── invalid-service-values.yaml         # Validation failure
└── verify.sh                           # Regression script
```

### Verification Command
```bash
bash ci/verify.sh
```

---

## 14. COMPOSITION & INTEGRATION POINTS

### Extensibility Hooks

| Hook | Purpose | Example |
|------|---------|---------|
| `secrets.existingSecret` | Reuse external Secret | Vault, Sealed Secrets, AWS Secrets Manager |
| `bootstrap.existingConfigMap` | Reuse external ConfigMap | Shared platform config |
| `extraEnv` | Add environment variables | Custom settings |
| `extraEnvFrom` | Reference ConfigMaps/Secrets | Platform-managed config |
| `extraVolumes` / `extraVolumeMounts` | Add volumes | Projected credentials, shared storage |
| `extraInitContainers` | Add init containers | Bootstrap jobs, setup scripts |
| `extraContainers` | Add sidecars | Service mesh, logging agents |
| `extraObjects` | Add arbitrary resources | ServiceMonitor, ExternalSecret, policies |

### Example: External Secrets Operator Integration
```yaml
externalSecret:
  enabled: true
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: platform-secrets
  target:
    name: hermes-tenant-a-secrets
    creationPolicy: Owner
    deletionPolicy: Retain
  data:
    - secretKey: OPENROUTER_API_KEY
      remoteRef:
        key: hermes-agent/prod
        property: OPENROUTER_API_KEY
    - secretKey: API_SERVER_KEY
      remoteRef:
        key: hermes-agent/prod
        property: API_SERVER_KEY
```

---

## 15. PRODUCTION DEPLOYMENT CHECKLIST

### Pre-Deployment
- [ ] Verify K8s cluster version >= 1.25.0
- [ ] Provision PVCs per tenant (or use `persistence.existingClaim`)
- [ ] Create Secrets per tenant (or use External Secrets Operator)
- [ ] Configure storage class (or use cluster default)
- [ ] Set up Ingress controller (nginx, Istio, etc.)
- [ ] Configure DNS for tenant domains
- [ ] Set up cert-manager for TLS (if using Ingress)

### Deployment
- [ ] Create namespace per tenant
- [ ] Create PVC per tenant
- [ ] Create Secret per tenant
- [ ] Render Helm chart: `helm template hermes . -f values-tenant.yaml`
- [ ] Review rendered manifests
- [ ] Install: `helm install hermes . -f values-tenant.yaml`
- [ ] Verify pod is running: `kubectl get pods -n tenant-a`
- [ ] Check logs: `kubectl logs -n tenant-a deployment/hermes-tenant-a`

### Post-Deployment
- [ ] Verify config.yaml was bootstrapped: `kubectl exec -n tenant-a deployment/hermes-tenant-a -- cat /opt/data/config.yaml`
- [ ] Test API server: `curl http://hermes.tenant-a.example.com/health`
- [ ] Verify ingress is working
- [ ] Test with sample request
- [ ] Monitor resource usage
- [ ] Set up alerts for pod restarts, OOMKilled, etc.

### Ongoing
- [ ] Monitor PVC usage (alert at 80%)
- [ ] Review logs for errors
- [ ] Test failover (delete pod, verify recovery)
- [ ] Plan upgrades (test in staging first)
- [ ] Document tenant-specific configurations

---

## 16. KNOWN LIMITATIONS & WORKAROUNDS

### Limitation 1: Single Replica
**Problem**: Cannot scale horizontally within a single release.  
**Workaround**: Create multiple releases (one per tenant or per shard).

### Limitation 2: Recreate Strategy
**Problem**: Downtime during updates.  
**Workaround**: Use blue-green deployments (separate releases).

### Limitation 3: PVC Binding
**Problem**: PVC is tied to a specific node (if using local storage).  
**Workaround**: Use network storage (NFS, EBS, GCE Persistent Disk).

### Limitation 4: No Built-in Backup
**Problem**: PVC data is not automatically backed up.  
**Workaround**: Use Velero or cloud-native backup solutions.

### Limitation 5: Operator Mode Not Bundled
**Problem**: Chart doesn't include a controller.  
**Workaround**: Implement your own controller or use a third-party one.

---

## 17. COMPARISON: DIRECT vs OPERATOR-READY MODE

| Aspect | Direct Mode | Operator-Ready Mode |
|--------|-------------|-------------------|
| **Helm Role** | Full lifecycle owner | CRD/CR producer only |
| **Controller** | None needed | Required (external) |
| **Scaling** | Multiple releases | Multiple CRs |
| **Config Updates** | Helm upgrade | CR update |
| **Complexity** | Lower | Higher |
| **Flexibility** | Limited | Higher |
| **Production Ready** | ✅ Yes | ⚠️ Experimental |

---

## 18. GITHUB PERMALINKS

All code references use commit SHA: `e3b685d4d0288668a37216435742cd0a659ebc6c`

- **Deployment**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/deployment.yaml
- **ConfigMap**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/configmap.yaml
- **Secret**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/secret.yaml
- **PVC**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/pvc.yaml
- **Service**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/service.yaml
- **Ingress**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/ingress.yaml
- **External Secret**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/external-secret.yaml
- **Helpers**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/templates/_helpers.tpl
- **Values Schema**: https://github.com/ultraworkers/hermes-agent-helm-chart/blob/e3b685d4d0288668a37216435742cd0a659ebc6c/values.schema.json

---

## SUMMARY FOR PRODUCT INTEGRATION

### ✅ What Works Well for SaaS
1. **One release per tenant** = clean isolation
2. **Flexible secret management** = integrates with Vault, AWS Secrets Manager, etc.
3. **Composable bootstrap** = external ConfigMap support
4. **Network policies** = tenant-scoped isolation
5. **Operator-ready** = future-proof for controller-based deployments

### ⚠️ What Requires Customization
1. **Horizontal scaling** = must create multiple releases
2. **Config updates** = must use `bootstrap.overwrite=false` + external ConfigMap
3. **Secrets rotation** = must use External Secrets Operator
4. **Multi-region** = must deploy separate clusters/releases
5. **Backup/restore** = must implement separately

### 🔴 What's Missing
1. **Built-in controller** = operator mode requires external implementation
2. **Backup automation** = no native backup/restore
3. **Monitoring** = no ServiceMonitor or Prometheus integration
4. **Logging** = no structured logging or log aggregation
5. **Metrics** = no built-in metrics collection

