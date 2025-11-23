# Yantra

**A reliable, visual workflow automation platform for building complex business processes without code.**

Yantra enables you to design and execute workflows using a visual, node-based editor. What you see in the editor is exactly what executes—no hidden compilation steps, no surprises.

## What is Yantra?

Yantra is a workflow automation server that lets you:
- **Design workflows visually** using a drag-and-drop WYSIWYG editor
- **Execute reliably** with guaranteed delivery and fault tolerance
- **Integrate easily** with HTTP APIs, databases, email, Slack, and more
- **Scale effortlessly** with built-in job queues and horizontal scaling
- **Monitor comprehensively** with execution history and debugging tools

### Core Features

🎨 **WYSIWYG Workflow Design**
- Visual node-based editor
- Drag-and-drop workflow creation
- Real-time validation
- What you design is what executes—no hidden transformations

🔒 **Guaranteed Reliability**
- Transactional outbox pattern ensures no lost jobs
- Automatic checkpointing for failure recovery
- Exactly-once execution semantics
- Resume workflows from last successful checkpoint

⚡ **Powerful Node Types**
- **Data Processing**: JSON, transforms, CSV conversion
- **Control Flow**: Conditionals, loops, delays
- **Integrations**: HTTP, Email, Slack
- **Advanced**: Loop accumulators, array processing

🔄 **Flexible Triggers**
- Manual execution
- Scheduled (cron expressions)
- Webhooks (with optional authentication)
- API calls

📊 **Comprehensive Monitoring**
- Execution history and logs
- Node-level debugging
- Success/failure metrics
- Input/output inspection

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- (Optional) Docker & Docker Compose

### Local Development

1. **Clone the repository**
```bash
git clone https://github.com/patali/yantra-server.git
cd yantra-server
```

2. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. **Start the database**
```bash
docker-compose up -d postgres
```

4. **Run the server**
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:3000` and automatically run database migrations.

### Docker Deployment

```bash
docker-compose up --build
```

## Architecture

Yantra uses a robust architecture designed for reliability and scalability:

### Workflow Execution Model

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

### Key Design Principles

**1. WYSIWYG Execution**
- The visual workflow you design is exactly what executes
- Node positions are visual-only; execution follows edges
- No compilation or transformation steps
- Immediate feedback on validation errors

**2. Transactional Outbox Pattern**
- Async nodes (email, Slack) use the outbox pattern for reliable side-effect execution
- Node execution and outbox message created in a single database transaction
- Background worker processes outbox messages with retry logic
- Provides at-least-once delivery for side effects
- Workflow triggering uses direct River queue for immediate execution

**3. Fault Tolerance**
- Every node execution is checkpointed in the database
- Workflows can resume from the last successful node after failures
- Context cancellation doesn't lose progress
- Configurable retry policies

**4. Resource Protection**
- Maximum execution duration (30 minutes)
- Maximum loop iterations (10,000)
- Maximum data size limits (10MB)
- Nested loop depth limits

For detailed architecture documentation, see:
- [Workflow Architecture](./docs/WORKFLOW_ARCHITECTURE.md) - Complete workflow engine design
- [Outbox Pattern](./docs/OUTBOX_ARCHITECTURE.md) - Reliability and guaranteed delivery

## Node Types

| Category | Node Type | Description |
|----------|-----------|-------------|
| **Control** | `start` | Workflow entry point |
| | `end` | Workflow termination |
| | `conditional` | Boolean branching logic |
| | `delay` | Time-based pauses (milliseconds) |
| | `sleep` | Long-term delays (days/weeks/specific dates) |
| **Data** | `json` | Static/dynamic JSON data |
| | `json-array` | Arrays with schema validation |
| | `transform` | Map, extract, parse, stringify |
| | `json_to_csv` | Convert JSON to CSV |
| **Iteration** | `loop` | Iterate over arrays |
| | `loop-accumulator` | Collect iteration results |
| **Integration** | `http` | HTTP/REST API calls |
| | `email` | Email with templates |
| | `slack` | Slack notifications |

## Node Input/Output Format

All nodes in Yantra follow a **standardized input/output format** to ensure consistency and ease of use across workflows.

### Output Format Standard

Every node returns an output object that **always includes a `data` field** containing the primary result of the node's execution. Additional metadata fields may also be included for specific node types.

**Standard Structure:**
```json
{
  "data": <primary_output>,
  // Additional metadata fields...
}
```

### Node-Specific Output Examples

| Node Type | `data` Field | Additional Fields | Example |
|-----------|--------------|-------------------|---------|
| `json` | The JSON object/value | - | `{"data": {"name": "John", "age": 30}}` |
| `transform` | Transformed data | - | `{"data": {"firstName": "John"}}` |
| `http` | Response body (parsed JSON or string) | `status_code`, `url`, `method`, `headers` | `{"data": {...}, "status_code": 200, "headers": {...}}` |
| `conditional` | Boolean result | `result` (backward compat), `condition` | `{"data": true, "result": true, "condition": "x > 5"}` |
| `loop` | Array of iteration results | `iteration_count`, `items`, `results` | `{"data": [{...}], "iteration_count": 10}` |
| `loop-accumulator` | Array of accumulated results | `iteration_count`, `items`, `accumulationMode` | `{"data": [{...}], "iteration_count": 10}` |
| `json-array` | The validated array | `count`, `schema`, `array` | `{"data": [{...}], "count": 5, "schema": {...}}` |
| `json_to_csv` | CSV string | `row_count`, `headers`, `csv`, `converted` | `{"data": "name,age\nJohn,30", "row_count": 1}` |
| `email` | Success boolean (true) | `sent`, `messageId` | `{"data": true, "sent": true, "messageId": "..."}` |
| `slack` | Success boolean (true) | `sent`, `channel`, `text`, `statusCode` | `{"data": true, "sent": true, "channel": "#general"}` |
| `delay` | Delay duration in milliseconds | `delayed_ms` | `{"data": 1000, "delayed_ms": 1000}` |
| `sleep` | Wake-up time (ISO 8601) or true | `sleep_scheduled_until`, `sleep_duration_ms`, `mode` | `{"data": "2025-12-25T10:00:00Z", "mode": "absolute"}` |

### Accessing Node Outputs

In workflows, you can access any node's output using the standardized `data` field:

**In Conditional Nodes:**
```javascript
// Access previous node output
nodeId.data > 10

