# Integration Review — Hermes Multi-Cluster Platform

> **Date:** 2026-06-10
> **Scope:** `hermes-runtime-operator` (bridge, charts, infra, provisioning) and `hermes-hut` (backend, admin frontend).
> **Method:** Code-verified against both repos, not docs-trusted. Every contract claim below was checked against the actual route tables, handlers, and services.

---

## 1. Executive summary

**Direction is coherent.** The control-plane split is respected in code, not just docs: the backend never imports a Kubernetes client; all runtime mutation goes through `createBridgeClient` → HTTP → bridge; OpenTofu/kube-hetzner only creates clusters; instances are Helm releases created by the bridge. One bridge per cluster, addressed via `clusters.bridgeUrl` per row. No super-bridge.

**Aligned (verified, not assumed):**

- **Registration contract works end-to-end.** `provision-cluster.sh` POSTs `{cluster_id, bridge_url, bridge_secret, region, status}` to `POST /api/admin/clusters` with `Bearer $ADMIN_API_SECRET` — exactly what `requireAdminApiSecret` + `registerClusterFromApi` expect (upsert, AES-256-GCM-encrypts the bridge secret). The earlier 404 on `/internal/runtime-clusters/register` was a doc artifact — that route never existed and nothing calls it anymore.
- **Bridge `GET /version` exists** (unauthenticated root, returns `{version, build, clusterId, apiVersion}`) and the backend's `bridge.version.get()` fetches `${baseUrl}/version` raw — matched.
- **No stuck-creating-forever:** cron `*/5` runs `failStuckProvisioningInstances` (10-min bridge cutoff) and `drainStuckPendingOperations`; cron `*/2` heartbeats every cluster, degrades after missed polls, reconciles `capacityStatus` ok↔full from headroom minus live reservations.
- **All five admin/support phases** (operations API+page, version surfacing, maintenance UI, classifier, redaction, triage search) are implemented and TypeScript-clean — uncommitted in the hermes-hut working tree at review time.

**Broken or risky:**

1. **Diagnostics path leaks unredacted logs** — `getAdminInstanceDiagnostics` returns the bridge diagnostics payload verbatim (`diagnostics` and a duplicate in `issue.raw`). Bridge diagnostics embed pod log excerpts; only the dedicated `/logs` route redacts. (See §6.)
2. **Bridge maintenance mode is in-memory** (`b.maintenance` atomic) — a bridge pod restart silently disables maintenance. No persistence.
3. **Two "drain" concepts collide:** backend `capacityStatus=draining` (safe, stops scheduling) vs bridge `POST /v1/cluster/drain` (**deletes all managed instances**). The backend exposes both; only the safe one is in the UI, but the destructive one is one admin API call away with no confirmation token.
4. Maintenance response-shape mismatch (`.enabled` vs `.maintenance`) and stale `routeTree.gen.ts` were caught and fixed during this review session — but they show the new UI paths have **never run in a browser**.

**Biggest blocker right now:** nothing is hard-blocked. The critical path is: commit/PR the hermes-hut admin work, then close the diagnostics-redaction hole before anyone pastes a diagnostics JSON into a support ticket.

---

## 2. Integration contract review

