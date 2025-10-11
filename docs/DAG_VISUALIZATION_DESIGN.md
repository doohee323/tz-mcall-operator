# DAG Visualization Design

**Version**: 1.0  
**Date**: 2025-10-10  
**Status**: 🚧 In Progress

## 📋 Overview

McallWorkflow를 DAG (Directed Acyclic Graph) 형태로 시각화하여 workflow의 진행 상황을 직관적으로 모니터링하고 디버깅할 수 있도록 합니다.

## 🎯 Goals

1. **실시간 모니터링** - Workflow 실행 상태를 실시간으로 시각화
2. **직관적 디버깅** - Task 간 의존성과 데이터 흐름을 명확히 표시
3. **상세 정보 제공** - 각 Task의 input/output, 실행 시간, 에러 정보 표시
4. **오픈소스 활용** - 검증된 오픈소스 라이브러리 사용

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

#### Step 1.1: API Types 확장
- [ ] `api/v1/mcallworkflow_types.go`에 DAG 구조 추가
- [ ] DeepCopy 함수 생성
- [ ] CRD 재생성

```bash
# Generate code
make generate-objects
make generate-crds

# Test
make build
```

#### Step 1.2: Controller DAG 생성 로직
- [ ] `controller/mcallworkflow_controller.go`에 `buildWorkflowDAG()` 함수 추가
- [ ] Task 상태 변경 시 DAG 업데이트
- [ ] Auto-layout 알고리즘 구현 (simple layered layout)

#### Step 1.3: 로컬 테스트
```bash
# Apply updated CRDs
kubectl apply -f crds/

# Test with existing workflow
kubectl get mcallworkflow health-monitor -n mcall-dev -o jsonpath='{.status.dag}' | jq
```

### Phase 2: Backend - MCP Server API (2 hours)

#### Step 2.1: REST API 엔드포인트
- [ ] `mcp-server/src/dag-api.ts` 생성
- [ ] GET `/api/workflows/:namespace/:name/dag` 구현
- [ ] GET `/api/workflows/:namespace/:name` 구현 (workflow info)

#### Step 2.2: WebSocket 실시간 업데이트
- [ ] `mcp-server/src/dag-websocket.ts` 생성
- [ ] Kubernetes Watch API 연동
- [ ] WebSocket으로 클라이언트에 push

#### Step 2.3: 로컬 테스트
```bash
cd mcp-server
npm install
npm run dev

# Test API
curl http://localhost:3000/api/workflows/mcall-dev/health-monitor/dag
```

### Phase 3: Frontend - React UI (3-4 hours)

#### Step 3.1: 프로젝트 설정
- [ ] `mcp-server/ui/` 디렉토리 생성
- [ ] React + Vite 프로젝트 초기화
- [ ] ReactFlow, Socket.IO 클라이언트 설치

```bash
cd mcp-server
npm create vite@latest ui -- --template react-ts
cd ui
npm install reactflow socket.io-client @tanstack/react-query
```

#### Step 3.2: DAG 컴포넌트 구현
- [ ] `WorkflowDAG.tsx` - 메인 DAG 뷰
- [ ] `CustomNode.tsx` - Task 노드 (상태별 색상)
- [ ] `NodeDetailsPanel.tsx` - Task 상세 정보
- [ ] `WorkflowHeader.tsx` - Workflow 정보 헤더

#### Step 3.3: 스타일링
- [ ] Tailwind CSS 설정
- [ ] 상태별 색상 테마
- [ ] 반응형 레이아웃

### Phase 4: Integration & Testing (2 hours)

#### Step 4.1: 로컬 통합 테스트
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

#### Step 4.2: E2E 시나리오 테스트
- [ ] Workflow 생성 → DAG 표시 확인
- [ ] Task 실행 → 실시간 상태 업데이트 확인
- [ ] Task 완료 → Duration 표시 확인
- [ ] Task 실패 → 에러 정보 표시 확인
- [ ] Conditional edges → 조건부 실행 시각화 확인

### Phase 5: Deployment (1 hour)

#### Step 5.1: Docker 이미지 빌드
- [ ] `mcp-server/Dockerfile` 수정 (UI 포함)
- [ ] Multi-stage build로 최적화

#### Step 5.2: Helm 차트 업데이트
- [ ] `values-dev.yaml`에 DAG UI 설정 추가
- [ ] Ingress 경로 추가

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

- [ ] Workflow 편집 기능 (Drag & Drop)
- [ ] Workflow 템플릿 라이브러리
- [ ] 성능 메트릭 그래프 (실행 시간 추이)
- [ ] 알림 설정 (Slack, Email)
- [ ] Workflow 비교 (Diff View)
- [ ] Export (PNG, SVG, PDF)


