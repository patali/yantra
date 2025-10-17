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

	// Sleep for the specified duration
	time.Sleep(time.Duration(duration) * time.Millisecond)

	output := map[string]interface{}{
		"delayed_ms": duration,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
