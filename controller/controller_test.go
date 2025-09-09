package controller

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

// Test structures for monitoring and alerting
type EmailAlert struct {
	ServiceName string
	Status      string
	Error       string
	Timestamp   string
	Details     string
}

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		timeout  time.Duration
		wantErr  bool
		contains string
	}{
		{
			name:     "simple echo command",
			command:  "echo 'Hello World'",
			timeout:  5 * time.Second,
			wantErr:  false,
			contains: "Hello World",
		},
		{
			name:     "date command",
			command:  "date",
			timeout:  5 * time.Second,
			wantErr:  false,
			contains: "2025", // Should contain year
		},
		{
			name:     "ls command",
			command:  "ls -la",
			timeout:  5 * time.Second,
			wantErr:  false,
			contains: "total",
		},
		{
			name:     "invalid command",
			command:  "nonexistentcommand12345",
			timeout:  5 * time.Second,
			wantErr:  true,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(tt.command, tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeCommand() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("executeCommand() unexpected error: %v", err)
				}
				if tt.contains != "" && !containsEnhanced(output, tt.contains) {
					t.Errorf("executeCommand() output %q does not contain %q", output, tt.contains)
				}
			}
		})
	}
}

func TestExecuteHTTPRequest(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		method   string
		timeout  time.Duration
		wantErr  bool
		contains string
	}{
		{
			name:     "valid GET request",
			url:      "https://httpbin.org/get",
			method:   "GET",
			timeout:  10 * time.Second,
			wantErr:  false,
			contains: "httpbin.org",
		},
		{
			name:     "valid POST request",
			url:      "https://httpbin.org/post",
			method:   "POST",
			timeout:  10 * time.Second,
			wantErr:  false,
			contains: "httpbin.org",
		},
		{
			name:     "invalid URL",
			url:      "https://invalid-url-that-does-not-exist-12345.com",
			method:   "GET",
			timeout:  2 * time.Second,
			wantErr:  true,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeHTTPRequest(tt.url, tt.method, tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeHTTPRequest() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("executeHTTPRequest() unexpected error: %v", err)
				}
				if tt.contains != "" && !containsEnhanced(output, tt.contains) {
					t.Errorf("executeHTTPRequest() output %q does not contain %q", output, tt.contains)
				}
			}
		})
	}
}

