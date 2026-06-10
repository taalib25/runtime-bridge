# ADR 0006: Start With Maintenance-Window Backups

## Status

Accepted for V1.

## Context

Agent runtimes have mutable filesystem data on PVCs. Live backups can be inconsistent if the agent writes during backup.

## Decision

V1 backup mode is maintenance-window backup: pause/stop/quiet the agent if required, copy PVC data, upload backup, then resume.

A backup is valid only after restore is tested.

## Consequences

Benefits:

- Simpler consistency model.
- Easier to debug restore.
- Avoids over-promising zero-downtime backup.

Costs:

- Backup may cause short runtime interruption.
- Product must communicate backup guarantees carefully.

## Non-Goals

- Do not promise live zero-downtime backups in V1.
- Do not assume Hetzner server snapshots protect PVC data.
