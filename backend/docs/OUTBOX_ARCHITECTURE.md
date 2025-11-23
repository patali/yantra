# Outbox Pattern Architecture

This document describes how Yantra implements the transactional outbox pattern to ensure reliable execution of **async nodes** (email, Slack) with guaranteed message delivery.

## Important Note: Scope of Outbox Pattern

**The outbox pattern in Yantra is used for async node execution (email/Slack), NOT for workflow triggering.**

- **Workflow triggering**: Uses direct River queue insertion (synchronous, no outbox needed)
- **Async nodes (email/Slack)**: Use outbox pattern for reliable side-effect execution

This design ensures:
- Workflows execute immediately without outbox overhead
- Side-effect operations (email/Slack) have retry logic and guaranteed delivery
- HTTP nodes execute synchronously so their output is available to downstream nodes

## Problem Statement

Async operations with side effects face a critical reliability issue:

```go
// ❌ UNRELIABLE: Race condition between DB and side effect
func ExecuteEmailNode(node) {
    // 1. Save execution to database
    execution := db.CreateNodeExecution(node)

    // 💥 SYSTEM CRASH HERE = Email never sent!

    // 2. Send email
    emailService.Send(node.config)
}
```

**Problems:**
- If the system crashes between steps 1 and 2, the node execution exists but email never sent
- The side effect never happens
- No automatic recovery or retry
- Manual intervention required

## Solution: Transactional Outbox Pattern

The outbox pattern ensures **at-least-once delivery** for async operations by:
1. Writing both the node execution record and the outbox message in a **single transaction**
2. Using a separate processor to reliably execute side effects from the outbox queue
3. Guaranteeing that if the node execution exists, the side effect will eventually be processed

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                   Workflow Execution Engine                        │
│                  (executing async node: email/Slack)               │
└────────────┬───────────────────────────────────────────────────────┘
             │
             │ 1. Single Transaction
             ▼
┌────────────────────────────────────────────────────────────────────┐
│                         Database                                   │
│                                                                    │
│  ┌──────────────────────────┐      ┌─────────────────────────┐   │
│  │ workflow_node_executions │      │   outbox_messages       │   │
│  │                          │      │                         │   │
│  │ - id                     │      │ - id                    │   │
│  │ - execution_id           │◄─────┤ - node_execution_id     │   │
│  │ - node_id                │      │ - event_type            │   │
│  │ - node_type: email       │      │ - payload               │   │
│  │ - status: pending        │      │ - status: pending       │   │
│  │ - created_at             │      │ - created_at            │   │
│  └──────────────────────────┘      │ - attempts: 0           │   │
│                                     │ - max_attempts: 4       │   │
│                                     └─────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
             │
             │ 2. Outbox Worker (Background, polls every 1s)
             ▼
┌────────────────────────────────────────────────────────────────────┐
│                      Outbox Processor                              │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  - Fetch pending messages                                    │ │
│  │  - Execute side effect (send email, Slack message)           │ │
│  │  - Mark as completed or retry with exponential backoff       │ │
│  └──────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
             │
             │ 3. Execute Side Effect
             ▼
┌────────────────────────────────────────────────────────────────────┐
│              External Services (SMTP, Slack API)                   │
└────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Outbox Message Table

Stores async operations to be processed:

```sql
CREATE TABLE outbox_messages (
    id                UUID PRIMARY KEY,
    node_execution_id UUID NOT NULL,           -- References workflow_node_executions
    event_type        VARCHAR(50) NOT NULL,    -- 'email.send', 'slack.send'
    payload           JSONB NOT NULL,          -- Node config + input data
    status            VARCHAR(20) NOT NULL,    -- 'pending', 'processing', 'completed', 'dead_letter'
    attempts          INT NOT NULL DEFAULT 0,  -- Number of execution attempts
    max_attempts      INT NOT NULL DEFAULT 4,  -- Maximum retry attempts (configurable per node)
    created_at        TIMESTAMP NOT NULL,
    processed_at      TIMESTAMP,
    last_attempt_at   TIMESTAMP,
    next_retry_at     TIMESTAMP,              -- When to retry next (exponential backoff)
    last_error        TEXT,
    idempotency_key   VARCHAR(255) UNIQUE
);
```