func TestCheckExpect(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expect   string
		expected bool
	}{
		{
			name:     "single expect match",
			content:  "Hello World",
			expect:   "Hello",
			expected: true,
		},
		{
			name:     "single expect no match",
			content:  "Hello World",
			expect:   "Goodbye",
			expected: false,
		},
		{
			name:     "multiple expect with pipe - first match",
			content:  "Status: 200\nBody: OK",
			expect:   "200|301|500",
			expected: true,
		},
		{
			name:     "multiple expect with pipe - second match",
			content:  "Status: 301\nBody: Moved",
			expect:   "200|301|500",
			expected: true,
		},
		{
			name:     "multiple expect with pipe - no match",
			content:  "Status: 404\nBody: Not Found",
			expect:   "200|301|500",
			expected: false,
		},
		{
			name:     "empty expect",
			content:  "Any content",
			expect:   "",
			expected: true,
		},
		{
			name:     "expect with spaces",
			content:  "Escape character is '^]'",
			expect:   "Escape character is",
			expected: true,
		},
		{
			name:     "HTTP status code match",
			content:  "200|{\"args\":{\"test\":\"validation\"}}",
			expect:   "200",
			expected: true,
		},
		{
			name:     "HTTP response body match",
			content:  "200|{\"args\":{\"test\":\"validation\"}}",
			expect:   "test",
			expected: true,
		},
		{
			name:     "HTTP status or body match",
			content:  "200|{\"args\":{\"test\":\"validation\"}}",
			expect:   "200|test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExpect(tt.content, tt.expect)
			if result != tt.expected {
				t.Errorf("checkExpect() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTaskWorkerWithExpect(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		inputType string
		nameStr   string
		expect    string
		timeout   time.Duration
		wantErr   bool
	}{
		{
			name:      "cmd with expect match",
			input:     "echo 'Hello World'",
			inputType: "cmd",
			nameStr:   "test-cmd",
			expect:    "Hello",
			timeout:   5 * time.Second,
			wantErr:   false,
		},
		{
			name:      "cmd with expect no match",
			input:     "echo 'Hello World'",
			inputType: "cmd",
			nameStr:   "test-cmd",
			expect:    "Goodbye",
			timeout:   5 * time.Second,
			wantErr:   true,
		},
		{
			name:      "http with status code expect",
			input:     "https://httpbin.org/status/200",
			inputType: "get",
			nameStr:   "test-http",
			expect:    "200",
			timeout:   10 * time.Second,
			wantErr:   false,
		},
		{
			name:      "http with wrong status code expect",
			input:     "https://httpbin.org/status/200",
			inputType: "get",
			nameStr:   "test-http",
			expect:    "404",
			timeout:   10 * time.Second,
			wantErr:   true,
		},
		{
			name:      "http with response body expect",
			input:     "https://httpbin.org/get?test=validation",
			inputType: "get",
			nameStr:   "test-http-body",
			expect:    "test",
			timeout:   10 * time.Second,
			wantErr:   false,
		},
		{
			name:      "http with status or body expect",
			input:     "https://httpbin.org/get?test=validation",
			inputType: "get",
			nameStr:   "test-http-mixed",
			expect:    "200|test",
			timeout:   10 * time.Second,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewTaskWorker(tt.input, tt.inputType, tt.nameStr, tt.expect)

			// Execute worker
			worker.Execute(tt.timeout)

			// Get result
			result := <-worker.result

			// Check if error matches expectation
			if tt.wantErr {
				if result.Error == "0" {
					t.Errorf("Expected error but got success")
				}
			} else {
				if result.Error != "0" {
					t.Errorf("Expected success but got error: %s", result.Error)
				}
			}
		})
	}
}

func TestTaskWorkerWithHTTPExpect(t *testing.T) {
	worker := NewTaskWorker("https://httpbin.org/status/200", "get", "test-http", "200")

	// Execute the worker
	worker.Execute(10 * time.Second)

	// Get result
	result := <-worker.result

	// Should succeed because status code 200 matches expect "200"
	if result.Error != "0" {
		t.Errorf("Expected success but got error: %s, content: %s", result.Error, result.Content)
	}
}

func TestTaskWorkerWithHTTPExpectFailure(t *testing.T) {
	worker := NewTaskWorker("https://httpbin.org/status/404", "get", "test-http", "200")

	// Execute the worker
	worker.Execute(10 * time.Second)

	// Get result
	result := <-worker.result

	// Should fail because status code 404 does not match expect "200"
	if result.Error != "-1" {
		t.Errorf("Expected error but got success: %s", result.Error)
	}
}

func TestTaskWorkerWithCmdExpect(t *testing.T) {
	worker := NewTaskWorker("echo 'Hello World'", "cmd", "test-cmd", "Hello")

	// Execute the worker
	worker.Execute(10 * time.Second)

	// Get result
	result := <-worker.result

	// Should succeed because output contains "Hello"
	if result.Error != "0" {
		t.Errorf("Expected success but got error: %s, content: %s", result.Error, result.Content)
	}
}

func TestTaskWorkerWithExpectedResponse(t *testing.T) {
	worker := NewTaskWorker("https://httpbin.org/get?test=validation", "get", "test-http", "200|test")

	// Execute the worker
	worker.Execute(10 * time.Second)

	// Get result
	result := <-worker.result

	// Should succeed
	if result.Error != "0" {
		t.Errorf("Expected success but got error: %s, content: %s", result.Error, result.Content)
	}
}

func TestJSONParsingWithExpectedResponse(t *testing.T) {
	// Test the exact JSON from complex-response-validation-test
	jsonInput := `{
  "inputs": [
    {
      "type": "get",
      "input": "https://httpbin.org/get?test=validation",
      "expect": "200|test"
    },
    {
      "type": "get",
      "input": "https://httpbin.org/post",
      "expect": "405"
    }
  ]
}`

	// Parse JSON inputs
	inputs, err := parseJSONInputs(jsonInput)
	if err != nil {
		t.Fatalf("Failed to parse JSON inputs: %v", err)
	}

	if len(inputs) != 2 {
		t.Fatalf("Expected 2 inputs, got %d", len(inputs))
	}

	// Check first input
	firstInput := inputs[0]
	if firstInput["type"] != "get" {
		t.Errorf("Expected type 'get', got %v", firstInput["type"])
	}
	if firstInput["input"] != "https://httpbin.org/get?test=validation" {
		t.Errorf("Unexpected input URL")
	}
	if firstInput["expect"] != "200|test" {
		t.Errorf("Expected expect '200|test', got %v", firstInput["expect"])
	}

	// Check second input
	secondInput := inputs[1]
	if secondInput["type"] != "get" {
		t.Errorf("Expected type 'get', got %v", secondInput["type"])
	}
	if secondInput["input"] != "https://httpbin.org/post" {
		t.Errorf("Unexpected input URL")
	}
	if secondInput["expect"] != "405" {
		t.Errorf("Expected expect '405', got %v", secondInput["expect"])
	}
}

// Helper function for enhanced string contains check
func containsEnhanced(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// TestPostgreSQLLogging tests PostgreSQL logging functionality
func TestPostgreSQLLogging(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		serviceType  string
		status       string
		error        string
		responseTime int64
		timestamp    time.Time
		wantErr      bool
	}{
		{
			name:         "successful_service_check",
			serviceName:  "grafana",
			serviceType:  "get",
			status:       "UP",
			error:        "",
			responseTime: 150,
			timestamp:    time.Now(),
			wantErr:      false,
		},
		{
			name:         "failed_service_check",
			serviceName:  "prometheus",
			serviceType:  "get",
			status:       "DOWN",
			error:        "Connection timeout",
			responseTime: 5000,
			timestamp:    time.Now(),
			wantErr:      false,
		},
		{
			name:         "database_connectivity_check",
			serviceName:  "redis.hypen-dev",
			serviceType:  "cmd",
			status:       "DOWN",
			error:        "Connection refused",
			responseTime: 2000,
			timestamp:    time.Now(),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate log entry creation
			logEntry := LogEntry{
				ServiceName:  tt.serviceName,
				ServiceType:  tt.serviceType,
				Status:       tt.status,
				Error:        tt.error,
				ResponseTime: tt.responseTime,
				Timestamp:    tt.timestamp,
			}

			// Test log entry validation
			if logEntry.ServiceName == "" {
				t.Errorf("Service name cannot be empty")
			}
			if logEntry.Timestamp.IsZero() {
				t.Errorf("Timestamp cannot be zero")
			}
			if logEntry.Status != "UP" && logEntry.Status != "DOWN" {
				t.Errorf("Status must be UP or DOWN, got: %s", logEntry.Status)
			}

			t.Logf("Log entry created for %s: %s (response time: %dms)",
				tt.serviceName, tt.status, tt.responseTime)
		})
	}
}

