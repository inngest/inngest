package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	DefaultStdlibLevel = slog.LevelInfo

	stdlibCtxKey = stdlibKey{}
)

type stdlibKey struct{}

type handler int

const (
	JSONHandler handler = iota
	TextHandler
)

type LoggerOpt func(o *loggerOpts)

type loggerOpts struct {
	writer  io.Writer
	level   slog.Level
	handler handler
}

func WithLoggerLevel(lvl string) LoggerOpt {
	return func(o *loggerOpts) {
		o.level = StdlibLevel(lvl)
	}
}

func WithLoggerWriter(w io.Writer) LoggerOpt {
	return func(o *loggerOpts) {
		o.writer = w
	}
}

func WithHandler(h handler) LoggerOpt {
	return func(o *loggerOpts) {
		o.handler = h
	}
}

func newLogger(opts ...LoggerOpt) *slog.Logger {
	o := &loggerOpts{
		level:   StdlibLevel(level("LOG_LEVEL")),
		writer:  os.Stderr,
		handler: JSONHandler,
	}

	for _, apply := range opts {
		apply(o)
	}

	hopts := slog.HandlerOptions{
		Level: o.level,
	}

	switch o.handler {
	case TextHandler:
		return slog.New(slog.NewTextHandler(o.writer, &hopts))

	default:
		return slog.New(slog.NewJSONHandler(o.writer, &hopts))
	}
}

// StdlibLoggger returns the stdlib logger in context, or a new logger
// if none stored.
func StdlibLogger(ctx context.Context, opts ...LoggerOpt) *slog.Logger {
	logger := ctx.Value(stdlibCtxKey)
	if logger == nil {
		return newLogger(opts...)
	}
	return logger.(*slog.Logger)
}

func VoidLogger() *slog.Logger {
	return newLogger(WithLoggerWriter(io.Discard))
}

func StdlibLoggerWithCustomVarName(ctx context.Context, varName string) *slog.Logger {
	logger := ctx.Value(stdlibCtxKey)
	if logger == nil {
		return newLogger(WithLoggerLevel(level(varName)))
	}
	return logger.(*slog.Logger)
}

func WithStdlib(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, stdlibCtxKey, logger)
}

func StdlibLevel(levelVarName string) slog.Level {
	switch strings.ToLower(levelVarName) {
	case "trace":
		return slog.LevelDebug
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

func level(levelVarName string) string {
	return os.Getenv(levelVarName)
}
