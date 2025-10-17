package executors

import (
	"context"
	"encoding/json"
	"fmt"
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

// mapFields maps fields from input to output
func (e *TransformExecutor) mapFields(config map[string]interface{}, data interface{}) (interface{}, error) {
	// TODO: Implement field mapping
	return data, nil
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