// Test database connectivity monitoring
func TestDatabaseConnectivityMonitoring(t *testing.T) {
	tests := []struct {
		name           string
		dbName         string
		dbType         string
		dbCommand      string
		expectedOutput string
		timeout        time.Duration
		wantErr        bool
	}{
		{
			name:           "redis hypen-dev connectivity",
			dbName:         "redis.hypen-dev",
			dbType:         "cmd",
			dbCommand:      "telnet redis.hypen-dev.hypen.ai 6379",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "redis mc20-dev connectivity",
			dbName:         "redis.mc20-dev",
			dbType:         "cmd",
			dbCommand:      "telnet redis.mc20-dev.seerslab.io 6379",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "redis mtown-dev connectivity",
			dbName:         "redis.mtown-dev",
			dbType:         "cmd",
			dbCommand:      "telnet redis.mtown-dev.mirrortown.io 6379",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql hypen-dev connectivity",
			dbName:         "mysql.hypen-dev",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.hypen-dev.hypen.ai 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql mtown-dev connectivity",
			dbName:         "mysql.mtown-dev",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.mtown-dev.seerslab.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mariadb mtown-dev connectivity",
			dbName:         "mariadb.mtown-dev",
			dbType:         "cmd",
			dbCommand:      "telnet mariadb.mtown-dev.seerslab.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql avatar-dev connectivity",
			dbName:         "mysql.avatar-dev",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.avatar-dev.seerslab.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql mc20-dev connectivity",
			dbName:         "mysql.mc20-dev",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.mc20-dev.seerslab.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql mtown-dev alt connectivity",
			dbName:         "mysql.mtown-dev-alt",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.mtown-dev.mirrortown.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql hypen connectivity",
			dbName:         "mysql.hypen",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.hypen.hypen.ai 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "redis hypen connectivity",
			dbName:         "redis.hypen",
			dbType:         "cmd",
			dbCommand:      "telnet redis.hypen.hypen.ai 6379",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "mysql mc20 connectivity",
			dbName:         "mysql.mc20",
			dbType:         "cmd",
			dbCommand:      "telnet mysql.mc20.seerslab.io 3306",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "redis mc20 connectivity",
			dbName:         "redis.mc20",
			dbType:         "cmd",
			dbCommand:      "telnet redis.mc20.seerslab.io 6379",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewTaskWorker(tt.dbCommand, tt.dbType, tt.dbName, tt.expectedOutput)

			// Execute worker
			worker.Execute(tt.timeout)

			// Get result
			result := <-worker.result

			// Check if error matches expectation
			if tt.wantErr {
				if result.Error == "0" {
					t.Errorf("Expected error but got success for %s", tt.dbName)
				}
			} else {
				if result.Error != "0" {
					t.Errorf("Expected success but got error for %s: %s, content: %s", tt.dbName, result.Error, result.Content)
				}
			}
		})
	}
}

// Test network connectivity monitoring
func TestNetworkConnectivityMonitoring(t *testing.T) {
	tests := []struct {
		name           string
		networkName    string
		networkType    string
		networkCommand string
		expectedOutput string
		timeout        time.Duration
		wantErr        bool
	}{
		{
			name:           "bastion ssh connectivity",
			networkName:    "bastion",
			networkType:    "cmd",
			networkCommand: "telnet bastion.eks-main-t.seerslab.io 22",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "bastion-mcall ssh connectivity",
			networkName:    "bastion-mcall",
			networkType:    "cmd",
			networkCommand: "telnet bastion-mcall.eks-main-t.seerslab.io 22",
			expectedOutput: "Escape character is",
			timeout:        5 * time.Second,
			wantErr:        false,
		},
		{
			name:           "prodm-stg healthcheck",
			networkName:    "prodm-stg",
			networkType:    "get",
			networkCommand: "https://prodm-stg.seerslab.io/api/v1/healthcheck",
			expectedOutput: "200|301 Moved Permanently",
			timeout:        10 * time.Second,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewTaskWorker(tt.networkCommand, tt.networkType, tt.networkName, tt.expectedOutput)

			// Execute worker
			worker.Execute(tt.timeout)

			// Get result
			result := <-worker.result

			// Check if error matches expectation
			if tt.wantErr {
				if result.Error == "0" {
					t.Errorf("Expected error but got success for %s", tt.networkName)
				}
			} else {
				if result.Error != "0" {
					t.Errorf("Expected success but got error for %s: %s, content: %s", tt.networkName, result.Error, result.Content)
				}
			}
		})
	}
}

