package repositories

import (
	"context"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

// UserRepository defines operations for user data access
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUsernameOrEmail(ctx context.Context, username, email string) (*models.User, error)
	FindUsersInAccounts(ctx context.Context, accountIDs []string) ([]models.User, error)
	FindWithResetToken(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

// AccountRepository defines operations for account data access
type AccountRepository interface {
	FindByID(ctx context.Context, id string) (*models.Account, error)
	FindByUserID(ctx context.Context, userID string) ([]models.Account, error)
	Create(ctx context.Context, account *models.Account) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

// AccountMemberRepository defines operations for account membership
type AccountMemberRepository interface {
	FindByUserID(ctx context.Context, userID string) ([]models.AccountMember, error)
	FindByAccountID(ctx context.Context, accountID string) ([]models.AccountMember, error)
	FindByUserAndAccount(ctx context.Context, userID, accountID string) (*models.AccountMember, error)
	IsUserMemberOfAccount(ctx context.Context, userID, accountID string) (bool, error)
	GetUserRole(ctx context.Context, userID, accountID string) (string, error)
	Create(ctx context.Context, member *models.AccountMember) error
	Delete(ctx context.Context, accountID, userID string) error
}

// WorkflowRepository defines operations for workflow data access
type WorkflowRepository interface {
	FindByID(ctx context.Context, id string) (*models.Workflow, error)
	FindByIDAndAccount(ctx context.Context, id, accountID string) (*models.Workflow, error)
	FindByAccountID(ctx context.Context, accountID string) ([]models.Workflow, error)
	Create(ctx context.Context, workflow *models.Workflow) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	CountExecutions(ctx context.Context, workflowID string) (int64, error)
}

// WorkflowVersionRepository defines operations for workflow versions
type WorkflowVersionRepository interface {
	FindByWorkflowID(ctx context.Context, workflowID string) ([]models.WorkflowVersion, error)
	FindByWorkflowIDAndVersion(ctx context.Context, workflowID string, version int) (*models.WorkflowVersion, error)
	FindLatestByWorkflowID(ctx context.Context, workflowID string) (*models.WorkflowVersion, error)
	CountByWorkflowID(ctx context.Context, workflowID string) (int64, error)
	Create(ctx context.Context, version *models.WorkflowVersion) error
}

// ExecutionRepository defines operations for workflow execution data access
type ExecutionRepository interface {
	FindByID(ctx context.Context, id string) (*models.WorkflowExecution, error)
	FindByWorkflowID(ctx context.Context, workflowID string) ([]models.WorkflowExecution, error)
	FindAll(ctx context.Context, limit int, status string) ([]models.WorkflowExecution, error)
	FindFailed(ctx context.Context, limit int) ([]models.WorkflowExecution, error)
	FindAllByAccountID(ctx context.Context, accountID string, limit int, status string) ([]models.WorkflowExecution, error)
	FindFailedByAccountID(ctx context.Context, accountID string, limit int) ([]models.WorkflowExecution, error)
	Create(ctx context.Context, execution *models.WorkflowExecution) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
}

// NodeExecutionRepository defines operations for node executions
type NodeExecutionRepository interface {
	FindByExecutionID(ctx context.Context, executionID string) ([]models.WorkflowNodeExecution, error)
	FindByID(ctx context.Context, id string) (*models.WorkflowNodeExecution, error)
	Create(ctx context.Context, nodeExecution *models.WorkflowNodeExecution) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateByExecutionIDAndStatus(ctx context.Context, executionID, status string, updates map[string]interface{}) error
}

// OutboxRepository defines operations for outbox messages
type OutboxRepository interface {
	CountOrphanedMessages(ctx context.Context) (int64, error)
	Create(ctx context.Context, message *models.OutboxMessage) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateOrphanedMessages(ctx context.Context, updates map[string]interface{}) error
}

// EmailProviderRepository defines operations for email provider settings
type EmailProviderRepository interface {
	FindByID(ctx context.Context, id string) (*models.EmailProviderSettings, error)
	FindByAccountID(ctx context.Context, accountID string) ([]models.EmailProviderSettings, error)
	FindByAccountIDAndProvider(ctx context.Context, accountID, provider string) (*models.EmailProviderSettings, error)
	FindActiveByAccountID(ctx context.Context, accountID string) (*models.EmailProviderSettings, error)
	Create(ctx context.Context, settings *models.EmailProviderSettings) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	DeactivateAllForAccount(ctx context.Context, accountID string) error
	Delete(ctx context.Context, id string) error
}

// TxRepository represents repositories within a transaction context
type TxRepository interface {
	User() UserRepository
	Account() AccountRepository
	AccountMember() AccountMemberRepository
	Workflow() WorkflowRepository
	WorkflowVersion() WorkflowVersionRepository
	Execution() ExecutionRepository
	NodeExecution() NodeExecutionRepository
	Outbox() OutboxRepository
	EmailProvider() EmailProviderRepository
}

// Repository aggregates all repository interfaces
type Repository interface {
	TxRepository
	// Transaction executes a function within a database transaction
	Transaction(ctx context.Context, fn func(TxRepository) error) error
	// DB returns the underlying GORM DB instance (for backward compatibility)
	DB() *gorm.DB
}
