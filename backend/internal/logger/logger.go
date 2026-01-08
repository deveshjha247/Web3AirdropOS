package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

// ContextKey for logger context
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
)

// Config for logger
type Config struct {
	Level      string // debug, info, warn, error
	Pretty     bool   // human-readable output (dev only)
	Output     io.Writer
	TimeFormat string
}

// Init initializes the global logger
func Init(cfg Config) {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}

	zerolog.TimeFieldFormat = cfg.TimeFormat

	var output io.Writer = cfg.Output
	if cfg.Pretty {
		output = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: "15:04:05",
		}
	}

	level := parseLevel(cfg.Level)
	log = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()
}

func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Get returns the global logger
func Get() *zerolog.Logger {
	return &log
}

// WithRequestID adds request ID to logger context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds user ID to logger context
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// FromContext gets a logger with context values
func FromContext(ctx context.Context) zerolog.Logger {
	l := log.With()

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		l = l.Str("request_id", requestID)
	}
	if userID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		l = l.Str("user_id", userID.String())
	}

	return l.Logger()
}

// Info logs at info level
func Info() *zerolog.Event {
	return log.Info()
}

// Debug logs at debug level
func Debug() *zerolog.Event {
	return log.Debug()
}

// Warn logs at warn level
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs at error level
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs at fatal level and exits
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// GinMiddleware returns a gin middleware for request logging
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()

		// Add request ID to context
		c.Set("request_id", requestID)
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set response header
		c.Header("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Log request
		duration := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Int("status", status).
			Dur("duration", duration).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int("size", c.Writer.Size()).
			Msg("HTTP request")
	}
}

// GinRecovery returns a gin recovery middleware with logging
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")

				log.Error().
					Interface("error", err).
					Str("request_id", requestID.(string)).
					Str("path", c.Request.URL.Path).
					Msg("Panic recovered")

				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
