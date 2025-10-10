package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// McallWorkflowPhase represents the current phase of workflow execution
type McallWorkflowPhase string

const (
	McallWorkflowPhasePending   McallWorkflowPhase = "Pending"
	McallWorkflowPhaseRunning   McallWorkflowPhase = "Running"
	McallWorkflowPhaseSucceeded McallWorkflowPhase = "Succeeded"
	McallWorkflowPhaseFailed    McallWorkflowPhase = "Failed"
)

// McallWorkflowSpec defines the desired state of McallWorkflow
type McallWorkflowSpec struct {
	// Tasks is the list of McallTask references in this workflow
	Tasks []WorkflowTaskRef `json:"tasks"`

	// Schedule is the cron schedule for workflow execution (optional)
	// Format: "minute hour day month weekday"
	// Example: "0 2 * * *" (every day at 2 AM)
	Schedule string `json:"schedule,omitempty"`

	// Concurrency is the maximum number of concurrent task executions
	Concurrency int32 `json:"concurrency,omitempty"`

	// Timeout is the overall workflow timeout in seconds
	Timeout int32 `json:"timeout,omitempty"`

	// RetryPolicy defines the retry policy for the workflow
	RetryPolicy *WorkflowRetryPolicy `json:"retryPolicy,omitempty"`

	// Environment variables for all tasks in the workflow
	Environment map[string]string `json:"environment,omitempty"`

	// Resources defines resource requirements for all tasks
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// WorkflowTaskRef represents a reference to a McallTask in a workflow
type WorkflowTaskRef struct {
	// Name is the name of the task in the workflow
	Name string `json:"name"`

	// TaskRef is the reference to the McallTask
	TaskRef TaskRef `json:"taskRef"`

	// Dependencies is the list of task names this task depends on
	Dependencies []string `json:"dependencies,omitempty"`

	// Condition defines when this task should run
	Condition *TaskCondition `json:"condition,omitempty"`

	// InputSources defines data to pass from other tasks
	InputSources []TaskInputSource `json:"inputSources,omitempty"`

	// InputTemplate for variable substitution
	InputTemplate string `json:"inputTemplate,omitempty"`
}

// TaskCondition defines execution conditions for a task
type TaskCondition struct {
	// DependentTask: name of the task whose result to check
	DependentTask string `json:"dependentTask"`

	// When: execution condition
	// - "success": run only if dependent task succeeded
	// - "failure": run only if dependent task failed
	// - "always": run always after dependent task completes
	// - "completed": run when dependent task completes (success or failure)
	When string `json:"when"`

	// FieldEquals: run if specific field equals specific value
	FieldEquals *FieldCondition `json:"fieldEquals,omitempty"`

	// OutputContains: run if output contains specific string
	OutputContains string `json:"outputContains,omitempty"`
}

// FieldCondition defines a field-based condition
type FieldCondition struct {
	// Field name to check (e.g., "errorCode", "phase")
	Field string `json:"field"`

	// Expected value
	Value string `json:"value"`
}

// TaskRef represents a reference to a McallTask
type TaskRef struct {
	// Name is the name of the McallTask
	Name string `json:"name"`

	// Namespace is the namespace of the McallTask
	Namespace string `json:"namespace,omitempty"`
}

// WorkflowRetryPolicy defines the retry policy for a workflow
type WorkflowRetryPolicy struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// RetryDelay is the delay between retries in seconds
	RetryDelay int32 `json:"retryDelay,omitempty"`

	// BackoffPolicy is the backoff policy for retries
	BackoffPolicy string `json:"backoffPolicy,omitempty"`
}

// McallWorkflowStatus defines the observed state of McallWorkflow
type McallWorkflowStatus struct {
	// Phase represents the current phase of workflow execution
	Phase McallWorkflowPhase `json:"phase,omitempty"`

	// StartTime is the time when the workflow started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time when the workflow completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// TaskStatuses is the status of individual tasks
	TaskStatuses []TaskStatus `json:"taskStatuses,omitempty"`

	// Message is a human-readable message about the workflow status
	Message string `json:"message,omitempty"`

	// Reason is a brief reason for the current status
	Reason string `json:"reason,omitempty"`

	// RetryCount is the number of times the workflow has been retried
	RetryCount int32 `json:"retryCount,omitempty"`

	// LastRetryTime is the time of the last retry
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`

	// LastRunTime is the time when the workflow was last executed
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`

	// DAG representation for UI visualization
	DAG *WorkflowDAG `json:"dag,omitempty"`
}

