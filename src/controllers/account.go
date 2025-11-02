package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/dto"
	"github.com/patali/yantra/src/services"
)

type AccountController struct {
	accountService *services.AccountService
}

func NewAccountController(accountService *services.AccountService) *AccountController {
	return &AccountController{
		accountService: accountService,
	}
}

// RegisterRoutes registers account routes
func (ctrl *AccountController) RegisterRoutes(rg *gin.RouterGroup, authService *services.AuthService) {
	accounts := rg.Group("/accounts")
	accounts.Use(middleware.AuthMiddleware(authService))
	{
		accounts.GET("/", ctrl.ListMyAccounts)
		accounts.POST("/", ctrl.CreateAccount)
		accounts.GET("/:id", ctrl.GetAccountByID)
		accounts.PUT("/:id", ctrl.UpdateAccount)
		accounts.POST("/:id/members", ctrl.AddMember)
		accounts.DELETE("/:id/members/:userId", ctrl.RemoveMember)
	}
}

// ListMyAccounts returns all accounts the user is a member of
// GET /api/accounts
func (ctrl *AccountController) ListMyAccounts(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	accounts, err := ctrl.accountService.ListMyAccounts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

// CreateAccount creates a new account with the user as owner
// POST /api/accounts
func (ctrl *AccountController) CreateAccount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := ctrl.accountService.CreateAccount(req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// GetAccountByID returns an account by ID
// GET /api/accounts/:id
func (ctrl *AccountController) GetAccountByID(c *gin.Context) {
	// SECURITY: Get user ID from auth middleware
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	account, err := ctrl.accountService.GetAccountByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	// SECURITY: Verify user is a member of this account
	isMember, err := ctrl.accountService.IsUserMemberOfAccount(userID, id)
	if err != nil || !isMember {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or access denied"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// UpdateAccount updates account details
// PUT /api/accounts/:id
func (ctrl *AccountController) UpdateAccount(c *gin.Context) {
	// SECURITY: Get user ID from auth middleware
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// SECURITY: Verify user is a member of this account
	isMember, err := ctrl.accountService.IsUserMemberOfAccount(userID, id)
	if err != nil || !isMember {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or access denied"})
		return
	}

	// SECURITY: Only admin or owner can update account details
	role, err := ctrl.accountService.GetUserRoleInAccount(userID, id)
	if err != nil || (role != "admin" && role != "owner") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or owners can update account details"})
		return
	}

	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.accountService.UpdateAccount(id, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account updated successfully"})
}

// AddMember adds a user to an account
// POST /api/accounts/:id/members
func (ctrl *AccountController) AddMember(c *gin.Context) {
	// SECURITY: Get user ID from auth middleware
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	accountID := c.Param("id")

	// SECURITY: Verify user is a member of this account
	isMember, err := ctrl.accountService.IsUserMemberOfAccount(userID, accountID)
	if err != nil || !isMember {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or access denied"})
		return
	}

	// SECURITY: Only admin or owner can add members
	role, err := ctrl.accountService.GetUserRoleInAccount(userID, accountID)
	if err != nil || (role != "admin" && role != "owner") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or owners can add members"})
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberRole := req.Role
	if memberRole == "" {
		memberRole = "member"
	}

	if err := ctrl.accountService.AddMember(accountID, req.UserID, memberRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// RemoveMember removes a user from an account
// DELETE /api/accounts/:id/members/:userId
func (ctrl *AccountController) RemoveMember(c *gin.Context) {
	// SECURITY: Get user ID from auth middleware
	currentUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	accountID := c.Param("id")
	targetUserID := c.Param("userId")

	// SECURITY: Verify user is a member of this account
	isMember, err := ctrl.accountService.IsUserMemberOfAccount(currentUserID, accountID)
	if err != nil || !isMember {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or access denied"})
		return
	}

	// SECURITY: Only admin or owner can remove members (unless removing themselves)
	if currentUserID != targetUserID {
		role, err := ctrl.accountService.GetUserRoleInAccount(currentUserID, accountID)
		if err != nil || (role != "admin" && role != "owner") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or owners can remove other members"})
			return
		}
	}

	// SECURITY: Prevent removing the last owner
	if currentUserID == targetUserID {
		targetRole, err := ctrl.accountService.GetUserRoleInAccount(targetUserID, accountID)
		if err == nil && targetRole == "owner" {
			// Check if this is the last owner
			// This is a basic check - you might want a more robust implementation
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove yourself if you are the owner. Transfer ownership first."})
			return
		}
	}

	if err := ctrl.accountService.RemoveMember(accountID, targetUserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
