package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patali/yantra/internal/models"
	"gorm.io/gorm"
)

type WorkflowService struct {
	db               *gorm.DB
	queueService     *QueueService
	schedulerService *SchedulerService
}

func NewWorkflowService(db *gorm.DB, queueService *QueueService) *WorkflowService {
	return &WorkflowService{
		db:           db,
		queueService: queueService,
	}
}

// SetScheduler sets the scheduler service (called after both services are initialized)
func (s *WorkflowService) SetScheduler(scheduler *SchedulerService) {
	s.schedulerService = scheduler
}

type CreateWorkflowRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition" binding:"required"`
	Schedule    *string                `json:"schedule"`
	Timezone    *string                `json:"timezone"`
	IsActive    *bool                  `json:"is_active"`
}

type UpdateWorkflowRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	ChangeLog   *string                `json:"change_log"`
}

type UpdateScheduleRequest struct {
	Schedule *string `json:"schedule"` // Can be null to clear, omitted to keep existing
	Timezone *string `json:"timezone"`
	IsActive *bool   `json:"isActive"` // Use camelCase to match frontend
}

type ExecuteWorkflowRequest struct {
	Input map[string]interface{} `json:"input"`
}

type WorkflowCreator struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type WorkflowCount struct {
	Executions int `json:"executions"`
	Versions   int `json:"versions"`
}

type WorkflowResponse struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Description    *string          `json:"description,omitempty"`
	IsActive       bool             `json:"isActive"`
	Schedule       *string          `json:"schedule,omitempty"`
	Timezone       string           `json:"timezone"`
	CurrentVersion int              `json:"currentVersion"`
	CreatedBy      string           `json:"createdBy"` // Creator user ID
	Creator        *WorkflowCreator `json:"creator"`   // Creator details
	Count          *WorkflowCount   `json:"_count"`    // Counts
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

