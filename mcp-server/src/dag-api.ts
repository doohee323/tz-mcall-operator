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
    const workflow = await k8sClient.getWorkflow(name, namespace);
    
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
    
    res.json(response);
  } catch (error) {
    console.error('Error getting workflow DAG:', error);
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
        const workflows = await k8sClient.listWorkflows(namespace);
        if (workflows.length > 0) {
          availableNamespaces.push({
            namespace,
            workflowCount: workflows.length
          });
        }
      } catch {
        // Namespace might not exist or no access
        continue;
      }
    }
    
    res.json({
      success: true,
      data: availableNamespaces
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

