package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patali/yantra/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: jwtSecret,
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
