package services

import (
	"context"
	"fmt"

	"github.com/patali/yantra/src/dto"

	"github.com/patali/yantra/src/db/repositories"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repositories.Repository
}

func NewUserService(repo repositories.Repository) *UserService {
	return &UserService{repo: repo}
}

// GetUserById retrieves a user by ID
func (s *UserService) GetUserById(id string) (*dto.UserResponse, error) {
	user, err := s.repo.User().FindByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetUserByIdInSameAccounts retrieves a user by ID only if they share an account with the current user
func (s *UserService) GetUserByIdInSameAccounts(id, currentUserID string) (*dto.UserResponse, error) {
	ctx := context.Background()

	// Find accounts the current user belongs to
	currentUserMemberships, err := s.repo.AccountMember().FindByUserID(ctx, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current user memberships: %w", err)
	}

	if len(currentUserMemberships) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	// Extract account IDs
	accountIDs := make([]string, len(currentUserMemberships))
	for i, m := range currentUserMemberships {
		accountIDs[i] = m.AccountID
	}

	// Check if the target user is a member of any of these accounts
	for _, accountID := range accountIDs {
		isMember, err := s.repo.AccountMember().IsUserMemberOfAccount(ctx, id, accountID)
		if err == nil && isMember {
			// Get the user details
			return s.GetUserById(id)
		}
	}

	return nil, fmt.Errorf("user not found")
}

// GetAllUsersForUser retrieves all users who share at least one account with the given user
func (s *UserService) GetAllUsersForUser(userID string) ([]dto.UserResponse, error) {
	ctx := context.Background()

	// Find accounts the user belongs to
	memberships, err := s.repo.AccountMember().FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user memberships: %w", err)
	}

	if len(memberships) == 0 {
		return []dto.UserResponse{}, nil
	}

	// Extract account IDs
	accountIDs := make([]string, len(memberships))
	for i, m := range memberships {
		accountIDs[i] = m.AccountID
	}

	// Find all users who are members of any of these accounts
	users, err := s.repo.User().FindUsersInAccounts(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	responses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Theme:     user.Theme,
			CreatedAt: user.CreatedAt,
		}
	}

	return responses, nil
}

// UpdateTheme updates a user's theme preference
func (s *UserService) UpdateTheme(id, theme string) (*dto.UserResponse, error) {
	if theme != "light" && theme != "dark" {
		return nil, fmt.Errorf("invalid theme: must be 'light' or 'dark'")
	}

	ctx := context.Background()
	user, err := s.repo.User().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"theme": theme,
	}

	if err := s.repo.User().Update(ctx, user, updates); err != nil {
		return nil, fmt.Errorf("failed to update theme: %w", err)
	}

	// Reload user
	user, _ = s.repo.User().FindByID(ctx, id)

	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	}, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(id, currentUserID string) error {
	// Prevent deleting yourself
	if id == currentUserID {
		return fmt.Errorf("cannot delete your own account")
	}

	ctx := context.Background()
	return s.repo.User().Delete(ctx, id)
}

// DeleteUserInSameAccounts deletes a user only if they share an account with the current user
func (s *UserService) DeleteUserInSameAccounts(id, currentUserID string) error {
	// Prevent deleting yourself
	if id == currentUserID {
		return fmt.Errorf("cannot delete your own account")
	}

	ctx := context.Background()

	// Find accounts the current user belongs to
	currentUserMemberships, err := s.repo.AccountMember().FindByUserID(ctx, currentUserID)
	if err != nil {
		return fmt.Errorf("failed to fetch current user memberships: %w", err)
	}

	if len(currentUserMemberships) == 0 {
		return fmt.Errorf("user not found")
	}

	// Extract account IDs
	accountIDs := make([]string, len(currentUserMemberships))
	for i, m := range currentUserMemberships {
		accountIDs[i] = m.AccountID
	}

	// Check if the target user is a member of any of these accounts
	isMember := false
	for _, accountID := range accountIDs {
		member, _ := s.repo.AccountMember().IsUserMemberOfAccount(ctx, id, accountID)
		if member {
			isMember = true
			break
		}
	}

	if !isMember {
		return fmt.Errorf("user not found")
	}

	// Delete the user
	return s.repo.User().Delete(ctx, id)
}

// UpdatePassword updates a user's password after verifying the current password
func (s *UserService) UpdatePassword(userID, currentPassword, newPassword string) error {
	ctx := context.Background()

	// Fetch user
	user, err := s.repo.User().FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	updates := map[string]interface{}{
		"password": string(hashedPassword),
	}

	if err := s.repo.User().Update(ctx, user, updates); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
