# Production Sweep (GHCR + K3s)

## 1) Image Naming and Registry

- Public-safe abstract image name: `runtime-node-core`
- Target registry path: `ghcr.io/taalib25/runtime-node-core:<tag>`
- Image is published with `sha-<commit>` tag per CI build for traceability; `latest` is also pushed.

## 2) Secrets and Keys

- Do not hardcode API keys in manifests.
- Configure runtime keys in `k8s/01-secret.yaml`:
  - `OPENCODE_GO_API_KEY`
  - `OPENAI_API_KEY`
  - `ANTHROPIC_API_KEY`
  - `OPENROUTER_API_KEY`
- If GHCR package is private, create pull secret `ghcr-pull-secret` and reference via `imagePullSecrets`.

## 3) Runtime Provider Defaults

Current default provider/model wiring in deployment:

- `HERMES_INFERENCE_PROVIDER=opencode-go`
- `HERMES_MODEL=opencode-go/qwen3.6-plus`
- `HERMES_WEBUI_DEFAULT_MODEL=opencode-go/qwen3.6-plus`
- `HERMES_BASE_URL=https://opencode.ai/zen/go`

These are deployment-level env values and can be overridden per environment.

## 4) K3s Deploy Requirements

- Namespace exists (`hermes-test` in sample manifests).
- PVCs bound:
  - `hermes-state-test`
  - `hermes-workspace-test`
- Deployment + service applied from `k8s/`.
- Probes succeed on `/health` at port `8787`.

## 5) Network Exposure

- Internal: service `hermescloud-runtime-test:8787`
- Local test: `kubectl port-forward --address 0.0.0.0 svc/hermescloud-runtime-test 8787:8787`
- K3s external: expose through Ingress/Traefik or NodePort (avoid NodePort for production internet exposure unless required).

## 6) API Surface Expectations

- This image/runtime currently serves WebUI and `/health`.
- Several `/api/*` endpoints previously tested returned `404`; treat those as unavailable on this runtime build unless implemented by your upstream WebUI/backend version.
- Production validation should focus on:
  - WebUI loads
  - chat flow works end-to-end with provider key
  - state/session persistence across pod restart

## 7) Publish + Pull Flow

1. Build local image with tag.
2. Push to GHCR.
3. Update deployment image to GHCR tag.
4. Apply manifests to K3s.
5. Verify pod, logs, health, chat, persistence.
