# mcall-operator Helm Chart

Official Helm chart for deploying the tz-mcall-operator and MCP server on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.x
- kubectl configured with cluster access

## Quick Start

### Installation

```bash
# Development environment
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml

# Production environment
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-operator/values.yaml
```

### Verification

```bash
# Check pods
kubectl get pods -n mcall-dev

# Check services
kubectl get svc -n mcall-dev

# Check CRDs
kubectl get crd | grep mcall
```

## Components

This Helm chart deploys:

1. **McallOperator** (CRD Controller)
   - Manages McallTask and McallWorkflow CRDs
   - Handles task scheduling and execution
   
2. **MCP Server** (Optional)
   - Model Context Protocol server for AI assistant integration
   - Enabled by default in dev, disabled in production

3. **Supporting Resources**
   - CRDs (McallTask, McallWorkflow)
   - RBAC (ServiceAccounts, Roles, RoleBindings)
   - Services (Metrics, Webhook, MCP)
   - Ingress (MCP Server - optional)
   - Jobs (Cleanup, Log cleanup, PostgreSQL init)

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Operator image repository | `doohee323/tz-mcall-operator` |
| `image.tag` | Operator image tag | `1.0.0` |
| `controller.replicas` | Number of controller replicas | `2` |
| `namespace.name` | Namespace name | `mcall-system` |
| `mcpServer.enabled` | Enable MCP server | `false` |
| `mcpServer.image.repository` | MCP server image repository | `doohee323/mcall-operator-mcp-server` |
| `mcpServer.ingress.enabled` | Enable MCP server ingress | `false` |

### MCP Server Configuration

```yaml
mcpServer:
  enabled: true
  
  image:
    repository: doohee323/mcall-operator-mcp-server
    tag: "latest"
    pullPolicy: IfNotPresent
  
  replicas: 2
  
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
```

### Logging Configuration

```yaml
logging:
  enabled: true
  backend: "postgres"  # postgres, mysql, elasticsearch, kafka
  
  postgresql:
    enabled: true
    host: "postgres.default.svc.cluster.local"
    port: 5432
    username: "admin"
    password: ""  # Set via --set or values-secrets.yaml
    database: "mcall_logs"
```

## Local Development & Testing

### Validate Chart

```bash
# Method 1: Using Makefile (recommended)
make helm-test

# Method 2: Using validation script
cd helm/mcall-operator
./quick-test.sh values-dev.yaml

# Method 3: Manual validation
helm lint ./helm/mcall-operator -f ./helm/mcall-operator/values-dev.yaml
helm template test ./helm/mcall-operator -f ./helm/mcall-operator/values-dev.yaml
```

### Simulate Jenkins Pipeline

```bash
# Quick simulation (skip Docker build)
make jenkins-sim

# Full simulation (with Docker build)
./scripts/local-jenkins-test.sh local-test dev

# Custom parameters
./scripts/local-jenkins-test.sh <build-number> <branch> <skip-docker>
```

### Test Individual Commands

```bash
# Lint
helm lint ./helm/mcall-operator -f ./helm/mcall-operator/values-dev.yaml

# Template rendering
helm template test ./helm/mcall-operator \
  -f ./helm/mcall-operator/values-dev.yaml \
  > /tmp/output.yaml

# Check specific resources
helm template test ./helm/mcall-operator \
  -f ./helm/mcall-operator/values-dev.yaml \
  -s templates/mcp-server-deployment.yaml

# Override values
helm template test ./helm/mcall-operator \
  -f ./helm/mcall-operator/values-dev.yaml \
  --set mcpServer.enabled=false \
  --set image.tag=custom-123
```

### Package Chart

```bash
make helm-package
# Output: dist/mcall-operator-*.tgz
```

## Installation Examples

### Development with MCP Server

```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-operator/values-dev.yaml \
  --set image.tag=latest \
  --set mcpServer.image.tag=dev
```

### Production without MCP Server

```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-operator/values.yaml \
  --set mcpServer.enabled=false
```

### Custom Configuration

```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --create-namespace \
  --set controller.replicas=5 \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.enabled=true \
  --set mcpServer.ingress.hosts[0].host=mcp.example.com
```

## Upgrading

```bash
# Upgrade with new image tag
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --set image.tag=v1.1.0 \
  --set mcpServer.image.tag=v1.1.0 \
  --reuse-values

# Upgrade with new values
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-dev \
  --values ./helm/mcall-operator/values-dev.yaml
```

## Uninstalling

### Automatic Cleanup

