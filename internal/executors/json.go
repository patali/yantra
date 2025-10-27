package executors

import (
	"context"
	"encoding/json"
	"fmt"
)

type JSONExecutor struct{}

func NewJSONExecutor() *JSONExecutor {
	return &JSONExecutor{}
}

// Execute returns a static JSON object defined in the node configuration
// This is useful for defining constants, test data, or configuration objects
func (e *JSONExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get the JSON data from config
	data, ok := execCtx.NodeConfig["data"]
	if !ok {
		return &ExecutionResult{
			Success: false,
			Error:   "data field is required",
		}, nil
	}

	// If data is a string, try to parse it as JSON
	if dataStr, isString := data.(string); isString {
		var parsed interface{}
		if err := json.Unmarshal([]byte(dataStr), &parsed); err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("invalid JSON string: %v", err),
			}, nil
		}
		data = parsed
	}

	// Return the data as output
	output := map[string]interface{}{
		"data": data,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
