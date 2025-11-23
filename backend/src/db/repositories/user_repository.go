package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindByUsernameOrEmail(ctx context.Context, username, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ? OR email = ?", username, email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil without error if not found
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindUsersInAccounts(ctx context.Context, accountIDs []string) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).
		Joins("JOIN account_members ON account_members.user_id = users.id").
		Where("account_members.account_id IN ?", accountIDs).
		Distinct("users.*").
		Order("users.created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	return users, nil
}

func (r *userRepository) FindWithResetToken(ctx context.Context) ([]models.User, error) {
	now := time.Now()
	var users []models.User
	if err := r.db.WithContext(ctx).Where("reset_token_hash IS NOT NULL AND reset_token_exp > ?", now).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to find users with reset tokens: %w", err)
	}
	return users, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
