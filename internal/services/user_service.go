package services

import (
	"fmt"

	"github.com/patali/yantra/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type UpdateThemeRequest struct {
	Theme string `json:"theme" binding:"required"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

// GetUserById retrieves a user by ID
func (s *UserService) GetUserById(id string) (*UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetUserByIdInSameAccounts retrieves a user by ID only if they share an account with the current user
func (s *UserService) GetUserByIdInSameAccounts(id, currentUserID string) (*UserResponse, error) {
	// Find accounts the current user belongs to
	var currentUserMemberships []models.AccountMember
	if err := s.db.Where("user_id = ?", currentUserID).Find(&currentUserMemberships).Error; err != nil {
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
	var targetUserMembership models.AccountMember
	if err := s.db.Where("user_id = ? AND account_id IN ?", id, accountIDs).First(&targetUserMembership).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get the user details
	return s.GetUserById(id)
}

// GetAllUsersForUser retrieves all users who share at least one account with the given user
func (s *UserService) GetAllUsersForUser(userID string) ([]UserResponse, error) {
	// Find accounts the user belongs to
	var memberships []models.AccountMember
	if err := s.db.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch user memberships: %w", err)
	}

	if len(memberships) == 0 {
		return []UserResponse{}, nil
	}

	// Extract account IDs
	accountIDs := make([]string, len(memberships))
	for i, m := range memberships {
		accountIDs[i] = m.AccountID
	}

	// Find all users who are members of any of these accounts
	var users []models.User
	err := s.db.Joins("JOIN account_members ON account_members.user_id = users.id").
		Where("account_members.account_id IN ?", accountIDs).
		Distinct("users.*").
		Order("users.created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = UserResponse{
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
func (s *UserService) UpdateTheme(id, theme string) (*UserResponse, error) {
	if theme != "light" && theme != "dark" {
		return nil, fmt.Errorf("invalid theme: must be 'light' or 'dark'")
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if err := s.db.Model(&user).Update("theme", theme).Error; err != nil {
		return nil, fmt.Errorf("failed to update theme: %w", err)
	}

	// Reload user
	s.db.First(&user, "id = ?", id)

	return &UserResponse{
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

	result := s.db.Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeleteUserInSameAccounts deletes a user only if they share an account with the current user
func (s *UserService) DeleteUserInSameAccounts(id, currentUserID string) error {
	// Prevent deleting yourself
	if id == currentUserID {
		return fmt.Errorf("cannot delete your own account")
	}

	// Find accounts the current user belongs to
	var currentUserMemberships []models.AccountMember
	if err := s.db.Where("user_id = ?", currentUserID).Find(&currentUserMemberships).Error; err != nil {
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
	var targetUserMembership models.AccountMember
	if err := s.db.Where("user_id = ? AND account_id IN ?", id, accountIDs).First(&targetUserMembership).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Delete the user
	result := s.db.Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePassword updates a user's password after verifying the current password
func (s *UserService) UpdatePassword(userID, currentPassword, newPassword string) error {
	// Fetch user
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
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
	if err := s.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
