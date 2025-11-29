# Email Template Examples - Copy & Paste Ready

## 🔁 Loop Accumulator Results

### Basic List
```
Processing Complete!

Processed Items:
{{range .accumulated}}
• {{.name}}
{{end}}

Total: {{.iteration_count}} items
```

### Numbered List
```
Results:

{{range .accumulated}}
{{.index}}. {{.name}}
{{end}}

━━━━━━━━━━━━━━━━━━━━━━━
Completed: {{.iteration_count}} items
```

### Table Format
```
User Processing Report

┌────────┬─────────────────────┬──────────┐
│ Index  │ Name                │ Status   │
├────────┼─────────────────────┼──────────┤
{{range .accumulated}}
│ {{.index}}      │ {{.name}}           │ {{.status}}  │
{{end}}
└────────┴─────────────────────┴──────────┘

Total: {{.iteration_count}}
```

### With Status Icons
```
Daily Report - {{.date}}

{{range .accumulated}}
{{if eq .status "success"}}✅{{else}}❌{{end}} {{.name}}
{{end}}

Summary: {{.iteration_count}} items processed
```

### Detailed Cards
```
{{range .accumulated}}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Item #{{.index}}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Name: {{.name}}
Email: {{.email}}
Status: {{.status | upper}}
{{if .notes}}
Notes: {{.notes}}
{{end}}

{{end}}
Total Processed: {{.iteration_count}}
```

## 📊 Data Reports

### Success/Failure Report
```
Workflow Execution Report

✅ SUCCESSFUL
{{range .results}}
{{if eq .status "success"}}
• {{.name}} - Completed in {{.duration}}
{{end}}
{{end}}

❌ FAILED
{{range .results}}
{{if eq .status "failed"}}
• {{.name}} - Error: {{.error}}
{{end}}
{{end}}

━━━━━━━━━━━━━━━━━━━━━━━
Total: {{.total}} | Success: {{.successCount}} | Failed: {{.failedCount}}
```

### Metrics Dashboard
```
📈 Performance Metrics

{{if gt .requestCount 1000}}
🔥 High traffic detected!
{{end}}

Metrics:
- Total Requests: {{.requestCount}}
- Success Rate: {{.successRate}}%
- Avg Response Time: {{.avgResponseTime}}ms
- Errors: {{.errorCount}}

{{if gt .errorCount 0}}
⚠️ Recent Errors:
{{range .recentErrors}}
  • {{.time}}: {{.message}}
{{end}}
{{end}}
```

## 🛒 E-commerce

### Order Confirmation
```
🎉 Order Confirmed!

Hi {{.customerName | title}},

Thank you for your order!

ORDER #{{.orderId}}
━━━━━━━━━━━━━━━━━━━━━━━

{{range .items}}
{{.name}}
  Qty: {{.quantity}} × ${{.price}} = ${{.total}}
{{end}}

━━━━━━━━━━━━━━━━━━━━━━━
{{if .discount}}
Subtotal:    ${{.subtotal}}
Discount:   -${{.discount}}
{{end}}
Shipping:    ${{.shipping}}
Tax:         ${{.tax}}
━━━━━━━━━━━━━━━━━━━━━━━
TOTAL:       ${{.total}}

{{if .trackingNumber}}
📦 Track your package: {{.trackingNumber}}
{{else}}
We'll email you tracking info once shipped!
{{end}}
```

### Shipping Notification
```
📦 Your Order Has Shipped!

Hi {{.customerName}},

Great news! Your order #{{.orderId}} is on its way!

Tracking Number: {{.trackingNumber}}
Carrier: {{.carrier}}
Expected Delivery: {{.deliveryDate}}

Items Shipped:
{{range .items}}
• {{.name}} (Qty: {{.quantity}})
{{end}}

Track your package: {{.trackingUrl}}
```

## 👥 User Management

### Welcome Email
```
Welcome to {{.companyName}}! 🎉

Hi {{.firstName | title}},

We're excited to have you on board!

Your account details:
━━━━━━━━━━━━━━━━━━━━━━━
Email: {{.email}}
Username: {{.username}}
Account Type: {{.accountType | title}}
{{if .isPremium}}
Status: ⭐ Premium Member
{{end}}

{{if .hasTrialDays}}
Your {{.trialDays}}-day free trial starts now!
{{end}}

Get Started:
1. Complete your profile
2. Explore features
3. Invite team members

Questions? Just reply to this email!
```

