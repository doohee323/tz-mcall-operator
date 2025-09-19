#!/bin/bash

# Test script for mcall CRD cleanup functionality

set -e

echo "Testing mcall CRD cleanup functionality..."

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
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    print_error "helm is not installed or not in PATH"
    exit 1
fi

# Check if we're connected to a cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Not connected to a Kubernetes cluster"
    exit 1
fi

print_status "Connected to cluster: $(kubectl config current-context)"

# Test 1: Install the chart
print_status "Test 1: Installing mcall-operator chart..."
helm install test-mcall-operator ./helm/mcall-operator --create-namespace --namespace test-mcall-system

# Wait for deployment to be ready
print_status "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/test-mcall-operator -n test-mcall-system

# Test 2: Create some test resources
print_status "Test 2: Creating test mcalltask resources..."

cat <<EOF | kubectl apply -f -
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: test-task-1
  namespace: test-mcall-system
spec:
  type: cmd
  input: "echo 'Hello World'"
  name: "test-task-1"
  timeout: 10
---
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: test-task-2
  namespace: test-mcall-system
spec:
  type: cmd
  input: "echo 'Test Task 2'"
  name: "test-task-2"
  timeout: 10
EOF

# Wait a moment for resources to be created
sleep 5

# Verify resources exist
print_status "Verifying test resources were created..."
kubectl get mcalltasks -n test-mcall-system

# Test 3: Uninstall the chart (this should trigger cleanup)
print_status "Test 3: Uninstalling chart (this should trigger cleanup)..."
helm uninstall test-mcall-operator --namespace test-mcall-system

# Wait for cleanup to complete
print_status "Waiting for cleanup to complete..."
sleep 10

# Test 4: Verify cleanup
print_status "Test 4: Verifying cleanup..."

# Check if namespace still exists
if kubectl get namespace test-mcall-system &> /dev/null; then
    print_warning "Namespace still exists, checking for remaining resources..."
    
    # Check for remaining CRD resources
    if kubectl get mcalltasks -n test-mcall-system &> /dev/null; then
        print_error "McallTask resources still exist after cleanup!"
        kubectl get mcalltasks -n test-mcall-system
    else
        print_status "McallTask resources successfully cleaned up"
    fi
    
    # Check namespace status
    namespace_status=$(kubectl get namespace test-mcall-system -o jsonpath='{.status.phase}')
    if [ "$namespace_status" = "Terminating" ]; then
        print_warning "Namespace is in Terminating state, this may be due to other resources"
    else
        print_status "Namespace status: $namespace_status"
    fi
else
    print_status "Namespace successfully deleted"
fi

# Cleanup: Force delete namespace if it still exists
if kubectl get namespace test-mcall-system &> /dev/null; then
    print_status "Force deleting remaining namespace..."
    kubectl delete namespace test-mcall-system --force --grace-period=0
fi

print_status "Cleanup test completed!"
