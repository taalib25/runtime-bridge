# HermesCloud Runtime Platform Glossary

This glossary defines the words the coding agent must use consistently. It intentionally separates product language from Kubernetes implementation language.

## Core Terms

### Control Plane
The HermesCloud product layer that owns users, billing, plans, agents, runtime cluster registry, runtime operations, support visibility, and routing decisions.

### Runtime Cluster
A Kubernetes cluster that runs Hermes agent runtimes for customers.

### Data Plane
The runtime execution layer where customer agents actually run.

### Cluster Factory
The infrastructure process that creates or updates runtime clusters and their base networking/DNS. In the new path this is Terraform/OpenTofu + kube-hetzner + Cloudflare provider.

### Bootstrap Pipeline
The post-cluster process that deploys the bridge, validates cluster readiness, and registers the runtime cluster with the control plane.

### Bridge
The per-runtime-cluster service that accepts HermesCloud runtime commands and applies them inside its own runtime cluster.

The bridge is not the product backend. The bridge is not a central super-admin service for all clusters. The bridge only controls its own runtime cluster.

### Agent Runtime
A customer-owned Hermes agent execution environment with persistent data, configuration, secrets, runtime URL, and lifecycle operations.

### Runtime Operation
A tracked action that changes or inspects an agent runtime, such as create, update config, update secrets, restart, delete, backup, restore, diagnostics, or terminal.

### Capacity Reservation
A short-lived backend record that reserves estimated cluster resources before an agent create operation is sent to a bridge. It prevents concurrent creates from over-assigning the same apparent headroom.

### Active Runtime Data
The mutable filesystem data used by a running agent runtime. This lives on a PVC.

### Backup
A restorable copy of active runtime data stored outside the runtime cluster. A backup is not considered valid until restore has been tested.

### Support Diagnostics
The admin-facing process of classifying why a customer agent is broken and recommending the next action.

## Canonical Language

Use:

- **runtime cluster**, not “server” when discussing Kubernetes-level capacity.
- **control plane**, not just “backend” when discussing product ownership.
- **bridge**, not “operator” unless referring to Kubernetes controller semantics.
- **agent runtime**, not “pod” when discussing customer-facing product behavior.
- **cluster factory**, not “provision script” for the Terraform/kube-hetzner path.
- **bootstrap pipeline**, not “deployment script” for DNS/bridge/validation/backend registration.
- **capacity reservation**, not “headroom lock.”

## Layer Boundary

```txt
Terraform/kube-hetzner = infrastructure state
Cloudflare provider    = cluster DNS state
GitHub Actions         = bootstrap execution
Bridge                 = runtime execution inside one cluster
Backend/control plane  = product state and routing
Admin dashboard        = support visibility
```
