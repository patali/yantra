package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/dto"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/services"
)

type WorkflowController struct {
	workflowService *services.WorkflowService
}

func NewWorkflowController(workflowService *services.WorkflowService) *WorkflowController {
	return &WorkflowController{
		workflowService: workflowService,
	}
}

// RegisterRoutes registers workflow routes
func (ctrl *WorkflowController) RegisterRoutes(rg *gin.RouterGroup, authService *services.AuthService) {
	workflows := rg.Group("/workflows")
	workflows.Use(middleware.AuthMiddleware(authService))
	{
		workflows.GET("", ctrl.GetAllWorkflows)  // Frontend compatible (no trailing slash)
		workflows.GET("/", ctrl.GetAllWorkflows) // Alternative with slash
		workflows.POST("", ctrl.CreateWorkflow)  // Frontend compatible (no trailing slash)
		workflows.POST("/", ctrl.CreateWorkflow) // Alternative with slash
		workflows.GET("/:id", ctrl.GetWorkflowById)
		workflows.PUT("/:id", ctrl.UpdateWorkflow)
		workflows.DELETE("/:id", ctrl.DeleteWorkflow)
		workflows.POST("/:id/execute", ctrl.ExecuteWorkflow)
		workflows.PUT("/:id/schedule", ctrl.UpdateSchedule)
		workflows.GET("/:id/versions", ctrl.GetVersionHistory)
		workflows.GET("/:id/executions", ctrl.GetWorkflowExecutions)                       // Frontend endpoint
		workflows.GET("/:id/executions/:executionId", ctrl.GetWorkflowExecutionById)       // Frontend endpoint
		workflows.GET("/:id/executions/:executionId/stream", ctrl.StreamWorkflowExecution) // SSE stream endpoint
		workflows.POST("/:id/executions/:executionId/resume", ctrl.ResumeExecution)        // Resume execution endpoint
		workflows.POST("/:id/versions/restore", ctrl.RestoreVersion)                       // Frontend endpoint
		workflows.POST("/:id/duplicate", ctrl.DuplicateWorkflow)                           // Frontend endpoint
	}
}

