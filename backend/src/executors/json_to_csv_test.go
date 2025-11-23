package executors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestJSONToCSVExecutor tests the JSON to CSV executor
func TestJSONToCSVExecutor(t *testing.T) {
	executor := NewJSONToCSVExecutor()

	t.Run("Convert array of objects to CSV", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"name": "John", "age": "30", "city": "NYC"},
			map[string]interface{}{"name": "Jane", "age": "25", "city": "LA"},
		}
		execCtx := ExecutionContext{
			NodeID:      "json-to-csv-node",
			NodeConfig:  map[string]interface{}{},
			Input:       input,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["row_count"])
		csv := result.Output["csv"].(string)
		assert.Contains(t, csv, "John")
		assert.Contains(t, csv, "Jane")
		assert.Contains(t, csv, "30")
		assert.Contains(t, csv, "25")
	})

	t.Run("Convert from config data", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "json-to-csv-node",
			NodeConfig: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": "1", "value": "test1"},
					map[string]interface{}{"id": "2", "value": "test2"},
				},
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Output["row_count"])
		csv := result.Output["csv"].(string)
		assert.Contains(t, csv, "1")
		assert.Contains(t, csv, "test1")
	})

	t.Run("No data error", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "json-to-csv-node",
			NodeConfig:  map[string]interface{}{},
			Input:       map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "no data to convert")
	})
}
