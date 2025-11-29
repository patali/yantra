package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/patali/yantra/src/db/models"
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

	// Webhook routes (public, no auth middleware, but with rate limiting)
	webhooks := rg.Group("/webhooks")
	// Apply rate limiting to webhook endpoints (60 requests per minute per IP, burst of 10)
	// This is stricter than global limit to prevent abuse while allowing legitimate usage
	webhooks.Use(middleware.RateLimitByMinute(60, 10))
	{
		webhooks.POST("/:workflowId", ctrl.TriggerWebhook)       // Default webhook endpoint
		webhooks.POST("/:workflowId/:path", ctrl.TriggerWebhook) // Custom path webhook endpoint
	}

	// Webhook secret management (requires auth)
	workflows.POST("/:id/webhook-secret/regenerate", ctrl.RegenerateWebhookSecret)

	// Example workflows (requires auth)
	rg.GET("/examples/workflows", middleware.AuthMiddleware(authService), ctrl.GetExampleWorkflows)
	rg.POST("/examples/workflows/:exampleId/duplicate", middleware.AuthMiddleware(authService), ctrl.DuplicateExampleWorkflow)
}

// GetAllWorkflows returns all workflows for the current account
// GET /api/workflows
func (ctrl *WorkflowController) GetAllWorkflows(c *gin.Context) {
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	workflows, err := ctrl.workflowService.GetAllWorkflows(accountID)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, workflows)
}

// GetWorkflowById returns a workflow by ID
// GET /api/workflows/:id
func (ctrl *WorkflowController) GetWorkflowById(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	workflow, err := ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, workflow)
}

// CreateWorkflow creates a new workflow
// POST /api/workflows
func (ctrl *WorkflowController) CreateWorkflow(c *gin.Context) {
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var req dto.CreateWorkflowRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	workflow, err := ctrl.workflowService.CreateWorkflow(c.Request.Context(), req, userID, accountID)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, workflow)
}

// UpdateWorkflow updates a workflow
// PUT /api/workflows/:id
func (ctrl *WorkflowController) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var req dto.UpdateWorkflowRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	workflow, err := ctrl.workflowService.UpdateWorkflowByAccount(id, accountID, req)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, workflow)
}

// DeleteWorkflow deletes a workflow
// DELETE /api/workflows/:id
func (ctrl *WorkflowController) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	if err := ctrl.workflowService.DeleteWorkflowByAccount(c.Request.Context(), id, accountID); err != nil {
		middleware.RespondNotFound(c, "Workflow not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
}

// ExecuteWorkflow executes a workflow
// POST /api/workflows/:id/execute
func (ctrl *WorkflowController) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")

	var req dto.ExecuteWorkflowRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	jobID, executionID, err := ctrl.workflowService.ExecuteWorkflow(c.Request.Context(), id, req.Input)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
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
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := ctrl.workflowService.UpdateSchedule(c.Request.Context(), id, req); err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Schedule updated successfully"})
}

// GetVersionHistory returns version history for a workflow
// GET /api/workflows/:id/versions
func (ctrl *WorkflowController) GetVersionHistory(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	versions, err := ctrl.workflowService.GetVersionHistory(id)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, versions)
}

// GetWorkflowExecutions returns all executions for a workflow
// GET /api/workflows/:id/executions
func (ctrl *WorkflowController) GetWorkflowExecutions(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	executions, err := ctrl.workflowService.GetWorkflowExecutions(id)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, executions)
}

