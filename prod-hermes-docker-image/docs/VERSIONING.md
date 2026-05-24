# Versioning Model

Use aligned versions for the current phase:

- Git tag: `v0.1.0`
- Docker image: `ghcr.io/taalib25/runtime-node-core:0.1.0`
- Helm chart version: `0.1.0`
- Helm appVersion: `0.1.0`

Do not use `latest` in production.

## Upgrade

Release next image tag (example `0.1.1`) and run:

```bash
helm upgrade runtime-test ./charts/runtime-node-core \
  --namespace zamo-workspace \
  --set image.repository=ghcr.io/taalib25/runtime-node-core \
  --set image.tag=0.1.1
```

## Rollback

```bash
helm rollback runtime-test
```

## Digest-Based Deploy

For strict production pinning, deploy by digest:

```bash
helm upgrade --install runtime-test ./charts/runtime-node-core \
  --namespace zamo-workspace \
  --set image.repository=ghcr.io/taalib25/runtime-node-core \
  --set image.digest=sha256:<digest> \
  --set image.tag=""
```
