# Conditional Node Documentation

## Overview

The Conditional node evaluates boolean conditions and outputs `true` or `false`. It supports both a simple string format and a structured format (used by the frontend UI).

## Structured Format (Recommended)

The frontend UI uses a structured format that makes it easy to build conditions visually:

```json
{
  "conditions": [
    {
      "left": "data.success",
      "operator": "eq",
      "right": "true"
    }
  ],
  "logicalOperator": "AND"
}
```

### Supported Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `data.status` == `"success"` |
| `neq` | Not Equals | `data.status` != `"failed"` |
| `gt` | Greater Than | `data.count` > `5` |
| `lt` | Less Than | `data.count` < `10` |
| `gte` | Greater or Equal | `data.age` >= `18` |
| `lte` | Less or Equal | `data.age` <= `65` |
| `exists` | Field Exists | `data.userId` != `nil` |
| `contains` | String Contains | `contains(data.message, "error")` |

### Value Types

The right-hand value is automatically typed based on its content:

- **Booleans**: `true`, `false` (no quotes)
- **Numbers**: `123`, `45.67` (no quotes)
- **Strings**: `"hello"`, `"active"` (with quotes)
- **Field References**: `data.otherField`, `workflow.previousNode.data` (no quotes)

### Multiple Conditions

Use the `logicalOperator` field to combine multiple conditions:

**AND** (all conditions must be true):
```json
{
  "conditions": [
    {"left": "data.age", "operator": "gte", "right": "18"},
    {"left": "data.status", "operator": "eq", "right": "active"}
  ],
  "logicalOperator": "AND"
}
```

**OR** (at least one condition must be true):
```json
{
  "conditions": [
    {"left": "data.isAdmin", "operator": "eq", "right": "true"},
    {"left": "data.isModerator", "operator": "eq", "right": "true"}
  ],
  "logicalOperator": "OR"
}
```

## Legacy String Format

You can also provide a condition as a raw gval expression string:

```json
{
  "condition": "data.success == true && data.count > 5"
}
```

This format is supported for backward compatibility and advanced use cases.

## Data Access

The conditional node has access to:

1. **Input data**: Access via `input.field` or directly via `field` (for nested data)
2. **Workflow data**: Access via `workflow.nodeId.data` or directly via `nodeId` (for previous node outputs)
3. **Nested data**: The executor automatically flattens `input.data.*` to the root level

### Example Data Access

If the input is:
```json
{
  "data": {
    "success": true,
    "count": 10,
    "user": {
      "id": "123",
      "name": "John"
    }
  }
}
```

You can access:
- `success` or `input.data.success` → `true`
- `count` or `input.data.count` → `10`
- `user` or `input.data.user` → `{"id": "123", "name": "John"}`

## Output

The conditional node outputs:
```json
{
  "data": true,      // Boolean result of the condition
  "result": true,    // Same as data (for backward compatibility)
  "condition": "...", // The evaluated condition string
  "input": {...}     // Pass-through of input data for easier access in branches
}
```

The `input` field contains the complete input data that was passed to the conditional node. This makes it easier to access the original data in downstream branches without having to reference the original source node.

## Using Conditional Results in Edges

**IMPORTANT**: To create branching paths based on conditional results, you **must** add edge conditions. Without edge conditions, **all connected nodes will execute**.

**✨ NEW: The UI now automatically adds edge conditions when you connect from conditional nodes!** When you drag a connection from the "true" or "false" handle of a conditional node, the appropriate condition is automatically added to the edge.

### How Edge Conditions Work

When the workflow engine executes a conditional node:
1. The conditional node evaluates and outputs `{data: true}` or `{data: false}`
2. The engine checks **each outgoing edge** for a `condition` field
3. If an edge has a condition, it's evaluated using the conditional node's output
4. Only edges where the condition evaluates to `true` will execute

### Auto-Generated Edge Conditions

When you connect edges from a conditional node in the workflow editor:

- **From "true" handle** → Automatically adds: `condition: "data.data == true"`
- **From "false" handle** → Automatically adds: `condition: "data.data == false"`

You can manually edit these conditions in the workflow JSON if you need different logic.

### Edge Condition Syntax

Edge conditions use the same gval expression syntax as conditional nodes:

