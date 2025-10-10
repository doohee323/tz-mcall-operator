# MCP Server Documentation

Documentation for using the McallOperator MCP Server with AI assistants.

## Quick Links

- **[Claude Desktop Setup](./CLAUDE_DESKTOP_SETUP.md)** - Configure Claude Desktop to use MCP server
- **[Cursor Setup](./CURSOR_SETUP.md)** - Configure Cursor IDE to use MCP server
- **[Usage Examples](./USAGE_EXAMPLES.md)** - Real-world usage examples

## Configuration Files

### Claude Desktop Configuration

**Example configuration file**: [claude_desktop_config.json](./claude_desktop_config.json)

**Location**:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

**Simple Configuration**:
```json
{
  "mcpServers": {
    "mcall-operator": {
      "command": "node",
      "args": ["/Users/dhong/workspaces/tz-mcall-operator/mcp-server/dist/index.js"],
      "env": {
        "SERVER_MODE": "stdio"
      }
    }
  }
}
```

**Advanced Configuration**: See [claude_desktop_config.example.json](./claude_desktop_config.example.json)

### Installation Steps

1. **Copy example configuration**:
   ```bash
   # macOS
   cp docs/claude_desktop_config.json ~/Library/Application\ Support/Claude/claude_desktop_config.json
   
   # Linux
   cp docs/claude_desktop_config.json ~/.config/Claude/claude_desktop_config.json
   ```

2. **Restart Claude Desktop**

3. **Verify connection**:
   Ask Claude: "Can you list all Kubernetes tasks?"

## Quick Start

### Test MCP Server

```bash
# Check server health
curl https://mcp-dev.drillquiz.com/health

# Get server info
curl https://mcp-dev.drillquiz.com/

# Expected response:
{
  "name": "McallOperator MCP Server",
  "version": "1.0.0",
  "tools": [...]
}
```

### First Task via AI

After configuring Claude or Cursor, try:

```
User: "Create a test task to run 'echo Hello MCP'"

AI: I'll create that task for you.
    [Calls create_mcall_task]
    ✅ Task created: test-task
```

## Available Tools

The MCP server provides 9 tools for managing Kubernetes tasks and workflows:

### Task Management (5 tools)
1. **create_mcall_task** - Create new tasks (cmd/get/post)
2. **get_mcall_task** - Get task details and status
3. **list_mcall_tasks** - List all tasks
4. **delete_mcall_task** - Delete a task
5. **get_mcall_task_logs** - View task execution logs

### Workflow Management (4 tools)
1. **create_mcall_workflow** - Create workflows with dependencies
2. **get_mcall_workflow** - Get workflow details
3. **list_mcall_workflows** - List all workflows
4. **delete_mcall_workflow** - Delete a workflow

## Server Endpoints

- **Root**: `https://mcp-dev.drillquiz.com/` - Server information
- **Health**: `https://mcp-dev.drillquiz.com/health` - Health check
- **Ready**: `https://mcp-dev.drillquiz.com/ready` - Readiness check
- **MCP**: `https://mcp-dev.drillquiz.com/mcp` - MCP protocol endpoint

## Deployment Status

### Current Deployment (mcall-dev)

```bash
# Check MCP server pods
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Check service
kubectl get svc -n mcall-dev | grep mcp

# Check ingress
kubectl get ingress -n mcall-dev
```

### Access Methods

**External (via Ingress)**:
```
https://mcp-dev.drillquiz.com
```

**Internal (within cluster)**:
```
http://tz-mcall-operator-dev-mcp-server.mcall-dev.svc.cluster.local
```

**Local (via port-forward)**:
```bash
kubectl port-forward -n mcall-dev svc/tz-mcall-operator-dev-mcp-server 3000:80
curl http://localhost:3000/health
```

## Example Use Cases

### Monitoring
```
"Monitor https://api.example.com every 5 minutes"
"Check database connectivity every 10 minutes"
"Alert if any service is down"
```

### Automation
```
"Backup database daily at 2 AM"
"Clean up old files every week"
"Restart services when needed"
```

### Workflows
```
"Create a deployment workflow with health checks and tests"
"Set up a data pipeline that runs nightly"
"Build a CI/CD verification workflow"
```

## Troubleshooting

### Connection Issues

```bash
# Verify server is running
kubectl get pods -n mcall-dev -l app.kubernetes.io/component=mcp-server

# Check server logs
kubectl logs -n mcall-dev -l app.kubernetes.io/component=mcp-server -f

# Test connectivity
curl https://mcp-dev.drillquiz.com/health
```

### Configuration Issues

1. Check JSON syntax in config file
2. Restart AI assistant after config changes
3. Verify URL is correct: `https://mcp-dev.drillquiz.com/mcp`

## Security

- MCP server uses Kubernetes RBAC for authorization
- Only has permissions for McallTask and McallWorkflow CRDs
- TLS/HTTPS enforced via nginx-ingress
- All operations audited via Kubernetes logs

## References

- [MCP Server Guide](../MCP_SERVER_GUIDE.md) - Complete guide
- [Helm Chart Guide](../helm/mcall-operator/README.md) - Deployment guide
- [Model Context Protocol](https://modelcontextprotocol.io) - Official MCP documentation

