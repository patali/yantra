package executors

import (
	"context"
	"testing"

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

// TestConditionalExecutorStructuredFormat tests the conditional executor with structured conditions from frontend
func TestConditionalExecutorStructuredFormat(t *testing.T) {
	executor := NewConditionalExecutor()

	tests := []struct {
		name         string
		nodeConfig   map[string]interface{}
		input        interface{}
		workflowData map[string]interface{}
		wantSuccess  bool
		wantResult   bool
		wantError    string
	}{
		{
			name: "Simple equals condition",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "data.success",
						"operator": "eq",
						"right":    "true",
					},
				},
				"logicalOperator": "AND",
			},
			input: map[string]interface{}{
				"data": map[string]interface{}{
					"success": true,
				},
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Greater than condition",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "count",
						"operator": "gt",
						"right":    "5",
					},
				},
				"logicalOperator": "AND",
			},
			workflowData: map[string]interface{}{
				"count": 10,
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Multiple conditions with AND",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "age",
						"operator": "gte",
						"right":    "18",
					},
					map[string]interface{}{
						"left":     "status",
						"operator": "eq",
						"right":    "active",
					},
				},
				"logicalOperator": "AND",
			},
			workflowData: map[string]interface{}{
				"age":    25,
				"status": "active",
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Multiple conditions with OR (one true)",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "isAdmin",
						"operator": "eq",
						"right":    "true",
					},
					map[string]interface{}{
						"left":     "isModerator",
						"operator": "eq",
						"right":    "true",
					},
				},
				"logicalOperator": "OR",
			},
			workflowData: map[string]interface{}{
				"isAdmin":     false,
				"isModerator": true,
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Exists condition",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "userId",
						"operator": "exists",
					},
				},
				"logicalOperator": "AND",
			},
			workflowData: map[string]interface{}{
				"userId": "12345",
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Not equals condition",
			nodeConfig: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"left":     "status",
						"operator": "neq",
						"right":    "failed",
					},
				},
				"logicalOperator": "AND",
			},
			workflowData: map[string]interface{}{
				"status": "success",
			},
			wantSuccess: true,
			wantResult:  true,
		},
		{
			name: "Empty conditions array",
			nodeConfig: map[string]interface{}{
				"conditions":      []interface{}{},
				"logicalOperator": "AND",
			},
			wantSuccess: false,
			wantError:   "condition not specified",
		},
		{
			name: "Missing conditions field",
			nodeConfig: map[string]interface{}{
				"logicalOperator": "AND",
			},
			wantSuccess: false,
			wantError:   "condition not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCtx := ExecutionContext{
				NodeID:       "conditional-node",
				NodeConfig:   tt.nodeConfig,
				Input:        tt.input,
				WorkflowData: tt.workflowData,
				ExecutionID:  "test-execution",
				AccountID:    "test-account",
			}

			result, err := executor.Execute(context.Background(), execCtx)

			if tt.wantError != "" {
				assert.NoError(t, err)
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.wantError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSuccess, result.Success)
				if tt.wantSuccess {
					assert.Equal(t, tt.wantResult, result.Output["data"])
				}
			}
		})
	}
}

// TestConditionalExecutorInputPassThrough tests that input is passed through in the output
func TestConditionalExecutorInputPassThrough(t *testing.T) {
	executor := NewConditionalExecutor()

	input := map[string]interface{}{
		"data": map[string]interface{}{
			"count":   42,
			"message": "test message",
			"nested": map[string]interface{}{
				"field": "value",
			},
		},
	}

	execCtx := ExecutionContext{
		NodeID: "conditional-node",
		NodeConfig: map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"left":     "data.count",
					"operator": "gt",
					"right":    "10",
				},
			},
			"logicalOperator": "AND",
		},
		Input:       input,
		ExecutionID: "test-execution",
		AccountID:   "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	// Verify execution succeeded
	assert.NoError(t, err)
	assert.True(t, result.Success)

	// Verify boolean result
	assert.Equal(t, true, result.Output["data"])
	assert.Equal(t, true, result.Output["result"])

	// Verify input is passed through
	assert.NotNil(t, result.Output["input"])
	passedInput, ok := result.Output["input"].(map[string]interface{})
	assert.True(t, ok, "input should be a map")

	// Verify input contains the original data
	assert.Equal(t, input, passedInput, "passed through input should match original input")

	// Verify nested data is accessible
	dataMap, ok := passedInput["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 42, dataMap["count"])
	assert.Equal(t, "test message", dataMap["message"])
}
