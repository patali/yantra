package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	urlStr, ok := execCtx.NodeConfig["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Replace template variables in URL with URL-encoded values
	urlStr = e.replaceTemplateVariablesWithEncoding(urlStr, execCtx.Input, true)

	method, ok := execCtx.NodeConfig["method"].(string)
	if !ok || method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	// Get headers and replace template variables (no URL encoding for headers)
	headers := make(map[string]string)
	if h, ok := execCtx.NodeConfig["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if strVal, ok := v.(string); ok {
				// Replace template variables in header values without encoding
				headers[k] = e.replaceTemplateVariablesWithEncoding(strVal, execCtx.Input, false)
			}
		}
	}

	// Get body (for POST, PUT, PATCH)
	var bodyReader io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if body, ok := execCtx.NodeConfig["body"]; ok {
			// If body is already a string, use it directly (after replacing variables)
			if bodyStr, ok := body.(string); ok {
				// Replace template variables in body without URL encoding
				replacedBody := e.replaceTemplateVariablesWithEncoding(bodyStr, execCtx.Input, false)
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
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request using shared HTTP client
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
		"url":         urlStr,
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

// replaceTemplateVariablesWithEncoding replaces {{variable}} patterns with values from input
// If urlEncode is true, values are URL-encoded to handle spaces and special characters
func (e *HTTPExecutor) replaceTemplateVariablesWithEncoding(text string, input interface{}, urlEncode bool) string {
	// Match patterns like {{input.field}} or {{variable}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])

		// Get the value from input
		value := e.getValueFromPath(input, varName)

		// Convert to string
		if value != nil {
			strValue := fmt.Sprintf("%v", value)
			if urlEncode {
				// URL-encode the value to handle spaces and special characters
				return url.QueryEscape(strValue)
			}
			return strValue
		}

		// If not found, keep the original placeholder
		return match
	})

	return result
}

// replaceTemplateVariables replaces {{variable}} patterns with values from input (no encoding)
// Kept for backward compatibility with tests
func (e *HTTPExecutor) replaceTemplateVariables(text string, input interface{}) string {
	return e.replaceTemplateVariablesWithEncoding(text, input, false)
}

// getValueFromPath navigates through nested objects to get a value
// Supports paths like "input.field", "field.nested", "field", etc.
// Special handling: if path starts with "input." and input is a map, try without the "input." prefix
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
			// Special case: if path started with "input." and we couldn't find it,
			// try without the "input." prefix (for backward compatibility)
			if len(parts) > 1 && parts[0] == "input" {
				// Retry without "input." prefix
				return e.getValueFromPath(data, strings.Join(parts[1:], "."))
			}
			return nil
		}
	}

	return current
}
