#!/bin/sh

# Kubernetes CRD deployment script
set -e

# Environment variable setup
BUILD_NUMBER=${1:-"latest"}
GIT_BRANCH=${2:-"main"}
NAMESPACE=${3:-"mcall-system"}
VALUES_FILE=${4:-"values-dev.yaml"}
ACTION=${5:-"deploy"}

# Sanitize GIT_BRANCH
GIT_BRANCH=$(echo "${GIT_BRANCH}" | sed 's|^origin/||' | sed 's|_|-|g' | sed 's|/|-|g')

# Namespace setup by branch
if [ "${NAMESPACE}" = "mcall-system" ]; then
    if [ "${GIT_BRANCH}" = "main" ] || [ "${GIT_BRANCH}" = "qa" ]; then
        NAMESPACE="mcall-system"
        STAGING_POSTFIX=""
    else
        NAMESPACE="mcall-dev"
        STAGING_POSTFIX="-dev"
    fi
else
    # If NAMESPACE is explicitly set, determine postfix based on branch
    if [ "${GIT_BRANCH}" = "main" ] || [ "${GIT_BRANCH}" = "qa" ]; then
        STAGING_POSTFIX=""
    else
        STAGING_POSTFIX="-dev"
    fi
fi

# Values file setup by branch
if [ "${VALUES_FILE}" = "values-dev.yaml" ]; then
    if [ "${GIT_BRANCH}" = "main" ]; then
        VALUES_FILE="values.yaml"
    elif [ "${GIT_BRANCH}" = "qa" ]; then
        VALUES_FILE="values-staging.yaml"
    else
        VALUES_FILE="values-dev.yaml"
    fi
fi

echo "🔍 CRD Deployment info:"
echo "BUILD_NUMBER: ${BUILD_NUMBER}"
echo "GIT_BRANCH: ${GIT_BRANCH}"
echo "NAMESPACE: ${NAMESPACE}"
echo "VALUES_FILE: ${VALUES_FILE}"
echo "STAGING_POSTFIX: ${STAGING_POSTFIX}"
echo "ACTION: ${ACTION}"

# Install Helm if not present
install_helm() {
    if ! command -v helm &> /dev/null; then
        echo "📥 Installing Helm..."
        
        # Install to local directory without sudo
        HELM_INSTALL_DIR=/tmp/helm
        mkdir -p $HELM_INSTALL_DIR
        
        # Download and extract Helm
        curl -fsSL https://get.helm.sh/helm-v3.18.6-linux-amd64.tar.gz | tar -xzC $HELM_INSTALL_DIR
        
        # Add to PATH
        export PATH=$PATH:$HELM_INSTALL_DIR/linux-amd64
        
        # Verify installation
        $HELM_INSTALL_DIR/linux-amd64/helm version
    else
        echo "✅ Helm already installed: $(helm version --short 2>/dev/null || helm version 2>/dev/null | head -1)"
    fi
}

# Install kubectl if not present
install_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        echo "📥 Installing kubectl..."
        wget -q https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x ./kubectl
        export PATH=$PATH:./
    else
        echo "✅ kubectl already installed: $(kubectl version --client 2>/dev/null | head -1)"
    fi
}

