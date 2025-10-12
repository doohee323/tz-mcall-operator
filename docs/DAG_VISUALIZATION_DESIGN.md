# DAG Visualization Design

**Version**: 1.0  
**Date**: 2025-10-10  
**Status**: 🚧 In Progress

## 📋 Overview

Visualize McallWorkflow in DAG (Directed Acyclic Graph) format to intuitively monitor workflow progress and enable debugging.

## 🎯 Goals

1. **Real-time Monitoring** - Visualize workflow execution status in real-time
2. **Intuitive Debugging** - Clearly display dependencies and data flow between tasks
3. **Detailed Information** - Show each task's input/output, execution time, and error information
4. **Open Source Utilization** - Use proven open source libraries

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    McallWorkflow CRD                        │
│  spec.tasks[] + status.dag (DAG metadata)                   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Workflow Controller                             │
│  - Build DAG from spec.tasks                                │
│  - Update DAG status in real-time                           │
│  - Calculate node positions (auto-layout)                   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              MCP Server (HTTP API)                           │
│  GET  /api/workflows/:namespace/:name/dag                   │
│  WS   /api/workflows/:namespace/:name/watch                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│           React UI (ReactFlow)                               │
│  - DAG visualization                                         │
│  - Real-time updates via WebSocket                          │
│  - Node details panel                                       │
│  - Zoom/Pan/Fit controls                                    │
└─────────────────────────────────────────────────────────────┘
```

## 📊 Data Structure

### 1. CRD Status Extension

```go
// api/v1/mcallworkflow_types.go

type McallWorkflowStatus struct {
    // ... existing fields ...
    
    // DAG representation for UI
    DAG *WorkflowDAG `json:"dag,omitempty"`
}

type WorkflowDAG struct {
    Nodes    []DAGNode `json:"nodes"`
    Edges    []DAGEdge `json:"edges"`
    Layout   string    `json:"layout,omitempty"` // "dagre", "elk", "auto"
    Metadata DAGMetadata `json:"metadata,omitempty"`
}

type DAGNode struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Type        string            `json:"type"` // "cmd", "get", "post"
    Phase       McallTaskPhase    `json:"phase"`
    StartTime   *metav1.Time      `json:"startTime,omitempty"`
    EndTime     *metav1.Time      `json:"endTime,omitempty"`
    Duration    string            `json:"duration,omitempty"`
    
    // Task details
    Input       string            `json:"input,omitempty"`
    Output      string            `json:"output,omitempty"`
    ErrorCode   string            `json:"errorCode,omitempty"`
    ErrorMsg    string            `json:"errorMessage,omitempty"`
    
    // UI positioning
    Position    *NodePosition     `json:"position,omitempty"`
    
    // Metadata
    Retries     int32             `json:"retries,omitempty"`
    TaskRef     string            `json:"taskRef,omitempty"`
}

type DAGEdge struct {
    ID        string `json:"id"`
    Source    string `json:"source"`
    Target    string `json:"target"`
    Type      string `json:"type,omitempty"` // "dependency", "success", "failure", "always"
    Condition string `json:"condition,omitempty"`
    Label     string `json:"label,omitempty"`
}

type NodePosition struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}

type DAGMetadata struct {
    TotalNodes    int    `json:"totalNodes"`
    TotalEdges    int    `json:"totalEdges"`
    SuccessCount  int    `json:"successCount"`
    FailureCount  int    `json:"failureCount"`
    RunningCount  int    `json:"runningCount"`
    PendingCount  int    `json:"pendingCount"`
    SkippedCount  int    `json:"skippedCount"`
}
```

### 2. API Response Format

```json
{
  "workflow": {
    "name": "health-monitor",
    "namespace": "mcall-dev",
    "phase": "Running",
    "startTime": "2025-10-10T20:00:00Z"
  },
  "dag": {
    "nodes": [
      {
        "id": "healthcheck",
        "name": "healthcheck",
        "type": "get",
        "phase": "Succeeded",
        "startTime": "2025-10-10T20:00:01Z",
        "endTime": "2025-10-10T20:00:02Z",
        "duration": "1.2s",
        "input": "https://us.drillquiz.com/aaa",
        "output": "<!doctype html>...",
        "errorCode": "0",
        "position": { "x": 250, "y": 100 },
        "taskRef": "health-check-template"
      },
      {
        "id": "log-success",
        "name": "log-success",
        "type": "cmd",
        "phase": "Running",
        "startTime": "2025-10-10T20:00:03Z",
        "position": { "x": 100, "y": 250 }
      },
      {
        "id": "log-failure",
        "name": "log-failure",
        "type": "cmd",
        "phase": "Skipped",
        "position": { "x": 400, "y": 250 }
      }
    ],
    "edges": [
      {
        "id": "healthcheck-log-success",
        "source": "healthcheck",
        "target": "log-success",
        "type": "success",
        "label": "when: success"
      },
      {
        "id": "healthcheck-log-failure",
        "source": "healthcheck",
        "target": "log-failure",
        "type": "failure",
        "label": "when: failure"
      }
    ],
    "metadata": {
      "totalNodes": 3,
      "totalEdges": 2,
      "successCount": 1,
      "runningCount": 1,
      "skippedCount": 1
    }
  }
}
```

## 🔧 Implementation Plan

### Phase 1: Backend - CRD & Controller (2-3 hours)

#### Step 1.1: Extend API Types
- [ ] Add DAG structure to `api/v1/mcallworkflow_types.go`
- [ ] Generate DeepCopy functions
- [ ] Regenerate CRDs

```bash
# Generate code
make generate-objects
make generate-crds

