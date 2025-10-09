import express from "express";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { KubernetesClient } from "./kubernetes-client.js";
import { setupToolHandlers, TOOLS } from "./tools.js";

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
  
  // For SSE transport, POST messages are handled through the GET endpoint
  // This endpoint can be used for health checks or other purposes
  res.status(200).json({
    status: "ok",
    message: "Use GET /mcp for SSE connection"
  });
});

// Start server
app.listen(PORT, () => {
  console.log(`MCP Server listening on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`Ready check: http://localhost:${PORT}/ready`);
  console.log(`MCP endpoint: http://localhost:${PORT}/mcp`);
});

export default app;

