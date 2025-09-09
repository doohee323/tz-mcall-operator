# Jenkins Credentials Setup Guide

## Required Credentials

The following credentials need to be configured in Jenkins:

### 1. Logging Backend Passwords

**Reuse existing credential**: `POSTGRES_PASSWORD`
- Already configured (reused from PostgreSQL logging backend)

**New credentials to add**:

**Credential ID**: `MYSQL_PASSWORD`
- **Type**: Secret text
- **Secret**: `your-mysql-password` (actual MySQL password)
- **Description**: MySQL logging backend password

**Credential ID**: `ELASTICSEARCH_PASSWORD`
- **Type**: Secret text
- **Secret**: `your-elasticsearch-password` (actual Elasticsearch password)
- **Description**: Elasticsearch logging backend password

### 2. Existing Credentials (may already be configured)

- `POSTGRES_PASSWORD` - Existing PostgreSQL password
- `DOCKERHUB_CREDENTIALS_ID` - Docker Hub authentication
- `KUBECONFIG_CREDENTIALS_ID` - Kubernetes configuration
- `GITHUP_TOKEN` - GitHub token
- `VAULT_TOKEN` - Vault token
- `ARGOCD_PASSWORD` - ArgoCD password
- `DOCKER_PASSWORD` - Docker password
- `GOOGLE_OAUTH_CLIENT_SECRET` - Google OAuth
- `MINIO_SECRET_KEY` - MinIO secret key
- `OPENAI_API_KEY` - OpenAI API key

## Jenkins Credentials Setup Method

### 1. Login as Jenkins Administrator
- Access Jenkins web interface
- Click "Manage Jenkins" → "Manage Credentials"

### 2. Add Credential
- Click "Add Credentials"
- Kind: Select "Secret text"
- Secret: Enter actual password
- ID: Enter the Credential ID specified above
- Description: Enter description
- Click "OK"

### 3. Verification
- Check newly added items in Credentials list
- Verify that ID matches exactly

## Environment-specific Configuration

### Development Environment (dev branch)
- `POSTGRES_PASSWORD`: Development PostgreSQL password
- `MYSQL_PASSWORD`: Development MySQL password (empty if not used)
- `ELASTICSEARCH_PASSWORD`: Development Elasticsearch password (empty if not used)

### Staging Environment (qa branch)
- `POSTGRES_PASSWORD`: Staging PostgreSQL password
- `MYSQL_PASSWORD`: Staging MySQL password
- `ELASTICSEARCH_PASSWORD`: Staging Elasticsearch password

### Production Environment (main branch)
- `POSTGRES_PASSWORD`: Production PostgreSQL password
- `MYSQL_PASSWORD`: Production MySQL password
- `ELASTICSEARCH_PASSWORD`: Production Elasticsearch password

## Testing

After configuring credentials, run Jenkins pipeline to verify:

1. **Check environment variables in build logs**:
   ```
   POSTGRES_PASSWORD=***
   MYSQL_PASSWORD=***
   ELASTICSEARCH_PASSWORD=***
   ```

2. **Verify password passing in Helm install command**:
   ```bash
   --set logging.postgresql.password="***" \
   --set logging.mysql.password="***" \
   --set logging.elasticsearch.password="***"
   ```

3. **Verify Kubernetes Secret creation**:
   ```bash
   kubectl get secret tz-mcall-crd-logging-secret -n mcall-system
   kubectl describe secret tz-mcall-crd-logging-secret -n mcall-system
   ```

## Troubleshooting

### Credential not found error
- Check ID in Jenkins Credentials list
- Verify exact case match
- Check if Credential is configured in correct Scope

### Password not being passed issue
- Check environment section in Jenkinsfile
- Check --set options in k8s.sh script
- Check environment variable values in build logs

### Secret not being created issue
- Check logging-secret.yaml template in Helm chart
- Check logging configuration in values.yaml
- Verify password is not empty