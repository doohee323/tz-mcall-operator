# CRD Helm Management Guide

## Problem

CRDs manually applied with kubectl cannot be managed by Helm, resulting in errors like:

```
Error: UPGRADE FAILED: Unable to continue with update: CustomResourceDefinition "mcalltasks.mcall.tz.io" 
in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; 
label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"
```

## Solution

### Transfer Existing CRDs to Helm Management

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

### For Production (mcall-system) Environment

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

### Verification

```bash
kubectl get crd mcalltasks.mcall.tz.io mcallworkflows.mcall.tz.io \
  -o custom-columns=NAME:.metadata.name,MANAGED-BY:.metadata.labels.app\\.kubernetes\\.io/managed-by,RELEASE:.metadata.annotations.meta\\.helm\\.sh/release-name
```

**Expected output**:
```
NAME                         MANAGED-BY   RELEASE
mcalltasks.mcall.tz.io       Helm         tz-mcall-operator-dev
mcallworkflows.mcall.tz.io   Helm         tz-mcall-operator-dev
```

## Best Practices

### ❌ Don't Do This

```bash
# Don't apply CRDs directly with kubectl
kubectl apply -f helm/mcall-operator/templates/crds/
```

### ✅ Recommended Approach

1. **Use Jenkins Deployment Only**
   - Jenkins pipeline automatically deploys CRDs via Helm
   - Correct metadata is automatically added

2. **For Local Testing**
   ```bash
   # Install using Helm
   helm upgrade --install tz-mcall-operator-dev ./helm/mcall-operator \
     -f ./helm/mcall-operator/values-dev.yaml \
     -n mcall-dev \
     --create-namespace
   ```

3. **When Updating CRDs Only**
   ```bash
   # Regenerate CRDs with controller-gen
   ~/go/bin/controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds
   
   # Commit and push to Git
   git add helm/mcall-operator/templates/crds/
   git commit -m "Update CRDs"
   git push
   
   # Rebuild in Jenkins (Helm automatically updates CRDs)
   ```

## CRD Update Workflow

### 1. Modify API Types
```bash
# Edit api/v1/mcalltask_types.go or mcallworkflow_types.go
vim api/v1/mcalltask_types.go
```

### 2. Regenerate CRDs
```bash
~/go/bin/controller-gen crd paths=./api/... output:crd:dir=./helm/mcall-operator/templates/crds
```

### 3. Regenerate DeepCopy Methods (if needed)
```bash
~/go/bin/controller-gen object paths=./api/...
```

### 4. Commit and Push
```bash
git add api/v1/ helm/mcall-operator/templates/crds/
git commit -m "Update API types and CRDs"
git push origin <branch>
```

### 5. Jenkins Build
- Run build for the branch in Jenkins UI
- Helm automatically updates CRDs

## Troubleshooting

### Issue: Helm upgrade fails with CRD ownership error

**Cause**: CRDs were manually applied with kubectl and lack Helm metadata

**Solution**:
```bash
# Run the "Transfer Existing CRDs to Helm Management" script above
```

### Issue: CRD changes not reflected

**Cause**: Helm does not upgrade existing CRDs by default

**Solution**:
1. Verify Jenkins pipeline explicitly applies CRDs before `helm upgrade`
2. Or manually apply CRDs and add Helm metadata:
   ```bash
   kubectl apply -f helm/mcall-operator/templates/crds/
   # Run metadata addition script above
   ```

### Issue: Different CRD versions across environments (dev, staging, prod)

**Cause**: CRDs are cluster-scoped resources and cannot have different versions per namespace

**Recommendation**:
- Use the same CRD version across all environments
- Design APIs to maintain backward compatibility
- Use new API versions (v1 → v2) for breaking changes

## References

- [Helm CRD Management Documentation](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)
- [controller-gen Usage](https://book.kubebuilder.io/reference/controller-gen.html)
- [Kubernetes API Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)

---
**Date**: 2025-10-10  
**Related Issue**: CRD ownership error during Jenkins deployment
