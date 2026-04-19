#!/bin/bash

set -e

echo "Testing Hermes Runtime Operator..."

# Configuration
OPERATOR_URL="http://localhost:8080"
SHARED_SECRET="test-secret"

# Test 1: Health check
echo "Test 1: Health check"
curl -s -o /dev/null -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  "$OPERATOR_URL/healthz" | grep -q "200" && echo "✓ PASS" || echo "✗ FAIL"

# Test 2: Create runtime
echo "Test 2: Create runtime"
RUNTIME_ID="test-$(date +%s)"
curl -s -o /tmp/create-response.json -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "test-tenant",
    "runtimeId": "'$RUNTIME_ID'",
    "image": "nginx:alpine",
    "plan": {"cpu": "100m", "memory": "128Mi"},
    "storage": {"size": "100Mi"},
    "network": {"ingress": false}
  }' \
  "$OPERATOR_URL/runtimes" > /tmp/create-status.txt

if grep -q "201" /tmp/create-status.txt; then
  echo "✓ PASS"
  cat /tmp/create-response.json | jq '.'
else
  echo "✗ FAIL"
  cat /tmp/create-status.txt
fi

# Test 3: Get runtime
echo "Test 3: Get runtime"
curl -s -o /tmp/get-response.json -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  "$OPERATOR_URL/runtimes/$RUNTIME_ID" > /tmp/get-status.txt

if grep -q "200" /tmp/get-status.txt; then
  echo "✓ PASS"
  cat /tmp/get-response.json | jq '.phase'
else
  echo "✗ FAIL"
  cat /tmp/get-status.txt
fi

# Test 4: List runtimes
echo "Test 4: List runtimes"
curl -s -o /tmp/list-response.json -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  "$OPERATOR_URL/runtimes" > /tmp/list-status.txt

if grep -q "200" /tmp/list-status.txt; then
  echo "✓ PASS"
  cat /tmp/list-response.json | jq '.runtimes | length'
else
  echo "✗ FAIL"
  cat /tmp/list-status.txt
fi

# Test 5: Get runtime health
echo "Test 5: Get runtime health"
curl -s -o /tmp/health-response.json -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  "$OPERATOR_URL/runtimes/$RUNTIME_ID/health" > /tmp/health-status.txt

if grep -q "200" /tmp/health-status.txt; then
  echo "✓ PASS"
  cat /tmp/health-response.json | jq '.healthy'
else
  echo "✗ FAIL"
  cat /tmp/health-status.txt
fi

# Test 6: Delete runtime
echo "Test 6: Delete runtime"
curl -s -o /tmp/delete-response.json -w "%{http_code}" \
  -H "X-Controller-Secret: $SHARED_SECRET" \
  -X DELETE "$OPERATOR_URL/runtimes/$RUNTIME_ID" > /tmp/delete-status.txt

if grep -q "202" /tmp/delete-status.txt; then
  echo "✓ PASS"
  cat /tmp/delete-response.json | jq '.message'
else
  echo "✗ FAIL"
  cat /tmp/delete-status.txt
fi

echo "Testing completed!"