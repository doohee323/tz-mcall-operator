import { useState, useEffect, useCallback } from 'react';

interface ApiEndpoint {
  name: string;
  method: 'GET' | 'POST' | 'DELETE';
  path: string;
  description: string;
  params?: { name: string; type: 'path' | 'query'; required: boolean; description: string }[];
  bodyExample?: string;
}

const API_ENDPOINTS: ApiEndpoint[] = [
  {
    name: 'List Namespaces',
    method: 'GET',
    path: '/api/namespaces',
    description: 'Get available namespaces with McallWorkflow resources'
  },
  {
    name: 'List Workflows',
    method: 'GET',
    path: '/api/workflows/{namespace}',
    description: 'List all workflows in a specific namespace',
    params: [
      { name: 'namespace', type: 'path', required: true, description: 'Namespace name (e.g., mcall-dev)' }
    ]
  },
  {
    name: 'Get Workflow DAG',
    method: 'GET',
    path: '/api/workflows/{namespace}/{name}/dag',
    description: 'Get workflow DAG visualization data',
    params: [
      { name: 'namespace', type: 'path', required: true, description: 'Namespace name' },
      { name: 'name', type: 'path', required: true, description: 'Workflow name' }
    ]
  },
  {
    name: 'List Tasks',
    method: 'GET',
    path: '/api/tasks/{namespace}',
    description: 'List all tasks in a specific namespace',
    params: [
      { name: 'namespace', type: 'path', required: true, description: 'Namespace name (e.g., mcall-dev)' }
    ]
  },
  {
    name: 'Get Task Details',
    method: 'GET',
    path: '/api/tasks/{namespace}/{name}',
    description: 'Get detailed information about a task',
    params: [
      { name: 'namespace', type: 'path', required: true, description: 'Namespace name' },
      { name: 'name', type: 'path', required: true, description: 'Task name' }
    ]
  }
];

