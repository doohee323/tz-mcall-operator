package controller

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	mcallv1 "github.com/doohee323/tz-mcall-operator/api/v1"
)

func TestMcallWorkflowController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McallWorkflow Controller Suite")
}

var _ = Describe("McallWorkflow Controller", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		reconciler *McallWorkflowReconciler
		mockClient client.Client
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		// Setup logger and add to context
		logger := zap.New(zap.UseDevMode(true))
		log.SetLogger(logger)
		ctx = log.IntoContext(ctx, logger)

		// Setup scheme
		scheme = runtime.NewScheme()
		Expect(mcallv1.AddToScheme(scheme)).To(Succeed())

		// Setup mock client
		mockClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&mcallv1.McallWorkflow{}).
			WithStatusSubresource(&mcallv1.McallTask{}).
			Build()

		// Setup reconciler
		reconciler = &McallWorkflowReconciler{
			Client: mockClient,
			Scheme: scheme,
		}
	})

	AfterEach(func() {
		cancel()
	})

	Context("Workflow Creation", func() {
		It("should create a simple workflow without schedule", func() {
			// Create referenced McallTask first
			referencedTask := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task1-ref",
					Namespace: "default",
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Hello World'",
					Timeout: 30,
				},
			}
			Expect(mockClient.Create(ctx, referencedTask)).To(Succeed())

			// Create workflow
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Tasks: []mcallv1.WorkflowTaskRef{
						{
							Name: "task1",
							TaskRef: mcallv1.TaskRef{
								Name:      "task1-ref",
								Namespace: "default",
							},
						},
					},
				},
			}

			Expect(mockClient.Create(ctx, workflow)).To(Succeed())

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(1 * time.Second))

			// Check workflow status
			var updatedWorkflow mcallv1.McallWorkflow
			Expect(mockClient.Get(ctx, types.NamespacedName{
				Name:      "test-workflow",
				Namespace: "default",
			}, &updatedWorkflow)).To(Succeed())
			Expect(updatedWorkflow.Status.Phase).To(Equal(mcallv1.McallWorkflowPhasePending))

			// Reconcile again to create tasks
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			// Check workflow status
			Expect(mockClient.Get(ctx, types.NamespacedName{
				Name:      "test-workflow",
				Namespace: "default",
			}, &updatedWorkflow)).To(Succeed())
			Expect(updatedWorkflow.Status.Phase).To(Equal(mcallv1.McallWorkflowPhaseRunning))

			// Check if task was created
			var tasks mcallv1.McallTaskList
			Expect(mockClient.List(ctx, &tasks, client.MatchingLabels{"mcall.tz.io/workflow": "test-workflow"})).To(Succeed())
			Expect(len(tasks.Items)).To(Equal(1))
			Expect(tasks.Items[0].Name).To(Equal("test-workflow-task1"))
		})

		It("should create workflow with dependencies", func() {
			// Create referenced McallTasks first
			task1 := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task1-ref",
					Namespace: "default",
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Task 1'",
					Timeout: 30,
				},
			}
			Expect(mockClient.Create(ctx, task1)).To(Succeed())

			task2 := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task2-ref",
					Namespace: "default",
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Task 2'",
					Timeout: 30,
				},
			}
			Expect(mockClient.Create(ctx, task2)).To(Succeed())

			task3 := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task3-ref",
					Namespace: "default",
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Task 3'",
					Timeout: 30,
				},
			}
			Expect(mockClient.Create(ctx, task3)).To(Succeed())

			// Create workflow with dependencies
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dependency-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Tasks: []mcallv1.WorkflowTaskRef{
						{
							Name: "task2",
							TaskRef: mcallv1.TaskRef{
								Name:      "task2-ref",
								Namespace: "default",
							},
							Dependencies: []string{"task1"},
						},
						{
							Name: "task1",
							TaskRef: mcallv1.TaskRef{
								Name:      "task1-ref",
								Namespace: "default",
							},
						},
						{
							Name: "task3",
							TaskRef: mcallv1.TaskRef{
								Name:      "task3-ref",
								Namespace: "default",
							},
							Dependencies: []string{"task1"},
						},
					},
				},
			}

			Expect(mockClient.Create(ctx, workflow)).To(Succeed())

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "dependency-workflow",
					Namespace: "default",
				},
			}

			// First reconcile - initialize status
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(1 * time.Second))

			// Second reconcile - create tasks
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())

			// Check if tasks were created in correct order
			var tasks mcallv1.McallTaskList
			Expect(mockClient.List(ctx, &tasks, client.MatchingLabels{"mcall.tz.io/workflow": "dependency-workflow"})).To(Succeed())
			Expect(len(tasks.Items)).To(Equal(3))

			// Find tasks by name
			taskMap := make(map[string]mcallv1.McallTask)
			for _, task := range tasks.Items {
				taskMap[task.Name] = task
			}

			// Check dependencies
			Expect(taskMap["dependency-workflow-task1"].Spec.Dependencies).To(BeEmpty())
			Expect(taskMap["dependency-workflow-task2"].Spec.Dependencies).To(ContainElement("dependency-workflow-task1"))
			Expect(taskMap["dependency-workflow-task3"].Spec.Dependencies).To(ContainElement("dependency-workflow-task1"))
		})
	})

	Context("Cron Scheduling", func() {
		It("should handle workflow with cron schedule", func() {
			// Create referenced McallTask first
			referencedTask := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scheduled-task-ref",
					Namespace: "default",
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Scheduled task'",
					Timeout: 30,
				},
			}
			Expect(mockClient.Create(ctx, referencedTask)).To(Succeed())

			// Create workflow with schedule
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scheduled-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Schedule: "0 2 * * *", // Every day at 2 AM
					Tasks: []mcallv1.WorkflowTaskRef{
						{
							Name: "scheduled-task",
							TaskRef: mcallv1.TaskRef{
								Name:      "scheduled-task-ref",
								Namespace: "default",
							},
						},
					},
				},
			}

			Expect(mockClient.Create(ctx, workflow)).To(Succeed())

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "scheduled-workflow",
					Namespace: "default",
				},
			}

			// First reconcile - initialize status
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(1 * time.Second))

			// Second reconcile - should run immediately for first time
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())

			// Check workflow status
			var updatedWorkflow mcallv1.McallWorkflow
			Expect(mockClient.Get(ctx, types.NamespacedName{
				Name:      "scheduled-workflow",
				Namespace: "default",
			}, &updatedWorkflow)).To(Succeed())
			Expect(updatedWorkflow.Status.Phase).To(Equal(mcallv1.McallWorkflowPhaseRunning))
		})
	})

	Context("Workflow Status Management", func() {
		It("should update workflow status when all tasks complete", func() {
			// Create workflow
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Tasks: []mcallv1.WorkflowTaskRef{
						{
							Name: "status-task",
							TaskRef: mcallv1.TaskRef{
								Name:      "status-task-ref",
								Namespace: "default",
							},
						},
					},
				},
				Status: mcallv1.McallWorkflowStatus{
					Phase: mcallv1.McallWorkflowPhaseRunning,
				},
			}

			Expect(mockClient.Create(ctx, workflow)).To(Succeed())

			// Create completed task
			task := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-workflow-status-task",
					Namespace: "default",
					Labels: map[string]string{
						"mcall.tz.io/workflow": "status-workflow",
					},
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Status task'",
					Timeout: 30,
				},
				Status: mcallv1.McallTaskStatus{
					Phase: mcallv1.McallTaskPhaseSucceeded,
				},
			}

			Expect(mockClient.Create(ctx, task)).To(Succeed())

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "status-workflow",
					Namespace: "default",
				},
			}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())

			// Check workflow status
			var updatedWorkflow mcallv1.McallWorkflow
			Expect(mockClient.Get(ctx, types.NamespacedName{
				Name:      "status-workflow",
				Namespace: "default",
			}, &updatedWorkflow)).To(Succeed())
			Expect(updatedWorkflow.Status.Phase).To(Equal(mcallv1.McallWorkflowPhaseSucceeded))
		})

		It("should update workflow status to failed when task fails", func() {
			// Create workflow
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Tasks: []mcallv1.WorkflowTaskRef{
						{
							Name: "failed-task",
							TaskRef: mcallv1.TaskRef{
								Name:      "failed-task-ref",
								Namespace: "default",
							},
						},
					},
				},
				Status: mcallv1.McallWorkflowStatus{
					Phase: mcallv1.McallWorkflowPhaseRunning,
				},
			}

			Expect(mockClient.Create(ctx, workflow)).To(Succeed())

			// Create failed task
			task := &mcallv1.McallTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-workflow-failed-task",
					Namespace: "default",
					Labels: map[string]string{
						"mcall.tz.io/workflow": "failed-workflow",
					},
				},
				Spec: mcallv1.McallTaskSpec{
					Type:    "cmd",
					Input:   "echo 'Failed task'",
					Timeout: 30,
				},
				Status: mcallv1.McallTaskStatus{
					Phase: mcallv1.McallTaskPhaseFailed,
				},
			}

			Expect(mockClient.Create(ctx, task)).To(Succeed())

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "failed-workflow",
					Namespace: "default",
				},
			}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())

			// Check workflow status
			var updatedWorkflow mcallv1.McallWorkflow
			Expect(mockClient.Get(ctx, types.NamespacedName{
				Name:      "failed-workflow",
				Namespace: "default",
			}, &updatedWorkflow)).To(Succeed())
			Expect(updatedWorkflow.Status.Phase).To(Equal(mcallv1.McallWorkflowPhaseFailed))
		})
	})

	Context("Dependency Sorting", func() {
		It("should sort tasks by dependencies correctly", func() {
			tasks := []mcallv1.WorkflowTaskRef{
				{
					Name: "task3",
					TaskRef: mcallv1.TaskRef{
						Name:      "task3-ref",
						Namespace: "default",
					},
					Dependencies: []string{"task1", "task2"},
				},
				{
					Name: "task1",
					TaskRef: mcallv1.TaskRef{
						Name:      "task1-ref",
						Namespace: "default",
					},
				},
				{
					Name: "task2",
					TaskRef: mcallv1.TaskRef{
						Name:      "task2-ref",
						Namespace: "default",
					},
					Dependencies: []string{"task1"},
				},
			}

			sortedTasks := reconciler.sortTasksByDependencies(tasks)
			Expect(len(sortedTasks)).To(Equal(3))
			Expect(sortedTasks[0].Name).To(Equal("task1"))
			Expect(sortedTasks[1].Name).To(Equal("task2"))
			Expect(sortedTasks[2].Name).To(Equal("task3"))
		})

		It("should handle circular dependencies", func() {
			tasks := []mcallv1.WorkflowTaskRef{
				{
					Name: "task1",
					TaskRef: mcallv1.TaskRef{
						Name:      "task1-ref",
						Namespace: "default",
					},
					Dependencies: []string{"task2"},
				},
				{
					Name: "task2",
					TaskRef: mcallv1.TaskRef{
						Name:      "task2-ref",
						Namespace: "default",
					},
					Dependencies: []string{"task1"},
				},
			}

			sortedTasks := reconciler.sortTasksByDependencies(tasks)
			Expect(len(sortedTasks)).To(Equal(2))
			// Should still return tasks even with circular dependency
		})
	})

	Context("Dependency Conversion", func() {
		It("should convert dependencies correctly", func() {
			dependencies := []string{"task1", "task2"}
			converted := reconciler.convertDependencies("workflow-name", dependencies)
			Expect(converted).To(Equal([]string{"workflow-name-task1", "workflow-name-task2"}))
		})
	})
})

