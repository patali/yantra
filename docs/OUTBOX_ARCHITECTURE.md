# Outbox Pattern Architecture

This document describes how Yantra implements the transactional outbox pattern to ensure reliable workflow execution with guaranteed message delivery.

## Problem Statement

Traditional job queue systems face a critical reliability issue:

```go
// ❌ UNRELIABLE: Race condition between DB and queue
func TriggerWorkflow(workflow) {
    // 1. Save to database
    execution := db.CreateExecution(workflow)

    // 💥 SYSTEM CRASH HERE = Job never queued!

    // 2. Queue job
    queue.Enqueue(execution.ID)
}
```

**Problems:**
- If the system crashes between steps 1 and 2, the execution record exists but no job is queued
- The workflow never executes
- No automatic recovery
- Manual intervention required

## Solution: Transactional Outbox Pattern

The outbox pattern ensures **exactly-once semantics** by:
1. Writing both the execution record and the outbox message in a **single transaction**
2. Using a separate processor to reliably move messages from outbox to queue
3. Guaranteeing that if the execution exists, a job will eventually be queued

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                     Application Layer                              │
└────────────┬───────────────────────────────────────────────────────┘
             │
             │ 1. Single Transaction
             ▼
┌────────────────────────────────────────────────────────────────────┐
│                         Database                                   │
│                                                                    │
│  ┌──────────────────────┐      ┌─────────────────────────┐       │
│  │ workflow_executions  │      │   outbox_messages       │       │
│  │                      │      │                         │       │
│  │ - id                 │      │ - id                    │       │
│  │ - workflow_id        │◄─────┤ - aggregate_id          │       │
│  │ - status: queued     │      │ - event_type            │       │
│  │ - input_data         │      │ - payload               │       │
│  │ - created_at         │      │ - status: pending       │       │
│  └──────────────────────┘      │ - created_at            │       │
│                                 └─────────────────────────┘       │
└────────────────────────────────────────────────────────────────────┘
             │
             │ 2. Outbox Processor (Background)
             ▼
┌────────────────────────────────────────────────────────────────────┐
│                      River Job Queue                               │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  Job: execute_workflow                                       │ │
│  │  - workflow_id                                               │ │
│  │  - execution_id                                              │ │
│  │  - input_data                                                │ │
│  └──────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
             │
             │ 3. River Worker
             ▼
┌────────────────────────────────────────────────────────────────────┐
│                   Workflow Execution Engine                        │
└────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Outbox Message Table

Stores messages to be processed:

```sql
CREATE TABLE outbox_messages (
    id              UUID PRIMARY KEY,
    aggregate_id    VARCHAR(255) NOT NULL,  -- execution_id
    aggregate_type  VARCHAR(50) NOT NULL,   -- 'workflow_execution'
    event_type      VARCHAR(50) NOT NULL,   -- 'workflow.triggered'
    payload         JSONB NOT NULL,         -- Job details
    status          VARCHAR(20) NOT NULL,   -- 'pending', 'processed', 'failed'
    created_at      TIMESTAMP NOT NULL,
    processed_at    TIMESTAMP
);
```

### 2. Outbox Service

Handles transactional writes:

```go
type OutboxService struct {
    db *gorm.DB
}

// AddWorkflowExecution writes both execution and outbox message
func (s *OutboxService) AddWorkflowExecution(
    tx *gorm.DB,
    execution *WorkflowExecution,
    payload map[string]interface{},
) error {
    // Create execution record
    if err := tx.Create(execution).Error; err != nil {
        return err // Transaction will rollback
    }

    // Create outbox message
    message := &OutboxMessage{
        AggregateID:   execution.ID,
        AggregateType: "workflow_execution",
        EventType:     "workflow.triggered",
        Payload:       payload,
        Status:        "pending",
    }

    if err := tx.Create(message).Error; err != nil {
        return err // Transaction will rollback
    }

    // Both succeed or both fail atomically
    return nil
}
```

### 3. Outbox Processor

Background worker that moves messages to River queue:

```go
func (s *OutboxService) ProcessPendingMessages(ctx context.Context) {
    // Run every 1 second
    ticker := time.NewTicker(1 * time.Second)

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.processBatch()
        }
    }
}

func (s *OutboxService) processBatch() {
    // Get pending messages (oldest first)
    var messages []OutboxMessage
    s.db.Where("status = ?", "pending").
        Order("created_at ASC").
        Limit(100).
        Find(&messages)

    for _, msg := range messages {
        // Insert into River queue
        if err := s.queueJob(msg); err != nil {
            s.markFailed(msg.ID, err)
            continue
        }

        // Mark as processed
        s.markProcessed(msg.ID)
    }
}
```

### 4. River Integration

Job queue for workflow execution:

```go
type WorkflowExecutionArgs struct {
    WorkflowID  string                 `json:"workflow_id"`
    ExecutionID string                 `json:"execution_id"`
    InputData   map[string]interface{} `json:"input_data"`
    TriggerType string                 `json:"trigger_type"`
}

type WorkflowExecutionWorker struct {
    river.WorkerDefaults[WorkflowExecutionArgs]
    engineService *WorkflowEngineService
}

func (w *WorkflowExecutionWorker) Work(
    ctx context.Context,
    job *river.Job[WorkflowExecutionArgs],
) error {
    return w.engineService.ExecuteWorkflow(
        ctx,
        job.Args.WorkflowID,
        job.Args.ExecutionID,
        job.Args.InputData,
        job.Args.TriggerType,
    )
}
```

