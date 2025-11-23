# Workflow Architecture

This document describes the architecture and execution model of Yantra's workflow engine.

## Overview

Yantra is a visual workflow automation platform that uses a node-based, directed graph model to execute complex business processes. Workflows are designed visually in a WYSIWYG editor and executed reliably using a transactional outbox pattern with job queues.

## Core Concepts

### Workflow Structure

A workflow consists of:
- **Nodes**: Individual execution units (transforms, conditions, HTTP calls, etc.)
- **Edges**: Directed connections between nodes defining execution flow
- **Workflow Data**: Shared state accessible across all nodes
- **Node Outputs**: Individual results from each node execution

### Node Types

Yantra supports 13 node types across several categories:

#### Control Flow
- **start**: Entry point for workflow execution
- **end**: Terminal node marking workflow completion
- **conditional**: Boolean evaluation for branching logic
- **delay**: Short-term pauses in execution (milliseconds)
- **sleep**: Long-term delays with scheduling (days/weeks/specific dates)

#### Data Processing
- **json**: Static or dynamic JSON data injection
- **json-array**: Array data with schema validation
- **transform**: Data transformation operations (map, extract, parse, stringify, concat)
- **json_to_csv**: Convert JSON arrays to CSV format

#### Iteration
- **loop**: Iterate over arrays with configurable variables
- **loop-accumulator**: Collect results across iterations

#### External Integration
- **http**: HTTP requests with full configuration
- **email**: Email sending with templating
- **slack**: Slack webhook notifications

## Workflow Execution Model

### 1. Workflow Definition

Workflows are stored as JSON with versioning:

```json
{
  "nodes": [
    {
      "id": "node-1",
      "type": "json",
      "label": "Load Data",
      "position": {"x": 100, "y": 100},
      "config": {
        "data": {"users": [...]}
      }
    },
    {
      "id": "node-2",
      "type": "transform",
      "label": "Process",
      "position": {"x": 300, "y": 100},
      "config": {
        "operations": [...]
      }
    }
  ],
  "edges": [
    {
      "id": "e1",
      "source": "node-1",
      "target": "node-2"
    }
  ]
}
```

### 2. Execution Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Workflow Triggered                                   │
│    (Manual, Webhook, Schedule, API)                     │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Create Execution Record                              │
│    - Generate execution ID                              │
│    - Store trigger type & input                         │
│    - Set status: "queued"                               │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Queue Job via Outbox Pattern                         │
│    - Insert into outbox table                           │
│    - Transactionally committed with execution           │
│    - Outbox processor moves to River queue              │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. River Worker Picks Up Job                            │
│    - Dequeues from job queue                            │
│    - Loads workflow definition                          │
│    - Initializes execution context                      │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Execute Workflow Graph                               │
│    - Topological traversal from start node              │
│    - Execute each node with context                     │
│    - Store node results in database                     │
│    - Pass outputs to downstream nodes                   │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Complete Execution                                   │
│    - Mark execution as "success" or "failed"            │
│    - Record completion time                             │
│    - Store final outputs                                │
└─────────────────────────────────────────────────────────┘
```

### 3. Node Execution Context

Each node receives an execution context:

```go
type ExecutionContext struct {
    NodeID       string                 // Current node ID
    NodeConfig   map[string]interface{} // Node configuration
    Input        interface{}            // Input from previous node
    WorkflowData map[string]interface{} // Shared workflow state
    ExecutionID  string                 // Workflow execution ID
    AccountID    string                 // Tenant ID
}
```

### 4. Execution Result

Each node returns a result:

```go
type ExecutionResult struct {
    Success bool                   // Execution success
    Output  map[string]interface{} // Node output data
    Error   string                 // Error message if failed
}
```

## Graph Traversal

### Topological Execution

Workflows are executed using topological sort to ensure:
- Nodes execute only after dependencies complete
- Parallel execution where possible
- Deterministic execution order

### Branching Logic

Conditional nodes create branching paths:
1. Evaluate boolean expression
2. Route to `true` or `false` edge
3. Continue execution from selected branch

### Loop Handling

Loop nodes create sub-executions:
1. Extract array from input
2. For each item, create loop context
3. Execute loop body with item data
4. Collect results in accumulator (optional)

## Reliability & Fault Tolerance

### Abuse Prevention Limits

```go
const (
    MaxExecutionDuration = 30 * time.Minute
    MaxTotalNodes        = 10000
    MaxLoopDepth         = 5
    MaxIterations        = 10000
    MaxAccumulatorSize   = 10 * 1024 * 1024 // 10MB
    MaxDataSize          = 10 * 1024 * 1024 // 10MB
)
```

### Checkpointing

Every node execution is stored in the database:
- Enables resume after failure
- Provides execution history
- Allows debugging and auditing

### Error Handling

**Node-level errors**:
- Captured in execution result
- Stored in database
- Workflow marked as failed
- Error details available in UI

**System-level errors**:
- Database connection failures
- Context timeouts
- Out of memory conditions
- Workflow marked for retry

### Retry Strategy

Failed workflows can be retried:
- Resume from last successful checkpoint
- Skip completed nodes
- Recalculate remaining timeout
- Use background context to prevent cascading cancellation

## Data Flow

### Input Propagation

```
Node A Output → Node B Input
     ↓
