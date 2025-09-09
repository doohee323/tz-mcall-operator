# McallTask CRD Testing Guide

## Overview

This document provides comprehensive testing guidelines for the McallTask Custom Resource Definition (CRD) system. The testing suite validates the automatic status initialization, HTTP response validation, CLI command execution, and overall system functionality.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Test Environment Setup](#test-environment-setup)
- [Test Cases](#test-cases)
- [Running Tests](#running-tests)
- [Test Results](#test-results)
- [Troubleshooting](#troubleshooting)
- [Performance Testing](#performance-testing)

## Prerequisites

### System Requirements

- Kubernetes cluster (v1.20+)
- kubectl configured and connected to the cluster
- Helm 3.x installed
- McallTask CRD system deployed
- Controller running in `mcall-system` namespace

### Required Components

1. **CRDs Installed:**
   - `mcalltasks.mcall.tz.io`
   - `mcallworkflows.mcall.tz.io`

2. **Controller Running:**
   - Pod: `tz-mcall-crd-*`
   - Namespace: `mcall-system`
   - Status: `Running`

## Test Environment Setup

### Test Scripts Overview

The testing suite includes two main scripts with different purposes:

#### `local-test.sh` - Development Environment Setup
- **Purpose**: Set up local development environment and verify basic installation
- **Location**: `tests/scripts/local-test.sh`
- **Use Case**: Initial environment setup, basic connectivity testing
- **Features**:
  - Check required tools (helm, kubectl)
  - Verify Kubernetes cluster connectivity
  - Lint and validate Helm chart
  - Install CRDs and Helm chart
  - Create example McallTask
  - Basic status verification

#### `run-tests.sh` - Comprehensive CRD Functionality Testing
- **Purpose**: Run comprehensive tests for McallTask CRD functionality
- **Location**: `tests/test-cases/run-tests.sh`
- **Use Case**: Detailed functionality testing, regression testing
- **Features**:
  - 18 different test cases covering various scenarios
  - Monitor task execution and status changes
  - Validate HTTP responses, CLI outputs, error handling
  - Generate detailed test reports
  - Clean up test resources automatically

### 1. Verify System Status

```bash
# Check if namespace exists
kubectl get namespace mcall-system

# Check if CRDs are installed
kubectl get crd | grep mcall

# Check if controller is running
kubectl get pods -n mcall-system -l app.kubernetes.io/name=tz-mcall-crd
```

### 2. Run System Check

```bash
# Run the test script to check system status
./tests/test-cases/run-tests.sh --check
```

Expected output:
```
✅ Namespace mcall-system exists
✅ mcalltasks.mcall.tz.io exists
✅ mcallworkflows.mcall.tz.io exists
✅ Controller is running (1 pod(s))
✅ System is ready for testing
```

## Test Cases

### 1. Basic Command Execution Test

**Purpose:** Validate basic CLI command execution and status initialization.

**Test Case:** `basic-command-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: basic-command-test
  namespace: mcall-system
spec:
  type: cmd
  input: "echo 'Basic Command Test - $(date)'"
  name: "basic-command-test"
  timeout: 30
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Task executes successfully
- Final status: "Succeeded"
- Output contains timestamp

### 2. HTTP GET Request Test

**Purpose:** Validate HTTP GET request handling and response validation.

**Test Case:** `http-get-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: http-get-test
  namespace: mcall-system
spec:
  type: get
  input: "https://httpbin.org/status/200"
  name: "http-get-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- HTTP request executed successfully
- Response status code: 200
- Final status: "Succeeded"

### 3. Complex Command with JSON Input Test

**Purpose:** Validate complex command execution with multiple inputs including both CLI and HTTP requests.

**Test Case:** `complex-command-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: complex-command-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'System Information:'"
        },
        {
          "type": "cmd",
          "input": "uname -a"
        },
        {
          "type": "cmd",
          "input": "df -h /"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/json"
        }
      ]
    }
  name: "complex-command-test"
  timeout: 120
```

**Expected Results:**
- Status automatically initialized to "Pending"
- All commands execute successfully
- HTTP request returns JSON response
- Final status: "Succeeded"

### 4. Health Check Test

**Purpose:** Validate health check functionality with multiple HTTP status codes.

**Test Case:** `health-check-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: health-check-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'Health Check - $(date)'"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/200"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/201"
        }
      ]
    }
  name: "health-check-test"
  timeout: 90
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Health check commands execute successfully
- HTTP status codes 200 and 201 validated
- Final status: "Succeeded"

### 5. Error Handling Test

**Purpose:** Validate error handling for HTTP error status codes.

**Test Case:** `error-handling-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: error-handling-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'Testing error handling'"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/404"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/500"
        }
      ]
    }
  name: "error-handling-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Error status codes handled appropriately
- Final status: "Failed" (due to error status codes)

### 6. Timeout Test

**Purpose:** Validate timeout handling for long-running tasks.

**Test Case:** `timeout-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: timeout-test
  namespace: mcall-system
spec:
  type: cmd
  input: "sleep 5 && echo 'Timeout test completed'"
  name: "timeout-test"
  timeout: 10
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Task completes within timeout period
- Final status: "Succeeded"

### 7. Large Output Test

**Purpose:** Validate handling of large command outputs.

**Test Case:** `large-output-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: large-output-test
  namespace: mcall-system
spec:
  type: cmd
  input: "echo 'Large output test:' && for i in {1..100}; do echo 'Line $i: This is a test line with some content'; done"
  name: "large-output-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Large output handled successfully
- Final status: "Succeeded"

### 8. JSON Response Test

**Purpose:** Validate JSON response handling from HTTP requests.

**Test Case:** `json-response-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: json-response-test
  namespace: mcall-system
spec:
  type: get
  input: "https://httpbin.org/json"
  name: "json-response-test"
  timeout: 30
```

**Expected Results:**
- Status automatically initialized to "Pending"
- JSON response parsed successfully
- Final status: "Succeeded"

### 9. Multiple HTTP Requests Test

**Purpose:** Validate multiple HTTP requests in a single task.

**Test Case:** `multiple-http-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: multiple-http-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/get"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/headers"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/user-agent"
        }
      ]
    }
  name: "multiple-http-test"
  timeout: 90
```

**Expected Results:**
- Status automatically initialized to "Pending"
- All HTTP requests execute successfully
- Final status: "Succeeded"

### 10. System Resource Test

**Purpose:** Validate system resource information gathering.

**Test Case:** `system-resource-test`

### 11. HTTP Response Validation Test

**Purpose:** Validate HTTP response body content validation.

**Test Case:** `http-response-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: http-response-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/json",
          "expectedResponse": {
            "statusCode": 200,
            "body": {
              "slideshow": {
                "title": "Sample Slide Show"
              }
            }
          }
        }
      ]
    }
  name: "http-response-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- HTTP response body content validated against expected structure
- Final status: "Succeeded"

### 12. HTTP Status Code Validation Test

**Purpose:** Validate HTTP status code validation.

**Test Case:** `http-status-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: http-status-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/status/200",
          "expectedResponse": {
            "statusCode": 200
          }
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/404",
          "expectedResponse": {
            "statusCode": 404
          }
        }
      ]
    }
  name: "http-status-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- HTTP status codes validated against expected values
