# Backend API Guide for Hermes Workspaces

## Connection Architecture

Do **not** proxy agent traffic through the backend — it wastes bandwidth and adds latency. Use credential delegation instead:

```
1. Frontend authenticates with your backend (JWT / session)
2. Backend returns workspace URL + API key
3. Frontend talks to the agent directly

Frontend ──── POST /auth/workspace ────► Backend (your API)
         ◄─── {url, api_key} ──────────
Frontend ──── direct ─────────────────► https://tenant-xxx.hermeshq.net
                                              (uses Hetzner bandwidth)
```

**What the backend exposes to the frontend:**

```json
{
  "workspaceUrl": "https://tenant-a3f9kx2m.hermeshq.net",
  "apiKey": "<API_SERVER_KEY>",
  "model": "hermes-agent"
}
```

**What the frontend does with it (OpenAI-compatible SDK):**

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "https://tenant-a3f9kx2m.hermeshq.net/v1",
  apiKey: apiKey,
  dangerouslyAllowBrowser: true,  // direct from browser
});

const stream = await client.chat.completions.create({
  model: "hermes-agent",
  messages: [{ role: "user", content: "hello" }],
  stream: true,
});
```

**What the backend stores per user (in DB):**

| Field | Description |
|-------|-------------|
| `workspace_id` | `tenant-a3f9kx2m` — used for bridge lifecycle calls |
| `api_server_key` | From bridge create response `secrets.API_SERVER_KEY` |
| `workspace_url` | `https://{workspace_id}.hermeshq.net` |
| `secrets` | `ANTHROPIC_API_KEY` etc — re-sent on every bridge update |
| `config` | Agent config — re-sent on every bridge update |

---

## Workspace ID Convention

Workspace IDs must follow the format: **`tenant-{8 random alphanumeric chars}`**

Examples: `tenant-a3f9kx2m`, `tenant-7bqr1nzp`

Generate on the backend before calling the bridge:
```python
import secrets, string
def new_workspace_id():
    chars = string.ascii_lowercase + string.digits
    return "tenant-" + "".join(secrets.choice(chars) for _ in range(8))
# → "tenant-a3f9kx2m"
```
```javascript
const newWorkspaceId = () =>
  "tenant-" + crypto.randomBytes(4).toString("hex"); // → "tenant-a3f9kx2m"
```

This ID is used as:
- The bridge workspace ID (`/v1/workspaces/{id}`)
- The Helm release name
- The Kubernetes namespace
- The subdomain (`{id}.hermeshq.net`)

Store it in your database tied to the tenant when you create the workspace.

---

## Simplified Backend API

### Create Workspace

**Endpoint:** `POST /workspaces`

**Request body (minimal):**
```json
{
  "tenantId": "tenant-a3f9kx2m",
  "plan": "medium",
  "config": {
    "model": {
      "default": "anthropic/claude-opus-4.6"
    },
    "agent": {
      "max_turns": 90
    }
  }
}
```

**Required fields:**
- `tenantId` (string): Generated workspace ID in `tenant-{random}` format
- `plan` (string): `small`, `medium`, `large` — controls resources and storage

**Optional fields:**
- `config` (object): Hermes config.yaml overrides (merged with defaults)
- `storageGb` (number): Override default storage size (default: 10 GB)

**Response:**
```json
{
  "workspaceId": "tenant-a3f9kx2m",
  "status": "creating",
  "url": "tenant-a3f9kx2m.hermeshq.net",
  "operationId": "op-uuid"
}
```

---

### Get Workspace Status

**Endpoint:** `GET /workspaces/{tenantId}`

**Response:**
```json
{
  "workspaceId": "tenant-001",
  "status": "ready",
  "healthy": true,
  "url": "tenant-001.hermeshq.net",
  "replicas": 1,
  "readyReplicas": 1,
  "createdAt": "2026-04-21T01:00:00Z",
  "lastCheckedAt": "2026-04-21T01:05:00Z"
}
```

---

### Update Workspace

**Endpoint:** `PUT /workspaces/{tenantId}`

**Request body:**
```json
{
  "plan": "large",
  "config": {
    "agent": {
      "max_turns": 120
    }
  }
}
```