| Feature | Frontend expects | Backend exposes | Backend calls bridge | Bridge exposes | Status | Fix |
|---|---|---|---|---|---|---|
| Cluster registration | n/a (CI-only) | `POST /api/admin/clusters` (Bearer `ADMIN_API_SECRET`) | — | — | ✅ | Update docs still naming `BACKEND_REGISTER_TOKEN` / `/internal/runtime-clusters/register` |
| Cluster list/detail | `clusters`, `cluster(id)` + `bridgeHealth`, `instanceCount` | `GET /clusters`, `GET /clusters/:id` | heartbeat `GET /v1/cluster/summary` | ✅ | ✅ | — |
| Bridge version | `{version, available}` | `GET /clusters/:id/version` | raw `GET ${baseUrl}/version` (no `/v1`, no secret) | `GET /version` unauthenticated | ✅ | — |
| Drain provisioning (backend) | enable/disable buttons | `POST …/disable-provisioning`, `enable-provisioning` | none (DB-only `capacityStatus`) | n/a | ✅ | — |
| Bridge maintenance mode | `{maintenance}` (fixed from `.enabled`) | `GET/PUT /clusters/:id/maintenance` | `GET/PUT /v1/cluster/maintenance` | `{clusterId, maintenance, updatedAt}`; PUT body `{enabled}` | ✅ after fix | Persist mode on bridge (ConfigMap); not yet browser-tested |
| Bridge drain (destructive) | not surfaced (good) | `POST /clusters/:id/drain` | `POST /v1/cluster/drain {purge}` | deletes all instances, 409 if draining | ⚠️ | Rename/guard; require explicit `confirm: clusterId` in body |
| Instance create | user flow via workspaces | queue producer → consumer | `POST /v1/instances/{id}` (202 + op) | ✅ async op | ✅ | — |
| Restart / repair / redeploy / rollback / upgrade | admin action buttons | `POST /instances/:id/<action>` | matching `/v1/instances/{id}/<action>` | ✅ all 5 | ✅ | — |
| Config update | n/a admin | workspace routes | `GET/PUT /v1/instances/{id}/config` | ✅ | ✅ | — |
| Secrets update | n/a | provider key flows | `POST/PUT /v1/instances/{id}/config/providers[/{name}]` | ✅ (no dedicated `/secrets`) | ✅ by convention | The doc'd `PATCH /secrets` is a phantom gap — providers endpoints already carry keys; only add it if non-provider instance secrets emerge |
| Diagnostics | `{diagnostics, issue, bridgeUnavailable}` | `GET /instances/:id/diagnostics` + classifier | `GET /v1/instances/{id}/diagnostics` | ✅ | ⚠️ | **Redact diagnostics payload; drop duplicate `issue.raw`** |
| Logs / events | redacted lines | `GET /instances/:id/logs` (redacted), events | `/v1/instances/{id}/logs`, `/events` | ✅ | ✅ | — |
| Operations list/detail | filters, pagination, drawer | `GET /operations`, `GET /operations/:id` | n/a (DB: `instanceOperations` ⋈ `instances`) | `GET /v1/operations/{id}` (used by consumer) | ✅ new | Browser-verify; empty state untested |
| Support search | grouped results | `GET /support/search?q=` | n/a (DB) | n/a | ✅ new | Op result links don't carry the op id into the operations page (no search param defined) |
| Terminal | out of scope | none | `terminal/recreate` only | WS `/v1/instances/{id}/exec` + single-use 2-min exec tokens (minted in `lifecycle.go`) | ⏸ deliberate | Keep deferred; fix `CheckOrigin: true` per existing hardening plan |

---

## 3. End-to-end flow review

### Provisioning → bridge deploy → registration

Operator repo is mid-migration: old hetzner-k3s CLI configs and per-cluster deploy workflows (`hermes-eu-1/2`) are deleted on `feat/p5-provision-eu-clusters`; the new path is `create-microos-snapshot.yml` → `provision-runtime-cluster.yml` → `_deploy-bridge.yml` plus `infra/kube-hetzner/` (module + `kh-test`). The script smoke-tests `/healthz`, `/readyz`, `/v1/cluster/summary` on-node, registers with the backend, then runs `validate-cluster.sh`. Sound sequence.

Two caveats:

- The entire `docs/multi-cluster-revamp/` and `validate-cluster.sh` are **untracked** — the contract source of truth isn't committed.
- When `CF_DNS=false` the registered `bridge_url` is `http://NODE_IP:8080`, after which every backend call sends `X-Bridge-Secret` in cleartext (the script warns, but nothing enforces follow-up).

### Instance create → selection → bridge → tracking