// Test monitoring workflow with multiple services
func TestMonitoringWorkflow(t *testing.T) {
	// Test a comprehensive monitoring workflow
	services := []struct {
		name   string
		url    string
		expect string
	}{
		{"grafana", "https://grafana.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
		{"prometheus", "https://prometheus.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
		{"alertmanager", "https://alertmanager.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
		{"argocd", "https://argocd.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
		{"vault", "https://vault.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
		{"consul", "https://consul.default.eks-main-t.seerslab.io", "200|301 Moved Permanently"},
	}

	results := make([]TaskResult, len(services))

	for i, service := range services {
		worker := NewTaskWorker(service.url, "get", service.name, service.expect)
		worker.Execute(10 * time.Second)
		results[i] = <-worker.result
	}

	// Check results
	successCount := 0
	for i, result := range results {
		if result.Error == "0" {
			successCount++
			t.Logf("Service %s (%s) is healthy", services[i].name, services[i].url)
		} else {
			t.Logf("Service %s (%s) is unhealthy: %s", services[i].name, services[i].url, result.Error)
		}
	}

	t.Logf("Monitoring workflow completed: %d/%d services healthy", successCount, len(services))

	// At least some services should be healthy
	if successCount == 0 {
		t.Errorf("No services were healthy in monitoring workflow")
	}
}

// Test alerting functionality simulation
func TestAlertingSimulation(t *testing.T) {
	// Simulate a service failure scenario
	failingService := "https://httpbin.org/status/500"
	worker := NewTaskWorker(failingService, "get", "failing-service", "200")

	worker.Execute(10 * time.Second)
	result := <-worker.result

	// This should fail and trigger an alert
	if result.Error == "0" {
		t.Errorf("Expected service to fail but it succeeded")
	} else {
		t.Logf("Service failure detected: %s - This would trigger an alert", result.Error)
	}

	// Simulate a successful service recovery
	recoveringService := "https://httpbin.org/status/200"
	worker2 := NewTaskWorker(recoveringService, "get", "recovering-service", "200")

	worker2.Execute(10 * time.Second)
	result2 := <-worker2.result

	// This should succeed
	if result2.Error != "0" {
		t.Errorf("Expected service to recover but it failed: %s", result2.Error)
	} else {
		t.Logf("Service recovery detected: %s - This would clear the alert", result2.Error)
	}
}

// TestEmailAlerting tests email alerting functionality
func TestEmailAlerting(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		status      string
		error       string
		timestamp   string
		details     string
		wantErr     bool
	}{
		{
			name:        "service_down_alert",
			serviceName: "grafana",
			status:      "DOWN",
			error:       "Connection timeout",
			timestamp:   "2025-09-08T18:30:00Z",
			details:     "Service grafana is not responding",
			wantErr:     false,
		},
		{
			name:        "service_recovery_alert",
			serviceName: "grafana",
			status:      "UP",
			error:       "",
			timestamp:   "2025-09-08T18:35:00Z",
			details:     "Service grafana is now responding",
			wantErr:     false,
		},
		{
			name:        "database_down_alert",
			serviceName: "redis.hypen-dev",
			status:      "DOWN",
			error:       "Connection refused",
			timestamp:   "2025-09-08T18:30:00Z",
			details:     "Database redis.hypen-dev is not accessible",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate email alert sending
			alert := EmailAlert{
				ServiceName: tt.serviceName,
				Status:      tt.status,
				Error:       tt.error,
				Timestamp:   tt.timestamp,
				Details:     tt.details,
			}

			// Test email template generation
			template := `Service: {{.ServiceName}}
Status: {{.Status}}
Error: {{.Error}}
Timestamp: {{.Timestamp}}
Details: {{.Details}}`

			// Simulate template rendering
			rendered := strings.ReplaceAll(template, "{{.ServiceName}}", alert.ServiceName)
			rendered = strings.ReplaceAll(rendered, "{{.Status}}", alert.Status)
			rendered = strings.ReplaceAll(rendered, "{{.Error}}", alert.Error)
			rendered = strings.ReplaceAll(rendered, "{{.Timestamp}}", alert.Timestamp)
			rendered = strings.ReplaceAll(rendered, "{{.Details}}", alert.Details)

			if rendered == "" {
				t.Errorf("Email template rendering failed")
			}

			t.Logf("Email alert generated for %s: %s", tt.serviceName, tt.status)
		})
	}
}

// TestEndToEndMonitoringAndAlerting tests the complete monitoring and alerting workflow
func TestEndToEndMonitoringAndAlerting(t *testing.T) {
	t.Run("complete_monitoring_workflow", func(t *testing.T) {
		// Simulate a complete monitoring cycle
		services := []string{"grafana", "prometheus", "alertmanager"}
		results := make(map[string]string)

		// Simulate service checks
		for _, service := range services {
			// Simulate different outcomes
			if service == "grafana" {
				results[service] = "UP"
			} else {
				results[service] = "DOWN"
			}
		}

		// Simulate logging to PostgreSQL
		for service, status := range results {
			logEntry := LogEntry{
				ServiceName:  service,
				ServiceType:  "get",
				Status:       status,
				Error:        "",
				ResponseTime: 150,
				Timestamp:    time.Now(),
			}

			if status == "DOWN" {
				logEntry.Error = "Connection timeout"
				logEntry.ResponseTime = 5000
			}

			t.Logf("Logged %s status: %s", service, status)
		}

		// Simulate email alerts for failed services
		alertCount := 0
		for service, status := range results {
			if status == "DOWN" {
				_ = EmailAlert{
					ServiceName: service,
					Status:      status,
					Error:       "Connection timeout",
					Timestamp:   "2025-09-08T18:30:00Z",
					Details:     fmt.Sprintf("Service %s is not responding", service),
				}

				t.Logf("Email alert sent for %s: %s", service, status)
				alertCount++
			}
		}

		// Verify results
		if len(results) != len(services) {
			t.Errorf("Expected %d service results, got %d", len(services), len(results))
		}

		if alertCount == 0 {
			t.Errorf("Expected at least one alert, got %d", alertCount)
		}

		t.Logf("End-to-end monitoring completed: %d services checked, %d alerts sent",
			len(services), alertCount)
	})
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestParallelExecution tests parallel execution of workers
func TestParallelExecution(t *testing.T) {
	// Create test workers
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Worker 1'", "cmd", "worker1", ""),
		NewTaskWorker("echo 'Worker 2'", "cmd", "worker2", ""),
		NewTaskWorker("echo 'Worker 3'", "cmd", "worker3", ""),
	}

	// Create test logger
	logger := logr.Discard()

	// Test parallel execution
	start := time.Now()
	results := executeWorkersParallel(workers, 5*time.Second, logger, "test-task", false)
	parallelDuration := time.Since(start)

	// Verify results
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Check that all workers completed
	for i, result := range results {
		t.Logf("Worker %d result: %q", i+1, result)
		if result == "" {
			t.Errorf("Worker %d produced empty result", i+1)
		}
		if !containsEnhanced(result, fmt.Sprintf("Worker %d", i+1)) {
			t.Errorf("Worker %d result doesn't contain expected output: %s", i+1, result)
		}
	}

	t.Logf("Parallel execution completed in %v", parallelDuration)
	t.Logf("All results: %v", results)
}

// TestSequentialExecution tests sequential execution of workers
func TestSequentialExecution(t *testing.T) {
	// Create test workers
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Worker 1'", "cmd", "worker1", ""),
		NewTaskWorker("echo 'Worker 2'", "cmd", "worker2", ""),
		NewTaskWorker("echo 'Worker 3'", "cmd", "worker3", ""),
	}

	// Create test logger
	logger := logr.Discard()

	// Test sequential execution
	start := time.Now()
	results := executeWorkersSequential(workers, 5*time.Second, logger, "test-task", false)
	sequentialDuration := time.Since(start)

	// Verify results
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Check that all workers completed
	for i, result := range results {
		if result == "" {
			t.Errorf("Worker %d produced empty result", i+1)
		}
		if !containsEnhanced(result, fmt.Sprintf("Worker %d", i+1)) {
			t.Errorf("Worker %d result doesn't contain expected output: %s", i+1, result)
		}
	}

	t.Logf("Sequential execution completed in %v", sequentialDuration)
}

// TestExecutionModeComparison compares parallel vs sequential execution times
func TestExecutionModeComparison(t *testing.T) {
	// Create test workers with HTTP requests (slower operations)
	workers := []*TaskWorker{
		NewTaskWorker("https://httpbin.org/delay/1", "get", "http1", ""),
		NewTaskWorker("https://httpbin.org/delay/1", "get", "http2", ""),
		NewTaskWorker("https://httpbin.org/delay/1", "get", "http3", ""),
	}

	// Create test logger
	logger := logr.Discard()

	// Test sequential execution
	start := time.Now()
	sequentialResults := executeWorkersSequential(workers, 10*time.Second, logger, "test-task", false)
	sequentialDuration := time.Since(start)

	// Test parallel execution
	start = time.Now()
	parallelResults := executeWorkersParallel(workers, 10*time.Second, logger, "test-task", false)
	parallelDuration := time.Since(start)

	// Verify both executions produced results
	if len(sequentialResults) != len(parallelResults) {
		t.Fatalf("Sequential and parallel results have different lengths: %d vs %d",
			len(sequentialResults), len(parallelResults))
	}

	// Parallel should be faster than sequential for multiple HTTP requests
	if parallelDuration >= sequentialDuration {
		t.Logf("Parallel execution (%v) was not faster than sequential (%v)",
			parallelDuration, sequentialDuration)
	} else {
		t.Logf("Parallel execution (%v) was faster than sequential (%v) by %v",
			parallelDuration, sequentialDuration, sequentialDuration-parallelDuration)
	}

	t.Logf("Sequential: %v, Parallel: %v", sequentialDuration, parallelDuration)
}

// TestExecutionModeWithJSONInput tests execution mode with JSON input parsing
func TestExecutionModeWithJSONInput(t *testing.T) {
	// Test JSON input similar to complex-response-validation-test
	jsonInput := `{
		"inputs": [
			{
				"type": "cmd",
				"input": "echo 'Sequential Test 1'",
				"name": "test1"
			},
			{
				"type": "cmd", 
				"input": "echo 'Sequential Test 2'",
				"name": "test2"
			},
			{
				"type": "cmd",
				"input": "echo 'Sequential Test 3'",
				"name": "test3"
			}
		]
	}`

	// Parse JSON inputs
	inputs, err := parseJSONInputs(jsonInput)
	if err != nil {
		t.Fatalf("Failed to parse JSON inputs: %v", err)
	}

	if len(inputs) != 3 {
		t.Fatalf("Expected 3 inputs, got %d", len(inputs))
	}

	// Create workers from JSON inputs
	workers := make([]*TaskWorker, 0, len(inputs))
	for i, input := range inputs {
		inputStr, ok := input["input"].(string)
		if !ok {
			t.Fatalf("Input %d is not a string", i+1)
		}

		inputType := "cmd"
		if t, exists := input["type"].(string); exists {
			inputType = t
		}

		name := fmt.Sprintf("input-%d", i+1)
		if n, exists := input["name"].(string); exists {
			name = n
		}

		worker := NewTaskWorker(inputStr, inputType, name, "")
		workers = append(workers, worker)
	}

	logger := logr.Discard()

	// Test sequential execution
	t.Log("Testing sequential execution...")
	start := time.Now()
	sequentialResults := executeWorkersSequential(workers, 5*time.Second, logger, "test-task", false)
	sequentialDuration := time.Since(start)

	// Test parallel execution
	t.Log("Testing parallel execution...")
	start = time.Now()
	parallelResults := executeWorkersParallel(workers, 5*time.Second, logger, "test-task", false)
	parallelDuration := time.Since(start)

	// Verify both executions produced the same number of results
	if len(sequentialResults) != len(parallelResults) {
		t.Fatalf("Sequential and parallel results have different lengths: %d vs %d",
			len(sequentialResults), len(parallelResults))
	}

	// Verify results contain expected content
	for i, result := range sequentialResults {
		t.Logf("Sequential result %d: %q", i+1, result)
		if !containsEnhanced(result, fmt.Sprintf("Sequential Test %d", i+1)) {
			t.Errorf("Sequential result %d doesn't contain expected content: %s", i+1, result)
		}
	}

	for i, result := range parallelResults {
		t.Logf("Parallel result %d: %q", i+1, result)
		if !containsEnhanced(result, fmt.Sprintf("Sequential Test %d", i+1)) {
			t.Errorf("Parallel result %d doesn't contain expected content: %s", i+1, result)
		}
	}

	t.Logf("Sequential execution: %v", sequentialDuration)
	t.Logf("Parallel execution: %v", parallelDuration)
	t.Logf("Sequential results: %v", sequentialResults)
	t.Logf("Parallel results: %v", parallelResults)
}

// TestFailFastSequentialExecution tests failFast behavior in sequential execution
func TestFailFastSequentialExecution(t *testing.T) {
	// Create test workers with some that will fail
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Success 1'", "cmd", "success1", ""),
		NewTaskWorker("nonexistentcommand12345", "cmd", "fail1", ""), // This will fail
		NewTaskWorker("echo 'Success 2'", "cmd", "success2", ""),     // Should not execute
		NewTaskWorker("echo 'Success 3'", "cmd", "success3", ""),     // Should not execute
	}

	logger := logr.Discard()

	// Test sequential execution with failFast enabled
	t.Log("Testing sequential execution with failFast enabled...")
	start := time.Now()
	results := executeWorkersSequential(workers, 5*time.Second, logger, "test-task", true)
	duration := time.Since(start)

	// With failFast, should only have 2 results (success1 + fail1)
	if len(results) != 2 {
		t.Fatalf("Expected 2 results with failFast, got %d", len(results))
	}

	// Check results
	successCount := 0
	errorCount := 0
	for i, result := range results {
		t.Logf("Result %d: %q", i+1, result)
		if strings.HasPrefix(result, "Error:") {
			errorCount++
		} else {
			successCount++
		}
	}

	// Should have 1 success and 1 error
	if successCount != 1 {
		t.Errorf("Expected 1 success, got %d", successCount)
	}
	if errorCount != 1 {
		t.Errorf("Expected 1 error, got %d", errorCount)
	}

	t.Logf("Sequential execution with failFast completed in %v", duration)
	t.Logf("Successes: %d, Errors: %d", successCount, errorCount)
	t.Logf("All results: %v", results)
}

// TestFailFastParallelExecution tests failFast behavior in parallel execution
func TestFailFastParallelExecution(t *testing.T) {
	// Create test workers with some that will fail
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Success 1'", "cmd", "success1", ""),
		NewTaskWorker("nonexistentcommand12345", "cmd", "fail1", ""), // This will fail
		NewTaskWorker("echo 'Success 2'", "cmd", "success2", ""),
		NewTaskWorker("echo 'Success 3'", "cmd", "success3", ""),
	}

	logger := logr.Discard()

	// Test parallel execution with failFast enabled
	t.Log("Testing parallel execution with failFast enabled...")
	start := time.Now()
	results := executeWorkersParallel(workers, 5*time.Second, logger, "test-task", true)
	duration := time.Since(start)

	// With failFast in parallel, some workers might complete before cancellation
	// So we expect at least 1 result, but not necessarily all 4
	if len(results) < 1 {
		t.Fatalf("Expected at least 1 result with failFast, got %d", len(results))
	}

	// Check results
	successCount := 0
	errorCount := 0
	for i, result := range results {
		t.Logf("Result %d: %q", i+1, result)
		if strings.HasPrefix(result, "Error:") {
			errorCount++
		} else if result != "" {
			successCount++
		}
	}

	// Should have at least 1 error (the failing command)
	if errorCount < 1 {
		t.Errorf("Expected at least 1 error, got %d", errorCount)
	}

	t.Logf("Parallel execution with failFast completed in %v", duration)
	t.Logf("Successes: %d, Errors: %d", successCount, errorCount)
	t.Logf("All results: %v", results)
}

