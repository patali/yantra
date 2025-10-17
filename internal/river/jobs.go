package river

import (
	"context"
	"fmt"
	"log"

	"github.com/riverqueue/river"
)

// WorkflowExecutionArgs defines the job arguments for workflow execution
type WorkflowExecutionArgs struct {
	WorkflowID  string `json:"workflow_id"`
	ExecutionID string `json:"execution_id"` // Pre-created execution record ID
	Input       string `json:"input"`        // JSON string
	TriggerType string `json:"trigger_type"` // manual, scheduled, api
}

// Kind returns the job type identifier
func (WorkflowExecutionArgs) Kind() string {
	return "workflow_execution"
}

// WorkflowExecutionWorker implements the River worker for workflow execution
type WorkflowExecutionWorker struct {
	river.WorkerDefaults[WorkflowExecutionArgs]
	engine WorkflowEngine
}

// WorkflowEngine interface to avoid circular dependencies
// The actual implementation will be injected when creating the worker
type WorkflowEngine interface {
	ExecuteWorkflow(ctx context.Context, workflowID, executionID, input, triggerType string) error
}

// NewWorkflowExecutionWorker creates a new workflow execution worker
func NewWorkflowExecutionWorker(engine WorkflowEngine) *WorkflowExecutionWorker {
	return &WorkflowExecutionWorker{
		engine: engine,
	}
}

// Work executes the workflow job
func (w *WorkflowExecutionWorker) Work(ctx context.Context, job *river.Job[WorkflowExecutionArgs]) error {
	log.Printf("🚀 Processing workflow execution job: workflow_id=%s, execution_id=%s, trigger=%s",
		job.Args.WorkflowID, job.Args.ExecutionID, job.Args.TriggerType)

	err := w.engine.ExecuteWorkflow(ctx, job.Args.WorkflowID, job.Args.ExecutionID, job.Args.Input, job.Args.TriggerType)
	if err != nil {
		log.Printf("❌ Workflow execution failed: %v", err)
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("✅ Workflow execution completed: workflow_id=%s", job.Args.WorkflowID)
	return nil
}

// PeriodicWorkflowJob represents a periodic job configuration
type PeriodicWorkflowJob struct {
	WorkflowID string
	CronExpr   string
	Timezone   string
	Input      map[string]interface{}
}

// GetPeriodicHandle generates a unique handle for the periodic job
func (p *PeriodicWorkflowJob) GetPeriodicHandle() string {
	// Use compact workflow ID (remove hyphens) similar to PG-Boss approach
	return fmt.Sprintf("wfs_%s", p.WorkflowID)
}
