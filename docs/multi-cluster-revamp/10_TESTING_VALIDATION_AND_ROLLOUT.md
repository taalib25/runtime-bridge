# Testing, Validation, Rollout, and Rollback

## Purpose

This document defines the proof required before using kube-hetzner clusters for real users.

## Migration Rule

```txt
Do not destroy the existing hetzner-k3s cluster until kube-hetzner clusters are proven.
```

## kh-test Proof Checklist

`kh-test` must pass before eu-1/eu-2.

```txt
[ ] Terraform remote state configured
[ ] terraform plan succeeds
[ ] terraform apply creates cluster
[ ] Cloudflare DNS records created
[ ] bridge deployed into hermes-system
[ ] bridge /healthz passes
[ ] bridge /readyz passes
[ ] bridge /version returns expected SHA
[ ] /v1/cluster/summary returns sane data
[ ] /v1/cluster/resources returns sane data
[ ] backend registers cluster as maintenance
[ ] admin dashboard shows kh-test
[ ] admin can set kh-test active/maintenance
```

## Agent Lifecycle Proof

```txt
[ ] create test agent through backend
[ ] backend creates capacity reservation
[ ] backend calls bridge
[ ] bridge returns operation_id
[ ] backend polls operation to succeeded
[ ] runtime URL works
[ ] PVC created and Bound
[ ] Secret created
[ ] ConfigMap created if used
[ ] Ingress created
[ ] Service routes to pod
[ ] agent reaches Live
```

## Config/Secret Proof

```txt
[ ] update Hermes YAML config
[ ] update model/provider setting
[ ] update Telegram token Secret
[ ] bridge restarts/reloads agent if needed
[ ] diagnostics confirms runtime healthy
[ ] secrets are not visible in admin logs
```

## Terminal Proof

```txt
[ ] authorized user can open terminal into own agent container
[ ] unauthorized user is denied
[ ] terminal cannot access bridge/system namespace
[ ] terminal session start/end audited
[ ] idle/concurrent limits work if implemented
```

## Diagnostics Proof

Simulate:

```txt
[ ] invalid provider key -> provider_key recommendation
[ ] invalid Telegram token -> messenger recommendation
[ ] forced CrashLoop -> runtime_boot recommendation
[ ] PVC pending simulation or mocked signal -> storage recommendation
[ ] bridge RBAC degraded -> bridge_rbac recommendation
[ ] capacity unavailable -> capacity recommendation
```

## Backup/Restore Proof

```txt
[ ] backup job runs for test agent
[ ] backup metadata stored
[ ] checksum stored
[ ] restore into same agent succeeds
[ ] agent starts after restore
[ ] restore failure leaves clear operation error
```

## Concurrent Create Proof

Test:

```txt
10 concurrent agent create requests
```

Expected:

```txt
capacity reservations prevent over-routing
some creates may queue/fail cleanly if capacity unavailable
no cluster gets over-assigned based on stale headroom
no agent stays creating forever
```

## Bridge Restart During Operation Proof

Test:

```txt
start create operation
restart bridge deployment mid-operation
```

Expected:

```txt
backend operation eventually recovers via polling/diagnostics
operation becomes succeeded, failed, or timeout
no stuck creating forever
reservation released or committed correctly
```

## Rollout to eu-1/eu-2

After kh-test passes:

```txt
1. provision eu-1
2. register as maintenance
3. run smoke test agent
4. run backup/restore smoke test
5. set eu-1 active
6. provision eu-2
7. repeat validation
8. set eu-2 active
```

## Rollback

If kube-hetzner path fails before real users:

```txt
keep current hetzner-k3s cluster as fallback
set kube-hetzner cluster disabled/unhealthy
stop routing new agents there
fix and retry
```

If eu-1 fails after active:

```txt
set eu-1 maintenance or unhealthy
route new agents to eu-2
existing eu-1 agents may be affected until recovery
use backups only if restore has been proven
```

## Old Cluster Retirement

Only retire old hetzner-k3s cluster after:

```txt
[ ] eu-1/eu-2 stable
[ ] admin support dashboard works
[ ] backup/restore works
[ ] no active agents depend on old cluster
[ ] billing/product state reconciled
[ ] final backup taken if needed
```
