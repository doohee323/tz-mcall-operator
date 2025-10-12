# MCP Server API Key Authentication - Helm Chart Guide

## Overview

This guide covers API Key authentication configuration for MCP Server when deploying with Helm.

**For complete authentication documentation, see**: [docs/MCP_AUTH_GUIDE.md](../../docs/MCP_AUTH_GUIDE.md)

## Quick Setup

### Production (Recommended)

**Step 1: Create Secret (before Helm deployment)**

```bash
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="prod-key-abc123,admin-key-xyz789" \
  -n mcall-system
```

**Step 2: Configure Helm values**

```yaml
# values-prod.yaml
mcpServer:
  auth:
    enabled: true
    existingSecret: "mcp-api-keys"  # Use existing Secret
    apiKeys: ""                     # Leave empty (secure)
```

**Step 3: Deploy**

```bash
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f ./helm/mcall-operator/values-prod.yaml \
  --namespace mcall-system \
  --create-namespace
```

### Development

For development environments, you can pass API keys via command line:

```bash
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f ./helm/mcall-operator/values-dev.yaml \
  --set mcpServer.auth.enabled=true \
  --set mcpServer.auth.apiKeys="dev-key-123,test-key-456" \
  --namespace mcall-dev
```

**Note**: Do NOT commit API keys to Git!

## Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mcpServer.auth.enabled` | Enable API key authentication | `false` |
| `mcpServer.auth.existingSecret` | Name of existing Secret containing API keys | `""` |
| `mcpServer.auth.apiKeys` | Comma-separated API keys (dev only) | `""` |

## Verification

```bash
# Check MCP Server pods
kubectl get pods -n mcall-system -l app.kubernetes.io/component=mcp-server

# Check logs (should show authentication enabled)
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server

# Test API
curl -H "X-API-Key: your-key" https://mcp.drillquiz.com/api/namespaces
```

## Best Practices

✅ **DO**:
- Use `existingSecret` in production
- Create Secrets before Helm deployment
- Rotate keys regularly (every 90 days recommended)
- Use header-based authentication (`X-API-Key`)

❌ **DON'T**:
- Commit API keys to Git
- Use query parameters (logged)
- Hardcode keys in values.yaml for production
- Disable authentication in public environments

## API Key Rotation

```bash
# 1. Add new key (keep existing key)
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="old-key,new-key" \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Restart pods
kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system

# 3. Update clients to use new-key

# 4. Remove old-key
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="new-key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system
```

## Troubleshooting

### 401 Unauthorized

```bash
# Check Secret exists
kubectl get secret mcp-api-keys -n mcall-system

# Verify Secret content
kubectl get secret mcp-api-keys -n mcall-system -o jsonpath='{.data.api-keys}' | base64 -d
```

### Authentication Not Working

```bash
# Restart deployment
kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system

# Check environment variables
kubectl exec -n mcall-system deployment/mcall-operator-mcp-server -- env | grep MCP
```

## Complete Documentation

For detailed information including:
- Claude Desktop configuration
- Cursor MCP configuration
- Advanced authentication scenarios
- Security best practices
- Rate limiting

See **[docs/MCP_AUTH_GUIDE.md](../../docs/MCP_AUTH_GUIDE.md)**

## References

- [MCP Server Guide](../../MCP_SERVER_GUIDE.md)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Helm Values](./values.yaml)
