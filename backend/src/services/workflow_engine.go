package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/executors"
	"gorm.io/gorm"
)

// Abuse prevention limits
const (
	MaxExecutionDuration = 30 * time.Minute // Maximum workflow execution time
	MaxTotalNodes        = 10000            // Maximum total nodes executed in a workflow
	MaxLoopDepth         = 5                // Maximum nested loop depth
	MaxIterations        = 10000            // Global maximum iterations per loop
	MaxAccumulatorSize   = 10 * 1024 * 1024 // 10MB max accumulated data size
	MaxDataSize          = 10 * 1024 * 1024 // 10MB max input/output size
)

type WorkflowEngineService struct {
	db               *gorm.DB
	executorFactory  *executors.ExecutorFactory
	outboxService    *OutboxService
	schedulerService *SchedulerService // Optional: for sleep node support
}

// executionLimits tracks execution limits to prevent abuse
type executionLimits struct {
	nodesExecuted int
	currentDepth  int
	startTime     time.Time
}

func NewWorkflowEngineService(db *gorm.DB, emailService executors.EmailServiceInterface) *WorkflowEngineService {
	return &WorkflowEngineService{
		db:               db,
		executorFactory:  executors.NewExecutorFactory(db, emailService),
		outboxService:    NewOutboxService(db),
		schedulerService: nil, // Set later via SetSchedulerService to avoid circular dependency
	}
}

// SetSchedulerService sets the scheduler service (called after initialization to avoid circular dependency)
func (s *WorkflowEngineService) SetSchedulerService(schedulerService *SchedulerService) {
	s.schedulerService = schedulerService
}

// checkDataSize validates that data size is within limits
func checkDataSize(data interface{}, dataType string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize %s: %w", dataType, err)
	}

	size := len(jsonData)
	if size > MaxDataSize {
		return fmt.Errorf("%s size (%d bytes) exceeds maximum allowed (%d bytes)", dataType, size, MaxDataSize)
	}

	return nil
}

// checkEdgeCondition evaluates an edge condition to determine if the target node should execute
func (s *WorkflowEngineService) checkEdgeCondition(edges []interface{}, sourceNodeID, targetNodeID string, nodeOutputs map[string]interface{}) bool {
	// Find the edge between source and target
	for _, edgeData := range edges {
		edge, ok := edgeData.(map[string]interface{})
		if !ok {
			continue
		}

		source, _ := edge["source"].(string)
		target, _ := edge["target"].(string)

		// Check if this is the edge we're looking for
		if source == sourceNodeID && target == targetNodeID {
			// Check if there's a condition on this edge
			condition, hasCondition := edge["condition"].(string)
			if !hasCondition || condition == "" {
				// No condition means always execute
				return true
			}

			// Build evaluation context with node outputs
			evalContext := make(map[string]interface{})

			// Add source node output to context
			if sourceOutput, ok := nodeOutputs[sourceNodeID].(map[string]interface{}); ok {
				// Add as "data" for easy access to conditional results
				evalContext["data"] = sourceOutput

				// Also add individual fields to root level (but skip "data" to avoid overwriting)
				for k, v := range sourceOutput {
					// Don't overwrite the "data" key we just set above
					if k != "data" {
						evalContext[k] = v
					}
				}
			}

			// Add all node outputs for access via nodeId.field
			for nodeID, output := range nodeOutputs {
				evalContext[nodeID] = output
			}

			// Evaluate the condition
			result, err := gval.Evaluate(condition, evalContext)
			if err != nil {
				log.Printf("  ⚠️  Failed to evaluate edge condition '%s': %v (defaulting to false)", condition, err)
				return false
			}

			// Convert to boolean
			boolResult, ok := result.(bool)
			if !ok {
				log.Printf("  ⚠️  Edge condition '%s' did not evaluate to boolean (got %T), defaulting to false", condition, result)
				return false
			}

			log.Printf("  🔍 Edge condition '%s' evaluated to: %v", condition, boolResult)
			return boolResult
		}
	}

	// No edge found or no condition - default to true
	return true
}

// checkExecutionLimits validates execution hasn't exceeded limits
func (s *WorkflowEngineService) checkExecutionLimits(ctx context.Context, limits *executionLimits) error {
	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		// Distinguish between timeout and cancellation (e.g., server shutdown)
		err := ctx.Err()
		elapsed := time.Since(limits.startTime)

		if err == context.DeadlineExceeded {
			// Actual timeout - elapsed time exceeded the limit
			return fmt.Errorf("workflow execution timeout after %v (limit: %v)", elapsed, MaxExecutionDuration)
		} else if err == context.Canceled {
			// Context was cancelled (e.g., server shutdown, graceful stop)
			// This is not a timeout - return a special error that can be checked upstream
			// to determine if the workflow should remain in "running" state for resumption
			return fmt.Errorf("workflow execution interrupted (context cancelled) after %v - can be resumed: %w", elapsed, err)
		} else {
			// Unknown cancellation reason
			return fmt.Errorf("workflow execution cancelled: %v (after %v): %w", err, elapsed, err)
		}
	default:
	}

	// Check node count
	if limits.nodesExecuted > MaxTotalNodes {
		return fmt.Errorf("workflow exceeded maximum node executions (%d > %d)", limits.nodesExecuted, MaxTotalNodes)
	}

	// Check depth
	if limits.currentDepth > MaxLoopDepth {
		return fmt.Errorf("workflow exceeded maximum nesting depth (%d > %d)", limits.currentDepth, MaxLoopDepth)
	}

	return nil
}

