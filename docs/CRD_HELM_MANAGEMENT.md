# CRD Helm Management Guide

## 문제 상황

kubectl로 수동 적용한 CRD는 Helm이 관리할 수 없어 다음과 같은 에러가 발생합니다:

```
Error: UPGRADE FAILED: Unable to continue with update: CustomResourceDefinition "mcalltasks.mcall.tz.io" 
in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; 
label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"
```

## 해결 방법

### 기존 CRD를 Helm 관리로 전환

```bash
# McallTask CRD
kubectl label crd mcalltasks.mcall.tz.io app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate crd mcalltasks.mcall.tz.io meta.helm.sh/release-name=tz-mcall-operator-dev --overwrite
kubectl annotate crd mcalltasks.mcall.tz.io meta.helm.sh/release-namespace=mcall-dev --overwrite

# McallWorkflow CRD
kubectl label crd mcallworkflows.mcall.tz.io app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate crd mcallworkflows.mcall.tz.io meta.helm.sh/release-name=tz-mcall-operator-dev --overwrite
kubectl annotate crd mcallworkflows.mcall.tz.io meta.helm.sh/release-namespace=mcall-dev --overwrite
```

### Production (mcall-system) 환경의 경우

```bash
# McallTask CRD
kubectl label crd mcalltasks.mcall.tz.io app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate crd mcalltasks.mcall.tz.io meta.helm.sh/release-name=tz-mcall-operator --overwrite
kubectl annotate crd mcalltasks.mcall.tz.io meta.helm.sh/release-namespace=mcall-system --overwrite

# McallWorkflow CRD
kubectl label crd mcallworkflows.mcall.tz.io app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate crd mcallworkflows.mcall.tz.io meta.helm.sh/release-name=tz-mcall-operator --overwrite
kubectl annotate crd mcallworkflows.mcall.tz.io meta.helm.sh/release-namespace=mcall-system --overwrite
```

### 확인

```bash
kubectl get crd mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io \
  -o custom-columns=NAME:.metadata.name,MANAGED-BY:.metadata.labels.app\\.kubernetes\\.io/managed-by,RELEASE:.metadata.annotations.meta\\.helm\\.sh/release-name
```

**예상 출력**:
```
NAME                         MANAGED-BY   RELEASE
mcalltasks.mcall.tz.io       Helm         tz-mcall-operator-dev
mcallworkflows.mcall.tz.io   Helm         tz-mcall-operator-dev
```

## 모범 사례

### ❌ 하지 말 것

```bash
# CRD를 kubectl로 직접 적용하지 마세요
kubectl apply -f helm/mcall-operator/templates/crds/
```

### ✅ 권장 방법

1. **Jenkins를 통한 배포만 사용**
   - Jenkins 파이프라인이 자동으로 Helm을 통해 CRD 배포
   - 올바른 메타데이터가 자동으로 추가됨

2. **로컬 테스트가 필요한 경우**
   ```bash
   # Helm을 사용하여 설치
   helm upgrade --install tz-mcall-operator-dev ./helm/mcall-operator \
     -f ./helm/mcall-operator/values-dev.yaml \
     -n mcall-dev \
     --create-namespace
   ```

3. **CRD만 업데이트하는 경우**
   ```bash
   # controller-gen으로 CRD 재생성
   ~/go/bin/controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds
   
   # Git에 커밋하고 푸시
   git add helm/mcall-operator/templates/crds/
   git commit -m "Update CRDs"
   git push
   
   # Jenkins에서 재빌드 (Helm이 자동으로 CRD 업데이트)
   ```

## CRD 업데이트 워크플로우

### 1. API 타입 수정
```bash
# api/v1/mcalltask_types.go 또는 mcallworkflow_types.go 수정
vim api/v1/mcalltask_types.go
```

### 2. CRD 재생성
```bash
~/go/bin/controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds
```

### 3. DeepCopy 메서드 재생성 (필요시)
```bash
~/go/bin/controller-gen object paths=./api/...
```

### 4. 커밋 및 푸시
```bash
git add api/v1/ helm/mcall-operator/templates/crds/
git commit -m "Update API types and CRDs"
git push origin <branch>
```

### 5. Jenkins 빌드
- Jenkins UI에서 해당 브랜치 빌드 실행
- Helm이 자동으로 CRD 업데이트

## 트러블슈팅

### 문제: Helm upgrade가 CRD ownership 에러로 실패

**원인**: CRD가 kubectl로 수동 적용되어 Helm 메타데이터가 없음

**해결**:
```bash
# 위의 "기존 CRD를 Helm 관리로 전환" 스크립트 실행
```

### 문제: CRD 변경사항이 반영되지 않음

**원인**: Helm은 기본적으로 기존 CRD를 업그레이드하지 않음

**해결**:
1. Jenkins 파이프라인이 `helm upgrade` 전에 CRD를 명시적으로 적용하는지 확인
2. 또는 수동으로 CRD 적용 후 Helm 메타데이터 추가:
   ```bash
   kubectl apply -f helm/mcall-operator/templates/crds/
   # 위의 메타데이터 추가 스크립트 실행
   ```

### 문제: 여러 환경(dev, staging, prod)에서 CRD 버전이 다름

**원인**: CRD는 클러스터 범위 리소스이므로 namespace별로 다른 버전을 가질 수 없음

**권장 사항**:
- 모든 환경에서 동일한 CRD 버전 사용
- 하위 호환성을 유지하도록 API 설계
- Breaking change가 필요한 경우 새로운 API 버전 사용 (v1 → v2)

## 참고 자료

- [Helm CRD 관리 문서](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)
- [controller-gen 사용법](https://book.kubebuilder.io/reference/controller-gen.html)
- [Kubernetes API 버전 관리](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)

---
**작성일**: 2025-10-10  
**관련 이슈**: Jenkins 배포 시 CRD ownership 에러

