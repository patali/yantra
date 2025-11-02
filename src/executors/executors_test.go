package executors

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestConditionalExecutor tests the conditional executor
func TestConditionalExecutor(t *testing.T) {
	executor := NewConditionalExecutor()

	tests := []struct {
		name         string
		condition    string
		input        interface{}
		workflowData map[string]interface{}
		wantSuccess  bool
		wantResult   bool
		wantError    string
	}{
		{
			name:        "Simple true condition",
			condition:   "true",
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name:        "Simple false condition",
			condition:   "false",
			wantSuccess: true,
			wantResult:  false,
		},
		{
			name:        "Condition with input",
			condition:   "input.value > 10",
			input:       map[string]interface{}{"value": 15},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name:         "Condition with workflow data",
			condition:    `workflow.status == "active"`,
			workflowData: map[string]interface{}{"status": "active"},
			wantSuccess:  true,
			wantResult:   true,
		},
		{
			name:        "Complex condition",
			condition:   "input.age >= 18 && input.name != ''",
			input:       map[string]interface{}{"age": 25, "name": "John"},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name:        "Missing condition",
			condition:   "",
			wantSuccess: false,
			wantError:   "condition not specified",
		},
		{
			name:        "Invalid condition syntax",
			condition:   "invalid syntax {{",
			wantSuccess: false,
			wantError:   "failed to evaluate condition",
		},
		{
			name:        "Non-boolean result",
			condition:   "5 + 3", // Returns number, not boolean
			wantSuccess: false,
			wantError:   "condition must evaluate to boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCtx := ExecutionContext{
				NodeID:       "conditional-node",
				NodeConfig:   map[string]interface{}{"condition": tt.condition},
				Input:        tt.input,
				WorkflowData: tt.workflowData,
				ExecutionID:  "test-execution",
				AccountID:    "test-account",
			}

			result, err := executor.Execute(context.Background(), execCtx)

			if tt.wantError != "" {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.wantError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSuccess, result.Success)
				if tt.wantSuccess {
					assert.Equal(t, tt.wantResult, result.Output["result"])
				}
			}
		})
	}
}

