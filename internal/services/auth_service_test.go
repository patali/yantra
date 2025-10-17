package services

import (
	"testing"

	"github.com/patali/yantra/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Account{}, &models.AccountMember{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestAuthService_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret")

	req := CreateUserRequest{
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
	authService := NewAuthService(db, "test-secret")

	req := CreateUserRequest{
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
	authService := NewAuthService(db, "test-secret")

	// Create user
	req := CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Login
	loginReq := LoginRequest{
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
	authService := NewAuthService(db, "test-secret")

	// Create user
	req := CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	_, err := authService.CreateUser(req, nil)
	assert.NoError(t, err)

	// Login with wrong password
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	_, err = authService.Login(loginReq)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestAuthService_SignupWithAccount(t *testing.T) {
	db := setupTestDB(t)
	authService := NewAuthService(db, "test-secret")

	req := SignupWithAccountRequest{
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
