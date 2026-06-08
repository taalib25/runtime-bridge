# Implementation Plan — Migrate per-tenant runtime: opencode webui → NousResearch Hermes Agent

Status: **in progress** · Owner: bridge/platform · Date: 2026-06-06

## Implementation status (updated 2026-06-06 — official `latest` image, dashboard model)
Implemented in the working tree (uncommitted on `feat/hermes-agent-runtime`). `helm lint`
clean, `go test -race ./bridge/...` green (108 tests), chart renders the full 9-resource set
against the **exact** values the bridge emits (fixture: `bridge/zz_dumpvalues_test.go` with
`BRIDGE_DUMP_PATH=...`).

### Ground truth from the PUBLISHED image (inspected via docker, 2026-06-06) — authoritative
`nousresearch/hermes-agent:0.8.0` **does not exist** on Docker Hub — published tags are CalVer
(`v2026.4.3` … `v2026.6.5`) + `latest`/`main`. We use **`latest`**. The published image **uses
s6-overlay** (the local source clone was stale and misled an earlier draft of this section that
claimed "tini, no s6" — that was WRONG; verified against the real image + Nous docker docs):
- Real ENTRYPOINT is **`/init` (s6-overlay)** + `main-wrapper.sh`; `docker/entrypoint.sh` is a
  deprecated shim. s6 supervises the CMD **`gateway run`** (a `sleep infinity` heartbeat that keeps
  the container alive while s6 manages the gateway) **AND** the **`dashboard`** s6 service.
- The **`dashboard` s6 service** is gated by **`HERMES_DASHBOARD`** (1/true/yes → up; else the slot
  stays down). When on it runs `hermes dashboard --host ${HERMES_DASHBOARD_HOST:-0.0.0.0}
  --port ${HERMES_DASHBOARD_PORT:-9119} --no-open`, dropping to the `hermes` user via `s6-setuidgid`.
