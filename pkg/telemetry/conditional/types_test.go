package conditional

import (
	"testing"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestObservabilityType_String(t *testing.T) {
	tests := []struct {
		name     string
		obsType  ObservabilityType
		expected string
	}{
		{"logs", ObservabilityTypeLogs, "logs"},
		{"metrics", ObservabilityTypeMetrics, "metrics"},
		{"traces", ObservabilityTypeTraces, "traces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.obsType.String())
		})
	}
}

func TestFeatureFlagContext_HasIdentifiers(t *testing.T) {
	t.Run("empty context has no identifiers", func(t *testing.T) {
		ctx := FeatureFlagContext{}
		require.False(t, ctx.HasAccountID())
		require.False(t, ctx.HasEnvID())
		require.False(t, ctx.HasFunctionID())
		require.False(t, ctx.HasRunID())
		require.False(t, ctx.HasEventID())
	})

	t.Run("context with identifiers", func(t *testing.T) {
		ctx := FeatureFlagContext{
			AccountID:  uuid.New(),
			EnvID:      uuid.New(),
			FunctionID: uuid.New(),
			RunID:      ulid.Make(),
			EventID:    ulid.Make(),
		}
		require.True(t, ctx.HasAccountID())
		require.True(t, ctx.HasEnvID())
		require.True(t, ctx.HasFunctionID())
		require.True(t, ctx.HasRunID())
		require.True(t, ctx.HasEventID())
	})
}

func TestFeatureFlagContext_Extra(t *testing.T) {
	t.Run("GetExtra on nil map returns nil", func(t *testing.T) {
		ctx := FeatureFlagContext{}
		require.Nil(t, ctx.GetExtra("key"))
	})

	t.Run("GetExtra returns value", func(t *testing.T) {
		ctx := FeatureFlagContext{
			Extra: map[string]any{
				"key": "value",
				"num": 42,
			},
		}
		require.Equal(t, "value", ctx.GetExtra("key"))
		require.Equal(t, 42, ctx.GetExtra("num"))
		require.Nil(t, ctx.GetExtra("nonexistent"))
	})

	t.Run("GetExtraString returns string value", func(t *testing.T) {
		ctx := FeatureFlagContext{
			Extra: map[string]any{
				"str": "hello",
				"num": 42,
			},
		}
		require.Equal(t, "hello", ctx.GetExtraString("str"))
		require.Equal(t, "", ctx.GetExtraString("num"))
		require.Equal(t, "", ctx.GetExtraString("nonexistent"))
	})
}

func TestFeatureFlagContext_Clone(t *testing.T) {
	t.Run("clone creates independent copy", func(t *testing.T) {
		original := FeatureFlagContext{
			AccountID:   uuid.New(),
			EnvID:       uuid.New(),
			FunctionID:  uuid.New(),
			RunID:       ulid.Make(),
			BillingPlan: "enterprise",
			Extra: map[string]any{
				"key": "value",
			},
		}

		clone := original.Clone()

		// Values should be equal
		require.Equal(t, original.AccountID, clone.AccountID)
		require.Equal(t, original.EnvID, clone.EnvID)
		require.Equal(t, original.FunctionID, clone.FunctionID)
		require.Equal(t, original.RunID, clone.RunID)
		require.Equal(t, original.BillingPlan, clone.BillingPlan)
		require.Equal(t, original.Extra["key"], clone.Extra["key"])

		// Modifying clone's Extra should not affect original
		clone.Extra["key"] = "modified"
		require.Equal(t, "value", original.Extra["key"])
		require.Equal(t, "modified", clone.Extra["key"])
	})

	t.Run("clone with nil Extra", func(t *testing.T) {
		original := FeatureFlagContext{
			AccountID: uuid.New(),
		}

		clone := original.Clone()
		require.Nil(t, clone.Extra)
	})
}
