package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/middleware"
)

type ExamplesController struct{}

func NewExamplesController() *ExamplesController {
	return &ExamplesController{}
}

// RegisterRoutes registers example API routes
func (ctrl *ExamplesController) RegisterRoutes(rg *gin.RouterGroup) {
	examples := rg.Group("/examples")
	{
		// Public endpoint for example workflows to use
		examples.GET("/time", ctrl.GetCurrentTime)
	}
}

// GetCurrentTime returns the current system time in various formats
// GET /api/examples/time
// This endpoint is public and can be used by example workflows for testing
// Query parameters are echoed back in the "params" field for testing purposes
func (ctrl *ExamplesController) GetCurrentTime(c *gin.Context) {
	now := time.Now()

	response := gin.H{
		"timestamp":      now.Unix(),
		"timestamp_ms":   now.UnixMilli(),
		"timestamp_nano": now.UnixNano(),
		"iso8601":        now.Format(time.RFC3339),
		"iso8601_nano":   now.Format(time.RFC3339Nano),
		"rfc1123":        now.Format(time.RFC1123),
		"unix_time":      now.Unix(),
		"date":           now.Format("2006-01-02"),
		"time":           now.Format("15:04:05"),
		"datetime":       now.Format("2006-01-02 15:04:05"),
		"timezone":       now.Format("MST"),
		"utc": gin.H{
			"timestamp": now.UTC().Unix(),
			"iso8601":   now.UTC().Format(time.RFC3339),
			"date":      now.UTC().Format("2006-01-02"),
			"time":      now.UTC().Format("15:04:05"),
			"datetime":  now.UTC().Format("2006-01-02 15:04:05"),
		},
	}

	// Echo back any query parameters for testing
	queryParams := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			queryParams[key] = values[0]
		} else {
			queryParams[key] = values
		}
	}

	// Only include params if there are any
	if len(queryParams) > 0 {
		response["params"] = queryParams
	}

	middleware.RespondSuccess(c, http.StatusOK, response)
}
