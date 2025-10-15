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
	"k8s.io/client-go/util/retry"
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

	// CRITICAL FIX: Double-check phase before proceeding
	if workflow.Status.Phase != mcallv1.McallWorkflowPhasePending {
		log.Info("Workflow phase changed, aborting pending handling", 
			"workflow", workflow.Name, 
			"expectedPhase", mcallv1.McallWorkflowPhasePending,
			"actualPhase", workflow.Status.Phase)
		return ctrl.Result{}, nil
	}

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

	// For scheduled workflows, update LastRunTime when starting execution
	if workflow.Spec.Schedule != "" {
		workflow.Status.LastRunTime = &metav1.Time{Time: time.Now()}
	}

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

		// CRITICAL FIX: Update LastRunTime when workflow completes for proper scheduling
		if workflow.Spec.Schedule != "" {
			workflow.Status.LastRunTime = &metav1.Time{Time: time.Now()}
		}

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

	// Update status with DAG (fetch latest version to avoid conflicts)
	log.Info("🔄 Starting DAG Status Update", "workflow", workflow.Name, "dagNodes", len(workflow.Status.DAG.Nodes), "dagEdges", len(workflow.Status.DAG.Edges))

	updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		log.Info("🔄 RetryOnConflict attempt - fetching latest workflow", "workflow", workflow.Name)

		// Get the latest version of the workflow
		latest := &mcallv1.McallWorkflow{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
		}, latest); err != nil {
			log.Error(err, "❌ Failed to get latest workflow version", "workflow", workflow.Name)
			return err
		}

		log.Info("✅ Got latest workflow version", "workflow", workflow.Name, "currentDAG", latest.Status.DAG != nil)

		// Update the DAG on the latest version
		latest.Status.DAG = workflow.Status.DAG

		log.Info("🔄 Setting DAG on latest version", "workflow", workflow.Name, "dagNodes", len(latest.Status.DAG.Nodes), "dagEdges", len(latest.Status.DAG.Edges))

		// Update the status
		log.Info("🔄 Calling Status().Update", "workflow", workflow.Name)
		updateErr := r.Status().Update(ctx, latest)
		if updateErr != nil {
			log.Error(updateErr, "❌ Status().Update failed", "workflow", workflow.Name, "error", updateErr.Error())
		} else {
			log.Info("✅ Status().Update succeeded", "workflow", workflow.Name)
		}
		return updateErr
	})

	if updateErr != nil {
		log.Error(updateErr, "❌ Failed to update workflow status with DAG after retries", "workflow", workflow.Name, "retries", retry.DefaultRetry.Steps)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	log.Info("✅ DAG Status Update completed successfully", "workflow", workflow.Name)

	// Continue monitoring
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *McallWorkflowReconciler) handleWorkflowCompleted(ctx context.Context, workflow *mcallv1.McallWorkflow) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// For scheduled workflows, clean up completed tasks and reset to Pending for next run
	if workflow.Spec.Schedule != "" {
		log.Info("Cleaning up completed scheduled workflow", "workflow", workflow.Name, "phase", workflow.Status.Phase)

		// Build final DAG before cleanup
		if err := r.buildWorkflowDAG(ctx, workflow); err != nil {
			log.Error(err, "Failed to build final DAG before cleanup", "workflow", workflow.Name)
		}

		// Delete workflow-specific task instances (not template tasks)
		if err := r.deleteWorkflowTasks(ctx, workflow); err != nil {
			log.Error(err, "Failed to delete workflow tasks", "workflow", workflow.Name)
			return ctrl.Result{}, err
		}

		// Update status with retry on conflict - reset for next run
		resetErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Get the latest version
			latest := &mcallv1.McallWorkflow{}
			if err := r.Get(ctx, types.NamespacedName{
				Name:      workflow.Name,
				Namespace: workflow.Namespace,
			}, latest); err != nil {
				return err
			}

			// Reset workflow status for next scheduled run
			// Keep DAG from last run (don't clear it)
			latest.Status.Phase = mcallv1.McallWorkflowPhasePending
			latest.Status.StartTime = nil
			latest.Status.CompletionTime = nil
			// Keep LastRunTime for proper scheduling - don't clear it
			// latest.Status.DAG = nil // Don't clear DAG - keep last run data for UI

			return r.Status().Update(ctx, latest)
		})

		if resetErr != nil {
			log.Error(resetErr, "Failed to reset workflow status after retries", "workflow", workflow.Name)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		log.Info("Workflow reset to Pending for next scheduled run",
			"workflow", workflow.Name)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// For non-scheduled workflows, just clean up
	return ctrl.Result{}, nil
}

