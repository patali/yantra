package models

// Trigger type constants - centralized source of truth for workflow execution trigger types
const (
	// TriggerTypeManual indicates the workflow was manually triggered via API
	TriggerTypeManual = "manual"

	// TriggerTypeScheduled indicates the workflow was triggered by a cron schedule
	TriggerTypeScheduled = "scheduled"

	// TriggerTypeWebhook indicates the workflow was triggered by a webhook call
	TriggerTypeWebhook = "webhook"

	// TriggerTypeResume indicates the workflow was resumed from a checkpoint
	TriggerTypeResume = "resume"
)

// AllTriggerTypes contains all valid trigger types for validation
var AllTriggerTypes = []string{
	TriggerTypeManual,
	TriggerTypeScheduled,
	TriggerTypeWebhook,
	TriggerTypeResume,
}

// IsValidTriggerType returns true if the trigger type is valid
func IsValidTriggerType(triggerType string) bool {
	for _, t := range AllTriggerTypes {
		if t == triggerType {
			return true
		}
	}
	return false
}