- Auth gate: on a non-loopback bind it **fails closed** unless a `dashboard_auth` provider is
  registered. Providers: **`basic`** (`HERMES_DASHBOARD_BASIC_AUTH_USERNAME`+`_PASSWORD`(/`_HASH`)+`_SECRET`),
  `nous` (OAuth, `HERMES_DASHBOARD_OAUTH_CLIENT_ID`), `self_hosted` (OIDC). We use **basic**.
  `HERMES_DASHBOARD_INSECURE=1` disables the gate (don't, unless behind ForwardAuth).
- `User: root`, `HERMES_HOME=/opt/data`, runtime `PATH=/opt/hermes/bin:/opt/hermes/.venv/bin:/opt/data/.local/bin:…`.

### How the runtime is wired (final — matches Nous docker docs)
- **`command: []`, `args: [gateway, run]`** — leaves the s6 `/init` ENTRYPOINT intact; the dashboard
  comes up purely from `HERMES_DASHBOARD=1`. Do NOT override beyond this.
- env: `HERMES_DASHBOARD=1`, `HERMES_DASHBOARD_BASIC_AUTH_USERNAME` (+ `_PASSWORD`/`_SECRET` via the
  instance Secret), `HERMES_UID/GID=1000` (s6 remap, == fsGroup). Host/port default to 0.0.0.0:9119.
- securityContext: **root start is REQUIRED** (s6 preinit + `s6-setuidgid` drop) — `runAsUser:0`,
  `allowPrivilegeEscalation:true`, `readOnlyRootFilesystem:false`, caps not fully dropped. This is the
  original Hard Constraint #1 and it is **valid for the published image**.
- **Routing:** 9119 (dashboard) is the single routed surface on the main host (RuntimePort=9119);
  no separate `dash-<id>` route, no exposed 8642 (the gateway heartbeat runs in-pod; the OpenAI API
  server stays off unless `API_SERVER_ENABLED` is added later for the remote-gateway path).
- **Dev-tool persistence (native):** s6 sets subprocess `HOME=/opt/data/home` (on the PVC), so
  cargo (`~/.cargo`), go (`~/go`), `pip --user` (`~/.local`), gem, pnpm, yarn, git/ssh/gh all persist
  with **no extra config** (confirmed by Nous docker docs). The chart adds `NPM_CONFIG_PREFIX=/opt/data/.local`
  for `npm -g`. NetworkPolicy egress allows all public registries (only cluster-internal + `169.254/16`
  blocked) → installs are never network-blocked.

### Retained
- **NetworkPolicy** on-by-default (ingress kube-system + hermes-bridge; egress DNS + internet `except`
  cluster CIDRs **+ `169.254.0.0/16`**). **`commonAnnotations`** on every resource. **Probes** TCP 9119.
  **ResourceQuota/LimitRange** on. Image **`latest`**.

### TODO
- Remove the legacy `charts/runtime-node-core` (opencode webui) chart once the hermes-agent runtime is
  proven on-cluster — kept for now as the rollback path.
- On-cluster smoke: dashboard stays up + basic-auth gate works + chat drives the agent; then pin a
  CalVer tag + digest. Hardening: try rootless/cap-drop (needs the smoke first).

This plan is end-to-end and assumes no prior context. It swaps the engine running inside
each per-tenant instance from the **opencode** webui (current `runtime-node-core` chart,
`provider: opencode-go`, port 8787) to the **real NousResearch Hermes Agent**
(`nousresearch/hermes-agent`), exposing Hermes's own **dashboard (:9119)** as the tenant UI
and its **OpenAI-compatible API server (:8642)** for the future local-desktop→cloud
remote-gateway path. Bridge stays the orchestrator (one Helm release per tenant namespace).

## Decisions locked (see memory `project_hermes_upstream_runtime`)
- Engine: `nousresearch/hermes-agent` (official image, s6-overlay PID 1 runs agent **and**
  dashboard in **one container** — no sidecar).
- Auth: **no WorkOS/OIDC**. Dashboard = **basic-auth** (`HERMES_DASHBOARD_BASIC_AUTH_*`).
  API/remote-gateway = **`API_SERVER_KEY`**. ForwardAuth = optional outer layer only.
- Provider default: **OpenRouter**. codex/claude-code = opt-in coding toolsets.
- Chart: **fork ultraworkers/hermes-agent-helm-chart** into `charts/`, port our specifics in.
- Storage: single PVC at **`/opt/data`** (drop separate `/workspace`).
- securityContext (**CORRECTED — empirically validated by running the image, 2026-06-06**):
  **image MUST start as root and drop internally via s6**. `runAsUser:0`, `runAsNonRoot:false`,
  `fsGroup:1000`; `allowPrivilegeEscalation:true`, `readOnlyRootFilesystem:false`,
  `capabilities.drop:[]`; env `HERMES_UID/HERMES_GID=1000`. Pod-level `runAsNonRoot` is NOT how
  we get "strict" here — the boundary is namespace + NetworkPolicy + ResourceQuota.

## Hard constraints discovered in upstream source + EMPIRICAL TESTING (do not violate)
1. 🔴 **The image MUST start as root (uid 0) — it CANNOT run rootless.** *(Empirically verified
   by running the image, 2026-06-06; this corrects an earlier wrong "UID 10000 non-root" claim.)*
   s6-overlay **preinit** (runs before any cont-init script) chowns `/run`, and the privilege
   drop uses `s6-setuidgid`→`setgroups()` — **both require real root**. A non-root start fails
   immediately:
   ```
   preinit: fatal: /run belongs to uid 0 instead of 1000 ... lacking the privileges to fix it
   s6-applyuidgid: fatal: unable to set supplementary group list: Operation not permitted
   cont-init: /etc/cont-init.d/01-hermes-setup exited 111
   ```
   `allowPrivilegeEscalation: false` (sets `no_new_privs`) also kills the drop. **Required SC:**
   ```yaml
   podSecurityContext: { runAsUser: 0, runAsNonRoot: false, fsGroup: 1000 }
   securityContext:     { allowPrivilegeEscalation: true, readOnlyRootFilesystem: false,
                          runAsNonRoot: false, capabilities: { drop: [] } }
   env: { HERMES_UID: "1000", HERMES_GID: "1000" }   # remap hermes→1000 before drop; == fsGroup
   ```
   The image's stage2 hook remaps the `hermes` user to `HERMES_UID/GID` and chowns the volume;
   each s6 service then drops to uid 1000 via `s6-setuidgid`. `fsGroup` MUST equal `HERMES_GID`.
   (Later hardening: try a minimal cap set — CHOWN, SETUID, SETGID, DAC_OVERRIDE, FOWNER, SETPCAP
   — instead of `drop: []`; but root start + `allowPrivilegeEscalation: true` are non-negotiable.)
2. 🔴 **`readOnlyRootFilesystem: true` is fatal** — `/run` immutable → s6 **preinit** fails
   (`exit 100`) before anything starts; also `lazy_deps.py` writes to `/opt/hermes/.venv`. Keep
   `false`. Needs writable `/run`, `/tmp`, `/dev/shm`.
3. **Dashboard fails closed on non-loopback bind without auth.** `docker/s6-rc.d/dashboard/run`:
   binds `HERMES_DASHBOARD_HOST` (default `0.0.0.0`) only if an auth provider is registered
   **or** `HERMES_DASHBOARD_INSECURE=1`. ⇒ MUST set basic-auth, or explicitly opt into
   INSECURE behind ForwardAuth. Dashboard service itself is OFF unless **`HERMES_DASHBOARD=1`**.
4. **Selector labels are chart-derived** (`app.kubernetes.io/name=<Chart.Name>`,
   `instance=<Release.Name>`); the bridge queries by `instance`=releaseName (already consistent).
   `…/name` is internal-only — update for cleanliness; existing instances recreate-not-upgrade.

---

## Phase 0 — DERISK FIRST: bare-image runtime smoke (before any chart/bridge work)
The entire single-container model assumes `hermes gateway run` **blocks forever** even with all
messaging gateways OFF (we default them off). If it exits when no gateway is configured, the
container dies and takes the s6 dashboard with it — a crash-loop that **no `helm template` or
`go test` will reveal**. Validate the runtime model against the real image *before* building the
scaffolding around it:
```bash
docker run --rm -p 9119:9119 -p 8642:8642 \
  -e HERMES_UID=1000 -e HERMES_GID=1000 \          # remap; container still starts as root (default)
  -e HERMES_DASHBOARD=1 \
  -e HERMES_DASHBOARD_BASIC_AUTH_USERNAME=admin \
  -e HERMES_DASHBOARD_BASIC_AUTH_PASSWORD=smoketest \
  -e API_SERVER_ENABLED=true -e API_SERVER_HOST=0.0.0.0 -e API_SERVER_PORT=8642 \
  -e API_SERVER_KEY=smoke -e OPENROUTER_API_KEY=sk-or-... \
  nousresearch/hermes-agent:0.8.0 gateway run
# NOTE: do NOT pass `--user` — the image starts as root and drops internally (Hard Constraint #1).
# Assert ALL of:
#  - container stays Running (gateway run does NOT exit with gateways off)  ← the key risk
#  - logs show s6 preinit OK + cont-init 01/02 exit 0 (NOT exit 100/111)   ← the SC risk
#  - curl localhost:9119/ → 401/login page (dashboard up + basic-auth registered, not refused)
#  - curl -H "Authorization: Bearer smoke" localhost:8642/v1/models → 200
# Under k8s (Phase 6) the equivalent SC is runAsUser:0 + fsGroup:1000 + the block in Hard Constraint #1.
```
**Do not proceed to Phase 1 until the container stays up and the dashboard answers.** If
`gateway run` exits with gateways off, fall back to keeping the dashboard as the main program
(`CMD=dashboard`) or enable a no-op keepalive — and revisit the process model before investing.

## Phase 0b — Image: mirror upstream to private GHCR
Goal: private, pinned, fast pulls reusing the existing `ghcr-pull-secret`.

1. Pull the official image, re-tag, push to `ghcr.io/taalib25/hermes-agent`:
   ```bash
   UPSTREAM=docker.io/nousresearch/hermes-agent:0.8.0   # pin a real published tag
   docker pull "$UPSTREAM"
   DIGEST=$(docker inspect --format='{{index .RepoDigests 0}}' "$UPSTREAM")
   docker tag "$UPSTREAM" ghcr.io/taalib25/hermes-agent:0.8.0
   docker push ghcr.io/taalib25/hermes-agent:0.8.0
   docker inspect --format='{{index .RepoDigests 0}}' ghcr.io/taalib25/hermes-agent:0.8.0  # record digest
   ```
2. Record the **digest** and pin by digest in `bridge_config.go` (not a moving tag).
3. No overlay image now. Revisit only if we must bake a custom CA, pinned npm MCP servers,
   or extra skills (then a 3-line `FROM ghcr…hermes-agent` Dockerfile).

Acceptance: `k3s ctr images pull` of the GHCR ref succeeds on the test node using
`ghcr-pull-secret`; digest recorded.

---

## Phase 1 — Fork the chart into `charts/hermes-agent`
1. Copy ultraworkers chart verbatim into `charts/hermes-agent/`. Add a header comment in
   `Chart.yaml` recording upstream ref + commit SHA for future manual diffs.
2. Set `Chart.yaml`: `name: hermes-agent` (this becomes `app.kubernetes.io/name` — Phase 3
   aligns the bridge to it), `appVersion: "0.8.0"`.
3. Keep `runtime-node-core` chart in-tree until cutover is proven, then delete.

Acceptance: `helm lint charts/hermes-agent` clean (stock values).

---

## Phase 2 — Port our specifics INTO the fork
Four additions; ultraworkers lacks all four.

### 2a. Label contract (`hermescloud.dev/*`)
- In `templates/_helpers.tpl`, extend `hermes-agent.labels` to merge `.Values.commonLabels`:
  ```gotemplate
  {{- with .Values.commonLabels }}
  {{ toYaml . }}
  {{- end }}
  ```
- Add a reusable annotations helper and emit it on every resource's `metadata.annotations`
  (deployment, service, ingress, pvc, secret, configmap, networkpolicy, quota):
  ```gotemplate
  {{- define "hermes-agent.commonAnnotations" -}}
  {{- with .Values.commonAnnotations }}{{ toYaml . }}{{- end }}
  {{- end -}}
  ```
- Add `commonLabels: {}` / `commonAnnotations: {}` to `values.yaml`. (Schema already permits
  unknown top-level keys; add them for documentation.)

### 2a-bis. Sane PVC default (don't inherit 5Gi)
ultraworkers defaults `persistence.size: 5Gi`. lightrag + sessions + memories + `npm-global`
(MCP) + playwright cache grow under load and will silently wedge the agent. Set the fork's
default to **20Gi** (and let the bridge override per plan tier). Single PVC now holds what the
old 10Gi(state)+20Gi(workspace) split did.

### 2b. ResourceQuota + LimitRange
- Copy `charts/runtime-node-core/templates/quota.yaml` → `charts/hermes-agent/templates/quota.yaml`.
- Gate on `.Values.resourceQuota.enabled`; pull caps from `.Values.resources` (1 pod, 2 PVCs).
- Add `resourceQuota: {enabled: true}` to `values.yaml`.

### 2c. Dashboard as a routable listener
The dashboard (9119) is **not** in `hermes-agent.servicePorts`. We expose it via the bridge
setting explicit `service.ports` (Phase 3), so **no template change is strictly required** —
but for standalone use, extend `hermes-agent.servicePorts` to append a `dashboard` port when
`.Values.dashboard.enabled`. Add `dashboard: {enabled: true, port: 9119}` to `values.yaml`.

### 2d. ResourceQuota/NetworkPolicy defaults
- Set `values.yaml` `securityContext`/`podSecurityContext` to the **root-start profile** from
  Hard Constraint #1 (`runAsUser:0`, `runAsNonRoot:false`, `allowPrivilegeEscalation:true`,
  `readOnlyRootFilesystem:false`, `capabilities.drop:[]`) so a direct `helm install` works.
