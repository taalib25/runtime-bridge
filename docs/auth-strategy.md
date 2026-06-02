# Authentication Strategy — Hermes Runtime Operator

## The Three Auth Layers

The system has three completely separate auth boundaries. Each protects a
different door and requires a different solution.

```
User (browser)
     │
     │  Layer 1 — WorkOS
     │  Who is this user? Are they logged in?
     ▼
Your Backend (API)
     │
     │  Layer 2 — Bridge Secret (M2M)
     │  Is this request from our backend or an attacker?
     ▼
Bridge
     │
     │  Layer 3 — ForwardAuth
     │  Does this user own this specific workspace?
     ▼
Workspace pod (tenant-xxx.hermeshq.net)
```

They guard three different doors. None of them replaces the others.

---

## Layer 1 — WorkOS (User → Backend)

WorkOS handles user-facing authentication — login, signup, session management,
and enterprise SSO. This is what your backend uses to know who the user is.

**What WorkOS gives you:**
- Login with Google, GitHub, email/password out of the box
- Enterprise SSO (SAML, OIDC) for business customers — one config per company,
  their employees log in with their company credentials automatically
- Directory Sync — user accounts stay in sync with the company's HR system
- JWTs issued per user session — your backend validates these on every request
- Roles and permissions — mark users as admin, free tier, pro, etc.

**What it does not do:**
- It does not protect server-to-server calls (that's the bridge secret)
- It does not know which workspace belongs to which user (that's your DB)

**Integration point:** your backend verifies the WorkOS JWT on every API call.
When a user creates a workspace, the backend associates `workspaceId` with
`userId` in its DB. WorkOS just tells you who the user is — your backend
decides what they're allowed to do.

---

## Layer 2 — Bridge Secret (Backend → Bridge)

The bridge secret is **machine-to-machine auth** — server talking to server.
Your backend calls the bridge the same way one microservice calls another.
WorkOS is the wrong tool here because WorkOS is designed for humans logging in,
not for automated backend services making API calls.

```
X-Bridge-Secret: <64-char hex>
```

Every request from your backend to the bridge must include this header.
The bridge validates it with `subtle.ConstantTimeCompare` (timing-safe).

**Why a shared secret is fine here:**
- The bridge is an internal service — not exposed to end users
- Only your backend knows the secret
- Cloudflare sits in front, so direct port scanning hits Cloudflare first
- The secret is stored as a Kubernetes Secret, never in code

**How to make it stronger (in order of effort):**

| Approach | Effort | What it adds |
|---|---|---|
| Rotate the secret regularly | 15 min | Limits exposure window if secret leaks |
| One secret per backend service | 1 hour | Revoke one without rotating all |
| mTLS (mutual TLS) | 1–2 days | No secret to steal — both sides present certs |

**mTLS** is the industry standard for internal service-to-service auth at scale.
Both the backend and bridge present TLS certificates. The bridge only accepts
connections from clients holding a cert it trusts. Even if an attacker intercepts
traffic, they can't connect without the client certificate.

Recommended path: keep the shared secret now, move to mTLS when you have
multiple backend instances or multiple clusters.

---

## Layer 3 — ForwardAuth (User → Workspace Pod)

Workspace pods are exposed at `tenant-xxx.hermeshq.net`. Traefik sits in front
and calls a ForwardAuth endpoint before allowing any request through. This is
how you ensure only the right user can access their workspace.

**The flow:**

```
User opens ws-abc123.hermeshq.net in browser
     ↓
Traefik intercepts the request
     ↓
Traefik calls: BRIDGE_FORWARD_AUTH_URL (your backend's verify endpoint)
     with the user's cookie / Authorization header forwarded
     ↓
Your backend checks:
  1. Is the WorkOS JWT valid and not expired?
  2. Does this user own ws-abc123? (query your DB)
     ↓
Yes → backend returns 200 → Traefik allows the request through
No  → backend returns 401 → Traefik returns 401 to the user
```

**This is where WorkOS JWTs flow into workspace security.** The user's WorkOS
session cookie is forwarded by Traefik to your backend's verify endpoint. Your
backend validates it with WorkOS, checks ownership in the DB, and replies. WorkOS
indirectly secures every workspace request without the bridge knowing anything
about users.

**Current status:** ForwardAuth is wired in the bridge but the backend verify
endpoint (`/api/client/auth/verify`) needs to be implemented. Until then,
ForwardAuth is disabled per workspace.

---

## How WorkOS Connects Everything

```
User logs in via WorkOS
     ↓
WorkOS issues JWT (contains userId, role, expiry)
     ↓
Backend stores: workspaces.user_id = userId for each workspace
     ↓
User opens their workspace URL
     ↓
Traefik → ForwardAuth → Backend verify endpoint
     ↓
Backend: decode WorkOS JWT → get userId → check workspaces table
     ↓
If userId owns this workspace → 200 (allow)
If not → 401 (block)
```

WorkOS handles the hard parts of user identity. Your DB handles the
workspace-to-user mapping. The bridge handles the Kubernetes operations.
Each layer does one thing.

---

## What to Build and When

### Now — WorkOS integration in the backend

If you haven't built user auth yet, use WorkOS. It eliminates weeks of work
(session management, password resets, OAuth flows, enterprise SSO).

Minimum integration:

```javascript
// On every backend API request
const session = await workos.userManagement.getSession({ sessionId });
if (!session) return res.status(401).json({ error: "unauthorized" });

const userId = session.user.id;
// now you know who the user is — check your DB for their workspace
```

### Soon — ForwardAuth verify endpoint

Once WorkOS is in your backend, implement the verify endpoint:

```javascript
// GET /api/client/auth/verify
// Called by Traefik's ForwardAuth for every workspace request
app.get('/api/client/auth/verify', async (req, res) => {
  // 1. Extract JWT from forwarded cookie or Authorization header
  const token = req.headers.authorization?.split(' ')[1]
               || req.cookies['session'];

  // 2. Validate with WorkOS
  const session = await workos.userManagement.getSession({ sessionId: token });
  if (!session) return res.status(401).end();

  // 3. Extract workspace ID from the Host header Traefik forwarded
  const host = req.headers['x-forwarded-host']; // "ws-abc123.hermeshq.net"
  const workspaceId = host.split('.')[0];        // "ws-abc123"

  // 4. Check ownership in your DB
  const workspace = await db.workspaces.findOne({
    where: { id: workspaceId, userId: session.user.id }
  });
  if (!workspace) return res.status(401).end();

  return res.status(200).end(); // Traefik allows the request
});
```

Then enable ForwardAuth per workspace by passing `forwardAuthURL` in the
bridge create request:

```json
POST /v1/instances/ws-abc123
{
  "forwardAuthURL": "https://api.hermeshq.net/api/client/auth/verify"
}
```

### Later — mTLS for bridge secret

When you have multiple backend services or multiple clusters, replace the
shared bridge secret with mutual TLS. Each backend service gets a client
certificate signed by your internal CA. The bridge only accepts connections
from clients presenting a valid cert.

---

## Summary

| Layer | Tool | Protects | Status |
|---|---|---|---|
| User login | WorkOS | Who the user is | Build this in backend |
| Backend → Bridge | Shared secret | M2M calls | Working, keep it |
| User → Workspace | ForwardAuth + WorkOS JWT | Workspace ownership | Endpoint needs building |
| Future M2M | mTLS | Bridge secret replacement | When you scale |
