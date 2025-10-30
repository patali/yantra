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
func (s *CleanupService) FixStuckExecutions(ctx context.Context) error {
	log.Println("🧹 Starting cleanup: Checking for stuck workflow executions...")

	type ExecutionStats struct {
		ExecutionID     string
		TotalNodes      int64
		FailedNodes     int64
		SuccessNodes    int64
		PendingMessages int64
	}

	// Find all running executions and their stats
	var stats []ExecutionStats
	err := s.db.Raw(`
		SELECT 
			we.id as execution_id,
			COUNT(DISTINCT wne.id) as total_nodes,
			SUM(CASE WHEN wne.status = 'error' THEN 1 ELSE 0 END) as failed_nodes,
			SUM(CASE WHEN wne.status = 'success' THEN 1 ELSE 0 END) as success_nodes,
			COUNT(DISTINCT CASE WHEN om.status IN ('pending', 'processing') THEN om.id END) as pending_messages
		FROM workflow_executions we
		LEFT JOIN workflow_node_executions wne ON wne.execution_id = we.id
		LEFT JOIN outbox_messages om ON om.node_execution_id = wne.id
		WHERE we.status = 'running'
		GROUP BY we.id
	`).Scan(&stats).Error

	if err != nil {
		return fmt.Errorf("failed to query execution stats: %w", err)
	}

	if len(stats) == 0 {
		log.Println("✅ No running executions found")
		return nil
	}

	log.Printf("🔍 Found %d running executions, checking if any are stuck...", len(stats))

	fixedCount := 0
	for _, stat := range stats {
		// Skip if there are pending messages (legitimately running)
		if stat.PendingMessages > 0 {
			log.Printf("  ⏳ Execution %s has %d pending messages, skipping",
				stat.ExecutionID[:8], stat.PendingMessages)
			continue
		}

		// Determine what the status should be
		var newStatus string
		var shouldFix bool

		if stat.FailedNodes > 0 && stat.SuccessNodes > 0 {
			newStatus = "partially_failed"
			shouldFix = true
		} else if stat.FailedNodes > 0 && stat.SuccessNodes == 0 {
			newStatus = "error"
			shouldFix = true
		} else if stat.FailedNodes == 0 && stat.SuccessNodes > 0 {
			newStatus = "success"
			shouldFix = true
		}

		if shouldFix {
			log.Printf("  🔧 Fixing execution %s: %d/%d nodes failed, should be: %s",
				stat.ExecutionID[:8], stat.FailedNodes, stat.TotalNodes, newStatus)

			now := time.Now()
			updates := map[string]interface{}{
				"status":       newStatus,
				"completed_at": now,
			}

			if stat.FailedNodes > 0 {
				updates["error"] = fmt.Sprintf("%d out of %d nodes failed",
					stat.FailedNodes, stat.TotalNodes)
			}

			err := s.db.Model(&models.WorkflowExecution{}).
				Where("id = ?", stat.ExecutionID).
				Updates(updates).Error

			if err != nil {
				log.Printf("  ❌ Failed to update execution %s: %v", stat.ExecutionID[:8], err)
				continue
			}

			fixedCount++
			log.Printf("  ✅ Fixed execution %s → %s", stat.ExecutionID[:8], newStatus)
		}
	}

	if fixedCount > 0 {
		log.Printf("✅ Cleanup complete: Fixed %d stuck executions", fixedCount)
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
