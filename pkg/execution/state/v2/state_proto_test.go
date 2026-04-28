package state

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigProtoRoundTrip(t *testing.T) {
	replayID := uuid.New()
	originalRunID := ulid.Make()
	batchID := ulid.Make()
	priorityFactor := int64(42)

	original := *InitConfig(&Config{
		FunctionVersion: 3,
		SpanID:          "span-abc",
		BatchID:         &batchID,
		StartedAt:       time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		EventIDs:        []ulid.ULID{ulid.Make(), ulid.Make()},
		RequestVersion:  2,
		Idempotency:     "idem-key-123",
		HasAI:           true,
		ReplayID:        &replayID,
		OriginalRunID:   &originalRunID,
		PriorityFactor:  &priorityFactor,
		CustomConcurrencyKeys: []CustomConcurrency{
			{Key: "f:fn-id:user-123", Hash: "hash1", Limit: 5},
			{Key: "a:acct-id:global", Hash: "hash2", Limit: 10},
		},
		ForceStepPlan: true,
		Context:       map[string]any{"user": "test", "count": float64(42)},
		Semaphores: []constraintapi.Semaphore{
			{ID: "app:app-uuid", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto},
			{ID: "fn:fn-uuid", UsageValue: "uv1", Weight: 2, Release: constraintapi.SemaphoreReleaseManual},
			{ID: "fnkey:hash123", UsageValue: "uv2", Weight: 3, Release: constraintapi.SemaphoreReleaseManual},
		},
	})

	pbCfg, err := ConfigToProto(original)
	require.NoError(t, err)

	// Verify the proto has the semaphores JSON populated
	assert.NotEmpty(t, pbCfg.SemaphoresJson)

	roundTripped, err := ConfigFromProto(pbCfg)
	require.NoError(t, err)

	// Compare all fields
	assert.Equal(t, original.FunctionVersion, roundTripped.FunctionVersion)
	assert.Equal(t, original.SpanID, roundTripped.SpanID)
	assert.Equal(t, original.BatchID.String(), roundTripped.BatchID.String())
	assert.Equal(t, original.StartedAt.UTC(), roundTripped.StartedAt.UTC())
	assert.Equal(t, len(original.EventIDs), len(roundTripped.EventIDs))
	for i := range original.EventIDs {
		assert.Equal(t, original.EventIDs[i].String(), roundTripped.EventIDs[i].String())
	}
	assert.Equal(t, original.RequestVersion, roundTripped.RequestVersion)
	assert.Equal(t, original.Idempotency, roundTripped.Idempotency)
	assert.Equal(t, original.HasAI, roundTripped.HasAI)
	assert.Equal(t, original.ReplayID.String(), roundTripped.ReplayID.String())
	assert.Equal(t, original.OriginalRunID.String(), roundTripped.OriginalRunID.String())
	assert.Equal(t, *original.PriorityFactor, *roundTripped.PriorityFactor)
	assert.Equal(t, original.CustomConcurrencyKeys, roundTripped.CustomConcurrencyKeys)
	assert.Equal(t, original.ForceStepPlan, roundTripped.ForceStepPlan)
	assert.Equal(t, original.Context, roundTripped.Context)

	// Semaphores
	require.Equal(t, len(original.Semaphores), len(roundTripped.Semaphores))
	for i := range original.Semaphores {
		assert.Equal(t, original.Semaphores[i].ID, roundTripped.Semaphores[i].ID)
		assert.Equal(t, original.Semaphores[i].UsageValue, roundTripped.Semaphores[i].UsageValue)
		assert.Equal(t, original.Semaphores[i].Weight, roundTripped.Semaphores[i].Weight)
		assert.Equal(t, original.Semaphores[i].Release, roundTripped.Semaphores[i].Release)
	}
}

func TestConfigProtoRoundTrip_EmptySemaphores(t *testing.T) {
	original := *InitConfig(&Config{
		FunctionVersion: 1,
		EventIDs:        []ulid.ULID{ulid.Make()},
	})

	pbCfg, err := ConfigToProto(original)
	require.NoError(t, err)
	assert.Empty(t, pbCfg.SemaphoresJson)

	roundTripped, err := ConfigFromProto(pbCfg)
	require.NoError(t, err)
	assert.Nil(t, roundTripped.Semaphores)
}

