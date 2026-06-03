#!/usr/bin/env bash
# run-on-cluster.sh — run a command or script on all nodes (or one node)
#
# Wraps hetzner-k3s run with safety confirmation for destructive commands.
#
# Usage:
#   HCLOUD_TOKEN=xxx ./run-on-cluster.sh <config-file> --command "df -h"
#   HCLOUD_TOKEN=xxx ./run-on-cluster.sh <config-file> --script ./fix.sh
#   HCLOUD_TOKEN=xxx ./run-on-cluster.sh <config-file> --command "hostname" --instance master-fsn1-1
set -euo pipefail

CONFIG="${1:?Usage: run-on-cluster.sh <config-file> --command <cmd> [--instance <name>]}"
shift

[ -n "${HCLOUD_TOKEN:-}" ] || { echo "Error: HCLOUD_TOKEN not set"; exit 1; }
[ -f "$CONFIG" ]           || { echo "Error: config not found: $CONFIG"; exit 1; }

CMD=""; SCRIPT_FILE=""; INSTANCE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --command)  CMD="$2";         shift 2 ;;
    --script)   SCRIPT_FILE="$2"; shift 2 ;;
    --instance) INSTANCE="$2";    shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

[ -n "$CMD" ] || [ -n "$SCRIPT_FILE" ] || { echo "Error: --command or --script required"; exit 1; }
[ -n "$CMD" ] && [ -n "$SCRIPT_FILE" ] && { echo "Error: --command and --script are mutually exclusive"; exit 1; }

CLUSTER_NAME=$(grep 'cluster_name:' "$CONFIG" | awk '{print $2}' | tr -d '"')
TARGET="${INSTANCE:-ALL NODES}"

# Warn on commands that look destructive
DISPLAY="${CMD:-$(basename "$SCRIPT_FILE")}"
if echo "$DISPLAY" | grep -qiE "rm |reboot|shutdown|poweroff|mkfs|dd |format|delete|destroy|purge"; then
  echo "WARNING: Potentially destructive command detected: $DISPLAY"
  read -rp "Type cluster name to confirm running on $TARGET ($CLUSTER_NAME): " confirm
  [ "$confirm" = "$CLUSTER_NAME" ] || { echo "Aborted."; exit 0; }
fi

echo "==> Running on $CLUSTER_NAME / $TARGET: $DISPLAY"

ARGS=(--config "$CONFIG")
[ -n "$CMD"         ] && ARGS+=(--command "$CMD")
[ -n "$SCRIPT_FILE" ] && ARGS+=(--script  "$SCRIPT_FILE")
[ -n "$INSTANCE"    ] && ARGS+=(--instance "$INSTANCE")

hetzner-k3s run "${ARGS[@]}"
