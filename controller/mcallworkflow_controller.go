package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcallv1 "github.com/doohee323/tz-mcall-operator/api/v1"
)

// McallWorkflowReconciler reconciles a McallWorkflow object
type McallWorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcallworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcallworkflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcallworkflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcalltasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mcall.tz.io,resources=mcalltasks/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *McallWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("=== WORKFLOW RECONCILE START ===", "workflow", req.NamespacedName)

	// Fetch the McallWorkflow instance
	var mcallWorkflow mcallv1.McallWorkflow
	if err := r.Get(ctx, req.NamespacedName, &mcallWorkflow); err != nil {
		log.Error(err, "unable to fetch McallWorkflow")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Fetched McallWorkflow", "workflow", mcallWorkflow.Name, "currentPhase", mcallWorkflow.Status.Phase)

	// Initialize status if not set
	if len(mcallWorkflow.Status.Phase) == 0 {
		log.Info("*** WORKFLOW STATUS PHASE IS EMPTY - INITIALIZING TO PENDING ***", "workflow", mcallWorkflow.Name)
		mcallWorkflow.Status.Phase = mcallv1.McallWorkflowPhasePending
		if err := r.Status().Update(ctx, &mcallWorkflow); err != nil {
			log.Error(err, "*** FAILED TO INITIALIZE WORKFLOW STATUS PHASE ***", "workflow", mcallWorkflow.Name, "error", err.Error())
			return ctrl.Result{}, err
		}
		log.Info("*** SUCCESSFULLY INITIALIZED WORKFLOW STATUS PHASE ***", "workflow", mcallWorkflow.Name, "phase", mcallWorkflow.Status.Phase)
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// Handle different phases
	switch mcallWorkflow.Status.Phase {
	case mcallv1.McallWorkflowPhasePending:
		return r.handleWorkflowPending(ctx, &mcallWorkflow)
	case mcallv1.McallWorkflowPhaseRunning:
		return r.handleWorkflowRunning(ctx, &mcallWorkflow)
	case mcallv1.McallWorkflowPhaseSucceeded, mcallv1.McallWorkflowPhaseFailed:
		return r.handleWorkflowCompleted(ctx, &mcallWorkflow)
	}

	return ctrl.Result{}, nil
}

func (r *McallWorkflowReconciler) handleWorkflowPending(ctx context.Context, workflow *mcallv1.McallWorkflow) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Check if workflow should be scheduled
	if workflow.Spec.Schedule != "" {
		shouldRun, err := r.shouldRunScheduledWorkflow(ctx, workflow)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !shouldRun {
			log.Info("Workflow not scheduled to run yet", "workflow", workflow.Name)
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	}

	// Create McallTask resources for each task in the workflow
	if err := r.createWorkflowTasks(ctx, workflow); err != nil {
		return ctrl.Result{}, err
	}

	// Update status to Running
	workflow.Status.Phase = mcallv1.McallWorkflowPhaseRunning
	workflow.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, workflow); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *McallWorkflowReconciler) handleWorkflowRunning(ctx context.Context, workflow *mcallv1.McallWorkflow) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Build/Update DAG for UI visualization
	if err := r.buildWorkflowDAG(ctx, workflow); err != nil {
		log.Error(err, "Failed to build workflow DAG", "workflow", workflow.Name)
		// Continue even if DAG build fails
	}

	// Check status of all tasks in the workflow
	allTasksCompleted, hasFailedTasks, err := r.checkWorkflowTasksStatus(ctx, workflow)
	if err != nil {
		return ctrl.Result{}, err
	}

	if allTasksCompleted {
		if hasFailedTasks {
			workflow.Status.Phase = mcallv1.McallWorkflowPhaseFailed
		} else {
			workflow.Status.Phase = mcallv1.McallWorkflowPhaseSucceeded
		}
		workflow.Status.CompletionTime = &metav1.Time{Time: time.Now()}
		
		// Build final DAG state
		if err := r.buildWorkflowDAG(ctx, workflow); err != nil {
			log.Error(err, "Failed to build final workflow DAG", "workflow", workflow.Name)
		}
		
		if err := r.Status().Update(ctx, workflow); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Workflow completed", "workflow", workflow.Name, "phase", workflow.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Update status with DAG
	if err := r.Status().Update(ctx, workflow); err != nil {
		return ctrl.Result{}, err
	}

	// Continue monitoring
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *McallWorkflowReconciler) handleWorkflowCompleted(ctx context.Context, workflow *mcallv1.McallWorkflow) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// For scheduled workflows, clean up completed tasks and reset to Pending for next run
	if workflow.Spec.Schedule != "" {
		log.Info("Cleaning up completed scheduled workflow", "workflow", workflow.Name, "phase", workflow.Status.Phase)

		// Delete workflow-specific task instances (not template tasks)
		if err := r.deleteWorkflowTasks(ctx, workflow); err != nil {
			log.Error(err, "Failed to delete workflow tasks", "workflow", workflow.Name)
			return ctrl.Result{}, err
		}

		// Reset workflow status to Pending for next scheduled run
		workflow.Status.Phase = mcallv1.McallWorkflowPhasePending
		workflow.Status.StartTime = nil
		workflow.Status.CompletionTime = nil
		if err := r.Status().Update(ctx, workflow); err != nil {
			log.Error(err, "Failed to reset workflow status", "workflow", workflow.Name)
			return ctrl.Result{}, err
		}

		log.Info("Workflow reset to Pending for next scheduled run", "workflow", workflow.Name)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// For non-scheduled workflows, just clean up
	return ctrl.Result{}, nil
}

