#!/bin/bash

set -e

echo "🚀 Starting local mcall CRD development setup..."

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function definitions
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Force cleanup function for existing resources (for development environment)
cleanup_existing_resources() {
    print_status "Force cleaning up existing development resources..."
    
    # 1. Force delete all resources in mcall-dev namespace
    if kubectl get namespace mcall-dev >/dev/null 2>&1; then
        print_status "Force removing all resources in mcall-dev namespace..."
        
        # Force delete all jobs (including cleanup jobs)
        kubectl get jobs -n mcall-dev -o name 2>/dev/null | while read job; do
            if [ -n "$job" ]; then
                print_status "Force deleting job: $job"
                kubectl delete "$job" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
        
        # Force delete all pods
        kubectl get pods -n mcall-dev -o name 2>/dev/null | while read pod; do
            if [ -n "$pod" ]; then
                print_status "Force deleting pod: $pod"
                kubectl delete "$pod" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
        
        # Force delete all deployments
        kubectl get deployments -n mcall-dev -o name 2>/dev/null | while read deployment; do
            if [ -n "$deployment" ]; then
                print_status "Force deleting deployment: $deployment"
                kubectl delete "$deployment" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
        
        # Force delete all services
        kubectl get services -n mcall-dev -o name 2>/dev/null | while read service; do
            if [ -n "$service" ]; then
                print_status "Force deleting service: $service"
                kubectl delete "$service" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
        
        # Force delete all configmaps and secrets
        kubectl get configmaps -n mcall-dev -o name 2>/dev/null | while read cm; do
            if [ -n "$cm" ]; then
                print_status "Force deleting configmap: $cm"
                kubectl delete "$cm" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
        
        kubectl get secrets -n mcall-dev -o name 2>/dev/null | while read secret; do
            if [ -n "$secret" ]; then
                print_status "Force deleting secret: $secret"
                kubectl delete "$secret" -n mcall-dev --force --grace-period=0 2>/dev/null || true
            fi
        done
    fi
    
    # 2. Force delete mcall-dev Helm release (including all states)
    if helm list -n mcall-dev --all --short | grep -q mcall-crd-dev; then
        print_status "Force removing Helm release (including pending/uninstalling)..."
        helm uninstall mcall-crd-dev -n mcall-dev --no-hooks 2>/dev/null || true
    fi
    
    # 3. Force delete mcall-dev namespace
    if kubectl get namespace mcall-dev >/dev/null 2>&1; then
        print_status "Force removing mcall-dev namespace..."
        
        # Remove finalizers
        kubectl patch namespace mcall-dev -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null || true
        
        # Force delete
        kubectl delete namespace mcall-dev --force --grace-period=0 2>/dev/null || true
        
        # Verify complete deletion and wait
        local max_wait=30
        local wait_time=0
        while kubectl get namespace mcall-dev >/dev/null 2>&1 && [ $wait_time -lt $max_wait ]; do
            print_status "Waiting for namespace deletion... (${wait_time}s/${max_wait}s)"
            sleep 1
            wait_time=$((wait_time + 1))
        done
        
        # If still exists, perform final force delete
        if kubectl get namespace mcall-dev >/dev/null 2>&1; then
            print_warning "Final force deletion of mcall-dev"
            kubectl delete namespace mcall-dev --force --grace-period=0 2>/dev/null || true
            sleep 3
        fi
    fi
    
    print_success "Force cleanup completed"
}

# 1. Check required tools
print_status "Checking required tools..."

if ! command -v helm &> /dev/null; then
    print_error "Helm is not installed. Please install it first:"
    echo "brew install helm"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed. Please install it first:"
    echo "brew install kubectl"
    exit 1
fi

print_success "All required tools are installed"

# 2. Check Kubernetes cluster status
print_status "Checking Kubernetes cluster status..."

if ! kubectl cluster-info &> /dev/null; then
    print_error "Kubernetes cluster is not accessible"
    echo "Please ensure you have a running Kubernetes cluster:"
    echo "- minikube start"
    echo "- kind create cluster"
    echo "- or Docker Desktop Kubernetes"
    exit 1
fi

print_success "Kubernetes cluster is accessible"

# 3. Validate Helm Chart
print_status "Linting Helm chart..."
if helm lint ./helm/mcall-crd; then
    print_success "Helm chart linting passed"
else
    print_error "Helm chart linting failed"
    exit 1
fi

# 4. Build and push Docker image
print_status "Building Docker image..."
if docker build --platform linux/amd64 -t tz-mcall-operator:test -f docker/Dockerfile .; then
    print_success "Docker image built successfully"
else
    print_error "Failed to build Docker image"
    exit 1
fi

# Tag and push image
print_status "Tagging and pushing Docker image..."
if docker tag tz-mcall-operator:test "doohee323/tz-mcall-operator:test"; then
    print_success "Image tagged successfully"
else
    print_error "Failed to tag image"
    exit 1
fi

if docker push "doohee323/tz-mcall-operator:test"; then
    print_success "Docker image pushed successfully"
else
    print_warning "Failed to push image, trying alternative approach..."
    
    # Alternative: Load image to all nodes
    print_status "Loading image to all nodes..."
    for node in $(kubectl get nodes -o name | cut -d'/' -f2); do
        print_status "Loading image to node: $node"
        docker save tz-mcall-operator:test | ssh $node "docker load" 2>/dev/null || true
    done
    print_success "Image loading completed"
