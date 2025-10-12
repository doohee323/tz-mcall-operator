import { useEffect, useState, useCallback, useRef } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
} from 'reactflow';
import type { Node, Edge, ReactFlowInstance } from 'reactflow';
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
  const reactFlowInstance = useRef<ReactFlowInstance | null>(null);
  const [workflow, setWorkflow] = useState<WorkflowInfo | null>(null);
  const [metadata, setMetadata] = useState<DAGData['metadata'] | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [dagHistory, setDAGHistory] = useState<any[]>([]);
  const [selectedRunID, setSelectedRunID] = useState<string>('current');
  const [lastSavedRunID, setLastSavedRunID] = useState<string | null>(null); // Track last saved runID
  
  // Current namespace and workflow state
  const [currentNamespace, setCurrentNamespace] = useState(namespace);
  const [currentWorkflowName, setCurrentWorkflowName] = useState(workflowName);
  const [availableWorkflows, setAvailableWorkflows] = useState<string[]>([]);
  const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);
  const [_isStaleDAG, setIsStaleDAG] = useState(false); // Track if showing stale data
  
  // Task detail popup state
  const [selectedTask, setSelectedTask] = useState<any>(null);
  const [showTaskPopup, setShowTaskPopup] = useState(false);

  // localStorage keys for caching current DAG and history
  const cacheKey = `dag-cache-${currentNamespace}-${currentWorkflowName}`;
  const historyKey = `dag-history-${currentNamespace}-${currentWorkflowName}`;

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

  // Load history from localStorage
  const loadHistoryFromStorage = useCallback(() => {
    try {
      const stored = localStorage.getItem(historyKey);
      if (stored) {
        const history = JSON.parse(stored);
        console.log('[DAG] 📚 Loaded history from localStorage:', history.length, 'items');
        // Set last saved runID from the most recent history item
        if (history.length > 0 && history[0].runID) {
          setLastSavedRunID(history[0].runID);
          console.log('[DAG] 🔖 Last saved runID:', history[0].runID);
        }
        return history;
      }
    } catch (e) {
      console.error('[DAG] ❌ Error loading history:', e);
    }
    return [];
  }, [historyKey]);

  // Save DAG to history (max 10 items)
  const saveToHistory = useCallback((dag: any) => {
    if (!dag || !dag.runID || !dag.nodes || dag.nodes.length === 0) {
      return;
    }

    try {
      const history = loadHistoryFromStorage();
      
      // Check if this runID already exists
      const existingIndex = history.findIndex((item: any) => item.runID === dag.runID);
      if (existingIndex >= 0) {
        console.log('[DAG] 📚 RunID already in history:', dag.runID);
        return;
      }

      // Add to beginning of history
      const newHistory = [dag, ...history];
      
      // Keep only last 10 items
      if (newHistory.length > 10) {
        newHistory.splice(10);
      }

      localStorage.setItem(historyKey, JSON.stringify(newHistory));
      setDAGHistory(newHistory);
      console.log('[DAG] 💾 Saved to history:', dag.runID, '- Total items:', newHistory.length);
    } catch (e) {
      console.error('[DAG] ❌ Error saving to history:', e);
    }
  }, [historyKey, loadHistoryFromStorage]);

  // Fetch available namespaces
  const fetchAvailableNamespaces = useCallback(async () => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const apiKey = (window as any).MCP_API_KEY || '';
      const response = await fetch(`${apiUrl}/api/namespaces`, {
        headers: apiKey ? { 'X-API-Key': apiKey } : {}
      });
      if (response.ok) {
        const data = await response.json();
        setAvailableNamespaces(data.namespaces || []);
      }
    } catch (error) {
      console.error('[DAG] ❌ Error fetching namespaces:', error);
      setAvailableNamespaces([]);
    }
  }, []);

  // Fetch available workflows in namespace
  const fetchAvailableWorkflows = useCallback(async (ns: string) => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const apiKey = (window as any).MCP_API_KEY || '';
      const response = await fetch(`${apiUrl}/api/workflows/${ns}`, {
        headers: apiKey ? { 'X-API-Key': apiKey } : {}
      });
      if (response.ok) {
        const workflows = await response.json();
        setAvailableWorkflows(workflows.workflows || []);
      }
    } catch (error) {
      console.error('[DAG] ❌ Error fetching workflows:', error);
      setAvailableWorkflows([]);
    }
  }, []);


  // Handle namespace change
  const handleNamespaceChange = useCallback((newNamespace: string) => {
    setCurrentNamespace(newNamespace);
    setAvailableWorkflows([]); // Clear workflows when namespace changes
    if (newNamespace) {
      fetchAvailableWorkflows(newNamespace);
    }
  }, [fetchAvailableWorkflows]);

  // Handle workflow change
  const handleWorkflowChange = useCallback((newWorkflowName: string) => {
    setCurrentWorkflowName(newWorkflowName);
  }, []);

  // Fetch task details
  const fetchTaskDetails = useCallback(async (taskName: string) => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const apiKey = (window as any).MCP_API_KEY || '';
      const response = await fetch(`${apiUrl}/api/tasks/${currentNamespace}/${taskName}`, {
        headers: apiKey ? { 'X-API-Key': apiKey } : {}
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedTask(data.data);
        setShowTaskPopup(true);
      }
    } catch (error) {
      console.error('[DAG] ❌ Error fetching task details:', error);
    }
  }, [currentNamespace]);

  // Handle task detail button click
  const handleTaskDetailClick = useCallback((taskName: string) => {
    fetchTaskDetails(taskName);
  }, [fetchTaskDetails]);

  // Load initial data
  useEffect(() => {
    fetchAvailableNamespaces();
    fetchAvailableWorkflows(currentNamespace);
  }, [fetchAvailableNamespaces, fetchAvailableWorkflows, currentNamespace]);

  // Setup auto-refresh with polling (simple HTTP polling instead of WebSocket)
  useEffect(() => {
    console.log('[DAG] 🚀 Starting auto-refresh for:', currentNamespace, '/', currentWorkflowName);
    setIsConnected(true);
    
    // Define fetchDAG function inside useEffect to avoid dependency issues
    const fetchDAGInternal = async () => {
      try {
        const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
        const apiKey = (window as any).MCP_API_KEY || '';
        const response = await fetch(`${apiUrl}/api/workflows/${currentNamespace}/${currentWorkflowName}/dag`, {
          headers: apiKey ? { 'X-API-Key': apiKey } : {}
        });
        const data = await response.json();

        console.log('[DAG] API Response:', {
          success: data.success,
          workflowPhase: data.workflow?.phase,
          dagNodes: data.dag?.nodes?.length || 0,
          dagEdges: data.dag?.edges?.length || 0,
          dagRunID: data.dag?.runID,
          timestamp: new Date().toISOString()
        });

        // Update workflow info
        setWorkflow(data.workflow);
        
        // Load history from localStorage
        const localHistory = loadHistoryFromStorage();
        setDAGHistory(localHistory);

        // Determine which DAG to display
        let displayDAG = data.dag;
        let showingStale = false;

        if (selectedRunID !== 'current' && localHistory.length > 0) {
          const selectedDAG = localHistory.find((dag: any) => dag.runID === selectedRunID);
          if (selectedDAG) {
            displayDAG = selectedDAG;
            showingStale = true; // Historical data is considered "stale"
            console.log('[DAG] 📜 Using historical DAG from localStorage:', selectedRunID);
          }
        }

        // Check if we should show stale data
        if (!displayDAG || !displayDAG.nodes || displayDAG.nodes.length === 0) {
          if (lastValidDAG && lastValidDAG.nodes && lastValidDAG.nodes.length > 0) {
            displayDAG = lastValidDAG;
            showingStale = true;
            console.log('[DAG] 🔄 Using cached DAG (API returned empty)');
          }
        }

        if (displayDAG && displayDAG.nodes && displayDAG.nodes.length > 0) {
          // Update cache
          setLastValidDAG(displayDAG);
          try {
            localStorage.setItem(cacheKey, JSON.stringify(displayDAG));
            console.log('[DAG] 💾 Saved to localStorage:', cacheKey);
          } catch (e) {
            console.warn('[DAG] ❌ Failed to cache DAG to localStorage:', e);
          }
          
          // Save to history ONLY if:
          // 1. This is current run (not viewing history)
          // 2. DAG is from server (not cache)
          // 3. runID is NEW (different from last saved)
          if (selectedRunID === 'current' && data.dag && data.dag.runID && data.dag.runID !== lastSavedRunID) {
            saveToHistory(displayDAG);
            setLastSavedRunID(data.dag.runID);
            console.log('[DAG] ✨ New run detected and saved to history:', data.dag.runID);
            setIsStaleDAG(false);
          } else if (showingStale) {
            setIsStaleDAG(true);
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
                {/* Show HTTP status code for HTTP requests, otherwise error code */}
                {(node.httpStatusCode || node.errorCode !== undefined) && (
                  <div style={{ fontSize: '10px', marginTop: '3px' }}>
                    {node.httpStatusCode ? (
                      <span style={{
                        color: node.httpStatusCode >= 200 && node.httpStatusCode < 300 ? '#4caf50' : 
                               node.httpStatusCode >= 400 ? '#f44336' : '#ff9800'
                      }}>
                        HTTP: {node.httpStatusCode}
                      </span>
                    ) : (
                      `Exit: ${node.errorCode}`
                    )}
                  </div>
                )}
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleTaskDetailClick(node.taskRef || node.name);
                  }}
                  style={{
                    marginTop: '5px',
                    padding: '2px 6px',
                    fontSize: '10px',
                    background: 'rgba(255,255,255,0.2)',
                    color: '#fff',
                    border: '1px solid rgba(255,255,255,0.3)',
                    borderRadius: '3px',
                    cursor: 'pointer'
                  }}
                >
                  📋 Details
                </button>
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
            width: 140,
            opacity: showingStale ? 0.8 : 1, // Dim when showing stale data
            filter: showingStale ? 'grayscale(20%)' : 'none',
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
            opacity: showingStale ? 0.7 : 1, // Dim edges when stale
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: getEdgeColor(edge.type),
          },
        }));

        setNodes(flowNodes);
        setEdges(flowEdges);
        console.log('[DAG] ✨ Nodes and Edges set successfully');
      } catch (error) {
        console.error('[DAG] ❌ Error fetching DAG:', error);
      }
    };

    fetchDAGInternal(); // Initial fetch

    // Auto-refresh every 5 seconds (workflow runs every 1 minute)
    const interval = setInterval(() => {
      console.log('[DAG] 🔄 Auto-refresh triggered');
      fetchDAGInternal();
    }, 5000);

    return () => {
      console.log('[DAG] 🛑 Stopping auto-refresh');
      clearInterval(interval);
      setIsConnected(false);
    };
  }, [currentNamespace, currentWorkflowName, selectedRunID]);

  // Set zoom when nodes are updated
  useEffect(() => {
    if (reactFlowInstance.current && nodes.length > 0) {
      setTimeout(() => {
        reactFlowInstance.current?.setViewport({ x: 0, y: 0, zoom: 0.8 });
      }, 200);
    }
  }, [nodes.length]);

  return (
    <div style={{ width: '100%', height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <div style={{ 
        padding: '15px 20px', 
        background: '#1976d2', 
        color: '#fff',
        overflow: 'hidden',
        boxSizing: 'border-box'
      }}>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: '10px',
          width: '100%',
          boxSizing: 'border-box'
        }}>
          <div style={{ 
            flex: '1',
            overflow: 'hidden',
            minWidth: 0
          }}>
            <h1 style={{ margin: 0, fontSize: '24px' }}>
              🔄 Workflow DAG Viewer
            </h1>
            <div style={{ 
              fontSize: '14px', 
              marginTop: '5px', 
              opacity: 0.9, 
              lineHeight: '1.4',
              display: 'flex',
              alignItems: 'center',
              gap: '15px',
              flexWrap: 'wrap'
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
                <span>Namespace:</span>
                <select
                  value={currentNamespace}
                  onChange={(e) => handleNamespaceChange(e.target.value)}
                  style={{
                    padding: '2px 6px',
                    borderRadius: '4px',
                    border: '1px solid rgba(255,255,255,0.3)',
                    background: 'rgba(255,255,255,0.1)',
                    color: '#fff',
                    fontSize: '12px'
                  }}
                >
                  {availableNamespaces.map(ns => (
                    <option key={ns} value={ns} style={{ color: '#333' }}>
                      {ns}
                    </option>
                  ))}
                </select>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
                <span>Workflow:</span>
                <select
                  value={currentWorkflowName}
                  onChange={(e) => handleWorkflowChange(e.target.value)}
                  style={{
                    padding: '2px 6px',
                    borderRadius: '4px',
                    border: '1px solid rgba(255,255,255,0.3)',
                    background: 'rgba(255,255,255,0.1)',
                    color: '#fff',
                    fontSize: '12px'
                  }}
                >
                  {availableWorkflows.map(wf => (
                    <option key={wf} value={wf} style={{ color: '#333' }}>
                      {wf}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div style={{ 
              fontSize: '12px', 
              marginTop: '3px', 
              opacity: 0.8, 
              lineHeight: '1.4'
            }}>
              Phase: {workflow?.phase || 'Loading...'} |
              {workflow?.schedule && ` Schedule: ${workflow.schedule} |`}
              {isConnected ? ' 🟢 Auto-refresh (5s)' : ' 🔴 Stopped'}
            </div>
          </div>
          <div style={{ 
            flexShrink: 0,
            alignSelf: 'flex-start',
            marginTop: '8px'
          }}>
            <button 
              onClick={() => {
                // Force refresh by updating the component state
                setCurrentNamespace(currentNamespace);
                setCurrentWorkflowName(currentWorkflowName);
              }}
              style={{
                padding: '6px 8px',
                background: '#fff',
                color: '#1976d2',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: 'bold',
                whiteSpace: 'nowrap',
                fontSize: '11px',
                minWidth: '60px'
              }}
            >
              ↻
            </button>
          </div>
        </div>
      </div>

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
      <div style={{ flex: 1, width: '100%' }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onInit={(instance) => {
            reactFlowInstance.current = instance;
          }}
          minZoom={0.3}
          maxZoom={2}
          attributionPosition="bottom-left"
        >
          <Background />
          <Controls />
          <MiniMap 
            nodeColor="#667eea"
            maskColor="rgba(0, 0, 0, 0.1)"
            position="bottom-right"
            style={{
              position: 'absolute',
              right: '10px',
              bottom: '10px' // Closest to bottom of screen
            }}
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

      {/* Task Details Modal */}
      {showTaskPopup && selectedTask && (
        <div 
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000
          }}
          onClick={() => setShowTaskPopup(false)}
        >
          <div 
            style={{
              backgroundColor: '#fff',
              borderRadius: '8px',
              padding: '20px',
              maxWidth: '800px',
              maxHeight: '80vh',
              overflow: 'auto',
              boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
              position: 'relative'
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Close button */}
            <button
              onClick={() => setShowTaskPopup(false)}
              style={{
                position: 'absolute',
                top: '10px',
                right: '10px',
                background: 'none',
                border: 'none',
                fontSize: '24px',
                cursor: 'pointer',
                color: '#666'
              }}
            >
              ×
            </button>

            {/* Task Details Content */}
            <h2 style={{ marginTop: 0, marginBottom: '20px', color: '#1976d2' }}>
              📋 Task Details: {selectedTask.metadata?.name}
            </h2>

            <div style={{ fontSize: '14px', lineHeight: '1.6' }}>
              {/* Basic Info */}
              <div style={{ marginBottom: '20px' }}>
                <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Basic Information</h3>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <tbody>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Name:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.metadata?.name}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Namespace:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.metadata?.namespace}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Type:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.spec?.type}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Phase:</td>
                      <td style={{ padding: '8px' }}>
                        <span style={{
                          padding: '2px 8px',
                          borderRadius: '4px',
                          backgroundColor: 
                            selectedTask.status?.phase === 'Succeeded' ? '#4caf50' :
                            selectedTask.status?.phase === 'Failed' ? '#f44336' :
                            selectedTask.status?.phase === 'Running' ? '#2196f3' :
                            '#9e9e9e',
                          color: '#fff',
                          fontSize: '12px'
                        }}>
                          {selectedTask.status?.phase || 'Unknown'}
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Input (URL, Command, or other) */}
              {selectedTask.spec?.input && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>
                    {selectedTask.spec.type === 'get' || selectedTask.spec.type === 'post' ? 'URL' : 'Input/Command'}
                  </h3>
                  <pre style={{
                    backgroundColor: '#f5f5f5',
                    padding: '10px',
                    borderRadius: '4px',
                    overflow: 'auto',
                    fontSize: '12px',
                    wordBreak: 'break-all',
                    whiteSpace: 'pre-wrap'
                  }}>
                    {selectedTask.spec.input}
                  </pre>
                </div>
              )}

              {/* Timing */}
              {(selectedTask.status?.startTime || selectedTask.status?.completionTime) && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Timing</h3>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <tbody>
                      {selectedTask.status?.startTime && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Start Time:</td>
                          <td style={{ padding: '8px' }}>{new Date(selectedTask.status.startTime).toLocaleString()}</td>
                        </tr>
                      )}
                      {selectedTask.status?.completionTime && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold' }}>Completion Time:</td>
                          <td style={{ padding: '8px' }}>{new Date(selectedTask.status.completionTime).toLocaleString()}</td>
                        </tr>
                      )}
                      {selectedTask.status?.executionTime && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold' }}>Execution Time:</td>
                          <td style={{ padding: '8px' }}>{selectedTask.status.executionTime}</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Result - Error Code & Output */}
              {selectedTask.status?.result && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Result</h3>
                  <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '10px' }}>
                    <tbody>
                      {selectedTask.status.result.errorCode !== undefined && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Error Code:</td>
                          <td style={{ padding: '8px' }}>
                            <span style={{
                              padding: '2px 8px',
                              borderRadius: '4px',
                              backgroundColor: selectedTask.status.result.errorCode === '0' ? '#4caf50' : '#f44336',
                              color: '#fff',
                              fontSize: '12px'
                            }}>
                              {selectedTask.status.result.errorCode}
                            </span>
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                  {selectedTask.status.result.output && (
                    <div>
                      <h4 style={{ fontSize: '14px', marginBottom: '8px', color: '#666' }}>Output:</h4>
                      <pre style={{
                        textAlign: 'left',
                        backgroundColor: '#f5f5f5',
                        padding: '10px',
                        borderRadius: '4px',
                        overflow: 'auto',
                        fontSize: '11px',
                        maxHeight: '300px',
                        wordBreak: 'break-all',
                        whiteSpace: 'pre-wrap'
                      }}>
                        {selectedTask.status.result.output}
                      </pre>
                    </div>
                  )}
                </div>
              )}

              {/* Error Message */}
              {selectedTask.status?.message && selectedTask.status?.phase === 'Failed' && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#f44336' }}>Error Message</h3>
                  <div style={{
                    backgroundColor: '#ffebee',
                    padding: '10px',
                    borderRadius: '4px',
                    color: '#c62828',
                    border: '1px solid #ef9a9a'
                  }}>
                    {selectedTask.status.message}
                  </div>
                </div>
              )}

              {/* Additional Spec Info */}
              {(selectedTask.spec?.timeout || selectedTask.spec?.retryCount !== undefined) && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Configuration</h3>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <tbody>
                      {selectedTask.spec?.timeout && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Timeout:</td>
                          <td style={{ padding: '8px' }}>{selectedTask.spec.timeout}s</td>
                        </tr>
                      )}
                      {selectedTask.spec?.retryCount !== undefined && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold' }}>Retry Count:</td>
                          <td style={{ padding: '8px' }}>{selectedTask.spec.retryCount}</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

