package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEmailExecutor tests the email executor (with mock service)
func TestEmailExecutor(t *testing.T) {
	// Mock email service
	mockEmailService := NewMockEmailService(false)
	executor := NewEmailExecutor(nil, mockEmailService)

	t.Run("Missing to field", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "email-node",
			NodeConfig:  map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// Email executor returns an error when config is invalid
		if err != nil {
			assert.Contains(t, err.Error(), "invalid email config")
		} else {
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "missing or invalid 'to' field")
		}
	})

	t.Run("Missing subject field", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to": "test@example.com",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// Email executor returns an error when config is invalid
		if err != nil {
			assert.Contains(t, err.Error(), "invalid email config")
		} else {
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "missing or invalid 'subject' field")
		}
	})

	t.Run("Valid email with template variables", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Hello {{input.name}}",
				"body":    "Welcome {{input.name}}!",
			},
			Input: map[string]interface{}{
				"name": "John",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})
}
