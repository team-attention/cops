package email

import "context"

// SendEmailParams contains parameters for sending an email.
type SendEmailParams struct {
	To       string // Recipient email address
	Subject  string // Email subject line
	TextBody string // Plain text body
	HTMLBody string // HTML body (optional)
}

// EmailServicePort defines interface for sending emails.
type EmailServicePort interface {
	// Send sends an email using the configured SMTP server.
	// Returns error if email cannot be sent.
	Send(ctx context.Context, params SendEmailParams) error
}