export function ApiTester() {
  const [selectedEndpoint, setSelectedEndpoint] = useState<ApiEndpoint>(API_ENDPOINTS[0]);
  const [pathParams, setPathParams] = useState<Record<string, string>>({});
  const [response, setResponse] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Available options for dropdowns
  const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);
  const [availableWorkflows, setAvailableWorkflows] = useState<Record<string, string[]>>({});
  const [availableTasks, setAvailableTasks] = useState<Record<string, string[]>>({});

  // Fetch available namespaces on mount
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
        const res = await fetch(`${apiUrl}/api/namespaces`);
        const data = await res.json();
        if (data.success && data.namespaces) {
          setAvailableNamespaces(data.namespaces);
          // Set default namespace
          if (data.namespaces.length > 0) {
            const defaultNs = data.namespaces[0];
            setPathParams(prev => {
              // Only set if not already set
              if (!prev.namespace) {
                return { ...prev, namespace: defaultNs };
              }
              return prev;
            });
          }
        }
      } catch (err) {
        console.error('[API-TESTER] Error fetching namespaces:', err);
      }
    };
    fetchNamespaces();
  }, []);

  // Fetch workflows when namespace changes
  const fetchWorkflows = useCallback(async (namespace: string) => {
    if (!namespace) return;
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const res = await fetch(`${apiUrl}/api/workflows/${namespace}`);
      const data = await res.json();
      console.log('[API-TESTER] Fetched workflows for namespace', namespace, ':', data);
      if (data.success && data.workflows) {
        setAvailableWorkflows(prev => {
          const updated = { ...prev, [namespace]: data.workflows };
          console.log('[API-TESTER] Updated availableWorkflows:', updated);
          return updated;
        });
        // Set default workflow if none selected
        if (data.workflows.length > 0) {
          setPathParams(prev => {
            // Only set if not already set or if namespace changed (cleared the name)
            if (!prev.name) {
              console.log('[API-TESTER] Setting default workflow:', data.workflows[0]);
              return { ...prev, name: data.workflows[0] };
            }
            return prev;
          });
        }
      }
    } catch (err) {
      console.error('[API-TESTER] Error fetching workflows:', err);
    }
  }, []);

  // Fetch workflows when namespace param changes
  useEffect(() => {
    if (pathParams.namespace) {
      fetchWorkflows(pathParams.namespace);
      // Also fetch tasks for this namespace
      fetchTasks(pathParams.namespace);
    }
  }, [pathParams.namespace, fetchWorkflows]);

  // Fetch tasks when namespace changes
  const fetchTasks = useCallback(async (namespace: string) => {
    if (!namespace) return;
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      const res = await fetch(`${apiUrl}/api/tasks/${namespace}`);
      const data = await res.json();
      console.log('[API-TESTER] Fetched tasks for namespace', namespace, ':', data);
      if (data.success && data.tasks) {
        setAvailableTasks(prev => {
          const updated = { ...prev, [namespace]: data.tasks };
          console.log('[API-TESTER] Updated availableTasks:', updated);
          return updated;
        });
      }
    } catch (err) {
      console.error('[API-TESTER] Error fetching tasks:', err);
    }
  }, []);

  const handleTest = async () => {
    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';
      let url = selectedEndpoint.path;
      
      // Replace path parameters
      Object.entries(pathParams).forEach(([key, value]) => {
        url = url.replace(`{${key}}`, value);
      });

      const fullUrl = `${apiUrl}${url}`;
      console.log('[API-TESTER] Testing:', selectedEndpoint.method, fullUrl);

      const res = await fetch(fullUrl, {
        method: selectedEndpoint.method,
        headers: {
          'Content-Type': 'application/json'
        }
      });

      const data = await res.json();
      setResponse({
        status: res.status,
        statusText: res.statusText,
        data: data
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <div style={{
        padding: '15px 20px',
        background: '#1976d2',
        color: '#fff'
      }}>
        <h1 style={{ margin: 0, fontSize: '24px' }}>
          🧪 MCP Server API Tester
        </h1>
        <div style={{ fontSize: '14px', marginTop: '5px', opacity: 0.9 }}>
          Test and explore MCP Server REST APIs
        </div>
      </div>

      {/* Main Content */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left Panel - API List */}
        <div style={{
          width: '350px',
          borderRight: '1px solid #ddd',
          overflow: 'auto',
          background: '#f8f9fa'
        }}>
          <div style={{ padding: '15px' }}>
            <h3 style={{ margin: '0 0 10px 0', fontSize: '16px' }}>Available APIs</h3>
            {API_ENDPOINTS.map((endpoint, idx) => (
              <div
                key={idx}
                onClick={() => {
                  setSelectedEndpoint(endpoint);
                  setPathParams({});
                  setResponse(null);
                  setError(null);
                }}
                style={{
                  padding: '12px',
                  marginBottom: '8px',
                  background: selectedEndpoint === endpoint ? '#1976d2' : '#fff',
                  color: selectedEndpoint === endpoint ? '#fff' : '#333',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  border: '1px solid #ddd',
                  transition: 'all 0.2s'
                }}
              >
                <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                  <span style={{
                    display: 'inline-block',
                    padding: '2px 6px',
                    borderRadius: '3px',
                    fontSize: '11px',
                    marginRight: '6px',
                    background: endpoint.method === 'GET' ? '#4caf50' : 
                               endpoint.method === 'POST' ? '#2196f3' : '#f44336',
                    color: '#fff'
                  }}>
                    {endpoint.method}
                  </span>
                  {endpoint.name}
                </div>
                <div style={{ fontSize: '11px', opacity: 0.8 }}>{endpoint.path}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Right Panel - Test Interface */}
        <div style={{ flex: 1, overflow: 'auto', padding: '20px' }}>
          <h2 style={{ marginTop: 0, marginBottom: '15px' }}>{selectedEndpoint.name}</h2>
          
          <div style={{ marginBottom: '20px' }}>
            <div style={{ marginBottom: '10px' }}>
              <span style={{
                display: 'inline-block',
                padding: '4px 12px',
                borderRadius: '4px',
                fontSize: '14px',
                fontWeight: 'bold',
                marginRight: '10px',
                background: selectedEndpoint.method === 'GET' ? '#4caf50' : 
                           selectedEndpoint.method === 'POST' ? '#2196f3' : '#f44336',
                color: '#fff'
              }}>
                {selectedEndpoint.method}
              </span>
              <code style={{
                background: '#f5f5f5',
                padding: '6px 12px',
                borderRadius: '4px',
                fontSize: '14px'
              }}>
                {selectedEndpoint.path}
              </code>
            </div>
            <p style={{ color: '#666', fontSize: '14px', margin: '10px 0' }}>
              {selectedEndpoint.description}
            </p>
          </div>

          {/* Parameters */}
          {selectedEndpoint.params && selectedEndpoint.params.length > 0 && (
            <div style={{ marginBottom: '20px' }}>
              <h3 style={{ fontSize: '16px', marginBottom: '10px' }}>Parameters</h3>
              {selectedEndpoint.params.map((param) => {
                // Determine if this should be a dropdown
                let useDropdown = false;
                let dropdownOptions: string[] = [];
                
                if (param.name === 'namespace' && availableNamespaces.length > 0) {
                  useDropdown = true;
                  dropdownOptions = availableNamespaces;
                } else if (param.name === 'name' && pathParams.namespace) {
                  // Check if this is workflow or task based on endpoint path
                  if (selectedEndpoint.path.includes('/tasks/')) {
                    // Task endpoint - use task list
                    const tasks = availableTasks[pathParams.namespace];
                    if (tasks && tasks.length > 0) {
                      useDropdown = true;
                      dropdownOptions = tasks;
                    }
                  } else if (selectedEndpoint.path.includes('/workflows/')) {
                    // Workflow endpoint - use workflow list
                    const workflows = availableWorkflows[pathParams.namespace];
                    if (workflows && workflows.length > 0) {
                      useDropdown = true;
                      dropdownOptions = workflows;
                    }
                  }
                }
                
                // Debug logging
                if (param.name === 'name') {
                  console.log('[API-TESTER] Rendering name param:', {
                    paramName: param.name,
                    useDropdown,
                    dropdownOptions,
                    endpointPath: selectedEndpoint.path,
                    namespace: pathParams.namespace,
                    availableWorkflows: availableWorkflows,
                    availableTasks: availableTasks
                  });
                }
                
                return (
                  <div key={param.name} style={{ 
                    marginBottom: '15px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px'
                  }}>
                    <label style={{
                      minWidth: '150px',
                      fontSize: '14px',
                      fontWeight: '500',
                      display: 'flex',
                      alignItems: 'center'
                    }}>
                      {param.name}
                      {param.required && <span style={{ color: '#f44336' }}>*</span>}
                      <span style={{ fontSize: '12px', color: '#666', marginLeft: '8px' }}>
                        ({param.type})
                      </span>
                    </label>
                    
                    {useDropdown ? (
                      <select
                        value={pathParams[param.name] || ''}
                        onChange={(e) => {
                          const newParams = { ...pathParams, [param.name]: e.target.value };
                          // If namespace changed, clear workflow/task selection
                          if (param.name === 'namespace') {
                            delete newParams.name;
                          }
                          setPathParams(newParams);
                        }}
                        style={{
                          width: '300px',
                          padding: '8px 12px',
                          border: '1px solid #ddd',
                          borderRadius: '4px',
                          fontSize: '14px',
                          boxSizing: 'border-box',
                          background: '#fff'
                        }}
                      >
                        <option value="">Select {param.name}...</option>
                        {dropdownOptions.map(option => (
                          <option key={option} value={option}>{option}</option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type="text"
                        value={pathParams[param.name] || ''}
                        onChange={(e) => setPathParams({ ...pathParams, [param.name]: e.target.value })}
                        placeholder={param.description}
                        style={{
                          width: '300px',
                          padding: '8px 12px',
                          border: '1px solid #ddd',
                          borderRadius: '4px',
                          fontSize: '14px',
                          boxSizing: 'border-box'
                        }}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {/* Test Button */}
          <button
            onClick={handleTest}
            disabled={loading}
            style={{
              padding: '12px 24px',
              background: loading ? '#ccc' : '#1976d2',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              fontSize: '16px',
              fontWeight: 'bold',
              cursor: loading ? 'not-allowed' : 'pointer',
              marginBottom: '20px'
            }}
          >
            {loading ? '⏳ Testing...' : '🚀 Send Request'}
          </button>

          {/* Error */}
          {error && (
            <div style={{
              padding: '15px',
              background: '#ffebee',
              border: '1px solid #ef9a9a',
              borderRadius: '4px',
              color: '#c62828',
              marginBottom: '20px'
            }}>
              <strong>❌ Error:</strong> {error}
            </div>
          )}

          {/* Response */}
          {response && (
            <div style={{ textAlign: 'left' }}>
              <h3 style={{ fontSize: '16px', marginBottom: '10px', textAlign: 'left' }}>Response</h3>
              <div style={{
                padding: '10px 15px',
                background: response.status >= 200 && response.status < 300 ? '#e8f5e9' : '#ffebee',
                border: `1px solid ${response.status >= 200 && response.status < 300 ? '#81c784' : '#ef9a9a'}`,
                borderRadius: '4px',
                marginBottom: '15px',
                textAlign: 'left'
              }}>
                <strong>Status:</strong> {response.status} {response.statusText}
              </div>
              <div style={{
                background: '#f5f5f5',
                border: '1px solid #ddd',
                borderRadius: '4px',
                padding: '15px',
                overflow: 'auto',
                maxHeight: '500px',
                textAlign: 'left'
              }}>
                <pre style={{
                  margin: 0,
                  fontSize: '12px',
                  lineHeight: '1.5',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  textAlign: 'left'
                }}>
                  {JSON.stringify(response.data, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Footer */}
      <div style={{
        padding: '10px 20px',
        background: '#f5f5f5',
        borderTop: '1px solid #ddd',
        fontSize: '12px',
        color: '#666',
        textAlign: 'center'
      }}>
        MCP Server API Tester | Base URL: {import.meta.env.VITE_API_URL || 'http://localhost:3000'}
      </div>
    </div>
  );
}

