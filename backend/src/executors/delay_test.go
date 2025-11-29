package executors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

	t.Run("Input data is passed through", func(t *testing.T) {
		inputData := map[string]interface{}{
			"userId":    12345,
			"taskId":    "task-abc",
			"timestamp": "2025-11-29T10:00:00Z",
			"nested": map[string]interface{}{
				"field1": "value1",
				"field2": 42,
			},
		}

		execCtx := ExecutionContext{
			NodeID: "delay-node",
			NodeConfig: map[string]interface{}{
				"duration": float64(100), // 100ms
			},
			Input:       inputData,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)

		// Verify input data fields are merged into output (not nested)
		assert.Equal(t, 12345, result.Output["userId"])
		assert.Equal(t, "task-abc", result.Output["taskId"])
		assert.Equal(t, "2025-11-29T10:00:00Z", result.Output["timestamp"])

		// Verify nested data is preserved
		nestedMap, ok := result.Output["nested"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value1", nestedMap["field1"])
		assert.Equal(t, 42, nestedMap["field2"])

		// Verify delay metadata is also present
		assert.Equal(t, 100, result.Output["delayed_ms"])
		assert.Equal(t, 100, result.Output["data"])
	})

	t.Run("Nil input is passed through", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "delay-node",
			NodeConfig: map[string]interface{}{
				"duration": float64(50),
			},
			Input:       nil,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)

		// With nil input, only delay metadata should be present
		assert.Equal(t, 50, result.Output["delayed_ms"])
		assert.NotContains(t, result.Output, "userId") // No user data
	})
}
