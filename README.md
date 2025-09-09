# tz-mcall-crd

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.19+-blue.svg)](https://kubernetes.io)

**Simple task runner for Kubernetes** - Execute commands and HTTP requests with built-in scheduling, monitoring, and management.

## What does it do?

**Two main things:**
1. **Run Commands** - Execute shell commands in Kubernetes pods
2. **Make HTTP Requests** - GET/POST requests to any URL

**Why CRD?** 
- Kubernetes manages everything (scheduling, monitoring, logs, scaling)
- No need for external cron servers or job schedulers
- Built-in retry, timeout, and error handling

## Key Features

- **Command Execution**: Run any shell command
- **HTTP Requests**: GET/POST to any URL  
- **Scheduling**: Cron-based scheduling (like `crontab`)
- **Parallel Execution**: Run multiple commands/requests at once
- **Error Handling**: Automatic retry and timeout
- **Monitoring**: Built-in Kubernetes monitoring and logging
- **Simple YAML**: Define everything in YAML files

## 📋 Table of Contents

- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [CRD Types](#crd-types)
- [Usage Examples](#usage-examples)
- [Deployment](#deployment)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## 🏗️ Architecture

The system consists of two main CRD types:

- **McallTask**: Individual task definitions (commands, HTTP requests)
- **McallWorkflow**: Grouped tasks with dependencies and execution order

```
              ┌────────────────────────────────────┐
              │       Kubernetes API Server        │
              │  ┌─────────────┐  ┌─────────────┐  │
              │  │ McallTask   │  │McallWorkflow│  │
              │  │   CRD       │  │    CRD      │  │
              │  └─────────────┘  └─────────────┘  │
              └────────────────────────────────────┘
                                │
                      ┌─────────────────┐
                      │   Controller    │ ← Leader Election
                      │   (Operator)    │   (Only one Leader among multiple)
                      └─────────────────┘
                                │
                      ┌─────────────────┐
                      │   Executor      │ ← Actual Worker
                      │   (Pods)        │   (Pods for CRD execution)
                      └─────────────────┘
```

### Component Roles

- **CRDs**: Task definitions and execution metadata
- **Controller**: Manages CRD lifecycle and creates executor pods
- **Executor Pods**: Actual workers that execute the tasks
- **Leader Election**: Ensures only one controller handles scheduled tasks

## 🛠️ Installation

### Prerequisites

- Kubernetes cluster (v1.19+)
- Helm 3.x
- kubectl

### Quick Install

```bash
# Clone the repository
git clone https://github.com/USERNAME/tz-mcall.git
cd tz-mcall

# Install CRDs and Controller
helm install mcall-crd ./helm/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-crd/values-dev.yaml

# Verify installation
kubectl get pods -n mcall-system
kubectl get crd | grep mcall
```

## Quick Start

### 1. Install
```bash
helm install mcall-crd ./helm/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-crd/values-dev.yaml
```

### 2. Run a Command
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: hello-world
spec:
  type: cmd
  input: "echo 'Hello from Kubernetes!'"
```

### 3. Make HTTP Request
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: health-check
spec:
  type: get
  input: "https://api.example.com/health"
  schedule: "*/5 * * * *"  # Every 5 minutes
```

### Apply and Check
```bash
# Apply the task
kubectl apply -f task.yaml

# Check status
kubectl get mcalltasks
kubectl describe mcalltask hello-world
```

## Real Examples

### Daily Database Backup
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: daily-backup
spec:
  type: cmd
  input: "pg_dump mydb > /backup/backup_$(date +%Y%m%d).sql"
  schedule: "0 2 * * *"  # Daily at 2 AM
  timeout: 1800
```

### Health Check
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: api-health
spec:
  type: get
  input: "https://api.example.com/health"
  schedule: "*/5 * * * *"  # Every 5 minutes
  timeout: 10
```

### Cleanup Old Files
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: cleanup-logs
spec:
  type: cmd
  input: "find /var/log -name '*.log' -mtime +30 -delete"
  schedule: "0 0 * * *"  # Daily at midnight
```


## Task Types

### McallTask - Run Commands or HTTP Requests

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: my-task
spec:
  type: cmd                    # cmd, get, or post
  input: "echo 'Hello World'"  # Command or URL
  schedule: "*/5 * * * *"      # Cron schedule (optional)
  timeout: 30                  # Timeout in seconds
  retryCount: 2                # Retry on failure
```

**Types:**
- `cmd`: Execute shell commands
- `get`: HTTP GET request  
- `post`: HTTP POST request

### McallWorkflow - Chain Multiple Tasks

```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: backup-workflow
spec:
  tasks:
  - name: "check-db"
    type: cmd
    input: "kubectl get pods -l app=postgres | grep Running"
  - name: "backup-db"
    type: cmd
    input: "pg_dump mydb > backup.sql"
    dependencies: ["check-db"]  # Only run if check-db succeeds
  schedule: "0 2 * * *"        # Daily at 2 AM
```

**Key Features:**
- **Dependencies**: Tasks run in order
- **Error Handling**: Stop if any task fails
- **Scheduling**: Run entire workflow on schedule


## Why Use This?

| Traditional Cron | tz-mcall-crd |
|------------------|--------------|
| ❌ No monitoring | ✅ Kubernetes monitoring |
| ❌ No retry logic | ✅ Built-in retry |
| ❌ No scaling | ✅ Auto-scaling |
| ❌ Manual setup | ✅ YAML configuration |

## More Examples

### Parallel Health Checks
```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: multi-health-check
spec:
  type: get
  inputs:
    - "https://api.example.com/health"
    - "https://auth.example.com/health"
    - "https://db.example.com/health"
  executionMode: "parallel"  # Check all at once
  timeout: 10
```

### Complex Workflow
```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: deployment-check
spec:
  tasks:
  - name: "check-pods"
    type: cmd
    input: "kubectl get pods | grep Running"
  - name: "test-api"
    type: get
    input: "https://api.example.com/health"
    dependencies: ["check-pods"]
  - name: "send-notification"
    type: post
    input: "https://slack.com/api/chat.postMessage"
    dependencies: ["test-api"]
```

## Installation

### Development
```bash
helm install mcall-crd ./helm/mcall-crd \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-crd/values-dev.yaml
```

### Production
```bash
helm install mcall-crd ./helm/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-crd/values.yaml
```

## Monitoring

### Check Status
```bash
# Check tasks
kubectl get mcalltasks
kubectl get mcallworkflows

# Check controller
kubectl get pods -n mcall-system

# View logs
kubectl logs -n mcall-system -l app.kubernetes.io/name=mcall-crd
```

### Task Status
- `Pending`: Waiting to run
- `Running`: Currently executing  
- `Succeeded`: Completed successfully
- `Failed`: Execution failed

## Troubleshooting

### Task Not Running
```bash
# Check task status
kubectl describe mcalltask <task-name>

# Check controller logs
kubectl logs -n mcall-system deployment/mcall-crd
```

### Common Issues
- **Task stuck in Pending**: Check dependencies and controller status
- **Task Failed**: Check logs and input validation
- **Controller not starting**: Check RBAC permissions and resources

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go coding standards
- Add tests for new features
- Update documentation
- Ensure backward compatibility

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- Dewey Hong - Initial work - [USERNAME](https://github.com/USERNAME)

---

**Made with ❤️ by the tz-mcall team**


