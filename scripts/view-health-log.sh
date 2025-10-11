#!/bin/bash

# Health Monitor 로그 조회 스크립트

echo "=== Health Monitor 로그 조회 ==="
echo ""
echo "📊 Workflow 상태:"
WF_STATUS=$(kubectl get mcallworkflow health-monitor -n mcall-dev -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
echo "  상태: $WF_STATUS"
echo ""

echo "📝 최근 로그:"
kubectl get mcalltask health-monitor-log-result -n mcall-dev -o jsonpath='{.status.result.output}' 2>/dev/null || echo "  아직 실행되지 않았습니다. 잠시 후 다시 확인하세요."
echo ""
echo ""

echo "📈 Task 실행 이력:"
kubectl get mcalltasks -n mcall-dev -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,TIME:.status.completionTime 2>/dev/null | grep -v template || echo "  실행 이력 없음"