### 2. Outbox Service

Handles transactional writes for async nodes:

```go
type OutboxService struct {
    db *gorm.DB
}

// ExecuteNodeWithOutbox writes both node execution and outbox message atomically
func (s *OutboxService) ExecuteNodeWithOutbox(
    ctx context.Context,
    executionID string,
    accountID *string,
    nodeID, nodeType string,
    nodeConfig, input map[string]interface{},
    eventType string,
) (*WorkflowNodeExecution, *OutboxMessage, error) {
    // Execute in a transaction
    err := s.db.Transaction(func(tx *gorm.DB) error {
        // Create node execution record
        nodeExecution := &WorkflowNodeExecution{
            ExecutionID: executionID,
            NodeID:      nodeID,
            NodeType:    nodeType,
            Status:      "pending",
            Input:       inputJSON,
        }
        if err := tx.Create(nodeExecution).Error; err != nil {
            return err
        }

        // Create outbox message
        message := &OutboxMessage{
            NodeExecutionID: nodeExecution.ID,
            EventType:       eventType,  // "email.send" or "slack.send"
            Payload:         payloadJSON,
            Status:          "pending",
            Attempts:        0,
            MaxAttempts:     maxRetries + 1,
        }
        if err := tx.Create(message).Error; err != nil {
            return err
        }

        return nil
    })

    // Both succeed or both fail atomically
    return nodeExecution, message, err
}
```

### 3. Outbox Worker

Background worker that processes async messages directly:

```go
type OutboxWorkerService struct {
    outboxService   *OutboxService
    executorFactory *executors.ExecutorFactory
}

func (w *OutboxWorkerService) Start(ctx context.Context) {
    go w.run(ctx)
}

func (w *OutboxWorkerService) run(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)  // Poll every second

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processBatch(ctx)
        }
    }
}

func (w *OutboxWorkerService) processBatch(ctx context.Context) {
    // Get pending messages ready for processing
    messages, err := w.outboxService.GetPendingMessages(100)
    if err != nil {
        return
    }

    for _, msg := range messages {
        // Mark as processing
        w.outboxService.MarkMessageProcessing(msg.ID)

        // Parse payload and execute the side effect
        var payload map[string]interface{}
        json.Unmarshal([]byte(msg.Payload), &payload)

        // Get the appropriate executor (email or Slack)
        executor, err := w.executorFactory.GetExecutor(msg.NodeExecution.NodeType)
        if err != nil {
            w.outboxService.MarkMessageFailed(msg.ID, err.Error())
            continue
        }

        // Execute the side effect
        result, err := executor.Execute(ctx, execContext)
        if err != nil || !result.Success {
            // Retry with exponential backoff or move to dead letter
            w.outboxService.MarkMessageFailed(msg.ID, result.Error)
            continue
        }

        // Mark as completed
        w.outboxService.MarkMessageCompleted(msg.ID, result.Output)
    }
}
```

**Key differences from workflow triggering:**
- Outbox worker directly executes side effects (no River queue involved)
- Polls every 1 second for pending messages
- Handles retries with exponential backoff
- Dead letter queue for permanently failed messages

## Reliability Guarantees

### At-Least-Once Delivery

**Guarantee:** Every async operation (email/Slack) will eventually be executed.

**How:**
1. Outbox worker polls continuously (every 1 second)
2. Retries failed messages with exponential backoff
3. Processes messages in order
4. No message loss (persisted in database)

### Idempotency Protection

**Problem:** Prevent duplicate emails/Slack messages on retry.

**Solution:**
1. **Idempotency Key**: Unique key per node execution prevents duplicates
2. **External Service Idempotency**: Email/Slack services should handle duplicate requests
3. **Status Tracking**: Messages marked as "processing" to prevent concurrent execution

```go
// Idempotency key format: {execution_id}-{node_id}-{uuid}
idempotencyKey := fmt.Sprintf("%s-%s-%s", executionID, nodeID, uuid.New())

message := &OutboxMessage{
    IdempotencyKey: idempotencyKey,  // Unique constraint in DB
    Status:         "pending",
}
```

### Retry Strategy with Exponential Backoff