# Force cleanup all mcall resources
force_cleanup_all_mcall_resources() {
    echo "🧹 Force cleaning up ALL mcall resources..."
    
    # Delete all mcall custom resources first
    echo "Deleting all mcall custom resources..."
    kubectl delete mcalltasks --all --all-namespaces --force --grace-period=0 --timeout=10s || echo "No mcalltasks found"
    kubectl delete mcallworkflows --all --all-namespaces --force --grace-period=0 --timeout=10s || echo "No mcallworkflows found"
    
    # Delete all mcall CRDs
    echo "Deleting all mcall CRDs..."
    for crd in mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io; do
        if kubectl get crd ${crd} >/dev/null 2>&1; then
            echo "Deleting CRD ${crd}..."
            kubectl delete crd ${crd} --force --grace-period=0 --timeout=10s || echo "Failed to delete ${crd}"
        fi
    done
    
    # Force remove finalizers if CRDs still exist
    sleep 3
    for crd in mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io; do
        if kubectl get crd ${crd} >/dev/null 2>&1; then
            echo "Force removing finalizers from ${crd}..."
            kubectl patch crd ${crd} --type json -p='[{"op": "remove", "path": "/metadata/finalizers"}]' || echo "Failed to remove finalizers from ${crd}"
            kubectl delete crd ${crd} --force --grace-period=0 --timeout=5s || echo "Final force deletion failed for ${crd}"
        fi
    done
    
    echo "✅ Force cleanup completed"
}

# Clean up conflicting CRDs and releases
cleanup_conflicting_resources() {
    echo "🧹 Cleaning up conflicting resources..."
    
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    OLD_RELEASE_NAME="mcall-operator${STAGING_POSTFIX}"
    
    # Check for old Helm releases
    if helm list -n ${NAMESPACE} | grep -q ${OLD_RELEASE_NAME}; then
        echo "Found old release ${OLD_RELEASE_NAME}, uninstalling..."
        helm uninstall ${OLD_RELEASE_NAME} -n ${NAMESPACE} --no-hooks --wait --timeout=5m || echo "Old release uninstall failed, continuing..."
    fi
    
    # Check for CRDs with conflicting ownership metadata
    echo "Checking for CRDs with conflicting ownership metadata..."
    for crd in mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io; do
        if kubectl get crd ${crd} >/dev/null 2>&1; then
            RELEASE_NAME=$(kubectl get crd ${crd} -o jsonpath='{.metadata.annotations.meta\.helm\.sh/release-name}' 2>/dev/null || echo "")
            if [ "${RELEASE_NAME}" = "${OLD_RELEASE_NAME}" ] || [ "${RELEASE_NAME}" != "${HELM_RELEASE_NAME}" ]; then
                echo "Found CRD ${crd} with conflicting ownership (${RELEASE_NAME}), removing..."
                
                # First try graceful deletion
                kubectl delete crd ${crd} --timeout=30s || echo "Graceful deletion failed for ${crd}"
                
                # Wait a bit for graceful deletion
                sleep 5
                
                # Check if still exists, then force delete
                if kubectl get crd ${crd} >/dev/null 2>&1; then
                    echo "Force deleting CRD ${crd}..."
                    kubectl delete crd ${crd} --force --grace-period=0 --timeout=10s || echo "Force deletion failed for ${crd}"
                fi
            fi
        fi
    done
    
    # Wait for CRD deletion to complete with timeout
    echo "Waiting for CRD deletion to complete..."
    TIMEOUT=60
    COUNTER=0
    while kubectl get crd | grep -q mcall && [ $COUNTER -lt $TIMEOUT ]; do
        echo "Waiting for CRD deletion... (${COUNTER}/${TIMEOUT}s)"
        sleep 2
        COUNTER=$((COUNTER + 2))
    done
    
    # If CRDs still exist after timeout, force remove finalizers
    if kubectl get crd | grep -q mcall; then
        echo "⚠️  CRD deletion timed out, force removing finalizers..."
        for crd in mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io; do
            if kubectl get crd ${crd} >/dev/null 2>&1; then
                echo "Force removing finalizers from ${crd}..."
                kubectl patch crd ${crd} --type json -p='[{"op": "remove", "path": "/metadata/finalizers"}]' || echo "Failed to remove finalizers from ${crd}"
                kubectl delete crd ${crd} --force --grace-period=0 --timeout=5s || echo "Final force deletion failed for ${crd}"
            fi
        done
        
        # Final wait
        sleep 5
        if kubectl get crd | grep -q mcall; then
            echo "❌ CRD deletion still failed, but continuing with deployment..."
        else
            echo "✅ CRD force deletion completed"
        fi
    else
        echo "✅ CRD deletion completed"
    fi
    
    echo "✅ Conflicting resources cleanup completed"
}

