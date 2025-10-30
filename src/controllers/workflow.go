package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/dto"
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
		workflows.GET("/:id/executions", ctrl.GetWorkflowExecutions)                 // Frontend endpoint
		workflows.GET("/:id/executions/:executionId", ctrl.GetWorkflowExecutionById) // Frontend endpoint
		workflows.POST("/:id/versions/restore", ctrl.RestoreVersion)                 // Frontend endpoint
		workflows.POST("/:id/duplicate", ctrl.DuplicateWorkflow)                     // Frontend endpoint
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
	id := c.Param("id")

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
	id := c.Param("id")

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
	executionId := c.Param("executionId")
	includeRecovery := c.Query("includeRecovery") == "true"

	execution, err := ctrl.workflowService.GetWorkflowExecutionById(executionId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
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

// RestoreVersion restores a workflow to a previous version
// POST /api/workflows/:id/versions/restore
func (ctrl *WorkflowController) RestoreVersion(c *gin.Context) {
	id := c.Param("id")

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
	id := c.Param("id")
	userID, _ := middleware.GetUserID(c)
	accountID, _ := middleware.GetAccountID(c)

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
