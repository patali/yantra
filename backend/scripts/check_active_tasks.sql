-- Check Active Tasks Status
-- Use this to see what's currently running before shutting down

-- ============================================================================
-- Active Workflow Executions
-- ============================================================================

SELECT 
    'WORKFLOW EXECUTIONS' as type,
    status,
    COUNT(*) as count,
    MIN(started_at) as oldest,
    MAX(started_at) as newest
FROM workflow_executions
WHERE status IN ('running', 'queued', 'interrupted')
GROUP BY status

UNION ALL

-- ============================================================================
-- Active Outbox Messages
-- ============================================================================

SELECT 
    'OUTBOX MESSAGES' as type,
    status,
    COUNT(*) as count,
    MIN(created_at) as oldest,
    MAX(created_at) as newest
FROM outbox_messages
WHERE status IN ('pending', 'processing')
GROUP BY status

UNION ALL

-- ============================================================================
-- Active Node Executions
-- ============================================================================

SELECT 
    'NODE EXECUTIONS' as type,
    status,
    COUNT(*) as count,
    MIN(started_at) as oldest,
    MAX(started_at) as newest
FROM workflow_node_executions
WHERE status IN ('pending', 'running')
GROUP BY status

ORDER BY type, status;

-- ============================================================================
-- Details of Running Workflows
-- ============================================================================

SELECT 
    we.id as execution_id,
    w.name as workflow_name,
    we.status,
    we.started_at,
    NOW() - we.started_at as duration,
    COUNT(wne.id) as total_nodes,
    COUNT(CASE WHEN wne.status = 'success' THEN 1 END) as completed_nodes,
    COUNT(CASE WHEN wne.status IN ('pending', 'running') THEN 1 END) as active_nodes,
    COUNT(om.id) as pending_outbox_messages
FROM workflow_executions we
JOIN workflows w ON w.id = we.workflow_id
LEFT JOIN workflow_node_executions wne ON wne.execution_id = we.id
LEFT JOIN outbox_messages om ON om.node_execution_id = wne.id AND om.status IN ('pending', 'processing')
WHERE we.status IN ('running', 'queued', 'interrupted')
GROUP BY we.id, w.name, we.status, we.started_at
ORDER BY we.started_at DESC;