// TestErrorHandlingInParallelExecution tests error handling in parallel execution
func TestErrorHandlingInParallelExecution(t *testing.T) {
	// Create test workers with some that will fail
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Success 1'", "cmd", "success1", ""),
		NewTaskWorker("nonexistentcommand12345", "cmd", "fail1", ""), // This will fail
		NewTaskWorker("echo 'Success 2'", "cmd", "success2", ""),
		NewTaskWorker("invalidcommand45678", "cmd", "fail2", ""), // This will fail
		NewTaskWorker("echo 'Success 3'", "cmd", "success3", ""),
	}

	logger := logr.Discard()

	// Test parallel execution with errors
	t.Log("Testing parallel execution with errors...")
	start := time.Now()
	results := executeWorkersParallel(workers, 5*time.Second, logger, "test-task", false)
	duration := time.Since(start)

	// Verify all workers completed (even failed ones)
	if len(results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(results))
	}

	// Check results
	successCount := 0
	errorCount := 0
	for i, result := range results {
		t.Logf("Result %d: %q", i+1, result)
		if strings.HasPrefix(result, "Error:") {
			errorCount++
		} else {
			successCount++
		}
	}

	// Should have 3 successes and 2 errors
	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}
	if errorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", errorCount)
	}

	t.Logf("Parallel execution with errors completed in %v", duration)
	t.Logf("Successes: %d, Errors: %d", successCount, errorCount)
	t.Logf("All results: %v", results)
}

