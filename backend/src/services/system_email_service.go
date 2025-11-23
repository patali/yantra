package services

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/patali/yantra/src/config"
	"github.com/resend/resend-go/v2"
)

// SystemEmailService handles sending system emails (like password resets)
// that don't belong to a specific account
type SystemEmailService struct {
	config *config.Config
}

func NewSystemEmailService(cfg *config.Config) *SystemEmailService {
	return &SystemEmailService{config: cfg}
}

type SystemEmailOptions struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// SendEmail sends a system email using the configured provider or logs it in development
func (s *SystemEmailService) SendEmail(options SystemEmailOptions) error {
	// In development mode, log the email
	if s.config.Environment == "development" {
		return s.logEmail(options)
	}

	// Route to appropriate provider based on configuration
	switch strings.ToLower(s.config.SystemEmailProvider) {
	case "resend":
		if s.config.SystemEmailResendAPIKey == "" {
			return fmt.Errorf("SYSTEM_EMAIL_RESEND_API_KEY is not configured")
		}
		return s.sendViaResend(options)
	case "smtp":
		if s.config.SystemEmailSMTPHost == "" {
			return fmt.Errorf("SYSTEM_EMAIL_SMTP_HOST is not configured")
		}
		return s.sendViaSMTP(options)
	default:
		return fmt.Errorf("unknown email provider: %s (must be 'smtp' or 'resend')", s.config.SystemEmailProvider)
	}
}

// logEmail logs the email content instead of sending (for development)
func (s *SystemEmailService) logEmail(options SystemEmailOptions) error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📧 SYSTEM EMAIL (Development Mode - Not Sent)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("From: %s <%s>\n", s.config.SystemEmailFromName, s.config.SystemEmailFrom)
	fmt.Printf("To: %s\n", options.To)
	fmt.Printf("Subject: %s\n", options.Subject)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if options.HTML != "" {
		fmt.Println("HTML Content:")
		fmt.Println(options.HTML)
	} else {
		fmt.Println("Text Content:")
		fmt.Println(options.Text)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

// sendViaSMTP sends email via SMTP
func (s *SystemEmailService) sendViaSMTP(options SystemEmailOptions) error {
	from := s.buildFromAddress()

	// Build email message
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", options.To))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", options.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if options.HTML != "" {
		// Multipart message
		boundary := "==YANTRA_BOUNDARY=="
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Text part
		if options.Text != "" {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			buf.WriteString("\r\n")
			buf.WriteString(options.Text)
			buf.WriteString("\r\n")
		}

		// HTML part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(options.HTML)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Simple text email
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(options.Text)
	}

	// Setup authentication
	auth := smtp.PlainAuth("", s.config.SystemEmailSMTPUser, s.config.SystemEmailSMTPPassword, s.config.SystemEmailSMTPHost)

	// Send email
	addr := fmt.Sprintf("%s:%s", s.config.SystemEmailSMTPHost, s.config.SystemEmailSMTPPort)
	err := smtp.SendMail(addr, auth, s.config.SystemEmailFrom, []string{options.To}, buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// sendViaResend sends email via Resend API
func (s *SystemEmailService) sendViaResend(options SystemEmailOptions) error {
	// Create Resend client
	client := resend.NewClient(s.config.SystemEmailResendAPIKey)

	// Build "from" address
	from := s.buildFromAddress()

	// Build email request
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{options.To},
		Subject: options.Subject,
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
		return fmt.Errorf("email must have either text or HTML content")
	}

	// Send email
	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}

	return nil
}

// buildFromAddress builds the "From" email address
func (s *SystemEmailService) buildFromAddress() string {
	if s.config.SystemEmailFromName != "" {
		return fmt.Sprintf("%s <%s>", s.config.SystemEmailFromName, s.config.SystemEmailFrom)
	}
	return s.config.SystemEmailFrom
}

// RenderPasswordResetEmail renders the password reset email template
func (s *SystemEmailService) RenderPasswordResetEmail(resetToken string) (html string, text string) {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimSuffix(s.config.AppURL, "/"), resetToken)

	html = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f4f4f4; padding: 20px 0;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
                    <!-- Header -->
                    <tr>
                        <td style="background-color: #4F46E5; padding: 30px 40px; text-align: center;">
                            <h1 style="margin: 0; color: #ffffff; font-size: 24px; font-weight: 600;">Reset Your Password</h1>
                        </td>
                    </tr>

                    <!-- Body -->
                    <tr>
                        <td style="padding: 40px;">
                            <p style="margin: 0 0 20px 0; color: #333333; font-size: 16px; line-height: 1.5;">
                                You recently requested to reset your password for your Yantra account. Click the button below to reset it.
                            </p>

                            <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="margin: 30px 0;">
                                <tr>
                                    <td align="center">
                                        <a href="%s" style="display: inline-block; padding: 14px 40px; background-color: #4F46E5; color: #ffffff; text-decoration: none; border-radius: 6px; font-size: 16px; font-weight: 600;">Reset Password</a>
                                    </td>
                                </tr>
                            </table>

                            <p style="margin: 20px 0; color: #666666; font-size: 14px; line-height: 1.5;">
                                If the button doesn't work, copy and paste this link into your browser:
                            </p>

                            <p style="margin: 10px 0; padding: 12px; background-color: #f8f8f8; border-radius: 4px; word-break: break-all; color: #4F46E5; font-size: 13px;">
                                %s
                            </p>

                            <p style="margin: 30px 0 10px 0; color: #666666; font-size: 14px; line-height: 1.5;">
                                <strong>This link will expire in 1 hour.</strong>
                            </p>

                            <p style="margin: 20px 0 0 0; color: #666666; font-size: 14px; line-height: 1.5;">
                                If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.
                            </p>
                        </td>
                    </tr>

                    <!-- Footer -->
                    <tr>
                        <td style="background-color: #f8f8f8; padding: 20px 40px; text-align: center; border-top: 1px solid #e5e5e5;">
                            <p style="margin: 0; color: #999999; font-size: 12px;">
                                This is an automated email from Yantra. Please do not reply to this email.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`, resetURL, resetURL)

	text = fmt.Sprintf(`
Reset Your Password

You recently requested to reset your password for your Yantra account.

To reset your password, click the following link or copy and paste it into your browser:

%s

This link will expire in 1 hour.

If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.

---
This is an automated email from Yantra. Please do not reply to this email.
`, resetURL)

	return html, text
}
