package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutboxMessage struct {
	ID              string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	NodeExecutionID string     `gorm:"type:uuid;not null" json:"nodeExecutionId"`
	EventType       string     `gorm:"not null" json:"eventType"`         // email.send, http.request, slack.send, etc.
	Payload         string     `gorm:"type:text;not null" json:"payload"` // JSON payload with all data needed
	Status          string     `gorm:"default:pending" json:"status"`     // pending, processing, completed, dead_letter
	IdempotencyKey  string     `gorm:"uniqueIndex;not null" json:"idempotencyKey"`
	Attempts        int        `gorm:"default:0" json:"attempts"`
	MaxAttempts     int        `gorm:"default:3" json:"maxAttempts"`
	LastError       *string    `gorm:"type:text" json:"lastError,omitempty"`
	LastAttemptAt   *time.Time `json:"lastAttemptAt,omitempty"`
	NextRetryAt     *time.Time `gorm:"index:idx_outbox_status_retry" json:"nextRetryAt,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	ProcessedAt     *time.Time `json:"processedAt,omitempty"`

	// Relationships
	NodeExecution WorkflowNodeExecution `gorm:"foreignKey:NodeExecutionID;constraint:OnDelete:CASCADE" json:"nodeExecution,omitempty"`
}

func (OutboxMessage) TableName() string {
	return "outbox_messages"
}

func (om *OutboxMessage) BeforeCreate(tx *gorm.DB) error {
	if om.ID == "" {
		om.ID = uuid.New().String()
	}
	return nil
}
