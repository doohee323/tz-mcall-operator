import * as k8s from "@kubernetes/client-node";

interface TaskParams {
  name: string;
  namespace?: string;
  type: "cmd" | "get" | "post";
  input: string;
  timeout?: number;
  retryCount?: number;
  schedule?: string;
  environment?: Record<string, string>;
}

interface WorkflowTask {
  name: string;
  type: "cmd" | "get" | "post";
  input: string;
  dependencies?: string[];
}

interface WorkflowParams {
  name: string;
  namespace?: string;
  tasks: WorkflowTask[];
  schedule?: string;
  timeout?: number;
}

export class KubernetesClient {
  private kc: k8s.KubeConfig;
  private k8sApi: k8s.CustomObjectsApi;
  private coreApi: k8s.CoreV1Api;
  private readonly defaultNamespace = "mcall-system";
  private readonly group = "mcall.tz.io";
  private readonly version = "v1";
  private readonly taskPlural = "mcalltasks";
  private readonly workflowPlural = "mcallworkflows";

  constructor() {
    this.kc = new k8s.KubeConfig();
    
    // Load kubeconfig from default location or in-cluster
    if (process.env.KUBERNETES_SERVICE_HOST) {
      // Running inside Kubernetes cluster
      this.kc.loadFromCluster();
    } else {
      // Running outside cluster, load from default kubeconfig
      this.kc.loadFromDefault();
    }

    this.k8sApi = this.kc.makeApiClient(k8s.CustomObjectsApi);
    this.coreApi = this.kc.makeApiClient(k8s.CoreV1Api);
  }

  private getNamespace(namespace?: string): string {
    return namespace || this.defaultNamespace;
  }

  // McallTask operations
  async createTask(params: TaskParams): Promise<any> {
    const namespace = this.getNamespace(params.namespace);
    
    const task = {
      apiVersion: `${this.group}/${this.version}`,
      kind: "McallTask",
      metadata: {
        name: params.name,
        namespace: namespace,
      },
      spec: {
        type: params.type,
        input: params.input,
        name: params.name,
        timeout: params.timeout || 30,
        retryCount: params.retryCount || 0,
        ...(params.schedule && { schedule: params.schedule }),
        ...(params.environment && { environment: params.environment }),
      },
    };

    try {
      const response = await this.k8sApi.createNamespacedCustomObject(
        this.group,
        this.version,
        namespace,
        this.taskPlural,
        task
      );
      return {
        status: "success",
        message: `McallTask '${params.name}' created successfully`,
        data: response.body,
      };
    } catch (error: any) {
      throw new Error(`Failed to create task: ${error.body?.message || error.message}`);
    }
  }

