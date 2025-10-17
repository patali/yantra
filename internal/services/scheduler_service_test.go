package services

import (
	"context"
	"testing"
	"time"

	"github.com/patali/yantra/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchedulerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&models.Workflow{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestSchedulerService_AddSchedule(t *testing.T) {
	db := setupSchedulerTestDB(t)

	// Mock queue service
	queueService := &QueueService{}

	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	// Start scheduler
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Add a schedule (every minute)
	err = scheduler.AddSchedule("workflow-123", "*/1 * * * *", "UTC")
	assert.NoError(t, err)

	// Check that workflow is in schedules
	workflows := scheduler.GetScheduledWorkflows()
	assert.Contains(t, workflows, "workflow-123")
}

func TestSchedulerService_RemoveSchedule(t *testing.T) {
	db := setupSchedulerTestDB(t)
	queueService := &QueueService{}
	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Add schedule
	err = scheduler.AddSchedule("workflow-123", "*/1 * * * *", "UTC")
	assert.NoError(t, err)

	// Remove schedule
	err = scheduler.RemoveSchedule("workflow-123")
	assert.NoError(t, err)

	// Verify removed
	workflows := scheduler.GetScheduledWorkflows()
	assert.NotContains(t, workflows, "workflow-123")
}

func TestSchedulerService_UpdateSchedule(t *testing.T) {
	db := setupSchedulerTestDB(t)
	queueService := &QueueService{}
	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Add initial schedule
	err = scheduler.AddSchedule("workflow-123", "*/1 * * * *", "UTC")
	assert.NoError(t, err)

	// Update schedule
	err = scheduler.UpdateSchedule("workflow-123", "*/5 * * * *", "UTC")
	assert.NoError(t, err)

	// Verify still scheduled
	workflows := scheduler.GetScheduledWorkflows()
	assert.Contains(t, workflows, "workflow-123")
}

func TestSchedulerService_InvalidCron(t *testing.T) {
	db := setupSchedulerTestDB(t)
	queueService := &QueueService{}
	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Try to add invalid cron expression
	err = scheduler.AddSchedule("workflow-123", "invalid cron", "UTC")
	assert.Error(t, err)
}

func TestSchedulerService_LoadSchedules(t *testing.T) {
	db := setupSchedulerTestDB(t)

	// Create test workflows with schedules
	schedule1 := "*/1 * * * *"
	schedule2 := "*/5 * * * *"

	db.Create(&models.Workflow{
		ID:             "workflow-1",
		Name:           "Test Workflow 1",
		Schedule:       &schedule1,
		Timezone:       "UTC",
		IsActive:       true,
		CurrentVersion: 1,
		CreatedBy:      "user-1",
	})

	db.Create(&models.Workflow{
		ID:             "workflow-2",
		Name:           "Test Workflow 2",
		Schedule:       &schedule2,
		Timezone:       "UTC",
		IsActive:       true,
		CurrentVersion: 1,
		CreatedBy:      "user-1",
	})

	// Create inactive workflow (shouldn't be scheduled)
	schedule3 := "*/1 * * * *"
	db.Create(&models.Workflow{
		ID:             "workflow-3",
		Name:           "Inactive Workflow",
		Schedule:       &schedule3,
		Timezone:       "UTC",
		IsActive:       false,
		CurrentVersion: 1,
		CreatedBy:      "user-1",
	})

	queueService := &QueueService{}
	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	// Start scheduler (will load schedules from DB)
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Give it a moment to load
	time.Sleep(100 * time.Millisecond)

	// Check loaded schedules
	workflows := scheduler.GetScheduledWorkflows()
	assert.Contains(t, workflows, "workflow-1")
	assert.Contains(t, workflows, "workflow-2")
	assert.NotContains(t, workflows, "workflow-3") // inactive workflow
}
