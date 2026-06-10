# Storage, Backups, Restore, and Cleanup

## Purpose

This document defines how active agent data, backups, and cleanup work in the kube-hetzner multi-cluster runtime.

## Storage Model

```txt
Active agent data:
  one PVC per agent runtime

Raw uploaded files/artifacts:
  object storage if possible

Backups:
  S3/R2/Storage Box outside the runtime cluster
```

Default:

```txt
1 agent = 1 PVC
```

Do not use object storage as the live agent filesystem.

## Hetzner CSI vs Longhorn

### V1 Recommendation

Use Hetzner CSI-backed PVCs first.

Why:

```txt
- simpler
- matches one-agent-one-PVC model
- less operational complexity
```

### Later Longhorn Evaluation

Longhorn can provide replicated Kubernetes-native storage, but it adds complexity.

Only evaluate when:

```txt
- single-volume/node limitations become painful
- restore/migration requirements increase
- team can operate Longhorn safely
```

Do not start with Longhorn unless there is a strong reason.

## Backup Contract

A backup is valid only if restore has been tested.

V1 backup mode:

```txt
maintenance-window backup
```

Meaning:

```txt
1. Mark agent backup operation running.
2. Temporarily pause/restart/quiet the agent if required for filesystem consistency.
3. Snapshot/copy PVC data using backup job.
4. Upload archive to S3/R2/Storage Box.
5. Store backup metadata in backend.
6. Resume agent.
7. Mark backup succeeded or failed.
```

Do not promise live zero-downtime backups until tested.

## Backup Metadata Table

```sql
create table agent_backups (
  id text primary key,
  agent_id text not null,
  cluster_id text not null,
  status text not null,
  storage_provider text not null,
  object_key text not null,
  size_bytes bigint,
  checksum text,
  started_at timestamp,
  finished_at timestamp,
  expires_at timestamp,
  error_code text,
  message text,
  created_at timestamp not null default now()
);
```

Statuses:

```txt
running
succeeded
failed
expired
restoring
```

## Restore Contract

Restore must support at least:

```txt
restore latest backup into same agent after stopping runtime
```

Later:

```txt
restore into new agent
cross-cluster restore
point-in-time restore
```

V1 restore flow:

```txt
1. Put agent in maintenance/restoring state.
2. Stop agent pod.
3. Attach or mount target PVC through restore job.
4. Download backup archive.
5. Verify checksum.
6. Replace PVC content.
7. Start agent.
8. Run health check.
9. Mark restore succeeded/failed.
```

## Backup Retention

Suggested product policy:

```txt
Free:
  no automatic backup or very short retention

Pro:
  daily backup, 7-day retention

Power:
  daily backup, 14/30-day retention
```

Do not sell backup guarantees until restore proof exists.

## Cleanup Model

Terraform should not perform nightly cleanup.

Cleanup should be a runtime/node maintenance process:

```txt
Option A:
  node-level systemd timer if cleanly supported by kube-hetzner/MicroOS bootstrap

Option B:
  controlled privileged DaemonSet/CronJob if MicroOS/systemd approach is too awkward
```

For kube-hetzner/MicroOS, coding agent must research the safest supported option.

Cleanup targets:

```txt
unused containerd images
dangling <none> image records
old logs/journal size
temporary backup/restore workspace
old failed job artifacts
```

Cleanup must include:

```txt
dry-run mode
last-run status
reclaimed bytes
errors
validation check
```

## Cleanup Visibility

Admin dashboard should show:

```txt
cluster_id
node_name
last cleanup run
status
reclaimed_mb
errors
```

If cleanup is not implemented yet, validation should clearly show:

```txt
cleanup status: not configured
```

Do not silently assume kubelet garbage collection is enough for Hermes-specific image churn.

## Delete/Purge Behavior

When user deletes agent:

```txt
soft delete:
  stop runtime, keep backup/PVC according to retention

purge=true:
  delete namespace/Helm release/PVC/Secrets/Ingress
  keep billing/audit records
  optionally keep final backup if product policy says so
```

Bridge diagnostics should confirm no leftover namespace/PVC after purge.

## Cross-Cluster Migration

Not V1.

For V1:

```txt
new agents route to new clusters
existing agents stay on assigned cluster
manual backup/restore based migration only after restore works
```
