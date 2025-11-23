# API Reference

Yantra provides a REST API for managing and executing workflows.

## Base URL

- **Local Development**: `http://localhost:3000`
- **Docker**: `http://localhost:4701`
- **Production**: Your configured domain

## Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Workflows

### List Workflows

```http
GET /api/workflows
```

**Response:**
```json
{
  "workflows": [
    {
      "id": "uuid",
      "name": "My Workflow",
      "description": "Workflow description",
      "created_at": "2025-11-23T10:00:00Z",
      "updated_at": "2025-11-23T10:00:00Z"
    }
  ]
}
```

### Create Workflow

```http
POST /api/workflows
Content-Type: application/json

{
  "name": "My Workflow",
  "description": "Workflow description",
  "definition": {
    "nodes": [...],
    "edges": [...]
  }
}
```

### Get Workflow

```http
GET /api/workflows/:id
```

### Update Workflow

```http
PUT /api/workflows/:id
Content-Type: application/json

{
  "name": "Updated Name",
  "definition": {...}
}
```

### Delete Workflow

```http
DELETE /api/workflows/:id
```

### Duplicate Workflow

```http
POST /api/workflows/:id/duplicate
Content-Type: application/json

{
  "name": "Copy of Workflow"
}
```

## Execution

### Execute Workflow

```http
POST /api/workflows/:id/execute
Content-Type: application/json

{
  "input": {
    "data": "your input data"
  }
}
```

**Response:**
```json
{
  "execution_id": "uuid",
  "status": "running",
  "started_at": "2025-11-23T10:00:00Z"
}
```

### List Executions

```http
GET /api/workflows/:id/executions
```

**Query Parameters:**
- `limit`: Number of results (default: 50)
- `offset`: Pagination offset
- `status`: Filter by status (running, completed, failed)

### Get Execution Details

```http
GET /api/workflows/:id/executions/:executionId
```

**Response:**
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "status": "completed",
  "input": {...},
  "output": {...},
  "node_results": [...],
  "started_at": "2025-11-23T10:00:00Z",
  "completed_at": "2025-11-23T10:05:00Z"
}
```

### Stream Execution Updates (SSE)

```http
GET /api/workflows/:id/executions/:executionId/stream
```

Returns Server-Sent Events with real-time execution updates.

**Event Types:**
- `node_started`: Node execution begins
- `node_completed`: Node execution completes
- `node_failed`: Node execution fails
- `execution_completed`: Workflow completes
- `execution_failed`: Workflow fails
- `error`: Error occurred

### Resume Execution

```http
POST /api/workflows/:id/executions/:executionId/resume
```

Resume a failed or interrupted execution from the last checkpoint.

## Scheduling

### Update Workflow Schedule

```http
PUT /api/workflows/:id/schedule
Content-Type: application/json

{
  "enabled": true,
  "cron_expression": "0 9 * * *",
  "timezone": "America/New_York"
}
```

**Cron Expression Examples:**
- `0 9 * * *` - Daily at 9:00 AM
- `0 */6 * * *` - Every 6 hours
- `0 0 * * 0` - Weekly on Sunday at midnight
- `0 0 1 * *` - Monthly on the 1st at midnight

### Disable Schedule

```http
PUT /api/workflows/:id/schedule
Content-Type: application/json

{
  "enabled": false
}
```

## Webhooks

### Trigger Workflow via Webhook

```http
POST /api/webhooks/:workflowId
Content-Type: application/json

{
  "data": "your webhook payload"
}
```

### Trigger with Custom Path

```http
POST /api/webhooks/:workflowId/custom/path
```

### Regenerate Webhook Secret

```http
POST /api/workflows/:id/webhook-secret/regenerate
```

**Response:**
```json
{
  "webhook_secret": "new-secret-token",
  "webhook_url": "https://your-domain.com/api/webhooks/:workflowId"
}
```

## Versioning

### Get Version History

```http
GET /api/workflows/:id/versions
```

**Response:**
```json
{
  "versions": [
    {
      "version": 5,
      "created_at": "2025-11-23T10:00:00Z",
      "created_by": "user@example.com",
      "changes": "Updated node configuration"
    }
  ]
}
```

### Restore Version

```http
POST /api/workflows/:id/versions/restore
Content-Type: application/json

{
  "version": 3
}
```

## User Management

### Register User

```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password",
  "name": "John Doe"
}
```

### Login

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password"
}
```

**Response:**
```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

### Get Current User

```http
GET /api/auth/me
Authorization: Bearer <token>
```

## Health Check

### Server Health

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-23T10:00:00Z"
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {...}
}
```

**Common Status Codes:**
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## Rate Limiting

API endpoints are rate-limited to prevent abuse:

- **Default**: 100 requests per minute per IP
- **Authentication**: 10 attempts per minute per IP

Exceeded limits return `429 Too Many Requests`.

## Migration API (Production)

### Run Migrations

```http
POST /api/migration/run
X-Migration-Key: <your-migration-api-key>
```

Only available when `MIGRATION_API_KEY` is configured.

For more details, see the [Configuration Guide](./CONFIGURATION.md).

