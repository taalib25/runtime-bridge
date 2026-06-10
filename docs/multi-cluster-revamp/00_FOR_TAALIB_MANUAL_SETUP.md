# What Taalib Must Manually Configure

This file is for Taalib. It lists the manual setup required before the automated multi-cluster pipeline can work.

## End Result

After setup, creating a new runtime cluster should be a controlled GitHub Actions workflow:

```txt
Run GitHub workflow
  -> Terraform creates kube-hetzner cluster
  -> Terraform creates Cloudflare DNS
  -> Workflow deploys bridge
  -> Workflow validates bridge
  -> Workflow registers cluster in backend
  -> Admin dashboard shows cluster
```

## 1. Decide Naming

Use short cluster IDs. Do not use names like `runtime-runtime-eu-1`.

Recommended:

```txt
kh-test = first kube-hetzner test cluster
eu-1    = first production runtime cluster
eu-2    = second production runtime cluster
```

Domain convention:

```txt
Bridge:
  bridge-kh-test.hermeshq.net
  bridge-eu-1.hermeshq.net
  bridge-eu-2.hermeshq.net

Agent wildcard runtime domains:
  *.runtime-kh-test.hermeshq.net
  *.runtime-eu-1.hermeshq.net
  *.runtime-eu-2.hermeshq.net
```

Example agent URL:

```txt
agent-abc123.runtime-eu-1.hermeshq.net
```

## 2. Keep Existing Cluster Alive

Do not delete the current `hetzner-k3s` cluster yet.

```txt
Current cluster:
  keep running
  no Terraform ownership

New kh-test cluster:
  created with kube-hetzner
  no real users
```

Rule:

```txt
Do not manage the same cluster with both hetzner-k3s and Terraform.
```

## 3. Create Hetzner Cloud API Token

In Hetzner Cloud Console:

1. Open the Hetzner project used for HermesCloud runtime infrastructure.
2. Go to Security/API Tokens.
3. Create token with Read & Write permissions.
4. Save as GitHub secret:

```txt
HCLOUD_TOKEN
```

This token allows Terraform to create servers, networks, load balancers, volumes, firewalls, and related resources.

## 4. Create Cloudflare API Token

In Cloudflare:

1. Go to Profile/API Tokens.
2. Create a scoped token.
3. Required permissions:

```txt
Zone:DNS:Edit
Zone:Zone:Read
```

4. Scope it only to the `hermeshq.net` zone.
5. Save as GitHub secret:

```txt
CLOUDFLARE_API_TOKEN
```

Also get the zone ID for `hermeshq.net` and save:

```txt
CLOUDFLARE_ZONE_ID
```

## 5. Decide TLS Strategy

Choose one for V1:

### Recommended V1: Cloudflare Edge TLS + Origin HTTP inside cluster

```txt
User/backend -> Cloudflare HTTPS -> cluster ingress -> bridge service HTTP
```

This is acceptable only if Cloudflare is always in front of the bridge domain and raw IP access is blocked by firewall/security group rules.

### Stronger V1.5: Cloudflare Edge TLS + origin cert / cert-manager

```txt
User/backend -> Cloudflare HTTPS -> cluster ingress HTTPS -> bridge service
```

This is stronger but more setup. It can be added after kh-test if Cloudflare Edge TLS is validated first.

Manual requirement:

```txt
Set Cloudflare SSL/TLS mode intentionally.
Do not leave production bridge domains serving raw HTTP.
Do not register http://NODE_IP bridge URLs in production.
```

## 6. Configure OpenTofu Remote State

**Decision: OpenTofu + Cloudflare R2 (free tier), one state key per cluster.**

Do not use HCP Terraform / Terraform Cloud. Do not use Hetzner Object Storage. Do not use a single shared state file.

Steps:

