package resend

import (
	"context"
	"fmt"
	"log/slog"

	resendclient "github.com/resend/resend-go/v2"

	"github.com/team-attention/cops/api/internal/platform/outbound/email"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// ResendEmailService implements EmailServicePort using Resend API.
type ResendEmailService struct {
	logger *slog.Logger
	client *resendclient.Client
	config *config.ResendConfig
}

// NewResendEmailService creates a new Resend email service.
func NewResendEmailService(l *slog.Logger, cfg *config.Config) email.EmailServicePort {
	var client *resendclient.Client
	if cfg.Resend.APIKey != "" {
		client = resendclient.NewClient(cfg.Resend.APIKey)
	}

	return &ResendEmailService{
		logger: l.With(slog.String("name", "platform.email.resend")),
		client: client,
		config: &cfg.Resend,
	}
}

// Send sends an email using Resend API.
func (s *ResendEmailService) Send(ctx context.Context, params email.SendEmailParams) error {
	// Check if Resend is configured
	if s.config.APIKey == "" {
		s.logger.Warn("Resend not configured, skipping email send",
			slog.String("to", params.To),
			slog.String("subject", params.Subject),
		)
		return nil
	}

	// Build From address string
	fromAddr := fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddr)

	// Create send email request
	request := &resendclient.SendEmailRequest{
		From:    fromAddr,
		To:      []string{params.To},
		Subject: params.Subject,
		Text:    params.TextBody,
	}

	// Add HTML body if provided
	if params.HTMLBody != "" {
		request.Html = params.HTMLBody
	}

	// Send the email
	_, err := s.client.Emails.SendWithContext(ctx, request)
	if err != nil {
		s.logger.Error("failed to send email via Resend",
			slog.String("to", params.To),
			slog.String("subject", params.Subject),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}

	s.logger.Info("email sent successfully via Resend",
		slog.String("to", params.To),
		slog.String("subject", params.Subject),
	)

	return nil
}

// Interface verification
var _ email.EmailServicePort = (*ResendEmailService)(nil)
