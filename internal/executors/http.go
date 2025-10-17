package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPExecutor struct{}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{}
}

func (e *HTTPExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get URL and method from config
	url, ok := execCtx.NodeConfig["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method, ok := execCtx.NodeConfig["method"].(string)
	if !ok || method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	// Get headers
	headers := make(map[string]string)
	if h, ok := execCtx.NodeConfig["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	// Get body (for POST, PUT, PATCH)
	var bodyReader io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if body, ok := execCtx.NodeConfig["body"]; ok {
			// If body is already a string, use it directly
			if bodyStr, ok := body.(string); ok {
				bodyReader = strings.NewReader(bodyStr)
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

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Execute request
	fmt.Printf("🌐 HTTP %s request to %s\n", method, url)
	resp, err := client.Do(req)
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
