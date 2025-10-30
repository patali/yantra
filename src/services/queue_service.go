package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	riverinternal "github.com/patali/yantra/src/river"
	"github.com/riverqueue/river"
)

type QueueService struct {
	riverClient *river.Client[pgx.Tx]
}

func NewQueueService(riverClient *river.Client[pgx.Tx]) *QueueService {
	return &QueueService{
		riverClient: riverClient,
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

// GetRiverClient returns the underlying River client
func (s *QueueService) GetRiverClient() *river.Client[pgx.Tx] {
	return s.riverClient
}
