# Deployment Verification Guide

## 🚀 Jenkins 배포 후 검증 절차

### 1. Operator 배포 확인

```bash
# Operator Pod 상태 확인
kubectl get pods -n mcall-dev -l app=tz-mcall-operator-dev

# Operator 로그 확인 (InputSources 처리 로그 찾기)
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=50 | grep -i "input"
```

**예상 출력**:
```
tz-mcall-operator-dev-xxxxx   1/1   Running   0   2m
```

### 2. CRD 업데이트 확인

```bash
# CRD에 inputSources 필드가 있는지 확인
kubectl get crd mcalltasks.mcall.tz.io -o yaml | grep -A 10 "inputSources"
```

**예상 출력**: inputSources, inputTemplate 스키마 정의가 표시되어야 함

### 3. Task Result Passing 테스트

#### Step 1: 소스 Task 생성
```bash
kubectl apply -f - <<'EOF'
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: test-source-task
  namespace: mcall-dev
spec:
  type: cmd
  input: "echo 'SUCCESS_FROM_SOURCE'"
  timeout: 10
EOF
```

#### Step 2: 완료 대기 (약 5초)
```bash
kubectl wait --for=jsonpath='{.status.phase}'=Succeeded --timeout=30s mcalltask/test-source-task -n mcall-dev
```

#### Step 3: Consumer Task 생성
```bash
kubectl apply -f - <<'EOF'
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: test-consumer-task
  namespace: mcall-dev
spec:
  type: cmd
  input: "will be overridden"
  timeout: 10
  dependencies:
    - test-source-task
  inputSources:
    - name: SOURCE_OUTPUT
      taskRef: test-source-task
      field: output
    - name: SOURCE_PHASE
      taskRef: test-source-task
      field: phase
    - name: SOURCE_ERROR_CODE
      taskRef: test-source-task
      field: errorCode
  inputTemplate: |
    echo "=== Task Result Passing Test ==="
    echo "Source output: ${SOURCE_OUTPUT}"
    echo "Source phase: ${SOURCE_PHASE}"
    echo "Source error code: ${SOURCE_ERROR_CODE}"
    echo "=============================="
EOF
```

#### Step 4: 결과 확인
```bash
# Consumer task 완료 대기
kubectl wait --for=jsonpath='{.status.phase}'=Succeeded --timeout=30s mcalltask/test-consumer-task -n mcall-dev

# 결과 확인
kubectl get mcalltask test-consumer-task -n mcall-dev -o jsonpath='{.status.result.output}'
```

**예상 출력**:
```
=== Task Result Passing Test ===
Source output: SUCCESS_FROM_SOURCE
Source phase: Succeeded
Source error code: 0
=============================
```

### 4. Conditional Execution 테스트

```bash
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml
```

**확인사항**:
- healthcheck task 성공 시 → log-success task만 실행
- healthcheck task 실패 시 → log-failure task만 실행

```bash
# Workflow 상태 확인
kubectl get mcallworkflow health-monitor -n mcall-dev

# Task 목록 확인
kubectl get mcalltask -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# 특정 task 상태 확인
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o yaml
kubectl get mcalltask health-monitor-log-failure -n mcall-dev -o yaml
```

### 5. Operator 로그에서 검증

```bash
# InputSources 처리 로그 확인
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=100 | grep -A 5 "Injected data from input sources"

# 조건부 실행 로그 확인
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=100 | grep -A 5 "Task condition"
```

**예상 로그**:
```
Injected data from input sources
  task: test-consumer-task
  sourceCount: 3
  envVars: 3
```

### 6. 정리

```bash
# 테스트 리소스 삭제
kubectl delete mcalltask test-source-task test-consumer-task -n mcall-dev
kubectl delete mcallworkflow health-monitor -n mcall-dev
```

## ✅ 성공 기준

- [ ] Operator pod가 Running 상태
- [ ] CRD에 inputSources, inputTemplate 필드 존재
- [ ] Test consumer task가 Succeeded 상태
- [ ] Consumer task의 output에 소스 task의 데이터가 포함됨
- [ ] Conditional workflow에서 조건에 맞는 task만 실행됨
- [ ] Operator 로그에 "Injected data from input sources" 메시지 존재

## 🐛 트러블슈팅

### Consumer Task가 Failed 상태인 경우

1. **Spec에 inputSources가 없는 경우**
   ```bash
   kubectl get mcalltask test-consumer-task -n mcall-dev -o yaml | grep -A 10 "spec:"
   ```
   → CRD가 제대로 업데이트되지 않았을 수 있음

2. **"will be overridden" 명령어가 실행된 경우**
   ```bash
   kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=50 | grep "will be overridden"
   ```
   → InputTemplate이 적용되지 않음. Operator 재시작 필요

3. **Referenced task not completed yet**
   ```bash
   kubectl get mcalltask test-source-task -n mcall-dev -o jsonpath='{.status.phase}'
   ```
   → 소스 task가 완료될 때까지 대기

### 해결 방법

```bash
# Operator 재시작
kubectl rollout restart deployment/tz-mcall-operator-dev -n mcall-dev
kubectl rollout status deployment/tz-mcall-operator-dev -n mcall-dev

# CRD 재적용
kubectl apply -f helm/mcall-operator/templates/crds/

# 테스트 재실행
kubectl delete mcalltask test-source-task test-consumer-task -n mcall-dev --ignore-not-found=true
# (위의 테스트 단계 다시 실행)
```

## 📊 추가 통합 테스트

```bash
# 전체 테스트 케이스 실행
kubectl apply -f tests/test-cases/task-result-passing-test-cases.yaml

# 테스트 결과 확인
kubectl get mcalltask -n mcall-dev -l test=result-passing
```

---
**관련 문서**:
- [TEST_REPORT.md](./TEST_REPORT.md) - 유닛 테스트 결과
- [TASK_RESULT_PASSING_DESIGN.md](./docs/TASK_RESULT_PASSING_DESIGN.md) - 설계 문서



