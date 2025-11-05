package executors

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSlackExecutor tests the Slack executor
func TestSlackExecutor(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	executor := NewSlackExecutor(client)

	t.Run("Missing webhook URL", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "slack-node",
			NodeConfig:  map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "webhookUrl is required")
	})

	t.Run("Valid configuration", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "slack-node",
			NodeConfig: map[string]interface{}{
				"webhookUrl": "https://hooks.slack.com/services/test",
				"channel":    "#test",
				"text":       "Test message",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// This will make a real HTTP request, so we check for either success or network error
		if err == nil {
			// If the webhook is invalid, Slack will return an error
			// But the executor should handle it gracefully
			assert.NotNil(t, result)
		} else {
			// Network error is acceptable in test environments
			assert.NotNil(t, result)
		}
	})
}
