package repositories

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) FindByID(ctx context.Context, id string) (*models.Account, error) {
	var account models.Account
	if err := r.db.WithContext(ctx).
		Preload("Members").
		Preload("Members.User").
		First(&account, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}
	return &account, nil
}

func (r *accountRepository) FindByUserID(ctx context.Context, userID string) ([]models.Account, error) {
	var accounts []models.Account
	err := r.db.WithContext(ctx).
		Joins("JOIN account_members ON account_members.account_id = accounts.id").
		Where("account_members.user_id = ?", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&accounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find accounts: %w", err)
	}
	return accounts, nil
}

func (r *accountRepository) Create(ctx context.Context, account *models.Account) error {
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (r *accountRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (r *accountRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Account{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

type accountMemberRepository struct {
	db *gorm.DB
}

// NewAccountMemberRepository creates a new account member repository
func NewAccountMemberRepository(db *gorm.DB) AccountMemberRepository {
	return &accountMemberRepository{db: db}
}

func (r *accountMemberRepository) FindByUserID(ctx context.Context, userID string) ([]models.AccountMember, error) {
	var members []models.AccountMember
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to find memberships: %w", err)
	}
	return members, nil
}

func (r *accountMemberRepository) FindByAccountID(ctx context.Context, accountID string) ([]models.AccountMember, error) {
	var members []models.AccountMember
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to find members: %w", err)
	}
	return members, nil
}

func (r *accountMemberRepository) FindByUserAndAccount(ctx context.Context, userID, accountID string) (*models.AccountMember, error) {
	var member models.AccountMember
	if err := r.db.WithContext(ctx).Where("user_id = ? AND account_id = ?", userID, accountID).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("membership not found")
		}
		return nil, fmt.Errorf("failed to find membership: %w", err)
	}
	return &member, nil
}

func (r *accountMemberRepository) IsUserMemberOfAccount(ctx context.Context, userID, accountID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AccountMember{}).
		Where("user_id = ? AND account_id = ?", userID, accountID).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return count > 0, nil
}

func (r *accountMemberRepository) GetUserRole(ctx context.Context, userID, accountID string) (string, error) {
	var member models.AccountMember
	err := r.db.WithContext(ctx).Where("user_id = ? AND account_id = ?", userID, accountID).First(&member).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("user is not a member of this account")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}
	return member.Role, nil
}

func (r *accountMemberRepository) Create(ctx context.Context, member *models.AccountMember) error {
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		return fmt.Errorf("failed to create membership: %w", err)
	}
	return nil
}

func (r *accountMemberRepository) Delete(ctx context.Context, accountID, userID string) error {
	result := r.db.WithContext(ctx).Where("account_id = ? AND user_id = ?", accountID, userID).Delete(&models.AccountMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete membership: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("membership not found")
	}
	return nil
}
