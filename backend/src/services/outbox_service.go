package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/patali/yantra/src/db/models"
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

		// Get max retries from node config (default: 3)
		maxRetries := 3
		if mr, ok := nodeConfig["maxRetries"].(float64); ok {
			maxRetries = int(mr)
		}
		// Ensure maxRetries is at least 0 and at most 10
		if maxRetries < 0 {
			maxRetries = 0
		}
		if maxRetries > 10 {
			maxRetries = 10
		}
		// MaxAttempts = maxRetries + 1 (initial attempt + retries)
		maxAttempts := maxRetries + 1
		log.Printf("  🔄 Node %s outbox message configured with %d max attempts (maxRetries=%d)", nodeID, maxAttempts, maxRetries)

		// Create outbox message
		now := time.Now()
		outboxMessage = models.OutboxMessage{
			NodeExecutionID: nodeExecution.ID,
			EventType:       eventType,
			Payload:         string(payloadJSON),
			Status:          "pending",
			IdempotencyKey:  idempotencyKey,
			Attempts:        0,
			MaxAttempts:     maxAttempts,
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

	// Only fetch pending messages (exclude cancelled, completed, dead_letter, processing)
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

		// Get the execution ID to check if all async operations are complete
		var nodeExecution models.WorkflowNodeExecution
		if err := tx.First(&nodeExecution, "id = ?", message.NodeExecutionID).Error; err != nil {
			return err
		}

		// Check if workflow execution is complete
		if err := s.checkAndCompleteWorkflowExecution(tx, nodeExecution.ExecutionID); err != nil {
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

	log.Printf("  📊 Message %s: attempts=%d, maxAttempts=%d", messageID[:8], message.Attempts, message.MaxAttempts)

	// The attempts counter has already been incremented in MarkMessageProcessing
	// So we check if current attempts >= maxAttempts (not <)
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
			log.Printf("  🔄 Message %s will retry (attempt %d/%d) in %v",
				messageID[:8], message.Attempts, message.MaxAttempts, retryDelay)
		} else {
			// Move to dead letter queue
			updates["status"] = "dead_letter"
			updates["next_retry_at"] = nil
			log.Printf("  💀 Message %s moving to dead letter (attempt %d/%d)",
				messageID[:8], message.Attempts, message.MaxAttempts)
		}

		// IMPORTANT: Update message status FIRST before checking workflow status
		// This ensures the pending count is accurate
		err := tx.Model(&models.OutboxMessage{}).
			Where("id = ?", messageID).
			Updates(updates).Error

		if err != nil {
			log.Printf("  ❌ Failed to update message %s: %v", messageID[:8], err)
			return err
		}

		log.Printf("  ✅ Message %s updated to status: %s", messageID[:8], updates["status"])

		// If message went to dead letter, update node execution and check workflow status
		if !shouldRetry {
			// Get the node execution to find the workflow execution
			var nodeExecution models.WorkflowNodeExecution
			if err := tx.First(&nodeExecution, "id = ?", message.NodeExecutionID).Error; err != nil {
				return err
			}

			// Update node execution
			if err := tx.Model(&models.WorkflowNodeExecution{}).
				Where("id = ?", message.NodeExecutionID).
				Updates(map[string]interface{}{
					"status":       "error",
					"error":        fmt.Sprintf("Failed after %d attempts: %s", message.MaxAttempts, errorMsg),
					"completed_at": time.Now(),
				}).Error; err != nil {
				return err
			}

			// Update workflow execution status
			// NOW the pending count will be correct because message is already dead_letter
			if err := s.updateWorkflowStatusOnNodeFailure(tx, nodeExecution.ExecutionID); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetDeadLetterMessages retrieves messages that have permanently failed
func (s *OutboxService) GetDeadLetterMessages(limit int) ([]models.OutboxMessage, error) {
	var messages []models.OutboxMessage

	err := s.db.Where("status = ?", "dead_letter").
		Order("last_attempt_at DESC").
		Limit(limit).
		Preload("NodeExecution").
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

// SECURITY: Account-filtered versions of the above methods

// GetDeadLetterMessagesByAccount returns dead letter messages filtered by account ID
func (s *OutboxService) GetDeadLetterMessagesByAccount(accountID string, limit int) ([]models.OutboxMessage, error) {
	var messages []models.OutboxMessage

	// Join through node_executions -> workflow_executions -> workflows to filter by account
	err := s.db.Table("outbox_messages").
		Select("outbox_messages.*").
		Joins("INNER JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Joins("INNER JOIN workflow_executions ON workflow_executions.id = workflow_node_executions.execution_id").
		Joins("INNER JOIN workflows ON workflows.id = workflow_executions.workflow_id").
		Where("workflows.account_id = ?", accountID).
		Where("outbox_messages.status = ?", "dead_letter").
		Order("outbox_messages.last_attempt_at DESC").
		Limit(limit).
		Preload("NodeExecution").
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch dead letter messages: %w", err)
	}

	return messages, nil
}

// RetryDeadLetterMessageByAccount resets a dead letter message for retry, with account ownership check
func (s *OutboxService) RetryDeadLetterMessageByAccount(messageID string, accountID string) error {
	now := time.Now()

	// Update with account ownership check using joins
	result := s.db.Table("outbox_messages").
		Joins("INNER JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Joins("INNER JOIN workflow_executions ON workflow_executions.id = workflow_node_executions.execution_id").
		Joins("INNER JOIN workflows ON workflows.id = workflow_executions.workflow_id").
		Where("outbox_messages.id = ?", messageID).
		Where("workflows.account_id = ?", accountID).
		Where("outbox_messages.status = ?", "dead_letter").
		Updates(map[string]interface{}{
			"status":          "pending",
			"attempts":        0,
			"next_retry_at":   now,
			"last_error":      nil,
			"last_attempt_at": nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found or access denied")
	}

	return nil
}

// checkAndCompleteWorkflowExecution checks if all async operations are complete and marks workflow as success
func (s *OutboxService) checkAndCompleteWorkflowExecution(tx *gorm.DB, executionID string) error {
	// Get the workflow execution
	var execution models.WorkflowExecution
	if err := tx.First(&execution, "id = ?", executionID).Error; err != nil {
		return err
	}

	// Only check if workflow is currently running
	if execution.Status != "running" {
		return nil
	}

	// Check if there are any pending/processing outbox messages
	var pendingCount int64
	tx.Model(&models.OutboxMessage{}).
		Joins("JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.execution_id = ? AND outbox_messages.status IN ?",
			executionID, []string{"pending", "processing"}).
		Count(&pendingCount)

	// If no pending messages, mark workflow as complete
	if pendingCount == 0 {
		now := time.Now()
		return tx.Model(&models.WorkflowExecution{}).
			Where("id = ?", executionID).
			Updates(map[string]interface{}{
				"status":       "success",
				"completed_at": now,
			}).Error
	}

	return nil
}

// updateWorkflowStatusOnNodeFailure updates the workflow execution status when a node goes to dead letter
func (s *OutboxService) updateWorkflowStatusOnNodeFailure(tx *gorm.DB, executionID string) error {
	// Get the workflow execution
	var execution models.WorkflowExecution
	if err := tx.First(&execution, "id = ?", executionID).Error; err != nil {
		log.Printf("❌ Failed to get execution %s: %v", executionID, err)
		return err
	}

	log.Printf("🔍 Checking workflow status for execution %s (current status: %s)", executionID, execution.Status)

	// Only update if the workflow is currently running or success (not already in error state)
	if execution.Status != "success" && execution.Status != "running" {
		log.Printf("⏭️  Execution %s already in final state: %s", executionID, execution.Status)
		return nil // Already marked as error or cancelled
	}

	// Count total nodes and failed nodes
	var totalNodes int64
	var failedNodes int64
	var successNodes int64

	tx.Model(&models.WorkflowNodeExecution{}).
		Where("execution_id = ?", executionID).
		Count(&totalNodes)

	tx.Model(&models.WorkflowNodeExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "error").
		Count(&failedNodes)

	tx.Model(&models.WorkflowNodeExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "success").
		Count(&successNodes)

	log.Printf("📊 Node stats for %s: total=%d, failed=%d, success=%d",
		executionID, totalNodes, failedNodes, successNodes)

	// Check if there are still pending async operations
	var pendingCount int64
	err := tx.Model(&models.OutboxMessage{}).
		Joins("JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.execution_id = ? AND outbox_messages.status IN ?",
			executionID, []string{"pending", "processing"}).
		Count(&pendingCount).Error

	if err != nil {
		log.Printf("❌ Failed to count pending messages for %s: %v", executionID, err)
		return err
	}

	log.Printf("📬 Pending outbox messages for %s: %d", executionID, pendingCount)

	// Determine the appropriate status
	var newStatus string
	var updates map[string]interface{}

	if pendingCount > 0 {
		// Still has pending operations, keep as running but note the failure
		log.Printf("⏳ Execution %s still has %d pending operations, keeping as running", executionID, pendingCount)
		return nil // Don't change status yet
	}

	// No more pending operations, determine final status
	if failedNodes > 0 && successNodes > 0 {
		newStatus = "partially_failed"
	} else if failedNodes > 0 {
		newStatus = "error"
	} else {
		log.Printf("✅ No failures found for %s, no update needed", executionID)
		return nil // No failures, should not reach here
	}

	// Update the workflow execution status
	now := time.Now()
	updates = map[string]interface{}{
		"status":       newStatus,
		"error":        fmt.Sprintf("%d out of %d nodes failed", failedNodes, totalNodes),
		"completed_at": now,
	}

	log.Printf("🔄 Updating execution %s to %s", executionID, newStatus)

	err = tx.Model(&models.WorkflowExecution{}).
		Where("id = ?", executionID).
		Updates(updates).Error

	if err != nil {
		log.Printf("❌ Failed to update execution %s: %v", executionID, err)
		return err
	}

	log.Printf("✅ Successfully updated execution %s to %s", executionID, newStatus)
	return nil
}

// CancelPendingMessagesForExecution cancels all pending/processing outbox messages for a given execution
// This should be called when a workflow execution is cancelled to prevent orphaned side effects
func (s *OutboxService) CancelPendingMessagesForExecution(executionID string) error {
	log.Printf("🛑 Cancelling outbox messages for execution %s", executionID)

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Find all pending or processing messages for this execution
		var messages []models.OutboxMessage
		err := tx.Table("outbox_messages").
			Select("outbox_messages.*").
			Joins("INNER JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
			Where("workflow_node_executions.execution_id = ?", executionID).
			Where("outbox_messages.status IN ?", []string{"pending", "processing"}).
			Find(&messages).Error

		if err != nil {
			return fmt.Errorf("failed to find outbox messages: %w", err)
		}

		if len(messages) == 0 {
			log.Printf("  ℹ️  No pending outbox messages found for execution %s", executionID)
			return nil
		}

		log.Printf("  🔄 Cancelling %d outbox message(s) for execution %s", len(messages), executionID)

		// Update all pending/processing messages to cancelled status
		now := time.Now()
		err = tx.Model(&models.OutboxMessage{}).
			Where("id IN ?", func() []string {
				ids := make([]string, len(messages))
				for i, msg := range messages {
					ids[i] = msg.ID
				}
				return ids
			}()).
			Updates(map[string]interface{}{
				"status":       "cancelled",
				"last_error":   "Workflow execution was cancelled",
				"processed_at": now,
			}).Error

		if err != nil {
			return fmt.Errorf("failed to cancel outbox messages: %w", err)
		}

		// Update corresponding node executions
		nodeExecIDs := make([]string, len(messages))
		for i, msg := range messages {
			nodeExecIDs[i] = msg.NodeExecutionID
		}

		err = tx.Model(&models.WorkflowNodeExecution{}).
			Where("id IN ?", nodeExecIDs).
			Where("status IN ?", []string{"pending", "running"}).
			Updates(map[string]interface{}{
				"status":       "cancelled",
				"error":        "Workflow execution was cancelled",
				"completed_at": now,
			}).Error

		if err != nil {
			return fmt.Errorf("failed to cancel node executions: %w", err)
		}

		log.Printf("  ✅ Successfully cancelled %d outbox message(s) for execution %s", len(messages), executionID)
		return nil
	})
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

	// Count cancelled messages
	var cancelledCount int64
	s.db.Model(&models.OutboxMessage{}).Where("status = ?", "cancelled").Count(&cancelledCount)
	result["cancelled_messages"] = int(cancelledCount)

	return result, nil
}
