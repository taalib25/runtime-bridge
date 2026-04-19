#!/bin/bash

set -e

DEPLOY_DIR="/tmp/hermes-runtime-operator-deploy"
mkdir -p $DEPLOY_DIR

cp config/crd/bases/runtime.hermescloud.io_runtimes.yaml $DEPLOY_DIR/
cp config/rbac/role.yaml $DEPLOY_DIR/
cp config/rbac/service_account.yaml $DEPLOY_DIR/
cp config/rbac/role_binding.yaml $DEPLOY_DIR/
cp config/manager/manager.yaml $DEPLOY_DIR/

cat > $DEPLOY_DIR/namespace.yaml << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: hermes-runtime-operator-system
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: hermes-runtime-operator
    app.kubernetes.io/managed-by: kustomize
EOF

docker run --rm -v $DEPLOY_DIR:/manifests -v /home/taalib/.kube:/root/.kube lachlanevenson/k8s-kubectl:latest apply -f /manifests/

echo "Deployment completed successfully!"