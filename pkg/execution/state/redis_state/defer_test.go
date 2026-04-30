package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"

	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

func TestSaveDeferRoundTrip(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})
	pauseStore := NewPauseStore(unshardedClient)

	mgr, err := New(ctx,
		WithShardedClient(shardedClient),
		WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)
	v2svc := MustRunServiceV2(mgr)

	// Create a run. Defers can't exist without a run to attach to.
	id := statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
		FunctionID: uuid.New(),
		Tenant: statev2.Tenant{
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     uuid.New(),
		},
	}
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = v2svc.Create(ctx, statev2.CreateState{
		Metadata: statev2.Metadata{
			ID: id,
			Config: *statev2.InitConfig(&statev2.Config{
				EventIDs: []ulid.ULID{eventID},
			}),
		},
		Events: []json.RawMessage{[]byte(`{"name":"test.event"}`)},
	})
	require.NoError(t, err)

	// --- the actual test ---
	want := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}

	err = v2svc.SaveDefer(ctx, id, want)
	require.NoError(t, err)

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.Len(t, defers, 1, "expected exactly one defer after a single SaveDefer call")
	require.Equal(t, want, defers[want.HashedID])
}

// TestSetDeferStatus verifies the atomic status-only update used by DeferCancel.
// It also checks that missing defers return an error and that other fields
// (FnSlug, Input) are preserved across the status change.
func TestSetDeferStatus(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})
	pauseStore := NewPauseStore(unshardedClient)

	mgr, err := New(ctx,
		WithShardedClient(shardedClient),
		WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)
	v2svc := MustRunServiceV2(mgr)

	id := statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
		FunctionID: uuid.New(),
		Tenant: statev2.Tenant{
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     uuid.New(),
		},
	}
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = v2svc.Create(ctx, statev2.CreateState{
		Metadata: statev2.Metadata{
			ID: id,
			Config: *statev2.InitConfig(&statev2.Config{
				EventIDs: []ulid.ULID{eventID},
			}),
		},
		Events: []json.RawMessage{[]byte(`{"name":"test.event"}`)},
	})
	require.NoError(t, err)

	// Error path: missing defer returns an error.
	err = v2svc.SetDeferStatus(ctx, id, "missing-hashed-id", statev2.ScheduleStatusCancelled)
	require.Error(t, err, "expected SetDeferStatus to error when defer is missing")

	// Seed a defer so we can update it.
	original := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user":{"id":"u_123"}}`),
	}
	require.NoError(t, v2svc.SaveDefer(ctx, id, original))

	// Flip status.
	require.NoError(t, v2svc.SetDeferStatus(ctx, id, original.HashedID, statev2.ScheduleStatusCancelled))

	// Status updated; every other field preserved.
	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.Len(t, defers, 1)

	got := defers[original.HashedID]
	require.Equal(t, statev2.ScheduleStatusCancelled, got.ScheduleStatus)
	require.Equal(t, original.FnSlug, got.FnSlug)
	require.Equal(t, original.HashedID, got.HashedID)
	require.JSONEq(t, string(original.Input), string(got.Input),
		"Input must survive the status update unchanged")
}

func newDeferTestRunService(t *testing.T) (statev2.RunService, statev2.ID) {
	t.Helper()
	ctx := context.Background()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})
	pauseStore := NewPauseStore(unshardedClient)

	mgr, err := New(ctx,
		WithShardedClient(shardedClient),
		WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)
	v2svc := MustRunServiceV2(mgr)

	id := statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
		FunctionID: uuid.New(),
		Tenant: statev2.Tenant{
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     uuid.New(),
		},
	}
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = v2svc.Create(ctx, statev2.CreateState{
		Metadata: statev2.Metadata{
			ID: id,
			Config: *statev2.InitConfig(&statev2.Config{
				EventIDs: []ulid.ULID{eventID},
			}),
		},
		Events: []json.RawMessage{[]byte(`{"name":"test.event"}`)},
	})
	require.NoError(t, err)

	return v2svc, id
}

// A retried SaveDefer must not undo an interleaved SetDeferStatus(Cancelled).
func TestSaveDefer_DoesNotResurrectCancelled(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	original := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}

	require.NoError(t, v2svc.SaveDefer(ctx, id, original))
	require.NoError(t, v2svc.SetDeferStatus(ctx, id, original.HashedID, statev2.ScheduleStatusCancelled))

	// T3: SDK retransmits the original DeferAdd. Must not error, must not resurrect.
	require.NoError(t, v2svc.SaveDefer(ctx, id, original))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.Len(t, defers, 1)

	got := defers[original.HashedID]
	require.Equal(t, statev2.ScheduleStatusCancelled, got.ScheduleStatus,
		"cancelled defer must not be resurrected by a retried DeferAdd")
	require.Equal(t, original.FnSlug, got.FnSlug, "FnSlug must be preserved across the no-op")
	require.JSONEq(t, string(original.Input), string(got.Input),
		"Input must be preserved across the no-op")
}

// Non-cancelled records get overwritten — the legitimate SDK retransmit path.
func TestSaveDefer_OverwritesAfterRun(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	first := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_old"}`),
	}
	require.NoError(t, v2svc.SaveDefer(ctx, id, first))

	second := first
	second.Input = json.RawMessage(`{"user_id":"u_new"}`)
	require.NoError(t, v2svc.SaveDefer(ctx, id, second))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.JSONEq(t, `{"user_id":"u_new"}`, string(defers[first.HashedID].Input),
		"non-cancelled defer should be overwritten by a subsequent SaveDefer")
}

// Regression: cjson cannot distinguish empty objects from empty arrays, so a
// raw `{}` Input round-tripped through setDeferStatus.lua would re-encode as
// `[]`. SaveDefer normalizes `{}` → nil before marshalling to defuse this; this
// test pins that normalization in place.
func TestSetDeferStatus_EmptyObjectInputDoesNotBecomeArray(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	d := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{}`),
	}
	require.NoError(t, v2svc.SaveDefer(ctx, id, d))
	require.NoError(t, v2svc.SetDeferStatus(ctx, id, d.HashedID, statev2.ScheduleStatusCancelled))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	got := defers[d.HashedID]

	require.Equal(t, statev2.ScheduleStatusCancelled, got.ScheduleStatus)
	require.NotEqual(t, "[]", string(got.Input),
		"cjson regression: empty-object Input must not round-trip as `[]`")
	// SaveDefer normalizes `{}` → nil, which marshals as JSON null.
	if len(got.Input) > 0 {
		require.JSONEq(t, `null`, string(got.Input))
	}
}

func TestSaveDefer_FirstWriteWhenMissing(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	d := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}
	require.NoError(t, v2svc.SaveDefer(ctx, id, d))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.Equal(t, d, defers[d.HashedID])
}
