#!/bin/bash

# Health Monitor Log Viewer Script

echo "=== Health Monitor Log Viewer ==="
echo ""
echo "📊 Workflow Status:"
WF_STATUS=$(kubectl get mcallworkflow health-monitor -n mcall-dev -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
echo "  Status: $WF_STATUS"
echo ""

echo "📝 Recent Logs:"
kubectl get mcalltask health-monitor-log-result -n mcall-dev -o jsonpath='{.status.result.output}' 2>/dev/null || echo "  Not executed yet. Please check again later."
echo ""
echo ""

echo "📈 Task Execution History:"
kubectl get mcalltasks -n mcall-dev -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,TIME:.status.completionTime 2>/dev/null | grep -v template || echo "  No execution history"