// ExecuteWorkflow executes a workflow (called by River worker)
func (s *WorkflowEngineService) ExecuteWorkflow(ctx context.Context, workflowID, executionID, inputJSON, triggerType string) error {
	log.Printf("🔄 Starting workflow execution: %s (execution: %s)", workflowID, executionID)

	// Get the existing execution record (created before queuing the job)
	// We need this early to calculate remaining time for context timeout
	var execution models.WorkflowExecution
	if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
		return fmt.Errorf("execution record not found: %w", err)
	}

	// Check for checkpoint to determine if we're resuming
	var completedNodes []models.WorkflowNodeExecution
	s.db.Where("execution_id = ? AND status = ?", executionID, "success").
		Find(&completedNodes)

	isResuming := len(completedNodes) > 0

	// When resuming, check if we've already exceeded the time limit
	// This prevents trying to continue when there's no time remaining
	if isResuming {
		elapsed := time.Since(execution.StartedAt)
		if elapsed >= MaxExecutionDuration {
			// Already exceeded limit - fail immediately with clear error
			return fmt.Errorf("workflow execution exceeded maximum duration: elapsed %v >= limit %v", elapsed, MaxExecutionDuration)
		}
		log.Printf("⏱️  Resuming workflow: %v elapsed, %v remaining (limit: %v)", elapsed, MaxExecutionDuration-elapsed, MaxExecutionDuration)
	}

	// Create context with timeout for abuse prevention
	// Calculate timeout based on remaining time when resuming
	var timeoutDuration time.Duration
	if isResuming {
		elapsed := time.Since(execution.StartedAt)
		timeoutDuration = MaxExecutionDuration - elapsed
		// Ensure minimum timeout of 1 second for safety
		if timeoutDuration < 1*time.Second {
			timeoutDuration = 1 * time.Second
		}
	} else {
		timeoutDuration = MaxExecutionDuration
	}

	// Create context with timeout
	// When resuming, use background context to prevent parent cancellation (e.g., shutdown) from interrupting
	// For fresh executions, use parent context to respect shutdown signals
	var baseCtx context.Context
	if isResuming {
		// Use background context for resumed workflows to allow completion even during shutdown
		baseCtx = context.Background()
		log.Printf("⏱️  Using background context for resumed workflow to prevent cancellation propagation")
	} else {
		// Use parent context for fresh workflows
		baseCtx = ctx
	}

	execCtx, cancel := context.WithTimeout(baseCtx, timeoutDuration)
	defer cancel()
	log.Printf("⏱️  Created execution context with timeout: %v", timeoutDuration)

	// Get workflow
	var workflow models.Workflow
	if err := s.db.First(&workflow, "id = ?", workflowID).Error; err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	if !workflow.IsActive {
		return fmt.Errorf("workflow is not active: %s", workflowID)
	}

	// Get the latest version
	var latestVersion models.WorkflowVersion
	if err := s.db.Where("workflow_id = ?", workflowID).
		Order("version DESC").
		First(&latestVersion).Error; err != nil {
		return fmt.Errorf("no version found for workflow: %w", err)
	}

	log.Printf("📖 Using workflow version %d", latestVersion.Version)

	// Parse input
	var input map[string]interface{}
	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return fmt.Errorf("failed to parse input: %w", err)
		}
	}

	// Check input size
	if err := checkDataSize(input, "input"); err != nil {
		return fmt.Errorf("input data too large: %w", err)
	}

	checkpoint := make(map[string]*models.WorkflowNodeExecution)
	for i := range completedNodes {
		checkpoint[completedNodes[i].NodeID] = &completedNodes[i]
	}

	// Count all completed node executions (including loop iterations) for limit tracking
	// This ensures that resuming from a checkpoint accounts for previously executed nodes
	var completedCount int64
	s.db.Model(&models.WorkflowNodeExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "success").
		Count(&completedCount)

	// Determine the actual start time for limit tracking
	// When resuming from a checkpoint, use the original execution start time
	// Otherwise use the current time for fresh executions
	var actualStartTime time.Time
	if isResuming {
		// Resuming: use the original execution start time to properly track total duration
		actualStartTime = execution.StartedAt
		log.Printf("🔄 Resuming workflow execution from checkpoint: %d unique nodes already completed, %d total node executions, original start: %v", len(checkpoint), completedCount, actualStartTime)
	} else {
		// Fresh execution: use current time
		actualStartTime = time.Now()
		log.Printf("🆕 Starting fresh workflow execution")
	}

	// Update status to running when resuming (transition from interrupted/error to running)
	// Clear any previous error messages since we're resuming from checkpoint
	// or keep as running if already running
	if execution.Status != "running" {
		updates := map[string]interface{}{
			"status": "running",
		}
		// Clear error field when resuming since we're starting fresh from checkpoint
		if execution.Error != nil {
			updates["error"] = nil
			log.Printf("🔄 Clearing previous error message when resuming from checkpoint")
		}
		s.db.Model(&execution).Updates(updates)
		log.Printf("🔄 Transitioning workflow execution from '%s' to 'running' for resumption", execution.Status)
	} else if execution.Error != nil {
		// Even if already running, clear stale error messages when resuming
		s.db.Model(&execution).Update("error", nil)
		log.Printf("🔄 Clearing stale error message from running workflow execution")
	}

	// Parse workflow definition
	var definition map[string]interface{}
	if err := json.Unmarshal([]byte(latestVersion.Definition), &definition); err != nil {
		return fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// Initialize execution limits tracker
	// Start with the count of already-executed nodes and original start time to properly track limits on resume
	limits := &executionLimits{
		nodesExecuted: int(completedCount),
		currentDepth:  0,
		startTime:     actualStartTime,
	}

	// Execute workflow with limits and checkpoint
	err := s.executeWorkflowDefinition(execCtx, execution.ID, workflow.AccountID, definition, input, limits, checkpoint)

	// Update execution status
	now := time.Now()
	if err != nil {
		errMsg := err.Error()

		// Check if this is a cancellation (shutdown/interruption) rather than a real error
		// Only context.Canceled should keep the execution in "running" state for resumption
		// context.DeadlineExceeded is a real timeout error and should mark as failed
		isCancellation := false
		if errors.Is(err, context.Canceled) ||
			strings.Contains(errMsg, "interrupted") ||
			strings.Contains(errMsg, "context cancelled") {
			isCancellation = true
		}

		if isCancellation {
			// Mark execution as "interrupted" so it can be clearly identified for resumption
			// Don't set completed_at, so the workflow can be resumed
			s.db.Model(&execution).Updates(map[string]interface{}{
				"status": "interrupted", // Mark as interrupted for resumption
				"error":  errMsg,        // Store error message for debugging
			})
			log.Printf("⏸️  Workflow execution interrupted, marked as 'interrupted' for resumption: %s", executionID)
		} else {
			// Real error - mark as failed
			s.db.Model(&execution).Updates(map[string]interface{}{
				"status":       "error",
				"error":        errMsg,
				"completed_at": now,
			})
		}
		return err
	}

	// Reload execution to check current status
	// The status may have been updated during execution (e.g., to "sleeping")
	if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
		log.Printf("⚠️  Failed to reload execution status: %v", err)
		// Continue with original execution state
	}

	// Check if workflow is in a waiting state (sleeping) - do NOT mark as complete
	if execution.Status == "sleeping" {
		log.Printf("💤 Workflow execution is sleeping, will be resumed later: %s", executionID)
		return nil
	}

	// Check if there are any pending outbox messages
	var pendingCount int64
	s.db.Model(&models.OutboxMessage{}).
		Joins("JOIN workflow_node_executions ON workflow_node_executions.id = outbox_messages.node_execution_id").
		Where("workflow_node_executions.execution_id = ? AND outbox_messages.status IN ?",
			execution.ID, []string{"pending", "processing"}).
		Count(&pendingCount)

	if pendingCount > 0 {
		// Workflow has pending async operations, keep it in running state
		s.db.Model(&execution).Update("status", "running")
		log.Printf("✅ Workflow execution completed with %d pending async operations: %s", pendingCount, workflowID)
	} else {
		// All operations completed
		s.db.Model(&execution).Updates(map[string]interface{}{
			"status":       "success",
			"completed_at": now,
		})
		log.Printf("✅ Workflow execution completed: %s", workflowID)
	}

	return nil
}