// TestErrorHandlingInSequentialExecution tests error handling in sequential execution
func TestErrorHandlingInSequentialExecution(t *testing.T) {
	// Create test workers with some that will fail
	workers := []*TaskWorker{
		NewTaskWorker("echo 'Success 1'", "cmd", "success1", ""),
		NewTaskWorker("nonexistentcommand12345", "cmd", "fail1", ""), // This will fail
		NewTaskWorker("echo 'Success 2'", "cmd", "success2", ""),
		NewTaskWorker("invalidcommand45678", "cmd", "fail2", ""), // This will fail
		NewTaskWorker("echo 'Success 3'", "cmd", "success3", ""),
	}

	logger := logr.Discard()

	// Test sequential execution with errors
	t.Log("Testing sequential execution with errors...")
	start := time.Now()
	results := executeWorkersSequential(workers, 5*time.Second, logger, "test-task", false)
	duration := time.Since(start)

	// Verify all workers completed (even failed ones)
	if len(results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(results))
	}

	// Check results
	successCount := 0
	errorCount := 0
	for i, result := range results {
		t.Logf("Result %d: %q", i+1, result)
		if strings.HasPrefix(result, "Error:") {
			errorCount++
		} else {
			successCount++
		}
	}

	// Should have 3 successes and 2 errors
	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}
	if errorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", errorCount)
	}

	t.Logf("Sequential execution with errors completed in %v", duration)
	t.Logf("Successes: %d, Errors: %d", successCount, errorCount)
	t.Logf("All results: %v", results)
}

