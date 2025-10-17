package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/internal/middleware"
	"github.com/patali/yantra/internal/services"
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

	var req services.CreateAccountRequest
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
	id := c.Param("id")

	account, err := ctrl.accountService.GetAccountByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// UpdateAccount updates account details
// PUT /api/accounts/:id
func (ctrl *AccountController) UpdateAccount(c *gin.Context) {
	id := c.Param("id")

	var req services.CreateAccountRequest
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
	accountID := c.Param("id")

	var req services.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := req.Role
	if role == "" {
		role = "member"
	}

	if err := ctrl.accountService.AddMember(accountID, req.UserID, role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// RemoveMember removes a user from an account
// DELETE /api/accounts/:id/members/:userId
func (ctrl *AccountController) RemoveMember(c *gin.Context) {
	accountID := c.Param("id")
	userID := c.Param("userId")

	if err := ctrl.accountService.RemoveMember(accountID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
