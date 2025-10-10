# Claude Desktop MCP Server Setup

## Configuration

### 1. Find Claude Desktop Config File

**macOS**:
```bash
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows**:
```
%APPDATA%\Claude\claude_desktop_config.json
```

**Linux**:
```
~/.config/Claude/claude_desktop_config.json
```

### 2. Add MCP Server Configuration

Edit the config file and add:

```json
{
  "mcpServers": {
    "mcall-operator": {
      "command": "node",
      "args": ["/Users/dhong/workspaces/tz-mcall-operator/mcp-server/dist/index.js"],
      "env": {
        "SERVER_MODE": "stdio"
      },
      "description": "Kubernetes Task and Workflow Manager via tz-mcall-operator"
    }
  }
}
```

Or if you have multiple servers:

```json
{
  "mcpServers": {
    "mcall-operator": {
      "command": "node",
      "args": ["/Users/dhong/workspaces/tz-mcall-operator/mcp-server/dist/index.js"],
      "env": {
        "SERVER_MODE": "stdio"
      }
    },
    "other-server": {
      "command": "node",
      "args": ["/path/to/other-server.js"]
    }
  }
}
```

### 3. Restart Claude Desktop

Close and reopen Claude Desktop app.

### 4. Verify Connection

In Claude Desktop, you should see the MCP server tools available. Try asking:

```
"Show me all available Kubernetes tasks"
```

Claude will use the `list_mcall_tasks` tool to retrieve tasks from your cluster.

## Usage Examples

### Example 1: Create a Health Check Task

```
User: "Create a task to check the health of https://api.example.com every 5 minutes"

Claude: I'll create a health check task for you.
        [uses create_mcall_task tool]
        
        ✅ Task created: api-health-check
        - Type: GET
        - URL: https://api.example.com
        - Schedule: */5 * * * *
```

### Example 2: Create a Database Backup Task

```
User: "Set up a daily database backup at 2 AM"

Claude: I'll create a backup task scheduled for 2 AM daily.
        [uses create_mcall_task tool]
        
        ✅ Task created: database-backup
        - Type: cmd
        - Command: pg_dump mydb > /backup/backup_$(date +%Y%m%d).sql
        - Schedule: 0 2 * * *
```

### Example 3: Create a Deployment Workflow

```
User: "Create a workflow that checks health, runs tests, then sends a notification"

Claude: I'll create a deployment workflow with dependencies.
        [uses create_mcall_workflow tool]
        
        ✅ Workflow created: deployment-workflow
        Tasks:
        1. check-health (no dependencies)
        2. run-tests (depends on: check-health)
        3. send-notification (depends on: run-tests)
```

### Example 4: Monitor Tasks

```
User: "Show me the status of all tasks"

Claude: Here are all the tasks in mcall-dev namespace:
        [uses list_mcall_tasks tool]
        
        Tasks:
        1. api-health-check - Running (last run: 2 mins ago)
        2. database-backup - Succeeded (last run: 4 hours ago)
        3. cleanup-logs - Pending (scheduled: tonight)
```

### Example 5: Check Task Logs

```
User: "Show me the logs for api-health-check task"

Claude: Here are the execution logs:
        [uses get_mcall_task_logs tool]
        
        Task: api-health-check
        Phase: Succeeded
        Output: HTTP 200 OK
        Last execution: 2025-10-09 04:45:00
```

## Available Commands

You can ask Claude in natural language, and it will use the appropriate tools:

### Task Management
- "Create a task to..."
- "Show me all tasks"
- "Get the status of [task-name]"
- "Delete the [task-name] task"
- "Show me logs for [task-name]"

### Workflow Management
- "Create a workflow with..."
- "Show me all workflows"
- "Get workflow status for [workflow-name]"
- "Delete workflow [workflow-name]"

### Examples of Natural Language Commands

```
✅ "Create a task to ping google.com every minute"
✅ "Set up a health check for https://api.example.com"
✅ "Run kubectl get pods in dev namespace every 10 minutes"
✅ "Create a daily backup task"
✅ "Show me all running tasks"
✅ "What tasks failed recently?"
✅ "Create a deployment workflow"
✅ "Delete the old-backup task"
```

## Troubleshooting

### MCP Server Not Showing in Claude

1. Check config file syntax (must be valid JSON)
2. Restart Claude Desktop
3. Check Claude Desktop logs

### Connection Errors

1. Verify server is accessible:
   ```bash
   curl https://mcp-dev.drillquiz.com/health
   ```

2. Check if server is running:
   ```bash
   kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server
   ```

3. Check ingress:
   ```bash
   kubectl get ingress -n mcall-dev
   ```

### Tools Not Working

1. Check Kubernetes permissions (RBAC)
2. Verify namespace exists
3. Check MCP server logs:
   ```bash
   kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f
   ```

## Security Notes

- MCP Server uses Kubernetes ServiceAccount for authentication
- Only has permissions to manage McallTasks and McallWorkflows
- Follows RBAC least privilege principle
- TLS/HTTPS enforced via Ingress

## Next Steps

1. Configure Claude Desktop with the MCP server
2. Restart Claude Desktop
3. Try natural language commands to manage tasks
4. Check [MCP_SERVER_GUIDE.md](../MCP_SERVER_GUIDE.md) for more details

