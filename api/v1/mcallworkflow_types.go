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