// executeWorkflowDefinition executes the workflow definition with proper graph-based execution
// checkpoint contains already-executed nodes for resumption
func (s *WorkflowEngineService) executeWorkflowDefinition(ctx context.Context, executionID string, accountID *string, definition map[string]interface{}, input map[string]interface{}, limits *executionLimits, checkpoint map[string]*models.WorkflowNodeExecution) error {
	nodes, ok := definition["nodes"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid workflow definition: missing nodes")
	}

	edges, ok := definition["edges"].([]interface{})
	if !ok {
		edges = []interface{}{} // Empty edges is okay
	}

	// Check execution limits before starting
	if err := s.checkExecutionLimits(ctx, limits); err != nil {
		return err
	}

	// Build node map and adjacency list
	nodeMap := make(map[string]map[string]interface{})
	adjacencyList := make(map[string][]string) // nodeID -> [targetNodeIDs]
	inDegree := make(map[string]int)           // nodeID -> number of incoming edges

	// Parse nodes
	for _, nodeData := range nodes {
		node, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		nodeID, _ := node["id"].(string)
		nodeMap[nodeID] = node
		inDegree[nodeID] = 0
		adjacencyList[nodeID] = []string{}
	}

	// Parse edges to build adjacency list
	for _, edgeData := range edges {
		edge, ok := edgeData.(map[string]interface{})
		if !ok {
			continue
		}

		source, _ := edge["source"].(string)
		target, _ := edge["target"].(string)

		if source != "" && target != "" {
			adjacencyList[source] = append(adjacencyList[source], target)
			inDegree[target]++
		}
	}

	// Find start node
	var startNodeID string
	for nodeID, node := range nodeMap {
		nodeType, _ := node["type"].(string)
		if nodeType == executors.NodeTypeStart {
			startNodeID = nodeID
			break
		}
	}

	if startNodeID == "" {
		return fmt.Errorf("no start node found in workflow")
	}

	// Store node outputs for passing to next nodes
	nodeOutputs := make(map[string]interface{})
	nodeOutputs[startNodeID] = input // Start node output is the workflow input

	// Execute workflow using BFS/topological traversal
	queue := []string{startNodeID}
	executed := make(map[string]bool)

	for len(queue) > 0 {
		// Check execution limits before each node
		if err := s.checkExecutionLimits(ctx, limits); err != nil {
			return err
		}

		currentNodeID := queue[0]
		queue = queue[1:]

		if executed[currentNodeID] {
			continue
		}

		// CHECK CHECKPOINT: Skip if already executed successfully
		if checkpointNode, exists := checkpoint[currentNodeID]; exists {
			log.Printf("⏭️  Skipping already-executed node: %s (status: %s)", currentNodeID, checkpointNode.Status)

			// Load output from checkpoint
			if checkpointNode.Output != nil {
				var output map[string]interface{}
				if err := json.Unmarshal([]byte(*checkpointNode.Output), &output); err == nil {
					nodeOutputs[currentNodeID] = output
					log.Printf("  📥 Loaded output from checkpoint for node: %s", currentNodeID)
				}
			}

			// Mark as executed and add children to queue (check edge conditions)
			executed[currentNodeID] = true
			for _, nextNodeID := range adjacencyList[currentNodeID] {
				if !executed[nextNodeID] {
					// Check edge conditions even when resuming from checkpoint
					shouldAdd := s.checkEdgeCondition(edges, currentNodeID, nextNodeID, nodeOutputs)
					if shouldAdd {
						queue = append(queue, nextNodeID)
					} else {
						log.Printf("  ⏭️  Skipping node %s (edge condition not satisfied on resume)", nextNodeID)
					}
				}
			}
			continue
		}

		executed[currentNodeID] = true
		currentNode := nodeMap[currentNodeID]
		nodeType, _ := currentNode["type"].(string)

		// Skip start and end nodes for execution
		if !executors.IsSkippableNode(nodeType) {
			// Increment node execution counter
			limits.nodesExecuted++
			// Get node config
			data, _ := currentNode["data"].(map[string]interface{})
			config, _ := data["config"].(map[string]interface{})

			// Get input from previous node (source of the first incoming edge)
			nodeInput := input // Default to workflow input
			for sourceNodeID := range nodeMap {
				for _, targetID := range adjacencyList[sourceNodeID] {
					if targetID == currentNodeID {
						if output, ok := nodeOutputs[sourceNodeID]; ok {
							nodeInput = output.(map[string]interface{})
						}
						break
					}
				}
			}

			// Create workflowData with all node outputs
			workflowData := map[string]interface{}{
				"nodeOutputs": nodeOutputs,
				"input":       input,
			}

			// Check if this is a loop node
			if nodeType == "loop" {
				// Increment depth for nested loop tracking
				limits.currentDepth++
				defer func() { limits.currentDepth-- }()

				// Execute loop and its child nodes iteratively
				err := s.executeLoopWithChildren(ctx, executionID, accountID, currentNodeID, config, nodeInput, workflowData, nodeMap, adjacencyList, edges, executed, nodeOutputs, limits)
				if err != nil {
					return fmt.Errorf("loop execution failed (%s): %w", currentNodeID, err)
				}
				// Skip adding next nodes to queue here - they're already executed in the loop
				continue
			} else if nodeType == "loop-accumulator" {
				// Increment depth for nested loop tracking
				limits.currentDepth++
				defer func() { limits.currentDepth-- }()

				// Execute loop accumulator with feedback loop
				err := s.executeLoopAccumulatorWithChildren(ctx, executionID, accountID, currentNodeID, config, nodeInput, workflowData, nodeMap, adjacencyList, edges, executed, nodeOutputs, limits)
				if err != nil {
					return fmt.Errorf("loop accumulator execution failed (%s): %w", currentNodeID, err)
				}
				// Add nodes connected to the "output" handle (Final Output) to the queue
				log.Printf("  🔍 Searching for final output edges from loop accumulator %s", currentNodeID)
				log.Printf("  🔍 Total edges in workflow: %d", len(edges))

				foundFinalOutputEdge := false
				for _, edgeData := range edges {
					edge, ok := edgeData.(map[string]interface{})
					if !ok {
						continue
					}

					source, _ := edge["source"].(string)
					sourceHandle, _ := edge["sourceHandle"].(string)
					target, _ := edge["target"].(string)

					// Log all edges from this loop accumulator node for debugging
					if source == currentNodeID {
						log.Printf("  🔍 Edge from loop accumulator: sourceHandle='%s', target='%s', executed=%v", sourceHandle, target, executed[target])
					}

					// Only add nodes connected to the final "output" handle
					if source == currentNodeID && sourceHandle == "output" && !executed[target] {
						queue = append(queue, target)
						log.Printf("  📤 Adding final output node to queue: %s", target)
						foundFinalOutputEdge = true
					}
				}

				if !foundFinalOutputEdge {
					log.Printf("  ⚠️  No final output edges found from loop accumulator %s", currentNodeID)
				}

				// Continue to skip normal adjacency list processing
				continue
			} else {
				// Execute node normally
				output, err := s.executeNodeAndGetOutput(ctx, executionID, accountID, currentNodeID, nodeType, config, nodeInput, workflowData)
				if err != nil {
					return fmt.Errorf("node execution failed (%s): %w", currentNodeID, err)
				}

				// Store output for next nodes
				nodeOutputs[currentNodeID] = output
			}
		}

		// Add next nodes to queue (for non-loop nodes that didn't use continue)
		// IMPORTANT: This must happen BEFORE checking for sleeping state to ensure
		// that when workflow resumes, next nodes are properly queued
		// Check edge conditions before adding nodes to queue
		for _, nextNodeID := range adjacencyList[currentNodeID] {
			if !executed[nextNodeID] {
				// Check if there's an edge condition that needs to be satisfied
				shouldAdd := s.checkEdgeCondition(edges, currentNodeID, nextNodeID, nodeOutputs)
				if shouldAdd {
					queue = append(queue, nextNodeID)
				} else {
					log.Printf("  ⏭️  Skipping node %s (edge condition not satisfied)", nextNodeID)
				}
			}
		}

		// CRITICAL: Check if workflow entered a terminal state (sleeping, completed, failed)
		// This prevents continuing execution when workflow should be paused
		// This check happens AFTER adding next nodes to queue so resumption works correctly
		if !executors.IsSkippableNode(nodeType) {
			var currentExecution models.WorkflowExecution
			if err := s.db.First(&currentExecution, "id = ?", executionID).Error; err == nil {
				if currentExecution.Status == "sleeping" {
					log.Printf("  💤 Workflow entered sleeping state after node %s - stopping execution", currentNodeID)
					log.Printf("  📋 Next nodes already queued: %v", adjacencyList[currentNodeID])
					return nil // Stop execution gracefully
				}
			}
		}
	}

	return nil
}

