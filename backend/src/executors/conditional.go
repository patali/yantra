package executors

import (
	"context"
	"fmt"
	"strings"

	"github.com/PaesslerAG/gval"
)

type ConditionalExecutor struct{}

func NewConditionalExecutor() *ConditionalExecutor {
	return &ConditionalExecutor{}
}

// buildConditionFromStructured converts structured conditions from frontend to gval expression string
func buildConditionFromStructured(nodeConfig map[string]interface{}) (string, error) {
	conditions, ok := nodeConfig["conditions"].([]interface{})
	if !ok || len(conditions) == 0 {
		return "", fmt.Errorf("condition not specified")
	}

	var conditionParts []string

	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		left, _ := condMap["left"].(string)
		operator, _ := condMap["operator"].(string)
		right, _ := condMap["right"].(string)

		if left == "" || operator == "" {
			continue
		}

		// Convert operator to gval syntax
		var gvalOp string
		switch operator {
		case "eq":
			gvalOp = "=="
		case "neq":
			gvalOp = "!="
		case "gt":
			gvalOp = ">"
		case "lt":
			gvalOp = "<"
		case "gte":
			gvalOp = ">="
		case "lte":
			gvalOp = "<="
		case "contains":
			// For contains, we'll use a different syntax
			if right == "" {
				conditionParts = append(conditionParts, fmt.Sprintf("%s != nil && %s != \"\"", left, left))
			} else {
				// Check if right value needs quotes (is it a string literal?)
				rightValue := right
				if !strings.HasPrefix(right, "\"") && !strings.HasSuffix(right, "\"") {
					rightValue = fmt.Sprintf("\"%s\"", right)
				}
				conditionParts = append(conditionParts, fmt.Sprintf("contains(%s, %s)", left, rightValue))
			}
			continue
		case "exists":
			conditionParts = append(conditionParts, fmt.Sprintf("%s != nil", left))
			continue
		default:
			return "", fmt.Errorf("unsupported operator: %s", operator)
		}

		// Build the condition part
		// Determine if right value needs quotes (is it a string literal?)
		rightValue := right
		if right != "" {
			// If right doesn't look like a field reference and isn't already quoted
			if !strings.Contains(right, ".") && !strings.HasPrefix(right, "\"") && !strings.HasSuffix(right, "\"") {
				// Check if it's a boolean
				if right == "true" || right == "false" {
					rightValue = right
				} else {
					// Try to parse as number, otherwise treat as string
					rightValue = right
					// For non-numeric values, add quotes
					if _, err := fmt.Sscanf(right, "%f", new(float64)); err != nil {
						rightValue = fmt.Sprintf("\"%s\"", right)
					}
				}
			}
		}

		conditionParts = append(conditionParts, fmt.Sprintf("%s %s %s", left, gvalOp, rightValue))
	}

	if len(conditionParts) == 0 {
		return "", fmt.Errorf("no valid conditions specified")
	}

	// Join conditions with logical operator
	logicalOp := " && "
	if lo, ok := nodeConfig["logicalOperator"].(string); ok && strings.ToUpper(lo) == "OR" {
		logicalOp = " || "
	}

	return strings.Join(conditionParts, logicalOp), nil
}

func (e *ConditionalExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get the condition expression - support both string format and structured format
	var condition string

	// Try to get condition as a string (legacy format)
	if condStr, ok := execCtx.NodeConfig["condition"].(string); ok && condStr != "" {
		condition = condStr
	} else {
		// Try to build condition from structured format (frontend format)
		builtCondition, err := buildConditionFromStructured(execCtx.NodeConfig)
		if err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
		condition = builtCondition
	}

	if condition == "" {
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
		// Also flatten input.data to inputData for easier access to nested data
		if inputMap, ok := execCtx.Input.(map[string]interface{}); ok {
			if data, ok := inputMap["data"].(map[string]interface{}); ok {
				// Add the data object itself for access via data.field
				evalContext["data"] = data
				// Also add nested data at root level for easier access (field without data. prefix)
				for k, v := range data {
					evalContext[k] = v
				}
			}
		}
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
		"data":      boolResult, // Primary output: boolean result
		"result":    boolResult, // Kept for backward compatibility
		"condition": condition,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
