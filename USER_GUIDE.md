# mcall CRD System User Guide

## Overview

This is a step-by-step guide for applying the mcall CRD system to actual production environments. It is written from the perspective of operators/DevOps engineers, not developers.

## Step 1: System Installation and Initial Setup

### 1.1 CRD Installation

```bash
# 1. Install CRDs (McallTask and McallWorkflow implemented)
kubectl apply -f https://raw.githubusercontent.com/USERNAME/tz-mcall-crd/main/helm/mcall-crd/crds/mcalltask-crd.yaml
kubectl apply -f https://raw.githubusercontent.com/USERNAME/tz-mcall-crd/main/helm/mcall-crd/crds/mcallworkflow-crd.yaml

# 2. Verify installation
kubectl get crd | grep mcall

# Implemented CRDs:
# - McallTask: Individual task execution (supports cmd, get, post types)
# - McallWorkflow: Task group management (supports McallTask references and dependencies)
```

### 1.2 Controller Installation

```bash
# Installation using Helm (recommended)
helm repo add mcall-crd https://USERNAME.github.io/tz-mcall-crd
helm install mcall-crd mcall-crd/mcall-crd --namespace mcall-system --create-namespace

# Or direct installation
kubectl apply -f https://raw.githubusercontent.com/USERNAME/tz-mcall-crd/main/ci/k8s-crd.yaml
```

### 1.3 Installation Verification

```bash
# Check controller status
kubectl get pods -n mcall-system

# Check CRDs
kubectl get crd mcalltasks.mcall.tz.io
kubectl get crd mcallworkflows.mcall.tz.io
```

## Step 2: Basic Environment Setup

### 2.1 Default Namespace Setup

```bash
# Default namespace for mcall system (already created)
kubectl get namespace mcall-system

# Create additional namespaces if needed (optional)
kubectl create namespace your-app-namespace
```

### 2.2 RBAC Setup (included in default installation)

```bash
# RBAC is included in default installation
# Additional setup only if needed
kubectl get serviceaccount -n mcall-system
kubectl get clusterrole | grep mcall
```

### 2.3 Basic Configuration (optional)

```yaml
# config.yaml - configure only if needed
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcall-config
  namespace: mcall-system
data:
  # Basic configuration
  default-timeout: "300"
  default-retry-count: "3"
  default-concurrency: "5"
```

## Step 3: Basic Usage

### 3.1 Simple Task Execution

```bash
# 1. Execute basic command
kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: hello-world
  namespace: mcall-system
spec:
  type: cmd
  input: "echo 'Hello, mcall CRD!'"
  timeout: 30
EOF

# 2. Check task status
kubectl get mcalltask hello-world -n mcall-system

# 3. Check task results
kubectl describe mcalltask hello-world -n mcall-system
```

### 3.2 HTTP Request Tasks

```bash
# HTTP GET request
kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: http-test
  namespace: mcall-system
spec:
  type: get
  input: "https://us.drillquiz.com/"
  timeout: 60
EOF
```

### 3.3 Multiple Task Execution Using JSON Input

```bash
# Execute multiple commands using JSON input
kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: multi-task
  namespace: mcall-system
spec:
  type: cmd
  input: |
    [
      {"input": "echo 'Task 1'", "type": "cmd", "name": "task1"},
      {"input": "echo 'Task 2'", "type": "cmd", "name": "task2"},
      {"input": "https://us.drillquiz.com/", "type": "get", "name": "http-task"}
    ]
  executionMode: parallel
  failFast: false
  timeout: 60
EOF
```

### 3.4 McallWorkflow Usage (basic implementation)

```bash
# McallWorkflow has only basic structure implemented
# Currently provides only CRD definition and basic controller

# Workflow creation example (basic structure)
kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: example-workflow
  namespace: mcall-system
spec:
  # Task definition (McallTask reference)
  tasks:
  - name: "workflow-task1"
    taskRef:
      name: "task1"
      namespace: "mcall-system"
    dependencies: []
  
  - name: "workflow-task2"
    taskRef:
      name: "task2"
      namespace: "mcall-system"
    dependencies:
      - "workflow-task1"
EOF

# Check workflow status
kubectl get mcallworkflow example-workflow -n mcall-system

# Check workflow details
kubectl describe mcallworkflow example-workflow -n mcall-system
```

**Note:**
- McallWorkflow has only basic structure implemented
- Cron scheduling and dependency management features are planned for future implementation
- Currently recommend using McallTask individually

### 3.5 Task Cleanup

```bash
# Delete individual tasks
kubectl delete mcalltask hello-world -n mcall-system
kubectl delete mcalltask http-test -n mcall-system
kubectl delete mcalltask multi-task -n mcall-system

# Delete workflow
kubectl delete mcallworkflow example-workflow -n mcall-system
```

## Step 4: Monitoring and Management

### 4.1 Task Status Monitoring

