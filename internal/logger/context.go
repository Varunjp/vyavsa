package logger

import (
	"context"

	"github.com/rs/zerolog"
)

type contextKey string

const loggerKey contextKey = "logger"

func WithContext(
	ctx context.Context,
	log zerolog.Logger,
) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

func FromContext(ctx context.Context) zerolog.Logger {
	log, ok := ctx.Value(loggerKey).(zerolog.Logger)

	if !ok {
		return zerolog.Nop()
	}

	return log
}
