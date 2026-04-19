# HERMES PRODUCT INTEGRATION MASTER GUIDE

## 1. Executive Summary

**What Hermes is in the context of my product**: Hermes Agent is a stateful, single-writer AI agent runtime that provides LLM-powered automation with built-in tools for web browsing, terminal access, file operations, and messaging platform integration. It's designed as a per-tenant service where each tenant gets their own isolated Hermes instance.

**How Hermes should fit into my hosted architecture**: Each tenant should have exactly one Hermes instance running as a Kubernetes Deployment with persistent storage. The architecture follows a clean separation: Hono backend manages tenant metadata and business logic, a cluster-side daemon handles Kubernetes/Helm operations, and Hermes runs as a stateful container with its own persistent data.

**Key recommendation in plain language**: Deploy one Helm release per tenant using the official ultraworkers/hermes-agent-helm-chart. Use External Secrets Operator for secret management, ensure single-replica deployments, and implement proper backup strategies for the persistent data volume.

**Main integration risks and decisions**:
- **Stateful single-writer constraint**: Hermes cannot scale horizontally within a single instance - this is by design due to shared mutable state
- **Heavy image size**: 2.6GB image impacts deployment speed and storage requirements
- **Secret management complexity**: 40+ potential API keys need proper rotation and security
- **Backup requirements**: No built-in backup mechanism - must implement external solution
- **Domain routing**: Wildcard subdomains require proper DNS and ingress configuration

## 2. Final Architecture

### Responsibility Split

| Component | Owns | Should NOT Own | Data Needs |
|-----------|------|----------------|------------|
| **Frontend** | Tenant UI, provider selection, status display | Direct Kubernetes operations, secret management | Tenant metadata, provider readiness status, runtime status |
| **Hono Backend** | Tenant lifecycle, metadata storage, API orchestration | Direct Helm/Kubernetes calls, cluster operations | Tenant ID, domain, provider config, secret references |
| **Cluster-Side Bridge/Daemon** | Helm operations, Kubernetes resource management | Business logic, tenant metadata storage | Helm values, tenant ID, cluster credentials |
| **Kubernetes/k3s** | Container orchestration, networking, storage | Tenant business logic, secret content | Pod specs, PVCs, Services, Ingresses |
| **Helm Chart** | Kubernetes resource templating, configuration injection | Tenant metadata, secret content | Values.yaml, secrets, config |
| **Hermes Runtime** | Agent execution, persistent data, tool execution | Network routing, secret management, tenant metadata | config.yaml, .env, memories, skills, cron jobs |
| **Cloudflare** | DNS, TLS termination, DDoS protection | Application routing, tenant isolation | Wildcard DNS records, TLS certificates |

### Clean Boundaries
- **Frontend ↔ Backend**: REST API only - no direct cluster access
- **Backend ↔ Bridge**: gRPC or REST API - bridge abstracts Kubernetes complexity  
- **Bridge ↔ Kubernetes**: Direct API calls or Helm CLI - no business logic
- **Hermes ↔ Everything**: Isolated container - only receives config via mounted volumes/secrets

## 3. Hermes Runtime Model

### What Hermes Expects at Runtime

**Persistent Data Directory (`/opt/data`)**:
- `config.yaml` - Main configuration file (model, terminal, browser settings)
- `.env` - Environment variables and API keys (auto-generated from secrets)
- `SOUL.md` - Agent personality/system prompt
- `memories/` - Cross-session memory files (MEMORY.md, USER.md)
- `skills/` - Custom and bundled skills (procedural memory)
- `cron/` - Scheduled job definitions
- `sessions/` - Chat history and session data
- `logs/` - Gateway and error logs
- `state.db` - SQLite database for response storage and state
- `workspace/` - Working directory for file operations

**Runtime Requirements**:
- **Single replica only**: Hermes has shared mutable state - multiple replicas cause data corruption
- **Persistent storage**: All data must survive pod restarts/deletions
- **Proper file permissions**: Runs as non-root user (UID 1000) with fsGroup 1000
- **Resource limits**: Minimum 1Gi memory, 500m CPU; recommended 4Gi memory, 2 CPU
- **Init container**: Bootstrap config/SOUL from ConfigMap on first run

### Kubernetes Mounting Strategy

