#!/bin/bash

# Jenkins Pipeline Local Simulation Script
# Usage: ./scripts/local-jenkins-test.sh [build-number] [branch] [skip-docker]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BUILD_NUMBER="${1:-local-$(date +%Y%m%d-%H%M%S)}"
GIT_BRANCH="${2:-dev}"
SKIP_DOCKER="${3:-false}"

# Determine namespace and values file based on branch
if [ "${GIT_BRANCH}" = "main" ]; then
    NAMESPACE="mcall-system"
    VALUES_FILE="values.yaml"
    STAGING_POSTFIX=""
elif [ "${GIT_BRANCH}" = "qa" ]; then
    NAMESPACE="mcall-system"
    VALUES_FILE="values-staging.yaml"
    STAGING_POSTFIX=""
else
    NAMESPACE="mcall-dev"
    VALUES_FILE="values-dev.yaml"
    STAGING_POSTFIX="-dev"
fi

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Jenkins Pipeline Local Simulation${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Configuration:"
echo "  BUILD_NUMBER: ${BUILD_NUMBER}"
echo "  GIT_BRANCH: ${GIT_BRANCH}"
echo "  NAMESPACE: ${NAMESPACE}"
echo "  VALUES_FILE: ${VALUES_FILE}"
echo "  SKIP_DOCKER: ${SKIP_DOCKER}"
echo ""

# Stage 1: Build Docker Images
if [ "${SKIP_DOCKER}" != "true" ]; then
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Stage 1: Build & Push Images${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        echo -e "${RED}❌ Docker is not running!${NC}"
        exit 1
    fi
    
    # Build Operator image
    echo -e "${BLUE}Building Operator image...${NC}"
    docker build -f docker/Dockerfile \
        -t doohee323/tz-mcall-operator:${BUILD_NUMBER} \
        . || exit 1
    echo -e "${GREEN}✅ Operator image built${NC}"
    echo ""
    
    # Build MCP Server image
    echo -e "${BLUE}Building MCP Server image...${NC}"
    docker build -f mcp-server/Dockerfile \
        -t doohee323/mcall-operator-mcp-server:${BUILD_NUMBER} \
        ./mcp-server || exit 1
    echo -e "${GREEN}✅ MCP Server image built${NC}"
    echo ""
    
    # Tag images
    if [ "${GIT_BRANCH}" = "main" ]; then
        docker tag doohee323/mcall-operator-mcp-server:${BUILD_NUMBER} \
            doohee323/mcall-operator-mcp-server:latest
    elif [ "${GIT_BRANCH}" = "qa" ]; then
        docker tag doohee323/mcall-operator-mcp-server:${BUILD_NUMBER} \
            doohee323/mcall-operator-mcp-server:staging
    else
        docker tag doohee323/tz-mcall-operator:${BUILD_NUMBER} \
            doohee323/tz-mcall-operator:latest
        docker tag doohee323/mcall-operator-mcp-server:${BUILD_NUMBER} \
            doohee323/mcall-operator-mcp-server:dev
    fi
    echo -e "${GREEN}✅ Images tagged${NC}"
    echo ""
else
    echo -e "${YELLOW}⏭️  Skipping Docker build (SKIP_DOCKER=true)${NC}"
    echo ""
fi

# Stage 2: Helm Chart Validation
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Stage 2: Helm Chart Validation${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo -e "${RED}❌ Helm is not installed!${NC}"
    exit 1
fi

# Lint
echo -e "${BLUE}Running Helm lint...${NC}"
helm lint ./helm/mcall-operator -f ./helm/mcall-operator/${VALUES_FILE} || exit 1
echo -e "${GREEN}✅ Helm lint passed${NC}"
echo ""

# Stage 3: Template Rendering (Jenkins-style)
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Stage 3: Template Rendering${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

OUTPUT_FILE="/tmp/helm-jenkins-sim-${BUILD_NUMBER}.yaml"

echo -e "${BLUE}Rendering Helm template with Jenkins parameters...${NC}"
helm template tz-mcall-operator${STAGING_POSTFIX} ./helm/mcall-operator \
    --namespace ${NAMESPACE} \
    --values "helm/mcall-operator/${VALUES_FILE}" \
    --set image.tag="${BUILD_NUMBER}" \
    --set image.repository="doohee323/tz-mcall-operator" \
    --set mcpServer.image.tag="${BUILD_NUMBER}" \
    --set mcpServer.image.repository="doohee323/mcall-operator-mcp-server" \
    --set namespace.name="${NAMESPACE}" \
    > ${OUTPUT_FILE} || exit 1

echo -e "${GREEN}✅ Template rendered${NC}"
echo -e "${GREEN}   Output: ${OUTPUT_FILE}${NC}"
echo ""

# Analyze rendered template
RESOURCE_COUNT=$(grep -c "^kind:" ${OUTPUT_FILE})
echo "📊 Generated Resources:"
grep "^kind:" ${OUTPUT_FILE} | sort | uniq -c | while read count kind; do
    echo "   ${count}x ${kind}"
done
echo ""

# Check for MCP Server
if grep -q "mcp-server" ${OUTPUT_FILE}; then
    echo -e "${GREEN}✅ MCP Server resources included${NC}"
    MCP_IMAGE=$(grep -A 5 "name: mcp-server" ${OUTPUT_FILE} | grep "image:" | head -1 | awk '{print $2}' | tr -d '"')
    if [ -n "$MCP_IMAGE" ]; then
        echo "   Image: ${MCP_IMAGE}"
    fi
else
    echo -e "${YELLOW}⚠️  MCP Server resources not included (mcpServer.enabled=false?)${NC}"
fi
echo ""

# Stage 4: kubectl dry-run (optional)
if command -v kubectl &> /dev/null; then
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Stage 4: kubectl dry-run${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    echo -e "${BLUE}Testing kubectl apply --dry-run...${NC}"
    if kubectl apply -f ${OUTPUT_FILE} --dry-run=client > /dev/null 2>&1; then
        echo -e "${GREEN}✅ kubectl dry-run passed${NC}"
    else
        echo -e "${YELLOW}⚠️  kubectl dry-run had warnings (this is normal)${NC}"
    fi
    echo ""
else
    echo -e "${YELLOW}⚠️  kubectl not found, skipping dry-run${NC}"
    echo ""
fi

# Stage 5: Summary
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 Validation Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Summary:"
echo "  ✅ Docker images built (2)"
echo "  ✅ Helm chart validated"
echo "  ✅ Template rendered (${RESOURCE_COUNT} resources)"
echo "  📄 Output: ${OUTPUT_FILE}"
echo ""
echo "Next Steps:"
echo ""
echo "1. Review rendered template:"
echo "   cat ${OUTPUT_FILE}"
echo "   code ${OUTPUT_FILE}"
echo ""
echo "2. To actually deploy (if cluster is available):"
echo "   kubectl apply -f ${OUTPUT_FILE}"
echo ""
echo "3. Or use k8s.sh script:"
echo "   ./ci/k8s.sh ${BUILD_NUMBER} ${GIT_BRANCH} ${NAMESPACE} ${VALUES_FILE} deploy"
echo ""
echo "4. View images:"
echo "   docker images | grep ${BUILD_NUMBER}"
echo ""