## Reliability Guarantees

### Exactly-Once Semantics

**Problem:** Ensure each workflow executes exactly once, not zero times or multiple times.

**Solution:**
1. **Database Transaction**: Execution + Outbox written atomically
2. **Idempotent Processing**: Check if message already processed
3. **Unique Job IDs**: Prevent duplicate queueing

```go
func (s *OutboxService) queueJob(msg OutboxMessage) error {
    // Use execution_id as unique job ID
    jobID := msg.AggregateID

    // River deduplicates jobs with same ID
    _, err := s.riverClient.InsertTx(ctx, tx, &WorkflowExecutionArgs{
        ExecutionID: msg.AggregateID,
        // ... other args
    }, &river.InsertOpts{
        UniqueOpts: river.UniqueOpts{
            ByArgs: true,
        },
    })

    return err
}
```

### At-Least-Once Delivery

**Guarantee:** Every execution will eventually be queued and executed.

**How:**
1. Outbox processor runs continuously
2. Retries failed messages
3. Processes messages in order
4. No message loss (persisted in database)

### Failure Recovery

**Scenario 1: App crashes after DB write, before queue**
```
✅ Execution exists in DB
✅ Outbox message exists
✅ On restart, processor queues the message
✅ Workflow executes successfully
```

**Scenario 2: Queue unavailable**
```
✅ Outbox message remains "pending"
✅ Processor retries on next iteration
✅ Message eventually queued when queue recovers
```

**Scenario 3: Database transaction fails**
```
✅ Both execution and outbox rolled back
✅ No orphaned records
✅ Client receives error, can retry
```

## Processing Flow

### 1. Workflow Trigger

```go
// API Handler
func (c *WorkflowController) TriggerWorkflow(w http.ResponseWriter, r *http.Request) {
    // Start transaction
    tx := c.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Create execution
    execution := &WorkflowExecution{
        WorkflowID: workflowID,
        Status:     "queued",
        InputData:  inputData,
    }

    // Add to outbox (single transaction)
    if err := c.outboxService.AddWorkflowExecution(tx, execution, payload); err != nil {
        tx.Rollback()
        return err
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        return err
    }

    // ✅ At this point, workflow WILL execute
    return execution.ID
}
```

### 2. Outbox Processing

```go
// Background processor
func (s *OutboxService) processBatch() {
    messages := s.getPendingMessages()

    for _, msg := range messages {
        // Parse payload
        var args WorkflowExecutionArgs
        json.Unmarshal(msg.Payload, &args)

        // Queue job
        job, err := s.riverClient.Insert(ctx, &args, nil)
        if err != nil {
            log.Printf("Failed to queue job: %v", err)
            s.incrementRetryCount(msg.ID)
            continue
        }

        // Mark as processed
        s.db.Model(&OutboxMessage{}).
            Where("id = ?", msg.ID).
            Updates(map[string]interface{}{
                "status":       "processed",
                "processed_at": time.Now(),
            })
    }
}
```

### 3. Job Execution

```go
// River worker
func (w *WorkflowExecutionWorker) Work(ctx context.Context, job *river.Job[WorkflowExecutionArgs]) error {
    // Execute workflow
    err := w.engineService.ExecuteWorkflow(
        ctx,
        job.Args.WorkflowID,
        job.Args.ExecutionID,
        job.Args.InputData,
        job.Args.TriggerType,
    )

    // If error, River will retry based on retry policy
    return err
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

### Direct Queue Insertion

```go
// ❌ NOT RELIABLE
db.Create(execution)
queue.Enqueue(job) // Can fail after DB commit
```

**Problems:**
- No atomicity
- Can lose jobs
- No automatic recovery

### Two-Phase Commit

```go
// ❌ COMPLEX & SLOW
tx1 := db.Begin()
tx2 := queue.Begin()
// ... commit both ...
```

**Problems:**
- Complex implementation
- Performance overhead
- Requires 2PC support in queue

### Outbox Pattern (Yantra's Approach)

```go
// ✅ RELIABLE & SIMPLE
tx.Create(execution)
tx.Create(outboxMessage)
tx.Commit()
```

**Benefits:**
- ✅ Single database transaction
- ✅ Guaranteed delivery
- ✅ Simple implementation
- ✅ Observable and debuggable
- ✅ Automatic recovery

## Future Enhancements

1. **Dead Letter Queue**: Move permanently failed messages
2. **Metrics Dashboard**: Visualize outbox health
3. **Message Partitioning**: Shard by account for scale
4. **Compression**: Reduce payload size
5. **Archive Old Messages**: Clean up processed messages

---

**Related Documents:**
- [Workflow Architecture](./WORKFLOW_ARCHITECTURE.md)
- [Test Strategy](./TEST_STRATEGY.md)
