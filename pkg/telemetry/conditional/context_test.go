package conditional

import (
	"context"
	"testing"

	"github.com/google/uuid"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestContextOptions(t *testing.T) {
	t.Run("WithAccountID", func(t *testing.T) {
		id := uuid.New()
		ctx := FeatureFlagContext{}
		WithAccountID(id)(&ctx)
		require.Equal(t, id, ctx.AccountID)
	})

	t.Run("WithEnvID", func(t *testing.T) {
		id := uuid.New()
		ctx := FeatureFlagContext{}
		WithEnvID(id)(&ctx)
		require.Equal(t, id, ctx.EnvID)
	})

	t.Run("WithFunctionID", func(t *testing.T) {
		id := uuid.New()
		ctx := FeatureFlagContext{}
		WithFunctionID(id)(&ctx)
		require.Equal(t, id, ctx.FunctionID)
	})

	t.Run("WithRunID", func(t *testing.T) {
		id := ulid.Make()
		ctx := FeatureFlagContext{}
		WithRunID(id)(&ctx)
		require.Equal(t, id, ctx.RunID)
	})

	t.Run("WithEventID", func(t *testing.T) {
		id := ulid.Make()
		ctx := FeatureFlagContext{}
		WithEventID(id)(&ctx)
		require.Equal(t, id, ctx.EventID)
	})

	t.Run("WithBillingPlan", func(t *testing.T) {
		ctx := FeatureFlagContext{}
		WithBillingPlan("enterprise")(&ctx)
		require.Equal(t, "enterprise", ctx.BillingPlan)
	})

	t.Run("WithExtra", func(t *testing.T) {
		ctx := FeatureFlagContext{}
		WithExtra("key", "value")(&ctx)
		require.Equal(t, "value", ctx.Extra["key"])

		// Add another key
		WithExtra("key2", 42)(&ctx)
		require.Equal(t, "value", ctx.Extra["key"])
		require.Equal(t, 42, ctx.Extra["key2"])
	})
}

func TestFromStateID(t *testing.T) {
	stateID := statev2.ID{
		RunID:      ulid.Make(),
		FunctionID: uuid.New(),
		Tenant: statev2.Tenant{
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     uuid.New(),
		},
	}

	opts := FromStateID(stateID)
	require.Len(t, opts, 4)

	ctx := FeatureFlagContext{}
	for _, opt := range opts {
		opt(&ctx)
	}

	require.Equal(t, stateID.Tenant.AccountID, ctx.AccountID)
	require.Equal(t, stateID.Tenant.EnvID, ctx.EnvID)
	require.Equal(t, stateID.FunctionID, ctx.FunctionID)
	require.Equal(t, stateID.RunID, ctx.RunID)
}

func TestFromIdentifier(t *testing.T) {
	eventID := ulid.Make()
	identifier := statev1.Identifier{
		RunID:       ulid.Make(),
		WorkflowID:  uuid.New(),
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		EventID:     eventID,
	}

	opts := FromIdentifier(identifier)
	require.Len(t, opts, 5) // includes EventID

	ctx := FeatureFlagContext{}
	for _, opt := range opts {
		opt(&ctx)
	}

	require.Equal(t, identifier.AccountID, ctx.AccountID)
	require.Equal(t, identifier.WorkspaceID, ctx.EnvID)
	require.Equal(t, identifier.WorkflowID, ctx.FunctionID)
	require.Equal(t, identifier.RunID, ctx.RunID)
	require.Equal(t, identifier.EventID, ctx.EventID)
}

func TestFromIdentifier_ZeroEventID(t *testing.T) {
	identifier := statev1.Identifier{
		RunID:       ulid.Make(),
		WorkflowID:  uuid.New(),
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		// EventID is zero
	}

	opts := FromIdentifier(identifier)
	require.Len(t, opts, 4) // EventID not included when zero

	ctx := FeatureFlagContext{}
	for _, opt := range opts {
		opt(&ctx)
	}

	require.False(t, ctx.HasEventID())
}

func TestWithContext(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()

	ctx := WithContext(context.Background(),
		WithAccountID(accountID),
		WithEnvID(envID),
	)

	ffCtx := GetFromContext(ctx)
	require.Equal(t, accountID, ffCtx.AccountID)
	require.Equal(t, envID, ffCtx.EnvID)
}

func TestAddToContext(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()
	runID := ulid.Make()

	// Start with account and env
	ctx := WithContext(context.Background(),
		WithAccountID(accountID),
		WithEnvID(envID),
	)

	// Add run ID later
	ctx = AddToContext(ctx, WithRunID(runID))

	ffCtx := GetFromContext(ctx)
	require.Equal(t, accountID, ffCtx.AccountID)
	require.Equal(t, envID, ffCtx.EnvID)
	require.Equal(t, runID, ffCtx.RunID)
}

func TestAddToContext_NoExistingContext(t *testing.T) {
	accountID := uuid.New()

	// AddToContext on a context with no FeatureFlagContext
	ctx := AddToContext(context.Background(), WithAccountID(accountID))

	ffCtx := GetFromContext(ctx)
	require.Equal(t, accountID, ffCtx.AccountID)
}

func TestAddToContext_DoesNotMutateOriginal(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()

	ctx1 := WithContext(context.Background(),
		WithAccountID(accountID),
		WithExtra("key", "value1"),
	)

	ctx2 := AddToContext(ctx1,
		WithEnvID(envID),
		WithExtra("key", "value2"),
	)

	// Original context should be unchanged
	ffCtx1 := GetFromContext(ctx1)
	require.False(t, ffCtx1.HasEnvID())
	require.Equal(t, "value1", ffCtx1.Extra["key"])

	// New context should have merged values
	ffCtx2 := GetFromContext(ctx2)
	require.Equal(t, accountID, ffCtx2.AccountID)
	require.Equal(t, envID, ffCtx2.EnvID)
	require.Equal(t, "value2", ffCtx2.Extra["key"])
}

func TestGetFromContext_NoContext(t *testing.T) {
	ffCtx := GetFromContext(context.Background())
	require.Equal(t, FeatureFlagContext{}, ffCtx)
}

func TestHasContext(t *testing.T) {
	t.Run("no context", func(t *testing.T) {
		require.False(t, HasContext(context.Background()))
	})

	t.Run("with context", func(t *testing.T) {
		ctx := WithContext(context.Background(), WithAccountID(uuid.New()))
		require.True(t, HasContext(ctx))
	})
}

func TestWithScope(t *testing.T) {
	ctx := WithScope(context.Background(), "test.Scope")

	scope, ok := ScopeFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "test.Scope", scope)
}

func TestScopeFromContext(t *testing.T) {
	t.Run("no scope", func(t *testing.T) {
		scope, ok := ScopeFromContext(context.Background())
		require.False(t, ok)
		require.Empty(t, scope)
	})

	t.Run("with scope", func(t *testing.T) {
		ctx := WithScope(context.Background(), "my.Scope")
		scope, ok := ScopeFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, "my.Scope", scope)
	})
}

func TestHasScope(t *testing.T) {
	t.Run("no scope", func(t *testing.T) {
		require.False(t, HasScope(context.Background()))
	})

	t.Run("with scope", func(t *testing.T) {
		ctx := WithScope(context.Background(), "test.Scope")
		require.True(t, HasScope(ctx))
	})
}

func TestScopeAndFeatureFlagContextTogether(t *testing.T) {
	accountID := uuid.New()

	// Set both feature flag context and scope
	ctx := WithContext(context.Background(), WithAccountID(accountID))
	ctx = WithScope(ctx, "queue.Process")

	// Both should be retrievable
	ffCtx := GetFromContext(ctx)
	require.Equal(t, accountID, ffCtx.AccountID)

	scope, ok := ScopeFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "queue.Process", scope)
}
