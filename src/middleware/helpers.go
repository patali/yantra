package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BindJSON binds JSON request body to the given struct and returns error response if binding fails
// This reduces repeated JSON binding error handling code
//
// Usage:
//
//	var req dto.CreateWorkflowRequest
//	if !middleware.BindJSON(c, &req) {
//	    return // Error response already sent
//	}
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

// RespondSuccess sends a JSON success response
//
// Usage:
//
//	middleware.RespondSuccess(c, http.StatusOK, gin.H{"id": id})
func RespondSuccess(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, data)
}

// RespondNotFound sends a 404 Not Found response
func RespondNotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{"error": message})
}

// RespondUnauthorized sends a 401 Unauthorized response
func RespondUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": message})
}

// RespondForbidden sends a 403 Forbidden response
func RespondForbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, gin.H{"error": message})
}

// RespondBadRequest sends a 400 Bad Request response
func RespondBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

// RespondInternalError sends a 500 Internal Server Error response
func RespondInternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": message})
}