type NodeExecutionResponse struct {
	ID          string     `json:"id"`
	ExecutionID string     `json:"executionId"`
	NodeID      string     `json:"nodeId"`
	NodeType    string     `json:"nodeType"`
	Status      string     `json:"status"`
	Input       *string    `json:"input,omitempty"`
	Output      *string    `json:"output,omitempty"`
	Error       *string    `json:"error,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type ExecutionResponse struct {
	ID             string                  `json:"id"`
	WorkflowID     string                  `json:"workflowId"`
	Workflow       *WorkflowResponse       `json:"workflow,omitempty"`
	Version        int                     `json:"version"`
	Status         string                  `json:"status"`
	TriggerType    string                  `json:"triggerType"`
	Input          *string                 `json:"input,omitempty"`
	Output         *string                 `json:"output,omitempty"`
	Error          *string                 `json:"error,omitempty"`
	StartedAt      *time.Time              `json:"startedAt,omitempty"`
	CompletedAt    *time.Time              `json:"completedAt,omitempty"`
	NodeExecutions []NodeExecutionResponse `json:"nodeExecutions"`
}

// GetAllWorkflows retrieves all workflows for an account
func (s *WorkflowService) GetAllWorkflows(accountID string) ([]WorkflowResponse, error) {
	var workflows []models.Workflow
	err := s.db.Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&workflows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	responses := make([]WorkflowResponse, len(workflows))
	for i, w := range workflows {
		// Get creator details
		var creator models.User
		var creatorDetails *WorkflowCreator
		if err := s.db.Select("username", "email").First(&creator, "id = ?", w.CreatedBy).Error; err == nil {
			creatorDetails = &WorkflowCreator{
				Username: creator.Username,
				Email:    creator.Email,
			}
		}

		// Count executions for this workflow
		var executionCount int64
		s.db.Model(&models.WorkflowExecution{}).Where("workflow_id = ?", w.ID).Count(&executionCount)

		// Count versions for this workflow
		var versionCount int64
		s.db.Model(&models.WorkflowVersion{}).Where("workflow_id = ?", w.ID).Count(&versionCount)

		responses[i] = WorkflowResponse{
			ID:             w.ID,
			Name:           w.Name,
			Description:    w.Description,
			IsActive:       w.IsActive,
			Schedule:       w.Schedule,
			Timezone:       w.Timezone,
			CurrentVersion: w.CurrentVersion,
			CreatedBy:      w.CreatedBy,
			Creator:        creatorDetails,
			Count: &WorkflowCount{
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
	var workflow models.Workflow
	err := s.db.First(&workflow, "id = ?", id).Error

	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	return &workflow, nil
}

// GetWorkflowByIdAndAccount returns a workflow by ID and account ID
func (s *WorkflowService) GetWorkflowByIdAndAccount(id, accountID string) (*models.Workflow, error) {
	var workflow models.Workflow
	err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&workflow).Error

	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Load the versions for this workflow
	versions, err := s.GetVersionHistory(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow versions: %w", err)
	}

	// Add versions to the workflow struct
	// Note: This is a workaround since GORM relationships aren't defined in the model
	// In a proper implementation, we'd define the relationship in the Workflow model
	workflow.Versions = versions

	return &workflow, nil
}

// CreateWorkflow creates a new workflow with optional scheduling
func (s *WorkflowService) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest, createdBy, accountID string) (*models.Workflow, error) {
	timezone := "UTC"
	if req.Timezone != nil {
		timezone = *req.Timezone
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	definitionJSON, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal definition: %w", err)
	}

	var workflow models.Workflow

	// Create workflow and version in a transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create workflow
		workflow = models.Workflow{
			Name:           req.Name,
			Description:    req.Description,
			IsActive:       isActive,
			Schedule:       req.Schedule,
			Timezone:       timezone,
			CurrentVersion: 1,
			AccountID:      &accountID,
			CreatedBy:      createdBy,
		}

		if err := tx.Create(&workflow).Error; err != nil {
			return err
		}

		// Create first version
		version := models.WorkflowVersion{
			WorkflowID: workflow.ID,
			Version:    1,
			Definition: string(definitionJSON),
			ChangeLog:  stringPtr("Initial version"),
		}

		if err := tx.Create(&version).Error; err != nil {
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

// UpdateWorkflow updates a workflow
func (s *WorkflowService) UpdateWorkflow(id string, req UpdateWorkflowRequest) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Update fields
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
	}

	if req.Definition != nil {
		updates["definition"] = req.Definition
	}

	if err := s.db.Model(&workflow).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Reload
	s.db.First(&workflow, "id = ?", id)
	return &workflow, nil
}

// UpdateWorkflowByAccount updates a workflow by ID and account ID
func (s *WorkflowService) UpdateWorkflowByAccount(id, accountID string, req UpdateWorkflowRequest) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&workflow).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	newVersion := workflow.CurrentVersion

	// If definition is updated, create a new version
	if req.Definition != nil {
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

		if err := s.db.Create(&version).Error; err != nil {
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

	if err := s.db.Model(&workflow).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Reload workflow
	if err := s.db.First(&workflow, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &workflow, nil
}

// UpdateSchedule updates the workflow schedule
func (s *WorkflowService) UpdateSchedule(ctx context.Context, id string, req UpdateScheduleRequest) error {
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", id).Error; err != nil {
		return fmt.Errorf("workflow not found: %w", err)
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

	if err := s.db.Model(&workflow).Updates(updates).Error; err != nil {
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
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", id).Error; err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// Remove from scheduler if scheduled
	if workflow.Schedule != nil {
		// Note: Scheduler cleanup would be handled by the scheduler service
	}

	// Delete workflow (cascade will handle related records)
	if err := s.db.Delete(&workflow).Error; err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}

// DeleteWorkflowByAccount deletes a workflow by ID and account ID
func (s *WorkflowService) DeleteWorkflowByAccount(ctx context.Context, id, accountID string) error {
	var workflow models.Workflow
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&workflow).Error; err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// Unschedule if scheduled
	if workflow.Schedule != nil && workflow.IsActive && s.schedulerService != nil {
		_ = s.schedulerService.RemoveSchedule(id) // Ignore error
	}

	// Delete workflow (cascade will delete versions, executions, etc.)
	if err := s.db.Delete(&workflow).Error; err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}

// ExecuteWorkflow queues a workflow for execution
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, id string, input map[string]interface{}) (jobID string, executionID string, err error) {
	// Check if workflow exists and get latest version
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", id).Error; err != nil {
		return "", "", fmt.Errorf("workflow not found: %w", err)
	}

	var latestVersion models.WorkflowVersion
	if err := s.db.Where("workflow_id = ?", id).
		Order("version DESC").
		First(&latestVersion).Error; err != nil {
		return "", "", fmt.Errorf("no version found for workflow: %w", err)
	}

	// Create execution record first
	inputJSON, _ := json.Marshal(input)
	inputStr := string(inputJSON)

	execution := models.WorkflowExecution{
		WorkflowID:  id,
		Version:     latestVersion.Version,
		Status:      "queued",
		TriggerType: "manual",
	}
	if len(inputStr) > 0 && inputStr != "null" {
		execution.Input = &inputStr
	}

	if err := s.db.Create(&execution).Error; err != nil {
		return "", "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// Queue execution with the execution ID
	jobID, err = s.queueService.QueueWorkflowExecution(ctx, id, execution.ID, input, "manual")
	if err != nil {
		// Rollback: mark execution as failed
		s.db.Model(&execution).Updates(map[string]interface{}{
			"status": "error",
			"error":  "Failed to queue for execution",
		})
		return "", "", fmt.Errorf("failed to queue workflow execution: %w", err)
	}

	return jobID, execution.ID, nil
}

// GetVersionHistory retrieves version history for a workflow
func (s *WorkflowService) GetVersionHistory(id string) ([]models.WorkflowVersion, error) {
	var versions []models.WorkflowVersion
	err := s.db.Where("workflow_id = ?", id).
		Order("version DESC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	return versions, nil
}

// GetWorkflowExecutions returns all executions for a workflow
func (s *WorkflowService) GetWorkflowExecutions(id string) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	err := s.db.Where("workflow_id = ?", id).
		Order("started_at DESC").
		Find(&executions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch executions: %w", err)
	}

	return executions, nil
}

// GetWorkflowExecutionById returns a specific execution with node executions
func (s *WorkflowService) GetWorkflowExecutionById(executionId string) (*ExecutionResponse, error) {
	var execution models.WorkflowExecution
	err := s.db.Where("id = ?", executionId).First(&execution).Error
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	// Get all node executions (including retries/failures)
	// Ordered by started_at DESC so most recent attempts appear first
	var nodeExecutions []models.WorkflowNodeExecution
	s.db.Where("execution_id = ?", executionId).
		Order("started_at DESC").
		Find(&nodeExecutions)

	// Convert node executions to response format
	nodeExecResponses := make([]NodeExecutionResponse, len(nodeExecutions))
	for i, ne := range nodeExecutions {
		nodeExecResponses[i] = NodeExecutionResponse{
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

	// Build response
	response := &ExecutionResponse{
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
func (s *WorkflowService) GetAllWorkflowExecutions(limit int, status string) ([]ExecutionResponse, error) {
	var executions []models.WorkflowExecution
	query := s.db.Order("started_at DESC").Limit(limit)

	// Apply status filter if provided
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch executions: %w", err)
	}

	return s.convertExecutionsToResponses(executions), nil
}

// GetFailedWorkflowExecutions returns all failed and partially failed workflow executions
func (s *WorkflowService) GetFailedWorkflowExecutions(limit int) ([]ExecutionResponse, error) {
	var executions []models.WorkflowExecution
	err := s.db.Where("status IN ?", []string{"error", "partially_failed"}).
		Order("started_at DESC").
		Limit(limit).
		Find(&executions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch failed executions: %w", err)
	}

	return s.convertExecutionsToResponses(executions), nil
}

// convertExecutionsToResponses converts execution models to response DTOs
func (s *WorkflowService) convertExecutionsToResponses(executions []models.WorkflowExecution) []ExecutionResponse {
	responses := make([]ExecutionResponse, len(executions))
	for i, exec := range executions {
		// Get node executions
		var nodeExecutions []models.WorkflowNodeExecution
		s.db.Where("execution_id = ?", exec.ID).
			Order("started_at DESC").
			Find(&nodeExecutions)

		// Convert node executions
		nodeExecResponses := make([]NodeExecutionResponse, len(nodeExecutions))
		for j, ne := range nodeExecutions {
			nodeExecResponses[j] = NodeExecutionResponse{
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
		var workflow models.Workflow
		var workflowResp *WorkflowResponse
		if err := s.db.First(&workflow, "id = ?", exec.WorkflowID).Error; err == nil {
			workflowResp = &WorkflowResponse{
				ID:   workflow.ID,
				Name: workflow.Name,
			}
		}

		responses[i] = ExecutionResponse{
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
	var execution models.WorkflowExecution
	if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
		return fmt.Errorf("execution not found: %w", err)
	}

	// Only running or queued executions can be cancelled
	if execution.Status != "running" && execution.Status != "queued" {
		return fmt.Errorf("only running or queued executions can be cancelled")
	}

	// Update execution status to cancelled
	now := time.Now()
	return s.db.Model(&execution).Updates(map[string]interface{}{
		"status":       "cancelled",
		"completed_at": now,
		"error":        "Execution cancelled by user",
	}).Error
}

// RestoreWorkflowVersion restores a workflow to a previous version
func (s *WorkflowService) RestoreWorkflowVersion(id string, version int) error {
	// Get the version
	var workflowVersion models.WorkflowVersion
	err := s.db.Where("workflow_id = ? AND version = ?", id, version).
		First(&workflowVersion).Error
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// Update the workflow with the version's definition
	err = s.db.Model(&models.Workflow{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"current_version": version,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to restore version: %w", err)
	}

	return nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
