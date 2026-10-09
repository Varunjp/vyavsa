package mailer

import "context"

// EmailSender defines the low-level provider-agnostic email transport contract
type EmailSender interface {
	SendEmail(
		ctx context.Context,
		to string,
		subject string,
		htmlBody string,
		textBody string,
	) error
}
