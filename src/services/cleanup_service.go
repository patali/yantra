package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type CleanupService struct {
	db *gorm.DB
}

func NewCleanupService(db *gorm.DB) *CleanupService {
	return &CleanupService{
		db: db,
	}
}

// FixStuckExecutions finds and fixes workflow executions that are stuck in "running" state
// This should be called on application startup
// Uses deterministic logic based on workflow completion state rather than time-based heuristics:
// 1. Completed workflows (has "end" node execution) - marks as final status (success/error/partially_failed)
// 2. Incomplete workflows (no "end" node execution) - marks as "interrupted" for resumption
func (s *CleanupService) FixStuckExecutions(ctx context.Context) error {
	log.Println("🧹 Starting cleanup: Checking for stuck workflow executions...")

	type ExecutionInfo struct {
		ExecutionID        string
		WorkflowID         string
		Version            int
		TotalNodes         int64
		FailedNodes        int64
		SuccessNodes       int64
		RunningNodes       int64 // Node executions stuck in "running" state
		PendingMessages    int64
		HasEndNode         bool // Whether an "end" node executed successfully
		StartedAt          time.Time
	}

	// Find all running executions and their stats
	// Check if they have a completed "end" node to determine if they're truly finished
	// Also check for node executions stuck in "running" state (from crashes)
	var executionInfos []ExecutionInfo
	err := s.db.Raw(`
		SELECT
			we.id as execution_id,
			we.workflow_id as workflow_id,
			we.version as version,
			we.started_at as started_at,
			COUNT(DISTINCT wne.id) as total_nodes,
			SUM(CASE WHEN wne.status = 'error' THEN 1 ELSE 0 END) as failed_nodes,
			SUM(CASE WHEN wne.status = 'success' THEN 1 ELSE 0 END) as success_nodes,
			SUM(CASE WHEN wne.status = 'running' THEN 1 ELSE 0 END) as running_nodes,
			COUNT(DISTINCT CASE WHEN om.status IN ('pending', 'processing') THEN om.id END) as pending_messages,
			BOOL_OR(wne.node_type = 'end' AND wne.status = 'success') as has_end_node
		FROM workflow_executions we
		LEFT JOIN workflow_node_executions wne ON wne.execution_id = we.id
		LEFT JOIN outbox_messages om ON om.node_execution_id = wne.id
		WHERE we.status = 'running'
		GROUP BY we.id, we.workflow_id, we.version, we.started_at
	`).Scan(&executionInfos).Error

	if err != nil {
		return fmt.Errorf("failed to query execution stats: %w", err)
	}

	if len(executionInfos) == 0 {
		log.Println("✅ No running executions found")
		return nil
	}

	log.Printf("🔍 Found %d running executions, checking if any are stuck...", len(executionInfos))

	fixedCount := 0
	interruptedCount := 0

	for _, info := range executionInfos {
		// Skip if there are pending messages (legitimately running)
		if info.PendingMessages > 0 {
			log.Printf("  ⏳ Execution %s has %d pending messages, skipping",
				info.ExecutionID[:8], info.PendingMessages)
			continue
		}

		// Determine status based on workflow completion state
		var newStatus string
		var errorMsg string
		var shouldFix bool

		// Priority check: If there are node executions stuck in "running" state,
		// this is a clear sign of a crash - mark as interrupted regardless of other state
		if info.RunningNodes > 0 {
			log.Printf("  💥 Execution %s has %d node(s) stuck in 'running' state (server crashed)",
				info.ExecutionID[:8], info.RunningNodes)

			// First, mark stuck node executions as error so they're completed
			s.db.Model(&models.WorkflowNodeExecution{}).
				Where("execution_id = ? AND status = ?", info.ExecutionID, "running").
				Updates(map[string]interface{}{
					"status":       "error",
					"error":        "Node execution interrupted by server crash/restart - workflow can be resumed",
					"completed_at": time.Now(),
				})

			// Mark workflow as interrupted
			newStatus = "interrupted"
			errorMsg = fmt.Sprintf("Workflow interrupted by server crash with %d node(s) stuck in running state (detected on server restart) - can be resumed from checkpoint", info.RunningNodes)
			shouldFix = true
			interruptedCount++
		} else if info.HasEndNode {
			// Case 1: Workflow reached the "end" node - it's complete
			// Workflow completed, determine final status based on node failures
			if info.FailedNodes > 0 && info.SuccessNodes > 0 {
				newStatus = "partially_failed"
				errorMsg = fmt.Sprintf("%d out of %d nodes failed", info.FailedNodes, info.TotalNodes)
			} else if info.FailedNodes > 0 {
				newStatus = "error"
				errorMsg = fmt.Sprintf("%d out of %d nodes failed", info.FailedNodes, info.TotalNodes)
			} else {
				newStatus = "success"
			}
			shouldFix = true
			log.Printf("  ✓ Execution %s reached end node (completed)", info.ExecutionID[:8])
		} else {
			// Case 2: Workflow did NOT reach "end" node - it's interrupted
			// No pending messages and no stuck nodes means it's not actively running
			newStatus = "interrupted"

			if info.TotalNodes == 0 {
				// No nodes executed at all
				errorMsg = "Workflow interrupted before any nodes executed (detected on server restart) - can be resumed"
			} else if info.FailedNodes > 0 {
				// Some nodes failed, workflow stopped
				errorMsg = fmt.Sprintf("Workflow interrupted after %d node failures (detected on server restart) - can be resumed from checkpoint", info.FailedNodes)
			} else {
				// Nodes executed successfully but didn't reach end
				errorMsg = fmt.Sprintf("Workflow interrupted mid-execution with %d/%d nodes completed (detected on server restart) - can be resumed from checkpoint", info.SuccessNodes, info.TotalNodes)
			}
			shouldFix = true
			interruptedCount++
			log.Printf("  ⏸️  Execution %s did not reach end node (interrupted)", info.ExecutionID[:8])
		}

		if shouldFix {
			log.Printf("  🔧 Fixing execution %s: %d success, %d failed, end_reached=%v → %s",
				info.ExecutionID[:8], info.SuccessNodes, info.FailedNodes, info.HasEndNode, newStatus)

			updates := map[string]interface{}{
				"status": newStatus,
			}

			// Only set completed_at for final statuses (not for "interrupted")
			if newStatus != "interrupted" {
				updates["completed_at"] = time.Now()
			}

			// Set error message if any
			if errorMsg != "" {
				updates["error"] = errorMsg
			}

			err := s.db.Model(&models.WorkflowExecution{}).
				Where("id = ?", info.ExecutionID).
				Updates(updates).Error

			if err != nil {
				log.Printf("  ❌ Failed to update execution %s: %v", info.ExecutionID[:8], err)
				continue
			}

			fixedCount++
			log.Printf("  ✅ Fixed execution %s → %s", info.ExecutionID[:8], newStatus)
		}
	}

	if fixedCount > 0 {
		log.Printf("✅ Cleanup complete: Fixed %d stuck executions (%d marked as interrupted for resumption)",
			fixedCount, interruptedCount)
	} else {
		log.Println("✅ Cleanup complete: No stuck executions found")
	}

	return nil
}

