package repositories

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type workflowRepository struct {
	db *gorm.DB
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *gorm.DB) WorkflowRepository {
	return &workflowRepository{db: db}
}

func (r *workflowRepository) FindByID(ctx context.Context, id string) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := r.db.WithContext(ctx).First(&workflow, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}
	// Populate computed field
	workflow.HasWebhookSecret = workflow.WebhookSecretHash != nil && *workflow.WebhookSecretHash != ""
	return &workflow, nil
}

func (r *workflowRepository) FindByIDAndAccount(ctx context.Context, id, accountID string) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := r.db.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).First(&workflow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}
	// Populate computed field
	workflow.HasWebhookSecret = workflow.WebhookSecretHash != nil && *workflow.WebhookSecretHash != ""
	return &workflow, nil
}

func (r *workflowRepository) FindByAccountID(ctx context.Context, accountID string) ([]models.Workflow, error) {
	var workflows []models.Workflow
	err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&workflows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find workflows: %w", err)
	}

	// Populate computed field for all workflows
	for i := range workflows {
		workflows[i].HasWebhookSecret = workflows[i].WebhookSecretHash != nil && *workflows[i].WebhookSecretHash != ""
	}

	return workflows, nil
}

func (r *workflowRepository) Create(ctx context.Context, workflow *models.Workflow) error {
	if err := r.db.WithContext(ctx).Create(workflow).Error; err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}
	return nil
}

func (r *workflowRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.Workflow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}
	return nil
}

func (r *workflowRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Workflow{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete workflow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}

func (r *workflowRepository) CountExecutions(ctx context.Context, workflowID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.WorkflowExecution{}).Where("workflow_id = ?", workflowID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}
	return count, nil
}

type workflowVersionRepository struct {
	db *gorm.DB
}

// NewWorkflowVersionRepository creates a new workflow version repository
func NewWorkflowVersionRepository(db *gorm.DB) WorkflowVersionRepository {
	return &workflowVersionRepository{db: db}
}

func (r *workflowVersionRepository) FindByWorkflowID(ctx context.Context, workflowID string) ([]models.WorkflowVersion, error) {
	var versions []models.WorkflowVersion
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("version DESC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find versions: %w", err)
	}
	return versions, nil
}

func (r *workflowVersionRepository) FindByWorkflowIDAndVersion(ctx context.Context, workflowID string, version int) (*models.WorkflowVersion, error) {
	var wfVersion models.WorkflowVersion
	if err := r.db.WithContext(ctx).Where("workflow_id = ? AND version = ?", workflowID, version).First(&wfVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("version not found")
		}
		return nil, fmt.Errorf("failed to find version: %w", err)
	}
	return &wfVersion, nil
}

func (r *workflowVersionRepository) FindLatestByWorkflowID(ctx context.Context, workflowID string) (*models.WorkflowVersion, error) {
	var version models.WorkflowVersion
	if err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("version DESC").
		First(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no version found for workflow")
		}
		return nil, fmt.Errorf("failed to find latest version: %w", err)
	}
	return &version, nil
}

func (r *workflowVersionRepository) CountByWorkflowID(ctx context.Context, workflowID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.WorkflowVersion{}).Where("workflow_id = ?", workflowID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count versions: %w", err)
	}
	return count, nil
}

func (r *workflowVersionRepository) Create(ctx context.Context, version *models.WorkflowVersion) error {
	if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}
	return nil
}
