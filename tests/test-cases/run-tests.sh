#!/bin/bash

# McallTask & McallWorkflow Test Runner
# This script runs comprehensive tests for the McallTask and McallWorkflow CRD functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mcall-dev"
TASK_TEST_CASES_FILE="tests/test-cases/mcall-task-test-cases.yaml"
WORKFLOW_TEST_CASES_FILE="tests/test-cases/mcall-workflow-test-cases.yaml"
RESULTS_DIR="test-results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEBUG=false


# Create results directory
mkdir -p "$RESULTS_DIR"

# Function to run kubectl with debug logging
run_kubectl() {
    local cmd="$*"
    if [ "$DEBUG" = "true" ]; then
        echo -e "${YELLOW}🔧 DEBUG: kubectl $cmd${NC}"
    fi
    kubectl $cmd
}

# Function to check if namespace exists
check_namespace() {
    if ! run_kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        echo -e "${RED}❌ Namespace $NAMESPACE does not exist${NC}"
        echo "Please create the namespace first:"
        echo "kubectl create namespace $NAMESPACE"
        exit 1
    fi
    echo -e "${GREEN}✅ Namespace $NAMESPACE exists${NC}"
}

# Function to check if CRDs are installed
check_crds() {
    echo -e "${BLUE}🔍 Checking CRDs...${NC}"
    
    local crds=("mcalltasks.mcall.tz.io" "mcallworkflows.mcall.tz.io")
    local all_crds_exist=true
    
    for crd in "${crds[@]}"; do
        if run_kubectl get crd "$crd" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ $crd exists${NC}"
        else
            echo -e "${RED}❌ $crd not found${NC}"
            all_crds_exist=false
        fi
    done
    
    if [ "$all_crds_exist" = false ]; then
        echo -e "${RED}❌ Some CRDs are missing. Please install them first.${NC}"
        exit 1
    fi
}

# Function to check if controller is running
check_controller() {
    echo -e "${BLUE}🔍 Checking Controller...${NC}"
    
    local controller_pods=$(run_kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=tz-mcall-crd --no-headers | wc -l)
    
    if [ "$controller_pods" -eq 0 ]; then
        echo -e "${RED}❌ No controller pods found${NC}"
        exit 1
    fi
    
    local running_pods=$(run_kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=tz-mcall-crd --field-selector=status.phase=Running --no-headers | wc -l)
    
    if [ "$running_pods" -eq 0 ]; then
        echo -e "${RED}❌ No running controller pods found${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Controller is running ($running_pods pod(s))${NC}"
}