Create enqueues to `INSTANCE_PROVISION_QUEUE`; placement goes through `pickAndReserve` (atomic headroom check + `clusterReservations` row) while `pickProvisioningCluster` is only an availability probe sharing the same eligibility filter — good anti-drift design. The bridge returns 202 + operation; `instance-bridge-op.consumer` polls `GET /v1/operations/{id}` and stores `bridgePayload.bridgeOperationId`. Reapers cover both stuck instances and stuck pending ops. This flow is the most mature part of the system.

### Diagnostics → classifier → redaction

Classifier ordering is sensible (bridge-unreachable → CrashLoop → ImagePull → capacity → PVC → provider auth → integration → health → timeout → fallback). Two weaknesses:

- The corpus-flattening approach means any diagnostics that merely *mention* telegram/slack/discord plus the word "failed" classify as `integration` (over-matching).
- The `diagnostics`/`issue.raw` payload bypasses redaction entirely (the headline security finding, §6).

### Admin actions → operation records → visible timeline

Backend lifecycle routes create `instanceOperations` rows; the operations page joins through `instances` for cluster scoping and the drawer shows timeline, bridge op id, and redacted payload. Coherent. Gap: ops whose instance row is hard-deleted would vanish from the admin list (inner join), though instances soft-delete so this is theoretical today.

---

## 4. Backend control-plane gaps

| Area | State |
|---|---|
| Runtime cluster registry | ✅ Real: upsert registration, encrypted secret, heartbeat, missed-poll degradation, admin events on degrade/recover |
| Bridge URL per cluster | ✅ `clusters.bridgeUrl` + `toBridgeRequestConfig` (throws on `disabled`) |
| Status/capabilities | ✅ `status`, `capacityStatus`, `supportedRuntimeClasses` (hardcoded `["runc"]` at registration — fine until KubeVirt) |
| Cluster selection | ✅ Region-preferred → default → least-reserved; atomic reserve in scheduler |
| Capacity reservation | ✅ `clusterReservations` + headroom reconcile; null headroom (old bridge) treated as unknown, not excluded |
| Operation tracking | ✅ DB records + bridge op polling |
| Timeout/retry | ✅ queue `max_retries: 5`; crons reap stuck instances (10 min) and pending ops |
| Support/admin APIs | ✅ as of the (uncommitted) 5-phase admin work |
| Residual | `ensureDefaultClusterSeed()` runs on every `listClusters`/`getCluster` — legacy single-cluster crutch; remove once kh-test registers itself. `DEFAULT_CLUSTER_ID=hermes-test` + `DEFAULT_BRIDGE_URL` in wrangler vars are the same crutch. |

---

## 5. Admin/support gaps

All six previously-missing items are now **implemented but uncommitted and browser-unverified**. Remaining, in priority order:

1. **Commit + PR the work** — it only exists in the working tree.
2. **Browser smoke test** — maintenance tab, version tab, operations list (incl. empty state), support search; the two bugs already found in static review (`.enabled`, stale route tree) suggest more may surface at runtime.
3. **Regenerate `routeTree.gen.ts` via the TanStack generator** (run the dev server) — it was hand-edited; the generator must confirm parity or it will silently rewrite it.
4. Classifier unit tests (fixtures per pod state) — heuristics over a flattened corpus deserve regression coverage.
5. Operation deep-links: define a validated search param on `/dashboard/operations` so support-search results can open the exact operation.
6. Support search indexes (trigram/expression on `users.email`, `instances.bridgeWorkspaceId`, `bridgePayload->>'bridgeOperationId'`) — defer until scale demands.

---

## 6. Security/secrets review

