package conditional

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

// TestContextBasedConditionalLogging tests the context-based approach
// where logger.From() automatically checks the conditional scope.
func TestContextBasedConditionalLogging(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)

	t.Run("logs when scope is enabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("queue.Process"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))
		ctx = WithScope(ctx, "queue.Process")

		// logger.From(ctx) should return the real logger since scope is enabled
		logger.From(ctx).Debug("should log this")
		require.Contains(t, buf.String(), "should log this")
	})

	t.Run("does not log when scope is disabled", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("other.Scope"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))
		ctx = WithScope(ctx, "queue.Process")

		// logger.From(ctx) should return VoidLogger since scope is not enabled
		logger.From(ctx).Debug("should not log this")
		require.Empty(t, buf.String())
	})

	t.Run("logs normally when no scope is set", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(NeverEnabled) // Even with NeverEnabled, no scope means normal logging

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))
		// No WithScope call

		// logger.From(ctx) should return the real logger since no scope is set
		logger.From(ctx).Debug("normal logging")
		require.Contains(t, buf.String(), "normal logging")
	})

	t.Run("one-liner pattern", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("batch.Schedule"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))

		// One-liner pattern: logger.From(conditional.WithScope(ctx, "scope"))
		logger.From(WithScope(ctx, "batch.Schedule")).Debug("one-liner enabled")
		require.Contains(t, buf.String(), "one-liner enabled")

		buf.Reset()
		logger.From(WithScope(ctx, "other.Scope")).Debug("one-liner disabled")
		require.Empty(t, buf.String())
	})
}

func TestLoggerHelper(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)

	t.Run("Logger helper returns logger from context", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("test.Scope"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))
		ctx = WithScope(ctx, "test.Scope")

		// Logger(ctx) is equivalent to logger.From(ctx)
		Logger(ctx).Debug("via Logger helper")
		require.Contains(t, buf.String(), "via Logger helper")
	})

	t.Run("Logger with scope enabled logs message", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("queue.CapacityLease"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))

		// Logger(ctx, scope) applies scope and returns logger
		Logger(ctx, "queue.CapacityLease").Debug("scoped log enabled")
		require.Contains(t, buf.String(), "scoped log enabled")
	})

	t.Run("Logger with scope disabled returns void logger", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("other.Scope"))

		ctx := logger.WithStdlib(context.Background(), l)
		ctx = WithContext(ctx, WithAccountID(uuid.New()))

		// Logger(ctx, scope) should return VoidLogger since scope is not enabled
		Logger(ctx, "queue.CapacityLease").Debug("scoped log disabled")
		require.Empty(t, buf.String())
	})

	t.Run("Logger with scope preserves logger fields", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("test.Scope"))

		ctx := logger.WithStdlib(context.Background(), l.With("base_field", "base_value"))
		ctx = WithContext(ctx, WithAccountID(uuid.New()))

		Logger(ctx, "test.Scope").Debug("with fields", "extra_field", "extra_value")
		require.Contains(t, buf.String(), "with fields")
		require.Contains(t, buf.String(), "base_field=base_value")
		require.Contains(t, buf.String(), "extra_field=extra_value")
	})
}

func TestConditionalLoggingWithFields(t *testing.T) {
	defer ClearFeatureFlag()

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelDebug),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)

	t.Run("preserves logger fields with conditional scope", func(t *testing.T) {
		buf.Reset()
		RegisterFeatureFlag(ScopeEnabled("test.Scope"))

		ctx := logger.WithStdlib(context.Background(), l.With("base_field", "base_value"))
		ctx = WithContext(ctx, WithAccountID(uuid.New()))
		ctx = WithScope(ctx, "test.Scope")

		logger.From(ctx).Debug("message with fields", "extra_field", "extra_value")
		require.Contains(t, buf.String(), "message with fields")
		require.Contains(t, buf.String(), "base_field=base_value")
		require.Contains(t, buf.String(), "extra_field=extra_value")
	})
}

func TestConditionalLoggingAllLevels(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(ScopeEnabled("test.Scope"))

	buf := &bytes.Buffer{}
	l := logger.From(context.Background(),
		logger.WithLoggerLevel(logger.LevelTrace),
		logger.WithLoggerWriter(buf),
		logger.WithHandler(logger.TextHandler),
	)

	ctx := logger.WithStdlib(context.Background(), l)
	ctx = WithContext(ctx, WithAccountID(uuid.New()))
	ctx = WithScope(ctx, "test.Scope")

	tests := []struct {
		name  string
		logFn func(string, ...any)
		msg   string
	}{
		{"trace", logger.From(ctx).Trace, "trace msg"},
		{"debug", logger.From(ctx).Debug, "debug msg"},
		{"info", logger.From(ctx).Info, "info msg"},
		{"warn", logger.From(ctx).Warn, "warn msg"},
		{"error", logger.From(ctx).Error, "error msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn(tt.msg)
			require.Contains(t, buf.String(), tt.msg)
		})
	}
}
