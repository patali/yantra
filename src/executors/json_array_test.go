package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
