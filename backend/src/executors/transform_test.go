package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTransformExecutor tests the transform executor
func TestTransformExecutor(t *testing.T) {
	executor := NewTransformExecutor()

	t.Run("No operations - returns input as-is", func(t *testing.T) {
		input := map[string]interface{}{"name": "John", "age": 30}
		execCtx := ExecutionContext{
			NodeID:      "transform-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, input, result.Output["data"])
	})

	t.Run("Extract with JSONPath", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "extract",
						"config": map[string]interface{}{
							"jsonPath": "$.user.name",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "John", result.Output["data"])
	})

	t.Run("Map fields", func(t *testing.T) {
		input := map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       30,
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "map",
						"config": map[string]interface{}{
							"mappings": map[string]interface{}{
								"firstName": "first_name",
								"lastName":  "last_name",
							},
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Equal(t, "John", data["first_name"])
		assert.Equal(t, "Doe", data["last_name"])
		assert.Nil(t, data["firstName"]) // Original field should not exist
	})

	t.Run("Parse JSON string", func(t *testing.T) {
		input := map[string]interface{}{
			"jsonString": `{"name":"John","age":30}`,
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "parse",
						"config": map[string]interface{}{
							"inputKey":  "jsonString",
							"outputKey": "parsed",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		parsed := data["parsed"].(map[string]interface{})
		assert.Equal(t, "John", parsed["name"])
		assert.Equal(t, float64(30), parsed["age"])
	})

	t.Run("Stringify JSON", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "stringify",
						"config": map[string]interface{}{
							"inputKey":  "user",
							"outputKey": "userJson",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Contains(t, data["userJson"].(string), "John")
		assert.Contains(t, data["userJson"].(string), "30")
	})

	t.Run("Concat fields", func(t *testing.T) {
		input := map[string]interface{}{
			"first": "John",
			"last":  "Doe",
		}
		execCtx := ExecutionContext{
			NodeID: "transform-node",
			NodeConfig: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "concat",
						"config": map[string]interface{}{
							"inputs":    "first,last",
							"separator": " ",
							"outputKey": "fullName",
						},
					},
				},
			},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		data := result.Output["data"].(map[string]interface{})
		assert.Equal(t, "John Doe", data["fullName"])
	})
}
