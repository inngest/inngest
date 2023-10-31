package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var (
	DefaultStdlibLevel = slog.LevelInfo

	stdlibCtxKey = stdlibKey{}
)

type stdlibKey struct{}

// StdlibLoggger returns the stdlib logger in context, or a new logger
// if none stored.
func StdlibLogger(ctx context.Context) *slog.Logger {
	logger := ctx.Value(stdlibCtxKey)
	if logger == nil {
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: StdlibLevel(),
		}))
	}
	return logger.(*slog.Logger)
}

func WithStdlib(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, stdlibCtxKey, logger)
}

func StdlibLevel() slog.Level {
	switch strings.ToLower(Level()) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return DefaultStdlibLevel

	}
}

func Level() string {
	return os.Getenv("LOG_LEVEL")
}
