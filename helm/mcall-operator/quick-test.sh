#!/bin/bash
# Helm Chart Quick Validation Script

set -e

CHART_DIR="."
VALUES_FILE="${1:-values-dev.yaml}"

echo "🔍 Starting Helm Chart Validation..."
echo "Chart: ${CHART_DIR}"
echo "Values: ${VALUES_FILE}"
echo ""

# 1. Lint
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1️⃣  Running Lint..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if helm lint ${CHART_DIR} -f ${VALUES_FILE}; then
  echo "✅ Lint Passed"
else
  echo "❌ Lint Failed"
  exit 1
fi
echo ""

# 2. Template Rendering
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2️⃣  Rendering Template..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if helm template test-release ${CHART_DIR} \
  -f ${VALUES_FILE} \
  > /tmp/helm-test-output.yaml; then
  echo "✅ Rendering Complete"
  echo "📄 Output: /tmp/helm-test-output.yaml"
  
  # Show file size
  SIZE=$(wc -c < /tmp/helm-test-output.yaml)
  echo "📊 Size: ${SIZE} bytes"
else
  echo "❌ Rendering Failed"
  exit 1
fi
echo ""

# 3. Resource Count
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3️⃣  Generated Kubernetes Resources:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
grep -E "^kind:" /tmp/helm-test-output.yaml | sort | uniq -c | while read count kind; do
  echo "  ${count}x ${kind}"
done
echo ""

# 4. Operator Resource Check
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4️⃣  Operator Resource Check:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if grep -q "tz-mcall-operator" /tmp/helm-test-output.yaml; then
  echo "✅ Operator Deployment Included"
  
  # Check Operator image
  OPERATOR_IMAGE=$(grep -A 5 "name: controller" /tmp/helm-test-output.yaml | grep "image:" | head -1 | awk '{print $2}')
  if [ -n "$OPERATOR_IMAGE" ]; then
    echo "   Image: ${OPERATOR_IMAGE}"
  fi
else
  echo "❌ Operator Resources Not Found"
fi
echo ""

# 5. MCP Server Check
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5️⃣  MCP Server Resource Check:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if grep -q "mcp-server" /tmp/helm-test-output.yaml; then
  echo "✅ MCP Server Resources Included"
  
  # Check MCP image
  MCP_IMAGE=$(grep -A 5 "name: mcp-server" /tmp/helm-test-output.yaml | grep "image:" | head -1 | awk '{print $2}')
  if [ -n "$MCP_IMAGE" ]; then
    echo "   Image: ${MCP_IMAGE}"
  fi
  
  # Check Ingress
  if grep -q "mcp-server.*Ingress" /tmp/helm-test-output.yaml; then
    echo "✅ MCP Server Ingress Included"
    MCP_HOST=$(grep -A 10 "kind: Ingress" /tmp/helm-test-output.yaml | grep "host:" | grep "mcp" | head -1 | awk '{print $2}')
    if [ -n "$MCP_HOST" ]; then
      echo "   Host: ${MCP_HOST}"
    fi
  else
    echo "⚠️  MCP Server Ingress Not Found"
  fi
else
  echo "⚠️  MCP Server Resources Not Found"
  echo "💡 mcpServer.enabled may be set to false"
fi
echo ""

# 6. RBAC Check
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6️⃣  RBAC Resource Check:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SA_COUNT=$(grep -c "kind: ServiceAccount" /tmp/helm-test-output.yaml || echo "0")
ROLE_COUNT=$(grep -c "kind: ClusterRole" /tmp/helm-test-output.yaml || echo "0")
BINDING_COUNT=$(grep -c "kind: ClusterRoleBinding" /tmp/helm-test-output.yaml || echo "0")

echo "  ServiceAccount: ${SA_COUNT}"
echo "  ClusterRole: ${ROLE_COUNT}"
echo "  ClusterRoleBinding: ${BINDING_COUNT}"
echo ""

# 7. CRD Check
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "7️⃣  CRD Check:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if grep -q "kind: CustomResourceDefinition" /tmp/helm-test-output.yaml; then
  echo "✅ CRDs Included"
  grep "name: mcall" /tmp/helm-test-output.yaml | grep "tz.io" | while read line; do
    CRD_NAME=$(echo $line | awk '{print $2}')
    echo "   - ${CRD_NAME}"
  done
else
  echo "⚠️  CRDs Not Found (May be installed separately)"
fi
echo ""

# Final Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 Validation Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "You can review the full output with:"
echo "  cat /tmp/helm-test-output.yaml"
echo "  code /tmp/helm-test-output.yaml"
echo ""
echo "To test actual deployment:"
echo "  helm install test-release . -f ${VALUES_FILE} --namespace test --create-namespace"
echo ""

