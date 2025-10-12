import { useState, useEffect } from 'react';
import { WorkflowDAG } from './WorkflowDAG';
import { ApiTester } from './ApiTester';
import { McpToolsDoc } from './McpToolsDoc';
import './App.css';

type Page = 'dag' | 'api-tester' | 'mcp-tools';

function App() {
  // Load API key from localStorage
  const [apiKey, setApiKey] = useState(() => localStorage.getItem('mcp-api-key') || '');
  const [currentPage, setCurrentPage] = useState<Page>('dag');
  const [showConfig, setShowConfig] = useState(!apiKey);

  // Save API key to localStorage when it changes
  useEffect(() => {
    if (apiKey) {
      localStorage.setItem('mcp-api-key', apiKey);
      (window as any).MCP_API_KEY = apiKey;
    }
  }, [apiKey]);

  return (
    <div style={{ 
      width: '100%', 
      height: '100vh', 
      display: 'flex', 
      flexDirection: 'column'
    }}>
      {/* Configuration Panel */}
      {showConfig && (
        <div style={{
          padding: '20px',
          background: '#f5f5f5',
          borderBottom: '2px solid #ddd'
        }}>
          <h3 style={{ margin: '0 0 15px 0' }}>⚙️ API Configuration</h3>
          <div style={{ display: 'grid', gap: '10px', maxWidth: '600px' }}>
            <div>
              <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                🔑 API Key (Required)
              </label>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter your API Key (e.g., DevOps!323-1)"
                style={{
                  width: '100%',
                  padding: '10px',
                  fontSize: '14px',
                  border: '1px solid #ccc',
                  borderRadius: '4px'
                }}
              />
              <small style={{ color: '#666', fontSize: '12px', marginTop: '5px', display: 'block' }}>
                💡 API key is saved to browser localStorage automatically
              </small>
            </div>
            <button
              onClick={() => setShowConfig(false)}
              disabled={!apiKey}
              style={{
                padding: '10px',
                background: apiKey ? '#4caf50' : '#ccc',
                color: '#fff',
                border: 'none',
                borderRadius: '4px',
                cursor: apiKey ? 'pointer' : 'not-allowed',
                fontWeight: 'bold'
              }}
            >
              ✅ Continue
            </button>
          </div>
        </div>
      )}

      {/* Navigation */}
      <div style={{
        display: 'flex',
        background: '#0d47a1',
        padding: '0',
        gap: '0',
        justifyContent: 'space-between'
      }}>
        <div style={{ display: 'flex' }}>
          <button
            onClick={() => setCurrentPage('dag')}
            style={{
              padding: '12px 24px',
              background: currentPage === 'dag' ? '#1976d2' : 'transparent',
              color: '#fff',
              border: 'none',
              borderBottom: currentPage === 'dag' ? '3px solid #fff' : '3px solid transparent',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: currentPage === 'dag' ? 'bold' : 'normal',
              transition: 'all 0.3s'
            }}
          >
            🔄 Workflow DAG
          </button>
        <button
          onClick={() => setCurrentPage('api-tester')}
          style={{
            padding: '12px 24px',
            background: currentPage === 'api-tester' ? '#1976d2' : 'transparent',
            color: '#fff',
            border: 'none',
            borderBottom: currentPage === 'api-tester' ? '3px solid #fff' : '3px solid transparent',
            cursor: 'pointer',
            fontSize: '14px',
            fontWeight: currentPage === 'api-tester' ? 'bold' : 'normal',
            transition: 'all 0.3s'
          }}
        >
          🧪 API Tester
        </button>
        <button
          onClick={() => setCurrentPage('mcp-tools')}
          style={{
            padding: '12px 24px',
            background: currentPage === 'mcp-tools' ? '#1976d2' : 'transparent',
            color: '#fff',
            border: 'none',
            borderBottom: currentPage === 'mcp-tools' ? '3px solid #fff' : '3px solid transparent',
            cursor: 'pointer',
            fontSize: '14px',
            fontWeight: currentPage === 'mcp-tools' ? 'bold' : 'normal',
            transition: 'all 0.3s'
          }}
        >
          📚 MCP Tools
        </button>
        </div>
        <button
          onClick={() => setShowConfig(!showConfig)}
          style={{
            padding: '12px 24px',
            background: 'transparent',
            color: '#fff',
            border: 'none',
            cursor: 'pointer',
            fontSize: '14px'
          }}
          title="Show/Hide Configuration"
        >
          ⚙️ Config
        </button>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {currentPage === 'dag' && <WorkflowDAG namespace="mcall-dev" workflowName="health-monitor" />}
        {currentPage === 'api-tester' && <ApiTester />}
        {currentPage === 'mcp-tools' && <McpToolsDoc />}
      </div>
    </div>
  );
}

export default App;
