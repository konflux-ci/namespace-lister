package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/konflux-ci/namespace-lister/internal/envconfig"
	"github.com/konflux-ci/namespace-lister/internal/contextkey"
)

// BuildLogger constructs a new instance of the logger
func BuildLogger() *slog.Logger {
	logLevel := GetLogLevel()

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	return slog.New(handler)
}

// GetLogLevel fetches the log level from the appropriate environment variable
func GetLogLevel() slog.Level {
	env := os.Getenv(envconfig.EnvLogLevel)
	level, err := strconv.Atoi(env)
	if err != nil {
		return slog.LevelError
	}
	return slog.Level(level)
}

// SetLoggerIntoContext sets the provided logger into the context
func SetLoggerIntoContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextkey.ContextKeyLogger, logger)
}

// GetLoggerFromContext retrieves the logger from the context.
// If no logger is present in the context, it will return a DiscardAll logger.
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextkey.ContextKeyLogger).(*slog.Logger); ok && l != nil {
		return l
	}

	// return a discard all logger
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}
