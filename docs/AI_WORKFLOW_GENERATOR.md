# AI Workflow Generator

The AI Workflow Generator allows you to create workflows using natural language descriptions. Instead of manually building workflows node-by-node, simply describe what you want to accomplish and let AI generate the complete workflow definition.

## Table of Contents

- [Overview](#overview)
- [Setup](#setup)
- [API Usage](#api-usage)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Limitations](#limitations)

## Overview

The AI Workflow Generator uses Large Language Models (LLMs) to translate natural language descriptions into valid Yantra workflow definitions. It understands all 14 node types and can create complex workflows with:

- HTTP API integrations
- Data transformations
- Conditional branching
- Loops and iterations
- Email and Slack notifications
- Scheduled executions
- Webhook triggers

## Setup

### Environment Variables

Configure the AI service using environment variables:

```bash
# Required: OpenAI API Key
OPENAI_API_KEY=sk-your-api-key-here

# Optional: Custom AI provider base URL
AI_API_BASE_URL=https://api.openai.com/v1

# Optional: Model selection (default: gpt-4o-mini)
AI_MODEL=gpt-4o-mini
```

### Supported Models

- **gpt-4o-mini** (default) - Cost-effective, fast, good for most workflows
- **gpt-4o** - More powerful, better for complex workflows
- **gpt-4-turbo** - Balance of speed and capability

### Alternative AI Providers

You can use OpenAI-compatible APIs:

```bash
# Anthropic Claude via proxy
AI_API_BASE_URL=https://your-proxy.com/v1
AI_API_KEY=your-api-key

# Azure OpenAI
AI_API_BASE_URL=https://your-resource.openai.azure.com
AI_API_KEY=your-azure-key
AI_MODEL=gpt-4
```

## API Usage

### Endpoint

```
POST /api/workflows/generate
```

### Authentication

Requires JWT authentication token in Authorization header.

### Request Format

```json
{
  "description": "Your workflow description in natural language",
  "context": {
    "optional": "additional context",
    "apiEndpoint": "https://api.example.com",
    "emailRecipient": "user@example.com"
  }
}
```

**Fields:**
- `description` (required): Natural language description of what the workflow should do
- `context` (optional): Additional context or parameters to guide generation

### Response Format

```json
{
  "success": true,
  "data": {
    "name": "Generated Workflow Name",
    "description": "What the workflow does",
    "explanation": "Brief explanation of the workflow logic",
    "workflow": {
      "nodes": [...],
      "edges": [...]
    }
  }
}
```

### Using the Generated Workflow

The `workflow` object can be used directly in the Create Workflow API:

```bash
# 1. Generate the workflow
curl -X POST https://your-domain.com/api/workflows/generate \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Fetch users from JSONPlaceholder API and send summary email"
  }'

# 2. Extract the workflow definition from response and create it
curl -X POST https://your-domain.com/api/workflows \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User Summary Email",
    "description": "Fetches users and sends email",
    "definition": {
      "nodes": [...],
      "edges": [...]
    }
  }'
```

## Examples

### Example 1: Simple HTTP to Email

**Request:**
```json
{
  "description": "Fetch the latest post from JSONPlaceholder API and email it to admin@example.com"
}
```

**Generated Workflow:**
- Start node (manual trigger)
- HTTP GET to https://jsonplaceholder.typicode.com/posts/1
- Transform node to extract title and body
- Email node with template
- End node

### Example 2: Webhook with Conditional Logic

**Request:**
```json
{
  "description": "Create a webhook that receives order data. If the order amount is greater than 1000, send an email to sales@example.com, otherwise send a Slack message to #orders channel"
}
```

**Generated Workflow:**
- Start node (webhook trigger)
- Transform node to extract order amount
- Conditional node (amount > 1000)
- Email node (true branch)
- Slack node (false branch)
- End node

### Example 3: Scheduled Data Pipeline

**Request:**
```json
{
  "description": "Every day at 9 AM, fetch users from https://api.example.com/users, loop through them, transform each user to extract name and email, and send a daily report email with all users",
  "context": {
    "timezone": "America/New_York",
    "reportRecipient": "reports@example.com"
  }
}
```

**Generated Workflow:**
- Start node (cron: "0 9 * * *", timezone: America/New_York)
- HTTP GET to https://api.example.com/users
- Loop accumulator node
- Transform node (extract name, email)
- Email node with template showing all accumulated users
- End node

### Example 4: API Integration with Error Handling

**Request:**
```json
{
  "description": "Call weather API for New York, if successful send weather to Slack, if it fails send error email to ops@example.com"
}
```

**Generated Workflow:**
- Start node
- HTTP GET to weather API
- Conditional node (check status_code == 200)
- Slack node (success branch)
- Email node (failure branch)
- End node

### Example 5: Data Processing Pipeline

**Request:**
```json
{
  "description": "Fetch customer orders from API, filter only orders with status 'pending', loop through each order and send reminder email to customer"
}
```

**Generated Workflow:**
- Start node
- HTTP GET orders
- Transform to filter pending orders
- Loop accumulator
- Transform to extract customer email and order details
- Email node with order reminder template
- End node

## Best Practices

### Writing Effective Descriptions

**Be Specific:**
```
❌ "Send some emails"
✅ "Fetch users from API and send welcome email to each user"
```

**Include Trigger Type:**
```
❌ "Process webhook data"
✅ "Create a webhook at /orders that processes order data"
```

**Specify Conditions Clearly:**
```
❌ "Do something based on status"
✅ "If status is 'completed', send success email, otherwise send Slack alert"
```

**Mention Data Sources:**
```
❌ "Get data and transform it"
✅ "Fetch users from https://api.example.com/users and extract name, email, company fields"
```

### Using Context Parameter

Use the `context` field for:
- API endpoints
- Email addresses
- Slack webhook URLs
- Cron schedules
- Timezone preferences
- Field names or data structures

```json
{
  "description": "Daily report of new users",
  "context": {
    "apiEndpoint": "https://api.example.com/users",
    "schedule": "0 8 * * *",
    "timezone": "America/Los_Angeles",
    "recipient": "reports@example.com"
  }
}
```

### Iterative Refinement

Start simple and iterate:

1. Generate basic workflow
2. Test and review
3. Regenerate with more specific description
4. Manual adjustments as needed

## Limitations

### Current Limitations

1. **Node Type Coverage**: Supports all 14 node types, but complex configurations may need manual adjustment

2. **Template Complexity**: Simple templates work well, complex Go templates may need review

3. **Error Handling**: Generated workflows include basic error handling via conditionals, not comprehensive error recovery

4. **Nested Loops**: Can generate nested loops but complex loop logic may need manual refinement

5. **Authentication**: API authentication (headers, OAuth) needs to be specified in description or added manually

### When to Use Manual Creation

Consider manual workflow creation for:
- Very complex multi-step transformations
- Custom business logic requiring precise conditions
- Workflows with strict compliance requirements
- Integration with internal systems requiring specific configurations

### Validation

Always validate generated workflows:

1. **Review Node Configuration**: Check API URLs, email addresses, schedules
2. **Test Conditions**: Verify conditional logic matches requirements
3. **Check Data Flow**: Ensure data flows correctly between nodes
4. **Validate Templates**: Test email/Slack templates with sample data
5. **Security Review**: Verify no sensitive data in configurations

## Advanced Usage

### Combining AI Generation with Manual Editing

Best workflow:
1. Use AI to generate the workflow structure
2. Review and test
3. Manually adjust node configurations as needed
4. Save and version

### Custom Node Configurations

If AI-generated configurations need adjustment:

**HTTP Node Authentication:**
```json
{
  "type": "http",
  "data": {
    "config": {
      "url": "https://api.example.com/data",
      "headers": {
        "Authorization": "Bearer {{start-1.data.token}}",
        "X-API-Key": "your-api-key"
      }
    }
  }
}
```

**Email with HTML Templates:**
```json
{
  "type": "email",
  "data": {
    "config": {
      "body": "<html><body><h1>{{.title}}</h1><p>{{.content}}</p></body></html>",
      "isHtml": true
    }
  }
}
```

## Troubleshooting

### Common Issues

**"AI API key not configured"**
- Set `OPENAI_API_KEY` environment variable
- Restart the server after setting environment variables

**"Failed to parse generated workflow"**
- Try a more specific description
- Use the `context` field for additional details
- Check AI model settings

**"Generated workflow is missing 'start' node"**
- Regenerate with explicit mention of trigger type
- Example: "Create a manual workflow that..."

**Invalid Conditional Logic**
- Be explicit about conditions
- Example: "If the status field equals 'active' then..."

## API Reference

### Generate Workflow

**Endpoint:** `POST /api/workflows/generate`

**Request Body:**
```typescript
{
  description: string;      // Required: Natural language description
  context?: {               // Optional: Additional context
    [key: string]: any;
  };
}
```

**Response:**
```typescript
{
  success: boolean;
  data: {
    name: string;           // Generated workflow name
    description: string;    // Generated description
    explanation?: string;   // How the workflow works
    workflow: {
      nodes: Array<Node>;   // Workflow nodes
      edges: Array<Edge>;   // Node connections
    };
  };
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request format
- `500 Internal Server Error`: AI generation failed or API error

## Examples Repository

For more examples, see:
- `/backend/src/workflows/examples/` - Production workflow examples
- `/backend/src/workflows/testdata/workflows/` - Test workflow examples

These can serve as inspiration for what kinds of workflows can be generated.

## Support

For issues or questions:
1. Check this documentation
2. Review example workflows
3. Consult the main [NODE_TYPES.md](./NODE_TYPES.md) documentation
4. Open an issue in the repository
