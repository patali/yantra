package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/internal/middleware"
	"github.com/patali/yantra/internal/models"
	"github.com/patali/yantra/internal/services"
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
		auth.POST("/signup", ctrl.SignupWithAccount)                         // Node.js compatible endpoint
		auth.POST("/register", ctrl.SignupWithAccount)                       // Alternative endpoint
		auth.POST("/login", ctrl.Login)
		auth.GET("/me", middleware.AuthMiddleware(ctrl.authService), ctrl.GetMe)
		auth.POST("/request-password-reset", ctrl.RequestPasswordReset)     // Request password reset
		auth.POST("/reset-password", ctrl.ResetPassword)                     // Reset password with token
	}
}

// SignupWithAccount handles user registration with account creation
// POST /api/auth/register
func (ctrl *AuthController) SignupWithAccount(c *gin.Context) {
	var req services.SignupWithAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := ctrl.authService.SignupWithAccount(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login handles user authentication
// POST /api/auth/login
func (ctrl *AuthController) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := ctrl.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMe returns the current user information
// GET /api/auth/me
func (ctrl *AuthController) GetMe(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user from database
	var user models.User
	if err := c.MustGet("db").(*gorm.DB).First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, services.UserResponse{
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
	userID, _ := middleware.GetUserID(c)
	accountID, _ := middleware.GetAccountID(c)

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"userId":    userID,
		"accountId": accountID,
	})
}

// RequestPasswordReset generates and sends a password reset token
// POST /api/auth/request-password-reset
func (ctrl *AuthController) RequestPasswordReset(c *gin.Context) {
	var req services.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := ctrl.authService.RequestPasswordReset(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password reset request"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ResetPassword resets a user's password using a valid reset token
// POST /api/auth/reset-password
func (ctrl *AuthController) ResetPassword(c *gin.Context) {
	var req services.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
