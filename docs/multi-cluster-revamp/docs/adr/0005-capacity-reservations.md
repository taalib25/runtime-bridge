# ADR 0005: Use Backend Capacity Reservations for Concurrent Agent Creates

## Status

Accepted.

## Context

If the backend routes agent creates using only bridge headroom polling, concurrent requests can all see the same free capacity and over-assign one cluster.

## Decision

Before creating an agent, backend creates a short-lived capacity reservation in the selected cluster.

Routing subtracts active reservations from reported bridge headroom.

## Consequences

Benefits:

- Prevents over-routing during concurrent creates.
- Makes capacity decisions deterministic enough for V1.
- Keeps routing logic in backend where product plans are known.

Costs:

- Requires reservation cleanup/expiry.
- Requires careful commit/release behavior on operation success/failure.

## Non-Goals

- This is not a full scheduler.
- This does not replace Kubernetes scheduling.
