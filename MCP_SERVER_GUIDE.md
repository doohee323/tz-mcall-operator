# McallOperator MCP Server Guide

## Overview

Model Context Protocol (MCP) server for tz-mcall-operator. This server enables AI assistants to manage Kubernetes tasks and workflows through the MCP protocol.

**Important**: MCP Server is **fully integrated** with tz-mcall-operator. No separate deployment is needed - it's automatically built and deployed through the existing Jenkins pipeline.

## Key Features

- **Task Management**: Create, read, update, and delete McallTasks
- **Workflow Management**: Manage complex task workflows with dependencies
- **Real-time Status**: Check task execution status and retrieve logs
- **Kubernetes Native**: Direct integration with Kubernetes API
- **Secure**: Uses Kubernetes RBAC for authentication and authorization
- **Automated Build**: Integrated with Jenkins CI/CD pipeline

## Architecture

```
┌─────────────────────────────┐
│     AI Assistant (Claude)   │
│     - Cursor                 │
│     - Claude Desktop         │
└──────────┬──────────────────┘
           │ MCP Protocol
           │ (SSE/HTTP)
           │
┌──────────▼──────────────────┐
│   MCP Server                │
│   - HTTP Endpoints          │
│   - Tool Handlers           │
│   - Kubernetes Client       │
└──────────┬──────────────────┘
           │ Kubernetes API
           │
┌──────────▼──────────────────┐
│   tz-mcall-operator         │
│   - McallTask CRD           │
│   - McallWorkflow CRD       │
│   - Controller              │
└─────────────────────────────┘
```

## Integration with tz-mcall-operator

### Single Project, Single Pipeline

MCP Server is part of the tz-mcall-operator project:

```
tz-mcall-operator/
├── api/                    # Operator CRD definitions
├── cmd/                    # Operator entry point
├── controller/             # Operator controllers
├── mcp-server/            # MCP Server (integrated)
│   ├── src/
│   ├── Dockerfile
│   └── package.json
├── helm/mcall-operator/   # Single Helm chart (operator + MCP)
├── ci/
│   ├── Jenkinsfile        # Integrated pipeline
│   └── k8s.sh
└── Makefile
```

### Automated Build Process

Jenkins pipeline automatically builds **both images**:

```groovy
stage('Build & Push Images') {
    // 1. Build Operator image
    docker build -f docker/Dockerfile -t doohee323/tz-mcall-operator:${BUILD_NUMBER} .
    
    // 2. Build MCP Server image
    docker build -f mcp-server/Dockerfile -t doohee323/mcall-operator-mcp-server:${BUILD_NUMBER} ./mcp-server
}
```

### Single Helm Chart Deployment

Helm chart manages both operator and MCP server together:

```yaml
# values-dev.yaml
mcpServer:
  enabled: true  # Set to true to deploy MCP server
```

## Available Tools

### McallTask Operations

1. **create_mcall_task**: Create a new task
   - Types: `cmd` (shell command), `get` (HTTP GET), `post` (HTTP POST)
   - Supports scheduling, retries, timeouts, and environment variables

2. **get_mcall_task**: Get task details and status
3. **list_mcall_tasks**: List all tasks with optional filtering
4. **delete_mcall_task**: Delete a task
5. **get_mcall_task_logs**: Retrieve task execution logs

### McallWorkflow Operations

1. **create_mcall_workflow**: Create a workflow with multiple tasks
   - Supports task dependencies and execution order
   - Schedule entire workflows

2. **get_mcall_workflow**: Get workflow details
3. **list_mcall_workflows**: List all workflows
4. **delete_mcall_workflow**: Delete a workflow

## Deployment

### Branch-Based Image Tagging Strategy

| Branch | Operator Tag | MCP Server Tag | MCP Enabled |
|--------|--------------|----------------|-------------|
| main | `${BUILD_NUMBER}` | `${BUILD_NUMBER}`, `latest` | No (optional) |
| qa | `${BUILD_NUMBER}` | `${BUILD_NUMBER}`, `staging` | Configurable |
| dev | `${BUILD_NUMBER}`, `latest` | `${BUILD_NUMBER}`, `dev` | Yes |

