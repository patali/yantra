# HTTP Node URL Templating Guide

## Overview

The HTTP node in Yantra supports template variables in URLs, headers, and body content. This allows you to dynamically construct API requests using data from previous nodes in your workflow.

## Template Syntax

Use the `{{variable.path}}` syntax to insert data from previous nodes:

```
{{input.field}}           # Access top-level field
{{input.user.email}}      # Access nested field
{{input.data.items.0}}    # Access array elements
```

## Using Templates in URLs

### 1. Query Parameters

You can use template variables in query parameters:

```
https://api.example.com/users?id={{input.userId}}&status={{input.status}}
```

**Example Input:**
```json
{
  "userId": 12345,
  "status": "active"
}
```

**Resulting URL:**
```
https://api.example.com/users?id=12345&status=active
```

### 2. Path Parameters

You can use template variables in the URL path:

```
https://api.example.com/users/{{input.userId}}/profile
```

**Example Input:**
```json
{
  "userId": 12345
}
```

**Resulting URL:**
```
https://api.example.com/users/12345/profile
```

### 3. Nested Data

Access nested data using dot notation:

```
https://api.example.com/search?query={{input.search.term}}&limit={{input.search.limit}}
```

**Example Input:**
```json
{
  "search": {
    "term": "workflow automation",
    "limit": 10
  }
}
```

**Resulting URL:**
```
https://api.example.com/search?query=workflow%20automation&limit=10
```

## Using Templates in Headers

Template variables work in header values:

```json
{
  "Authorization": "Bearer {{input.token}}",
  "X-User-ID": "{{input.userId}}",
  "X-Custom-Header": "{{input.metadata.customValue}}"
}
```

## Using Templates in Body

Template variables work in request bodies (both JSON and plain text):

### JSON Body
```json
{
  "name": "{{input.user.name}}",
  "email": "{{input.user.email}}",
  "role": "{{input.role}}"
}
```

### Plain Text Body
```
User: {{input.name}}
Email: {{input.email}}
Status: {{input.status}}
```

## Complete Example Workflow

Here's a complete example that fetches user data and sends it to an API:

```json
{
  "nodes": [
    {
      "id": "start",
      "type": "json",
      "config": {
        "data": {
          "userId": 12345,
          "token": "secret-token-123",
          "updates": {
            "name": "John Doe",
            "email": "john@example.com"
          }
        }
      }
    },
    {
      "id": "api-call",
      "type": "http",
      "config": {
        "url": "https://api.example.com/users/{{input.userId}}",
        "method": "PUT",
        "headers": {
          "Authorization": "Bearer {{input.token}}",
          "Content-Type": "application/json"
        },
        "body": {
          "name": "{{input.updates.name}}",
          "email": "{{input.updates.email}}"
        }
      }
    }
  ]
}
```

## Testing with Examples Endpoint

Yantra provides a public testing endpoint at `/api/examples/time` that:
- Returns current system time in multiple formats
- Echoes back any query parameters you send

### Test URL Templating

```
http://localhost:3000/api/examples/time?user={{input.userId}}&task={{input.taskId}}
```

The response will include your parameters in the `params` field:

```json
{
  "status": "success",
  "data": {
    "timestamp": 1701270123,
    "iso8601": "2025-11-29T14:15:23Z",
    "params": {
      "user": "12345",
      "task": "task-abc-123"
    }
  }
}
```

## Tips and Best Practices

1. **Missing Variables**: If a template variable is not found in the input data, the placeholder (`{{variable}}`) will remain unchanged in the final URL/body.

2. **URL Encoding**: Values are automatically converted to strings but are NOT automatically URL-encoded. For special characters, ensure proper encoding in your input data.

3. **Complex Data Types**: Only primitive values (strings, numbers, booleans) are supported in templates. Objects and arrays will be converted using `String()` representation.

4. **Security**: Never expose sensitive data (passwords, API keys) in URLs. Use headers for authentication tokens.

5. **Testing**: Use the `/api/examples/time` endpoint to test your URL templates before using them with real APIs.

## Error Handling

If a URL template fails to resolve:
- The HTTP node will use the URL as-is with unreplaced placeholders
- The API call may fail with a 400/404 error
- Check the execution logs to see the final URL that was used

## See Also

- [HTTP Node Documentation](../NODE_TYPES.md#http-node)
- [Template Engine Reference](TEMPLATE_ENGINE_QUICK_REFERENCE.md)
- [Example Workflows](../src/workflows/examples/)