**Configuration:**
- Default max attempts: 4 (configurable per node via `maxRetries`)
- Backoff schedule: 2min, 4min, 8min (capped at 1 hour)
- After max attempts: Move to dead letter queue

**Retry Logic:**
```go
// Calculate backoff: 2^attempts minutes
retryDelay := time.Duration(1<<uint(attempts)) * time.Minute
if retryDelay > time.Hour {
    retryDelay = time.Hour
}
nextRetry := time.Now().Add(retryDelay)
```

### Failure Recovery Scenarios

**Scenario 1: App crashes after DB write, before side effect**
```
✅ Node execution exists in DB
✅ Outbox message exists
✅ On restart, worker processes the message
✅ Side effect executes successfully
```

**Scenario 2: External service temporarily unavailable**
```
✅ Outbox message remains "pending"
✅ Worker retries with exponential backoff
✅ Message eventually succeeds when service recovers
```

**Scenario 3: Permanent failure (max retries exceeded)**
```
✅ Message moved to dead letter queue
✅ Node execution marked as "error"
✅ Workflow execution marked as "partially_failed" or "error"
✅ Admin can manually retry from dead letter queue
```

**Scenario 4: Database transaction fails**
```
✅ Both node execution and outbox rolled back
✅ No orphaned records
✅ Workflow continues to next node
```

## Processing Flow

### 1. Async Node Execution (During Workflow)

```go
// Workflow Engine - encountering async node (email/Slack)
func (s *WorkflowEngineService) executeNode(ctx context.Context, nodeID string) error {
    nodeType := getNodeType(nodeID)

    // Check if node requires outbox pattern
    if executors.NodeRequiresOutbox(nodeType) {
        // Use outbox pattern for email/Slack nodes
        return s.executeNodeWithOutbox(ctx, executionID, nodeID, nodeType, config, input)
    }

    // Execute synchronously for other nodes (HTTP, transform, etc.)
    return s.executeSynchronousNode(ctx, executionID, nodeID, nodeType, config, input)
}

func (s *WorkflowEngineService) executeNodeWithOutbox(...) error {
    // Create node execution and outbox message atomically
    nodeExecution, outboxMessage, err := s.outboxService.ExecuteNodeWithOutbox(
        ctx, executionID, accountID, nodeID, nodeType, config, input, "email.send",
    )

    if err != nil {
        return err
    }

    log.Printf("📬 Node %s queued for outbox processing", nodeID)
    // Workflow continues to next node immediately
    // Email will be sent asynchronously by outbox worker
    return nil
}
```

### 2. Outbox Worker Processing

```go
// Background worker (runs every 1 second)
func (w *OutboxWorkerService) processBatch(ctx context.Context) {
    // Get pending messages
    messages, _ := w.outboxService.GetPendingMessages(100)

    for _, msg := range messages {
        // Mark as processing (prevents concurrent execution)
        w.outboxService.MarkMessageProcessing(msg.ID)

        // Parse payload
        var payload map[string]interface{}
        json.Unmarshal([]byte(msg.Payload), &payload)

        // Get executor and execute side effect
        executor, _ := w.executorFactory.GetExecutor(msg.NodeExecution.NodeType)
        result, err := executor.Execute(ctx, execContext)

        if err != nil || !result.Success {
            // Retry with exponential backoff or move to dead letter
            w.outboxService.MarkMessageFailed(msg.ID, result.Error)
            continue
        }

        // Mark as completed
        w.outboxService.MarkMessageCompleted(msg.ID, result.Output)
    }
}
```

### 3. Side Effect Execution

```go
// Email Executor
func (e *EmailExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
    // Extract config
    to := execCtx.NodeConfig["to"].(string)
    subject := execCtx.NodeConfig["subject"].(string)
    body := execCtx.NodeConfig["body"].(string)

    // Send email via SMTP
    err := e.emailService.SendEmail(to, subject, body)
    if err != nil {
        return &ExecutionResult{
            Success: false,
            Error:   err.Error(),
        }, err
    }

    return &ExecutionResult{
        Success: true,
        Output:  map[string]interface{}{"sent": true, "to": to},
    }, nil
}
```

## Error Handling

### Outbox Message Failures

