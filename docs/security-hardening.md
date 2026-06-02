# Security Hardening — Hermes Runtime Operator

## Current Security Posture (Audit)

What is already in place, what is missing, and what needs to be done.

---

## What Is Already Good

### Authentication
- Every bridge API call requires `X-Bridge-Secret` header — validated with
  `subtle.ConstantTimeCompare` (timing-safe, resistant to timing attacks)
- WebSocket exec sessions use short-lived single-use tokens (2-minute expiry,
  consumed on first use — cannot be replayed)
- Bridge secret is stored as a Kubernetes Secret, never in code or env files

### Input Validation
- All instance IDs validated against RFC 1123 DNS label regex before any
  Kubernetes operation: `^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`
- Applied consistently across every handler — no raw user input reaches k8s

### Resource Limits
- Bridge pod has CPU/memory limits (500m CPU, 512Mi RAM)
- Workspace pods get limits from the Helm chart values

### Pod Security (deploy/deployment.yaml — not yet applied to live cluster)
- `runAsNonRoot: true` + `runAsUser: 1000`
- `allowPrivilegeEscalation: false`
- `readOnlyRootFilesystem: true`
- `capabilities: drop: [ALL]`
- `seccompProfile: RuntimeDefault`

---

## What Is Missing or Weak

### 1. Live Bridge Pod Has No Security Context
The live deployment (`deployment-test.yaml`) has `securityContext: {}` — none
of the hardening in `deploy/deployment.yaml` is actually running in production.
The bridge pod runs as root with full capabilities right now.

### 2. ClusterRole Is Too Broad
The bridge ServiceAccount has a `ClusterRole` — it can read and write secrets,
pods, namespaces, and deployments **across the entire cluster**, including
`kube-system`. If the bridge is compromised, an attacker has full cluster access.

It should be a namespaced `Role` covering only workspace namespaces, not a
cluster-wide `ClusterRole`.

### 3. No Rate Limiting
Any caller with the bridge secret can send unlimited requests. A stolen secret
or a runaway backend loop could:
- Create thousands of workspaces (Hetzner cost explosion)
- Hammer the Kubernetes API into degradation
- Fill disk with Helm release history

### 4. No Network Policy on Bridge Namespace
The bridge pod can make outbound connections to anything — other pods,
the Kubernetes API, external internet. There is no NetworkPolicy restricting
what the bridge can reach.

### 5. Workspace Pods Can Reach Each Other
Tenant namespaces have no NetworkPolicy by default (the chart has one but it
is opt-in). A compromised workspace pod can reach other tenants' pods on the
same node via the cluster network.

### 6. Helm Release History Grows Unbounded
Every `helm upgrade` adds a new release revision stored as a Kubernetes Secret.
With frequent config changes, this accumulates indefinitely. No cleanup is configured.

### 7. Bridge Secret Rotation Has No Process
There is no documented process for rotating the bridge secret without downtime.
If the secret leaks, the only option is manual intervention.

### 8. Audit Logging Is Minimal
The bridge logs auth failures but not the full request context (IP, user agent,
which workspace was targeted). No structured log format for SIEM ingestion.

### 9. TLS Termination at Cloudflare Only
Traffic from Cloudflare to your Hetzner server (`178.104.185.60`) travels
over HTTP internally (Cloudflare → Traefik → bridge). If someone gains access
to the Hetzner network, they can read bridge traffic in plaintext.

---

## Hardening Plan — Ordered by Impact

---

### Fix 1 — Apply Security Context to Live Bridge (Critical, 30 min)

The `deployment-test.yaml` is what CI actually deploys. It needs the same
security context as `deploy/deployment.yaml`.

