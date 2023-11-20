package logger

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

const (
	DefaultLevel = zerolog.DebugLevel
)

var (
	logLevel, logFormat string
)

// SetLevel sets the default log level.
func SetLevel(to string) {
	lvl, err := zerolog.ParseLevel(to)
	if err == nil {
		logLevel = lvl.String()
	}
}

// SetLogLevel sets the default log level.
func SetFormat(to string) {
	logFormat = to
}

type loggerKey struct{}

// With sets a logger in the context for future use.  This allows functions
// to set key-value objects in the logger for later use with zero extra config.
func With(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// From returns the logger from the given context, defaulting to a new logger
// set to the given log level.
//
// This will always attempt to associate the logger with a trace from context,
// even if no logger is present within the context.
func From(ctx context.Context) *zerolog.Logger {
	logger := ctx.Value(loggerKey{})

	if logger == nil {
		return Default()
	}

	l := logger.(zerolog.Logger)
	return &l
}

// New returns a new logger set to the given level, with no associated context
// embedded.
func New(lvl zerolog.Level) *zerolog.Logger {
	l := zerolog.New(os.Stderr).Level(lvl).With().Timestamp().Logger()

	if !viper.GetBool("json") && logFormat != "json" {
		l = l.Output(zerolog.ConsoleWriter{
			Out: os.Stderr,
		})
	}

	return &l
}

// Default returns a new logger with no context, set to the default level.
func Default() *zerolog.Logger {
	if logLevel == "" {
		return New(DefaultLevel)
	}
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	return New(lvl)
}

func Buffered(buf io.Writer) *zerolog.Logger {
	l := zerolog.New(buf).Level(DefaultLevel).With().Timestamp().Logger()

	if !viper.GetBool("json") && logFormat != "json" {
		l = l.Output(zerolog.ConsoleWriter{
			Out:         buf,
			FormatLevel: func(i interface{}) string { return "" },
		})
	}

	return &l
}
