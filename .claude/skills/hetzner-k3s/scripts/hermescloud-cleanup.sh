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
# Runtime/agent images got NO keep-N grace until this was added — an image classified
# "unused" (not referenced by any currently-running container, e.g. a brief gap between
# pod restarts, or a node the pre-puller hasn't reached yet) was removed immediately,
# with no protection for "this is the active runtime image, it'll be needed again in
# seconds". Mirrors KEEP_BRIDGE_IMAGES's logic for the runtime image family.
KEEP_RUNTIME_IMAGES=2
JOURNAL_MAX_SIZE=200M
CLEAN_APT=true
BRIDGE_IMAGE_MATCH="hermes-bridge"
RUNTIME_IMAGE_MATCH="hermes-agent"
# Pinned digest the bridge actually uses for new instances (bridge_config.go's
# RuntimeNodeCoreImageDigest) — when set, this exact image is NEVER removed regardless
# of container-reference state or keep-N counting. The strongest protection available,
# since it doesn't depend on a pre-puller pod currently existing on this node.
PINNED_RUNTIME_DIGEST=""
# Disk usage percent (on /) above which a WARNING is logged instead of cleanup silently
# running as if nothing's wrong — high disk pressure on a node with images intentionally
# protected from removal is something a human should look at, not just absorb forever.
DISK_PRESSURE_WARN_PCT=85
STATUS_FILE=/var/log/hermescloud-cleanup.status
DRY_RUN=false

[ -f /etc/hermescloud-cleanup.conf ] && . /etc/hermescloud-cleanup.conf
[ "${1:-}" = "--dry-run" ] && DRY_RUN=true

