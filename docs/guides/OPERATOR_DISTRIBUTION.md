# Operator Distribution Guide

This repo supports three complementary distribution/install layers.

## 1. Kustomize Installer

Use when you want a plain YAML install artifact.

```sh
make build-installer IMG=<registry>/hermes-runtime-operator:<tag>
kubectl apply -f dist/install.yaml
```

Best for:

- internal installs
- GitOps
- direct Kubernetes application

## 2. Helm Chart

Use when your users already standardize on Helm.

```sh
make helm-lint
make helm-template
make chart-package
make helm-deploy IMG=<registry>/hermes-runtime-operator:<tag>
```

Important:

- this Helm chart installs the **operator**
- it does **not** replace the runtime lifecycle model
- tenant runtimes should still be created via `Runtime` CRs or the compatibility API

## 3. Operator SDK Bundle + OLM Catalog

Use when you want broad operator distribution and OLM-compatible packaging.

```sh
make bundle IMG=<registry>/hermes-runtime-operator:<tag> VERSION=0.1.0 CHANNELS=alpha DEFAULT_CHANNEL=alpha
make bundle-validate
make scorecard
make catalog
```

Outputs:

- `bundle/`
- `bundle.Dockerfile`
- `catalog/index.yaml`

Optional image builds:

```sh
make bundle-build BUNDLE_IMG=<registry>/hermes-runtime-operator-bundle:<tag>
make catalog-build CATALOG_IMG=<registry>/hermes-runtime-operator-catalog:<tag>
```

## How They Work Together

- **Operator install**: Kustomize or Helm or OLM
- **Runtime lifecycle**: always operator-driven (`Runtime` CR)
- **Backend control path**: Hono -> compatibility API -> `Runtime` CR -> reconciler

## Recommended Default

For now:

1. install the operator with **Helm** or `dist/install.yaml`
2. manage tenant runtimes through the **compatibility API**
3. keep **bundle/catalog** as the enterprise-ready distribution path

This gives you simple installs today and broad distribution later, without changing the core runtime control model.