func (r *McallWorkflowReconciler) shouldRunScheduledWorkflow(ctx context.Context, workflow *mcallv1.McallWorkflow) (bool, error) {
	log := log.FromContext(ctx)
	
	// CRITICAL FIX: Don't allow new execution if workflow is already running
	if workflow.Status.Phase == mcallv1.McallWorkflowPhaseRunning {
		log.Info("Workflow is already running, skipping new execution", 
			"workflow", workflow.Name, 
			"phase", workflow.Status.Phase,
			"startTime", workflow.Status.StartTime)
		return false, nil
	}
	
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

		// Debug: Log mcpConfig presence before any modifications
		if referencedTask.Spec.MCPConfig != nil {
			log.Info("Template task has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "template", taskRef.Name, "serverURL", referencedTask.Spec.MCPConfig.ServerURL)
		} else {
			log.Info("Template task missing mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "template", taskRef.Name, "type", referencedTask.Spec.Type)
		}

		if task.Spec.MCPConfig != nil {
			log.Info("After DeepCopy: task has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
		} else {
			log.Info("After DeepCopy: task missing mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", task.Spec.Type)
		}

		// CRITICAL FIX: Ensure mcpConfig is preserved before any modifications
		var preservedMCPConfig *mcallv1.MCPClientConfig
		if task.Spec.MCPConfig != nil {
			// Deep copy the mcpConfig to preserve it
			preservedMCPConfig = task.Spec.MCPConfig.DeepCopy()
			log.Info("PRESERVING mcpConfig before modifications", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", preservedMCPConfig.ServerURL)
		}

		// Update dependencies to use workflow task names
		task.Spec.Dependencies = r.convertDependencies(workflow.Name, taskSpec.Dependencies)

		// Debug: Check mcpConfig after Dependencies modification
		if task.Spec.MCPConfig != nil {
			log.Info("After Dependencies: task still has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)
		} else {
			log.Info("After Dependencies: task lost mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", task.Spec.Type)

			// CRITICAL FIX: Restore mcpConfig if it was lost
			if preservedMCPConfig != nil {
				task.Spec.MCPConfig = preservedMCPConfig
				log.Info("RESTORED mcpConfig after Dependencies modification", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
			}
		}

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

			// Debug: Check mcpConfig after InputSources modification
			if task.Spec.MCPConfig != nil {
				log.Info("After InputSources: task still has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)
			} else {
				log.Info("After InputSources: task lost mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", task.Spec.Type)

				// CRITICAL FIX: Restore mcpConfig if it was lost
				if preservedMCPConfig != nil {
					task.Spec.MCPConfig = preservedMCPConfig
					log.Info("RESTORED mcpConfig after InputSources modification", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
				}
			}
		}

		// Set InputTemplate if specified
		if taskSpec.InputTemplate != "" {
			task.Spec.InputTemplate = taskSpec.InputTemplate

			log.Info("Set task input template",
				"workflow", workflow.Name,
				"task", taskSpec.Name,
				"template", taskSpec.InputTemplate)

			// Debug: Check mcpConfig after InputTemplate modification
			if task.Spec.MCPConfig != nil {
				log.Info("After InputTemplate: task still has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)
			} else {
				log.Info("After InputTemplate: task lost mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", task.Spec.Type)

				// CRITICAL FIX: Restore mcpConfig if it was lost
				if preservedMCPConfig != nil {
					task.Spec.MCPConfig = preservedMCPConfig
					log.Info("RESTORED mcpConfig after InputTemplate modification", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
				}
			}
		}

		// CRITICAL FIX: Final mcpConfig validation and restoration
		if task.Spec.Type == "mcp-client" {
			if task.Spec.MCPConfig == nil && preservedMCPConfig != nil {
				task.Spec.MCPConfig = preservedMCPConfig
				log.Info("FINAL RESTORATION: mcpConfig restored before Create", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
			} else if task.Spec.MCPConfig != nil {
				log.Info("FINAL VALIDATION: mcpConfig is present before Create", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
			} else {
				log.Error(nil, "CRITICAL ERROR: mcp-client task has no mcpConfig before Create", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)
				return fmt.Errorf("mcp-client task %s has no mcpConfig before Create", task.Name)
			}
		}

		// ULTIMATE FIX: Force mcpConfig from template if missing for mcp-client tasks
		if task.Spec.Type == "mcp-client" && task.Spec.MCPConfig == nil {
			log.Error(nil, "ULTIMATE FIX: mcp-client task missing mcpConfig, attempting to get from template", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

			// Try to get mcpConfig from the template task again
			var templateTask mcallv1.McallTask
			if templateErr := r.Get(ctx, types.NamespacedName{Name: taskRef.Name, Namespace: taskRef.Namespace}, &templateTask); templateErr == nil {
				if templateTask.Spec.MCPConfig != nil {
					task.Spec.MCPConfig = templateTask.Spec.MCPConfig.DeepCopy()
					log.Info("ULTIMATE FIX: Successfully copied mcpConfig from template", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
				} else {
					log.Error(nil, "ULTIMATE FIX FAILED: Template task also missing mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "template", taskRef.Name)
					return fmt.Errorf("template task %s is missing mcpConfig for mcp-client task %s", taskRef.Name, task.Name)
				}
			} else {
				log.Error(templateErr, "ULTIMATE FIX FAILED: Could not fetch template task", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "template", taskRef.Name)
				return fmt.Errorf("could not fetch template task %s for mcp-client task %s", taskRef.Name, task.Name)
			}
		}

		// Debug: Final check before Create - Step 1: Detailed task object inspection
		log.Info("=== STEP 1: TASK OBJECT INSPECTION BEFORE CREATE ===", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

		if task.Spec.MCPConfig != nil {
			log.Info("STEP 1.1: task.Spec.MCPConfig is NOT nil", "workflow", workflow.Name, "task", taskSpec.Name, "serverURL", task.Spec.MCPConfig.ServerURL, "toolName", task.Spec.MCPConfig.ToolName)

			// Detailed mcpConfig inspection
			mcpConfigJSON, _ := json.Marshal(task.Spec.MCPConfig)
			log.Info("STEP 1.2: mcpConfig JSON content", "workflow", workflow.Name, "task", taskSpec.Name, "mcpConfigJSON", string(mcpConfigJSON))

			// Check each field individually
			log.Info("STEP 1.3: mcpConfig field details", "workflow", workflow.Name, "task", taskSpec.Name,
				"serverURL", task.Spec.MCPConfig.ServerURL,
				"toolName", task.Spec.MCPConfig.ToolName,
				"connectionTimeout", task.Spec.MCPConfig.ConnectionTimeout,
				"hasAuth", task.Spec.MCPConfig.Auth != nil,
				"hasArguments", task.Spec.MCPConfig.Arguments != nil,
				"hasHeaders", task.Spec.MCPConfig.Headers != nil)
		} else {
			log.Info("STEP 1.1: task.Spec.MCPConfig is NIL", "workflow", workflow.Name, "task", taskSpec.Name, "type", task.Spec.Type)
		}

		// Debug: Serialize entire task spec to JSON for inspection
		taskSpecJSON, _ := json.Marshal(task.Spec)
		log.Info("STEP 1.4: Complete task.Spec JSON", "workflow", workflow.Name, "task", taskSpec.Name, "taskSpecJSON", string(taskSpecJSON))

		log.Info("=== STEP 2: CALLING r.Create(ctx, task) ===", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

		if err := r.Create(ctx, task); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Task already exists, check if it's running before recreating
				log.Info("Task already exists, checking if recreation is needed", "workflow", workflow.Name, "task", taskSpec.Name)

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
					// Check if task is currently running
					if existingTask.Status.Phase == "Running" {
						log.Info("Task is currently running, skipping recreation to avoid race condition", "workflow", workflow.Name, "task", taskSpec.Name, "phase", existingTask.Status.Phase)
						continue // Skip recreation for running tasks
					}

					// CRITICAL FIX: Check if existing task is missing mcpConfig and fix it
					if existingTask.Spec.Type == "mcp-client" && existingTask.Spec.MCPConfig == nil && task.Spec.MCPConfig != nil {
						log.Info("CRITICAL FIX: Existing task missing mcpConfig, updating instead of recreating", "workflow", workflow.Name, "task", taskSpec.Name, "existingTask", existingTask.Name)

						// Update the existing task with mcpConfig
						existingTask.Spec.MCPConfig = task.Spec.MCPConfig

						if updateErr := r.Update(ctx, existingTask); updateErr != nil {
							log.Error(updateErr, "Failed to update existing task with mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name)
							// Fall through to delete and recreate if update fails
						} else {
							log.Info("CRITICAL FIX: Successfully updated existing task with mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
							continue // Skip deletion and recreation
						}
					}

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

				// CRITICAL FIX: Ensure mcpConfig is preserved before recreation
				var preservedMCPConfigForRecreation *mcallv1.MCPClientConfig
				if task.Spec.MCPConfig != nil {
					preservedMCPConfigForRecreation = task.Spec.MCPConfig.DeepCopy()
					log.Info("PRESERVING mcpConfig before recreation", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", preservedMCPConfigForRecreation.ServerURL)
				}

				// Debug: Check mcpConfig before recreating - Step 3: Recreation inspection
				log.Info("=== STEP 3: TASK RECREATION INSPECTION ===", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

				if task.Spec.MCPConfig != nil {
					log.Info("STEP 3.1: BEFORE Recreate - task has mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL, "toolName", task.Spec.MCPConfig.ToolName)

					// Detailed mcpConfig inspection for recreation
					mcpConfigJSON, _ := json.Marshal(task.Spec.MCPConfig)
					log.Info("STEP 3.2: BEFORE Recreate - mcpConfig JSON content", "workflow", workflow.Name, "task", taskSpec.Name, "mcpConfigJSON", string(mcpConfigJSON))
				} else {
					log.Info("STEP 3.1: BEFORE Recreate - task missing mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", task.Spec.Type)
				}

				// CRITICAL FIX: Final mcpConfig validation before recreation
				if preservedMCPConfigForRecreation != nil && task.Spec.MCPConfig == nil {
					task.Spec.MCPConfig = preservedMCPConfigForRecreation
					log.Info("FINAL RESTORATION BEFORE RECREATION: mcpConfig restored", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", task.Spec.MCPConfig.ServerURL)
				}

				log.Info("=== STEP 4: CALLING r.Create(ctx, task) FOR RECREATION ===", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

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
			log.Info("=== STEP 2.1: TASK CREATED SUCCESSFULLY ===", "workflow", workflow.Name, "task", taskSpec.Name, "originalTask", taskRef.Name, "dependencies", taskSpec.Dependencies)

			// Debug: Verify mcpConfig after creation by fetching from Kubernetes - Step 5: Post-creation verification
			log.Info("=== STEP 5: POST-CREATION KUBERNETES VERIFICATION ===", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

			time.Sleep(200 * time.Millisecond) // Increased delay to ensure resource is fully created
			var createdTask mcallv1.McallTask
			if getErr := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, &createdTask); getErr == nil {
				log.Info("STEP 5.1: Successfully fetched created task from Kubernetes", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name)

				if createdTask.Spec.MCPConfig != nil {
					log.Info("STEP 5.2: ✅ Kubernetes task HAS mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "serverURL", createdTask.Spec.MCPConfig.ServerURL, "toolName", createdTask.Spec.MCPConfig.ToolName)

					// Detailed comparison between sent and received mcpConfig
					createdMCPConfigJSON, _ := json.Marshal(createdTask.Spec.MCPConfig)
					log.Info("STEP 5.3: Created task mcpConfig JSON", "workflow", workflow.Name, "task", taskSpec.Name, "createdMCPConfigJSON", string(createdMCPConfigJSON))

					// Compare with original task spec
					if task.Spec.MCPConfig != nil {
						originalMCPConfigJSON, _ := json.Marshal(task.Spec.MCPConfig)
						log.Info("STEP 5.4: Original task mcpConfig JSON for comparison", "workflow", workflow.Name, "task", taskSpec.Name, "originalMCPConfigJSON", string(originalMCPConfigJSON))

						if string(originalMCPConfigJSON) == string(createdMCPConfigJSON) {
							log.Info("STEP 5.5: ✅ mcpConfig MATCHES between original and created task", "workflow", workflow.Name, "task", taskSpec.Name)
						} else {
							log.Info("STEP 5.5: ❌ mcpConfig MISMATCH between original and created task", "workflow", workflow.Name, "task", taskSpec.Name)
						}
					}
				} else {
					log.Info("STEP 5.2: ❌ Kubernetes task MISSING mcpConfig", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "type", createdTask.Spec.Type)

					// Show what was actually stored in Kubernetes
					createdTaskSpecJSON, _ := json.Marshal(createdTask.Spec)
					log.Info("STEP 5.3: Created task complete spec JSON", "workflow", workflow.Name, "task", taskSpec.Name, "createdTaskSpecJSON", string(createdTaskSpecJSON))
				}
			} else {
				log.Info("STEP 5.1: ❌ Could not fetch created task for verification", "workflow", workflow.Name, "task", taskSpec.Name, "createdTask", task.Name, "error", getErr.Error())
			}
		}
	}

	return nil
}

func (r *McallWorkflowReconciler) deleteWorkflowTasks(ctx context.Context, workflow *mcallv1.McallWorkflow) error {
	log := log.FromContext(ctx)

	log.Info("=== WORKFLOW DELETION: STARTING TASK CLEANUP ===", "workflow", workflow.Name)

	// Delete all workflow-specific task instances using labels
	// Template tasks should NOT have the workflow label
	var tasks mcallv1.McallTaskList
	if err := r.List(ctx, &tasks,
		client.InNamespace(workflow.Namespace),
		client.MatchingLabels{"mcall.tz.io/workflow": workflow.Name}); err != nil {
		log.Error(err, "Failed to list workflow tasks for deletion", "workflow", workflow.Name)
		return err
	}

	log.Info("WORKFLOW DELETION: Found tasks to delete", "workflow", workflow.Name, "taskCount", len(tasks.Items))

	tasksToDelete := []string{}
	for i, task := range tasks.Items {
		log.Info("WORKFLOW DELETION: Processing task", "workflow", workflow.Name, "taskIndex", i+1, "taskName", task.Name, "taskType", task.Spec.Type)

		// Skip template tasks (they have -template suffix)
		if strings.HasSuffix(task.Name, "-template") {
			log.Info("WORKFLOW DELETION: Skipping template task", "workflow", workflow.Name, "task", task.Name)
			continue
		}

		// Check task status before deletion
		log.Info("WORKFLOW DELETION: Task status before deletion", "workflow", workflow.Name, "task", task.Name,
			"phase", task.Status.Phase,
			"hasFinalizer", len(task.Finalizers) > 0,
			"deletionTimestamp", task.DeletionTimestamp != nil)

		if err := r.Delete(ctx, &task); err != nil {
			log.Error(err, "Failed to delete workflow task", "workflow", workflow.Name, "task", task.Name)
			return err
		}

		tasksToDelete = append(tasksToDelete, task.Name)
		log.Info("WORKFLOW DELETION: Successfully initiated deletion", "workflow", workflow.Name, "task", task.Name)
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

	// Generate unique RunID
	runID := fmt.Sprintf("%s-%s", workflow.Name, time.Now().Format("20060102-150405"))

	dag := &mcallv1.WorkflowDAG{
		RunID:         runID,
		Timestamp:     &metav1.Time{Time: time.Now()},
		WorkflowPhase: workflow.Status.Phase,
		Nodes:         []mcallv1.DAGNode{},
		Edges:         []mcallv1.DAGEdge{},
		Layout:        "dagre",
		Metadata: mcallv1.DAGMetadata{
			TotalNodes: len(workflow.Spec.Tasks),
		},
	}

	// Build nodes from tasks
	nodePositions := r.calculateNodePositions(workflow)

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

		// Get task template for default type
		var taskTemplate mcallv1.McallTask
		taskType := "cmd" // default
		if err := r.Get(ctx, types.NamespacedName{
			Name:      taskSpec.TaskRef.Name,
			Namespace: taskSpec.TaskRef.Namespace,
		}, &taskTemplate); err == nil {
			taskType = taskTemplate.Spec.Type
		}

		// Get position for this task
		pos, exists := nodePositions[taskSpec.Name]
		if !exists {
			// Fallback position if not found
			pos = mcallv1.NodePosition{X: 250, Y: 100 + (idx * 250)}
		}

		// Create node
		node := mcallv1.DAGNode{
			ID:       taskSpec.Name,
			Name:     taskSpec.Name,
			Type:     taskType, // Use template type
			Phase:    mcallv1.McallTaskPhasePending,
			TaskRef:  taskSpec.TaskRef.Name,
			Position: &pos,
		}

		// Fill in task details if task exists
		if taskExists {
			node.Phase = task.Status.Phase
			node.StartTime = task.Status.StartTime
			node.EndTime = task.Status.CompletionTime
			node.Type = task.Spec.Type // Override with actual task type
			node.Input = truncateForUI(task.Spec.Input, 200)

			// Calculate duration - use ExecutionTimeMs if available for better precision
			if task.Status.ExecutionTimeMs > 0 {
				node.Duration = formatDurationMs(task.Status.ExecutionTimeMs)
			} else if task.Status.StartTime != nil {
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

			// HTTP status code (for HTTP requests)
			if task.Status.HTTPStatusCode != 0 {
				node.HTTPStatusCode = task.Status.HTTPStatusCode
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

		// If task has condition but no matching dependency edge, create conditional edge
		if taskSpec.Condition != nil && taskSpec.Condition.DependentTask != "" {
			// Check if edge for this dependency already exists
			edgeExists := false
			for _, dep := range taskSpec.Dependencies {
				if dep == taskSpec.Condition.DependentTask {
					edgeExists = true
					break
				}
			}

			// Create conditional edge if not already covered by dependencies
			if !edgeExists {
				edge := mcallv1.DAGEdge{
					ID:        fmt.Sprintf("%s-%s", taskSpec.Condition.DependentTask, taskSpec.Name),
					Source:    taskSpec.Condition.DependentTask,
					Target:    taskSpec.Name,
					Type:      taskSpec.Condition.When,
					Condition: taskSpec.Condition.When,
				}

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

				dag.Edges = append(dag.Edges, edge)
				dag.Metadata.TotalEdges++
			}
		}
	}

	// Update workflow status
	workflow.Status.DAG = dag

	log.Info("🎨 Built workflow DAG",
		"workflow", workflow.Name,
		"nodes", len(dag.Nodes),
		"edges", len(dag.Edges),
		"success", dag.Metadata.SuccessCount,
		"running", dag.Metadata.RunningCount,
		"failed", dag.Metadata.FailureCount,
		"runID", dag.RunID)

	// Log detailed edge information
	for i, edge := range dag.Edges {
		log.Info("🔗 DAG Edge",
			"workflow", workflow.Name,
			"edgeIndex", i,
			"source", edge.Source,
			"target", edge.Target,
			"type", edge.Type,
			"condition", edge.Condition,
			"label", edge.Label)
	}

	return nil
}

// calculateNodePositions calculates positions for nodes in a simple layered layout
func (r *McallWorkflowReconciler) calculateNodePositions(workflow *mcallv1.McallWorkflow) map[string]mcallv1.NodePosition {
	positions := make(map[string]mcallv1.NodePosition)

	// Constants for layout
	levelHeight := 250
	nodeSpacing := 300
	startY := 100

	// Build dependency graph to determine levels
	taskLevels := make(map[string]int)
	tasksByLevel := make(map[int][]string)

	// First pass: assign levels based on dependencies
	for _, taskSpec := range workflow.Spec.Tasks {
		level := 0
		if len(taskSpec.Dependencies) > 0 {
			// Task depends on others, place it one level below its dependencies
			maxDepLevel := 0
			for _, dep := range taskSpec.Dependencies {
				if depLevel, exists := taskLevels[dep]; exists {
					if depLevel > maxDepLevel {
						maxDepLevel = depLevel
					}
				}
			}
			level = maxDepLevel + 1
		}
		taskLevels[taskSpec.Name] = level
		tasksByLevel[level] = append(tasksByLevel[level], taskSpec.Name)
	}

	// Second pass: calculate positions
	for level, tasks := range tasksByLevel {
		y := startY + (level * levelHeight)
		totalWidth := len(tasks) * nodeSpacing
		startX := 250 - (totalWidth / 2) + (nodeSpacing / 2)

		for i, taskName := range tasks {
			x := startX + (i * nodeSpacing)
			positions[taskName] = mcallv1.NodePosition{
				X: x,
				Y: y,
			}
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

func formatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := float64(ms) / 1000.0
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := seconds / 60.0
	if minutes < 60 {
		return fmt.Sprintf("%.1fm", minutes)
	}
	hours := minutes / 60.0
	return fmt.Sprintf("%.1fh", hours)
}
