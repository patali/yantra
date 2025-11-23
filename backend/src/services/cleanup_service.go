package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/patali/yantra/src/db/queries"
	"github.com/patali/yantra/src/db/repositories"
)

type CleanupService struct {
	repo repositories.Repository
}

func NewCleanupService(repo repositories.Repository) *CleanupService {
	return &CleanupService{
		repo: repo,
	}
}

// FixStuckExecutions finds and fixes workflow executions that are stuck in "running" state
// This should be called on application startup
// Uses deterministic logic based on workflow completion state rather than time-based heuristics:
// 1. Completed workflows (has "end" node execution) - marks as final status (success/error/partially_failed)
// 2. Incomplete workflows (no "end" node execution) - marks as "interrupted" for resumption
func (s *CleanupService) FixStuckExecutions(ctx context.Context) error {
	log.Println("🧹 Starting cleanup: Checking for stuck workflow executions...")

	// Find all running executions and their stats using query builder
	// Check if they have a completed "end" node to determine if they're truly finished
	// Also check for node executions stuck in "running" state (from crashes)
	executionInfos, err := queries.FindRunningExecutionsWithStats(s.repo.DB())
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
			s.repo.NodeExecution().UpdateByExecutionIDAndStatus(ctx, info.ExecutionID, "running", map[string]interface{}{
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

			err := s.repo.Execution().Update(ctx, info.ExecutionID, updates)
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
	orphanedCount, err := s.repo.Outbox().CountOrphanedMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to count orphaned messages: %w", err)
	}

	if orphanedCount > 0 {
		log.Printf("⚠️  Found %d orphaned outbox messages, marking as dead_letter", orphanedCount)

		err = s.repo.Outbox().UpdateOrphanedMessages(ctx, map[string]interface{}{
			"status":     "dead_letter",
			"last_error": "Node execution not found (orphaned message)",
		})

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
