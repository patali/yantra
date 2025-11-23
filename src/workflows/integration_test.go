// +build integration

package workflows_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/executors"
	"github.com/patali/yantra/src/services"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDatabase holds test database connection
type TestDatabase struct {
	db *gorm.DB
}

// setupTestDB creates a fresh test database for integration tests
func setupTestDB(t *testing.T) *TestDatabase {
	// Use test database URL from environment or default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/yantra_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Drop and recreate schema for clean state
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")

	// Run migrations
	err = db.AutoMigrate(
		&models.Account{},
		&models.AccountMember{},
		&models.User{},
		&models.Workflow{},
		&models.WorkflowVersion{},
		&models.WorkflowExecution{},
		&models.WorkflowNodeExecution{},
		&models.SleepSchedule{},
		&models.OutboxMessage{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return &TestDatabase{db: db}
}

// cleanup closes the database connection
func (td *TestDatabase) cleanup() {
	sqlDB, _ := td.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// MockEmailService for testing
type MockEmailService struct {
	SentEmails []executors.EmailOptions
}

func (m *MockEmailService) SendEmail(ctx context.Context, accountID string, options executors.EmailOptions) (*executors.EmailResult, error) {
	m.SentEmails = append(m.SentEmails, options)
	return &executors.EmailResult{
		Success:   true,
		MessageID: "mock-message-id",
	}, nil
}

// WorkflowTestCase represents a complete workflow test scenario
type WorkflowTestCase struct {
	Name               string
	WorkflowDefinition string // JSON definition
	InputData          map[string]interface{}
	ExpectedOutputs    map[string]interface{} // Expected node outputs
	ExpectedStatus     string                 // success, failed
	ExpectedError      string                 // If status is failed
	Timeout            time.Duration
}

// LoadWorkflowFromFile loads a workflow definition from testdata
func LoadWorkflowFromFile(t *testing.T, filename string) string {
	data, err := os.ReadFile(fmt.Sprintf("testdata/workflows/%s", filename))
	if err != nil {
		t.Fatalf("Failed to load workflow file %s: %v", filename, err)
	}
	return string(data)
}

// LoadFixtureFromFile loads test fixture data
func LoadFixtureFromFile(t *testing.T, filename string) map[string]interface{} {
	data, err := os.ReadFile(fmt.Sprintf("testdata/fixtures/%s", filename))
	if err != nil {
		t.Fatalf("Failed to load fixture file %s: %v", filename, err)
	}

	var fixture map[string]interface{}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("Failed to parse fixture file %s: %v", filename, err)
	}

	return fixture
}

// CreateTestWorkflow creates a workflow in the database
func CreateTestWorkflow(t *testing.T, db *gorm.DB, accountID, userID, definition string) *models.Workflow {
	workflow := &models.Workflow{
		Name:           "Test Workflow",
		Description:    stringPtr("Integration test workflow"),
		IsActive:       true,
		CurrentVersion: 1,
		AccountID:      &accountID,
		CreatedBy:      userID,
	}

	if err := db.Create(workflow).Error; err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Create workflow version
	version := &models.WorkflowVersion{
		WorkflowID: workflow.ID,
		Version:    1,
		Definition: definition,
	}

	if err := db.Create(version).Error; err != nil {
		t.Fatalf("Failed to create workflow version: %v", err)
	}

	return workflow
}

// ExecuteTestWorkflow executes a workflow and waits for completion
func ExecuteTestWorkflow(t *testing.T, db *gorm.DB, engineService *services.WorkflowEngineService, workflow *models.Workflow, inputData map[string]interface{}, timeout time.Duration) (*models.WorkflowExecution, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	inputJSON := "{}"
	if inputData != nil {
		data, _ := json.Marshal(inputData)
		inputJSON = string(data)
	}

	// Create execution record with valid UUID
	executionID := uuid.New().String()
	execution := &models.WorkflowExecution{
		ID:          executionID,
		WorkflowID:  workflow.ID,
		Version:     workflow.CurrentVersion,
		Status:      "queued",
		TriggerType: "manual",
	}
	if inputJSON != "" && inputJSON != "{}" {
		execution.Input = &inputJSON
	}
	if err := db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Execute workflow
	err := engineService.ExecuteWorkflow(ctx, workflow.ID, executionID, inputJSON, "manual")
	if err != nil {
		return execution, err
	}

	// Get updated execution result
	var updatedExecution models.WorkflowExecution
	if err := db.First(&updatedExecution, "id = ?", executionID).Error; err != nil {
		return execution, err
	}

	return &updatedExecution, nil
}

func stringPtr(s string) *string {
	return &s
}

// TestSimpleTransformWorkflow tests a basic transform workflow
func TestSimpleTransformWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	// Create test account and user
	account := &models.Account{Name: "Test Account"}
	testDB.db.Create(account)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}
	testDB.db.Create(user)

	// Create account membership
	membership := &models.AccountMember{
		AccountID: account.ID,
		UserID:    user.ID,
		Role:      "owner",
	}
	testDB.db.Create(membership)

	// Create mock services
	mockEmailService := &MockEmailService{}
	engineService := services.NewWorkflowEngineService(testDB.db, mockEmailService)

	// Create scheduler service for sleep node support
	queueService := &services.QueueService{}
	schedulerService := services.NewSchedulerService(testDB.db, queueService)
	ctx := context.Background()
	err := schedulerService.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler service: %v", err)
	}
	defer schedulerService.Stop(ctx)
	engineService.SetSchedulerService(schedulerService)

	// Define a simple transform workflow
	workflowDef := `{
		"nodes": [
			{
				"id": "start-1",
				"type": "start",
				"label": "Start",
				"position": {"x": 100, "y": 100},
				"data": {
					"label": "Start",
					"config": {}
				}
			},
			{
				"id": "transform-1",
				"type": "transform",
				"label": "Transform Data",
				"position": {"x": 300, "y": 100},
				"data": {
					"label": "Transform Data",
					"config": {
						"operations": [
							{
								"type": "map",
								"config": {
									"mappings": {
										"firstName": "first_name",
										"lastName": "last_name"
									}
								}
							}
						]
					}
				}
			},
			{
				"id": "end-1",
				"type": "end",
				"label": "End",
				"position": {"x": 500, "y": 100},
				"data": {
					"label": "End",
					"config": {}
				}
			}
		],
		"edges": [
			{"id": "e1", "source": "start-1", "target": "transform-1"},
			{"id": "e2", "source": "transform-1", "target": "end-1"}
		]
	}`

	// Create workflow
	workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

	// Execute workflow with input data
	inputData := map[string]interface{}{
		"firstName": "John",
		"lastName":  "Doe",
		"age":       30,
	}

	execution, err := ExecuteTestWorkflow(t, testDB.db, engineService, workflow, inputData, 10*time.Second)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, execution)
	// Add more assertions based on your execution model
}

