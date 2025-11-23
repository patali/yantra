package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// repository is the main repository implementation
type repository struct {
	db                  *gorm.DB
	userRepo            UserRepository
	accountRepo         AccountRepository
	accountMemberRepo   AccountMemberRepository
	workflowRepo        WorkflowRepository
	workflowVersionRepo WorkflowVersionRepository
	executionRepo       ExecutionRepository
	nodeExecutionRepo   NodeExecutionRepository
	outboxRepo          OutboxRepository
	emailProviderRepo   EmailProviderRepository
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db:                  db,
		userRepo:            NewUserRepository(db),
		accountRepo:         NewAccountRepository(db),
		accountMemberRepo:   NewAccountMemberRepository(db),
		workflowRepo:        NewWorkflowRepository(db),
		workflowVersionRepo: NewWorkflowVersionRepository(db),
		executionRepo:       NewExecutionRepository(db),
		nodeExecutionRepo:   NewNodeExecutionRepository(db),
		outboxRepo:          NewOutboxRepository(db),
		emailProviderRepo:   NewEmailProviderRepository(db),
	}
}

func (r *repository) User() UserRepository {
	return r.userRepo
}

func (r *repository) Account() AccountRepository {
	return r.accountRepo
}

func (r *repository) AccountMember() AccountMemberRepository {
	return r.accountMemberRepo
}

func (r *repository) Workflow() WorkflowRepository {
	return r.workflowRepo
}

func (r *repository) WorkflowVersion() WorkflowVersionRepository {
	return r.workflowVersionRepo
}

func (r *repository) Execution() ExecutionRepository {
	return r.executionRepo
}

func (r *repository) NodeExecution() NodeExecutionRepository {
	return r.nodeExecutionRepo
}

func (r *repository) Outbox() OutboxRepository {
	return r.outboxRepo
}

func (r *repository) EmailProvider() EmailProviderRepository {
	return r.emailProviderRepo
}

func (r *repository) DB() *gorm.DB {
	return r.db
}

// Transaction executes a function within a database transaction
func (r *repository) Transaction(ctx context.Context, fn func(TxRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{
			db:                  tx,
			userRepo:            NewUserRepository(tx),
			accountRepo:         NewAccountRepository(tx),
			accountMemberRepo:   NewAccountMemberRepository(tx),
			workflowRepo:        NewWorkflowRepository(tx),
			workflowVersionRepo: NewWorkflowVersionRepository(tx),
			executionRepo:       NewExecutionRepository(tx),
			nodeExecutionRepo:   NewNodeExecutionRepository(tx),
			outboxRepo:          NewOutboxRepository(tx),
			emailProviderRepo:   NewEmailProviderRepository(tx),
		}
		return fn(txRepo)
	})
}

// WithDB creates a new repository instance with a different DB instance
// This is useful for testing or when you need to use a specific DB session
func WithDB(db *gorm.DB) Repository {
	return NewRepository(db)
}

// GetDBFromContext extracts the DB instance from context (for backward compatibility)
func GetDBFromContext(ctx context.Context) (*gorm.DB, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database not found in context")
	}
	return db, nil
}
