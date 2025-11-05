package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