// executeNodeAndGetOutput executes a node and returns its output
func (s *WorkflowEngineService) executeNodeAndGetOutput(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}) (map[string]interface{}, error) {
	log.Printf("  ▶ Executing node %s (type: %s)", nodeID, nodeType)

	// Check if node requires outbox pattern (side effects)
	if executors.NodeRequiresOutbox(nodeType) {
		// For outbox nodes, we don't get immediate output
		// They execute asynchronously
		err := s.executeNodeWithOutbox(ctx, executionID, accountID, nodeID, nodeType, config, input, workflowData)
		if err != nil {
			return nil, err
		}
		// Return empty output for async nodes
		return map[string]interface{}{
			"status": "queued",
			"nodeId": nodeID,
		}, nil
	}

	// Execute synchronously and get output
	return s.executeSynchronousNodeWithOutput(ctx, executionID, accountID, nodeID, nodeType, config, input, workflowData)
}

// executeNode executes a single node (legacy method for compatibility)
func (s *WorkflowEngineService) executeNode(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}) error {
	log.Printf("  ▶ Executing node %s (type: %s)", nodeID, nodeType)

	// Check if node requires outbox pattern (side effects)
	if executors.NodeRequiresOutbox(nodeType) {
		return s.executeNodeWithOutbox(ctx, executionID, accountID, nodeID, nodeType, config, input, workflowData)
	}

	// Execute synchronously (no side effects)
	return s.executeSynchronousNode(ctx, executionID, accountID, nodeID, nodeType, config, input, workflowData)
}

// executeNodeWithOutbox executes a node with side effects using the outbox pattern
func (s *WorkflowEngineService) executeNodeWithOutbox(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}) error {
	// Determine event type
	eventType := fmt.Sprintf("%s.%s", nodeType, "send")
	if nodeType == "http" {
		eventType = "http.request"
	}

	// Create node execution and outbox message atomically
	nodeExecution, outboxMessage, err := s.outboxService.ExecuteNodeWithOutbox(
		ctx, executionID, accountID, nodeID, nodeType, config, input, eventType,
	)

	if err != nil {
		return fmt.Errorf("failed to create outbox entry: %w", err)
	}

	log.Printf("  📬 Node %s queued for outbox processing (message: %s)", nodeID, outboxMessage.ID)
	log.Printf("  ✅ Node execution created: %s (will be processed asynchronously)", nodeExecution.ID)
	return nil
}

// executeSynchronousNode executes a node without side effects synchronously
func (s *WorkflowEngineService) executeSynchronousNode(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}) error {
	// Create node execution record
	nodeExecution := models.WorkflowNodeExecution{
		ExecutionID: executionID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Status:      "running",
	}

	inputJSON, _ := json.Marshal(input)
	inputStr := string(inputJSON)
	nodeExecution.Input = &inputStr

	if err := s.db.Create(&nodeExecution).Error; err != nil {
		return fmt.Errorf("failed to create node execution: %w", err)
	}

	// Get executor
	executor, err := s.executorFactory.GetExecutor(nodeType)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return err
	}

	// Execute node
	accountIDStr := ""
	if accountID != nil {
		accountIDStr = *accountID
	}
	execCtx := executors.ExecutionContext{
		NodeID:       nodeID,
		NodeConfig:   config,
		Input:        input,
		WorkflowData: workflowData,
		ExecutionID:  executionID,
		AccountID:    accountIDStr,
	}

	result, err := executor.Execute(ctx, execCtx)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return err
	}

	if !result.Success {
		// Update node execution with error
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        result.Error,
			"completed_at": now,
		})
		return fmt.Errorf("node execution failed: %s", result.Error)
	}

	// Check if node needs to sleep
	if result.NeedsSleep && result.WakeUpAt != nil {
		log.Printf("  💤 Node %s requires sleep until %s", nodeID, result.WakeUpAt.Format(time.RFC3339))

		// Mark node execution as success (it completed successfully, just needs to sleep)
		outputJSON, _ := json.Marshal(result.Output)
		outputStr := string(outputJSON)
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "success",
			"output":       outputStr,
			"completed_at": now,
		})

		// Mark workflow execution as sleeping
		var execution models.WorkflowExecution
		if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
			log.Printf("  ❌ Failed to find execution for sleep: %v", err)
			return fmt.Errorf("failed to find execution: %w", err)
		}

		if err := s.db.Model(&execution).Update("status", "sleeping").Error; err != nil {
			log.Printf("  ❌ Failed to mark execution as sleeping: %v", err)
			return fmt.Errorf("failed to update execution status: %w", err)
		}

		// Schedule wake-up if scheduler service is available
		if s.schedulerService != nil {
			var workflow models.Workflow
			if err := s.db.First(&workflow, "id = ?", execution.WorkflowID).Error; err != nil {
				log.Printf("  ❌ Failed to find workflow for sleep: %v", err)
				// Rollback sleeping status
				s.db.Model(&execution).Update("status", "running")
				return fmt.Errorf("failed to find workflow: %w", err)
			}

			if err := s.schedulerService.ScheduleSleepWakeUp(executionID, workflow.ID, nodeID, *result.WakeUpAt); err != nil {
				log.Printf("  ❌ Failed to schedule sleep wake-up: %v", err)
				// Rollback sleeping status
				s.db.Model(&execution).Update("status", "running")
				return fmt.Errorf("failed to schedule sleep wake-up: %w", err)
			}

			log.Printf("  ✅ Workflow execution %s is now sleeping until %s", executionID, result.WakeUpAt.Format(time.RFC3339))
		} else {
			log.Printf("  ⚠️  Scheduler service not available, cannot schedule sleep wake-up")
			// Rollback sleeping status
			s.db.Model(&execution).Update("status", "running")
			return fmt.Errorf("scheduler service not available for sleep node")
		}

		// Return nil - caller will check execution status and stop workflow
		return nil
	}

	// Update node execution with success
	outputJSON, _ := json.Marshal(result.Output)
	outputStr := string(outputJSON)
	now := time.Now()
	s.db.Model(&nodeExecution).Updates(map[string]interface{}{
		"status":       "success",
		"output":       outputStr,
		"completed_at": now,
	})

	log.Printf("  ✅ Node completed: %s", nodeID)
	return nil
}

// executeNodeInLoop executes a node within a loop context and stores parent loop ID
func (s *WorkflowEngineService) executeNodeInLoop(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}, parentLoopNodeID string) (map[string]interface{}, error) {
	// Create node execution record with parent loop context
	nodeExecution := models.WorkflowNodeExecution{
		ExecutionID:      executionID,
		NodeID:           nodeID,
		NodeType:         nodeType,
		Status:           "running",
		ParentLoopNodeID: &parentLoopNodeID, // Mark this as a loop body execution
	}

	inputJSON, _ := json.Marshal(input)
	inputStr := string(inputJSON)
	nodeExecution.Input = &inputStr

	if err := s.db.Create(&nodeExecution).Error; err != nil {
		return nil, fmt.Errorf("failed to create node execution: %w", err)
	}

	// Get executor
	executor, err := s.executorFactory.GetExecutor(nodeType)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return nil, err
	}

	// Execute node
	accountIDStr := ""
	if accountID != nil {
		accountIDStr = *accountID
	}
	execCtx := executors.ExecutionContext{
		NodeID:       nodeID,
		NodeConfig:   config,
		Input:        input,
		WorkflowData: workflowData,
		ExecutionID:  executionID,
		AccountID:    accountIDStr,
	}

	result, err := executor.Execute(ctx, execCtx)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return nil, err
	}

	if !result.Success {
		// Update node execution with error
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        result.Error,
			"completed_at": now,
		})
		return nil, fmt.Errorf("node execution unsuccessful: %s", result.Error)
	}

	// Update node execution with success
	outputJSON, _ := json.Marshal(result.Output)
	outputStr := string(outputJSON)
	now := time.Now()
	s.db.Model(&nodeExecution).Updates(map[string]interface{}{
		"status":       "success",
		"output":       outputStr,
		"completed_at": now,
	})

	return result.Output, nil
}