Node B Output → Node C Input
```

### Workflow Data (Shared State)

```
┌────────────────────────────────────┐
│ Workflow Data (Global State)      │
│  - workflow.startTime              │
│  - workflow.triggerType            │
│  - workflow.customVariable         │
└─────────────┬──────────────────────┘
              │
              ├─→ Available to all nodes
              ├─→ Can be modified by nodes
              └─→ Persisted across execution
```

### Loop Variables

```
Loop Node:
  items = [A, B, C]

Iteration 1:
  item = A, index = 0

Iteration 2:
  item = B, index = 1

Iteration 3:
  item = C, index = 2
```

## Scalability

### Horizontal Scaling

- **Multiple River Workers**: Process jobs in parallel
- **Worker Pools**: Configurable concurrency per worker
- **Database Sharding**: Partition by account ID (future)

### Vertical Scaling

- **Batch Processing**: Loop execution batching
- **Lazy Loading**: Stream large datasets
- **Result Pagination**: Limit memory usage

## Monitoring & Observability

### Execution Tracking

Every execution creates:
- **WorkflowExecution**: High-level execution record
- **WorkflowNodeExecution**: Per-node execution details

### Metrics

Available metrics:
- Execution duration
- Node execution counts
- Success/failure rates
- Queue depth
- Active workers

### Debugging

Execution records include:
- Input data
- Output data
- Error messages
- Timestamps
- Execution path

## WYSIWYG Editor Integration

### Visual Design

The frontend provides:
- Drag-and-drop node placement
- Visual edge connections
- Real-time validation
- Configuration forms per node type

### Design-to-Execution

What you see is what executes:
- Visual layout mirrors execution graph
- Node configuration directly used by executors
- No compilation or transformation step
- Immediate feedback on errors

### Position Independence

Node positions are visual only:
- Stored in workflow definition
- Used only for rendering
- Execution order determined by edges
- Can reorganize without affecting behavior

## Sleep Node Architecture

### Overview

The sleep node enables workflows to pause execution for extended periods (hours, days, weeks, or until a specific date) without blocking worker threads. This is implemented using a "sleeping" workflow state and database-backed scheduling.

### Design Goals

1. **Non-Blocking**: Workers freed immediately when workflow enters sleep
2. **Persistent**: Sleep schedules survive server restarts
3. **Flexible**: Support both relative (duration from now) and absolute (specific date/time) scheduling
4. **Reliable**: Guaranteed wake-up using database-backed scheduler

### Sleep Execution Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Workflow Executing → Reaches Sleep Node             │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Sleep Executor Calculates Wake-Up Time              │
│    - Relative: now + duration                          │
│    - Absolute: parse target date                       │
│    - Returns NeedsSleep=true, WakeUpAt=<time>          │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Workflow Engine Handles Sleep Signal                │
│    - Mark workflow execution status = "sleeping"       │
│    - Create SleepSchedule record in database           │
│    - Stop workflow execution (free worker)             │
└────────────────┬────────────────────────────────────────┘
                 │
                 │ ... time passes ...
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Scheduler Service Polls (every 5 seconds)           │
│    - Query: SELECT * FROM sleep_schedules              │
│             WHERE wake_up_at <= NOW()                   │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Resume Workflow                                      │
│    - Update execution status = "running"               │
│    - Queue for execution (resume from checkpoint)      │
│    - Delete sleep schedule record                      │
└─────────────────────────────────────────────────────────┘
```

