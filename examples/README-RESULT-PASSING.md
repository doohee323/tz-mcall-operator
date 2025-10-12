# Task Result Passing - Quick Start Guide

This guide explains how to test task result passing and conditional execution features.

## 📋 Overview

The health check example demonstrates the following features:
1. **Task A**: `us.drillquiz.com` Health Check (GET request)
2. **Task B**: Log success when health check succeeds
3. **Task C**: Log failure when health check fails

## 🚀 Deployment and Testing

### 1. Update CRDs

```bash
# Regenerate CRDs (already completed)
cd /Users/dhong/workspaces/tz-mcall-operator
export PATH=$PATH:$(go env GOPATH)/bin
controller-gen crd:crdVersions=v1 paths="./api/..." output:crd:artifacts:config=helm/mcall-operator/templates/crds

# Apply CRDs to Kubernetes
kubectl apply -f helm/mcall-operator/templates/crds/mcalltask-crd.yaml
kubectl apply -f helm/mcall-operator/templates/crds/mcallworkflow-crd.yaml
```

### 2. Restart Controller

```bash
# Build and redeploy controller image
make docker-build
make docker-push

# Or redeploy with Helm
helm upgrade --install mcall-operator ./helm/mcall-operator -n mcall-dev

# Wait for controller pod restart
kubectl rollout status deployment/mcall-operator -n mcall-dev
```

### 3. Deploy Test Workflow

```bash
# Deploy workflow and template tasks
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

# Verify deployment
kubectl get mcalltasks -n mcall-dev
kubectl get mcallworkflows -n mcall-dev
```

### 4. Monitor Execution Results

```bash
# Check workflow status
kubectl get mcallworkflow health-monitor -n mcall-dev -o yaml

# Check task statuses
kubectl get mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Task details (first execution)
kubectl describe mcalltask health-monitor-healthcheck -n mcall-dev
kubectl describe mcalltask health-monitor-log-success -n mcall-dev
kubectl describe mcalltask health-monitor-log-failure -n mcall-dev

# Check logs (only successful tasks will have output)
kubectl logs -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Check result file (in controller pod)
CONTROLLER_POD=$(kubectl get pods -n mcall-dev -l app=mcall-operator -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n mcall-dev $CONTROLLER_POD -- cat /app/log/mcall/health_monitor.log
```

## 📊 Expected Results

### When Health Check Succeeds

```yaml
# health-monitor-healthcheck
status:
  phase: Succeeded
  result:
    errorCode: "0"
    output: "<!doctype html>..."

# health-monitor-log-success (executed)
status:
  phase: Succeeded
  result:
    errorCode: "0"
    output: |
      [2025-10-10 15:30:00] ✅ SUCCESS - Phase: Succeeded, ErrorCode: 0
        Started: 2025-10-10T15:29:58Z, Completed: 2025-10-10T15:30:00Z
        us.drillquiz.com is UP
      ---
      (last 30 lines)

# health-monitor-log-failure (skipped)
status:
  phase: Skipped
  result:
    errorCode: "0"
    errorMessage: "Skipped due to condition: when=failure"
```

### When Health Check Fails

```yaml
# health-monitor-healthcheck
status:
  phase: Failed
  result:
    errorCode: "-1"
    errorMessage: "failed to execute GET request: ..."

# health-monitor-log-success (skipped)
status:
  phase: Skipped

# health-monitor-log-failure (executed)
status:
  phase: Succeeded
  result:
    output: |
      [2025-10-10 15:30:00] ❌ FAILED - Phase: Failed, ErrorCode: -1
        Error: failed to execute GET request: timeout
        us.drillquiz.com is DOWN
```

## 🔍 Debugging

### Check Controller Logs

```bash
# Check InputSources processing in controller logs
kubectl logs -n mcall-dev -l app=mcall-operator | grep -A5 "Extracted data from source task"

# Example output:
# Extracted data from source task task=health-monitor-log-success sourceTask=health-monitor-healthcheck field=phase varName=HEALTH_PHASE valuePreview=Succeeded
# Extracted data from source task task=health-monitor-log-success sourceTask=health-monitor-healthcheck field=errorCode varName=ERROR_CODE valuePreview=0
```

### Check Task Condition

```bash
# Check condition stored in task annotation
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.metadata.annotations.mcall\.tz\.io/condition}' | jq .

# Example output:
# {
#   "dependentTask": "health-monitor-healthcheck",
#   "when": "success"
# }
```

### Check InputSources

```bash
# Check inputSources stored in task spec
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.spec.inputSources}' | jq .

# Example output:
# [
#   {
#     "name": "HEALTH_PHASE",
#     "taskRef": "health-monitor-healthcheck",
#     "field": "phase"
#   },
#   ...
# ]
```

## 🧪 Additional Tests

### 1. Manually Check Task Results

```bash
# Health check task results
kubectl get mcalltask health-monitor-healthcheck -n mcall-dev -o jsonpath='{.status.result}' | jq .

# Output:
# {
#   "output": "<!doctype html>...",
#   "errorCode": "0",
#   "errorMessage": ""
# }
```

### 2. Re-run Workflow

```bash
# Workflow has a schedule, so it will re-run automatically
# To immediately re-run, delete and recreate the workflow
kubectl delete mcallworkflow health-monitor -n mcall-dev
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

# Or delete only tasks (workflow will create new tasks)
kubectl delete mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor
```

### 3. Test Failure Scenario

To test health check with an invalid URL:

```bash
# Edit template task
kubectl edit mcalltask health-check-template -n mcall-dev

# Change spec.input to an invalid URL
# input: https://invalid-domain-that-does-not-exist.com

# Re-run workflow
kubectl delete mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor
```

## 📝 Cleanup

```bash
# Delete test resources
kubectl delete mcallworkflow health-monitor -n mcall-dev
kubectl delete mcalltasks -n mcall-dev health-check-template log-success-template log-failure-template

# Delete log file (in controller pod)
CONTROLLER_POD=$(kubectl get pods -n mcall-dev -l app=mcall-operator -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n mcall-dev $CONTROLLER_POD -- rm -f /app/log/mcall/health_monitor.log
```

## 🎯 Next Steps

1. **Add MCP Tools**: `get_task_result_schema`, `get_task_result_json`
2. **Test Complex JSONPath**: Extract specific fields from API JSON responses
3. **Multiple InputSources**: Use results from multiple tasks in one task
4. **Complex Conditions**: Test AND, OR conditions

## 🐛 Troubleshooting

### Task Stays in Pending State

```bash
# Dependent task may not be completed
kubectl get mcalltasks -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# Check controller logs
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Waiting for"
```

### InputTemplate Not Applied

```bash
# Check task spec
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.spec.inputTemplate}'

# Check template rendering in controller logs
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Rendered input template"
```

### Condition Not Working

```bash
# Check annotation
kubectl get mcalltask health-monitor-log-success -n mcall-dev -o jsonpath='{.metadata.annotations}'

# Check condition check in controller logs
kubectl logs -n mcall-dev -l app=mcall-operator | grep "Task condition"
```
