package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/dto"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/services"
	"gorm.io/gorm"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Register routes for auth controller
func (ctrl *AuthController) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/signup", ctrl.SignupWithAccount)   // Node.js compatible endpoint
		auth.POST("/register", ctrl.SignupWithAccount) // Alternative endpoint
		auth.POST("/login", ctrl.Login)
		auth.GET("/me", middleware.AuthMiddleware(ctrl.authService), ctrl.GetMe)
		auth.POST("/request-password-reset", ctrl.RequestPasswordReset)                                 // Request password reset
		auth.POST("/reset-password", ctrl.ResetPassword)                                                // Reset password with token
		auth.POST("/change-password", middleware.AuthMiddleware(ctrl.authService), ctrl.ChangePassword) // Change password (authenticated)
	}
}

// SignupWithAccount handles user registration with account creation
// POST /api/auth/register
func (ctrl *AuthController) SignupWithAccount(c *gin.Context) {
	var req dto.SignupWithAccountRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	response, err := ctrl.authService.SignupWithAccount(req)
	if err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, response)
}

// Login handles user authentication
// POST /api/auth/login
func (ctrl *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	response, err := ctrl.authService.Login(req)
	if err != nil {
		middleware.RespondUnauthorized(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, response)
}

// GetMe returns the current user information
// GET /api/auth/me
func (ctrl *AuthController) GetMe(c *gin.Context) {
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}

	// Fetch user from database
	var user models.User
	if err := c.MustGet("db").(*gorm.DB).First(&user, "id = ?", userID).Error; err != nil {
		middleware.RespondNotFound(c, "User not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Theme:     user.Theme,
		CreatedAt: user.CreatedAt,
	})
}

// ValidateToken validates the current JWT token
// POST /api/auth/validate
func (ctrl *AuthController) ValidateToken(c *gin.Context) {
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"valid":     true,
		"userId":    userID,
		"accountId": accountID,
	})
}

// RequestPasswordReset generates and sends a password reset token
// POST /api/auth/request-password-reset
func (ctrl *AuthController) RequestPasswordReset(c *gin.Context) {
	var req dto.RequestPasswordResetRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	response, err := ctrl.authService.RequestPasswordReset(req.Email)
	if err != nil {
		middleware.RespondInternalError(c, "Failed to process password reset request")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, response)
}

// ResetPassword resets a user's password using a valid reset token
// POST /api/auth/reset-password
func (ctrl *AuthController) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := ctrl.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ChangePassword changes a user's password (requires authentication and current password)
// POST /api/auth/change-password
func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}

	var req dto.ChangePasswordRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := ctrl.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Password changed successfully"})
}