**Note:** Updates trigger Helm upgrade (zero-downtime if possible).

---

### Delete Workspace

**Endpoint:** `DELETE /workspaces/{tenantId}`

**Response:** `204 No Content`

---

## Plan Specifications

Backends should only reference these tiers; infrastructure mapping is server-side:

| Plan | CPU | Memory | Storage |
|------|-----|--------|---------|
| **small** | 250m | 512Mi | 5Gi |
| **medium** | 500m | 1Gi | 10Gi |
| **large** | 1000m | 2Gi | 50Gi |

---

## Backend → Bridge Translation

Your backend receives minimal product data, then transforms it into bridge API calls:

### Example: Create Workspace

**Backend receives:**
```json
{
  "tenantId": "tenant-a3f9kx2m",
  "plan": "medium",
  "config": {
    "model": {
      "default": "anthropic/claude-opus-4.6"
    }
  }
}
```

**Backend transforms to bridge call:**
```bash
curl -X POST https://bridge.hermeshq.net/v1/workspaces/tenant-a3f9kx2m \
  -H "X-Bridge-Secret: {secret}" \
  -H "Content-Type: application/json" \
  -d '{
    "workspaceId": "tenant-a3f9kx2m",
    "tenantId": "tenant-a3f9kx2m",
    "image": "nousresearch/hermes-agent",
    "imageTag": "latest",
    "namespace": "tenant-a3f9kx2m",
    "resources": {
      "cpuRequest": "500m",
      "memoryRequest": "1Gi"
    },
    "storage": {
      "enabled": true,
      "size": "10Gi"
    },
    "network": {
      "host": "hermeshq.net",
      "subdomain": "tenant-a3f9kx2m"
    },
    "ingressEnabled": true,
    "createNamespace": true,
    "config": {
      "model": {
        "default": "anthropic/claude-opus-4.6"
      }
    }
  }'
```

**Key abstractions:**
- ✓ Image always latest
- ✓ Ingress always enabled
- ✓ Storage class auto-selected (local-path for test, configurable in prod)
- ✓ Namespace auto-generated from tenant ID
- ✓ Health checks always active
- ✓ CPU/Memory mapped from `plan` tier
- ✓ Config merged with Hermes defaults

---

## Implementation Checklist for Backend

- [ ] Accept minimal workspace creation requests (tenantId + plan)
- [ ] Map plan tier → CPU/memory/storage resources
- [ ] Transform product-level config into bridge WorkspaceSpec
- [ ] Call bridge API with X-Bridge-Secret header
- [ ] Parse bridge response (status, URL, operationId)
- [ ] Poll `/v1/workspaces/{id}/status` to track creation
- [ ] Return workspace URL to frontend once `status: ready`
- [ ] Handle update requests → call bridge PUT with new plan/config
- [ ] Handle delete requests → call bridge DELETE

---

## Config Merging Strategy

Backend can send partial `config` overrides. These merge with Hermes defaults:

**Default config** (in bridge deployment):
```yaml
model:
  default: anthropic/claude-opus-4.6
  provider: auto
agent:
  max_turns: 90
  gateway_timeout: 1800
```

**Backend override:**
```json
{
  "config": {
    "agent": {
      "max_turns": 120
    }
  }
}
```

**Final config** (deep merge):
```yaml
model:
  default: anthropic/claude-opus-4.6  # kept from default
  provider: auto                        # kept from default
agent:
  max_turns: 120                        # overridden
  gateway_timeout: 1800                # kept from default
```

---

## Error Handling

Bridge returns:
- `400 Bad Request` — invalid workspace spec
- `401 Unauthorized` — missing/wrong X-Bridge-Secret header
- `404 Not Found` — workspace doesn't exist
- `409 Conflict` — workspace already exists
- `500 Internal Server Error` — Helm/cluster failure

Backend should:
1. Retry 5xx errors (Helm operations can be transient)
2. Log 4xx errors for investigation
3. Expose operation status to frontend during long-running creates
4. Cache workspace URLs once created

---

## Polling for Creation

Creation is async. Backend should poll:

```bash
curl http://bridge.hermeshq.net/v1/workspaces/{tenantId}/status \
  -H "X-Bridge-Secret: {secret}"
```