# Deploy CRDs
deploy_crds() {
    echo "🔧 Deploying CRDs..."
    
    # Apply CRD manifests from Helm chart templates
    # Note: Helm doesn't update CRDs on upgrade, so we apply them explicitly
    # For CRDs, we use 'replace --force' to ensure schema updates are applied
    if [ -d "crds" ]; then
        echo "Applying CRDs from crds/"
        
        for crd_file in crds/*.yaml; do
            [ -f "$crd_file" ] || continue
            
            CRD_NAME=$(grep "name:" "$crd_file" | head -1 | awk '{print $2}')
            echo ""
            echo "========================================="
            echo "Processing CRD: $CRD_NAME"
            echo "========================================="
            
            # Check file content for new fields
            echo "📄 Checking CRD file content..."
            if grep -q "inputSources" "$crd_file"; then
                echo "  ✅ File contains 'inputSources' field"
            else
                echo "  ⚠️  File does NOT contain 'inputSources' field"
            fi
            if grep -q "inputTemplate" "$crd_file"; then
                echo "  ✅ File contains 'inputTemplate' field"
            else
                echo "  ⚠️  File does NOT contain 'inputTemplate' field"
            fi
            
            # Apply or replace CRD (avoids API server schema cache issues)
            echo "  📦 Applying CRD with new schema..."
            # Save output to temp file and check exit code
            CREATE_OUTPUT=$(mktemp)
            if kubectl apply -f "$crd_file" > "$CREATE_OUTPUT" 2>&1; then
                # Indent output
                sed 's/^/    /' "$CREATE_OUTPUT"
                rm -f "$CREATE_OUTPUT"
                
                echo "  ✅ Apply succeeded"
                
                # Add Helm metadata to CRD for ownership
                echo "  🏷️  Adding Helm metadata to CRD..."
                HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
                kubectl label crd "$CRD_NAME" app.kubernetes.io/managed-by=Helm --overwrite 2>&1 | sed 's/^/    /'
                kubectl annotate crd "$CRD_NAME" meta.helm.sh/release-name="${HELM_RELEASE_NAME}" --overwrite 2>&1 | sed 's/^/    /'
                kubectl annotate crd "$CRD_NAME" meta.helm.sh/release-namespace="${NAMESPACE}" --overwrite 2>&1 | sed 's/^/    /'
                echo "  ✅ Helm metadata added"
                
                # Wait for API server to process and load new schema
                echo "  ⏳ Waiting for API server to load new CRD schema..."
                sleep 10
                
                # Verify new fields are present
                echo "  🔍 Verifying new CRD..."
                kubectl get crd "$CRD_NAME" -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties}' 2>/dev/null | \
                    python3 -c "import sys, json; data = json.load(sys.stdin); fields = list(data.keys()); print('    Total fields:', len(fields)); print('    Has inputSources:', 'inputSources' in fields); print('    Has inputTemplate:', 'inputTemplate' in fields)" 2>/dev/null || echo "    (Could not verify new fields)"
            else
                # Show error output
                sed 's/^/    /' "$CREATE_OUTPUT"
                rm -f "$CREATE_OUTPUT"
                echo "  ❌ Apply failed for $CRD_NAME"
                return 1
            fi
        done
        
        echo ""
        echo "========================================="
        echo "✅ CRD deployment completed"
        echo "========================================="
        
        # Wait for CRDs to be established
        echo "⏳ Waiting for CRDs to be re-established..."
        sleep 5  # Give k8s API server time to process
        kubectl wait --for condition=established --timeout=60s crd/mcalltasks.mcall.tz.io 2>&1 || echo "⚠️  McallTask CRD not established yet"
        kubectl wait --for condition=established --timeout=60s crd/mcallworkflows.mcall.tz.io 2>&1 || echo "⚠️  McallWorkflow CRD not established yet"
        
        # Final verification with detailed field check
        echo ""
        echo "📋 Final Verification:"
        echo "========================================="
        if kubectl get crd mcalltasks.mcall.tz.io >/dev/null 2>&1; then
            echo "✅ mcalltasks.mcall.tz.io is present"
            kubectl get crd mcalltasks.mcall.tz.io -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties}' 2>/dev/null | \
                python3 -c "import sys, json; data = json.load(sys.stdin); fields = sorted(data.keys()); print('   Total fields:', len(fields)); print('   Has inputSources:', 'inputSources' in fields); print('   Has inputTemplate:', 'inputTemplate' in fields); print('   All fields:', ', '.join(fields))" 2>/dev/null || echo "   (Could not read fields)"
        else
            echo "❌ mcalltasks.mcall.tz.io is MISSING"
        fi
        
        if kubectl get crd mcallworkflows.mcall.tz.io >/dev/null 2>&1; then
            echo "✅ mcallworkflows.mcall.tz.io is present"
            kubectl get crd mcallworkflows.mcall.tz.io -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.tasks.items.properties}' 2>/dev/null | \
                python3 -c "import sys, json; data = json.load(sys.stdin); fields = sorted(data.keys()); print('   Total fields:', len(fields)); print('   Has condition:', 'condition' in fields); print('   Has inputSources:', 'inputSources' in fields); print('   All fields:', ', '.join(fields))" 2>/dev/null || echo "   (Could not read fields)"
        else
            echo "❌ mcallworkflows.mcall.tz.io is MISSING"
        fi
        echo "========================================="
    else
        echo "⚠️  CRD directory not found: crds/"
        echo "CRDs will be installed by Helm chart (first install only)..."
    fi
}

# Deploy Helm chart
deploy_helm_chart() {
    echo "🚀 Deploying Helm chart..."
    
    # Use --set options instead of modifying values file
    # This avoids sed parsing issues with image tags
    
    # Check if namespace exists
    if ! kubectl get namespace ${NAMESPACE} >/dev/null 2>&1; then
        echo "📦 Creating namespace ${NAMESPACE}..."
        kubectl create namespace ${NAMESPACE}
    else
        echo "✅ Namespace ${NAMESPACE} already exists"
    fi
    
    
    # Label namespace for Helm management
    echo "🏷️  Labeling namespace for Helm management..."
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    kubectl label namespace ${NAMESPACE} app.kubernetes.io/managed-by=Helm --overwrite
    kubectl annotate namespace ${NAMESPACE} meta.helm.sh/release-name=${HELM_RELEASE_NAME} --overwrite
    kubectl annotate namespace ${NAMESPACE} meta.helm.sh/release-namespace=${NAMESPACE} --overwrite
    
    # Determine MCP server image tag based on branch
    if [ "${GIT_BRANCH}" = "main" ]; then
        MCP_IMAGE_TAG="latest"
    elif [ "${GIT_BRANCH}" = "qa" ]; then
        MCP_IMAGE_TAG="staging"
    else
        MCP_IMAGE_TAG="dev"
    fi
    
    # Install or upgrade Helm chart with staging postfix
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    HELM_BIN="/tmp/helm/linux-amd64/helm"
    if [ -f "$HELM_BIN" ]; then
        $HELM_BIN upgrade --install ${HELM_RELEASE_NAME} helm/mcall-operator \
            --namespace ${NAMESPACE} \
            --values "helm/mcall-operator/${VALUES_FILE}" \
            --set image.tag="${BUILD_NUMBER}" \
            --set image.repository="doohee323/tz-mcall-operator" \
            --set mcpServer.image.tag="${BUILD_NUMBER}" \
            --set mcpServer.image.repository="doohee323/mcall-operator-mcp-server" \
            --set namespace.name="${NAMESPACE}" \
            --set logging.postgresql.password="${POSTGRES_PASSWORD:-}" \
            --set logging.mysql.password="${MYSQL_PASSWORD:-}" \
            --set logging.elasticsearch.password="${ELASTICSEARCH_PASSWORD:-}" \
            --wait \
            --timeout=10m
    else
        helm upgrade --install ${HELM_RELEASE_NAME} helm/mcall-operator \
            --namespace ${NAMESPACE} \
            --values "helm/mcall-operator/${VALUES_FILE}" \
            --set image.tag="${BUILD_NUMBER}" \
            --set image.repository="doohee323/tz-mcall-operator" \
            --set mcpServer.image.tag="${BUILD_NUMBER}" \
            --set mcpServer.image.repository="doohee323/mcall-operator-mcp-server" \
            --set namespace.name="${NAMESPACE}" \
            --set logging.postgresql.password="${POSTGRES_PASSWORD:-}" \
            --set logging.mysql.password="${MYSQL_PASSWORD:-}" \
            --set logging.elasticsearch.password="${ELASTICSEARCH_PASSWORD:-}" \
            --wait \
            --timeout=10m
    fi
    
    # Clean up temporary values file
    if [ -f "helm/mcall-operator/${VALUES_FILE}.tmp" ]; then
        rm "helm/mcall-operator/${VALUES_FILE}.tmp"
    fi
    if [ -f "helm/mcall-operator/${VALUES_FILE}.tmp.bak" ]; then
        rm "helm/mcall-operator/${VALUES_FILE}.tmp.bak"
    fi
    
    echo "✅ Helm chart deployed successfully"
}

# Verify deployment
verify_deployment() {
    echo "🔍 Verifying deployment..."
    
    # Install kubectl if not present
    install_kubectl
    
    # Check CRDs
    echo "=== CRD Status ==="
    if kubectl get crd | grep mcall; then
        echo "✅ CRDs found"
    else
        echo "⚠️  No CRDs found or insufficient permissions"
        echo "Note: Jenkins may need additional RBAC permissions for CRD access"
    fi
    
    # Check pods
    echo "=== Pod Status ==="
    kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/name=tz-mcall-operator || echo "No pods found"
    
    # Check services
    echo "=== Service Status ==="
    kubectl get svc -n ${NAMESPACE} || echo "No services found"
    
    # Check deployments
    echo "=== Deployment Status ==="
    kubectl get deployment -n ${NAMESPACE} || echo "No deployments found"
    
    # Wait for pods to be ready
    echo "⏳ Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tz-mcall-operator -n ${NAMESPACE} --timeout=300s || echo "Pods not ready yet"
    
    echo "✅ Deployment verification completed"
}

# Test CRD functionality
test_crd_functionality() {
    echo "🧪 Testing CRD functionality..."
    
    # Install kubectl if not present
    install_kubectl
    
    # Test Health Monitor Workflow (with Jenkins integration)
    if [ -f "examples/health-monitor-workflow-with-result-passing.yaml" ]; then
        echo "📋 Deploying Health Monitor Workflow with Jenkins integration..."
        kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml || echo "Failed to create Health Monitor Workflow"
        
        echo "⏳ Waiting for workflow to initialize (15 seconds)..."
        sleep 15
        
        echo ""
        echo "📊 Workflow Status:"
        echo "===================="
        kubectl get mcallworkflows -n ${NAMESPACE} || echo "No McallWorkflows found"
        
        echo ""
        echo "📋 Task Status:"
        echo "===================="
        kubectl get mcalltasks -n ${NAMESPACE} || echo "No McallTasks found"
        
        echo ""
        echo "🔍 Recent Workflow Execution Logs:"
        echo "===================="
        # Get the latest workflow pod logs if available
        WORKFLOW_POD=$(kubectl get pods -n ${NAMESPACE} -l workflow=health-monitor --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1:].metadata.name}' 2>/dev/null || echo "")
        if [ -n "$WORKFLOW_POD" ]; then
            echo "Latest workflow pod: $WORKFLOW_POD"
            kubectl logs -n ${NAMESPACE} $WORKFLOW_POD --tail=50 || echo "No logs available yet"
        else
            echo "No workflow pods found yet (first execution may be pending)"
        fi
        
        echo ""
        echo "✅ Health Monitor Workflow deployed successfully"
        echo "   - Workflow will execute every 1 minute"
        echo "   - On success: Logs to /app/log/mcall/health_monitor.log"
        echo "   - On failure: Logs + Triggers Jenkins docker-test build"
        echo "   - Monitor at: https://mcp-dev.drillquiz.com/"
        echo ""
        echo "⚠️  Note: Workflow will continue running. Clean up manually if needed:"
        echo "   kubectl delete -f examples/health-monitor-workflow-with-result-passing.yaml"
    else
        echo "⚠️  Health monitor workflow file not found"
        
        # Fallback to simple task test
        if [ -f "examples/mcalltask-example.yaml" ]; then
            echo "📋 Fallback: Testing with simple McallTask..."
            kubectl apply -f examples/mcalltask-example.yaml || echo "Failed to create McallTask"
            
            sleep 10
            
            echo "Checking McallTask status..."
            kubectl get mcalltasks -n ${NAMESPACE} || echo "No McallTasks found"
            
            echo "Cleaning up test McallTask..."
            kubectl delete -f examples/mcalltask-example.yaml || echo "Failed to delete McallTask"
        fi
    fi
    
    echo "✅ CRD functionality test completed"
}

# Rollback deployment
rollback_deployment() {
    echo "🔄 Rolling back deployment..."
    
    # Rollback Helm chart
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    HELM_BIN="/tmp/helm/linux-amd64/helm"
    if [ -f "$HELM_BIN" ]; then
        $HELM_BIN rollback ${HELM_RELEASE_NAME} -n ${NAMESPACE} || echo "No rollback available"
    else
        helm rollback ${HELM_RELEASE_NAME} -n ${NAMESPACE} || echo "No rollback available"
    fi
    
    echo "✅ Rollback completed"
}

# Clean up deployment
cleanup_deployment() {
    echo "🗑️  Cleaning up deployment..."
    
    # Delete Helm chart with force and no hooks
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    HELM_BIN="/tmp/helm/linux-amd64/helm"
    
    # Check if Helm release exists
    if [ -f "$HELM_BIN" ]; then
        if $HELM_BIN list -n ${NAMESPACE} | grep -q ${HELM_RELEASE_NAME}; then
            echo "Uninstalling Helm release: ${HELM_RELEASE_NAME}"
            $HELM_BIN uninstall ${HELM_RELEASE_NAME} -n ${NAMESPACE} --no-hooks --wait --timeout=5m || echo "Helm uninstall failed, trying force delete"
        else
            echo "No Helm release found: ${HELM_RELEASE_NAME}"
        fi
    else
        if helm list -n ${NAMESPACE} | grep -q ${HELM_RELEASE_NAME}; then
            echo "Uninstalling Helm release: ${HELM_RELEASE_NAME}"
            helm uninstall ${HELM_RELEASE_NAME} -n ${NAMESPACE} --no-hooks --wait --timeout=5m || echo "Helm uninstall failed, trying force delete"
        else
            echo "No Helm release found: ${HELM_RELEASE_NAME}"
        fi
    fi
    
    # Force delete any remaining resources
    echo "Force deleting any remaining resources..."
    kubectl delete all --all -n ${NAMESPACE} --force --grace-period=0 || echo "No resources to delete"
    kubectl delete jobs --all -n ${NAMESPACE} --force --grace-period=0 || echo "No jobs to delete"
    kubectl delete configmaps --all -n ${NAMESPACE} --force --grace-period=0 || echo "No configmaps to delete"
    kubectl delete secrets --all -n ${NAMESPACE} --force --grace-period=0 || echo "No secrets to delete"
    
    # Delete CRDs (optional - be careful in production)
    if [ "${NAMESPACE}" != "mcall-system" ]; then
        echo "Deleting CRDs (dev environment only)..."
        kubectl delete crd mcalltasks.mcall.tz.io --force --grace-period=0 || echo "CRD not found"
        kubectl delete crd mcallworkflows.mcall.tz.io --force --grace-period=0 || echo "CRD not found"
    fi
    
    # Delete namespace if it's a dev environment
    if [ "${NAMESPACE}" = "mcall-dev" ]; then
        echo "Deleting dev namespace: ${NAMESPACE}"
        kubectl delete namespace ${NAMESPACE} --force --grace-period=0 || echo "Namespace deletion failed"
    fi
    
    echo "✅ Cleanup completed"
}

# Main deployment function
deploy_to_kubernetes() {
    echo "🚀 Starting deployment to Kubernetes..."
    
    # Install required tools
    install_helm
    install_kubectl
    
    HELM_RELEASE_NAME="tz-mcall-operator${STAGING_POSTFIX}"
    
    # Check if this is a fresh install or upgrade
    HELM_BIN="/tmp/helm/linux-amd64/helm"
    if [ -f "$HELM_BIN" ]; then
        HELM_CMD="$HELM_BIN"
    else
        HELM_CMD="helm"
    fi
    
    # Check if Helm release exists
    if $HELM_CMD list -n ${NAMESPACE} 2>/dev/null | grep -q ${HELM_RELEASE_NAME}; then
        echo "📦 Existing Helm release found: ${HELM_RELEASE_NAME}"
        DEPLOYMENT_TYPE="upgrade"
    else
        echo "📦 No existing Helm release found, will perform fresh install"
        DEPLOYMENT_TYPE="install"
        
        # For fresh install, clean up any orphaned resources
        cleanup_conflicting_resources
    fi
    
    # Always update CRDs first (Helm doesn't update CRDs on upgrade)
    echo "🔧 Updating CRDs..."
    deploy_crds
    
    # Deploy Helm chart (install or upgrade)
    echo "🚀 Deploying Helm chart (${DEPLOYMENT_TYPE})..."
    deploy_helm_chart
    
    # Verify deployment
    verify_deployment
    
    # Test CRD functionality (optional, only for dev)
    if [ "${NAMESPACE}" != "mcall-system" ]; then
        test_crd_functionality
    fi
    
    echo "🎉 Deployment completed successfully!"
    echo "   Deployment type: ${DEPLOYMENT_TYPE}"
    echo "   Namespace: ${NAMESPACE}"
    echo "   Release: ${HELM_RELEASE_NAME}"
}

# Main execution logic
case "${ACTION}" in
    "deploy")
        deploy_to_kubernetes
        ;;
    "rollback")
        rollback_deployment
        ;;
    "cleanup")
        cleanup_deployment
        ;;
    "verify")
        verify_deployment
        ;;
    "test")
        test_crd_functionality
        ;;
    "force-cleanup")
        force_cleanup_all_mcall_resources
        ;;
    *)
        echo "❌ Invalid ACTION: ${ACTION}"
        echo "Usage: $0 <BUILD_NUMBER> <GIT_BRANCH> <NAMESPACE> <VALUES_FILE> [deploy|rollback|cleanup|verify|test|force-cleanup]"
        echo ""
        echo "Examples:"
        echo "  $0 latest main mcall-system values.yaml deploy"
        echo "  $0 latest dev mcall-dev values-dev.yaml deploy"
        echo "  $0 latest main mcall-system values.yaml rollback"
        echo "  $0 latest dev mcall-dev values-dev.yaml cleanup"
        echo "  $0 latest dev mcall-dev values-dev.yaml force-cleanup"
        exit 1
        ;;
esac