func (r *McallWorkflowReconciler) shouldRunScheduledWorkflow(ctx context.Context, workflow *mcallv1.McallWorkflow) (bool, error) {
	scheduler := NewCronScheduler(r.Client)
	return scheduler.ShouldRun(ctx, workflow)
}

func (r *McallWorkflowReconciler) createWorkflowTasks(ctx context.Context, workflow *mcallv1.McallWorkflow) error {
	log := log.FromContext(ctx)

	// Create tasks in dependency order
	tasksToCreate := r.sortTasksByDependencies(workflow.Spec.Tasks)

	for _, taskSpec := range tasksToCreate {
		// Get the referenced McallTask
		taskRef := taskSpec.TaskRef
		if taskRef.Namespace == "" {
			taskRef.Namespace = workflow.Namespace
		}

		var referencedTask mcallv1.McallTask
		if err := r.Get(ctx, types.NamespacedName{
			Name:      taskRef.Name,
			Namespace: taskRef.Namespace,
		}, &referencedTask); err != nil {
			log.Error(err, "Failed to get referenced task", "workflow", workflow.Name, "task", taskSpec.Name, "taskRef", taskRef)
			return err
		}

		// Create a copy of the referenced task with workflow-specific settings
		task := &mcallv1.McallTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", workflow.Name, taskSpec.Name),
				Namespace: workflow.Namespace,
				Labels: map[string]string{
					"mcall.tz.io/workflow":      workflow.Name,
					"mcall.tz.io/task":          taskSpec.Name,
					"mcall.tz.io/original-task": taskRef.Name,
				},
				Annotations: make(map[string]string),
			},
			Spec: *referencedTask.Spec.DeepCopy(),
		}

		// Update dependencies to use workflow task names
		task.Spec.Dependencies = r.convertDependencies(workflow.Name, taskSpec.Dependencies)

		// Set Condition annotation if specified
		if taskSpec.Condition != nil {
			// Update DependentTask to use workflow task name
			condition := *taskSpec.Condition
			condition.DependentTask = fmt.Sprintf("%s-%s", workflow.Name, condition.DependentTask)

			conditionJSON, err := json.Marshal(condition)
			if err != nil {
				log.Error(err, "Failed to marshal task condition", "workflow", workflow.Name, "task", taskSpec.Name)
				return err
			}
			task.Annotations["mcall.tz.io/condition"] = string(conditionJSON)

			log.Info("Set task condition",
				"workflow", workflow.Name,
				"task", taskSpec.Name,
				"condition", condition)
		}

		// Set InputSources if specified
		if len(taskSpec.InputSources) > 0 {
			// Convert task references to use workflow task names
			inputSources := make([]mcallv1.TaskInputSource, len(taskSpec.InputSources))
			for i, source := range taskSpec.InputSources {
				inputSources[i] = source
				// Convert TaskRef to workflow task name
				inputSources[i].TaskRef = fmt.Sprintf("%s-%s", workflow.Name, source.TaskRef)
			}
			task.Spec.InputSources = inputSources

			log.Info("Set task input sources",
				"workflow", workflow.Name,
				"task", taskSpec.Name,
				"sourceCount", len(inputSources))
		}

		// Set InputTemplate if specified
		if taskSpec.InputTemplate != "" {
			task.Spec.InputTemplate = taskSpec.InputTemplate

			log.Info("Set task input template",
				"workflow", workflow.Name,
				"task", taskSpec.Name,
				"template", taskSpec.InputTemplate)
		}

		if err := r.Create(ctx, task); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Task already exists, delete and recreate with updated specs
				log.Info("Task already exists, deleting and recreating", "workflow", workflow.Name, "task", taskSpec.Name)

				existingTask := &mcallv1.McallTask{}
				if getErr := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, existingTask); getErr != nil {
					log.Error(getErr, "Failed to get existing task", "workflow", workflow.Name, "task", taskSpec.Name)
					return getErr
				}

				// Check if task is already being deleted
				if existingTask.DeletionTimestamp != nil {
					log.Info("Task is already being deleted, waiting for deletion to complete", "workflow", workflow.Name, "task", taskSpec.Name)
					// Don't delete again, just wait for it to be removed
				} else {
					// Delete the existing task
					if delErr := r.Delete(ctx, existingTask); delErr != nil && !apierrors.IsNotFound(delErr) {
						log.Error(delErr, "Failed to delete existing task", "workflow", workflow.Name, "task", taskSpec.Name)
						return delErr
					}
				}

				// Wait for task to be fully deleted
				log.Info("Waiting for task deletion to complete", "workflow", workflow.Name, "task", taskSpec.Name)
				timeout := time.After(30 * time.Second)
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()

				taskDeleted := false
				for !taskDeleted {
					select {
					case <-timeout:
						log.Error(fmt.Errorf("timeout"), "Timeout waiting for task deletion", "workflow", workflow.Name, "task", taskSpec.Name)
						return fmt.Errorf("timeout waiting for task %s deletion", taskSpec.Name)
					case <-ticker.C:
						checkTask := &mcallv1.McallTask{}
						if getErr := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, checkTask); getErr != nil {
							if apierrors.IsNotFound(getErr) {
								// Task is fully deleted
								taskDeleted = true
								log.Info("Task deletion completed", "workflow", workflow.Name, "task", taskSpec.Name)
							} else {
								log.Error(getErr, "Error checking task deletion status", "workflow", workflow.Name, "task", taskSpec.Name)
								return getErr
							}
						}
						// Task still exists, continue waiting
					}
				}

				// Now recreate task
				if createErr := r.Create(ctx, task); createErr != nil {
					log.Error(createErr, "Failed to recreate task", "workflow", workflow.Name, "task", taskSpec.Name)
					return createErr
				}

				log.Info("Recreated task for workflow", "workflow", workflow.Name, "task", taskSpec.Name)
			} else {
				log.Error(err, "Failed to create task", "workflow", workflow.Name, "task", taskSpec.Name)
				return err
			}
		} else {
			log.Info("Created task for workflow", "workflow", workflow.Name, "task", taskSpec.Name, "originalTask", taskRef.Name, "dependencies", taskSpec.Dependencies)
		}
	}

	return nil
}

