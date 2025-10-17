package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/internal/middleware"
	"github.com/patali/yantra/internal/services"
)

type RecoveryController struct {
	outboxService   *services.OutboxService
	workflowService *services.WorkflowService
	workflowEngine  *services.WorkflowEngineService
}

func NewRecoveryController(
	outboxService *services.OutboxService,
	workflowService *services.WorkflowService,
	workflowEngine *services.WorkflowEngineService,
) *RecoveryController {
	return &RecoveryController{
		outboxService:   outboxService,
		workflowService: workflowService,
		workflowEngine:  workflowEngine,
	}
}

// RegisterRoutes registers recovery routes
func (ctrl *RecoveryController) RegisterRoutes(rg *gin.RouterGroup, authService *services.AuthService) {
	recovery := rg.Group("/recovery")
	recovery.Use(middleware.AuthMiddleware(authService))
	{
		// All workflow executions (runs)
		recovery.GET("/runs", ctrl.GetAllRuns)

		// Failed workflow executions
		recovery.GET("/failed-executions", ctrl.GetFailedExecutions)

		// Dead letter queue operations (for async node failures)
		recovery.GET("/dead-letter", ctrl.GetDeadLetterMessages)
		recovery.POST("/dead-letter/:messageId/retry", ctrl.RetryDeadLetterMessage)

		// Workflow restart operations
		recovery.POST("/workflows/:executionId/restart", ctrl.RestartWorkflow)

		// Workflow cancellation
		recovery.POST("/workflows/:executionId/cancel", ctrl.CancelWorkflow)

		// Node re-execution operations
		recovery.POST("/executions/:executionId/nodes/:nodeId/retry", ctrl.ReExecuteNode)
	}
}

// GetAllRuns returns all workflow executions (runs)
// GET /api/recovery/runs?status=all&limit=100
func (ctrl *RecoveryController) GetAllRuns(c *gin.Context) {
	// Get query parameters
	status := c.DefaultQuery("status", "all")
	limit := 100 // Default limit

	executions, err := ctrl.workflowService.GetAllWorkflowExecutions(limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetFailedExecutions returns all failed workflow executions
// GET /api/recovery/failed-executions
func (ctrl *RecoveryController) GetFailedExecutions(c *gin.Context) {
	executions, err := ctrl.workflowService.GetFailedWorkflowExecutions(100) // Limit to 100 executions
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetDeadLetterMessages returns all dead letter messages
// GET /api/recovery/dead-letter
func (ctrl *RecoveryController) GetDeadLetterMessages(c *gin.Context) {
	messages, err := ctrl.outboxService.GetDeadLetterMessages(100) // Limit to 100 messages
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// RetryDeadLetterMessage retries a specific dead letter message
// POST /api/recovery/dead-letter/:messageId/retry
func (ctrl *RecoveryController) RetryDeadLetterMessage(c *gin.Context) {
	messageId := c.Param("messageId")

	err := ctrl.outboxService.RetryDeadLetterMessage(messageId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dead letter message retry initiated"})
}

// RestartWorkflow restarts a failed workflow execution
// POST /api/recovery/workflows/:executionId/restart
func (ctrl *RecoveryController) RestartWorkflow(c *gin.Context) {
	executionId := c.Param("executionId")

	// Get the execution to find the workflow ID
	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	// Check if execution is in a failed state
	if execution.Status != "error" && execution.Status != "partially_failed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only failed or partially failed executions can be restarted"})
		return
	}

	// Create a new execution for the same workflow
	var input map[string]interface{}
	if execution.Input != nil {
		// Parse the input JSON if it exists
		if err := json.Unmarshal([]byte(*execution.Input), &input); err != nil {
			input = make(map[string]interface{})
		}
	} else {
		input = make(map[string]interface{})
	}

	// Execute the workflow again
	jobID, newExecutionID, err := ctrl.workflowService.ExecuteWorkflow(c.Request.Context(), execution.WorkflowID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":       jobID,
		"execution_id": newExecutionID,
		"message":      "Workflow restart initiated",
	})
}

// CancelWorkflow cancels a running workflow execution
// POST /api/recovery/workflows/:executionId/cancel
func (ctrl *RecoveryController) CancelWorkflow(c *gin.Context) {
	executionId := c.Param("executionId")

	err := ctrl.workflowService.CancelWorkflowExecution(executionId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Workflow execution cancelled successfully",
		"executionId": executionId,
	})
}

// ReExecuteNode retries a specific node execution
// POST /api/recovery/executions/:executionId/nodes/:nodeId/retry
func (ctrl *RecoveryController) ReExecuteNode(c *gin.Context) {
	executionId := c.Param("executionId")
	nodeId := c.Param("nodeId")

	// Get the execution
	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	// Find the specific node execution
	var nodeExecution *services.NodeExecutionResponse
	for _, nodeExec := range execution.NodeExecutions {
		if nodeExec.NodeID == nodeId {
			nodeExecution = &nodeExec
			break
		}
	}

	if nodeExecution == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node execution not found"})
		return
	}

	// Check if node is in a failed state
	if nodeExecution.Status != "error" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only failed nodes can be retried"})
		return
	}

	// Check if this is an asynchronous node that can be retried
	canRetry := false
	switch nodeExecution.NodeType {
	case "email", "http", "slack":
		canRetry = true
	default:
		canRetry = false
	}

	if !canRetry {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This node type cannot be retried individually"})
		return
	}

	// Parse node input
	var nodeInput map[string]interface{}
	if nodeExecution.Input != nil {
		if err := json.Unmarshal([]byte(*nodeExecution.Input), &nodeInput); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse node input"})
			return
		}
	} else {
		nodeInput = make(map[string]interface{})
	}

	// Get account ID from the workflow
	workflowModel, err := ctrl.workflowService.GetWorkflowById(execution.WorkflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow details"})
		return
	}

	accountID := workflowModel.AccountID

	// Create a new outbox message for retry
	// This will trigger the outbox worker to process the node again
	_, _, err = ctrl.outboxService.ExecuteNodeWithOutbox(
		c.Request.Context(),
		executionId,
		accountID,
		nodeId,
		nodeExecution.NodeType,
		make(map[string]interface{}), // Node config - would need to be retrieved from workflow definition
		nodeInput,
		getEventTypeForNodeType(nodeExecution.NodeType),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node retry initiated"})
}

// Helper function to get event type for node type
func getEventTypeForNodeType(nodeType string) string {
	switch nodeType {
	case "email":
		return "email.send"
	case "http":
		return "http.request"
	case "slack":
		return "slack.send"
	default:
		return "unknown"
	}
}
