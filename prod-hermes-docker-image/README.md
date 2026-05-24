# HermesCloud Runtime Image

This package creates the production HermesCloud runtime image for beta/prod deployments.

It replaces the slow test pattern:

```txt
webui image
→ init container copies hermes-agent source into PVC
→ webui pip-installs hermes-agent on startup
```

with this production-grade pattern:

```txt
custom image already contains:
- hermes-webui
- hermes-agent source
- hermes-agent installed into the WebUI venv
- common coding tools

PVC contains:
- sessions/settings/memory
- user-installed tools
- package manager caches
- workspace files
```

## What this gives you

```txt
✅ faster pod startup
✅ cleaner image-tag based updates
✅ easier rollback
✅ less first-boot failure
✅ non-root runtime user
✅ PVC-persisted user tools and workspace
✅ clear diagnostics when paths/permissions break
✅ Kind/test-server Kubernetes manifests
✅ bridge snippets for runtimeImageMode support
```

## Files

```txt
Dockerfile                         Custom HermesCloud runtime image
start.sh                           Robust runtime entrypoint
scripts/healthcheck.sh             Container healthcheck
scripts/diagnose.sh                Debug helper inside the pod
scripts/verify-agent-source.sh     Verify upstream Hermes Agent image layout
docker-compose.yml                 Local Docker test
k8s/                               Kind/test-server manifests
.github/workflows/build.yml        GHCR build/push workflow
bridge/bridge_snippets.go          Reference bridge/helm.go logic
docs/EDGE_CASES.md                 Edge cases and debugging notes
```

## Production deployment path

Use Helm chart deployment for production:

- `charts/runtime-node-core`
- `helm upgrade --install ...`

Raw manifests under `prod-hermes-docker-image/k8s/` are retained for local/manual smoke tests.

## Runtime layout

```txt
Image-owned runtime code:
/opt/hermes-agent
/opt/hermes-webui
/opt/hermes-webui/.venv

PVC-owned state/tools:
/home/hermeswebui/.hermes
/workspace
```

This is the important split:

```txt
Image = runtime version
PVC   = customer state and user-installed tools
```

Do not persist the base Python venv on the PVC. It makes upgrades harder because compiled wheels can become stale across image versions. This image installs the base venv at build time and keeps only user-installed tools on the PVC.

## 1. Verify the upstream Hermes Agent image

```bash
./scripts/verify-agent-source.sh nousresearch/hermes-agent:latest
```

Expected: it finds a folder with `pyproject.toml`.

## 2. Build locally

```bash
make build IMAGE=ghcr.io/taalib25/runtime-node-core TAG=0.1.0
```

Equivalent raw command:

```bash
docker build \
  --build-arg HERMES_AGENT_IMAGE=nousresearch/hermes-agent:latest \
  --build-arg HERMES_WEBUI_REF=main \
  -t runtime-node-core:0.1.0 \
  .
```

## 3. Run locally with Docker Compose

```bash
cp .env.example .env
# add test provider keys to .env if needed

docker compose up --build
```

Open:

```txt
http://localhost:8787
```

## 4. Push to GHCR manually

```bash
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u taalib25 --password-stdin

docker tag runtime-node-core:0.1.0 ghcr.io/taalib25/runtime-node-core:0.1.0
docker push ghcr.io/taalib25/runtime-node-core:0.1.0
```

For private GHCR images, create a pull secret:

```bash
export GITHUB_USERNAME=taalib25
export GITHUB_TOKEN=YOUR_GITHUB_TOKEN
export GITHUB_EMAIL=you@example.com
./k8s/ghcr-pull-secret-example.sh
```

Then uncomment `imagePullSecrets` in `k8s/03-deployment-prebuilt.yaml`.

## 5. GitHub Actions build

Push this repo to GitHub. The workflow publishes:

```txt
ghcr.io/taalib25/runtime-node-core:<tag>
ghcr.io/taalib25/runtime-node-core:sha-<commit>
```

Run manually from Actions with tag `0.1.0`, or push a tag like:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 6. Test on Kind

