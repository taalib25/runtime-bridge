# ADR 0004: Use Backend-Signed Short-Lived JWTs for Bridge Auth

## Status

Accepted as target. Shared secret may exist only as temporary compatibility during migration.

## Context

Per-cluster shared bridge secrets create secret sprawl and are risky if accidentally sent over raw HTTP. HermesCloud needs a cleaner backend-to-bridge authentication model.

## Decision

Backend signs short-lived JWTs with a private key. Bridges verify with a public key.

JWT includes cluster, action, operation, agent scope, and short expiry.

## Consequences

Benefits:

- Reduces per-cluster secret sprawl.
- Private signing key stays in backend/control plane.
- Public key can be deployed to every bridge.
- Tokens are short-lived and scoped.

Costs:

- Need key rotation support.
- Need strict claim validation.
- Need replay protection for high-risk actions if required.

## Non-Goals

- Do not let bridges share one long-lived production secret forever.
- Do not accept unsigned internal requests.