// TaskStatus represents the status of a single task in the workflow
type TaskStatus struct {
	// Name is the name of the task
	Name string `json:"name"`

	// Phase is the current phase of the task
	Phase McallTaskPhase `json:"phase,omitempty"`

	// StartTime is the time when the task started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time when the task completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Message is a human-readable message about the task status
	Message string `json:"message,omitempty"`

	// Reason is a brief reason for the current status
	Reason string `json:"reason,omitempty"`
}

// McallWorkflow is the Schema for the mcallworkflows API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Start Time",type="date",JSONPath=".status.startTime"
// +kubebuilder:printcolumn:name="Completion Time",type="date",JSONPath=".status.completionTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type McallWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   McallWorkflowSpec   `json:"spec,omitempty"`
	Status McallWorkflowStatus `json:"status,omitempty"`
}

// McallWorkflowList contains a list of McallWorkflow
// +kubebuilder:object:root=true
type McallWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []McallWorkflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&McallWorkflow{}, &McallWorkflowList{})
}

// WorkflowDAG represents the DAG structure of a workflow for UI visualization
type WorkflowDAG struct {
	// Nodes is the list of task nodes in the DAG
	Nodes []DAGNode `json:"nodes"`

	// Edges is the list of edges connecting nodes
	Edges []DAGEdge `json:"edges"`

	// Layout algorithm used for positioning (dagre, elk, auto)
	Layout string `json:"layout,omitempty"`

	// Metadata contains summary information about the DAG
	Metadata DAGMetadata `json:"metadata,omitempty"`
}

// DAGNode represents a task node in the DAG
type DAGNode struct {
	// ID is the unique identifier for the node (task name)
	ID string `json:"id"`

	// Name is the display name of the task
	Name string `json:"name"`

	// Type is the task type (cmd, get, post)
	Type string `json:"type"`

	// Phase is the current execution phase
	Phase McallTaskPhase `json:"phase"`

	// StartTime is when the task started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// EndTime is when the task completed
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration is the execution duration in human-readable format
	Duration string `json:"duration,omitempty"`

	// Input is the task input command or URL
	Input string `json:"input,omitempty"`

	// Output is the task execution output (truncated for UI)
	Output string `json:"output,omitempty"`

	// ErrorCode is the execution result code
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorMessage is the error message if failed
	ErrorMessage string `json:"errorMessage,omitempty"`

	// Position for UI layout
	Position *NodePosition `json:"position,omitempty"`

	// Retries is the number of retry attempts
	Retries int32 `json:"retries,omitempty"`

	// TaskRef is the original template task reference
	TaskRef string `json:"taskRef,omitempty"`
}

// DAGEdge represents a dependency edge between tasks
type DAGEdge struct {
	// ID is the unique identifier for the edge
	ID string `json:"id"`

	// Source is the source node ID
	Source string `json:"source"`

	// Target is the target node ID
	Target string `json:"target"`

	// Type is the edge type (dependency, success, failure, always)
	Type string `json:"type,omitempty"`

	// Condition is the execution condition
	Condition string `json:"condition,omitempty"`

	// Label for display
	Label string `json:"label,omitempty"`
}

// NodePosition represents the x,y coordinates of a node
type NodePosition struct {
	// X coordinate (in pixels)
	X int `json:"x"`

	// Y coordinate (in pixels)
	Y int `json:"y"`
}

// DAGMetadata contains summary statistics about the DAG
type DAGMetadata struct {
	// TotalNodes is the total number of nodes
	TotalNodes int `json:"totalNodes"`

	// TotalEdges is the total number of edges
	TotalEdges int `json:"totalEdges"`

	// SuccessCount is the number of succeeded tasks
	SuccessCount int `json:"successCount"`

	// FailureCount is the number of failed tasks
	FailureCount int `json:"failureCount"`

	// RunningCount is the number of running tasks
	RunningCount int `json:"runningCount"`

	// PendingCount is the number of pending tasks
	PendingCount int `json:"pendingCount"`

	// SkippedCount is the number of skipped tasks
	SkippedCount int `json:"skippedCount"`
}
