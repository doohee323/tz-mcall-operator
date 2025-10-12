import { useState } from 'react';

interface ToolSchema {
  name: string;
  description: string;
  inputSchema: {
    type: string;
    properties: Record<string, any>;
    required?: string[];
  };
}

const MCP_TOOLS: ToolSchema[] = [
  {
    name: "create_mcall_task",
    description: "Create a new McallTask to execute commands or HTTP requests in Kubernetes. Examples: run shell commands, make HTTP health checks, schedule periodic tasks.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Task name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
        type: { type: "string", enum: ["cmd", "get", "post"], description: "Task type" },
        input: { type: "string", description: "Command or URL to execute" },
        timeout: { type: "number", description: "Timeout in seconds" },
        retryCount: { type: "number", description: "Number of retries" },
        schedule: { type: "string", description: "Cron schedule (e.g., '*/5 * * * *')" },
        environment: { type: "object", description: "Environment variables" },
      },
      required: ["name", "type", "input"],
    },
  },
  {
    name: "get_mcall_task",
    description: "Get details and status of a specific McallTask",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Task name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
      required: ["name"],
    },
  },
  {
    name: "list_mcall_tasks",
    description: "List all McallTasks in a namespace with optional label filtering",
    inputSchema: {
      type: "object",
      properties: {
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
        labelSelector: { type: "string", description: "Label selector (e.g., 'app=myapp')" },
      },
    },
  },
  {
    name: "delete_mcall_task",
    description: "Delete a McallTask from Kubernetes",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Task name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
      required: ["name"],
    },
  },
  {
    name: "get_mcall_task_logs",
    description: "Get execution logs and output from a McallTask",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Task name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
      required: ["name"],
    },
  },
  {
    name: "create_mcall_workflow",
    description: "Create a McallWorkflow with multiple tasks and dependencies. Use this to chain tasks together with execution order and dependencies.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Workflow name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
        tasks: {
          type: "array",
          description: "List of tasks in workflow",
          items: {
            type: "object",
            properties: {
              name: { type: "string", description: "Task name" },
              type: { type: "string", enum: ["cmd", "get", "post"] },
              input: { type: "string", description: "Command or URL" },
              dependencies: { type: "array", items: { type: "string" } },
            },
          },
        },
        schedule: { type: "string", description: "Cron schedule" },
        timeout: { type: "number", description: "Overall timeout in seconds" },
      },
      required: ["name", "tasks"],
    },
  },
  {
    name: "get_mcall_workflow",
    description: "Get details and status of a specific McallWorkflow",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Workflow name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
      required: ["name"],
    },
  },
  {
    name: "list_mcall_workflows",
    description: "List all McallWorkflows in a namespace",
    inputSchema: {
      type: "object",
      properties: {
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
    },
  },
  {
    name: "delete_mcall_workflow",
    description: "Delete a McallWorkflow from Kubernetes",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Workflow name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
      },
      required: ["name"],
    },
  },
];

