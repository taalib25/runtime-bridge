# Coding Agent Rules

Rules for any automated agent or developer working in this repository.

## Non-Destructive Rules

1. Do not delete files unless explicitly asked.
2. Prefer `git mv` over delete + recreate for file renames.
3. Move uncertain legacy paths to `_archive/` only after verifying no CI or docs reference them.
4. Do not change API routes and file names in the same commit.
5. Do not change runtime chart behavior and bridge behavior in the same commit unless the task requires it.
6. Do not edit multiple runtime charts simultaneously unless the task explicitly says so.
7. Do not modify `cluster-config.yaml` or `cluster-config-test.yaml` unless the task is about Hetzner/K3s provisioning.
8. Do not commit `kubeconfig-test` or any file containing real credentials.
9. Do not hardcode API keys, tokens, or secrets in source files.
10. Always run tests after renames or logic changes.

## Required Commands

**Before making changes:**
```bash
git status
git branch --show-current
```

**After any Go change:**
```bash
go build ./bridge/...
go test ./bridge/... -count=1
go vet ./bridge/...
```

**After any chart change:**
```bash
helm lint charts/runtime-node-core
helm template test-ws charts/runtime-node-core \
  --set secrets.existingSecret=test-ws-secrets \
  --set fullnameOverride=test-ws \
  > /dev/null
```

**After moving or renaming files:**
```bash
grep -rn "old-file-name" --include="*.go" --include="*.yml" --include="*.yaml" --include="*.md" .
```

## Production Path Rule

The production runtime path is:

```
Bridge (bridge/) → Helm SDK → charts/runtime-node-core/
```

Do **not** add production features to:
- `_archive/` paths
- `hermes-k3s-runtime-test/`
- `charts/hermes-agent/` (unless the bridge is explicitly updated to use it)
- `prod-hermes-docker-image/` or its archive location

## Secret / Security Rules

- Provider API keys belong in the bridge-owned k8s Secret (`{workspace-id}-secrets`).
- Secret values are **never** written by Helm. Only key references go in Helm values (`extraSecretKeys`).
- `kubeconfig-test` contains real cluster credentials — never commit changes to it.
- `cluster-config.yaml` and `cluster-config-test.yaml` contain Hetzner API tokens — do not print or log their contents.

## File Ownership Map

| File/Directory | Owner | Notes |
|----------------|-------|-------|
| `bridge/` | Bridge team | Go source — all changes must pass `go test` |
| `charts/runtime-node-core/` | Bridge team | Canonical runtime chart |
| `charts/hermes-agent/` | Hermes team | Not wired to bridge — see LEGACY_PATHS.md |
| `deploy/` | Bridge team | Bridge service manifests, not runtime charts |
| `.github/workflows/bridge.yml` | Bridge team | CI/CD for the bridge |
| `docs/` | Bridge team | Architecture docs — update when behavior changes |
| `_archive/` | Read-only | Do not modify archived content |
| `hermes-k3s-runtime-test/` | Manual testing only | Do not extend |
| `cluster-config*.yaml` | Infra team | Do not modify |
| `kubeconfig-test` | Read-only secret | Do not commit |

## Canonical Naming Conventions

| Concept | Go file | Handler file |
|---------|---------|--------------|
| Workspace lifecycle | `bridge/helm.go`, `bridge/lifecycle.go` | `bridge/handlers_workspace.go` |
| Provider config | `bridge/providers.go` | `bridge/handlers_config.go` |
| Messaging integrations | `bridge/integrations.go` | `bridge/handlers_integrations.go` |
| Agent templates | `bridge/agent_templates.go` | `bridge/handlers_agent_templates.go` |
| Terminal exec (WebSocket) | `bridge/terminal_exec.go` | (registered in `bridge/bridge.go`) |
| Async operations queue | `bridge/bridge.go` (`submitOperation`) | — |
