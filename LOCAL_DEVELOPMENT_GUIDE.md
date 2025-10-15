# 로컬 개발 가이드

이 가이드는 Docker Desktop Kubernetes를 사용하여 로컬에서 tz-mcall-operator를 개발하고 테스트하는 방법을 설명합니다.

## 📋 목차

1. [필수 요구사항](#필수-요구사항)
2. [Docker Desktop Kubernetes 설정](#docker-desktop-kubernetes-설정)
3. [로컬 개발 환경 구성](#로컬-개발-환경-구성)
4. [Helm 차트 테스트](#helm-차트-테스트)
5. [Jenkins 이미지로 테스트](#jenkins-이미지로-테스트)
6. [문제 해결](#문제-해결)
7. [정리](#정리)

## 🛠 필수 요구사항

### 설치 필요 항목
- **Docker Desktop**: 최신 버전 (Kubernetes 지원)
- **kubectl**: Kubernetes CLI
- **helm**: Helm 패키지 매니저
- **Git**: 소스코드 관리

### 확인 명령어
```bash
# Docker Desktop 확인
docker --version
docker info

# kubectl 확인
kubectl version --client

# Helm 확인
helm version
```

## 🐳 Docker Desktop Kubernetes 설정

### 1. Kubernetes 활성화
1. **Docker Desktop 애플리케이션 실행**
2. **Settings (설정) 클릭** (우상단 톱니바퀴 아이콘)
3. **Kubernetes 탭 선택**
4. **"Enable Kubernetes" 체크박스 활성화**
5. **"Apply & Restart" 클릭**
6. **재시작 완료까지 대기** (약 2-3분)

### 2. 컨텍스트 전환
```bash
# Docker Desktop Kubernetes 컨텍스트로 전환
kubectl config use-context docker-desktop

# 클러스터 상태 확인
kubectl get nodes
```

## 🔧 로컬 개발 환경 구성

### 1. 소스코드 클론
```bash
git clone https://github.com/doohee323/tz-mcall-operator.git
cd tz-mcall-operator
```

### 2. 로컬 테스트용 Values 파일 생성
```bash
# 로컬 테스트용 values 파일 생성
cat > values-local-test.yaml << EOF
# Local test environment values
environment:
  name: "local-test"
  suffix: "-local-test"

namespace:
  name: "mcall-local-test"
  create: true

controller:
  replicas: 1
  reconcileInterval: 5
  taskTimeout: 5

mcpServer:
  enabled: true  # Jenkins 이미지 사용 시 활성화
  replicas: 1
  
  auth:
    enabled: true
    existingSecret: "mcp-api-keys"
    apiKeys: ""
  
  ingress:
    enabled: false  # 로컬에서는 Ingress 비활성화 (포트 포워딩 사용)

logging:
  enabled: false  # PostgreSQL 없이 테스트

image:
  repository: "doohee323/tz-mcall-operator"
  tag: "latest"
  pullPolicy: "IfNotPresent"

global:
  imageRegistry: ""
  imagePullSecrets:
    - name: tz-registrykey
EOF
```

## 🧪 Helm 차트 테스트

### 1. 차트 검증
```bash
# Helm 차트 문법 검증
helm lint helm/mcall-operator

# 템플릿 생성 테스트
helm template test-local helm/mcall-operator \
  --values values-local-test.yaml \
  --namespace mcall-local-test \
  --create-namespace \
  --debug
```

### 2. 로컬 설치 테스트
```bash
# 로컬에 설치 (CRD 제외)
helm template test-local helm/mcall-operator \
  --values values-local-test.yaml \
  --skip-crds | kubectl apply -f -

# 배포 상태 확인
kubectl get pods -n mcall-local-test
kubectl get services -n mcall-local-test
```

### 3. 컨트롤러 로그 확인
```bash
# 컨트롤러 로그 확인
kubectl logs -n mcall-local-test deployment/test-local-local-test --tail=20

# CRD 감지 확인
kubectl logs -n mcall-local-test deployment/test-local-local-test | grep "CRDs are now available"
```

### 4. 정리
```bash
# 테스트 완료 후 정리
kubectl delete namespace mcall-local-test
```

## 🚀 Jenkins 이미지로 테스트

### 1. 최신 Jenkins 이미지 확인
```bash
# 현재 운영 환경에서 사용 중인 이미지 태그 확인
kubectl get pods -n mcall-dev -o jsonpath='{.items[*].spec.containers[*].image}'
```

### 2. Jenkins 이미지로 로컬 테스트
```bash
# 로컬 테스트용 네임스페이스 생성
kubectl create namespace mcall-dev

# 필요한 Secret 생성
kubectl create secret generic tz-registrykey \
  --from-literal=.dockerconfigjson='{"auths":{"docker.io":{"username":"doohee323","password":"placeholder","auth":"placeholder"}}}' \
  --type=kubernetes.io/dockerconfigjson -n mcall-dev

kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys='["dev-key-123","test-key-456"]' -n mcall-dev

# Helm 차트 설치 (CRD 제외)
helm template test-local helm/mcall-operator \
  --values helm/mcall-operator/values-dev.yaml \
  --skip-crds | kubectl apply -f -

# Jenkins 이미지 태그로 업데이트
kubectl patch deployment test-local-dev -n mcall-dev \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"controller","image":"doohee323/tz-mcall-operator:157"}]}}}}'

kubectl patch deployment test-local-dev-mcp-server -n mcall-dev \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"mcp-server","image":"doohee323/mcall-operator-mcp-server:157"}]}}}}'
```

### 3. 배포 상태 확인
```bash
# Pod 상태 확인
kubectl get pods -n mcall-dev

# 서비스 확인
kubectl get services -n mcall-dev

# 로그 확인
kubectl logs -n mcall-dev deployment/test-local-dev --tail=10
kubectl logs -n mcall-dev deployment/test-local-dev-mcp-server --tail=10
```

### 4. MCP 서버 테스트

#### 옵션 1: 포트 포워딩 사용 (권장)

**방법 1: Lens를 통한 포트 포워딩 (가장 간단)**
1. **Lens 애플리케이션 실행**
2. **Services 탭에서 `test-local-local-dev-mcp-server` 선택**
3. **포트 포워딩 버튼 클릭**
4. **자동으로 할당된 포트로 접근** (예: `http://localhost:55414/`)

**방법 2: kubectl 명령어 사용**
```bash
# MCP 서버 포트 포워딩 (포트 8080)
kubectl port-forward -n mcall-dev service/test-local-local-dev-mcp-server 8080:80 &

# 메트릭스 서버 포트 포워딩 (포트 8081)
kubectl port-forward -n mcall-dev service/test-local-local-dev-metrics 8081:8080 &

# 포트 포워딩 상태 확인
kubectl get services -n mcall-dev

# 헬스 체크
curl http://localhost:8080/health

# API 테스트 (API Key 필요)
curl -H "X-API-Key: dev-key-123" http://localhost:8080/api/workflows/mcall-dev/health-monitor/dag

# 메트릭스 확인
curl http://localhost:8081/metrics
```

**테스트 URL 예시:**
```bash
# Lens 포트 포워딩 사용 시
curl http://localhost:55414/health
curl -H "X-API-Key: dev-key-123" http://localhost:55414/api/workflows/mcall-dev/health-monitor/dag
```

**포트 포워딩 관리:**
```bash
# 실행 중인 포트 포워딩 확인
ps aux | grep "kubectl port-forward"

# 포트 포워딩 중지
pkill -f "kubectl port-forward"

# 특정 포트 포워딩만 중지
kill <PID>
```

#### 옵션 2: Nginx Ingress Controller 설치
Docker Desktop Kubernetes에는 기본적으로 Ingress Controller가 설치되어 있지 않습니다.

```bash
# Nginx Ingress Controller 설치
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml

# 설치 확인
kubectl get pods -n ingress-nginx

# Ingress 리소스 확인
kubectl get ingress -n mcall-dev
```

**참고**: 로컬 개발에서는 포트 포워딩을 사용하는 것이 더 간단합니다.

### 5. 로컬용 API Key 정보
로컬 테스트용으로 설정된 API Key들:

**Secret 생성 시 사용된 API Key들:**
```bash
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys='["dev-key-123","test-key-456"]' -n mcall-dev
```

**사용 가능한 API Key들:**
- `dev-key-123` - 개발용 API Key
- `test-key-456` - 테스트용 API Key

**API 요청 예시:**
```bash
# 헤더에 API Key 포함
curl -H "X-API-Key: dev-key-123" http://localhost:8080/api/workflows/mcall-dev/health-monitor/dag

# 또는 다른 API Key 사용
curl -H "X-API-Key: test-key-456" http://localhost:8080/api/workflows/mcall-dev/health-monitor/dag
```

**API Key 확인 방법:**
```bash
# Secret에서 API Key 확인
kubectl get secret mcp-api-keys -n mcall-dev -o jsonpath='{.data.api-keys}' | base64 -d
```

## 🐛 문제 해결

### 1. 이미지 태그 문제
**문제**: `InvalidImageName` 또는 `ImagePullBackOff` 오류
```bash
# 현재 이미지 확인
kubectl describe pod -n mcall-dev <pod-name>

# 올바른 이미지 태그로 수정
kubectl patch deployment <deployment-name> -n mcall-dev \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container-name>","image":"doohee323/<image-name>:157"}]}}}}'
```

### 2. Secret 누락 문제
**문제**: `CreateContainerConfigError` - Secret not found
```bash
# 필요한 Secret 생성
kubectl create secret generic mcp-api-keys \
  --from-literal=api-keys='["dev-key-123","test-key-456"]' -n mcall-dev

# Pod 재시작으로 Secret 적용
kubectl delete pod -n mcall-dev <pod-name>
```

### 3. CORS 문제
**문제**: `Access to fetch at 'https://mcp-dev.drillquiz.com/api/namespaces' from origin 'http://localhost:64004' has been blocked by CORS policy: Request header field x-api-key is not allowed`

**해결방법**: Ingress에 CORS 헤더 허용 설정 추가
```bash
# CORS 헤더 허용 설정 추가
kubectl patch ingress test-local-dev-mcp-server -n mcall-dev --type='merge' \
  -p='{"metadata":{"annotations":{"nginx.ingress.kubernetes.io/cors-allow-headers":"Content-Type,Authorization,X-API-Key,x-api-key"}}}'

# 설정 확인
kubectl describe ingress test-local-dev-mcp-server -n mcall-dev | grep cors-allow-headers
```

**참고**: Docker Desktop Kubernetes에는 기본적으로 Ingress Controller가 설치되어 있지 않으므로, 로컬 개발에서는 포트 포워딩을 사용하는 것이 권장됩니다.

### 3. CRD 충돌 문제
**문제**: Helm 설치 시 CRD 충돌 오류
```bash
# CRD 제외하고 설치
helm template test-local helm/mcall-operator \
  --values values-local-test.yaml \
  --skip-crds | kubectl apply -f -
```

### 4. 네임스페이스 충돌
**문제**: 네임스페이스 소유권 충돌
```bash
# 새로운 네임스페이스 사용
helm template test-local helm/mcall-operator \
  --values values-local-test.yaml \
  --namespace mcall-local-test \
  --create-namespace \
  --skip-crds | kubectl apply -f -
```

## 🧹 정리

### 테스트 완료 후 정리
```bash
# 로컬 테스트 네임스페이스 삭제
kubectl delete namespace mcall-local-test

# 개발 테스트 네임스페이스 삭제 (필요시)
kubectl delete namespace mcall-dev

# 포트 포워딩 중지
pkill -f "kubectl port-forward"
```

### 리소스 확인
```bash
# 모든 리소스 확인
kubectl get all --all-namespaces

# CRD 확인
kubectl get crd | grep mcall

# Secret 확인
kubectl get secrets --all-namespaces | grep mcall
```

## 📝 유용한 명령어

### 개발 중 자주 사용하는 명령어
```bash
# 실시간 로그 모니터링
kubectl logs -n mcall-dev deployment/test-local-dev -f

# Pod 상태 모니터링
watch kubectl get pods -n mcall-dev

# 서비스 엔드포인트 확인
kubectl get endpoints -n mcall-dev

# 이벤트 확인
kubectl get events -n mcall-dev --sort-by='.lastTimestamp'

# 리소스 사용량 확인
kubectl top pods -n mcall-dev
```

### 디버깅 명령어
```bash
# Pod 상세 정보
kubectl describe pod -n mcall-dev <pod-name>

# 컨테이너 내부 접근
kubectl exec -it -n mcall-dev <pod-name> -- /bin/sh

# 서비스 연결 테스트
kubectl run debug --image=busybox -it --rm --restart=Never -- nslookup <service-name>.<namespace>.svc.cluster.local
```

## 🎯 개발 워크플로우

### 1. 코드 변경 후 테스트
```bash
# 1. 코드 수정
vim controller/controller.go

# 2. 로컬 빌드 (선택사항)
docker build -t doohee323/tz-mcall-operator:local .

# 3. 로컬 테스트
helm template test-local helm/mcall-operator \
  --values values-local-test.yaml \
  --skip-crds | kubectl apply -f -

# 4. 결과 확인
kubectl get pods -n mcall-local-test
kubectl logs -n mcall-local-test deployment/test-local-local-test
```

### 2. Jenkins 배포 후 테스트
```bash
# 1. GitHub에 푸시하여 Jenkins 빌드 트리거
git add .
git commit -m "Test changes"
git push origin mcp

# 2. Jenkins 빌드 완료 대기
# 3. 새 이미지 태그로 로컬 테스트
kubectl patch deployment test-local-dev -n mcall-dev \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"controller","image":"doohee323/tz-mcall-operator:<new-build-number>"}]}}}}'
```

---

이 가이드를 따라하면 Docker Desktop Kubernetes에서 안전하고 효율적으로 로컬 개발을 진행할 수 있습니다.
