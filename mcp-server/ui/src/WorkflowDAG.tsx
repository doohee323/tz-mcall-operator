import { useEffect, useState, useCallback } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
} from 'reactflow';
import type { Node, Edge } from 'reactflow';
import 'reactflow/dist/style.css';

interface WorkflowDAGProps {
  namespace: string;
  workflowName: string;
}

interface WorkflowInfo {
  name: string;
  namespace: string;
  phase: string;
  startTime?: string;
  completionTime?: string;
  schedule?: string;
}

interface DAGData {
  nodes: any[];
  edges: any[];
  metadata: {
    totalNodes: number;
    successCount: number;
    failureCount: number;
    runningCount: number;
    pendingCount: number;
    skippedCount: number;
  };
}

const getNodeColor = (phase: string): string => {
  switch (phase) {
    case 'Succeeded':
      return '#4caf50'; // Green
    case 'Failed':
      return '#f44336'; // Red
    case 'Running':
      return '#2196f3'; // Blue
    case 'Pending':
      return '#9e9e9e'; // Gray
    case 'Skipped':
      return '#e0e0e0'; // Light Gray
    default:
      return '#bdbdbd';
  }
};

const getEdgeColor = (type: string): string => {
  switch (type) {
    case 'success':
      return '#4caf50'; // Green
    case 'failure':
      return '#f44336'; // Red
    case 'always':
      return '#9e9e9e'; // Gray
    default:
      return '#757575';
  }
};

const getPhaseIcon = (phase: string): string => {
  switch (phase) {
    case 'Succeeded':
      return '✅';
    case 'Failed':
      return '🔴';
    case 'Running':
      return '🔵';
    case 'Pending':
      return '⚪';
    case 'Skipped':
      return '⏭️';
    default:
      return '❓';
  }
};

export function WorkflowDAG({ namespace, workflowName }: WorkflowDAGProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [workflow, setWorkflow] = useState<WorkflowInfo | null>(null);
  const [metadata, setMetadata] = useState<DAGData['metadata'] | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch initial DAG
  const fetchDAG = useCallback(async () => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const response = await fetch(`${apiUrl}/api/workflows/${namespace}/${workflowName}/dag`);
      const data = await response.json();

      if (data.success) {
        setWorkflow(data.workflow);
        setMetadata(data.dag.metadata);

        // Convert DAG nodes to ReactFlow nodes
        const flowNodes: Node[] = data.dag.nodes.map((node: any) => ({
          id: node.id,
          type: 'default',
          data: {
            label: (
              <div style={{ padding: '10px', textAlign: 'center' }}>
                <div style={{ fontWeight: 'bold', marginBottom: '5px' }}>
                  {getPhaseIcon(node.phase)} {node.name}
                </div>
                <div style={{ fontSize: '10px', color: '#666' }}>
                  {node.type?.toUpperCase()}
                </div>
                {node.duration && (
                  <div style={{ fontSize: '10px', marginTop: '3px' }}>
                    ⏱️ {node.duration}
                  </div>
                )}
                {node.errorCode !== undefined && (
                  <div style={{ fontSize: '10px', marginTop: '3px' }}>
                    Code: {node.errorCode}
                  </div>
                )}
              </div>
            ),
            ...node,
          },
          position: node.position || { x: 0, y: 0 },
          style: {
            background: getNodeColor(node.phase),
            color: '#fff',
            border: `2px solid ${getNodeColor(node.phase)}`,
            borderRadius: '8px',
            width: 200,
          },
        }));

        // Convert DAG edges to ReactFlow edges
        const flowEdges: Edge[] = data.dag.edges.map((edge: any) => ({
          id: edge.id,
          source: edge.source,
          target: edge.target,
          label: edge.label,
          type: 'smoothstep',
          animated: edge.type === 'success' || edge.type === 'failure',
          style: {
            stroke: getEdgeColor(edge.type),
            strokeWidth: 2,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: getEdgeColor(edge.type),
          },
        }));

        setNodes(flowNodes);
        setEdges(flowEdges);
      }
    } catch (error) {
      console.error('Error fetching DAG:', error);
    }
  }, [namespace, workflowName, setNodes, setEdges]);

  // Setup auto-refresh with polling (simple HTTP polling instead of WebSocket)
  useEffect(() => {
    setIsConnected(true);
    fetchDAG(); // Initial fetch

    // Auto-refresh every 5 seconds
    const interval = setInterval(() => {
      fetchDAG();
    }, 5000);

    return () => {
      clearInterval(interval);
      setIsConnected(false);
    };
  }, [namespace, workflowName, fetchDAG]);

  return (
    <div style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <div style={{ 
        padding: '20px', 
        background: '#1976d2', 
        color: '#fff',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <div>
          <h1 style={{ margin: 0, fontSize: '24px' }}>
            🔄 {workflow?.name || workflowName}
          </h1>
          <div style={{ fontSize: '14px', marginTop: '5px', opacity: 0.9 }}>
            Namespace: {workflow?.namespace || namespace} | 
            Phase: {workflow?.phase || 'Loading...'} |
            {workflow?.schedule && ` Schedule: ${workflow.schedule} |`}
            {isConnected ? ' 🟢 Auto-refresh (5s)' : ' 🔴 Stopped'}
          </div>
        </div>
        <button 
          onClick={fetchDAG}
          style={{
            padding: '8px 16px',
            background: '#fff',
            color: '#1976d2',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            fontWeight: 'bold'
          }}
        >
          ↻ Refresh
        </button>
      </div>

      {/* Stats Bar */}
      {metadata && (
        <div style={{ 
          padding: '10px 20px', 
          background: '#f5f5f5',
          display: 'flex',
          gap: '20px',
          fontSize: '14px'
        }}>
          <span>📊 Total: {metadata.totalNodes}</span>
          <span>✅ Success: {metadata.successCount}</span>
          <span>🔵 Running: {metadata.runningCount}</span>
          <span>🔴 Failed: {metadata.failureCount}</span>
          <span>⚪ Pending: {metadata.pendingCount}</span>
          <span>⏭️ Skipped: {metadata.skippedCount}</span>
        </div>
      )}

      {/* DAG Canvas */}
      <div style={{ flex: 1 }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          fitView
          attributionPosition="bottom-left"
        >
          <Background />
          <Controls />
          <MiniMap 
            nodeColor="#667eea"
            maskColor="rgba(0, 0, 0, 0.1)"
          />
        </ReactFlow>
      </div>

      {/* Legend */}
      <div style={{ 
        padding: '10px 20px', 
        background: '#f5f5f5',
        fontSize: '12px',
        display: 'flex',
        gap: '15px'
      }}>
        <span>Legend:</span>
        <span style={{ color: '#4caf50' }}>✅ Succeeded</span>
        <span style={{ color: '#2196f3' }}>🔵 Running</span>
        <span style={{ color: '#f44336' }}>🔴 Failed</span>
        <span style={{ color: '#9e9e9e' }}>⚪ Pending</span>
        <span style={{ color: '#bdbdbd' }}>⏭️ Skipped</span>
      </div>
    </div>
  );
}

