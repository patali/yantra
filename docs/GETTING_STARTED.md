# Getting Started with Yantra

This guide will help you get Yantra up and running quickly.

## Prerequisites

- **Docker and Docker Compose** (recommended for quick start)
- **PostgreSQL 15+** (running on your host machine)
- **Go 1.21+** (for backend development without Docker)
- **Node.js 20+** (for frontend development without Docker)
- **tmux** (optional, for local development script)

## Quick Start with Docker

### 1. Set up PostgreSQL

Create the database and user on your host machine:

```bash
psql -U postgres
```

```sql
CREATE DATABASE yantra;
CREATE USER yantra WITH PASSWORD 'yantra_dev_password';
GRANT ALL PRIVILEGES ON DATABASE yantra TO yantra;
```

### 2. Configure Environment (Optional)

Copy the example environment file:

```bash
cp env.example .env
```

Edit `.env` with your database credentials and JWT secret:

```bash
DATABASE_URL=postgresql://yantra:yantra_dev_password@host.docker.internal:5432/yantra?sslmode=disable
JWT_SECRET=your-secure-jwt-secret-min-32-chars-long
```

### 3. Start Services

```bash
docker-compose up --build
```

### 4. Access Yantra

- **Frontend**: http://localhost:4700
- **Backend API**: http://localhost:4701

## Local Development (Without Docker)

### Option 1: Tmux Script (Recommended)

Run both backend and frontend in a split tmux session:

```bash
./dev.sh
```

**Tmux commands:**
- `Ctrl+b` then `d` - Detach (services keep running)
- `tmux attach-session -t yantra-dev` - Reattach
- `Ctrl+b` then arrow keys - Switch panes
- `tmux kill-session -t yantra-dev` - Stop all

**Access URLs:**
- Frontend: http://localhost:5173
- Backend: http://localhost:3000

### Option 2: Manual Setup

**Terminal 1 - Backend:**

```bash
cd backend

# Install dependencies
go mod download

# Set environment
export DATABASE_URL="postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable"
export JWT_SECRET="your-secure-jwt-secret"
export PORT=3000

# Run
go run cmd/server/main.go
```

**Terminal 2 - Frontend:**

```bash
cd frontend

# Install dependencies
npm install

# Configure API URL
echo "VITE_API_URL=http://localhost:3000" > .env

# Run
npm run dev
```

## Next Steps

- [Configuration Guide](./CONFIGURATION.md) - Environment variables and settings
- [API Reference](./API.md) - API endpoints documentation
- [Node Types](./NODE_TYPES.md) - Available workflow nodes
- [Architecture](./ARCHITECTURE.md) - System design and principles

## Troubleshooting

### Database Connection Issues

**Docker:**
- Ensure PostgreSQL is running on your host machine
- Use `host.docker.internal` in DATABASE_URL (macOS/Windows)
- On Linux, use your host's IP or `network_mode: host`

**Local:**
- Ensure PostgreSQL is running: `psql -U yantra -d yantra -c "SELECT 1;"`
- Check DATABASE_URL format: `postgresql://user:password@localhost:5432/database`

### Frontend Build Errors

If you see module errors, reinstall dependencies:

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### Port Already in Use

Change ports in:
- **Docker**: Edit `docker-compose.yml`
- **Local Backend**: Set `PORT` environment variable
- **Local Frontend**: Edit `vite.config.ts`

