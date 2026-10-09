package mailer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

var (
	// apiKeyRegex redacts Resend API keys from error messages or logs
	apiKeyRegex = regexp.MustCompile(`re_[a-zA-Z0-9_]{10,}`)
)

// PermanentError identifies errors that should not be retried by the background worker
type PermanentError interface {
	IsPermanent() bool
}

// IsPermanentError returns true if err indicates an unrecoverable delivery error
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}
	var p PermanentError
	if errors.As(err, &p) {
		return p.IsPermanent()
	}
	return false
}

// deliveryError wraps an email dispatch error with a retryability flag
type deliveryError struct {
	err       error
	permanent bool
}

func (e *deliveryError) Error() string {
	return e.err.Error()
}

func (e *deliveryError) Unwrap() error {
	return e.err
}

func (e *deliveryError) IsPermanent() bool {
	return e.permanent
}

// errorSender represents a misconfigured or failed-to-initialize provider
type errorSender struct {
	initErr error
}

func newErrorSender(err error) EmailSender {
	return &errorSender{initErr: err}
}

func (s *errorSender) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	return &deliveryError{
		err:       fmt.Errorf("email provider is not properly configured: %w", s.initErr),
		permanent: true,
	}
}

// sanitizeErrorString strips API tokens, secrets, or passwords from error strings
func sanitizeErrorString(s string) string {
	if s == "" {
		return ""
	}
	return apiKeyRegex.ReplaceAllString(s, "[REDACTED_KEY]")
}
