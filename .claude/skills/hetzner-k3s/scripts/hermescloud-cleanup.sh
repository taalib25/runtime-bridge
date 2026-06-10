#!/usr/bin/env bash
# hermescloud-cleanup — node-level containerd image + log hygiene
#
# Runs ON a k3s node (installed to /usr/local/bin/hermescloud-cleanup by the
# cleanup install block that render-cluster-config.sh embeds into each cluster
# config's additional_post_k3s_commands). It is a HOST concern — it must never
# run inside the bridge pod, which stays a constrained control-plane component.
#
# What it does (safely):
#   - Removes dangling <none> containerd image records (the records pin layers;
#     `ctr content gc` alone never reclaims them — this is the historical leak).
#   - Removes non-bridge images not used by any container (old runtimes, an idle
#     agent image — which re-pulls from the registry on next use).
#   - Keeps the newest N hermes-bridge images for fast rollback (+ any in-use one).
#   - Never removes an image referenced by an existing container.
#   - Caps the systemd journal and (optionally) cleans the apt cache.
#   - Writes a machine-readable status file that validate-cluster.sh can read.
#
# Config: /etc/hermescloud-cleanup.conf (sourced if present). Dry run:
#   hermescloud-cleanup --dry-run   OR   DRY_RUN=true in the config.
set -uo pipefail

# ── Config defaults (override in /etc/hermescloud-cleanup.conf) ────────────────
KEEP_BRIDGE_IMAGES=2
JOURNAL_MAX_SIZE=200M
CLEAN_APT=true
BRIDGE_IMAGE_MATCH="hermes-bridge"
STATUS_FILE=/var/log/hermescloud-cleanup.status
DRY_RUN=false

[ -f /etc/hermescloud-cleanup.conf ] && . /etc/hermescloud-cleanup.conf
[ "${1:-}" = "--dry-run" ] && DRY_RUN=true

CRICTL="k3s crictl"
START_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

log()  { echo "[hermescloud-cleanup] $(date -u +%Y-%m-%dT%H:%M:%SZ) $*"; }
dry()  { [ "$DRY_RUN" = "true" ]; }

# ── Select removable images (logic in python3 — present on every k3s node) ─────
# Emits, one per line: "<image-id> <reason> <repoTag-or-none>"
select_removable() {
  local images containers
  images=$($CRICTL images -o json 2>/dev/null)   || { log "WARN: crictl images failed"; return 0; }
  containers=$($CRICTL ps -a -o json 2>/dev/null) || containers='{"containers":[]}'

  KEEP_BRIDGE_IMAGES="$KEEP_BRIDGE_IMAGES" BRIDGE_MATCH="$BRIDGE_IMAGE_MATCH" \
  python3 - "$images" "$containers" <<'PY'
import json, os, subprocess, sys

images = json.loads(sys.argv[1]).get("images", [])
containers = json.loads(sys.argv[2]).get("containers", [])
keep_n = int(os.environ["KEEP_BRIDGE_IMAGES"])
bridge_match = os.environ["BRIDGE_MATCH"]

# Image refs an existing container points at — never remove these.
in_use = set()
for c in containers:
    ref = c.get("imageRef") or c.get("imageId") or ""
    if ref:
        in_use.add(ref)
    img = (c.get("image") or {}).get("image") if isinstance(c.get("image"), dict) else c.get("image")
    if img:
        in_use.add(img)

def is_in_use(img):
    if img["id"] in in_use:
        return True
    for d in (img.get("repoDigests") or []):
        if d in in_use:
            return True
    for t in (img.get("repoTags") or []):
        if t in in_use:
            return True
    return False

def created_ns(img_id):
    # Only called for the few bridge images, to order keep-newest-N.
    try:
        out = subprocess.run(["k3s", "crictl", "inspecti", "-o", "json", img_id],
                             capture_output=True, text=True, timeout=20)
        info = json.loads(out.stdout)
        c = (info.get("status") or {}).get("createdAt") or \
            (((info.get("info") or {}).get("imageSpec") or {}).get("created"))
        return c or ""
    except Exception:
        return ""

bridge_kept = set()
bridge_imgs = [i for i in images if any(bridge_match in (t or "") for t in (i.get("repoTags") or []))]
bridge_imgs.sort(key=lambda i: created_ns(i["id"]), reverse=True)
for i in bridge_imgs[:keep_n]:
    bridge_kept.add(i["id"])

for img in images:
    if is_in_use(img):
        continue
    if img["id"] in bridge_kept:
        continue
    tags = img.get("repoTags") or []
    if not tags:
        reason = "dangling"
        tag = "<none>"
    elif any(bridge_match in (t or "") for t in tags):
        reason = "bridge-beyond-keep-n"
        tag = tags[0]
    else:
        reason = "unused"
        tag = tags[0]
    print(f'{img["id"]} {reason} {tag}')
PY
}

# ── 1. Image cleanup ──────────────────────────────────────────────────────────
log "=== cleanup start (dry_run=$DRY_RUN keep_bridge=$KEEP_BRIDGE_IMAGES) ==="
removed=0; reclaim_failed=0
while read -r id reason tag; do
  [ -n "$id" ] || continue
  if dry; then
    log "[dry-run] would remove $reason $tag ($id)"
  else
    if $CRICTL rmi "$id" >/dev/null 2>&1; then
      log "removed $reason $tag ($id)"
      removed=$((removed + 1))
    else
      reclaim_failed=$((reclaim_failed + 1))
    fi
  fi
done < <(select_removable)

# Reclaim now-orphaned content blobs.
if dry; then log "[dry-run] would run: k3s ctr content gc"
else k3s ctr content gc >/dev/null 2>&1 || true; fi

# ── 2. Journald cap ───────────────────────────────────────────────────────────
if dry; then log "[dry-run] would run: journalctl --vacuum-size=$JOURNAL_MAX_SIZE"
else journalctl --vacuum-size="$JOURNAL_MAX_SIZE" >/dev/null 2>&1 || true; fi

# ── 3. apt cache (optional) ───────────────────────────────────────────────────
if [ "$CLEAN_APT" = "true" ]; then
  if dry; then log "[dry-run] would run: apt-get clean"
  else apt-get clean >/dev/null 2>&1 || true; fi
fi

# ── 4. Status file ────────────────────────────────────────────────────────────
DISK=$(df -h / | awk 'NR==2{print $5" used, "$4" free"}')
if ! dry; then
  {
    echo "last_run=$START_TS"
    echo "finished=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "images_removed=$removed"
    echo "remove_failures=$reclaim_failed"
    echo "disk=$DISK"
    echo "dry_run=false"
  } > "$STATUS_FILE" 2>/dev/null || log "WARN: could not write $STATUS_FILE"
fi

log "=== done — removed=$removed disk: $DISK ==="
