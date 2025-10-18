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
      console.log('[K8s-Client] 🏗️ Running inside Kubernetes cluster');
      console.log('[K8s-Client] 🌐 KUBERNETES_SERVICE_HOST:', process.env.KUBERNETES_SERVICE_HOST);
      this.kc.loadFromCluster();
    } else {
      // Running outside cluster, load from default kubeconfig
      console.log('[K8s-Client] 🏠 Running outside cluster, loading from default kubeconfig');
      console.log('[K8s-Client] 📁 KUBECONFIG:', process.env.KUBECONFIG || '~/.kube/config');
      this.kc.loadFromDefault();
    }

    // Log current context and cluster info
    const currentContext = this.kc.getCurrentContext();
    const cluster = this.kc.getCluster(currentContext);
    console.log('[K8s-Client] 🎯 Current context:', currentContext);
    console.log('[K8s-Client] 🌐 Cluster server:', cluster?.server || 'unknown');
    console.log('[K8s-Client] 📋 Namespace:', cluster?.name || 'unknown');

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
        
        // Format JSON output if it looks like JSON
        let output = task.status.result.output || "N/A";
        if (output !== "N/A" && output.startsWith('{') && output.includes('"')) {
          try {
            // Try to parse and format JSON
            const jsonObj = JSON.parse(output);
            output = JSON.stringify(jsonObj, null, 2);
            logs += `- Output (Formatted JSON):\n${output}\n`;
          } catch (e) {
            // If JSON parsing fails, check if it's truncated
            if (output.endsWith('...') || output.length > 500) {
              logs += `- Output (Truncated JSON - ${output.length} chars):\n${output}\n`;
              logs += `- Note: Output appears to be truncated. Full output may be available in pod logs.\n`;
            } else {
              logs += `- Output: ${output}\n`;
            }
          }
        } else {
          logs += `- Output: ${output}\n`;
        }
        
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
    
    // First create template tasks if they don't exist
    const taskRefs = [];
    for (const task of params.tasks) {
      // Template task name with -template suffix
      const templateTaskName = `${params.name}-${task.name}-template`;
      
      taskRefs.push({
        name: task.name,
        taskRef: {
          name: templateTaskName,
          namespace: namespace,
        },
        ...(task.dependencies && { dependencies: task.dependencies }),
      });

      // Create the template McallTask (used as a template for workflow execution)
      try {
        await this.createTask({
          name: templateTaskName,
          namespace: namespace,
          type: task.type,
          input: task.input,
        });
      } catch (error: any) {
        // Task might already exist, continue
        if (!error.message.includes("already exists")) {
          console.error(`Warning: Failed to create template task ${task.name}:`, error.message);
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

  async getWorkflow(name: string, namespace?: string, forceRefresh: boolean = false): Promise<any> {
    const ns = this.getNamespace(namespace);
    
    console.log(`[K8sClient] 🔍 ========== GET WORKFLOW START ==========`);
    console.log(`[K8sClient] 🔍 Workflow: ${ns}/${name}`);
    console.log(`[K8sClient] 🔍 Force refresh: ${forceRefresh}`);
    console.log(`[K8sClient] 🔍 Timestamp: ${new Date().toISOString()}`);
    
    try {
      if (forceRefresh) {
        console.log(`[K8sClient] 🔄 Force refreshing workflow ${ns}/${name}`);
        console.log(`[K8sClient] 🕐 Force refresh timestamp: ${new Date().toISOString()}`);
        
        // Method 1: Try with kubectl command (bypass API client cache completely)
        try {
          console.log(`[K8sClient] 🔧 Method 1: Using kubectl command to bypass cache`);
          const { exec } = await import('child_process');
          const { promisify } = await import('util');
          const execAsync = promisify(exec);
          
          const kubectlCmd = `kubectl get mcallworkflow ${name} -n ${ns} -o json`;
          console.log(`[K8sClient] 🔧 Executing: ${kubectlCmd}`);
          
          const { stdout } = await execAsync(kubectlCmd);
          const workflow = JSON.parse(stdout);
          
          console.log(`[K8sClient] 🔧 Method 1 success - workflow found via kubectl`);
          console.log(`[K8sClient] 🔧 Workflow metadata:`, JSON.stringify(workflow.metadata, null, 2));
          console.log(`[K8sClient] 🔧 Workflow status:`, JSON.stringify(workflow.status, null, 2));
          
          console.log(`[K8sClient] ✅ Method 1 SUCCESS - runID: ${workflow.status?.dag?.runID}`);
          console.log(`[K8sClient] 📊 Method 1 - lastRunTime: ${workflow.status?.lastRunTime}`);
          console.log(`[K8sClient] 📊 Method 1 - startTime: ${workflow.status?.startTime}`);
          return workflow;
        } catch (error1: any) {
          console.log(`[K8sClient] ⚠️ Method 1 FAILED: ${error1.message}`);
        }
        
        // Method 2: Try with fresh API client (fallback)
        try {
          console.log(`[K8sClient] 🔧 Method 2: Using completely fresh API client`);
          const freshKc = new k8s.KubeConfig();
          freshKc.loadFromDefault();
          const freshApi = freshKc.makeApiClient(k8s.CustomObjectsApi);
          
          const response = await freshApi.getNamespacedCustomObject(
            this.group,
            this.version,
            ns,
            this.workflowPlural,
            name
          );
          
          console.log(`[K8sClient] ✅ Method 2 SUCCESS - runID: ${(response.body as any).status?.dag?.runID}`);
          console.log(`[K8sClient] 📊 Method 2 - lastRunTime: ${(response.body as any).status?.lastRunTime}`);
          console.log(`[K8sClient] 📊 Method 2 - startTime: ${(response.body as any).status?.startTime}`);
          return response.body;
        } catch (error2: any) {
          console.log(`[K8sClient] ⚠️ Method 2 FAILED: ${error2.message}`);
        }
      }
      
      // Fallback to normal method
      console.log(`[K8sClient] 🔧 Normal method: Using cached API client`);
      const response = await this.k8sApi.getNamespacedCustomObject(
        this.group,
        this.version,
        ns,
        this.workflowPlural,
        name
      );
      
      console.log(`[K8sClient] ✅ Normal method SUCCESS - runID: ${(response.body as any).status?.dag?.runID}`);
      console.log(`[K8sClient] 📊 Normal method - lastRunTime: ${(response.body as any).status?.lastRunTime}`);
      console.log(`[K8sClient] 📊 Normal method - startTime: ${(response.body as any).status?.startTime}`);
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

  async triggerJenkinsBuild(jobFullName: string, parameters?: Record<string, string>): Promise<any> {
    const jenkinsUrl = process.env.JENKINS_URL || "https://jenkins.drillquiz.com";
    const username = process.env.JENKINS_USERNAME || "admin";
    const token = process.env.JENKINS_TOKEN || "11197fa40f409842983025803948aa6bcc";
    
    // Build Jenkins API URL
    let buildUrl = `${jenkinsUrl}/job/${jobFullName}/build`;
    
    // Add parameters if provided
    if (parameters && Object.keys(parameters).length > 0) {
      const paramString = Object.entries(parameters)
        .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
        .join('&');
      buildUrl += `WithParameters?${paramString}`;
    }
    
    try {
      const response = await fetch(buildUrl, {
        method: 'POST',
        headers: {
          'Authorization': `Basic ${Buffer.from(`${username}:${token}`).toString('base64')}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        return {
          success: false,
          error: `Jenkins API error: ${response.status} ${response.statusText}`,
          details: errorText,
          jobFullName,
          buildUrl,
          statusCode: response.status
        };
      }

      // Check if job is disabled by looking for specific response patterns
      const responseText = await response.text();
      const isDisabled = responseText.includes('disabled') || 
                        responseText.includes('This project is currently disabled') ||
                        responseText.includes('disabled=true');

      if (isDisabled) {
        return {
          success: false,
          error: "Jenkins job is disabled",
          details: "The requested Jenkins job is currently disabled and cannot be triggered",
          jobFullName,
          buildUrl,
          statusCode: response.status,
          jobDisabled: true
        };
      }

      // Extract build number from Location header if available
      const locationHeader = response.headers.get('Location');
      let buildNumber = null;
      if (locationHeader) {
        const buildMatch = locationHeader.match(/\/build\/(\d+)/);
        if (buildMatch) {
          buildNumber = parseInt(buildMatch[1]);
        }
      }

      return {
        success: true,
        message: "Jenkins build triggered successfully",
        jobFullName,
        buildUrl,
        buildNumber,
        statusCode: response.status,
        location: locationHeader
      };

    } catch (error: any) {
      return {
        success: false,
        error: `Failed to trigger Jenkins build: ${error.message}`,
        jobFullName,
        buildUrl,
        details: error.toString()
      };
    }
  }
}

