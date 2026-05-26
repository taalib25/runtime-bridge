#!/usr/bin/env bash
set -Eeuo pipefail

log()  { printf '[hermescloud-runtime] %s\n' "$*"; }
fail() { log "ERROR: $*"; /usr/local/bin/hermescloud-diagnose || true; exit 1; }

# ── Canonical env vars (Dockerfile bakes these; shell defaults are a safety net) ──
export HERMES_HOME="${HERMES_HOME:-/home/hermeswebui/.hermes}"
export HERMES_CONFIG_PATH="${HERMES_CONFIG_PATH:-$HERMES_HOME/config.yaml}"
export HERMES_WEBUI_HOST="${HERMES_WEBUI_HOST:-0.0.0.0}"
export HERMES_WEBUI_PORT="${HERMES_WEBUI_PORT:-8787}"
export HERMES_WEBUI_STATE_DIR="${HERMES_WEBUI_STATE_DIR:-$HERMES_HOME/webui}"
export HERMES_WEBUI_DEFAULT_WORKSPACE="${HERMES_WEBUI_DEFAULT_WORKSPACE:-/workspace}"
export HERMES_WEBUI_AGENT_DIR="${HERMES_WEBUI_AGENT_DIR:-/opt/hermes-agent}"
export HERMES_WEBUI_PYTHON="${HERMES_WEBUI_PYTHON:-/opt/hermes-webui/.venv/bin/python}"
export PATH="${PATH:-/home/hermeswebui/.hermes/bin:/home/hermeswebui/.local/bin:/opt/hermes-webui/.venv/bin:/usr/local/bin:/usr/bin:/bin}"

# ── Required writable directories ─────────────────────────────────────────────
# fsGroup=1024 in the k8s SecurityContext makes empty PVCs writable on first mount.
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

# ── Permission checks ──────────────────────────────────────────────────────────
[ -w "$HERMES_HOME" ]                      || fail "$HERMES_HOME is not writable. Check PVC fsGroup/runAsUser."
[ -w "$HERMES_WEBUI_DEFAULT_WORKSPACE" ]   || fail "/workspace is not writable. Check workspace PVC permissions."

# ── Source validation ──────────────────────────────────────────────────────────
[ -d "$HERMES_WEBUI_AGENT_DIR" ]           || fail "Agent source missing: $HERMES_WEBUI_AGENT_DIR"
[ -f "$HERMES_WEBUI_AGENT_DIR/pyproject.toml" ] || fail "Agent pyproject.toml missing at $HERMES_WEBUI_AGENT_DIR"
[ -d /opt/hermes-webui ]                   || fail "WebUI source missing: /opt/hermes-webui"
[ -f /opt/hermes-webui/server.py ]         || fail "WebUI server.py missing: /opt/hermes-webui/server.py"
[ -f "$HERMES_WEBUI_PYTHON" ]              || fail "WebUI Python missing: $HERMES_WEBUI_PYTHON"

log "Starting HermesCloud runtime"
log "user=$(id -u):$(id -g)  home=$HERMES_HOME  port=$HERMES_WEBUI_PORT"

# ── Python validation ──────────────────────────────────────────────────────────
"$HERMES_WEBUI_PYTHON" - <<'PY'
import sys
print('python', sys.version)
try:
    from run_agent import AIAgent
    print('run_agent.AIAgent import ok')
except Exception as exc:
    print('run_agent.AIAgent import failed:', repr(exc))
    raise
PY

# ── Bootstrap persisted config files (only if absent — never overwrite) ────────
_provider="${HERMES_INFERENCE_PROVIDER:-opencode-go}"
_model_with_prefix="${HERMES_WEBUI_DEFAULT_MODEL:-opencode-go/qwen3.6-plus}"
_model_slug="${_model_with_prefix#*/}"   # strip "opencode-go/" prefix if present

if [ ! -f "$HERMES_HOME/auth.json" ]; then
  log "Writing auth.json (provider: $_provider)"
  "$HERMES_WEBUI_PYTHON" -c "
import json
data = {'active_provider': '$_provider', 'providers': {'$_provider': {'authenticated': True}}}
open('$HERMES_HOME/auth.json', 'w').write(json.dumps(data, indent=2))
"
fi

if [ ! -f "$HERMES_CONFIG_PATH" ]; then
  log "Writing config.yaml (model: $_model_slug, provider: $_provider)"
  printf 'model:\n  name: %s\n  provider: %s\nproviders:\n  %s: {}\n' \
    "$_model_slug" "$_provider" "$_provider" > "$HERMES_CONFIG_PATH"
fi

if [ ! -f "$HERMES_HOME/.env" ]; then
  log "Creating empty .env"
  touch "$HERMES_HOME/.env"
fi

# ── Final validation ───────────────────────────────────────────────────────────
[ -f "$HERMES_CONFIG_PATH" ] || fail "config.yaml missing after bootstrap: $HERMES_CONFIG_PATH"
[ -f "$HERMES_HOME/.env" ]   || fail ".env missing after bootstrap: $HERMES_HOME/.env"

# ── Launch WebUI (explicit — no auto-detection) ────────────────────────────────
cd /opt/hermes-webui
exec "$HERMES_WEBUI_PYTHON" server.py
