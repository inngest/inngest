package log

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	DefaultLevel     = zerolog.InfoLevel
	ViperLogLevelKey = "log.level"
)

var (
	prettyFormatter = zerolog.ConsoleWriter{
		Out:         os.Stderr,
		FormatLevel: func(i any) string { return "" },
	}
)

type loggerKey struct{}

func With(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func From(ctx context.Context) *zerolog.Logger {
	logger := ctx.Value(loggerKey{})

	if logger == nil {
		return Default()
	}

	l := logger.(zerolog.Logger)
	return &l
}

func New(lvl zerolog.Level) *zerolog.Logger {
	l := zerolog.New(os.Stderr).Level(lvl).With().Timestamp().Logger()
	if ttyLogger() {
		l = l.Output(prettyFormatter)
	}
	return &l
}

func Copy(l zerolog.Logger) zerolog.Logger {
	c := l.Output(os.Stderr)
	if ttyLogger() {
		c = c.Output(prettyFormatter)
	}
	return c
}

func Default() *zerolog.Logger {
	lvl, err := zerolog.ParseLevel(viper.GetString(ViperLogLevelKey))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	return New(lvl)
}

func ttyLogger() bool {
	switch f := viper.GetString("log.type"); f {
	case "tty":
		return true
	case "json":
		return false
	default:
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}