```yaml
volumes:
- name: data
  persistentVolumeClaim:
    claimName: hermes-{{ .tenantId }}-data

volumeMounts:
- name: data
  mountPath: /opt/data
```

**PVC Requirements**:
- StorageClass: Any (standard works fine)
- AccessMode: ReadWriteOnce (single writer)
- Size: Minimum 5Gi, recommended 10Gi for production
- Retention: Must survive pod deletion/recreation

## 4. Helm Chart Integration

### Helm Responsibilities
- Template Kubernetes resources (Deployment, Service, Ingress, PVC, ConfigMap, Secret)
- Inject configuration via bootstrap init container
- Handle secret mounting and environment variable setup
- Enforce single-replica constraint when persistence is enabled
- Provide validation via JSON Schema

### Required Values Overrides

**Essential for Multi-Tenant**:
```yaml
nameOverride: "hermes-{{ .tenantId }}"
fullnameOverride: "hermes-{{ .tenantId }}"

# Tenant-specific domain
ingress:
  hosts:
    - host: "{{ .tenantId }}.hermeshq.net"
      paths:
        - path: /
          pathType: Prefix

# Provider configuration
config:
  values:
    model:
      default: "{{ .providerModel }}"
      provider: "{{ .providerType }}"
      base_url: "{{ .providerBaseUrl }}"

# Secrets (via External Secrets Operator)
secrets:
  existingSecret: "hermes-{{ .tenantId }}-secrets"

# Persistence
persistence:
  existingClaim: "hermes-{{ .tenantId }}-data"
```

### Configuration Injection Strategy

**Config.yaml**: Use `config.values` in Helm values to generate config.yaml via bootstrap ConfigMap
**Secrets**: Use External Secrets Operator to populate Kubernetes Secret, reference via `secrets.existingSecret`
**SOUL.md**: Use `soul.text` in Helm values for custom personality

### Chart Management vs Platform Management

| Managed by Helm Chart | Managed by Platform |
|----------------------|-------------------|
| Kubernetes resource templates | Tenant metadata storage |
| ConfigMap generation | Secret content (via ESO) |
| PVC creation | Backup/restore operations |
| Service/Ingress creation | Monitoring/alerting |
| Resource limits | Scaling decisions |

### Chart Limitations and Workarounds

**Current Issues**:
- No built-in backup mechanism
- Recreate strategy causes downtime during updates
- Limited monitoring integration

**Workarounds**:
- Implement external backup with Velero
- Use blue/green deployment pattern for zero-downtime updates
- Add custom monitoring sidecar if needed

## 5. Kubernetes / k3s Runtime Design

### Deployment vs StatefulSet
**Recommendation: Deployment** (not StatefulSet)

**Why Deployment**:
- Helm chart is designed for Deployment
- Single replica requirement doesn't need StatefulSet features
- Simpler upgrade/rollback semantics
- Works well with Recreate strategy

### Replica Count and Single Replica Constraint
- **ReplicaCount**: Always 1
- **Why single replica matters**: Hermes stores mutable shared state in `/opt/data` including SQLite databases, memory files, and session data. Multiple replicas would cause data corruption and race conditions.

### Service Requirements
```yaml
service:
  enabled: true
  type: ClusterIP
  ports:
    - name: api-server
      port: 8642
      targetPort: 8642
      protocol: TCP
```

### Storage/PVC Requirements
- **StorageClass**: Any (k3s local-path works fine)
- **AccessMode**: ReadWriteOnce
- **Size**: 10Gi recommended for production
- **Retention**: Must survive pod recreation

### Readiness/Liveness Considerations
**Current**: No built-in probes in Helm chart
**Recommendation**: Add HTTP probe to `/health` endpoint on port 8642

```yaml
probes:
  readiness:
    httpGet:
      path: /health
      port: api-server
    initialDelaySeconds: 10
    periodSeconds: 30
  liveness:
    httpGet:
      path: /health
      port: api-server
    initialDelaySeconds: 30
    periodSeconds: 60
```

### Ingress/Routing Considerations
- **Ingress Controller**: NGINX Ingress Controller (standard for k3s)
- **TLS**: Let's Encrypt via cert-manager for wildcard certificate
- **Host**: `{{tenantId}}.hermeshq.net`
- **Path**: Root path `/` with rewrite to root

