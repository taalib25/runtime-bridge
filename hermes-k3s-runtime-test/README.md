# Manual K3s Runtime Test

This folder contains raw Kubernetes manifests for direct runtime testing without Helm.

**It is not the production deployment path.**

Production runtime instances are created by:

```
Bridge → Helm SDK → charts/runtime-node-core/
```

Use this folder only for fast manual smoke tests when you need to verify a new
runtime image works before going through the full Helm chart path.

**Do not delete this folder.** Do not add production features here.

## Usage

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-secret.yaml      # edit with real API keys first
kubectl apply -f 02-pvc.yaml
kubectl apply -f 03-deployment.yaml
kubectl apply -f 04-service.yaml

kubectl get pods -n hermes-runtime-test
kubectl logs -n hermes-runtime-test -l app=hermes-runtime --tail=50
```
