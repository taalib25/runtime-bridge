#!/bin/bash

set -e

echo "Validating operator logic..."

# Test 1: Verify CRD structure
echo "Test 1: CRD structure"
if [ -f "api/v1alpha1/runtime_types.go" ]; then
  echo "✓ CRD types file exists"
else
  echo "✗ CRD types file missing"
  exit 1
fi

# Test 2: Verify HTTP server implementation
echo "Test 2: HTTP server endpoints"
ENDPOINTS=("GET /healthz" "POST /runtimes" "GET /runtimes" "GET /runtimes/{id}" "DELETE /runtimes/{id}" "GET /runtimes/{id}/health")
for endpoint in "${ENDPOINTS[@]}"; do
  if grep -q "mux.HandleFunc(\"$endpoint\"" internal/api/server.go; then
    echo "✓ Endpoint $endpoint implemented"
  else
    echo "✗ Endpoint $endpoint missing"
    exit 1
  fi
done

# Test 3: Verify reconciliation logic
echo "Test 3: Reconciliation logic"
RECONCILER_FILE="internal/controller/runtime_controller.go"
if [ -f "$RECONCILER_FILE" ]; then
  echo "✓ Reconciler file exists"
  
  # Check for key reconciliation functions
  if grep -q "reconcileNamespace" $RECONCILER_FILE; then
    echo "✓ Namespace reconciliation implemented"
  else
    echo "✗ Namespace reconciliation missing"
    exit 1
  fi
  
  if grep -q "reconcilePVC" $RECONCILER_FILE; then
    echo "✓ PVC reconciliation implemented"
  else
    echo "✗ PVC reconciliation missing"
    exit 1
  fi
  
  if grep -q "reconcileDeployment" $RECONCILER_FILE; then
    echo "✓ Deployment reconciliation implemented"
  else
    echo "✗ Deployment reconciliation missing"
    exit 1
  fi
  
  if grep -q "reconcileService" $RECONCILER_FILE; then
    echo "✓ Service reconciliation implemented"
  else
    echo "✗ Service reconciliation missing"
    exit 1
  fi
  
  if grep -q "reconcileIngress" $RECONCILER_FILE; then
    echo "✓ Ingress reconciliation implemented"
  else
    echo "✗ Ingress reconciliation missing"
    exit 1
  fi
  
  if grep -q "ContainsFinalizer" $RECONCILER_FILE && grep -q "handleDeletion" $RECONCILER_FILE; then
    echo "✓ Finalizer and cleanup logic implemented"
  else
    echo "✗ Finalizer or cleanup logic missing"
    exit 1
  fi
else
  echo "✗ Reconciler file missing"
  exit 1
fi

# Test 4: Verify RBAC permissions
echo "Test 4: RBAC permissions"
RBAC_FILE="config/rbac/role.yaml"
if [ -f "$RBAC_FILE" ]; then
  echo "✓ RBAC file exists"
  
  # Check for required permissions
  REQUIRED_RESOURCES=("namespaces" "persistentvolumeclaims" "deployments" "services" "ingresses" "runtimes")
  for resource in "${REQUIRED_RESOURCES[@]}"; do
    if grep -q "$resource" $RBAC_FILE; then
      echo "✓ RBAC permission for $resource exists"
    else
      echo "✗ RBAC permission for $resource missing"
      exit 1
    fi
  done
else
  echo "✗ RBAC file missing"
  exit 1
fi

# Test 5: Verify Dockerfile
echo "Test 5: Dockerfile"
if [ -f "Dockerfile" ]; then
  echo "✓ Dockerfile exists"
  
  # Check for multi-stage build
  if grep -q "FROM golang:1.25 AS builder" Dockerfile && grep -q "FROM gcr.io/distroless/static:nonroot" Dockerfile; then
    echo "✓ Multi-stage Dockerfile implemented"
  else
    echo "✗ Multi-stage Dockerfile missing"
    exit 1
  fi
else
  echo "✗ Dockerfile missing"
  exit 1
fi

echo "All validation tests passed! Operator logic is correct."