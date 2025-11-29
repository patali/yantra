package executors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSleepExecutor(t *testing.T) {
	executor := NewSleepExecutor()

	t.Run("Missing mode config", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID:      "sleep-node",
			NodeConfig:  map[string]interface{}{},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "mode is required")
	})

	t.Run("Invalid mode value", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode": "invalid",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid mode")
	})

	t.Run("Relative mode - missing duration_value", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":          "relative",
				"duration_unit": "days",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "duration_value is required")
	})

	t.Run("Relative mode - missing duration_unit", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 5.0,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "duration_unit is required")
	})

	t.Run("Relative mode - invalid duration_unit", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 5.0,
				"duration_unit":  "years",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid duration_unit")
	})

	t.Run("Relative mode - negative duration_value", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": -5.0,
				"duration_unit":  "days",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "must be non-negative")
	})

	t.Run("Relative mode - seconds", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 30.0,
				"duration_unit":  "seconds",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)
		after := time.Now().UTC()

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Wake-up time should be ~30 seconds in the future
		expectedWakeUp := before.Add(30 * time.Second)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
		assert.True(t, result.WakeUpAt.After(after))
	})

	t.Run("Relative mode - minutes", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 15.0,
				"duration_unit":  "minutes",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Wake-up time should be ~15 minutes in the future
		expectedWakeUp := before.Add(15 * time.Minute)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Relative mode - hours", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 2.0,
				"duration_unit":  "hours",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Wake-up time should be ~2 hours in the future
		expectedWakeUp := before.Add(2 * time.Hour)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Relative mode - days", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 7.0,
				"duration_unit":  "days",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Wake-up time should be ~7 days in the future
		expectedWakeUp := before.Add(7 * 24 * time.Hour)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Relative mode - weeks", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 2.0,
				"duration_unit":  "weeks",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Wake-up time should be ~14 days in the future
		expectedWakeUp := before.Add(14 * 24 * time.Hour)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Absolute mode - missing target_date", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode": "absolute",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "target_date is required")
	})

	t.Run("Absolute mode - invalid date format", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": "not-a-date",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid target_date format")
	})

	t.Run("Absolute mode - future date (RFC3339)", func(t *testing.T) {
		futureTime := time.Now().UTC().Add(24 * time.Hour)
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": futureTime.Format(time.RFC3339),
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)
		assert.WithinDuration(t, futureTime, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Absolute mode - future date (ISO 8601)", func(t *testing.T) {
		futureTime := time.Now().UTC().Add(48 * time.Hour)
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": futureTime.Format("2006-01-02T15:04:05"),
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)
		assert.WithinDuration(t, futureTime, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Absolute mode - ISO 8601 without seconds", func(t *testing.T) {
		futureTime := time.Now().UTC().Add(48 * time.Hour)
		// Format: "2006-01-02T15:04" (no seconds)
		dateStr := futureTime.Format("2006-01-02T15:04")
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": dateStr,
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)
		// Should parse correctly (seconds will default to :00)
		expectedTime := time.Date(
			futureTime.Year(), futureTime.Month(), futureTime.Day(),
			futureTime.Hour(), futureTime.Minute(), 0, 0, time.UTC,
		)
		assert.WithinDuration(t, expectedTime, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Absolute mode - past date (completes immediately)", func(t *testing.T) {
		pastTime := time.Now().UTC().Add(-24 * time.Hour)
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": pastTime.Format(time.RFC3339),
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.False(t, result.NeedsSleep) // Should NOT need sleep
		assert.Nil(t, result.WakeUpAt)
		assert.True(t, result.Output["sleep_skipped"].(bool))
		assert.Equal(t, "target time already passed", result.Output["reason"])
	})

	t.Run("Absolute mode - with timezone", func(t *testing.T) {
		// Future time in New York timezone
		loc, _ := time.LoadLocation("America/New_York")
		futureTime := time.Now().In(loc).Add(24 * time.Hour)

		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": futureTime.Format("2006-01-02T15:04:05"),
				"timezone":    "America/New_York",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)
		assert.NotNil(t, result.WakeUpAt)

		// Should be stored in UTC but represent the same moment in time
		expectedUTC := futureTime.UTC()
		assert.WithinDuration(t, expectedUTC, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Absolute mode - invalid timezone", func(t *testing.T) {
		futureTime := time.Now().UTC().Add(24 * time.Hour)
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": futureTime.Format(time.RFC3339),
				"timezone":    "Invalid/Timezone",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid timezone")
	})

	t.Run("Output format - relative mode", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 1.0,
				"duration_unit":  "days",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "sleep_scheduled_until")
		assert.Contains(t, result.Output, "sleep_duration_ms")
		assert.Contains(t, result.Output, "mode")
		assert.Contains(t, result.Output, "scheduled_at")
		assert.Equal(t, "relative", result.Output["mode"])

		// Duration should be approximately 24 hours in milliseconds
		durationMs := result.Output["sleep_duration_ms"].(int64)
		expectedMs := int64(24 * 60 * 60 * 1000)        // 86,400,000 ms
		assert.InDelta(t, expectedMs, durationMs, 1000) // Allow 1 second variance
	})

	t.Run("Relative mode - fractional values", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 1.5,
				"duration_unit":  "days",
			},
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		before := time.Now().UTC()
		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)

		// Wake-up time should be ~1.5 days (36 hours) in the future
		expectedWakeUp := before.Add(36 * time.Hour)
		assert.WithinDuration(t, expectedWakeUp, *result.WakeUpAt, 1*time.Second)
	})

	t.Run("Input passthrough - relative mode", func(t *testing.T) {
		// Simulate input from a previous node (like JSON node) that wraps data in "data" field
		inputData := map[string]interface{}{
			"data": map[string]interface{}{
				"userId":    12345,
				"taskId":    "task-abc",
				"timestamp": "2025-11-29T10:00:00Z",
				"nested": map[string]interface{}{
					"field1": "value1",
					"field2": 42,
				},
			},
		}

		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 5.0,
				"duration_unit":  "minutes",
			},
			Input:       inputData,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsSleep)

		// Verify input data fields are merged into output (not nested)
		assert.Equal(t, 12345, result.Output["userId"])
		assert.Equal(t, "task-abc", result.Output["taskId"])
		assert.Equal(t, "2025-11-29T10:00:00Z", result.Output["timestamp"])

		// Verify nested data is preserved
		nestedMap, ok := result.Output["nested"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value1", nestedMap["field1"])
		assert.Equal(t, 42, nestedMap["field2"])

		// Verify sleep metadata is also present
		assert.Contains(t, result.Output, "sleep_scheduled_until")
		assert.Contains(t, result.Output, "sleep_duration_ms")
		assert.Equal(t, "relative", result.Output["mode"])
	})

	t.Run("Input passthrough - absolute mode (past time)", func(t *testing.T) {
		inputData := map[string]interface{}{
			"data": map[string]interface{}{
				"userId": 99999,
				"action": "test",
			},
		}

		pastTime := time.Now().UTC().Add(-24 * time.Hour)
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":        "absolute",
				"target_date": pastTime.Format(time.RFC3339),
			},
			Input:       inputData,
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)
		assert.False(t, result.NeedsSleep) // Should NOT need sleep

		// Verify input data fields are merged into output
		assert.Equal(t, 99999, result.Output["userId"])
		assert.Equal(t, "test", result.Output["action"])

		// Verify sleep metadata is also present
		assert.True(t, result.Output["sleep_skipped"].(bool))
	})

	t.Run("Input passthrough - nil input", func(t *testing.T) {
		execCtx := ExecutionContext{
			NodeID: "sleep-node",
			NodeConfig: map[string]interface{}{
				"mode":           "relative",
				"duration_value": 1.0,
				"duration_unit":  "seconds",
			},
			Input:       nil, // No input data
			ExecutionID: "test-execution",
			AccountID:   "test-account",
		}

		result, err := executor.Execute(context.Background(), execCtx)

		assert.Nil(t, err)
		assert.True(t, result.Success)

		// With nil input, only sleep metadata should be present
		assert.Contains(t, result.Output, "sleep_scheduled_until")
		assert.NotContains(t, result.Output, "userId") // No user data
	})
}
