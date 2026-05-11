# Charts

## Canonical Runtime Chart

The canonical runtime chart installed by the bridge is:

```
charts/runtime-node-core/
```

This is the chart the bridge uses for all `CreateWorkspace` / `UpdateWorkspace` calls.
It is designed for bridge-managed workspaces and supports:

- `extraSecretKeys` — flat map of `ENV_VAR → secret-data-key` rendered as `secretKeyRef` blocks
- `extraEnv` — flat map of plain env vars (platform flags, workspace identity, allowlists)
- `secrets.existingSecret` — bridge-owned k8s Secret name (bridge creates/patches directly; Helm never writes values)
- `bootstrap.overwrite` — controls whether config.yaml is overwritten on pod start
- `config.values` — HermesConfig rendered into the pod's `HERMES_HOME/config.yaml`

**Add new runtime behavior here.** Do not add production features to `charts/hermes-agent/`.

---

## Feature-Rich Chart (not currently wired to bridge)

```
charts/hermes-agent/
```

The original Helm chart for Hermes agent deployments. Contains additional templates:
RBAC, NetworkPolicy, ExternalSecret, VirtualService, PodDisruptionBudget, Ingress,
IngressRoute, and a strict JSON schema.

**Status:** Not the chart installed by the bridge (`bridge/helm.go` uses `runtime-node-core`).

**Decision needed:** Either wire `hermes-agent` into the bridge as the canonical chart
(updating `BRIDGE_CHART_PATH` and `buildValues` accordingly), or continue evolving
`runtime-node-core`. Until that decision is made and documented here, do not add
production bridge features to `hermes-agent`.

---

## Adding a New Provider or Integration

New LLM providers:
→ Add to `providerSecretKeys` in `bridge/providers.go` only. No chart change needed.
  All provider keys flow through `extraSecretKeys` automatically.

New messaging platforms:
→ Add to `platformSecretKeys` + `platformEnvKeys` in `bridge/integrations.go`.
→ Add request type to `bridge/types.go`.
→ Add decoder case to `bridge/handlers_integrations.go`.
  No chart change needed.