CRICTL="k3s crictl"
START_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Auto-resolve PINNED_RUNTIME_DIGEST from the local bridge if not explicitly configured
# — the bridge pod runs on this same node, so this stays in sync with whatever image
# the bridge actually pins without needing a separate config-distribution step. Reads
# the bridge secret straight from the node's k3s secret store (this script already
# runs as root on the node, same trust level as the bridge's own config).
if [ -z "$PINNED_RUNTIME_DIGEST" ]; then
  bsecret=$(KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl -n hermes-bridge get secret bridge-auth \
    -o jsonpath='{.data.secret}' 2>/dev/null | base64 -d 2>/dev/null || true)
  if [ -n "$bsecret" ]; then
    summary=$(curl -sf --max-time 5 -H "X-Bridge-Secret: $bsecret" http://localhost:8080/v1/cluster/summary 2>/dev/null || true)
    if [ -n "$summary" ]; then
      PINNED_RUNTIME_DIGEST=$(echo "$summary" | python3 -c "
import json, sys
try:
    ref = json.load(sys.stdin).get('runtimeImageReference', '') or ''
    print(ref.split('@', 1)[1] if '@' in ref else '')
except Exception:
    pass
" 2>/dev/null || true)
    fi
  fi
fi

log()  { echo "[hermescloud-cleanup] $(date -u +%Y-%m-%dT%H:%M:%SZ) $*"; }
dry()  { [ "$DRY_RUN" = "true" ]; }

# ── Select removable images (logic in python3 — present on every k3s node) ─────
# Emits, one per line: "<image-id> <reason> <repoTag-or-none>"
select_removable() {
  local images containers
  images=$($CRICTL images -o json 2>/dev/null)   || { log "WARN: crictl images failed"; return 0; }
  containers=$($CRICTL ps -a -o json 2>/dev/null) || containers='{"containers":[]}'

  KEEP_BRIDGE_IMAGES="$KEEP_BRIDGE_IMAGES" BRIDGE_MATCH="$BRIDGE_IMAGE_MATCH" \
  KEEP_RUNTIME_IMAGES="$KEEP_RUNTIME_IMAGES" RUNTIME_MATCH="$RUNTIME_IMAGE_MATCH" \
  PINNED_RUNTIME_DIGEST="$PINNED_RUNTIME_DIGEST" \
  python3 - "$images" "$containers" <<'PY'
import json, os, subprocess, sys

images = json.loads(sys.argv[1]).get("images", [])
containers = json.loads(sys.argv[2]).get("containers", [])
bridge_match = os.environ["BRIDGE_MATCH"]
runtime_match = os.environ["RUNTIME_MATCH"]
pinned_digest = os.environ.get("PINNED_RUNTIME_DIGEST", "").strip()

# Image refs an existing container points at — never remove these. Covers the
# pre-puller's own long-sleeping container, so as long as it's running on THIS node,
# the runtime image is already protected without needing the digest check below too —
# but the digest check stays as a second, independent line of defense (e.g. a brief
# window where the pre-puller pod is restarting, or hasn't reached this node yet).
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

def is_pinned_digest(img):
    if not pinned_digest:
        return False
    for d in (img.get("repoDigests") or []):
        if pinned_digest in d:
            return True
    return False

def created_ns(img_id):
    # Only called for the few bridge/runtime images, to order keep-newest-N.
    try:
        out = subprocess.run(["k3s", "crictl", "inspecti", "-o", "json", img_id],
                             capture_output=True, text=True, timeout=20)
        info = json.loads(out.stdout)
        c = (info.get("status") or {}).get("createdAt") or \
            (((info.get("info") or {}).get("imageSpec") or {}).get("created"))
        return c or ""
    except Exception:
        return ""

def keep_newest_n(match, env_key):
    keep_n = int(os.environ[env_key])
    kept = set()
    matched = [i for i in images if any(match in (t or "") for t in (i.get("repoTags") or []))]
    matched.sort(key=lambda i: created_ns(i["id"]), reverse=True)
    for i in matched[:keep_n]:
        kept.add(i["id"])
    return kept

bridge_kept = keep_newest_n(bridge_match, "KEEP_BRIDGE_IMAGES")
runtime_kept = keep_newest_n(runtime_match, "KEEP_RUNTIME_IMAGES")

for img in images:
    if is_pinned_digest(img):
        continue
    if is_in_use(img):
        continue
    if img["id"] in bridge_kept or img["id"] in runtime_kept:
        continue
    tags = img.get("repoTags") or []
    if not tags:
        reason = "dangling"
        tag = "<none>"
    elif any(bridge_match in (t or "") for t in tags):
        reason = "bridge-beyond-keep-n"
        tag = tags[0]
    elif any(runtime_match in (t or "") for t in tags):
        reason = "runtime-beyond-keep-n"
        tag = tags[0]
    else:
        reason = "unused"
        tag = tags[0]
    print(f'{img["id"]} {reason} {tag}')
PY
}

# ── 0. Disk pressure check ─────────────────────────────────────────────────────
# A high-disk-pressure node with protected (in-use/keep-N/pinned-digest) images that
# this script will never remove is something a human should know about, not a state
# this script silently absorbs forever by quietly doing its best each run.
disk_pct_before=$(df -P / | awk 'NR==2{gsub("%","",$5); print $5}')
if [ -n "$disk_pct_before" ] && [ "$disk_pct_before" -ge "$DISK_PRESSURE_WARN_PCT" ] 2>/dev/null; then
  log "WARNING: disk usage at ${disk_pct_before}% (threshold ${DISK_PRESSURE_WARN_PCT}%) — protected images (in-use, keep-N, pinned digest) will NOT be removed even under pressure; this node may need attention (resize, more aggressive keep-N, or investigate what's actually consuming space)."
fi

# ── 1. Image cleanup ──────────────────────────────────────────────────────────
log "=== cleanup start (dry_run=$DRY_RUN keep_bridge=$KEEP_BRIDGE_IMAGES keep_runtime=$KEEP_RUNTIME_IMAGES pinned_digest=${PINNED_RUNTIME_DIGEST:-none}) ==="
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
