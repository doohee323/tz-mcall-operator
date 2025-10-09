# McallOperator MCP Server

Model Context Protocol (MCP) server for tz-mcall-operator. This server enables AI assistants to manage Kubernetes tasks and workflows through the MCP protocol.

## Features

- **Task Management**: Create, read, update, and delete McallTasks
- **Workflow Management**: Manage complex task workflows with dependencies
- **Real-time Status**: Check task execution status and retrieve logs
- **Kubernetes Native**: Direct integration with Kubernetes API
- **Secure**: Uses Kubernetes RBAC for authentication and authorization

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

## Installation

### Prerequisites

- Kubernetes cluster with tz-mcall-operator installed
- kubectl configured with cluster access
- Docker (for building image)

### Build and Deploy

```bash
# Build Docker image
docker build -t mcall-operator-mcp-server:latest .

# Deploy to Kubernetes
kubectl apply -f k8s/

# Or use Helm (integrated with mcall-operator chart)
helm upgrade --install mcall-operator ../helm/mcall-operator \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.host=mcp.drillquiz.com
```

## Configuration

### Environment Variables

- `KUBERNETES_SERVICE_HOST`: Auto-set when running in Kubernetes
- `NODE_ENV`: Set to `production` for production deployments

### Kubernetes RBAC

The MCP server requires the following permissions:
- Read/Write access to `mcalltasks.mcall.tz.io`
- Read/Write access to `mcallworkflows.mcall.tz.io`
- Read access to pods (for log retrieval)

## Usage Examples

### Create a Simple Task

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

### Create a Scheduled Health Check

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

### Create a Workflow

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

## Development

### Local Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run locally (requires kubeconfig)
npm start

# Watch mode
npm run watch
```

### Testing

```bash
# Install in dev cluster
kubectl config use-context dev-cluster

# Create test task
cat <<EOF | kubectl apply -f -
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: test-task
spec:
  type: cmd
  input: "echo 'test'"
EOF

# Check status
kubectl get mcalltask test-task
```

## Architecture

```
┌─────────────────┐
│   AI Assistant  │
└────────┬────────┘
         │ MCP Protocol
         │
┌────────▼────────┐
│   MCP Server    │
└────────┬────────┘
         │ Kubernetes API
         │
┌────────▼────────────────┐
│  McallOperator          │
│  - McallTask CRD        │
│  - McallWorkflow CRD    │
└─────────────────────────┘
```

## Security Considerations

1. **RBAC**: Server uses Kubernetes ServiceAccount with limited permissions
2. **Network Policies**: Restrict ingress/egress as needed
3. **TLS**: Enable TLS for ingress endpoints
4. **Authentication**: Consider adding additional auth layer for public exposure

## Troubleshooting

### Common Issues

**Connection Failed**
```bash
# Check if operator is running
kubectl get pods -n mcall-system

# Check RBAC permissions
kubectl auth can-i create mcalltasks --as=system:serviceaccount:mcall-system:mcp-server
```

**Task Not Executing**
```bash
# Check task status
kubectl describe mcalltask <task-name>

# Check operator logs
kubectl logs -n mcall-system deployment/mcall-operator
```

## License

MIT License - See parent project LICENSE file

## Links

- [tz-mcall-operator](https://github.com/doohee323/tz-mcall-operator)
- [Model Context Protocol](https://modelcontextprotocol.io)

