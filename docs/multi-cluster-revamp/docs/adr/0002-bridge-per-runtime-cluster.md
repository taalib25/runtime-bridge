# ADR 0002: Use One Bridge Per Runtime Cluster

## Status

Accepted.

## Context

HermesCloud could use a single central bridge that holds access to all runtime clusters, or a separate bridge in each runtime cluster.

## Decision

Run one bridge per runtime cluster.

The HermesCloud control plane routes operations to the correct bridge based on the agent's cluster assignment.

## Consequences

Benefits:

- Smaller blast radius.
- Cluster access remains local to each runtime cluster.
- Terminal/log/diagnostic traffic is local to the cluster.
- One cluster failure does not take down every cluster.

Costs:

- Bridge deployments must be rolled out to multiple clusters.
- The admin dashboard must show bridge version/health per cluster.

## Non-Goals

- Do not build a central super-bridge holding kubeconfigs for every cluster.
- Do not expose raw Kubernetes APIs to the HermesCloud backend for customer operations.
