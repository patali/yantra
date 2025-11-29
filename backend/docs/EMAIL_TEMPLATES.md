# Email Template Variables Guide

> 💡 **Want a complete working example?** Check out [EXAMPLE_HTTP_TO_EMAIL_REPORT.md](EXAMPLE_HTTP_TO_EMAIL_REPORT.md) - a full HTTP API → Loop → Email workflow.

## Overview

The email node supports powerful Go template engine features that allow you to dynamically insert data, iterate over lists, apply conditionals, and format output from previous workflow steps.

## Template Syntax

### Simple Variables (Legacy Support)

Use double curly braces to insert variables:

```
{{variable_name}}
```

### Advanced Templates (Go Template Engine)

For advanced features like loops and conditionals, use the dot notation:

```
{{.variable_name}}
```

**Available Features:**
- ✅ Variable interpolation
- ✅ Iterators/loops with `{{range}}`
- ✅ Conditionals with `{{if}}`, `{{else}}`
- ✅ Custom functions (`json`, `upper`, `lower`, etc.)
- ✅ Nested templates
- ✅ Pipeline operations

## Quick Start Examples

### Simple Variables (Backward Compatible)

```
Hello {{name}}, your order #{{orderId}} is ready!
```

### Advanced Templates

```
Hello {{.name}},

{{range .items}}
- {{.name}}: ${{.price}}
{{end}}

Total: ${{.total}}
```

## Common Use Cases

### 1. Simple Field Replacement

If your input data looks like:
```json
{
  "name": "John",
  "email": "john@example.com"
}
```

Use in email body:
```
Hello {{name}},

Your email is {{email}}.
```

Result:
```
Hello John,

Your email is john@example.com.
```

### 2. Loop Accumulator Results - Basic

If your input is from a **Loop Accumulator** node:
```json
{
  "accumulated": [
    {"index": 0, "name": "Leanne Graham"},
    {"index": 1, "name": "Ervin Howell"},
    {"index": 2, "name": "Clementine Bauch"}
  ],
  "iteration_count": 3
}
```

**Option A: Simple JSON dump**
```
Processing complete!

Results:
{{accumulated}}

Total items: {{iteration_count}}
```

**Option B: Beautiful formatting with iterators** ⭐ NEW!
```
Processing complete!

Processed Users:
{{range .accumulated}}
  {{.index}}. {{.name}}
{{end}}

Total: {{.iteration_count}} users
```

Result:
```
Processing complete!

Processed Users:
  0. Leanne Graham
  1. Ervin Howell
  2. Clementine Bauch

Total: 3 users
```

### 3. Nested Field Access

If your input has nested objects:
```json
{
  "data": {
    "user": {
      "name": "Alice",
      "address": {
        "city": "New York"
      }
    }
  }
}
```

Use dot notation:
```
Hello {{data.user.name}},

You live in {{data.user.address.city}}.
```

Result:
```
Hello Alice,

You live in New York.
```

### 4. HTTP Response Data

If your email follows an HTTP node:
```json
{
  "data": {
    "status": "success",
    "message": "Order placed"
  }
}
```

Access the nested data:
```
Status: {{data.status}}
Message: {{data.message}}
```

## Important Notes

### ❌ Common Mistakes

1. **Using the wrong path**
   - Wrong: `{{data.accumulated}}` (when accumulated is at the top level)
   - Right: `{{accumulated}}`

2. **Case sensitivity**
   - Wrong: `{{AccumulateD}}`
   - Right: `{{accumulated}}`

3. **Missing fields**
   - If a field doesn't exist, the template `{{missing_field}}` stays as-is in the email

### ✅ Best Practices

1. **Check your data structure first**
   - Look at the output from the previous node
   - Use the workflow execution logs to see the exact structure

2. **Use simple paths when possible**
   - `{{name}}` is better than `{{data.name}}` if `name` is at the top level

3. **Test with actual data**
   - Run your workflow and check the email
   - Verify the formatting looks good

## Data Type Formatting

### Strings and Numbers
Display as-is:
```
Name: {{name}}
Age: {{age}}
```

### Arrays and Objects
Automatically formatted as pretty-printed JSON:
```
Data:
{{accumulated}}
```