### Deployment Control via Helm Values

Control MCP Server deployment through Helm values files:

#### Production (values.yaml)
```yaml
mcpServer:
  enabled: false  # Disabled by default in production
```

#### Development (values-dev.yaml)
```yaml
mcpServer:
  enabled: true   # Enabled in dev
  image:
    repository: doohee323/mcall-operator-mcp-server
    tag: "dev"    # Jenkins overrides with BUILD_NUMBER
```

### Quick Deployment

```bash
# Development (MCP server enabled by default)
helm upgrade --install mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml

# Production (enable MCP server explicitly)
helm upgrade --install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.enabled=true \
  --set mcpServer.ingress.hosts[0].host=mcp.drillquiz.com
```

## Local Development

### Build Docker Images

```bash
# Build operator image only
make build-docker

# Build MCP server image only
make build-mcp-docker

# Build both
make build-docker-all
```

### Install Dependencies

```bash
cd mcp-server
npm install
```

### Build TypeScript

```bash
npm run build
```

### Run Locally

```bash
# HTTP mode (for Kubernetes deployment testing)
SERVER_MODE=http npm start

# Stdio mode (for MCP client testing)
SERVER_MODE=stdio npm start
```

Server endpoints (HTTP mode):
- Health check: http://localhost:3000/health
- Ready check: http://localhost:3000/ready
- Server info: http://localhost:3000/
- MCP endpoint: http://localhost:3000/mcp

## Accessing the MCP Server

### Development Environment

```bash
# Via Ingress (after DNS configuration)
https://mcp-dev.drillquiz.com

# Via Port Forward
kubectl port-forward -n mcall-dev svc/tz-mcall-operator-mcp-server 3000:80
curl http://localhost:3000/health
```

### Production Environment

```bash
# Via Ingress
https://mcp.drillquiz.com
```

## Usage Examples

### Example 1: Create a Simple Task

```json
{
  "name": "create_mcall_task",
  "arguments": {
    "name": "hello-world",
    "type": "cmd",
    "input": "echo 'Hello from MCP!'"
  }
}
```

### Example 2: Create a Scheduled Health Check

```json
{
  "name": "create_mcall_task",
  "arguments": {
    "name": "api-health-check",
    "type": "get",
    "input": "https://api.example.com/health",
    "schedule": "*/5 * * * *",
    "timeout": 10
  }
}
```

### Example 3: Create a Workflow

```json
{
  "name": "create_mcall_workflow",
  "arguments": {
    "name": "deployment-workflow",
    "tasks": [
      {
        "name": "check-health",
        "type": "get",
        "input": "https://api.example.com/health"
      },
      {
        "name": "run-tests",
        "type": "cmd",
        "input": "kubectl run test-runner --image=test:latest",
        "dependencies": ["check-health"]
      },
      {
        "name": "notify",
        "type": "post",
        "input": "https://slack.com/webhook",
        "dependencies": ["run-tests"]
      }
    ]
  }
}
```

## Jenkins Integration

### No Separate Deployment Needed

MCP Server is automatically built and deployed through Jenkins:

1. **Push code** to repository
2. **Jenkins automatically**:
   - Builds operator image
   - Builds MCP server image
   - Deploys both via Helm

### Jenkins Pipeline Changes

```groovy
// Added MCP_DOCKER_NAME environment variable
environment {
    DOCKER_NAME = "tz-mcall-operator"
    MCP_DOCKER_NAME = "mcall-operator-mcp-server"  // Added
}

// Build both images
stage('Build & Push Images') {
    // Build operator image
    // Build MCP server image
}
```

### Deployment Verification

```bash
# Check operator pods
kubectl get pods -n mcall-dev

# Check MCP server pods
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Check services
kubectl get svc -n mcall-dev

# Check MCP server logs
kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f
```

