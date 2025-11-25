#!/bin/bash

# Emergency Shutdown Script for Yantra Active Tasks
# This script cancels all running/queued workflows and pending outbox messages

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default database URL
DEFAULT_DB_URL="postgresql://postgres:dbpassword@localhost:5432/yantra?sslmode=disable"

# Get database URL from environment or use default
DB_URL="${DATABASE_URL:-$DEFAULT_DB_URL}"

# Extract connection details
DB_HOST=$(echo "$DB_URL" | sed -n 's/.*@\([^:]*\):.*/\1/p')
DB_PORT=$(echo "$DB_URL" | sed -n 's/.*:\([0-9]*\)\/.*/\1/p')
DB_NAME=$(echo "$DB_URL" | sed -n 's/.*\/\([^?]*\).*/\1/p')
DB_USER=$(echo "$DB_URL" | sed -n 's/.*:\/\/\([^:]*\):.*/\1/p')
DB_PASS=$(echo "$DB_URL" | sed -n 's/.*:\/\/[^:]*:\([^@]*\)@.*/\1/p')

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}    Yantra Active Tasks Shutdown${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Function to execute SQL
execute_sql() {
    local sql_file=$1
    PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$sql_file"
}

# Check if this is a dry run
if [ "$1" == "--check" ] || [ "$1" == "-c" ]; then
    echo -e "${YELLOW}📊 Checking active tasks (read-only)...${NC}"
    echo
    execute_sql "$SCRIPT_DIR/check_active_tasks.sql"
    echo
    echo -e "${GREEN}✅ Check complete${NC}"
    echo -e "${YELLOW}💡 To shutdown all tasks, run: $0 --shutdown${NC}"
    exit 0
fi

# Check if this is an actual shutdown
if [ "$1" == "--shutdown" ] || [ "$1" == "-s" ]; then
    echo -e "${YELLOW}⚠️  WARNING: This will cancel ALL active workflows and outbox messages!${NC}"
    echo -e "${YELLOW}   - All running workflow executions will be marked as cancelled${NC}"
    echo -e "${YELLOW}   - All pending email/Slack operations will be cancelled${NC}"
    echo -e "${YELLOW}   - This action cannot be undone${NC}"
    echo
    
    # Check active tasks first
    echo -e "${BLUE}📊 Current active tasks:${NC}"
    echo
    execute_sql "$SCRIPT_DIR/check_active_tasks.sql"
    echo
    
    # Confirm
    read -p "$(echo -e ${RED}Are you sure you want to proceed? [yes/NO]:${NC} )" -r
    echo
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo -e "${YELLOW}❌ Shutdown cancelled${NC}"
        exit 0
    fi
    
    echo -e "${RED}🛑 Shutting down all active tasks...${NC}"
    echo
    execute_sql "$SCRIPT_DIR/shutdown_all_active_tasks.sql"
    echo
    echo -e "${GREEN}✅ Shutdown complete!${NC}"
    exit 0
fi

# Show usage if no valid option provided
echo -e "${YELLOW}Usage:${NC}"
echo -e "  $0 --check      Check active tasks (read-only)"
echo -e "  $0 --shutdown   Shutdown all active tasks (requires confirmation)"
echo
echo -e "${YELLOW}Examples:${NC}"
echo -e "  # Check what's running"
echo -e "  $0 --check"
echo
echo -e "  # Emergency shutdown of all tasks"
echo -e "  $0 --shutdown"
echo
echo -e "${YELLOW}Environment:${NC}"
echo -e "  DATABASE_URL=${DB_URL}"
echo

