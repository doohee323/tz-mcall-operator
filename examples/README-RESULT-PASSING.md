# Task Result Passing - Quick Start Guide

이 가이드는 Task 간 결과 전달 및 조건부 실행 기능을 테스트하는 방법을 설명합니다.

## 📋 개요

Health check 예제를 통해 다음 기능을 시연합니다:
1. **Task A**: `us.drillquiz.com` Health Check (GET 요청)
2. **Task B**: Health Check 성공 시 성공 로그 기록
3. **Task C**: Health Check 실패 시 실패 로그 기록

## 🚀 배포 및 테스트

### 1. CRD 업데이트

```bash
# CRD 재생성 (이미 완료됨)
cd /Users/dhong/workspaces/tz-mcall-operator
export PATH=$PATH:$(go env GOPATH)/bin
controller-gen crd:crdVersions=v1 paths="./api/..." output:crd:artifacts:config=helm/mcall-operator/templates/crds

# Kubernetes에 CRD 적용
kubectl apply -f helm/mcall-operator/templates/crds/mcalltask-crd.yaml
kubectl apply -f helm/mcall-operator/templates/crds/mcallworkflow-crd.yaml
```

### 2. Controller 재시작

```bash
# Controller 이미지 빌드 및 재배포
make docker-build
make docker-push

# 또는 Helm으로 재배포
helm upgrade --install mcall-operator ./helm/mcall-operator -n mcall-dev

# Controller Pod 재시작 대기
kubectl rollout status deployment/mcall-operator -n mcall-dev
```

### 3. 테스트 Workflow 배포

```bash
# Workflow 및 Template Tasks 배포
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

# 배포 확인
kubectl get mcalltasks -n mcall-dev
kubectl get mcallworkflows -n mcall-dev
```

### 4. 실행 결과 모니터링

```bash
# Workflow 상태 확인
kubectl get mcallworkflow health-monitor -n mcall-dev -o yaml

# Task 상태 확인
kubectl get mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Task 상세 정보 (첫 번째 실행)
kubectl describe mcalltask health-monitor-healthcheck -n mcall-dev
kubectl describe mcalltask health-monitor-log-success -n mcall-dev
kubectl describe mcalltask health-monitor-log-failure -n mcall-dev

# 로그 확인 (성공한 Task만 로그 출력됨)
kubectl logs -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# 결과 파일 확인 (Controller Pod에서)
CONTROLLER_POD=$(kubectl get pods -n mcall-dev -l app=mcall-operator -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n mcall-dev $CONTROLLER_POD -- cat /app/log/mcall/health_monitor.log
```

## 📊 예상 결과

### Health Check 성공 시

```yaml
# health-monitor-healthcheck
status:
  phase: Succeeded
  result:
    errorCode: "0"
    output: "<!doctype html>..."

# health-monitor-log-success (실행됨)
status:
  phase: Succeeded
  result:
    errorCode: "0"
    output: |
      [2025-10-10 15:30:00] ✅ SUCCESS - Phase: Succeeded, ErrorCode: 0
        Started: 2025-10-10T15:29:58Z, Completed: 2025-10-10T15:30:00Z
        us.drillquiz.com is UP
      ---
      (최근 30줄)

# health-monitor-log-failure (건너뛰어짐)
status:
  phase: Skipped
  result:
    errorCode: "0"
    errorMessage: "Skipped due to condition: when=failure"
```

### Health Check 실패 시

```yaml
# health-monitor-healthcheck
status:
  phase: Failed
  result:
    errorCode: "-1"
    errorMessage: "failed to execute GET request: ..."

# health-monitor-log-success (건너뛰어짐)
status:
  phase: Skipped

# health-monitor-log-failure (실행됨)
status:
  phase: Succeeded
  result:
    output: |
      [2025-10-10 15:30:00] ❌ FAILED - Phase: Failed, ErrorCode: -1
        Error: failed to execute GET request: timeout
        us.drillquiz.com is DOWN
```

## 🔍 디버깅

### Controller 로그 확인

```bash
# Controller 로그에서 InputSources 처리 확인
kubectl logs -n mcall-dev -l app=mcall-operator | grep -A5 "Extracted data from source task"

# 예시 출력:
# Extracted data from source task task=health-monitor-log-success sourceTask=health-monitor-healthcheck field=phase varName=HEALTH_PHASE valuePreview=Succeeded
# Extracted data from source task task=health-monitor-log-success sourceTask=health-monitor-healthcheck field=errorCode varName=ERROR_CODE valuePreview=0
```

### Task Condition 확인

```bash
# Task annotation에 저장된 condition 확인
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.metadata.annotations.mcall\.tz\.io/condition}' | jq .

# 예시 출력:
# {
#   "dependentTask": "health-monitor-healthcheck",
#   "when": "success"
# }
```

### InputSources 확인

```bash
# Task spec에 저장된 inputSources 확인
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.spec.inputSources}' | jq .

# 예시 출력:
# [
#   {
#     "name": "HEALTH_PHASE",
#     "taskRef": "health-monitor-healthcheck",
#     "field": "phase"
#   },
#   ...
# ]
```

## 🧪 추가 테스트

### 1. 수동으로 Task 결과 확인

```bash
# Health check task의 결과
kubectl get mcalltask health-monitor-healthcheck -n mcall-dev -o jsonpath='{.status.result}' | jq .

# 출력:
# {
#   "output": "<!doctype html>...",
#   "errorCode": "0",
#   "errorMessage": ""
# }
```

### 2. Workflow 재실행

```bash
# Workflow가 schedule을 가지고 있으므로 자동으로 재실행됨
# 즉시 재실행하려면 Workflow 삭제 후 재생성
kubectl delete mcallworkflow health-monitor -n mcall-dev
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

# 또는 Task만 삭제 (Workflow는 새 Task 생성)
kubectl delete mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor
```

### 3. 실패 시나리오 테스트

잘못된 URL로 Health Check를 테스트하려면:

```bash
# Template Task 수정
kubectl edit mcalltask health-check-template -n mcall-dev

# spec.input을 잘못된 URL로 변경
# input: https://invalid-domain-that-does-not-exist.com

# Workflow 재실행
kubectl delete mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor
```

## 📝 정리

```bash
# 테스트 리소스 삭제
kubectl delete mcallworkflow health-monitor -n mcall-dev
kubectl delete mcalltasks -n mcall-dev health-check-template log-success-template log-failure-template

# 로그 파일 삭제 (Controller Pod에서)
CONTROLLER_POD=$(kubectl get pods -n mcall-dev -l app=mcall-operator -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n mcall-dev $CONTROLLER_POD -- rm -f /app/log/mcall/health_monitor.log
```

## 🎯 다음 단계

1. **MCP Tools 추가**: `get_task_result_schema`, `get_task_result_json`
2. **복잡한 JSONPath 테스트**: API JSON 응답에서 특정 필드 추출
3. **여러 InputSources**: 여러 Task의 결과를 하나의 Task에서 사용
4. **복합 조건**: AND, OR 조건 테스트

## 🐛 문제 해결

### Task가 계속 Pending 상태

```bash
# 의존 Task가 완료되지 않았을 수 있음
kubectl get mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Controller 로그 확인
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Waiting for"
```

### InputTemplate이 적용되지 않음

```bash
# Task spec 확인
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.spec.inputTemplate}'

# Controller 로그에서 렌더링 확인
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Rendered input template"
```

### Condition이 작동하지 않음

```bash
# Annotation 확인
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.metadata.annotations}'

# Controller 로그에서 조건 체크 확인
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Task condition"
```

