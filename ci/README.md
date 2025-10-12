# CI/CD Documentation

## Jenkins Pipeline Structure

This project uses Jenkins for automated builds and deployments.

### Built Images

1. **Operator Image**: `doohee323/tz-mcall-operator:${BUILD_NUMBER}`
   - Kubernetes CRD controller
   - Written in Go
   - Dockerfile: `docker/Dockerfile`

2. **MCP Server Image**: `doohee323/mcall-operator-mcp-server:${BUILD_NUMBER}`
   - Model Context Protocol server
   - Written in TypeScript/Node.js
   - Dockerfile: `mcp-server/Dockerfile`

### Pipeline Stages

1. **Checkout**: Checkout Git repository
2. **Configuration**: Configure branch-specific settings
3. **Build & Push Images**: 
   - Build and push Operator image
   - Build and push MCP Server image
4. **Helm Chart Validation**: Validate Helm chart
5. **Deploy CRDs**: Deploy CRDs
6. **Deploy Helm Chart**: Deploy Helm chart
7. **Post-Deployment Tests**: Run post-deployment tests

### Branch-Based Deployment Strategy

#### main branch
- Namespace: `mcall-system`
- Values file: `values.yaml`
- Operator image tag: `${BUILD_NUMBER}`
- MCP Server image tag: `${BUILD_NUMBER}` + `latest`
- MCP Server enabled: false (optional in production)

#### qa branch
- Namespace: `mcall-system`
- Values file: `values-staging.yaml`
- Operator image tag: `${BUILD_NUMBER}`
- MCP Server image tag: `${BUILD_NUMBER}` + `staging`
- MCP Server enabled: Configurable

#### dev branch (others)
- Namespace: `mcall-dev`
- Values file: `values-dev.yaml`
- Operator image tag: `${BUILD_NUMBER}` + `latest`
- MCP Server image tag: `${BUILD_NUMBER}` + `dev`
- MCP Server enabled: true

### Environment Variables

Required environment variables:
- `DOCKERHUB_CREDENTIALS_ID`: Docker Hub credentials
- `KUBECONFIG_CREDENTIALS_ID`: Kubernetes configuration
- `POSTGRES_PASSWORD`: PostgreSQL password (for logging)

Optional environment variables:
- `MYSQL_PASSWORD`: MySQL password
- `ELASTICSEARCH_PASSWORD`: Elasticsearch password

### k8s.sh Script

`ci/k8s.sh` is the core deployment script.

#### Usage

```bash
./ci/k8s.sh <BUILD_NUMBER> <GIT_BRANCH> <NAMESPACE> <VALUES_FILE> <ACTION>
```

#### Actions

- `deploy`: Perform full deployment
- `verify`: Verify deployment only
- `test`: Test CRD functionality
- `rollback`: Rollback to previous version
- `cleanup`: Clean up resources
- `force-cleanup`: Force cleanup

#### Examples

```bash
# Development deployment
./ci/k8s.sh latest dev mcall-dev values-dev.yaml deploy

# Production deployment
./ci/k8s.sh 123 main mcall-system values.yaml deploy

# Rollback
./ci/k8s.sh latest main mcall-system values.yaml rollback
```

## MCP Server Integration

### Build Process

MCP Server is automatically built with the operator:

1. Jenkins builds image using `mcp-server/Dockerfile`
2. Tags with build number
3. Additional branch-specific tags (`dev`, `staging`, `latest`)
4. Pushes to Docker Hub

### Deployment Control

MCP Server is controlled via Helm values files:

```yaml
mcpServer:
  enabled: true  # Deploy when set to true
  image:
    repository: doohee323/mcall-operator-mcp-server
    tag: "dev"  # Jenkins overrides with BUILD_NUMBER
```

### Branch-Specific Activation

- **dev**: MCP Server enabled (values-dev.yaml)
- **qa**: Configurable (values-staging.yaml)
- **main**: Disabled (values.yaml) - optional deployment in production

## Local Testing

### Build Operator Only

```bash
make build-docker
```

### Build MCP Server Only

```bash
make build-mcp-docker
```

### Build All Images

```bash
make build-docker-all
```

### Test Local Deployment

```bash
make deploy-dev
```

## 🐛 Bug Fixes & Change Log

### HTTP Status Code Validation Fix (2025-10-10)

**Issue**: Health check workflow가 간헐적으로 잘못된 성공/실패 판정을 하는 문제 발견
- `executeHTTPRequest` 함수가 HTTP 상태 코드를 검증하지 않음
- 404, 503 등 에러 응답도 Task Phase "Succeeded"로 처리
- 예: `https://us.drillquiz.com/aaa` 가 503 반환해도 성공으로 처리

**Fix**: 
```go
// controller/controller.go:539-542
// Check HTTP status code - fail if not 2xx
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    return string(doc), fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
}
```

**Deployment**:
- ⚠️ **모든 배포는 Jenkins CI/CD를 통해서만 수행**
- Git push 후 Jenkins가 자동으로 빌드 및 배포 처리
- 로컬 `make build-docker`, `make deploy`는 개발/테스트 용도만

**Testing After Deployment**:
```bash
# 배포 후 health monitor workflow 확인
kubectl get mcallworkflow health-monitor -n mcall-dev
kubectl get mcalltask -n mcall-dev -l mcall.tz.io/workflow=health-monitor

# 로그 확인
tail -f /app/log/mcall/health_monitor.log
# 이제 200 응답만 SUCCESS, 나머지는 FAILED로 정확하게 구분됨
```

---

## Troubleshooting

### MCP Server Image Build Failure

1. Check `mcp-server/package.json`
2. Verify Node.js version (Dockerfile uses node:20-alpine)
3. Check TypeScript compilation errors in build logs

### Deployment Failure

1. Validate Helm chart:
   ```bash
   helm lint ./helm/mcall-operator
   ```

2. Check values file:
   ```bash
   helm template test ./helm/mcall-operator -f ./helm/mcall-operator/values-dev.yaml
   ```

3. Verify image exists:
   ```bash
   docker pull doohee323/mcall-operator-mcp-server:dev
   ```

### MCP Server Pod Not Starting

1. Check pod logs:
   ```bash
   kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server
   ```

2. Check pod status:
   ```bash
   kubectl describe pod -n mcall-dev -l app.kubernetes.io/component=mcp-server
   ```

3. Check RBAC permissions:
   ```bash
   kubectl auth can-i create mcalltasks --as=system:serviceaccount:mcall-dev:mcall-operator-mcp-server
   ```

## Monitoring

### Check from Jenkins

Information automatically displayed after pipeline execution:
- Pod status
- CRD status
- Service status

### Manual Check

```bash
# Pod status
kubectl get pods -n mcall-dev

# MCP Server status
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Logs
kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f
```

## References

- [Jenkinsfile](./Jenkinsfile) - Pipeline definition
- [k8s.sh](./k8s.sh) - Deployment script
- [MCP Server Guide](../MCP_SERVER_GUIDE.md) - Complete MCP server guide
- [Helm Chart Guide](../helm/mcall-operator/README.md) - Helm deployment and local testing
- [Main Documentation](../README.md) - tz-mcall-operator main docs