export function McpToolsDoc() {
  const [selectedTool, setSelectedTool] = useState<ToolSchema>(MCP_TOOLS[0]);

  const generateMcpExample = (tool: ToolSchema): string => {
    const args: Record<string, any> = {};
    
    // Generate example arguments
    Object.entries(tool.inputSchema.properties).forEach(([key, prop]) => {
      if (tool.inputSchema.required?.includes(key)) {
        if (key === 'name') args[key] = 'example-task';
        else if (key === 'type') args[key] = 'get';
        else if (key === 'input') args[key] = prop.enum ? prop.enum[0] : 'https://example.com';
        else if (key === 'tasks') args[key] = [{ name: 'task1', type: 'cmd', input: 'echo hello' }];
        else args[key] = `example-${key}`;
      }
    });
    
    // Add common optional params
    if (tool.inputSchema.properties.namespace) {
      args.namespace = 'mcall-dev';
    }
    
    return JSON.stringify({ name: tool.name, arguments: args }, null, 2);
  };

  const generateKubectlExample = (tool: ToolSchema): string => {
    if (tool.name === 'create_mcall_task') {
      return `kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallTask
metadata:
  name: example-task
  namespace: mcall-dev
spec:
  type: get
  input: https://example.com
  timeout: 30
  retryCount: 3
EOF`;
    } else if (tool.name === 'create_mcall_workflow') {
      return `kubectl apply -f - <<EOF
apiVersion: mcall.tz.io/v1
kind: McallWorkflow
metadata:
  name: example-workflow
  namespace: mcall-dev
spec:
  schedule: '*/5 * * * *'
  tasks:
    - name: task1
      taskRef:
        name: task1-template
        namespace: mcall-dev
    - name: task2
      taskRef:
        name: task2-template
        namespace: mcall-dev
      dependencies:
        - task1
EOF`;
    } else if (tool.name.startsWith('get_')) {
      const resource = tool.name.includes('workflow') ? 'mcallworkflow' : 'mcalltask';
      return `kubectl get ${resource} example-name -n mcall-dev -o yaml`;
    } else if (tool.name.startsWith('list_')) {
      const resource = tool.name.includes('workflow') ? 'mcallworkflows' : 'mcalltasks';
      return `kubectl get ${resource} -n mcall-dev`;
    } else if (tool.name.startsWith('delete_')) {
      const resource = tool.name.includes('workflow') ? 'mcallworkflow' : 'mcalltask';
      return `kubectl delete ${resource} example-name -n mcall-dev`;
    } else if (tool.name === 'get_mcall_task_logs') {
      return `kubectl get mcalltask example-task -n mcall-dev -o jsonpath='{.status.result.output}'`;
    }
    return '';
  };

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <div style={{
        padding: '15px 20px',
        background: '#1976d2',
        color: '#fff'
      }}>
        <h1 style={{ margin: 0, fontSize: '24px' }}>
          📚 MCP Tools Documentation
        </h1>
        <div style={{ fontSize: '14px', marginTop: '5px', opacity: 0.9 }}>
          Model Context Protocol tools for AI integration (Claude Desktop, ChatGPT, etc.)
        </div>
      </div>

      {/* Main Content */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left Panel - Tools List */}
        <div style={{
          width: '300px',
          borderRight: '1px solid #ddd',
          overflow: 'auto',
          background: '#f8f9fa'
        }}>
          <div style={{ padding: '15px' }}>
            <h3 style={{ margin: '0 0 15px 0', fontSize: '16px' }}>Available Tools ({MCP_TOOLS.length})</h3>
            
            {/* Task Tools */}
            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ fontSize: '13px', color: '#666', marginBottom: '8px', textTransform: 'uppercase' }}>
                📋 Task Management
              </h4>
              {MCP_TOOLS.filter(t => t.name.includes('task')).map((tool) => (
                <div
                  key={tool.name}
                  onClick={() => setSelectedTool(tool)}
                  style={{
                    padding: '10px',
                    marginBottom: '6px',
                    background: selectedTool === tool ? '#1976d2' : '#fff',
                    color: selectedTool === tool ? '#fff' : '#333',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    border: '1px solid #ddd',
                    fontSize: '13px',
                    transition: 'all 0.2s'
                  }}
                >
                  {tool.name.replace('_', ' ').replace('mcall ', '')}
                </div>
              ))}
            </div>

            {/* Workflow Tools */}
            <div>
              <h4 style={{ fontSize: '13px', color: '#666', marginBottom: '8px', textTransform: 'uppercase' }}>
                🔄 Workflow Management
              </h4>
              {MCP_TOOLS.filter(t => t.name.includes('workflow')).map((tool) => (
                <div
                  key={tool.name}
                  onClick={() => setSelectedTool(tool)}
                  style={{
                    padding: '10px',
                    marginBottom: '6px',
                    background: selectedTool === tool ? '#1976d2' : '#fff',
                    color: selectedTool === tool ? '#fff' : '#333',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    border: '1px solid #ddd',
                    fontSize: '13px',
                    transition: 'all 0.2s'
                  }}
                >
                  {tool.name.replace('_', ' ').replace('mcall ', '')}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Right Panel - Tool Documentation */}
        <div style={{ flex: 1, overflow: 'auto', padding: '20px' }}>
          <h2 style={{ marginTop: 0, marginBottom: '10px', color: '#1976d2' }}>
            {selectedTool.name}
          </h2>
          <p style={{ color: '#666', fontSize: '14px', lineHeight: '1.6', marginBottom: '25px' }}>
            {selectedTool.description}
          </p>

          {/* Parameters Section */}
          <div style={{ marginBottom: '30px' }}>
            <h3 style={{ fontSize: '18px', marginBottom: '15px', color: '#333' }}>Parameters</h3>
            
            {/* Required Parameters */}
            {selectedTool.inputSchema.required && selectedTool.inputSchema.required.length > 0 && (
              <div style={{ marginBottom: '20px' }}>
                <h4 style={{ fontSize: '14px', color: '#f44336', marginBottom: '10px' }}>
                  Required Parameters
                </h4>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                  <thead>
                    <tr style={{ background: '#f5f5f5', borderBottom: '2px solid #ddd' }}>
                      <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Name</th>
                      <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Type</th>
                      <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {selectedTool.inputSchema.required.map((paramName) => {
                      const param = selectedTool.inputSchema.properties[paramName];
                      return (
                        <tr key={paramName} style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '10px', fontWeight: '500', color: '#f44336' }}>
                            {paramName}*
                          </td>
                          <td style={{ padding: '10px' }}>
                            <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: '3px' }}>
                              {param.enum ? param.enum.join(' | ') : param.type}
                            </code>
                          </td>
                          <td style={{ padding: '10px', color: '#666' }}>{param.description}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {/* Optional Parameters */}
            <div>
              <h4 style={{ fontSize: '14px', color: '#2196f3', marginBottom: '10px' }}>
                Optional Parameters
              </h4>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr style={{ background: '#f5f5f5', borderBottom: '2px solid #ddd' }}>
                    <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Name</th>
                    <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Type</th>
                    <th style={{ padding: '10px', textAlign: 'left', fontWeight: '600' }}>Description</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(selectedTool.inputSchema.properties)
                    .filter(([name]) => !selectedTool.inputSchema.required?.includes(name))
                    .map(([paramName, param]) => (
                      <tr key={paramName} style={{ borderBottom: '1px solid #eee' }}>
                        <td style={{ padding: '10px', fontWeight: '500' }}>{paramName}</td>
                        <td style={{ padding: '10px' }}>
                          <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: '3px' }}>
                            {param.enum ? param.enum.join(' | ') : param.type}
                          </code>
                        </td>
                        <td style={{ padding: '10px', color: '#666' }}>{param.description}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* MCP Usage Example */}
          <div style={{ marginBottom: '30px' }}>
            <h3 style={{ fontSize: '18px', marginBottom: '15px', color: '#333' }}>
              💬 MCP Usage (Claude Desktop, ChatGPT)
            </h3>
            <p style={{ fontSize: '13px', color: '#666', marginBottom: '10px' }}>
              Example MCP tool call for AI assistants:
            </p>
            <div style={{
              background: '#2d2d2d',
              color: '#f8f8f2',
              padding: '15px',
              borderRadius: '6px',
              overflow: 'auto',
              position: 'relative'
            }}>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(generateMcpExample(selectedTool));
                }}
                style={{
                  position: 'absolute',
                  top: '10px',
                  right: '10px',
                  padding: '4px 10px',
                  background: '#4caf50',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '4px',
                  fontSize: '11px',
                  cursor: 'pointer'
                }}
              >
                📋 Copy
              </button>
              <pre style={{
                margin: 0,
                fontSize: '12px',
                lineHeight: '1.6',
                fontFamily: 'Monaco, Consolas, monospace',
                textAlign: 'left'
              }}>
                {generateMcpExample(selectedTool)}
              </pre>
            </div>
          </div>

          {/* kubectl Example */}
          <div style={{ marginBottom: '30px' }}>
            <h3 style={{ fontSize: '18px', marginBottom: '15px', color: '#333' }}>
              ⚙️ Equivalent kubectl Command
            </h3>
            <p style={{ fontSize: '13px', color: '#666', marginBottom: '10px' }}>
              Direct Kubernetes command (without MCP):
            </p>
            <div style={{
              background: '#2d2d2d',
              color: '#f8f8f2',
              padding: '15px',
              borderRadius: '6px',
              overflow: 'auto',
              position: 'relative'
            }}>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(generateKubectlExample(selectedTool));
                }}
                style={{
                  position: 'absolute',
                  top: '10px',
                  right: '10px',
                  padding: '4px 10px',
                  background: '#4caf50',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '4px',
                  fontSize: '11px',
                  cursor: 'pointer'
                }}
              >
                📋 Copy
              </button>
            <pre style={{
              margin: 0,
              fontSize: '12px',
              lineHeight: '1.6',
              fontFamily: 'Monaco, Consolas, monospace',
              textAlign: 'left'
            }}>
              {generateKubectlExample(selectedTool)}
            </pre>
            </div>
          </div>

          {/* Usage Guide */}
          <div style={{
            background: '#e3f2fd',
            border: '1px solid #90caf9',
            borderRadius: '6px',
            padding: '15px',
            marginBottom: '20px'
          }}>
            <h4 style={{ margin: '0 0 10px 0', fontSize: '14px', color: '#1565c0' }}>
              💡 How to Use MCP Tools
            </h4>
            <ol style={{ margin: 0, paddingLeft: '20px', fontSize: '13px', lineHeight: '1.8', color: '#333' }}>
              <li>
                <strong>Claude Desktop</strong>: Configure MCP server in <code>claude_desktop_config.json</code>
                <br />
                <code style={{ fontSize: '11px', background: 'rgba(0,0,0,0.05)', padding: '2px 4px', borderRadius: '3px' }}>
                  {"{ mcpServers: { mcall: { url: \"https://mcp-dev.drillquiz.com/mcp\" } } }"}
                </code>
              </li>
              <li>
                <strong>ChatGPT (with MCP plugin)</strong>: Add server URL and use tools in conversation
              </li>
              <li>
                <strong>Custom Integration</strong>: Use MCP SDK to connect to <code>/mcp</code> endpoint via SSE
              </li>
            </ol>
          </div>

          {/* MCP Server Info */}
          <div style={{
            background: '#f5f5f5',
            border: '1px solid #ddd',
            borderRadius: '6px',
            padding: '15px'
          }}>
            <h4 style={{ margin: '0 0 10px 0', fontSize: '14px', color: '#666' }}>
              🔗 MCP Server Endpoints
            </h4>
            <table style={{ fontSize: '13px', width: '100%' }}>
              <tbody>
                <tr>
                  <td style={{ padding: '5px 10px 5px 0', fontWeight: '500' }}>MCP Endpoint:</td>
                  <td style={{ padding: '5px 0' }}>
                    <code style={{ background: '#fff', padding: '4px 8px', borderRadius: '3px', border: '1px solid #ddd' }}>
                      https://mcp-dev.drillquiz.com/mcp
                    </code>
                  </td>
                </tr>
                <tr>
                  <td style={{ padding: '5px 10px 5px 0', fontWeight: '500' }}>Protocol:</td>
                  <td style={{ padding: '5px 0' }}>SSE (Server-Sent Events)</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}

