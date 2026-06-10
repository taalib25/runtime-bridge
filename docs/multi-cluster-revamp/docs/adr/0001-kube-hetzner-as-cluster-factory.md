# ADR 0001: Use kube-hetzner/Terraform as the Future Runtime Cluster Factory

## Status

Accepted for new runtime clusters. Existing hetzner-k3s clusters remain untouched until replacement is proven.

## Context

HermesCloud needs repeatable multi-cluster runtime infrastructure on Hetzner. The previous direction used hetzner-k3s plus custom scripts. That worked, but cluster cloning, DNS, bridge bootstrap, cleanup, validation, and backend registration were spread across imperative scripts.

## Decision

Use kube-hetzner/Terraform for new runtime clusters.

Terraform manages infrastructure and cluster-level DNS. The bridge continues to manage customer agent runtimes.

## Consequences

Benefits:

- More declarative cluster lifecycle.
- Clearer infrastructure review through plan/apply.
- Better fit for multi-cluster repeatability.
- Cloudflare DNS can be managed in the same infrastructure state.

Costs:

- Terraform remote state becomes critical.
- GitHub Actions needs stronger safety gates.
- Existing hetzner-k3s clusters should not be converted in place.

## Non-Goals

- Terraform must not create customer agent runtimes.
- Terraform must not replace the bridge.
- kube-hetzner must not be introduced by modifying the existing hetzner-k3s cluster in place.
