# Database Management Scripts

## Emergency Shutdown of Active Tasks

These scripts help you manage and shutdown all active workflow executions and outbox messages.

### Quick Start

#### 1. Check Active Tasks (Read-Only)

```bash
cd backend/scripts
./shutdown_tasks.sh --check
```

This will show you:
- Running/queued workflow executions
- Pending/processing outbox messages
- Active node executions
- Detailed breakdown of each running workflow

#### 2. Shutdown All Active Tasks

```bash
cd backend/scripts
./shutdown_tasks.sh --shutdown
```

This will:
- Show you what's currently active
- Ask for confirmation
- Cancel all running/queued workflows
- Cancel all pending/processing outbox messages
- Cancel all active node executions
- Show summary of what was cancelled

### Manual SQL Execution

If you prefer to run SQL directly:

#### Check Active Tasks

```bash
psql $DATABASE_URL -f check_active_tasks.sql
```

#### Shutdown All Tasks

```bash
psql $DATABASE_URL -f shutdown_all_active_tasks.sql
```

### Using Docker Database

If your database is running in Docker:

```bash
# Check tasks
docker exec -i your-postgres-container psql -U yantra -d yantra < check_active_tasks.sql

# Shutdown tasks
docker exec -i your-postgres-container psql -U yantra -d yantra < shutdown_all_active_tasks.sql
```

### What Gets Cancelled

When you run the shutdown script:

1. **Outbox Messages**: All `pending` and `processing` async operations (emails, Slack)
2. **Node Executions**: All `pending` and `running` node executions
3. **Workflow Executions**: All `running`, `queued`, and `interrupted` workflows

All cancelled items will have:
- Status set to `cancelled`
- Error message: `"System shutdown - all active tasks cancelled"`
- Completed timestamp set to now

### Use Cases

- **Emergency shutdown**: Server maintenance or emergency stop
- **Clean up stuck tasks**: Clear workflows that are hanging
- **Before upgrades**: Ensure clean state before deploying
- **Development reset**: Clear all active work during testing

### Safety

- The script requires explicit confirmation before proceeding
- Uses database transactions for atomicity
- Provides summary of what was cancelled
- Read-only check available first

### Troubleshooting

#### Permission Denied

```bash
chmod +x shutdown_tasks.sh
```

#### Database Connection Failed

Set your `DATABASE_URL` environment variable:

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/yantra?sslmode=disable"
./shutdown_tasks.sh --check
```

#### Custom Database Location

Edit the `DEFAULT_DB_URL` in `shutdown_tasks.sh` or set `DATABASE_URL`:

```bash
DATABASE_URL="postgresql://custom:pass@remote:5432/yantra" ./shutdown_tasks.sh --check
```

### Examples

#### Daily Maintenance

```bash
# Check what's running
./shutdown_tasks.sh --check

# If there are stuck tasks older than 1 hour, shutdown
./shutdown_tasks.sh --shutdown
```

#### Before Server Restart

```bash
# Gracefully cancel all tasks
./shutdown_tasks.sh --shutdown

# Stop server
systemctl stop yantra-server

# Upgrade/maintenance
...

# Start server
systemctl start yantra-server
```

#### Monitoring Script

```bash
# Create a monitoring cron job
*/5 * * * * cd /path/to/backend/scripts && ./shutdown_tasks.sh --check | mail -s "Yantra Active Tasks" admin@example.com
```

### Related Files

- `check_active_tasks.sql` - Read-only query to see active tasks
- `shutdown_all_active_tasks.sql` - Shutdown all active tasks
- `shutdown_tasks.sh` - Convenient shell wrapper

