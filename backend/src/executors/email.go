package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

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

func NewEmailExecutor(db *gorm.DB, emailService EmailServiceInterface) *EmailExecutor {
	return &EmailExecutor{
		db:           db,
		emailService: emailService,
	}
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
				"data":      true, // Primary output: email sent successfully (boolean)
				"sent":      true, // Kept for backward compatibility
				"messageId": result.MessageID,
			},
		}, nil
	}

	// Fallback if email service not set (shouldn't happen in production)
	return &ExecutionResult{
		Success: true,
		Output: map[string]interface{}{
			"data": true, // Primary output: email sent successfully (boolean)
			"sent": true, // Kept for backward compatibility
			"note": "email service not configured",
		},
	}, nil
}

// replaceTemplateVariables processes template text with Go template engine
// Supports both simple {{.variable}} and advanced features like {{range}}, {{if}}, etc.
func (e *EmailExecutor) replaceTemplateVariables(text string, input interface{}) string {
	// Check if template uses advanced features (range, if, with, etc.)
	hasAdvancedFeatures := e.hasAdvancedTemplateFeatures(text)

	if hasAdvancedFeatures {
		// Use Go template engine for advanced features
		return e.executeGoTemplate(text, input)
	}

	// For simple variable replacement, use the legacy method for backward compatibility
	// This handles {{variable}} without the dot prefix
	return e.executeSimpleTemplate(text, input)
}

// hasAdvancedTemplateFeatures checks if the template uses Go template engine features
func (e *EmailExecutor) hasAdvancedTemplateFeatures(text string) bool {
	// Check for advanced keywords with flexible spacing
	advancedKeywords := []string{"range", "if", "with", "end", "else", "define", "template", "block", "$"}
	for _, keyword := range advancedKeywords {
		// Match {{keyword or {{ keyword (with space after {{)
		pattern := regexp.MustCompile(`\{\{\s*` + regexp.QuoteMeta(keyword) + `\s`)
		if pattern.MatchString(text) {
			log.Printf("✅ Detected Go template feature: %s", keyword)
			return true
		}
	}
	// Check if using dot notation like {{.variable}} which is Go template syntax
	re := regexp.MustCompile(`\{\{\s*\.[A-Za-z_]`)
	if re.MatchString(text) {
		log.Printf("✅ Detected Go template dot notation")
		return true
	}
	log.Printf("⚠️ No Go template features detected, using simple template")
	return false
}

// executeGoTemplate executes the template using Go's template engine
func (e *EmailExecutor) executeGoTemplate(text string, input interface{}) string {
	// Create a new template with custom functions
	tmpl, err := template.New("email").Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
		"jsonCompact": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"add": func(a, b interface{}) int {
			return toInt(a) + toInt(b)
		},
		"sub": func(a, b interface{}) int {
			return toInt(a) - toInt(b)
		},
		"mul": func(a, b interface{}) int {
			return toInt(a) * toInt(b)
		},
		"div": func(a, b interface{}) int {
			if toInt(b) == 0 {
				return 0
			}
			return toInt(a) / toInt(b)
		},
	}).Parse(text)

	if err != nil {
		// If template parsing fails, log error and return original text
		log.Printf("❌ Email template parsing failed: %v", err)
		log.Printf("Template text: %s", text)
		return text
	}

	// Execute the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, input)
	if err != nil {
		// If execution fails, log error and return original text
		log.Printf("❌ Email template execution failed: %v", err)
		log.Printf("Input data: %+v", input)
		return text
	}

	return buf.String()
}

// executeSimpleTemplate handles simple {{variable}} replacement (legacy behavior)
// This maintains backward compatibility with templates that don't use the dot prefix
func (e *EmailExecutor) executeSimpleTemplate(text string, input interface{}) string {
	// Match patterns like {{input.field}} or {{variable}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])

		// Get the value from input
		value := e.getValueFromPath(input, varName)

		// Convert to string
		if value != nil {
			return e.valueToString(value)
		}

		// If not found, keep the original placeholder
		return match
	})

	return result
}

// valueToString converts a value to a human-readable string
// For arrays and objects, it uses JSON formatting for readability
func (e *EmailExecutor) valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64, float32, float64, bool:
		return fmt.Sprintf("%v", v)
	case []interface{}, map[string]interface{}:
		// For arrays and objects, use pretty-printed JSON
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			// Fallback to default formatting if JSON encoding fails
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", v)
	}
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

// toInt converts an interface{} value to int
// Handles float64 (JSON numbers), int, and other numeric types
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case string:
		// Try to parse string as int
		if i, err := fmt.Sscanf(val, "%d", new(int)); err == nil && i == 1 {
			var result int
			fmt.Sscanf(val, "%d", &result)
			return result
		}
		return 0
	default:
		return 0
	}
}
