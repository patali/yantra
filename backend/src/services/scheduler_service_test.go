package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/patali/yantra/src/db/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSchedulerTestDB(t *testing.T) *gorm.DB {
	// Use test database from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/yantra_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(testDBURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v\nMake sure PostgreSQL is running and TEST_DATABASE_URL is set", err)
	}

	// Clean up test data before running tests
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")

	err = db.AutoMigrate(&models.Workflow{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Set PostgreSQL UUID defaults
	db.Exec("ALTER TABLE workflows ALTER COLUMN id SET DEFAULT gen_random_uuid()")

	// Clean up after test
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

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
	userID := "00000000-0000-0000-0000-000000000001" // Valid UUID for testing
	workflowID1 := "00000000-0000-0000-0000-000000000010" // Valid UUID for workflow-1
	workflowID2 := "00000000-0000-0000-0000-000000000020" // Valid UUID for workflow-2
	workflowID3 := "00000000-0000-0000-0000-000000000030" // Valid UUID for workflow-3

	w1 := models.Workflow{
		ID:             workflowID1,
		Name:           "Test Workflow 1",
		Schedule:       &schedule1,
		Timezone:       "UTC",
		IsActive:       true,
		CurrentVersion: 1,
		CreatedBy:      userID,
	}
	err := db.Create(&w1).Error
	assert.NoError(t, err)

	w2 := models.Workflow{
		ID:             workflowID2,
		Name:           "Test Workflow 2",
		Schedule:       &schedule2,
		Timezone:       "UTC",
		IsActive:       true,
		CurrentVersion: 1,
		CreatedBy:      userID,
	}
	err = db.Create(&w2).Error
	assert.NoError(t, err)

	// Create inactive workflow (shouldn't be scheduled)
	schedule3 := "*/1 * * * *"
	w3 := models.Workflow{
		ID:             workflowID3,
		Name:           "Inactive Workflow",
		Schedule:       &schedule3,
		Timezone:       "UTC",
		CurrentVersion: 1,
		CreatedBy:      userID,
	}
	err = db.Create(&w3).Error
	assert.NoError(t, err)

	// Explicitly set IsActive to false (database default is true)
	err = db.Model(&w3).Update("is_active", false).Error
	assert.NoError(t, err)

	// Verify the inactive workflow was created with IsActive = false
	var verifyWorkflow models.Workflow
	db.First(&verifyWorkflow, "id = ?", workflowID3)
	assert.False(t, verifyWorkflow.IsActive, "Workflow 3 should be inactive")

	queueService := &QueueService{}
	scheduler := NewSchedulerService(db, queueService)
	ctx := context.Background()

	// Start scheduler (will load schedules from DB)
	err = scheduler.Start(ctx)
	assert.NoError(t, err)
	defer scheduler.Stop(ctx)

	// Give it a moment to load
	time.Sleep(100 * time.Millisecond)

	// Check loaded schedules
	workflows := scheduler.GetScheduledWorkflows()
	assert.Contains(t, workflows, workflowID1)
	assert.Contains(t, workflows, workflowID2)
	assert.NotContains(t, workflows, workflowID3) // inactive workflow
}