# Function to run a single McallTask test case
run_task_test_case() {
    local test_name="$1"
    local test_file="$2"
    
    echo -e "${YELLOW}🧪 Running McallTask test: $test_name${NC}"
    
    # Check if test case already exists
    if run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Test case already exists${NC}"
    else
        # Create a temporary file with corrected namespace
        local temp_file="/tmp/mcall-test-${test_name}.yaml"
        
        # Extract the test case and add namespace
        if awk "/name: $test_name/,/^---$/" "$test_file" | sed "1i\\
apiVersion: mcall.tz.io/v1\\
kind: McallTask\\
metadata:\\
  name: $test_name\\
  namespace: $NAMESPACE\\
" > "$temp_file"; then
            # Apply the test case with corrected namespace
            if run_kubectl apply -f "$temp_file" >/dev/null 2>&1; then
                echo -e "${GREEN}✅ Test case applied successfully${NC}"
            else
                echo -e "${RED}❌ Failed to apply test case${NC}"
                rm -f "$temp_file"
                return 1
            fi
        else
            echo -e "${RED}❌ Failed to prepare test case${NC}"
            return 1
        fi
        
        # Clean up temp file
        rm -f "$temp_file"
    fi
    
    # Wait for status to be initialized
    echo -e "${BLUE}⏳ Waiting for status initialization...${NC}"
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        local status=$(run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ -n "$status" ] && [ "$status" != "" ]; then
            echo -e "${GREEN}✅ Status initialized: $status${NC}"
            break
        fi
        
        echo -e "${BLUE}⏳ Attempt $((attempt + 1))/$max_attempts: Status not yet initialized...${NC}"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        echo -e "${RED}❌ Status initialization timeout${NC}"
        return 1
    fi
    
    # Wait for completion
    echo -e "${BLUE}⏳ Waiting for task completion...${NC}"
    # HTTP tests complete immediately after 1 second, other tests have environment-dependent wait time
    local max_wait
    if [[ "$test_name" == *"http"* ]]; then
        max_wait=3  # HTTP test: 1 second completion + 2 seconds buffer
    elif [ "$NAMESPACE" = "mcall-system" ]; then
        max_wait=10  # 10 seconds (5 seconds task completion + 5 seconds buffer)
    else
        max_wait=10  # 10 seconds (3 seconds task completion + 7 seconds buffer)
    fi
    local wait_time=0
    
    while [ $wait_time -lt $max_wait ]; do
        local status=$(run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ "$status" = "Succeeded" ]; then
            echo -e "${GREEN}✅ Task completed successfully${NC}"
            break
        elif [ "$status" = "Failed" ]; then
            echo -e "${RED}❌ Task failed${NC}"
            break
        fi
        
        echo -e "${BLUE}⏳ Status: $status (waiting... ${wait_time}s/${max_wait}s)${NC}"
        sleep 1
        wait_time=$((wait_time + 1))
    done
    
    if [ $wait_time -ge $max_wait ]; then
        echo -e "${RED}❌ Task completion timeout${NC}"
        return 1
    fi
    
    # Get final status and results
    local final_status=$(run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    local results=$(run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.results}' 2>/dev/null || echo "No results")
    
    echo -e "${BLUE}📊 Final Status: $final_status${NC}"
    echo -e "${BLUE}📊 Results: $results${NC}"
    
    # Save results
    local result_file="$RESULTS_DIR/${test_name}_${TIMESTAMP}.json"
    run_kubectl get mcalltask "$test_name" -n "$NAMESPACE" -o json > "$result_file"
    echo -e "${BLUE}💾 Results saved to: $result_file${NC}"
    
    # Clean up
    run_kubectl delete mcalltask "$test_name" -n "$NAMESPACE" >/dev/null 2>&1
    echo -e "${GREEN}✅ Test case cleaned up${NC}"
    
    echo ""
    
    # Return appropriate exit code based on final status
    if [ "$final_status" = "Succeeded" ]; then
        return 0
    else
        return 1
    fi
}

# Function to run a single McallWorkflow test case
run_workflow_test_case() {
    local test_name="$1"
    local test_file="$2"
    
    echo -e "${YELLOW}🧪 Running McallWorkflow test: $test_name${NC}"
    
    # Check if workflow test case already exists
    if run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Workflow test case already exists${NC}"
    else
        # Create a temporary file with corrected namespace
        local temp_file="/tmp/mcall-workflow-test-${test_name}.yaml"
        
        # First, create all referenced tasks before creating the workflow
        echo -e "${BLUE}⏳ Creating referenced tasks for workflow...${NC}"
        
        # Extract all McallTask resources from the test file and create them first
        # Use a simpler approach: extract complete YAML documents for McallTask
        awk '/^---$/{if(doc) print doc "---"; doc=""; next} /^apiVersion: mcall.tz.io\/v1$/{flag=1; doc=$0; next} flag && /^kind: McallTask$/{flag=2; doc=doc "\n" $0; next} flag==2{doc=doc "\n" $0} END{if(doc) print doc}' "$test_file" | \
        sed "s/namespace: mcall-system/namespace: $NAMESPACE/g" > "$temp_file"
        
        # Apply the tasks first
        if [ -s "$temp_file" ]; then
            if run_kubectl apply -f "$temp_file" >/dev/null 2>&1; then
                echo -e "${GREEN}✅ Referenced tasks created successfully${NC}"
            else
                echo -e "${YELLOW}⚠️ Some referenced tasks may already exist${NC}"
            fi
        fi
        
        # Now create the workflow
        if awk "/name: $test_name/,/^---$/" "$test_file" | sed "1i\\
apiVersion: mcall.tz.io/v1\\
kind: McallWorkflow\\
metadata:\\
  name: $test_name\\
  namespace: $NAMESPACE\\
" | sed "s/namespace: mcall-system/namespace: $NAMESPACE/g" | sed "s/namespace: \"mcall-system\"/namespace: \"$NAMESPACE\"/g" > "$temp_file"; then
            # Apply the workflow test case with corrected namespace
            if run_kubectl apply -f "$temp_file" >/dev/null 2>&1; then
                echo -e "${GREEN}✅ Workflow test case applied successfully${NC}"
            else
                echo -e "${RED}❌ Failed to apply workflow test case${NC}"
                rm -f "$temp_file"
                return 1
            fi
        else
            echo -e "${RED}❌ Failed to prepare workflow test case${NC}"
            return 1
        fi
        
        # Clean up temp file
        rm -f "$temp_file"
    fi
    
    # Wait for status to be initialized
    echo -e "${BLUE}⏳ Waiting for workflow status initialization...${NC}"
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        local status=$(run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ -n "$status" ] && [ "$status" != "" ]; then
            echo -e "${GREEN}✅ Workflow status initialized: $status${NC}"
            break
        fi
        
        echo -e "${BLUE}⏳ Attempt $((attempt + 1))/$max_attempts: Workflow status not yet initialized...${NC}"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        echo -e "${RED}❌ Workflow status initialization timeout${NC}"
        return 1
    fi
    
    # Wait for completion
    echo -e "${BLUE}⏳ Waiting for workflow completion...${NC}"
    local max_wait=20  # Workflows may take longer due to dependencies
    local wait_time=0
    
    while [ $wait_time -lt $max_wait ]; do
        local status=$(run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ "$status" = "Succeeded" ]; then
            echo -e "${GREEN}✅ Workflow completed successfully${NC}"
            break
        elif [ "$status" = "Failed" ]; then
            echo -e "${RED}❌ Workflow failed${NC}"
            break
        fi
        
        echo -e "${BLUE}⏳ Workflow Status: $status (waiting... ${wait_time}s/${max_wait}s)${NC}"
        sleep 1
        wait_time=$((wait_time + 1))
    done
    
    if [ $wait_time -ge $max_wait ]; then
        echo -e "${RED}❌ Workflow completion timeout${NC}"
        return 1
    fi
    
    # Get final status and results
    local final_status=$(run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    local task_statuses=$(run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" -o jsonpath='{.status.taskStatuses}' 2>/dev/null || echo "No task statuses")
    
    echo -e "${BLUE}📊 Final Workflow Status: $final_status${NC}"
    echo -e "${BLUE}📊 Task Statuses: $task_statuses${NC}"
    
    # Save results
    local result_file="$RESULTS_DIR/workflow_${test_name}_${TIMESTAMP}.json"
    run_kubectl get mcallworkflow "$test_name" -n "$NAMESPACE" -o json > "$result_file"
    echo -e "${BLUE}💾 Results saved to: $result_file${NC}"
    
    # Clean up workflow and related tasks
    run_kubectl delete mcallworkflow "$test_name" -n "$NAMESPACE" >/dev/null 2>&1
    # Also clean up any tasks created by the workflow
    run_kubectl delete mcalltask -l "mcall.tz.io/workflow=$test_name" -n "$NAMESPACE" >/dev/null 2>&1
    echo -e "${GREEN}✅ Workflow test case cleaned up${NC}"
    
    echo ""
    
    # Return appropriate exit code based on final status
    if [ "$final_status" = "Succeeded" ]; then
        return 0
    else
        return 1
    fi
}

# Function to run all McallTask test cases
run_all_task_tests() {
    echo -e "${BLUE}🚀 Running all McallTask test cases...${NC}"
    echo ""
    
    local test_cases=(
        "basic-command-test"
        "http-get-test"
        "complex-command-test"
        "health-check-test"
        "error-handling-test"
        "timeout-test"
        "large-output-test"
        "json-response-test"
        "multiple-http-test"
        "system-resource-test"
        "http-response-validation-test"
        "http-status-validation-test"
        "http-header-validation-test"
        "complex-response-validation-test"
        "complex-response-validation-parallel-test"
        "failfast-sequential-test"
        "failfast-parallel-test"
        "cli-output-validation-test"
        "mixed-validation-test"
        "error-response-validation-test"
        "json-schema-validation-test"
    )
    
    local passed=0
    local failed=0
    
    for test_case in "${test_cases[@]}"; do
        if run_task_test_case "$test_case" "$TASK_TEST_CASES_FILE"; then
            passed=$((passed + 1))
        else
            failed=$((failed + 1))
        fi
        echo "----------------------------------------"
    done
    
    echo -e "${BLUE}📊 McallTask Test Summary${NC}"
    echo "=========================================="
    echo -e "${GREEN}✅ Passed: $passed${NC}"
    echo -e "${RED}❌ Failed: $failed${NC}"
    echo -e "${BLUE}📁 Results saved in: $RESULTS_DIR${NC}"
    
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}🎉 All McallTask tests passed!${NC}"
        return 0
    else
        echo -e "${RED}💥 Some McallTask tests failed!${NC}"
        return 1
    fi
}

# Function to run all McallWorkflow test cases
run_all_workflow_tests() {
    echo -e "${BLUE}🚀 Running all McallWorkflow test cases...${NC}"
    echo ""
    
    local test_cases=(
        "basic-workflow-test"
        "scheduled-workflow-test"
        "complex-dependency-workflow-test"
        "health-check-workflow-test"
        "error-handling-workflow-test"
        "mixed-task-types-workflow-test"
        "long-running-workflow-test"
        "resource-intensive-workflow-test"
        "validation-workflow-test"
        "circular-dependency-workflow-test"
    )
    
    local passed=0
    local failed=0
    
    for test_case in "${test_cases[@]}"; do
        if run_workflow_test_case "$test_case" "$WORKFLOW_TEST_CASES_FILE"; then
            passed=$((passed + 1))
        else
            failed=$((failed + 1))
        fi
        echo "----------------------------------------"
    done
    
    echo -e "${BLUE}📊 McallWorkflow Test Summary${NC}"
    echo "=========================================="
    echo -e "${GREEN}✅ Passed: $passed${NC}"
    echo -e "${RED}❌ Failed: $failed${NC}"
    echo -e "${BLUE}📁 Results saved in: $RESULTS_DIR${NC}"
    
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}🎉 All McallWorkflow tests passed!${NC}"
        return 0
    else
        echo -e "${RED}💥 Some McallWorkflow tests failed!${NC}"
        return 1
    fi
}

# Function to run a specific McallTask test case
run_specific_task_test() {
    local test_name="$1"
    
    if [ -z "$test_name" ]; then
        echo -e "${RED}❌ Please specify a test name${NC}"
        echo "Available McallTask test cases:"
        run_kubectl get mcalltask -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print "  - " $1}' || echo "  No test cases found"
        exit 1
    fi
    
    echo -e "${BLUE}🎯 Running specific McallTask test: $test_name${NC}"
    echo ""
    
    if run_task_test_case "$test_name" "$TASK_TEST_CASES_FILE"; then
        echo -e "${GREEN}🎉 McallTask test passed!${NC}"
        exit 0
    else
        echo -e "${RED}💥 McallTask test failed!${NC}"
        exit 1
    fi
}

