package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	stdlibCtxKey = stdlibKey{}
)

type stdlibKey struct{}

type handler int

const (
	JSONHandler handler = iota
	TextHandler
)

// NOTE: reference
// https://go.dev/src/log/slog/example_custom_levels_test.go
const (
	DefaultStdlibLevel = slog.LevelInfo

	LevelTrace     = slog.Level(-8)
	LevelDebug     = slog.LevelDebug
	LevelInfo      = slog.LevelInfo
	LevelNotice    = slog.Level(2)
	LevelWarning   = slog.LevelWarn
	LevelError     = slog.LevelError
	LevelEmergency = slog.Level(12)
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
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarning
	case "error":
		return LevelError
	case "emergency":
		return LevelEmergency
	default:
		return DefaultStdlibLevel
	}
}

func level(levelVarName string) string {
	return os.Getenv(levelVarName)
}

// logger is a wrapper over slog with additional levels
type logger struct {
	*slog.Logger
}

func (l *logger) Trace(msg string, attrs ...any) {
	l.Logger.Log(context.Background(), LevelTrace, msg, attrs...)
}

func (l *logger) TraceContext(ctx context.Context, msg string, attrs ...any) {
	l.Logger.Log(ctx, LevelTrace, msg, attrs...)
}

func (l *logger) Notice(msg string, attrs ...any) {
	l.Logger.Log(context.Background(), LevelNotice, msg, attrs...)
}

func (l *logger) NoticeContext(ctx context.Context, msg string, attrs ...any) {
	l.Logger.Log(ctx, LevelNotice, msg, attrs...)
}

func (l *logger) Emergency(msg string, attrs ...any) {
	l.Logger.Log(context.Background(), LevelEmergency, msg, attrs...)
}

func (l *logger) EmergencyContext(ctx context.Context, msg string, attrs ...any) {
	l.Logger.Log(ctx, LevelEmergency, msg, attrs...)
}
