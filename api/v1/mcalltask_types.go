package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// McallTaskSpec defines the desired state of McallTask
type McallTaskSpec struct {
	// Type of request (command, HTTP GET, HTTP POST)
	Type string `json:"type"`

	// Input command or URL to execute
	Input string `json:"input"`

	// Name identifier for this task
	Name string `json:"name,omitempty"`

	// Timeout in seconds
	Timeout int32 `json:"timeout,omitempty"`

	// Number of retries on failure
	RetryCount int32 `json:"retryCount,omitempty"`

	// Cron schedule for recurring tasks (optional)
	Schedule string `json:"schedule,omitempty"`

	// List of task names this task depends on
	Dependencies []string `json:"dependencies,omitempty"`

	// Environment variables for task execution
	Environment map[string]string `json:"environment,omitempty"`

	// Resource requirements for task execution
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// HTTP response validation for GET/POST requests
	HttpValidation *HttpValidation `json:"httpValidation,omitempty"`

	// Command output validation for CMD requests
	OutputValidation *OutputValidation `json:"outputValidation,omitempty"`

	// Execution mode for multiple inputs (sequential/parallel)
	ExecutionMode string `json:"executionMode,omitempty"`

	// Fail fast on error - stop execution on first error (default: false)
	FailFast bool `json:"failFast,omitempty"`

	// InputSources: reference results from previous tasks
	InputSources []TaskInputSource `json:"inputSources,omitempty"`

	// InputTemplate: template string with variable substitution
	InputTemplate string `json:"inputTemplate,omitempty"`
}

// TaskInputSource represents a reference to another task's result
type TaskInputSource struct {
	// Name: variable name for template substitution or environment variable
	Name string `json:"name"`

	// TaskRef: name of the task to reference
	TaskRef string `json:"taskRef"`

	// Field: which field to extract from task result
	// - "output": task execution output
	// - "errorCode": execution result code ("0" or "-1")
	// - "phase": task status (Succeeded, Failed, etc)
	// - "errorMessage": error message if failed
	// - "all": all information as JSON
	Field string `json:"field"`

	// JSONPath: extract specific field from JSON output (optional)
	// Example: "$.data.status", "$.items[0].name"
	JSONPath string `json:"jsonPath,omitempty"`

	// Default: default value if field not found or task failed
	Default string `json:"default,omitempty"`
}

// McallTaskStatus defines the observed state of McallTask
type McallTaskStatus struct {
	// Current phase of the task
	Phase McallTaskPhase `json:"phase"`

	// When the task started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// When the task completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Task execution result
	Result *McallTaskResult `json:"result,omitempty"`

	// Current retry count
	RetryCount int32 `json:"retryCount,omitempty"`

	// Last retry attempt time
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`
}

// McallTaskResult represents the result of task execution
type McallTaskResult struct {
	// Task output
	Output string `json:"output,omitempty"`

	// Error code (0 for success, -1 for failure)
	ErrorCode string `json:"errorCode,omitempty"`

	// Error message if failed
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// McallTaskPhase represents the phase of a task
type McallTaskPhase string

const (
	McallTaskPhasePending   McallTaskPhase = "Pending"
	McallTaskPhaseRunning   McallTaskPhase = "Running"
	McallTaskPhaseSucceeded McallTaskPhase = "Succeeded"
	McallTaskPhaseFailed    McallTaskPhase = "Failed"
	McallTaskPhaseSkipped   McallTaskPhase = "Skipped"
)

// Execution mode constants
const (
	ExecutionModeSequential = "sequential"
	ExecutionModeParallel   = "parallel"
)

const McallTaskFinalizer = "mcall.tz.io/finalizer"

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
//+kubebuilder:printcolumn:name="Input",type="string",JSONPath=".spec.input"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// McallTask is the Schema for the mcalltasks API
type McallTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   McallTaskSpec   `json:"spec,omitempty"`
	Status McallTaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// McallTaskList contains a list of McallTask
type McallTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []McallTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&McallTask{}, &McallTaskList{})
}

// HttpValidation defines HTTP response validation rules
type HttpValidation struct {
	// Expected HTTP status codes
	ExpectedStatusCodes []int `json:"expectedStatusCodes,omitempty"`

	// Expected response body content
	ExpectedResponseBody string `json:"expectedResponseBody,omitempty"`

	// How to match response body
	ResponseBodyMatch string `json:"responseBodyMatch,omitempty"`

	// Regex pattern for response body matching
	ResponseBodyPattern string `json:"responseBodyPattern,omitempty"`

	// Expected response headers
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`

	// HTTP response timeout in seconds
	ResponseTimeout int32 `json:"responseTimeout,omitempty"`

	// Whether to follow redirects
	FollowRedirects bool `json:"followRedirects,omitempty"`

	// Maximum number of redirects to follow
	MaxRedirects int32 `json:"maxRedirects,omitempty"`
}

// OutputValidation defines command output validation rules
type OutputValidation struct {
	// Expected output content
	ExpectedOutput string `json:"expectedOutput,omitempty"`

	// How to match output content
	OutputMatch string `json:"outputMatch,omitempty"`

	// Regex pattern for output matching
	OutputPattern string `json:"outputPattern,omitempty"`

	// Success criteria for output validation
	SuccessCriteria string `json:"successCriteria,omitempty"`

	// Failure criteria for output validation
	FailureCriteria string `json:"failureCriteria,omitempty"`

	// Whether output matching is case sensitive
	CaseSensitive bool `json:"caseSensitive,omitempty"`

	// Whether to support multiline output
	Multiline bool `json:"multiline,omitempty"`

	// Expected number of output lines
	ExpectedLines int32 `json:"expectedLines,omitempty"`

	// Output timeout in seconds
	OutputTimeout int32 `json:"outputTimeout,omitempty"`

	// JSONPath expression for JSON validation
	JsonPath string `json:"jsonPath,omitempty"`

	// Expected JSON value at specified path
	ExpectedJsonValue string `json:"expectedJsonValue,omitempty"`

	// Expected output content that indicates failure
	ExpectedFailureOutput string `json:"expectedFailureOutput,omitempty"`
}
