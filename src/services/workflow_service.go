package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/db/repositories"
	"github.com/patali/yantra/src/dto"
	"github.com/patali/yantra/src/executors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type WorkflowService struct {
	db               *gorm.DB // Keep for backward compatibility
	repo             repositories.Repository
	queueService     *QueueService
	schedulerService *SchedulerService
}

func NewWorkflowService(db *gorm.DB, queueService *QueueService) *WorkflowService {
	return &WorkflowService{
		db:           db,
		repo:         repositories.NewRepository(db),
		queueService: queueService,
	}
}

// SetScheduler sets the scheduler service (called after both services are initialized)
func (s *WorkflowService) SetScheduler(scheduler *SchedulerService) {
	s.schedulerService = scheduler
}

// GetAllWorkflows retrieves all workflows for an account
func (s *WorkflowService) GetAllWorkflows(accountID string) ([]dto.WorkflowResponse, error) {
	ctx := context.Background()

	workflows, err := s.repo.Workflow().FindByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	responses := make([]dto.WorkflowResponse, len(workflows))
	for i, w := range workflows {
		// Get creator details
		var creatorDetails *dto.WorkflowCreator
		if creator, err := s.repo.User().FindByID(ctx, w.CreatedBy); err == nil {
			creatorDetails = &dto.WorkflowCreator{
				Username: creator.Username,
				Email:    creator.Email,
			}
		}

		// Count executions for this workflow
		executionCount, _ := s.repo.Workflow().CountExecutions(ctx, w.ID)

		// Count versions for this workflow
		versionCount, _ := s.repo.WorkflowVersion().CountByWorkflowID(ctx, w.ID)

		responses[i] = dto.WorkflowResponse{
			ID:                 w.ID,
			Name:               w.Name,
			Description:        w.Description,
			IsActive:           w.IsActive,
			Schedule:           w.Schedule,
			Timezone:           w.Timezone,
			WebhookPath:        w.WebhookPath,
			WebhookRequireAuth: w.WebhookRequireAuth,
			CurrentVersion:     w.CurrentVersion,
			CreatedBy:          w.CreatedBy,
			Creator:            creatorDetails,
			Count: &dto.WorkflowCount{
				Executions: int(executionCount),
				Versions:   int(versionCount),
			},
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
		}
	}

	return responses, nil
}

// GetWorkflowById retrieves a workflow by ID
func (s *WorkflowService) GetWorkflowById(id string) (*models.Workflow, error) {
	ctx := context.Background()
	return s.repo.Workflow().FindByID(ctx, id)
}

// GetWorkflowByIdAndAccount returns a workflow by ID and account ID
func (s *WorkflowService) GetWorkflowByIdAndAccount(id, accountID string) (*models.Workflow, error) {
	ctx := context.Background()

	workflow, err := s.repo.Workflow().FindByIDAndAccount(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	// Load the versions for this workflow
	versions, err := s.GetVersionHistory(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow versions: %w", err)
	}

	// Add versions to the workflow struct
	workflow.Versions = versions

	return workflow, nil
}

// validateWebhookPath validates and sanitizes webhook path input
func validateWebhookPath(path *string) (*string, error) {
	if path == nil || *path == "" {
		return nil, nil
	}

	webhookPath := strings.TrimSpace(*path)

	// Check for path traversal attempts
	if strings.Contains(webhookPath, "..") || strings.Contains(webhookPath, "/") || strings.Contains(webhookPath, "\\") {
		return nil, fmt.Errorf("webhook path contains invalid characters")
	}

	// Limit path length to prevent abuse
	if len(webhookPath) > 100 {
		return nil, fmt.Errorf("webhook path exceeds maximum length of 100 characters")
	}

	return &webhookPath, nil
}

// validateWorkflowDefinition validates a workflow definition structure
func validateWorkflowDefinition(definition map[string]interface{}) error {
	// Check for nodes
	nodesInterface, ok := definition["nodes"]
	if !ok {
		return fmt.Errorf("workflow definition must contain 'nodes' field")
	}

	nodes, ok := nodesInterface.([]interface{})
	if !ok {
		return fmt.Errorf("'nodes' field must be an array")
	}

	if len(nodes) == 0 {
		return fmt.Errorf("workflow must contain at least one node")
	}

	startCount := 0
	endCount := 0

	// Validate each node
	for i, nodeInterface := range nodes {
		node, ok := nodeInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("node at index %d is not a valid object", i)
		}

		// Check node ID
		nodeID, ok := node["id"].(string)
		if !ok || nodeID == "" {
			return fmt.Errorf("node at index %d missing or invalid 'id' field", i)
		}

		// Check node type
		nodeType, ok := node["type"].(string)
		if !ok || nodeType == "" {
			return fmt.Errorf("node '%s' missing or invalid 'type' field", nodeID)
		}

		// Validate node type is supported
		if !executors.IsValidNodeType(nodeType) {
			return fmt.Errorf("node '%s' has unsupported type '%s'", nodeID, nodeType)
		}

		// Count start and end nodes
		if nodeType == executors.NodeTypeStart {
			startCount++
		}
		if nodeType == executors.NodeTypeEnd {
			endCount++
		}
	}

	// Validate start node count
	if startCount != 1 {
		return fmt.Errorf("workflow must have exactly one start node, found %d", startCount)
	}

	// Validate end node count
	if endCount < 1 {
		return fmt.Errorf("workflow must have at least one end node, found %d", endCount)
	}

	return nil
}

