package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// JsonArrayTriggerExecutor executes a JSON array trigger node
type JsonArrayTriggerExecutor struct{}

// NewJsonArrayTriggerExecutor creates a new JSON array trigger executor
func NewJsonArrayTriggerExecutor() *JsonArrayTriggerExecutor {
	return &JsonArrayTriggerExecutor{}
}

// Execute validates and outputs a JSON array with uniform object types
func (e *JsonArrayTriggerExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Extract configuration
	jsonArrayStr, ok := execCtx.NodeConfig["jsonArray"].(string)
	if !ok || jsonArrayStr == "" {
		return &ExecutionResult{
			Success: false,
			Error:   "jsonArray configuration is required",
			Output:  map[string]interface{}{},
		}, fmt.Errorf("jsonArray configuration is required")
	}

	validateSchema := true
	if val, ok := execCtx.NodeConfig["validateSchema"].(bool); ok {
		validateSchema = val
	}

	// Parse JSON array
	var array []interface{}
	if err := json.Unmarshal([]byte(jsonArrayStr), &array); err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid JSON: %s", err.Error()),
			Output:  map[string]interface{}{},
		}, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate array is not empty
	if len(array) == 0 {
		return &ExecutionResult{
			Success: false,
			Error:   "Array cannot be empty",
			Output:  map[string]interface{}{},
		}, fmt.Errorf("array cannot be empty")
	}

	// Validate all elements are objects (maps)
	for i, item := range array {
		if _, ok := item.(map[string]interface{}); !ok {
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("Element at index %d is not an object", i),
				Output:  map[string]interface{}{},
			}, fmt.Errorf("element at index %d is not an object", i)
		}
	}

	// Validate uniform schema if enabled
	if validateSchema {
		if err := e.validateUniformSchema(array); err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   err.Error(),
				Output:  map[string]interface{}{},
			}, err
		}
	}

	// Success - return the array
	return &ExecutionResult{
		Success: true,
		Output: map[string]interface{}{
			"data":   array, // Primary output: the array itself
			"array":  array, // Kept for backward compatibility
			"count":  len(array),
			"schema": e.detectSchema(array[0].(map[string]interface{})),
		},
		Error: "",
	}, nil
}

// validateUniformSchema checks that all objects have the same keys
func (e *JsonArrayTriggerExecutor) validateUniformSchema(array []interface{}) error {
	if len(array) == 0 {
		return nil
	}

	// Get keys from first object
	firstObj := array[0].(map[string]interface{})
	firstKeys := e.getKeys(firstObj)
	sort.Strings(firstKeys)

	// Compare with all other objects
	for i := 1; i < len(array); i++ {
		obj := array[i].(map[string]interface{})
		keys := e.getKeys(obj)
		sort.Strings(keys)

		if !e.equalKeys(firstKeys, keys) {
			return fmt.Errorf("object at index %d has different properties than the first object", i)
		}
	}

	return nil
}

// getKeys extracts keys from a map
func (e *JsonArrayTriggerExecutor) getKeys(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys
}

// equalKeys checks if two string slices are equal
func (e *JsonArrayTriggerExecutor) equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// detectSchema returns a map of property names and their types
func (e *JsonArrayTriggerExecutor) detectSchema(obj map[string]interface{}) map[string]string {
	schema := make(map[string]string)
	for key, value := range obj {
		schema[key] = reflect.TypeOf(value).String()
	}
	return schema
}
