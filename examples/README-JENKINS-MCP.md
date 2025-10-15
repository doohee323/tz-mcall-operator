# Jenkins MCP Client Integration

이 문서는 `tz-mcall-operator`의 `mcp-client` 타입을 사용하여 Jenkins MCP Server와 통합하는 방법을 설명합니다.

## 개요

Jenkins MCP Server 플러그인은 Model Context Protocol을 통해 Jenkins를 AI 에이전트 및 다른 시스템과 통합할 수 있게 해줍니다.

### 지원하는 Jenkins MCP Tools

| Tool | 설명 | 주요 파라미터 |
|------|------|---------------|
| `getJobs` | Job 목록 조회 (페이지네이션) | `offset`, `limit` |
| `getJob` | 특정 Job 상세 정보 | `jobFullName` |
| `getBuild` | 특정 빌드 정보 | `jobFullName`, `buildNumber` (optional) |
| `getBuildLog` | 빌드 로그 조회 | `jobFullName`, `buildNumber`, `offset`, `limit` |
| `triggerBuild` | 빌드 실행 | `jobFullName`, `parameters` (optional) |
| `updateBuild` | 빌드 정보 업데이트 | `jobFullName`, `buildNumber`, `displayName`, `description` |
| `getJobScm` | Job의 SCM 설정 | `jobFullName` |
| `getBuildScm` | 빌드의 SCM 정보 | `jobFullName`, `buildNumber` |
| `getBuildChangeSets` | 변경 세트 조회 | `jobFullName`, `buildNumber` |
| `getStatus` | Jenkins 상태 확인 | (파라미터 없음) |
| `whoAmI` | 현재 사용자 정보 | (파라미터 없음) |

## 사전 준비

### 1. Jenkins API Token 생성

1. Jenkins에 로그인
2. 우측 상단 사용자 아이콘 클릭 → Security
3. "Add new token" 클릭
4. 토큰 이름 입력 (예: `mcp-operator-token`)
5. "Generate" 클릭
6. 생성된 토큰 복사 (다시 볼 수 없음!)

### 2. Kubernetes Secret 생성

```bash
kubectl create secret generic jenkins-mcp-credentials \
  --from-literal=username=admin \
  --from-literal=token=YOUR_API_TOKEN_HERE \
  -n default
```

## 사용 예제

### 예제 1: Job 목록 조회

가장 간단한 사용 사례:

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-list-jobs
  namespace: default
spec:
  type: mcp-client
  schedule: "*/10 * * * *"
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "getJobs"
    arguments:
      offset: 0
      limit: 5
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

**적용:**
```bash
kubectl apply -f examples/jenkins-mcp-list-jobs.yaml
```

**결과 확인:**
```bash
kubectl get mcalltask jenkins-list-jobs -o yaml | grep -A 50 "status:"
```

### 예제 2: 특정 Job 히스토리 조회

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-get-crd-history
  namespace: default
spec:
  type: mcp-client
  schedule: "*/15 * * * *"
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "getJob"
    arguments:
      jobFullName: "tz-drillquiz-crd"
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

### 예제 3: 빌드 실행 (파라미터 없음)

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-trigger-build
  namespace: default
spec:
  type: mcp-client
  schedule: "0 2 * * *"  # 매일 오전 2시
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "triggerBuild"
    arguments:
      jobFullName: "tz-drillquiz-crd"
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

### 예제 4: 파라미터화된 빌드 실행

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-trigger-param-build
  namespace: default
spec:
  type: mcp-client
  schedule: "0 3 * * *"
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "triggerBuild"
    arguments:
      jobFullName: "tz-drillquiz-usecase"
      parameters:
        USECASE_TARGET_DOMAIN: "drillquiz-usecase.devops-qa.svc.cluster.local"
        USECASE_TARGET_PORT: "80"
        USECASE_TEST_TYPE: "k8s-qa"
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

### 예제 5: 빌드 로그 모니터링

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-monitor-build-log
  namespace: default
spec:
  type: mcp-client
  schedule: "*/5 * * * *"  # 5분마다
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "getBuildLog"
    arguments:
      jobFullName: "tz-drillquiz-crd"
      offset: 0
      limit: 50  # 최근 50 라인
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

### 예제 6: Jenkins 헬스 체크

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: jenkins-health-check
  namespace: default
spec:
  type: mcp-client
  schedule: "*/5 * * * *"
  input: ""
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "getStatus"
    arguments: {}
    auth:
      type: "basic"
      secretRef:
        name: jenkins-mcp-credentials
        namespace: default
      usernameKey: "username"
      passwordKey: "token"
    headers:
      Accept: "text/event-stream, application/json"
    connectionTimeout: 30
```

## MCP 프로토콜 동작 방식

### 1. Initialize Handshake
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "tz-mcall-operator",
      "version": "1.0.0"
    }
  }
}
```

**Response Header:**
- `mcp-session-id`: 세션 ID (이후 요청에 필요)

### 2. Notifications (Optional)
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

### 3. Tool Call
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "getJobs",
    "arguments": {
      "offset": 0,
      "limit": 10
    }
  }
}
```

**Request Headers:**
- `mcp-session-id`: 세션 ID (Step 1에서 받은 값)
- `Authorization`: Basic 인증
- `Accept`: `text/event-stream, application/json`

## 구현 세부사항

### Controller 구현

`tz-mcall-operator` controller는 자동으로 다음을 처리합니다:

1. ✅ **세션 관리**: Initialize에서 세션 ID 추출
2. ✅ **헤더 관리**: 자동으로 필요한 헤더 추가
3. ✅ **인증**: Kubernetes Secret에서 안전하게 로드
4. ✅ **응답 파싱**: MCP JSON-RPC 응답 처리
5. ✅ **에러 처리**: HTTP 및 MCP 에러 핸들링
6. ✅ **대용량 응답 처리**: 10KB 초과 시 자동 truncate

### 응답 크기 관리

각 text content가 10KB를 초과하면 자동으로 truncate됩니다:

```
Original: 50KB
Result: 10KB + "... [truncated, original length: 50000 bytes]"
```

## 실전 사용 사례

### Use Case 1: 매일 밤 자동 빌드

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: nightly-build-crd
spec:
  type: mcp-client
  schedule: "0 2 * * *"  # 매일 오전 2시
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "triggerBuild"
    arguments:
      jobFullName: "tz-drillquiz-crd"
    # ... auth 설정
```

