# Node Types Reference

Yantra provides 14 node types for building workflows. All nodes follow a standardized input/output format.

## Node Categories

| Category | Node Type | Description |
|----------|-----------|-------------|
| **Control** | `start` | Workflow entry point |
| | `end` | Workflow termination |
| | `conditional` | Boolean branching logic |
| | `delay` | Time-based pauses (milliseconds) |
| | `sleep` | Long-term delays (days/weeks/specific dates) |
| **Data** | `json` | Static/dynamic JSON data |
| | `json-array` | Arrays with schema validation |
| | `transform` | Map, extract, parse, stringify, concat |
| | `json_to_csv` | Convert JSON to CSV |
| **Iteration** | `loop` | Iterate over arrays |
| | `loop-accumulator` | Collect iteration results |
| **Integration** | `http` | HTTP/REST API calls |
| | `email` | Email with templates |
| | `slack` | Slack notifications |

## Output Format Standard

Every node returns an output object that **always includes a `data` field** containing the primary result:

```json
{
  "data": <primary_output>,
  // Additional metadata fields...
}
```

## Node-Specific Details

### Control Nodes

#### Start Node
- **Purpose**: Entry point for workflow execution
- **Output**: Initial workflow input data

#### End Node
- **Purpose**: Marks successful workflow completion
- **Output**: Final workflow result

#### Conditional Node
- **Purpose**: Boolean branching logic
- **Configuration**: JavaScript expression
- **Output**: 
  ```json
  {
    "data": true,
    "result": true,
    "condition": "x > 5"
  }
  ```

#### Delay Node
- **Purpose**: Pause execution for milliseconds
- **Configuration**: Duration in milliseconds
- **Output**:
  ```json
  {
    "data": 1000,
    "delayed_ms": 1000
  }
  ```

#### Sleep Node
- **Purpose**: Long-term delays (days/weeks) without blocking workers
- **Configuration**: 
  - Relative: `{ "mode": "relative", "duration_value": 7, "duration_unit": "days" }`
  - Absolute: `{ "mode": "absolute", "target_date": "2025-12-25T10:00:00Z" }`
- **Output**:
  ```json
  {
    "data": "2025-12-25T10:00:00Z",
    "sleep_scheduled_until": "2025-12-25T10:00:00Z",
    "mode": "absolute"
  }
  ```

### Data Nodes

#### JSON Node
- **Purpose**: Provide static or dynamic JSON data
- **Output**:
  ```json
  {
    "data": { "name": "John", "age": 30 }
  }
  ```

#### Transform Node
- **Purpose**: Map, extract, parse, stringify, or concatenate data
- **Operations**: extract, map, parse, stringify, concat
- **Output**:
  ```json
  {
    "data": { "firstName": "John" }
  }
  ```

#### JSON Array Node
- **Purpose**: Arrays with schema validation
- **Output**:
  ```json
  {
    "data": [{ "id": 1 }, { "id": 2 }],
    "count": 2,
    "schema": { "type": "object" }
  }
  ```

#### JSON to CSV Node
- **Purpose**: Convert JSON arrays to CSV format
- **Output**:
  ```json
  {
    "data": "name,age\nJohn,30",
    "row_count": 1,
    "headers": ["name", "age"]
  }
  ```

### Iteration Nodes

#### Loop Node
- **Purpose**: Iterate over arrays
- **Configuration**: Array source and iteration logic
- **Output**:
  ```json
  {
    "data": [{ "result": 1 }, { "result": 2 }],
    "iteration_count": 2,
    "items": [...]
  }
  ```

#### Loop Accumulator Node
- **Purpose**: Collect and accumulate results from loop iterations
- **Configuration**: Accumulation mode (default: "array")
- **Output**:
  ```json
  {
    "data": [1, 2, 3, 4, 5],
    "iteration_count": 5,
    "accumulationMode": "array"
  }
  ```

### Integration Nodes

#### HTTP Node
- **Purpose**: Make HTTP/REST API calls
- **Configuration**: URL, method, headers, body
- **Output**:
  ```json
  {
    "data": { "response": "body" },
    "status_code": 200,
    "headers": { "content-type": "application/json" },
    "url": "https://api.example.com",
    "method": "GET"
  }
  ```

#### Email Node
- **Purpose**: Send emails with templates
- **Configuration**: SMTP settings, recipients, subject, body
- **Output**:
  ```json
  {
    "data": true,
    "sent": true,
    "messageId": "abc123"
  }
  ```
- **Note**: Uses outbox pattern for reliability

#### Slack Node
- **Purpose**: Send Slack notifications
- **Configuration**: Webhook URL, channel, message
- **Output**:
  ```json
  {
    "data": true,
    "sent": true,
    "channel": "#general",
    "statusCode": 200
  }
  ```
- **Note**: Uses outbox pattern for reliability

## Accessing Node Outputs

### In Conditional Nodes

```javascript
// Access previous node output
nodeId.data > 10

// Access nested data
nodeId.data.users.length > 0
```

### In HTTP Request Bodies

```json
{
  "userId": "{{nodeId.data.id}}",
  "items": "{{loopNode.data}}"
}
```

### In Transform Operations

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

## Resource Limits

To protect system resources, the following limits are enforced:

- **Maximum execution duration**: 30 minutes
- **Maximum loop iterations**: 10,000
- **Maximum data size**: 10MB
- **Nested loop depth limit**: Enforced

## Backward Compatibility

Existing workflows maintain original field names alongside the `data` field:
- `conditional`: Still includes `result` field
- `json-array`: Still includes `array` field
- `loop`: Still includes `results` field

For more details on specific nodes, see the [Backend Documentation](../backend/README.md).

