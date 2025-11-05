package executors

import (
	"context"
	"time"
)

// ExecutionContext holds the context for node execution
type ExecutionContext struct {
	NodeID       string                 `json:"node_id"`
	NodeConfig   map[string]interface{} `json:"node_config"`
	Input        interface{}            `json:"input"`
	WorkflowData map[string]interface{} `json:"workflow_data"`
	ExecutionID  string                 `json:"execution_id"`
	AccountID    string                 `json:"account_id"`
}

// ExecutionResult holds the result of node execution
type ExecutionResult struct {
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	NeedsSleep bool                   `json:"needs_sleep,omitempty"` // If true, workflow should enter sleeping state
	WakeUpAt   *time.Time             `json:"wake_up_at,omitempty"`  // When to resume execution (UTC)
}

// Executor interface that all node executors must implement
type Executor interface {
	Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error)
}

// NodeRequiresOutbox returns true if the node type requires outbox pattern
// Outbox pattern should only be used for side effects that need retry logic
// HTTP nodes should be synchronous so their output can be used by downstream nodes
func NodeRequiresOutbox(nodeType string) bool {
	return IsAsyncNode(nodeType)
}
