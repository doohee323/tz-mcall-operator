package controller

import (
	"context"
	"testing"

	mcallv1 "github.com/doohee323/tz-mcall-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestExtractJSONPath tests JSONPath extraction from JSON strings
func TestExtractJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple field extraction",
			jsonStr:  `{"status": "ok", "count": 100}`,
			path:     "$.status",
			expected: "ok",
			wantErr:  false,
		},
		{
			name:     "numeric field extraction",
			jsonStr:  `{"status": "ok", "count": 100}`,
			path:     "$.count",
			expected: "100",
			wantErr:  false,
		},
		{
			name:     "nested field extraction",
			jsonStr:  `{"data": {"status": "healthy", "uptime": 3600}}`,
			path:     "$.data.status",
			expected: "healthy",
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonStr:  `{invalid json}`,
			path:     "$.status",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "non-existent field",
			jsonStr:  `{"status": "ok"}`,
			path:     "$.nonexistent",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractJSONPath(tt.jsonStr, tt.path)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractJSONPath() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("extractJSONPath() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("extractJSONPath() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

// TestRenderTemplate tests template variable substitution
func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
	}{
		{
			name:     "single variable substitution",
			template: "Hello ${NAME}",
			data:     map[string]interface{}{"NAME": "World"},
			expected: "Hello World",
		},
		{
			name:     "multiple variable substitution",
			template: "Status: ${STATUS}, Code: ${CODE}",
			data:     map[string]interface{}{"STATUS": "OK", "CODE": "200"},
			expected: "Status: OK, Code: 200",
		},
		{
			name:     "numeric values",
			template: "Count: ${COUNT}, Price: ${PRICE}",
			data:     map[string]interface{}{"COUNT": 42, "PRICE": 99.99},
			expected: "Count: 42, Price: 99.99",
		},
		{
			name:     "no variables",
			template: "No variables here",
			data:     map[string]interface{}{},
			expected: "No variables here",
		},
		{
			name:     "unused variables",
			template: "Only ${USED}",
			data:     map[string]interface{}{"USED": "this", "UNUSED": "that"},
			expected: "Only this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderTemplate(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("renderTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestCheckTaskCondition tests task execution condition checking
func TestCheckTaskCondition(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mcallv1.AddToScheme(scheme)

	tests := []struct {
		name           string
		depTaskPhase   mcallv1.McallTaskPhase
		depErrorCode   string
		conditionWhen  string
		fieldEquals    *mcallv1.FieldCondition
		expectedResult bool
		wantErr        bool
	}{
		{
			name:           "success condition - dep succeeded",
			depTaskPhase:   mcallv1.McallTaskPhaseSucceeded,
			conditionWhen:  "success",
			expectedResult: true,
			wantErr:        false,
		},
		{
			name:           "success condition - dep failed",
			depTaskPhase:   mcallv1.McallTaskPhaseFailed,
			conditionWhen:  "success",
			expectedResult: false,
			wantErr:        false,
		},
		{
			name:           "failure condition - dep failed",
			depTaskPhase:   mcallv1.McallTaskPhaseFailed,
			conditionWhen:  "failure",
			expectedResult: true,
			wantErr:        false,
		},
		{
			name:           "failure condition - dep succeeded",
			depTaskPhase:   mcallv1.McallTaskPhaseSucceeded,
			conditionWhen:  "failure",
			expectedResult: false,
			wantErr:        false,
		},
		{
			name:           "always condition - dep succeeded",
			depTaskPhase:   mcallv1.McallTaskPhaseSucceeded,
			conditionWhen:  "always",
			expectedResult: true,
			wantErr:        false,
		},
		{
			name:           "always condition - dep failed",
			depTaskPhase:   mcallv1.McallTaskPhaseFailed,
			conditionWhen:  "always",
			expectedResult: true,
			wantErr:        false,
		},
		{
			name:          "fieldEquals - errorCode match",
			depTaskPhase:  mcallv1.McallTaskPhaseSucceeded,
			depErrorCode:  "0",
			conditionWhen: "completed",
			fieldEquals: &mcallv1.FieldCondition{
				Field: "errorCode",
				Value: "0",
			},
			expectedResult: true,
			wantErr:        false,
		},
		{
			name:          "fieldEquals - errorCode mismatch",
			depTaskPhase:  mcallv1.McallTaskPhaseFailed,
			depErrorCode:  "-1",
			conditionWhen: "completed",
			fieldEquals: &mcallv1.FieldCondition{
				Field: "errorCode",
				Value: "0",
			},
			expectedResult: false,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dependent task  
			depTask := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dep-task",
					Namespace: "test-ns",
				},
				Status: mcallv1.McallTaskStatus{
					Phase: tt.depTaskPhase,
					Result: &mcallv1.McallTaskResult{
						ErrorCode: tt.depErrorCode,
					},
				},
			}

			// Create current task
			currentTask := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "current-task",
					Namespace: "test-ns",
				},
			}

			// Create fake client with dependent task
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(depTask, currentTask).
				Build()

			reconciler := &McallTaskReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			condition := &mcallv1.TaskCondition{
				DependentTask: "dep-task",
				When:          tt.conditionWhen,
				FieldEquals:   tt.fieldEquals,
			}

			ctx := context.Background()
			result, err := reconciler.checkTaskCondition(ctx, currentTask, condition)

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkTaskCondition() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("checkTaskCondition() unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("checkTaskCondition() = %v, want %v", result, tt.expectedResult)
				}
			}
		})
	}
}

