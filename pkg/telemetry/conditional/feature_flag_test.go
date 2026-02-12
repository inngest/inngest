package conditional

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterFeatureFlag(t *testing.T) {
	defer ClearFeatureFlag()

	called := false
	RegisterFeatureFlag(func(_ context.Context, _ FeatureFlagContext, _ ObservabilityType, _ string) bool {
		called = true
		return true
	})

	fn := GetFeatureFlagFn()
	require.NotNil(t, fn)

	result := fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "test")
	require.True(t, called)
	require.True(t, result)
}

func TestIsEnabled_NoFeatureFlagRegistered(t *testing.T) {
	ClearFeatureFlag()

	result := IsEnabled(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "test")
	require.False(t, result)
}

func TestIsEnabled_WithFeatureFlag(t *testing.T) {
	defer ClearFeatureFlag()

	targetAccountID := uuid.New()
	RegisterFeatureFlag(func(_ context.Context, ffCtx FeatureFlagContext, obsType ObservabilityType, scope string) bool {
		return ffCtx.AccountID == targetAccountID && obsType == ObservabilityTypeLogs
	})

	t.Run("matching account and type", func(t *testing.T) {
		result := IsEnabled(context.Background(), FeatureFlagContext{
			AccountID: targetAccountID,
		}, ObservabilityTypeLogs, "test")
		require.True(t, result)
	})

	t.Run("different account", func(t *testing.T) {
		result := IsEnabled(context.Background(), FeatureFlagContext{
			AccountID: uuid.New(),
		}, ObservabilityTypeLogs, "test")
		require.False(t, result)
	})

	t.Run("different type", func(t *testing.T) {
		result := IsEnabled(context.Background(), FeatureFlagContext{
			AccountID: targetAccountID,
		}, ObservabilityTypeMetrics, "test")
		require.False(t, result)
	})
}

func TestIsEnabledFromContext(t *testing.T) {
	defer ClearFeatureFlag()

	targetAccountID := uuid.New()
	RegisterFeatureFlag(func(_ context.Context, ffCtx FeatureFlagContext, _ ObservabilityType, _ string) bool {
		return ffCtx.AccountID == targetAccountID
	})

	t.Run("with matching context", func(t *testing.T) {
		ctx := WithContext(context.Background(), WithAccountID(targetAccountID))
		require.True(t, IsEnabledFromContext(ctx, ObservabilityTypeLogs, "test"))
	})

	t.Run("with different context", func(t *testing.T) {
		ctx := WithContext(context.Background(), WithAccountID(uuid.New()))
		require.False(t, IsEnabledFromContext(ctx, ObservabilityTypeLogs, "test"))
	})

	t.Run("no context", func(t *testing.T) {
		require.False(t, IsEnabledFromContext(context.Background(), ObservabilityTypeLogs, "test"))
	})
}

func TestConvenienceFunctions(t *testing.T) {
	defer ClearFeatureFlag()

	RegisterFeatureFlag(func(_ context.Context, _ FeatureFlagContext, obsType ObservabilityType, _ string) bool {
		return obsType == ObservabilityTypeLogs
	})

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	require.True(t, IsLoggingEnabled(ctx, "test"))
	require.False(t, IsMetricsEnabled(ctx, "test"))
	require.False(t, IsTracingEnabled(ctx, "test"))
}

func TestAlwaysEnabled(t *testing.T) {
	result := AlwaysEnabled(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "test")
	require.True(t, result)
}

func TestNeverEnabled(t *testing.T) {
	result := NeverEnabled(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "test")
	require.False(t, result)
}

func TestScopeEnabled(t *testing.T) {
	fn := ScopeEnabled("queue.Process", "constraintapi.Acquire")

	require.True(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "queue.Process"))
	require.True(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeMetrics, "constraintapi.Acquire"))
	require.False(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "other.Scope"))
}

func TestTypeEnabled(t *testing.T) {
	fn := TypeEnabled(ObservabilityTypeLogs, ObservabilityTypeTraces)

	require.True(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeLogs, "any"))
	require.True(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeTraces, "any"))
	require.False(t, fn(context.Background(), FeatureFlagContext{}, ObservabilityTypeMetrics, "any"))
}

func TestClearFeatureFlag(t *testing.T) {
	RegisterFeatureFlag(AlwaysEnabled)
	require.NotNil(t, GetFeatureFlagFn())

	ClearFeatureFlag()
	require.Nil(t, GetFeatureFlagFn())
}
