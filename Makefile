# Makefile for mcall CRD project

.PHONY: test-debug test-verbose test-specific test-all test-cleanup test-jenkins build deploy clean help

# =============================================================================
# LOCAL DEVELOPMENT & TESTING
# =============================================================================

# Run tests with debugging information
test-debug:
	@echo "=== Running tests in debug mode ==="
	go test -v -race ./controller

# Run specific test function
test-specific:
	@echo "=== Running specific test function ==="
	@read -p "Enter test function name (e.g., TestExecuteCommand): " testname; \
	go test -v -run $$testname ./controller

# Run all tests with verbose logging
test-verbose:
	@echo "=== Running all tests with verbose logging ==="
	go test -v -race -count=1 ./controller

# Run tests in parallel
test-parallel:
	@echo "=== Running parallel tests ==="
	go test -v -race -parallel 4 ./controller

# Run tests with coverage
test-coverage:
	@echo "=== Running tests with coverage ==="
	go test -v -race -coverprofile=coverage.out ./controller
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated in coverage.html"

# Run benchmark tests
test-benchmark:
	@echo "=== Running benchmark tests ==="
	go test -v -bench=. -benchmem ./controller

# =============================================================================
# INTEGRATION & CLEANUP TESTS
# =============================================================================

# Run cleanup test (requires kubectl and cluster access)
test-cleanup:
	@echo "=== Running cleanup integration test ==="
	@if ! command -v kubectl &> /dev/null; then \
		echo "Error: kubectl is required for cleanup test"; \
		exit 1; \
	fi
	@if ! command -v helm &> /dev/null; then \
		echo "Error: helm is required for cleanup test"; \
		exit 1; \
	fi
	chmod +x tests/scripts/test-cleanup.sh
	./tests/scripts/test-cleanup.sh

# Run Jenkins-style tests (validation only, no cluster required)
test-jenkins:
	@echo "=== Running Jenkins-style validation tests ==="
	chmod +x tests/scripts/jenkins-test.sh
	./tests/scripts/jenkins-test.sh

# Run Jenkins tests with custom parameters
test-jenkins-custom:
	@echo "=== Running Jenkins tests with custom parameters ==="
	@read -p "Enter BUILD_NUMBER (default: latest): " build_num; \
	read -p "Enter GIT_BRANCH (default: main): " git_branch; \
	read -p "Enter NAMESPACE (default: mcall-dev): " namespace; \
	read -p "Enter VALUES_FILE (default: values-dev.yaml): " values_file; \
	chmod +x tests/scripts/jenkins-test.sh; \
	./tests/scripts/jenkins-test.sh $${build_num:-latest} $${git_branch:-main} $${namespace:-mcall-dev} $${values_file:-values-dev.yaml}

# =============================================================================
# BUILD & DEPLOYMENT
# =============================================================================

# Build the controller binary
build:
	@echo "=== Building controller binary ==="
	go build -o bin/controller ./cmd/controller

# Build Docker image
build-docker:
	@echo "=== Building Docker image ==="
	@if ! command -v docker &> /dev/null; then \
		echo "Error: Docker is required for image build"; \
		exit 1; \
	fi
	docker build -f docker/Dockerfile -t mcall-controller:latest .

# Deploy to cluster (requires kubectl and helm)
deploy:
	@echo "=== Deploying to cluster ==="
	@if ! command -v kubectl &> /dev/null; then \
		echo "Error: kubectl is required for deployment"; \
		exit 1; \
	fi
	@if ! command -v helm &> /dev/null; then \
		echo "Error: helm is required for deployment"; \
		exit 1; \
	fi
	chmod +x ci/k8s.sh
	./ci/k8s.sh latest main mcall-dev values-dev.yaml deploy

# Deploy to dev environment
deploy-dev:
	@echo "=== Deploying to dev environment ==="
	chmod +x ci/k8s.sh
	./ci/k8s.sh latest dev mcall-dev values-dev.yaml deploy

# Deploy to staging environment
deploy-staging:
	@echo "=== Deploying to staging environment ==="
	chmod +x ci/k8s.sh
	./ci/k8s.sh latest staging mcall-staging values-staging.yaml deploy

