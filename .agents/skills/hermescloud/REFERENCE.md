# HermesCloud — Backend Contract & Reference

## Backend (hermes-client) requirements for multi-cluster

### 1. Neon DB — clusters table

```sql
CREATE TABLE clusters (
  cluster_id        TEXT PRIMARY KEY,           -- e.g. "hermes-test", "hermes-eu-2"
  bridge_url        TEXT NOT NULL,              -- http://<NODE_IP>:8080 (internal, never public)
  bridge_secret     TEXT NOT NULL,              -- X-Bridge-Secret value
  region            TEXT NOT NULL,              -- "eu" | "us" | "ap"
  plan_tier         TEXT NOT NULL DEFAULT 'shared', -- "shared" | "enterprise"
  status            TEXT NOT NULL DEFAULT 'active', -- "active" | "draining" | "offline"
  registered_at     TIMESTAMPTZ DEFAULT now(),
  updated_at        TIMESTAMPTZ DEFAULT now()
);
```

Bridge registers/updates itself on every deploy via:
```
POST https://api.hermeshq.net/api/admin/clusters
Authorization: Bearer <ADMIN_API_SECRET>
Content-Type: application/json

{
  "cluster_id":   "hermes-test",
  "bridge_url":   "http://178.104.185.60:8080",
  "bridge_secret": "...",
  "region":       "eu",
  "status":       "active"
}
```

---

### 2. CF KV — cluster stats cache

**Key:** `cluster-stats:{cluster_id}`
**TTL:** 60 seconds (lazy — refresh on read if stale, serve stale while refreshing)

**Value (JSON):** Full `ClusterSummaryResponse` from bridge:
```json
{
  "clusterId":           "hermes-test",
  "bridgeHealthy":       true,
  "kubernetesReachable": true,
  "nodeCount":           1,
  "instanceCount":       3,
  "pods": {
    "running":      3,
    "pending":      0,
    "failed":       0,
    "crashLooping": 0,
    "total":        3
  },
  "resources": {
    "reservedCpuM":      1500,
    "reservedMemoryMiB": 3072
  },
  "lastSeenAt": "2026-05-24T12:00:00Z"
}
```

**Routing algorithm** (implemented in CF Worker or Hono route handler):
```ts
async function pickCluster(userPlan: string, regionPref?: string) {
  const clusters = await db.query('SELECT * FROM clusters WHERE status = $1', ['active']);

  const candidates = await Promise.all(clusters.rows.map(async (c) => {
    const cached = await kv.get(`cluster-stats:${c.cluster_id}`, 'json');
    let stats = cached;
    if (!stats || isStale(stats.lastSeenAt, 60)) {
      // Refresh in background, serve stale for now
      refreshClusterStats(c).catch(() => {});
    }
    return { cluster: c, stats };
  }));

  return candidates
    .filter(({ stats }) =>
      stats?.kubernetesReachable &&
      stats?.pods.crashLooping === 0 &&
      !hasNodePressure(stats)
    )
    .filter(({ cluster }) =>
      planAllowed(cluster.plan_tier, userPlan)
    )
    .filter(({ cluster }) =>
      !regionPref || cluster.region === regionPref
    )
    .sort((a, b) =>
      (a.stats?.resources.reservedMemoryMiB ?? Infinity) -
      (b.stats?.resources.reservedMemoryMiB ?? Infinity)
    )[0]?.cluster ?? null;
}

function planAllowed(tier: string, userPlan: string) {
  if (tier === 'enterprise') return userPlan === 'enterprise';
  return userPlan !== 'enterprise'; // shared clusters serve free + pro
}

function hasNodePressure(stats: ClusterSummaryResponse) {
  // Future: parse node pressure from /v1/cluster/resources
  return false;
}

async function refreshClusterStats(cluster: ClusterRow) {
  const res = await fetch(`${cluster.bridge_url}/v1/cluster/summary`, {
    headers: { 'X-Bridge-Secret': cluster.bridge_secret },
  });
  const stats = await res.json();
  await kv.put(`cluster-stats:${cluster.cluster_id}`, JSON.stringify(stats), { expirationTtl: 120 });
}
```

---

### 3. Instance creation flow (async, CF Queues)

```
POST /api/instances
→ CF Worker receives, validates, picks cluster
→ Enqueue job to CF Queue: { userId, instanceId, clusterRow, instanceSpec }
→ Return 202: { instanceId, status: "queued" }

CF Queue consumer:
  1. POST cluster.bridge_url/v1/instances   (InstanceSpec body, X-Bridge-Secret header)
  2. Poll bridge GET /v1/instances/:id until phase == "running" or timeout
  3. Update DB instances table: status = active, cluster_id, bridge_url
  4. Optionally push WS or SSE event to user
```

