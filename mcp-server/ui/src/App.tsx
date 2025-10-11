import { useState } from 'react';
import { WorkflowDAG } from './WorkflowDAG';
import './App.css';

function App() {
  const [namespace, setNamespace] = useState('mcall-dev');
  const [workflowName, setWorkflowName] = useState('health-monitor');

  // 바로 DAG 화면으로 이동
  return <WorkflowDAG namespace={namespace} workflowName={workflowName} />;
}

export default App;