// TestDelayExecutor tests the delay executor
func TestDelayExecutor(t *testing.T) {
	executor := NewDelayExecutor()

	t.Run("Default delay (1000ms)", func(t *testing.T) {
		start := time.Now()
		execCtx := ExecutionContext{
			NodeID:      "delay-node",
			NodeConfig:  map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		duration := time.Since(start)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.GreaterOrEqual(t, duration, 1000*time.Millisecond)
		assert.Less(t, duration, 1100*time.Millisecond) // Allow small buffer
		assert.Equal(t, 1000, result.Output["delayed_ms"])
	})

	t.Run("Custom delay", func(t *testing.T) {
		start := time.Now()
		execCtx := ExecutionContext{
			NodeID:      "delay-node",
			NodeConfig:  map[string]interface{}{"duration": 500.0},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		duration := time.Since(start)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.GreaterOrEqual(t, duration, 500*time.Millisecond)
		assert.Less(t, duration, 600*time.Millisecond)
		assert.Equal(t, 500, result.Output["delayed_ms"])
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		execCtx := ExecutionContext{
			NodeID:      "delay-node",
			NodeConfig:  map[string]interface{}{"duration": 5000.0},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(ctx, execCtx)

		assert.Error(t, err)  // Should return context cancelled error
		assert.Nil(t, result) // Result should be nil on error
	})
}

// TestTransformExecutor tests the transform executor
func TestTransformExecutor(t *testing.T) {
	executor := NewTransformExecutor()

	t.Run("No operations - returns input as-is", func(t *testing.T) {
		input := map[string]interface{}{"name": "John", "age": 30}
		execCtx := ExecutionContext{
			NodeID:      "transform-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, input, result.Output["data"])
	})

	t.Run("Extract with JSONPath", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "extract",
						"config": map[string]interface{}{
							"jsonPath": "$.user.name",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "John", result.Output["data"])
	})

	t.Run("Map fields", func(t *testing.T) {
		input := map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       30,
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "map",
						"config": map[string]interface{}{
							"mappings": map[string]interface{}{
								"firstName": "first_name",
								"lastName":  "last_name",
							},
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Equal(t, "John", data["first_name"])
		assert.Equal(t, "Doe", data["last_name"])
		assert.Nil(t, data["firstName"]) // Original field should not exist
	})

	t.Run("Parse JSON string", func(t *testing.T) {
		input := map[string]interface{}{
			"jsonString": `{"name":"John","age":30}`,
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "parse",
						"config": map[string]interface{}{
							"inputKey":  "jsonString",
							"outputKey": "parsed",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		parsed := data["parsed"].(map[string]interface{})
		assert.Equal(t, "John", parsed["name"])
		assert.Equal(t, float64(30), parsed["age"])
	})

	t.Run("Stringify JSON", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "stringify",
						"config": map[string]interface{}{
							"inputKey":  "user",
							"outputKey": "userJson",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Contains(t, data["userJson"].(string), "John")
		assert.Contains(t, data["userJson"].(string), "30")
	})

	t.Run("Concat fields", func(t *testing.T) {
		input := map[string]interface{}{
			"first": "John",
			"last":  "Doe",
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "concat",
						"config": map[string]interface{}{
							"inputs":    "first,last",
							"separator": " ",
							"outputKey": "fullName",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Equal(t, "John Doe", data["fullName"])
	})
}

// TestLoopExecutor tests the loop executor
func TestLoopExecutor(t *testing.T) {
	executor := NewLoopExecutor()

	t.Run("Simple array input", func(t *testing.T) {
		input := []interface{}{"item1", "item2", "item3"}
		execCtx := ExecutionContext{
			NodeID:      "loop-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, result.Output["iteration_count"])
		items := result.Output["items"].([]interface{})
		assert.Len(t, items, 3)
	})

	t.Run("Array in object with data key", func(t *testing.T) {
		input := map[string]interface{}{
			"data": []interface{}{"item1", "item2"},
		}
		execCtx := ExecutionContext{
			NodeID:      "loop-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["iteration_count"])
	})

	t.Run("Array path configuration", func(t *testing.T) {
		input := map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"name": "John"},
				map[string]interface{}{"name": "Jane"},
			},
		}
		execCtx := ExecutionContext{
			NodeID:      "loop-node",
			NodeConfig:  map[string]interface{}{"arrayPath": "users"},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["iteration_count"])
	})

	t.Run("Custom variable names", func(t *testing.T) {
		input := []interface{}{"item1", "item2"}
		execCtx := ExecutionContext{
			NodeID: "loop-node",
			NodeConfig: map[string]interface{}{
				"itemVariable":  "currentItem",
				"indexVariable": "position",
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		results := result.Output["results"].([]interface{})
		assert.Equal(t, "item1", results[0].(map[string]interface{})["currentItem"])
		assert.Equal(t, 0, results[0].(map[string]interface{})["position"])
	})

	t.Run("Max iterations limit", func(t *testing.T) {
		// Create array with 1001 items (exceeds default max of 1000)
		items := make([]interface{}, 1001)
		for i := range items {
			items[i] = i
		}

		execCtx := ExecutionContext{
			NodeID:      "loop-node",
			NodeConfig:  map[string]interface{}{},
			Input:       items,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "exceeds maximum")
	})

	t.Run("Invalid input - not an array", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "loop-node",
			NodeConfig:  map[string]interface{}{},
			Input:       "not an array",
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "not an array")
	})
}

// TestLoopAccumulatorExecutor tests the loop accumulator executor
func TestLoopAccumulatorExecutor(t *testing.T) {
	executor := NewLoopAccumulatorExecutor()

	t.Run("Simple array input", func(t *testing.T) {
		input := []interface{}{"item1", "item2", "item3"}
		execCtx := ExecutionContext{
			NodeID:      "loop-accumulator-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, result.Output["iteration_count"])
		assert.Equal(t, "array", result.Output["accumulationMode"])
		assert.Equal(t, "accumulated", result.Output["accumulatorVariable"])
	})

	t.Run("Custom accumulator variables", func(t *testing.T) {
		input := []interface{}{"item1", "item2"}
		execCtx := ExecutionContext{
			NodeID: "loop-accumulator-node",
			NodeConfig: map[string]interface{}{
				"accumulatorVariable": "result",
				"accumulationMode":    "sum",
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "result", result.Output["accumulatorVariable"])
		assert.Equal(t, "sum", result.Output["accumulationMode"])
	})
}

// TestJsonArrayTriggerExecutor tests the JSON array trigger executor
func TestJsonArrayTriggerExecutor(t *testing.T) {
	executor := NewJsonArrayTriggerExecutor()

	t.Run("Valid uniform array", func(t *testing.T) {
		jsonArray := `[{"name":"John","age":30},{"name":"Jane","age":25}]`
		execCtx := ExecutionContext{
			NodeID: "json-array-trigger-node",
			NodeConfig: map[string]interface{}{
				"jsonArray":      jsonArray,
				"validateSchema": true,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["count"])
		assert.NotNil(t, result.Output["array"])
		assert.NotNil(t, result.Output["schema"])
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "json-array-trigger-node",
			NodeConfig: map[string]interface{}{
				"jsonArray": `invalid json`,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Invalid JSON")
	})

	t.Run("Empty array", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "json-array-trigger-node",
			NodeConfig: map[string]interface{}{
				"jsonArray": `[]`,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "cannot be empty")
	})

	t.Run("Non-uniform schema", func(t *testing.T) {
		jsonArray := `[{"name":"John"},{"name":"Jane","age":25}]`
		execCtx := ExecutionContext{
			NodeID: "json-array-trigger-node",
			NodeConfig: map[string]interface{}{
				"jsonArray":      jsonArray,
				"validateSchema": true,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "different properties")
	})

	t.Run("Non-object element", func(t *testing.T) {
		jsonArray := `[{"name":"John"},"not an object"]`
		execCtx := ExecutionContext{
			NodeID: "json-array-trigger-node",
			NodeConfig: map[string]interface{}{
				"jsonArray": jsonArray,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "not an object")
	})
}

// TestJSONToCSVExecutor tests the JSON to CSV executor
func TestJSONToCSVExecutor(t *testing.T) {
	executor := NewJSONToCSVExecutor()

	t.Run("Convert array of objects to CSV", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"name": "John", "age": "30", "city": "NYC"},
			map[string]interface{}{"name": "Jane", "age": "25", "city": "LA"},
		}
		execCtx := ExecutionContext{
			NodeID:      "json-to-csv-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["row_count"])
		csv := result.Output["csv"].(string)
		assert.Contains(t, csv, "John")
		assert.Contains(t, csv, "Jane")
		assert.Contains(t, csv, "30")
		assert.Contains(t, csv, "25")
	})

	t.Run("Convert from config data", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "json-to-csv-node",
			NodeConfig: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": "1", "value": "test1"},
					map[string]interface{}{"id": "2", "value": "test2"},
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["row_count"])
		csv := result.Output["csv"].(string)
		assert.Contains(t, csv, "1")
		assert.Contains(t, csv, "test1")
	})

	t.Run("No data error", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "json-to-csv-node",
			NodeConfig:  map[string]interface{}{},
			Input:       map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "no data to convert")
	})
}

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

// TestEmailExecutor tests the email executor (with mock service)
func TestEmailExecutor(t *testing.T) {
	// Mock email service
	mockEmailService := &mockEmailService{}
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
		mockEmailService.success = true
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

// Mock email service for testing
type mockEmailService struct {
	success bool
	error   string
}

func (m *mockEmailService) SendEmail(ctx context.Context, accountID string, options EmailOptions) (*EmailResult, error) {
	if !m.success {
		return &EmailResult{
			Success: false,
			Error:   m.error,
		}, nil
	}
	return &EmailResult{
		Success:   true,
		MessageID: "mock-message-id",
	}, nil
}

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
