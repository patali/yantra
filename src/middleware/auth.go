package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/services"
)

// AuthMiddleware validates JWT tokens and sets user context
// Supports both Authorization header and token query parameter (for SSE/EventSource which doesn't support custom headers)
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// If no token from header, try query parameter (for SSE endpoints)
		if token == "" {
			token = c.Query("token")
		}

		// If still no token, return unauthorized
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header or token query parameter required"})
			c.Abort()
			return
		}

		// Validate token
		userID, accountID, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user and account in context
		c.Set("userId", userID)
		c.Set("accountId", accountID)

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("userId")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// GetAccountID extracts account ID from context
func GetAccountID(c *gin.Context) (string, bool) {
	accountID, exists := c.Get("accountId")
	if !exists {
		return "", false
	}
	return accountID.(string), true
}
