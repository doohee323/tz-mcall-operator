package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcallv1 "github.com/USERNAME/tz-mcall-crd/api/v1"
)

// CronScheduler handles cron schedule parsing and execution
type CronScheduler struct {
	client.Client
}

// NewCronScheduler creates a new CronScheduler instance
func NewCronScheduler(client client.Client) *CronScheduler {
	return &CronScheduler{
		Client: client,
	}
}

// CronExpression represents a parsed cron expression
type CronExpression struct {
	Minute  string
	Hour    string
	Day     string
	Month   string
	Weekday string
	LastRun time.Time
	NextRun time.Time
}

// ParseCronExpression parses a cron expression
func (cs *CronScheduler) ParseCronExpression(expr string) (*CronExpression, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(parts))
	}

	return &CronExpression{
		Minute:  parts[0],
		Hour:    parts[1],
		Day:     parts[2],
		Month:   parts[3],
		Weekday: parts[4],
	}, nil
}

// ShouldRun checks if a workflow should run based on its cron schedule
func (cs *CronScheduler) ShouldRun(ctx context.Context, workflow *mcallv1.McallWorkflow) (bool, error) {
	log := log.FromContext(ctx)

	if workflow.Spec.Schedule == "" {
		// No schedule, run immediately
		return true, nil
	}

	// Parse cron expression
	cron, err := cs.ParseCronExpression(workflow.Spec.Schedule)
	if err != nil {
		log.Error(err, "Failed to parse cron expression", "workflow", workflow.Name, "schedule", workflow.Spec.Schedule)
		return false, err
	}

	// Check if this is the first run
	if workflow.Status.LastRunTime == nil {
		log.Info("First run of scheduled workflow", "workflow", workflow.Name)
		return true, nil
	}

	// Calculate next run time
	now := time.Now()
	nextRun, err := cs.calculateNextRun(cron, workflow.Status.LastRunTime.Time)
	if err != nil {
		log.Error(err, "Failed to calculate next run time", "workflow", workflow.Name)
		return false, err
	}

	// Check if it's time to run
	shouldRun := now.After(nextRun) || now.Equal(nextRun)

	if shouldRun {
		log.Info("Workflow scheduled to run", "workflow", workflow.Name, "nextRun", nextRun, "now", now)
	}

	return shouldRun, nil
}

// calculateNextRun calculates the next run time based on cron expression
func (cs *CronScheduler) calculateNextRun(cron *CronExpression, lastRun time.Time) (time.Time, error) {
	now := time.Now()

	// Start from the last run time or now, whichever is later
	start := lastRun
	if now.After(lastRun) {
		start = now
	}

	// Add 1 minute to start checking from the next minute
	start = start.Add(1 * time.Minute).Truncate(time.Minute)

	// Check the next 24 hours for a match
	for i := 0; i < 24*60; i++ {
		checkTime := start.Add(time.Duration(i) * time.Minute)

		if cs.matchesCron(cron, checkTime) {
			return checkTime, nil
		}
	}

	// If no match found in 24 hours, return next day at the same time
	return start.Add(24 * time.Hour), nil
}

// matchesCron checks if a time matches the cron expression
func (cs *CronScheduler) matchesCron(cron *CronExpression, t time.Time) bool {
	// Check minute
	if !cs.matchesField(cron.Minute, t.Minute()) {
		return false
	}

	// Check hour
	if !cs.matchesField(cron.Hour, t.Hour()) {
		return false
	}

	// Check day of month
	if !cs.matchesField(cron.Day, t.Day()) {
		return false
	}

	// Check month
	if !cs.matchesField(cron.Month, int(t.Month())) {
		return false
	}

	// Check weekday (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	if !cs.matchesField(cron.Weekday, int(t.Weekday())) {
		return false
	}

	return true
}

// matchesField checks if a value matches a cron field
func (cs *CronScheduler) matchesField(field string, value int) bool {
	// Handle wildcard
	if field == "*" {
		return true
	}

	// Handle comma-separated values
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			if cs.matchesField(strings.TrimSpace(part), value) {
				return true
			}
		}
		return false
	}

	// Handle range (e.g., "1-5")
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return false
		}
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return false
		}
		return value >= start && value <= end
	}

	// Handle step (e.g., "*/5")
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil {
			return false
		}
		return value%step == 0
	}

	// Handle exact value
	exact, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return value == exact
}

// UpdateLastRunTime updates the last run time for a workflow
func (cs *CronScheduler) UpdateLastRunTime(ctx context.Context, workflow *mcallv1.McallWorkflow) error {
	now := metav1.Time{Time: time.Now()}
	workflow.Status.LastRunTime = &now

	return cs.Status().Update(ctx, workflow)
}
