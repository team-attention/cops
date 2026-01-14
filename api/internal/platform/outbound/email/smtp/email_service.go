package smtp

import (
	"context"
	"fmt"
	"log/slog"

	gomail "github.com/wneessen/go-mail"

	"github.com/team-attention/cops/api/internal/platform/outbound/email"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// SMTPEmailService implements EmailServicePort using SMTP.
type SMTPEmailService struct {
	logger *slog.Logger
	config *config.SMTPConfig
}

// NewSMTPEmailService creates a new SMTP email service.
func NewSMTPEmailService(l *slog.Logger, cfg *config.Config) email.EmailServicePort {
	return &SMTPEmailService{
		logger: l.With(slog.String("name", "platform.email.smtp")),
		config: &cfg.SMTP,
	}
}

// Send sends an email using SMTP.
func (s *SMTPEmailService) Send(ctx context.Context, params email.SendEmailParams) error {
	// Check if SMTP is configured
	if s.config.Host == "" {
		s.logger.Warn("SMTP not configured, skipping email send",
			slog.String("to", params.To),
			slog.String("subject", params.Subject),
		)
		return nil
	}

	// Create new go-mail message
	msg := gomail.NewMsg()

	// Set From address
	fromAddr := fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddr)
	if err := msg.FromFormat(s.config.FromName, s.config.FromAddr); err != nil {
		s.logger.Error("failed to set FROM address",
			slog.String("from", fromAddr),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to set FROM address: %w", err)
	}

	// Set To address
	if err := msg.To(params.To); err != nil {
		s.logger.Error("failed to set TO address",
			slog.String("to", params.To),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to set TO address: %w", err)
	}

	// Set Subject
	msg.Subject(params.Subject)

	// Set plain text body
	msg.SetBodyString(gomail.TypeTextPlain, params.TextBody)

	// If HTMLBody provided, add HTML alternative
	if params.HTMLBody != "" {
		msg.AddAlternativeString(gomail.TypeTextHTML, params.HTMLBody)
	}

	// Create SMTP client with TLS mandatory on port 587
	client, err := gomail.NewClient(
		s.config.Host,
		gomail.WithPort(s.config.Port),
		gomail.WithTLSPortPolicy(gomail.TLSMandatory),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(s.config.Username),
		gomail.WithPassword(s.config.Password),
	)
	if err != nil {
		s.logger.Error("failed to create SMTP client",
			slog.String("host", s.config.Host),
			slog.Int("port", s.config.Port),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Send the email
	if err := client.DialAndSend(msg); err != nil {
		s.logger.Error("failed to send email",
			slog.String("to", params.To),
			slog.String("subject", params.Subject),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("email sent successfully",
		slog.String("to", params.To),
		slog.String("subject", params.Subject),
	)

	return nil
}

// Interface verification
var _ email.EmailServicePort = (*SMTPEmailService)(nil)
