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
  const [dagHistory, setDAGHistory] = useState<any[]>([]);
  const [selectedRunID, setSelectedRunID] = useState<string>('current');
  const [isStaleDAG, setIsStaleDAG] = useState(false); // Track if showing stale data

  // localStorage key for caching
  const cacheKey = `dag-cache-${namespace}-${workflowName}`;

  // Load cached DAG from localStorage on mount
  const [lastValidDAG, setLastValidDAG] = useState<any>(() => {
    try {
      const cached = localStorage.getItem(cacheKey);
      if (cached) {
        const parsedDAG = JSON.parse(cached);
        console.log('[DAG] 📦 Loaded from localStorage:', {
          nodes: parsedDAG.nodes?.length || 0,
          runID: parsedDAG.runID
        });
        return parsedDAG;
      } else {
        console.log('[DAG] 📦 No cache found in localStorage');
        return null;
      }
    } catch (e) {
      console.warn('[DAG] ❌ Failed to load from localStorage:', e);
      return null;
    }
  });

  // Fetch initial DAG
  const fetchDAG = useCallback(async () => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const response = await fetch(`${apiUrl}/api/workflows/${namespace}/${workflowName}/dag`);
      const data = await response.json();

      console.log('[DAG] API Response:', {
        success: data.success,
        workflowPhase: data.workflow?.phase,
        dagNodes: data.dag?.nodes?.length || 0,
        dagEdges: data.dag?.edges?.length || 0,
        dagRunID: data.dag?.runID,
        historyCount: data.dagHistory?.length || 0,
        timestamp: new Date().toISOString()
      });

      if (data.success) {
        setWorkflow(data.workflow);
        
        // Store history
        if (data.dagHistory) {
          setDAGHistory(data.dagHistory);
          console.log('[DAG] History updated:', data.dagHistory.length, 'runs');
        }
        
        // Determine which DAG to display
        let displayDAG = data.dag;
        let showingStale = false;
        
        console.log('[DAG] Display DAG before processing:', {
          nodes: displayDAG?.nodes?.length || 0,
          edges: displayDAG?.edges?.length || 0,
          runID: displayDAG?.runID
        });
        
        if (selectedRunID !== 'current' && data.dagHistory) {
          const historicalDAG = data.dagHistory.find((h: any) => h.runID === selectedRunID);
          if (historicalDAG) {
            displayDAG = historicalDAG;
          }
        }
        
        // If current DAG is empty but we have lastValidDAG, use it with stale indicator
        if (selectedRunID === 'current' && (!displayDAG.nodes || displayDAG.nodes.length === 0) && lastValidDAG && lastValidDAG.nodes && lastValidDAG.nodes.length > 0) {
          console.log('[DAG] 🔄 Using cached DAG (API returned empty)');
          displayDAG = lastValidDAG;
          showingStale = true;
        }
        
        // Cache valid DAG for later use (both memory and localStorage)
        if (displayDAG.nodes && displayDAG.nodes.length > 0) {
          console.log('[DAG] ✅ Caching valid DAG:', {
            nodes: displayDAG.nodes.length,
            runID: displayDAG.runID,
            phase: displayDAG.workflowPhase
          });
          setLastValidDAG(displayDAG);
          // Save to localStorage for persistence across page reloads
          try {
            localStorage.setItem(cacheKey, JSON.stringify(displayDAG));
            console.log('[DAG] 💾 Saved to localStorage:', cacheKey);
          } catch (e) {
            console.warn('[DAG] ❌ Failed to cache DAG to localStorage:', e);
          }
          if (selectedRunID === 'current') {
            setIsStaleDAG(false);
          }
        } else if (showingStale) {
          console.log('[DAG] ⚠️ Showing stale DAG');
          setIsStaleDAG(true);
        } else {
          console.log('[DAG] ⚠️ No DAG to display (empty and no cache)');
        }
        
        setMetadata(displayDAG.metadata);

        // Convert DAG nodes to ReactFlow nodes
        const flowNodes: Node[] = (displayDAG.nodes || []).map((node: any) => ({
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
            opacity: showingStale ? 0.5 : 1, // Dim when showing stale data
            filter: showingStale ? 'grayscale(30%)' : 'none',
          },
        }));

        // Convert DAG edges to ReactFlow edges
        const flowEdges: Edge[] = (displayDAG.edges || []).map((edge: any) => ({
          id: edge.id,
          source: edge.source,
          target: edge.target,
          label: edge.label,
          type: 'smoothstep',
          animated: showingStale ? false : (edge.type === 'success' || edge.type === 'failure'),
          style: {
            stroke: getEdgeColor(edge.type),
            strokeWidth: 2,
            opacity: showingStale ? 0.4 : 1, // Dim edges when stale
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: getEdgeColor(edge.type),
          },
        }));

        console.log('[DAG] 🎨 Rendering:', {
          flowNodes: flowNodes.length,
          flowEdges: flowEdges.length,
          isStale: showingStale,
          selectedRunID
        });

        setNodes(flowNodes);
        setEdges(flowEdges);
        
        console.log('[DAG] ✨ Nodes and Edges set successfully');
      }
    } catch (error) {
      console.error('[DAG] ❌ Error fetching DAG:', error);
    }
  }, [namespace, workflowName, selectedRunID, setNodes, setEdges, cacheKey, lastValidDAG]);

  // Setup auto-refresh with polling (simple HTTP polling instead of WebSocket)
  useEffect(() => {
    console.log('[DAG] 🚀 Starting auto-refresh for:', namespace, '/', workflowName);
    setIsConnected(true);
    fetchDAG(); // Initial fetch

    // Auto-refresh every 5 seconds
    const interval = setInterval(() => {
      console.log('[DAG] 🔄 Auto-refresh triggered');
      fetchDAG();
    }, 5000);

    return () => {
      console.log('[DAG] 🛑 Stopping auto-refresh');
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

      {/* Stale DAG Warning */}
      {isStaleDAG && selectedRunID === 'current' && (
        <div style={{
          padding: '10px 20px',
          background: '#d1ecf1',
          borderBottom: '1px solid #bee5eb',
          color: '#0c5460',
          fontSize: '14px',
          display: 'flex',
          alignItems: 'center',
          gap: '10px'
        }}>
          <span>ℹ️</span>
          <span>
            <strong>Previous run data</strong> - Workflow is resetting for next schedule. 
            New DAG will appear when next run starts.
          </span>
        </div>
      )}

      {/* Run History Selector */}
      {dagHistory.length > 0 && (
        <div style={{
          padding: '10px 20px',
          background: '#fff3cd',
          borderBottom: '1px solid #dee2e6',
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
          fontSize: '14px'
        }}>
          <span style={{ fontWeight: '500' }}>📜 Run History:</span>
          <select
            value={selectedRunID}
            onChange={(e) => setSelectedRunID(e.target.value)}
            style={{
              padding: '6px 12px',
              borderRadius: '4px',
              border: '1px solid #ddd',
              fontSize: '14px',
              cursor: 'pointer',
              flex: 1,
              maxWidth: '500px'
            }}
          >
            <option value="current">
              🔄 Current Run (Phase: {workflow?.phase || 'Loading...'})
            </option>
            {dagHistory.map((dag: any, idx: number) => (
              <option key={dag.runID} value={dag.runID}>
                {idx === 0 && '🕐 '} Run {idx + 1}: {dag.timestamp ? new Date(dag.timestamp).toLocaleString() : dag.runID} - {dag.workflowPhase} 
                {` (✅${dag.metadata?.successCount || 0} 🔴${dag.metadata?.failureCount || 0})`}
              </option>
            ))}
          </select>
          {selectedRunID !== 'current' && (
            <span style={{ color: '#856404', fontSize: '12px' }}>
              ⚠️ Viewing historical run
            </span>
          )}
        </div>
      )}

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