- **Token separation: real, but misnamed in docs.** Registration uses `ADMIN_API_SECRET` (Bearer, only on `POST /clusters`); per-cluster `BRIDGE_SECRET` is stored AES-256-GCM-encrypted and decrypted only at call time; admin reads/actions use Clerk identity via `requirePlatformAdmin` (`ADMIN_USER_IDS`). The doc'd `BACKEND_REGISTER_TOKEN` doesn't exist — align the docs, not the code. JWT bridge auth remains future work.
- **`SECRETS_ENCRYPTION_KEY`**: required (falls back to `ENCRYPTION_KEY`, throws if absent); key derived via SHA-256, GCM with random IV + auth tag. Sound. One key for everything — rotation story is "re-register clusters", acceptable for now.
- **Redaction gap (the finding):** `GET /instances/:id/diagnostics` returns bridge diagnostics — which per the bridge spec include log excerpts — **unredacted**, twice (`diagnostics` and `issue.raw`). Fix: run the diagnostics payload through redaction (deep-walk strings with `redactText`) and make `issue.raw` a reference or drop it.
- **Bridge `SecretMeta` lists key names only** — enforced bridge-side. Frontend never receives secret values; the admin API client holds only the Clerk token. No frontend exposure found.
- **WS exec `CheckOrigin: return true`** — any origin can attempt the upgrade (still gated by single-use 2-min exec tokens, which caps the blast radius). The existing `instance-security-hardening` plan covers this plus the missing NetworkPolicy/ResourceQuota in `runtime-node-core` (bridge RBAC for those was already granted in PR #26). That plan is the right next operator-repo PR.
- **HTTP bridge URL fallback** at registration (above) — cleartext `X-Bridge-Secret` until DNS/TLS exists.

---

## 7. Cloudflare deployment risks

The backend is **already Workers-native** — lower-risk than assumed:

- ✅ `nodejs_compat` covers `node:crypto`/`Buffer` (crypto.ts); Hyperdrive binding for Postgres with per-request `runWithRequestDb`; queue consumer and both crons (`*/2`, `*/5`) wired in `worker.ts`; module-level `appPromise` caching is correct for isolates.
- ⚠️ `ADMIN_FRONTEND_URL: "http://localhost:3001"` in **production** wrangler vars — admin-app CORS/redirects will break or open up oddly when the admin app deploys.
- ⚠️ Bridge calls from heartbeat fan out per cluster per 2 min — fine at 2–3 clusters; watch sub-request limits later.
- ⚠️ Migrations run outside the worker (drizzle CLI) — make sure the deploy pipeline orders migrate-before-deploy; nothing in-repo enforces it.
- ✅ Terminal/WS deliberately not proxied through the Worker (browser→bridge direct with exec token) — correct for Workers' WS constraints.
- ℹ️ `@types/pg` missing is build-noise only; bun/IDE module-resolution diagnostics are not real errors (`tsc` is clean).

---

## 8. Risk register

| Risk | Severity | Area | Why it matters | Recommended fix | Priority |
|---|---|---|---|---|---|
| Diagnostics payload unredacted (`diagnostics` + `issue.raw`) | **High** | Backend/admin | Pod-log secrets reach support UIs and copy-pastes | Deep-redact diagnostics; dedupe `raw` | P0 |
| Bridge maintenance mode lost on pod restart | High | Bridge | Cluster silently re-enters service mid-incident | Persist to ConfigMap/annotation, restore on boot | P0/P1 |
| Destructive `POST /clusters/:id/drain` one call from safe "draining" | High | Backend API | Name collision invites a catastrophic mis-call | Require `{confirm: "<clusterId>"}`; consider renaming to `/purge` | P1 |
| 5-phase admin work uncommitted + browser-unverified | Medium | Frontend/backend | Two contract bugs already found statically; more likely at runtime | Commit, run dev servers, regenerate route tree, smoke test | P0 (process) |
| HTTP bridge URL registration fallback | Medium | Provisioning | Cleartext bridge secret on every backend call | Block registration of `http://` unless `ALLOW_INSECURE=1`; re-register after DNS | P1 |
| WS exec accepts any Origin | Medium | Bridge | Cross-origin upgrade attempts (token-gated) | Existing hardening plan, change 1 | P1 |
| No NetworkPolicy/ResourceQuota in runtime-node-core | Medium | Charts | Instance pods can reach each other/cluster services | Existing hardening plan, changes 2–3 | P1 |
| Revamp docs + validate-cluster.sh untracked | Medium | Operator repo | Contract source of truth can be lost; docs still cite dead endpoint/token names | Commit; fix `BACKEND_REGISTER_TOKEN`/`/internal/...` references | P1 |
| Classifier integration over-match | Low | Backend | Mislabels issues when diagnostics mention messaging platforms | Scope match to integration-status fields; add fixtures | P2 |
| `ADMIN_FRONTEND_URL` localhost in prod vars | Low | Cloudflare | Broken admin CORS/redirects at deploy | Set real URL before admin app ships | P2 |
| Default-cluster seed crutch | Low | Backend | Masks registration failures; extra query per call | Remove after kh-test self-registers | P3 |

---

## 9. Course correction plan

- **Phase 0 — Land and verify what exists (now):** commit hermes-hut admin/support branch; regenerate route tree via dev server; browser-smoke all new pages against kh-test; commit `docs/multi-cluster-revamp/` + `validate-cluster.sh` in the operator repo and correct the dead `BACKEND_REGISTER_TOKEN`/`/internal/runtime-clusters/register` references.
- **Phase 1 — Registration hardening:** reject `http://` bridge URLs by default; re-register kh-test over HTTPS; delete the default-cluster seed once registration is the only path.
- **Phase 2 — Operation tracking polish:** operation deep-link search param; classifier test fixtures; surface stuck-op reaper outcomes as admin events.
- **Phase 3 — Admin/support visibility:** done modulo Phase 0 verification; then bridge-maintenance persistence so the UI toggle is trustworthy.
- **Phase 4 — Diagnostics/redaction/secrets hardening:** diagnostics deep-redaction (first PR, §10); drain-endpoint confirmation guard; operator-repo security plan (WS origin, NetworkPolicy, quota).
- **Phase 5 — Cloudflare readiness:** fix prod vars (`ADMIN_FRONTEND_URL`); document migrate-before-deploy; load-sanity-check heartbeat fan-out; only then consider JWT bridge auth.

---

## 10. First PR recommendation

*(After committing the in-flight admin/support branch, which is finished work, not a new PR.)*

**PR title:** `fix(admin): redact diagnostics payloads and deduplicate classifier raw output`

**Scope:** Close the one live security gap in already-written code. Add a deep redaction walker that applies `redactText` to every string in a JSON value (plus key-name redaction via the existing `redactBridgePayload`), apply it to the diagnostics payload in `getAdminInstanceDiagnostics`, and stop embedding a second unredacted copy in `issue.raw` (redact it or replace with a short fingerprint).

**Files likely to change:**

- `apps/server/src/lib/redact.ts` — add `redactDeep(value: unknown): unknown`
- `apps/server/src/modules/admin/admin-instance.service.ts` — wrap `diagnostics` before return
- `apps/server/src/modules/admin/issue-classifier.ts` — classify on the raw payload, return redacted `raw`
- `apps/server/src/lib/redact.test.ts` (new) — pattern fixtures: `sk-` keys, bearer/JWT, bot tokens, DB URLs, PEM, nested objects/arrays

**Endpoints/schema changes:** none — same response shape, redacted content. No bridge changes, no DB changes.

**Validation:** unit tests on `redactDeep`; manual `GET /api/admin/instances/:id/diagnostics` against a kh-test instance whose logs contain a planted fake `sk-...` key, confirming placeholder output in both `diagnostics` and `issue.raw`; confirm classifier verdict unchanged on the same fixtures (classification reads pre-redaction text).

**What not to include:** bridge maintenance persistence (operator-repo PR), drain confirmation guard, classifier heuristic changes, search indexes, terminal/origin work, any Cloudflare config — each is a separate small PR per the phase plan.
