package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/dto"
	"github.com/patali/yantra/src/services"
)

type UserController struct {
	userService *services.UserService
	authService *services.AuthService
}

func NewUserController(userService *services.UserService, authService *services.AuthService) *UserController {
	return &UserController{
		userService: userService,
		authService: authService,
	}
}

// RegisterRoutes registers user routes
func (ctrl *UserController) RegisterRoutes(rg *gin.RouterGroup, authService *services.AuthService) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(authService))
	{
		users.GET("/", ctrl.GetAllUsers)
		users.GET("/me", ctrl.GetCurrentUser)           // Frontend compatible endpoint
		users.POST("", ctrl.CreateUser)                 // Frontend endpoint for creating users
		users.GET("/:id", ctrl.GetUserById)
		users.POST("/theme", ctrl.UpdateThemeFromToken) // Frontend compatible endpoint
		users.PATCH("/:id/theme", ctrl.UpdateTheme)     // Alternative endpoint
		users.POST("/password", ctrl.UpdatePassword)    // Update password endpoint
		users.DELETE("/:id", ctrl.DeleteUser)
		users.POST("/invite", ctrl.InviteUser)
	}
}

// GetAllUsers returns all users who share accounts with the current user
// GET /api/users
func (ctrl *UserController) GetAllUsers(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	users, err := ctrl.userService.GetAllUsersForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetCurrentUser returns the current user's information
// GET /api/users/me
func (ctrl *UserController) GetCurrentUser(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := ctrl.userService.GetUserById(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user (frontend endpoint)
// POST /api/users
func (ctrl *UserController) CreateUser(c *gin.Context) {
	currentUserID, _ := middleware.GetUserID(c)

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.authService.CreateUser(req, &currentUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUserById returns a user by ID (only if they share an account)
// GET /api/users/:id
func (ctrl *UserController) GetUserById(c *gin.Context) {
	id := c.Param("id")
	currentUserID, _ := middleware.GetUserID(c)

	user, err := ctrl.userService.GetUserByIdInSameAccounts(id, currentUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateThemeFromToken updates the current user's theme using the JWT token
// POST /api/users/theme (Node.js compatible)
func (ctrl *UserController) UpdateThemeFromToken(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.UpdateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.userService.UpdateTheme(userID, req.Theme)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateTheme updates a user's theme preference
// PATCH /api/users/:id/theme
func (ctrl *UserController) UpdateTheme(c *gin.Context) {
	id := c.Param("id")
	userID, _ := middleware.GetUserID(c)

	// Users can only update their own theme
	if id != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update another user's theme"})
		return
	}

	var req dto.UpdateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.userService.UpdateTheme(id, req.Theme)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user (only if they share an account)
// DELETE /api/users/:id
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	currentUserID, _ := middleware.GetUserID(c)

	if err := ctrl.userService.DeleteUserInSameAccounts(id, currentUserID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// InviteUser creates a new user and adds them to the inviter's accounts
// POST /api/users/invite
func (ctrl *UserController) InviteUser(c *gin.Context) {
	currentUserID, _ := middleware.GetUserID(c)

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.authService.CreateUser(req, &currentUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdatePassword updates the current user's password
// POST /api/users/password
func (ctrl *UserController) UpdatePassword(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.userService.UpdatePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
