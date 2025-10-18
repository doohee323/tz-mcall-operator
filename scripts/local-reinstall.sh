#!/bin/bash

# Local development reinstall script
# This script can delete resources only or delete and reinstall
# Usage: 
#   ./local-reinstall.sh --delete-only    # Delete resources only
#   ./local-reinstall.sh --install-only   # Install only (default)
#   ./local-reinstall.sh                  # Delete and install (default)

set -e

# Parse command line arguments
DELETE_ONLY=false
INSTALL_ONLY=false

for arg in "$@"; do
    case $arg in
        --delete-only)
            DELETE_ONLY=true
            shift
            ;;
        --install-only)
            INSTALL_ONLY=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--delete-only|--install-only]"
            echo "  --delete-only    Delete resources only"
            echo "  --install-only   Install only"
            echo "  (no args)        Delete and install (default)"
            exit 0
            ;;
        *)
            echo "Unknown option: $arg"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

if [ "$DELETE_ONLY" = true ] && [ "$INSTALL_ONLY" = true ]; then
    echo "Error: Cannot specify both --delete-only and --install-only"
    exit 1
fi

if [ "$DELETE_ONLY" = true ]; then
    echo "🗑️  Starting resource deletion only..."
elif [ "$INSTALL_ONLY" = true ]; then
    echo "🚀 Starting installation only..."
else
    echo "🚀 Starting local development reinstall (delete and install)..."
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to delete resources
delete_resources() {
    # Step 1: Delete existing Helm release
    print_status "Step 1: Deleting existing Helm release..."
    if helm list -n mcall-dev | grep -q test-local-dev; then
        helm uninstall test-local-dev -n mcall-dev || true
        print_status "Helm release deleted"
    else
        print_warning "No existing Helm release found"
    fi

    # Step 2: Remove finalizers first, then delete CRD resources and namespace
    print_status "Step 2: Removing finalizers from CRD resources..."
    # Remove finalizers from CRD resources first
    for task in $(kubectl get mcalltask -n mcall-dev -o name 2>/dev/null || true); do
        print_status "Removing finalizers from $task..."
        kubectl patch $task -n mcall-dev -p '{"metadata":{"finalizers":[]}}' --type=merge || true
    done
    for workflow in $(kubectl get mcallworkflow -n mcall-dev -o name 2>/dev/null || true); do
        print_status "Removing finalizers from $workflow..."
        kubectl patch $workflow -n mcall-dev -p '{"metadata":{"finalizers":[]}}' --type=merge || true
    done

    print_status "Step 3: Deleting CRD resources and jobs..."
    # Now delete CRD resources that might block namespace deletion
    kubectl delete mcallworkflow --all -n mcall-dev --force --grace-period=0 || true
    kubectl delete mcalltask --all -n mcall-dev --force --grace-period=0 || true

    # Delete any cleanup jobs that might block namespace deletion
    kubectl delete job -l app.kubernetes.io/name=mcall-operator -n mcall-dev --force --grace-period=0 || true
    kubectl delete job test-local-dev-local-dev-cleanup -n mcall-dev --force --grace-period=0 || true
    # Force delete namespace
    kubectl delete namespace mcall-dev --force --grace-period=0 || true
    print_status "CRD resources, jobs and namespace deleted"

    # Step 4: Wait for namespace to be fully deleted
    print_status "Step 4: Waiting for namespace to be fully deleted..."
    sleep 10
}

# Function to install resources
install_resources() {
    # Step 1: Create namespace with proper labels
    print_status "Step 1: Creating namespace with proper labels..."
    kubectl create namespace mcall-dev || true
    kubectl label namespace mcall-dev app.kubernetes.io/managed-by=Helm || true
    kubectl annotate namespace mcall-dev meta.helm.sh/release-name=test-local-dev || true
    kubectl annotate namespace mcall-dev meta.helm.sh/release-namespace=mcall-dev || true

    # Step 2: Create necessary secrets BEFORE Helm installation
    print_status "Step 2: Creating necessary secrets..."
    
    # Create mcp-api-keys secret (delete first if exists)
    kubectl delete secret mcp-api-keys -n mcall-dev || true
    kubectl create secret generic mcp-api-keys \
        --from-literal=api-keys="local-dev-key-12345" \
        --namespace mcall-dev

    # Create jenkins-mcp-credentials secret (delete first if exists)
    kubectl delete secret jenkins-mcp-credentials -n mcall-dev || true
    kubectl create secret generic jenkins-mcp-credentials \
        --from-literal=username="admin" \
        --from-literal=token="11e8b8a0b8b8b8b8b8b8b8b8b8b8b8b8b8" \
        --namespace mcall-dev

    print_status "Secrets created successfully"

    # Step 3: Install Helm chart
    print_status "Step 3: Installing Helm chart..."
    helm install test-local-dev ./helm/mcall-operator \
        --namespace mcall-dev \
        --values values-local-dev.yaml \
        --set image.repository=doohee323/tz-mcall-operator \
        --set image.tag=latest \
        --set image.pullPolicy=Always

    if [ $? -eq 0 ]; then
        print_status "Helm chart installed successfully"
    else
        print_error "Helm chart installation failed"
        exit 1
    fi

    # Step 4: Wait for pods to be ready
    print_status "Step 4: Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mcall-operator -n mcall-dev --timeout=300s || true

    # Step 5: Apply workflow and tasks
    print_status "Step 5: Applying workflow and tasks..."
    kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

    print_status "Workflow and tasks applied successfully"

    # Step 6: Show status
    print_status "Step 6: Showing final status..."
    echo ""
    echo "=== Pod Status ==="
    kubectl get pods -n mcall-dev

    echo ""
    echo "=== Secrets ==="
    kubectl get secrets -n mcall-dev

    echo ""
    echo "=== McallWorkflow ==="
    kubectl get mcallworkflow -n mcall-dev

    echo ""
    echo "=== McallTask ==="
    kubectl get mcalltask -n mcall-dev

    echo ""
    print_status "✅ Installation completed successfully!"
    print_status "You can now check the logs with: kubectl logs -n mcall-dev deployment/test-local-dev"
    print_status "Or check the UI at: http://localhost:3000 (if mcp-server is running locally)"
}

# Main execution logic
if [ "$DELETE_ONLY" = true ]; then
    delete_resources
    print_status "✅ Resource deletion completed successfully!"
elif [ "$INSTALL_ONLY" = true ]; then
    install_resources
else
    # Default: delete and install
    delete_resources
    install_resources
fi