# Function to run a specific McallWorkflow test case
run_specific_workflow_test() {
    local test_name="$1"
    
    if [ -z "$test_name" ]; then
        echo -e "${RED}❌ Please specify a workflow test name${NC}"
        echo "Available McallWorkflow test cases:"
        run_kubectl get mcallworkflow -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print "  - " $1}' || echo "  No workflow test cases found"
        exit 1
    fi
    
    echo -e "${BLUE}🎯 Running specific McallWorkflow test: $test_name${NC}"
    echo ""
    
    if run_workflow_test_case "$test_name" "$WORKFLOW_TEST_CASES_FILE"; then
        echo -e "${GREEN}🎉 McallWorkflow test passed!${NC}"
        exit 0
    else
        echo -e "${RED}💥 McallWorkflow test failed!${NC}"
        exit 1
    fi
}

# Function to show help
show_help() {
    echo "McallTask & McallWorkflow CRD Test Runner"
    echo ""
    echo "Usage: $0 [OPTIONS] [TEST_NAME]"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  -t, --task           Run all McallTask test cases"
    echo "  -f, --flow           Run all McallWorkflow test cases"
    echo "  -a, --all            Run all test cases (both task and workflow)"
    echo "  -c, --check          Check system status only"
    echo "  -d, --debug          Enable debug mode (show kubectl commands)"
    echo "  --namespace NAMESPACE Specify target namespace (default: mcall-dev)"
    echo ""
    echo "Arguments:"
    echo "  TEST_NAME            Run a specific test case (auto-detect type)"
    echo ""
    echo "Examples:"
    echo "  $0 --task                                    # Run all McallTask test cases"
    echo "  $0 --flow                                    # Run all McallWorkflow test cases"
    echo "  $0 --all                                     # Run all test cases"
    echo "  $0 --namespace mcall-dev --task             # Run McallTask tests in mcall-dev"
    echo "  $0 basic-command-test                        # Run specific McallTask test"
    echo "  $0 basic-workflow-test                       # Run specific McallWorkflow test"
    echo "  $0 --namespace mcall-system --check         # Check system status in mcall-system"
    echo "  $0 --debug http-response-validation-test    # Run with debug"
}

