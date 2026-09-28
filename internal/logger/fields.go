package logger

import (
	"github.com/rs/zerolog"
)

func WithRequestID(
	log zerolog.Logger,
	requestID string,
) zerolog.Logger {
	return log.With().
		Str("request_id", requestID).
		Logger()
}