  async getTask(name: string, namespace?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      const response = await this.k8sApi.getNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.taskPlural,
        name
      );
      return response.body;
    } catch (error: any) {
      throw new Error(`Failed to get task: ${error.body?.message || error.message}`);
    }
  }

  async listTasks(namespace?: string, labelSelector?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      const response = await this.k8sApi.listNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.taskPlural,
        undefined,
        undefined,
        undefined,
        undefined,
        labelSelector
      );
      return response.body;
    } catch (error: any) {
      throw new Error(`Failed to list tasks: ${error.body?.message || error.message}`);
    }
  }

  async deleteTask(name: string, namespace?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      await this.k8sApi.deleteNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.taskPlural,
        name
      );
      return {
        status: "success",
        message: `McallTask '${name}' deleted successfully`,
      };
    } catch (error: any) {
      throw new Error(`Failed to delete task: ${error.body?.message || error.message}`);
    }
  }

  async getTaskLogs(name: string, namespace?: string): Promise<string> {
    const ns = this.getNamespace(namespace);
    
    try {
      // First get the task to check its status
      const task: any = await this.getTask(name, namespace);
      
      if (!task.status) {
        return "Task has not been executed yet. No logs available.";
      }

      let logs = `Task: ${name}\n`;
      logs += `Phase: ${task.status.phase}\n`;
      logs += `Start Time: ${task.status.startTime || "N/A"}\n`;
      logs += `Completion Time: ${task.status.completionTime || "N/A"}\n\n`;

      if (task.status.result) {
        logs += `Result:\n`;
        logs += `- Output: ${task.status.result.output || "N/A"}\n`;
        logs += `- Error Code: ${task.status.result.errorCode || "N/A"}\n`;
        logs += `- Error Message: ${task.status.result.errorMessage || "N/A"}\n`;
      }

      // Try to get pod logs if available
      try {
        const podSelector = `mcall-task=${name}`;
        const pods = await this.coreApi.listNamespacedPod(
          ns,
          undefined,
          undefined,
          undefined,
          undefined,
          podSelector
        );

        if (pods.body.items.length > 0) {
          logs += `\n--- Pod Logs ---\n`;
          for (const pod of pods.body.items) {
            if (pod.metadata?.name) {
              try {
                const podLogs = await this.coreApi.readNamespacedPodLog(
                  pod.metadata.name,
                  ns
                );
                logs += `\nPod: ${pod.metadata.name}\n${podLogs.body}\n`;
              } catch (e) {
                logs += `\nPod: ${pod.metadata.name}\n(Logs not available)\n`;
              }
            }
          }
        }
      } catch (podError) {
        // Pod logs might not be available, that's okay
      }

      return logs;
    } catch (error: any) {
      throw new Error(`Failed to get task logs: ${error.body?.message || error.message}`);
    }
  }

  // McallWorkflow operations
  async createWorkflow(params: WorkflowParams): Promise<any> {
    const namespace = this.getNamespace(params.namespace);
    
    // First create individual tasks if they don't exist
    const taskRefs = [];
    for (const task of params.tasks) {
      taskRefs.push({
        name: task.name,
        taskRef: {
          name: `${params.name}-${task.name}`,
          namespace: namespace,
        },
        ...(task.dependencies && { dependencies: task.dependencies }),
      });

      // Create the actual McallTask
      try {
        await this.createTask({
          name: `${params.name}-${task.name}`,
          namespace: namespace,
          type: task.type,
          input: task.input,
        });
      } catch (error: any) {
        // Task might already exist, continue
        if (!error.message.includes("already exists")) {
          console.error(`Warning: Failed to create task ${task.name}:`, error.message);
        }
      }
    }

    const workflow = {
      apiVersion: `${this.group}/${this.version}`,
      kind: "McallWorkflow",
      metadata: {
        name: params.name,
        namespace: namespace,
      },
      spec: {
        tasks: taskRefs,
        ...(params.schedule && { schedule: params.schedule }),
        ...(params.timeout && { timeout: params.timeout }),
      },
    };

    try {
      const response = await this.k8sApi.createNamespacedCustomObject(
        this.group,
        this.version,
        namespace,
        this.workflowPlural,
        workflow
      );
      return {
        status: "success",
        message: `McallWorkflow '${params.name}' created successfully`,
        data: response.body,
      };
    } catch (error: any) {
      throw new Error(`Failed to create workflow: ${error.body?.message || error.message}`);
    }
  }

  async getWorkflow(name: string, namespace?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      const response = await this.k8sApi.getNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.workflowPlural,
        name
      );
      return response.body;
    } catch (error: any) {
      throw new Error(`Failed to get workflow: ${error.body?.message || error.message}`);
    }
  }

  async listWorkflows(namespace?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      const response = await this.k8sApi.listNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.workflowPlural
      );
      return response.body;
    } catch (error: any) {
      throw new Error(`Failed to list workflows: ${error.body?.message || error.message}`);
    }
  }

  async deleteWorkflow(name: string, namespace?: string): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    try {
      await this.k8sApi.deleteNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.workflowPlural,
        name
      );
      return {
        status: "success",
        message: `McallWorkflow '${name}' deleted successfully`,
      };
    } catch (error: any) {
      throw new Error(`Failed to delete workflow: ${error.body?.message || error.message}`);
    }
  }
}

