# Yantra

**A reliable, visual workflow automation platform for building complex business processes without code.**

Yantra lets you design and execute workflows using a visual, node-based editor. What you see in the editor is exactly what executes—no hidden compilation steps, no surprises.

## Features

- 🎨 **Visual Workflow Design** - Drag-and-drop WYSIWYG editor
- 🔒 **Guaranteed Reliability** - Transactional outbox pattern, checkpointing, fault tolerance
- ⚡ **Powerful Integrations** - HTTP APIs, Email, Slack, JSON processing, loops, conditionals
- 🔄 **Flexible Triggers** - Manual, scheduled (cron), webhooks, API calls
- 📊 **Comprehensive Monitoring** - Real-time execution tracking, history, debugging

## Quick Start

### With Docker (Recommended)

1. **Set up PostgreSQL:**

```bash
psql -U postgres -c "CREATE DATABASE yantra;"
psql -U postgres -c "CREATE USER yantra WITH PASSWORD 'yantra_dev_password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE yantra TO yantra;"
```

2. **Configure environment:**

```bash
cp env.example .env
# Edit .env with your database credentials
```

3. **Start services:**

```bash
docker-compose up --build
```

4. **Access Yantra:**
   - Frontend: http://localhost:4700
   - Backend API: http://localhost:4701

### Without Docker

Use the tmux development script for local development:

```bash
./dev.sh
```

This runs both backend and frontend in a split tmux session.

- Frontend: http://localhost:5173
- Backend: http://localhost:3000

## Project Structure

```
yantra/
├── backend/          # Go backend server
├── frontend/         # Vue.js frontend application
├── docs/             # Documentation
├── dev.sh            # Development script (tmux)
├── docker-compose.yml
└── env.example       # Environment template
```

## Documentation

- **[Getting Started](./docs/GETTING_STARTED.md)** - Detailed setup guide
- **[Configuration](./docs/CONFIGURATION.md)** - Environment variables and settings
- **[API Reference](./docs/API.md)** - REST API documentation
- **[Node Types](./docs/NODE_TYPES.md)** - Available workflow nodes
- **[Architecture](./docs/ARCHITECTURE.md)** - System design and principles
- **[Deployment](./docs/DEPLOYMENT.md)** - Production deployment guide

**Backend-Specific:**
- [Workflow Architecture](./backend/docs/WORKFLOW_ARCHITECTURE.md)
- [Outbox Pattern](./backend/docs/OUTBOX_ARCHITECTURE.md)

## Core Concepts

### WYSIWYG Execution
The visual workflow you design is exactly what executes. No compilation or transformation steps.

### Transactional Outbox Pattern
Ensures reliable side-effect execution (email, Slack) without distributed transactions.

### Fault Tolerance
Every node execution is checkpointed. Workflows can resume from the last successful state after failures.

## Available Node Types

| Category | Nodes |
|----------|-------|
| **Control** | Start, End, Conditional, Delay, Sleep |
| **Data** | JSON, JSON Array, Transform, JSON to CSV |
| **Iteration** | Loop, Loop Accumulator |
| **Integration** | HTTP, Email, Slack |

See [Node Types](./docs/NODE_TYPES.md) for detailed documentation.

## Development

### Prerequisites

- Docker & Docker Compose (for containerized setup)
- PostgreSQL 15+ (running on host)
- Go 1.21+ (for backend development)
- Node.js 20+ (for frontend development)
- tmux (optional, for dev.sh script)

### Testing

**Backend:**
```bash
cd backend
go test ./...
```

## Configuration

Key environment variables:

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | Yes |
| `SMTP_*` | Email configuration | For email nodes |
| `VITE_API_URL` | Frontend API URL | Build time |

See [Configuration Guide](./docs/CONFIGURATION.md) for details.

## Production Deployment

1. Set strong `JWT_SECRET` and `DATABASE_URL`
2. Configure SMTP for email nodes
3. Set up SSL/TLS certificates
4. Use reverse proxy (nginx/caddy)
5. Enable monitoring and backups

See [Deployment Guide](./docs/DEPLOYMENT.md) for complete instructions.

## Contributing

We welcome contributions!

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes in `backend/` or `frontend/`
4. Run tests
5. Commit: `git commit -m 'Add amazing feature'`
6. Push and open a Pull Request

## License

[MIT LICENSE](./LICENSE)

## Support

- **Issues**: [GitHub Issues](https://github.com/patali/yantra/issues)
- **Documentation**: [docs/](./docs/)
---
