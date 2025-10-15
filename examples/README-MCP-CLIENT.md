# MCP Client Feature Guide

The `mcp-client` task type enables McallOperator to act as a client to other MCP (Model Context Protocol) servers. This allows you to orchestrate and automate interactions with Jenkins, GitHub Actions, or any other service that exposes an MCP server interface.

## Features

- ✅ Call MCP servers via JSON-RPC over HTTP/HTTPS
- ✅ Multiple authentication methods (API Key, Bearer Token, Basic Auth)
- ✅ Kubernetes Secret integration for secure credential management
- ✅ Task result passing between MCP calls
- ✅ Conditional execution based on MCP call results
- ✅ Custom headers support
- ✅ Configurable timeouts and retries

## Quick Start

### 1. Create Credentials Secret

```bash
kubectl create secret generic jenkins-mcp-credentials \
  --from-literal=api-key='your-api-key-here' \
  -n mcall-dev
```

### 2. Create a Simple MCP Task

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: my-mcp-call
  namespace: mcall-dev
spec:
  type: mcp-client
  timeout: 30
  
  mcpConfig:
    serverUrl: http://jenkins-mcp-server:3000/mcp
    toolName: list_mcall_tasks
    arguments:
      namespace: mcall-dev
    
    auth:
      type: apiKey
      secretRef:
        name: jenkins-mcp-credentials
      secretKey: api-key
```

### 3. Apply and Check Status

```bash
# Apply the task
kubectl apply -f my-mcp-task.yaml

# Check status
kubectl get mcalltask my-mcp-call -n mcall-dev

# View results
kubectl get mcalltask my-mcp-call -n mcall-dev -o jsonpath='{.status.result.output}' | jq .
```

## Configuration Reference

### MCPConfig Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serverUrl` | string | Yes* | MCP server endpoint URL |
| `toolName` | string | Yes | Name of the MCP tool to call |
| `arguments` | object | No | Arguments to pass to the tool |
| `auth` | object | No | Authentication configuration |
| `headers` | map[string]string | No | Additional HTTP headers |
| `connectionTimeout` | int32 | No | Connection timeout in seconds |

*Can use `spec.input` as fallback if not specified

### Authentication Types

#### 1. API Key Authentication

```yaml
auth:
  type: apiKey
  secretRef:
    name: my-credentials
    namespace: mcall-dev  # Optional, defaults to task namespace
  secretKey: api-key
  headerName: X-API-Key  # Optional, default is "X-API-Key"
```

#### 2. Bearer Token Authentication

```yaml
auth:
  type: bearer
  secretRef:
    name: my-token
  secretKey: token
```

Sends: `Authorization: Bearer <token>`

#### 3. Basic Authentication

```yaml
auth:
  type: basic
  secretRef:
    name: my-credentials
  usernameKey: username
  passwordKey: password
```

Sends: `Authorization: Basic <base64(username:password)>`

#### 4. No Authentication

```yaml
auth:
  type: none
```

## Examples

### Example 1: Trigger Jenkins Build

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: trigger-jenkins-build
  namespace: mcall-dev
spec:
  type: mcp-client
  timeout: 60
  
  mcpConfig:
    serverUrl: http://jenkins-mcp-server:3000/mcp
    toolName: create_mcall_task
    arguments:
      name: build-job
      type: cmd
      input: "jenkins-cli build MyProject --parameters branch=main"
      timeout: 300
    
    auth:
      type: apiKey
      secretRef:
        name: jenkins-mcp-credentials
      secretKey: api-key
```

### Example 2: Check Task Status with Result Passing

```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: build-and-check
  namespace: mcall-dev
spec:
  tasks:
  - name: trigger-build
    spec:
      type: mcp-client
      mcpConfig:
        serverUrl: http://jenkins-mcp-server:3000/mcp
        toolName: create_mcall_task
        arguments:
          name: build-job
          type: cmd
          input: "make build"
        auth:
          type: apiKey
          secretRef:
            name: jenkins-mcp-credentials
          secretKey: api-key
  
  - name: check-status
    dependencies:
      - trigger-build
    inputSources:
      - name: TASK_NAME
        taskRef: trigger-build
        field: output
        jsonPath: "$.metadata.name"
    spec:
      type: mcp-client
      mcpConfig:
        serverUrl: http://jenkins-mcp-server:3000/mcp
        toolName: get_mcall_task
        arguments:
          name: build-job  # Or use ${TASK_NAME} with inputTemplate
          namespace: mcall-dev
        auth:
          type: apiKey
          secretRef:
            name: jenkins-mcp-credentials
          secretKey: api-key
```

### Example 3: Using Environment Variables from Secrets

```yaml
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: mcp-with-secrets
  namespace: mcall-dev
spec:
  type: mcp-client
  
  # Inject secrets as environment variables
  secretRefs:
    - envVarName: WEBHOOK_URL
      secretRef:
        name: notification-secrets
      secretKey: slack-webhook-url
  
  mcpConfig:
    serverUrl: http://mcp-server:3000/mcp
    toolName: send_notification
    arguments:
      webhook: "${WEBHOOK_URL}"  # Will be replaced
      message: "Build completed"
    
    auth:
      type: apiKey
      secretRef:
        name: mcp-credentials
      secretKey: api-key
```

## Workflow Integration

MCP client tasks work seamlessly with workflows:

```yaml
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: jenkins-ci-pipeline
  namespace: mcall-dev
