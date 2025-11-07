package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/db/repositories"
	"github.com/patali/yantra/src/dto"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	repo           repositories.Repository
	db             *gorm.DB // Keep for backward compatibility during migration
	jwtSecret      string
	systemEmailSvc *SystemEmailService
}

func NewAuthService(db *gorm.DB, jwtSecret string, systemEmailSvc *SystemEmailService) *AuthService {
	return &AuthService{
		repo:           repositories.NewRepository(db),
		db:             db,
		jwtSecret:      jwtSecret,
		systemEmailSvc: systemEmailSvc,
	}
}

// CreateUser creates a new user (for invitations)
func (s *AuthService) CreateUser(req dto.CreateUserRequest, createdBy *string) (*dto.UserResponse, error) {
	ctx := context.Background()

	// Check if user already exists
	existingUser, _ := s.repo.User().FindByUsernameOrEmail(ctx, req.Username, req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with this username or email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedBy: createdBy,
	}

	// If createdBy is provided, get their account memberships to add the new user
	if createdBy != nil {
		memberships, _ := s.repo.AccountMember().FindByUserID(ctx, *createdBy)

		// Create user with memberships in a transaction
		err = s.repo.Transaction(ctx, func(txRepo repositories.TxRepository) error {
			if err := txRepo.User().Create(ctx, &user); err != nil {
				return err
			}

			for _, m := range memberships {
				membership := models.AccountMember{
					AccountID: m.AccountID,
					UserID:    user.ID,
					Role:      "member",
				}
				if err := txRepo.AccountMember().Create(ctx, &membership); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		if err := s.repo.User().Create(ctx, &user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	ctx := context.Background()

	// Find user
	user, err := s.repo.User().FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Get user's primary account (first account they joined)
	memberships, err := s.repo.AccountMember().FindByUserID(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, fmt.Errorf("user is not associated with any account")
	}
	accountID := memberships[0].AccountID

	// Generate JWT token
	token, err := s.generateToken(user.ID, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Theme:     user.Theme,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

// SignupWithAccount creates a new user and account (for first-time signup)
func (s *AuthService) SignupWithAccount(req dto.SignupWithAccountRequest) (*dto.LoginResponse, error) {
	ctx := context.Background()

	// Check if user already exists
	existingUser, _ := s.repo.User().FindByUsernameOrEmail(ctx, req.Username, req.Email)
	if existingUser != nil {
		if existingUser.Username == req.Username {
			return nil, fmt.Errorf("username is already taken")
		}
		return nil, fmt.Errorf("email is already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Transactional create: user, account, membership
	var user models.User
	var account models.Account

	err = s.repo.Transaction(ctx, func(txRepo repositories.TxRepository) error {
		// Create user
		user = models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: string(hashedPassword),
		}
		if err := txRepo.User().Create(ctx, &user); err != nil {
			return err
		}

		// Create account
		account = models.Account{
			Name: req.Name,
		}
		if err := txRepo.Account().Create(ctx, &account); err != nil {
			return err
		}

		// Create membership
		membership := models.AccountMember{
			AccountID: account.ID,
			UserID:    user.ID,
			Role:      "owner",
		}
		if err := txRepo.AccountMember().Create(ctx, &membership); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create user and account: %w", err)
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID, account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Theme:     user.Theme,
			CreatedAt: user.CreatedAt,
		},
		Account: &dto.AccountResponse{
			ID:        account.ID,
			Name:      account.Name,
			CreatedAt: account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// generateToken generates a JWT token with user and account ID
func (s *AuthService) generateToken(userID, accountID string) (string, error) {
	claims := jwt.MapClaims{
		"userId":    userID,
		"accountId": accountID,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns user and account ID
func (s *AuthService) ValidateToken(tokenString string) (userID, accountID string, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, _ = claims["userId"].(string)
		accountID, _ = claims["accountId"].(string)
		return userID, accountID, nil
	}

	return "", "", fmt.Errorf("invalid token")
}

// RequestPasswordReset generates a password reset token and saves it to the database
func (s *AuthService) RequestPasswordReset(email string) (*dto.PasswordResetResponse, error) {
	ctx := context.Background()

	// Find user by email
	user, err := s.repo.User().FindByEmail(ctx, email)
	if err != nil {
		// Return success even if user not found (security best practice)
		return &dto.PasswordResetResponse{
			Message: "If an account with that email exists, a password reset link has been sent",
		}, nil
	}

	// Generate secure random token (32 bytes = 64 hex characters)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}
	resetToken := hex.EncodeToString(tokenBytes)

	// Hash the token before storing (same as password hashing)
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(resetToken), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to hash reset token: %w", err)
	}

	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)
	tokenHashStr := string(hashedToken)

	// Save hashed token and expiration to user
	updates := map[string]interface{}{
		"reset_token_hash": tokenHashStr,
		"reset_token_exp":  expiresAt,
	}
	if err := s.repo.User().Update(ctx, user, updates); err != nil {
		return nil, fmt.Errorf("failed to save reset token: %w", err)
	}

	// Send password reset email
	if s.systemEmailSvc != nil {
		html, text := s.systemEmailSvc.RenderPasswordResetEmail(resetToken)
		err := s.systemEmailSvc.SendEmail(SystemEmailOptions{
			To:      user.Email,
			Subject: "Reset Your Password - Yantra",
			HTML:    html,
			Text:    text,
		})
		if err != nil {
			// Log error but don't fail the request (security - don't reveal if email exists)
			// Error is silently ignored to prevent email enumeration
		}
	}

	response := &dto.PasswordResetResponse{
		Message: "If an account with that email exists, a password reset link has been sent",
	}

	// Only include token in development mode for testing
	if s.systemEmailSvc != nil && s.systemEmailSvc.config.Environment == "development" {
		response.Token = resetToken
	}

	return response, nil
}

// ResetPassword resets a user's password using a valid reset token
func (s *AuthService) ResetPassword(resetToken, newPassword string) error {
	ctx := context.Background()

	// Find all users with non-expired reset tokens
	users, err := s.repo.User().FindWithResetToken(ctx)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Try to match the token with each user's hashed token
	var matchedUser *models.User
	for i := range users {
		if err := bcrypt.CompareHashAndPassword([]byte(*users[i].ResetTokenHash), []byte(resetToken)); err == nil {
			matchedUser = &users[i]
			break
		}
	}

	if matchedUser == nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token fields
	updates := map[string]interface{}{
		"password":         string(hashedPassword),
		"reset_token_hash": nil,
		"reset_token_exp":  nil,
	}
	if err := s.repo.User().Update(ctx, matchedUser, updates); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password (requires current password verification)
func (s *AuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	ctx := context.Background()

	// Find the user
	user, err := s.repo.User().FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash the new password
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