```json
{
  "edges": [
    {
      "id": "e1",
      "source": "conditional-1",
      "target": "success-handler",
      "condition": "data.data === true",
      "label": "true"
    },
    {
      "id": "e2",
      "source": "conditional-1", 
      "target": "failure-handler",
      "condition": "data.data === false",
      "label": "false"
    }
  ]
}
```

### Edge Condition Context

Edge conditions have access to:
- `data` - The source node's output (e.g., `data.data` for conditional result)
- `nodeId` - Direct access to any node's output (e.g., `conditional-1.data`)
- All fields from the source node's output at root level

**Example:**
If conditional node outputs `{data: true, result: true, condition: "...", input: {...}}`:
- Use `data.data === true` to check the boolean result
- Use `data.result === true` (backward compatible)
- Use `data.condition` to access the condition string
- Use `data.input` to access the original input data passed to the conditional node

### Common Edge Condition Patterns

**True/False Branching:**
```json
{
  "edges": [
    {"source": "conditional-1", "target": "on-true", "condition": "data.data === true"},
    {"source": "conditional-1", "target": "on-false", "condition": "data.data === false"}
  ]
}
```

**Multi-way Branching:**
```json
{
  "edges": [
    {"source": "conditional-1", "target": "path-a", "condition": "data.value > 100"},
    {"source": "conditional-1", "target": "path-b", "condition": "data.value <= 100 && data.value > 0"},
    {"source": "conditional-1", "target": "path-c", "condition": "data.value <= 0"}
  ]
}
```

**No Condition = Always Execute:**
```json
{
  "edges": [
    {"source": "node-1", "target": "node-2"}  // No condition - always executes
  ]
}
```

## Common Errors

### Error: "condition not specified"

**Cause**: The node configuration doesn't contain either:
- A `condition` string, OR
- A valid `conditions` array

**Solution**: Ensure you're providing conditions in the correct format:
```json
{
  "conditions": [
    {"left": "data.field", "operator": "eq", "right": "value"}
  ],
  "logicalOperator": "AND"
}
```

### Error: "condition must evaluate to boolean"

**Cause**: Your condition evaluates to a non-boolean value (e.g., a number or string).

**Solution**: Ensure your condition uses comparison operators:
- ❌ `data.count` → Returns a number
- ✅ `data.count > 5` → Returns boolean

### Error: "failed to evaluate condition"

**Cause**: The condition syntax is invalid or references non-existent fields.

**Solution**: 
- Check field names match your input data
- Verify operator syntax is correct
- Use the structured format for better error handling

## Examples

### Example 1: Simple Success Check
```json
{
  "conditions": [
    {"left": "data.success", "operator": "eq", "right": "true"}
  ],
  "logicalOperator": "AND"
}
```

### Example 2: Range Check
```json
{
  "conditions": [
    {"left": "data.temperature", "operator": "gte", "right": "20"},
    {"left": "data.temperature", "operator": "lte", "right": "25"}
  ],
  "logicalOperator": "AND"
}
```

### Example 3: Field Existence
```json
{
  "conditions": [
    {"left": "data.userId", "operator": "exists"}
  ],
  "logicalOperator": "AND"
}
```

### Example 4: String Contains
```json
{
  "conditions": [
    {"left": "data.message", "operator": "contains", "right": "error"}
  ],
  "logicalOperator": "AND"
}
```

### Example 5: Multiple OR Conditions
```json
{
  "conditions": [
    {"left": "data.priority", "operator": "eq", "right": "high"},
    {"left": "data.priority", "operator": "eq", "right": "urgent"}
  ],
  "logicalOperator": "OR"
}
```

### Example 6: Using Input Pass-Through in Branches

The conditional node now passes through its input, making it easier to access data in downstream branches:

**Workflow Structure:**
```
HTTP Node → Conditional Node → [True Branch] → Email Node
                             → [False Branch] → Slack Node
```

**In the Email Node (true branch):**
You can access the HTTP response data via the conditional node's output:
```json
{
  "type": "email",
  "config": {
    "subject": "High Count Alert",
    "body": "Count is {{conditional-1.input.data.count}} - threshold exceeded!"
  }
}
```

**Alternative (referencing original source):**
```json
{
  "body": "Count is {{http-1.data.count}} - threshold exceeded!"
}
```

Both approaches work, but using `conditional-1.input` is more convenient when you have complex workflow paths or when the original source node is far away in the workflow graph.

## Testing

See `conditional_test.go` for comprehensive test examples covering both string and structured formats.

