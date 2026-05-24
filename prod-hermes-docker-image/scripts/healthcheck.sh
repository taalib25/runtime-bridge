#!/usr/bin/env bash
set -euo pipefail
PORT="${HERMES_WEBUI_PORT:-8787}"
python - <<PY
import urllib.request, sys
url = "http://127.0.0.1:${PORT}/health"
try:
    with urllib.request.urlopen(url, timeout=3) as r:
        if 200 <= r.status < 500:
            sys.exit(0)
        sys.exit(1)
except Exception:
    sys.exit(1)
PY
