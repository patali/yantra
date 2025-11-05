package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