The chart includes a pre-delete hook that automatically removes finalizers:

```bash
helm uninstall mcall-operator -n mcall-dev
```

### Manual Cleanup (if needed)

```bash
# Remove finalizers from resources
kubectl get mcalltasks -n mcall-dev -o name | \
  xargs -I {} kubectl patch {} -n mcall-dev -p '{"metadata":{"finalizers":[]}}' --type=merge

kubectl get mcallworkflows -n mcall-dev -o name | \
  xargs -I {} kubectl patch {} -n mcall-dev -p '{"metadata":{"finalizers":[]}}' --type=merge

# Delete CRDs (if needed)
kubectl delete crd mcalltasks.mcall.tz.io
kubectl delete crd mcallworkflows.mcall.tz.io
```

## Monitoring

### Check Status

```bash
# Pod status
kubectl get pods -n mcall-dev -l app.kubernetes.io/name=mcall-operator

# MCP server status
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Service status
kubectl get svc -n mcall-dev

# Ingress status
kubectl get ingress -n mcall-dev
```

### View Logs

```bash
# Operator logs
kubectl logs -n mcall-dev -l app.kubernetes.io/name=mcall-operator -f

# MCP server logs
kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f
```

### Health Checks

```bash
# Port forward MCP server
kubectl port-forward -n mcall-dev svc/tz-mcall-operator-mcp-server 3000:80

# Test endpoints
curl http://localhost:3000/health
curl http://localhost:3000/ready
curl http://localhost:3000/
```

## Troubleshooting

### Common Issues

#### Namespace Ownership Conflict

**Symptom:**
```
Error: Unable to continue with install: Namespace "xxx" exists and cannot be imported
```

**Solutions:**
```bash
# Option 1: Use different namespace
helm install test ./helm/mcall-operator \
  --namespace helm-test --create-namespace

# Option 2: Use helm template (no namespace management)
helm template test ./helm/mcall-operator -f values-dev.yaml

# Option 3: Check and use existing release
helm list -n mcall-dev
helm upgrade <existing-release> ./helm/mcall-operator
```

#### Template Rendering Errors

```bash
# Debug mode
helm template test ./helm/mcall-operator \
  -f values-dev.yaml \
  --debug

# Check specific template
helm template test ./helm/mcall-operator \
  -f values-dev.yaml \
  -s templates/deployment.yaml \
  --debug
```

#### MCP Server Not Deploying

```bash
# Check if enabled
helm template test ./helm/mcall-operator \
  -f values-dev.yaml \
  | grep -c "mcp-server"

# Verify values
helm show values ./helm/mcall-operator
```

#### Image Pull Errors

```bash
# Check images exist
docker pull doohee323/tz-mcall-operator:latest
docker pull doohee323/mcall-operator-mcp-server:dev

# Check pull secrets
kubectl get secret -n mcall-dev
```

## Advanced Usage

### Custom Values File

Create a `values-custom.yaml`:

```yaml
controller:
  replicas: 3
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi

mcpServer:
  enabled: true
  replicas: 3
  ingress:
    enabled: true
    hosts:
      - host: mcp.custom.com
        paths:
          - path: /
            pathType: Prefix
```

Install:
```bash
helm install mcall-operator ./helm/mcall-operator \
  --namespace mcall-custom \
  --create-namespace \
  --values values-custom.yaml
```

### Multi-Environment Deployment

```bash
# Development
helm install mcall-dev ./helm/mcall-operator \
  -f values-dev.yaml \
  -n mcall-dev --create-namespace

# Staging
helm install mcall-staging ./helm/mcall-operator \
  -f values-staging.yaml \
  -n mcall-staging --create-namespace

# Production
helm install mcall-prod ./helm/mcall-operator \
  -f values.yaml \
  -n mcall-system --create-namespace
```

## Development Workflow

1. **Make changes** to code
2. **Test locally**:
   ```bash
   make jenkins-sim
   ```
3. **Commit and push**:
   ```bash
   git add .
   git commit -m "feat: add feature"
   git push origin dev
   ```
4. **Jenkins automatically**:
   - Builds images
   - Validates chart
   - Deploys to cluster

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test using `make jenkins-sim`
5. Submit a pull request

## License

MIT License - See parent project LICENSE file

## Links

- [Parent Project](../../README.md)
- [MCP Server Guide](../../MCP_SERVER_GUIDE.md)
- [CI/CD Documentation](../../ci/README.md)
- [Technical Documentation](../../TECHNICAL_DOCS.md)
