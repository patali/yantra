package services

import (
	"context"
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

	log.Println("✅ Scheduler service started")

	// Start a goroutine to periodically sync schedules from database
	go s.syncSchedulesLoop(ctx)

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
	log.Println("✅ Scheduler service stopped")

	return nil
}

// loadSchedules loads all scheduled workflows from the database
func (s *SchedulerService) loadSchedules(ctx context.Context) error {
	var workflows []models.Workflow

	err := s.db.Where("schedule IS NOT NULL AND schedule != ? AND is_active = ?", "", true).Find(&workflows).Error
	if err != nil {
		return err
	}

	for _, workflow := range workflows {
		if err := s.addWorkflowSchedule(workflow.ID, *workflow.Schedule, workflow.Timezone); err != nil {
			log.Printf("❌ Failed to schedule workflow %s: %v", workflow.ID, err)
		}
	}

	log.Printf("📅 Loaded %d scheduled workflows", len(workflows))
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

	// Create the job function
	job := func() {
		ctx := context.Background()

		// Get workflow and version info
		var workflow models.Workflow
		if err := s.db.First(&workflow, "id = ?", workflowID).Error; err != nil {
			log.Printf("❌ Failed to find workflow %s: %v", workflowID, err)
			return
		}

		var latestVersion models.WorkflowVersion
		if err := s.db.Where("workflow_id = ?", workflowID).
			Order("version DESC").
			First(&latestVersion).Error; err != nil {
			log.Printf("❌ Failed to find version for workflow %s: %v", workflowID, err)
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
			log.Printf("❌ Failed to create execution record for workflow %s: %v", workflowID, err)
			return
		}

		_, err := s.queueService.QueueWorkflowExecution(ctx, workflowID, execution.ID, map[string]interface{}{}, "scheduled")
		if err != nil {
			log.Printf("❌ Failed to queue scheduled workflow %s: %v", workflowID, err)
			// Mark execution as failed
			s.db.Model(&execution).Updates(map[string]interface{}{
				"status": "error",
				"error":  "Failed to queue for execution",
			})
		} else {
			log.Printf("✅ Queued scheduled workflow %s (execution: %s)", workflowID, execution.ID)
		}
	}

	// Add to cron scheduler
	entryID, err := s.cron.AddFunc(cronExpr, job)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	// Store mapping
	s.schedules[workflowID] = entryID

	log.Printf("📅 Scheduled workflow %s with cron '%s'", workflowID, cronExpr)
	return nil
}

// removeWorkflowSchedule removes a workflow from the cron scheduler
func (s *SchedulerService) removeWorkflowSchedule(workflowID string) {
	if entryID, exists := s.schedules[workflowID]; exists {
		s.cron.Remove(entryID)
		delete(s.schedules, workflowID)
		log.Printf("🗑️  Removed schedule for workflow %s", workflowID)
	}
}

// AddSchedule adds or updates a workflow schedule
func (s *SchedulerService) AddSchedule(workflowID, cronExpr, timezone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
				log.Printf("❌ Error syncing schedules: %v", err)
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
	err := s.db.Where("schedule IS NOT NULL AND schedule != ? AND is_active = ?", "", true).Find(&workflows).Error
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
				log.Printf("❌ Failed to sync schedule for workflow %s: %v", workflowID, err)
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