# Test
make build
```

#### Step 1.2: Controller DAG Generation Logic
- [ ] Add `buildWorkflowDAG()` function to `controller/mcallworkflow_controller.go`
- [ ] Update DAG when task state changes
- [ ] Implement auto-layout algorithm (simple layered layout)

#### Step 1.3: Local Testing
```bash
# Apply updated CRDs
kubectl apply -f crds/

# Test with existing workflow
kubectl get mcallworkflow health-monitor -n mcall-dev -o jsonpath='{.status.dag}' | jq
```

### Phase 2: Backend - MCP Server API (2 hours)

#### Step 2.1: REST API Endpoints
- [ ] Create `mcp-server/src/dag-api.ts`
- [ ] Implement GET `/api/workflows/:namespace/:name/dag`
- [ ] Implement GET `/api/workflows/:namespace/:name` (workflow info)

#### Step 2.2: WebSocket Real-time Updates
- [ ] Create `mcp-server/src/dag-websocket.ts`
- [ ] Integrate with Kubernetes Watch API
- [ ] Push to clients via WebSocket

#### Step 2.3: Local Testing
```bash
cd mcp-server
npm install
npm run dev

# Test API
curl http://localhost:3000/api/workflows/mcall-dev/health-monitor/dag
```

### Phase 3: Frontend - React UI (3-4 hours)

#### Step 3.1: Project Setup
- [ ] Create `mcp-server/ui/` directory
- [ ] Initialize React + Vite project
- [ ] Install ReactFlow and Socket.IO client

```bash
cd mcp-server
npm create vite@latest ui -- --template react-ts
cd ui
npm install reactflow socket.io-client @tanstack/react-query
```

#### Step 3.2: Implement DAG Components
- [ ] `WorkflowDAG.tsx` - Main DAG view
- [ ] `CustomNode.tsx` - Task nodes (status-based colors)
- [ ] `NodeDetailsPanel.tsx` - Task details
- [ ] `WorkflowHeader.tsx` - Workflow info header

#### Step 3.3: Styling
- [ ] Configure Tailwind CSS
- [ ] Status-based color theme
- [ ] Responsive layout

### Phase 4: Integration & Testing (2 hours)

#### Step 4.1: Local Integration Testing
```bash
# Terminal 1: Run controller locally
cd /Users/dhong/workspaces/tz-mcall-operator
make build
./bin/controller

# Terminal 2: Run MCP server with UI
cd mcp-server
npm run dev

# Terminal 3: Apply test workflow
kubectl apply -f examples/health-monitor-workflow-with-result-passing.yaml

# Browser: Open http://localhost:3000
```

#### Step 4.2: E2E Scenario Testing
- [ ] Workflow creation → Verify DAG display
- [ ] Task execution → Verify real-time status updates
- [ ] Task completion → Verify duration display
- [ ] Task failure → Verify error information display
- [ ] Conditional edges → Verify conditional execution visualization

### Phase 5: Deployment (1 hour)

#### Step 5.1: Build Docker Image
- [ ] Modify `mcp-server/Dockerfile` (include UI)
- [ ] Optimize with multi-stage build

#### Step 5.2: Update Helm Chart
- [ ] Add DAG UI configuration to `values-dev.yaml`
- [ ] Add ingress path

```yaml
mcpServer:
  dagUI:
    enabled: true
    port: 3000
  ingress:
    hosts:
      - host: mcall-dag.drillquiz.com
        paths:
          - path: /
            pathType: Prefix
