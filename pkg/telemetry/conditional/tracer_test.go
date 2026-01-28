package conditional

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestScopedConditionalTracer(t *testing.T) {
	defer ClearFeatureFlag()

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("Start creates span when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		sct := NewScopedConditionalTracer(tracer, "test.Scope")
		require.Equal(t, "test.Scope", sct.Scope())
		require.NotNil(t, sct.Tracer())

		newCtx, span := sct.Start(ctx, "test-span")
		defer span.End()

		require.NotEqual(t, ctx, newCtx)
		require.NotNil(t, span)
		// Check it's not a noop span
		require.NotEqual(t, noop.Span{}, span)
	})

	t.Run("Start returns noop span when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		sct := NewScopedConditionalTracer(tracer, "test.Scope")
		newCtx, span := sct.Start(ctx, "test-span")

		require.Equal(t, ctx, newCtx)
		require.IsType(t, noop.Span{}, span)
	})

	t.Run("WithScope returns new tracer with different scope", func(t *testing.T) {
		RegisterFeatureFlag(ScopeEnabled("new.Scope"))

		sct := NewScopedConditionalTracer(tracer, "old.Scope")
		sct2 := sct.WithScope("new.Scope")

		require.Equal(t, "old.Scope", sct.Scope())
		require.Equal(t, "new.Scope", sct2.Scope())

		// Old scope disabled
		_, span1 := sct.Start(ctx, "span1")
		require.IsType(t, noop.Span{}, span1)

		// New scope enabled
		_, span2 := sct2.Start(ctx, "span2")
		defer span2.End()
		require.NotEqual(t, noop.Span{}, span2)
	})
}

func TestConditionalStart(t *testing.T) {
	defer ClearFeatureFlag()

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("creates span when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		newCtx, span := ConditionalStart(ctx, tracer, "test.Scope", "test-span")
		defer span.End()

		require.NotEqual(t, ctx, newCtx)
		require.NotEqual(t, noop.Span{}, span)
	})

	t.Run("returns noop span when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		newCtx, span := ConditionalStart(ctx, tracer, "test.Scope", "test-span")

		require.Equal(t, ctx, newCtx)
		require.IsType(t, noop.Span{}, span)
	})
}

func TestStartConditionalSpan(t *testing.T) {
	defer ClearFeatureFlag()

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("creates enabled span when feature flag is enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		cs := StartConditionalSpan(ctx, tracer, "test.Scope", "test-span")
		defer cs.End()

		require.True(t, cs.Enabled())
		require.NotEqual(t, ctx, cs.Context())
		require.NotEqual(t, noop.Span{}, cs.Span())
	})

	t.Run("creates disabled span when feature flag is disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		cs := StartConditionalSpan(ctx, tracer, "test.Scope", "test-span")

		require.False(t, cs.Enabled())
		require.Equal(t, ctx, cs.Context())
		require.IsType(t, noop.Span{}, cs.Span())
	})
}

func TestConditionalSpan_Methods(t *testing.T) {
	defer ClearFeatureFlag()

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("methods work on enabled span", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		cs := StartConditionalSpan(ctx, tracer, "test.Scope", "test-span")
		defer cs.End()

		// These should not panic
		cs.SetAttributes(attribute.String("key", "value"))
		cs.SetStatus(codes.Ok, "success")
		cs.SetName("new-name")
		cs.AddEvent("test-event")
		cs.RecordError(nil)
	})

	t.Run("methods are safe on disabled span", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		cs := StartConditionalSpan(ctx, tracer, "test.Scope", "test-span")

		// These should not panic even on noop span
		cs.SetAttributes(attribute.String("key", "value"))
		cs.SetStatus(codes.Error, "error")
		cs.SetName("new-name")
		cs.AddEvent("test-event")
		cs.RecordError(nil)
		cs.End()
	})
}

func TestConditionalSpan_WithSpanOptions(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(AlwaysEnabled)

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	cs := StartConditionalSpan(ctx, tracer, "test.Scope", "test-span",
		trace.WithAttributes(attribute.String("init-key", "init-value")),
	)
	defer cs.End()

	require.True(t, cs.Enabled())
}

func TestScopedConditionalTracer_StartWithContext(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(AlwaysEnabled)

	tracer := otel.Tracer("test")
	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	sct := NewScopedConditionalTracer(tracer, "test.Scope")
	newCtx, span := sct.StartWithContext(ctx, "test-span")
	defer span.End()

	require.NotEqual(t, ctx, newCtx)
	require.NotEqual(t, noop.Span{}, span)
}
