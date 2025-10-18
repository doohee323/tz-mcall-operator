import express from "express";
import { createServer } from "http";
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
  DeleteWorkflowSchema,
  TriggerBuildSchema
} from "./tools.js";
import dagApiRouter from "./dag-api.js";
import { setupWebSocket } from "./dag-websocket.js";
import { AuthService } from "./auth.js";

const app = express();
const httpServer = createServer(app);
const PORT = process.env.PORT || 3000;

app.use(express.json());

// CORS for development
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-API-Key');
  if (req.method === 'OPTIONS') {
    return res.sendStatus(200);
  }
  next();
});

// Initialize authentication service
const authService = new AuthService();

// Serve static files from UI build (if exists) - no auth required
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
app.use(express.static(join(__dirname, '../ui/dist')));

// DAG API routes - auth required
app.use('/api', authService.authenticate(), dagApiRouter);

// Setup WebSocket
setupWebSocket(httpServer);

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
    authRequired: process.env.MCP_REQUIRE_AUTH === 'true',
    endpoints: {
      health: "/health",
      ready: "/ready",
      mcp: "/mcp (requires API key if auth enabled)",
      api: "/api/* (requires API key if auth enabled)"
    },
    tools: TOOLS.map((tool) => ({
      name: tool.name,
      description: tool.description,
    })),
  });
});

// MCP endpoint using SSE - auth required
app.get("/mcp", authService.authenticate(), async (req, res) => {
  console.log("New MCP client connected via GET");

  // Check if this is a streamable HTTP request (mcp-proxy)
  const acceptHeader = req.headers.accept || '';
  if (acceptHeader.includes('text/event-stream')) {
    console.log("SSE connection detected");
    
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
  } else {
    // For mcp-proxy streamable HTTP, return server info
    console.log("Streamable HTTP connection detected");
    res.status(200).json({
      jsonrpc: "2.0",
      result: {
        serverInfo: {
          name: "mcall-operator-mcp-server",
          version: "1.0.0"
        },
        capabilities: {
          tools: {}
        }
      }
    });
  }
});

// Handle DELETE requests for session cleanup
app.delete("/mcp", authService.authenticate(), async (req, res) => {
  console.log("MCP DELETE request received (session cleanup)");
  res.status(200).json({ status: "ok", message: "Session cleaned up" });
});

app.post("/mcp", authService.authenticate(), async (req, res) => {
  console.log("MCP POST message received");
  
  try {
    // Handle initialize request
    if (req.body.method === "initialize") {
      console.log("Handling initialize request");
      
      // Generate session ID
      const sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      
      // Set session ID in response header
      res.setHeader('mcp-session-id', sessionId);
      
      // Send proper JSON-RPC initialize response
      res.status(200).json({
        jsonrpc: "2.0",
        id: req.body.id,
        result: {
          protocolVersion: "2024-11-05",
          capabilities: {
            tools: {}
          },
          serverInfo: {
            name: "mcall-operator-mcp-server",
            version: "1.0.0"
          }
        }
      });
      return;
    }
    
    // Handle notifications/initialized
    if (req.body.method === "notifications/initialized") {
      console.log("Handling notifications/initialized");
      // For notifications, we don't send a response (it's a one-way message)
      // But mcp-proxy expects a JSON response, so send empty object
      res.status(200).json({});
      return;
    }
    
    // Handle tools/list request
    if (req.body.method === "tools/list") {
      console.log("Handling tools/list request");
      
      res.status(200).json({
        jsonrpc: "2.0",
        id: req.body.id,
        result: {
          tools: TOOLS.map(tool => ({
            name: tool.name,
            description: tool.description,
            inputSchema: tool.inputSchema
          }))
        }
      });
      return;
    }
    
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
        case "triggerBuild": {
          const params = TriggerBuildSchema.parse(toolArgs);
          result = await k8sClient.triggerJenkinsBuild(params.jobFullName, params.parameters);
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
httpServer.listen(Number(PORT), '0.0.0.0', () => {
  console.log(`MCP Server listening on port ${PORT} (IPv4/IPv6)`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`Ready check: http://localhost:${PORT}/ready`);
  console.log(`MCP endpoint: http://localhost:${PORT}/mcp`);
  console.log(`DAG API: http://localhost:${PORT}/api/workflows/:namespace/:name/dag`);
  console.log(`WebSocket: ws://localhost:${PORT}/socket.io/`);
});

export default app;

