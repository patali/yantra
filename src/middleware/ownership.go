package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/services"
)

// OwnershipChecker is a function type that checks if a resource belongs to an account
// It should return nil if the resource is owned, or an error if not
type OwnershipChecker func(resourceID, accountID string) error

// RequireResourceOwnership creates a middleware that verifies resource ownership
// This middleware extracts the resource ID from URL params and verifies it belongs to the authenticated account
//
// Parameters:
//   - paramName: The URL parameter name containing the resource ID (e.g., "id", "workflowId")
//   - checker: A function that verifies ownership
//   - errorMessage: The error message to return if ownership check fails
//
// Usage in RegisterRoutes:
//
//	workflows.GET("/:id",
//	    middleware.RequireResourceOwnership("id", workflowService.CheckOwnership, "Workflow not found"),
//	    ctrl.GetWorkflowById)
//
// The middleware will:
// 1. Extract accountID from context (set by AuthMiddleware)
// 2. Extract resource ID from URL params
// 3. Call the checker function to verify ownership
// 4. Return 404 if ownership check fails
// 5. Continue to next handler if ownership is verified
func RequireResourceOwnership(paramName string, checker OwnershipChecker, errorMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get account ID from context
		accountID, exists := GetAccountID(c)
		if !exists {
			RespondUnauthorized(c, "Unauthorized")
			c.Abort()
			return
		}

		// Get resource ID from URL params
		resourceID := c.Param(paramName)
		if resourceID == "" {
			RespondBadRequest(c, paramName+" is required")
			c.Abort()
			return
		}

		// Check ownership
		if err := checker(resourceID, accountID); err != nil {
			RespondNotFound(c, errorMessage)
			c.Abort()
			return
		}

		// Ownership verified, continue to handler
		c.Next()
	}
}

// SetResource stores a resource in the gin context for later retrieval
// This is useful when you fetch a resource in middleware and want to use it in the handler
//
// Usage:
//
//	middleware.SetResource(c, "workflow", workflow)
func SetResource(c *gin.Context, key string, value any) {
	c.Set(key, value)
}

// GetResource retrieves a resource from the gin context
//
// Usage:
//
//	workflow, exists := middleware.GetResource(c, "workflow")
//	if !exists {
//	    // handle error
//	}
func GetResource(c *gin.Context, key string) (any, bool) {
	return c.Get(key)
}

// RequireResource retrieves a resource from context and returns an error if not found
// Use this when you need a resource that should have been set by middleware
//
// Usage:
//
//	workflow, err := middleware.RequireResource(c, "workflow")
//	if err != nil {
//	    middleware.RespondInternalError(c, "Workflow not found in context")
//	    return
//	}
//	wf := workflow.(*models.Workflow)
func RequireResource(c *gin.Context, key string) (any, error) {
	value, exists := c.Get(key)
	if !exists {
		return nil, errors.New("resource '" + key + "' not found in context")
	}
	return value, nil
}

// RequireAccountMembership creates a middleware that verifies the authenticated user is a member of the account
// Optionally checks if the user has one of the required roles
// This middleware extracts the account ID from URL params and verifies the user is a member
//
// Parameters:
//   - accountService: The account service to use for membership checks
//   - paramName: The URL parameter name containing the account ID (e.g., "id")
//   - allowedRoles: Optional variadic list of allowed roles (e.g., "admin", "owner"). If provided, user must have one of these roles
//
// Usage in RegisterRoutes:
//
//	// Just check membership
//	accounts.GET("/:id",
//	    middleware.RequireAccountMembership(accountService, "id"),
//	    ctrl.GetAccountByID)
//
//	// Check membership AND require admin or owner role
//	accounts.PUT("/:id",
//	    middleware.RequireAccountMembership(accountService, "id", "admin", "owner"),
//	    ctrl.UpdateAccount)
//
// The middleware will:
// 1. Extract userID from context (set by AuthMiddleware)
// 2. Extract account ID from URL params
// 3. Check if user is a member of the account
// 4. If allowedRoles provided, check if user has one of the required roles
// 5. Store user's role in context with key "accountRole" for handler access
// 6. Return 404 if not a member, 403 if role check fails
// 7. Continue to next handler if all checks pass
func RequireAccountMembership(accountService *services.AccountService, paramName string, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userID, exists := GetUserID(c)
		if !exists {
			RespondUnauthorized(c, "Unauthorized")
			c.Abort()
			return
		}

		// Get account ID from URL params
		accountID := c.Param(paramName)
		if accountID == "" {
			RespondBadRequest(c, paramName+" is required")
			c.Abort()
			return
		}

		// Check membership
		isMember, err := accountService.IsUserMemberOfAccount(userID, accountID)
		if err != nil || !isMember {
			RespondNotFound(c, "Account not found or access denied")
			c.Abort()
			return
		}

		// If role check is required, verify user has one of the allowed roles
		if len(allowedRoles) > 0 {
			role, err := accountService.GetUserRoleInAccount(userID, accountID)
			if err != nil {
				RespondForbidden(c, "Unable to verify user role")
				c.Abort()
				return
			}

			// Check if user's role is in the allowed list
			roleAllowed := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				// Build error message
				roleList := ""
				if len(allowedRoles) == 1 {
					roleList = allowedRoles[0]
				} else if len(allowedRoles) == 2 {
					roleList = allowedRoles[0] + " or " + allowedRoles[1]
				} else {
					roleList = allowedRoles[0]
					for i := 1; i < len(allowedRoles)-1; i++ {
						roleList += ", " + allowedRoles[i]
					}
					roleList += ", or " + allowedRoles[len(allowedRoles)-1]
				}
				RespondForbidden(c, "Only "+roleList+" can perform this action")
				c.Abort()
				return
			}

			// Store role in context for handler access
			c.Set("accountRole", role)
		} else {
			// Even if no role check, store role in context for convenience
			role, _ := accountService.GetUserRoleInAccount(userID, accountID)
			c.Set("accountRole", role)
		}

		// Membership (and role if required) verified, continue to handler
		c.Next()
	}
}

// GetAccountRole extracts the account role from context (set by RequireAccountMembership middleware)
func GetAccountRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("accountRole")
	if !exists {
		return "", false
	}
	return role.(string), true
}
