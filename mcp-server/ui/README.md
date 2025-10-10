# Workflow DAG Visualization UI

React 기반 McallWorkflow DAG 시각화 도구

## 🚀 빠른 시작

### 1. 클러스터 API 사용 (개발 환경)

```bash
# 환경변수 설정
echo "VITE_API_URL=https://mcp-dev.drillquiz.com" > .env.local

# 의존성 설치
npm install

# 개발 서버 실행
npm run dev

# 브라우저에서 열기
open http://localhost:5173
```

### 2. 로컬 API 사용 (로컬 테스트)

```bash
# 환경변수 설정
echo "VITE_API_URL=http://localhost:3000" > .env.local

# MCP Server 실행 (다른 터미널에서)
cd ..
npm start

# UI 개발 서버 실행
npm run dev
```

## 📊 사용 방법

1. **Namespace 입력**: `mcall-dev`
2. **Workflow Name 입력**: `health-monitor`
3. **View DAG 클릭**
4. **자동 업데이트**: 5초마다 자동 refresh

## 🎨 기능

- ✅ **실시간 모니터링**: 5초마다 자동 업데이트
- ✅ **상태별 색상**: 
  - 🟢 Succeeded (Green)
  - 🔵 Running (Blue)  
  - 🔴 Failed (Red)
  - ⚪ Pending (Gray)
  - ⏭️ Skipped (Light Gray)
- ✅ **Task 정보**: Duration, Error Code 표시
- ✅ **조건부 엣지**: success ✓, failure ✗ 표시
- ✅ **통계**: Success/Failed/Running 개수
- ✅ **줌/팬**: ReactFlow 기본 컨트롤

## 🔧 빌드

### Production 빌드

```bash
npm run build
```

빌드된 파일은 `dist/` 폴더에 생성됩니다.

### MCP Server에서 serve

MCP Server가 자동으로 `dist/` 폴더를 serve합니다:

```bash
cd ..
npm start

# UI 접속
open http://localhost:3000
```

## 📁 프로젝트 구조

```
ui/
├── src/
│   ├── App.tsx              # 메인 앱 (입력 폼)
│   ├── WorkflowDAG.tsx      # DAG 시각화 컴포넌트
│   └── main.tsx             # Entry point
├── .env.local               # 환경변수 (git ignored)
└── package.json
```

## 🌐 환경변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `VITE_API_URL` | MCP Server API URL | `http://localhost:3000` |

## 🐛 트러블슈팅

### API 연결 실패

```bash
# MCP Server가 실행 중인지 확인
curl https://mcp-dev.drillquiz.com/health

# 또는 로컬
curl http://localhost:3000/health
```

### DAG가 비어있음

- **원인**: 클러스터 controller가 구버전
- **해결**: Jenkins 배포 완료 대기
- **확인**: 
  ```bash
  kubectl get mcallworkflow health-monitor -n mcall-dev -o jsonpath='{.status.dag}' | jq
  ```

### CORS 에러

MCP Server에 CORS 설정이 포함되어 있습니다. 만약 에러가 발생하면:

```bash
# MCP Server 재시작
cd ..
npm run build
npm start
```

## 📚 관련 문서

- [DAG 시각화 설계 문서](../../docs/DAG_VISUALIZATION_DESIGN.md)
- [MCP Server Guide](../../MCP_SERVER_GUIDE.md)
- [ReactFlow Documentation](https://reactflow.dev/)