- Final status: "Succeeded"

### 13. HTTP Header Validation Test

**Purpose:** Validate HTTP response header validation.

**Test Case:** `http-header-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: http-header-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/headers",
          "expectedResponse": {
            "statusCode": 200,
            "headers": {
              "Content-Type": "application/json"
            }
          }
        }
      ]
    }
  name: "http-header-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- HTTP response headers validated against expected values
- Final status: "Succeeded"

### 14. Complex Response Validation Test

**Purpose:** Validate complex HTTP response validation with body, headers, and status codes.

**Test Case:** `complex-response-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: complex-response-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/get?test=validation",
          "expectedResponse": {
            "statusCode": 200,
            "body": {
              "args": {
                "test": "validation"
              }
            },
            "headers": {
              "Content-Type": "application/json"
            }
          }
        }
      ]
    }
  name: "complex-response-validation-test"
  timeout: 90
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Complex response validation (status, body, headers) works correctly
- Final status: "Succeeded"

### 15. CLI Output Validation Test

**Purpose:** Validate CLI command output validation.

**Test Case:** `cli-output-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: cli-output-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'Hello World'",
          "expectedOutput": "Hello World"
        },
        {
          "type": "cmd",
          "input": "date",
          "expectedOutput": {
            "pattern": "\\d{4}-\\d{2}-\\d{2}"
          }
        }
      ]
    }
  name: "cli-output-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- CLI output validation works correctly (exact match and regex pattern)
- Final status: "Succeeded"

### 16. Mixed Validation Test

**Purpose:** Validate mixed CLI and HTTP validation in a single task.

**Test Case:** `mixed-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: mixed-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'Starting validation test'",
          "expectedOutput": "Starting validation test"
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/200",
          "expectedResponse": {
            "statusCode": 200
          }
        }
      ]
    }
  name: "mixed-validation-test"
  timeout: 90
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Both CLI and HTTP validation work together
- Final status: "Succeeded"

### 17. Error Response Validation Test

**Purpose:** Validate error response handling and validation.

**Test Case:** `error-response-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: error-response-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/status/400",
          "expectedResponse": {
            "statusCode": 400
          }
        },
        {
          "type": "get",
          "input": "https://httpbin.org/status/500",
          "expectedResponse": {
            "statusCode": 500
          }
        }
      ]
    }
  name: "error-response-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Error status codes validated correctly
- Final status: "Succeeded"

### 18. JSON Schema Validation Test

**Purpose:** Validate complex JSON schema validation.

**Test Case:** `json-schema-validation-test`

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: json-schema-validation-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "get",
          "input": "https://httpbin.org/json",
          "expectedResponse": {
            "statusCode": 200,
            "body": {
              "slideshow": {
                "title": "Sample Slide Show",
                "date": "date of publication",
                "author": "Yours Truly",
                "slides": [
                  {
                    "title": "Wake up to WonderWidgets!",
                    "type": "all"
                  }
                ]
              }
            }
          }
        }
      ]
    }
  name: "json-schema-validation-test"
  timeout: 60
```

