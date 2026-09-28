package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	Zerolog zerolog.Logger
}

func NewLogger(cfg Config) *Logger {
	level := parseLevel(cfg.Level)

	var output io.Writer

	if cfg.Environment == "development" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	} else {
		output = os.Stdout
	}

	log := zerolog.New(output).
		Level(level).With().Timestamp().Str("service", cfg.ServiceName).Str("environment", cfg.Environment).Logger()

	return &Logger{
		Zerolog: log,
	}
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel

	case "warn":
		return zerolog.WarnLevel

	case "error":
		return zerolog.ErrorLevel

	case "fatal":
		return zerolog.FatalLevel

	case "panic":
		return zerolog.PanicLevel

	default:
		return zerolog.InfoLevel
	}
}
