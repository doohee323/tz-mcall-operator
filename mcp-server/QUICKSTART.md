# MCP Server Quick Start

Quick guide to get started with tz-mcall-operator MCP server.

## Get Started in 5 Minutes

### 1. Prerequisites

- Kubernetes cluster access
- kubectl configured
- Helm 3.x installed
- tz-mcall-operator installed

### 2. Quick Deployment

```bash
# Clone repository (skip if already cloned)
cd tz-mcall-operator

# Deploy with Helm (MCP server enabled)
helm upgrade --install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --set mcpServer.enabled=true \
  --wait

# Verify deployment
kubectl get pods -n mcall-system
```

### 3. Local Access via Port Forward

```bash
# Set up port forward
kubectl port-forward -n mcall-system \
  svc/mcall-operator-mcp-server 3000:80

# Test in another terminal
curl http://localhost:3000/health
```

Expected output:
```json
{
  "status": "healthy"
}
```

### 4. Create Your First Task

In your MCP client (Cursor AI, Claude Desktop, etc.):

```
"Create a task named 'hello-world' that runs the command 'echo Hello from Kubernetes!'"
```

AI will automatically create the Task!

### 5. Check Task Status

```bash
# List tasks
kubectl get mcalltasks -n mcall-system

# View task details
kubectl describe mcalltask hello-world -n mcall-system
```

## Local Development Quick Start

### 1. Install Dependencies and Build

```bash
cd mcp-server
npm install
npm run build
```

### 2. Run Locally

```bash
# Run in HTTP mode
npm start

# Open in browser
open http://localhost:3000
```

### 3. Test

If kubeconfig is configured, you can immediately communicate with Kubernetes API.

```bash
# Health check
curl http://localhost:3000/health

# Server info
curl http://localhost:3000/
```

## Ingress Setup (External Access)

### 1. DNS Configuration

Point `mcp.drillquiz.com` to your cluster's ingress controller IP

### 2. Enable Ingress

```bash
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.enabled=true \
  --set mcpServer.ingress.hosts[0].host=mcp.drillquiz.com \
  --reuse-values
```

### 3. Test Access

```bash
curl https://mcp.drillquiz.com/health
```

## Next Steps

- Read the [Complete Guide](../MCP_SERVER_GUIDE.md)
- Check [Deployment Guide](./DEPLOYMENT.md)
- Review [Usage Examples](./README.md#usage-examples)
- See [Helm Chart Guide](../helm/mcall-operator/README.md) for local testing

## Troubleshooting

### Pod Not in Running State

```bash
kubectl describe pod -n mcall-system -l app.kubernetes.io/component=mcp-server
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server
```

### Port Forward Not Working

```bash
# Check service
kubectl get svc -n mcall-system

# Verify service health
kubectl describe svc mcall-operator-mcp-server -n mcall-system
```

### Need Help?

- See [Troubleshooting Guide](./DEPLOYMENT.md#troubleshooting)
- Check [Helm Chart Guide](../helm/mcall-operator/README.md)
- Open an issue on GitHub