**Expected Results:**
- Status automatically initialized to "Pending"
- Complex JSON schema validation works correctly
- Final status: "Succeeded"

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: system-resource-test
  namespace: mcall-system
spec:
  type: cmd
  input: |
    {
      "inputs": [
        {
          "type": "cmd",
          "input": "echo 'CPU Information:'"
        },
        {
          "type": "cmd",
          "input": "cat /proc/cpuinfo | grep 'model name' | head -1"
        },
        {
          "type": "cmd",
          "input": "echo 'Memory Information:'"
        },
        {
          "type": "cmd",
          "input": "free -h"
        },
        {
          "type": "cmd",
          "input": "echo 'Disk Information:'"
        },
        {
          "type": "cmd",
          "input": "df -h"
        }
      ]
    }
  name: "system-resource-test"
  timeout: 120
```

**Expected Results:**
- Status automatically initialized to "Pending"
- System resource information gathered successfully
- Final status: "Succeeded"

## Running Tests

### 1. Development Environment Setup

For initial setup and basic verification, use the development environment script:

```bash
# Set up development environment (automatically handles cleanup and installation)
./tests/scripts/local-test.sh
```

This script will:
- Check required tools (helm, kubectl, docker)
- Verify Kubernetes cluster connectivity
- Build Docker image locally
- Push image to Docker Hub (with fallback to node loading)
- Force clean up existing resources (including pending Helm releases)
- Install CRDs and Helm chart (with fallback to manual installation)
- Create example McallTask
- Show final status

### 2. Run Comprehensive Tests

#### Check System Status
```bash
# Check prerequisites and system status
./tests/test-cases/run-tests.sh --namespace mcall-dev --check
```

#### Run All Test Cases
```bash
# Run all test cases in development environment
./tests/test-cases/run-tests.sh --namespace mcall-dev --all
```

#### Run Specific Test
```bash
# Run a specific test case
./tests/test-cases/run-tests.sh --namespace mcall-dev basic-command-test
```

#### Run with Debug Mode
```bash
# Run with debug logging
./tests/test-cases/run-tests.sh --namespace mcall-dev --debug http-response-validation-test
```

### 3. Final Test Commands

**Complete Testing Workflow:**
```bash
# Step 1: Set up development environment
./tests/scripts/local-test.sh

# Step 2: Run comprehensive tests
./tests/test-cases/run-tests.sh --namespace mcall-dev --all
```

**Quick Status Check:**
```bash
# Check if system is ready for testing
./tests/test-cases/run-tests.sh --namespace mcall-dev --check
```

## Troubleshooting

### Common Issues

#### Controller Not Running
```bash
# Check controller status
kubectl get pods -n mcall-dev

# Check controller logs
kubectl logs -n mcall-dev -l app.kubernetes.io/name=mcall-crd -f
```

#### Helm Installation Issues
```bash
# Force clean up and retry
./tests/scripts/local-test.sh
```

#### Test Failures
```bash
# Run with debug mode to see detailed output
./tests/test-cases/run-tests.sh --namespace mcall-dev --debug --all
```

### Cleanup Commands

```bash
# Clean up development environment
helm uninstall mcall-crd-dev -n mcall-dev
kubectl delete namespace mcall-dev

# Force clean up (if stuck)
kubectl delete namespace mcall-dev --force --grace-period=0
```

## Performance Testing

### Load Testing
```bash
# Run multiple test cases simultaneously
for i in {1..10}; do
  ./tests/test-cases/run-tests.sh --namespace mcall-dev basic-command-test &
done
wait
```

### Resource Monitoring
```bash
# Monitor resource usage during tests
kubectl top pods -n mcall-dev
kubectl top nodes
```

## Test Results

### Success Criteria

- ✅ All test cases pass
- ✅ Status automatically initialized to "Pending"
- ✅ No "status.phase: Unsupported value" errors
- ✅ HTTP requests execute successfully
- ✅ CLI commands execute successfully
- ✅ Error handling works correctly
- ✅ Timeout handling works correctly

### Test Results Directory

Test results are saved in the `test-results/` directory with the following format:
- `{test-name}_{timestamp}.json`

Each result file contains the complete McallTask resource with status and results.

### Resource Monitoring

```bash
# Monitor controller resource usage
kubectl top pods -n mcall-system

# Check system resources
kubectl get nodes
kubectl describe nodes
```

## Conclusion

This testing guide provides comprehensive validation of the McallTask CRD system. The test suite ensures that:

1. **Automatic status initialization** works correctly
2. **HTTP response validation** functions properly
3. **CLI command execution** operates as expected
4. **Error handling** works appropriately
5. **System performance** meets requirements

Regular testing using this guide ensures the system remains stable and functional across updates and deployments.

## Support

For issues or questions regarding testing:

1. Check the troubleshooting section
2. Review controller logs
3. Verify system prerequisites
4. Contact the development team

---

**Last Updated:** $(date)
**Version:** 1.0.0
**Tested With:** McallTask CRD v1.0.0