```bash
# Check currently running tasks
kubectl get mcalltasks -A

# Check tasks in specific namespace
kubectl get mcalltasks -n monitoring

# Check task details
kubectl describe mcalltask <task-name> -n <namespace>

# Check task logs
kubectl logs -l app=mcall-controller -n mcall-system
```

### 4.2 Logging Configuration (implemented)

```yaml
# logging-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: logging-config
  namespace: mcall-system
data:
  # Logging backend configuration
  backend: "postgres"  # Supports postgres, mysql, elasticsearch, kafka
  
  # PostgreSQL configuration
  postgres-host: "postgres-service"
  postgres-port: "5432"
  postgres-database: "mcall_logs"
  postgres-table: "task_executions"
  
  # Log level
  log-level: "info"
```

### 4.3 Performance Tuning (default configuration)

```yaml
# controller-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: controller-config
  namespace: mcall-system
data:
  # Default task timeout (seconds)
  default-task-timeout: "300"
  
  # Default retry configuration
  default-retry-count: "0"
  
  # Resource limits (configured in Helm values)
  cpu-limit: "500m"
  memory-limit: "512Mi"
```

## Step 5: Troubleshooting

### 5.1 Common Issues

#### When tasks are not executing
```bash
# 1. Check controller status
kubectl get pods -n mcall-system

# 2. Check task status
kubectl describe mcalltask <task-name> -n mcall-system

# 3. Check events
kubectl get events -n mcall-system --sort-by='.lastTimestamp'
```

#### Check logs
```bash
# Controller logs
kubectl logs -l app=mcall-controller -n mcall-system --tail=100

# Specific task logs
kubectl logs -l mcall.tz.io/task=<task-name> -n mcall-system
```

## Step 6: Real Production Scenarios

### 6.1 Microservice Monitoring and Health Checks

#### Required resources and configuration
```bash
# Create namespace
kubectl create namespace monitoring

# Configure service endpoints
kubectl create configmap service-endpoints -n monitoring \
  --from-literal=user-service="http://user-service.monitoring.svc.cluster.local:8080/health" \
  --from-literal=order-service="http://order-service.monitoring.svc.cluster.local:8080/health"
```

#### Monitoring tasks (using McallTask)
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: microservices-health-check
  namespace: monitoring
spec:
  type: cmd
  input: |
    [
      {"input": "curl -f http://user-service.monitoring.svc.cluster.local:8080/health", "type": "cmd", "name": "user-service-health"},
      {"input": "curl -f http://order-service.monitoring.svc.cluster.local:8080/health", "type": "cmd", "name": "order-service-health"},
      {"input": "kubectl get pods -n monitoring | grep -E '(user-service|order-service)' | grep Running | wc -l", "type": "cmd", "name": "pod-count-check"}
    ]
  executionMode: parallel
  failFast: true
  timeout: 60
```

### 6.2 Database Backup Automation

#### Required resources and configuration
```bash
# Create namespace
kubectl create namespace backup

# Configure database access credentials
kubectl create secret generic postgres-backup-credentials -n backup \
  --from-literal=username="backup_user" \
  --from-literal=password="" \
  --from-literal=host="postgres-primary.database.svc.cluster.local"
```

#### Backup tasks (using McallTask)
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: database-backup
  namespace: backup
spec:
  type: cmd
  input: |
    [
      {"input": "kubectl get pods -n database | grep postgres | grep Running | wc -l", "type": "cmd", "name": "pre-backup-check"},
      {"input": "pg_dump -h postgres-primary.database.svc.cluster.local -U backup_user -d maindb | gzip > /backup/postgres_$(date +%Y%m%d_%H%M%S).sql.gz", "type": "cmd", "name": "postgres-backup"},
      {"input": "ls -la /backup/postgres_*.sql.gz | tail -1", "type": "cmd", "name": "backup-verification"}
    ]
  executionMode: sequential
  failFast: true
  timeout: 1800
```

### 6.3 CI/CD Pipeline Integration

#### Deployment validation tasks (using McallTask)
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: deployment-validation
  namespace: cicd
spec:
  type: cmd
  input: |
    [
      {"input": "kubectl get nodes | grep Ready | wc -l", "type": "cmd", "name": "node-check"},
      {"input": "kubectl get pods -n staging | grep app | grep Running | wc -l", "type": "cmd", "name": "staging-check"},
      {"input": "curl -f http://app-staging.company.com/api/health", "type": "get", "name": "health-check"}
    ]
  executionMode: sequential
  failFast: true
  timeout: 300
```

## Conclusion

By following this guide, you can easily get started with the mcall CRD system and apply it to real production environments.

### 📋 Quick Start Checklist

- [ ] CRD and Controller installation completed
- [ ] Basic task execution test completed
- [ ] HTTP request task test completed
- [ ] Task status monitoring verification completed
- [ ] Production scenario application completed