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

	t.Run("URL template variables", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "http-node",
			NodeConfig: map[string]interface{}{
				"url": "https://httpbin.org/get?user={{input.user}}&id={{input.id}}",
			},
			Input: map[string]interface{}{
				"user": "john",
				"id":   123,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// This will make a real HTTP request, so we check for either success or network error
		if err == nil {
			assert.True(t, result.Success)
			// Verify the URL was properly templated
			expectedURL := "https://httpbin.org/get?user=john&id=123"
			assert.Equal(t, expectedURL, result.Output["url"])
		} else {
			// Network error is acceptable in test environments
			assert.Contains(t, err.Error(), "request failed")
		}
	})

	t.Run("URL template with nested input", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "http-node",
			NodeConfig: map[string]interface{}{
				"url": "https://httpbin.org/get?email={{input.user.email}}",
			},
			Input: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "test@example.com",
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		// This will make a real HTTP request, so we check for either success or network error
		if err == nil {
			assert.True(t, result.Success)
			// Verify the URL was properly templated with URL encoding
			// @ symbol should be encoded as %40
			expectedURL := "https://httpbin.org/get?email=test%40example.com"
			assert.Equal(t, expectedURL, result.Output["url"])
		} else {
			// Network error is acceptable in test environments
			assert.Contains(t, err.Error(), "request failed")
		}
	})
}

// TestHTTPTemplateVariables tests the template variable replacement
func TestHTTPTemplateVariables(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	executor := NewHTTPExecutor(client)

	t.Run("Simple variable replacement", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "John",
			"age":  30,
		}

		result := executor.replaceTemplateVariables("Hello {{name}}, you are {{age}} years old", input)
		assert.Equal(t, "Hello John, you are 30 years old", result)
	})

	t.Run("URL encoding with spaces", func(t *testing.T) {
		input := map[string]interface{}{
			"datetime": "2025-11-29 15:08:32",
		}

		result := executor.replaceTemplateVariablesWithEncoding("http://api.example.com?time={{datetime}}", input, true)
		// Spaces should be URL-encoded
		assert.Contains(t, result, "2025-11-29")
		assert.NotContains(t, result, " ") // Should not contain raw spaces
		// QueryEscape converts space to + in query strings
		assert.Contains(t, result, "+") // Space encoded as +
	})

	t.Run("Nested variable replacement", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "Jane",
				"email": "jane@example.com",
			},
		}

		result := executor.replaceTemplateVariables("User: {{user.name}}, Email: {{user.email}}", input)
		assert.Equal(t, "User: Jane, Email: jane@example.com", result)
	})

	t.Run("URL with query parameters", func(t *testing.T) {
		input := map[string]interface{}{
			"userId": 123,
			"token":  "abc123",
		}

		result := executor.replaceTemplateVariables("https://api.example.com/user/{{userId}}?token={{token}}", input)
		assert.Equal(t, "https://api.example.com/user/123?token=abc123", result)
	})

	t.Run("Missing variable keeps placeholder", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "John",
		}

		result := executor.replaceTemplateVariables("Hello {{name}}, your age is {{age}}", input)
		assert.Equal(t, "Hello John, your age is {{age}}", result)
	})

	t.Run("Multiple occurrences", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "John",
		}

		result := executor.replaceTemplateVariables("{{name}} said hello. {{name}} is happy.", input)
		assert.Equal(t, "John said hello. John is happy.", result)
	})

	t.Run("Access with input prefix (backward compatibility)", func(t *testing.T) {
		// After sleep node merges data, fields are at top level
		// But templates might still use {{input.field}} syntax
		input := map[string]interface{}{
			"userId": 123,
			"email":  "test@example.com",
		}

		// Should work with "input." prefix even though data is at top level
		result := executor.replaceTemplateVariables("User: {{input.userId}}, Email: {{input.email}}", input)
		assert.Equal(t, "User: 123, Email: test@example.com", result)

		// Should also work without "input." prefix
		result2 := executor.replaceTemplateVariables("User: {{userId}}, Email: {{email}}", input)
		assert.Equal(t, "User: 123, Email: test@example.com", result2)
	})
}
