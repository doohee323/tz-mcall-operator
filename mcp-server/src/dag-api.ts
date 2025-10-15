import { Router } from 'express';
import { KubernetesClient } from './kubernetes-client.js';

const router = Router();
const k8sClient = new KubernetesClient();

// GET /api/workflows - List all workflows
router.get('/workflows', async (req, res) => {
  try {
    const namespace = req.query.namespace as string || 'default';
    const workflows = await k8sClient.listWorkflows(namespace);
    
    res.json({
      success: true,
      data: workflows,
      count: workflows.length
    });
  } catch (error) {
    console.error('Error listing workflows:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/workflows/:namespace - List workflows in specific namespace
router.get('/workflows/:namespace', async (req, res) => {
  try {
    const { namespace } = req.params;
    const response = await k8sClient.listWorkflows(namespace);
    const workflows = response.items || [];
    
    // Extract just the workflow names
    const workflowNames = workflows.map((wf: any) => wf.metadata?.name).filter(Boolean);
    
    console.log('[DAG-API] 📋 List workflows in namespace:', namespace, '- found:', workflowNames.length);
    
    res.json({
      success: true,
      workflows: workflowNames,
      count: workflowNames.length
    });
  } catch (error) {
    console.error('Error listing workflows in namespace:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/workflows/:namespace/:name - Get workflow details
router.get('/workflows/:namespace/:name', async (req, res) => {
  try {
    const { namespace, name } = req.params;
    const workflow = await k8sClient.getWorkflow(name, namespace);
    
    res.json({
      success: true,
      data: workflow
    });
  } catch (error) {
    console.error('Error getting workflow:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/workflows/:namespace/:name/dag - Get workflow DAG
router.get('/workflows/:namespace/:name/dag', async (req, res) => {
  try {
    const { namespace, name } = req.params;
    
    // Force refresh by adding cache-busting parameters
    const forceRefresh = req.query.force === 'true' || req.query.refresh === 'true';
    
    console.log(`[DAG-API] 🔄 Fetching workflow ${namespace}/${name}, forceRefresh: ${forceRefresh}`);
    console.log(`[DAG-API] 🕐 Request timestamp: ${new Date().toISOString()}`);
    console.log(`[DAG-API] 🔗 Request URL: ${req.url}`);
    console.log(`[DAG-API] 📋 Query params:`, req.query);
    
    const workflow = await k8sClient.getWorkflow(name, namespace, forceRefresh);
    
    // Log raw workflow status for debugging
    console.log('[DAG-API] 📊 Raw workflow.metadata:', JSON.stringify(workflow.metadata, null, 2));
    console.log('[DAG-API] 📊 Raw workflow.status:', JSON.stringify(workflow.status, null, 2));
    console.log('[DAG-API] 🔗 workflow.status.dag:', JSON.stringify(workflow.status?.dag, null, 2));
    console.log('[DAG-API] 🆔 workflow.status.dag.runID:', workflow.status?.dag?.runID);
    console.log('[DAG-API] 📅 workflow.status.lastRunTime:', workflow.status?.lastRunTime);
    console.log('[DAG-API] ⏰ workflow.status.startTime:', workflow.status?.startTime);
    
    // Extract DAG from workflow status
    const dag = workflow.status?.dag || {
      nodes: [],
      edges: [],
      layout: 'dagre',
      metadata: {
        totalNodes: 0,
        totalEdges: 0,
        successCount: 0,
        failureCount: 0,
        runningCount: 0,
        pendingCount: 0,
        skippedCount: 0
      }
    };
    
    // Enhance nodes with detailed error information from individual McallTasks
    if (dag.nodes && dag.nodes.length > 0) {
      dag.nodes = await Promise.all(dag.nodes.map(async (node: any) => {
        const enhancedNode = { ...node };
        
        try {
          // Get individual McallTask status for this node
          // Task names are generated as {workflow-name}-{node.name}
          const taskName = `${name}-${node.name}`;
          if (taskName) {
            const task = await k8sClient.getTask(taskName, namespace);
            
            // Add error information from the actual McallTask
            if (task.status?.result?.errorMessage) {
              enhancedNode.errorMessage = task.status.result.errorMessage;
            }
            if (task.status?.result?.errorCode) {
              enhancedNode.errorCode = task.status.result.errorCode;
            }
            
            // Also add execution details
            if (task.status?.executionTimeMs) {
              enhancedNode.executionTimeMs = task.status.executionTimeMs;
            }
            if (task.status?.output) {
              enhancedNode.output = task.status.output;
            }
            
            console.log('[DAG-API] 🔍 Enhanced node with task data:', {
              id: node.id,
              name: node.name,
              phase: node.phase,
              errorCode: enhancedNode.errorCode,
              errorMessage: enhancedNode.errorMessage,
              executionTimeMs: enhancedNode.executionTimeMs
            });
          }
        } catch (error) {
          console.warn('[DAG-API] ⚠️ Could not fetch task details for node:', node.name, error instanceof Error ? error.message : String(error));
        }
        
        return enhancedNode;
      }));
    }
    
    console.log('[DAG-API] ✅ Extracted DAG nodes:', dag.nodes?.length || 0, 'edges:', dag.edges?.length || 0);
    
    // Build response with workflow info and DAG
    const response = {
      success: true,
      workflow: {
        name: workflow.metadata?.name,
        namespace: workflow.metadata?.namespace,
        phase: workflow.status?.phase,
        startTime: workflow.status?.startTime,
        completionTime: workflow.status?.completionTime,
        schedule: workflow.spec?.schedule,
        lastRunTime: workflow.status?.lastRunTime
      },
      dag: dag
    };
    
    console.log('[DAG-API] 🚀 Sending response - nodes:', response.dag.nodes?.length, 'edges:', response.dag.edges?.length);
    
    res.json(response);
  } catch (error) {
    console.error('Error getting workflow DAG:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/tasks/:namespace - List tasks in specific namespace
router.get('/tasks/:namespace', async (req, res) => {
  try {
    const { namespace } = req.params;
    const response = await k8sClient.listTasks(namespace);
    const tasks = response.items || [];
    
    // Extract just the task names
    const taskNames = tasks.map((task: any) => task.metadata?.name).filter(Boolean);
    
    console.log('[DAG-API] 📋 List tasks in namespace:', namespace, '- found:', taskNames.length);
    
    res.json({
      success: true,
      tasks: taskNames,
      count: taskNames.length
    });
  } catch (error) {
    console.error('Error listing tasks in namespace:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/tasks/:namespace/:name - Get task details
router.get('/tasks/:namespace/:name', async (req, res) => {
  try {
    const { namespace, name } = req.params;
    const task = await k8sClient.getTask(name, namespace);
    
    res.json({
      success: true,
      data: task
    });
  } catch (error) {
    console.error('Error getting task:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

// GET /api/namespaces - Get available namespaces with mcall resources
router.get('/namespaces', async (req, res) => {
  try {
    // Get all workflows and extract unique namespaces
    const allNamespaces = ['mcall-dev', 'mcall-system', 'default']; // Common namespaces
    const availableNamespaces = [];
    
    for (const namespace of allNamespaces) {
      try {
        const response = await k8sClient.listWorkflows(namespace);
        const workflows = response.items || [];
        if (workflows.length > 0) {
          availableNamespaces.push(namespace);
        }
      } catch {
        // Namespace might not exist or no access
        continue;
      }
    }
    
    console.log('[DAG-API] 🗂️ Available namespaces:', availableNamespaces);
    
    res.json({
      success: true,
      namespaces: availableNamespaces
    });
  } catch (error) {
    console.error('Error listing namespaces:', error);
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
});

export default router;