# =============================================================================
# CLEANUP
# =============================================================================

# Clean test results and build artifacts
clean:
	@echo "=== Cleaning test results and build artifacts ==="
	rm -f coverage.out coverage.html
	rm -rf test-results/
	rm -rf bin/
	go clean -cache

# Clean Docker images
clean-docker:
	@echo "=== Cleaning Docker images ==="
	@if command -v docker &> /dev/null; then \
		docker rmi mcall-controller:latest 2>/dev/null || true; \
		docker system prune -f; \
	fi

# Clean everything
clean-all: clean clean-docker
	@echo "=== Cleaning everything ==="

# =============================================================================
# CODE GENERATION
# =============================================================================

# Generate code using controller-gen
generate:
	@echo "=== Generating code using controller-gen ==="
	@if ! command -v controller-gen &> /dev/null; then \
		echo "Error: controller-gen is required for code generation"; \
		echo "Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"; \
		exit 1; \
	fi
	controller-gen object paths=./api/...
	controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds
	controller-gen rbac:roleName=manager-role paths=./controller/... output:rbac:dir=./helm/mcall-operator/templates

# Generate only DeepCopy methods
generate-objects:
	@echo "=== Generating DeepCopy methods ==="
	@if ! command -v controller-gen &> /dev/null; then \
		echo "Error: controller-gen is required for code generation"; \
		echo "Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"; \
		exit 1; \
	fi
	controller-gen object paths=./api/...

# Generate only CRDs
generate-crds:
	@echo "=== Generating CRDs ==="
	@if ! command -v controller-gen &> /dev/null; then \
		echo "Error: controller-gen is required for code generation"; \
		echo "Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"; \
		exit 1; \
	fi
	controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds

# Generate only RBAC
generate-rbac:
	@echo "=== Generating RBAC permissions ==="
	@if ! command -v controller-gen &> /dev/null; then \
		echo "Error: controller-gen is required for code generation"; \
		echo "Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"; \
		exit 1; \
	fi
	controller-gen rbac:roleName=manager-role paths=./controller/... output:rbac:dir=./helm/mcall-operator/templates

# =============================================================================
# UTILITY COMMANDS
# =============================================================================

# Run all local tests
test-all: test-verbose test-coverage test-benchmark

# Run all validation tests (no cluster required)
validate: test-jenkins

# Run all integration tests (requires cluster)
integration: test-cleanup

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "CODE GENERATION:"
	@echo "  generate           - Generate all code (DeepCopy, CRDs, RBAC)"
	@echo "  generate-objects   - Generate DeepCopy methods only"
	@echo "  generate-crds      - Generate CRDs only"
	@echo "  generate-rbac      - Generate RBAC permissions only"
	@echo ""
	@echo "LOCAL DEVELOPMENT & TESTING:"
	@echo "  test-debug         - Run tests in debug mode"
	@echo "  test-specific      - Run specific test function"
	@echo "  test-verbose       - Run all tests with verbose logging"
	@echo "  test-parallel      - Run tests in parallel"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-benchmark     - Run benchmark tests"
	@echo "  test-all           - Run all local tests"
	@echo ""
	@echo "INTEGRATION & CLEANUP TESTS:"
	@echo "  test-cleanup       - Run cleanup integration test (requires cluster)"
	@echo "  test-jenkins       - Run Jenkins-style validation tests"
	@echo "  test-jenkins-custom - Run Jenkins tests with custom parameters"
	@echo "  validate           - Run all validation tests (no cluster required)"
	@echo "  integration        - Run all integration tests (requires cluster)"
	@echo ""
	@echo "BUILD & DEPLOYMENT:"
	@echo "  build              - Build controller binary"
	@echo "  build-docker       - Build Docker image"
	@echo "  deploy             - Deploy to cluster"
	@echo "  deploy-dev         - Deploy to dev environment"
	@echo "  deploy-staging     - Deploy to staging environment"
	@echo ""
	@echo "CLEANUP:"
	@echo "  clean              - Clean test results and build artifacts"
	@echo "  clean-docker       - Clean Docker images"
	@echo "  clean-all          - Clean everything"
	@echo ""
	@echo "  help               - Show this help message"
