package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
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

// MustGetResource retrieves a resource from context and returns an error if not found
// Deprecated: Use RequireResource instead for more idiomatic Go error handling
func MustGetResource(c *gin.Context, key string) (any, error) {
	return RequireResource(c, key)
}
