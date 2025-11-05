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
}
