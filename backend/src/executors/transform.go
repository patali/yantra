package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

type TransformExecutor struct{}

func NewTransformExecutor() *TransformExecutor {
	return &TransformExecutor{}
}

func (e *TransformExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get operations from config
	operations, ok := execCtx.NodeConfig["operations"].([]interface{})
	if !ok || len(operations) == 0 {
		// No operations configured, return input as-is
		return &ExecutionResult{
			Success: true,
			Output:  map[string]interface{}{"data": execCtx.Input},
		}, nil
	}

	// Start with the input data
	currentData := execCtx.Input

	// Apply each operation in sequence
	for i, opData := range operations {
		operation, ok := opData.(map[string]interface{})
		if !ok {
			continue
		}

		opType, _ := operation["type"].(string)
		opConfig, _ := operation["config"].(map[string]interface{})

		var err error
		currentData, err = e.applyOperation(opType, opConfig, currentData)
		if err != nil {
			return nil, fmt.Errorf("operation %d (%s) failed: %w", i+1, opType, err)
		}
	}

	// Return the transformed data
	output := map[string]interface{}{
		"data": currentData,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}

func (e *TransformExecutor) applyOperation(opType string, config map[string]interface{}, data interface{}) (interface{}, error) {
	switch opType {
	case "extract":
		return e.extractWithJSONPath(config, data)
	case "map":
		return e.mapFields(config, data)
	case "parse":
		return e.parseJSON(config, data)
	case "stringify":
		return e.stringifyJSON(config, data)
	case "concat":
		return e.concatenateFields(config, data)
	default:
		return data, nil // Unknown operation, return data unchanged
	}
}

// extractWithJSONPath extracts data using JSONPath
func (e *TransformExecutor) extractWithJSONPath(config map[string]interface{}, data interface{}) (interface{}, error) {
	jsonPathExpr, ok := config["jsonPath"].(string)
	if !ok || jsonPathExpr == "" {
		return nil, fmt.Errorf("jsonPath is required for extract operation")
	}

	// Convert data to JSON-compatible format
	var jsonData interface{}
	if dataMap, ok := data.(map[string]interface{}); ok {
		jsonData = dataMap
	} else {
		// Try to marshal and unmarshal to ensure proper JSON format
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	// Apply JSONPath
	result, err := jsonpath.Get(jsonPathExpr, jsonData)
	if err != nil {
		return nil, fmt.Errorf("jsonpath query failed: %w", err)
	}

	// If outputKey is specified, wrap result in object
	if outputKey, ok := config["outputKey"].(string); ok && outputKey != "" {
		return map[string]interface{}{outputKey: result}, nil
	}

	return result, nil
}

// getNestedValue retrieves a value from a nested map using dot notation
// Supports array indexing with numeric parts (e.g., "items.0.name")
func (e *TransformExecutor) getNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		// Try to parse as array index first
		if idx, err := strconv.Atoi(part); err == nil {
			// Current should be an array
			if arr, ok := current.([]interface{}); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
					continue
				}
				return nil, false
			}
			return nil, false
		}

		// Otherwise treat as map key
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}

		value, exists := currentMap[part]
		if !exists {
			return nil, false
		}

		current = value
	}

	return current, true
}

// mapFields maps fields from input to output according to a mapping configuration
// Config should contain "mappings" which is an array of {from, to} or an object mapping
func (e *TransformExecutor) mapFields(config map[string]interface{}, data interface{}) (interface{}, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object for map operation")
	}

	// Support two formats:
	// 1. "mappings" as array of objects: [{from: "oldName", to: "newName"}, ...]
	// 2. "mappings" as object: {"oldName": "newName", ...}

	mappings := config["mappings"]
	if mappings == nil {
		return data, nil // No mappings specified, return as-is
	}

	result := make(map[string]interface{})

	// Handle array format
	if mappingsArray, ok := mappings.([]interface{}); ok {
		// Start with empty result - only include mapped fields
		// (unless includeUnmapped is true)
		includeUnmapped, _ := config["includeUnmapped"].(bool)

		if includeUnmapped {
			// Start with original data if includeUnmapped is true
			for k, v := range dataMap {
				result[k] = v
			}
		}

		// Apply mappings
		for _, mappingItem := range mappingsArray {
			mapping, ok := mappingItem.(map[string]interface{})
			if !ok {
				continue
			}

			fromField, _ := mapping["from"].(string)
			toField, _ := mapping["to"].(string)

			if fromField == "" || toField == "" {
				continue
			}

			// Copy value from 'from' field to 'to' field
			// Support nested field access using dot notation
			value, exists := e.getNestedValue(dataMap, fromField)
			if exists {
				result[toField] = value

				// If removeSource is true and we're including unmapped fields, remove the original field
				// Note: only works for top-level fields, not nested paths
				if includeUnmapped && !strings.Contains(fromField, ".") {
					if removeSource, ok := mapping["removeSource"].(bool); ok && removeSource && fromField != toField {
						delete(result, fromField)
					}
				}
			}
		}

		return result, nil
	}

	// Handle object format (simple key-value mapping)
	if mappingsObj, ok := mappings.(map[string]interface{}); ok {
		for fromField, toFieldRaw := range mappingsObj {
			toField, ok := toFieldRaw.(string)
			if !ok {
				continue
			}

			// Support nested field access using dot notation
			value, exists := e.getNestedValue(dataMap, fromField)
			if exists {
				result[toField] = value
			}
		}

		// Include fields that weren't mapped
		includeUnmapped, _ := config["includeUnmapped"].(bool)
		if includeUnmapped {
			for k, v := range dataMap {
				if _, exists := result[k]; !exists {
					if _, isMapped := mappingsObj[k]; !isMapped {
						result[k] = v
					}
				}
			}
		}

		return result, nil
	}

	return nil, fmt.Errorf("mappings must be an array or object")
}

// parseJSON parses a JSON string field
func (e *TransformExecutor) parseJSON(config map[string]interface{}, data interface{}) (interface{}, error) {
	inputKey, _ := config["inputKey"].(string)
	outputKey, _ := config["outputKey"].(string)

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object for parse operation")
	}

	jsonStr, ok := dataMap[inputKey].(string)
	if !ok {
		return nil, fmt.Errorf("input key %s not found or not a string", inputKey)
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := make(map[string]interface{})
	for k, v := range dataMap {
		result[k] = v
	}
	result[outputKey] = parsed

	return result, nil
}

// stringifyJSON converts data to JSON string
func (e *TransformExecutor) stringifyJSON(config map[string]interface{}, data interface{}) (interface{}, error) {
	inputKey, _ := config["inputKey"].(string)
	outputKey, _ := config["outputKey"].(string)

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object for stringify operation")
	}

	value := dataMap[inputKey]
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify: %w", err)
	}

	result := make(map[string]interface{})
	for k, v := range dataMap {
		result[k] = v
	}
	result[outputKey] = string(jsonBytes)

	return result, nil
}

// concatenateFields concatenates multiple fields
func (e *TransformExecutor) concatenateFields(config map[string]interface{}, data interface{}) (interface{}, error) {
	inputsStr, _ := config["inputs"].(string)
	separator, _ := config["separator"].(string)
	outputKey, _ := config["outputKey"].(string)

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object for concat operation")
	}

	inputs := strings.Split(inputsStr, ",")
	var values []string
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if val, ok := dataMap[input]; ok {
			values = append(values, fmt.Sprintf("%v", val))
		}
	}

	result := make(map[string]interface{})
	for k, v := range dataMap {
		result[k] = v
	}
	result[outputKey] = strings.Join(values, separator)

	return result, nil
}
