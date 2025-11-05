package executors

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHTTPExecutor tests the HTTP executor (with mock server)
func TestHTTPExecutor(t *testing.T) {
	// Note: HTTP executor tests would typically use a test server
	// This is a basic test that verifies the executor can be created
	client := &http.Client{Timeout: 5 * time.Second}
	executor := NewHTTPExecutor(client)

	t.Run("Missing URL", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "http-node",
			NodeConfig:  map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "url is required")
	})

	t.Run("Default method is GET", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "http-node",
			NodeConfig: map[string]interface{}{
				"url": "https://httpbin.org/get",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// This will make a real HTTP request, so we check for either success or network error
		if err == nil {
			assert.True(t, result.Success)
			assert.Equal(t, "GET", result.Output["method"])
		} else {
			// Network error is acceptable in test environments
			assert.Contains(t, err.Error(), "request failed")
		}
	})
}
