package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/patali/yantra/src/db/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// SchedulerService manages cron-based workflow scheduling
type SchedulerService struct {
	db           *gorm.DB
	queueService *QueueService
	cron         *cron.Cron
	schedules    map[string]cron.EntryID // workflowID -> cron entryID
	mu           sync.RWMutex
	running      bool
}

// TimezoneSchedule wraps a cron.Schedule to execute in a specific timezone
type TimezoneSchedule struct {
	schedule cron.Schedule
	location *time.Location
}

// Next returns the next time the schedule should run, adjusted for timezone
func (ts *TimezoneSchedule) Next(t time.Time) time.Time {
	// Convert current time to the target timezone
	tInZone := t.In(ts.location)

	// Get next scheduled time in the target timezone
	next := ts.schedule.Next(tInZone)

	// Convert back to local time for the cron scheduler
	return next
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(db *gorm.DB, queueService *QueueService) *SchedulerService {
	return &SchedulerService{
		db:           db,
		queueService: queueService,
		cron:         cron.New(cron.WithSeconds()), // Support seconds-level precision
		schedules:    make(map[string]cron.EntryID),
		running:      false,
	}
}

// Start starts the scheduler
func (s *SchedulerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	// Load all active scheduled workflows from database
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()
	s.running = true

	// Start a goroutine to periodically sync schedules from database
	go s.syncSchedulesLoop(ctx)

	// Start a goroutine to poll for sleep schedule wake-ups
	go s.pollSleepSchedules(ctx)

	return nil
}

// Stop stops the scheduler
func (s *SchedulerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	cronCtx := s.cron.Stop()
	<-cronCtx.Done()

	s.running = false
	return nil
}

// loadSchedules loads all scheduled workflows from the database
func (s *SchedulerService) loadSchedules(ctx context.Context) error {
	var workflows []models.Workflow

	// Load all active workflows with schedules
	// Only active workflows should be scheduled
	// Query for workflows where schedule is not null, not empty, and IsActive is true
	err := s.db.Where("schedule IS NOT NULL AND schedule <> '' AND is_active = ?", true).Find(&workflows).Error
	if err != nil {
		return err
	}

	for _, workflow := range workflows {
		if err := s.addWorkflowSchedule(workflow.ID, *workflow.Schedule, workflow.Timezone); err != nil {
			log.Printf("Failed to schedule workflow %s (%s): %v", workflow.ID, workflow.Name, err)
		}
	}

	return nil
}

// addWorkflowSchedule adds a workflow to the cron scheduler
func (s *SchedulerService) addWorkflowSchedule(workflowID, cronExpr, timezone string) error {
	// Remove existing schedule if it exists
	s.removeWorkflowSchedule(workflowID)

	// Parse and validate cron expression
	// River/standard cron format: "second minute hour day month weekday"
	// For simplicity, we'll use standard 5-field cron and prepend "0" for seconds
	if len(cronExpr) > 0 && !s.hasSixFields(cronExpr) {
		cronExpr = "0 " + cronExpr // Add seconds field
	}

	// Load timezone location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	// Create the job function
	job := func() {
		ctx := context.Background()

		// Get workflow and version info
		var workflow models.Workflow
		if err := s.db.First(&workflow, "id = ?", workflowID).Error; err != nil {
			log.Printf("Failed to find workflow %s: %v", workflowID, err)
			return
		}

		var latestVersion models.WorkflowVersion
		if err := s.db.Where("workflow_id = ?", workflowID).
			Order("version DESC").
			First(&latestVersion).Error; err != nil {
			log.Printf("Failed to find version for workflow %s: %v", workflowID, err)
			return
		}

		// Create execution record
		execution := models.WorkflowExecution{
			WorkflowID:  workflowID,
			Version:     latestVersion.Version,
			Status:      "queued",
			TriggerType: models.TriggerTypeScheduled,
		}

		if err := s.db.Create(&execution).Error; err != nil {
			log.Printf("Failed to create execution record for workflow %s: %v", workflowID, err)
			return
		}

		_, err := s.queueService.QueueWorkflowExecution(ctx, workflowID, execution.ID, map[string]interface{}{}, "scheduled")
		if err != nil {
			log.Printf("Failed to queue scheduled workflow %s: %v", workflowID, err)
			// Mark execution as failed
			s.db.Model(&execution).Updates(map[string]interface{}{
				"status": "error",
				"error":  "Failed to queue for execution",
			})
		}
	}

	// Parse the cron expression with timezone awareness
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	// Create a timezone-aware schedule wrapper
	timezoneSchedule := &TimezoneSchedule{
		schedule: schedule,
		location: loc,
	}

	// Add to cron scheduler with timezone-aware schedule
	entryID := s.cron.Schedule(timezoneSchedule, cron.FuncJob(job))

	// Store mapping
	s.schedules[workflowID] = entryID

	return nil
}

// removeWorkflowSchedule removes a workflow from the cron scheduler
func (s *SchedulerService) removeWorkflowSchedule(workflowID string) {
	if entryID, exists := s.schedules[workflowID]; exists {
		s.cron.Remove(entryID)
		delete(s.schedules, workflowID)
	}
}

// AddSchedule adds or updates a workflow schedule
func (s *SchedulerService) AddSchedule(workflowID, cronExpr, timezone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler not running")
	}

	return s.addWorkflowSchedule(workflowID, cronExpr, timezone)
}

