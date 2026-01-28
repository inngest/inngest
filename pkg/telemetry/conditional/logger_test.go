package conditional

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestConditionalDebug(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))

	t.Run("logs when enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		ConditionalDebug(ctx, "test.Scope", "test message", "key", "value")
		require.Contains(t, buf.String(), "test message")
		require.Contains(t, buf.String(), "key=value")
	})

	t.Run("does not log when disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled)

		ConditionalDebug(ctx, "test.Scope", "test message", "key", "value")
		require.Empty(t, buf.String())
	})
}

func TestConditionalInfo(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelInfo),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))

	t.Run("logs when enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		ConditionalInfo(ctx, "test.Scope", "info message")
		require.Contains(t, buf.String(), "info message")
	})

	t.Run("does not log when disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled)

		ConditionalInfo(ctx, "test.Scope", "info message")
		require.Empty(t, buf.String())
	})
}

func TestConditionalWarn(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelWarning),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))

	t.Run("logs when enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		ConditionalWarn(ctx, "test.Scope", "warn message")
		require.Contains(t, buf.String(), "warn message")
	})

	t.Run("does not log when disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled)

		ConditionalWarn(ctx, "test.Scope", "warn message")
		require.Empty(t, buf.String())
	})
}

func TestConditionalError(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelError),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))

	t.Run("logs when enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		ConditionalError(ctx, "test.Scope", "error message")
		require.Contains(t, buf.String(), "error message")
	})

	t.Run("does not log when disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled)

		ConditionalError(ctx, "test.Scope", "error message")
		require.Empty(t, buf.String())
	})
}

func TestConditionalLogger(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("scoped logger logs when enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		cl := NewConditionalLogger(l, "test.Scope")
		require.Equal(t, "test.Scope", cl.Scope())

		cl.Debug(ctx, "debug message", "key", "value")
		require.Contains(t, buf.String(), "debug message")
	})

	t.Run("scoped logger does not log when disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled)

		cl := NewConditionalLogger(l, "test.Scope")
		cl.Debug(ctx, "debug message")
		require.Empty(t, buf.String())
	})

	t.Run("With returns new logger with attributes", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(AlwaysEnabled)

		cl := NewConditionalLogger(l, "test.Scope")
		cl2 := cl.With("attr", "value")

		require.NotSame(t, cl, cl2)
		require.Equal(t, cl.Scope(), cl2.Scope())

		cl2.Info(ctx, "info message")
		require.Contains(t, buf.String(), "attr=value")
	})

	t.Run("WithScope returns new logger with different scope", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("new.Scope"))

		cl := NewConditionalLogger(l, "test.Scope")
		cl2 := cl.WithScope("new.Scope")

		require.Equal(t, "test.Scope", cl.Scope())
		require.Equal(t, "new.Scope", cl2.Scope())

		// Original scope disabled
		cl.Info(ctx, "message 1")
		require.Empty(t, buf.String())

		// New scope enabled
		cl2.Info(ctx, "message 2")
		require.Contains(t, buf.String(), "message 2")
	})
}

func TestNewConditionalLoggerFromContext(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(AlwaysEnabled)

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))

	cl := NewConditionalLoggerFromContext(ctx, "test.Scope")
	cl.Debug(ctx, "test message")
	require.Contains(t, buf.String(), "test message")
}

func TestConditionalLogger_AllLevels(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(AlwaysEnabled)

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelTrace),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	cl := NewConditionalLogger(l, "test.Scope")

	tests := []struct {
		name  string
		logFn func(context.Context, string, ...any)
		level string
		msg   string
	}{
		{"trace", cl.Trace, "TRACE", "trace msg"},
		{"debug", cl.Debug, "DEBUG", "debug msg"},
		{"info", cl.Info, "INFO", "info msg"},
		{"warn", cl.Warn, "WARN", "warn msg"},
		{"error", cl.Error, "ERROR", "error msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn(ctx, tt.msg)
			require.Contains(t, buf.String(), tt.msg)
		})
	}
}