# Main execution
main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -d|--debug)
                DEBUG=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--task)
                echo -e "${BLUE}🧪 Starting McallTask CRD Test Suite${NC}"
                echo "=========================================="
                echo "Namespace: $NAMESPACE"
                echo "Test Cases: $TASK_TEST_CASES_FILE"
                echo "Results Directory: $RESULTS_DIR"
                echo "Timestamp: $TIMESTAMP"
                echo ""
                check_namespace
                check_crds
                check_controller
                run_all_task_tests
                exit $?
                ;;
            -f|--flow)
                echo -e "${BLUE}🧪 Starting McallWorkflow CRD Test Suite${NC}"
                echo "=========================================="
                echo "Namespace: $NAMESPACE"
                echo "Test Cases: $WORKFLOW_TEST_CASES_FILE"
                echo "Results Directory: $RESULTS_DIR"
                echo "Timestamp: $TIMESTAMP"
                echo ""
                check_namespace
                check_crds
                check_controller
                run_all_workflow_tests
                exit $?
                ;;
            -a|--all)
                echo -e "${BLUE}🧪 Starting Complete CRD Test Suite${NC}"
                echo "=========================================="
                echo "Namespace: $NAMESPACE"
                echo "Task Test Cases: $TASK_TEST_CASES_FILE"
                echo "Workflow Test Cases: $WORKFLOW_TEST_CASES_FILE"
                echo "Results Directory: $RESULTS_DIR"
                echo "Timestamp: $TIMESTAMP"
                echo ""
                check_namespace
                check_crds
                check_controller
                
                local task_result=0
                local workflow_result=0
                
                echo -e "${BLUE}🚀 Running McallTask tests first...${NC}"
                if ! run_all_task_tests; then
                    task_result=1
                fi
                
                echo ""
                echo -e "${BLUE}🚀 Running McallWorkflow tests...${NC}"
                if ! run_all_workflow_tests; then
                    workflow_result=1
                fi
                
                echo ""
                echo -e "${BLUE}📊 Complete Test Summary${NC}"
                echo "=========================================="
                if [ $task_result -eq 0 ] && [ $workflow_result -eq 0 ]; then
                    echo -e "${GREEN}🎉 All tests passed!${NC}"
                    exit 0
                else
                    echo -e "${RED}💥 Some tests failed!${NC}"
                    exit 1
                fi
                ;;
            -c|--check)
                echo -e "${BLUE}🧪 Starting CRD Test Suite Check${NC}"
                echo "=========================================="
                echo "Namespace: $NAMESPACE"
                echo "Task Test Cases: $TASK_TEST_CASES_FILE"
                echo "Workflow Test Cases: $WORKFLOW_TEST_CASES_FILE"
                echo "Results Directory: $RESULTS_DIR"
                echo "Timestamp: $TIMESTAMP"
                echo ""
                check_namespace
                check_crds
                check_controller
                echo -e "${GREEN}✅ System is ready for testing${NC}"
                exit 0
                ;;
            *)
                # Treat as test name - auto-detect type
                echo -e "${BLUE}🧪 Starting CRD Test Suite${NC}"
                echo "=========================================="
                echo "Namespace: $NAMESPACE"
                echo "Test Name: $1"
                echo "Results Directory: $RESULTS_DIR"
                echo "Timestamp: $TIMESTAMP"
                echo ""
                check_namespace
                check_crds
                check_controller
                
                # Auto-detect test type based on name
                if [[ "$1" == *"workflow"* ]]; then
                    run_specific_workflow_test "$1"
                else
                    run_specific_task_test "$1"
                fi
                exit 0
                ;;
        esac
    done
    
    # No arguments provided
    show_help
    exit 1
}

# Run main function with all arguments
main "$@"
