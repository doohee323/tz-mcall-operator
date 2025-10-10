import { useState } from 'react';
import { WorkflowDAG } from './WorkflowDAG';
import './App.css';

function App() {
  const [namespace, setNamespace] = useState('mcall-dev');
  const [workflowName, setWorkflowName] = useState('health-monitor');
  const [viewing, setViewing] = useState(false);

  const handleView = () => {
    setViewing(true);
  };

  if (viewing) {
    return <WorkflowDAG namespace={namespace} workflowName={workflowName} />;
  }

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      color: '#fff',
      fontFamily: 'system-ui, -apple-system, sans-serif'
    }}>
      <div style={{
        background: 'rgba(255, 255, 255, 0.95)',
        padding: '40px',
        borderRadius: '12px',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)',
        maxWidth: '500px',
        width: '90%',
        color: '#333'
      }}>
        <h1 style={{ 
          margin: '0 0 10px 0', 
          fontSize: '32px',
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent'
        }}>
          🔄 Workflow DAG Viewer
        </h1>
        <p style={{ margin: '0 0 30px 0', color: '#666', fontSize: '14px' }}>
          Visualize McallWorkflow execution in real-time
        </p>

        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: '500' }}>
            Namespace:
          </label>
          <input
            type="text"
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            style={{
              width: '100%',
              padding: '10px',
              borderRadius: '6px',
              border: '1px solid #ddd',
              fontSize: '14px',
              boxSizing: 'border-box'
            }}
            placeholder="e.g., mcall-dev"
          />
        </div>

        <div style={{ marginBottom: '30px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: '500' }}>
            Workflow Name:
          </label>
          <input
            type="text"
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
            style={{
              width: '100%',
              padding: '10px',
              borderRadius: '6px',
              border: '1px solid #ddd',
              fontSize: '14px',
              boxSizing: 'border-box'
            }}
            placeholder="e.g., health-monitor"
          />
        </div>

        <button
          onClick={handleView}
          disabled={!namespace || !workflowName}
          style={{
            width: '100%',
            padding: '12px',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            color: '#fff',
            border: 'none',
            borderRadius: '6px',
            fontSize: '16px',
            fontWeight: 'bold',
            cursor: namespace && workflowName ? 'pointer' : 'not-allowed',
            opacity: namespace && workflowName ? 1 : 0.6,
            transition: 'all 0.3s'
          }}
        >
          View DAG →
        </button>

        <div style={{ marginTop: '20px', fontSize: '12px', color: '#999', textAlign: 'center' }}>
          💡 Tip: Workflow will update in real-time via WebSocket
        </div>
      </div>

      <div style={{ marginTop: '30px', fontSize: '12px', opacity: 0.8 }}>
        Powered by tz-mcall-operator | ReactFlow
      </div>
    </div>
  );
}

export default App;
