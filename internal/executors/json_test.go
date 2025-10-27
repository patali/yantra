package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONExecutor_WithObject(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID: "json-node-1",
		NodeConfig: map[string]interface{}{
			"data": map[string]interface{}{
				"name":    "John Doe",
				"email":   "john@example.com",
				"age":     30,
				"active":  true,
				"address": map[string]interface{}{
					"city":  "New York",
					"state": "NY",
				},
			},
		},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Output)

	data := result.Output["data"].(map[string]interface{})
	assert.Equal(t, "John Doe", data["name"])
	assert.Equal(t, "john@example.com", data["email"])
	assert.Equal(t, 30, data["age"])
	assert.Equal(t, true, data["active"])

	address := data["address"].(map[string]interface{})
	assert.Equal(t, "New York", address["city"])
	assert.Equal(t, "NY", address["state"])
}

func TestJSONExecutor_WithArray(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID: "json-node-2",
		NodeConfig: map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"id": 1, "name": "Item 1"},
				map[string]interface{}{"id": 2, "name": "Item 2"},
				map[string]interface{}{"id": 3, "name": "Item 3"},
			},
		},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Output["data"].([]interface{})
	assert.Len(t, data, 3)

	item1 := data[0].(map[string]interface{})
	assert.Equal(t, 1, item1["id"])
	assert.Equal(t, "Item 1", item1["name"])
}

func TestJSONExecutor_WithJSONString(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID: "json-node-3",
		NodeConfig: map[string]interface{}{
			"data": `{"name": "Jane", "score": 95, "tags": ["go", "testing"]}`,
		},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Output["data"].(map[string]interface{})
	assert.Equal(t, "Jane", data["name"])
	assert.Equal(t, float64(95), data["score"])

	tags := data["tags"].([]interface{})
	assert.Len(t, tags, 2)
	assert.Equal(t, "go", tags[0])
	assert.Equal(t, "testing", tags[1])
}

func TestJSONExecutor_WithPrimitiveValue(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID: "json-node-4",
		NodeConfig: map[string]interface{}{
			"data": 42, // Use a number instead of string to avoid JSON parsing
		},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 42, result.Output["data"])
}

func TestJSONExecutor_MissingData(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID:       "json-node-5",
		NodeConfig:   map[string]interface{}{},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "data field is required", result.Error)
}

func TestJSONExecutor_InvalidJSONString(t *testing.T) {
	executor := NewJSONExecutor()

	execCtx := ExecutionContext{
		NodeID: "json-node-6",
		NodeConfig: map[string]interface{}{
			"data": `{invalid json}`,
		},
		Input:        nil,
		WorkflowData: map[string]interface{}{},
		ExecutionID:  "test-execution",
		AccountID:    "test-account",
	}

	result, err := executor.Execute(context.Background(), execCtx)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid JSON string")
}
