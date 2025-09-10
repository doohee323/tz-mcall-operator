package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcallv1 "github.com/doohee323/tz-mcall-crd/api/v1"
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
		if err := r.Status().Update(ctx, workflow); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Workflow completed", "workflow", workflow.Name, "phase", workflow.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Continue monitoring
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *McallWorkflowReconciler) handleWorkflowCompleted(ctx context.Context, workflow *mcallv1.McallWorkflow) (ctrl.Result, error) {
	// Clean up resources if needed
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
			},
			Spec: referencedTask.Spec,
		}

		// Update dependencies to use workflow task names
		task.Spec.Dependencies = r.convertDependencies(workflow.Name, taskSpec.Dependencies)

		if err := r.Create(ctx, task); err != nil {
			log.Error(err, "Failed to create task", "workflow", workflow.Name, "task", taskSpec.Name)
			return err
		}

		log.Info("Created task for workflow", "workflow", workflow.Name, "task", taskSpec.Name, "originalTask", taskRef.Name, "dependencies", taskSpec.Dependencies)
	}

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