// GetWorkflowExecutionById returns a specific workflow execution with node executions
// GET /api/workflows/:id/executions/:executionId
func (ctrl *WorkflowController) GetWorkflowExecutionById(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	workflowId := c.Param("id")
	executionId := c.Param("executionId")
	includeRecovery := c.Query("includeRecovery") == "true"

	// SECURITY: Verify workflow belongs to user's account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(workflowId, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionId)
	if err != nil {
		middleware.RespondNotFound(c, "Execution not found")
		return
	}

	// SECURITY: Double-check execution belongs to the workflow
	if execution.WorkflowID != workflowId {
		middleware.RespondNotFound(c, "Execution not found or access denied")
		return
	}

	if !includeRecovery {
		middleware.RespondSuccess(c, http.StatusOK, execution)
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

	middleware.RespondSuccess(c, http.StatusOK, response)
}

// StreamWorkflowExecution streams workflow execution updates via Server-Sent Events (SSE)
// GET /api/workflows/:id/executions/:executionId/stream?token=<jwt_token>
// Note: Token passed as query param since EventSource doesn't support custom headers
// The auth middleware handles token extraction from query parameter
func (ctrl *WorkflowController) StreamWorkflowExecution(c *gin.Context) {
	executionID := c.Param("executionId")
	workflowID := c.Param("id")

	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
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
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	workflowID := c.Param("id")
	executionID := c.Param("executionId")

	// SECURITY: Verify workflow belongs to user's account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(workflowID, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	// SECURITY: Verify execution belongs to this workflow
	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionID)
	if err != nil || execution.WorkflowID != workflowID {
		middleware.RespondNotFound(c, "Execution not found or access denied")
		return
	}

	jobID, err := ctrl.workflowService.ResumeWorkflow(c.Request.Context(), executionID)
	if err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"job_id":  jobID,
		"message": "Workflow execution queued for resumption",
	})
}

// RestoreVersion restores a workflow to a previous version
// POST /api/workflows/:id/versions/restore
func (ctrl *WorkflowController) RestoreVersion(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	id := c.Param("id")

	// SECURITY: Verify workflow belongs to user's account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := ctrl.workflowService.RestoreWorkflowVersion(id, req.Version); err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Version restored successfully"})
}

// DuplicateWorkflow creates a copy of an existing workflow
// POST /api/workflows/:id/duplicate
func (ctrl *WorkflowController) DuplicateWorkflow(c *gin.Context) {
	// SECURITY: Get account ID from auth middleware
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	id := c.Param("id")
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}

	// SECURITY: Verify source workflow belongs to user's account (prevent IP theft!)
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found or access denied")
		return
	}

	duplicatedWorkflow, err := ctrl.workflowService.DuplicateWorkflow(c.Request.Context(), id, userID, accountID)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, duplicatedWorkflow)
}

// GetExampleWorkflows returns all example workflow templates
// GET /api/examples/workflows
func (ctrl *WorkflowController) GetExampleWorkflows(c *gin.Context) {
	examples, err := ctrl.workflowService.GetExampleWorkflows()
	if err != nil {
		middleware.RespondInternalError(c, "Failed to fetch example workflows")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, examples)
}

