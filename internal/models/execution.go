package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkflowExecution struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	WorkflowID  string     `gorm:"type:uuid;not null" json:"workflowId"`
	Version     int        `gorm:"not null" json:"version"`
	Status      string     `gorm:"not null" json:"status"`            // running, success, error
	TriggerType string     `gorm:"not null" json:"triggerType"`       // manual, scheduled, api
	Input       *string    `gorm:"type:text" json:"input,omitempty"`  // JSON string
	Output      *string    `gorm:"type:text" json:"output,omitempty"` // JSON string
	Error       *string    `gorm:"type:text" json:"error,omitempty"`
	StartedAt   time.Time  `gorm:"autoCreateTime" json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

func (we *WorkflowExecution) BeforeCreate(tx *gorm.DB) error {
	if we.ID == "" {
		we.ID = uuid.New().String()
	}
	return nil
}

type WorkflowNodeExecution struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExecutionID      string     `gorm:"type:uuid;not null" json:"executionId"`
	NodeID           string     `gorm:"not null" json:"nodeId"`
	NodeType         string     `gorm:"not null" json:"nodeType"`
	Status           string     `gorm:"not null" json:"status"`            // pending, processing, running, success, error
	Input            *string    `gorm:"type:text" json:"input,omitempty"`  // JSON string
	Output           *string    `gorm:"type:text" json:"output,omitempty"` // JSON string
	Error            *string    `gorm:"type:text" json:"error,omitempty"`
	ParentLoopNodeID *string    `gorm:"type:text" json:"parentLoopNodeId,omitempty"` // Node ID of parent loop (if this execution is part of a loop body)
	IdempotencyKey   *string    `gorm:"uniqueIndex" json:"idempotencyKey,omitempty"`
	StartedAt        time.Time  `gorm:"autoCreateTime" json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

func (WorkflowNodeExecution) TableName() string {
	return "workflow_node_executions"
}

func (wne *WorkflowNodeExecution) BeforeCreate(tx *gorm.DB) error {
	if wne.ID == "" {
		wne.ID = uuid.New().String()
	}
	return nil
}