// GetAllWorkflows returns all workflows for the current account
// GET /api/workflows
func (ctrl *WorkflowController) GetAllWorkflows(c *gin.Context) {
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workflows, err := ctrl.workflowService.GetAllWorkflows(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// GetWorkflowById returns a workflow by ID
// GET /api/workflows/:id
func (ctrl *WorkflowController) GetWorkflowById(c *gin.Context) {
	id := c.Param("id")
	accountID, _ := middleware.GetAccountID(c)

	workflow, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CreateWorkflow creates a new workflow
// POST /api/workflows
func (ctrl *WorkflowController) CreateWorkflow(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	accountID, _ := middleware.GetAccountID(c)

	var req dto.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := ctrl.workflowService.CreateWorkflow(c.Request.Context(), req, userID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// UpdateWorkflow updates a workflow
// PUT /api/workflows/:id
func (ctrl *WorkflowController) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	accountID, _ := middleware.GetAccountID(c)

	var req dto.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := ctrl.workflowService.UpdateWorkflowByAccount(id, accountID, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// DeleteWorkflow deletes a workflow
// DELETE /api/workflows/:id
func (ctrl *WorkflowController) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	accountID, _ := middleware.GetAccountID(c)

	if err := ctrl.workflowService.DeleteWorkflowByAccount(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
}

// ExecuteWorkflow executes a workflow
// POST /api/workflows/:id/execute
func (ctrl *WorkflowController) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")

	var req dto.ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID, executionID, err := ctrl.workflowService.ExecuteWorkflow(c.Request.Context(), id, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":       jobID,
		"execution_id": executionID,
		"message":      "Workflow execution queued",
	})
}

// UpdateSchedule updates workflow schedule
// PUT /api/workflows/:id/schedule
func (ctrl *WorkflowController) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.workflowService.UpdateSchedule(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated successfully"})
}

// GetVersionHistory returns version history for a workflow
// GET /api/workflows/:id/versions
func (ctrl *WorkflowController) GetVersionHistory(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	versions, err := ctrl.workflowService.GetVersionHistory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// GetWorkflowExecutions returns all executions for a workflow
// GET /api/workflows/:id/executions
func (ctrl *WorkflowController) GetWorkflowExecutions(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	executions, err := ctrl.workflowService.GetWorkflowExecutions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetWorkflowExecutionById returns a specific workflow execution with node executions
// GET /api/workflows/:id/executions/:executionId
func (ctrl *WorkflowController) GetWorkflowExecutionById(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workflowId := c.Param("id")
	executionId := c.Param("executionId")
	includeRecovery := c.Query("includeRecovery") == "true"

	// SECURITY: Verify workflow belongs to user's account
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(workflowId, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	// SECURITY: Double-check execution belongs to the workflow
	if execution.WorkflowID != workflowId {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found or access denied"})
		return
	}

	if !includeRecovery {
		c.JSON(http.StatusOK, execution)
		return
	}

	// Include recovery options
	response := gin.H{
		"execution": execution,
		"recoveryOptions": gin.H{
			"canRestartWorkflow": execution.Status == "error" || execution.Status == "partially_failed",
			"canRetryNodes":      getRetryableNodes(execution.NodeExecutions),
			"deadLetterMessages": []gin.H{}, // Will be populated by outbox service
		},
	}

	c.JSON(http.StatusOK, response)
}

// StreamWorkflowExecution streams workflow execution updates via Server-Sent Events (SSE)
// GET /api/workflows/:id/executions/:executionId/stream?token=<jwt_token>
// Note: Token passed as query param since EventSource doesn't support custom headers
// The auth middleware handles token extraction from query parameter
func (ctrl *WorkflowController) StreamWorkflowExecution(c *gin.Context) {
	executionID := c.Param("executionId")
	workflowID := c.Param("id")

	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.SSEvent("error", gin.H{"error": "Unauthorized"})
		c.Writer.Flush()
		return
	}

	// SECURITY: Verify workflow belongs to user's account
	workflow, err := ctrl.workflowService.GetWorkflowByIdAndAccount(workflowID, accountID)
	if err != nil {
		c.SSEvent("error", gin.H{"error": "Workflow not found or access denied"})
		c.Writer.Flush()
		return
	}

	// SECURITY: Verify execution belongs to this workflow
	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionID)
	if err != nil || execution.WorkflowID != workflow.ID {
		c.SSEvent("error", gin.H{"error": "Execution not found or access denied"})
		c.Writer.Flush()
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a channel to detect client disconnect
	ctx := c.Request.Context()
	ticker := time.NewTicker(1 * time.Second) // Poll every second
	defer ticker.Stop()

	var lastExecution *dto.ExecutionResponse
	var lastNodeCount int

	// Send initial connection message
	c.SSEvent("connected", gin.H{"message": "Connected to execution stream"})
	c.Writer.Flush()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Fetch current execution state (already verified ownership above)
			execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionID)
			if err != nil {
				c.SSEvent("error", gin.H{"error": "Execution not found"})
				c.Writer.Flush()
				return
			}

			// Check if execution has changed
			hasChanged := false
			if lastExecution == nil {
				hasChanged = true
			} else {
				// Check status change
				if lastExecution.Status != execution.Status {
					hasChanged = true
				}
				// Check node executions count change
				if len(execution.NodeExecutions) != lastNodeCount {
					hasChanged = true
				}
				// Check for new node executions or status changes
				if len(execution.NodeExecutions) > 0 && len(lastExecution.NodeExecutions) > 0 {
					// Compare latest node executions
					latestLastNode := lastExecution.NodeExecutions[0]
					for _, newNode := range execution.NodeExecutions {
						if newNode.ID == latestLastNode.ID {
							// Check if status changed
							if newNode.Status != latestLastNode.Status {
								hasChanged = true
								break
							}
						} else {
							// New node execution found
							hasChanged = true
							break
						}
					}
				}
			}

			if hasChanged {
				// Send update
				c.SSEvent("update", execution)
				c.Writer.Flush()

				lastExecution = execution
				lastNodeCount = len(execution.NodeExecutions)

				// If execution is complete (final state), stop streaming
				// Note: "interrupted" can be resumed, so we keep connection open for it
				if execution.Status == "success" || execution.Status == "error" || execution.Status == "partially_failed" || execution.Status == "cancelled" {
					c.SSEvent("complete", gin.H{"status": execution.Status})
					c.Writer.Flush()
					// Keep connection open for a few more seconds to catch any final updates
					time.Sleep(2 * time.Second)
					return
				}
			}

			// Send heartbeat to keep connection alive
			c.SSEvent("heartbeat", gin.H{"timestamp": time.Now().Unix()})
			c.Writer.Flush()
		}
	}
}

// ResumeExecution resumes a failed or interrupted workflow execution
// POST /api/workflows/:id/executions/:executionId/resume
func (ctrl *WorkflowController) ResumeExecution(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workflowID := c.Param("id")
	executionID := c.Param("executionId")

	// SECURITY: Verify workflow belongs to user's account
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(workflowID, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	// SECURITY: Verify execution belongs to this workflow
	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionID)
	if err != nil || execution.WorkflowID != workflowID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found or access denied"})
		return
	}

	jobID, err := ctrl.workflowService.ResumeWorkflow(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":  jobID,
		"message": "Workflow execution queued for resumption",
	})
}

// RestoreVersion restores a workflow to a previous version
// POST /api/workflows/:id/versions/restore
func (ctrl *WorkflowController) RestoreVersion(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.workflowService.RestoreWorkflowVersion(id, req.Version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version restored successfully"})
}

// DuplicateWorkflow creates a copy of an existing workflow
// POST /api/workflows/:id/duplicate
func (ctrl *WorkflowController) DuplicateWorkflow(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, exists := middleware.GetAccountID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	userID, _ := middleware.GetUserID(c)

	// SECURITY: Verify source workflow belongs to user's account (prevent IP theft!)
	_, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found or access denied"})
		return
	}

	duplicatedWorkflow, err := ctrl.workflowService.DuplicateWorkflow(c.Request.Context(), id, userID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, duplicatedWorkflow)
}

// getRetryableNodes returns a list of node IDs that can be retried
func getRetryableNodes(nodeExecutions []dto.NodeExecutionResponse) []string {
	var retryableNodes []string

	for _, nodeExec := range nodeExecutions {
		if nodeExec.Status == "error" {
			// Only asynchronous nodes can be retried individually
			switch nodeExec.NodeType {
			case "email", "http", "slack":
				retryableNodes = append(retryableNodes, nodeExec.NodeID)
			}
		}
	}

	return retryableNodes
}
