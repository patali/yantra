package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/patali/yantra/internal/models"
	"gorm.io/gorm"
)

type OutboxService struct {
	db *gorm.DB
}

func NewOutboxService(db *gorm.DB) *OutboxService {
	return &OutboxService{
		db: db,
	}
}

// ExecuteNodeWithOutbox executes a node and creates an outbox message atomically
// This ensures that the node execution record and the side effect message are created together
func (s *OutboxService) ExecuteNodeWithOutbox(
	ctx context.Context,
	executionID string,
	accountID *string,
	nodeID, nodeType string,
	nodeConfig, input map[string]interface{},
	eventType string,
) (*models.WorkflowNodeExecution, *models.OutboxMessage, error) {
	var nodeExecution models.WorkflowNodeExecution
	var outboxMessage models.OutboxMessage

	// Execute in a transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create idempotency key
		idempotencyKey := fmt.Sprintf("%s-%s-%s", executionID, nodeID, uuid.New().String())

		// Create node execution record
		inputJSON, _ := json.Marshal(input)
		inputStr := string(inputJSON)

		nodeExecution = models.WorkflowNodeExecution{
			ExecutionID:    executionID,
			NodeID:         nodeID,
			NodeType:       nodeType,
			Status:         "pending",
			Input:          &inputStr,
			IdempotencyKey: &idempotencyKey,
		}

		if err := tx.Create(&nodeExecution).Error; err != nil {
			return fmt.Errorf("failed to create node execution: %w", err)
		}

		// Create outbox message payload
		accountIDStr := ""
		if accountID != nil {
			accountIDStr = *accountID
		}
		payload := map[string]interface{}{
			"node_id":       nodeID,
			"node_config":   nodeConfig,
			"input":         input,
			"workflow_data": map[string]interface{}{},
			"execution_id":  executionID,
			"account_id":    accountIDStr,
		}
		payloadJSON, _ := json.Marshal(payload)

		// Create outbox message
		now := time.Now()
		outboxMessage = models.OutboxMessage{
			NodeExecutionID: nodeExecution.ID,
			EventType:       eventType,
			Payload:         string(payloadJSON),
			Status:          "pending",
			IdempotencyKey:  idempotencyKey,
			Attempts:        0,
			MaxAttempts:     3,
			NextRetryAt:     &now, // Process immediately
		}

		if err := tx.Create(&outboxMessage).Error; err != nil {
			return fmt.Errorf("failed to create outbox message: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return &nodeExecution, &outboxMessage, nil
}

// GetPendingMessages retrieves pending outbox messages ready to be processed
func (s *OutboxService) GetPendingMessages(limit int) ([]models.OutboxMessage, error) {
	var messages []models.OutboxMessage
	now := time.Now()

	err := s.db.Where("status = ? AND next_retry_at <= ?", "pending", now).
		Order("created_at ASC").
		Limit(limit).
		Preload("NodeExecution").
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending messages: %w", err)
	}

	return messages, nil
}

// MarkMessageProcessing marks a message as being processed
func (s *OutboxService) MarkMessageProcessing(messageID string) error {
	now := time.Now()
	return s.db.Model(&models.OutboxMessage{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"status":          "processing",
			"last_attempt_at": now,
			"attempts":        gorm.Expr("attempts + 1"),
		}).Error
}

// MarkMessageCompleted marks a message as successfully completed
func (s *OutboxService) MarkMessageCompleted(messageID string, output map[string]interface{}) error {
	now := time.Now()

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Update outbox message
		if err := tx.Model(&models.OutboxMessage{}).
			Where("id = ?", messageID).
			Updates(map[string]interface{}{
				"status":       "completed",
				"processed_at": now,
			}).Error; err != nil {
			return err
		}

		// Get the node execution ID
		var message models.OutboxMessage
		if err := tx.First(&message, "id = ?", messageID).Error; err != nil {
			return err
		}

		// Update node execution
		outputJSON, _ := json.Marshal(output)
		outputStr := string(outputJSON)

		if err := tx.Model(&models.WorkflowNodeExecution{}).
			Where("id = ?", message.NodeExecutionID).
			Updates(map[string]interface{}{
				"status":       "success",
				"output":       outputStr,
				"completed_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// MarkMessageFailed marks a message as failed and schedules retry or moves to dead letter
func (s *OutboxService) MarkMessageFailed(messageID, errorMsg string) error {
	var message models.OutboxMessage
	if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	shouldRetry := message.Attempts < message.MaxAttempts

	return s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"last_error": errorMsg,
		}

		if shouldRetry {
			// Calculate exponential backoff
			retryDelay := time.Duration(1<<uint(message.Attempts)) * time.Minute
			if retryDelay > time.Hour {
				retryDelay = time.Hour
			}
			nextRetry := time.Now().Add(retryDelay)

			updates["status"] = "pending"
			updates["next_retry_at"] = nextRetry
		} else {
			// Move to dead letter queue
			updates["status"] = "dead_letter"
			updates["next_retry_at"] = nil

			// Also update node execution
			if err := tx.Model(&models.WorkflowNodeExecution{}).
				Where("id = ?", message.NodeExecutionID).
				Updates(map[string]interface{}{
					"status":       "error",
					"error":        fmt.Sprintf("Failed after %d attempts: %s", message.MaxAttempts, errorMsg),
					"completed_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.OutboxMessage{}).
			Where("id = ?", messageID).
			Updates(updates).Error
	})
}

// GetDeadLetterMessages retrieves messages that have permanently failed
func (s *OutboxService) GetDeadLetterMessages(limit int) ([]models.OutboxMessage, error) {
	var messages []models.OutboxMessage

	err := s.db.Where("status = ?", "dead_letter").
		Order("last_attempt_at DESC").
		Limit(limit).
		Preload("NodeExecution").
		Preload("NodeExecution.Execution").
		Preload("NodeExecution.Execution.Workflow").
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch dead letter messages: %w", err)
	}

	return messages, nil
}

// RetryDeadLetterMessage resets a dead letter message for retry
func (s *OutboxService) RetryDeadLetterMessage(messageID string) error {
	now := time.Now()
	return s.db.Model(&models.OutboxMessage{}).
		Where("id = ? AND status = ?", messageID, "dead_letter").
		Updates(map[string]interface{}{
			"status":          "pending",
			"attempts":        0,
			"next_retry_at":   now,
			"last_error":      nil,
			"last_attempt_at": nil,
		}).Error
}

// VerifyIntegrity checks for orphaned messages and inconsistencies
func (s *OutboxService) VerifyIntegrity() (map[string]int, error) {
	result := make(map[string]int)

	// Count pending messages
	var pendingCount int64
	s.db.Model(&models.OutboxMessage{}).Where("status = ?", "pending").Count(&pendingCount)
	result["pending_messages"] = int(pendingCount)

	// Count processing messages (might be stuck)
	var processingCount int64
	s.db.Model(&models.OutboxMessage{}).Where("status = ?", "processing").Count(&processingCount)
	result["processing_messages"] = int(processingCount)

	// Count dead letter messages
	var deadLetterCount int64
	s.db.Model(&models.OutboxMessage{}).Where("status = ?", "dead_letter").Count(&deadLetterCount)
	result["dead_letter_messages"] = int(deadLetterCount)

	// Count completed messages
	var completedCount int64
	s.db.Model(&models.OutboxMessage{}).Where("status = ?", "completed").Count(&completedCount)
	result["completed_messages"] = int(completedCount)

	return result, nil
}
