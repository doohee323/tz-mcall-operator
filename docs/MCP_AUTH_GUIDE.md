# MCP Server API Key Authentication Guide

## 개요

MCP Server는 API Key 기반 인증을 지원하여 보안을 강화할 수 있습니다.

## 인증 방식

### 1. **X-API-Key Header** (권장)

```bash
curl -H "X-API-Key: your-api-key-here" https://mcp-dev.drillquiz.com/api/namespaces
```

### 2. **Authorization Bearer Token**

```bash
curl -H "Authorization: Bearer your-api-key-here" https://mcp-dev.drillquiz.com/api/namespaces
```

### 3. **Query Parameter** (테스트용)

```bash
curl "https://mcp-dev.drillquiz.com/api/namespaces?apiKey=your-api-key-here"
```

## 로컬 개발 환경

### 인증 비활성화 (기본값)

```bash
cd mcp-server
npm start
```

### 인증 활성화

```bash
cd mcp-server
MCP_REQUIRE_AUTH=true MCP_API_KEYS=test-key-123,admin-key-456 npm start
```

## Claude Desktop 설정

### Stdio Mode (로컬, 인증 불필요)

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

## Cursor MCP 설정

### 원격 서버 (SSE, 인증 필요)

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

또는 URL에 포함:

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

## Kubernetes 배포

### 1. Helm Values 설정

```yaml
# values-dev.yaml
mcpServer:
  enabled: true
  auth:
    enabled: true
    apiKeys: "dev-key-12345,staging-key-67890"
```

### 2. Secret 생성 (Production)

```bash
# API Keys를 Kubernetes Secret으로 생성
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="prod-key-abc123,admin-key-xyz789" \
  -n mcall-system

# Helm 배포
helm upgrade --install mcall-operator ./helm/mcall-operator \
  -f helm/mcall-operator/values.yaml \
  --namespace mcall-system
```

### 3. 배포 확인

```bash
# MCP Server Pod 확인
kubectl get pods -n mcall-system -l app.kubernetes.io/component=mcp-server

# 로그 확인
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server
# 출력: 🔐 API Key authentication enabled with 2 key(s)

# API 테스트
curl -H "X-API-Key: prod-key-abc123" https://mcp-dev.drillquiz.com/api/namespaces
```

## 환경변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `MCP_REQUIRE_AUTH` | 인증 활성화 여부 | `false` |
| `MCP_API_KEYS` | API Keys (쉼표 구분) | - |

## 보안 권장사항

### ✅ DO (권장)

1. **Production에서는 반드시 인증 활성화**
   ```yaml
   mcpServer:
     auth:
       enabled: true
   ```

2. **API Keys를 Kubernetes Secret으로 관리**
   ```bash
   kubectl create secret generic mcp-api-keys --from-literal=api-keys="..."
   ```

3. **헤더 방식 사용 (X-API-Key 또는 Bearer)**
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

### 401 Unauthorized 오류

```bash
# 로그 확인
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server | grep "Unauthorized"

# API Key 확인
kubectl get secret mcp-api-keys -n mcall-system -o jsonpath='{.data.api-keys}' | base64 -d

# 테스트
curl -v -H "X-API-Key: your-key" https://mcp-dev.drillquiz.com/api/namespaces
```

### 인증 설정이 적용되지 않음

```bash
# Pod 재시작
kubectl rollout restart deployment/mcall-operator-mcp-server -n mcall-system

# 환경변수 확인
kubectl exec -n mcall-system deployment/mcall-operator-mcp-server -- env | grep MCP
```

## Rate Limiting (선택사항)

Rate limiting을 활성화하려면 `http-server.ts`에 다음을 추가:

```typescript
// Rate limiting: 100 requests per minute
app.use('/mcp', authService.rateLimit(100, 60000));
app.use('/api', authService.rateLimit(1000, 60000));
```

## 참고 자료

- [MCP Server Guide](../MCP_SERVER_GUIDE.md)
- [Kubernetes Secret 관리](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Helm Values 설정](../helm/mcall-operator/README.md)