### Traefik/NGINX Assumptions
- **NGINX Ingress**: Standard for k3s, supports rewrite annotations
- **Traefik**: Would require different annotation syntax but same functionality
- **Assumption**: Ingress controller is already installed and configured

### Wildcard Subdomain Integration
- **DNS**: Cloudflare manages `*.hermeshq.net` wildcard DNS record
- **Certificate**: Wildcard TLS certificate via cert-manager + Let's Encrypt
- **Routing**: Each tenant gets unique subdomain routed to their Service

## 6. Cloudflare + Domain Routing Model

### Public Routing Architecture
```
User Browser → Cloudflare (DNS + TLS) → k3s Ingress → Tenant Service → Hermes Pod
```

### Cloudflare Responsibilities
- **DNS**: Wildcard `*.hermeshq.net` pointing to k3s cluster IP
- **TLS**: Wildcard certificate termination at edge
- **DDoS Protection**: Automatic bot detection and rate limiting
- **Caching**: Static asset caching (minimal for Hermes)

### Cluster Ingress Responsibilities
- **Routing**: Map subdomain to correct tenant Service
- **Authentication**: Basic auth or API key validation if needed
- **Rate Limiting**: Per-tenant request limiting
- **Logging**: Access logs for debugging

### Backend Knowledge Requirements
- **Tenant Domain**: Backend stores `tenantId.hermeshq.net` for each tenant
- **Certificate Status**: Backend should monitor certificate validity
- **DNS Propagation**: Backend should handle DNS propagation delays during provisioning

### Frontend Assumptions
- **API Endpoint**: `https://{{tenantId}}.hermeshq.net/v1/chat/completions`
- **Authentication**: Bearer token in Authorization header
- **CORS**: Proper CORS headers for browser access

### Domain Types Clarification
| Domain Type | Purpose | Example |
|-------------|---------|---------|
| **API Domain** | Direct LLM API access | `tenant123.hermeshq.net` |
| **Tenant Runtime Subdomain** | Same as API domain - one subdomain per tenant | `tenant123.hermeshq.net` |
| **Dashboard Access Pattern** | Separate dashboard domain | `dashboard.hermeshq.net` |

## 7. Backend (Hono) Requirements

### Tenant Metadata Storage
**Required Fields**:
```typescript
interface Tenant {
  id: string;                    // Unique tenant ID
  domain: string;               // tenant123.hermeshq.net
  status: 'provisioning' | 'ready' | 'error' | 'suspended';
  provider: {
    type: 'openrouter' | 'anthropic' | 'openai';
    model: string;
    apiKeyRef: string;         // Reference to secret in ESO
  };
  createdAt: Date;
  updatedAt: Date;
}
```

### Backend-Generated Configuration
- **Tenant ID**: UUID or slug-based identifier
- **Domain**: `{{tenantId}}.hermeshq.net`
- **Helm Release Name**: `hermes-{{tenantId}}`
- **Kubernetes Namespace**: `hermes-{{tenantId}}` (optional, can use shared namespace)

### Non-Backend Concerns
- **Direct Kubernetes operations**: Handled by cluster daemon
- **Secret content**: Managed by External Secrets Operator
- **Helm chart details**: Abstracted by cluster daemon

### Cluster Daemon Communication
**API Contract**:
```typescript
// Create tenant
POST /tenants
{
  tenantId: string,
  helmValues: HelmValues,
  namespace: string
}

// Get status  
GET /tenants/{tenantId}/status

// Update provider config
PATCH /tenants/{tenantId}/provider
{
  provider: { type, model, apiKeyRef }
}

// Suspend tenant
POST /tenants/{tenantId}/suspend

// Resume tenant  
POST /tenants/{tenantId}/resume

// Delete tenant
DELETE /tenants/{tenantId}
```

### Exposed APIs to Frontend
```typescript
// Create new tenant
POST /api/tenants
{
  provider: { type: 'openrouter', model: 'anthropic/claude-opus-4.6' }
}

// Get tenant status
GET /api/tenants/{tenantId}

// Update provider config
PATCH /api/tenants/{tenantId}/provider
{
  apiKey: string  // Backend stores in ESO, returns reference
}

// Get runtime status
GET /api/tenants/{tenantId}/runtime

// Suspend/resume
POST /api/tenants/{tenantId}/actions
{ action: 'suspend' | 'resume' | 'restart' }
```

