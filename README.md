# tz-mcall-operator

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.19+-blue.svg)](https://kubernetes.io)

**Kubernetes Task Operator** - Execute commands and HTTP requests with built-in scheduling, monitoring, and management using Custom Resource Definitions.

## What does it do?

**Two main things:**
1. **Run Commands** - Execute shell commands in Kubernetes pods
2. **Make HTTP Requests** - GET/POST requests to any URL

**Why Kubernetes Operator?** 
- Kubernetes manages everything (scheduling, monitoring, logs, scaling)
- No need for external cron servers or job schedulers
- Built-in retry, timeout, and error handling
- Native Kubernetes integration with Custom Resource Definitions

## Key Features

### Core Capabilities
- **Command Execution**: Run any shell command
- **HTTP Requests**: GET/POST to any URL  
- **MCP Client**: Call other MCP servers (Jenkins, GitHub Actions, etc.)
- **Scheduling**: Cron-based scheduling (like `crontab`)
- **Parallel Execution**: Run multiple commands/requests at once
- **Error Handling**: Automatic retry and timeout
- **Monitoring**: Built-in Kubernetes monitoring and logging
- **Simple YAML**: Define everything in YAML files

### Advanced Features
- **Task Result Passing**: Pass results between tasks with inputSources and inputTemplate
- **Conditional Execution**: Execute tasks based on previous task results
- **MCP Client Integration**: Orchestrate external services via MCP protocol
  - Multiple authentication methods (API Key, Bearer, Basic Auth)
  - Kubernetes Secret integration for secure credentials
  - Support for any MCP-compliant server
- **DAG Visualization**: Real-time workflow visualization with execution history
- **MCP Server**: AI-assisted task management via Model Context Protocol
- **API Key Authentication**: Secure access control for MCP endpoints

## 📋 Table of Contents

- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [MCP Server](#mcp-server)
- [CRD Types](#crd-types)
- [Usage Examples](#usage-examples)
- [Development & Testing](#development--testing)
- [Deployment](#deployment)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## 🏗️ Architecture

The operator consists of two main CRD types:

- **McallTask**: Individual task definitions (commands, HTTP requests)
- **McallWorkflow**: Grouped tasks with dependencies and execution order

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

### Component Roles

- **CRDs**: Task definitions and execution metadata
- **Controller**: Manages CRD lifecycle and creates executor pods
- **Executor Pods**: Actual workers that execute the tasks
- **Leader Election**: Ensures only one controller handles scheduled tasks

## 🛠️ Installation

### Prerequisites

- Kubernetes cluster (v1.19+)
- Helm 3.x
- kubectl

### Quick Install

```bash
# Clone the repository
git clone https://github.com/doohee323/tz-mcall-operator.git
cd tz-mcall-operator

# Install CRDs and Controller
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml

# Verify installation
kubectl get pods -n mcall-system
kubectl get crd | grep mcall
```

## Quick Start

### 1. Install
```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml
```

### 2. Run a Command
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: hello-world
spec:
  type: cmd
  input: "echo 'Hello from Kubernetes!'"
```

### 3. Make HTTP Request
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: health-check
spec:
  type: get
  input: "https://api.example.com/health"
  schedule: "*/5 * * * *"  # Every 5 minutes
```

### Apply and Check
```bash
# Apply the task
kubectl apply -f task.yaml

# Check status
kubectl get mcalltasks
kubectl describe mcalltask hello-world
```

## 🤖 MCP Server

**NEW!** Control your Kubernetes tasks and workflows through AI assistants using Model Context Protocol (MCP).

### What is MCP Server?

The MCP Server enables AI assistants (like Claude, Cursor AI) to interact with tz-mcall-operator. You can create tasks, workflows, and monitor executions through natural language!

### Quick Setup

```bash
# Enable MCP Server with Helm
helm upgrade --install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.enabled=true \
  --set mcpServer.ingress.hosts[0].host=mcp.drillquiz.com
```

### Features

- ✅ **Natural Language Control**: "Create a daily backup task at 2 AM"
- ✅ **Task Management**: Create, monitor, and manage tasks through AI
- ✅ **Workflow Orchestration**: Build complex workflows with dependencies
- ✅ **Real-time Status**: Check task execution and logs
- ✅ **Secure**: Kubernetes RBAC-based access control

### Example Usage

```
You: "Create a health check task for https://api.example.com every 5 minutes"

AI: I'll create that for you.
    [automatically creates McallTask with proper configuration]

You: "Show me the status"

AI: [retrieves and displays task status]
```

### Documentation

- [MCP Server Guide](./MCP_SERVER_GUIDE.md) - Complete guide with integration details
- [Setup Guides](./docs/README.md) - Claude Desktop & Cursor setup
- [Usage Examples](./docs/USAGE_EXAMPLES.md) - Real-world examples
- [Quick Start](./mcp-server/QUICKSTART.md) - Get started in 5 minutes
- [Deployment Guide](./mcp-server/DEPLOYMENT.md) - Detailed deployment
- [Helm Chart Guide](./helm/mcall-operator/README.md) - Helm documentation

### Access

After deployment with ingress enabled:
- **Production**: `https://mcp.drillquiz.com`
- **Development**: `https://mcp-dev.drillquiz.com` (with values-dev.yaml)
- **Local**: `http://localhost:3000` (port-forward)

## Real Examples

### Daily Database Backup
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: daily-backup
spec:
  type: cmd
  input: "pg_dump mydb > /backup/backup_$(date +%Y%m%d).sql"
  schedule: "0 2 * * *"  # Daily at 2 AM
  timeout: 1800
```

### Health Check
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: api-health
spec:
  type: get
  input: "https://api.example.com/health"
  schedule: "*/5 * * * *"  # Every 5 minutes
  timeout: 10
```

### Call MCP Server (Jenkins Build Trigger)
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: trigger-jenkins-build
spec:
  type: mcp-client
  timeout: 60
  mcpConfig:
    serverUrl: http://jenkins-mcp-server:3000/mcp
    toolName: create_mcall_task
    arguments:
      name: build-job
      type: cmd
      input: "jenkins-cli build MyProject"
      timeout: 300
    auth:
      type: apiKey
      secretRef:
        name: jenkins-mcp-credentials
      secretKey: api-key
```

### Cleanup Old Files
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: cleanup-logs
spec:
  type: cmd
  input: "find /var/log -name '*.log' -mtime +30 -delete"
  schedule: "0 0 * * *"  # Daily at midnight
```


## Task Types

### McallTask - Run Commands or HTTP Requests

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: my-task
spec:
  type: cmd                    # cmd, get, or post
  input: "echo 'Hello World'"  # Command or URL
  schedule: "*/5 * * * *"      # Cron schedule (optional)
  timeout: 30                  # Timeout in seconds
  retryCount: 2                # Retry on failure
```

**Types:**
- `cmd`: Execute shell commands
- `get`: HTTP GET request  
- `post`: HTTP POST request
- `mcp-client`: Call MCP (Model Context Protocol) servers

**MCP Client Example:**
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-mcp-call
spec:
  type: mcp-client
  timeout: 60
  mcpConfig:
    serverUrl: http://jenkins-mcp-server:3000/mcp
    toolName: create_mcall_task
    arguments:
      name: build-job
      type: cmd
      input: "make build"
    auth:
      type: apiKey
      secretRef:
        name: jenkins-mcp-credentials
      secretKey: api-key
```

For more details, see: [MCP Client Guide](./examples/README-MCP-CLIENT.md)

### McallWorkflow - Chain Multiple Tasks

```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: backup-workflow
spec:
  tasks:
  - name: "check-db"
    type: cmd
    input: "kubectl get pods -l app=postgres | grep Running"
  - name: "backup-db"
    type: cmd
    input: "pg_dump mydb > backup.sql"
    dependencies: ["check-db"]  # Only run if check-db succeeds
  schedule: "0 2 * * *"        # Daily at 2 AM
```

**Key Features:**
- **Dependencies**: Tasks run in order
- **Error Handling**: Stop if any task fails
- **Scheduling**: Run entire workflow on schedule


## Why Use This?

| Traditional Cron | tz-mcall-operator |
|------------------|--------------|
| ❌ No monitoring | ✅ Kubernetes monitoring |
| ❌ No retry logic | ✅ Built-in retry |
| ❌ No scaling | ✅ Auto-scaling |
| ❌ Manual setup | ✅ YAML configuration |

## More Examples

### Parallel Health Checks
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: multi-health-check
spec:
  type: get
  inputs:
    - "https://api.example.com/health"
    - "https://auth.example.com/health"
    - "https://db.example.com/health"
  executionMode: "parallel"  # Check all at once
  timeout: 10
```

### Complex Workflow
```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: deployment-check
spec:
  tasks:
  - name: "check-pods"
    type: cmd
    input: "kubectl get pods | grep Running"
  - name: "test-api"
    type: get
    input: "https://api.example.com/health"
    dependencies: ["check-pods"]
  - name: "send-notification"
    type: post
    input: "https://slack.com/api/chat.postMessage"
    dependencies: ["test-api"]
```

## 🧪 Development & Testing

### Makefile Commands

The project includes a comprehensive Makefile for development and testing workflows:

#### Code Generation
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

#### Local Development & Testing
```bash
# Run tests with debugging information
make test-debug

# Run specific test function
make test-specific

# Run all tests with verbose logging
make test-verbose

# Run tests with coverage
make test-coverage

# Run all local tests
make test-all
```

#### Integration & Cleanup Tests
```bash
# Run cleanup integration test (requires cluster)
make test-cleanup

# Run Jenkins-style validation tests
make test-jenkins

# Run all validation tests (no cluster required)
make validate

# Run all integration tests (requires cluster)
make integration
```

#### Build & Deployment
```bash
# Build controller binary
make build

# Build Docker image
make build-docker

# Deploy to cluster
make deploy

# Deploy to specific environments
make deploy-dev
make deploy-staging
```

#### Cleanup
```bash
# Clean test results and build artifacts
make clean

# Clean Docker images
make clean-docker

# Clean everything
make clean-all
```

### Test Scripts

The project includes specialized test scripts for different scenarios:

#### `test-cleanup.sh` - Integration Testing
- **Purpose**: Test CRD cleanup functionality
- **Requirements**: kubectl, helm, cluster access
- **Features**:
  - Install chart
  - Create test resources
  - Verify cleanup on uninstall
  - Force cleanup if needed

#### `jenkins-test.sh` - CI/CD Validation
- **Purpose**: Jenkins-style validation tests
- **Requirements**: helm, kubectl (dry-run only)
- **Features**:
  - Helm chart validation
  - Template rendering
  - CRD validation
  - Example manifests validation
  - Docker image build test

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

### Debugging

#### VS Code Debugging
1. Set breakpoints in `controller_test.go`
2. Press **F5** or select "Debug Tests"
3. Use **F11** (Step Into) to debug `controller.go` functions
4. See [DEBUG_GUIDE.md](DEBUG_GUIDE.md) for detailed instructions

#### Terminal Debugging
```bash
# Debug specific test
make test-specific

# Run with verbose output
make test-verbose

# Run with coverage
make test-coverage
```

### Help
```bash
# Show all available commands
make help
```

## Installation

### Development
```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml
```

### Production
```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-operator/values.yaml
```

## Monitoring

### Check Status
```bash
# Check tasks
kubectl get mcalltasks
kubectl get mcallworkflows

# Check controller
kubectl get pods -n mcall-system

# View logs
kubectl logs -n mcall-system -l app.kubernetes.io/name=mcall-operator
```

### Task Status
- `Pending`: Waiting to run
- `Running`: Currently executing  
- `Succeeded`: Completed successfully
- `Failed`: Execution failed

## Troubleshooting

### Task Not Running
```bash
# Check task status
kubectl describe mcalltask <task-name>

# Check controller logs
kubectl logs -n mcall-system deployment/mcall-operator
```

### Common Issues
- **Task stuck in Pending**: Check dependencies and controller status
- **Task Failed**: Check logs and input validation
- **Controller not starting**: Check RBAC permissions and resources

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go coding standards
- Add tests for new features
- Update documentation
- Ensure backward compatibility

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- Dewey Hong - Initial work - [doohee323](https://github.com/doohee323)

---

**Made with ❤️ by the tz-mcall team**