Poll every 2-5 seconds until:
- `status: ready` + `healthy: true` → Done, return URL to frontend
- `status: failed` → Error, show user message
- Timeout after 10 minutes → Log alert, show user "still creating"

---

## Example: Backend Service Pseudocode

```python
class HermesWorkspaceManager:
    def __init__(self, bridge_url, bridge_secret):
        self.bridge_url = bridge_url
        self.bridge_secret = bridge_secret
        self.plans = {
            "small": {"cpu": "250m", "memory": "512Mi", "storage": "5Gi"},
            "medium": {"cpu": "500m", "memory": "1Gi", "storage": "10Gi"},
            "large": {"cpu": "1000m", "memory": "2Gi", "storage": "50Gi"},
        }
    
    def create_workspace(self, plan, config_overrides=None):
        # Generate unique workspace ID
        import secrets, string
        chars = string.ascii_lowercase + string.digits
        workspace_id = "tenant-" + "".join(secrets.choice(chars) for _ in range(8))

        plan_spec = self.plans[plan]
        
        # Build workspace spec
        spec = {
            "workspaceId": workspace_id,
            "tenantId": workspace_id,
            "image": "nousresearch/hermes-agent",
            "imageTag": "latest",
            "namespace": tenant_id,
            "resources": {
                "cpuRequest": plan_spec["cpu"],
                "memoryRequest": plan_spec["memory"],
            },
            "storage": {
                "enabled": True,
                "size": plan_spec["storage"],
            },
            "network": {
                "subdomain": workspace_id,
                "host": "hermeshq.net",
            },
            "ingressEnabled": True,
            "config": config_overrides or {},
        }
        
        # POST to bridge
        response = requests.post(
            f"{self.bridge_url}/v1/workspaces/{workspace_id}",
            json=spec,
            headers={"X-Bridge-Secret": self.bridge_secret},
        )
        return response.json()  # store workspaceId in your DB tied to user
    
    def get_status(self, tenant_id):
        response = requests.get(
            f"{self.bridge_url}/v1/workspaces/{tenant_id}",
            headers={"X-Bridge-Secret": self.bridge_secret},
        )
        return response.json()
    
    def update_workspace(self, tenant_id, plan, config_overrides=None):
        plan_spec = self.plans[plan]
        spec = {
            "workspaceId": tenant_id,
            "resources": {
                "cpuRequest": plan_spec["cpu"],
                "memoryRequest": plan_spec["memory"],
            },
            "storage": {
                "size": plan_spec["storage"],
            },
            "config": config_overrides or {},
        }
        response = requests.put(
            f"{self.bridge_url}/v1/workspaces/{tenant_id}",
            json={"spec": spec},
            headers={"X-Bridge-Secret": self.bridge_secret},
        )
        return response.json()
    
    def delete_workspace(self, tenant_id):
        requests.delete(
            f"{self.bridge_url}/v1/workspaces/{tenant_id}",
            headers={"X-Bridge-Secret": self.bridge_secret},
        )
```

---

## Testing

Backend can test with:

```bash
SECRET="{bridge-secret}"
WID="tenant-$(openssl rand -hex 4)"   # e.g. tenant-a3f9kx2m

# Create
curl -X POST https://bridge.hermeshq.net/v1/workspaces/$WID \
  -H "X-Bridge-Secret: $SECRET" \
  -H "Content-Type: application/json" \
  -d "{\"workspaceId\":\"$WID\",\"tenantId\":\"$WID\",\"image\":\"nousresearch/hermes-agent\",\"imageTag\":\"latest\",\"namespace\":\"$WID\",\"ingressEnabled\":true,\"createNamespace\":true,\"network\":{\"subdomain\":\"$WID\",\"host\":\"hermeshq.net\"}}"

# Check status (poll until phase=ready)
curl https://bridge.hermeshq.net/v1/workspaces/$WID \
  -H "X-Bridge-Secret: $SECRET"

# Access workspace
curl https://$WID.hermeshq.net

# Delete
curl -X DELETE https://bridge.hermeshq.net/v1/workspaces/$WID \
  -H "X-Bridge-Secret: $SECRET"
```
