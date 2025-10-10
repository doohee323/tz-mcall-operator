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
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcallv1 "github.com/doohee323/tz-mcall-operator/api/v1"
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute command through bash shell to support:
	// - Redirections (>, >>, |)
	// - Variable substitution ($VAR, $(command))
	// - Multiple commands with && or ;
	// - Other shell features
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
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

	// Check HTTP status code - fail if not 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(doc), fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
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

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

// checkTaskCondition checks if a task should run based on its condition
func (r *McallTaskReconciler) checkTaskCondition(ctx context.Context, task *mcallv1.McallTask, condition *mcallv1.TaskCondition) (bool, error) {
	logger := log.FromContext(ctx)

	// Get the dependent task
	var depTask mcallv1.McallTask
	if err := r.Get(ctx, types.NamespacedName{
		Name:      condition.DependentTask,
		Namespace: task.Namespace,
	}, &depTask); err != nil {
		return false, fmt.Errorf("dependent task %s not found: %w", condition.DependentTask, err)
	}

	// Check if dependent task is completed
	if depTask.Status.Phase != mcallv1.McallTaskPhaseSucceeded &&
		depTask.Status.Phase != mcallv1.McallTaskPhaseFailed &&
		depTask.Status.Phase != mcallv1.McallTaskPhaseSkipped {
		logger.Info("Dependent task not completed yet",
			"task", task.Name,
			"dependentTask", condition.DependentTask,
			"dependentPhase", depTask.Status.Phase)
		return false, fmt.Errorf("dependent task %s not completed yet (phase: %s)", condition.DependentTask, depTask.Status.Phase)
	}

	// Check "when" condition
	switch condition.When {
	case "success":
		if depTask.Status.Phase != mcallv1.McallTaskPhaseSucceeded {
			logger.Info("Task condition not met: dependent task did not succeed",
				"task", task.Name,
				"dependentTask", condition.DependentTask,
				"dependentPhase", depTask.Status.Phase)
			return false, nil
		}
	case "failure":
		if depTask.Status.Phase != mcallv1.McallTaskPhaseFailed {
			logger.Info("Task condition not met: dependent task did not fail",
				"task", task.Name,
				"dependentTask", condition.DependentTask,
				"dependentPhase", depTask.Status.Phase)
			return false, nil
		}
	case "always", "completed":
		// Run regardless of dependent task result
		logger.Info("Task condition met: running after dependent task completion",
			"task", task.Name,
			"dependentTask", condition.DependentTask,
			"dependentPhase", depTask.Status.Phase)
	default:
		return false, fmt.Errorf("unknown condition.when value: %s", condition.When)
	}

	// Check FieldEquals condition
	if condition.FieldEquals != nil {
		var actualValue string
		switch condition.FieldEquals.Field {
		case "errorCode":
			if depTask.Status.Result != nil {
				actualValue = depTask.Status.Result.ErrorCode
			}
		case "phase":
			actualValue = string(depTask.Status.Phase)
		case "output":
			if depTask.Status.Result != nil {
				actualValue = depTask.Status.Result.Output
			}
		default:
			return false, fmt.Errorf("unknown field for condition: %s", condition.FieldEquals.Field)
		}

		if actualValue != condition.FieldEquals.Value {
			logger.Info("Task condition not met: field value mismatch",
				"task", task.Name,
				"field", condition.FieldEquals.Field,
				"expected", condition.FieldEquals.Value,
				"actual", truncateString(actualValue, 100))
			return false, nil
		}
	}

	// Check OutputContains condition
	if condition.OutputContains != "" {
		if depTask.Status.Result == nil ||
			!strings.Contains(depTask.Status.Result.Output, condition.OutputContains) {
			logger.Info("Task condition not met: output does not contain expected string",
				"task", task.Name,
				"expected", condition.OutputContains,
				"output", truncateString(depTask.Status.Result.Output, 100))
			return false, nil
		}
	}

	logger.Info("Task condition met, proceeding with execution",
		"task", task.Name,
		"dependentTask", condition.DependentTask)
	return true, nil
}

