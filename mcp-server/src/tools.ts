import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  Tool,
} from "@modelcontextprotocol/sdk/types.js";
import { KubernetesClient } from "./kubernetes-client.js";
import { z } from "zod";

// Define tool schemas
export const CreateTaskSchema = z.object({
  name: z.string().describe("Task name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
  type: z.enum(["cmd", "get", "post"]).describe("Task type: cmd (command), get (HTTP GET), post (HTTP POST)"),
  input: z.string().describe("Command or URL to execute"),
  timeout: z.number().optional().describe("Timeout in seconds"),
  retryCount: z.number().optional().describe("Number of retries on failure"),
  schedule: z.string().optional().describe("Cron schedule for recurring tasks (e.g., '*/5 * * * *')"),
  environment: z.record(z.string()).optional().describe("Environment variables"),
});

export const GetTaskSchema = z.object({
  name: z.string().describe("Task name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const ListTasksSchema = z.object({
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
  labelSelector: z.string().optional().describe("Label selector for filtering tasks"),
});

export const DeleteTaskSchema = z.object({
  name: z.string().describe("Task name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const GetTaskLogsSchema = z.object({
  name: z.string().describe("Task name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const CreateWorkflowSchema = z.object({
  name: z.string().describe("Workflow name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
  tasks: z.array(z.object({
    name: z.string().describe("Task name in workflow"),
    type: z.enum(["cmd", "get", "post"]).describe("Task type"),
    input: z.string().describe("Command or URL"),
    dependencies: z.array(z.string()).optional().describe("Task dependencies (task names)"),
  })).describe("List of tasks in workflow"),
  schedule: z.string().optional().describe("Cron schedule for workflow"),
  timeout: z.number().optional().describe("Overall workflow timeout in seconds"),
});

export const GetWorkflowSchema = z.object({
  name: z.string().describe("Workflow name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const ListWorkflowsSchema = z.object({
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const DeleteWorkflowSchema = z.object({
  name: z.string().describe("Workflow name"),
  namespace: z.string().optional().describe("Kubernetes namespace (default: mcall-system)"),
});

export const TriggerBuildSchema = z.object({
  jobFullName: z.string().describe("Full name of the Jenkins job (e.g., 'docker-test')"),
  parameters: z.record(z.string()).optional().describe("Build parameters as key-value pairs"),
});

// Define available tools
export const TOOLS: Tool[] = [
  {
    name: "create_mcall_task",
    description: "Create a new McallTask to execute commands or HTTP requests in Kubernetes. " +
      "Examples: run shell commands, make HTTP health checks, schedule periodic tasks.",
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
    description: "Create a McallWorkflow with multiple tasks and dependencies. " +
      "Use this to chain tasks together with execution order and dependencies.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Workflow name" },
        namespace: { type: "string", description: "Kubernetes namespace (default: mcall-system)" },
        tasks: {
          type: "array",
          items: {
            type: "object",
            properties: {
              name: { type: "string", description: "Task name" },
              type: { type: "string", enum: ["cmd", "get", "post"] },
              input: { type: "string", description: "Command or URL" },
              dependencies: { type: "array", items: { type: "string" } },
            },
            required: ["name", "type", "input"],
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
  {
    name: "triggerBuild",
    description: "Trigger a Jenkins build and return detailed status information",
    inputSchema: {
      type: "object",
      properties: {
        jobFullName: { type: "string", description: "Full name of the Jenkins job (e.g., 'docker-test')" },
        parameters: { type: "object", description: "Build parameters as key-value pairs" },
      },
      required: ["jobFullName"],
    },
  },
];

export function setupToolHandlers(server: Server, k8sClient: KubernetesClient) {
  // Handle list_tools request
  server.setRequestHandler(ListToolsRequestSchema, async () => ({
    tools: TOOLS,
  }));

  // Handle call_tool request
  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;

    try {
      switch (name) {
        case "create_mcall_task": {
          const params = CreateTaskSchema.parse(args);
          const result = await k8sClient.createTask(params);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "get_mcall_task": {
          const params = GetTaskSchema.parse(args);
          const result = await k8sClient.getTask(params.name, params.namespace);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "list_mcall_tasks": {
          const params = ListTasksSchema.parse(args);
          const result = await k8sClient.listTasks(params.namespace, params.labelSelector);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "delete_mcall_task": {
          const params = DeleteTaskSchema.parse(args);
          const result = await k8sClient.deleteTask(params.name, params.namespace);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "get_mcall_task_logs": {
          const params = GetTaskLogsSchema.parse(args);
          const result = await k8sClient.getTaskLogs(params.name, params.namespace);
          return {
            content: [
              {
                type: "text",
                text: result,
              },
            ],
          };
        }

        case "create_mcall_workflow": {
          const params = CreateWorkflowSchema.parse(args);
          const result = await k8sClient.createWorkflow(params);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "get_mcall_workflow": {
          const params = GetWorkflowSchema.parse(args);
          const result = await k8sClient.getWorkflow(params.name, params.namespace);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "list_mcall_workflows": {
          const params = ListWorkflowsSchema.parse(args);
          const result = await k8sClient.listWorkflows(params.namespace);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "delete_mcall_workflow": {
          const params = DeleteWorkflowSchema.parse(args);
          const result = await k8sClient.deleteWorkflow(params.name, params.namespace);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        case "triggerBuild": {
          const params = TriggerBuildSchema.parse(args);
          const result = await k8sClient.triggerJenkinsBuild(params.jobFullName, params.parameters);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(result, null, 2),
              },
            ],
          };
        }

        default:
          throw new Error(`Unknown tool: ${name}`);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        content: [
          {
            type: "text",
            text: `Error: ${errorMessage}`,
          },
        ],
        isError: true,
      };
    }
  });
}

