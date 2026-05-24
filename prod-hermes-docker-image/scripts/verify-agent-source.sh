#!/usr/bin/env bash
set -euo pipefail
IMAGE="${1:-nousresearch/hermes-agent:latest}"
echo "Verifying Hermes Agent source in $IMAGE"
docker run --rm "$IMAGE" sh -c '
  set -e
  for p in /opt/hermes /app /workspace; do
    if [ -f "$p/pyproject.toml" ]; then
      echo "FOUND:$p"
      ls -la "$p" | head
      exit 0
    fi
  done
  echo "ERROR: no source dir with pyproject.toml found" >&2
  find / -maxdepth 4 \( -name pyproject.toml -o -name "*hermes*" \) 2>/dev/null | head -100 >&2
  exit 1
'
