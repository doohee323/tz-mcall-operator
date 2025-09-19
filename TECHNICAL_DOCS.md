# mcall Operator Technical Documentation

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [CRD Definitions](#crd-definitions)
- [Controller Implementation](#controller-implementation)
- [Development Guide](#development-guide)
- [Development & Testing](#development--testing)
- [Deployment Guide](#deployment-guide)
- [Jenkins CI/CD](#jenkins-cicd)
- [Troubleshooting](#troubleshooting)
- [Performance Analysis](#performance-analysis)

## Architecture Overview

### System Components

```
              ┌────────────────────────────────────┐
              │       Kubernetes API Server        │
              │  ┌─────────────┐  ┌─────────────┐  │
              │  │ McallTask   │  │McallWorkflow│  │
              │  │   CRD       │  │    CRD      │  │
              │  └─────────────┘  └─────────────┘  │
              └────────────────────────────────────┘
                                │
                      ┌─────────────────┐
                      │   Controller    │ ← Leader Election
                      │   (Operator)    │   (Only one Leader among multiple)
                      └─────────────────┘
                                │
                      ┌─────────────────┐
                      │   Executor      │ ← Actual Worker
                      │   (Pods)        │   (Pods for CRD execution)
                      └─────────────────┘
```

### Detailed Architecture Flow

```
┌─────────────────────────────────────────────────────────────┐
│                        User/Admin                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   kubectl   │  │   Web UI    │  │   API       │          │
│  │   apply     │  │   (future)  │  │   calls     │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                    │
│               ┌─────────────┐  ┌─────────────┐              │
│               │ McallTask   │  │McallWorkflow│              │
│               │   CRD       │  │    CRD      │              │
│               └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Controller Layer                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │ Controller Pod 1│  │ Controller Pod 2│  │ Controller  │  │
│  │   (Leader)      │  │   (Follower)    │  │   Pod 3     │  │
│  │                 │  │                 │  │ (Follower)  │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
│           │ Leader Election (Kubernetes Leases)             │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Executor Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Executor    │  │ Executor    │  │ Executor    │          │
│  │ Pod 1       │  │ Pod 2       │  │ Pod N       │          │
│  │ (Task A)    │  │ (Task B)    │  │ (Task X)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Role | Responsibilities |
|-----------|------|------------------|
| **CRDs** | Definition | Task specifications, schedules, dependencies |
| **Controller** | Management | CRD lifecycle, pod creation, status updates |
| **Executor Pods** | Execution | Actual task execution, result reporting |
| **Leader Election** | Coordination | Prevents duplicate processing, handles scheduled tasks |

## CRD and Example Architecture

### CRD-Example Relationship Flow

```
          ┌────────────────────────────────────────────┐
          │          CRD Definitions (crds/)           │
          │  ┌─────────────────┐  ┌─────────────────┐  │
          │  │ McallTask       │  │ McallWorkflow   │  │
          │  │ CRD Schema      │  │ CRD Schema      │  │
          │  │                 │  │                 │  │
          │  │ • type: cmd     │  │ • tasks[]       │  │
          │  │ • input: string │  │ • schedule      │  │
          │  │ • schedule      │  │ • concurrency   │  │
          │  │ • timeout       │  │ • dependencies  │  │
          │  │ • retryCount    │  │ • environment   │  │
          │  └─────────────────┘  └─────────────────┘  │
          └────────────────────────────────────────────┘
                                │
                                │ Schema Validation
                                ▼
          ┌────────────────────────────────────────────┐
          │       Example Files (examples/)            │
          │  ┌─────────────────┐  ┌─────────────────┐  │
          │  │ mcalltask-      │  │ mcallworkflow-  │  │
          │  │ example.yaml    │  │ example.yaml    │  │
          │  │                 │  │                 │  │
          │  │ • system-info   │  │ • monitoring-   │  │
          │  │ • http-health   │  │   workflow      │  │
          │  │ • disk-monitor  │  │ • deployment-   │  │
          │  │ • memory-check  │  │   workflow      │  │
          │  │ • backup-task   │  │ • backup-       │  │
          │  │                 │  │   workflow      │  │
          │  └─────────────────┘  └─────────────────┘  │
          └────────────────────────────────────────────┘
                                │
                                │ kubectl apply
                                ▼
          ┌────────────────────────────────────────────┐
          │           Kubernetes API Server            │
          │  ┌─────────────────┐  ┌─────────────────┐  │
          │  │ McallTask       │  │ McallWorkflow   │  │
          │  │                 │  │                 │  │
          │  │ • system-info   │  │ • monitoring-   │  │
          │  │ • http-health   │  │   workflow      │  │
          │  │ • disk-monitor  │  │ • deployment-   │  │
          │  │ • memory-check  │  │   workflow      │  │
          │  │ • backup-task   │  │ • backup-       │  │
          │  │                 │  │   workflow      │  │
          │  └─────────────────┘  └─────────────────┘  │
          └────────────────────────────────────────────┘
                                │
                                │ Controller Reconciliation
                                ▼
          ┌────────────────────────────────────────────┐
          │              Controller Layer              │
          │  ┌─────────────────┐  ┌─────────────────┐  │
          │  │ McallTask       │  │ McallWorkflow   │  │
          │  │                 │  │                 │  │
          │  │ • Status Update │  │ • Task          │  │
          │  │ • Pod Creation  │  │   Orchestration │  │
          │  │ • Result        │  │ • Dependency    │  │
          │  │   Collection    │  │   Management    │  │
          │  │ • Retry Logic   │  │ • Concurrency   │  │
          │  │                 │  │   Control       │  │
          │  └─────────────────┘  └─────────────────┘  │
          └────────────────────────────────────────────┘
                                │
                                │ Pod Creation
                                ▼
          ┌────────────────────────────────────────────┐
          │                Executor Pods               │
          │  ┌─────────────────┐  ┌─────────────────┐  │
          │  │ Task Executor   │  │ Workflow        │  │
          │  │ Pods            │  │ Executor Pods   │  │
          │  └─────────────────┘  └─────────────────┘  │
          └────────────────────────────────────────────┘
```

### Data Flow

#### 1. CRD Definition → Example Creation
```
CRD Schema (crds/) → Example Files (examples/) → kubectl apply → Kubernetes Resources
```

#### 2. Example File Roles
- **Real usage examples of CRD Schema**
- **Templates that developers can reference**
- **Instances for testing and demonstration**

#### 3. Layer Relationships

| Layer | Role | CRD Relationship |
|-------|------|------------------|
| **CRD Schema** | Data structure definition | Base schema |
| **Example Files** | Real usage examples | Schema-based instances |
| **Kubernetes Resources** | Running resources | Instances created from examples |
| **Controller** | Lifecycle management | CRD instance monitoring |
| **Executor Pods** | Actual task execution | Work pods created by controller |

### File Structure Relationship

```
crds/
├── mcalltask-crd.yaml          # McallTask schema definition
├── mcallworkflow-crd.yaml      # McallWorkflow schema definition

examples/
├── mcalltask-example.yaml      # McallTask usage examples
├── mcallworkflow-example.yaml  # McallWorkflow usage examples
```

### Key Points

1. **CRD = Schema Definition** (What to define)
2. **Example = Usage Examples** (How to use)
3. **Controller = Execution Management** (Lifecycle management)
4. **Executor Pods = Actual Work** (Concrete execution)

**CRDs define "what", Examples show "how to use", and Controllers handle "actual execution"!**

## CRD Definitions

### McallTask CRD

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: mcalltasks.mcall.tz.io
spec:
  group: mcall.tz.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              type:
                type: string
                enum: ["cmd", "get", "post"]
              input:
                type: string
              expect:
                type: string
                description: "Expected output pattern for validation"
              name:
                type: string
              timeout:
                type: integer
                default: 30
              retryCount:
                type: integer
                default: 0
              executionMode:
                type: string
                enum: ["sequential", "parallel"]
                default: "sequential"
                description: "Execution mode for multiple inputs (sequential/parallel)"
              failFast:
                type: boolean
                default: false
                description: "Fail fast on error - stop execution on first error"
              expect:
                type: string
                description: "Expected output pattern for validation"
              httpValidation:
                type: object
                properties:
                  expectedStatusCodes:
                    type: array
                    items:
                      type: integer
                  expectedResponseBody:
                    type: string
                  responseBodyMatch:
                    type: string
                  responseHeaders:
                    type: object
                    additionalProperties:
                      type: string
                  responseTimeout:
                    type: integer
                  followRedirects:
                    type: boolean
                  maxRedirects:
                    type: integer
                description: "HTTP response validation for GET/POST requests (Note: Currently only expect field is implemented)"
              outputValidation:
                type: object
                properties:
                  expectedOutput:
                    type: string
                  outputMatch:
                    type: string
                  successCriteria:
                    type: string
                  failureCriteria:
                    type: string
                  caseSensitive:
                    type: boolean
                  expectedFailureOutput:
                    type: string
                description: "Command output validation for CMD requests"
              schedule:
                type: string
                pattern: '^(\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*)$'
              environment:
                type: object
                additionalProperties:
                  type: string
              resources:
                type: object
                properties:
                  requests:
                    type: object
                    properties:
                      memory:
                        type: string
                      cpu:
                        type: string
                  limits:
                    type: object
                    properties:
                      memory:
                        type: string
                      cpu:
                        type: string
              dependencies:
                type: array
                items:
                  type: string
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Running", "Succeeded", "Failed", "Skipped"]
              message:
                type: string
              startTime:
                type: string
                format: date-time
              completionTime:
                type: string
                format: date-time
              result:
                type: object
                properties:
                  output:
                    type: string
                  errorCode:
                    type: string
                  errorMessage:
                    type: string
              retryCount:
                type: integer
              lastRetryTime:
                type: string
                format: date-time
```

### McallWorkflow CRD

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: mcallworkflows.mcall.tz.io
spec:
  group: mcall.tz.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              tasks:
                type: array
                items:
                  type: object
                  properties:
                    name:
                      type: string
                    type:
                      type: string
                      enum: ["cmd", "get", "post"]
                    input:
                      type: string
                    timeout:
                      type: integer
                      default: 30
                    retryCount:
                      type: integer
                      default: 0
                    dependencies:
                      type: array
                      items:
                        type: string
                    environment:
                      type: object
                      additionalProperties:
                        type: string
                    resources:
                      type: object
                      properties:
                        requests:
                          type: object
                          properties:
                            memory:
                              type: string
                            cpu:
                              type: string
                        limits:
                          type: object
                          properties:
                            memory:
                              type: string
                            cpu:
                              type: string
              schedule:
                type: string
                pattern: '^(\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*) (\*|(\d+|\*/\d+)(,\d+)*)$'
              concurrency:
                type: integer
                default: 1
              timeout:
                type: integer
                default: 300
              environment:
                type: object
                additionalProperties:
                  type: string
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Running", "Succeeded", "Failed", "Skipped"]
              message:
                type: string
              startTime:
                type: string
                format: date-time
              completionTime:
                type: string
                format: date-time
              taskStatuses:
                type: object
                additionalProperties:
                  type: object
                  properties:
                    phase:
                      type: string
                      enum: ["Pending", "Running", "Succeeded", "Failed", "Skipped"]
                    message:
                      type: string
                    startTime:
                      type: string
                      format: date-time
                    completionTime:
                      type: string
                      format: date-time
                    result:
                      type: string
                    error:
                      type: string
```


## Controller Implementation

### Controller Structure

**Location**: `controller/controller.go`

The controller is implemented as a Kubernetes controller-runtime reconciler with the following key components:

#### 1. McallTaskReconciler
**Location**: `controller/controller.go:613-616`
- Main reconciler struct
- Implements `client.Client` and `*runtime.Scheme`

#### 2. Reconcile Function
**Location**: `controller/controller.go:625-678`
- Main reconciliation loop
- Handles different phases: Pending, Running, Succeeded, Failed
- Initializes status and delegates to phase handlers

#### 3. Phase Handlers
**Location**: `controller/controller.go:680-885`
- `handlePending()`: Check dependencies and create execution pod
- `handleRunning()`: Execute task based on type and input
- `handleCompleted()`: Clean up resources
- `handleDeletion()`: Remove finalizers

#### 4. Task Execution Logic
**Location**: `controller/controller.go:721-885`
- Execute task based on type (cmd, get, post)
- Handle JSON input parsing for multiple commands
- Support sequential and parallel execution modes
- Implement failFast logic for error handling

#### 5. TaskWorker Implementation
**Location**: `controller/controller.go:150-272`
- `TaskWorker` struct for individual task execution
- `Execute()` method for command/HTTP execution
- Result channel communication for parallel execution

#### 6. Execution Modes
**Location**: `controller/controller.go:274-327` (Sequential), `controller/controller.go:329-406` (Parallel)
- `executeWorkersSequential()`: Execute tasks one by one
- `executeWorkersParallel()`: Execute tasks concurrently with goroutines

#### 7. JSON Input Parsing
**Location**: `controller/controller.go:408-450`
- `parseJSONInputs()` function
- Parse JSON string into array of input objects
- Each object contains: input, type, name, expect

#### 8. Response Validation
**Location**: `controller/controller.go:452-500`
- `validateResponse()` function
- Check if response matches expected pattern
- Support regex patterns and exact matches

### Task Execution Details

**Location**: `controller/controller.go:721-885`

The task execution logic handles different input types and execution modes:

#### 1. Command Execution (cmd)
- Parse JSON inputs for multiple commands
- Create TaskWorkers for each input
- Support sequential and parallel execution
- Implement failFast logic for error handling

#### 2. HTTP Execution (get/post)
- Execute HTTP requests with timeout
- Support GET and POST methods
- Handle response validation

#### 3. Status Updates
- Update task status based on execution results
- Store output, error code, and error message
- Set completion time

### Pod Creation

**Location**: `controller/controller.go:923-926`

The `createExecutionPod()` function creates Kubernetes pods for task execution:
- Generate unique pod names with timestamps
- Set appropriate labels and annotations
- Configure container specifications
- Set controller references for ownership

### Execution Modes Implementation

**Location**: `controller/controller.go:274-406`

#### Sequential Execution
- Execute tasks one by one
- Stop on first error if failFast is enabled
- Collect results in order

#### Parallel Execution
- Execute tasks concurrently using goroutines
- Use channels for result communication
- Cancel remaining tasks on first error if failFast is enabled

### TaskWorker Implementation

**Location**: `controller/controller.go:150-272`

The TaskWorker struct and related functions handle individual task execution:

#### 1. TaskWorker Struct
- `input`: Command or URL to execute
- `inputType`: Type of input (cmd, get, post)
- `name`: Worker name for logging
- `expect`: Expected output pattern for validation
- `result`: Channel for result communication

#### 2. NewTaskWorker Function
- Create new TaskWorker instance
- Initialize result channel
- Set up input parameters

#### 3. Execute Method
- Execute command or HTTP request
- Validate output against expect pattern
- Send result through channel

### Response Validation

**Location**: `controller/controller.go:452-500`

The `validateResponse()` function handles output validation:
- Check if response matches expected pattern
- Support regex patterns and exact matches
- Validate JSON structure for complex responses
- Return validation result for task success/failure

## Development Guide

### Prerequisites

**Required Tools**:
- Go 1.19+
- kubectl
- Helm 3.x
- Docker
- controller-gen
- kubebuilder

**Installation Commands**: See `tests/scripts/local-test.sh` for setup script

### Code Generation with controller-gen

This project uses `controller-gen` for automatic code generation from kubebuilder markers. The following commands are available in the Makefile:

#### Available Commands

```bash
# Generate all code (DeepCopy, CRDs, RBAC)
make generate

# Generate only DeepCopy methods
make generate-objects

# Generate only CRDs
make generate-crds

# Generate only RBAC permissions
make generate-rbac
```

#### When to Use Code Generation

**1. API Type Changes**
```bash
# After modifying api/v1/*_types.go files
make generate-objects
```

**2. CRD Schema Updates**
```bash
# After adding/removing kubebuilder markers like:
# //+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
# //+kubebuilder:validation:Required
make generate-crds
```

**3. RBAC Permission Changes**
```bash
# After modifying RBAC markers like:
# //+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
make generate-rbac
```

**4. Full Regeneration**
```bash
# After major changes to API types or controllers
make generate
```

#### Prerequisites

Install `controller-gen` if not already installed:
```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

#### Generated Files

- **DeepCopy Methods**: `api/v1/zz_generated.deepcopy.go`
- **CRD Definitions**: `helm/mcall-operator/templates/crds/*.yaml`
- **RBAC Permissions**: `helm/mcall-operator/templates/rbac.yaml`

### Local Development Setup

**Location**: `tests/scripts/local-test.sh`

The local development setup includes:
1. Start local Kubernetes cluster (minikube)
2. Install CRDs
3. Build and run controller locally
4. Test with example tasks

### Hot Reload Development

**Location**: `.air.toml` (if created)

For hot reload development:
1. Install air tool
2. Create `.air.toml` configuration
3. Run `air` for automatic rebuilds

### Testing

**Location**: `controller/controller_test.go`

Available test commands:
- Unit tests: `go test ./...`
- Integration tests: `go test -tags=integration ./...`
- Coverage: `go test -cover ./...`
- Specific test: `go test -run TestMcallTaskReconciler ./controller/`

## Development & Testing

### Makefile Integration

The project includes a comprehensive Makefile that integrates all testing and development workflows:

#### Testing Commands
```bash
# Local development testing
make test-debug          # Run tests with debugging information
make test-specific       # Run specific test function
make test-verbose        # Run all tests with verbose logging
make test-coverage       # Run tests with coverage analysis
make test-benchmark      # Run benchmark tests
make test-all           # Run all local tests

# Integration testing
make test-cleanup       # Run cleanup integration test (requires cluster)
make test-jenkins       # Run Jenkins-style validation tests
make validate           # Run all validation tests (no cluster required)
make integration        # Run all integration tests (requires cluster)
```

#### Build & Deployment Commands
```bash
# Build commands
make build              # Build controller binary
make build-docker       # Build Docker image

# Deployment commands
make deploy             # Deploy to cluster
make deploy-dev         # Deploy to dev environment
make deploy-staging     # Deploy to staging environment
```

#### Cleanup Commands
```bash
make clean              # Clean test results and build artifacts
make clean-docker       # Clean Docker images
make clean-all          # Clean everything
```

### Test Scripts Architecture

The project uses a layered testing approach:

#### 1. Makefile Commands (Developer Convenience)
- **Purpose**: Fast, convenient testing for developers
- **Use Case**: Local development, debugging, quick validation
- **Dependencies**: Minimal (Go, basic tools)

#### 2. Integration Scripts (CI/CD Integration)
- **`test-cleanup.sh`**: CRD cleanup functionality testing
- **`jenkins-test.sh`**: CI/CD pipeline validation
- **Use Case**: Automated testing, CI/CD pipelines
- **Dependencies**: kubectl, helm, cluster access

#### 3. Comprehensive Test Suite
- **`run-tests.sh`**: 18 different test cases
- **Use Case**: Regression testing, comprehensive validation
- **Dependencies**: Full cluster environment

### Development Workflow

#### 1. Local Development
```bash
# Start with debugging
make test-debug

# Run specific tests
make test-specific

# Check coverage
make test-coverage
```

#### 2. Pre-commit Validation
```bash
# Run all validation tests
make validate

# Build and test
make build
make test-all
```

#### 3. Integration Testing
```bash
# Test with cluster (requires kubectl access)
make integration

# Test cleanup functionality
make test-cleanup
```

#### 4. CI/CD Pipeline
```bash
# Jenkins/GitHub Actions
./tests/scripts/jenkins-test.sh $BUILD_NUMBER $BRANCH $NAMESPACE $VALUES_FILE
./tests/scripts/test-cleanup.sh
```

### Debugging Support

#### VS Code Debugging
- Set breakpoints in `controller_test.go`
- Use **F11** (Step Into) to debug `controller.go` functions
- See [DEBUG_GUIDE.md](DEBUG_GUIDE.md) for detailed instructions

#### Terminal Debugging
```bash
# Debug specific test
make test-specific

# Run with verbose output
make test-verbose

# Run with coverage
make test-coverage
```

### Benefits of Makefile Integration

1. **Developer Experience**: Simple commands for common tasks
2. **Consistency**: Standardized commands across team
3. **CI/CD Compatibility**: Scripts can be called directly with parameters
4. **Maintainability**: Centralized command definitions
5. **Extensibility**: Easy to add new commands

### Help and Documentation
```bash
# Show all available commands
make help
```

## Deployment Guide

### Helm Chart Structure

**Location**: `helm/mcall-operator/`

The Helm chart includes:
- `Chart.yaml`: Chart metadata
- `values.yaml`: Default values
- `values-dev.yaml`: Development values
- `values-staging.yaml`: Staging values
- `values.yaml`: Production values
- `crds/`: CRD definitions
- `templates/`: Kubernetes resources

### Environment-specific Values

**Location**: `helm/mcall-operator/values-*.yaml`

#### Development Configuration
- Single replica for development
- Lower resource limits
- Debug logging enabled
- Webhook disabled

#### Production Configuration
- Multiple replicas for high availability
- Higher resource limits
- Info level logging
- Webhook enabled with cert-manager

### Deployment Commands

**Location**: `ci/k8s.sh`

Available deployment commands:
- Development: `helm install mcall-operator-dev ./helm/mcall-operator --namespace mcall-dev --create-namespace --values ./helm/mcall-operator/values-dev.yaml`

- Staging: `helm install mcall-operator-staging ./helm/mcall-operator --namespace mcall-staging --create-namespace --values ./helm/mcall-operator/values-staging.yaml`
- Production: `helm install mcall-operator-prod ./helm/mcall-operator --namespace mcall-system --create-namespace --values ./helm/mcall-operator/values.yaml --wait --timeout=10m`

### Upgrade and Rollback

**Location**: `ci/k8s.sh`

Available upgrade/rollback commands:
- Upgrade: `helm upgrade mcall-operator-prod ./helm/mcall-operator --namespace mcall-system --values ./helm/mcall-operator/values.yaml --set image.tag=1.1.0`
- Rollback: `helm rollback mcall-operator-prod 1 -n mcall-system`
- History: `helm history mcall-operator-prod -n mcall-system`

## Jenkins CI/CD

### Pipeline Structure

**Location**: `ci/Jenkinsfile`

The Jenkins pipeline includes:
- Kubernetes agent with Docker, kubectl, and Helm containers
- Environment variables for branch-based configuration
- Multi-stage pipeline: Checkout, Configuration, Build, Deploy, Test

### Pipeline Stages

**Location**: `ci/Jenkinsfile`

#### 1. Checkout Stage
- Check out source code from Git

#### 2. Configuration Stage
- Run configuration script based on branch
- Set values file and namespace based on branch:
  - `main` → `values.yaml`, `mcall-system`
  - `qa` → `values-staging.yaml`, `mcall-system`
  - `dev` → `values-dev.yaml`, `mcall-dev`

#### 3. Build & Push Image Stage
- Build Docker image with build number tag
- Push to Docker registry

#### 4. Helm Chart Validation Stage
- Run `helm lint` to validate chart syntax
- Run `helm template --dry-run` to validate templates

#### 5. Deploy CRDs Stage
- Deploy CRDs using `ci/k8s.sh` script
- Apply CRD definitions to cluster

#### 6. Deploy Helm Chart Stage
- Install/upgrade Helm chart with new image tag
- Wait for deployment to complete

#### 7. Post-Deployment Tests Stage
- Run post-deployment tests using `ci/k8s.sh` script
- Validate deployment success

#### 8. Post Actions
- Always: Check deployment status
- Success: Send Slack notification
- Failure: Send Slack notification

### Jenkins Job Configuration

**Location**: Jenkins UI

#### Pipeline Job Settings
- Name: `mcall-operator-deployment`
- Type: Pipeline
- Definition: Pipeline script from SCM
- SCM: Git
- Repository: `https://github.com/doohee323/tz-mcall-operator.git`
- Script Path: `ci/Jenkinsfile`

#### Build Triggers
- GitHub hook trigger for GITScm polling
- Poll SCM: `H/5 * * * *` (5 minutes)

#### Environment Variables
- `GIT_BRANCH`: Git branch name
- `STAGING`: Environment (prod/qa/dev)
- `BUILD_NUMBER`: Build number
- `DOCKER_NAME`: Docker image name
- `APP_NAME`: Application name
- `REGISTRY`: Docker registry

## Troubleshooting

### Common Issues

#### 1. CRD Installation Failures
**Commands**: See troubleshooting section in `README.md`
- Check CRD status: `kubectl get crd | grep mcall`
- Check CRD details: `kubectl describe crd mcalltasks.mcall.tz.io`
- Check events: `kubectl get events --sort-by='.lastTimestamp'`

#### 2. Controller Startup Issues
**Commands**: See troubleshooting section in `README.md`
- Check controller logs: `kubectl logs -n mcall-system deployment/mcall-operator`
- Check RBAC permissions: `kubectl auth can-i create mcalltasks --as=system:serviceaccount:mcall-system:mcall-operator`
- Check service account: `kubectl get serviceaccount mcall-operator -n mcall-system`

#### 3. Task Execution Failures
**Commands**: See troubleshooting section in `README.md`
- Check task status: `kubectl get mcalltasks`
- Check execution pods: `kubectl get pods -l task-name=<task-name>`
- Check pod logs: `kubectl logs <pod-name>`

#### 4. Helm Chart Issues
**Commands**: See troubleshooting section in `README.md`
- Check Helm release status: `helm status mcall-operator -n mcall-system`
- Check Helm values: `helm get values mcall-operator -n mcall-system`
- Check Helm history: `helm history mcall-operator -n mcall-system`

### Debug Commands

**Location**: `README.md` troubleshooting section

Available debug commands:
- Enable debug logging
- Check controller metrics
- Check webhook configuration

## Leader Election Implementation

### Overview

The mcall CRD system uses Kubernetes standard Leader Election functionality to ensure only one controller instance is active at a time, preventing duplicate task processing.

### Leader Election Features

#### 1. Standard Kubernetes Leader Election
- Uses Kubernetes `coordination.k8s.io/v1/leases` resource for leader election
- Automatic failover when the current leader fails or terminates
- Prevents duplicate task processing across multiple controller instances

#### 2. Controller Reconciliation
- Each controller instance reconciles McallTask resources
- Only the leader processes tasks, followers remain idle
- Tasks are processed based on CRD events, not scheduled generation

#### 3. Task Processing
- Tasks are executed directly by the controller (no separate worker pods)
- JSON input parsing for multiple command execution
- Sequential and parallel execution modes supported

### Leader Election Setup

#### Environment Variables
The following environment variables are available:
- `RECONCILE_INTERVAL`: Controller reconciliation interval (default: 5 seconds)
- `TASK_TIMEOUT`: Task execution timeout (default: 5 seconds)

#### RBAC Permissions
The controller requires the following permissions:
- `mcall.tz.io/mcalltasks`: Full CRUD operations on McallTask resources
- `core/pods`: Create and manage execution pods (if needed)
- `core/configmaps`: Read configuration and store results

#### Operation Flow
1. **Startup**: Controller starts with leader election enabled
2. **Leader Election**: One controller instance becomes leader
3. **Event Processing**: Leader processes McallTask CRD events
4. **Task Execution**: Tasks are executed directly by the controller
5. **Status Update**: Task results are stored in McallTask status

#### Monitoring
```bash
# Check controller pods
kubectl get pods -n mcall-system -l app.kubernetes.io/name=tz-mcall-operator

# Check leader election lease
kubectl get lease tz-mcall-operator -n mcall-system

# Check task status
kubectl get mcalltasks -n mcall-system
```


This technical documentation provides comprehensive information for developers, operators, and DevOps engineers working with the mcall CRD system.
