package dto

import "time"

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
	Name               string                 `json:"name" binding:"required"`
	Description        *string                `json:"description"`
	Definition         map[string]interface{} `json:"definition" binding:"required"`
	Schedule           *string                `json:"schedule"`
	Timezone           *string                `json:"timezone"`
	WebhookPath        *string                `json:"webhookPath"`
	WebhookRequireAuth *bool                  `json:"webhookRequireAuth"`
	// Note: IsActive removed - workflows are always active
}

// UpdateWorkflowRequest represents the request to update a workflow
type UpdateWorkflowRequest struct {
	Name               *string                `json:"name"`
	Description        *string                `json:"description"`
	Definition         map[string]interface{} `json:"definition"`
	ChangeLog          *string                `json:"change_log"`
	Schedule           *string                `json:"schedule"`
	Timezone           *string                `json:"timezone"`
	WebhookPath        *string                `json:"webhookPath"`
	WebhookRequireAuth *bool                  `json:"webhookRequireAuth"`
	// Note: IsActive removed - workflows are always active
}

// UpdateScheduleRequest represents the request to update workflow schedule
type UpdateScheduleRequest struct {
	Schedule *string `json:"schedule"` // Can be null to clear, omitted to keep existing
	Timezone *string `json:"timezone"`
	IsActive *bool   `json:"isActive"` // Use camelCase to match frontend
}

// ExecuteWorkflowRequest represents the request to execute a workflow
type ExecuteWorkflowRequest struct {
	Input map[string]interface{} `json:"input"`
}

// WorkflowCreator represents the workflow creator information
type WorkflowCreator struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// WorkflowCount represents workflow counts
type WorkflowCount struct {
	Executions int `json:"executions"`
	Versions   int `json:"versions"`
}

// WorkflowResponse represents the workflow response
type WorkflowResponse struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Description        *string          `json:"description,omitempty"`
	IsActive           bool             `json:"isActive"`
	Schedule           *string          `json:"schedule,omitempty"`
	Timezone           string           `json:"timezone"`
	WebhookPath        *string          `json:"webhookPath,omitempty"`
	WebhookRequireAuth bool             `json:"webhookRequireAuth"`
	CurrentVersion     int              `json:"currentVersion"`
	CreatedBy          string           `json:"createdBy"` // Creator user ID
	Creator            *WorkflowCreator `json:"creator"`   // Creator details
	Count              *WorkflowCount   `json:"_count"`    // Counts
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
}

// NodeExecutionResponse represents the node execution response
type NodeExecutionResponse struct {
	ID               string     `json:"id"`
	ExecutionID      string     `json:"executionId"`
	NodeID           string     `json:"nodeId"`
	NodeType         string     `json:"nodeType"`
	Status           string     `json:"status"`
	Input            *string    `json:"input,omitempty"`
	Output           *string    `json:"output,omitempty"`
	Error            *string    `json:"error,omitempty"`
	ParentLoopNodeID *string    `json:"parentLoopNodeId,omitempty"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// ExecutionResponse represents the workflow execution response
type ExecutionResponse struct {
	ID             string                  `json:"id"`
	WorkflowID     string                  `json:"workflowId"`
	Workflow       *WorkflowResponse       `json:"workflow,omitempty"`
	Version        int                     `json:"version"`
	Status         string                  `json:"status"`
	TriggerType    string                  `json:"triggerType"`
	Input          *string                 `json:"input,omitempty"`
	Output         *string                 `json:"output,omitempty"`
	Error          *string                 `json:"error,omitempty"`
	StartedAt      *time.Time              `json:"startedAt,omitempty"`
	CompletedAt    *time.Time              `json:"completedAt,omitempty"`
	NodeExecutions []NodeExecutionResponse `json:"nodeExecutions"`
}
