# HermesCloud Runtime Edge Cases

## 1. Hermes Agent path changes in the upstream image

The Dockerfile does not directly `COPY /opt/hermes`, because that would fail without useful diagnostics if upstream changes layout. It normalizes the source in a build stage:

- `/opt/hermes`
- `/app`
- `/workspace`

The selected folder must contain `pyproject.toml`.

## 2. WebUI startup file changes

`start.sh` tries, in order:

1. `startup.py`
2. `server.py`
3. `app.py`
4. `python -m hermes_webui`

If none exist, it prints diagnostics and exits.

## 3. PVC permissions

The container runs as UID/GID `1024`. In Kubernetes, set:

```yaml
securityContext:
  fsGroup: 1024
  fsGroupChangePolicy: OnRootMismatch
```

Without this, empty PVCs may be root-owned and the runtime will fail early with a clear message.

## 4. Do not persist the base Python venv on PVC

Base WebUI/Hermes Agent dependencies are installed into the image venv at build time. User-installed tools go to PVC. This avoids stale compiled wheels and Python-version mismatch after image upgrades.

## 5. Runtime code vs user state

Production split:

```txt
Image:
- /opt/hermes-agent
- /opt/hermes-webui
- base Python venv

PVC:
- /home/hermeswebui/.hermes
- /workspace
```

Do not make `/home/hermeswebui/.hermes/hermes-agent` the long-term source of truth in production.

## 6. GHCR private image pull

For private images, create `ghcr-pull-secret` in the target namespace and add it to `imagePullSecrets`.

## 7. Kind storage

Kind PVCs survive pod restarts and rollouts, but not `kind delete cluster` unless you configured mounted host storage.

## 8. Debug commands

```bash
kubectl -n hermes-test logs deployment/hermescloud-runtime-test -c hermes-runtime
kubectl -n hermes-test exec -it deploy/hermescloud-runtime-test -c hermes-runtime -- hermescloud-diagnose
kubectl -n hermes-test port-forward svc/hermescloud-runtime-test 8787:8787
```