// CreateWorkflow creates a new workflow with optional scheduling
func (s *WorkflowService) CreateWorkflow(ctx context.Context, req dto.CreateWorkflowRequest, createdBy, accountID string) (*models.Workflow, error) {
	timezone := "UTC"
	if req.Timezone != nil {
		timezone = *req.Timezone
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// SECURITY: Validate webhook path if provided
	validatedWebhookPath, err := validateWebhookPath(req.WebhookPath)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook path: %w", err)
	}

	// Validate workflow definition structure
	if err := validateWorkflowDefinition(req.Definition); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	definitionJSON, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal definition: %w", err)
	}

	var workflow models.Workflow

	// Create workflow and version in a transaction
	err = s.repo.Transaction(ctx, func(txRepo repositories.TxRepository) error {
		// Create workflow
		workflow = models.Workflow{
			Name:               req.Name,
			Description:        req.Description,
			IsActive:           isActive,
			Schedule:           req.Schedule,
			Timezone:           timezone,
			WebhookPath:        validatedWebhookPath,
			WebhookRequireAuth: req.WebhookRequireAuth != nil && *req.WebhookRequireAuth,
			CurrentVersion:     1,
			AccountID:          &accountID,
			CreatedBy:          createdBy,
		}

		if err := txRepo.Workflow().Create(ctx, &workflow); err != nil {
			return err
		}

		// Create first version
		version := models.WorkflowVersion{
			WorkflowID: workflow.ID,
			Version:    1,
			Definition: string(definitionJSON),
			ChangeLog:  stringPtr("Initial version"),
		}

		if err := txRepo.WorkflowVersion().Create(ctx, &version); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// If scheduling is needed, schedule with robfig/cron (outside transaction)
	if req.Schedule != nil && isActive && s.schedulerService != nil {
		if err := s.schedulerService.AddSchedule(workflow.ID, *req.Schedule, timezone); err != nil {
			// Log error but don't fail the workflow creation
			// This provides eventual consistency similar to the Node version
			fmt.Printf("⚠️  Failed to schedule workflow %s: %v\n", workflow.ID, err)
		}
	}

	return &workflow, nil
}

// DuplicateWorkflow creates a copy of an existing workflow
func (s *WorkflowService) DuplicateWorkflow(ctx context.Context, id, userID, accountID string) (*models.Workflow, error) {
	// Get the original workflow with its latest version
	originalWorkflow, err := s.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Get the latest version definition
	latestVersion, err := s.repo.WorkflowVersion().FindLatestByWorkflowID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow version: %w", err)
	}

	// Parse the definition
	var definition map[string]interface{}
	if err := json.Unmarshal([]byte(latestVersion.Definition), &definition); err != nil {
		return nil, fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// Create a new workflow with "Copy" suffix
	copyName := originalWorkflow.Name + " (Copy)"

	// Create the duplicate workflow (inactive by default, no schedule)
	createReq := dto.CreateWorkflowRequest{
		Name:        copyName,
		Description: originalWorkflow.Description,
		Definition:  definition,
		IsActive:    boolPtr(false), // Start as inactive
		Schedule:    nil,            // Don't copy schedule
		Timezone:    &originalWorkflow.Timezone,
	}

	return s.CreateWorkflow(ctx, createReq, userID, accountID)
}

// UpdateWorkflow updates a workflow
func (s *WorkflowService) UpdateWorkflow(id string, req dto.UpdateWorkflowRequest) (*models.Workflow, error) {
	ctx := context.Background()

	// Check if workflow exists
	_, err := s.repo.Workflow().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate workflow definition if provided
	if req.Definition != nil {
		if err := validateWorkflowDefinition(req.Definition); err != nil {
			return nil, fmt.Errorf("invalid workflow definition: %w", err)
		}
	}

	// Update fields
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
	}

	if req.Definition != nil {
		updates["definition"] = req.Definition
	}

	if err := s.repo.Workflow().Update(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Reload workflow
	return s.repo.Workflow().FindByID(ctx, id)
}

// UpdateWorkflowByAccount updates a workflow by ID and account ID
func (s *WorkflowService) UpdateWorkflowByAccount(id, accountID string, req dto.UpdateWorkflowRequest) (*models.Workflow, error) {
	ctx := context.Background()

	workflow, err := s.repo.Workflow().FindByIDAndAccount(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	newVersion := workflow.CurrentVersion

	// If definition is updated, create a new version
	if req.Definition != nil {
		// Validate workflow definition structure
		if err := validateWorkflowDefinition(req.Definition); err != nil {
			return nil, fmt.Errorf("invalid workflow definition: %w", err)
		}

		newVersion = workflow.CurrentVersion + 1
		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal definition: %w", err)
		}

		changeLog := fmt.Sprintf("Version %d", newVersion)
		if req.ChangeLog != nil {
			changeLog = *req.ChangeLog
		}

		version := models.WorkflowVersion{
			WorkflowID: id,
			Version:    newVersion,
			Definition: string(definitionJSON),
			ChangeLog:  &changeLog,
		}

		if err := s.repo.WorkflowVersion().Create(ctx, &version); err != nil {
			return nil, fmt.Errorf("failed to create version: %w", err)
		}
	}

	// Update workflow
	updates := map[string]interface{}{
		"current_version": newVersion,
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Schedule != nil {
		updates["schedule"] = req.Schedule
	}
	if req.Timezone != nil {
		updates["timezone"] = *req.Timezone
	}
	if req.WebhookPath != nil {
		// SECURITY: Validate webhook path before updating
		validatedWebhookPath, err := validateWebhookPath(req.WebhookPath)
		if err != nil {
			return nil, fmt.Errorf("invalid webhook path: %w", err)
		}
		updates["webhook_path"] = validatedWebhookPath
	}
	if req.WebhookRequireAuth != nil {
		updates["webhook_require_auth"] = *req.WebhookRequireAuth
	}

	if err := s.repo.Workflow().Update(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Update scheduler if schedule changed
	if req.Schedule != nil && s.schedulerService != nil {
		workflow, _ := s.repo.Workflow().FindByID(ctx, id)
		if workflow != nil && workflow.IsActive && req.Schedule != nil && *req.Schedule != "" {
			if err := s.schedulerService.AddSchedule(id, *req.Schedule, workflow.Timezone); err != nil {
				log.Printf("⚠️  Failed to update schedule for workflow %s: %v\n", id, err)
			}
		} else if workflow != nil && (!workflow.IsActive || (req.Schedule != nil && *req.Schedule == "")) {
			// Remove schedule if workflow is inactive or schedule is cleared
			s.schedulerService.RemoveSchedule(id)
		}
	}

	// Reload workflow
	return s.repo.Workflow().FindByID(ctx, id)
}

// UpdateSchedule updates the workflow schedule
func (s *WorkflowService) UpdateSchedule(ctx context.Context, id string, req dto.UpdateScheduleRequest) error {
	workflow, err := s.repo.Workflow().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Determine timezone (use provided or keep existing)
	timezone := workflow.Timezone
	if req.Timezone != nil {
		timezone = *req.Timezone
	}

	// Determine if active (use provided or keep existing)
	isActive := workflow.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Determine schedule
	// If isActive is being set to false, we should clear/ignore the schedule
	// If isActive is true and schedule is provided, use it
	// Otherwise keep existing
	var schedule *string
	if !isActive {
		// When deactivating, clear the schedule
		schedule = nil
	} else if req.Schedule != nil && *req.Schedule != "" {
		// Active and schedule provided
		schedule = req.Schedule
	} else if req.Schedule == nil {
		// No schedule in request, keep existing
		schedule = workflow.Schedule
	} else {
		// Empty string provided, clear it
		schedule = nil
	}

	fmt.Printf("🔍 UpdateSchedule called for workflow %s: isActive=%v, schedule=%v, req.Schedule=%v\n", id, isActive, schedule, req.Schedule)

	// Update workflow schedule in database
	updates := map[string]interface{}{
		"schedule":  schedule,
		"timezone":  timezone,
		"is_active": isActive,
	}

	if err := s.repo.Workflow().Update(ctx, id, updates); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Update scheduler (using robfig/cron via SchedulerService)
	if s.schedulerService == nil {
		return fmt.Errorf("scheduler service not initialized")
	}

	if isActive && schedule != nil && *schedule != "" {
		// Schedule or update the workflow schedule
		fmt.Printf("📅 Scheduling workflow %s with schedule: %s\n", id, *schedule)
		if err := s.schedulerService.UpdateSchedule(id, *schedule, timezone); err != nil {
			return fmt.Errorf("failed to schedule workflow: %w", err)
		}
	} else {
		// Unschedule the workflow (either inactive or no schedule)
		fmt.Printf("🗑️  Unscheduling workflow %s (isActive: %v, schedule: %v)\n", id, isActive, schedule)
		if err := s.schedulerService.RemoveSchedule(id); err != nil {
			return fmt.Errorf("failed to unschedule workflow: %w", err)
		}
	}

	return nil
}

// DeleteWorkflow deletes a workflow and unschedules it
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, id string) error {
	workflow, err := s.repo.Workflow().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Remove from scheduler if scheduled
	if workflow.Schedule != nil {
		// Note: Scheduler cleanup would be handled by the scheduler service
	}

	// Delete workflow (cascade will handle related records)
	return s.repo.Workflow().Delete(ctx, id)
}

// DeleteWorkflowByAccount deletes a workflow by ID and account ID
func (s *WorkflowService) DeleteWorkflowByAccount(ctx context.Context, id, accountID string) error {
	workflow, err := s.repo.Workflow().FindByIDAndAccount(ctx, id, accountID)
	if err != nil {
		return err
	}

	// Unschedule if scheduled
	if workflow.Schedule != nil && workflow.IsActive && s.schedulerService != nil {
		_ = s.schedulerService.RemoveSchedule(id) // Ignore error
	}

	// Delete workflow (cascade will delete versions, executions, etc.)
	return s.repo.Workflow().Delete(ctx, id)
}

// ExecuteWorkflow queues a workflow for execution
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, id string, input map[string]interface{}) (jobID string, executionID string, err error) {
	return s.ExecuteWorkflowWithTrigger(ctx, id, input, models.TriggerTypeManual)
}

// ExecuteWorkflowWithTrigger queues a workflow for execution with a specific trigger type
func (s *WorkflowService) ExecuteWorkflowWithTrigger(ctx context.Context, id string, input map[string]interface{}, triggerType string) (jobID string, executionID string, err error) {
	// SECURITY: Validate workflow ID format (must be valid UUID)
	// This provides defense in depth even though route params should already be validated
	if _, err := uuid.Parse(id); err != nil {
		return "", "", fmt.Errorf("invalid workflow identifier: %w", err)
	}

	// Check if workflow exists
	_, err = s.repo.Workflow().FindByID(ctx, id)
	if err != nil {
		return "", "", err
	}

	// Get latest version
	latestVersion, err := s.repo.WorkflowVersion().FindLatestByWorkflowID(ctx, id)
	if err != nil {
		return "", "", err
	}

	// SECURITY: Validate input size before creating execution record
	// This prevents DoS attacks via large payloads
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", "", fmt.Errorf("failed to serialize input: %w", err)
	}

	// MaxDataSize is 10MB as defined in workflow_engine.go
	const MaxDataSize = 10 * 1024 * 1024
	if len(inputJSON) > MaxDataSize {
		return "", "", fmt.Errorf("input size (%d bytes) exceeds maximum allowed (%d bytes)", len(inputJSON), MaxDataSize)
	}

	inputStr := string(inputJSON)

	execution := models.WorkflowExecution{
		WorkflowID:  id,
		Version:     latestVersion.Version,
		Status:      "queued",
		TriggerType: triggerType,
	}
	if len(inputStr) > 0 && inputStr != "null" {
		execution.Input = &inputStr
	}

	if err := s.repo.Execution().Create(ctx, &execution); err != nil {
		return "", "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// Queue execution with the execution ID
	jobID, err = s.queueService.QueueWorkflowExecution(ctx, id, execution.ID, input, triggerType)
	if err != nil {
		// Rollback: mark execution as failed
		s.repo.Execution().Update(ctx, execution.ID, map[string]interface{}{
			"status": "error",
			"error":  "Failed to queue for execution",
		})
		return "", "", fmt.Errorf("failed to queue workflow execution: %w", err)
	}

	return jobID, execution.ID, nil
}

// ResumeWorkflow resumes a failed or interrupted workflow execution from checkpoint
func (s *WorkflowService) ResumeWorkflow(ctx context.Context, executionID string) (jobID string, err error) {
	// Get the execution record
	execution, err := s.repo.Execution().FindByID(ctx, executionID)
	if err != nil {
		return "", err
	}

	// Verify execution can be resumed (must be in error, running, or interrupted state)
	if execution.Status != "error" && execution.Status != "running" && execution.Status != "interrupted" {
		return "", fmt.Errorf("cannot resume execution with status: %s (must be 'error', 'running', or 'interrupted')", execution.Status)
	}

	// Get the workflow
	_, err = s.repo.Workflow().FindByID(ctx, execution.WorkflowID)
	if err != nil {
		return "", err
	}

	// Parse input from original execution
	var input map[string]interface{}
	if execution.Input != nil {
		if err := json.Unmarshal([]byte(*execution.Input), &input); err != nil {
			return "", fmt.Errorf("failed to parse execution input: %w", err)
		}
	}

	// Re-queue the workflow execution with the same execution ID
	// The workflow engine will detect already-executed nodes and skip them
	jobID, err = s.queueService.QueueWorkflowExecution(ctx, execution.WorkflowID, execution.ID, input, models.TriggerTypeResume)
	if err != nil {
		return "", fmt.Errorf("failed to queue workflow resumption: %w", err)
	}

	log.Printf("🔄 Workflow execution queued for resumption: execution_id=%s, job_id=%s", executionID, jobID)
	return jobID, nil
}

// GetVersionHistory retrieves version history for a workflow
func (s *WorkflowService) GetVersionHistory(id string) ([]models.WorkflowVersion, error) {
	ctx := context.Background()
	return s.repo.WorkflowVersion().FindByWorkflowID(ctx, id)
}

// GetWorkflowExecutions returns all executions for a workflow
func (s *WorkflowService) GetWorkflowExecutions(id string) ([]models.WorkflowExecution, error) {
	ctx := context.Background()
	return s.repo.Execution().FindByWorkflowID(ctx, id)
}

// GetWorkflowExecutionById returns a specific execution with node executions
func (s *WorkflowService) GetWorkflowExecutionById(executionId string) (*dto.ExecutionResponse, error) {
	ctx := context.Background()

	execution, err := s.repo.Execution().FindByID(ctx, executionId)
	if err != nil {
		return nil, err
	}

	// Get all node executions (including retries/failures)
	// Ordered by started_at DESC so most recent attempts appear first
	nodeExecutions, _ := s.repo.NodeExecution().FindByExecutionID(ctx, executionId)

	// Convert node executions to response format
	nodeExecResponses := make([]dto.NodeExecutionResponse, len(nodeExecutions))
	for i, ne := range nodeExecutions {
		nodeExecResponses[i] = dto.NodeExecutionResponse{
			ID:               ne.ID,
			ExecutionID:      ne.ExecutionID,
			NodeID:           ne.NodeID,
			NodeType:         ne.NodeType,
			Status:           ne.Status,
			Input:            ne.Input,
			Output:           ne.Output,
			Error:            ne.Error,
			ParentLoopNodeID: ne.ParentLoopNodeID,
			StartedAt:        &ne.StartedAt,
			CompletedAt:      ne.CompletedAt,
		}
	}

	// Build response
	response := &dto.ExecutionResponse{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID,
		Version:        execution.Version,
		Status:         execution.Status,
		TriggerType:    execution.TriggerType,
		Input:          execution.Input,
		Output:         execution.Output,
		Error:          execution.Error,
		StartedAt:      &execution.StartedAt,
		CompletedAt:    execution.CompletedAt,
		NodeExecutions: nodeExecResponses,
	}

	return response, nil
}

// GetAllWorkflowExecutions returns all workflow executions with optional filtering
func (s *WorkflowService) GetAllWorkflowExecutions(limit int, status string) ([]dto.ExecutionResponse, error) {
	ctx := context.Background()

	executions, err := s.repo.Execution().FindAll(ctx, limit, status)
	if err != nil {
		return nil, err
	}

	return s.convertExecutionsToResponses(executions), nil
}

// GetFailedWorkflowExecutions returns all failed and partially failed workflow executions
func (s *WorkflowService) GetFailedWorkflowExecutions(limit int) ([]dto.ExecutionResponse, error) {
	ctx := context.Background()

	executions, err := s.repo.Execution().FindFailed(ctx, limit)
	if err != nil {
		return nil, err
	}

	return s.convertExecutionsToResponses(executions), nil
}

// SECURITY: Account-filtered versions of the above methods

// GetAllWorkflowExecutionsByAccount returns all executions filtered by account ID
func (s *WorkflowService) GetAllWorkflowExecutionsByAccount(accountID string, limit int, status string) ([]dto.ExecutionResponse, error) {
	ctx := context.Background()

	executions, err := s.repo.Execution().FindAllByAccountID(ctx, accountID, limit, status)
	if err != nil {
		return nil, err
	}

	return s.convertExecutionsToResponses(executions), nil
}

// GetFailedWorkflowExecutionsByAccount returns failed executions filtered by account ID
func (s *WorkflowService) GetFailedWorkflowExecutionsByAccount(accountID string, limit int) ([]dto.ExecutionResponse, error) {
	ctx := context.Background()

	executions, err := s.repo.Execution().FindFailedByAccountID(ctx, accountID, limit)
	if err != nil {
		return nil, err
	}

	return s.convertExecutionsToResponses(executions), nil
}

// convertExecutionsToResponses converts execution models to response DTOs
func (s *WorkflowService) convertExecutionsToResponses(executions []models.WorkflowExecution) []dto.ExecutionResponse {
	ctx := context.Background()

	responses := make([]dto.ExecutionResponse, len(executions))
	for i, exec := range executions {
		// Get node executions
		nodeExecutions, _ := s.repo.NodeExecution().FindByExecutionID(ctx, exec.ID)

		// Convert node executions
		nodeExecResponses := make([]dto.NodeExecutionResponse, len(nodeExecutions))
		for j, ne := range nodeExecutions {
			nodeExecResponses[j] = dto.NodeExecutionResponse{
				ID:          ne.ID,
				ExecutionID: ne.ExecutionID,
				NodeID:      ne.NodeID,
				NodeType:    ne.NodeType,
				Status:      ne.Status,
				Input:       ne.Input,
				Output:      ne.Output,
				Error:       ne.Error,
				StartedAt:   &ne.StartedAt,
				CompletedAt: ne.CompletedAt,
			}
		}

		// Get workflow details
		var workflowResp *dto.WorkflowResponse
		if workflow, err := s.repo.Workflow().FindByID(ctx, exec.WorkflowID); err == nil {
			workflowResp = &dto.WorkflowResponse{
				ID:   workflow.ID,
				Name: workflow.Name,
			}
		}

		responses[i] = dto.ExecutionResponse{
			ID:             exec.ID,
			WorkflowID:     exec.WorkflowID,
			Workflow:       workflowResp,
			Version:        exec.Version,
			Status:         exec.Status,
			TriggerType:    exec.TriggerType,
			Input:          exec.Input,
			Output:         exec.Output,
			Error:          exec.Error,
			StartedAt:      &exec.StartedAt,
			CompletedAt:    exec.CompletedAt,
			NodeExecutions: nodeExecResponses,
		}
	}

	return responses
}

// CancelWorkflowExecution cancels a running or queued workflow execution
func (s *WorkflowService) CancelWorkflowExecution(executionID string) error {
	ctx := context.Background()

	execution, err := s.repo.Execution().FindByID(ctx, executionID)
	if err != nil {
		return err
	}

	// Only running or queued executions can be cancelled
	if execution.Status != "running" && execution.Status != "queued" {
		return fmt.Errorf("only running or queued executions can be cancelled")
	}

	// Update execution status to cancelled
	now := time.Now()
	return s.repo.Execution().Update(ctx, executionID, map[string]interface{}{
		"status":       "cancelled",
		"completed_at": now,
		"error":        "Execution cancelled by user",
	})
}

// GenerateWebhookSecret generates a new webhook secret and returns both the plain secret and its hash
// The plain secret should be shown to the user once, then only the hash is stored
func (s *WorkflowService) GenerateWebhookSecret() (plainSecret string, secretHash string, err error) {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	// Encode as hex string (64 characters)
	plainSecret = hex.EncodeToString(bytes)

	// Hash the secret using bcrypt
	secretHashBytes, err := bcrypt.GenerateFromPassword([]byte(plainSecret), 10)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash secret: %w", err)
	}

	secretHash = string(secretHashBytes)
	return plainSecret, secretHash, nil
}