## Configuration

### Helm Values Configuration

```yaml
mcpServer:
  enabled: true
  
  image:
    repository: doohee323/mcall-operator-mcp-server
    tag: "latest"
    pullPolicy: IfNotPresent
  
  replicas: 2
  
  service:
    type: ClusterIP
    port: 80
    targetPort: 3000
  
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: mcp.drillquiz.com
        paths:
          - path: /
            pathType: Prefix
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
  
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  
  env:
    NODE_ENV: production
    DEFAULT_NAMESPACE: mcall-system
```

## Local Testing

### Test Jenkins Pipeline Locally

```bash
# Quick validation (skip Docker build)
make jenkins-sim

# Full simulation (with Docker build)
./scripts/local-jenkins-test.sh local-test dev

# Production simulation
./scripts/local-jenkins-test.sh local-test main
```

### Test Helm Chart Only

```bash
# Lint
make helm-lint

# Render template
make helm-template

# Run all Helm tests
make helm-test
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Check logs
kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server
```

### RBAC Permission Issues

```bash
# Check ServiceAccount
kubectl get sa -n mcall-dev tz-mcall-operator-mcp-server

# Test permissions
kubectl auth can-i create mcalltasks \
  --as=system:serviceaccount:mcall-dev:tz-mcall-operator-mcp-server
```

### Ingress Issues

```bash
# Check ingress status
kubectl describe ingress -n mcall-dev

# Check nginx controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Test DNS
nslookup mcp-dev.drillquiz.com
```

### Image Pull Errors

```bash
# Check if images exist
docker pull doohee323/mcall-operator-mcp-server:dev

# Check image pull secrets
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server -o yaml | grep -A 5 imagePullSecrets
```

## Security Considerations

1. **RBAC**: Server uses ServiceAccount with minimal required permissions
2. **TLS**: HTTPS enforced via Ingress
3. **CORS**: Configure allowed origins appropriately
4. **Rate Limiting**: Set via nginx ingress annotations
5. **Network Policy**: Restrict pod-to-pod communication (optional)

## Advantages

### ✅ Single Pipeline
- One Jenkins job manages everything
- Synchronized build numbers
- Consistent deployment timing

### ✅ Simple Management
- One Helm chart
- One Git repository
- Easy control via values files

### ✅ Optional Deployment
- Disable in production if not needed
- Enable only in dev/staging
- Per-environment control

### ✅ Automation
- Jenkins handles everything automatically
- No manual intervention required
- Consistent deployment process

## FAQ

### Q: Do I need to deploy MCP server separately?
**A:** No. Jenkins pipeline automatically builds and deploys it.

### Q: How to deploy MCP server to production?
**A:** Set `mcpServer.enabled: true` in `helm/mcall-operator/values.yaml` and commit.

### Q: How to redeploy only MCP server?
**A:** Re-run Jenkins job to rebuild both images, or change Helm values and run `helm upgrade`.

### Q: How to develop MCP server locally?
**A:** Run `cd mcp-server && npm install && npm run dev` with kubeconfig configured.

### Q: How to disable MCP server?
**A:** Set `mcpServer.enabled: false` in values file.

## API Endpoints

- `GET /`: Server info and available tools list
- `GET /health`: Health check
- `GET /ready`: Readiness check
- `GET /mcp`: MCP SSE endpoint (client connection)
- `POST /mcp`: MCP message handling (info only)

## Documentation

- [MCP Server README](./mcp-server/README.md) - Technical details
- [Deployment Guide](./mcp-server/DEPLOYMENT.md) - Detailed deployment
- [Quick Start](./mcp-server/QUICKSTART.md) - Get started in 5 minutes
- [CI/CD Guide](./ci/README.md) - Jenkins pipeline details

## License

MIT License - See parent project LICENSE file

## Links

- [tz-mcall-operator](https://github.com/doohee323/tz-mcall-operator)
- [Model Context Protocol](https://modelcontextprotocol.io)
