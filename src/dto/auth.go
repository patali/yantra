package dto

import "time"

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents the request to login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SignupWithAccountRequest represents the request to signup with account creation
type SignupWithAccountRequest struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RequestPasswordResetRequest represents the request to reset password
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the request to reset password with token
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// PasswordResetResponse represents the response for password reset
type PasswordResetResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // Only for development/testing
}

// ChangePasswordRequest represents the request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

// UserResponse represents the user response
type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Theme     string    `json:"theme"`
	CreatedAt time.Time `json:"createdAt"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token   string           `json:"token"`
	User    UserResponse     `json:"user"`
	Account *AccountResponse `json:"account,omitempty"`
}
