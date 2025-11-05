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
