# Yantra Architecture

This document explains the key architectural decisions and design principles of Yantra.

## System Overview

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Browser   │────────▶│   Frontend   │────────▶│   Backend    │
│             │         │  (Vue.js)    │         │    (Go)      │
└─────────────┘         └──────────────┘         └──────┬───────┘
                                                         │
                        ┌────────────────────────────────┼────────┐
                        │                                │        │
                  ┌─────▼─────┐                   ┌─────▼────┐   │
                  │ PostgreSQL │                   │  River   │   │
                  │  Database  │                   │  Queue   │   │
                  └────────────┘                   └──────────┘   │
                                                                  │
                                                        ┌─────────▼──────┐
                                                        │ Outbox Worker  │
                                                        │ (Email, Slack) │
                                                        └────────────────┘
```

## Workflow Execution Model

```
User Designs Workflow (WYSIWYG)
         ↓
Workflow Saved (JSON Definition)
         ↓
Trigger (Manual/Webhook/Schedule)
         ↓
Execution Record Created → Job Queue (River)
         ↓
Worker Executes Nodes
         ↓
Async Nodes → Outbox Pattern (email/Slack)
         ↓
Results Stored with Checkpoints
```

## Key Design Principles

### 1. WYSIWYG Execution

**What you see is what executes** - no hidden transformations or compilation steps.

- Visual workflow designer directly maps to execution
- Node positions are visual-only; execution follows edges
- Real-time validation provides immediate feedback
- JSON workflow definition is the source of truth

### 2. Transactional Outbox Pattern

Ensures reliable side-effect execution without distributed transactions.

**How it works:**
1. Node completes successfully → Write to outbox table in **same transaction**
2. Background worker polls outbox table
3. Worker executes side effect (email, Slack, etc.)
4. On success, mark outbox message as sent
5. On failure, automatic retry with exponential backoff

**Benefits:**
- At-least-once delivery guarantee
- No lost messages even on crashes
- Decouples side effects from workflow execution
- Transactional consistency

**Implementation:**
- Email and Slack nodes use outbox pattern
- Workflow triggering uses direct River queue (immediate execution)
- See `backend/docs/OUTBOX_ARCHITECTURE.md` for details

### 3. Fault Tolerance & Checkpointing

Workflows can recover from failures and resume from last successful state.

**Checkpointing Strategy:**
- Every node execution result stored in database
- Checkpoint created after each successful node
- Failed workflows can resume from last checkpoint
- Context cancellation doesn't lose progress

**Recovery Process:**
1. Detect failed execution
2. Load last successful checkpoint
3. Resume from next node in graph
4. Continue with existing context

### 4. Resource Protection

Prevents runaway workflows from consuming excessive resources.

**Limits:**
- **Max execution time**: 30 minutes
- **Max loop iterations**: 10,000
- **Max data size**: 10MB per node
- **Nested loop depth**: Enforced limits

## Technology Stack

### Backend (Go)

- **Framework**: Custom HTTP router with middleware
- **Database ORM**: GORM
- **Job Queue**: River (PostgreSQL-based)
- **Authentication**: JWT tokens
- **Migrations**: GORM AutoMigrate

**Key Packages:**
- `github.com/riverqueue/river` - Job queue
- `gorm.io/gorm` - ORM
- `github.com/golang-jwt/jwt` - JWT handling
- `github.com/robfig/cron/v3` - Cron scheduling

### Frontend (Vue.js 3)

- **Framework**: Vue 3 with Composition API
- **UI Library**: Vuetify 3
- **Workflow Editor**: Vue Flow (node-based editor)
- **State Management**: Pinia
- **Routing**: Vue Router
- **Build Tool**: Vite

### Database (PostgreSQL)

**Schema Design:**
- `workflows` - Workflow definitions
- `workflow_versions` - Version history
- `executions` - Execution records
- `node_results` - Node execution results (checkpoints)
- `outbox_messages` - Pending side effects
- `sleep_schedules` - Scheduled wake-ups for sleep nodes
- `users` - User accounts
- `accounts` - Multi-tenant accounts

## Execution Engine

### Node Executor Pattern

Each node type implements the `NodeExecutor` interface:

```go
type NodeExecutor interface {
    Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
    Validate() error
}
```

**Benefits:**
- Consistent interface across all node types
- Easy to add new node types
- Testable in isolation
- Type-safe execution

### Workflow Engine

The workflow engine (`backend/src/services/workflow_engine.go`) orchestrates execution:

1. Load workflow definition
2. Build execution graph
3. Find start node
4. Execute nodes in topological order
5. Handle branching (conditionals)
6. Handle loops (iterate over arrays)
7. Store checkpoints after each node
8. Handle errors and recovery

### Async Operations

**Synchronous Nodes:**
- Execute immediately in workflow
- Block until completion
- Examples: JSON, Transform, HTTP, Conditional

**Asynchronous Nodes:**
- Write to outbox, marked complete immediately
- Background worker processes later
- Examples: Email, Slack

## Scheduling System

### Cron-Based Scheduling

Workflows can be scheduled using cron expressions:

```go
// Parse cron expression
schedule, _ := cron.ParseStandard("0 9 * * *")

// Next execution time
next := schedule.Next(time.Now())
```

**Scheduler Service:**
- Runs every minute
- Checks for due scheduled workflows
- Enqueues jobs in River
- Updates next execution time

### Sleep Node Scheduling

Long-term delays use a dedicated table:

```sql
CREATE TABLE sleep_schedules (
  execution_id UUID,
  wake_up_at TIMESTAMP,
  status VARCHAR
);
```

**Sleep Scheduler:**
- Polls `sleep_schedules` every 5 seconds
- Finds executions ready to wake
- Resumes workflow from sleep node
- Worker-friendly (no blocking)

## Security

### Authentication

- JWT-based authentication
- Token expires after 24 hours
- Secure password hashing (bcrypt)
- Email verification (optional)

### Authorization

- Account-based multi-tenancy
- User ownership of workflows
- Middleware enforces ownership checks
- API endpoints protected by auth middleware

### Input Validation

- Request body validation
- Workflow definition validation
- Node configuration validation
- SQL injection prevention (ORM)

## Monitoring & Observability

### Logging

- Structured logging throughout
- Log levels: DEBUG, INFO, WARN, ERROR
- Request/response logging
- Error stack traces

### Execution Tracking

- Real-time execution updates via Server-Sent Events (SSE)
- Complete execution history
- Node-level execution details
- Input/output inspection

### Health Checks

- `/health` endpoint for monitoring
- Database connection check
- Worker status check
- Readiness probes for K8s

## Scalability

### Horizontal Scaling

**Backend:**
- Stateless design enables horizontal scaling
- Multiple backend instances share database
- River queue distributes work across workers
- No session state (JWT tokens)

**Database:**
- PostgreSQL replication for read scaling
- Connection pooling
- Indexing for common queries

### Performance Optimization

- Efficient graph traversal algorithms
- Bulk database operations where possible
- Lazy loading of node results
- Caching of workflow definitions

## Further Reading

- [Workflow Architecture](../backend/docs/WORKFLOW_ARCHITECTURE.md) - Detailed workflow engine design
- [Outbox Pattern](../backend/docs/OUTBOX_ARCHITECTURE.md) - Reliable side-effect execution
- [API Reference](./API.md) - REST API documentation
- [Node Types](./NODE_TYPES.md) - Available workflow nodes

