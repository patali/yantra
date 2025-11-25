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

	t.Run("Email with array template variable", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "User Report",
				"body":    "Processed users:\n{{accumulated}}",
			},
			Input: map[string]interface{}{
				"accumulated": []interface{}{
					map[string]interface{}{"index": 0, "name": "John"},
					map[string]interface{}{"index": 1, "name": "Jane"},
				},
				"iteration_count": 2,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with nested path", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Test",
				"body":    "User: {{data.user.name}}",
			},
			Input: map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"name": "Alice",
					},
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with range iterator", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "User List",
				"body": `Users processed:
{{range .accumulated}}
- {{.name}} (Index: {{.index}})
{{end}}
Total: {{.iteration_count}}`,
			},
			Input: map[string]interface{}{
				"accumulated": []interface{}{
					map[string]interface{}{"index": 0, "name": "John"},
					map[string]interface{}{"index": 1, "name": "Jane"},
					map[string]interface{}{"index": 2, "name": "Bob"},
				},
				"iteration_count": 3,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with if conditional", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Status Update",
				"body": `{{if .isActive}}
Your account is ACTIVE
{{else}}
Your account is INACTIVE
{{end}}`,
			},
			Input: map[string]interface{}{
				"isActive": true,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with custom json function", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Data Report",
				"body": `Raw data:
{{json .data}}`,
			},
			Input: map[string]interface{}{
				"data": map[string]interface{}{
					"name": "Alice",
					"age":  30,
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with nested range", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Order Report",
				"body": `{{range .users}}
User: {{.name}}
Orders:
{{range .orders}}
  - Order #{{.id}}: ${{.total}}
{{end}}
{{end}}`,
			},
			Input: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name": "John",
						"orders": []interface{}{
							map[string]interface{}{"id": 1, "total": 100},
							map[string]interface{}{"id": 2, "total": 200},
						},
					},
					map[string]interface{}{
						"name": "Jane",
						"orders": []interface{}{
							map[string]interface{}{"id": 3, "total": 150},
						},
					},
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Email with template functions", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Test",
				"body": `Name: {{.name | upper}}
Email: {{.email | lower}}`,
			},
			Input: map[string]interface{}{
				"name":  "John Doe",
				"email": "JOHN@EXAMPLE.COM",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.Output["sent"].(bool))
	})

	t.Run("Backward compatibility - simple variables without dot", func(t *testing.T) {
		mockEmailService.ShouldFail = false
		execCtx := ExecutionContext{
			NodeID: "email-node",
			NodeConfig: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Test",
				"body":    "Name: {{name}}, Count: {{count}}",
			},
			Input: map[string]interface{}{
				"name":  "Alice",
				"count": 42,
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
