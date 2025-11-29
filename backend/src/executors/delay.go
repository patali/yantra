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

	output := make(map[string]interface{})
	output["data"] = duration       // Primary output: delay duration in milliseconds
	output["delayed_ms"] = duration // Kept for backward compatibility

	// Merge input data if it's a map
	if inputMap, ok := execCtx.Input.(map[string]interface{}); ok {
		// If input has a "data" field that's a map, merge those fields at top level
		if inputData, ok := inputMap["data"].(map[string]interface{}); ok {
			for k, v := range inputData {
				if _, exists := output[k]; !exists { // Don't override our fields
					output[k] = v
				}
			}
		}

		// Also merge other top-level fields from input
		for k, v := range inputMap {
			if k != "data" { // Skip "data" as we handled it above
				if _, exists := output[k]; !exists { // Don't override our fields
					output[k] = v
				}
			}
		}
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
