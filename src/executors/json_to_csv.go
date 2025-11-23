package executors

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

type JSONToCSVExecutor struct{}

func NewJSONToCSVExecutor() *JSONToCSVExecutor {
	return &JSONToCSVExecutor{}
}

func (e *JSONToCSVExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get JSON data from config or input
	var data []map[string]interface{}

	// Try to get from config first
	if jsonData, ok := execCtx.NodeConfig["data"]; ok {
		if jsonBytes, err := json.Marshal(jsonData); err == nil {
			json.Unmarshal(jsonBytes, &data)
		}
	}

	// If not in config, try input
	if len(data) == 0 {
		if inputData, ok := execCtx.Input.([]interface{}); ok {
			for _, item := range inputData {
				if mapItem, ok := item.(map[string]interface{}); ok {
					data = append(data, mapItem)
				}
			}
		}
	}

	if len(data) == 0 {
		return &ExecutionResult{
			Success: false,
			Error:   "no data to convert",
		}, nil
	}

	// Extract headers from first object
	headers := make([]string, 0)
	for key := range data[0] {
		headers = append(headers, key)
	}

	// Create CSV
	var csvBuffer strings.Builder
	writer := csv.NewWriter(&csvBuffer)

	// Write headers
	if err := writer.Write(headers); err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write headers: %v", err),
		}, nil
	}

	// Write rows
	for _, row := range data {
		record := make([]string, len(headers))
		for i, header := range headers {
			if val, ok := row[header]; ok {
				record[i] = fmt.Sprintf("%v", val)
			} else {
				record[i] = ""
			}
		}
		if err := writer.Write(record); err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("failed to write row: %v", err),
			}, nil
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("CSV writer error: %v", err),
		}, nil
	}

	csvString := csvBuffer.String()

	output := map[string]interface{}{
		"data":      csvString, // Primary output: CSV string
		"csv":       csvString, // Kept for backward compatibility
		"row_count": len(data),
		"headers":   headers,
		"converted": true,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