// DuplicateExampleWorkflow duplicates an example workflow to user's account
// POST /api/examples/workflows/:exampleId/duplicate
func (ctrl *WorkflowController) DuplicateExampleWorkflow(c *gin.Context) {
	exampleID := c.Param("exampleId")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}

	workflow, err := ctrl.workflowService.DuplicateExampleWorkflow(c.Request.Context(), exampleID, userID, accountID)
	if err != nil {
		middleware.RespondBadRequest(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, workflow)
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

// TriggerWebhook handles webhook requests to trigger workflows
// POST /api/webhooks/:workflowId or POST /api/webhooks/:workflowId/:path
func (ctrl *WorkflowController) TriggerWebhook(c *gin.Context) {
	workflowID := c.Param("workflowId")
	webhookPath := c.Param("path")

	// SECURITY: Validate workflow ID format (must be valid UUID)
	// This prevents SQL injection and invalid ID attacks
	if _, err := uuid.Parse(workflowID); err != nil {
		middleware.RespondBadRequest(c, "Invalid workflow identifier")
		return
	}

	// SECURITY: Validate and sanitize webhook path to prevent path traversal
	// Only allow alphanumeric, hyphens, underscores, and dots
	if webhookPath != "" {
		// Check for path traversal attempts
		if strings.Contains(webhookPath, "..") || strings.Contains(webhookPath, "/") || strings.Contains(webhookPath, "\\") {
			middleware.RespondBadRequest(c, "Invalid webhook path")
			return
		}
		// Limit path length to prevent abuse
		if len(webhookPath) > 100 {
			middleware.RespondBadRequest(c, "Webhook path too long")
			return
		}
	}

	// Get workflow
	workflow, err := ctrl.workflowService.GetWorkflowById(workflowID)
	if err != nil {
		// SECURITY: Don't reveal if workflow exists or not - use generic error
		// This prevents workflow enumeration attacks
		middleware.RespondUnauthorized(c, "Invalid webhook credentials")
		return
	}

	// Check if workflow is active
	if !workflow.IsActive {
		// SECURITY: Don't reveal workflow state - use same error as invalid credentials
		middleware.RespondUnauthorized(c, "Invalid webhook credentials")
		return
	}

	// Verify webhook path matches if custom path is configured
	if workflow.WebhookPath != nil {
		// If custom path is set, it must match exactly
		if webhookPath != *workflow.WebhookPath {
			middleware.RespondUnauthorized(c, "Invalid webhook credentials")
			return
		}
	} else {
		// If no custom path, the path param should be empty
		if webhookPath != "" {
			middleware.RespondUnauthorized(c, "Invalid webhook credentials")
			return
		}
	}

	// All webhooks require authentication - verify webhook secret
	if workflow.WebhookSecretHash == nil || *workflow.WebhookSecretHash == "" {
		// SECURITY: Don't reveal configuration state
		middleware.RespondUnauthorized(c, "Invalid webhook credentials")
		return
	}

	// Get secret from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		middleware.RespondUnauthorized(c, "Authorization header required")
		return
	}

	// Extract secret (supports both "Bearer <secret>" and plain secret)
	secret := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		secret = authHeader[7:]
	}

	// SECURITY: Limit secret length to prevent DoS (bcrypt hashing is expensive)
	if len(secret) > 1024 {
		middleware.RespondUnauthorized(c, "Invalid authorization header")
		return
	}

	if secret == "" {
		middleware.RespondUnauthorized(c, "Invalid authorization header")
		return
	}

	// Validate secret against stored hash
	// SECURITY: bcrypt.CompareHashAndPassword is constant-time, safe from timing attacks
	if !ctrl.workflowService.ValidateWebhookSecret(secret, *workflow.WebhookSecretHash) {
		middleware.RespondUnauthorized(c, "Invalid webhook credentials")
		return
	}

	// Parse request body as workflow input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		// If no JSON body, use empty input
		input = make(map[string]interface{})
	}

	// Also try to get raw body for cases where content-type might not be JSON
	if len(input) == 0 {
		var rawBody interface{}
		if err := c.ShouldBindBodyWith(&rawBody, nil); err == nil {
			input = make(map[string]interface{})
			input["body"] = rawBody
		}
	}

	// SECURITY: Validate input size before processing (prevent DoS via large payloads)
	// Check size by marshaling to JSON (same check as workflow_engine)
	inputJSON, err := json.Marshal(input)
	if err != nil {
		middleware.RespondBadRequest(c, "Invalid request payload")
		return
	}

	// MaxDataSize is 10MB (10 * 1024 * 1024 bytes) as defined in workflow_engine.go
	const MaxDataSize = 10 * 1024 * 1024
	if len(inputJSON) > MaxDataSize {
		middleware.RespondBadRequest(c, "Request payload too large. Maximum size is 10MB.")
		return
	}

	// Trigger workflow execution with "webhook" trigger type
	jobID, executionID, err := ctrl.workflowService.ExecuteWorkflowWithTrigger(c.Request.Context(), workflowID, input, models.TriggerTypeWebhook)
	if err != nil {
		// SECURITY: Don't expose internal errors - use generic message
		middleware.RespondInternalError(c, "Failed to trigger workflow")
		return
	}

	middleware.RespondSuccess(c, http.StatusAccepted, gin.H{
		"job_id":       jobID,
		"execution_id": executionID,
		"message":      "Workflow execution queued",
	})
}

// RegenerateWebhookSecret generates a new webhook secret for a workflow
// POST /api/workflows/:id/webhook-secret/regenerate
func (ctrl *WorkflowController) RegenerateWebhookSecret(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	// Verify workflow belongs to account
	_, err = ctrl.workflowService.GetWorkflowByIdAndAccount(id, accountID)
	if err != nil {
		middleware.RespondNotFound(c, "Workflow not found")
		return
	}

	// Generate new secret
	plainSecret, err := ctrl.workflowService.RegenerateWebhookSecret(c.Request.Context(), id)
	if err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"secret":  plainSecret,
		"message": "Webhook secret regenerated. Save this secret securely - it cannot be retrieved again.",
	})
}