// executeSynchronousNodeWithOutput executes a node and returns its output
func (s *WorkflowEngineService) executeSynchronousNodeWithOutput(ctx context.Context, executionID string, accountID *string, nodeID, nodeType string, config, input, workflowData map[string]interface{}) (map[string]interface{}, error) {
	// Create node execution record
	nodeExecution := models.WorkflowNodeExecution{
		ExecutionID: executionID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Status:      "running",
	}

	inputJSON, _ := json.Marshal(input)
	inputStr := string(inputJSON)
	nodeExecution.Input = &inputStr

	if err := s.db.Create(&nodeExecution).Error; err != nil {
		return nil, fmt.Errorf("failed to create node execution: %w", err)
	}

	// Get executor
	executor, err := s.executorFactory.GetExecutor(nodeType)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return nil, err
	}

	// Execute node
	accountIDStr := ""
	if accountID != nil {
		accountIDStr = *accountID
	}
	execCtx := executors.ExecutionContext{
		NodeID:       nodeID,
		NodeConfig:   config,
		Input:        input,
		WorkflowData: workflowData,
		ExecutionID:  executionID,
		AccountID:    accountIDStr,
	}

	result, err := executor.Execute(ctx, execCtx)
	if err != nil {
		// Update node execution with error
		errMsg := err.Error()
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return nil, err
	}

	if !result.Success {
		// Update node execution with error
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        result.Error,
			"completed_at": now,
		})
		return nil, fmt.Errorf("node execution failed: %s", result.Error)
	}

	// Check if node needs to sleep
	if result.NeedsSleep && result.WakeUpAt != nil {
		log.Printf("  💤 Node %s requires sleep until %s", nodeID, result.WakeUpAt.Format(time.RFC3339))

		// Mark node execution as success (it completed successfully, just needs to sleep)
		outputJSON, _ := json.Marshal(result.Output)
		outputStr := string(outputJSON)
		now := time.Now()
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "success",
			"output":       outputStr,
			"completed_at": now,
		})

		// Mark workflow execution as sleeping
		var execution models.WorkflowExecution
		if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
			log.Printf("  ❌ Failed to find execution for sleep: %v", err)
			return nil, fmt.Errorf("failed to find execution: %w", err)
		}

		if err := s.db.Model(&execution).Update("status", "sleeping").Error; err != nil {
			log.Printf("  ❌ Failed to mark execution as sleeping: %v", err)
			return nil, fmt.Errorf("failed to update execution status: %w", err)
		}

		// Schedule wake-up if scheduler service is available
		if s.schedulerService != nil {
			var workflow models.Workflow
			if err := s.db.First(&workflow, "id = ?", execution.WorkflowID).Error; err != nil {
				log.Printf("  ❌ Failed to find workflow for sleep: %v", err)
				// Rollback sleeping status
				s.db.Model(&execution).Update("status", "running")
				return nil, fmt.Errorf("failed to find workflow: %w", err)
			}

			if err := s.schedulerService.ScheduleSleepWakeUp(executionID, workflow.ID, nodeID, *result.WakeUpAt); err != nil {
				log.Printf("  ❌ Failed to schedule sleep wake-up: %v", err)
				// Rollback sleeping status
				s.db.Model(&execution).Update("status", "running")
				return nil, fmt.Errorf("failed to schedule sleep wake-up: %w", err)
			}

			log.Printf("  ✅ Workflow execution %s is now sleeping until %s", executionID, result.WakeUpAt.Format(time.RFC3339))
		} else {
			log.Printf("  ⚠️  Scheduler service not available, cannot schedule sleep wake-up")
			// Rollback sleeping status
			s.db.Model(&execution).Update("status", "running")
			return nil, fmt.Errorf("scheduler service not available for sleep node")
		}

		// Return the sleep output (BFS loop will check execution status and stop)
		return result.Output, nil
	}

	// Update node execution with success
	outputJSON, _ := json.Marshal(result.Output)
	outputStr := string(outputJSON)
	now := time.Now()
	s.db.Model(&nodeExecution).Updates(map[string]interface{}{
		"status":       "success",
		"output":       outputStr,
		"completed_at": now,
	})

	log.Printf("  ✅ Node completed: %s", nodeID)

	// Return the output for the next node
	return result.Output, nil
}

// executeLoopWithChildren executes a loop node and iteratively executes its child nodes
func (s *WorkflowEngineService) executeLoopWithChildren(
	ctx context.Context,
	executionID string,
	accountID *string,
	loopNodeID string,
	loopConfig map[string]interface{},
	loopInput map[string]interface{},
	workflowData map[string]interface{},
	nodeMap map[string]map[string]interface{},
	adjacencyList map[string][]string,
	edges []interface{},
	executed map[string]bool,
	nodeOutputs map[string]interface{},
	limits *executionLimits,
) error {
	log.Printf("  🔄 Executing loop node %s", loopNodeID)

	// Check execution limits
	if err := s.checkExecutionLimits(ctx, limits); err != nil {
		return err
	}

	// First, execute the loop node itself to get the items array
	loopOutput, err := s.executeNodeAndGetOutput(ctx, executionID, accountID, loopNodeID, "loop", loopConfig, loopInput, workflowData)
	if err != nil {
		return fmt.Errorf("loop node execution failed: %w", err)
	}

	// Extract items/results from loop output
	results, ok := loopOutput["results"].([]interface{})
	if !ok {
		log.Printf("  ⚠️  Loop node did not return results array, skipping iteration")
		nodeOutputs[loopNodeID] = loopOutput
		return nil
	}

	iterationCount := len(results)
	log.Printf("  🔄 Loop will iterate %d times", iterationCount)

	// Get iteration delay from config (default: 0ms = no delay)
	iterationDelay := 0
	if delay, ok := loopConfig["iterationDelay"].(float64); ok {
		iterationDelay = int(delay)
	}
	if iterationDelay > 0 {
		log.Printf("  ⏱️  Iteration delay: %dms", iterationDelay)
	}

	// Find all child nodes (nodes that are directly connected after the loop)
	childNodeIDs := adjacencyList[loopNodeID]
	if len(childNodeIDs) == 0 {
		log.Printf("  ⚠️  Loop node has no child nodes")
		nodeOutputs[loopNodeID] = loopOutput
		return nil
	}

	// For each iteration
	for i, resultData := range results {
		log.Printf("  🔄 Loop iteration %d/%d", i+1, iterationCount)

		// Each result should be a map with index and item
		iterationInput, ok := resultData.(map[string]interface{})
		if !ok {
			log.Printf("  ⚠️  Iteration %d: result is not an object, skipping", i)
			continue
		}

		// Execute child nodes for this iteration
		// We need to execute the subgraph starting from child nodes
		for _, childNodeID := range childNodeIDs {
			err := s.executeSubgraph(ctx, executionID, accountID, childNodeID, iterationInput, workflowData, nodeMap, adjacencyList, edges, executed, limits)
			if err != nil {
				log.Printf("  ❌ Loop iteration %d failed at node %s: %v", i, childNodeID, err)
				// Continue with next iteration even if this one fails
			}
		}

		// Delay before next iteration (if configured and not the last iteration)
		if iterationDelay > 0 && i < iterationCount-1 {
			log.Printf("  ⏱️  Waiting %dms before next iteration...", iterationDelay)
			time.Sleep(time.Duration(iterationDelay) * time.Millisecond)
		}
	}

	log.Printf("  ✅ Loop completed %d iterations", iterationCount)

	// Mark all child nodes and their descendants as executed
	s.markSubgraphAsExecuted(loopNodeID, adjacencyList, executed)

	// Store loop output for any nodes after the loop subgraph
	nodeOutputs[loopNodeID] = loopOutput

	return nil
}

