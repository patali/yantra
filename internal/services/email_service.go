package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/patali/yantra/internal/executors"
	"github.com/patali/yantra/internal/models"
	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

type EmailProvider string

const (
	ProviderResend  EmailProvider = "resend"
	ProviderMailgun EmailProvider = "mailgun"
	ProviderSES     EmailProvider = "ses"
	ProviderSMTP    EmailProvider = "smtp"
)

type EmailService struct {
	db *gorm.DB

	// Client caches for performance (thread-safe)
	resendClients  sync.Map // map[apiKey]*resend.Client
	mailgunClients sync.Map // map[domain:apiKey]mailgun.Mailgun
	sesClients     sync.Map // map[region:accessKey]*ses.Client
}

func NewEmailService(db *gorm.DB) *EmailService {
	return &EmailService{db: db}
}

// getResendClient returns a cached Resend client or creates a new one
func (s *EmailService) getResendClient(apiKey string) *resend.Client {
	if client, ok := s.resendClients.Load(apiKey); ok {
		return client.(*resend.Client)
	}

	client := resend.NewClient(apiKey)
	s.resendClients.Store(apiKey, client)
	return client
}

// getMailgunClient returns a cached Mailgun client or creates a new one
func (s *EmailService) getMailgunClient(domain, apiKey string) mailgun.Mailgun {
	key := domain + ":" + apiKey
	if client, ok := s.mailgunClients.Load(key); ok {
		return client.(mailgun.Mailgun)
	}

	client := mailgun.NewMailgun(domain, apiKey)
	s.mailgunClients.Store(key, client)
	return client
}

// getSESClient returns a cached SES client or creates a new one
func (s *EmailService) getSESClient(ctx context.Context, region, accessKey, secretKey string) (*ses.Client, error) {
	key := region + ":" + accessKey
	if client, ok := s.sesClients.Load(key); ok {
		return client.(*ses.Client), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, secretKey, "",
		)),
	)
	if err != nil {
		return nil, err
	}

	client := ses.NewFromConfig(cfg)
	s.sesClients.Store(key, client)
	return client, nil
}

// GetActiveProvider retrieves the active email provider configuration for an account
func (s *EmailService) GetActiveProvider(accountID string) (*models.EmailProviderSettings, error) {
	var settings models.EmailProviderSettings
	err := s.db.Where("account_id = ? AND is_active = ?", accountID, true).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active email provider configured")
		}
		return nil, err
	}
	return &settings, nil
}

// GetProviderByType retrieves a specific email provider configuration
func (s *EmailService) GetProviderByType(accountID string, provider EmailProvider) (*models.EmailProviderSettings, error) {
	var settings models.EmailProviderSettings
	err := s.db.Where("account_id = ? AND provider = ?", accountID, string(provider)).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("provider %s not configured", provider)
		}
		return nil, err
	}
	return &settings, nil
}

