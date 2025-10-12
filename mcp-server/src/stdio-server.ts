#!/usr/bin/env node

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { KubernetesClient } from "./kubernetes-client.js";
import { setupToolHandlers } from "./tools.js";

// Initialize Kubernetes client
const k8sClient = new KubernetesClient();

// Create MCP server
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

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("McallOperator MCP Server running on stdio");
}

main().catch((error) => {
  console.error("Fatal error in main():", error);
  process.exit(1);
});

