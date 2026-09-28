package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config configures structured logging
type Config struct {
	Environment string
	ServiceName string
	Level       string
	Format      string // "json" or "text"
	Output      io.Writer
}

// Logger wraps standard slog.Logger
type Logger struct {
	*slog.Logger
}

// New creates and configures a new structured logger
func New(cfg Config) *Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" || cfg.Environment == "production" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	// Always wrap with our ContextualHandler to automatically extract request_id, tenant_id, user_id from context
	handler = newContextualHandler(handler)

	baseLogger := slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	return &Logger{
		Logger: baseLogger,
	}
}

// Default creates a default development logger
func Default() *Logger {
	return New(Config{
		Environment: "development",
		ServiceName: "vyavsa-bill-book-api",
		Level:       "info",
		Format:      "text",
	})
}

func parseLevel(levelStr string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	tenantIDKey  contextKey = "tenant_id"
	userIDKey    contextKey = "user_id"
	loggerKey    contextKey = "logger_instance"
)

// WithRequestID adds a request_id to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts request_id from context
func RequestIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}

// WithTenantID adds a tenant_id to context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantIDFromContext extracts tenant_id from context
func TenantIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(tenantIDKey).(string); ok {
		return val
	}
	return ""
}

// WithUserID adds a user_id to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts user_id from context
func UserIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(userIDKey).(string); ok {
		return val
	}
	return ""
}

// WithLogger stores the logger in the context
func WithLogger(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// FromContext retrieves logger from context or returns default slog logger
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(loggerKey).(*Logger); ok && log != nil {
		return log.Logger
	}
	return slog.Default()
}

// contextualHandler automatically extracts standard context fields and adds them as slog attributes
type contextualHandler struct {
	slog.Handler
}

func newContextualHandler(h slog.Handler) *contextualHandler {
	return &contextualHandler{Handler: h}
}

func (h *contextualHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	if tenantID := TenantIDFromContext(ctx); tenantID != "" {
		r.AddAttrs(slog.String("tenant_id", tenantID))
	}
	if userID := UserIDFromContext(ctx); userID != "" {
		r.AddAttrs(slog.String("user_id", userID))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newContextualHandler(h.Handler.WithAttrs(attrs))
}

func (h *contextualHandler) WithGroup(name string) slog.Handler {
	return newContextualHandler(h.Handler.WithGroup(name))
}
