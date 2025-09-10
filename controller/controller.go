package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcallv1 "github.com/doohee323/tz-mcall-crd/api/v1"
)

// LogEntry represents a log entry for storage
type LogEntry struct {
	ServiceName  string
	ServiceType  string
	Status       string
	Error        string
	ResponseTime int64
	Timestamp    time.Time
}

// LoggingBackend defines the interface for different logging backends
type LoggingBackend interface {
	Connect() error
	Log(entry LogEntry) error
	Close() error
	IsEnabled() bool
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Enabled bool
	Backend string // "postgres", "mysql", "elasticsearch", "kafka"

	// PostgreSQL configuration
	PostgreSQL struct {
		Enabled  bool
		Host     string
		Port     int
		Username string
		Password string
		Database string
		SSLMode  string
		Table    struct {
			Name       string
			AutoCreate bool
		}
	}

	// MySQL configuration
	MySQL struct {
		Enabled  bool
		Host     string
		Port     int
		Username string
		Password string
		Database string
		Table    struct {
			Name       string
			AutoCreate bool
		}
	}

	// Elasticsearch configuration
	Elasticsearch struct {
		Enabled  bool
		URL      string
		Index    string
		Username string
		Password string
	}

	// Kafka configuration
	Kafka struct {
		Enabled bool
		Brokers []string
		Topic   string
	}
}

// PostgreSQLBackend implements LoggingBackend for PostgreSQL
type PostgreSQLBackend struct {
	config LoggingConfig
	db     *sql.DB
}

func (p *PostgreSQLBackend) Connect() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.config.PostgreSQL.Host,
		p.config.PostgreSQL.Port,
		p.config.PostgreSQL.Username,
		p.config.PostgreSQL.Password,
		p.config.PostgreSQL.Database,
		p.config.PostgreSQL.SSLMode,
	)

	var err error
	p.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return p.db.Ping()
}

func (p *PostgreSQLBackend) Log(entry LogEntry) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (service_name, service_type, status, error_message, response_time_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.config.PostgreSQL.Table.Name)

	_, err := p.db.Exec(query,
		entry.ServiceName,
		entry.ServiceType,
		entry.Status,
		entry.Error,
		entry.ResponseTime,
		entry.Timestamp,
	)

	return err
}

func (p *PostgreSQLBackend) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgreSQLBackend) IsEnabled() bool {
	return p.config.PostgreSQL.Enabled
}

// MySQLBackend implements LoggingBackend for MySQL
type MySQLBackend struct {
	config LoggingConfig
	db     *sql.DB
}

func (m *MySQLBackend) Connect() error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.config.MySQL.Username,
		m.config.MySQL.Password,
		m.config.MySQL.Host,
		m.config.MySQL.Port,
		m.config.MySQL.Database,
	)

	var err error
	m.db, err = sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	return m.db.Ping()
}

func (m *MySQLBackend) Log(entry LogEntry) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (service_name, service_type, status, error_message, response_time_ms, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, m.config.MySQL.Table.Name)

	_, err := m.db.Exec(query,
		entry.ServiceName,
		entry.ServiceType,
		entry.Status,
		entry.Error,
		entry.ResponseTime,
		entry.Timestamp,
	)

	return err
}

func (m *MySQLBackend) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQLBackend) IsEnabled() bool {
	return m.config.MySQL.Enabled
}

// ElasticsearchBackend implements LoggingBackend for Elasticsearch
type ElasticsearchBackend struct {
	config LoggingConfig
	client *http.Client
}