**Temporary failures** (queue unavailable):
- Message remains "pending"
- Retried on next processor iteration
- Exponential backoff on repeated failures

**Permanent failures** (invalid payload):
- Message marked as "failed"
- Error logged for investigation
- Admin notification (future)

### Job Execution Failures

**Transient errors** (network timeout):
- River retries based on retry policy
- Exponential backoff between attempts
- Max retry limit (e.g., 5 attempts)

**Permanent errors** (invalid workflow):
- Execution marked as "failed"
- Error details stored in database
- No automatic retry (requires manual fix)

## Monitoring

### Metrics to Track

1. **Outbox Lag**: Time between message creation and processing
2. **Pending Count**: Number of unprocessed messages
3. **Failed Count**: Number of failed messages
4. **Processing Rate**: Messages processed per second
5. **Queue Depth**: Number of jobs in River queue

### Alerts

**Critical:**
- Outbox lag > 60 seconds
- Failed messages > 100
- Processor not running

**Warning:**
- Outbox lag > 10 seconds
- Pending messages > 1000
- Failed messages > 10

## Performance Optimization

### Batch Processing

Process multiple messages in a single iteration:

```go
messages := s.getPendingMessages(batchSize: 100)
```

### Parallel Processing

Process messages concurrently (with care):

```go
// Use worker pool
for _, msg := range messages {
    s.workerPool.Submit(func() {
        s.processMessage(msg)
    })
}
```

### Index Optimization

```sql
-- Fast pending message lookup
CREATE INDEX idx_outbox_status_created
ON outbox_messages(status, created_at)
WHERE status = 'pending';

-- Fast aggregate lookup
CREATE INDEX idx_outbox_aggregate
ON outbox_messages(aggregate_id, aggregate_type);
```

## Comparison with Alternatives

### Direct Side Effect Execution (Synchronous)

```go
// ❌ NOT RELIABLE for side effects
func ExecuteEmailNode(node) {
    db.CreateNodeExecution(node)
    emailService.Send(...)  // Can fail after DB commit
}
```

**Problems:**
- No atomicity between DB write and side effect
- Can lose messages if process crashes
- No automatic retry
- Blocks workflow execution

### Immediate Queue Insertion

```go
// ⚠️ BETTER but still has issues
func ExecuteEmailNode(node) {
    db.CreateNodeExecution(node)
    queue.Enqueue(emailJob)  // Separate system, can fail
}
```

**Problems:**
- Two separate systems (DB + Queue)
- Race condition if crash between steps
- Requires queue to be always available
- No built-in retry logic

### Outbox Pattern (Yantra's Approach)

```go
// ✅ RELIABLE & SIMPLE
tx := db.Begin()
tx.Create(nodeExecution)
tx.Create(outboxMessage)
tx.Commit()
// Worker processes outbox messages asynchronously
```

**Benefits:**
- ✅ Single database transaction (atomic)
- ✅ Guaranteed delivery (at-least-once)
- ✅ Simple implementation (no external queue needed)
- ✅ Observable and debuggable (all messages in DB)
- ✅ Automatic recovery (worker polls continuously)
- ✅ Built-in retry with exponential backoff
- ✅ Dead letter queue for failed messages
- ✅ Non-blocking for workflow execution

## Current Features

✅ **Dead Letter Queue**: Permanently failed messages automatically moved to dead letter status
✅ **Retry Logic**: Exponential backoff with configurable max attempts
✅ **Idempotency**: Unique keys prevent duplicate processing
✅ **Account Isolation**: Messages filtered by account for multi-tenancy
✅ **Manual Retry**: Dead letter messages can be manually retried via API

## Future Enhancements

1. **Metrics Dashboard**: Visualize outbox health and processing rates
2. **Message Partitioning**: Shard by account for horizontal scaling
3. **Compression**: Reduce payload size for large messages
4. **Archive Old Messages**: Clean up processed/completed messages
5. **Priority Queue**: Priority-based message processing
6. **Batch Processing**: Send multiple emails in a single SMTP connection

---

**Related Documents:**
- [Workflow Architecture](./WORKFLOW_ARCHITECTURE.md)
- [Test Strategy](./TEST_STRATEGY.md)