### Activity Summary
```
📊 Weekly Activity Summary

Hello {{.userName}},

Here's what happened this week:

ACTIVITY
{{range .activities}}
• {{.date}}: {{.action}} {{if .details}}({{.details}}){{end}}
{{end}}

STATISTICS
━━━━━━━━━━━━━━━━━━━━━━━
Total Actions: {{.totalActions}}
{{if gt .totalActions .lastWeekActions}}
📈 {{.growth}}% increase from last week!
{{end}}

{{range .achievements}}
🏆 Achievement Unlocked: {{.name}}
{{end}}
```

## 🔔 Notifications

### Alert with Severity
```
{{if eq .severity "critical"}}🚨{{else if eq .severity "warning"}}⚠️{{else}}ℹ️{{end}} {{.title | upper}}

{{.message}}

Details:
{{range .details}}
• {{.key}}: {{.value}}
{{end}}

{{if eq .severity "critical"}}
⚠️ IMMEDIATE ACTION REQUIRED
{{end}}

Time: {{.timestamp}}
```

### Approval Request
```
📋 Approval Request

Hi {{.approverName}},

{{.requesterName}} is requesting approval for:

REQUEST DETAILS
━━━━━━━━━━━━━━━━━━━━━━━
Type: {{.requestType}}
Amount: ${{.amount}}
Reason: {{.reason}}

{{if .items}}
Items:
{{range .items}}
  • {{.description}} - ${{.cost}}
{{end}}
{{end}}

Please review and approve/reject:
[Approve] [Reject]

Submitted: {{.submittedDate}}
```

## 🔄 Workflow Updates

### Workflow Completion
```
✅ Workflow Completed Successfully

Workflow: {{.workflowName}}
Execution ID: {{.executionId}}

SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━
Started: {{.startTime}}
Completed: {{.endTime}}
Duration: {{.duration}}

NODES EXECUTED
{{range .nodeExecutions}}
{{if eq .status "success"}}✓{{else}}✗{{end}} {{.nodeName}} ({{.duration}})
{{end}}

{{if .outputData}}
Final Output:
{{json .outputData}}
{{end}}
```

### Workflow Error
```
❌ Workflow Failed

Workflow: {{.workflowName}}
Execution ID: {{.executionId}}

ERROR DETAILS
━━━━━━━━━━━━━━━━━━━━━━━
Node: {{.failedNode}}
Error: {{.errorMessage}}

{{if .stackTrace}}
Stack Trace:
{{.stackTrace}}
{{end}}

Execution Timeline:
{{range .nodeExecutions}}
{{if eq .status "success"}}✓{{else if eq .status "error"}}✗{{else}}○{{end}} {{.nodeName}}
{{end}}

Time: {{.timestamp}}
```

## 🎯 Marketing

### Personalized Campaign
```
{{if .isPremium}}👑{{end}} Hey {{.firstName}}!

{{if .lastPurchaseDays}}
{{if lt .lastPurchaseDays 7}}
Thanks for your recent purchase!
{{else if lt .lastPurchaseDays 30}}
We miss you! Here's what's new:
{{else}}
It's been a while! Welcome back with 20% off:
{{end}}
{{end}}

FEATURED ITEMS
{{range .recommendations}}
━━━━━━━━━━━━━━━━━━━━━━━
{{.name}}
{{if .discount}}
Was: ${{.originalPrice}} → Now: ${{.price}} ({{.discount}}% off!)
{{else}}
Price: ${{.price}}
{{end}}
{{end}}

{{if .couponCode}}
Use code: {{.couponCode}} at checkout
{{end}}
```

## 💡 Pro Tips

1. **Copy the template you need**
2. **Replace variable names** with your actual data fields
3. **Test with real data** from your workflow
4. **Adjust formatting** (whitespace, icons, borders) to match your style
5. **Add your branding** (logo, colors, company name)

## 🔧 Customization

All templates can be customized:
- Change icons (✅ ❌ 📊 🎉 etc.)
- Adjust borders (━ ═ │ ┌ └)
- Add/remove sections
- Modify conditional logic
- Change formatting functions (upper, lower, title)

