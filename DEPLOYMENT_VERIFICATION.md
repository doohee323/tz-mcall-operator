# Deployment Verification Guide

## 🚀 Post-Jenkins Deployment Verification

### 1. Verify Operator Deployment

```bash
# Check Operator Pod status
kubectl get pods -n mcall-dev -l app=tz-mcall-operator-dev

# Check Operator logs (look for InputSources processing logs)
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=50 | grep -i "input"
```

**Expected output**:
```
tz-mcall-operator-dev-xxxxx   1/1   Running   0   2m
```

### 2. Verify CRD Updates

```bash
# Check if inputSources field exists in CRD
kubectl get crd mcalltasks.mcall.tz.io -o yaml | grep -A 10 "inputSources"
```

**Expected output**: Should display inputSources and inputTemplate schema definitions

### 3. Test Task Result Passing

#### Step 1: Create Source Task
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

#### Step 2: Wait for Completion (about 5 seconds)
```bash
kubectl wait --for=jsonpath='{.status.phase}'=Succeeded --timeout=30s mcalltask/test-source-task -n mcall-dev
```

#### Step 3: Create Consumer Task
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

#### Step 4: Verify Results
```bash
# Wait for consumer task completion
kubectl wait --for=jsonpath='{.status.phase}'=Succeeded --timeout=30s mcalltask/test-consumer-task -n mcall-dev

# Check results
kubectl get mcalltask test-consumer-task -n mcall-dev -o jsonpath='{.status.result.output}'
```

**Expected output**:
```
=== Task Result Passing Test ===
Source output: SUCCESS_FROM_SOURCE
Source phase: Succeeded
Source error code: 0
=============================
```

### 4. Test Conditional Execution

```bash
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml
```

**Verification points**:
- When healthcheck task succeeds → only log-success task runs
- When healthcheck task fails → only log-failure task runs

```bash
# Check workflow status
kubectl get mcallworkflow health-monitor -n mcall-dev

# List tasks
kubectl get mcalltask -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Check specific task status
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o yaml
kubectl get mcalltask health-monitor-log-failure -n mcall-dev -o yaml
```

### 5. Verify in Operator Logs

```bash
# Check InputSources processing logs
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=100 | grep -A 5 "Injected data from input sources"

# Check conditional execution logs
kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=100 | grep -A 5 "Task condition"
```

**Expected logs**:
```
Injected data from input sources
  task: test-consumer-task
  sourceCount: 3
  envVars: 3
```

### 6. Cleanup

```bash
# Delete test resources
kubectl delete mcalltask test-source-task test-consumer-task -n mcall-dev
kubectl delete mcallworkflow health-monitor -n mcall-dev
```

## ✅ Success Criteria

- [ ] Operator pod is in Running state
- [ ] CRD has inputSources and inputTemplate fields
- [ ] Test consumer task is in Succeeded state
- [ ] Consumer task output contains source task data
- [ ] In conditional workflow, only tasks matching conditions are executed
- [ ] Operator logs contain "Injected data from input sources" message

## 🐛 Troubleshooting

### When Consumer Task is in Failed State

1. **If spec has no inputSources**
   ```bash
   kubectl get mcalltask test-consumer-task -n mcall-dev -o yaml | grep -A 10 "spec:"
   ```
   → CRD may not be properly updated

2. **If "will be overridden" command was executed**
   ```bash
   kubectl logs -n mcall-dev deployment/tz-mcall-operator-dev --tail=50 | grep "will be overridden"
   ```
   → InputTemplate not applied. Operator restart required

3. **Referenced task not completed yet**
   ```bash
   kubectl get mcalltask test-source-task -n mcall-dev -o jsonpath='{.status.phase}'
   ```
   → Wait for source task to complete

### Solutions

```bash
# Restart operator
kubectl rollout restart deployment/tz-mcall-operator-dev -n mcall-dev
kubectl rollout status deployment/tz-mcall-operator-dev -n mcall-dev

# Reapply CRDs
kubectl apply -f helm/mcall-operator/templates/crds/

# Re-run tests
kubectl delete mcalltask test-source-task test-consumer-task -n mcall-dev --ignore-not-found=true
# (Re-run the test steps above)
```

## 📊 Additional Integration Tests

```bash
# Run all test cases
kubectl apply -f tests/test-cases/task-result-passing-test-cases.yaml

# Check test results
kubectl get mcalltask -n mcall-dev -l test=result-passing
```

---
**Related Documentation**:
- [TEST_REPORT.md](./TEST_REPORT.md) - Unit test results
- [TASK_RESULT_PASSING_DESIGN.md](./docs/TASK_RESULT_PASSING_DESIGN.md) - Design document




