package repositories

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/db/models"
	"gorm.io/gorm"
)

type emailProviderRepository struct {
	db *gorm.DB
}

// NewEmailProviderRepository creates a new email provider repository
func NewEmailProviderRepository(db *gorm.DB) EmailProviderRepository {
	return &emailProviderRepository{db: db}
}

func (r *emailProviderRepository) FindByID(ctx context.Context, id string) (*models.EmailProviderSettings, error) {
	var provider models.EmailProviderSettings
	if err := r.db.WithContext(ctx).First(&provider, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("email provider not found")
		}
		return nil, fmt.Errorf("failed to find email provider: %w", err)
	}
	return &provider, nil
}

func (r *emailProviderRepository) FindByAccountID(ctx context.Context, accountID string) ([]models.EmailProviderSettings, error) {
	var providers []models.EmailProviderSettings
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to find email providers: %w", err)
	}
	return providers, nil
}

func (r *emailProviderRepository) FindByAccountIDAndProvider(ctx context.Context, accountID, provider string) (*models.EmailProviderSettings, error) {
	var emailProvider models.EmailProviderSettings
	if err := r.db.WithContext(ctx).Where("account_id = ? AND provider = ?", accountID, provider).First(&emailProvider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil without error if not found
		}
		return nil, fmt.Errorf("failed to find email provider: %w", err)
	}
	return &emailProvider, nil
}

func (r *emailProviderRepository) FindActiveByAccountID(ctx context.Context, accountID string) (*models.EmailProviderSettings, error) {
	var provider models.EmailProviderSettings
	if err := r.db.WithContext(ctx).Where("account_id = ? AND is_active = ?", accountID, true).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active email provider found")
		}
		return nil, fmt.Errorf("failed to find active email provider: %w", err)
	}
	return &provider, nil
}

func (r *emailProviderRepository) Create(ctx context.Context, settings *models.EmailProviderSettings) error {
	if err := r.db.WithContext(ctx).Create(settings).Error; err != nil {
		return fmt.Errorf("failed to create email provider: %w", err)
	}
	return nil
}

func (r *emailProviderRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.EmailProviderSettings{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update email provider: %w", err)
	}
	return nil
}

func (r *emailProviderRepository) DeactivateAllForAccount(ctx context.Context, accountID string) error {
	if err := r.db.WithContext(ctx).Model(&models.EmailProviderSettings{}).
		Where("account_id = ?", accountID).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate email providers: %w", err)
	}
	return nil
}

func (r *emailProviderRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.EmailProviderSettings{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete email provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("email provider not found")
	}
	return nil
}