func (r *McallWorkflowReconciler) deleteWorkflowTasks(ctx context.Context, workflow *mcallv1.McallWorkflow) error {
	log := log.FromContext(ctx)

	// Delete all workflow-specific task instances using labels
	// Template tasks should NOT have the workflow label
	var tasks mcallv1.McallTaskList
	if err := r.List(ctx, &tasks,
		client.InNamespace(workflow.Namespace),
		client.MatchingLabels{"mcall.tz.io/workflow": workflow.Name}); err != nil {
		log.Error(err, "Failed to list workflow tasks for deletion", "workflow", workflow.Name)
		return err
	}

	tasksToDelete := []string{}
	for _, task := range tasks.Items {
		// Skip template tasks (they have -template suffix)
		if strings.HasSuffix(task.Name, "-template") {
			continue
		}

		if err := r.Delete(ctx, &task); err != nil {
			log.Error(err, "Failed to delete workflow task", "workflow", workflow.Name, "task", task.Name)
			return err
		}

		tasksToDelete = append(tasksToDelete, task.Name)
		log.Info("Deleted workflow task", "workflow", workflow.Name, "task", task.Name)
	}

	// Wait for all tasks to be fully deleted
	if len(tasksToDelete) > 0 {
		log.Info("Waiting for tasks to be fully deleted", "workflow", workflow.Name, "count", len(tasksToDelete))
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				log.Info("Timeout waiting for task deletion, continuing anyway", "workflow", workflow.Name)
				return nil
			case <-ticker.C:
				allDeleted := true
				for _, taskName := range tasksToDelete {
					var task mcallv1.McallTask
					err := r.Get(ctx, types.NamespacedName{
						Name:      taskName,
						Namespace: workflow.Namespace,
					}, &task)
					if err == nil {
						// Task still exists
						allDeleted = false
						break
					}
				}
				if allDeleted {
					log.Info("All tasks fully deleted", "workflow", workflow.Name)
					return nil
				}
			}
		}
	}

	log.Info("Cleaned up workflow tasks", "workflow", workflow.Name, "deletedCount", len(tasks.Items))
	return nil
}

