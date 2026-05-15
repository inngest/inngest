package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/rand"
	"os"
	"runtime/debug"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/lmittmann/tint"
)

var stdlibCtxKey = stdlibKey{}

type stdlibKey struct{}

type handler int

// noop doesn't log.
var noop = &logger{
	Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	level:  LevelEmergency,
}

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

// LoggerEventName is a special attribute key used for extracting the event name for event logs.
const LoggerEventName = "inngest_logger_event_name"

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
	Level() slog.Level
	With(args ...any) Logger

	// Optional uses the [DefaultLogEnabler] function to only log on truthy results
	Optional(accountID uuid.UUID, logname string) Logger

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

	// DebugSample samples a % of time to produce a debug log, between 0-100.
	// Non-sampled logs are logged as a trace.
	DebugSample(percent int, msg string, args ...any)

	// ReportError is a wrapper over Error, and will also submit a report to the error report tool
	ReportError(err error, msg string, opts ...ReportErrorOpt)
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
	handler := DevHandler
	switch strings.ToLower(os.Getenv("LOG_HANDLER")) {
	case "json":
		handler = JSONHandler
	case "dev":
		handler = DevHandler
	case "txt", "text":
		handler = TextHandler
	}

	o := &loggerOpts{
		level:   StdlibLevel(level("LOG_LEVEL")),
		writer:  os.Stderr,
		handler: handler,
	}

	for _, apply := range opts {
		apply(o)
	}

	hopts := slog.HandlerOptions{
		Level: o.level,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.LevelKey && len(groups) == 0 {
				if lvl, ok := attr.Value.Any().(slog.Level); ok {
					// annotate additional levels properly
					switch lvl {
					case LevelTrace:
						return slog.String(attr.Key, "TRACE")
					case LevelNotice:
						return slog.String(attr.Key, "NOTICE")
					case LevelEmergency:
						return slog.String(attr.Key, "EMERGENCY")
					}
				}
			}
			return attr
		},
	}

	switch o.handler {
	case DevHandler:
		return &logger{
			Logger: slog.New(tint.NewHandler(o.writer, &tint.Options{
				Level:      o.level,
				TimeFormat: "[15:04:05.000]", // millisecond
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == slog.LevelKey && len(groups) == 0 {
						lvl, ok := a.Value.Any().(slog.Level)
						if ok {
							// ref:
							// https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
							//
							// keep default color for warn and error
							switch lvl {
							case LevelTrace:
								return tint.Attr(13, slog.String(a.Key, "TRC"))
							case LevelDebug:
								return tint.Attr(3, slog.String(a.Key, "DBG"))
							case LevelInfo:
								return tint.Attr(14, slog.String(a.Key, "INF"))
							case LevelNotice:
								return tint.Attr(10, slog.String(a.Key, "NTC"))
							case LevelEmergency:
								return tint.Attr(9, slog.String(a.Key, "EMR"))
							}
						}
					}
					return a
				},
			})),
			level: o.level,
			attrs: []any{},
		}

	case TextHandler:
		return &logger{
			Logger: slog.New(slog.NewTextHandler(o.writer, &hopts)),
			level:  o.level,
			attrs:  []any{},
		}

	default:
		return &logger{
			Logger: slog.New(slog.NewJSONHandler(o.writer, &hopts)),
			level:  o.level,
			attrs:  []any{},
		}
	}
}

// From returns the stdlib logger in context, or a new logger
// if none stored.
func From(ctx context.Context, opts ...LoggerOpt) Logger {
	l := ctx.Value(stdlibCtxKey)
	if l == nil {
		return newLogger(opts...)
	}
	return l.(Logger)
}

// StdlibLogger is an alternative name for From for backcompat.
var StdlibLogger = From

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

func FromSlog(l *slog.Logger, level slog.Level) Logger {
	return &logger{
		Logger: l,
		level:  level,
	}
}

// logger is a wrapper over slog with additional levels
type logger struct {
	*slog.Logger
	level slog.Level

	// attrs represent the additional attributes used for this logger
	attrs []any
}

func (l *logger) DebugSample(percent int, msg string, args ...any) {
	if rand.Intn(100) < percent {
		l.Debug(msg, args...)
		return
	}
	l.Trace(msg, args...)
}

func (l *logger) Level() slog.Level {
	return l.level
}

func (l *logger) With(args ...any) Logger {
	if len(args) == 0 {
		return l
	}

	log := l.Logger.With(args...)
	return &logger{
		Logger: log,
		attrs:  append(l.attrs, args...),
	}
}

func (l *logger) Optional(accountID uuid.UUID, logname string) Logger {
	if DefaultLogEnabler(accountID, logname) {
		return l.With("logname", logname)
	}
	return noop
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

// reportErrorOpt provides options to tweak error reporting behaviors
type reportErrorOpt struct {
	// log indicates if the error report also emit an error log.
	// defaults to true.
	log bool
	// tags provides the additional tags to be added to the error report
	tags map[string]string
}

type ReportErrorOpt func(o *reportErrorOpt)

func (l *logger) ReportError(err error, msg string, opts ...ReportErrorOpt) {
	opt := reportErrorOpt{
		log:  true, // NOTE: defaults to true for now, can be disabled later
		tags: map[string]string{},
	}
	for _, apply := range opts {
		apply(&opt)
	}

	if sentry.CurrentHub().Client() != nil {
		tags := l.errorTags()
		tags["msg"] = msg
		// merge included attrs
		l.mergeAttrsWithErrorTags(tags)
		// merge additional tags
		maps.Copy(tags, opt.tags)

		// only report to sentry if initialize
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetTags(tags)
			scope.SetLevel(sentry.LevelError)
			sentry.CaptureException(err)
		})
	}

	if opt.log {
		args := l.attrs
		for k, v := range opt.tags {
			args = append(args, k, v)
		}

		args = append(args, "err", err, "stack", debug.Stack())

		l.Error(msg, args...)
	}
}

// errorTags returns the default list of KV to be used for reporting
func (l *logger) errorTags() map[string]string {
	tags := map[string]string{
		"host": host,
	}

	return tags
}

func (l *logger) mergeAttrsWithErrorTags(tags map[string]string) {
	if tags == nil {
		return
	}

	// increment by 2 since log attrs are a list where even items are key, and odd items are values
	for i := 0; i < len(l.attrs)-1; i += 2 {
		k := l.attrs[i]
		v := l.attrs[i+1]

		if key, ok := k.(string); ok {
			switch val := v.(type) {
			case string:
				tags[key] = val
			case fmt.Stringer:
				tags[key] = val.String()

			default:
				// no-op
			}
		}
	}
}

func WithErrorReportLog(enable bool) ReportErrorOpt {
	return func(o *reportErrorOpt) {
		o.log = enable
	}
}

func WithErrorReportTags(tags map[string]string) ReportErrorOpt {
	return func(o *reportErrorOpt) {
		o.tags = tags
	}
}