```

## 🎨 UI Design

### Main View

```
┌────────────────────────────────────────────────────────────────┐
│ 🔄 Health Monitor Workflow          Phase: Running  ⏱ 2m 15s  │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Controls: [🔍 Zoom In] [🔍 Zoom Out] [⬜ Fit View] [↻ Refresh]│
│                                                                 │
│           ┌────────────────────┐                               │
│           │   healthcheck      │                               │
│           │   GET request      │ ✅ Succeeded (1.2s)          │
│           │   200 OK           │                               │
│           └─────────┬──────────┘                               │
│                     │                                           │
│          ┌──────────┴───────────┐                              │
│     success│                    │failure                       │
│          ▼                      ▼                              │
│  ┌───────────────┐      ┌──────────────┐                      │
│  │ log-success   │      │ log-failure  │                      │
│  │ CMD           │ 🔵   │ CMD          │ ⚪ Skipped           │
│  │ Running       │      │              │                      │
│  └───────────────┘      └──────────────┘                      │
│                                                                 │
│  Stats: ✅ 1 Success  🔵 1 Running  ⚪ 1 Skipped              │
└────────────────────────────────────────────────────────────────┘

┌─ Node Details: log-success ─────────────────────────────────┐
│                                                               │
│ Status: 🔵 Running                                           │
│ Started: 2025-10-10 20:00:03                                 │
│ Duration: 12s (running)                                      │
│                                                               │
│ Input Template:                                              │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ echo "[$(date)] ✅ SUCCESS"                              │ │
│ │ echo "  us.drillquiz.com is UP"                          │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                               │
│ Variables:                                                    │
│ • HEALTH_PHASE: Succeeded                                    │
│ • ERROR_CODE: 0                                              │
│                                                               │
│ [View Full Output] [View Logs]                               │
└───────────────────────────────────────────────────────────────┘
```

### Color Scheme

| Status    | Color       | Icon |
|-----------|-------------|------|
| Pending   | Gray #9e9e9e | ⚪ |
| Running   | Blue #2196f3 | 🔵 |
| Succeeded | Green #4caf50 | ✅ |
| Failed    | Red #f44336  | 🔴 |
| Skipped   | Light Gray #e0e0e0 | ⏭️ |

### Edge Types

| Type       | Style        | Label |
|------------|--------------|-------|
| dependency | Solid line   | -     |
| success    | Solid green  | ✓     |
| failure    | Dashed red   | ✗     |
| always     | Solid gray   | *     |

## 🔌 Tech Stack

### Backend
- **Go 1.18+** - Controller
- **Node.js 20** - MCP Server
- **Express** - HTTP API
- **Socket.IO** - WebSocket
- **@kubernetes/client-node** - K8s API

### Frontend
- **React 18** - UI Framework
- **TypeScript** - Type Safety
- **ReactFlow** - DAG Visualization
- **Vite** - Build Tool
- **Tailwind CSS** - Styling
- **@tanstack/react-query** - Data Fetching

### DevOps
- **Docker** - Containerization
- **Helm** - Kubernetes Deployment
- **Nginx** - Static File Serving

## 📝 Testing Strategy

### Unit Tests
- [ ] DAG 생성 로직 테스트
- [ ] Auto-layout 알고리즘 테스트
- [ ] API endpoint 테스트

### Integration Tests
- [ ] Controller → CRD Status 업데이트
- [ ] MCP Server → Kubernetes API 연동
- [ ] WebSocket 실시간 업데이트

### E2E Tests
- [ ] Workflow 생성부터 완료까지 전체 플로우
- [ ] 다양한 workflow 패턴 (parallel, conditional, etc.)

## 🚀 Milestones

| Phase | Description | ETA | Status |
|-------|-------------|-----|--------|
| 1 | Backend - CRD & Controller | 3h | 🚧 |
| 2 | Backend - MCP Server API | 2h | ⏳ |
| 3 | Frontend - React UI | 4h | ⏳ |
| 4 | Integration & Testing | 2h | ⏳ |
| 5 | Deployment | 1h | ⏳ |

**Total ETA**: ~12 hours

## 📚 References

- [ReactFlow Documentation](https://reactflow.dev/)
- [Argo Workflows UI](https://github.com/argoproj/argo-workflows)
- [Dagre Layout Algorithm](https://github.com/dagrejs/dagre)
- [Kubernetes Watch API](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)

## 🔄 Future Enhancements

- [ ] Workflow editing (Drag & Drop)
- [ ] Workflow template library
- [ ] Performance metrics graphs (execution time trends)
- [ ] Notification settings (Slack, Email)
- [ ] Workflow comparison (Diff View)
- [ ] Export (PNG, SVG, PDF)



