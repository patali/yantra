package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor(client *http.Client) *HTTPExecutor {
	return &HTTPExecutor{
		client: client,
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get URL and method from config
	url, ok := execCtx.NodeConfig["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	fmt.Printf("🔍 HTTP Executor - Original URL from config: %s\n", url)
	fmt.Printf("🔍 HTTP Executor - Input data: %+v\n", execCtx.Input)

	// Replace template variables in URL with input data
	url = e.replaceTemplateVariables(url, execCtx.Input)

	fmt.Printf("🔍 HTTP Executor - URL after template replacement: %s\n", url)

	method, ok := execCtx.NodeConfig["method"].(string)
	if !ok || method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	// Get headers and replace template variables
	headers := make(map[string]string)
	if h, ok := execCtx.NodeConfig["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if strVal, ok := v.(string); ok {
				// Replace template variables in header values
				headers[k] = e.replaceTemplateVariables(strVal, execCtx.Input)
			}
		}
	}

	// Get body (for POST, PUT, PATCH)
	var bodyReader io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if body, ok := execCtx.NodeConfig["body"]; ok {
			// If body is already a string, use it directly (after replacing variables)
			if bodyStr, ok := body.(string); ok {
				// Replace template variables in body
				replacedBody := e.replaceTemplateVariables(bodyStr, execCtx.Input)
				bodyReader = strings.NewReader(replacedBody)
			} else {
				// Otherwise, marshal to JSON
				bodyBytes, err := json.Marshal(body)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal body: %w", err)
				}
				bodyReader = bytes.NewReader(bodyBytes)
				// Set Content-Type to application/json if not already set
				if _, exists := headers["Content-Type"]; !exists {
					headers["Content-Type"] = "application/json"
				}
			}
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request using shared HTTP client
	fmt.Printf("🌐 HTTP %s request to %s\n", method, url)
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON, otherwise return as string
	var data interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		// Not JSON, return as string
		data = string(respBody)
	}

	// Build output
	output := map[string]interface{}{
		"status_code": resp.StatusCode,
		"url":         url,
		"method":      method,
		"data":        data,
		"headers":     resp.Header,
	}

	// Check if request was successful (2xx status code)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return &ExecutionResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("HTTP request failed with status %d", resp.StatusCode),
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}

// replaceTemplateVariables replaces {{variable}} patterns with values from input
func (e *HTTPExecutor) replaceTemplateVariables(text string, input interface{}) string {
	// Match patterns like {{input.field}} or {{variable}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])

		// Get the value from input
		value := e.getValueFromPath(input, varName)

		// Convert to string
		if value != nil {
			return fmt.Sprintf("%v", value)
		}

		// If not found, keep the original placeholder
		return match
	})

	return result
}

// getValueFromPath navigates through nested objects to get a value
// Supports paths like "input.field", "field.nested", "index", etc.
func (e *HTTPExecutor) getValueFromPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case map[interface{}]interface{}:
			current = v[part]
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}
