package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	riverinternal "github.com/patali/yantra/internal/river"
	"github.com/riverqueue/river"
)

type QueueService struct {
	riverClient  *river.Client[pgx.Tx]
	periodicJobs map[string]string // workflowID -> periodicHandle mapping
}

func NewQueueService(riverClient *river.Client[pgx.Tx]) *QueueService {
	return &QueueService{
		riverClient:  riverClient,
		periodicJobs: make(map[string]string),
	}
}

// QueueWorkflowExecution queues a workflow for immediate execution
func (s *QueueService) QueueWorkflowExecution(ctx context.Context, workflowID, executionID string, input map[string]interface{}, triggerType string) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input: %w", err)
	}

	job, err := s.riverClient.Insert(ctx, riverinternal.WorkflowExecutionArgs{
		WorkflowID:  workflowID,
		ExecutionID: executionID,
		Input:       string(inputJSON),
		TriggerType: triggerType,
	}, nil)

	if err != nil {
		return "", fmt.Errorf("failed to insert job: %w", err)
	}

	return fmt.Sprintf("%d", job.Job.ID), nil
}

// ScheduleWorkflow schedules a workflow to run on a cron schedule
func (s *QueueService) ScheduleWorkflow(ctx context.Context, workflowID, cronExpr, timezone string) error {
	// Create periodic handle (unique identifier for this scheduled workflow)
	periodicHandle := s.getScheduleHandle(workflowID)

	// Remove existing schedule if it exists
	_ = s.UnscheduleWorkflow(ctx, workflowID)

	// Create periodic job using River's PeriodicJobs
	// Note: River's periodic jobs API may vary, this is a conceptual implementation
	// You may need to adjust based on River's actual API in version 0.26.0

	// For now, we'll use a simplified approach
	// River's PeriodicJobs are typically configured at client creation time
	// For dynamic scheduling, we might need to use InsertMany with scheduled times
	// or implement a custom scheduler

	// Store the mapping
	s.periodicJobs[workflowID] = periodicHandle

	// Log the scheduling
	fmt.Printf("📅 Scheduled workflow %s with cron '%s' (timezone: %s)\n", workflowID, cronExpr, timezone)

	// TODO: Implement actual River periodic job scheduling
	// This might require using River's PeriodicJobs API or a custom implementation
	// For now, this is a placeholder that stores the schedule information

	return nil
}

// UnscheduleWorkflow removes a scheduled workflow
func (s *QueueService) UnscheduleWorkflow(ctx context.Context, workflowID string) error {
	periodicHandle, exists := s.periodicJobs[workflowID]
	if !exists {
		// Not scheduled, nothing to do
		return nil
	}

	// TODO: Implement actual River periodic job removal
	// This might require using River's PeriodicJobs API

	// Remove from mapping
	delete(s.periodicJobs, workflowID)

	fmt.Printf("🗑️  Unscheduled workflow %s (handle: %s)\n", workflowID, periodicHandle)

	return nil
}

// GetSchedules returns all active schedules
func (s *QueueService) GetSchedules() map[string]string {
	// Return a copy of the periodic jobs mapping
	schedules := make(map[string]string)
	for k, v := range s.periodicJobs {
		schedules[k] = v
	}
	return schedules
}

// getScheduleHandle generates a unique handle for a scheduled workflow
// Similar to PG-Boss's queue naming convention
func (s *QueueService) getScheduleHandle(workflowID string) string {
	// Remove hyphens and use prefix 'wfs_'
	compactID := strings.ReplaceAll(workflowID, "-", "")
	if len(compactID) > 32 {
		compactID = compactID[:32]
	}
	return fmt.Sprintf("wfs_%s", compactID)
}

// GetRiverClient returns the underlying River client
func (s *QueueService) GetRiverClient() *river.Client[pgx.Tx] {
	return s.riverClient
}
