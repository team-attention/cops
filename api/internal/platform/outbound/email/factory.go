package email

import (
	"log/slog"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// NewEmailService creates an email service based on configuration.
// Priority: Resend (if configured) > SMTP (if configured) > No-op (logs warning)
func NewEmailService(l *slog.Logger, cfg *config.Config, resendSvc EmailServicePort, smtpSvc EmailServicePort) EmailServicePort {
	logger := l.With(slog.String("name", "platform.email.factory"))

	// Check if Resend is configured (priority)
	if cfg.Resend.APIKey != "" {
		logger.Info("Using Resend email service")
		return resendSvc
	}

	// Check if SMTP is configured (fallback)
	if cfg.SMTP.Host != "" {
		logger.Info("Using SMTP email service")
		return smtpSvc
	}

	// No email service configured - return SMTP which handles gracefully
	logger.Warn("No email service configured, emails will be skipped")
	return smtpSvc
}