- **Expose `HERMES_UID`/`HERMES_GID` as first-class chart values** (per the empirical-fix
  suggestion) — e.g. `hermes.uid`/`hermes.gid` that drive BOTH the env vars AND `fsGroup`, so
  they can't drift apart. Default `1000/1000`.

Acceptance: `helm template t charts/hermes-agent --show-only templates/quota.yaml` and
`--show-only templates/_helpers… ` via a probe template render the quota + labels;
`helm lint` clean.

---

## Phase 3 — Rewire the bridge
All in `bridge/helm.go` `buildInstanceValues` (the value map), plus `bridge_config.go`,
`labels.go`, and `types.go` defaults.

### 3a. `bridge_config.go`
- `RuntimeNodeCoreImage` → `ghcr.io/taalib25/hermes-agent`; `RuntimeNodeCoreImageTag` →
  `0.8.0` (and pin digest via `image.digest`).
- `ChartPath` default → the new `charts/hermes-agent` (env `BRIDGE_CHART_PATH`).

### 3b. `labels.go` — selector identity (verified: lower risk than first thought)
- **Verified:** the bridge's live pod/PVC/svc queries key on **`app.kubernetes.io/instance: <releaseName>` ONLY**
  (`admin_diagnostics.go:69/150`, `admin_resources.go:74`) — never on `…/name`. The load-bearing
  invariant is therefore **releaseName consistency**, which already holds (chart sets
  `instance: {{ .Release.Name }}`; bridge installs + queries with `b.releaseName(instanceID)`).