// SendEmail sends an email using the configured provider
func (s *EmailService) SendEmail(ctx context.Context, accountID string, options executors.EmailOptions) (*executors.EmailResult, error) {
	var providerConfig *models.EmailProviderSettings
	var err error

	// Use provider override if specified
	if options.ProviderOverride != nil {
		providerConfig, err = s.GetProviderByType(accountID, EmailProvider(*options.ProviderOverride))
	} else {
		providerConfig, err = s.GetActiveProvider(accountID)
	}

	if err != nil {
		return &executors.EmailResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Apply template if specified
	if options.Template != "" && options.TemplateVariables != nil {
		options.HTML = s.RenderTemplate(options.Template, options.TemplateVariables)
	}

	// Route to appropriate provider
	switch EmailProvider(providerConfig.Provider) {
	case ProviderResend:
		return s.sendViaResend(ctx, options, providerConfig)
	case ProviderMailgun:
		return s.sendViaMailgun(ctx, options, providerConfig)
	case ProviderSES:
		return s.sendViaSES(ctx, options, providerConfig)
	case ProviderSMTP:
		return s.sendViaSMTP(ctx, options, providerConfig)
	default:
		return &executors.EmailResult{
			Success: false,
			Error:   fmt.Sprintf("unknown email provider: %s", providerConfig.Provider),
		}, fmt.Errorf("unknown provider")
	}
}

// sendViaResend sends email via Resend API
func (s *EmailService) sendViaResend(_ctx context.Context, options executors.EmailOptions, config *models.EmailProviderSettings) (*executors.EmailResult, error) {
	if config.APIKey == nil || *config.APIKey == "" {
		return &executors.EmailResult{Success: false, Error: "Resend API key not configured"}, fmt.Errorf("API key missing")
	}

	// Get cached Resend client
	client := s.getResendClient(*config.APIKey)

	// Build "from" address
	from := s.buildFromAddress(config)

	// Build email request
	params := &resend.SendEmailRequest{
		From:    from,
		To:      options.To,
		Subject: options.Subject,
	}

	// Add CC and BCC if provided
	if len(options.CC) > 0 {
		params.Cc = options.CC
	}
	if len(options.BCC) > 0 {
		params.Bcc = options.BCC
	}

	// Set content (HTML takes priority over text)
	if options.HTML != "" {
		params.Html = options.HTML
		if options.Text != "" {
			params.Text = options.Text // Include text as fallback
		}
	} else if options.Text != "" {
		params.Text = options.Text
	} else {
		return &executors.EmailResult{Success: false, Error: "email must have either text or HTML content"}, fmt.Errorf("missing content")
	}

	// Add attachments if any
	if len(options.Attachments) > 0 {
		attachments := make([]*resend.Attachment, len(options.Attachments))
		for i, att := range options.Attachments {
			attachments[i] = &resend.Attachment{
				Filename: att.Filename,
				Content:  att.Content,
			}
		}
		params.Attachments = attachments
	}

	// Send email
	sent, err := client.Emails.Send(params)
	if err != nil {
		return &executors.EmailResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &executors.EmailResult{
		Success:   true,
		MessageID: sent.Id,
	}, nil
}

// sendViaMailgun sends email via Mailgun API
func (s *EmailService) sendViaMailgun(ctx context.Context, options executors.EmailOptions, config *models.EmailProviderSettings) (*executors.EmailResult, error) {
	if config.APIKey == nil || *config.APIKey == "" || config.Domain == nil || *config.Domain == "" {
		return &executors.EmailResult{Success: false, Error: "Mailgun API key and domain required"}, fmt.Errorf("missing configuration")
	}

	mg := s.getMailgunClient(*config.Domain, *config.APIKey)

	from := s.buildFromAddress(config)

	message := mg.NewMessage(
		from,
		options.Subject,
		options.Text,
		options.To...,
	)

	if options.HTML != "" {
		message.SetHtml(options.HTML)
	}

	if len(options.CC) > 0 {
		for _, cc := range options.CC {
			message.AddCC(cc)
		}
	}

	if len(options.BCC) > 0 {
		for _, bcc := range options.BCC {
			message.AddBCC(bcc)
		}
	}

	// Add attachments
	for _, att := range options.Attachments {
		message.AddBufferAttachment(att.Filename, att.Content)
	}

	_, id, err := mg.Send(ctx, message)
	if err != nil {
		return &executors.EmailResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &executors.EmailResult{
		Success:   true,
		MessageID: id,
	}, nil
}

// sendViaSES sends email via AWS SES
func (s *EmailService) sendViaSES(ctx context.Context, options executors.EmailOptions, providerConfig *models.EmailProviderSettings) (*executors.EmailResult, error) {
	if providerConfig.AccessKeyID == nil || providerConfig.SecretAccessKey == nil || providerConfig.Region == nil {
		return &executors.EmailResult{Success: false, Error: "AWS SES requires accessKeyId, secretAccessKey, and region"}, fmt.Errorf("missing configuration")
	}

	// Get cached SES client
	client, err := s.getSESClient(ctx, *providerConfig.Region, *providerConfig.AccessKeyID, *providerConfig.SecretAccessKey)
	if err != nil {
		return &executors.EmailResult{Success: false, Error: err.Error()}, err
	}

	from := s.buildFromAddress(providerConfig)

	input := &ses.SendEmailInput{
		Source: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: options.To,
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(options.Subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{},
		},
	}

	if len(options.CC) > 0 {
		input.Destination.CcAddresses = options.CC
	}
	if len(options.BCC) > 0 {
		input.Destination.BccAddresses = options.BCC
	}

	if options.HTML != "" {
		input.Message.Body.Html = &types.Content{
			Data:    aws.String(options.HTML),
			Charset: aws.String("UTF-8"),
		}
	}
	if options.Text != "" {
		input.Message.Body.Text = &types.Content{
			Data:    aws.String(options.Text),
			Charset: aws.String("UTF-8"),
		}
	}

	if options.HTML == "" && options.Text == "" {
		return &executors.EmailResult{Success: false, Error: "email must have either text or HTML content"}, fmt.Errorf("missing content")
	}

	result, err := client.SendEmail(ctx, input)
	if err != nil {
		return &executors.EmailResult{Success: false, Error: err.Error()}, err
	}

	return &executors.EmailResult{
		Success:   true,
		MessageID: *result.MessageId,
	}, nil
}

// sendViaSMTP sends email via SMTP
func (s *EmailService) sendViaSMTP(ctx context.Context, options executors.EmailOptions, config *models.EmailProviderSettings) (*executors.EmailResult, error) {
	if config.SMTPHost == nil || config.SMTPPort == nil || config.SMTPUser == nil || config.SMTPPassword == nil {
		return &executors.EmailResult{Success: false, Error: "SMTP requires host, port, user, and password"}, fmt.Errorf("missing configuration")
	}

	from := s.buildFromAddress(config)

	// Build email message
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(options.To, ", ")))
	if len(options.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(options.CC, ", ")))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", options.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(options.Attachments) > 0 || options.HTML != "" {
		// Multipart message
		boundary := "==BOUNDARY=="
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Text/HTML part
		if options.HTML != "" {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(options.HTML)
			buf.WriteString("\r\n")
		} else if options.Text != "" {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(options.Text)
			buf.WriteString("\r\n")
		}

		// Attachments
		for _, att := range options.Attachments {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", att.Filename))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
			buf.WriteString("\r\n")
			buf.WriteString(base64.StdEncoding.EncodeToString(att.Content))
			buf.WriteString("\r\n")
		}

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Simple text email
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(options.Text)
	}

	// Setup authentication
	auth := smtp.PlainAuth("", *config.SMTPUser, *config.SMTPPassword, *config.SMTPHost)

	// Determine recipients
	recipients := append([]string{}, options.To...)
	recipients = append(recipients, options.CC...)
	recipients = append(recipients, options.BCC...)

	// Send email
	addr := fmt.Sprintf("%s:%d", *config.SMTPHost, *config.SMTPPort)
	err := smtp.SendMail(addr, auth, *config.SMTPUser, recipients, buf.Bytes())
	if err != nil {
		return &executors.EmailResult{Success: false, Error: err.Error()}, err
	}

	return &executors.EmailResult{
		Success:   true,
		MessageID: "smtp-sent",
	}, nil
}