**Loading state machine the frontend can display:**
```
queued → routing → provisioning → starting → healthy
```
Map bridge instance phases:
- `Pending` → "provisioning"
- `Running` (pods not all ready) → "starting"  
- `Running` (all containers ready) → "healthy"

---

### 4. Full InstanceSpec the backend must POST to bridge

```json
{
  "instanceId":      "ws-<16 hex chars>",
  "runtimeMode":     "runtime-node-core",
  "runtimePort":     8787,
  "plan":            "free | pro | enterprise",
  "clusterId":       "hermes-test",
  "image":           "ghcr.io/taalib25/runtime-node-core",
  "imageTag":        "latest",
  "imagePullPolicy": "Always",
  "hermesConfig": {
    "model": {
      "provider": "openrouter",
      "name": "anthropic/claude-3.5-sonnet"
    }
  },
  "secrets": {
    "OPENROUTER_API_KEY": "sk-or-..."
  },
  "network": {
    "subdomain": "ws-<hex>",
    "host":      "hermeshq.net",
    "scheme":    "https"
  }
}
```

Notes:
- `clusterId` must match the bridge's own `BRIDGE_CLUSTER_NAME` env var (bridge rejects mismatches)
- `imageTag: "latest"` → bridge auto-sets `imagePullPolicy: Always` if not specified
- `secrets` keys become k8s Secret entries injected as env vars
- `network.subdomain` drives the Traefik IngressRoute hostname — must be globally unique

---

### 5. Bridge API reference (all routes require `X-Bridge-Secret` header)

| Method | Path | Body / Params | Description |
|--------|------|---------------|-------------|
| `POST` | `/v1/instances` | InstanceSpec JSON | Create instance (Helm install) |
| `GET` | `/v1/instances` | — | List all instances on this cluster |
| `GET` | `/v1/instances/:id` | — | Get instance status + phase |
| `PUT` | `/v1/instances/:id` | InstanceSpec JSON | Update (Helm upgrade) |
| `DELETE` | `/v1/instances/:id` | — | Delete instance + namespace |
| `GET` | `/v1/cluster/summary` | — | Pod counts, resource load, k8s health |
| `GET` | `/v1/cluster/resources` | — | Per-node allocatable CPU/mem + pressure |
| `GET` | `/v1/instances/:id/diagnostics` | — | Pod logs, events, recommendation |
| `GET` | `/v1/instances/:id/profiles` | — | Hermes profile list (proxied) |
| `GET` | `/api/providers` | — | Configured LLM providers (proxied) |

**Instance status response fields** (from `GET /v1/instances/:id`):
```json
{
  "instanceId": "ws-abc123",
  "phase":      "Running | Pending | Failed | Unknown",
  "ready":      true,
  "podName":    "ws-abc123-hermes-instance-abc-xyz",
  "namespace":  "ws-abc123",
  "url":        "https://ws-abc123.hermeshq.net"
}
```

---

### 6. Admin cluster registration endpoint (backend must implement)

The bridge calls this on every deploy — backend must expose:

```
POST /api/admin/clusters
Authorization: Bearer <ADMIN_API_SECRET>

{
  "cluster_id":    "hermes-test",
  "bridge_url":    "http://178.104.185.60:8080",
  "bridge_secret": "...",
  "region":        "eu",
  "status":        "active"
}
```

Behavior: upsert into clusters table. `updated_at = now()`.

---

### 7. Instance DB table (backend must track)

```sql
CREATE TABLE instances (
  instance_id  TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL,
  cluster_id   TEXT NOT NULL REFERENCES clusters(cluster_id),
  status       TEXT NOT NULL DEFAULT 'queued',
  plan         TEXT NOT NULL,
  created_at   TIMESTAMPTZ DEFAULT now(),
  updated_at   TIMESTAMPTZ DEFAULT now()
);
```

The bridge URL is NOT stored in instances table — always looked up via `clusters.bridge_url`. This means if a cluster's IP changes, only the clusters table needs updating.

---

### 8. ForwardAuth (deferred — do not implement yet)

When `/api/client/auth/verify` is live on the backend:
- Bridge's Traefik IngressRoutes will use ForwardAuth middleware
- Currently patched OFF; instances are accessible without auth
- Re-enable via `PUT /v1/instances/:id` with `hermesConfig.forwardAuth: true` (future field)

---

### 9. Multi-cluster checklist for backend

- [ ] `POST /api/admin/clusters` upsert endpoint (bridge calls this on deploy)
- [ ] `GET /api/admin/clusters` list (admin dashboard)
- [ ] CF KV namespace bound in Worker for `cluster-stats:*` keys
- [ ] `pickCluster()` routing function (see §2)
- [ ] CF Queue consumer for async instance creation
- [ ] `POST /api/instances` → enqueue, return 202
- [ ] `GET /api/instances/:id/status` → poll bridge, return loading state
- [ ] instances table with `cluster_id` FK
