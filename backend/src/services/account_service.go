package services

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/dto"

	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/db/repositories"
)

type AccountService struct {
	repo repositories.Repository
}

func NewAccountService(repo repositories.Repository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateAccount creates a new account with the user as owner
func (s *AccountService) CreateAccount(name, ownerUserID string) (*dto.AccountResponse, error) {
	ctx := context.Background()
	var account models.Account

	err := s.repo.Transaction(ctx, func(txRepo repositories.TxRepository) error {
		// Create account
		account = models.Account{
			Name: name,
		}
		if err := txRepo.Account().Create(ctx, &account); err != nil {
			return err
		}

		// Add user as owner
		member := models.AccountMember{
			AccountID: account.ID,
			UserID:    ownerUserID,
			Role:      "owner",
		}
		if err := txRepo.AccountMember().Create(ctx, &member); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Load with relationships
	account2, _ := s.repo.Account().FindByID(ctx, account.ID)

	return s.toAccountResponse(account2), nil
}

// AddMember adds a user to an account
func (s *AccountService) AddMember(accountID, userID, role string) error {
	ctx := context.Background()

	if role == "" {
		role = "member"
	}

	// Validate role
	if role != "owner" && role != "admin" && role != "member" {
		return fmt.Errorf("invalid role: %s", role)
	}

	// Check if account exists
	_, err := s.repo.Account().FindByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("account not found")
	}

	// Check if user exists
	_, err = s.repo.User().FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Check if member already exists
	existingMember, _ := s.repo.AccountMember().FindByUserAndAccount(ctx, userID, accountID)
	if existingMember != nil {
		return fmt.Errorf("user is already a member of this account")
	}

	// Create membership
	member := models.AccountMember{
		AccountID: accountID,
		UserID:    userID,
		Role:      role,
	}

	if err := s.repo.AccountMember().Create(ctx, &member); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// RemoveMember removes a user from an account
func (s *AccountService) RemoveMember(accountID, userID string) error {
	ctx := context.Background()
	return s.repo.AccountMember().Delete(ctx, accountID, userID)
}

// ListMyAccounts lists all accounts the user is a member of
func (s *AccountService) ListMyAccounts(userID string) ([]dto.AccountResponse, error) {
	ctx := context.Background()

	accounts, err := s.repo.Account().FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	responses := make([]dto.AccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = *s.toAccountResponse(&account)
	}

	return responses, nil
}

// GetAccountByID retrieves an account by ID with all members
func (s *AccountService) GetAccountByID(accountID string) (*dto.AccountResponse, error) {
	ctx := context.Background()

	account, err := s.repo.Account().FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return s.toAccountResponse(account), nil
}

// UpdateAccount updates account details
func (s *AccountService) UpdateAccount(accountID, name string) error {
	ctx := context.Background()
	updates := map[string]interface{}{
		"name": name,
	}
	return s.repo.Account().Update(ctx, accountID, updates)
}

// IsUserMemberOfAccount checks if a user is a member of an account
func (s *AccountService) IsUserMemberOfAccount(userID, accountID string) (bool, error) {
	ctx := context.Background()
	return s.repo.AccountMember().IsUserMemberOfAccount(ctx, userID, accountID)
}

// GetUserRoleInAccount gets the user's role in an account (owner, admin, member)
func (s *AccountService) GetUserRoleInAccount(userID, accountID string) (string, error) {
	ctx := context.Background()
	return s.repo.AccountMember().GetUserRole(ctx, userID, accountID)
}

// Helper to convert model to response
func (s *AccountService) toAccountResponse(account *models.Account) *dto.AccountResponse {
	response := &dto.AccountResponse{
		ID:        account.ID,
		Name:      account.Name,
		CreatedAt: account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Fetch members separately
	ctx := context.Background()
	accountMembers, err := s.repo.AccountMember().FindByAccountID(ctx, account.ID)
	if err == nil && len(accountMembers) > 0 {
		members := make([]dto.MemberResponse, len(accountMembers))
		for i, member := range accountMembers {
			members[i] = dto.MemberResponse{
				UserID:   member.UserID,
				Role:     member.Role,
				JoinedAt: member.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
			// Fetch user details separately
			user, err := s.repo.User().FindByID(ctx, member.UserID)
			if err == nil {
				members[i].User = dto.UserResponse{
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
