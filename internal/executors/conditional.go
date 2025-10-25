package executors

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
)

type ConditionalExecutor struct{}

func NewConditionalExecutor() *ConditionalExecutor {
	return &ConditionalExecutor{}
}

func (e *ConditionalExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get the condition expression
	condition, ok := execCtx.NodeConfig["condition"].(string)
	if !ok || condition == "" {
		return &ExecutionResult{
			Success: false,
			Error:   "condition not specified",
		}, nil
	}

	// Prepare the evaluation context with input data and workflow data
	evalContext := make(map[string]interface{})

	// Add input data
	if execCtx.Input != nil {
		evalContext["input"] = execCtx.Input
	}

	// Add workflow data (contains previous node outputs)
	if execCtx.WorkflowData != nil {
		evalContext["workflow"] = execCtx.WorkflowData
		// Also add at root level for easier access
		for k, v := range execCtx.WorkflowData {
			evalContext[k] = v
		}
	}

	// Evaluate the condition using gval
	result, err := gval.Evaluate(condition, evalContext)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to evaluate condition: %v", err),
		}, nil
	}

	// Convert result to boolean
	boolResult, ok := result.(bool)
	if !ok {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("condition must evaluate to boolean, got %T", result),
		}, nil
	}

	output := map[string]interface{}{
		"result":    boolResult,
		"condition": condition,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
