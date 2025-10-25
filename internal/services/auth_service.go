package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patali/yantra/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db               *gorm.DB
	jwtSecret        string
	systemEmailSvc   *SystemEmailService
}

func NewAuthService(db *gorm.DB, jwtSecret string, systemEmailSvc *SystemEmailService) *AuthService {
	return &AuthService{
		db:             db,
		jwtSecret:      jwtSecret,
		systemEmailSvc: systemEmailSvc,
	}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignupWithAccountRequest struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type PasswordResetResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // Only for development/testing
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Theme     string    `json:"theme"`
	CreatedAt time.Time `json:"createdAt"`
}

type LoginResponse struct {
	Token   string           `json:"token"`
	User    UserResponse     `json:"user"`
	Account *AccountResponse `json:"account,omitempty"`
}

// CreateUser creates a new user (for invitations)
func (s *AuthService) CreateUser(req CreateUserRequest, createdBy *string) (*UserResponse, error) {
	// Check if user already exists
	var existingUser models.User
	result := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser)
	if result.Error == nil {
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
		var memberships []models.AccountMember
		s.db.Where("user_id = ?", *createdBy).Find(&memberships)

		// Create user with memberships in a transaction
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&user).Error; err != nil {
				return err
			}

			for _, m := range memberships {
				membership := models.AccountMember{
					AccountID: m.AccountID,
					UserID:    user.ID,
					Role:      "member",
				}
				if err := tx.Create(&membership).Error; err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		if err := s.db.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	// Find user
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Get user's primary account (first account they joined)
	var membership models.AccountMember
	if err := s.db.Where("user_id = ?", user.ID).First(&membership).Error; err != nil {
		return nil, fmt.Errorf("user is not associated with any account")
	}
	accountID := membership.AccountID

	// Generate JWT token
	token, err := s.generateToken(user.ID, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &LoginResponse{
		Token: token,
		User: UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Theme:     user.Theme,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

// SignupWithAccount creates a new user and account (for first-time signup)
func (s *AuthService) SignupWithAccount(req SignupWithAccountRequest) (*LoginResponse, error) {
	// Check if user already exists
	var existingUser models.User
	result := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser)
	if result.Error == nil {
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

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create user
		user = models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: string(hashedPassword),
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create account
		account = models.Account{
			Name: req.Name,
		}
		if err := tx.Create(&account).Error; err != nil {
			return err
		}

		// Create membership
		membership := models.AccountMember{
			AccountID: account.ID,
			UserID:    user.ID,
			Role:      "owner",
		}
		if err := tx.Create(&membership).Error; err != nil {
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

	return &LoginResponse{
		Token: token,
		User: UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Theme:     user.Theme,
			CreatedAt: user.CreatedAt,
		},
		Account: &AccountResponse{
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
func (s *AuthService) RequestPasswordReset(email string) (*PasswordResetResponse, error) {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		// Return success even if user not found (security best practice)
		return &PasswordResetResponse{
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
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"reset_token_hash": tokenHashStr,
		"reset_token_exp":  expiresAt,
	}).Error; err != nil {
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
			fmt.Printf("Warning: Failed to send password reset email to %s: %v\n", user.Email, err)
		}
	}

	response := &PasswordResetResponse{
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
	// Find all users with non-expired reset tokens
	now := time.Now()
	var users []models.User
	if err := s.db.Where("reset_token_hash IS NOT NULL AND reset_token_exp > ?", now).Find(&users).Error; err != nil {
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
	if err := s.db.Model(matchedUser).Updates(map[string]interface{}{
		"password":         string(hashedPassword),
		"reset_token_hash": nil,
		"reset_token_exp":  nil,
	}).Error; err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password (requires current password verification)
func (s *AuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	// Find the user
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
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
	if err := s.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
