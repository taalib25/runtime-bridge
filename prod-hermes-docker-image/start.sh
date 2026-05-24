#!/usr/bin/env bash
set -Eeuo pipefail

log() { printf '[hermescloud-runtime] %s\n' "$*"; }
fail() { log "ERROR: $*"; /usr/local/bin/hermescloud-diagnose || true; exit 1; }

export HERMES_HOME="${HERMES_HOME:-/home/hermeswebui/.hermes}"
export HERMES_WEBUI_HOST="${HERMES_WEBUI_HOST:-0.0.0.0}"
export HERMES_WEBUI_PORT="${HERMES_WEBUI_PORT:-8787}"
export HERMES_WEBUI_STATE_DIR="${HERMES_WEBUI_STATE_DIR:-$HERMES_HOME/webui}"
export HERMES_WEBUI_DEFAULT_WORKSPACE="${HERMES_WEBUI_DEFAULT_WORKSPACE:-/workspace}"
export HERMES_WEBUI_AGENT_DIR="${HERMES_WEBUI_AGENT_DIR:-/opt/hermes-agent}"
export PATH="${PATH:-/home/hermeswebui/.hermes/bin:/home/hermeswebui/.local/bin:/opt/hermes-webui/.venv/bin:/usr/local/bin:/usr/bin:/bin}"

# Required writable directories. In Kubernetes, fsGroup=1024 should make empty PVCs writable.
mkdir -p \
  "$HERMES_HOME" \
  "$HERMES_WEBUI_STATE_DIR" \
  "$HERMES_WEBUI_DEFAULT_WORKSPACE" \
  "$HERMES_HOME/bin" \
  "$HERMES_HOME/cache" \
  "$HERMES_HOME/cache/pip" \
  "$HERMES_HOME/cache/npm" \
  "$HERMES_HOME/python" \
  "$HERMES_HOME/npm" \
  "$HERMES_HOME/pnpm" \
  "$HERMES_HOME/uv" \
  "$HERMES_HOME/pipx" \
  "$HERMES_HOME/.config" \
  /tmp/hermescloud

# Fast permission diagnostics before WebUI fails in a confusing way.
[ -w "$HERMES_HOME" ] || fail "$HERMES_HOME is not writable. Check PVC fsGroup/runAsUser."
[ -w "$HERMES_WEBUI_DEFAULT_WORKSPACE" ] || fail "$HERMES_WEBUI_DEFAULT_WORKSPACE is not writable. Check workspace PVC permissions."
[ -d "$HERMES_WEBUI_AGENT_DIR" ] || fail "HERMES_WEBUI_AGENT_DIR does not exist: $HERMES_WEBUI_AGENT_DIR"
[ -f "$HERMES_WEBUI_AGENT_DIR/pyproject.toml" ] || fail "Hermes Agent pyproject.toml missing at $HERMES_WEBUI_AGENT_DIR"

log "Starting HermesCloud runtime"
log "user=$(id -u):$(id -g)"
log "HERMES_HOME=$HERMES_HOME"
log "HERMES_WEBUI_AGENT_DIR=$HERMES_WEBUI_AGENT_DIR"
log "HERMES_WEBUI_DEFAULT_WORKSPACE=$HERMES_WEBUI_DEFAULT_WORKSPACE"
log "HERMES_WEBUI_PORT=$HERMES_WEBUI_PORT"

cd /opt/hermes-webui

python - <<'PY'
import sys
print('python', sys.version)
try:
    import run_agent
    print('run_agent import ok at runtime')
except Exception as exc:
    print('run_agent import failed:', repr(exc))
    raise
PY

# Start the WebUI with common fallback entrypoints. This keeps the image useful if the WebUI repo changes naming.
if [ -f startup.py ]; then
  exec python startup.py
fi

if [ -f server.py ]; then
  exec python server.py
fi

if [ -f app.py ]; then
  exec python app.py
fi

# Some Python projects expose a module entrypoint after pip install -e .
if python -c "import hermes_webui" >/dev/null 2>&1; then
  exec python -m hermes_webui
fi

fail "Could not find a WebUI startup file in /opt/hermes-webui"
