package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AIWorkflowService handles AI-based workflow generation
type AIWorkflowService struct {
	apiKey     string
	apiBaseURL string
	httpClient *http.Client
}

// NewAIWorkflowService creates a new AI workflow service
func NewAIWorkflowService() *AIWorkflowService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY") // Fallback for other providers
	}

	return &AIWorkflowService{
		apiKey:     apiKey,
		apiBaseURL: getAPIBaseURL(),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func getAPIBaseURL() string {
	baseURL := os.Getenv("AI_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return baseURL
}

// GenerateWorkflowRequest represents the request to generate a workflow
type GenerateWorkflowRequest struct {
	Description string                 `json:"description" binding:"required"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// GenerateWorkflowResponse represents the response with the generated workflow
type GenerateWorkflowResponse struct {
	Workflow    map[string]interface{} `json:"workflow"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Explanation string                 `json:"explanation,omitempty"`
}

// OpenAI API structures
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateWorkflow generates a workflow definition from a natural language description
func (s *AIWorkflowService) GenerateWorkflow(req GenerateWorkflowRequest) (*GenerateWorkflowResponse, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("AI API key not configured. Set OPENAI_API_KEY or AI_API_KEY environment variable")
	}

	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(req)

	// Call OpenAI API
	aiReq := openAIRequest{
		Model: getModel(),
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3, // Lower temperature for more consistent output
		MaxTokens:   4000,
	}

	body, err := json.Marshal(aiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.apiBaseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var aiResp openAIResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	if aiResp.Error != nil {
		return nil, fmt.Errorf("AI API error: %s", aiResp.Error.Message)
	}

	if len(aiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	// Parse the generated workflow
	content := aiResp.Choices[0].Message.Content
	return parseAIResponse(content)
}

func getModel() string {
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o-mini" // Use cost-effective model by default
	}
	return model
}

func buildSystemPrompt() string {
	return `You are an expert workflow automation engineer. Your task is to generate workflow definitions in JSON format based on user descriptions.

## Workflow Structure

A workflow is a JSON object with:
- "name": Workflow name
- "description": What the workflow does
- "nodes": Array of node objects
- "edges": Array of edge objects connecting nodes

## Available Node Types

1. **start** - Entry point (required)
   - Config: { "triggerType": "manual|webhook|cron", "webhookPath": "...", "cronSchedule": "..." }

2. **end** - Exit point (required)

3. **http** - Make HTTP requests
   - Config: { "method": "GET|POST|PUT|DELETE", "url": "...", "headers": {}, "body": {...}, "timeout": 30000, "maxRetries": 3 }

4. **transform** - Transform data
   - Config: { "operations": [{ "type": "map|extract|parse|stringify|concat", "config": {...} }] }
   - Map: { "mappings": [{ "from": "source.field", "to": "target" }] }
   - Extract: { "path": "data.items[0].name" }

5. **conditional** - Branch based on conditions
   - Config: { "conditions": [{ "left": "fieldName", "operator": "eq|ne|gt|lt|gte|lte|contains", "right": "value" }], "logicalOperator": "AND|OR" }

6. **loop** - Iterate over arrays (must connect to loop-accumulator)

7. **loop-accumulator** - Collect loop results
   - Config: { "accumulationMode": "array", "accumulatorVariable": "results" }

8. **email** - Send emails
   - Config: { "to": "...", "subject": "...", "body": "...", "provider": "resend", "isHtml": false }

9. **slack** - Send Slack messages
   - Config: { "webhookUrl": "...", "channel": "#...", "message": "..." }

10. **json** - Static JSON data
    - Config: { "data": {...} }

11. **json-array** - Array data
    - Config: { "items": [...] }

12. **delay** - Short pause (milliseconds)
    - Config: { "duration": 1000 }

13. **sleep** - Long pause (hours/days)
    - Config: { "mode": "relative|absolute", "duration_value": 7, "duration_unit": "hours|days|weeks" }

14. **json_to_csv** - Convert JSON to CSV
    - Config: { "headers": ["col1", "col2"] }

## Node Structure

{
  "id": "unique-id",
  "type": "node-type",
  "label": "Display Name",
  "position": { "x": 100, "y": 100 },
  "data": {
    "config": { /* node-specific config */ }
  }
}

## Edge Structure

{
  "id": "edge-id",
  "source": "source-node-id",
  "target": "target-node-id",
  "sourceHandle": "true|false|loop-output|output",  // Optional, for branching
  "targetHandle": "input|accumulator-input"  // Optional, for loops
}

## Template Variables

Use {{node-id.field}} to reference node outputs:
- {{http-1.data}} - HTTP response
- {{transform-1.data.name}} - Transformed field
- {{start-1.data.email}} - Input data

## Rules

1. Every workflow MUST have exactly one "start" and one "end" node
2. Node IDs must be unique (use: start-1, http-1, transform-1, etc.)
3. Edge IDs must be unique (use: e1, e2, e3, etc.)
4. Loops require loop-accumulator with proper handles:
   - loop-accumulator "loop-output" → loop body → "accumulator-input" back to accumulator
   - accumulator "output" → next node after loop
5. Conditional nodes use sourceHandle "true" or "false" for branching
6. Position nodes in a readable flow (increment y by 100-150 per level)
7. All nodes must be connected in a valid execution path

## Response Format

Return ONLY a JSON object with this structure:
{
  "name": "Workflow Name",
  "description": "What it does",
  "explanation": "Brief explanation of the workflow logic",
  "workflow": {
    "nodes": [...],
    "edges": [...]
  }
}

Do NOT include markdown code blocks or any other text. Return only the JSON object.`
}

func buildUserPrompt(req GenerateWorkflowRequest) string {
	prompt := fmt.Sprintf("Generate a workflow for: %s", req.Description)

	if req.Context != nil && len(req.Context) > 0 {
		contextJSON, _ := json.MarshalIndent(req.Context, "", "  ")
		prompt += fmt.Sprintf("\n\nAdditional context:\n%s", string(contextJSON))
	}

	prompt += "\n\nGenerate a complete, valid workflow definition following all the rules."

	return prompt
}

func parseAIResponse(content string) (*GenerateWorkflowResponse, error) {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var response GenerateWorkflowResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse generated workflow: %w. Content: %s", err, content)
	}

	// Validate basic structure
	if response.Workflow == nil {
		return nil, fmt.Errorf("generated workflow is missing 'workflow' field")
	}

	nodes, ok := response.Workflow["nodes"].([]interface{})
	if !ok || len(nodes) == 0 {
		return nil, fmt.Errorf("generated workflow has invalid or empty 'nodes' array")
	}

	edges, ok := response.Workflow["edges"].([]interface{})
	if !ok || len(edges) == 0 {
		return nil, fmt.Errorf("generated workflow has invalid or empty 'edges' array")
	}

	// Validate start and end nodes
	hasStart := false
	hasEnd := false
	for _, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}
		nodeType, ok := nodeMap["type"].(string)
		if !ok {
			continue
		}
		if nodeType == "start" {
			hasStart = true
		}
		if nodeType == "end" {
			hasEnd = true
		}
	}

	if !hasStart {
		return nil, fmt.Errorf("generated workflow is missing 'start' node")
	}
	if !hasEnd {
		return nil, fmt.Errorf("generated workflow is missing 'end' node")
	}

	return &response, nil
}
