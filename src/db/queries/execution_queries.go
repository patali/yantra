package queries

import (
	"fmt"
	"time"

	"github.com/patali/yantra/src/executors"
	"gorm.io/gorm"
)

// ExecutionInfo holds aggregated information about workflow executions
type ExecutionInfo struct {
	ExecutionID     string
	WorkflowID      string
	Version         int
	TotalNodes      int64
	FailedNodes     int64
	SuccessNodes    int64
	RunningNodes    int64
	PendingMessages int64
	HasEndNode      bool
	StartedAt       time.Time
}

// FindRunningExecutionsWithStats queries running executions with their statistics
// This includes node execution counts, pending messages, and whether they reached the end node
func FindRunningExecutionsWithStats(db *gorm.DB) ([]ExecutionInfo, error) {
	var executionInfos []ExecutionInfo

	// Build query using node type constant to ensure consistency with code
	query := fmt.Sprintf(`
		SELECT
			we.id as execution_id,
			we.workflow_id as workflow_id,
			we.version as version,
			we.started_at as started_at,
			COUNT(DISTINCT wne.id) as total_nodes,
			SUM(CASE WHEN wne.status = 'error' THEN 1 ELSE 0 END) as failed_nodes,
			SUM(CASE WHEN wne.status = 'success' THEN 1 ELSE 0 END) as success_nodes,
			SUM(CASE WHEN wne.status = 'running' THEN 1 ELSE 0 END) as running_nodes,
			COUNT(DISTINCT CASE WHEN om.status IN ('pending', 'processing') THEN om.id END) as pending_messages,
			BOOL_OR(wne.node_type = '%s' AND wne.status = 'success') as has_end_node
		FROM workflow_executions we
		LEFT JOIN workflow_node_executions wne ON wne.execution_id = we.id
		LEFT JOIN outbox_messages om ON om.node_execution_id = wne.id
		WHERE we.status = 'running'
		GROUP BY we.id, we.workflow_id, we.version, we.started_at
	`, executors.NodeTypeEnd)

	err := db.Raw(query).Scan(&executionInfos).Error
	if err != nil {
		return nil, err
	}

	return executionInfos, nil
}

// ExecutionWithWorkflowInfo holds execution data with workflow details for listing
type ExecutionWithWorkflowInfo struct {
	ExecutionID  string
	WorkflowID   string
	WorkflowName string
	Status       string
	StartedAt    time.Time
	CompletedAt  *time.Time
}

// FindExecutionsWithWorkflowInfo queries executions with their workflow information
// This is useful for listing executions with workflow names
func FindExecutionsWithWorkflowInfo(db *gorm.DB, accountID string, limit int, status string) ([]ExecutionWithWorkflowInfo, error) {
	var results []ExecutionWithWorkflowInfo

	query := db.Table("workflow_executions we").
		Select("we.id as execution_id, we.workflow_id, w.name as workflow_name, we.status, we.started_at, we.completed_at").
		Joins("INNER JOIN workflows w ON w.id = we.workflow_id").
		Where("w.account_id = ?", accountID).
		Order("we.started_at DESC")

	if status != "" && status != "all" {
		query = query.Where("we.status = ?", status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

// NodeExecutionStats holds statistics about node executions for a workflow execution
type NodeExecutionStats struct {
	ExecutionID  string
	TotalNodes   int64
	SuccessNodes int64
	FailedNodes  int64
	RunningNodes int64
	PendingNodes int64
}

// GetNodeExecutionStats retrieves aggregated statistics for node executions
func GetNodeExecutionStats(db *gorm.DB, executionID string) (*NodeExecutionStats, error) {
	var stats NodeExecutionStats

	query := `
		SELECT
			execution_id,
			COUNT(*) as total_nodes,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_nodes,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as failed_nodes,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running_nodes,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_nodes
		FROM workflow_node_executions
		WHERE execution_id = ?
		GROUP BY execution_id
	`

	err := db.Raw(query, executionID).Scan(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &NodeExecutionStats{ExecutionID: executionID}, nil
		}
		return nil, err
	}

	return &stats, nil
}