// TestConditionalBranchingWorkflow tests conditional logic
func TestConditionalBranchingWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	t.Skip("Conditional branching test not yet implemented")
}

// TestLoopProcessingWorkflow tests loop execution
func TestLoopProcessingWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	t.Skip("Loop processing test not yet implemented")
}

// TestErrorHandlingWorkflow tests error propagation
func TestErrorHandlingWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	t.Skip("Error handling test not yet implemented")
}

// TestSleepNodeWorkflow tests sleep node functionality
func TestSleepNodeWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	// Create test account and user
	account := &models.Account{Name: "Test Account"}
	testDB.db.Create(account)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}
	testDB.db.Create(user)

	// Create account membership
	membership := &models.AccountMember{
		AccountID: account.ID,
		UserID:    user.ID,
		Role:      "owner",
	}
	testDB.db.Create(membership)

	// Create mock services
	mockEmailService := &MockEmailService{}
	engineService := services.NewWorkflowEngineService(testDB.db, mockEmailService)

	// Create scheduler service for sleep node support
	queueService := &services.QueueService{}
	schedulerService := services.NewSchedulerService(testDB.db, queueService)
	ctx := context.Background()
	err := schedulerService.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler service: %v", err)
	}
	defer schedulerService.Stop(ctx)
	engineService.SetSchedulerService(schedulerService)

	t.Run("Sleep node with relative mode (5 seconds)", func(t *testing.T) {
		// Define a workflow with a sleep node
		workflowDef := `{
			"nodes": [
				{
					"id": "start-1",
					"type": "start",
					"label": "Start",
					"position": {"x": 100, "y": 100},
					"data": {
						"label": "Start",
						"config": {}
					}
				},
				{
					"id": "json-1",
					"type": "json",
					"label": "Input Data",
					"position": {"x": 250, "y": 100},
					"data": {
						"label": "Input Data",
						"config": {
							"data": {"message": "before sleep"}
						}
					}
				},
				{
					"id": "sleep-1",
					"type": "sleep",
					"label": "Sleep 5 seconds",
					"position": {"x": 400, "y": 100},
					"data": {
						"label": "Sleep 5 seconds",
						"config": {
							"mode": "relative",
							"duration_value": 5,
							"duration_unit": "seconds"
						}
					}
				},
				{
					"id": "json-2",
					"type": "json",
					"label": "After Sleep",
					"position": {"x": 550, "y": 100},
					"data": {
						"label": "After Sleep",
						"config": {
							"data": {"message": "after sleep"}
						}
					}
				},
				{
					"id": "end-1",
					"type": "end",
					"label": "End",
					"position": {"x": 700, "y": 100},
					"data": {
						"label": "End",
						"config": {}
					}
				}
			],
			"edges": [
				{"id": "e1", "source": "start-1", "target": "json-1"},
				{"id": "e2", "source": "json-1", "target": "sleep-1"},
				{"id": "e3", "source": "sleep-1", "target": "json-2"},
				{"id": "e4", "source": "json-2", "target": "end-1"}
			]
		}`

		// Create workflow
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		// Create execution record
		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		// Execute workflow (it should hit sleep node and enter sleeping state)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		err = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Workflow should stop at sleep node (error indicates workflow entered sleeping state)
		if err != nil {
			assert.Contains(t, err.Error(), "sleeping state")
		}

		// Verify execution status is "sleeping"
		var updatedExecution models.WorkflowExecution
		err = testDB.db.First(&updatedExecution, "id = ?", execution.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "sleeping", updatedExecution.Status, "Execution should be in sleeping state")

		// Verify sleep schedule was created
		var sleepSchedules []models.SleepSchedule
		err = testDB.db.Where("execution_id = ?", execution.ID).Find(&sleepSchedules).Error
		assert.NoError(t, err)
		assert.Len(t, sleepSchedules, 1, "One sleep schedule should be created")

		if len(sleepSchedules) > 0 {
			schedule := sleepSchedules[0]
			assert.Equal(t, workflow.ID, schedule.WorkflowID)
			assert.Equal(t, "sleep-1", schedule.NodeID)

			// Wake up time should be approximately 5 seconds in the future
			expectedWakeUp := time.Now().UTC().Add(5 * time.Second)
			assert.WithinDuration(t, expectedWakeUp, schedule.WakeUpAt, 2*time.Second)
		}

		// Verify sleep node execution was successful
		var nodeExecutions []models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND node_id = ?", execution.ID, "sleep-1").Find(&nodeExecutions).Error
		assert.NoError(t, err)
		assert.Len(t, nodeExecutions, 1, "Sleep node should have one execution record")

		if len(nodeExecutions) > 0 {
			nodeExec := nodeExecutions[0]
			assert.Equal(t, "success", nodeExec.Status, "Sleep node execution should be successful")
			assert.NotNil(t, nodeExec.Output, "Sleep node should have output")

			// Verify output contains sleep metadata
			var output map[string]interface{}
			if nodeExec.Output != nil {
				json.Unmarshal([]byte(*nodeExec.Output), &output)
				assert.Contains(t, output, "sleep_scheduled_until")
				assert.Contains(t, output, "sleep_duration_ms")
				assert.Equal(t, "relative", output["mode"])
			}
		}
	})

	t.Run("Sleep node with absolute mode (past date)", func(t *testing.T) {
		// Define a workflow with a sleep node set to past date
		pastDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		workflowDef := fmt.Sprintf(`{
			"nodes": [
				{
					"id": "start-1",
					"type": "start",
					"label": "Start",
					"position": {"x": 100, "y": 100},
					"data": {
						"label": "Start",
						"config": {}
					}
				},
				{
					"id": "sleep-1",
					"type": "sleep",
					"label": "Sleep (past date)",
					"position": {"x": 300, "y": 100},
					"data": {
						"label": "Sleep (past date)",
						"config": {
							"mode": "absolute",
							"target_date": "%s"
						}
					}
				},
				{
					"id": "end-1",
					"type": "end",
					"label": "End",
					"position": {"x": 500, "y": 100},
					"data": {
						"label": "End",
						"config": {}
					}
				}
			],
			"edges": [
				{"id": "e1", "source": "start-1", "target": "sleep-1"},
				{"id": "e2", "source": "sleep-1", "target": "end-1"}
			]
		}`, pastDate)

		// Create workflow
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		// Create execution record
		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		// Execute workflow
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		err = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Workflow should complete normally (past date means no sleep)
		// Note: Actual behavior depends on implementation - might complete or have different behavior

		// Verify no sleep schedule was created (since date is in the past)
		var sleepSchedules []models.SleepSchedule
		err = testDB.db.Where("execution_id = ?", execution.ID).Find(&sleepSchedules).Error
		assert.NoError(t, err)
		assert.Len(t, sleepSchedules, 0, "No sleep schedule should be created for past dates")

		// Verify sleep node execution shows it was skipped
		var nodeExecutions []models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND node_id = ?", execution.ID, "sleep-1").Find(&nodeExecutions).Error
		assert.NoError(t, err)

		if len(nodeExecutions) > 0 {
			nodeExec := nodeExecutions[0]
			assert.Equal(t, "success", nodeExec.Status)

			// Verify output indicates sleep was skipped
			if nodeExec.Output != nil {
				var output map[string]interface{}
				json.Unmarshal([]byte(*nodeExec.Output), &output)
				if skipped, ok := output["sleep_skipped"].(bool); ok {
					assert.True(t, skipped, "Sleep should be skipped for past dates")
					assert.Equal(t, "target time already passed", output["reason"])
				}
			}
		}
	})
}