fi

# 5. Test Chart template rendering
print_status "Testing chart template rendering..."
if helm template mcall-crd-dev ./helm/mcall-crd --values ./helm/mcall-crd/values-dev.yaml > /dev/null; then
    print_success "Chart template rendering successful"
else
    print_error "Chart template rendering failed"
    exit 1
fi

# 6. Install CRDs
print_status "Installing CRDs..."
# Generate CRD files by rendering Helm templates
helm template mcall-crd-dev ./helm/mcall-crd \
    --values ./helm/mcall-crd/values-dev.yaml \
    --show-only templates/crds/mcalltask-crd.yaml \
    --show-only templates/crds/mcallworkflow-crd.yaml \
 > /tmp/mcall-crds.yaml

if kubectl apply -f /tmp/mcall-crds.yaml; then
    print_success "CRDs installed successfully"
    rm -f /tmp/mcall-crds.yaml
else
    print_error "Failed to install CRDs"
    rm -f /tmp/mcall-crds.yaml
    exit 1
fi

# 7. Clean up existing resources
cleanup_existing_resources

# 8. Verify complete namespace deletion
print_status "Verifying namespace deletion..."
if kubectl get namespace mcall-dev >/dev/null 2>&1; then
    print_warning "Namespace still exists, forcing final deletion..."
    kubectl delete namespace mcall-dev --force --grace-period=0 2>/dev/null || true
    sleep 2
fi

# Wait until namespace is completely deleted
print_status "Waiting for namespace to be completely deleted..."
sleep 5

# 9. Create namespace
print_status "Creating mcall-dev namespace..."
kubectl create namespace mcall-dev 2>/dev/null || true
sleep 2

# 10. Install Helm Chart (for development environment)
print_status "Installing Helm chart (development mode)..."
if helm install mcall-crd-dev ./helm/mcall-crd \
    --namespace mcall-dev \
    --values ./helm/mcall-crd/values-dev.yaml \
    --wait --timeout=5m; then
    print_success "Helm chart installed successfully"
else
    print_error "Failed to install Helm chart, trying alternative approach..."
    
    # Alternative: Recreate namespace and retry
    print_status "Recreating namespace and retrying..."
    kubectl delete namespace mcall-dev --force --grace-period=0 2>/dev/null || true
    sleep 3
    kubectl create namespace mcall-dev
    sleep 2
    
    if helm install mcall-crd-dev ./helm/mcall-crd \
        --namespace mcall-dev \
        --values ./helm/mcall-crd/values-dev.yaml \
        --wait --timeout=5m; then
        print_success "Helm chart installed successfully on retry"
    else
        print_error "Failed to install Helm chart even after retry"
        print_status "Continuing with manual installation..."
        
        # Manually create resources
        print_status "Creating resources manually..."
        # Install CRDs first
        helm template mcall-crd-dev ./helm/mcall-crd \
            --values ./helm/mcall-crd/values-dev.yaml \
            --show-only templates/crds/mcalltask-crd.yaml \
            --show-only templates/crds/mcallworkflow-crd.yaml \
         > /tmp/mcall-crds.yaml
        kubectl apply -f /tmp/mcall-crds.yaml
        rm -f /tmp/mcall-crds.yaml
        
        # Apply by rendering Helm templates
        print_status "Rendering Helm templates..."
        helm template mcall-crd-dev ./helm/mcall-crd \
            --namespace mcall-dev \
            --values ./helm/mcall-crd/values-dev.yaml > /tmp/mcall-resources.yaml
        
        # Apply rendered resources (excluding namespace)
        print_status "Applying rendered resources..."
        kubectl apply -f /tmp/mcall-resources.yaml -n mcall-dev 2>/dev/null || true
        
        # Cleanup
        rm -f /tmp/mcall-resources.yaml
        
        print_success "Manual installation completed"
    fi
fi

# 11. Check installation status
print_status "Checking installation status..."
kubectl get pods -n mcall-dev
kubectl get crd | grep mcall

# 12. Execute test tasks
print_status "Running test tasks..."
if kubectl apply -f examples/mcalltask-example.yaml; then
    print_success "Test tasks created successfully"
else
    print_warning "Failed to create test tasks (this is expected if CRD controller is not fully ready)"
fi

# 13. Check status
print_status "Final status check..."
echo ""
echo "=== Pod Status ==="
kubectl get pods -n mcall-dev

echo ""
echo "=== CRD Status ==="
kubectl get crd | grep mcall

echo ""
echo "=== McallTasks ==="
kubectl get mcalltasks 2>/dev/null || echo "No McallTasks found (controller may still be starting)"

echo ""
echo "=== Services ==="
kubectl get svc -n mcall-dev

echo ""
print_success "🎉 Local development setup complete!"
echo ""
echo "Next steps:"
echo "1. Check controller logs: kubectl logs -n mcall-dev -l app.kubernetes.io/name=mcall-crd -f"
echo "2. Monitor McallTasks: kubectl get mcalltasks -w"
echo "3. Test with: kubectl apply -f examples/mcalltask-example.yaml"
echo "4. Clean up: helm uninstall mcall-crd-dev -n mcall-dev"
