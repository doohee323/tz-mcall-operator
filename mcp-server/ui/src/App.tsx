import { useState } from 'react';
import { WorkflowDAG } from './WorkflowDAG';
import { ApiTester } from './ApiTester';
import { McpToolsDoc } from './McpToolsDoc';
import './App.css';

type Page = 'dag' | 'api-tester' | 'mcp-tools';

function App() {
  const namespace = 'mcall-dev';
  const workflowName = 'health-monitor';
  const [currentPage, setCurrentPage] = useState<Page>('dag');

  return (
    <div style={{ 
      width: '100%', 
      height: '100vh', 
      display: 'flex', 
      flexDirection: 'column'
    }}>
      {/* Navigation */}
      <div style={{
        display: 'flex',
        background: '#0d47a1',
        padding: '0',
        gap: '0'
      }}>
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

      {/* Content */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {currentPage === 'dag' && <WorkflowDAG namespace={namespace} workflowName={workflowName} />}
        {currentPage === 'api-tester' && <ApiTester />}
        {currentPage === 'mcp-tools' && <McpToolsDoc />}
      </div>
    </div>
  );
}

export default App;
