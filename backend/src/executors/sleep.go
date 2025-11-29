package executors

import (
	"context"
	"fmt"
	"time"
)

type SleepExecutor struct{}

func NewSleepExecutor() *SleepExecutor {
	return &SleepExecutor{}
}

func (e *SleepExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Get mode from config
	mode, ok := execCtx.NodeConfig["mode"].(string)
	if !ok || mode == "" {
		return &ExecutionResult{
			Success: false,
			Error:   "mode is required (must be 'absolute' or 'relative')",
		}, nil
	}

	var wakeUpTime time.Time
	var err error

	switch mode {
	case "absolute":
		wakeUpTime, err = e.calculateAbsoluteWakeUp(execCtx.NodeConfig)
		if err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

	case "relative":
		wakeUpTime, err = e.calculateRelativeWakeUp(execCtx.NodeConfig)
		if err != nil {
			return &ExecutionResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

	default:
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("invalid mode '%s' (must be 'absolute' or 'relative')", mode),
		}, nil
	}

	// Check if wake-up time is in the past
	now := time.Now().UTC()
	if wakeUpTime.Before(now) || wakeUpTime.Equal(now) {
		// Complete immediately - target time already passed
		// Start with the input data and merge in sleep metadata
		output := e.mergeInputWithMetadata(execCtx.Input, map[string]interface{}{
			"data":          true, // Primary output: sleep completed (true/false)
			"slept_until":   wakeUpTime.Format(time.RFC3339),
			"sleep_skipped": true,
			"reason":        "target time already passed",
			"mode":          mode,
		})

		return &ExecutionResult{
			Success: true,
			Output:  output,
		}, nil
	}

	// Calculate sleep duration for output
	sleepDuration := wakeUpTime.Sub(now)

	// Start with the input data and merge in sleep metadata
	output := e.mergeInputWithMetadata(execCtx.Input, map[string]interface{}{
		"data":                  wakeUpTime.Format(time.RFC3339), // Primary output: wake-up time
		"sleep_scheduled_until": wakeUpTime.Format(time.RFC3339),
		"sleep_duration_ms":     sleepDuration.Milliseconds(),
		"mode":                  mode,
		"scheduled_at":          now.Format(time.RFC3339),
	})

	// Signal that this node needs to sleep
	return &ExecutionResult{
		Success:    true,
		NeedsSleep: true,
		WakeUpAt:   &wakeUpTime,
		Output:     output,
	}, nil
}

// calculateAbsoluteWakeUp calculates wake-up time for absolute mode
func (e *SleepExecutor) calculateAbsoluteWakeUp(config map[string]interface{}) (time.Time, error) {
	// Get target_date
	targetDateStr, ok := config["target_date"].(string)
	if !ok || targetDateStr == "" {
		return time.Time{}, fmt.Errorf("target_date is required for absolute mode")
	}

	// Parse timezone (optional, defaults to UTC)
	timezone := "UTC"
	if tz, ok := config["timezone"].(string); ok && tz != "" {
		timezone = tz
	}

	// Load timezone location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone '%s': %w", timezone, err)
	}

	// Parse target date in the specified timezone
	// Support multiple common formats
	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,      // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05", // ISO 8601 without timezone
		"2006-01-02T15:04",    // ISO 8601 without seconds and timezone
		"2006-01-02 15:04:05", // Common datetime format
		"2006-01-02 15:04",    // Common datetime format without seconds
		"2006-01-02",          // Date only (assumes 00:00:00)
	}

	var targetTime time.Time
	var parseErr error

	for _, format := range formats {
		targetTime, parseErr = time.ParseInLocation(format, targetDateStr, loc)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return time.Time{}, fmt.Errorf("invalid target_date format '%s': expected RFC3339 or ISO 8601 format", targetDateStr)
	}

	// Convert to UTC for storage
	return targetTime.UTC(), nil
}

// calculateRelativeWakeUp calculates wake-up time for relative mode
func (e *SleepExecutor) calculateRelativeWakeUp(config map[string]interface{}) (time.Time, error) {
	// Get duration_value
	durationValue, ok := config["duration_value"].(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("duration_value is required for relative mode (must be a number)")
	}

	if durationValue < 0 {
		return time.Time{}, fmt.Errorf("duration_value must be non-negative")
	}

	// Get duration_unit
	durationUnit, ok := config["duration_unit"].(string)
	if !ok || durationUnit == "" {
		return time.Time{}, fmt.Errorf("duration_unit is required for relative mode")
	}

	// Calculate duration based on unit
	var duration time.Duration
	switch durationUnit {
	case "seconds":
		duration = time.Duration(durationValue) * time.Second
	case "minutes":
		duration = time.Duration(durationValue) * time.Minute
	case "hours":
		duration = time.Duration(durationValue) * time.Hour
	case "days":
		duration = time.Duration(durationValue*24) * time.Hour
	case "weeks":
		duration = time.Duration(durationValue*24*7) * time.Hour
	default:
		return time.Time{}, fmt.Errorf("invalid duration_unit '%s' (must be 'seconds', 'minutes', 'hours', 'days', or 'weeks')", durationUnit)
	}

	// Parse timezone (optional, for display purposes - calculation is timezone-agnostic)
	timezone := "UTC"
	if tz, ok := config["timezone"].(string); ok && tz != "" {
		timezone = tz
		// Validate timezone
		if _, err := time.LoadLocation(timezone); err != nil {
			return time.Time{}, fmt.Errorf("invalid timezone '%s': %w", timezone, err)
		}
	}

	// Calculate wake-up time from now
	now := time.Now().UTC()
	wakeUpTime := now.Add(duration)

	return wakeUpTime, nil
}

// mergeInputWithMetadata merges the input data with sleep metadata
// If input is a map, it spreads the input fields into the output alongside metadata
// If input.data exists and is a map, those fields are also merged at the top level
func (e *SleepExecutor) mergeInputWithMetadata(input interface{}, metadata map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{})

	// First, copy metadata (this includes the sleep node's "data" field)
	for k, v := range metadata {
		output[k] = v
	}

	// Then merge input data
	if inputMap, ok := input.(map[string]interface{}); ok {
		// If input has a "data" field that's a map, merge those fields at top level
		if inputData, ok := inputMap["data"].(map[string]interface{}); ok {
			for k, v := range inputData {
				if _, exists := output[k]; !exists { // Don't override metadata fields
					output[k] = v
				}
			}
		}

		// Also merge other top-level fields from input (like from conditionals, loops, etc.)
		for k, v := range inputMap {
			if k != "data" { // Skip "data" as we handled it above
				if _, exists := output[k]; !exists { // Don't override metadata fields
					output[k] = v
				}
			}
		}
	}

	return output
}
