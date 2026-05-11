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

## `charts/runtime-node-core/`

**Status:** Active — current canonical runtime chart for bridge-managed workspaces.

**Purpose:**
Lightweight Helm chart used by the bridge to deploy Hermes runtime instances.
All bridge secrets/env handling (`extraSecretKeys`, `extraEnv`) is wired to this chart.

**Coding agent rule:** This is the chart to modify when changing runtime deployment
behavior. `charts/hermes-agent/` is a more feature-rich chart (see below) that is
not currently wired into the bridge.

---

## `charts/hermes-agent/`

**Status:** Feature-rich chart — not currently wired into the bridge.

**Purpose:**
The original Helm chart for Hermes agent deployments. Contains more templates
(RBAC, NetworkPolicy, ExternalSecret, VirtualService, etc.) and a strict JSON
schema. Was the chart used before `runtime-node-core` was introduced for the
bridge-managed path.

**Decision needed (tracked separately):** Either wire this chart into the bridge
as the canonical runtime chart, or continue with `runtime-node-core`. Do not add
new production features to this chart unless the bridge is updated to use it.

---

## `deploy/`

**Status:** Active — production deployment manifests for the bridge service itself.

**Not** the runtime deployment path. These manifests deploy the bridge binary
(the Go service) into the `hermes-bridge` namespace. They are applied by CI
on push to `main`/`go-bridge`.

**Do not** confuse `deploy/` (bridge service manifests) with `charts/` (runtime
Helm charts installed by the bridge).
