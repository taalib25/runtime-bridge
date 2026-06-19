#!/usr/bin/env bash
# Fails CI if the rendered hermes-agent manifest would resolve to a mutable image
# reference or an implicit/Always pull policy for the customer runtime container.
#
# Kubernetes only defaults imagePullPolicy to Always when the field is omitted AND
# the tag is ":latest" or absent — so this must check the RENDERED manifest, not
# just values.yaml, since an omitted field in values.yaml still renders an explicit
# field if the template always sets one (or doesn't, which is exactly the bug this
# guards against).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT_DIR/charts/hermes-agent"
BRIDGE_CONFIG="$ROOT_DIR/bridge/bridge_config.go"

# Single source of truth for the pinned digest is bridge_config.go's default — extract
# it rather than duplicating the value here, so this check can never silently drift
# from what real deployments actually use.
DIGEST="$(grep -oE 'RuntimeNodeCoreImageDigest:\s*"sha256:[a-f0-9]{64}"' "$BRIDGE_CONFIG" | grep -oE 'sha256:[a-f0-9]{64}' || true)"

if [ -z "$DIGEST" ]; then
  echo "FAIL: bridge_config.go's RuntimeNodeCoreImageDigest default is empty — every instance falls back to the floating :latest tag with no pin at all."
  exit 1
fi

VALUES_FILE="$(mktemp)"
trap 'rm -f "$VALUES_FILE"' EXIT
cat > "$VALUES_FILE" <<EOF
apiServer:
  enabled: true
service:
  enabled: true
secrets:
  API_SERVER_KEY: ci-check-placeholder
ingressRoute:
  enabled: false
ingress:
  hosts:
    - host: ci-check.example.com
# Mirrors what the bridge actually passes at install time (bridge/helm.go preferring
# RuntimeNodeCoreImageDigest over the floating tag) — checking the chart's own bare
# values.yaml defaults alone wouldn't catch what real deployments resolve to.
image:
  digest: "$DIGEST"
EOF

RENDERED_FILE="$(mktemp)"
trap 'rm -f "$VALUES_FILE" "$RENDERED_FILE"' EXIT
helm template ci-check "$CHART_DIR" -f "$VALUES_FILE" > "$RENDERED_FILE"

DIGEST="$DIGEST" python3 - "$RENDERED_FILE" <<'PYEOF'
import os
import sys
import yaml

digest = os.environ["DIGEST"]
path = sys.argv[1]

with open(path) as f:
    docs = list(yaml.safe_load_all(f))

fail = False
checked = 0

for doc in docs:
    if not doc or doc.get("kind") not in ("Deployment", "StatefulSet", "DaemonSet"):
        continue
    pod_spec = doc["spec"]["template"]["spec"]
    for kind in ("initContainers", "containers"):
        for c in pod_spec.get(kind, []):
            checked += 1
            name = c.get("name", "<unnamed>")
            policy = c.get("imagePullPolicy")
            image = c.get("image", "")
            if policy is None:
                print(f"FAIL: {doc['kind']}/{doc['metadata']['name']} container {name!r} has no explicit imagePullPolicy (Kubernetes would default it to Always for an unpinned image).")
                fail = True
            elif policy == "Always":
                print(f"FAIL: {doc['kind']}/{doc['metadata']['name']} container {name!r} uses imagePullPolicy: Always — must be IfNotPresent so pre-pulled/cached images are reused.")
                fail = True
            if image.endswith(":latest"):
                print(f"FAIL: {doc['kind']}/{doc['metadata']['name']} container {name!r} uses a :latest tag ({image!r}). Pin an immutable tag or digest.")
                fail = True
            elif f"@{digest}" not in image:
                print(f"FAIL: {doc['kind']}/{doc['metadata']['name']} container {name!r} does not reference the pinned digest {digest} ({image!r}) — the hermes-agent.image helper may have regressed.")
                fail = True

if checked == 0:
    print("FAIL: no containers found in the rendered manifest — check didn't actually verify anything.")
    fail = True

if not fail:
    print(f"OK: {checked} container(s) all pin {digest} with an explicit imagePullPolicy and no :latest tags.")

sys.exit(1 if fail else 0)
PYEOF
