package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/patali/yantra/internal/executors"
	"github.com/patali/yantra/internal/models"
)

type OutboxWorkerService struct {
	outboxService   *OutboxService
	executorFactory *executors.ExecutorFactory
	isRunning       bool
	pollInterval    time.Duration
}

func NewOutboxWorkerService(outboxService *OutboxService, executorFactory *executors.ExecutorFactory) *OutboxWorkerService {
	return &OutboxWorkerService{
		outboxService:   outboxService,
		executorFactory: executorFactory,
		isRunning:       false,
		pollInterval:    5 * time.Second,
	}
}

// Start starts the outbox worker
func (w *OutboxWorkerService) Start(ctx context.Context) {
	if w.isRunning {
		log.Println("⚠️  Outbox worker is already running")
		return
	}

	w.isRunning = true
	log.Println("🚀 Starting outbox worker...")

	// Process immediately on start
	go w.processMessages(ctx)

	// Then poll at regular intervals
	ticker := time.NewTicker(w.pollInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if w.isRunning {
					w.processMessages(ctx)
				}
			case <-ctx.Done():
				ticker.Stop()
				w.isRunning = false
				log.Println("✅ Outbox worker stopped")
				return
			}
		}
	}()

	log.Printf("✅ Outbox worker started (polling every %v)\n", w.pollInterval)
}

// Stop stops the outbox worker
func (w *OutboxWorkerService) Stop() {
	if !w.isRunning {
		return
	}

	log.Println("🛑 Stopping outbox worker...")
	w.isRunning = false
}

// processMessages processes pending outbox messages
func (w *OutboxWorkerService) processMessages(ctx context.Context) {
	if !w.isRunning {
		return
	}

	messages, err := w.outboxService.GetPendingMessages(10)
	if err != nil {
		log.Printf("❌ Error fetching pending messages: %v\n", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("📬 Processing %d outbox messages...\n", len(messages))

	// Process messages in parallel (with reasonable concurrency)
	for _, message := range messages {
		// Process each message
		w.processMessage(ctx, message)
	}
}

// processMessage processes a single outbox message
func (w *OutboxWorkerService) processMessage(ctx context.Context, message models.OutboxMessage) {
	log.Printf("  ▶ Processing message %s (type: %s, attempt: %d)\n",
		message.ID, message.EventType, message.Attempts+1)

	// Mark as processing
	if err := w.outboxService.MarkMessageProcessing(message.ID); err != nil {
		log.Printf("  ❌ Failed to mark message as processing: %v\n", err)
		return
	}

	// Parse payload
	var payload executors.ExecutionContext
	if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
		log.Printf("  ❌ Failed to parse payload: %v\n", err)
		w.outboxService.MarkMessageFailed(message.ID, fmt.Sprintf("Invalid payload: %v", err))
		return
	}

	// Execute based on event type
	var result *executors.ExecutionResult
	var err error

	switch message.EventType {
	case "email.send":
		result, err = w.executeEmail(ctx, payload)
	case "http.request":
		result, err = w.executeHTTP(ctx, payload)
	case "slack.send":
		result, err = w.executeSlack(ctx, payload)
	default:
		err = fmt.Errorf("unknown event type: %s", message.EventType)
	}

	if err != nil {
		log.Printf("  ❌ Message %s execution error: %v\n", message.ID, err)
		w.outboxService.MarkMessageFailed(message.ID, err.Error())
		return
	}

	if !result.Success {
		log.Printf("  ❌ Message %s execution failed: %s\n", message.ID, result.Error)
		w.outboxService.MarkMessageFailed(message.ID, result.Error)
		return
	}

	// Mark as completed
	if err := w.outboxService.MarkMessageCompleted(message.ID, result.Output); err != nil {
		log.Printf("  ❌ Failed to mark message as completed: %v\n", err)
		return
	}

	log.Printf("  ✅ Message %s completed successfully\n", message.ID)
}

// executeEmail executes an email node
func (w *OutboxWorkerService) executeEmail(ctx context.Context, execCtx executors.ExecutionContext) (*executors.ExecutionResult, error) {
	executor, err := w.executorFactory.GetExecutor("email")
	if err != nil {
		return nil, err
	}

	return executor.Execute(ctx, execCtx)
}

// executeHTTP executes an HTTP node
func (w *OutboxWorkerService) executeHTTP(ctx context.Context, execCtx executors.ExecutionContext) (*executors.ExecutionResult, error) {
	executor, err := w.executorFactory.GetExecutor("http")
	if err != nil {
		return nil, err
	}

	return executor.Execute(ctx, execCtx)
}

// executeSlack executes a Slack node
func (w *OutboxWorkerService) executeSlack(ctx context.Context, execCtx executors.ExecutionContext) (*executors.ExecutionResult, error) {
	executor, err := w.executorFactory.GetExecutor("slack")
	if err != nil {
		return nil, err
	}

	return executor.Execute(ctx, execCtx)
}

// GetStats returns worker statistics
func (w *OutboxWorkerService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	integrity, err := w.outboxService.VerifyIntegrity()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"is_running":            w.isRunning,
		"poll_interval_seconds": w.pollInterval.Seconds(),
		"pending_messages":      integrity["pending_messages"],
		"processing_messages":   integrity["processing_messages"],
		"completed_messages":    integrity["completed_messages"],
		"dead_letter_messages":  integrity["dead_letter_messages"],
	}

	return stats, nil
}