### Provider Readiness Awareness
Backend must track:
- **Provider selection**: Which LLM provider is chosen
- **API key status**: Whether valid API key is provided
- **Runtime status**: Whether Hermes pod is running and healthy
- **Configuration completeness**: Whether all required settings are present

### Supported Lifecycle Actions
- **Create**: Provision new tenant with initial config
- **Get Status**: Return current tenant and runtime status
- **Update Provider**: Change LLM provider or API key
- **Suspend**: Scale deployment to 0 replicas (preserve data)
- **Resume**: Scale deployment back to 1 replica
- **Restart**: Rollout restart of deployment
- **Delete**: Remove all Kubernetes resources and data

## 8. Cluster-Side Bridge / Daemon Requirements

### API Surface
**Minimal gRPC/REST API**:
- `CreateTenant(request: CreateTenantRequest) returns (CreateTenantResponse)`
- `GetTenantStatus(tenantId: string) returns (TenantStatus)`
- `UpdateTenantConfig(request: UpdateConfigRequest) returns (UpdateConfigResponse)`
- `SuspendTenant(tenantId: string) returns (OperationStatus)`
- `ResumeTenant(tenantId: string) returns (OperationStatus)`
- `DeleteTenant(tenantId: string) returns (OperationStatus)`

### Translation Layer Responsibilities
- **Helm Operations**: Execute Helm install/upgrade/uninstall commands
- **Kubernetes Validation**: Verify resources exist and are healthy
- **Error Handling**: Translate Kubernetes/Helm errors to user-friendly messages
- **Status Aggregation**: Combine pod status, service status, and ingress status

### Communication Method
**Recommendation: Helm CLI over Kubernetes API**

**Why Helm CLI**:
- Official chart is designed for Helm
- Handles complex templating and validation
- Built-in rollback capabilities
- Simpler than raw Kubernetes API calls
- Leverages Helm's release management

### Response to Hono Backend
**Success Response**:
```json
{
  "status": "success",
  "tenantId": "tenant123",
  "releaseName": "hermes-tenant123",
  "namespace": "hermes",
  "domain": "tenant123.hermeshq.net"
}
```

**Error Response**:
```json
{
  "status": "error", 
  "code": "HELM_INSTALL_FAILED",
  "message": "Helm install failed: timeout waiting for deployment",
  "details": { /* Helm error details */ }
}
```

### Upward Abstraction
**Should NOT expose**:
- Raw Kubernetes resource YAML
- Helm chart internal details
- Cluster-specific implementation details
- Low-level networking configuration

**Should expose**:
- High-level tenant status
- Simple success/error indicators
- User-friendly error messages
- Standardized API contract

## 9. Frontend Requirements

### Runtime State Display
**High-Level States**:
- **Not Configured**: No provider selected
- **Configuring**: Provider selected, waiting for API key
- **Deploying**: Backend provisioning resources
- **Ready**: Running and accepting requests
- **Suspended**: Stopped but data preserved
- **Error**: Failed to deploy or run

### Config/Setup States to Surface
- **Provider Selection**: Dropdown of supported providers
- **API Key Input**: Secure input field for API key
- **Model Selection**: Model options based on provider
- **Domain Display**: Show assigned subdomain
- **Status Indicators**: Visual indicators for each state

### Frontend Boundaries
**Should NEVER manage directly**:
- Kubernetes resources
- Helm releases
- Secret content
- Network configuration
- Backup operations

### Hermes Instance Mental Model
- **One instance per tenant**: Each tenant has exactly one Hermes
- **Persistent identity**: Instance survives restarts/suspensions
- **Provider-configurable**: Can change LLM provider without losing data
- **Domain-bound**: Accessible only via assigned subdomain

### Useful Setup/Readiness States
- **"Waiting for API key"**: Provider selected but no key provided
- **"Deploying..."**: Resources being created (show progress)
- **"Testing connection"**: Verifying LLM provider connectivity
- **"Ready to use!"**: Fully operational with green indicator
- **"Suspended - click to resume"**: Clear call-to-action

## 10. Provider / Config Readiness Model