### Booleans
Display as `true` or `false`:
```
Active: {{isActive}}
```

## Examples by Node Type

### After Transform Node

Transform output:
```json
{
  "data": {
    "firstName": "John",
    "lastName": "Doe"
  }
}
```

Email template:
```
Hello {{data.firstName}} {{data.lastName}}!
```

### After Loop Accumulator Node

Loop output:
```json
{
  "accumulated": [...],
  "iteration_count": 10
}
```

Email template:
```
Processed {{iteration_count}} items:

{{accumulated}}
```

### After HTTP Node

HTTP output:
```json
{
  "data": {
    "users": [...]
  },
  "statusCode": 200
}
```

Email template:
```
API Response ({{data.statusCode}}):

{{data.users}}
```

## Real-World Template Examples

### E-commerce Order Summary

```
🛍️ Order Confirmation

Hi {{.customer.name | title}},

Your order #{{.orderId}} has been confirmed!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ORDER DETAILS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{{range .items}}
{{.name}}
  Qty: {{.quantity}} × ${{.unitPrice}} = ${{.totalPrice}}
{{end}}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{{if .discount}}
Subtotal:     ${{.subtotal}}
Discount:    -${{.discount}}
{{end}}
Tax:          ${{.tax}}
Shipping:     ${{.shipping}}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TOTAL:        ${{.total}}

{{if .trackingNumber}}
📦 Tracking: {{.trackingNumber}}
{{else}}
We'll send you tracking info once shipped.
{{end}}

Questions? Reply to this email!
```

### Data Processing Report

```
📊 Daily Processing Report - {{.date}}

{{range .workflows}}
Workflow: {{.name}}
  Status: {{.status | upper}}
  Started: {{.startTime}}
  Duration: {{.duration}}
  
  {{if eq .status "success"}}
  ✅ Completed successfully
  Records processed: {{.recordCount}}
  {{else if eq .status "error"}}
  ❌ Failed: {{.error}}
  {{else}}
  ⏳ In progress...
  {{end}}

{{end}}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary:
- Total workflows: {{.totalCount}}
- Successful: {{.successCount}}
- Failed: {{.failedCount}}
```

### Weekly User Activity Summary

```
📈 Weekly Activity Report

Hello {{.adminName}},

Here's your weekly user activity summary:

TOP ACTIVE USERS
{{range .topUsers}}
{{.rank}}. {{.name}} ({{.email}})
   - Logins: {{.loginCount}}
   - Actions: {{.actionCount}}
{{end}}

NEW USERS THIS WEEK
{{range .newUsers}}
• {{.name}} - Joined {{.joinDate}}
{{else}}
No new users this week.
{{end}}

USAGE STATISTICS
Total Active Users: {{.activeUserCount}}
Total Actions: {{.totalActions}}
Average per User: {{.avgActions}}

{{if gt .errorCount 0}}
⚠️ Errors Detected: {{.errorCount}}
Please review the error log.
{{end}}
```

## Troubleshooting

### Problem: Getting `{{variable}}` literally in email

**Cause**: The variable doesn't exist in the input data or the path is wrong.

**Solution**:
1. Check the previous node's output
2. Verify the exact field name (case-sensitive)
3. For template engine features, use `{{.variable}}` with a dot
4. For simple replacement, `{{variable}}` without dot works too

### Problem: Template engine not activating

**Cause**: Not using template engine syntax.

**Solution**: 
- Use `{{.variable}}` with dot prefix, OR
- Use template keywords like `{{range}}`, `{{if}}`
- The engine auto-activates when it detects these patterns

### Problem: "executing template" error

**Cause**: Invalid template syntax.

**Solution**:
1. Make sure every `{{range}}` has a matching `{{end}}`
2. Make sure every `{{if}}` has a matching `{{end}}`
3. Check for typos in template keywords
4. Test your template with simple data first

### Problem: Array showing as ugly Go format

**Cause**: Using simple syntax without template engine.

**Solution**: 
- Use `{{range .array}}` to iterate and format each item, OR
- Use `{{json .array}}` function for pretty JSON output
- Simple `{{array}}` now outputs JSON by default

### Problem: Nested field not found

**Cause**: Using wrong path or field doesn't exist.