1. In Cloudflare dashboard, go to R2 → Create bucket named `hermes-tofu-state`.
2. Go to R2 → Manage R2 API Tokens → Create token with Object Read & Write permissions scoped to `hermes-tofu-state`.
3. Note your Cloudflare Account ID (shown in R2 dashboard).
4. Endpoint: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`
5. Add these as GitHub secrets in every cluster environment:

```txt
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_ENDPOINT          (https://<ACCOUNT_ID>.r2.cloudflarestorage.com)
```

State key pattern (one per cluster):

```txt
clusters/kh-test/tofu.tfstate
clusters/eu-1/tofu.tfstate
clusters/eu-2/tofu.tfstate
```

Locking: OpenTofu native `use_lockfile = true` — no DynamoDB required.

Validate: confirm concurrent `tofu apply` runs are blocked before any production cluster.

## 7. GitHub Environments

Create GitHub Environments:

```txt
kh-test
eu-1
eu-2
```

Recommended protection:

```txt
kh-test:
  no approval required

eu-1/eu-2:
  require manual approval before terraform apply
```

Environment secrets:

```txt
HCLOUD_TOKEN
R2_ACCESS_KEY_ID
R2_SECRET_ACCESS_KEY
R2_ENDPOINT
CLOUDFLARE_API_TOKEN
CLOUDFLARE_ZONE_ID
ADMIN_API_SECRET
BRIDGE_AUTH_PUBLIC_KEY
```

Only put `BRIDGE_AUTH_PRIVATE_KEY` where the backend/build truly needs it. The bridge runtime should not receive the private key.

## 8. Backend Registration Token

Create a strong internal token for GitHub Actions to register a cluster with the backend.

Secret name:

```txt
ADMIN_API_SECRET
```

Backend endpoint (real, existing):

```txt
POST /api/admin/clusters
Authorization: Bearer $ADMIN_API_SECRET
Content-Type: application/json

{
  "cluster_id":    "<cluster-name>",
  "bridge_url":    "https://bridge-<cluster>.hermeshq.net",
  "bridge_secret": "<BRIDGE_SECRET>",
  "region":        "eu",
  "status":        "active"
}
```

Only GitHub Actions should use this. The call is idempotent — safe to re-run.

## 9. Bridge Auth Keys

Move away from one shared `X-Bridge-Secret` per cluster.

Recommended model:

```txt
Backend signs short-lived JWTs with private key.
Bridge verifies JWTs with public key.
```

You need:

```txt
BRIDGE_AUTH_PRIVATE_KEY -> backend/control plane only
BRIDGE_AUTH_PUBLIC_KEY  -> deployed to every bridge
```

## 10. Backup Storage

Choose backup destination before paid users:

```txt
Option A: Cloudflare R2
Option B: S3-compatible object storage
Option C: Hetzner Storage Box, if integration is simpler for you
```

Manual requirement:

```txt
Create bucket/storage.
Create access credentials.
Add credentials to GitHub/backend/backup job secrets according to implementation.
```

Do not assume Hetzner server snapshots protect agent PVC data.

## 11. What You Should Not Do Manually

Do not manually create agent pods.

Do not manually create per-agent DNS records.

Do not manually edit runtime cluster state in Hetzner UI after Terraform owns it.

Do not destroy old hetzner-k3s until kh-test and eu-1/eu-2 are proven.

## 12. Launch Readiness Checklist

Before real users:

```txt
[ ] kh-test provisioned by Terraform
[ ] Cloudflare DNS created by Terraform
[ ] Bridge deployed to kh-test
[ ] Bridge /healthz passes
[ ] Bridge /readyz passes
[ ] Bridge /version visible
[ ] Backend registers kh-test
[ ] Admin dashboard shows kh-test
[ ] Create test agent through backend -> bridge
[ ] Inject Hermes YAML config
[ ] Inject Telegram/provider Secret
[ ] Terminal works into agent container only
[ ] PVC persists after restart
[ ] Delete/purge cleans namespace/PVC according to plan
[ ] Backup job creates backup
[ ] Restore test succeeds
[ ] Operation timeout works
[ ] Capacity reservation works under concurrent creates
```