### Normalized Readiness States
| State | Condition | Frontend Display |
|-------|-----------|------------------|
| **ready** | Provider selected + valid API key + runtime healthy | Green "Ready" badge |
| **missing_provider** | No provider selected | "Select LLM Provider" CTA |
| **missing_api_key** | Provider selected but no API key | "Enter API Key" input field |
| **runtime_unreachable** | Runtime deployed but not responding | Red "Error" with retry option |
| **partially_configured** | API key provided but untested | Yellow "Testing..." indicator |

### Backend Status Shape
```typescript
interface ProviderReadiness {
  status: 'ready' | 'missing_provider' | 'missing_api_key' | 'runtime_unreachable' | 'partially_configured';
  provider?: {
    type: string;
    model: string;
    lastTestedAt?: Date;
    testResult?: 'success' | 'failure';
    errorMessage?: string;
  };
  runtime?: {
    status: 'running' | 'pending' | 'failed' | 'unknown';
    podStatus?: string;
    lastUpdated: Date;
  };
}
```

### Detection Logic
- **missing_provider**: `tenant.provider === null`
- **missing_api_key**: `tenant.provider !== null && tenant.provider.apiKeyRef === null`
- **partially_configured**: `tenant.provider.apiKeyRef !== null && provider.testResult === null`
- **ready**: `provider.testResult === 'success' && runtime.status === 'running'`
- **runtime_unreachable**: `runtime.status === 'failed' || runtime.status === 'unknown'`

## 11. Persistent Data and State Ownership

### Data Location Ownership
| Data | Location | Owner After Provisioning |
|------|----------|-------------------------|
| **Tenant metadata** | Backend database | Backend |
| **Hermes config.yaml** | PVC (`/opt/data/config.yaml`) | Hermes runtime |
| **API keys/secrets** | Kubernetes Secret (via ESO) | External Secrets Operator |
| **Memory files** | PVC (`/opt/data/memories/`) | Hermes runtime |
| **Skills** | PVC (`/opt/data/skills/`) | Hermes runtime |
| **Session history** | PVC (`/opt/data/sessions/`) | Hermes runtime |
| **Cron jobs** | PVC (`/opt/data/cron/`) | Hermes runtime |
| **SQLite databases** | PVC (`/opt/data/*.db`) | Hermes runtime |

### Seeding vs Runtime Ownership
**Seeded Once During Provisioning**:
- Initial `config.yaml` (from Helm bootstrap)
- Initial `SOUL.md` (from Helm values)
- Initial `.env` (from Kubernetes Secret)

**Hermes Owns After Boot**:
- All modifications to config.yaml
- All memory files and user profiles
- All custom skills created by agent
- All session history and chat logs
- All cron job definitions
- All SQLite database content

**Platform Continues to Own**:
- PVC existence and sizing
- Secret content (via ESO updates)
- Backup/restore operations
- Monitoring and alerting

### Backup Strategy
**Critical Data to Backup**:
- Entire PVC contents (`/opt/data`)
- Kubernetes Secret contents
- Tenant metadata from backend

**Recommended Tools**:
- **Velero**: For Kubernetes resource and PVC backup
- **External Secrets Operator**: Handles secret backup automatically
- **Database backup**: Regular dumps of backend tenant metadata

## 12. What Is Optional vs Required

| Feature | V1 Status | Reason |
|---------|-----------|--------|
| **Operator/Controller** | Not needed | Helm chart sufficient for V1 |
| **External browser service** | Required | Avoid 500MB Playwright overhead |
| **Browser automation** | Optional | Can use Firecrawl/Jina.ai APIs instead |
| **Terraform** | Not needed | Helm handles resource creation |
| **Multi-cluster support** | Not needed | Single k3s cluster sufficient |
| **Teams/RBAC** | Not needed | Single tenant per instance |
| **Enterprise controls** | Not needed | Basic security sufficient for V1 |
| **Custom domains** | Not needed | Wildcard subdomains sufficient |
| **GPU support** | Not needed | Hermes doesn't require GPU |
| **Modal/Daytona/RL** | Not needed | Advanced features for later |

### V1 Required Features
- Single tenant per Hermes instance
- Wildcard subdomain routing
- External Secrets Operator integration
- Basic provider configuration (OpenRouter, Anthropic, OpenAI)
- API server with authentication
- Persistent data storage
- Simple suspend/resume lifecycle