// executeSubgraph executes a node and all its descendants (for loop iterations)
func (s *WorkflowEngineService) executeSubgraph(
	ctx context.Context,
	executionID string,
	accountID *string,
	startNodeID string,
	input map[string]interface{},
	workflowData map[string]interface{},
	nodeMap map[string]map[string]interface{},
	adjacencyList map[string][]string,
	edges []interface{},
	executedGlobal map[string]bool,
	limits *executionLimits,
) error {
	// Track what we execute in this subgraph iteration
	queue := []string{startNodeID}
	currentOutput := input

	for len(queue) > 0 {
		// Check execution limits
		if err := s.checkExecutionLimits(ctx, limits); err != nil {
			return err
		}

		nodeID := queue[0]
		queue = queue[1:]

		node, exists := nodeMap[nodeID]
		if !exists {
			continue
		}

		nodeType, _ := node["type"].(string)

		// Skip start and end nodes in subgraph
		if executors.IsSkippableNode(nodeType) {
			// Add children to queue but don't execute
			for _, nextNodeID := range adjacencyList[nodeID] {
				queue = append(queue, nextNodeID)
			}
			continue
		}

		// Handle nested loops - track depth and execute
		if nodeType == "loop" {
			data, _ := node["data"].(map[string]interface{})
			config, _ := data["config"].(map[string]interface{})

			log.Printf("    🔄 Executing nested loop node %s (current depth: %d)", nodeID, limits.currentDepth)

			// Increment depth for nested loop
			limits.currentDepth++

			// Check depth limit before executing
			if err := s.checkExecutionLimits(ctx, limits); err != nil {
				limits.currentDepth-- // Restore depth before returning error
				return fmt.Errorf("nested loop depth limit exceeded at node %s: %w", nodeID, err)
			}

			err := s.executeLoopWithChildren(ctx, executionID, accountID, nodeID, config, currentOutput, workflowData, nodeMap, adjacencyList, edges, executedGlobal, workflowData["nodeOutputs"].(map[string]interface{}), limits)

			// Decrement depth after execution
			limits.currentDepth--

			if err != nil {
				return fmt.Errorf("nested loop execution failed: %w", err)
			}
			// Loop handles its own children, don't add them to queue
			continue
		}

		if nodeType == "loop-accumulator" {
			data, _ := node["data"].(map[string]interface{})
			config, _ := data["config"].(map[string]interface{})

			log.Printf("    🔄 Executing nested loop accumulator node %s (current depth: %d)", nodeID, limits.currentDepth)

			// Increment depth for nested loop accumulator
			limits.currentDepth++

			// Check depth limit before executing
			if err := s.checkExecutionLimits(ctx, limits); err != nil {
				limits.currentDepth-- // Restore depth before returning error
				return fmt.Errorf("nested loop accumulator depth limit exceeded at node %s: %w", nodeID, err)
			}

			// Execute loop accumulator with edges
			err := s.executeLoopAccumulatorWithChildren(ctx, executionID, accountID, nodeID, config, currentOutput, workflowData, nodeMap, adjacencyList, edges, executedGlobal, workflowData["nodeOutputs"].(map[string]interface{}), limits)

			// Decrement depth after execution
			limits.currentDepth--

			if err != nil {
				return fmt.Errorf("nested loop accumulator execution failed: %w", err)
			}
			// Loop accumulator handles its own children, don't add them to queue
			continue
		}

		// Get node config
		data, _ := node["data"].(map[string]interface{})
		config, _ := data["config"].(map[string]interface{})

		// Execute this node
		log.Printf("    ▶ Executing child node %s (type: %s)", nodeID, nodeType)
		output, err := s.executeNodeAndGetOutput(ctx, executionID, accountID, nodeID, nodeType, config, currentOutput, workflowData)
		if err != nil {
			return fmt.Errorf("subgraph node %s execution failed: %w", nodeID, err)
		}

		// Update current output for next node in chain
		currentOutput = output

		// Add child nodes to queue (check edge conditions in subgraph)
		// Note: In loop subgraphs, we typically don't have conditional edges,
		// but we should still check for completeness
		for _, nextNodeID := range adjacencyList[nodeID] {
			// For subgraphs, we use a simple nodeOutputs map
			subgraphNodeOutputs := map[string]interface{}{
				nodeID: currentOutput,
			}
			shouldAdd := s.checkEdgeCondition(edges, nodeID, nextNodeID, subgraphNodeOutputs)
			if shouldAdd {
				queue = append(queue, nextNodeID)
			}
		}
	}

	return nil
}

// markSubgraphAsExecuted marks all nodes in a subgraph as executed
func (s *WorkflowEngineService) markSubgraphAsExecuted(
	startNodeID string,
	adjacencyList map[string][]string,
	executed map[string]bool,
) {
	s.markSubgraphAsExecutedWithStop(startNodeID, "", adjacencyList, executed)
}

// markSubgraphAsExecutedWithStop marks all nodes in a subgraph as executed, stopping at stopNodeID
func (s *WorkflowEngineService) markSubgraphAsExecutedWithStop(
	startNodeID string,
	stopNodeID string,
	adjacencyList map[string][]string,
	executed map[string]bool,
) {
	queue := []string{startNodeID}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		if visited[nodeID] {
			continue
		}

		// Stop if we reach the stop node
		if stopNodeID != "" && nodeID == stopNodeID {
			log.Printf("    🛑 Stopping mark as executed at node %s", nodeID)
			continue
		}

		visited[nodeID] = true
		executed[nodeID] = true

		// Add children to queue
		for _, childID := range adjacencyList[nodeID] {
			if !visited[childID] {
				// Don't follow edges to the stop node
				if stopNodeID != "" && childID == stopNodeID {
					continue
				}
				queue = append(queue, childID)
			}
		}
	}
}

