package services

import (
	"fmt"

	"github.com/patali/yantra/internal/models"
	"gorm.io/gorm"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

type CreateAccountRequest struct {
	Name string `json:"name" binding:"required"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role"`
}

type AccountResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Members   []MemberResponse `json:"members,omitempty"`
	CreatedAt string           `json:"createdAt"`
}

type MemberResponse struct {
	UserID   string       `json:"userId"`
	Role     string       `json:"role"`
	User     UserResponse `json:"user,omitempty"`
	JoinedAt string       `json:"joinedAt"`
}

// CreateAccount creates a new account with the user as owner
func (s *AccountService) CreateAccount(name, ownerUserID string) (*AccountResponse, error) {
	var account models.Account
	var member models.AccountMember

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create account
		account = models.Account{
			Name: name,
		}
		if err := tx.Create(&account).Error; err != nil {
			return err
		}

		// Add user as owner
		member = models.AccountMember{
			AccountID: account.ID,
			UserID:    ownerUserID,
			Role:      "owner",
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Load with relationships
	s.db.Preload("Members").Preload("Members.User").First(&account, "id = ?", account.ID)

	return s.toAccountResponse(&account), nil
}

// AddMember adds a user to an account
func (s *AccountService) AddMember(accountID, userID, role string) error {
	if role == "" {
		role = "member"
	}

	// Validate role
	if role != "owner" && role != "admin" && role != "member" {
		return fmt.Errorf("invalid role: %s", role)
	}

	// Check if account exists
	var account models.Account
	if err := s.db.First(&account, "id = ?", accountID).Error; err != nil {
		return fmt.Errorf("account not found")
	}

	// Check if user exists
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Check if member already exists
	var existingMember models.AccountMember
	result := s.db.Where("account_id = ? AND user_id = ?", accountID, userID).First(&existingMember)
	if result.Error == nil {
		return fmt.Errorf("user is already a member of this account")
	}

	// Create membership
	member := models.AccountMember{
		AccountID: accountID,
		UserID:    userID,
		Role:      role,
	}

	if err := s.db.Create(&member).Error; err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// RemoveMember removes a user from an account
func (s *AccountService) RemoveMember(accountID, userID string) error {
	result := s.db.Where("account_id = ? AND user_id = ?", accountID, userID).Delete(&models.AccountMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove member: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// ListMyAccounts lists all accounts the user is a member of
func (s *AccountService) ListMyAccounts(userID string) ([]AccountResponse, error) {
	var accounts []models.Account

	err := s.db.Joins("JOIN account_members ON account_members.account_id = accounts.id").
		Where("account_members.user_id = ?", userID).
		Preload("Members").
		Preload("Members.User").
		Find(&accounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	responses := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = *s.toAccountResponse(&account)
	}

	return responses, nil
}

// GetAccountByID retrieves an account by ID with all members
func (s *AccountService) GetAccountByID(accountID string) (*AccountResponse, error) {
	var account models.Account

	err := s.db.Preload("Members").Preload("Members.User").First(&account, "id = ?", accountID).Error
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	return s.toAccountResponse(&account), nil
}

// UpdateAccount updates account details
func (s *AccountService) UpdateAccount(accountID, name string) error {
	result := s.db.Model(&models.Account{}).Where("id = ?", accountID).Update("name", name)
	if result.Error != nil {
		return fmt.Errorf("failed to update account: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// Helper to convert model to response
func (s *AccountService) toAccountResponse(account *models.Account) *AccountResponse {
	response := &AccountResponse{
		ID:        account.ID,
		Name:      account.Name,
		CreatedAt: account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Fetch members separately
	var accountMembers []models.AccountMember
	if err := s.db.Where("account_id = ?", account.ID).Find(&accountMembers).Error; err == nil && len(accountMembers) > 0 {
		members := make([]MemberResponse, len(accountMembers))
		for i, member := range accountMembers {
			members[i] = MemberResponse{
				UserID:   member.UserID,
				Role:     member.Role,
				JoinedAt: member.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
			// Fetch user details separately
			var user models.User
			if err := s.db.Where("id = ?", member.UserID).First(&user).Error; err == nil {
				members[i].User = UserResponse{
					ID:        user.ID,
					Username:  user.Username,
					Email:     user.Email,
					Theme:     user.Theme,
					CreatedAt: user.CreatedAt,
				}
			}
		}
		response.Members = members
	}

	return response
}
