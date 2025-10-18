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
  // const [lastSavedRunID, setLastSavedRunID] = useState<string | null>(null); // Track last saved runID - CACHE DISABLED
  
  // Current namespace and workflow state
  const [currentNamespace, setCurrentNamespace] = useState(namespace);
  const [currentWorkflowName, setCurrentWorkflowName] = useState(workflowName);
  const [availableWorkflows, setAvailableWorkflows] = useState<string[]>([]);
  const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);
  // const [_isStaleDAG, setIsStaleDAG] = useState(false); // Track if showing stale data - CACHE DISABLED
  
  // Task detail popup state
  const [selectedTask, setSelectedTask] = useState<any>(null);
  const [showTaskPopup, setShowTaskPopup] = useState(false);
  
  // Ref to store the fetchDAGInternal function for manual refresh
  const fetchDAGRef = useRef<((forceRefresh?: boolean) => Promise<void>) | null>(null);

  // localStorage keys for caching current DAG and history
  // const cacheKey = `dag-cache-${currentNamespace}-${currentWorkflowName}`;
  // const historyKey = `dag-history-${currentNamespace}-${currentWorkflowName}`;

  // Load cached DAG from localStorage on mount - CACHE DISABLED
  // const [lastValidDAG, setLastValidDAG] = useState<any>(() => {
  //   // CACHE DISABLED - Always return null to force fresh data
  //   console.log('[DAG] 🚫 Cache disabled - forcing fresh data');
  //   return null;
  //   
  //   /* CACHE CODE COMMENTED OUT
  //   try {
  //     const cached = localStorage.getItem(cacheKey);
  //     console.log('[DAG] 🔍 Checking localStorage for cacheKey:', cacheKey);
  //     console.log('[DAG] 🔍 Cached data exists:', !!cached);
  //     if (cached) {
  //       const parsedDAG = JSON.parse(cached);
  //       console.log('[DAG] 📦 Loaded from localStorage:', {
  //         nodes: parsedDAG.nodes?.length || 0,
  //         runID: parsedDAG.runID,
  //         timestamp: parsedDAG.timestamp,
  //         workflowPhase: parsedDAG.workflowPhase
  //       });
  //       console.log('[DAG] 📦 Full cached DAG:', parsedDAG);
  //       return parsedDAG;
  //     } else {
  //       console.log('[DAG] 📦 No cache found in localStorage for key:', cacheKey);
  //       return null;
  //     }
  //   } catch (e) {
  //     console.warn('[DAG] ❌ Failed to load from localStorage:', e);
  //     return null;
  //   }
  //   */
  // });

  // Load history from localStorage - CACHE DISABLED
  // const loadHistoryFromStorage = useCallback(() => {
  //   // CACHE DISABLED - Always return empty array
  //   console.log('[DAG] 🚫 History cache disabled - returning empty array');
  //   return [];
  //   
  //   /* CACHE CODE COMMENTED OUT
  //   try {
  //     const stored = localStorage.getItem(historyKey);
  //     if (stored) {
  //       const history = JSON.parse(stored);
  //       console.log('[DAG] 📚 Loaded history from localStorage:', history.length, 'items');
  //       // Set last saved runID from the most recent history item
  //       if (history.length > 0 && history[0].runID) {
  //         setLastSavedRunID(history[0].runID);
  //         console.log('[DAG] 🔖 Last saved runID:', history[0].runID);
  //       }
  //       return history;
  //     }
  //   } catch (e) {
  //     console.error('[DAG] ❌ Error loading history:', e);
  //   }
  //   return [];
  //   */
  // }, []); // Removed historyKey dependency

  // Save DAG to history (max 10 items) - CACHE DISABLED
  // const saveToHistory = useCallback((dag: any) => {
  //   // CACHE DISABLED - Skip saving to history
  //   console.log('[DAG] 🚫 History cache disabled - skipping save');
  //   return;
  //   
  //   /* CACHE CODE COMMENTED OUT
  //   console.log('[DAG] 💾 saveToHistory called with:', {
  //     hasDag: !!dag,
  //     runID: dag?.runID,
  //     nodesLength: dag?.nodes?.length,
  //     timestamp: dag?.timestamp,
  //     workflowPhase: dag?.workflowPhase
  //   });
  //   
  //   if (!dag || !dag.runID || !dag.nodes || dag.nodes.length === 0) {
  //     console.log('[DAG] 💾 Skipping save - invalid DAG data');
  //     return;
  //   }

  //   try {
  //     const history = loadHistoryFromStorage();
  //     console.log('[DAG] 💾 Current history length:', history.length);
  //     console.log('[DAG] 💾 Current history runIDs:', history.map((h: any) => h.runID));
  //     
  //     // Check if this runID already exists
  //     const existingIndex = history.findIndex((item: any) => item.runID === dag.runID);
  //     if (existingIndex >= 0) {
  //       console.log('[DAG] 📚 RunID already in history:', dag.runID, 'at index:', existingIndex);
  //       return;
  //     }

  //     // Add to beginning of history
  //     const newHistory = [dag, ...history];
  //     console.log('[DAG] 💾 New history will have length:', newHistory.length);
  //     
  //     // Keep only last 10 items
  //     if (newHistory.length > 10) {
  //       newHistory.splice(10);
  //       console.log('[DAG] 💾 Trimmed history to 10 items');
  //     }

  //     localStorage.setItem(historyKey, JSON.stringify(newHistory));
  //     setDAGHistory(newHistory);
  //     console.log('[DAG] 💾 Saved to history:', dag.runID, '- Total items:', newHistory.length);
  //     console.log('[DAG] 💾 New history runIDs:', newHistory.map((h: any) => h.runID));
  //   } catch (e) {
  //     console.error('[DAG] ❌ Error saving to history:', e);
  //   }
  //   */
  // }, []); // Removed dependencies

  // Fetch available namespaces
  const fetchAvailableNamespaces = useCallback(async () => {
    try {
      // Use current origin for API calls (supports port-forwarding)
      const apiUrl = import.meta.env.VITE_API_URL || window.location.origin;
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
      // Use current origin for API calls (supports port-forwarding)
      const apiUrl = import.meta.env.VITE_API_URL || window.location.origin;
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
    console.log('[DAG] 🔍 ========== fetchTaskDetails START ==========');
    console.log('[DAG] 🔍 Called with taskName:', taskName);
    console.log('[DAG] 🔍 Current namespace:', currentNamespace);
    console.log('[DAG] 🔍 Current workflow:', currentWorkflowName);
    console.log('[DAG] 🔍 Nodes array length:', nodes.length);
    console.log('[DAG] 🔍 Nodes array:', nodes);
    
    try {
      // If nodes array is empty, fetch fresh DAG data
      let dagData = null;
      if (nodes.length === 0) {
        console.log('[DAG] 🔍 Nodes array is empty, fetching fresh DAG data...');
        const apiUrl = import.meta.env.VITE_API_URL || window.location.origin;
        const apiKey = (window as any).MCP_API_KEY || '';
        const response = await fetch(`${apiUrl}/api/workflows/${currentNamespace}/${currentWorkflowName}/dag?force=true&t=${Date.now()}`, {
          headers: apiKey ? { 'X-API-Key': apiKey } : {}
        });
        const data = await response.json();
        dagData = data.dag;
        console.log('[DAG] 🔍 Fresh DAG data fetched:', dagData);
      }
      
      // Use either React state nodes or fresh DAG data
      const searchNodes = nodes.length > 0 ? nodes : (dagData?.nodes || []);
      console.log('[DAG] 🔍 Search nodes length:', searchNodes.length);
      console.log('[DAG] 🔍 Search nodes:', searchNodes);
      console.log('[DAG] 🔍 Node IDs:', searchNodes.map((n: any) => n.id));
      console.log('[DAG] 🔍 Node data names:', searchNodes.map((n: any) => n.data?.name));
      console.log('[DAG] 🔍 Node taskRefs:', searchNodes.map((n: any) => n.data?.taskRef));
      console.log('[DAG] 🔍 Node phases:', searchNodes.map((n: any) => n.data?.phase));
      console.log('[DAG] 🔍 Node errorMessages:', searchNodes.map((n: any) => n.data?.errorMessage));
      
      // First try to find by node.data.name (actual task name from workflow execution)
      console.log('[DAG] 🔍 Searching by node.data.name === taskName...');
      let currentTask = searchNodes.find((node: any) => node.data?.name === taskName);
      console.log('[DAG] 🔍 Found task by data.name:', currentTask);
      
      // If not found, try by node.id
      if (!currentTask) {
        console.log('[DAG] 🔍 Searching by node.id === taskName...');
        currentTask = searchNodes.find((node: any) => node.id === taskName);
        console.log('[DAG] 🔍 Found task by id:', currentTask);
      }
      
      // If still not found, try to find by removing '-template' suffix
      if (!currentTask && taskName.endsWith('-template')) {
        const actualTaskName = taskName.replace('-template', '');
        console.log('[DAG] 🔍 Searching by actual task name (removed -template):', actualTaskName);
        currentTask = searchNodes.find((node: any) => node.data?.name === actualTaskName);
        console.log('[DAG] 🔍 Found task by actual name:', currentTask);
      }
      
      if (currentTask) {
        console.log('[DAG] 🔍 ✅ Task found! Fetching detailed task information...');
        console.log('[DAG] 🔍 Raw currentTask object:', currentTask);
        console.log('[DAG] 🔍 currentTask.data:', currentTask.data);
        console.log('[DAG] 🔍 currentTask.id:', currentTask.id);
        
        // Generate actual task name using workflow-name pattern
        const actualTaskName = `${currentWorkflowName}-${currentTask.id}`;
        console.log('[DAG] 🔍 Generated actual task name:', actualTaskName);
        
        try {
          // Fetch detailed task information from API
          const apiUrl = import.meta.env.VITE_API_URL || window.location.origin;
          const apiKey = (window as any).MCP_API_KEY || '';
          const response = await fetch(`${apiUrl}/api/tasks/${currentNamespace}/${actualTaskName}`, {
            headers: apiKey ? { 'X-API-Key': apiKey } : {}
          });
          
          if (response.ok) {
            const taskData = await response.json();
            console.log('[DAG] 🔍 Fetched task data from API:', taskData);
            
            if (taskData.success && taskData.data) {
              const task = taskData.data;
              const taskDetails = {
                name: task.metadata?.name || currentTask.id,
                namespace: task.metadata?.namespace || currentNamespace,
                type: task.spec?.type || currentTask.type,
                phase: task.status?.phase || currentTask.phase || 'Unknown',
                input: task.spec?.input || currentTask.input,
                output: task.status?.result?.output || currentTask.output,
                errorMessage: task.status?.result?.errorMessage || currentTask.errorMessage,
                startTime: task.status?.startTime || currentTask.startTime,
                endTime: task.status?.completionTime || currentTask.endTime,
                duration: task.status?.executionTimeMs || currentTask.duration,
                errorCode: task.status?.result?.errorCode || currentTask.errorCode,
                httpStatusCode: task.status?.httpStatusCode || currentTask.httpStatusCode,
                dependencies: task.spec?.dependencies || [],
                timeout: task.spec?.timeout,
                resources: task.spec?.resources,
                labels: task.metadata?.labels || {},
                annotations: task.metadata?.annotations || {}
              };
              console.log('[DAG] 🔍 Task details created from API:', taskDetails);
              setSelectedTask(taskDetails);
              setShowTaskPopup(true);
              console.log('[DAG] 🔍 ✅ Task popup opened successfully');
              return;
            }
          }
          
          console.warn('[DAG] ⚠️ Failed to fetch task details from API, using DAG data');
        } catch (error) {
          console.warn('[DAG] ⚠️ Error fetching task details from API:', error);
        }
        
        // Fallback: Create task details object from DAG node data
        const taskDetails = {
          name: currentTask.id || currentTask.data?.name,
          namespace: currentNamespace,
          type: currentTask.data?.type || currentTask.type,
          phase: currentTask.data?.phase || currentTask.phase || 'Unknown',
          input: currentTask.data?.input || currentTask.input,
          output: currentTask.data?.output || currentTask.output,
          errorMessage: currentTask.data?.errorMessage || currentTask.errorMessage,
          startTime: currentTask.data?.startTime || currentTask.startTime,
          endTime: currentTask.data?.endTime || currentTask.endTime,
          duration: currentTask.data?.duration || currentTask.duration,
          errorCode: currentTask.data?.errorCode || currentTask.errorCode,
          httpStatusCode: currentTask.data?.httpStatusCode || currentTask.httpStatusCode
        };
        console.log('[DAG] 🔍 Task details created from DAG data (fallback):', taskDetails);
        setSelectedTask(taskDetails);
        setShowTaskPopup(true);
        console.log('[DAG] 🔍 ✅ Task popup opened successfully');
      } else {
        console.error('[DAG] ❌ Task not found in current DAG:', taskName);
        console.log('[DAG] 🔍 Available task names:', searchNodes.map((n: any) => n.data?.name));
        console.log('[DAG] 🔍 Available task IDs:', searchNodes.map((n: any) => n.id));
        console.log('[DAG] 🔍 Available taskRefs:', searchNodes.map((n: any) => n.data?.taskRef));
      }
    } catch (error) {
      console.error('[DAG] ❌ Error getting task details from DAG:', error);
      console.error('[DAG] ❌ Error stack:', error instanceof Error ? error.stack : 'No stack trace');
    }
    console.log('[DAG] 🔍 ========== fetchTaskDetails END ==========');
  }, [currentNamespace, currentWorkflowName, nodes]);

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
    const fetchDAGInternal = async (forceRefresh: boolean = false) => {
      try {
        // Skip API call if namespace or workflow name is empty
        if (!currentNamespace || !currentWorkflowName) {
          console.log('[DAG] ⚠️ Skipping API call - namespace or workflow name is empty');
          return;
        }
        
        // Use current origin for API calls (supports port-forwarding)
        const apiUrl = import.meta.env.VITE_API_URL || window.location.origin;
        const apiKey = (window as any).MCP_API_KEY || '';
        
        // Add force parameter if forceRefresh is true
        const forceParam = forceRefresh ? '&force=true' : '';
        const response = await fetch(`${apiUrl}/api/workflows/${currentNamespace}/${currentWorkflowName}/dag?t=${Date.now()}${forceParam}`, {
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
        
        // CACHE DISABLED - Always use fresh data from API
        console.log('[DAG] 🚫 Cache disabled - using fresh API data only');
        const localHistory: any[] = []; // Empty history
        setDAGHistory(localHistory);

        // Always use fresh data from API
        let displayDAG = data.dag;
        let showingStale = false;

        /* CACHE CODE COMMENTED OUT
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
        // IMPORTANT: Only use cache if API returned empty data AND we're not forcing refresh
        // If API returned data with a different runID, we should use that instead of cache
        if (!displayDAG || !displayDAG.nodes || displayDAG.nodes.length === 0) {
          if (!forceRefresh && lastValidDAG && lastValidDAG.nodes && lastValidDAG.nodes.length > 0) {
            displayDAG = lastValidDAG;
            showingStale = true;
            console.log('[DAG] 🔄 Using cached DAG (API returned empty)');
          } else if (forceRefresh) {
            console.log('[DAG] 🔄 Force refresh - NOT using cached DAG even though API returned empty');
          }
        } else {
          // API returned data - check if it's a new runID
          const currentRunID = displayDAG.runID;
          const cachedRunID = lastValidDAG?.runID;
          
          if (currentRunID && cachedRunID && currentRunID !== cachedRunID) {
            console.log('[DAG] 🆕 NEW RUNID DETECTED!', {
              currentRunID,
              cachedRunID,
              message: 'Using new runID from API instead of cache'
            });
            // Don't use cache - use the new data from API
            showingStale = false;
          } else if (currentRunID === cachedRunID) {
            console.log('[DAG] 🔄 Same runID as cache:', currentRunID);
          }
        }
        */

        console.log('[DAG] 🔍 ========== fetchDAGInternal DAG Processing ==========');
        console.log('[DAG] 🔍 displayDAG:', displayDAG);
        console.log('[DAG] 🔍 displayDAG.nodes:', displayDAG?.nodes);
        console.log('[DAG] 🔍 displayDAG.nodes.length:', displayDAG?.nodes?.length);
        
        if (displayDAG && displayDAG.nodes && displayDAG.nodes.length > 0) {
          console.log('[DAG] 🔍 ✅ Valid DAG found, processing nodes...');
          console.log('[DAG] 🔍 Raw nodes data:', displayDAG.nodes);
          
          // CACHE DISABLED - Skip all caching operations
          console.log('[DAG] 🚫 Cache disabled - skipping all cache operations');
          
          /* CACHE CODE COMMENTED OUT
          // Update cache
          setLastValidDAG(displayDAG);
          try {
            console.log('[DAG] 💾 Saving to localStorage:', {
              cacheKey,
              runID: displayDAG.runID,
              timestamp: displayDAG.timestamp,
              workflowPhase: displayDAG.workflowPhase,
              nodesLength: displayDAG.nodes?.length
            });
            localStorage.setItem(cacheKey, JSON.stringify(displayDAG));
            console.log('[DAG] 💾 Successfully saved to localStorage:', cacheKey);
          } catch (e) {
            console.warn('[DAG] ❌ Failed to cache DAG to localStorage:', e);
          }
          
          // Save to history ONLY if:
          // 1. This is current run (not viewing history)
          // 2. DAG is from server (not cache)
          // 3. runID is NEW (different from last saved)
          console.log('[DAG] 🔍 Checking if should save to history:', {
            selectedRunID,
            hasDataDag: !!data.dag,
            dataDagRunID: data.dag?.runID,
            lastSavedRunID,
            isCurrentRun: selectedRunID === 'current',
            hasRunID: !!data.dag?.runID,
            isNewRunID: data.dag?.runID !== lastSavedRunID
          });
          
          if (selectedRunID === 'current' && data.dag && data.dag.runID && data.dag.runID !== lastSavedRunID) {
            console.log('[DAG] ✨ NEW RUN DETECTED! Saving to history:', data.dag.runID);
            saveToHistory(displayDAG);
            setLastSavedRunID(data.dag.runID);
            console.log('[DAG] ✨ New run detected and saved to history:', data.dag.runID);
            setIsStaleDAG(false);
          } else {
            console.log('[DAG] 📚 NOT saving to history - conditions not met:', {
              reason: !selectedRunID ? 'not current run' : 
                      !data.dag ? 'no data.dag' :
                      !data.dag.runID ? 'no runID' :
                      data.dag.runID === lastSavedRunID ? 'same runID' : 'unknown'
            });
            if (showingStale) {
              setIsStaleDAG(true);
            }
          }
          */
        } else {
          console.log('[DAG] ⚠️ No DAG to display (empty data)');
        }
        
        /* CACHE CODE COMMENTED OUT
        } else if (showingStale) {
          console.log('[DAG] ⚠️ Showing stale DAG');
          setIsStaleDAG(true);
        } else {
          console.log('[DAG] ⚠️ No DAG to display (empty and no cache)');
        }
        */
        
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
                    // Use node.name (actual task name) instead of node.taskRef (template name)
                    handleTaskDetailClick(node.name);
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

        console.log('[DAG] 🔍 ========== Setting ReactFlow Nodes ==========');
        console.log('[DAG] 🔍 flowNodes length:', flowNodes.length);
        console.log('[DAG] 🔍 flowNodes:', flowNodes);
        console.log('[DAG] 🔍 flowEdges length:', flowEdges.length);
        console.log('[DAG] 🔍 flowEdges:', flowEdges);
        
        setNodes(flowNodes);
        setEdges(flowEdges);
        
        console.log('[DAG] 🔍 ✅ ReactFlow nodes and edges set successfully');
        console.log('[DAG] ✨ Nodes and Edges set successfully');
      } catch (error) {
        console.error('[DAG] ❌ Error fetching DAG:', error);
      }
    };

    // Store the function in ref for manual refresh
    fetchDAGRef.current = fetchDAGInternal;
    
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

  // Control auto-refresh based on workflow status
  useEffect(() => {
    if (workflow?.phase === 'NotFound') {
      console.log('[DAG] 🛑 Stopping auto-refresh - workflow not found');
      setIsConnected(false);
    } else if (workflow?.phase && workflow.phase !== 'NotFound') {
      console.log('[DAG] 🟢 Starting auto-refresh - workflow found');
      setIsConnected(true);
    }
  }, [workflow?.phase]);

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
                console.log('[DAG] 🔄 Manual refresh button clicked - forcing refresh');
                // Force refresh by calling fetchDAGInternal with forceRefresh=true
                if (fetchDAGRef.current) {
                  fetchDAGRef.current(true);
                } else {
                  console.warn('[DAG] ⚠️ fetchDAGRef.current is null - falling back to state update');
                  setCurrentNamespace(currentNamespace);
                  setCurrentWorkflowName(currentWorkflowName);
                }
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
              📋 Task Details: {selectedTask.name || selectedTask.metadata?.name}
            </h2>

            <div style={{ fontSize: '14px', lineHeight: '1.6' }}>
              {/* Basic Info */}
              <div style={{ marginBottom: '20px' }}>
                <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Basic Information</h3>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <tbody>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Name:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.name || selectedTask.metadata?.name}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Namespace:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.namespace || selectedTask.metadata?.namespace}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Type:</td>
                      <td style={{ padding: '8px' }}>{selectedTask.type || selectedTask.spec?.type}</td>
                    </tr>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                      <td style={{ padding: '8px', fontWeight: 'bold' }}>Phase:</td>
                      <td style={{ padding: '8px' }}>
                        <span style={{
                          padding: '2px 8px',
                          borderRadius: '4px',
                          backgroundColor: 
                            (selectedTask.phase || selectedTask.status?.phase) === 'Succeeded' ? '#4caf50' :
                            (selectedTask.phase || selectedTask.status?.phase) === 'Failed' ? '#f44336' :
                            (selectedTask.phase || selectedTask.status?.phase) === 'Running' ? '#2196f3' :
                            '#9e9e9e',
                          color: '#fff',
                          fontSize: '12px'
                        }}>
                          {selectedTask.phase || selectedTask.status?.phase || 'Unknown'}
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Input (URL, Command, or other) */}
              {(selectedTask.input || selectedTask.spec?.input) && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>
                    {(selectedTask.type || selectedTask.spec?.type) === 'get' || (selectedTask.type || selectedTask.spec?.type) === 'post' ? 'URL' : 'Input/Command'}
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
                    {selectedTask.input || selectedTask.spec?.input}
                  </pre>
                </div>
              )}

              {/* Timing */}
              {(selectedTask.startTime || selectedTask.endTime || selectedTask.duration || selectedTask.status?.startTime || selectedTask.status?.completionTime) && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Timing</h3>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <tbody>
                      {(selectedTask.startTime || selectedTask.status?.startTime) && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Start Time:</td>
                          <td style={{ padding: '8px' }}>{new Date(selectedTask.startTime || selectedTask.status?.startTime).toLocaleString()}</td>
                        </tr>
                      )}
                      {(selectedTask.endTime || selectedTask.status?.completionTime) && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold' }}>Completion Time:</td>
                          <td style={{ padding: '8px' }}>{new Date(selectedTask.endTime || selectedTask.status?.completionTime).toLocaleString()}</td>
                        </tr>
                      )}
                      {(selectedTask.duration || selectedTask.status?.executionTimeMs) && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold' }}>Execution Time:</td>
                          <td style={{ padding: '8px' }}>{selectedTask.duration || selectedTask.status?.executionTimeMs}ms</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Result - Error Code & Output */}
              {(selectedTask.errorCode !== undefined || selectedTask.errorMessage || selectedTask.output || selectedTask.status?.result) && (
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ fontSize: '16px', marginBottom: '10px', color: '#333' }}>Result</h3>
                  <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '10px' }}>
                    <tbody>
                      {(selectedTask.errorCode !== undefined || (selectedTask.status?.result?.errorCode !== undefined)) && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Error Code:</td>
                          <td style={{ padding: '8px' }}>
                            <span style={{
                              padding: '2px 8px',
                              borderRadius: '4px',
                              backgroundColor: (selectedTask.errorCode === 0 || selectedTask.status?.result?.errorCode === '0') ? '#4caf50' : '#f44336',
                              color: '#fff',
                              fontSize: '12px'
                            }}>
                              {selectedTask.errorCode !== undefined ? selectedTask.errorCode : selectedTask.status?.result?.errorCode}
                            </span>
                          </td>
                        </tr>
                      )}
                      {(selectedTask.errorMessage || selectedTask.status?.result?.errorMessage) && (
                        <tr style={{ borderBottom: '1px solid #eee' }}>
                          <td style={{ padding: '8px', fontWeight: 'bold', width: '150px' }}>Error Message:</td>
                          <td style={{ padding: '8px', color: '#f44336', fontWeight: 'bold' }}>
                            {selectedTask.errorMessage || selectedTask.status?.result?.errorMessage}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                  {(selectedTask.output || selectedTask.status?.result?.output) && (
                    <div>
                      <h4 style={{ fontSize: '14px', marginBottom: '8px', color: '#666' }}>Output:</h4>
                      {(() => {
                        const output = selectedTask.output || selectedTask.status?.result?.output;
                        console.log('[JSON-FORMAT] 🔍 Raw output:', output);
                        console.log('[JSON-FORMAT] 🔍 Output type:', typeof output);
                        console.log('[JSON-FORMAT] 🔍 Output length:', output?.length);
                        
                        let formattedOutput = output;
                        let isJson = false;
                        let isTruncated = false;
                        
                        // Check if output looks like JSON
                        if (output && typeof output === 'string' && output.trim().startsWith('{')) {
                          console.log('[JSON-FORMAT] 🔍 Detected JSON-like output');
                          console.log('[JSON-FORMAT] 📏 Output length:', output.length);
                          console.log('[JSON-FORMAT] 🔚 Ends with dots:', output.endsWith('...'));
                          
                          // Check if it's truncated
                          if (output.endsWith('...') || output.length > 500) {
                            isTruncated = true;
                            console.log('[JSON-FORMAT] ⚠️ Output appears to be truncated');
                            console.log('[JSON-FORMAT] 📊 Truncation details:', {
                              endsWithDots: output.endsWith('...'),
                              length: output.length,
                              isLong: output.length > 500
                            });
                          }
                          
                          // Try to parse as JSON and format it
                          try {
                            console.log('[JSON-FORMAT] 🔄 Attempting JSON.parse...');
                            console.log('[JSON-FORMAT] 📝 Raw output to parse:', output.substring(0, 100) + '...');
                            const parsed = JSON.parse(output);
                            console.log('[JSON-FORMAT] ✅ JSON.parse successful');
                            console.log('[JSON-FORMAT] 📦 Parsed object keys:', Object.keys(parsed));
                            formattedOutput = JSON.stringify(parsed, null, 2);
                            isJson = true;
                            console.log('[JSON-FORMAT] ✅ JSON formatting applied');
                            console.log('[JSON-FORMAT] 📏 Formatted length:', formattedOutput.length);
                          } catch (e) {
                            console.log('[JSON-FORMAT] ❌ JSON.parse failed:', e instanceof Error ? e.message : String(e));
                            console.log('[JSON-FORMAT] 🔍 Error details:', {
                              name: e instanceof Error ? e.name : 'Unknown',
                              message: e instanceof Error ? e.message : String(e),
                              stack: e instanceof Error ? e.stack : undefined
                            });
                            
                            // Try to fix truncated JSON
                            if (isTruncated) {
                              console.log('[JSON-FORMAT] 🔄 Attempting to fix truncated JSON...');
                              try {
                                let fixedOutput = output.trim();
                                console.log('[JSON-FORMAT] 🧹 After trim:', fixedOutput.substring(0, 100) + '...');
                                
                                // Remove trailing dots
                                fixedOutput = fixedOutput.replace(/\.\.\.$/, '');
                                console.log('[JSON-FORMAT] ✂️ After removing dots:', fixedOutput.substring(0, 100) + '...');
                                
                                // Try to close incomplete JSON structures
                                if (fixedOutput.startsWith('{')) {
                                  // Count braces to see if we need to close
                                  const openBraces = (fixedOutput.match(/\{/g) || []).length;
                                  const closeBraces = (fixedOutput.match(/\}/g) || []).length;
                                  
                                  console.log('[JSON-FORMAT] 🔢 Brace count:', { openBraces, closeBraces });
                                  
                                  if (openBraces > closeBraces) {
                                    // Add missing closing braces
                                    const missingBraces = openBraces - closeBraces;
                                    console.log('[JSON-FORMAT] ➕ Adding', missingBraces, 'missing closing braces');
                                    for (let i = 0; i < missingBraces; i++) {
                                      fixedOutput += '}';
                                    }
                                  }
                                  
                                  // Try to close incomplete string values
                                  if (fixedOutput.match(/"[^"]*$/)) {
                                    console.log('[JSON-FORMAT] 🔤 Closing incomplete string');
                                    fixedOutput += '"';
                                  }
                                }
                                
                                console.log('[JSON-FORMAT] 🔧 Final fixed output:', fixedOutput.substring(0, 200) + '...');
                                
                                const parsed = JSON.parse(fixedOutput);
                                console.log('[JSON-FORMAT] ✅ Fixed JSON parsed successfully');
                                console.log('[JSON-FORMAT] 📦 Fixed object keys:', Object.keys(parsed));
                                formattedOutput = JSON.stringify(parsed, null, 2);
                                isJson = true;
                                console.log('[JSON-FORMAT] ✅ Truncated JSON fixed and formatted');
                                console.log('[JSON-FORMAT] 📏 Final formatted length:', formattedOutput.length);
                              } catch (e2) {
                                console.log('[JSON-FORMAT] ❌ Failed to fix truncated JSON:', e2 instanceof Error ? e2.message : String(e2));
                                console.log('[JSON-FORMAT] 🔍 Fix error details:', {
                                  name: e2 instanceof Error ? e2.name : 'Unknown',
                                  message: e2 instanceof Error ? e2.message : String(e2)
                                });
                                // Show raw output with truncation warning
                                formattedOutput = output + '\n\n⚠️ Note: This JSON appears to be truncated. Full output may be available in pod logs.';
                                console.log('[JSON-FORMAT] ⚠️ Using raw output with warning');
                              }
                            } else {
                              console.log('[JSON-FORMAT] 🔄 Using raw output as-is (not truncated)');
                            }
                          }
                        } else {
                          console.log('[JSON-FORMAT] 🔄 Output does not appear to be JSON, using as-is');
                          console.log('[JSON-FORMAT] 🔍 Output type:', typeof output);
                          console.log('[JSON-FORMAT] 🔍 Starts with {:', output?.startsWith?.('{'));
                        }
                        
                        return (
                          <pre style={{
                            textAlign: 'left',
                            backgroundColor: isJson ? '#f8f9fa' : '#f5f5f5',
                            padding: '10px',
                            borderRadius: '4px',
                            overflow: 'auto',
                            fontSize: '11px',
                            maxHeight: '400px',
                            wordBreak: isJson ? 'normal' : 'break-all',
                            whiteSpace: 'pre-wrap',
                            border: isJson ? '1px solid #e9ecef' : 'none',
                            fontFamily: isJson ? 'Monaco, Menlo, "Ubuntu Mono", monospace' : 'inherit',
                            lineHeight: '1.4',
                            maxWidth: '100%'
                          }}>
                            {formattedOutput}
                          </pre>
                        );
                      })()}
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

