package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandler is a middleware for centralized error handling
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error
			log.Printf("❌ Error: %v", err.Err)

			// Determine status code
			statusCode := http.StatusInternalServerError
			if c.Writer.Status() != http.StatusOK {
				statusCode = c.Writer.Status()
			}

			// Send error response
			c.JSON(statusCode, gin.H{
				"error": err.Error(),
			})
		}
	}
}

// Recovery middleware for panic recovery
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}

// CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
