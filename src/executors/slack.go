package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackExecutor struct {
	httpClient *http.Client
}

func NewSlackExecutor(client *http.Client) *SlackExecutor {
	return &SlackExecutor{
		httpClient: client,
	}
}

type slackMessage struct {
	Channel  string                   `json:"channel,omitempty"`
	Text     string                   `json:"text,omitempty"`
	Username string                   `json:"username,omitempty"`
	IconURL  string                   `json:"icon_url,omitempty"`
	Blocks   []map[string]interface{} `json:"blocks,omitempty"`
}

func (e *SlackExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get webhook URL (required)
	webhookURL, ok := execCtx.NodeConfig["webhookUrl"].(string)
	if !ok || webhookURL == "" {
		return &ExecutionResult{
			Success: false,
			Error:   "webhookUrl is required",
		}, nil
	}

	// Build the Slack message
	message := slackMessage{}

	// Basic fields
	if channel, ok := execCtx.NodeConfig["channel"].(string); ok {
		message.Channel = channel
	}
	if text, ok := execCtx.NodeConfig["text"].(string); ok {
		message.Text = text
	}
	if username, ok := execCtx.NodeConfig["username"].(string); ok {
		message.Username = username
	}
	if iconURL, ok := execCtx.NodeConfig["iconUrl"].(string); ok {
		message.IconURL = iconURL
	}

	// Support blocks for rich formatting
	if blocks, ok := execCtx.NodeConfig["blocks"].([]interface{}); ok {
		message.Blocks = make([]map[string]interface{}, len(blocks))
		for i, block := range blocks {
			if blockMap, ok := block.(map[string]interface{}); ok {
				message.Blocks[i] = blockMap
			}
		}
	}

	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal message: %v", err),
		}, nil
	}

	// Send POST request to Slack webhook
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to send webhook: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("slack webhook returned status %d", resp.StatusCode),
		}, nil
	}

	fmt.Printf("💬 Slack message sent to %s: %s\n", message.Channel, message.Text)

	output := map[string]interface{}{
		"sent":       true,
		"channel":    message.Channel,
		"text":       message.Text,
		"statusCode": resp.StatusCode,
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
