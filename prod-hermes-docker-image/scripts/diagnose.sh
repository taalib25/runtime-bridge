#!/usr/bin/env bash
set +e
printf '\n--- HermesCloud Runtime Diagnostics ---\n'
id
pwd
printf '\nENV core:\n'
printenv | grep -E '^(HERMES|PYTHON|PIP|PIPX|UV|NPM|PNPM|YARN|COREPACK|BUN|DENO|CARGO|RUSTUP|GOPATH|GOBIN|GEM|COMPOSER|DOTNET|PATH|HOME|XDG|GH_)=' | sort
printf '\nPaths:\n'
for p in /opt/hermes-agent /opt/hermes-webui "$HERMES_HOME" "$HERMES_WEBUI_STATE_DIR" "$HERMES_WEBUI_DEFAULT_WORKSPACE" /tmp; do
  echo "### $p"
  ls -lad "$p" 2>&1
  ls -la "$p" 2>&1 | head -30
 done
printf '\nPython import:\n'
python - <<'PY'
try:
    import run_agent
    print('run_agent: ok')
except Exception as e:
    print('run_agent: failed', repr(e))
PY
printf '\nProcesses:\n'
ps aux | head -40
printf '\nListening ports:\n'
(lsof -i -P -n || true) | head -80
printf -- '--- End Diagnostics ---\n'
