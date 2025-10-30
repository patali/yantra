package services

import (
	"os"
	"testing"
	"time"

	"github.com/patali/yantra/src/db/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use test database from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/yantra_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(testDBURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v\nMake sure PostgreSQL is running and TEST_DATABASE_URL is set", err)
	}

	// Clean up test data before running tests
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Account{}, &models.AccountMember{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Set PostgreSQL UUID defaults
	db.Exec("ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid()")
	db.Exec("ALTER TABLE accounts ALTER COLUMN id SET DEFAULT gen_random_uuid()")

	// Clean up after test
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

func TestAuthService_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	user, err := authService.CreateUser(req, nil)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, user.ID)
}

func TestAuthService_CreateUser_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// Create first user
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Try to create second user with same email
	req.Username = "testuser2"
	_, err = authService.CreateUser(req, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAuthService_Login(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Login
	loginReq := dto.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	loginResp, err := authService.Login(loginReq)

	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.Token)
	assert.Equal(t, "testuser", loginResp.User.Username)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Login with wrong password
	loginReq := dto.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	_, err = authService.Login(loginReq)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestAuthService_SignupWithAccount(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	req := dto.SignupWithAccountRequest{
		Name:     "Test Account",
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	loginResp, err := authService.SignupWithAccount(req)

	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.Token)
	assert.Equal(t, "testuser", loginResp.User.Username)
	assert.NotNil(t, loginResp.Account)
	assert.Equal(t, "Test Account", loginResp.Account.Name)
}

func TestUserService_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)
	userService := NewUserService(db)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "oldpassword123",
	}
	user, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Update password
	err = userService.UpdatePassword(user.ID, "oldpassword123", "newpassword123")
	assert.NoError(t, err)

	// Verify new password works
	loginReq := dto.LoginRequest{
		Username: "testuser",
		Password: "newpassword123",
	}
	loginResp, err := authService.Login(loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.Token)
}

func TestUserService_UpdatePassword_WrongCurrentPassword(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)
	userService := NewUserService(db)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "oldpassword123",
	}
	user, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Try to update password with wrong current password
	err = userService.UpdatePassword(user.ID, "wrongpassword", "newpassword123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect")
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	user, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Request password reset
	resetResp, err := authService.RequestPasswordReset("test@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, resetResp)
	assert.Contains(t, resetResp.Message, "password reset")

	// Verify token was saved to database
	var updatedUser models.User
	db.First(&updatedUser, "id = ?", user.ID)
	assert.NotNil(t, updatedUser.ResetTokenHash)
	assert.NotNil(t, updatedUser.ResetTokenExp)
}

func TestAuthService_RequestPasswordReset_NonExistentEmail(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Request password reset for non-existent email (should still return success for security)
	resetResp, err := authService.RequestPasswordReset("nonexistent@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, resetResp)
	assert.Contains(t, resetResp.Message, "password reset")
}

func TestAuthService_ResetPassword(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "oldpassword123",
	}
	user, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Request password reset
	_, err = authService.RequestPasswordReset("test@example.com")
	assert.NoError(t, err)

	// Get the reset token from database directly for testing
	var userWithToken models.User
	db.First(&userWithToken, "id = ?", user.ID)
	assert.NotNil(t, userWithToken.ResetTokenHash)

	// Since the token is hashed, we need to use a mock token
	// For this test, we'll manually create a known token
	testToken := "test-reset-token-123456789012345678901234567890123456789012345678"
	hashedToken, _ := bcrypt.GenerateFromPassword([]byte(testToken), 10)
	hashedTokenStr := string(hashedToken)
	expiresAt := time.Now().Add(1 * time.Hour)
	db.Model(&userWithToken).Updates(map[string]interface{}{
		"reset_token_hash": hashedTokenStr,
		"reset_token_exp":  expiresAt,
	})

	// Reset password with token
	err = authService.ResetPassword(testToken, "newpassword456")
	assert.NoError(t, err)

	// Verify new password works
	loginReq := dto.LoginRequest{
		Username: "testuser",
		Password: "newpassword456",
	}
	loginResp, err := authService.Login(loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.Token)

	// Verify reset token was cleared
	var updatedUser models.User
	db.First(&updatedUser, "id = ?", user.ID)
	assert.Nil(t, updatedUser.ResetTokenHash)
	assert.Nil(t, updatedUser.ResetTokenExp)
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "oldpassword123",
	}
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Request password reset (but don't use the token)
	_, err = authService.RequestPasswordReset("test@example.com")
	assert.NoError(t, err)

	// Try to reset with invalid token
	err = authService.ResetPassword("invalidtoken123", "newpassword456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestAuthService_ResetPassword_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret", nil)

	// Create user
	req := dto.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "oldpassword123",
	}
	user, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Request password reset
	_, err = authService.RequestPasswordReset("test@example.com")
	assert.NoError(t, err)

	// Create a test token and set it to expired
	testToken := "expired-reset-token-123456789012345678901234567890123456789012"
	hashedToken, _ := bcrypt.GenerateFromPassword([]byte(testToken), 10)
	hashedTokenStr := string(hashedToken)
	pastTime := user.CreatedAt // Use a time in the past

	var updatedUser models.User
	db.First(&updatedUser, "id = ?", user.ID)
	db.Model(&updatedUser).Updates(map[string]interface{}{
		"reset_token_hash": hashedTokenStr,
		"reset_token_exp":  pastTime,
	})

	// Try to reset with expired token
	err = authService.ResetPassword(testToken, "newpassword456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}