// TestWorkflowExamples runs all example workflows from testdata/workflows directory
// These serve as both integration tests and usage examples
func TestWorkflowExamples(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.cleanup()

	// Create test account and user
	account := &models.Account{Name: "Test Account"}
	testDB.db.Create(account)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}
	testDB.db.Create(user)

	// Create account membership
	membership := &models.AccountMember{
		AccountID: account.ID,
		UserID:    user.ID,
		Role:      "owner",
	}
	testDB.db.Create(membership)

	// Create mock services
	mockEmailService := &MockEmailService{}
	engineService := services.NewWorkflowEngineService(testDB.db, mockEmailService)

	// Create scheduler service for sleep node support
	queueService := &services.QueueService{}
	schedulerService := services.NewSchedulerService(testDB.db, queueService)
	ctx := context.Background()
	err := schedulerService.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler service: %v", err)
	}
	defer schedulerService.Stop(ctx)
	engineService.SetSchedulerService(schedulerService)

	t.Run("sleep_relative_short", func(t *testing.T) {
		workflowDef := LoadWorkflowFromFile(t, "sleep_relative_short.json")
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		// Create execution record
		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		// Execute workflow - should enter sleeping state
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		_ = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Verify execution status is "sleeping"
		var updatedExecution models.WorkflowExecution
		err = testDB.db.First(&updatedExecution, "id = ?", execution.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "sleeping", updatedExecution.Status, "Workflow should enter sleeping state")

		// Verify sleep schedule created
		var sleepSchedules []models.SleepSchedule
		err = testDB.db.Where("execution_id = ?", execution.ID).Find(&sleepSchedules).Error
		assert.NoError(t, err)
		assert.Len(t, sleepSchedules, 1, "Sleep schedule should be created")
	})

	t.Run("sleep_absolute_past", func(t *testing.T) {
		workflowDef := LoadWorkflowFromFile(t, "sleep_absolute_past.json")
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		_ = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Verify no sleep schedule created (past date)
		var sleepSchedules []models.SleepSchedule
		err = testDB.db.Where("execution_id = ?", execution.ID).Find(&sleepSchedules).Error
		assert.NoError(t, err)
		assert.Len(t, sleepSchedules, 0, "No sleep schedule for past dates")

		// Verify sleep node output shows skipped
		var nodeExecutions []models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND node_type = ?", execution.ID, "sleep").Find(&nodeExecutions).Error
		assert.NoError(t, err)

		if len(nodeExecutions) > 0 {
			nodeExec := nodeExecutions[0]
			if nodeExec.Output != nil {
				var output map[string]interface{}
				json.Unmarshal([]byte(*nodeExec.Output), &output)
				if skipped, ok := output["sleep_skipped"].(bool); ok {
					assert.True(t, skipped, "Sleep should be skipped for past dates")
				}
			}
		}
	})

	t.Run("sleep_with_data_flow", func(t *testing.T) {
		workflowDef := LoadWorkflowFromFile(t, "sleep_with_data_flow.json")
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		_ = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Verify workflow sleeping
		var updatedExecution models.WorkflowExecution
		err = testDB.db.First(&updatedExecution, "id = ?", execution.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "sleeping", updatedExecution.Status)

		// Verify nodes before sleep were executed
		var nodeExecutions []models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND status = ?", execution.ID, "success").
			Order("started_at").Find(&nodeExecutions).Error
		assert.NoError(t, err)

		// Should have json-1, transform-1, and sleep-1 executed
		assert.GreaterOrEqual(t, len(nodeExecutions), 3, "Pre-sleep nodes should be executed")

		// Verify transform-1 produced expected output
		var transformNode models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND node_id = ?", execution.ID, "transform-1").First(&transformNode).Error
		if err == nil && transformNode.Output != nil {
			var output map[string]interface{}
			json.Unmarshal([]byte(*transformNode.Output), &output)
			assert.Contains(t, output, "data", "Transform output should contain data")
		}
	})

	t.Run("sleep_with_conditional", func(t *testing.T) {
		workflowDef := LoadWorkflowFromFile(t, "sleep_with_conditional.json")
		workflow := CreateTestWorkflow(t, testDB.db, account.ID, user.ID, workflowDef)

		execution := &models.WorkflowExecution{
			WorkflowID:  workflow.ID,
			Version:     1,
			Status:      "running",
			TriggerType: "manual",
		}
		err := testDB.db.Create(execution).Error
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		inputJSON := "{}"
		_ = engineService.ExecuteWorkflow(ctx, workflow.ID, execution.ID, inputJSON, "manual")

		// Since condition is "high" priority, workflow should NOT sleep
		// (sleep is on false branch)
		var updatedExecution models.WorkflowExecution
		err = testDB.db.First(&updatedExecution, "id = ?", execution.ID).Error
		assert.NoError(t, err)

		// Verify conditional node executed
		var conditionalNode models.WorkflowNodeExecution
		err = testDB.db.Where("execution_id = ? AND node_type = ?", execution.ID, "conditional").First(&conditionalNode).Error
		assert.NoError(t, err)
		assert.Equal(t, "success", conditionalNode.Status)
	})
}

// TestSleepWorkflowResumption tests the full sleep cycle including resumption
// This test should:
// 1. Execute workflow with sleep node
// 2. Verify sleep schedule created
// 3. Manually trigger resumption (simulating scheduler)
// 4. Verify workflow continues from checkpoint
// 5. Verify nodes after sleep execute correctly
func TestSleepWorkflowResumption(t *testing.T) {
	t.Skip("Requires scheduler service and time-based testing - implement with mock scheduler")
}