// TestProvider tests an email provider configuration (sends to fromEmail)
func (s *EmailService) TestProvider(ctx context.Context, accountID string, provider EmailProvider, config *models.EmailProviderSettings) (*executors.EmailResult, error) {
	if config.FromEmail == nil || *config.FromEmail == "" {
		return &executors.EmailResult{Success: false, Error: "from email is required"}, fmt.Errorf("from email is required")
	}
	return s.TestProviderToEmail(ctx, accountID, provider, config, *config.FromEmail)
}

// TestProviderToEmail tests an email provider configuration by sending to a specific email
func (s *EmailService) TestProviderToEmail(ctx context.Context, accountID string, provider EmailProvider, config *models.EmailProviderSettings, toEmail string) (*executors.EmailResult, error) {
	testEmail := executors.EmailOptions{
		To:      []string{toEmail},
		Subject: "Test Email - Yantra",
		Text:    "This is a test email from your Yantra workflow system. If you received this, your email configuration is working correctly!",
		HTML:    "<h1>Test Email</h1><p>This is a test email from your Yantra workflow system.</p><p>If you received this, your email configuration is working correctly!</p>",
	}

	switch provider {
	case ProviderResend:
		return s.sendViaResend(ctx, testEmail, config)
	case ProviderMailgun:
		return s.sendViaMailgun(ctx, testEmail, config)
	case ProviderSES:
		return s.sendViaSES(ctx, testEmail, config)
	case ProviderSMTP:
		return s.sendViaSMTP(ctx, testEmail, config)
	default:
		return &executors.EmailResult{Success: false, Error: "unknown provider"}, fmt.Errorf("unknown provider")
	}
}

// RenderTemplate renders a template with variables
func (s *EmailService) RenderTemplate(template string, variables map[string]interface{}) string {
	rendered := template
	for key, value := range variables {
		pattern := fmt.Sprintf(`{{\s*%s\s*}}`, regexp.QuoteMeta(key))
		re := regexp.MustCompile(pattern)
		rendered = re.ReplaceAllString(rendered, fmt.Sprintf("%v", value))
	}
	return rendered
}

// buildFromAddress builds the "From" email address
func (s *EmailService) buildFromAddress(config *models.EmailProviderSettings) string {
	if config.FromEmail == nil {
		return "noreply@example.com"
	}

	if config.FromName != nil && *config.FromName != "" {
		return fmt.Sprintf("%s <%s>", *config.FromName, *config.FromEmail)
	}

	return *config.FromEmail
}