- `chartName`/`…/name` is **internal-only** (labels.go `deriveSelectorLabels` + `bridge_api_test.go:140`).
  Update the const to `hermes-agent` and the test for cleanliness — NOT required for correctness.
- Still true: Deployment `selector.matchLabels` is immutable, so existing instances must be
  **recreated, not upgraded** when switching charts (the chart's `…/name` value changes).

### 3c. `helm.go` value map — concrete diff
**Delete:**
- the entire `rncEnv` block (HERMES_WEBUI_* + toolchain PATH/cache env), lines ~560–598.
- the `/workspace` `extraVolumeMounts` subPath entry.
- the busybox `init-dirs` `extraInitContainers` entry.
- `persistence.mountPath` override (chart default `/opt/data` is correct).
- `command`/`args = []` clearing.

**Change/add:**
```go
values["args"] = []any{"gateway", "run"}          // dashboard auto-starts via s6
values["persistence"].(map[string]any)["accessMode"] = firstOr(spec.Storage.AccessModes, "ReadWriteOnce") // chart uses singular

// Two listeners; dashboard FIRST so ingress primaryServicePortNumber routes to it.
const dashboardPort = 9119
const apiServerPort = 8642
values["service"] = map[string]any{
    "enabled": true,
    "ports": []any{
        map[string]any{"name": "dashboard", "port": dashboardPort, "targetPort": dashboardPort, "protocol": "TCP"},
        map[string]any{"name": "api-server", "port": apiServerPort, "targetPort": apiServerPort, "protocol": "TCP"},
    },
}
values["apiServer"] = map[string]any{"enabled": true, "host": "0.0.0.0", "port": apiServerPort, "corsOrigins": spec.CORSOrigins}

// CORRECTED (empirically validated): root start + internal s6 drop. See Hard Constraint #1.
values["podSecurityContext"] = map[string]any{
    "runAsUser": int64(0), "runAsNonRoot": false,
    "fsGroup": int64(1000), "fsGroupChangePolicy": "OnRootMismatch",
    "seccompProfile": map[string]any{"type": "RuntimeDefault"},
}
values["securityContext"] = map[string]any{
    "allowPrivilegeEscalation": true, "readOnlyRootFilesystem": false,
    "runAsNonRoot": false,
    "capabilities": map[string]any{"drop": []any{}}, // s6 needs chown/setuid/setgid; tighten later
    "seccompProfile": map[string]any{"type": "RuntimeDefault"},
}
// Remap the in-image hermes user to match fsGroup, set in spec.EnvMap below:
//   HERMES_UID=1000  HERMES_GID=1000   (MUST equal podSecurityContext.fsGroup)

// Probes — TCP on the dashboard port is robust without knowing the HTTP health path.
// (Optional upgrade: httpGet /health on the api-server port once confirmed.)
values["probes"] = map[string]any{
    "readiness": map[string]any{"tcpSocket": map[string]any{"port": dashboardPort},
        "initialDelaySeconds": 20, "periodSeconds": 10, "timeoutSeconds": 5, "failureThreshold": 12},
    "liveness": map[string]any{"tcpSocket": map[string]any{"port": dashboardPort},
        "initialDelaySeconds": 120, "periodSeconds": 20, "timeoutSeconds": 5, "failureThreshold": 6},
}
```

**Non-secret env** — set via `values["env"]` (a **map**; chart renders it as name/value pairs,
deployment.yaml lines 198–200). Do **NOT** use `extraEnv` for this (see bug-fix below):
```go
spec.EnvMap["HERMES_DASHBOARD"] = "1"                 // enable dashboard s6 service
spec.EnvMap["HERMES_DASHBOARD_PORT"] = "9119"
spec.EnvMap["HERMES_DASHBOARD_BASIC_AUTH_USERNAME"] = "tenant"  // username can be plain
spec.EnvMap["HOME"] = "/opt/data"                     // parity with s6 services (chart else sets /opt/data/home)
spec.EnvMap["HERMES_UID"] = "1000"                    // remap hermes user before s6 drops privileges
spec.EnvMap["HERMES_GID"] = "1000"                    // MUST equal podSecurityContext.fsGroup
values["env"] = spec.EnvMap
// HERMES_DASHBOARD_HOST defaults 0.0.0.0; API_SERVER_* are auto-set from apiServer.* (verified);
// API_SERVER_KEY + HERMES_DASHBOARD_BASIC_AUTH_PASSWORD + OPENROUTER_API_KEY arrive via envFrom secretRef.
```

**Bug-fixes against this chart (verified by reading `deployment.yaml`):**
- 🔴 **Remove `values["extraEnv"] = spec.EnvMap`.** ultraworkers `env` is a map, `extraEnv` is a
  **list** of `{name,value}` (`{{- with .Values.extraEnv }}{{ toYaml . }}` under `env:`). Passing a
  map produces invalid YAML. Put flat env in `env` only; if structured pairs are ever needed, set
  `extraEnv` as a list.
- 🔴 **Remove `values["extraSecretKeys"]`.** ultraworkers has no such mechanism — it injects the
  whole instance Secret via `envFrom: secretRef: <existingSecret>`. All secret keys become env
  automatically. Drop `extraEnvList` too unless mapped to the list-shaped `extraEnv`.
- ℹ️ **MCP servers:** install per-tenant MCP `npx` packages via the `npmPackages` value (list) →
  the npm-install initContainer persists them to `/opt/data/npm-global` and they land on `PATH`.

**Ingress:** drop the bad `servicePort` key; rely on `primaryServicePortNumber` (= dashboard,
first port). Keep host/className/CORS+ForwardAuth middleware annotations as-is.

**NetworkPolicy** (raw passthrough — the fork renders `networkPolicy.ingress/egress` verbatim):
```go
values["networkPolicy"] = map[string]any{
    "enabled": true,
    "policyTypes": []any{"Ingress", "Egress"},
    "ingress": []any{ /* from kube-system (Traefik) + hermes-bridge ns to 9119 & 8642 */ },
    "egress":  []any{ /* DNS to kube-system:53; internet 0.0.0.0/0 EXCEPT the blocks below */ },
}
values["resourceQuota"] = map[string]any{"enabled": true}
```
🔴 **STRICT egress — the NetworkPolicy is the real boundary** (`terminal.backend: local` runs an
adversarial LLM in-pod; the config denylist/tirith is only a soft control). The internet egress
`ipBlock.except` MUST include, beyond the cluster/service CIDRs:
- **`169.254.0.0/16`** — link-local incl. Hetzner cloud **metadata `169.254.169.254`** (SSRF/credential theft)
- the **node's private subnet** (e.g. `10.0.0.0/16` if the Hetzner private network is used)
```yaml
egress:
  - to: [{ipBlock: {cidr: 0.0.0.0/0, except: [10.42.0.0/16, 10.43.0.0/16, 169.254.0.0/16, <node-private-subnet>]}}]
  - to: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: kube-system}}}]
    ports: [{protocol: UDP, port: 53}, {protocol: TCP, port: 53}]
```

### 3d. `types.go`
- `RuntimePort` default semantics → dashboard `9119` (the routed surface). Add `APIServerPort`
  (8642) if you want it configurable; otherwise const.
- Update the doc comments (they reference 8787/runtime-node-core).

---

## Phase 4 — Secrets & opinionated config defaults
### 4a. Per-tenant Secret (bridge-managed, `providers.go`/`helm.go` secret writer)
Ensure these land in the instance Secret (consumed via `envFrom: secretRef`):
- `API_SERVER_KEY` — already generated (`handlers_instance.go:84`).
- `OPENROUTER_API_KEY` — provider key (from backend or admin key-set flow).
- `HERMES_DASHBOARD_BASIC_AUTH_PASSWORD` — **generate per tenant** (`randomHex(24)`); return
  it to the backend in the create response alongside `API_SERVER_KEY` so the dashboard URL is
  usable. (Confirmed env names in `plugins/dashboard_auth/basic/__init__.py`: `_USERNAME`,
  `_PASSWORD` or `_PASSWORD_HASH` (preferred), `_SECRET`, `_TTL_SECONDS`.)
- `HERMES_DASHBOARD_BASIC_AUTH_SECRET` — **generate per tenant** (`randomHex(32)`); without it,
  HMAC session tokens don't survive pod restarts (tenant re-logs-in on every Recreate).
- (optional) gateway tokens (`TELEGRAM_BOT_TOKEN`, …) only when a tenant opts in.

### 4b. Opinionated `config.yaml` defaults in `charts/hermes-agent/values.yaml` `config.values`
Default-on (cost + safety), overridable per tenant via `spec.HermesConfig`:
```yaml
config:
  values:
    model:
      provider: openrouter
      base_url: https://openrouter.ai/api/v1
      default: anthropic/claude-3.7-sonnet      # pick a real OpenRouter slug
    compression: { enabled: true, threshold: 0.5, target_ratio: 0.2, protect_last_n: 20 }
    security:    { redact_secrets: true, tirith_enabled: true, tirith_fail_open: true }
    terminal:    { backend: local }             # in-pod; per-tenant namespace is the boundary
```
Opt-in per tenant (bridge merges into `config.values`): MCP servers, chat gateways, cron.

---

## Phase 5 — Local verification gauntlet (must all pass)
```bash
# Chart renders with the EXACT values the bridge emits (capture from a unit test fixture):
helm lint charts/hermes-agent
helm template t charts/hermes-agent -f /tmp/bridge-emitted-values.yaml \
  --set secrets.existingSecret=x | tee /tmp/render.yaml
# Assert in the render:
#  - Deployment args: [gateway, run]; image = ghcr…hermes-agent@<digest>
#  - podSecurityContext.runAsUser=0, runAsNonRoot=false, fsGroup=1000; securityContext.allowPrivilegeEscalation=true, readOnlyRootFilesystem=false, capabilities.drop=[]
#  - env has HERMES_UID=1000, HERMES_GID=1000 (== fsGroup)
#  - env has HERMES_DASHBOARD=1, HERMES_DASHBOARD_BASIC_AUTH_USERNAME
#  - Service has ports dashboard(9119)+api-server(8642); Ingress backend port = 9119
#  - NetworkPolicy ingress(9119,8642)+egress(DNS+internet-except-cluster); ResourceQuota present
#  - every resource carries hermescloud.dev/* commonLabels + commonAnnotations
#  - selector app.kubernetes.io/name=hermes-agent matches bridge labels.go

# Bridge:
go build ./bridge/...
go test -race -count=1 ./bridge/...
```

---

## Phase 6 — Deploy to hermes-test + end-to-end smoke
1. Build/push bridge image; deploy bridge (existing CI path).
2. Create a throwaway instance via the bridge API. Then verify on-cluster (SSH):
   ```bash
   kubectl -n <ns> get pod -o wide                       # 1/1 Running, restartCount low
   kubectl -n <ns> logs <pod> | grep -i 's6-rc\|dashboard\|gateway'   # both services up
   kubectl -n <ns> get svc,ingress,networkpolicy,resourcequota
   ```
3. **Dashboard auth fail-closed check (critical):** confirm the dashboard actually started
   (basic-auth provider registered). `curl -k https://wsp_<id>.hermeshq.net/` → expect a
   **login page / 401**, NOT connection-refused (refused = it failed closed → auth misconfig).
   Authenticate with `tenant` + the generated password → dashboard loads.
4. **API server:** `curl -H "Authorization: Bearer $API_SERVER_KEY" https://…:<api>/v1/models`
   (or the cluster-internal svc:8642) returns 200; without the key → 401.
5. **Agent works:** send a chat turn via the dashboard or the OpenAI-compatible endpoint with
   the OpenRouter key set → a model response comes back.
6. **State persists:** delete the pod (not the release) → it reschedules, `/opt/data` survives
   (session/memory intact).
7. **Isolation:** from the pod, `nc -z <other-tenant-svc> 9119` blocked; DNS + OpenRouter
   egress allowed; `kubectl -n <ns> describe resourcequota` shows pods 1/1 used.
8. Run the existing bridge feature smoke (maintenance 503, capacity headroom, label 422,
   WebSocket origin) — unchanged behavior.

---

## Phase 7 — Rollout & rollback
- **Rollout:** ship behind the existing per-instance image/runtime selection. New instances get
  `hermes-agent`; existing opencode instances keep running until migrated (recreate, not
  upgrade — selector label changes). Optionally a `runtimeMode` value to choose engine.
- **Rollback:** revert `bridge_config.go` image default + `ChartPath` to `runtime-node-core`;
  redeploy bridge. Per-instance: `redeploy`/`rollback` op to the previous chart/image.
- Delete `charts/runtime-node-core` only after a tenant has run on `hermes-agent` for a week.

## Risk register
| Risk | Mitigation |
|---|---|
| `gateway run` exits with all gateways off → container + dashboard die (crash-loop) | **Phase 0 bare-image smoke gates this before any build**; fallback CMP=dashboard / keepalive |
| Adversarial agent hits Hetzner metadata `169.254.169.254` (SSRF/cred theft) | NetworkPolicy egress `except` includes `169.254.0.0/16` + node subnet (Phase 3c) |
| Dashboard fails closed (no auth) → connection refused | Basic-auth `_USERNAME`+`_PASSWORD`(+`_SECRET`) mandatory; Phase 0 + Phase 6 step 3 gate |
| Non-root start → s6 preinit/cont-init fail (exit 100/111) | **Must start as root** (runAsUser:0, allowPrivEsc:true, drop:[]) + HERMES_UID/GID=fsGroup; verified empirically (Hard Constraint #1) |
| Selector label drift → bridge can't find pods / immutable-selector upgrade error | Phase 3b alignment + recreate-not-upgrade for existing |
| `config.values` deep-merge surprises | Unit-test the merged render against a fixture |
| OpenRouter model slug invalid | Pin a verified slug; smoke step 5 |
| Lazy adapter install needs network/rootfs | rootfs writable (false); egress allows PyPI if needed, else bake in overlay |
| API server bound to localhost only | Verify deployment sets `API_SERVER_HOST` from `apiServer.host=0.0.0.0`; else add to fork |

## Open verification items (confirm during Phase 2/5, do not assume)
- ✅ RESOLVED: fork `deployment.yaml` sets `API_SERVER_ENABLED/HOST/PORT/CORS_ORIGINS/MODEL_NAME`
  from `apiServer.*` (host `0.0.0.0`); `API_SERVER_KEY` arrives via `envFrom secretRef`. No change.
- ✅ RESOLVED: `.Values.env` (map) renders to container env name/value pairs — dashboard env works.
- Hermes dashboard HTTP health path (to upgrade probes from TCP to httpGet) — minor.
- Exact OpenRouter default model slug + whether codex/claude-code need extra config keys.
- Confirm `helm template` with the bridge's exact value map renders valid YAML after the
  `extraEnv`/`extraSecretKeys` fixes (Phase 5 is the gate).
