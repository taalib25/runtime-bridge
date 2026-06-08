#!/usr/bin/env bash
# validate-cleanup.sh — verify the cleanup system is installed + healthy on a node
#
# Fails fast (non-zero) if cleanup is missing, the timer is disabled, the on-node
# script doesn't match the expected version, the status file isn't writable, or a
# dry-run errors. provision-cluster.sh runs this against every node and aborts the
# deploy if it fails.
#
# Usage:
#   validate-cleanup.sh <node-ip>
# Env:
#   SSH_PRIVATE_KEY   optional — key CONTENTS (not path); falls back to default ssh
#   EXPECTED_HASH     optional — sha256 of the canonical script (default: compute
#                     from the sibling hermescloud-cleanup.sh in this repo)
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NODE_IP="${1:?Usage: validate-cleanup.sh <node-ip>}"
EXPECTED_HASH="${EXPECTED_HASH:-$(sha256sum "$SCRIPT_DIR/hermescloud-cleanup.sh" | awk '{print $1}')}"

if [ -n "${SSH_PRIVATE_KEY:-}" ]; then
  remote() { ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i <(echo "$SSH_PRIVATE_KEY") root@"$NODE_IP" "$@"; }
else
  remote() { ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 root@"$NODE_IP" "$@"; }
fi

fail=0
check() { # <description> <remote-command-that-exits-0-on-pass>
  if remote "$2" >/dev/null 2>&1; then echo "  [ok]   $1"; else echo "  [FAIL] $1"; fail=1; fi
}

echo "=== validate-cleanup @ $NODE_IP (expected sha256 ${EXPECTED_HASH:0:12}…) ==="

check "cleanup binary present"      "test -x /usr/local/bin/hermescloud-cleanup"
check "timer enabled"               "systemctl is-enabled hermescloud-cleanup.timer"
check "version file matches script" "test \"\$(sha256sum /usr/local/bin/hermescloud-cleanup | awk '{print \$1}')\" = \"$EXPECTED_HASH\" && test \"\$(cat /etc/hermescloud-cleanup.version 2>/dev/null)\" = \"$EXPECTED_HASH\""
check "status file writable"        "touch /var/log/hermescloud-cleanup.status"
check "dry-run succeeds"            "/usr/local/bin/hermescloud-cleanup --dry-run"

if [ "$fail" -eq 0 ]; then echo "=== cleanup OK on $NODE_IP ==="; else echo "=== cleanup VALIDATION FAILED on $NODE_IP ===" >&2; fi
exit "$fail"
