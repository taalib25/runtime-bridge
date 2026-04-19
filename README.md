# Hermes Runtime Operator

Hermes Runtime Operator is a Go operator built with Kubebuilder/controller-runtime that manages Hermes runtime workloads through a `Runtime` custom resource.

It provides two layers:

- **Kubernetes operator core**: CRD, reconciler, status, owned resources, lifecycle
- **backend compatibility API**: HTTP endpoints that a Hono.js backend can call for runtime and secret lifecycle operations

## What It Manages

For each `Runtime` resource, the controller reconciles:

- `ConfigMap`
- managed `Secret`
- `PersistentVolumeClaim`
- single-replica `Deployment`
- `Service`
- `Ingress`

The operator is the source of truth for tenant runtime infrastructure. The backend should not patch Deployments, Services, Ingresses, or pods directly.

## Core Contract

- API group: `hermes.hermeshq.net/v1alpha1`
- Kind: `Runtime`
- Reconciler: `RuntimeReconciler`
- Compatibility API: implemented in `internal/api/server.go`

See `DEPLOYMENT.md` for the exact runtime API contract and local-against-prod run mode.

## Installation Modes

This repo supports three install/distribution layers:

1. **Kustomize installer**
2. **Helm chart** for installing the operator itself
3. **Operator SDK bundle + file-based catalog** for OLM-style distribution

Helm is used to install the operator. Helm is **not** the per-tenant runtime lifecycle mechanism.

## Prerequisites

- Go 1.24+
- Docker
- kubectl
- Helm 3
- Access to a Kubernetes cluster (Kind for local validation is recommended)

## Main Development Commands

### Core operator development

```sh
make manifests
make generate
make test
make test-e2e
make build
```

### Installer / packaging

```sh
make build-installer IMG=<operator-image>
make bundle IMG=<operator-image> VERSION=0.1.0 CHANNELS=alpha DEFAULT_CHANNEL=alpha
make bundle-validate
make scorecard
make catalog
make helm-lint
make helm-template
make chart-package
make verify-packaging
```

## Kustomize Install Path

Generate a single install artifact:

```sh
make build-installer IMG=<registry>/hermes-runtime-operator:<tag>
```

This writes:

- `dist/install.yaml`

Install with:

```sh
kubectl apply -f dist/install.yaml
```

## Helm Install Path

The generated operator chart lives at:

- `charts/chart`

Validate it:

```sh
make helm-lint
make helm-template
make chart-package
```

Install the operator with Helm:

```sh
make helm-deploy IMG=<registry>/hermes-runtime-operator:<tag>
```

This installs the **operator**, not tenant runtimes.

## OLM / Bundle Path

Generate the bundle:

```sh
make bundle IMG=<registry>/hermes-runtime-operator:<tag> VERSION=0.1.0 CHANNELS=alpha DEFAULT_CHANNEL=alpha
```

Validate the bundle:

```sh
make bundle-validate
make scorecard
```

Generate a file-based catalog:

```sh
make catalog
```

This produces:

- `bundle/`
- `bundle.Dockerfile`
- `catalog/index.yaml`

Optional images:

```sh
make bundle-build BUNDLE_IMG=<registry>/hermes-runtime-operator-bundle:<tag>
make catalog-build CATALOG_IMG=<registry>/hermes-runtime-operator-catalog:<tag>
```

## How Helm and OLM Fit Together

- **Helm**: best for direct installation by users/teams who already use Helm
- **Bundle/OLM**: best for broader operator distribution, catalog-based installs, and long-term packaging metadata
- **Runtime lifecycle**: always goes through the operator via `Runtime` CRs / compatibility API

Do not use Helm as the primary per-tenant lifecycle tool once the operator is installed.

## Backend Integration

The backend should talk to the operator API, not to Kubernetes resources directly.

Supported compatibility endpoints include:

- `POST /runtimes`
- `PUT /runtimes/{id}`
- `GET /runtimes`
- `GET /runtimes/{id}`
- `GET /runtimes/{id}/health`
- `GET /runtimes/{id}/secrets`
- `POST /runtimes/{id}/secrets`
- `DELETE /runtimes/{id}/secrets/{key}`
- `DELETE /runtimes/{id}`

Examples live in:

- `docs/examples/hono-runtime-client.ts`
- `docs/examples/hono-routes.ts`

## Local Against Production

You can run the operator locally against a production kubeconfig before deploying the manager in-cluster.

```sh
go build -o /tmp/runtime-operator ./cmd

CONTROLLER_SHARED_SECRET=local-test-secret \
RUNTIME_NAMESPACE=hermes-prod \
/tmp/runtime-operator \
  --leader-elect=false \
  --metrics-bind-address=0 \
  --api-bind-address=127.0.0.1:18080 \
  --kubeconfig /path/to/prod-kubeconfig
```

## Recommended Product Model

- Use **Helm** or `dist/install.yaml` to install the operator
- Use **Runtime CRs** (or the compatibility API) to manage tenant runtimes
- Use **bundle + catalog** for broader/enterprise/operator-ecosystem distribution

## Key Docs

- `DEPLOYMENT.md` — runtime API and local/prod flow
- `HERMES_PRODUCT_INTEGRATION_MASTER.md` — product architecture guidance
- `docs/examples/hono-runtime-client.ts` — exact backend client example
- `docs/examples/hono-routes.ts` — Hono integration example
