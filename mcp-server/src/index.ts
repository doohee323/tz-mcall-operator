#!/usr/bin/env node

// Choose server mode based on environment
const SERVER_MODE = process.env.SERVER_MODE || "http";

if (SERVER_MODE === "http") {
  // HTTP mode for Kubernetes deployment
  import("./http-server.js");
} else {
  // Stdio mode for local development/testing
  import("./stdio-server.js");
}