// Access nested data
nodeId.data.users.length > 0
```

**In HTTP Request Bodies:**
```json
{
  "userId": "{{nodeId.data.id}}",
  "items": "{{loopNode.data}}"
}
```

**In Transform Operations:**
```json
{
  "operations": [
    {
      "type": "extract",
      "config": {
        "jsonPath": "$.data.users[0]"
      }
    }
  ]
}
```

### Backward Compatibility

For existing workflows, original field names are maintained alongside the new `data` field:
- `conditional` still includes `result` field
- `json-array` still includes `array` field  
- `loop` still includes `results` field
- All nodes with specialized fields retain them

This ensures that existing workflows continue to work without modification while new workflows can adopt the standardized `data` field for consistency.

### Sleep Node

The `sleep` node enables workflows to pause execution for extended periods without blocking workers. When a workflow hits a sleep node, it enters a "sleeping" state and is scheduled to resume at the specified time.

**Key Features:**
- **Worker-Friendly**: Workflow enters sleeping state immediately, freeing workers
- **Persistent**: Survives server restarts (schedules stored in database)
- **Full Granularity**: Support for seconds, minutes, hours, days, and weeks
- **Two Modes**: Absolute (specific date) or Relative (duration from now)
- **Timezone Support**: Schedule wake-ups in any timezone
- **Unlimited Duration**: No arbitrary limits on sleep duration

**Configuration Examples:**

```json
// Sleep for 7 days (relative mode)
{
  "type": "sleep",
  "config": {
    "mode": "relative",
    "duration_value": 7,
    "duration_unit": "days"
  }
}

// Sleep until specific date (absolute mode)
{
  "type": "sleep",
  "config": {
    "mode": "absolute",
    "target_date": "2025-12-25T10:00:00Z",
    "timezone": "America/New_York"
  }
}

