package executors

import (
	"context"
	"time"
)

type DelayExecutor struct{}

func NewDelayExecutor() *DelayExecutor {
	return &DelayExecutor{}
}

func (e *DelayExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get duration from config
	duration := 1000 // Default 1 second in milliseconds
	if d, ok := execCtx.NodeConfig["duration"].(float64); ok {
		duration = int(d)
	}

	// Use context-aware delay that can be cancelled
	// This allows the delay to be interrupted if the context is cancelled (e.g., server shutdown)
	select {
	case <-time.After(time.Duration(duration) * time.Millisecond):
		// Delay completed normally
	case <-ctx.Done():
		// Context was cancelled (timeout, shutdown, etc.)
		return nil, ctx.Err()
	}

	output := map[string]interface{}{
		"data":       duration, // Primary output: delay duration in milliseconds
		"delayed_ms": duration, // Kept for backward compatibility
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