func (e *ElasticsearchBackend) Connect() error {
	e.client = &http.Client{Timeout: 10 * time.Second}
	// Test connection
	resp, err := e.client.Get(e.config.Elasticsearch.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Elasticsearch connection failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *ElasticsearchBackend) Log(entry LogEntry) error {
	// Create JSON document
	doc := map[string]interface{}{
		"service_name":     entry.ServiceName,
		"service_type":     entry.ServiceType,
		"status":           entry.Status,
		"error_message":    entry.Error,
		"response_time_ms": entry.ResponseTime,
		"timestamp":        entry.Timestamp,
	}

	jsonData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Send to Elasticsearch
	url := fmt.Sprintf("%s/%s/_doc", e.config.Elasticsearch.URL, e.config.Elasticsearch.Index)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to Elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Elasticsearch request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *ElasticsearchBackend) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

func (e *ElasticsearchBackend) IsEnabled() bool {
	return e.config.Elasticsearch.Enabled
}

// KafkaBackend implements LoggingBackend for Kafka
type KafkaBackend struct {
	config LoggingConfig
	// In a real implementation, you would use a Kafka client library
	// For now, we'll use HTTP for simplicity
	client *http.Client
}

func (k *KafkaBackend) Connect() error {
	k.client = &http.Client{Timeout: 10 * time.Second}
	// Kafka connection logic would go here
	// For now, we'll assume connection is successful
	return nil
}

func (k *KafkaBackend) Log(entry LogEntry) error {
	// Create JSON message
	message := map[string]interface{}{
		"service_name":     entry.ServiceName,
		"service_type":     entry.ServiceType,
		"status":           entry.Status,
		"error_message":    entry.Error,
		"response_time_ms": entry.ResponseTime,
		"timestamp":        entry.Timestamp,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// In a real implementation, you would send to Kafka
	// For now, we'll log the message
	fmt.Printf("Kafka message: %s\n", string(jsonData))

	return nil
}

func (k *KafkaBackend) Close() error {
	// Kafka client cleanup would go here
	return nil
}

func (k *KafkaBackend) IsEnabled() bool {
	return k.config.Kafka.Enabled
}

// GetLoggingConfig returns the logging configuration from environment variables
func GetLoggingConfig() LoggingConfig {
	config := LoggingConfig{}

	// Check if logging is enabled
	config.Enabled = os.Getenv("LOGGING_ENABLED") == "true"
	if !config.Enabled {
		return config
	}

	// Get backend type
	config.Backend = getEnvOrDefault("LOGGING_BACKEND", "postgres")

	// PostgreSQL configuration
	config.PostgreSQL.Enabled = os.Getenv("LOGGING_POSTGRESQL_ENABLED") == "true"
	config.PostgreSQL.Host = getEnvOrDefault("LOGGING_POSTGRESQL_HOST", "devops-postgres-postgresql.devops.svc.cluster.local")
	config.PostgreSQL.Port = getEnvIntOrDefault("LOGGING_POSTGRESQL_PORT", 5432)
	config.PostgreSQL.Username = getEnvOrDefault("LOGGING_POSTGRESQL_USERNAME", "admin")
	config.PostgreSQL.Password = getEnvOrDefault("LOGGING_POSTGRES_PASSWORD", "")
	config.PostgreSQL.Database = getEnvOrDefault("LOGGING_POSTGRESQL_DATABASE", "mcall_logs")
	config.PostgreSQL.SSLMode = getEnvOrDefault("LOGGING_POSTGRESQL_SSLMODE", "disable")
	config.PostgreSQL.Table.Name = getEnvOrDefault("LOGGING_POSTGRESQL_TABLE_NAME", "monitoring_logs")
	config.PostgreSQL.Table.AutoCreate = os.Getenv("LOGGING_POSTGRESQL_TABLE_AUTOCREATE") == "true"

	// MySQL configuration
	config.MySQL.Enabled = os.Getenv("LOGGING_MYSQL_ENABLED") == "true"
	config.MySQL.Host = getEnvOrDefault("LOGGING_MYSQL_HOST", "localhost")
	config.MySQL.Port = getEnvIntOrDefault("LOGGING_MYSQL_PORT", 3306)
	config.MySQL.Username = getEnvOrDefault("LOGGING_MYSQL_USERNAME", "root")
	config.MySQL.Password = getEnvOrDefault("LOGGING_MYSQL_PASSWORD", "")
	config.MySQL.Database = getEnvOrDefault("LOGGING_MYSQL_DATABASE", "mcall_logs")
	config.MySQL.Table.Name = getEnvOrDefault("LOGGING_MYSQL_TABLE_NAME", "monitoring_logs")
	config.MySQL.Table.AutoCreate = os.Getenv("LOGGING_MYSQL_TABLE_AUTOCREATE") == "true"

	// Elasticsearch configuration
	config.Elasticsearch.Enabled = os.Getenv("LOGGING_ELASTICSEARCH_ENABLED") == "true"
	config.Elasticsearch.URL = getEnvOrDefault("LOGGING_ELASTICSEARCH_URL", "http://localhost:9200")
	config.Elasticsearch.Index = getEnvOrDefault("LOGGING_ELASTICSEARCH_INDEX", "mcall-logs")
	config.Elasticsearch.Username = getEnvOrDefault("LOGGING_ELASTICSEARCH_USERNAME", "")
	config.Elasticsearch.Password = getEnvOrDefault("LOGGING_ELASTICSEARCH_PASSWORD", "")

	// Kafka configuration
	config.Kafka.Enabled = os.Getenv("LOGGING_KAFKA_ENABLED") == "true"
	config.Kafka.Topic = getEnvOrDefault("LOGGING_KAFKA_TOPIC", "mcall-logs")
	// Kafka brokers would be parsed from environment variable
	brokersStr := getEnvOrDefault("LOGGING_KAFKA_BROKERS", "localhost:9092")
	config.Kafka.Brokers = strings.Split(brokersStr, ",")

	return config
}

// getEnvOrDefault returns environment variable value or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault returns environment variable as int or default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// CreateLoggingBackend creates the appropriate logging backend based on configuration
func CreateLoggingBackend(config LoggingConfig) (LoggingBackend, error) {
	switch config.Backend {
	case "postgres":
		return &PostgreSQLBackend{config: config}, nil
	case "mysql":
		return &MySQLBackend{config: config}, nil
	case "elasticsearch":
		return &ElasticsearchBackend{config: config}, nil
	case "kafka":
		return &KafkaBackend{config: config}, nil
	default:
		return nil, fmt.Errorf("unsupported logging backend: %s", config.Backend)
	}
}

// LogToBackend logs the task execution result to the configured backend
func LogToBackend(logEntry LogEntry, config LoggingConfig) error {
	if !config.Enabled {
		return nil // Logging disabled
	}

	// Create backend
	backend, err := CreateLoggingBackend(config)
	if err != nil {
		return fmt.Errorf("failed to create logging backend: %w", err)
	}

	// Check if backend is enabled
	if !backend.IsEnabled() {
		return nil // Backend disabled
	}

	// Connect to backend
	if err := backend.Connect(); err != nil {
		return fmt.Errorf("failed to connect to logging backend: %w", err)
	}
	defer backend.Close()

	// Log the entry
	if err := backend.Log(logEntry); err != nil {
		return fmt.Errorf("failed to log entry: %w", err)
	}

	return nil
}

// getReconcileInterval returns the reconcile interval from environment variable
func getReconcileInterval() time.Duration {
	intervalStr := os.Getenv("RECONCILE_INTERVAL")
	if intervalStr == "" {
		return 5 * time.Second // default value
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		return 5 * time.Second // default value
	}

	return time.Duration(interval) * time.Second
}

// getTaskTimeout returns the task timeout from environment variable
func getTaskTimeout() time.Duration {
	timeoutStr := os.Getenv("TASK_TIMEOUT")
	if timeoutStr == "" {
		return 5 * time.Second // default value
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeout <= 0 {
		return 5 * time.Second // default value
	}

	return time.Duration(timeout) * time.Second
}

// executeCommand executes a shell command with timeout
func executeCommand(command string, timeout time.Duration) (string, error) {
	if command == "" {
		return "", fmt.Errorf("empty command")
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmdName := parts[0]
	args := parts[1:]

	// Clean up arguments (from original mcall.go)
	for i := range args {
		if args[i] == "'Content-Type_application/json'" {
			args[i] = "'Content-Type: application/json'"
		} else {
			args[i] = strings.Replace(args[i], "`", " ", -1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command execution timed out")
		}
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// executeHTTPRequest executes HTTP GET/POST request
func executeHTTPRequest(url, method string, timeout time.Duration) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty URL")
	}

	var req *http.Request
	var err error

	if method == "POST" {
		// For POST requests, we might need to extract data from the URL
		// This is a simplified implementation - you might want to enhance it
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create GET request: %w", err)
		}
	}

	// Set User-Agent header to avoid 403 Forbidden errors
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute %s request: %w", method, err)
	}
	defer resp.Body.Close()

	doc, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Return only body content (like mcall.go fetchHtml)
	return string(doc), nil
}

// getHTTPStatusCode gets the HTTP status code for expect validation (like mcall.go)
func getHTTPStatusCode(url, method string, timeout time.Duration) string {
	if url == "" {
		return ""
	}

	var req *http.Request
	var err error

	if method == "POST" {
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return ""
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return ""
		}
	}

	// Set User-Agent header to avoid 403 Forbidden errors
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	return strconv.Itoa(resp.StatusCode)
}

// TaskResult represents the result of a task execution (based on mcall.go FetchedResult)
type TaskResult struct {
	Input   string `json:"input"`
	Name    string `json:"name"`
	Error   string `json:"errorCode"`
	Content string `json:"result"`
	TS      string `json:"ts"`
}

// TaskWorker represents a single task execution (based on mcall.go CallFetch)
type TaskWorker struct {
	input     string
	inputType string
	name      string
	expect    string // For expect validation (like mcall.go) - supports HTTP status codes and response body text
	result    chan TaskResult
}

// NewTaskWorker creates a new TaskWorker instance
func NewTaskWorker(input, inputType, name, expect string) *TaskWorker {
	return &TaskWorker{
		input:     input,
		inputType: inputType,
		name:      name,
		expect:    expect,
		result:    make(chan TaskResult, 1),
	}
}

// Execute implements the task execution (based on mcall.go CallFetch.Execute)
func (tw *TaskWorker) Execute(timeout time.Duration) {
	var content string
	var err error

	// Execute based on type (like original mcall.go)
	var statusCode string
	switch tw.inputType {
	case "cmd":
		content, err = executeCommand(tw.input, timeout)
	case "get":
		content, err = executeHTTPRequest(tw.input, "GET", timeout)
		if err == nil {
			// For HTTP requests, we need to get status code for expect validation
			statusCode = getHTTPStatusCode(tw.input, "GET", timeout)
		}
	case "post":
		content, err = executeHTTPRequest(tw.input, "POST", timeout)
		if err == nil {
			// For HTTP requests, we need to get status code for expect validation
			statusCode = getHTTPStatusCode(tw.input, "POST", timeout)
		}
	default:
		content, err = executeCommand(tw.input, timeout)
	}

	// Validate expect string (like mcall.go checkRslt)
	if tw.expect != "" && err == nil {
		var expectContent string
		if tw.inputType == "cmd" {
			// For cmd, check command output
			expectContent = content
		} else {
			// For HTTP requests, check both status code and response body
			// Format: "statusCode|responseBody" so expect can match either
			expectContent = statusCode + "|" + content
		}

		if !checkExpect(expectContent, tw.expect) {
			err = fmt.Errorf("expect validation failed: expected %s in %s", tw.expect, expectContent)
		}
	}

	// Set error code (like original mcall.go)
	var errCode string
	if err != nil {
		errCode = "-1" // ErrorCodeFailure
	} else {
		errCode = "0" // ErrorCodeSuccess
	}

	// Create result (like original mcall.go FetchedResult)
	now := time.Now().UTC()
	result := TaskResult{
		Input:   tw.input,
		Name:    tw.name,
		Error:   errCode,
		Content: content,
		TS:      now.Format("2006-01-02T15:04:05.000"),
	}

	tw.result <- result
}

// executeWorkersSequential executes workers sequentially
func executeWorkersSequential(workers []*TaskWorker, timeout time.Duration, logger logr.Logger, taskName string, failFast bool) []string {
	var results []string

	for i, worker := range workers {
		logger.Info("Executing input",
			"task", taskName,
			"input", i+1,
			"type", worker.inputType,
			"command", worker.input)

		// Execute worker
		worker.Execute(timeout)

		// Get result
		result := <-worker.result

		// Debug logging
		logger.Info("TaskWorker result",
			"task", taskName,
			"input", i+1,
			"type", worker.inputType,
			"errorCode", result.Error,
			"content", result.Content)

		// Format result
		if result.Error == "-1" {
			results = append(results, fmt.Sprintf("Error: %s", result.Content))
			logger.Error(fmt.Errorf("input execution failed"), "Input execution failed",
				"task", taskName,
				"input", i+1,
				"type", worker.inputType,
				"error", result.Content)

			// If failFast is enabled, stop execution on first error
			if failFast {
				logger.Info("FailFast enabled, stopping execution on first error",
					"task", taskName,
					"input", i+1,
					"error", result.Content)
				break
			}
		} else {
			results = append(results, result.Content)
			logger.Info("Input execution succeeded",
				"task", taskName,
				"input", i+1,
				"type", worker.inputType)
		}
	}

	return results
}

// executeWorkersParallel executes workers in parallel using goroutines
func executeWorkersParallel(workers []*TaskWorker, timeout time.Duration, logger logr.Logger, taskName string, failFast bool) []string {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string
	var hasError bool

	// Initialize results slice with correct size
	results = make([]string, len(workers))

	// Create context for cancellation if failFast is enabled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Execute all workers in parallel
	for i, worker := range workers {
		wg.Add(1)
		go func(index int, w *TaskWorker) {
			defer wg.Done()

			// Check if context is cancelled (failFast)
			select {
			case <-ctx.Done():
				logger.Info("Worker cancelled due to failFast",
					"task", taskName,
					"input", index+1)
				return
			default:
			}

			logger.Info("Executing input",
				"task", taskName,
				"input", index+1,
				"type", w.inputType,
				"command", w.input)

			// Execute worker
			w.Execute(timeout)

			// Get result
			result := <-w.result

			// Debug logging
			logger.Info("TaskWorker result",
				"task", taskName,
				"input", index+1,
				"type", w.inputType,
				"errorCode", result.Error,
				"content", result.Content)

			// Format result and store in correct position
			mu.Lock()
			if result.Error == "-1" {
				results[index] = fmt.Sprintf("Error: %s", result.Content)
				logger.Error(fmt.Errorf("input execution failed"), "Input execution failed",
					"task", taskName,
					"input", index+1,
					"type", w.inputType,
					"error", result.Content)

				// If failFast is enabled, cancel other workers
				if failFast && !hasError {
					hasError = true
					logger.Info("FailFast enabled, cancelling other workers on first error",
						"task", taskName,
						"input", index+1,
						"error", result.Content)
					cancel()
				}
			} else {
				results[index] = result.Content
				logger.Info("Input execution succeeded",
					"task", taskName,
					"input", index+1,
					"type", w.inputType)
			}
			mu.Unlock()
		}(i, worker)
	}

	// Wait for all workers to complete
	wg.Wait()

	return results
}

// parseJSONInputs parses JSON inputs array from input string (based on mcall.go parseConfigInput)
func parseJSONInputs(inputStr string) ([]map[string]interface{}, error) {
	// First try to parse as JSON array directly
	var inputs []map[string]interface{}
	if err := json.Unmarshal([]byte(inputStr), &inputs); err == nil {
		return inputs, nil
	}

	// Try to parse as JSON object with "inputs" field (like original mcall.go)
	type Inputs struct {
		Inputs []map[string]interface{} `json:"inputs"`
	}

	var data Inputs
	if err := json.Unmarshal([]byte(inputStr), &data); err == nil {
		return data.Inputs, nil
	}

	// If neither JSON array nor JSON object with inputs field, treat as single input
	singleInput := map[string]interface{}{
		"input": inputStr,
		"type":  "cmd",
		"name":  "single-task",
	}
	inputs = []map[string]interface{}{singleInput}

	return inputs, nil
}

// validateResponse validates HTTP response against expectedResponse
func validateResponse(content string, expected map[string]interface{}) bool {
	// For HTTP requests, content is just the response body (like mcall.go)
	body := content

	// Check expected status code (we need to get it separately)
	// This will be handled by the caller with status code validation

	// Check expected body content
	if expectedBody, exists := expected["body"]; exists {
		// First try string contains check (like mcall.go)
		if expectedBodyStr, ok := expectedBody.(string); ok {
			return strings.Contains(body, expectedBodyStr)
		}

		// Then try JSON object comparison
		expectedBodyBytes, err := json.Marshal(expectedBody)
		if err != nil {
			return false
		}

		// Try to parse actual body as JSON for comparison
		var actualBody, expectedBodyParsed interface{}
		if err := json.Unmarshal([]byte(body), &actualBody); err == nil {
			if err := json.Unmarshal(expectedBodyBytes, &expectedBodyParsed); err == nil {
				// Partial comparison of JSON objects (check if expected fields exist in actual)
				return compareJSONObjectsPartial(actualBody, expectedBodyParsed)
			}
		}

		// Final fallback: convert expected body to string and check contains
		expectedBodyStr := string(expectedBodyBytes)
		return strings.Contains(body, expectedBodyStr)
	}

	return true
}

// compareJSONObjectsPartial compares two JSON objects with partial matching
func compareJSONObjectsPartial(actual, expected interface{}) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}

	switch expectedVal := expected.(type) {
	case map[string]interface{}:
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		// Check if all expected fields exist in actual with matching values
		for key, expectedValue := range expectedVal {
			if actualValue, exists := actualMap[key]; !exists || !compareJSONObjectsPartial(actualValue, expectedValue) {
				return false
			}
		}
		return true
	case []interface{}:
		actualArray, ok := actual.([]interface{})
		if !ok {
			return false
		}
		// For arrays, check if all expected elements exist in actual
		for _, expectedItem := range expectedVal {
			found := false
			for _, actualItem := range actualArray {
				if compareJSONObjectsPartial(actualItem, expectedItem) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return actual == expected
	}
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// compareJSONObjects compares two JSON objects recursively
func compareJSONObjects(obj1, obj2 interface{}) bool {
	if obj1 == nil && obj2 == nil {
		return true
	}
	if obj1 == nil || obj2 == nil {
		return false
	}

	switch v1 := obj1.(type) {
	case map[string]interface{}:
		v2, ok := obj2.(map[string]interface{})
		if !ok {
			return false
		}
		if len(v1) != len(v2) {
			return false
		}
		for key, val1 := range v1 {
			if val2, exists := v2[key]; !exists || !compareJSONObjects(val1, val2) {
				return false
			}
		}
		return true
	case []interface{}:
		v2, ok := obj2.([]interface{})
		if !ok {
			return false
		}
		if len(v1) != len(v2) {
			return false
		}
		for i, val1 := range v1 {
			if !compareJSONObjects(val1, v2[i]) {
				return false
			}
		}
		return true
	default:
		return v1 == obj2
	}
}

// validateOutput validates command output against expectedOutput
func validateOutput(content, expected string) bool {
	// Simple string comparison
	return strings.Contains(content, expected)
}

// checkExpect validates result against expect string (based on mcall.go checkRslt)
// checkExpect validates content against expect string (based on mcall.go checkRslt)
func checkExpect(content, expect string) bool {
	if expect == "" {
		return true
	}

	// Split expect by "|" (OR condition like mcall.go)
	expectArray := strings.Split(expect, "|")

	for _, expectItem := range expectArray {
		// Check for count-based validation (like mcall.go)
		if strings.Contains(expectItem, "$count < ") || strings.Contains(expectItem, " > $count") ||
			strings.Contains(expectItem, "$count > ") || strings.Contains(expectItem, " < $count") {
			// Handle count validation (simplified version for now)
			continue
		} else {
			// Simple string contains check (like mcall.go)
			if strings.Contains(content, strings.TrimSpace(expectItem)) {
				return true // Any match is success
			}
		}
	}

	return false // No matches found
}

// McallTaskReconciler reconciles a McallTask object
type McallTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcalltasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcalltasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcalltasks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *McallTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("=== RECONCILE START ===", "task", req.NamespacedName)

	// Fetch the McallTask instance
	var mcallTask mcallv1.McallTask
	if err := r.Get(ctx, req.NamespacedName, &mcallTask); err != nil {
		log.Error(err, "unable to fetch McallTask")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Fetched McallTask", "task", mcallTask.Name, "currentPhase", mcallTask.Status.Phase, "phaseLength", len(mcallTask.Status.Phase))

	// Initialize status if not set (before any other processing)
	if len(mcallTask.Status.Phase) == 0 {
		log.Info("*** STATUS PHASE IS EMPTY - INITIALIZING TO PENDING ***", "task", mcallTask.Name)
		mcallTask.Status.Phase = mcallv1.McallTaskPhasePending
		log.Info("About to update status", "task", mcallTask.Name, "newPhase", mcallTask.Status.Phase)
		if err := r.Status().Update(ctx, &mcallTask); err != nil {
			log.Error(err, "*** FAILED TO INITIALIZE STATUS PHASE ***", "task", mcallTask.Name, "error", err.Error())
			return ctrl.Result{}, err
		}
		log.Info("*** SUCCESSFULLY INITIALIZED STATUS PHASE ***", "task", mcallTask.Name, "phase", mcallTask.Status.Phase)
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	log.Info("Status phase already set", "task", mcallTask.Name, "phase", mcallTask.Status.Phase)

	// Handle deletion
	if !mcallTask.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &mcallTask)
	}

	// Add finalizer if not present
	if !containsString(mcallTask.Finalizers, mcallv1.McallTaskFinalizer) {
		mcallTask.Finalizers = append(mcallTask.Finalizers, mcallv1.McallTaskFinalizer)
		if err := r.Update(ctx, &mcallTask); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile based on current status
	switch mcallTask.Status.Phase {
	case mcallv1.McallTaskPhasePending:
		return r.handlePending(ctx, &mcallTask)
	case mcallv1.McallTaskPhaseRunning:
		return r.handleRunning(ctx, &mcallTask)
	case mcallv1.McallTaskPhaseSucceeded, mcallv1.McallTaskPhaseFailed:
		return r.handleCompleted(ctx, &mcallTask)
	default:
		log.Info("Unknown phase", "phase", mcallTask.Status.Phase)
		return ctrl.Result{}, nil
	}
}

func (r *McallTaskReconciler) handlePending(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Check dependencies
	if len(task.Spec.Dependencies) > 0 {
		allDepsReady, err := r.checkDependencies(ctx, task)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !allDepsReady {
			log.Info("Dependencies not ready, skipping task", "task", task.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Check if task should be scheduled
	if task.Spec.Schedule != "" {
		shouldRun, err := r.shouldRunScheduledTask(ctx, task)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !shouldRun {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	}

	// Create execution pod
	if err := r.createExecutionPod(ctx, task); err != nil {
		return ctrl.Result{}, err
	}

	// Update status to Running
	task.Status.Phase = mcallv1.McallTaskPhaseRunning
	task.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, task); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *McallTaskReconciler) handleRunning(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	taskTimeout := getTaskTimeout()

	// Check if task has already been executed
	if task.Status.Phase == mcallv1.McallTaskPhaseSucceeded || task.Status.Phase == mcallv1.McallTaskPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Execute the actual task based on type
	var output string
	var errCode string
	var errMsg string
	var execErr error

	logger.Info("Executing task",
		"task", task.Name,
		"type", task.Spec.Type,
		"input", task.Spec.Input)

	switch task.Spec.Type {
	case "cmd":
		// Parse JSON inputs (like original mcall.go)
		jsonInputs, err := parseJSONInputs(task.Spec.Input)
		if err != nil {
			logger.Error(err, "Failed to parse JSON inputs", "task", task.Name)
			execErr = err
			output = fmt.Sprintf("Error parsing inputs: %v", err)
		} else {
			// Process inputs using mcall structure
			logger.Info("Processing inputs", "task", task.Name, "count", len(jsonInputs))

			var results []string
			var hasErrors bool

			// Create TaskWorkers for each input (like original mcall.go)
			workers := make([]*TaskWorker, 0, len(jsonInputs))
			for i, input := range jsonInputs {
				inputStr, ok := input["input"].(string)
				if !ok {
					continue
				}

				inputType := "cmd"
				if t, exists := input["type"].(string); exists {
					inputType = t
				}

				name := fmt.Sprintf("input-%d", i+1)
				if n, exists := input["name"].(string); exists {
					name = n
				}

				// Extract validation fields
				var expect string
				if e, exists := input["expect"]; exists {
					if eStr, ok := e.(string); ok {
						expect = eStr
					}
				}

				var expectedResponse map[string]interface{}
				if er, exists := input["expectedResponse"]; exists {
					if erMap, ok := er.(map[string]interface{}); ok {
						expectedResponse = erMap
						logger.Info("Found expectedResponse", "task", task.Name, "input", i+1, "expectedResponse", expectedResponse)
					} else {
						logger.Info("expectedResponse exists but not a map", "task", task.Name, "input", i+1, "type", fmt.Sprintf("%T", er), "value", er)
					}
				} else {
					logger.Info("No expectedResponse found", "task", task.Name, "input", i+1, "available_keys", getMapKeys(input))
				}

				// Create worker with expect validation (like mcall.go)
				worker := NewTaskWorker(inputStr, inputType, name, expect)
				workers = append(workers, worker)
			}

			// Determine execution mode (default to sequential for backward compatibility)
			executionMode := task.Spec.ExecutionMode
			if executionMode == "" {
				executionMode = mcallv1.ExecutionModeSequential
			}

			// Get failFast setting (default to false for backward compatibility)
			failFast := task.Spec.FailFast

			logger.Info("Executing workers", "task", task.Name, "mode", executionMode, "failFast", failFast, "count", len(workers))

			if executionMode == mcallv1.ExecutionModeParallel {
				// Parallel execution using goroutines
				results = executeWorkersParallel(workers, taskTimeout, logger, task.Name, failFast)
			} else {
				// Sequential execution (default behavior)
				results = executeWorkersSequential(workers, taskTimeout, logger, task.Name, failFast)
			}

			// Check for errors in results
			for i, result := range results {
				if strings.HasPrefix(result, "Error:") {
					hasErrors = true
					logger.Error(fmt.Errorf("input execution failed"), "Input execution failed",
						"task", task.Name,
						"input", i+1,
						"error", result)
				}
			}

			// Join results (like original mcall.go)
			output = strings.Join(results, "\n---\n")
			if hasErrors {
				execErr = fmt.Errorf("one or more inputs failed execution")
			} else {
				execErr = nil
			}
		}

	case "get":
		output, execErr = executeHTTPRequest(task.Spec.Input, "GET", taskTimeout)

	case "post":
		output, execErr = executeHTTPRequest(task.Spec.Input, "POST", taskTimeout)

	default:
		// Default to cmd execution
		output, execErr = executeCommand(task.Spec.Input, taskTimeout)
	}

	// Set result based on execution
	if execErr != nil {
		errCode = "-1"
		errMsg = execErr.Error()
		logger.Error(execErr, "Task execution failed", "task", task.Name)
	} else {
		errCode = "0"
		errMsg = ""
		logger.Info("Task execution completed successfully", "task", task.Name)
	}

	// Log to configured backend if logging is enabled
	loggingConfig := GetLoggingConfig()
	if loggingConfig.Enabled {
		logEntry := LogEntry{
			ServiceName: task.Name,
			ServiceType: task.Spec.Type,
			Status: func() string {
				if execErr != nil {
					return "DOWN"
				}
				return "UP"
			}(),
			Error: errMsg,
			ResponseTime: func() int64 {
				// Calculate response time (simplified - you might want to measure actual execution time)
				if execErr != nil {
					return 5000 // Default error response time
				}
				return 150 // Default success response time
			}(),
			Timestamp: time.Now(),
		}

		if err := LogToBackend(logEntry, loggingConfig); err != nil {
			logger.Error(err, "Failed to log to backend", "task", task.Name, "backend", loggingConfig.Backend)
		} else {
			logger.Info("Successfully logged to backend", "task", task.Name, "status", logEntry.Status, "backend", loggingConfig.Backend)
		}
	}

	// Update task status
	if execErr != nil {
		task.Status.Phase = mcallv1.McallTaskPhaseFailed
	} else {
		task.Status.Phase = mcallv1.McallTaskPhaseSucceeded
	}

	task.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	task.Status.Result = &mcallv1.McallTaskResult{
		Output:       output,
		ErrorCode:    errCode,
		ErrorMessage: errMsg,
	}

	if err := r.Status().Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update task status", "task", task.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Task status updated",
		"task", task.Name,
		"phase", task.Status.Phase,
		"errorCode", errCode)

	return ctrl.Result{}, nil
}

func (r *McallTaskReconciler) handleCompleted(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	// Clean up resources if needed
	return ctrl.Result{}, nil
}

func (r *McallTaskReconciler) handleDeletion(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling deletion", "task", task.Name)

	// Clean up associated pods
	if err := r.cleanupExecutionPods(ctx, task); err != nil {
		log.Error(err, "Failed to cleanup execution pods", "task", task.Name)
		return ctrl.Result{}, err
	}

	// Clean up any other resources (configmaps, secrets, etc.)
	if err := r.cleanupAssociatedResources(ctx, task); err != nil {
		log.Error(err, "Failed to cleanup associated resources", "task", task.Name)
		return ctrl.Result{}, err
	}

	// Remove finalizer to allow deletion
	task.Finalizers = removeString(task.Finalizers, mcallv1.McallTaskFinalizer)
	if err := r.Update(ctx, task); err != nil {
		log.Error(err, "Failed to remove finalizer", "task", task.Name)
		return ctrl.Result{}, err
	}

	log.Info("Successfully cleaned up and removed finalizer", "task", task.Name)
	return ctrl.Result{}, nil
}

func (r *McallTaskReconciler) checkDependencies(ctx context.Context, task *mcallv1.McallTask) (bool, error) {
	for _, depName := range task.Spec.Dependencies {
		var depTask mcallv1.McallTask
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: task.Namespace,
			Name:      depName,
		}, &depTask); err != nil {
			return false, err
		}
		if depTask.Status.Phase != mcallv1.McallTaskPhaseSucceeded {
			return false, nil
		}
	}
	return true, nil
}

func (r *McallTaskReconciler) shouldRunScheduledTask(ctx context.Context, task *mcallv1.McallTask) (bool, error) {
	// Implement cron schedule checking logic
	// For now, always return true
	return true, nil
}

func (r *McallTaskReconciler) createExecutionPod(ctx context.Context, task *mcallv1.McallTask) error {
	// Implementation to create execution pod would go here
	return nil
}

// cleanupExecutionPods removes all pods associated with this task
func (r *McallTaskReconciler) cleanupExecutionPods(ctx context.Context, task *mcallv1.McallTask) error {
	log := log.FromContext(ctx)

	// List all pods with the task label
	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(task.Namespace),
		client.MatchingLabels{"mcall.tz.io/task": task.Name}); err != nil {
		return err
	}

	// Delete each pod
	for _, pod := range pods.Items {
		log.Info("Deleting execution pod", "pod", pod.Name, "task", task.Name)
		if err := r.Delete(ctx, &pod); err != nil {
			log.Error(err, "Failed to delete pod", "pod", pod.Name, "task", task.Name)
			return err
		}
	}

	return nil
}

// cleanupAssociatedResources removes any other resources created by this task
func (r *McallTaskReconciler) cleanupAssociatedResources(ctx context.Context, task *mcallv1.McallTask) error {
	log := log.FromContext(ctx)

	// Clean up configmaps
	var configmaps corev1.ConfigMapList
	if err := r.List(ctx, &configmaps, client.InNamespace(task.Namespace),
		client.MatchingLabels{"mcall.tz.io/task": task.Name}); err != nil {
		return err
	}

	for _, cm := range configmaps.Items {
		log.Info("Deleting configmap", "configmap", cm.Name, "task", task.Name)
		if err := r.Delete(ctx, &cm); err != nil {
			log.Error(err, "Failed to delete configmap", "configmap", cm.Name, "task", task.Name)
			return err
		}
	}

	// Add cleanup for other resource types as needed (secrets, services, etc.)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *McallTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := log.FromContext(context.Background())
	
	// Add CRD availability check with retry
	go func() {
		maxRetries := 10
		retryInterval := 5 * time.Second
		
		for i := 0; i < maxRetries; i++ {
			ctx := context.Background()
			var mcallTasks mcallv1.McallTaskList
			if err := r.Client.List(ctx, &mcallTasks); err != nil {
				log.Error(err, "McallTask CRD not available, retrying...", "attempt", i+1, "maxRetries", maxRetries)
				time.Sleep(retryInterval)
				continue
			}
			
			var mcallWorkflows mcallv1.McallWorkflowList
			if err := r.Client.List(ctx, &mcallWorkflows); err != nil {
				log.Error(err, "McallWorkflow CRD not available, retrying...", "attempt", i+1, "maxRetries", maxRetries)
				time.Sleep(retryInterval)
				continue
			}
			
			log.Info("CRDs are now available", "mcallTasks", len(mcallTasks.Items), "mcallWorkflows", len(mcallWorkflows.Items))
			return
		}
		
		log.Error(nil, "CRDs not available after maximum retries", "maxRetries", maxRetries)
	}()
	
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcallv1.McallTask{}).
		Complete(r)
}

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// validateHttpResponse validates HTTP response based on validation rules
func (r *McallTaskReconciler) validateHttpResponse(response *http.Response, body string, validation *mcallv1.HttpValidation) error {
	if validation == nil {
		return nil
	}

	// Validate status codes
	if validation.ExpectedStatusCodes != nil && len(validation.ExpectedStatusCodes) > 0 {
		found := false
		for _, expectedCode := range validation.ExpectedStatusCodes {
			if response.StatusCode == expectedCode {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unexpected status code: %d, expected: %v", response.StatusCode, validation.ExpectedStatusCodes)
		}
	}

	// Validate response body
	if validation.ExpectedResponseBody != "" {
		switch validation.ResponseBodyMatch {
		case "exact":
			if body != validation.ExpectedResponseBody {
				return fmt.Errorf("response body mismatch: expected '%s', got '%s'", validation.ExpectedResponseBody, body)
			}
		case "contains":
			if !strings.Contains(body, validation.ExpectedResponseBody) {
				return fmt.Errorf("response body does not contain expected content: '%s'", validation.ExpectedResponseBody)
			}
		case "regex":
			if validation.ResponseBodyPattern != "" {
				matched, err := regexp.MatchString(validation.ResponseBodyPattern, body)
				if err != nil {
					return fmt.Errorf("regex pattern error: %v", err)
				}
				if !matched {
					return fmt.Errorf("response body does not match pattern: '%s'", validation.ResponseBodyPattern)
				}
			}
		case "json":
			// Basic JSON validation - could be enhanced with JSONPath
			if !strings.HasPrefix(strings.TrimSpace(body), "{") && !strings.HasPrefix(strings.TrimSpace(body), "[") {
				return fmt.Errorf("response body is not valid JSON")
			}
		}
	}

	// Validate response headers
	if validation.ResponseHeaders != nil {
		for key, expectedValue := range validation.ResponseHeaders {
			actualValue := response.Header.Get(key)
			if actualValue != expectedValue {
				return fmt.Errorf("header mismatch: %s expected '%s', got '%s'", key, expectedValue, actualValue)
			}
		}
	}

	return nil
}

// validateCommandOutput validates command output based on validation rules
func (r *McallTaskReconciler) validateCommandOutput(output string, validation *mcallv1.OutputValidation) error {
	if validation == nil {
		return nil
	}

	// Handle case sensitivity
	outputToCheck := output
	expectedOutput := validation.ExpectedOutput
	if !validation.CaseSensitive {
		outputToCheck = strings.ToLower(output)
		expectedOutput = strings.ToLower(validation.ExpectedOutput)
	}

	// Check success criteria
	if validation.SuccessCriteria != "" {
		if !r.checkOutputCriteria(outputToCheck, expectedOutput, validation.SuccessCriteria, validation) {
			return fmt.Errorf("output does not meet success criteria: %s", validation.SuccessCriteria)
		}
	}

	// Check failure criteria
	if validation.FailureCriteria != "" {
		if r.checkOutputCriteria(outputToCheck, expectedOutput, validation.FailureCriteria, validation) {
			return fmt.Errorf("output meets failure criteria: %s", validation.FailureCriteria)
		}
	}

	return nil
}

// checkOutputCriteria checks if output meets specific criteria
func (r *McallTaskReconciler) checkOutputCriteria(output, expectedOutput, criteria string, validation *mcallv1.OutputValidation) bool {
	switch criteria {
	case "contains":
		return strings.Contains(output, expectedOutput)
	case "exact":
		return output == expectedOutput
	case "regex":
		if validation.OutputPattern != "" {
			matched, err := regexp.MatchString(validation.OutputPattern, output)
			return err == nil && matched
		}
		return false
	case "empty":
		return strings.TrimSpace(output) == ""
	case "not_empty":
		return strings.TrimSpace(output) != ""
	case "json_match":
		return r.validateJsonOutput(output, validation)
	}
	return false
}

// validateJsonOutput validates JSON output using JSONPath
func (r *McallTaskReconciler) validateJsonOutput(output string, validation *mcallv1.OutputValidation) bool {
	// Basic JSON validation - could be enhanced with JSONPath library
	if validation.JsonPath != "" && validation.ExpectedJsonValue != "" {
		// For now, simple string matching in JSON
		// In production, use a proper JSONPath library
		return strings.Contains(output, validation.ExpectedJsonValue)
	}
	return false
}