// sortTasksByDependencies sorts tasks by their dependencies (topological sort)
func (r *McallWorkflowReconciler) sortTasksByDependencies(tasks []mcallv1.WorkflowTaskRef) []mcallv1.WorkflowTaskRef {
	// Create a map of task names to tasks
	taskMap := make(map[string]mcallv1.WorkflowTaskRef)
	for _, task := range tasks {
		taskMap[task.Name] = task
	}

	// Track visited tasks and their dependencies
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []mcallv1.WorkflowTaskRef

	// DFS to sort tasks by dependencies
	var visit func(string)
	visit = func(taskName string) {
		if visiting[taskName] {
			// Circular dependency detected
			return
		}
		if visited[taskName] {
			return
		}

		visiting[taskName] = true
		task := taskMap[taskName]

		// Visit dependencies first
		for _, dep := range task.Dependencies {
			visit(dep)
		}

		visiting[taskName] = false
		visited[taskName] = true
		result = append(result, task)
	}

	// Visit all tasks
	for _, task := range tasks {
		visit(task.Name)
	}

	return result
}

// convertDependencies converts workflow task dependencies to McallTask dependencies
func (r *McallWorkflowReconciler) convertDependencies(workflowName string, dependencies []string) []string {
	var converted []string
	for _, dep := range dependencies {
		converted = append(converted, fmt.Sprintf("%s-%s", workflowName, dep))
	}
	return converted
}

func (r *McallWorkflowReconciler) checkWorkflowTasksStatus(ctx context.Context, workflow *mcallv1.McallWorkflow) (bool, bool, error) {
	log := log.FromContext(ctx)

	// Get all tasks for this workflow
	var tasks mcallv1.McallTaskList
	if err := r.List(ctx, &tasks, client.InNamespace(workflow.Namespace), client.MatchingLabels{"mcall.tz.io/workflow": workflow.Name}); err != nil {
		return false, false, err
	}

	allCompleted := true
	hasFailed := false

	for _, task := range tasks.Items {
		switch task.Status.Phase {
		case mcallv1.McallTaskPhasePending, mcallv1.McallTaskPhaseRunning:
			allCompleted = false
		case mcallv1.McallTaskPhaseFailed:
			hasFailed = true
		case mcallv1.McallTaskPhaseSucceeded:
			// Task completed successfully
		}
	}

	log.Info("Workflow tasks status", "workflow", workflow.Name, "totalTasks", len(tasks.Items), "allCompleted", allCompleted, "hasFailed", hasFailed)

	return allCompleted, hasFailed, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *McallWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcallv1.McallWorkflow{}).
		Complete(r)
}