### Nice to Have Later
- Multi-tenant sharing (teams)
- Custom domain support
- Advanced monitoring and logging
- Automated backup and restore
- Blue/green deployment for zero downtime
- Advanced provider features (fine-tuning, custom models)

## 13. Known Risks / Mismatches / Things to Verify

| Risk | Why It Matters | Current Recommendation | Blocks V1? |
|------|----------------|------------------------|------------|
| **Helm chart drift** | Official chart may update breaking changes | Pin to specific version (0.8.0) | No |
| **Image tag mismatch** | `latest` tag may break compatibility | Use specific version tags | Yes - must pin |
| **Ingress controller assumptions** | Assumes NGINX Ingress | Verify k3s has NGINX Ingress | No - k3s default |
| **Stateful runtime behavior** | Single-writer constraint | Enforce single replica strictly | No - by design |
| **Persistence assumptions** | Assumes PVC survives | Test backup/restore workflow | Yes - critical |
| **Dashboard/plugin mismatch** | May expect different APIs | Use standard OpenAI-compatible API | No |
| **Version compatibility** | Backend/frontend may expect different versions | Standardize on v0.8.0 | No |
| **Heavy image size implications** | 2.6GB impacts deployment time | Pre-pull images to nodes | No - acceptable |

### Critical Verification Items
1. **Image pinning**: Must use specific version, not `latest`
2. **Backup testing**: Verify PVC backup/restore works
3. **Single replica enforcement**: Ensure no scaling beyond 1 replica
4. **Secret rotation**: Test ESO secret updates propagate correctly
5. **Domain routing**: Verify wildcard DNS and TLS work end-to-end

## 14. Recommended V1 Implementation Plan

### Phase 1: Local/kind Validation
**Goal**: Validate end-to-end flow locally
- [ ] Deploy k3s/kind cluster locally
- [ ] Install NGINX Ingress Controller
- [ ] Install cert-manager with self-signed issuer
- [ ] Deploy External Secrets Operator
- [ ] Test single tenant provisioning with mock secrets
- [ ] Verify API server accessibility via subdomain
- [ ] Test basic LLM functionality with test API key

**Duration**: 1-2 days

### Phase 2: Single-Server Production Shape
**Goal**: Production-ready single server deployment
- [ ] Deploy k3s on production server (46.62.238.47)
- [ ] Configure Cloudflare DNS wildcard record
- [ ] Set up Let's Encrypt wildcard certificate
- [ ] Implement Hono backend with tenant metadata
- [ ] Build cluster-side daemon with Helm operations
- [ ] Connect frontend to backend APIs
- [ ] Implement basic monitoring and alerting
- [ ] Test full tenant lifecycle (create, configure, suspend, delete)

**Duration**: 1-2 weeks

### Phase 3: Scale-Up Path
**Goal**: Prepare for multiple tenants and growth
- [ ] Implement automated backup with Velero
- [ ] Add proper monitoring (Prometheus/Grafana)
- [ ] Implement rate limiting per tenant
- [ ] Add audit logging for tenant operations
- [ ] Optimize image pull times (pre-pull to nodes)
- [ ] Plan for multi-server deployment if needed

**Duration**: 2-4 weeks after Phase 2

## 15. Final Backend/Frontend/Server Contract

### Backend Must Know
- Tenant metadata structure and lifecycle states
- Provider configuration requirements and validation
- Cluster daemon API contract for tenant operations
- Secret management via External Secrets Operator
- Domain assignment and DNS requirements

### Cluster/Server Must Know
- Helm chart values structure and overrides
- Kubernetes resource naming conventions
- Single-replica enforcement and persistence requirements
- Ingress configuration for wildcard subdomains
- Error handling and status reporting patterns

### Frontend Must Know
- Tenant status states and transitions
- Provider configuration UI patterns
- API endpoint construction (`https://{{tenant}}.hermeshq.net/v1/...`)
- Authentication requirements (Bearer token)
- Error handling and user feedback patterns

### Hermes Runtime Expects
- Persistent volume mounted at `/opt/data`
- Environment variables for provider configuration
- Single-replica deployment with proper resource limits
- Network access to LLM providers and web services
- Proper file permissions (runs as UID 1000, fsGroup 1000)

---

**This document serves as the single source of truth for Hermes Agent integration into your multi-tenant SaaS product. All teams should reference this guide for implementation decisions and architectural boundaries.**