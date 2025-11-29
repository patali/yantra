# Email Template Engine - Quick Reference

## Syntax Modes

### Simple Mode (Backward Compatible)
```
Hello {{name}}, order {{orderId}} is ready!
```
- No dot prefix needed
- Direct variable replacement
- Works for simple use cases

### Template Engine Mode
```
Hello {{.name}}, 

{{range .orders}}
- Order #{{.id}}
{{end}}
```
- Use dot prefix: `{{.variable}}`
- Unlocks advanced features
- Auto-activates with `{{range}}`, `{{if}}`, etc.

## Core Features

### Variables
```
{{.name}}
{{.user.email}}
{{.data.items.0.price}}
```

### Loops
```
{{range .items}}
- {{.name}}: ${{.price}}
{{end}}
```

### Conditionals
```
{{if .isActive}}
Active
{{else}}
Inactive
{{end}}
```

### Empty Check
```
{{range .items}}
- {{.name}}
{{else}}
No items found
{{end}}
```

### Equality Check
```
{{if eq .status "success"}}
✅ Success
{{else if eq .status "error"}}
❌ Error
{{else}}
⏳ Processing
{{end}}
```

### Greater Than / Less Than
```
{{if gt .count 10}}
More than 10 items
{{end}}

{{if lt .age 18}}
Under 18
{{end}}
```

## Built-in Functions

### json - Pretty JSON
```
{{json .data}}
```

### jsonCompact - Compact JSON
```
{{jsonCompact .data}}
```

### upper - Uppercase
```
{{.text | upper}}
```

### lower - Lowercase
```
{{.email | lower}}
```

### title - Title Case
```
{{.name | title}}
```

### add - Addition
```
{{add .index 1}}
```

### sub - Subtraction
```
{{sub .total .discount}}
```

### mul - Multiplication
```
{{mul .quantity .price}}
```

### div - Division
```
{{div .total .count}}
```

## Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `{{if eq .status "active"}}` |
| `ne` | Not equal | `{{if ne .count 0}}` |
| `lt` | Less than | `{{if lt .age 18}}` |
| `le` | Less or equal | `{{if le .score 100}}` |
| `gt` | Greater than | `{{if gt .count 5}}` |
| `ge` | Greater or equal | `{{if ge .total 1000}}` |

## Common Patterns

### List with Numbers
```
{{range $index, $item := .items}}
{{add $index 1}}. {{$item.name}}
{{end}}
```

### Conditional in Loop
```
{{range .users}}
{{if .isActive}}
✓ {{.name}} (Active)
{{end}}
{{end}}
```

### Nested Objects
```
{{range .departments}}
Department: {{.name}}
{{range .employees}}
  - {{.name}} ({{.role}})
{{end}}
{{end}}
```

### Table Format
```
| Name          | Status   | Count |
|---------------|----------|-------|
{{range .items}}
| {{.name}}     | {{.status}} | {{.count}} |
{{end}}
```

### Numbered List with Details
```
{{range $i, $user := .users}}
━━━━━━━━━━━━━━━━━━━━━━━
User #{{add $i 1}}
━━━━━━━━━━━━━━━━━━━━━━━
Name: {{$user.name}}
Email: {{$user.email}}
{{if $user.isPremium}}
Status: ⭐ Premium Member
{{else}}
Status: Regular Member
{{end}}
{{end}}
```

## Loop Accumulator Examples

### Input from Loop Accumulator:
```json
{
  "accumulated": [
    {"index": 0, "name": "John", "status": "success"},
    {"index": 1, "name": "Jane", "status": "success"},
    {"index": 2, "name": "Bob", "status": "failed"}
  ],
  "iteration_count": 3
}
```

### Template Options:

**Option 1: Simple list**
```
{{range .accumulated}}
• {{.name}}: {{.status}}
{{end}}
```

**Option 2: Numbered with status icons**
```
{{range .accumulated}}
{{.index}}. {{.name}} {{if eq .status "success"}}✅{{else}}❌{{end}}
{{end}}
```

**Option 3: Filtered by status**
```
Successful:
{{range .accumulated}}
{{if eq .status "success"}}
✓ {{.name}}
{{end}}
{{end}}

Failed:
{{range .accumulated}}
{{if eq .status "failed"}}
✗ {{.name}}
{{end}}
{{end}}
```

**Option 4: Detailed report**
```
Processing Report
═════════════════════════════════════

{{range .accumulated}}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Record #{{.index}}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Name: {{.name | title}}
Status: {{.status | upper}}
{{if .error}}
Error: {{.error}}
{{end}}
{{end}}

═════════════════════════════════════
Total Processed: {{.iteration_count}}
```

## Tips

1. **Always match your {{end}} tags** - Every `{{range}}` and `{{if}}` needs an `{{end}}`
2. **Use {{json .}} to debug** - See the entire data structure
3. **Check field names** - Case-sensitive! `{{.Name}}` ≠ `{{.name}}`
4. **Test incrementally** - Start simple, add complexity
5. **Watch whitespace** - Templates preserve newlines and spaces

## When to Use Each Mode

### Use Simple Mode When:
- ✅ Just inserting a few variables
- ✅ No loops or conditions needed
- ✅ Want quick and simple

### Use Template Engine When:
- ✅ Iterating over arrays
- ✅ Conditional content
- ✅ Need formatting functions
- ✅ Complex data structures
- ✅ Professional reports

## Migration from Simple to Template Engine

**Before (Simple):**
```
Results: {{accumulated}}
```

**After (Template Engine):**
```
Results:
{{range .accumulated}}
• {{.name}} - {{.status}}
{{end}}
```

Just add a dot to your variables and you unlock all features!