// RemoveSchedule removes a workflow schedule
func (s *SchedulerService) RemoveSchedule(workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeWorkflowSchedule(workflowID)
	return nil
}

// UpdateSchedule updates an existing workflow schedule
func (s *SchedulerService) UpdateSchedule(workflowID, cronExpr, timezone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeWorkflowSchedule(workflowID)
	return s.addWorkflowSchedule(workflowID, cronExpr, timezone)
}

// GetScheduledWorkflows returns all currently scheduled workflows
func (s *SchedulerService) GetScheduledWorkflows() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflowIDs := make([]string, 0, len(s.schedules))
	for workflowID := range s.schedules {
		workflowIDs = append(workflowIDs, workflowID)
	}
	return workflowIDs
}

// syncSchedulesLoop periodically syncs schedules from the database
func (s *SchedulerService) syncSchedulesLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Sync every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.syncSchedules(ctx); err != nil {
				log.Printf("Error syncing schedules: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// syncSchedules syncs schedules with the database
func (s *SchedulerService) syncSchedules(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var workflows []models.Workflow
	// Load all active workflows with schedules
	// Only active workflows should be scheduled
	// Query for workflows where schedule is not null, not empty, and IsActive is true
	err := s.db.Where("schedule IS NOT NULL AND schedule <> '' AND is_active = ?", true).Find(&workflows).Error
	if err != nil {
		return err
	}

	// Create a set of current workflow IDs in database
	dbWorkflows := make(map[string]models.Workflow)
	for _, w := range workflows {
		dbWorkflows[w.ID] = w
	}

	// Remove schedules that no longer exist in DB
	for workflowID := range s.schedules {
		if _, exists := dbWorkflows[workflowID]; !exists {
			s.removeWorkflowSchedule(workflowID)
		}
	}

	// Add or update schedules from DB
	for workflowID, workflow := range dbWorkflows {
		// Check if schedule needs updating
		if _, exists := s.schedules[workflowID]; !exists || s.scheduleChanged(workflowID, *workflow.Schedule) {
			if err := s.addWorkflowSchedule(workflowID, *workflow.Schedule, workflow.Timezone); err != nil {
				log.Printf("Failed to sync schedule for workflow %s: %v", workflowID, err)
			}
		}
	}

	return nil
}

// scheduleChanged checks if a workflow's schedule has changed
func (s *SchedulerService) scheduleChanged(workflowID, newCron string) bool {
	// For simplicity, we'll always refresh
	// In production, you'd want to track the cron expression and compare
	return true
}

// hasSixFields checks if a cron expression has 6 fields (includes seconds)
func (s *SchedulerService) hasSixFields(cronExpr string) bool {
	fields := 0
	inField := false

	for _, char := range cronExpr {
		if char == ' ' {
			if inField {
				fields++
				inField = false
			}
		} else {
			inField = true
		}
	}

	if inField {
		fields++
	}

	return fields >= 6
}

// ScheduleSleepWakeUp schedules a one-time wake-up for a sleeping workflow
func (s *SchedulerService) ScheduleSleepWakeUp(executionID, workflowID, nodeID string, wakeUpAt time.Time) error {
	// Create sleep schedule record in database
	sleepSchedule := models.SleepSchedule{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		WakeUpAt:    wakeUpAt,
	}

	if err := s.db.Create(&sleepSchedule).Error; err != nil {
		return fmt.Errorf("failed to create sleep schedule: %w", err)
	}

	return nil
}

// pollSleepSchedules polls for sleep schedules that need to wake up
func (s *SchedulerService) pollSleepSchedules(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.processSleepSchedules(ctx); err != nil {
				log.Printf("Error processing sleep schedules: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// processSleepSchedules processes sleep schedules that are ready to wake up
func (s *SchedulerService) processSleepSchedules(ctx context.Context) error {
	now := time.Now().UTC()

	// Get all sleep schedules where wake_up_at <= now
	var schedules []models.SleepSchedule
	if err := s.db.Where("wake_up_at <= ?", now).Find(&schedules).Error; err != nil {
		return fmt.Errorf("failed to query sleep schedules: %w", err)
	}

	if len(schedules) == 0 {
		return nil
	}

	for _, schedule := range schedules {
		// Resume the workflow execution
		if err := s.resumeWorkflowFromSleep(ctx, schedule.ExecutionID, schedule.WorkflowID); err != nil {
			log.Printf("Failed to resume execution %s: %v", schedule.ExecutionID, err)
			// Continue to next schedule even if this one fails
			continue
		}

		// Delete the sleep schedule after successful resume
		if err := s.db.Delete(&schedule).Error; err != nil {
			log.Printf("Failed to delete sleep schedule %s: %v", schedule.ID, err)
			// Don't return error - the execution is already resumed
		}
	}

	return nil
}

// resumeWorkflowFromSleep resumes a workflow execution from sleeping state
func (s *SchedulerService) resumeWorkflowFromSleep(ctx context.Context, executionID, workflowID string) error {
	// Get the execution
	var execution models.WorkflowExecution
	if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
		return fmt.Errorf("failed to find execution: %w", err)
	}

	// Verify it's in sleeping state
	if execution.Status != "sleeping" {
		return nil
	}

	// Get workflow to find version
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", workflowID).Error; err != nil {
		return fmt.Errorf("failed to find workflow: %w", err)
	}

	// Parse original input from execution record
	// This is crucial - we need to pass the original input when resuming
	// so that the workflow can continue with the same input it started with
	var input map[string]interface{}
	if execution.Input != nil && *execution.Input != "" {
		if err := json.Unmarshal([]byte(*execution.Input), &input); err != nil {
			// Continue with empty input rather than failing
			input = map[string]interface{}{}
		}
	} else {
		input = map[string]interface{}{}
	}

	// Queue the execution for resumption with the ORIGINAL input FIRST
	// This ensures that if queueing fails, we don't change the status
	// The workflow engine will update status to "running" when it starts executing
	_, err := s.queueService.QueueWorkflowExecution(ctx, workflowID, executionID, input, "resume_from_sleep")
	if err != nil {
		return fmt.Errorf("failed to queue execution: %w", err)
	}

	// Now that queueing succeeded, update execution status to running
	// This prevents the poller from trying to resume the same execution again
	if err := s.db.Model(&execution).Update("status", "running").Error; err != nil {
		// If status update fails, the workflow will still execute from the queue
		// but the poller might try to queue it again on the next poll
		// This is acceptable as River will handle duplicate jobs
		// Don't return error - the execution is already queued
	}

	return nil
}