### Use Case 2: 빌드 실패 모니터링

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: monitor-build-failures
spec:
  type: mcp-client
  schedule: "*/10 * * * *"  # 10분마다
  mcpConfig:
    serverUrl: "https://jenkins.drillquiz.com/mcp-server/mcp"
    toolName: "getBuild"
    arguments:
      jobFullName: "tz-drillquiz-crd"
      # Last build 확인
    # ... auth 설정
```

결과를 `inputSources`로 다른 task에 전달하여 실패 시 알림 발송 가능!

### Use Case 3: Multi-environment 배포

```yaml
# QA 환경 빌드
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: deploy-to-qa
spec:
  type: mcp-client
  mcpConfig:
    toolName: "triggerBuild"
    arguments:
      jobFullName: "tz-drillquiz"
      parameters:
        ENV: "qa"
        VERSION: "v1.2.3"
---
# Production 빌드 (QA 성공 시)
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: deploy-to-prod
spec:
  type: mcp-client
  inputSources:
    - taskRef:
        name: deploy-to-qa
      jsonPath: "$.result.result"  # QA 빌드 결과 확인
  mcpConfig:
    toolName: "triggerBuild"
    arguments:
      jobFullName: "tz-drillquiz"
      parameters:
        ENV: "production"
        VERSION: "v1.2.3"
```

## 테스트 방법

### 1. 전체 예제 적용

```bash
kubectl apply -f examples/jenkins-mcp-complete-examples.yaml
```

### 2. Task 목록 확인

```bash
kubectl get mcalltask | grep jenkins
```

### 3. 특정 Task 상태 확인

```bash
kubectl get mcalltask jenkins-list-jobs -o yaml
```

### 4. 실행 결과 조회

```bash
# 성공한 결과 보기
kubectl get mcalltask jenkins-list-jobs -o jsonpath='{.status.result.output}' | jq '.'

# 에러 메시지 보기 (실패 시)
kubectl get mcalltask jenkins-list-jobs -o jsonpath='{.status.result.errorMessage}'
```

### 5. 로그 확인

```bash
kubectl logs -n mcall-operator-system deployment/mcall-operator-controller-manager | grep jenkins-list-jobs
```

## 문제 해결

### 문제 1: "Session ID required" 에러

**원인**: MCP 초기화가 실패했거나 세션 ID가 누락됨

**해결**: Controller 로그 확인
```bash
kubectl logs -n mcall-operator-system deployment/mcall-operator-controller-manager | grep "Session ID"
```

### 문제 2: "HTTP 403 Forbidden"

**원인**: API 토큰이 잘못되었거나 만료됨

**해결**: 새 API 토큰 생성 후 Secret 업데이트
```bash
kubectl delete secret jenkins-mcp-credentials
kubectl create secret generic jenkins-mcp-credentials \
  --from-literal=username=admin \
  --from-literal=token=NEW_TOKEN
```

### 문제 3: "text/event-stream required"

**원인**: Accept 헤더 누락

**해결**: YAML에 headers 추가 (이미 포함됨)
```yaml
headers:
  Accept: "text/event-stream, application/json"
```

### 문제 4: 응답이 너무 큼

**원인**: Jenkins MCP가 전체 히스토리 반환

**해결 1**: limit 파라미터 축소
```yaml
arguments:
  limit: 2  # 적은 수로
```

**해결 2**: 특정 job만 조회
```yaml
toolName: "getJob"
arguments:
  jobFullName: "specific-job-name"
```

## 고급 사용법

### Workflow와 통합

```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: jenkins-deploy-pipeline
spec:
  schedule: "0 1 * * *"  # 매일 오전 1시
  tasks:
    # 1. QA 빌드 트리거
    - name: trigger-qa-build
      type: mcp-client
      mcpConfig:
        toolName: "triggerBuild"
        arguments:
          jobFullName: "tz-drillquiz"
          parameters:
            ENV: "qa"
    
    # 2. 빌드 완료 대기 (polling)
    - name: check-qa-build-status
      type: mcp-client
      dependencies: ["trigger-qa-build"]
      retryPolicy:
        maxRetries: 10
        retryInterval: "1m"
      mcpConfig:
        toolName: "getBuild"
        arguments:
          jobFullName: "tz-drillquiz"
    
    # 3. QA 성공 시 Production 빌드
    - name: trigger-prod-build
      type: mcp-client
      dependencies: ["check-qa-build-status"]
      inputSources:
        - taskRef:
            name: check-qa-build-status
          jsonPath: "$.result"
      mcpConfig:
        toolName: "triggerBuild"
        arguments:
          jobFullName: "tz-drillquiz"
          parameters:
            ENV: "production"
```

## 참고 자료

- [Jenkins MCP Server Plugin](https://plugins.jenkins.io/mcp-server/)
- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [tz-mcall-operator MCP Client 문서](README-MCP-CLIENT.md)

## 버전 정보

- **Jenkins**: 2.516.3+
- **MCP Server Plugin**: 0.95+
- **MCP Protocol**: 2024-11-05
- **tz-mcall-operator**: v1.0+ (mcp branch)







