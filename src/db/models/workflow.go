package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Workflow struct {
	ID             string            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name           string            `gorm:"not null" json:"name"`
	Description    *string           `json:"description,omitempty"`
	IsActive       bool              `gorm:"default:true" json:"isActive"`
	Schedule       *string           `json:"schedule,omitempty"` // Cron expression
	Timezone       string            `gorm:"default:UTC" json:"timezone"`
	CurrentVersion int               `gorm:"default:1" json:"currentVersion"`
	AccountID      *string           `gorm:"type:uuid" json:"accountId,omitempty"`
	CreatedBy      string            `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt      time.Time         `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time         `gorm:"autoUpdateTime" json:"updatedAt"`
	Versions       []WorkflowVersion `gorm:"-" json:"versions,omitempty"` // Not stored in DB, populated manually
}

func (Workflow) TableName() string {
	return "workflows"
}

func (w *Workflow) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}

type WorkflowVersion struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	WorkflowID string    `gorm:"type:uuid;not null" json:"workflowId"`
	Version    int       `gorm:"not null" json:"version"`
	Definition string    `gorm:"type:text;not null" json:"definition"` // JSON string
	ChangeLog  *string   `json:"changeLog,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (WorkflowVersion) TableName() string {
	return "workflow_versions"
}

func (wv *WorkflowVersion) BeforeCreate(tx *gorm.DB) error {
	if wv.ID == "" {
		wv.ID = uuid.New().String()
	}
	return nil
}