// FixOrphanedOutboxMessages finds outbox messages whose node executions or workflows don't exist
func (s *CleanupService) FixOrphanedOutboxMessages(ctx context.Context) error {
	log.Println("🧹 Starting cleanup: Checking for orphaned outbox messages...")

	// Find messages with non-existent node executions
	var orphanedCount int64
	err := s.db.Model(&models.OutboxMessage{}).
		Joins("LEFT JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.id IS NULL").
		Count(&orphanedCount).Error

	if err != nil {
		return fmt.Errorf("failed to count orphaned messages: %w", err)
	}

	if orphanedCount > 0 {
		log.Printf("⚠️  Found %d orphaned outbox messages, marking as dead_letter", orphanedCount)

		err = s.db.Model(&models.OutboxMessage{}).
			Joins("LEFT JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
			Where("workflow_node_executions.id IS NULL AND outbox_messages.status NOT IN ?",
				[]string{"dead_letter", "completed"}).
			Updates(map[string]interface{}{
				"status":     "dead_letter",
				"last_error": "Node execution not found (orphaned message)",
			}).Error

		if err != nil {
			return fmt.Errorf("failed to update orphaned messages: %w", err)
		}

		log.Printf("✅ Marked %d orphaned messages as dead_letter", orphanedCount)
	} else {
		log.Println("✅ No orphaned outbox messages found")
	}

	return nil
}

// RunAllCleanups runs all cleanup routines
func (s *CleanupService) RunAllCleanups(ctx context.Context) error {
	log.Println("🧹 Starting all cleanup routines...")

	if err := s.FixStuckExecutions(ctx); err != nil {
		log.Printf("⚠️  Error in FixStuckExecutions: %v", err)
	}

	if err := s.FixOrphanedOutboxMessages(ctx); err != nil {
		log.Printf("⚠️  Error in FixOrphanedOutboxMessages: %v", err)
	}

	log.Println("🧹 All cleanup routines completed")
	return nil
}
