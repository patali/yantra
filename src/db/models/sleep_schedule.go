package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SleepSchedule represents a scheduled wake-up for a sleeping workflow execution
type SleepSchedule struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ExecutionID string    `gorm:"type:uuid;not null;index" json:"executionId"` // References workflow_executions
	WorkflowID  string    `gorm:"type:uuid;not null" json:"workflowId"`
	NodeID      string    `gorm:"not null" json:"nodeId"`      // The sleep node ID
	WakeUpAt    time.Time `gorm:"not null;index" json:"wakeUpAt"` // When to wake up (UTC)
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (SleepSchedule) TableName() string {
	return "workflow_sleep_schedules"
}

func (ss *SleepSchedule) BeforeCreate(tx *gorm.DB) error {
	if ss.ID == "" {
		ss.ID = uuid.New().String()
	}
	return nil
}
