# ADR 0003: Terraform Owns Cluster DNS, Bridge Owns Agent Lifecycle

## Status

Accepted.

## Context

Cluster DNS and customer runtime DNS are different concerns. Terraform can create stable cluster-level records, while customer agent routes are dynamic and created when users create agents.

## Decision

Terraform creates cluster-level DNS records:

- `bridge-<cluster>.hermeshq.net`
- `*.runtime-<cluster>.hermeshq.net`

The bridge creates per-agent Kubernetes Ingress resources under the wildcard runtime domain.

## Consequences

Benefits:

- Cluster bootstrap has stable DNS outputs.
- Agents can be created dynamically without Terraform applies.
- Backend registration can use stable HTTPS bridge URLs.

Costs:

- TLS/certificate strategy must be explicitly implemented.
- Wildcard DNS and WebSocket behavior must be validated for terminal sessions.

## Non-Goals

- Do not create one Terraform DNS record per customer agent.
- Do not register raw HTTP node IP bridge URLs in production.