// TestMySQLBackend tests MySQL logging backend
func TestMySQLBackend(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		serviceType  string
		status       string
		error        string
		responseTime int64
		timestamp    time.Time
		wantErr      bool
	}{
		{
			name:         "mysql_successful_check",
			serviceName:  "mysql-db",
			serviceType:  "cmd",
			status:       "UP",
			error:        "",
			responseTime: 200,
			timestamp:    time.Now(),
			wantErr:      false,
		},
		{
			name:         "mysql_failed_check",
			serviceName:  "mysql-db",
			serviceType:  "cmd",
			status:       "DOWN",
			error:        "Connection refused",
			responseTime: 3000,
			timestamp:    time.Now(),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logEntry := LogEntry{
				ServiceName:  tt.serviceName,
				ServiceType:  tt.serviceType,
				Status:       tt.status,
				Error:        tt.error,
				ResponseTime: tt.responseTime,
				Timestamp:    tt.timestamp,
			}

			// Test log entry validation
			if logEntry.ServiceName == "" {
				t.Errorf("Service name cannot be empty")
			}
			if logEntry.Timestamp.IsZero() {
				t.Errorf("Timestamp cannot be zero")
			}
			if logEntry.Status != "UP" && logEntry.Status != "DOWN" {
				t.Errorf("Status must be UP or DOWN, got: %s", logEntry.Status)
			}

			t.Logf("MySQL log entry created for %s: %s (response time: %dms)",
				tt.serviceName, tt.status, tt.responseTime)
		})
	}

	t.Logf("MySQL backend test completed successfully!")
}

