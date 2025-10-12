# MCP Server API Key Authentication Guide

## Overview

MCP Server supports API Key-based authentication to enhance security.

## Authentication Methods

### 1. **X-API-Key Header** (Recommended)

```bash
curl -H "X-API-Key: your-api-key-here" https://mcp-dev.drillquiz.com/api/namespaces
```

### 2. **Authorization Bearer Token**

```bash
curl -H "Authorization: Bearer your-api-key-here" https://mcp-dev.drillquiz.com/api/namespaces
```

### 3. **Query Parameter** (Testing only)

```bash
curl "https://mcp-dev.drillquiz.com/api/namespaces?apiKey=your-api-key-here"
```

## Local Development Environment

### Disable Authentication (Default)

```bash
cd mcp-server
npm start
```

### Enable Authentication

```bash
cd mcp-server
MCP_REQUIRE_AUTH=true MCP_API_KEYS=test-key-123,admin-key-456 npm start
```

## Claude Desktop Configuration

### Stdio Mode (Local, no authentication required)

```json
{
  "mcpServers": {
    "mcall-operator": {
      "command": "node",
      "args": ["/Users/dhong/workspaces/tz-mcall-operator/mcp-server/dist/index.js"],
      "env": {
        "SERVER_MODE": "stdio",
        "DEFAULT_NAMESPACE": "mcall-dev"
      }
    }
  }
}
```

## Cursor MCP Configuration

### Remote Server (SSE, authentication required)

```json
{
  "mcpServers": {
    "mcall-operator": {
      "url": "https://mcp-dev.drillquiz.com/mcp",
      "transport": "sse",
      "headers": {
        "X-API-Key": "your-api-key-here"
      }
    }
  }
}
```

Or include in URL:

```json
{
  "mcpServers": {
    "mcall-operator": {
      "url": "https://mcp-dev.drillquiz.com/mcp?apiKey=your-api-key-here",
      "transport": "sse"
    }
  }
}
```

## Kubernetes Deployment

### Option 1: Production (Recommended) - Use Existing Secret

**Step 1: Create Secret (Before Helm deployment)**

```bash
# Create API Keys as Kubernetes Secret first
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="prod-key-abc123,admin-key-xyz789" \
  -n mcall-system
```

**Step 2: Helm Deployment**

```yaml
# values-prod.yaml
mcpServer:
  auth:
    enabled: true
    existingSecret: "mcp-api-keys"  # Use existing Secret
    apiKeys: ""  # Leave empty (secure)
```

```bash
# Deploy with Helm
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f helm/mcall-operator/values-prod.yaml \
  --namespace mcall-system
```

### Option 2: Development - Helm Manages Secret

**Use in development only (Do NOT commit to Git!)**

```bash
# Pass API Key via command line
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f helm/mcall-operator/values-dev.yaml \
  --set mcpServer.auth.enabled=true \
  --set mcpServer.auth.apiKeys="dev-key-12345,test-key-67890" \
  --namespace mcall-dev
```

Or in a separate file:

```yaml
# values-dev-local.yaml (Must be in .gitignore!)
mcpServer:
  auth:
    enabled: true
    apiKeys: "dev-key-12345,test-key-67890"
```

```bash
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f helm/mcall-operator/values-dev.yaml \
  -f helm/mcall-operator/values-dev-local.yaml \
  --namespace mcall-dev
```

### Option 3: API Key Rotation (Zero Downtime)

```bash
# 1. Add new key (keep existing key)
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="old-key,new-key" \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Restart pods (load new Secret)
kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system

# 3. Switch clients to new-key

# 4. Remove old-key
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="new-key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system
```

### Verify Deployment

```bash
# Check MCP Server pods
kubectl get pods -n mcall-system -l app.kubernetes.io/component=mcp-server

# Check logs
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server
# Output: 🔐 API Key authentication enabled with 2 key(s)

# Test API
curl -H "X-API-Key: prod-key-abc123" https://mcp-dev.drillquiz.com/api/namespaces
```

## Environment Variables

| Variable | Description | Default |
|------|------|--------|
| `MCP_REQUIRE_AUTH` | Enable authentication | `false` |
| `MCP_API_KEYS` | API Keys (comma-separated) | - |

## Security Best Practices

### ✅ DO (Recommended)

1. **Always enable authentication in Production**
   ```yaml
   mcpServer:
     auth:
       enabled: true
   ```

2. **Manage API Keys as Kubernetes Secrets**
   ```bash
   kubectl create secret generic mcp-api-keys --from-literal=api-keys="..."
   ```

3. **Use header method (X-API-Key or Bearer)**
   ```bash
   curl -H "X-API-Key: your-key" https://...
   ```

4. **정기적으로 API Key 교체**
   ```bash
   # 새로운 키 생성
   kubectl create secret generic mcp-api-keys \
     --from-literal=api-keys="new-key-1,new-key-2" \
     --dry-run=client -o yaml | kubectl apply -f -
   
   # Pod 재시작
   kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system
   ```

### ❌ DON'T (금지)

1. **Query Parameter 방식 사용 금지 (로그에 노출)**
   ```bash
   # ❌ 금지: 로그에 API Key가 기록됨
   curl "https://...?apiKey=secret-key"
   ```

2. **API Keys를 코드나 values.yaml에 하드코딩 금지**
   ```yaml
   # ❌ 금지
   apiKeys: "my-secret-key-12345"
   ```

3. **Public 환경에서 인증 비활성화 금지**
   ```yaml
   # ❌ 금지 (외부 노출 시)
   auth:
     enabled: false
   ```

## 트러블슈팅

### 401 Unauthorized Error

```bash
# Check logs
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server | grep "Unauthorized"

# Check API Key
kubectl get secret mcp-api-keys -n mcall-system -o jsonpath='{.data.api-keys}' | base64 -d

# Test
curl -v -H "X-API-Key: your-key" https://mcp-dev.drillquiz.com/api/namespaces
```

### Authentication Settings Not Applied

```bash
# Restart pods
kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system

# Check environment variables
kubectl exec -n mcall-system deployment/mcall-operator-mcp-server -- env | grep MCP
```

## Rate Limiting (Optional)

To enable rate limiting, add the following to `http-server.ts`:

```typescript
// Rate limiting: 100 requests per minute
app.use('/mcp', authService.rateLimit(100, 60000));
app.use('/api', authService.rateLimit(1000, 60000));
```

## References

- [MCP Server Guide](../MCP_SERVER_GUIDE.md)
- [Kubernetes Secret Management](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Helm Values Configuration](../helm/mcall-operator/README.md)