spec:
  schedule: '0 2 * * *'  # Daily at 2 AM
  tasks:
  
  - name: trigger-build
    spec:
      type: mcp-client
      mcpConfig:
        serverUrl: http://jenkins-mcp-server:3000/mcp
        toolName: create_mcall_task
        arguments:
          name: nightly-build
          type: cmd
          input: "make test && make build"
        auth:
          type: apiKey
          secretRef:
            name: jenkins-mcp-credentials
          secretKey: api-key
  
  - name: check-result
    dependencies:
      - trigger-build
    condition:
      dependentTask: trigger-build
      when: success
    spec:
      type: mcp-client
      mcpConfig:
        serverUrl: http://jenkins-mcp-server:3000/mcp
        toolName: get_mcall_task
        arguments:
          name: nightly-build
          namespace: mcall-dev
        auth:
          type: apiKey
          secretRef:
            name: jenkins-mcp-credentials
          secretKey: api-key
  
  - name: notify-success
    dependencies:
      - check-result
    condition:
      dependentTask: check-result
      when: success
    spec:
      type: cmd
      input: |
        curl -X POST ${SLACK_WEBHOOK} \
          -d '{"text": "✅ Nightly build succeeded!"}'
    secretRefs:
      - envVarName: SLACK_WEBHOOK
        secretRef:
          name: notification-secrets
        secretKey: slack-webhook-url
```

## Troubleshooting

### Common Issues

#### 1. Authentication Failed

**Error:** `MCP server returned error: 401` or `403`

**Solutions:**
- Verify secret exists: `kubectl get secret jenkins-mcp-credentials -n mcall-dev`
- Check secret content: `kubectl get secret jenkins-mcp-credentials -n mcall-dev -o yaml`
- Ensure correct secret key name is used
- Verify MCP server is configured to accept the credentials

#### 2. Connection Timeout

**Error:** `MCP request failed: context deadline exceeded`

**Solutions:**
- Increase timeout: `spec.timeout: 60`
- Add connection timeout: `mcpConfig.connectionTimeout: 30`
- Check network connectivity: `kubectl exec -it <pod> -- curl http://jenkins-mcp-server:3000/health`
- Verify MCP server is running: `kubectl get pods -n <mcp-namespace>`

#### 3. Invalid Arguments

**Error:** `MCP error -32602: Invalid params`

**Solutions:**
- Verify tool name is correct
- Check arguments format matches tool schema
- Use `kubectl logs` to see the actual request being sent
- Test with MCP server's API documentation

#### 4. Secret Not Found

**Error:** `failed to get auth secret: secrets "xxx" not found`

**Solutions:**
- Create the secret: `kubectl create secret generic ...`
- Check namespace: secret must be in same namespace or explicitly specified
- Verify RBAC permissions for the operator to read secrets

### Debug Mode

To enable detailed logging:

```bash
# View operator logs
kubectl logs -n mcall-system deployment/mcall-operator -f

# View task details
kubectl describe mcalltask my-mcp-call -n mcall-dev

# View raw status
kubectl get mcalltask my-mcp-call -n mcall-dev -o yaml
```

### Testing MCP Server Connection

```bash
# Manual test with curl
curl -X POST http://jenkins-mcp-server:3000/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "list_mcall_tasks",
      "arguments": {"namespace": "mcall-dev"}
    }
  }'
```

## Security Best Practices

1. **Always use Kubernetes Secrets** for credentials
   ```bash
   # Never hardcode credentials in YAML
   kubectl create secret generic my-creds --from-literal=api-key='xxx'
   ```

2. **Use RBAC to limit secret access**
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: secret-reader
   rules:
   - apiGroups: [""]
     resources: ["secrets"]
     resourceNames: ["jenkins-mcp-credentials"]
     verbs: ["get"]
   ```

3. **Rotate credentials regularly**
   ```bash
   kubectl create secret generic jenkins-mcp-credentials \
     --from-literal=api-key='new-key' \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

4. **Use namespace isolation**
   - Keep production and dev credentials in separate namespaces
   - Use NetworkPolicies to restrict MCP server access

5. **Enable TLS for external MCP servers**
   ```yaml
   mcpConfig:
     serverUrl: https://secure-mcp.example.com/mcp  # Use HTTPS
   ```

## Performance Tips

1. **Set appropriate timeouts**
   - Connection timeout: 10-30 seconds
   - Execution timeout: Based on expected operation duration

2. **Use retries for transient failures**
   ```yaml
   spec:
     retryCount: 3
     timeout: 30
   ```

3. **Batch operations when possible**
   - Use workflow parallelism for independent MCP calls
   - Combine multiple operations in a single tool call if supported

4. **Monitor MCP server health**
   - Add health check tasks before critical operations
   - Set up alerts for MCP server downtime

## Complete Example

See the following example files:
- `jenkins-mcp-workflow.yaml` - Full workflow example
- `jenkins-mcp-secrets.yaml` - Secret configurations
- `simple-mcp-client-example.yaml` - Simple usage examples

## API Reference

For complete API reference, see:
- `/Users/dhong/workspaces/tz-mcall-operator/api/v1/mcalltask_types.go`
- CRD: `/Users/dhong/workspaces/tz-mcall-operator/crds/mcall.tz.io_mcalltasks.yaml`

## Support

For issues and questions:
- GitHub Issues: https://github.com/doohee323/tz-mcall-operator/issues
- Documentation: `/Users/dhong/workspaces/tz-mcall-operator/README.md`







