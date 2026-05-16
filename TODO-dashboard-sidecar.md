# TODO: Official Dashboard Sidecar

## Feature Overview

Add an optional dashboard sidecar to the `hermes-agent` chart that runs alongside the
main runtime-node-core container. Disabled by default — zero impact on existing workspaces.

## Chart Changes Required

### New chart values (hermes-agent/values.yaml)

```yaml
dashboard:
  enabled: false
  image:
    repository: ""        # dashboard container image
    tag: ""
  port: 9119
  servicePort: 9119
  ingress:
    enabled: false
    host: ""              # e.g. dashboard-<wsID>.hermeshq.net
    forwardAuth: true     # protect via Traefik ForwardAuth
```

### Deployment template

- Add `{{ if .Values.dashboard.enabled }}` sidecar container block
- Port: `9119`
- Same `volumeMounts` as main container (shares the `data` PVC)
- No extra `command`/`args` by default — image provides its own entrypoint

### Service template

Add a second port to the existing Service when `dashboard.enabled`:

```yaml
- name: dashboard
  port: {{ .Values.dashboard.servicePort }}
  targetPort: {{ .Values.dashboard.port }}
  protocol: TCP
```

### Traefik IngressRoute (bridge-managed, not chart-managed)

When `dashboard.enabled && dashboard.ingress.enabled`:

1. Bridge creates a second `IngressRoute` in the workspace namespace pointing at the
   dashboard Service port (9119).
2. Route is always protected by the workspace ForwardAuth middleware (same as main route).
3. Bridge `DELETE /instances/:id` must also delete the dashboard IngressRoute.

## Bridge Changes Required

### types.go — WorkspaceSpec

```go
Dashboard struct {
    Enabled bool   `json:"enabled,omitempty"`
    Image   string `json:"image,omitempty"`
    Tag     string `json:"tag,omitempty"`
    Port    int    `json:"port,omitempty"` // default 9119
} `json:"dashboard,omitempty"`
```

### helm.go — buildValues

```go
if spec.Dashboard.Enabled {
    values["dashboard"] = map[string]any{
        "enabled": true,
        "image": map[string]any{
            "repository": spec.Dashboard.Image,
            "tag":        spec.Dashboard.Tag,
        },
        "port":        orDefault(spec.Dashboard.Port, 9119),
        "servicePort": orDefault(spec.Dashboard.Port, 9119),
    }
}
```

### traefik.go — new helper

`EnsureDashboardIngressRoute(ctx, ns, host, servicePort, hasAuth)` — analogous to
`EnsureIngressRoute` but targets the dashboard Service port.

### handlers.go — handleDeleteWorkspace

Already calls `DeleteDashboardIngressRoute` — no change needed, just implement the
helper in `traefik.go`.

## Notes

- `dashboard.enabled=false` is the default — no regression risk for existing deployments.
- The sidecar shares the same PVC so it can read agent state, sessions, logs, etc.
- ForwardAuth is strongly recommended; expose without it only for dev/internal clusters.
- Deferred until: the dashboard image (`image.repository`) is ready to publish.
- Build blocker: image build deferred (not enough data bandwidth to push to GHCR yet).