func TestMetadataProtoRoundTrip(t *testing.T) {
	functionID := uuid.New()
	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()
	runID := ulid.Make()

	original := Metadata{
		ID: ID{
			RunID:      runID,
			FunctionID: functionID,
			Tenant: Tenant{
				AppID:     appID,
				EnvID:     envID,
				AccountID: accountID,
			},
		},
		Config: *InitConfig(&Config{
			FunctionVersion: 5,
			SpanID:          "test-span",
			EventIDs:        []ulid.ULID{ulid.Make()},
			RequestVersion:  2,
			Semaphores: []constraintapi.Semaphore{
				{ID: "fn:abc", Weight: 1, Release: constraintapi.SemaphoreReleaseManual},
			},
		}),
		Metrics: RunMetrics{
			StateSize: 1024,
			EventSize: 256,
			StepCount: 3,
		},
		Stack: []string{"step-1", "step-2", "step-3"},
	}

	pbMd, err := MetadataToProto(original)
	require.NoError(t, err)

	roundTripped, err := MetadataFromProto(pbMd)
	require.NoError(t, err)

	// ID
	assert.Equal(t, original.ID.RunID, roundTripped.ID.RunID)
	assert.Equal(t, original.ID.FunctionID, roundTripped.ID.FunctionID)
	assert.Equal(t, original.ID.Tenant.AppID, roundTripped.ID.Tenant.AppID)
	assert.Equal(t, original.ID.Tenant.EnvID, roundTripped.ID.Tenant.EnvID)
	assert.Equal(t, original.ID.Tenant.AccountID, roundTripped.ID.Tenant.AccountID)

	// Config
	assert.Equal(t, original.Config.FunctionVersion, roundTripped.Config.FunctionVersion)
	assert.Equal(t, original.Config.SpanID, roundTripped.Config.SpanID)
	require.Len(t, roundTripped.Config.Semaphores, 1)
	assert.Equal(t, original.Config.Semaphores[0], roundTripped.Config.Semaphores[0])

	// Metrics
	assert.Equal(t, original.Metrics.StateSize, roundTripped.Metrics.StateSize)
	assert.Equal(t, original.Metrics.EventSize, roundTripped.Metrics.EventSize)
	assert.Equal(t, original.Metrics.StepCount, roundTripped.Metrics.StepCount)

	// Stack
	assert.Equal(t, original.Stack, roundTripped.Stack)
}

func TestConfigFromProto_InvalidSemaphoresJSON(t *testing.T) {
	pbCfg, err := ConfigToProto(*InitConfig(&Config{
		EventIDs: []ulid.ULID{ulid.Make()},
	}))
	require.NoError(t, err)

	pbCfg.SemaphoresJson = `{invalid json`

	_, err = ConfigFromProto(pbCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal semaphores_json")
}

func TestV1FromMetadata_IncludesSemaphores(t *testing.T) {
	md := Metadata{
		ID: ID{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			Tenant: Tenant{
				AppID:     uuid.New(),
				EnvID:     uuid.New(),
				AccountID: uuid.New(),
			},
		},
		Config: *InitConfig(&Config{
			FunctionVersion: 1,
			EventIDs:        []ulid.ULID{ulid.Make()},
			Semaphores: []constraintapi.Semaphore{
				{ID: "app:app-1", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto},
				{ID: "fn:fn-1", UsageValue: "uv", Weight: 2, Release: constraintapi.SemaphoreReleaseManual},
			},
		}),
	}

	v1 := V1FromMetadata(md)

	require.Len(t, v1.Semaphores, 2)
	assert.Equal(t, "app:app-1", v1.Semaphores[0].ID)
	assert.Equal(t, constraintapi.SemaphoreReleaseAuto, v1.Semaphores[0].Release)
	assert.Equal(t, "fn:fn-1", v1.Semaphores[1].ID)
	assert.Equal(t, "uv", v1.Semaphores[1].UsageValue)
	assert.Equal(t, int64(2), v1.Semaphores[1].Weight)
	assert.Equal(t, constraintapi.SemaphoreReleaseManual, v1.Semaphores[1].Release)
}

func TestV1FromMetadata_NilSemaphores(t *testing.T) {
	md := Metadata{
		ID: ID{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			Tenant: Tenant{
				AppID:     uuid.New(),
				EnvID:     uuid.New(),
				AccountID: uuid.New(),
			},
		},
		Config: *InitConfig(&Config{
			FunctionVersion: 1,
			EventIDs:        []ulid.ULID{ulid.Make()},
		}),
	}

	v1 := V1FromMetadata(md)
	assert.Nil(t, v1.Semaphores)
}
