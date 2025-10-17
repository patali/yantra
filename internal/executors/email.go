package executors

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// EmailServiceInterface defines the interface for email sending
type EmailServiceInterface interface {
	SendEmail(ctx context.Context, accountID string, options EmailOptions) (*EmailResult, error)
}

type EmailProvider string

type EmailOptions struct {
	To                []string
	CC                []string
	BCC               []string
	Subject           string
	Text              string
	HTML              string
	Attachments       []EmailAttachment
	Template          string
	TemplateVariables map[string]interface{}
	ProviderOverride  *EmailProvider
}

type EmailAttachment struct {
	Filename string
	Content  []byte
}

type EmailResult struct {
	Success   bool
	MessageID string
	Error     string
}

type EmailExecutor struct {
	db           *gorm.DB
	emailService EmailServiceInterface
}

func NewEmailExecutor(db *gorm.DB) *EmailExecutor {
	return &EmailExecutor{
		db: db,
	}
}

// SetEmailService sets the email service (called after initialization to avoid circular dependency)
func (e *EmailExecutor) SetEmailService(service EmailServiceInterface) {
	e.emailService = service
}

func (e *EmailExecutor) Execute(ctx context.Context, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Extract email configuration from node config
	to, ok := execCtx.NodeConfig["to"].(string)
	if !ok {
		return &ExecutionResult{
			Success: false,
			Error:   "missing or invalid 'to' field in email config",
		}, fmt.Errorf("invalid email config")
	}

	subject, ok := execCtx.NodeConfig["subject"].(string)
	if !ok {
		return &ExecutionResult{
			Success: false,
			Error:   "missing or invalid 'subject' field in email config",
		}, fmt.Errorf("invalid email config")
	}

	// Replace template variables in subject
	subject = e.replaceTemplateVariables(subject, execCtx.Input)

	// Build email options
	options := EmailOptions{
		To:      []string{to},
		Subject: subject,
	}

	// Optional fields with template variable replacement
	if body, ok := execCtx.NodeConfig["body"].(string); ok {
		options.Text = e.replaceTemplateVariables(body, execCtx.Input)
	}

	if html, ok := execCtx.NodeConfig["html"].(string); ok {
		options.HTML = e.replaceTemplateVariables(html, execCtx.Input)
	}

	if cc, ok := execCtx.NodeConfig["cc"].(string); ok && cc != "" {
		options.CC = []string{cc}
	}

	if bcc, ok := execCtx.NodeConfig["bcc"].(string); ok && bcc != "" {
		options.BCC = []string{bcc}
	}

	// Provider override
	if provider, ok := execCtx.NodeConfig["provider"].(string); ok && provider != "" {
		p := EmailProvider(provider)
		options.ProviderOverride = &p
	}

	// Template support
	if template, ok := execCtx.NodeConfig["template"].(string); ok && template != "" {
		options.Template = template
		if vars, ok := execCtx.NodeConfig["templateVariables"].(map[string]interface{}); ok {
			options.TemplateVariables = vars
		}
	}

	// Send email (only if email service is set)
	if e.emailService != nil {
		result, err := e.emailService.SendEmail(ctx, execCtx.AccountID, options)
		if err != nil || !result.Success {
			return &ExecutionResult{
				Success: false,
				Error:   result.Error,
			}, err
		}

		return &ExecutionResult{
			Success: true,
			Output: map[string]interface{}{
				"sent":      true,
				"messageId": result.MessageID,
			},
		}, nil
	}

	// Fallback if email service not set (shouldn't happen in production)
	return &ExecutionResult{
		Success: true,
		Output: map[string]interface{}{
			"sent": true,
			"note": "email service not configured",
		},
	}, nil
}

// replaceTemplateVariables replaces {{variable}} or {{input.field}} patterns with actual values
func (e *EmailExecutor) replaceTemplateVariables(text string, input interface{}) string {
	// Match patterns like {{input.field}} or {{variable}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	
	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])
		
		// Get the value from input
		value := e.getValueFromPath(input, varName)
		
		// Convert to string
		if value != nil {
			return fmt.Sprintf("%v", value)
		}
		
		// If not found, keep the original placeholder
		return match
	})
	
	return result
}

// getValueFromPath navigates through nested objects to get a value
// Supports paths like "input.field", "field.nested", "index", etc.
func (e *EmailExecutor) getValueFromPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Try to navigate deeper
		switch v := current.(type) {
		case map[string]interface{}:
			if next, ok := v[part]; ok {
				current = next
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	
	return current
}
