# MCP Server Deployment Guide

This guide explains how to deploy the tz-mcall-operator MCP server to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Helm 3.x
- tz-mcall-operator must be installed
- nginx-ingress-controller must be installed
- cert-manager (for automatic TLS certificate issuance, optional)

## Deployment Methods

### 1. Build and Push Docker Images

```bash
# Build image
docker build -t doohee323/mcall-operator-mcp-server:latest .

# Push image
docker push doohee323/mcall-operator-mcp-server:latest
```

### 2. Deploy via Helm (Recommended)

Deploy MCP server together with mcall-operator using Helm chart.

#### Production Deployment

```bash
cd ../helm/mcall-operator

helm upgrade --install mcall-operator . \
  --namespace mcall-system \
  --create-namespace \
  --set mcpServer.enabled=true \
  --set mcpServer.ingress.enabled=true \
  --set mcpServer.ingress.hosts[0].host=mcp.drillquiz.com \
  --set mcpServer.ingress.hosts[0].paths[0].path=/ \
  --set mcpServer.ingress.hosts[0].paths[0].pathType=Prefix \
  --wait
```

#### Development Deployment

```bash
cd ../helm/mcall-operator

helm upgrade --install mcall-operator . \
  --namespace mcall-dev \
  --create-namespace \
  --values values-dev.yaml \
  --wait
```

The MCP server is already enabled in values-dev.yaml and configured for `mcp-dev.drillquiz.com`.

### 3. Deploy Directly via kubectl

To deploy without Helm:

```bash
# Create RBAC
kubectl apply -f k8s/rbac.yaml

# Create Deployment
kubectl apply -f k8s/deployment.yaml

# Create Service
kubectl apply -f k8s/service.yaml

# Create Ingress (modify hostname as needed)
kubectl apply -f k8s/ingress.yaml
```

### 4. Deploy via kustomize

```bash
kubectl apply -k k8s/
```

## Deployment Verification

### 1. Check Pod Status

```bash
# Production
kubectl get pods -n mcall-system -l app.kubernetes.io/component=mcp-server

# Development
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server
```

Expected output:
```
NAME                                        READY   STATUS    RESTARTS   AGE
mcall-operator-mcp-server-xxxxxxxxx-xxxxx   1/1     Running   0          2m
mcall-operator-mcp-server-xxxxxxxxx-xxxxx   1/1     Running   0          2m
```

### 2. Check Service

```bash
kubectl get svc -n mcall-system -l app.kubernetes.io/component=mcp-server
```

### 3. Check Ingress

```bash
kubectl get ingress -n mcall-system
```

Expected output:
```
NAME                        CLASS   HOSTS                ADDRESS        PORTS     AGE
mcall-operator-mcp-server   nginx   mcp.drillquiz.com   x.x.x.x        80, 443   2m
```

### 4. Check Logs

```bash
# View real-time logs
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server -f

# View specific pod logs
kubectl logs -n mcall-system <pod-name>
```

### 5. Health Check

```bash
# Via ClusterIP Service
kubectl port-forward -n mcall-system svc/mcall-operator-mcp-server 3000:80
curl http://localhost:3000/health

# Via Ingress (after DNS configuration)
curl https://mcp.drillquiz.com/health
```

Expected response:
```json
{
  "status": "healthy"
}
```

## Service Access

### 1. Local Access via Port Forward

```bash
kubectl port-forward -n mcall-system svc/mcall-operator-mcp-server 3000:80

# In another terminal
curl http://localhost:3000/
```

### 2. External Access via Ingress

After DNS configuration is complete:

```bash
# Check server info
curl https://mcp.drillquiz.com/

# Health check
curl https://mcp.drillquiz.com/health

# Ready check
curl https://mcp.drillquiz.com/ready
```

## Configuration Customization

### Customize values.yaml

```yaml
mcpServer:
  enabled: true
  
  image:
    repository: doohee323/mcall-operator-mcp-server
    tag: "v1.0.0"  # Use specific version
    pullPolicy: IfNotPresent
  
  replicas: 3  # Increase for high availability
  
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi
  
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
      nginx.ingress.kubernetes.io/rate-limit: "100"
    hosts:
      - host: mcp.drillquiz.com
        paths:
          - path: /
            pathType: Prefix
```

### Add Environment Variables

```yaml
mcpServer:
  env:
    NODE_ENV: production
    DEFAULT_NAMESPACE: mcall-system
    LOG_LEVEL: info
    # Additional environment variables
    CUSTOM_VAR: "value"
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod -n mcall-system <pod-name>

# Check events
kubectl get events -n mcall-system --sort-by=.metadata.creationTimestamp
```

### RBAC Permission Issues

```bash
# Check ServiceAccount
kubectl get sa -n mcall-system mcall-operator-mcp-server

# Check ClusterRole
kubectl get clusterrole mcall-operator-mcp-server

# Check ClusterRoleBinding
kubectl get clusterrolebinding mcall-operator-mcp-server

# Test permissions
kubectl auth can-i create mcalltasks \
  --as=system:serviceaccount:mcall-system:mcall-operator-mcp-server
```

### Ingress Issues

```bash
# Check ingress details
kubectl describe ingress -n mcall-system mcall-operator-mcp-server

# Check nginx-ingress-controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Check DNS
nslookup mcp.drillquiz.com
```

### TLS Certificate Issues

```bash
# If using cert-manager
kubectl get certificate -n mcall-system
kubectl describe certificate -n mcall-system mcp-server-tls

# Reissue certificate
kubectl delete certificate -n mcall-system mcp-server-tls
```

## Upgrading

### Upgrade via Helm

```bash
# Upgrade to new image version
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --set mcpServer.image.tag=v1.1.0 \
  --reuse-values

# Upgrade with configuration changes
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --values custom-values.yaml
```

### Rolling Update

```bash
# Update image
kubectl set image deployment/mcall-operator-mcp-server \
  -n mcall-system \
  mcp-server=doohee323/mcall-operator-mcp-server:v1.1.0

# Check rollout status
kubectl rollout status deployment/mcall-operator-mcp-server -n mcall-system

# Rollback
kubectl rollout undo deployment/mcall-operator-mcp-server -n mcall-system
```

## Removal

### Remove via Helm

```bash
# Disable MCP server only
helm upgrade mcall-operator ./helm/mcall-operator \
  --namespace mcall-system \
  --set mcpServer.enabled=false \
  --reuse-values

# Complete removal
helm uninstall mcall-operator -n mcall-system
```

### Remove via kubectl

```bash
kubectl delete -f k8s/
```

## Monitoring

### Prometheus Metrics (Coming Soon)

```yaml
mcpServer:
  metrics:
    enabled: true
    port: 9090
    path: /metrics
```

### Log Collection

MCP server logs are sent to stdout, so they are automatically collected by your cluster's logging solution (ELK, Loki, etc.).

## Security Recommendations

1. **RBAC Least Privilege**: Grant only minimum required permissions
2. **Network Policies**: Restrict pod-to-pod communication
3. **TLS**: Require TLS on Ingress
4. **Authentication**: Consider additional authentication at Ingress level
5. **Rate Limiting**: Set rate limits via nginx annotations

## References

- [MCP Server Guide](../MCP_SERVER_GUIDE.md) - Complete guide with integration details
- [Helm Chart Guide](../helm/mcall-operator/README.md) - Helm deployment and local testing
- [tz-mcall-operator README](../README.md) - Main documentation
- [Model Context Protocol](https://modelcontextprotocol.io)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)
