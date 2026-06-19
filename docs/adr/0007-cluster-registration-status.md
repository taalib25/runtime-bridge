# ADR 0007 — New clusters register as `draining`, promoted to `active` after smoke-test

**Date:** 2026-06-12  
**Status:** Accepted

## Context

The bootstrap pipeline registered new clusters with `status: "maintenance"`. The cluster status enum only knows `active`, `draining`, `disabled`. The unrecognised value silently fell through `mapRegistrationStatus` to `active`, making the cluster immediately routable before any smoke-test passed.

The handoff contract requires clusters to be excluded from routing until validated.

## Decision

New clusters register with `status: "draining"`. `draining` is an existing status that excludes the cluster from new instance routing (`clusterEligibilityFilter` requires `active`). After the bootstrap pipeline smoke-test passes, an operator promotes the cluster to `active` via the admin API. No new status enum value is introduced.

## Alternatives considered

**Add a `maintenance` cluster status** — would require schema migration, enum change, and routing filter update. `draining` already has the same routing semantics. Rejected as unnecessary complexity.

## Consequences

- Bootstrap pipeline sends `status: "draining"` (was `"maintenance"`).
- `mapRegistrationStatus` requires no change — `draining` is already handled.
- Cluster is invisible to the scheduler until explicitly promoted.
- The bridge-level Maintenance Mode flag (separate concept — `PUT /v1/cluster/maintenance`) is unaffected.
