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
- **delay**: Time-based pauses in execution

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
