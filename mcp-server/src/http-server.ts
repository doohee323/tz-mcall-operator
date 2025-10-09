import express from "express";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { KubernetesClient } from "./kubernetes-client.js";
import { 
  setupToolHandlers, 
  TOOLS,
  CreateTaskSchema,
  GetTaskSchema,
  ListTasksSchema,
  DeleteTaskSchema,
  GetTaskLogsSchema,
  CreateWorkflowSchema,
  GetWorkflowSchema,
  ListWorkflowsSchema,
  DeleteWorkflowSchema
} from "./tools.js";

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());

// Health check endpoint
app.get("/health", (req, res) => {
  res.status(200).json({ status: "healthy" });
});

// Ready check endpoint
app.get("/ready", (req, res) => {
  res.status(200).json({ status: "ready" });
});

// Info endpoint
app.get("/", (req, res) => {
  res.json({
    name: "McallOperator MCP Server",
    version: "1.0.0",
    description: "Model Context Protocol server for tz-mcall-operator",
    endpoints: {
      health: "/health",
      ready: "/ready",
      mcp: "/mcp",
    },
    tools: TOOLS.map((tool) => ({
      name: tool.name,
      description: tool.description,
    })),
  });
});

// MCP endpoint using SSE
app.get("/mcp", async (req, res) => {
  console.log("New MCP client connected");

  const k8sClient = new KubernetesClient();
  const server = new Server(
    {
      name: "mcall-operator-mcp-server",
      version: "1.0.0",
    },
    {
      capabilities: {
        tools: {},
      },
    }
  );

  // Set up tool handlers
  setupToolHandlers(server, k8sClient);

  const transport = new SSEServerTransport("/mcp", res);
  await server.connect(transport);

  req.on("close", () => {
    console.log("MCP client disconnected");
  });
});

app.post("/mcp", async (req, res) => {
  console.log("MCP POST message received");
  
  try {
    // Check if this is a tool call request
    if (req.body.method === "tools/call") {
      const k8sClient = new KubernetesClient();
      const server = new Server(
        {
          name: "mcall-operator-mcp-server",
          version: "1.0.0",
        },
        {
          capabilities: {
            tools: {},
          },
        }
      );

      // Set up tool handlers
      setupToolHandlers(server, k8sClient);

      // Create a mock transport for direct HTTP calls
      const mockTransport = {
        send: (message: any) => {
          // Store the response to send back
          res.status(200).json(message);
        },
        start: () => Promise.resolve(),
        close: () => Promise.resolve(),
      };

      // Handle the tool call directly
      const toolName = req.body.params?.name;
      const toolArgs = req.body.params?.arguments || {};
      
      let result;
      
      switch (toolName) {
        case "create_mcall_task": {
          const params = CreateTaskSchema.parse(toolArgs);
          result = await k8sClient.createTask(params);
          break;
        }
        case "get_mcall_task": {
          const params = GetTaskSchema.parse(toolArgs);
          result = await k8sClient.getTask(params.name, params.namespace);
          break;
        }
        case "list_mcall_tasks": {
          const params = ListTasksSchema.parse(toolArgs);
          result = await k8sClient.listTasks(params.namespace, params.labelSelector);
          break;
        }
        case "delete_mcall_task": {
          const params = DeleteTaskSchema.parse(toolArgs);
          result = await k8sClient.deleteTask(params.name, params.namespace);
          break;
        }
        case "get_mcall_task_logs": {
          const params = GetTaskLogsSchema.parse(toolArgs);
          result = await k8sClient.getTaskLogs(params.name, params.namespace);
          break;
        }
        case "create_mcall_workflow": {
          const params = CreateWorkflowSchema.parse(toolArgs);
          result = await k8sClient.createWorkflow(params);
          break;
        }
        case "get_mcall_workflow": {
          const params = GetWorkflowSchema.parse(toolArgs);
          result = await k8sClient.getWorkflow(params.name, params.namespace);
          break;
        }
        case "list_mcall_workflows": {
          const params = ListWorkflowsSchema.parse(toolArgs);
          result = await k8sClient.listWorkflows(params.namespace);
          break;
        }
        case "delete_mcall_workflow": {
          const params = DeleteWorkflowSchema.parse(toolArgs);
          result = await k8sClient.deleteWorkflow(params.name, params.namespace);
          break;
        }
        default:
          return res.status(400).json({
            jsonrpc: "2.0",
            id: req.body.id,
            error: {
              code: -32601,
              message: `Tool '${toolName}' not found`
            }
          });
      }
      
      res.status(200).json({
        jsonrpc: "2.0",
        id: req.body.id,
        result: {
          content: [
            {
              type: "text",
              text: JSON.stringify(result, null, 2)
            }
          ]
        }
      });
    } else {
      // For SSE transport, POST messages are handled through the GET endpoint
      res.status(200).json({
        status: "ok",
        message: "Use GET /mcp for SSE connection or POST with tools/call method for direct calls"
      });
    }
  } catch (error) {
    console.error("Error processing POST request:", error);
    res.status(500).json({
      jsonrpc: "2.0",
      id: req.body?.id,
      error: {
        code: -32603,
        message: "Internal error",
        data: error instanceof Error ? error.message : String(error)
      }
    });
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`MCP Server listening on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`Ready check: http://localhost:${PORT}/ready`);
  console.log(`MCP endpoint: http://localhost:${PORT}/mcp`);
});

export default app;

