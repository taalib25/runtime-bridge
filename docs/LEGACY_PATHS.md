# Legacy and Experimental Paths

The following paths are **not** the canonical production runtime path unless
explicitly reactivated. A coding agent must not extend or modify these paths
as part of normal bridge/runtime work.

---

## `_archive/old-runtime-experiments/prod-hermes-docker-image/`

**Status:** Archived legacy experiment.

**Purpose:**
Previously used to build a custom bundled WebUI + Hermes runtime Docker image for
standalone (non-Kubernetes) deployment via Docker Compose or a raw VM.

**Do not** modify for production bridge work. The canonical production path is
K3s + Helm + bridge, not a custom bundled Docker image.

---

## `hermes-k3s-runtime-test/`

**Status:** Manual smoke-test path.

**Purpose:**
Raw Kubernetes manifests for direct pod testing without Helm. Useful for quickly
verifying a new runtime image works before going through the full Helm chart path.

**Do not** treat as a production deployment source. Do not add new features here.
Production/runtime instances are created by: Bridge → Helm SDK → canonical chart.

---

## `charts/runtime-node-core/` — REMOVED (2026-06-18)

The lightweight legacy chart (opencode-webui era) has been deleted from the repo
and from the root `Dockerfile`'s `COPY` instructions. `BRIDGE_CHART_PATH` had
already pointed at `/charts/hermes-agent` in `deploy/deployment.yaml` for a while
before the directory itself was removed — this chart was dead weight kept only
as a rollback option. Do not recreate it; if a rollback is ever needed, restore
it from git history instead.

---

## `charts/hermes-agent/`

**Status:** Active — the only runtime chart, wired into the bridge via
`BRIDGE_CHART_PATH=/charts/hermes-agent` (see `deploy/deployment.yaml`).

**Purpose:**
Feature-rich Helm chart for Hermes agent deployments (RBAC, NetworkPolicy,
ExternalSecret, VirtualService, CRD-based operator mode, strict JSON schema).
This is the chart to modify when changing runtime deployment behavior — there
is no other chart in the repo.

---

## `deploy/`

**Status:** Active — production deployment manifests for the bridge service itself.

**Not** the runtime deployment path. These manifests deploy the bridge binary
(the Go service) into the `hermes-bridge` namespace. They are applied by CI
on push to `main`/`go-bridge`.

**Do not** confuse `deploy/` (bridge service manifests) with `charts/` (runtime
Helm charts installed by the bridge).