// executeLoopAccumulatorWithChildren executes a loop accumulator node with feedback loop
func (s *WorkflowEngineService) executeLoopAccumulatorWithChildren(
	ctx context.Context,
	executionID string,
	accountID *string,
	loopNodeID string,
	loopConfig map[string]interface{},
	loopInput map[string]interface{},
	workflowData map[string]interface{},
	nodeMap map[string]map[string]interface{},
	adjacencyList map[string][]string,
	edges []interface{},
	executed map[string]bool,
	nodeOutputs map[string]interface{},
	limits *executionLimits,
) error {
	log.Printf("  🔄 Executing loop accumulator node %s", loopNodeID)

	// Check execution limits
	if err := s.checkExecutionLimits(ctx, limits); err != nil {
		return err
	}

	// Create node execution record at the START
	nodeExecution := models.WorkflowNodeExecution{
		ExecutionID: executionID,
		NodeID:      loopNodeID,
		NodeType:    "loop-accumulator",
		Status:      "running",
	}

	inputJSON, _ := json.Marshal(loopInput)
	inputStr := string(inputJSON)
	nodeExecution.Input = &inputStr

	if err := s.db.Create(&nodeExecution).Error; err != nil {
		return fmt.Errorf("failed to create loop accumulator node execution: %w", err)
	}

	log.Printf("  📝 Created loop accumulator node execution record: %s", nodeExecution.ID)

	// Get the executor to prepare iteration data (but don't create node execution)
	executor, err := s.executorFactory.GetExecutor("loop-accumulator")
	if err != nil {
		return fmt.Errorf("failed to get loop accumulator executor: %w", err)
	}

	// Execute to get the iteration data
	accountIDStr := ""
	if accountID != nil {
		accountIDStr = *accountID
	}
	execCtx := executors.ExecutionContext{
		NodeID:       loopNodeID,
		NodeConfig:   loopConfig,
		Input:        loopInput,
		WorkflowData: workflowData,
		ExecutionID:  executionID,
		AccountID:    accountIDStr,
	}

	result, err := executor.Execute(ctx, execCtx)
	if err != nil || !result.Success {
		// Update node execution with error
		now := time.Now()
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = result.Error
		}
		s.db.Model(&nodeExecution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return fmt.Errorf("loop accumulator preparation failed: %w", err)
	}

	loopOutput := result.Output

	// Extract items/results from loop output
	results, ok := loopOutput["results"].([]interface{})
	if !ok {
		log.Printf("  ⚠️  Loop accumulator node did not return results array, skipping iteration")
		nodeOutputs[loopNodeID] = loopOutput
		return nil
	}

	iterationCount := len(results)
	log.Printf("  🔄 Loop accumulator will iterate %d times", iterationCount)

	// Get accumulation mode
	accumulationMode, _ := loopOutput["accumulationMode"].(string)
	if accumulationMode == "" {
		accumulationMode = "array"
	}
	log.Printf("  📊 Accumulation mode: %s", accumulationMode)

	// Get accumulator variable name
	accumulatorVariable, _ := loopOutput["accumulatorVariable"].(string)
	if accumulatorVariable == "" {
		accumulatorVariable = "accumulated"
	}

	// Get error handling mode (default: "skip")
	errorHandling, _ := loopConfig["errorHandling"].(string)
	if errorHandling == "" {
		errorHandling = "skip"
	}
	log.Printf("  🛡️  Error handling: %s", errorHandling)

	// Get iteration delay from config (default: 0ms = no delay)
	iterationDelay := 0
	if delay, ok := loopConfig["iterationDelay"].(float64); ok {
		iterationDelay = int(delay)
	}
	if iterationDelay > 0 {
		log.Printf("  ⏱️  Iteration delay: %dms", iterationDelay)
	}

	// Find loop body nodes - nodes connected to the "loop-output" handle (left side)
	// We need to identify which edges come from the loop-output handle
	var loopBodyNodeIDs []string

	log.Printf("  🔍 Searching for loop body nodes connected to loop-output handle")
	// Parse edges to find which ones originate from loop-output handle
	for _, edgeData := range edges {
		edge, ok := edgeData.(map[string]interface{})
		if !ok {
			continue
		}

		source, _ := edge["source"].(string)
		sourceHandle, _ := edge["sourceHandle"].(string)
		target, _ := edge["target"].(string)

		// Check if this edge comes from our loop node's loop-output handle
		if source == loopNodeID && sourceHandle == "loop-output" {
			loopBodyNodeIDs = append(loopBodyNodeIDs, target)
			log.Printf("  ✅ Added loop body node: %s", target)
		}
	}

	if len(loopBodyNodeIDs) == 0 {
		log.Printf("  ⚠️  Loop accumulator node has no loop body nodes (no connections from loop-output handle)")

		// Still output the results even with no body nodes
		finalOutput := map[string]interface{}{
			"iteration_count":   iterationCount,
			accumulatorVariable: []interface{}{},
		}
		nodeOutputs[loopNodeID] = finalOutput
		return nil
	}

	log.Printf("  🔄 Loop body nodes: %v", loopBodyNodeIDs)

	// Initialize accumulator based on mode
	var accumulated interface{}
	if accumulationMode == "array" {
		accumulated = []interface{}{}
	} else {
		accumulated = nil
	}

	// For each iteration
	for i, resultData := range results {
		log.Printf("  🔄 Loop accumulator iteration %d/%d", i+1, iterationCount)

		// Each result should be a map with index, item, and accumulated
		iterationInput, ok := resultData.(map[string]interface{})
		if !ok {
			log.Printf("  ⚠️  Iteration %d: result is not an object, skipping", i)
			continue
		}

		// Add current accumulated value to the iteration input
		iterationInput[accumulatorVariable] = accumulated

		// Execute loop body nodes for this iteration
		var iterationOutput map[string]interface{}
		iterationFailed := false
		for _, bodyNodeID := range loopBodyNodeIDs {
			output, err := s.executeSubgraphAndGetOutputWithParent(ctx, executionID, accountID, bodyNodeID, loopNodeID, iterationInput, workflowData, nodeMap, adjacencyList, edges, limits)
			if err != nil {
				log.Printf("  ❌ Loop accumulator iteration %d failed at node %s: %v", i, bodyNodeID, err)
				iterationFailed = true

				// Check error handling mode
				if errorHandling == "fail" {
					// Fail the entire loop
					return fmt.Errorf("loop accumulator failed at iteration %d: %w", i, err)
				}
				// Otherwise skip this iteration
				break
			}
			iterationOutput = output
		}

		// Skip accumulation if iteration failed or returned nil/null/undefined
		if iterationFailed || iterationOutput == nil {
			log.Printf("  ⚠️  Skipping iteration %d (failed or null result)", i)
			// Continue to next iteration
		} else {
			// Extract the value to accumulate
			// By default, unwrap "data" key if it exists (most executors wrap output in "data")
			// Users can set "unwrapData" to false in config to keep the full output
			var valueToAccumulate interface{} = iterationOutput
			unwrapData := true // Default to true
			if unwrap, ok := loopConfig["unwrapData"].(bool); ok {
				unwrapData = unwrap
			}

			if unwrapData {
				if dataValue, hasData := iterationOutput["data"]; hasData {
					valueToAccumulate = dataValue
				}
			}

			// Accumulate the result based on mode
			if accumulationMode == "array" {
				// Add to array
				if accArray, ok := accumulated.([]interface{}); ok {
					accumulated = append(accArray, valueToAccumulate)

					// Check accumulated data size to prevent memory abuse
					if err := checkDataSize(accumulated, "accumulated data"); err != nil {
						return fmt.Errorf("accumulator size limit exceeded at iteration %d: %w", i, err)
					}
				}
			} else if accumulationMode == "last" {
				// Replace with latest
				accumulated = valueToAccumulate

				// Check output size
				if err := checkDataSize(accumulated, "accumulated data"); err != nil {
					return fmt.Errorf("accumulator size limit exceeded at iteration %d: %w", i, err)
				}
			}
		}

		// Delay before next iteration (if configured and not the last iteration)
		if iterationDelay > 0 && i < iterationCount-1 {
			log.Printf("  ⏱️  Waiting %dms before next iteration...", iterationDelay)
			time.Sleep(time.Duration(iterationDelay) * time.Millisecond)
		}
	}

	log.Printf("  ✅ Loop accumulator completed %d iterations", iterationCount)

	// Mark only the loop body nodes as executed (not nodes connected to final output)
	// We need to mark the loop body subgraph, but NOT nodes connected to the "output" handle
	// Stop marking at the loop node to prevent marking nodes after the loop
	for _, bodyNodeID := range loopBodyNodeIDs {
		s.markSubgraphAsExecutedWithStop(bodyNodeID, loopNodeID, adjacencyList, executed)
	}

	// Also mark the loop accumulator node itself as executed
	executed[loopNodeID] = true

	// Output final accumulated results
	finalOutput := map[string]interface{}{
		"iteration_count":   iterationCount,
		accumulatorVariable: accumulated,
	}
	nodeOutputs[loopNodeID] = finalOutput

	log.Printf("  📤 Loop accumulator final output: %v", finalOutput)

	// NOW mark the loop accumulator node execution as complete
	outputJSON, _ := json.Marshal(finalOutput)
	outputStr := string(outputJSON)
	now := time.Now()
	s.db.Model(&nodeExecution).Updates(map[string]interface{}{
		"status":       "success",
		"output":       outputStr,
		"completed_at": now,
	})

	log.Printf("  ✅ Loop accumulator node execution marked complete: %s", nodeExecution.ID)

	return nil
}

