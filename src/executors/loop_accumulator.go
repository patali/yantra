package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type LoopAccumulatorExecutor struct{}

func NewLoopAccumulatorExecutor() *LoopAccumulatorExecutor {
	return &LoopAccumulatorExecutor{}
}

// Execute for loop accumulator
// This prepares the iteration data similar to the regular loop executor
// The workflow engine handles the actual iteration and accumulation logic
func (e *LoopAccumulatorExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Debug: Print input structure
	inputJSON, _ := json.MarshalIndent(execCtx.Input, "", "  ")
	fmt.Printf("🔍 Loop Accumulator node input: %s\n", string(inputJSON))

	// Get arrayPath from config (e.g., "data" or "data.users")
	arrayPath, _ := execCtx.NodeConfig["arrayPath"].(string)
	fmt.Printf("🔍 Loop Accumulator arrayPath config: '%s'\n", arrayPath)

	var items []interface{}
	var ok bool

	// If arrayPath is provided, navigate to it
	if arrayPath != "" && strings.TrimSpace(arrayPath) != "" {
		// Remove "input." prefix if user added it
		arrayPath = strings.TrimPrefix(arrayPath, "input.")

		items, ok = e.extractArrayFromPath(execCtx.Input, arrayPath)
		if !ok {
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("could not find array at path: %s (input structure logged above)", arrayPath),
			}, nil
		}
	} else {
		// Try to get array directly from input
		if inputMap, ok := execCtx.Input.(map[string]interface{}); ok {
			// If input is an object, try common keys
			if dataArray, ok := inputMap["data"].([]interface{}); ok {
				items = dataArray
			} else if itemsArray, ok := inputMap["items"].([]interface{}); ok {
				items = itemsArray
			} else {
				return &ExecutionResult{
					Success: false,
					Error:   "input is an object but no array found. Specify arrayPath config (e.g., 'data' or 'items').",
				}, nil
			}
		} else if inputArray, ok := execCtx.Input.([]interface{}); ok {
			// Input is already an array
			items = inputArray
		} else {
			return &ExecutionResult{
				Success: false,
				Error:   "input is not an array or object with array field",
			}, nil
		}
	}

	fmt.Printf("🔍 Loop Accumulator found %d items\n", len(items))

	// Get iteration config
	iterationCount := len(items)
	maxIterations := 100 // Safety limit
	if max, ok := execCtx.NodeConfig["max_iterations"].(float64); ok {
		maxIterations = int(max)
	}

	if iterationCount > maxIterations {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("iteration count %d exceeds maximum %d", iterationCount, maxIterations),
		}, nil
	}

	// Get variable names
	itemVariable := "item"
	if iv, ok := execCtx.NodeConfig["itemVariable"].(string); ok && iv != "" {
		itemVariable = iv
	}

	indexVariable := "index"
	if iv, ok := execCtx.NodeConfig["indexVariable"].(string); ok && iv != "" {
		indexVariable = iv
	}

	accumulatorVariable := "accumulated"
	if av, ok := execCtx.NodeConfig["accumulatorVariable"].(string); ok && av != "" {
		accumulatorVariable = av
	}

	// Get accumulation mode (default: "array")
	accumulationMode := "array"
	if mode, ok := execCtx.NodeConfig["accumulationMode"].(string); ok && mode != "" {
		accumulationMode = mode
	}

	// Store results from each iteration
	results := make([]interface{}, iterationCount)
	for i, item := range items {
		results[i] = map[string]interface{}{
			indexVariable:       i,
			itemVariable:        item,
			accumulatorVariable: []interface{}{}, // Initial empty accumulator
		}
	}

	output := map[string]interface{}{
		"iteration_count":     iterationCount,
		"results":             results,
		"items":               items,
		"accumulationMode":    accumulationMode,
		"accumulatorVariable": accumulatorVariable,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}

// extractArrayFromPath navigates through a dot-separated path to find an array
func (e *LoopAccumulatorExecutor) extractArrayFromPath(data interface{}, path string) ([]interface{}, bool) {
	parts := strings.Split(path, ".")
	current := data

	// Navigate through each part of the path
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fmt.Printf("🔍 Loop Accumulator navigating path part %d: '%s'\n", i, part)

		// Current must be a map to navigate deeper
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			fmt.Printf("❌ Current data is not an object at path part '%s'\n", part)
			return nil, false
		}

		// Get the next level
		next, exists := currentMap[part]
		if !exists {
			fmt.Printf("❌ Key '%s' not found in object. Available keys: %v\n", part, e.getKeys(currentMap))
			return nil, false
		}

		current = next
	}

	// Final value should be an array
	if arr, ok := current.([]interface{}); ok {
		return arr, true
	}

	fmt.Printf("❌ Final value is not an array, it's a %T\n", current)
	return nil, false
}

// getKeys returns the keys of a map for debugging
func (e *LoopAccumulatorExecutor) getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