var _ = Describe("CronScheduler", func() {
	var scheduler *CronScheduler
	var mockClient client.Client
	var scheme *runtime.Scheme

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(mcallv1.AddToScheme(scheme)).To(Succeed())

		mockClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&mcallv1.McallWorkflow{}).
			Build()

		scheduler = NewCronScheduler(mockClient)
	})

	Context("Cron Expression Parsing", func() {
		It("should parse valid cron expressions", func() {
			expr, err := scheduler.ParseCronExpression("0 2 * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(expr.Minute).To(Equal("0"))
			Expect(expr.Hour).To(Equal("2"))
			Expect(expr.Day).To(Equal("*"))
			Expect(expr.Month).To(Equal("*"))
			Expect(expr.Weekday).To(Equal("*"))
		})

		It("should reject invalid cron expressions", func() {
			_, err := scheduler.ParseCronExpression("0 2 *")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Cron Field Matching", func() {
		It("should match wildcard fields", func() {
			Expect(scheduler.matchesField("*", 5)).To(BeTrue())
		})

		It("should match exact values", func() {
			Expect(scheduler.matchesField("5", 5)).To(BeTrue())
			Expect(scheduler.matchesField("5", 3)).To(BeFalse())
		})

		It("should match ranges", func() {
			Expect(scheduler.matchesField("1-5", 3)).To(BeTrue())
			Expect(scheduler.matchesField("1-5", 7)).To(BeFalse())
		})

		It("should match step values", func() {
			Expect(scheduler.matchesField("*/5", 10)).To(BeTrue())
			Expect(scheduler.matchesField("*/5", 7)).To(BeFalse())
		})

		It("should match comma-separated values", func() {
			Expect(scheduler.matchesField("1,3,5", 3)).To(BeTrue())
			Expect(scheduler.matchesField("1,3,5", 2)).To(BeFalse())
		})
	})

	Context("Workflow Scheduling", func() {
		It("should run workflow without schedule immediately", func() {
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-schedule-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					// No schedule
				},
			}

			shouldRun, err := scheduler.ShouldRun(context.Background(), workflow)
			Expect(err).ToNot(HaveOccurred())
			Expect(shouldRun).To(BeTrue())
		})

		It("should run workflow with schedule on first run", func() {
			workflow := &mcallv1.McallWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "first-run-workflow",
					Namespace: "default",
				},
				Spec: mcallv1.McallWorkflowSpec{
					Schedule: "0 2 * * *",
				},
				Status: mcallv1.McallWorkflowStatus{
					// No LastRunTime - first run
				},
			}

			shouldRun, err := scheduler.ShouldRun(context.Background(), workflow)
			Expect(err).ToNot(HaveOccurred())
			Expect(shouldRun).To(BeTrue())
		})
	})
})