// executeSubgraphAndGetOutput executes a subgraph and returns the final output
func (s *WorkflowEngineService) executeSubgraphAndGetOutput(
	ctx context.Context,
	executionID string,
	accountID *string,
	startNodeID string,
	input map[string]interface{},
	workflowData map[string]interface{},
	nodeMap map[string]map[string]interface{},
	adjacencyList map[string][]string,
	edges []interface{},
	limits *executionLimits,
) (map[string]interface{}, error) {
	return s.executeSubgraphAndGetOutputWithParent(ctx, executionID, accountID, startNodeID, "", input, workflowData, nodeMap, adjacencyList, edges, limits)
}

func (s *WorkflowEngineService) executeSubgraphAndGetOutputWithParent(
	ctx context.Context,
	executionID string,
	accountID *string,
	startNodeID string,
	parentLoopNodeID string,
	input map[string]interface{},
	workflowData map[string]interface{},
	nodeMap map[string]map[string]interface{},
	adjacencyList map[string][]string,
	edges []interface{},
	limits *executionLimits,
) (map[string]interface{}, error) {
	queue := []string{startNodeID}
	currentOutput := input

	for len(queue) > 0 {
		// Check execution limits
		if err := s.checkExecutionLimits(ctx, limits); err != nil {
			return nil, err
		}

		nodeID := queue[0]
		queue = queue[1:]

		node, exists := nodeMap[nodeID]
		if !exists {
			continue
		}

		nodeType, _ := node["type"].(string)

		// Skip start and end nodes in subgraph
		if executors.IsSkippableNode(nodeType) {
			// Add children to queue but don't execute
			for _, nextNodeID := range adjacencyList[nodeID] {
				queue = append(queue, nextNodeID)
			}
			continue
		}

		// Check if this is the parent loop node (feedback edge)
		if (nodeType == "loop" || nodeType == "loop-accumulator") && parentLoopNodeID != "" && nodeID == parentLoopNodeID {
			log.Printf("    🔙 Reached parent loop node %s, stopping traversal", nodeID)
			continue
		}

		// Handle nested loops in loop bodies - track depth and execute
		if nodeType == "loop" {
			data, _ := node["data"].(map[string]interface{})
			config, _ := data["config"].(map[string]interface{})

			log.Printf("    🔄 Executing nested loop node %s in loop body (current depth: %d)", nodeID, limits.currentDepth)

			// Increment depth for nested loop
			limits.currentDepth++

			// Check depth limit before executing
			if err := s.checkExecutionLimits(ctx, limits); err != nil {
				limits.currentDepth-- // Restore depth before returning error
				return nil, fmt.Errorf("nested loop depth limit exceeded at node %s: %w", nodeID, err)
			}

			// Create a local executed map for the nested loop
			nestedExecuted := make(map[string]bool)
			err := s.executeLoopWithChildren(ctx, executionID, accountID, nodeID, config, currentOutput, workflowData, nodeMap, adjacencyList, edges, nestedExecuted, workflowData["nodeOutputs"].(map[string]interface{}), limits)

			// Decrement depth after execution
			limits.currentDepth--

			if err != nil {
				return nil, fmt.Errorf("nested loop execution failed: %w", err)
			}
			// Get the loop output and continue
			if loopOutput, ok := workflowData["nodeOutputs"].(map[string]interface{})[nodeID]; ok {
				currentOutput = loopOutput.(map[string]interface{})
			}
			// Loop handles its own children, don't add them to queue
			continue
		}

		if nodeType == "loop-accumulator" {
			data, _ := node["data"].(map[string]interface{})
			config, _ := data["config"].(map[string]interface{})

			log.Printf("    🔄 Executing nested loop accumulator node %s in loop body (current depth: %d)", nodeID, limits.currentDepth)

			// Increment depth for nested loop accumulator
			limits.currentDepth++

			// Check depth limit before executing
			if err := s.checkExecutionLimits(ctx, limits); err != nil {
				limits.currentDepth-- // Restore depth before returning error
				return nil, fmt.Errorf("nested loop accumulator depth limit exceeded at node %s: %w", nodeID, err)
			}

			// Create a local executed map for the nested loop accumulator
			nestedExecuted := make(map[string]bool)
			err := s.executeLoopAccumulatorWithChildren(ctx, executionID, accountID, nodeID, config, currentOutput, workflowData, nodeMap, adjacencyList, edges, nestedExecuted, workflowData["nodeOutputs"].(map[string]interface{}), limits)

			// Decrement depth after execution
			limits.currentDepth--

			if err != nil {
				return nil, fmt.Errorf("nested loop accumulator execution failed: %w", err)
			}
			// Get the loop accumulator output and continue
			if loopOutput, ok := workflowData["nodeOutputs"].(map[string]interface{})[nodeID]; ok {
				currentOutput = loopOutput.(map[string]interface{})
			}
			// Loop accumulator handles its own children, don't add them to queue
			continue
		}

		// Get node config
		data, _ := node["data"].(map[string]interface{})
		config, _ := data["config"].(map[string]interface{})

		// Execute this node and create node execution record with parent context
		log.Printf("    ▶ Executing loop body node %s (type: %s)", nodeID, nodeType)
		output, err := s.executeNodeInLoop(ctx, executionID, accountID, nodeID, nodeType, config, currentOutput, workflowData, parentLoopNodeID)
		if err != nil {
			return nil, fmt.Errorf("subgraph node %s execution failed: %w", nodeID, err)
		}

		// Update current output for next node in chain
		currentOutput = output

		// Add child nodes to queue, but skip if child is the parent loop node
		for _, nextNodeID := range adjacencyList[nodeID] {
			// Don't follow edges back to the parent loop node
			if parentLoopNodeID != "" && nextNodeID == parentLoopNodeID {
				log.Printf("    🔙 Skipping edge to parent loop node %s", nextNodeID)
				continue
			}

			// Check edge conditions
			subgraphNodeOutputs := map[string]interface{}{
				nodeID: currentOutput,
			}
			shouldAdd := s.checkEdgeCondition(edges, nodeID, nextNodeID, subgraphNodeOutputs)
			if shouldAdd {
				queue = append(queue, nextNodeID)
			}
		}
	}

	return currentOutput, nil
}
