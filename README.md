# Yantra Monorepo

**A reliable, visual workflow automation platform for building complex business processes without code.**

Yantra enables you to design and execute workflows using a visual, node-based editor. What you see in the editor is exactly what executes—no hidden compilation steps, no surprises.

## Monorepo Structure

This repository contains both the backend server and frontend web application in a unified monorepo:

```
yantra/
├── backend/          # Go backend server
│   ├── cmd/         # Command-line applications
│   ├── src/         # Source code
│   ├── docs/        # Backend documentation
│   └── go.mod       # Go dependencies
├── frontend/        # Vue.js frontend application
│   ├── src/         # Vue components and views
│   ├── public/      # Static assets
│   └── package.json # Node dependencies
├── docker-compose.yml        # Docker orchestration
├── Dockerfile.backend        # Backend container definition
├── Dockerfile.frontend       # Frontend container definition
└── README.md                # This file
```

## Quick Start

### Prerequisites

- Docker and Docker Compose (recommended)
- PostgreSQL 15+ (for local development)
- Go 1.21+ (for backend development without Docker)
- Node.js 20+ (for frontend development without Docker)

### Running with Docker (Recommended)

The easiest way to run Yantra is with Docker Compose:

1. **Set up PostgreSQL on your host machine**

```bash
# Create the database and user
psql -U postgres
```

```sql
CREATE DATABASE yantra;
CREATE USER yantra WITH PASSWORD 'yantra_dev_password';
GRANT ALL PRIVILEGES ON DATABASE yantra TO yantra;
```

2. **Set environment variables** (optional)

Copy the example file and update with your values:

```bash
cp env.example .env
# Edit .env with your database credentials and JWT secret
```

Or create a `.env` file manually:

```bash
DATABASE_URL=postgresql://yantra:yantra_dev_password@host.docker.internal:5432/yantra?sslmode=disable
JWT_SECRET=your-secure-jwt-secret-min-32-chars-long
NODE_ENV=development
MIGRATION_API_KEY=
```

3. **Start all services**

```bash
docker-compose up --build
```

The services will be available at:
- **Frontend**: http://localhost:4700
- **Backend API**: http://localhost:4701
- **PostgreSQL**: localhost:5432 (on host machine)

### Local Development (Without Docker)

#### Quick Start with Tmux (Recommended)

The easiest way to run both backend and frontend together:

```bash
./dev.sh
```

This script will:
- ✅ Check all prerequisites (Go, Node, tmux, PostgreSQL)
- ✅ Install dependencies if needed
- ✅ Create a tmux session with both services running side-by-side
- ✅ Auto-attach to the session

**Tmux commands:**
- **Detach**: `Ctrl+b` then `d` (services keep running)
- **Reattach**: `tmux attach-session -t yantra-dev`
- **Switch panes**: `Ctrl+b` then arrow keys
- **Stop all**: `tmux kill-session -t yantra-dev`

**Access URLs:**
- Frontend: http://localhost:5173 (Vite dev server)
- Backend: http://localhost:3000

#### Manual Backend Development

```bash
cd backend

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable"
export JWT_SECRET="your-secure-jwt-secret"
export PORT=3000

# Run the server
go run cmd/server/main.go
```

The backend will start on http://localhost:3000

#### Manual Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Set environment variables (create .env file)
echo "VITE_API_URL=http://localhost:3000" > .env

# Run development server
npm run dev
```

The frontend will start on http://localhost:5173 (Vite default)

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

**3. Fault Tolerance**
- Every node execution is checkpointed in the database
- Workflows can resume from the last successful node after failures
- Context cancellation doesn't lose progress
- Configurable retry policies

For detailed architecture documentation, see:
- [Workflow Architecture](./backend/docs/WORKFLOW_ARCHITECTURE.md) - Complete workflow engine design
- [Outbox Pattern](./backend/docs/OUTBOX_ARCHITECTURE.md) - Reliability and guaranteed delivery

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

## Testing

### Backend Tests

```bash
cd backend

# Run all tests
go test ./...

# Unit tests only
go test ./src/executors/... ./src/services/...

# Integration tests
go test ./src/workflows/... -tags=integration

# With coverage
go test -cover ./...
```

### Frontend Tests

```bash
cd frontend

# Run unit tests
npm run test

# Run with coverage
npm run test:coverage
```

## Docker Networking Notes

- **macOS/Windows**: The containers use `host.docker.internal` to connect to PostgreSQL on your host machine
- **Linux**: If `host.docker.internal` doesn't work, you may need to:
  - Use the host's IP address in the `DATABASE_URL`
  - Or add `network_mode: host` to the yantra-server service in docker-compose.yml

## Configuration

### Backend Environment Variables

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

### Frontend Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL | `http://localhost:4701` |

## Production Deployment

### Docker Production Build

```bash
# Build images
docker-compose build

# Run in production mode
NODE_ENV=production docker-compose up -d
```

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
- [ ] Set proper `VITE_API_URL` for frontend

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

### Webhooks
- `POST /api/webhooks/:workflowId` - Trigger via webhook
- `POST /api/webhooks/:workflowId/:path` - Custom webhook path
- `POST /api/workflows/:id/webhook-secret/regenerate` - Regenerate webhook secret

## Contributing

We welcome contributions! To contribute:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes in the appropriate directory (`backend/` or `frontend/`)
4. Run tests to ensure everything works
5. Commit: `git commit -m 'Add amazing feature'`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

## Documentation

- [Backend Documentation](./backend/README.md) - Detailed backend documentation
- [Workflow Architecture](./backend/docs/WORKFLOW_ARCHITECTURE.md) - Workflow engine design
- [Outbox Pattern](./backend/docs/OUTBOX_ARCHITECTURE.md) - Reliability and messaging

## Support

- GitHub Issues: Report bugs or request features
- Documentation: See the docs/ directory
- Community: Join our community (coming soon)

## License

[Add your license here]

---

**Built with ❤️ for workflow automation**

