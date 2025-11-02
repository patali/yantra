package repositories

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type executionRepository struct {
	db *gorm.DB
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(db *gorm.DB) ExecutionRepository {
	return &executionRepository{db: db}
}

func (r *executionRepository) FindByID(ctx context.Context, id string) (*models.WorkflowExecution, error) {
	var execution models.WorkflowExecution
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution not found")
		}
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}
	return &execution, nil
}

func (r *executionRepository) FindByWorkflowID(ctx context.Context, workflowID string) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("started_at DESC").
		Find(&executions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) FindAll(ctx context.Context, limit int, status string) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	query := r.db.WithContext(ctx).Order("started_at DESC").Limit(limit)

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find all executions: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) FindFailed(ctx context.Context, limit int) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{"error", "partially_failed"}).
		Order("started_at DESC").
		Limit(limit).
		Find(&executions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find failed executions: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) FindAllByAccountID(ctx context.Context, accountID string, limit int, status string) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution

	query := r.db.WithContext(ctx).Table("workflow_executions").
		Select("workflow_executions.*").
		Joins("INNER JOIN workflows ON workflows.id = workflow_executions.workflow_id").
		Where("workflows.account_id = ?", accountID).
		Order("workflow_executions.started_at DESC").
		Limit(limit)

	if status != "" && status != "all" {
		query = query.Where("workflow_executions.status = ?", status)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find executions by account: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) FindFailedByAccountID(ctx context.Context, accountID string, limit int) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution

	query := r.db.WithContext(ctx).
		Table("workflow_executions").
		Select("workflow_executions.*").
		Joins("INNER JOIN workflows ON workflows.id = workflow_executions.workflow_id").
		Where("workflows.account_id = ?", accountID).
		Where("workflow_executions.status IN ?", []string{"error", "partially_failed"}).
		Order("workflow_executions.started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find failed executions: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) FindRunningExecutions(ctx context.Context) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	if err := r.db.WithContext(ctx).Where("status = ?", "running").Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to find running executions: %w", err)
	}
	return executions, nil
}

func (r *executionRepository) Create(ctx context.Context, execution *models.WorkflowExecution) error {
	if err := r.db.WithContext(ctx).Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}
	return nil
}

func (r *executionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.WorkflowExecution{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}
	return nil
}

type nodeExecutionRepository struct {
	db *gorm.DB
}

// NewNodeExecutionRepository creates a new node execution repository
func NewNodeExecutionRepository(db *gorm.DB) NodeExecutionRepository {
	return &nodeExecutionRepository{db: db}
}

func (r *nodeExecutionRepository) FindByExecutionID(ctx context.Context, executionID string) ([]models.WorkflowNodeExecution, error) {
	var nodeExecutions []models.WorkflowNodeExecution
	if err := r.db.WithContext(ctx).
		Where("execution_id = ?", executionID).
		Order("started_at DESC").
		Find(&nodeExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to find node executions: %w", err)
	}
	return nodeExecutions, nil
}

func (r *nodeExecutionRepository) FindByID(ctx context.Context, id string) (*models.WorkflowNodeExecution, error) {
	var nodeExecution models.WorkflowNodeExecution
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&nodeExecution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("node execution not found")
		}
		return nil, fmt.Errorf("failed to find node execution: %w", err)
	}
	return &nodeExecution, nil
}

func (r *nodeExecutionRepository) Create(ctx context.Context, nodeExecution *models.WorkflowNodeExecution) error {
	if err := r.db.WithContext(ctx).Create(nodeExecution).Error; err != nil {
		return fmt.Errorf("failed to create node execution: %w", err)
	}
	return nil
}

func (r *nodeExecutionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.WorkflowNodeExecution{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update node execution: %w", err)
	}
	return nil
}

func (r *nodeExecutionRepository) UpdateByExecutionIDAndStatus(ctx context.Context, executionID, status string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.WorkflowNodeExecution{}).
		Where("execution_id = ? AND status = ?", executionID, status).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update node executions: %w", err)
	}
	return nil
}
