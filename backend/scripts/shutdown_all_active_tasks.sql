-- Emergency Shutdown: Cancel All Active Workflow Executions and Outbox Messages
-- This script safely cancels all running/queued workflows and their pending async operations
-- Run this during maintenance, emergency shutdown, or to clean up stuck tasks

-- ============================================================================
-- STEP 1: Cancel all pending/processing outbox messages
-- ============================================================================

UPDATE outbox_messages
SET 
    status = 'cancelled',
    last_error = 'System shutdown - all active tasks cancelled',
    processed_at = NOW()
WHERE status IN ('pending', 'processing');

-- ============================================================================
-- STEP 2: Cancel all pending/running node executions
-- ============================================================================

UPDATE workflow_node_executions
SET 
    status = 'cancelled',
    error = 'System shutdown - all active tasks cancelled',
    completed_at = COALESCE(completed_at, NOW())
WHERE status IN ('pending', 'running');

-- ============================================================================
-- STEP 3: Cancel all running/queued workflow executions
-- ============================================================================

UPDATE workflow_executions
SET 
    status = 'cancelled',
    error = 'System shutdown - all active tasks cancelled',
    completed_at = COALESCE(completed_at, NOW())
WHERE status IN ('running', 'queued', 'interrupted');

-- ============================================================================
-- Summary: Show what was cancelled
-- ============================================================================

-- Count of affected records
SELECT 
    'Outbox Messages Cancelled' as operation,
    COUNT(*) as count
FROM outbox_messages
WHERE status = 'cancelled' 
AND last_error = 'System shutdown - all active tasks cancelled'

UNION ALL

SELECT 
    'Node Executions Cancelled' as operation,
    COUNT(*) as count
FROM workflow_node_executions
WHERE status = 'cancelled'
AND error = 'System shutdown - all active tasks cancelled'

UNION ALL

SELECT 
    'Workflow Executions Cancelled' as operation,
    COUNT(*) as count
FROM workflow_executions
WHERE status = 'cancelled'
AND error = 'System shutdown - all active tasks cancelled';