// Sleep for 2 hours (relative mode)
{
  "type": "sleep",
  "config": {
    "mode": "relative",
    "duration_value": 2,
    "duration_unit": "hours"
  }
}
```

**Duration Units (Relative Mode):**
- `seconds` - Sleep for X seconds
- `minutes` - Sleep for X minutes
- `hours` - Sleep for X hours
- `days` - Sleep for X days
- `weeks` - Sleep for X weeks

**Use Cases:**
- Scheduled reminders (e.g., follow-up after 7 days)
- Campaign automation (e.g., send series of emails over weeks)
- Delayed notifications (e.g., trial expiration warnings)
- Seasonal workflows (e.g., activate on specific dates)

## API Endpoints

### Workflows
- `GET /api/workflows` - List all workflows
- `POST /api/workflows` - Create workflow
- `GET /api/workflows/:id` - Get workflow details
- `PUT /api/workflows/:id` - Update workflow
- `DELETE /api/workflows/:id` - Delete workflow
- `POST /api/workflows/:id/duplicate` - Duplicate workflow

### Execution
- `POST /api/workflows/:id/execute` - Execute workflow
- `POST /api/workflows/:id/executions/:executionId/resume` - Resume interrupted execution
- `GET /api/workflows/:id/executions` - List workflow executions
- `GET /api/workflows/:id/executions/:executionId` - Get execution details
- `GET /api/workflows/:id/executions/:executionId/stream` - Stream execution updates (SSE)

### Scheduling
- `PUT /api/workflows/:id/schedule` - Update workflow schedule (cron)

### Versioning
- `GET /api/workflows/:id/versions` - Get version history
- `POST /api/workflows/:id/versions/restore` - Restore previous version

### Webhooks
- `POST /api/webhooks/:workflowId` - Trigger via webhook
- `POST /api/webhooks/:workflowId/:path` - Custom webhook path
- `POST /api/workflows/:id/webhook-secret/regenerate` - Regenerate webhook secret

## Testing

Yantra includes comprehensive testing:

```bash
# Run all tests
go test ./...

# Unit tests only
go test ./src/executors/... ./src/services/...

# Integration tests
go test ./src/workflows/... -tags=integration

# With coverage
go test -cover ./...
```

See [Test Strategy](./docs/TEST_STRATEGY.md) for detailed testing documentation.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `PORT` | Server port | `3000` |
| `JWT_SECRET` | JWT signing secret | Required |
| `SMTP_HOST` | Email SMTP host | Optional |
| `SMTP_PORT` | Email SMTP port | `587` |
| `SMTP_USER` | Email username | Optional |
| `SMTP_PASS` | Email password | Optional |
| `MIGRATION_API_KEY` | API key for migration endpoint | Optional |

### Database Migrations

Migrations run automatically on server startup. For production deployments with manual control:

1. Generate a secure API key:
```bash
openssl rand -hex 32
```

2. Set `MIGRATION_API_KEY` in environment

3. Run migrations via API:
```bash
curl -X POST -H "X-Migration-Key: your-key" \
  http://localhost:3000/api/migration/run
```

## Deployment

### Docker

```bash
docker build -t yantra .
docker run -p 3000:3000 \
  -e DATABASE_URL="postgres://..." \
  -e JWT_SECRET="your-secret" \
  yantra
```

### Coolify

1. Set environment variables in Coolify
2. Deploy from Git repository
3. Run migrations after deployment (if using manual migration mode)

### Production Checklist

- [ ] Set strong `JWT_SECRET`
- [ ] Configure `DATABASE_URL` for production database
- [ ] Set `MIGRATION_API_KEY` for controlled migrations
- [ ] Configure SMTP for email nodes
- [ ] Set up SSL/TLS certificates
- [ ] Configure reverse proxy (nginx/caddy)
- [ ] Set up monitoring and alerts
- [ ] Enable database backups
- [ ] Configure log aggregation

## Contributing

We welcome contributions! Please see our contributing guidelines (coming soon).

### Development Setup

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Commit: `git commit -m 'Add amazing feature'`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

## Documentation

- [Workflow Architecture](./docs/WORKFLOW_ARCHITECTURE.md) - Detailed workflow engine design
- [Outbox Pattern](./docs/OUTBOX_ARCHITECTURE.md) - Reliability and messaging
- [Test Strategy](./docs/TEST_STRATEGY.md) - Testing approach and guidelines
- [API Documentation](./docs/API.md) - Coming soon
- [Deployment Guide](./docs/DEPLOYMENT.md) - Coming soon

## License

[Add your license here]

## Support

- GitHub Issues: [Report bugs or request features](https://github.com/patali/yantra-server/issues)
- Documentation: [Full documentation](./docs/)
- Email: [Your contact email]

---

**Built with ❤️ for workflow automation**