// buildWorkflowDAG builds the DAG representation of the workflow for UI visualization
func (r *McallWorkflowReconciler) buildWorkflowDAG(ctx context.Context, workflow *mcallv1.McallWorkflow) error {
	log := log.FromContext(ctx)

	dag := &mcallv1.WorkflowDAG{
		Nodes:  []mcallv1.DAGNode{},
		Edges:  []mcallv1.DAGEdge{},
		Layout: "dagre",
		Metadata: mcallv1.DAGMetadata{
			TotalNodes: len(workflow.Spec.Tasks),
		},
	}

	// Build nodes from tasks
	nodePositions := r.calculateNodePositions(len(workflow.Spec.Tasks))
	
	for idx, taskSpec := range workflow.Spec.Tasks {
		taskName := fmt.Sprintf("%s-%s", workflow.Name, taskSpec.Name)

		// Get actual task status from Kubernetes
		var task mcallv1.McallTask
		taskExists := true
		if err := r.Get(ctx, types.NamespacedName{
			Name:      taskName,
			Namespace: workflow.Namespace,
		}, &task); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Failed to get task for DAG", "task", taskName)
			}
			taskExists = false
		}

		// Create node
		node := mcallv1.DAGNode{
			ID:       taskSpec.Name,
			Name:     taskSpec.Name,
			Type:     "cmd", // default
			Phase:    mcallv1.McallTaskPhasePending,
			TaskRef:  taskSpec.TaskRef.Name,
			Position: &mcallv1.NodePosition{X: nodePositions[idx].X, Y: nodePositions[idx].Y},
		}

		// Fill in task details if task exists
		if taskExists {
			node.Phase = task.Status.Phase
			node.StartTime = task.Status.StartTime
			node.EndTime = task.Status.CompletionTime
			node.Type = task.Spec.Type
			node.Input = truncateForUI(task.Spec.Input, 200)

			// Calculate duration
			if task.Status.StartTime != nil {
				if task.Status.CompletionTime != nil {
					duration := task.Status.CompletionTime.Sub(task.Status.StartTime.Time)
					node.Duration = formatDuration(duration)
				} else if task.Status.Phase == mcallv1.McallTaskPhaseRunning {
					duration := time.Since(task.Status.StartTime.Time)
					node.Duration = formatDuration(duration) + " (running)"
				}
			}

			// Task result
			if task.Status.Result != nil {
				node.Output = truncateForUI(task.Status.Result.Output, 500)
				node.ErrorCode = task.Status.Result.ErrorCode
				node.ErrorMessage = task.Status.Result.ErrorMessage
			}

			// Update metadata counts
			switch node.Phase {
			case mcallv1.McallTaskPhaseSucceeded:
				dag.Metadata.SuccessCount++
			case mcallv1.McallTaskPhaseFailed:
				dag.Metadata.FailureCount++
			case mcallv1.McallTaskPhaseRunning:
				dag.Metadata.RunningCount++
			case mcallv1.McallTaskPhasePending:
				dag.Metadata.PendingCount++
			case mcallv1.McallTaskPhaseSkipped:
				dag.Metadata.SkippedCount++
			}
		} else {
			dag.Metadata.PendingCount++
		}

		dag.Nodes = append(dag.Nodes, node)
	}

	// Build edges from dependencies and conditions
	for _, taskSpec := range workflow.Spec.Tasks {
		// Standard dependency edges
		for _, dep := range taskSpec.Dependencies {
			edge := mcallv1.DAGEdge{
				ID:     fmt.Sprintf("%s-%s", dep, taskSpec.Name),
				Source: dep,
				Target: taskSpec.Name,
				Type:   "dependency",
			}

			// Add condition information
			if taskSpec.Condition != nil {
				edge.Type = taskSpec.Condition.When
				edge.Condition = taskSpec.Condition.When
				switch taskSpec.Condition.When {
				case "success":
					edge.Label = "✓"
				case "failure":
					edge.Label = "✗"
				case "always":
					edge.Label = "*"
				default:
					edge.Label = taskSpec.Condition.When
				}
			}

			dag.Edges = append(dag.Edges, edge)
			dag.Metadata.TotalEdges++
		}
	}

	// Update workflow status
	workflow.Status.DAG = dag

	log.Info("Built workflow DAG",
		"workflow", workflow.Name,
		"nodes", len(dag.Nodes),
		"edges", len(dag.Edges),
		"success", dag.Metadata.SuccessCount,
		"running", dag.Metadata.RunningCount,
		"failed", dag.Metadata.FailureCount)

	return nil
}

// calculateNodePositions calculates positions for nodes in a simple layered layout
func (r *McallWorkflowReconciler) calculateNodePositions(nodeCount int) []mcallv1.NodePosition {
	positions := make([]mcallv1.NodePosition, nodeCount)
	
	// Simple vertical layout with centering
	nodeHeight := 100
	verticalSpacing := 150
	horizontalOffset := 250

	for i := 0; i < nodeCount; i++ {
		positions[i] = mcallv1.NodePosition{
			X: horizontalOffset,
			Y: 100 + (i * (nodeHeight + verticalSpacing)),
		}
	}

	return positions
}

// truncateForUI truncates string for UI display
func truncateForUI(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatDuration formats duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