func (r *McallTaskReconciler) handlePending(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Check condition if present (from workflow annotation)
	if conditionStr, exists := task.Annotations["mcall.tz.io/condition"]; exists && conditionStr != "" {
		var condition mcallv1.TaskCondition
		if err := json.Unmarshal([]byte(conditionStr), &condition); err != nil {
			log.Error(err, "Failed to parse task condition", "task", task.Name)
			return ctrl.Result{}, err
		}

		shouldRun, err := r.checkTaskCondition(ctx, task, &condition)
		if err != nil {
			// If dependent task not completed yet, requeue
			if strings.Contains(err.Error(), "not completed yet") {
				log.Info("Waiting for dependent task to complete",
					"task", task.Name,
					"dependentTask", condition.DependentTask)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
			log.Error(err, "Failed to check task condition", "task", task.Name)
			return ctrl.Result{}, err
		}

		if !shouldRun {
			log.Info("Task condition not met, skipping", "task", task.Name, "condition", condition)
			task.Status.Phase = mcallv1.McallTaskPhaseSkipped
			task.Status.CompletionTime = &metav1.Time{Time: time.Now()}
			task.Status.Result = &mcallv1.McallTaskResult{
				ErrorCode:    "0",
				ErrorMessage: fmt.Sprintf("Skipped due to condition: when=%s", condition.When),
			}
			if err := r.Status().Update(ctx, task); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

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
		
		// Update with retry on conflict
		updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Get the latest version
			latest := &mcallv1.McallTask{}
			if err := r.Get(ctx, types.NamespacedName{
				Name:      task.Name,
				Namespace: task.Namespace,
			}, latest); err != nil {
				return err
			}
			
			// Apply changes to latest version
			latest.Status.Phase = mcallv1.McallTaskPhaseRunning
			latest.Status.StartTime = &metav1.Time{Time: time.Now()}
			
			return r.Status().Update(ctx, latest)
		})
		
		if updateErr != nil {
			log.Error(updateErr, "Failed to update task status after retries", "task", task.Name)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

	return ctrl.Result{}, nil
}

// processInputSources processes InputSources and returns processed input and environment variables
func (r *McallTaskReconciler) processInputSources(ctx context.Context, task *mcallv1.McallTask) (string, map[string]string, error) {
	logger := log.FromContext(ctx)
	inputData := make(map[string]interface{})
	envVars := make(map[string]string)

	for _, source := range task.Spec.InputSources {
		// Get referenced task
		var refTask mcallv1.McallTask
		err := r.Get(ctx, types.NamespacedName{
			Name:      source.TaskRef,
			Namespace: task.Namespace,
		}, &refTask)

		if err != nil {
			// Task not found - use default if available
			if source.Default != "" {
				logger.Info("Referenced task not found, using default value",
					"task", task.Name,
					"sourceTask", source.TaskRef,
					"defaultValue", truncateString(source.Default, 100))
				inputData[source.Name] = source.Default
				envVars[source.Name] = source.Default
				continue
			}
			return "", nil, fmt.Errorf("referenced task %s not found and no default value: %w", source.TaskRef, err)
		}

		// Check if referenced task is completed
		if refTask.Status.Phase != mcallv1.McallTaskPhaseSucceeded &&
			refTask.Status.Phase != mcallv1.McallTaskPhaseFailed {
			logger.Info("Referenced task not completed yet, waiting...",
				"task", task.Name,
				"sourceTask", source.TaskRef,
				"sourcePhase", refTask.Status.Phase)
			return "", nil, fmt.Errorf("referenced task %s not completed yet (phase: %s)", source.TaskRef, refTask.Status.Phase)
		}

		// Extract value based on field
		var value string
		switch source.Field {
		case "output":
			if refTask.Status.Result != nil {
				value = refTask.Status.Result.Output

				// Apply JSONPath if specified
				if source.JSONPath != "" {
					extracted, err := extractJSONPath(value, source.JSONPath)
					if err != nil {
						logger.Error(err, "Failed to extract JSONPath",
							"task", task.Name,
							"sourceTask", source.TaskRef,
							"jsonPath", source.JSONPath)
						if source.Default != "" {
							value = source.Default
						} else {
							return "", nil, fmt.Errorf("failed to extract JSONPath %s: %w", source.JSONPath, err)
						}
					} else {
						value = extracted
					}
				}
			}
		case "errorCode":
			if refTask.Status.Result != nil {
				value = refTask.Status.Result.ErrorCode
			}
		case "phase":
			value = string(refTask.Status.Phase)
		case "errorMessage":
			if refTask.Status.Result != nil {
				value = refTask.Status.Result.ErrorMessage
			}
		case "startTime":
			if refTask.Status.StartTime != nil {
				value = refTask.Status.StartTime.Format(time.RFC3339)
			}
		case "completionTime":
			if refTask.Status.CompletionTime != nil {
				value = refTask.Status.CompletionTime.Format(time.RFC3339)
			}
		case "all":
			// All information as JSON
			allData := map[string]interface{}{
				"phase": string(refTask.Status.Phase),
			}
			if refTask.Status.StartTime != nil {
				allData["startTime"] = refTask.Status.StartTime.Format(time.RFC3339)
			}
			if refTask.Status.CompletionTime != nil {
				allData["completionTime"] = refTask.Status.CompletionTime.Format(time.RFC3339)
			}
			if refTask.Status.Result != nil {
				allData["output"] = refTask.Status.Result.Output
				allData["errorCode"] = refTask.Status.Result.ErrorCode
				allData["errorMessage"] = refTask.Status.Result.ErrorMessage
			}
			jsonBytes, _ := json.Marshal(allData)
			value = string(jsonBytes)
		default:
			return "", nil, fmt.Errorf("unknown field: %s", source.Field)
		}

		inputData[source.Name] = value
		envVars[source.Name] = value

		logger.Info("Extracted data from source task",
			"task", task.Name,
			"sourceTask", source.TaskRef,
			"field", source.Field,
			"varName", source.Name,
			"valuePreview", truncateString(value, 100))
	}

	// Render input template if specified
	if task.Spec.InputTemplate != "" {
		renderedInput := renderTemplate(task.Spec.InputTemplate, inputData)
		logger.Info("Rendered input template",
			"task", task.Name,
			"template", truncateString(task.Spec.InputTemplate, 100),
			"rendered", truncateString(renderedInput, 100))
		return renderedInput, envVars, nil
	}

	// If no template, return JSON of all input data
	jsonBytes, err := json.Marshal(inputData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	return string(jsonBytes), envVars, nil
}

func (r *McallTaskReconciler) handleRunning(ctx context.Context, task *mcallv1.McallTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	taskTimeout := getTaskTimeout()

	// Check if task has already been executed
	if task.Status.Phase == mcallv1.McallTaskPhaseSucceeded || task.Status.Phase == mcallv1.McallTaskPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Process InputSources if present
	if len(task.Spec.InputSources) > 0 {
		processedInput, envVars, err := r.processInputSources(ctx, task)
		if err != nil {
			// Check if it's a "not ready" error (referenced task not completed)
			if strings.Contains(err.Error(), "not completed yet") {
				logger.Info("Waiting for input sources to be ready",
					"task", task.Name,
					"error", err.Error())
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			// Other errors are failures
			logger.Error(err, "Failed to process input sources", "task", task.Name)
			task.Status.Phase = mcallv1.McallTaskPhaseFailed
			task.Status.CompletionTime = &metav1.Time{Time: time.Now()}
			task.Status.Result = &mcallv1.McallTaskResult{
				ErrorCode:    "-1",
				ErrorMessage: fmt.Sprintf("Failed to process input sources: %v", err),
			}
			
			// Update with retry on conflict
			updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Get the latest version
				latest := &mcallv1.McallTask{}
				if err := r.Get(ctx, types.NamespacedName{
					Name:      task.Name,
					Namespace: task.Namespace,
				}, latest); err != nil {
					return err
				}
				
				// Apply changes to latest version
				latest.Status.Result = &mcallv1.McallTaskResult{
					ErrorCode:    "-1",
					ErrorMessage: fmt.Sprintf("Failed to process input sources: %v", err),
				}
				
				return r.Status().Update(ctx, latest)
			})
			
			if updateErr != nil {
				logger.Error(updateErr, "Failed to update task status after retries", "task", task.Name)
			}
			return ctrl.Result{}, err
		}

		// Use processed input if InputTemplate was specified
		if task.Spec.InputTemplate != "" {
			task.Spec.Input = processedInput
			logger.Info("Using processed input from template",
				"task", task.Name,
				"input", truncateString(processedInput, 200))
		}

		// Merge environment variables
		if task.Spec.Environment == nil {
			task.Spec.Environment = make(map[string]string)
		}
		for k, v := range envVars {
			task.Spec.Environment[k] = v
		}

		logger.Info("Injected data from input sources",
			"task", task.Name,
			"sourceCount", len(task.Spec.InputSources),
			"envVars", len(envVars))
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

	// Update with retry on conflict
	updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get the latest version
		latest := &mcallv1.McallTask{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      task.Name,
			Namespace: task.Namespace,
		}, latest); err != nil {
			return err
		}
		
		// Apply changes to latest version
		latest.Status.Phase = task.Status.Phase
		latest.Status.CompletionTime = task.Status.CompletionTime
		latest.Status.Result = &mcallv1.McallTaskResult{
			Output:       output,
			ErrorCode:    errCode,
			ErrorMessage: errMsg,
		}
		
		return r.Status().Update(ctx, latest)
	})

	if updateErr != nil {
		logger.Error(updateErr, "Failed to update task status after retries", "task", task.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
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

// renderTemplate replaces ${VAR_NAME} with actual values
func renderTemplate(template string, data map[string]interface{}) string {
	result := template

	for key, value := range data {
		placeholder := fmt.Sprintf("${%s}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result
}

// extractJSONPath extracts value from JSON string using simple path expression
// Supports simple paths like: $.field, $.nested.field
// For complex JSONPath, consider using github.com/oliveagle/jsonpath library
func extractJSONPath(jsonStr string, path string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Remove leading "$."
	path = strings.TrimPrefix(path, "$.")
	if path == "$" || path == "" {
		// Return whole JSON
		jsonBytes, _ := json.Marshal(data)
		return string(jsonBytes), nil
	}

	// Split path by "."
	fields := strings.Split(path, ".")
	current := data

	for _, field := range fields {
		// Check for array index like "items[0]"
		if strings.Contains(field, "[") {
			re := regexp.MustCompile(`^([^\[]+)\[(\d+)\]$`)
			matches := re.FindStringSubmatch(field)
			if len(matches) == 3 {
				fieldName := matches[1]
				index, _ := strconv.Atoi(matches[2])

				// Get field
				if m, ok := current.(map[string]interface{}); ok {
					current = m[fieldName]
				} else {
					return "", fmt.Errorf("field %s not found", fieldName)
				}

				// Get array element
				if arr, ok := current.([]interface{}); ok {
					if index < len(arr) {
						current = arr[index]
					} else {
						return "", fmt.Errorf("array index %d out of bounds", index)
					}
				} else {
					return "", fmt.Errorf("field %s is not an array", fieldName)
				}
				continue
			}
		}

		// Regular field access
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[field]; exists {
				current = val
			} else {
				return "", fmt.Errorf("field %s not found in JSON", field)
			}
		} else {
			return "", fmt.Errorf("cannot access field %s in non-object", field)
		}
	}

	// Convert result to string
	switch v := current.(type) {
	case string:
		return v, nil
	case float64, int, int64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		// Object or array - return as JSON
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes), nil
	}
}

// truncateString truncates a string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// maskSensitiveData masks potentially sensitive information
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)["\s:=]+([^\s"]+)`),
	regexp.MustCompile(`(?i)(api[-_]?key|apikey)["\s:=]+([^\s"]+)`),
	regexp.MustCompile(`(?i)(token|secret)["\s:=]+([^\s"]+)`),
}

func maskSensitiveData(output string) string {
	masked := output
	for _, pattern := range sensitivePatterns {
		masked = pattern.ReplaceAllString(masked, "$1=***MASKED***")
	}
	return masked
}