// TestProcessInputSources tests input sources processing
func TestProcessInputSources(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mcallv1.AddToScheme(scheme)

	tests := []struct {
		name          string
		inputSources  []mcallv1.TaskInputSource
		inputTemplate string
		refTaskOutput string
		refTaskPhase  mcallv1.McallTaskPhase
		refErrorCode  string
		expectedInput string
		expectEnvVars map[string]string
		wantErr       bool
	}{
		{
			name: "extract phase field",
			inputSources: []mcallv1.TaskInputSource{
				{
					Name:    "PHASE",
					TaskRef: "ref-task",
					Field:   "phase",
				},
			},
			refTaskPhase:  mcallv1.McallTaskPhaseSucceeded,
			expectedInput: `{"PHASE":"Succeeded"}`,
			expectEnvVars: map[string]string{"PHASE": "Succeeded"},
			wantErr:       false,
		},
		{
			name: "extract errorCode field",
			inputSources: []mcallv1.TaskInputSource{
				{
					Name:    "ERROR_CODE",
					TaskRef: "ref-task",
					Field:   "errorCode",
				},
			},
			refTaskPhase:  mcallv1.McallTaskPhaseSucceeded,
			refErrorCode:  "0",
			expectedInput: `{"ERROR_CODE":"0"}`,
			expectEnvVars: map[string]string{"ERROR_CODE": "0"},
			wantErr:       false,
		},
		{
			name: "template with multiple sources",
			inputSources: []mcallv1.TaskInputSource{
				{
					Name:    "PHASE",
					TaskRef: "ref-task",
					Field:   "phase",
				},
				{
					Name:    "CODE",
					TaskRef: "ref-task",
					Field:   "errorCode",
				},
			},
			inputTemplate: "Status: ${PHASE}, Code: ${CODE}",
			refTaskPhase:  mcallv1.McallTaskPhaseSucceeded,
			refErrorCode:  "0",
			expectedInput: "Status: Succeeded, Code: 0",
			expectEnvVars: map[string]string{"PHASE": "Succeeded", "CODE": "0"},
			wantErr:       false,
		},
		{
			name: "default value when task not found",
			inputSources: []mcallv1.TaskInputSource{
				{
					Name:    "STATUS",
					TaskRef: "nonexistent-task",
					Field:   "phase",
					Default: "unknown",
				},
			},
			expectedInput: `{"STATUS":"unknown"}`,
			expectEnvVars: map[string]string{"STATUS": "unknown"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create reference task if needed
			objects := []client.Object{}
			if tt.refTaskPhase != "" {
				refTask := &mcallv1.McallTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ref-task",
						Namespace: "test-ns",
					},
					Status: mcallv1.McallTaskStatus{
						Phase: tt.refTaskPhase,
						Result: &mcallv1.McallTaskResult{
							Output:    tt.refTaskOutput,
							ErrorCode: tt.refErrorCode,
						},
					},
				}
				objects = append(objects, refTask)
			}

			// Create current task
			currentTask := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "current-task",
					Namespace: "test-ns",
				},
				Spec: mcallv1.McallTaskSpec{
					InputSources:  tt.inputSources,
					InputTemplate: tt.inputTemplate,
				},
			}
			objects = append(objects, currentTask)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			reconciler := &McallTaskReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			ctx := context.Background()
			inputStr, envVars, err := reconciler.processInputSources(ctx, currentTask)

			if tt.wantErr {
				if err == nil {
					t.Errorf("processInputSources() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("processInputSources() unexpected error: %v", err)
				return
			}

			if inputStr != tt.expectedInput {
				t.Errorf("processInputSources() input = %q, want %q", inputStr, tt.expectedInput)
			}

			for k, expectedVal := range tt.expectEnvVars {
				if envVars[k] != expectedVal {
					t.Errorf("processInputSources() envVars[%s] = %q, want %q", k, envVars[k], expectedVal)
				}
			}
		})
	}
}

// TestTruncateString tests string truncation utility
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world this is a test",
			maxLen:   10,
			expected: "hello worl...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

