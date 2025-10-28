package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/patali/yantra/internal/executors"
	"github.com/patali/yantra/internal/models"
	"gorm.io/gorm"
)

type WorkflowEngineService struct {
	db              *gorm.DB
	executorFactory *executors.ExecutorFactory
	outboxService   *OutboxService
}

func NewWorkflowEngineService(db *gorm.DB) *WorkflowEngineService {
	return &WorkflowEngineService{
		db:              db,
		executorFactory: executors.NewExecutorFactory(db),
		outboxService:   NewOutboxService(db),
	}
}

// ExecuteWorkflow executes a workflow (called by River worker)
func (s *WorkflowEngineService) ExecuteWorkflow(ctx context.Context, workflowID, executionID, inputJSON, triggerType string) error {
	log.Printf("🔄 Starting workflow execution: %s (execution: %s)", workflowID, executionID)

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

	// Get the existing execution record (created before queuing the job)
	var execution models.WorkflowExecution
	if err := s.db.First(&execution, "id = ?", executionID).Error; err != nil {
		return fmt.Errorf("execution record not found: %w", err)
	}

	// Update status to running
	s.db.Model(&execution).Update("status", "running")

	// Parse workflow definition
	var definition map[string]interface{}
	if err := json.Unmarshal([]byte(latestVersion.Definition), &definition); err != nil {
		return fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// Execute workflow
	err := s.executeWorkflowDefinition(ctx, execution.ID, workflow.AccountID, definition, input)

	// Update execution status
	now := time.Now()
	if err != nil {
		errMsg := err.Error()
		s.db.Model(&execution).Updates(map[string]interface{}{
			"status":       "error",
			"error":        errMsg,
			"completed_at": now,
		})
		return err
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
func (s *WorkflowEngineService) executeWorkflowDefinition(ctx context.Context, executionID string, accountID *string, definition map[string]interface{}, input map[string]interface{}) error {
	nodes, ok := definition["nodes"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid workflow definition: missing nodes")
	}

	edges, ok := definition["edges"].([]interface{})
	if !ok {
		edges = []interface{}{} // Empty edges is okay
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
		if nodeType == "start" {
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
		currentNodeID := queue[0]
		queue = queue[1:]

		if executed[currentNodeID] {
			continue
		}

		executed[currentNodeID] = true
		currentNode := nodeMap[currentNodeID]
		nodeType, _ := currentNode["type"].(string)

		// Skip start and end nodes for execution
		if nodeType != "start" && nodeType != "end" {
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
				// Execute loop and its child nodes iteratively
				err := s.executeLoopWithChildren(ctx, executionID, accountID, currentNodeID, config, nodeInput, workflowData, nodeMap, adjacencyList, executed, nodeOutputs)
				if err != nil {
					return fmt.Errorf("loop execution failed (%s): %w", currentNodeID, err)
				}
				// Skip adding next nodes to queue here - they're already executed in the loop
				continue
			} else if nodeType == "loop-accumulator" {
				// Execute loop accumulator with feedback loop
				err := s.executeLoopAccumulatorWithChildren(ctx, executionID, accountID, currentNodeID, config, nodeInput, workflowData, nodeMap, adjacencyList, edges, executed, nodeOutputs)
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
		for _, nextNodeID := range adjacencyList[currentNodeID] {
			if !executed[nextNodeID] {
				queue = append(queue, nextNodeID)
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
	executed map[string]bool,
	nodeOutputs map[string]interface{},
) error {
	log.Printf("  🔄 Executing loop node %s", loopNodeID)

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
			err := s.executeSubgraph(ctx, executionID, accountID, childNodeID, iterationInput, workflowData, nodeMap, adjacencyList, executed)
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
	executedGlobal map[string]bool,
) error {
	// Track what we execute in this subgraph iteration
	queue := []string{startNodeID}
	currentOutput := input

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		node, exists := nodeMap[nodeID]
		if !exists {
			continue
		}

		nodeType, _ := node["type"].(string)

		// Skip start, end, and loop nodes in subgraph
		if nodeType == "start" || nodeType == "end" || nodeType == "loop" {
			// Add children to queue but don't execute
			for _, nextNodeID := range adjacencyList[nodeID] {
				queue = append(queue, nextNodeID)
			}
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

		// Add child nodes to queue
		for _, nextNodeID := range adjacencyList[nodeID] {
			queue = append(queue, nextNodeID)
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
) error {
	log.Printf("  🔄 Executing loop accumulator node %s", loopNodeID)

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

		// Debug: Log edge details when source is our loop node
		if source == loopNodeID {
			log.Printf("  🔍 Found edge from loop node: source=%s, sourceHandle=%s, target=%s", source, sourceHandle, target)
		}

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
			"iteration_count": iterationCount,
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
			output, err := s.executeSubgraphAndGetOutputWithParent(ctx, executionID, accountID, bodyNodeID, loopNodeID, iterationInput, workflowData, nodeMap, adjacencyList)
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
			// Accumulate the result based on mode
			if accumulationMode == "array" {
				// Add to array
				if accArray, ok := accumulated.([]interface{}); ok {
					accumulated = append(accArray, iterationOutput)
				}
			} else if accumulationMode == "last" {
				// Replace with latest
				accumulated = iterationOutput
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
		"iteration_count":  iterationCount,
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
) (map[string]interface{}, error) {
	return s.executeSubgraphAndGetOutputWithParent(ctx, executionID, accountID, startNodeID, "", input, workflowData, nodeMap, adjacencyList)
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
) (map[string]interface{}, error) {
	queue := []string{startNodeID}
	currentOutput := input

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		node, exists := nodeMap[nodeID]
		if !exists {
			continue
		}

		nodeType, _ := node["type"].(string)

		// Skip start, end, and loop nodes in subgraph
		if nodeType == "start" || nodeType == "end" || nodeType == "loop" || nodeType == "loop-accumulator" {
			// Don't add children if this is the parent loop node (feedback edge)
			if parentLoopNodeID != "" && nodeID == parentLoopNodeID {
				log.Printf("    🔙 Reached parent loop node %s, stopping traversal", nodeID)
				continue
			}
			// Add children to queue but don't execute
			for _, nextNodeID := range adjacencyList[nodeID] {
				queue = append(queue, nextNodeID)
			}
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
			queue = append(queue, nextNodeID)
		}
	}

	return currentOutput, nil
}