**Solution**:
1. Template engine: `{{.data.user.name}}`
2. Simple syntax: `{{data.user.name}}`
3. Check workflow execution logs for exact structure
4. Use `{{json .}}` to see the full data structure

## Advanced Template Engine Features ⭐ NEW

### Iterators with {{range}}

Loop through arrays to format each item:

```
User Report:

{{range .accumulated}}
━━━━━━━━━━━━━━━━━━━━━━━
Name: {{.name}}
Email: {{.email}}
Status: {{.status}}
{{end}}
━━━━━━━━━━━━━━━━━━━━━━━
Total: {{.iteration_count}} users processed
```

### Conditionals with {{if}}

Display content based on conditions:

```
Hello {{.name}},

{{if .isPremium}}
✨ Thank you for being a Premium member!
You have access to all exclusive features.
{{else}}
Upgrade to Premium to unlock exclusive features!
{{end}}

{{if .hasOrders}}
You have {{.orderCount}} pending orders.
{{end}}
```

### Nested Ranges

Iterate through nested data structures:

```
Order Summary:

{{range .customers}}
Customer: {{.name}}
  Orders:
{{range .orders}}
  - Order #{{.id}}: ${{.amount}}
    Status: {{.status}}
{{end}}
{{end}}
```

### Custom Functions

#### `json` - Pretty-print JSON

```
Debug Data:
{{json .debugInfo}}
```

#### `jsonCompact` - Compact JSON (single line)

```
Data: {{jsonCompact .data}}
```

#### `upper` - Convert to uppercase

```
IMPORTANT: {{.message | upper}}
```

#### `lower` - Convert to lowercase

```
Email: {{.email | lower}}
```

#### `title` - Title case

```
Name: {{.firstName | title}} {{.lastName | title}}
```

### Pipeline Operations

Chain multiple operations:

```
{{.userName | lower | title}}
```

### Complex Example: Order Confirmation

```
Order Confirmation #{{.orderId}}

Dear {{.customerName | title}},

Thank you for your order!

Items:
{{range .items}}
  • {{.name}} - Quantity: {{.quantity}} - Price: ${{.price}}
{{end}}

{{if .hasDiscount}}
Discount Applied: -${{.discount}}
{{end}}

Subtotal: ${{.subtotal}}
Tax: ${{.tax}}
━━━━━━━━━━━━━━━━━━━━━━━
TOTAL: ${{.total}}

{{if .isPriority}}
⚡ Priority Shipping - Arrives by {{.deliveryDate}}
{{else}}
Standard Shipping - Arrives in 5-7 business days
{{end}}

Best regards,
The Team
```

### Empty Range Handling

Handle empty arrays gracefully:

```
{{range .items}}
- {{.name}}
{{else}}
No items found.
{{end}}
```

### Index and Value in Range

Access both index and value:

```
{{range $index, $item := .users}}
{{$index}}. {{$item.name}}
{{end}}
```

## Backward Compatibility

### Legacy Simple Syntax

The old syntax without dots still works:

```
Hello {{name}}, your order {{orderId}} is ready!
```

### New Template Engine Syntax

For advanced features, use dot notation:

```
Hello {{.name}},

{{range .orders}}
Order #{{.id}} - {{.status}}
{{end}}
```

**Rule:** If your template contains `{{range}}`, `{{if}}`, or `{{.variable}}` (with dot), the template engine is automatically activated.

## Advanced Usage

### Multiple Variables in Subject

```
Subject: Order #{{orderId}} - Status: {{status}}
```

Or with template engine:

```
Subject: Order #{{.orderId}} - {{.status | upper}}
```

### Combining Static Text and Variables

```
Dear {{firstName}},

Your account ({{accountId}}) has been updated.

Details:
{{accountDetails}}

Best regards,
The Team
```

### Working with Loop Output

**Simple version:**
```
Loop Processing Report

Completed: {{iteration_count}} iterations

Successful items:
{{accumulated}}
```

**Formatted version with template engine:**
```
Loop Processing Report

Completed: {{.iteration_count}} iterations

Processed Items:
{{range .accumulated}}
✓ Item {{.index}}: {{.name}} - {{.status}}
{{end}}

━━━━━━━━━━━━━━━━━━━━━━━
Summary: {{.iteration_count}} items successfully processed
```

