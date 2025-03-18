package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
)

// buildLogger constructs a new instance of the logger
func buildLogger() *slog.Logger {
	logLevel := getLogLevel()

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	return slog.New(handler)
}

// getLogLevel fetches the log level from the appropriate environment variable
func getLogLevel() slog.Level {
	env := os.Getenv(EnvLogLevel)
	level, err := strconv.Atoi(env)
	if err != nil {
		return slog.LevelError
	}
	return slog.Level(level)
}

// setLoggerIntoContext sets the provided logger into the context
func setLoggerIntoContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ContextKeyLogger, logger)
}

// getLoggerFromContext retrieves the logger from the context.
// If no logger is present in the context, it will return a DiscardAll logger.
func getLoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ContextKeyLogger).(*slog.Logger); ok && l != nil {
		return l
	}

	// return a discard all logger
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}
