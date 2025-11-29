#!/bin/bash

# Yantra Development Script
# Runs both backend and frontend in a tmux session

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SESSION_NAME="yantra-dev"

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    print_error "tmux is not installed. Please install it first:"
    echo "  macOS: brew install tmux"
    echo "  Linux: sudo apt-get install tmux"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

# Check if Node is installed
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed. Please install Node.js 20+ first."
    exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed. Please install npm first."
    exit 1
fi

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"

# Check if directories exist
if [ ! -d "$BACKEND_DIR" ]; then
    print_error "Backend directory not found: $BACKEND_DIR"
    exit 1
fi

if [ ! -d "$FRONTEND_DIR" ]; then
    print_error "Frontend directory not found: $FRONTEND_DIR"
    exit 1
fi

# Check if session already exists
if tmux has-session -t $SESSION_NAME 2>/dev/null; then
    print_warning "Session '$SESSION_NAME' already exists."
    read -p "Do you want to kill it and create a new one? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Killing existing session..."
        tmux kill-session -t $SESSION_NAME
    else
        print_info "Attaching to existing session..."
        tmux attach-session -t $SESSION_NAME
        exit 0
    fi
fi

# Check for environment variables
# The backend uses godotenv.Load() which loads from CWD (backend/)
if [ ! -f "$BACKEND_DIR/.env" ]; then
    print_warning "No .env file found in backend directory."
    print_info "You may want to create backend/.env with:"
    echo "  DATABASE_URL=postgresql://yantra:password@localhost:5432/yantra?sslmode=disable"
    echo "  JWT_SECRET=your-secure-jwt-secret-min-32-chars-long"
    echo ""
    print_info "Or copy from env.example:"
    echo "  cp env.example backend/.env"
    echo ""
fi

# Check if PostgreSQL is accessible (optional check)
print_info "Checking database connection..."
if command -v psql &> /dev/null; then
    # Try to get DATABASE_URL from backend/.env if it exists
    if [ -f "$BACKEND_DIR/.env" ]; then
        source <(grep -E '^DATABASE_URL=' "$BACKEND_DIR/.env" | sed 's/^/export /')
    fi
    DB_URL=${DATABASE_URL:-"postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable"}
    # Extract connection details (simple parsing)
    if [[ $DB_URL =~ postgresql://([^:]+):([^@]+)@([^:]+):([^/]+)/([^?]+) ]]; then
        DB_USER="${BASH_REMATCH[1]}"
        DB_HOST="${BASH_REMATCH[3]}"
        DB_PORT="${BASH_REMATCH[4]}"
        DB_NAME="${BASH_REMATCH[5]}"
        
        if psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
            print_success "Database connection successful!"
        else
            print_warning "Could not connect to database. Make sure PostgreSQL is running."
            print_info "Backend may fail to start if database is not available."
        fi
    fi
else
    print_info "psql not found, skipping database check."
fi

# Install frontend dependencies if needed
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
    print_info "Installing frontend dependencies..."
    cd "$FRONTEND_DIR"
    npm install
    cd "$SCRIPT_DIR"
    print_success "Frontend dependencies installed!"
else
    # Check if node_modules has broken symlinks
    print_info "Checking frontend dependencies..."
    if [ ! -f "$FRONTEND_DIR/node_modules/.bin/vite" ]; then
        print_warning "node_modules appears corrupted, reinstalling..."
        cd "$FRONTEND_DIR"
        rm -rf node_modules package-lock.json
        npm install
        cd "$SCRIPT_DIR"
        print_success "Frontend dependencies reinstalled!"
    else
        print_success "Frontend dependencies already installed!"
    fi
fi

# Download backend dependencies if needed
print_info "Checking backend dependencies..."
cd "$BACKEND_DIR"
go mod download
cd "$SCRIPT_DIR"
print_success "Backend dependencies ready!"

print_success "Starting Yantra development environment..."
echo ""
print_info "Creating tmux session: $SESSION_NAME"
print_info "Layout: Backend (left) | Frontend (right)"
echo ""

# Create new tmux session with backend
tmux new-session -d -s $SESSION_NAME -n "yantra"

# Split the window vertically (backend left, frontend right)
tmux split-window -h -t $SESSION_NAME

# Set up backend pane (left - pane 0)
tmux send-keys -t $SESSION_NAME:0.0 "cd '$BACKEND_DIR'" C-m
tmux send-keys -t $SESSION_NAME:0.0 "echo '🚀 Starting Yantra Backend...'" C-m
tmux send-keys -t $SESSION_NAME:0.0 "echo 'Working directory: \$(pwd)'" C-m
tmux send-keys -t $SESSION_NAME:0.0 "echo ''" C-m

# Load .env if exists from backend directory (where godotenv.Load() looks)
if [ -f "$BACKEND_DIR/.env" ]; then
    tmux send-keys -t $SESSION_NAME:0.0 "export \$(cat '$BACKEND_DIR/.env' | grep -v '^#' | xargs)" C-m
fi

# Set default environment variables if not set
tmux send-keys -t $SESSION_NAME:0.0 "export DATABASE_URL=\${DATABASE_URL:-postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable}" C-m
tmux send-keys -t $SESSION_NAME:0.0 "export JWT_SECRET=\${JWT_SECRET:-your-dev-jwt-secret-min-32-chars-long}" C-m
tmux send-keys -t $SESSION_NAME:0.0 "export PORT=\${PORT:-3000}" C-m
tmux send-keys -t $SESSION_NAME:0.0 "echo 'Environment variables loaded'" C-m
tmux send-keys -t $SESSION_NAME:0.0 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:0.0 "go run cmd/server/main.go" C-m

# Set up frontend pane (right - pane 1)
tmux send-keys -t $SESSION_NAME:0.1 "cd '$FRONTEND_DIR'" C-m
tmux send-keys -t $SESSION_NAME:0.1 "echo '🎨 Starting Yantra Frontend...'" C-m
tmux send-keys -t $SESSION_NAME:0.1 "echo 'Working directory: \$(pwd)'" C-m
tmux send-keys -t $SESSION_NAME:0.1 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:0.1 "export VITE_API_URL=http://localhost:3000" C-m
tmux send-keys -t $SESSION_NAME:0.1 "echo 'API URL: \$VITE_API_URL'" C-m
tmux send-keys -t $SESSION_NAME:0.1 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:0.1 "npm run dev" C-m

# Select the backend pane
tmux select-pane -t $SESSION_NAME:0.0

print_success "Tmux session '$SESSION_NAME' created successfully!"
echo ""
print_info "Access URLs:"
echo "  Frontend: http://localhost:5173 (Vite dev server)"
echo "  Backend:  http://localhost:3000"
echo ""
print_info "Tmux commands:"
echo "  Attach to session:     tmux attach-session -t $SESSION_NAME"
echo "  Detach from session:   Ctrl+b then d"
echo "  Switch panes:          Ctrl+b then arrow keys"
echo "  Kill session:          tmux kill-session -t $SESSION_NAME"
echo "  List sessions:         tmux ls"
echo ""
print_info "Attaching to session in 2 seconds..."
sleep 2

# Attach to the session
tmux attach-session -t $SESSION_NAME

