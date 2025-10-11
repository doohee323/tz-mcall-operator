# MCP Server API Key Authentication - Helm Chart Guide

## 배포 시나리오별 가이드

### 🔐 Scenario 1: Production (권장 방식)

**특징**:
- ✅ API Keys가 Git에 저장되지 않음
- ✅ Kubernetes Secret으로 안전하게 관리
- ✅ Helm과 독립적으로 Secret 관리

**Step 1: Secret 생성 (Helm 배포 전)**

```bash
# API Keys를 Kubernetes Secret으로 생성
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="prod-key-abc123,admin-key-xyz789" \
  -n mcall-system
```

**Step 2: Helm 배포**

```bash
helm upgrade --install mcall-operator . \
  -f values-prod.yaml \
  --namespace mcall-system \
  --create-namespace
```

**values-prod.yaml** 내용:
```yaml
mcpServer:
  enabled: true
  auth:
    enabled: true
    existingSecret: "mcp-api-keys"  # 기존 Secret 참조
    apiKeys: ""                     # 비워둠!
```

### 🧪 Scenario 2: Development (로컬/테스트)

**특징**:
- 인증 비활성화 (빠른 개발)
- 또는 간단한 테스트 키 사용

#### Option A: 인증 비활성화

```bash
helm upgrade --install mcall-operator . \
  -f values-dev.yaml \
  --namespace mcall-dev \
  --create-namespace
```

**values-dev.yaml** 내용:
```yaml
mcpServer:
  enabled: true
  auth:
    enabled: false  # 인증 끔
```

#### Option B: 테스트 키 사용 (명령줄)

```bash
helm upgrade --install mcall-operator . \
  -f values-dev.yaml \
  --set mcpServer.auth.enabled=true \
  --set mcpServer.auth.apiKeys="test-key-123,dev-key-456" \
  --namespace mcall-dev \
  --create-namespace
```

#### Option C: 로컬 values 파일 (Git ignore)

```bash
# values-dev-local.yaml 생성 (Git에 커밋하지 말 것!)
cat > values-dev-local.yaml <<EOF
mcpServer:
  auth:
    enabled: true
    apiKeys: "my-local-test-key-12345"
EOF

# 배포
helm upgrade --install mcall-operator . \
  -f values-dev.yaml \
  -f values-dev-local.yaml \
  --namespace mcall-dev
```

### 🔄 Scenario 3: API Key 교체 (Zero Downtime)

```bash
# 1. 새 키 추가 (기존 키 유지)
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="old-key-abc,new-key-xyz" \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Pod 재시작 (Rolling update)
kubectl rollout restart deployment -n mcall-system \
  -l app.kubernetes.io/component=mcp-server

# 3. 클라이언트를 new-key로 전환 (시간을 두고 진행)

# 4. 확인 후 old-key 제거
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys="new-key-xyz" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment -n mcall-system \
  -l app.kubernetes.io/component=mcp-server
```

### 🔍 Scenario 4: Staging (Existing Secret + Git 관리)

**특징**:
- Production과 동일한 방식
- 별도의 Secret 사용

```bash
# Secret 생성
kubectl create secret generic mcp-api-keys-staging \
  --from-literal=api-keys="staging-key-1,staging-key-2" \
  -n mcall-staging

# 배포
helm upgrade --install mcall-operator . \
  -f values-staging.yaml \
  --namespace mcall-staging
```

**values-staging.yaml** 내용:
```yaml
mcpServer:
  auth:
    enabled: true
    existingSecret: "mcp-api-keys-staging"
    apiKeys: ""
```

## 📝 Values 파일 구조

```
helm/mcall-operator/
├── values.yaml              # 기본값 (Git ✅)
├── values-dev.yaml          # 개발 환경 (Git ✅)
├── values-staging.yaml      # Staging 환경 (Git ✅)
├── values-prod.yaml         # Production 환경 (Git ✅)
└── values-*-local.yaml      # 로컬 테스트 (Git ❌)
```

## 🛡️ 보안 체크리스트

### ✅ DO (권장)

- [x] Production에서 `existingSecret` 사용
- [x] API Keys를 Kubernetes Secret으로 관리
- [x] `values-*-local.yaml` 파일을 .gitignore에 추가
- [x] Production에서 `auth.enabled: true` 설정
- [x] 정기적으로 API Key 교체 (90일 권장)
- [x] 최소 권한 원칙: 환경별로 다른 키 사용

### ❌ DON'T (금지)

- [ ] API Keys를 Git에 커밋
- [ ] values.yaml에 Production 키 하드코딩
- [ ] Public 환경에서 인증 비활성화
- [ ] 모든 환경에서 동일한 키 사용
- [ ] Query Parameter로 API Key 전송 (로그 노출 위험)

## 🧪 테스트

### Secret 확인

```bash
# Secret 존재 여부 확인
kubectl get secret mcp-api-keys -n mcall-system

# Secret 내용 확인 (Base64 디코드)
kubectl get secret mcp-api-keys -n mcall-system -o jsonpath='{.data.api-keys}' | base64 -d
```

### 인증 테스트

```bash
# 1. Pod 확인
kubectl get pods -n mcall-system -l app.kubernetes.io/component=mcp-server

# 2. 로그 확인
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server | grep "authentication"
# 출력: 🔐 API Key authentication enabled with N key(s)

# 3. API 테스트 (인증 없음 - 실패)
curl https://mcp.drillquiz.com/api/namespaces
# 출력: {"error":"Unauthorized",...}

# 4. API 테스트 (인증 포함 - 성공)
curl -H "X-API-Key: your-prod-key" https://mcp.drillquiz.com/api/namespaces
# 출력: {"success":true,"namespaces":[...]}
```

## 🔧 트러블슈팅

### Pod가 시작하지 않음

```bash
# 1. Pod 상태 확인
kubectl describe pod -n mcall-system -l app.kubernetes.io/component=mcp-server

# 2. Secret 확인
kubectl get secret mcp-api-keys -n mcall-system

# 3. 환경변수 확인
kubectl exec -n mcall-system deployment/mcall-operator-mcp-server -- env | grep MCP
```

### 인증 실패 (401 Unauthorized)

```bash
# 1. Secret 내용 확인
kubectl get secret mcp-api-keys -n mcall-system -o jsonpath='{.data.api-keys}' | base64 -d

# 2. 전송한 API Key 확인
curl -v -H "X-API-Key: your-key" https://...
# 헤더에 X-API-Key가 포함되었는지 확인

# 3. Pod 로그 확인
kubectl logs -n mcall-system -l app.kubernetes.io/component=mcp-server | grep "Unauthorized"
```

## 📚 참고 자료

- [MCP Server API Key Authentication Guide](../../docs/MCP_AUTH_GUIDE.md)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Helm Values Files](https://helm.sh/docs/chart_template_guide/values_files/)

