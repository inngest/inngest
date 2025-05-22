package logger

import (
	"context"
	"fmt"
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
	DevHandler
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

type Logger interface {
	//
	// Methods from slog.Logger
	//
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	LogAttrs(ctx context.Context, level slog.Level, msg string, args ...slog.Attr)
	Handler() slog.Handler
	With(args ...any) Logger

	//
	// Methods added in wrapper
	//
	Trace(msg string, args ...any)
	TraceContext(ctx context.Context, msg string, args ...any)
	Notice(msg string, args ...any)
	NoticeContext(ctx context.Context, msg string, args ...any)
	Emergency(msg string, args ...any)
	EmergencyContext(ctx context.Context, msg string, args ...any)
	SLog() *slog.Logger
}

type LoggerOpt func(o *loggerOpts)

type loggerOpts struct {
	writer  io.Writer
	level   slog.Level
	handler handler
}

func WithLoggerLevel(lvl slog.Level) LoggerOpt {
	return func(o *loggerOpts) {
		o.level = lvl
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

func newLogger(opts ...LoggerOpt) Logger {
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
	case DevHandler:
		return &logger{
			Logger: slog.New(newDevHandler(o.writer, &hopts)),
		}

	case TextHandler:
		return &logger{
			Logger: slog.New(slog.NewTextHandler(o.writer, &hopts)),
		}

	default:
		return &logger{
			Logger: slog.New(slog.NewJSONHandler(o.writer, &hopts)),
		}
	}
}

// StdlibLoggger returns the stdlib logger in context, or a new logger
// if none stored.
func StdlibLogger(ctx context.Context, opts ...LoggerOpt) Logger {
	l := ctx.Value(stdlibCtxKey)
	if l == nil {
		return newLogger(opts...)
	}
	return l.(Logger)
}

func VoidLogger() Logger {
	return newLogger(WithLoggerWriter(io.Discard))
}

func StdlibLoggerWithCustomVarName(ctx context.Context, varName string) Logger {
	l := ctx.Value(stdlibCtxKey)
	if l == nil {
		return newLogger(WithLoggerLevel(StdlibLevel(level(varName))))
	}
	return l.(Logger)
}

func WithStdlib(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, stdlibCtxKey, l)
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

func (l *logger) With(args ...any) Logger {
	if len(args) == 0 {
		return l
	}

	log := l.Logger.With(args...)
	return &logger{
		Logger: log,
	}
}

func (l *logger) Trace(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelTrace, msg, args...)
}

func (l *logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelTrace, msg, args...)
}

func (l *logger) Notice(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelNotice, msg, args...)
}

func (l *logger) NoticeContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelNotice, msg, args...)
}

func (l *logger) Emergency(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelEmergency, msg, args...)
}

func (l *logger) EmergencyContext(ctx context.Context, msg string, args ...any) {
	l.Logger.Log(ctx, LevelEmergency, msg, args...)
}

func (l *logger) SLog() *slog.Logger {
	return l.Logger
}

// newDevHandler constructs a dev handler to be used
func newDevHandler(w io.Writer, opts *slog.HandlerOptions) *devHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{
			Level: LevelInfo,
		}
	}

	return &devHandler{
		writer: w,
		opts:   opts,
	}
}

// devHandler is used for development purposes and also provide nicer log output for the dev server
type devHandler struct {
	writer io.Writer
	opts   *slog.HandlerOptions
}

func (d *devHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return d.opts.Level != nil && lvl >= d.opts.Level.Level()
}

func (d *devHandler) Handle(ctx context.Context, rec slog.Record) error {
	return fmt.Errorf("not implemented")
}

func (d *devHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return d
}

func (d *devHandler) WithGroup(name string) slog.Handler {
	return d
}
