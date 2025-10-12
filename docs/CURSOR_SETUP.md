# Cursor MCP Server Setup

## Configuration for Cursor IDE

Cursor IDE can connect to the MCP server to enable AI-assisted Kubernetes task management.

### 1. Configure MCP Server in Cursor

Open Cursor settings and configure the MCP server:

**Cursor Settings Path**:
- macOS: `~/Library/Application Support/Cursor/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`
- Or use Cursor's Settings UI

### 2. Add Server Configuration

Add the following to your MCP settings:

```json
{
  "mcpServers": {
    "mcall-operator": {
      "url": "https://mcp-dev.drillquiz.com/mcp",
      "transport": "sse",
      "description": "Kubernetes Task and Workflow Manager"
    }
  }
}
```

### 3. Restart Cursor

Restart Cursor IDE to load the new MCP server configuration.

## Using MCP Server in Cursor

### Natural Language Commands

Once configured, you can interact with your Kubernetes tasks using natural language in Cursor's AI chat:

#### Example 1: Create a Health Check

```
You: "Create a Kubernetes task to check https://api.example.com/health every 5 minutes"

AI: I'll create a health check task for you.
    
    [AI automatically calls create_mcall_task with:
     - name: api-health-check
     - type: get
     - input: https://api.example.com/health
     - schedule: */5 * * * *
    ]
    
    ✅ Created McallTask 'api-health-check' in mcall-dev namespace
```

#### Example 2: Create a Scheduled Command

```
You: "Run 'kubectl get pods' in dev namespace every 10 minutes"

AI: I'll create a scheduled task to list pods every 10 minutes.
    
    [AI calls create_mcall_task]
    
    ✅ Task created: pod-list-dev
```

#### Example 3: List All Tasks

```
You: "Show me all running tasks"

AI: Here are the tasks in your cluster:
    
    [AI calls list_mcall_tasks]
    
    Tasks in mcall-dev:
    1. api-health-check - Running (last: 2m ago)
    2. pod-list-dev - Running (last: 8m ago)
    3. database-backup - Succeeded (last: 4h ago)
```

#### Example 4: Create a Workflow

```
You: "Create a deployment workflow:
     1. Check API health
     2. Run tests if health is good
     3. Send Slack notification after tests"

AI: I'll create a multi-stage workflow with dependencies.
    
    [AI calls create_mcall_workflow]
    
    ✅ Workflow created: deployment-workflow
    - Task 1: check-health (no deps)
    - Task 2: run-tests (depends on: check-health)
    - Task 3: notify-slack (depends on: run-tests)
```

#### Example 5: Check Task Logs

```
You: "Show me the logs for the api-health-check task"

AI: Here are the execution logs:
    
    [AI calls get_mcall_task_logs]
    
    Task: api-health-check
    Phase: Succeeded
    Start Time: 2025-10-09 04:50:00
    Output: HTTP 200 OK - {"status":"healthy"}
```

## Practical Use Cases

### 1. Automated Monitoring

```
"Create tasks to monitor:
 - API health every 5 minutes
 - Database connectivity every 10 minutes
 - Disk space every hour"
```

### 2. Scheduled Maintenance

```
"Set up:
 - Daily backup at 2 AM
 - Weekly cleanup at Sunday midnight
 - Monthly report generation"
```

### 3. CI/CD Integration

```
"Create a deployment workflow:
 1. Run smoke tests
 2. Check service health
 3. Run integration tests
 4. Send notification to Slack"
```

### 4. Operations Tasks

```
"Create tasks to:
 - Restart pods when memory > 80%
 - Clear cache every 6 hours
 - Sync data at midnight"
```

## Task Types

### Command (cmd)
Execute shell commands in Kubernetes pods:
```
"Run 'echo Hello World'"
"Execute 'kubectl get pods'"
"Run database migration script"
```

### HTTP GET (get)
Make HTTP GET requests:
```
"Check health of https://api.example.com"
"Monitor endpoint https://status.example.com"
"Fetch data from API"
```

### HTTP POST (post)
Make HTTP POST requests:
```
"Send notification to Slack webhook"
"Post data to API endpoint"
"Trigger external service"
```

## Advanced Features

### Task Parameters

You can specify advanced parameters:

```
"Create a task with:
 - Name: complex-task
 - Command: long-running-script.sh
 - Timeout: 1800 seconds (30 minutes)
 - Retry: 3 times on failure
 - Environment: DEBUG=true, LOG_LEVEL=info"
```

### Workflow Dependencies

```
"Create a workflow where:
 - Task A runs first
 - Task B runs only if A succeeds
 - Task C runs only if B succeeds
 - Tasks D and E run in parallel after C"
```

### Scheduled Tasks

```
"Create a scheduled task:
 - Run daily at 2 AM
 - Run every 5 minutes
 - Run weekly on Monday at 9 AM
 - Run on the 1st of each month"
```

## Verification

### Check MCP Server Connection

In Cursor, ask:
```
"Can you list all Kubernetes tasks?"
```

If connected properly, AI will call `list_mcall_tasks` and show results.

### Check Available Tools

Ask:
```
"What Kubernetes operations can you help me with?"
```

AI should mention:
- Creating tasks (cmd, HTTP GET/POST)
- Scheduling recurring tasks
- Creating workflows with dependencies
- Monitoring task status
- Viewing task logs

## Troubleshooting

### MCP Server Not Available in Cursor

1. **Check configuration**:
   ```bash
   # Verify server is accessible
   curl https://mcp-dev.drillquiz.com/health
   ```

2. **Restart Cursor** after configuration changes

3. **Check Cursor logs** for MCP connection errors

### Tools Not Working

1. **Verify RBAC permissions**:
   ```bash
   kubectl auth can-i create mcalltasks \
     --as=system:serviceaccount:mcall-dev:tz-mcall-operator-dev-mcp-server
   ```

2. **Check server logs**:
   ```bash
   kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f
   ```

3. **Test server manually**:
   ```bash
   curl https://mcp-dev.drillquiz.com/
   ```

## Local Development Setup

For local development, use localhost:

```json
{
  "mcpServers": {
    "mcall-operator-local": {
      "command": "node",
      "args": ["./mcp-server/dist/index.js"],
      "env": {
        "SERVER_MODE": "stdio"
      }
    }
  }
}
```

Then run:
```bash
cd mcp-server
npm install
npm run build
```

## Security Considerations

- MCP Server uses Kubernetes RBAC for authorization
- Only has permissions for McallTask and McallWorkflow CRDs
- Cannot access other cluster resources
- All operations are audited via Kubernetes audit logs

## References

- [MCP Server Guide](../MCP_SERVER_GUIDE.md)
- [Model Context Protocol Docs](https://modelcontextprotocol.io)
- [Cursor Documentation](https://cursor.sh/docs)


