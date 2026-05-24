#!/usr/bin/env bash
set -euo pipefail
kubectl -n hermes-test create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="${GITHUB_USERNAME:?set GITHUB_USERNAME}" \
  --docker-password="${GITHUB_TOKEN:?set GITHUB_TOKEN}" \
  --docker-email="${GITHUB_EMAIL:-you@example.com}" \
  --dry-run=client -o yaml | kubectl apply -f -
