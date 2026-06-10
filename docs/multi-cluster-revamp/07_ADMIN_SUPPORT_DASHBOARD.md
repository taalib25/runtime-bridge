# Admin Runtime and Support Dashboard

## Purpose

The admin dashboard is the support command center for HermesCloud runtime operations.

It must answer:

```txt
Is the customer agent healthy?
If not, why?
What action should support take?
```

## Required Pages

```txt
/admin/runtime
/admin/runtime/:clusterId
/admin/agents/:agentId/diagnostics
/admin/operations
/admin/support
```

## /admin/runtime

Shows all runtime clusters.

Columns/cards:

```txt
cluster_id
status
health_status
bridge_url
bridge_version
provisioner
region
headroom memory
running pods
pending pods
failed pods
crashlooping pods
active reservations
last_seen_at
last_validated_at
```

Actions:

```txt
refresh
set active
set maintenance
set draining
disable
open bridge health
open Grafana/Portainer/k9s note if configured
```

## /admin/runtime/:clusterId

Sections:

```txt
Cluster summary
Node resources
Bridge version/readiness
Capacity reservations
Recent operations
Failed/pending agents
Validation history
```

Warnings:

```txt
bridge unreachable
readyz degraded
bridge version incompatible
raw http bridge URL registered
pending pods > 0
PVC pending
cleanup failed
backup job failing
```

## /admin/agents/:agentId/diagnostics

Flow:

```txt
Backend loads agent.
Backend reads agent.cluster_id.
Backend calls correct bridge diagnostics endpoint.
Backend renders product-friendly diagnosis.
```

Show:

```txt
agent name
user/workspace
cluster_id
runtime_url
status
last operation
pod/container status
PVC status
latest events
redacted latest logs
recommendation
issue category
```

Actions:

```txt
restart agent
refresh diagnostics
update setup link
open terminal if authorized
mark resolved
create support note
```

## /admin/operations

Show operation timeline.

Columns:

```txt
operation_id
agent
cluster_id
type
status
duration
error_code
message
created_at
```

Operation detail:

```txt
backend operation status
bridge operation id
steps
bridge raw response
reservation id
timeout_at
final result
```

No operation should stay running forever.

## /admin/support

Search by:

```txt
user email
workspace
agent name
agent id
runtime URL
cluster id
operation id
```

Default filters:

```txt
failed agents
stuck creating
unhealthy clusters
recent failed operations
backup failures
```

## Issue Classifier

Map raw signals to support categories:

```txt
Pod Pending + low headroom -> capacity
PVC Pending -> storage
CrashLoopBackOff -> runtime_boot
Readiness failed -> runtime_health
Provider auth error -> provider_key
Telegram/Discord/Slack auth error -> messenger
Ingress/DNS mismatch -> network_dns
Bridge readyz degraded -> bridge_rbac
No clear signal -> unknown
```

## Support Recommendations

Examples:

```txt
capacity:
  "Runtime capacity is temporarily full. Route new agents to another cluster or add capacity."

storage:
  "PVC is pending. Check CSI/storage class and node volume limits."

provider_key:
  "User's provider key needs attention. Ask user to reconnect or update key."

messenger:
  "Messaging integration token is invalid. Ask user to reconnect Telegram/Discord/Slack."

bridge_rbac:
  "Bridge is missing Kubernetes permission. Keep cluster in maintenance until RBAC is fixed."
```

## Logs and Redaction

Admin logs must be redacted before rendering.

Redact:

```txt
API keys
tokens
Authorization headers
Telegram bot tokens
Discord/Slack tokens
provider keys
cookies
private URLs with credentials
```

Add a visible note:

```txt
Logs are redacted before display. Downloading raw logs requires elevated admin permission.
```

## Normal User Status

Normal users should see simple statuses:

```txt
Creating
Starting
Live
Needs setup
Needs reconnection
Capacity delayed
Restarting
Failed
Maintenance
```

Normal users should not see:

```txt
CrashLoopBackOff
PVC Pending
RBAC forbidden
Helm failed
NetworkPolicy
IngressRoute
```