Replace the image in `k8s/03-deployment-prebuilt.yaml`:

```yaml
image: ghcr.io/taalib25/runtime-node-core:0.1.0
```

For local image without GHCR:

```bash
make build TAG=0.1.0
kind load docker-image runtime-node-core:0.1.0 --name hermes-test
```

Then set image to:

```yaml
image: runtime-node-core:0.1.0
imagePullPolicy: IfNotPresent
```

Apply:

```bash
kubectl apply -f k8s/
kubectl -n hermes-test get pods -w
```

Port-forward:

```bash
kubectl -n hermes-test port-forward svc/hermescloud-runtime-test 8787:8787
```

Open:

```txt
http://localhost:8787
```

## 7. Test on your test server

Use the same manifests first, or let your bridge create the deployment.

For manual smoke test:

```bash
kubectl apply -f k8s/
kubectl -n hermes-test get pods -w
kubectl -n hermes-test port-forward svc/hermescloud-runtime-test 8787:8787
```

For a remote server, SSH tunnel:

```bash
ssh -L 8787:127.0.0.1:8787 root@YOUR_SERVER_IP
```

Then run port-forward on the server:

```bash
kubectl -n hermes-test port-forward svc/hermescloud-runtime-test 8787:8787
```

Your local browser/frontend uses:

```txt
http://localhost:8787
```

## 8. Bridge integration

Support two image modes.

### Current test mode

```json
{
  "runtimeMode": "webui",
  "runtimeImageMode": "init-install",
  "image": "ghcr.io/ashneil12/hermes-webui",
  "imageTag": "stable"
}
```

Bridge behavior:

```txt
add install-agent init container
HERMES_WEBUI_AGENT_DIR=/home/hermeswebui/.hermes/hermes-agent
```

### Prebuilt mode

```json
{
  "runtimeMode": "webui",
  "runtimeImageMode": "prebuilt",
  "image": "ghcr.io/taalib25/runtime-node-core",
  "imageTag": "0.1.0"
}
```

Bridge behavior:

```txt
no install-agent init container
HERMES_WEBUI_AGENT_DIR=/opt/hermes-agent
same PVCs
same package-manager env vars
same service port 8787
```

See `bridge/bridge_snippets.go`.

## 9. Debug checklist

```bash
kubectl -n hermes-test get pods
kubectl -n hermes-test describe pod -l app=hermescloud-runtime-test
kubectl -n hermes-test logs deployment/hermescloud-runtime-test -c hermes-runtime
kubectl -n hermes-test exec -it deploy/hermescloud-runtime-test -c hermes-runtime -- hermescloud-diagnose
```

Inside pod checks:

```bash
echo $HERMES_HOME
echo $HERMES_WEBUI_AGENT_DIR
python -c "import run_agent; print('ok')"
ls -la /opt/hermes-agent
ls -la /home/hermeswebui/.hermes
ls -la /workspace
```

## 10. Persistence checklist

```txt
[ ] WebUI opens
[ ] Chat works
[ ] /health returns 200
[ ] /workspace file survives pod restart
[ ] /home/hermeswebui/.hermes/bin tool survives pod restart
[ ] npm/pip-installed user tool survives pod restart
[ ] Image rollout keeps sessions/workspace
[ ] Runtime starts without install-agent init container
```

## 11. Security boundaries

This image is safer than the boot-install version because runtime code is image-owned.

Still do not put these in the runtime pod:

```txt
KUBECONFIG
cluster admin token
bridge signing secret
database URL
billing secret
cloud provider token
global platform API key
```

The pod can have user-owned provider keys during testing, but platform secrets should stay in the bridge/backend.

## 12. Production note

Both upstream sources track `latest`/`main`:

```txt
HERMES_AGENT_IMAGE=nousresearch/hermes-agent:latest
HERMES_WEBUI_REF=main
```

Rollback is a normal Helm/image rollback while PVC data stays intact. The runtime image itself is tagged by git SHA (`sha-<commit>`) on each CI build for traceability.