// ValidateWebhookSecret compares a provided secret against the stored hash
func (s *WorkflowService) ValidateWebhookSecret(providedSecret string, secretHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(providedSecret))
	return err == nil
}

// RegenerateWebhookSecret generates a new secret for a workflow
func (s *WorkflowService) RegenerateWebhookSecret(ctx context.Context, workflowID string) (plainSecret string, err error) {
	// Check if workflow exists
	_, err = s.repo.Workflow().FindByID(ctx, workflowID)
	if err != nil {
		return "", err
	}

	// Generate new secret
	plainSecret, secretHash, err := s.GenerateWebhookSecret()
	if err != nil {
		return "", err
	}

	// Update workflow with new secret hash
	updates := map[string]interface{}{
		"webhook_secret_hash": secretHash,
	}

	if err := s.repo.Workflow().Update(ctx, workflowID, updates); err != nil {
		return "", fmt.Errorf("failed to update webhook secret: %w", err)
	}

	return plainSecret, nil
}

// RestoreWorkflowVersion restores a workflow to a previous version
func (s *WorkflowService) RestoreWorkflowVersion(id string, version int) error {
	ctx := context.Background()

	// Get the version to ensure it exists
	_, err := s.repo.WorkflowVersion().FindByWorkflowIDAndVersion(ctx, id, version)
	if err != nil {
		return err
	}

	// Update the workflow with the version
	return s.repo.Workflow().Update(ctx, id, map[string]interface{}{
		"current_version": version,
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
