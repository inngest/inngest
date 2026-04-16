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

// TestSaveDeferRoundTrip pins down the next behavior after OpcodeDeferAdd is
// parsed: the executor persists a Defer record into run state so that
// finalization can load it back and emit inngest/deferred.start events.
//
// To make this pass you'll need to build, roughly in this order:
//
//  1. statev2.ScheduleStatus enum (in pkg/execution/state/v2/) with the four
//     constants from the ticket: Unknown, Scheduled, AfterRun, Cancelled.
//  2. statev2.Defer struct with CompanionID, HashedID, ScheduleStatus, Input.
//  3. Add SaveDefer + LoadDefers methods to the RunService / StateLoader
//     interfaces in pkg/execution/state/v2/interfaces.go.
//  4. Implement them on the Redis adapter (v2_adapter.go), storing defers in a
//     hash at the key `{state:runID}:groups:fnID:runID` (add a Defers() method
//     to RunStateKeyGenerator in key_generator.go alongside Actions/Stack/etc).
func TestSaveDeferRoundTrip(t *testing.T) {
	ctx := context.Background()

	// --- boilerplate: miniredis + v2 state service ---
	// Copied verbatim from TestV2Adapter. Everything up to the Create() call
	// is plumbing — skip over it on your first read.
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
		CompanionID:    "score",
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
// (CompanionID, Input) are preserved across the status change.
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
		CompanionID:    "score",
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
	require.Equal(t, original.CompanionID, got.CompanionID)
	require.Equal(t, original.HashedID, got.HashedID)
	require.JSONEq(t, string(original.Input), string(got.Input),
		"Input must survive the status update unchanged")
}