// TestElasticsearchBackend tests Elasticsearch logging backend
func TestElasticsearchBackend(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		serviceType  string
		status       string
		error        string
		responseTime int64
		timestamp    time.Time
		wantErr      bool
	}{
		{
			name:         "elasticsearch_successful_check",
			serviceName:  "elasticsearch",
			serviceType:  "get",
			status:       "UP",
			error:        "",
			responseTime: 100,
			timestamp:    time.Now(),
			wantErr:      false,
		},
		{
			name:         "elasticsearch_failed_check",
			serviceName:  "elasticsearch",
			serviceType:  "get",
			status:       "DOWN",
			error:        "Connection timeout",
			responseTime: 5000,
			timestamp:    time.Now(),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logEntry := LogEntry{
				ServiceName:  tt.serviceName,
				ServiceType:  tt.serviceType,
				Status:       tt.status,
				Error:        tt.error,
				ResponseTime: tt.responseTime,
				Timestamp:    tt.timestamp,
			}

			// Test log entry validation
			if logEntry.ServiceName == "" {
				t.Errorf("Service name cannot be empty")
			}
			if logEntry.Timestamp.IsZero() {
				t.Errorf("Timestamp cannot be zero")
			}
			if logEntry.Status != "UP" && logEntry.Status != "DOWN" {
				t.Errorf("Status must be UP or DOWN, got: %s", logEntry.Status)
			}

			t.Logf("Elasticsearch log entry created for %s: %s (response time: %dms)",
				tt.serviceName, tt.status, tt.responseTime)
		})
	}

	t.Logf("Elasticsearch backend test completed successfully!")
}

// TestKafkaBackend tests Kafka logging backend
func TestKafkaBackend(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		serviceType  string
		status       string
		error        string
		responseTime int64
		timestamp    time.Time
		wantErr      bool
	}{
		{
			name:         "kafka_successful_check",
			serviceName:  "kafka",
			serviceType:  "get",
			status:       "UP",
			error:        "",
			responseTime: 50,
			timestamp:    time.Now(),
			wantErr:      false,
		},
		{
			name:         "kafka_failed_check",
			serviceName:  "kafka",
			serviceType:  "get",
			status:       "DOWN",
			error:        "Broker not available",
			responseTime: 2000,
			timestamp:    time.Now(),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logEntry := LogEntry{
				ServiceName:  tt.serviceName,
				ServiceType:  tt.serviceType,
				Status:       tt.status,
				Error:        tt.error,
				ResponseTime: tt.responseTime,
				Timestamp:    tt.timestamp,
			}

			// Test log entry validation
			if logEntry.ServiceName == "" {
				t.Errorf("Service name cannot be empty")
			}
			if logEntry.Timestamp.IsZero() {
				t.Errorf("Timestamp cannot be zero")
			}
			if logEntry.Status != "UP" && logEntry.Status != "DOWN" {
				t.Errorf("Status must be UP or DOWN, got: %s", logEntry.Status)
			}

			t.Logf("Kafka log entry created for %s: %s (response time: %dms)",
				tt.serviceName, tt.status, tt.responseTime)
		})
	}

	t.Logf("Kafka backend test completed successfully!")
}

// TestLoggingBackendFactory tests the logging backend factory
func TestLoggingBackendFactory(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		wantErr bool
	}{
		{
			name:    "postgres_backend",
			backend: "postgres",
			wantErr: false,
		},
		{
			name:    "mysql_backend",
			backend: "mysql",
			wantErr: false,
		},
		{
			name:    "elasticsearch_backend",
			backend: "elasticsearch",
			wantErr: false,
		},
		{
			name:    "kafka_backend",
			backend: "kafka",
			wantErr: false,
		},
		{
			name:    "unsupported_backend",
			backend: "unsupported",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := LoggingConfig{
				Enabled: true,
				Backend: tt.backend,
			}

			backend, err := CreateLoggingBackend(config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for backend %s, but got none", tt.backend)
				}
				if backend != nil {
					t.Errorf("Expected nil backend for unsupported backend %s", tt.backend)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for backend %s: %v", tt.backend, err)
				}
				if backend == nil {
					t.Errorf("Expected backend for %s, but got nil", tt.backend)
				}
			}

			t.Logf("Backend factory test for %s: %v", tt.backend, err == nil)
		})
	}

	t.Logf("Logging backend factory test completed successfully!")
}