### Database Schema

**SleepSchedule Table:**
```sql
CREATE TABLE workflow_sleep_schedules (
    id UUID PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES workflow_executions(id),
    workflow_id UUID NOT NULL,
    node_id VARCHAR NOT NULL,
    wake_up_at TIMESTAMP NOT NULL,  -- UTC
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_wake_up_at ON workflow_sleep_schedules(wake_up_at);
```

### Configuration Modes

**Relative Mode** (duration from now):
```json
{
  "mode": "relative",
  "duration_value": 7,
  "duration_unit": "days"  // seconds, minutes, hours, days, weeks
}
```

**Absolute Mode** (specific date):
```json
{
  "mode": "absolute",
  "target_date": "2025-12-25T10:00:00Z",
  "timezone": "America/New_York"  // optional
}
```

### Time Zones

- All wake-up times stored as UTC in database
- Relative mode: timezone-agnostic (duration from current moment)
- Absolute mode: parse date in specified timezone, convert to UTC
- Supports all IANA timezone names (e.g., "America/New_York", "Europe/London")

### Edge Cases

**Past Target Dates:**
- Sleep node completes immediately
- Returns success with `sleep_skipped: true`
- Workflow continues to next node

**Server Restarts:**
- Sleep schedules persisted in database
- Scheduler service loads schedules on startup
- Polling resumes automatically
- No sleep time lost

**Very Long Sleeps:**
- No artificial duration limits
- Unlimited sleep duration supported
- Scheduler polls every 5 seconds regardless of duration

### Checkpointing

When a workflow enters sleeping state:
1. Sleep node execution marked as "success" (completed successfully)
2. Node output stored (includes scheduled wake-up time)
3. Workflow execution status = "sleeping"
4. On resume, workflow continues from next node after sleep

### Comparison: Sleep vs Delay

| Feature | Delay Node | Sleep Node |
|---------|-----------|------------|
| Duration | Milliseconds to seconds | Seconds to unlimited |
| Execution | Blocks worker during delay | Non-blocking (sleeping state) |
| Use Case | Short pauses (1-60 seconds) | Long pauses (hours to weeks) |
| Scheduling | In-memory timer | Database-backed scheduler |
| Restart Safety | ✗ Lost on restart | ✓ Survives restarts |
| Date Support | ✗ No | ✓ Yes (absolute mode) |

### Performance Considerations

- **Scheduler Polling**: Every 5 seconds (configurable)
- **Database Load**: Single query per poll cycle
- **Scalability**: Handles thousands of concurrent sleeping workflows
- **Memory**: No in-memory state (all in database)

## Future Enhancements

### Planned Features

1. **Parallel Execution**: Execute independent branches concurrently
2. **Sub-workflows**: Reusable workflow components
3. **Dynamic Routing**: Computed edge selection
4. **Event Triggers**: Real-time event processing
5. **Webhook Responses**: Synchronous webhook handling
6. **Rate Limiting**: Per-node and per-workflow limits
7. **Caching**: Node output caching for efficiency

### Performance Improvements

1. **Query Optimization**: Reduce database round-trips
2. **Connection Pooling**: Efficient database connections
3. **Compiled Expressions**: Cache parsed conditions
4. **Streaming**: Handle large datasets efficiently

---

**Related Documents:**
- [Outbox Architecture](./OUTBOX_ARCHITECTURE.md)
- [Test Strategy](./TEST_STRATEGY.md)
- [Deployment Guide](./DEPLOYMENT.md) (coming soon)
