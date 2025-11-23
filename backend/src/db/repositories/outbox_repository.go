package repositories

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type outboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository creates a new outbox repository
func NewOutboxRepository(db *gorm.DB) OutboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) CountOrphanedMessages(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OutboxMessage{}).
		Joins("LEFT JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.id IS NULL").
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count orphaned messages: %w", err)
	}
	return count, nil
}

func (r *outboxRepository) Create(ctx context.Context, message *models.OutboxMessage) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	return nil
}

func (r *outboxRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.OutboxMessage{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update outbox message: %w", err)
	}
	return nil
}

func (r *outboxRepository) UpdateOrphanedMessages(ctx context.Context, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).Model(&models.OutboxMessage{}).
		Joins("LEFT JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.id IS NULL AND outbox_messages.status NOT IN ?",
			[]string{"dead_letter", "completed"}).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update orphaned messages: %w", err)
	}
	return nil
}
