# mcall-crd Helm Chart

A Helm chart for deploying the mcall CRD-based task execution system on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.x
- cert-manager (for webhook certificates)

## Installation

### Add the Helm repository

```bash
helm repo add mcall https://charts.example.com/mcall
helm repo update
```

### Install the chart

```bash
# Install with default values
helm install mcall-crd mcall/mcall-crd

# Install with custom values
helm install mcall-crd mcall/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values values.yaml

# Install for development
helm install mcall-crd-dev mcall/mcall-crd \
  --namespace mcall-dev \
  --create-namespace \
  --values values-dev.yaml

# Install with environment variables (recommended)
export POSTGRES_PASSWORD=""
./install-with-env.sh
```

## Security Configuration

### Secret Management

Sensitive information such as database passwords are managed through Kubernetes Secrets. The chart automatically creates secrets for logging backends.

#### Setting up secrets

**Using environment variables**
```bash
# Set environment variables
export POSTGRES_PASSWORD=""
export MYSQL_PASSWORD="your-mysql-password"
export ELASTICSEARCH_PASSWORD="your-elasticsearch-password"

# Run installation script
./install-with-env.sh
```

**Or use Helm command directly:**
```bash
helm install mcall-crd mcall/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values values.yaml \
  --set logging.postgresql.password="$POSTGRES_PASSWORD" \
  --set logging.mysql.password="$MYSQL_PASSWORD" \
  --set logging.elasticsearch.password="$ELASTICSEARCH_PASSWORD"
```

#### Secret Structure

The chart creates the following secrets:
- `{release-name}-logging-secret`: Contains encrypted passwords for all logging backends

#### ConfigMap vs Secret

- **ConfigMap**: Non-sensitive configuration (hosts, ports, database names, etc.)
- **Secret**: Sensitive data (passwords, API keys, certificates)

## Configuration

The following table lists the configurable parameters and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `""` |
| `image.tag` | Image tag | `1.0.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `controller.replicas` | Number of replicas | `2` |
| `controller.resources.limits.cpu` | CPU limit | `200m` |
| `controller.resources.limits.memory` | Memory limit | `256Mi` |
| `controller.resources.requests.cpu` | CPU request | `100m` |
| `controller.resources.requests.memory` | Memory request | `128Mi` |
| `service.metrics.enabled` | Enable metrics service | `true` |
| `service.webhook.enabled` | Enable webhook service | `true` |
| `webhook.enabled` | Enable webhooks | `true` |
| `webhook.certManager.enabled` | Use cert-manager for certificates | `false` |
| `rbac.create` | Create RBAC resources | `true` |
| `crds.install` | Install CRDs | `true` |
| `namespace.create` | Create namespace | `true` |
| `namespace.name` | Namespace name | `mcall-system` |

## Examples

### Development Environment

```bash
helm install mcall-crd-dev ./helm/mcall-crd \
  --namespace mcall-dev \
  --create-namespace \
  --values ./helm/mcall-crd/values-dev.yaml
```

### Production Environment

```bash
# Install cert-manager first
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install mcall-crd
helm install mcall-crd-prod ./helm/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --values ./helm/mcall-crd/values.yaml \
  --wait \
  --timeout=10m
```

### Custom Configuration

```bash
helm install mcall-crd ./helm/mcall-crd \
  --namespace mcall-system \
  --create-namespace \
  --set controller.replicas=5 \
  --set controller.resources.limits.cpu=1000m \
  --set controller.resources.limits.memory=1Gi \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true
```

## Upgrading

```bash
# Upgrade to latest version
helm upgrade mcall-crd mcall/mcall-crd

# Upgrade with custom values
helm upgrade mcall-crd mcall/mcall-crd \
  --values values.yaml

# Upgrade specific image tag
helm upgrade mcall-crd mcall/mcall-crd \
  --set image.tag=1.1.0
```

## Uninstalling

### Automatic Cleanup

The chart includes a pre-delete hook that automatically removes finalizers from all CRD resources before uninstalling:

```bash
# Uninstall with automatic cleanup
helm uninstall mcall-crd
```

### Manual Cleanup

If you need to manually clean up resources:

```bash
# Remove finalizers from all mcalltasks
kubectl get mcalltasks -n mcall-system -o name | xargs -I {} kubectl patch {} -n mcall-system -p '{"metadata":{"finalizers":[]}}' --type=merge

# Remove finalizers from all mcallworkflows
kubectl get mcallworkflows -n mcall-system -o name | xargs -I {} kubectl patch {} -n mcall-system -p '{"metadata":{"finalizers":[]}}' --type=merge


# Force delete namespace if stuck in Terminating state
kubectl patch namespace mcall-system -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Cleanup Configuration

The cleanup feature can be configured in `values.yaml`:

```yaml
cleanup:
  # Enable/disable cleanup job
  enabled: true
  
  # Cleanup job image
  image:
    repository: bitnami/kubectl
    tag: "latest"
    pullPolicy: IfNotPresent
  
  # Resource limits for cleanup job
  resources:
    requests:
      memory: "64Mi"
      cpu: "100m"
    limits:
      memory: "128Mi"
      cpu: "200m"
```

## Monitoring

### Prometheus Integration

```yaml
serviceMonitor:
  enabled: true
  namespace: monitoring
  labels:
    app.kubernetes.io/name: mcall-crd
  interval: 30s
  timeout: 10s
  path: /metrics
```

### Health Checks

```bash
# Check pod health
kubectl get pods -n mcall-system -l app.kubernetes.io/name=mcall-crd

# Check service health
kubectl get svc -n mcall-system

# Check CRD status
kubectl get crd | grep mcall
```

## Troubleshooting

### Common Issues

1. **CRD Installation Failed**
   ```bash
   kubectl get crd | grep mcall
   kubectl describe crd mcalltasks.mcall.tz.io
   ```

2. **Webhook Certificate Issues**
   ```bash
   kubectl get secret mcall-crd-webhook-certs -n mcall-system
   kubectl describe validatingwebhookconfigurations
   ```

3. **Resource Constraints**
   ```bash
   kubectl top pods -n mcall-system
   kubectl describe pod -n mcall-system -l app.kubernetes.io/name=mcall-crd
   ```

### Debug Mode

```bash
# Enable debug logging
helm upgrade mcall-crd ./helm/mcall-crd \
  --namespace mcall-system \
  --set controller.env.DEBUG=true \
  --set controller.env.LOG_LEVEL=debug
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test the changes
5. Submit a pull request

## License

This chart is licensed under the MIT License.