Add to `deploy/deployment-test.yaml` under `spec.template.spec`:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault
```

Add to the container spec:

```yaml
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
```

This stops the bridge from running as root. If the bridge binary is exploited,
the attacker gets a non-root process with no Linux capabilities — not a root shell.

---

### Fix 2 — Scope Down RBAC (High, 1–2 hours)

Replace the `ClusterRole` with a setup that only grants cluster-level permissions
where strictly necessary, and uses namespace-scoped permissions everywhere else.

**What actually needs cluster-level access:**
- `namespaces` — create/get (to create tenant namespaces)
- `nodes` — get/list/watch (for status checks)
- `storageclasses` — get/list/watch (for PVC provisioning)

**What should be namespace-scoped (only in workspace namespaces):**
- `secrets`, `configmaps`, `pods`, `services`, `pvcs`, `serviceaccounts`
- `deployments`, `replicasets`
- `ingresses`, `middlewares`, `ingressroutes`
- `pods/exec`

The bridge already creates per-workspace namespaces. A `RoleBinding` created
in each namespace at workspace creation time gives the bridge access to only
that tenant's resources.

```yaml
# Keep a minimal ClusterRole for cluster-scoped resources only
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hermes-bridge-cluster
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list", "watch", "create"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
---
# Namespace-scoped Role template — bridge creates a RoleBinding to this
# in each workspace namespace at creation time
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hermes-bridge-workspace
rules:
  - apiGroups: [""]
    resources: ["secrets", "configmaps", "pods", "services",
                "persistentvolumeclaims", "serviceaccounts", "events"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["traefik.io", "traefik.containo.us"]
    resources: ["middlewares", "ingressroutes"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

---

### Fix 3 — Rate Limiting on Bridge API (High, 2–3 hours)

Add a per-IP token bucket in the bridge middleware. The shared secret already
limits who can call the bridge, but rate limiting prevents runaway backends
and stolen-secret abuse.

Limits that make sense:

| Endpoint | Limit |
|---|---|
| `POST /v1/instances` (create) | 10 per minute per caller |
| `DELETE /v1/instances/{id}` | 20 per minute |
| All other endpoints | 120 per minute |

Implementation: use `golang.org/x/time/rate` (already in the Go stdlib ecosystem).
A `sync.Map` of `rate.Limiter` keyed by the caller's IP, cleaned up on a ticker.

---

### Fix 4 — Enable NetworkPolicy on Workspace Pods (High, 1 hour)

The chart already has a NetworkPolicy template — it just isn't enabled by default.
Enable it in the Helm values the bridge sends when creating workspaces:

```go
// In buildValues() in helm.go — add to the values map:
"networkPolicy": map[string]any{
    "enabled": true,
    "policyTypes": []string{"Ingress", "Egress"},
    "ingress": []map[string]any{
        // only allow inbound from Traefik (ingress controller)
        {"from": []map[string]any{
            {"namespaceSelector": map[string]any{
                "matchLabels": map[string]any{
                    "kubernetes.io/metadata.name": "kube-system",
                },
            }},
        }},
    },
},
"tenantIsolation": map[string]any{
    "enabled":   true,
    "allowDns":  true,  // workspace pods need DNS
    "tenant": map[string]any{"id": spec.InstanceID},
},
```

This means:
- Workspace pods can only receive traffic from Traefik (the ingress)
- Workspace pods cannot reach other tenants' pods
- Workspace pods can make outbound connections (needed for LLM API calls)
  but only DNS is allowed to `kube-system`

---

### Fix 5 — Helm History Limit (Medium, 15 min)

Add `--history-max` to all Helm operations. Keeps the last 5 revisions only.

In `bridge/helm.go`, on every `action.NewUpgrade`:

```go
upgrade.MaxHistory = 5
```

And on `action.NewInstall`:

```go
install.Replace = true  // reuse release name if previous install failed
```

This prevents the Kubernetes API from being polluted with hundreds of old
release Secrets for active workspaces.

---

### Fix 6 — Bridge NetworkPolicy (Medium, 30 min)

Add a NetworkPolicy to the `hermes-bridge` namespace so the bridge can only
reach what it actually needs:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hermes-bridge
  namespace: hermes-bridge
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hermes-bridge
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Only accept traffic from Traefik
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
  egress:
    # Kubernetes API server
    - ports:
        - port: 6443
          protocol: TCP
    # DNS
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - port: 53
          protocol: UDP
    # Outbound HTTPS (ForwardAuth calls, health checks to workspace pods)
    - ports:
        - port: 443
          protocol: TCP
        - port: 8787
          protocol: TCP
```

---

### Fix 7 — Structured Audit Logging (Medium, 2 hours)

Add a structured log line for every authenticated request including:
- Timestamp
- Method + path
- Instance ID targeted
- Caller IP
- HTTP status returned
- Duration

Format as JSON so it can be ingested by any log aggregator (Grafana Loki,
Datadog, etc.):

```json
{
  "time": "2026-05-20T09:00:00Z",
  "method": "POST",
  "path": "/v1/instances/ws-abc123",
  "instanceId": "ws-abc123",
  "ip": "1.2.3.4",
  "status": 202,
  "durationMs": 45
}
```

---

### Fix 8 — Secret Rotation Process (Medium, document only)

Document and automate bridge secret rotation:

```bash
# 1. Generate new secret
NEW_SECRET=$(openssl rand -hex 32)

# 2. Update k8s Secret (bridge reads it on next request via env var reload)
kubectl -n hermes-bridge create secret generic bridge-auth \
  --from-literal=secret=$NEW_SECRET \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. Rolling restart picks up new secret
kubectl rollout restart deployment/hermes-bridge -n hermes-bridge

# 4. Update backend cluster registry with new secret
# (this is the only downtime window — between step 3 and 4)
```

Zero-downtime rotation requires the bridge to accept both old and new secrets
during a transition window — a future improvement when you have multiple backend
instances that can't all update simultaneously.

---

### Fix 9 — End-to-End TLS (Lower priority, 1–2 hours)

Currently: `User → Cloudflare (TLS) → Hetzner (HTTP) → Traefik → bridge`

The Cloudflare → Hetzner hop is HTTP. Enable Cloudflare's "Full (Strict)" SSL
mode and configure Traefik to serve a self-signed or origin cert so Cloudflare
validates the connection to your server.

In Traefik config, add an origin certificate from Cloudflare (free, 15-year cert)
so the full chain is encrypted. Low urgency since the Hetzner network is not
public — but worth doing before you handle sensitive user data at scale.

---

## Priority Order

| # | Fix | Effort | Impact |
|---|---|---|---|
| 1 | Security context on live bridge pod | 30 min | Critical — running as root now |
| 2 | Scope down RBAC | 2 hours | High — blast radius if bridge is compromised |
| 3 | Enable NetworkPolicy on workspace pods | 1 hour | High — tenant isolation |
| 4 | Rate limiting | 2–3 hours | High — prevents cost explosions |
| 5 | Helm history limit | 15 min | Medium — operational hygiene |
| 6 | Bridge NetworkPolicy | 30 min | Medium — defense in depth |
| 7 | Structured audit logging | 2 hours | Medium — needed for debugging at scale |
| 8 | Secret rotation process | 1 hour | Medium — incident response readiness |
| 9 | End-to-end TLS | 2 hours | Lower — not exposed to public network |

Fix 1 and 5 are quick wins — do them immediately.
Fix 2, 3, 4 are the meaningful security improvements.
Fix 6–9 are defense-in-depth for when you have real users.
